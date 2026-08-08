# 11. The demonstration scenario

Kampong Speu Sugar, a fortnight into the 2026–2027 campaign.

```bash
./demo.sh
```

Go and nothing else — no database, no identity provider, no build step. The
scenario is deterministic: the same dates, quantities and failures every time,
so a demonstration can be scripted and a screenshot stays true. It is also
idempotent, so a container that restarts finds the data and leaves it alone
rather than doubling it.

---

## 11.1 How it is built, and why that matters

Everything in it goes through the **services**, never the store.

That is the whole point. A demonstration assembled by writing rows into tables
would skip the validation, the audit trail, the outbox and the permission
checks — and the first figure anybody questioned would turn out not to
reconcile. Driving the services means every row on every screen is a row the
system produced, by the same code path a person uses.

Three consequences you can check:

- The order that finished 60 t short **had** to be given a reason. The system
  refuses a silent close outside tolerance, so `VAR-CANE` on the variance report
  is a reason somebody actually had to supply.
- The stock movements balance because the posting engine balanced them, not
  because the numbers were chosen to.
- The audit trail has fifty-odd entries in it before anybody logs in, because
  building the demonstration *was* activity.

---

## 11.2 What is on each screen

| Screen | What the scenario put there |
| --- | --- |
| **Executive overview** | 137 days of plan, 14 of actuals, eight charts, and two real capacity alerts |
| **Season plans** | The budget V1, still a draft awaiting submission, and the actuals container |
| **Daily planning board** | Cane on 137 days, production and shipment across all 276 days of the campaign |
| **Cane supply** | 8 sources across 6 zones committing 2,320,000 t, a 492-row delivery schedule and 14 days of arrivals at the gate |
| **Downtime** | 7 stoppages across 4 reasons — the Pareto is 62 % boiler |
| **Production orders** | 5 orders, released and confirmed; one closed 60 t short |
| **Warehouse / Stock** | 10 documents: 5 goods receipts from the confirmations, a transfer, a stock correction and its reversal, and the quality hold and its release — because blocking stock is a movement, not a flag |
| **Quality** | 3 samples, 6 results, one failure that blocked 240 t and was then released |
| **Materials** | The packaging requirement for the whole season |
| **Costing** | A saved run, `RUN-W49-W50`, over the recorded fortnight |
| **Reports** | All 15, in CSV, Excel and PDF |
| **Inbox** | The capacity alerts, addressed to the roles whose job they are |
| **Import** | Two mapping templates, ready to stage a file against |
| **Audit trail** | Everything above, with actor, time and correlation id |

---

## 11.3 A walkthrough that shows the thing worth showing

Roughly fifteen minutes. Each step is chosen because it demonstrates something
that is hard to fake.

**1. Sign in as `planner`, open the Executive overview.**

Two capacity alerts, in red, before anything else on the page. They are not
decoration: FG-WH1 is forecast to fill on 1 January and FG-WH3 on 23 February,
and the system says what shipment rate would prevent it. That is a genuine
finding about the reference figures, recorded in
[01-assumptions-and-questions.md](01-assumptions-and-questions.md), not a
scripted warning.

Scroll through the charts. The daily trend has a notch on day 3 where the
factory ran at 61 % of target. Hold that thought.

**2. Click "View the daily rows" under Cane crushed.**

The board opens on the actual series, on the fortnight the tile was describing —
not on the season's first fortnight, and not on a filter you have to rebuild.
Every headline figure on the dashboard has this link, because a number nobody
can get behind is a number nobody can check.

**3. Go to Downtime.**

Seven stoppages. The Pareto above the list ranks them, and the boiler is 62 % of
the lost hours across three separate failures — one of which is the day 3 notch
you just saw. The two halves of the demonstration agree with each other because
they are the same data.

Narrow the date range: the ranking follows the filter, so "what stops this line"
and "what stops this factory" are different questions with different answers.

**4. Production orders → the one that reads TECHNICALLY_CLOSED.**

Planned 300 t, confirmed 240. Now try to imagine closing it silently — you
cannot, and neither could the demonstration: the system refuses a close outside
tolerance without a variance reason, *and* refuses to accept one from somebody
who does not hold the approval authority. It took a factory manager holding two
roles. That is separation of duties, and it is why the reason on the row means
something.

**5. Quality → the failed sample.**

Polarisation 99.640 against a floor of 99.700. Just outside — which is what a
real rejection looks like. It blocked 240 t of refined sugar in FG-WH1, and the
hold was released after a rework with the reason recorded. Open the certificate
on a passed sample: it prints with the limits the batch was judged against, not
the specification in force today.

**6. Sign out. Sign in as `executive`.**

Everything is readable and nothing is editable. Now try `auditor`: the audit
trail is there, and it is the only account that can see it. Try `warehouse`:
stock postings, and no way to touch the plan.

The screens hide what you cannot do, which is good manners. The *reason* you
cannot do it is that the server refuses — every API test in this repository
calls the endpoints directly, with no browser involved, to prove exactly that.

**7. Reports → Raw sugar recovery and mass balance.**

Fourteen days of cane crushed against raw sugar produced. The recovery drifts
between 10.7 % and 11.1 % against an 11.00 % assumption, and eleven of the
fourteen days fall outside the 0.5 % mass-balance tolerance. The three that do
not are the days the raw house happened to land on 11.00 % exactly — which is
the honest picture: two figures measured separately agree occasionally and
disagree the rest of the time, and a plan's tidy arithmetic is not what a
factory produces.

Export it as PDF. The header carries the company, factory, season, plan version,
generation time and who ran it, and so does the Excel and the CSV.

---

## 11.4 The figures, so they can be checked

| | |
| --- | --- |
| Cane target | 2,300,000 t over 137 campaign days, of which **131 crush** and 6 are wash-outs |
| Crushing curve | 17,000 t start-up, 19,000 t plateau (solved), half rate before each wash-out, run-down 15,000 → 3,000 t — the mill's own shape |
| Campaign | 1 Dec 2026 to 2 Sep 2027: cane to 16 April, refining and shipping to September |
| Cane committed | 2,320,000 t from 8 sources — 100.870 % coverage, 20,000 t of margin |
| Delivery schedule | 492 rows over all 137 days, 133,195 lorry movements |
| Recovery assumption | 11.00 %, so 253,000 t of raw sugar |
| Refinery rates | 400 t/day refined, 500 t/day white — the workbook's own, not an even spread |
| Finished goods | 106,700 refined + 133,400 white + 2,000 super = 242,100 t |
| Raw storage | 45,000 + 65,000 = 110,000 t |
| Finished storage | 22,000 + 47,000 = 69,000 t |
| Recorded actuals | 14 days, recovery drifting 10.7 % to 11.1 % |
| Downtime | 29 h across 7 events: boiler 18 h, rain 7 h, mill 2 h, power 2 h |
| Orders | 1,452 t planned, 1,403 t confirmed — 96.625 % |
| Quality | 6 results, 2 outside specification, 1 hold of 240 t placed and released |
| Mass balance | 11 of 14 days outside the 0.5 % tolerance |
| Cane delivered | 14 days from the estate block cutting that fortnight, 86,362.961 t against 86,921.744 t scheduled |

The three findings the reference figures themselves contain are in
[01-assumptions-and-questions.md](01-assumptions-and-questions.md) and are worth
raising in any demonstration, because they are the kind of thing this system
exists to surface: 242,100 t of finished goods needs 254,205 t of raw sugar at
the 1.05 remelt factor, against 253,000 t produced — a 1,205 t shortfall the
plan generator reports rather than papers over.

---

## 11.5 Against a real database

```bash
createdb sugarplan_demo
./demo.sh --postgres "postgres://localhost/sugarplan_demo"
```

It migrates first — a separate, observable step, as in production — and then
loads the same scenario. Or with Docker:

```bash
cd deploy && cp .env.example .env    # then set POSTGRES_PASSWORD
docker compose up --build
```

`SEED_EXECUTION=false` loads the plan without the factory life, for a
demonstration that is only about planning.

`SEED_DEMO` is refused outright when `APP_ENV=production`. Demonstration data
must never appear in a real plant, and that is a configuration error rather than
a warning.
