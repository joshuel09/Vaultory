# Research: Search and Sort a Collection

## 1. Words, substrings, one predicate

- **Decision**: Go splits the search on whitespace (at most 20 words, at most 200 characters), escapes
  `\`, `%` and `_` in each word, and passes them as one `text[]`. SQL requires that no word fails to
  match:
  `NOT EXISTS (SELECT 1 FROM unnest($n::text[]) w WHERE NOT (<doc> ILIKE '%' || w || '%'))`, where
  `<doc>` is the eight searchable attributes joined with a newline.
- **Rationale**: every word must appear somewhere (FR-002). Joining with a newline means a match cannot
  straddle two attributes, because a word contains no whitespace. Escaping in Go and binding as a
  parameter makes every character literal (FR-003) and leaves no injection surface.
- **Alternatives considered**: PostgreSQL full-text search (stems and tokenises: "1/4" and part-words
  would not match, which contradicts the substring semantics the spec asks for); one ILIKE per word
  built into the SQL text (works, but varies the statement with the input).

## 2. A sort-aware keyset cursor

- **Decision**: ten orders, each a whitelisted pair of key expression and direction:

  | Order | Key |
  |---|---|
  | `added_desc` (default), `added_asc` | `created_at` |
  | `name_asc`, `name_desc` | `lower(name)` |
  | `price_desc`, `price_asc` | `purchase_price` |
  | `purchased_desc`, `purchased_asc` | `purchase_date` |
  | `released_desc`, `released_asc` | `release_date` |

  `ORDER BY (k IS NULL), k <dir>, id <dir>`. The cursor encodes `order | missing | value | id`. The
  next-page predicate is:
  - after a present value: `(k IS NOT NULL AND (k <op> v OR (k = v AND id <op> c))) OR k IS NULL`;
  - after a missing value: `k IS NULL AND id <op> c`.

  Here `<op>` is `>` for ascending and `<` for descending.
- **Rationale**: one row comparison cannot express "missing values last" in both directions, so the
  predicate is spelled out. `id` in the same direction as the key makes the order total (FR-008). A
  cursor whose order differs from the request's is refused (FR-012).
- **Compatibility**: the old two-part cursor (`nanos|id`) is still read, as `added_desc`, so a page
  opened before deploy can keep scrolling.

## 3. No index, no extension — yet

- **Decision**: no migration. Matching and sorting run over one collector's rows, which the existing
  `collector_id` indexes already narrow to.
- **Rationale**: vaults are hundreds to low thousands of rows. A trigram index (`pg_trgm`) would help
  only across collectors, which search never does. SC-005 is checked by timing the 500-row
  integration fixture. If a vault ever grows past what that shows is comfortable, a trigram index is
  additive.
- **Alternatives considered**: `pg_trgm` GIN index now (an extension and an index carried for a load
  nobody has); per-order composite indexes (ten indexes for sorts that run over a few hundred rows).

## 4. Name order ignores case

- **Decision**: the key is `lower(name)`, in the database's collation, in both `ORDER BY` and the
  cursor predicate.
- **Rationale**: FR-009. Using the same expression in both places is what keeps paging exact;
  lowering in Go would be a second implementation that need not agree.

## 5. The controls

- **Decision**: one form above the gallery: a search field (`type="search"`, submitted on Enter or a
  button) and a sort `<select>` that applies on change. Both write the URL, keeping every other
  parameter; the status filter already keeps parameters it does not own.
- **Rationale**: FR-011 and FR-015. Native elements give keyboard operation and labels for free, and
  the URL is the single source of truth, as it already is for filters.
