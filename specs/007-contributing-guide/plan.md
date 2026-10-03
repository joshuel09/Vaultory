# Implementation Plan: Contributor Guide

**Branch**: `28-contributing-guide` · **Spec**: [spec.md](spec.md) · **Issue**: #28
**Date**: 2026-10-03 · **Status**: planned

## Summary

A tracked `CONTRIBUTING.md` at the repository root carries the workflow and the non-negotiable rules,
each with its reason. The README links to it near the top, and its stale "Working on it" section and
duplicated rules list become pointers. **No application code, schema, or contract changes.** The agent
instruction files stay untracked.

## Technical Context

| | |
|---|---|
| **Deliverable** | One new Markdown file, `CONTRIBUTING.md`; edits to `README.md` |
| **Backend / Frontend / Database** | Unchanged |
| **Hosting** | GitHub, which surfaces a root `CONTRIBUTING.md` in the repository sidebar and on new issue and pull request forms |
| **Testing** | Review against the spec's acceptance scenarios, walked through in [quickstart.md](quickstart.md). No automated suite applies to prose |
| **Unknowns** | None. Five decisions in [research.md](research.md) |

## Constitution Check

| Principle | Assessment |
|---|---|
| **I. Collector-First** | Not affected: no user-facing product change |
| **II. Separation** | Not affected. The guide restates the separation (Go owns business logic; Next.js renders) so contributors keep it |
| **III. Contract-First** | Not affected. The guide states the OpenAPI contract is the source of truth for the seam |
| **IV. Data Integrity and Security** | The guide contains no secrets or environment values (FR-011). The agent files' ignore status is untouched (FR-010) |
| **V. Quality, Simplicity, Spec-Driven** | Began from a specification. The guide summarises and links rather than restating the constitution or copying README commands, which is the simplest form that cannot drift far |
| **Governance** | The guide is explicitly subordinate to the constitution and says so. The constitution itself is **not amended**: the workflow it already mandates is what the guide describes |

**Gate**: passes. No violations, so Complexity Tracking is empty.

**Re-check after design**: passes. The design adds one document and edits another; nothing else moves.

## Project Structure

```text
specs/007-contributing-guide/
├── spec.md              11 FRs, 5 SCs
├── plan.md              this file
├── research.md          5 decisions
├── quickstart.md        walkthroughs A–E, one per acceptance check
└── checklists/requirements.md
```

No `data-model.md` and no `contracts/`: the feature has no data and exposes no interface.

### Repository changes

```text
CONTRIBUTING.md          new — workflow, spec cycle, non-negotiable rules, quality gates
README.md                link to the guide near the top; "Governance" and "Working on it" become pointers
```

**Structure Decision**: root-level `CONTRIBUTING.md` (research decision 1). The README keeps running
and testing; the guide owns how to work; the constitution owns the rules.

## Complexity Tracking

None.
