# 14. User testing log

The section 27 acceptance criteria are checked by `cmd/acceptance`, which drives a
running instance over HTTP and answers in sixty-one pass or fail lines. That is the
machine's answer. It is not a person's.

This is the log a person works through: 28 cases across 9 areas, each naming the
account to sign in as, what to do, and what should happen. Every expected result was
read from a running instance rather than written from memory, so a case that does not
match is worth reporting rather than worth arguing about.

The log is published as a page that records pass, fail or blocked against each case,
keeps notes, and exports the whole thing as Markdown to paste into a report. It is held
in the tester's own browser and goes nowhere else, which the page says on it.

## The cases

### Signing in

| # | Signed in as | Case | Expected |
| --- | --- | --- | --- |
| A1 | Any account | Sign in with each account in turn and look at what appears. | Each account lands on the same shell but sees only the screens its roles allow. The account name and its roles are shown in the header throughout. |
| A2 | Sokha Planner | Try to record cane arriving at the weighbridge. | Refused. 403 — Planner requires the actual:cane permission |
| A3 | Rithy Weighbridge | Try to commit a cane source to the season. | Refused. 403 — Weighbridge requires the plan:write permission |
| A4 | Chanthou Warehouse | Try to enter a quality sample. | Refused. 403 — Warehouse requires the quality:write permission |
| A5 | Sokha Planner, then Sovann Auditor | Open the audit trail as each. | The planner is refused (403). The auditor sees it (200). |
### The season plan

| # | Signed in as | Case | Expected |
| --- | --- | --- | --- |
| B1 | Bopha Executive | Open the overview for season 2026-2027. | Cane target 2,300,000 t. Recovery target 11 %, actual 10.89 %. 4 capacity alerts open. |
| B2 | Sokha Planner | Open the plan version and read the crushing shape. | 137 campaign days, of which 131 crush and 6 are wash-outs. Full rate 19,000 t/day, season total 2,300,000 t. |
| B3 | Sokha Planner | Change the raw sugar recovery assumption and regenerate the plan. | The plan regenerates. Raw sugar and finished goods move with it; the cane target does not. |
| B4 | Sokha Planner | Add a product to the mix, then remove it again. | The line is saved and appears in the table. A message says the plan must be regenerated for the daily rows to follow. |
### Versions

| # | Signed in as | Case | Expected |
| --- | --- | --- | --- |
| C1 | Sokha Planner | Create a scenario from the budget with recovery at 9.5 %. | A new version is created and opens. It has no daily rows until it is generated. |
| C2 | Sokha Planner | Generate the scenario, then compare it with the baseline and the actuals. | Three columns: V1, UAT-V2, ACTUAL (actual). Only the assumption that was changed is listed: RAW_RECOVERY_PCT. |
| C3 | Sokha Planner | Change which column the differences are measured from. | The chosen column's own differences become zero and every other column is measured from it. |
| C4 | Bopha Executive | After the planner has created the scenario, reopen the overview. | Still the budget: cane target 2,300,000 t, unchanged by anything the planner is trying out. A scenario is never what the board reads. |
### Cane supply

| # | Signed in as | Case | Expected |
| --- | --- | --- | --- |
| D1 | Sokha Planner | Open the cane supply plan. | 8 sources committing 2,320,000 t against a 2,300,000 t target — 100.87 % coverage. |
| D2 | Sokha Planner | Read the warnings on the supply plan. | One warning, about haulage: CS-O01 must move 2432.258 t a day across its window; 190 trucks at 12 t carry 2280 t |
| D3 | Sokha Planner | Look at the delivery schedule against what arrived at the gate. | The scheduled bars run the whole season; arrivals are recorded for the first fortnight only, and deliberately do not match the schedule. |
### Recording the day

| # | Signed in as | Case | Expected |
| --- | --- | --- | --- |
| E1 | Rithy Weighbridge | Record a cane delivery: choose a source, a date, a tonnage and a lorry count. | Accepted (200). The row appears in the day's list. |
| E2 | Rithy Weighbridge | Record a delivery with a polarisation of 140 %. | Refused. 400 — polarisation is a percentage, between 0 and 100 |
| E3 | Vanna Shift Supervisor | Record a shift where 30 hours were lost to a stoppage. | Refused. 400 — stoppage hours cannot exceed the available hours |
| E4 | Vanna Shift Supervisor | Record a shift where delivered does not equal accepted plus rejected. | Accepted, and flagged as outside the mass-balance tolerance. |
| E5 | Vanna Shift Supervisor | Open the stoppage log, the production orders and the quality screens. | 7 stoppages, 5 production orders, 10 stock documents, 3 quality samples and 1 hold. |
### Approval

| # | Signed in as | Case | Expected |
| --- | --- | --- | --- |
| F1 | Sokha Planner | Create a draft, then submit it for approval. | Accepted (200). The version moves to IN_REVIEW. |
| F2 | Sokha Planner | Try to approve the plan you just submitted. | Refused (403). A planner cannot approve their own plan. |
| F3 | Dara Factory Manager | Approve the plan, then release it. | Both accepted (200, 200). |
| F4 | Sokha Planner | Try to regenerate the released plan. | Refused (409). A released baseline is locked; copy it into a scenario to change it. |
### Stock and capacity

| # | Signed in as | Case | Expected |
| --- | --- | --- | --- |
| G1 | Bopha Executive | Open stock and capacity and read the four stores. | FG-WH1 fills 2027-03-02 · FG-WH3 fills 2027-08-20 · RAW-WH1 fills 2027-03-23 · RAW-WH2 fills 2027-03-23 |
### Reports

| # | Signed in as | Case | Expected |
| --- | --- | --- | --- |
| G2 | Sokha Planner | Run a report and export it as CSV, Excel and PDF. | 15 reports are offered. Each exports in all three formats and carries the company, season and version in its header. |
| G3 | Sokha Planner | Import the mill's own production plan through the import screen. | 305 rows read, 304 accepted, 1 refused — the sheet's own SUM line, reported against file row 325. |
### Two factories

| # | Signed in as | Case | Expected |
| --- | --- | --- | --- |
| G4 | Sokha Planner | Check that one factory's planner cannot see the other's season. | Only the season for the factory the account is scoped to is listed. |

## Where the figures come from

The expected results are generated, not typed. Booting the demonstration and reading
the API gives the season's own figures — the crushing shape, the supply coverage, the
warehouse dates — and the refusals are the server's own words, captured by making the
request and recording what came back. When the plan changes, this log has to be
regenerated with it; a test log quoting last month's tonnage teaches a tester to
distrust the log rather than the system.
