package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:7438", "test-project", "")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.baseURL != "http://localhost:7438" {
		t.Errorf("BaseURL mismatch: got %s, want http://localhost:7438", client.baseURL)
	}

	if client.project != "test-project" {
		t.Errorf("Project mismatch: got %s, want test-project", client.project)
	}
}

func TestClient_Save(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		
		if r.URL.Path != "/api/v1/observations" {
			t.Errorf("Expected /api/v1/observations, got %s", r.URL.Path)
		}
		
		var req map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		
		if req["title"] != "Test Observation" {
			t.Errorf("Title mismatch: got %v", req["title"])
		}
		
		if req["type"] != "discovery" {
			t.Errorf("Type mismatch: got %v", req["type"])
		}
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "obs-123",
			"message": "Observation saved",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-project", "")

	err := client.Save(context.Background(), &Observation{
		Title:   "Test Observation",
		Content: "Test content",
		Type:    "discovery",
		Scope:   "project",
	})
	
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestClient_Save_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "test-project", "")

	err := client.Save(context.Background(), &Observation{
		Title:   "Test",
		Content: "Content",
		Type:    "discovery",
	})

	if err == nil {
		t.Error("Expected error for server error")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Error should mention status 500: %v", err)
	}
}

func TestClient_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		
		if r.URL.Path != "/api/v1/search" {
			t.Errorf("Expected /api/v1/search, got %s", r.URL.Path)
		}
		
		query := r.URL.Query().Get("q")
		if query != "test query" {
			t.Errorf("Query mismatch: got %s", query)
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"id":      1,
					"title":   "Result 1",
					"content": "Content 1",
					"score":   0.95,
				},
			},
			"total": 1,
		})
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "test-project", "")

	results, err := client.Search(context.Background(), "test query", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	
	if results[0].Title != "Result 1" {
		t.Errorf("Title mismatch: got %s", results[0].Title)
	}
}

func TestClient_Search_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{},
			"total":   0,
		})
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "test-project", "")

	results, err := client.Search(context.Background(), "nonexistent", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestClient_Context(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/context" {
			t.Errorf("Expected /api/v1/context, got %s", r.URL.Path)
		}
		
		limit := r.URL.Query().Get("limit")
		if limit != "5" {
			t.Errorf("Limit mismatch: got %s, want 5", limit)
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"context": []map[string]interface{}{
				{
					"session_id": 1,
					"goal":       "Test session",
				},
			},
		})
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "test-project", "")

	results, err := client.Context(context.Background(), 5)
	if err != nil {
		t.Fatalf("Context failed: %v", err)
	}
	
	if len(results) == 0 {
		t.Error("Expected non-empty results")
	}
}

func TestClient_SaveDecision(t *testing.T) {
	var captured map[string]interface{}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "obs-decision",
		})
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "test-project", "")

	err := client.SaveDecision(context.Background(), "Test Decision", "Test what", "Test why", "config.yaml", "Test learned")
	if err != nil {
		t.Fatalf("SaveDecision failed: %v", err)
	}
	
	if captured["type"] != "decision" {
		t.Errorf("Type should be 'decision', got %v", captured["type"])
	}
	
	if captured["title"] != "Test Decision" {
		t.Errorf("Title mismatch: got %v", captured["title"])
	}
	
	content, ok := captured["content"].(string)
	if !ok {
		t.Fatal("Content should be string")
	}
	
	if !strings.Contains(content, "Test why") {
		t.Error("Content should contain 'why' rationale")
	}
	
	if !strings.Contains(content, "config.yaml") {
		t.Error("Content should contain affected files")
	}
}

func TestClient_SaveProgress(t *testing.T) {
	var captured map[string]interface{}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "obs-progress",
		})
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "test-project", "")

	err := client.SaveProgress(context.Background(), "Test Task", "In progress", "Step 1 done", "tasks/test-task")
	if err != nil {
		t.Fatalf("SaveProgress failed: %v", err)
	}
	
	if captured["type"] != "discovery" {
		t.Errorf("Type should be 'discovery', got %v", captured["type"])
	}
	
	topicKey, ok := captured["topic_key"].(string)
	if !ok || !strings.HasPrefix(topicKey, "tasks/") {
		t.Errorf("topic_key should start with 'tasks/', got %v", topicKey)
	}
}

func TestClient_Timeout(t *testing.T) {
	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "test-project", "")

	// Context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	err := client.Save(ctx, &Observation{
		Title:   "Timeout Test",
		Content: "Should timeout",
		Type:    "discovery",
	})
	
	if err == nil {
		t.Error("Expected timeout error")
	}
	
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Error should mention timeout: %v", err)
	}
}

func TestClient_InvalidURL(t *testing.T) {
	client := NewClient("http://invalid-nonexistent-host-12345.local", "test", "")
	
	err := client.Save(context.Background(), &Observation{
		Title:   "Test",
		Content: "Test",
		Type:    "discovery",
	})
	
	if err == nil {
		t.Error("Expected error for invalid host")
	}
}
