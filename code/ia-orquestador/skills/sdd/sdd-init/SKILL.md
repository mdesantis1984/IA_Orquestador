# sdd-init — Bootstrap SDD Context

## Purpose
Initialize Spec-Driven Development context for a project. Detects repo structure, tooling, test frameworks, and establishes baseline state for orchestration.

## When to Use
- Starting a new SDD workflow on an existing codebase
- Onboarding a new developer/agent to a project
- Re-initializing after major refactor or tooling changes
- Preparing context for explore/propose/spec phases

## When NOT to Use
- Emergency hotfixes (skip directly to sdd-tasks + sdd-apply)
- Simple config changes with known scope
- When SDD context already exists and is recent (< 1 week)

## Inputs
- `project_root` (path): Root directory of the codebase
- `repo_url` (optional): Git remote URL for metadata
- `language` (optional): Primary language (auto-detected if omitted)
- `test_framework` (optional): Override auto-detection

## Outputs
- `sdd_context.json`: Structured metadata (language, frameworks, folder structure, entry points)
- `tooling_report.md`: Detected tools, CI/CD pipelines, linters, formatters
- `test_surface.md`: Test coverage summary, frameworks, integration test availability
- IA_Recuerdo observation: topic_key=`sdd/init/{project_name}`, type=`config`

## Workflow
1. Scan `project_root` for language indicators (package.json, go.mod, .csproj, etc.)
2. Detect test frameworks (Jest, xUnit, pytest, etc.)
3. Identify CI/CD configs (.github/workflows, .gitlab-ci.yml, Makefile)
4. Generate `sdd_context.json` with:
   - Detected language & version
   - Frameworks & dependencies
   - Folder structure (src/, tests/, docs/, configs/)
   - Entry points (main.go, Program.cs, index.js)
5. Save tooling report and test surface to artifacts
6. persist to IA_Recuerdo with topic_key for downstream phases

## Examples

### Example 1: Initialize Go project
```bash
# Agent receives:
{
  "skill": "sdd-init",
  "params": {
    "project_root": "/home/user/myapp",
    "repo_url": "https://github.com/user/myapp"
  }
}

# Outputs:
# /artifacts/sdd_context.json
{
  "language": "go",
  "version": "1.23",
  "test_framework": "testing",
  "ci_detected": "github-actions",
  "entry_points": ["cmd/server/main.go"],
  "test_coverage": "75%"
}
```

### Example 2: Initialize .NET solution
```bash
# Agent receives:
{
  "skill": "sdd-init",
  "params": {
    "project_root": "/home/user/MyApp.sln"
  }
}

# Outputs:
{
  "language": "csharp",
  "version": "net8.0",
  "test_framework": "xUnit",
  "projects": ["MyApp.Core", "MyApp.Web", "MyApp.Tests"],
  "ci_detected": "azure-pipelines"
}
```

## Gotchas & Edge Cases
- **Monorepos**: Detect multiple languages, generate per-project contexts
- **Missing tests**: Flag as warning, suggest test scaffolding
- **Custom build tools**: Require explicit `Makefile` or `build.sh` to detect properly
- **Private dependencies**: May fail on `go mod download` or `dotnet restore` without credentials

## Implementation Notes
- Use `git ls-files` to list tracked files (respects .gitignore)
- Prefer static analysis over compilation (faster, no side effects)
- Timeout: 30 seconds max for large repos
- Cache detection results in `.sdd/cache.json` for 24h

## References
- SDD workflow overview: `/workspace/03-SDD-Orchestration-Patterns.md`
- IA_Recuerdo topic_key conventions: `/workspace/AGENTS.md` Memory Protocol
- Test framework detection: Language-specific skill catalogs
