# Feature Specification: Collection Value & Statistics

**Feature Branch**: `39-collection-value-stats`

**Created**: 2026-10-13

**Status**: Draft

**Input**: User description: "Collection value and statistics: totals, counts by status, and breakdowns by series and manufacturer, computed in Go over a collector's own vault."

Feature 006 completed the lifecycle of a single collectible — it can be added, corrected, and removed. What a collector still cannot do is look at the vault *as a whole* and learn anything from it. The gallery shows pieces; it never shows the collection.

This feature is deliberately scheduled after 006 rather than before it. A total computed over a collection that cannot be corrected is a liability: one mistyped price would have been permanent, and the headline figure would have been quietly wrong forever with no way to put it right. Now that a price can be fixed, an aggregate can be trusted to converge on the truth as a collector tidies their vault.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See what the collection holds and what it cost (Priority: P1)

A collector who has been adding pieces for months opens their vault and wants the shape of it in one glance: how many pieces there are, and what has been spent on them.

Today the only way to answer "how much have I spent?" is to open every collectible in turn and add the prices up by hand — which is both tedious and, for a collection of any size, something a collector simply will not do. The number exists in the data and has never been shown.

**Why this priority**: This is the feature. Every other story here is a refinement of it, and a release containing only this story is already worth having — a collector learns something about their collection that the product has never told them. It is also the story that carries all the exactness risk, so building it first means the hard part is proven before anything is layered on top.

**Independent Test**: Add a known set of collectibles with known prices, open the statistics view, and confirm the count and the total match what was entered — to the cent. Deliverable on its own: the collector sees a figure they could not previously obtain.

**Acceptance Scenarios**:

1. **Given** a vault holding six collectibles with recorded purchase prices, **When** the collector opens the statistics view, **Then** the piece count reads 6 and the total equals the exact sum of those six prices, with no rounding drift.
2. **Given** a vault where some collectibles have no recorded purchase price, **When** the collector opens the statistics view, **Then** the total covers only the priced ones and the view states plainly how many pieces are not represented in it.
3. **Given** an empty vault, **When** the collector opens the statistics view, **Then** a designed empty state explains there is nothing to summarise yet and offers the route to adding a first collectible — not a screen of zeros.
4. **Given** two collectors with different vaults, **When** each opens the statistics view, **Then** each sees only their own figures, and nothing a collector does can cause another collector's numbers to be computed or returned.
5. **Given** prices that are awkward in binary — 0.10, 0.20, 10.05 — **When** they are summed, **Then** the total is exact, because the sum is computed and transmitted as an exact decimal and never passes through a floating-point value.

---

### User Story 2 - Understand the split across collection statuses (Priority: P2)

A collector's vault mixes four quite different things: pieces owned, pieces preordered and not yet arrived, pieces merely wished for, and pieces since sold. A single headline total blurs them together, and the blur is misleading in a specific way — committed money, aspirational money, and money that came back are not the same thing.

**Why this priority**: It is the first question anyone asks after seeing the headline figure, and the answer changes how the headline should be read. It depends on Story 1's machinery but adds a dimension to it rather than altering it.

**Independent Test**: Create collectibles across all four statuses with known prices, and confirm each status reports the right count and the right subtotal, including statuses that hold nothing.

**Acceptance Scenarios**:

1. **Given** a vault with collectibles in three of the four statuses, **When** the collector opens the statistics view, **Then** all four statuses are listed, and the unused one shows a count of zero rather than being omitted.
2. **Given** a vault containing preorders, **When** the collector reads the statistics view, **Then** preordered value is presented distinctly from owned value, so that money not yet spent is not read as money already spent.
3. **Given** a collectible whose status is changed from Preordered to Owned, **When** the collector returns to the statistics view, **Then** the figures reflect the change without any further action.
4. **Given** the status subtotals, **When** they are added together, **Then** they reconcile exactly with the counts and totals shown in Story 1 — no piece is counted twice and none is lost.

---

### User Story 3 - See which series and manufacturers dominate the shelf (Priority: P3)

A collector who has drifted toward one series or one manufacturer usually senses it without being able to prove it. The data already records both fields; nothing has ever grouped on them.

**Why this priority**: It is genuinely useful and genuinely optional. It is also the story with the messiest input — both fields are free text — so placing it last keeps that mess away from the exactness work in Story 1.

**Independent Test**: Create collectibles across several series and manufacturers, confirm the groups returned match, and confirm pieces that name neither are accounted for rather than dropped.

**Acceptance Scenarios**:

1. **Given** a vault with collectibles across five series, **When** the collector opens the statistics view, **Then** the series are listed with a count and a value each, ordered so the largest is first.
2. **Given** collectibles that record no series at all, **When** the breakdown is shown, **Then** those pieces appear in a clearly labelled group rather than vanishing from a breakdown that claims to cover the collection.
3. **Given** a vault with more distinct series than the view shows, **When** the breakdown is displayed, **Then** the remainder is represented as a summary group so that the counts still reconcile with the collection total.
4. **Given** a breakdown group, **When** the collector selects it, **Then** they are taken to the gallery filtered to those pieces, so a figure can always be traced back to the things it describes.

---

### Edge Cases

- **A vault where no collectible has a price.** The total is zero, which is true but useless on its own; the view must say that zero of N pieces carry a price, or the collector will read a real zero as a broken screen.
- **A total that exceeds what a single amount can hold.** A single recorded amount is bounded, but a sum across a large vault is not bounded by the same limit. The behaviour at the top end must be defined rather than left to overflow silently into a negative or wrapped figure — this is the failure that would be least visible and most damaging.
- **Series and manufacturer are free text.** "Good Smile Company", "Good Smile", and "good smile company" are three groups to an exact match and one group to a collector. Whatever is chosen, it is a decision to be stated, not an accident to be inherited.
- **Blank versus absent.** A field holding an empty string and a field holding nothing must land in the same group; two differently-named "unspecified" buckets would be a visible defect.
- **A very large vault.** The view must not degrade as the collection grows, and must not issue work proportional to the number of distinct series.
- **Sold pieces.** The only money recorded is what was *paid*, never what was *received*. A "value" for a sold piece is therefore the purchase price of something no longer held, which is a figure that can be read two incompatible ways.
- **A collectible edited while the view is open.** The figures are a snapshot of a moment; nothing here writes, so there is no version to reconcile, but the view must not present a stale figure as live.
- **The statistics of a vault that is not yours.** There is no collectible identifier in this view's input, so there is no "someone else's statistics" to request — and the design must keep it that way, never accepting a collector identifier from the caller.

## Requirements *(mandatory)*

### Functional Requirements

**Where the numbers come from**

- **FR-001**: Every figure presented by this feature MUST be computed server-side. The presentation layer MUST NOT sum, average, count, or otherwise derive any displayed figure from a list of collectibles.
- **FR-002**: Every query backing these figures MUST be predicated on the requesting collector's identifier, taken from the verified session and never from any header, body, path, or query parameter supplied by the caller.
- **FR-003**: The feature MUST expose no way to request another collector's figures. There is no identifier in its input by which such a request could be expressed.
- **FR-004**: Monetary totals MUST be exact decimals throughout — computed, stored, and transmitted without ever being represented as a floating-point value.
- **FR-005**: Monetary totals MUST cross the API as decimal strings, consistent with how individual amounts already travel.
- **FR-006**: The number of queries issued to produce the complete view MUST be bounded by a constant, independent of how many collectibles, series, or manufacturers the vault holds.

**The headline figures**

- **FR-007**: The feature MUST report the total number of collectibles in the collector's vault.
- **FR-008**: The feature MUST report the total of the recorded purchase prices across the vault.
- **FR-009**: Collectibles with no recorded purchase price MUST be excluded from monetary totals and MUST NOT be treated as costing zero.
- **FR-010**: The feature MUST report how many collectibles carry a recorded purchase price, so that every monetary total can be read against the number of pieces it actually covers.
- **FR-011**: The view MUST state, in words a collector can act on, when some pieces are absent from a total because they carry no price.
- **FR-012**: [NEEDS CLARIFICATION: which statuses contribute to the headline total — Owned only, Owned and Preordered, or all four? A wishlist is aspiration rather than expenditure, and a sold piece is money that came back, so a single "total value" across all four has no coherent meaning.]
- **FR-013**: [NEEDS CLARIFICATION: what must happen when a total would exceed the maximum a single monetary amount can represent — refuse, saturate and say so, or widen the type for totals? Silent overflow is the one unacceptable answer.]

**The split by status**

- **FR-014**: The feature MUST report a count for each of the four collection statuses — Owned, Preordered, Wishlist, and Sold.
- **FR-015**: A status holding no collectibles MUST be reported with a count of zero rather than omitted, so the set of statuses is always complete and always the same shape.
- **FR-016**: The feature MUST report the total of recorded purchase prices within each status.
- **FR-017**: The per-status counts MUST sum exactly to the total count, and the per-status monetary totals MUST sum exactly to the total of all priced collectibles.
- **FR-018**: The set of statuses MUST be defined in the contract between frontend and backend rather than restated in the presentation layer, so that adding a status cannot leave the view silently omitting it.
- **FR-019**: [NEEDS CLARIFICATION: should Sold pieces report what was paid for them, given that no sale price is recorded anywhere? Reporting the purchase price of something no longer owned invites a reader to treat recovered money as held value.]

**The breakdowns**

- **FR-020**: The feature MUST report a breakdown of the collection by series, each entry carrying a count and a monetary total.
- **FR-021**: The feature MUST report a breakdown of the collection by manufacturer, each entry carrying a count and a monetary total.
- **FR-022**: Breakdown entries MUST be ordered with the largest first, by a stated and stable measure.
- **FR-023**: Ordering MUST be deterministic: entries that tie on the ordering measure MUST still come back in a stable, repeatable order, so that the view does not reshuffle between loads.
- **FR-024**: Collectibles recording no value for the grouped field MUST be collected into a single clearly labelled group, and a field holding only whitespace MUST land in that same group as one holding nothing at all.
- **FR-025**: The number of breakdown entries returned MUST be bounded, with everything beyond the bound represented as a single remainder group.
- **FR-026**: Counts across all breakdown entries, including the unspecified group and the remainder group, MUST reconcile exactly with the total count.
- **FR-027**: [NEEDS CLARIFICATION: are series and manufacturer grouped by their exact stored text, or by a normalised form that folds case and surrounding whitespace? Exact matching will split near-duplicates that a collector considers one group.]

**Reaching the figures, and tracing them back**

- **FR-028**: [NEEDS CLARIFICATION: does this live at its own route, or as a summary above the existing gallery? The two give very different answers to "how often does a collector see it".]
- **FR-029**: The view MUST be reachable from the collection gallery by an affordance that names what it leads to.
- **FR-030**: Selecting a status figure MUST take the collector to the gallery filtered to that status, so a number can always be traced back to the pieces behind it.
- **FR-031**: Selecting a breakdown entry MUST take the collector to the gallery filtered to that group, with the same intent.
- **FR-032**: The view MUST require a verified session, and a request without one MUST be redirected to sign in rather than shown an error.

**Presentation**

- **FR-033**: Monetary figures MUST be displayed in a form a collector reads as money, without altering the exact value underneath.
- **FR-034**: An empty vault MUST produce a designed empty state that explains why there is nothing to show and offers the route to adding a collectible, rather than a grid of zeros.
- **FR-035**: Every figure MUST be legible in both the dark and light appearances.
- **FR-036**: Every interactive element MUST be reachable and operable by keyboard alone, and MUST carry an accessible name describing what it leads to.
- **FR-037**: The view MUST be usable at the desktop, tablet, and mobile viewports, and MUST NOT scroll horizontally at any of them.
- **FR-038**: While the figures are being fetched, the view MUST show a loading state rather than an empty frame or a figure of zero that will shortly change.
- **FR-039**: Figures MUST reflect the vault as of the moment they were computed, and the view MUST NOT present figures from a previous visit as current.

**Contract and schema**

- **FR-040**: The API contract MUST describe the statistics response completely, and the presentation layer's types MUST be generated from it rather than hand-written.
- **FR-041**: This feature MUST NOT require a schema change. It reads what feature 001 and feature 006 already store; if it appears to need a new column, that is a signal the scope has drifted into recording new data.
- **FR-042**: No figure defined here may be derived from a field this product does not record. In particular, there is no market value and no sale price, and this feature MUST NOT invent either.

### Key Entities

- **Collection Summary**: The whole-vault figures — how many pieces, how many carry a price, and the monetary totals. Derived entirely from existing collectibles; stored nowhere.
- **Status Group**: One of the four collection statuses, with the count and monetary total of the pieces holding it. Always four of them, including empty ones.
- **Breakdown Entry**: A named group — a series or a manufacturer — with the count and monetary total of the pieces in it. Includes the unspecified group and, when the list is truncated, a remainder group.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A collector with a vault of known contents sees a piece count and a total that match that vault exactly; an independent sum of the same prices agrees to the cent.
- **SC-002**: Summing amounts chosen to expose floating-point error — such as 0.10 and 0.20 — produces an exact result, verified against a total assembled independently of the code under test.
- **SC-003**: Per-status counts and totals reconcile exactly with the whole-vault figures, and breakdown counts reconcile with the total count including the unspecified and remainder groups.
- **SC-004**: The complete view is produced by a number of queries that does not grow with the size of the collection, demonstrated against vaults of markedly different sizes.
- **SC-005**: The view renders within the performance budget already applied to the collection gallery, for a vault an order of magnitude larger than a typical one.
- **SC-006**: A collector cannot obtain another collector's figures by any input the API accepts.
- **SC-007**: All four statuses appear whenever the view is shown, including any holding nothing.
- **SC-008**: An empty vault produces the designed empty state, and a vault whose pieces carry no prices produces a total that is accompanied by a statement of how many pieces it covers.
- **SC-009**: Every figure and every control is legible in both appearances and operable by keyboard alone, verified by automated accessibility checks at all three viewports.
- **SC-010**: Selecting any status figure or breakdown entry arrives at a gallery showing exactly the pieces that figure counted.

## Assumptions

- **A single currency.** Amounts are recorded as plain decimals with no currency attached, so summing them is meaningful. Were a currency ever added, every total defined here would need revisiting, because summing across currencies is not addition.
- **Purchase price is the only money recorded.** "Value" throughout this specification means recorded purchase price. The product holds no market value, appraisal, or sale price, and this feature does not add one.
- **The four statuses are those validated today** — Owned, Preordered, Wishlist, Sold. The requirements are written so that adding a fifth surfaces as a contract change rather than as a quietly incomplete view.
- **Statistics are read-only.** Nothing here writes, so the optimistic-concurrency rules that govern edits do not apply; the only freshness concern is not showing a stale snapshot as live.
- **Existing filter parameters are reused** for tracing a figure back to its pieces. The gallery already filters by status; grouping filters may need adding, and that is in scope for this feature's plan.

## Out of Scope

- Market value, appraisals, sale prices, or profit and loss. These require recording data the product does not hold, and each is a feature in its own right.
- Charts, graphs, or any visualisation beyond presenting figures and groups legibly.
- Value over time, trend lines, or history. Nothing records when a collectible's price changed, so no honest time series can be drawn from what exists.
- Exporting the figures. That belongs with a general collection export.
- Statistics across multiple collectors, or any comparison between vaults. Vaults are private.
