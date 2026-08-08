# 12. The test system

```bash
./test-system.sh
```

Two tenants, the suites, and section 27 checked against a running instance.

This is not [the demonstration](11-demonstration-scenario.md). The
demonstration is one company at one factory, because that is the reference
scenario and its figures are meant to be quotable. The test system adds a second
mill, narrows every account to one of them, and then goes at the API with no
browser involved.

---

## 12.1 Why a second tenant

With one factory, "a planner at Kampong Speu cannot see another factory's plan"
can only be checked against a factory that does not exist. That proves a caller
scoped to nothing sees nothing, and says nothing at all about whether two real
tenants are kept apart — which is the most security-relevant thing this system
claims.

So `SEED_TENANTS=true` adds **Battambang Cane Millers Ltd.** (`BTB`), one
factory (`F2`), one milling train, two stores and a season of its own:

| | Kampong Speu | Battambang |
| --- | --- | --- |
| Cane target | 2,300,000 t | 640,000 t |
| Days | 137 | 96 |
| Recovery | 11.00 % | 10.40 % |
| Season starts | 2026-12-01 | 2026-12-15 |
| Finished goods | 242,100 t | 62,000 t |
| Stores | 110,000 raw + 69,000 finished | 18,000 raw + 9,000 finished |

The figures differ on purpose. A second tenant carrying the same numbers would
let a scope leak pass unnoticed: the test would be reading the right total from
the wrong factory.

It reuses the shared master data — products, units, packaging, reason codes —
because those are not factory-scoped, and duplicating them would be inventing a
difference the schema does not have.

**Accounts.** Under the test profile every demonstration account is narrowed to
`F1`, and two new ones belong to `F2`:

| | |
| --- | --- |
| `btb-planner` | Rin Planner (Battambang) |
| `btb-warehouse` | Sreypov Warehouse (Battambang) |

An account naming a factory that does not exist fails the start-up rather than
being widened to everything, because a typo that grants scope is worse than one
that fails.

`SEED_TENANTS` needs `SEED_DEMO`, and is refused outright when
`APP_ENV=production`.

---

## 12.2 The harness

`backend/cmd/acceptance` drives a running instance over HTTP, signs in as real
accounts, and checks the eleven acceptance criteria in section 27 of the
specification. It prints one line per check, a verdict per criterion, and exits
non-zero if anything failed.

```bash
go run ./cmd/acceptance -base http://localhost:8080 -v
```

It refuses to run anywhere the development login is off, which is what keeps it
away from a real plant: it posts stock and builds a season, and an instance
authenticating against the enterprise identity provider will not issue it a
token.

It is not a unit test, and the difference is the point. It finds what a unit
test cannot: a route registered but not wired, a permission enforced in the
service and forgotten in the handler, a figure that reconciles inside one
process and not across two requests, a query that works against the in-memory
store and fails against PostgreSQL.

**What it does to the data.** It builds its own thirty-day season, posts its own
documents, stages its own import and reverses what it posts. It reads everything
else. The reference figures are the same after a run as before one — criterion 6
checks exactly that.

Three of the eleven checks are skips, and deliberately so: whether the test
suites pass, whether there are security findings, and whether a backup restores
are not questions an HTTP client can answer. `test-system.sh` runs those around
the harness and fails the whole run if any of them fails. A skip is not a pass,
and the report says so.

The same rule applies to a step that *could not* run. A missing tool, or a
vulnerability database that cannot be reached from the network the run is on,
fails the run and says which it was — distinguished in the output from a scan
that found something, because the two need different things done about them. An
unverified claim reported as a pass is the one outcome worse than a red one.

> In the sandbox this was built in, `vuln.go.dev` is blocked by the network
> policy, so `govulncheck` cannot fetch its database and the run ends red on
> exactly that line. Everything else — the Go suite, the frontend tests, all
> eleven criteria and the restore drill — passes. CI has network access and runs
> the scan for real.

---

## 12.3 What one run covers

| Criterion | Checked by |
| --- | --- |
| 1. Plan, assumptions, generation, comparison | Building a season end to end and comparing two versions of it |
| 2. Approve, reject, release, history | The full workflow, plus a planner refused their own approval and a released baseline refusing a write |
| 3. Operators enter actuals | Cane, production, shipment, downtime and a laboratory sheet, plus a keeper refused a sample |
| 4. Derived figures | Cumulative + remaining = target; recovery recomputed from the two tonnages; capacity use recomputed from balance and usable capacity |
| 5. Postings atomic, reversible, auditable | A receipt, a two-line document with one bad line, a reversal, a second reversal refused, and both in the audit trail |
| 6. Dashboards reconcile | Downtime headline against the stoppage records, crushing against the daily rows, and the reference figures unmoved |
| 7. Alerts and thresholds | Evaluation fills role inboxes; the capacity threshold moved from 1 % to 99.9 % changes what is flagged |
| 8. Role and tenant scope | Both tenants, both directions, by list and by id, over reads and writes |
| 9. Import errors before commit | A file with two bad rows: the errors name rows 3 and 5, the commit is refused, the cancel leaves nothing behind |
| 10. Report exports | Every report in the caller's catalogue, in CSV, Excel and PDF, checked for content type, file name and magic bytes |
| 11. Tests, security, restore | Delegated to `test-system.sh`; the harness checks health, metrics, anonymous refusal and the problem-details format |

---

## 12.4 Against a real database

```bash
createdb sugarplan_test
./test-system.sh --postgres "postgres://localhost/sugarplan_test"
```

This adds the backup and restore drill: `pg_dump`, a `pg_restore --list` to
prove the dump is readable, a restore into a scratch database, and a comparison
of row counts across the planning, execution and audit halves. The scratch
database is dropped afterwards. It is the drill from
[the runbook](runbook.md), run rather than described.

It is also the configuration that matters. Two of the defects below only
appeared against PostgreSQL.

`--acceptance-only` skips the suites, for when they have already run. `--keep`
leaves the instance up so a failure can be poked at.

CI runs the whole thing against PostgreSQL 16 on every push.

---

## 12.5 What the first runs found

Every one of these is in the repository as a fix and a regression test. They are
listed because they are the argument for having a harness at all: none of them
was caught by the unit tests, and all of them were caught within two runs.

**The Reports page offered what it would refuse.** The packaging requirement
report needs `materials:read`, which only the planner holds. Every other role
holds `report:read`, saw the report on their Reports page, and got a 403 on
clicking it. The catalogue is now built per caller, and naming the report
directly is still refused — hiding it is a courtesy, the refusal is the control.

**A report asked for by season returned two different wrong answers.** Half the
catalogue reads daily rows for a plan version. Asked for a season with no
version named, they were handed an empty id: the in-memory store matched nothing
and returned an empty report, and PostgreSQL raised `invalid input syntax for
type uuid` and returned a 500. A season now resolves to its plan version the way
the dashboard does, and a blank id in a filter matches nothing rather than
everything.

**The second tenant could not generate a plan at all**, because the seed omitted
one required assumption. A missing figure, found in the first second of the
first run.

**The restore drill passed while doing nothing.** Its first version counted a
table that does not exist; `psql` printed an error, both counts came back empty,
and two empty strings compared equal. It now requires the counts to be numbers
and checks three tables.

---

## 12.6 What it does not cover

Said plainly, because a test report that implies more than it checked is worse
than no test report.

- **No browser.** Every check is an HTTP request. The SAPUI5 application is
  covered by unit tests over its pure modules and by structural checks on views,
  routes and translations in CI; there are no OPA5 journeys, and the SAPUI5 CDN
  is not reachable from this build environment to add them.
- **No load or soak testing.** One run, one caller, a few hundred requests.
- **Two tenants, not many.** Enough to prove isolation is enforced; not a
  multi-tenancy scale test.
- **The restore drill is a dump and reload**, not a point-in-time recovery from
  archived WAL. The runbook describes that; nothing here rehearses it.
