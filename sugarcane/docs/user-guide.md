# User guide

The 21 screens of the application, in the order a planning season actually runs.

## Signing in

Open the client, choose a demo account (or type your own credentials) and sign in. The menu
adapts to your roles; a screen you may not use tells you so instead of failing silently.

---

## Master data

### 1. Enterprise and land structure — `/land-structure`

Five tabs. **Tree** shows company → estate → farm → zone → block with the area of each node.
The other tabs maintain companies, estates, farms and zones with search, paging and an inline
editor. A record still in use cannot be deleted — the message tells you what is holding it.

### 2. Growing seasons — `/seasons`

A season fixes the planting and harvest windows. The planting window must sit inside the
season, and a **Closed** season accepts no further changes. Projection lines are validated
against the planting window.

### 3. Sugarcane varieties — `/varieties`

Seed rate per hectare drives the seed-cane requirement. Expected yield and loss percentage are
copied onto new projection lines as defaults. The recommended planting window may wrap across
the year end (for example October to February).

### 4. Plantation blocks — `/blocks`

The **plantable area** is the hard cap on everything planned for the block. Soil type decides
which activity material standard applies. Plantable area may not exceed the total area.

### 5. Block locations — `/block-map`

Every block that has a latitude and longitude, plotted on a map: coloured by how far its activity
programme has got — complete, in progress, planned, not planned — and sized by plantable area, so
the biggest fields read as the biggest circles. Choosing a marker, or a row in the list beneath
it, opens the block's detail alongside: farm and zone, plantable area, soil and irrigation, its
programme percentage and its coordinates. **Open in Google Maps** hands the coordinates to Google
in a new tab.

Coordinates are edited on the *Plantation blocks* screen; blocks without them are not plotted and
are counted in a badge at the top so they are not silently missing.

With `GoogleMaps:ApiKey` set in `wwwroot/appsettings.json` the page shows a real Google satellite
map. Without a key — or on an estate network with no route to `maps.googleapis.com` — it draws the
same blocks from the stored coordinates and says why, and **Retry** tries the map again. The
Google Maps links work either way; they are only URLs.

### 6. Activities and dependencies — `/activities`

**Activities** tab: the configurable activity master — sequence, category, crop type, standard
start-day offset, capacity per hour and per day, hours per hectare, labor-days per hectare and
the four *requires* flags that tell the engines what to calculate.

**Dependencies** tab: "activity X waits for activity Y", with an optional lag in days. A
blocking dependency stops scheduling; an advisory one only warns. A dependency that would
close a loop is rejected.

### 7. Tractor master — `/tractors`

Daily capacity feeds the requirement formula. Availability and the maintenance window are
enforced when booking. A tractor with live bookings cannot be deleted, and it cannot be put
into maintenance while bookings fall inside that window.

### 8. Equipment and compatibility — `/equipment`

Implements record the minimum tractor horsepower they need. On the **Tractor compatibility**
tab you pair machines explicitly; once any pairing exists for an implement, only listed
tractors may pull it. A pairing below the minimum horsepower is refused outright.

### 9. Materials and standards — `/materials`

**Materials**: code, category, base and alternative unit with the conversion factor, standard
rate and the min/max application band.

**Activity standards**: what an activity consumes, optionally narrowed to a crop type, variety
or soil type. The most specific effective row wins. Two equally specific rows may not overlap
in time.

**Stock**: read-only positions delivered by the ERP through `POST /api/materials/stock/sync`.

### 10. Operators and work teams — `/workforce`

The workforce the labor projection counts and the scheduler books. Operators carry a skill,
an optional licence with an expiry (shown in red once it has passed), a farm and a crew.
Work teams carry a supervisor, a primary skill and a head count that is kept in step with
the active members automatically. An operator or team with live bookings cannot be deleted.

---

## Planning

### 11. Planting projections — `/projections`

Create a projection for an estate and season; the number is issued automatically
(`PP-<season>-0001`). Open it to add one line per block: crop type, variety, area, planting
window, yield and loss. The screen previews the harvestable area and expected production
before you save.

Refused, with the reason shown:

- area greater than the block's plantable area (per line and summed per block)
- planting dates outside the season's planting window
- another line or another approved projection already covering that block in that period

The **Workflow** panel offers only the actions legal in the current status:

```
Draft ──Submit──▶ Submitted ──Review──▶ Under review ──Approve──▶ Approved ──Close──▶ Closed
   ▲                   │                      │                       │
   └──Return───────────┴──────────────────────┘                       └──Revise──▶ new version
                       └──Reject──▶ Rejected ──Submit──▶ …
```

Rejecting requires a comment. **Revise** copies the approved version, issues version *n+1* and
freezes the previous one read-only.

### 12. Monthly and weekly plan — `/period-plan`

Planting targets per month or ISO week against what has been completed, plus the area
breakdown by farm, zone or block.

### 13. Activity plan and Gantt — `/activity-plan`

Choose a projection and press **Generate activity plan**. Each line becomes a chain of
activities honouring sequence, day offsets and blocking dependencies; the derived working
days, daily target, machine hours, tractor count, workers and fuel are calculated for each.
Warnings appear when an activity would finish after the plan period.

Three views: **Gantt** (bars coloured by status, completion shaded in), **List** (editable
dates, area, supervisor and status) and **Calendar**.

> Regenerating is blocked while live bookings exist — cancel them first, so no schedule is
> silently orphaned.

### 14. Resource scheduling — `/scheduling`

Book a tractor, implement, operator or crew against an activity plan. **Check conflicts** runs
all eight checks without saving; a blocking conflict stops the save and says why.

An activity-dependency conflict can be overridden by a manager holding the override
permission, who must give a reason — recorded on the booking and in the audit trail.

The board groups by day, tractor, equipment, operator, farm or block, with utilisation per
resource.

### 15. Material requirements — `/material-requirements`

Consolidated requirement versus availability, grouped by material, activity, block, farm,
month or variety. Every row shows base, waste, total, stock, reserved, incoming, net
available, shortage, surplus, the required delivery date and a status. **Recalculate from
plan** refreshes the stored rows after the activity plan changes.

### 16. Fuel and labor projection — `/fuel-labor`

Fuel by area and by hour side by side — procurement uses the larger. Labor shows required
labor-days, required workers, available workers and the gap.

---

## Analysis

### 17. Capacity and scenarios — `/capacity`

Required versus available for tractors, each equipment category, the workforce, every
material, the daily hectare rate and the completion date, each with a coverage bar, a status
and a recommendation.

Below it, **what-if scenarios**: add rental tractors, add equipment, extend working hours,
reduce the planting area, shift planting dates, change activity durations or add workers.
**Simulate** shows baseline versus scenario side by side. The approved plan is never modified.

### 18. Approvals and revision history — `/approvals`

The version chain of a projection, the approval history of the selected version and a
field-by-field comparison of any two versions, marking each difference Added, Removed or
Changed.

### 19. Projection versus actual — `/projection-vs-actual`

**Record actual progress** captures actual dates, completed area, machine and operator used,
hours, fuel, labor-days, material consumption and a delay reason. Variances and the completion
percentage are derived, and the activity status becomes Completed, In Progress or Delayed.

The comparison groups by block, farm, activity or month.

### 20. Reports and audit log — `/reports`, `/audit`

All 22 reports share one surface: pick the report and the filters, sort by clicking any
column, then **Print preview**, **Export PDF** or **Export Excel**.

| | | |
|---|---|---|
| Planting projection | Monthly plan | Farm / zone / block plan |
| New planting and ratoon | Activity schedule | Activity calendar |
| Tractor requirement | Tractor utilization | Tractor shortage |
| Equipment requirement | Equipment utilization | Equipment shortage |
| Seed cane | Fertilizer | Chemicals |
| Material shortage | Fuel | Labor |
| Capacity analysis | Delayed activities | Revision comparison |
| Projection versus actual | | |

The **audit log** (System Administrator) filters by user, table, record, action and date, and
expands to the full old/new value JSON with the IP address and device.

### 21. Users and roles — `/users`

A System Administrator creates users, assigns any of the ten roles and sets the company that
scopes everything they can reach. Every signed-in user can change their own password here and
see exactly which roles and permissions they hold — the panel on the right works even for
users who may not list the others.

---

## Dashboard — `/`

KPI cards for projected area, new planting versus ratoon, expected production, planted to
date and overall completion; monthly targets versus actual; resource gaps; material
requirement by category; delayed activities and blocks at risk.

## Roles at a glance

| Role | Can do |
|------|--------|
| System Administrator | everything, plus user administration and the audit log |
| Plantation Director | approve, reject, close, view everything |
| Plantation Manager | approve, reject, revise, override dependencies, manage master data |
| Farm Manager | create and submit projections, schedule, record actuals |
| Agricultural Planner | master data, projections, activity plans, submit, revise |
| Machinery Manager | tractor and equipment master, scheduling |
| Material Planner | material master, standards, material requirements |
| Field Supervisor | scheduling and actual progress |
| Management Approver | approve and reject |
| Report Viewer | view and export only |
