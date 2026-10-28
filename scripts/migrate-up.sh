#!/bin/sh
# Apply the outstanding migrations, and explain the one failure that reads like a broken stack.
#
# `golang-migrate` tracks a single version number. A branch carrying a migration moves the
# development database forward; coming back to a branch without that migration leaves the database
# on a version this checkout has no file for, and `up` fails with
#
#     error: no migration found for version 12: read down for version 12 .: file does not exist
#
# which is true, is only in `docker compose logs migrate`, and tells nobody what to do. What
# `make up` shows is `service "migrate" didn't complete successfully: exit 1`.
#
# Nothing is broken and nothing is lost when this happens: the database simply holds a schema
# belonging to a branch that is not checked out. This says that, in those words (issue #62).
set -eu

migrations=${MIGRATIONS_PATH:-/migrations}

# The newest migration this checkout has.
#
# `10#` forces base ten. A migration is named with leading zeros, and a shell arithmetic context
# reads a leading zero as octal — so without this, 000008 and 000009 are not just wrong, they are
# an error, and only on the two files that have them.
newest=0
for file in "$migrations"/*.up.sql; do
	n=$(basename "$file" | cut -d_ -f1)
	n=$((10#$n))
	[ "$n" -gt "$newest" ] && newest=$n
done

# `migrate version` writes the version to stderr and exits 0. On a database that has never been
# migrated it writes an error there instead, which is the first-run case and means zero. A dirty
# database reports `12 (dirty)`, so take the first field and let `up` report the dirtiness itself.
current=$(migrate -path="$migrations" -database="$DATABASE_URL" version 2>&1 || true)
current=$(printf '%s\n' "$current" | head -1 | cut -d' ' -f1)
case $current in
'' | *[!0-9]*) current=0 ;;
esac

if [ "$current" -gt "$newest" ]; then
	cat >&2 <<MESSAGE

  ────────────────────────────────────────────────────────────────────────────────
  The development database is ahead of this checkout.

      the database is at migration   $current
      the newest migration here is   $newest

  A branch carrying later migrations was run against this database, and those
  migrations are not in the branch checked out now. Your collection is intact: the
  schema just belongs to a branch you are not on.

  Three ways out, and the choice is yours to make:

      go forward again     git switch <that branch> && make up

      come back to here    git switch <that branch>
                           make db-rewind VERSION=$newest
                           git switch -   &&  make up

      start clean          make reset && make up        (destroys your data)

  The rewind has to happen on the other branch because the down migrations for
  everything past $newest live there, not here — which is the same reason this
  failed in the first place.
  ────────────────────────────────────────────────────────────────────────────────

MESSAGE
	exit 1
fi

exec migrate -path="$migrations" -database="$DATABASE_URL" up
