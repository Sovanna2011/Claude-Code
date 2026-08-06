# 07 — Controlling (`co`)

Controlling objects are derived and updated automatically by the posting
engine: a journal line carrying a cost element plus a cost centre, internal
order or profit centre produces the matching `co.ControllingPosting` row inside
the same transaction. Plan values, allocations and settlements post through the
same path, so plan-versus-actual reporting reads one table.

---

## 7.1 Cost elements and cost centres

### `co.CostElement`
**Cost element (primary = G/L expense account, secondary = internal)** · reference: `CSKB`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `CostElement` | `nvarchar(10)` | AK | no | Cost element number |
| `CostElementCategory` | `nvarchar(2)` | | no | `1` primary, `11` revenue, `12` sales deduction, `21` internal settlement, `41` overhead, `42` assessment, `43` activity allocation |
| `IsPrimary` | `bit` | | no | Primary cost element (has a G/L account) |
| `GLAccountId` | `bigint` | FK | yes | G/L account for primary cost elements |
| `Name` | `nvarchar(40)` | | no | Description |
| `Description` | `nvarchar(60)` | | yes | Long description |
| `FunctionalAreaId` | `bigint` | FK | yes | Default functional area |
| `IsRecordQuantity` | `bit` | | no | Quantity recorded with the posting |
| `UnitOfMeasure` | `nvarchar(3)` | FK | yes | Unit for the quantity |
| `DefaultAccountAssignmentType` | `nvarchar(20)` | | yes | `CostCenter`, `InternalOrder`, `None` |
| `DefaultCostCenterId` | `bigint` | FK | yes | Default cost centre |
| `DefaultInternalOrderId` | `bigint` | FK | yes | Default internal order |
| *include* | `#VALIDITY` | | | Validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.CostCenterCategory`
**Cost centre category** · reference: `TKA05`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CategoryCode` | `nvarchar(1)` | AK | no | Category key, e.g. `F` production, `V` sales, `H` service |
| `Name` | `nvarchar(40)` | | no | Description |
| `IsActualPrimaryCostsLocked` | `bit` | | no | Lock actual primary costs |
| `IsActualSecondaryCostsLocked` | `bit` | | no | Lock actual secondary costs |
| `IsActualRevenuesLocked` | `bit` | | no | Lock actual revenues |
| `IsCommitmentUpdateLocked` | `bit` | | no | Lock commitment update |
| `IsPlanPrimaryCostsLocked` | `bit` | | no | Lock plan primary costs |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.CostCenter`
**Cost centre master** · reference: `CSKS`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `CostCenter` | `nvarchar(10)` | AK | no | Cost centre number |
| `Name` | `nvarchar(40)` | | no | Short name |
| `Description` | `nvarchar(60)` | | yes | Description |
| `CostCenterCategoryId` | `bigint` | FK | no | Category |
| `HierarchyNodeId` | `bigint` | FK,IX | no | Node in the standard hierarchy |
| `CompanyCodeId` | `bigint` | FK | no | Company code |
| `BusinessAreaId` | `bigint` | FK | yes | Business area |
| `ProfitCenterId` | `bigint` | FK,IX | yes | Profit centre |
| `FunctionalAreaId` | `bigint` | FK | yes | Functional area |
| `SegmentId` | `bigint` | FK | yes | Segment |
| `PlantId` | `bigint` | FK | yes | Plant |
| `DepartmentCode` | `nvarchar(10)` | | yes | Department |
| `ResponsiblePersonPartnerId` | `bigint` | FK | yes | Person responsible (employee BP) |
| `ResponsibleUserName` | `nvarchar(64)` | | yes | Responsible system user |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Cost centre currency |
| `IsLockedForActualPrimaryCosts` | `bit` | | no | Lock actual primary postings |
| `IsLockedForActualSecondaryCosts` | `bit` | | no | Lock actual secondary postings |
| `IsLockedForActualRevenues` | `bit` | | no | Lock actual revenues |
| `IsLockedForPlanning` | `bit` | | no | Lock planning |
| `IsLockedForCommitments` | `bit` | | no | Lock commitments |
| `RecordQuantity` | `bit` | | no | Quantities recorded |
| `AddressId` | `bigint` | FK | yes | Address |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| *include* | `#VALIDITY` | | | Master data validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.HierarchyNode`
**Node of a standard cost-centre or profit-centre hierarchy** · reference: `SETNODE`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `HierarchyType` | `nvarchar(20)` | AK | no | `CostCenter`, `ProfitCenter`, `CostElement`, `InternalOrder` |
| `HierarchyId` | `nvarchar(12)` | AK | no | Hierarchy id |
| `NodeCode` | `nvarchar(20)` | AK | no | Node key |
| `ParentNodeId` | `bigint` | FK | yes | Parent node |
| `NodeName` | `nvarchar(60)` | | no | Node description |
| `NodeLevel` | `int` | | no | Depth in the hierarchy |
| `DisplayOrder` | `int` | | no | Order among siblings |
| `HierarchyPath` | `nvarchar(500)` | IX | no | Materialised path for fast roll-ups |
| `IsStandardHierarchy` | `bit` | | no | Node of the standard hierarchy |
| `IsLeaf` | `bit` | | no | Objects may be assigned to this node |
| *include* | `#VALIDITY` | | | Validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.ProfitCenter`
**Profit centre master** · reference: `CEPC`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `ProfitCenter` | `nvarchar(10)` | AK | no | Profit centre number |
| `Name` | `nvarchar(40)` | | no | Short name |
| `Description` | `nvarchar(60)` | | yes | Description |
| `HierarchyNodeId` | `bigint` | FK,IX | no | Node in the standard hierarchy |
| `CompanyCodeId` | `bigint` | FK | yes | Company code (blank = cross-company) |
| `SegmentId` | `bigint` | FK,IX | yes | Segment derived from this profit centre |
| `BusinessAreaId` | `bigint` | FK | yes | Business area |
| `ResponsiblePersonPartnerId` | `bigint` | FK | yes | Person responsible |
| `ResponsibleUserName` | `nvarchar(64)` | | yes | Responsible system user |
| `DepartmentCode` | `nvarchar(10)` | | yes | Department |
| `IsLockedForPosting` | `bit` | | no | Locked for postings |
| `IsLockedForPlanning` | `bit` | | no | Locked for planning |
| `AddressId` | `bigint` | FK | yes | Address |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| *include* | `#VALIDITY` | | | Master data validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.ProfitCenterAssignment`
**Assignment of an object to a profit centre with validity**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ObjectType` | `nvarchar(20)` | AK | no | `CostCenter`, `InternalOrder`, `Asset`, `GLAccount`, `Plant` |
| `ObjectId` | `bigint` | AK | no | Assigned object |
| `ProfitCenterId` | `bigint` | AK,FK | no | Profit centre |
| `CompanyCodeId` | `bigint` | FK | yes | Company code the assignment applies to |
| *include* | `#VALIDITY` | | | Assignment validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.ActivityType`
**Activity type** · reference: `CSLA`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `ActivityType` | `nvarchar(6)` | AK | no | Activity type key |
| `Name` | `nvarchar(40)` | | no | Description |
| `UnitOfMeasure` | `nvarchar(3)` | FK | no | Activity unit |
| `ActivityTypeCategory` | `nvarchar(1)` | | no | `1` manual entry / manual allocation, `2` indirect determination, `3` manual entry / no allocation |
| `AllocationCostElementId` | `bigint` | FK | no | Secondary cost element used for allocation |
| `PriceIndicator` | `nvarchar(1)` | | no | `1` plan price automatic, `2` plan price political, `3` manual |
| `IsBlocked` | `bit` | | no | Blocked |
| *include* | `#VALIDITY` | | | Validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.ActivityPrice`
**Planned or actual price of an activity type per cost centre and period**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `CostCenterId` | `bigint` | AK,FK | no | Sending cost centre |
| `ActivityTypeId` | `bigint` | AK,FK | no | Activity type |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period |
| `PlanVersion` | `nvarchar(3)` | AK | no | Plan version, e.g. `000` |
| `FixedPrice` | `decimal(19,4)` | | no | Fixed portion of the price |
| `VariablePrice` | `decimal(19,4)` | | no | Variable portion of the price |
| `PriceUnit` | `int` | | no | Price unit (price per n units) |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `PlannedQuantity` | `decimal(23,6)` | | yes | Planned activity quantity |
| `CapacityQuantity` | `decimal(23,6)` | | yes | Capacity |
| `IsActualPrice` | `bit` | | no | Actual price calculated at period end |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.StatisticalKeyFigure`
**Statistical key figure** · reference: `TKA03`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `StatisticalKeyFigure` | `nvarchar(6)` | AK | no | Key figure code, e.g. `HEADCNT`, `AREA` |
| `Name` | `nvarchar(40)` | | no | Description |
| `UnitOfMeasure` | `nvarchar(3)` | FK | no | Unit |
| `KeyFigureCategory` | `nvarchar(1)` | | no | `1` fixed value, `2` total value |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.StatisticalKeyFigureValue`
**Posted statistical key figure value**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `StatisticalKeyFigureId` | `bigint` | AK,FK | no | Key figure |
| `ObjectType` | `nvarchar(20)` | AK | no | `CostCenter`, `InternalOrder`, `ProfitCenter` |
| `ObjectId` | `bigint` | AK | no | Receiving object |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period |
| `PlanVersion` | `nvarchar(3)` | AK | no | Plan version (`000` = actual) |
| `IsPlan` | `bit` | AK | no | Plan or actual value |
| `Quantity` | `decimal(23,6)` | | no | Value |
| `UnitOfMeasure` | `nvarchar(3)` | FK | no | Unit |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 7.2 Internal orders

### `co.InternalOrderType`
**Internal order type** · reference: `T003O`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `OrderType` | `nvarchar(4)` | AK | no | Order type key |
| `Name` | `nvarchar(40)` | | no | Description |
| `OrderCategory` | `nvarchar(20)` | | no | `Overhead`, `Investment`, `Accrual`, `RevenueBearing`, `Production` |
| `NumberRangeObjectId` | `bigint` | FK | no | Number range object |
| `NumberRangeCode` | `nvarchar(2)` | | no | Number range interval |
| `IsStatisticalOnly` | `bit` | | no | Statistical orders only |
| `IsRevenuePostingAllowed` | `bit` | | no | Revenue postings allowed |
| `IsCommitmentManagementActive` | `bit` | | no | Commitment management active |
| `IsBudgetControlActive` | `bit` | | no | Availability control active |
| `SettlementProfile` | `nvarchar(6)` | | yes | Settlement profile |
| `PlanningProfile` | `nvarchar(6)` | | yes | Planning profile |
| `BudgetProfile` | `nvarchar(6)` | | yes | Budget profile |
| `StatusProfile` | `nvarchar(8)` | | yes | Status profile |
| `IsMasterDataFieldsRequired` | `bit` | | no | Responsible cost centre mandatory |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.InternalOrder`
**Internal order master** · reference: `AUFK`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `OrderNumber` | `nvarchar(12)` | AK | no | Internal order number (not an internal counter) |
| `OrderTypeId` | `bigint` | FK | no | Order type |
| `Description` | `nvarchar(60)` | IX | no | Order description |
| `LongText` | `nvarchar(max)` | | yes | Long text |
| `CompanyCodeId` | `bigint` | FK | no | Company code |
| `BusinessAreaId` | `bigint` | FK | yes | Business area |
| `PlantId` | `bigint` | FK | yes | Plant |
| `ResponsibleCostCenterId` | `bigint` | FK | no | Responsible cost centre |
| `RequestingCostCenterId` | `bigint` | FK | yes | Requesting cost centre |
| `ProfitCenterId` | `bigint` | FK | yes | Profit centre |
| `SegmentId` | `bigint` | FK | yes | Segment |
| `FunctionalAreaId` | `bigint` | FK | yes | Functional area |
| `ResponsiblePersonPartnerId` | `bigint` | FK | yes | Person responsible |
| `ApplicantName` | `nvarchar(60)` | | yes | Applicant |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Order currency |
| `IsStatistical` | `bit` | | no | Statistical order — no settlement |
| `IsRevenueBearing` | `bit` | | no | Revenue postings allowed |
| `SystemStatus` | `nvarchar(20)` | IX | no | `Created`, `Released`, `TechnicallyCompleted`, `Closed`, `Locked`, `MarkedForDeletion` |
| `UserStatus` | `nvarchar(20)` | | yes | User status |
| `WorkStartDate` | `date` | | yes | Planned start |
| `WorkEndDate` | `date` | | yes | Planned finish |
| `ActualStartDate` | `date` | | yes | Actual start |
| `ActualEndDate` | `date` | | yes | Actual finish |
| `EstimatedTotalCost` | `decimal(19,4)` | | yes | Estimated cost |
| `InvestmentReason` | `nvarchar(2)` | | yes | Investment reason |
| `AssetId` | `bigint` | FK | yes | Asset under construction linked to the order |
| `IsBudgetControlActive` | `bit` | | no | Availability control active |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.InternalOrderBudget`
**Annual and overall budget with availability control** · reference: `BPJA`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `InternalOrderId` | `bigint` | AK,FK | no | Internal order |
| `FiscalYear` | `smallint` | AK | no | Fiscal year (`0` = overall budget) |
| `BudgetVersion` | `nvarchar(3)` | AK | no | Budget version |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `OriginalBudget` | `decimal(19,4)` | | no | Original budget |
| `SupplementAmount` | `decimal(19,4)` | | no | Supplements |
| `ReturnAmount` | `decimal(19,4)` | | no | Returns |
| `CurrentBudget` | `decimal(19,4)` | | no | Current budget |
| `AssignedAmount` | `decimal(19,4)` | | no | Actuals + commitments |
| `AvailableAmount` | `decimal(19,4)` | | no | Remaining budget |
| `UsagePercent` | `decimal(9,4)` | | no | Utilisation |
| `WarningThresholdPercent` | `decimal(9,4)` | | yes | Threshold for a warning |
| `ErrorThresholdPercent` | `decimal(9,4)` | | yes | Threshold that blocks postings |
| `IsBudgetExceeded` | `bit` | | no | Budget exceeded |
| `ReleasedAmount` | `decimal(19,4)` | | yes | Budget released for use |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.SettlementRule`
**Settlement rule of a sender object** · reference: `COBRB`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SenderObjectType` | `nvarchar(20)` | AK | no | `InternalOrder`, `AssetUnderConstruction`, `Project` |
| `SenderObjectId` | `bigint` | AK,FK | no | Sender object |
| `RuleNumber` | `int` | AK | no | Distribution rule number |
| `ReceiverType` | `nvarchar(20)` | | no | `CostCenter`, `GLAccount`, `Asset`, `InternalOrder`, `ProfitCenter`, `Project` |
| `ReceiverObjectId` | `bigint` | | no | Receiver key |
| `SettlementType` | `nvarchar(3)` | | no | `PER` periodic, `FUL` full settlement |
| `Percentage` | `decimal(9,4)` | | yes | Share in percent |
| `EquivalenceNumber` | `int` | | yes | Equivalence number for proportional split |
| `AmountLimit` | `decimal(19,4)` | | yes | Maximum amount to settle |
| `SettlementCostElementId` | `bigint` | FK | yes | Settlement cost element |
| `ValidFromPeriod` | `tinyint` | | yes | First period the rule applies |
| `ValidFromYear` | `smallint` | | yes | First year the rule applies |
| `ValidToPeriod` | `tinyint` | | yes | Last period |
| `ValidToYear` | `smallint` | | yes | Last year |
| `IsBlocked` | `bit` | | no | Rule blocked |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.SettlementDocument`
**Executed settlement** · reference: settlement documents from `KO88`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `SettlementDocumentNumber` | `nvarchar(20)` | AK | no | Settlement document number |
| `SenderObjectType` | `nvarchar(20)` | | no | Sender object type |
| `SenderObjectId` | `bigint` | FK,IX | no | Sender object |
| `FiscalYear` | `smallint` | | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | | no | Settlement period |
| `PostingDate` | `date` | | no | Posting date |
| `SettlementType` | `nvarchar(3)` | | no | `PER`, `FUL` |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `TotalSettledAmount` | `decimal(19,4)` | | no | Total settled |
| `IsTestRun` | `bit` | | no | Test run |
| `IsReversed` | `bit` | | no | Reversed |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Accounting document created |
| `Status` | `nvarchar(20)` | | no | `Simulated`, `Posted`, `Reversed`, `Failed` |
| `ExecutedAt` | `datetime2(3)` | | no | Execution timestamp (UTC) |
| `ExecutedBy` | `nvarchar(64)` | | no | Executing user or job |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.SettlementDocumentItem`
**Amount settled to one receiver**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SettlementDocumentId` | `bigint` | AK,FK | no | Settlement document |
| `ItemNumber` | `int` | AK | no | Item number |
| `SettlementRuleId` | `bigint` | FK | yes | Rule applied |
| `ReceiverType` | `nvarchar(20)` | | no | Receiver type |
| `ReceiverObjectId` | `bigint` | | no | Receiver key |
| `CostElementId` | `bigint` | FK | no | Settlement cost element |
| `SettledAmount` | `decimal(19,4)` | | no | Amount settled |
| `SettledAmountInLocalCurrency` | `decimal(19,4)` | | no | Amount in local currency |
| `Quantity` | `decimal(23,6)` | | yes | Quantity settled |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 7.3 Postings, allocations, planning

### `co.ControllingPosting`
**CO line item — actual and plan** · reference: `COEP`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `ControllingDocumentNumber` | `nvarchar(30)` | AK | no | CO document number - the FI number plus a `-CO###` suffix, so it needs more room than the FI number itself |
| `LineItemNumber` | `int` | AK | no | Line number |
| `FiscalYear` | `smallint` | IX | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | IX | no | Period |
| `PostingDate` | `date` | IX | no | Posting date |
| `PlanVersion` | `nvarchar(3)` | | no | Version (`000` = actual) |
| `IsPlan` | `bit` | | no | Plan record |
| `ValueType` | `nvarchar(2)` | | no | `04` actual, `01` plan, `11` statistical actual, `21` commitment |
| `ObjectType` | `nvarchar(20)` | IX | no | `CostCenter`, `InternalOrder`, `ProfitCenter`, `Asset`, `Project` |
| `ObjectId` | `bigint` | IX | no | Account assignment object |
| `PartnerObjectType` | `nvarchar(20)` | | yes | Partner object type (sender/receiver) |
| `PartnerObjectId` | `bigint` | | yes | Partner object |
| `CostElementId` | `bigint` | FK,IX | no | Cost element |
| `ActivityTypeId` | `bigint` | FK | yes | Activity type |
| `CompanyCodeId` | `bigint` | FK | no | Company code |
| `ProfitCenterId` | `bigint` | FK | yes | Profit centre |
| `FunctionalAreaId` | `bigint` | FK | yes | Functional area |
| `SegmentId` | `bigint` | FK | yes | Segment |
| `DebitCreditIndicator` | `nvarchar(1)` | | no | `S` debit, `H` credit |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Transaction currency |
| `AmountInTransactionCurrency` | `decimal(19,4)` | | no | Amount in transaction currency |
| `AmountInControllingAreaCurrency` | `decimal(19,4)` | | no | Amount in controlling area currency |
| `AmountInCompanyCodeCurrency` | `decimal(19,4)` | | no | Amount in company code currency |
| `FixedAmount` | `decimal(19,4)` | | yes | Fixed cost portion |
| `Quantity` | `decimal(23,6)` | | yes | Quantity |
| `UnitOfMeasure` | `nvarchar(3)` | FK | yes | Unit of measure |
| `TransactionType` | `nvarchar(20)` | | no | `Primary`, `Distribution`, `Assessment`, `ActivityAllocation`, `Settlement`, `Reposting`, `Surcharge` |
| `ReferenceDocumentNumber` | `nvarchar(20)` | | yes | Source document |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Related FI document |
| `AllocationRunId` | `bigint` | FK | yes | Allocation run that produced the line |
| `SettlementDocumentId` | `bigint` | FK | yes | Settlement document |
| `LineItemText` | `nvarchar(255)` | | yes | Item text |
| `IsReversed` | `bit` | | no | Reversed |
| `CreatedAt` | `datetime2(3)` | | no | Creation timestamp (UTC) |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user or job |

---

### `co.ControllingTotal`
**Period totals per object, cost element and version** · reference: `COSP` / `COSS`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `ObjectType` | `nvarchar(20)` | AK | no | Object type |
| `ObjectId` | `bigint` | AK | no | Object |
| `CostElementId` | `bigint` | AK,FK | no | Cost element |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period |
| `PlanVersion` | `nvarchar(3)` | AK | no | Version |
| `ValueType` | `nvarchar(2)` | AK | no | Actual / plan / commitment |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `TotalAmount` | `decimal(19,4)` | | no | Total amount |
| `FixedAmount` | `decimal(19,4)` | | no | Fixed portion |
| `TotalQuantity` | `decimal(23,6)` | | yes | Total quantity |
| `LastUpdatedAt` | `datetime2(3)` | | no | Last update (UTC) |
| `RowVersion` | `rowversion` | | no | Concurrency token |

---

### `co.AllocationCycle`
**Distribution or assessment cycle** · reference: `T811C`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `CycleCode` | `nvarchar(6)` | AK | no | Cycle key |
| `StartDate` | `date` | AK | no | Cycle start date |
| `Name` | `nvarchar(60)` | | no | Description |
| `AllocationType` | `nvarchar(20)` | | no | `Distribution`, `Assessment`, `PeriodicReposting`, `IndirectActivityAllocation` |
| `IsPlan` | `bit` | | no | Plan allocation |
| `PlanVersion` | `nvarchar(3)` | | yes | Plan version |
| `EndDate` | `date` | | no | Cycle end date |
| `IterationAllowed` | `bit` | | no | Iterative processing allowed |
| `CumulativeProcessing` | `bit` | | no | Cumulative allocation |
| `Status` | `nvarchar(20)` | | no | `Draft`, `Active`, `Blocked` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.AllocationCycleSegment`
**Segment of an allocation cycle: senders, receivers and the tracing factor** · reference: `T811S`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AllocationCycleId` | `bigint` | AK,FK | no | Owning cycle |
| `SegmentCode` | `nvarchar(6)` | AK | no | Segment key |
| `Name` | `nvarchar(60)` | | no | Description |
| `SegmentOrder` | `int` | | no | Processing order |
| `SenderObjectType` | `nvarchar(20)` | | no | Sender object type |
| `SenderSelection` | `nvarchar(max)` | | no | Sender selection (JSON: groups, intervals) |
| `SenderCostElementSelection` | `nvarchar(max)` | | yes | Cost elements allocated |
| `SenderRule` | `nvarchar(20)` | | no | `PostedAmounts`, `FixedAmounts`, `FixedRates` |
| `SenderPercent` | `decimal(9,4)` | | yes | Percentage of the sender to allocate |
| `ReceiverObjectType` | `nvarchar(20)` | | no | Receiver object type |
| `ReceiverSelection` | `nvarchar(max)` | | no | Receiver selection (JSON) |
| `ReceiverRule` | `nvarchar(20)` | | no | `VariablePortions`, `FixedPercentages`, `FixedPortions`, `FixedAmounts` |
| `TracingFactorType` | `nvarchar(20)` | | yes | `StatisticalKeyFigure`, `PostedCosts`, `ActivityQuantity`, `Percentage` |
| `StatisticalKeyFigureId` | `bigint` | FK | yes | Key figure used as the tracing factor |
| `AssessmentCostElementId` | `bigint` | FK | yes | Assessment cost element |
| `ScalingNegativeTracingFactors` | `nvarchar(20)` | | yes | Handling of negative tracing factors |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.AllocationCycleReceiver`
**Fixed receiver share of an allocation segment**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AllocationCycleSegmentId` | `bigint` | AK,FK | no | Owning segment |
| `ReceiverObjectType` | `nvarchar(20)` | AK | no | Receiver type |
| `ReceiverObjectId` | `bigint` | AK | no | Receiver object |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period |
| `Percentage` | `decimal(9,4)` | | yes | Fixed percentage |
| `Portion` | `decimal(23,6)` | | yes | Fixed portion |
| `FixedAmount` | `decimal(19,4)` | | yes | Fixed amount |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.AllocationRun`
**Execution of an allocation cycle**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AllocationCycleId` | `bigint` | AK,FK | no | Cycle executed |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period |
| `RunSequence` | `int` | AK | no | Sequence when repeated |
| `IsTestRun` | `bit` | | no | Test run |
| `PostingDate` | `date` | | no | Posting date |
| `Status` | `nvarchar(20)` | | no | `Running`, `Completed`, `Failed`, `Reversed` |
| `SenderCount` | `int` | | yes | Number of senders processed |
| `ReceiverCount` | `int` | | yes | Number of receivers credited |
| `TotalAllocatedAmount` | `decimal(19,4)` | | yes | Total allocated |
| `CurrencyCode` | `nvarchar(5)` | FK | yes | Currency |
| `ControllingDocumentNumber` | `nvarchar(30)` | | yes | CO document produced |
| `ExecutedAt` | `datetime2(3)` | | no | Execution timestamp (UTC) |
| `ExecutedBy` | `nvarchar(64)` | | no | Executing user or job |
| `LogText` | `nvarchar(max)` | | yes | Run log |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.PlanEntry`
**Cost centre / order / profit centre planning line**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `PlanVersion` | `nvarchar(3)` | AK | no | Plan version |
| `ObjectType` | `nvarchar(20)` | AK | no | `CostCenter`, `InternalOrder`, `ProfitCenter` |
| `ObjectId` | `bigint` | AK | no | Planned object |
| `CostElementId` | `bigint` | AK,FK | no | Cost element |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period |
| `ActivityTypeId` | `bigint` | AK,FK | yes | Activity type |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `PlannedFixedAmount` | `decimal(19,4)` | | no | Planned fixed costs |
| `PlannedVariableAmount` | `decimal(19,4)` | | no | Planned variable costs |
| `PlannedTotalAmount` | `decimal(19,4)` | | no | Total planned amount |
| `PlannedQuantity` | `decimal(23,6)` | | yes | Planned quantity |
| `UnitOfMeasure` | `nvarchar(3)` | FK | yes | Unit |
| `PlanningMethod` | `nvarchar(20)` | | no | `Manual`, `CopyFromActual`, `CopyFromPlan`, `Distribution`, `Formula` |
| `IsLocked` | `bit` | | no | Plan line locked |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `co.Commitment`
**Open commitment from purchase requisitions and orders** · reference: `COOI`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `ObjectType` | `nvarchar(20)` | AK | no | `CostCenter`, `InternalOrder` |
| `ObjectId` | `bigint` | AK | no | Account assignment object |
| `SourceDocumentType` | `nvarchar(20)` | AK | no | `PurchaseRequisition`, `PurchaseOrder`, `Contract` |
| `SourceDocumentNumber` | `nvarchar(20)` | AK | no | Source document |
| `SourceDocumentItem` | `int` | AK | no | Source item |
| `CostElementId` | `bigint` | FK | no | Cost element |
| `FiscalYear` | `smallint` | | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | | no | Period |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `CommitmentAmount` | `decimal(19,4)` | | no | Committed amount |
| `ReducedAmount` | `decimal(19,4)` | | no | Amount already reduced by actuals |
| `OpenCommitmentAmount` | `decimal(19,4)` | | no | Remaining commitment |
| `Status` | `nvarchar(20)` | | no | `Open`, `PartiallyReduced`, `Closed` |
| *include* | `#AUDIT` | | | Standard audit columns |
