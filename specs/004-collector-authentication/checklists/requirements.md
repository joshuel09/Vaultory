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
