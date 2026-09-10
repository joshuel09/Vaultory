# Phase 0 Research: Add & Browse Collectibles

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Date**: 2026-09-07

The Technical Context carried no unresolved `NEEDS CLARIFICATION` items: the constitution fixes the
stack, and the one open product question — how a collector supplies the primary image — was
answered and recorded in the spec's Clarifications section on 2026-09-06. What remained were design
decisions where the constitution constrains the outcome without dictating it. Each is settled below.

---

## Decision 1 — Collector identity is resolved behind a single server-side seam

**Decision**: Add `internal/identity` to the Go service, exposing one operation that resolves the
acting collector from an incoming request and returns either a collector identifier or a failure.
Every handler obtains the acting collector from this seam and from nowhere else. For this feature
the implementation is development-only: a signed, HTTP-only cookie issued by a development endpoint
that maps to a collector row seeded by migration. Real registration and login replace this one
package later.

**Rationale**: The spec excludes authentication implementation but requires collector-private data
(FR-025 through FR-029) and forbids trusting any client claim of identity (FR-028). Those
requirements cannot be satisfied — or tested — unless something server-side answers "who is
acting". Confining that to one package means the privacy rules are genuinely enforced now, the
ownership tests are real tests rather than placeholders, and the excluded work has exactly one
future replacement point. The `collectors` table is a real table with a real foreign key from
`collectibles`, so no data restructuring is needed when authentication arrives.

**Alternatives considered**:

- *Accept a collector identifier from a request header or query parameter.* Rejected outright: it
  contradicts FR-028 and Principle IV's prohibition on accepting client-provided ownership values.
  It would also make every ownership test vacuous, since any client could assume any identity.
- *Hard-code a single global collector and omit the notion of ownership entirely.* Rejected: FR-026
  and FR-027 would be unimplemented and untestable, and cross-collector isolation is the hardest
  thing to retrofit safely. Building it now costs a foreign key and a WHERE clause.
- *Implement full authentication anyway.* Rejected: explicitly out of scope, and Principle V forbids
  adding functionality beyond the approved scope.

**Consequence recorded**: This is the sole entry in the plan's Complexity Tracking table.

---

## Decision 2 — Next.js proxies `/api/*` to Go so the session travels same-origin

**Decision**: Configure a rewrite in `next.config.ts` mapping `/api/*` to the Go service's address.
The browser therefore talks only to the Next.js origin, and the session cookie accompanies every
request — including the `<img>` requests that fetch collectible images. Collectible images are
rendered with a plain `<img>` (or `next/image` with `unoptimized`), pointing at the authenticated
image path, because Go already produces a correctly sized rendition.

**Rationale**: FR-015 requires an authorization check on *every* image request. A browser fetching
an image from a different origin does not attach the session cookie, and Next.js's image optimizer
fetches the source server-side without the collector's credentials — either arrangement would
either break image loading or force images to be served unauthenticated, violating FR-015. A
rewrite solves it with configuration and no code, keeps Next.js strictly presentational as
Principle II requires, and avoids cross-origin credential handling entirely.

**Alternatives considered**:

- *Serve images from unguessable public addresses with no per-request check.* Rejected: the
  clarification of 2026-09-06 chose owner-only access explicitly. An address that leaks through a
  shared screenshot, browser history, or a referrer header would expose that image permanently.
- *A Next.js route handler that forwards each request and re-attaches cookies.* Workable, but it is
  hand-written plumbing in the frontend for something a rewrite does declaratively, and every added
  line in Next.js is a line where a business rule could later be tempted to settle.
- *Cross-origin requests with permissive credentials.* Rejected: more configuration surface, more
  ways to get wrong, and `<img>` still would not send credentials cross-origin without further
  contortions.

**Note for implementation**: the rewrite proxy has its own request body limit; it must be raised
above 10 MB, or the upload will be refused before Go ever applies the rule in FR-010.

---

## Decision 3 — Upload is a separate operation from creating the collectible

**Decision**: Two operations. `POST /api/images` accepts one `multipart/form-data` file, validates
it, derives the gallery rendition, stores both, and returns an image identifier. `POST
/api/collectibles` accepts JSON that may carry that identifier. Client-side checks may pre-empt
obvious problems for a faster response, but the server re-validates everything; no client check is
load-bearing.

**Rationale**: This maps directly onto FR-013 — a rejected image must not block saving the rest of
the collectible. With the upload separated, an oversized file fails on its own and the collector
proceeds with a different image or none, all without losing the form values FR-022 requires be
preserved. It also keeps the collectible operation pure JSON, which keeps the OpenAPI contract
clean and lets the frontend's generated types cover it completely.

**Alternatives considered**:

- *A single multipart request carrying both the file and the fields.* Rejected: a rejected image
  fails the whole submission, which is precisely the coupling FR-013 forbids, and it makes the
  collectible request awkward to describe and to type.
- *Base64-encoding the image inside the JSON body.* Rejected: inflates a 10 MB upload by roughly a
  third, forces the whole payload through JSON parsing, and gives worse validation behaviour.

**Accepted cost**: an upload can be abandoned before its collectible is saved, leaving an
unreferenced image. Images are therefore recorded with an owning collector from the moment of
upload, and reclaiming unreferenced ones is noted as deferred work rather than built now (YAGNI,
Principle V).

---

## Decision 4 — Go validates images by decoding them and derives a fixed-aspect rendition

**Decision**: On upload, Go enforces the byte limit while reading, decodes the image to confirm it
genuinely is JPEG, PNG, or WebP, applies any EXIF orientation, and derives a gallery rendition at
**4:5 portrait, 800×1000 pixels** — filling the frame and trimming overflow centrally, never
letterboxing and never distorting. Both the original and the rendition are stored.
The gallery is served renditions; the original is retained as the collector's asset. Renditions are
encoded as JPEG.

**Rationale**: FR-009 and FR-010 must be enforced against what was actually uploaded, not what the
client claimed — Principle III requires validating external input at the boundary, and the spec's
edge cases explicitly include a file with an image extension whose content is not an image. FR-014
requires consistent framing without collector-side cropping, which means the system, not the
collector, decides presentation geometry; deriving it once at upload also satisfies the technology
constraint that images be optimized, and keeps a grid of entries from shipping many megabytes.
EXIF orientation matters concretely here: phone photographs of figures otherwise appear rotated.

**Why 4:5 at 800×1000**: figures, statues, and boxed collectibles are predominantly taller than
wide, so a portrait frame wastes the least of the card on empty space; a square frame would trim the
top of a tall statue or strand it in margins. 800×1000 covers a roughly 400-pixel-wide card at
double density, which is the largest the gallery grid uses at any supported width.

**Alternatives considered**:

- *Trust the declared content type or the file extension.* Rejected: trivially spoofed, and
  contradicted by an explicit spec edge case.
- *A square rendition.* Rejected: it either trims tall collectibles badly or leaves large margins,
  and collectible photography is mostly portrait.
- *Preserve each image's own proportions.* Rejected outright by FR-014 — the gallery must frame
  every entry identically without the collector cropping anything.
- *Store only the original and size it in the browser with CSS.* Rejected: fails the optimization
  constraint and SC-004 — a 500-entry gallery would transfer originals — and offers no defence
  against extreme aspect ratios.
- *Generate renditions lazily on first request.* Rejected: puts variable latency on the browsing
  path the performance goals govern, and complicates the authorization path for no present benefit.
- *Encode renditions as WebP.* Rejected for now: Go's supported packages decode WebP but do not
  encode it, so this would require a cgo-backed dependency. JPEG renditions are adequate and
  dependency-free; revisit only if measurement justifies it (Principle V).

---

## Decision 5 — Absence of another collector's resource is indistinguishable from its non-existence

**Decision**: A request for a collectible or an image that exists but belongs to another collector
returns the same `404` as one that does not exist at all. A request whose acting collector cannot be
resolved returns `401` and no collection content. Ownership is expressed as a predicate inside every
query rather than as a check performed after loading a row.

**Rationale**: FR-027 requires refusal without revealing whether the resource exists, so `403`
cannot be used — it confirms existence. FR-029 requires that no content be shown when identity is
unresolvable. Putting ownership in the query itself means there is no code path that has the wrong
collector's row in hand, which is a stronger guarantee than remembering to check.

**Alternatives considered**:

- *`403 Forbidden` for another collector's resource.* Rejected: leaks existence, contrary to FR-027.
- *Load the row, then compare owners in the handler.* Rejected: correct only as long as every
  handler remembers, and Principle IV asks for security-sensitive operations that fail safely by
  construction.

---

## Decision 6 — One `collectibles` table now, with room for a catalog later

**Decision**: Model each entry as one row in a single `collectibles` table owned by a collector, with
descriptive attributes stored as free text on the row. No shared catalog of collectible products,
and no separate notion of an owned instance, is built in this feature.

**Rationale**: The spec records free-text attributes with no controlled vocabulary, and FR-023
requires that two identical entries stay independent and never be merged or expressed as a
quantity — which a single-row-per-entry model gives directly. The constitution asks that the
architecture *be capable* of distinguishing a catalog-level collectible from a collector's owned
instance when that becomes necessary; it does not ask for it now. That future step is additive: a
nullable reference from `collectibles` to a new catalog table, with existing free-text values
retained. Building the split now would add a table, a matching problem, and a deduplication policy
in service of no requirement in this spec.

**Alternatives considered**:

- *Catalog plus instance tables from the start.* Rejected as speculative under Principle V, and it
  raises questions the spec deliberately leaves out — who curates the catalog, how variants are
  identified, what happens to an entry whose catalog match is wrong.
- *A JSON column for the optional attributes.* Rejected: the constitution requires proper relational
  modelling where relationships and constraints are meaningful, and these attributes are exactly the
  columns a future filter or facet will need.

---

## Decision 7 — Keyset pagination ordered by creation, newest first

**Decision**: The listing operation returns a page of entries ordered by `created_at` descending with
`id` as a tiebreaker, and returns an opaque cursor for the next page. The status filter is applied
in the same query. A composite index supports the ordering per collector and per status.

**Rationale**: FR-034 fixes the default order as most-recently-added first, and FR-035 requires that
large collections not be presented all at once. Keyset pagination keeps page cost constant as a
collection grows, where offset pagination degrades; the `id` tiebreaker makes the order total, so
entries added in the same instant cannot be skipped or repeated across pages. Filtering inside the
query keeps the listing to one round trip, satisfying the constraint against N+1 access.

**Alternatives considered**:

- *Offset and limit.* Simpler, but page cost grows with depth and concurrent inserts shift rows
  between pages — visible as duplicated or skipped entries while scrolling.
- *Return the whole collection and page in the browser.* Rejected: contradicts FR-035 and SC-004
  outright.

---

## Decision 8 — Dependencies, each justified

Principle V requires that no dependency be introduced without clear project value, and the
technology constraints favour Go's standard library over frameworks.

| Dependency | Purpose | Why not the standard library alone |
|------------|---------|------------------------------------|
| `net/http` (stdlib) | Routing and serving | No web framework needed; current stdlib routing patterns cover these four operations |
| `pgx` | PostgreSQL access | The maintained PostgreSQL driver; gives exact numeric handling, which `NUMERIC` money requires |
| `golang-migrate` | Schema migrations | Principle IV requires version-controlled, reproducible, reversible migrations; hand-rolling this is real work with no upside |
| `golang.org/x/image` | WebP decoding | An accepted format the standard library's `image` package cannot decode. Chosen over a cgo-backed library because decoding is all that is required |
| Next.js, Tailwind CSS, shadcn/ui | Frontend foundation | Mandated by the constitution's technology constraints |
| `openapi-typescript` | Generate TS types from the contract | Principle III requires explicit contracts and strict typing without `any`; generated types make the contract the single source of truth instead of hand-copied shapes |
| Vitest, Testing Library, Playwright | Frontend and journey tests | Principle V requires automated tests for important rules and end-to-end coverage of important journeys |

Deliberately not adopted: a Go web framework, an ORM, a decimal library for money (`pgx` plus
`NUMERIC` suffices, with amounts carried as strings across the contract to avoid any float
conversion), a state-management library, and a component library beyond shadcn/ui.

---

## Decision 9 — Dark is the default appearance, light is fully supported

**Decision**: Both appearances are built from one set of themed tokens, dark is the default, and the
collector's system preference is honoured. No manual toggle in this feature.

**Rationale**: The constitution requires dark mode be a first-class experience, and this spec did not
mention appearance at all until the 2026-09-10 clarification — a gap `/speckit-analyze` raised as a
constitution violation. Dark suits a cinematic gallery and collectible photography reads better
against dark surfaces, which is why dark is the default rather than merely available. Supporting
light from the start costs a second palette on the same tokens; discovering it is needed after every
surface is styled costs a revisit of all of them.

**Alternatives considered**:

- *Dark only.* Rejected: a collector in a bright room has no recourse, and adding light later touches
  every component.
- *Light default with dark available.* Rejected: it inverts the constitution's emphasis.
- *A persisted manual toggle now.* Deferred, not rejected — additive once both palettes exist, and no
  requirement asks for it yet (Principle V).

---

## Decision 10 — A client-supplied submission key makes retries idempotent

**Decision**: The add-collectible request carries a submission key the client generates per attempt.
The server records `(collector_id, submission_key)` in the same transaction as the collectible; a
repeat within a bounded window returns the collectible already created. A different key always
creates an independent collectible.

**Rationale**: The spec's edge cases required that a double submit or a post-failure retry produce
only the collectible the collector intended, while FR-023 requires that identical collectibles stay
independent — a direct contradiction `/speckit-analyze` flagged. A per-attempt key separates the two
cases exactly: a retry replays a key, a deliberate second copy brings a new one. It matters more than
it might seem, because deleting is out of scope, so an accidental duplicate would be permanent for
the collector.

The uniqueness constraint on the key, not an application-level lookup, is what holds under a genuine
concurrent double submit — two simultaneous requests cannot both pass a check-then-insert.

**Alternatives considered**:

- *Deduplicate by comparing field values.* Rejected: it directly violates FR-023 and would silently
  swallow a collector's legitimate second copy.
- *Disable the button after the first click.* Necessary but insufficient — it does nothing for a
  retry after a lost response, and nothing for a client that never received the reply.
- *A server-issued token fetched before the form is submitted.* Rejected: an extra round trip for the
  same guarantee a client-generated key provides.

---

## Deferred, with reasons

- **Reclaiming unreferenced uploads** (from Decision 3) — needs a retention policy nobody has asked
  for. Uploads carry an owner, so nothing is orphaned in the privacy sense.
- **WebP renditions** (Decision 4) — revisit only if measurement shows JPEG renditions are the
  bottleneck.
- **Catalog and owned-instance split** (Decision 6) — additive when a requirement calls for it.
- **Currency selection** — the spec assumes a single currency; a `currency` column is a later
  additive change.
- **A persisted appearance toggle** (Decision 9) — additive once both palettes exist.
- **Reclaiming expired submission keys** (Decision 10) — rows outside the window are reclaimable;
  nothing depends on retaining them.
- **Rate limiting on upload** — no requirement in this spec. Worth revisiting before the feature is
  exposed publicly, since decoding a 10 MB image is the most expensive operation here.
