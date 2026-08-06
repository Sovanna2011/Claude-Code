#!/usr/bin/env bash
#
# test-system.sh — boot a test system and check it against the acceptance
# criteria in section 27 of the specification.
#
#   ./test-system.sh                 the whole run, in memory
#   ./test-system.sh --postgres URL  against a real database, with the
#                                    backup and restore drill included
#   ./test-system.sh --keep          leave the instance running afterwards
#   ./test-system.sh --port 9100     somewhere else
#
# This is not the demonstration. The demonstration is one company at one
# factory, because that is the reference scenario and its figures are meant to
# be quotable. The test system adds a second mill — Battambang, its own season,
# its own stores, deliberately different numbers — and narrows every account to
# one of them, so that "a planner at one factory cannot read another factory's
# plan" is a claim two real tenants can be held to rather than one checked
# against a factory that does not exist.
#
# What it runs, in order:
#
#   1. go vet and the Go test suite
#   2. govulncheck, if it is installed
#   3. the frontend unit tests, if node is installed
#   4. the server, seeded with both tenants
#   5. cmd/acceptance against it, over HTTP
#   6. with --postgres: a dump, a restore into a scratch database, and a
#      comparison of the row counts
#
# It exits non-zero if any of them fails, which is what makes the eleventh
# criterion — "automated tests pass, no critical security findings remain, and
# backup/restore has been demonstrated" — something a run can be held to rather
# than a sentence in a document.
set -uo pipefail

PORT=8080
DSN=""
KEEP=false
SKIP_SUITES=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --port)       PORT="$2"; shift 2 ;;
        --postgres)   DSN="$2";  shift 2 ;;
        --keep)       KEEP=true; shift ;;
        --acceptance-only) SKIP_SUITES=true; shift ;;
        -h|--help)    awk 'NR>1 && /^#/ {sub(/^# ?/, ""); print; next} NR>1 {exit}' "$0"; exit 0 ;;
        *) echo "unknown option: $1" >&2; exit 2 ;;
    esac
done

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT/backend"

WORK="$(mktemp -d)"
SERVER_PID=""
FAILURES=()

cleanup() {
    if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2> /dev/null; then
        if [[ "$KEEP" == true ]]; then
            echo "› the instance is still running on http://localhost:$PORT (pid $SERVER_PID)"
            return
        fi
        kill "$SERVER_PID" 2> /dev/null || true
        wait "$SERVER_PID" 2> /dev/null || true
    fi
    [[ "$KEEP" == true ]] || rm -rf "$WORK"
}
trap cleanup EXIT

step() { printf '\n\033[1m› %s\033[0m\n' "$*"; }
record() { FAILURES+=("$1"); printf '\033[31m  FAIL %s\033[0m\n' "$1"; }

# ---------------------------------------------------------------------------
# 1-3. The suites. Criterion 11 asks for these, and a harness that skipped them
#      and reported eleven criteria met would be lying by omission.
# ---------------------------------------------------------------------------

if [[ "$SKIP_SUITES" != true ]]; then
    step "go vet"
    go vet ./... || record "go vet"

    step "go test ./..."
    go test ./... || record "go test"

    step "govulncheck"
    if command -v govulncheck > /dev/null; then
        govulncheck ./... || record "govulncheck"
    else
        echo "  not installed — go install golang.org/x/vuln/cmd/govulncheck@latest"
        echo "  (this leaves 'no critical security findings' unchecked, not met)"
    fi

    step "frontend unit tests"
    if command -v node > /dev/null; then
        (cd "$ROOT/frontend" && npm test) || record "npm test"
    else
        echo "  node is not installed — the frontend unit tests did not run"
    fi
fi

# ---------------------------------------------------------------------------
# 4. The instance, with both tenants.
# ---------------------------------------------------------------------------

step "starting the test system on port $PORT"

export AUTH_MODE=dev
export AUTH_DEV_SECRET=local-development-secret
export SEED_DEMO=true
export SEED_EXECUTION=true
export SEED_TENANTS=true
export HTTP_STATIC_DIR=../frontend/webapp
export HTTP_ADDR=":$PORT"
# The rate limiter is generous by default and the harness makes a few hundred
# requests in a few seconds; raising it here keeps a throttle from reading as a
# failed criterion.
export HTTP_RATE_LIMIT=100000

if [[ -n "$DSN" ]]; then
    export STORE=postgres DATABASE_URL="$DSN"
    echo "  migrating $DSN"
    go run ./cmd/migrate up || { record "migrate"; exit 1; }
else
    export STORE=memory
fi

# Build a real binary rather than `go run`: a `go run` wrapper is not the
# process that holds the port, so killing it leaves the server behind and the
# next run finds the port taken.
go build -o "$WORK/server" ./cmd/server || { record "build the server"; exit 1; }
go build -o "$WORK/acceptance" ./cmd/acceptance || { record "build the harness"; exit 1; }

"$WORK/server" > "$WORK/server.log" 2>&1 &
SERVER_PID=$!

# ---------------------------------------------------------------------------
# 5. The acceptance criteria, over HTTP.
# ---------------------------------------------------------------------------

step "acceptance criteria"
if ! "$WORK/acceptance" -base "http://localhost:$PORT"; then
    record "acceptance criteria"
    echo
    echo "  the server log:"
    tail -40 "$WORK/server.log" | sed 's/^/    /'
fi

# ---------------------------------------------------------------------------
# 6. Backup and restore, when there is a database to do it to.
# ---------------------------------------------------------------------------

if [[ -n "$DSN" ]]; then
    step "backup and restore drill"
    if ! command -v pg_dump > /dev/null || ! command -v psql > /dev/null; then
        echo "  pg_dump or psql is missing — the drill did not run"
        record "backup/restore (tools missing)"
    else
        SCRATCH="sugarplan_restore_$$"
        DUMP="$WORK/sugarplan.dump"
        # Swap the database name, keeping any connection parameters: dropping
        # ?sslmode=... would make the drill fail against a server that requires
        # it, and look like a restore problem rather than a connection one.
        BASE="${DSN%%\?*}"
        QUERY=""
        [[ "$DSN" == *\?* ]] && QUERY="?${DSN#*\?}"
        ADMIN="${BASE%/*}/postgres$QUERY"
        SCRATCH_DSN="${BASE%/*}/$SCRATCH$QUERY"

        if pg_dump --format=custom --file="$DUMP" "$DSN" \
            && pg_restore --list "$DUMP" > /dev/null \
            && psql -q "$ADMIN" -c "CREATE DATABASE \"$SCRATCH\"" \
            && pg_restore --no-owner -d "$SCRATCH_DSN" "$DUMP" > /dev/null 2>&1; then

            # A restore nobody has counted is a hope. Three tables across the
            # planning, execution and audit halves, because a dump that restores
            # empty passes a pg_restore and fails a plant.
            #
            # The count is required to be a number: the first version of this
            # asked a table that does not exist, psql printed an error, and two
            # empty strings compared equal and reported a passing drill.
            COUNTS="SELECT (SELECT count(*) FROM daily_cane_plans),
                           (SELECT count(*) FROM inventory_documents),
                           (SELECT count(*) FROM audit_events)"
            ORIGINAL="$(psql -tAX "$DSN" -c "$COUNTS" 2> /dev/null)"
            RESTORED="$(psql -tAX "$SCRATCH_DSN" -c "$COUNTS" 2> /dev/null)"
            echo "  cane rows / stock documents / audit events"
            echo "    original: ${ORIGINAL:-<nothing>}"
            echo "    restored: ${RESTORED:-<nothing>}"
            if [[ ! "$ORIGINAL" =~ ^[1-9][0-9]*\|[0-9]+\|[1-9][0-9]*$ ]]; then
                record "backup/restore (the original database did not answer with counts)"
            elif [[ "$ORIGINAL" != "$RESTORED" ]]; then
                record "backup/restore (the restore does not match the original)"
            else
                echo "  the restore matches"
            fi
            psql -q "$ADMIN" -c "DROP DATABASE \"$SCRATCH\"" > /dev/null 2>&1 || true
        else
            record "backup/restore"
            psql -q "$ADMIN" -c "DROP DATABASE IF EXISTS \"$SCRATCH\"" > /dev/null 2>&1 || true
        fi
    fi
fi

# ---------------------------------------------------------------------------

echo
printf '%s\n' "──────────────────────────────────────────────────────────────────────────────"
if [[ ${#FAILURES[@]} -eq 0 ]]; then
    printf '\033[32mthe test system passed everything it ran\033[0m\n'
    [[ -n "$DSN" ]] || echo "run again with --postgres URL to include the backup and restore drill"
    exit 0
fi
printf '\033[31m%d failed: %s\033[0m\n' "${#FAILURES[@]}" "$(printf '%s, ' "${FAILURES[@]}" | sed 's/, $//')"
exit 1
