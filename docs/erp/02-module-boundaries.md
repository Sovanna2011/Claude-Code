# 2. Module Boundaries

Thirteen bounded modules (§3). Each is a *vertical slice*: its own domain,
application, infrastructure, database schema, API surface, permissions, and
tests. A module owns its tables exclusively — no other module may read or write
them through EF; cross-module reads go through a published **query contract**,
cross-module writes through **commands** or **integration events**.

## 2.1 Module map and dependency direction

```
                          ┌────────────────────────────────┐
                          │      SharedKernel (types)      │
                          │ Money · CurrencyCode · TenantId│
                          │ DateRange · Result · DomainEvent│
                          └──────────────┬─────────────────┘
                                         │ (referenced by all)
   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
   │ Organization │◄──│BusinessPartner│  │ DataDictionary│──►│Customization │
   │    (org)     │   │    (mdm)      │  │    (cfg)     │   │  (cfg/ext)   │
   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘   └──────────────┘
          │  config           │ identity        │ metadata          │
          ▼                   ▼                 ▼                   ▼
   ┌───────────────────────────────────────────────────────────────────────┐
   │                    POSTING ENGINE  (fin core, shared)                 │
   └───────────┬───────────────────┬───────────────────┬───────────────────┘
               ▼                   ▼                   ▼
        ┌────────────┐      ┌────────────┐      ┌────────────┐
        │  Finance   │      │ Controlling│      │   Assets   │
        │   (fin)    │      │    (co)    │      │(fin.Asset*)│
        └─────┬──────┘      └──────┬─────┘      └──────┬─────┘
              └────────────┬───────┴───────────────────┘
                           ▼ integration events
   ┌────────────┐   ┌────────────┐   ┌────────────┐   ┌────────────┐
   │  Workflow  │   │  Security  │   │ Reporting  │   │Integration │
   │    (wf)    │   │   (sec)    │   │   (rpt)    │   │  (intg)    │
   └────────────┘   └────────────┘   └────────────┘   └────────────┘
                    ┌────────────┐   ┌────────────┐
                    │TableBrowser│   │   Audit    │
                    │ (reads cfg)│   │  (audit)   │
                    └────────────┘   └────────────┘
```

**Dependency rule:** arrows point *downstream only*. Finance may query
Organization and BusinessPartner contracts; Organization knows nothing about
Finance. Workflow, Reporting, Integration, and Audit are downstream of everything
and are never depended upon by the financial core — they react to events.

## 2.2 Module responsibilities and owned schemas

| Module | Schema(s) | Owns | Key contracts it publishes |
|---|---|---|---|
| **Organization** | `org`, part of `cfg` | Tenant, Company, CompanyCode, BusinessArea, Plant, Branch, Location, Department, SalesOrg, PurchasingOrg, ControllingArea, OperatingConcern, CreditControlArea, FunctionalArea, Segment, ChartOfAccounts, FiscalYearVariant, FiscalPeriod, PostingPeriodVariant, FieldStatusVariant, DocumentType, NumberRange, Currency, ExchangeRateType, ExchangeRate, TaxJurisdiction, TaxCode | `ICompanyCodeQueries`, `IFiscalPeriodService`, `INumberRangeService`, `ICurrencyConversionService`, `IOrgAssignmentValidator` |
| **BusinessPartner** | `mdm` | BP identity, roles, addresses, communication, identification, tax numbers, banks, relationships, company-code data, customer facet, vendor facet, sales-area, purchasing-org, credit profile | `IBusinessPartnerQueries`, `IReconciliationAccountResolver`, `IPaymentTermsResolver`, events `BpRoleAssigned`, `BpBlocked` |
| **Finance** | `fin` | Chart of accounts *values* (G/L master), universal journal, open items, clearing, customer/vendor invoices, payments, dunning, FX valuation, financial statement versions, period close | `IJournalQueries`, `IOpenItemQueries`, events `DocumentPosted`, `DocumentReversed`, `ItemsCleared` |
| **Controlling** | `co` | Cost centers + hierarchy, profit centers + hierarchy, internal orders, activity types, statistical key figures, allocation cycles (distribution/assessment), settlement rules, CO plan data, commitments | `ICostObjectValidator`, `ICoDerivationService`, events `OrderSettled`, `CycleExecuted` |
| **Assets** | `fin` (`fin.Asset*`) | Asset classes, asset master + sub-assets, depreciation areas/keys, asset transactions, depreciation postings, asset history | `IAssetQueries`, events `AssetCapitalized`, `AssetRetired` |
| **Workflow** | `wf` | Workflow definitions, rules (amount/company/type thresholds), instances, steps, approvals, delegation, substitution, escalation, notifications | `IApprovalService`, `IApprovalInboxQueries`, events `ApprovalRequested/Granted/Rejected` |
| **Security** | `sec` | Users, profiles, roles, permissions, authorization objects/fields/values, T-code registry, sessions, login history, password history, substitutions, SoD rules | `IAuthorizationService`, `IOrgScopeProvider`, `ITransactionCodeRegistry` |
| **DataDictionary** | `cfg` | Domains, data elements, tables, structures, views, foreign keys, indexes, search helps, lock objects, change log, activation requests | `IDictionaryMetadataProvider` (labels, help, F4, field rules — consumed by *every* UI and by SE16N) |
| **TableBrowser** | — (reads via dictionary) | Query variants, layouts, execution audit | `ITableBrowserService` |
| **Customization** | `cfg`, `ext`, `z` | Custom table definitions, custom field definitions, change requests, deployment history, generated migrations | `ICustomFieldProvider` (extension values for host aggregates) |
| **Reporting** | `rpt` | Report catalogue, parameter definitions, saved layouts/variants, drill-down maps, export renderers | `IReportRunner` |
| **Integration** | `intg` | API keys/clients, webhooks + delivery log, inbound message log, import jobs, bank statement staging, external ID mapping | `IEventPublisher`, `IWebhookDispatcher` |
| **Audit** | `audit` | Immutable audit log, field-level change history, query audit, sensitive-access log | `IAuditWriter`, `IAuditQueries` |

## 2.3 Communication rules

**1. In-process query contracts (synchronous, same transaction).**
`Finance` needs the reconciliation account for a BP + company code during
posting. It calls `IReconciliationAccountResolver` — an interface defined in
`Erp.Modules.BusinessPartner.Contracts` and implemented inside the BP module.
Finance never touches `mdm.*` tables and holds no EF reference to BP entities.

**2. Integration events (asynchronous, after commit, via outbox).**
`DocumentPosted` is written to `intg.OutboxMessage` **inside** the posting
transaction and dispatched afterwards by a Hangfire worker. Consumers: Reporting
(cache invalidation), Integration (webhooks), Workflow (follow-on tasks),
SignalR notifier. A failed consumer never rolls back a posting.

**3. Commands (synchronous, cross-module write, same transaction).**
Only downstream-to-nowhere modules accept these: e.g. Finance asks Workflow to
`StartApproval(documentRef, ruleContext)` during a park-and-submit. Because
Workflow is downstream of Finance and never calls back, no cycle is created.

**4. Referencing across modules.**
By **identifier + type**, never by navigation property. A journal line stores
`BusinessPartnerId uniqueidentifier` and `CostCenterId uniqueidentifier`, with a
*database* foreign key (same database — so integrity is real) but **no EF
navigation** across the module boundary. Enforced by an architecture test.

> Design tension acknowledged: cross-schema FKs technically couple the modules at
> the database level. This is deliberate — referential integrity in a ledger is
> worth more than the theoretical purity of a future extraction. If a module is
> ever extracted, those FKs become the explicit contract to break.

## 2.4 Anatomy of a module (required contents, §3)

Every module ships:

```
Erp.Modules.<Name>/
  Contracts/        public interfaces + DTOs other modules may reference
  Domain/           entities, value objects, aggregates, domain services, events
  Application/      commands, queries, handlers, validators, permissions
  Infrastructure/   EF configurations, repositories, Dapper queries, migrations
  Api/              controller/endpoint group registered by Erp.Api
  Permissions.cs    the module's permission constants + T-code links
tests/
  Erp.Modules.<Name>.UnitTests/
  Erp.Modules.<Name>.IntegrationTests/
```

A module is "done" for its milestone only when all seven exist and its slice of
the §23 mandatory-rule tests passes.

## 2.5 Designed-for-later modules

The brief requires that Purchasing, Inventory, Sales, Production, Payroll,
Projects, Maintenance, and Quality Management be addable **without redesigning
the financial core**. Three mechanisms make that true:

1. **The posting engine takes a generic `PostingRequest`** with a `SourceModule`
   discriminator and an `AccountingPrinciple`/`DocumentType` — a goods receipt or
   a payroll result is just another document type with its own account
   determination rules, not new posting code.
2. **`fin.JournalEntryLine` already carries the dimensions those modules need**
   (plant, business area, functional area, segment, profit center, internal
   order, partner company) plus a reserved *account-assignment object* pair
   (`AccountAssignmentType`, `AccountAssignmentId`) so a future WBS element,
   production order, or maintenance order needs **no column addition**.
3. **Account determination is table-driven** (`cfg.AccountDetermination`), keyed
   by *(chart of accounts, transaction key, valuation/account-modifier)* — new
   modules register new transaction keys (e.g. `GBB`, `BSX`-equivalents named
   originally) rather than hard-coding accounts.

Quantities on the journal line (`Quantity decimal(23,6)`, `BaseUnitOfMeasure`)
are present from day one for exactly this reason, even though Phase-1 modules
rarely populate them.
