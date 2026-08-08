# 1. Assumptions and open questions

This is the first deliverable asked for in section 26: what was assumed, what
was decided, and what still needs a business answer.

---

## 1.1 The reference workbook

The specification asked for a workbook to be used as the business reference for
field mapping and reconciliation. It was not supplied at first, and for most of
this project everything under the season target was reasonable invention built
from the figures quoted in section 2.

**The workbook has since arrived**: `ProductionPlan_2627_2.3mt Rev.1
(corrected)`, prepared 16 July 2026, with a summary tab and a 305-row daily
sheet named `RW'2627(2.3mt)Rev1(re)`. Every headline figure the specification
quoted is confirmed by it. What it added is everything underneath them, and two
of those things changed the model rather than the data — see
[§1.2](#12-what-the-workbook-changed).

The stated figures, all confirmed:

| Figure | Value | Where it lives now |
| --- | --- | --- |
| Company | Kampong Speu Sugar Co., Ltd. | `companies` master record `KSS` |
| Season | 2026–2027 | `seasons.code` |
| Cane target | 2,300,000 t | assumption `CANE_TARGET_TONS` |
| Season length | 137 days | assumption `SEASON_DAYS` |
| Recovery | 11.00 % | assumption `RAW_RECOVERY_PCT` |
| Expected raw sugar | 253,000 t | calculated, C10 |
| Raw direct to refining | 124,950 t | assumption `RAW_DIRECT_TO_REFINE_PCT` = 49.387 % |
| Raw to storage | 128,050 t | calculated remainder |
| Raw warehouses | 45,000 t + 65,000 t = 110,000 t | `warehouses` RAW-WH1, RAW-WH2 |
| Jumbo packing | 300 t/day, 1.10 t/bag, 20,700 t | packaging `PJUMBO`, mix `dailyRateTons` |
| Refined / white / super refined | 106,700 / 133,400 / 2,000 t | `product_mix_entries` |
| Finished goods capacity | 22,000 t + 47,000 t = 69,000 t | `warehouses` FG-WH1, FG-WH3 |
| Quota shipment | 500 t/day | assumption `QUOTA_SHIPMENT_TPD` |

**The field mapping is now against the real file.** Two import templates,
`KSS-PLAN-CANE` and `KSS-ACTUAL-CANE`, read the daily sheet directly — column C
for the date, D for the day's target and E for what was crushed. They map by
*position*, not by heading, and that is deliberate: the sheet's headings are
split across rows 5 and 6, several are blank, and "R 50kg (ton)" appears twice,
once for the conditioning silo and once for Factory 1. Matching on a heading
that is not unique picks whichever column comes first, silently.

Run against the real file the importer reads **305 rows, 304 of them valid**.
The one rejection is the sheet's own `SUM` totals line, reported against file
row 325 so somebody can find it in Excel. `internal/seed/workbook_test.go`
holds the generated plan to the workbook figure for figure.

The specification also mentions the workbook tracks shipment in a column per
trader ("Wilmar, Jie Srey, You Hour"). Those are modelled as
`shipment_channels` master records rather than as columns, so a new trader is
master data maintenance, not a schema change. **The three names have not been
seeded**: they are real trading companies, and putting them into demonstration
data would misrepresent a commercial relationship. The seed uses `TRD-A`,
`TRD-B`, `TRD-C` instead.

---

## 1.2 What the workbook changed

Two things in it were not assumptions that turned out wrong — they were parts
of the model that did not exist.

### A crushing season is not a straight line

The generator spread the cane target evenly: 2,300,000 t over 137 days is
16,788 t every day, from the first to the last. The mill's own plan opens at
17,000 t while the boilers come up, settles at 19,000, drops to **half rate the
day before each wash-out**, stops for the wash-out itself, and runs down through
15,000, 12,000, 8,000, 4,000 and 3,000 t as the last cane arrives. Six wash-outs
inside the campaign, so **137 days of season carry 131 days of crushing**.

That is now a `CrushingProfile` on the plan version (C49, migration 0014). The
shoulders and the wash-out days are the planner's; the full rate is **solved**
from the target, so changing the tonnage gives a plan that still looks like a
season. Against 2,300,000 t over 137 days it solves to the workbook's own
19,000 t.

### The campaign is twice as long as the crushing season

The mill crushes to 16 April and goes on refining stored raw sugar, and
shipping to the quota, until **2 September**. The generator planned finished
goods over the crushing days alone — nine months of production compressed into
four and a half. Everything downstream of that was wrong by months, and two of
those wrong answers had been written up in this document as findings about the
business (below). They were findings about a loop.

`CAMPAIGN_DAYS` now separates the two. Cane rows exist on crushing days;
production, stock and shipment run the whole campaign.

---

## 1.3 The findings, corrected

Two of the three findings previously recorded here were artefacts of that
compressed horizon. They are kept rather than deleted, because a document that
quietly removes what it got wrong teaches nobody anything.

### Finding 1 — the plan is 1,205 t of raw sugar short

The finished goods mix totals 242,100 t. At the stated remelt input factor of
1.05 (section 7 of the specification), producing it needs

```
242,100 t × 1.05 = 254,205 t of raw sugar
```

The cane target and recovery assumption produce

```
2,300,000 t × 11.00 % = 253,000 t
```

a shortfall of **1,205 t**. The generator raises `REMELT_SUPPLY_SHORT` as an
error before the plan can be released, and names the four ways out: reduce the
finished goods mix, raise the cane target, raise the recovery assumption, or
plan an opening stock of raw sugar.

**The workbook confirms the 1.05 factor rather than dismissing it.** It does not
state one, but it computes one: the daily sheet feeds the refinery 945 t of raw
sugar to make 400 t of refined and 500 t of white, and 945 / 900 is exactly
1.05. So this finding stands, and it is a real one — the mill's own plan is
1,205 t of raw sugar short of the finished goods it schedules.

**Question for the business:** where does the 1,205 t come from — opening stock,
a slightly better recovery, or 1,205 t less finished sugar than planned?

### Finding 2 — the finished goods stores fill, but not when we said

**What this document used to say:** finished goods are produced at 242,100 t /
137 days ≈ 1,767 t/day against 500 t/day of shipment, so FG-WH1 fills on
1 January 2027 and 844 t/day would be needed instead.

**That was wrong, and the error was ours.** The mill does not make 1,767 t a
day. It makes 900 t a day — 400 refined and 500 white — for nine months. The
1,767 t figure was 242,100 t divided by the crushing season instead of by the
campaign, and the January date followed from it.

The stores still fill. The mill's own workbook says so in its key observations:
even at 500 t/day the finished stock passes 69,000 t of capacity on
**28 May 2027** and ends the campaign at about 106,100 t, some 37,100 t above
what there is room for. With no deliveries at all it would be full by
20 February. The plan now reaches the same conclusion from the same data, and
raises the capacity alerts across February to August rather than in the first
week of January.

The workbook also answers the rate question it left open: about **640 t/day**
keeps ending stock inside capacity, and about **890 t/day** clears the whole
campaign by 2 September.

**Question for the business, unchanged:** is 500 t/day the whole shipment plan
or only the domestic quota channel? 136,000 t over 272 selling days is 56 % of
what the season produces, and the other 44 % has to go somewhere.

---

## 1.4 Decisions taken, and why

Where the specification left a choice, this is what was decided. Each is
reversible through configuration unless noted.

### Planning and versions

| # | Decision | Reasoning |
| --- | --- | --- |
| A1 | Actuals live in a dedicated `ACTUAL` plan version, created automatically with the season. One per season, enforced by a partial unique index. | Section 17 requires that "actual posting never overwrites an approved plan". Separate containers make that structural rather than a rule somebody has to remember. |
| A2 | Rows carry an explicit `series` of `PLAN` or `ACTUAL` as well as living in a typed version. | Belt and braces: a defect in the service layer still cannot write an actual into a budget row, because the natural key differs. |
| A3 | At most one `RELEASED` version per season, enforced by a partial unique index. Releasing a second supersedes the first automatically. | "The released baseline" has to be singular or the dashboards become ambiguous. |
| A4 | A `WHATIF` version can never be submitted, approved or released. To adopt a simulation, copy it into a `REVISED` version. | Keeps simulations out of the approval trail entirely. |
| A5 | The submitter of a plan cannot approve it. | Separation of duties (section 19). Enforced in the service, not only in the UI. |
| A6 | Releasing takes an optional `lockThrough` date. Dates up to and including it become read-only; later dates stay editable. | A released season plan still has to accommodate re-planning of future months. An empty value locks the whole version. |

### Calculations

| # | Decision | Reasoning |
| --- | --- | --- |
| B1 | Every quantity is an exact decimal (`numeric` in PostgreSQL, `shopspring/decimal` in Go). Binary floating point is used nowhere. | Section 16. `0.1 + 0.2` must be `0.3`, and 2.3 M tonnes must not drift. |
| B2 | Season totals are split across days by rounding the running cumulative, not each day independently (`AllocateEvenly`, C31). | Rounding each of 137 days independently loses 23 kg of the cane target. The tests assert the parts sum back exactly. |
| B3 | Derived daily series use the same technique (`ScaleSeries`, C34). | Same reason: 137 days of `round(cane × 11 %)` sums to 252,999.955 t, not 253,000 t. |
| B4 | Rounding is half-away-from-zero at 3 decimals for quantities and rates, 6 for factors. | Matches how the business writes tonnages. Stated once, in `domain/decimalx.go`. |
| B5 | Division by zero returns zero everywhere, never an error or NaN. | Day 1 of a season has no cumulative target. Reporting "0 %" is correct; crashing is not. |
| B6 | The completion forecast uses the trailing 7-day average of *recorded* days, ignoring days with no actual. | A season that has not started must not forecast from a zero average. Window configurable per request. |
| B7 | A day with no recorded actual is flagged (`hasActual`), not treated as zero. | Otherwise the actual curve runs flat to the end of the season, which reads as "we stopped crushing". |

### Master data and inventory

| # | Decision | Reasoning |
| --- | --- | --- |
| C1 | Warehouses carry a nominal capacity and a usable percentage. All capacity checks use usable capacity. | A silo is never filled to its nominal rating. The seed sets 100 % so the demonstration reconciles with the workbook figures exactly; a real site sets 90–95 %. |
| C2 | The raw sugar receipt is shared between raw stores in proportion to usable capacity. | Deterministic, and keeps both stores at the same utilisation, which is what a capacity dashboard should show. A fill-one-then-the-next policy is a future option. |
| C3 | The generator never draws raw stock below zero; it caps the draw and reports the shortfall. | A plan that empties a silo past zero is not a plan. |
| C4 | `adjustment` is signed; every other movement is a magnitude. | One signed column makes a correction obvious in the ledger. The check constraints enforce it. |
| C5 | Master data is deactivated, never deleted. | Section 17. Referential history stays intact. |
| C6 | Stock balances are stored on the daily row and recalculated forward whenever a movement changes. | A season is 137 days across several stores; replaying the ledger on every read would not meet the p95 targets. |

### Execution

**A quantity is entered unsigned and the document type gives it its sign.** That
is how a warehouse keeper describes a movement — "issue 120", not "add minus
120" — and it removes a class of sign errors that no amount of validation
catches. Adjustments and counts are the exception, because a correction can go
either way and forcing a choice of document type by the sign of the difference
would be busy-work.

**A negative balance is refused by default and permitted with authorisation.**
Sites that book consumption before the matching receipt need it, and reversing a
receipt whose stock has since shipped needs it. The database constraint was
written to forbid negative stock outright, which contradicted this; migration
0006 relaxes it to say only what it should — nothing may be held that is not
there — and leaves the business decision to the domain, checked against the
caller's permission and recorded on the audit event.

**A confirmation receipts only its yield.** Scrap has left the process and rework
has not finished it. Recording either as stock would put sugar in a shed that
does not hold any. Both are still recorded on the confirmation, because they are
what the shift produced.

**Over-confirmation completes an order.** Factories make more than planned. The
alternative — refusing the confirmation or leaving the order open — would make
the operator lie about what they made.

**Orders come only from a released plan.** An order is an instruction to the
floor, and instructions do not come from a draft somebody is still editing. The
actuals container is refused as a source too, even though it is released from the
day it is created: it records what happened rather than what to make.

**A quality result keeps the limits it was judged against.** Copying the limits
onto the result rather than pointing at the specification is what lets somebody
tighten a limit next season without retrospectively failing this one. Where two
specifications for the same parameter are both in force, the later start date
wins — leaving it to the order the repository returned would make a verdict
depend on the storage engine.

**A measurement with no specification in force is reported, not passed.** It
judges nothing, and saying so is the point: an unspecified parameter is a
configuration gap, and silence is how a limit goes years without being set.

**Only approved, factory-wide maintenance removes a crushing day.** A window
somebody is still thinking about must not quietly move the end of the season, and
a line outage does not stop the mill. The season is extended rather than
shortened: the same cane still has to be crushed.

### Security and operations

| # | Decision | Reasoning |
| --- | --- | --- |
| D1 | The data scope fails closed: a principal with no companies and no factories sees nothing. | The alternative — "no scope means all access" — is the wrong default for a multi-company system. |
| D2 | No password is ever stored. Identity is delegated to the OIDC provider; the local `app_users` table holds only what authorisation and audit attribution need. | Section 19. |
| D3 | The development sign-in (`AUTH_MODE=dev`) is refused when `APP_ENV=production`, as are the in-memory store and demo seed. | Configuration validation fails at start-up rather than allowing an unsafe box to run. |
| D4 | The audit trail is append-only, enforced by PostgreSQL rules as well as by grants. | "Append-only for application users" (section 19) should not depend only on a `GRANT` somebody might change. |
| D5 | Exports are generated with no third-party library: `.xlsx` is a zip of XML, PDF is written directly. | An on-premises deployment should not carry licence questions for a report writer. Trade-off: the PDF uses the standard Helvetica fonts, so Khmer and Thai text needs an embedded font — see the open questions below. |

---

## 1.5 Open questions for the business

These materially affect the architecture or the numbers, which is the bar
section 26 sets. Each has a working assumption so that development is not
blocked.

| # | Question | Working assumption |
| --- | --- | --- |
| ~~Q1~~ | ~~Is the remelt input factor really 1.05?~~ **Answered by the workbook.** Its daily sheet feeds the refinery 945 t of raw sugar for 900 t of finished output, which is exactly 1.05. The 1,205 t shortfall is real and is finding 1. | Closed. The remaining question is where the 1,205 t comes from. |
| Q2 | Is 500 t/day the whole shipment plan or just the quota channel? Both finished goods stores overflow at that rate (finding 2). | Quota channel only; other channels are configured per site. |
| Q3 | What usable percentage applies to each store? Nominal capacity is rarely fillable. | 100 %, so the demonstration reconciles with the stated figures. Expect 90–95 % in reality. |
| ~~Q4~~ | ~~Is the 137-day season 137 calendar days or 137 crushing days?~~ **Answered, and it was neither reading.** 137 is the campaign, 1 December to 16 April; six of those days are wash-outs, so **131 are crushing days**. The working assumption said the reference plan had no shutdowns. It has six. | Closed. `SEASON_DAYS` is the campaign; the wash-outs are in the crushing profile. |
| ~~Q5~~ | ~~Is the 20,700 t of jumbo packing part of the 242,100 t, or raw sugar packed separately?~~ **Answered.** The workbook lists it under *Raw Sugar Allocation*, not under finished goods: raw sugar packed out of the silo at 300 t/day between 21 January and 30 March, to keep the peak stock inside 110,000 t. The working assumption was right. | Closed. Still not in the default mix — it is a raw-sugar draw, not a finished good, and the system does not yet model it as one. See Q15. |
| Q6 | What is the acceptable recovery operating window? | 9.8 % to 13.0 %, configurable as `RECOVERY_MIN_PCT` / `RECOVERY_MAX_PCT`. |
| Q7 | What mass-balance tolerance applies per process? | 0.5 % of throughput, configurable as `MASS_BALANCE_TOLERANCE_PCT`. |
| Q8 | Which shift pattern applies — three eight-hour shifts, or two twelve-hour? | Three eight-hour shifts (A/B/C) seeded. The model supports any pattern; daily rows may be shift-level or day-level. |
| Q9 | Are Khmer and Thai needed in printed PDF output, or only on screen? | Screen only for now. Both languages are **fully translated on screen** — 626 of 626 keys, held there by `frontend/test/i18n.test.js` — but the PDF writer uses the standard fonts and substitutes non-Latin characters. Embedding a Khmer font is a small, contained change when the answer is yes. |
| Q10 | Do the Khmer and Thai industry terms match this mill's house vocabulary? | The translations follow general Cambodian and Thai mill usage. A mill's own words for recovery, remelt, bagasse and the like often differ, and no test can check word choice. Worth an hour with the people who read these screens daily. |
| Q11 | Which system is the source of truth for cane weights — the weighbridge, or this system? | The weighbridge. The import and API path is designed for it (section 22); until it is connected, weights are entered by hand. |
| Q12 | How long must audit and transaction history be retained, and may it be partitioned by season? | 10 years (section 23). Partitioning is prepared for but not enabled; see [05-data-model.md](05-data-model.md). |
| Q13 | What margin does the mill contract above the crushing target, and what gap is worth an alert? | 2 % either way, as `SUPPLY_TOLERANCE_PCT`. The demonstration contracts 2,320,000 t against a 2,300,000 t target — 0.87 % over, which is inside the tolerance and says nothing. A shortfall is an error and a surplus only a warning, because a mill short of cane stops and a mill with too much leaves it standing. |
| Q14 | Are the cane sources, areas, yields and haulage in the demonstration anywhere near this mill's real ones? | They are **invented**, and no part of them came from the reference figures — which gave a season target and nothing behind it. Eight sources across six Kampong Speu districts, with areas and yields chosen so the commitments add up to the target and exactly one source is short of lorries. The shape is right; the numbers need replacing with the real contracts before anyone quotes them. |
| Q15 | The workbook keeps the raw silo inside capacity by packing 20,700 t into jumbo bags between 21 January and 30 March. The plan reproduces the overflow but not the remedy: jumbo packing is a repack out of raw storage, and the mix entry mechanism only makes finished goods. Should it be modelled as a repack? | Yes, and it is not built. The plan raises the raw-storage capacity alerts the workbook raises, and stops there. Until it is modelled, the peak reads about 129,000 t rather than the workbook's 109,720 t. |
| Q16 | The workbook idles the refinery until 5 December and then ramps it — 350, 400, then 500 t/day — where this plan starts it on day one at full rate. Is that four-day lag a commissioning constraint or a choice? | Treated as a choice and not modelled. It is the cause of the two known differences from the workbook: refinery input during crushing reads 131,565 t against 124,950 t, and the campaign finishes 24 August rather than 2 September. |

---

## 1.6 What is built, and what is not

The specification describes five phases. This delivery covers **all five**.
Nothing in the delivered scope is a placeholder: there are no stub methods, no
fake success responses and no unexplained TODOs.

**Built and tested**

- Identity, authorisation, data scoping, audit trail
- Organisation and master data (15 entity types, full CRUD, effective dating)
- Seasons, plan versions, assumptions, product mix, the full workflow
- Plan generation from assumptions, scenario copy, version comparison
- Daily cane, production, stock ledger and shipment planning, plan and actual
- Capacity forecasting, required-shipment-rate calculation, alerting
- Packaging material requirement planning
- Cane supply planning: sources, season commitments reconciled against the cane
  target, a generated delivery schedule and the arrivals recorded against it
- Executive dashboard and nine reports, exported to CSV, Excel and PDF
- PostgreSQL schema for every table group, with reversible migrations

- Inventory documents and reversals, production orders and confirmations,
  quality samples, results and holds, and the maintenance calendar — with their
  screens
- Downtime recording, which feeds the dashboard's lost-tonnage figure
- Costing: elements with drivers, effective-dated standard and actual rates,
  multi-currency, cost per ton, and a variance split that always reconciles
- The transactional outbox, its dispatcher, and the weighbridge and laboratory
  adapters, with the machine account they run as
- Background jobs behind a database lease, so a pair of instances share the
  recurring work rather than duplicating it
- Certificate of analysis, and the packaging bill of materials that drives both
  requirement planning and the components a confirmation consumes

- The controlled import of section 22: mapping templates, a staging area, a
  preview with row-level errors, and a commit that goes through the ordinary
  planning service
- Background jobs behind a database lease: the outbox dispatcher and the alert
  evaluation that fills people's inboxes

**Partly built: the Excel migration**

The framework is here and works: a site defines a mapping template — which
column holds which field, how the dates and numbers in it are written — uploads a
`.csv` or `.xlsx`, reads a preview with every problem attached to the line of the
file it is on, and commits.

What is *not* here is a pre-canned mapping for the reference workbook
`5_6296369270787941755.xlsx`, because that file was never provided. Its sheet
names, header rows and units are unknown, and a column mapping written against a
file nobody has seen would be guesswork dressed as progress. Defining it once the
workbook exists is a screen's worth of data entry, not a code change.

**Not built: the rest**

Advanced forecasting (seasonality, weather, cane maturity), SAPUI5 unit and OPA5
tests, and operational hardening at ten years of data. None is a gap in the
specification's core scope; each is listed under "Still open" in the
implementation plan with an estimate.

See [10-implementation-plan.md](10-implementation-plan.md) for the phase plan
with acceptance criteria.
