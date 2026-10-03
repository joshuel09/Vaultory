# Quickstart: verifying the contributor guide

Each walkthrough checks one part of the spec. They need a browser and a terminal; no running stack.

## A — A fresh clone contains the guide (FR-001, FR-010, SC-001)

```bash
git clone https://github.com/joshuel09/Vaultory.git /tmp/vaultory-check && cd /tmp/vaultory-check
ls CONTRIBUTING.md                     # present
git ls-files CLAUDE.md AGENTS.md       # prints nothing: still untracked
```

**Expected**: the guide is there; the agent files are not.

## B — Reachable from the front page and the forms (FR-008, SC-003)

1. Open the repository page on GitHub. The README links to the guide within its first screen.
2. Click **Issues → New issue**, and start a new pull request. Each form links to the contribution
   guidelines.

**Expected**: one click from the front page; offered on both forms.

## C — The workflow, from the guide alone (FR-002–FR-004, US1)

Using only `CONTRIBUTING.md`, answer: how does a new piece of work start, what is the branch called,
which stages does it go through, how does it land, and when may the stages be shortened?

**Expected**: issue on the project board first → `<issue-number>-<summary>` branch → specify through
implement → pull request with `Closes #N`; shortened only for changes with no behaviour change.

## D — The six rules, from the guide alone (FR-005, FR-006, SC-002)

Using only `CONTRIBUTING.md`, answer the six questions in the spec's User Story 2 test.

**Expected**: all six correct, each with its reason, and the guide names the constitution as the
authority with a working link.

## E — No drift, no tooling, no secrets (FR-007, FR-009, FR-011, SC-004, SC-005)

```bash
grep -n -i -E 'claude|anthropic|copilot|cursor|chatgpt|openai|agent' CONTRIBUTING.md README.md
grep -n -E '(PASSWORD|SECRET|TOKEN)=' CONTRIBUTING.md
grep -n -E 'go test|npm run|make ' CONTRIBUTING.md
grep -n 'Current feature' README.md
```

**Expected**: no output from any of them. Then read the guide's rules against
`.specify/memory/constitution.md`: no rule contradicts it.
