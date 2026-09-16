# Working on Vaultory

## Task workflow — follow this for every piece of work

**Never commit directly to `main`.** Every change goes through an issue, a branch, and a pull
request. This is how the work stays monitorable.

The order matters:

1. **Create the GitHub Issue first**, before writing any code.
   ```bash
   gh issue create --title "<short imperative title>" --body "<what and why>" --project Vaultory
   ```
   The body must say what the task is and why it exists — not just restate the title. Add it to the
   project board at creation time, so nothing is tracked in only one place.

2. **Create a branch for that issue**, named so the link is obvious.
   ```bash
   git switch -c <issue-number>-<short-kebab-summary>    # e.g. 12-docker-compose-profiles
   ```

3. **Connect the branch to the issue.** Reference the issue number in commit messages
   (`Refs #12`) and close it from the PR body (`Closes #12`), so merging moves the task
   automatically.

4. **Do the work on that branch.** Commit as usual — small, described commits are fine, and they
   stay off `main`.

5. **Open a PR when done**, describing what changed and why.
   ```bash
   git push -u origin <branch>
   gh pr create --title "<title>" --body "...\n\nCloses #<issue-number>" --fill-first
   ```

6. **Merge the PR** once it is green. Merging closes the issue, which moves the card to Done on the
   board.
   ```bash
   gh pr merge --squash --delete-branch
   ```

7. **Confirm the issue closed** and the board reflects it. If the automation did not fire, close it
   explicitly rather than leaving it open.

If a task turns out to be larger than one issue, split it into several issues and branches rather
than growing one branch until it is unreviewable.

## What this project is

Vaultory is a collectibles collection management platform. Collectors keep a private digital vault
of figures, statues, comics, and trading collectibles, presented as a visual gallery rather than an
inventory table.

```text
backend/    Go — domain, validation, authorization, persistence, image handling
frontend/   Next.js App Router — presentation only
specs/      Spec Kit artifacts per feature
.specify/   Spec Kit configuration and the constitution
```

## Rules that are not negotiable

`.specify/memory/constitution.md` governs this project and supersedes informal convention. The ones
that most often catch people out:

- **Go owns business logic**, validation, authorization, and persistence. Next.js renders and
  nothing more. A business rule that exists only in the frontend is a bug.
- **Monetary values are exact decimals, never floating point.** `Money` in the domain has no float
  in its API for this reason.
- **A collector's vault is private, enforced server-side.** Every query carries `collector_id` as a
  predicate. Another collector's resource returns **404, never 403** — a 403 confirms it exists.
- **The OpenAPI contract is the source of truth** for the frontend/backend seam. Frontend types are
  generated from it; never hand-write a response shape.
- **Schema changes ship as reversible migrations.** No exceptions.
- **The development identity resolver must never reach a production build.** It mints a session for
  anyone who asks. `internal/identity/dev.go` is `//go:build !production` and `backend/Dockerfile`
  builds with `-tags production`, so the shipped binary does not contain it — a misconfigured
  environment variable cannot turn authentication off. If you touch that package, run
  `go test -tags production ./tests/unit/...`.
- Work begins from a written specification, and tasks trace back to its requirements.

## Spec Kit

Features follow: `/speckit-specify` → `/speckit-clarify` → `/speckit-plan` → `/speckit-tasks` →
`/speckit-analyze` → `/speckit-implement`.

`/speckit-plan` and the commands after it operate on the feature named in `.specify/feature.json`.
**New scope needs a new feature directory** — running `/speckit-plan` for unrelated work would plan
it inside whichever feature is current.

Current features:

- `specs/001-add-browse-collectibles/` — complete, 95/97 tasks. The two open tasks need a database.
- `specs/002-docker-dev-environment/` — built, 26/32 tasks. The six open tasks need Docker installed,
  which it is not here. Two of them (image size, multi-arch) are partially verified without it.

## Running and testing

With Docker — the normal path, and the only prerequisite:

```bash
make up          # database, migrations, backend, frontend
make watch       # the same, syncing your edits into the running stack
make test        # backend unit, integration, and contract suites
make test-e2e    # the browser suite; needs `make up` first
make down        # stop, keeping data.   make reset destroys it.
```

`make test` uses a throwaway database with no route to the development one. That isolation is
structural, not a convention — the integration harness truncates tables between tests.

Without Docker, and what runs with no database at all:

```bash
cd backend  && go vet ./... && go build ./... && go test ./tests/unit/...   # 34 tests
cd frontend && npm run lint && npm run typecheck && npm run test && npm run build   # 30 tests
```

Full instructions in `README.md`.
