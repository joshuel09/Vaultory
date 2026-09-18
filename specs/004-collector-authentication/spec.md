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
2. **Given** a visitor with no session, **When** they open a vault page, **Then** they see a
   "sign in to see your vault" state, not "your collection could not be loaded".
3. **Given** a signed-out collector, **When** they press the browser's back button to a vault page,
   **Then** no collection content is served from that page.

---

### Edge Cases

- A request arrives carrying a session that is valid in form but was issued for a collector who no
  longer exists.
- A request asserts an identity directly — a header, a body field, or a query parameter naming a
  collector — while carrying no valid session, or while carrying a session for someone else.
- A session is tampered with: its payload edited, its signature replaced, or one collector's
  session presented with another's identifier.
- Registration and sign-in are attempted repeatedly and rapidly with different passwords.
- Two registrations for the same email arrive simultaneously.
- A collector signs in on one device and signs out on another.
- The existing seeded development collectors from feature 001 already own collectibles when
  authentication arrives.

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
- **FR-010**: A session MUST expire no more than 30 days after it was issued.
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
- **FR-017**: Collectibles, images, and submissions belonging to collectors that already exist MUST
  remain attached to those collectors once authentication is introduced, with nothing orphaned.

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
- **FR-022**: A request for a vault page without a valid session MUST result in an invitation to
  sign in, distinct from the state shown when the collection genuinely cannot be reached.
- **FR-023**: After signing in or registering, a collector MUST arrive at their collection.
- **FR-024**: The landing page's primary call to action MUST lead to registration.
- **FR-025**: A signed-in collector MUST be able to see which account they are signed in as, and
  reach sign-out from any vault page.
- **FR-026**: Registration and sign-in MUST report failures the way the rest of Vaultory does:
  every problem at once, against the field responsible, announced to assistive technology, and
  never discarding what was typed.

**Not being negligent**

- **FR-027**: The system MUST limit how rapidly sign-in attempts can be made against a single
  account, and MUST say when it has done so.
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
- **SC-003**: A collector who closes the browser and returns within the session lifetime is not
  asked to sign in again.
- **SC-004**: No request carrying an absent, expired, tampered, or another collector's session
  returns any collection content, in 100% of attempts.
- **SC-005**: An identity asserted by any means other than a verified session is ignored, in 100%
  of attempts.
- **SC-006**: A wrong password and an unregistered email produce responses indistinguishable to the
  person attempting them.
- **SC-007**: Every collectible that existed before this feature is still owned by the same
  collector afterwards, and none is orphaned.
- **SC-008**: A shipped production build contains no mechanism that issues a session without
  verifying credentials, demonstrated by inspecting the build rather than by assertion.
- **SC-009**: A production build starts and serves traffic when authentication is configured.
- **SC-010**: A visitor reaching a vault page without a session is invited to sign in, and is never
  shown a message attributing the refusal to a problem reaching the service.
- **SC-011**: Registration and sign-in are completable using a keyboard alone, and every failure is
  announced to assistive technology.

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
- The two seeded development collectors from feature 001 are fixtures, not real users. They may be
  left without accounts, or removed, provided FR-017 holds for any collector that owns data.
- Sessions are stored server-side so that sign-out can end one immediately, rather than waiting for
  a self-contained token to expire.
- A 30-day session lifetime is a reasonable default for a personal collection tool; it is not
  derived from a stated requirement.
- The existing `collectors` table and every foreign key built on it remain as they are. This
  feature adds to the schema; it does not reshape what feature 001 established.
- **The architecture is already decided and is deliberately not restated here.** Issue #15 records
  it: the presentation layer runs registration and sign-in, and the service that owns collection
  data verifies every session independently. This specification states *what must be true*
  (FR-013, FR-014, SC-005), and `plan.md` is where the chosen libraries, token format, and schema
  live. A reader who needs the "how" should start at issue #15 and then the plan.
