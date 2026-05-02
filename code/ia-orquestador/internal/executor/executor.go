// Package executor handles MCP skill execution with multiple backends:
//   - skill-content : SDD / DotNet / Wasm skills — returns SKILL.md for AI context
//   - local-exec    : Shell/binary scripts — runs the entrypoint process
//   - http          : Remote HTTP skill — POSTs input JSON to the entrypoint URL
//   - mock          : Fallback when no real execution is possible
package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"github.com/thiscloud/ia-orquestador/pkg/types"
)

// Result is the output of any skill execution backend.
type Result struct {
	Output interface{} `json:"output"`
	Mode   string      `json:"mode"` // skill-content | skill-metadata | local-exec | http | mock
}

// httpClient is reused across calls; timeout is enforced via context.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// Execute dispatches to the correct backend based on skill type and entrypoint.
func Execute(ctx context.Context, skill *types.Skill, input json.RawMessage, timeoutMs int) (*Result, error) {
	tracer := otel.Tracer("ia-orquestador/executor")
	ctx, span := tracer.Start(ctx, "skill.execute")
	span.SetAttributes(
		attribute.String("skill.id", skill.ID),
		attribute.String("skill.name", skill.Name),
		attribute.String("skill.type", string(skill.Type)),
	)
	defer span.End()

	if timeoutMs <= 0 {
		timeoutMs = 60000
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// SDD, DotNet and Wasm skills deliver SKILL.md content as AI context.
	if skill.Type == types.SkillTypeSDD || skill.Type == types.SkillTypeDotNet || skill.Type == types.SkillTypeWASM {
		return executeSkillContent(execCtx, skill, input)
	}

	// No entrypoint configured → return informative mock.
	if skill.Entrypoint == "" {
		return mockResult(skill, "no entrypoint configured"), nil
	}

	// Remote HTTP entrypoint.
	if strings.HasPrefix(skill.Entrypoint, "http://") || strings.HasPrefix(skill.Entrypoint, "https://") {
		return executeHTTP(execCtx, skill, input)
	}

	// Local script / binary.
	return executeLocal(execCtx, skill, input)
}

// executeSkillContent reads SKILL.md from skill.Path and returns its content
// so the calling AI agent can use those instructions as context.
func executeSkillContent(_ context.Context, skill *types.Skill, input json.RawMessage) (*Result, error) {
	candidates := buildSkillMDCandidates(skill)

	for _, p := range candidates {
		content, err := os.ReadFile(p) //#nosec G304 -- path is admin-registered
		if err == nil {
			var inputMap map[string]interface{}
			json.Unmarshal(input, &inputMap) //nolint:errcheck

			return &Result{
				Output: map[string]interface{}{
					"skill":   skill.Name,
					"version": skill.Version,
					"type":    string(skill.Type),
					"content": string(content),
					"input":   inputMap,
				},
				Mode: "skill-content",
			}, nil
		}
	}

	// SKILL.md not on disk — return structured metadata as fallback.
	var metadata types.SkillMetadata
	json.Unmarshal(skill.Metadata, &metadata) //nolint:errcheck

	return &Result{
		Output: map[string]interface{}{
			"skill":        skill.Name,
			"version":      skill.Version,
			"type":         string(skill.Type),
			"description":  metadata.Description,
			"capabilities": metadata.Capabilities,
			"note":         fmt.Sprintf("SKILL.md not found (tried: %s); returning metadata", strings.Join(candidates, ", ")),
		},
		Mode: "skill-metadata",
	}, nil
}

// executeHTTP POSTs input JSON to the skill's HTTP entrypoint and returns the response.
func executeHTTP(ctx context.Context, skill *types.Skill, input json.RawMessage) (*Result, error) {
	if len(input) == 0 || string(input) == "null" {
		input = json.RawMessage(`{}`)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, skill.Entrypoint, bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("build HTTP request for skill %s: %w", skill.Name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP call to skill %s failed: %w", skill.Name, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from skill %s: %w", skill.Name, err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("skill %s returned HTTP %d: %s", skill.Name, resp.StatusCode, string(body))
	}

	var parsed interface{}
	if json.Unmarshal(body, &parsed) == nil {
		return &Result{Output: parsed, Mode: "http"}, nil
	}
	return &Result{Output: map[string]interface{}{"raw": string(body)}, Mode: "http"}, nil
}

// executeLocal runs a local script or binary, passing input on stdin and
// collecting JSON (or raw text) from stdout.
func executeLocal(ctx context.Context, skill *types.Skill, input json.RawMessage) (*Result, error) {
	if _, err := os.Stat(skill.Entrypoint); err != nil {
		return mockResult(skill, fmt.Sprintf("entrypoint not found at %s", skill.Entrypoint)), nil
	}

	stdin := input
	if len(stdin) == 0 || string(stdin) == "null" {
		stdin = json.RawMessage(`{}`)
	}

	cmd := exec.CommandContext(ctx, skill.Entrypoint) //#nosec G204 -- entrypoint is admin-registered
	cmd.Stdin = bytes.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("skill %s: %w stderr=%s", skill.Name, err, strings.TrimSpace(stderr.String()))
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return &Result{Output: map[string]interface{}{}, Mode: "local-exec"}, nil
	}

	var parsed interface{}
	if json.Unmarshal([]byte(out), &parsed) == nil {
		return &Result{Output: parsed, Mode: "local-exec"}, nil
	}
	return &Result{Output: map[string]interface{}{"raw": out}, Mode: "local-exec"}, nil
}

// buildSkillMDCandidates returns possible SKILL.md paths in priority order.
func buildSkillMDCandidates(skill *types.Skill) []string {
	var candidates []string
	if skill.Path != "" {
		base := strings.TrimRight(skill.Path, "/")
		candidates = append(candidates, base+"/SKILL.md")
	}
	if strings.HasSuffix(skill.Entrypoint, ".md") {
		candidates = append(candidates, skill.Entrypoint)
	}
	return candidates
}

// mockResult returns a non-error response explaining why execution was skipped.
func mockResult(skill *types.Skill, reason string) *Result {
	return &Result{
		Output: map[string]interface{}{
			"skill":   skill.Name,
			"message": reason,
		},
		Mode: "mock",
	}
}
