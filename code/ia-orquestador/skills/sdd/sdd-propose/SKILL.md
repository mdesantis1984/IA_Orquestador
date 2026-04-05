# sdd-propose — Create Change Proposal

## Purpose
Generate structured change proposals based on exploration findings. Defines intent, scope, impact, and preferred approach for upcoming work.

## When to Use
- After `sdd-explore` to formalize change scope
- When multiple approaches exist and need evaluation
- Before committing to spec/design phases
- To communicate intent to stakeholders/reviewers

## When NOT to Use
- For trivial changes with obvious solutions
- When the approach is already decided and documented
- Emergency hotfixes (skip to sdd-tasks)

## Inputs
- `discovery_notes.md` (from sdd-explore)
- `risky_areas.md` (from sdd-explore)
- `user_intent` (string): High-level goal (e.g., "Add authentication", "Refactor API layer")
- `constraints` (optional): Budget, timeline, backward compatibility requirements

## Outputs
- `proposal.md`: Structured proposal with:
  - Intent & motivation
  - Scope (what changes, what stays)
  - Impact analysis (affected modules, breaking changes)
  - Alternatives considered
  - Recommended approach
  - Risk assessment
- IA_Recuerdo observation: topic_key=`sdd/propose/{feature_name}`, type=`decision`

## Workflow
1. Load exploration artifacts
2. Analyze `user_intent` against `risky_areas` and `discovery_notes`
3. Generate 2-3 alternative approaches:
   - Quick fix (minimal changes, technical debt)
   - Balanced (pragmatic, medium effort)
   - Ideal (long-term, higher effort)
4. Evaluate each approach:
   - Effort estimation (hours/days)
   - Risk level (low/medium/high)
   - Backward compatibility
   - Test requirements
5. Recommend preferred approach with justification
6. Save proposal and persist to IA_Recuerdo

## Examples

### Example 1: Add JWT Authentication
```json
{
  "skill": "sdd-propose",
  "params": {
    "discovery_notes": "artifacts/discovery_notes.md",
    "user_intent": "Add JWT-based authentication to REST API"
  }
}
```

**Output `proposal.md`:**
```markdown
# Proposal: JWT Authentication for REST API

## Intent
Secure REST endpoints with JWT-based authentication to replace API key auth.

## Scope
**In scope:**
- Middleware for JWT validation
- `/auth/login` endpoint
- User model with hashed passwords

**Out of scope:**
- OAuth2/OIDC integration (future phase)
- Role-based access control (separate feature)

## Alternatives

### 1. Quick Fix: Add JWT to existing middleware
- Effort: 4 hours
- Risk: Medium (tight coupling, hard to test)

### 2. Balanced: New auth package with dependency injection
- Effort: 8 hours
- Risk: Low (clean separation, testable)
- **Recommended**

### 3. Ideal: Full identity microservice
- Effort: 3 days
- Risk: High (overengineering for current scale)

## Impact Analysis
- Breaking change: Clients must include `Authorization: Bearer <token>`
- Migration: Provide API key → JWT conversion endpoint for 30 days

## Risk Assessment
- **Low risk**: Well-understood pattern, existing libraries (golang-jwt)
- **Mitigation**: Comprehensive integration tests, feature flag rollout
```

### Example 2: Refactor Legacy ASPX to Blazor
```json
{
  "skill": "sdd-propose",
  "params": {
    "discovery_notes": "artifacts/discovery_notes.md",
    "user_intent": "Migrate ASPX pages to Blazor Server",
    "constraints": {"timeline": "6 months", "budget": "2 devs"}
  }
}
```

**Output:**
```markdown
# Proposal: ASPX → Blazor Server Migration

## Alternatives

### 1. Big Bang Rewrite
- Effort: 6 months
- Risk: **High** (business disruption, regression risk)

### 2. Strangler Fig Pattern (Incremental)
- Effort: 8 months
- Risk: Low (gradual rollout, A/B testing)
- **Recommended**

### 3. Hybrid (ASPX + Blazor coexist)
- Effort: 3 months
- Risk: Medium (maintenance overhead, mixed architecture)

## Recommended Approach: Strangler Fig
1. Deploy Blazor Server alongside ASPX
2. Migrate pages by priority (high-traffic first)
3. Use reverse proxy to route by URL pattern
4. Decommission ASPX modules incrementally
```

## Gotchas & Edge Cases
- **Conflicting constraints**: Flag when timeline and quality goals conflict
- **Missing context**: If exploration is incomplete, recommend re-running sdd-explore
- **Stakeholder disagreement**: Proposal is input to human decision, not final verdict
- **Over-specification**: Keep proposals high-level; defer details to sdd-spec

## Implementation Notes
- Use templated proposal format (Markdown with standard headings)
- Estimate effort based on historical data (query IA_Recuerdo for similar tasks)
- Risk scoring: combine complexity, test coverage, team familiarity
- Timeout: 1 minute for proposal generation

## References
- SDD workflow: `/workspace/03-SDD-Orchestration-Patterns.md`
- Effort estimation: Use story points or T-shirt sizes (S/M/L/XL)
- Decision records: Save proposals as ADRs (Architecture Decision Records)
