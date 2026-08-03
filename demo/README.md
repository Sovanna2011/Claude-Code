# HR Module — Full Demo System

Two ways to try it:

1. **Hosted link (nothing to install)** — a self-contained, single-file
   interactive demo you can open or share:
   **https://claude.ai/code/artifact/69dbebaf-47c7-4e77-b38f-818902a1f7e3**
   Source: [`demo/standalone.html`](standalone.html). It reproduces the same
   screens, seed data and business logic entirely in the browser (no server,
   no CDN). It mirrors the app rather than running the SAPUI5 build itself.

2. **Local full run** — the **real SAPUI5 web app** + REST API with a **single
   command and zero dependencies** (no .NET, no SQL Server, no build step):

```bash
node demo/server.js
# then open http://localhost:8080
```

### Sign in (demo accounts)

The app requires authentication. Three role-based accounts are seeded:

| Username | Password | Role | Sees |
|----------|----------|------|------|
| `admin` | `admin123` | HR Administrator | Everything: People, Org Chart, Positions, Leave, Recruitment, Training |
| `manager` | `manager123` | HR Manager | Manager Self-Service: My Team, leave Approvals, Org Chart, Training |
| `linda` | `linda123` | Employee | Employee Self-Service: My Profile, My Leave (request), Training |

Role gates both the UI and the API: employees may read only their own record;
only admins hire/maintain master data; admins & managers record time and approve
leave. In the hosted single-file demo, click a demo account to fill the form.

### Modules included

Personnel Administration · Organizational Management (org chart + **reporting
lines**) · Time Management · **Leave Management** (request/approve) ·
**Recruitment** (requisitions & applicants) · **Training** (catalog & bookings) ·
**ESS** (employee self-service) · **MSS** (manager self-service).

That's it. One Node process serves, on one origin:

- the **real SAPUI5 web app** from `frontend/webapp` (unmodified), and
- a **stand-in of the REST API** under `/api/*`.

> **Internet note:** the web app bootstraps SAPUI5 from the public CDN
> (`ui5.sap.com`), so rendering the **UI** needs an internet connection. The
> **API** endpoints work fully offline (see *Offline UI* below to also render
> the UI without internet).

---

## What this is (and isn't)

This demo lets you explore the complete front end and the exact API contract
**without** setting up the production stack.

- It is **not** the production backend. The production backend is the
  **C# / ASP.NET Core + EF Core + SQL Server** project under
  `backend/HRModule.Api`, which implements the **same REST contract** against a
  real SQL Server database. To run that, see the top-level `README.md`.
- The stand-in faithfully reproduces:
  - the **seed data** from `database/06_seed_reference_data.sql`;
  - the **core business logic** of the C# services — key-date reads, SAP
    time-constraint-1 updates (delimit + new time slice), organizational-path
    evaluation (org tree, chief position, head counts), and absence-quota
    deduction with insufficient-balance rejection;
  - the **same JSON shapes** (camelCase DTOs) the SAPUI5 app consumes.
- Data is held **in memory** and **resets on restart**, so you can experiment
  freely. You can also reset without restarting:
  `curl -X POST http://localhost:8080/api/reset`.

---

## Try it

Once running, open <http://localhost:8080> and:

1. **Employees** — search, set a **key date**, click a row to open master data.
2. **Employee detail** — browse the infotype tabs (Personal Data, Org
   Assignment, Address & Communication, Working Time & Pay, Time & Leave).
3. **Hire** — create a new employee; it gets the next personnel number and
   appears in the list and org head counts.
4. **Reassign / Edit Personal Data** — creates a new time slice; switch the key
   date to see history vs. the new record.
5. **Record Absence** — deducts from the leave quota; try to exceed it to see
   the validation error.
6. **Org Chart** / **Positions** — expand the hierarchy; see holders and
   vacancies.

### Exercise the API directly

```bash
# List employees (2 seeded: 1000 Schmidt, 1001 Nguyen)
curl http://localhost:8080/api/employees

# Hire -> assigns the next personnel number
curl -X POST http://localhost:8080/api/employees/hire -H 'Content-Type: application/json' \
  -d '{"hireDate":"2026-08-03","firstName":"Maria","lastName":"Rossi","gender":"2","companyCode":"1000","personnelArea":"1000","employeeGroup":"1","employeeSubgroup":"DU","orgUnit":50000020,"email":"m.rossi@globalcorp.com"}'

# Org hierarchy with head counts and chief position
curl http://localhost:8080/api/orgunits/50000001/structure

# Record an absence (deducts quota); exceeding the balance returns HTTP 400
curl -X POST http://localhost:8080/api/employees/1001/absences -H 'Content-Type: application/json' \
  -d '{"absenceType":"0100","begda":"2026-09-01","endda":"2026-09-03"}'
curl http://localhost:8080/api/employees/1001/leave-balances
```

Change the port with `PORT=9000 node demo/server.js`.

---

## Offline UI (local OpenUI5, no CDN)

If you have no internet (or the CDN is blocked), serve OpenUI5 locally with the
UI5 Tooling and proxy the API to this server. From `frontend/`:

1. Add a `framework` section to `frontend/ui5.yaml`:
   ```yaml
   framework:
     name: OpenUI5
     version: "1.120.14"
     libraries:
       - name: sap.ui.core
       - name: sap.m
       - name: sap.ui.layout
       - name: themelib_sap_horizon
   ```
2. Point the bootstrap in `frontend/webapp/index.html` at local resources:
   `src="resources/sap-ui-core.js"`.
3. Start the API stand-in on port 5000 (`PORT=5000 node demo/server.js`) — the
   `ui5.yaml` proxy already forwards `/api` there — then `cd frontend && npm
   install && npm start`.

This is exactly how the demo screenshots in the project were produced.

---

## Endpoints

| Method & path | Purpose |
|---------------|---------|
| `GET /api/employees?search=&keyDate=` | Employee list |
| `GET /api/employees/{pernr}?keyDate=` | Full master data |
| `POST /api/employees/hire` | Hiring action |
| `PUT /api/employees/{pernr}/personaldata` | Change personal data |
| `PUT /api/employees/{pernr}/reassign` | Organizational reassignment |
| `GET /api/employees/{pernr}/leave-balances` | Leave balances |
| `POST /api/employees/{pernr}/absences` | Record absence (IT2001) |
| `PUT /api/employees/{pernr}/address` | Maintain address (IT0006) |
| `POST /api/employees/{pernr}/family` | Add family member (IT0021) |
| `POST /api/employees/{pernr}/attendances` | Record attendance (IT2002) |
| `GET /api/orgunits?keyDate=` | Flat org unit list |
| `GET /api/orgunits/{root}/structure?keyDate=` | Nested org hierarchy |
| `GET /api/orgunits/positions?orgUnitId=` | Positions with holder/vacancy |
| `GET /api/valuehelp/*`, `GET /api/valuehelp/domain/{d}` | Value helps |
| `POST /api/reset` | Reset demo data to the seed |
| `GET /health` | Health check |
