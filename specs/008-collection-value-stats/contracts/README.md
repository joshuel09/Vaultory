# Contracts

This feature's API contract lives in the one OpenAPI document the project has:

**[`specs/001-add-browse-collectibles/contracts/openapi.yaml`](../../001-add-browse-collectibles/contracts/openapi.yaml)**

Feature 008 bumps it to **0.3.0** and adds:

- `GET /api/collectibles/statistics` → `CollectionStatistics`
- Schemas: `CollectionStatistics`, `StatusSummary`, `GroupSummary`
- Two optional query parameters on `GET /api/collectibles` — `series` and `manufacturer` — so that a
  figure can link to the collectibles behind it (FR-030, FR-031)

## Why it is not in this folder

The document describes one API, not one feature's slice of it, and splitting it per feature would
mean the frontend generating types from several files that each claim to describe the same server.

It sits under feature 001 for the historical reason that feature 001 created it. Moving it somewhere
neutral is tracked as **issue #33**, which is assigned and not yet done. Feature 006 set this
precedent — edit in place, leave a pointer — and following it keeps the eventual move a single
reviewable change rather than a scavenger hunt.

## Regenerating the frontend types

```bash
cd frontend && npm run generate:types
```

Hand-writing a response shape in TypeScript is forbidden by the constitution: the contract is the
source of truth for the frontend/backend seam, and a hand-written type is a second source that drifts
silently.
