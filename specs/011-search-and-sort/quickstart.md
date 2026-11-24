# Quickstart: verifying search and sort

Backend checks need PostgreSQL with migrations applied.

- **A — Search (US1)**: integration `TestSearch*`: every attribute, several words across attributes,
  case, literal `%` and `_`, no match. Browser: type a manufacturer word, press Enter.
- **B — Sort (US2, SC-002, SC-003)**: integration `TestEveryOrderPagesExactlyOnce` pages 500
  collectibles with ties and missing values through all ten orders in pages of 24.
- **C — Together (US3)**: integration `TestSearchSortAndFiltersCompose`; Vitest for `galleryHref`
  carrying `q` and `sort`; browser: filter, search, sort, edit, save, and the view is unchanged.
- **D — Refusals (FR-012, FR-013)**: contract tests: `q` over 200, unknown `sort`, cross-order cursor.
- **E — Privacy (SC-006)**: integration: neither the Trash nor another collector's vault is searched.
