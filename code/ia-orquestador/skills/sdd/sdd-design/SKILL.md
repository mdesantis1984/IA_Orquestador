# sdd-design — Technical Design

## Purpose
Create technical design documentation: component diagrams, interface contracts, data models, architecture decisions, and implementation guidelines.

## When to Use
- After `sdd-spec` to define "how" (not "what")
- Before `sdd-tasks` to guide implementation
- For complex features requiring architectural clarity
- When multiple developers will implement in parallel

## When NOT to Use
- For trivial changes with obvious implementation
- When design is already documented
- Emergency hotfixes (skip to sdd-tasks)

## Inputs
- `requirements.md` (from sdd-spec)
- `api_spec.yaml` (from sdd-spec)
- `sdd_context.json` (from sdd-init)

## Outputs
- `architecture.md`: Component diagrams, system boundaries, integration points
- `data_model.md`: Entity-relationship diagrams, schema definitions
- `interface_contracts.md`: Internal APIs, service boundaries, DTOs
- `adr.md`: Architecture Decision Records (why specific choices were made)
- IA_Recuerdo observation: topic_key=`sdd/design/{feature_name}`, type=`architecture`

## Workflow
1. Load specs and context
2. Identify components and modules
3. Define data model (entities, relationships, migrations)
4. Design interfaces between components
5. Document architectural decisions (ADRs)
6. Create diagrams (text-based: PlantUML, Mermaid)
7. Define implementation guidelines (naming, patterns, testing strategy)
8. Save all artifacts and persist to IA_Recuerdo

## Examples

### Example 1: JWT Auth Design
```json
{
  "skill": "sdd-design",
  "params": {
    "requirements": "artifacts/requirements.md",
    "api_spec": "artifacts/api_spec.yaml"
  }
}
```

**Output `architecture.md`:**
```markdown
## Component Diagram
\`\`\`mermaid
graph LR
  Client -->|POST /auth/login| AuthHandler
  AuthHandler --> UserRepo[(User DB)]
  AuthHandler --> JWTService
  JWTService -->|signed token| Client
\`\`\`

## Components
- **AuthHandler**: HTTP endpoint, validates credentials, calls JWTService
- **JWTService**: Signs and verifies tokens (uses HS256)
- **UserRepo**: Database access for user lookup and password verification
```

**Output `data_model.md`:**
```markdown
## User Entity
| Field         | Type      | Constraints        |
|---------------|-----------|--------------------|
| id            | UUID      | Primary Key        |
| username      | string    | Unique, Not Null   |
| password_hash | string    | Not Null (bcrypt)  |
| created_at    | timestamp | Not Null           |

## Migration
\`\`\`sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
\`\`\`
```

**Output `adr.md`:**
```markdown
# ADR-001: Use HS256 for JWT Signing

## Status: Accepted

## Context
Need to sign JWTs for stateless authentication.

## Decision
Use HS256 (HMAC with SHA-256) with a shared secret.

## Consequences
**Pros:**
- Simple, no key rotation needed for MVP
- Well-supported by libraries

**Cons:**
- Shared secret must be secured (not RSA public/private)
- All services must share the secret (not ideal for microservices)

## Alternatives Considered
- RS256 (RSA): Rejected for MVP (adds complexity)
- ES256 (ECDSA): Rejected (library support varies)
```

### Example 2: Blazor Component Design
```json
{
  "skill": "sdd-design",
  "params": {
    "requirements": "artifacts/requirements.md"
  }
}
```

**Output `interface_contracts.md`:**
```markdown
## UserProfileComponent API

### Props
| Name     | Type   | Required | Default | Description          |
|----------|--------|----------|---------|----------------------|
| UserId   | string | Yes      | -       | User ID to display   |
| ReadOnly | bool   | No       | false   | Disable editing      |

### Events
| Name       | Payload       | Description              |
|------------|---------------|--------------------------|
| OnSave     | UserDto       | Fired when user saves    |
| OnCancel   | -             | Fired when user cancels  |

### Dependencies
- `IUserService` (injected): `Task<UserDto> GetUserAsync(string id)`
- `IUserService.UpdateUserAsync(UserDto user)`: Save changes
```

## Gotchas & Edge Cases
- **Over-design**: Keep design pragmatic, avoid premature optimization
- **Missing context**: If specs are incomplete, recommend re-running sdd-spec
- **Conflicting patterns**: Flag architectural inconsistencies with existing code
- **Diagram complexity**: For large systems, create multiple focused diagrams

## Implementation Notes
- Use text-based diagrams (PlantUML, Mermaid) for version control
- ADRs follow standard format: Context, Decision, Consequences, Alternatives
- Link design to specs: Reference requirement IDs (e.g., `FR1`)
- Timeout: 3 minutes for design generation

## References
- SDD workflow: `/workspace/03-SDD-Orchestration-Patterns.md`
- ADR template: Michael Nygard's ADR format
- Diagrams: PlantUML, Mermaid.js, C4 model
- .NET design patterns: `/workspace/04-DotNet-Skills-Research.md`
