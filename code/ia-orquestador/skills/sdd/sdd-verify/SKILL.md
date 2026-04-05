# sdd-verify — Quality Gate

## Purpose
Verify implementation against specs: run tests, check coverage, validate against acceptance criteria, perform code review. Acts as quality gate before merge.

## When to Use
- After `sdd-apply` to validate changes
- Before merging to main/production
- As part of CI/CD pipeline
- For automated code review

## When NOT to Use
- Before implementation (specs verification is during design)
- For trivial changes with no tests

## Inputs
- `pr_url` or `commit_hash` (from sdd-apply)
- `test_scenarios.md` (from sdd-spec)
- `requirements.md` (from sdd-spec)
- `coverage_threshold` (optional, default: 80%)

## Outputs
- `verification_report.md`: Pass/fail status, test results, coverage, linting
- `review_comments.md`: Code review feedback (style, logic, best practices)
- `acceptance_status.yaml`: Per-scenario pass/fail mapping
- IA_Recuerdo observation: topic_key=`sdd/verify/{task_id}`, type=`bugfix` (if issues found) or `discovery`

## Workflow
1. Load PR/commit and specs
2. Run test suite (unit, integration, e2e)
3. Check test coverage against threshold
4. Validate against acceptance criteria from specs
5. Run linters and formatters
6. Perform automated code review:
   - Check for anti-patterns
   - Verify naming conventions
   - Detect security vulnerabilities (SAST)
7. Generate verification report with:
   - ✅ or ❌ for each acceptance criterion
   - Test results (passed/failed/skipped)
   - Coverage % (overall and per-module)
   - Linter warnings/errors
   - Review comments
8. Save artifacts and persist to IA_Recuerdo
9. Return pass/fail status

## Examples

### Example 1: Verify JWT Login Endpoint
```json
{
  "skill": "sdd-verify",
  "params": {
    "pr_url": "https://github.com/user/repo/pull/42",
    "test_scenarios": "artifacts/test_scenarios.md",
    "requirements": "artifacts/requirements.md"
  }
}
```

**Output `verification_report.md`:**
```markdown
# Verification Report: PR #42 - JWT Login Endpoint

## Test Results
| Suite       | Passed | Failed | Skipped | Duration |
|-------------|--------|--------|---------|----------|
| Unit        | 12     | 0      | 0       | 1.2s     |
| Integration | 3      | 0      | 0       | 2.5s     |
| **Total**   | **15** | **0**  | **0**   | **3.7s** |

## Coverage
| Package         | Coverage |
|-----------------|----------|
| internal/api    | 92%      |
| internal/model  | 100%     |
| pkg/jwt         | 88%      |
| **Overall**     | **91%**  | ✅ (threshold: 80%)

## Acceptance Criteria
| Criterion                                  | Status |
|--------------------------------------------|--------|
| Valid credentials → 200 + JWT              | ✅     |
| Invalid credentials → 401                  | ✅     |
| Token expires in 1 hour                    | ✅     |
| Token includes user ID and roles           | ✅     |
| Missing username/password → 400            | ✅     |

## Linting
- **golangci-lint**: 0 errors, 0 warnings
- **gofmt**: All files formatted
- **go vet**: No issues

## Code Review (Automated)
✅ **No critical issues**

**Suggestions:**
- Consider extracting magic number (3600) to constant `TokenExpirySeconds`
- Add logging for failed login attempts (security audit trail)

## Final Status: ✅ **PASS**
All acceptance criteria met. Ready to merge.
```

**Output `acceptance_status.yaml`:**
```yaml
scenarios:
  - id: SCENARIO-001
    title: "Successful login"
    status: PASS
    test_name: "TestLogin_ValidCredentials"

  - id: SCENARIO-002
    title: "Invalid password"
    status: PASS
    test_name: "TestLogin_InvalidPassword"

  - id: SCENARIO-003
    title: "Missing credentials"
    status: PASS
    test_name: "TestLogin_MissingFields"
```

### Example 2: Failed Verification (Low Coverage)
```json
{
  "skill": "sdd-verify",
  "params": {
    "commit_hash": "a1b2c3d",
    "test_scenarios": "artifacts/test_scenarios.md",
    "coverage_threshold": 85
  }
}
```

**Output:**
```markdown
# Verification Report: Commit a1b2c3d

## Test Results
| Suite       | Passed | Failed | Skipped |
|-------------|--------|--------|---------|
| Unit        | 8      | 2      | 1       |
| **Total**   | **8**  | **2**  | **1**   |

❌ **2 tests failed**

**Failures:**
1. `TestJWTService_Verify_ExpiredToken`: Expected error, got nil
2. `TestAuthHandler_Login_RateLimited`: Expected 429, got 200

## Coverage
| Package         | Coverage |
|-----------------|----------|
| internal/api    | 75%      | ❌ (threshold: 85%)
| pkg/jwt         | 60%      | ❌ (threshold: 85%)
| **Overall**     | **68%**  | ❌

**Missing coverage:**
- `pkg/jwt/verify.go`: Lines 45-52 (expired token check)
- `internal/api/auth_handler.go`: Lines 78-85 (rate limiting)

## Final Status: ❌ **FAIL**
- 2 test failures
- Coverage below threshold (68% < 85%)

**Action Required:**
1. Fix failing tests
2. Add tests for uncovered code paths
```

## Gotchas & Edge Cases
- **Flaky tests**: Re-run failed tests once before failing (detect flakiness)
- **External dependencies**: Mock or use test doubles (avoid real DBs/APIs in CI)
- **Long-running tests**: Timeout after 10 minutes
- **Coverage false positives**: Exclude generated code, test files from coverage

## Implementation Notes
- Run tests in isolated environment (Docker, VM, CI runner)
- Parse test output (JUnit XML, Go test JSON, xUnit XML)
- SAST tools: gosec, semgrep, SonarQube, Snyk
- Code review heuristics: cyclomatic complexity, file length, naming conventions
- Timeout: 15 minutes for full verification

## References
- SDD workflow: `/workspace/03-SDD-Orchestration-Patterns.md`
- Test frameworks: go test, xUnit, Jest, pytest
- Coverage tools: go test -cover, dotnet-coverage, istanbul
- SAST: gosec, Snyk, SonarQube, CodeQL
