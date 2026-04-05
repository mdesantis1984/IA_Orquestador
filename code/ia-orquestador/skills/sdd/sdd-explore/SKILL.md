# sdd-explore — Codebase Investigation

## Purpose
Deep analysis of codebase structure, dependencies, risky areas, and test surface. Provides discovery notes to inform proposal and design phases.

## When to Use
- After `sdd-init` to understand the existing system
- Before making architectural changes
- Investigating legacy code or unfamiliar modules
- Identifying refactoring candidates

## When NOT to Use
- For well-known, recently-explored codebases (check IA_Recuerdo for recent explore results)
- Simple one-file changes
- Emergency hotfixes with clear scope

## Inputs
- `sdd_context.json` (from sdd-init)
- `focus_areas` (optional): List of modules/packages to prioritize
- `search_patterns` (optional): Regex/keywords to highlight (e.g., `TODO`, `FIXME`, deprecated APIs)

## Outputs
- `discovery_notes.md`: Key findings, architecture insights, data flow diagrams (text-based)
- `risky_areas.md`: Code smells, high-complexity modules, missing tests
- `test_coverage_detail.md`: Per-module coverage, integration test gaps
- `scope_suggestions.md`: Recommended scope for upcoming changes
- IA_Recuerdo observation: topic_key=`sdd/explore/{project_name}`, type=`discovery`

## Workflow
1. Load `sdd_context.json` to understand project structure
2. Scan for complexity metrics (cyclomatic complexity, file length, dependency graphs)
3. Identify untested or under-tested modules
4. Detect anti-patterns (God classes, circular dependencies, tight coupling)
5. Map data flows and external dependencies (DBs, APIs, message queues)
6. Highlight risky areas based on:
   - Low test coverage
   - High complexity
   - Recent churn (many commits)
   - Known vulnerabilities (if security scan available)
7. Generate summary and scope suggestions
8. Save all artifacts and persist to IA_Recuerdo

## Examples

### Example 1: Explore Go microservice
```json
{
  "skill": "sdd-explore",
  "params": {
    "sdd_context": "artifacts/sdd_context.json",
    "focus_areas": ["internal/api", "pkg/auth"]
  }
}
```

**Output `risky_areas.md`:**
```markdown
## High Complexity
- `internal/api/handler.go` (cyclomatic complexity: 45)
- `pkg/auth/jwt.go` (many nested conditionals)

## Low Test Coverage
- `pkg/auth/` (35% coverage)
- `internal/worker/` (0% coverage - no tests!)

## External Dependencies
- PostgreSQL (direct SQL queries, no ORM)
- Redis (no connection pooling)
```

### Example 2: Explore .NET Web App
```json
{
  "skill": "sdd-explore",
  "params": {
    "sdd_context": "artifacts/sdd_context.json",
    "search_patterns": ["TODO", "HACK"]
  }
}
```

**Output `discovery_notes.md`:**
```markdown
## Architecture
- Onion Architecture with MediatR
- EF Core with Code-First migrations
- SignalR for real-time updates

## Key Findings
- 15 TODOs in Controllers/ (deferred validations)
- No integration tests for SignalR hubs
- Startup.cs exceeds 500 lines (consider splitting)
```

## Gotchas & Edge Cases
- **Large codebases**: May timeout; use `focus_areas` to limit scope
- **External dependencies**: Require network access for dependency analysis
- **Compiled languages**: May need build artifacts for accurate analysis
- **Dynamic imports/reflection**: Hard to statically analyze, flag as unknown

## Implementation Notes
- Use AST parsing for accurate complexity metrics (go/parser, Roslyn for C#)
- Integrate with existing tools: SonarQube, gocyclo, Roslyn analyzers
- Timeout: 2 minutes for full scan, 30s for focused scan
- Cache dependency graphs for 1 hour

## References
- SDD workflow: `/workspace/03-SDD-Orchestration-Patterns.md`
- Complexity metrics: McCabe cyclomatic complexity, cognitive complexity
- Test coverage tools: go test -cover, dotnet-coverage, Jest --coverage
