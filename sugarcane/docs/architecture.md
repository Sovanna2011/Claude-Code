# Architecture

## Layers and dependency direction

```
┌──────────────────────────┐   HTTPS / JSON   ┌───────────────────────────┐   T-SQL   ┌──────────────┐
│  Blazor WebAssembly      │ ───────────────▶ │  ASP.NET Core Web API     │ ────────▶ │  SQL Server  │
│  Bootstrap · 18 screens  │ ◀─────────────── │  controllers · policies   │ ◀──────── │  planning    │
└──────────────────────────┘                  └───────────────────────────┘           └──────────────┘
        Presentation                                  Delivery                          Persistence
                                                          │
                                     ┌────────────────────┴────────────────────┐
                                     │            Application layer            │
                                     │  services · engines · mapping · rules   │
                                     └────────────────────┬────────────────────┘
                                                          │
                                     ┌────────────────────┴────────────────────┐
                                     │              Domain layer               │
                                     │  entities · enums · PlanningFormulas    │
                                     └─────────────────────────────────────────┘
```

Dependencies point inwards only:

| Project | References | Contains |
|---------|------------|----------|
| `Domain` | *(nothing)* | entities, enums, `PlanningFormulas`, domain exceptions |
| `Contracts` | Domain | DTOs shared verbatim by the API and the Blazor client, plus the role/policy map |
| `Application` | Domain, Contracts, EF Core *abstractions* | services, the five engines, mapping, `IAppDbContext`, `ICurrentUser`, `IDateTimeProvider` |
| `Infrastructure` | Application | `AppDbContext`, Fluent API configurations, migrations, Identity, audit interceptor, PDF/Excel exporters |
| `Api` | Infrastructure | controllers, JWT issuing, authorization policies, global exception middleware |
| `Client` | Contracts | Blazor components, typed API client, JWT auth state provider |

The application layer never sees `AppDbContext`; it depends on `IAppDbContext`, which is why
the integration tests can run the real services over an in-memory database.

## The five engines

| Engine | Service | What it does |
|--------|---------|--------------|
| Activity scheduling | `ActivityPlanService` | Turns each approved projection line into a chain of activity plans, honouring sequence, standard start-day offset, blocking dependency lag and the working calendar. |
| Conflict detection | `SchedulingService` | Eight checks before any booking is written (see below). |
| Material requirement | `MaterialRequirementService` | Resolves the most specific standard per activity, applies rate × applications + waste, nets against stock. |
| Capacity | `CapacityService` | Compares required versus available tractors, implements, workers, materials, daily hectares and the completion date. |
| Scenario | `ScenarioService` | Re-runs the capacity engine with levers applied in memory; the stored plan is never modified. |

### Conflict checks (section 10)

1. Tractor double-booking — overlapping time windows on the same machine
2. Equipment double-booking
3. Operator double-booking
4. Assignment during maintenance or breakdown
5. Insufficient tractor horsepower for the implement
6. Tractor or equipment location conflict (machine stationed at another farm)
7. Activity dependency not satisfied — overridable by a manager, with a recorded reason
8. Scheduling outside the approved plan period

The permitted plan period is the projection's planning window **widened by the activity
plan's own dates**, because land preparation legitimately runs before the planting window
via negative day offsets.

## Cross-cutting concerns

**Multi-tenancy.** Every entity implementing `ICompanyScoped` gets a global query filter
`!e.IsDeleted && (TenantCompanyId == null || e.CompanyId == TenantCompanyId)`. The tenant comes
from the `company_id` claim on the JWT, so no query can leak another company's rows.

**Soft deletion.** `BaseEntity.IsDeleted` plus the same global filter. Services refuse to
delete records still referenced by a plan or a booking.

**Optimistic concurrency.** Every table carries a SQL Server `rowversion`. Update DTOs carry
the token; `ServiceBase.ApplyConcurrencyToken` seeds it as the original value, so a stale save
raises `DbUpdateConcurrencyException`, which the middleware turns into `409 CONCURRENCY_CONFLICT`.
Providers other than SQL Server fall back to a plain concurrency token.

**Audit trail.** `AuditSaveChangesInterceptor` stamps created/modified fields and writes one
`AuditLog` row per changed entity with the old values, new values, changed columns, user,
timestamp, IP address and device. Workflow, export and login events are written explicitly by
`AuditService`.

**Transactions.** Multi-step operations (create projection with lines, revise, generate plans,
book a resource, submit for approval) open an explicit transaction through
`IAppDbContext.BeginTransactionAsync`. A nested call returns a no-op handle so the outer scope
keeps control of the commit.

**Locking the read-then-write rules.** Two rules cannot be expressed as a constraint, because
they are about *overlapping intervals*: "this tractor, implement or operator is not already
booked in this window" and "this block has no committed plan in this window". Both are
therefore read-then-write — query, decide, insert — and under simultaneous requests both callers
read "free" and both write. `IAppDbContext.LockAsync` takes an exclusive lock on each logical
key (`tractor:17`, `activity-plan:42`, `block:9`) for the life of the surrounding transaction,
so the check and the write are one step. Keys are always taken in ordinal order, which is what
rules out a deadlock between two bookings sharing one resource. On SQL Server the lock is
`sp_getapplock` with `@LockOwner = 'Transaction'`, so it is released by the commit or the
rollback and no code path can leak it; a caller that waits more than fifteen seconds is refused
with `RESOURCE_BUSY` rather than blocking the request thread indefinitely.

**Error handling.** `ExceptionHandlingMiddleware` maps `NotFoundException` → 404,
`ForbiddenException` → 403, `BusinessRuleException` → 422 with its stable code,
`DbUpdateConcurrencyException` → 409, everything else → 500 with a trace id. The client turns
the body back into `ApiException` so screens can show the real reason.

**Authorization.** Ten roles and seventeen policies (`perm:view`, `perm:approve`,
`perm:override-dependency`, …). `Policies.RoleMap` in `Contracts` is the single source of
truth, used to register the ASP.NET Core policies, to answer `ICurrentUser.HasPolicy` inside
services, and — because `Contracts` is shared verbatim — to filter the Blazor navigation from
the same map, so the menu and the endpoint cannot disagree about who may open a screen.

**Sign-in** is the one endpoint an unauthenticated caller may hit freely, so it is the one that
needs a rate limit of its own. Five failed attempts lock the account for fifteen minutes:
`AccessFailedAsync` records each failure, `IsLockedOutAsync` is consulted before the password is
even checked, and a success clears the counter. `UserManager.CheckPasswordAsync` does none of
that on its own, so configuring `MaxFailedAccessAttempts` without calling them leaves the
password guessable indefinitely — `LoginLockoutTests` is what keeps that from returning. An
unknown user and a wrong password answer identically so the endpoint cannot be used to
enumerate user names; a locked account is told plainly, since by then the name is already known
and the person needs to understand why their correct password stopped working.

The client is a convenience, never the control: it hides what a role cannot use, and the API
refuses it regardless. Note that the menu is filtered but the per-screen action buttons are not
— a Report Viewer opening *Approvals & revisions* still sees Approve and Reject, and learns
they are refused only when the API answers 403. Gating those controls is worth doing; the
security boundary does not depend on it.

All 123 endpoints declare the permission they need as an attribute, and `AuthorizationTests`
enforces that by reflection: an action that forgets its attribute is authenticated-only, so any
signed-in user could call it, and the test fails with its name. Three endpoints are allowed to
be authenticated-only, each for a stated reason — `auth/me` and `auth/change-password` are
self-scoped (both resolve the caller from their own token, and the password change requires the
current password), and the projection workflow endpoint needs a permission that depends on the
action in the request body, so `ProjectionService.ExecuteWorkflowAsync` checks it instead. That
check runs *before* the document is loaded: doing it afterwards would answer 404 rather than 403
for an unknown id, letting an unauthorized caller probe for valid ids. `AllowAnonymous` appears
on exactly one action, `auth/login`, and the test pins that list too.

## Request flow

```
Blazor page → PlanningApi (typed client) → ApiClient (bearer token, error translation)
   → controller (policy check) → application service (business rules, engines)
      → IAppDbContext → AppDbContext (query filters, interceptor) → SQL Server
```

## Testing strategy

`UnitTests` cover `PlanningFormulas` (including the specification's worked example
2,600 ÷ (4 × 90) = 7.22 → 8 tractors), the dependency-cycle guard, working-day arithmetic,
material-standard resolution, scenario levers and status derivation.

`IntegrationTests` build the real service graph over an isolated in-memory database and drive
the whole process of section 24 — including all eight conflict checks, the workflow state
machine, revision and version comparison, and every one of the 22 reports.

They also execute `DbSeeder.SeedTransactionsAsync` against a real service provider, so the
start-up seeding path is verified on a machine with no SQL Server.

`AuthorizationTests` checks the section-21 permission model from both ends: by reflection, that
every endpoint declares a policy and that every policy name exists in the role map; and
behaviourally, that a report viewer cannot submit, approve, reject, revise or close, that a
planner may submit but not approve, and that an approver may approve but not submit.

`SqlServerIntegrationTests` runs the same service graph against an actual SQL Server, creating
a throw-away database and applying the real migration to it. `ConcurrencyTests` goes further and
fires genuinely simultaneous requests through separate scopes — one connection each, exactly as
the API would — to prove the locking above. Set `SUGARCANE_TEST_SQLSERVER` to enable both;
without that variable the seventeen facts skip, so `dotnet test` stays green on a machine with
no server.

## What only a real database caught

These defects passed every in-memory test and failed on SQL Server. All are now covered by the
SQL Server suite.

**Retry-on-failure broke every transaction.** `EnableRetryOnFailure` installs
`SqlServerRetryingExecutionStrategy`, which refuses user-initiated transactions. Creating a
projection, revising one and generating an activity plan each open an explicit transaction, so
all three threw `InvalidOperationException` — and the API died during start-up seeding. Retry
is now off: making those blocks retriable safely would mean running the whole unit through the
execution strategy *and* rebuilding the change tracker per attempt, since a transient fault
would otherwise re-insert the rows the failed attempt already added. Correct transactions beat
transient retries; the trade-off is written down in `Infrastructure/DependencyInjection.cs`.

**The audit trail recorded `RecordId = 0` for every insert.** The interceptor builds its rows
during `SavingChanges`, before SQL Server has assigned the identity value — so section 22's
"record ID" was useless for creations, while updates and deletes were fine. In-memory keys are
assigned client-side, which is exactly why the defect was invisible there. The interceptor now
remembers each insert's audit row, fills in the key in `SavedChanges` and saves once more; the
second pass sees only `AuditLog` changes, which are never audited, so it cannot recurse.

**Exported reports all downloaded as `report.pdf`.** The API sets a descriptive
`Content-Disposition`, and every server-side test saw it — but the client runs on a different
origin, and a browser only reveals non-simple response headers listed in
`Access-Control-Expose-Headers`. The Blazor client therefore fell back to its default name, so
exporting several reports overwrote the same file. The CORS policy now exposes the header, and
`CorsPolicyTests` guards it.

**Double-booking was preventable one request at a time only.** Six simultaneous bookings of the
same tractor in the same hour all passed the conflict check and all six were stored. The eight
checks are read-then-write, and nothing held a lock between the read and the insert; the earlier
green test was passing for the wrong reason, because `CreateAsync` moves the activity plan from
Planned to Scheduled and the racers were colliding on *that* row's `rowversion` instead of on the
booking. Warm the plan up to Scheduled first and the guard vanished entirely. Bookings now
validate and insert inside one transaction holding a lock on the plan and on every resource, so
one caller wins and the rest get `SCHEDULE_CONFLICT`.

**Two projections could both take the same block.** The overlap rule ran only when a line was
entered, where two drafts on one block are deliberately legal — planners work in parallel. Since
nothing re-checked at the moment a document was *committed*, two drafts written before either was
submitted could both be submitted and both approved, double-booking the land. `Submit` now
re-runs the rule against the stored lines under a lock on each block.

**The published client never started.** Every browser check until now had run the dev server,
which serves assets under their plain names. `dotnet publish` fingerprints them — `dotnet.js`
becomes `dotnet.<hash>.js` — but the published `blazor.webassembly.js` still imports `dotnet.js`
and carries no fingerprint map, so the very first module import 404s and the app sits on
*Loading the planning workspace…* forever. It would have failed identically on IIS, nginx, a CDN
or Docker: everything except the one way it had been tested. `index.html` also hard-coded the
loader's own plain name, which publish had renamed. The client now publishes stable names and
`index.html` uses the `#[.{fingerprint}]` placeholder, and `test-system/` runs the published
output rather than the dev server, so a publish that does not boot fails visibly.

**The schema script could never be applied.** `sqlcmd` connects with `QUOTED_IDENTIFIER` OFF,
and SQL Server refuses to create filtered indexes in that state, so the documented
`sqlcmd -i database/01_schema.sql` created one table and stopped with *Msg 1934*.
`database/generate-schema.sh` now prepends the required `SET` options on every regeneration.
