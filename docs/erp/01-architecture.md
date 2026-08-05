# 1. Proposed Architecture

## 1.1 Architectural style

**Modular monolith, Clean Architecture inside each module, DDD tactical patterns
in the financial core, CQRS where it earns its place.**

```
┌───────────────────────────────────────────────────────────────────────────────┐
│  CLIENTS                                                                      │
│  React 19 + TypeScript SPA  ·  Mobile browser  ·  External systems  ·  Jobs   │
└───────────────┬───────────────────────────────────────────────────────────────┘
                │ HTTPS · JSON · OIDC/JWT · Problem Details (RFC 7807)
┌───────────────▼───────────────────────────────────────────────────────────────┐
│  WEB API  (Erp.Api — ASP.NET Core 10)                                         │
│  Versioned controllers /api/v1/*  ·  Swagger/OpenAPI  ·  SignalR hubs          │
│  Middleware: correlation → tenant resolution → auth → problem details → audit  │
└───────────────┬───────────────────────────────────────────────────────────────┘
                │ MediatR commands / queries (in-process)
┌───────────────▼───────────────────────────────────────────────────────────────┐
│  APPLICATION LAYER  (per module: Erp.Modules.<Name>.Application)               │
│  Commands · Queries · Handlers · FluentValidation · Authorization behaviours   │
│  Integration-event publication · Unit-of-work orchestration                    │
└───────────────┬───────────────────────────────────────────────────────────────┘
                │ domain calls (no infrastructure types)
┌───────────────▼───────────────────────────────────────────────────────────────┐
│  DOMAIN LAYER  (per module: Erp.Modules.<Name>.Domain)                         │
│  Aggregates · Entities · Value objects · Domain services · Domain events       │
│  Invariants: debits=credits, period open, immutability of posted documents     │
│  ── Erp.Posting.Domain: the single central posting engine ──                   │
└───────────────┬───────────────────────────────────────────────────────────────┘
                │ interfaces only (repositories, clocks, number ranges, rates)
┌───────────────▼───────────────────────────────────────────────────────────────┐
│  INFRASTRUCTURE  (Erp.Infrastructure + per-module Infrastructure)              │
│  EF Core 10 · SQL Server 2025 · Dapper (read models) · Serilog · Hangfire      │
│  Identity · Caching · Outbox · File/attachment store · Bank & payroll adapters │
└───────────────┬───────────────────────────────────────────────────────────────┘
                │ T-SQL
┌───────────────▼───────────────────────────────────────────────────────────────┐
│  SQL SERVER 2025 — schemas: org cfg mdm fin co wf sec rpt intg audit ext       │
└───────────────────────────────────────────────────────────────────────────────┘
```

### Why a modular monolith and not microservices

A financial posting must atomically create: a journal header, N journal lines,
open items, asset transactions, controlling postings, a workflow instance, an
audit record, and an outbox message (§15, steps 13–19). Across services this
becomes a saga with compensating transactions — which for accounting means
*reversal documents for internal plumbing failures*. Unacceptable in a ledger.

So the rule is: **one database transaction owns one posting**. Modules are
compile-time separated (own projects, own schemas, no cross-module entity
references), which preserves the DDD boundary and keeps a future extraction
cheap for the modules that could tolerate eventual consistency (Reporting,
Integration, Workflow notifications). The financial core stays together by
design, not by accident.

### Where CQRS applies

CQRS is applied **selectively**, not uniformly (uniform CQRS on master-data CRUD
is pure ceremony):

| Area | Pattern | Reason |
|------|---------|--------|
| Journal posting, payment run, clearing, depreciation run, settlement, allocation | **Full CQRS** — commands via MediatR, aggregates, domain events, outbox | Complex invariants, multi-entity atomicity, audit, idempotency |
| Reports, line-item lists, trial balance, aging, table browser | **Query-only** — Dapper against read models/indexed views, no EF tracking | Read-shaped, wide, paginated, must not drag aggregates into memory |
| Master data (BP, G/L account, cost center, asset master) | **Command + EF aggregate**, no separate read model | Moderate invariants; a projection would add lag for no benefit |
| Configuration (SPRO-like) | **CRUD + validation + cache invalidation** | Low volume, high read amplification |

Read and write both hit the **same database**. No eventual-consistency read
store in Phase 3 — a trial balance that lags the posting it must reconcile with
is a defect. Materialized reporting (`rpt` schema) is introduced only in Phase 5,
and only for period-closed, snapshot-shaped data.

---

## 1.2 Layer catalogue

The brief (§3) names nine layers. Mapping to physical projects:

| Brief layer | Projects | Responsibility |
|---|---|---|
| **Domain** | `Erp.SharedKernel`, `Erp.Modules.*.Domain`, `Erp.Posting.Domain` | Entities, value objects, aggregates, invariants, domain events. No EF, no ASP.NET, no I/O. |
| **Application** | `Erp.Modules.*.Application`, `Erp.Application.Abstractions` | Commands/queries + handlers, validators, authorization behaviours, transaction scope, integration-event publishing. |
| **Infrastructure** | `Erp.Infrastructure`, `Erp.Modules.*.Infrastructure` | EF Core `DbContext`, configurations, repositories, Dapper query services, Serilog sinks, caching, outbox dispatcher, file store. |
| **Web API** | `Erp.Api` | Controllers, minimal-API endpoint groups, filters, Swagger, SignalR hubs, Problem Details. |
| **Web UI** | `erp-web` (React 19 + TS + Vite) | SPA, design system, dashboards, journal entry, SE11/SE16N screens, i18n (en/km). |
| **Background Processing** | `Erp.Jobs` (Hangfire) | Payment run, depreciation run, allocation cycles, FX valuation, dunning, outbox dispatch, dictionary activation deploys. |
| **Reporting** | `Erp.Reporting` | Report definitions, query composition, drill-down contracts, Excel/PDF renderers. |
| **Integration** | `Erp.Integration` | Webhooks, bank statement import (CAMT/MT940-shaped), CSV/Excel import pipelines, external API clients, idempotent inbound endpoints. |
| **Automated Tests** | `tests/*` | xUnit unit, integration (Testcontainers SQL Server), API contract, and the mandatory accounting rule suite (§23). |

### The dependency rule

```
Api ──► Application ──► Domain ◄── Infrastructure
 │            │                          ▲
 └────────────┴──── composition root ────┘   (DI wiring only, in Erp.Api)
```

Domain references nothing. Application references Domain + Abstractions only.
Infrastructure implements Application/Domain interfaces. Api wires them.
Enforced by an **architecture test** (`ArchitectureTests`, NetArchTest) that
fails the build on a violating reference — Phase 3, milestone M3.

---

## 1.3 Cross-cutting concerns

### Request pipeline (ASP.NET Core middleware order)

```
1. Correlation-ID          assign/propagate X-Correlation-Id → Serilog LogContext
2. Serilog request logging structured, with tenant/user/correlation enrichment
3. Exception → RFC 7807    domain/validation/authorization → typed ProblemDetails
4. Authentication          OIDC bearer or JWT; ASP.NET Core Identity backing store
5. Tenant resolution       claim → ITenantContext (ambient, scoped, immutable)
6. Authorization           policy handlers: T-code × activity × org scope
7. Endpoint                controller → MediatR
```

### MediatR behaviour pipeline (per command/query)

```
LoggingBehaviour → ValidationBehaviour (FluentValidation)
  → AuthorizationBehaviour (org-scope check on the command's declared scope)
  → IdempotencyBehaviour  (posting commands only)
  → TransactionBehaviour  (opens the SQL transaction; commits; dispatches outbox)
  → Handler
```

`TransactionBehaviour` is what guarantees §15 step 16 ("save everything in one
SQL transaction"). Handlers never call `SaveChanges` themselves.

### Concurrency & integrity

- **Optimistic concurrency** — every mutable table carries `RowVersion rowversion`;
  EF Core `IsRowVersion()`; a conflict surfaces as HTTP 409 with Problem Details.
- **Number-range concurrency** — an atomic `UPDATE … OUTPUT` against the number
  range status row inside the posting transaction (design and the discarded
  alternatives: [06-posting-engine.md §6.4](06-posting-engine.md)).
- **Transient errors** — EF Core `EnableRetryOnFailure` with a custom
  `SqlServerRetryingExecutionStrategy`; retries are safe because posting is
  idempotent on `IdempotencyKey`.
- **Immutability** — posted journal rows have no update or delete path in the
  domain model; the physical guard is documented in
  [12-database-schema-strategy.md §12.6](12-database-schema-strategy.md).

### Time

`IClock` (UTC-only) is injected everywhere; `DateTime.UtcNow` is banned by an
analyzer rule. All `datetime2(7)` columns store UTC. The user's IANA time zone
(`sec.UserProfile.TimeZoneId`) is applied **only in the presentation layer** and
in report headers. *Posting date* and *document date* are `date` — accounting
calendar values, deliberately not timestamps and never time-zone converted.

### Caching

Configuration (company codes, fiscal variants, document types, field status,
account determination, dictionary metadata) is read on nearly every posting and
changes rarely: `IConfigurationCache` backed by `IMemoryCache` (single node) or
Redis (scale-out), keyed by tenant, invalidated by integration event on any
configuration change. Never cache authorization decisions or posted data.

### Observability

Serilog structured logging (console + rolling file + SQL sink for `audit`
correlation), OpenTelemetry traces/metrics, health checks at `/health/live` and
`/health/ready`, and a posting-engine metric set (postings/sec, failures by
validation step, number-range contention, average post latency).

### Real-time

SignalR `NotificationHub` (user-scoped groups + role groups) pushes: approval
requests arriving in the inbox, approval decisions, background-job completion
(payment run, depreciation), and dictionary activation results. Delivered from
the **outbox dispatcher**, not from inside the posting transaction — an offline
client must never block a post.

---

## 1.4 Multi-tenancy

**Model chosen: shared database, shared schema, `TenantId` discriminator on every
tenant-owned table, enforced by EF Core global query filters *and* by a defence
layer in the query pipeline.**

- Every tenant-owned table: `TenantId uniqueidentifier NOT NULL`, first column of
  the clustered or a covering index, in every unique constraint.
- `ITenantContext` is resolved once per request from the token claim; it is
  immutable and cannot be set from a query string or header.
- EF Core `HasQueryFilter(e => e.TenantId == _tenant.Id)` on every entity, with a
  test that fails if any tenant-owned entity lacks one.
- Cross-tenant access exists for exactly one user type (platform operator) and is
  a separate, audited code path — not a filter bypass flag.
- SE16N and Reporting add tenant as a *mandatory, non-removable* predicate
  ([08-table-browser.md](08-table-browser.md)).

**Alternatives considered:** database-per-tenant gives the strongest isolation and
per-tenant restore, but multiplies migration and dictionary-activation work by
the tenant count — and the SE11 activation pipeline (§10) would have to fan a DDL
change across N databases transactionally. Schema-per-tenant fails for the same
reason plus SQL Server object-count limits. The discriminator model is chosen for
Phases 1–5; the design keeps the door open (all tenant filtering is centralized
in two classes) should a large customer later require physical isolation.

---

## 1.5 Technology decisions

| Concern | Choice | Note |
|---|---|---|
| Runtime | .NET 10 / ASP.NET Core 10 | LTS |
| ORM | EF Core 10 (writes, master data) + Dapper (reports, SE16N) | EF for invariants, Dapper for wide reads |
| Database | SQL Server 2025 | `decimal(19,4)` amounts, `decimal(23,6)` quantities/rates, `rowversion`, JSON columns for *non-accounting* payloads only |
| Mediation | MediatR | Commands, queries, notifications, pipeline behaviours |
| Validation | FluentValidation | Command validators; dictionary-driven field rules layered on top |
| Auth | ASP.NET Core Identity + JWT (dev/simple) or OIDC (production, external IdP) | Same claims contract either way |
| Background | **Hangfire** | Chosen over Quartz.NET: built-in persistent dashboard, retries, and job history in SQL Server — the audit story is materially better for a payment run |
| Logging | Serilog | + OpenTelemetry |
| Real-time | SignalR | Approvals, job completion |
| API docs | Swashbuckle / OpenAPI 3.1 | Per-version documents |
| Frontend | React 19, TypeScript 5, Vite, TanStack Query + Router, Zod, react-hook-form, AG Grid Community (data grids), i18next (en/km) | See [11-project-structure.md](11-project-structure.md) |
| Tests | xUnit, FluentAssertions, Testcontainers (SQL Server), Respawn, WireMock.Net, Playwright (E2E) | |
| Packaging | Docker + docker-compose; GitHub Actions CI/CD | |

---

## 1.6 Deployment topology

```
                       ┌────────────────┐
   Browser  ───HTTPS──►│  Reverse proxy │
                       └───────┬────────┘
              ┌────────────────┼─────────────────┐
              ▼                ▼                 ▼
      ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
      │  erp-web     │  │   Erp.Api    │  │   Erp.Jobs   │
      │ (static SPA) │  │  (N replicas)│  │ (Hangfire,   │
      └──────────────┘  └──────┬───────┘  │  1..N nodes) │
                               │          └──────┬───────┘
                               └────────┬────────┘
                                        ▼
                              ┌────────────────────┐
                              │  SQL Server 2025   │
                              │  (primary + read   │
                              │   replica for rpt) │
                              └────────────────────┘
                        + Redis (cache/SignalR backplane, optional)
                        + Blob/file store (attachments)
```

`Erp.Api` is stateless and horizontally scalable. `Erp.Jobs` runs the same
application assemblies with the API surface disabled, so a background payment run
executes *the identical posting engine code* as an interactive posting — one of
the reasons for the modular monolith.

---

## 1.7 Design principles that constrain every later phase

1. **One posting path.** No module may write `fin.*` transaction tables directly.
2. **The journal is the source of truth.** Every financial report is derived from
   `fin.JournalEntryLine`; no report reads a separate accumulator that could drift.
3. **Posted is immutable.** Correction = reversal or adjusting document. Ever.
4. **Metadata before screens.** A field exists in the dictionary before it exists
   on a page; labels, help, F4, and validation come from there.
5. **Authorization is organizational.** Every read and write is scoped by tenant
   and company code at minimum; CO objects add cost/profit-center scope.
6. **No business logic in triggers** (§22). Domain services only, tested.
7. **No uncontrolled DDL from the UI** (§10, §12).
8. **Everything financial is auditable** — who, what, before, after, when, from where.
