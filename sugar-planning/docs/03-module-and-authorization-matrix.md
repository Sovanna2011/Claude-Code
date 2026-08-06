# 3. Module and authorisation matrix

The modules the system is divided into, the permissions each exposes, and which
role holds which permission.

This document and `internal/auth/auth.go` describe the same thing. The Go file
is the one the system enforces; if they ever disagree, the code wins and this
file is wrong.

---

## 3.1 Modules

The backend is a modular monolith. Each module owns its data and exposes a
service; nothing reaches around a service into another module's tables.

| Module | Package | Owns | Depends on |
| --- | --- | --- | --- |
| identity | `internal/auth` | Principal, roles, permissions, data scope, token verification | — |
| organization | `internal/store` (master data) | Companies, factories, areas, lines, stations, shifts, work calendar | identity |
| masterdata | `internal/store` (master data) | Products, units, conversions, packaging, warehouses, customers, channels, materials, reason codes | organization |
| seasons | `internal/service` (planning) | Seasons, plan versions, assumptions, product mix | organization, identity |
| planning | `internal/domain`, `internal/service` | Plan generation, daily cane/production/storage/shipment rows | seasons, masterdata |
| cane | within planning | Cane supply, crushing, utilisation | planning |
| production | within planning | Finished goods output, remelt, loss, rework | planning |
| inventory | within planning + `0003` schema | Stock ledger, balances, documents, reversals | planning, masterdata |
| shipment | within planning | Shipment plan and actual by channel | planning, masterdata |
| materials | `internal/service` (materials) | Packaging requirement planning | planning, masterdata |
| quality | `0004` schema | Specs, samples, results, holds | masterdata (phase 4) |
| downtime | `internal/store` (planning) | Downtime events, maintenance windows | organization |
| workflow | `internal/domain` (workflow) | Plan state machine, transitions, locking | seasons, identity |
| reporting | `internal/api`, `internal/report` | Report catalogue, builders, CSV/XLSX/PDF | everything read-only |
| costing | `0007` schema | Cost elements, rates, exchange rates, cost runs and variance | planning, masterdata |
| integration | `internal/integration`, `0005` schema | Transactional outbox, the dispatcher, and the weighbridge and laboratory adapters | planning, quality, inventory |
| audit | `internal/store` (audit) | Append-only audit trail | identity |

Boundaries are enforced by the repository interfaces: a service can only reach
what `store.Store` exposes, and the transport layer can only reach services.

---

## 3.2 Permissions

Permissions are verbs, not screens. A screen is allowed to show a button when
the caller holds the permission the backend will check.

| Permission | Allows |
| --- | --- |
| `masterdata:read` | Read any master data, and the value helps that use it |
| `masterdata:write` | Create, change and deactivate master data |
| `plan:read` | Read seasons, versions, daily rows, dashboards |
| `plan:write` | Create and edit versions, assumptions, mix and planned daily rows; generate; copy |
| `plan:submit` | Submit a plan for review, and recall it |
| `plan:approve` | Approve or reject a plan in review |
| `plan:release` | Release, supersede and close a plan |
| `plan:reopen` | Reopen an approved, released or closed plan |
| `actual:cane` | Post actual cane supply and crushing |
| `actual:production` | Post actual production |
| `actual:stock` | Post actual stock movements and counts |
| `actual:shipment` | Post actual shipments |
| `quality:write` | Record samples and results |
| `quality:release` | Release quality-held stock |
| `downtime:write` | Record downtime events and maintenance windows |
| `materials:read` | Read packaging requirement planning |
| `report:read` | Run and export reports |
| `audit:read` | Read the audit trail |
| `cost:read` | Read the cost structure, rates and cost runs |
| `cost:write` | Maintain cost elements, rates and exchange rates; save a run |
| `integration:read` | See the outbox, retry an event, run the dispatcher |
| `integration:write` | Feed the inbound interfaces: gate tickets and laboratory readings |
| `capacity:override` | Assign stock beyond usable capacity, recording the override |
| `admin` | Administration: users, roles, scopes, system settings |

---

## 3.3 The matrix

`R` = read, `W` = write, `A` = approve or authorise, `–` = no access.

| Role | Master data | Plans | Submit | Approve / release | Actual cane | Actual prod. | Actual stock | Actual ship. | Quality | Downtime | Materials | Reports | Audit | Admin |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| System Administrator | W | R | – | – | – | – | – | – | – | – | – | R | R | A |
| Master Data Administrator | W | R | – | – | – | – | – | – | – | – | – | R | – | – |
| Production Planner | R | W | ✔ | – | – | – | – | – | – | – | R | R | – | – |
| Cane / Weighbridge Operator | R | R | – | – | W | – | – | – | – | – | – | R | – | – |
| Shift Supervisor | R | R | – | – | W | W | – | – | – | W | – | R | – | – |
| Production Operator | R | R | – | – | – | W | – | – | – | – | – | R | – | – |
| Warehouse Operator | R | R | – | – | – | – | W | W | – | – | – | R | – | – |
| Quality / Laboratory | R | R | – | – | – | – | – | – | W + release | – | – | R | – | – |
| Maintenance | R | R | – | – | – | – | – | – | – | W | – | R | – | – |
| Sales / Shipment Planner | R | W | – | – | – | – | – | W | – | – | – | R | – | – |
| Finance / Cost Controller | R | R | – | – | – | – | – | – | – | – | – | R | – | – |
| Approver / Factory Manager | R | R | – | A + override | – | – | – | – | – | – | – | R | – | – |
| Executive Viewer | R | R | – | – | – | – | – | – | – | – | – | R | – | – |
| Auditor / Read Only | R | R | – | – | – | – | – | – | – | – | – | R | R | – |
| Interface (machine) | R | R | – | – | W | – | – | – | W | – | – | – | – | – |

### What the matrix is saying

**Least privilege.** A weighbridge operator can post cane weights and read the
plan they are working to. They cannot post production, touch stock, or edit a
plan. Each operator role holds exactly the actual-posting permission for its own
area.

**Separation of duties.** The Production Planner builds and submits; the
Approver approves and releases. Neither holds the other's permission, and the
service refuses an approval by the plan's own submitter even if somebody is
granted both roles.

**An interface is an account, not an exception.** The weighbridge terminal and
the laboratory system sign in as the `INTEGRATION` machine account, which holds
`integration:write` and the two posting permissions its readings need - and
nothing else. It cannot move stock, release a hold or touch a plan. An interface
running as an administrator is an interface nobody can safely change, and a
credential on a terminal in a yard is the one most likely to be copied.

**The administrator is not a superuser.** System Administrator holds `admin` and
master data, but *not* `plan:write`, `plan:approve` or any actual-posting
permission. Administering the system is a different job from running the
factory, and an administrator who needs to plan is granted the planner role
explicitly and appears as such in the audit trail.

**Read-only means read-only.** Auditor and Executive Viewer hold no write
permission at all. Auditor additionally holds `audit:read`, which nobody else
but the administrator does.

### How execution uses these permissions

Three places in the execution layer need a second permission on top of the one
that opens the screen, because the action costs somebody something:

| Action | Base permission | Additional | Why |
| --- | --- | --- | --- |
| Post beyond a store's usable capacity, or below zero stock | `actual:stock` | `capacity:override` | Both break a rule the ledger otherwise enforces, and both are recorded on the audit event with the override named |
| Technically close an order outside the variance tolerance | `actual:production` | `plan:approve`, and a configured reason code | An order that did not make what it was told to make is closed by somebody accountable for the explanation |
| Approve a maintenance window | `downtime:write` | `plan:approve` | An approved factory-wide window removes crushing days and moves the end of the season |

Two more follow the opposite principle - one permission is enough, because the
stock movement is a *consequence* of the act rather than a separate act:

- Confirming production receipts its own yield. A supervisor does not need the
  keeper's permission to put the sugar they just made into a shed.
- A failed quality sample blocks the material it covers. Blocking is part of
  failing it, not a warehouse operation.

Releasing a quality hold is deliberately a different permission
(`quality:release`) from placing one (`quality:write`), so a site that wants the
person who stops the sugar leaving to be a different person from the one who lets
it go can arrange that by granting the two to different roles. The shipped
Quality User role holds both; splitting them is a configuration decision, not a
code change.

---

## 3.4 Data scope

Permissions say *what* a user may do; the data scope says *where*.

Each user is granted companies and factories (`data_scopes`). Every service
checks the scope of the record it is about to return or change:

- `ListSeasons` filters out seasons in factories the caller cannot see.
- `GetSeason`, `GetVersion` and everything reached through them return 403 for a
  factory outside the scope — not 404, because the record does exist and
  pretending otherwise makes support harder.
- Every row of a bulk write is checked against the scope, so a payload cannot
  smuggle in a row for another factory.

**The scope fails closed.** A principal with no companies and no factories sees
nothing at all. That is why the demo profile explicitly grants its development
accounts the seeded factory: without it, they would correctly see an empty
system.

Row-level security in PostgreSQL is available as a second line of defence for
strict segregation (section 17), but service-layer authorisation is mandatory
and is what the tests assert.

---

## 3.5 Where each check happens

| Check | Where | Test |
| --- | --- | --- |
| Is the token valid? | `Authentication` middleware | `TestUnauthenticatedRequestsAreRejected` |
| Does the caller hold the permission? | Service entry, `Principal.Require` | `TestPermissionsAreEnforcedPerEndpoint` |
| Is the record in the caller's scope? | Service, `Principal.RequireFactory` | `TestDataScopeIsEnforcedOverHTTP`, `TestDataScopeHidesOtherFactories` |
| Is the workflow action legal here? | `domain.ApplyTransition` | `TestPlanTransitions` |
| Does this action need a reason? | `domain.ApplyTransition` | `TestWorkflowOverHTTP` |
| Is the submitter approving their own plan? | `Planning.Transition` | `TestPlanWorkflowEndToEnd` |
| Is the date locked? | `domain.CheckWritable` | `TestDateLockingAndWritability` |
| Is this series allowed in this version? | `domain.CheckWritable` | `TestActualsCannotBePostedToAPlanVersion` |
| Per-row permission on a bulk write | `Planning.checkRow` | `TestOperatorPermissionsAreEnforcedPerSeries` |

## 3.6 The one thing that needs no permission

Saved views carry no permission check, and that is a decision rather than an
omission.

A view belongs to whoever made it, the way a notification is addressed to a
role. There is no operation that reaches somebody else's, so there is nothing to
authorise beyond having signed in — and gating personalization behind a
permission would mean a role that can open a screen could not remember how they
like to look at it. An Executive Viewer, who may change nothing else in this
system, may still save the filter they read the board with.

What replaces the permission check is ownership, enforced in the store rather
than above it:

| Question | Answer |
| --- | --- |
| Whose views may I list? | Mine, plus shared ones from factories in my scope |
| Whose may I change or delete? | Only mine — somebody else's is `404`, not `403` |
| Which factory does a shared view reach? | The caller's, taken from their scope, never from the request |
| May an account scoped to several factories share? | No: a shared view is published to one factory, and there would be no way to say which |

`404` rather than `403` on somebody else's variant is the same shape as the
inbox: who holds which variants is not a question this system answers to a
caller.

---

Note what is *not* in that list: the UI. The SAPUI5 application hides actions
the caller cannot perform, which is good manners, but every one of those checks
runs again on the server. The API tests call the endpoints directly, with no
browser involved, precisely to prove that.
