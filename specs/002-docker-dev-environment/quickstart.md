# Quickstart & Validation: Docker Development Environment

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Date**: 2026-09-14

How to run and validate this feature once it is built. Every walkthrough maps to numbered acceptance
scenarios in `spec.md`, so a reviewer can work down the list and see each requirement hold.

This is the developer-facing interface for feature 002 — the commands themselves. There is no
`contracts/` directory for this feature and no new data model; see the plan's Project Structure for
why.

---

## Prerequisites

- Docker Engine with **Compose v2.22 or newer** — `docker compose watch` requires it
  (research Decision 2). Check with `docker compose version`.
- Nothing else. No Go, no Node, no PostgreSQL, no `golang-migrate`. That is the point (FR-003).

```bash
cp .env.example .env      # ports and development-only values; edit only if something collides
```

---

## Walkthrough A — Start the stack from a clean clone

Covers User Story 1 scenario 1, FR-001, FR-002, FR-003, FR-004.

```bash
make up          # or: docker compose --profile dev up --build
```

Expect, in order: PostgreSQL starts and passes its health check; the `migrate` service applies all
migrations and exits successfully; the backend starts only after that; the frontend starts.

Then open <http://localhost:3000> and create an account. Feature 004 removed the seeded
development collectors and the sign-in that minted sessions for them; registration is the only way
into a vault now.

```bash
curl -s -c /tmp/vaultory.jar -X POST http://localhost:3000/api/auth/sign-up/email \
  -H 'Content-Type: application/json' -H 'Origin: http://localhost:3000' \
  -d '{"email":"you@example.test","password":"a-long-enough-password","name":"you"}'
```

**The core proof**: no toolchain was installed on the host, and Vaultory is running.

To confirm migrations really did run before the backend served anything, check the ordering in the
logs — the backend's "listening" line must come after the migration service exits.

## Walkthrough B — Add a collectible, exactly as on the host

Covers User Story 1 scenario 2.

Through the browser, add a collectible with a photograph and confirm it appears in the gallery with
its image. Then confirm the same through the API:

```bash
curl -s -b /tmp/vaultory.jar -X POST http://localhost:3000/api/collectibles \
  -H 'Content-Type: application/json' \
  -d '{"submissionKey":"'"$(uuidgen)"'","name":"Docker Sentinel","collectionStatus":"owned"}' | jq
```

Expect `201`. Everything in feature 001's
[quickstart](../001-add-browse-collectibles/quickstart.md) — Walkthroughs A through I — must behave
identically here. That is the real acceptance test for this feature: **nothing about Vaultory
changes, only where it runs.**

## Walkthrough C — Data survives a restart

Covers User Story 1 scenario 3, FR-006, SC-005.

```bash
make down        # stops and removes containers; volumes are kept
make up
```

Expect every collectible from Walkthrough B still present, with its image still loading. Losing
uploaded images here would leave image rows pointing at bytes that no longer exist — worse than
losing both (research Decision 6).

## Walkthrough D — A source change without a rebuild

Covers User Story 1 scenario 4, FR-010, SC-004.

```bash
make watch       # or: docker compose --profile dev watch
```

With that running, edit a heading in `frontend/app/collection/page.tsx` and save. Expect the browser
to reflect it within about ten seconds, with no image rebuild.

Then edit a log line in `backend/cmd/vaultory-api/main.go` and save. Expect the backend to restart
and the change to appear in the logs.

Finally, edit `backend/go.mod` or `frontend/package.json`. Expect a rebuild this time — dependency
manifests are the case where a rebuild is correct.

## Walkthrough E — Run the suites that have never run

Covers User Story 2, FR-013, FR-015, FR-016, SC-003, SC-008. **This is the walkthrough that closes
the gap left by feature 001.**

First, record the state of the development database so the claim in SC-008 is checked rather than
assumed:

```bash
BEFORE=$(docker compose exec -T postgres psql -U vaultory -d vaultory_dev -tAc \
  'SELECT count(*) FROM collectibles')

make test        # or: docker compose --profile test run --rm backend-test

AFTER=$(docker compose exec -T postgres psql -U vaultory -d vaultory_dev -tAc \
  'SELECT count(*) FROM collectibles')
[ "$BEFORE" = "$AFTER" ] && echo "PASS: development data untouched" || echo "FAIL: tests touched development data"
```

Expect the backend unit, integration, and contract suites to run against the throwaway test database
and report results — and the development database to be unchanged.

Confirm the exit code is usable in automation (FR-016):

```bash
make test; echo "exit: $?"
```

## Walkthrough F — The browser suite

Covers User Story 2 scenario 2, FR-014.

```bash
make up          # the e2e suite runs against a running stack
make test-e2e    # or: docker compose --profile e2e run --rm e2e
```

Expect Playwright to run feature 001's 138 tests across three viewports and report results.

## Walkthrough G — Production images

Covers User Story 3, FR-017, FR-018, FR-019, FR-020, SC-006.

```bash
make prod-build
```

`make prod-build` rather than a bare `docker compose build`: `compose.prod.yaml` marks
`POSTGRES_PASSWORD` and `VAULTORY_SESSION_SECRET` required with `${VAR:?}`, and Compose
interpolates the whole file even for a build, so a bare build fails on variables the build does not
use. The target supplies throwaway values; `up` still demands real ones.

The images are named for the Compose project, which is `vaultory-prod`:

```bash
docker image ls --format '{{.Repository}}: {{.Size}}' | grep '^vaultory-prod'
```

Measured on 2026-09-17, arm64, Docker 29.8.0:

| Image | Size | SC-006 (150 MB) |
|---|---|---|
| `vaultory-prod-backend` | **20.5 MB** | passes, with 7x of headroom |
| `vaultory-prod-frontend` | **458.6 MB** | **fails, by 308 MB** |

The frontend layer breakdown says why, and says that nothing in this feature could have fixed it:

| Layer | Size |
|---|---|
| Debian bookworm base | 108 MB |
| Node 22 runtime | 151 MB |
| npm and friends | 7.3 MB |
| `.next/standalone` | 85.3 MB |
| `.next/static` + `public/` | 1.0 MB |

The application's own contribution is ~86 MB; the runtime underneath it is ~266 MB. Even an empty
Next app on this base misses the criterion, so trimming the bundle cannot get there. The real
options are a substantially smaller runtime base or a different criterion, and both are decisions
for a later feature rather than something to reach for here.

An earlier estimate in this file put the figure near 270 MB. That was too low — it was built from
compressed registry layer sizes and on-disk `du` output, which are not what `docker image ls`
reports. The measurement above supersedes it.

Then confirm the images contain no toolchain and do not run as root:

```bash
docker run --rm --entrypoint sh vaultory-prod-backend -c 'echo reachable'
# Expect a failure: distroless has no shell at all (FR-017)

docker run --rm --entrypoint id vaultory-prod-frontend -u
# Expect 1000, not 0 (FR-018)
```

And the security check in FR-019:

Both checks below point at an address nothing listens on. That is deliberate: it proves the
identity check runs *before* the database connection, so the refusal is what you see rather than a
connection error. `main.go` orders it that way for exactly this reason.

```bash
BAD=postgres://nope:nope@10.255.255.1:5432/nope

docker run --rm -e VAULTORY_DATABASE_URL=$BAD -e VAULTORY_SESSION_SECRET=x \
  vaultory-prod-backend; echo "exit: $?"
# Expect: "no collector resolver is configured: set VAULTORY_DEV_IDENTITY=enabled for local
#          development, or supply a real authentication resolver before deploying", exit 1

docker run --rm -e VAULTORY_DEV_IDENTITY=enabled -e VAULTORY_DATABASE_URL=$BAD \
  -e VAULTORY_SESSION_SECRET=x vaultory-prod-backend; echo "exit: $?"
# Expect: "development identity is not available in a production build: unset
#          VAULTORY_DEV_IDENTITY, or build without -tags production", exit 1
```

Both observed on 2026-09-17. The strongest form of the check does not need a container at all —
extract the shipped binary and look for the development session cookie name:

```bash
cid=$(docker create vaultory-prod-backend); docker cp $cid:/usr/local/bin/vaultory-api /tmp/b
docker rm -f $cid
strings /tmp/b | grep -c vaultory_dev_session    # 0, on both amd64 and arm64
```

Two refusals, and both are correct. The production image is built with `-tags production`, so the
development resolver is not in the binary at all — the first command cannot enable what is not
there. The second is feature 001's original guarantee, unchanged.

This also means **the production stack cannot serve traffic yet**, and that is the honest state of
the project rather than a gap in these files: authentication is out of scope for feature 001, so
the only resolver that exists is the development one. `compose.prod.yaml` says so at the top.

The same property can be checked without Docker, which is how it was verified here:

```bash
cd backend
go test -tags production -run TestDevResolverIsUnavailable ./tests/unit/... -v

go build -tags production -o /tmp/prod ./cmd/vaultory-api && strings /tmp/prod | grep -c vaultory_dev_session   # 0
go build              -o /tmp/dev  ./cmd/vaultory-api && strings /tmp/dev  | grep -c vaultory_dev_session   # 1
```

## Walkthrough H — Failure modes

Covers the spec's edge cases.

```bash
# Port conflict (FR-008, FR-009)
VAULTORY_DB_PORT=5432 make up     # with something already on 5432
# Expect a failure naming the port. Then resolve it without editing a tracked file:
echo 'VAULTORY_DB_PORT=5433' >> .env && make up

# Starting twice (FR-005)
make up && make up
# Expect no duplicated collectors, no error on already-applied migrations

# Deliberate data removal (FR-007) — destroys everything, on purpose
make reset        # or: docker compose --profile dev down -v
```

`make down` keeps data; `make reset` discards it. The difference must be obvious enough that nobody
runs the second by accident.

---

## Acceptance summary

| Requirement | Walkthrough |
|---|---|
| FR-001 to FR-004 — one command, migrated, seeded, no toolchain | A |
| FR-005 — starting twice is safe | H |
| FR-006, FR-007 — data persists; deliberate removal exists | C, H |
| FR-008, FR-009 — port conflicts named and overridable | H |
| FR-010, FR-011, FR-012 — live reload, caches, logs | D |
| FR-013 to FR-016 — suites runnable, isolated, exit codes | E, F |
| FR-017 to FR-020 — production images | G |
| FR-021, FR-022 — no committed secrets, configurable | A, G |
| Nothing about Vaultory changes | B |

## Troubleshooting

| Symptom | Likely cause |
|---------|--------------|
| `docker compose watch` is not a command | Compose is older than v2.22 (research Decision 2) |
| The backend starts before migrations finish | `depends_on` is missing `condition: service_completed_successfully` |
| A source change does nothing | The path is not covered by a `develop.watch` rule, or `make watch` is not running |
| Uploaded images vanish after a rebuild | The image store is on container storage rather than a named volume (research Decision 6) |
| A test run emptied the development collection | The test runner is pointed at the development database — the one thing Decision 4 exists to prevent |
| Frontend start is slow every time | `node_modules` is not on a persisted volume (research Decision 8) |
| The production backend image will not start | Expected until the FR-019 check in `main.go` is reconciled; see the plan's Complexity Tracking |
