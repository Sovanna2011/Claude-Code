# 6. Calculation catalogue

Every calculation the system performs: the formula, its unit, its rounding, the
fields it reads and a worked example.

Each entry names the Go function that implements it. The functions live in
`backend/internal/domain/calc.go`, `generator.go` and `costing.go`, and each
carries the same formula in its doc comment. The tests in `calc_test.go`,
`generator_test.go` and `costing_test.go` assert the examples below.

---

## Numeric contract

Before the formulas, the rules that apply to all of them.

| Rule | Value |
| --- | --- |
| Number type | Exact decimal end to end. `numeric` in PostgreSQL, `shopspring/decimal` in Go. Binary floating point is used in no quantity path. |
| Quantity scale | 3 decimals (tonnes to the kilogram) |
| Rate scale | 3 decimals |
| Percentage scale | 3 decimals |
| Factor scale | 6 decimals |
| Rounding | Half away from zero, applied **once**, at the end of a calculation |
| Division by zero | Returns zero. Never an error, never NaN. |
| Season splits | Round the running cumulative, not each day (C31–C34) |

That last rule matters more than it looks. Rounding each of 137 days
independently loses tonnage:

```
2,300,000 / 137          = 16,788.321167... t/day
round to 3 dp            = 16,788.321 t
× 137                    = 2,299,999.977 t     ← 23 kg missing
```

`AllocateEvenly` rounds the cumulative instead, so the parts add back to exactly
2,300,000 t while no day differs from the naive figure by more than 1 kg.

---

## C1–C7 Target versus actual

### C1 Cumulative target
```
cumulative target(d) = cumulative target(d-1) + daily target(d)
```
Unit: tonnes, scale 3. Source: `daily_cane_plans.cane_crushed`, series `PLAN`.
Go: `BuildSeries`.

### C2 Cumulative actual
```
cumulative actual(d) = cumulative actual(d-1) + daily actual(d)
```
Days with no recorded actual contribute nothing **and are flagged**
(`hasActual = false`), so the actual curve stops at today instead of running
flat to the end of the season. Go: `BuildSeries`.

### C3 Daily variance
```
variance = actual - target
```
Positive is ahead of plan. Unit: tonnes, scale 3. Go: `Variance`.

### C4 Achievement percentage
```
achievement % = 100 × cumulative actual / cumulative target
```
Zero-safe. Unit: percent, scale 3. Go: `AchievementPct`.

*Example.* After three days: actual 49,288 t, target 50,364 t →
49,288 / 50,364 × 100 = 97.8635533… → **97.864 %**.

### C5 Remaining
```
remaining = max(0, season target - cumulative actual)
```
Unit: tonnes, scale 3. Go: `Remaining`.

### C6 Rolling average
```
rolling average(d) = mean of the actuals in the last `window` days that have one
```
Default window 7 days, configurable per request. Days without an actual are
excluded rather than counted as zero, so a season that has not started does not
report a zero average. Unit: tonnes/day, scale 3. Go: `RollingAverage`.

### C7 Forecast completion date
```
days remaining  = ceil(remaining / daily rate)
completion date = from + days remaining
```
Returns "not forecastable" when the rate is zero or negative, because a campaign
that is not moving never completes and a date would be a lie. Go:
`ForecastCompletion`.

*Example.* 20,001 t remaining at 1,000 t/day from 1 December 2026 →
ceil(20.001) = 21 days → **22 December 2026**.

---

## C8–C12 Crushing and recovery

### C8 Effective crushing capacity
```
effective capacity = rate per hour × max(0, available hours - stoppage hours)
```
Unit: tonnes, scale 3. Go: `EffectiveCrushCapacity`.

*Example.* 750 t/h over 24 h with 2.5 h stopped → 750 × 21.5 = **16,125.000 t**.

### C9 Utilisation
```
utilisation % = 100 × (available hours - stoppage hours) / available hours
```
Stoppage beyond the shift floors at zero rather than going negative. Go:
`UtilisationPct`.

### C10 Expected raw sugar
```
expected raw sugar = cane crushed × recovery % / 100
```
Unit: tonnes, scale 3. Go: `ExpectedRawSugar`.

*Reference reconciliation.* 2,300,000 t × 11.00 % = **253,000 t**.

### C11 Actual recovery
```
actual recovery % = 100 × raw sugar produced / cane crushed
```
Zero-safe: no cane crushed reports 0 %, not a division error. Go:
`ActualRecoveryPct`.

### C12 Recovery verdict
```
recovery < 0 or > 100                → ERROR
recovery outside [min, max]          → WARNING
otherwise                            → SUCCESS
```
Window from `RECOVERY_MIN_PCT` / `RECOVERY_MAX_PCT`, defaulting to the target
± 1 point when unset. Go: `RecoveryVerdict`.

---

## C13–C21 Stock ledger, capacity and shipment

### C13 Beginning balance
```
beginning balance(d) = ending balance(d-1)
```
The first day starts from the warehouse's opening balance. Continuity is
asserted for every date by tests at three layers. Go: `RollLedger`.

### C14 Ending balance
```
ending balance = beginning balance
               + production receipt + transfer in + repack in
               + adjustment
               - remelt issue - shipment - transfer out - repack out
               - process loss
```
`adjustment` is signed; every other movement is a magnitude, enforced by check
constraints. Unit: tonnes, scale 3. Go: `RollLedger`.

### C15 Available balance
```
available balance = ending balance - quantity on quality hold
```
Held stock counts towards capacity but cannot be shipped or consumed. Go:
`RollLedger`.

### C16 Capacity use
```
capacity use % = 100 × ending balance / usable capacity
usable capacity = nominal capacity × usable % / 100
```
Always measured against *usable* capacity, never nominal. Go: `CapacityUsePct`,
`Warehouse.UsableCapacity`.

### C17 First breach date
```
first breach = the earliest day where ending balance ≥ limit
```
The limit is usable capacity × threshold (80 %, 90 % or 100 %). Go:
`FirstBreach`.

### C18 Required daily shipment
The smallest constant daily rate that keeps the stock at or below the limit on
**every** day of the horizon.

For each day *i* (1-based), the balance without shipment is
`B(i) = opening + Σ receipts up to i`. With a constant rate *s* the balance is
`B(i) - i·s`, so `B(i) - i·s ≤ limit` gives

```
s ≥ (B(i) - limit) / i        for every i
s = max over i of that, floored at 0
```

Unit: tonnes/day, scale 3. Go: `RequiredDailyShipment`.

*Why the maximum and not the average.* An early peak binds harder than a late
one. Opening 70,000 t, a 50,000 t receipt on day 1 and nothing after, limit
110,000 t: day 1 needs (120,000 − 110,000)/1 = 10,000 t/day, while by day 4 the
same excess spread over four days would suggest 2,500 t/day. Ship 2,500 t and
the store overflows on day one. The answer is 10,000 t/day.

### C19 Shipment to clear by a date
```
rate = (opening + total receipts - target balance) / days
```
Floored at zero. Unit: tonnes/day, scale 3. Go: `ShipmentToClearBy`.

### C20 Mass-balance difference
```
difference = calculated ending balance - recorded physical balance
```
Reported only for days that carry a physical count. Go: `MassBalanceDiff`.

### C21 Tolerance test
```
within tolerance = |difference| ≤ |base| × tolerance % / 100
```
The base is the period's throughput, not the closing balance, so a small store
with heavy turnover is not judged too harshly. Default 0.5 %
(`MASS_BALANCE_TOLERANCE_PCT`). Go: `WithinTolerance`.

---

## C22–C27 Conversions and packaging

### C22 Tons to units
```
units = tons × 1,000 / net weight in kg
```
Go: `TonsToUnits`.

### C23 Required packages
```
packages = ceil(tons × 1,000 / net weight kg × (1 + scrap % / 100))
```
Rounded **up**: a fraction of a bag cannot be issued. Unit: pieces. Go:
`RequiredPackages`.

*Reference reconciliation.* 20,700 t in 1.10 t jumbo bags, no scrap:
20,700 × 1,000 / 1,100 = 18,818.18… → **18,819 bags**.

### C24 Purchase requirement
```
purchase = max(0, gross requirement + safety stock - on hand - on order)
```
Unit: pieces, scale 3. Go: `PurchaseRequirement`.

### C25 Suggested order date
```
suggested order date = required-by date - lead time days
```
Go: `SuggestedOrderDate`.

### C26 Unit conversion
```
quantity in target unit = quantity × factor
```
Factors are effective-dated master data, per product or global. Go:
`ConvertQty`.

### C27 Remelt input
```
raw sugar input = finished goods output × remelt input factor
```
The factor is an effective-dated assumption (`REMELT_INPUT_FACTOR`, 1.05 in the
reference scenario) and is shown next to any figure derived from it. Go:
`RemeltInput`.

---

## C28–C30 Downtime

### C28 Availability
```
availability % = 100 × (available hours - downtime hours) / available hours
```
Go: `AvailabilityPct`.

### C29 Lost tonnage
```
lost tons = downtime hours × rated throughput of the line
```
Go: `LostTons`.

### C30 Throughput against rating
```
throughput % = 100 × (tons / run hours) / rated tons per hour
```
Go: `ThroughputPct`.

---

## C31–C34 Allocation

These preserve totals when a season figure is spread across days. They are the
reason the reference reconciliation is exact rather than nearly right.

### C31 Even allocation
```
cumulative(i) = round(total × i / n)
part(i)       = cumulative(i) - cumulative(i-1)
```
Guarantees `Σ parts = round(total)`. Go: `AllocateEvenly`.

### C32 Proportional allocation
```
cumulative(i) = round(total × Σ weights up to i / Σ all weights)
part(i)       = cumulative(i) - cumulative(i-1)
```
Zero total weight falls back to an even split. Used to share a raw sugar receipt
between stores in proportion to usable capacity. Go: `AllocateProportional`.

### C33 Allocation at a fixed rate
```
part(i) = min(daily rate, remaining)
```
Runs at the rate until the total is used up; later days are zero. Returns the
shortfall when the horizon is too short. This is how jumbo bag packing is
planned: 300 t/day until 20,700 t, which is 69 days. Go: `AllocateAtRate`.

### C34 Scaled series
```
cumulative out(i) = round(cumulative in(i) × factor)
part(i)           = cumulative out(i) - cumulative out(i-1)
```
Guarantees `Σ ScaleSeries(daily, f) = round(Σ daily × f)`. Used for raw sugar
from cane and for remelt input from finished goods. Go: `ScaleSeries`.

*Why it exists.* 137 days of `round(16,788.321 × 0.11)` sums to 252,999.955 t.
`ScaleSeries` gives exactly 253,000.000 t, which is the figure the business
recognises.

---

## C35–C42 Costing

All of costing rests on one idea: a cost is a **rate** multiplied by a **driver
quantity**. Keeping the two apart is what makes the difference from plan
decomposable — when a cost moves, it moved because the rate changed or because
the quantity changed, and a controller has to be able to say which.

Money is scale 2 and unit rates are scale 6. A rate of $18.500000 per ton of
cane is a real figure; rounding it to cents before multiplying by 2.3 million
tons would move the season total by thousands of dollars.

### C35 Driver quantity
```
CANE_TON     -> tons of cane crushed
SUGAR_TON    -> tons of finished sugar
RUN_HOUR     -> hours the mill ran
CALENDAR_DAY -> days of the campaign, whether it ran or not
FIXED_SEASON -> 1
```
A lump sum for the season has a driver quantity of one, so the same
`rate × quantity` arithmetic covers it without a special case. Go:
`DriverQuantities.Quantity`.

### C36 Element cost
```
cost = rate × driver quantity
```
Unit: money, scale 2. Go: `ElementCost`.

### C37 Total cost
```
total = Σ element costs
```
Written out rather than derived from a driver: a season's cost is what its
elements add up to and nothing else.

### C38 Unit cost
```
unit cost = total cost / output tons
```
Zero output gives zero rather than an error. At the start of a season the cost
is real and the output is not yet, and a dashboard should say `0.00` rather than
fail. Go: `UnitCost`.

### C39 Rate variance
```
rate variance = (actual rate - standard rate) × actual quantity
```
The part of the difference caused by paying a different price. Positive is
unfavourable: it cost more than the plan allowed. Go: `RateVariance`.

### C40 Usage variance
```
usage variance = (actual quantity - planned quantity) × standard rate
```
The part caused by using a different quantity. The **standard** rate is
deliberately the one used: valuing the extra quantity at the actual rate would
count the price difference twice, and the two halves would no longer add up.
Go: `UsageVariance`.

### C41 Variance reconciliation
```
rate variance + usage variance = actual cost - planned cost
```
This is asserted, not assumed. A run that cannot reconcile raises
`COST_VARIANCE_UNRECONCILED` at error severity rather than reporting figures
that do not add up.

In practice a run does not compute C40 directly. It rounds the rate half — the
half a controller checks against an invoice, which has to match the paperwork to
the cent — and takes the usage half as the remainder, so the two always
reconstruct the total exactly. The residual is at most one cent and is the
conventional accounting treatment. Go: `SplitVariance`, `VarianceCheck`.

### C42 Currency conversion
```
converted = amount × rate(from -> to)
```
A conversion to the same currency is the identity and needs no rate, so a
single-currency site configures nothing. An inverse quotation is as good as a
direct one: a site that entered USD→KHR need not also enter KHR→USD. Go:
`Convert`.

---

## C43 Bill-of-materials component

```
quantity = packages × qty per package × (1 + component scrap % / 100)
```

The package count comes from C23, which uses the **bag's** scrap rate: a bag
that tears is a bag that had a liner in it. The component then applies its
**own** scrap rate, because a liner that tears one time in fifty wastes liners,
not bags, and charging the bag's rate to the thread would quietly misstate both.

Unit: the component's own unit, scale 3. Go: `ComponentQuantity`.

*Worked example.* 330 t of sugar in 1.10 t jumbo bags is 300 bags; the bag's
1 % scrap makes it 303. One liner per bag at the liner's own 1 % scrap is
306.030 liners. Thread at 0.004 spools per bag with 3 % scrap is 1.248 spools.

---

## Reference reconciliation

The figures section 24 requires, and the tests that assert them.

| Reconciliation | Expected | Test |
| --- | --- | --- |
| 2,300,000 × 11.00 % | 253,000 t | `TestReferenceScenarioReconciliation` |
| Raw capacity 45,000 + 65,000 | 110,000 t | same |
| Finished capacity 22,000 + 47,000 | 69,000 t | same |
| Finished goods 106,700 + 133,400 + 2,000 | 242,100 t | same |
| Raw split 124,950 + 128,050 | 253,000 t | same |
| 20,700 t in 1.10 t bags | 18,819 bags | same |
| 20,700 t at 300 t/day | 69 days | same |
| Daily balance continuity, every date | holds | `TestRollLedgerContinuity`, `TestGenerateLedgerContinuityForEveryDate`, `TestGeneratedLedgerBalancesAreStoredAndContinuous` |
| Season allocation sums back to the target | exact | `TestAllocateEvenlyPreservesTotal`, `TestScaleSeriesPreservesTheRoundedTotal` |
| End-to-end generated plan | 2,300,000 t / 253,000 t / 242,100 t over 137 days | `TestSeedProducesTheReferenceScenario` |

---

## Alerts derived from these calculations

| Code | Raised when | Severity |
| --- | --- | --- |
| `CAPACITY_WARNING` | Planned stock crosses the warning threshold (default 80 %) | Warning |
| `CAPACITY_EXCEEDED` | Planned stock reaches usable capacity | Error |
| `CRUSHING_BEHIND_SCHEDULE` | C7 forecasts completion after the planned end date | Warning, Error beyond 7 days |
| `RECOVERY_OUT_OF_RANGE` | C12 returns a warning or error verdict | as C12 |
| `REMELT_SUPPLY_SHORT` | Refining demand exceeds the planned raw sugar supply | Error |
| `MIX_RATE_TOO_LOW` | C33 cannot deliver the season tonnage at the configured rate | Warning |

Every threshold is an assumption on the plan version, not a constant in the code.
