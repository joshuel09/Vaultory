# Contributing to Vaultory

How to work on Vaultory: how a change moves from idea to `main`, the specification cycle it goes
through, and the rules that will fail review if broken.

The [constitution](.specify/memory/constitution.md) governs this project. This guide summarises the
parts that matter day to day; where the two disagree, the constitution wins and this guide is wrong.
How to run and test the stack lives in the [README](README.md).

## How work moves

**Nothing is committed directly to `main`.** Every change goes through an issue, a branch, and a pull
request, in that order.

1. **Open an issue first**, on the [Vaultory project board](https://github.com/users/joshuel09/projects/1),
   before writing any code. The body says what the task is and why it exists, not just the title
   again.
2. **Branch from `main`**, named after the issue: `<issue-number>-<short-summary>`, for example
   `26-edit-delete-collectibles`.
3. **Reference the issue in every commit** with `Refs #<issue-number>` in the message body.
4. **Open a pull request** whose description says what changed and why, and ends with
   `Closes #<issue-number>`, so merging closes the issue and moves its card to Done.
5. **Merge once it is green and reviewed.** If the issue does not close, close it by hand rather than
   leaving it open.

If a task outgrows one reviewable branch, split it into several issues and branches rather than
growing one.

## The specification cycle

Significant work starts from a written specification, not from code. The project uses
[Spec Kit](https://github.com/github/spec-kit), and the constitution's
[development workflow](.specify/memory/constitution.md#development-workflow) has eight stages:

1. **Specify**: what the feature must do and why, with testable requirements. No implementation detail.
2. **Clarify**: resolve anything ambiguous before it is designed around.
3. **Plan**: the technical approach, checked against the constitution.
4. **Tasks**: small, ordered tasks, each traced to a requirement.
5. **Analyze**: check spec, plan and tasks against each other and the constitution. Fix anything
   critical or high before going on.
6. **Implement**: work through the tasks, within the approved scope.
7. **Test**: the quality gates below must pass.
8. **Review**: the pull request.

Each feature's artifacts live in `specs/NNN-<short-name>/`. That number is Spec Kit's own sequence and
is independent of the issue number.

### When the cycle may be shortened

A change with **no behaviour change** may go straight from issue to implementation: a typo, a
documentation edit, a dependency bump with no API change, a build or tooling fix. It still needs an
issue, a branch, and a pull request.

Anything a collector could notice takes the full cycle. So does anything that touches authorization,
money, the database schema, or the API contract, however small it looks.

## Rules that are not negotiable

These are the rules that most often catch people out. Each is invisible when followed and a defect
when not.

### From the constitution

- **Go owns business logic, validation, authorization, and persistence. Next.js renders.**
  A rule that exists only in the frontend can be bypassed by anyone who calls the API directly.
  ([II](.specify/memory/constitution.md#ii-clear-architecture-and-separation-of-responsibilities))
- **Money is an exact decimal, never a floating-point number.** Floating point cannot represent most
  prices exactly, and the error compounds across a collection.
  ([IV](.specify/memory/constitution.md#iv-data-integrity-and-security))
- **The OpenAPI contract describes every frontend/backend request and response.** It is the one place
  the two applications agree on, so a change that is not in it is a change the other side does not
  know about. ([III](.specify/memory/constitution.md#iii-contract-first-and-type-safe-development))
- **Schema changes ship as version-controlled migrations.** The database must be reproducible from the
  repository alone. ([IV](.specify/memory/constitution.md#iv-data-integrity-and-security))
- **Secrets and credentials are never committed.** The repository is public; anything committed is
  published, and stays in history after it is deleted.
  ([IV](.specify/memory/constitution.md#iv-data-integrity-and-security))

### Project conventions

These are not in the constitution. They were decided in earlier features and are applied to every
change since.

- **Another collector's resource returns 404, never 403.** A 403 confirms the resource exists, which
  leaks information about someone else's private vault.
  ([feature 004](specs/004-collector-authentication/spec.md))
- **Every migration has an up and a down file** in `backend/migrations/`. A migration that cannot be
  reversed turns a bad deploy into a restore from backup.
  ([feature 001](specs/001-add-browse-collectibles/research.md))
- **Frontend types are generated from the contract, never written by hand.** The contract is
  `specs/001-add-browse-collectibles/contracts/openapi.yaml`. A hand-written response
  shape is a second copy of the contract, and it drifts.
  ([feature 001](specs/001-add-browse-collectibles/plan.md))

## Before you open a pull request

The constitution's [quality gates](.specify/memory/constitution.md#quality-gates) must all pass before
work counts as complete:

- Type checking and linting pass.
- Backend tests and frontend tests pass.
- Production builds succeed.
- API contract changes are reflected in the OpenAPI document.
- Schema changes include their migrations.
- The specification's acceptance criteria have been verified.
- The change has been reviewed against the constitution.

The commands for each are in the README's [Testing](README.md#testing) section.
