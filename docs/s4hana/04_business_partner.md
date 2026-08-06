# 04 — Business Partner and Master Data (`mdm`)

One central **Business Partner** identity. Customer and vendor are *roles* on
that identity, never separate records — `mdm.BusinessPartnerCustomer` and
`mdm.BusinessPartnerVendor` are role extensions keyed by `BusinessPartnerId`,
so the same partner can be both without duplicating name, address or tax data.

Role synchronisation: assigning role `FLCU01` (Customer) creates the
`BusinessPartnerCustomer` row and, per company code, the
`BusinessPartnerCompanyCode` row with the customer reconciliation account —
without touching the general identity.

---

## 4.1 Business Partner core

### `mdm.BusinessPartner`
**Central business partner (general data)** · reference: `BUT000`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PartnerNumber` | `nvarchar(10)` | AK | no | Business partner number |
| `PartnerCategory` | `nvarchar(1)` | | no | `1` Person, `2` Organization, `3` Group |
| `BusinessPartnerGroupId` | `bigint` | FK | no | BP grouping — drives the number range |
| `PartnerType` | `nvarchar(4)` | | yes | Additional classification |
| `Title` | `nvarchar(4)` | | yes | Form of address key |
| `Name1` | `nvarchar(40)` | IX | yes | Organization name 1 / last name |
| `Name2` | `nvarchar(40)` | | yes | Organization name 2 / first name |
| `Name3` | `nvarchar(40)` | | yes | Name line 3 |
| `Name4` | `nvarchar(40)` | | yes | Name line 4 |
| `FullName` | `nvarchar(160)` | IX | no | Formatted display name (computed on save) |
| `SearchTerm1` | `nvarchar(20)` | IX | yes | Search term 1 |
| `SearchTerm2` | `nvarchar(20)` | IX | yes | Search term 2 |
| `FirstName` | `nvarchar(40)` | | yes | First name (person) |
| `LastName` | `nvarchar(40)` | | yes | Last name (person) |
| `MiddleName` | `nvarchar(40)` | | yes | Middle name (person) |
| `NickName` | `nvarchar(40)` | | yes | Nickname |
| `DateOfBirth` | `date` | | yes | Date of birth (person) |
| `PlaceOfBirth` | `nvarchar(40)` | | yes | Place of birth |
| `Gender` | `nvarchar(1)` | | yes | `1` male, `2` female, `3` not specified |
| `MaritalStatus` | `nvarchar(1)` | | yes | Marital status key |
| `NationalityCountryCode` | `nvarchar(3)` | FK | yes | Nationality |
| `LanguageCode` | `nvarchar(2)` | FK | no | Correspondence language |
| `IndustrySector` | `nvarchar(10)` | | yes | Industry key |
| `LegalForm` | `nvarchar(2)` | | yes | Legal form of the organisation |
| `LegalEntityCode` | `nvarchar(2)` | | yes | Legal entity classification |
| `RegistrationNumber` | `nvarchar(20)` | | yes | Commercial register number |
| `RegistrationCountryCode` | `nvarchar(3)` | FK | yes | Country of registration |
| `RegistrationDate` | `date` | | yes | Date of registration |
| `FoundationDate` | `date` | | yes | Foundation / establishment date |
| `LiquidationDate` | `date` | | yes | Liquidation date |
| `EmployeeCount` | `int` | | yes | Number of employees |
| `AnnualRevenue` | `decimal(19,4)` | | yes | Annual revenue |
| `AnnualRevenueCurrencyCode` | `nvarchar(5)` | FK | yes | Currency of the revenue figure |
| `DefaultAddressId` | `bigint` | FK | yes | Standard address |
| `Status` | `nvarchar(20)` | | no | `Active`, `Blocked`, `MarkedForDeletion`, `Archived` |
| `IsCentralBlocked` | `bit` | | no | Central posting block |
| `IsMarkedForDeletion` | `bit` | | no | Central deletion flag |
| `IsIntercompany` | `bit` | | no | Group company partner |
| `TradingPartnerCompany` | `nvarchar(6)` | FK | yes | Company key for intercompany elimination |
| `ExternalPartnerNumber` | `nvarchar(20)` | | yes | Number in a legacy or external system |
| *include* | `#VALIDITY` | | | Partner validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerGroup`
**BP grouping — number range and screen defaults** · reference: `TB001` / `T077D`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `GroupCode` | `nvarchar(4)` | AK | no | Grouping key |
| `Name` | `nvarchar(60)` | | no | Description |
| `NumberRangeObjectId` | `bigint` | FK | no | Number range object |
| `NumberRangeCode` | `nvarchar(2)` | | no | Number range interval |
| `IsExternalNumbering` | `bit` | | no | Number entered by the user |
| `PartnerCategory` | `nvarchar(1)` | | yes | Restricted to this category |
| `IsOneTimeAccount` | `bit` | | no | One-time account (CpD) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerRole`
**Catalogue of available BP roles** · reference: `TB003`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RoleCode` | `nvarchar(6)` | AK | no | Role key, e.g. `000000` general, `FLCU00`/`FLCU01` customer, `FLVN00`/`FLVN01` vendor |
| `Name` | `nvarchar(60)` | | no | Role name |
| `RoleCategory` | `nvarchar(20)` | | no | `General`, `Customer`, `FICustomer`, `Vendor`, `FIVendor`, `Employee`, `ContactPerson`, `Bank`, `Intercompany` |
| `RequiresCompanyCodeData` | `bit` | | no | Company code segment required |
| `RequiresSalesArea` | `bit` | | no | Sales area segment required |
| `RequiresPurchasingOrganization` | `bit` | | no | Purchasing segment required |
| `SyncTargetEntity` | `nvarchar(40)` | | yes | Entity created by role synchronisation |
| `IsStandardRole` | `bit` | | no | Delivered role (not customer-defined) |
| `DisplayOrder` | `int` | | no | Order in the role list |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerRoleAssignment`
**Roles held by a partner, with validity** · reference: `BUT100`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `BusinessPartnerRoleId` | `bigint` | AK,FK | no | Role |
| `IsSynchronized` | `bit` | | no | Role-specific data has been created |
| `SynchronizedAt` | `datetime2(3)` | | yes | Last synchronisation (UTC) |
| `SyncStatus` | `nvarchar(20)` | | no | `Pending`, `Completed`, `Failed` |
| `SyncMessage` | `nvarchar(255)` | | yes | Last synchronisation message |
| *include* | `#VALIDITY` | | | Role validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerIdentification`
**Identification numbers (passport, licence, registration)** · reference: `BUT0ID`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `IdentificationType` | `nvarchar(6)` | AK | no | `PASSPORT`, `NATID`, `DRVLIC`, `BUSLIC` |
| `IdentificationNumber` | `nvarchar(60)` | AK | no | Number (masked for unauthorised users) |
| `IssuingInstitution` | `nvarchar(40)` | | yes | Issuing authority |
| `IssuingCountryCode` | `nvarchar(3)` | FK | yes | Country of issue |
| `IssuingRegionCode` | `nvarchar(3)` | | yes | Region of issue |
| `IssueDate` | `date` | | yes | Date of issue |
| `ExpiryDate` | `date` | | yes | Expiry date |
| `IsPersonalData` | `bit` | | no | Subject to masking and retention rules |
| *include* | `#VALIDITY` | | | Validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.Address`
**Central address record** · reference: `ADRC`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AddressNumber` | `nvarchar(10)` | AK | no | Address number |
| `Title` | `nvarchar(4)` | | yes | Form of address |
| `Name1` | `nvarchar(40)` | | yes | Name line 1 |
| `Name2` | `nvarchar(40)` | | yes | Name line 2 |
| `Street` | `nvarchar(60)` | | yes | Street |
| `HouseNumber` | `nvarchar(10)` | | yes | House number |
| `StreetSupplement1` | `nvarchar(40)` | | yes | Street line 2 |
| `StreetSupplement2` | `nvarchar(40)` | | yes | Street line 3 |
| `District` | `nvarchar(40)` | | yes | District / khan / sangkat |
| `City` | `nvarchar(40)` | IX | yes | City |
| `PostalCode` | `nvarchar(10)` | | yes | Postal code |
| `PoBox` | `nvarchar(10)` | | yes | PO box |
| `PoBoxPostalCode` | `nvarchar(10)` | | yes | PO box postal code |
| `RegionCode` | `nvarchar(3)` | FK | yes | Region / province |
| `CountryCode` | `nvarchar(3)` | FK | no | Country |
| `LanguageCode` | `nvarchar(2)` | FK | no | Address language |
| `TimeZoneId` | `nvarchar(64)` | | yes | Time zone |
| `Latitude` | `decimal(9,6)` | | yes | Geo latitude |
| `Longitude` | `decimal(9,6)` | | yes | Geo longitude |
| `FormattedAddress` | `nvarchar(500)` | | yes | Address rendered in the country format |
| *include* | `#VALIDITY` | | | Address validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerAddress`
**Address usage of a partner** · reference: `BUT020`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `AddressId` | `bigint` | AK,FK | no | Address |
| `AddressUsage` | `nvarchar(10)` | AK | no | `STANDARD`, `BILLTO`, `SHIPTO`, `DELIVERY`, `HOME`, `WORK` |
| `IsStandardAddress` | `bit` | | no | Standard address for the usage |
| *include* | `#VALIDITY` | | | Usage validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerCommunication`
**Communication channel of a partner or address** · reference: `ADR2` / `ADR6`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `AddressId` | `bigint` | FK | yes | Address the channel belongs to |
| `CommunicationType` | `nvarchar(10)` | AK | no | `PHONE`, `MOBILE`, `FAX`, `EMAIL`, `WEB`, `TELEX` |
| `SequenceNumber` | `int` | AK | no | Sequence within the type |
| `CountryDialCode` | `nvarchar(5)` | | yes | Country dialling code |
| `Value` | `nvarchar(241)` | | no | Number, address or URL |
| `Extension` | `nvarchar(10)` | | yes | Telephone extension |
| `IsDefault` | `bit` | | no | Standard channel of the type |
| `IsMarketingAllowed` | `bit` | | no | Consent for marketing contact |
| `Notes` | `nvarchar(255)` | | yes | Notes |
| *include* | `#VALIDITY` | | | Validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerBank`
**Bank details of a partner** · reference: `BUT0BK`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `BankDetailId` | `nvarchar(4)` | AK | no | Bank details id, e.g. `0001` |
| `BankCountryCode` | `nvarchar(3)` | FK | no | Bank country |
| `BankKey` | `nvarchar(15)` | FK | no | Bank key / routing code |
| `BankAccountNumber` | `nvarchar(18)` | | yes | Account number (masked in the browser) |
| `BankAccountHolder` | `nvarchar(60)` | | yes | Account holder name |
| `Iban` | `nvarchar(34)` | | yes | IBAN |
| `SwiftCode` | `nvarchar(11)` | | yes | SWIFT / BIC |
| `BankControlKey` | `nvarchar(2)` | | yes | Bank control key |
| `CurrencyCode` | `nvarchar(5)` | FK | yes | Account currency |
| `IsDefaultForPayment` | `bit` | | no | Default account for payments |
| `PaymentMethodRestriction` | `nvarchar(10)` | | yes | Payment methods allowed on this account |
| `IsVerified` | `bit` | | no | Verified by a second user (maker-checker) |
| `VerifiedBy` | `nvarchar(64)` | | yes | Verifying user |
| `VerifiedAt` | `datetime2(3)` | | yes | Verification timestamp (UTC) |
| *include* | `#VALIDITY` | | | Validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerTaxNumber`
**Tax numbers of a partner** · reference: `BUT0TX`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `TaxNumberCategory` | `nvarchar(6)` | AK | no | Category, e.g. `KH0` VAT TIN, `EU0` VAT id |
| `TaxNumber` | `nvarchar(20)` | AK | no | Tax number |
| `CountryCode` | `nvarchar(3)` | FK | no | Issuing country |
| `IsNaturalPerson` | `bit` | | no | Natural person indicator |
| `IsValidated` | `bit` | | no | Format validated against the country rule |
| `ValidatedAt` | `datetime2(3)` | | yes | Validation timestamp (UTC) |
| *include* | `#VALIDITY` | | | Validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerRelationship`
**Validity-dated relationship between two partners** · reference: `BUT050`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RelationshipNumber` | `nvarchar(12)` | AK | no | Relationship number |
| `RelationshipCategory` | `nvarchar(6)` | | no | `PARENT`, `SUBSID`, `CONTACT`, `EMPLOY`, `SOLDTO`, `SHIPTO`, `BILLTO`, `PAYER`, `SUPPL`, `RELCO`, `ICOMP` |
| `SourceBusinessPartnerId` | `bigint` | FK,IX | no | Partner 1 |
| `TargetBusinessPartnerId` | `bigint` | FK,IX | no | Partner 2 |
| `IsDirectional` | `bit` | | no | Relationship has a direction |
| `RelationshipRole` | `nvarchar(40)` | | yes | Role of the target in the relationship |
| `ShareholdingPercent` | `decimal(9,4)` | | yes | Ownership percentage |
| `IsStandard` | `bit` | | no | Default partner of this category |
| `DepartmentText` | `nvarchar(40)` | | yes | Department (contact person) |
| `FunctionText` | `nvarchar(40)` | | yes | Function (contact person) |
| `Notes` | `nvarchar(255)` | | yes | Notes |
| *include* | `#VALIDITY` | | | Relationship validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 4.2 Company-code and role-specific data

### `mdm.BusinessPartnerCompanyCode`
**Company code segment (customer and vendor)** · reference: `KNB1` / `LFB1`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `RoleCategory` | `nvarchar(20)` | AK | no | `Customer` or `Vendor` — one segment per side |
| `ReconciliationGLAccountId` | `bigint` | FK | no | Reconciliation account — must be flagged as such |
| `AlternativePayerPayeeId` | `bigint` | FK | yes | Alternative payer / payee partner |
| `HeadOfficePartnerId` | `bigint` | FK | yes | Head office for branch accounting |
| `SortKey` | `nvarchar(3)` | | yes | Allocation (assignment) field rule |
| `PlanningGroup` | `nvarchar(10)` | | yes | Cash management planning group |
| `PaymentTermsId` | `bigint` | FK | yes | Payment terms |
| `PaymentMethods` | `nvarchar(10)` | | yes | Permitted payment methods |
| `PaymentBlockReason` | `nvarchar(1)` | | yes | Payment block key |
| `IsPostingBlocked` | `bit` | | no | Posting block for this company code |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag for this company code |
| `HouseBankId` | `bigint` | FK | yes | House bank used for payments |
| `PaymentGroupingKey` | `nvarchar(2)` | | yes | Grouping key for the payment run |
| `IsIndividualPayment` | `bit` | | no | Pay each item separately |
| `ToleranceGroupId` | `bigint` | FK | yes | Tolerance group |
| `DunningProcedureId` | `bigint` | FK | yes | Dunning procedure |
| `DunningRecipientPartnerId` | `bigint` | FK | yes | Alternative dunning recipient |
| `DunningBlockReason` | `nvarchar(1)` | | yes | Dunning block |
| `LastDunningDate` | `date` | | yes | Date of the last dunning run |
| `DunningLevel` | `tinyint` | | yes | Current dunning level |
| `DunningClerk` | `nvarchar(2)` | | yes | Dunning clerk |
| `AccountingClerk` | `nvarchar(2)` | | yes | Accounting clerk |
| `WithholdingTaxCodeId` | `bigint` | FK | yes | Withholding tax code |
| `IsWithholdingTaxExempt` | `bit` | | no | Exempt from withholding tax |
| `WithholdingTaxExemptionNumber` | `nvarchar(20)` | | yes | Exemption certificate number |
| `WithholdingTaxExemptionValidTo` | `date` | | yes | Certificate expiry |
| `IsClearingWithVendorAllowed` | `bit` | | no | Clearing between customer and vendor side allowed |
| `ClearingPartnerId` | `bigint` | FK | yes | Partner used for cross-clearing |
| `InterestCalculationIndicator` | `nvarchar(2)` | | yes | Interest calculation indicator |
| `LastInterestRunDate` | `date` | | yes | Last interest calculation |
| `CorrespondenceType` | `nvarchar(20)` | | yes | Default correspondence type |
| `LocalCurrencyCode` | `nvarchar(5)` | FK | yes | Account currency, if not the company code currency |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerCustomer`
**Customer role — general customer data** · reference: `KNA1`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `CustomerNumber` | `nvarchar(10)` | AK | no | Customer account number |
| `CustomerAccountGroupId` | `bigint` | FK | no | Customer account group |
| `CustomerClassification` | `nvarchar(2)` | | yes | Customer classification |
| `IndustryKey` | `nvarchar(4)` | | yes | Industry |
| `CustomerGroup` | `nvarchar(2)` | | yes | Customer group for statistics |
| `NielsenIndicator` | `nvarchar(2)` | | yes | Regional market indicator |
| `TaxClassification` | `nvarchar(1)` | | yes | Tax classification of the customer |
| `VatRegistrationNumber` | `nvarchar(20)` | | yes | VAT registration number |
| `IsOneTimeCustomer` | `bit` | | no | One-time account |
| `IsOrderBlocked` | `bit` | | no | Order block |
| `IsDeliveryBlocked` | `bit` | | no | Delivery block |
| `IsBillingBlocked` | `bit` | | no | Billing block |
| `IsPostingBlocked` | `bit` | | no | Central posting block |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| `TransportationZone` | `nvarchar(10)` | | yes | Transportation zone |
| `AuthorizationGroup` | `nvarchar(4)` | | yes | Authorization group |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerVendor`
**Vendor / supplier role — general vendor data** · reference: `LFA1`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `VendorNumber` | `nvarchar(10)` | AK | no | Vendor account number |
| `VendorAccountGroupId` | `bigint` | FK | no | Vendor account group |
| `VendorClassification` | `nvarchar(2)` | | yes | Vendor classification |
| `IndustryKey` | `nvarchar(4)` | | yes | Industry |
| `VatRegistrationNumber` | `nvarchar(20)` | | yes | VAT registration number |
| `TaxNumberType` | `nvarchar(2)` | | yes | Tax number type |
| `IsOneTimeVendor` | `bit` | | no | One-time account |
| `IsPurchasingBlocked` | `bit` | | no | Purchasing block |
| `IsPostingBlocked` | `bit` | | no | Central posting block |
| `IsPaymentBlocked` | `bit` | | no | Central payment block |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| `IsServiceProvider` | `bit` | | no | Service provider (withholding tax relevance) |
| `IsSubcontractor` | `bit` | | no | Subcontractor |
| `QualityRating` | `nvarchar(2)` | | yes | Supplier quality rating |
| `AuthorizationGroup` | `nvarchar(4)` | | yes | Authorization group |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerSalesArea`
**Customer sales area data** · reference: `KNVV`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `SalesAreaId` | `bigint` | AK,FK | no | Sales organisation / channel / division |
| `CustomerGroup` | `nvarchar(2)` | | yes | Customer group |
| `SalesDistrict` | `nvarchar(6)` | | yes | Sales district |
| `SalesOffice` | `nvarchar(4)` | | yes | Sales office |
| `SalesGroup` | `nvarchar(3)` | | yes | Sales group |
| `PriceGroup` | `nvarchar(2)` | | yes | Price group |
| `PriceListType` | `nvarchar(2)` | | yes | Price list type |
| `CustomerPricingProcedure` | `nvarchar(1)` | | yes | Pricing procedure determination |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Order currency |
| `PaymentTermsId` | `bigint` | FK | yes | Sales payment terms |
| `Incoterms1` | `nvarchar(3)` | | yes | Incoterms part 1 |
| `Incoterms2` | `nvarchar(28)` | | yes | Incoterms part 2 (location) |
| `ShippingConditions` | `nvarchar(2)` | | yes | Shipping conditions |
| `DeliveryPriority` | `nvarchar(2)` | | yes | Delivery priority |
| `DeliveringPlantId` | `bigint` | FK | yes | Default delivering plant |
| `IsCompleteDeliveryRequired` | `bit` | | no | Complete delivery required |
| `PartialDeliveryPerItem` | `nvarchar(1)` | | yes | Partial delivery indicator |
| `MaxPartialDeliveries` | `tinyint` | | yes | Maximum partial deliveries |
| `OrderCombinationAllowed` | `bit` | | no | Order combination permitted |
| `AccountAssignmentGroup` | `nvarchar(2)` | | yes | Account assignment group for revenue determination |
| `TaxClassification` | `nvarchar(1)` | | yes | Tax classification |
| `IsOrderBlocked` | `bit` | | no | Sales area order block |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerPurchasingOrganization`
**Vendor purchasing organisation data** · reference: `LFM1`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `PurchasingOrganizationId` | `bigint` | AK,FK | no | Purchasing organisation |
| `PurchasingGroup` | `nvarchar(3)` | | yes | Purchasing group |
| `OrderCurrencyCode` | `nvarchar(5)` | FK | no | Order currency |
| `PaymentTermsId` | `bigint` | FK | yes | Purchasing payment terms |
| `Incoterms1` | `nvarchar(3)` | | yes | Incoterms part 1 |
| `Incoterms2` | `nvarchar(28)` | | yes | Incoterms part 2 (location) |
| `MinimumOrderValue` | `decimal(19,4)` | | yes | Minimum order value |
| `IsGoodsReceiptBasedInvoiceVerification` | `bit` | | no | GR-based invoice verification |
| `IsGoodsReceiptRequired` | `bit` | | no | Goods receipt expected |
| `IsInvoiceReceiptRequired` | `bit` | | no | Invoice receipt expected |
| `IsEvaluatedReceiptSettlement` | `bit` | | no | ERS (self-billing) |
| `IsAutomaticPurchaseOrderAllowed` | `bit` | | no | Automatic PO generation allowed |
| `IsReturnsVendor` | `bit` | | no | Returns vendor |
| `SchemaGroup` | `nvarchar(2)` | | yes | Pricing schema group |
| `PlannedDeliveryDays` | `int` | | yes | Planned delivery time |
| `IsPurchasingBlocked` | `bit` | | no | Block for this purchasing organisation |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerCreditProfile`
**Credit management segment** · reference: `UKMBP_CMS_SGM`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Business partner |
| `CreditControlAreaId` | `bigint` | AK,FK | no | Credit control area |
| `CreditLimit` | `decimal(19,4)` | | no | Credit limit |
| `CreditLimitCurrencyCode` | `nvarchar(5)` | FK | no | Limit currency |
| `CreditExposure` | `decimal(19,4)` | | no | Current exposure (open items + orders) |
| `CreditLimitUsedPercent` | `decimal(9,4)` | | no | Utilisation in percent |
| `RiskCategory` | `nvarchar(3)` | | yes | Risk category |
| `CreditRating` | `nvarchar(10)` | | yes | Internal or external rating |
| `RatingAgency` | `nvarchar(20)` | | yes | Rating source |
| `CreditLimitValidTo` | `date` | | yes | Limit expiry |
| `LastReviewDate` | `date` | | yes | Last credit review |
| `NextReviewDate` | `date` | | yes | Next scheduled review |
| `IsCreditBlocked` | `bit` | | no | Credit block active |
| `BlockReason` | `nvarchar(255)` | | yes | Reason for the block |
| `PaymentBehaviourIndex` | `decimal(9,4)` | | yes | Average days beyond terms |
| `HighestDunningLevel` | `tinyint` | | yes | Highest dunning level reached |
| `OldestOpenItemDate` | `date` | | yes | Date of the oldest open item |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.BusinessPartnerAttachment`
**Document attached to a partner**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BusinessPartnerId` | `bigint` | FK,IX | no | Business partner |
| `FileName` | `nvarchar(255)` | | no | Original file name |
| `ContentType` | `nvarchar(100)` | | no | MIME type |
| `FileSizeBytes` | `bigint` | | no | Size in bytes |
| `StorageUri` | `nvarchar(500)` | | no | Location in the document store |
| `DocumentCategory` | `nvarchar(40)` | | yes | `Contract`, `TaxCertificate`, `Registration`, `Other` |
| `Checksum` | `nvarchar(64)` | | no | SHA-256 of the content |
| `UploadedAt` | `datetime2(3)` | | no | Upload timestamp (UTC) |
| `UploadedBy` | `nvarchar(64)` | | no | Uploading user |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 4.3 G/L account and bank master

### `mdm.GLAccount`
**G/L account — chart of accounts area** · reference: `SKA1`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ChartOfAccountsId` | `bigint` | AK,FK | no | Chart of accounts |
| `GLAccount` | `nvarchar(10)` | AK | no | G/L account number |
| `AccountGroupId` | `bigint` | FK | no | Account group |
| `AccountType` | `nvarchar(30)` | | no | `BalanceSheet`, `PrimaryCost`, `SecondaryCost`, `NonOperatingExpenseRevenue`, `CashAccount` |
| `IsBalanceSheetAccount` | `bit` | | no | Balance sheet account |
| `IsProfitAndLossAccount` | `bit` | | no | P&L account |
| `IsReconciliationAccount` | `bit` | | no | Reconciliation account — no direct posting |
| `ReconciliationAccountType` | `nvarchar(1)` | | yes | `D` customer, `K` vendor, `A` asset |
| `RetainedEarningsAccountKey` | `nvarchar(2)` | | yes | P&L statement account type for carry-forward |
| `GroupAccountNumber` | `nvarchar(10)` | | yes | Group chart account |
| `TradingPartnerRequired` | `bit` | | no | Trading partner mandatory |
| `IsBlockedForCreation` | `bit` | | no | Blocked for company-code creation |
| `IsBlockedForPosting` | `bit` | | no | Blocked for posting |
| `IsBlockedForPlanning` | `bit` | | no | Blocked for planning |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| `SampleAccountNumber` | `nvarchar(10)` | | yes | Sample account used as a template |
| `FunctionalAreaId` | `bigint` | FK | yes | Default functional area |
| `CostElementCategory` | `nvarchar(2)` | | yes | CO cost element category (`1`, `11`, `21`, `41`, `42`, `43`) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.GLAccountText`
**G/L account descriptions per language** · reference: `SKAT`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `GLAccountId` | `bigint` | AK,FK | no | G/L account |
| `LanguageCode` | `nvarchar(2)` | AK,FK | no | Language |
| `ShortText` | `nvarchar(20)` | | no | Short text |
| `LongText` | `nvarchar(50)` | | no | Long text |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.GLAccountCompanyCode`
**G/L account — company code area** · reference: `SKB1`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `GLAccountId` | `bigint` | AK,FK | no | G/L account |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `AccountCurrencyCode` | `nvarchar(5)` | FK | no | Account currency |
| `IsOnlyBalancesInLocalCurrency` | `bit` | | no | Only local currency balances |
| `TaxCategory` | `nvarchar(2)` | | yes | Tax category (`-`, `+`, `*`, tax code) |
| `IsPostingWithoutTaxAllowed` | `bit` | | no | Posting without tax code allowed |
| `IsOpenItemManaged` | `bit` | | no | Open item management |
| `IsLineItemDisplay` | `bit` | | no | Line item display |
| `SortKey` | `nvarchar(3)` | | yes | Rule filling the allocation field |
| `FieldStatusGroupId` | `bigint` | FK | no | Field status group |
| `IsPostAutomaticallyOnly` | `bit` | | no | Post automatically only |
| `IsReconciliationAccountForAccountType` | `nvarchar(1)` | | yes | Reconciliation account type in this company code |
| `IsRelevantToCashFlow` | `bit` | | no | Relevant to cash flow |
| `HouseBankId` | `bigint` | FK | yes | House bank of a bank account |
| `HouseBankAccountId` | `bigint` | FK | yes | House bank account |
| `InterestCalculationIndicator` | `nvarchar(2)` | | yes | Interest indicator |
| `PlanningGroup` | `nvarchar(10)` | | yes | Cash management planning group |
| `IsBlockedForPosting` | `bit` | | no | Blocked for posting in this company code |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| `AuthorizationGroup` | `nvarchar(4)` | | yes | Authorization group |
| `CostCenterRequired` | `bit` | | no | Cost centre mandatory on postings |
| `ProfitCenterRequired` | `bit` | | no | Profit centre mandatory on postings |
| `IsInflationRelevant` | `bit` | | no | Subject to inflation adjustment |
| `ValuationGroup` | `nvarchar(4)` | | yes | Foreign currency valuation group |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.Bank`
**Bank master** · reference: `BNKA`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BankCountryCode` | `nvarchar(3)` | AK,FK | no | Bank country |
| `BankKey` | `nvarchar(15)` | AK | no | Bank key / routing number |
| `BankName` | `nvarchar(60)` | | no | Bank name |
| `BankBranch` | `nvarchar(40)` | | yes | Branch |
| `Street` | `nvarchar(60)` | | yes | Street |
| `City` | `nvarchar(40)` | | yes | City |
| `PostalCode` | `nvarchar(10)` | | yes | Postal code |
| `SwiftCode` | `nvarchar(11)` | | yes | SWIFT / BIC |
| `BankGroup` | `nvarchar(2)` | | yes | Bank group |
| `IsBlocked` | `bit` | | no | Blocked |
| `IsMarkedForDeletion` | `bit` | | no | Deletion flag |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.HouseBank`
**House bank of a company code** · reference: `T012`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `HouseBankCode` | `nvarchar(5)` | AK | no | House bank key |
| `BankId` | `bigint` | FK | no | Bank master record |
| `Description` | `nvarchar(60)` | | yes | Description |
| `PaymentFileFormat` | `nvarchar(20)` | | yes | Payment file format used |
| `IsDefault` | `bit` | | no | Default house bank |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `mdm.HouseBankAccount`
**Bank account of a house bank** · reference: `T012K`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `HouseBankId` | `bigint` | AK,FK | no | House bank |
| `AccountId` | `nvarchar(5)` | AK | no | Account id |
| `BankAccountNumber` | `nvarchar(18)` | | yes | Account number |
| `Iban` | `nvarchar(34)` | | yes | IBAN |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Account currency |
| `GLAccountId` | `bigint` | FK | no | Bank G/L account |
| `ClearingGLAccountId` | `bigint` | FK | yes | Bank clearing account |
| `AccountHolder` | `nvarchar(60)` | | yes | Account holder |
| `OverdraftLimit` | `decimal(19,4)` | | yes | Overdraft limit |
| `IsActive` | `bit` | | no | Active |
| *include* | `#AUDIT` | | | Standard audit columns |
