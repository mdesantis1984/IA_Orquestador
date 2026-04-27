// Package main is the entry point for the MCP Orchestrator
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/thiscloud/ia-orquestador/internal/admin"
	"github.com/thiscloud/ia-orquestador/internal/auth"
	"github.com/thiscloud/ia-orquestador/internal/db"
	memory "github.com/thiscloud/ia-orquestador/internal/engram"
	"github.com/thiscloud/ia-orquestador/internal/executor"
	"github.com/thiscloud/ia-orquestador/internal/jsonrpc"
	"github.com/thiscloud/ia-orquestador/internal/metrics"
	"github.com/thiscloud/ia-orquestador/internal/skills"
	"github.com/thiscloud/ia-orquestador/internal/tracing"
	"github.com/thiscloud/ia-orquestador/internal/transport"
	"github.com/thiscloud/ia-orquestador/pkg/types"
)

var (
	transportMode  = flag.String("transport", "stdio", "Transport mode: stdio or http")
	httpAddr       = flag.String("http-addr", ":8080", "HTTP server address")
	dbDriver       = flag.String("db-driver", "postgres", "Database driver: postgres")
	dbDSN          = flag.String("db-dsn", "", "PostgreSQL DSN (required when -db-driver=postgres)")
	memoryURL      = flag.String("memory-url", "http://127.0.0.1:7438", "IA_Recuerdo service URL")
	memoryKey      = flag.String("memory-key", "", "IA_Recuerdo API key (X-Api-Key header)")
	memoryProject  = flag.String("project", "ia-orquestador", "IA_Recuerdo project name")
	createToken    = flag.String("create-token", "", "Create an API key with the given name and exit")
	// OpenTelemetry
	otelExporter = flag.String("otel-exporter", "none", "OTel trace exporter: none | stdout | otlp | both")
	otelEndpoint = flag.String("otel-endpoint", "localhost:4318", "OTLP HTTP endpoint (used with -otel-exporter=otlp|both)")
	// Skill hot-reload
	skillReloadInterval = flag.Duration("skill-reload-interval", 0, "Skill hot-reload polling interval (e.g. 30s); 0 = disabled")
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("=== MCP Orchestrator Starting ===")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OpenTelemetry tracing
	shutdownTracing, err := tracing.Setup(ctx, "ia-orquestador", *otelExporter, *otelEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer func() {
		if err := shutdownTracing(context.Background()); err != nil {
			log.Printf("Tracing shutdown error: %v", err)
		}
	}()
	log.Printf("Tracing exporter: %s", *otelExporter)

	// Initialize database
	database, err := db.Open(db.Config{
		Driver: *dbDriver,
		DSN:    *dbDSN,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if err := db.Initialize(ctx, database, *dbDriver); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize auth validator
	authValidator := auth.NewValidator(database)

	// -create-token flag: generate a new API key and exit
	if *createToken != "" {
		key, err := authValidator.Generate(ctx, *createToken, "admin")
		if err != nil {
			log.Fatalf("Failed to create API key: %v", err)
		}
		fmt.Printf("\n API key created for %q:\n\n  %s\n\n Store it securely — it will not be shown again.\n\n", *createToken, key)
		return
	}

	// Bootstrap: if no API keys exist yet, generate one automatically
	if count, _ := authValidator.CountKeys(ctx); count == 0 {
		key, err := authValidator.Generate(ctx, "bootstrap", "admin")
		if err != nil {
			log.Fatalf("Failed to generate bootstrap API key: %v", err)
		}
		log.Printf("\n╔══════════════════════════════════════════════════════╗")
		log.Printf(" FIRST RUN — Bootstrap API key (save, shown once only):")
		log.Printf(" %s", key)
		log.Printf("╚══════════════════════════════════════════════════════╝\n")
	}

	// Initialize metrics
	met := metrics.New()

	// Initialize IA_Recuerdo client
	memClient := memory.NewClient(*memoryURL, *memoryProject, *memoryKey)

	// Initialize skill registry
	skillRegistry := skills.NewRegistry(database)
	if err := skillRegistry.LoadAll(ctx); err != nil {
		log.Fatalf("Failed to load skills: %v", err)
	}
	met.SetSkillsLoaded(int64(skillRegistry.Count()))

	// Start skill hot-reload if requested
	if *skillReloadInterval > 0 {
		skillRegistry.StartHotReload(ctx, *skillReloadInterval)
	}

	// Create orchestrator
	orch := NewOrchestrator(skillRegistry, memClient, database, met)

	// Create JSON-RPC dispatcher
	dispatcher := jsonrpc.NewDispatcher()
	orch.RegisterHandlers(dispatcher)

	// Create transport
	var trans transport.Transport
	switch *transportMode {
	case "stdio":
		trans = transport.NewSTDIOTransport(dispatcher)
	case "http":
		httpTrans := transport.NewHTTPTransport(*httpAddr, dispatcher)
		adminHandler := admin.New(skillRegistry, authValidator)
		httpTrans.OnRoutes(func(mux *http.ServeMux) {
			adminHandler.RegisterRoutes(mux)
			mux.HandleFunc("GET /metrics", met.Handler())
			// Standard MCP Streamable HTTP endpoint — used by VS Code / AI agents.
			orch.RegisterMCPRoutes(mux)
		})
		trans = httpTrans
	default:
		log.Fatalf("Unknown transport mode: %s", *transportMode)
	}

	// Start transport
	if err := trans.Start(ctx); err != nil {
		log.Fatalf("Failed to start transport: %v", err)
	}

	log.Printf("Orchestrator running with %s transport", trans.Name())

	// Record startup in IA_Recuerdo
	memClient.Save(ctx, &memory.Observation{
		Title: "Orchestrator started",
		Content: fmt.Sprintf("**What**: MCP Orchestrator started\n**Transport**: %s\n**Skills**: %d loaded",
			trans.Name(), skillRegistry.Count()),
		Type: "discovery",
	})

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutdown signal received")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := trans.Stop(shutdownCtx); err != nil {
		log.Printf("Transport shutdown error: %v", err)
	}

	log.Println("=== MCP Orchestrator Stopped ===")
}

// Orchestrator manages MCP operations
type Orchestrator struct {
	skills     *skills.Registry
	memory     *memory.Client
	db         *sql.DB
	met        *metrics.Metrics
	sessions   map[string]*types.Session
	sessionsMu sync.RWMutex
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(skillReg *skills.Registry, memClient *memory.Client, database *sql.DB, met *metrics.Metrics) *Orchestrator {
	return &Orchestrator{
		skills:   skillReg,
		memory:   memClient,
		db:       database,
		met:      met,
		sessions: make(map[string]*types.Session),
	}
}

// RegisterHandlers registers all MCP JSON-RPC handlers
func (o *Orchestrator) RegisterHandlers(d *jsonrpc.Dispatcher) {
	d.Register("mcp.initialize", o.handleInitialize)
	d.Register("mcp.tools.list", o.handleToolsList)
	d.Register("mcp.tools.call", o.handleToolsCall)
	d.Register("mcp.tools.status", o.handleToolsStatus)
}

// handleInitialize handles mcp.initialize requests
func (o *Orchestrator) handleInitialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ClientID           string          `json:"clientId"`
		ProtocolVersion    string          `json:"protocolVersion"`
		ClientCapabilities json.RawMessage `json:"clientCapabilities"`
		SessionHints       struct {
			TopicKey       string `json:"topic_key"`
			PreferredSkill string `json:"preferredSkill"`
		} `json:"sessionHints"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if req.ClientID == "" {
		req.ClientID = "anonymous-" + uuid.New().String()[:8]
	}

	// Create session
	session := &types.Session{
		ID:              uuid.New().String(),
		ClientID:        req.ClientID,
		ProtocolVersion: req.ProtocolVersion,
		Capabilities:    req.ClientCapabilities,
		TopicKey:        req.SessionHints.TopicKey,
		StartedAt:       time.Now(),
		LastSeenAt:      time.Now(),
		State:           string(types.SessionStateActive),
	}

	o.sessionsMu.Lock()
	o.sessions[session.ID] = session
	o.sessionsMu.Unlock()

	log.Printf("[MCP] Initialize: client=%s session=%s", req.ClientID, session.ID)

	// Update metrics
	o.met.IncrSession()

	// Record in IA_Recuerdo
	o.memory.Save(ctx, &memory.Observation{
		Title: fmt.Sprintf("Session started: %s", req.ClientID),
		Content: fmt.Sprintf("**Client**: %s\n**Protocol**: %s\n**Session**: %s",
			req.ClientID, req.ProtocolVersion, session.ID),
		Type:     "discovery",
		TopicKey: session.TopicKey,
	})

	return map[string]interface{}{
		"serverVersion": "0.2.0",
		"supportedFeatures": map[string]bool{
			"sse":        true,
			"ws":         true,
			"tools":      true,
			"skillsCRUD": true,
		},
		"sessionId": session.ID,
		"policy": map[string]interface{}{
			"maxConcurrentCalls": 10,
		},
	}, nil
}

// handleToolsList handles mcp.tools.list requests
func (o *Orchestrator) handleToolsList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		Filter struct {
			Capability string `json:"capability"`
			Tag        string `json:"tag"`
		} `json:"filter"`
		Paging struct {
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		} `json:"paging"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if req.Paging.Limit == 0 {
		req.Paging.Limit = 50
	}

	// List active skills
	skillsList, err := o.skills.List(types.SkillStatusActive, "", req.Paging.Limit, req.Paging.Offset)
	if err != nil {
		return nil, err
	}

	tools := make([]map[string]interface{}, 0, len(skillsList))
	for _, skill := range skillsList {
		var metadata types.SkillMetadata
		json.Unmarshal(skill.Metadata, &metadata)

		tools = append(tools, map[string]interface{}{
			"id":           skill.ID,
			"name":         skill.Name,
			"version":      skill.Version,
			"capabilities": metadata.Capabilities,
			"summary":      metadata.Description,
			"status":       skill.Status,
		})
	}

	return map[string]interface{}{
		"tools": tools,
		"total": o.skills.Count(),
	}, nil
}

// handleToolsCall handles mcp.tools.call requests
func (o *Orchestrator) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		ToolID      string          `json:"toolId"`
		Version     string          `json:"version"`
		SessionID   string          `json:"sessionId"`
		Input       json.RawMessage `json:"input"`
		CallOptions struct {
			Stream    bool `json:"stream"`
			TimeoutMs int  `json:"timeoutMs"`
			MaxTokens int  `json:"maxTokens"`
		} `json:"callOptions"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// Get skill
	skill, err := o.skills.Get(req.ToolID)
	if err != nil {
		return nil, fmt.Errorf("skill not found: %w", err)
	}

	log.Printf("[MCP] Tool call: skill=%s session=%s", skill.Name, req.SessionID)

	requestID := uuid.New().String()
	res, err := executor.Execute(ctx, skill, req.Input, req.CallOptions.TimeoutMs)
	if err != nil {
		return nil, err
	}

	// Update metrics
	o.met.IncrToolCall(skill.Name, res.Mode)

	// Persist execution in IA_Recuerdo for observability.
	o.memory.Save(ctx, &memory.Observation{ //nolint:errcheck
		Title: fmt.Sprintf("Tool call: %s [%s]", skill.Name, res.Mode),
		Content: fmt.Sprintf("**Skill**: %s v%s\n**Mode**: %s\n**Session**: %s\n**RequestID**: %s",
			skill.Name, skill.Version, res.Mode, req.SessionID, requestID),
		Type:     "discovery",
		TopicKey: fmt.Sprintf("mcp/tools/%s", skill.Name),
	})

	return map[string]interface{}{
		"status":    "completed",
		"requestId": requestID,
		"mode":      res.Mode,
		"output":    res.Output,
		"message":   fmt.Sprintf("Skill %s execution finished (%s)", skill.Name, res.Mode),
	}, nil
}

// handleToolsStatus handles mcp.tools.status requests
func (o *Orchestrator) handleToolsStatus(_ context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		RequestID string `json:"requestId"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if req.RequestID == "" {
		return nil, fmt.Errorf("requestId is required")
	}

	// Status tracking is in-memory only for now; async jobs are a Phase 4+ feature.
	return map[string]interface{}{
		"status":    "done",
		"requestId": req.RequestID,
		"progress":  1.0,
	}, nil
}
