---

description: "Task list for feature 001-add-browse-collectibles"
---

# Tasks: Add & Browse Collectibles

**Input**: Design documents from `/specs/001-add-browse-collectibles/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[data-model.md](./data-model.md), [contracts/openapi.yaml](./contracts/openapi.yaml),
[quickstart.md](./quickstart.md)

**Tests**: Included. Vaultory's constitution requires automated tests for important business rules,
meaningful edge cases, and important user journeys (Principle V), and its Quality Gates require
backend and frontend tests to pass before a feature is complete. Test tasks are therefore mandatory
here, not optional.

**Organization**: Tasks are grouped by user story so each can be implemented, tested, and demonstrated
independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete work)
- **[Story]**: Which user story the task serves (US1, US2, US3)
- Exact file paths are given in every task

## Path Conventions

Two separated applications per the plan's Structure Decision: `backend/` (Go) and `frontend/`
(Next.js App Router). Backend domain code lives under `backend/internal/`, and nothing in
`internal/domain` may import HTTP or SQL.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Bring both applications into existence with the toolchain, contract generation, and the
same-origin proxy the design depends on.

- [ ] T001 Create the two-application directory structure — `backend/` and `frontend/` — exactly as listed in `specs/001-add-browse-collectibles/plan.md` under Source Code, including empty package directories with a `doc.go` or `.gitkeep` so the layout is committed
- [ ] T002 Initialize the Go module in `backend/go.mod` and add the dependencies justified in `research.md` Decision 8: `pgx`, `golang.org/x/image`, and nothing more
- [ ] T003 [P] Initialize the Next.js App Router application in `frontend/` with TypeScript, Tailwind CSS, and shadcn/ui, writing `frontend/package.json` and `frontend/tsconfig.json` with `strict: true` per Principle III
- [ ] T004 [P] Configure Go formatting, `go vet`, and a linter in `backend/Makefile` (or `backend/.golangci.yml`) so the constitution's type-check and lint gates are runnable
- [ ] T005 [P] Configure frontend linting and formatting in `frontend/eslint.config.mjs`, including a rule that forbids `any` per Principle III
- [ ] T006 [P] Add a local PostgreSQL and image-store development environment in `docker-compose.yml` at the repository root, plus `backend/.env.example` listing every variable in `quickstart.md`'s Environment table and no real secrets
- [ ] T007 Configure the same-origin proxy in `frontend/next.config.ts`: rewrite `/api/*` to the Go service and raise the proxy body limit above 10 MB, per `research.md` Decision 2. **Without this, image authorization and 10 MB uploads both fail**
- [ ] T008 [P] Add an `openapi-typescript` generation script to `frontend/package.json` that writes `frontend/lib/types/api.ts` from `specs/001-add-browse-collectibles/contracts/openapi.yaml`, and run it so generated types are committed
- [ ] T009 [P] Add the frontend test toolchain — Vitest with Testing Library and Playwright — in `frontend/vitest.config.ts` and `frontend/playwright.config.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The schema, the identity seam, the error envelope, the image store, and the test
harness. Everything here is shared by all three stories.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Database schema and migrations

- [ ] T010 Add the migration tooling and a `migrate` target in `backend/Makefile`, with `backend/migrations/` as the migration directory per Principle IV (version-controlled, reversible)
- [ ] T011 Write the `collectors` table migration in `backend/migrations/000001_create_collectors.up.sql` and its reverse in `000001_create_collectors.down.sql`, matching `data-model.md`
- [ ] T012 Write the `collectible_images` table migration in `backend/migrations/000002_create_collectible_images.up.sql` (and `.down.sql`), including the `content_type` CHECK, the `byte_size` CHECK bounded at 10485760, and the `UNIQUE (id, collector_id)` constraint that the composite reference from `collectibles` targets
- [ ] T013 Write the `collectibles` table migration in `backend/migrations/000003_create_collectibles.up.sql` (and `.down.sql`) with every column, CHECK, and default from `data-model.md` (FR-006, FR-025), the trimmed-name length CHECK, the four-value status CHECK, `purchase_price NUMERIC(12,2)`, and the **composite** foreign key `(image_id, collector_id) REFERENCES collectible_images(id, collector_id) ON DELETE SET NULL`
- [ ] T014 Write the two gallery indexes in `backend/migrations/000004_create_gallery_indexes.up.sql` (and `.down.sql`): `collectibles_gallery_idx` and `collectibles_status_gallery_idx` as specified in `data-model.md`
- [ ] T015 Seed the development collector in `backend/migrations/000005_seed_dev_collector.up.sql` (and `.down.sql`), guarded so it is a no-op outside development
- [ ] T016 Verify migrations apply and reverse cleanly against a real database in `backend/tests/integration/migrations_test.go`

### Core backend infrastructure

- [ ] T017 [P] Implement environment configuration loading in `backend/internal/config/config.go`, reading every variable from `quickstart.md`'s Environment table and failing fast on a missing required value; no secret has a default
- [ ] T018 [P] Implement the PostgreSQL connection pool in `backend/internal/store/postgres/pool.go` using `pgx`
- [ ] T019 [P] Define `CollectionStatus` with exactly the four permitted values and their parsing and rejection behaviour in `backend/internal/domain/collectible/status.go`, per FR-004
- [ ] T020 [P] Define the exact-decimal monetary type in `backend/internal/domain/collectible/money.go` — parsing from a decimal string, at most two fractional digits, rejecting negatives and out-of-range values, and never converting through a float, per FR-016 and FR-017
- [ ] T021 [P] Unit-test status parsing and money parsing in `backend/tests/unit/status_test.go` and `backend/tests/unit/money_test.go`, covering zero, negative, excess precision, very large amounts, and every invalid status
- [ ] T022 Implement the structured error envelope and its mapping to status codes in `backend/internal/transport/httpapi/errors.go`, matching the `ErrorResponse` schema in the contract: all field problems reported together (FR-020), no internal detail exposed, 404 rather than 403 for another collector's resource (FR-027), 401 with no content when identity is unresolvable (FR-029)
- [ ] T023 Implement the acting-collector resolution seam in `backend/internal/identity/identity.go` — one operation returning the acting collector or a failure — plus the development-only implementation in `backend/internal/identity/dev.go` (signed HTTP-only cookie mapping to the seeded collector), enabled solely by `VAULTORY_DEV_IDENTITY`, per `research.md` Decision 1. **No collector identifier may be accepted from a request body, header, or query parameter** (FR-028)
- [ ] T024 Implement request routing, middleware, and server wiring in `backend/internal/transport/httpapi/router.go` and `backend/cmd/vaultory-api/main.go`, with identity resolution applied before any handler and structured request logging that records no collector-supplied content verbatim
- [ ] T025 [P] Define the image storage interface and its filesystem implementation in `backend/internal/imagestore/store.go` and `backend/internal/imagestore/filesystem.go` — put, get, and delete by locator, with no image-format knowledge
- [ ] T026 Build the integration test harness in `backend/tests/integration/main_test.go`: a real PostgreSQL instance, migrations applied per run, and helpers that create two distinct collectors so cross-collector isolation is testable throughout

**Checkpoint**: Schema, identity, errors, routing, storage, and the test harness exist. User story
work can begin.

---

## Phase 3: User Story 1 - Add a collectible to my vault (Priority: P1) 🎯 MVP

**Goal**: A collector can add a collectible with only a name and a status, optionally with all eleven
other attributes and one uploaded image, see it confirmed, and see that it is present in their
collection.

**Independent Test**: Add a collectible with only a name and status, confirm the success state, then
confirm it is present in the collector's collection. Delivers a usable personal record on its own.

**Scope note**: The list operation lands in this phase because User Story 1's own acceptance requires
confirming the collectible is present. This phase renders that list plainly; User Story 2 turns it
into the designed visual gallery.

### Tests for User Story 1 ⚠️

> Write these first and confirm they fail before implementing.

- [ ] T027 [P] [US1] Unit-test every domain validation rule in `backend/tests/unit/collectible_validation_test.go`: name and status both required (FR-002), whitespace-only name rejected (FR-003), name length cap, status required and constrained, name-plus-status alone valid (FR-005), optional values trimmed with empty treated as absent (FR-007), notes length cap, purchase date not in the future (FR-018), release date accepted past or future in any combination (FR-019), and **all problems returned together rather than one at a time** (FR-020)
- [ ] T028 [P] [US1] Unit-test image validation in `backend/tests/unit/imaging_test.go`: over-10 MB refused (FR-010), each of JPEG, PNG, and WebP accepted, a non-image with an image extension and declared image content type refused (FR-009), a corrupt file refused, EXIF orientation applied, and a rendition produced at the one fixed aspect ratio from portrait, landscape, square, and extreme-ratio inputs (FR-014, SC-013)
- [ ] T029 [P] [US1] Contract-test `POST /collectibles` in `backend/tests/contract/add_collectible_test.go` against `contracts/openapi.yaml`: 201 body shape, the 400 error envelope with multiple field entries, 401, and that `purchasePrice` is carried as a string
- [ ] T030 [P] [US1] Contract-test `POST /images` and `GET /images/{imageId}/rendition` in `backend/tests/contract/images_test.go`: 201 body shape, 413 with `image_too_large`, 400 with `unsupported_image_format`, 404 for another collector's image, and 200 with `image/jpeg`
- [ ] T031 [P] [US1] Integration-test ownership and privacy in `backend/tests/integration/ownership_test.go`: a second collector receives 404 (**not 403**) for the first collector's collectible and image (FR-026, FR-027), an unauthenticated request receives 401 with no content (FR-029), and no request can assert an identity through a body field, header, or query parameter (FR-028)
- [ ] T032 [P] [US1] Integration-test that the database refuses a collectible referencing another collector's image, in `backend/tests/integration/image_ownership_constraint_test.go` — exercising the composite foreign key directly, not only the domain rule (FR-015)
- [ ] T033 [P] [US1] Integration-test duplicate independence in `backend/tests/integration/duplicates_test.go`: two identical submissions create two rows with distinct identifiers, neither merged nor expressed as a quantity, each carrying its own status, price, date, notes, and image (FR-023, FR-024)
- [ ] T034 [P] [US1] Integration-test monetary exactness in `backend/tests/integration/money_roundtrip_test.go`: `0.00`, fractional, and very large amounts all round-trip byte-identically through PostgreSQL (FR-016)
- [ ] T035 [P] [US1] Integration-test that a refused upload leaves no stored bytes and no image row, and that the collectible still saves afterwards with no image, in `backend/tests/integration/upload_rejection_test.go` (FR-013)
- [ ] T036 [P] [US1] Unit-test the add-collectible form's states in `frontend/tests/unit/add-collectible-form.test.tsx`: validation messages naming each offending field, the success state, the error state, and that entered values survive a failed save (FR-021, FR-022)
- [ ] T037 [P] [US1] End-to-end test the add journey in `frontend/tests/e2e/add-collectible.spec.ts`: add with name and status only, see the confirmation, see it present in the collection; then attempt a 12 MB upload, see the refusal naming the limit, and still save successfully with no image

### Backend implementation for User Story 1

- [ ] T038 [P] [US1] Implement the `Collectible` domain type and its complete validation in `backend/internal/domain/collectible/collectible.go`, covering the required and optional attributes (FR-001, FR-002, FR-006), accumulating all violations before returning, with no import of HTTP or SQL packages (Principle II)
- [ ] T039 [P] [US1] Implement image decoding and validation in `backend/internal/imaging/decode.go`: enforce the 10 MB ceiling while reading (FR-008), determine the format by decoding rather than by the declared content type or extension, and reject anything that is not a decodable JPEG, PNG, or WebP
- [ ] T040 [US1] Implement EXIF orientation handling and fixed-aspect rendition derivation in `backend/internal/imaging/rendition.go`, emitting JPEG renditions at one aspect ratio and bounded dimensions for every input (depends on T039)
- [ ] T041 [P] [US1] Implement collectible persistence in `backend/internal/store/postgres/collectibles.go`: insert, and a single-statement page read joining `collectible_images` for the rendition locator, with `collector_id` as a predicate in **every** query (FR-025) (research Decision 5)
- [ ] T042 [P] [US1] Implement image persistence in `backend/internal/store/postgres/images.go`: insert an image row owned by the acting collector, and read one by identifier scoped to its owner
- [ ] T043 [US1] Implement the image upload use case in `backend/internal/collection/upload_image.go`, composing validation, rendition derivation, storage, and the image row within a transaction so a failure leaves nothing behind (depends on T039, T040, T042, T025)
- [ ] T044 [US1] Implement the add-collectible use case in `backend/internal/collection/add_collectible.go`, verifying that any referenced image is owned by the acting collector before insert (depends on T038, T041, T042)
- [ ] T045 [US1] Implement the list-collection use case in `backend/internal/collection/list_collection.go`: newest-first keyset pagination with the identifier as tiebreaker, an opaque cursor, and the `totalUnfiltered` count as a second indexed statement (FR-034, FR-035; research Decision 7)
- [ ] T046 [US1] Implement the `POST /api/collectibles` handler in `backend/internal/transport/httpapi/add_collectible.go`, decoding strictly with unknown fields refused and mapping domain violations onto the error envelope (depends on T044, T022)
- [ ] T047 [US1] Implement the `POST /api/images` handler in `backend/internal/transport/httpapi/upload_image.go`, bounding the request body at 10 MB before reading and returning 413 or 400 as the contract specifies (depends on T043, T022)
- [ ] T048 [US1] Implement the `GET /api/collectibles` handler in `backend/internal/transport/httpapi/list_collectibles.go`, validating `cursor` and `limit` and returning a `CollectionPage` (depends on T045, T022)
- [ ] T049 [US1] Implement the `GET /api/images/{imageId}/rendition` handler in `backend/internal/transport/httpapi/get_rendition.go`, authorizing against the acting collector on **every** request, returning 404 for another collector's image, and setting `Cache-Control: private` so no shared cache retains it (FR-015)

### Frontend implementation for User Story 1

- [ ] T050 [P] [US1] Implement typed request wrappers over the contract in `frontend/lib/api/collectibles.ts` and `frontend/lib/api/images.ts`, using the generated types from `frontend/lib/types/api.ts` with no hand-written response shapes and no `any`
- [ ] T051 [US1] Build the add-collectible form in `frontend/components/collection/AddCollectibleForm.tsx` as a Client Component: name and status required, the eleven optional attributes, and one image picker. Client-side checks are courtesy only — **the server's verdict is authoritative** (Principle III)
- [ ] T052 [US1] Render server-returned field errors against their inputs, and preserve every entered value when a save fails, in `frontend/components/collection/AddCollectibleForm.tsx` (FR-020, FR-022)
- [ ] T053 [US1] Implement the image picker with its own upload, refusal messages naming the 10 MB limit and the accepted formats, and the ability to proceed with no image (FR-011, FR-012), in `frontend/components/collection/ImagePicker.tsx` (FR-009, FR-010, FR-013)
- [ ] T054 [US1] Add the add-collectible route and its success confirmation in `frontend/app/collection/new/page.tsx` (FR-021)
- [ ] T055 [US1] Add the collection route in `frontend/app/collection/page.tsx` as a Server Component that lists the collector's collectibles plainly, sufficient to confirm a newly added collectible is present. User Story 2 replaces this presentation with the designed gallery

**Checkpoint**: A collector can add collectibles, with or without an image, and confirm they are
present. Privacy, duplicates, monetary exactness, and upload refusal are all covered by tests. This
is the MVP.

---

## Phase 4: User Story 2 - Browse my collection as a visual gallery (Priority: P2)

**Goal**: The collection is presented as a premium, image-forward gallery — consistent framing,
designed placeholders, and every interface state intentionally handled.

**Independent Test**: Seed a collector's collection, open the collection view, and confirm the gallery
presentation, image-forward cards, placeholder treatment for imageless entries, and the empty,
loading, and error states.

### Tests for User Story 2 ⚠️

- [ ] T056 [P] [US2] Unit-test the collectible card in `frontend/tests/unit/collectible-card.test.tsx`: imagery is the dominant element, name and status are both present (FR-032), status is conveyed by more than colour (FR-044), and the image carries a text alternative (FR-045)
- [ ] T057 [P] [US2] Unit-test the placeholder path in `frontend/tests/unit/collectible-card-placeholder.test.tsx`: an entry with no image renders the designed placeholder rather than a broken image, and stays visually consistent with the rest of the gallery (FR-033)
- [ ] T058 [P] [US2] Unit-test the gallery's states in `frontend/tests/unit/collection-gallery-states.test.tsx`: empty with a path to add a first collectible (FR-041), loading without a blank screen and without a flash of the empty state (FR-042), and error with a retry (FR-043)
- [ ] T059 [P] [US2] Integration-test keyset pagination in `backend/tests/integration/pagination_test.go`: pages tile the collection with no entry repeated or skipped, including when rows are inserted between page reads, and entries sharing a creation instant are ordered totally by the identifier tiebreaker
- [ ] T060 [P] [US2] End-to-end test browsing in `frontend/tests/e2e/browse-collection.spec.ts`: a seeded collection renders as a gallery, scrolling loads further entries incrementally, and the empty state appears for a collector with nothing

### Implementation for User Story 2

- [ ] T061 [P] [US2] Establish the visual language in `frontend/app/globals.css` and `frontend/tailwind.config.ts` — typography, spacing, surfaces, and a first-class dark theme — and restyle the shadcn/ui primitives in `frontend/components/ui/` away from their defaults to Vaultory's own identity (Principle I; Technology Constraints)
- [ ] T062 [US2] Build the collectible card in `frontend/components/collection/CollectibleCard.tsx`: the rendition fills the card at the fixed aspect ratio with no collector-side cropping, name and status legible alongside (FR-014, FR-030, FR-032)
- [ ] T063 [P] [US2] Build the designed placeholder in `frontend/components/collection/ImagePlaceholder.tsx`, consistent with the gallery's visual language rather than a generic broken-image affordance (FR-033)
- [ ] T064 [US2] Build the gallery layout in `frontend/components/collection/CollectionGallery.tsx` as an image-forward grid — explicitly **not** a row-and-column table — responsive across desktop, tablet, and mobile with no horizontal page scroll (FR-030, FR-031, FR-036)
- [ ] T065 [P] [US2] Build the empty state in `frontend/components/collection/EmptyCollection.tsx`, offering a path to add a first collectible (FR-041)
- [ ] T066 [P] [US2] Build the loading state in `frontend/app/collection/loading.tsx` so it renders while the collection is fetched and never flashes the empty state (FR-042)
- [ ] T067 [P] [US2] Build the error state with a retry in `frontend/app/collection/error.tsx` (FR-043)
- [ ] T068 [US2] Replace the plain listing from T055 with the gallery in `frontend/app/collection/page.tsx`, keeping it a Server Component and passing the rendition paths through unchanged so `<img>` requests travel same-origin and carry the session (research Decision 2)
- [ ] T069 [US2] Implement incremental loading in `frontend/components/collection/CollectionGallery.tsx`, consuming `nextCursor` so a large collection is never presented at once (FR-035)
- [ ] T070 [US2] Confirm images load through the authorized path in the browser, not through a server-side optimizer that would strip the session — set `unoptimized` or a custom loader in `frontend/components/collection/CollectibleCard.tsx` as `research.md` Decision 2 requires

**Checkpoint**: Adding and browsing both work. The collection reads as a gallery of collectibles
rather than an inventory listing.

---

## Phase 5: User Story 3 - Filter my collection by status (Priority: P3)

**Goal**: A collector narrows the gallery to one collection status and returns to all of it in one
action, with a no-results state distinct from an empty vault.

**Independent Test**: Seed a collection containing entries in each of the four statuses, apply each
filter in turn, and confirm only matching entries appear and that a no-match result is
distinguishable from an empty vault.

### Tests for User Story 3 ⚠️

- [ ] T071 [P] [US3] Integration-test status filtering in `backend/tests/integration/status_filter_test.go`: each status returns only its own entries, an unrecognized status value is refused, `totalUnfiltered` stays independent of the filter, and the filtered query uses `collectibles_status_gallery_idx`
- [ ] T072 [P] [US3] Unit-test the filter control in `frontend/tests/unit/status-filter.test.tsx`: the active filter is indicated (FR-039), an all-statuses option is present (FR-038), and the control is keyboard operable (FR-045)
- [ ] T073 [P] [US3] Unit-test the no-results state in `frontend/tests/unit/no-results.test.tsx`, asserting it is textually and visually distinct from the empty-collection state (FR-040)
- [ ] T074 [P] [US3] End-to-end test filtering in `frontend/tests/e2e/filter-collection.spec.ts`: filter to a populated status, filter to an empty one and see the no-results state, then clear the filter in a single action

### Implementation for User Story 3

- [ ] T075 [US3] Accept and validate the `status` query parameter in `backend/internal/transport/httpapi/list_collectibles.go` and apply it in `backend/internal/collection/list_collection.go` and `backend/internal/store/postgres/collectibles.go`, within the same single statement as the page read (FR-037)
- [ ] T076 [P] [US3] Build the status filter control in `frontend/components/collection/StatusFilter.tsx`: single-select across the four statuses plus all, with the active selection clearly indicated and a one-action return to all (FR-037, FR-038, FR-039)
- [ ] T077 [P] [US3] Build the no-results state in `frontend/components/collection/NoResults.tsx`, distinct from the empty-collection state and driven by `totalUnfiltered` being above zero while the page is empty (FR-040)
- [ ] T078 [US3] Wire the filter into the collection route in `frontend/app/collection/page.tsx`, reflecting the active status in the URL so a filtered view is shareable and reloadable, and resetting pagination when the filter changes

**Checkpoint**: All three user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T079 [P] Audit accessibility across `frontend/app/collection/` and `frontend/components/collection/`: full keyboard operation of adding and browsing, focus order and visible focus, text alternatives on every image, and status never signalled by colour alone (FR-044, FR-045, SC-011)
- [ ] T080 [P] Verify the gallery at desktop, tablet, and mobile widths with no horizontal page scroll and no clipped or overlapping content, recording the checks in `frontend/tests/e2e/responsive.spec.ts` (FR-036, SC-010)
- [ ] T081 Measure a 500-collectible collection becoming browsable within 2 seconds and scrolling without stalling, with a seeding helper in `backend/tests/integration/seed_large_collection_test.go` (SC-004)
- [ ] T082 [P] Confirm no N+1 access on the gallery path by asserting statement counts per page load in `backend/tests/integration/query_count_test.go` — the page read and the count, and nothing per entry
- [ ] T083 [P] Verify text fidelity for non-Latin scripts, accented characters, and emoji through the whole path in `backend/tests/integration/text_fidelity_test.go`
- [ ] T084 [P] Confirm error responses never carry internal detail — no driver messages, no SQL, no stack traces — in `backend/tests/integration/error_leakage_test.go` (Principle IV)
- [ ] T085 [P] Document how to run both applications, become the development collector, and apply migrations in `README.md`, referring to `specs/001-add-browse-collectibles/quickstart.md` rather than duplicating it
- [ ] T086 [P] Record the deferred items from `research.md` — reclaiming unreferenced uploads, WebP renditions, the catalog and owned-instance split, currency selection, upload rate limiting — as tracked follow-ups in `docs/deferred.md`, so none is silently forgotten
- [ ] T087 Regenerate `frontend/lib/types/api.ts` from the contract and confirm it is unchanged, proving the implementation did not drift from `contracts/openapi.yaml` (Principle III)
- [ ] T088 Walk every scenario in `specs/001-add-browse-collectibles/quickstart.md` — Walkthroughs A through H — and confirm each expected result, **including Walkthrough G, where another collector's image must return 404 rather than 403**
- [ ] T089 Run the constitution's Quality Gates in full: `go vet ./...`, `go build ./...`, `go test ./...` in `backend/`, and `npm run lint`, `npx tsc --noEmit`, `npm run build`, `npm run test`, `npm run test:e2e` in `frontend/`
- [ ] T090 Review this feature against `.specify/memory/constitution.md` and confirm the Constitution Check table in `plan.md` still holds, with the dev-only identity seam remaining the only entry in Complexity Tracking

---

## Phase 7: Remediation from `/speckit-analyze` (2026-09-10)

**Purpose**: Close the one constitution violation and the two conflicts the analysis found. T091 and
T093 are **not optional polish** — they carry requirements added to the spec on 2026-09-10.

**Ordering**: T091 belongs with Phase 4's visual work (do it alongside T061, not after). T093 and
T094 belong with Phase 3's add path (do them alongside T044 and T046). They are listed separately
only because they postdate the original breakdown.

- [ ] T091 Define both appearance palettes on one set of themed tokens in `frontend/app/globals.css` and `frontend/tailwind.config.ts` — dark as the default, light honoured from the collector's system preference, no manual toggle — and apply them across `frontend/components/collection/` and `frontend/components/ui/` (FR-046). **Do this with T061; retrofitting a second palette after the gallery is styled means revisiting every surface**
- [ ] T092 [P] Verify every screen and every interface state in both appearances in `frontend/tests/e2e/appearance.spec.ts` — empty, loading, validation, success, error, and no-results — asserting contrast compliance in each (SC-015)
- [ ] T093 Write the `collectible_submissions` table migration in `backend/migrations/000006_create_collectible_submissions.up.sql` (and `.down.sql`) with the composite primary key `(collector_id, submission_key)` from `data-model.md`, then make the add-collectible use case in `backend/internal/collection/add_collectible.go` idempotent per submission key — recording the key in the same transaction as the collectible and returning the existing collectible on a repeat (FR-047)
- [ ] T094 Accept and validate the submission key in `backend/internal/transport/httpapi/add_collectible.go`, and generate one per add attempt in `frontend/components/collection/AddCollectibleForm.tsx`, regenerating it only when the collector begins a new collectible so a retry replays the same key (FR-047)
- [ ] T095 [P] Integration-test submission idempotency in `backend/tests/integration/idempotency_test.go`: a replayed key returns the same collectible and creates no second row, two **concurrent** requests with one key create exactly one collectible (the uniqueness constraint, not a check-then-insert, must be what holds), and a different key with identical values creates an independent entry — **proving FR-047 without weakening FR-023** (SC-016)
- [ ] T096 [P] Regenerate `frontend/lib/types/api.ts` from the contract and confirm the generated request type carries the required `submissionKey`. The contract already declares it — `specs/001-add-browse-collectibles/contracts/openapi.yaml` was amended 2026-09-10 — so this task only proves the generated types have not drifted (Principle III)
- [ ] T097 [P] Pin the rendition geometry — 4:5 portrait at 800×1000, filling the frame and trimming centrally — as named constants in `backend/internal/imaging/rendition.go`, and match the card's frame to it in `frontend/components/collection/CollectibleCard.tsx` so the two cannot drift (FR-014)

**Checkpoint**: The constitution's dark-mode requirement is satisfied and verified, a retried add can
no longer strand a collector with an undeletable duplicate, and the gallery's framing is a stated
constant rather than an incidental choice.

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (Phase 1)**: no dependencies; start immediately
- **Foundational (Phase 2)**: needs Setup; **blocks all three user stories**
- **User Story 1 (Phase 3)**: needs Foundational. No dependency on US2 or US3
- **User Story 2 (Phase 4)**: needs Foundational. Depends on US1 only for the list operation and image renditions it consumes; against a seeded collection it is independently testable
- **User Story 3 (Phase 5)**: needs Foundational. Extends the list operation from US1; independently testable against a seeded collection
- **Polish (Phase 6)**: needs the user stories you intend to ship
- **Remediation (Phase 7)**: not a trailing phase. T091 runs with T061 in US2; T093, T094, T096, and
  T097 run with US1's add path; T092 and T095 follow their implementations

### Critical path

T001 → T002 → T010 → T011 → T012 → T013 → T023 → T024 → T038 → T044 → T046 → T055

T007 is off the critical path but **blocks all image behaviour and every browser-side image test**;
do it during Setup, not later.

### Within each user story

- Tests are written first and must fail before implementation
- Domain types before use cases; use cases before handlers; handlers before frontend consumption
- Migrations before any persistence work
- T039 before T040 (rendition derivation needs decoding)
- T061 before T062 and T064 (the visual language precedes what is built on it)

### Parallel opportunities

- Setup: T003, T004, T005, T006, T008, T009 all in parallel after T001 and T002
- Foundational: T017, T018, T019, T020, T021, T025 in parallel; the migration chain T011 → T014 is sequential
- US1: all ten test tasks T027–T037 in parallel; then T038, T039, T041, T042 in parallel
- US2: T063, T065, T066, T067 in parallel once T061 lands
- US3: T076 and T077 in parallel with the backend work in T075
- Polish: T079, T080, T082, T083, T084, T085, T086 all in parallel

### Parallel Example: User Story 1

```bash
# All User Story 1 tests together — they must fail before implementation:
Task: "Unit-test domain validation in backend/tests/unit/collectible_validation_test.go"
Task: "Unit-test image validation in backend/tests/unit/imaging_test.go"
Task: "Contract-test POST /collectibles in backend/tests/contract/add_collectible_test.go"
Task: "Contract-test the image operations in backend/tests/contract/images_test.go"
Task: "Integration-test ownership and privacy in backend/tests/integration/ownership_test.go"
Task: "Integration-test the composite image constraint in backend/tests/integration/image_ownership_constraint_test.go"
Task: "Integration-test duplicate independence in backend/tests/integration/duplicates_test.go"
Task: "Integration-test monetary exactness in backend/tests/integration/money_roundtrip_test.go"
Task: "Integration-test upload rejection in backend/tests/integration/upload_rejection_test.go"
Task: "Unit-test the add form's states in frontend/tests/unit/add-collectible-form.test.tsx"

# Then the independent implementation pieces:
Task: "Collectible domain type in backend/internal/domain/collectible/collectible.go"
Task: "Image decoding and validation in backend/internal/imaging/decode.go"
Task: "Collectible persistence in backend/internal/store/postgres/collectibles.go"
Task: "Image persistence in backend/internal/store/postgres/images.go"
```

---

## Implementation Strategy

### MVP first (User Story 1 only)

1. Phase 1 — Setup, T007 included
2. Phase 2 — Foundational
3. Phase 3 — User Story 1
4. **Stop and validate**: Walkthroughs A, B, C, D, G, and H from `quickstart.md`
5. A collector can record their collection privately, images and all. Demonstrable

### Incremental delivery

1. Setup + Foundational → foundation ready
2. User Story 1 → validate → **MVP**: adding and confirming presence
3. User Story 2 → validate → the collection becomes the premium gallery the product promises
4. User Story 3 → validate → filtering makes a growing collection manageable
5. Polish → accessibility, performance, and the constitution's quality gates

### Parallel team strategy

Foundational is genuinely blocking, so complete it together. Afterwards, one developer can own the
Go service through US1 while another establishes the visual language in T061 — the highest-leverage
frontend task and the one everything else in US2 and US3 builds on. US3 is small enough to fold into
whoever finishes first.

---

## Notes

- [P] marks tasks touching different files with no dependency on incomplete work
- [Story] labels map each task to a user story for traceability
- Every task names the file it changes
- Confirm tests fail before implementing against them
- Commit after each task or logical group
- Any checkpoint is a safe place to stop and validate a story on its own
- Two rules are load-bearing and easy to get wrong: **another collector's resource returns 404, never
  403** (FR-027), and **money never passes through a floating-point type** (FR-016)
