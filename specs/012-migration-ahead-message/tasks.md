# Tasks: Say What Is Wrong When the Database Is Ahead

- [ ] T001 `backend/scripts/migrate.sh`: read the version, find the checkout's newest; up when empty, behind or current; stop with exit 3 and the explanation when ahead; exit 4 with its own explanation when dirty; `goto N` passes through; any other argument refused (FR-001, FR-002, FR-005–FR-007)
- [ ] T002 `compose.yaml`: the `migrate` service runs the script, with the script mounted read-only and the database URL in the environment (FR-001)
- [ ] T003 `Makefile`: `up` prints the `migrate` step's output when that step exited non-zero; `migrate-to` requires `VERSION` and runs the script's `goto` (FR-003, FR-004)
- [ ] T004 README: what this message means and what to do, under running locally
- [ ] T005 Verify the script on the host against PostgreSQL 17 with migrate 4.18.1 in every state — ahead (nothing changed), dirty, empty, behind, current, `goto` both ways, no version — and record the output (SC-001, SC-002)
- [ ] T006 Note in the PR what could not run here: `make`, Compose, and the image's busybox `sh`
