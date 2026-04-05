package skills

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/thiscloud/ia-orquestador/pkg/types"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create skills table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS skills (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			type TEXT NOT NULL,
			entrypoint TEXT NOT NULL,
			path TEXT,
			metadata TEXT,
			status TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			UNIQUE(name, version)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create skills table: %v", err)
	}

	return db
}

func TestNewRegistry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}

	if r.db != db {
		t.Error("Registry db not set correctly")
	}

	if r.cache == nil {
		t.Error("Registry cache not initialized")
	}
}

func TestRegistry_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	skill := &types.Skill{
		Name:       "test-skill",
		Version:    "1.0.0",
		Type:       types.SkillTypeSDD,
		Entrypoint: "main.py",
		Status:     types.SkillStatusActive,
		Metadata:   json.RawMessage(`{"description":"test skill"}`),
	}

	// Create
	err := r.Create(ctx, skill)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if skill.ID == "" {
		t.Error("Skill ID not generated")
	}

	// Get by ID
	retrieved, err := r.Get(skill.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Name != skill.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, skill.Name)
	}

	if retrieved.Version != skill.Version {
		t.Errorf("Version mismatch: got %s, want %s", retrieved.Version, skill.Version)
	}
}

func TestRegistry_GetByName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	skill := &types.Skill{
		Name:       "search-skill",
		Version:    "2.0.0",
		Type:       types.SkillTypeDotNet,
		Entrypoint: "Program.dll",
		Status:     types.SkillStatusActive,
	}

	err := r.Create(ctx, skill)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get by name with version
	retrieved, err := r.GetByName("search-skill", "2.0.0")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}

	if retrieved.ID != skill.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, skill.ID)
	}

	// Get by name without version
	retrieved, err = r.GetByName("search-skill", "")
	if err != nil {
		t.Fatalf("GetByName without version failed: %v", err)
	}

	if retrieved.Name != "search-skill" {
		t.Errorf("Name mismatch: got %s, want search-skill", retrieved.Name)
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)

	_, err := r.Get("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent skill")
	}
}

func TestRegistry_LoadAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	// Create multiple skills
	skills := []*types.Skill{
		{
			Name:       "skill-1",
			Version:    "1.0.0",
			Type:       types.SkillTypeSDD,
			Entrypoint: "main.py",
			Status:     types.SkillStatusActive,
		},
		{
			Name:       "skill-2",
			Version:    "1.0.0",
			Type:       types.SkillTypeDotNet,
			Entrypoint: "App.dll",
			Status:     types.SkillStatusActive,
		},
	}

	for _, skill := range skills {
		if err := r.Create(ctx, skill); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Clear cache
	r.cache = make(map[string]*types.Skill)

	// Load all
	err := r.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if r.Count() != 2 {
		t.Errorf("Expected 2 skills, got %d", r.Count())
	}
}

func TestRegistry_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	// Create skills with different types and statuses
	skills := []*types.Skill{
		{Name: "sdd-1", Version: "1.0", Type: types.SkillTypeSDD, Entrypoint: "main.py", Status: types.SkillStatusActive},
		{Name: "sdd-2", Version: "1.0", Type: types.SkillTypeSDD, Entrypoint: "main.py", Status: types.SkillStatusInactive},
		{Name: "dotnet-1", Version: "1.0", Type: types.SkillTypeDotNet, Entrypoint: "App.dll", Status: types.SkillStatusActive},
	}

	for _, skill := range skills {
		if err := r.Create(ctx, skill); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// List all
	all, err := r.List("", "", 100, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 skills, got %d", len(all))
	}

	// List active only
	active, err := r.List(types.SkillStatusActive, "", 100, 0)
	if err != nil {
		t.Fatalf("List active failed: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("Expected 2 active skills, got %d", len(active))
	}

	// List by type
	sddSkills, err := r.List("", types.SkillTypeSDD, 100, 0)
	if err != nil {
		t.Fatalf("List SDD failed: %v", err)
	}
	if len(sddSkills) != 2 {
		t.Errorf("Expected 2 SDD skills, got %d", len(sddSkills))
	}

	// Test pagination
	page1, err := r.List("", "", 2, 0)
	if err != nil {
		t.Fatalf("List page 1 failed: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("Expected 2 skills in page 1, got %d", len(page1))
	}

	page2, err := r.List("", "", 2, 2)
	if err != nil {
		t.Fatalf("List page 2 failed: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("Expected 1 skill in page 2, got %d", len(page2))
	}
}

func TestRegistry_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	skill := &types.Skill{
		Name:       "update-test",
		Version:    "1.0.0",
		Type:       types.SkillTypeSDD,
		Entrypoint: "main.py",
		Status:     types.SkillStatusActive,
	}

	err := r.Create(ctx, skill)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update
	time.Sleep(10 * time.Millisecond) // Ensure timestamp difference

	updates := map[string]interface{}{
		"name":    "updated-name",
		"version": "2.0.0",
		"status":  "inactive",
	}

	err = r.Update(ctx, skill.ID, updates)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	updated, err := r.Get(skill.ID)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}

	if updated.Name != "updated-name" {
		t.Errorf("Name not updated: got %s, want updated-name", updated.Name)
	}

	if updated.Version != "2.0.0" {
		t.Errorf("Version not updated: got %s, want 2.0.0", updated.Version)
	}

	if updated.Status != types.SkillStatusInactive {
		t.Errorf("Status not updated: got %s, want inactive", updated.Status)
	}

	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

func TestRegistry_Delete_Soft(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	skill := &types.Skill{
		Name:       "delete-test",
		Version:    "1.0.0",
		Type:       types.SkillTypeSDD,
		Entrypoint: "main.py",
		Status:     types.SkillStatusActive,
	}

	err := r.Create(ctx, skill)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Soft delete
	err = r.Delete(ctx, skill.ID, false)
	if err != nil {
		t.Fatalf("Soft delete failed: %v", err)
	}

	// Should still exist but inactive
	deleted, err := r.Get(skill.ID)
	if err != nil {
		t.Fatalf("Get after soft delete failed: %v", err)
	}

	if deleted.Status != types.SkillStatusInactive {
		t.Errorf("Status should be inactive, got %s", deleted.Status)
	}
}

func TestRegistry_Delete_Hard(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	skill := &types.Skill{
		Name:       "hard-delete-test",
		Version:    "1.0.0",
		Type:       types.SkillTypeSDD,
		Entrypoint: "main.py",
		Status:     types.SkillStatusActive,
	}

	err := r.Create(ctx, skill)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Hard delete
	err = r.Delete(ctx, skill.ID, true)
	if err != nil {
		t.Fatalf("Hard delete failed: %v", err)
	}

	// Should not exist
	_, err = r.Get(skill.ID)
	if err == nil {
		t.Error("Expected error for deleted skill")
	}
}

func TestRegistry_Watch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	// Watch for events
	events := r.Watch()

	// Create skill
	skill := &types.Skill{
		Name:       "watch-test",
		Version:    "1.0.0",
		Type:       types.SkillTypeSDD,
		Entrypoint: "main.py",
		Status:     types.SkillStatusActive,
	}

	err := r.Create(ctx, skill)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Should receive added event
	select {
	case event := <-events:
		if event.Type != "added" {
			t.Errorf("Expected 'added' event, got %s", event.Type)
		}
		if event.Skill.ID != skill.ID {
			t.Errorf("Event skill ID mismatch")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for added event")
	}

	// Update skill
	err = r.Update(ctx, skill.ID, map[string]interface{}{
		"name": "updated-watch",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Should receive updated event
	select {
	case event := <-events:
		if event.Type != "updated" {
			t.Errorf("Expected 'updated' event, got %s", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for updated event")
	}
}

func TestRegistry_Count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	if r.Count() != 0 {
		t.Errorf("Expected 0 skills, got %d", r.Count())
	}

	// Add skills
	for i := 0; i < 5; i++ {
		skill := &types.Skill{
			Name:       "skill-" + string(rune('A'+i)),
			Version:    "1.0.0",
			Type:       types.SkillTypeSDD,
			Entrypoint: "main.py",
			Status:     types.SkillStatusActive,
		}
		err := r.Create(ctx, skill)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	if r.Count() != 5 {
		t.Errorf("Expected 5 skills, got %d", r.Count())
	}
}

func TestDebugDeleteSoft(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := NewRegistry(db)
	ctx := context.Background()

	skill := &types.Skill{
		Name:       "delete-test-debug",
		Version:    "1.0.0",
		Type:       types.SkillTypeSDD,
		Entrypoint: "main.py",
		Status:     types.SkillStatusActive,
	}

	err := r.Create(ctx, skill)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	t.Logf("Before delete - ID: %s, Status: %s, Ptr: %p", skill.ID, skill.Status, skill)

	// Soft delete
	err = r.Delete(ctx, skill.ID, false)
	if err != nil {
		t.Fatalf("Soft delete failed: %v", err)
	}

	// Get from cache
	deleted, err := r.Get(skill.ID)
	if err != nil {
		t.Fatalf("Get after soft delete failed: %v", err)
	}

	t.Logf("After delete - ID: %s, Status: %s, Ptr: %p", deleted.ID, deleted.Status, deleted)
	t.Logf("Original skill - Status: %s, Ptr: %p", skill.Status, skill)

	if deleted.Status != types.SkillStatusInactive {
		t.Errorf("Status should be inactive, got %s", deleted.Status)
	}
}
