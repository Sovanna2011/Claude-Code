# 5. Universal Journal & Core Accounting Model

## 5.1 The principle

**One line table holds every accounting fact.** G/L, accounts receivable,
accounts payable, asset accounting, and controlling all write to
`fin.JournalEntryLine`. There are no parallel accumulators, no separate CO
document table with its own amounts, no subledger tables that could drift from
the G/L.

What this buys:

- **Reconciliation by construction.** "Customer subledger equals the
  reconciliation account balance" is not a nightly job — both sides are
  `SUM()` over the same rows with different predicates.
- **Instant drill-down.** Balance sheet → line item → source document is one
  query path, not a chain of link tables.
- **One extension point.** A new dimension (a future WBS element, production
  order) is one nullable column plus dictionary metadata, and *every* report can
  use it immediately.

What it costs, and how it is handled: one very wide, very large table. Mitigated
by a deliberate index strategy, partitioning by fiscal year, and read-only
reporting projections in `rpt` for closed periods
([12-database-schema-strategy.md](12-database-schema-strategy.md)).

```
                        ┌──────────────────────────┐
   FI journal (FB50) ──►│                          │
   Customer invoice  ──►│                          │──► Trial balance / GL
   Vendor invoice    ──►│  fin.JournalEntryHeader  │──► Customer & vendor balances
   Payments/clearing ──►│  fin.JournalEntryLine    │──► Aging / open items
   Asset postings    ──►│    (universal journal)   │──► Cost & profit center reports
   Depreciation run  ──►│                          │──► Internal order reports
   CO allocations    ──►│                          │──► Balance sheet / P&L / FSV
   (future modules)  ──►│                          │──► Intercompany reconciliation
                        └──────────────────────────┘
```

## 5.2 Ledgers and accounting principles

Multi-ledger and multi-GAAP (§2) are handled by a **ledger dimension on the
header and line**, not by separate tables:

| Concept | Design |
|---|---|
| `fin.Ledger` | `LedgerKey`, name, `IsLeading`, `AccountingPrincipleId`, `FiscalYearVariantId`, currency types carried |
| `fin.AccountingPrinciple` | e.g. `IFRS`, `CIFRS` (local Cambodian), `TAX` |
| `fin.LedgerCompanyCode` | which ledgers are active per company code, with their own posting period variant |
| Posting | A document posts to the **leading ledger** plus every assigned parallel ledger, in the same transaction, sharing one document number but distinguished by `LedgerId` |
| Ledger-specific postings | A valuation or provision that exists only under IFRS posts to that ledger alone (`fin.DocumentType.RestrictedToLedgerId`) |

Reports always take a ledger parameter; it defaults to the leading ledger.

## 5.3 Journal header

`fin.JournalEntryHeader` — business key `(TenantId, CompanyCodeId, FiscalYear,
DocumentNumber, LedgerId)`.

| Group | Fields |
|---|---|
| Identity | `Id`, `TenantId`, `CompanyCodeId`, `LedgerId`, `FiscalYear`, `DocumentNumber`, `DocumentTypeId` |
| Dates | `DocumentDate`, `PostingDate`, `EntryDate` (UTC), `FiscalPeriod`, `TranslationDate` |
| Currency | `DocumentCurrency`, `ExchangeRate decimal(23,6)`, `ExchangeRateTypeId`, `LocalCurrency`, `GroupCurrency` |
| Text | `Reference`, `HeaderText`, `DocumentHeaderReference` (external), `Assignment` |
| Status | `Status` (Draft/Held/Parked/Submitted/PendingApproval/Approved/Posted/Reversed/Cancelled), `PostedAt`, `PostedBy` |
| Reversal | `IsReversal`, `ReversedDocumentId`, `ReversalDocumentId`, `ReversalReasonId`, `ReversalDate` |
| Source | `SourceModule` (FI/AR/AP/AA/CO/MM/SD/PY/…), `SourceDocumentType`, `SourceDocumentId`, `IdempotencyKey`, `CorrelationId` |
| Control | `TotalDebitDocument`, `TotalCreditDocument`, `TotalDebitLocal`, `TotalCreditLocal`, `LineCount` |
| Intercompany | `IntercompanyTransactionId`, `PartnerCompanyId` |
| Workflow | `WorkflowInstanceId` |
| Audit | `CreatedAt/By`, `ModifiedAt/By`, `RowVersion` |

Header totals are stored (not computed on read) so a "debits = credits" integrity
sweep is a cheap scan, and so document lists don't aggregate millions of lines.

## 5.4 Journal line — the universal line

`fin.JournalEntryLine` — key `(JournalEntryHeaderId, LineNumber)`.
Every field required by §13 is present:

| Group | Fields |
|---|---|
| Keys | `Id`, `TenantId`, `JournalEntryHeaderId`, `CompanyCodeId`, `LedgerId`, `FiscalYear`, `DocumentNumber`, `LineNumber`, `PostingDate`, `FiscalPeriod` |
| Posting control | `PostingKeyId`, `DebitCreditIndicator` (D/C), `AccountType` (G/L, Customer, Vendor, Asset, Material-future) |
| Account | `GLAccountId`, `BusinessPartnerId`, `BusinessPartnerRole`, `ReconciliationAccountId`, `AssetId`, `AssetSubNumber`, `AssetTransactionTypeId` |
| CO dimensions | `CostCenterId`, `ProfitCenterId`, `InternalOrderId`, `FunctionalAreaId`, `SegmentId`, `PartnerProfitCenterId`, `CostElementId`, `ActivityTypeId` |
| FI dimensions | `BusinessAreaId`, `PlantId`, `BranchId`, `PartnerCompanyId`, `PartnerBusinessAreaId`, `TradingPartnerSegmentId` |
| Extensible assignment | `AccountAssignmentType`, `AccountAssignmentId` (reserved for WBS / production order / maintenance order — added by later modules with **no schema change**) |
| Amounts | `AmountDocument decimal(19,4)`, `AmountLocal`, `AmountGroup`, `AmountControllingArea`, `AmountHard`, `AmountIndex`, each with its currency code; `SignedAmountLocal` (computed, +debit/−credit) |
| Tax | `TaxCodeId`, `TaxBaseAmountDocument`, `TaxAmountDocument`, `TaxAmountLocal`, `TaxJurisdictionId`, `IsTaxLine`, `WithholdingTaxCodeId`, `WithholdingTaxAmount` |
| Quantity | `Quantity decimal(23,6)`, `BaseUnitOfMeasure` (for statistical/consumption reporting and future logistics) |
| Text & refs | `Assignment` (sort field), `LineItemText`, `Reference1/2/3` |
| Open item | `IsOpenItemManaged`, `OpenItemStatus` (Open/PartiallyCleared/Cleared), `DueDate`, `BaselineDate`, `PaymentTermsId`, `PaymentMethod`, `PaymentBlockId`, `DunningLevel`, `ClearingDocumentNumber`, `ClearingDate`, `ClearingId` |
| Source | `SourceModule`, `SourceLineReference` |
| Audit | `CreatedAt/By`, `RowVersion` (no `ModifiedAt` — posted lines do not change) |

### Multi-currency on every line

A line always stores at least three amount triplets: **document**, **local
(company code)**, **group**. Controlling-area, hard, and index currencies are
populated when the company code/CO area configuration requires them. This is what
makes "foreign currencies reconcile with local and group currencies" (§23) a
row-level property rather than an aggregation-time reconstruction.

### Derived vs. stored

| Value | Stored? | Why |
|---|---|---|
| Local/group amounts | **stored** | The rate at posting time is a fact; recomputing later gives a different answer |
| Reconciliation account | **stored** | Configuration may change; the document must not |
| Profit center, segment, functional area | **stored after derivation** | Derivation rules change over time |
| Balances (`GLBalance`, customer balance) | **not stored** in Phases 1–4 | Aggregations over indexed views; snapshot tables in `rpt` only at Phase 5, and always rebuildable from the journal |

## 5.5 Subledgers as views

```
Customer open items   = JournalEntryLine WHERE AccountType='Customer'
                                           AND IsOpenItemManaged AND OpenItemStatus<>'Cleared'
Vendor balance        = Σ SignedAmountLocal WHERE AccountType='Vendor' AND BusinessPartnerId=@bp
G/L balance           = Σ SignedAmountLocal WHERE GLAccountId=@acct  (period/ledger scoped)
Cost center actuals   = Σ SignedAmountLocal WHERE CostCenterId=@cc AND CostElementId IS NOT NULL
Asset values          = Σ over lines WHERE AssetId=@asset (+ fin.AssetTransaction detail)
```

The **mandatory reconciliation test** (§23) is therefore:

```sql
-- must always return zero rows
SELECT bp.BusinessPartnerId, SUM(...)  -- subledger by recon account
EXCEPT
SELECT gl.GLAccountId, SUM(...)        -- G/L balance of the same recon account
```

`fin.OpenItem` exists as a **narrow companion table** (line reference, BP,
account, due date, remaining amounts per currency, status) purely as a
performance index for payment proposals, dunning, and aging — it is a *derived*
structure maintained inside the posting transaction and fully rebuildable from
the journal. It is never an independent truth.

## 5.6 G/L account master

| Level | Table | Content |
|---|---|---|
| Chart of accounts level | `fin.GLAccount` | `ChartOfAccountsId`, `AccountNumber`, `AccountGroupId`, `AccountType` (Balance sheet / P&L / Reconciliation / Secondary cost element / Statistical), descriptions (multilingual), `IsBlocked` |
| Company-code level | `fin.GLAccountCompanyCode` | `CurrencyCode`, `IsOnlyLocalCurrency`, `IsOpenItemManaged`, `IsLineItemDisplay`, `SortKey`, `FieldStatusGroupId`, `IsPostingBlocked`, `IsReconciliationAccountFor` (Customer/Vendor/Asset), `TaxCategory`, `IsTaxPostingWithoutTaxCodeAllowed`, `IsRelevantToCashFlow`, `AlternativeAccountNumber`, `PlanningLevel`, `HouseBankId` |
| CO level | `co.CostElement` | primary (linked to a P&L G/L account) or secondary (allocation-only), `CostElementCategory` |

**Invariants enforced at posting time**

1. A reconciliation account cannot be posted to directly (§23) — only through a
   BP or asset line that derives it.
2. A P&L account requires a CO account assignment when its cost element is
   defined and the account category demands it.
3. A blocked account (chart or company-code level) rejects postings.
4. Open-item management cannot be switched off while open items exist.
5. Account currency ≠ local currency restricts which document currencies post.

## 5.7 Document flow & status model

```
        ┌────────┐  save draft   ┌────────┐  hold  ┌────────┐
        │  New   │──────────────►│ Draft  │───────►│  Held  │
        └────────┘               └───┬────┘        └───┬────┘
                                     │ park            │
                                     ▼                 │
                                 ┌────────┐            │
                                 │ Parked │◄───────────┘
                                 └───┬────┘
                            submit   │
                                     ▼
                          ┌────────────────────┐   reject   ┌──────────┐
                          │ Pending Approval   │───────────►│ Rejected │
                          └─────────┬──────────┘            └────┬─────┘
                            approve │                            │ resubmit
                                    ▼                            │
                              ┌──────────┐                       │
                              │ Approved │◄──────────────────────┘
                              └────┬─────┘
                                   │ post  (posting engine, §15)
                                   ▼
   ┌──────────┐  clear    ┌────────────┐   reverse   ┌──────────┐
   │ Cleared  │◄──────────│   Posted   │────────────►│ Reversed │
   └──────────┘ partially └────────────┘             └──────────┘
                          (immutable from here on)
```

- **Draft / Held / Parked** live in the same header table with a non-Posted
  status and **no document number from the posting range** (they draw from a
  separate parked-document range so posted numbering stays gapless-by-intent).
- **Posted is terminal.** No update path exists. Clearing and reversal *add*
  documents and set reference fields; they never rewrite the original amounts.
- **Reversal** creates a new document of the configured reversal type with
  inverted debit/credit, `ReversedDocumentId` pointing back, and reopens any open
  items the original had cleared. Reversal is refused if the original is already
  reversed, if its period is closed (unless an alternate posting date in an open
  period is supplied), or if the resulting open item was already cleared onward.

## 5.8 Asset accounting model (condensed)

| Table | Purpose |
|---|---|
| `fin.AssetClass` | Number range, account determination key, default depreciation areas/keys, useful-life defaults, `IsAssetUnderConstruction`, `IsLowValue` |
| `fin.Asset` | `AssetNumber`, `SubNumber` (main asset = 0000), class, description, cost center, profit center, plant, location, capitalization date, deactivation date, quantity, `SerialNumber`, custom fields |
| `fin.DepreciationArea` | Area 01 = leading/book, 15 = tax, 30 = group; `TargetLedgerId`, `PostsToGL` (real/statistical/derived), currency |
| `fin.AssetDepreciationArea` | Per asset × area: depreciation key, useful life (years/periods), start date, scrap value, changeover |
| `fin.DepreciationKey` | Method (straight-line, declining balance, manual, unit-of-production), base value, period control (pro-rata, mid-month, full-year), multi-level rules |
| `fin.AssetTransaction` | Acquisition, vendor acquisition, transfer, retirement, sale, write-up, unplanned depreciation, impairment, settlement from AuC; links to the journal document |
| `fin.AssetValue` | Per asset × area × fiscal year: APC, accumulated depreciation, planned/posted depreciation by period, net book value |
| `fin.DepreciationPosting` | Depreciation run results: run id, period, asset, area, amount, journal document |

Depreciation run (`AFAB`) is a Hangfire job: calculates planned values, groups by
account determination + cost center, and posts **through the posting engine** as
one document per company code/period (configurable granularity). Rerunning the
same period is idempotent — the run key `(CompanyCode, FiscalYear, Period, Area,
RunType)` is the idempotency key.

## 5.9 Controlling model (condensed)

| Table | Purpose |
|---|---|
| `co.CostCenter` | Key, name, category, `ControllingAreaId`, responsible person (BP/user), department, `ProfitCenterId`, `FunctionalAreaId`, `CompanyCodeId`, validity dates, lock indicators (actual primary/secondary/revenue/commitment) |
| `co.CostCenterGroup` / `co.CostCenterHierarchyNode` | Standard + alternative hierarchies |
| `co.ProfitCenter`, `co.ProfitCenterHierarchyNode` | Standard hierarchy; `SegmentId` for derivation |
| `co.InternalOrder` | Order type, number range, description, responsible cost center, requesting cost center, `IsStatistical`, `IsRealOrder`, currency, status (Created/Released/TechnicallyComplete/Closed), validity |
| `co.InternalOrderBudget` | Budget by year/period, `AvailabilityControlProfile`, tolerance levels (warn/error %) |
| `co.SettlementRule` / `co.SettlementLine` | Receiver type (cost center, G/L, asset, order, project-future), percentage or equivalence numbers, validity, settlement type (periodic/full) |
| `co.ActivityType`, `co.ActivityPrice` | Activity types, planned/actual prices per cost center × year |
| `co.StatisticalKeyFigure`, `co.StatisticalKeyFigureValue` | Allocation bases (headcount, m², kWh) |
| `co.AllocationCycle` / `…Segment` / `…SenderReceiver` | Distribution and assessment cycles; assessment posts via secondary cost elements |
| `co.Commitment` | Open commitments from future purchasing; already modelled so availability control works when Purchasing arrives |
| `co.PlanVersion`, `co.PlanLine` | Plan data by version/year/period for plan-vs-actual |

**CO postings are journal lines.** A distribution or assessment creates a
document with a secondary cost element, sender credit and receiver debit — in the
same universal journal, in a CO-only ledger view (`PostsToGL = false` for
statistical). There is no separate CO amount store; `co.ControllingPosting`
exists only as a **link table** carrying CO-specific attributes (cycle id,
allocation run, sender/receiver pair) referencing the journal line.

## 5.10 FI ⇄ CO integration (§8.4)

Order of derivation during posting (each step may be overridden by an explicit
entry from the user):

```
1. Cost element check   → is the G/L account a cost element? if yes, a CO object is REQUIRED
2. Default assignment   → cost center default on the G/L account × company code
3. Substitution rules   → user-defined rules (cfg.SubstitutionRule) may set/replace values
4. Profit center        → from cost center / internal order / asset / BP / default
5. Segment              → from profit center (or explicit)
6. Functional area      → from cost center category / G/L account / explicit
7. Partner profit center→ from the counter-line for intercompany or cross-PC postings
8. Validation rules     → cfg.ValidationRule: reject the document if a rule fails
```

`cfg.ValidationRule` and `cfg.SubstitutionRule` are stored as (callup point,
prerequisite expression, check/assignment expression) using a **safe,
parsed expression language** — a restricted grammar over document fields with no
code execution. Rules are versioned and validity-dated.

**Statistical vs. real:** exactly one *real* CO object per line (cost center OR
real internal order OR asset); statistical orders and profit centers may accompany
it. Enforced by `ICostObjectValidator` and covered by unit tests.

## 5.11 Period control and closing

| Table | Purpose |
|---|---|
| `cfg.PostingPeriodStatus` | Open/closed intervals per (variant, account type, account range, fiscal year), plus a second interval openable only to an authorization group (month-end team) |
| `fin.PeriodCloseTask` | Checklist per company code × period: FX valuation, recurring entries, depreciation, allocations, GR/IR-future, reconciliation, blocking |
| `fin.PeriodCloseLog` | Who closed what, when, with the balance snapshot at close |
| `fin.YearEndCarryForward` | Balance carry-forward per account/ledger/year, retained-earnings posting reference |

Posting into a closed period is rejected in step 2 of the posting engine — one of
the mandatory tests (§23).

## 5.12 Financial statement versions

`cfg.FinancialStatementVersion` → hierarchical `…Node` (with node type: header,
account group, total, retained earnings, not-assigned) → `…AccountAssignment`
(account intervals, debit/credit-dependent placement). Reports (balance sheet,
P&L) render an FSV against journal aggregates, so a new statement layout requires
no code.
