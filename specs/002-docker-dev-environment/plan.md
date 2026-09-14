# Implementation Plan: Docker Development Environment

**Branch**: `002-docker-dev-environment` | **Date**: 2026-09-14 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-docker-dev-environment/spec.md`

## Summary

Make a clone of Vaultory runnable, and fully testable, with Docker as the only prerequisite.

Today `docker-compose.yml` provides PostgreSQL alone; both applications must be run from the host
with Go, Node, and `golang-migrate` installed, and a connection string reconciled by hand. The
result is that the integration, contract, and end-to-end suites — roughly forty per cent of
Vaultory's tests — have never been executed anywhere, which is why feature 001 stands at 95 of 97
tasks.

The approach is one Compose project with profiles: `dev` brings up PostgreSQL, a migration
one-shot, the Go service, and the Next.js dev server, with source synced into the containers so a
change is visible without a rebuild; `test` brings up a separate throwaway database and runs the
backend suites against it; `e2e` runs the browser suite against the dev stack. Production image
definitions are multi-stage and ship a compiled artefact with no toolchain.

**One finding, raised rather than worked around**: the spec assumes no application source changes,
and that assumption does not survive contact with FR-019. `cmd/vaultory-api/main.go` currently
*requires* `VAULTORY_DEV_IDENTITY=enabled` and refuses to start without it — deliberately, so that
feature 001 could never silently resolve every request to one collector. A production image
therefore cannot start at all today. Delivering FR-019 means inverting that check behind a build or
configuration signal. See Complexity Tracking.

## Technical Context

**Language/Version**: No new languages. Containers use Go (matching `backend/go.mod`), Node LTS
(matching `frontend/package.json`), PostgreSQL 17, and the `migrate/migrate` image.

**Primary Dependencies**: Docker Engine with Compose v2.22 or newer — `docker compose watch`
requires 2.22, and it is what removes the need for a file-watcher dependency inside the images.

**Storage**: Named volumes for the database, for uploaded images, and for dependency caches. The
test database uses tmpfs, since a throwaway database that never touches a disk is both faster and
incapable of polluting development data.

**Testing**: The suites are unchanged. This feature changes only where they run. `go test` and
`go test -tags=integration` execute in a Go container against a test database; Playwright executes
in its official image against the dev stack.

**Target Platform**: Linux containers on arm64 and x86-64 hosts. Every base image chosen is
published for both.

**Project Type**: Developer tooling for an existing two-application web project. No application
behaviour changes.

**Performance Goals**: First run under 10 minutes on a normal connection; subsequent starts under
60 seconds; a source change visible within 10 seconds (SC-002, SC-004).

**Constraints**: No host toolchain. Development data survives restarts. Test runs cannot touch
development data. Production images under 150 MB, non-root, no compiler.

**Scale/Scope**: One developer machine. Four services in the dev profile, two more across the test
and e2e profiles, and two production image definitions.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Evaluated against Vaultory Constitution v1.0.0. Most principles govern product code and are
untouched here; the gates that genuinely apply are listed, and the rest are recorded as not
applicable rather than silently claimed as passing.

| Gate | Source | Status | Basis |
|------|--------|--------|-------|
| Collector-first experience | I | N/A | No collector-facing change. The gallery, the form, and both appearances are untouched |
| Next.js owns presentation; Go is authoritative | II | PASS | Containers run the same two applications, unchanged, with the same boundary between them |
| Frontend and backend remain separate services | II | PASS | Separate images, separate containers, communicating over the same REST contract |
| No new architectural abstraction without justification | II | PASS | Compose profiles and multi-stage builds are configuration, not architecture. No indirection is added to either application |
| Contract-first; OpenAPI describes the seam | III | PASS | The contract is untouched |
| TypeScript strict; no `any` | III | PASS | No TypeScript is added beyond, possibly, one build-output setting |
| Migrations version-controlled and reproducible | IV | PASS | The same `backend/migrations` files, applied by a dedicated one-shot service rather than by hand — strictly more reproducible than today |
| Secrets not committed | IV | PASS | Only plainly development-only values, documented as such (FR-021). Production images take configuration from the environment |
| Security-sensitive operations fail safely | IV | **AT RISK** | FR-019 requires a production image to refuse development identity. The current check is inverted — see Complexity Tracking. The *intent* is preserved either way: the server must never quietly resolve every request to one collector |
| Simplicity; no speculative functionality | V | PASS | No orchestrator, no registry, no CI, no deployment target. Compose only |
| Dependencies justified | V | PASS | One tool — Docker — replacing four host installs. `docker compose watch` avoids adding a file-watcher dependency to the images |
| Important journeys have automated coverage | V | PASS | This feature's purpose is to make the existing coverage executable. It adds no tests of its own beyond the acceptance checks in `quickstart.md` |
| Feature began from a written specification | V | PASS | `spec.md`, written before this plan |
| Type checking, linting, tests, builds must pass | Quality Gates | PASS | Unchanged, and reachable in more places than before |

Result: **PASS**, with one at-risk gate whose resolution requires an application change, recorded
in Complexity Tracking rather than decided unilaterally.

**Post-Phase 1 re-check**: Re-evaluated after `research.md` and `quickstart.md`. No design decision
introduced an abstraction into either application, changed the contract, or altered collector-facing
behaviour. The FR-019 finding is unchanged and remains the only entry in Complexity Tracking.
Result: **PASS**.

## Project Structure

### Documentation (this feature)

```text
specs/002-docker-dev-environment/
├── plan.md              # This file (/speckit-plan command output)
├── spec.md              # Feature specification
├── research.md          # Phase 0 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

**`data-model.md` is deliberately absent**: this feature introduces no entity, no schema change, and
no persisted structure. The data model in
[feature 001](../001-add-browse-collectibles/data-model.md) is unchanged, and restating it here
would create a second copy to drift.

**`contracts/` is deliberately absent**: no new interface is exposed to users or other systems. The
REST contract in
[feature 001](../001-add-browse-collectibles/contracts/openapi.yaml) is the only one, and this
feature does not touch it. The plan template allows skipping contracts for internal tooling, and
this is that case. The developer-facing "interface" — the commands themselves — is specified in
`quickstart.md`.

### Source Code (repository root)

```text
compose.yaml                     # Replaces docker-compose.yml: all services, organised by profile
compose.prod.yaml                # Production-shaped images, for verifying them locally

backend/
├── Dockerfile                   # Multi-stage: builder -> distroless runtime
├── Dockerfile.dev               # Toolchain image for `go run` and the test suites
└── .dockerignore

frontend/
├── Dockerfile                   # Multi-stage: deps -> builder -> standalone runtime
├── Dockerfile.dev               # Node image for `next dev`
├── .dockerignore
└── next.config.ts               # Gains output: 'standalone' (the one anticipated source change)

.dockerignore                    # Repository root: keeps .git and specs out of every build context
.env.example                     # Ports and development-only values, copied to .env
Makefile                         # Thin wrappers: up, down, logs, test, test-e2e, clean, reset
```

**Structure Decision**: One `compose.yaml` carrying every service, separated by Compose profiles
rather than by separate files, so a developer learns one file and one command shape. The exception
is `compose.prod.yaml`, kept separate because it builds fundamentally different images and shares
almost nothing with the development services — merging it would mean a file where half the settings
apply and the reader must work out which half.

Development and production Dockerfiles are separate files per application rather than one file with
extra stages. A development image wants a toolchain, the source mounted, and no build step; a
production image wants none of those. One file trying to be both is the kind of thing that
accidentally ships a compiler.

The root `Makefile` exists so the documented commands are short and stable. It is a thin wrapper
over `docker compose` and adds no logic of its own.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| **FR-019 requires changing `cmd/vaultory-api/main.go`, contradicting the spec's "no application source changes" assumption** | The server currently *requires* `VAULTORY_DEV_IDENTITY=enabled` and refuses to start otherwise — deliberate in feature 001, so no deployment could silently resolve every request to one collector. FR-019 requires the opposite for a production image: refuse when development identity *is* enabled. A production image cannot start at all until this is reconciled | Leaving it alone means FR-019 and User Story 3 cannot be delivered, and the production Dockerfile builds an image that cannot run. Faking it in the entrypoint — unsetting the variable, or wrapping the binary — would hide a security-relevant decision in shell rather than expressing it in the program. The honest fix is a single reconciled check in `main.go`: refuse to start when *no* resolver is configured, and refuse to start when the development resolver is configured outside development. That keeps feature 001's guarantee intact and satisfies FR-019 with one condition rather than two contradictory ones |

No other deviation. Everything else in this feature is configuration files and image definitions.
