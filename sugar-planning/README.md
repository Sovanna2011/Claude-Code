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

No database, no identity provider, no configuration:

```bash
cd backend
STORE=memory AUTH_MODE=dev AUTH_DEV_SECRET=local-development-secret \
SEED_DEMO=true HTTP_STATIC_DIR=../frontend/webapp \
go run ./cmd/server
```

Open <http://localhost:8080> and sign in as **Sokha Planner**. The Kampong Speu
2026–2027 season is already there: 2,300,000 t of cane over 137 days, generated
through exactly the same code path a planner uses.

Other accounts show the role model — `approver` can release a plan and `planner`
cannot; `auditor` can read the audit trail and nobody else can; `executive` can
read everything and change nothing.

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

On the reference figures it finds three real problems:

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

The store conformance suite runs against **both** the in-memory and the
PostgreSQL implementations, so the two cannot drift apart. The reference
reconciliation figures from the specification are assertions, not comments:
2,300,000 × 11.00 % = 253,000 t; capacities 45,000 + 65,000 = 110,000 t and
22,000 + 47,000 = 69,000 t; finished goods 106,700 + 133,400 + 2,000 =
242,100 t; and daily balance continuity for every date.

## Status

Phases 1 to 3 of the plan are delivered and tested: identity and authorisation,
master data, season and version management, the full planning chain, dashboards,
capacity forecasting, alerting, reports and exports. The schema for phases 4 and
5 is in place. Nothing in the delivered scope is a stub.

See [docs/10-implementation-plan.md](docs/10-implementation-plan.md) for what is
next and why.
