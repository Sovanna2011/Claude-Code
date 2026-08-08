#!/bin/bash
# Bring the demonstration stack up from whatever state it is in.
# The scratchpad's permissions and the background processes do not survive
# between shells here, so this is idempotent and safe to re-run.
set -e
SP=/tmp/claude-0/-home-user-Claude-Code/f64650e8-4ec0-5e17-86c2-e4faedef65d5/scratchpad
PGBIN=$(ls /usr/lib/postgresql/*/bin/pg_ctl | head -1)
export PGHOST=/tmp PGPORT=5433 PGUSER=postgres
export DATABASE_URL='postgres://postgres@/sugarplan_hub?host=/tmp&port=5433&sslmode=disable'
export AUTH_MODE=dev AUTH_DEV_SECRET=dev-secret-for-local-testing
export SEED_ACTUAL_DAYS=14 SEED_DEMO=true SEED_EXECUTION=true

chmod o+x /tmp/claude-0 /tmp/claude-0/-home-user-Claude-Code \
  /tmp/claude-0/-home-user-Claude-Code/f64650e8-4ec0-5e17-86c2-e4faedef65d5 "$SP" 2>/dev/null || true

if ! psql -Atc "select 1" >/dev/null 2>&1; then
  rm -f /tmp/.s.PGSQL.5433 /tmp/.s.PGSQL.5433.lock
  su postgres -c "$PGBIN -D $SP/pgdata -o '-p 5433 -k /tmp' -l $SP/pg.log start" >/dev/null
  sleep 3
fi

if [ "$1" = "--fresh" ]; then
  pkill -f "$SP/server" || true; sleep 1
  dropdb --if-exists sugarplan_hub; createdb sugarplan_hub
  $SP/migrate up >/dev/null
fi

if ! curl -s -o /dev/null localhost:8080/healthz; then
  setsid nohup $SP/server > $SP/hub-server.log 2>&1 < /dev/null &
  for i in $(seq 1 20); do sleep 1; curl -s -o /dev/null localhost:8080/healthz && break; done
fi

B=http://localhost:8080/api/v1
tok(){ curl -s -X POST $B/auth/dev-login -H 'Content-Type: application/json' -d "{\"username\":\"$1\"}" \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['accessToken'])"; }
T=$(tok planner)
S=$(curl -s $B/seasons -H "Authorization: Bearer $T" | python3 -c 'import sys,json;print(json.load(sys.stdin)["value"][0]["id"])')
HAVE=$(curl -s $B/seasons/$S/versions -H "Authorization: Bearer $T" | python3 -c 'import sys,json;print(len(json.load(sys.stdin)["value"]))')
if [ "$HAVE" -lt 4 ]; then
  V1=$(curl -s $B/seasons/$S/versions -H "Authorization: Bearer $T" | python3 -c "import sys,json;print([v['id'] for v in json.load(sys.stdin)['value'] if v['code']=='V1'][0])")
  for SPEC in 'V2|A drier season: recovery at 9.5 %|RAW_RECOVERY_PCT|9.5' 'V3|Cane supply short: 2.0 Mt|CANE_TARGET_TONS|2000000'; do
    IFS='|' read -r CODE DESC KEY VAL <<< "$SPEC"
    NEW=$(curl -s -X POST "$B/versions/$V1/copy" -H "Authorization: Bearer $T" -H 'Content-Type: application/json' \
      -d "{\"sourceVersionId\":\"$V1\",\"code\":\"$CODE\",\"description\":\"$DESC\",\"planType\":\"FORECAST\",\"assumptionOverrides\":{\"$KEY\":\"$VAL\"}}" \
      | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
    curl -s -o /dev/null -X POST "$B/versions/$NEW/generate" -H "Authorization: Bearer $T" \
      -H 'Content-Type: application/json' -d '{"replace":true}'
  done
fi
echo "up: $(curl -s -o /dev/null -w '%{http_code}' localhost:8080/healthz), versions: $(curl -s $B/seasons/$S/versions -H "Authorization: Bearer $T" | python3 -c 'import sys,json;print(",".join(v["code"] for v in sorted(json.load(sys.stdin)["value"], key=lambda v:v["versionNo"])))')"
