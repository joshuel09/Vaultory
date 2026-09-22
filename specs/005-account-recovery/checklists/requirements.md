# Specification Quality Checklist: Account Recovery

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-09-22
**Feature**: [spec.md](../spec.md)

## Content Quality

- [X] No implementation details (languages, frameworks, APIs)
- [X] Focused on user value and business needs
- [X] Written for non-technical stakeholders
- [X] All mandatory sections completed

## Requirement Completeness

- [X] No [NEEDS CLARIFICATION] markers remain
- [X] Requirements are testable and unambiguous
- [X] Success criteria are measurable
- [X] Success criteria are technology-agnostic (no implementation details)
- [X] All acceptance scenarios are defined
- [X] Edge cases are identified
- [X] Scope is clearly bounded
- [X] Dependencies and assumptions identified

## Feature Readiness

- [X] All functional requirements have clear acceptance criteria
- [X] User scenarios cover primary flows
- [X] Feature meets measurable outcomes defined in Success Criteria
- [X] No implementation details leak into specification

## Notes

**Four requirements carry the weight**, and they are the ones most likely to be quietly lost in
planning because each is invisible when it works:

- **FR-015 / SC-009** — the stored form of a token must not be usable in place of the token. If
  tokens are stored as issued, anyone who can read the table can take over any account, and nothing
  about the running system would look wrong.
- **FR-018 / SC-005** — a reset must end other sessions. A reset that leaves them working does not
  do the thing people reset passwords for; it changes the key without changing the lock.
- **FR-008 / SC-003** — a reset request must not reveal whether an address has an account. The
  same reasoning that shaped sign-in in feature 004.
- **FR-025** — no recovery flow may mint a session by another route. This feature adds two new
  ways to become authenticated, which is exactly when feature 004's central guarantee is most
  likely to be undone by accident.

**One assumption is a deliberate product decision rather than a default**: an unverified collector
may still use their vault. Verification gates *recovery*, not *access*. Blocking a collection
behind an email round trip would punish the collector for a risk that is ours to manage, and the
address only becomes load-bearing when it is used to recover an account. Stated in Assumptions so
it can be argued with.

**Two lifetimes are conventional rather than derived**: 24 hours for verification, 1 hour for
reset. The asymmetry is the point — a reset link is worth more to an attacker than a verification
link, so it lives for less time.
