# Quickstart: Edit & Delete Collectibles

**Feature**: `specs/006-edit-delete-collectibles` | **Date**: 2026-10-01

How to run this feature and prove it does what the spec says. Every scenario below maps to a
requirement; a scenario that passes for the wrong reason is called out where that is a real risk.

## Prerequisites

Docker, and nothing else (`README.md`). Then:

```bash
make up          # database, migrations through 000010, backend, frontend
```

A collector account is needed. Register at <http://localhost:3000/register> and collect the
verification mail from Mailpit at <http://localhost:8025>.

## Run the suites

```bash
make test        # backend unit, integration, contract
make test-e2e    # browser suite; needs `make up` first
```

Without Docker, what still runs:

```bash
cd backend  && go vet ./... && go build ./... && go test ./tests/unit/...
cd backend  && go vet -tags integration ./...
cd frontend && npm run lint && npm run typecheck && npm run test && npm run build
```

After changing the contract, regenerate the frontend's types and confirm nothing was hand-written:

```bash
cd frontend && npm run generate:api && git diff --stat lib/types/api.ts
```

---

## Scenario 1 — Correct a collectible (US1, FR-004, FR-028)

1. Add a collectible named `Kaiju Sentinel`, status Preordered, purchase price `1250.00`.
2. Add a second collectible afterwards, so the first is not at the front of the gallery.
3. Open the first for editing from its gallery entry. **Expect** every value already filled in, and
   fields you never supplied empty rather than defaulted (FR-003).
4. Change the name to `Kaiju Sentinel MkII` and the status to Owned. Save.
5. **Expect** a success confirmation, the new values in the gallery — and the collectible **in the
   same position as before**, not promoted to the front (FR-028).

Reload the page before judging step 5. A client-side update that looks right and a stored change are
different things.

## Scenario 2 — Clearing an optional value (FR-005)

1. Edit a collectible that has a purchase price, clear the field, save.
2. **Expect** the collectible to show no purchase price — not `0.00`.

Check the database rather than the screen if in doubt; the two are distinguishable only there:

```bash
docker compose exec db psql -U vaultory -d vaultory \
  -c "select name, purchase_price, version from collectibles order by created_at desc limit 5;"
```

`purchase_price` must be `NULL`, and `version` must have risen by one.

## Scenario 3 — A stale edit is refused (FR-027, the reason this feature has a migration)

1. Open the same collectible for editing in two browser tabs.
2. In tab A, change the name and save.
3. In tab B — loaded before that — change the notes and save.
4. **Expect** tab B to refuse the save, say the collectible changed since it was opened, and show
   the values as they now stand. Tab A's change must still be there.

This is the scenario that is easy to pass accidentally: if tab B silently reloaded at some point it
will hold the current version and succeed. Make the change in tab B **without** reloading it.

## Scenario 4 — Delete with confirmation (US2, FR-023, FR-025)

1. Add two identical collectibles.
2. Open one for editing and choose to delete. **Expect** a confirmation that names that collectible
   and says the deletion cannot be undone.
3. Press `Escape`. **Expect** nothing deleted.
4. Open it again, press `Enter` on the confirmation without moving focus. **Expect** nothing deleted
   — the destructive choice is not the default action (FR-023).
5. Click the delete button. **Expect** the collectible gone, the other identical one untouched, and
   it still gone after a reload.
6. With the browser's network tools, replay the `DELETE` request. **Expect** `204` again, and no
   error shown (FR-025).

## Scenario 5 — Photograph replaced, removed, and gone (US3, FR-020)

1. Add a collectible with a photograph. Note the rendition URL from the card's `<img src>`.
2. Open that URL directly in the same browser. **Expect** the image.
3. Edit the collectible, upload a different photograph, save.
4. Open the **old** URL again. **Expect** `404` — not a cached image. Hard-reload, or use a private
   window, because the response is `Cache-Control: private, max-age=3600`.
5. Edit again and remove the photograph. **Expect** the designed placeholder on the card, and the
   second rendition URL now answering `404` as well.

Then confirm the files are actually gone, not merely unreachable:

```bash
docker compose exec backend sh -c 'ls -R /var/lib/vaultory/images | head -40'
docker compose exec db psql -U vaultory -d vaultory -c "select count(*) from pending_image_deletions;"
```

The queue should be empty. A non-zero count means the drain is failing — the collector is unaffected
and the photograph is already unfetchable, which is the design (FR-020a), but it is a fault worth
investigating.

## Scenario 6 — Editing does not touch the photograph it was not asked about (FR-018)

1. Edit a collectible that has a photograph, change only its name, save.
2. **Expect** the photograph still there.

This is the contract's sharpest edge: `PUT` is a full replacement, so a client that omits `imageId`
removes the photograph. Covered by an integration test as well, because the failure would otherwise
reach a collector before it reached the suite.

## Scenario 7 — Another collector's collectible (FR-030, FR-031, SC-002)

With two accounts, A and B, and a collectible id belonging to A:

```bash
# As B. Expect 404 — never 403, and no clue that the id is real.
curl -i -b b.cookies http://localhost:3000/api/collectibles/<A-collectible-id>

# As B, an edit. Expect 404.
curl -i -X PUT -b b.cookies -H 'Content-Type: application/json' \
  -d '{"expectedVersion":1,"name":"Taken","collectionStatus":"owned"}' \
  http://localhost:3000/api/collectibles/<A-collectible-id>

# As B, a delete. Expect 204 — and A's collectible still there afterwards.
curl -i -X DELETE -b b.cookies http://localhost:3000/api/collectibles/<A-collectible-id>
```

The third one is the surprising answer and it is correct: a delete of something not in your vault
succeeds because the state you asked for already holds, and answering anything else would tell B
that the id is real. Verify as A that the collectible is untouched.

Run the same three with a random UUID and confirm the responses are byte-identical.

## Scenario 8 — Validation parity (SC-008)

For each rule, confirm the add form and the edit form refuse the same value with the same message:
an empty or whitespace-only name, a negative purchase price, a purchase date in the future, a status
outside the four, a name over 200 characters, notes over 2000.

```bash
cd backend && go test ./tests/unit/ -run Validation -v
```

The unit suite is where this is settled, because it is the only place both paths are exercised
against one rule set rather than against two forms.

## Scenario 9 — Keyboard and appearance (FR-037, FR-038)

1. Reach the edit screen from the gallery using only the keyboard.
2. Open the delete confirmation with the keyboard. **Expect** focus inside it, on Cancel.
3. `Tab` repeatedly. **Expect** focus never to leave the dialog.
4. `Escape`. **Expect** the dialog closed and focus back on the control that opened it.
5. Switch the system appearance between dark and light with the edit screen and the dialog open.
   **Expect** both to follow, with no unreadable text.

## What this feature must not have broken

- Adding still works, including the submission-key retry behaviour (feature 001 FR-047).
- The gallery still orders by when collectibles were added, and paging is unaffected.
- A rendition for an image that is still referenced is still served to its owner.
- `make test` passes with `-p 1`; the integration and contract packages share one database.
