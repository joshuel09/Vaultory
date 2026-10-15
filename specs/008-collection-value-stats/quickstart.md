# Quickstart: Collection Value & Statistics

How to see this feature work, and — more usefully — how to see it fail.

## Running it

```bash
make up
```

Sign in, add a few collectibles with prices across different statuses, series and manufacturers, then
open **`/collection/stats`**, or follow the link from the gallery.

## Seeing it work

| What to try | What must happen |
|---|---|
| A vault with six priced pieces | The count reads 6 and the total is their exact sum |
| Add a piece with no price | The count rises; the total does not; the view says how many pieces it covers |
| A status holding nothing | Still listed, showing zero — not omitted |
| Change a piece from Preordered to Owned | The figures move on the next visit, with no other action |
| Select a status figure | The gallery opens filtered to exactly those pieces |
| Select a series group | The gallery opens filtered to that series, showing the same number the group claimed |
| An empty vault | The designed empty state, not a grid of zeros |

## Seeing it fail — the checks that matter

These are the failures this feature is actually at risk of. Each is invisible unless provoked.

### The exactness one

Add three collectibles priced `0.10`, `0.20` and `0.30`. The total must read **`0.60`**.

A float-based implementation returns `0.6000000000000001` or, having been rounded for display,
`0.60` — correct here and wrong by a cent somewhere else. The integration test for this compares
against a total assembled independently of the code under test, because a test that sums the same way
as the code proves only that the code agrees with itself.

### The near-duplicate one

Add two collectibles with manufacturer `Good Smile Company` and one with `good smile company`.

They must appear as **one group of three**, labelled with the spelling used twice. Two groups means
normalisation is missing or is being applied in the wrong place.

Then add one with manufacturer left blank and one with only spaces. Both must land in the **same**
unspecified group.

Through the form, that check cannot fail: the domain trims optional text and records a blank value as
absent, so both arrive as `NULL` before the grouping ever sees them. The rows that make the check
meaningful are the ones a migration or a direct correction leaves behind, and they have to be written
with SQL. `TestNullEmptyAndWhitespaceFormOneUnspecifiedGroup` does exactly that, updating the column
underneath the domain. Two unspecified-looking groups there means `coalesce` is being applied after
normalisation rather than before.

### The traceability one

Open any breakdown group, note its count, and follow its link.

The gallery must show that many pieces. A group of five that links to a gallery of three means the
grouping and the filter normalise differently — the single most likely inconsistency in this feature,
and one no unit test catches.

### The N+1 one

```bash
make test
```

`TestStatisticsCostDoesNotGrowWithCollection` asserts the cost is the *same* for a vault of one and
for a vault of sixty across twenty-five series. A number that moves means an aggregate became a loop.

It asserts equality rather than a constant, deliberately. The helper it uses reads
`xact_commit + xact_rollback`, so it counts transactions, not statements — and the view runs all
three of its aggregates inside one read-only transaction, so the figure is one. Asserting "three
statements" would be asserting a detail of the helper rather than the property that matters.

### The privacy one

Two collectors, two vaults. Neither sees the other's figures, and there is no parameter by which
either could ask. If a collector identifier can be passed to this endpoint at all, that is the bug —
FR-003 is satisfied structurally, by the endpoint having no such input, not by a check.

## Running the suites

```bash
make test        # backend unit, integration, contract
make test-e2e    # browser, needs `make up`
```

By hand, without Docker:

```bash
cd backend  && go vet ./... && go test ./tests/unit/...
cd frontend && npm run lint && npm run typecheck && npm run test && npm run build
```

Backend integration results must be confirmed **by test name**. A missing
`VAULTORY_TEST_DATABASE_URL` makes `TestMain` exit 0 with nothing run, which reads exactly like
success — this has caused a false "all green" report in this project before.
