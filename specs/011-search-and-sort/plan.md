# Implementation Plan: Search and Sort a Collection

**Branch**: `32-search-and-sort` (stacked on `31-recoverable-trash`, #58) · **Spec**: [spec.md](spec.md) · **Issue**: #32
**Date**: 2026-10-27 · **Status**: planned

## Summary

`GET /api/collectibles` gains `q` and `sort`. The search is split into words in Go and matched in one
SQL predicate, so every word must appear in a collectible's searchable text, as a substring, matched
literally. Sorting uses a whitelist of ten orders, each a fixed SQL fragment with missing values last
and `id` as the final tiebreak. A new cursor carries the sort, whether the value was missing, the
value, and the id, so paging is exact under every order and a cursor cannot cross orders. The
gallery gets a search box and a sort menu that live in the URL beside the existing filters.

## Technical Context

| | |
|---|---|
| **Backend** | Go 1.26: domain parsing of search and sort; `Store.List` takes both; a sort-aware keyset cursor |
| **Database** | PostgreSQL 17. **No migration**: matching runs inside one collector's rows (Decision 3) |
| **API** | `listCollectibles` adds `q` (≤200) and `sort` (enum); response unchanged; contract 0.5.0 |
| **Frontend** | `SearchAndSort` control; `GalleryFilter` and `galleryHref` carry `q` and `sort`; `NoResults` names the search |
| **Testing** | Go unit tests (parsing); integration (matching, every order's paging over 500 rows, cross-order cursor refusal, Trash and cross-collector exclusion); contract (params); Vitest (links, control, no-results) |
| **Unknowns** | None. Five decisions in [research.md](research.md) |

## Constitution Check

| Principle | Assessment |
|---|---|
| **I. Collector-First** | Finding one piece without scrolling. Controls labelled and keyboard-operable; the views are shareable addresses |
| **II. Separation** | What a search means and which orders exist are domain rules in Go. The frontend sends what the collector typed |
| **III. Contract-First** | Two parameters are added to the OpenAPI document and types are regenerated; the response is unchanged |
| **IV. Data Integrity and Security** | `collector_id` and `deleted_at IS NULL` stay in every predicate. Sort SQL comes only from a whitelist, never from input. Search terms are bound parameters with wildcards escaped |
| **V. Quality, Simplicity, Spec-Driven** | No index, extension or full-text machinery until a measured need (Decision 3). One cursor format for the gallery, not one per order |

**Gate**: passes. **Re-check after design**: passes.

## Project Structure

```text
specs/011-search-and-sort/ — spec, plan, research, quickstart, tasks, contracts/list-parameters.md

backend/internal/domain/collectible/browse.go          SortOrder, ParseSortOrder, ParseSearch (new)
backend/internal/store/postgres/collectibles.go        List: search predicate, ordering, sort-aware cursor
backend/internal/store/postgres/gallery_cursor.go      the cursor (new)
backend/internal/collection/list_collection.go         passes search and sort
backend/internal/transport/httpapi/collectibles.go     reads and validates q and sort
backend/api/openapi.yaml                               q, sort; version 0.5.0
backend/tests/unit/browse_test.go                      new
backend/tests/integration/search_sort_test.go          new
backend/tests/contract/list_filters_test.go            q and sort cases
frontend/lib/gallery-link.ts, lib/api/collectibles.ts, lib/api/types.ts (regenerated)
frontend/components/collection/SearchAndSort.tsx       new
frontend/components/collection/{CollectionGallery,NoResults}.tsx, app/collection/page.tsx
frontend/tests/unit/search-and-sort.test.tsx, gallery-link tests; tests/e2e/search-sort.spec.ts
```

## Complexity Tracking

None.
