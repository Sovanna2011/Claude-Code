# 12. Database Schema Strategy

## 12.1 Schemas

| Schema | Owner module | Contents |
|---|---|---|
| `org` | Organization | Tenant, Company, CompanyCode, BusinessArea, Plant, Branch, Location, Department, Sales/Purchasing orgs, ControllingArea, OperatingConcern, CreditControlArea, FunctionalArea, Segment |
| `cfg` | Organization / DataDictionary / Customization | Chart of accounts, fiscal & posting period variants, field status, document types, number ranges, posting keys, currencies & rates, tax, FSV, account determination, validation/substitution rules, **dictionary metadata**, change requests, browser variants |
| `mdm` | BusinessPartner | Business Partner and all facets |
| `fin` | Finance + Assets | G/L master, ledgers, universal journal, open items, clearing, invoices, payments, dunning, FX valuation, assets, depreciation, period close |
| `co` | Controlling | Cost centers, profit centers, internal orders, activity types, SKFs, allocation cycles, settlement, plan data, commitments |
| `wf` | Workflow | Definitions, rules, instances, steps, approvals, notifications, delegation |
| `sec` | Security | Users, roles, permissions, authorization objects/values, sessions, SoD, row-level policies |
| `rpt` | Reporting | Report catalogue, saved layouts/variants, materialized period snapshots (Phase 5) |
| `intg` | Integration | Outbox, inbox, webhooks + deliveries, API clients, import jobs, external ID mapping, bank staging |
| `audit` | Audit | Audit log, field changes, sensitive access, query log, denial log |
| `ext` | Customization | 1:1 extension tables for custom fields on core objects |
| `z` | Customization | Customer/partner custom tables (`Z*`, `Y*`) |

Cross-schema foreign keys are allowed and used (see [02 §2.3](02-module-boundaries.md)
for why); EF navigations across module boundaries are not.

## 12.2 Universal conventions

Every table:

```
Id           uniqueidentifier NOT NULL DEFAULT NEWSEQUENTIALID()  -- surrogate PK
TenantId     uniqueidentifier NOT NULL                            -- tenant-owned tables
CreatedAt    datetime2(7)     NOT NULL                            -- UTC
CreatedBy    uniqueidentifier NOT NULL
ModifiedAt   datetime2(7)     NULL                                -- UTC (mutable tables)
ModifiedBy   uniqueidentifier NULL
RowVersion   rowversion       NOT NULL                            -- optimistic concurrency
```

Plus, where applicable: `IsActive bit`, `ValidFrom date`, `ValidTo date`
(`9999-12-31` for open-ended), `IsDeleted bit` (permitted master data only).

**Business keys are separate from surrogate keys.** `org.CompanyCode` has
`Id uniqueidentifier` PK and `UQ(TenantId, CompanyCodeKey)`. FKs use the
surrogate; users see and search the business key. This keeps a company-code
rename possible before first posting, and keeps FK width predictable.

### Data types (§1)

| Use | Type | Note |
|---|---|---|
| Financial amounts | `decimal(19,4)` | All amount columns, all currencies |
| Quantities, exchange rates, percentages | `decimal(23,6)` | |
| Currency codes | `char(3)` | ISO 4217 |
| Org keys | `varchar(4..10)` | Company code `varchar(4)`, segment `varchar(10)` |
| Timestamps | `datetime2(7)` | **UTC always** |
| Accounting dates | `date` | Posting/document/due dates — no time, no time zone |
| Flags | `bit` | |
| Text | `nvarchar(n)` | Unicode throughout (Khmer script support is mandatory) |
| Free JSON (non-accounting only) | `nvarchar(max)` with `ISJSON` check | Never for reportable financial values |

Collation: database-level `Latin1_General_100_CI_AS_SC_UTF8` (UTF-8 enabled) so
Khmer text stores and compares correctly while remaining index-friendly.

## 12.3 Constraint strategy

| Constraint | Applied to |
|---|---|
| PK | Every table (surrogate) |
| UQ | Every business key, always including `TenantId` |
| FK | Every reference — including cross-schema; `NO ACTION` on delete by default |
| CHECK | Enumerations (`DebitCreditIndicator IN ('D','C')`), sign rules, `ValidFrom <= ValidTo`, amount-sign consistency, `ISJSON(ExtensionData) = 1` |
| Filtered UQ | "One default per parent" patterns (`WHERE IsDefault = 1`) |
| Computed | `SignedAmountLocal AS CASE WHEN DebitCreditIndicator='D' THEN AmountLocal ELSE -AmountLocal END PERSISTED` |

## 12.4 Index strategy for the journal

`fin.JournalEntryLine` is the largest and most-queried table; its indexes are a
design decision, not an afterthought.

| Index | Purpose |
|---|---|
| CLUSTERED `(TenantId, CompanyCodeId, FiscalYear, JournalEntryHeaderId, LineNumber)` | Natural document locality; range scans by company code + year |
| NC `(TenantId, LedgerId, GLAccountId, FiscalYear, FiscalPeriod) INCLUDE (SignedAmountLocal, AmountGroup)` | Trial balance, G/L balance, financial statements |
| NC `(TenantId, BusinessPartnerId, BusinessPartnerRole, OpenItemStatus, DueDate) INCLUDE (…amounts)` | Aging, open items, payment proposals, dunning |
| NC `(TenantId, CostCenterId, FiscalYear, FiscalPeriod)` filtered `WHERE CostCenterId IS NOT NULL` | Cost center reporting |
| NC `(TenantId, ProfitCenterId, FiscalYear, FiscalPeriod)` filtered | Profit center reporting |
| NC `(TenantId, InternalOrderId, FiscalYear)` filtered | Internal order reporting |
| NC `(TenantId, AssetId, FiscalYear)` filtered | Asset values |
| NC `(TenantId, PostingDate, CompanyCodeId)` | Date-range reports, journal register |
| NC `(TenantId, ClearingDocumentNumber)` filtered `WHERE ClearingDocumentNumber IS NOT NULL` | Clearing lookups |
| **Columnstore** (non-clustered, filtered to closed fiscal years) | Analytical aggregation without hurting OLTP inserts |

Filtered indexes matter here: most lines have `NULL` for most dimensions, so
filtering keeps these indexes small.

**Partitioning:** `fin.JournalEntryHeader`, `fin.JournalEntryLine`,
`audit.AuditLog`, and `intg.OutboxMessage` are partitioned by `FiscalYear`
(audit/outbox by month) from Phase 2, so year-end archival is a partition switch
rather than a delete of hundreds of millions of rows.

## 12.5 Views (design-time)

| View | Purpose |
|---|---|
| `fin.vw_GLLineItem` | Journal lines joined to account/BP/CO descriptions — the FBL3N backing view |
| `fin.vw_CustomerOpenItem` / `fin.vw_VendorOpenItem` | Open items with aging buckets |
| `fin.vw_TrialBalance` | Per ledger/account/period: opening, debit, credit, closing in local & group currency |
| `fin.vw_SubledgerReconciliation` | Subledger vs reconciliation-account balances — returns rows **only when they disagree** |
| `co.vw_CostCenterActual` / `co.vw_PlanActual` | CO reporting |
| `fin.vw_AssetValue` | APC, accumulated depreciation, NBV per asset/area/year |
| `fin.vw_IntercompanyBalance` | Per company-code pair and period; must net to zero |

These are *reporting* views; write paths never read through them.

## 12.6 Immutability of posted documents (§13, §26.8)

Four layers, because one is not enough:

1. **Domain** — no method exists to mutate a posted `JournalEntry`; the aggregate
   exposes only `Reverse(...)` and `Clear(...)`.
2. **EF Core** — a `SaveChanges` interceptor throws if any entry of type
   `JournalEntryLine`/`JournalEntryHeader` with `Status = Posted` is in
   `Modified` or `Deleted` state (except the explicitly whitelisted clearing
   fields on the line, which are the *only* mutable columns after posting:
   `OpenItemStatus`, `ClearingDocumentNumber`, `ClearingDate`, `ClearingId`,
   `DunningLevel`).
3. **Database permissions** — the application login has `INSERT`/`SELECT` on
   journal tables and `UPDATE` granted **only on those clearing columns**
   (column-level `GRANT UPDATE`); `DELETE` is not granted at all.
4. **Audit** — any attempt is logged as a high-severity event.

Note the deliberate exception: clearing *must* update the original line's status,
so the column-level grant is exactly as wide as that need and no wider. Tests
assert that an attempt to update an amount, account, or date on a posted line
fails at the database level even with the ORM bypassed.

Per §22, **no triggers carry business logic**. The only triggers permitted are
audit-integrity guards (insert-only enforcement on `audit.*`), and even those are
secondary to database permissions.

## 12.7 Migrations & seeding

- **EF Core migrations** are the single source of schema truth; `db/scripts/` is a
  generated, idempotent export for DBAs who need to review or apply DDL manually.
- One migration history table for the whole solution (modules share a database),
  with migration names prefixed by module for readability.
- **Migrations never run automatically at startup in production.** `Erp.Api`
  verifies the schema version and refuses to start on mismatch; migrations are
  applied by an explicit pipeline step (`dotnet ef database update` or the
  generated script) with a maintenance window and a rollback script prepared.
- **Seed data** is layered and idempotent:
  1. `00_system` — dictionary definitions of all core objects, authorization
     objects, T-codes, permissions, default roles
  2. `10_reference` — currencies, exchange-rate types, countries, units, languages
  3. `20_configuration` — sample enterprise structure (§24): 2 companies,
     3 company codes, chart of accounts, fiscal/posting variants, document types,
     number ranges, posting keys, tax codes, FSV
  4. `30_master_data` — G/L accounts, Business Partners (incl. one with both
     customer and vendor roles), cost centers, profit centers, internal orders,
     assets
  5. `40_demo_transactions` — opening balances, customer/vendor invoices,
     payments, an intercompany pair, foreign-currency documents, a depreciation
     run — enough to make every report non-empty on first login

Seeds 00–20 are required in every environment; 30–40 are development/demo only,
gated by configuration.

## 12.8 Performance, integrity, and operations (§22)

| Concern | Approach |
|---|---|
| Concurrency | `rowversion` everywhere; number ranges via atomic `UPDATE…OUTPUT`; 409 on conflict |
| Transactions | `READ COMMITTED SNAPSHOT` on; explicit transaction per command via `TransactionBehaviour`; short transactions; no user interaction inside one |
| Transient errors | EF Core retry + Polly on external calls; safe because postings are idempotent |
| Read scale | Dapper + read-only connection for reports; optional read replica routed by `IDbConnectionFactory(readOnly: true)` |
| Caching | Configuration + dictionary metadata; invalidated by integration event |
| Background | Hangfire for payment runs, depreciation, allocations, FX valuation, dunning, outbox, archiving |
| Archiving | Partition switch by fiscal year into `*_Archive` tables, then export; posted data is never deleted, only relocated |
| Backup/DR | Full + differential + log backups, tested restore runbook, RPO/RTO documented in the Phase-5 deployment guide |
| Monitoring | Query Store enabled; slow-query alerts; index-usage review each release; posting-engine metrics |

## 12.9 Approximate table count (Phase 2 scope)

| Schema | Tables (est.) |
|---|---|
| `org` | 22 |
| `cfg` | 58 (incl. 16 dictionary tables) |
| `mdm` | 17 |
| `fin` | 41 (incl. 13 asset tables) |
| `co` | 24 |
| `wf` | 10 |
| `sec` | 19 |
| `intg` | 9 |
| `audit` | 5 |
| `rpt` | 6 |
| `ext` | 7 |
| **Total** | **≈ 218** |

The full catalogue with every column, key, and index is the **Phase 2**
deliverable; this blueprint fixes the schemas, conventions, and the strategy they
will follow.
