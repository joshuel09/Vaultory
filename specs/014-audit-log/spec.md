# Feature Specification: An Audit Log of Account and Collection Changes

**Feature Branch**: `100-audit-log`

**Created**: 2026-11-21

**Status**: Draft

**Input**: Issue #100 — nothing durably records who signed in, or who created, edited or deleted
a collectible. The company security guideline asks for sign-ins, permission changes and data
changes to be recorded as an audit log, without passwords or personal information.

## Clarifications

### Session 2026-11-21

- Q: Is the request's network address recorded? → A: No. The guideline forbids personal
  information in logs, and a network address can identify a person. The account, the action, the
  object and the time are what an audit needs.
- Q: How long are entries kept? → A: One year, then removed in bounded batches.
- Q: If a collection change cannot be recorded, does the change still happen? → A: No. The change
  and its record are one unit: neither without the other.
- Q: If an account event cannot be recorded, is the collector refused? → A: No. A failure to
  record is reported to the operator, and signing in still works — the audit must not become a way
  to lock everyone out.
- Q: Restoring from and emptying the Trash (feature 010)? → A: Recorded the same way once that
  feature is merged; this feature covers what `main` has.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Account events leave a trail (Priority: P1)

An operator investigating a report — "someone was in my vault" — can see when the account was
registered, every sign-in and failed sign-in, sign-outs, email verification and password resets,
in order.

**Why this priority**: Sign-in is where an intrusion starts; it is the record the guideline names
first.

**Independent Test**: Register, fail a sign-in, sign in, reset the password and sign out; the log
holds those events in that order, for that account, with no password, token or email address in
any of them.

**Acceptance Scenarios**:

1. **Given** a new registration, **Then** an entry records it for the new account.
2. **Given** a failed sign-in for a registered address, **Then** an entry records the failure
   without the address in readable form; **given** one for an address with no account, **Then** the
   entry looks the same.
3. **Given** a sign-in refused because the address is locked (feature 013), **Then** an entry
   records the refusal.
4. **Given** a successful sign-in, sign-out, email verification or completed password reset,
   **Then** an entry records it for the account.

---

### User Story 2 - Collection changes leave a trail (Priority: P1)

The operator can see when each collectible was created, edited and deleted, and by which account.

**Why this priority**: The guideline names data changes; a collection that changed unexpectedly is
the other report the log must answer.

**Independent Test**: Create, edit and delete a collectible; the log holds three entries for that
collectible and that account, and none of its names, notes or prices.

**Acceptance Scenarios**:

1. **Given** a collectible is created, edited or deleted, **Then** one entry records the action,
   the account and the collectible.
2. **Given** a change that is refused (validation, a version conflict, someone else's
   collectible), **Then** nothing is recorded as a change.
3. **Given** the entry cannot be written, **Then** the change does not happen either.

---

### User Story 3 - The record cannot be rewritten (Priority: P2)

Entries can be added and, after a year, removed — never edited, and never removed early, even by
the services that write them.

**Why this priority**: A log the application could rewrite is not evidence.

**Independent Test**: As each service's database role, try to change an entry and to delete a
recent one; both are refused. A year-old entry is removed by the retention step.

**Acceptance Scenarios**:

1. **Given** any entry, **When** a service tries to change it, **Then** the database refuses.
2. **Given** an entry younger than a year, **When** a service tries to delete it, **Then** the
   database refuses.
3. **Given** entries older than a year, **When** the retention step runs, **Then** they are
   removed in bounded batches.

### Edge Cases

- A failed sign-in for an address with no account has no account to name; its entry carries the
  same one-way key feature 013 uses, so repeated failures for one address can be told apart without
  the address.
- A deletion of a collectible that no longer exists, or belongs to someone else, is not a change
  and is not recorded as one.
- The frontend's restricted database role may add entries and do nothing else with the log —
  not read it, not change it.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST record, for each of these events, when it happened, what happened,
  and its outcome: registration; sign-in succeeded, failed, or refused because the address is
  locked; sign-out; email verified; password reset completed; collectible created, edited and
  deleted.
- **FR-002**: Each entry MUST name the account when there is one, and the collectible for
  collection events.
- **FR-003**: No entry MUST contain a password, a token, an email address in readable form, a
  network address, or any collectible's text, price or image.
- **FR-004**: A collection change and its entry MUST be recorded together or not at all.
- **FR-005**: A failure to record an account event MUST be reported to the operator and MUST NOT
  refuse the collector.
- **FR-006**: Entries MUST NOT be changeable by any service, and MUST NOT be deletable until they
  are a year old; the database MUST enforce this, not the application.
- **FR-007**: Entries older than a year MUST be removed in bounded batches, without a dedicated
  schedule.
- **FR-008**: The frontend's restricted database role MUST be able to add entries and nothing else
  with the log.

### Key Entities

- **Audit entry**: when, the action, the outcome, the account (if any), the object (a collectible,
  or the one-way key of an address), and nothing else.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Each event in FR-001 produces exactly one entry, in 100% of runs.
- **SC-002**: 0 entries contain a password, token, readable email address, network address, or
  collectible text, price or image, checked against the full log after a run with distinctive
  values.
- **SC-003**: 100% of attempts by a service role to change an entry, or to delete one younger than
  a year, are refused by the database.
- **SC-004**: 0 collection changes are applied without their entry.

## Assumptions

- Reading the log is done by an operator through the database; no screen is part of this feature.
- Trash events (restore, delete permanently) follow once feature 010 is merged.

### Current Trash lifecycle

Audit records distinguish moving to Trash, restoring, permanent deletion, and expiry
purge. Each successful transition and its audit record commit atomically. A missing,
foreign, already transitioned, or expired resource must not create a success event.
No content or identifying text is added to these records.
