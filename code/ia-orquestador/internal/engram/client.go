// Package memory implements the IA_Recuerdo HTTP client for memory persistence
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Client is an HTTP client for IA_Recuerdo memory service
type Client struct {
	baseURL    string
	httpClient *http.Client
	project    string
	apiKey     string
}

// NewClient creates a new IA_Recuerdo client
func NewClient(baseURL, project, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		project: project,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// applyAuth sets the X-Api-Key header if an API key is configured.
func (c *Client) applyAuth(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
}

// Observation represents an IA_Recuerdo observation
type Observation struct {
	Title    string                 `json:"title"`
	Content  string                 `json:"content"`
	Type     string                 `json:"type"`
	Project  string                 `json:"project,omitempty"`
	Scope    string                 `json:"scope,omitempty"`
	TopicKey string                 `json:"topic_key,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SearchResult represents a search result from IA_Recuerdo
type SearchResult struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	Scope     string    `json:"scope"`
	TopicKey  string    `json:"topic_key"`
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score,omitempty"`
}

// ContextResult represents recent context from IA_Recuerdo
type ContextResult struct {
	SessionID   int64     `json:"session_id"`
	Goal        string    `json:"goal"`
	StartedAt   time.Time `json:"started_at"`
	Summary     string    `json:"summary,omitempty"`
	Observation *SearchResult `json:"observation,omitempty"`
}

// Save stores an observation in IA_Recuerdo
func (c *Client) Save(ctx context.Context, obs *Observation) error {
	if obs.Project == "" {
		obs.Project = c.project
	}
	if obs.Scope == "" {
		obs.Scope = "project"
	}

	payload, err := json.Marshal(obs)
	if err != nil {
		return fmt.Errorf("failed to marshal observation: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/api/v1/observations", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ia-recuerdo error %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[IA_RECUERDO] Saved observation: %s (topic: %s)", obs.Title, obs.TopicKey)
	return nil
}

// Search performs a full-text search in IA_Recuerdo
func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", fmt.Sprintf("%d", limit))
	if c.project != "" {
		params.Set("project", c.project)
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/v1/search?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ia-recuerdo error %d: %s", resp.StatusCode, string(body))
	}

	var results struct {
		Results []SearchResult `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("[IA_RECUERDO] Search returned %d results for: %s", len(results.Results), query)
	return results.Results, nil
}

// Context retrieves recent session context from IA_Recuerdo
func (c *Client) Context(ctx context.Context, limit int) ([]ContextResult, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	if c.project != "" {
		params.Set("project", c.project)
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/v1/context?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ia-recuerdo error %d: %s", resp.StatusCode, string(body))
	}

	var results struct {
		Context []ContextResult `json:"context"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("[IA_RECUERDO] Context returned %d entries", len(results.Context))
	return results.Context, nil
}

// SaveDecision is a convenience method for saving decision observations in IA_Recuerdo
func (c *Client) SaveDecision(ctx context.Context, title, what, why, where, learned string) error {
	content := fmt.Sprintf("**What**: %s\n**Why**: %s\n**Where**: %s", what, why, where)
	if learned != "" {
		content += fmt.Sprintf("\n**Learned**: %s", learned)
	}
	
	return c.Save(ctx, &Observation{
		Title:   title,
		Content: content,
		Type:    "decision",
	})
}

// SaveProgress saves task progress to IA_Recuerdo with topic_key for updates
func (c *Client) SaveProgress(ctx context.Context, taskName, status, progress, topicKey string) error {
	content := fmt.Sprintf("**Status**: %s\n**Progress**: %s", status, progress)
	
	return c.Save(ctx, &Observation{
		Title:    fmt.Sprintf("Task: %s", taskName),
		Content:  content,
		Type:     "discovery",
		TopicKey: topicKey,
	})
}
