#!/usr/bin/env bash
#
# load-test.sh — hold the system to the performance targets in section 25, at
# the history volume section 25 asks it to carry.
#
#   ./load-test.sh --postgres URL          ten seasons, then measure
#   ./load-test.sh --postgres URL --seasons 20
#   ./load-test.sh --postgres URL --docs-per-day 400
#   ./load-test.sh --postgres URL --measure-only
#
# The targets, quoted rather than invented:
#
#   list APIs   p95 under 500 ms   for normal indexed filters
#   dashboards  p95 under 2 s
#
# at "at least 10 years of daily history and high-volume transaction and audit
# data".
#
# It needs PostgreSQL. There is no in-memory variant on purpose: the numbers
# this exists to produce are about indexes, query plans and a connection pool,
# and a map in a process has none of those.
#
# Every endpoint is measured twice — alone, and under concurrent load. Alone is
# the query; under load is the machine. A report that gave only the second
# would blame an index for a small server.
set -uo pipefail

PORT=8090
DSN=""
SEASONS=10
DOCS=150
CONCURRENCY=8
MEASURE_ONLY=false
KEEP=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --postgres)     DSN="$2"; shift 2 ;;
        --port)         PORT="$2"; shift 2 ;;
        --seasons)      SEASONS="$2"; shift 2 ;;
        --docs-per-day) DOCS="$2"; shift 2 ;;
        --concurrency)  CONCURRENCY="$2"; shift 2 ;;
        --measure-only) MEASURE_ONLY=true; shift ;;
        --keep)         KEEP=true; shift ;;
        -h|--help)      awk 'NR>1 && /^#/ {sub(/^# ?/, ""); print; next} NR>1 {exit}' "$0"; exit 0 ;;
        *) echo "unknown option: $1" >&2; exit 2 ;;
    esac
done

if [[ -z "$DSN" ]]; then
    echo "--postgres URL is required; see --help" >&2
    exit 2
fi

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT/backend"

WORK="$(mktemp -d)"
SERVER_PID=""

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

export AUTH_MODE=dev
export AUTH_DEV_SECRET=local-development-secret
export STORE=postgres DATABASE_URL="$DSN"
export HTTP_ADDR=":$PORT"
# The measurement is of the endpoints, not of the throttle in front of them.
export HTTP_RATE_LIMIT=1000000

step "migrating"
go run ./cmd/migrate up || exit 1

go build -o "$WORK/server" ./cmd/server || exit 1
go build -o "$WORK/loadtest" ./cmd/loadtest || exit 1

if [[ "$MEASURE_ONLY" != true ]]; then
    # The reference scenario first: the fixture extends the seeded factory
    # rather than inventing an organisation, so it needs the master data.
    step "seeding the reference scenario"
    SEED_DEMO=true SEED_EXECUTION=true "$WORK/server" > "$WORK/seed.log" 2>&1 &
    SEED_PID=$!
    for _ in $(seq 1 60); do
        curl -fsS "http://localhost:$PORT/readyz" > /dev/null 2>&1 && break
        sleep 1
    done
    if ! curl -fsS "http://localhost:$PORT/readyz" > /dev/null 2>&1; then
        echo "the server did not become ready:"; tail -20 "$WORK/seed.log"; exit 1
    fi
    kill "$SEED_PID" 2> /dev/null; wait "$SEED_PID" 2> /dev/null

    step "generating $SEASONS seasons of history"
    "$WORK/loadtest" -generate -seasons "$SEASONS" -docs-per-day "$DOCS" -dsn "$DSN" || exit 1
fi

step "starting the instance"
"$WORK/server" > "$WORK/server.log" 2>&1 &
SERVER_PID=$!

step "measuring"
"$WORK/loadtest" -measure -base "http://localhost:$PORT" -concurrency "$CONCURRENCY"
code=$?

if [[ $code -ne 0 ]]; then
    echo
    echo "  the server log:"
    tail -30 "$WORK/server.log" | sed 's/^/    /'
fi
exit $code
