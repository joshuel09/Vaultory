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

### With Docker — the short way

Docker is the only prerequisite. No Go, no Node, no PostgreSQL, no migration tool.

```bash
make up
```

That starts the database, applies every migration, waits for them to finish, then starts the
backend and the frontend. Open <http://localhost:3000>.

```bash
make            # list every command
make watch      # start, and sync your edits into the running stack
make logs       # follow the logs
make down       # stop; your data is kept
make reset      # DESTROY all data, then stop
```

To change a port or any other value, create a `.env` beside `compose.yaml`. Compose reads it
automatically and it is gitignored, so your local settings never appear in a diff:

```bash
echo 'VAULTORY_DB_PORT=5433' >> .env    # if you already run PostgreSQL on 5432
```

Every variable has a development-only default documented inline in `compose.yaml`; there is nothing
you must set to get started.

### Without Docker — running from the host

Still supported. You need Go, Node, PostgreSQL, and `golang-migrate` installed:

```bash
createdb vaultory_dev
export VAULTORY_DATABASE_URL="postgres://$(whoami)@localhost:5432/vaultory_dev?sslmode=disable"
migrate -path backend/migrations -database "$VAULTORY_DATABASE_URL" up

# Shell 1
cd backend
export VAULTORY_SESSION_SECRET=dev-only-not-a-real-secret-change-me
go run ./cmd/vaultory-api

# Shell 2
cd frontend && npm install && npm run dev
```

Full instructions, including every walkthrough that demonstrates the feature, are in
[specs/001-add-browse-collectibles/quickstart.md](specs/001-add-browse-collectibles/quickstart.md).

### Configuration this needs

| Variable | Used by | Notes |
|---|---|---|
| `VAULTORY_SESSION_SECRET` | backend, frontend | The same value feeds both. **At least 32 characters** — the service refuses to start below that, because this is the only input to the signature it verifies. Required explicitly in production |
| `VAULTORY_AUTH_DB_PASSWORD` | frontend, provisioning | Password for the restricted `vaultory_auth` role. Required explicitly in production |
| `VAULTORY_PUBLIC_URL` | frontend | The origin collectors reach Vaultory on. Production only |
| `MAIL_HOST` / `MAIL_PORT` / `MAIL_FROM` | frontend | Where recovery mail is sent. Development points at Mailpit; production needs a real service |
| `MAIL_USER` / `MAIL_PASSWORD` | frontend | SMTP credentials. Production only, and required explicitly |

The frontend holds a database connection of its own, because the authentication library owns the
account and session tables. It connects as `vaultory_auth`, a role with privileges on those tables
and **no access to any collection table** — so a bug in the frontend cannot read a collection, and
that is enforced by PostgreSQL rather than by convention.

That role is created by a migration **without a password**, and given one at startup from
`VAULTORY_AUTH_DB_PASSWORD`. A credential inside a tracked migration is exactly what the
constitution forbids, and this repository is public. A `LOGIN` role with no password cannot
authenticate, so a missed provisioning step stops the frontend connecting rather than leaving an
open account.

### Reading mail in development

Vaultory sends verification and password-reset messages. Locally they go to **Mailpit**, which
starts with the rest of the stack:

```
http://localhost:8025
```

Mailpit holds messages in memory and has no outbound path, so nothing you send while developing can
reach a real inbox — not by accident and not by misconfiguration. No third-party account is needed.

### Becoming a collector

Register an account. That is all — there is no development back door any more.

Open <http://localhost:3000>, follow the call to action, and give an email address and a password
of at least 12 characters. You land in your own empty vault, and nothing in it is visible to anyone
else.

For cross-collector checks, register a second account in a private window. Two collectors are what
make the privacy rules exercisable: another collector's collectible answers **404, never 403**,
because a 403 would confirm it exists.

### Production images

`backend/Dockerfile` and `frontend/Dockerfile` build the shipped images; `compose.prod.yaml` exists
to verify them locally and is **not** a deployment artifact.

```bash
docker compose -f compose.prod.yaml build
```

The backend image is distroless — no shell, no package manager, non-root. It contains no mechanism
that issues a session without verifying a credential, because no such mechanism exists in any build
any more: the development resolver was deleted in feature 004, and the `production` build tag that
used to exclude it went with it. A binary that cannot verify a session refuses to start.

## Testing

With Docker, everything is two commands:

```bash
make test        # backend: unit, integration, and contract suites
make up && make test-e2e    # the browser suite against a running stack
```

`make test` runs against a throwaway database on tmpfs, never your development one. That is not a
convention to remember — the test services have no route to the development database at all. The
integration harness deletes every collectible, image, and submission before each test, so pointing
it at your own data would destroy your collection without warning. Verified rather than assumed:
a development database holding 658 collectibles was counted before and after a full `make test`
run and was unchanged.

Tests that need a real database sit behind a build tag, because the guarantees they check — the
composite foreign key, submission-key uniqueness under concurrency, `numeric(12,2)` exactness —
live in the schema, and a fake would only assert that the Go code believes in them.

### Without Docker

```bash
cd backend  && go vet ./... && go build ./... && go test ./tests/unit/... ./internal/...
cd frontend && npm run lint && npm run typecheck && npm run build && npm run test

# The build-tagged suites are invisible to a plain `go vet`, so a signature change under
# internal/ can break them without anything here failing. This type-checks them with no database:
cd backend  && go vet -tags integration ./...   # type-check the integration and contract suites

export VAULTORY_TEST_DATABASE_URL="$VAULTORY_DATABASE_URL"
# -p 1 matters: both packages share one database, and `go test` runs packages in parallel.
cd backend && go test -tags=integration -p 1 ./tests/integration/... ./tests/contract/...
cd frontend && npm run test:e2e
```

The constitution requires all of these to pass before a feature is complete.

## Working on it

This project uses [Spec Kit](https://github.com/github/spec-kit). Work starts from a specification
rather than from code:

`/speckit-specify` → `/speckit-clarify` → `/speckit-plan` → `/speckit-tasks` → `/speckit-analyze` →
`/speckit-implement`

Current feature: [001-add-browse-collectibles](specs/001-add-browse-collectibles/spec.md) — adding
collectibles to a private vault and browsing them as a gallery.
