# sdd-tasks — Task Breakdown

## Purpose
Break down design into granular implementation tasks: PRs, GitHub issues, branches, conventional commits. Defines work units for `sdd-apply`.

## When to Use
- After `sdd-design` to plan implementation
- Before `sdd-apply` to parallelize work
- When creating backlog for sprint planning
- For tracking progress across developers/agents

## When NOT to Use
- For single-file changes (no breakdown needed)
- When tasks are already defined in issue tracker

## Inputs
- `architecture.md` (from sdd-design)
- `requirements.md` (from sdd-spec)
- `team_size` (optional): Number of developers (affects parallelization)

## Outputs
- `tasks.yaml`: Structured task list with dependencies, estimates, assignees
- `github_issues.md`: Issue templates ready for creation
- `branch_strategy.md`: Git branching model (feature branches, PR workflow)
- `commit_conventions.md`: Conventional commit format for the feature
- IA_Recuerdo observation: topic_key=`sdd/tasks/{feature_name}`, type=`decision`

## Workflow
1. Load design and requirements
2. Identify implementation units (modules, files, tests)
3. Create tasks with:
   - ID (e.g., `TASK-001`)
   - Title (concise, actionable)
   - Description (acceptance criteria from specs)
   - Estimate (hours/story points)
   - Dependencies (other task IDs)
   - Assignee (developer/agent)
4. Define branch strategy (feature/jwt-auth, feature/jwt-auth-tests)
5. Define commit conventions (feat:, fix:, test:, docs:)
6. Generate GitHub issue templates
7. Save all artifacts and persist to IA_Recuerdo

## Examples

### Example 1: JWT Auth Tasks
```json
{
  "skill": "sdd-tasks",
  "params": {
    "architecture": "artifacts/architecture.md",
    "requirements": "artifacts/requirements.md"
  }
}
```

**Output `tasks.yaml`:**
```yaml
tasks:
  - id: TASK-001
    title: "Create User entity and migration"
    description: |
      - Define User struct with id, username, password_hash, created_at
      - Write migration SQL (CREATE TABLE users)
      - Add to migration runner
    estimate: 2h
    dependencies: []
    assignee: dev-1
    branch: feature/jwt-auth-user-model

  - id: TASK-002
    title: "Implement UserRepo with bcrypt password hashing"
    description: |
      - CreateUser(username, password) → hash password, insert to DB
      - GetUserByUsername(username) → fetch user
      - Write unit tests with test database
    estimate: 3h
    dependencies: [TASK-001]
    assignee: dev-1
    branch: feature/jwt-auth-repo

  - id: TASK-003
    title: "Implement JWTService (sign & verify)"
    description: |
      - Sign(userID, roles) → JWT string
      - Verify(token) → claims or error
      - Use HS256, 1-hour expiry
      - Unit tests with test secret
    estimate: 2h
    dependencies: []
    assignee: dev-2
    branch: feature/jwt-auth-service

  - id: TASK-004
    title: "Create /auth/login endpoint"
    description: |
      - POST handler: validate credentials, call JWTService
      - Return 200 + token or 401
      - Integration test with test DB
    estimate: 3h
    dependencies: [TASK-002, TASK-003]
    assignee: dev-1
    branch: feature/jwt-auth-login

  - id: TASK-005
    title: "Add JWT middleware to protect routes"
    description: |
      - Middleware: extract token, verify, inject userID to context
      - Return 401 if invalid/missing
      - Integration test with protected endpoint
    estimate: 2h
    dependencies: [TASK-003]
    assignee: dev-2
    branch: feature/jwt-auth-middleware
```

**Output `github_issues.md`:**
```markdown
## Issue Template: TASK-001

**Title:** Create User entity and migration

**Description:**
Implements `User` struct and database migration for JWT auth.

**Acceptance Criteria:**
- [ ] User struct with id, username, password_hash, created_at
- [ ] SQL migration creates `users` table
- [ ] Migration runs successfully on test DB

**Estimate:** 2 hours  
**Branch:** feature/jwt-auth-user-model  
**Labels:** enhancement, backend, database
```

**Output `commit_conventions.md`:**
```markdown
# Commit Conventions

Use Conventional Commits format:

- `feat(auth): add User entity and migration`
- `feat(auth): implement UserRepo with bcrypt`
- `test(auth): add integration tests for /auth/login`
- `docs(auth): update API spec with /auth/login endpoint`

**Scope:** `auth` for all JWT-related commits
**Types:** feat, fix, test, docs, refactor, chore
```

### Example 2: Blazor Component Tasks
```yaml
tasks:
  - id: TASK-001
    title: "Create UserProfileComponent.razor"
    description: "Implement component structure, props, dependency injection"
    estimate: 1h
    dependencies: []

  - id: TASK-002
    title: "Add IUserService interface and implementation"
    description: "GetUserAsync, UpdateUserAsync methods"
    estimate: 2h
    dependencies: []

  - id: TASK-003
    title: "Wire up component events (OnSave, OnCancel)"
    description: "Emit events, handle in parent component"
    estimate: 1h
    dependencies: [TASK-001]

  - id: TASK-004
    title: "Write unit tests for UserProfileComponent"
    description: "Use bUnit, test rendering, editing, saving"
    estimate: 2h
    dependencies: [TASK-001, TASK-002]
```

## Gotchas & Edge Cases
- **Overdecomposition**: Avoid tasks < 1 hour (merge small tasks)
- **Underdecomposition**: Avoid tasks > 8 hours (split large tasks)
- **Circular dependencies**: Flag and resolve during task creation
- **Parallel work conflicts**: Ensure tasks touching same files have dependencies

## Implementation Notes
- Use YAML for structured task lists (easier to parse than Markdown)
- Estimate granularity: 0.5h increments (round up for uncertainty)
- Branch naming: `feature/{feature-name}-{component}`
- Timeout: 2 minutes for task breakdown

## References
- SDD workflow: `/workspace/03-SDD-Orchestration-Patterns.md`
- Conventional Commits: https://www.conventionalcommits.org/
- GitHub issues API: For automated issue creation
- Story points vs. hours: Use hours for agent-driven tasks
