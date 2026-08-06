# ErpS4.Api — HTTP surface

Minimal-API host over the posting engine. .NET 10, JWT bearer, RFC 7807
everywhere, OpenAPI in development.

```bash
dotnet run --project backend/ErpS4.Api      # http://localhost:5000
# GET /health, GET /openapi/v1.json (Development only)
```

## Endpoints

| Method | Route | T-code | Permission |
|--------|-------|--------|------------|
| `POST` | `/api/v1/journal-entries` | FB50 | `Finance.JournalEntry.Post` |
| `POST` | `/api/v1/journal-entries/simulate` | — | `Finance.JournalEntry.Create` |
| `POST` | `/api/v1/journal-entries/{year}/{doc}/reversal` | FB08 | `Finance.JournalEntry.Reverse` |
| `GET` | `/api/v1/journal-entries/{year}/{doc}` | FB03 | `Finance.Report.Read` |
| `GET` | `/api/v1/journal-entries/line-items` | FBL3N | `Finance.Report.Read` |
| `GET` | `/api/v1/business-partners` | BP | `Master.BusinessPartner.Read` |
| `POST` | `/api/v1/business-partners/{n}/roles` | BP_ROLE | `Master.BusinessPartner.Update` |
| `POST` | `/api/v1/business-partners/{n}/synchronize` | BP_SYNC | `Master.BusinessPartner.Update` |
| `GET` | `/api/v1/business-partners/{n}/consistency-check` | BP_CHECK | `Master.BusinessPartner.Read` |
| `POST` | `/api/v1/payments` | F-28 / F-53 | `Finance.JournalEntry.Post` |
| `POST` | `/api/v1/clearing/{year}/{doc}/reset` | — | `Finance.JournalEntry.Reverse` |
| `GET` | `/api/v1/open-items` | FBL5N / FBL1N | `Finance.Report.Read` |
| `POST` | `/api/v1/assets/{n}/acquisitions` | F-90 | `Assets.Asset.Post` |
| `POST` | `/api/v1/assets/{n}/retirement` | ABAVN | `Assets.Asset.Post` |
| `POST` | `/api/v1/depreciation-runs/preview` | — | `Assets.Asset.Read` |
| `POST` | `/api/v1/depreciation-runs` | AFAB | `Assets.Asset.Post` |
| `GET` | `/api/v1/assets` | AS03 | `Assets.Asset.Read` |
| `GET` | `/api/v1/approvals/inbox` | — | authenticated |
| `POST` | `/api/v1/approvals/{id}/approve` | — | `Finance.JournalEntry.Approve` |
| `POST` | `/api/v1/approvals/{id}/reject` | — | `Finance.JournalEntry.Approve` |
| `GET` | `/api/v1/dictionary/objects` | SE11 | `Admin.Dictionary.Read` |
| `GET` | `/api/v1/dictionary/tables/{schema}/{table}` | SE11 | `Admin.Dictionary.Read` |
| `GET` | `/api/v1/dictionary/tables/{schema}/{table}/ddl` | SE11 | `Admin.Dictionary.Read` |
| `GET` | `/api/v1/dictionary/tables/{schema}/{table}/where-used` | SE11 | `Admin.Dictionary.Read` |
| `GET` | `/api/v1/dictionary/domains/{name}` | SE11 | `Admin.Dictionary.Read` |
| `GET` | `/api/v1/dictionary/data-elements/{name}` | SE11 | `Admin.Dictionary.Read` |
| `GET` | `/api/v1/table-browser/tables` | SE16N | `Admin.TableBrowser.Read` |
| `POST` | `/api/v1/table-browser/query` | SE16N | `Admin.TableBrowser.Read` |
| `POST` | `/api/v1/table-browser/export` | SE16N | `Admin.TableBrowser.Export` |

## How the pieces fit

**Deny by default.** The fallback policy requires an authenticated user, so a
new endpoint is protected before anyone remembers to protect it. Anonymous
access is opt-in (`/health`).

**Permissions come from the database, not the token.** A revoked role takes
effect on the next request rather than at the next login. `PermissionPolicyProvider`
mints a policy per permission code on demand, so endpoints name the permission
they need without a registration list to keep in sync.

**Organisational access is checked against the body.** The company code is in
the payload, not the route, so `IOrganizationalAccessGuard` runs inside the
handler once the command is bound. Levels are ordered: `Approve` implies
`Post` implies `Write` implies `Read`.

**A rejected document is a 422, not a 400.** The request was well formed; the
accounting rules refused it. The body carries the full `errors` array — code,
message, field path, line number — so the UI puts each message beside the field
that caused it instead of showing one error at a time.

**Idempotency-Key is a header.** A retry returns `200` with the original
document; the first call returns `201`. That difference is the only way a
client can tell whether its retry did the work.

**Paging is mandatory and capped.** `PageSize` is clamped to 500, sorting is
restricted to an allow list, and every sort ends on the primary key — without a
unique tie-break two pages can repeat one row and skip another. Total count is
opt-in because it costs a second query.

**Correlation ids** are accepted or minted per request, echoed in
`X-Correlation-Id`, put in the log scope, and stored on the audit row and the
outbox message the request writes.

**Errors never leak internals.** Stack traces, SQL and connection strings stay
in the log; the client gets a stable `errorCode` and a `traceId` to quote.

**Export is a separate permission from display.** `/table-browser/query` and
`/table-browser/export` run the same code; taking rows out of the system is a
different act from looking at them, so it needs its own grant and is logged as
an export. A browser query narrowed to a company code is still checked against
the caller's organisational access, so the browser cannot hand out what the
posting endpoints refuse.

## What the stack has to be

| | |
|---|---|
| **.NET** | 10.0 throughout — every project in `backend/` is `net10.0`, including the HR module. Versions come from [`Directory.Packages.props`](../Directory.Packages.props): EF Core 10.0.10, JwtBearer 10.0.0, `Microsoft.OpenApi` pinned to 2.11.0 because everything below 2.5.0 carries GHSA-v5pm-xwqc-g5wc. |
| **SQL Server** | Built and tested on **2025 (17.0.4065.4)**, and `00_create_database.sql` raises the database to **compatibility level 170** — the level, not the product version, is what selects the cardinality estimator. **2012 is the floor** the syntax needs: `datetime2`, `rowversion`, `decimal(19,4)`, `MERGE` and `UPDATE … OUTPUT` are 2008; `OFFSET … FETCH`, `FORMAT`, `DATEFROMPARTS`, `EOMONTH` and `THROW` are 2012. Nothing needs 2025 to run. Azure SQL Database and LocalDB both work. |
| **Isolation** | `READ_COMMITTED_SNAPSHOT ON`, set by `00_create_database.sql`. That is why nothing in the code uses `NOLOCK` — readers do not block writers, and the browser never shows a half-written document. |
| **Resilience** | `EnableRetryOnFailure` covers EF's own commands; the transaction helper and the browser's raw query both run *inside* the execution strategy, so a transient error retries rather than surfacing as a 500. |
| **Connection** | `TrustServerCertificate=True` in the sample string — `Microsoft.Data.SqlClient` 4.0 and later default to `Encrypt=true`, so a local instance without a trusted certificate fails to connect without it. Remove it in production and install a real certificate. |

## Verified against a real server

Built with .NET SDK 10.0.110 and run against **SQL Server 2025 (17.0.4065.4)**
with the schema installed from `database/s4hana/run_all.sql`. The endpoints were
exercised with a JWT carrying `erp:tenant` and `erp:user`, and the permissions
resolved out of `sec.Permission` as designed:

```
GET  /health                                          -> 200 {"status":"ok"}
GET  /api/v1/table-browser/tables?search=JournalEntry -> the fin tables, FINC group
POST /api/v1/table-browser/query   fin.JournalEntryHeader -> the 7 seeded documents
POST /api/v1/table-browser/query   sec.User            -> 422 SE16N.TABLE_PROTECTED
POST .../query  PostingDate GE 2026-02-01              -> 3 rows, DateOnly parameter
POST .../query  PostingDate EQ 'not-a-date'            -> 422 SE16N.VALUE_INVALID
POST .../query  DocumentNumber EQ "x'; DROP TABLE ..." -> 0 rows, table intact
POST .../query  mdm.HouseBankAccount                   -> SELECT NULL AS [Iban] ...
POST .../query  filter on Iban                         -> 422 SE16N.FIELD_MASKED
GET  /api/v1/dictionary/tables/org/CompanyCode/where-used -> 46 referencing tables
```
