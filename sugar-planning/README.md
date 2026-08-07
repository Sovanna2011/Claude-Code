# Sugar Production Planning and Execution System

A production planning and daily execution system for a sugar factory: seasonal
planning through to shift-level recording, with the capacity, recovery and
shipment mathematics the spreadsheet it replaces could not do.

Built as an original application. It follows modern ERP and Fiori usability
conventions; it copies no SAP source, schema, screen or transaction.

```
Go 1.25  ·  PostgreSQL 16  ·  SAPUI5 1.120  ·  OpenAPI 3.1  ·  Docker
```

---

## Try it in one command

Go, and nothing else. No database to install, no identity provider to configure,
no build step:

```bash
./demo.sh
```

Open <http://localhost:8080> and sign in as **Sokha Planner**.

What is loaded is the Kampong Speu 2026–2027 season — 2,300,000 t of cane over
137 days — generated through exactly the code path a planner uses, plus a
fortnight of actuals and the factory life that goes with them:

| Screen | What is on it |
| --- | --- |
| Executive overview | Two real capacity warnings, and eight charts |
| Downtime | Seven stoppages; the Pareto is 62 % boiler |
| Production orders | Five, one closed 60 t short with the reason it needed |
| Warehouse and stock | Ten documents: receipts, a transfer, a stock correction and its reversal |
| Quality | Three samples: a pass, a failure that blocked 240 t, and its release |
| Costing | A saved run over the recorded fortnight |
| Reports | Fifteen, in CSV, Excel and PDF |

Every row of it is produced by driving the services, so nothing on any screen is
data written past the rules that guard it. It is deterministic — the same dates,
quantities and failures every time — and idempotent: restarting the container
does not double it.

```bash
./demo.sh --port 9000              # somewhere else
./demo.sh --postgres "$DSN"        # against a real database
./demo.sh --plan-only              # the plan without the factory life
```

And to check the system rather than show it:

```bash
./test-system.sh                   # two tenants, the suites, and section 27
./test-system.sh --postgres "$DSN" # plus the backup and restore drill
./load-test.sh --postgres "$DSN"   # section 25, at ten years of history
```

That boots a second company at a second factory, narrows every account to one of
them, and drives the eleven acceptance criteria over HTTP against the running
instance — printing pass or fail per criterion.
[What it covers, and what it found](docs/12-test-system.md), and
[what ten years of history does to the queries](docs/13-performance.md).

The accounts show the role model, and the difference between them is enforced by
the server rather than hidden by the screen: `approver` can release a plan and
`planner` cannot; `warehouse` posts stock and cannot touch the plan; `auditor`
can read the audit trail and nobody else can; `executive` reads everything and
changes nothing.

With Docker instead:

```bash
cd deploy && cp .env.example .env   # then edit POSTGRES_PASSWORD
docker compose up --build
```

---

## What it does

**Plans a season.** Enter the cane target, the recovery assumption, the product
mix and the capacities; the system generates 137 days of cane, raw sugar,
finished goods, stock and shipment rows, and tells you what is wrong with the
plan before anybody releases it.

On the reference figures it finds real problems, before anybody releases the
plan:

- the finished goods mix needs **254,205 t** of raw sugar but the plan produces
  **253,000 t** — a 1,205 t shortfall at the stated 1.05 remelt factor;
- Refined/White Warehouse 1 **fills on 1 January 2027**, six weeks into a
  four-and-a-half month campaign;
- keeping it inside capacity needs **844 t/day** of shipment against the
  **500 t/day** planned.

**Runs the day.** Production orders come from the released plan; shifts confirm
against them and the yield is receipted into a store in the same transaction.
Stock moves only through documents — receipt, issue, transfer, adjustment, count,
hold, release, shipment — and a mistake is corrected by a reversal that points
back at the original, never by an edit. The laboratory judges samples against
effective-dated limits and blocks the material that fails. Actuals live in their
own container and can never overwrite an approved plan.

**Refuses what should be refused, and says why.** A posting that would drive a
balance below zero, ship quality-held sugar, or overfill a store is rejected with
the offending line named and the figures quoted. Breaking one of those rules is
possible, but it takes a second permission and it lands on the audit record.

**Costs it.** Every cost element has a driver — dollars per ton of cane, per hour
run, per calendar day — so the difference from plan splits into the part caused
by paying a different price and the part caused by using a different quantity,
and the two always add back up to the total. Multi-currency, effective-dated
rates, and a saved run keeps the rates it used so the figure is reproducible
after they have moved on.

**Talks to the other systems, without being held hostage by them.** Every change
worth publishing is written to an outbox in the same transaction as the change
itself, so a confirmation that rolls back cannot leave a message saying it
happened, and an ERP that is down cannot cause a confirmation to be refused. A
dispatcher retries on a widening backoff; an event that has given up after
twenty-five attempts waits for a person rather than disappearing. Inbound, the
weighbridge and the laboratory system feed through the ordinary services, so an
interface cannot reach a verdict a person could not.

**Takes the spreadsheet in, without taking its mistakes.** A site defines a
mapping template once — which column holds which field, whether the numbers are
`1,234.56` or `1.234,56`, how the dates are written — then uploads a `.csv` or
`.xlsx` and reads a preview: what would be written, what would be replaced, and
every problem attached to the line of the file it is on. Nothing reaches the plan
until somebody commits, and the commit goes through the same service a planner
types into.

**Tells somebody.** The alerts are evaluated on a timer, not on a page load: a
store that fills on 1 January should reach the shipment planner in November, not
the next time anybody happens to open the dashboard. Notifications are addressed
to a role at a factory rather than to a named person, so an alert never belongs
to somebody who has left.

**Answers the capacity question.** When does this store fill? What shipment rate
prevents it? What does a lower recovery do to the season? Each is a calculation,
not a guess.

**Keeps the record.** Every change is audited with actor, time, before and after
state, reason and correlation id. Approvals are separated from planning, released
periods are locked, and the trail is append-only.

---

## Documentation

| | |
| --- | --- |
| [1. Assumptions and open questions](docs/01-assumptions-and-questions.md) | **Read first.** What was assumed, what was found in the reference figures, what still needs a business answer |
| [2. Business process map](docs/02-business-process-map.md) | Season plan to daily execution to reporting |
| [3. Module and authorisation matrix](docs/03-module-and-authorization-matrix.md) | Modules, permissions, roles, data scope |
| [4. Architecture](docs/04-architecture.md) | Layers, repository structure, deployment, failure behaviour |
| [5. Data model](docs/05-data-model.md) | ERD, data dictionary, constraints, partitioning |
| [6. Calculation catalogue](docs/06-calculation-catalogue.md) | Every formula, unit, rounding rule and worked example |
| [7. API catalogue](docs/07-api-catalogue.md) | Endpoints, conventions, worked requests |
| [8. UI sitemap](docs/08-ui-sitemap.md) | Pages, wireframes, UX rules |
| [9. Workflow diagrams](docs/09-workflow-diagrams.md) | State machines and transitions |
| [10. Implementation plan](docs/10-implementation-plan.md) | Phases, status, acceptance criteria |
| [11. Demonstration scenario](docs/11-demonstration-scenario.md) | What `./demo.sh` loads, and a walkthrough of it |
| [12. Test system](docs/12-test-system.md) | The second tenant, the acceptance harness, and what running it found |
| [13. Performance](docs/13-performance.md) | The section 25 targets, measured at ten years of history, and the three things that fixed |
| [Runbook](docs/runbook.md) | Environment variables, deployment, backup and restore, troubleshooting |

The API contract is `backend/internal/api/openapi.yaml`, served live at
`/api/v1/openapi.yaml`.

---

## Layout

```
sugar-planning/
├── backend/          Go: domain, store, services, API, reports
├── frontend/         SAPUI5 application
├── database/         how to run the migrations
├── deploy/           Dockerfile, compose, environment template
└── docs/             the documentation above
```

## Tests

```bash
cd backend
go test ./...                                              # unit and service
TEST_DATABASE_URL=postgres://… go test ./...               # plus PostgreSQL integration
```

```bash
./test-system.sh                                           # section 27, over HTTP
```

The store conformance suite runs against **both** the in-memory and the
PostgreSQL implementations, so the two cannot drift apart. Above them,
`test-system.sh` boots a real instance with two tenants in it and checks the
acceptance criteria over the wire — which is where a route that is registered
but not wired, or a query that works in memory and fails on PostgreSQL, actually
shows up. The reference
reconciliation figures from the specification are assertions, not comments:
2,300,000 × 11.00 % = 253,000 t; capacities 45,000 + 65,000 = 110,000 t and
22,000 + 47,000 = 69,000 t; finished goods 106,700 + 133,400 + 2,000 =
242,100 t; and daily balance continuity for every date.

## Status

All five phases are delivered and tested: identity and authorisation, master
data, seasons and versions, the full planning chain, dashboards, capacity
forecasting, alerting, reports and exports; then execution — postings, orders,
confirmations, quality, maintenance — and finally costing, the transactional
outbox with its adapters, and the background scheduler.

Nothing in the delivered scope is a stub. Verified against PostgreSQL 16 and in
a browser, not only in unit tests.

Two things are deliberately not here. A **pre-canned mapping for the reference
workbook**: the import framework is built and works, but that particular file was
never supplied, so its sheet names, header rows and units are unknown — defining
the template once it exists is data entry, not a code change. And **the findings
in the reference figures above**, which await a business answer rather than a
code change; they are listed at the top of
[docs/01-assumptions-and-questions.md](docs/01-assumptions-and-questions.md).

See [docs/10-implementation-plan.md](docs/10-implementation-plan.md) for what
each phase delivered and what is still open.
