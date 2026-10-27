# Feature Specification: Search and Sort a Collection

**Feature Branch**: `32-search-and-sort`

**Created**: 2026-10-27

**Status**: Draft

**Input**: Issue #32 — "Search and sort a collection." The gallery orders one way (newest first) and
narrows one way (by status, plus the series and manufacturer links from the statistics). At a few
hundred entries that stops being enough: a collector looking for one piece by name, or what they
paid most for, has to scroll. Done when: a collector with a large vault can find one collectible
without scrolling, and can order the gallery by something other than when they added things.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find one collectible by what I remember about it (Priority: P1)

A collector with a large vault remembers part of a name, a character, a series, or who made it. They
type it into a search box on the gallery and see only the collectibles that match, without scrolling.

**Why this priority**: The issue's first done-condition, and the most common need in a large vault.

**Independent Test**: In a vault of several hundred collectibles, search for a word that appears in
exactly one collectible's manufacturer; the gallery shows that one collectible.

**Acceptance Scenarios**:

1. **Given** a vault, **When** the collector searches for text that appears anywhere in a
   collectible's name, character, series, manufacturer, category, scale, edition or notes, **Then**
   that collectible is shown, regardless of letter case.
2. **Given** a search of several words, **When** results are shown, **Then** each shown collectible
   contains every word somewhere among those attributes — not necessarily in the same one.
3. **Given** a search that matches nothing, **When** results are shown, **Then** the no-results state
   says which search found nothing and offers a way back to the whole vault; it is not the empty-vault
   state.
4. **Given** an active search, **When** the collector clears it, **Then** the gallery returns to
   showing everything the other active filters allow.

---

### User Story 2 - Order the gallery by something other than when I added things (Priority: P1)

A collector wants to see what they paid most for, what is coming out soonest, or their collection
alphabetically. They choose a sort order and the gallery reorders.

**Why this priority**: The issue's second done-condition.

**Independent Test**: Choose "Price, highest first"; the gallery starts with the most expensive
collectible, and entries with no price come after every priced one.

**Acceptance Scenarios**:

1. **Given** a vault, **When** the collector chooses a sort order, **Then** the gallery is ordered by
   it: newest or oldest added; name A–Z or Z–A; purchase price highest or lowest first; purchase date
   newest or oldest first; release date newest or oldest first.
2. **Given** a sort by an attribute some collectibles do not record, **When** the gallery is shown,
   **Then** those collectibles come after every collectible that records it, in either direction.
3. **Given** no sort chosen, **When** the gallery is shown, **Then** it is newest added first, as
   today.
4. **Given** a sorted gallery larger than one page, **When** the collector keeps scrolling, **Then**
   every collectible appears exactly once, in order, including when many share the same sort value.

---

### User Story 3 - Combine them, and keep my view (Priority: P2)

A collector searches within one status, sorted by price, opens one to edit it, saves, and comes back
to the same view. They can share or reload the address and get the same view again.

**Why this priority**: Search and sort are far less useful if they reset every time the collector
looks away.

**Independent Test**: Filter to Owned, search "kaiju", sort by price; edit a result and save; the
gallery returns with the same filter, search and sort.

**Acceptance Scenarios**:

1. **Given** a status filter, a statistics series or manufacturer filter, a search and a sort, **When**
   all are active, **Then** the gallery shows only collectibles meeting every filter and the search,
   in the chosen order.
2. **Given** any combination, **When** the collector reloads, shares the address, or returns from
   editing, **Then** the same combination is shown.
3. **Given** an active search or sort, **When** the collector changes the status filter, **Then** the
   search and sort are kept.

---

### Edge Cases

- **Whitespace-only or empty search**: no search.
- **Very long search**: refused above 200 characters with a field-level message.
- **Characters with special meaning in matching** (`%`, `_`, `\`): matched literally.
- **A cursor from one sort used with another**: refused as an invalid page reference, never a page
  from a different order.
- **Ties** (same name, same price, same date): ordered stably so paging neither repeats nor skips.
- **Trashed collectibles** (feature 010): never searched, never sorted into the gallery.
- **An unknown sort value** in the address: refused with a field-level message rather than silently
  ignored.
- **The empty vault**: the search and sort controls are not shown, just as the status filter is not.

## Requirements *(mandatory)*

### Functional Requirements

#### Search

- **FR-001**: Collectors MUST be able to search their own vault with free text from the gallery.
- **FR-002**: A search MUST match collectibles where every whitespace-separated word of the search
  appears, case-insensitively, as a substring of at least one of: name, character, series,
  manufacturer, category, scale, edition, notes.
- **FR-003**: Characters in the search MUST be matched literally; none may act as a wildcard.
- **FR-004**: A search MUST be at most 200 characters; an empty or whitespace-only search MUST mean no
  search.
- **FR-005**: A search that matches nothing MUST show the existing no-results state (001 FR-040),
  naming the search, distinct from the empty-vault state.

#### Sort

- **FR-006**: Collectors MUST be able to order the gallery by: date added (newest or oldest first),
  name (A–Z or Z–A), purchase price (highest or lowest first), purchase date (newest or oldest
  first), release date (newest or oldest first). The default MUST remain newest added first (001
  FR-034).
- **FR-007**: Collectibles that do not record the sorted-by attribute MUST come after all that do,
  whichever direction is chosen.
- **FR-008**: Ordering MUST be total and stable: collectibles sharing a sort value MUST appear in a
  consistent order, so that paging through a sorted, searched, filtered gallery shows every matching
  collectible exactly once.
- **FR-009**: Name order MUST ignore letter case.

#### Together

- **FR-010**: Search, sort, the status filter, and the series and manufacturer filters (008) MUST
  compose: results meet every active filter and the search, in the chosen order.
- **FR-011**: The active search and sort MUST be part of the gallery's address, kept on reload, on
  sharing, on returning from editing, and when the status filter changes; and MUST be shown in the
  controls.
- **FR-012**: A page reference from one sort order MUST be refused when used with another.
- **FR-013**: An unknown sort order MUST be refused with a field-level message.
- **FR-014**: Search and sort MUST only ever consider the acting collector's own collectibles in
  their vault — never another collector's, never the Trash (010 FR-002).
- **FR-015**: The search and sort controls MUST be operable by keyboard and labelled for assistive
  technology (001 FR-045), and MUST work at mobile width without horizontal scrolling (001 FR-036).

### Key Entities

- **Search**: free text, up to 200 characters, split into words.
- **Sort order**: one of ten named orders; part of every page reference.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a vault of 500 collectibles, a collector finds one collectible by a word from its
  name, character, series or manufacturer with one search and no scrolling.
- **SC-002**: For every sort order, paging through a 500-collectible vault in pages of 24 returns each
  collectible exactly once, in order — including with many ties and missing values.
- **SC-003**: 100% of collectibles missing the sorted-by attribute appear after all that have it.
- **SC-004**: A search, sort and filter combination survives reload, sharing and returning from an
  edit in 100% of cases.
- **SC-005**: A search or sorted page of a 500-collectible vault is returned as quickly as an
  unsorted, unsearched page is today, to the collector's eye.
- **SC-006**: 0 searches or sorts return a trashed or another collector's collectible.

## Clarifications

### Session 2026-10-27

The issue listed questions for specification; answered here with reasons, open to review.

- Q: Which attributes does search cover? → A: Every free-text attribute a collector records — name,
  character, series, manufacturer, category, scale, edition, notes. The issue said "at minimum" four;
  a collector remembering "signed" or "1/4 scale" should find it too.
- Q: How do several words combine? → A: Every word must appear, in any attribute. "kaiju prime"
  finds a Kaiju figure made by Prime 1; requiring the phrase in one attribute would not.
- Q: Where do missing values sort? → A: Last, in both directions. A collector sorting by price wants
  to see prices; unpriced pieces at the top of "lowest first" would bury the answer.
- Q: Substring or full-text? → A: A planning decision (the issue says so). The spec requires
  substring semantics from the collector's side: part of a word matches.

## Assumptions

- Vaults are hundreds to low thousands of collectibles per collector; search runs within one
  collector's collectibles.
- Built on feature 010 (PR #58, not yet merged), whose Trash predicates the list query already
  carries. This branch is stacked on it.
- The empty-vault state hides the controls, as it hides the status filter today.
- Spec directory 011: 009 and 010 are in use on open branches.
