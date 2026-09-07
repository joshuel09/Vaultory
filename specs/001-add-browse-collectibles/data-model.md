# Phase 1 Data Model: Add & Browse Collectibles

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Date**: 2026-09-07

PostgreSQL is the authoritative store (Principle II). All schema changes arrive as reversible,
version-controlled `golang-migrate` pairs under `backend/migrations/` (Principle IV).

---

## Entities

### `collectors`

The owner of a vault. Present as a real table with a real key from the start, even though
authentication is out of scope, so that ownership is enforceable and nothing needs restructuring
when registration and login arrive (research Decision 1).

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| `id` | `uuid` | PK, default generated | Referenced by everything a collector owns |
| `display_name` | `text` | NOT NULL, length 1–100 | Not surfaced by this feature; present so a seeded collector is identifiable |
| `created_at` | `timestamptz` | NOT NULL, default `now()` | |

A development-only collector is seeded by migration. No columns for credentials, sessions, or
profile exist in this feature — that is the excluded authentication work.

---

### `collectibles`

One entry in one collector's vault. Each added entry is a distinct row; two identical entries remain
two rows and are never merged or collapsed into a quantity (FR-023, FR-024).

| Column | Type | Constraints | Requirement |
|--------|------|-------------|-------------|
| `id` | `uuid` | PK, default generated | |
| `collector_id` | `uuid` | NOT NULL, FK → `collectors(id)` ON DELETE CASCADE | FR-025 |
| `name` | `text` | NOT NULL, CHECK trimmed length between 1 and 200 | FR-002, FR-003 |
| `collection_status` | `text` | NOT NULL, CHECK in (`owned`, `preordered`, `wishlist`, `sold`) | FR-004 |
| `character` | `text` | NULL, length ≤ 200 | FR-006 |
| `series` | `text` | NULL, length ≤ 200 | FR-006 |
| `manufacturer` | `text` | NULL, length ≤ 200 | FR-006 |
| `category` | `text` | NULL, length ≤ 100 | FR-006 |
| `scale` | `text` | NULL, length ≤ 50 | FR-006 |
| `edition` | `text` | NULL, length ≤ 200 | FR-006 |
| `purchase_price` | `numeric(12,2)` | NULL, CHECK ≥ 0 | FR-016, FR-017 |
| `purchase_date` | `date` | NULL, CHECK ≤ `CURRENT_DATE` | FR-018 |
| `release_date` | `date` | NULL, no bound | FR-019 |
| `notes` | `text` | NULL, length ≤ 2000 | FR-006 |
| `image_id` | `uuid` | NULL, composite FK `(image_id, collector_id)` → `collectible_images(id, collector_id)` ON DELETE SET NULL | FR-008, FR-011, FR-012, FR-015 |
| `created_at` | `timestamptz` | NOT NULL, default `now()` | FR-034 ordering |

**Deliberate choices**

- `numeric(12,2)`, never a floating-point type — Principle IV forbids float arithmetic for money.
  Carried across the contract as a decimal string so no JSON number conversion can occur.
- `date`, not `timestamptz`, for `purchase_date` and `release_date`: these are calendar dates without
  a time of day, per the spec's assumptions.
- Status as a `text` column with a `CHECK` constraint rather than a PostgreSQL `enum`. Both enforce
  the four values; a `CHECK` is amended by a plain migration, whereas altering an enum type is more
  constrained. The four values are closed by the constraint, so an unknown status is rejected by the
  database as well as by the domain.
- `NULL` means the collector recorded nothing; it is distinct from `''` and from `0`, which is what
  FR-007 requires. A `purchase_price` of `0.00` is a recorded amount (a gift), not an absence.
- The `name` check is on the *trimmed* length, so a whitespace-only name fails at the database as
  well as in the domain (FR-003).
- `image_id` is nullable and singular, which is FR-011 and FR-012 expressed structurally: a
  collectible cannot reference two images, and needs none.
- The image reference is a **composite** foreign key on `(image_id, collector_id)` rather than on
  `image_id` alone. A single-column key would permit a collectible to reference an image owned by a
  different collector, leaving FR-015 defended only by the domain layer; the composite key makes the
  database refuse it. This is why `collectible_images` carries `UNIQUE (id, collector_id)`, which is
  otherwise redundant against its primary key — it exists to be the target of this reference.
- `ON DELETE SET NULL` on the image reference: should an image record ever be removed, the
  collectible survives and falls back to the designed placeholder (FR-033).

**Indexes**

| Index | Columns | Purpose |
|-------|---------|---------|
| `collectibles_gallery_idx` | `(collector_id, created_at DESC, id DESC)` | The unfiltered gallery page in creation order (FR-034), keyset-paginated (research Decision 7) |
| `collectibles_status_gallery_idx` | `(collector_id, collection_status, created_at DESC, id DESC)` | The same page with a status filter applied (FR-037) |

There is intentionally no unique constraint across name and attributes: duplicates are legitimate
and required (FR-023).

---

### `collectible_images`

One uploaded primary image and its derived gallery rendition. A row exists from the moment of upload,
before any collectible references it, because the upload is a separate operation (research
Decision 3).

| Column | Type | Constraints | Requirement |
|--------|------|-------------|-------------|
| `id` | `uuid` | PK, default generated | |
| `collector_id` | `uuid` | NOT NULL, FK → `collectors(id)` ON DELETE CASCADE | FR-015 |
| — | — | UNIQUE `(id, collector_id)` | Target of the composite reference from `collectibles`, so the database enforces that a collectible and its image share one owner (FR-015) |
| `original_key` | `text` | NOT NULL | Locator of the uploaded bytes in the image store |
| `rendition_key` | `text` | NOT NULL | Locator of the fixed-aspect gallery rendition |
| `content_type` | `text` | NOT NULL, CHECK in (`image/jpeg`, `image/png`, `image/webp`) | FR-009 |
| `byte_size` | `bigint` | NOT NULL, CHECK > 0 AND ≤ 10485760 | FR-010 |
| `width` | `integer` | NOT NULL, CHECK > 0 | Recorded from the decode |
| `height` | `integer` | NOT NULL, CHECK > 0 | Recorded from the decode |
| `created_at` | `timestamptz` | NOT NULL, default `now()` | |

**Deliberate choices**

- `collector_id` is on the image itself, so an image is authorizable on its own request without
  first resolving the collectible that references it — which is what FR-015's per-request check
  needs, including for an image not yet attached to anything.
- `content_type` records what the decode determined, never what the client declared (research
  Decision 4).
- `byte_size` is constrained to the 10 MB limit in the database as well as in the request path, so
  the rule in FR-010 cannot be bypassed by a future code path.
- Bytes live in the image store, not in PostgreSQL. Only locators and metadata are relational.

---

## Relationships

```text
collectors 1 ──── 0..n collectibles          (collectibles.collector_id, NOT NULL)
collectors 1 ──── 0..n collectible_images    (collectible_images.collector_id, NOT NULL)
collectibles 0..1 ──── 0..1 collectible_images (collectibles.(image_id, collector_id), NULL)
                                              same owner enforced by the composite key
```

Every owned row carries its owner directly. Ownership is therefore a predicate available in a single
query with no join required, which is what lets every query be scoped by collector without risk of
loading another collector's row (research Decision 5).

---

## Domain validation rules

Enforced in `internal/domain/collectible` — independent of HTTP and SQL, so it is testable without
either (Principle II). The database constraints above are the second line, not the only one.

| Rule | Requirement |
|------|-------------|
| Name required; leading and trailing whitespace trimmed before the emptiness test | FR-002, FR-003 |
| Name at most 200 characters after trimming | Spec edge case on very long names |
| Collection status required and one of exactly four values | FR-002, FR-004 |
| A name and a status alone constitute a valid collectible | FR-005 |
| Optional text attributes trimmed; a value that is empty after trimming is stored as absent | FR-007 |
| Notes at most 2000 characters | Spec edge case on very long notes |
| Purchase price, if present, parsed as an exact decimal with at most two fractional digits; rejected if negative; `0` accepted as recorded | FR-016, FR-017 |
| Purchase price rejected if it exceeds the representable range, rather than silently rounded | Spec edge case on excess precision and very large amounts |
| Purchase date, if present, not later than today. The domain performs this check against the collector's own current date; the database `CHECK` against `CURRENT_DATE` is a coarse backstop and is evaluated in the database session's timezone, so it is deliberately the looser of the two | FR-018 |
| Release date, if present, accepted in the past or the future, in any combination with status and purchase date | FR-019 |
| Image reference, if present, must name an image owned by the acting collector | FR-015 |
| All violations collected and returned together, never one at a time | FR-020 |
| Text values preserved exactly as entered, including non-Latin scripts, accents, and emoji | Spec edge case |

Image validation, in `internal/imaging`:

| Rule | Requirement |
|------|-------------|
| Reading stops at 10 MB; a larger upload is refused with the limit stated | FR-010 |
| Format determined by decoding, not by declared content type or file extension | FR-009, spec edge case |
| A file that fails to decode, or decodes to an unsupported format, is refused as an unsupported image | Spec edge case |
| EXIF orientation applied before the rendition is derived | Spec edge case on presentation |
| Rendition derived at one fixed aspect ratio and bounded dimensions for every image, whatever its original proportions | FR-014, SC-013 |
| A refused image leaves no stored bytes and no image row | FR-013 |

---

## State transitions

`collection_status` is set when a collectible is added and cannot change within this feature, because
editing is out of scope. There are consequently no status transitions to model, and no history table.
Status is read-only after creation, and the four values are peers rather than stages — a collectible
may be added directly as `sold` without ever having been `owned`.

When editing arrives, transitions become a real concern (a preorder becoming owned, an owned item
becoming sold) and will likely want an audit trail. Nothing in this schema prevents adding one: the
status column stays, and a history table references `collectibles(id)`.

---

## Scale notes

- Design target: tens of thousands of collectibles per collector; pages of 24 to 48 entries.
- The two composite indexes cover both gallery queries, filtered and unfiltered, in creation order.
  No query in this feature sorts or filters on a column outside those indexes.
- Gallery listing reads only the columns the cards render; it does not fetch `notes` and performs no
  per-entry image lookup — the rendition locator arrives through a single `LEFT JOIN` on
  `collectible_images` in the same statement. One statement serves one page, so there is no N+1
  access.
- A gallery page load therefore costs **two** indexed statements, not one: the page itself, and the
  `totalUnfiltered` count that lets the frontend distinguish an empty vault (FR-041) from a filter
  matching nothing (FR-040). The count is covered by `collectibles_gallery_idx` and is constant
  work per page load regardless of collection size; it is not per-entry work.
