# S/4HANA-Inspired ERP — Table Catalogue (Phase 2, step 1)

This folder is the **data dictionary** for the web-based ERP described in
`ERP_S4HANA_Inspired_Development_Prompt5.txt`: every table, every field, every
SQL Server data type.

It is an **original design**. SAP S/4HANA table names (`ACDOCA`, `BKPF`,
`BUT000`, …) appear only in the *reference* line of each table so a functional
consultant can recognise the concept — no SAP metadata, code or content is
reproduced. See [`10_sap_reference_mapping.md`](10_sap_reference_mapping.md)
for the concept-to-concept mapping.

## Contents

| File | Schema(s) | Tables |
|------|-----------|--------|
| [`01_enterprise_structure.md`](01_enterprise_structure.md) | `org` | Tenant, company, company code, plant, controlling area, segment … |
| [`02_configuration.md`](02_configuration.md) | `cfg` | Chart of accounts, fiscal year, periods, document types, number ranges, currencies, tax, terms |
| [`03_data_dictionary.md`](03_data_dictionary.md) | `cfg` | SE11 metadata (domains, data elements, tables, views, search helps) + SE16N variants + custom-object framework |
| [`04_business_partner.md`](04_business_partner.md) | `mdm` | Business Partner and its roles, plus G/L account and bank master |
| [`05_financial_accounting.md`](05_financial_accounting.md) | `fin` | Universal journal, open items, clearing, AR, AP, payments, dunning, balances |
| [`06_asset_accounting.md`](06_asset_accounting.md) | `fin` | Asset classes, master, depreciation areas/keys, transactions, values |
| [`07_controlling.md`](07_controlling.md) | `co` | Cost elements/centres, profit centres, activity types, internal orders, allocations, settlement, plan |
| [`08_workflow_security_audit.md`](08_workflow_security_audit.md) | `wf`, `sec`, `audit` | Approval workflow, users/roles/authorization objects, audit log, change documents |
| [`09_reporting_integration.md`](09_reporting_integration.md) | `rpt`, `intg` | Report definitions, layouts, outbox/inbox, webhooks, idempotency, bank statement import |
| [`10_sap_reference_mapping.md`](10_sap_reference_mapping.md) | — | Real S/4HANA tables and key fields → tables in this design |
| [`table_catalogue.csv`](table_catalogue.csv) | all | Flat `Schema,Table,Field,DataType,Key,Nullable,Description` export (includes expanded) |

The catalogue is the source of truth for the physical artefacts. After editing
any file here, regenerate all three:

```bash
python3 tools/generate_table_catalogue.py   # markdown  -> table_catalogue.csv
python3 tools/generate_sql_ddl.py           # CSV       -> database/s4hana/*.sql
python3 tools/generate_ef_core.py           # CSV       -> backend/ErpS4.Database
python3 tools/validate_generated_sql.py     # parse + referential + name checks
```

| Artefact | Location |
|----------|----------|
| SQL Server DDL, foreign keys, indexes, dictionary seed | [`database/s4hana/`](../../database/s4hana/) |
| EF Core entities, configurations, DbContext | [`backend/ErpS4.Database/`](../../backend/ErpS4.Database/README.md) |

## Reading a table entry

```
### `fin.JournalEntryHeader`
**Universal journal document header** · reference: `BKPF`

| Field Name | Data Type | Key | Null | Description |
```

* **Key** — `PK` primary key, `AK` alternate (business) key, `FK` foreign key,
  `IX` indexed. A field can be several (`PK,FK`).
* **Null** — `no` = `NOT NULL`.
* An `*include*` row pulls in one of the standard column groups below; the CSV
  export expands them into real fields.

## Standard column groups (includes)

Reused instead of repeating on ~150 tables. `#AUDIT` is on every table,
`#TENANT` on every tenant-dependent table (the equivalent of SAP's client
field `MANDT`).

### `#TENANT`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `TenantId` | `int` | FK,IX | no | Owning tenant — every query is filtered by it |

### `#AUDIT`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `CreatedAt` | `datetime2(3)` | | no | Creation timestamp (UTC) |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user name |
| `ModifiedAt` | `datetime2(3)` | | yes | Last change timestamp (UTC) |
| `ModifiedBy` | `nvarchar(64)` | | yes | Last changing user name |
| `RowVersion` | `rowversion` | | no | Optimistic concurrency token |
| `IsActive` | `bit` | | no | Soft-delete / active flag (master + config only) |

### `#VALIDITY`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `ValidFrom` | `date` | IX | no | First day the record is valid |
| `ValidTo` | `date` | IX | no | Last day valid (`9999-12-31` = open ended) |

Posted financial documents (`fin.JournalEntry*`, `fin.AssetTransaction`,
`fin.DepreciationPosting`, `co.ControllingPosting`) carry `#AUDIT` **without**
`IsActive` and **without** `RowVersion` — they are immutable: never updated,
never deleted, only reversed.

## Domain catalogue

Every field below is typed from one of these domains. Change the domain, and
every field that uses it changes with it — this is what `cfg.DictionaryDomain`
stores at runtime.

| Domain | SQL Server type | Length / precision | Used for |
|--------|-----------------|--------------------|----------|
| `D_ID` | `bigint` | identity | Surrogate primary keys |
| `D_TENANT` | `int` | | Tenant discriminator |
| `D_CODE2` | `nvarchar(2)` | 2 | Posting key, language, period variant keys |
| `D_CODE3` | `nvarchar(3)` | 3 | Country, unit of measure, currency-ish short keys |
| `D_CODE4` | `nvarchar(4)` | 4 | Company code, plant, controlling area, document type, business area |
| `D_CODE5` | `nvarchar(5)` | 5 | Currency key, exchange-rate type |
| `D_CODE10` | `nvarchar(10)` | 10 | G/L account, BP number, cost centre, order number, asset number |
| `D_CODE20` | `nvarchar(20)` | 20 | Long configuration keys, external references |
| `D_TEXT40` | `nvarchar(40)` | 40 | Short descriptions, names, search terms |
| `D_TEXT60` | `nvarchar(60)` | 60 | Medium descriptions |
| `D_TEXT255` | `nvarchar(255)` | 255 | Long text, line-item text, notes |
| `D_TEXTMAX` | `nvarchar(max)` | — | Documentation, JSON payloads, generated SQL |
| `D_AMOUNT` | `decimal(19,4)` | 19,4 | All financial amounts |
| `D_QUANTITY` | `decimal(23,6)` | 23,6 | Quantities |
| `D_RATE` | `decimal(23,6)` | 23,6 | Exchange rates, allocation factors |
| `D_PERCENT` | `decimal(9,4)` | 9,4 | Percentages, tax rates |
| `D_YEAR` | `smallint` | | Fiscal year (`YYYY`) |
| `D_PERIOD` | `tinyint` | | Fiscal period `1..16` |
| `D_SEQ` | `int` | | Line/item/sequence numbers |
| `D_DATE` | `date` | | Business dates (posting, document, due, validity) |
| `D_TIMESTAMP` | `datetime2(3)` | | System timestamps, always UTC |
| `D_FLAG` | `bit` | | Yes/no indicators |
| `D_STATUS` | `nvarchar(20)` | 20 | Status enumerations (`Draft`, `Posted`, …) |
| `D_UUID` | `uniqueidentifier` | | Correlation / idempotency / external ids |
| `D_ROWVER` | `rowversion` | | Concurrency token |

## Design rules applied throughout

1. **Surrogate + business key.** Every table has `Id` (`bigint identity`) as
   PK and a unique business key (`TenantId` + the SAP-style code) as `AK`.
   Foreign keys point at `Id`, never at mutable codes.
2. **Tenant isolation.** `TenantId` is the first column of every unique index
   and of every foreign key on tenant-dependent tables.
3. **Amounts always come in threes.** Journal lines carry document, local
   (company-code), and group currency amounts plus their currency keys, so no
   report ever has to convert at read time.
4. **Time dependency** uses `#VALIDITY`, never a "current record" flag.
5. **Posted documents are immutable.** No `UPDATE` path exists; correction is
   by reversal document referencing the original.
6. **Every configuration and master table is change-logged** into
   `audit.ChangeDocumentHeader` / `audit.ChangeDocumentItem`.
7. **No business logic in triggers.** Change documents are written by the
   application's `SaveChanges` interceptor inside the same transaction.

## Table count by schema

**228 tables, 4 379 fields** (counts produced by the CSV generator, includes
expanded).

| Schema | Purpose | Tables |
|--------|---------|-------:|
| `org` | Enterprise structure | 22 |
| `cfg` | Configuration + SE11 / SE16N / customizing framework | 75 |
| `mdm` | Business Partner and other master data | 24 |
| `fin` | Financial accounting (G/L, AR, AP) and asset accounting | 36 |
| `co` | Controlling | 24 |
| `wf` | Workflow and approval | 7 |
| `sec` | Users and security | 19 |
| `audit` | Audit and change documents | 5 |
| `rpt` | Reporting metadata | 5 |
| `intg` | Integration and API management | 11 |
| **Total** | | **228** |

By catalogue file: `01` 22 · `02` 45 · `03` 30 · `04` 24 · `05` 24 · `06` 12 ·
`07` 24 · `08` 31 · `09` 16.
