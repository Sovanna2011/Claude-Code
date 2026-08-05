# 01 — Enterprise Structure (`org`)

The organisational skeleton every posting is validated against.

**Hierarchy:** Tenant → Company → Company Code → { Plant, Branch, Business Area }.
Controlling Area groups compatible company codes; Segment and Profit Center
carry the management view.

---

### `org.Tenant`
**Client / tenant — the top isolation boundary** · reference: `T000`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `int` | PK | no | Tenant id, used as `TenantId` everywhere else |
| `TenantCode` | `nvarchar(4)` | AK | no | Short tenant key (SAP client number equivalent) |
| `Name` | `nvarchar(60)` | | no | Tenant name |
| `LogicalSystem` | `nvarchar(20)` | | yes | Logical system name for integration |
| `DefaultLanguage` | `nvarchar(2)` | | no | Default UI language (`EN`, `KM`) |
| `DefaultCurrencyCode` | `nvarchar(5)` | FK | yes | Default currency proposal |
| `TimeZoneId` | `nvarchar(64)` | | no | IANA time zone used for display |
| `IsProduction` | `bit` | | no | Production client — blocks unreviewed DDL and test postings |
| `AllowCustomizingChanges` | `bit` | | no | Whether configuration may be changed directly |
| `ValidFrom` | `date` | | no | Tenant activation date |
| `ValidTo` | `date` | | no | Tenant expiry date |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.Company`
**Legal or consolidation group entity** · reference: `T880`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeGroup` | `nvarchar(6)` | AK | no | Company key (trading partner id) |
| `Name` | `nvarchar(60)` | | no | Company name |
| `Name2` | `nvarchar(60)` | | yes | Name line 2 |
| `CountryCode` | `nvarchar(3)` | FK | no | Country of registration |
| `GroupCurrencyCode` | `nvarchar(5)` | FK | no | Consolidation (group) currency |
| `LanguageCode` | `nvarchar(2)` | FK | no | Correspondence language |
| `RegistrationNumber` | `nvarchar(20)` | | yes | Commercial register number |
| `TaxNumber` | `nvarchar(20)` | | yes | Group tax number |
| `AccountingStandard` | `nvarchar(10)` | | yes | `IFRS`, `LOCAL`, `USGAAP` |
| `ConsolidationRelevant` | `bit` | | no | Included in consolidation |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.CompanyCode`
**Smallest unit producing a complete set of books** · reference: `T001`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCode` | `nvarchar(4)` | AK | no | Company code key (e.g. `KH01`) |
| `CompanyId` | `bigint` | FK | no | Owning company |
| `Name` | `nvarchar(60)` | | no | Company code name |
| `Name2` | `nvarchar(60)` | | yes | Additional name |
| `City` | `nvarchar(40)` | | yes | Registered city |
| `CountryCode` | `nvarchar(3)` | FK | no | Country — drives tax and address format |
| `LocalCurrencyCode` | `nvarchar(5)` | FK | no | Company code (local) currency |
| `GroupCurrencyCode` | `nvarchar(5)` | FK | yes | Group currency, defaulted from company |
| `HardCurrencyCode` | `nvarchar(5)` | FK | yes | Hard currency for high-inflation countries |
| `IndexCurrencyCode` | `nvarchar(5)` | FK | yes | Index-based currency |
| `LanguageCode` | `nvarchar(2)` | FK | no | Correspondence language |
| `ChartOfAccountsId` | `bigint` | FK | no | Operational chart of accounts |
| `CountryChartOfAccountsId` | `bigint` | FK | yes | Country-specific chart of accounts |
| `FiscalYearVariantId` | `bigint` | FK | no | Fiscal year variant |
| `PostingPeriodVariantId` | `bigint` | FK | no | Posting period variant |
| `FieldStatusVariantId` | `bigint` | FK | no | Field status variant |
| `CreditControlAreaId` | `bigint` | FK | yes | Default credit control area |
| `ControllingAreaId` | `bigint` | FK | yes | Assigned controlling area |
| `TaxJurisdictionSchemaId` | `bigint` | FK | yes | Tax jurisdiction schema |
| `VatRegistrationNumber` | `nvarchar(20)` | | yes | VAT registration number |
| `AddressId` | `bigint` | FK | yes | Registered address |
| `MaxExchangeRateDeviationPercent` | `decimal(9,4)` | | yes | Warning threshold on manual rates |
| `ProposeFiscalYear` | `bit` | | no | Propose fiscal year on entry screens |
| `NegativePostingAllowed` | `bit` | | no | Allow negative postings on reversal |
| `BusinessAreaFinancialStatements` | `bit` | | no | Business-area balance sheets required |
| `ProfitCenterMandatory` | `bit` | | no | Profit centre required on every line |
| `SegmentMandatory` | `bit` | | no | Segment required on every line |
| `DocumentEntryScreenVariant` | `nvarchar(4)` | | yes | Entry screen variant |
| `IsProductive` | `bit` | | no | Productive — deletion of test data blocked |
| `ValidFrom` | `date` | | no | First day the company code may post |
| `ValidTo` | `date` | | no | Last day the company code may post |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.BusinessArea`
**Cross-company-code reporting unit** · reference: `TGSB`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessArea` | `nvarchar(4)` | AK | no | Business area key |
| `Name` | `nvarchar(40)` | | no | Description |
| `ConsolidationBusinessArea` | `nvarchar(4)` | | yes | Consolidation business area |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.FunctionalArea`
**Cost-of-sales classification** · reference: `TFKB`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FunctionalArea` | `nvarchar(16)` | AK | no | Functional area key |
| `Name` | `nvarchar(40)` | | no | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.Segment`
**Segment for segment reporting (IFRS 8)** · reference: `FAGL_SEGM`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `Segment` | `nvarchar(10)` | AK | no | Segment key |
| `Name` | `nvarchar(40)` | | no | Description |
| `DerivationPriority` | `int` | | no | Order in which derivation rules are evaluated |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.Plant`
**Operational site / production or storage location** · reference: `T001W`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `Plant` | `nvarchar(4)` | AK | no | Plant key |
| `CompanyCodeId` | `bigint` | FK | no | Owning company code |
| `Name` | `nvarchar(60)` | | no | Plant name |
| `CountryCode` | `nvarchar(3)` | FK | no | Country |
| `City` | `nvarchar(40)` | | yes | City |
| `AddressId` | `bigint` | FK | yes | Plant address |
| `PurchasingOrganizationId` | `bigint` | FK | yes | Default purchasing organisation |
| `ValuationArea` | `nvarchar(4)` | | yes | Valuation area (future inventory module) |
| `FactoryCalendarId` | `bigint` | FK | yes | Factory calendar |
| `TaxJurisdictionCode` | `nvarchar(15)` | | yes | Tax jurisdiction |
| *include* | `#VALIDITY` | | | Assignment validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.Branch`
**Branch / office of a company code** · reference: `J_1BBRANCH`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `Branch` | `nvarchar(4)` | AK | no | Branch key |
| `CompanyCodeId` | `bigint` | FK | no | Owning company code |
| `Name` | `nvarchar(60)` | | no | Branch name |
| `LocationId` | `bigint` | FK | yes | Physical location |
| `BusinessAreaId` | `bigint` | FK | yes | Default business area |
| `ProfitCenterId` | `bigint` | FK | yes | Default profit centre |
| `TaxNumber` | `nvarchar(20)` | | yes | Branch tax registration |
| `IsHeadOffice` | `bit` | | no | Head-office branch flag |
| *include* | `#VALIDITY` | | | Assignment validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.Location`
**Physical location (address anchor)** · reference: `TLOC`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `LocationCode` | `nvarchar(10)` | AK | no | Location key |
| `Name` | `nvarchar(60)` | | no | Location name |
| `AddressId` | `bigint` | FK | yes | Address |
| `CountryCode` | `nvarchar(3)` | FK | no | Country |
| `RegionCode` | `nvarchar(3)` | FK | yes | Region / province |
| `Latitude` | `decimal(9,6)` | | yes | Geo latitude |
| `Longitude` | `decimal(9,6)` | | yes | Geo longitude |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.Department`
**Internal department, used for approval routing** · reference: `T527X`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DepartmentCode` | `nvarchar(10)` | AK | no | Department key |
| `CompanyCodeId` | `bigint` | FK | no | Owning company code |
| `ParentDepartmentId` | `bigint` | FK | yes | Parent department (hierarchy) |
| `Name` | `nvarchar(60)` | | no | Department name |
| `CostCenterId` | `bigint` | FK | yes | Default cost centre |
| `ManagerBusinessPartnerId` | `bigint` | FK | yes | Head of department (BP, employee role) |
| *include* | `#VALIDITY` | | | Assignment validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.SalesOrganization`
**Sales organisation** · reference: `TVKO`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SalesOrganization` | `nvarchar(4)` | AK | no | Sales organisation key |
| `CompanyCodeId` | `bigint` | FK | no | Assigned company code |
| `Name` | `nvarchar(60)` | | no | Description |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Statistics currency |
| `AddressId` | `bigint` | FK | yes | Address |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.DistributionChannel`
**Distribution channel** · reference: `TVTW`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DistributionChannel` | `nvarchar(2)` | AK | no | Distribution channel key |
| `Name` | `nvarchar(40)` | | no | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.Division`
**Product division** · reference: `TSPA`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `Division` | `nvarchar(2)` | AK | no | Division key |
| `Name` | `nvarchar(40)` | | no | Description |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.SalesArea`
**Permitted sales organisation / channel / division combination** · reference: `TVTA`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SalesOrganizationId` | `bigint` | AK,FK | no | Sales organisation |
| `DistributionChannelId` | `bigint` | AK,FK | no | Distribution channel |
| `DivisionId` | `bigint` | AK,FK | no | Division |
| `IsActive` | `bit` | | no | Combination allowed |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.PurchasingOrganization`
**Purchasing organisation** · reference: `T024E`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PurchasingOrganization` | `nvarchar(4)` | AK | no | Purchasing organisation key |
| `CompanyCodeId` | `bigint` | FK | yes | Assigned company code (blank = cross-company) |
| `Name` | `nvarchar(60)` | | no | Description |
| `IsCrossCompany` | `bit` | | no | May purchase for several company codes |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.ControllingArea`
**Controlling area — the CO boundary** · reference: `TKA01`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingArea` | `nvarchar(4)` | AK | no | Controlling area key |
| `Name` | `nvarchar(40)` | | no | Description |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Controlling area currency |
| `CurrencyTypeCode` | `nvarchar(2)` | | no | `10` company code, `20` controlling area, `30` group |
| `ChartOfAccountsId` | `bigint` | FK | no | Chart of accounts — must match assigned company codes |
| `FiscalYearVariantId` | `bigint` | FK | no | Fiscal year variant |
| `CostCenterStandardHierarchy` | `nvarchar(12)` | | no | Standard cost centre hierarchy id |
| `ProfitCenterStandardHierarchy` | `nvarchar(12)` | | yes | Standard profit centre hierarchy id |
| `AssignmentControl` | `nvarchar(1)` | | no | `1` one company code, `2` cross-company-code |
| `OperatingConcernId` | `bigint` | FK | yes | Assigned operating concern |
| `ReconciliationLedgerActive` | `bit` | | no | Reconciliation ledger active |
| `ProfitCenterAccountingActive` | `bit` | | no | Profit centre accounting active |
| `ActivateCommitmentManagement` | `bit` | | no | Commitment management active |
| `ValidFromYear` | `smallint` | | no | First fiscal year the area is usable |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.ControllingAreaCompanyCode`
**Company codes assigned to a controlling area** · reference: `TKA02`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ControllingAreaId` | `bigint` | AK,FK | no | Controlling area |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| *include* | `#VALIDITY` | | | Assignment validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.OperatingConcern`
**Operating concern for profitability analysis** · reference: `TKEB`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `OperatingConcern` | `nvarchar(4)` | AK | no | Operating concern key |
| `Name` | `nvarchar(40)` | | no | Description |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Operating concern currency |
| `FiscalYearVariantId` | `bigint` | FK | no | Fiscal year variant |
| `IsCostingBased` | `bit` | | no | Costing-based profitability analysis active |
| `IsAccountBased` | `bit` | | no | Account-based profitability analysis active |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.CreditControlArea`
**Credit control area** · reference: `T014`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CreditControlArea` | `nvarchar(4)` | AK | no | Credit control area key |
| `Name` | `nvarchar(40)` | | no | Description |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Credit limit currency |
| `DefaultCreditLimit` | `decimal(19,4)` | | yes | Default limit for new customers |
| `RiskCategory` | `nvarchar(3)` | | yes | Default risk category |
| `UpdateGroup` | `nvarchar(6)` | | yes | Credit exposure update group |
| `AllOrganizationsAllowed` | `bit` | | no | May be used by any company code |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.OrganizationalAssignment`
**Validity-dated assignment between any two organisational objects** · reference: assignment views in SPRO

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SourceObjectType` | `nvarchar(20)` | AK | no | e.g. `Plant`, `SalesOrganization` |
| `SourceObjectId` | `bigint` | AK | no | Id of the source object |
| `TargetObjectType` | `nvarchar(20)` | AK | no | e.g. `CompanyCode`, `ControllingArea` |
| `TargetObjectId` | `bigint` | AK | no | Id of the target object |
| `AssignmentType` | `nvarchar(20)` | AK | no | `AssignedTo`, `SuppliesTo`, `ReportsTo` |
| `IsPrimary` | `bit` | | no | Primary assignment when several exist |
| *include* | `#VALIDITY` | | | Assignment validity — no overlaps per type |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.FactoryCalendar`
**Working-day calendar for due-date and depreciation calculation** · reference: `TFACD`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CalendarCode` | `nvarchar(2)` | AK | no | Factory calendar key |
| `Name` | `nvarchar(40)` | | no | Description |
| `CountryCode` | `nvarchar(3)` | FK | yes | Public holiday country |
| `WorkingDaysMask` | `nvarchar(7)` | | no | 7 chars `Mon..Sun`, `X` = working day |
| `ValidFromYear` | `smallint` | | no | First year covered |
| `ValidToYear` | `smallint` | | no | Last year covered |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `org.FactoryCalendarHoliday`
**Non-working day of a factory calendar** · reference: `THOC`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `FactoryCalendarId` | `bigint` | AK,FK | no | Owning calendar |
| `HolidayDate` | `date` | AK | no | Non-working date |
| `Name` | `nvarchar(40)` | | no | Holiday name |
| `IsHalfDay` | `bit` | | no | Half working day |
| *include* | `#AUDIT` | | | Standard audit columns |
