# Feature Specification: Contributor Guide

**Feature Branch**: `28-contributing-guide`

**Created**: 2026-10-03

**Status**: Draft

**Input**: Issue #28 — "Make the working rules available to everyone who clones the repo." The
working rules live only in agent instruction files that were deliberately untracked in #20, so a
fresh clone contains none of them. Decision: a tracked, tool-neutral contributor guide carries the
human-facing workflow and rules; the agent instruction files stay untracked; the README points to
the guide.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A new contributor finds how to work on Vaultory (Priority: P1)

Someone clones the repository for the first time, intending to contribute. Without being told where
to look, they find a single guide that tells them how a piece of work moves from idea to merged
change: it starts as an issue on the project board, is done on its own branch named after that
issue, follows the specification cycle, and lands through a pull request that closes the issue.

**Why this priority**: This is the gap the issue names. Today the workflow can only be inferred from
commit history; a contributor who does not follow it produces work that cannot be tracked or
reviewed the way the rest of the project is.

**Independent Test**: Give a fresh clone to someone who has not seen the project. Ask them how to
start a new piece of work. They answer correctly, from the repository alone, in under five minutes.

**Acceptance Scenarios**:

1. **Given** a fresh clone, **When** a contributor opens the repository's front page, **Then** it
   links directly to the contributor guide.
2. **Given** the contributor guide, **When** a contributor reads the workflow section, **Then** they
   can state, in order: create the issue on the project board, create a branch named
   `<issue-number>-<summary>`, work through the specification cycle, open a pull request that closes
   the issue.
3. **Given** the hosting platform's standard locations, **When** a contributor opens a new issue or
   pull request, **Then** the platform offers them a link to the guide without them searching for it.

---

### User Story 2 - A contributor knows the rules that cannot be broken (Priority: P1)

Before writing code, a contributor learns the rules that will fail review if broken, stated plainly
enough to act on, and knows that the constitution is the authority behind them.

**Why this priority**: These are the rules that "actually bite". Each is invisible when followed and
costly when not — a float in a money calculation or a 403 that confirms another collector's item
exists is a defect, not a style choice.

**Independent Test**: Ask a reader of the guide alone to answer: who owns business logic; how money
is represented; what a request for another collector's item returns, and why; where the API shape is
defined; how a schema change ships; whether secrets may be committed. All six answers are correct.

**Acceptance Scenarios**:

1. **Given** the guide, **When** a contributor looks for the non-negotiable rules, **Then** each rule
   is stated with the reason it exists, in one place.
2. **Given** a rule in the guide, **When** it conflicts with something the contributor believed,
   **Then** the guide names the constitution as the governing document and links to it.

---

### User Story 3 - A contributor knows what "done" means (Priority: P2)

A contributor about to open a pull request knows which checks must pass before work counts as
complete, and how to run them.

**Why this priority**: Valuable but partly covered already — the README documents the commands. The
guide's job is to state the gate and point to them, not to duplicate them.

**Independent Test**: A reader of the guide can list the completion gates (type checks, lint, backend
and frontend tests, production build, contract and migration updates) and find the commands for
each.

**Acceptance Scenarios**:

1. **Given** the guide, **When** a contributor prepares a pull request, **Then** they find the list of
   quality gates and a link to where the commands are documented.

---

### Edge Cases

- **Guide and constitution disagree**: the constitution wins. The guide says so, and summarises rather
  than restates, so there is less to drift.
- **Guide and README disagree about how to run something**: the README owns running and testing; the
  guide links to it rather than copying commands.
- **A contributor uses an assistant or agent tooling**: the guide is tool-neutral and applies
  regardless. Any agent-specific instruction files remain local and untracked, and the guide must
  work without them.
- **A small change (typo, one-line fix)**: the guide states when the full specification cycle may be
  shortened, so contributors are not forced through ceremony for trivial work — but still need an
  issue and a branch.
- **Stale references**: the README's existing "Working on it" section names feature 001 as current,
  which is out of date. It must not survive as a second, contradicting description of the workflow.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The repository MUST contain a tracked contributor guide at the location the hosting
  platform recognises as the project's contribution guidelines.
- **FR-002**: The guide MUST describe the work lifecycle in order: issue on the project board first,
  branch named `<issue-number>-<short-summary>`, commits referencing the issue, pull request that
  closes the issue, merge only through a pull request — never directly to `main`.
- **FR-003**: The guide MUST describe the specification cycle (specify → clarify → plan → tasks →
  analyze → implement) and state that significant work begins from a written specification.
- **FR-004**: The guide MUST state when the specification cycle may be shortened for trivial changes,
  and that an issue and branch are still required.
- **FR-005**: The guide MUST state each non-negotiable rule with its reason: the backend owns business
  logic, validation, authorization and persistence; monetary values are exact decimals, never
  floating point; another collector's resource answers not-found, never forbidden; the API contract
  document is the source of truth for the frontend/backend seam; schema changes ship as reversible
  migrations; secrets and credentials are never committed.
- **FR-006**: The guide MUST name the constitution as the governing document, link to it, and state
  that it takes precedence over the guide.
- **FR-007**: The guide MUST list the quality gates required before work is complete and link to the
  README for the commands, rather than duplicating them.
- **FR-008**: The README MUST link to the guide from a prominent position, and its existing "Working
  on it" section MUST be replaced by a short pointer to the guide, removing the stale "current
  feature" reference.
- **FR-009**: The guide MUST be tool-neutral: it MUST NOT name, depend on, or require any specific
  assistant or agent tooling.
- **FR-010**: The agent instruction files MUST remain untracked; this feature MUST NOT change their
  ignore status.
- **FR-011**: The guide MUST NOT contain secrets, credentials, or environment-specific values.

### Key Entities

- **Contributor guide**: the tracked document a person reads to learn how to work on the project.
- **Constitution**: the existing governing document; the guide summarises and defers to it.
- **README**: the existing front page; owns running and testing, and points to the guide.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From a fresh clone, a newcomer locates the workflow and the non-negotiable rules in
  under 2 minutes without being told where to look.
- **SC-002**: A reader of the guide alone answers all 6 rule questions in User Story 2's test
  correctly.
- **SC-003**: The guide is reachable in one step from the repository front page, and is offered by the
  hosting platform when opening a new issue or pull request.
- **SC-004**: Zero rules in the guide contradict the constitution, and zero commands are duplicated
  from the README.
- **SC-005**: Zero references to any specific assistant or agent tooling appear in the guide or the
  changed README.

## Assumptions

- Option 2 from the issue was chosen: keeping the agent files untracked preserves the decision made in
  #20, and the repository owner may switch to option 1 or 3 in review.
- The human-facing rules are not sensitive; the repository is public and they are good practice.
- Spec directory is numbered 007 because 006 is in use on the open branch for #26.
- No application code, schema, or API changes are involved; the backend and frontend quality gates
  are unaffected, and verification is by review against the acceptance scenarios.
- Contributors have read access to the project board; the guide names it but does not grant access.
