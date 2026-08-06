#!/usr/bin/env bash
#
# demo.sh — start the demonstration system.
#
# It needs Go and nothing else. There is no database to install, no identity
# provider to configure and no build step: the in-memory store holds the whole
# scenario, and the SAPUI5 application is served by the same process.
#
#   ./demo.sh                 the demonstration, on http://localhost:8080
#   ./demo.sh --port 9000     somewhere else
#   ./demo.sh --postgres URL  against a real database instead
#   ./demo.sh --plan-only     the plan without the fortnight of factory life
#
# What it loads is the Kampong Speu 2026–2027 season — 2,300,000 t of cane over
# 137 days — generated through exactly the code path a planner uses, plus a
# fortnight of actuals and the factory life that goes with them: stoppages,
# production orders, stock movements, laboratory results, a cost run and the
# alerts they raise. Every row of it is produced by driving the services, so
# nothing on any screen is data that was written past the rules.
set -euo pipefail

PORT=8080
DSN=""
EXECUTION=true

while [[ $# -gt 0 ]]; do
    case "$1" in
        --port)      PORT="$2"; shift 2 ;;
        --postgres)  DSN="$2";  shift 2 ;;
        --plan-only) EXECUTION=false; shift ;;
        -h|--help)   awk 'NR>1 && /^#/ {sub(/^# ?/, ""); print; next} NR>1 {exit}' "$0"; exit 0 ;;
        *) echo "unknown option: $1" >&2; exit 2 ;;
    esac
done

cd "$(dirname "$0")/backend"

if ! command -v go > /dev/null; then
    echo "Go is required: https://go.dev/dl/" >&2
    exit 1
fi

export AUTH_MODE=dev
export AUTH_DEV_SECRET=local-development-secret
export SEED_DEMO=true
export SEED_EXECUTION="$EXECUTION"
export HTTP_STATIC_DIR=../frontend/webapp
export HTTP_ADDR=":$PORT"

if [[ -n "$DSN" ]]; then
    export STORE=postgres DATABASE_URL="$DSN"
    # Migrations are a separate, observable step in production and this keeps
    # them one here, rather than teaching the API to migrate on boot.
    echo "› migrating $DSN"
    go run ./cmd/migrate up
else
    export STORE=memory
fi

cat <<BANNER

  Sugar Production Planning — demonstration
  ─────────────────────────────────────────
  http://localhost:$PORT

  Sign in as any of these; the roles are real, and the difference between
  them is enforced by the server, not hidden by the screen:

    planner      Sokha Planner            builds and submits the plan
    approver     Dara Factory Manager     approves and releases it, and cannot write it
    supervisor   Vanna Shift Supervisor   records production and stoppages
    warehouse    Chanthou Warehouse       posts stock, and cannot touch the plan
    quality      Nary Laboratory          enters results, blocks and frees sugar
    controller   Mealea Cost Controller   runs the costing
    executive    Bopha Executive          reads everything, changes nothing
    auditor      Sovann Auditor           the only one who can read the audit trail

  Worth looking at:
    Executive overview   two real capacity warnings, and eight charts
    Downtime             a Pareto the boiler dominates — 62 % of the lost hours
    Production orders    one closed 60 t short, with the reason it needed
    Quality              a failed sample, the sugar it blocked, and its release
    Reports              fifteen of them, in CSV, Excel and PDF

  Ctrl-C to stop.

BANNER

exec go run ./cmd/server
