---

description: "Task list for feature 008 — collection value and statistics"
---

# Tasks: Collection Value & Statistics

**Input**: Design documents from `/specs/008-collection-value-stats/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: Required. The constitution puts business logic in Go and this feature's entire value is
being correct about numbers — a statistics feature without tests asserting exactness is a statistics
feature nobody should trust.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel — different files, no dependency
- **[Story]**: US1, US2, US3, or blank for shared work

## Path Conventions

Web application: `backend/` (Go) and `frontend/` (Next.js), per [plan.md](./plan.md).

---

## Phase 1: Setup — the contract first, because everything is generated from it

- [X] T001 Bump `specs/001-add-browse-collectibles/contracts/openapi.yaml` to **0.3.0**
- [X] T002 Add `GET /api/collectibles/statistics` to the contract — no parameters of any kind, so that FR-003 is satisfied by the endpoint's shape rather than by a check
- [X] T003 Add `CollectionStatistics`, `StatusSummary` and `GroupSummary` schemas; every monetary field is a decimal **string**, never a number (FR-005, FR-040)
- [X] T004 Add the `kind` enum (`named`, `unspecified`, `remainder`) to `GroupSummary`, so the view never infers a group's nature from its label
- [X] T005 Add optional `series` and `manufacturer` query parameters to `GET /api/collectibles` (FR-030, FR-031 — research.md Decision 8)
- [X] T006 Declare the four collection statuses in the contract so the frontend cannot hand-write them (FR-018) — already present as `CollectionStatus` since 0.1.0; verified rather than re-added
- [X] T007 Regenerate `frontend/lib/types/api.ts` with `npm run generate:api` (the script is `generate:api`, not `generate:types` as first written here); +163 lines, all generated

**Checkpoint**: the contract describes the feature, and the frontend's types come from it.

---

## Phase 2: Foundational — exactness, before anything can display a wrong number

**⚠️ BLOCKING**: no story may begin until a total can be represented and parsed safely.

- [X] T008 Extend `backend/internal/domain/collectible/money.go` with a parse taking an explicit maximum; keep `ParseMoney` bound to `maxMoneyUnits` (the column's limit) unchanged
- [X] T009 Add a totals parse bound by the representation rather than the column, with a **named** error — `ErrTotalTooLarge` or equivalent (FR-013, research.md Decision 3)
- [X] T010 [P] Unit test: a total above a single amount's ceiling parses successfully, proving totals are not bound by the column's limit
- [X] T011 [P] Unit test: a total above the *representation's* ceiling returns the named error rather than a wrapped or negative figure — provoked deliberately, because no realistic input reaches it
- [X] T012 [P] Unit test: exact decimal round-trip through the totals parse, including values with trailing zeros
- [X] T013 Create `backend/internal/domain/collectible/statistics.go` with `CollectionStatistics`, `StatusSummary`, `GroupSummary` and the `GroupKind` enum
- [X] T014 Implement the folding rule in the domain: four statuses always present, absent ones filled with zeros (FR-015)
- [X] T015 Implement the ordering rule: named groups first by size, then the unspecified group, then the remainder — both below every named group whatever their own size (FR-022, corrected during planning after a real run put *everything else* above *unspecified*)
- [X] T016 [P] Unit test: a vault using two statuses still yields four `StatusSummary` entries
- [X] T017 [P] Unit test: ordering puts a large remainder below a small named group
- [X] T018 [P] Unit test: `pricedCount` distinguishes "no priced items" from "items that cost nothing"

**Checkpoint**: money and shapes are provably safe; nothing yet reads the database.

---

## Phase 3: User Story 1 — See what the collection holds and what it cost (P1) 🎯 MVP

**Goal**: a collector sees a piece count and an exact total they could not previously obtain.

**Independent Test**: add collectibles with known prices; the count and total match to the cent.

- [X] T019 [US1] Create `backend/internal/store/postgres/statistics.go` with the status aggregate from data-model.md, predicated on `collector_id`, inside a single **read-only transaction** shared by all three statements so every figure describes one moment (FR-002, FR-017a, research.md Decision 10)
- [X] T020 [US1] Scan `sum(purchase_price)` into a **string** and parse it; no `float64` anywhere in the path (FR-004, research.md Decision 2)
- [X] T021 [US1] Create `backend/internal/collection/statistics.go`, deriving the total piece count and the total of recorded prices from the four status rows in Go, never in the browser (FR-001, FR-007, FR-008)
- [X] T022 [US1] Create `backend/internal/transport/httpapi/statistics.go` and add the response shapes to `dto.go`
- [X] T023 [US1] Register `GET /api/collectibles/statistics` behind `requireCollector` in `server.go`
- [X] T024 [P] [US1] Integration test: counts and totals over a vault of known contents
- [X] T025 [P] [US1] Integration test: **exactness** — prices of `0.10`, `0.20`, `0.30` total exactly `0.60`, compared against a total assembled independently of the code under test (SC-002)
- [X] T026 [P] [US1] Integration test: collectibles without a price are excluded from totals and counted in `pricedCount` (FR-009, FR-010)
- [X] T027 [P] [US1] Integration test: an empty vault returns zeros with success, not an error (SC-008)
- [X] T028 [P] [US1] Integration test: one collector's figures are unaffected by another collector's vault (FR-002, SC-006)
- [X] T029 [P] [US1] Contract test: the response matches the contract, with monetary fields as strings
- [X] T030 [US1] Extend `backend/tests/integration/query_count_test.go`: the cost does **not move** between a small vault and a large one (FR-006, SC-004). Note in the test that `statementCount` reads `xact_commit + xact_rollback` and so counts *transactions*, not statements — assert the property, not a number whose name misleads
- [X] T031 [US1] Create `frontend/lib/api/statistics.ts` using the generated types; hand-write no response shape
- [X] T032 [US1] Create `frontend/app/collection/stats/page.tsx` at its own route as an RSC forwarding cookies, computed per request so figures are never stale; no session redirects to sign-in rather than erroring (FR-028, FR-032, FR-039)
- [X] T033 [US1] Create `frontend/components/collection/CollectionSummary.tsx` — headline figures, displaying what the API returns and computing nothing (FR-001, FR-033)
- [X] T034 [US1] State how many pieces a total covers whenever some carry no price (FR-011)
- [X] T035 [P] [US1] Unit test: the summary renders the API's figures verbatim and derives nothing
- [X] T036 [P] [US1] Unit test: the "not all pieces are priced" statement appears only when it is true

**Checkpoint**: US1 delivers alone — a collector learns what their vault cost.

---

## Phase 4: User Story 2 — Understand the split across statuses (P2)

**Goal**: four statuses, each with its own count and total, the headline meaning one thing.

**Independent Test**: collectibles across all four statuses report the right counts and subtotals, including statuses holding nothing.

- [X] T037 [US2] Return all four `StatusSummary` entries from the service with their counts and monetary totals, zeros included (FR-014, FR-015, FR-016)
- [X] T038 [US2] Create `frontend/components/collection/StatusBreakdown.tsx`
- [X] T039 [US2] Present Owned as the headline; show Preordered, Wishlist and Sold distinctly (FR-012)
- [X] T040 [US2] Label the Sold figure as what was **paid**, in those words, and keep it out of the headline (FR-019)
- [X] T041 [US2] Link each status figure to the gallery filtered to it (FR-030)
- [X] T042 [P] [US2] Integration test: per-status counts and totals reconcile exactly with the whole-vault figures, and the three aggregates agree with each other because they share one transaction (FR-017, FR-017a, SC-003)
- [X] T043 [P] [US2] Integration test: a status holding nothing returns a zero entry rather than being omitted (SC-007)
- [X] T044 [P] [US2] Integration test: changing a collectible's status moves it between subtotals
- [X] T045 [P] [US2] Unit test: all four statuses render even when three are empty
- [X] T046 [P] [US2] Unit test: the Sold figure carries its "paid" wording — asserted, because it is a requirement and not copy

**Checkpoint**: US1 and US2 both stand alone.

---

## Phase 5: User Story 3 — Which series and manufacturers dominate (P3)

**Goal**: breakdowns that group the way a collector thinks, and that trace back to real pieces.

**Independent Test**: groups match the collectibles behind them, and nothing is dropped.

- [X] T047 [US3] Add the series and manufacturer breakdown statements to `store/postgres/statistics.go`, grouping on `lower(btrim(…))` with `coalesce` **before** normalising, so `NULL`, `''` and `'   '` form one group (FR-020, FR-021, FR-024, FR-027, research.md Decision 6)
- [X] T048 [US3] Use `mode() WITHIN GROUP` for the display spelling; ties resolve alphabetically on the trimmed spelling, which the test asserts as *repeatable* rather than as the collector's preference (FR-027a)
- [X] T049 [US3] Fold the tail in SQL with `row_number()`, so the response is bounded in rows as well as statements (FR-025, research.md Decision 5)
- [X] T050 [US3] Map key `''` to `unspecified` and key `NULL` to `remainder` — verified distinct against PostgreSQL during planning, though `psql` renders both blank
- [X] T051 [US3] Add `series` and `manufacturer` filters to `Store.List` in `collectibles.go` as `coalesce(lower(btrim(col)), '') = $n`, normalising **identically** to the grouping; the unspecified group's link passes an empty value and must return the `NULL`, `''` and whitespace rows alike — `IS NULL` is the obvious choice and is wrong (research.md Decision 8)
- [X] T052 [US3] Accept the two filters in the list handler and thread them through the gallery
- [X] T053 [P] [US3] Integration test: near-duplicate spellings form one group, labelled with the most-used spelling
- [X] T054 [P] [US3] Integration test: `NULL`, empty and whitespace-only values land in **one** unspecified group, not two
- [X] T055 [P] [US3] Integration test: breakdown counts reconcile with the total, including unspecified and remainder (FR-026, SC-003)
- [X] T056 [P] [US3] Integration test: **a group's count equals the number of collectibles its own link returns** — the inconsistency no unit test catches (SC-010)
- [X] T057 [P] [US3] Integration test: a vault with more distinct series than the bound returns a bounded row count
- [X] T058 [P] [US3] Integration test: ordering is stable across repeated calls when groups tie, and the unspecified and remainder groups sort below every named group however large they are (FR-022, FR-023)
- [X] T059 [US3] Create `frontend/components/collection/GroupBreakdown.tsx`, shared by both dimensions
- [X] T060 [US3] Link each group to its filtered gallery (FR-031)
- [X] T061 [P] [US3] Unit test: the unspecified and remainder groups are rendered from `kind`, never inferred from the label
- [X] T062 [P] [US3] Unit test: a group's link carries the filter that reproduces it

**Checkpoint**: all three stories independently functional.

---

## Phase 6: Polish & Cross-Cutting

- [X] T063 Create `frontend/app/collection/stats/loading.tsx` — a loading state, never a zero that will shortly change (FR-038)
- [X] T064 Designed empty state for a vault with nothing in it, offering the route to adding a collectible (FR-034)
- [X] T065 Add the named link from the gallery to the statistics (FR-029)
- [X] T066 [P] E2E `frontend/tests/e2e/statistics.spec.ts`: the headline journey, desktop
- [X] T067 [P] E2E: a figure links to a gallery showing exactly the pieces it counted (SC-010)
- [X] T068 [P] E2E: the empty state, and the partially-priced statement
- [X] T069 [P] Extend `frontend/tests/e2e/appearance.spec.ts` — the statistics view in both appearances (FR-035, SC-009)
- [X] T070 [P] Extend `frontend/tests/e2e/accessibility.spec.ts` — keyboard reachable, every control named, at all three viewports (FR-036, SC-009)
- [X] T071 [P] Extend `frontend/tests/e2e/responsive.spec.ts` — three viewports, no horizontal scroll (FR-037, SC-009)
- [X] T072 Confirm no migration was added — `git diff --stat main...HEAD -- backend/migrations/` is empty; the directory still ends at `000010` (FR-041, research.md Decision 1)
- [X] T073 Confirm no figure derives from a field the product does not record — a grep for market value, sale price, profit and resale across `backend/internal` and the frontend returns one hit, in the generated contract types, and it is the sentence saying no such figure exists (FR-042)
- [X] T074 Update `CLAUDE.md`: add feature 008, and mark 006 complete
- [ ] T075 Run `quickstart.md` end to end, including each of its four deliberate failure checks

---

## Phase 7: Verification — every gate run, every number recorded as measured

- [ ] T076 `cd backend && go vet ./... && go build ./...` — check the **exit status**, not the output of a pipeline whose last command is `head`
- [ ] T077 `go vet -tags integration ./...` — type-check the tagged suites
- [ ] T078 Backend unit suite
- [ ] T079 `make test` — integration and contract against a real database, confirmed **by test name**, because a missing `VAULTORY_TEST_DATABASE_URL` makes `TestMain` exit 0 with nothing run
- [ ] T080 Frontend lint, typecheck, unit, build
- [ ] T081 `make test-e2e` full suite — with **nothing else touching Docker**, because a contended run has twice produced failures that were mine and not the product's
- [ ] T082 Feature 008 browser tests alone on desktop, one worker, for a clean signal
- [ ] T083 `make prod-build`; record the frontend image size, which this feature moves again (issue #29) — record it, do not fix it
- [ ] T084 Measure the statistics view against a vault an order of magnitude larger than typical and record the render time against the budget the gallery already meets (SC-005) — a gap found by the analyze pass, which had no task covering it
- [ ] T085 Record every result in this file as measured, including flakes, rather than rounded away

---

## Dependencies & Execution Order

- **Phase 1** → everything. The contract is the source of truth; building before it exists means hand-writing a shape.
- **Phase 2** blocks all stories. A story that displays a total before totals can be represented safely is a story that displays a wrong total.
- **US1 (Phase 3)** → **US2 (Phase 4)**: they share the status statement. US2 is a presentation layer over a query US1 already built, which is why they are not parallel despite being separate stories.
- **US3 (Phase 5)** depends only on Phase 2 and may be built alongside US1 if staffed, since its statements and components are its own.
- **Phases 6 and 7** depend on the stories being done.

### Parallel opportunities

All `[P]` tests within a phase touch different files. The three `[P]` unit tests in Phase 2
(T010–T012) are the earliest useful parallel batch; T024–T029 and T053–T058 are the largest.

---

## Implementation Strategy

**MVP is Phase 1 + 2 + 3.** It delivers the figure a collector cannot get today. Stop there and it is
still worth shipping; SC-001, SC-002, SC-004, SC-006 and part of SC-008 are all satisfiable at that
point.

Then US2, then US3. Each adds a dimension without changing what came before.

---

## Notes

- Commit after each logical group, push at every checkpoint. A local commit is invisible.
- The failure this feature is most at risk of is a number that is quietly wrong. Prefer a test that
  compares against an independently assembled expectation over one that recomputes the same way the
  code does — the latter proves only that the code agrees with itself.
- Where a test is asserted to pass, mutate the thing it checks and confirm it fails. A test that
  cannot fail is not evidence.
