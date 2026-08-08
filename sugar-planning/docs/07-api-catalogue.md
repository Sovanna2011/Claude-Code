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
| Lists | `$top`, `$skip`, `$search`, `$orderby`; response `{ value, count, skip, top }` |
| Filtering | Documented, allow-listed query parameters only. No client-supplied query language |
| Sorting | `$orderby` on master-data lists, against an allow-listed column set. Transaction lists carry a fixed, meaningful order instead — a stock ledger sorted by tonnage is not a ledger |
| Field selection | Not offered. See below |
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

### Cane supply

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| GET | `/versions/{id}/supply` | `plan:read` | The commitments, the reconciliation against the season target, and the warnings |
| PUT | `/versions/{id}/supply` | `plan:write` | Commit a source to the season, or correct its commitment |
| DELETE | `/versions/{id}/supply/{entryId}` | `plan:write` | Remove a commitment |
| POST | `/versions/{id}/supply/generate` | `plan:write` | Build the delivery schedule from the commitments |
| GET | `/versions/{id}/cane-supply` | `plan:read` | The schedule, or the deliveries recorded against it |
| POST | `/versions/{id}/cane-supply` | `plan:write` or `actual:cane` | Write schedule rows, or record what arrived at the gate |

`PUT` on `/supply` rather than `POST`: there is one commitment per source per
version, so writing the same pair again is a correction and not a second
contract. The same asymmetry as the daily rows applies to `/cane-supply` —
`PLAN` rows need `plan:write`, `ACTUAL` rows need `actual:cane`, so a planner
cannot record what came through the gate.

`/cane-supply` takes `series`, `from`, `to`, `sourceId` (repeatable) and
`$top`. It comes back ordered by date and then by source, which is the order a
delivery schedule is read in and what a caller taking the first N rows depends
on.

Generating the schedule is idempotent under a repeated `Idempotency-Key`: it is
expensive and it is exactly the request a client retries.

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

### Certificate of analysis

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| GET | `/quality/samples/{id}/certificate` | The document that goes with a consignment | `report:read` |

`?format=pdf` is what is put in the envelope; `json` is what the screen shows.

Only a **completed** sample can produce one. A certificate is read as a
guarantee, and issuing one from a half-finished sheet would be a statement the
laboratory has not made yet.

The limits printed are those copied onto each result when the sample was
completed, not the specification in force today. A limit tightened next season
must not retrospectively change what a customer was told about sugar shipped
this one — which is also why the limits are copied onto the result in the first
place rather than pointed at.

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

## 7.5 Materials

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| GET | `/master/packaging-bom` | The packaging bill of materials | `masterdata:read` |
| PUT | `/master/packaging-bom` | Add or change a line | `masterdata:write` |
| DELETE | `/master/packaging-bom/{id}` | Remove a line | `masterdata:write` |

What a package consumes besides its own bag: the liner inside the jumbo bag, the
thread that sews it, the label on the consumer pack, a share of the pallet it is
stacked on. The primary bag is not held here — a packaging type points at its own
bag material — because a bag is one per package by definition, and giving it a
quantity would invite somebody to set it to two.

The business key is the pair of packaging type and material, so the same material
cannot appear twice on one package. Quantities are scale 6: 0.0035 spools of
thread per bag is a real figure, not a rounding error.

Two things read it. Requirement planning (`/materials/requirements`) adds the
components to the bags, so the report does not say the factory needs bags and
nothing else. And a production confirmation that names no components takes them
from here — because an operator confirming a shift is not going to key how many
liners went into 300 jumbo bags, and a consumption nobody records is a stock
figure that drifts until somebody counts the shed. An operator who counted what
actually went out of the store can still send the real figures, and then only
those are recorded.

---

## 7.6 Imports

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| GET | `/import-fields` | What each kind of file may contain | `plan:read` |
| GET | `/import-mappings` | The mapping templates | `plan:read` |
| PUT | `/import-mappings` | Create or change a template | `masterdata:write` |
| POST | `/imports` | Upload a file and stage it | the permission its commit will need |
| GET | `/imports` | The import history | `plan:read` |
| GET | `/imports/{id}` | A staged import and its rows | `plan:read` |
| GET | `/imports/{id}/errors` | The rows that failed, as CSV | `plan:read` |
| POST | `/imports/{id}/commit` | Write it into the plan | as above |
| POST | `/imports/{id}/cancel` | Abandon it | `plan:read` |

**A file is never posted as it arrives.** It is read through its mapping,
staged, validated row by row, shown to somebody, and committed only when they say
so. An import that wrote straight through would be a way past every rule the rest
of the system enforces — and the spreadsheet this application replaces is exactly
where the bad data comes from.

The **mapping template** is master data: which heading holds which field, or
which position when the sheet has no headings; the date format written as an
example of 2 January 2006; whether the numbers are `1,234.56` or `1.234,56`.
Getting that last one wrong turns a thousand tons into one, so it is a setting
rather than a guess made per cell. A template that could never work — a field
mapped twice, a required field not mapped, data starting above the headings — is
refused when it is saved, not when a file is uploaded.

Reading copes with what spreadsheets are really like: a byte order mark,
thousands separators and non-breaking spaces, accounting parentheses for a
negative, a currency symbol, a date column formatted as an Excel day serial, a
blank cell that must hold its place rather than shifting the next column left,
and a trailing row of nothing. `.xlsx` is read directly; only the first worksheet,
and that is said rather than left to be discovered.

Three things are reported that are not errors and matter anyway: columns of the
file the mapping ignores, mapped columns the file does not have — reported even
when a default covers them, because a renamed column looks exactly like an absent
one — and rows whose key already exists and would therefore be replaced.

The commit goes through the ordinary planning service, so an imported row meets
the same validation, period locking, authorisation and audit trail as a row
somebody types into the grid. The permission the commit will need is checked at
**upload**, so nobody reads a preview of figures they may not post. The default
is all or nothing.

The stock ledger imports movements and never balances: the opening and closing
balances are calculated forward from the movements, and a file that could set
them would let a spreadsheet contradict the ledger — which is the disagreement
this application exists to end.

---

## 7.7 Inbox

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| GET | `/notifications` | What has been raised for the caller | — |
| POST | `/notifications/{id}/read` | Mark one read | — |
| POST | `/notifications/evaluate` | Run the alert evaluation now | `plan:read` |

A notification is addressed to a **role at a factory**, not to a named person.
There is no user directory here — identity, roles and data scope all come from
the token — so somebody's inbox is what their roles and their scope entitle them
to see, and there is no permission for reading anybody else's because there is no
such thing. A notification addressed to a role the caller does not hold is *not
found* rather than forbidden.

The distinction from an alert is the point. An alert is calculated: it is true of
the plan at the moment somebody looks at the dashboard. A notification is a
record — this was raised, somebody was told, it has or has not been read — and it
survives the alert ceasing to be true.

An alert already raised within the last day is not raised again. Reading one
means "I know", not "remind me at the next tick", so the suppression counts read
notifications too; an alert still true a day later comes back, because one nobody
acted on should not be forgotten either.

Only warnings and errors are sent. An informational alert belongs on the
dashboard, and an inbox that fills with them is an inbox nobody reads — which
costs the warnings that do matter.

---

## 7.8 Interfaces

The two directions are deliberately different in kind. What leaves this system
goes through an outbox and is delivered asynchronously; what arrives comes in
through ordinary, synchronous endpoints that answer with what they did.

### Outbound: the transactional outbox

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| GET | `/integration/events` | The outbox, newest first | `integration:read` |
| POST | `/integration/events/{id}/retry` | Deliver one event now, whatever its backoff says | `integration:read` |
| POST | `/integration/dispatch` | Run a dispatcher pass on demand | `integration:read` |

An event is written **in the same transaction as the change it describes**. That
is the whole point: a confirmation that rolls back cannot leave a message
telling the ERP it happened, and an ERP that is down cannot cause a confirmation
to be refused. A dispatcher reads the outbox afterwards and delivers.

Eight topics are published: `plan.released`, `production.confirmed`,
`stock.posted`, `stock.reversed`, `quality.failed`, `quality.released`,
`shipment.dispatched` and `cost.run.completed`. Stock movements are emitted from
the single posting door every movement passes through, so a movement added later
cannot quietly stop telling anyone.

Delivery is **at least once**. A crash between delivering an event and recording
that it was delivered sends it again, which is why every envelope carries an `id`
to deduplicate on and the `X-Event-Id` header repeats it for a gateway that would
rather not parse the body. Exactly-once across two systems is not a promise this
or any other system can keep, and pretending otherwise would just move the
duplicate somewhere less visible.

Retries back off from one second, doubling to a ceiling of an hour. After 25
attempts an event stops being retried and waits for a person; it is never
discarded, and it keeps the last thing the far end said. `?exhausted=true` is the
list somebody should be looking at.

With no `INTEGRATION_ENDPOINT` configured the dispatcher publishes to the
application log. That is a real destination rather than a pretend one - the
events appear where the operator already looks - and pointing the variable at the
ERP is the only change needed to start feeding it.

### Inbound: the weighbridge

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| POST | `/integration/weighbridge` | Gate tickets become actual cane | `integration:write` |

A ticket carries the gross and the tare, not only the net, because a dispute
about a delivery is settled by the two weights and a system holding only their
difference could not settle it. Accepted tonnage is
`(gross − tare − rejected) / 1000`, and cane refused at the gate needs a reason
code, since that is what the grower is shown.

Tickets are **added** to the day and factory they name: the gate weighs a lorry
at a time, and each message carries what has arrived since the last one. The
crushed figure is never touched - it comes from the mill, and a gate reading is
not evidence about it.

The default is all or nothing, so a terminal that sent a bad batch resends the
batch rather than working out what got through; `partial: true` accepts the
sound tickets and reports the rest by row. Send an `Idempotency-Key`: a terminal
that lost the network mid-send retries, and the retry must not weigh the same
lorries twice.

### Inbound: the laboratory system

| Method | Path | Purpose | Permission |
| --- | --- | --- | --- |
| POST | `/integration/lab-results` | Instrument readings become quality results | `integration:write` |

The message names the sample by the number printed on the bottle and the
parameters by their codes, because that is what an instrument knows; asking a
laboratory technician to key a uuid would guarantee the interface went unused.

The readings are then judged by the ordinary quality service - the same
effective-dated specifications, the same verdict, the same hold on failing
material. An interface that could reach a different verdict from a technician
entering the same numbers would be worse than no interface at all.
`complete: false` saves an interim sheet without producing a verdict, which is
what an instrument reporting one parameter at a time needs.

### The machine account

Both inbound endpoints sit behind `integration:write`, held by the `INTEGRATION`
role and by nothing else. That account can read master data, record cane and
enter laboratory readings. It cannot move stock, release a hold or touch a plan.

---

## 7.9 What the API does not do

Worth stating so nobody looks for it:

- **No client-supplied query language.** Filters are named parameters on an
  allow-list. There is no `$filter` expression evaluator to escape from.
### Why there is no field selection

Section 16 asks for "pagination, filter, sorting, and field-selection rules".
Three of those exist; the fourth is a deliberate absence, and this is the rule.

A `$select` would let a client drop `rowVersion` from a response and then be
unable to write the record back, because `If-Match` needs it — a footgun that
looks like an optimisation. It would also make every response shape variable,
which the SAPUI5 client would then have to defend against, and it would put a
client-supplied column list into the SQL, which is exactly the class of thing
the allow-listed filtering above exists to avoid.

The payloads it would trim are small: a master-data row is a few hundred bytes,
and the one genuinely large response — the dashboard — is a computed aggregate
whose whole content is the point. Where a caller really does want less, there is
a narrower endpoint rather than a narrower projection.

- **No unbounded lists.** `$top` is capped at 1,000; daily-row queries are
  bounded by their date range.
- **No unknown fields.** A payload with a misspelled field is rejected rather
  than silently ignored, so a typo surfaces in development.
- **No partial financial postings.** A bulk write is all-or-nothing unless the
  caller explicitly asks for partial acceptance, which the planning grid never
  does.
