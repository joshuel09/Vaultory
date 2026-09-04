# Vaultory Constitution

## Core Principles

### I. Collector-First Experience

Vaultory MUST feel like a premium product designed specifically for collectors of figures, statues, action figures, comics, trading collectibles, and similar items.

- Collectible imagery MUST be treated as a primary part of the user experience.
- The interface MUST feel modern, cinematic, polished, and collection-focused.
- The product MUST NOT resemble a generic inventory system, admin dashboard, or default SaaS template.
- Dark mode MUST be treated as a first-class experience.
- Typography, spacing, cards, imagery, and interactions MUST follow a consistent visual language.
- Motion and visual effects SHOULD be subtle, purposeful, and performance-conscious.
- The interface MUST work well across desktop, tablet, and mobile.
- Accessibility MUST be considered part of the product rather than an optional enhancement.
- Loading, empty, success, validation, and error states MUST be intentionally designed.

The collection itself is the centerpiece of Vaultory. Technical decisions and UI patterns should support the feeling of owning and exploring a personal digital vault.

### II. Clear Architecture and Separation of Responsibilities

Vaultory MUST maintain clear boundaries between frontend presentation, backend business logic, and persistence.

- Next.js MUST be responsible primarily for presentation, rendering, navigation, and frontend interaction.
- Go MUST be the authoritative application backend.
- Business rules MUST NOT exist exclusively in the Next.js frontend.
- Authorization, authoritative validation, persistence rules, and domain behavior MUST be enforced by the Go backend.
- PostgreSQL MUST be the authoritative persistent data store.
- Frontend and backend MUST remain clearly separated applications or services.
- Domain logic MUST NOT be unnecessarily coupled to HTTP handlers, UI components, or database implementation details.
- Components, packages, functions, and modules SHOULD have focused responsibilities.
- New architectural abstractions MUST have a clear justification.

The architecture should remain understandable and maintainable as Vaultory grows rather than becoming unnecessarily complex.

### III. Contract-First and Type-Safe Development

Communication between Vaultory's frontend and backend MUST use explicit and predictable contracts.

- REST MUST be the default application API style.
- REST APIs MUST be documented through OpenAPI.
- API request and response structures MUST be explicitly defined.
- Breaking API changes MUST be intentional and documented.
- TypeScript MUST use strict type checking.
- `any` SHOULD NOT be used unless there is a documented technical reason.
- Go MUST use explicit domain, transport, and persistence types where appropriate.
- External input MUST be validated at system boundaries.
- Client-side validation MAY improve user experience but MUST NOT replace backend validation.
- The frontend MUST NOT make undocumented assumptions about backend behavior.

Explicit contracts reduce frontend/backend integration problems and allow both applications to evolve independently.

### IV. Data Integrity and Security

Collector data MUST be treated as durable, private, and trustworthy.

- PostgreSQL schema changes MUST use version-controlled and reproducible migrations.
- Appropriate foreign keys, constraints, indexes, and transactions SHOULD be used to protect data integrity.
- Collection information MUST NOT be silently lost, overwritten, or corrupted.
- A user MUST NOT be able to access another user's private collection without explicit authorization.
- Authentication and authorization MUST be enforced server-side.
- Client-provided ownership, user IDs, roles, permissions, prices, or trusted values MUST NOT be accepted without verification.
- Monetary values MUST use precise decimal representations and MUST NOT use floating-point arithmetic for financial calculations.
- Secrets, credentials, tokens, and environment-specific sensitive values MUST NOT be committed to source control.
- Sensitive internal errors MUST NOT be exposed directly to clients.
- Security-sensitive operations MUST fail safely.

Collectors should be able to trust Vaultory as the authoritative record of their collection and its history.

### V. Quality, Simplicity, and Spec-Driven Development

Vaultory development MUST favor deliberate requirements, maintainable implementation, and verifiable behavior.

- Significant features MUST begin with a written specification.
- Specifications MUST describe required behavior and expected outcomes before implementation details.
- Ambiguous requirements MUST be clarified before implementation begins.
- Implementation tasks MUST be traceable to specification requirements.
- Implementations MUST remain within the approved feature scope.
- Unrelated refactoring MUST NOT be introduced into feature work without justification.
- Important business rules MUST have automated tests.
- Tests MUST cover meaningful edge cases in addition to happy paths.
- Bug fixes SHOULD include regression tests when practical.
- Important user journeys SHOULD have integration or end-to-end coverage.
- Solutions MUST prefer simplicity and explicit behavior over unnecessary abstraction.
- Dependencies MUST NOT be introduced unless they provide clear project value.
- YAGNI principles SHOULD be followed; functionality MUST NOT be added merely because it may become useful later.

Specifications define what Vaultory should do; implementation should satisfy those specifications with the simplest maintainable solution.

## Technology & Architecture Constraints

Vaultory uses the following primary technology stack and architecture.

### Frontend

- Next.js
- TypeScript
- App Router
- Tailwind CSS
- shadcn/ui

Next.js MUST own the frontend and presentation layer.

React Server Components SHOULD be preferred when server-side rendering is appropriate.

Client Components SHOULD only be introduced when browser-side state, interaction, or APIs require them.

shadcn/ui MAY be used as a component foundation, but its default appearance MUST be customized to establish Vaultory's own visual identity.

The interface SHOULD prioritize collectible photography and visual browsing instead of dense administrative tables wherever appropriate.

### Backend

- Go

Go MUST own:

- Business logic
- Domain rules
- Server-side validation
- Authentication enforcement
- Authorization
- Persistence orchestration
- Application APIs

Backend business logic SHOULD remain independently testable from HTTP transport code.

The project SHOULD favor Go's standard library and focused libraries over large frameworks unless a framework provides a clear measurable benefit.

### API

- REST
- OpenAPI

REST MUST be the default interface between Next.js and Go unless a future specification explicitly introduces another protocol.

OpenAPI MUST describe public frontend/backend contracts.

API behavior MUST use predictable status codes and structured error responses.

### Database

- PostgreSQL

PostgreSQL MUST be the primary application database.

Relational domain data SHOULD use proper relational modeling rather than arbitrary JSON storage when relationships and constraints are meaningful.

Database design MUST support multiple copies of the same collectible belonging to a collector when required by the domain.

The architecture SHOULD be capable of distinguishing a catalog-level collectible from a user's owned instance when that distinction becomes necessary.

### Performance

- Unnecessary client-side JavaScript SHOULD be avoided.
- Collectible images MUST be optimized appropriately.
- N+1 database query patterns MUST be avoided.
- Unnecessary API requests MUST be avoided.
- Large collection views MUST support an efficient pagination or incremental-loading strategy when required.
- Performance optimization SHOULD be based on measurable needs rather than premature complexity.

## Development Workflow

Significant Vaultory features MUST follow the Spec Kit development lifecycle:

1. Specify
2. Clarify
3. Plan
4. Tasks
5. Analyze
6. Implement
7. Test
8. Review

### Specification

Feature development MUST begin by defining user-visible behavior, requirements, constraints, and expected outcomes.

Specifications SHOULD avoid unnecessary implementation details.

### Clarification

Ambiguous or incomplete requirements MUST be resolved before implementation.

Important edge cases, failure behavior, authorization rules, and data-state transitions SHOULD be identified during this stage.

### Planning

Technical implementation decisions MUST respect this constitution.

Plans MUST preserve the separation between Next.js, Go, API contracts, and PostgreSQL responsibilities.

Any deviation from the established architecture MUST include explicit justification.

### Tasks

Implementation tasks MUST map back to specification requirements.

Tasks SHOULD be small enough to implement and verify independently where practical.

### Implementation

Implementation MUST remain focused on the approved scope.

AI-assisted development MUST NOT introduce speculative functionality or unrelated architectural changes.

### Quality Gates

Before a feature is considered complete:

- Type checking MUST pass.
- Linting MUST pass.
- Required automated tests MUST pass.
- Backend tests MUST pass.
- Frontend tests MUST pass where applicable.
- Production builds MUST succeed.
- API contract changes MUST be reflected in OpenAPI.
- Database schema changes MUST include migrations.
- Important acceptance criteria from the specification MUST be verified.
- Constitution compliance MUST be reviewed.

## Governance

This constitution defines the primary engineering and product-development rules for Vaultory and supersedes informal development conventions where they conflict.

- All feature specifications, implementation plans, and significant architectural decisions MUST comply with this constitution.
- Deviations MUST be explicitly documented and justified.
- Complexity introduced beyond the current specification MUST be justified before implementation.
- Constitution amendments MUST be intentional and documented.
- Changes that alter architectural boundaries, core technology choices, security requirements, or development workflow MUST be treated as significant amendments.
- Amendments MUST update the constitution version according to semantic versioning.
- MAJOR version changes represent fundamental governance or architecture changes.
- MINOR version changes introduce or materially expand principles or constraints.
- PATCH version changes clarify existing rules without materially changing their meaning.
- Every amendment MUST update the Last Amended date.
- Specifications and plans created after an amendment MUST follow the newest ratified constitution.
- Existing implementation SHOULD be brought into compliance when reasonably practical if constitutional requirements change.

**Version**: 1.0.0 | **Ratified**: 2026-09-04 | **Last Amended**: 2026-09-04