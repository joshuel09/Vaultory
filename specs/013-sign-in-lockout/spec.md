# Feature Specification: Lock Sign-In per Email Address After Repeated Failures

**Feature Branch**: `99-account-sign-in-lockout`

**Created**: 2026-11-20

**Status**: Draft

**Input**: Issue #99 — sign-in throttling is counted per network address, not per account. The
company security guideline asks for a temporary lock after five consecutive failures, without
revealing whether an account exists. Amends feature 004 FR-027.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Guessing a password stops after five tries (Priority: P1)

Someone tries passwords against a collector's email address. After five wrong attempts in a row,
sign-in for that address is refused for 15 minutes, whatever password is tried and from whichever
network.

**Why this priority**: This is the protection itself. Today the limit follows the network address,
so an attacker who changes address is never stopped, and the account's owner is never protected.

**Independent Test**: Make five failed sign-ins for one address from different network addresses;
the sixth is refused even with the correct password, and succeeds once 15 minutes have passed.

**Acceptance Scenarios**:

1. **Given** an account with no recent failures, **When** five sign-ins with a wrong password are
   made in a row, **Then** the next sign-in for that address is refused with the "too many
   attempts" message, even with the right password.
2. **Given** the five failures came from five different network addresses, **When** a sixth is
   made, **Then** it is refused the same way.
3. **Given** sign-in is refused for an address, **When** 15 minutes have passed since the lock
   began, **Then** the right password signs in, without anyone intervening.
4. **Given** four failures in a row, **When** the right password is used, **Then** sign-in succeeds
   and the count starts again from zero.

---

### User Story 2 - Nobody learns which addresses have accounts (Priority: P1)

Someone who does not know whether an address is registered tries it. Every response they can see
is the same as for a registered address: wrong passwords look the same, the lock arrives after the
same number of attempts, and the locked message is the same.

**Why this priority**: A lock that only appears for registered addresses would turn the protection
into a way to list accounts. The guideline asks for both together.

**Independent Test**: Run the same sequence against a registered and an unregistered address and
compare every response the client receives; they must be identical, apart from a successful
sign-in, which only a registered address can have.

**Acceptance Scenarios**:

1. **Given** an address with no account, **When** five sign-ins are attempted, **Then** each
   receives the same response a registered address receives for a wrong password.
2. **Given** that address, **When** a sixth sign-in is attempted, **Then** it receives the same
   locked response a registered address receives.

---

### User Story 3 - A locked-out collector can get back in (Priority: P2)

A collector who has forgotten their password and locked themselves out resets it. The reset ends
the lock, so they can sign in straight away with the new password.

**Why this priority**: Feature 004 refused to lock accounts because there was no way back in.
Password reset now exists, and it has to be that way back.

**Independent Test**: Lock an address, complete a password reset for its account, then sign in
with the new password immediately.

**Acceptance Scenarios**:

1. **Given** sign-in is locked for an address, **When** its account completes a password reset,
   **Then** sign-in with the new password succeeds without waiting.
2. **Given** sign-in is locked, **When** a reset is only requested and not completed, **Then** the
   lock stays.

---

### Edge Cases

- The same address typed with different capitals or surrounding spaces counts as one address.
- Several failed attempts arriving at the same moment are all counted; none is lost.
- A failure during the lock is not counted again and does not extend the lock.
- After the lock lifts, the count starts from zero: one more wrong password does not lock again.
- A sign-in refused because the email address is not yet verified is not a password failure and
  does not count; it already has its own message (feature 004).
- A restart or a second instance of the service does not reset or split the count.
- The per-network ceiling still applies on its own, as abuse protection; whichever limit is reached
  first refuses.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST count consecutive failed sign-ins for each email address, whether or
  not an account exists for it, after normalizing the address (case, surrounding whitespace).
- **FR-002**: After 5 consecutive failures, the system MUST refuse sign-in for that address for 15
  minutes, with any password and from any network.
- **FR-003**: The refusal MUST lift on its own after 15 minutes, and the count MUST then start from
  zero.
- **FR-004**: A successful sign-in MUST reset the count to zero.
- **FR-005**: Completing a password reset MUST end any lock on the account's address and reset its
  count; requesting a reset MUST NOT.
- **FR-006**: Every response a client can observe — status, message, and the number of attempts
  before the lock — MUST be the same for an address with no account as for one with an account,
  except a successful sign-in.
- **FR-007**: The locked response MUST say plainly that there have been too many attempts and that
  the collector can try again later or reset their password, without saying whether the address
  has an account.
- **FR-008**: The count MUST be kept so that restarts and multiple running instances share it.
- **FR-009**: The system MUST NOT store the email address in readable form for this purpose.
- **FR-010**: Sign-ins refused for an unverified email address MUST NOT count as failures.
- **FR-011**: The per-network limit on sign-in attempts MUST remain, as abuse protection, and stay
  configurable for the browser test suite.
- **FR-012**: Feature 004's FR-027 MUST be amended to this behaviour, and the explanation beside the
  sign-in limit in the code MUST be corrected — it says an account is never locked because there is
  no password reset, which is no longer true.

### Key Entities

- **Sign-in failure record**: one per email address that has failed recently — a one-way hash of
  the normalized address, the number of consecutive failures, and when sign-in is refused until.
  Holds nothing that identifies the person in readable form.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The sixth consecutive wrong-password attempt for an address is refused in 100% of
  runs, whether the attempts came from one network address or many.
- **SC-002**: For a registered and an unregistered address, the sequence of responses across six
  failed attempts is identical in 100% of runs.
- **SC-003**: A collector who completes a password reset while locked signs in with the new
  password on the first attempt.
- **SC-004**: A lock lifts within 15 minutes without intervention, and the first attempt after it
  with the right password succeeds.
- **SC-005**: Of 20 failed attempts sent concurrently for one address, all are counted.
- **SC-006**: 0 email addresses are stored in readable form by this feature.

## Assumptions

- Five and fifteen minutes come from the company security guideline and feature 004's existing
  lock duration; both are fixed values, not per-collector settings.
- An attacker can still try five passwords against every address every 15 minutes; slowing that
  further (CAPTCHA, longer locks for repeat offenders) is out of scope.
- Locking an address can be used to keep its owner out for 15 minutes at a time. Password reset is
  the way back (User Story 3); emailing the owner when their address is locked is out of scope.
- The email address is the only identifier sign-in accepts, so counting per address is counting per
  account wherever an account exists.

## Out of Scope

- CAPTCHA, device recognition, and reputation scoring.
- Emailing the account holder about failed attempts or locks.
- Any screen showing failed attempts.
