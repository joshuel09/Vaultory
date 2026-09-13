# Vaultory

A premium collectibles collection management platform. Collectors build a personal digital vault
and track figures, statues, comics, and trading collectibles — what they own, what they have on
preorder, what they are still hunting, and what they have sold.

The collection is presented as a visual gallery rather than an inventory table. That is the point
of the product, and it is written into the project's constitution.

## Repository layout

```text
backend/    Go service — domain, validation, authorization, persistence, image handling
frontend/   Next.js App Router — presentation only
specs/      Spec Kit artifacts: specification, plan, data model, contract, tasks
.specify/   Spec Kit configuration and the project constitution
```

`backend/` and `frontend/` are separate applications and communicate only over the REST contract in
`specs/001-add-browse-collectibles/contracts/openapi.yaml`. The frontend's request and response
types are generated from that file; it is the single source of truth for the seam.

## Governance

`.specify/memory/constitution.md` defines the engineering rules this project works to, and
supersedes informal convention where they conflict. The ones that shape the code most:

- Go owns business logic, validation, authorization, and persistence. Next.js renders.
- REST, described by OpenAPI. Breaking changes are deliberate and documented.
- Monetary values are exact decimals, never floating point.
- A collector's vault is private, enforced server-side.
- Features begin with a written specification, and tasks trace back to its requirements.

## Running it locally

Full instructions, including every walkthrough that demonstrates the feature, are in
[specs/001-add-browse-collectibles/quickstart.md](specs/001-add-browse-collectibles/quickstart.md).
The short version:

```bash
# 1. PostgreSQL
docker compose up -d postgres

# 2. Schema
cp backend/.env.example backend/.env          # then edit if your setup differs
export $(grep -v '^#' backend/.env | xargs)
migrate -path backend/migrations -database "$VAULTORY_DATABASE_URL" up

# 3. Backend
cd backend && go run ./cmd/vaultory-api

# 4. Frontend, in a second shell
cd frontend && npm install && npm run dev
```

Then open <http://localhost:3000>.

### Becoming a collector

Authentication is not implemented yet — it is explicitly out of scope for the first feature. In its
place, a development-only sign-in mints a session for one of two seeded collectors:

```bash
curl -c /tmp/vaultory.jar -X POST http://localhost:3000/api/dev/session
curl -c /tmp/other.jar    -X POST 'http://localhost:3000/api/dev/session?collector=second'
```

The second collector exists so that cross-collector privacy can actually be exercised. This
endpoint requires `VAULTORY_DEV_IDENTITY=enabled` and **must never be enabled outside local
development**: it issues a session to anyone who asks.

## Quality gates

The constitution requires all of these to pass before a feature is complete:

```bash
cd backend  && go vet ./... && go build ./... && go test ./tests/unit/... ./internal/...
cd frontend && npm run lint && npm run typecheck && npm run build && npm run test
```

Tests that need a real database are behind a build tag, because the guarantees they check — the
composite foreign key, submission-key uniqueness under concurrency, `numeric(12,2)` exactness —
live in the schema, and a fake would only assert that the Go code believes in them:

```bash
export VAULTORY_TEST_DATABASE_URL="$VAULTORY_DATABASE_URL"
cd backend && go test -tags=integration ./tests/integration/... ./tests/contract/...
```

End-to-end journeys need the whole stack running:

```bash
cd frontend && npm run test:e2e
```

## Working on it

This project uses [Spec Kit](https://github.com/github/spec-kit). Work starts from a specification
rather than from code:

`/speckit-specify` → `/speckit-clarify` → `/speckit-plan` → `/speckit-tasks` → `/speckit-analyze` →
`/speckit-implement`

Current feature: [001-add-browse-collectibles](specs/001-add-browse-collectibles/spec.md) — adding
collectibles to a private vault and browsing them as a gallery.
