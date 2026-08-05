# ErpS4.Application — the central posting engine

Phase 3, first milestone: the piece every other module depends on. Section 15
of the design says one posting engine validates and commits every document, so
that is what this project is, and nothing more yet.

```
ErpS4.Domain/                 no packages, no persistence, no framework
├── Money.cs                  amount + currency, rounding by currency precision
├── JournalEntryDraft.cs      a document before configuration is consulted
├── DepreciationCalculator.cs pure period depreciation arithmetic
└── PostingError.cs           stable error codes the API and UI bind to

ErpS4.Application/
├── Posting/
│   ├── PostingContracts.cs   IPostingEngine, request/result types
│   ├── PostingConfiguration.cs  configuration snapshot for one document
│   ├── PostingEngine.cs      orchestration and persistence
│   ├── PostingEngine.Validation.cs  the rule set
│   └── PostJournalEntryCommand.cs   CQRS command, validator, handler
├── Assets/
│   ├── AssetContracts.cs     IAssetService, run requests, results
│   └── AssetService.cs       acquisition, retirement, depreciation run
├── Clearing/
│   ├── ClearingContracts.cs  IClearingService, allocations, results
│   └── ClearingService.cs    payments, partial, residual, discount, reset
├── BusinessPartners/
│   ├── BusinessPartnerContracts.cs  IBusinessPartnerSyncService, results
│   └── BusinessPartnerSyncService.cs  role assignment, BP_SYNC, BP_CHECK
├── Services/CurrencyConverter.cs
└── DependencyInjection.cs

ErpS4.Database/               (existing) + IErpDataContext, NumberRangeService
ErpS4.Tests/                  77 tests, no database required
```

## What the engine enforces

Validation runs in one pass and returns **every** broken rule, each with a
stable code and the field path that caused it, so a user fixes a document once
instead of once per round trip.

| Area | Rule |
|------|------|
| Structure | at least two lines, no zero amounts, one document currency, debits = credits |
| Company code | exists, and the posting date is inside its validity |
| Period | fiscal period exists, is `Open`, and the OB52 window for that account type contains it (both intervals) |
| Document type | permits the account types the posting keys imply |
| Posting key | exists, not blocked, debit/credit indicator agrees with the sign of the amount |
| Account | exists in the chart, open in the company code, not blocked at either level |
| Reconciliation | a G/L line may not touch a reconciliation account; a partner line must use exactly the account that partner's company code segment points at |
| Business partner | exists, not centrally or company-code blocked, has the role segment being posted |
| Assignment | cost centre or internal order where the account demands one; profit centre where the company code demands one; cost objects valid on the posting date |
| Currency | rate exists for document → local and document → group; conversion happens per line, and the local currency has to balance too |

After the rules pass, one transaction writes the header, the lines, the open
items, the controlling documents, the period balances, the audit record and the
integration event.

## Decisions worth knowing

**Amounts are signed.** Debit positive, credit negative, everywhere — in the
draft, in `Money`, and in `fin.JournalEntryLine`. A document is balanced when
its lines sum to zero, which is one rule instead of two.

**Rounding is a currency property.** `Money.Round` takes the decimal places of
the currency involved, so KHR rounds to whole Riel. Rounding a Riel amount to
two decimals would invent precision the currency does not have, and the
difference would surface later as an unexplained balance.

**Conversion is per line, not per document.** A rounding residue then shows up
during posting, where someone can fix it, instead of in a report months later.
The engine refuses a document whose *converted* amounts do not balance.

**Simulation runs the same code.** `Simulate = true` executes every rule and
every conversion and returns the calculated lines without writing — the
simulation screen cannot disagree with the posting.

**Idempotency is a key, not a guess.** A retry with the same key returns the
first document instead of posting a second one. The check is a query on the
unique `IdempotencyKey`, so two racing retries cannot both pass it.

**Reversal never edits.** `ReverseAsync` posts a mirrored document that
references the original, and the only thing it changes on the original is the
pointer to its reversal. There is no code path that alters a posted line.

**Errors are values, not exceptions.** A rejected document returns a list of
`PostingError`; exceptions are reserved for genuine faults. That keeps the
validation pipeline composable and the API mapping trivial.

## Tests

77 tests, all running against in-memory lists — no database, no EF provider,
milliseconds to run. They encode the rules section 23 of the design calls
mandatory:

* debits equal credits, in the document currency and in local currency
* a closed period, and a period outside the OB52 window, reject the posting
* a reconciliation account cannot be posted directly
* a customer line must use its partner's reconciliation account
* a blocked partner is refused
* a retried request does not post twice
* a reversal references its original, mirrors every posting key, and a document
  cannot be reversed twice
* a simulation writes nothing and draws no document number
* every broken rule is reported in one pass
* one partner holds the customer and the supplier role on a single identity
* a repeated role assignment changes nothing
* a company code segment is refused unless its reconciliation account matches
  the role, which is what stops the failure surfacing later as a bad posting
* a payment clears its item, a partial payment leaves the remainder open, and a
  residual closes the invoice and re-ages the balance from the payment date
* an item cannot be paid twice, or for more than it owes
* depreciation stops at the scrap value, charges the first period pro rata, and
  takes exactly what is left in the final period
* a planned depreciation run refuses to run twice for the same period

```bash
dotnet test backend/ErpS4.Tests
```

> **Not compiled or run in this environment.** The .NET SDK could not be
> installed here (`builds.dotnet.microsoft.com` is blocked by the network
> policy), so this code is written against the generated model and checked
> statically — balanced scopes, unique type names, property names verified
> against the catalogue — but neither built nor executed. Run
> `dotnet build` and `dotnet test` before relying on it.

## Not built yet

The rest of Phase 3, in the order it makes sense to add:

1. ~~**Web API**~~ — done: see [`ErpS4.Api`](../ErpS4.Api/README.md).
2. ~~**Business partner synchronisation**~~ — done: `IBusinessPartnerSyncService`
   assigns a role, creates only the data that role needs, and never touches the
   identity. `SynchronizeAsync` repairs partners whose role data is missing;
   `CheckAsync` (BP_CHECK) reports roles without data, data without roles,
   reconciliation accounts of the wrong type, and number mismatches.
3. ~~**Clearing and payments**~~ — done for manual payments:
   `IClearingService` posts the payment **through the posting engine** rather
   than writing its own document, then links it to the items it settles.
   Partial payments, residuals, cash discount and reset are covered. Two gaps
   remain: foreign-currency clearing (needs a per-line local amount override on
   the engine to realise the exchange difference — refused explicitly rather
   than guessed) and the automatic payment run (F110).
4. ~~**Asset and depreciation services**~~ — done: `IAssetService` capitalises,
   retires (booking the gain or loss against net book value) and runs
   depreciation, all through the posting engine. `DepreciationCalculator` is
   pure arithmetic — straight line, declining balance with the switch to
   straight line, immediate, pro rata first period, scrap value floor — so an
   auditor can recalculate it. Accounts come from account determination, and
   missing configuration is a reported violation rather than a null at posting
   time. Not yet: transfers, write-ups, impairment and assets under
   construction settlement.
5. **Workflow** — routing a document to approval instead of posting it when a
   `wf.WorkflowRule` matches, with maker-checker enforced.
6. **Dictionary, table browser and custom object services** — the SE11 and
   SE16N back ends over the metadata tables.
