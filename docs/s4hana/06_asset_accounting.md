# 06 — Asset Accounting (`fin`)

Asset accounting posts through the same central engine: every acquisition,
transfer, retirement and depreciation run produces `fin.JournalEntryLine` rows
carrying `AssetId`, so the asset register always reconciles to the G/L.

Values are held per **depreciation area**, which is what allows book, tax and
group depreciation to differ without duplicating the asset master.

---

### `fin.AssetClass`
**Asset class — defaults, number range and account determination** · reference: `ANKA`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AssetClass` | `nvarchar(8)` | AK | no | Asset class key, e.g. `2000` machinery |
| `Name` | `nvarchar(60)` | | no | Description |
| `AccountDeterminationKey` | `nvarchar(8)` | | no | Account determination key |
| `NumberRangeObjectId` | `bigint` | FK | no | Number range object |
| `NumberRangeCode` | `nvarchar(2)` | | no | Number range interval |
| `IsExternalNumbering` | `bit` | | no | Asset number entered by the user |
| `ScreenLayoutKey` | `nvarchar(4)` | | yes | Screen layout for the master record |
| `IsAssetUnderConstruction` | `bit` | | no | Assets under construction class |
| `IsLowValueAsset` | `bit` | | no | Low value asset class |
| `LowValueAmountLimit` | `decimal(19,4)` | | yes | Low value threshold |
| `DefaultUsefulLifeYears` | `int` | | yes | Default useful life in years |
| `DefaultUsefulLifePeriods` | `int` | | yes | Default useful life in periods |
| `AllowSubNumbers` | `bit` | | no | Sub-assets permitted |
| `IsBlocked` | `bit` | | no | Blocked for new assets |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.DepreciationArea`
**Depreciation area (book, tax, group, cost accounting)** · reference: `T093`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `DepreciationArea` | `nvarchar(2)` | AK | no | Area key, e.g. `01` book, `15` tax, `30` group |
| `Name` | `nvarchar(60)` | | no | Description |
| `AreaType` | `nvarchar(20)` | | no | `Book`, `Tax`, `Group`, `CostAccounting`, `Derived` |
| `LedgerId` | `bigint` | FK | yes | Ledger the area posts to |
| `AccountingPrincipleId` | `bigint` | FK | yes | Accounting principle |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Area currency |
| `PostsToGeneralLedger` | `nvarchar(20)` | | no | `RealTime`, `Periodic`, `NoPosting`, `DepreciationOnly` |
| `IsRealDepreciationArea` | `bit` | | no | Real (not derived) area |
| `DerivedFromArea1` | `nvarchar(2)` | | yes | First area of a derived area |
| `DerivedFromArea2` | `nvarchar(2)` | | yes | Second area of a derived area |
| `AcquisitionValueRule` | `nvarchar(20)` | | no | `AllValuesAllowed`, `PositiveOnly`, `NegativeOnly`, `ZeroOnly` |
| `NetBookValueRule` | `nvarchar(20)` | | no | Allowed net book value sign |
| `IsAreaForParallelValuation` | `bit` | | no | Parallel valuation area |
| `IsActive` | `bit` | | no | Active |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.AssetClassDepreciationArea`
**Default depreciation settings per asset class and area** · reference: `ANKB`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AssetClassId` | `bigint` | AK,FK | no | Asset class |
| `DepreciationAreaId` | `bigint` | AK,FK | no | Depreciation area |
| `DepreciationKeyId` | `bigint` | FK | no | Default depreciation key |
| `UsefulLifeYears` | `int` | | yes | Default useful life in years |
| `UsefulLifePeriods` | `int` | | yes | Default useful life in periods |
| `IsAreaDeactivated` | `bit` | | no | Area not used for this class |
| `ScrapValuePercent` | `decimal(9,4)` | | yes | Default scrap value percentage |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.DepreciationKey`
**Depreciation key — method, base and period control** · reference: `T090NA`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DepreciationKey` | `nvarchar(4)` | AK | no | Key, e.g. `LINR`, `DG20` |
| `Name` | `nvarchar(60)` | | no | Description |
| `DepreciationMethod` | `nvarchar(20)` | | no | `StraightLine`, `DecliningBalance`, `SumOfYearsDigits`, `UnitOfProduction`, `Manual`, `Immediate` |
| `BaseValueRule` | `nvarchar(20)` | | no | `AcquisitionValue`, `NetBookValue`, `ReplacementValue`, `HalfAcquisitionValue` |
| `DeclineFactor` | `decimal(9,4)` | | yes | Multiplier for declining balance |
| `PeriodControlAcquisition` | `nvarchar(3)` | | no | Period control for acquisitions, e.g. `01` pro rata |
| `PeriodControlAddition` | `nvarchar(3)` | | no | Period control for subsequent acquisitions |
| `PeriodControlRetirement` | `nvarchar(3)` | | no | Period control for retirements |
| `PeriodControlTransfer` | `nvarchar(3)` | | no | Period control for transfers |
| `ChangeMethodAtEnd` | `nvarchar(20)` | | yes | Change to straight line when it is more favourable |
| `IsScrapValueConsidered` | `bit` | | no | Depreciation stops at the scrap value |
| `AllowNegativeDepreciation` | `bit` | | no | Write-ups permitted |
| `IsShutdownRelevant` | `bit` | | no | Shutdown periods suspend depreciation |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.AssetTransactionType`
**Asset transaction type** · reference: `TABW`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `TransactionType` | `nvarchar(3)` | AK | no | Type key, e.g. `100` acquisition, `200` retirement |
| `Name` | `nvarchar(60)` | | no | Description |
| `TransactionCategory` | `nvarchar(20)` | | no | `Acquisition`, `Retirement`, `Transfer`, `WriteUp`, `PostCapitalization`, `Depreciation`, `Impairment` |
| `DebitCreditIndicator` | `nvarchar(1)` | | no | `S` debit, `H` credit |
| `AffectsAcquisitionValue` | `bit` | | no | Updates the acquisition value |
| `AffectsAccumulatedDepreciation` | `bit` | | no | Updates accumulated depreciation |
| `IsPriorYearAcquisition` | `bit` | | no | Refers to a prior-year acquisition |
| `RequiresRevenueAccount` | `bit` | | no | Requires a revenue account (sale) |
| `IsRetirementWithRevenue` | `bit` | | no | Retirement with revenue |
| `DepreciationAreaRestriction` | `nvarchar(255)` | | yes | Areas the type may post to |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.Asset`
**Asset master record** · reference: `ANLA`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `AssetNumber` | `nvarchar(12)` | AK | no | Main asset number |
| `AssetSubNumber` | `int` | AK | no | Sub-number (`0` = main asset) |
| `AssetClassId` | `bigint` | FK | no | Asset class |
| `Description` | `nvarchar(60)` | IX | no | Asset description |
| `Description2` | `nvarchar(60)` | | yes | Additional description |
| `SerialNumber` | `nvarchar(40)` | | yes | Serial number |
| `InventoryNumber` | `nvarchar(25)` | | yes | Inventory number |
| `Quantity` | `decimal(23,6)` | | yes | Quantity |
| `UnitOfMeasure` | `nvarchar(3)` | FK | yes | Unit of measure |
| `CapitalizationDate` | `date` | | yes | Capitalisation date — starts depreciation |
| `AcquisitionDate` | `date` | | yes | First acquisition date |
| `InServiceDate` | `date` | | yes | Date placed in service |
| `DeactivationDate` | `date` | | yes | Deactivation / retirement date |
| `PlannedRetirementDate` | `date` | | yes | Planned retirement |
| `CostCenterId` | `bigint` | FK | yes | Responsible cost centre |
| `ProfitCenterId` | `bigint` | FK | yes | Profit centre |
| `SegmentId` | `bigint` | FK | yes | Segment |
| `FunctionalAreaId` | `bigint` | FK | yes | Functional area |
| `BusinessAreaId` | `bigint` | FK | yes | Business area |
| `InternalOrderId` | `bigint` | FK | yes | Investment order |
| `PlantId` | `bigint` | FK | yes | Plant |
| `LocationId` | `bigint` | FK | yes | Location |
| `RoomNumber` | `nvarchar(20)` | | yes | Room |
| `ResponsiblePersonPartnerId` | `bigint` | FK | yes | Person responsible (employee BP) |
| `VendorBusinessPartnerId` | `bigint` | FK | yes | Supplying vendor |
| `ManufacturerName` | `nvarchar(60)` | | yes | Manufacturer |
| `LicensePlateNumber` | `nvarchar(20)` | | yes | Licence plate (vehicles) |
| `IsAssetUnderConstruction` | `bit` | | no | Asset under construction |
| `SettlementProfile` | `nvarchar(6)` | | yes | Settlement profile for AuC |
| `IsLowValueAsset` | `bit` | | no | Low value asset |
| `IsInvestmentSupport` | `bit` | | no | Investment support asset |
| `Status` | `nvarchar(20)` | IX | no | `Created`, `Capitalized`, `Active`, `Retired`, `Sold`, `Scrapped`, `Blocked`, `MarkedForDeletion` |
| `IsPostingBlocked` | `bit` | | no | Blocked for postings |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| `LegacyAssetNumber` | `nvarchar(20)` | | yes | Number in the legacy system |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.AssetTimeDependent`
**Time-dependent asset assignments** · reference: `ANLZ`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AssetId` | `bigint` | AK,FK | no | Asset |
| `ValidFrom` | `date` | AK | no | Valid from |
| `ValidTo` | `date` | | no | Valid to |
| `CostCenterId` | `bigint` | FK | yes | Cost centre in this interval |
| `ProfitCenterId` | `bigint` | FK | yes | Profit centre |
| `SegmentId` | `bigint` | FK | yes | Segment |
| `InternalOrderId` | `bigint` | FK | yes | Internal order |
| `PlantId` | `bigint` | FK | yes | Plant |
| `LocationId` | `bigint` | FK | yes | Location |
| `IsShutdown` | `bit` | | no | Asset shut down — depreciation suspended |
| `ShiftFactor` | `decimal(9,4)` | | yes | Multiple-shift factor |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.AssetDepreciationArea`
**Depreciation parameters of an asset per area** · reference: `ANLB`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AssetId` | `bigint` | AK,FK | no | Asset |
| `DepreciationAreaId` | `bigint` | AK,FK | no | Depreciation area |
| `DepreciationKeyId` | `bigint` | FK | no | Depreciation key |
| `UsefulLifeYears` | `int` | | no | Useful life in years |
| `UsefulLifePeriods` | `int` | | no | Additional periods |
| `ExpiredUsefulLifeYears` | `int` | | no | Expired years |
| `ExpiredUsefulLifePeriods` | `int` | | no | Expired periods |
| `DepreciationStartDate` | `date` | | yes | Ordinary depreciation start |
| `SpecialDepreciationStartDate` | `date` | | yes | Special depreciation start |
| `ScrapValue` | `decimal(19,4)` | | yes | Scrap value |
| `ScrapValuePercent` | `decimal(9,4)` | | yes | Scrap value percentage |
| `IsDeactivated` | `bit` | | no | Area deactivated for this asset |
| `IsManualDepreciation` | `bit` | | no | Depreciation entered manually |
| `IndexSeries` | `nvarchar(4)` | | yes | Index series for replacement values |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.AssetTransaction`
**Asset transaction (acquisition, retirement, transfer, write-up)** · reference: `ANEP`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `AssetId` | `bigint` | AK,FK | no | Asset |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `AssetDocumentNumber` | `nvarchar(20)` | AK | no | Asset document number |
| `LineItemNumber` | `int` | AK | no | Line number |
| `DepreciationAreaId` | `bigint` | AK,FK | no | Depreciation area |
| `TransactionTypeId` | `bigint` | FK | no | Transaction type |
| `PostingDate` | `date` | IX | no | Posting date |
| `DocumentDate` | `date` | | no | Document date |
| `AssetValueDate` | `date` | IX | no | Asset value date — drives period control |
| `FiscalPeriod` | `tinyint` | | no | Period |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Transaction currency |
| `TransactionAmount` | `decimal(19,4)` | | no | Amount in transaction currency |
| `AmountInLocalCurrency` | `decimal(19,4)` | | no | Amount in local currency |
| `AmountInAreaCurrency` | `decimal(19,4)` | | no | Amount in the area currency |
| `Quantity` | `decimal(23,6)` | | yes | Quantity retired or acquired |
| `AccumulatedDepreciationRetired` | `decimal(19,4)` | | yes | Accumulated depreciation removed on retirement |
| `RevenueAmount` | `decimal(19,4)` | | yes | Sale revenue |
| `GainLossAmount` | `decimal(19,4)` | | yes | Gain or loss on retirement |
| `PercentRetired` | `decimal(9,4)` | | yes | Percentage retired |
| `PartnerBusinessPartnerId` | `bigint` | FK | yes | Vendor or customer involved |
| `TargetAssetId` | `bigint` | FK | yes | Receiving asset on transfer |
| `JournalEntryHeaderId` | `bigint` | FK | no | Accounting document created |
| `ReferenceDocumentNumber` | `nvarchar(20)` | | yes | Reference document |
| `Text` | `nvarchar(255)` | | yes | Transaction text |
| `IsReversed` | `bit` | | no | Reversed |
| `ReversalAssetDocumentNumber` | `nvarchar(20)` | | yes | Reversing asset document |
| `CreatedAt` | `datetime2(3)` | | no | Creation timestamp (UTC) |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user |

---

### `fin.AssetValue`
**Cumulative asset values per year and area** · reference: `ANLC`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AssetId` | `bigint` | AK,FK | no | Asset |
| `DepreciationAreaId` | `bigint` | AK,FK | no | Depreciation area |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency of the values |
| `AcquisitionValueBroughtForward` | `decimal(19,4)` | | no | Opening acquisition value |
| `AccumulatedDepreciationBroughtForward` | `decimal(19,4)` | | no | Opening accumulated depreciation |
| `CurrentYearAcquisitions` | `decimal(19,4)` | | no | Acquisitions in the year |
| `CurrentYearRetirements` | `decimal(19,4)` | | no | Retirements in the year |
| `CurrentYearTransfers` | `decimal(19,4)` | | no | Transfers in the year |
| `OrdinaryDepreciationPosted` | `decimal(19,4)` | | no | Ordinary depreciation posted |
| `SpecialDepreciationPosted` | `decimal(19,4)` | | no | Special depreciation posted |
| `UnplannedDepreciationPosted` | `decimal(19,4)` | | no | Unplanned depreciation posted |
| `ImpairmentPosted` | `decimal(19,4)` | | no | Impairment posted |
| `WriteUpPosted` | `decimal(19,4)` | | no | Write-ups posted |
| `PlannedDepreciationYear` | `decimal(19,4)` | | no | Planned depreciation for the year |
| `NetBookValue` | `decimal(19,4)` | | no | Net book value at the end of the year |
| `ScrapValue` | `decimal(19,4)` | | yes | Scrap value |
| `ReplacementValue` | `decimal(19,4)` | | yes | Replacement value |
| `LastUpdatedAt` | `datetime2(3)` | | no | Last update (UTC) |
| `RowVersion` | `rowversion` | | no | Concurrency token |

---

### `fin.DepreciationRun`
**Depreciation posting run (AFAB)**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period posted |
| `RunType` | `nvarchar(20)` | | no | `Planned`, `Repeat`, `Restart`, `Unplanned` |
| `IsTestRun` | `bit` | | no | Test run — no postings created |
| `Status` | `nvarchar(20)` | | no | `Scheduled`, `Running`, `Completed`, `Failed`, `Cancelled` |
| `AssetsProcessed` | `int` | | yes | Number of assets processed |
| `TotalDepreciationAmount` | `decimal(19,4)` | | yes | Total depreciation posted |
| `ErrorCount` | `int` | | yes | Number of errors |
| `StartedAt` | `datetime2(3)` | | no | Start timestamp (UTC) |
| `CompletedAt` | `datetime2(3)` | | yes | Completion timestamp (UTC) |
| `ExecutedBy` | `nvarchar(64)` | | no | Executing user or job |
| `LogText` | `nvarchar(max)` | | yes | Run log |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.DepreciationPosting`
**Depreciation posted per asset, area and period** · reference: `ANLP`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DepreciationRunId` | `bigint` | FK,IX | no | Run that produced the posting |
| `AssetId` | `bigint` | AK,FK | no | Asset |
| `DepreciationAreaId` | `bigint` | AK,FK | no | Depreciation area |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period |
| `PostingDate` | `date` | | no | Posting date |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `OrdinaryDepreciationAmount` | `decimal(19,4)` | | no | Ordinary depreciation |
| `SpecialDepreciationAmount` | `decimal(19,4)` | | no | Special depreciation |
| `UnplannedDepreciationAmount` | `decimal(19,4)` | | no | Unplanned depreciation |
| `ImpairmentAmount` | `decimal(19,4)` | | no | Impairment |
| `WriteUpAmount` | `decimal(19,4)` | | no | Write-up |
| `TotalPostedAmount` | `decimal(19,4)` | | no | Total posted in the period |
| `NetBookValueAfterPosting` | `decimal(19,4)` | | no | Net book value after the posting |
| `CostCenterId` | `bigint` | FK | yes | Cost centre charged |
| `ProfitCenterId` | `bigint` | FK | yes | Profit centre charged |
| `InternalOrderId` | `bigint` | FK | yes | Internal order charged |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Accounting document created |
| `IsReversed` | `bit` | | no | Reversed by a repeat run |
| `CreatedAt` | `datetime2(3)` | | no | Creation timestamp (UTC) |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user or job |
