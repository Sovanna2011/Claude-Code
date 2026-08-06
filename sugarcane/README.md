# Sugarcane Planting Planning Management System

An enterprise planning and control system for **sugarcane planting operations** — by
plantation activity, tractor, equipment, material, workforce, location and schedule.

| Layer | Technology |
|-------|------------|
| Frontend | **Blazor WebAssembly** (.NET 10) + Bootstrap 5, responsive web and mobile |
| API | **ASP.NET Core 10 Web API** — REST, JWT bearer, Swagger |
| Application | C# services: projection validation, scheduling engine, MRP, capacity and scenario engines |
| Persistence | **EF Core 10** → **Microsoft SQL Server** (migrations, row-version concurrency, soft delete) |
| Identity | **ASP.NET Core Identity** with ten roles and seventeen permission policies |
| Reporting | 22 reports with print preview, **PDF** (QuestPDF) and **Excel** (ClosedXML) export |
| Tests | 190 automated tests (xUnit) — 66 unit, 107 integration, 17 against a real SQL Server |

The solution follows **Clean Architecture**: `Domain` has no dependencies, `Application`
depends only on `Domain` + `Contracts`, `Infrastructure` implements the persistence
abstractions, and `Api` / `Client` are the delivery mechanisms.

```
sugarcane/
├── SugarcanePlanning.sln
├── src/
│   ├── SugarcanePlanning.Domain/          entities · enums · PlanningFormulas
│   ├── SugarcanePlanning.Contracts/       DTOs shared by the API and the Blazor client
│   ├── SugarcanePlanning.Application/     services, engines, mapping, abstractions
│   ├── SugarcanePlanning.Infrastructure/  EF Core, Identity, audit interceptor, exporters
│   ├── SugarcanePlanning.Api/             controllers, JWT, global error handling
│   └── SugarcanePlanning.Client/          Blazor WebAssembly UI (20 screens)
├── tests/
│   ├── SugarcanePlanning.UnitTests/       formulas and engine logic
│   └── SugarcanePlanning.IntegrationTests/full process over a real service graph
├── database/01_schema.sql                 idempotent SQL Server DDL (36 tables)
├── database/generate-schema.sh            regenerates it from the migrations
├── test-system/                           docker compose stack: SQL Server + API + client
└── docs/                                  architecture · data model · deployment · user guide
```

## Quick start

The fastest way to see the whole thing running is the disposable test stack — SQL Server, the
API and the client, seeded with demo data, nothing to install but Docker:

```bash
cd sugarcane/test-system
docker compose up -d --build         # then open http://localhost:5150
```

See [test-system/README.md](test-system/README.md). To run it from source instead:

```bash
cd sugarcane

# 1. Database — either let the API migrate on start-up (default) …
#    … or run the script by hand:
sqlcmd -S localhost -d SugarcanePlanning -i database/01_schema.sql

# 2. API (http://localhost:5100, Swagger at /swagger)
dotnet run --project src/SugarcanePlanning.Api

# 3. Blazor client (http://localhost:5150)
dotnet run --project src/SugarcanePlanning.Client
```

On first run the API applies the migration and seeds a complete demo tenant: one company,
an estate with 3 farms / 6 zones / 24 blocks, a 2026 season, 3 varieties, the 19 sample
planting activities with their dependency chain, 10 tractors, 14 implements with a
compatibility matrix, 10 operators, 3 crews, 8 materials with standards and stock, and an
approved planting projection of 12 lines — plus its generated activity plans, material
requirements, a set of live resource bookings and part-recorded field progress, so the
dashboard, Gantt, MRP and variance screens all open with real content.

**Demo accounts** — password `Planner#2026` for all of them:

| User | Role | Typical task |
|------|------|--------------|
| `admin` | System Administrator | everything, including the audit log |
| `director` | Plantation Director | approve and close plans |
| `manager` | Plantation Manager | approve, revise, override dependencies |
| `planner` | Agricultural Planner | build projections, generate activity plans |
| `machinery` | Machinery Manager | tractor / equipment master, scheduling |
| `materials` | Material Planner | material master, standards, MRP |
| `supervisor` | Field Supervisor | scheduling and actual progress |
| `farmmanager` · `approver` · `viewer` | Farm Manager · Management Approver · Report Viewer | |

## The planning process

```
Configure master data
  → create growing season          seasons, varieties
  → select plantation blocks       land structure
  → create planting projection     validated per block, per season
  → generate activity schedule     19 activities, offsets, dependency lag
  → tractor requirement            ceil(area ÷ (capacity/day × working days))
  → equipment requirement          per implement category
  → material requirement           standard → base + waste, netted against stock
  → fuel and labor requirement     by area and by hour; workers = ceil(labor-days ÷ days)
  → capacity and shortage analysis Sufficient · At Risk · Shortage · Unavailable
  → adjust schedule or resources   what-if scenarios, never overwriting the plan
  → submit and approve             Draft → Submitted → Review → Approved
  → assign tractors, equipment, operators   with eight conflict checks
  → record actual progress
  → compare projection with actual
```

## Key business rules

- **Projected area never exceeds the block's plantable area** — checked per line and as a
  sum over all lines for the same block.
- **Committed plans for one block may not overlap in time** — two drafts on the same block are
  allowed, and the rule is applied again on submission, when the block is actually taken;
  planting dates must fall inside the season's planting window.
- **Header totals are always derived** from the lines, never entered.
- **A dependent activity cannot start before its blocking predecessor completes**, unless a
  manager with the override permission records a reason (kept in the audit trail).
- **No double-booking** of a tractor, implement or operator — enforced under a database lock, so
  two simultaneous requests cannot both win; no booking during maintenance; the tractor must meet
  the implement's minimum horsepower and appear on its compatibility list where one exists.
- **Revising an approved plan** copies it, issues a new version number, freezes the previous
  version read-only and records the reason, creator, reviewer and approver.
- **Every company's data is isolated** by a global query filter on `CompanyId`.

## Documentation

| Document | Contents |
|----------|----------|
| [docs/architecture.md](docs/architecture.md) | layers, dependency rules, engines, request flow |
| [docs/data-model.md](docs/data-model.md) | entity-relationship model and every table |
| [docs/deployment.md](docs/deployment.md) | build, configure, deploy to IIS / Linux / Docker / Azure |
| [docs/user-guide.md](docs/user-guide.md) | the 20 screens, step by step |
| [docs/formulas.md](docs/formulas.md) | every calculation with a worked example |

## Running the tests

```bash
dotnet test                      # 173 tests, no database required

# The 17 SQL Server tests skip unless a server is configured. To run them:
export SUGARCANE_TEST_SQLSERVER="Server=127.0.0.1,1433;User Id=sa;Password=…;TrustServerCertificate=True"
dotnet test                      # 190 tests
```

Integration tests run the real service graph (projection → activity plan → MRP → scheduling →
capacity → actuals → reports) against an isolated in-memory database, and execute the sample
data seeder itself so the start-up path is covered even without SQL Server.

The SQL Server suite covers what no in-memory provider can: that the migration applies, that
the multi-step operations survive their explicit transactions, that the unique indexes, check
constraints and `rowversion` tokens actually bite, and — by firing genuinely simultaneous
requests through separate connections — that no two callers can book the same tractor or commit
the same block. It exists because it caught five real defects, four of them invisible to every
other layer of testing; see
[docs/architecture.md](docs/architecture.md#what-only-a-real-database-caught).
