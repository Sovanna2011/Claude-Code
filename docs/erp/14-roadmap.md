# 14. Implementation Roadmap

Milestones are **executable** (§26.3): each ends with running code, passing
tests, and a demonstrable slice — not a folder of stubs. Every milestone closes
with the §26.17 report: completed components, files created, database changes,
tests performed, known limitations, next phase.

Sizing assumes a small team (2–3 backend, 1–2 frontend, 1 functional/QA). Durations
are indicative; sequence and dependencies are the binding part.

---

## Phase 1 — Solution Blueprint ✅ (this document set)

Architecture, module boundaries, enterprise structure, BP + synchronization,
universal journal, posting engine, currency, SE11, SE16N, customization, security,
project structure, schema strategy, T-code catalogue, processes, roadmap.

**Exit criterion: written approval of this blueprint.** No production code before
that (§26.2, project brief closing instruction).

---

## Phase 2 — Database Design *(≈ 3 weeks)*

| Milestone | Content | Done when |
|---|---|---|
| **M2.1 Foundation schema** | `org`, `cfg` core (company codes, fiscal/posting variants, currencies, rates, document types, number ranges, posting keys, field status), `sec` core, `audit` | DDL + EF configurations + migration applied; seeds 00–20 load; consistency-check queries pass |
| **M2.2 Master data schema** | `mdm` (BP + all facets), `fin.GLAccount*`, `co` masters, `fin.Asset*` masters | Seeds 30 load; FKs and unique constraints proven by negative tests |
| **M2.3 Transaction schema** | `fin.JournalEntryHeader/Line`, `OpenItem`, clearing, payments, asset transactions, `co.ControllingPosting`, `wf`, `intg`, partitioning, index set | Journal insert benchmark ≥ target; index plan reviewed; partition switch rehearsed |
| **M2.4 Dictionary seed** | All core tables described in `cfg.Dictionary*`; domains and data elements for every field | SE11 (Phase 4) can display every core table; label coverage test = 100 % |

**Deliverables** — complete table catalogue, ER diagrams, SQL Server DDL, keys,
constraints, indexes, EF Core configurations, migrations, seed data (§25 Phase 2).

---

## Phase 3 — Backend *(≈ 10 weeks)*

| Milestone | Content | Done when |
|---|---|---|
| **M3.1 Skeleton & cross-cutting** | Solution, layers, DI, MediatR pipeline, tenant context, Problem Details, Serilog, health checks, architecture tests | `dotnet test` green; architecture tests enforce the dependency rule |
| **M3.2 Organization & configuration** | Company codes, fiscal periods, number ranges, currencies + conversion service, document types, field status; SPRO APIs | Org rules R1–R12 tested; `ConcurrentNumberRangeTests` passes with N=200 parallel draws |
| **M3.3 Security** | Identity, JWT/OIDC, roles, permissions, authorization objects, org scope, T-code registry, SoD, maker-checker, audit writer | Authorization tests pass; a user cannot read another company code's data through any endpoint |
| **M3.4 Business Partner** | BP aggregate, roles, facets, relationships, **synchronization service**, consistency check, APIs | BP-sync tests: one BP with both roles; incomplete role blocked from posting; `BP_CHECK` finds seeded defects |
| **M3.5 Posting engine** ⭐ | The 19-step pipeline, simulation, reversal, idempotency, tax, currency conversion, CO derivation, validation/substitution rules | **The §23 mandatory-rule suite passes.** This is the gate for everything after it |
| **M3.6 General Ledger** | G/L master, journal entry (park/hold/submit/post), documents, clearing, recurring entries, accruals | Record-to-report (P1) end-to-end via API |
| **M3.7 AR & AP** | Customer/vendor invoices, credit memos, payments, partial/residual, down payments, dunning, payment run | P2 and P3 end-to-end; subledger ↔ G/L reconciliation test green |
| **M3.8 Asset Accounting** | Asset master, acquisition, transfer, retirement, depreciation keys and run, AuC settlement | P4 end-to-end; depreciation run idempotent across re-runs |
| **M3.9 Controlling** | Cost/profit centers, internal orders, budget & availability control, distribution/assessment, settlement, plan data | P5 and P6 end-to-end; FI→CO integration test: every cost-element posting carries a real CO object |
| **M3.10 Workflow** | Rules, instances, sequential/parallel/multi-level approval, delegation, substitution, escalation, notifications | Approval matrix tests; maker-checker cannot be bypassed by substitution |
| **M3.11 Dictionary & Table Browser** | SE11 services + validation + activation request pipeline; SE16N safe query builder + authorization + audit | SE16N security tests (row/field/mask) pass; activation cannot execute production DDL |
| **M3.12 Customization** | Custom table designer service, custom field service, dynamic EF model, change requests, migration generation | A `Z*` table is created, browsable, API-exposed, and posts through the engine; reserved-name test passes |
| **M3.13 Period close & FX** | Period open/close, foreign-currency valuation, year-end carry-forward, financial statement versions | Closed period rejects postings; FX valuation reverses correctly next period |
| **M3.14 Integration & jobs** | Outbox, webhooks, imports, bank statement staging, Hangfire jobs, SignalR | Payment run and depreciation run execute as jobs with full audit |

**Gate:** M3.5 must be complete and green before M3.6+ start. Everything
financial depends on the engine being right.

---

## Phase 4 — Frontend *(≈ 9 weeks, overlapping Phase 3 from M3.6)*

| Milestone | Content | Done when |
|---|---|---|
| **M4.1 Design system & shell** | Tokens, light/dark, primitives, DataGrid, AppShell, left nav, breadcrumbs, company-code/period/currency selectors, command box, favourites, recents, i18n scaffolding (en/km) | Accessibility audit (WCAG 2.1 AA) on the shell; keyboard navigation complete |
| **M4.2 Dictionary-driven forms** | `DictField`, F1 help, F4 value help, field status binding, validation surfacing from Problem Details | A form renders entirely from metadata, no hard-coded labels |
| **M4.3 Configuration (SPRO)** | Structure tree + generated maintenance views + consistency check | Full sample structure maintainable through the UI |
| **M4.4 Business Partner** | BP page with all tabs, role-driven visibility, relationships, change history | BP with both roles maintainable end-to-end |
| **M4.5 Journal entry** ⭐ | Header/lines/footer per §19, simulate, park, hold, submit, post, reverse, attachments, running totals + difference | A journal posts from the UI with live debit/credit/difference and simulation |
| **M4.6 AR / AP / Assets / CO screens** | Invoices, payments, payment run, asset master and runs, cost/profit centers, internal orders, allocations | P2–P6 drivable from the UI |
| **M4.7 Approval inbox & notifications** | Inbox, detail with document preview, approve/reject with reason, delegation, SignalR toasts, notification center | Approval round-trip under 3 clicks |
| **M4.8 SE11 / SE16N / designers** | Dictionary workbench, table browser with filters/layouts/variants/export, custom table & field designers | An end user builds a `Z*` table and browses it without developer help |
| **M4.9 Reports & dashboards** | Report launcher, all §21 reports, drill-down, saved layouts, Excel/PDF export, role dashboards | Every report reconciles to the journal (automated check) |
| **M4.10 Khmer localization** | Full km translation, Khmer numerals/date formats, font handling, RTL-safe layout checks | Language switch with no untranslated key in the core flows |

---

## Phase 5 — Testing & Deployment *(≈ 4 weeks, continuous from Phase 3)*

| Milestone | Content |
|---|---|
| **M5.1 Test completion** | Full §23 suite; coverage targets: domain ≥ 90 %, application ≥ 80 %; performance test on the journal at 10 M lines |
| **M5.2 Docker & environments** | Compose stack, per-environment configuration, secrets handling, health/readiness probes |
| **M5.3 CI/CD** | Build → test → security scan → migration validation → deploy dev/qa/prod, with the dictionary-activation pipeline wired in |
| **M5.4 Operations** | Backup & recovery guide, DR runbook, archiving, monitoring dashboards, runbook for a failed payment run |
| **M5.5 Documentation** | Deployment guide, administrator guide, user guide (en/km), API reference, process documentation |

---

## Future modules (post-Phase 5, no core redesign — [02 §2.5](02-module-boundaries.md))

| Order | Module | Reuses |
|---|---|---|
| 1 | **Purchasing** | BP vendor role, posting engine (GR/IR document types), commitments already modelled in CO |
| 2 | **Inventory** | Plant/location already in org; journal line already carries quantity + UoM |
| 3 | **Sales** | BP customer role, sales areas, credit management, pricing |
| 4 | **Payroll** | Existing HR module posts through `IPostingEngine`; employee = BP with Employee role |
| 5 | **Projects** | `AccountAssignmentType/Id` on the journal line is already reserved for WBS |
| 6 | **Production / Maintenance / QM** | Same reservation; order settlement already exists in CO |

---

## Risk register

| Risk | Impact | Mitigation |
|---|---|---|
| Posting-engine complexity underestimated | Everything downstream slips | M3.5 is an explicit gate with its own test suite; step-per-class design keeps it reviewable |
| Journal table performance at scale | Reports degrade | Partitioning + index plan fixed in Phase 2; benchmark at 10 M lines in M5.1, not at go-live |
| Multi-currency rounding disputes | Ledger fails to balance in group currency | Rounding-difference account handled explicitly at engine step 12; dedicated tests |
| Custom-field sprawl on the journal line | Table width and index bloat | Designer warns, change request requires justification, filtered index mandated |
| Dictionary activation blocking delivery | Frustration, workarounds | Dev environment allows direct apply for `Z*`; production stays gated |
| Khmer typography and formatting | Poor local usability | Localization is a milestone (M4.10), not a translation pass at the end |
| Scope pressure to start coding early | Rework | Phase-1 approval gate; module boundaries make parallel work safe once approved |
