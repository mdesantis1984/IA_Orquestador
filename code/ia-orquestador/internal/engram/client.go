// Package memory implements the IA_Recuerdo HTTP client.
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is an HTTP client for the IA_Recuerdo memory service.
type Client struct {
	baseURL string
	project string
	apiKey  string
	http    *http.Client
}

// Observation represents a memory observation to persist.
type Observation struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Type     string   `json:"type"`
	Scope    string   `json:"scope,omitempty"`
	TopicKey string   `json:"topic_key,omitempty"`
	Project  string   `json:"project,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

// SearchResult represents a single search result from IA_Recuerdo.
type SearchResult struct {
	ID      int     `json:"id"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// NewClient creates a new IA_Recuerdo HTTP client.
func NewClient(baseURL, project, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		project: project,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Save persists an observation in IA_Recuerdo.
// Errors are non-fatal by design — callers should log them rather than fail.
func (c *Client) Save(ctx context.Context, obs *Observation) error {
	if obs.Project == "" {
		obs.Project = c.project
	}

	body, err := json.Marshal(obs)
	if err != nil {
		return fmt.Errorf("marshal observation: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/observations", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("save observation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("save observation: server returned %d", resp.StatusCode)
	}
	return nil
}

// Search performs a full-text search in IA_Recuerdo.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("project", c.project)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/v1/search?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search: server returned %d", resp.StatusCode)
	}

	var response struct {
		Results []SearchResult `json:"results"`
		Total   int            `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	return response.Results, nil
}

// Context retrieves recent session context from IA_Recuerdo.
func (c *Client) Context(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("project", c.project)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/v1/context?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("context: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("context: server returned %d", resp.StatusCode)
	}

	var response struct {
		Context []map[string]interface{} `json:"context"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode context response: %w", err)
	}
	return response.Context, nil
}

// SaveDecision is a convenience helper to persist an architectural/technical decision.
func (c *Client) SaveDecision(ctx context.Context, title, what, why, files, learned string) error {
	content := fmt.Sprintf("**What**: %s\n**Why**: %s\n**Where**: %s\n**Learned**: %s",
		what, why, files, learned)
	return c.Save(ctx, &Observation{
		Title:   title,
		Content: content,
		Type:    "decision",
		Project: c.project,
	})
}

// SaveProgress is a convenience helper to persist task progress.
func (c *Client) SaveProgress(ctx context.Context, task, status, progress, topicKey string) error {
	content := fmt.Sprintf("**Task**: %s\n**Status**: %s\n**Progress**: %s",
		task, status, progress)
	return c.Save(ctx, &Observation{
		Title:    fmt.Sprintf("Progress: %s", task),
		Content:  content,
		Type:     "discovery",
		TopicKey: topicKey,
		Project:  c.project,
	})
}
