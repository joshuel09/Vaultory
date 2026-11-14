# Feature Specification: Sweep Abandoned Uploads

**Feature Branch**: `30-sweep-abandoned-uploads`

**Created**: 2026-10-24

**Status**: Draft

**Input**: Issue #30 — "Sweep uploaded images that were never attached to a collectible." Uploading a
photograph and attaching it to a collectible are separate operations: the upload is stored the moment
it succeeds, before anything references it. When the collector abandons the form, replaces the
photograph before saving, or closes the tab, the upload stays forever. Feature 006 removes images it
orphans itself and explicitly left abandoned uploads out of scope. Done when: an upload abandoned
before any save stops consuming storage indefinitely, and no in-progress form ever loses its
photograph to the sweeper.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Abandoned uploads stop consuming storage (Priority: P1)

A collector starts adding a collectible, uploads a photograph, and then leaves — closes the tab,
navigates away, or picks a different photograph before saving. Some time later, Vaultory removes the
photograph nobody used, both its record and its stored files. The collector never sees this happen.

**Why this priority**: This is the defect the issue names. Without it, every abandoned form leaves up
to 10 MB of storage behind permanently, and nothing ever reclaims it.

**Independent Test**: Upload a photograph without saving a collectible. Make the upload older than
the grace period. Trigger the sweep. The photograph's record is gone, its files are gone from the
image store, and fetching it returns not-found.

**Acceptance Scenarios**:

1. **Given** an upload no collectible references, older than the grace period, **When** the sweep
   runs, **Then** the upload's record is removed and its original and rendition files are deleted.
2. **Given** a collector uploaded two photographs in one form and saved with the second, **When** the
   first becomes older than the grace period and the sweep runs, **Then** the first is removed and the
   saved collectible's photograph is untouched.
3. **Given** a swept upload whose files could not be deleted because of a storage fault, **When** a
   later sweep or deletion drain runs, **Then** deleting the files is retried until it succeeds.

---

### User Story 2 - An in-progress form never loses its photograph (Priority: P1)

A collector uploads a photograph and keeps filling in a long form. However long they take within the
grace period, the photograph is still there when they save, and the saved collectible shows it.

**Why this priority**: The issue's second done-condition, and the one that makes the sweeper safe to
ship. A sweeper that deletes a photograph someone is about to save is worse than no sweeper.

**Independent Test**: Upload a photograph and leave it unsaved for most of the grace period; save a
collectible with it; the collectible shows the photograph. Separately, race a save against a sweep of
the same upload: the outcome is one of the two permitted outcomes in FR-006, never a third.

**Acceptance Scenarios**:

1. **Given** an upload younger than the grace period, **When** the sweep runs, **Then** the upload is
   untouched.
2. **Given** an upload referenced by any collectible, **When** the sweep runs, **Then** the upload is
   untouched, regardless of its age.
3. **Given** a save referencing an upload and a sweep of that upload running at the same moment,
   **When** both finish, **Then** either the collectible was saved with its photograph and the upload
   survives, or the save was refused with a message to upload the photograph again — never a saved
   collectible whose photograph has silently disappeared.

---

### User Story 3 - A form left open past the grace period fails clearly (Priority: P2)

A collector leaves a form open with an uploaded photograph for longer than the grace period, then
tries to save. The photograph has been swept. The save is refused, nothing is half-saved, and the
form tells them to upload the photograph again.

**Why this priority**: Rare by construction, since the grace period is longer than a session stays
idle. But when it happens, it must not look like a mysterious failure.

**Independent Test**: Sweep an upload, then submit a form referencing it. The save is refused with a
field-level message on the photograph saying it is no longer available and asking for it to be
uploaded again; no collectible is created.

**Acceptance Scenarios**:

1. **Given** a form whose upload has been swept, **When** the collector saves, **Then** no collectible
   is created or changed, and the photograph field says the photograph is no longer available and
   should be uploaded again.

---

### Edge Cases

- **The sweep and a save target the same upload at once**: covered by US2 scenario 3. The sweep must
  never remove an upload that a save is in the middle of attaching.
- **An edit replaces a collectible's photograph with a fresh upload**: the fresh upload is referenced
  once the edit saves; the old one is already removed by feature 006. Before the edit saves, the fresh
  upload is an ordinary unattached upload and is protected by the grace period.
- **A very large backlog** (a service that has accumulated many abandoned uploads): each sweep is
  bounded, so no single collector's request or a start-up pays for the whole backlog. Repeated sweeps
  work through it.
- **A storage fault during file deletion**: the record is still removed (the privacy guarantee); the
  files are retried through the existing deletion queue (US1 scenario 3).
- **A collector whose account is deleted**: their uploads are removed with the account by the existing
  cascade; the sweep has nothing to add.
- **The upload preview**: a photograph's preview is served from the same record; once swept, the
  preview returns not-found, as any absent image does.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST identify uploads that no collectible references and that were uploaded
  longer ago than the grace period.
- **FR-002**: The grace period MUST be 30 days, the time a session survives without use. It MUST be
  a single named value, so changing it is one change.
- **FR-003**: For each identified upload, System MUST remove its record and queue its files for
  deletion through the same deletion queue feature 006 uses, so failed file deletions are retried
  rather than abandoned. There MUST NOT be a second deletion path.
- **FR-004**: System MUST NOT remove an upload that is referenced by any collectible, at any age.
- **FR-005**: System MUST NOT remove an upload younger than the grace period.
- **FR-006**: When a save that references an upload and a sweep of that upload run concurrently,
  System MUST end in exactly one of two states: the collectible is saved with its photograph and the
  upload survives; or the save is refused and the upload is removed. A saved collectible MUST NEVER
  lose its photograph to the sweep. This MUST be guaranteed by the persistence layer, not by timing.
- **FR-007**: A save that references a removed upload MUST be refused as a whole, with a field-level
  message on the photograph saying it is no longer available and should be uploaded again.
- **FR-008**: The sweep MUST be bounded per run, so it never makes one request or start-up do
  unbounded work.
- **FR-009**: The sweep MUST run without a dedicated background schedule: when new uploads arrive,
  and once at service start, matching how feature 006 drains its deletion queue.
- **FR-010**: The sweep MUST NOT affect any collector's ability to upload or save: a failure while
  sweeping is recorded and never turns a successful upload into a failed one.
- **FR-011**: Each run that removes anything MUST be recorded operationally with the number removed,
  without recording file contents or anything that identifies a collector beyond what deletion
  already logs.

### Key Entities

- **Upload**: a stored photograph and its derived rendition, owned by one collector, with the time it
  was uploaded. Referenced by at most one collectible.
- **Deletion queue**: feature 006's record of files whose upload record is already gone and which are
  still to be deleted from storage.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of uploads unreferenced for longer than the grace period are removed within one
  sweep of becoming eligible, given a backlog no larger than one sweep's bound.
- **SC-002**: 0 saved collectibles lose their photograph to the sweep, including when a save and a
  sweep of the same upload are run concurrently many times.
- **SC-003**: 0 uploads younger than the grace period are removed.
- **SC-004**: A storage fault during a sweep leaves 0 files permanently orphaned: every queued file is
  deleted once storage recovers.
- **SC-005**: Upload response time is not noticeably affected: a sweep adds at most one bounded batch
  of work to an upload.

## Clarifications

### Session 2026-10-24

- Q: How long is the grace period? → A: 30 days, the session's idle lifetime. No session lifetime
  strictly bounds how long a form can stay open (signing in again in another tab revives an old one),
  so the grace period cannot be a guarantee on its own. FR-006 and FR-007 carry the guarantee: within
  the grace period nothing is swept, and past it a save fails clearly rather than silently.

## Assumptions

- Feature 006's deletion queue and its retrying drain are reused unchanged in behaviour.
- Storage cost of keeping abandoned uploads for 30 days is acceptable: each is at most 10 MB, and
  abandoned uploads accrue only as fast as collectors upload.
- Uploads already orphaned before this feature ships are swept like any other once eligible; there is
  no separate backfill.
- No new user-facing screen. The only visible change is the clearer message in FR-007.
- Spec directory 009: 008 is in use.
