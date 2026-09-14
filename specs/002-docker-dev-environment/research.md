# Phase 0 Research: Docker Development Environment

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Date**: 2026-09-14

No `NEEDS CLARIFICATION` markers survived the specification: the two open questions — whether
"Docker-friendly" meant the database or the whole stack, and whether this is a product feature —
were resolved in the spec's Clarifications. What follows are the design decisions the requirements
constrain without dictating.

---

## Decision 1 — One Compose project, separated by profiles

**Decision**: A single `compose.yaml` containing every service, with Compose profiles `dev`, `test`,
and `e2e`. `compose.prod.yaml` stays separate.

**Rationale**: A developer should learn one file and one command shape. Profiles let `docker compose
--profile test run …` start only the test database and the runner, without a second file whose
relationship to the first has to be worked out. `compose.prod.yaml` is the exception because it
builds different images from different Dockerfiles and shares almost none of the development
settings — merging it would produce a file where half the keys apply and the reader must determine
which half.

**Alternatives considered**:

- *`docker-compose.yml` plus an override file.* The override mechanism is implicit: a developer
  reading one file cannot see what the other changes. Profiles make the grouping explicit in the
  file itself.
- *A file per concern (`compose.dev.yaml`, `compose.test.yaml`, `compose.e2e.yaml`).* Three files
  repeating the same database definition, which then drift.

---

## Decision 2 — `docker compose watch` for source sync, not a file-watcher inside the image

**Decision**: Use Compose's `develop.watch` to sync source into the running containers, with
`action: sync` for code and `action: rebuild` for dependency manifests. Requires Compose v2.22 or
newer, which the quickstart states as a prerequisite.

**Rationale**: FR-010 requires a source change to be visible without a rebuild, and SC-004 puts ten
seconds on it. The usual answer is a watcher inside the image — `air` or `reflex` for Go, which is a
dependency the project does not otherwise have, installed into an image, and configured in a file
nobody else reads. Compose does it natively. Principle V asks that dependencies provide clear
project value; a watcher that Compose already includes does not.

Bind-mounting the whole source tree is the other common answer, and it is what `watch` improves on:
a bind mount of `node_modules` across a macOS or Windows host is slow, and worse, it can mix host
binaries built for a different platform into a Linux container.

**Alternatives considered**:

- *`air` for Go and Next's own HMR through a bind mount.* Adds a dependency and a config file for
  the Go half, and keeps the bind-mount platform hazard for the Node half.
- *Rebuild on every change.* Fails SC-004 by orders of magnitude.
- *No live reload; restart manually.* Technically satisfies nothing in the spec and makes the
  environment worse than running from the host, which is the outcome to avoid.

---

## Decision 3 — Migrations run as a one-shot service the backend waits for

**Decision**: A `migrate` service using the official `migrate/migrate` image, depending on
PostgreSQL passing its health check, running `up` and exiting. The backend declares
`depends_on: { migrate: { condition: service_completed_successfully } }`.

**Rationale**: FR-002 requires migrations to be applied before the backend serves anything, and an
edge case in the spec calls out a backend starting against an unmigrated database. Compose's
`service_completed_successfully` expresses exactly that ordering, and does it declaratively — no
sleep, no retry loop, no entrypoint script that races. FR-005 requires a second run to be safe,
which `migrate up` already is: it applies only what is outstanding.

Using the official image rather than installing `golang-migrate` into the backend image keeps the
migration tool out of the runtime, which matters for FR-017.

**Alternatives considered**:

- *Run migrations from the backend's entrypoint.* Puts the migration tool in the runtime image, and
  means two backend replicas would race. It also conflates "start the server" with "change the
  schema", which are different decisions with different blast radii.
- *Migrate automatically from Go on startup.* Same coupling, and it would be an application change
  this feature has no mandate to make.
- *`postgres`'s `docker-entrypoint-initdb.d`.* Runs only on first initialisation of an empty volume,
  so it would not apply a migration added later — precisely the common case.

---

## Decision 4 — The test database is separate and lives in tmpfs

**Decision**: A `postgres-test` service under the `test` profile, on its own tmpfs-backed storage,
with its own migration one-shot. Test runs never point at the development database.

**Rationale**: FR-015 requires that tests not alter development data, and SC-008 requires the
development database to be byte-identical afterwards. The integration harness truncates tables
between tests — pointed at a development database, it would delete a developer's collection without
warning. A separate service removes the possibility rather than relying on an environment variable
being set correctly.

tmpfs because the test database is created, migrated, used, and discarded within one run. Never
touching a disk makes it faster and makes accidental persistence impossible.

**Alternatives considered**:

- *A separate database inside the same PostgreSQL instance.* One mistyped connection string away
  from destroying development data. The safety here is worth a container.
- *Reuse the development database and restore afterwards.* More moving parts, and a failed run
  leaves the developer's data in an unknown state.
- *A persistent test volume.* Invites state to leak between runs, which makes a failure depend on
  what ran before it.

---

## Decision 5 — Distroless for the Go runtime, Next standalone for the frontend

**Decision**: Backend — `golang:<version>` builder producing a statically linked binary, copied into
`gcr.io/distroless/static-debian12:nonroot`. Frontend — a Node builder running `next build` with
`output: 'standalone'`, copied into a slim Node runtime, running as a non-root user.

**Rationale**: FR-017 requires no compiler, no package manager cache, and no source in the runtime
image; FR-018 requires non-root; SC-006 sets 150 MB. Distroless `static` has no shell and no package
manager at all, which satisfies the first two by construction and lands the backend around 20 MB.

Next's `standalone` output is the supported way to get a runnable server without `node_modules` —
it traces exactly what is needed. This is the one anticipated application change the spec allowed
for: `next.config.ts` gains `output: 'standalone'`.

**A size risk worth stating plainly**: the backend will be comfortably under 150 MB. The frontend
will not obviously be — a slim Node runtime is roughly 120 MB before any application code. If
SC-006 cannot be met for the frontend, that is a finding to report against the criterion, not a
reason to reach for an unsupported runtime.

**Alternatives considered**:

- *`scratch` for the backend.* Smaller still, but no CA certificates and no `/etc/passwd`, so any
  future outbound TLS call would fail confusingly. Distroless `nonroot` costs a couple of megabytes
  and avoids that.
- *Alpine for both.* A shell and a package manager in the runtime, which is what FR-017 is trying to
  avoid. For Go, it also means either cgo against musl or disabling it anyway.
- *A single Dockerfile with development and production stages.* Rejected in the plan's Structure
  Decision: one file trying to be both is how a compiler ends up in a shipped image.

---

## Decision 6 — Uploaded images live in a named volume, not a bind mount

**Decision**: `VAULTORY_IMAGE_STORE_PATH` points at a named volume shared by the backend across
restarts and rebuilds.

**Rationale**: FR-006 and SC-005 require uploaded images to survive a stop and start, and an edge
case calls out losing test data on every change. A named volume is owned by Compose, survives
`docker compose down` (removed only by `down -v`), and avoids the file-ownership friction a bind
mount creates between a container's non-root user and a host filesystem.

**Alternatives considered**:

- *Bind mount to a host directory.* Visible to the developer, which is a real advantage, but brings
  ownership and permission problems on Linux hosts where the container user is not the host user.
- *Ephemeral storage.* Every rebuild would silently orphan every image row in the database — rows
  pointing at bytes that no longer exist, which is worse than losing both.

---

## Decision 7 — Every port is overridable, and nothing is hard-coded

**Decision**: Published ports come from environment variables with defaults —
`${VAULTORY_FRONTEND_PORT:-3000}`, `${VAULTORY_BACKEND_PORT:-8080}`,
`${VAULTORY_DB_PORT:-5432}` — read from a `.env` a developer copies from `.env.example`.

**Rationale**: FR-009 requires ports to be overridable without editing a tracked file, and FR-008
requires a port conflict to name the port. Docker's own bind failure already names the port and
address; what it cannot do is let a developer resolve it without editing a committed file. PostgreSQL
on 5432 is the most likely collision, since developers commonly already run one.

**Alternatives considered**:

- *Fixed ports.* Guarantees a collision for anyone already running PostgreSQL, and the fix is a
  local edit to a tracked file that then shows up in every diff.
- *Not publishing the database port at all.* Tempting, and it would remove the most likely conflict,
  but a developer inspecting their data with a GUI client is a reasonable thing to want.

---

## Decision 8 — Dependency caches persist in named volumes

**Decision**: Named volumes for the Go module cache, the Go build cache, and the frontend's
`node_modules`.

**Rationale**: FR-011 requires that a routine start not re-download the dependency tree, and SC-002
allows 60 seconds for a subsequent start. Without a cache, every start re-fetches, which fails both.
Holding `node_modules` in a volume rather than a bind mount is also what keeps host-built native
binaries out of the Linux container.

---

## Deferred, with reasons

- **CI pipeline** — explicitly out of scope. These images and commands are what a pipeline would
  call, so nothing here forecloses one.
- **Image registry, tags, and release process** — out of scope; no deployment target is chosen.
- **Podman compatibility** — the spec assumes Docker. Most of this would work, but `compose watch`
  and health-check condition support vary, and verifying that is work nobody has asked for.
- **A production Compose file for actual deployment** — `compose.prod.yaml` exists to verify the
  images locally, not to deploy them. Real deployment needs decisions about secrets, TLS, and a host
  that this feature deliberately does not make.
- **Multi-architecture image publishing** — images must *build* on both architectures (FR-020, SC-007),
  which is a local build concern. Publishing a multi-arch manifest belongs with the registry work.
