#!/usr/bin/env bash
# Start, stop or restart the Farm Area API locally.
#
# It keeps a pid file rather than matching on the process name: "pkill -f api" also matches the
# shell that is running the command, which kills the caller instead of the server.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(dirname "$here")"
pidfile="${FARMAREA_PIDFILE:-/tmp/farm-area-api.pid}"
logfile="${FARMAREA_LOGFILE:-/tmp/farm-area-api.log}"

export FARMAREA_DATABASE_URL="${FARMAREA_DATABASE_URL:-postgres://farmarea:farmarea@127.0.0.1:5432/farmarea}"
export FARMAREA_JWT_SECRET="${FARMAREA_JWT_SECRET:-a-development-signing-key-at-least-32-chars}"
export FARMAREA_ADDR="${FARMAREA_ADDR:-:8080}"
export FARMAREA_CORS_ORIGINS="${FARMAREA_CORS_ORIGINS:-http://localhost:8081,http://127.0.0.1:8081}"
export FARMAREA_MIGRATIONS_DIR="${FARMAREA_MIGRATIONS_DIR:-$root/db/migrations}"

stop() {
    if [[ -f "$pidfile" ]]; then
        pid="$(cat "$pidfile")"
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            for _ in $(seq 1 20); do
                kill -0 "$pid" 2>/dev/null || break
                sleep 0.25
            done
        fi
        rm -f "$pidfile"
    fi
}

start() {
    stop
    cd "$root/backend"
    go build -o /tmp/farm-area-api ./cmd/api
    setsid /tmp/farm-area-api >"$logfile" 2>&1 < /dev/null &
    echo $! > "$pidfile"

    for _ in $(seq 1 40); do
        if curl -fsS -m 2 "http://127.0.0.1${FARMAREA_ADDR}/health" >/dev/null 2>&1; then
            echo "api listening on ${FARMAREA_ADDR}"
            return 0
        fi
        sleep 0.5
    done
    echo "api did not become healthy; last log lines:" >&2
    tail -20 "$logfile" >&2
    return 1
}

case "${1:-start}" in
    start)   start ;;
    stop)    stop; echo "api stopped" ;;
    restart) start ;;
    log)     tail -f "$logfile" ;;
    *)       echo "usage: $0 {start|stop|restart|log}" >&2; exit 2 ;;
esac
