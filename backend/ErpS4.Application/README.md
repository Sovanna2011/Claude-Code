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
├── DataDictionary/
│   ├── DictionaryContracts.cs  IDictionaryService, table/domain/element views
│   └── DictionaryService.cs    SE11 display and generated DDL
├── TableBrowser/
│   ├── TableBrowserContracts.cs  ITableBrowserService, filters, operators
│   └── TableBrowserService.cs    SE16N query building, masking, logging
├── Workflow/
│   ├── WorkflowContracts.cs  IWorkflowService, decisions, inbox
│   └── WorkflowService.cs    rules, tasks, maker-checker, substitution
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
ErpS4.Tests/                  123 tests, no database required
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

**The browser builds SQL; nothing else in the system does.** `IErpDataContext`
exposes exactly one raw-query method, and `TableBrowserService` is its only
caller. Identifiers come from the dictionary rows the service just loaded and
are bracket-quoted; every value the caller supplies becomes a parameter. There
is no branch in which request text reaches the statement — which is why a field
name that is not in the dictionary is refused rather than escaped.

**Masked means never read.** A masked column is selected as a literal `NULL`,
so the value never leaves the database, and it cannot be filtered on or sorted
by either: repeated equality probes and an ordering both recover what the
display refuses to show.

**Filter values are parsed before they are sent.** `FilterValue.TryConvert`
turns the text a user typed into the CLR type the column holds, using the
invariant culture. Left as `nvarchar` parameters they would rely on implicit
conversion, which fails two ways that matter: an unparseable value raises a
`SqlException` (message 241) — a 500 for what is really a typing mistake — and
date parsing follows the session's language and `DATEFORMAT`, so the same
filter can mean different days on different servers. `LIKE` is refused outright
on non-text columns, where it would force a per-row conversion and a scan.

## Tests

123 tests, all running against in-memory lists — no database, no EF provider,
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
* a parked document has a number and lines but no balances or open items
* the submitter is never given a task on their own document, and cannot decide
* only the final approval posts; a rejection leaves the ledger untouched
* approval after the period closed escalates rather than posting
* a date filter arrives as a `DateOnly` and a key filter as a `long`, not as
  text the server has to convert; a value that is not the column's type is a
  violation, and `LIKE` on a date column is refused
* a security table is refused by the browser and absent from its table list
* a filter value carrying `'; DROP TABLE …` travels as a parameter, and a field
  name that is not in the dictionary is refused outright
* `%` and `_` typed by a user match themselves instead of expanding
* the tenant condition is added by the service, never asked for by the caller
* the authorization group's row cap beats the request, and a truncated result
  says so
* a masked column comes back as asterisks and cannot be filtered or sorted on
* every query is logged with who ran it, what it selected and how much it
  returned
* the DDL SE11 renders is the DDL the dictionary rows describe

```bash
dotnet test backend/ErpS4.Tests
```

All 123 pass on .NET 10.0.110. Getting there cost four real bugs that static
checking had not found, described in the repository history: a draft with mixed
currencies threw out of `Validate` instead of reporting the mixture, a payment
left its own open item dangling on the customer account, the SE11 field join
was a null reference the moment it ran over objects rather than SQL, and the
number-range mask produced document numbers two characters wider than the
column that stores them.

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
5. ~~**Workflow**~~ — done: `IPostingEngine.ParkAsync` writes a document with a
   number and lines but **no ledger effect**; `IWorkflowService` matches a rule,
   creates the tasks, and only the final approval calls `PostParkedAsync` to
   give the document its balances and open items. Maker-checker keeps the
   submitter off their own document twice over — they never receive a task, and
   a decision from them is refused. Substitutes can decide, and the task records
   who acted. Rules are re-checked at approval, so a period that closed while
   the document waited escalates instead of reopening itself.
6. ~~**Dictionary and table browser**~~ — done: `IDictionaryService` is SE11's
   display half — objects, a table with its fields, indexes and foreign keys,
   domains and data elements, a where-used list, and the DDL those rows imply.
   `ITableBrowserService` is SE16N: it resolves the table and its fields through
   the dictionary, refuses anything it cannot find there, adds tenant isolation
   itself, caps the result by authorization group, masks what is marked masked,
   and logs every query.

## Still not built

* **SE11's change half** — activating a dictionary object, generating the
  migration and routing it through approval (design section 10.4). The display
  service deliberately stops at showing the DDL: a service that also ran it
  would be a way to change the database without a migration.
* **Custom object services** — custom tables, fields and their transport.
* **Foreign-currency clearing** and the **automatic payment run** (F110).
* **Asset transfers, write-ups, impairment** and assets-under-construction
  settlement.
* Phase 4 (frontend) and Phase 5 (testing and deployment).
