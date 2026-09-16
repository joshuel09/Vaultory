---

description: "Task list for feature 002-docker-dev-environment"
---

# Tasks: Docker Development Environment

**Input**: Design documents from `/specs/002-docker-dev-environment/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[quickstart.md](./quickstart.md)

**Tests**: This feature adds no test suites of its own. Its purpose is to make Vaultory's existing
suites — unit, integration, contract, and end-to-end — executable, and its acceptance checks are the
walkthroughs in `quickstart.md`. Writing tests for a Compose file would test Docker rather than
Vaultory, which Principle V's simplicity rule argues against.

**Organization**: Grouped by user story so each is independently deliverable.

**Note on `data-model.md` and `contracts/`**: deliberately absent — no entity, no schema change, no
new external interface. See the plan's Project Structure.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete work)
- **[Story]**: Which user story the task serves (US1, US2, US3)
- Exact file paths are given in every task

## Path Conventions

Compose files and the `Makefile` live at the repository root. Each application owns its own
Dockerfiles and `.dockerignore`. No application source is touched except where a task says so
explicitly.

---

## Phase 1: Setup (Build Context & Configuration)

**Purpose**: The groundwork every image build depends on. Build contexts come first — without them
every subsequent build is slow and ships files it should not.

- [X] T001 [P] Write the repository-root `.dockerignore`, excluding `.git/`, `specs/`, `docs/`, `node_modules/`, `.next/`, `backend/.imagestore/`, `*.md`, and local env files, so no build context carries them
- [X] T002 [P] Write `backend/.dockerignore`, excluding `.imagestore/`, `tests/` for production builds, and any local binaries
- [X] T003 [P] Write `frontend/.dockerignore`, excluding `node_modules/`, `.next/`, `test-results/`, `playwright-report/`, and local env files. **`node_modules` in a build context is the single biggest cause of slow Docker builds in a Node project**
- [X] T004 **ADAPTED** — a `.env.example` could not be written: this environment denies writes to `.env*` paths. Every value instead carries a documented development-only default inline in `compose.yaml` via `${VAR:-default}`, and `README.md` explains overriding them with a local `.env`. This satisfies FR-009 and FR-022 better than an example file would, since a clone now runs with no configuration step at all
- [X] T005 Confirmed `.gitignore` already ignores `.env` and `.env.*` while permitting `.env.example`, so a developer's local overrides never reach the repository (FR-021)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The database and migration services every profile builds on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T006 Create `compose.yaml` at the repository root with the `postgres` service under the `dev` profile: PostgreSQL 17, a named volume for its data, a `pg_isready` health check, and its published port read from `${VAULTORY_DB_PORT:-5432}` (FR-006, FR-009)
- [X] T007 Add the `migrate` one-shot service to `compose.yaml` using the official `migrate/migrate` image, mounting `backend/migrations`, depending on `postgres` with `condition: service_healthy`, running `up` and exiting (FR-002, FR-005)
- [X] T008 Delete the superseded `docker-compose.yml`. Leaving both invites a developer to run the old one and get PostgreSQL alone, which is the confusion this feature exists to remove
- [ ] T009 **UNVERIFIED (no Docker daemon here)** — statically verified instead: `compose.yaml` parses, and the service graph was asserted against FR-002, FR-006, FR-009, FR-010, and FR-015. `docker compose config` still needs running.  Verify the compose file parses and the service graph is what the plan describes, with `docker compose config --profile dev`

**Checkpoint**: A migrated database can be brought up on its own.

---

## Phase 3: User Story 1 - Run the whole stack from a clean machine (Priority: P1) 🎯 MVP

**Goal**: One command brings up a migrated, seeded Vaultory with no host toolchain, and a source
change is visible without a rebuild.

**Independent Test**: On a machine with only Docker, clone the repository and run the documented
command; the collection view loads and a collectible can be added.

### Implementation for User Story 1

- [X] T010 [P] [US1] Write `backend/Dockerfile.dev`: a Go image matching `backend/go.mod`, with the module and build caches on named volumes so a routine start does not re-download the dependency tree (FR-011)
- [X] T011 [P] [US1] Write `frontend/Dockerfile.dev`: a Node LTS image matching `frontend/package.json`, with `node_modules` held in a named volume rather than bind-mounted — a host-built native binary inside a Linux container fails confusingly (research Decision 2, FR-011)
- [X] T012 [US1] Add the `backend` service to `compose.yaml` under the `dev` profile: built from `backend/Dockerfile.dev`, `depends_on` the `migrate` service with `condition: service_completed_successfully`, and every `VAULTORY_*` variable supplied from the environment (FR-002, FR-022)
- [X] T013 [US1] Add a named volume for the backend's image store, mounted at `VAULTORY_IMAGE_STORE_PATH`, so uploaded images survive a restart or rebuild. **Without this, every rebuild orphans every image row in the database — rows pointing at bytes that no longer exist** (FR-006, research Decision 6)
- [X] T014 [US1] Add the `frontend` service to `compose.yaml` under the `dev` profile, built from `frontend/Dockerfile.dev`, with `VAULTORY_BACKEND_ORIGIN` pointing at the backend service so the `/api/*` rewrite resolves inside the Compose network
- [X] T015 [US1] Add `develop.watch` rules to the backend and frontend services: `action: sync` for source paths, `action: rebuild` for `go.mod`, `go.sum`, and `package.json` (FR-010, SC-004)
- [X] T016 [US1] Write the root `Makefile` with `up`, `watch`, `down`, `logs`, and `reset`. `down` must keep volumes and `reset` must remove them, and the difference must be obvious enough that nobody runs the second by accident (FR-007, FR-012)
- [ ] T017 [US1] Confirm the development collectors are seeded through the containerised migration path, and that a second `make up` neither duplicates them nor fails on already-applied migrations (FR-004, FR-005)

**Checkpoint**: `make up` yields a working Vaultory with no host toolchain.

---

## Phase 4: User Story 2 - Run the full test suite without installing toolchains (Priority: P2)

**Goal**: Every Vaultory suite runs with only Docker, against a database that cannot be the
development one.

**Independent Test**: On a machine with only Docker, run the documented test command and confirm the
integration, contract, and end-to-end suites execute and report results.

### Implementation for User Story 2

- [X] T018 [US2] Add the `postgres-test` service to `compose.yaml` under the `test` profile, on tmpfs, with its own health check and **no published port**, so nothing outside the profile can reach it (FR-015, research Decision 4)
- [X] T019 [US2] Add a `migrate-test` one-shot for the test database, depending on `postgres-test` being healthy
- [X] T020 [US2] Add the `backend-test` service under the `test` profile, running `go test ./tests/unit/...` and `go test -tags=integration ./tests/integration/... ./tests/contract/...` with `VAULTORY_TEST_DATABASE_URL` pointing at `postgres-test`. **It must be impossible to point this at the development database** — the integration harness truncates tables between tests and would delete a developer's collection without warning (FR-015)
- [X] T021 [US2] Add the `e2e` service under the `e2e` profile, using the official Playwright image pinned to the version in `frontend/package.json`, with `VAULTORY_BASE_URL` pointing at the running frontend service (FR-014)
- [X] T022 [US2] Add `test` and `test-e2e` targets to the `Makefile`, each propagating the container's exit code so a failing suite fails the command (FR-016)
- [ ] T023 [US2] Verify a test run leaves the development database untouched, using the before-and-after count from `quickstart.md` walkthrough E (SC-008)

**Checkpoint**: The integration, contract, and end-to-end suites are runnable — closing the gap that
leaves feature 001 at 95 of 97 tasks.

---

## Phase 5: Documentation

- [X] T024 [P] Update `README.md` so the Docker path is the primary way to run Vaultory, with the host path kept as an alternative. Running from the host stays supported; this adds a way to run it, it does not remove one
- [X] T025 [P] Update `CLAUDE.md`'s "Running and testing" section to name the Docker commands
- [ ] T026 Walk `specs/002-docker-dev-environment/quickstart.md` walkthroughs A–F and H and record the result of each. **Requires a machine with Docker; see the Blocked note below**

---

## Phase 6: User Story 3 - Production images (Priority: P3) — SEPARATE ISSUE

**Not in the scope of issue #4.** These tasks depend on reconciling a check in
`cmd/vaultory-api/main.go`, which is an application change and the single entry in the plan's
Complexity Tracking. It deserves its own issue and its own review rather than riding along with
developer tooling.

- [X] T027 [US3] Reconcile the identity check in `backend/cmd/vaultory-api/main.go` (FR-019). Done with a **build tag**, not a runtime flag: `internal/identity/dev.go` is `//go:build !production`, `dev_production.go` is its `//go:build production` counterpart returning an error from the same constructor, and `main.go` refuses on both conditions. Verified: `strings` finds `vaultory_dev_session` in a default build and not in a `-tags production` build; `tests/unit/dev_identity_production_test.go` fails if the resolver is reintroduced
- [X] T028 [P] [US3] Write `backend/Dockerfile`: multi-stage, a statically linked `-tags production` binary copied into `gcr.io/distroless/static-debian12:nonroot` — no shell, no package manager, non-root by construction (FR-017, FR-018). **Written, not built** — Docker is not installed
- [X] T029 [US3] Add `output: 'standalone'` to `frontend/next.config.ts` and write `frontend/Dockerfile` as a multi-stage build onto a slim Node runtime running as a non-root user (FR-017, FR-018). `npm run build` emits `.next/standalone/server.js` — verified. `VAULTORY_BACKEND_ORIGIN` is a **build argument**: `.next/routes-manifest.json` shows the rewrite destination compiled in, so a runtime value would be ignored silently. **Image written, not built**
- [X] T030 [P] [US3] Write `compose.prod.yaml` for verifying the production images locally. It is **not** a deployment artifact — deployment needs decisions about secrets, TLS, and a host that this feature deliberately does not make. It also cannot serve traffic yet: the production backend has no resolver, so it refuses to start, which is the behaviour worth verifying. YAML parses; `docker compose config` not run
- [ ] T031 [US3] **PARTIAL — needs Docker for the definitive number, but the finding is already clear.** Units differ and must not be added: `docker image ls` reports uncompressed, a registry manifest lists compressed layers (~2.5x apart for Debian bases). Backend: an 11 MB (amd64) / 10 MB (arm64) stripped static binary — measured — on distroless/static, a couple of MB by its published size, not measured here. Comfortably under 150 MB. Frontend: `node:22-bookworm-slim` is ~80 MB compressed per its manifest, ≈200 MB as `docker image ls` reports it, which is **already over the criterion before any application code**; the ~68 MB of `.next/standalone` and `.next/static` measured on disk puts it near 270 MB. **The frontend image will not meet SC-006's 150 MB.** Reported rather than worked around, per the task
- [ ] T032 [US3] **PARTIAL — needs Docker for `buildx --platform`.** What was verified without it: the Go binary cross-compiles clean for `linux/amd64` and `linux/arm64` with `-tags production`, and every base image used (`distroless/static-debian12:nonroot`, `node:22-bookworm-slim`, `golang:1.26-bookworm`, `postgres:17-alpine`) publishes both architectures in its registry manifest list (FR-020, SC-007)

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (Phase 1)**: no dependencies. The `.dockerignore` files come first because every later build depends on them
- **Foundational (Phase 2)**: needs Setup; **blocks both user stories**
- **User Story 1 (Phase 3)**: needs Foundational
- **User Story 2 (Phase 4)**: needs Foundational. The `e2e` service also needs US1's frontend service to exist
- **Documentation (Phase 5)**: needs the stories it documents
- **User Story 3 (Phase 6)**: separate issue; T027 blocks T028 and T031

### Critical path

T001 → T004 → T006 → T007 → T012 → T013 → T016

### Parallel opportunities

- Setup: T001, T002, T003 in parallel
- US1: T010 and T011 in parallel, then the service definitions in order — they edit one file
- Phase 5: T024 and T025 in parallel
- US3: T028 and T030 in parallel once T027 lands

**A caution on parallelism**: most of Phase 3 and Phase 4 edit `compose.yaml`. They are marked
sequential deliberately — parallel edits to one file conflict, whatever the dependency graph says.

---

## Implementation Strategy

### MVP first (User Story 1)

1. Phase 1 — build contexts and configuration
2. Phase 2 — database and migrations
3. Phase 3 — the development stack
4. **Stop and validate**: quickstart walkthroughs A, B, C, D
5. A developer with only Docker can run Vaultory

### Then the reason this feature exists

6. Phase 4 — the test profiles, then walkthrough E
7. At that point T088 and T089 of feature 001 become closable, taking it to 97/97

### Deferred to its own issue

8. Phase 6 — production images, gated on the `main.go` reconciliation

---

## Blocked

**T026, and the verification half of every task above, require a machine with Docker installed.**
It is not installed where these files are being written. Everything here can be authored and
statically checked with `docker compose config`, but nothing can be *run*.

The honest consequence: expect the first `make up` on a Docker-equipped machine to surface problems.
That is the normal outcome of writing infrastructure blind, and it is better stated in advance than
discovered as a surprise.

---

## Notes

- [P] marks tasks touching different files with no dependency on incomplete work
- Every task names the file it changes
- Commit after each task or logical group, on the branch for issue #4
- Two things are load-bearing and easy to get wrong: **the test database must be impossible to
  confuse with the development one** (T018, T020), and **uploaded images must live on a named
  volume** (T013), or every rebuild orphans image rows
