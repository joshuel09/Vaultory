# Data Model: Collection Value & Statistics

Nothing here is stored. Every shape below is derived on read from rows features 001 and 006 already
write, and exists only for the duration of a request.

## What it reads

From `collectibles`, for one `collector_id`:

| Column | Type | Read for |
|---|---|---|
| `collection_status` | `text`, constrained to `owned`, `preordered`, `wishlist`, `sold` | the status split |
| `purchase_price` | `numeric(12,2)`, **nullable** | every monetary total |
| `series` | `text`, nullable | the series breakdown |
| `manufacturer` | `text`, nullable | the manufacturer breakdown |

The nullability of `purchase_price` is the one that shapes the feature: a collectible without a price
is not a collectible that cost nothing. It is excluded from totals and counted separately, so that a
total can always be read against the number of pieces it covers (FR-009, FR-010).

## The shapes

### CollectionStatistics

The whole response.

| Field | Meaning |
|---|---|
| `totalCount` | every collectible in the vault |
| `pricedCount` | how many carry a recorded purchase price |
| `byStatus` | exactly four entries, one per status, always present |
| `bySeries` | bounded breakdown, largest first |
| `byManufacturer` | bounded breakdown, largest first |

There is deliberately **no `headlineValue` field**. The API returns four honest status figures; which
one a collector sees largest is a presentation decision, and putting it in the contract would make a
change of emphasis a contract change (research.md Decision 7).

### StatusSummary

| Field | Meaning |
|---|---|
| `status` | one of the four |
| `count` | collectibles with this status |
| `pricedCount` | how many of those carry a price |
| `total` | exact decimal string; the sum of those prices |

Always four of these, including statuses holding nothing, which report zeros rather than being
omitted (FR-015). A status with no priced collectibles reports `"0.00"`, which is a true statement
about a sum of nothing — and `pricedCount` is what distinguishes it from a genuine zero.

### GroupSummary

| Field | Meaning |
|---|---|
| `label` | what to display |
| `kind` | `named`, `unspecified`, or `remainder` |
| `count` | collectibles in the group |
| `pricedCount` | how many carry a price |
| `total` | exact decimal string |

`kind` exists so the presentation never has to infer a group's nature from its label. An
`unspecified` group whose label happened to be the literal text "Unspecified" would otherwise be
indistinguishable from a collector who typed that as a manufacturer.

## The statements

Three, constant, none growing with the collection (FR-006, SC-004).

### 1 — The status split, from which the headline is derived

```sql
SELECT collection_status,
       count(*)                            AS n,
       count(purchase_price)               AS priced,
       coalesce(sum(purchase_price), 0)    AS total
FROM collectibles
WHERE collector_id = $1
GROUP BY collection_status
```

At most four rows. Go fills in the statuses PostgreSQL did not return, because a status holding
nothing produces no row and FR-015 requires it to appear anyway. `count(purchase_price)` counts
non-null values, which is exactly `pricedCount` and costs nothing extra.

The whole-vault figures in Story 1 are summed from these four rows in Go. That is the business layer
doing arithmetic, which is where it belongs; FR-001 forbids the *presentation* layer doing it.

### 2 and 3 — The breakdowns

Identical but for the column. Shown for `series`:

```sql
WITH grouped AS (
    SELECT lower(btrim(coalesce(series, '')))                 AS key,
           mode() WITHIN GROUP (ORDER BY btrim(series))       AS display,
           count(*)                                           AS n,
           count(purchase_price)                              AS priced,
           coalesce(sum(purchase_price), 0)                   AS total
    FROM collectibles
    WHERE collector_id = $1
    GROUP BY 1
),
ranked AS (
    SELECT *, row_number() OVER (ORDER BY n DESC, key ASC) AS rn
    FROM grouped
)
SELECT CASE WHEN rn <= $2 THEN key     ELSE NULL END AS key,
       CASE WHEN rn <= $2 THEN display ELSE NULL END AS display,
       sum(n), sum(priced), sum(total)
FROM ranked
GROUP BY 1, 2
ORDER BY (key IS NULL),      -- the remainder last
         (key = ''),         -- the unspecified group next-to-last
         sum(n) DESC,
         key ASC
```

The `ORDER BY` is part of the statement rather than something Go applies afterwards, so there is one
place the ordering is defined and T047 and T049 cannot re-decide it. The first two keys are what
FR-022 requires: both the unspecified group and the remainder sort below every named group whatever
their own size.

Three things are load-bearing:

- **`coalesce` before normalising, not after.** `NULL`, `''` and `'   '` must become one group. Doing
  it the other way leaves two differently-named unspecified buckets, which is visible to a collector
  as a defect (research.md Decision 6).
- **`mode() WITHIN GROUP`** returns the most frequent original spelling. Where two spellings are
  equally common it returns the first in the given sort order — which, because `btrim` preserves
  case, means `Good Smile Company` beats `good smile company` alphabetically. That is deterministic
  and repeatable, and FR-027a says so in those terms rather than claiming the result is the one the
  collector preferred: a tie does not record a preference. `mode()` is an ordered-set aggregate and
  takes a single sort expression, so a tiebreak column cannot be added to it.
- **The outer `GROUP BY` folds the tail in SQL.** Everything past the bound collapses to one row with
  `NULL` keys, so the statement returns at most bound + 2 rows no matter how many distinct series a
  vault holds. Ranking in SQL and folding in Go would be simpler and would transfer eight hundred
  rows to display eight (research.md Decision 5).

`row_number()` rather than `rank()`: ties must not produce more rows than the bound allows.

### Verified against PostgreSQL 17, not assumed

The statement above was run against the real database before this plan was committed. Three things
it confirmed, and one it corrected:

- `sum` of `0.10`, `0.20`, `0.30` returned exactly `0.60`. `numeric` addition is exact.
- `mode() WITHIN GROUP` returned `Good Smile Company` for a group holding that spelling twice and
  `good smile company` once — the collector's own most-used wording, as FR-027a requires.
- **The unspecified group and the remainder group do not collide**, which is not obvious. The
  unspecified group's key is the empty string `''`; the remainder's is `NULL`. The outer `GROUP BY`
  keeps them apart, and `IS NULL` distinguishes them on read. A `psql` session renders both as blank,
  which makes this look like a collision when it is not — worth knowing before someone "fixes" it.
- **Corrected**: `ORDER BY n DESC` alone puts the remainder wherever its count lands, which in a test
  vault placed *Everything else* above the unspecified group. Both must sort below every named group
  regardless of count. The ordering is therefore by group kind first — named, then unspecified, then
  remainder — and by count within that, which is what FR-022 means by "a stated and stable measure".

## One transaction, one moment

The three statements run inside a single read-only transaction (research.md Decision 10). Three
independent reads observe three independent snapshots, and a collectible added between two of them is
counted by one and not the other — which breaks FR-017 and FR-026 silently, and looks like a bug in
the arithmetic rather than in the sequencing.

Note for anyone extending `query_count_test.go`: its `statementCount` helper reads
`xact_commit + xact_rollback`, so it counts **transactions**, not statements, despite the name. The
property worth asserting is that the cost does not move between a small vault and a large one.

## Exactness

The rule, stated once: **`total` is scanned from PostgreSQL as a string and parsed into exact minor
units. No `float64` appears between the database and the browser.**

- `numeric` arithmetic in PostgreSQL is exact, not floating point, so summing in the database loses
  nothing.
- The scan target is `string`. A `float64` scan target compiles, runs, passes small tests, and is
  wrong by a cent on a real vault.
- The wire format is a decimal string, matching how single amounts already travel. A JSON number
  invites a float on the other side.

### The totals ceiling

`maxMoneyUnits` is `9_999_999_999_99`, which is what `numeric(12,2)` holds. That bound belongs to a
*stored* amount. A total is derived, is never stored, and is bounded instead by the representation:
int64 minor units reach roughly 9.2 × 10¹⁶ currency units.

`money.go` gains a parse that takes the maximum explicitly. `ParseMoney` keeps the column's bound;
totals use the representation's bound and return a **named error** when it is exceeded.

The guard will never fire for a real vault — it needs millions of maximum-value pieces. It exists
because the alternative failure is silent: an unguarded accumulation wraps to a negative total, and a
negative total is not obviously a bug to anyone reading it. A test provokes it deliberately, because
nothing else will.

## What this model deliberately does not contain

- **Market value, appraisal, sale price, profit.** None is recorded anywhere in the product. FR-042
  forbids deriving a figure from a field that does not exist, and the Sold figure is therefore
  labelled as what was *paid* — wording that is a requirement, not copy.
- **Any stored summary.** No table, no trigger, no cache. The figures are exact because they are
  computed from the rows every time (research.md Decision 1, Decision 9).
- **Anything time-series.** Nothing records when a price changed, so no honest trend can be drawn.
