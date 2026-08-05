# 02 — Configuration (`cfg`)

Everything the posting engine reads before it will accept a document: chart of
accounts, fiscal calendar, period control, document types, number ranges,
currencies, tax, payment terms, ledgers and account determination.

---

## 2.1 Chart of accounts and ledgers

### `cfg.ChartOfAccounts`
**Chart of accounts** · reference: `T004`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ChartOfAccounts` | `nvarchar(4)` | AK | no | Chart of accounts key |
| `Name` | `nvarchar(60)` | | no | Description |
| `MaintenanceLanguage` | `nvarchar(2)` | FK | no | Language of account names |
| `AccountNumberLength` | `tinyint` | | no | Length of G/L account numbers (max 10) |
| `GroupChartOfAccountsId` | `bigint` | FK | yes | Group chart for consolidation |
| `IsBlockedForPosting` | `bit` | | no | Chart blocked |
| `ChartType` | `nvarchar(20)` | | no | `Operational`, `Group`, `Country` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.AccountGroup`
**G/L account group — controls number interval and field status** · reference: `T077S`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ChartOfAccountsId` | `bigint` | AK,FK | no | Owning chart of accounts |
| `AccountGroup` | `nvarchar(4)` | AK | no | Account group key |
| `Name` | `nvarchar(40)` | | no | Description |
| `FromAccount` | `nvarchar(10)` | | no | Lower limit of the number interval |
| `ToAccount` | `nvarchar(10)` | | no | Upper limit of the number interval |
| `FieldStatusGroupId` | `bigint` | FK | yes | Field status for master-data maintenance |
| `AppliesTo` | `nvarchar(10)` | | no | `GL`, `Customer`, `Vendor` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.AccountingPrinciple`
**Accounting principle (IFRS, local GAAP)** · reference: `T001`/`FAGL` accounting principle

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AccountingPrinciple` | `nvarchar(4)` | AK | no | Key, e.g. `IFRS`, `LGAP` |
| `Name` | `nvarchar(60)` | | no | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.Ledger`
**Ledger of the universal journal (leading + parallel)** · reference: `T881`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `Ledger` | `nvarchar(2)` | AK | no | Ledger key, e.g. `0L`, `2L` |
| `Name` | `nvarchar(40)` | | no | Description |
| `IsLeading` | `bit` | | no | Leading ledger — exactly one per tenant |
| `AccountingPrincipleId` | `bigint` | FK | no | Accounting principle represented |
| `LedgerType` | `nvarchar(10)` | | no | `Standard`, `Extension`, `Appendix` |
| `IsExtensionLedger` | `bit` | | no | Postings are deltas on the underlying ledger |
| `UnderlyingLedgerId` | `bigint` | FK | yes | Base ledger for an extension ledger |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.LedgerCompanyCode`
**Company code settings per ledger** · reference: `T882G`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `LedgerId` | `bigint` | AK,FK | no | Ledger |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYearVariantId` | `bigint` | FK | no | Fiscal year variant for this ledger |
| `PostingPeriodVariantId` | `bigint` | FK | no | Posting period variant for this ledger |
| `Currency1TypeCode` | `nvarchar(2)` | | no | Currency type of amount 1 (`10` local) |
| `Currency2TypeCode` | `nvarchar(2)` | | yes | Currency type of amount 2 (`30` group) |
| `Currency3TypeCode` | `nvarchar(2)` | | yes | Currency type of amount 3 (`40` hard / `50` index) |
| `IsActive` | `bit` | | no | Ledger active for this company code |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 2.2 Fiscal calendar and period control

### `cfg.FiscalYearVariant`
**Fiscal year variant** · reference: `T009`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FiscalYearVariant` | `nvarchar(2)` | AK | no | Variant key, e.g. `K4` |
| `Name` | `nvarchar(40)` | | no | Description |
| `NumberOfPostingPeriods` | `tinyint` | | no | Normal periods (1–12) |
| `NumberOfSpecialPeriods` | `tinyint` | | no | Special periods (0–4) |
| `IsCalendarYear` | `bit` | | no | Periods equal calendar months |
| `IsYearDependent` | `bit` | | no | Period boundaries differ per year |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.FiscalYearVariantPeriod`
**Period boundaries of a fiscal year variant** · reference: `T009B`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FiscalYearVariantId` | `bigint` | AK,FK | no | Owning variant |
| `CalendarYear` | `smallint` | AK | no | Year (`0000` when year-independent) |
| `CalendarMonth` | `tinyint` | AK | no | Month of the period end |
| `CalendarDay` | `tinyint` | AK | no | Day of the period end |
| `FiscalPeriod` | `tinyint` | | no | Resulting posting period |
| `YearShift` | `smallint` | | no | `-1`, `0`, `+1` fiscal year offset |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.FiscalPeriod`
**Materialised fiscal period calendar (posting date → year/period)** · derived from `T009B`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FiscalYearVariantId` | `bigint` | AK,FK | no | Fiscal year variant |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Posting period `1..16` |
| `PeriodStartDate` | `date` | IX | no | First calendar day of the period |
| `PeriodEndDate` | `date` | IX | no | Last calendar day of the period |
| `IsSpecialPeriod` | `bit` | | no | Special (year-end adjustment) period |
| `PeriodStatus` | `nvarchar(20)` | | no | `Open`, `ClosedForEntry`, `Closed` |
| `ClosedAt` | `datetime2(3)` | | yes | Period-close timestamp |
| `ClosedBy` | `nvarchar(64)` | | yes | User who closed the period |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.PostingPeriodVariant`
**Posting period variant** · reference: `T010O`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PostingPeriodVariant` | `nvarchar(4)` | AK | no | Variant key |
| `Name` | `nvarchar(40)` | | no | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.PostingPeriodControl`
**Open periods per account type and authorisation group (OB52)** · reference: `T001B`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PostingPeriodVariantId` | `bigint` | AK,FK | no | Posting period variant |
| `AccountType` | `nvarchar(1)` | AK | no | `+` all, `S` G/L, `D` customer, `K` vendor, `A` asset, `M` material |
| `FromAccount` | `nvarchar(10)` | AK | no | Lower account limit |
| `ToAccount` | `nvarchar(10)` | | no | Upper account limit |
| `FromPeriod1` | `tinyint` | | no | First open period, interval 1 |
| `FromYear1` | `smallint` | | no | Fiscal year, interval 1 from |
| `ToPeriod1` | `tinyint` | | no | Last open period, interval 1 |
| `ToYear1` | `smallint` | | no | Fiscal year, interval 1 to |
| `FromPeriod2` | `tinyint` | | yes | First open period, interval 2 (closing) |
| `FromYear2` | `smallint` | | yes | Fiscal year, interval 2 from |
| `ToPeriod2` | `tinyint` | | yes | Last open period, interval 2 |
| `ToYear2` | `smallint` | | yes | Fiscal year, interval 2 to |
| `AuthorizationGroup` | `nvarchar(4)` | | yes | Group allowed to post in interval 2 |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 2.3 Documents, posting keys, field status, numbering

### `cfg.DocumentType`
**Document type — controls number range and allowed account types** · reference: `T003`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DocumentType` | `nvarchar(2)` | AK | no | Document type key, e.g. `SA`, `KR`, `DR` |
| `Name` | `nvarchar(40)` | | no | Description |
| `NumberRangeObjectId` | `bigint` | FK | no | Number range object used |
| `NumberRangeCode` | `nvarchar(2)` | | no | Number range interval key |
| `ReverseDocumentType` | `nvarchar(2)` | | yes | Document type used for reversals |
| `AllowCustomerAccounts` | `bit` | | no | Account type `D` permitted |
| `AllowVendorAccounts` | `bit` | | no | Account type `K` permitted |
| `AllowGLAccounts` | `bit` | | no | Account type `S` permitted |
| `AllowAssetAccounts` | `bit` | | no | Account type `A` permitted |
| `AllowMaterialAccounts` | `bit` | | no | Account type `M` permitted |
| `IsNetDocumentType` | `bit` | | no | Net posting procedure |
| `RequireReferenceNumber` | `bit` | | no | Reference number mandatory |
| `RequireDocumentHeaderText` | `bit` | | no | Header text mandatory |
| `IsExternalNumbering` | `bit` | | no | Number supplied by the user |
| `NegativePostingPermitted` | `bit` | | no | Negative postings allowed |
| `IsIntercompany` | `bit` | | no | Used for cross-company-code postings |
| `SourceModule` | `nvarchar(10)` | | yes | `FI`, `AR`, `AP`, `AA`, `CO`, `MM`, `SD` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.PostingKey`
**Posting key — debit/credit and account type per line** · reference: `TBSL`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PostingKey` | `nvarchar(2)` | AK | no | Posting key, e.g. `40`, `50`, `01`, `31` |
| `Name` | `nvarchar(40)` | | no | Description |
| `DebitCreditIndicator` | `nvarchar(1)` | | no | `S` debit, `H` credit |
| `AccountType` | `nvarchar(1)` | | no | `S`, `D`, `K`, `A`, `M` |
| `IsSalesRelated` | `bit` | | no | Sales-related (updates customer statistics) |
| `IsSpecialGLPosting` | `bit` | | no | Requires a special G/L indicator |
| `IsReversalPostingKey` | `bit` | | no | Used only by reversals |
| `PaymentTransaction` | `bit` | | no | Payment-relevant |
| `FieldStatusGroupId` | `bigint` | FK | yes | Field status of the line item |
| `IsBlocked` | `bit` | | no | Blocked for entry |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.FieldStatusVariant`
**Field status variant** · reference: `T004V`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FieldStatusVariant` | `nvarchar(4)` | AK | no | Variant key |
| `Name` | `nvarchar(40)` | | no | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.FieldStatusGroup`
**Field status group** · reference: `T004G`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FieldStatusVariantId` | `bigint` | AK,FK | no | Owning variant |
| `FieldStatusGroup` | `nvarchar(4)` | AK | no | Group key, e.g. `G001` |
| `Name` | `nvarchar(60)` | | no | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.FieldStatusFieldControl`
**Per-field status inside a field status group** · reference: `T004F`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FieldStatusGroupId` | `bigint` | AK,FK | no | Owning group |
| `FieldName` | `nvarchar(64)` | AK | no | Journal line field controlled |
| `FieldGroup` | `nvarchar(40)` | | no | Screen group, e.g. `Additional account assignments` |
| `FieldStatus` | `nvarchar(10)` | | no | `Suppress`, `Optional`, `Required`, `Display` |
| `DisplayOrder` | `int` | | no | Order on the entry screen |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.NumberRangeObject`
**Number range object** · reference: `TNRO`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `NumberRangeObject` | `nvarchar(20)` | AK | no | Object key, e.g. `RF_BELEG`, `BP`, `ASSET` |
| `Name` | `nvarchar(60)` | | no | Description |
| `IsYearDependent` | `bit` | | no | Intervals are per fiscal year |
| `IsCompanyCodeDependent` | `bit` | | no | Intervals are per company code |
| `NumberLength` | `tinyint` | | no | Number of digits, zero padded |
| `Prefix` | `nvarchar(10)` | | yes | Static prefix, e.g. `KSS` |
| `NumberFormat` | `nvarchar(60)` | | yes | Format mask, e.g. `{Prefix}-{Year}-{Type}-{Number:0000000000}` |
| `WarnPercent` | `tinyint` | | yes | Warn when this % of the interval is used |
| `GapMonitoring` | `bit` | | no | Detect and log number gaps |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.NumberRangeInterval`
**Number range interval and its current level** · reference: `NRIV`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `NumberRangeObjectId` | `bigint` | AK,FK | no | Number range object |
| `NumberRangeCode` | `nvarchar(2)` | AK | no | Interval key, e.g. `01` |
| `CompanyCodeId` | `bigint` | AK,FK | yes | Company code (null = all) |
| `FiscalYear` | `smallint` | AK | no | Fiscal year (`0` = year-independent) |
| `FromNumber` | `bigint` | | no | Lower limit |
| `ToNumber` | `bigint` | | no | Upper limit |
| `CurrentNumber` | `bigint` | | no | Last number issued — incremented under `UPDLOCK` |
| `IsExternal` | `bit` | | no | External numbering (no automatic assignment) |
| `IsBlocked` | `bit` | | no | Interval blocked |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.NumberRangeGap`
**Recorded gap in an issued number sequence** · audit support for `NRIV`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `NumberRangeIntervalId` | `bigint` | FK,IX | no | Interval concerned |
| `MissingNumber` | `bigint` | | no | Number drawn but never committed |
| `DetectedAt` | `datetime2(3)` | | no | Detection timestamp (UTC) |
| `Reason` | `nvarchar(255)` | | yes | Rollback reason if known |
| `CorrelationId` | `uniqueidentifier` | | yes | Request that lost the number |

---

## 2.4 Currency

### `cfg.Currency`
**Currency master** · reference: `TCURC`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CurrencyCode` | `nvarchar(5)` | AK | no | ISO currency key, e.g. `USD`, `KHR`, `THB` |
| `IsoCode` | `nvarchar(3)` | | no | ISO 4217 code |
| `NumericCode` | `nvarchar(3)` | | yes | ISO numeric code |
| `Name` | `nvarchar(40)` | | no | Currency name |
| `ShortText` | `nvarchar(15)` | | yes | Short name for reports |
| `DecimalPlaces` | `tinyint` | | no | Decimals used in display and rounding |
| `Symbol` | `nvarchar(5)` | | yes | Display symbol |
| `ValidFrom` | `date` | | yes | Introduction date |
| `IsBlocked` | `bit` | | no | Blocked for new transactions |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.CurrencyDecimal`
**Currencies whose decimals differ from 2** · reference: `TCURX`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CurrencyCode` | `nvarchar(5)` | AK,FK | no | Currency |
| `DecimalPlaces` | `tinyint` | | no | Number of decimals (0 for `KHR`, `JPY`) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.ExchangeRateType`
**Exchange rate type** · reference: `TCURV`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ExchangeRateType` | `nvarchar(4)` | AK | no | Rate type, e.g. `M`, `B`, `G`, `EURX` |
| `Name` | `nvarchar(40)` | | no | Description |
| `QuotationType` | `nvarchar(10)` | | no | `Direct`, `Indirect` |
| `ReferenceCurrencyCode` | `nvarchar(5)` | FK | yes | Base currency for cross rates |
| `IsInversionAllowed` | `bit` | | no | Inverted rate may be used when none is found |
| `UseFixedRate` | `bit` | | no | Fixed rate (e.g. currency peg) |
| `IsBuyingRate` | `bit` | | no | Bank buying rate |
| `IsSellingRate` | `bit` | | no | Bank selling rate |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.ExchangeRate`
**Exchange rate valid from a date** · reference: `TCURR`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ExchangeRateTypeId` | `bigint` | AK,FK | no | Rate type |
| `FromCurrencyCode` | `nvarchar(5)` | AK,FK | no | Source currency |
| `ToCurrencyCode` | `nvarchar(5)` | AK,FK | no | Target currency |
| `ValidFrom` | `date` | AK | no | Valid-from date — latest ≤ posting date wins |
| `Rate` | `decimal(23,6)` | | no | Exchange rate |
| `FromRatio` | `int` | | no | Ratio for the source currency |
| `ToRatio` | `int` | | no | Ratio for the target currency |
| `Source` | `nvarchar(20)` | | yes | `Manual`, `Import`, `CentralBank` |
| `ImportedAt` | `datetime2(3)` | | yes | Import timestamp (UTC) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.CurrencyTranslationRatio`
**Translation ratios per currency pair and rate type** · reference: `TCURF`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ExchangeRateTypeId` | `bigint` | AK,FK | no | Rate type |
| `FromCurrencyCode` | `nvarchar(5)` | AK,FK | no | Source currency |
| `ToCurrencyCode` | `nvarchar(5)` | AK,FK | no | Target currency |
| `ValidFrom` | `date` | AK | no | Valid-from date |
| `FromRatio` | `int` | | no | Source ratio |
| `ToRatio` | `int` | | no | Target ratio |
| `AlternativeExchangeRateType` | `nvarchar(4)` | | yes | Rate type used instead |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 2.5 Country, region, language, units

### `cfg.Country`
**Country** · reference: `T005`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CountryCode` | `nvarchar(3)` | AK | no | Country key, e.g. `KH`, `TH`, `US` |
| `IsoCode2` | `nvarchar(2)` | | no | ISO 3166-1 alpha-2 |
| `IsoCode3` | `nvarchar(3)` | | yes | ISO 3166-1 alpha-3 |
| `Name` | `nvarchar(60)` | | no | Country name |
| `CurrencyCode` | `nvarchar(5)` | FK | yes | Default currency |
| `LanguageCode` | `nvarchar(2)` | FK | yes | Default language |
| `AddressFormatKey` | `nvarchar(4)` | | yes | Address layout key |
| `TaxNumberRule` | `nvarchar(60)` | | yes | Validation regex for the tax number |
| `IsEuMember` | `bit` | | no | EU member (intra-community tax handling) |
| `DateFormat` | `nvarchar(10)` | | yes | Preferred date format |
| `DecimalSeparator` | `nvarchar(1)` | | yes | Decimal separator for output |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.Region`
**Region / province / state** · reference: `T005S`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CountryCode` | `nvarchar(3)` | AK,FK | no | Country |
| `RegionCode` | `nvarchar(3)` | AK | no | Region key |
| `Name` | `nvarchar(60)` | | no | Region name |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.Language`
**Language** · reference: `T002`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| `LanguageCode` | `nvarchar(2)` | AK | no | Language key, e.g. `EN`, `KM` |
| `IsoCode` | `nvarchar(5)` | | no | Culture code, e.g. `en-US`, `km-KH` |
| `Name` | `nvarchar(40)` | | no | Language name |
| `IsRightToLeft` | `bit` | | no | Right-to-left script |
| `IsActive` | `bit` | | no | Available in the UI |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.UnitOfMeasure`
**Unit of measure** · reference: `T006`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UnitOfMeasure` | `nvarchar(3)` | AK | no | Internal unit key |
| `IsoCode` | `nvarchar(3)` | | yes | ISO unit code |
| `Name` | `nvarchar(40)` | | no | Unit name |
| `Dimension` | `nvarchar(10)` | | yes | `MASS`, `VOLUME`, `TIME`, `AREA` |
| `DecimalPlaces` | `tinyint` | | no | Decimals used for quantities |
| `ConversionFactor` | `decimal(23,6)` | | yes | Factor to the base unit of the dimension |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 2.6 Tax

### `cfg.TaxJurisdiction`
**Tax jurisdiction** · reference: `TTXJ`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `TaxJurisdictionCode` | `nvarchar(15)` | AK | no | Jurisdiction code |
| `CountryCode` | `nvarchar(3)` | FK | no | Country |
| `SchemaCode` | `nvarchar(4)` | | no | Jurisdiction schema |
| `Name` | `nvarchar(60)` | | no | Description |
| `ParentJurisdictionId` | `bigint` | FK | yes | Higher-level jurisdiction |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.TaxCode`
**Tax code** · reference: `T007A`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CountryCode` | `nvarchar(3)` | AK,FK | no | Tax country |
| `TaxCode` | `nvarchar(2)` | AK | no | Tax code key, e.g. `V1`, `A1` |
| `Name` | `nvarchar(60)` | | no | Description |
| `TaxType` | `nvarchar(1)` | | no | `V` input tax, `A` output tax |
| `TaxCategory` | `nvarchar(20)` | | no | `VAT`, `WithholdingTax`, `SalesTax`, `Exempt` |
| `IsReverseCharge` | `bit` | | no | Reverse charge mechanism |
| `IsNonDeductible` | `bit` | | no | Non-deductible input tax |
| `TargetTaxCode` | `nvarchar(2)` | | yes | Target code for deferred tax |
| `CheckIndicator` | `nvarchar(1)` | | yes | Error / warning on tax deviation |
| `IsBlocked` | `bit` | | no | Blocked for posting |
| *include* | `#VALIDITY` | | | Validity of the code |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.TaxCodeRate`
**Rate per tax code, condition and validity** · reference: `A003` / `KONP`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `TaxCodeId` | `bigint` | AK,FK | no | Tax code |
| `ConditionType` | `nvarchar(4)` | AK | no | Condition, e.g. `MWVS`, `MWAS`, `NAVS` |
| `TaxJurisdictionCode` | `nvarchar(15)` | AK | yes | Jurisdiction (when jurisdiction-based) |
| `ValidFrom` | `date` | AK | no | Valid from |
| `ValidTo` | `date` | | no | Valid to |
| `RatePercent` | `decimal(9,4)` | | no | Tax rate in percent |
| `TaxAccountKey` | `nvarchar(3)` | | no | Account key for determination, e.g. `VST`, `MWS` |
| `IsDeductiblePortion` | `bit` | | no | Deductible portion of the rate |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.WithholdingTaxType`
**Withholding tax type** · reference: `T059P`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CountryCode` | `nvarchar(3)` | AK,FK | no | Country |
| `WithholdingTaxType` | `nvarchar(2)` | AK | no | Type key |
| `Name` | `nvarchar(60)` | | no | Description |
| `PostingTime` | `nvarchar(20)` | | no | `Invoice`, `Payment` |
| `BaseAmountType` | `nvarchar(20)` | | no | `GrossAmount`, `NetAmount`, `TaxAmount` |
| `RoundingRule` | `nvarchar(20)` | | no | `Commercial`, `Up`, `Down` |
| `IsAccumulationActive` | `bit` | | no | Accumulate base amounts per year |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.WithholdingTaxCode`
**Withholding tax code and rate** · reference: `T059Z`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `WithholdingTaxTypeId` | `bigint` | AK,FK | no | Withholding tax type |
| `WithholdingTaxCode` | `nvarchar(2)` | AK | no | Code key |
| `Name` | `nvarchar(60)` | | no | Description |
| `RatePercent` | `decimal(9,4)` | | no | Withholding rate |
| `BasePercent` | `decimal(9,4)` | | no | Percentage of the base subject to tax |
| `MinimumBaseAmount` | `decimal(19,4)` | | yes | Exemption threshold |
| `GLAccountId` | `bigint` | FK | yes | Withholding tax G/L account |
| *include* | `#VALIDITY` | | | Validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 2.7 Payment, dunning, tolerance

### `cfg.PaymentTerms`
**Payment terms** · reference: `T052`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PaymentTerms` | `nvarchar(4)` | AK | no | Terms key, e.g. `0001`, `NT30` |
| `Name` | `nvarchar(60)` | | no | Description |
| `BaselineDateRule` | `nvarchar(20)` | | no | `DocumentDate`, `PostingDate`, `EntryDate`, `NoDefault` |
| `AdditionalDays` | `int` | | no | Days added to the baseline date |
| `FixedDay` | `tinyint` | | yes | Fixed calendar day for the due date |
| `AdditionalMonths` | `tinyint` | | yes | Months added |
| `NetDueDays` | `int` | | no | Days until the net amount is due |
| `CashDiscount1Days` | `int` | | yes | Days for discount level 1 |
| `CashDiscount1Percent` | `decimal(9,4)` | | yes | Discount percentage level 1 |
| `CashDiscount2Days` | `int` | | yes | Days for discount level 2 |
| `CashDiscount2Percent` | `decimal(9,4)` | | yes | Discount percentage level 2 |
| `IsForCustomer` | `bit` | | no | Valid for customers |
| `IsForVendor` | `bit` | | no | Valid for vendors |
| `IsInstallmentPlan` | `bit` | | no | Split into instalments |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.PaymentTermsInstallment`
**Instalment plan lines** · reference: `T052S`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PaymentTermsId` | `bigint` | AK,FK | no | Instalment payment terms |
| `InstallmentNumber` | `tinyint` | AK | no | Sequence of the instalment |
| `Percentage` | `decimal(9,4)` | | no | Share of the invoice amount |
| `InstallmentPaymentTermsId` | `bigint` | FK | no | Payment terms applied to the instalment |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.PaymentMethod`
**Payment method** · reference: `T042Z`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CountryCode` | `nvarchar(3)` | AK,FK | no | Country |
| `PaymentMethod` | `nvarchar(1)` | AK | no | Method key, e.g. `T`, `C`, `B` |
| `Name` | `nvarchar(60)` | | no | Description |
| `PaymentType` | `nvarchar(20)` | | no | `BankTransfer`, `Check`, `Cash`, `Card`, `Draft` |
| `IsForOutgoing` | `bit` | | no | Outgoing payments |
| `IsForIncoming` | `bit` | | no | Incoming payments |
| `RequireBankDetails` | `bit` | | no | Bank details mandatory |
| `AllowForeignCurrency` | `bit` | | no | Foreign currency permitted |
| `AllowForeignBank` | `bit` | | no | Foreign bank permitted |
| `PaymentFileFormat` | `nvarchar(20)` | | yes | `ISO20022`, `CSV`, `ACH`, `Local` |
| `DocumentTypeId` | `bigint` | FK | yes | Document type used by the payment run |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DunningProcedure`
**Dunning procedure** · reference: `T047A`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DunningProcedure` | `nvarchar(4)` | AK | no | Procedure key |
| `Name` | `nvarchar(60)` | | no | Description |
| `DunningIntervalDays` | `int` | | no | Minimum days between dunning runs |
| `NumberOfDunningLevels` | `tinyint` | | no | Number of levels |
| `GracePeriodDays` | `int` | | no | Line item grace days |
| `MinimumDaysInArrears` | `int` | | no | Minimum arrears before dunning |
| `InterestIndicator` | `nvarchar(2)` | | yes | Interest calculation indicator |
| `IsStandardTransactionDunning` | `bit` | | no | Dun standard transactions |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.DunningLevel`
**Dunning level settings** · reference: `T047`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DunningProcedureId` | `bigint` | AK,FK | no | Dunning procedure |
| `DunningLevel` | `tinyint` | AK | no | Level `1..9` |
| `DaysInArrears` | `int` | | no | Days in arrears to reach this level |
| `CalculateInterest` | `bit` | | no | Interest charged at this level |
| `PrintAllItems` | `bit` | | no | Print all open items |
| `AlwaysDun` | `bit` | | no | Dun even without new items |
| `DunningCharge` | `decimal(19,4)` | | yes | Fixed dunning charge |
| `MinimumAmount` | `decimal(19,4)` | | yes | Minimum amount to dun |
| `FormId` | `bigint` | FK | yes | Correspondence form |
| `IsLegalDunning` | `bit` | | no | Legal dunning procedure |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.ToleranceGroup`
**Posting and payment tolerances** · reference: `T043T` / `T043G`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `ToleranceGroup` | `nvarchar(4)` | AK | no | Group key (blank = default) |
| `ToleranceType` | `nvarchar(20)` | AK | no | `User`, `Customer`, `Vendor`, `GL` |
| `Name` | `nvarchar(60)` | | yes | Description |
| `MaxDocumentAmount` | `decimal(19,4)` | | yes | Maximum amount per document |
| `MaxLineItemAmount` | `decimal(19,4)` | | yes | Maximum amount per open item |
| `MaxCashDiscountPercent` | `decimal(9,4)` | | yes | Maximum cash discount |
| `PaymentDifferenceGainAmount` | `decimal(19,4)` | | yes | Permitted revenue from differences |
| `PaymentDifferenceGainPercent` | `decimal(9,4)` | | yes | Permitted revenue in percent |
| `PaymentDifferenceLossAmount` | `decimal(19,4)` | | yes | Permitted expense from differences |
| `PaymentDifferenceLossPercent` | `decimal(9,4)` | | yes | Permitted expense in percent |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 2.8 Account determination and financial statements

### `cfg.AccountDeterminationRule`
**Automatic account determination (tax, gain/loss, clearing, retained earnings)** · reference: `T030`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ChartOfAccountsId` | `bigint` | AK,FK | no | Chart of accounts |
| `TransactionKey` | `nvarchar(4)` | AK | no | Key, e.g. `MWS`, `VST`, `KDF`, `BSX`, `BIL` |
| `AccountModifier` | `nvarchar(4)` | AK | yes | Additional differentiation |
| `CompanyCodeId` | `bigint` | AK,FK | yes | Company code (null = all) |
| `CurrencyCode` | `nvarchar(5)` | AK,FK | yes | Currency-specific rule |
| `DebitGLAccountId` | `bigint` | FK | yes | Debit account |
| `CreditGLAccountId` | `bigint` | FK | yes | Credit account |
| `Description` | `nvarchar(60)` | | yes | Purpose of the rule |
| *include* | `#VALIDITY` | | | Validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.FinancialStatementVersion`
**Financial statement version** · reference: `T011`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FinancialStatementVersion` | `nvarchar(4)` | AK | no | Version key |
| `Name` | `nvarchar(60)` | | no | Description |
| `ChartOfAccountsId` | `bigint` | FK | yes | Chart of accounts (null = account-group based) |
| `MaintenanceLanguage` | `nvarchar(2)` | FK | no | Language of node texts |
| `IsGroupAccountVersion` | `bit` | | no | Built on group accounts |
| `AccountingPrincipleId` | `bigint` | FK | yes | Accounting principle presented |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.FinancialStatementNode`
**Hierarchy node of a financial statement version** · reference: `T011` structure

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FinancialStatementVersionId` | `bigint` | AK,FK | no | Owning version |
| `NodeCode` | `nvarchar(20)` | AK | no | Node key |
| `ParentNodeId` | `bigint` | FK | yes | Parent node |
| `NodeText` | `nvarchar(60)` | | no | Node description |
| `DisplayOrder` | `int` | | no | Sort order among siblings |
| `NodeType` | `nvarchar(20)` | | no | `Header`, `Total`, `AccountGroup`, `NotAssigned`, `PLResult` |
| `Section` | `nvarchar(20)` | | no | `Assets`, `Liabilities`, `Income`, `Expense` |
| `DebitCreditShift` | `bit` | | no | Move balance to the opposite side when the sign flips |
| `TotalNodeCode` | `nvarchar(20)` | | yes | Node receiving the total |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.FinancialStatementNodeAccount`
**Accounts assigned to a statement node** · reference: `T011` account assignment

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FinancialStatementNodeId` | `bigint` | AK,FK | no | Node |
| `FromGLAccount` | `nvarchar(10)` | AK | no | Lower account limit |
| `ToGLAccount` | `nvarchar(10)` | | no | Upper account limit |
| `BalanceSide` | `nvarchar(10)` | | no | `Debit`, `Credit`, `Both` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.SpecialGLIndicator`
**Special G/L indicator (down payments, guarantees)** · reference: `T074U`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AccountType` | `nvarchar(1)` | AK | no | `D` customer, `K` vendor |
| `SpecialGLIndicator` | `nvarchar(1)` | AK | no | Indicator, e.g. `A` down payment, `F` request |
| `Name` | `nvarchar(60)` | | no | Description |
| `SpecialGLType` | `nvarchar(20)` | | no | `DownPayment`, `BillOfExchange`, `Other`, `Noted` |
| `IsNotedItem` | `bit` | | no | Noted item (no G/L update) |
| `IsFreeOffsettingEntry` | `bit` | | no | Free offsetting entry allowed |
| `TargetSpecialGLIndicator` | `nvarchar(1)` | | yes | Target indicator on transfer |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.SpecialGLAccount`
**Reconciliation account per special G/L indicator** · reference: `T074`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ChartOfAccountsId` | `bigint` | AK,FK | no | Chart of accounts |
| `SpecialGLIndicatorId` | `bigint` | AK,FK | no | Special G/L indicator |
| `ReconciliationGLAccountId` | `bigint` | AK,FK | no | Normal reconciliation account |
| `SpecialGLAccountId` | `bigint` | FK | no | Alternative reconciliation account |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `cfg.CorrespondenceForm`
**Print form for invoices, statements, dunning letters** · reference: `T048`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FormCode` | `nvarchar(20)` | AK | no | Form key |
| `LanguageCode` | `nvarchar(2)` | AK,FK | no | Form language |
| `Name` | `nvarchar(60)` | | no | Description |
| `CorrespondenceType` | `nvarchar(20)` | | no | `Invoice`, `Statement`, `Dunning`, `PaymentAdvice` |
| `TemplateBody` | `nvarchar(max)` | | yes | Template source (HTML/Razor) |
| `OutputFormat` | `nvarchar(10)` | | no | `PDF`, `HTML`, `CSV` |
| *include* | `#AUDIT` | | | Standard audit columns |
