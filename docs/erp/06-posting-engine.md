# 6. Central Posting Engine & Currency Architecture

## 6.1 Contract

Every module posts through one interface. There is no second path to the journal.

```csharp
// Erp.Posting.Domain (referenced by every module that posts)
public interface IPostingEngine
{
    Task<PostingResult> PostAsync(PostingRequest request, CancellationToken ct);
    Task<SimulationResult> SimulateAsync(PostingRequest request, CancellationToken ct);
    Task<PostingResult> ReverseAsync(ReversalRequest request, CancellationToken ct);
}

public sealed record PostingRequest(
    Guid TenantId,
    Guid CompanyCodeId,
    string DocumentTypeKey,
    DateOnly DocumentDate,
    DateOnly PostingDate,
    string DocumentCurrency,
    IReadOnlyList<PostingLine> Lines,
    string IdempotencyKey,              // required — see §6.5
    SourceModule Source,
    string? Reference = null,
    string? HeaderText = null,
    ExchangeRateOverride? Rate = null,
    Guid? WorkflowContextId = null,
    IReadOnlyList<Guid>? LedgerIds = null,   // null = leading + all assigned
    Guid? IntercompanyGroupId = null);
```

`SimulateAsync` runs steps 1–15 **and then rolls back** — it returns the exact
document that *would* be created, including derived accounts, taxes, converted
amounts, CO derivations, and validation messages. This is what the Journal Entry
page's **Simulate** button calls (§19: "Simulation must display the complete
accounting result without posting").

## 6.2 The 19-step pipeline (§15)

```
                       ┌──── SQL transaction opens at step 13 ────┐
 1  Validate tenant & company code        (exists, active, user authorized to it)
 2  Validate fiscal period                (derive year+period from posting date;
                                           period open for every account type used)
 3  Validate authorization                (T-code × activity 'Post' × org scope;
                                           maker-checker: poster ≠ approver)
 4  Generate document number              (number range, concurrency-safe, §6.4)
 5  Determine automatic accounts          (account determination by transaction key)
 6  Validate posting keys                 (key ↔ account type ↔ debit/credit consistent)
 7  Validate required fields              (field status variant/group + dictionary rules)
 8  Calculate taxes                       (tax code → base, rate, deductible split,
                                           withholding; generates tax lines)
 9  Convert currencies                    (document → local → group → CO/hard/index)
10  Derive controlling dimensions         (cost element check, defaults, substitutions,
                                           profit center, segment, functional area)
11  Validate reconciliation accounts      (no direct posting; BP/asset line derives it)
12  Check debits = credits                (per ledger, per currency, per company code)
13  Create open items                     (for open-item-managed accounts and BP lines)
14  Create asset postings                 (fin.AssetTransaction + value updates)
15  Create controlling postings           (co.ControllingPosting links, commitments)
16  Save everything in ONE SQL transaction
17  Write an immutable audit record       (audit.AuditLog + document snapshot hash)
18  Start approval when required          (workflow rule match → wf.WorkflowInstance)
19  Publish an integration event          (intg.OutboxMessage — dispatched after commit)
                       └──────────────── commit ─────────────────┘
```

Notes on ordering:

- Steps 1–12 are **pure validation and derivation**; they run in `Simulate` too.
- The document number (step 4) is drawn *inside* the transaction so a rollback
  returns it — see §6.4 for the gap trade-off.
- Step 18 matters: a document that requires approval is posted as *Parked +
  PendingApproval*, not *Posted*. The engine's `PostingMode` decides whether the
  request may bypass approval (e.g. a system-generated depreciation posting under
  a rule that authorizes it).
- Step 19 writes to the outbox in the same transaction; dispatch happens after
  commit, so a webhook outage can never roll back a posting.

Each step is an `IPostingStep` implementation in an ordered pipeline. This makes
the engine testable step-by-step and lets a future module insert a step
(e.g. inventory valuation) without editing the others.

## 6.3 Failure semantics

| Failure | HTTP | Behaviour |
|---|---|---|
| Validation (steps 1–12) | 422 with RFC 7807 + `errors[]` carrying step, field, message code | Nothing written; document not created |
| Authorization (step 3) | 403 Problem Details, reason logged to `audit.SensitiveAccessLog` | Nothing written |
| Concurrency (row version) | 409 | Retry-able by the client |
| Transient SQL | retried by execution strategy | Safe due to idempotency |
| Duplicate idempotency key | 200 with the **original** result | No second document |

Validation messages are message *codes* resolved through the dictionary, so they
are translatable (en/km) and stable for API consumers.

## 6.4 Document numbering (§14)

```
cfg.NumberRange        (TenantId, RangeKey, CompanyCodeId?, DocumentTypeKey?, ObjectType,
                        FiscalYear?, FromNumber, ToNumber, Prefix, PaddingLength,
                        IsExternal, IsYearDependent, WarnThresholdPercent)
cfg.NumberRangeStatus  (NumberRangeId, FiscalYear, CurrentNumber, RowVersion)
cfg.NumberRangeGapLog  (NumberRangeId, FiscalYear, Number, ReasonCode, OccurredAt, UserId)
```

**Concurrency-safe draw** — a single atomic statement, no read-then-write race:

```sql
UPDATE cfg.NumberRangeStatus WITH (ROWLOCK)
   SET CurrentNumber = CurrentNumber + 1
OUTPUT inserted.CurrentNumber
 WHERE NumberRangeId = @rangeId AND FiscalYear = @fy;
```

Formatting: `{Prefix}-{FiscalYear}-{DocumentTypeKey}-{Number:D(PaddingLength)}`
→ `KSS-2026-SA-0000000123`.

**The gap trade-off, stated explicitly.** Drawing the number *inside* the posting
transaction means a rollback consumes a number and leaves a gap. Drawing it
*outside* means no gaps on rollback but a crash between draw and commit leaves a
gap anyway — and introduces a window where two documents could claim one number
if the outside draw is not itself transactional.

**Decision:** draw inside the transaction (correctness first, duplicates
impossible), and *monitor* gaps rather than pretend to prevent them:
`cfg.NumberRangeGapLog` records every rolled-back draw with a reason, and a
report lists gaps per range/year with their cause. Jurisdictions requiring
strictly gapless numbering (invoice numbering in some countries) get a separate
**post-commit sequential legal number** assigned by a dedicated single-writer
service — modelled now, implemented when a market requires it.

Steps 4 + 12 together satisfy §14: *"Duplicate numbers must be impossible during
concurrent posting"* — covered by `ConcurrentNumberRangeTests` (§23), which runs
N parallel postings and asserts N distinct numbers.

## 6.5 Idempotency

`PostingRequest.IdempotencyKey` is mandatory.

```
fin.PostingIdempotency (TenantId, IdempotencyKey PK, RequestHash, JournalEntryHeaderId,
                        CreatedAt, ResultJson)
```

- Insert of the idempotency row happens **inside** the posting transaction, with a
  unique constraint on `(TenantId, IdempotencyKey)`.
- A repeated call with the same key and the same `RequestHash` returns the stored
  result. Same key, *different* hash → 409 (`idempotency-key-reuse`).
- Keys: UI supplies a client-generated GUID per form submission; background jobs
  use a deterministic key such as `DEP-1000-2026-08-01` (depreciation, company
  code, period) or `PAYRUN-{proposalId}`; API callers pass `Idempotency-Key`
  header. This makes retry storms, double-clicks, and job re-runs harmless.

## 6.6 Currency architecture (§5)

### Currency types carried

| Type | Source | Stored on line |
|---|---|---|
| Document (transaction) | user/source document | `AmountDocument` + `DocumentCurrency` |
| Local (company code) | `org.CompanyCode.LocalCurrency` | `AmountLocal` |
| Group | `org.Company.GroupCurrency` | `AmountGroup` |
| Controlling area | `org.ControllingArea.Currency` | `AmountControllingArea` |
| Hard | company-code config (high-inflation) | `AmountHard` |
| Index-based | company-code config | `AmountIndex` |
| Reporting | *not stored* — translated at report time with a chosen rate type/date | — |

### Master data

```
cfg.Currency          CurrencyCode(3), Name, DecimalPlaces (0–5), RoundingRule,
                      RoundingUnit, IsActive
cfg.ExchangeRateType  RateTypeKey(4), Name, QuotationDirection (Direct|Indirect),
                      IsInversionAllowed, ReferenceRateTypeId, BasisCurrency
cfg.ExchangeRate      RateTypeId, FromCurrency, ToCurrency, ValidFrom,
                      Rate decimal(23,6), FromRatio int, ToRatio int
cfg.CurrencyConversionRounding  per currency pair / company code: rounding mode
```

### Conversion service

```csharp
public interface ICurrencyConversionService
{
    ConversionResult Convert(Money amount, CurrencyCode target,
                            ExchangeRateTypeKey rateType, DateOnly rateDate,
                            Guid companyCodeId, decimal? manualRate = null);
}
```

Rules:

1. **Rate selection** — latest `ValidFrom <= rateDate` for the (type, from, to)
   triple. If absent and the rate type allows inversion, use the inverse of
   (to, from). If still absent and a reference type is configured, translate via
   the basis currency (e.g. KHR→USD→THB). Otherwise: hard error, never a
   silent 1.0.
2. **Quotation direction** — direct (`1 FOR = rate × LOC`) vs indirect, with
   ratios for currencies quoted per 100/1000 units.
3. **Manual rate override** on the document header is allowed only if the user
   holds the `FI_RATE_OVERRIDE` permission; the override and the would-be
   automatic rate are both recorded on the header and in the audit log.
4. **Rounding** to the target currency's `DecimalPlaces` using banker's-rounding
   or half-up per `RoundingRule`. Rounding differences within a document are
   posted to the configured **rounding difference account** so debits = credits
   holds *in every currency*, not just the document currency. This is the single
   most common source of "the ledger doesn't balance in group currency" and is
   handled explicitly at step 12.
5. `decimal(23,6)` for rates; `decimal(19,4)` for amounts; conversions compute in
   `decimal` (never `double`).

### FX gain/loss

| Case | When | Posting |
|---|---|---|
| **Realized** | Clearing an open item at a rate different from the posting rate | Difference → realized FX gain/loss account (per company code + currency, from account determination), with the CO assignment of the original line |
| **Unrealized** | Period-end foreign-currency valuation (`FAGL_VAL`-equivalent job) | Revaluation of open items and FC balance-sheet accounts → unrealized gain/loss + valuation adjustment account; **reversed on the first day of the next period** unless the valuation method is "no reversal" |
| **Translation** | Group reporting / consolidation | Not posted; applied at report time with the reporting rate type |

`fin.ForeignCurrencyValuationRun` stores run parameters, per-item results, and the
generated documents so a valuation is auditable and repeatable.

## 6.7 Tax calculation (step 8)

```
cfg.TaxCode (CompanyCodeId?, CountryCode, TaxCodeKey, TaxType (Output/Input/Withholding),
             TargetTaxCodeKey, IsReverseCharge, IsDeductible, DeductiblePercent)
cfg.TaxRate (TaxCodeId, ValidFrom, RatePercent decimal(9,4), TransactionKey,
             AccountDeterminationKey)
```

- Amounts may be entered **net or gross** (document type controlled); the engine
  derives base and tax accordingly.
- The engine generates the tax line(s) itself — users never hand-key a tax line
  when a tax code is present.
- Non-deductible portions are added back to the expense/asset line, not to the
  tax account.
- Withholding tax is computed on the BP line at payment or invoice time per the
  BP's withholding configuration, produces its own line, and feeds the tax report.
- Reverse-charge generates a paired input/output line at zero net effect.

## 6.8 Intercompany postings (§16)

```
IntercompanyPostingRequest { SourceCompanyCode, PartnerCompanyCode, Lines… }
   → engine builds TWO PostingRequests sharing an IntercompanyGroupId
   → each gets its own document number in its own company code
   → each is completed with the configured intercompany clearing account
     (receivable side in one, payable side in the other)
   → both post inside ONE SQL transaction (same database, modular monolith)
   → cross-company balance check: Σ intercompany clearing per pair per period = 0
```

`fin.IntercompanyTransaction` records the pair, and the intercompany
reconciliation report lists mismatched pairs by period. Because both company
codes live in one database, atomicity is real — a genuine benefit of the monolith
decision (ADR-001).

## 6.9 Where each business process enters the engine (§16)

| Process | Entry point | Document type | Notes |
|---|---|---|---|
| Record-to-Report | `FB50`/`FB01` → `PostJournalEntryCommand` | `SA` | Park → approve → post |
| Customer invoice | AR command → engine | `DR` | Creates customer open item |
| Incoming payment | `F-28` → engine + clearing service | `DZ` | Clearing may post realized FX |
| Vendor invoice | AP command → engine | `KR` | Creates vendor open item |
| Payment run | `F110` Hangfire job → proposal → engine per payment | `KZ` | One idempotency key per proposal line |
| Asset acquisition | `AS01`+ acquisition command → engine | `AA` | Also `fin.AssetTransaction` |
| Depreciation | `AFAB` job → engine | `AF` | Idempotent per period |
| Allocation cycle | `KSU5`-equivalent job → engine | `CO` | Secondary cost elements |
| Order settlement | `KO88` → engine | `CO`/`AA` | Receiver-dependent |
| Payroll (future) | HR module → engine | `PY` | Same contract, no new posting path |
| Goods receipt (future) | Inventory module → engine | `WE` | Same contract |
