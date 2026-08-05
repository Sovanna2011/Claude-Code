# Calculations

Every formula lives once, in `SugarcanePlanning.Domain.Calculations.PlanningFormulas`, so the
API, the engines, the reports and the tests cannot drift apart. All quantities are rounded to
four decimals (1 m² on a hectare), away from zero.

## Projection (section 5)

| Figure | Formula | Example |
|--------|---------|---------|
| Harvestable area | `projected area × (1 − loss % / 100)` | 80 ha at 5 % loss → **76 ha** |
| Expected cane production | `harvestable area × yield per ha` | 76 ha × 90 t/ha → **6,840 t** |

Header totals are the sum over the live lines; they are never entered by hand.

## Activity planning (section 7)

| Figure | Formula | Example |
|--------|---------|---------|
| Working days | inclusive day count, skipping the configured rest days and holidays | 2 Mar–31 Mar without Sundays → **26 days** |
| Daily target | `planned area ÷ available working days` | 80 ha ÷ 26 → **3.0769 ha/day** |
| Planned machine hours | `planned area × standard hours per ha` | 80 ha × 2.2 → **176 h** |
| Activity duration | `ceil(area ÷ standard capacity per day)`, minimum one day | 100 ha ÷ 4 → **25 days** |

An activity's start is `line planting start + standard start-day offset`, pushed out where a
blocking predecessor ends later: `predecessor end + lag + 1`.

## Machinery requirement (sections 8, 9)

```
Required tractors = ceil( planned area ÷ (capacity per tractor per day × available working days) )
```

The specification's worked example: `2,600 ha ÷ (4 ha/day × 90 days) = 7.22` → **8 tractors**.
The exact ratio is reported next to the rounded figure so planners can see how close the call
was. Equipment uses the same arithmetic per implement category.

## Material requirement (sections 12, 13)

| Figure | Formula |
|--------|---------|
| Base requirement | `planned area × standard rate per ha × number of applications` |
| Waste quantity | `base requirement × waste % / 100` |
| Total requirement | `base + waste` |
| Required seed cane | `new planting area × seed rate per ha` (from the variety) |
| Required chemical | `treatment area × application rate × number of applications` |
| Net available | `available stock + incoming − reserved` |
| Shortage | `max(total requirement − net available, 0)` |
| Surplus | `max(net available − total requirement, 0)` |

Worked example — 80 ha of NPK at 350 kg/ha with 2 % waste:
`80 × 350 = 28,000 kg` base, `+ 560 kg` waste, **28,560 kg** total. With 10,000 in stock,
5,000 incoming and 500 reserved the net available is 14,500 kg, so the shortage is 14,060 kg.

**Which standard applies.** Among the rows effective on the activity's start date that match
the crop type, variety and soil type, the most specific wins: crop type scores 4, variety 2,
soil type 1. Ties break on the later `EffectiveFrom`. One row per material.

Required delivery date = activity start − `Planning:MaterialDeliveryLeadDays` (default 7).

## Fuel and labor (section 14)

| Figure | Formula |
|--------|---------|
| Projected fuel by area | `planned area × litres per hectare` |
| Projected fuel by hour | `planned working hours × litres per hour` |
| Required labor-days | `planned area × standard labor-days per ha` |
| Required workers | `ceil(required labor-days ÷ available working days)` |

Procurement uses the larger of the two fuel figures. Where tractors are already booked the
hourly figure comes from those machines; otherwise it uses the active-fleet average.

## Capacity (section 15)

```
coverage = available ÷ required

available ≥ required                       → Sufficient
available = 0 (with a requirement)         → Unavailable
coverage  ≥ AtRiskThreshold (default 0.90) → At Risk
otherwise                                  → Shortage
```

The overall status of an analysis is the worst line. Scenario levers are applied in memory:

| Lever | Effect |
|-------|--------|
| Additional rental tractors / equipment / workers | raises the available count |
| Extended working hours | scales daily machine capacity by `(standard hours + extra) ÷ standard hours` |
| Reduced planting area | scales every required quantity by `(1 − % / 100)` |
| Changed planting dates | shifts the period, changing the working-day count |
| Changed activity duration | extends or shortens the period end |

## Projection versus actual (section 17)

| Figure | Formula |
|--------|---------|
| Area variance | `actual area − planned area` |
| Material variance | `actual material − planned material` |
| Fuel variance | `actual fuel − planned fuel` |
| Schedule variance | `actual completion date − planned completion date`, in days |
| Completion % | `actual completed area ÷ planned area × 100` |

Plan status is derived: 100 % with a completion date → **Completed**; started and past the
planned end → **Delayed**; started and still inside the window → **In Progress**.

## Unit conversion (section 11)

`base quantity = alternative quantity × conversion factor` — 10 bags × 50 kg = 500 kg. The
inverse divides. A factor of zero or less is rejected.
