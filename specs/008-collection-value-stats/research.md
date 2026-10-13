# Research: Collection Value & Statistics

Decisions taken before the plan, each with what was rejected and why. Where a decision rests on
something read in the codebase rather than assumed, the file is named.

---

## Decision 1 — The whole feature needs no schema change

**Decision**: No migration. Not a column, not an index.

`backend/migrations/000003_create_collectibles.up.sql` already holds everything this feature reads:
`collection_status` (constrained to exactly `owned`, `preordered`, `wishlist`, `sold`),
`purchase_price numeric(12,2)` nullable, and `series` / `manufacturer` as nullable text.

**Why no index either.** The obvious instinct is an index on the normalised grouping expression.
It would buy nothing: every figure here aggregates a collector's *entire* vault, so the query reads
all of that collector's rows whatever happens. The index that matters is the one on `collector_id`,
which feature 001 already created. An index on `lower(btrim(series))` would add write cost to every
collectible insert and edit in exchange for no read benefit.

**Rejected**: a materialised summary table kept in step by triggers. It converts an exact,
always-correct read into a cache that can drift, and it makes every write in the product more
expensive to speed up a page nobody opens on every visit. Revisit only if SC-005 is ever missed.

---

## Decision 2 — `numeric` sums are scanned as text, never as a float

**Decision**: `SUM(purchase_price)` is scanned into a Go `string` and parsed into the exact
minor-unit representation. No `float64` appears anywhere in the path.

This is the single likeliest way this feature breaks Principle IV, and it is close to invisible: a
total scanned into a float is correct for small, round test values and wrong by a cent on a real
vault. `backend/internal/domain/collectible/money.go` is explicit that `Money` has no float in its
API precisely so that this cannot be done by accident — but a `pgx` scan into `float64` bypasses the
type entirely, which is why it needs saying here rather than being left to the type to enforce.

**Why sum in SQL at all**, rather than in Go: summing in Go means fetching every row, which is
unbounded work and defeats Decision 4. PostgreSQL's `numeric` addition is exact — it is not
floating point — so the sum is as exact in the database as it would be in the domain.

---

## Decision 3 — Totals get a higher ceiling than a single amount, and guard it explicitly

**Decision**: Extend `money.go` with a parse that takes an explicit maximum, keep `ParseMoney`
capped where it is, and add a totals parse with a far higher ceiling that returns a *named* error
when exceeded.

`maxMoneyUnits` is `9_999_999_999_99` — ten integer digits, matching what `numeric(12,2)` holds.
That bound belongs to a single *stored* amount. A total is derived and is not stored, so it has no
reason to inherit a column's limit; what it must respect is the limit of the representation, and
int64 minor units reach roughly 9.2 × 10¹⁶ currency units — about seven orders of magnitude above a
single amount. A vault would need millions of maximum-value pieces to approach it.

**Why guard a ceiling that cannot realistically be reached**: because the failure mode is silent.
An unguarded accumulation that wraps produces a *negative* total, which is not obviously a bug to
anyone reading it, and which no test will produce by accident. The guard costs one comparison and
converts an invisible wrong answer into a loud one. This is what FR-013 asks for.

**Rejected**: widening totals to a big-decimal type. It introduces a second money representation
into a codebase whose central money claim is that there is exactly one, and it buys headroom beyond
any vault that will ever exist.

---

## Decision 4 — Three statements, each aggregating, none growing with the collection

**Decision**: the complete view costs three statements — one for the status split (from which the
headline figures are derived), one for the series breakdown, one for the manufacturer breakdown.

The status statement is a single `GROUP BY collection_status` returning at most four rows. Every
headline figure in Story 1 is derivable from those four rows, so Story 1 and Story 2 share one
statement rather than issuing one each. Go folds four rows into the headline; that is the business
layer doing arithmetic, which is where the constitution puts it, and it is not the presentation
layer doing it, which is what FR-001 forbids.

`backend/tests/integration/query_count_test.go` already asserts the gallery costs exactly two
statements and provides a `statementCount(t)` helper. The same helper asserts three here, and
asserts the figure does not move between a small vault and a large one.

**Rejected**: one statement with `UNION ALL` across the three dimensions. It would be two fewer
round trips and considerably harder to read, and the round trips are not the cost here.

**Rejected**: a statement per status. Four queries that each scan the same rows, to produce what one
`GROUP BY` produces in a single pass.

---

## Decision 5 — The breakdown tail is folded in SQL, not in Go

**Decision**: the breakdown statements rank groups with a window function and aggregate everything
past the bound into a single remainder row, so the statement returns at most ten rows regardless of
how many distinct series a vault holds.

The easy version ranks in SQL and folds the tail in Go. It is simpler, and it is bounded in
*statements* — which is all FR-006 asks for — but not in rows: a collector with eight hundred
distinct series transfers eight hundred rows to display eight. Folding in SQL keeps both bounded,
and the response size becomes a constant.

**Display spelling** comes from `mode() WITHIN GROUP (ORDER BY btrim(series))`, which returns the
most frequent value and resolves ties by the given sort order rather than arbitrarily. That is what
makes FR-027a's determinism requirement satisfiable without a second pass.

---

## Decision 6 — Normalisation is `lower(btrim(…))`, and `NULL` and `''` are the same group

**Decision**: group on `lower(btrim(coalesce(series, '')))`. A field holding `NULL`, one holding
`''`, and one holding `'   '` all normalise to the empty string and therefore land in one group,
which is presented as the unspecified group.

The edge case named in the spec — two differently-named "unspecified" buckets — is exactly what
happens if `NULL` and `''` are grouped separately, and it is visible to a collector as an obvious
defect. `coalesce` before normalising, rather than after, is what prevents it.

**On case folding and Unicode**: `lower()` is locale-dependent at the margins, and this is not
attempting a linguistically correct case fold for every script. It is attempting to merge
"Good Smile Company" with "good smile company", which it does. Recording the limit rather than
overstating the guarantee.

---

## Decision 7 — Wishlist and Sold are reported, labelled, and kept out of the headline

**Decision**: all four statuses return a count and a total. The API reports them as four peers; the
*presentation* decides which is the headline, and the contract names what each figure means.

This follows the clarification directly, but it has a design consequence worth stating: the backend
does not have a concept of "the headline". It returns four honest figures and a whole-vault total of
priced items. Baking "owned is special" into the response shape would mean a product decision about
emphasis living in the API, and changing the emphasis later would be a contract change rather than a
presentation change.

**What this does not do**: invent a sale price. FR-042 forbids deriving any figure from a field the
product does not record, and "what you got back for sold items" is precisely such a field. The Sold
figure is labelled as what was *paid*, and that wording is a requirement rather than copy.

---

## Decision 8 — Tracing a figure back needs gallery filters this feature must add

**Decision**: FR-030 and FR-031 require every figure to link to the pieces behind it. Status already
filters (`/collection?status=sold`, built in feature 001). Series and manufacturer do not filter at
all, so this feature adds those filters to the gallery and to the list endpoint.

This is genuine added scope and is called out rather than discovered during implementation. It is
small — the list query already takes an optional status predicate and gains two more of the same
shape — but it touches the gallery, which is the most-tested surface in the product.

**The filter must match the grouping.** A group assembled with `lower(btrim(…))` that links to a
filter matching exact text will show a collector a group of five and then a gallery of three. The
filter normalises identically, and an integration test asserts a group's count equals the number of
collectibles its own link returns — because this is the kind of inconsistency that no unit test
catches and every collector notices.

**The unspecified group's link is the awkward one.** Its members are the rows where the field is
`NULL`, `''`, or whitespace. A filter written as `manufacturer IS NULL` returns two of the three; a
filter written as `manufacturer = ''` returns the other one. The predicate must be
`coalesce(lower(btrim(manufacturer)), '') = $1` with an empty `$1`, which is the same expression the
grouping uses — the point of Decision 6 restated on the read side. Naming it here because `IS NULL`
is the obvious thing to reach for and is wrong by exactly the amount nobody checks.

**No index for the filter either**, consistent with Decision 1: the predicate is evaluated over one
collector's rows, which is the same set the grouping already scans.

---

## Decision 9 — Statistics are a fresh read on every visit

**Decision**: the route is rendered server-side per request, with no caching layer, consistent with
how the gallery already works.

FR-039 asks that figures not be presented stale. The simplest way to satisfy it is to have nothing
to go stale. A collector who edits a price and navigates to the statistics sees the new figure
because the figures were computed when the page was requested.

**Rejected**: client-side caching with revalidation. It would add a staleness window to a feature
whose entire value is being correct about numbers, for a page that is not on the hot path.


---

## Decision 10 — The three statements share one read-only transaction

**Decision**: the three aggregates run inside a single read-only transaction, so every figure in a
response describes the same moment.

**This was found by reading the test harness rather than by reasoning.**
`backend/tests/integration/query_count_test.go` names its helper `statementCount`, but what it
actually reads is `xact_commit + xact_rollback` from `pg_stat_database` — it counts **transactions**,
not statements. Checking that before writing a test that asserts "three statements" turned up the
real problem it would have missed.

Three independent reads observe three independent snapshots. A collectible added between the status
query and the series query is counted by one and not the other, and the response then violates FR-017
and FR-026 — counts that must reconcile exactly, failing through no fault of the arithmetic. It is
rare, it is silent, and it looks exactly like a bug in the aggregation when it finally shows up.

One transaction makes the whole response a consistent snapshot. It costs nothing: these are reads of
one collector's rows, held briefly, taking no locks that any writer waits on.

**Consequence for the query-count test**: the correct assertion is that the cost is *constant*
between a small vault and a large one, which is the N+1 property the budget exists to protect. The
absolute number the helper reports is one transaction, not three statements, and the test should say
so rather than repeating a figure that does not mean what its name suggests.

**Rejected**: `REPEATABLE READ`. The default isolation already gives each statement a consistent
snapshot, and a single transaction is what makes the three agree with each other; a stricter level
buys nothing here and invites serialisation failures on a read-only path.
