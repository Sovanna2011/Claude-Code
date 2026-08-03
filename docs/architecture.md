# HR Module — Architecture

A three-tier Human Capital Management (HCM) module modelled on **SAP ECC 6.0
EHP8**. It reproduces the core HR data model and business behaviour using an
open technology stack.

```
┌──────────────────────────┐     HTTPS/JSON      ┌──────────────────────────┐     T-SQL      ┌─────────────────┐
│      SAPUI5 (Fiori)       │  ───────────────▶   │   ASP.NET Core Web API    │  ──────────▶   │   SQL Server    │
│  MVC · JSONModel · Router │  ◀───────────────   │   EF Core · Services      │  ◀──────────   │  HR schema      │
└──────────────────────────┘                     └──────────────────────────┘                └─────────────────┘
        Presentation                                    Application                                Persistence
```

## Layer responsibilities

| Layer | Technology | Responsibility |
|-------|------------|----------------|
| Presentation | SAPUI5 1.120+, Fiori Horizon theme | Fiori-style master/detail UI, routing, value helps, personnel actions |
| Application  | C# / ASP.NET Core 8, EF Core 8 | REST API, infotype read on key date, SAP "delimit + new time slice" logic, org path evaluation |
| Persistence  | Microsoft SQL Server | SAP-faithful schema (PA infotypes, HRP1000/1001, T-tables), stored procedures, reporting views |

## SAP concepts reproduced

### Infotypes (Personnel Administration, PA)
Every PA table carries the SAP infotype key
`PERNR · SUBTY · OBJPS · SPRPS · BEGDA · ENDDA · SEQNR` plus the administrative
fields `AEDTM`/`UNAME`. Implemented:

| Table | Infotype | Meaning |
|-------|----------|---------|
| PA0000 | 0000 | Actions |
| PA0001 | 0001 | Organizational Assignment |
| PA0002 | 0002 | Personal Data |
| PA0006 | 0006 | Addresses |
| PA0007 | 0007 | Planned Working Time |
| PA0008 | 0008 | Basic Pay (+ wage type lines) |
| PA0009 | 0009 | Bank Details |
| PA0105 | 0105 | Communication |
| PA2001 | 2001 | Absences |
| PA2006 | 2006 | Absence Quotas |

### Time-dependency & time constraints
Master data is **time sliced**: reading returns the record whose
`BEGDA ≤ keyDate ≤ ENDDA`. Updates follow SAP **time constraint 1**: the
currently open record is *delimited* (its `ENDDA` set to the day before the new
`BEGDA`) and a new slice is inserted with `ENDDA = 9999-12-31`. See
`EmployeeService.UpdatePersonalDataAsync` / `ReassignAsync` and the
`usp_DelimitPA0001` stored procedure.

### Organizational Management (OM)
Built from **objects** (`HRP1000`, object types O/S/C/P) and **relationships**
(`HRP1001`, evaluation paths). The relationships used:

| RSIGN/RELAT | Meaning |
|-------------|---------|
| A 002 | reports to |
| A 003 | position belongs to org unit |
| A 007 | position described by job |
| B 008 | position held by person |
| B 012 | org unit managed by chief position |

`OrgService` walks these relationships to build the org tree, head counts and
manager (chief position) per unit — the equivalent of SAP PPOME/PPOSE.

### Customizing (T-tables)
Enterprise & personnel structure configuration lives in the standard control
tables: `T001` (company code), `T500P`/`T001P` (personnel area/subarea),
`T501`/`T503K` (employee group/subgroup), `T529A`/`T530` (actions/reasons),
`T554S` (absence types), `T005` (countries), `T512T` (wage type texts). Simple
domain fixed values (gender, marital status, employment status, communication
type) are held in `DomainValue`. Personnel numbers are drawn from a number
range (`NumberRange`, emulating SAP SNRO).

## REST API surface

| Method & path | Purpose (SAP equivalent) |
|---------------|--------------------------|
| `GET /api/employees?search=&keyDate=` | Employee list (PA20 header) |
| `GET /api/employees/{pernr}?keyDate=` | Full master data across infotypes |
| `POST /api/employees/hire` | Hiring action → creates IT0000/0001/0002 (PA40) |
| `PUT /api/employees/{pernr}/personaldata` | Change IT0002 (new time slice) |
| `PUT /api/employees/{pernr}/reassign` | Org reassignment, delimits IT0001 |
| `GET /api/employees/{pernr}/leave-balances` | IT2006 quota balances |
| `POST /api/employees/{pernr}/absences` | Record IT2001 + quota deduction |
| `GET /api/orgunits` | Flat org unit list |
| `GET /api/orgunits/{root}/structure` | Nested org hierarchy (PPOME) |
| `GET /api/orgunits/positions?orgUnitId=` | Positions with holder/vacancy |
| `GET /api/valuehelp/*` | Value helps (F4) for dropdowns |

## Design notes
- **DTO boundary** — entities never leave the API; services map to DTOs that
  also resolve text descriptions (e.g. gender key → "Male").
- **Error handling** — `ExceptionHandlingMiddleware` maps business-rule
  violations (`ArgumentException`/`InvalidOperationException`) to HTTP 400 with
  a consistent JSON body; unexpected errors to 500.
- **Transactions** — hiring, personal-data change, reassignment and absence
  recording each run in a single DB transaction.
- **Extensibility** — a new infotype is added by (1) a table + entity deriving
  from `InfotypeBase`, (2) a `ConfigureInfotype<T>` line in `HRDbContext`,
  (3) DTO/service/controller members. OM is extended by adding relationship
  types to `HRP1001` and evaluation logic in `OrgService`.
