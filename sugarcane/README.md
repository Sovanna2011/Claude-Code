# Sugarcane Planting Planning — PostgreSQL · Go · SAPUI5

An enterprise planning and control system for **sugarcane planting operations**, on the hierarchy
**farm → zone → block**. This replaces the earlier ASP.NET Core / SQL Server / Blazor
implementation — that code is in git history, and nothing of it remains in the working tree.

The port is being done module by module, each complete and tested before the next.

| Module | State |
|--------|-------|
| Farm, zone and block master data, land classification, geography | **ported** |
| Farm area monitoring dashboard — KPIs, tree, map, plan versus actual | **ported** |
| Planning formulas — every calculation in the specification | **ported** |
| Growing seasons, cane varieties, the 19 planting activities and their dependency chain | **ported** |
| Planting projections, versioning and the approval workflow | **ported** |
| Activity-plan generation over the calendar | to do |
| Machinery, workforce and resource scheduling with its eight conflict checks | to do |
| Material master, standards and requirement planning | to do |
| Fuel and labour, capacity analysis, what-if scenarios | to do |
| Execution, actuals and the 22 reports | to do |

| Layer | Technology |
|-------|------------|
| Database | **PostgreSQL 16 + PostGIS 3.4** — real `geography(MultiPolygon, 4326)` boundaries, areas measured with `ST_Area` |
| Backend | **Go 1.24** REST API — repository pattern, service layer, constructor injection, transactions, global error handling, JWT role-based authorisation, optimistic concurrency, pagination and filtering, audit logging |
| Frontend | **SAPUI5 (OpenUI5 1.151)** — Fiori Horizon, `sap.ui.table.TreeTable`, KPI tiles, analytical charts, filter bar, interactive map |
| Tests | 135 Go tests — 76 unit, 59 integration against a real PostGIS database |

```
sugarcane/
├── db/migrations/          schema and seed, applied in order on start-up
│   ├── 001_schema.sql      hierarchy, land classification, planting records, identity, audit
│   ├── 002_seed.sql        the sample plantation
│   ├── 003_activities.sql  seasons, varieties, the activity master and its dependencies
│   ├── 004_activity_seed.sql  the specification's nineteen sample activities
│   └── 005_projections.sql  planting projections, their lines and the approval trail
├── backend/
│   ├── cmd/api/            the composition root
│   └── internal/
│       ├── domain/         planning formulas, area classification, validation, model types
│       ├── repository/     every line of SQL in the system
│       ├── service/        business rules, transactions, audit
│       ├── httpapi/        handlers, middleware, error rendering
│       ├── auth/           JWT and password hashing
│       ├── database/       pool, migrations, transaction helper
│       └── config/         environment
├── frontend/webapp/        the SAPUI5 application
└── scripts/api.sh          start · stop · restart the API locally
```

## Quick start

```bash
# 1. Database
createdb farmarea
psql -d farmarea -c 'CREATE EXTENSION postgis'

# 2. API — applies the migrations and seeds the sample plantation on first run
cd sugarcane && ./scripts/api.sh start          # http://localhost:8080

# 3. Dashboard
cd frontend && npm install && npx ui5 serve --port 8081   # http://localhost:8081
```

Sign in with **admin / Farm#2026**. The other accounts — `manager`, `planner`, `viewer` — share
the password and differ in what they may change.

The sample plantation is one company, one plantation, three farms, nine zones (one of them empty,
so the roll-up is exercised against an empty branch) and twenty blocks near Lusaka, with real
polygon boundaries, a breakdown of why each block's unplantable part cannot be planted, and two
crop years of planting records — 2025 finished, 2026 part recorded — so variance and monthly
progress have something to show.

## What the dashboard shows

**KPI cards** — total area, new planting, ratoon, area with cane, available for planting, cannot
be planted. Each carries its hectares and its share of the total area. Every figure is calculated
from block-level data; nothing is entered at dashboard level.

**Charts** — land utilisation, new planting versus ratoon, area by farm, area by zone, planting
progress by month. Clicking a farm or zone bar filters the whole screen to it.

**Tree report** — farm → zone → block in a `TreeTable`, expandable, searchable, with a Google Maps
link on every row and an export to Excel that mirrors the hierarchy as a real outline.

**Map** — the blocks' own polygons from PostGIS, with the farm and zone outlines beneath them,
coloured by cane status. Clicking a block selects its row in the tree and fills the detail panel;
selecting a block row highlights it on the map.

**Planning versus actual** — planned against actual by farm, zone or block, with variance and
achievement, and a monthly breakdown beside it.

**Block maintenance** — the pencil on a block row, or *Edit* on the map's detail panel, opens the
block's master data: its fields, the breakdown of why part of it cannot be planted, and its
planting records. The derived figures — the unplantable remainder, the area with cane, the area
still available — move as you type and cannot be entered. A rejected save keeps the dialog open,
puts the server's message on the field it is about, and says how much of the unplantable area the
recorded reasons account for. A Report Viewer sees the dialog but no Save button.

Changing any filter refreshes the cards, the charts, the tree, the map and the comparison in one
pass, from one filter value — they cannot end up describing different land.

## Planting projections

A projection is the committed answer to **what will be planted, where, when and with which
variety**. Everything downstream is generated from an approved one, so it carries a workflow rather
than being ordinary master data.

*Planting projections* in the dashboard header opens the plan list; a plan opens on its blocks, its
header and its approval trail, with the projected area, harvestable area, expected tonnage and seed
cane across the top. All four are derived — from the lines, and the lines from the block, the
variety's yield, its expected loss and its seed rate. None of them has a field to type into.

| Step | Who | What it means |
|------|-----|---------------|
| Submit | Planner, Manager, Admin | The draft is finished. A plan with no blocks is refused. |
| Start review | Manager, Admin | Someone is looking at it. Optional — a manager may approve straight away. |
| Approve | Manager, Admin | **The land is committed.** No other plan may take those blocks in that window. |
| Reject · Return for correction | Manager, Admin | Refused, or sent back to draft. Both need a reason. |
| Open a revision | Planner, Manager, Admin | A new version carrying a copy of the blocks; this one is marked superseded. |
| Close | Manager, Admin | Finished. Nothing further happens to it. |

The buttons a screen shows come from the server, on the projection itself — the same graph the
server enforces, so a button that appears always works and one that would be refused never appears.
A Report Viewer is offered none of them.

Two planners may draft alternatives for the same block; that is how options get compared. The land
is taken only on approval, and a second plan overlapping an approved one in time is refused naming
the plan that holds it. Revising an approved plan releases the commitment until the new version is
approved in its turn.

## Land classification

Only two figures about a block are ever entered: its **total area** and its **plantable area**.
Everything else is derived.

| Figure | Definition |
|--------|------------|
| Total area | Everything inside the boundary. |
| Plantable area | The part that can carry cane. |
| Cannot be planted | Total less plantable, itemised by reason — road, canal, pond, building, mountain, forest, flooded, infrastructure, reserved, other. |
| New planting | Recorded planting of a fresh crop for the filtered crop year. |
| Ratoon | Recorded ratoon area for the filtered crop year. |
| Area with cane | New planting plus ratoon. |
| Available for planting | Plantable area with no cane on it. |

Farm and zone rows have no area columns **in the database at all**: their figures are the sum of
the blocks beneath them, and the surest way to stop someone entering one by hand is to give them
nowhere to enter it.

## The rules the system will not let you break

Section 10 of the specification is enforced twice — in Go, so the message names the field, and in
PostgreSQL, so a migration script or a direct `UPDATE` cannot walk past it.

- Total, plantable, new planting and ratoon area cannot be negative.
- Plantable area cannot exceed the total area (`CHECK`).
- The recorded non-plantable reasons cannot account for more than the unplantable part.
- New planting, ratoon, and the two of them together, cannot exceed the plantable area.
- Shrinking a block's plantable area below the cane already on it is refused.
- An activity that needs a tractor or an implement must have a daily capacity, or the engine
  cannot work out how long it takes.
- A dependency that would close a loop is refused — every activity in the cycle would wait for
  another that waits for it, and the schedule could never be satisfied. A recursive walk enforces
  it at commit, so a data fix applied with `psql` cannot create one either.
- A projection line cannot exceed its block, and neither can every line one plan has on that
  block put together — a new planting and a ratoon share the same ground.
- An approved plan cannot be edited, only revised: other modules have already read it.
- Two approved plans cannot hold the same block over overlapping planting dates.

Several of these span rows — one new-planting record plus one ratoon record on the same block — so
no single `CHECK` can express them. They are deferred constraint triggers, checked at `COMMIT`,
which lets a caller move area between the two records in either order without tripping over itself
halfway. A deferred trigger's exception surfaces from `COMMIT` rather than from the statement that
caused it, past everything the repository maps, so the projection service translates commit errors
too — otherwise breaking a cross-row rule would answer 500 instead of naming the field.

## API

| Endpoint | Purpose |
|----------|---------|
| `POST /api/auth/login` · `GET /api/auth/me` | sign in, current user |
| `GET/POST /api/farms` · `GET/PUT /api/farms/{id}` | farm master |
| `GET /api/farms/{id}/zones` | zones of a farm |
| `GET/POST /api/zones` · `GET/PUT /api/zones/{id}` | zone master |
| `GET /api/zones/{id}/blocks` | blocks of a zone |
| `GET/POST /api/blocks` · `GET/PUT /api/blocks/{id}` | block master, with the non-plantable breakdown |
| `GET/PUT /api/blocks/{id}/geometry` | boundary as GeoJSON; the response carries the area PostGIS measures |
| `GET /api/land-classification` | the non-plantable reasons |
| `GET/POST /api/seasons` · `GET/PUT /api/seasons/{id}` | growing seasons and their planting window |
| `GET/POST /api/varieties` · `GET/PUT /api/varieties/{id}` | cane varieties: seed rate, yield, loss |
| `GET/POST /api/activities` · `GET/PUT /api/activities/{id}` | the planting activity master |
| `GET /api/activities/chain` | the activity chain in the order the engine walks it |
| `POST /api/activities/dependencies` · `DELETE .../{id}` | "this activity waits for that one" |
| `GET/POST /api/projections` · `GET/PUT /api/projections/{id}` | planting projections |
| `POST /api/projections/{id}/lines` · `PUT`/`DELETE` `.../lines/{lineId}` | the blocks a plan covers |
| `POST /api/projections/{id}/{submit\|review\|approve\|reject\|returnforcorrection\|revise\|close}` | one step of the workflow |
| `GET /api/projections/workflow` | the transition graph, so a client need not hold a copy |
| `GET/POST /api/planting` · `DELETE /api/planting/{id}` | planting records, planned and actual |
| `GET /api/summaries/farm-area` · `/zone-area` · `/block-area` | one level of the tree |
| `GET /api/farms/tree` · `GET /api/reports/farm-area-tree` | the whole hierarchy |
| `GET /api/reports/farm-area-tree.xlsx` | the same report as a workbook |
| `GET /api/dashboard/farm-area` | KPI cards |
| `GET /api/dashboard/farm-area/map` | block polygons and farm/zone outlines as GeoJSON |
| `GET /api/dashboard/charts` | every chart series in one response |
| `GET /api/reports/planting-plan-vs-actual` | planned against actual, by farm, zone, block and month |
| `GET /api/lookups/{kind}` | the filter bar's values |
| `GET /api/audit` | who changed what (administrators only) |

Every read takes the same filter parameters: `companyId`, `plantationId`, `farmId`, `zoneId`,
`blockId`, `cropYear`, `plantingYear`, `seasonId`, `varietyId`, `plantingType`, `landStatus`,
`caneStatus`, `search`, plus `page` and `pageSize` on the list endpoints.

```bash
TOKEN=$(curl -s -X POST localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"userName":"admin","password":"Farm#2026"}' | jq -r .token)

curl -s "localhost:8080/api/dashboard/farm-area?cropYear=2026" -H "Authorization: Bearer $TOKEN" | jq
```

## Roles

| Role | May |
|------|-----|
| Admin | everything, including the audit trail |
| Manager | farm, zone and block master data, geometry, planting records, and deciding projections |
| Planner | planting records, projections, and submitting them |
| Viewer | read only |

## Configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `FARMAREA_DATABASE_URL` | — (required) | PostgreSQL connection string |
| `FARMAREA_JWT_SECRET` | — (required, ≥ 32 chars) | token signing key |
| `FARMAREA_ADDR` | `:8080` | listen address |
| `FARMAREA_CORS_ORIGINS` | `http://localhost:8081` | comma-separated origins allowed to call the API |
| `FARMAREA_TOKEN_TTL_MINUTES` | `480` | token lifetime |
| `FARMAREA_MIGRATE_ON_START` | `true` | apply pending migrations at start-up |
| `FARMAREA_MIGRATIONS_DIR` | `../db/migrations` | where they live |

## Tests

```bash
cd backend
go test ./...                                   # 76 unit tests; the database tests skip

createdb farmarea_test && psql -d farmarea_test -c 'CREATE EXTENSION postgis'
export FARMAREA_TEST_DATABASE_URL="postgres://farmarea:farmarea@127.0.0.1:5432/farmarea_test"
go test ./...                                   # 135 tests
```

The integration tests run over the real stack — HTTP handler, service, repository, PostGIS — and
check what no in-memory substitute can: that the SQL is valid, that the constraint triggers fire,
that `ST_Area` agrees with the registered figures, that the tree, the KPI cards and the map
describe the same land under the same filter, that a stale version is a 409 rather than a silent
overwrite, that approving a plan over land another plan already holds is refused, and that the
exported bytes actually open as a workbook.

## Notes on the technology choices

**OpenUI5, not the SAP-delivered SAPUI5.** The runtime is vendored with the application from npm,
so the dashboard starts on an estate network with no route to a CDN. The consequence is that
`sap.viz` — the closed-source chart library — is not available, so the five analytical charts are
drawn by a small custom control (`control/AreaChart.js`) as inline SVG, using the theme's own
colour parameters. To switch to the SAP distribution instead, point the bootstrap in
`webapp/index.html` at `https://ui5.sap.com/resources/sap-ui-core.js`; nothing else changes, and
the charts can then be replaced with `VizFrame` if you prefer.

**The map is drawn by the application, not by Google.** The boundaries are the system's own data
in PostGIS, the estate has to be visible without an internet connection, and an embedded Google
map cannot draw a polygon the system stores without an API key and a per-load charge. Google Maps
is used where it is genuinely better — the *Open in Google Maps* action on every farm, zone and
block, which hands the coordinates to Google for a satellite view.
