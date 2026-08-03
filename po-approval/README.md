# PO Approval Module — Purchase Order Release (SAP ECC 6.0 EHP8 style)

A full-stack **Purchase Order approval / release-strategy module** modelled on
the **SAP ECC 6.0 EHP8** Materials Management (MM-PUR) release procedure, built
on an open stack with **user-level authentication**:

- **Database** — Microsoft **SQL Server** (SAP-faithful schema: EKKO/EKPO,
  LFA1, release-strategy customizing T16Fx, stored procedures, views)
- **Backend** — **C# / ASP.NET Core 8** Web API with **Entity Framework Core**
  and token-based (bearer) authentication
- **Frontend** — **SAPUI5** (Fiori, Horizon theme) app with a **login screen**
  and role-aware release actions
- **SAP connectivity** — an `ISapEccConnector` seam abstracts the connection to
  SAP ECC purchasing documents; the shipped implementation uses the SQL replica,
  and a production build plugs in an **RFC/BAPI** connector (see below)

It reproduces the core purchase-order release procedure:

- **Release strategy determination** by net order value (SAP classification of
  `CEKKO-GNETW`) — three strategies (up to €5k, €5k–25k, €25k+)
- **Sequential sign-off** by release code (Dept. Manager → Finance → CFO)
- **Release / reject** with a full audit trail
- **Authorization** — a user may only release a code they hold (SAP auth object
  `M_EINK_FRG`, `FRGGR`/`FRGCO`)

See **[docs/architecture.md](docs/architecture.md)** for the full design and the
**[User Manual](docs/USER_MANUAL.md)** for step-by-step instructions.

```
po-approval/
├── database/                     # SQL Server scripts (run in numeric order)
│   ├── 01_create_database.sql … 08_views.sql
│   └── run_all.sql               # SQLCMD master installer
├── backend/PoApproval.Api/       # ASP.NET Core 8 Web API (EF Core)
│   ├── Models/                   # EKKO/EKPO, LFA1, T16Fx, AppUser, ...
│   ├── Data/PoDbContext.cs       # EF Core mappings (schema [PO])
│   ├── Security/                 # PasswordHasher, TokenService, auth handler
│   ├── Services/                 # Auth, PurchaseOrder, Release, ValueHelp
│   │   └── Sap/                  # ISapEccConnector + DB implementation
│   ├── DTOs/ Controllers/ Middleware/
│   └── Program.cs · appsettings.json
├── frontend/                     # SAPUI5 (Fiori) app
│   ├── webapp/                   # Component, manifest, views, controllers, i18n
│   ├── ui5.yaml · package.json
└── docs/architecture.md · docs/USER_MANUAL.md
```

## Prerequisites

| Tool | Version | Used for |
|------|---------|----------|
| SQL Server | 2019+ (or Azure SQL / LocalDB) | database |
| .NET SDK | 8.0 | backend build/run |
| Node.js | 18+ | UI5 dev server / build |

## Quick start with Docker (SQL Server + API)

The fastest way to get the backend running is the bundled Compose stack, which
starts **SQL Server 2022**, installs the database (`run_all.sql`) and builds &
runs the **API** — no local .NET SDK or SQL Server required:

```bash
cd po-approval
docker compose up --build
```

- `sqlserver` comes up first; `db-init` waits for the engine and seeds the DB;
  `api` starts once seeding completes.
- API on **http://localhost:5000** (Swagger at `/swagger`, health at `/health`).
- Override secrets via env: `SA_PASSWORD` and `AUTH_SIGNING_KEY` (defaults are
  for local use only — change them for anything shared).

Then start the SAPUI5 front end (§3); its dev server proxies `/api` to
`localhost:5000`. To run the pieces by hand instead, follow §1–§3 below.

## 1. Database

Run the scripts in order (idempotent). With **sqlcmd**:

```bash
cd po-approval/database
sqlcmd -S localhost -i run_all.sql
```

Or open `01…08` individually in SQL Server Management Studio. This creates the
`POApproval` database, the `PO` schema, all tables, the release strategy, four
demo purchase orders in different release states, and four application users.

## 2. Backend API

Set the connection string in `backend/PoApproval.Api/appsettings.json`
(`ConnectionStrings:POApproval`) if it differs from the local default, **change
`Auth:SigningKey`** for anything beyond local use, then:

```bash
cd po-approval/backend/PoApproval.Api
dotnet restore
dotnet run
```

The API starts on `http://localhost:5000` (Swagger UI at `/swagger` in
Development — click **Authorize** and paste a login token to try secured
endpoints). Health check: `GET /health`.

Quick smoke test:

```bash
# Log on and capture the token
TOKEN=$(curl -s http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"jdoe","password":"Welcome1"}' | jq -r .token)

# Worklist and one PO detail
curl -s http://localhost:5000/api/purchase-orders -H "Authorization: Bearer $TOKEN"
curl -s http://localhost:5000/api/purchase-orders/4500000002 -H "Authorization: Bearer $TOKEN"

# jdoe (Dept. Manager, code 01) releases PO 4500000001
curl -s -X POST http://localhost:5000/api/purchase-orders/4500000001/release \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}'
```

## 3. Frontend (SAPUI5)

```bash
cd po-approval/frontend
npm install
npm start
```

Opens `http://localhost:8080` and shows the **login page**. The dev server
proxies `/api/*` to the backend on port 5000 (see `ui5.yaml`), so no CORS setup
is needed for local development.

> The app bootstraps SAPUI5 from the public CDN (`ui5.sap.com`). For an
> air-gapped setup, point the bootstrap `src` in `webapp/index.html` at a local
> UI5 runtime and serve it alongside the app.

## Demo users (password `Welcome1`)

| User | Name | Release code | Can release |
|------|------|--------------|-------------|
| `jdoe` | John Doe | 01 Department Manager | step 1 of every strategy |
| `msmith` | Mary Smith | 02 Finance Controller | step 2 (POs ≥ €5,000) |
| `klee` | Karen Lee | 03 Chief Financial Officer | step 3 (POs ≥ €25,000) |
| `rbuyer` | Robert Buyer | *(none)* | view/create only |

## Demo purchase orders

| PO | Vendor | Net value | Strategy | State |
|----|--------|-----------|----------|-------|
| 4500000001 | Dell Technologies | €3,200 | S1 `[01]` | Blocked — pending 01 |
| 4500000002 | Siemens AG | €12,500 | S2 `[01,02]` | 01 released, pending 02 |
| 4500000003 | SAP SE | €48,000 | S3 `[01,02,03]` | Blocked — pending 01 |
| 4500000004 | Staples | €850 | S1 `[01]` | **Fully released** |

## Business rules of note (SAP fidelity)

- **Strategy determination** — the release group/strategy is derived from the
  net order value band (`T16FS.ValFrom..ValTo`), emulating classification of
  `CEKKO-GNETW`.
- **Sequential release** — only the next pending code (`T16FS_Code` step order)
  may be released; `FRGZU` accumulates the effected codes.
- **Completion** — when the last step signs off, `FRGKE` becomes `R` (released)
  and `FRGRL` (release-incomplete) is cleared, so the PO may be transmitted.
- **Reject** — resets the strategy from the rejected step (re-blocks the PO).
- **Authorization** — the API refuses a release for a code the user does not
  hold (403), reproducing SAP auth object `M_EINK_FRG`.

## Connecting to a live SAP ECC system

By default `Sap:UseLocalMirror` is `true` and the module uses the SQL replica of
the SAP purchasing tables. To connect to a real ECC 6.0 EHP8 system, implement
`ISapEccConnector` against the **SAP .NET Connector (NCo 3)** — mapping the same
POCOs to `BAPI_PO_GETITEMS` / `BAPI_PO_GETDETAIL` (read), `BAPI_PO_RELEASE`
(release/cancel) and `BAPI_TRANSACTION_COMMIT` — and register it in
`Program.cs`. The RFC destination is configured in the `Sap:Rfc` section of
`appsettings.json`. No business service changes are required.
