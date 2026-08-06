# Operational runbook

Environment variables, deployment, backup and restore, and what to do when
something is wrong.

---

## 1. Environment variable reference

Nothing here has a usable default for production. The application validates its
configuration at start-up and exits with a list of problems rather than running
in a state it cannot honour.

### Application

| Variable | Default | Notes |
| --- | --- | --- |
| `APP_ENV` | `development` | `development`, `test`, `uat`, `production`. Production forbids the development sign-in, the in-memory store and demo data |
| `APP_VERSION` | `dev` | Reported by `/healthz`. Set it to the build tag |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` for shipping, `text` for a terminal |
| `FACTORY_TIMEZONE` | `Asia/Phnom_Penh` | An IANA zone; validated at start-up |

### HTTP

| Variable | Default | Notes |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | |
| `HTTP_REQUEST_TIMEOUT` | `30s` | Per-request deadline |
| `HTTP_SHUTDOWN_TIMEOUT` | `20s` | Drain time on SIGTERM |
| `HTTP_ALLOWED_ORIGINS` | empty | Only for a genuinely cross-origin frontend. Empty means same-origin only |
| `HTTP_RATE_LIMIT` | `600` | Requests per interval per caller |
| `HTTP_RATE_INTERVAL` | `1m` | |
| `HTTP_STATIC_DIR` | empty | Path to the built SAPUI5 files, when one process serves both |

### Database

| Variable | Default | Notes |
| --- | --- | --- |
| `STORE` | `postgres` | `postgres` or `memory`. `memory` is demo and test only |
| `DATABASE_URL` | — | Required for `postgres`. `postgres://user:pass@host:5432/db?sslmode=require` |
| `DB_MAX_CONNS` | `20` | Pool size. Keep `instances × DB_MAX_CONNS` below the server's `max_connections` |
| `DB_MIN_CONNS` | `2` | |
| `DB_CONN_LIFETIME` | `1h` | |
| `DB_CONNECT_TIMEOUT` | `10s` | |
| `DB_MIGRATE_ON_START` | `false` | Development convenience. In production migration is a separate step |

### Authentication

| Variable | Default | Notes |
| --- | --- | --- |
| `AUTH_MODE` | `oidc` | `oidc` or `dev`. `dev` is refused in production |
| `AUTH_ISSUER` | — | Required for `oidc`. Its discovery document is read at start-up |
| `AUTH_AUDIENCE` | empty | Expected `aud` claim |
| `AUTH_ROLES_CLAIM` | `roles` | Dotted path. Keycloak uses `realm_access.roles` |
| `AUTH_ROLE_MAPPING` | empty | `provider-group=APP_ROLE,other=OTHER_ROLE` |
| `AUTH_DEV_SECRET` | — | Required for `dev`. At least 16 characters |

### Interfaces and background jobs

| Variable | Default | Notes |
| --- | --- | --- |
| `INTEGRATION_ENDPOINT` | *(none)* | Where integration events are POSTed. With none set they go to the application log, which is a real destination, not a silent drop |
| `INTEGRATION_AUTH_HEADER` | *(none)* | e.g. `Authorization`. Set with `INTEGRATION_AUTH_VALUE` or not at all |
| `INTEGRATION_AUTH_VALUE` | *(none)* | e.g. `Bearer …`. A secret: inject it, never bake it in |
| `INTEGRATION_TIMEOUT` | `15s` | One delivery attempt |
| `INTEGRATION_SOURCE` | `sugarplan` | Names this system in the envelope |
| `INTEGRATION_DISPATCH_INTERVAL` | `30s` | How often the dispatcher runs. `0` switches it off and says so in the log |
| `INTEGRATION_DISPATCH_BATCH` | `50` | Events per pass |
| `ALERT_INTERVAL` | `15m` | How often the alerts are evaluated into people's inboxes. `0` switches it off, which leaves alerts on the dashboard reaching nobody — said out loud in the log |

The dispatcher runs inside the server process. Two instances behind a load
balancer both have one, and each job is taken under a lease in `job_leases`, so
only one instance runs it per tick. An instance that dies holding a lease blocks
its job only until the lease expires, not until somebody notices.

### Seed data

| Variable | Default | Notes |
| --- | --- | --- |
| `SEED_DEMO` | `false` | Loads the Kampong Speu scenario. Refused in production |
| `SEED_ACTUAL_DAYS` | `14` | Days of demonstration actuals |

---

## 2. Deployment

### First install

```bash
# 1. Create the database and a least-privilege role.
psql -h db -U postgres <<'SQL'
CREATE DATABASE sugarplan;
CREATE ROLE sugarplan_app LOGIN PASSWORD '<from the secret store>';
GRANT CONNECT ON DATABASE sugarplan TO sugarplan_app;
SQL

# 2. Migrate. This is a job that runs to completion, not something the API does.
docker run --rm \
  -e STORE=postgres -e DATABASE_URL="$DATABASE_URL" \
  -e APP_ENV=production -e AUTH_MODE=oidc -e AUTH_ISSUER="$AUTH_ISSUER" \
  --entrypoint /app/migrate sugarplan:1.0.0 up

# 3. Grant the application role what it needs, and no more.
psql -h db -U postgres -d sugarplan <<'SQL'
GRANT USAGE ON SCHEMA public TO sugarplan_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO sugarplan_app;
-- The audit trail is append-only for the application. The database rules make
-- this structural; the grant makes it explicit.
REVOKE UPDATE, DELETE ON audit_events FROM sugarplan_app;
SQL

# 4. Start the API.
docker run -d --name sugarplan-api -p 8080:8080 \
  -e APP_ENV=production -e APP_VERSION=1.0.0 \
  -e STORE=postgres -e DATABASE_URL="$DATABASE_URL" \
  -e AUTH_MODE=oidc -e AUTH_ISSUER="$AUTH_ISSUER" -e AUTH_AUDIENCE="$AUTH_AUDIENCE" \
  sugarplan:1.0.0

# 5. Verify.
curl -fsS http://localhost:8080/healthz   # process
curl -fsS http://localhost:8080/readyz    # database
```

### Upgrade

```bash
docker pull sugarplan:1.1.0
# Migrate first, then roll the instances. The migration is additive within a
# minor version, so the old binary keeps working while the roll proceeds.
docker run --rm -e STORE=postgres -e DATABASE_URL="$DATABASE_URL" \
  -e APP_ENV=production -e AUTH_MODE=oidc -e AUTH_ISSUER="$AUTH_ISSUER" \
  --entrypoint /app/migrate sugarplan:1.1.0 up
# Then roll, gating each instance on /readyz.
```

### Rollback

```bash
# Roll the application back first.
docker run -d ... sugarplan:1.0.0

# Reverse a migration only if the new version's schema is incompatible, and
# only after a backup. The command refuses to run with APP_ENV=production, so
# this is a deliberate act.
docker run --rm -e STORE=postgres -e DATABASE_URL="$DATABASE_URL" \
  -e APP_ENV=uat -e AUTH_MODE=oidc -e AUTH_ISSUER="$AUTH_ISSUER" \
  --entrypoint /app/migrate sugarplan:1.1.0 down -steps 1
```

---

## 3. Backup and restore

### What has to be backed up

Only PostgreSQL. The API is stateless and the containers are disposable.

### Nightly

```bash
pg_dump --format=custom --compress=9 \
        --file="/backup/sugarplan-$(date +%Y%m%d).dump" \
        "$DATABASE_URL"

# Verify the dump can be read before trusting it. A backup nobody has read is a
# hope, not a backup.
pg_restore --list "/backup/sugarplan-$(date +%Y%m%d).dump" > /dev/null
```

### Point in time recovery

Continuous archiving is what makes "restore to just before the mistake"
possible:

```
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'test ! -f /wal/%f && cp %p /wal/%f'
```

Recover to a moment:

```bash
pg_restore --clean --if-exists -d sugarplan /backup/sugarplan-20270115.dump
# then, for PITR, a recovery.signal with:
#   restore_command = 'cp /wal/%f %p'
#   recovery_target_time = '2027-01-15 06:30:00+07'
```

### Retention

| Backup | Kept |
| --- | --- |
| Nightly full | 30 days |
| Weekly full | 12 weeks |
| Monthly full | 24 months |
| Season-end full | 10 years (section 23) |
| WAL archive | 30 days |

### Restore rehearsal — quarterly, not optional

```bash
createdb sugarplan_restore_test
pg_restore -d sugarplan_restore_test /backup/sugarplan-latest.dump

# Prove the data is actually there and internally consistent.
psql -d sugarplan_restore_test <<'SQL'
SELECT count(*) AS seasons FROM seasons;
SELECT count(*) AS versions FROM plan_versions;
SELECT count(*) AS audit_events FROM audit_events;

-- Ledger continuity: every day's beginning balance must equal the previous
-- day's ending balance. This should return no rows.
SELECT warehouse_id, product_id, business_date
FROM (
  SELECT warehouse_id, product_id, business_date, beginning_balance,
         lag(ending_balance) OVER (
           PARTITION BY version_id, warehouse_id, product_id, series
           ORDER BY business_date) AS prev_ending
  FROM daily_storage_plans
) t
WHERE prev_ending IS NOT NULL AND beginning_balance <> prev_ending;
SQL

dropdb sugarplan_restore_test
```

Record the restore time. That number is the recovery time objective, and it is
only real once it has been measured.

---

## 4. Monitoring

### Signals

| Signal | Alert when |
| --- | --- |
| `/readyz` failing | 2 consecutive failures — the database is unreachable |
| HTTP 5xx rate | above 1 % over 5 minutes |
| p95 list latency | above 500 ms over 10 minutes |
| p95 dashboard latency | above 2 s over 10 minutes |
| Database connections | above 80 % of `max_connections` |
| Replication lag | above 30 seconds |
| Backup age | no successful backup in 26 hours |
| Unpublished outbox events | older than 15 minutes |
| Exhausted outbox events | any at all — they have stopped being retried and are waiting for a person (`GET /api/v1/integration/events?exhausted=true`) |
| Outbox dispatcher | `finishedAt` from `GET /api/v1/integration/jobs` older than three intervals — the scheduler has stopped |
| Alert evaluation | the same, for the `alert-evaluation` job |
| Disk | above 80 % on the data volume |

### Business monitoring

These are not infrastructure problems, but somebody needs to see them:

- No actual cane posted for a factory in the last 36 hours during a campaign
- Recovery outside the configured window for three consecutive days
- Any store forecast to exceed capacity within 14 days
- A material shortage with a suggested order date already in the past
- A plan submitted for review and untouched for more than 48 hours
- No weighbridge tickets received for a shift during a campaign — the gate
  terminal has probably lost its network, and the cane figures are silently
  behind rather than wrong

### Logs

One structured line per request:

```json
{"time":"…","level":"INFO","msg":"request","correlationId":"6f1c…",
 "method":"POST","path":"/api/v1/versions/…/generate","status":200,
 "durationMs":412,"user":"planner"}
```

Every problem document returned to a client carries the same `correlationId`, so
a user quoting a reference leads straight to the request and to the audit record
it produced.

---

## 5. Troubleshooting

### The application will not start

Read the message: configuration validation lists every problem at once.

| Message | Cause |
| --- | --- |
| `DATABASE_URL is required when STORE=postgres` | Not set |
| `AUTH_MODE must be oidc when APP_ENV=production` | The development sign-in was left enabled |
| `SEED_DEMO must be off when APP_ENV=production` | Demonstration data was left enabled |
| `AUTH_DEV_SECRET must be at least 16 characters` | Too short |
| `read openid configuration: …` | The issuer URL is wrong or unreachable |
| `FACTORY_TIMEZONE … is not a known IANA time zone` | A typo, or `tzdata` is missing from the image |

### `/readyz` fails but `/healthz` is fine

The process is up and the database is not reachable. Check credentials,
connectivity, and whether the connection pool is exhausted:

```sql
SELECT count(*), state FROM pg_stat_activity
WHERE datname = 'sugarplan' GROUP BY state;
```

### A user sees no seasons

Almost always the data scope, which fails closed by design. Check what they
have:

```sql
SELECT u.username, c.code AS company, f.code AS factory
FROM app_users u
LEFT JOIN data_scopes d ON d.user_id = u.id
LEFT JOIN companies c ON c.id = d.company_id
LEFT JOIN factories f ON f.id = d.factory_id
WHERE u.username = '<username>';
```

No rows means no access, which is correct behaviour and a missing grant.

### "The record was changed by somebody else" (412)

Two people edited the same record. The message names who. The client should
re-read and re-apply; this is working as intended, not a fault.

### A number looks wrong

1. Get the correlation id from the screen or the export footer.
2. Find the request in the logs.
3. Find the audit record: `GET /api/v1/audit?entityId=…`, which carries the
   before and after state.
4. Check the assumptions the version used — every rate is an effective-dated
   assumption, and a scenario copy may have changed one.
5. Reproduce the calculation from
   [06-calculation-catalogue.md](06-calculation-catalogue.md); each formula
   names its Go function and its test.

### A report or export is slow

Check the date range first: an unbounded range over ten years of history is the
usual cause. Then:

```sql
SELECT query, calls, mean_exec_time
FROM pg_stat_statements
WHERE query LIKE '%daily_%' ORDER BY mean_exec_time DESC LIMIT 10;
```

If the daily tables have grown past the point where the composite indexes are
enough, [05-data-model.md](05-data-model.md) sets out the partitioning trigger.

### Emergency: an unauthorised change

1. Find it: `GET /api/v1/audit?actor=…` or by entity.
2. The audit record carries the before state, so the previous values are known.
3. Correct it through the application, never with `UPDATE` — a direct update
   leaves no trail and is exactly what the audit exists to prevent.
4. For a released plan, reopen it with a reason, correct, and re-release. The
   whole sequence stays in the trail.

---

## 6. Routine tasks

| Task | Frequency |
| --- | --- |
| Verify last night's backup was written and is readable | Daily |
| Review alerts and error rates | Daily |
| Full restore rehearsal into a scratch database | Quarterly |
| Review user roles and data scopes | Quarterly |
| `govulncheck` and dependency review | Monthly |
| Review growth and the partitioning trigger point | Season end |
| Archive the season: close the plan versions, export the reports | Season end |
