# PO Approval — Architecture

This module reproduces the **SAP ECC 6.0 EHP8** Purchase Order **release
procedure** (MM-PUR) on an open stack: **SQL Server + ASP.NET Core + SAPUI5**,
with **user-level authentication**. It is a sibling of the HR module in this
repository and follows the same layering and SAP-faithful naming conventions.

## 1. Layers

```
 SAPUI5 (browser)                  ASP.NET Core 8 API                 SQL Server
 ┌───────────────┐   HTTPS/JSON   ┌──────────────────────────┐      ┌──────────┐
 │ Login          │──── Bearer ──▶│ AuthController            │      │ schema   │
 │ Worklist       │    token      │ PurchaseOrdersController  │      │  [PO]    │
 │ PO Detail      │◀────JSON──────│ ValueHelpController       │      │          │
 └───────────────┘               │   │                        │      │ EKKO     │
        ▲  token in               │   ▼ services               │      │ EKPO     │
        │  sessionStorage         │ AuthService                │      │ LFA1     │
        │                         │ PurchaseOrderService  ─────┼─EF──▶│ T16Fx    │
        │                         │ ReleaseService             │      │ AppUser  │
        │                         │   │                        │      │ ...      │
        │                         │   ▼ ISapEccConnector       │      └──────────┘
        │                         │ DbSapEccConnector (mirror) │
        │                         │   └▶ [ RFC/NCo connector ] ─┼───▶ SAP ECC (RFC/BAPI)
        └─────────────────────────┴────────────────────────────┘
```

- **Presentation** — SAPUI5 (MVC). A login view authenticates the user; the
  token is kept in `sessionStorage` and sent as `Authorization: Bearer …` on
  every call. A route guard forces unauthenticated users to the login page.
- **API** — thin controllers over services. A custom authentication handler
  validates the bearer token; `[Authorize]` protects every endpoint except
  `POST /api/auth/login`.
- **Domain / services** — release-strategy business rules live in
  `ReleaseService`; read models in `PurchaseOrderService`.
- **Integration** — `ISapEccConnector` is the single seam to SAP purchasing
  documents. `DbSapEccConnector` uses the SQL replica; an RFC implementation
  talks to a live ECC (see §6).
- **Persistence** — EF Core over the `[PO]` schema (SAP table names).

## 2. Data model (SAP MM-PUR)

| Table | SAP object | Purpose |
|-------|------------|---------|
| `EKKO` | Purchasing document header | PO header + release control fields |
| `EKPO` | Purchasing document item | line items and net values |
| `LFA1` | Vendor master (general) | supplier data |
| `T16FG` | Release groups | groups a set of strategies |
| `T16FS` | Release strategies | + net-value band for determination |
| `T16FC` | Release codes | approver roles (01/02/03) |
| `T16FS_Code` | Strategy steps | ordered codes per strategy (SAP FRGC1..8) |
| `T001/T024/T024E/T161` | Customizing | company code, purchasing groups/orgs, doc types |
| `ReleaseLog` | *(application)* | audit trail of release/reject/reset |
| `AppUser` / `AppUserReleaseCode` | *(application, ≈ M_EINK_FRG)* | login accounts + granted release codes |

### Release control fields on `EKKO`

| Field | Meaning |
|-------|---------|
| `FRGGR` | Release group (`T16FG`) |
| `FRGSX` | Release strategy (`T16FS`) |
| `FRGZU` | Release status — concatenation of effected codes, e.g. `0102` |
| `FRGKE` | Release indicator — `' '` not subject · `B` blocked · `R` released |
| `FRGRL` | Release incomplete — `X` until fully released |
| `RLWRT` | Total net order value (≙ `CEKKO-GNETW`) |

## 3. Release procedure (business rules)

1. **Determination** — the net order value (`Σ EKPO.NETWR`) selects the strategy
   whose band `[ValFrom, ValTo]` contains it. No band ⇒ *not subject to release*.
2. **Sequential sign-off** — the *next pending* code is the lowest-numbered step
   in `T16FS_Code` whose code is not yet in `FRGZU`. Only that code may be
   released.
3. **Authorization** — the acting user must hold `(FRGGR, code)` in
   `AppUserReleaseCode`; otherwise the API returns **403** (SAP `M_EINK_FRG`).
4. **Completion** — appending the last step's code sets `FRGKE = 'R'`,
   `FRGRL = ' '` (PO releasable for output).
5. **Reject** — resets `FRGZU` to the codes preceding the rejected step and
   re-blocks (`FRGKE = 'B'`, `FRGRL = 'X'`), logging the reason.

The same rules are expressed twice for clarity: in C# (`ReleaseService`,
authoritative at runtime) and in T-SQL (`usp_ReleasePO` / `usp_RejectRelease` /
`usp_DetermineReleaseStrategy`, for batch/import scenarios).

## 4. Authentication & authorization

- **Passwords** — PBKDF2-HMAC-SHA256 (`PasswordHasher`), stored as
  `iterations.salt.hash`. No plaintext.
- **Tokens** — `TokenService` issues a compact HMAC-SHA256 signed token
  (`base64url(payload).base64url(signature)`) carrying `sub/uid/name/exp`. No
  external JWT dependency; the signing key comes from `Auth:SigningKey`.
- **Handler** — `TokenAuthenticationHandler` (scheme `PoToken`) validates the
  bearer token and builds the `ClaimsPrincipal`; `[Authorize]` enforces it.
- **Release authorization** — checked in `ReleaseService` against the user's
  current codes in the database (so revoking a code takes effect immediately,
  independent of the token).

> The token lifetime and signing key are configurable. For production, store the
> key outside source control (environment variable / user-secrets / key vault)
> and serve the API over HTTPS.

## 5. API surface

| Method & path | Auth | Purpose |
|---------------|------|---------|
| `POST /api/auth/login` | anon | authenticate, return token + profile |
| `GET  /api/auth/me` | user | current profile & release codes |
| `GET  /api/purchase-orders` | user | worklist (`search`, `onlyPending`, `purchasingGroup`) |
| `GET  /api/purchase-orders/{ebeln}` | user | header, items, strategy steps, log |
| `POST /api/purchase-orders/{ebeln}/release` | user | release the next pending step |
| `POST /api/purchase-orders/{ebeln}/reject` | user | reject/cancel, reset strategy |
| `GET  /api/valuehelp/{name}` | user | reference-data F4 lists |
| `GET  /health` | anon | health probe |

Errors are normalised by `ExceptionHandlingMiddleware`: business-rule violations
→ **400**, authorization failures → **403**, missing entities → **404**.

## 6. Connecting to a live SAP ECC 6.0 EHP8

`ISapEccConnector` is the only place that reads/writes purchasing documents, so
the backing system is a composition-root choice:

```csharp
// Program.cs
if (useMirror)
    services.AddScoped<ISapEccConnector, DbSapEccConnector>();   // SQL replica (default)
else
    services.AddScoped<ISapEccConnector, NcoSapEccConnector>();  // your RFC impl
```

A production `NcoSapEccConnector` uses the **SAP .NET Connector (NCo 3)** with an
RFC destination built from `appsettings.json → Sap:Rfc` (`AppServerHost`,
`SystemNumber`, `Client`, `User`, `Language`) and maps the shared POCOs to:

| Operation | BAPI |
|-----------|------|
| Read headers/items | `BAPI_PO_GETITEMS`, `BAPI_PO_GETDETAIL` |
| Release / cancel | `BAPI_PO_RELEASE` |
| Commit LUW | `BAPI_TRANSACTION_COMMIT` |

Because the business services depend only on the interface and the POCOs, no
service or UI code changes when switching from the mirror to RFC.

## 7. Design decisions

- **SQL replica as the default runtime** keeps the whole module runnable without
  a SAP system, exactly like the HR module, while the connector seam keeps the
  path to real ECC integration open.
- **Strategy steps normalised** into `T16FS_Code` (instead of SAP's `FRGC1..8`
  columns + prerequisite table) for a clearer, query-friendly model.
- **Value-band determination** (instead of full classification) covers the most
  common single-characteristic release procedure without a classification engine.
- **Custom token scheme** avoids a JWT package while remaining tamper-proof and
  expiring; swap in JWT bearer if your landscape standardises on it.
