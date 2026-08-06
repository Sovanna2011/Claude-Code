# 03 — Data Dictionary, Table Browser, Customization (`cfg`)

Three related frameworks:

* **SE11-like Data Dictionary** — the metadata that describes every table,
  field, domain and data element in this catalogue.
* **SE16N-like Table Browser** — authorised, read-only, saved-variant querying.
* **Custom object framework** — `Z*` tables and `ZZ*` fields with a controlled
  Draft → Approve → Migrate → Production lifecycle.

Rule enforced by the design: the dictionary **describes and generates
reviewed migrations**; it never executes ad-hoc DDL from a browser session.

---

## 3.1 SE11 — Dictionary metadata

### `cfg.DictionaryObject`
**Registry of every dictionary object and its activation status**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ObjectType` | `nvarchar(20)` | AK | no | `Table`, `View`, `Structure`, `Domain`, `DataElement`, `SearchHelp`, `LockObject`, `Index` |
| `ObjectName` | `nvarchar(64)` | AK | no | Object name, e.g. `fin.JournalEntryLine`, `ZCANE_CONTRACT` |
| `ShortDescription` | `nvarchar(255)` | | no | Description shown in the object list |
| `Package` | `nvarchar(40)` | | yes | Development package / module |
| `Status` | `nvarchar(20)` | | no | `Draft`, `Checked`, `Active`, `Inactive`, `Deprecated` |
| `IsCustomObject` | `bit` | | no | `Z*` / `Y*` object |
| `NamespacePrefix` | `nvarchar(4)` | | yes | `Z`, `Y`, `ZZ` |
| `ResponsibleUser` | `nvarchar(64)` | | yes | Owner |
| `LastActivatedAt` | `datetime2(3)` | | yes | Last successful activation (UTC) |
| `LastActivatedBy` | `nvarchar(64)` | | yes | Activating user |
| `ActiveVersion` | `int` | | no | Version currently active |
| `DraftVersion` | `int` | | no | Highest draft version |
| `DocumentationText` | `nvarchar(max)` | | yes | Object documentation |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryDomain`
**Domain — technical type, length, value range**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DomainName` | `nvarchar(64)` | AK | no | Domain name, e.g. `D_AMOUNT` |
| `ShortDescription` | `nvarchar(255)` | | no | Description |
| `DataType` | `nvarchar(20)` | | no | Logical type: `CHAR`, `NUMC`, `DEC`, `INT`, `DATE`, `TIME`, `BOOL`, `GUID`, `CLOB` |
| `SqlType` | `nvarchar(40)` | | no | Generated SQL Server type, e.g. `decimal(19,4)` |
| `Length` | `int` | | yes | Field length |
| `DecimalPlaces` | `tinyint` | | yes | Number of decimals |
| `IsSigned` | `bit` | | no | Negative values allowed |
| `OutputLength` | `int` | | yes | Display length |
| `ConversionRoutine` | `nvarchar(20)` | | yes | Conversion exit, e.g. leading-zero handling |
| `CaseSensitive` | `bit` | | no | Lower case permitted |
| `ValueTableName` | `nvarchar(64)` | | yes | Value (check) table |
| `HasFixedValues` | `bit` | | no | Fixed value list maintained |
| `LowerLimit` | `nvarchar(40)` | | yes | Lower interval limit |
| `UpperLimit` | `nvarchar(40)` | | yes | Upper interval limit |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryDomainValue`
**Fixed value or value range of a domain**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DictionaryDomainId` | `bigint` | AK,FK | no | Owning domain |
| `LowValue` | `nvarchar(40)` | AK | no | Fixed value or interval start |
| `HighValue` | `nvarchar(40)` | | yes | Interval end (blank = single value) |
| `Description` | `nvarchar(60)` | | no | Value text |
| `LanguageCode` | `nvarchar(2)` | AK,FK | no | Text language |
| `DisplayOrder` | `int` | | no | Order in drop-downs |
| `IsDefault` | `bit` | | no | Proposed value |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryDataElement`
**Data element — business meaning and labels on top of a domain**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DataElementName` | `nvarchar(64)` | AK | no | Data element name, e.g. `DE_COMPANY_CODE` |
| `DictionaryDomainId` | `bigint` | FK | no | Underlying domain |
| `ShortDescription` | `nvarchar(255)` | | no | Description |
| `ShortLabel` | `nvarchar(10)` | | yes | Short field label |
| `MediumLabel` | `nvarchar(20)` | | yes | Medium field label |
| `LongLabel` | `nvarchar(40)` | | yes | Long field label |
| `HeaderLabel` | `nvarchar(55)` | | yes | Column header |
| `SearchHelpId` | `bigint` | FK | yes | Attached search help |
| `SearchHelpParameter` | `nvarchar(64)` | | yes | Export parameter of the search help |
| `ParameterId` | `nvarchar(20)` | | yes | User default (SET/GET parameter equivalent) |
| `DocumentationText` | `nvarchar(max)` | | yes | F1 help text |
| `IsChangeDocumentRelevant` | `bit` | | no | Changes are logged in change documents |
| `IsPersonalData` | `bit` | | no | Personal data — masking and retention apply |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryTable`
**Table definition and technical settings**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SchemaName` | `nvarchar(20)` | AK | no | Database schema, e.g. `fin` |
| `TableName` | `nvarchar(64)` | AK | no | Table name |
| `ShortDescription` | `nvarchar(255)` | | no | Description |
| `TableCategory` | `nvarchar(20)` | | no | `Transparent`, `Configuration`, `Master`, `Transaction`, `Text`, `Custom` |
| `DeliveryClass` | `nvarchar(1)` | | no | `A` application, `C` customizing, `S` system, `L` temporary |
| `MaintenanceType` | `nvarchar(20)` | | no | `NotAllowed`, `Display`, `Maintain` |
| `IsTenantDependent` | `bit` | | no | Carries `TenantId` |
| `IsCompanyCodeDependent` | `bit` | | no | Carries `CompanyCodeId` |
| `SizeCategory` | `tinyint` | | no | Expected volume class `0..9` |
| `BufferingType` | `nvarchar(20)` | | no | `None`, `FullyBuffered`, `GenericKey`, `SingleRecord` |
| `IsLogged` | `bit` | | no | Table changes written to change documents |
| `IsImmutable` | `bit` | | no | Posted data — updates and deletes rejected |
| `AuthorizationGroup` | `nvarchar(4)` | | yes | Table authorization group for SE16N |
| `PrimaryKeyFields` | `nvarchar(255)` | | no | Comma-separated PK field list |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryTableField`
**Field of a table** — this is the metadata behind this whole catalogue

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DictionaryTableId` | `bigint` | AK,FK | no | Owning table |
| `FieldName` | `nvarchar(64)` | AK | no | Field name |
| `FieldPosition` | `int` | | no | Column order |
| `DataElementId` | `bigint` | FK | yes | Data element (null for include-generated fields) |
| `DomainName` | `nvarchar(64)` | | yes | Domain resolved from the data element |
| `SqlType` | `nvarchar(40)` | | no | Physical SQL Server type |
| `Length` | `int` | | yes | Length |
| `DecimalPlaces` | `tinyint` | | yes | Decimals |
| `IsKey` | `bit` | | no | Part of the primary key |
| `IsRequired` | `bit` | | no | `NOT NULL` |
| `IsIdentity` | `bit` | | no | Identity / auto-increment |
| `DefaultValue` | `nvarchar(255)` | | yes | Default expression |
| `CheckTableName` | `nvarchar(64)` | | yes | Check table for input validation |
| `ForeignKeyId` | `bigint` | FK | yes | Foreign key definition |
| `SearchHelpId` | `bigint` | FK | yes | Field-level search help |
| `CurrencyReferenceField` | `nvarchar(64)` | | yes | Field holding the currency of an amount |
| `UnitReferenceField` | `nvarchar(64)` | | yes | Field holding the unit of a quantity |
| `IncludeName` | `nvarchar(64)` | | yes | Include the field came from, e.g. `#AUDIT` |
| `IsCustomField` | `bit` | | no | `ZZ*` customer field |
| `IsMasked` | `bit` | | no | Masked in the table browser and exports |
| `ShortDescription` | `nvarchar(255)` | | no | Field description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryStructure`
**Structure (no database table behind it)**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `StructureName` | `nvarchar(64)` | AK | no | Structure name, e.g. `#AUDIT` |
| `ShortDescription` | `nvarchar(255)` | | no | Description |
| `StructureType` | `nvarchar(20)` | | no | `Include`, `Dto`, `ScreenStructure` |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryStructureField`
**Field of a structure**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DictionaryStructureId` | `bigint` | AK,FK | no | Owning structure |
| `FieldName` | `nvarchar(64)` | AK | no | Field name |
| `FieldPosition` | `int` | | no | Order |
| `DataElementId` | `bigint` | FK | yes | Data element |
| `SqlType` | `nvarchar(40)` | | no | Physical type |
| `IsRequired` | `bit` | | no | Mandatory |
| `ShortDescription` | `nvarchar(255)` | | no | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryView`
**View definition (join or projection)**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SchemaName` | `nvarchar(20)` | AK | no | Schema |
| `ViewName` | `nvarchar(64)` | AK | no | View name |
| `ShortDescription` | `nvarchar(255)` | | no | Description |
| `ViewType` | `nvarchar(20)` | | no | `Database`, `Projection`, `Maintenance`, `Help` |
| `BaseTableName` | `nvarchar(64)` | | no | Primary table |
| `JoinDefinition` | `nvarchar(max)` | | yes | Join conditions in JSON |
| `SelectionCondition` | `nvarchar(max)` | | yes | Fixed `WHERE` condition |
| `IsReadOnly` | `bit` | | no | Read-only view |
| `GeneratedSql` | `nvarchar(max)` | | yes | Generated `CREATE VIEW` statement |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryViewField`
**Field of a view**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DictionaryViewId` | `bigint` | AK,FK | no | Owning view |
| `ViewFieldName` | `nvarchar(64)` | AK | no | Field name in the view |
| `SourceTableName` | `nvarchar(64)` | | no | Source table |
| `SourceFieldName` | `nvarchar(64)` | | no | Source field |
| `FieldPosition` | `int` | | no | Order |
| `IsKey` | `bit` | | no | Key field of the view |
| `AggregateFunction` | `nvarchar(10)` | | yes | `SUM`, `MIN`, `MAX`, `COUNT` |
| `ShortDescription` | `nvarchar(255)` | | yes | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryForeignKey`
**Foreign key / check table relationship**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ForeignKeyName` | `nvarchar(128)` | AK | no | Constraint name (a SQL Server identifier) |
| `SourceSchemaName` | `nvarchar(20)` | | no | Source schema |
| `SourceTableName` | `nvarchar(64)` | | no | Source (dependent) table |
| `TargetSchemaName` | `nvarchar(20)` | | no | Target schema |
| `TargetTableName` | `nvarchar(64)` | | no | Target (check) table |
| `Cardinality` | `nvarchar(5)` | | no | `1:1`, `1:N`, `C:N`, `1:CN` |
| `ForeignKeyType` | `nvarchar(20)` | | no | `KeyFields`, `NonKeyFields`, `TextTable` |
| `CheckRequired` | `bit` | | no | Input checked against the target table |
| `OnDeleteAction` | `nvarchar(20)` | | no | `NoAction`, `Cascade`, `SetNull`, `Restrict` |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryForeignKeyField`
**Field pair of a foreign key**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DictionaryForeignKeyId` | `bigint` | AK,FK | no | Owning foreign key |
| `SourceFieldName` | `nvarchar(64)` | AK | no | Field in the source table |
| `TargetFieldName` | `nvarchar(64)` | | no | Field in the target table |
| `ConstantValue` | `nvarchar(40)` | | yes | Constant instead of a field |
| `FieldPosition` | `int` | | no | Order in the composite key |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryIndex`
**Secondary index definition**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DictionaryTableId` | `bigint` | AK,FK | no | Owning table |
| `IndexName` | `nvarchar(128)` | AK | no | Index name (a SQL Server identifier) |
| `ShortDescription` | `nvarchar(255)` | | yes | Purpose of the index |
| `IsUnique` | `bit` | | no | Unique index |
| `IsClustered` | `bit` | | no | Clustered index |
| `IsColumnStore` | `bit` | | no | Columnstore index (reporting tables) |
| `FilterPredicate` | `nvarchar(255)` | | yes | Filtered index predicate |
| `IncludedColumns` | `nvarchar(255)` | | yes | Included (non-key) columns |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryIndexField`
**Field of a secondary index**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DictionaryIndexId` | `bigint` | AK,FK | no | Owning index |
| `FieldName` | `nvarchar(64)` | AK | no | Indexed field |
| `FieldPosition` | `int` | | no | Position in the index |
| `SortDirection` | `nvarchar(4)` | | no | `ASC`, `DESC` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionarySearchHelp`
**Search help (F4)**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SearchHelpName` | `nvarchar(64)` | AK | no | Search help name |
| `ShortDescription` | `nvarchar(255)` | | no | Description |
| `SearchHelpType` | `nvarchar(20)` | | no | `Elementary`, `Collective` |
| `SelectionMethod` | `nvarchar(64)` | | no | Table or view read |
| `TextTableName` | `nvarchar(64)` | | yes | Text table for descriptions |
| `DialogType` | `nvarchar(20)` | | no | `ImmediateDisplay`, `DialogWithRestriction` |
| `HotKey` | `nvarchar(1)` | | yes | Shortcut in collective search helps |
| `MaxHits` | `int` | | no | Maximum rows returned |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionarySearchHelpParameter`
**Import / export parameter of a search help**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DictionarySearchHelpId` | `bigint` | AK,FK | no | Owning search help |
| `ParameterName` | `nvarchar(64)` | AK | no | Parameter name |
| `DataElementId` | `bigint` | FK | yes | Data element |
| `IsImport` | `bit` | | no | Import parameter |
| `IsExport` | `bit` | | no | Export parameter |
| `IsListField` | `bit` | | no | Shown in the hit list |
| `IsSelectionField` | `bit` | | no | Shown in the restriction dialog |
| `ListPosition` | `int` | | yes | Column order in the hit list |
| `SelectionPosition` | `int` | | yes | Order in the restriction dialog |
| `DefaultValue` | `nvarchar(255)` | | yes | Default value |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryLockObject`
**Lock object for application-level locking**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `LockObjectName` | `nvarchar(64)` | AK | no | Lock object name, e.g. `E_JOURNAL` |
| `ShortDescription` | `nvarchar(255)` | | no | Description |
| `PrimaryTableName` | `nvarchar(64)` | | no | Table locked |
| `LockMode` | `nvarchar(10)` | | no | `Exclusive`, `Shared`, `ExclusiveNonCumulative` |
| `LockArgumentFields` | `nvarchar(255)` | | no | Fields forming the lock argument |
| `LockTimeoutSeconds` | `int` | | no | Timeout before the lock is refused |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DictionaryChangeLog`
**Every dictionary change, with before/after definition**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DictionaryObjectId` | `bigint` | FK,IX | no | Object changed |
| `VersionNumber` | `int` | | no | Version produced by the change |
| `Action` | `nvarchar(20)` | | no | `Create`, `Change`, `Copy`, `Activate`, `Deactivate`, `Delete` |
| `ChangedAt` | `datetime2(3)` | IX | no | Timestamp (UTC) |
| `ChangedBy` | `nvarchar(64)` | | no | User |
| `OldDefinition` | `nvarchar(max)` | | yes | Previous definition (JSON) |
| `NewDefinition` | `nvarchar(max)` | | yes | New definition (JSON) |
| `MigrationScriptId` | `bigint` | FK | yes | Generated migration |
| `ApprovedBy` | `nvarchar(64)` | | yes | Approver |
| `ApprovedAt` | `datetime2(3)` | | yes | Approval timestamp (UTC) |
| `TransportRequestId` | `bigint` | FK | yes | Change request carrying the object |
| `CorrelationId` | `uniqueidentifier` | | yes | Request correlation id |

---

## 3.2 SE16N — Table browser

### `cfg.TableAuthorizationGroup`
**Authorization group controlling table access** · reference: `TBRG`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AuthorizationGroup` | `nvarchar(4)` | AK | no | Group key, e.g. `FI01`, `SEC0` |
| `Name` | `nvarchar(60)` | | no | Description |
| `IsSystemProtected` | `bit` | | no | System/security tables — browser access always denied |
| `AllowExport` | `bit` | | no | Export permitted for this group |
| `MaxRowsPerQuery` | `int` | | no | Row cap applied to browser queries |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.BrowserVariant`
**Saved selection variant / layout of the table browser**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `VariantName` | `nvarchar(40)` | AK | no | Variant name |
| `SchemaName` | `nvarchar(20)` | AK | no | Schema of the queried object |
| `ObjectName` | `nvarchar(64)` | AK | no | Table or view queried |
| `OwnerUserId` | `bigint` | AK,FK | no | Owning user |
| `IsShared` | `bit` | | no | Visible to other users |
| `IsDefault` | `bit` | | no | Opened by default |
| `Description` | `nvarchar(255)` | | yes | Description |
| `SortDefinition` | `nvarchar(255)` | | yes | Sort fields and directions |
| `PageSize` | `int` | | no | Rows per page |
| `ShowTechnicalNames` | `bit` | | no | Display field names instead of labels |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.BrowserVariantField`
**Selected output field of a variant**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BrowserVariantId` | `bigint` | AK,FK | no | Owning variant |
| `FieldName` | `nvarchar(64)` | AK | no | Field |
| `DisplayOrder` | `int` | | no | Column order |
| `ColumnWidth` | `int` | | yes | Column width in pixels |
| `IsVisible` | `bit` | | no | Column displayed |
| `AggregateFunction` | `nvarchar(10)` | | yes | `SUM`, `COUNT`, `AVG` for totals rows |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.BrowserVariantFilter`
**Selection criterion of a variant (range table)**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BrowserVariantId` | `bigint` | AK,FK | no | Owning variant |
| `FieldName` | `nvarchar(64)` | AK | no | Filtered field |
| `LineNumber` | `int` | AK | no | Criterion line |
| `SignIndicator` | `nvarchar(1)` | | no | `I` include, `E` exclude |
| `Operator` | `nvarchar(10)` | | no | `EQ`, `NE`, `GT`, `GE`, `LT`, `LE`, `BT`, `CP`, `SW`, `EW`, `NULL`, `NN`, `IN`, `NI` |
| `LowValue` | `nvarchar(255)` | | yes | Value / interval start |
| `HighValue` | `nvarchar(255)` | | yes | Interval end |
| `ValueList` | `nvarchar(max)` | | yes | Values for `IN` / `NI` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.BrowserQueryLog`
**Audit trail of every table-browser query**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UserId` | `bigint` | FK,IX | no | Executing user |
| `ExecutedAt` | `datetime2(3)` | IX | no | Execution timestamp (UTC) |
| `SchemaName` | `nvarchar(20)` | | no | Schema queried |
| `ObjectName` | `nvarchar(64)` | IX | no | Table or view queried |
| `SelectedFields` | `nvarchar(max)` | | yes | Fields returned |
| `FilterJson` | `nvarchar(max)` | | yes | Parameterised filter (values redacted where masked) |
| `RowsReturned` | `int` | | no | Number of rows returned |
| `RowLimitApplied` | `int` | | no | Row cap in force |
| `WasExported` | `bit` | | no | Result exported |
| `ExportFormat` | `nvarchar(10)` | | yes | `XLSX`, `CSV`, `PDF` |
| `DurationMs` | `int` | | no | Execution time in milliseconds |
| `WasTruncated` | `bit` | | no | Result cut off by the row limit |
| `CorrelationId` | `uniqueidentifier` | | yes | Request correlation id |
| `ClientIpAddress` | `nvarchar(45)` | | yes | Request origin |

---

## 3.3 Custom objects (`Z*`, `Y*`, `ZZ*`)

### `cfg.CustomTable`
**User-defined table definition**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CustomTableName` | `nvarchar(64)` | AK | no | Name, must start `Z` or `Y`, e.g. `ZCANE_CONTRACT` |
| `SchemaName` | `nvarchar(20)` | | no | Target schema (`cus` by default) |
| `ShortDescription` | `nvarchar(255)` | | no | Description |
| `TableCategory` | `nvarchar(20)` | | no | `Master`, `TransactionHeader`, `TransactionItem`, `Configuration`, `Translation`, `History`, `Relationship` |
| `IsTenantDependent` | `bit` | | no | Include `TenantId` |
| `IsCompanyCodeDependent` | `bit` | | no | Include `CompanyCodeId` |
| `HasValidityPeriod` | `bit` | | no | Include `ValidFrom` / `ValidTo` |
| `HasAuditFields` | `bit` | | no | Include `#AUDIT` |
| `AuthorizationGroup` | `nvarchar(4)` | FK | yes | Authorization group |
| `NumberRangeObjectId` | `bigint` | FK | yes | Number range for generated keys |
| `IsApiExposed` | `bit` | | no | Generate a REST API |
| `IsBrowserVisible` | `bit` | | no | Visible in the table browser |
| `LifecycleStatus` | `nvarchar(20)` | | no | `Draft`, `Validated`, `ImpactReviewed`, `Approved`, `Development`, `Test`, `Quality`, `Production` |
| `DictionaryObjectId` | `bigint` | FK | yes | Dictionary registry entry |
| `Status` | `nvarchar(20)` | | no | Activation status |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.CustomTableField`
**Field of a user-defined table**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CustomTableId` | `bigint` | AK,FK | no | Owning custom table |
| `FieldName` | `nvarchar(64)` | AK | no | Field name |
| `FieldPosition` | `int` | | no | Column order |
| `DataElementId` | `bigint` | FK | yes | Data element |
| `DictionaryDomainId` | `bigint` | FK | yes | Domain (when no data element is used) |
| `SqlType` | `nvarchar(40)` | | no | Physical SQL Server type |
| `Length` | `int` | | yes | Length |
| `DecimalPlaces` | `tinyint` | | yes | Decimals |
| `IsKey` | `bit` | | no | Part of the key |
| `IsRequired` | `bit` | | no | Mandatory |
| `IsUnique` | `bit` | | no | Unique constraint |
| `DefaultValue` | `nvarchar(255)` | | yes | Default |
| `CheckTableName` | `nvarchar(64)` | | yes | Check table |
| `SearchHelpId` | `bigint` | FK | yes | Search help |
| `ValidationExpression` | `nvarchar(255)` | | yes | Server-side validation rule |
| `LabelEn` | `nvarchar(60)` | | no | English label |
| `LabelKm` | `nvarchar(60)` | | yes | Khmer label |
| `ShortDescription` | `nvarchar(255)` | | yes | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.CustomFieldDefinition`
**`ZZ*` extension field added to a standard entity**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `HostEntity` | `nvarchar(64)` | AK | no | `BusinessPartner`, `Asset`, `JournalEntryHeader`, `JournalEntryLine`, `CostCenter`, `ProfitCenter`, `InternalOrder` |
| `FieldName` | `nvarchar(64)` | AK | no | Field name, must start `ZZ`, e.g. `ZZOldAssetNumber` |
| `LabelEn` | `nvarchar(60)` | | no | English label |
| `LabelKm` | `nvarchar(60)` | | yes | Khmer label |
| `ShortDescription` | `nvarchar(255)` | | yes | Description |
| `DataElementId` | `bigint` | FK | yes | Data element |
| `SqlType` | `nvarchar(40)` | | no | Physical type of the generated column |
| `Length` | `int` | | yes | Length |
| `DecimalPlaces` | `tinyint` | | yes | Decimals |
| `IsRequired` | `bit` | | no | Mandatory on the screen |
| `DefaultValue` | `nvarchar(255)` | | yes | Default |
| `SearchHelpId` | `bigint` | FK | yes | Search help |
| `ValidationExpression` | `nvarchar(255)` | | yes | Server-side validation rule |
| `DisplayOrder` | `int` | | no | Position on the screen |
| `ScreenSection` | `nvarchar(40)` | | no | Tab / group the field appears in |
| `AuthorizationGroup` | `nvarchar(4)` | FK | yes | Authorization group |
| `IsReportingEnabled` | `bit` | | no | Available as a report dimension |
| `IsApiExposed` | `bit` | | no | Returned by the REST API |
| `IsPhysicalColumn` | `bit` | | no | Generated as a real column (required for reportable fields) |
| `LifecycleStatus` | `nvarchar(20)` | | no | Lifecycle stage |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.CustomFieldValue`
**Value store for non-physical `ZZ*` fields (never used for accounting-relevant data)**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CustomFieldDefinitionId` | `bigint` | AK,FK | no | Field definition |
| `HostEntity` | `nvarchar(64)` | AK | no | Owning entity type |
| `HostEntityId` | `bigint` | AK | no | Owning entity key |
| `TextValue` | `nvarchar(255)` | | yes | Text value |
| `NumericValue` | `decimal(23,6)` | | yes | Numeric value |
| `DateValue` | `date` | | yes | Date value |
| `BooleanValue` | `bit` | | yes | Boolean value |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.CustomObjectRequest`
**Change request carrying custom objects through the landscape**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RequestNumber` | `nvarchar(20)` | AK | no | Request number |
| `Title` | `nvarchar(255)` | | no | Short text |
| `RequestType` | `nvarchar(20)` | | no | `Workbench`, `Customizing` |
| `Status` | `nvarchar(20)` | | no | `Draft`, `Validated`, `ImpactReviewed`, `Approved`, `Released`, `ImportedTest`, `ImportedQuality`, `ImportedProduction`, `RolledBack` |
| `RequestedBy` | `nvarchar(64)` | | no | Requester |
| `DeveloperUserName` | `nvarchar(64)` | | yes | Developer |
| `ApprovedBy` | `nvarchar(64)` | | yes | Approver |
| `ApprovedAt` | `datetime2(3)` | | yes | Approval timestamp (UTC) |
| `TargetEnvironment` | `nvarchar(20)` | | no | `Development`, `Test`, `Quality`, `Production` |
| `ImpactAssessment` | `nvarchar(max)` | | yes | Affected objects and risk notes |
| `RollbackPlan` | `nvarchar(max)` | | yes | Rollback instructions |
| `ReleasedAt` | `datetime2(3)` | | yes | Release timestamp (UTC) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.CustomObjectRequestItem`
**Object contained in a change request**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CustomObjectRequestId` | `bigint` | AK,FK | no | Owning request |
| `ObjectType` | `nvarchar(20)` | AK | no | Dictionary object type |
| `ObjectName` | `nvarchar(64)` | AK | no | Object name |
| `ObjectVersion` | `int` | | no | Version transported |
| `OperationType` | `nvarchar(20)` | | no | `Create`, `Change`, `Delete` |
| `DependencyList` | `nvarchar(max)` | | yes | Objects that must be imported first |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.MigrationScript`
**Reviewed migration generated by activation — executed only by CI/CD**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `MigrationName` | `nvarchar(128)` | AK | no | Migration name, e.g. `20260805_AddZCaneContract` |
| `DictionaryObjectId` | `bigint` | FK | yes | Object that produced the migration |
| `CustomObjectRequestId` | `bigint` | FK | yes | Change request |
| `UpScript` | `nvarchar(max)` | | no | Forward DDL |
| `DownScript` | `nvarchar(max)` | | yes | Rollback DDL |
| `GeneratedAt` | `datetime2(3)` | | no | Generation timestamp (UTC) |
| `GeneratedBy` | `nvarchar(64)` | | no | Generating user |
| `ReviewedBy` | `nvarchar(64)` | | yes | Reviewer |
| `ReviewedAt` | `datetime2(3)` | | yes | Review timestamp (UTC) |
| `AppliedEnvironment` | `nvarchar(20)` | | yes | Environment where it was applied |
| `AppliedAt` | `datetime2(3)` | | yes | Application timestamp (UTC) |
| `Checksum` | `nvarchar(64)` | | no | SHA-256 of the script — tamper detection |
| `Status` | `nvarchar(20)` | | no | `Generated`, `Reviewed`, `Approved`, `Applied`, `Failed`, `RolledBack` |
| *include* | `#AUDIT` | | | Standard audit columns |
