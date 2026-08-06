/* ============================================================================
   S/4HANA-inspired ERP - sample data (design prompt, section 24)

   Two companies, three company codes, one shared chart of accounts, one
   controlling area, USD group currency, USD/KHR/THB transaction currencies,
   customer / vendor / dual-role business partners, cost centres, profit
   centres, an internal order, an asset, an intercompany posting and a
   foreign-currency posting.

   Conventions used by the postings
     * Amounts are signed: debit positive, credit negative. Every document
       therefore sums to zero per currency, which is what the balance rebuild
       at the end of this script relies on.
     * Open items and account balances are derived from the journal, never
       entered by hand - the same rule the posting engine follows.

   Runs as one transaction and is skipped if the sample data is already there.
   ============================================================================ */

USE [ErpS4];
GO

SET NOCOUNT ON;
SET XACT_ABORT ON;
GO

IF EXISTS (SELECT 1 FROM [org].[Company] WHERE TenantId = 1 AND CompanyCodeGroup = N'1000')
BEGIN
    PRINT 'Sample data is already present - nothing to do.';
    RETURN;
END;

BEGIN TRY
BEGIN TRANSACTION;

DECLARE @Tenant int = 1;
DECLARE @By nvarchar(64) = N'SYSTEM';
DECLARE @Open date = '2000-01-01';
DECLARE @Never date = '9999-12-31';

/* ---------------------------------------------------------------------------
   0. Tenant
   --------------------------------------------------------------------------- */
IF NOT EXISTS (SELECT 1 FROM [org].[Tenant] WHERE Id = @Tenant)
BEGIN
    SET IDENTITY_INSERT [org].[Tenant] ON;
    INSERT INTO [org].[Tenant]
        (Id, TenantCode, Name, DefaultLanguage, TimeZoneId, IsProduction,
         AllowCustomizingChanges, ValidFrom, ValidTo, CreatedBy)
    VALUES
        (1, N'100', N'Default tenant', N'EN', N'Asia/Phnom_Penh', 0, 1,
         @Open, @Never, @By);
    SET IDENTITY_INSERT [org].[Tenant] OFF;
END;

/* ---------------------------------------------------------------------------
   1. Language, currency, country, unit of measure
   --------------------------------------------------------------------------- */
INSERT INTO [cfg].[Language] (LanguageCode, IsoCode, Name, IsRightToLeft, IsActive, CreatedBy)
VALUES (N'EN', N'en-US', N'English', 0, 1, @By),
       (N'KM', N'km-KH', N'Khmer',   0, 1, @By);

INSERT INTO [cfg].[Currency]
    (TenantId, CurrencyCode, IsoCode, NumericCode, Name, ShortText, DecimalPlaces, Symbol, CreatedBy)
VALUES (@Tenant, N'USD', N'USD', N'840', N'US Dollar',      N'USD', 2, N'$',  @By),
       (@Tenant, N'KHR', N'KHR', N'116', N'Cambodian Riel', N'Riel', 0, N'KHR', @By),
       (@Tenant, N'THB', N'THB', N'764', N'Thai Baht',      N'Baht', 2, N'THB', @By);

INSERT INTO [cfg].[CurrencyDecimal] (TenantId, CurrencyCode, DecimalPlaces, CreatedBy)
VALUES (@Tenant, N'KHR', 0, @By);

INSERT INTO [cfg].[Country]
    (TenantId, CountryCode, IsoCode2, IsoCode3, Name, CurrencyCode, LanguageCode,
     IsEuMember, DateFormat, DecimalSeparator, CreatedBy)
VALUES (@Tenant, N'KH', N'KH', N'KHM', N'Cambodia',      N'KHR', N'KM', 0, N'dd/MM/yyyy', N'.', @By),
       (@Tenant, N'TH', N'TH', N'THA', N'Thailand',      N'THB', N'EN', 0, N'dd/MM/yyyy', N'.', @By),
       (@Tenant, N'US', N'US', N'USA', N'United States', N'USD', N'EN', 0, N'MM/dd/yyyy', N'.', @By);

INSERT INTO [cfg].[Region] (TenantId, CountryCode, RegionCode, Name, CreatedBy)
VALUES (@Tenant, N'KH', N'PP', N'Phnom Penh',   @By),
       (@Tenant, N'KH', N'KS', N'Kampong Speu', @By),
       (@Tenant, N'TH', N'BK', N'Bangkok',      @By);

INSERT INTO [cfg].[UnitOfMeasure]
    (TenantId, UnitOfMeasure, IsoCode, Name, Dimension, DecimalPlaces, CreatedBy)
VALUES (@Tenant, N'EA', N'EA', N'Each',      NULL,     0, @By),
       (@Tenant, N'KG', N'KGM', N'Kilogram', N'MASS',  3, @By),
       (@Tenant, N'HR', N'HUR', N'Hour',     N'TIME',  2, @By);

INSERT INTO [cfg].[ExchangeRateType]
    (TenantId, ExchangeRateType, Name, QuotationType, IsInversionAllowed, CreatedBy)
VALUES (@Tenant, N'M', N'Average rate (posting)', N'Direct', 1, @By);

DECLARE @RateType bigint =
    (SELECT Id FROM [cfg].[ExchangeRateType] WHERE TenantId = @Tenant AND ExchangeRateType = N'M');

INSERT INTO [cfg].[ExchangeRate]
    (TenantId, ExchangeRateTypeId, FromCurrencyCode, ToCurrencyCode, ValidFrom,
     Rate, FromRatio, ToRatio, Source, CreatedBy)
VALUES (@Tenant, @RateType, N'USD', N'KHR', '2026-01-01', 4100.000000, 1, 1, N'Manual', @By),
       (@Tenant, @RateType, N'KHR', N'USD', '2026-01-01',    0.000244, 1, 1, N'Manual', @By),
       (@Tenant, @RateType, N'USD', N'THB', '2026-01-01',   35.000000, 1, 1, N'Manual', @By),
       (@Tenant, @RateType, N'THB', N'USD', '2026-01-01',    0.028571, 1, 1, N'Manual', @By);

/* ---------------------------------------------------------------------------
   2. Fiscal calendar, period control, field status
   --------------------------------------------------------------------------- */
INSERT INTO [cfg].[AccountingPrinciple] (TenantId, AccountingPrinciple, Name, CreatedBy)
VALUES (@Tenant, N'IFRS', N'International Financial Reporting Standards', @By);

DECLARE @Principle bigint =
    (SELECT Id FROM [cfg].[AccountingPrinciple] WHERE TenantId = @Tenant AND AccountingPrinciple = N'IFRS');

INSERT INTO [cfg].[Ledger]
    (TenantId, Ledger, Name, IsLeading, AccountingPrincipleId, LedgerType, CreatedBy)
VALUES (@Tenant, N'0L', N'Leading ledger (IFRS)', 1, @Principle, N'Standard', @By);

DECLARE @Ledger bigint =
    (SELECT Id FROM [cfg].[Ledger] WHERE TenantId = @Tenant AND Ledger = N'0L');

INSERT INTO [cfg].[FiscalYearVariant]
    (TenantId, FiscalYearVariant, Name, NumberOfPostingPeriods, NumberOfSpecialPeriods,
     IsCalendarYear, IsYearDependent, CreatedBy)
VALUES (@Tenant, N'K4', N'Calendar year, 12 periods + 4 special', 12, 4, 1, 0, @By);

DECLARE @Fyv bigint =
    (SELECT Id FROM [cfg].[FiscalYearVariant] WHERE TenantId = @Tenant AND FiscalYearVariant = N'K4');

DECLARE @Period tinyint = 1;
WHILE @Period <= 12
BEGIN
    INSERT INTO [cfg].[FiscalPeriod]
        (TenantId, FiscalYearVariantId, FiscalYear, FiscalPeriod,
         PeriodStartDate, PeriodEndDate, IsSpecialPeriod, PeriodStatus, CreatedBy)
    VALUES
        (@Tenant, @Fyv, 2026, @Period,
         DATEFROMPARTS(2026, @Period, 1), EOMONTH(DATEFROMPARTS(2026, @Period, 1)),
         0, N'Open', @By);
    SET @Period = @Period + 1;
END;

INSERT INTO [cfg].[PostingPeriodVariant] (TenantId, PostingPeriodVariant, Name, CreatedBy)
VALUES (@Tenant, N'KH00', N'Cambodia group posting periods', @By);

DECLARE @Ppv bigint =
    (SELECT Id FROM [cfg].[PostingPeriodVariant] WHERE TenantId = @Tenant AND PostingPeriodVariant = N'KH00');

INSERT INTO [cfg].[PostingPeriodControl]
    (TenantId, PostingPeriodVariantId, AccountType, FromAccount, ToAccount,
     FromPeriod1, FromYear1, ToPeriod1, ToYear1, CreatedBy)
VALUES (@Tenant, @Ppv, N'+', N'', N'ZZZZZZZZZZ', 1, 2026, 12, 2026, @By),
       (@Tenant, @Ppv, N'S', N'', N'ZZZZZZZZZZ', 1, 2026, 12, 2026, @By),
       (@Tenant, @Ppv, N'D', N'', N'ZZZZZZZZZZ', 1, 2026, 12, 2026, @By),
       (@Tenant, @Ppv, N'K', N'', N'ZZZZZZZZZZ', 1, 2026, 12, 2026, @By),
       (@Tenant, @Ppv, N'A', N'', N'ZZZZZZZZZZ', 1, 2026, 12, 2026, @By);

INSERT INTO [cfg].[FieldStatusVariant] (TenantId, FieldStatusVariant, Name, CreatedBy)
VALUES (@Tenant, N'KH00', N'Cambodia group field status', @By);

DECLARE @Fsv bigint =
    (SELECT Id FROM [cfg].[FieldStatusVariant] WHERE TenantId = @Tenant AND FieldStatusVariant = N'KH00');

INSERT INTO [cfg].[FieldStatusGroup] (TenantId, FieldStatusVariantId, FieldStatusGroup, Name, CreatedBy)
VALUES (@Tenant, @Fsv, N'G001', N'General (with text, allocation)', @By),
       (@Tenant, @Fsv, N'G004', N'Cost accounts',                   @By),
       (@Tenant, @Fsv, N'G005', N'Bank accounts',                   @By);

DECLARE @FsgGeneral bigint =
    (SELECT Id FROM [cfg].[FieldStatusGroup] WHERE TenantId = @Tenant AND FieldStatusVariantId = @Fsv AND FieldStatusGroup = N'G001');
DECLARE @FsgCost bigint =
    (SELECT Id FROM [cfg].[FieldStatusGroup] WHERE TenantId = @Tenant AND FieldStatusVariantId = @Fsv AND FieldStatusGroup = N'G004');

INSERT INTO [cfg].[FieldStatusFieldControl]
    (TenantId, FieldStatusGroupId, FieldName, FieldGroup, FieldStatus, DisplayOrder, CreatedBy)
VALUES (@Tenant, @FsgCost, N'CostCenterId',   N'Additional account assignments', N'Required', 10, @By),
       (@Tenant, @FsgCost, N'ProfitCenterId', N'Additional account assignments', N'Optional', 20, @By),
       (@Tenant, @FsgCost, N'LineItemText',   N'General data',                   N'Required', 30, @By);

/* ---------------------------------------------------------------------------
   3. Chart of accounts, account groups, number ranges, document types
   --------------------------------------------------------------------------- */
INSERT INTO [cfg].[ChartOfAccounts]
    (TenantId, ChartOfAccounts, Name, MaintenanceLanguage, AccountNumberLength, ChartType, CreatedBy)
VALUES (@Tenant, N'INT', N'International chart of accounts', N'EN', 6, N'Operational', @By);

DECLARE @Coa bigint =
    (SELECT Id FROM [cfg].[ChartOfAccounts] WHERE TenantId = @Tenant AND ChartOfAccounts = N'INT');

INSERT INTO [cfg].[AccountGroup]
    (TenantId, ChartOfAccountsId, AccountGroup, Name, FromAccount, ToAccount,
     FieldStatusGroupId, AppliesTo, CreatedBy)
VALUES (@Tenant, @Coa, N'BS', N'Balance sheet accounts', N'100000', N'399999', @FsgGeneral, N'GL',       @By),
       (@Tenant, @Coa, N'PL', N'Profit and loss accounts', N'400000', N'699999', @FsgCost,  N'GL',       @By),
       (@Tenant, @Coa, N'CUST', N'Customers',              N'0000000001', N'0000999999', NULL, N'Customer', @By),
       (@Tenant, @Coa, N'VEND', N'Vendors',                N'0001000000', N'0001999999', NULL, N'Vendor',   @By);

DECLARE @AgBs bigint = (SELECT Id FROM [cfg].[AccountGroup] WHERE TenantId = @Tenant AND ChartOfAccountsId = @Coa AND AccountGroup = N'BS');
DECLARE @AgPl bigint = (SELECT Id FROM [cfg].[AccountGroup] WHERE TenantId = @Tenant AND ChartOfAccountsId = @Coa AND AccountGroup = N'PL');
DECLARE @AgCust bigint = (SELECT Id FROM [cfg].[AccountGroup] WHERE TenantId = @Tenant AND ChartOfAccountsId = @Coa AND AccountGroup = N'CUST');
DECLARE @AgVend bigint = (SELECT Id FROM [cfg].[AccountGroup] WHERE TenantId = @Tenant AND ChartOfAccountsId = @Coa AND AccountGroup = N'VEND');

INSERT INTO [cfg].[NumberRangeObject]
    (TenantId, NumberRangeObject, Name, IsYearDependent, IsCompanyCodeDependent,
     NumberLength, Prefix, NumberFormat, GapMonitoring, CreatedBy)
VALUES (@Tenant, N'RF_BELEG', N'Accounting document', 1, 1, 8, N'KSS',
        N'{Prefix}-{Year}-{Type}-{Number:00000000}', 1, @By),
       (@Tenant, N'BP',       N'Business partner',    0, 0, 10, NULL, NULL, 1, @By),
       (@Tenant, N'ASSET',    N'Asset',               0, 1, 12, NULL, NULL, 1, @By),
       (@Tenant, N'CO_ORDER', N'Internal order',      0, 0, 12, N'I',  NULL, 1, @By);

DECLARE @NroDoc bigint = (SELECT Id FROM [cfg].[NumberRangeObject] WHERE TenantId = @Tenant AND NumberRangeObject = N'RF_BELEG');
DECLARE @NroBp bigint = (SELECT Id FROM [cfg].[NumberRangeObject] WHERE TenantId = @Tenant AND NumberRangeObject = N'BP');
DECLARE @NroAsset bigint = (SELECT Id FROM [cfg].[NumberRangeObject] WHERE TenantId = @Tenant AND NumberRangeObject = N'ASSET');
DECLARE @NroOrder bigint = (SELECT Id FROM [cfg].[NumberRangeObject] WHERE TenantId = @Tenant AND NumberRangeObject = N'CO_ORDER');

INSERT INTO [cfg].[DocumentType]
    (TenantId, DocumentType, Name, NumberRangeObjectId, NumberRangeCode, ReverseDocumentType,
     AllowCustomerAccounts, AllowVendorAccounts, AllowGLAccounts, AllowAssetAccounts,
     RequireReferenceNumber, SourceModule, CreatedBy)
VALUES (@Tenant, N'SA', N'G/L account document',  @NroDoc, N'01', N'SA', 0, 0, 1, 0, 0, N'FI', @By),
       (@Tenant, N'DR', N'Customer invoice',      @NroDoc, N'02', N'DA', 1, 0, 1, 0, 1, N'AR', @By),
       (@Tenant, N'DZ', N'Customer payment',      @NroDoc, N'03', N'DA', 1, 0, 1, 0, 0, N'AR', @By),
       (@Tenant, N'KR', N'Vendor invoice',        @NroDoc, N'04', N'KA', 0, 1, 1, 1, 1, N'AP', @By),
       (@Tenant, N'KZ', N'Vendor payment',        @NroDoc, N'05', N'KA', 0, 1, 1, 0, 0, N'AP', @By),
       (@Tenant, N'AA', N'Asset posting',         @NroDoc, N'06', N'AA', 0, 1, 1, 1, 0, N'AA', @By);

INSERT INTO [cfg].[NumberRangeInterval]
    (TenantId, NumberRangeObjectId, NumberRangeCode, FiscalYear, FromNumber, ToNumber,
     CurrentNumber, IsExternal, CreatedBy)
VALUES (@Tenant, @NroDoc,   N'01', 2026, 1, 999999, 0, 0, @By),
       (@Tenant, @NroDoc,   N'02', 2026, 1, 999999, 0, 0, @By),
       (@Tenant, @NroDoc,   N'03', 2026, 1, 999999, 0, 0, @By),
       (@Tenant, @NroDoc,   N'04', 2026, 1, 999999, 0, 0, @By),
       (@Tenant, @NroDoc,   N'05', 2026, 1, 999999, 0, 0, @By),
       (@Tenant, @NroDoc,   N'06', 2026, 1, 999999, 0, 0, @By),
       (@Tenant, @NroBp,    N'01', 0, 1000000001, 1999999999, 1000000005, 0, @By),
       (@Tenant, @NroAsset, N'01', 0, 100000000001, 199999999999, 100000000001, 0, @By),
       (@Tenant, @NroOrder, N'01', 0, 100001, 199999, 100001, 0, @By);

INSERT INTO [cfg].[PostingKey]
    (TenantId, PostingKey, Name, DebitCreditIndicator, AccountType, IsSalesRelated,
     PaymentTransaction, FieldStatusGroupId, CreatedBy)
VALUES (@Tenant, N'40', N'Debit G/L account',    N'S', N'S', 0, 0, @FsgCost,    @By),
       (@Tenant, N'50', N'Credit G/L account',   N'H', N'S', 0, 0, @FsgCost,    @By),
       (@Tenant, N'01', N'Customer invoice',     N'S', N'D', 1, 0, @FsgGeneral, @By),
       (@Tenant, N'11', N'Customer credit memo', N'H', N'D', 1, 0, @FsgGeneral, @By),
       (@Tenant, N'15', N'Incoming payment',     N'H', N'D', 0, 1, @FsgGeneral, @By),
       (@Tenant, N'21', N'Vendor credit memo',   N'S', N'K', 0, 0, @FsgGeneral, @By),
       (@Tenant, N'25', N'Outgoing payment',     N'S', N'K', 0, 1, @FsgGeneral, @By),
       (@Tenant, N'31', N'Vendor invoice',       N'H', N'K', 0, 0, @FsgGeneral, @By),
       (@Tenant, N'70', N'Debit asset',          N'S', N'A', 0, 0, @FsgGeneral, @By),
       (@Tenant, N'75', N'Credit asset',         N'H', N'A', 0, 0, @FsgGeneral, @By);

/* ---------------------------------------------------------------------------
   4. Tax and payment terms
   --------------------------------------------------------------------------- */
INSERT INTO [cfg].[TaxCode]
    (TenantId, CountryCode, TaxCode, Name, TaxType, TaxCategory, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, N'KH', N'V0', N'Input VAT 0%',  N'V', N'VAT', @Open, @Never, @By),
       (@Tenant, N'KH', N'V1', N'Input VAT 10%', N'V', N'VAT', @Open, @Never, @By),
       (@Tenant, N'KH', N'A1', N'Output VAT 10%', N'A', N'VAT', @Open, @Never, @By);

DECLARE @TaxV0 bigint = (SELECT Id FROM [cfg].[TaxCode] WHERE TenantId = @Tenant AND CountryCode = N'KH' AND TaxCode = N'V0');
DECLARE @TaxV1 bigint = (SELECT Id FROM [cfg].[TaxCode] WHERE TenantId = @Tenant AND CountryCode = N'KH' AND TaxCode = N'V1');
DECLARE @TaxA1 bigint = (SELECT Id FROM [cfg].[TaxCode] WHERE TenantId = @Tenant AND CountryCode = N'KH' AND TaxCode = N'A1');

INSERT INTO [cfg].[TaxCodeRate]
    (TenantId, TaxCodeId, ConditionType, ValidFrom, ValidTo, RatePercent, TaxAccountKey, CreatedBy)
VALUES (@Tenant, @TaxV0, N'MWVS', @Open, @Never,  0.0000, N'VST', @By),
       (@Tenant, @TaxV1, N'MWVS', @Open, @Never, 10.0000, N'VST', @By),
       (@Tenant, @TaxA1, N'MWAS', @Open, @Never, 10.0000, N'MWS', @By);

INSERT INTO [cfg].[PaymentTerms]
    (TenantId, PaymentTerms, Name, BaselineDateRule, AdditionalDays, NetDueDays,
     CashDiscount1Days, CashDiscount1Percent, IsForCustomer, IsForVendor, CreatedBy)
VALUES (@Tenant, N'0001', N'Payable immediately', N'DocumentDate', 0,  0, NULL, NULL, 1, 1, @By),
       (@Tenant, N'NT30', N'Net 30 days, 2% in 10 days', N'DocumentDate', 0, 30, 10, 2.0000, 1, 1, @By);

DECLARE @TermsNet30 bigint = (SELECT Id FROM [cfg].[PaymentTerms] WHERE TenantId = @Tenant AND PaymentTerms = N'NT30');

INSERT INTO [cfg].[PaymentMethod]
    (TenantId, CountryCode, PaymentMethod, Name, PaymentType, IsForOutgoing, IsForIncoming,
     RequireBankDetails, PaymentFileFormat, CreatedBy)
VALUES (@Tenant, N'KH', N'T', N'Bank transfer', N'BankTransfer', 1, 1, 1, N'ISO20022', @By);

/* ---------------------------------------------------------------------------
   5. Enterprise structure: 2 companies, 3 company codes, 1 controlling area
   --------------------------------------------------------------------------- */
INSERT INTO [org].[Company]
    (TenantId, CompanyCodeGroup, Name, CountryCode, GroupCurrencyCode, LanguageCode,
     AccountingStandard, ConsolidationRelevant, CreatedBy)
VALUES (@Tenant, N'1000', N'Kampong Speu Sugar Group', N'KH', N'USD', N'EN', N'IFRS', 1, @By),
       (@Tenant, N'2000', N'Siam Cane Trading Group',  N'TH', N'USD', N'EN', N'IFRS', 1, @By);

DECLARE @Company1 bigint = (SELECT Id FROM [org].[Company] WHERE TenantId = @Tenant AND CompanyCodeGroup = N'1000');
DECLARE @Company2 bigint = (SELECT Id FROM [org].[Company] WHERE TenantId = @Tenant AND CompanyCodeGroup = N'2000');

INSERT INTO [org].[CreditControlArea]
    (TenantId, CreditControlArea, Name, CurrencyCode, DefaultCreditLimit, CreatedBy)
VALUES (@Tenant, N'KH00', N'Group credit control', N'USD', 50000.0000, @By);

DECLARE @Cca bigint = (SELECT Id FROM [org].[CreditControlArea] WHERE TenantId = @Tenant AND CreditControlArea = N'KH00');

INSERT INTO [org].[CompanyCode]
    (TenantId, CompanyCode, CompanyId, Name, City, CountryCode, LocalCurrencyCode,
     GroupCurrencyCode, LanguageCode, ChartOfAccountsId, FiscalYearVariantId,
     PostingPeriodVariantId, FieldStatusVariantId, CreditControlAreaId,
     ProfitCenterMandatory, SegmentMandatory, IsProductive, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, N'KH01', @Company1, N'KSS Sugar Mill (Cambodia)',  N'Kampong Speu', N'KH', N'USD', N'USD', N'EN', @Coa, @Fyv, @Ppv, @Fsv, @Cca, 1, 0, 1, @Open, @Never, @By),
       (@Tenant, N'KH02', @Company1, N'KSS Plantation (Cambodia)',  N'Kampong Speu', N'KH', N'USD', N'USD', N'EN', @Coa, @Fyv, @Ppv, @Fsv, @Cca, 1, 0, 1, @Open, @Never, @By),
       (@Tenant, N'TH01', @Company2, N'Siam Cane Trading (Thailand)', N'Bangkok',    N'TH', N'THB', N'USD', N'EN', @Coa, @Fyv, @Ppv, @Fsv, @Cca, 0, 0, 1, @Open, @Never, @By);

DECLARE @Kh01 bigint = (SELECT Id FROM [org].[CompanyCode] WHERE TenantId = @Tenant AND CompanyCode = N'KH01');
DECLARE @Kh02 bigint = (SELECT Id FROM [org].[CompanyCode] WHERE TenantId = @Tenant AND CompanyCode = N'KH02');
DECLARE @Th01 bigint = (SELECT Id FROM [org].[CompanyCode] WHERE TenantId = @Tenant AND CompanyCode = N'TH01');

INSERT INTO [cfg].[LedgerCompanyCode]
    (TenantId, LedgerId, CompanyCodeId, FiscalYearVariantId, PostingPeriodVariantId,
     Currency1TypeCode, Currency2TypeCode, IsActive, CreatedBy)
VALUES (@Tenant, @Ledger, @Kh01, @Fyv, @Ppv, N'10', N'30', 1, @By),
       (@Tenant, @Ledger, @Kh02, @Fyv, @Ppv, N'10', N'30', 1, @By),
       (@Tenant, @Ledger, @Th01, @Fyv, @Ppv, N'10', N'30', 1, @By);

INSERT INTO [org].[BusinessArea] (TenantId, BusinessArea, Name, CreatedBy)
VALUES (@Tenant, N'1000', N'Sugar operations', @By),
       (@Tenant, N'2000', N'Trading',          @By);

DECLARE @Ba1000 bigint = (SELECT Id FROM [org].[BusinessArea] WHERE TenantId = @Tenant AND BusinessArea = N'1000');

INSERT INTO [org].[Segment] (TenantId, Segment, Name, DerivationPriority, CreatedBy)
VALUES (@Tenant, N'SUGAR', N'Sugar production', 10, @By),
       (@Tenant, N'TRADE', N'Trading',          20, @By);

DECLARE @SegSugar bigint = (SELECT Id FROM [org].[Segment] WHERE TenantId = @Tenant AND Segment = N'SUGAR');
DECLARE @SegTrade bigint = (SELECT Id FROM [org].[Segment] WHERE TenantId = @Tenant AND Segment = N'TRADE');

INSERT INTO [org].[ControllingArea]
    (TenantId, ControllingArea, Name, CurrencyCode, CurrencyTypeCode, ChartOfAccountsId,
     FiscalYearVariantId, CostCenterStandardHierarchy, ProfitCenterStandardHierarchy,
     AssignmentControl, ProfitCenterAccountingActive, ValidFromYear, CreatedBy)
VALUES (@Tenant, N'KH00', N'Group controlling area', N'USD', N'20', @Coa, @Fyv,
        N'KH00-CC', N'KH00-PC', N'2', 1, 2026, @By);

DECLARE @Coar bigint = (SELECT Id FROM [org].[ControllingArea] WHERE TenantId = @Tenant AND ControllingArea = N'KH00');

INSERT INTO [org].[ControllingAreaCompanyCode]
    (TenantId, ControllingAreaId, CompanyCodeId, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @Coar, @Kh01, @Open, @Never, @By),
       (@Tenant, @Coar, @Kh02, @Open, @Never, @By),
       (@Tenant, @Coar, @Th01, @Open, @Never, @By);

INSERT INTO [org].[Plant]
    (TenantId, Plant, CompanyCodeId, Name, CountryCode, City, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, N'P100', @Kh01, N'Kampong Speu sugar mill', N'KH', N'Kampong Speu', @Open, @Never, @By),
       (@Tenant, N'P200', @Kh02, N'Cane plantation',         N'KH', N'Kampong Speu', @Open, @Never, @By);

DECLARE @PlantP100 bigint = (SELECT Id FROM [org].[Plant] WHERE TenantId = @Tenant AND Plant = N'P100');

INSERT INTO [org].[Branch]
    (TenantId, Branch, CompanyCodeId, Name, BusinessAreaId, IsHeadOffice, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, N'B001', @Kh01, N'Phnom Penh head office', @Ba1000, 1, @Open, @Never, @By);

/* ---------------------------------------------------------------------------
   6. G/L accounts (shared chart, all three company codes)
   --------------------------------------------------------------------------- */
INSERT INTO [mdm].[GLAccount]
    (TenantId, ChartOfAccountsId, GLAccount, AccountGroupId, AccountType,
     IsBalanceSheetAccount, IsProfitAndLossAccount, IsReconciliationAccount,
     ReconciliationAccountType, RetainedEarningsAccountKey, CostElementCategory, CreatedBy)
VALUES (@Tenant, @Coa, N'110000', @AgBs, N'CashAccount',    1, 0, 0, NULL, NULL, NULL, @By),
       (@Tenant, @Coa, N'120000', @AgBs, N'BalanceSheet',   1, 0, 1, N'D',  NULL, NULL, @By),
       (@Tenant, @Coa, N'125000', @AgBs, N'BalanceSheet',   1, 0, 0, NULL, NULL, NULL, @By),
       (@Tenant, @Coa, N'130000', @AgBs, N'BalanceSheet',   1, 0, 0, NULL, NULL, NULL, @By),
       (@Tenant, @Coa, N'150000', @AgBs, N'BalanceSheet',   1, 0, 1, N'A',  NULL, NULL, @By),
       (@Tenant, @Coa, N'151000', @AgBs, N'BalanceSheet',   1, 0, 1, N'A',  NULL, NULL, @By),
       (@Tenant, @Coa, N'200000', @AgBs, N'BalanceSheet',   1, 0, 1, N'K',  NULL, NULL, @By),
       (@Tenant, @Coa, N'210000', @AgBs, N'BalanceSheet',   1, 0, 0, NULL, NULL, NULL, @By),
       (@Tenant, @Coa, N'225000', @AgBs, N'BalanceSheet',   1, 0, 0, NULL, NULL, NULL, @By),
       (@Tenant, @Coa, N'300000', @AgBs, N'BalanceSheet',   1, 0, 0, NULL, NULL, NULL, @By),
       (@Tenant, @Coa, N'320000', @AgBs, N'BalanceSheet',   1, 0, 0, NULL, N'X', NULL, @By),
       (@Tenant, @Coa, N'400000', @AgPl, N'PrimaryCost',    0, 1, 0, NULL, NULL, N'11', @By),
       (@Tenant, @Coa, N'500000', @AgPl, N'PrimaryCost',    0, 1, 0, NULL, NULL, N'1',  @By),
       (@Tenant, @Coa, N'510000', @AgPl, N'PrimaryCost',    0, 1, 0, NULL, NULL, N'1',  @By),
       (@Tenant, @Coa, N'520000', @AgPl, N'PrimaryCost',    0, 1, 0, NULL, NULL, N'1',  @By),
       (@Tenant, @Coa, N'530000', @AgPl, N'PrimaryCost',    0, 1, 0, NULL, NULL, N'1',  @By),
       (@Tenant, @Coa, N'600000', @AgPl, N'NonOperatingExpenseRevenue', 0, 1, 0, NULL, NULL, NULL, @By),
       (@Tenant, @Coa, N'610000', @AgPl, N'NonOperatingExpenseRevenue', 0, 1, 0, NULL, NULL, NULL, @By),
       (@Tenant, @Coa, N'620000', @AgPl, N'NonOperatingExpenseRevenue', 0, 1, 0, NULL, NULL, NULL, @By),
       (@Tenant, @Coa, N'630000', @AgPl, N'NonOperatingExpenseRevenue', 0, 1, 0, NULL, NULL, NULL, @By);

INSERT INTO [mdm].[GLAccountText] (TenantId, GLAccountId, LanguageCode, ShortText, LongText, CreatedBy)
SELECT @Tenant, a.Id, N'EN', t.ShortText, t.LongText, @By
FROM   [mdm].[GLAccount] AS a
JOIN  (VALUES
        (N'110000', N'Bank USD',        N'Bank account - US Dollar'),
        (N'120000', N'Trade receiv.',   N'Trade receivables - domestic'),
        (N'125000', N'IC receivables',  N'Intercompany receivables'),
        (N'130000', N'Input VAT',       N'Input VAT receivable'),
        (N'150000', N'Machinery',       N'Machinery and equipment'),
        (N'151000', N'Acc. depr.',      N'Accumulated depreciation - machinery'),
        (N'200000', N'Trade payables',  N'Trade payables - domestic'),
        (N'210000', N'Output VAT',      N'Output VAT payable'),
        (N'225000', N'IC payables',     N'Intercompany payables'),
        (N'300000', N'Share capital',   N'Share capital'),
        (N'320000', N'Retained earn.',  N'Retained earnings brought forward'),
        (N'400000', N'Sugar revenue',   N'Revenue - sugar sales'),
        (N'500000', N'Raw material',    N'Raw material and cane purchases'),
        (N'510000', N'Salaries',        N'Salaries and wages'),
        (N'520000', N'Depreciation',    N'Depreciation expense'),
        (N'530000', N'Utilities',       N'Utilities expense'),
        (N'600000', N'FX gain',         N'Realised foreign exchange gain'),
        (N'610000', N'FX loss',         N'Realised foreign exchange loss'),
        (N'620000', N'Disposal gain',    N'Gain on asset disposal'),
        (N'630000', N'Disposal loss',    N'Loss on asset disposal')
      ) AS t(GLAccount, ShortText, LongText) ON t.GLAccount = a.GLAccount
WHERE  a.TenantId = @Tenant AND a.ChartOfAccountsId = @Coa;

/* Every account is opened in every company code, in that company code's own
   currency. Open item management follows the account, not the company code. */
INSERT INTO [mdm].[GLAccountCompanyCode]
    (TenantId, GLAccountId, CompanyCodeId, AccountCurrencyCode, IsOnlyBalancesInLocalCurrency,
     IsOpenItemManaged, IsLineItemDisplay, FieldStatusGroupId, IsRelevantToCashFlow,
     CostCenterRequired, ProfitCenterRequired, CreatedBy)
SELECT @Tenant, a.Id, c.Id, c.LocalCurrencyCode, 0,
       CASE WHEN a.GLAccount IN (N'125000', N'225000') THEN 1 ELSE 0 END,
       1,
       CASE WHEN a.IsProfitAndLossAccount = 1 THEN @FsgCost ELSE @FsgGeneral END,
       CASE WHEN a.GLAccount = N'110000' THEN 1 ELSE 0 END,
       CASE WHEN a.GLAccount IN (N'500000', N'510000', N'520000', N'530000') THEN 1 ELSE 0 END,
       CASE WHEN a.IsProfitAndLossAccount = 1 THEN 1 ELSE 0 END,
       @By
FROM   [mdm].[GLAccount] AS a
CROSS JOIN [org].[CompanyCode] AS c
WHERE  a.TenantId = @Tenant AND c.TenantId = @Tenant;

DECLARE @GlBank bigint      = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'110000');
DECLARE @GlAr bigint        = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'120000');
DECLARE @GlIcAr bigint      = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'125000');
DECLARE @GlAsset bigint     = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'150000');
DECLARE @GlAccDep bigint    = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'151000');
DECLARE @GlAp bigint        = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'200000');
DECLARE @GlOutputVat bigint = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'210000');
DECLARE @GlIcAp bigint      = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'225000');
DECLARE @GlRevenue bigint   = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'400000');
DECLARE @GlMaterial bigint  = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'500000');
DECLARE @GlDepExp bigint    = (SELECT Id FROM [mdm].[GLAccount] WHERE TenantId = @Tenant AND GLAccount = N'520000');

/* ---------------------------------------------------------------------------
   7. Business partners: customer, vendor, one partner in both roles
   --------------------------------------------------------------------------- */
INSERT INTO [mdm].[BusinessPartnerGroup]
    (TenantId, GroupCode, Name, NumberRangeObjectId, NumberRangeCode, IsExternalNumbering, CreatedBy)
VALUES (@Tenant, N'0001', N'Business partner (internal numbering)', @NroBp, N'01', 0, @By);

DECLARE @BpGroup bigint = (SELECT Id FROM [mdm].[BusinessPartnerGroup] WHERE TenantId = @Tenant AND GroupCode = N'0001');

INSERT INTO [mdm].[BusinessPartnerRole]
    (TenantId, RoleCode, Name, RoleCategory, RequiresCompanyCodeData, RequiresSalesArea,
     RequiresPurchasingOrganization, SyncTargetEntity, IsStandardRole, DisplayOrder, CreatedBy)
VALUES (@Tenant, N'000000', N'Business partner (general)', N'General',      0, 0, 0, NULL,                          1, 10, @By),
       (@Tenant, N'FLCU00', N'Customer',                   N'Customer',     0, 1, 0, N'BusinessPartnerCustomer',    1, 20, @By),
       (@Tenant, N'FLCU01', N'FI Customer',                N'FICustomer',   1, 0, 0, N'BusinessPartnerCompanyCode', 1, 30, @By),
       (@Tenant, N'FLVN00', N'Supplier',                   N'Vendor',       0, 0, 1, N'BusinessPartnerVendor',      1, 40, @By),
       (@Tenant, N'FLVN01', N'FI Supplier',                N'FIVendor',     1, 0, 0, N'BusinessPartnerCompanyCode', 1, 50, @By),
       (@Tenant, N'BUP003', N'Employee',                   N'Employee',     0, 0, 0, NULL,                          1, 60, @By),
       (@Tenant, N'ICOMP',  N'Intercompany partner',       N'Intercompany', 1, 0, 0, NULL,                          1, 70, @By);

DECLARE @RoleGeneral bigint = (SELECT Id FROM [mdm].[BusinessPartnerRole] WHERE TenantId = @Tenant AND RoleCode = N'000000');
DECLARE @RoleCust bigint    = (SELECT Id FROM [mdm].[BusinessPartnerRole] WHERE TenantId = @Tenant AND RoleCode = N'FLCU00');
DECLARE @RoleFiCust bigint  = (SELECT Id FROM [mdm].[BusinessPartnerRole] WHERE TenantId = @Tenant AND RoleCode = N'FLCU01');
DECLARE @RoleVend bigint    = (SELECT Id FROM [mdm].[BusinessPartnerRole] WHERE TenantId = @Tenant AND RoleCode = N'FLVN00');
DECLARE @RoleFiVend bigint  = (SELECT Id FROM [mdm].[BusinessPartnerRole] WHERE TenantId = @Tenant AND RoleCode = N'FLVN01');
DECLARE @RoleEmployee bigint = (SELECT Id FROM [mdm].[BusinessPartnerRole] WHERE TenantId = @Tenant AND RoleCode = N'BUP003');
DECLARE @RoleIntercompany bigint = (SELECT Id FROM [mdm].[BusinessPartnerRole] WHERE TenantId = @Tenant AND RoleCode = N'ICOMP');

INSERT INTO [mdm].[BusinessPartner]
    (TenantId, PartnerNumber, PartnerCategory, BusinessPartnerGroupId, Name1, FullName,
     SearchTerm1, FirstName, LastName, NationalityCountryCode, LanguageCode, IndustrySector,
     RegistrationNumber, RegistrationCountryCode, Status, IsIntercompany,
     TradingPartnerCompany, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, N'1000000001', N'2', @BpGroup, N'Angkor Distribution Co., Ltd', N'Angkor Distribution Co., Ltd',
        N'ANGKOR', NULL, NULL, NULL, N'EN', N'WHOLESALE', N'KH-00123456', N'KH', N'Active', 0, NULL, @Open, @Never, @By),
       (@Tenant, N'1000000002', N'2', @BpGroup, N'Mekong Equipment Supply', N'Mekong Equipment Supply',
        N'MEKONG', NULL, NULL, NULL, N'EN', N'MACHINERY', N'KH-00987654', N'KH', N'Active', 0, NULL, @Open, @Never, @By),
       (@Tenant, N'1000000003', N'2', @BpGroup, N'Sokha Trading Co., Ltd', N'Sokha Trading Co., Ltd',
        N'SOKHA', NULL, NULL, NULL, N'EN', N'TRADING', N'KH-00456789', N'KH', N'Active', 0, NULL, @Open, @Never, @By),
       (@Tenant, N'1000000004', N'1', @BpGroup, NULL, N'Sopheak Chan',
        N'SOPHEAK', N'Sopheak', N'Chan', N'KH', N'KM', NULL, NULL, NULL, N'Active', 0, NULL, @Open, @Never, @By),
       (@Tenant, N'1000000005', N'2', @BpGroup, N'Siam Cane Trading Co., Ltd', N'Siam Cane Trading Co., Ltd',
        N'SIAM', NULL, NULL, NULL, N'EN', N'TRADING', N'TH-00112233', N'TH', N'Active', 1, N'2000', @Open, @Never, @By);

DECLARE @BpAngkor bigint = (SELECT Id FROM [mdm].[BusinessPartner] WHERE TenantId = @Tenant AND PartnerNumber = N'1000000001');
DECLARE @BpMekong bigint = (SELECT Id FROM [mdm].[BusinessPartner] WHERE TenantId = @Tenant AND PartnerNumber = N'1000000002');
DECLARE @BpSokha bigint  = (SELECT Id FROM [mdm].[BusinessPartner] WHERE TenantId = @Tenant AND PartnerNumber = N'1000000003');
DECLARE @BpSopheak bigint = (SELECT Id FROM [mdm].[BusinessPartner] WHERE TenantId = @Tenant AND PartnerNumber = N'1000000004');
DECLARE @BpSiam bigint   = (SELECT Id FROM [mdm].[BusinessPartner] WHERE TenantId = @Tenant AND PartnerNumber = N'1000000005');

/* Roles. Sokha Trading holds the customer and the supplier role on one
   identity - the point of the central business partner. */
INSERT INTO [mdm].[BusinessPartnerRoleAssignment]
    (TenantId, BusinessPartnerId, BusinessPartnerRoleId, IsSynchronized, SyncStatus,
     ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @BpAngkor,  @RoleGeneral,      1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpAngkor,  @RoleCust,         1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpAngkor,  @RoleFiCust,       1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpMekong,  @RoleGeneral,      1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpMekong,  @RoleVend,         1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpMekong,  @RoleFiVend,       1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSokha,   @RoleGeneral,      1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSokha,   @RoleCust,         1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSokha,   @RoleFiCust,       1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSokha,   @RoleVend,         1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSokha,   @RoleFiVend,       1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSopheak, @RoleGeneral,      1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSopheak, @RoleEmployee,     1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSiam,    @RoleGeneral,      1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSiam,    @RoleIntercompany, 1, N'Completed', @Open, @Never, @By),
       -- The intercompany partner has a customer company code segment, so it
       -- needs the customer role that segment belongs to. Data without the
       -- matching role is exactly what BP_CHECK reports as an inconsistency.
       (@Tenant, @BpSiam,    @RoleCust,         1, N'Completed', @Open, @Never, @By),
       (@Tenant, @BpSiam,    @RoleFiCust,       1, N'Completed', @Open, @Never, @By);

INSERT INTO [mdm].[Address]
    (TenantId, AddressNumber, Name1, Street, HouseNumber, District, City, PostalCode,
     RegionCode, CountryCode, LanguageCode, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, N'0000000001', N'Angkor Distribution Co., Ltd', N'Norodom Boulevard', N'118', N'Chamkarmon', N'Phnom Penh', N'12301', N'PP', N'KH', N'EN', @Open, @Never, @By),
       (@Tenant, N'0000000002', N'Mekong Equipment Supply',      N'Russian Boulevard', N'42',  N'Toul Kork',  N'Phnom Penh', N'12151', N'PP', N'KH', N'EN', @Open, @Never, @By),
       (@Tenant, N'0000000003', N'Sokha Trading Co., Ltd',       N'National Road 4',   N'7',   NULL,          N'Kampong Speu', N'05101', N'KS', N'KH', N'EN', @Open, @Never, @By),
       (@Tenant, N'0000000004', N'Sopheak Chan',                 N'Street 271',        N'908', N'Sen Sok',    N'Phnom Penh', N'12102', N'PP', N'KH', N'KM', @Open, @Never, @By),
       (@Tenant, N'0000000005', N'Siam Cane Trading Co., Ltd',   N'Sukhumvit Road',    N'55',  N'Watthana',   N'Bangkok',    N'10110', N'BK', N'TH', N'EN', @Open, @Never, @By);

INSERT INTO [mdm].[BusinessPartnerAddress]
    (TenantId, BusinessPartnerId, AddressId, AddressUsage, IsStandardAddress, ValidFrom, ValidTo, CreatedBy)
SELECT @Tenant, b.Id, a.Id, N'STANDARD', 1, @Open, @Never, @By
FROM  (VALUES (N'1000000001', N'0000000001'), (N'1000000002', N'0000000002'),
              (N'1000000003', N'0000000003'), (N'1000000004', N'0000000004'),
              (N'1000000005', N'0000000005')
      ) AS m(PartnerNumber, AddressNumber)
JOIN   [mdm].[BusinessPartner] AS b ON b.TenantId = @Tenant AND b.PartnerNumber = m.PartnerNumber
JOIN   [mdm].[Address] AS a ON a.TenantId = @Tenant AND a.AddressNumber = m.AddressNumber;

UPDATE b
SET    b.DefaultAddressId = ba.AddressId
FROM   [mdm].[BusinessPartner] AS b
JOIN   [mdm].[BusinessPartnerAddress] AS ba ON ba.BusinessPartnerId = b.Id
WHERE  b.TenantId = @Tenant;

INSERT INTO [mdm].[BusinessPartnerCommunication]
    (TenantId, BusinessPartnerId, CommunicationType, SequenceNumber, CountryDialCode,
     Value, IsDefault, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @BpAngkor, N'EMAIL', 1, NULL,   N'ap@angkor-distribution.example', 1, @Open, @Never, @By),
       (@Tenant, @BpAngkor, N'PHONE', 1, N'+855', N'23 123 456',                    1, @Open, @Never, @By),
       (@Tenant, @BpMekong, N'EMAIL', 1, NULL,   N'sales@mekong-equipment.example', 1, @Open, @Never, @By),
       (@Tenant, @BpSokha,  N'EMAIL', 1, NULL,   N'office@sokha-trading.example',   1, @Open, @Never, @By),
       (@Tenant, @BpSiam,   N'EMAIL', 1, NULL,   N'finance@siam-cane.example',      1, @Open, @Never, @By);

INSERT INTO [mdm].[BusinessPartnerTaxNumber]
    (TenantId, BusinessPartnerId, TaxNumberCategory, TaxNumber, CountryCode,
     IsNaturalPerson, IsValidated, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @BpAngkor, N'KH0', N'K001-901234567', N'KH', 0, 1, @Open, @Never, @By),
       (@Tenant, @BpMekong, N'KH0', N'K001-901234568', N'KH', 0, 1, @Open, @Never, @By),
       (@Tenant, @BpSokha,  N'KH0', N'K001-901234569', N'KH', 0, 1, @Open, @Never, @By),
       (@Tenant, @BpSiam,   N'TH0', N'0105551234567',  N'TH', 0, 1, @Open, @Never, @By);

/* Company code segments: the customer side and the supplier side of the same
   partner point at different reconciliation accounts. */
INSERT INTO [mdm].[BusinessPartnerCompanyCode]
    (TenantId, BusinessPartnerId, CompanyCodeId, RoleCategory, ReconciliationGLAccountId,
     SortKey, PaymentTermsId, PaymentMethods, IsClearingWithVendorAllowed, CreatedBy)
VALUES (@Tenant, @BpAngkor, @Kh01, N'Customer', @GlAr, N'001', @TermsNet30, N'T', 0, @By),
       (@Tenant, @BpSokha,  @Kh01, N'Customer', @GlAr, N'001', @TermsNet30, N'T', 1, @By),
       (@Tenant, @BpMekong, @Kh01, N'Vendor',   @GlAp, N'001', @TermsNet30, N'T', 0, @By),
       (@Tenant, @BpSokha,  @Kh01, N'Vendor',   @GlAp, N'001', @TermsNet30, N'T', 1, @By),
       (@Tenant, @BpSiam,   @Kh01, N'Customer', @GlIcAr, N'001', @TermsNet30, N'T', 0, @By);

INSERT INTO [mdm].[BusinessPartnerCustomer]
    (TenantId, BusinessPartnerId, CustomerNumber, CustomerAccountGroupId, CustomerGroup,
     IndustryKey, TaxClassification, VatRegistrationNumber, CreatedBy)
VALUES (@Tenant, @BpAngkor, N'0000010001', @AgCust, N'01', N'WHOL', N'1', N'K001-901234567', @By),
       (@Tenant, @BpSokha,  N'0000010002', @AgCust, N'01', N'TRAD', N'1', N'K001-901234569', @By),
       (@Tenant, @BpSiam,   N'0000010003', @AgCust, N'02', N'TRAD', N'0', N'0105551234567',  @By);

INSERT INTO [mdm].[BusinessPartnerVendor]
    (TenantId, BusinessPartnerId, VendorNumber, VendorAccountGroupId, IndustryKey,
     VatRegistrationNumber, IsServiceProvider, CreatedBy)
VALUES (@Tenant, @BpMekong, N'0001000001', @AgVend, N'MACH', N'K001-901234568', 0, @By),
       (@Tenant, @BpSokha,  N'0001000002', @AgVend, N'TRAD', N'K001-901234569', 0, @By);

INSERT INTO [mdm].[BusinessPartnerCreditProfile]
    (TenantId, BusinessPartnerId, CreditControlAreaId, CreditLimit, CreditLimitCurrencyCode,
     CreditExposure, CreditLimitUsedPercent, RiskCategory, LastReviewDate, NextReviewDate, CreatedBy)
VALUES (@Tenant, @BpAngkor, @Cca, 50000.0000, N'USD', 11000.0000, 22.0000, N'B', '2026-01-05', '2026-07-05', @By),
       (@Tenant, @BpSokha,  @Cca, 20000.0000, N'USD',     0.0000,  0.0000, N'C', '2026-01-05', '2026-07-05', @By);

/* ---------------------------------------------------------------------------
   8. Controlling: cost centres, profit centres, cost elements, internal order
   --------------------------------------------------------------------------- */
INSERT INTO [co].[CostCenterCategory] (TenantId, CategoryCode, Name, CreatedBy)
VALUES (@Tenant, N'F', N'Production',     @By),
       (@Tenant, N'V', N'Sales',          @By),
       (@Tenant, N'H', N'Administration', @By);

DECLARE @CatProd bigint = (SELECT Id FROM [co].[CostCenterCategory] WHERE TenantId = @Tenant AND CategoryCode = N'F');
DECLARE @CatSales bigint = (SELECT Id FROM [co].[CostCenterCategory] WHERE TenantId = @Tenant AND CategoryCode = N'V');
DECLARE @CatAdmin bigint = (SELECT Id FROM [co].[CostCenterCategory] WHERE TenantId = @Tenant AND CategoryCode = N'H');

INSERT INTO [co].[HierarchyNode]
    (TenantId, ControllingAreaId, HierarchyType, HierarchyId, NodeCode, ParentNodeId,
     NodeName, NodeLevel, DisplayOrder, HierarchyPath, IsStandardHierarchy, IsLeaf,
     ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @Coar, N'CostCenter',   N'KH00-CC', N'ROOT', NULL, N'KSS group',          1, 10, N'/ROOT',            1, 0, @Open, @Never, @By),
       (@Tenant, @Coar, N'ProfitCenter', N'KH00-PC', N'ROOT', NULL, N'KSS group',          1, 10, N'/ROOT',            1, 0, @Open, @Never, @By);

DECLARE @CcRoot bigint = (SELECT Id FROM [co].[HierarchyNode] WHERE TenantId = @Tenant AND HierarchyType = N'CostCenter' AND NodeCode = N'ROOT');
DECLARE @PcRoot bigint = (SELECT Id FROM [co].[HierarchyNode] WHERE TenantId = @Tenant AND HierarchyType = N'ProfitCenter' AND NodeCode = N'ROOT');

INSERT INTO [co].[HierarchyNode]
    (TenantId, ControllingAreaId, HierarchyType, HierarchyId, NodeCode, ParentNodeId,
     NodeName, NodeLevel, DisplayOrder, HierarchyPath, IsStandardHierarchy, IsLeaf,
     ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @Coar, N'CostCenter',   N'KH00-CC', N'MILL',  @CcRoot, N'Sugar mill',      2, 10, N'/ROOT/MILL',  1, 1, @Open, @Never, @By),
       (@Tenant, @Coar, N'CostCenter',   N'KH00-CC', N'ADMIN', @CcRoot, N'Administration',  2, 20, N'/ROOT/ADMIN', 1, 1, @Open, @Never, @By),
       (@Tenant, @Coar, N'ProfitCenter', N'KH00-PC', N'SUGAR', @PcRoot, N'Sugar',           2, 10, N'/ROOT/SUGAR', 1, 1, @Open, @Never, @By),
       (@Tenant, @Coar, N'ProfitCenter', N'KH00-PC', N'TRADE', @PcRoot, N'Trading',         2, 20, N'/ROOT/TRADE', 1, 1, @Open, @Never, @By);

DECLARE @NodeMill bigint  = (SELECT Id FROM [co].[HierarchyNode] WHERE TenantId = @Tenant AND HierarchyType = N'CostCenter' AND NodeCode = N'MILL');
DECLARE @NodeAdmin bigint = (SELECT Id FROM [co].[HierarchyNode] WHERE TenantId = @Tenant AND HierarchyType = N'CostCenter' AND NodeCode = N'ADMIN');
DECLARE @NodeSugar bigint = (SELECT Id FROM [co].[HierarchyNode] WHERE TenantId = @Tenant AND HierarchyType = N'ProfitCenter' AND NodeCode = N'SUGAR');
DECLARE @NodeTrade bigint = (SELECT Id FROM [co].[HierarchyNode] WHERE TenantId = @Tenant AND HierarchyType = N'ProfitCenter' AND NodeCode = N'TRADE');

INSERT INTO [co].[ProfitCenter]
    (TenantId, ControllingAreaId, ProfitCenter, Name, Description, HierarchyNodeId,
     CompanyCodeId, SegmentId, ResponsiblePersonPartnerId, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @Coar, N'PC1000', N'Sugar mill', N'Sugar production profit centre', @NodeSugar, @Kh01, @SegSugar, @BpSopheak, @Open, @Never, @By),
       (@Tenant, @Coar, N'PC2000', N'Trading',    N'Trading profit centre',          @NodeTrade, @Th01, @SegTrade, NULL,       @Open, @Never, @By);

DECLARE @PcSugar bigint = (SELECT Id FROM [co].[ProfitCenter] WHERE TenantId = @Tenant AND ProfitCenter = N'PC1000');
DECLARE @PcTrade bigint = (SELECT Id FROM [co].[ProfitCenter] WHERE TenantId = @Tenant AND ProfitCenter = N'PC2000');

INSERT INTO [co].[CostCenter]
    (TenantId, ControllingAreaId, CostCenter, Name, Description, CostCenterCategoryId,
     HierarchyNodeId, CompanyCodeId, BusinessAreaId, ProfitCenterId, SegmentId, PlantId,
     ResponsiblePersonPartnerId, CurrencyCode, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @Coar, N'1000', N'Administration', N'Head office administration', @CatAdmin, @NodeAdmin, @Kh01, @Ba1000, @PcSugar, @SegSugar, NULL,       @BpSopheak, N'USD', @Open, @Never, @By),
       (@Tenant, @Coar, N'2000', N'Production',     N'Sugar mill production',      @CatProd,  @NodeMill,  @Kh01, @Ba1000, @PcSugar, @SegSugar, @PlantP100, @BpSopheak, N'USD', @Open, @Never, @By),
       (@Tenant, @Coar, N'3000', N'Sales',          N'Domestic sales',             @CatSales, @NodeAdmin, @Kh01, @Ba1000, @PcSugar, @SegSugar, NULL,       NULL,       N'USD', @Open, @Never, @By);

DECLARE @CcAdmin bigint = (SELECT Id FROM [co].[CostCenter] WHERE TenantId = @Tenant AND CostCenter = N'1000');
DECLARE @CcProd bigint  = (SELECT Id FROM [co].[CostCenter] WHERE TenantId = @Tenant AND CostCenter = N'2000');
DECLARE @CcSales bigint = (SELECT Id FROM [co].[CostCenter] WHERE TenantId = @Tenant AND CostCenter = N'3000');

/* Primary cost elements mirror the P&L accounts; 940000 is a secondary cost
   element and deliberately has no G/L account. */
INSERT INTO [co].[CostElement]
    (TenantId, ControllingAreaId, CostElement, CostElementCategory, IsPrimary, GLAccountId,
     Name, Description, DefaultAccountAssignmentType, DefaultCostCenterId, ValidFrom, ValidTo, CreatedBy)
SELECT @Tenant, @Coar, a.GLAccount, N'1', 1, a.Id, t.Name, t.Name,
       CASE WHEN a.GLAccount = N'530000' THEN N'CostCenter' ELSE N'None' END,
       CASE WHEN a.GLAccount = N'530000' THEN @CcAdmin ELSE NULL END,
       @Open, @Never, @By
FROM   [mdm].[GLAccount] AS a
JOIN  (VALUES (N'500000', N'Raw material'), (N'510000', N'Salaries'),
              (N'520000', N'Depreciation'), (N'530000', N'Utilities')
      ) AS t(GLAccount, Name) ON t.GLAccount = a.GLAccount
WHERE  a.TenantId = @Tenant;

INSERT INTO [co].[CostElement]
    (TenantId, ControllingAreaId, CostElement, CostElementCategory, IsPrimary, GLAccountId,
     Name, Description, DefaultAccountAssignmentType, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @Coar, N'940000', N'42', 0, NULL, N'Assessment - administration',
        N'Secondary cost element for administration assessment', N'None', @Open, @Never, @By);

INSERT INTO [co].[InternalOrderType]
    (TenantId, OrderType, Name, OrderCategory, NumberRangeObjectId, NumberRangeCode,
     IsBudgetControlActive, CreatedBy)
VALUES (@Tenant, N'OH01', N'Overhead order', N'Overhead', @NroOrder, N'01', 1, @By);

DECLARE @OrderType bigint = (SELECT Id FROM [co].[InternalOrderType] WHERE TenantId = @Tenant AND OrderType = N'OH01');

INSERT INTO [co].[InternalOrder]
    (TenantId, ControllingAreaId, OrderNumber, OrderTypeId, Description, CompanyCodeId,
     BusinessAreaId, ResponsibleCostCenterId, ProfitCenterId, SegmentId,
     ResponsiblePersonPartnerId, CurrencyCode, SystemStatus, IsBudgetControlActive,
     WorkStartDate, WorkEndDate, CreatedBy)
VALUES (@Tenant, @Coar, N'I-100001', @OrderType, N'Harvest campaign 2026', @Kh01,
        @Ba1000, @CcProd, @PcSugar, @SegSugar, @BpSopheak, N'USD', N'Released', 1,
        '2026-01-01', '2026-06-30', @By);

DECLARE @Order1 bigint = (SELECT Id FROM [co].[InternalOrder] WHERE TenantId = @Tenant AND OrderNumber = N'I-100001');

INSERT INTO [co].[InternalOrderBudget]
    (TenantId, InternalOrderId, FiscalYear, BudgetVersion, CurrencyCode, OriginalBudget,
     SupplementAmount, ReturnAmount, CurrentBudget, AssignedAmount, AvailableAmount,
     UsagePercent, WarningThresholdPercent, ErrorThresholdPercent, CreatedBy)
VALUES (@Tenant, @Order1, 2026, N'000', N'USD', 50000.0000, 0.0000, 0.0000,
        50000.0000, 0.0000, 50000.0000, 0.0000, 80.0000, 100.0000, @By);

/* ---------------------------------------------------------------------------
   9. Asset accounting master data
   --------------------------------------------------------------------------- */
INSERT INTO [fin].[AssetClass]
    (TenantId, AssetClass, Name, AccountDeterminationKey, NumberRangeObjectId, NumberRangeCode,
     DefaultUsefulLifeYears, AllowSubNumbers, CreatedBy)
VALUES (@Tenant, N'2000', N'Machinery and equipment', N'20000', @NroAsset, N'01', 10, 1, @By);

DECLARE @AssetClass bigint = (SELECT Id FROM [fin].[AssetClass] WHERE TenantId = @Tenant AND AssetClass = N'2000');

INSERT INTO [fin].[DepreciationArea]
    (TenantId, CompanyCodeId, DepreciationArea, Name, AreaType, LedgerId, AccountingPrincipleId,
     CurrencyCode, PostsToGeneralLedger, IsRealDepreciationArea, AcquisitionValueRule,
     NetBookValueRule, IsActive, CreatedBy)
VALUES (@Tenant, @Kh01, N'01', N'Book depreciation (IFRS)', N'Book', @Ledger, @Principle, N'USD', N'RealTime', 1, N'PositiveOnly', N'PositiveOnly', 1, @By),
       (@Tenant, @Kh01, N'15', N'Tax depreciation',         N'Tax',  NULL,    NULL,       N'USD', N'NoPosting', 1, N'PositiveOnly', N'PositiveOnly', 1, @By);

DECLARE @Area01 bigint = (SELECT Id FROM [fin].[DepreciationArea] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND DepreciationArea = N'01');
DECLARE @Area15 bigint = (SELECT Id FROM [fin].[DepreciationArea] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND DepreciationArea = N'15');

INSERT INTO [fin].[DepreciationKey]
    (TenantId, DepreciationKey, Name, DepreciationMethod, BaseValueRule,
     PeriodControlAcquisition, PeriodControlAddition, PeriodControlRetirement,
     PeriodControlTransfer, IsScrapValueConsidered, CreatedBy)
VALUES (@Tenant, N'LINR', N'Straight line, pro rata', N'StraightLine', N'AcquisitionValue',
        N'01', N'01', N'01', N'01', 1, @By);

DECLARE @DepKey bigint = (SELECT Id FROM [fin].[DepreciationKey] WHERE TenantId = @Tenant AND DepreciationKey = N'LINR');

INSERT INTO [fin].[AssetClassDepreciationArea]
    (TenantId, AssetClassId, DepreciationAreaId, DepreciationKeyId, UsefulLifeYears,
     UsefulLifePeriods, CreatedBy)
VALUES (@Tenant, @AssetClass, @Area01, @DepKey, 10, 0, @By),
       (@Tenant, @AssetClass, @Area15, @DepKey, 8,  0, @By);

INSERT INTO [fin].[AssetTransactionType]
    (TenantId, TransactionType, Name, TransactionCategory, DebitCreditIndicator,
     AffectsAcquisitionValue, AffectsAccumulatedDepreciation, CreatedBy)
VALUES (@Tenant, N'100', N'External acquisition',   N'Acquisition', N'S', 1, 0, @By),
       (@Tenant, N'200', N'Retirement without revenue', N'Retirement', N'H', 1, 1, @By);

DECLARE @TtAcquire bigint = (SELECT Id FROM [fin].[AssetTransactionType] WHERE TenantId = @Tenant AND TransactionType = N'100');

INSERT INTO [fin].[Asset]
    (TenantId, CompanyCodeId, AssetNumber, AssetSubNumber, AssetClassId, Description,
     SerialNumber, InventoryNumber, Quantity, UnitOfMeasure, CapitalizationDate,
     AcquisitionDate, InServiceDate, CostCenterId, ProfitCenterId, SegmentId,
     BusinessAreaId, PlantId, ResponsiblePersonPartnerId, VendorBusinessPartnerId,
     ManufacturerName, Status, CreatedBy)
VALUES (@Tenant, @Kh01, N'100000000001', 0, @AssetClass, N'Sugar mill boiler',
        N'BLR-2026-0091', N'INV-000123', 1.000000, N'EA', '2026-01-15',
        '2026-01-15', '2026-01-20', @CcProd, @PcSugar, @SegSugar, @Ba1000, @PlantP100,
        @BpSopheak, @BpMekong, N'Mekong Equipment Supply', N'Capitalized', @By);

DECLARE @Asset1 bigint = (SELECT Id FROM [fin].[Asset] WHERE TenantId = @Tenant AND AssetNumber = N'100000000001' AND AssetSubNumber = 0);

INSERT INTO [fin].[AssetDepreciationArea]
    (TenantId, AssetId, DepreciationAreaId, DepreciationKeyId, UsefulLifeYears,
     UsefulLifePeriods, ExpiredUsefulLifeYears, ExpiredUsefulLifePeriods,
     DepreciationStartDate, ScrapValue, CreatedBy)
VALUES (@Tenant, @Asset1, @Area01, @DepKey, 10, 0, 0, 1, '2026-01-15', 0.0000, @By),
       (@Tenant, @Asset1, @Area15, @DepKey,  8, 0, 0, 1, '2026-01-15', 0.0000, @By);

INSERT INTO [fin].[AssetTimeDependent]
    (TenantId, AssetId, ValidFrom, ValidTo, CostCenterId, ProfitCenterId, SegmentId,
     PlantId, IsShutdown, CreatedBy)
VALUES (@Tenant, @Asset1, '2026-01-15', @Never, @CcProd, @PcSugar, @SegSugar, @PlantP100, 0, @By);

/* Account determination for asset accounting. Without these rows the asset
   service refuses to post rather than guessing an account. */
INSERT INTO [cfg].[AccountDeterminationRule]
    (TenantId, ChartOfAccountsId, TransactionKey, AccountModifier, CompanyCodeId,
     DebitGLAccountId, CreditGLAccountId, Description, ValidFrom, ValidTo, CreatedBy)
SELECT @Tenant, @Coa, t.TransactionKey, N'20000', NULL, a.Id, a.Id, t.Description,
       @Open, @Never, @By
FROM  (VALUES
        (N'ANL', N'150000', N'Asset balance sheet account'),
        (N'AFA', N'151000', N'Accumulated depreciation'),
        (N'AFX', N'520000', N'Depreciation expense'),
        (N'AAV', N'620000', N'Gain on asset disposal'),
        (N'AAL', N'630000', N'Loss on asset disposal')
      ) AS t(TransactionKey, GLAccount, Description)
JOIN   [mdm].[GLAccount] AS a
       ON a.TenantId = @Tenant AND a.ChartOfAccountsId = @Coa AND a.GLAccount = t.GLAccount;

/* ---------------------------------------------------------------------------
   10. Postings
        DOC 1  customer invoice, USD, with output VAT
        DOC 2  vendor invoice, KHR - foreign currency for a USD company code
        DOC 3  incoming payment, partial clearing of DOC 1
        DOC 4  intercompany: KH01 side
        DOC 5  intercompany: TH01 side, local currency THB
        DOC 6  asset acquisition from a vendor
        DOC 7  monthly depreciation
   --------------------------------------------------------------------------- */
DECLARE @DtSa bigint = (SELECT Id FROM [cfg].[DocumentType] WHERE TenantId = @Tenant AND DocumentType = N'SA');
DECLARE @DtDr bigint = (SELECT Id FROM [cfg].[DocumentType] WHERE TenantId = @Tenant AND DocumentType = N'DR');
DECLARE @DtDz bigint = (SELECT Id FROM [cfg].[DocumentType] WHERE TenantId = @Tenant AND DocumentType = N'DZ');
DECLARE @DtKr bigint = (SELECT Id FROM [cfg].[DocumentType] WHERE TenantId = @Tenant AND DocumentType = N'KR');
DECLARE @DtAa bigint = (SELECT Id FROM [cfg].[DocumentType] WHERE TenantId = @Tenant AND DocumentType = N'AA');

DECLARE @Doc1 nvarchar(20) = N'KSS-2026-DR-00000001';
DECLARE @Doc2 nvarchar(20) = N'KSS-2026-KR-00000001';
DECLARE @Doc3 nvarchar(20) = N'KSS-2026-DZ-00000001';
DECLARE @Doc4 nvarchar(20) = N'KSS-2026-SA-00000001';
DECLARE @Doc5 nvarchar(20) = N'SCT-2026-SA-00000001';
DECLARE @Doc6 nvarchar(20) = N'KSS-2026-AA-00000001';
DECLARE @Doc7 nvarchar(20) = N'KSS-2026-AA-00000002';

INSERT INTO [fin].[JournalEntryHeader]
    (TenantId, CompanyCodeId, FiscalYear, DocumentNumber, LedgerId, DocumentTypeId,
     DocumentDate, PostingDate, EntryDate, EntryTime, FiscalPeriod, DocumentCurrencyCode,
     LocalCurrencyCode, GroupCurrencyCode, ExchangeRate, ExchangeRateType, GroupExchangeRate,
     TotalDebitAmount, TotalCreditAmount, ReferenceDocumentNumber, DocumentHeaderText,
     Status, PostedAt, PostedBy, IsIntercompany, IntercompanyDocumentNumber,
     SourceModule, TransactionCode, CreatedBy)
VALUES
 (@Tenant, @Kh01, 2026, @Doc1, @Ledger, @DtDr, '2026-01-20', '2026-01-20', '2026-01-20', '09:15:00', 1,
  N'USD', N'USD', N'USD', 1.000000, N'M', 1.000000, 11000.0000, 11000.0000, N'INV-2026-0001',
  N'Sugar delivery January', N'Posted', '2026-01-20T02:15:00', N'kss.accountant', 0, NULL, N'AR', N'FB70', @By),
 (@Tenant, @Kh01, 2026, @Doc2, @Ledger, @DtKr, '2026-01-22', '2026-01-22', '2026-01-22', '11:02:00', 1,
  N'KHR', N'USD', N'USD', 0.000244, N'M', 0.000244, 41000000.0000, 41000000.0000, N'MES-8842',
  N'Cane purchase - Riel invoice', N'Posted', '2026-01-22T04:02:00', N'kss.accountant', 0, NULL, N'AP', N'FB60', @By),
 (@Tenant, @Kh01, 2026, @Doc3, @Ledger, @DtDz, '2026-02-05', '2026-02-05', '2026-02-05', '14:40:00', 2,
  N'USD', N'USD', N'USD', 1.000000, N'M', 1.000000, 5000.0000, 5000.0000, N'BANK-0207',
  N'Part payment Angkor Distribution', N'Posted', '2026-02-05T07:40:00', N'kss.accountant', 0, NULL, N'AR', N'F-28', @By),
 (@Tenant, @Kh01, 2026, @Doc4, @Ledger, @DtSa, '2026-02-10', '2026-02-10', '2026-02-10', '10:05:00', 2,
  N'USD', N'USD', N'USD', 1.000000, N'M', 1.000000, 8000.0000, 8000.0000, N'IC-2026-000001',
  N'Intercompany sale to Siam Cane', N'Posted', '2026-02-10T03:05:00', N'kss.accountant', 1, N'IC-2026-000001', N'FI', N'FB50', @By),
 (@Tenant, @Th01, 2026, @Doc5, @Ledger, @DtSa, '2026-02-10', '2026-02-10', '2026-02-10', '10:05:00', 2,
  N'USD', N'THB', N'USD', 35.000000, N'M', 1.000000, 8000.0000, 8000.0000, N'IC-2026-000001',
  N'Intercompany purchase from KSS', N'Posted', '2026-02-10T03:05:00', N'kss.accountant', 1, N'IC-2026-000001', N'FI', N'FB50', @By),
 (@Tenant, @Kh01, 2026, @Doc6, @Ledger, @DtAa, '2026-01-15', '2026-01-15', '2026-01-15', '08:30:00', 1,
  N'USD', N'USD', N'USD', 1.000000, N'M', 1.000000, 120000.0000, 120000.0000, N'MES-8801',
  N'Boiler acquisition', N'Posted', '2026-01-15T01:30:00', N'kss.accountant', 0, NULL, N'AA', N'F-90', @By),
 (@Tenant, @Kh01, 2026, @Doc7, @Ledger, @DtAa, '2026-01-31', '2026-01-31', '2026-01-31', '23:10:00', 1,
  N'USD', N'USD', N'USD', 1.000000, N'M', 1.000000, 1000.0000, 1000.0000, NULL,
  N'Depreciation January 2026', N'Posted', '2026-01-31T16:10:00', N'AFAB', 0, NULL, N'AA', N'AFAB', @By);

DECLARE @H1 bigint = (SELECT Id FROM [fin].[JournalEntryHeader] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND FiscalYear = 2026 AND DocumentNumber = @Doc1 AND LedgerId = @Ledger);
DECLARE @H2 bigint = (SELECT Id FROM [fin].[JournalEntryHeader] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND FiscalYear = 2026 AND DocumentNumber = @Doc2 AND LedgerId = @Ledger);
DECLARE @H3 bigint = (SELECT Id FROM [fin].[JournalEntryHeader] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND FiscalYear = 2026 AND DocumentNumber = @Doc3 AND LedgerId = @Ledger);
DECLARE @H4 bigint = (SELECT Id FROM [fin].[JournalEntryHeader] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND FiscalYear = 2026 AND DocumentNumber = @Doc4 AND LedgerId = @Ledger);
DECLARE @H5 bigint = (SELECT Id FROM [fin].[JournalEntryHeader] WHERE TenantId = @Tenant AND CompanyCodeId = @Th01 AND FiscalYear = 2026 AND DocumentNumber = @Doc5 AND LedgerId = @Ledger);
DECLARE @H6 bigint = (SELECT Id FROM [fin].[JournalEntryHeader] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND FiscalYear = 2026 AND DocumentNumber = @Doc6 AND LedgerId = @Ledger);
DECLARE @H7 bigint = (SELECT Id FROM [fin].[JournalEntryHeader] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND FiscalYear = 2026 AND DocumentNumber = @Doc7 AND LedgerId = @Ledger);

INSERT INTO [fin].[JournalEntryLine]
    (TenantId, JournalEntryHeaderId, CompanyCodeId, FiscalYear, DocumentNumber, LineItemNumber,
     LedgerId, PostingDate, FiscalPeriod, PostingKey, DebitCreditIndicator, AccountType,
     GLAccountId, GLAccount, BusinessPartnerId, BusinessPartnerRoleCategory, AssetId,
     CostCenterId, ProfitCenterId, SegmentId, BusinessAreaId, InternalOrderId,
     PartnerCompanyCodeId, TradingPartnerCompany, PlantId,
     DocumentCurrencyCode, AmountInDocumentCurrency, LocalCurrencyCode, AmountInLocalCurrency,
     GroupCurrencyCode, AmountInGroupCurrency, ExchangeRate, TaxCodeId, IsTaxLine,
     TaxBaseAmountInDocumentCurrency, TaxAmountInDocumentCurrency, TaxAmountInLocalCurrency,
     PaymentTermsId, BaselineDate, DueDate, IsOpenItemManaged, ClearingStatus,
     InvoiceReferenceDocumentNumber, AssignmentReference, LineItemText, SourceModule, CreatedBy)
VALUES
 /* DOC 1 - customer invoice 11,000 USD gross (10,000 net + 1,000 output VAT) */
 (@Tenant, @H1, @Kh01, 2026, @Doc1, 1, @Ledger, '2026-01-20', 1, N'01', N'S', N'D',
  @GlAr, N'120000', @BpAngkor, N'Customer', NULL, NULL, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, NULL,
  N'USD', 11000.0000, N'USD', 11000.0000, N'USD', 11000.0000, 1.000000, NULL, 0, NULL, NULL, NULL,
  @TermsNet30, '2026-01-20', '2026-02-19', 1, N'Open', NULL, N'INV-2026-0001', N'Sugar delivery January', N'AR', @By),
 (@Tenant, @H1, @Kh01, 2026, @Doc1, 2, @Ledger, '2026-01-20', 1, N'50', N'H', N'S',
  @GlRevenue, N'400000', NULL, NULL, NULL, @CcSales, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, @PlantP100,
  N'USD', -10000.0000, N'USD', -10000.0000, N'USD', -10000.0000, 1.000000, @TaxA1, 0, 10000.0000, 1000.0000, 1000.0000,
  NULL, NULL, NULL, 0, N'Open', NULL, N'INV-2026-0001', N'Sugar 200 t', N'AR', @By),
 (@Tenant, @H1, @Kh01, 2026, @Doc1, 3, @Ledger, '2026-01-20', 1, N'50', N'H', N'S',
  @GlOutputVat, N'210000', NULL, NULL, NULL, NULL, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, NULL,
  N'USD', -1000.0000, N'USD', -1000.0000, N'USD', -1000.0000, 1.000000, @TaxA1, 1, 10000.0000, 1000.0000, 1000.0000,
  NULL, NULL, NULL, 0, N'Open', NULL, N'INV-2026-0001', N'Output VAT 10%', N'AR', @By),

 /* DOC 2 - vendor invoice 41,000,000 KHR = 10,000 USD at 4,100 KHR/USD */
 (@Tenant, @H2, @Kh01, 2026, @Doc2, 1, @Ledger, '2026-01-22', 1, N'40', N'S', N'S',
  @GlMaterial, N'500000', NULL, NULL, NULL, @CcProd, @PcSugar, @SegSugar, @Ba1000, @Order1, NULL, NULL, @PlantP100,
  N'KHR', 41000000.0000, N'USD', 10000.0000, N'USD', 10000.0000, 0.000244, @TaxV0, 0, NULL, NULL, NULL,
  NULL, NULL, NULL, 0, N'Open', NULL, N'MES-8842', N'Cane purchase 500 t', N'AP', @By),
 (@Tenant, @H2, @Kh01, 2026, @Doc2, 2, @Ledger, '2026-01-22', 1, N'31', N'H', N'K',
  @GlAp, N'200000', @BpMekong, N'Vendor', NULL, NULL, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, NULL,
  N'KHR', -41000000.0000, N'USD', -10000.0000, N'USD', -10000.0000, 0.000244, NULL, 0, NULL, NULL, NULL,
  @TermsNet30, '2026-01-22', '2026-02-21', 1, N'Open', NULL, N'MES-8842', N'Cane purchase 500 t', N'AP', @By),

 /* DOC 3 - incoming payment 5,000 USD against DOC 1 */
 (@Tenant, @H3, @Kh01, 2026, @Doc3, 1, @Ledger, '2026-02-05', 2, N'40', N'S', N'S',
  @GlBank, N'110000', NULL, NULL, NULL, NULL, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, NULL,
  N'USD', 5000.0000, N'USD', 5000.0000, N'USD', 5000.0000, 1.000000, NULL, 0, NULL, NULL, NULL,
  NULL, NULL, NULL, 0, N'Open', NULL, N'BANK-0207', N'Incoming transfer', N'AR', @By),
 (@Tenant, @H3, @Kh01, 2026, @Doc3, 2, @Ledger, '2026-02-05', 2, N'15', N'H', N'D',
  @GlAr, N'120000', @BpAngkor, N'Customer', NULL, NULL, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, NULL,
  N'USD', -5000.0000, N'USD', -5000.0000, N'USD', -5000.0000, 1.000000, NULL, 0, NULL, NULL, NULL,
  NULL, NULL, NULL, 0, N'Cleared', @Doc1, N'INV-2026-0001', N'Part payment', N'AR', @By),

 /* DOC 4 - intercompany, KH01 side */
 (@Tenant, @H4, @Kh01, 2026, @Doc4, 1, @Ledger, '2026-02-10', 2, N'40', N'S', N'S',
  @GlIcAr, N'125000', @BpSiam, N'Customer', NULL, NULL, @PcSugar, @SegSugar, @Ba1000, NULL, @Th01, N'2000', NULL,
  N'USD', 8000.0000, N'USD', 8000.0000, N'USD', 8000.0000, 1.000000, NULL, 0, NULL, NULL, NULL,
  @TermsNet30, '2026-02-10', '2026-03-12', 1, N'Open', NULL, N'IC-2026-000001', N'IC sale of raw sugar', N'FI', @By),
 (@Tenant, @H4, @Kh01, 2026, @Doc4, 2, @Ledger, '2026-02-10', 2, N'50', N'H', N'S',
  @GlRevenue, N'400000', NULL, NULL, NULL, @CcSales, @PcSugar, @SegSugar, @Ba1000, NULL, @Th01, N'2000', NULL,
  N'USD', -8000.0000, N'USD', -8000.0000, N'USD', -8000.0000, 1.000000, NULL, 0, NULL, NULL, NULL,
  NULL, NULL, NULL, 0, N'Open', NULL, N'IC-2026-000001', N'IC sale of raw sugar', N'FI', @By),

 /* DOC 5 - intercompany, TH01 side: 8,000 USD = 280,000 THB */
 (@Tenant, @H5, @Th01, 2026, @Doc5, 1, @Ledger, '2026-02-10', 2, N'40', N'S', N'S',
  @GlMaterial, N'500000', NULL, NULL, NULL, NULL, @PcTrade, @SegTrade, NULL, NULL, @Kh01, N'1000', NULL,
  N'USD', 8000.0000, N'THB', 280000.0000, N'USD', 8000.0000, 35.000000, NULL, 0, NULL, NULL, NULL,
  NULL, NULL, NULL, 0, N'Open', NULL, N'IC-2026-000001', N'IC purchase of raw sugar', N'FI', @By),
 (@Tenant, @H5, @Th01, 2026, @Doc5, 2, @Ledger, '2026-02-10', 2, N'50', N'H', N'S',
  @GlIcAp, N'225000', @BpSiam, N'Vendor', NULL, NULL, @PcTrade, @SegTrade, NULL, NULL, @Kh01, N'1000', NULL,
  N'USD', -8000.0000, N'THB', -280000.0000, N'USD', -8000.0000, 35.000000, NULL, 0, NULL, NULL, NULL,
  @TermsNet30, '2026-02-10', '2026-03-12', 1, N'Open', NULL, N'IC-2026-000001', N'IC purchase of raw sugar', N'FI', @By),

 /* DOC 6 - asset acquisition from vendor, 120,000 USD */
 (@Tenant, @H6, @Kh01, 2026, @Doc6, 1, @Ledger, '2026-01-15', 1, N'70', N'S', N'A',
  @GlAsset, N'150000', NULL, NULL, @Asset1, @CcProd, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, @PlantP100,
  N'USD', 120000.0000, N'USD', 120000.0000, N'USD', 120000.0000, 1.000000, NULL, 0, NULL, NULL, NULL,
  NULL, NULL, NULL, 0, N'Open', NULL, N'INV-000123', N'Boiler acquisition', N'AA', @By),
 (@Tenant, @H6, @Kh01, 2026, @Doc6, 2, @Ledger, '2026-01-15', 1, N'31', N'H', N'K',
  @GlAp, N'200000', @BpMekong, N'Vendor', NULL, NULL, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, NULL,
  N'USD', -120000.0000, N'USD', -120000.0000, N'USD', -120000.0000, 1.000000, NULL, 0, NULL, NULL, NULL,
  @TermsNet30, '2026-01-15', '2026-02-14', 1, N'Open', NULL, N'MES-8801', N'Boiler acquisition', N'AA', @By),

 /* DOC 7 - depreciation for January: 120,000 / 10 years / 12 months */
 (@Tenant, @H7, @Kh01, 2026, @Doc7, 1, @Ledger, '2026-01-31', 1, N'40', N'S', N'S',
  @GlDepExp, N'520000', NULL, NULL, @Asset1, @CcProd, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, @PlantP100,
  N'USD', 1000.0000, N'USD', 1000.0000, N'USD', 1000.0000, 1.000000, NULL, 0, NULL, NULL, NULL,
  NULL, NULL, NULL, 0, N'Open', NULL, N'100000000001', N'Depreciation January', N'AA', @By),
 (@Tenant, @H7, @Kh01, 2026, @Doc7, 2, @Ledger, '2026-01-31', 1, N'50', N'H', N'A',
  @GlAccDep, N'151000', NULL, NULL, @Asset1, NULL, @PcSugar, @SegSugar, @Ba1000, NULL, NULL, NULL, NULL,
  N'USD', -1000.0000, N'USD', -1000.0000, N'USD', -1000.0000, 1.000000, NULL, 0, NULL, NULL, NULL,
  NULL, NULL, NULL, 0, N'Open', NULL, N'100000000001', N'Depreciation January', N'AA', @By);

INSERT INTO [fin].[JournalEntryTax]
    (TenantId, CompanyCodeId, FiscalYear, DocumentNumber, TaxLineNumber, TaxCodeId,
     ConditionType, TaxRatePercent, TaxBaseAmountInLocalCurrency, TaxAmountInLocalCurrency,
     TaxBaseAmountInDocumentCurrency, TaxAmountInDocumentCurrency, TaxGLAccountId,
     IsOutputTax, ReportingDate, CreatedBy)
VALUES (@Tenant, @Kh01, 2026, @Doc1, 1, @TaxA1, N'MWAS', 10.0000,
        10000.0000, 1000.0000, 10000.0000, 1000.0000, @GlOutputVat, 1, '2026-01-31', @By);

/* ---------------------------------------------------------------------------
   11. Asset and depreciation documents behind DOC 6 and DOC 7
   --------------------------------------------------------------------------- */
INSERT INTO [fin].[AssetTransaction]
    (TenantId, CompanyCodeId, AssetId, FiscalYear, AssetDocumentNumber, LineItemNumber,
     DepreciationAreaId, TransactionTypeId, PostingDate, DocumentDate, AssetValueDate,
     FiscalPeriod, CurrencyCode, TransactionAmount, AmountInLocalCurrency,
     AmountInAreaCurrency, Quantity, PartnerBusinessPartnerId, JournalEntryHeaderId,
     ReferenceDocumentNumber, Text, CreatedBy)
VALUES (@Tenant, @Kh01, @Asset1, 2026, N'AS-2026-0000000001', 1, @Area01, @TtAcquire,
        '2026-01-15', '2026-01-15', '2026-01-15', 1, N'USD', 120000.0000, 120000.0000,
        120000.0000, 1.000000, @BpMekong, @H6, @Doc6, N'Boiler acquisition', @By),
       (@Tenant, @Kh01, @Asset1, 2026, N'AS-2026-0000000001', 2, @Area15, @TtAcquire,
        '2026-01-15', '2026-01-15', '2026-01-15', 1, N'USD', 120000.0000, 120000.0000,
        120000.0000, 1.000000, @BpMekong, @H6, @Doc6, N'Boiler acquisition (tax area)', @By);

INSERT INTO [fin].[DepreciationRun]
    (TenantId, CompanyCodeId, FiscalYear, FiscalPeriod, RunNumber, RunType, IsTestRun,
     Status, AssetsProcessed, TotalDepreciationAmount, ErrorCount, StartedAt, CompletedAt,
     ExecutedBy, CreatedBy)
VALUES (@Tenant, @Kh01, 2026, 1, 1, N'Planned', 0, N'Completed', 1, 1000.0000, 0,
        '2026-01-31T16:00:00', '2026-01-31T16:10:00', N'AFAB', @By);

DECLARE @DepRun bigint =
    (SELECT Id FROM [fin].[DepreciationRun] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND FiscalYear = 2026 AND FiscalPeriod = 1);

INSERT INTO [fin].[DepreciationPosting]
    (TenantId, DepreciationRunId, AssetId, DepreciationAreaId, FiscalYear, FiscalPeriod,
     PostingDate, CurrencyCode, OrdinaryDepreciationAmount, SpecialDepreciationAmount,
     UnplannedDepreciationAmount, ImpairmentAmount, WriteUpAmount, TotalPostedAmount,
     NetBookValueAfterPosting, CostCenterId, ProfitCenterId, JournalEntryHeaderId, CreatedBy)
VALUES (@Tenant, @DepRun, @Asset1, @Area01, 2026, 1, '2026-01-31', N'USD',
        1000.0000, 0.0000, 0.0000, 0.0000, 0.0000, 1000.0000, 119000.0000,
        @CcProd, @PcSugar, @H7, @By);

INSERT INTO [fin].[AssetValue]
    (TenantId, AssetId, DepreciationAreaId, FiscalYear, CurrencyCode,
     AcquisitionValueBroughtForward, AccumulatedDepreciationBroughtForward,
     CurrentYearAcquisitions, CurrentYearRetirements, CurrentYearTransfers,
     OrdinaryDepreciationPosted, SpecialDepreciationPosted, UnplannedDepreciationPosted,
     ImpairmentPosted, WriteUpPosted, PlannedDepreciationYear, NetBookValue,
     ScrapValue, LastUpdatedAt)
VALUES (@Tenant, @Asset1, @Area01, 2026, N'USD', 0.0000, 0.0000, 120000.0000, 0.0000, 0.0000,
        1000.0000, 0.0000, 0.0000, 0.0000, 0.0000, 11000.0000, 119000.0000, 0.0000,
        '2026-01-31T16:10:00'),
       (@Tenant, @Asset1, @Area15, 2026, N'USD', 0.0000, 0.0000, 120000.0000, 0.0000, 0.0000,
        0.0000, 0.0000, 0.0000, 0.0000, 0.0000, 13750.0000, 120000.0000, 0.0000,
        '2026-01-31T16:10:00');

/* ---------------------------------------------------------------------------
   12. Derived data - the posting engine builds these from the journal, so the
        seed does too rather than typing them a second time.
   --------------------------------------------------------------------------- */

/* Open items for every open-item-managed line. */
INSERT INTO [fin].[OpenItem]
    (TenantId, JournalEntryLineId, CompanyCodeId, FiscalYear, DocumentNumber, LineItemNumber,
     AccountType, BusinessPartnerId, GLAccountId, PostingDate, DocumentDate, BaselineDate,
     DueDate, DocumentCurrencyCode, OriginalAmountInDocumentCurrency, OpenAmountInDocumentCurrency,
     OriginalAmountInLocalCurrency, OpenAmountInLocalCurrency, ClearedAmountInDocumentCurrency,
     DebitCreditIndicator, PaymentTermsId, DunningLevel, AssignmentReference,
     ReferenceDocumentNumber, LineItemText, Status, CreatedBy)
SELECT l.TenantId, l.Id, l.CompanyCodeId, l.FiscalYear, l.DocumentNumber, l.LineItemNumber,
       l.AccountType, l.BusinessPartnerId, l.GLAccountId, l.PostingDate, h.DocumentDate,
       l.BaselineDate, l.DueDate, l.DocumentCurrencyCode,
       ABS(l.AmountInDocumentCurrency), ABS(l.AmountInDocumentCurrency),
       ABS(l.AmountInLocalCurrency), ABS(l.AmountInLocalCurrency), 0.0000,
       l.DebitCreditIndicator, l.PaymentTermsId, 0, l.AssignmentReference,
       h.ReferenceDocumentNumber, l.LineItemText, N'Open', @By
FROM   [fin].[JournalEntryLine] AS l
JOIN   [fin].[JournalEntryHeader] AS h ON h.Id = l.JournalEntryHeaderId
WHERE  l.TenantId = @Tenant AND l.IsOpenItemManaged = 1;

/* The February payment clears 5,000 USD of the 11,000 USD invoice. */
DECLARE @InvoiceItem bigint =
    (SELECT Id FROM [fin].[OpenItem]
     WHERE TenantId = @Tenant AND DocumentNumber = @Doc1 AND LineItemNumber = 1);

INSERT INTO [fin].[ClearingDocument]
    (TenantId, CompanyCodeId, FiscalYear, ClearingDocumentNumber, ClearingDate, PostingDate,
     ClearingType, JournalEntryHeaderId, CurrencyCode, TotalClearedAmount, DifferenceAmount,
     DifferenceHandling, CreatedBy)
VALUES (@Tenant, @Kh01, 2026, @Doc3, '2026-02-05', '2026-02-05', N'IncomingPayment',
        @H3, N'USD', 5000.0000, 0.0000, N'PartialPayment', @By);

DECLARE @Clearing bigint =
    (SELECT Id FROM [fin].[ClearingDocument] WHERE TenantId = @Tenant AND CompanyCodeId = @Kh01 AND FiscalYear = 2026 AND ClearingDocumentNumber = @Doc3);

INSERT INTO [fin].[ClearingItem]
    (TenantId, ClearingDocumentId, OpenItemId, ClearedAmountInDocumentCurrency,
     ClearedAmountInLocalCurrency, IsPartialClearing, CreatedBy)
VALUES (@Tenant, @Clearing, @InvoiceItem, 5000.0000, 5000.0000, 1, @By);

UPDATE [fin].[OpenItem]
SET    OpenAmountInDocumentCurrency = 6000.0000,
       OpenAmountInLocalCurrency    = 6000.0000,
       ClearedAmountInDocumentCurrency = 5000.0000,
       Status = N'PartiallyCleared'
WHERE  Id = @InvoiceItem;

UPDATE [fin].[JournalEntryLine]
SET    ClearingStatus = N'PartiallyCleared',
       ClearingDocumentNumber = @Doc3,
       ClearingDate = '2026-02-05'
WHERE  TenantId = @Tenant AND DocumentNumber = @Doc1 AND LineItemNumber = 1;

/* Controlling documents for every journal line that carries a cost element and
   a cost centre - the FI/CO integration the design calls for. */
INSERT INTO [co].[ControllingPosting]
    (TenantId, ControllingAreaId, ControllingDocumentNumber, LineItemNumber, FiscalYear,
     FiscalPeriod, PostingDate, PlanVersion, IsPlan, ValueType, ObjectType, ObjectId,
     CostElementId, CompanyCodeId, ProfitCenterId, SegmentId, DebitCreditIndicator,
     CurrencyCode, AmountInTransactionCurrency, AmountInControllingAreaCurrency,
     AmountInCompanyCodeCurrency, TransactionType, ReferenceDocumentNumber,
     JournalEntryHeaderId, LineItemText, CreatedBy)
SELECT @Tenant, @Coar,
       N'CO-2026-' + RIGHT(N'0000000000' + CAST(ROW_NUMBER() OVER (ORDER BY l.Id) AS nvarchar(10)), 10),
       1, l.FiscalYear, l.FiscalPeriod, l.PostingDate, N'000', 0, N'04',
       N'CostCenter', l.CostCenterId, ce.Id, l.CompanyCodeId, l.ProfitCenterId, l.SegmentId,
       l.DebitCreditIndicator, l.DocumentCurrencyCode, l.AmountInDocumentCurrency,
       l.AmountInLocalCurrency, l.AmountInLocalCurrency, N'Primary', l.DocumentNumber,
       l.JournalEntryHeaderId, l.LineItemText, @By
FROM   [fin].[JournalEntryLine] AS l
JOIN   [co].[CostElement] AS ce
       ON  ce.TenantId = l.TenantId
       AND ce.GLAccountId = l.GLAccountId
       AND ce.IsPrimary = 1
WHERE  l.TenantId = @Tenant AND l.CostCenterId IS NOT NULL;

/* Period balances in local currency, aggregated from the journal. */
INSERT INTO [fin].[AccountBalance]
    (TenantId, LedgerId, CompanyCodeId, FiscalYear, FiscalPeriod, GLAccountId,
     BusinessPartnerId, ProfitCenterId, SegmentId, FunctionalAreaId, BusinessAreaId,
     CurrencyType, CurrencyCode, DebitTotal, CreditTotal, PeriodBalance,
     CumulativeBalance, LastUpdatedAt)
SELECT l.TenantId, l.LedgerId, l.CompanyCodeId, l.FiscalYear, l.FiscalPeriod, l.GLAccountId,
       NULL, NULL, NULL, NULL, NULL,
       N'10', MIN(l.LocalCurrencyCode),
       SUM(CASE WHEN l.AmountInLocalCurrency > 0 THEN l.AmountInLocalCurrency ELSE 0 END),
       SUM(CASE WHEN l.AmountInLocalCurrency < 0 THEN -l.AmountInLocalCurrency ELSE 0 END),
       SUM(l.AmountInLocalCurrency),
       SUM(l.AmountInLocalCurrency),
       SYSUTCDATETIME()
FROM   [fin].[JournalEntryLine] AS l
WHERE  l.TenantId = @Tenant
GROUP BY l.TenantId, l.LedgerId, l.CompanyCodeId, l.FiscalYear, l.FiscalPeriod, l.GLAccountId;

/* Number range levels follow the numbers this script issued. */
UPDATE i
SET    i.CurrentNumber = 1
FROM   [cfg].[NumberRangeInterval] AS i
WHERE  i.TenantId = @Tenant AND i.NumberRangeObjectId = @NroDoc
       AND i.NumberRangeCode IN (N'02', N'03', N'04');

UPDATE i
SET    i.CurrentNumber = 2
FROM   [cfg].[NumberRangeInterval] AS i
WHERE  i.TenantId = @Tenant AND i.NumberRangeObjectId = @NroDoc
       AND i.NumberRangeCode IN (N'01', N'06');

/* ---------------------------------------------------------------------------
   13. Users, roles and transaction codes
   --------------------------------------------------------------------------- */
INSERT INTO [sec].[User]
    (TenantId, UserName, EmployeeNumber, BusinessPartnerId, FirstName, LastName, DisplayName,
     Email, UserType, LanguageCode, TimeZoneId, DefaultCompanyCodeId, DefaultControllingAreaId,
     DefaultCurrencyCode, DefaultTransactionCode, Status, FailedLoginCount, IsMfaEnabled,
     ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, N'kss.admin', NULL, NULL, N'System', N'Administrator', N'System Administrator',
        N'admin@kss.example', N'Dialog', N'EN', N'Asia/Phnom_Penh', @Kh01, @Coar, N'USD', N'SPRO',
        N'Active', 0, 1, @Open, @Never, @By),
       (@Tenant, N'kss.accountant', N'10001', @BpSopheak, N'Sopheak', N'Chan', N'Sopheak Chan',
        N'sopheak.chan@kss.example', N'Dialog', N'EN', N'Asia/Phnom_Penh', @Kh01, @Coar, N'USD', N'FB50',
        N'Active', 0, 0, @Open, @Never, @By),
       (@Tenant, N'kss.approver', N'10002', NULL, N'Dara', N'Sok', N'Dara Sok',
        N'dara.sok@kss.example', N'Dialog', N'EN', N'Asia/Phnom_Penh', @Kh01, @Coar, N'USD', N'FB03',
        N'Active', 0, 1, @Open, @Never, @By),
       (@Tenant, N'kss.approver2', N'10003', NULL, N'Vanna', N'Ly', N'Vanna Ly',
        N'vanna.ly@kss.example', N'Dialog', N'EN', N'Asia/Phnom_Penh', @Kh01, @Coar, N'USD', N'FB03',
        N'Active', 0, 1, @Open, @Never, @By),
       (@Tenant, N'svc.integration', NULL, NULL, NULL, NULL, N'Integration service account',
        N'integration@kss.example', N'Integration', N'EN', N'UTC', @Kh01, @Coar, N'USD', NULL,
        N'Active', 0, 0, @Open, @Never, @By);

DECLARE @UserAdmin bigint = (SELECT Id FROM [sec].[User] WHERE TenantId = @Tenant AND UserName = N'kss.admin');
DECLARE @UserAccountant bigint = (SELECT Id FROM [sec].[User] WHERE TenantId = @Tenant AND UserName = N'kss.accountant');
DECLARE @UserApprover bigint = (SELECT Id FROM [sec].[User] WHERE TenantId = @Tenant AND UserName = N'kss.approver');
DECLARE @UserApprover2 bigint = (SELECT Id FROM [sec].[User] WHERE TenantId = @Tenant AND UserName = N'kss.approver2');

INSERT INTO [sec].[Role]
    (TenantId, RoleCode, Name, Description, RoleType, IsCriticalRole, IsSystemRole,
     ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, N'SYS_ADMIN',     N'System administrator', N'Configuration and security', N'Single', 1, 1, @Open, @Never, @By),
       (@Tenant, N'FI_ACCOUNTANT', N'Financial accountant', N'Enters and posts documents', N'Single', 0, 1, @Open, @Never, @By),
       (@Tenant, N'FI_APPROVER',   N'Financial approver',   N'Approves parked documents',  N'Single', 1, 1, @Open, @Never, @By),
       (@Tenant, N'FI_DIRECTOR',   N'Finance director',     N'Second approval above the threshold', N'Single', 1, 1, @Open, @Never, @By);

DECLARE @RoleAdmin bigint = (SELECT Id FROM [sec].[Role] WHERE TenantId = @Tenant AND RoleCode = N'SYS_ADMIN');
DECLARE @RoleAccountant bigint = (SELECT Id FROM [sec].[Role] WHERE TenantId = @Tenant AND RoleCode = N'FI_ACCOUNTANT');
DECLARE @RoleApprover bigint = (SELECT Id FROM [sec].[Role] WHERE TenantId = @Tenant AND RoleCode = N'FI_APPROVER');
DECLARE @RoleDirector bigint = (SELECT Id FROM [sec].[Role] WHERE TenantId = @Tenant AND RoleCode = N'FI_DIRECTOR');

INSERT INTO [sec].[Permission]
    (TenantId, PermissionCode, Name, Module, ObjectType, Action, IsCritical, CreatedBy)
VALUES (@Tenant, N'Finance.JournalEntry.Create',  N'Create journal entry',  N'Finance', N'JournalEntry', N'Create',  0, @By),
       (@Tenant, N'Finance.JournalEntry.Post',    N'Post journal entry',    N'Finance', N'JournalEntry', N'Post',    1, @By),
       (@Tenant, N'Finance.JournalEntry.Approve', N'Approve journal entry', N'Finance', N'JournalEntry', N'Approve', 1, @By),
       (@Tenant, N'Finance.JournalEntry.Reverse', N'Reverse journal entry', N'Finance', N'JournalEntry', N'Reverse', 1, @By),
       (@Tenant, N'Finance.Report.Read',          N'Display reports',       N'Finance', N'Report',       N'Read',    0, @By),
       (@Tenant, N'Finance.Report.Export',        N'Export reports',        N'Finance', N'Report',       N'Export',  1, @By),
       (@Tenant, N'Master.BusinessPartner.Read',   N'Display partners',    N'Master',  N'BusinessPartner', N'Read',   0, @By),
       (@Tenant, N'Master.BusinessPartner.Update', N'Maintain partners',    N'Master',  N'BusinessPartner', N'Update', 0, @By),
       (@Tenant, N'Admin.Dictionary.Read',        N'Display dictionary',    N'Admin',   N'DictionaryObject', N'Read',   0, @By),
       (@Tenant, N'Admin.Dictionary.Update',      N'Maintain dictionary',   N'Admin',   N'DictionaryObject', N'Update', 1, @By),
       (@Tenant, N'Admin.TableBrowser.Read',      N'Browse tables',         N'Admin',   N'TableBrowser', N'Read',    1, @By),
       (@Tenant, N'Admin.TableBrowser.Export',    N'Export browser results', N'Admin',  N'TableBrowser', N'Export',  1, @By),
       (@Tenant, N'Assets.Asset.Read',            N'Display assets',        N'Assets',  N'Asset',        N'Read',    0, @By),
       (@Tenant, N'Assets.Asset.Post',            N'Post asset movements',  N'Assets',  N'Asset',        N'Post',    1, @By);

INSERT INTO [sec].[RolePermission] (TenantId, RoleId, PermissionId, IsGranted, CreatedBy)
SELECT @Tenant, @RoleAdmin, p.Id, 1, @By FROM [sec].[Permission] AS p WHERE p.TenantId = @Tenant;

INSERT INTO [sec].[RolePermission] (TenantId, RoleId, PermissionId, IsGranted, CreatedBy)
SELECT @Tenant, @RoleAccountant, p.Id, 1, @By
FROM   [sec].[Permission] AS p
WHERE  p.TenantId = @Tenant
       AND p.PermissionCode IN (N'Finance.JournalEntry.Create', N'Finance.JournalEntry.Post',
                                N'Finance.Report.Read', N'Master.BusinessPartner.Read',
                                N'Master.BusinessPartner.Update', N'Assets.Asset.Read',
                                N'Assets.Asset.Post');

INSERT INTO [sec].[RolePermission] (TenantId, RoleId, PermissionId, IsGranted, CreatedBy)
SELECT @Tenant, @RoleApprover, p.Id, 1, @By
FROM   [sec].[Permission] AS p
WHERE  p.TenantId = @Tenant
       AND p.PermissionCode IN (N'Finance.JournalEntry.Approve', N'Finance.JournalEntry.Reverse',
                                N'Finance.Report.Read', N'Finance.Report.Export',
                                N'Master.BusinessPartner.Read');

INSERT INTO [sec].[UserRole]
    (TenantId, UserId, RoleId, AssignedBy, AssignedAt, ApprovedBy, ApprovedAt,
     AssignmentReason, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @UserAdmin,      @RoleAdmin,      @By, '2026-01-01T00:00:00', @By, '2026-01-01T00:00:00', N'Initial setup', @Open, @Never, @By),
       (@Tenant, @UserAdmin,      @RoleDirector,   @By, '2026-01-01T00:00:00', @By, '2026-01-01T00:00:00', N'Second approval step', @Open, @Never, @By),
       (@Tenant, @UserAccountant, @RoleAccountant, @By, '2026-01-01T00:00:00', NULL, NULL,                 N'Initial setup', @Open, @Never, @By),
       (@Tenant, @UserApprover,   @RoleApprover,   @By, '2026-01-01T00:00:00', @By, '2026-01-01T00:00:00', N'Initial setup', @Open, @Never, @By),
       (@Tenant, @UserApprover2,  @RoleApprover,   @By, '2026-01-01T00:00:00', @By, '2026-01-01T00:00:00', N'Second approver so maker-checker has a fallback', @Open, @Never, @By);

INSERT INTO [sec].[UserCompanyCode]
    (TenantId, UserId, CompanyCodeId, IsDefault, AccessLevel, ValidFrom, ValidTo, CreatedBy)
VALUES (@Tenant, @UserAdmin,      @Kh01, 1, N'Approve', @Open, @Never, @By),
       (@Tenant, @UserAdmin,      @Kh02, 0, N'Approve', @Open, @Never, @By),
       (@Tenant, @UserAdmin,      @Th01, 0, N'Approve', @Open, @Never, @By),
       (@Tenant, @UserAccountant, @Kh01, 1, N'Post',    @Open, @Never, @By),
       (@Tenant, @UserAccountant, @Kh02, 0, N'Post',    @Open, @Never, @By),
       (@Tenant, @UserApprover,   @Kh01, 1, N'Approve', @Open, @Never, @By);

INSERT INTO [sec].[TransactionCode]
    (TenantId, TransactionCode, Name, Module, RoutePath, Category, IsFavoriteEligible, CreatedBy)
VALUES (@Tenant, N'SPRO',  N'Configuration',        N'Admin',    N'/configuration',                 N'Configuration', 1, @By),
       (@Tenant, N'OBY6',  N'Company code',         N'Org',      N'/org/company-codes',             N'Configuration', 1, @By),
       (@Tenant, N'OB52',  N'Posting periods',      N'Finance',  N'/configuration/posting-periods', N'Configuration', 1, @By),
       (@Tenant, N'FS00',  N'G/L account',          N'Finance',  N'/master/gl-accounts',            N'Master data',   1, @By),
       (@Tenant, N'BP',    N'Business partner',     N'Master',   N'/master/business-partners',      N'Master data',   1, @By),
       (@Tenant, N'FB50',  N'Journal entry',        N'Finance',  N'/finance/journal-entry/new',     N'Postings',      1, @By),
       (@Tenant, N'FB03',  N'Display document',     N'Finance',  N'/finance/documents',             N'Postings',      1, @By),
       (@Tenant, N'FB08',  N'Reverse document',     N'Finance',  N'/finance/documents/reverse',     N'Postings',      1, @By),
       (@Tenant, N'FBL5N', N'Customer line items',  N'Finance',  N'/finance/customers/line-items',  N'Reporting',     1, @By),
       (@Tenant, N'FBL1N', N'Vendor line items',    N'Finance',  N'/finance/vendors/line-items',    N'Reporting',     1, @By),
       (@Tenant, N'F-28',  N'Incoming payment',     N'Finance',  N'/finance/payments/incoming',     N'Postings',      1, @By),
       (@Tenant, N'AS03',  N'Display asset',        N'Assets',   N'/assets',                        N'Master data',   1, @By),
       (@Tenant, N'AFAB',  N'Depreciation run',     N'Assets',   N'/assets/depreciation-run',       N'Period close',  1, @By),
       (@Tenant, N'KS03',  N'Display cost centre',  N'Controlling', N'/controlling/cost-centers',   N'Master data',   1, @By),
       (@Tenant, N'KO01',  N'Internal order',       N'Controlling', N'/controlling/internal-orders', N'Master data',  1, @By),
       (@Tenant, N'SE11',  N'Data dictionary',      N'Admin',    N'/dictionary',                    N'Tools',         1, @By),
       (@Tenant, N'SE16N', N'Table browser',        N'Admin',    N'/table-browser',                 N'Tools',         1, @By);

INSERT INTO [sec].[RoleTransactionCode] (TenantId, RoleId, TransactionCodeId, IsGranted, CreatedBy)
SELECT @Tenant, @RoleAdmin, t.Id, 1, @By FROM [sec].[TransactionCode] AS t WHERE t.TenantId = @Tenant;

INSERT INTO [sec].[RoleTransactionCode] (TenantId, RoleId, TransactionCodeId, IsGranted, CreatedBy)
SELECT @Tenant, @RoleAccountant, t.Id, 1, @By
FROM   [sec].[TransactionCode] AS t
WHERE  t.TenantId = @Tenant
       AND t.TransactionCode IN (N'FS00', N'BP', N'FB50', N'FB03', N'FBL5N', N'FBL1N', N'F-28', N'AS03', N'KS03');

INSERT INTO [sec].[SegregationOfDutiesRule]
    (TenantId, RuleCode, Name, Description, RiskLevel, ConflictType, FirstObjectCode,
     SecondObjectCode, EnforcementMode, IsActive, CreatedBy)
VALUES (@Tenant, N'SOD_FI_01', N'Post and approve the same document',
        N'A user who can post must not also approve - maker-checker.', N'High', N'PermissionPair',
        N'Finance.JournalEntry.Post', N'Finance.JournalEntry.Approve', N'Block', 1, @By),
       (@Tenant, N'SOD_FI_02', N'Maintain partner bank details and run payments',
        N'Changing bank details and releasing payments is a fraud path.', N'Critical', N'PermissionPair',
        N'Master.BusinessPartner.Update', N'Finance.JournalEntry.Post', N'WarnAndLog', 1, @By);

/* ---------------------------------------------------------------------------
   12. Approval workflow

   Without this the approval feature has no configuration and never triggers,
   so a document above any threshold posts straight through. One definition,
   one rule at 10 000 USD, two sequential steps: the approver, then the
   director. Maker-checker is on, so the person who submitted a document is
   never given a task on it.
   --------------------------------------------------------------------------- */
INSERT INTO [wf].[WorkflowDefinition]
    (TenantId, WorkflowCode, Version, Name, ObjectType, ApprovalMode,
     IsMakerCheckerEnforced, AllowDelegation, AllowResubmission, EscalationHours,
     ReminderHours, IsActive, EffectiveFrom, CreatedBy)
VALUES (@Tenant, N'JE_APPROVAL', 1, N'Journal entry approval', N'JournalEntry',
        N'Sequential', 1, 1, 1, 48, 24, 1, @Open, @By);

DECLARE @Workflow bigint = (SELECT Id FROM [wf].[WorkflowDefinition]
                            WHERE TenantId = @Tenant AND WorkflowCode = N'JE_APPROVAL' AND Version = 1);

INSERT INTO [wf].[WorkflowRule]
    (TenantId, WorkflowDefinitionId, RuleSequence, Name, CompanyCodeId, CurrencyCode,
     MinimumAmount, SourceModule, IsActive, CreatedBy)
VALUES (@Tenant, @Workflow, 1, N'Journal entries of 10 000 USD or more',
        @Kh01, N'USD', 10000.0000, N'FI', 1, @By);

INSERT INTO [wf].[WorkflowStep]
    (TenantId, WorkflowDefinitionId, StepNumber, Name, StepType,
     ApproverDeterminationType, ApproverRoleId, RequiredApprovals, IsParallel,
     IsOptional, EscalationHours, OnRejectAction, CreatedBy)
VALUES (@Tenant, @Workflow, 1, N'Financial approver', N'Approval', N'Role',
        @RoleApprover, 1, 0, 0, 48, N'Reject', @By),
       (@Tenant, @Workflow, 2, N'Finance director', N'Approval', N'Role',
        @RoleDirector, 1, 0, 0, 48, N'Reject', @By);

COMMIT TRANSACTION;

PRINT 'Sample data loaded:';
PRINT '  2 companies, 3 company codes, 1 controlling area, 1 shared chart of accounts';
PRINT '  5 business partners (customer, vendor, dual role, employee, intercompany)';
PRINT '  3 cost centres, 2 profit centres, 1 internal order, 1 asset';
PRINT '  7 posted documents including an intercompany pair and a KHR invoice';
PRINT '  1 approval workflow: two sequential steps above 10 000 USD';

END TRY
BEGIN CATCH
    IF @@TRANCOUNT > 0
        ROLLBACK TRANSACTION;
    THROW;
END CATCH;
GO
