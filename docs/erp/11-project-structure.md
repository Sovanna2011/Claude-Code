# 11. Project Structure

Exact paths, as required by §26.6. The ERP lives in a new `erp/` tree; the
existing HR module (`backend/`, `frontend/`, `database/`) is untouched.

## 11.1 Repository layout

```
erp/
├── Erp.sln
├── Directory.Build.props                # shared: net10.0, nullable, warnings-as-errors, analyzers
├── Directory.Packages.props             # central package version management
├── .editorconfig
├── docker-compose.yml                   # api + jobs + web + sqlserver + redis + seq
├── docker-compose.override.yml          # local dev
├── Dockerfile.api  Dockerfile.jobs  Dockerfile.web
│
├── src/
│   ├── Erp.SharedKernel/                        ── no dependencies
│   │   ├── Primitives/                 Entity, AggregateRoot, ValueObject, Result, Error
│   │   ├── ValueObjects/               Money, CurrencyCode, ExchangeRate, DateRange,
│   │   │                               FiscalPeriodRef, DocumentNumber, OrgKey
│   │   ├── Events/                     IDomainEvent, IIntegrationEvent
│   │   ├── Abstractions/               IClock, ITenantContext, ICurrentUser
│   │   └── Guards/                     Ensure, DomainException
│   │
│   ├── Erp.Application.Abstractions/             ── MediatR contracts, behaviours' interfaces
│   │   ├── Messaging/                  ICommand, IQuery, ICommandHandler, IQueryHandler
│   │   ├── Authorization/              IRequiresAuthorization, AuthorizationRequest
│   │   ├── Persistence/                IUnitOfWork, IRepository<T>, IDbConnectionFactory
│   │   └── Events/                     IEventPublisher, IOutbox
│   │
│   ├── Erp.Posting.Domain/                       ── THE posting engine (shared by all modules)
│   │   ├── Engine/                     IPostingEngine, PostingPipeline, IPostingStep
│   │   ├── Steps/                      01…19 — one class per pipeline step
│   │   ├── Requests/                   PostingRequest, PostingLine, ReversalRequest
│   │   ├── Results/                    PostingResult, SimulationResult, PostingError
│   │   └── Rules/                      BalanceRule, PeriodRule, ReconciliationAccountRule…
│   │
│   ├── Erp.Modules.Organization/
│   │   ├── Contracts/                  ICompanyCodeQueries, IFiscalPeriodService,
│   │   │                               INumberRangeService, ICurrencyConversionService
│   │   ├── Domain/                     Tenant, Company, CompanyCode, ControllingArea,
│   │   │                               FiscalYearVariant, NumberRange, Currency…
│   │   ├── Application/                Commands/, Queries/, Validators/, Permissions.cs
│   │   ├── Infrastructure/             EF configurations, repositories, migrations
│   │   └── Api/                        CompanyCodesEndpoints, ConfigurationEndpoints…
│   │
│   ├── Erp.Modules.BusinessPartner/    (same five folders)
│   ├── Erp.Modules.Finance/            GL · AR · AP · clearing · period close · FSV
│   ├── Erp.Modules.Controlling/        cost/profit centers · internal orders · allocations
│   ├── Erp.Modules.Assets/             asset master · depreciation · transactions
│   ├── Erp.Modules.Workflow/           rules · instances · approvals · delegation
│   ├── Erp.Modules.Security/           users · roles · authorization objects · SoD
│   ├── Erp.Modules.DataDictionary/     SE11 metadata · validation · activation
│   ├── Erp.Modules.TableBrowser/       SE16N query builder · variants · layouts
│   ├── Erp.Modules.Customization/      custom tables/fields · change requests
│   ├── Erp.Modules.Reporting/          report catalogue · runners · exporters
│   ├── Erp.Modules.Integration/        webhooks · imports · bank statements · API keys
│   ├── Erp.Modules.Audit/              audit writer · audit queries
│   │
│   ├── Erp.Infrastructure/                       ── cross-cutting infrastructure
│   │   ├── Persistence/                ErpDbContext, interceptors (audit, rowversion,
│   │   │                               tenant filter), UnitOfWork, execution strategy
│   │   ├── Persistence/Migrations/     EF Core migrations (all modules, one history table)
│   │   ├── Dapper/                     read-only connection factory (erp_reader)
│   │   ├── Caching/                    IConfigurationCache (memory/Redis)
│   │   ├── Identity/                   Identity stores mapped to sec.*
│   │   ├── Outbox/                     OutboxWriter, OutboxDispatcher
│   │   ├── Files/                      attachment storage (disk/blob), virus-scan hook
│   │   ├── Logging/                    Serilog configuration + enrichers
│   │   └── Time/                       SystemClock (UTC)
│   │
│   ├── Erp.Api/                                  ── ASP.NET Core 10 host
│   │   ├── Program.cs                  composition root, module registration
│   │   ├── Endpoints/                  versioned route groups per module
│   │   ├── Middleware/                 CorrelationId, TenantResolution, ProblemDetails
│   │   ├── Authorization/              policy providers, org-scope handlers
│   │   ├── Hubs/                       NotificationHub (SignalR)
│   │   ├── OpenApi/                    Swagger config, examples, security schemes
│   │   ├── appsettings.json  appsettings.Development.json
│   │   └── Erp.Api.csproj
│   │
│   ├── Erp.Jobs/                                 ── Hangfire host
│   │   ├── Program.cs                  same modules, no public API surface
│   │   ├── Jobs/                       PaymentRunJob, DepreciationRunJob, AllocationJob,
│   │   │                               FxValuationJob, DunningJob, OutboxDispatchJob,
│   │   │                               RecurringEntryJob, SoDDriftJob, ArchiveJob
│   │   └── Scheduling/                 recurring job registration
│   │
│   ├── Erp.Reporting/                            ── report definitions & renderers
│   │   ├── Definitions/                trial balance, GL, aging, BS/PL, asset register…
│   │   ├── Execution/                  parameter binding, drill-down maps
│   │   └── Export/                     ExcelRenderer (ClosedXML), PdfRenderer (QuestPDF)
│   │
│   └── Erp.Integration/                          ── external adapters
│       ├── Webhooks/                   dispatcher, retry, signature
│       ├── Import/                     CSV/Excel pipelines with row-level error reports
│       ├── Banking/                    statement parsers, payment file writers
│       └── Clients/                    typed HTTP clients (Polly resilience)
│
├── tests/
│   ├── Erp.SharedKernel.UnitTests/
│   ├── Erp.Posting.UnitTests/                  balance, period, currency, derivation
│   ├── Erp.Posting.IntegrationTests/           full pipeline against real SQL Server
│   ├── Erp.Modules.<Name>.UnitTests/           one per module
│   ├── Erp.Modules.<Name>.IntegrationTests/    one per module
│   ├── Erp.Api.ContractTests/                  OpenAPI snapshot + Problem Details shape
│   ├── Erp.ArchitectureTests/                  layer/dependency/naming rules
│   ├── Erp.AccountingRuleTests/                the §23 mandatory-rule suite
│   └── Erp.E2ETests/                           Playwright against docker-compose
│
├── db/
│   ├── scripts/                        idempotent DDL export (per schema)
│   ├── seed/                           sample enterprise structure, BPs, CoA, rates
│   └── README.md                       migration & seeding instructions
│
└── docs/                               (this blueprint; API docs; process docs)
```

## 11.2 Frontend layout

```
erp/web/                                 ── React 19 + TypeScript 5 + Vite
├── package.json  tsconfig.json  vite.config.ts  .eslintrc  playwright.config.ts
├── src/
│   ├── main.tsx  App.tsx  router.tsx
│   ├── app/
│   │   ├── providers/          Auth, Tenant, Theme(light/dark), I18n, Query, SignalR
│   │   ├── layout/             AppShell, LeftNav, HeaderBar, Breadcrumbs, Footer
│   │   └── context/            CompanyCodeSelector, FiscalPeriodSelector, CurrencySelector
│   ├── design-system/          ── the design system (§Phase 4)
│   │   ├── tokens/             colour, spacing, typography, elevation, radii (light+dark)
│   │   ├── primitives/         Button, Input, Select, Checkbox, Radio, Switch, Badge…
│   │   ├── data/               DataGrid (AG Grid wrapper), Toolbar, ColumnPicker,
│   │   │                       FilterBar, SavedLayout, ExportMenu, Pagination
│   │   ├── forms/              Form, FormField, DictField (dictionary-driven), ValueHelp(F4)
│   │   ├── feedback/           Toast, Dialog, ProblemDetailsAlert, EmptyState, Skeleton
│   │   └── charts/             KPI tile, bar/line/donut (accessible palette)
│   ├── features/
│   │   ├── dashboard/          role-based home, KPI tiles, approval inbox widget
│   │   ├── command-box/        global T-code & object search (§9)
│   │   ├── configuration/      SPRO tree + generated maintenance views
│   │   ├── business-partner/   BP page (tabs), search, roles, relationships
│   │   ├── gl/                 G/L accounts (FS00), journal entry (FB50), documents (FB03)
│   │   ├── ar/                 customer invoices, incoming payments, dunning, FBL5N
│   │   ├── ap/                 vendor invoices, payments, F110, FBL1N
│   │   ├── assets/             asset master, transactions, depreciation run
│   │   ├── controlling/        cost centers, profit centers, internal orders, allocations
│   │   ├── approvals/          inbox, detail, delegate, history
│   │   ├── dictionary/         SE11 workbench
│   │   ├── table-browser/      SE16N screen
│   │   ├── customization/      custom table designer, custom field designer
│   │   ├── reports/            report launcher, viewer, drill-down, saved layouts
│   │   └── admin/              users, roles, authorizations, SoD, sessions, audit
│   ├── api/                    generated OpenAPI client + TanStack Query hooks
│   ├── lib/                    money & date formatting (user tz), zod schemas, errors
│   ├── i18n/                   en.json, km.json (Khmer), formatting rules
│   └── types/
└── tests/                      component tests (Vitest + Testing Library)
```

## 11.3 Conventions

| Concern | Convention |
|---|---|
| Namespaces | `Erp.Modules.Finance.Domain.GeneralLedger` — namespace mirrors folder |
| Commands/queries | `PostJournalEntryCommand` / `PostJournalEntryCommandHandler` / `PostJournalEntryCommandValidator` |
| EF configurations | `JournalEntryLineConfiguration : IEntityTypeConfiguration<JournalEntryLine>` |
| Endpoints | `/api/v1/{resource}` kebab-case plural; one route group class per module |
| Permissions | `MODULE.OBJECT.ACTION` → `FIN.JOURNAL.POST` |
| Tests | `MethodOrScenario_Condition_ExpectedOutcome` |
| Migrations | `yyyyMMddHHmm_<Module>_<Change>` |
| Schemas | one per module (§13 of the brief), custom objects in `z`, extensions in `ext` |
| Async | every I/O method is `async` and takes a `CancellationToken` |
| Nullability | enabled solution-wide; warnings are errors |

## 11.4 Solution build order

```
SharedKernel → Application.Abstractions → Posting.Domain
   → Modules.* (Domain → Application → Infrastructure → Api)
      → Infrastructure → Api / Jobs / Reporting / Integration
         → tests
```

`Erp.ArchitectureTests` fails the build if any project references a project it
must not (e.g. `Domain` → `Infrastructure`, or `Finance` → `BusinessPartner.Domain`
instead of `.Contracts`).
