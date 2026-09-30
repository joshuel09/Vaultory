# Feature Specification: Edit & Delete Collectibles

**Feature Branch**: `26-edit-delete-collectibles`

**Created**: 2026-09-28

**Status**: Draft

**Input**: User description: "Editing and deleting collectibles (feature 006). Feature 001 built adding and browsing and explicitly excluded both editing and deleting. Its own spec records the cost at line 90: the submission-key defence against double-submit exists to close 'a defect that would be unrecoverable, since deleting is out of scope.'

A collection changes. A preorder arrives and becomes owned. A price was recorded wrongly. A name has a typo. A statue was added twice. A piece is sold. Today none of that can be put right — a collector can only keep adding.

In scope: changing any attribute of a collectible already in a vault, including its collection status and its photograph; replacing a photograph or removing it so the collectible falls back to the designed placeholder; deleting a collectible with a confirmation proportionate to the fact that it cannot be undone; and both operations restricted to the collector who owns the collectible.

Out of scope: undo, a trash or archive that holds deleted collectibles, restoring something deleted, bulk editing, bulk deleting, and any edit history or audit trail. Those are a larger feature about reversibility. This one is about the collection being editable at all.

Constraints that must survive from feature 001, and that editing makes easier to lose because these are the first writes that change existing rows rather than only insert new ones: every query for collection data is scoped to the owning collector; another collector's collectible answers 404 and never 403, because a 403 confirms it exists; the database itself refuses a cross-collector image reference through a composite foreign key; monetary values are exact decimals and never floating point; the same validation rules that apply when adding apply when editing, enforced by the Go service rather than only in the browser; and schema changes ship as reversible migrations.

Architecture unchanged: Go owns business logic, validation, authorization and persistence; Next.js renders. The Go service verifies every session itself and never trusts an identity asserted by the frontend. The OpenAPI contract is the source of truth for the frontend/backend seam, so new operations belong in it and frontend types are generated from it rather than hand-written."

## Clarifications

### Session 2026-09-30

- Q: When a collector saves an edit from a form they opened before that same collectible was already
  changed, what should happen? (FR-027) → A: Refuse the save, tell the collector the collectible
  changed since they opened it, and show them the current values so they can redo their change
  deliberately. The collectible carries a version or last-changed marker that the save is checked
  against. Last-write-wins is rejected: it is precisely the silent overwrite Principle IV forbids.
  Field-by-field merging is rejected as well, because two collectors editing the same field leaves
  it ambiguous and it needs per-field original values to be carried around.
- Q: When a photograph stops being referenced, when should the stored image file itself be
  destroyed? (FR-020) → A: The reference is removed immediately, so the photograph is unfetchable at
  once; the file is destroyed straight after. A failure to destroy the file does not fail the
  deletion or the edit — the leftover file is retried rather than abandoned. Making storage success
  a condition of deletion would let a storage fault make a collectible undeletable, which is the
  worse failure.
- Q: Where does a collector start an edit, and where is deleting offered? → A: Editing is started
  from the collectible's own entry in the gallery and happens on a dedicated editing screen, mirroring
  the screen for adding. Deleting is offered on that editing screen only, never as an action on a
  gallery entry, so deletion is never one stray click away while browsing. No separate read-only
  detail view is introduced by this feature.
- Q: A collector deletes a collectible that is already gone — a retry, a second tab, or an entry
  deleted elsewhere. Is that an error? (FR-025, FR-029) → A: No. Deleting something that is not in
  the collector's vault is answered as a success, whatever the reason it is absent, because the end
  state the collector asked for already holds. Retry-safety is a rule of the system rather than a
  convention the browser observes. Editing something that is no longer there is still reported as
  not found, because there is nothing to show the collector.
- Q: Should a deletion leave any record, given that this feature deliberately ships no undo and no
  history? → A: Yes, server-side only: which collector, which collectible identifier, and when —
  never the collectible's content, and never surfaced in the product. An irreversible operation with
  no trace at all cannot be investigated when a collector reports something missing. This is an
  operational record, not the in-product history that remains out of scope.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Correct a collectible already in my vault (Priority: P1)

A collector notices that a statue they added last month is recorded with the wrong purchase price, and
that its name has a typo. They open that collectible from their gallery, see every value they
originally entered already filled in, change the two that are wrong, and save. The gallery shows the
corrected values immediately, in the same place in the gallery as before.

The same journey covers a preorder that has arrived: the collector changes its collection status from
Preordered to Owned. Nothing else about the collectible changes, and no second entry appears.

**Why this priority**: Everything else in this feature is a special case of it. A collection is a
record that is wrong the moment reality moves on, and today a collector's only remedy for any
mistake — a typo, a wrong price, a status that has changed — is to leave it wrong. Correcting
attributes is the smallest slice that makes the vault a maintained record rather than an append-only
log, and it delivers standalone value even if deleting is never built.

**Independent Test**: Can be fully tested by adding a collectible, opening it for editing, changing
its name and collection status, saving, and confirming both the gallery and a fresh reload show the
new values and no additional entry. Delivers a maintainable collection with no other capability in
this feature built.

**Acceptance Scenarios**:

1. **Given** a collector with a saved collectible, **When** they open it for editing, **Then** every
   attribute they previously recorded is shown already filled in with its current value, and
   attributes they never supplied are shown empty.
2. **Given** a collector editing a collectible, **When** they change its name and save, **Then** the
   new name is stored, a success confirmation is shown, and the collectible appears in the gallery
   under its new name.
3. **Given** a collector editing a collectible marked Preordered, **When** they change its collection
   status to Owned and save, **Then** the status is Owned, no second collectible is created, and no
   other attribute changes.
4. **Given** a collector editing a collectible that has a purchase price recorded, **When** they clear
   the purchase price and save, **Then** the collectible is stored with no purchase price rather than
   with zero.
5. **Given** a collector editing a collectible, **When** they clear the name and save, **Then** the
   collectible is not changed and a validation message identifies the name as required.
6. **Given** a collector editing a collectible, **When** they enter a negative purchase price or a
   purchase date later than today, **Then** the collectible is not changed and a validation message
   identifies each offending value.
7. **Given** a collector editing a collectible, **When** saving fails, **Then** an error state is
   shown, nothing is changed, and the values they entered are preserved so they can retry without
   retyping.
8. **Given** a collector editing a collectible, **When** they abandon the edit without saving,
   **Then** the collectible is unchanged.
9. **Given** a collectible that belongs to another collector, **When** a collector tries to open it
   for editing or to save a change to it, **Then** the request is reported as not found, the
   collectible is not changed, and nothing in the response distinguishes it from an identifier that
   never existed.

---

### User Story 2 - Remove a collectible from my vault (Priority: P2)

A collector realises they added the same statue twice, or has parted with a piece they no longer want
recorded at all. They open the duplicate, choose to delete it, and are shown a confirmation that names
the collectible and says plainly that this cannot be undone. They confirm, the collectible disappears
from their gallery, and it does not come back on reload.

**Why this priority**: Deletion is the one repair that editing cannot perform. Feature 001 shipped a
submission-key defence specifically because a duplicate created by a double-click would have been
permanent; that defect class stops being unrecoverable only when deleting exists. It ranks below
editing because it addresses fewer situations, and because an incorrect entry can at least be
corrected in place today once User Story 1 exists.

**Independent Test**: Can be fully tested by adding two identical collectibles, deleting one,
confirming the gallery shows exactly one afterwards and still does after a reload, and confirming the
deletion required an explicit confirmation step.

**Acceptance Scenarios**:

1. **Given** a collector viewing a collectible, **When** they choose to delete it, **Then** a
   confirmation naming that collectible is shown and states that deleting cannot be undone, and
   nothing is deleted until they confirm.
2. **Given** the delete confirmation is shown, **When** the collector cancels or dismisses it,
   **Then** the collectible is not deleted and remains in the gallery.
3. **Given** the delete confirmation is shown, **When** the collector confirms, **Then** the
   collectible is removed from their collection, a confirmation of the deletion is shown, and it is
   absent from the gallery after a fresh reload.
4. **Given** a collector has deleted a collectible, **When** they delete two identical entries one
   after the other, **Then** only the entries they confirmed are removed and any remaining identical
   entry is untouched.
5. **Given** a collector confirms a deletion, **When** the same deletion is submitted a second time —
   by a double click, or a retry after a connection loss — **Then** the outcome is still a deleted
   collectible and the collector is not shown an error.
6. **Given** a collectible that belongs to another collector, **When** a collector tries to delete it,
   **Then** the collectible is not deleted, and the answer is exactly the answer given for an
   identifier that never existed, revealing nothing about whether it exists.
7. **Given** a deleted collectible had a photograph, **When** anyone including its former owner
   requests that photograph afterwards, **Then** it is not retrievable.

---

### User Story 3 - Change or remove a collectible's photograph (Priority: P3)

A collector photographed a figure badly, or photographed the box before unboxing it. They open the
collectible, replace the photograph with a better one, and the gallery shows the new image. A
collector who would rather show no photograph at all than a poor one removes it instead, and the
collectible falls back to Vaultory's designed placeholder.

**Why this priority**: Imagery is the centrepiece of the gallery, so a wrong photograph is more
visible than a wrong field — but it is the only attribute a collector can work around today by
deleting and re-adding once User Story 2 exists. It is also the attribute with its own upload rules,
so it is cleanly separable from the rest of editing.

**Independent Test**: Can be fully tested by adding a collectible with a photograph, replacing it with
a different one, confirming the gallery shows the new image, then removing the photograph and
confirming the placeholder is shown in its place.

**Acceptance Scenarios**:

1. **Given** a collectible with a photograph, **When** the collector uploads a different photograph
   and saves, **Then** the collectible shows the new photograph everywhere it is displayed, framed
   consistently with every other collectible.
2. **Given** a collectible with no photograph, **When** the collector adds one and saves, **Then** the
   placeholder is replaced by that photograph.
3. **Given** a collectible with a photograph, **When** the collector removes the photograph and saves,
   **Then** the collectible is stored with no photograph and the designed placeholder is shown in its
   place.
4. **Given** a collector replacing a photograph, **When** they upload a file larger than 10 MB or in a
   format other than JPEG, PNG, or WebP, **Then** the upload is refused with a message stating the
   limit or the accepted formats, and the collectible keeps the photograph it already had.
5. **Given** a collector replacing a photograph, **When** the replacement is accepted and saved,
   **Then** the photograph it replaced is no longer retrievable by anyone, including that collector.
6. **Given** a collector editing a collectible, **When** they change other attributes without touching
   the photograph, **Then** the existing photograph is retained unchanged.

---

### Edge Cases

- **A collectible is edited from two places at once.** A collector has the same collectible open in
  two tabs, edits one, then saves the other from a form loaded before that change. The second save
  MUST be refused with a message explaining that the collectible changed since it was opened, rather
  than silently discarding the first edit (FR-027).
- **A collectible is edited after it has already been deleted elsewhere.** The edit is reported as not
  found and the collector is returned to their gallery with an explanation, rather than being shown a
  generic failure.
- **A collectible is deleted after it has already been deleted elsewhere.** Answered as a success
  (FR-025): the collector asked for it to be gone and it is gone. No error is shown.
- **Editing does not reorder the gallery.** A collectible edited today does not jump to the front of a
  gallery ordered by when collectibles were added (FR-028).
- **A name is changed to only whitespace.** Treated as a missing name and refused, exactly as when
  adding.
- **Every optional attribute is cleared at once.** Permitted: the result is a collectible with only a
  name and a collection status, which is a valid collectible.
- **A photograph upload succeeds but the save that would reference it fails.** The collectible keeps
  the photograph it already had, and the uploaded file is not left retrievable indefinitely.
- **Destroying a photograph's file fails.** The deletion or edit still succeeds and the photograph is
  still unfetchable, because the reference is already gone. The leftover file is retried rather than
  abandoned, and the collector is shown no error (FR-020a).
- **A collector deletes the last collectible in their vault.** The gallery shows the designed empty
  state rather than an empty grid.
- **A collector deletes a collectible while a status filter is active.** The filtered view refreshes
  without the deleted entry and shows the no-results state if it was the only match.
- **An add is retried after the collectible it created was deleted.** A retried add with a submission
  key whose collectible no longer exists creates a new collectible rather than failing. The original
  deletion stands; this is accepted behaviour, not a resurrection.

## Requirements *(mandatory)*

### Functional Requirements

#### Opening a collectible for editing

- **FR-001**: Collectors MUST be able to open a collectible for editing from that collectible's own
  entry in their gallery, on a dedicated editing screen.
- **FR-002**: System MUST present, when a collectible is opened for editing, the current stored value
  of every attribute that collectible has, including its collection status and its photograph.
- **FR-003**: System MUST present attributes the collector never supplied as empty rather than as
  invented defaults.

#### Changing attributes

- **FR-004**: Collectors MUST be able to change any attribute of a collectible they own: name,
  character, series, manufacturer, category, scale, edition, collection status, purchase price,
  purchase date, release date, notes, and photograph.
- **FR-005**: System MUST allow any optional attribute to be cleared back to having no value, and MUST
  distinguish an absent value from an empty or zero one.
- **FR-006**: System MUST allow a collection status to be changed to any of the four statuses — Owned,
  Preordered, Wishlist, Sold — from any other, in any order, with no transition forbidden.
- **FR-007**: System MUST apply an edit to exactly one collectible and MUST NOT alter any other entry,
  including entries sharing the same name and attributes.
- **FR-008**: System MUST NOT create a new collectible as a result of an edit.
- **FR-009**: System MUST persist an edit in full or not at all, leaving no partially updated
  collectible.

#### Validating an edit

- **FR-010**: System MUST apply to an edit exactly the same validation rules that apply when adding a
  collectible, with no rule relaxed or added.
- **FR-011**: System MUST require a name and a collection status on every saved edit, and MUST treat a
  name consisting only of whitespace as a missing name.
- **FR-012**: System MUST reject a negative purchase price and a purchase date later than the current
  date, with a validation message identifying the offending value.
- **FR-013**: System MUST record purchase price as an exact monetary amount and MUST NOT introduce
  rounding or representation error when an existing price is read, redisplayed, and saved unchanged.
- **FR-014**: System MUST report all validation problems in an edit together rather than one at a
  time, and MUST NOT change the collectible when any of them is present.
- **FR-015**: System MUST enforce every validation rule on the server, independently of any check
  performed in the browser.

#### The photograph

- **FR-016**: Collectors MUST be able to replace the photograph on a collectible they own, subject to
  the same accepted formats — JPEG, PNG, WebP — and the same 10 MB limit that apply when adding.
- **FR-017**: Collectors MUST be able to remove the photograph from a collectible, after which the
  designed placeholder is shown in its place.
- **FR-018**: System MUST retain a collectible's existing photograph when an edit does not address the
  photograph.
- **FR-019**: System MUST keep the existing photograph in place when a replacement is refused or the
  edit fails.
- **FR-020**: System MUST make a photograph unretrievable, by any requester including its owner, from
  the moment the collectible referencing it is deleted or the photograph is replaced or removed.
- **FR-020a**: System MUST destroy the stored image file once nothing references it, and MUST NOT make
  the success of a deletion or an edit conditional on that file having been destroyed. A file that
  could not be destroyed MUST be retried rather than abandoned.
- **FR-021**: System MUST NOT allow a collectible to reference a photograph belonging to a different
  collector, and this MUST be prevented by the persistence layer itself rather than by application
  code alone.

#### Deleting

- **FR-022**: Collectors MUST be able to delete a collectible they own, permanently.
- **FR-022a**: System MUST offer deleting from the screen where a collectible is being edited, and MUST
  NOT offer it as an action on a collectible's entry in the gallery.
- **FR-023**: System MUST require an explicit confirmation before deleting, which names the
  collectible being deleted and states that deletion cannot be undone; the destructive choice MUST NOT
  be the action taken by dismissing the confirmation or by pressing Enter on it.
- **FR-024**: System MUST delete exactly the confirmed collectible and MUST NOT affect any other entry,
  including identical ones.
- **FR-025**: System MUST answer a deletion of a collectible that is not in the acting collector's
  vault as a success — whether it never existed, was already deleted, or belongs to another collector
  — so that a repeated or retried deletion is never an error, and MUST enforce this on the server
  rather than relying on the browser to interpret a failure as success.
- **FR-026**: System MUST remove a deleted collectible from the gallery, from every status filter, and
  from any subsequent retrieval of the collection.

#### Not losing data

- **FR-027**: System MUST refuse a save based on a version of the collectible that has since changed,
  MUST tell the collector that the collectible changed since they opened it, and MUST show them the
  collectible's current values, rather than overwriting the newer values silently.
- **FR-027a**: System MUST carry, on every collectible, a marker that changes whenever the
  collectible changes, and MUST check a submitted edit against the marker the collector was shown
  when they opened it.
- **FR-028**: System MUST NOT change a collectible's position in a gallery ordered by when its entries
  were added; editing a collectible MUST NOT move it.
- **FR-029**: System MUST report an edit of a collectible that no longer exists as not found, and MUST
  return the collector to a usable view of their collection.

#### Ownership and privacy

- **FR-030**: System MUST scope every retrieval, edit, and deletion to the collector the request
  belongs to, so that no operation can reach a collectible in another collector's vault.
- **FR-031**: System MUST answer a request to open, edit, or delete another collector's collectible
  identically to a request for an identifier that does not exist, revealing nothing about whether the
  collectible exists.
- **FR-032**: System MUST determine which collector a request belongs to on the server side, and MUST
  NOT accept a collector identity asserted by the caller.
- **FR-033**: System MUST refuse every edit and deletion when the acting collector's identity cannot be
  established.

#### States, feedback, and accessibility

- **FR-034**: System MUST confirm to the collector, through a visible success state, that an edit was
  saved and that a collectible was deleted.
- **FR-035**: System MUST present an error state, with a way to retry, when an edit or deletion fails,
  and MUST preserve the collector's entered values on a failed edit.
- **FR-036**: System MUST show a loading state while a collectible is being retrieved for editing, and
  MUST prevent a second submission of the same edit or deletion while one is in flight.
- **FR-037**: System MUST make editing, photograph replacement and removal, and deletion — including
  the delete confirmation — fully operable by keyboard, with the confirmation receiving focus when it
  opens and returning focus sensibly when it closes.
- **FR-038**: System MUST present every screen and dialog in this feature in both a dark and a light
  appearance, following the collector's system preference, consistently with the rest of the product.
- **FR-039**: System MUST present editing and deletion legibly and usably on desktop, tablet, and
  mobile.

#### Keeping a record of what was destroyed

- **FR-040**: System MUST record, server-side, that a deletion occurred — which collector, which
  collectible identifier, and when — and MUST NOT record the collectible's content in that record.
- **FR-041**: System MUST NOT surface any such record in the product; it exists for operational
  investigation, not as collector-visible history.

### Key Entities *(include if data involved)*

- **Collectible**: An entry in one collector's vault, as defined in feature 001. This feature adds two
  things to it: a marker that changes whenever the collectible changes, so a collector can be told
  their edit is based on a stale view rather than having it silently applied (FR-027, FR-027a), and
  the fact that it can now cease to exist.
- **Collectible photograph**: An uploaded image owned by a collector and referenced by at most one
  collectible. This feature makes the reference changeable and removable, and makes a photograph's
  lifetime end when the collectible referencing it is deleted or its reference replaced (FR-020).
- **Collector**: Unchanged. Remains the owner of every collectible and photograph, and the subject
  every retrieval, edit, and deletion is scoped to.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A collector can correct a single wrong attribute on a collectible in their gallery — for
  example a price or a status — and see the corrected value, in under 30 seconds and without leaving
  and re-entering any other value.
- **SC-002**: 100% of attempts to open or edit a collectible belonging to another collector are
  answered as not found, and 100% of attempts to delete one are answered exactly as a deletion of an
  identifier that never existed. In neither case does the response distinguish an existing
  collectible from a non-existent one, and in no case is the collectible changed or deleted.
- **SC-003**: 0 collectibles are deleted without the collector having passed an explicit confirmation
  step that named the collectible.
- **SC-004**: 100% of edits leave the collectible in the same gallery position it occupied before the
  edit.
- **SC-005**: 100% of photographs that have been replaced, removed, or attached to a deleted
  collectible are unretrievable afterwards, including by the collector who uploaded them.
- **SC-006**: An edit saved from a view of the collectible that is no longer current is refused and
  reported to the collector in 100% of cases; 0 edits overwrite a newer value without the collector
  being told.
- **SC-007**: A collector who edits a collectible and reloads the gallery sees exactly the values they
  saved, and exactly one entry, in 100% of cases.
- **SC-008**: Every validation rule that rejects a value when adding a collectible rejects the same
  value when editing one, with no rule applying in only one of the two.
- **SC-009**: 100% of deletions leave a server-side record identifying the collector, the collectible,
  and the time; 0 such records contain the collectible's name, notes, or any other content.
- **SC-010**: Repeating a deletion that has already succeeded produces a success and no error message,
  in 100% of cases.

## Assumptions

- **Editing is reached from the gallery, on its own screen.** A collector starts an edit from the
  collectible's own entry in their gallery, and edits on a dedicated screen mirroring the one for
  adding (FR-001). Feature 001 did not build a single-collectible view, so retrieving one collectible
  for editing is new work this feature introduces; no read-only detail view is added.
- **Deleting is offered on the editing screen only** (FR-022a), not as an action on gallery entries,
  so that deletion is never one stray click away in a browsing context.
- **The confirmation is a dialog, not a typed phrase.** Naming the collectible, stating that deletion
  is permanent, and requiring a deliberate non-default action is proportionate for a single entry.
  Requiring the collector to type the collectible's name is the friction appropriate to bulk or
  account-level destruction and is not used here.
- **Stale edits are refused rather than merged.** The constitution requires that collection
  information MUST NOT be silently lost or overwritten. Last-write-wins silently discards the earlier
  edit, so a save from a stale view is refused and the collector is told (FR-027). Field-level merging
  is deliberately not attempted.
- **Deletion is immediate and permanent.** There is no trash, no retention window, and no restore. A
  collector who deletes something and wants it back must add it again. Because there is no undo, the
  operation leaves a server-side record (FR-040) so that a collector reporting something missing can
  be answered.
- **The four collection statuses are unchanged** from feature 001, and this feature introduces no new
  status and no meaning attached to the order of status changes.
- **Photograph rules are unchanged** from feature 001: JPEG, PNG and WebP, at most 10 MB, at most one
  photograph per collectible, framed consistently in the gallery.
- **Authentication already exists.** Features 004 and 005 established collector accounts and sessions;
  this feature relies on them and adds no authentication behaviour.
- **Photographs uploaded but never attached to a collectible** — an upload abandoned before any save —
  are a pre-existing gap from feature 001 and are not addressed here. This feature is responsible for
  photographs it orphans by replacement, removal, or deletion.

## Out of Scope

- Undo of an edit or a deletion.
- A trash, archive, or recycle bin holding deleted collectibles, and restoring from one.
- Edit history, revision history, or a collector-visible audit trail of who changed what and when.
  The server-side record of deletions required by FR-040 is an operational trace, not a product
  feature, and holds no collection content.
- Bulk editing and bulk deleting of multiple collectibles in one action.
- Deleting a collector's entire account or vault.
- Merging two entries a collector considers duplicates into one.
- Any change to how collectibles are added or browsed, beyond what editing and deleting require.
