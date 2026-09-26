# Feature Specification: Account Recovery

**Feature Branch**: `24-account-recovery`

**Created**: 2026-09-22

**Status**: Draft

**Issue**: #24 · **Depends on**: #15 (authentication, merged)

**Input**: Verifying a collector's email address, and resetting a forgotten password.

## Why this exists

Authentication shipped with no way back in. A collector who forgets their password is locked out of
their vault permanently — there is no reset, and no channel to reset through.

That was stated rather than hidden: feature 004's Assumptions record it, and it is why the sign-in
rate limit deliberately never permanently locks an account. A lockout with no recovery would be
unrecoverable.

**The two capabilities are one feature.** A reset link is a way into a vault, so it may only be
sent to an address somebody has proven they control. Addresses are currently unverified — anyone
can register using anyone else's — so shipping reset alone would mean mailing a way into a
stranger's collection to an address they never claimed. Verification comes first; reset depends on
it.

## Clarifications

### Session 2026-09-22

- Q: FR-020 requires limiting recovery messages but names no number. What should the limit be? → A: 3 per address per hour, with verification and reset counted separately so exhausting one cannot block the other. The refusal lifts on its own and never becomes permanent.
- Q: Does completing a password reset mark an unverified address as verified? → A: Yes. Following a link sent to that address proves control of the inbox, which is what verification tests — and the reset link is the shorter-lived, higher-bar of the two.
- Q: What happens after a verification link is followed, often on a device with no session? → A: Confirm the address is verified, then send them onward — to their vault if they already have a session, to sign-in if not. Verification never grants access by itself.
- Q: Can FR-004 and FR-006 hold for verification tokens? → A: No. They are self-contained and nothing is stored, so there is no spend to record and nothing to invalidate. Both narrow to reset links, where the consequence of a replay is account takeover rather than nothing. Decided during implementation; see plan.md Complexity Tracking.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Prove the address is mine (Priority: P1)

A collector registers, receives a message at the address they gave, follows the link, and their
account is marked as verified. Until they do, Vaultory says so and offers to send it again.

**Why this priority**: Nothing else in this feature is safe without it. A reset sent to an
unproven address is a way into a vault handed to whoever happens to own it.

**Independent Test**: Register, follow the link from the sent message, and confirm the account
moves from unverified to verified — and that an unfollowed link leaves it unverified.

**Acceptance Scenarios**:

1. **Given** a collector who has just registered, **When** they open their vault, **Then** they
   are told the address is unverified and offered a way to send the message again.
2. **Given** a verification link, **When** the collector follows it while signed in, **Then** the
   account is marked verified, they are told so, and they reach their collection.
3. **Given** a verification link opened in a browser with no session — a phone, or a different
   machine — **When** it is followed, **Then** the account is marked verified and they are offered
   sign-in rather than being signed in.
4. **Given** a verification link that has already been used, **When** it is followed again,
   **Then** it is refused and the account's state does not change.
5. **Given** a verification link older than its lifetime, **When** it is followed, **Then** it is
   refused and a fresh one can be requested.
6. **Given** an unverified collector, **When** they ask for the message again, **Then** a new link
   is sent and any previous one stops working.

---

### User Story 2 - Get back into my vault (Priority: P1)

A collector who cannot remember their password asks for a reset, receives a link, chooses a new
password, and is signed in to their own collection.

**Why this priority**: This is the hole the feature exists to close. Without it a forgotten
password means a collection is gone.

**Independent Test**: Request a reset for a registered address, follow the link, set a new
password, and sign in with it — and confirm the old password no longer works.

**Acceptance Scenarios**:

1. **Given** a registered, verified collector, **When** they request a reset, **Then** a message is
   sent to their address containing a link.
2. **Given** a reset link, **When** the collector follows it and chooses a new password, **Then**
   they can sign in with the new one and not with the old.
3. **Given** a reset request for an address with no account, **When** it is submitted, **Then** the
   response is indistinguishable from one for an address that has an account.
4. **Given** a reset link that has been used, **When** it is followed again, **Then** it is refused.
5. **Given** a reset link older than its lifetime, **When** it is followed, **Then** it is refused
   and a fresh one can be requested.
6. **Given** a new password that does not meet the minimum, **When** it is submitted, **Then** it
   is refused with a message saying what is required, and the link remains usable.
7. **Given** a collector whose address was never verified, **When** they complete a reset, **Then**
   the address is marked verified and they are no longer told it is unverified.

---

### User Story 3 - A reset actually evicts whoever prompted it (Priority: P2)

A collector who resets their password because they fear someone else has it finds that the other
person's access ends immediately.

**Why this priority**: A reset that leaves existing sessions working does not do the thing people
reset passwords for. It is the difference between changing a lock and ordering a new key.

**Independent Test**: Sign in on two devices, reset the password from one, and confirm the other
can no longer reach the collection.

**Acceptance Scenarios**:

1. **Given** a collector signed in on more than one device, **When** the password is reset,
   **Then** every session other than the one completing the reset stops working immediately.
2. **Given** a session invalidated by a reset, **When** it is used, **Then** no collection content
   is returned.

---

### Edge Cases

- A reset is requested for an address that has an account but has never been verified.
- Several reset links are requested in quick succession; older ones must not remain usable.
- A verification link and a reset link for the same account are outstanding at once.
- A reset link is followed by someone who is already signed in as a different collector.
- A token is guessed, truncated, altered, or replayed from another account.
- A reset is completed while the same collector is midway through adding a collectible elsewhere.
- Sending fails, so no message arrives.
- Reset requests are made repeatedly for the same address to generate mail to it.

## Requirements *(mandatory)*

### Verifying an address

- **FR-001**: The system MUST send a verification message when an account is created.
- **FR-002**: The system MUST mark an account verified when a valid, unused, unexpired verification
  link is followed, and MUST tell the collector it has done so.
- **FR-002a**: Following a verification link MUST NOT by itself establish a session. A collector
  who already has one is sent to their collection; one who does not is offered sign-in. Proving
  control of an inbox is not proving knowledge of a password, and the link lives for 24 hours —
  long enough that treating it as a way in would make a forwarded or archived message a way into a
  vault.
- **FR-003**: A verification link MUST expire no more than 24 hours after it is issued.
- **FR-004**: A **reset** link MUST be usable once; a second use MUST be refused (see FR-011). A
  **verification** link is exempt, and this is a narrowing made during implementation rather than a
  requirement quietly dropped. Verification tokens are self-contained and nothing is stored when
  one is issued, so there is no record of a spend to keep. The consequence is specific and small: a
  replayed verification link sets a boolean that is already true. The consequence for a reset link
  would be account takeover, which is why that half is enforced.
- **FR-005**: The system MUST show an unverified collector that their address is unverified, and
  MUST let them request the message again.
- **FR-006**: Requesting a new **reset** link MUST invalidate any previous one for that account
  (see FR-012). A previous **verification** link stays valid until it expires, for the same reason
  as FR-004: nothing is stored to invalidate. A stale verification link grants nothing — it
  re-confirms an address its holder has already proven they control.

### Resetting a password

- **FR-007**: A collector MUST be able to request a password reset using their email address,
  without being signed in.
- **FR-008**: The response to a reset request MUST be the same whether or not the address has an
  account, so the form cannot be used to discover who has one.
- **FR-009**: The system MUST send a message containing a reset link only to an address that has an
  account.
- **FR-010**: A reset link MUST expire no more than 1 hour after it is issued. It is a way into a
  vault, so it lives for less time than a verification link.
- **FR-011**: A reset link MUST be usable once. A second use MUST be refused.
- **FR-012**: Requesting a new reset MUST invalidate any previous outstanding reset link for that
  account.
- **FR-013**: A new password MUST meet the same minimum the system requires at registration, and a
  refusal MUST say what is required without consuming the link.
- **FR-014**: Completing a reset MUST sign the collector in to their own collection.
- **FR-014a**: Completing a reset MUST mark the address verified if it was not already. Following a
  link sent to that address proves control of the inbox, which is precisely what verification
  tests; asking them to prove it again afterwards would be asking for something they have just
  demonstrated.

### Tokens are credentials

- **FR-015**: Verification and reset tokens MUST NOT be recoverable from stored data. Holding the
  stored form MUST NOT be enough to use one.
- **FR-016**: Tokens MUST NOT appear in logs, in error messages, or in anything shown to a
  collector other than the link itself.
- **FR-017**: A token that is expired, already used, altered, or issued for a different account
  MUST be refused, and the refusals MUST be indistinguishable from one another.

### What a reset does to access

- **FR-018**: Completing a reset MUST end every session for that account except the one completing
  it, and those sessions MUST stop returning collection content immediately.
- **FR-019**: The old password MUST stop working the moment a reset completes.

### Not being negligent

- **FR-020**: The system MUST refuse a fourth verification message, and separately a fourth reset
  message, for the same address within an hour, and MUST say so when it refuses. The two are
  counted separately: somebody who has exhausted verification resends must still be able to request
  a reset, which is the moment they are most likely to need one. The refusal MUST lift without
  intervention and MUST NOT become permanent.
- **FR-021**: The system MUST record that recovery events happened — requested, sent, completed,
  refused — without recording tokens or passwords.

### Sending messages

- **FR-022**: The system MUST deliver messages in production through a real mail service.
- **FR-023**: In local development the system MUST let a developer read every sent message without
  registering for a third-party service and without mail leaving the machine.
- **FR-024**: A message MUST state what it is for, who it is for, how long the link lasts, and what
  to do if they did not ask for it.

### What must not change

- **FR-025**: No recovery flow may establish a session by any route other than the one the service
  already verifies. The service that owns collection data MUST continue to verify every session
  itself and MUST NOT accept an identity asserted by the presentation layer.
- **FR-026**: Every request for collection data MUST remain scoped to the collector the verified
  session identifies.

### Key Entities

- **Verification token**: Evidence that whoever holds it can read mail sent to an account's
  address. Has an issue time, an expiry, and can be spent exactly once.
- **Reset token**: Evidence that whoever holds it may choose a new password for one account. Same
  shape, shorter life, and higher consequence — it is a way into a vault.
- **Account**: Gains a verified state. Otherwise unchanged from feature 004.

## Success Criteria *(mandatory)*

- **SC-001**: A collector who has forgotten their password can reach their collection again in
  under 5 minutes without anyone intervening.
- **SC-002**: A collector can verify their address in under 2 minutes of registering.
- **SC-003**: Reset requests for an address with an account and one without are indistinguishable
  in status, body, and timing, in 100% of attempts.
- **SC-004**: An expired, used, altered, or foreign token is refused in 100% of attempts, and no
  refusal distinguishes which of those it was.
- **SC-005**: After a reset, a session established before it returns no collection content, in 100%
  of attempts.
- **SC-006**: After a reset, the old password is refused in 100% of attempts.
- **SC-012**: A collector whose address was unverified is verified after completing a reset, and
  is not asked to verify again.
- **SC-013**: Following a verification link in a browser with no session verifies the address and
  returns no collection content.
- **SC-007**: No token appears in any log or error message, verified by inspecting output rather
  than by assertion.
- **SC-008**: A developer can read every message a local Vaultory sends, with no third-party
  account and no mail leaving the machine.
- **SC-009**: The stored form of a token cannot be used in place of the token.
- **SC-010**: Recovery and verification are completable by keyboard alone, and every failure is
  announced to assistive technology.
- **SC-011**: The fourth verification request for one address within an hour is refused with a
  message saying when to retry, while a reset request for the same address in the same hour still
  succeeds.

## Out of Scope

Changing an email address, account deletion, social login, SSO, multi-factor authentication, and
magic-link sign-in as a replacement for passwords. Administrative password resets, account
lockout review, and any notion of a support operator.

Requiring verification before a collector may use their vault is also out of scope: an unverified
collector is told, and can still add and browse collectibles.

## Assumptions

- **An unverified collector may still use Vaultory.** Verification gates recovery, not access.
  Blocking a vault behind an email round trip would punish the collector for a risk that is ours to
  manage, and the address is only load-bearing when it is used to recover an account.
- A reset requested for an unverified address still sends, because refusing would leave the
  collector with no route at all and no way to earn one — they cannot sign in to trigger a resend
  either, so the account would be permanently unreachable, which is the hole this feature exists to
  close. The shorter reset lifetime is what limits the exposure, and FR-014a means one round trip
  settles both.
- 24 hours for verification and 1 hour for reset are conventional defaults, chosen because a reset
  link is worth more to an attacker than a verification link. They are not derived from a stated
  requirement.
- Messages are plain and transactional. There is no template system, no marketing, and no tracking.
- **Rate limits are keyed by network address, not by account.** FR-027 in feature 004 describes
  ten failed sign-ins *per account*; the library counts per address, so several collectors behind
  one connection share a budget while an attacker with many addresses does not. Found while
  building this feature, recorded rather than quietly accepted. It is stricter than specified for
  shared networks and looser for a determined attacker, and correcting it is a change to feature
  004 rather than to this one.
- The architecture is unchanged from feature 004 and deliberately not restated here. Issue #15
  records it, and `plan.md` is where libraries, token storage, and mail transport belong.
