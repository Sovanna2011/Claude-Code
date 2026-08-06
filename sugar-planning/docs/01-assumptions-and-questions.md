# 1. Assumptions and open questions

This is the first deliverable asked for in section 26: what was assumed, what
was decided, and what still needs a business answer.

---

## 1.1 The reference workbook was not available

The specification says:

> Use the attached workbook "5_6296369270787941755.xlsx" as a business reference
> for field mapping and reconciliation.

**That file was not provided.** Only the specification text reached the
development environment. Everything in this system that refers to the workbook
is therefore built from the figures quoted in section 2 of the specification
itself, which are reproduced exactly:

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

**What this means for the field-to-domain mapping.** Section 26 asks for a
field mapping against the workbook before coding. The mapping in
[05-data-model.md](05-data-model.md) is therefore a mapping of the *stated
business content*, not of the workbook's actual column headers. It has to be
reviewed against the real file before the first data migration. The reference
reconciliation figures are covered by automated tests
(`TestReferenceScenarioReconciliation`), so once the workbook arrives, any
disagreement will show up as a failing test rather than as a surprise in UAT.

The specification also mentions the workbook tracks shipment in a column per
trader ("Wilmar, Jie Srey, You Hour"). Those are modelled as
`shipment_channels` master records rather than as columns, so a new trader is
master data maintenance, not a schema change. **The three names have not been
seeded**: they are real trading companies, and putting them into demonstration
data would misrepresent a commercial relationship. The seed uses `TRD-A`,
`TRD-B`, `TRD-C` instead.

---

## 1.2 Two findings in the reference figures

These came out of building the calculation chain, and both need a business
decision. Neither is a defect in the software: the system reports them, which
is the point.

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

**Question for the business:** which is intended? The most likely explanation is
that 1.05 is an illustrative figure in the specification rather than the
site's actual factor — at a factor of 1.0450 or below the plan balances.

### Finding 2 — 500 t/day of shipment cannot keep up with production

Finished goods are produced at 242,100 t / 137 days ≈ **1,767 t/day**. The
planned quota shipment is **500 t/day**. The finished goods stores hold 69,000 t
between them, so they fill during the campaign: FG-WH1 on 1 January 2027 and
FG-WH3 on 23 February 2027.

The system calculates what would be needed instead: **844 t/day** for FG-WH1 and
**470 t/day** for FG-WH3 to stay inside the alert threshold across the plan.

**Question for the business:** is the 500 t/day quota rate the *whole* shipment
plan, or only the domestic quota channel, with export and direct sales tracked
separately? The data model supports any number of channels; the seed only
populates the quota channel, because that is the only rate the specification
states.

---

## 1.3 Decisions taken, and why

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

### Security and operations

| # | Decision | Reasoning |
| --- | --- | --- |
| D1 | The data scope fails closed: a principal with no companies and no factories sees nothing. | The alternative — "no scope means all access" — is the wrong default for a multi-company system. |
| D2 | No password is ever stored. Identity is delegated to the OIDC provider; the local `app_users` table holds only what authorisation and audit attribution need. | Section 19. |
| D3 | The development sign-in (`AUTH_MODE=dev`) is refused when `APP_ENV=production`, as are the in-memory store and demo seed. | Configuration validation fails at start-up rather than allowing an unsafe box to run. |
| D4 | The audit trail is append-only, enforced by PostgreSQL rules as well as by grants. | "Append-only for application users" (section 19) should not depend only on a `GRANT` somebody might change. |
| D5 | Exports are generated with no third-party library: `.xlsx` is a zip of XML, PDF is written directly. | An on-premises deployment should not carry licence questions for a report writer. Trade-off: the PDF uses the standard Helvetica fonts, so Khmer and Thai text needs an embedded font — see the open questions below. |

---

## 1.4 Open questions for the business

These materially affect the architecture or the numbers, which is the bar
section 26 sets. Each has a working assumption so that development is not
blocked.

| # | Question | Working assumption |
| --- | --- | --- |
| Q1 | Is the remelt input factor really 1.05? At 1.05 the reference plan is 1,205 t short (finding 1). | 1.05 as stated, with the shortfall reported as an error. |
| Q2 | Is 500 t/day the whole shipment plan or just the quota channel? Both finished goods stores overflow at that rate (finding 2). | Quota channel only; other channels are configured per site. |
| Q3 | What usable percentage applies to each store? Nominal capacity is rarely fillable. | 100 %, so the demonstration reconciles with the stated figures. Expect 90–95 % in reality. |
| Q4 | Is the 137-day season 137 *calendar* days or 137 *crushing* days? | Crushing days. Maintenance windows extend the campaign rather than cutting tonnage. The reference plan has no shutdowns, so both readings give 1 December to 16 April. |
| Q5 | Should the 20,700 t of jumbo packing come out of the 242,100 t of finished goods, or is it raw sugar packed for direct sale? | Raw sugar packed for sale, i.e. *additional* to the 242,100 t. It is seeded as a separate mix entry in the tests but not in the default demo mix, because including it would change the finished goods total the specification states. |
| Q6 | What is the acceptable recovery operating window? | 9.8 % to 13.0 %, configurable as `RECOVERY_MIN_PCT` / `RECOVERY_MAX_PCT`. |
| Q7 | What mass-balance tolerance applies per process? | 0.5 % of throughput, configurable as `MASS_BALANCE_TOLERANCE_PCT`. |
| Q8 | Which shift pattern applies — three eight-hour shifts, or two twelve-hour? | Three eight-hour shifts (A/B/C) seeded. The model supports any pattern; daily rows may be shift-level or day-level. |
| Q9 | Are Khmer and Thai needed in printed PDF output, or only on screen? | Screen only for now. The UI is fully translatable; the PDF writer uses the standard fonts and substitutes non-Latin characters. Embedding a Khmer font is a small, contained change when the answer is yes. |
| Q10 | Which system is the source of truth for cane weights — the weighbridge, or this system? | The weighbridge. The import and API path is designed for it (section 22); until it is connected, weights are entered by hand. |
| Q11 | How long must audit and transaction history be retained, and may it be partitioned by season? | 10 years (section 23). Partitioning is prepared for but not enabled; see [05-data-model.md](05-data-model.md). |

---

## 1.5 What is built, and what is not

The specification describes five phases. This delivery covers **phases 1 to 3**
in full, plus the complete data model for phases 4 and 5. Nothing in the
delivered scope is a placeholder: there are no stub methods, no fake success
responses and no unexplained TODOs.

**Built and tested**

- Identity, authorisation, data scoping, audit trail
- Organisation and master data (14 entity types, full CRUD, effective dating)
- Seasons, plan versions, assumptions, product mix, the full workflow
- Plan generation from assumptions, scenario copy, version comparison
- Daily cane, production, stock ledger and shipment planning, plan and actual
- Capacity forecasting, required-shipment-rate calculation, alerting
- Packaging material requirement planning
- Executive dashboard and nine reports, exported to CSV, Excel and PDF
- PostgreSQL schema for every table group, with reversible migrations

**Data model present, application logic scheduled for phase 4**

Production orders and confirmations, inventory documents and reversals, quality
samples/results/holds, and maintenance windows all have their tables, constraints
and indexes in the migrations, and the domain enumerations exist. Their services
and screens are phase 4. Downtime is the exception: it is recorded and it feeds
the dashboard's lost-tonnage figure today.

**Not started**

Costing (section 21) and the external integration adapters (section 22) are
phase 5. The transactional outbox table exists so that integration events can be
written from day one without a schema change.

See [10-implementation-plan.md](10-implementation-plan.md) for the phase plan
with acceptance criteria.
