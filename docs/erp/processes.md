# Annex — Financial Process Diagrams (§16)

Each process is specified with: preconditions, steps, accounting entries, status
changes, approval points, errors, reversal, tables, APIs, screens, reports.
Amounts in the examples use the sample structure from
[03 §3.6](03-enterprise-structure.md) (CC 1000, local KHR, group USD).

---

## P1 · Record-to-Report

**Preconditions** — company code active; fiscal period open for the account
types used; user authorized for `FB50`, activity Post; G/L accounts unblocked.

```
Draft → Validate → Submit → Approve → Post → Report → Close period → Statements
```

| Step | Screen / T-code | Status | Notes |
|---|---|---|---|
| 1 Capture | `FB50` | Draft | Header + lines; simulate any time |
| 2 Simulate | `FB50` → Simulate | Draft | Engine steps 1–15, rolled back |
| 3 Submit | `FB50` → Submit | Submitted → PendingApproval | Workflow rule matched on amount/type |
| 4 Approve | `INBOX` | Approved | Maker ≠ checker enforced |
| 5 Post | automatic on approval | Posted | Document number assigned, immutable |
| 6 Report | `FBL3N`, trial balance | — | Same lines, different predicates |
| 7 Close | `OB52` + close checklist | Period closed | Further postings rejected |
| 8 Statements | `F.01` with an FSV | — | Balance sheet / P&L |

**Accounting entry (accrual example)**

| PK | Account | D/C | Doc (KHR) | Local (KHR) | Group (USD) | CO |
|---|---|---|---|---|---|---|
| 40 | 6110 Office expense | D | 4,000,000 | 4,000,000 | 1,000.00 | CC-ADM |
| 50 | 2190 Accrued expenses | C | 4,000,000 | 4,000,000 | 1,000.00 | — |

**Errors** — period closed (`period-not-open`), unbalanced (`debits-not-equal-credits`),
reconciliation account posted directly (`direct-recon-posting`), missing CO object
for a cost element (`co-assignment-required`), unauthorized company code.

**Reversal** — `FB08`, reversal reason, posting date in an open period; creates a
new document with inverted D/C and links both ways.

**Tables** `fin.JournalEntryHeader/Line`, `fin.OpenItem`, `wf.WorkflowInstance`,
`audit.AuditLog` · **API** `POST /api/v1/journal-entries`,
`POST /api/v1/journal-entries/{id}/simulate|submit|reverse` ·
**Reports** journal register, G/L line items, trial balance, BS/P&L.

---

## P2 · Customer Invoice → Receipt (Order-to-Cash, financial part)

**Preconditions** — BP holds an active Customer/FI-Customer role with company-code
data and a reconciliation account; credit limit checked if the check rule requires.

```
Create invoice → Approve → Post receivable → Receive payment → Match open item → Clear
```

**Invoice** (USD 10,000 to a KHR company code, rate 4,050)

| PK | Account | D/C | Doc (USD) | Local (KHR) | Group (USD) |
|---|---|---|---|---|---|
| 01 | BP 1000001 → recon 1210 Trade receivables | D | 11,000.00 | 44,550,000 | 11,000.00 |
| 50 | 4100 Revenue | C | 10,000.00 | 40,500,000 | 10,000.00 |
| 50 | 2320 Output VAT (10%) | C | 1,000.00 | 4,050,000 | 1,000.00 |

**Receipt at a later rate (4,100)** — full settlement

| PK | Account | D/C | Doc (USD) | Local (KHR) |
|---|---|---|---|---|
| 40 | 1110 Bank | D | 11,000.00 | 45,100,000 |
| 15 | BP 1000001 → recon 1210 | C | 11,000.00 | 44,550,000 |
| 40 | 6910 Realized FX loss/gain | D/C | 0.00 | 550,000 (gain → credit) |

The open item's status moves `Open → Cleared`, `ClearingDocumentNumber` and
`ClearingDate` are stamped on the original line — the only mutation permitted on
a posted line ([12 §12.6](12-database-schema-strategy.md)).

**Variants** — partial payment (residual open item retained), residual payment
(original cleared, new open item for the remainder), payment on account
(unallocated credit), overpayment tolerance.

**Errors** — credit limit exceeded (block or warn per check rule), BP posting
blocked, payment applied to an already-cleared item (`item-already-cleared`),
clearing across company codes without an intercompany relationship.

**Reversal** — reset clearing (`FBRA`-equivalent) reopens the items, then reverse
the payment document.

**Reports** customer balances, open items, aging, statements, dunning history.

---

## P3 · Vendor Invoice → Payment (Procure-to-Pay, financial part)

```
Create invoice → Approve → Post payable → Payment proposal → Approve proposal
   → Generate bank file → Post payment → Clear invoice
```

**Invoice** (THB 50,000 vendor, withholding tax 3%)

| PK | Account | D/C | Doc (THB) |
|---|---|---|---|
| 40 | 6210 Services expense | D | 50,000.00 |
| 40 | 1330 Input VAT (7%) | D | 3,500.00 |
| 31 | BP 1000002 → recon 2110 Trade payables | C | 52,000.00 |
| 50 | 2340 Withholding tax payable | C | 1,500.00 |

**Payment run (`F110`)** — a Hangfire job: select due items by company code,
payment method, due date, and block status → build a proposal → approval →
generate the bank file → post one payment document per payee/bank/method with a
deterministic idempotency key (`PAYRUN-{proposalId}-{lineId}`), so a re-run
after a crash cannot double-pay. This is the single most important idempotency
case in the system.

**Errors** — payment block on BP or item, missing bank details for the method,
insufficient bank-account authorization, proposal already executed.

**Reversal** — payment reversal resets the clearing and reopens the invoice;
the bank file is marked void and a new run is required.

---

## P4 · Asset Lifecycle

```
Create asset → Acquire → Capitalize → Depreciate → Post depreciation → Transfer / Retire
```

| Event | Entry |
|---|---|
| Acquisition from vendor | D Asset APC (via account determination) / C BP vendor recon |
| Capitalization of AuC | D Asset APC / C Asset under construction (settlement `KO88`) |
| Monthly depreciation (`AFAB`) | D Depreciation expense (cost center from asset) / C Accumulated depreciation |
| Retirement without revenue | D Accumulated depreciation, D Loss on disposal / C Asset APC |
| Sale | D Bank/receivable, D Accum. depreciation / C Asset APC, C/D Gain or loss |
| Write-up / impairment | Configured revaluation accounts, per depreciation area |

Each depreciation area may post to a **different ledger** — so IFRS and tax
depreciation coexist on the same asset without a second asset master.

**Idempotency** — `DEP-{companyCode}-{fiscalYear}-{period}-{area}`; a repeat run
returns the original result rather than double-posting.

**Reports** asset register, asset history sheet, depreciation forecast, G/L
reconciliation (asset balances vs. their reconciliation accounts).

---

## P5 · Cost Allocation

```
Capture costs → Define cycle → Execute distribution or assessment → Validate → Post → Report
```

- **Distribution** keeps the original primary cost element; senders are credited
  and receivers debited on the same element.
- **Assessment** aggregates onto a **secondary cost element**, hiding the original
  detail — used for management allocations such as IT services to departments.
- Bases: fixed percentages, fixed amounts, statistical key figures (headcount, m²),
  or actual amounts on the sender.
- Executed as a background job per period; reversible as a unit (`cycle run id`),
  which reverses every document the run created.

**Entry (assessment of CC-ADM to CC-SLS / CC-PRD, 60/40)**

| Account | Object | D/C | Local |
|---|---|---|---|
| 9310 Secondary CE — administration | CC-ADM | C | 10,000,000 |
| 9310 Secondary CE — administration | CC-SLS | D | 6,000,000 |
| 9310 Secondary CE — administration | CC-PRD | D | 4,000,000 |

Net G/L effect is zero — an assessment moves cost between CO objects, which is
why the secondary cost element is not a P&L account.

---

## P6 · Internal Order

```
Create order → Approve budget → Capture cost → Availability check → Settle → Close
```

- **Real order**: costs land on the order and are settled onward.
- **Statistical order**: costs land on a cost center; the order carries a
  statistical copy for reporting only.
- **Availability control**: at posting, committed + actual against budget →
  warning at 90 %, error at 100 % (tolerance-configurable, per order type).
- **Settlement (`KO88`)**: rules distribute the balance to receivers — asset
  (investment orders), cost centers, G/L accounts, or (later) projects — as
  percentages or equivalence numbers, periodic or full.
- **Closing**: refuses if a balance remains or open commitments exist.

---

## P7 · Intercompany Posting

```
Create source posting → Create partner posting → Validate IC balance → Post both atomically → Reconcile
```

**Example** — CC 1000 pays an expense on behalf of CC 1100 (USD 5,000):

| Company code | Account | D/C | Amount |
|---|---|---|---|
| 1000 | 6210 Services expense | D | 5,000.00 |
| 1000 | 1290 IC receivable — 1100 | D | — |
| 1000 | 1110 Bank | C | 5,000.00 |
| 1100 | 6210 Services expense | D | 5,000.00 |
| 1100 | 2190 IC payable — 1000 | C | 5,000.00 |

Both documents share an `IntercompanyGroupId`, are numbered in their own company
codes, and commit in **one SQL transaction** ([06 §6.8](06-posting-engine.md)).
`PartnerCompanyId` is stamped on every line so the elimination report can pair
them. Reconciliation report: Σ IC clearing accounts per pair per period **must be
zero** — any non-zero row is an exception to investigate, and it is an assertion
in the intercompany test suite.

**Reversal** — reverses both documents together; a partial reversal of one side
is rejected.

---

## Cross-process rules

1. Every one of these processes enters the ledger through `IPostingEngine`. No
   process has its own posting code.
2. Every process step that changes status writes an audit record.
3. Every process supports simulation before the irreversible step.
4. Every reversal references the original document and never edits it.
5. Every process is reportable from the universal journal alone.
