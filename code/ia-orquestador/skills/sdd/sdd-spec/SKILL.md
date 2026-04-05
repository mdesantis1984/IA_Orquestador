# sdd-spec — Write Specifications

## Purpose
Create executable specifications: requirements, user stories, acceptance criteria, API contracts, and test scenarios. Bridges proposal and implementation.

## When to Use
- After `sdd-propose` approval to formalize requirements
- Before `sdd-design` to define "what" (not "how")
- When TDD/BDD is part of workflow
- To align stakeholders on expected behavior

## When NOT to Use
- For implementation details (that's sdd-design)
- Trivial changes with obvious specs
- When specs already exist and are current

## Inputs
- `proposal.md` (from sdd-propose)
- `user_stories` (optional): Pre-written stories from product owner
- `api_contracts` (optional): OpenAPI/Protobuf definitions

## Outputs
- `requirements.md`: Functional and non-functional requirements
- `user_stories.md`: User stories with acceptance criteria (Given/When/Then)
- `api_spec.yaml`: API contracts (OpenAPI, GraphQL schema, Protobuf)
- `test_scenarios.md`: BDD scenarios (Gherkin-style)
- IA_Recuerdo observation: topic_key=`sdd/spec/{feature_name}`, type=`architecture`

## Workflow
1. Load approved proposal
2. Extract functional requirements from proposal intent and scope
3. Define non-functional requirements (performance, security, scalability)
4. Write user stories in standard format:
   ```
   As a [role], I want [goal] so that [benefit]
   ```
5. Add acceptance criteria (Given/When/Then)
6. Define API contracts (if applicable):
   - REST: OpenAPI 3.0
   - GraphQL: SDL schema
   - gRPC: Protobuf definitions
7. Write test scenarios aligned with acceptance criteria
8. Save all artifacts and persist to IA_Recuerdo

## Examples

### Example 1: JWT Auth Spec
```json
{
  "skill": "sdd-spec",
  "params": {
    "proposal": "artifacts/proposal.md"
  }
}
```

**Output `user_stories.md`:**
```markdown
## Story 1: User Login
As an API consumer, I want to authenticate with username/password so that I receive a JWT for subsequent requests.

### Acceptance Criteria
- Given valid credentials
- When I POST to `/auth/login`
- Then I receive a JWT with 1-hour expiry
- And the token includes user ID and roles

### Acceptance Criteria (Negative)
- Given invalid credentials
- When I POST to `/auth/login`
- Then I receive 401 Unauthorized
- And no token is issued
```

**Output `api_spec.yaml`:**
```yaml
openapi: 3.0.0
paths:
  /auth/login:
    post:
      summary: Authenticate user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                username: { type: string }
                password: { type: string }
      responses:
        200:
          description: Login successful
          content:
            application/json:
              schema:
                type: object
                properties:
                  token: { type: string }
                  expires_at: { type: string, format: date-time }
        401:
          description: Invalid credentials
```

**Output `test_scenarios.md`:**
```gherkin
Feature: JWT Authentication

  Scenario: Successful login
    Given a registered user with username "alice" and password "secret123"
    When I POST to /auth/login with valid credentials
    Then the response status is 200
    And the response contains a JWT token
    And the token is valid for 1 hour

  Scenario: Invalid password
    Given a registered user with username "alice"
    When I POST to /auth/login with incorrect password
    Then the response status is 401
    And no token is issued
```

### Example 2: Blazor Component Spec
```json
{
  "skill": "sdd-spec",
  "params": {
    "proposal": "artifacts/proposal.md",
    "user_stories": "artifacts/product_stories.md"
  }
}
```

**Output `requirements.md`:**
```markdown
## Functional Requirements
- FR1: Display user profile in MudBlazor MudCard component
- FR2: Allow inline editing of name and email
- FR3: Save changes via PUT /api/users/{id}

## Non-Functional Requirements
- NFR1: Component must render in <100ms on desktop
- NFR2: Support keyboard navigation (tab, enter)
- NFR3: Accessible (ARIA labels, screen reader compatible)
```

## Gotchas & Edge Cases
- **Ambiguous requirements**: Flag unclear specs, ask for clarification
- **Conflicting acceptance criteria**: Raise inconsistencies for stakeholder resolution
- **Over-specification**: Keep specs at behavior level, not implementation
- **Missing API contracts**: Auto-generate OpenAPI from code if available

## Implementation Notes
- Use standard formats: User stories (As a.../I want.../So that...), Gherkin (Given/When/Then)
- Validate API specs with linters: Spectral (OpenAPI), buf (Protobuf)
- Link specs to tests: Use spec IDs (e.g., `REQ-001`) in test comments
- Timeout: 2 minutes for spec generation

## References
- SDD workflow: `/workspace/03-SDD-Orchestration-Patterns.md`
- BDD/Gherkin: Cucumber docs, SpecFlow (.NET)
- OpenAPI: Swagger/OpenAPI 3.0 specification
- User story format: Agile Alliance best practices
