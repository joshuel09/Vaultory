# Feature Specification: Undo and a Recoverable Trash

**Feature Branch**: `31-recoverable-trash`

**Created**: 2026-10-25

**Status**: Draft

**Input**: Issue #31 — "Undo and a recoverable trash for deleted collectibles." Feature 006 made
deleting possible and permanent: the collectible and its photograph are gone in one transaction, and
a collector who deletes the wrong statue can only add it again from memory. This feature makes the
vault forgiving. Done when: a collector who deletes something by mistake can get it back, including
its photograph, within a stated window.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Undo a deletion immediately (Priority: P1)

A collector deletes a collectible and, a moment later, realises it was the wrong one. Right where they
land after the deletion, Vaultory tells them it was moved to the Trash and offers to undo. One action
puts it back exactly as it was — every attribute, its photograph, and its place in the gallery.

**Why this priority**: Most mistaken deletions are noticed immediately. This is the cheapest path back
and the one most collectors will use.

**Independent Test**: Delete a collectible with a photograph; on the gallery, choose Undo; the
collectible is back in its original gallery position with its photograph and every attribute
unchanged.

**Acceptance Scenarios**:

1. **Given** a collector has just deleted a collectible, **When** the gallery appears, **Then** it says
   the collectible was moved to the Trash and offers Undo and a way to the Trash.
2. **Given** that notice, **When** the collector chooses Undo, **Then** the collectible is restored with
   its photograph and attributes unchanged, in the same gallery position it held before deletion.
3. **Given** that notice, **When** the collector does nothing and keeps browsing, **Then** the
   collectible stays in the Trash and the notice does not reappear on later pages.

---

### User Story 2 - Recover from the Trash within 30 days (Priority: P1)

A collector notices days later that something is missing. They open the Trash, see what they deleted,
newest first, with its photograph and how long it will be kept, and restore it.

**Why this priority**: The issue's done-condition: recovery within a stated window, not only in the
moment.

**Independent Test**: Delete a collectible; open the Trash; it is listed with its photograph and the
date it will be removed for good; restore it; it is back in the gallery in its original position and
gone from the Trash.

**Acceptance Scenarios**:

1. **Given** collectibles deleted within the last 30 days, **When** the collector opens the Trash,
   **Then** each is listed, most recently deleted first, with its name, photograph, and the date it will
   be removed for good.
2. **Given** a collectible in the Trash, **When** the collector restores it, **Then** it returns to the
   gallery, every status filter, the collection statistics and the breakdowns exactly as before, and
   leaves the Trash.
3. **Given** a collectible deleted more than 30 days ago, **When** the collector opens the Trash,
   **Then** it is not listed and cannot be restored.
4. **Given** an empty Trash, **When** the collector opens it, **Then** it says so and explains that
   deleted collectibles are kept for 30 days.

---

### User Story 3 - Delete something for good, now (Priority: P2)

A collector wants a collectible gone completely — its photograph included — without waiting 30 days.
From the Trash they choose to delete it permanently, confirm, and it is destroyed as feature 006
destroyed things.

**Why this priority**: The Trash must not take away the ability to make something truly gone, which
feature 006 guaranteed and some collectors will want for privacy.

**Independent Test**: Move a collectible with a photograph to the Trash; delete it permanently; it is
no longer in the Trash, cannot be restored, and its photograph is no longer retrievable by anyone,
including its owner.

**Acceptance Scenarios**:

1. **Given** a collectible in the Trash, **When** the collector chooses Delete permanently, **Then** a
   confirmation names it and says this cannot be undone; dismissing the confirmation or pressing Enter
   on it does not delete.
2. **Given** the collector confirms, **When** the deletion completes, **Then** the collectible and its
   photograph are destroyed with feature 006's guarantees: the photograph is unretrievable at once and
   its files are removed, retried if storage fails.

---

### Edge Cases

- **Opening or editing a collectible that is in the Trash**: it is not in the vault, so it answers
  not-found, exactly as a deleted collectible did before this feature. An edit form left open across
  a deletion is refused as not-found rather than resurrecting the collectible.
- **Restoring something already restored, permanently deleted, expired, never deleted, or another
  collector's**: refused as not-found, with no indication of which. Restore is not idempotent-success:
  the collector needs to know it did not come back.
- **Deleting twice, or deleting something already in the Trash**: a success with no further effect,
  as in feature 006 (006 FR-025). The 30 days are counted from the first deletion.
- **Permanently deleting twice, or something not in the Trash**: a success with no effect, matching
  006 FR-025.
- **An add retried with a submission key after its collectible was moved to the Trash**: creates a new
  collectible, as 006 does when the original was deleted; it does not return the trashed one into the
  gallery.
- **A restore and the 30-day expiry at the same moment**: one wins. Either the collectible is restored
  and kept, or it is removed for good and the restore is refused. Never both, never half.
- **The photograph of a trashed collectible**: kept, and still fetchable by its owner so the Trash can
  show it; never by anyone else. Feature 009's sweep of unattached uploads leaves it alone because the
  trashed collectible still references it.
- **A collector deletes their account**: everything, Trash included, goes with it, as today.
- **A large Trash**: listed in pages, like the gallery.

## Requirements *(mandatory)*

### Functional Requirements

#### Moving to the Trash

- **FR-001**: Deleting a collectible (006 FR-022) MUST move it to the Trash instead of destroying it.
  Its attributes, photograph, and gallery position MUST be retained unchanged while it is there.
- **FR-002**: A collectible in the Trash MUST NOT appear in, or be counted by, the gallery, any status
  filter, the collection statistics, the breakdowns, or any other view of the vault, and MUST answer
  not-found to being opened or edited.
- **FR-003**: The deletion confirmation (006 FR-023) MUST say the collectible will be moved to the
  Trash and kept for 30 days, instead of saying the deletion cannot be undone.
- **FR-004**: Deleting something that is not in the vault — absent, already in the Trash, or another
  collector's — MUST remain a success with no effect (006 FR-025), and MUST NOT restart its 30 days.

#### Getting it back

- **FR-005**: Immediately after a deletion, the collector MUST be offered Undo and a way to the Trash,
  on the screen they land on.
- **FR-006**: The Trash MUST list the collector's own collectibles deleted within the last 30 days,
  most recently deleted first, each with its name, photograph, and the date it will be removed for
  good, in pages.
- **FR-007**: Collectors MUST be able to restore a collectible from the Trash. Restoring MUST return it
  with every attribute, its photograph, its version, and its original gallery position unchanged.
  Undo (FR-005) MUST be the same operation.
- **FR-008**: Restoring MUST be refused as not-found for anything not in the acting collector's Trash
  within the window, without revealing which case applies.

#### Retention and permanent deletion

- **FR-009**: A collectible MUST be kept in the Trash for 30 days from its deletion, a single named
  value, and MUST NOT be restorable or listed after that.
- **FR-010**: Collectibles past their 30 days MUST be destroyed with feature 006's guarantees (006
  FR-020, FR-020a): the collectible and photograph rows removed transactionally, the files removed
  afterwards and retried on failure. This MUST run without a dedicated schedule — on each deletion and
  at service start, bounded per run — and MUST NOT be able to fail a collector's deletion.
- **FR-011**: Collectors MUST be able to delete a collectible in the Trash permanently, behind a
  confirmation that names it, says it cannot be undone, and does not delete when dismissed or when
  Enter is pressed on it. Permanent deletion MUST destroy it with the guarantees in FR-010, at once.
- **FR-012**: A restore racing the expiry of the same collectible MUST end with exactly one outcome:
  restored and kept, or destroyed and the restore refused. This MUST be guaranteed by the persistence
  layer.

#### Privacy and records

- **FR-013**: Every Trash operation MUST be scoped to the acting collector, enforced server-side; another
  collector's trashed collectible MUST be indistinguishable from an absent one.
- **FR-014**: A trashed collectible's photograph MUST remain retrievable by its owner only, for as long
  as the collectible is in the Trash.
- **FR-015**: Moving to the Trash, restoring, permanent deletion, and expiry MUST each be recorded
  operationally with identifiers only — collector, collectible, and when — never content, extending
  006 FR-040 and FR-041.
- **FR-016**: Only deletions are undoable. Edits are not recorded for undo; this feature adds no edit
  history.

- **FR-018**: A photograph MUST be referenced by at most one collectible, enforced by the persistence
  layer. A save naming a photograph another collectible already uses — including one in the Trash —
  MUST be refused with a field-level message on the photograph. Without this, destroying one
  collectible (FR-010, FR-011) would silently remove the photograph of another that shares it.

#### Amendments to feature 006

- **FR-017**: 006 FR-020, FR-022 and FR-023, and SC-005, MUST be amended in feature 006's spec to say
  that a deletion moves a collectible to the Trash, and that the guarantees they state now apply when
  it leaves the Trash for good.

### Key Entities

- **Trashed collectible**: a collectible the collector deleted, with the moment it was deleted. Same
  attributes, photograph and position as before; invisible to the vault; restorable until 30 days have
  passed.
- **Trash**: the collector's own list of trashed collectibles within the window.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A collector can undo a deletion in one action from the screen they land on after it.
- **SC-002**: 100% of collectibles restored within 30 days come back with every attribute, their
  photograph, and their original gallery position.
- **SC-003**: 0 trashed collectibles appear in or are counted by any view of the vault.
- **SC-004**: 0 collectibles are restorable after 30 days, and 100% of collectibles past 30 days are
  destroyed with their photographs within one run of the expiry after becoming eligible, given a
  backlog no larger than one run's bound.
- **SC-005**: 100% of permanently deleted collectibles' photographs are unretrievable immediately,
  including by their owner.
- **SC-006**: 0 Trash operations reveal whether another collector's collectible exists.

## Clarifications

### Session 2026-10-25

The issue posed four questions for specification. Answered here with reasons; each is open to review.

- Q: A trash with a retention window, or a short-lived undo? → A: Both, as one mechanism. Undo is a
  restore offered at the moment of deletion; the Trash is the same restore offered for 30 days.
  Two mechanisms would mean two sets of guarantees for one intent.
- Q: How long is the window? → A: 30 days — a common, explainable period, long enough to notice a
  missing piece on the next visit, short enough that "delete" still means something.
- Q: Does restoring put a collectible back in its original gallery position? → A: Yes. The row is
  preserved, and `created_at` is never written by an edit, so its position is preserved by
  construction.
- Q: Does undo extend to edits? → A: No. Edits are recoverable by editing again; deletions are not.
- Q: Is permanent deletion still available? → A: Yes, from the Trash (US3). The Trash must not remove
  the ability to make something truly gone.

## Assumptions

- 006 Decision 4 still holds for anything leaving the Trash for good: deleting the image row is the
  privacy guarantee, and file removal is hygiene that is retried. What changes is *when* it applies.
- Feature 009 (sweep of unattached uploads, PR #57) needs no change: a trashed collectible still
  references its photograph, so the sweep leaves it alone.
- The idempotency record of an add (006) need not survive a deletion; a retried add after a deletion
  creating a new collectible is already accepted behaviour in 006.
- Spec directory 010: 009 is in use on the open branch for #30.
