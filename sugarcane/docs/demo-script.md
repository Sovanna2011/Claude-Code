# Demo script

A twenty-minute walkthrough of the system against the seeded demo estate. It follows the
planning process in order, so each screen builds on the one before it rather than being a tour
of the menu.

Bring the stack up first and wait for every service to report healthy:

```bash
cd sugarcane/test-system
docker compose up -d --build          # then http://localhost:5150
```

The password is `Planner#2026` for every demo account. Sign in and out as the script says — the
navigation and the buttons change per role, and that is part of what is being shown.

## The numbers you should see

If these do not match, the tenant was seeded differently and the rest of the script will drift.

| Figure | Value |
|--------|-------|
| Projected area | 1,876.8 ha over 12 lines (1,251.2 new planting, 625.6 ratoon) |
| Expected production | 157,238 t after the loss percentage |
| Planted to date | 683.56 ha — 36.4% complete |
| Activity plans generated | 188 |
| Land | 3 farms · 6 zones · 24 blocks |
| Resources | 10 tractors · 14 implements · 10 operators · 3 crews · 8 materials |

## 1 · Where the estate stands — 2 min

**Sign in as `planner`.** The dashboard opens on the current season.

- Projected area, expected production, planted to date, overall completion.
- Monthly target versus actual: the first four months carry real progress.
- Resource requirement versus availability, graded per resource. Point out the tractor
  shortage — 16 required against 9 available — because stage 6 comes back to it.

## 2 · The land and the season — 2 min

*Enterprise & land* → the hierarchy from company down to block. *Plantation blocks* → the
plantable area, which is the hard cap on every projection line. *Growing seasons* → the planting
window that projection dates must fall inside. *Sugarcane varieties* → seed rate, growing period
and expected yield, which supply the defaults further down.

## 3 · The activity model — 2 min

*Activities & dependencies* → 19 activities with sequence, standard start-day offset, daily
capacity, duration and labour per hectare, and what each one needs (tractor, implement,
material, labour). Show one blocking dependency: planting cannot start until ploughing finishes,
plus its lag. **This is configuration, not code** — an estate that works differently changes it
here.

## 4 · The projection — 3 min

*Planting projections* → open the approved document. One line per block; the header totals are
derived from the lines and cannot be typed.

Worth demonstrating live, because it is the rule people ask about: edit a line and set the area
above the block's plantable area. The save is refused with `AREA_EXCEEDS_BLOCK`, and the same
check applies to the sum of every line on that block, not just the one being edited.

## 5 · The generated schedule — 3 min

*Activity plan & Gantt* → 188 plans, produced by applying each activity's offset to the planting
date, honouring dependencies and skipping non-working days. *Monthly / weekly plan* → the same
data rolled up into period targets.

## 6 · What the plan needs — 4 min

- *Tractors* and *Equipment & compatibility* — horsepower, daily capacity, and which implement
  fits which tractor.
- *Material requirements* — rate × applications plus waste, netted against stock, with the
  shortage per material.
- *Fuel & labour projection* — fuel by area and by hour; workers = ceil(labour-days ÷ days).

The tractor number is the specification's own worked example: 2,600 ha ÷ (4 ha/day × 90 days)
= 7.22, rounded up to **8 tractors**. Rounding up is the point — 7.22 tractors cannot plant.

## 7 · Capacity, scenarios and approval — 3 min

*Capacity & scenarios* → required against available, graded Sufficient / At risk / Shortage /
Unavailable. Build a scenario that adds tractors or widens the window and re-run it.

**Say plainly that the scenario has not touched the plan** — it is evaluated in memory, and the
approved document is untouched. Then *Approvals & revisions*: **sign in as `director`** and walk
Draft → Submitted → Under review → Approved. Revising an approved plan copies it, issues a new
version, and freezes the previous one read-only.

## 8 · Booking the resources — 3 min

**Sign in as `machinery`.** *Resource scheduling* → set the board to 24 Jan – 6 Feb 2026, where
the seeded bookings sit. The board defaults to the current fortnight, which is the right default
for an operations screen but empty for this historical data.

Then book a tractor that is already busy. The booking is refused with `SCHEDULE_CONFLICT`. Eight
checks run before anything is written: double-booking of tractor, implement or operator;
assignment during maintenance; insufficient horsepower; a machine stationed at another farm; an
unsatisfied dependency; and scheduling outside the approved plan period.

If the audience is technical, this is the moment for the strongest claim in the system: the
check and the insert happen inside one transaction holding a lock on the plan and on every
resource, so six simultaneous requests to book the same tractor produce one booking and five
refusals. `ConcurrencyTests` proves it against a real SQL Server.

## 9 · Actual versus plan — 2 min

**Sign in as `supervisor`.** *Projection vs actual* → area, fuel, labour and schedule variance
per block and activity, including the land-clearing job deliberately left at 45% with "Heavy
rain stopped work for four days" as its delay reason.

## 10 · Governance — 2 min

**Sign in as `admin`.** *Users & roles* → ten roles with different permission sets. *Audit log*
→ who changed what, when, from which address, with old and new values. *Reports* → 22 reports,
each with print preview and PDF or Excel export; export one to show the file name is real rather
than `report.pdf`.

## Questions that come up

**"Can we change the activities?"** Yes — they are master data, including the dependency chain,
the offsets and the per-hectare standards. Regenerate the activity plans afterwards.

**"What if two planners work at once?"** Committed plans cannot overlap on a block, and the rule
is re-checked when a document is submitted, under a lock. Of two drafts written at the same time,
exactly one can be submitted.

**"Does it work for more than one company?"** Every table is filtered by the signed-in user's
company, so no query can return another company's rows. Seasons are per company as well.

**"Can a supervisor override a dependency?"** No. Only a role holding the override permission
can, and the reason is recorded in the audit trail with the booking.
