# Feature Specification: Add & Browse Collectibles

**Feature Branch**: `001-add-browse-collectibles`

**Created**: 2026-09-04

**Status**: Draft

**Input**: User description: "Collectors can add collectibles to their private Vaultory collection and browse the collectibles they have added.

A collector can add a collectible with:
- name
- character
- series or franchise
- manufacturer
- category
- scale
- edition or variant
- collection status
- purchase price
- purchase date
- release date
- notes
- one primary image

Initial collection statuses are:
- Owned
- Preordered
- Wishlist
- Sold

Most collectible metadata is optional, but every collectible must have a name and collection status.

Collectors may own multiple copies of the same collectible.

The collection is displayed as a premium visual gallery where collectible imagery is prominent rather than as an inventory-style table.

Collectors can filter their collection by status.

The collection must have appropriate empty, loading, validation, success, and error states.

A collector's collection is private and must not be accessible by another collector.

This first feature includes adding and viewing collectibles only.

Explicitly exclude:
- editing collectibles
- deleting collectibles
- authentication implementation
- social features
- public profiles
- marketplace functionality
- price scraping
- automatic collection valuation
- notifications
- AI identification
- barcode scanning"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add a collectible to my vault (Priority: P1)

A collector has just received a new statue. They open Vaultory, choose to add a collectible, type
its name, mark it as Owned, and save. Vaultory confirms the collectible was added and it appears in
their collection. If they have more detail on hand — the character, the series, the manufacturer,
the scale, the edition, what they paid and when, the release date, a note about the box condition,
and a photo — they can record all of it in the same step, but none of it is required to finish.

**Why this priority**: Nothing else in Vaultory has value until a collector can get a collectible
into their vault. This is the single capability that turns an empty product into a personal
collection record, and it is the smallest slice that delivers standalone value.

**Independent Test**: Can be fully tested by adding a collectible with only a name and a collection
status, confirming the success state, and confirming the collectible is present in the collector's
collection afterward. Delivers a usable personal record even with no other capability built.

**Acceptance Scenarios**:

1. **Given** a collector with an empty collection, **When** they add a collectible with the name
   "Kaiju Sentinel" and the status Owned, **Then** the collectible is saved, a success confirmation
   is shown, and the collectible appears in their collection.
2. **Given** a collector adding a collectible, **When** they supply every optional attribute
   alongside the name and status, **Then** all supplied values are saved and shown with that
   collectible.
3. **Given** a collector adding a collectible, **When** they submit without a name, **Then** the
   collectible is not saved and a validation message identifies the name as required.
4. **Given** a collector adding a collectible, **When** they submit without choosing a collection
   status, **Then** the collectible is not saved and a validation message identifies the collection
   status as required.
5. **Given** a collector who already has a collectible named "Kaiju Sentinel" marked Owned,
   **When** they add a second collectible with the same name and attributes, **Then** a separate
   second entry is created and both appear independently in their collection.
6. **Given** a collector adding a collectible, **When** saving fails, **Then** an error state is
   shown, the collectible is not saved, and the values they entered are preserved so they can
   retry without retyping.

---

### User Story 2 - Browse my collection as a visual gallery (Priority: P2)

A collector opens their vault to look at what they own. Their collectibles are presented as a
premium visual gallery in which each collectible's image is the dominant element, with its name and
collection status readable alongside. Browsing feels like walking a display shelf rather than
reading a spreadsheet. Collectibles without a photo still look intentional rather than broken.

**Why this priority**: Adding collectibles is only worthwhile if a collector can see and enjoy the
result. This is the experience that distinguishes Vaultory from a generic inventory tool, but it
depends on collectibles existing first.

**Independent Test**: Can be fully tested by seeding a collector's collection, opening the
collection view, and confirming the gallery presentation, the image-forward entries, the placeholder
treatment for imageless collectibles, and the empty, loading, and error states. Delivers value as a
viewable collection even without filtering.

**Acceptance Scenarios**:

1. **Given** a collector with several collectibles, **When** they open their collection, **Then**
   the collectibles are presented as a visual gallery in which imagery is the most prominent
   element of each entry, not as a row-and-column inventory table.
2. **Given** a collector with a collectible that has no image, **When** they view their collection,
   **Then** that entry shows an intentionally designed placeholder and remains readable and
   visually consistent with the rest of the gallery.
3. **Given** a collector with no collectibles at all, **When** they open their collection, **Then**
   an empty state is shown that explains the vault is empty and offers a way to add a first
   collectible.
4. **Given** a collector opening their collection, **When** the collection has not finished being
   retrieved, **Then** a loading state is shown rather than a blank screen or a flash of the empty
   state.
5. **Given** a collector opening their collection, **When** the collection cannot be retrieved,
   **Then** an error state is shown that explains the failure and offers a way to try again.
6. **Given** a collector with a large collection, **When** they browse it, **Then** the gallery
   remains responsive and navigable without requiring the entire collection to be presented at
   once.

---

### User Story 3 - Filter my collection by status (Priority: P3)

A collector wants to see only what they have on preorder, or only what is on their wishlist. They
choose a collection status and the gallery narrows to just those collectibles. They can return to
seeing everything in one action.

**Why this priority**: Filtering makes a growing collection manageable and gives the four statuses
practical meaning, but a collector gets real value from adding and viewing before any filtering
exists.

**Independent Test**: Can be fully tested by seeding a collection containing collectibles in each
of the four statuses, applying each status filter in turn, and confirming that only matching
collectibles are shown and that a no-match result is distinguishable from an empty vault.

**Acceptance Scenarios**:

1. **Given** a collector whose collection contains collectibles in several statuses, **When** they
   filter by Preordered, **Then** only their Preordered collectibles are shown.
2. **Given** a collector viewing a filtered collection, **When** they clear the filter, **Then**
   all of their collectibles are shown again.
3. **Given** a collector whose collection contains no Sold collectibles, **When** they filter by
   Sold, **Then** a no-results state is shown that makes clear the filter matched nothing and is
   distinct from the empty-vault state.
4. **Given** a collector viewing a filtered collection, **When** they look at the view, **Then**
   the active status filter is clearly indicated.

---

### Edge Cases

- A collector adds a collectible supplying only a name and a collection status, leaving every other
  attribute blank — the collectible must save successfully and display without gaps that look like
  errors.
- A collector adds a collectible with no image — the gallery must show a designed placeholder, not
  a broken or missing-image artifact.
- A status filter matches nothing — this must be presented differently from a collection that has
  no collectibles at all.
- A collector submits the add form twice in quick succession, or retries after a connection loss —
  only the collectible they intended is created, and a failed attempt must not leave a partial
  entry behind.
- A collector enters a purchase price of zero (a gift or a giveaway win) — this must be accepted and
  distinguished from no price being recorded.
- A collector enters a negative purchase price — this must be rejected with a validation message.
- A collector enters a purchase price with more precision than a currency supports, or with
  thousands separators or a currency symbol — the value must either be interpreted unambiguously or
  rejected with a clear message, never silently rounded into a different amount.
- A collector enters a very large purchase price — it must be stored and displayed exactly, without
  loss of precision.
- A collector enters a purchase date in the future — this must be rejected with a validation
  message.
- A collector marks a collectible Preordered but gives a release date already in the past, or marks
  a collectible Owned with a release date in the future — both are legitimate collector situations
  and must be accepted as recorded.
- A collector enters a purchase date earlier than the release date, or later than it — both are
  legitimate and must be accepted.
- A collector enters an extremely long name or an extremely long note — the value must either be
  accepted and displayed without breaking the gallery layout, or rejected with a length message
  before saving.
- A collector enters a name consisting only of whitespace — this must be treated as a missing name
  and rejected.
- A collector enters characters from non-Latin scripts, accented characters, or emoji in text
  attributes — these must be stored and displayed faithfully.
- A collector supplies an image that is too large, or in a format Vaultory does not support — this
  must be refused with a message stating the limit or the accepted formats, and must not prevent
  the collectible from being saved without an image.
- A collector attempts to reach a collectible belonging to another collector — the attempt must be
  refused, and the refusal must not reveal whether that collectible exists.
- A collector's session identity cannot be established when they open their collection — no other
  collector's data may be shown, and the situation must be surfaced rather than presented as an
  empty vault.

## Requirements *(mandatory)*

### Functional Requirements

**Adding a collectible**

- **FR-001**: Collectors MUST be able to add a collectible to their own collection.
- **FR-002**: System MUST require a name and a collection status on every collectible, and MUST
  reject any submission missing either, identifying which required value is absent.
- **FR-003**: System MUST treat a name consisting only of whitespace as a missing name.
- **FR-004**: System MUST accept exactly four collection statuses — Owned, Preordered, Wishlist,
  and Sold — and MUST reject any other status value.
- **FR-005**: System MUST allow a collectible to be saved when only a name and a collection status
  are supplied.
- **FR-006**: System MUST allow a collector to optionally record, on each collectible: character,
  series or franchise, manufacturer, category, scale, edition or variant, purchase price, purchase
  date, release date, notes, and one primary image.
- **FR-007**: System MUST store and display every optional value the collector supplied, and MUST
  distinguish an unrecorded value from an empty or zero value.
- **FR-008**: System MUST accept one primary image per collectible.
  [NEEDS CLARIFICATION: How does a collector supply the primary image — by selecting an image file
  from their device, or by providing a link to an image hosted elsewhere?]
- **FR-009**: System MUST allow a collectible to be saved with no image.
- **FR-010**: System MUST record purchase price as an exact monetary amount and MUST NOT introduce
  rounding or precision loss in storage or display.
- **FR-011**: System MUST reject a negative purchase price with a validation message, and MUST
  accept a purchase price of zero as a recorded amount.
- **FR-012**: System MUST reject a purchase date later than the current date with a validation
  message.
- **FR-013**: System MUST accept a release date in the past or in the future, in any combination
  with collection status and purchase date.
- **FR-014**: System MUST validate a submission before saving and MUST report all validation
  problems in a single response rather than one at a time.
- **FR-015**: System MUST confirm to the collector, through a visible success state, that a
  collectible was added.
- **FR-016**: System MUST present an error state when adding fails, MUST NOT save a partial
  collectible, and MUST preserve the values the collector entered so they can retry without
  re-entering them.
- **FR-017**: System MUST allow a collector to add multiple collectibles sharing the same name and
  attributes; each MUST be stored and displayed as an independent entry and MUST NOT be merged,
  de-duplicated, or represented as a quantity.
- **FR-018**: System MUST allow each such entry to carry its own collection status, purchase price,
  purchase date, notes, and image, independently of the others.

**Privacy and ownership**

- **FR-019**: System MUST associate every collectible with the collector who added it.
- **FR-020**: System MUST NOT allow a collector to view, browse, filter, or otherwise reach a
  collectible belonging to another collector.
- **FR-021**: System MUST refuse a request for another collector's collectible without revealing
  whether that collectible exists.
- **FR-022**: System MUST determine which collector a request belongs to on the server side, and
  MUST NOT rely on a collector-supplied claim of identity or ownership.
- **FR-023**: System MUST NOT display any collection content when the acting collector's identity
  cannot be established.

**Browsing the collection**

- **FR-024**: System MUST present a collector's collection as a visual gallery in which each
  collectible's imagery is the most prominent element of its entry.
- **FR-025**: System MUST NOT use a row-and-column inventory table as the primary presentation of
  the collection.
- **FR-026**: Each gallery entry MUST show at least the collectible's name and its collection
  status.
- **FR-027**: System MUST display an intentionally designed placeholder in place of missing imagery,
  visually consistent with the rest of the gallery.
- **FR-028**: System MUST order the collection with the most recently added collectible first by
  default.
- **FR-029**: System MUST remain responsive and navigable for large collections without presenting
  the entire collection at once.
- **FR-030**: System MUST present the collection legibly and usably on desktop, tablet, and mobile
  screen sizes.

**Filtering**

- **FR-031**: Collectors MUST be able to filter their collection by collection status.
- **FR-032**: System MUST provide a way to return to viewing all collectibles regardless of status.
- **FR-033**: System MUST clearly indicate which status filter is currently active.
- **FR-034**: System MUST show a no-results state when a filter matches no collectibles, visually
  and textually distinct from the empty-collection state.

**Interface states**

- **FR-035**: System MUST show a designed empty state, offering a path to add a first collectible,
  when a collector has no collectibles.
- **FR-036**: System MUST show a loading state while a collection is being retrieved, without a
  blank screen and without briefly showing the empty state.
- **FR-037**: System MUST show an error state, with a way to retry, when a collection cannot be
  retrieved.
- **FR-038**: System MUST convey collection status by a means other than color alone.
- **FR-039**: System MUST make adding and browsing operable by keyboard, and MUST provide a text
  alternative for collectible imagery.

### Key Entities *(include if feature involves data)*

- **Collector**: The person who owns a vault. Owns zero or more collectibles. A collector's
  collection is visible only to that collector. Identity is assumed already established; see
  Assumptions.
- **Collectible**: A single entry in a collector's vault, representing one physical or intended
  item. Always has a name and a collection status. May additionally carry a character, a series or
  franchise, a manufacturer, a category, a scale, an edition or variant, a purchase price, a
  purchase date, a release date, notes, and one primary image. Belongs to exactly one collector.
  Two collectibles with identical attributes remain distinct entries.
- **Collection Status**: The state of a collectible in the collector's vault — one of Owned,
  Preordered, Wishlist, or Sold. Exactly one applies to a collectible at a time, and it is the
  basis for filtering.
- **Primary Image**: The single representative picture of a collectible, and the dominant element of
  its gallery entry. Optional; when absent, a designed placeholder takes its place.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A collector can add a collectible with only its required values and see it in their
  collection in under 60 seconds, without consulting documentation or help.
- **SC-002**: A collector can add a collectible with all thirteen recordable attributes in under 3
  minutes.
- **SC-003**: 95% of collectors succeed at adding their first collectible on the first attempt,
  without abandoning the task or producing a validation error they cannot resolve.
- **SC-004**: A collector opening a collection of 500 collectibles can begin browsing it within 2
  seconds, and can scroll through it without visible stalling.
- **SC-005**: 100% of attempts by a collector to reach another collector's collectible are refused,
  verified across direct reach attempts, browsing, and filtering.
- **SC-006**: Every one of the empty, loading, validation, success, error, and no-results states is
  reachable in a walkthrough, and each is visually distinct from the others.
- **SC-007**: A collector shown their collection identifies it as a visual gallery of their
  collectibles rather than an inventory or admin listing, in 9 out of 10 first impressions.
- **SC-008**: A collector can narrow their collection to a single status, and return to viewing all
  of it, in one action each.
- **SC-009**: 100% of monetary amounts entered are stored and redisplayed as the exact amount
  entered, verified across zero, fractional, and very large values.
- **SC-010**: The collection view is usable at common desktop, tablet, and mobile widths with no
  horizontal scrolling of the page and no overlapping or clipped content.
- **SC-011**: Adding and browsing can be completed entirely by keyboard, and every collectible
  image exposes a text alternative.

## Assumptions

**Collector identity**

- Each collector acts within an already-established identity. How that identity is created,
  proven, or ended — registration, login, sessions, password recovery — is out of scope for this
  feature and assumed to be provided. Scoping every collectible to its owning collector, and
  refusing access to any other collector's collectibles, **is** in scope and specified above.
- A collectible belongs to exactly one collector. Shared, joint, or transferred vaults are not
  contemplated.

**Data and defaults**

- Purchase prices are recorded in a single currency for all collectors; no currency is selected or
  stored per collectible. Multi-currency support is a later concern.
- Character, series or franchise, manufacturer, category, scale, and edition or variant are recorded
  as free text. No controlled vocabulary, catalog, or predefined list backs them in this feature,
  which is why status is the only filter.
- Purchase date and release date are calendar dates without a time of day.
- A collectible marked Sold retains its purchase price; no sale price, sale date, or profit
  calculation is captured.
- The status filter selects one status at a time, plus an option for all statuses. Multi-status
  selection is not included.
- The collection is ordered most-recently-added first. Collector-chosen sorting is not included.
- Free-text search is not included in this feature; status filtering is the only way to narrow the
  collection.
- Once added, a collectible's values are fixed for the life of this feature, because editing is out
  of scope. A collector who makes a mistake cannot correct it yet, and cannot remove the entry —
  this is an accepted, temporary consequence of the stated scope.

**Scope boundaries** — the following are explicitly excluded from this feature:

- Editing collectibles
- Deleting collectibles
- Authentication implementation
- Social features
- Public profiles
- Marketplace functionality
- Price scraping
- Automatic collection valuation
- Notifications
- AI identification
- Barcode scanning
