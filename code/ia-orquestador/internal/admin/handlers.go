// Package admin provides the REST admin API for skill and API key management.
// All routes require a valid X-Api-Key header.
//
// Routes:
//
//	GET    /api/v1/skills           list skills (query: status, type)
//	POST   /api/v1/skills           create skill
//	GET    /api/v1/skills/{id}      get skill by id
//	PATCH  /api/v1/skills/{id}      update skill fields
//	DELETE /api/v1/skills/{id}      soft-delete (sets status=deprecated)
//	GET    /api/v1/tokens           list active API keys (no hash)
//	POST   /api/v1/tokens           create API key (returns plaintext once)
//	DELETE /api/v1/tokens/{id}      revoke API key
package admin

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/thiscloud/ia-orquestador/internal/auth"
	"github.com/thiscloud/ia-orquestador/internal/skills"
	"github.com/thiscloud/ia-orquestador/pkg/types"
)

// Handler implements the admin REST API.
type Handler struct {
	skills *skills.Registry
	auth   *auth.Validator
}

// New creates a new admin Handler.
func New(skillReg *skills.Registry, authValidator *auth.Validator) *Handler {
	return &Handler{skills: skillReg, auth: authValidator}
}

// RegisterRoutes registers all admin routes on mux, wrapped with the auth middleware.
// Uses Go 1.22+ enhanced ServeMux patterns (METHOD /path/{id}).
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	authed := h.auth.Middleware

	// Skills CRUD
	mux.Handle("GET /api/v1/skills", authed(http.HandlerFunc(h.listSkills)))
	mux.Handle("POST /api/v1/skills", authed(http.HandlerFunc(h.createSkill)))
	mux.Handle("GET /api/v1/skills/{id}", authed(http.HandlerFunc(h.getSkill)))
	mux.Handle("PATCH /api/v1/skills/{id}", authed(http.HandlerFunc(h.updateSkill)))
	mux.Handle("DELETE /api/v1/skills/{id}", authed(http.HandlerFunc(h.deleteSkill)))

	// Token management
	mux.Handle("GET /api/v1/tokens", authed(http.HandlerFunc(h.listTokens)))
	mux.Handle("POST /api/v1/tokens", authed(http.HandlerFunc(h.createToken)))
	mux.Handle("DELETE /api/v1/tokens/{id}", authed(http.HandlerFunc(h.revokeToken)))
}

// ── Skills ────────────────────────────────────────────────────────────────────

func (h *Handler) listSkills(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := types.SkillStatus(q.Get("status"))
	skillType := types.SkillType(q.Get("type"))

	list, err := h.skills.List(status, skillType, 500, 0)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"skills": list, "total": h.skills.Count()})
}

func (h *Handler) createSkill(w http.ResponseWriter, r *http.Request) {
	var skill types.Skill
	if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if skill.Name == "" || skill.Type == "" {
		jsonError(w, "name and type are required", http.StatusUnprocessableEntity)
		return
	}
	if skill.Version == "" {
		skill.Version = "1.0.0"
	}
	if skill.Status == "" {
		skill.Status = types.SkillStatusActive
	}
	if skill.ID == "" {
		skill.ID = uuid.New().String()
	}
	if skill.Metadata == nil {
		skill.Metadata = json.RawMessage(`{}`)
	}
	if err := h.skills.Create(r.Context(), &skill); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, skill)
}

func (h *Handler) getSkill(w http.ResponseWriter, r *http.Request) {
	skill, err := h.skills.Get(r.PathValue("id"))
	if err != nil {
		jsonError(w, "skill not found", http.StatusNotFound)
		return
	}
	jsonOK(w, skill)
}

func (h *Handler) updateSkill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := h.skills.Update(r.Context(), id, updates); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	skill, _ := h.skills.Get(id)
	jsonOK(w, skill)
}

func (h *Handler) deleteSkill(w http.ResponseWriter, r *http.Request) {
	if err := h.skills.Update(r.Context(), r.PathValue("id"), map[string]interface{}{
		"status": string(types.SkillStatusDeprecated),
	}); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Tokens ────────────────────────────────────────────────────────────────────

func (h *Handler) listTokens(w http.ResponseWriter, r *http.Request) {
	keys, err := h.auth.ListKeys(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"keys": keys})
}

func (h *Handler) createToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}
	key, err := h.auth.Generate(r.Context(), body.Name)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]string{
		"key":  key,
		"note": "Store this key securely — it will not be shown again.",
	})
}

func (h *Handler) revokeToken(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.Revoke(r.Context(), r.PathValue("id")); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}
