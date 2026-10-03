# Research: Contributor Guide

Five decisions. None needed external research; each follows from the spec, the constitution, and how
the repository is already worked.

## 1. Where the guide lives

- **Decision**: `CONTRIBUTING.md` at the repository root.
- **Rationale**: GitHub recognises the root, `docs/`, and `.github/`, and in all three it links the
  file from the sidebar and from new issue and pull request forms (FR-001, US1 scenario 3). Only the
  root also puts it in the file listing a newcomer sees first.
- **Alternatives considered**: `.github/CONTRIBUTING.md` — hidden in a dot-directory, which works
  against "without being told where to look". `docs/CONTRIBUTING.md` — `docs/` currently holds only
  deferred-work notes.

## 2. How the README changes

- **Decision**: a one-line link to the guide directly under the opening description. The "Governance"
  section shrinks to a pointer to the constitution and the guide. "Working on it" becomes a pointer to
  the guide.
- **Rationale**: The README's rules list and the guide's would otherwise be two copies of the same
  six rules, and they would drift (SC-004). "Working on it" still calls feature 001 current; it must not
  survive as a second, wrong description of the workflow (FR-008).
- **Alternatives considered**: leaving "Governance" as is — the cheapest change, but it keeps the
  duplicate. Deleting it outright — loses the constitution pointer from the front page.

## 3. When the spec cycle may be shortened

- **Decision**: changes with no behaviour change — typos, documentation, dependency bumps with no API
  change, build or tooling fixes — may skip specify-through-analyze. They still need an issue, a
  branch, and a pull request. Anything a collector could notice, or anything touching authorization,
  money, schema, or the API contract, takes the full cycle.
- **Rationale**: The constitution requires the cycle for *significant* features. Naming what is not
  significant stops contributors either skipping it for real work or forcing a typo through six stages
  (FR-004).
- **Alternatives considered**: always the full cycle — ceremony without value. Contributor's
  judgement alone — "significant" is exactly what people disagree about, so the boundary is listed.

## 4. How much of each rule to state

- **Decision**: each non-negotiable rule is one bold sentence plus one sentence of reason, with a link
  to the constitution section it comes from.
- **Rationale**: a rule without its reason gets argued with or worked around; a rule restated at full
  length drifts from its source (FR-005, FR-006).
- **Alternatives considered**: linking to the constitution only — fails US2's independent test,
  which must be answerable from the guide alone.

## 5. Issue and pull request templates

- **Decision**: not added.
- **Rationale**: decision 1 already gets the guide offered on both forms. Templates would be a second
  place describing what an issue must contain.
- **Alternatives considered**: templates with a checklist — reasonable later if issues arrive
  missing context; no requirement asks for it now (constitution V, YAGNI).
