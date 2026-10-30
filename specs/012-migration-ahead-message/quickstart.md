# Quickstart: verifying feature 012

Without Docker, the script runs on the host:

```bash
export MIGRATE_BIN=$(command -v migrate)          # golang-migrate 4.18.1
export MIGRATE_DATABASE_URL=postgres://...         # a scratch database
export MIGRATIONS_PATH=backend/migrations
sh backend/scripts/migrate.sh                      # ahead / dirty / empty / behind / current
sh backend/scripts/migrate.sh goto 10              # the move
```

With Docker, the real path:

1. On a branch with a newer migration (e.g. #58's `31-recoverable-trash`): `make up`.
2. `git switch main && make up`: expect the explanation in the output and exit status non-zero.
3. Follow it: `git switch 31-recoverable-trash && make migrate-to VERSION=10 && git switch main && make up`. The stack starts.
