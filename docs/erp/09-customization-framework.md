# 9. User-Defined Tables & Custom-Field Framework

Customer- and partner-specific extensions without forking the product, and
without turning the financial core into a bag of untyped JSON.

## 9.1 The central decision: generated real columns, not EAV, not JSON

§12.3 states it directly: *"Do not store important reportable or accounting custom
values only in an unvalidated JSON column."* The design honours that.

| Option | Verdict |
|---|---|
| EAV (`ObjectId, FieldName, Value` rows) | **Rejected.** Untyped, unindexable, joins explode, reporting becomes string-casting, referential integrity impossible |
| JSON column only | **Rejected as primary storage.** No FK, weak typing, poor index economics for reporting, no dictionary-driven validation at the database level |
| **Generated real columns in an extension table per host entity** | **Chosen.** Typed, indexable, FK-capable, reportable, SE16N-visible, and still isolated from core tables so core migrations never collide with customer fields |

A JSON column (`ExtensionData nvarchar(max)`) is still present on extension
tables — but only for **non-reportable, non-accounting** annotations, explicitly
marked as such in the field designer, and never usable in a validation rule or a
posting. If a field is flagged *reportable* or *accounting-relevant*, the designer
forces a real column.

## 9.2 Namespaces (§12)

| Prefix | Owner | Example |
|---|---|---|
| `Z*` | Customer-developed objects | `ZSUPPLIER_RATING`, `ZLAND_MASTER`, `ZCANE_CONTRACT` |
| `Y*` | Partner-developed objects | `YIMPLEMENTATION_LOG` |
| `ZZ*` | Custom fields on standard objects | `ZZOldAssetNumber`, `ZZLandArea`, `ZZContractReference` |

Enforced by the dictionary validator: a custom object outside `Z`/`Y` is rejected,
and **a custom object may never take a core name** — one of the mandatory tests
(§23: *"Custom tables cannot use reserved core names"*). Core namespaces (`fin`,
`org`, `mdm`, `co`, `cfg`, `sec`, `wf`, `rpt`, `intg`, `audit` object names) are
in a protected list; custom tables are physically created in the `z` schema
(`z.ZSUPPLIER_RATING`) so even a naming mistake cannot shadow a core table.

## 9.3 Custom table types (§12.1)

| Type | Required standard fields | Extra behaviour |
|---|---|---|
| Master data | audit set + `ValidFrom/To` optional | Soft delete allowed, search help auto-generated |
| Transaction header | audit set + `CompanyCodeId`, `FiscalYear`, `DocumentNumber`, `Status` | Number range required; immutability optional |
| Transaction item | audit set + FK to header + `LineNumber` | Cascade rules defined |
| Configuration | audit set + `IsActive` | Cached, change-logged, transportable |
| Translation | `LanguageCode` + FK to the translated object | Composite key includes language |
| History | audit set + `ValidFrom/To` + `ChangedBy` | Insert-only; no update path |
| Relationship | audit set + `FromId`, `ToId`, `RelationshipType`, validity | Directionality flag, inverse type |

## 9.4 Standard field set (§12.2)

Every custom table automatically receives:

```
TenantId     uniqueidentifier  NOT NULL   (when tenant-dependent — default yes)
Id           uniqueidentifier  NOT NULL   PK, default NEWSEQUENTIALID()
CreatedAt    datetime2(7)      NOT NULL   UTC
CreatedBy    uniqueidentifier  NOT NULL
ModifiedAt   datetime2(7)      NULL       UTC
ModifiedBy   uniqueidentifier  NULL
RowVersion   rowversion        NOT NULL
IsActive     bit               NOT NULL   DEFAULT 1
```

Transaction tables additionally receive `CompanyCodeId`, `FiscalYear`,
`DocumentNumber`, `Status`, and optionally `ValidFrom`/`ValidTo`. The designer
adds these automatically and they cannot be removed — they are what make custom
tables work with tenant filtering, audit, optimistic concurrency, and the table
browser without special-casing.

## 9.5 Custom Table Designer

Configuration surface (§12.2), all persisted as dictionary objects so a custom
table is *the same kind of thing* as a core table:

```
Technical name · Description (multilingual) · Table category · Authorization group
Primary key definition · Fields (name, domain/data element, type, length, decimals,
  default, required, unique) · Foreign keys + check tables · Search helps · Indexes
Audit logging on/off · Effective dating · Company-code dependency · Tenant dependency
```

Generated automatically once the definition is active:

| Artefact | Content |
|---|---|
| SQL table | `z.<NAME>` with constraints and indexes |
| EF Core mapping | Dynamic model built from dictionary metadata at startup (`IModelCustomizer`), so no code generation is needed for CRUD |
| REST API | `/api/v1/custom-objects/{tableName}` — list, get, create, update, deactivate — with dictionary-driven validation |
| UI | Generic list + form screens rendered from metadata; a T-code may be assigned to the table so it appears in the command box |
| Permissions | `Z_<NAME>_DISPLAY` / `_MAINTAIN`, auto-registered |
| Table browser | Immediately browsable, subject to its authorization group |

## 9.6 Custom fields on standard objects (§12.3)

Extensible hosts (approved list): **Business Partner, Asset, Journal header,
Journal line, Cost center, Profit center, Internal order**.

Storage: a **1:1 extension table per host**, in the `ext` schema:

```
ext.BusinessPartnerExt   (BusinessPartnerId PK/FK, TenantId, ZZ… columns…, ExtensionData json, RowVersion)
ext.AssetExt             (AssetId PK/FK, …)
ext.JournalEntryHeaderExt(JournalEntryHeaderId PK/FK, …)
ext.JournalEntryLineExt  (JournalEntryLineId PK/FK, …)     ← see the caution below
ext.CostCenterExt / ext.ProfitCenterExt / ext.InternalOrderExt
```

Why a side table rather than columns on the core table:

- Core migrations and customer extensions never collide.
- A core table's row width and index design stay under our control — important
  for `fin.JournalEntryLine`, which is the hottest table in the system.
- Extensions can be deployed/rolled back independently.
- The 1:1 FK keeps the join trivial and lets EF map it as an owned/optional
  navigation, so `bp.Ext.ZZLandArea` reads naturally in code.

> **Caution documented in the framework:** custom fields on `JournalEntryLine`
> multiply by the largest table in the system. The designer warns on this host,
> requires a justification on the change request, and mandates a filtered index
> if the field is marked reportable.

Per-field configuration (§12.3): field name, labels, description, data type,
length, decimals, required, default, search help, value list, validation rule,
display order, screen section, authorization, **reporting availability**, **API
availability**. The last two are real switches: a field with
`AvailableInReporting = true` is exposed to the report engine and SE16N; a field
with `AvailableInApi = true` is added to the OpenAPI schema of the host resource.

**Custom fields participate in posting validation.** A `ZZ`-field on the journal
line marked required-for-document-type X is checked at step 7 of the posting
engine — the field-status mechanism reads the dictionary, so extensions are
first-class in validation, not an afterthought.

## 9.7 Deployment lifecycle (§12.4)

```
Draft → Validate → Impact review → Approve → Generate migration
      → Deploy to DEV → Test → Transport to QA → Approve → Deploy to PROD
```

`cfg.ChangeRequest` is the transport unit — the object of record for the whole
lifecycle:

| Table | Content |
|---|---|
| `cfg.ChangeRequest` | Number, title, description, requester, type (Dictionary/CustomTable/CustomField/Config), status, target release |
| `cfg.ChangeRequestObject` | The dictionary objects included, with their versions |
| `cfg.ChangeRequestApproval` | Approver, decision, timestamp, comment (SoD: approver ≠ requester) |
| `cfg.ChangeRequestDeployment` | Environment, migration artefact, applied at/by, duration, result, rollback artefact |
| `cfg.ObjectVersion` | Full snapshot of each object version for compare/rollback |
| `cfg.ChangeRequestDependency` | Other requests that must ship first |

Maintained per §12.4: object version, change request, developer, approval status,
environment, deployment history, rollback migration, dependencies.

## 9.8 Validation rules for custom objects

Beyond the dictionary checks (§7.5):

1. Reserved-name check against the core object list **and** against SQL Server
   reserved words.
2. Field count and row-width limits (SQL Server 8060-byte in-row limit is
   computed at design time and reported before activation).
3. A custom FK may reference a core table, but a **core table may never reference
   a custom table** — that would make the core undeployable without the extension.
4. Cascade deletes to core data are forbidden; custom rows referencing core rows
   use `NO ACTION` with an application-level guard.
5. A custom field flagged accounting-relevant requires: a data element, a
   validation rule, and a report availability decision — no silent free text on
   an accounting object.
6. Custom objects cannot be created in the `fin` schema at all.

## 9.9 Example — `ZCANE_CONTRACT` (agri-industry sugar-cane supply contract)

Shows the framework end to end:

```
Table       z.ZCANE_CONTRACT          category: Transaction header
Key         TenantId, Id              alt key: TenantId, CompanyCodeId, DocumentNumber
Fields      DocumentNumber   ZDE_DOCNUM       (number range ZCANE, KSS-2026-CN-#######)
            BusinessPartnerId FK → mdm.BusinessPartner  (check table, search help BP)
            PlantId          FK → org.Plant
            ZZLandArea       ZDE_AREA_HA      decimal(23,6)   required, reportable
            ContractCurrency ZDE_CURRKEY      char(3)          check table cfg.Currency
            ContractValue    ZDE_AMOUNT       decimal(19,4)    reportable
            SeasonYear       ZDE_FISCALYEAR   int              required
            Status           ZDOM_CONTRACT_ST fixed: Draft/Active/Fulfilled/Cancelled
Items       z.ZCANE_CONTRACT_ITEM     category: Transaction item (FK to header, LineNumber)
Extension   ext.BusinessPartnerExt.ZZSupplierRating  → check table z.ZSUPPLIER_RATING
Permissions Z_ZCANE_CONTRACT_DISPLAY / _MAINTAIN     T-code ZCN01 / ZCN03
Posting     Contract fulfilment posts through IPostingEngine with document type ZC
            and account determination key ZCANE — no bespoke posting code.
```

The last line is the point of the whole framework: an industry-specific process
gets its own master data, screens, and authorizations, while every financial
consequence still flows through the one posting engine into the one journal.
