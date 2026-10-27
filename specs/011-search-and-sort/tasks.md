# Tasks: Search and Sort a Collection

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [contracts/list-parameters.md](contracts/list-parameters.md), [quickstart.md](quickstart.md)

## Phase 1: Setup

- [X] T001 Add `q` and `sort` to `listCollectibles` in `backend/api/openapi.yaml`, version 0.5.0; regenerate `frontend/lib/types/api.ts`

## Phase 2: Foundational

- [X] T002 `backend/internal/domain/collectible/browse.go`: `SortOrder` with the ten values and `DefaultSortOrder`; `ParseSortOrder`; `ParseSearch(raw) ([]string, *Violation)` — trim, ≤200 characters, split on whitespace, ≤20 words. No escaping here: that is an `ILIKE` detail and belongs to the store (Constitution II) (FR-002–FR-004, FR-013)
- [X] T003 `backend/tests/unit/browse_test.go`: parsing, escaping, limits, unknown sort
- [X] T004 `backend/internal/store/postgres/gallery_cursor.go`: encode and decode `order|missing|value|id`, the value as the database's own text form and bound back with a cast (`::numeric`, `::date`, `::timestamptz`) — never through a float (Constitution IV); read the legacy two-part cursor as `added_desc`; refuse an order mismatch (FR-012, research Decision 2)

## Phase 3: User Story 1 — Search (P1) 🎯 MVP

- [X] T005 [US1] `ListFilter.Search []string` and the predicate in `Store.List` (research Decision 1), escaping `\ % _` in the store; `TotalUnfiltered` unchanged (FR-001–FR-003, FR-014)
- [X] T006 [US1] Handler reads `q` through `ParseSearch`, 400 on violation (FR-004)
- [X] T007 [US1] `backend/tests/integration/search_sort_test.go`: each attribute matches; words across attributes; case; literal `%`/`_`/`\`; no match; Trash and another collector never match (FR-002, FR-003, FR-014, SC-006)

## Phase 4: User Story 2 — Sort (P1)

- [X] T008 [US2] `Store.List` takes a `SortOrder`: whitelisted ORDER BY and the keyset predicate per research Decision 2 (FR-006–FR-009)
- [X] T009 [US2] Handler reads `sort` through `ParseSortOrder`, 400 on unknown; a cursor/order mismatch is 400 like a malformed cursor (FR-012, FR-013)
- [X] T010 [US2] *As built: its first run caught a cursor ending on a missing value binding `''`,
  which PostgreSQL casts while planning; a missing value now binds NULL.* Same test file: 500 collectibles with tied and missing values; for every order, pages of 24 return each exactly once, in order, with missing values last; time the first page of each order (FR-007, FR-008, SC-002, SC-003, SC-005)
- [X] T011 [US2] Same test file: a cursor from one order refused under another; a legacy cursor still pages `added_desc`

## Phase 5: User Story 3 — Together, and kept (P2)

- [X] T012 [US3] Same test file: status, series and manufacturer filters with search and sort compose (FR-010)
- [X] T013 [US3] `frontend/lib/gallery-link.ts`: `GalleryFilter.q` and `.sort`; `galleryHref` writes them (omitting the default order); `isFiltered` counts `q`. `lib/api/collectibles.ts` `ListParams` gains both (FR-011)
- [X] T014 [US3] `frontend/components/collection/SearchAndSort.tsx`: labelled search field (Enter or button), Clear, sort select applying on change; keeps every other parameter, drops `cursor` (FR-011, FR-015)
- [X] T015 [US3] `app/collection/page.tsx` reads `q` and `sort` (an unknown sort treated as none), passes them to the fetch, the gallery and the control; `CollectionGallery` sends both on every further page; `NoResults` names the search; the controls are hidden on an empty vault, as the status filter is (FR-005, FR-011, spec edge cases)
- [X] T016 [P] [US3] Vitest: `galleryHref` with `q`/`sort`; `SearchAndSort` submits and preserves parameters; `NoResults` names the search
- [X] T017 [US3] Contract tests in `backend/tests/contract/list_filters_test.go`: `q` over 200 and unknown `sort` are 400; a cross-order cursor is 400; a sorted response has the `CollectionPage` shape

## Phase 6: Polish

- [X] T018 `frontend/tests/e2e/search-sort.spec.ts`: search, sort, edit and return (written; runs with the Docker stack)
- [X] T019 Full gate: backend gofmt, vet (± integration tag), build, unit, integration, contract; frontend lint, typecheck, unit, build
