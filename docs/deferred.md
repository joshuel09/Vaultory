# Deferred work

Decisions taken during feature 001 that were deliberately left for later, recorded here so none is
quietly forgotten. Each was deferred under the constitution's simplicity principle: no requirement
asks for it yet.

Source: `specs/001-add-browse-collectibles/research.md`, section "Deferred, with reasons".

## From the design

| Item | Why deferred | What would trigger it |
|------|--------------|-----------------------|
| **Reclaiming unreferenced uploads** | An upload can be abandoned before its collectible is saved, leaving an image row nothing points at. Every image carries an owner, so nothing is orphaned in the privacy sense, and a retention policy nobody has asked for would be guesswork | Storage cost becoming material, or a retention requirement arriving |
| **Reclaiming expired submission keys** | Rows outside the idempotency window serve no purpose, but nothing depends on removing them | The table growing enough to matter |
| **WebP renditions** | Go's supported packages decode WebP but cannot encode it, so this needs a cgo-backed dependency. JPEG renditions are adequate and dependency-free | Measurement showing rendition size is a real bottleneck |
| **Catalog and owned-instance split** | The constitution asks the architecture be *capable* of distinguishing a catalog-level collectible from a collector's owned instance; it does not ask for it now. The step is additive — a nullable reference from `collectibles` to a new catalog table | A requirement for shared metadata, matching, or cross-collector facts about a product |
| **Currency selection** | Prices are recorded in one currency for all collectors. Adding a `currency` column is additive | Collectors buying in more than one currency |
| **Rate limiting on upload** | No requirement in this spec. Decoding a 10 MB image is the most expensive thing the service does | Before the feature is exposed publicly |
| **A persisted appearance toggle** | Dark is the default and light follows the system preference. A manual override is additive once both palettes exist | A collector asking to override their system setting |

## From the specification

| Item | Note |
|------|------|
| **Editing collectibles** | Out of scope for feature 001. Its absence has a real consequence: a collector who mistypes a name or attaches the wrong photograph cannot correct it. This is the most likely thing to want next |
| **Deleting collectibles** | Out of scope. Combined with the above, a mistaken entry is permanent — which is why submission idempotency (FR-047) was worth building now |
| **Authentication** | Out of scope. `internal/identity` is the single seam it replaces; the `collectors` table, its foreign keys, and every ownership-scoped query already exist |
| **Status transitions and history** | Status is fixed at creation because editing is out of scope, so there are no transitions to model yet. When editing arrives, a preorder becoming owned will likely want an audit trail |

## Raised by `/speckit-analyze`, accepted as low impact

| Finding | Status |
|---------|--------|
| **SC-003 and SC-007** state usability targets (95% first-attempt success; 9 of 10 first impressions read it as a gallery) that no planned test can verify | Reframe as post-launch metrics, or add a usability check |
| **Scale mismatch** — `plan.md` designs for tens of thousands of collectibles per collector while SC-004 verifies 500 | Either measure at the designed scale or lower the stated target |
| **US2 independence** — the spec calls User Story 2 independently testable, but the list operation lands in User Story 1 | Wording only; `tasks.md` records the real dependency |
