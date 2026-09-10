# Implementation Plan: Add & Browse Collectibles

**Branch**: `001-add-browse-collectibles` | **Date**: 2026-09-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-add-browse-collectibles/spec.md`

## Summary

Collectors add collectibles to a private vault and browse them as an image-forward gallery filtered
by collection status. A collectible requires only a name and one of four statuses; eleven further
attributes are optional, including one uploaded primary image (JPEG/PNG/WebP, 10 MB maximum) that
Vaultory stores and serves only to the owning collector.

The approach follows the constitution's separation: a Go service owns the domain, validation,
authorization, persistence, and image handling; PostgreSQL is the authoritative store; a REST
contract described in OpenAPI is the only seam between the two applications; a Next.js App Router
frontend renders the gallery and the add form and holds no business rules. Next.js proxies
`/api/*` to Go so the collector's session travels same-origin, which is what makes per-request
image authorization workable in a browser. Go derives a fixed-aspect gallery image at upload time,
so the gallery is visually consistent without collectors cropping anything and without shipping
10 MB originals into a grid.

Authentication is out of scope, so identity resolution is isolated behind a single seam with a
development-only implementation, allowing real authentication to replace it without touching
domain or transport code.

## Technical Context

**Language/Version**: Go (backend, pin 1.25 or newer at implementation); TypeScript 5.x with
Node.js LTS (frontend, pin at implementation)

**Primary Dependencies**: Backend — Go standard library `net/http` for transport, `pgx` for
PostgreSQL access, `golang-migrate` for schema migrations, `golang.org/x/image` for WebP decoding.
Frontend — Next.js (App Router), Tailwind CSS, shadcn/ui, `openapi-typescript` to generate request
and response types from the OpenAPI contract.

**Storage**: PostgreSQL (pin 16 or newer) for all collector and collectible data. Uploaded image
bytes and their derived gallery renditions are stored in an object store accessed through a single
Go interface, with a local filesystem implementation for development.

**Testing**: Backend — Go `testing` with `net/http/httptest` for transport, and persistence and
authorization tests against a real PostgreSQL instance. Frontend — Vitest with Testing Library for
components and states, Playwright for the add-then-browse journey the constitution asks to cover
end to end.

**Target Platform**: Linux server for the Go service and PostgreSQL; evergreen desktop, tablet, and
mobile browsers for the frontend.

**Project Type**: Web application — two clearly separated applications (Next.js frontend, Go
backend) communicating over REST.

**Performance Goals**: A 500-collectible collection becomes browsable within 2 seconds (SC-004) and
scrolls without visible stalling. A gallery page costs two indexed statements — the page and the
total count — with no per-entry queries.
Gallery renditions are small enough that a full page of entries costs far less than one original
upload.

**Constraints**: Gallery renditions fixed at 4:5 portrait, 800×1000. Uploads capped at 10 MB and
rejected by decoding rather than by trusting a declared type. Add-collectible requests idempotent
per submission key. Monetary amounts held as exact decimals, never floating point. Every collectible and every
stored image authorized against the acting collector on every request. No horizontal page scrolling
at any supported width. Adding and browsing fully keyboard operable.

**Scale/Scope**: Single-collector vaults, no sharing. Design target of tens of thousands of
collectibles per collector, browsed in pages of 24 to 48. This feature covers two screens — the
collection gallery and the add-collectible form — and four backend operations.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Evaluated against Vaultory Constitution v1.0.0.

**Pre-Phase 0**

| Gate | Source | Status | Basis |
|------|--------|--------|-------|
| Collectible imagery is primary to the experience | I | PASS | Gallery is image-forward per FR-030; consistent framing per FR-014; designed placeholder per FR-033 |
| Not an inventory or admin listing | I | PASS | FR-031 forbids a table as the primary presentation |
| Dark mode first-class; consistent visual language | I | PASS | FR-046 requires both appearances with dark as the default; SC-015 verifies every state in both; T091 and T092 build and test it. `/speckit-analyze` found this gate previously asserted without any spec requirement behind it — the spec was amended 2026-09-10 |
| Works across desktop, tablet, mobile | I | PASS | FR-036, SC-010 |
| Accessibility treated as part of the product | I | PASS | FR-044 (not color alone), FR-045 (keyboard, text alternatives), SC-011 |
| All interface states intentionally designed | I | PASS | FR-041 empty, FR-042 loading, FR-043 error, FR-040 no-results, FR-021 success, FR-020 validation |
| Next.js limited to presentation | II | PASS | Next.js renders and proxies only; no validation or authorization decisions, see research decision 2 |
| Go is the authoritative backend | II | PASS | Domain, validation, authorization, persistence, image derivation all in Go |
| No business rules exclusive to the frontend | II | PASS | Every rule in FR-002 through FR-024 enforced server-side; client validation is duplicate courtesy only |
| PostgreSQL authoritative | II | PASS | All collector and collectible data in PostgreSQL |
| Frontend and backend separated | II | PASS | Two applications, REST between them |
| Domain logic not coupled to transport or storage | II | PASS | Separate `domain`, `transport`, `store`, `imagestore` packages |
| New abstractions justified | II | PASS | Only two interfaces introduced — identity resolution and image storage — each justified in Complexity Tracking or research |
| REST is the default API style | III | PASS | Four REST operations, no alternative protocol |
| REST documented through OpenAPI | III | PASS | `contracts/openapi.yaml` is the source of truth |
| Request and response structures explicit | III | PASS | Every schema named and typed in the contract |
| TypeScript strict; `any` avoided | III | PASS | Strict mode; frontend types generated from the contract, so no hand-written response shapes |
| External input validated at boundaries | III | PASS | FR-020 validates before saving; images validated by decode, not by declared type |
| Client validation does not replace backend validation | III | PASS | Stated explicitly in research decision 3 |
| Migrations version-controlled and reproducible | IV | PASS | `golang-migrate` files committed, forward and reverse |
| Constraints, keys, indexes, transactions protect integrity | IV | PASS | See `data-model.md`; notably a composite foreign key makes the database, not just the domain, refuse a collectible that references another collector's image |
| No cross-collector access | IV | PASS | FR-026, FR-027; ownership predicate in every query, verified by test |
| Authorization enforced server-side | IV | PASS | FR-028; identity resolved only in Go |
| Client-supplied ownership never trusted | IV | PASS | No collector identifier accepted in any request body or header, see Complexity Tracking |
| Money exact, never floating point | IV | PASS | `NUMERIC(12,2)` in PostgreSQL, exact decimal in Go, string in the contract |
| Secrets not committed | IV | PASS | Connection strings and storage credentials from environment only |
| Internal errors not exposed | IV | PASS | Structured error responses carry codes and field paths, never internal detail |
| Security-sensitive operations fail safely | IV | PASS | Unresolvable identity yields 401 and no content (FR-029); another collector's resource yields 404 |
| Feature began from a written specification | V | PASS | `spec.md`, clarified 2026-09-06 |
| Ambiguity resolved before implementation | V | PASS | Zero open clarification markers; checklist 16/16 |
| Important business rules have automated tests | V | PASS | Testing strategy in `quickstart.md` maps tests to requirements |
| Simplicity, no speculative functionality | V | PASS | No catalog, no editing, no valuation; see research decision 6 |
| Dependencies justified | V | PASS | Each dependency justified in research decision 8 |
| Go standard library favoured over frameworks | Tech constraints | PASS | `net/http` only; no web framework |
| RSC preferred, Client Components only where needed | Tech constraints | PASS | Gallery is a Server Component; the add form and the filter control are Client Components |
| shadcn/ui defaults customized | Tech constraints | PASS | Themed to Vaultory's identity, see Structure Decision |
| Images optimized | Tech constraints | PASS | Fixed-aspect gallery renditions derived at upload |
| N+1 query patterns avoided | Tech constraints | PASS | Two indexed statements per gallery page — the page itself and the `totalUnfiltered` count — with no per-entry work; the image rendition arrives via a join in the same statement |
| Large collections paginate or load incrementally | Tech constraints | PASS | Keyset pagination, FR-035 |

Result: **PASS**, with one documented deviation — no real authentication — recorded in Complexity
Tracking.

**Post-Phase 1 re-check**: Re-evaluated after `research.md`, `data-model.md`, `contracts/openapi.yaml`,
and `quickstart.md` were written. All gates above still hold. The design added no new abstraction
beyond the two justified interfaces, introduced no business rule into the frontend, and left the
identity deviation as the only entry in Complexity Tracking. Result: **PASS**.

## Project Structure

### Documentation (this feature)

```text
specs/001-add-browse-collectibles/
├── plan.md              # This file (/speckit-plan command output)
├── research.md           # Phase 0 output (/speckit-plan command)
├── data-model.md          # Phase 1 output (/speckit-plan command)
├── quickstart.md          # Phase 1 output (/speckit-plan command)
├── contracts/             # Phase 1 output (/speckit-plan command)
│   └── openapi.yaml
├── checklists/
│   └── requirements.md    # Spec quality checklist (/speckit-specify command)
└── tasks.md               # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
backend/
├── cmd/
│   └── vaultory-api/            # Process entry point: config, wiring, http.Server
├── internal/
│   ├── domain/
│   │   └── collectible/         # Collectible, CollectionStatus, Money; validation rules; no I/O
│   ├── collection/              # Application service: add and list use cases
│   ├── transport/
│   │   └── httpapi/             # Handlers, request decoding, error envelope, routing
│   ├── store/
│   │   └── postgres/            # pgx queries; every query scoped by collector
│   ├── imagestore/              # Image storage interface + filesystem and object-store impls
│   ├── imaging/                 # Decode, validate, EXIF orientation, gallery rendition
│   └── identity/                # Acting-collector resolution seam (dev implementation for now)
├── migrations/                  # golang-migrate .up.sql / .down.sql pairs
└── tests/
    ├── contract/                # Responses conform to contracts/openapi.yaml
    ├── integration/             # Against real PostgreSQL: ownership, pagination, uploads
    └── unit/                    # Domain validation, money, imaging

frontend/
├── app/
│   ├── layout.tsx
│   └── collection/
│       ├── page.tsx             # Gallery (Server Component); loading and error boundaries
│       └── new/page.tsx         # Add-collectible route
├── components/
│   ├── collection/              # CollectibleCard, CollectionGallery, StatusFilter, states
│   └── ui/                      # shadcn/ui primitives, themed to Vaultory
├── lib/
│   ├── api/                     # Typed fetch wrappers over the REST contract
│   └── types/                   # Types generated from contracts/openapi.yaml
├── next.config.ts               # /api/* rewrite to the Go service
└── tests/
    ├── unit/                    # Vitest + Testing Library: cards, filter, all states
    └── e2e/                     # Playwright: add a collectible, then browse and filter
```

**Structure Decision**: Two separated applications, `backend/` and `frontend/`, matching the
constitution's requirement that Next.js and Go remain distinct services. Inside `backend/`, domain
rules live in `internal/domain` with no knowledge of HTTP or SQL, so business logic is testable
without a transport, as the technology constraints require; `transport/httpapi`, `store/postgres`,
`imagestore`, and `imaging` are the only packages permitted to touch the outside world.
`internal/identity` exists solely so the out-of-scope authentication decision has exactly one
future replacement point. In `frontend/`, `components/ui` holds shadcn/ui primitives restyled to
Vaultory's own visual identity rather than their defaults, and `lib/types` is generated from the
OpenAPI contract so no response shape is hand-maintained.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| An `internal/identity` seam with a development-only collector resolution, rather than real authentication | The spec requires collector-private data (FR-025 to FR-029) while explicitly excluding authentication implementation. Something must resolve the acting collector for ownership scoping to exist and be testable | Accepting a collector identifier from the client — a header, query parameter, or body field — directly contradicts FR-028 and Principle IV's rule against trusting client-provided ownership values. Confining resolution to one server-side seam keeps the privacy requirements genuinely enforced and testable now, and makes real authentication a single-package replacement rather than a rewrite |
