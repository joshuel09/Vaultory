# Specification Quality Checklist: Collector Authentication

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-09-18
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

**On "no implementation details".** The architecture for this feature was decided before the spec
was written, and is recorded on issue #15. It is deliberately kept out of the requirements: no
library, token format, or table layout is named here. FR-013 and FR-014 state the property that
matters — the service owning collection data verifies every session itself and never trusts an
asserted identity — which is testable without knowing how it is implemented. The "how" belongs in
`plan.md`.

**Three requirements are security properties rather than features**, and are worth watching through
planning because they are the ones that quietly get lost: FR-013 (independent verification),
FR-020 (no session-minting mechanism in a production build), and FR-017 (no existing vault
orphaned). SC-005, SC-007 and SC-008 are their measurable forms, and SC-008 asks for the build to
be inspected rather than asserted about — the same standard feature 002 was held to.

**One assumption is load-bearing and temporary**: email addresses are not verified, which is why
password reset is out of scope. There is no trusted channel to reset through until addresses are
verified. This is stated in Assumptions rather than hidden, and is the first thing a follow-up
feature should address.

**Re-validated after clarification (2026-09-19).** Four questions were asked and integrated. Two
items were already passing but are materially stronger now: "Requirements are testable and
unambiguous" (FR-027 carried no number and could not have been accepted; FR-010 said "no more than
30 days" without saying whether use renewed it) and "Success criteria are measurable" (SC-003 and
SC-007 referred to conditions the spec never defined). No item regressed.

One clarification produced a requirement nobody asked for: FR-022a. Remembering where a signed-out
visitor was headed means taking a destination from the request and following it after sign-in,
which is an open redirect unless it is constrained to Vaultory's own paths. SC-013 is its
measurable form.
