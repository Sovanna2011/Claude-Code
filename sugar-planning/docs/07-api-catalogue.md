# 7. API catalogue

The machine-readable contract is `backend/internal/api/openapi.yaml`, embedded
in the binary and served at `GET /api/v1/openapi.yaml`. It is OpenAPI 3.1 and
carries request and response schemas, examples and error shapes. This document
is the map.

Two tests hold the document to the code, one in each direction: a public test
parses the specification and asserts that every documented path is routed, and
an internal one reads the router's own pattern list and asserts that every route
is documented. The first catches a specification running ahead of the code; the
second catches the quieter failure, an endpoint added and never written down.

---

## 7.1 Conventions

| Concern | Rule |
| --- | --- |
| Base path | `/api/v1` |
| Authentication | `Authorization: Bearer <token>` on every path except the sign-in and the specification itself |
| Timestamps | ISO 8601, stored in UTC |
| Business dates | ISO calendar dates in the factory time zone, no time component |
| Numbers | Exact decimals carried as **JSON strings**, for example `"16788.321"`. A JSON number would be read back as a binary float by most clients, and 2,300,000 t split across 137 days does not survive that. A request may send either form; a response is always a string |
| Lists | `$top`, `$skip`, `$search`; response `{ value, count, skip, top }` |
| Filtering | Documented, allow-listed query parameters only. No client-supplied query language |
| Concurrency | `ETag` on read, `If-Match` on write; mismatch is `412` |
| Idempotency | `Idempotency-Key` on posting endpoints; a repeat replays the first response |
| Errors | RFC 9457 problem documents, served as `application/problem+json`, with `errors[]` addressed by `row` and `field` |
| Correlation | `X-Correlation-Id` echoed on every response and stored on the audit record |

### Status codes

| Code | Meaning |
| --- | --- |
| 200 / 201 / 204 | Success |
| 400 | Validation failure; `errors[]` names the row and field |
| 401 | No usable token |
| 403 | Permission or data scope |
| 404 | No such record |
| 409 | Duplicate business key, locked period, or an illegal workflow action |
| 412 | `If-Match` did not match: somebody else changed the record |
| 429 | Rate limit |
| 500 | Unexpected; the detail is a correlation id, never a stack trace |

---

## 7.2 Endpoints

### Session

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| GET | `/session` | authenticated | Profile, roles, permissions, data scope |
| POST | `/auth/dev-login` | public | Development sign-in. Refused unless `AUTH_MODE=dev` |

### Master data

Fourteen entities behind one uniform shape: `companies`, `factories`,
`production-lines`, `shifts`, `product-categories`, `products`, `units`,
`unit-conversions`, `packaging-types`, `warehouses`, `customers`,
`shipment-channels`, `materials`, `reason-codes`.

| Method | Path | Permission |
| --- | --- | --- |
| GET | `/master/{entity}` | `masterdata:read` |
| GET | `/master/{entity}/{id}` | `masterdata:read` |
| POST | `/master/{entity}` | `masterdata:write` |
| PUT | `/master/{entity}/{id}` | `masterdata:write` |
| DELETE | `/master/{entity}/{id}` | `masterdata:write`, `If-Match` required |

DELETE deactivates; master data is never removed.

### Seasons and versions

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| GET | `/seasons` | `plan:read` | Seasons in scope |
| POST | `/seasons` | `plan:write` | Create a season and open its actuals container |
| GET/PUT | `/seasons/{id}` | `plan:read` / `plan:write` | |
| GET | `/seasons/{id}/versions` | `plan:read` | |
| POST | `/seasons/{id}/versions` | `plan:write` | |
| GET | `/versions/{id}` | `plan:read` | Version with assumptions, mix and the actions this caller may take |
| PUT | `/versions/{id}` | `plan:write` | Header only; status belongs to the workflow |
| PUT | `/versions/{id}/assumptions` | `plan:write` | Upsert on `(version, code, validFrom)` |
| PUT | `/versions/{id}/product-mix` | `plan:write` | |
| DELETE | `/versions/{id}/product-mix/{mixId}` | `plan:write` | |

### Planning

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| POST | `/versions/{id}/generate` | `plan:write` | Rebuild the daily plan; returns the summary and the warnings |
| POST | `/versions/{id}/copy` | `plan:write` | Scenario copy with assumption overrides |
| POST | `/versions/compare` | `plan:read` | Two versions by date, process, product, warehouse or channel |
| GET/POST | `/versions/{id}/cane` | `plan:read` / `plan:write` or `actual:cane` | |
| GET/POST | `/versions/{id}/production` | `plan:read` / `plan:write` or `actual:production` | |
| GET/POST | `/versions/{id}/storage` | `plan:read` / `plan:write` or `actual:stock` | |
| GET/POST | `/versions/{id}/shipments` | `plan:read` / `plan:write` or `actual:shipment` | |

The write permission depends on the `series` in the payload: `PLAN` rows need
`plan:write`, `ACTUAL` rows need the operator permission for that area.

### Workflow

| Method | Path | Permission |
| --- | --- | --- |
| POST | `/versions/{id}/transition` | depends on the action |

`SUBMIT`/`RECALL` need `plan:submit`; `APPROVE`/`REJECT` need `plan:approve`;
`RELEASE`/`SUPERSEDE`/`CLOSE` need `plan:release`; `REOPEN` needs `plan:reopen`.
`REJECT`, `SUPERSEDE`, `CLOSE` and `REOPEN` require a reason.

### Analytics, materials, downtime

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| GET | `/dashboard` | `plan:read` | KPIs, trend, storage, shipments, downtime, alerts |
| GET | `/versions/{id}/material-requirements` | `materials:read` | Packaging requirement planning |
| GET | `/downtime` | `plan:read` | |
| POST | `/downtime` | `downtime:write` | |

### Reports and audit

| Method | Path | Permission |
| --- | --- | --- |
| GET | `/reports` | `report:read` |
| GET | `/reports/{code}?format=json\|csv\|xlsx\|pdf` | `report:read` |
| GET | `/audit` | `audit:read` |

### Operations

| Method | Path | Auth |
| --- | --- | --- |
| GET | `/healthz` | public — liveness |
| GET | `/readyz` | public — readiness, checks the database |
| GET | `/api/v1/openapi.yaml` | public |

---

## 7.3 Worked examples

### Bulk upsert with a row-level rejection

```http
POST /api/v1/versions/{id}/cane
Content-Type: application/json

{ "rows": [
    { "businessDate": "2026-12-01", "series": "ACTUAL", "caneCrushed": 15000, "availableHours": 24 },
    { "businessDate": "2026-12-02", "series": "ACTUAL", "caneCrushed": -5,    "availableHours": 24 }
] }
```

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "type": "https://sugarplan.example.com/problems/validation",
  "title": "The request was not valid",
  "status": 400,
  "detail": "One or more fields could not be accepted.",
  "correlationId": "6f1c…",
  "errors": [
    { "row": 1, "field": "caneCrushed", "code": "NEGATIVE",
      "message": "caneCrushed cannot be negative" }
  ]
}
```

Nothing was written: the batch is all-or-nothing unless `"partial": true` is
sent. `row` is the index in the payload, and it is present even when it is `0`,
so a grid can highlight the first row without guessing.

### Optimistic concurrency

```http
GET /api/v1/seasons/{id}          →  200, ETag: "3"
PUT /api/v1/seasons/{id}          with If-Match: "3"  →  200, ETag: "4"
PUT /api/v1/seasons/{id}          with If-Match: "3"  →  412
```

```json
{
  "type": "https://sugarplan.example.com/problems/version-conflict",
  "title": "The record was changed by somebody else",
  "status": 412,
  "detail": "Season 2026-2027 was changed by planner (version 4, you have 3)"
}
```

### Generating a plan

```http
POST /api/v1/versions/{id}/generate
Idempotency-Key: 4f2a-…

{ "replace": true }
```

```json
{
  "summary": {
    "caneAllocatedTons": "2300000",
    "rawSugarExpectedTons": "253000.000",
    "finishedGoodsTons": "242100",
    "remeltInputTons": "254205.000",
    "workingDays": 137
  },
  "warnings": [
    { "code": "REMELT_SUPPLY_SHORT", "severity": "ERROR",
      "title": "Refining demand exceeds the planned raw sugar supply",
      "detail": "The finished goods mix needs 254205.000 t of raw sugar at a factor of 1.05, but the plan produces only 253000.000 t. The shortfall of 1205.000 t first appears on 2027-04-16. …" },
    { "code": "CAPACITY_full", "severity": "ERROR",
      "title": "Refined/White Warehouse 1 reaches 100% of usable capacity",
      "date": "2027-01-01" }
  ],
  "rowCounts": { "cane": 137, "production": 411, "storage": 685, "shipment": 411 },
  "firstDate": "2026-12-01", "lastDate": "2027-04-16"
}
```

A repeat with the same `Idempotency-Key` answers `Idempotent-Replay: true`
rather than rebuilding.


---

## 7.4 Execution

The daily factory. Twenty-two endpoints across stock, production orders, quality
and maintenance.

### Stock and postings

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| GET | `/stock` | Balance, held quantity, available and utilisation per warehouse and product | `plan:read` |
| GET | `/inventory/documents` | The ledger, newest first, each document with its lines | `plan:read` |
| POST | `/inventory/documents` | Move stock | `actual:stock` |
| GET | `/inventory/documents/{id}` | One posting | `plan:read` |
| POST | `/inventory/documents/{id}/reverse` | Post the counter-document | `actual:stock` |

A posting request carries **unsigned** quantities; the document type gives them
their sign. That is how a warehouse keeper thinks about it - an issue of 120 t is
"issue 120", not "add minus 120" - and it removes a whole class of sign errors.
An adjustment and a count are the exception: they carry their own sign, because
a correction can go either way. A transfer names the receiving store and is
expanded server-side into the pair of signed lines that move both balances, so
the two halves can never be posted apart from one another.

A posting is refused, with the offending line named in `errors[]`, when it would

- drive a balance below zero,
- move stock that quality has blocked, or
- take a store past its usable capacity.

`allowNegativeStock` and `capacityOverride` ask for the first and third to be
waived. Both need `capacity:override`, both are refused rather than silently
ignored when the caller lacks it, and both are recorded on the audit event.

Every posting endpoint honours `Idempotency-Key`. The key is claimed before the
work starts, so two concurrent retries cannot both post; the response is attached
afterwards, so a later retry replays the document the first request produced
rather than a bare acknowledgement.

### Production orders

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| GET | `/production-orders` | List, filtered by factory, date, line, product, status, or open only | `plan:read` |
| POST | `/production-orders` | Create one by hand | `actual:production` |
| GET | `/production-orders/{id}` | The order, its confirmations, and what may be done next | `plan:read` |
| POST | `/production-orders/{id}/action` | Release, complete, technically close, cancel | `actual:production` |
| POST | `/production-orders/{id}/confirm` | Record what a shift produced | `actual:production` |
| POST | `/versions/{id}/production-orders` | Create orders from the released plan | `actual:production` |
| POST | `/confirmations/{id}/reverse` | Undo a confirmation | `actual:production` |

Orders may only be raised from a **released** plan: an order is an instruction to
the floor, and instructions do not come from a draft somebody is still editing.
The actuals container is refused too, even though it is released from the day it
is created - it records what happened rather than what to make.

Confirming writes the confirmation, receipts the good output into the named
warehouse, advances the order's quantity and status, and records the audit event
in one transaction. Scrap and rework are recorded but not receipted: scrap has
left the process and rework has not finished it, and inventing a balance for
either would put sugar in a shed that does not hold any. Omitting the warehouse
records the production without moving stock.

Closing an order whose confirmed quantity is more than the tolerance away from
its plan needs a `varianceReason` that is a configured reason code, **and** a
caller holding `plan:approve`. The tolerance defaults to 5% and is overridden per
plan by the `CLOSE_VARIANCE_TOLERANCE_PCT` assumption.

### Quality

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| GET / PUT | `/quality/parameters` | The measurable properties | `masterdata:read` / `masterdata:write` |
| GET / PUT | `/quality/specs` | Effective-dated limits for a product | `masterdata:read` / `masterdata:write` |
| GET | `/quality/samples` | Samples with their results | `plan:read` |
| POST | `/quality/samples` | Open a sample | `quality:write` |
| GET | `/quality/samples/{id}` | One sample | `plan:read` |
| POST | `/quality/samples/{id}/results` | Enter the laboratory sheet | `quality:write` |
| GET | `/quality/holds` | Holds on stock | `plan:read` |
| POST | `/quality/holds` | Block stock by hand | `quality:write` |
| POST | `/quality/holds/{id}/release` | Free blocked stock | `quality:release` |

Each measurement is judged against the specification in force on the **sample's**
business date, and the limits are copied onto the result rather than pointed at,
so a result keeps saying what it was judged against even after somebody edits the
specification. Where two specifications for one parameter are both in force - a
limit tightened without ending the older one - the later start date wins.

A measurement whose parameter has no specification in force is recorded and
reported back in `unspecified`. It judges nothing: an unspecified parameter is a
configuration gap, and passing it silently is how a limit goes years without
being set.

A completed sheet whose verdict is `FAIL` blocks the quantity named on it. The
hold record and the HOLD posting that moves the held quantity are written
together, so a hold on the quality record and a balance that still shows the
sugar as available cannot drift apart.

Placing a hold sits behind `quality:write` and releasing one behind
`quality:release`, so a site that wants the person who stops the sugar leaving to
be a different person from the one who lets it go can arrange that by granting
the two to different roles.

### Maintenance

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| GET | `/maintenance` | The outage calendar | `plan:read` |
| PUT | `/maintenance` | Create, amend or approve a window | `downtime:write`, plus `plan:approve` to approve |

Approving is a second permission because an approved, factory-wide window
lengthens the campaign: the generator treats its days as non-working and the
season's end date moves. A line-specific window does not stop the mill and never
removes a crushing day. The generator reads the calendar itself, so nobody has to
remember to type the dates into a generate request.

---

## 7.5 What the API does not do

Worth stating so nobody looks for it:

- **No client-supplied query language.** Filters are named parameters on an
  allow-list. There is no `$filter` expression evaluator to escape from.
- **No unbounded lists.** `$top` is capped at 1,000; daily-row queries are
  bounded by their date range.
- **No unknown fields.** A payload with a misspelled field is rejected rather
  than silently ignored, so a typo surfaces in development.
- **No partial financial postings.** A bulk write is all-or-nothing unless the
  caller explicitly asks for partial acceptance, which the planning grid never
  does.
