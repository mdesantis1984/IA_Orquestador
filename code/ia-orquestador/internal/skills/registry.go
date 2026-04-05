// Package skills implements the dynamic skill registry
package skills

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/thiscloud/ia-orquestador/pkg/types"
)

// Registry manages MCP skills
type Registry struct {
	db       *sql.DB
	cache    map[string]*types.Skill
	cacheMu  sync.RWMutex
	watchers []chan SkillEvent
	watchMu  sync.RWMutex
}

// SkillEvent represents a skill lifecycle event
type SkillEvent struct {
	Type  string       `json:"type"` // added, updated, removed
	Skill *types.Skill `json:"skill"`
}

// NewRegistry creates a new skill registry
func NewRegistry(db *sql.DB) *Registry {
	return &Registry{
		db:       db,
		cache:    make(map[string]*types.Skill),
		watchers: make([]chan SkillEvent, 0),
	}
}

// LoadAll loads all skills from database into cache
func (r *Registry) LoadAll(ctx context.Context) error {
	log.Println("[SKILLS] Loading all skills from database")

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, version, type, entrypoint, path, metadata, status, created_at, updated_at
		FROM skills
		ORDER BY created_at DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to query skills: %w", err)
	}
	defer rows.Close()

	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	r.cache = make(map[string]*types.Skill)
	count := 0

	for rows.Next() {
		skill := &types.Skill{}
		var createdAt, updatedAt int64
		var path sql.NullString
		var metadataBytes []byte

		err := rows.Scan(
			&skill.ID,
			&skill.Name,
			&skill.Version,
			&skill.Type,
			&skill.Entrypoint,
			&path,
			&metadataBytes,
			&skill.Status,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			log.Printf("[SKILLS] Failed to scan skill: %v", err)
			continue
		}

		skill.Path = path.String
		if len(metadataBytes) > 0 {
			skill.Metadata = metadataBytes
		} else {
			skill.Metadata = json.RawMessage(`{}`)
		}
		skill.CreatedAt = time.Unix(createdAt, 0)
		skill.UpdatedAt = time.Unix(updatedAt, 0)

		r.cache[skill.ID] = skill
		count++
	}

	log.Printf("[SKILLS] Loaded %d skills", count)
	return nil
}

// Get retrieves a skill by ID
func (r *Registry) Get(id string) (*types.Skill, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	skill, exists := r.cache[id]
	if !exists {
		return nil, fmt.Errorf("skill not found: %s", id)
	}

	return skill, nil
}

// GetByName retrieves a skill by name and version
func (r *Registry) GetByName(name, version string) (*types.Skill, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	for _, skill := range r.cache {
		if skill.Name == name && (version == "" || skill.Version == version) {
			return skill, nil
		}
	}

	return nil, fmt.Errorf("skill not found: %s@%s", name, version)
}

// List returns all skills matching filters
func (r *Registry) List(status types.SkillStatus, skillType types.SkillType, limit, offset int) ([]*types.Skill, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	results := make([]*types.Skill, 0)

	for _, skill := range r.cache {
		// Apply filters
		if status != "" && skill.Status != status {
			continue
		}
		if skillType != "" && skill.Type != skillType {
			continue
		}
		results = append(results, skill)
	}

	// Apply pagination
	if offset >= len(results) {
		return []*types.Skill{}, nil
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end], nil
}

// Create adds a new skill to the registry
func (r *Registry) Create(ctx context.Context, skill *types.Skill) error {
	// Generate ID if not set
	if skill.ID == "" {
		skill.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	skill.CreatedAt = now
	skill.UpdatedAt = now

	// Insert into database
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO skills (id, name, version, type, entrypoint, path, metadata, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		skill.ID,
		skill.Name,
		skill.Version,
		skill.Type,
		skill.Entrypoint,
		sql.NullString{String: skill.Path, Valid: skill.Path != ""},
		skill.Metadata,
		skill.Status,
		skill.CreatedAt.Unix(),
		skill.UpdatedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to insert skill: %w", err)
	}

	// Update cache
	r.cacheMu.Lock()
	r.cache[skill.ID] = skill
	r.cacheMu.Unlock()

	// Notify watchers
	r.notifyWatchers(SkillEvent{Type: "added", Skill: skill})

	log.Printf("[SKILLS] Created skill: %s (%s)", skill.Name, skill.ID)
	return nil
}

// Update modifies an existing skill
func (r *Registry) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	r.cacheMu.RLock()
	skill, exists := r.cache[id]
	r.cacheMu.RUnlock()

	if !exists {
		return fmt.Errorf("skill not found: %s", id)
	}

	// Create a copy to avoid race
	updatedSkill := new(types.Skill)
	*updatedSkill = *skill

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		updatedSkill.Name = name
	}
	if version, ok := updates["version"].(string); ok {
		updatedSkill.Version = version
	}
	if status, ok := updates["status"].(types.SkillStatus); ok {
		updatedSkill.Status = status
	} else if statusStr, ok := updates["status"].(string); ok {
		updatedSkill.Status = types.SkillStatus(statusStr)
	}
	if metadata, ok := updates["metadata"]; ok {
		metaBytes, _ := json.Marshal(metadata)
		updatedSkill.Metadata = metaBytes
	}

	updatedSkill.UpdatedAt = time.Now()

	// Update database
	_, err := r.db.ExecContext(ctx, `
		UPDATE skills
		SET name = ?, version = ?, status = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`,
		updatedSkill.Name,
		updatedSkill.Version,
		updatedSkill.Status,
		updatedSkill.Metadata,
		updatedSkill.UpdatedAt.Unix(),
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update skill: %w", err)
	}

	// Update cache
	r.cacheMu.Lock()
	r.cache[id] = updatedSkill
	r.cacheMu.Unlock()

	// Notify watchers
	r.notifyWatchers(SkillEvent{Type: "updated", Skill: updatedSkill})

	log.Printf("[SKILLS] Updated skill: %s (%s)", updatedSkill.Name, id)
	return nil
}

// Delete removes a skill from the registry
func (r *Registry) Delete(ctx context.Context, id string, hard bool) error {
	if hard {
		// Hard delete from database
		_, err := r.db.ExecContext(ctx, "DELETE FROM skills WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete skill: %w", err)
		}

		// Remove from cache
		r.cacheMu.Lock()
		skill := r.cache[id]
		delete(r.cache, id)
		r.cacheMu.Unlock()

		// Notify watchers
		if skill != nil {
			r.notifyWatchers(SkillEvent{Type: "removed", Skill: skill})
		}

		log.Printf("[SKILLS] Hard deleted skill: %s", id)
	} else {
		// Soft delete (mark inactive)
		err := r.Update(ctx, id, map[string]interface{}{
			"status": types.SkillStatusInactive,
		})
		if err != nil {
			return err
		}
		log.Printf("[SKILLS] Soft deleted skill: %s", id)
	}

	return nil
}

// Watch registers a channel to receive skill events
func (r *Registry) Watch() <-chan SkillEvent {
	ch := make(chan SkillEvent, 10)

	r.watchMu.Lock()
	r.watchers = append(r.watchers, ch)
	r.watchMu.Unlock()

	return ch
}

// notifyWatchers sends an event to all registered watchers
func (r *Registry) notifyWatchers(event SkillEvent) {
	r.watchMu.RLock()
	defer r.watchMu.RUnlock()

	for _, watcher := range r.watchers {
		select {
		case watcher <- event:
		default:
			log.Println("[SKILLS] Watcher channel full, dropping event")
		}
	}
}

// Count returns the total number of skills
func (r *Registry) Count() int {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	return len(r.cache)
}

// StartHotReload starts a background goroutine that polls the database every
// interval and syncs the in-memory cache with the current DB state.
// Added skills are emitted as "added" events, updated as "updated", removed as "removed".
func (r *Registry) StartHotReload(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.reload(ctx); err != nil {
					log.Printf("[SKILLS] Hot-reload error: %v", err)
				}
			}
		}
	}()
	log.Printf("[SKILLS] Hot-reload enabled (interval: %s)", interval)
}

// reload syncs the in-memory cache against the current database state.
func (r *Registry) reload(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, version, type, entrypoint, path, metadata, status, created_at, updated_at
		FROM skills
		ORDER BY created_at DESC
	`)
	if err != nil {
		return fmt.Errorf("reload query: %w", err)
	}
	defer rows.Close()

	fresh := make(map[string]*types.Skill)
	for rows.Next() {
		skill := &types.Skill{}
		var createdAt, updatedAt int64
		var path sql.NullString
		var metadataBytes []byte
		if err := rows.Scan(
			&skill.ID, &skill.Name, &skill.Version, &skill.Type,
			&skill.Entrypoint, &path, &metadataBytes, &skill.Status,
			&createdAt, &updatedAt,
		); err != nil {
			log.Printf("[SKILLS] Hot-reload scan error: %v", err)
			continue
		}
		skill.Path = path.String
		if len(metadataBytes) > 0 {
			skill.Metadata = metadataBytes
		} else {
			skill.Metadata = json.RawMessage(`{}`)
		}
		skill.CreatedAt = time.Unix(createdAt, 0)
		skill.UpdatedAt = time.Unix(updatedAt, 0)
		fresh[skill.ID] = skill
	}

	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	// Added or updated
	for id, freshSkill := range fresh {
		cached, exists := r.cache[id]
		if !exists {
			r.cache[id] = freshSkill
			r.notifyWatchers(SkillEvent{Type: "added", Skill: freshSkill})
			log.Printf("[SKILLS] Hot-reload: added %s", freshSkill.Name)
		} else if freshSkill.UpdatedAt.After(cached.UpdatedAt) {
			r.cache[id] = freshSkill
			r.notifyWatchers(SkillEvent{Type: "updated", Skill: freshSkill})
			log.Printf("[SKILLS] Hot-reload: updated %s", freshSkill.Name)
		}
	}

	// Removed
	for id, cached := range r.cache {
		if _, exists := fresh[id]; !exists {
			delete(r.cache, id)
			r.notifyWatchers(SkillEvent{Type: "removed", Skill: cached})
			log.Printf("[SKILLS] Hot-reload: removed %s", cached.Name)
		}
	}

	return nil
}
