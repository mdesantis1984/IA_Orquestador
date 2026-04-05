package executor_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thiscloud/ia-orquestador/internal/executor"
	"github.com/thiscloud/ia-orquestador/pkg/types"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func skill(id, name string, t types.SkillType, entrypoint, path string) *types.Skill {
	return &types.Skill{
		ID:         id,
		Name:       name,
		Version:    "1.0.0",
		Type:       t,
		Entrypoint: entrypoint,
		Path:       path,
		Status:     types.SkillStatusActive,
	}
}

func rawJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// ── mock backend ─────────────────────────────────────────────────────────────

func TestExecute_NoEntrypoint_ReturnsMock(t *testing.T) {
	s := skill("1", "no-ep", types.SkillTypeHTTP, "", "")
	res, err := executor.Execute(context.Background(), s, rawJSON(map[string]string{"k": "v"}), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "mock" {
		t.Errorf("expected mode=mock, got %s", res.Mode)
	}
}

// ── skill-content backend ────────────────────────────────────────────────────

func TestExecute_SDDSkill_ReturnsSkillMDContent(t *testing.T) {
	// Create a temporary directory with SKILL.md
	dir := t.TempDir()
	content := "# Test Skill\nSome instructions."
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	s := skill("2", "sdd-test", types.SkillTypeSDD, "", dir)
	res, err := executor.Execute(context.Background(), s, rawJSON(map[string]string{"project": "/tmp/proj"}), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "skill-content" {
		t.Errorf("expected mode=skill-content, got %s", res.Mode)
	}

	out, ok := res.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map output, got %T", res.Output)
	}
	if out["content"] != content {
		t.Errorf("content mismatch")
	}
}

func TestExecute_DotNetSkill_ReturnsSkillMDContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# DotNet"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := skill("3", "blazor-server", types.SkillTypeDotNet, "", dir)
	res, err := executor.Execute(context.Background(), s, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "skill-content" {
		t.Errorf("expected mode=skill-content, got %s", res.Mode)
	}
}

func TestExecute_SDDSkill_NoSKILLmd_ReturnsFallbackMetadata(t *testing.T) {
	s := skill("4", "sdd-noskill", types.SkillTypeSDD, "", "/nonexistent/path")
	s.Metadata = rawJSON(map[string]interface{}{
		"description":  "A test skill",
		"capabilities": []string{"testing"},
	})

	res, err := executor.Execute(context.Background(), s, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "skill-metadata" {
		t.Errorf("expected mode=skill-metadata, got %s", res.Mode)
	}
}

// ── http backend ─────────────────────────────────────────────────────────────

func TestExecute_HTTPSkill_PostsAndReturnsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":"ok","echo":true}`)) //nolint:errcheck
	}))
	defer srv.Close()

	s := skill("5", "http-skill", types.SkillTypeHTTP, srv.URL, "")
	res, err := executor.Execute(context.Background(), s, rawJSON(map[string]string{"msg": "hello"}), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "http" {
		t.Errorf("expected mode=http, got %s", res.Mode)
	}
}

func TestExecute_HTTPSkill_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := skill("6", "http-fail", types.SkillTypeHTTP, srv.URL, "")
	_, err := executor.Execute(context.Background(), s, nil, 0)
	if err == nil {
		t.Error("expected error for HTTP 500, got nil")
	}
}

// ── local-exec backend ───────────────────────────────────────────────────────

func TestExecute_LocalSkill_NotFound_ReturnsMock(t *testing.T) {
	s := skill("7", "local-missing", types.SkillTypeHTTP, "/nonexistent/script.sh", "")
	res, err := executor.Execute(context.Background(), s, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "mock" {
		t.Errorf("expected mode=mock, got %s", res.Mode)
	}
}

func TestExecute_LocalSkill_EchoScript_ReturnsOutput(t *testing.T) {
	// Write an inline echo script
	dir := t.TempDir()
	script := filepath.Join(dir, "echo.sh")
	content := "#!/bin/sh\necho '{\"status\":\"ok\"}'"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}

	s := skill("8", "echo-local", types.SkillTypeHTTP, script, "")
	res, err := executor.Execute(context.Background(), s, rawJSON(map[string]string{"text": "hi"}), 5000)
	if err != nil {
		// On Windows without bash this may fail — skip the test
		t.Skipf("local exec not available: %v", err)
	}
	if res.Mode != "local-exec" {
		t.Errorf("expected mode=local-exec, got %s", res.Mode)
	}
}

// ── timeout ──────────────────────────────────────────────────────────────────

func TestExecute_HTTPSkill_Timeout_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the client cancels or a safety timeout expires.
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()

	s := skill("9", "slow-http", types.SkillTypeHTTP, srv.URL, "")
	_, err := executor.Execute(context.Background(), s, nil, 100) // 100ms timeout
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}
