# Feature 003 — A landing page for Vaultory

**Issue**: #14 · **Status**: specified · **Depends on**: nothing · **Blocks**: #15 (authentication)

## Why this exists

Today `/` redirects unconditionally to `/collection`: a private vault the visitor has no session
for. They land on an error whose only button cannot help them. Nothing in the product says what
Vaultory is, who it is for, or why a collector would want one.

This ships **before** login and registration on purpose. A sign-up form is only worth building once
something persuades a person to fill it in, and this page is what the authentication work will link
out of.

## User scenarios

### US1 — A collector arrives knowing nothing (P1)

Someone follows a link with no idea what Vaultory is. They land on `/`, and within one screen they
understand that it keeps a personal collection of figures, statues, comics and trading
collectibles, presented as a gallery rather than a spreadsheet. They can see what it looks like
before committing to anything.

### US2 — A returning collector arrives (P2)

Someone who already has a vault does not want a marketing page between them and their collection.
The route to their vault is one obvious click, and never an obstacle.

### US3 — A collector browses in either appearance (P3)

The page is designed dark-first and is equally legible in light, because the constitution makes
dark a first-class experience rather than an inverted afterthought.

## Functional requirements

- **FR-001** `/` MUST render a public page, reachable with no session, and MUST NOT redirect to
  `/collection`.
- **FR-002** The page MUST state what Vaultory is in a collector's language, not as a feature list.
- **FR-003** It MUST show the four collection statuses — owned, preordered, wishlist, sold — as the
  vocabulary the product is built on.
- **FR-004** It MUST name what a vault records: character, series, manufacturer, category, scale,
  edition, purchase price, purchase date, release date, notes, and a photograph.
- **FR-005** It MUST present the gallery as the centrepiece, since presenting a collection visually
  rather than as an inventory table is the product's premise.
- **FR-006** It MUST offer one primary call to action leading into a vault. Until #15 lands this
  points at the development sign-in; afterwards it points at registration.
- **FR-007** It MUST be legible and usable from 360px to 1920px with no horizontal scrolling.
- **FR-008** It MUST work in both appearances, using only the existing design tokens. A hard-coded
  colour is a defect.
- **FR-009** It MUST reach the same accessibility standard as feature 001: meaningful text
  alternatives, a single `h1`, headings in order, and full keyboard operability.
- **FR-010** It MUST contain no business logic and make no authenticated request. It renders.

## Success criteria

- **SC-001** A visitor with no session can reach `/` and read it without an error state.
- **SC-002** Reaching a vault from `/` takes exactly one click.
- **SC-003** No horizontal scroll at 360, 390, 820, 1180, 1440 and 1920 px.
- **SC-004** Dark and light both render, and are genuinely different rather than one palette.
- **SC-005** The page makes no network request to `/api/*`.

## Out of scope

Registration, login, sessions, password recovery. Pricing, blog, testimonials, analytics,
cookie banners, newsletter capture. The call to action links out to authentication rather than
implementing any part of it.

## Assumptions

- Authentication does not exist yet, so the call to action uses the development sign-in. That is a
  temporary target, named as such in the code, and #15 replaces it.
- No new backend endpoint, no schema change, no contract change. This feature is frontend-only.
