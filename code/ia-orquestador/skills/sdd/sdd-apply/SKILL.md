# sdd-apply — Implementation

## Purpose
Execute implementation tasks: write code, create commits/PRs, run tests, trigger CI/CD. Applies designs and specs to codebase.

## When to Use
- After `sdd-tasks` to execute planned work
- When ready to write code
- For automated implementation via agent
- To trigger CI/CD pipelines

## When NOT to Use
- For planning (that's sdd-tasks)
- For verification (that's sdd-verify)
- Without prior approval on production systems

## Inputs
- `tasks.yaml` (from sdd-tasks)
- `architecture.md` (from sdd-design)
- `api_spec.yaml` (from sdd-spec)
- `task_id` (required): Which task to execute (e.g., `TASK-001`)

## Outputs
- Code changes (committed to branch)
- `commit_log.txt`: List of commits created
- `pr_url.txt`: Pull request URL (if auto-created)
- `ci_status.txt`: CI/CD run status
- IA_Recuerdo observation: topic_key=`sdd/apply/{task_id}`, type=`pattern`

## Workflow
1. Load task by `task_id` from `tasks.yaml`
2. Check dependencies: ensure prerequisite tasks are completed
3. Create feature branch (if not exists)
4. Generate code based on:
   - Architecture design
   - API specs
   - Test scenarios (TDD: write tests first)
5. Run local tests (unit, integration)
6. Commit changes with conventional commit message
7. Push branch to remote
8. Create pull request (optional, based on config)
9. Trigger CI/CD (via webhook or API)
10. Save artifacts and persist to IA_Recuerdo

## Examples

### Example 1: Implement TASK-001 (User entity)
```json
{
  "skill": "sdd-apply",
  "params": {
    "tasks": "artifacts/tasks.yaml",
    "architecture": "artifacts/architecture.md",
    "task_id": "TASK-001"
  }
}
```

**Generated code (`internal/model/user.go`):**
```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID           uuid.UUID `db:"id" json:"id"`
    Username     string    `db:"username" json:"username"`
    PasswordHash string    `db:"password_hash" json:"-"`
    CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
```

**Generated migration (`migrations/001_create_users.sql`):**
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Output `commit_log.txt`:**
```
feat(auth): add User entity and migration

- Define User struct with id, username, password_hash, created_at
- Add SQL migration for users table
- Completes TASK-001

Refs: TASK-001
```

**Output `ci_status.txt`:**
```
Branch: feature/jwt-auth-user-model
Commit: a1b2c3d
CI Run: https://github.com/user/repo/actions/runs/12345
Status: ✅ Passed (tests: 15/15, coverage: 85%)
```

### Example 2: Implement TASK-004 (Login endpoint)
```json
{
  "skill": "sdd-apply",
  "params": {
    "tasks": "artifacts/tasks.yaml",
    "task_id": "TASK-004"
  }
}
```

**Generated code (`internal/api/auth_handler.go`):**
```go
package api

import (
    "encoding/json"
    "net/http"
    "myapp/internal/service"
)

type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Token     string `json:"token"`
    ExpiresAt string `json:"expires_at"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    user, err := h.userRepo.GetUserByUsername(req.Username)
    if err != nil || !service.VerifyPassword(user.PasswordHash, req.Password) {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    token, expiresAt, err := h.jwtService.Sign(user.ID, []string{"user"})
    if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(LoginResponse{Token: token, ExpiresAt: expiresAt})
}
```

**Generated test (`internal/api/auth_handler_test.go`):**
```go
package api_test

import (
    "bytes"
    "encoding/json"
    "net/http/httptest"
    "testing"
    "myapp/internal/api"
)

func TestLogin_ValidCredentials(t *testing.T) {
    // Setup test DB, mock services
    handler := setupAuthHandler(t)

    body := `{"username":"alice","password":"secret123"}`
    req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte(body)))
    w := httptest.NewRecorder()

    handler.Login(w, req)

    if w.Code != 200 {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    var resp api.LoginResponse
    json.NewDecoder(w.Body).Decode(&resp)

    if resp.Token == "" {
        t.Fatal("expected token, got empty string")
    }
}
```

## Gotchas & Edge Cases
- **Dependency failures**: If prerequisite task failed CI, block execution
- **Merge conflicts**: Auto-detect, flag for manual resolution
- **Test failures**: Stop execution, rollback commit, report errors
- **Rate limits**: GitHub/GitLab API limits (use PAT, respect 429)
- **Large diffs**: PRs > 500 lines may need manual review gates

## Implementation Notes
- TDD: Write tests before implementation code
- Run tests locally before pushing (avoid CI churn)
- Auto-format code (gofmt, dotnet format, prettier)
- Lint before commit (golangci-lint, eslint, Roslyn analyzers)
- Timeout: 10 minutes per task (abort if exceeded)
- Atomic commits: One task = one commit (or multiple if logical)

## Safety
- **Production**: Require explicit approval before applying to main/prod branches
- **Destructive changes**: Preview diffs, confirm before commit
- **Secrets**: Never commit secrets (use .gitignore, pre-commit hooks)

## References
- SDD workflow: `/workspace/03-SDD-Orchestration-Patterns.md`
- TDD: Write failing test → implement → verify green
- CI/CD: GitHub Actions, GitLab CI, Jenkins webhooks
- Code generation: Use templates, AST manipulation (go/ast, Roslyn)
