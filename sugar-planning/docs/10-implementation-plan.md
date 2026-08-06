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

## Phase 4 — Execution, quality, downtime, materials ⏳ next

**Already in place:** the complete schema for orders, confirmations, inventory
documents, quality and maintenance; the downtime recording path and its
lost-tonnage impact on the dashboard; packaging material requirement planning.

**To build**

| Item | Estimate |
| --- | --- |
| Excel import: mapping template, staging, preview, row-level errors, controlled commit, idempotency | 3 weeks — needs the workbook |
| Production orders: create from a released plan, release, confirm, reverse, close | 3 weeks |
| Inventory documents: receipt, issue, transfer, adjustment, count, hold, release, reversal | 3 weeks |
| Quality: samples, results against effective-dated specs, holds blocking shipment, certificate of analysis | 2 weeks |
| Maintenance windows feeding the generator's non-working days automatically | 1 week |
| Background jobs: forecast recalculation, alert notification, scheduled exports | 1 week |

**Acceptance criteria**

- A production order is created from a released plan, confirmed, and posts a
  warehouse receipt and component consumption atomically
- A confirmation is reversed by a document; nothing is deleted
- An order closes only after reconciliation or with an authorised variance reason
- Quality-held stock cannot be shipped or consumed
- An import identifies every error before commit and preserves source-row
  traceability
- An approved maintenance window reduces planned capacity without being
  re-entered

---

## Phase 5 — Costing, integration, optimisation 📋 planned

| Item | Estimate |
| --- | --- |
| Costing: standard and actual rates, cost per ton, variance by component, multi-currency | 4 weeks |
| Weighbridge and LIMS adapters | 3 weeks |
| MES / historian and ERP interfaces over the transactional outbox | 3 weeks |
| Advanced forecasting: seasonality, weather, cane maturity | 4 weeks |
| Operational hardening: partitioning, read replicas, cache tuning, load testing at ten years of data | 2 weeks |

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
| Browser walkthrough of every page | ✅ verified manually in Chromium |
| SAPUI5 unit and OPA5 tests | ⏳ phase 4 |
| Load and performance at ten years of data | ⏳ phase 5 |
| Backup and restore rehearsal | ⏳ deployment task; runbook written |

---

## Definition of done, per phase

1. Runnable code, no placeholder methods and no unexplained TODOs
2. Migrations, up and down, applied against a real PostgreSQL
3. Automated tests covering the calculations, the permissions and the failure paths
4. Sample data exercising the same code path production uses
5. OpenAPI updated, with the routing test proving it
6. Screens where the phase includes them
7. Documentation updated in this set
