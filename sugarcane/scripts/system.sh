#!/usr/bin/env bash
# Start, stop, restart or inspect the whole system: PostgreSQL/PostGIS, the Go API and the SAPUI5
# server. One command, because after a machine or container restart all three are down and starting
# them in the wrong order gives an API that cannot reach its database.
#
#   ./scripts/system.sh start      bring everything up (creates the database on first run)
#   ./scripts/system.sh stop       shut the API and the web server down; PostgreSQL is left running
#   ./scripts/system.sh restart    stop, then start
#   ./scripts/system.sh status     what is up, and on which port
#   ./scripts/system.sh log        follow the API log
#
# Two habits this script keeps, both learned the hard way:
#   - processes are tracked by pid file, never by `pkill -f`, which matches the shell running the
#     command and kills the caller instead of the server;
#   - every service is waited for until it actually answers, rather than assumed up after a sleep.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(dirname "$here")"

PG_VERSION="${FARMAREA_PG_VERSION:-16}"
PG_CLUSTER="${FARMAREA_PG_CLUSTER:-main}"
DB_NAME="${FARMAREA_DB_NAME:-farmarea}"
DB_USER="${FARMAREA_DB_USER:-farmarea}"
DB_PASSWORD="${FARMAREA_DB_PASSWORD:-farmarea}"
DB_HOST="${FARMAREA_DB_HOST:-127.0.0.1}"
DB_PORT="${FARMAREA_DB_PORT:-5432}"

UI5_PORT="${FARMAREA_UI5_PORT:-8081}"
UI5_PIDFILE="${FARMAREA_UI5_PIDFILE:-/tmp/farm-area-ui5.pid}"
UI5_LOGFILE="${FARMAREA_UI5_LOGFILE:-/tmp/farm-area-ui5.log}"

export FARMAREA_DATABASE_URL="${FARMAREA_DATABASE_URL:-postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME}"
export FARMAREA_ADDR="${FARMAREA_ADDR:-:8080}"

say() { printf '  %-12s %s\n' "$1" "$2"; }

# The pid actually listening on a port. Used to clear a server this script does not own — one
# started by hand in another shell, whose pid file we never wrote. Matching on the listening socket
# rather than the command line is the safe way to do it: `pkill -f ui5` also matches the shell
# running this script.
# Distributions disagree about which of these is installed — ss is absent from slim container
# images, lsof from some minimal hosts — so each is tried in turn rather than assumed.
port_pid() {
    local port="$1" pid=""
    if command -v ss >/dev/null 2>&1; then
        pid="$(ss -ltnpH "sport = :$port" 2>/dev/null | grep -oE 'pid=[0-9]+' | head -1 | cut -d= -f2)"
    fi
    if [[ -z "$pid" ]] && command -v lsof >/dev/null 2>&1; then
        pid="$(lsof -t -i "tcp:$port" -s TCP:LISTEN 2>/dev/null | head -1)"
    fi
    if [[ -z "$pid" ]] && command -v fuser >/dev/null 2>&1; then
        pid="$(fuser "$port/tcp" 2>/dev/null | tr -s ' ' '\n' | grep -E '^[0-9]+$' | head -1)"
    fi
    printf '%s' "$pid"
}

free_port() {
    local port="$1" label="$2" pid
    pid="$(port_pid "$port")"
    [[ -z "$pid" ]] && return 0
    say "$label" "port $port held by pid $pid, which this script did not start — stopping it"
    kill "$pid" 2>/dev/null || true
    for _ in $(seq 1 20); do
        [[ -z "$(port_pid "$port")" ]] && return 0
        sleep 0.25
    done
    kill -9 "$pid" 2>/dev/null || true
}

# ---------------------------------------------------------------- PostgreSQL

postgres_running() { pg_isready -h "$DB_HOST" -p "$DB_PORT" -q 2>/dev/null; }

start_postgres() {
    if postgres_running; then
        say "postgres" "already up on $DB_PORT"
        return 0
    fi

    # A container that was reclaimed mid-run leaves a pid file behind; pg_ctlcluster clears it.
    if command -v pg_ctlcluster >/dev/null 2>&1; then
        pg_ctlcluster "$PG_VERSION" "$PG_CLUSTER" start >/dev/null 2>&1 || true
    fi
    if ! postgres_running && [[ -d "/var/lib/postgresql/$PG_VERSION/$PG_CLUSTER" ]]; then
        su postgres -c "/usr/lib/postgresql/$PG_VERSION/bin/pg_ctl \
            -D /var/lib/postgresql/$PG_VERSION/$PG_CLUSTER \
            -l /var/log/postgresql/postgresql-$PG_VERSION-$PG_CLUSTER.log start" >/dev/null 2>&1 || true
    fi

    for _ in $(seq 1 40); do
        postgres_running && break
        sleep 0.5
    done
    if ! postgres_running; then
        echo "postgres did not start; see /var/log/postgresql/" >&2
        return 1
    fi
    say "postgres" "started on $DB_PORT"
}

# The role, the database and PostGIS, created only if they are not already there. This is what makes
# the script work on a machine that has never run the system before.
provision_database() {
    local psql_super=(su postgres -c)

    if ! "${psql_super[@]}" "psql -tAc \"SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'\"" | grep -q 1; then
        "${psql_super[@]}" "psql -c \"CREATE ROLE $DB_USER LOGIN PASSWORD '$DB_PASSWORD'\"" >/dev/null
        say "role" "created $DB_USER"
    fi

    for db in "$DB_NAME" "${DB_NAME}_test"; do
        if ! "${psql_super[@]}" "psql -tAc \"SELECT 1 FROM pg_database WHERE datname='$db'\"" | grep -q 1; then
            "${psql_super[@]}" "createdb -O $DB_USER $db" >/dev/null
            say "database" "created $db"
        fi
        "${psql_super[@]}" \
            "psql -q -d $db -c 'SET client_min_messages=warning; CREATE EXTENSION IF NOT EXISTS postgis'" >/dev/null
    done
}

# ---------------------------------------------------------------- SAPUI5 server

ui5_running() { curl -fsS -m 2 "http://127.0.0.1:$UI5_PORT/index.html" >/dev/null 2>&1; }

stop_ui5() {
    if [[ -f "$UI5_PIDFILE" ]]; then
        local pid; pid="$(cat "$UI5_PIDFILE")"
        if kill -0 "$pid" 2>/dev/null; then
            # setsid puts the server in its own process group; negating the pid takes the whole
            # group, so npx's child dies with it rather than holding the port.
            kill -- "-$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
            for _ in $(seq 1 20); do
                kill -0 "$pid" 2>/dev/null || break
                sleep 0.25
            done
        fi
        rm -f "$UI5_PIDFILE"
    fi
    # Whatever the pid file said, the port is what matters: a server left over from a hand-started
    # run would otherwise keep answering while this script reported it stopped.
    free_port "$UI5_PORT" "ui5"
}

start_ui5() {
    stop_ui5
    if [[ ! -d "$root/frontend/node_modules" ]]; then
        say "ui5" "installing dependencies, this takes a minute"
        (cd "$root/frontend" && npm install --silent) || {
            echo "npm install failed; the UI5 runtime is vendored, so the app cannot start without it" >&2
            return 1
        }
    fi

    # --fork matters. Plain `setsid cmd &` execs the server in setsid's own process, so it stays a
    # child of this script — and a script whose stdout is a pipe then blocks in wait() forever,
    # never reaching its own exit. Forking reparents the server to init immediately, leaving this
    # script nothing to wait for.
    ( cd "$root/frontend" && setsid --fork npx ui5 serve --port "$UI5_PORT" \
        >"$UI5_LOGFILE" 2>&1 < /dev/null & )

    for _ in $(seq 1 60); do
        if ui5_running; then
            # The pid that actually holds the port, not setsid's — which has already exited.
            port_pid "$UI5_PORT" > "$UI5_PIDFILE"
            say "ui5" "serving on http://localhost:$UI5_PORT"
            return 0
        fi
        sleep 0.5
    done
    echo "the UI5 server did not answer; last log lines:" >&2
    tail -20 "$UI5_LOGFILE" >&2
    return 1
}

# ---------------------------------------------------------------- the whole stack

start() {
    echo "Starting the sugarcane planting planning system"
    start_postgres
    provision_database
    # The API applies any pending migrations on the way up, so a new migration needs no extra step.
    free_port "${FARMAREA_ADDR#:}" "api"
    "$here/api.sh" start | sed 's/^/  api          /'
    start_ui5
    echo
    status
}

stop() {
    "$here/api.sh" stop >/dev/null 2>&1 || true
    free_port "${FARMAREA_ADDR#:}" "api"
    say "api" "stopped"
    stop_ui5
    say "ui5" "stopped"
    say "postgres" "left running — stop it with: pg_ctlcluster $PG_VERSION $PG_CLUSTER stop"
}

status() {
    echo "Status"
    if postgres_running; then
        local blocks
        blocks="$(psql "$FARMAREA_DATABASE_URL" -tAc 'SELECT count(*) FROM block' 2>/dev/null || echo '?')"
        say "postgres" "up on $DB_PORT · $DB_NAME · $blocks blocks"
    else
        say "postgres" "down"
    fi

    if curl -fsS -m 2 "http://127.0.0.1${FARMAREA_ADDR}/health" >/dev/null 2>&1; then
        say "api" "up on http://localhost${FARMAREA_ADDR}"
    else
        say "api" "down"
    fi

    if ui5_running; then
        say "ui5" "up on http://localhost:$UI5_PORT"
        echo
        echo "  Sign in at http://localhost:$UI5_PORT — admin · manager · planner · viewer, password Farm#2026"
    else
        say "ui5" "down"
    fi
}

case "${1:-start}" in
    start)   start ;;
    stop)    stop ;;
    restart) stop; echo; start ;;
    status)  status ;;
    log)     "$here/api.sh" log ;;
    *)       echo "usage: $0 {start|stop|restart|status|log}" >&2; exit 2 ;;
esac
