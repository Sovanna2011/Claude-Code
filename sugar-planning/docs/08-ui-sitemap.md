# 8. SAPUI5 sitemap, page specifications and wireframes

The application is an original Fiori-style design. It borrows the interaction
patterns the specification names — list report, object page, filter bar,
semantic colouring — and copies no SAP screen, layout or code.

---

## 8.1 Sitemap

```mermaid
flowchart TD
    L[Sign in] --> H[Launchpad]
    H --> O[Executive overview]
    H --> P[Season plans]
    P --> PD[Plan detail]
    PD --> B[Daily planning board]
    P --> B
    H --> W[Warehouse and silo]
    H --> ST[Stock]
    H --> PO[Production orders]
    H --> Q[Quality]
    H --> MN[Maintenance]
    H --> S[Shipments]
    H --> M[Materials]
    H --> R[Reports]
    H --> C[Costing]
    H --> MD[Master data]
    H --> I[Interfaces]
    H --> A[Audit trail]
```

Routes are hash-based and deep-linkable: `#/plans/{versionId}` opens a plan,
`#/board/{versionId}` opens its board.

### The shell

A tool header carrying the application name, the **season selector** and the
user menu; a side navigation for the areas; a page container.

The season selector is the application's context. Changing it raises one event
and every open page reloads against the new season — which is also how pages
recover when they were mounted before sign-in finished.

---

## 8.2 Pages

### Sign in

Selects a development account and signs in. The account list is fetched from the
server rather than kept in the browser, so an account added to the configuration
and forgotten in the page cannot happen — which it once did: the quality user was
configured on the server and missing from the page, and the laboratory role could
not be demonstrated at all. In an OIDC deployment the list comes back empty and
the button redirects to the identity provider; nothing else changes.

### Launchpad

Five KPI tiles answering "is the season on track today" — cane crushed,
achievement, recovery, forecast completion, open alerts — above twelve
navigation tiles. Each KPI tile is coloured by the same semantic rules used
everywhere else and drills into the overview.

### Executive overview

```
┌──────────────────────────────────────────────────────────────────┐
│ Executive overview                    Plan version [V1  ▾]   ⟳   │
├──────────────────────────────────────────────────────────────────┤
│ Exceptions                                            2 open     │
│  ⊗ Refined/White Warehouse 1 runs out of space     Jan 01, 2027  │
│  ⊗ Refined/White Warehouse 3 runs out of space     Feb 23, 2027  │
├──────────────────────────────────────────────────────────────────┤
│ Cane crushed    │ Cane remaining │ Forecast compl. │ Recovery     │
│ 226,474 t       │ 2,073,526 t    │ Apr 15, 2027    │ 10.89 %      │
│ of 235,036 t    │ at 17,100 t/day│ planned Apr 16  │ target 11.00 │
│ ▓▓▓▓▓▓▓░ 96.4 % │                │ 1 day early     │              │
├──────────────────────────────────────────────────────────────────┤
│ Cumulative cane, target against actual                           │
│   2.30 Mt ┤                                    ╱ ─ ─ ─ ─         │
│   1.15 Mt ┤                      ╱ ─ ─                           │
│        0 ┤━━━●                                                   │
│           2026-12-01        2027-02-07        2027-04-16         │
│           ─ ─ target    ── actual                                │
├──────────────────────────────────────────────────────────────────┤
│ Storage capacity                                                 │
│ Warehouse   Capacity  Usable   Closing   Use      First full     │
│ FG-WH1        22,000  22,000    97,090  441.3 %   Jan 01, 2027   │
├──────────────────────────────────────────────────────────────────┤
│ Finished goods by product     │  Shipment by channel             │
└──────────────────────────────────────────────────────────────────┘
```

**Exceptions come first.** A dashboard that leads with green numbers and buries
the problem is decoration. Alerts sort worst-first.

The chart is inline SVG generated in the browser. A charting library would be a
large dependency for two line charts, and an on-premises install has to serve
every asset itself. The actual line stops at the last recorded day rather than
running flat to April.

### Season plans

A list report over the season's versions: code, type, status, owner, lock date,
approver. Two actions in the header: **compare versions** and **new scenario**.

The scenario dialog is the what-if entry point: pick a source, give the copy a
code, optionally override the recovery assumption, and choose whether to carry
the daily rows across. The comparison dialog leads with *which assumptions
differ*, because that is usually the explanation for the tonnage difference
underneath it.

### Plan detail

An object page: workflow actions, assumptions, product mix, and what the last
generate run found.

- **Workflow actions** are rendered from `allowedActions`, which the server
  computes from the status, the plan type *and* the caller's permissions. A
  button the backend would refuse is never drawn.
- **Assumptions** are editable in place and save on change, because a planner
  adjusting a rate expects the next generate to use it.
- **Generate** warns first (it replaces the daily rows), then reports the
  summary and the warnings — capacity breaches, supply shortfalls, rates that
  cannot deliver.

### Daily planning board

This is the screen that replaces the 86-column worksheet.

```
┌────────────────────────────────────────────────────────────────────────┐
│ ‹ Daily planning board                                    V1  [Draft]  │
│ [Cane][Production][Stock][Shipments]  From … To …  Series [Plan ▾]     │
│                                       ● Unsaved changes  [Save][Discard]│
├────────────────────────────────────────────────────────────────────────┤
│ Date        Delivered  Accepted  Rejected  Crushed   Rate  Stop  Util. │
│ Dec 01,2026 [16788.32][16788.32][      0][16788.32][ 700][  0] 100.00 %│
│ Dec 02,2026 [16788.32][16788.32][      0][16788.32][ 700][  0] 100.00 %│
└────────────────────────────────────────────────────────────────────────┘
```

Instead of one wide sheet: **one process at a time**, one date range at a time,
with the series chosen explicitly.

| Requirement | How |
| --- | --- |
| Editable versus calculated | Editable cells are input fields; calculated ones (utilisation, balances) are plain text. The difference is visible without reading a header |
| Mass entry | Every cell in the range is editable; only changed rows are sent |
| Unsaved-change protection | A dirty indicator, a guard on navigation and tab switching, and a `beforeunload` handler |
| Validation | Non-numeric and negative values are caught in the field; everything else is caught by the server and the offending row is highlighted |
| All-or-nothing saves | A planner fixes one cell rather than discovering later that half the week saved |
| Read-only states | When the version is released and locked, or the series is wrong for the version, or the caller lacks the permission, the board says which and disables the inputs |
| Sticky headers | Column headers stay put while scrolling a fortnight |

### Warehouse and silo

The capacity table for every store — capacity, usable, closing stock, use
percentage, first warning date, first full date, and the shipment rate that
would prevent it. Selecting a store draws its balance against the capacity lines
and lists the daily ledger.

### Stock

The warehouse keeper's page. The current position first — on hand, on hold,
available, capacity and utilisation per store and product — and the movements
that produced it below, because a keeper wants to know what is in the shed before
they want to know how it got there.

**Post a movement** opens one dialog for every movement type. The quantity is
entered as a positive number and the movement type decides the sign; the
receiving store appears only for a transfer. A refused posting leaves the dialog
open with the offending line named, so the keeper corrects it rather than typing
the whole movement again.

**Reverse** posts the counter-document. Nothing is deleted and nothing is edited:
both documents stay in the ledger, which is what lets somebody six months later
see that a mistake was made and what was done about it.

### Production orders

What the floor has been told to make, and what it actually made. **Create from
plan** raises orders for a range of days from the released plan and reports both
numbers — created and already covered — because "nothing created" is a normal
answer when the range is already done. The button is disabled when the season has
no released plan, and the message strip says why.

Selecting an order opens its detail: quantities, variance, its confirmations, and
the actions its status allows. The buttons follow `allowedActions` from the API,
so a button that would be refused is disabled rather than offered and then
rejected.

**Confirm** offers the open quantity as the default, so a shift that made what it
was asked to make presses one button. Only the yield is receipted into the chosen
store; scrap and rework are recorded but not.

### Quality

The laboratory. Samples above, holds on stock below.

Selecting an open sample opens the laboratory sheet with one row per configured
parameter. A parameter left blank was not measured and is not sent — an empty box
is not a reading of zero. The sheet is judged server-side and the verdict comes
back with it; a parameter with no limits in force is reported separately, because
an unspecified parameter is a configuration gap rather than a quality event.

A failed sheet blocks the quantity named on it. The holds table shows what is
blocking and what has been released; releasing needs the quality release
permission, which the keeper does not hold.

A completed sample carries a **Certificate** button that downloads the
certificate of analysis as a PDF — the document that goes in the envelope with a
consignment. It appears only once the sample is complete: a certificate is read
as a guarantee, and an unfinished sheet is not one the laboratory has given yet.

### Maintenance

The outage calendar, with a standing warning at the top: an approved,
factory-wide window removes crushing days, and the season is extended rather than
shortened, so the campaign ends later. Approving asks for confirmation and says
how many days it will cost. Leaving the line unset means the whole factory stops,
which is the case that moves the end of the season; a line outage does not.

### Shipments

Planned against actual by channel, plus what the finished goods stores require:
the smallest constant daily rate that keeps each inside its threshold.

### Materials

The packaging requirement from the plan: gross requirement, safety stock,
available, on order, shortage, purchase requirement, required-by and order-by
dates. The formula is on the screen, because a buyer asked to place a large
order will want to see it.

### Reports

A master-detail: the catalogue on the left, parameters and preview on the right,
with Excel, CSV and PDF buttons. Preview and export come from the same
server-side builder, so they cannot disagree.

### Master data

One generic screen for all fourteen entities. Because every master entity has
the same API shape, the table is built from a column list per entity and a new
entity needs no new page.

### Interfaces

What this system has told the connected systems, and what it has not managed to
tell them yet. Three tiles — waiting, given up, delivered — and the outbox
below, filterable by topic.

The **given up** tile is the one that matters and the easiest to leave off a
screen: those events stopped being retried after twenty-five attempts and are
waiting for a person. They are not lost, and a red strip says so in words rather
than leaving somebody to work it out from a count. **Send now** drains what is
due, which is what somebody does after an outage rather than waiting for the next
tick; **Retry** on a single row delivers it whatever its backoff says, which is
what they do once the far end is fixed.

Behind `integration:read`, which an administrator and an auditor hold. A retry
that still fails shows the far end's own words, because that is what the operator
pressed the button to find out.

### Audit trail

Filterable by entity and action, showing when, what, who, why and the
correlation id. Users without `audit:read` see an explanation instead of an
empty table.

---

## 8.3 UX rules applied throughout

**Semantic colour means one thing.** Error is red, warning orange, success
green, information blue, everywhere, driven by shared formatters. Note that
`sap.m.ValueColor` (Good/Critical/Error/Neutral) and `sap.ui.core.ValueState`
(Success/Warning/Error/Information) are different enumerations; each has its own
formatter, because mixing them throws at render time.

**Every KPI drills down.** A tile opens the overview; a warehouse row opens its
ledger; a report row opens the transactions behind it.

**Business-friendly display, exact storage.** Dates render in the user's locale
and are stored as ISO. Quantities render with thousands separators and are
stored as exact decimals. Formatters are display-only; a formatted value is
never fed back into a calculation.

**Errors name the field.** The problem document's `errors[]` array carries the
row and the field, and the error dialog renders them as "Row 2, caneCrushed:
…" rather than "invalid input".

**Responsive.** The side navigation collapses on a phone; tables use
`demandPopin` to fold secondary columns on narrow screens; the content density
is compact on a desktop and cozy on a touch device.

**Accessible.** Semantic controls throughout, so roles and labels come for free.
Every input has a `Label` with `labelFor`; the chart carries a text alternative
describing what it shows; colour is never the only signal — a state always has
text or an icon beside it.

---

## 8.4 Internationalisation

English is the source language in `i18n/i18n.properties`; every user-visible
string is there. `i18n_km.properties` and `i18n_th.properties` carry the same
keys, and a key that is absent falls back to English, so an untranslated string
appears in English rather than as a raw key.

The Khmer and Thai files are seeded with the shell and navigation — what an
operator meets first — and are structurally complete rather than fully
translated. Adding a language is a properties file and one entry in
`supportedLocales`.

One caveat, recorded as open question Q9: the PDF writer uses the standard
Helvetica fonts, which cannot render Khmer or Thai. Screen and Excel output are
fine; PDF needs an embedded font when the business confirms it is required.

---

## 8.5 Verification

The application was driven end to end in Chromium against the real API during
development:

- Sign in, and the season list loading into the shell
- Every navigation target rendering with live data and no console errors
- The dashboard's KPIs, alerts, chart and storage table matching the API
- The planning board: 84 editable cells over a fortnight, a value edited, the
  dirty indicator appearing, the save persisting (verified by re-reading through
  the API), the row version incrementing and an audit record being written
- A negative value flagged in the field before it reaches the server
- Master data, reports, materials and shipments rendering their real data
- The audit trail correctly refusing a planner, who lacks `audit:read`

The execution pages were walked the same way, as five different users:

- A keeper posting a 500 t receipt and seeing the balance, capacity and
  utilisation update
- An issue of 900 t against 500 t on hand refused, with the offending line named
  and the figures quoted, and the ledger left with only the receipt
- A reversal posted, the balance returning to zero and both documents staying in
  the ledger
- A supervisor raising orders from the released plan for three days, then running
  the same range again and getting "0 created, 6 already covered"
- An order released and confirmed at 973.723 t against a plan of 973.723 t, the
  status reaching COMPLETED and the goods receipt posted
- The laboratory opening a sample and its sheet showing all five seeded
  parameters
- An approver planning a three-day outage, approving it after the warning, and
  the next plan generation ending on 19 April instead of 16 April — 137 working
  days either way

Two defects the walkthrough found, both invisible from the code: a `sap.m.Select`
bound to an empty key displays its first item while the model stays empty, so the
dialog showed a warehouse chosen that the server never received; and the order
list rendered raw product UUIDs, because an order carries the product's id and
nothing had resolved it.
