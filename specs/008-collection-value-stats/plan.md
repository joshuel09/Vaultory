# Implementation Plan: Collection Value & Statistics

**Branch**: `39-collection-value-stats` | **Date**: 2026-10-13 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/008-collection-value-stats/spec.md`

## Summary

A collector opens `/collection/stats` and learns what their vault holds: how many pieces, what was
paid for the ones they own, how that splits across the four collection statuses, and which series and
manufacturers dominate the shelf. Every figure is computed in Go over the requesting collector's own
rows; the browser receives finished numbers and sums nothing.

This feature adds no schema. It reads what features 001 and 006 already store. Almost every risk in
it is the same risk wearing different clothes — **that a number is quietly wrong**:

- **`SUM(numeric)` scanned as text, never as a float.** This is the one that would survive every
  small test and fail on a real vault. `Money` has no float in its API precisely to prevent it, but
  a `pgx` scan into `float64` bypasses the type entirely.
- **Totals get a ceiling of their own**, seven orders of magnitude above a single stored amount, and
  the accumulation guards it with a named error. An unguarded int64 total wraps to a *negative*
  number, which nobody reads as a bug.
- **Three statements, constant.** One `GROUP BY collection_status` serves both the headline and the
  status split; one statement per breakdown dimension. The breakdown tail is folded in SQL so the
  response is bounded in rows as well as in queries.
- **Groups and their links must agree.** A group assembled with `lower(btrim(…))` that links to a
  filter matching exact text shows a collector a group of five and then a gallery of three.

The one piece of genuinely new scope is that **the gallery gains series and manufacturer filters**,
because FR-030 and FR-031 require every figure to trace back to the pieces behind it and only status
filters today. It is called out here rather than discovered mid-implementation.

Full reasoning for each decision, including what was rejected, is in [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.26 (backend), TypeScript 5.x on Next.js 15.5 App Router (frontend)

**Primary Dependencies**: Go standard library `net/http`, `jackc/pgx/v5`, `google/uuid`; React 19,
Tailwind CSS, shadcn/ui foundations. **No new dependency in either application.** The view is figures
and groups rendered as text and layout — charting libraries are out of scope by the spec, which is
also why none is needed.

**Storage**: PostgreSQL 17. **No migration.** `collectibles` already carries `collection_status`
(constrained to the four values), `purchase_price numeric(12,2)` nullable, and nullable `series` and
`manufacturer` (research.md Decision 1).

**Testing**: `go test` — unit (no database), integration and contract (tagged, real database, `-p 1`);
Vitest for frontend units; Playwright across desktop, tablet and mobile projects.

**Target Platform**: Linux containers under Docker Compose; browsers from mobile to desktop.

**Performance Goals**: the complete view costs three statements regardless of collection size
(SC-004), asserted with the existing `statementCount(t)` helper. Render within the budget the
gallery already meets, for a vault an order of magnitude larger than typical (SC-005).

**Constraints**: exact decimals end to end; every query predicated on `collector_id` from the
verified session; no collector identifier accepted from the caller in any form.

**Scale/Scope**: one new read endpoint, one new route, two new gallery filters, three new frontend
components, and the extension of `money.go` for totals.

## Constitution Check

| Principle | How this feature satisfies it | Where it could go wrong |
|---|---|---|
| Go owns business logic | Every count, sum, grouping, ranking and ordering happens in Go and PostgreSQL. The frontend receives finished figures. | A "helpful" total computed in React from a list — FR-001 exists to forbid exactly this. |
| Exact decimals, never float | `numeric` sums scanned as text into minor units; no `float64` in the path. | A `pgx` scan into `float64`, or a JSON number on the wire. Both are invisible until a real vault. |
| Vault privacy, 404 never 403 | The endpoint takes no identifier at all; the collector comes from the verified session. There is no "someone else's statistics" to request. | Accepting a collector id as a convenience parameter. FR-003 forbids it structurally. |
| OpenAPI is the source of truth | The response is described completely in the contract, bumped to 0.3.0; frontend types regenerate. | Hand-writing the response shape in TypeScript. |
| Reversible migrations | Not applicable — no schema change (research.md Decision 1). | Reaching for a materialised summary table, which this plan rejects. |
| Version on write | Not applicable — nothing here writes. | — |
| Image rows vs unlinking | Not applicable — no image handling. | — |
| Session verified in Go | The route requires a collector, like every other collection route. | — |

**No deviations to record.** The Complexity Tracking table below is empty, and that is the honest
result rather than an omission.

## Project Structure

### Documentation (this feature)

```text
specs/008-collection-value-stats/
├── spec.md
├── plan.md              # this file
├── research.md          # nine decisions, with what was rejected
├── data-model.md        # the shapes, the statements, and the exactness rules
├── contracts/README.md  # pointer to the contract's real home
├── quickstart.md        # how to see it work, and how to see it fail
└── tasks.md             # produced by the tasks step
```

### Source Code (repository root)

```text
backend/
├── internal/
│   ├── domain/collectible/
│   │   ├── money.go               # extended: a totals ceiling and a named error
│   │   └── statistics.go          # NEW: the summary types and the folding rules
│   ├── store/postgres/
│   │   ├── statistics.go          # NEW: three aggregate statements
│   │   └── collectibles.go        # extended: series and manufacturer filters on List
│   ├── collection/
│   │   └── statistics.go          # NEW: the application service
│   └── transport/httpapi/
│       ├── statistics.go          # NEW: GET /api/collectibles/statistics
│       ├── dto.go                 # extended: the statistics response shapes
│       └── server.go              # extended: one route
└── tests/
    ├── unit/statistics_test.go            # NEW: folding, ordering, totals ceiling
    ├── integration/statistics_test.go     # NEW: real aggregates against real rows
    ├── integration/statistics_money_test.go  # NEW: exactness, deliberately awkward values
    ├── integration/query_count_test.go    # extended: three statements, constant
    └── contract/statistics_test.go        # NEW: the response matches the contract

frontend/
├── app/collection/stats/
│   ├── page.tsx                   # NEW: RSC, forwards cookies, redirects without a session
│   └── loading.tsx                # NEW: FR-038
├── components/collection/
│   ├── CollectionSummary.tsx      # NEW: headline figures
│   ├── StatusBreakdown.tsx        # NEW: four statuses, zeros included
│   ├── GroupBreakdown.tsx         # NEW: series and manufacturer, shared
│   └── CollectionGallery.tsx      # extended: the link to the statistics
├── lib/
│   ├── api/statistics.ts          # NEW
│   └── types/api.ts               # regenerated from the contract
└── tests/
    ├── unit/collection-summary.test.tsx   # NEW
    ├── unit/group-breakdown.test.tsx      # NEW
    └── e2e/statistics.spec.ts             # NEW

specs/001-add-browse-collectibles/contracts/openapi.yaml   # bumped to 0.3.0
```

**Why the contract still lives in feature 001's folder**: moving it is issue #33, assigned and not
yet done. Feature 006 set the precedent of editing it in place and leaving a `contracts/README.md`
pointer in the new feature's folder; doing the same here keeps one contract rather than two, and
keeps the move a single reviewable change when #33 is picked up.

## Phases

**Phase 1 — Exactness, before anything can display a wrong number.** Extend `money.go` with the
totals ceiling and its named error. Unit tests first, including the wrap case, which must be
provoked deliberately because no realistic input reaches it.

**Phase 2 — The aggregates (User Story 1 and 2 together).** The status statement, the service, the
endpoint, the contract. These two stories share one statement, so splitting them would mean building
the same query twice. Integration tests assert exactness against independently assembled totals, and
the query-count test is extended here rather than at the end.

**Phase 3 — The breakdowns (User Story 3).** The two breakdown statements with their window-function
tail folding and `mode()` display spelling, plus the gallery filters that make the groups traceable.
The filter and the grouping must normalise identically; an integration test asserts a group's count
equals what its own link returns.

**Phase 4 — The view.** The route, the three components, the loading and empty states, both
appearances, three viewports, keyboard and screen reader.

**Phase 5 — Verification.** Every suite, every number recorded as measured rather than as hoped,
including the production image size, which this feature will move again (issue #29).

## Complexity Tracking

No constitutional deviation is being requested, and no complexity needs justifying. The one addition
beyond the obvious reading of the spec — gallery filters for series and manufacturer — is required by
FR-030 and FR-031 rather than chosen, and is recorded in research.md Decision 8.
