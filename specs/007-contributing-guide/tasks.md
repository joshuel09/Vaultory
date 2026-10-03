# Tasks: Contributor Guide

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [quickstart.md](quickstart.md)

**Tests**: none automated — the deliverable is prose. Each story ends with its quickstart walkthrough
as the check.

**Format**: `[ID] [P?] [Story] Description`. `[P]` = different file, no dependency on an unfinished task.

## Phase 1: Setup

- [ ] T001 Create `CONTRIBUTING.md` at the repository root with a title, a one-paragraph purpose
  ("how to work on Vaultory; the constitution governs, this guide summarises it"), and empty section
  headings: How work moves, The specification cycle, Rules that are not negotiable, Before you open a
  pull request (research decision 1)

## Phase 2: Foundational

None. The README edits depend on the guide existing (T001), not on any shared groundwork.

## Phase 3: User Story 1 — A new contributor finds how to work on Vaultory (P1) 🎯 MVP

**Goal**: from a fresh clone, a newcomer finds the workflow in one step.

**Independent test**: quickstart walkthroughs A, B and C.

- [ ] T002 [US1] Write the "How work moves" section of `CONTRIBUTING.md`: issue first, on the
  Vaultory project board, with a body saying what and why; branch `<issue-number>-<short-summary>` from
  `main`; commits reference `Refs #N`; pull request whose body ends `Closes #N`; never commit to `main`;
  split a task that outgrows one reviewable branch (FR-002)
- [ ] T003 [US1] Write "The specification cycle" section of `CONTRIBUTING.md`: the constitution's
  eight stages in order — specify, clarify, plan, tasks, analyze, implement, test, review — with one
  line each on what the stage produces, test pointing to the quality gates (T009) and review to the
  pull request; that the spec directory under `specs/` is
  numbered independently of the issue; and when the cycle may be shortened — changes with no behaviour
  change only, with the listed exclusions (authorization, money, schema, API contract, anything a
  collector could notice), still with issue, branch and pull request (FR-003, FR-004, research
  decision 3)
- [ ] T004 [P] [US1] In `README.md`, add one line directly under the opening description linking to
  `CONTRIBUTING.md` (FR-008, research decision 2)
- [ ] T005 [US1] In `README.md`, replace the "Working on it" section with a two-sentence pointer
  to `CONTRIBUTING.md`, removing the stale "Current feature: 001" line (FR-008)

**Checkpoint**: run quickstart A and C; B after push.

## Phase 4: User Story 2 — A contributor knows the rules that cannot be broken (P1)

**Goal**: the six rules, each with its reason, answerable from the guide alone.

**Independent test**: quickstart walkthrough D.

- [ ] T006 [US2] Write "Rules that are not negotiable" in `CONTRIBUTING.md`: one bold sentence and
  one sentence of reason each, in two groups attributed to their real source (FR-006).
  **From the constitution**, linked to it: Go owns business logic, validation, authorization and
  persistence (II, Backend); money is exact decimals, never floating point (IV); OpenAPI describes the
  frontend/backend contract (III, API); schema changes ship as version-controlled migrations (IV,
  Quality Gates); secrets and credentials are never committed (IV).
  **Project conventions**, linked to where they were decided: another collector's resource returns
  404, never 403, because a 403 confirms it exists (`specs/004-collector-authentication/spec.md`);
  every migration has an up and a down file in `backend/migrations/` (`specs/001-add-browse-collectibles/research.md`);
  frontend types are generated from `openapi.yaml`, never hand-written (`specs/001-add-browse-collectibles/plan.md`)
  (FR-005, FR-006, research decision 4)
- [ ] T007 [US2] Open the same section of `CONTRIBUTING.md` with the precedence statement: the
  constitution at `.specify/memory/constitution.md` governs and wins any conflict with this guide
  (FR-006)
- [ ] T008 [US2] In `README.md`, shrink the "Governance" section to a pointer: the constitution
  governs, and `CONTRIBUTING.md` summarises the rules that most often catch people out — removing the
  duplicated rules list (SC-004, research decision 2)

**Checkpoint**: run quickstart D.

## Phase 5: User Story 3 — A contributor knows what "done" means (P2)

**Goal**: the completion gates, with a pointer to the commands.

**Independent test**: a reader lists the gates and finds the commands via the README link.

- [ ] T009 [US3] Write "Before you open a pull request" in `CONTRIBUTING.md`: the constitution's
  quality gates (type checks, lint, backend tests, frontend tests, production build, OpenAPI updated
  for contract changes, migrations for schema changes, acceptance criteria verified) and a link to the
  README's Testing section for the commands — no commands copied (FR-007)

**Checkpoint**: confirm the README link resolves.

## Phase 6: Polish & Cross-Cutting

- [ ] T010 Run quickstart walkthrough E against `CONTRIBUTING.md` and `README.md`: no tool names, no
  secret-shaped values, no copied commands, no "Current feature" line (FR-009, FR-011, SC-004, SC-005)
- [ ] T011 Read every rule in `CONTRIBUTING.md` against `.specify/memory/constitution.md`; fix any
  contradiction in the guide, never in the constitution (SC-004)
- [ ] T012 Confirm `git ls-files CLAUDE.md AGENTS.md` prints nothing and `.gitignore` is unchanged
  (FR-010)
- [ ] T013 After pushing, run quickstart walkthrough B on GitHub: the README links the guide in its
  first screen, and the new issue and pull request forms both offer it (SC-003)

## Dependencies

- T001 blocks T002, T003, T006, T007, T009 (all write into `CONTRIBUTING.md`, in that order — same file,
  so not parallel with each other).
- T004, T005, T008 edit `README.md` and depend only on T001 existing. T004 is [P] with the guide
  work (different file); T005 and T008 follow it in order, since they share `README.md`.
- US1 → US2 → US3 in file order; each is independently checkable once written.
- Phase 6 after all stories.

## Parallel example

```text
Writer A: T002 → T003 → T006 → T007 → T009   (CONTRIBUTING.md)
Writer B: T004 → T005 → T008                  (README.md)
```

## Implementation strategy

MVP is US1 (T001–T005): the workflow is findable and the stale section is gone. US2 then adds the
rules, US3 the gates. One pull request for all of it — the change is small enough to review whole.
