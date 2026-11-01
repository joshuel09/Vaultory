#!/bin/sh
# Run the database migrations — and say plainly what is wrong when they cannot run (feature 012).
#
# This is the `migrate` one-shot's entrypoint in compose.yaml. Without arguments it applies every
# pending migration, exactly as `migrate up` did before. First, though, it checks the case that made
# `make up` fail with nothing but "exit 1": a database already *ahead* of this checkout, because
# another branch's migrations were applied to it while reviewing that branch (issue #62).
#
#   migrate.sh          check, then apply what is pending
#   migrate.sh goto N   move the database to version N with this checkout's migrations
#   migrate.sh force N  record version N without running anything — only to clear a dirty flag
#
# It never reverses anything on its own. Which way to go is the developer's choice (FR-005).
#
# Environment:
#   MIGRATE_DATABASE_URL  the database. Carries the password, so it is never printed.
#   MIGRATIONS_PATH       the migrations directory (default /migrations)
#   MIGRATE_BIN           the golang-migrate binary (default: migrate)
#
# POSIX sh, because the migrate image is Alpine with busybox and nothing else.

set -u

MIGRATIONS_PATH=${MIGRATIONS_PATH:-/migrations}
MIGRATE_BIN=${MIGRATE_BIN:-migrate}

if [ -z "${MIGRATE_DATABASE_URL:-}" ]; then
	echo "migrate.sh: MIGRATE_DATABASE_URL is not set." >&2
	exit 2
fi

run_migrate() {
	"$MIGRATE_BIN" -path "$MIGRATIONS_PATH" -database "$MIGRATE_DATABASE_URL" "$@"
}

case "${1:-}" in
"")
	;;
goto | force)
	case "${2:-}" in
	'' | *[!0-9]*)
		echo "migrate.sh: $1 needs a version number, for example: $1 10" >&2
		exit 2
		;;
	esac
	run_migrate "$1" "$2"
	exit $?
	;;
*)
	echo "migrate.sh: unknown command '$1'. Use no arguments, goto <version>, or force <version>" >&2
	exit 2
	;;
esac

# The newest migration this checkout knows, from the file names (NNNNNN_name.up.sql). Leading
# zeros are stripped so the comparison below is numeric, not textual.
newest=0
for file in "$MIGRATIONS_PATH"/*.up.sql; do
	[ -e "$file" ] || continue
	name=${file##*/}
	number=$(echo "${name%%_*}" | sed 's/^0*//')
	number=${number:-0}
	if [ "$number" -gt "$newest" ]; then
		newest=$number
	fi
done

# What the database has applied. golang-migrate prints this on stderr: "12", or "12 (dirty)", or
# "error: no migration" (with exit 1) for a database nothing has been applied to yet.
output=$(run_migrate version 2>&1)
status=$?

if [ $status -ne 0 ]; then
	case "$output" in
	*"no migration"*)
		# A new database. Nothing to compare; apply everything.
		exec "$MIGRATE_BIN" -path "$MIGRATIONS_PATH" -database "$MIGRATE_DATABASE_URL" up
		;;
	*)
		# Could not read the version for some other reason — let migrate say why, as before.
		echo "$output" >&2
		exit $status
		;;
	esac
fi

current=$(echo "$output" | tr -d '\r' | awk 'NF { print $1; exit }')
case "$current" in
'' | *[!0-9]*)
	echo "migrate.sh: could not read the database version from: $output" >&2
	exit 1
	;;
esac

case "$output" in
*dirty*)
	cat >&2 <<MSG

  ✗ The database is marked dirty at migration $current.

    Migration $current started and did not finish, so the database may be part-way through it.
    Nothing has been changed now. This is not the "database ahead of the checkout" case, and its
    fix is different: inspect the database (make shell-db), repair or undo what $current did, then
    record the version you have actually reached — which clears the flag without running anything:

        docker compose --profile dev run --rm migrate force <version>

    If this is a development database you do not need, make reset starts again (it deletes all
    data).

MSG
	exit 4
	;;
esac

if [ "$current" -gt "$newest" ]; then
	cat >&2 <<MSG

  ✗ The database is ahead of this checkout.

    The database is at migration $current. This checkout only has migrations up to $newest.

    That happens when you run the stack on a branch that adds migrations and then switch back —
    reviewing a pull request, for instance. Your data is intact: nothing has been changed, and
    nothing will be until you choose which way to go.

    To go back to this checkout's version, move the database from the branch that added the
    newer migrations, since only it has the files that undo them:

        git switch <that branch>
        make migrate-to VERSION=$newest
        git switch -
        make up

    Or carry on with that branch instead: switch to it and run make up there.
    (make reset also clears this, by deleting every collectible and image. You should not need it.)

MSG
	exit 3
fi

# At or behind the newest migration here: apply what is pending, exactly as before (FR-007).
exec "$MIGRATE_BIN" -path "$MIGRATIONS_PATH" -database "$MIGRATE_DATABASE_URL" up
