# Feature Specification: Collector Authentication

**Feature Branch**: `15-collector-authentication`

**Created**: 2026-09-18

**Status**: Draft

**Issue**: #15 · **Depends on**: #14 (landing page, merged)

**Input**: Collectors register with an email address and a password, sign in, stay signed in across
visits, and sign out. This replaces the development-only sign-in that currently mints a session for
anyone who asks.

## Why this exists

Feature 001 excluded authentication and assumed it would be provided. Everything beneath it was
built anyway: a collectors table, foreign keys from every collectible, `collector_id` as a
predicate on every query, and 404-never-403 for another collector's resource. The only missing
piece is deciding *which* collector a request belongs to.

Until that exists, two things are true and neither is acceptable for a real product. Anyone who can
reach the service can mint a session for a seeded collector and read that vault. And the production
image refuses to start at all, correctly, because it has no way to identify anybody — which makes
Vaultory undeployable rather than merely incomplete.

## Clarifications

### Session 2026-09-19

- Q: FR-027 requires limiting sign-in attempts but names no number. What should the limit be? → A: 10 failed attempts per account per 15 minutes, then a 15-minute lockout that lifts on its own; a successful sign-in resets the counter; the account is never permanently locked.
- Q: Is the 30-day session expiry absolute from sign-in, or does it renew with use? → A: Sliding. Each authenticated request extends expiry to 30 days from that moment, subject to a hard cap of 90 days from sign-in, after which the collector must sign in again.
- Q: What happens to the two seeded development collectors and the collectibles they own? → A: A reversible migration deletes them. They are fixtures rather than people, so nothing real is orphaned; the down migration re-seeds them.
- Q: When a signed-out visitor opens a vault page, does the URL change? → A: Yes — redirect to the sign-in page carrying the originally requested path, and return them there after signing in.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create an account and start a vault (Priority: P1)

A collector arrives from the landing page, chooses to create an account, gives an email address and
a password, and lands in their own empty vault. Nothing they add is visible to anyone else.

**Why this priority**: Without registration there are no collectors, only seeded fixtures. This is
the smallest slice that turns Vaultory from a demonstration into something a person can own.

**Independent Test**: Register a new account and confirm the vault is empty, private, and accepts a
collectible — without any development-only endpoint being available.

**Acceptance Scenarios**:

1. **Given** a visitor with no account, **When** they register with a valid email and password,
   **Then** they are signed in and see their own empty vault.
2. **Given** an email address already registered, **When** someone tries to register with it again,
   **Then** registration is refused with a clear message and no second account is created.
3. **Given** a registered collector, **When** they add a collectible, **Then** it belongs to them
   and is invisible to every other collector.
4. **Given** a password that does not meet the minimum strength rule, **When** registration is
   attempted, **Then** it is refused with a message saying what is required.

---

### User Story 2 - Sign in and stay signed in (Priority: P1)

A returning collector signs in with their email and password and sees their collection. Closing the
browser and returning later does not make them sign in again, until the session expires.

**Why this priority**: A vault that forgets you on every visit is not a vault. Registration without
sign-in delivers an account that can only ever be used once.

**Independent Test**: Sign in, close the browser, reopen it, and confirm the collection is still
reachable without re-entering credentials.

**Acceptance Scenarios**:

1. **Given** a registered collector, **When** they sign in with correct credentials, **Then** they
   reach their collection.
2. **Given** a registered collector, **When** they sign in with an incorrect password, **Then**
   sign-in is refused with a message that does not reveal whether the email exists.
3. **Given** a signed-in collector, **When** they close and reopen the browser within the session
   lifetime, **Then** they are still signed in.
4. **Given** a session that has expired, **When** the collector returns, **Then** they are asked to
   sign in and no collection content is shown.
5. **Given** a collector who visits at least once a month, **When** 30 days pass since sign-in,
   **Then** they are still signed in.
6. **Given** a session issued 90 days ago and used continuously, **When** the collector returns,
   **Then** they are asked to sign in again.

---

### User Story 3 - Sign out, and be told when you are not signed in (Priority: P2)

A collector signs out and their session stops working immediately. Anyone who reaches a vault page
without a session is told to sign in, rather than being shown an error implying the service is
broken.

**Why this priority**: Signing out matters most on a shared or borrowed device, and it is the point
at which a collector has to trust the product. The 401 state is currently actively misleading, and
its only offered action cannot help.

**Independent Test**: Sign out and confirm the previous session no longer reaches any collection
content; visit a vault page with no session and confirm the page invites sign-in.

**Acceptance Scenarios**:

1. **Given** a signed-in collector, **When** they sign out, **Then** their session no longer grants
   access to any collection content.
2. **Given** a visitor with no session, **When** they open a vault page, **Then** they arrive at
   the sign-in page rather than an error claiming the collection could not be reached.
3. **Given** a signed-out collector, **When** they press the browser's back button to a vault page,
   **Then** no collection content is served from that page.
4. **Given** a signed-out visitor who opened a deep link such as the add-collectible page,
   **When** they sign in, **Then** they arrive at the page they originally asked for.
5. **Given** a sign-in link carrying a destination outside Vaultory, **When** the collector signs
   in, **Then** they are not sent there.

---

### Edge Cases

- A request arrives carrying a session that is valid in form but was issued for a collector who no
  longer exists.
- A sign-in link carries a destination pointing at another site, or at a path crafted to look
  internal.
- A request asserts an identity directly — a header, a body field, or a query parameter naming a
  collector — while carrying no valid session, or while carrying a session for someone else.
- A session is tampered with: its payload edited, its signature replaced, or one collector's
  session presented with another's identifier.
- Registration and sign-in are attempted repeatedly and rapidly with different passwords.
- Two registrations for the same email arrive simultaneously.
- A collector signs in on one device and signs out on another.
- A development database holds collectibles owned by the seeded fixtures when the migration runs.
- The migration is rolled back after accounts already exist.

## Requirements *(mandatory)*

### Functional Requirements

**Registration and credentials**

- **FR-001**: The system MUST allow a visitor to register with an email address and a password.
- **FR-002**: The system MUST reject a registration whose email address is already registered, and
  MUST NOT create a second account for it.
- **FR-003**: The system MUST treat email addresses case-insensitively for the purpose of
  uniqueness and sign-in.
- **FR-004**: The system MUST enforce a minimum password length of 12 characters and MUST state the
  requirement when refusing.
- **FR-005**: The system MUST store passwords only in a form from which the original cannot be
  recovered, using a deliberately slow, salted password hash.
- **FR-006**: The system MUST NOT log, display, or transmit a password or password hash outside the
  act of verifying a sign-in.

**Sign-in, session, sign-out**

- **FR-007**: The system MUST allow a registered collector to sign in with their email and
  password.
- **FR-008**: The system MUST give the same refusal for an unknown email as for a wrong password,
  so that sign-in cannot be used to discover who has an account.
- **FR-009**: The system MUST keep a collector signed in across browser restarts until their
  session expires.
- **FR-010**: A session MUST expire 30 days after it was last used, and MUST expire no more than
  90 days after it was issued regardless of use. The sliding window keeps an active collector from
  being signed out for no reason they would recognise; the cap is what stops a session living
  indefinitely on a device its owner no longer controls.
- **FR-011**: The system MUST allow a signed-in collector to sign out, after which that session
  MUST NOT grant access to any collection content.
- **FR-012**: The system MUST carry the session in a way that a page's own scripts cannot read, and
  MUST NOT place it anywhere a browser would retain in a URL or history entry.

**Authorization — the boundary this feature exists to protect**

- **FR-013**: The service that owns collection data MUST verify every session itself, on every
  request, and MUST NOT accept any identity asserted by the presentation layer.
- **FR-014**: The system MUST refuse any request whose session is absent, expired, unverifiable, or
  issued for a collector that no longer exists, and MUST disclose no collection content when it
  does.
- **FR-015**: Every request for collection data MUST remain scoped to the collector the verified
  session identifies, preserving the privacy guarantees already specified in feature 001.
- **FR-016**: A collector MUST correspond to exactly one account, and an account to exactly one
  collector.
- **FR-017**: Collectibles, images, and submissions belonging to a collector with an account MUST
  remain attached to that collector once authentication is introduced, with nothing orphaned.
- **FR-017a**: The two seeded development collectors from feature 001, and everything they own,
  MUST be removed by a reversible migration. They are fixtures rather than people: every row they
  own was produced by test runs. Removing them is what makes FR-017 trivially true, and leaves no
  vault reachable without an account.

**Replacing the development stand-in**

- **FR-018**: The system MUST NOT require any development-only identity mechanism in order to run,
  and the documented way to use Vaultory MUST be registration and sign-in.
- **FR-019**: A production build MUST be able to start and serve traffic once authentication is
  configured, and MUST still refuse to start if it has no way to establish identity.
- **FR-020**: A production build MUST NOT contain any mechanism that issues a session without
  verifying credentials.

**Interface**

- **FR-021**: The system MUST present a registration page and a sign-in page, each linking to the
  other.
- **FR-022**: A request for a vault page without a valid session MUST send the visitor to the
  sign-in page, carrying the path they asked for, and MUST return them to that path once they sign
  in. This MUST be visibly distinct from the state shown when the collection genuinely cannot be
  reached — the current message blames the service for what is simply a missing session, and
  offers a retry that cannot succeed.
- **FR-022a**: The remembered path MUST be rejected unless it is a path within Vaultory itself. A
  destination taken from a request and followed after sign-in is an open redirect, which turns the
  sign-in page into a credible way to send a collector somewhere hostile.
- **FR-023**: After signing in or registering, a collector MUST arrive at their collection.
- **FR-024**: The landing page's primary call to action MUST lead to registration.
- **FR-025**: A signed-in collector MUST be able to see which account they are signed in as, and
  reach sign-out from any vault page.
- **FR-026**: Registration and sign-in MUST report failures the way the rest of Vaultory does:
  every problem at once, against the field responsible, announced to assistive technology, and
  never discarding what was typed.

**Not being negligent**

- **FR-027**: The system MUST refuse further sign-in attempts for an account after 10 failed
  attempts within 15 minutes, for 15 minutes, and MUST say so when it refuses. A successful
  sign-in MUST reset the count, the refusal MUST lift without intervention, and an account MUST
  NOT become permanently locked — password reset is out of scope, so a collector locked out
  indefinitely would have no way back in.
- **FR-028**: The system MUST record that authentication events happened — registration, sign-in,
  sign-in failure, sign-out — without recording credentials.

### Key Entities

- **Account**: What a person signs in as. Holds an email address, a password verifier, and the
  times it was created and last updated. Exactly one account per collector.
- **Collector**: The owner of a vault, already established in feature 001 and referenced by every
  collectible, image, and submission. This feature attaches an account to it and changes nothing
  else about it.
- **Session**: Evidence that a request is being made by a particular collector. Has an issue time
  and an expiry, can be ended early by signing out, and is verifiable by the service that owns
  collection data without asking the presentation layer to vouch for it.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A visitor can go from the landing page to their own empty vault in under 2 minutes,
  with no prior knowledge of Vaultory.
- **SC-002**: A returning collector reaches their collection in under 30 seconds.
- **SC-003**: A collector who closes the browser and returns within 30 days of their last visit is
  not asked to sign in again; one who returns after 30 days of not visiting, or more than 90 days
  after signing in, is.
- **SC-004**: No request carrying an absent, expired, tampered, or another collector's session
  returns any collection content, in 100% of attempts.
- **SC-005**: An identity asserted by any means other than a verified session is ignored, in 100%
  of attempts.
- **SC-006**: A wrong password and an unregistered email produce responses indistinguishable to the
  person attempting them.
- **SC-007**: After the migration, every collectible in the database is owned by a collector that
  has an account, and no collector exists without one. The seeded fixtures and their collectibles
  are gone, and the down migration restores them.
- **SC-008**: A shipped production build contains no mechanism that issues a session without
  verifying credentials, demonstrated by inspecting the build rather than by assertion.
- **SC-009**: A production build starts and serves traffic when authentication is configured.
- **SC-010**: A visitor reaching a vault page without a session arrives at the sign-in page, is
  never shown a message attributing the refusal to a problem reaching the service, and after
  signing in lands on the page they originally requested.
- **SC-013**: No destination outside Vaultory is ever followed after sign-in, in 100% of attempts.
- **SC-011**: Registration and sign-in are completable using a keyboard alone, and every failure is
  announced to assistive technology.
- **SC-012**: The 11th failed sign-in for one account within 15 minutes is refused with a message
  saying when to try again, and the same account signs in successfully once 15 minutes have
  passed.

## Out of Scope

Social login and SSO, multi-factor authentication, password reset and recovery, email address
verification, changing an email address or password, account deletion, teams or shared vaults,
administrative or moderator roles, and session management across devices ("sign out everywhere").

Rate limiting beyond FR-027 — reputation scoring, CAPTCHA, IP-based blocking — is also excluded.

## Assumptions

- Registration is open: anyone with an email address may create an account. There is no invitation
  code, waiting list, or approval step.
- Email addresses are not verified in this feature. An unverified address is accepted, which is why
  password reset is out of scope — there is no trusted channel to reset through yet. This is a
  deliberate, temporary limitation and the first thing a follow-up feature should address.
- The two seeded development collectors from feature 001 are fixtures, not real users, and are
  removed by migration (FR-017a). Anyone holding a development database loses the collectibles in
  it — which is the intended outcome, since those rows came from test runs rather than from use.
- Sessions are stored server-side so that sign-out can end one immediately, rather than waiting for
  a self-contained token to expire.
- The 30-day idle window and 90-day cap are reasonable defaults for a personal collection tool;
  they are not derived from a stated requirement. See Clarifications.
- The existing `collectors` table and every foreign key built on it remain as they are. This
  feature adds to the schema; it does not reshape what feature 001 established.
- **The architecture is already decided and is deliberately not restated here.** Issue #15 records
  it: the presentation layer runs registration and sign-in, and the service that owns collection
  data verifies every session independently. This specification states *what must be true*
  (FR-013, FR-014, SC-005), and `plan.md` is where the chosen libraries, token format, and schema
  live. A reader who needs the "how" should start at issue #15 and then the plan.
