# sdd-archive — Close and Sync

## Purpose
Finalize SDD workflow: sync specs with implementation, archive decision records, update documentation, close tasks. Ensures traceability and knowledge retention.

## When to Use
- After `sdd-verify` passes
- When feature is merged to main
- To close out SDD workflow
- For post-mortem and retrospective

## When NOT to Use
- Before verification is complete
- For abandoned/rejected features (use different workflow)

## Inputs
- `verification_report.md` (from sdd-verify)
- `proposal.md` (from sdd-propose)
- `tasks.yaml` (from sdd-tasks)
- `pr_url` (merged PR)

## Outputs
- `archive_summary.md`: What was built, what changed, lessons learned
- `updated_docs.md`: Documentation updates (README, API docs, ADRs)
- `closed_tasks.txt`: List of closed GitHub issues/tasks
- IA_Recuerdo observation: topic_key=`sdd/archive/{feature_name}`, type=`learning`

## Workflow
1. Load verification report and proposal
2. Confirm all tasks completed and verified
3. Update documentation:
   - Sync API specs with implemented endpoints
   - Update README with new features
   - Archive ADRs in `docs/architecture/decisions/`
4. Close GitHub issues/tasks
5. Generate archive summary:
   - **What**: Feature summary
   - **Why**: Original motivation (from proposal)
   - **Accomplished**: What was built
   - **Learned**: Gotchas, edge cases, deviations from design
   - **Next**: Follow-up work or technical debt
6. Save all artifacts and persist to IA_Recuerdo
7. Mark SDD workflow as complete

## Examples

### Example 1: Archive JWT Auth Feature
```json
{
  "skill": "sdd-archive",
  "params": {
    "verification_report": "artifacts/verification_report.md",
    "proposal": "artifacts/proposal.md",
    "pr_url": "https://github.com/user/repo/pull/42"
  }
}
```

**Output `archive_summary.md`:**
```markdown
# Archive Summary: JWT Authentication

## What
JWT-based authentication for REST API, replacing API key auth.

## Why
- Improve security with stateless tokens
- Enable user-specific permissions
- Prepare for future OAuth2 integration

## Accomplished
- ✅ User entity and database migration
- ✅ UserRepo with bcrypt password hashing
- ✅ JWTService (HS256, 1-hour expiry)
- ✅ `/auth/login` endpoint
- ✅ JWT middleware for protected routes
- ✅ Integration tests (15 tests, 91% coverage)

**Commits:**
- a1b2c3d: feat(auth): add User entity and migration
- b2c3d4e: feat(auth): implement UserRepo with bcrypt
- c3d4e5f: feat(auth): implement JWTService
- d4e5f6a: feat(auth): add /auth/login endpoint
- e5f6a7b: feat(auth): add JWT middleware

**PR:** https://github.com/user/repo/pull/42 (merged to main)

## Learned
- **Gotcha**: bcrypt default cost (10) was too slow for tests → used cost 4 for test DB
- **Edge case**: Expired tokens require separate error code (401 vs 403) for clients to refresh
- **Deviation**: Originally planned RS256, switched to HS256 for MVP simplicity (documented in ADR-001)

## Next Steps
- TODO: Add refresh token endpoint (for long-lived sessions)
- TODO: Implement role-based access control (RBAC)
- TODO: Migrate to RS256 when multi-service auth needed
- Technical debt: Extract magic numbers (token expiry, bcrypt cost) to config
```

**Output `updated_docs.md`:**
```markdown
## Documentation Updates

### README.md
Added section:
```markdown
## Authentication

This API uses JWT-based authentication. To authenticate:

1. POST to `/auth/login` with credentials:
   \`\`\`json
   {"username": "alice", "password": "secret123"}
   \`\`\`

2. Include token in subsequent requests:
   \`\`\`
   Authorization: Bearer <token>
   \`\`\`

Tokens expire after 1 hour.
\`\`\`

### API Documentation (OpenAPI)
Updated `docs/api/openapi.yaml` with `/auth/login` endpoint.

### ADRs
Archived `ADR-001-use-hs256-for-jwt.md` to `docs/architecture/decisions/`.
```

**Output `closed_tasks.txt`:**
```
Closed GitHub Issues:
- #101: TASK-001 - Create User entity and migration
- #102: TASK-002 - Implement UserRepo with bcrypt
- #103: TASK-003 - Implement JWTService
- #104: TASK-004 - Create /auth/login endpoint
- #105: TASK-005 - Add JWT middleware

All tasks completed and verified.
```

### Example 2: Archive Blazor Migration (Partial)
```json
{
  "skill": "sdd-archive",
  "params": {
    "verification_report": "artifacts/verification_report.md",
    "proposal": "artifacts/proposal.md"
  }
}
```

**Output:**
```markdown
# Archive Summary: ASPX → Blazor Migration (Phase 1)

## What
Migrated 5 high-traffic ASPX pages to Blazor Server (Strangler Fig pattern).

## Accomplished
- ✅ UserProfile page
- ✅ Dashboard page
- ✅ Settings page
- ❌ Reports page (deferred to Phase 2)
- ❌ Admin panel (deferred to Phase 2)

## Learned
- **Pattern**: Strangler Fig worked well — no downtime, gradual rollout
- **Challenge**: ASPX ViewState migration required manual data extraction
- **Win**: Blazor Server reduced page load time by 40% (2.5s → 1.5s)

## Next Steps
- Phase 2: Reports and Admin pages (Q2 2026)
- Consider Blazor WASM for offline-first pages
```

## Gotchas & Edge Cases
- **Incomplete features**: Document what was NOT built, why, and next steps
- **Divergence from design**: Explain deviations in "Learned" section
- **Failed tasks**: Archive separately, include root cause analysis
- **Documentation drift**: Verify docs match implementation (not just design)

## Implementation Notes
- Auto-close GitHub issues via API (use PR merge event or manual call)
- Save archive summary to `docs/features/{feature_name}.md`
- Update CHANGELOG.md with release notes
- Tag IA_Recuerdo observation with `topic_key="sdd/archive/{feature}"` for future retrieval
- Timeout: 5 minutes

## References
- SDD workflow: `/workspace/03-SDD-Orchestration-Patterns.md`
- ADR format: Michael Nygard's template
- Retrospective: What went well, what didn't, what to improve
- IA_Recuerdo topic_key: Persistent context for follow-up work
