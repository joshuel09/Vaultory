# Quickstart: verifying the sweep

Needs PostgreSQL with migrations applied (`make test` does this; or set `VAULTORY_TEST_DATABASE_URL`).

## A — Eligible uploads are swept, files and all (US1, FR-001, FR-003)

`go test -tags=integration -run TestSweep ./tests/integration/...` in `backend/`. Expect an upload
backdated past 30 days to lose its row, gain a queue entry, and lose both files after the drain.

## B — Young and referenced uploads survive (US2, FR-004, FR-005)

Same run. An upload one hour short of the grace period, and an old upload a collectible references,
are both untouched.

## C — The race (US2 scenario 3, FR-006, SC-002)

Same run. A save and a sweep of the same eligible upload are started together, many times. Every
iteration ends with a collectible that has its photograph and an upload that survives, or with a
refused save and no upload. Never a collectible whose `image_id` was set to null.

## D — Retry after a storage fault (US1 scenario 3, SC-004)

Same run, with an image store that fails deletes: the queue entry stays; once it stops failing, the
next drain removes the files.

## E — The refused save, in the browser (US3, FR-007)

`npm run test` in `frontend/`: an `imageId` violation renders on the photograph field with the
"upload it again" message and leaves the form's values in place.
