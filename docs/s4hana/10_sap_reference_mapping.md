# 10 — SAP S/4HANA Reference Mapping

The tables in this catalogue are an original design. This page exists so a
functional consultant can move between the two worlds: for each SAP concept it
shows the **SAP table**, its **key fields with ABAP types**, and the
**table.field in this system** that plays the same role.

Two notes before using it:

* Field names and types below are the commonly documented ones for SAP
  ERP / S/4HANA. Release, industry solution and enhancement packages change
  details — verify against `SE11` in the specific system you are integrating
  with before writing a mapping program.
* In S/4HANA, `BSEG` and the classic index tables (`BSID`, `BSAD`, `BSIK`,
  `BSAK`, `BSIS`, `BSAS`, `GLT0`, `COSP`, `FAGLFLEXA`, …) are largely
  compatibility views over **`ACDOCA`**. This design follows the same idea:
  `fin.JournalEntryLine` is the one line table, and open items and balances are
  maintained indexes over it, not independent sources of truth.

**ABAP type legend** — `CLNT` client, `CHAR` character, `NUMC` numeric text,
`DATS` date (8), `TIMS` time (6), `CUKY` currency key, `CURR` currency amount,
`QUAN` quantity, `UNIT` unit, `DEC` packed decimal, `LANG` language, `RAW`
binary.

---

## 10.1 Enterprise structure

| SAP table | SAP field | ABAP type | Meaning | This system |
|---|---|---|---|---|
| `T001` | `MANDT` | `CLNT(3)` | Client | `org.CompanyCode.TenantId` |
| `T001` | `BUKRS` | `CHAR(4)` | Company code | `org.CompanyCode.CompanyCode` |
| `T001` | `BUTXT` | `CHAR(25)` | Company code name | `org.CompanyCode.Name` |
| `T001` | `ORT01` | `CHAR(25)` | City | `org.CompanyCode.City` |
| `T001` | `LAND1` | `CHAR(3)` | Country key | `org.CompanyCode.CountryCode` |
| `T001` | `WAERS` | `CUKY(5)` | Local currency | `org.CompanyCode.LocalCurrencyCode` |
| `T001` | `SPRAS` | `LANG(1)` | Language key | `org.CompanyCode.LanguageCode` |
| `T001` | `KTOPL` | `CHAR(4)` | Chart of accounts | `org.CompanyCode.ChartOfAccountsId` |
| `T001` | `PERIV` | `CHAR(2)` | Fiscal year variant | `org.CompanyCode.FiscalYearVariantId` |
| `T001` | `RCOMP` | `CHAR(6)` | Company | `org.CompanyCode.CompanyId` |
| `T001` | `ADRNR` | `CHAR(10)` | Address number | `org.CompanyCode.AddressId` |
| `T001` | `STCEG` | `CHAR(20)` | VAT registration number | `org.CompanyCode.VatRegistrationNumber` |
| `T880` | `RCOMP` | `CHAR(6)` | Company | `org.Company.CompanyCodeGroup` |
| `T880` | `CURR` | `CUKY(5)` | Group currency | `org.Company.GroupCurrencyCode` |
| `T001W` | `WERKS` | `CHAR(4)` | Plant | `org.Plant.Plant` |
| `T001W` | `NAME1` | `CHAR(30)` | Plant name | `org.Plant.Name` |
| `T001W` | `FABKL` | `CHAR(2)` | Factory calendar | `org.Plant.FactoryCalendarId` |
| `TGSB` | `GSBER` | `CHAR(4)` | Business area | `org.BusinessArea.BusinessArea` |
| `TKA01` | `KOKRS` | `CHAR(4)` | Controlling area | `org.ControllingArea.ControllingArea` |
| `TKA01` | `WAERS` | `CUKY(5)` | CO area currency | `org.ControllingArea.CurrencyCode` |
| `TKA01` | `KHINR` | `CHAR(12)` | Standard hierarchy | `org.ControllingArea.CostCenterStandardHierarchy` |
| `TKA02` | `BUKRS` | `CHAR(4)` | Assigned company code | `org.ControllingAreaCompanyCode.CompanyCodeId` |
| `T014` | `KKBER` | `CHAR(4)` | Credit control area | `org.CreditControlArea.CreditControlArea` |
| `TVKO` | `VKORG` | `CHAR(4)` | Sales organisation | `org.SalesOrganization.SalesOrganization` |
| `T024E` | `EKORG` | `CHAR(4)` | Purchasing organisation | `org.PurchasingOrganization.PurchasingOrganization` |
| `FAGL_SEGM` | `SEGMENT` | `CHAR(10)` | Segment | `org.Segment.Segment` |
| `TFKB` | `FKBER` | `CHAR(16)` | Functional area | `org.FunctionalArea.FunctionalArea` |

---

## 10.2 Configuration

| SAP table | SAP field | ABAP type | Meaning | This system |
|---|---|---|---|---|
| `T004` | `KTOPL` | `CHAR(4)` | Chart of accounts | `cfg.ChartOfAccounts.ChartOfAccounts` |
| `T004` | `KTPLT` | `CHAR(50)` | Description | `cfg.ChartOfAccounts.Name` |
| `T077S` | `KTOKS` | `CHAR(4)` | G/L account group | `cfg.AccountGroup.AccountGroup` |
| `T009` | `PERIV` | `CHAR(2)` | Fiscal year variant | `cfg.FiscalYearVariant.FiscalYearVariant` |
| `T009` | `ANZBP` | `NUMC(2)` | Number of posting periods | `cfg.FiscalYearVariant.NumberOfPostingPeriods` |
| `T009` | `ANZSP` | `NUMC(2)` | Number of special periods | `cfg.FiscalYearVariant.NumberOfSpecialPeriods` |
| `T009B` | `BUMON` | `NUMC(2)` | Period end month | `cfg.FiscalYearVariantPeriod.CalendarMonth` |
| `T009B` | `BUTAG` | `NUMC(2)` | Period end day | `cfg.FiscalYearVariantPeriod.CalendarDay` |
| `T009B` | `RELJR` | `NUMC(1)` | Year shift | `cfg.FiscalYearVariantPeriod.YearShift` |
| `T009B` | `POPER` | `NUMC(3)` | Posting period | `cfg.FiscalYearVariantPeriod.FiscalPeriod` |
| `T001B` | `MKOAR` | `CHAR(1)` | Account type | `cfg.PostingPeriodControl.AccountType` |
| `T001B` | `FRPE1` / `TOPE1` | `NUMC(3)` | Open period from / to | `cfg.PostingPeriodControl.FromPeriod1` / `.ToPeriod1` |
| `T001B` | `AUGRP` | `CHAR(4)` | Authorization group | `cfg.PostingPeriodControl.AuthorizationGroup` |
| `T003` | `BLART` | `CHAR(2)` | Document type | `cfg.DocumentType.DocumentType` |
| `T003` | `NUMKR` | `CHAR(2)` | Number range | `cfg.DocumentType.NumberRangeCode` |
| `T003` | `STBLA` | `CHAR(2)` | Reverse document type | `cfg.DocumentType.ReverseDocumentType` |
| `T003` | `XKOAD` / `XKOAK` / `XKOAS` / `XKOAA` | `CHAR(1)` | Allowed account types D/K/S/A | `cfg.DocumentType.AllowCustomerAccounts` … |
| `TBSL` | `BSCHL` | `CHAR(2)` | Posting key | `cfg.PostingKey.PostingKey` |
| `TBSL` | `SHKZG` | `CHAR(1)` | Debit/credit indicator | `cfg.PostingKey.DebitCreditIndicator` |
| `TBSL` | `KOART` | `CHAR(1)` | Account type | `cfg.PostingKey.AccountType` |
| `TBSL` | `XSONU` | `CHAR(1)` | Special G/L posting | `cfg.PostingKey.IsSpecialGLPosting` |
| `T004V` | `FSTVA` | `CHAR(4)` | Field status variant | `cfg.FieldStatusVariant.FieldStatusVariant` |
| `T004G` | `FSTAG` | `CHAR(4)` | Field status group | `cfg.FieldStatusGroup.FieldStatusGroup` |
| `T004F` | `FAUS1` / `FAUS2` | `CHAR(n)` | Field status strings | `cfg.FieldStatusFieldControl.FieldStatus` (per field) |
| `NRIV` | `OBJECT` | `CHAR(10)` | Number range object | `cfg.NumberRangeObject.NumberRangeObject` |
| `NRIV` | `NRRANGENR` | `CHAR(2)` | Interval number | `cfg.NumberRangeInterval.NumberRangeCode` |
| `NRIV` | `FROMNUMBER` / `TONUMBER` | `CHAR(20)` | Interval limits | `cfg.NumberRangeInterval.FromNumber` / `.ToNumber` |
| `NRIV` | `NRLEVEL` | `CHAR(20)` | Current number level | `cfg.NumberRangeInterval.CurrentNumber` |
| `TCURC` | `WAERS` | `CUKY(5)` | Currency key | `cfg.Currency.CurrencyCode` |
| `TCURC` | `ISOCD` | `CHAR(3)` | ISO code | `cfg.Currency.IsoCode` |
| `TCURX` | `CURRDEC` | `INT1` | Decimal places | `cfg.CurrencyDecimal.DecimalPlaces` |
| `TCURV` | `KURST` | `CHAR(4)` | Exchange rate type | `cfg.ExchangeRateType.ExchangeRateType` |
| `TCURV` | `XINVR` | `CHAR(1)` | Inversion allowed | `cfg.ExchangeRateType.IsInversionAllowed` |
| `TCURR` | `FCURR` / `TCURR` | `CUKY(5)` | From / to currency | `cfg.ExchangeRate.FromCurrencyCode` / `.ToCurrencyCode` |
| `TCURR` | `GDATU` | `CHAR(8)` | Valid from (inverted date) | `cfg.ExchangeRate.ValidFrom` (stored as a real date) |
| `TCURR` | `UKURS` | `DEC(9,5)` | Exchange rate | `cfg.ExchangeRate.Rate` (`decimal(23,6)`) |
| `TCURF` | `FFACT` / `TFACT` | `DEC(9)` | Ratios | `cfg.CurrencyTranslationRatio.FromRatio` / `.ToRatio` |
| `T005` | `LAND1` | `CHAR(3)` | Country key | `cfg.Country.CountryCode` |
| `T005S` | `BLAND` | `CHAR(3)` | Region | `cfg.Region.RegionCode` |
| `T006` | `MSEHI` | `UNIT(3)` | Unit of measure | `cfg.UnitOfMeasure.UnitOfMeasure` |
| `T007A` | `MWSKZ` | `CHAR(2)` | Tax code | `cfg.TaxCode.TaxCode` |
| `T007A` | `KALSM` | `CHAR(6)` | Tax procedure | `cfg.TaxCode.TaxCategory` context |
| `T007A` | `MWART` | `CHAR(1)` | Tax type (`V`/`A`) | `cfg.TaxCode.TaxType` |
| `A003` / `KONP` | `KBETR` | `CURR/DEC` | Condition rate | `cfg.TaxCodeRate.RatePercent` |
| `T052` | `ZTERM` | `CHAR(4)` | Payment terms | `cfg.PaymentTerms.PaymentTerms` |
| `T052` | `ZTAG1` / `ZPRZ1` | `NUMC(2)` / `DEC(5,3)` | Discount days / percent | `cfg.PaymentTerms.CashDiscount1Days` / `.CashDiscount1Percent` |
| `T042Z` | `ZLSCH` | `CHAR(1)` | Payment method | `cfg.PaymentMethod.PaymentMethod` |
| `T047A` | `MAHNA` | `CHAR(4)` | Dunning procedure | `cfg.DunningProcedure.DunningProcedure` |
| `T043T` | `TOGRU` | `CHAR(4)` | Tolerance group | `cfg.ToleranceGroup.ToleranceGroup` |
| `T030` | `KTOSL` | `CHAR(3)` | Transaction key | `cfg.AccountDeterminationRule.TransactionKey` |
| `T011` | `VERSN` | `CHAR(4)` | Financial statement version | `cfg.FinancialStatementVersion.FinancialStatementVersion` |
| `T074U` | `UMSKZ` | `CHAR(1)` | Special G/L indicator | `cfg.SpecialGLIndicator.SpecialGLIndicator` |
| `T881` | `RLDNR` | `CHAR(2)` | Ledger | `cfg.Ledger.Ledger` |

---

## 10.3 Business Partner, customer, vendor

| SAP table | SAP field | ABAP type | Meaning | This system |
|---|---|---|---|---|
| `BUT000` | `PARTNER` | `CHAR(10)` | Business partner number | `mdm.BusinessPartner.PartnerNumber` |
| `BUT000` | `PARTNER_GUID` | `RAW(16)` | BP GUID | `mdm.BusinessPartner.Id` (surrogate) |
| `BUT000` | `TYPE` | `CHAR(1)` | BP category (1/2/3) | `mdm.BusinessPartner.PartnerCategory` |
| `BUT000` | `BU_GROUP` | `CHAR(4)` | BP grouping | `mdm.BusinessPartner.BusinessPartnerGroupId` |
| `BUT000` | `NAME_ORG1` … `NAME_ORG4` | `CHAR(40)` | Organisation name lines | `mdm.BusinessPartner.Name1` … `.Name4` |
| `BUT000` | `NAME_FIRST` / `NAME_LAST` | `CHAR(40)` | Person name | `mdm.BusinessPartner.FirstName` / `.LastName` |
| `BUT000` | `BU_SORT1` / `BU_SORT2` | `CHAR(20)` | Search terms | `mdm.BusinessPartner.SearchTerm1` / `.SearchTerm2` |
| `BUT000` | `BIRTHDT` | `DATS(8)` | Date of birth | `mdm.BusinessPartner.DateOfBirth` |
| `BUT000` | `XBLCK` | `CHAR(1)` | Central block | `mdm.BusinessPartner.IsCentralBlocked` |
| `BUT000` | `XDELE` | `CHAR(1)` | Central deletion flag | `mdm.BusinessPartner.IsMarkedForDeletion` |
| `BUT100` | `RLTYP` | `CHAR(6)` | BP role | `mdm.BusinessPartnerRoleAssignment.BusinessPartnerRoleId` |
| `BUT100` | `VALID_FROM` / `VALID_TO` | `DEC(15)` | Role validity (timestamp) | `mdm.BusinessPartnerRoleAssignment.ValidFrom` / `.ValidTo` |
| `BUT020` | `ADDRNUMBER` | `CHAR(10)` | Address number | `mdm.BusinessPartnerAddress.AddressId` |
| `BUT050` | `RELNR` | `CHAR(12)` | Relationship number | `mdm.BusinessPartnerRelationship.RelationshipNumber` |
| `BUT050` | `PARTNER1` / `PARTNER2` | `CHAR(10)` | Related partners | `.SourceBusinessPartnerId` / `.TargetBusinessPartnerId` |
| `BUT050` | `RELTYP` | `CHAR(6)` | Relationship category | `mdm.BusinessPartnerRelationship.RelationshipCategory` |
| `BUT0ID` | `IDNUMBER` | `CHAR(60)` | Identification number | `mdm.BusinessPartnerIdentification.IdentificationNumber` |
| `BUT0BK` | `BANKN` | `CHAR(18)` | Bank account number | `mdm.BusinessPartnerBank.BankAccountNumber` |
| `BUT0BK` | `IBAN` | `CHAR(34)` | IBAN | `mdm.BusinessPartnerBank.Iban` |
| `ADRC` | `ADDRNUMBER` | `CHAR(10)` | Address number | `mdm.Address.AddressNumber` |
| `ADRC` | `STREET` / `HOUSE_NUM1` | `CHAR(60)` / `CHAR(10)` | Street / house number | `mdm.Address.Street` / `.HouseNumber` |
| `ADRC` | `CITY1` / `POST_CODE1` | `CHAR(40)` / `CHAR(10)` | City / postal code | `mdm.Address.City` / `.PostalCode` |
| `ADR6` | `SMTP_ADDR` | `CHAR(241)` | Email address | `mdm.BusinessPartnerCommunication.Value` (type `EMAIL`) |
| `KNA1` | `KUNNR` | `CHAR(10)` | Customer number | `mdm.BusinessPartnerCustomer.CustomerNumber` |
| `KNA1` | `KTOKD` | `CHAR(4)` | Customer account group | `mdm.BusinessPartnerCustomer.CustomerAccountGroupId` |
| `KNA1` | `STCEG` | `CHAR(20)` | VAT registration number | `mdm.BusinessPartnerCustomer.VatRegistrationNumber` |
| `KNB1` | `AKONT` | `CHAR(10)` | Reconciliation account | `mdm.BusinessPartnerCompanyCode.ReconciliationGLAccountId` |
| `KNB1` | `ZTERM` | `CHAR(4)` | Payment terms | `mdm.BusinessPartnerCompanyCode.PaymentTermsId` |
| `KNB1` | `MAHNA` | `CHAR(4)` | Dunning procedure | `mdm.BusinessPartnerCompanyCode.DunningProcedureId` |
| `KNB1` | `ZUAWA` | `CHAR(3)` | Sort key | `mdm.BusinessPartnerCompanyCode.SortKey` |
| `KNVV` | `VKORG` / `VTWEG` / `SPART` | `CHAR(4)/(2)/(2)` | Sales area | `mdm.BusinessPartnerSalesArea.SalesAreaId` |
| `KNVV` | `INCO1` / `INCO2` | `CHAR(3)` / `CHAR(28)` | Incoterms | `mdm.BusinessPartnerSalesArea.Incoterms1` / `.Incoterms2` |
| `LFA1` | `LIFNR` | `CHAR(10)` | Vendor number | `mdm.BusinessPartnerVendor.VendorNumber` |
| `LFA1` | `KTOKK` | `CHAR(4)` | Vendor account group | `mdm.BusinessPartnerVendor.VendorAccountGroupId` |
| `LFB1` | `AKONT` | `CHAR(10)` | Reconciliation account | `mdm.BusinessPartnerCompanyCode.ReconciliationGLAccountId` |
| `LFB1` | `ZWELS` | `CHAR(10)` | Payment methods | `mdm.BusinessPartnerCompanyCode.PaymentMethods` |
| `LFB1` | `ZAHLS` | `CHAR(1)` | Payment block | `mdm.BusinessPartnerCompanyCode.PaymentBlockReason` |
| `LFM1` | `EKORG` | `CHAR(4)` | Purchasing organisation | `mdm.BusinessPartnerPurchasingOrganization.PurchasingOrganizationId` |
| `LFM1` | `WAERS` | `CUKY(5)` | Order currency | `mdm.BusinessPartnerPurchasingOrganization.OrderCurrencyCode` |
| `LFM1` | `WEBRE` | `CHAR(1)` | GR-based invoice verification | `.IsGoodsReceiptBasedInvoiceVerification` |
| `UKMBP_CMS_SGM` | `CREDIT_LIMIT` | `CURR` | Credit limit | `mdm.BusinessPartnerCreditProfile.CreditLimit` |

---

## 10.4 G/L master and universal journal

| SAP table | SAP field | ABAP type | Meaning | This system |
|---|---|---|---|---|
| `SKA1` | `SAKNR` | `CHAR(10)` | G/L account number | `mdm.GLAccount.GLAccount` |
| `SKA1` | `KTOPL` | `CHAR(4)` | Chart of accounts | `mdm.GLAccount.ChartOfAccountsId` |
| `SKA1` | `XBILK` | `CHAR(1)` | Balance sheet account | `mdm.GLAccount.IsBalanceSheetAccount` |
| `SKA1` | `KTOKS` | `CHAR(4)` | Account group | `mdm.GLAccount.AccountGroupId` |
| `SKA1` | `GVTYP` | `CHAR(2)` | P&L statement account type | `mdm.GLAccount.RetainedEarningsAccountKey` |
| `SKAT` | `TXT20` / `TXT50` | `CHAR(20)` / `CHAR(50)` | Account texts | `mdm.GLAccountText.ShortText` / `.LongText` |
| `SKB1` | `MITKZ` | `CHAR(1)` | Reconciliation account type | `mdm.GLAccountCompanyCode.IsReconciliationAccountForAccountType` |
| `SKB1` | `XOPVW` | `CHAR(1)` | Open item management | `mdm.GLAccountCompanyCode.IsOpenItemManaged` |
| `SKB1` | `XKRES` | `CHAR(1)` | Line item display | `mdm.GLAccountCompanyCode.IsLineItemDisplay` |
| `SKB1` | `FSTAG` | `CHAR(4)` | Field status group | `mdm.GLAccountCompanyCode.FieldStatusGroupId` |
| `SKB1` | `ZUAWA` | `CHAR(3)` | Sort key | `mdm.GLAccountCompanyCode.SortKey` |
| `SKB1` | `HBKID` / `HKTID` | `CHAR(5)` | House bank / account id | `.HouseBankId` / `.HouseBankAccountId` |
| `BKPF` | `BELNR` | `CHAR(10)` | Document number | `fin.JournalEntryHeader.DocumentNumber` |
| `BKPF` | `GJAHR` | `NUMC(4)` | Fiscal year | `fin.JournalEntryHeader.FiscalYear` |
| `BKPF` | `BLART` | `CHAR(2)` | Document type | `fin.JournalEntryHeader.DocumentTypeId` |
| `BKPF` | `BLDAT` | `DATS(8)` | Document date | `fin.JournalEntryHeader.DocumentDate` |
| `BKPF` | `BUDAT` | `DATS(8)` | Posting date | `fin.JournalEntryHeader.PostingDate` |
| `BKPF` | `MONAT` | `NUMC(2)` | Posting period | `fin.JournalEntryHeader.FiscalPeriod` |
| `BKPF` | `CPUDT` / `CPUTM` | `DATS` / `TIMS` | Entry date / time | `.EntryDate` / `.EntryTime` |
| `BKPF` | `USNAM` | `CHAR(12)` | Entered by | `fin.JournalEntryHeader.CreatedBy` |
| `BKPF` | `TCODE` | `CHAR(20)` | Transaction code | `fin.JournalEntryHeader.TransactionCode` |
| `BKPF` | `XBLNR` | `CHAR(16)` | Reference document | `fin.JournalEntryHeader.ReferenceDocumentNumber` |
| `BKPF` | `BKTXT` | `CHAR(25)` | Document header text | `fin.JournalEntryHeader.DocumentHeaderText` |
| `BKPF` | `WAERS` | `CUKY(5)` | Document currency | `fin.JournalEntryHeader.DocumentCurrencyCode` |
| `BKPF` | `KURSF` | `DEC(9,5)` | Exchange rate | `fin.JournalEntryHeader.ExchangeRate` |
| `BKPF` | `STBLG` / `STJAH` | `CHAR(10)` / `NUMC(4)` | Reversal document / year | `.ReversalDocumentNumber` |
| `BKPF` | `BSTAT` | `CHAR(1)` | Document status (parked etc.) | `fin.JournalEntryHeader.Status` |
| `BKPF` | `AWTYP` / `AWKEY` | `CHAR(5)` / `CHAR(20)` | Reference procedure / key | `.SourceDocumentType` / `.SourceDocumentId` |
| `ACDOCA` | `RLDNR` | `CHAR(2)` | Ledger | `fin.JournalEntryLine.LedgerId` |
| `ACDOCA` | `RBUKRS` | `CHAR(4)` | Company code | `fin.JournalEntryLine.CompanyCodeId` |
| `ACDOCA` | `BELNR` / `DOCLN` | `CHAR(10)` / `CHAR(6)` | Document / line | `.DocumentNumber` / `.LineItemNumber` |
| `ACDOCA` | `RACCT` | `CHAR(10)` | G/L account | `fin.JournalEntryLine.GLAccount` |
| `ACDOCA` | `DRCRK` | `CHAR(1)` | Debit/credit indicator | `fin.JournalEntryLine.DebitCreditIndicator` |
| `ACDOCA` | `HSL` | `CURR(23,2)` | Amount in local currency | `fin.JournalEntryLine.AmountInLocalCurrency` |
| `ACDOCA` | `WSL` | `CURR(23,2)` | Amount in transaction currency | `.AmountInDocumentCurrency` |
| `ACDOCA` | `KSL` | `CURR(23,2)` | Amount in group currency | `.AmountInGroupCurrency` |
| `ACDOCA` | `OSL` | `CURR(23,2)` | Amount in CO area currency | `.AmountInControllingAreaCurrency` |
| `ACDOCA` | `MSL` | `QUAN(23,3)` | Quantity | `fin.JournalEntryLine.Quantity` |
| `ACDOCA` | `RHCUR` / `RWCUR` / `RKCUR` | `CUKY(5)` | Local / transaction / group currency | `.LocalCurrencyCode` / `.DocumentCurrencyCode` / `.GroupCurrencyCode` |
| `ACDOCA` | `RCNTR` | `CHAR(10)` | Cost centre | `fin.JournalEntryLine.CostCenterId` |
| `ACDOCA` | `PRCTR` | `CHAR(10)` | Profit centre | `fin.JournalEntryLine.ProfitCenterId` |
| `ACDOCA` | `PPRCTR` | `CHAR(10)` | Partner profit centre | `.PartnerProfitCenterId` |
| `ACDOCA` | `SEGMENT` / `PSEGMENT` | `CHAR(10)` | Segment / partner segment | `.SegmentId` / `.PartnerSegmentId` |
| `ACDOCA` | `RFAREA` | `CHAR(16)` | Functional area | `.FunctionalAreaId` |
| `ACDOCA` | `RBUSA` | `CHAR(4)` | Business area | `.BusinessAreaId` |
| `ACDOCA` | `AUFNR` | `CHAR(12)` | Internal order | `.InternalOrderId` |
| `ACDOCA` | `ANLN1` / `ANLN2` | `CHAR(12)` / `CHAR(4)` | Asset / sub-number | `.AssetId` / `.AssetSubNumber` |
| `ACDOCA` | `KUNNR` / `LIFNR` | `CHAR(10)` | Customer / vendor | `.BusinessPartnerId` + `.BusinessPartnerRoleCategory` |
| `ACDOCA` | `VBUND` | `CHAR(6)` | Trading partner | `.TradingPartnerCompany` |
| `ACDOCA` | `AUGDT` / `AUGBL` | `DATS` / `CHAR(10)` | Clearing date / document | `.ClearingDate` / `.ClearingDocumentNumber` |
| `ACDOCA` | `ZUONR` | `CHAR(18)` | Assignment | `fin.JournalEntryLine.AssignmentReference` |
| `ACDOCA` | `SGTXT` | `CHAR(50)` | Item text | `fin.JournalEntryLine.LineItemText` |
| `BSEG` | `BSCHL` | `CHAR(2)` | Posting key | `fin.JournalEntryLine.PostingKey` |
| `BSEG` | `KOART` | `CHAR(1)` | Account type | `fin.JournalEntryLine.AccountType` |
| `BSEG` | `UMSKZ` | `CHAR(1)` | Special G/L indicator | `fin.JournalEntryLine.SpecialGLIndicator` |
| `BSEG` | `ZFBDT` | `DATS(8)` | Baseline date | `fin.JournalEntryLine.BaselineDate` |
| `BSEG` | `ZTERM` | `CHAR(4)` | Payment terms | `fin.JournalEntryLine.PaymentTermsId` |
| `BSEG` | `MWSKZ` | `CHAR(2)` | Tax code | `fin.JournalEntryLine.TaxCodeId` |
| `BSET` | `KTOSL` | `CHAR(3)` | Tax account key | `fin.JournalEntryTax.TaxGLAccountId` (via determination) |
| `BSET` | `HWBAS` / `FWBAS` | `CURR` | Tax base local / document | `.TaxBaseAmountInLocalCurrency` / `.TaxBaseAmountInDocumentCurrency` |
| `BSET` | `HWSTE` / `FWSTE` | `CURR` | Tax amount local / document | `.TaxAmountInLocalCurrency` / `.TaxAmountInDocumentCurrency` |
| `BSID` / `BSIK` / `BSIS` | — | — | Open items customer / vendor / G/L | `fin.OpenItem` (one table, `AccountType` distinguishes) |
| `BSAD` / `BSAK` / `BSAS` | — | — | Cleared items | `fin.OpenItem` with `Status = 'Cleared'` |
| `GLT0` / `FAGLFLEXT` | `HSL01`…`HSL16` | `CURR` | Period totals | `fin.AccountBalance` (one row per period) |
| `REGUH` | `LAUFD` / `LAUFI` | `DATS` / `CHAR(6)` | Payment run date / id | `fin.PaymentRun.RunDate` / `.RunIdentifier` |
| `REGUP` | `BELNR` | `CHAR(10)` | Paid document | `fin.PaymentAllocation.OpenItemId` |
| `RBKP` | `BELNR` / `GJAHR` | `CHAR(10)` / `NUMC(4)` | Invoice document | `fin.VendorInvoice.InvoiceNumber` |
| `RSEG` | `BUZEI` | `NUMC(3)` | Invoice item | `fin.VendorInvoiceItem.ItemNumber` |
| `BKDF` | — | — | Recurring entry data | `fin.RecurringEntry` |
| `MHNK` / `MHND` | `MAHSK` / `MANSP` | `CHAR(1)` | Dunning level / block | `fin.DunningNotice.DunningLevel` |

---

## 10.5 Asset accounting

| SAP table | SAP field | ABAP type | Meaning | This system |
|---|---|---|---|---|
| `ANLA` | `ANLN1` / `ANLN2` | `CHAR(12)` / `CHAR(4)` | Asset / sub-number | `fin.Asset.AssetNumber` / `.AssetSubNumber` |
| `ANLA` | `ANLKL` | `CHAR(8)` | Asset class | `fin.Asset.AssetClassId` |
| `ANLA` | `TXT50` / `TXA50` | `CHAR(50)` | Description | `fin.Asset.Description` / `.Description2` |
| `ANLA` | `AKTIV` | `DATS(8)` | Capitalisation date | `fin.Asset.CapitalizationDate` |
| `ANLA` | `DEAKT` | `DATS(8)` | Deactivation date | `fin.Asset.DeactivationDate` |
| `ANLA` | `SERNR` / `INVNR` | `CHAR(18)` / `CHAR(25)` | Serial / inventory number | `.SerialNumber` / `.InventoryNumber` |
| `ANLA` | `MENGE` / `MEINS` | `QUAN` / `UNIT(3)` | Quantity / unit | `.Quantity` / `.UnitOfMeasure` |
| `ANLZ` | `KOSTL` | `CHAR(10)` | Cost centre (time-dependent) | `fin.AssetTimeDependent.CostCenterId` |
| `ANLZ` | `ADATU` / `BDATU` | `DATS(8)` | Validity from / to | `.ValidFrom` / `.ValidTo` |
| `ANLB` | `AFABE` | `NUMC(2)` | Depreciation area | `fin.AssetDepreciationArea.DepreciationAreaId` |
| `ANLB` | `AFASL` | `CHAR(4)` | Depreciation key | `.DepreciationKeyId` |
| `ANLB` | `NDJAR` / `NDPER` | `NUMC(3)` / `NUMC(3)` | Useful life years / periods | `.UsefulLifeYears` / `.UsefulLifePeriods` |
| `ANLB` | `AFABG` | `DATS(8)` | Depreciation start date | `.DepreciationStartDate` |
| `ANLB` | `SCHRW` | `CURR` | Scrap value | `.ScrapValue` |
| `ANLC` | `KANSW` | `CURR` | Cumulative acquisition value | `fin.AssetValue.AcquisitionValueBroughtForward` |
| `ANLC` | `KNAFA` | `CURR` | Accumulated ordinary depreciation | `.AccumulatedDepreciationBroughtForward` |
| `ANLC` | `ANSWL` | `CURR` | Acquisitions in the year | `.CurrentYearAcquisitions` |
| `ANLC` | `NAFAG` | `CURR` | Ordinary depreciation posted | `.OrdinaryDepreciationPosted` |
| `ANEP` | `BWASL` | `CHAR(3)` | Transaction type | `fin.AssetTransaction.TransactionTypeId` |
| `ANEP` | `BZDAT` | `DATS(8)` | Asset value date | `fin.AssetTransaction.AssetValueDate` |
| `ANEP` | `ANBTR` | `CURR` | Transaction amount | `fin.AssetTransaction.TransactionAmount` |
| `ANLP` | `NAFAZ` / `SAFAZ` / `AAFAZ` | `CURR` | Ordinary / special / unplanned depreciation | `fin.DepreciationPosting.OrdinaryDepreciationAmount` … |
| `ANKA` | `ANLKL` | `CHAR(8)` | Asset class | `fin.AssetClass.AssetClass` |
| `ANKA` | `KTOGR` | `CHAR(8)` | Account determination | `fin.AssetClass.AccountDeterminationKey` |
| `ANKB` | `AFASL` | `CHAR(4)` | Default depreciation key | `fin.AssetClassDepreciationArea.DepreciationKeyId` |
| `T093` | `AFABE` | `NUMC(2)` | Depreciation area | `fin.DepreciationArea.DepreciationArea` |
| `T090NA` | `AFASL` | `CHAR(4)` | Depreciation key | `fin.DepreciationKey.DepreciationKey` |
| `TABW` | `BWASL` | `CHAR(3)` | Asset transaction type | `fin.AssetTransactionType.TransactionType` |

---

## 10.6 Controlling

| SAP table | SAP field | ABAP type | Meaning | This system |
|---|---|---|---|---|
| `CSKS` | `KOSTL` | `CHAR(10)` | Cost centre | `co.CostCenter.CostCenter` |
| `CSKS` | `KOKRS` | `CHAR(4)` | Controlling area | `co.CostCenter.ControllingAreaId` |
| `CSKS` | `DATAB` / `DATBI` | `DATS(8)` | Valid from / to | `co.CostCenter.ValidFrom` / `.ValidTo` |
| `CSKS` | `KOSAR` | `CHAR(1)` | Cost centre category | `co.CostCenter.CostCenterCategoryId` |
| `CSKS` | `VERAK` | `CHAR(20)` | Person responsible | `co.CostCenter.ResponsiblePersonPartnerId` |
| `CSKS` | `PRCTR` | `CHAR(10)` | Profit centre | `co.CostCenter.ProfitCenterId` |
| `CSKS` | `KHINR` | `CHAR(12)` | Hierarchy area | `co.CostCenter.HierarchyNodeId` |
| `CSKT` | `KTEXT` / `LTEXT` | `CHAR(20)` / `CHAR(40)` | Cost centre texts | `co.CostCenter.Name` / `.Description` |
| `CSKB` | `KSTAR` | `CHAR(10)` | Cost element | `co.CostElement.CostElement` |
| `CSKB` | `KATYP` | `NUMC(2)` | Cost element category | `co.CostElement.CostElementCategory` |
| `CSLA` | `LSTAR` | `CHAR(6)` | Activity type | `co.ActivityType.ActivityType` |
| `CSLA` | `LEINH` | `UNIT(3)` | Activity unit | `co.ActivityType.UnitOfMeasure` |
| `CEPC` | `PRCTR` | `CHAR(10)` | Profit centre | `co.ProfitCenter.ProfitCenter` |
| `CEPC` | `SEGMENT` | `CHAR(10)` | Segment | `co.ProfitCenter.SegmentId` |
| `AUFK` | `AUFNR` | `CHAR(12)` | Order number | `co.InternalOrder.OrderNumber` |
| `AUFK` | `AUART` | `CHAR(4)` | Order type | `co.InternalOrder.OrderTypeId` |
| `AUFK` | `KTEXT` | `CHAR(40)` | Description | `co.InternalOrder.Description` |
| `AUFK` | `KOSTV` | `CHAR(10)` | Responsible cost centre | `co.InternalOrder.ResponsibleCostCenterId` |
| `AUFK` | `ASTKZ` | `CHAR(1)` | Statistical order | `co.InternalOrder.IsStatistical` |
| `COEP` | `OBJNR` | `CHAR(22)` | Object number | `co.ControllingPosting.ObjectType` + `.ObjectId` |
| `COEP` | `WRTTP` | `CHAR(2)` | Value type | `co.ControllingPosting.ValueType` |
| `COEP` | `VERSN` | `CHAR(3)` | Version | `co.ControllingPosting.PlanVersion` |
| `COEP` | `WKGBTR` | `CURR` | Value in CO area currency | `.AmountInControllingAreaCurrency` |
| `COEP` | `WTGBTR` | `CURR` | Value in transaction currency | `.AmountInTransactionCurrency` |
| `COEP` | `MEGBTR` / `MEINH` | `QUAN` / `UNIT(3)` | Quantity / unit | `.Quantity` / `.UnitOfMeasure` |
| `COSP` / `COSS` | `WKG001`…`WKG016` | `CURR` | Period totals | `co.ControllingTotal` (one row per period) |
| `COBRB` | `PERBZ` / `PROZS` | `CHAR(3)` / `DEC(5,2)` | Settlement type / percentage | `co.SettlementRule.SettlementType` / `.Percentage` |
| `COOI` | — | — | Commitment line items | `co.Commitment` |
| `T811C` / `T811S` | `KRSNAM` / `SEGNAME` | `CHAR(6)` | Cycle / segment | `co.AllocationCycle.CycleCode` / `co.AllocationCycleSegment.SegmentCode` |
| `TKA03` | `STAGR` | `CHAR(6)` | Statistical key figure | `co.StatisticalKeyFigure.StatisticalKeyFigure` |
| `BPJA` | `WTJHR` | `CURR` | Annual budget value | `co.InternalOrderBudget.CurrentBudget` |

---

## 10.7 Security, audit, dictionary

| SAP table | SAP field | ABAP type | Meaning | This system |
|---|---|---|---|---|
| `USR02` | `BNAME` | `CHAR(12)` | User name | `sec.User.UserName` |
| `USR02` | `USTYP` | `CHAR(1)` | User type | `sec.User.UserType` |
| `USR02` | `GLTGV` / `GLTGB` | `DATS(8)` | Valid from / to | `sec.User.ValidFrom` / `.ValidTo` |
| `USR02` | `UFLAG` | `RAW(1)` | Lock indicator | `sec.User.IsLocked` |
| `USR02` | `TRDAT` / `LTIME` | `DATS` / `TIMS` | Last logon date / time | `sec.User.LastLoginAt` |
| `AGR_DEFINE` | `AGR_NAME` | `CHAR(30)` | Role name | `sec.Role.RoleCode` |
| `AGR_USERS` | `FROM_DAT` / `TO_DAT` | `DATS(8)` | Role assignment validity | `sec.UserRole.ValidFrom` / `.ValidTo` |
| `TOBJ` | `OBJCT` | `CHAR(10)` | Authorization object | `sec.AuthorizationObject.ObjectCode` |
| `TACT` | `ACTVT` | `CHAR(2)` | Activity (01 create, 02 change, 03 display) | `sec.Permission.Action` |
| `TSTC` | `TCODE` | `CHAR(20)` | Transaction code | `sec.TransactionCode.TransactionCode` |
| `TBRG` | `BRGRU` | `CHAR(4)` | Authorization group | `cfg.TableAuthorizationGroup.AuthorizationGroup` |
| `CDHDR` | `OBJECTCLAS` | `CHAR(15)` | Object class | `audit.ChangeDocumentHeader.ObjectClass` |
| `CDHDR` | `OBJECTID` | `CHAR(90)` | Object key | `audit.ChangeDocumentHeader.ObjectKeyText` |
| `CDHDR` | `CHANGENR` | `CHAR(10)` | Change number | `audit.ChangeDocumentHeader.ChangeDocumentNumber` |
| `CDHDR` | `UDATE` / `UTIME` | `DATS` / `TIMS` | Change date / time | `audit.ChangeDocumentHeader.ChangedAt` |
| `CDPOS` | `TABNAME` / `FNAME` | `CHAR(30)` | Table / field changed | `audit.ChangeDocumentItem.TableName` / `.FieldName` |
| `CDPOS` | `VALUE_OLD` / `VALUE_NEW` | `CHAR(254)` | Old / new value | `audit.ChangeDocumentItem.OldValue` / `.NewValue` |
| `DD02L` | `TABNAME` | `CHAR(30)` | Table definition | `cfg.DictionaryTable.TableName` |
| `DD03L` | `FIELDNAME` | `CHAR(30)` | Table field | `cfg.DictionaryTableField.FieldName` |
| `DD04L` | `ROLLNAME` | `CHAR(30)` | Data element | `cfg.DictionaryDataElement.DataElementName` |
| `DD01L` | `DOMNAME` | `CHAR(30)` | Domain | `cfg.DictionaryDomain.DomainName` |
| `DD07L` | `DOMVALUE_L` | `CHAR(10)` | Fixed value | `cfg.DictionaryDomainValue.LowValue` |

---

## 10.8 Type mapping rules

| ABAP type | SQL Server type used here | Note |
|---|---|---|
| `CLNT(3)` | `int` | Tenant is a surrogate integer, not a 3-character client |
| `CHAR(n)` | `nvarchar(n)` | Unicode throughout; trailing blanks are not stored |
| `NUMC(n)` | `smallint` / `tinyint` / `nvarchar(n)` | Numeric when it is a number (year, period), text when it is a code |
| `DATS(8)` | `date` | `9999-12-31` keeps its "open ended" meaning |
| `TIMS(6)` | `time(0)` | |
| `CURR(x,2)` | `decimal(19,4)` | Four decimals cover currencies with 3-decimal minor units |
| `QUAN(x,3)` | `decimal(23,6)` | Quantities and rates |
| `DEC(9,5)` (rate) | `decimal(23,6)` | Exchange rates |
| `CUKY(5)` | `nvarchar(5)` | Currency key |
| `UNIT(3)` | `nvarchar(3)` | Unit of measure |
| `LANG(1)` | `nvarchar(2)` | Two-character language code (`EN`, `KM`) |
| `RAW(16)` (GUID) | `uniqueidentifier` | Correlation and idempotency keys |
| `CHAR(1)` flag `'X'`/`' '` | `bit` | Never store `'X'`; use a real boolean |
| inverted date `GDATU` | `date` | Stored as a normal date; sorting is handled by the index |
