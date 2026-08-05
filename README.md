# HR Module — Human Capital Management (SAP ECC 6.0 EHP8 style)

A full-stack **HR / HCM module** modelled on the **SAP ECC 6.0 EHP8** Human
Capital Management application, built on an open stack:

- **Database** — Microsoft **SQL Server** (SAP-faithful schema: PA infotypes,
  Organizational Management, customizing/T-tables, stored procedures, views)
- **Backend** — **C# / ASP.NET Core 8** Web API with **Entity Framework Core**
- **Frontend** — **SAPUI5** (Fiori, Horizon theme) master–detail application

It covers the core HCM sub-modules plus common HR processes:

- **PA — Personnel Administration**: infotypes 0000, 0001, 0002, 0006, 0007,
  0008, 0009, 0016, 0019, 0021, 0022, 0023, 0024, 0105 with SAP time-slicing
  and time-constraint-1 updates
- **OM — Organizational Management**: objects (HRP1000) and relationships
  (HRP1001) — org units, positions, jobs, **reporting lines (line manager)**,
  chief positions
- **PT — Time Management**: absences (2001), attendances (2002), quotas (2006)
- **Leave Management**: request / approval workflow (ESS → MSS/HR)
- **Recruitment**: job requisitions and applicant pipeline
- **Training & Event Management**: course catalog and bookings
- **ESS / MSS**: employee and manager self-service
- **Authentication & authorization**: JWT login with three roles
  (HR Administrator, HR Manager, Employee) gating both UI and API

See **[docs/architecture.md](docs/architecture.md)** for the full design, and the
**[User Manual](docs/USER_MANUAL.md)** for step-by-step end-user instructions.

## S/4HANA-inspired ERP — table catalogue

Separate from the HR module, **[docs/s4hana/](docs/s4hana/README.md)** holds the
data dictionary for the S/4HANA-inspired finance ERP: **228 tables / 4 379
fields** with table name, field name and SQL Server data type across the `org`,
`cfg`, `mdm`, `fin`, `co`, `wf`, `sec`, `audit`, `rpt` and `intg` schemas, plus a
[mapping to real S/4HANA tables and ABAP types](docs/s4hana/10_sap_reference_mapping.md)
and [nine entity relationship diagrams](docs/s4hana/11_entity_relationships.md).

The catalogue drives both physical artefacts, so they cannot drift from it:

| | |
|---|---|
| **[`backend/ErpS4.Application/`](backend/ErpS4.Application/README.md)** | The central posting engine — one validation pipeline for every module, with 36 tests covering the mandatory rules |
| **[`database/s4hana/`](database/s4hana/)** | SQL Server DDL — 228 tables, 838 foreign keys, 172 indexes, the SE11 dictionary seed, and a sample dataset (2 companies, 3 company codes, customer/vendor/dual-role partners, an asset, an intercompany pair, a KHR invoice) |
| **[`backend/ErpS4.Database/`](backend/ErpS4.Database/README.md)** | .NET 10 / EF Core 10 model — 228 entities and configurations, tenant query filters, audit stamping, append-only enforcement |

```bash
cd database/s4hana && sqlcmd -S localhost -i run_all.sql   # install the schema
python3 tools/generate_sql_ddl.py && python3 tools/generate_ef_core.py   # regenerate
python3 tools/validate_generated_sql.py                    # check before shipping
```

## Try it in one command (demo)

Want to see it running without installing .NET or SQL Server? A self-contained
demo serves the real SAPUI5 app plus a faithful stand-in of the API from a
single zero-dependency Node process:

```bash
node demo/server.js       # then open http://localhost:8080
```

See **[demo/README.md](demo/README.md)** for details. For the full production
stack (C# + SQL Server), follow the setup steps below.

```
Claude-Code/
├── database/                 # SQL Server scripts (run in numeric order)
│   ├── 01_create_database.sql … 08_views.sql
│   └── run_all.sql           # SQLCMD master installer
├── backend/HRModule.Api/     # ASP.NET Core 8 Web API (EF Core)
│   ├── Models/               # EmployeeMaster, Infotypes, OrgManagement, Customizing
│   ├── Data/HRDbContext.cs   # EF Core mappings (schema [HR])
│   ├── DTOs/ Services/ Controllers/ Middleware/
│   └── Program.cs · appsettings.json
├── frontend/                 # SAPUI5 (Fiori) app
│   ├── webapp/               # Component, manifest, views, controllers, fragments, i18n
│   ├── ui5.yaml · package.json
└── docs/architecture.md
```

## Prerequisites

| Tool | Version | Used for |
|------|---------|----------|
| SQL Server | 2019+ (or Azure SQL / LocalDB) | database |
| .NET SDK | 8.0 | HR module backend build/run |
| .NET SDK | 10.0 | S/4HANA ERP database project (`backend/ErpS4.Database`) |
| Node.js | 18+ | UI5 dev server / build |

## 1. Database

Run the scripts in order (idempotent). With **sqlcmd**:

```bash
cd database
sqlcmd -S localhost -i run_all.sql
```

Or open `01…08` individually in SQL Server Management Studio. This creates the
`HRModule` database, the `HR` schema, all tables, seed customizing data, a small
organizational structure, two demo employees (PERNR **1000** & **1001**),
stored procedures and reporting views.

## 2. Backend API

Set the connection string in `backend/HRModule.Api/appsettings.json`
(`ConnectionStrings:HRModule`) if it differs from the local default, then:

```bash
cd backend/HRModule.Api
dotnet restore
dotnet run
```

The API starts on `http://localhost:5000` (Swagger UI at `/swagger` in
Development). Health check: `GET /health`.

Quick smoke test:

```bash
curl http://localhost:5000/api/employees
curl http://localhost:5000/api/orgunits/50000001/structure
```

## 3. Frontend (SAPUI5)

```bash
cd frontend
npm install
npm start
```

Opens `http://localhost:8080`. The dev server proxies `/api/*` to the backend on
port 5000 (see `ui5.yaml`), so no CORS setup is needed for local development.

> The app bootstraps SAPUI5 from the public CDN (`ui5.sap.com`). For an
> air-gapped setup, point the bootstrap `src` in `webapp/index.html` at a local
> UI5 runtime and serve it alongside the app.

## Features

**Personnel Administration**
- Employee list with name/PERNR search and **key-date** selection (time travel)
- Full master-data detail across all infotypes in a tabbed layout
- **Hire** action (PA40-style) — creates a new personnel number + IT0000/1/2
- **Change Personal Data** (IT0002) — creates a new time slice, delimits prior
- **Organizational reassignment** (IT0001) — delimits current, carries fields forward

**Organizational Management**
- Org structure **tree** with head counts and manager per unit
- **Positions** list with job assignment, holder and vacancy status

**Time Management**
- Leave-balance display (entitlement / deducted / remaining)
- **Record absence** (IT2001) with automatic quota deduction and validation

## Business rules of note (SAP fidelity)
- **Time slicing** — reads return the record valid on the key date.
- **Time constraint 1** — updates delimit the open record at `newBegin − 1` and
  insert a new open-ended slice; no overlaps.
- **Number ranges** — personnel numbers drawn atomically from `NumberRange`.
- **Quota deduction** — an absence deducts from the matching IT2006 quota and is
  rejected if the remaining balance is insufficient.

## Demo data
| PERNR | Name | Position | Org unit |
|-------|------|----------|----------|
| 1000 | Andreas Schmidt | Head of Human Resources | Human Resources |
| 1001 | Linda Nguyen | HR Specialist | Human Resources |

Org units: **Executive Board** → { **Human Resources**, **Finance** }.
