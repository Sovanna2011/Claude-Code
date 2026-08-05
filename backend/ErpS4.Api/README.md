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

> Not compiled or run: no .NET SDK is available in this environment
> (`builds.dotnet.microsoft.com` is blocked by network policy). Run
> `dotnet build backend/ErpS4.Api` before relying on it.
