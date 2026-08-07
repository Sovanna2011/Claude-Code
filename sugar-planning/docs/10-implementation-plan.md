# 10. Phased implementation plan and acceptance criteria

Five phases as the specification sets out. **Phases 1 to 3 are delivered.**

---

## Phase 1 — Foundation ✅ delivered

Identity, organisation, master data, season and version management.

**Delivered**

- Modular monolith in Go with domain, store, service and transport layers
- OIDC token verification (discovery, JWKS, RS256, configurable roles claim) and
  a development mode that is refused in production
- 14 roles, 20 permissions, company and factory data scoping that fails closed
- 14 master data entities with effective dating, soft deactivation, optimistic
  concurrency and a uniform API
- Seasons and plan versions with the full state machine
- PostgreSQL schema for every table group, migrations reversible and verified
- Append-only audit trail with before and after state and a correlation id

**Acceptance criteria — met**

| Criterion | Evidence |
| --- | --- |
| A user signs in and sees only their factories | `TestDataScopeIsEnforcedOverHTTP`, `TestDataScopeHidesOtherFactories` |
| Master data supports code, dates, active status, audit and version | store conformance suite, both implementations |
| A stale update is rejected with the other editor named | `TestConcurrency`, `TestETagAndIfMatch` |
| Migrations apply and reverse cleanly | run against PostgreSQL 16.13; `MigrateUp`/`MigrateDown` |
| Every change is audited | `TestAudit`, `TestWorkflowOverHTTP` |

---

## Phase 2 — Daily planning ✅ delivered

Cane, raw sugar, finished goods, shipment and storage planning.

**Delivered**

- The calculation catalogue, C1–C34, with table-driven tests
  (C35–C43 followed with costing and the packaging bill of materials)
- Plan generation from assumptions and product mix, in one transaction
- Daily cane, production, stock ledger and shipment rows, plan and actual
- Bulk upsert with row-level validation, all-or-nothing by default
- Stock ledger with balances stored and recalculated forward on any edit
- Scenario copy with assumption overrides; version comparison
- The reference scenario as seed data, generated through the same code path a
  planner uses

**Acceptance criteria — met**

| Criterion | Evidence |
| --- | --- |
| A planner creates a season, enters assumptions and generates daily targets | `TestSeedProducesTheReferenceScenario` |
| 2,300,000 × 11.00 % = 253,000 t | `TestReferenceScenarioReconciliation` |
| Daily balance continuity holds for every date | `TestGeneratedLedgerBalancesAreStoredAndContinuous` |
| Season splits sum back to the target exactly | `TestAllocateEvenlyPreservesTotal`, `TestScaleSeriesPreservesTheRoundedTotal` |
| Actuals cannot overwrite an approved plan | `TestActualsCannotBePostedToAPlanVersion` |
| A failed posting leaves nothing behind | `TestTransactionRollback` |
| Versions can be compared | `TestWhatIfScenarioChangesTheOutcome` |

---

## Phase 3 — Dashboards, approval, capacity, alerts ✅ delivered

**Delivered**

- Executive dashboard: cane, recovery, product, storage, shipment and downtime
  KPIs with trend and rolling average
- Capacity forecasting: first warning date, first full date, and the required
  daily shipment rate
- Alerting on capacity, schedule, recovery, supply and material shortage
- Approval workflow with locking, separation of duties and mandatory reasons
- Nine reports exported to CSV, Excel and PDF, all carrying full metadata
- The SAPUI5 application, eleven pages, verified in a browser

**Acceptance criteria — met**

| Criterion | Evidence |
| --- | --- |
| An approver approves and releases a locked baseline with full history | `TestPlanWorkflowEndToEnd`, `TestWorkflowOverHTTP` |
| Dashboards reconcile to the transaction data | `TestDashboardReconcilesWithTheSeededPlan` |
| Capacity risk dates and required shipment rates are calculated | same, plus `TestRequiredDailyShipment` |
| Alerts fire on real conditions | `TestDashboardWarnsWhenCrushingFallsBehind` |
| Reports export correctly to Excel, PDF and CSV | `TestExportFormats` — zip structure, numeric cells, PDF catalogue and xref |
| Role restrictions are enforced by the backend | `TestPermissionsAreEnforcedPerEndpoint` |

**Not delivered in phase 3: Excel migration.** The specification asks for an
Excel import with preview, validation, duplicate detection and controlled
commit. The `import_jobs` table exists, but the importer is not written, because
the workbook it must read was never provided (see
[01-assumptions-and-questions.md](01-assumptions-and-questions.md)). Writing a
column mapping against a file nobody has seen would be guesswork. It moves to
phase 4, where it is the first item.

---

## Phase 4 — Execution, quality, maintenance ✅ delivered

**Delivered**

- One posting door. Every stock movement in the application - hand-entered
  corrections, production confirmations, quality holds, releases and reversals -
  goes through `Execution.postChecked`, which reads the affected positions, asks
  the domain whether the movement is allowed, and writes it. There is exactly
  one place the rules can be applied and exactly one place they can be forgotten.
- Inventory documents: receipt, issue, transfer, adjustment, count, hold,
  release, shipment and reversal. Quantities are entered unsigned and given
  their sign by the document type; a transfer becomes the pair of signed lines
  that move both balances, so the halves cannot be posted apart.
- Production orders: created by hand or from a released plan, released,
  confirmed, reversed, technically closed. A day and product already covered is
  skipped rather than duplicated, so the run repeats safely as the season
  advances.
- Quality: a parameter catalogue, effective-dated specifications, samples judged
  against the limits in force on the sample's business date, and holds that move
  the held quantity on the balance so blocked sugar cannot be shipped.
- Maintenance windows that reach the plan generator on their own. An approved,
  factory-wide window removes crushing days and the campaign is extended rather
  than shortened.
- Four SAPUI5 pages - Stock, Production orders, Quality, Maintenance - walked
  through in a browser end to end.

**Acceptance criteria — met**

| Criterion | Evidence |
| --- | --- |
| An order is created from a released plan, confirmed, and posts a receipt atomically | `TestOrdersAreCreatedFromTheReleasedPlanAndNotDuplicated`, `TestConfirmingReceiptsTheYieldAndAdvancesTheOrder` |
| A confirmation is reversed by a document; nothing is deleted | `TestReversingAConfirmationUndoesBothTheOrderAndTheStock` |
| An order closes only after reconciliation or with an authorised variance reason | `TestClosingAnOrderOutsideToleranceNeedsAnAuthorisedReason` |
| Quality-held stock cannot be shipped or consumed | `TestAFailedSampleBlocksTheStockItCovers` |
| An approved maintenance window reduces planned capacity without being re-entered | `TestApprovedMaintenanceLengthensTheCampaign`, `TestALineOutageDoesNotStopTheFactory` |
| A refused posting writes nothing at all | `TestAnIssueBeyondTheBalanceIsRefusedWithTheFigures`, `InventoryRollback` in the store conformance suite |
| A retried posting does not move the balance twice | `TestARetriedPostingReplaysInsteadOfPostingTwice` |
| Both store implementations behave identically | the conformance suite, run against in-memory and PostgreSQL |

**Defects the phase found and fixed**

Four in code that already existed, all found by writing the tests rather than by
reading the code:

- `ListDocuments` built its aliased select list by text substitution, which
  turned `factory_id` into `factory_d.id` and made the query unrunnable.
- `PostDocument` moved balances with an upsert carrying the deltas. PostgreSQL
  checks table constraints against the tuple an INSERT proposes before it
  discovers the conflict, so a hold of 100 t against a stock of 380 t arrived as
  "quantity 0, hold 100" and tripped the rule that nothing may be held that is
  not there.
- Row stamps were taken at nanosecond resolution and stored at the microsecond
  resolution of a `timestamptz`, so the creation stamp an insert returned never
  equalled the one a later update returned.
- The `stock_balances` check forbade a negative balance outright, contradicting
  the posting rules, where a negative balance is refused by default but permitted
  for a caller holding the override. Migration 0006 relaxes it.

Three more in the API and the demonstration data:

- Problem documents were served as `application/json`; RFC 9457 gives them their
  own media type.
- The specification said quantities travel as JSON numbers. They travel as
  strings, and should.
- The sign-in page kept its own copy of the demonstration accounts, and the copy
  had drifted: the quality user was configured on the server and missing from the
  page, so the laboratory role could not be demonstrated at all. The list now
  comes from the server.

**Not delivered in phase 4: Excel migration.** It carries forward unchanged, and
for the same reason: the workbook it must read was never provided. See
[01-assumptions-and-questions.md](01-assumptions-and-questions.md).

**Also outstanding from the phase-4 scope**

| Item | Estimate |
| --- | --- |
| Excel import: mapping template, staging, preview, row-level errors, controlled commit | 3 weeks — needs the workbook |
| Certificate of analysis as a printed document | 1 week |
| Component consumption against a bill of materials rather than as entered | 1 week |
| Background jobs: forecast recalculation, alert notification, scheduled exports | 1 week |

---

## Phase 5 — Costing and integration ✅ delivered

**Delivered**

- Costing: cost elements with drivers, effective-dated standard and actual
  rates, multi-currency, cost per ton, and a variance split into its rate and
  usage halves that always reconciles (C35–C42)
- A transactional outbox: eight topics, written in the same transaction as the
  change, delivered at least once with a widening backoff and a retry ceiling
  that puts an event in front of a person rather than discarding it
- Inbound adapters: weighbridge gate tickets and laboratory results, both
  through the ordinary services so an interface cannot reach a verdict a person
  could not
- An in-process job scheduler behind a database lease, so a pair of instances
  share the recurring work rather than duplicating it
- Certificate of analysis, printed from a completed sample with the limits it
  was judged against
- Packaging bill of materials, driving both requirement planning and the
  components a confirmation consumes (C43)
- The controlled import of section 22: a mapping template, a staging area, a
  preview with row-level errors and a downloadable error file, and a commit
  through the ordinary planning service
- The rest of the scheduled jobs: alert evaluation into role-addressed inboxes,
  alongside the outbox dispatcher

**Acceptance criteria — met**

- An event whose posting rolled back is never published; asserted by
  `TestAFailedPostingPublishesNothing`
- A delivery that fails records its error and is retried later, not immediately;
  asserted by `TestADeliveryThatFailsIsRetriedWithItsErrorRecorded`
- Two schedulers never run the same job at the same tick; asserted by
  `TestOnlyOneInstanceRunsAJob`
- A gate terminal that resends a batch does not weigh the same lorries twice
- A certificate keeps the limits in force when the sample was completed, even
  after the specification is tightened

---

## Phase 6 — closing the specification sweep ✅ delivered

Everything above was built against the specification section by section. This
phase came from reading it again, end to end, against what had actually been
delivered — and finding seven things that had been counted as done and were not.
They are recorded here rather than quietly fixed, because "we thought it was
finished" is the interesting part.

| Found | What was actually there |
| --- | --- |
| Six of section 14's fifteen reports | Recovery and mass balance, remelt and refining, packing by package, order variance, downtime, and quality results had no builder at all. The data for every one of them was already stored |
| Five of section 13's eight visuals | Only three chart renderers existed. `DowntimeKPIs` aggregated to three scalars and threw the reason code away, so the Pareto had nothing to rank |
| "Every KPI must drill down" | The dashboard had one `navTo`, to the launchpad |
| Downtime, a section 15 application area | The API had recorded stoppages since the beginning and nothing displayed them |
| Variant management, saved views, personalization | Absent; `saved_views` is migration 0011 |
| Sorting and grouping | No `Sorter` anywhere in the application |
| Metrics | The runbook prescribed alerting on p95 latency and on a stalled job, with nothing that could measure either |

Two absences turned out to be decisions that had never been written down, and
now are: there is no `$select`, and CSRF does not apply to a bearer-token API
that sets no cookie. Both are argued in the documents rather than left to look
like oversights.

The frontend also had no tests at all: `npm test` printed a message telling you
to open a QUnit page that did not exist. The chart renderers and the formatters
are pure functions, so they now run under `node --test` — eighteen of them,
asserting the decisions rather than the pixels.

**Acceptance criteria — met**

- Every report in section 14 exists and its arithmetic is asserted against
  figures somebody could redo on paper
- Every visual in section 13 is drawn, and every KPI links to the daily rows it
  is a sum of
- A saved view belongs to its owner; somebody else's is `404`, not `403`
- `GET /metrics` labels by route pattern, never by path

---

## Phase 7 — the test system ✅ delivered

A demonstration shows the system. This checks it.

Two things were missing. There was no harness that drove a **running** instance:
every test in the repository ran the handler in-process, which cannot find a
route that is registered and not wired, or a query that works against the
in-memory store and fails against PostgreSQL. And the seed created one company
at one factory, so the most security-relevant claim the system makes — that two
tenants are kept apart — could only be checked against a factory that did not
exist, which proves a caller scoped to nothing sees nothing and nothing else.

So: `SEED_TENANTS=true` adds Battambang Cane Millers at a second factory with a
season and figures of its own, and narrows every development account to one mill
or the other. `backend/cmd/acceptance` drives a running instance over HTTP
against all eleven of section 27's criteria and exits non-zero on failure.
`./test-system.sh` boots the instance, runs the suites and the vulnerability
check around it, and — against PostgreSQL — dumps, restores into a scratch
database and compares the row counts. CI runs the whole thing on every push.

[12-test-system.md](12-test-system.md) has the coverage table and the honest
list of what it does *not* check.

**What the first two runs found**, none of it caught by the unit tests:

| Found | Fix |
| --- | --- |
| The Reports page offered the packaging requirement report to every role and answered 403 to all but the planner — it needs `materials:read` | The catalogue is built per caller, and naming the report directly is still refused |
| A report asked for by season with no version named returned an empty table in memory and a **500 on an invalid uuid** against PostgreSQL | A season resolves to its plan version the way the dashboard does; a blank id in a filter now matches nothing rather than raising |
| The second tenant could not generate a plan at all — the seed omitted a required assumption | Added, in the first second of the first run |
| The restore drill passed while doing nothing: it counted a table that does not exist, both counts came back empty, and two empty strings compared equal | The counts must be numbers, across three tables |

**Acceptance criteria — met**

- All eleven criteria pass against PostgreSQL 16 and against the in-memory store
- Neither tenant can read, or write into, the other — by list, by id, over plans,
  dashboards, stock and postings
- The harness refuses to run where the development login is off
- Running it twice against the same instance passes twice
- The reference figures are the same after a run as before one

---

## Phase 8 — performance at ten years of history ✅ delivered

Section 25 sets two numbers - list APIs p95 under 500 ms, dashboards under 2 s -
at "at least 10 years of daily history and high-volume transaction and audit
data". Both had been designed for and neither had been measured, which makes
them wishes rather than targets.

`backend/cmd/loadtest` generates the volume (616,511 document items, 205,510
documents, 205,550 audit events in about thirty seconds, by `COPY` rather than
through the services - this fixture is about volume, not correctness, and says
so) and then measures every endpoint over HTTP twice: alone, and under
concurrent load. Alone is the query; under load is the machine. A report giving
only the second would blame an index for a small server.

[13-performance.md](13-performance.md) has the table and the caveats.

**What the first run found:**

| Found | Fix |
| --- | --- |
| The exact `count(*)` beside a page of documents cost 84 ms against 1.5 ms for the page itself, and grew with the table forever. Filtered to one warehouse under load: **p95 829 ms against a 500 ms target** | The count is bounded by a `LIMIT`, so the planner stops early: 21 ms, and 0.1 ms when nothing matches. Past `store.CountLimit` the response sets `countCapped`, so a screen says "10,000+" rather than a floor pretending to be a total |
| A page of a hundred documents made a hundred round trips for their lines | One `WHERE document_id = ANY($1)` for the whole page |
| The warehouse filter's `EXISTS` could not use an index-only scan | Migration 0012 adds `(warehouse_id, document_id)`; heap fetches 153,450 → 0 |
| **That index made the isolated query faster and the concurrent one worse** - 84→68 ms alone, 829→1236 ms under load, because the index-only scan encouraged a parallel plan and eight callers on four cores cannot afford one | Kept the index, fixed the real cost (the count), and made the harness measure both - an `EXPLAIN ANALYZE` on a quiet database decides nothing |

**Acceptance criteria — met**

- Every endpoint is inside its target at ten seasons of history, alone and at
  eight concurrent callers
- The one with no margin is named rather than averaged away: documents filtered
  to one warehouse, 494 ms against 500 ms
- Both stores report the same bounded count, held there by the conformance suite
- All 61 acceptance checks still pass after the store change

---

## Still open

| Item | Estimate |
| --- | --- |
| A pre-canned mapping for the reference workbook — the framework is built; only the template for that one file is missing, because the file was never supplied | an hour of data entry once the file exists |
| Advanced forecasting: seasonality, weather, cane maturity | 4 weeks |
| OPA5 end-to-end journeys in a browser | 1 week |
| Operational hardening: partitioning, read replicas, cache tuning | 2 weeks. Load testing at ten years of data is **done** - see [13-performance.md](13-performance.md); partitioning is measured as not yet justified |
| An OpenTelemetry exporter, if a deployment has a collector to send to | 2 days |

---

## The demonstration system

`./demo.sh` — Go and nothing else — loads the reference scenario and plays a
fortnight of factory life through it, so every screen has something on it rather
than only the planning half. It is built by driving the services, never the
store: a demonstration assembled by writing rows would skip the validation, the
audit trail and the permission checks, and the first figure anybody questioned
would turn out not to reconcile.

[11-demonstration-scenario.md](11-demonstration-scenario.md) has the walkthrough
and the figures to check it against.

Building it found two defects that the tests had not, which is the argument for
running a thing rather than only testing it:

- The downtime Pareto's cumulative share reached **100.001 %**. Each reason's
  share was rounded to three places and then added up; 62.069 + 24.138 + 6.897 +
  6.897 overshoots. It is now calculated from the running hours.
- The scenario reported eight inventory documents and the ledger held **ten**:
  placing a quality hold and releasing it are each their own document, which is
  right and which the hand-kept tally did not know. The count is read back from
  the ledger now.

And one it found about itself: the first version was not idempotent, so a
container restart tripled the stoppages and the orders.

---

## Testing status

| Suite | State |
| --- | --- |
| Domain calculations, table-driven, including zero and rounding edges | ✅ passing |
| Reference workbook reconciliation | ✅ passing |
| Plan generator, including ledger continuity and warnings | ✅ passing |
| Workflow transitions and locking | ✅ passing |
| Store conformance, in-memory **and** PostgreSQL 16.13 | ✅ passing |
| Transaction rollback | ✅ passing |
| Service layer: authorisation, data scope, bulk validation, dashboard | ✅ passing |
| API: authentication, RBAC, problem details, ETag, idempotency, exports | ✅ passing |
| OpenAPI paths all routed | ✅ passing |
| Browser walkthrough of every page | ✅ verified manually in Chromium, up to phase 5. The phase-6 screens — downtime, the new charts, the variant bar — have **not** been through a browser: the SAPUI5 runtime is loaded from `ui5.sap.com`, which the environment they were built in blocks. They were verified by unit-testing the pure modules, by checking every view parses and every i18n key and route resolves, and by exercising the endpoints behind them against a running server. That is not the same as looking at the page, and is why the OPA5 row below matters |
| Costing, including the variance reconciliation | ✅ passing |
| Outbox: backoff, exhaustion, rollback, lease, both stores | ✅ passing |
| Weighbridge and laboratory adapters, including their authorisation | ✅ passing |
| Certificate of analysis and bill-of-materials consumption | ✅ passing |
| Spreadsheet reading: separators, serial dates, blank cells, .xlsx | ✅ passing |
| Import staging, validation, duplicate detection, partial commit | ✅ passing |
| Alert evaluation, deduplication, inbox addressing | ✅ passing |
| SAPUI5 formatter and chart unit tests (`npm test`, 18) | ✅ passing |
| Saved views: ownership, sharing, defaults, both stores | ✅ passing |
| Metrics: route labelling, no business data in the exposition | ✅ passing |
| The demonstration scenario: contents, idempotency, reconciliation with the dashboard | ✅ passing |
| The second tenant: its own plan, isolation both ways, idempotency | ✅ passing |
| Section 27 against a running instance (`./test-system.sh`, 61 checks) | ✅ passing, in memory and against PostgreSQL 16 |
| Backup and restore rehearsal | ✅ run by `./test-system.sh --postgres`: dump, readability check, restore into a scratch database, row counts compared |
| OPA5 end-to-end journeys in a browser | ⏳ still open |
| Load and performance at ten years of data (`./load-test.sh`, 616k document items) | ✅ passing, with `documents, one warehouse` at 494 ms against a 500 ms target under load - no margin |

---

## Definition of done, per phase

1. Runnable code, no placeholder methods and no unexplained TODOs
2. Migrations, up and down, applied against a real PostgreSQL
3. Automated tests covering the calculations, the permissions and the failure paths
4. Sample data exercising the same code path production uses
5. OpenAPI updated, with the routing test proving it
6. Screens where the phase includes them
7. Documentation updated in this set
