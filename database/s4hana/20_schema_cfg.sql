/* ============================================================================
   S/4HANA-inspired ERP - schema [cfg]
   Configuration, data dictionary, custom objects (75 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* cfg.AccountDeterminationRule - Automatic account determination (tax, gain/loss, clearing, retained earnings) (reference: T030) */
IF OBJECT_ID(N'cfg.AccountDeterminationRule', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[AccountDeterminationRule]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [ChartOfAccountsId] bigint NOT NULL,                                                                          -- Chart of accounts
        [TransactionKey]    nvarchar(4) NOT NULL,                                                                     -- Key, e.g. MWS, VST, KDF, BSX, BIL
        [AccountModifier]   nvarchar(4) NULL,                                                                         -- Additional differentiation
        [CompanyCodeId]     bigint NULL,                                                                              -- Company code (null = all)
        [CurrencyCode]      nvarchar(5) NULL,                                                                         -- Currency-specific rule
        [DebitGLAccountId]  bigint NULL,                                                                              -- Debit account
        [CreditGLAccountId] bigint NULL,                                                                              -- Credit account
        [Description]       nvarchar(60) NULL,                                                                        -- Purpose of the rule
        [ValidFrom]         date NOT NULL,                                                                            -- First day the record is valid
        [ValidTo]           date NOT NULL,                                                                            -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_cfg_AccountDeterminationRule_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_cfg_AccountDeterminationRule_IsActive] DEFAULT (1),           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_AccountDeterminationRule] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_AccountDeterminationRule] UNIQUE ([TenantId], [ChartOfAccountsId], [TransactionKey], [AccountModifier], [CompanyCodeId], [CurrencyCode])
    );
END
GO

/* cfg.AccountGroup - G/L account group - controls number interval and field status (reference: T077S) */
IF OBJECT_ID(N'cfg.AccountGroup', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[AccountGroup]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [ChartOfAccountsId]  bigint NOT NULL,                                                                         -- Owning chart of accounts
        [AccountGroup]       nvarchar(4) NOT NULL,                                                                    -- Account group key
        [Name]               nvarchar(40) NOT NULL,                                                                   -- Description
        [FromAccount]        nvarchar(10) NOT NULL,                                                                   -- Lower limit of the number interval
        [ToAccount]          nvarchar(10) NOT NULL,                                                                   -- Upper limit of the number interval
        [FieldStatusGroupId] bigint NULL,                                                                             -- Field status for master-data maintenance
        [AppliesTo]          nvarchar(10) NOT NULL,                                                                   -- GL, Customer, Vendor
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_AccountGroup_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_AccountGroup_IsActive] DEFAULT (1),                      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_AccountGroup] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_AccountGroup] UNIQUE ([TenantId], [ChartOfAccountsId], [AccountGroup])
    );
END
GO

/* cfg.AccountingPrinciple - Accounting principle (IFRS, local GAAP) (reference: T001/FAGL accounting principle) */
IF OBJECT_ID(N'cfg.AccountingPrinciple', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[AccountingPrinciple]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [AccountingPrinciple] nvarchar(4) NOT NULL,                                                                   -- Key, e.g. IFRS, LGAP
        [Name]                nvarchar(60) NOT NULL,                                                                  -- Description
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_cfg_AccountingPrinciple_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_cfg_AccountingPrinciple_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_AccountingPrinciple] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_AccountingPrinciple] UNIQUE ([TenantId], [AccountingPrinciple])
    );
END
GO

/* cfg.BrowserQueryLog - Audit trail of every table-browser query */
IF OBJECT_ID(N'cfg.BrowserQueryLog', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[BrowserQueryLog]
    (
        [Id]              bigint IDENTITY(1,1) NOT NULL,                                                              -- Surrogate key
        [TenantId]        int NOT NULL,                                                                               -- Owning tenant - every query is filtered by it
        [UserId]          bigint NOT NULL,                                                                            -- Executing user
        [ExecutedAt]      datetime2(3) NOT NULL,                                                                      -- Execution timestamp (UTC)
        [SchemaName]      nvarchar(20) NOT NULL,                                                                      -- Schema queried
        [ObjectName]      nvarchar(64) NOT NULL,                                                                      -- Table or view queried
        [SelectedFields]  nvarchar(max) NULL,                                                                         -- Fields returned
        [FilterJson]      nvarchar(max) NULL,                                                                         -- Parameterised filter (values redacted where masked)
        [RowsReturned]    int NOT NULL,                                                                               -- Number of rows returned
        [RowLimitApplied] int NOT NULL,                                                                               -- Row cap in force
        [WasExported]     bit NOT NULL CONSTRAINT [DF_cfg_BrowserQueryLog_WasExported] DEFAULT (0),                   -- Result exported
        [ExportFormat]    nvarchar(10) NULL,                                                                          -- XLSX, CSV, PDF
        [DurationMs]      int NOT NULL,                                                                               -- Execution time in milliseconds
        [WasTruncated]    bit NOT NULL CONSTRAINT [DF_cfg_BrowserQueryLog_WasTruncated] DEFAULT (0),                  -- Result cut off by the row limit
        [CorrelationId]   uniqueidentifier NULL,                                                                      -- Request correlation id
        [ClientIpAddress] nvarchar(45) NULL,                                                                          -- Request origin
        CONSTRAINT [PK_cfg_BrowserQueryLog] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* cfg.BrowserVariant - Saved selection variant / layout of the table browser */
IF OBJECT_ID(N'cfg.BrowserVariant', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[BrowserVariant]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [VariantName]        nvarchar(40) NOT NULL,                                                                   -- Variant name
        [SchemaName]         nvarchar(20) NOT NULL,                                                                   -- Schema of the queried object
        [ObjectName]         nvarchar(64) NOT NULL,                                                                   -- Table or view queried
        [OwnerUserId]        bigint NOT NULL,                                                                         -- Owning user
        [IsShared]           bit NOT NULL CONSTRAINT [DF_cfg_BrowserVariant_IsShared] DEFAULT (0),                    -- Visible to other users
        [IsDefault]          bit NOT NULL CONSTRAINT [DF_cfg_BrowserVariant_IsDefault] DEFAULT (0),                   -- Opened by default
        [Description]        nvarchar(255) NULL,                                                                      -- Description
        [SortDefinition]     nvarchar(255) NULL,                                                                      -- Sort fields and directions
        [PageSize]           int NOT NULL,                                                                            -- Rows per page
        [ShowTechnicalNames] bit NOT NULL CONSTRAINT [DF_cfg_BrowserVariant_ShowTechnicalNames] DEFAULT (0),          -- Display field names instead of labels
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_BrowserVariant_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_BrowserVariant_IsActive] DEFAULT (1),                    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_BrowserVariant] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_BrowserVariant] UNIQUE ([TenantId], [VariantName], [SchemaName], [ObjectName], [OwnerUserId])
    );
END
GO

/* cfg.BrowserVariantField - Selected output field of a variant */
IF OBJECT_ID(N'cfg.BrowserVariantField', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[BrowserVariantField]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [BrowserVariantId]  bigint NOT NULL,                                                                          -- Owning variant
        [FieldName]         nvarchar(64) NOT NULL,                                                                    -- Field
        [DisplayOrder]      int NOT NULL,                                                                             -- Column order
        [ColumnWidth]       int NULL,                                                                                 -- Column width in pixels
        [IsVisible]         bit NOT NULL CONSTRAINT [DF_cfg_BrowserVariantField_IsVisible] DEFAULT (0),               -- Column displayed
        [AggregateFunction] nvarchar(10) NULL,                                                                        -- SUM, COUNT, AVG for totals rows
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_cfg_BrowserVariantField_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_cfg_BrowserVariantField_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_BrowserVariantField] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_BrowserVariantField] UNIQUE ([TenantId], [BrowserVariantId], [FieldName])
    );
END
GO

/* cfg.BrowserVariantFilter - Selection criterion of a variant (range table) */
IF OBJECT_ID(N'cfg.BrowserVariantFilter', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[BrowserVariantFilter]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [BrowserVariantId] bigint NOT NULL,                                                                           -- Owning variant
        [FieldName]        nvarchar(64) NOT NULL,                                                                     -- Filtered field
        [LineNumber]       int NOT NULL,                                                                              -- Criterion line
        [SignIndicator]    nvarchar(1) NOT NULL,                                                                      -- I include, E exclude
        [Operator]         nvarchar(10) NOT NULL,                                                                     -- EQ, NE, GT, GE, LT, LE, BT, CP, SW, EW, NULL, NN, IN, NI
        [LowValue]         nvarchar(255) NULL,                                                                        -- Value / interval start
        [HighValue]        nvarchar(255) NULL,                                                                        -- Interval end
        [ValueList]        nvarchar(max) NULL,                                                                        -- Values for IN / NI
        [CreatedAt]        datetime2(3) NOT NULL CONSTRAINT [DF_cfg_BrowserVariantFilter_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]        nvarchar(64) NOT NULL,                                                                     -- Creating user name
        [ModifiedAt]       datetime2(3) NULL,                                                                         -- Last change timestamp (UTC)
        [ModifiedBy]       nvarchar(64) NULL,                                                                         -- Last changing user name
        [RowVersion]       rowversion NOT NULL,                                                                       -- Optimistic concurrency token
        [IsActive]         bit NOT NULL CONSTRAINT [DF_cfg_BrowserVariantFilter_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_BrowserVariantFilter] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_BrowserVariantFilter] UNIQUE ([TenantId], [BrowserVariantId], [FieldName], [LineNumber])
    );
END
GO

/* cfg.ChartOfAccounts - Chart of accounts (reference: T004) */
IF OBJECT_ID(N'cfg.ChartOfAccounts', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[ChartOfAccounts]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [ChartOfAccounts]        nvarchar(4) NOT NULL,                                                                -- Chart of accounts key
        [Name]                   nvarchar(60) NOT NULL,                                                               -- Description
        [MaintenanceLanguage]    nvarchar(2) NOT NULL,                                                                -- Language of account names
        [AccountNumberLength]    tinyint NOT NULL,                                                                    -- Length of G/L account numbers (max 10)
        [GroupChartOfAccountsId] bigint NULL,                                                                         -- Group chart for consolidation
        [IsBlockedForPosting]    bit NOT NULL CONSTRAINT [DF_cfg_ChartOfAccounts_IsBlockedForPosting] DEFAULT (0),    -- Chart blocked
        [ChartType]              nvarchar(20) NOT NULL,                                                               -- Operational, Group, Country
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_ChartOfAccounts_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_ChartOfAccounts_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_ChartOfAccounts] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_ChartOfAccounts] UNIQUE ([TenantId], [ChartOfAccounts])
    );
END
GO

/* cfg.CorrespondenceForm - Print form for invoices, statements, dunning letters (reference: T048) */
IF OBJECT_ID(N'cfg.CorrespondenceForm', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[CorrespondenceForm]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [FormCode]           nvarchar(20) NOT NULL,                                                                   -- Form key
        [LanguageCode]       nvarchar(2) NOT NULL,                                                                    -- Form language
        [Name]               nvarchar(60) NOT NULL,                                                                   -- Description
        [CorrespondenceType] nvarchar(20) NOT NULL,                                                                   -- Invoice, Statement, Dunning, PaymentAdvice
        [TemplateBody]       nvarchar(max) NULL,                                                                      -- Template source (HTML/Razor)
        [OutputFormat]       nvarchar(10) NOT NULL,                                                                   -- PDF, HTML, CSV
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_CorrespondenceForm_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_CorrespondenceForm_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_CorrespondenceForm] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_CorrespondenceForm] UNIQUE ([TenantId], [FormCode], [LanguageCode])
    );
END
GO

/* cfg.Country - Country (reference: T005) */
IF OBJECT_ID(N'cfg.Country', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[Country]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [CountryCode]      nvarchar(3) NOT NULL,                                                                      -- Country key, e.g. KH, TH, US
        [IsoCode2]         nvarchar(2) NOT NULL,                                                                      -- ISO 3166-1 alpha-2
        [IsoCode3]         nvarchar(3) NULL,                                                                          -- ISO 3166-1 alpha-3
        [Name]             nvarchar(60) NOT NULL,                                                                     -- Country name
        [CurrencyCode]     nvarchar(5) NULL,                                                                          -- Default currency
        [LanguageCode]     nvarchar(2) NULL,                                                                          -- Default language
        [AddressFormatKey] nvarchar(4) NULL,                                                                          -- Address layout key
        [TaxNumberRule]    nvarchar(60) NULL,                                                                         -- Validation regex for the tax number
        [IsEuMember]       bit NOT NULL CONSTRAINT [DF_cfg_Country_IsEuMember] DEFAULT (0),                           -- EU member (intra-community tax handling)
        [DateFormat]       nvarchar(10) NULL,                                                                         -- Preferred date format
        [DecimalSeparator] nvarchar(1) NULL,                                                                          -- Decimal separator for output
        [CreatedAt]        datetime2(3) NOT NULL CONSTRAINT [DF_cfg_Country_CreatedAt] DEFAULT (SYSUTCDATETIME()),    -- Creation timestamp (UTC)
        [CreatedBy]        nvarchar(64) NOT NULL,                                                                     -- Creating user name
        [ModifiedAt]       datetime2(3) NULL,                                                                         -- Last change timestamp (UTC)
        [ModifiedBy]       nvarchar(64) NULL,                                                                         -- Last changing user name
        [RowVersion]       rowversion NOT NULL,                                                                       -- Optimistic concurrency token
        [IsActive]         bit NOT NULL CONSTRAINT [DF_cfg_Country_IsActive] DEFAULT (1),                             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_Country] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_Country] UNIQUE ([TenantId], [CountryCode])
    );
END
GO

/* cfg.Currency - Currency master (reference: TCURC) */
IF OBJECT_ID(N'cfg.Currency', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[Currency]
    (
        [Id]            bigint IDENTITY(1,1) NOT NULL,                                                                -- Surrogate key
        [TenantId]      int NOT NULL,                                                                                 -- Owning tenant - every query is filtered by it
        [CurrencyCode]  nvarchar(5) NOT NULL,                                                                         -- ISO currency key, e.g. USD, KHR, THB
        [IsoCode]       nvarchar(3) NOT NULL,                                                                         -- ISO 4217 code
        [NumericCode]   nvarchar(3) NULL,                                                                             -- ISO numeric code
        [Name]          nvarchar(40) NOT NULL,                                                                        -- Currency name
        [ShortText]     nvarchar(15) NULL,                                                                            -- Short name for reports
        [DecimalPlaces] tinyint NOT NULL,                                                                             -- Decimals used in display and rounding
        [Symbol]        nvarchar(5) NULL,                                                                             -- Display symbol
        [ValidFrom]     date NULL,                                                                                    -- Introduction date
        [IsBlocked]     bit NOT NULL CONSTRAINT [DF_cfg_Currency_IsBlocked] DEFAULT (0),                              -- Blocked for new transactions
        [CreatedAt]     datetime2(3) NOT NULL CONSTRAINT [DF_cfg_Currency_CreatedAt] DEFAULT (SYSUTCDATETIME()),      -- Creation timestamp (UTC)
        [CreatedBy]     nvarchar(64) NOT NULL,                                                                        -- Creating user name
        [ModifiedAt]    datetime2(3) NULL,                                                                            -- Last change timestamp (UTC)
        [ModifiedBy]    nvarchar(64) NULL,                                                                            -- Last changing user name
        [RowVersion]    rowversion NOT NULL,                                                                          -- Optimistic concurrency token
        [IsActive]      bit NOT NULL CONSTRAINT [DF_cfg_Currency_IsActive] DEFAULT (1),                               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_Currency] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_Currency] UNIQUE ([TenantId], [CurrencyCode])
    );
END
GO

/* cfg.CurrencyDecimal - Currencies whose decimals differ from 2 (reference: TCURX) */
IF OBJECT_ID(N'cfg.CurrencyDecimal', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[CurrencyDecimal]
    (
        [Id]            bigint IDENTITY(1,1) NOT NULL,                                                                -- Surrogate key
        [TenantId]      int NOT NULL,                                                                                 -- Owning tenant - every query is filtered by it
        [CurrencyCode]  nvarchar(5) NOT NULL,                                                                         -- Currency
        [DecimalPlaces] tinyint NOT NULL,                                                                             -- Number of decimals (0 for KHR, JPY)
        [CreatedAt]     datetime2(3) NOT NULL CONSTRAINT [DF_cfg_CurrencyDecimal_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]     nvarchar(64) NOT NULL,                                                                        -- Creating user name
        [ModifiedAt]    datetime2(3) NULL,                                                                            -- Last change timestamp (UTC)
        [ModifiedBy]    nvarchar(64) NULL,                                                                            -- Last changing user name
        [RowVersion]    rowversion NOT NULL,                                                                          -- Optimistic concurrency token
        [IsActive]      bit NOT NULL CONSTRAINT [DF_cfg_CurrencyDecimal_IsActive] DEFAULT (1),                        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_CurrencyDecimal] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_CurrencyDecimal] UNIQUE ([TenantId], [CurrencyCode])
    );
END
GO

/* cfg.CurrencyTranslationRatio - Translation ratios per currency pair and rate type (reference: TCURF) */
IF OBJECT_ID(N'cfg.CurrencyTranslationRatio', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[CurrencyTranslationRatio]
    (
        [Id]                          bigint IDENTITY(1,1) NOT NULL,                                                  -- Surrogate key
        [TenantId]                    int NOT NULL,                                                                   -- Owning tenant - every query is filtered by it
        [ExchangeRateTypeId]          bigint NOT NULL,                                                                -- Rate type
        [FromCurrencyCode]            nvarchar(5) NOT NULL,                                                           -- Source currency
        [ToCurrencyCode]              nvarchar(5) NOT NULL,                                                           -- Target currency
        [ValidFrom]                   date NOT NULL,                                                                  -- Valid-from date
        [FromRatio]                   int NOT NULL,                                                                   -- Source ratio
        [ToRatio]                     int NOT NULL,                                                                   -- Target ratio
        [AlternativeExchangeRateType] nvarchar(4) NULL,                                                               -- Rate type used instead
        [CreatedAt]                   datetime2(3) NOT NULL CONSTRAINT [DF_cfg_CurrencyTranslationRatio_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                   nvarchar(64) NOT NULL,                                                          -- Creating user name
        [ModifiedAt]                  datetime2(3) NULL,                                                              -- Last change timestamp (UTC)
        [ModifiedBy]                  nvarchar(64) NULL,                                                              -- Last changing user name
        [RowVersion]                  rowversion NOT NULL,                                                            -- Optimistic concurrency token
        [IsActive]                    bit NOT NULL CONSTRAINT [DF_cfg_CurrencyTranslationRatio_IsActive] DEFAULT (1), -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_CurrencyTranslationRatio] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_CurrencyTranslationRatio] UNIQUE ([TenantId], [ExchangeRateTypeId], [FromCurrencyCode], [ToCurrencyCode], [ValidFrom])
    );
END
GO

/* cfg.CustomFieldDefinition - ZZ* extension field added to a standard entity */
IF OBJECT_ID(N'cfg.CustomFieldDefinition', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[CustomFieldDefinition]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [HostEntity]           nvarchar(64) NOT NULL,                                                                 -- BusinessPartner, Asset, JournalEntryHeader, JournalEntryLine, CostCenter, ProfitCenter, InternalOrder
        [FieldName]            nvarchar(64) NOT NULL,                                                                 -- Field name, must start ZZ, e.g. ZZOldAssetNumber
        [LabelEn]              nvarchar(60) NOT NULL,                                                                 -- English label
        [LabelKm]              nvarchar(60) NULL,                                                                     -- Khmer label
        [ShortDescription]     nvarchar(255) NULL,                                                                    -- Description
        [DataElementId]        bigint NULL,                                                                           -- Data element
        [SqlType]              nvarchar(40) NOT NULL,                                                                 -- Physical type of the generated column
        [Length]               int NULL,                                                                              -- Length
        [DecimalPlaces]        tinyint NULL,                                                                          -- Decimals
        [IsRequired]           bit NOT NULL CONSTRAINT [DF_cfg_CustomFieldDefinition_IsRequired] DEFAULT (0),         -- Mandatory on the screen
        [DefaultValue]         nvarchar(255) NULL,                                                                    -- Default
        [SearchHelpId]         bigint NULL,                                                                           -- Search help
        [ValidationExpression] nvarchar(255) NULL,                                                                    -- Server-side validation rule
        [DisplayOrder]         int NOT NULL,                                                                          -- Position on the screen
        [ScreenSection]        nvarchar(40) NOT NULL,                                                                 -- Tab / group the field appears in
        [AuthorizationGroup]   nvarchar(4) NULL,                                                                      -- Authorization group
        [IsReportingEnabled]   bit NOT NULL CONSTRAINT [DF_cfg_CustomFieldDefinition_IsReportingEnabled] DEFAULT (0), -- Available as a report dimension
        [IsApiExposed]         bit NOT NULL CONSTRAINT [DF_cfg_CustomFieldDefinition_IsApiExposed] DEFAULT (0),       -- Returned by the REST API
        [IsPhysicalColumn]     bit NOT NULL CONSTRAINT [DF_cfg_CustomFieldDefinition_IsPhysicalColumn] DEFAULT (0),   -- Generated as a real column (required for reportable fields)
        [LifecycleStatus]      nvarchar(20) NOT NULL,                                                                 -- Lifecycle stage
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_CustomFieldDefinition_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_CustomFieldDefinition_IsActive] DEFAULT (1),           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_CustomFieldDefinition] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_CustomFieldDefinition] UNIQUE ([TenantId], [HostEntity], [FieldName])
    );
END
GO

/* cfg.CustomFieldValue - Value store for non-physical ZZ* fields (never used for accounting-relevant data) */
IF OBJECT_ID(N'cfg.CustomFieldValue', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[CustomFieldValue]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [CustomFieldDefinitionId] bigint NOT NULL,                                                                    -- Field definition
        [HostEntity]              nvarchar(64) NOT NULL,                                                              -- Owning entity type
        [HostEntityId]            bigint NOT NULL,                                                                    -- Owning entity key
        [TextValue]               nvarchar(255) NULL,                                                                 -- Text value
        [NumericValue]            decimal(23,6) NULL,                                                                 -- Numeric value
        [DateValue]               date NULL,                                                                          -- Date value
        [BooleanValue]            bit NULL,                                                                           -- Boolean value
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_cfg_CustomFieldValue_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_cfg_CustomFieldValue_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_CustomFieldValue] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_CustomFieldValue] UNIQUE ([TenantId], [CustomFieldDefinitionId], [HostEntity], [HostEntityId])
    );
END
GO

/* cfg.CustomObjectRequest - Change request carrying custom objects through the landscape */
IF OBJECT_ID(N'cfg.CustomObjectRequest', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[CustomObjectRequest]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [RequestNumber]     nvarchar(20) NOT NULL,                                                                    -- Request number
        [Title]             nvarchar(255) NOT NULL,                                                                   -- Short text
        [RequestType]       nvarchar(20) NOT NULL,                                                                    -- Workbench, Customizing
        [Status]            nvarchar(20) NOT NULL,                                                                    -- Draft, Validated, ImpactReviewed, Approved, Released, ImportedTest, ImportedQuality, ImportedProduction, RolledBack
        [RequestedBy]       nvarchar(64) NOT NULL,                                                                    -- Requester
        [DeveloperUserName] nvarchar(64) NULL,                                                                        -- Developer
        [ApprovedBy]        nvarchar(64) NULL,                                                                        -- Approver
        [ApprovedAt]        datetime2(3) NULL,                                                                        -- Approval timestamp (UTC)
        [TargetEnvironment] nvarchar(20) NOT NULL,                                                                    -- Development, Test, Quality, Production
        [ImpactAssessment]  nvarchar(max) NULL,                                                                       -- Affected objects and risk notes
        [RollbackPlan]      nvarchar(max) NULL,                                                                       -- Rollback instructions
        [ReleasedAt]        datetime2(3) NULL,                                                                        -- Release timestamp (UTC)
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_cfg_CustomObjectRequest_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_cfg_CustomObjectRequest_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_CustomObjectRequest] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_CustomObjectRequest] UNIQUE ([TenantId], [RequestNumber])
    );
END
GO

/* cfg.CustomObjectRequestItem - Object contained in a change request */
IF OBJECT_ID(N'cfg.CustomObjectRequestItem', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[CustomObjectRequestItem]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [CustomObjectRequestId] bigint NOT NULL,                                                                      -- Owning request
        [ObjectType]            nvarchar(20) NOT NULL,                                                                -- Dictionary object type
        [ObjectName]            nvarchar(64) NOT NULL,                                                                -- Object name
        [ObjectVersion]         int NOT NULL,                                                                         -- Version transported
        [OperationType]         nvarchar(20) NOT NULL,                                                                -- Create, Change, Delete
        [DependencyList]        nvarchar(max) NULL,                                                                   -- Objects that must be imported first
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_cfg_CustomObjectRequestItem_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_cfg_CustomObjectRequestItem_IsActive] DEFAULT (1),        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_CustomObjectRequestItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_CustomObjectRequestItem] UNIQUE ([TenantId], [CustomObjectRequestId], [ObjectType], [ObjectName])
    );
END
GO

/* cfg.CustomTable - User-defined table definition */
IF OBJECT_ID(N'cfg.CustomTable', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[CustomTable]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [CustomTableName]        nvarchar(64) NOT NULL,                                                               -- Name, must start Z or Y, e.g. ZCANE_CONTRACT
        [SchemaName]             nvarchar(20) NOT NULL,                                                               -- Target schema (cus by default)
        [ShortDescription]       nvarchar(255) NOT NULL,                                                              -- Description
        [TableCategory]          nvarchar(20) NOT NULL,                                                               -- Master, TransactionHeader, TransactionItem, Configuration, Translation, History, Relationship
        [IsTenantDependent]      bit NOT NULL CONSTRAINT [DF_cfg_CustomTable_IsTenantDependent] DEFAULT (0),          -- Include TenantId
        [IsCompanyCodeDependent] bit NOT NULL CONSTRAINT [DF_cfg_CustomTable_IsCompanyCodeDependent] DEFAULT (0),     -- Include CompanyCodeId
        [HasValidityPeriod]      bit NOT NULL CONSTRAINT [DF_cfg_CustomTable_HasValidityPeriod] DEFAULT (0),          -- Include ValidFrom / ValidTo
        [HasAuditFields]         bit NOT NULL CONSTRAINT [DF_cfg_CustomTable_HasAuditFields] DEFAULT (0),             -- Include #AUDIT
        [AuthorizationGroup]     nvarchar(4) NULL,                                                                    -- Authorization group
        [NumberRangeObjectId]    bigint NULL,                                                                         -- Number range for generated keys
        [IsApiExposed]           bit NOT NULL CONSTRAINT [DF_cfg_CustomTable_IsApiExposed] DEFAULT (0),               -- Generate a REST API
        [IsBrowserVisible]       bit NOT NULL CONSTRAINT [DF_cfg_CustomTable_IsBrowserVisible] DEFAULT (0),           -- Visible in the table browser
        [LifecycleStatus]        nvarchar(20) NOT NULL,                                                               -- Draft, Validated, ImpactReviewed, Approved, Development, Test, Quality, Production
        [DictionaryObjectId]     bigint NULL,                                                                         -- Dictionary registry entry
        [Status]                 nvarchar(20) NOT NULL,                                                               -- Activation status
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_CustomTable_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_CustomTable_IsActive] DEFAULT (1),                   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_CustomTable] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_CustomTable] UNIQUE ([TenantId], [CustomTableName])
    );
END
GO

/* cfg.CustomTableField - Field of a user-defined table */
IF OBJECT_ID(N'cfg.CustomTableField', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[CustomTableField]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [CustomTableId]        bigint NOT NULL,                                                                       -- Owning custom table
        [FieldName]            nvarchar(64) NOT NULL,                                                                 -- Field name
        [FieldPosition]        int NOT NULL,                                                                          -- Column order
        [DataElementId]        bigint NULL,                                                                           -- Data element
        [DictionaryDomainId]   bigint NULL,                                                                           -- Domain (when no data element is used)
        [SqlType]              nvarchar(40) NOT NULL,                                                                 -- Physical SQL Server type
        [Length]               int NULL,                                                                              -- Length
        [DecimalPlaces]        tinyint NULL,                                                                          -- Decimals
        [IsKey]                bit NOT NULL CONSTRAINT [DF_cfg_CustomTableField_IsKey] DEFAULT (0),                   -- Part of the key
        [IsRequired]           bit NOT NULL CONSTRAINT [DF_cfg_CustomTableField_IsRequired] DEFAULT (0),              -- Mandatory
        [IsUnique]             bit NOT NULL CONSTRAINT [DF_cfg_CustomTableField_IsUnique] DEFAULT (0),                -- Unique constraint
        [DefaultValue]         nvarchar(255) NULL,                                                                    -- Default
        [CheckTableName]       nvarchar(64) NULL,                                                                     -- Check table
        [SearchHelpId]         bigint NULL,                                                                           -- Search help
        [ValidationExpression] nvarchar(255) NULL,                                                                    -- Server-side validation rule
        [LabelEn]              nvarchar(60) NOT NULL,                                                                 -- English label
        [LabelKm]              nvarchar(60) NULL,                                                                     -- Khmer label
        [ShortDescription]     nvarchar(255) NULL,                                                                    -- Description
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_CustomTableField_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_CustomTableField_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_CustomTableField] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_CustomTableField] UNIQUE ([TenantId], [CustomTableId], [FieldName])
    );
END
GO

/* cfg.DictionaryChangeLog - Every dictionary change, with before/after definition */
IF OBJECT_ID(N'cfg.DictionaryChangeLog', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryChangeLog]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [DictionaryObjectId] bigint NOT NULL,                                                                         -- Object changed
        [VersionNumber]      int NOT NULL,                                                                            -- Version produced by the change
        [Action]             nvarchar(20) NOT NULL,                                                                   -- Create, Change, Copy, Activate, Deactivate, Delete
        [ChangedAt]          datetime2(3) NOT NULL,                                                                   -- Timestamp (UTC)
        [ChangedBy]          nvarchar(64) NOT NULL,                                                                   -- User
        [OldDefinition]      nvarchar(max) NULL,                                                                      -- Previous definition (JSON)
        [NewDefinition]      nvarchar(max) NULL,                                                                      -- New definition (JSON)
        [MigrationScriptId]  bigint NULL,                                                                             -- Generated migration
        [ApprovedBy]         nvarchar(64) NULL,                                                                       -- Approver
        [ApprovedAt]         datetime2(3) NULL,                                                                       -- Approval timestamp (UTC)
        [TransportRequestId] bigint NULL,                                                                             -- Change request carrying the object
        [CorrelationId]      uniqueidentifier NULL,                                                                   -- Request correlation id
        CONSTRAINT [PK_cfg_DictionaryChangeLog] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* cfg.DictionaryDataElement - Data element - business meaning and labels on top of a domain */
IF OBJECT_ID(N'cfg.DictionaryDataElement', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryDataElement]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [DataElementName]          nvarchar(64) NOT NULL,                                                             -- Data element name, e.g. DE_COMPANY_CODE
        [DictionaryDomainId]       bigint NOT NULL,                                                                   -- Underlying domain
        [ShortDescription]         nvarchar(255) NOT NULL,                                                            -- Description
        [ShortLabel]               nvarchar(10) NULL,                                                                 -- Short field label
        [MediumLabel]              nvarchar(20) NULL,                                                                 -- Medium field label
        [LongLabel]                nvarchar(40) NULL,                                                                 -- Long field label
        [HeaderLabel]              nvarchar(55) NULL,                                                                 -- Column header
        [SearchHelpId]             bigint NULL,                                                                       -- Attached search help
        [SearchHelpParameter]      nvarchar(64) NULL,                                                                 -- Export parameter of the search help
        [ParameterId]              nvarchar(20) NULL,                                                                 -- User default (SET/GET parameter equivalent)
        [DocumentationText]        nvarchar(max) NULL,                                                                -- F1 help text
        [IsChangeDocumentRelevant] bit NOT NULL CONSTRAINT [DF_cfg_DictionaryDataElement_IsChangeDocumentRelevant] DEFAULT (0),-- Changes are logged in change documents
        [IsPersonalData]           bit NOT NULL CONSTRAINT [DF_cfg_DictionaryDataElement_IsPersonalData] DEFAULT (0), -- Personal data - masking and retention apply
        [Status]                   nvarchar(20) NOT NULL,                                                             -- Activation status
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryDataElement_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_cfg_DictionaryDataElement_IsActive] DEFAULT (1),       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryDataElement] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryDataElement] UNIQUE ([TenantId], [DataElementName])
    );
END
GO

/* cfg.DictionaryDomain - Domain - technical type, length, value range */
IF OBJECT_ID(N'cfg.DictionaryDomain', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryDomain]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [DomainName]        nvarchar(64) NOT NULL,                                                                    -- Domain name, e.g. D_AMOUNT
        [ShortDescription]  nvarchar(255) NOT NULL,                                                                   -- Description
        [DataType]          nvarchar(20) NOT NULL,                                                                    -- Logical type: CHAR, NUMC, DEC, INT, DATE, TIME, BOOL, GUID, CLOB
        [SqlType]           nvarchar(40) NOT NULL,                                                                    -- Generated SQL Server type, e.g. decimal(19,4)
        [Length]            int NULL,                                                                                 -- Field length
        [DecimalPlaces]     tinyint NULL,                                                                             -- Number of decimals
        [IsSigned]          bit NOT NULL CONSTRAINT [DF_cfg_DictionaryDomain_IsSigned] DEFAULT (0),                   -- Negative values allowed
        [OutputLength]      int NULL,                                                                                 -- Display length
        [ConversionRoutine] nvarchar(20) NULL,                                                                        -- Conversion exit, e.g. leading-zero handling
        [CaseSensitive]     bit NOT NULL CONSTRAINT [DF_cfg_DictionaryDomain_CaseSensitive] DEFAULT (0),              -- Lower case permitted
        [ValueTableName]    nvarchar(64) NULL,                                                                        -- Value (check) table
        [HasFixedValues]    bit NOT NULL CONSTRAINT [DF_cfg_DictionaryDomain_HasFixedValues] DEFAULT (0),             -- Fixed value list maintained
        [LowerLimit]        nvarchar(40) NULL,                                                                        -- Lower interval limit
        [UpperLimit]        nvarchar(40) NULL,                                                                        -- Upper interval limit
        [Status]            nvarchar(20) NOT NULL,                                                                    -- Activation status
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryDomain_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_cfg_DictionaryDomain_IsActive] DEFAULT (1),                   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryDomain] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryDomain] UNIQUE ([TenantId], [DomainName])
    );
END
GO

/* cfg.DictionaryDomainValue - Fixed value or value range of a domain */
IF OBJECT_ID(N'cfg.DictionaryDomainValue', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryDomainValue]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [DictionaryDomainId] bigint NOT NULL,                                                                         -- Owning domain
        [LowValue]           nvarchar(40) NOT NULL,                                                                   -- Fixed value or interval start
        [HighValue]          nvarchar(40) NULL,                                                                       -- Interval end (blank = single value)
        [Description]        nvarchar(60) NOT NULL,                                                                   -- Value text
        [LanguageCode]       nvarchar(2) NOT NULL,                                                                    -- Text language
        [DisplayOrder]       int NOT NULL,                                                                            -- Order in drop-downs
        [IsDefault]          bit NOT NULL CONSTRAINT [DF_cfg_DictionaryDomainValue_IsDefault] DEFAULT (0),            -- Proposed value
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryDomainValue_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_DictionaryDomainValue_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryDomainValue] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryDomainValue] UNIQUE ([TenantId], [DictionaryDomainId], [LowValue], [LanguageCode])
    );
END
GO

/* cfg.DictionaryForeignKey - Foreign key / check table relationship */
IF OBJECT_ID(N'cfg.DictionaryForeignKey', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryForeignKey]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [ForeignKeyName]   nvarchar(64) NOT NULL,                                                                     -- Constraint name
        [SourceSchemaName] nvarchar(20) NOT NULL,                                                                     -- Source schema
        [SourceTableName]  nvarchar(64) NOT NULL,                                                                     -- Source (dependent) table
        [TargetSchemaName] nvarchar(20) NOT NULL,                                                                     -- Target schema
        [TargetTableName]  nvarchar(64) NOT NULL,                                                                     -- Target (check) table
        [Cardinality]      nvarchar(5) NOT NULL,                                                                      -- 1:1, 1:N, C:N, 1:CN
        [ForeignKeyType]   nvarchar(20) NOT NULL,                                                                     -- KeyFields, NonKeyFields, TextTable
        [CheckRequired]    bit NOT NULL CONSTRAINT [DF_cfg_DictionaryForeignKey_CheckRequired] DEFAULT (0),           -- Input checked against the target table
        [OnDeleteAction]   nvarchar(20) NOT NULL,                                                                     -- NoAction, Cascade, SetNull, Restrict
        [Status]           nvarchar(20) NOT NULL,                                                                     -- Activation status
        [CreatedAt]        datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryForeignKey_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]        nvarchar(64) NOT NULL,                                                                     -- Creating user name
        [ModifiedAt]       datetime2(3) NULL,                                                                         -- Last change timestamp (UTC)
        [ModifiedBy]       nvarchar(64) NULL,                                                                         -- Last changing user name
        [RowVersion]       rowversion NOT NULL,                                                                       -- Optimistic concurrency token
        [IsActive]         bit NOT NULL CONSTRAINT [DF_cfg_DictionaryForeignKey_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryForeignKey] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryForeignKey] UNIQUE ([TenantId], [ForeignKeyName])
    );
END
GO

/* cfg.DictionaryForeignKeyField - Field pair of a foreign key */
IF OBJECT_ID(N'cfg.DictionaryForeignKeyField', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryForeignKeyField]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [DictionaryForeignKeyId] bigint NOT NULL,                                                                     -- Owning foreign key
        [SourceFieldName]        nvarchar(64) NOT NULL,                                                               -- Field in the source table
        [TargetFieldName]        nvarchar(64) NOT NULL,                                                               -- Field in the target table
        [ConstantValue]          nvarchar(40) NULL,                                                                   -- Constant instead of a field
        [FieldPosition]          int NOT NULL,                                                                        -- Order in the composite key
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryForeignKeyField_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_DictionaryForeignKeyField_IsActive] DEFAULT (1),     -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryForeignKeyField] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryForeignKeyField] UNIQUE ([TenantId], [DictionaryForeignKeyId], [SourceFieldName])
    );
END
GO

/* cfg.DictionaryIndex - Secondary index definition */
IF OBJECT_ID(N'cfg.DictionaryIndex', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryIndex]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [DictionaryTableId] bigint NOT NULL,                                                                          -- Owning table
        [IndexName]         nvarchar(64) NOT NULL,                                                                    -- Index name
        [ShortDescription]  nvarchar(255) NULL,                                                                       -- Purpose of the index
        [IsUnique]          bit NOT NULL CONSTRAINT [DF_cfg_DictionaryIndex_IsUnique] DEFAULT (0),                    -- Unique index
        [IsClustered]       bit NOT NULL CONSTRAINT [DF_cfg_DictionaryIndex_IsClustered] DEFAULT (0),                 -- Clustered index
        [IsColumnStore]     bit NOT NULL CONSTRAINT [DF_cfg_DictionaryIndex_IsColumnStore] DEFAULT (0),               -- Columnstore index (reporting tables)
        [FilterPredicate]   nvarchar(255) NULL,                                                                       -- Filtered index predicate
        [IncludedColumns]   nvarchar(255) NULL,                                                                       -- Included (non-key) columns
        [Status]            nvarchar(20) NOT NULL,                                                                    -- Activation status
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryIndex_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_cfg_DictionaryIndex_IsActive] DEFAULT (1),                    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryIndex] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryIndex] UNIQUE ([TenantId], [DictionaryTableId], [IndexName])
    );
END
GO

/* cfg.DictionaryIndexField - Field of a secondary index */
IF OBJECT_ID(N'cfg.DictionaryIndexField', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryIndexField]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [DictionaryIndexId] bigint NOT NULL,                                                                          -- Owning index
        [FieldName]         nvarchar(64) NOT NULL,                                                                    -- Indexed field
        [FieldPosition]     int NOT NULL,                                                                             -- Position in the index
        [SortDirection]     nvarchar(4) NOT NULL,                                                                     -- ASC, DESC
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryIndexField_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_cfg_DictionaryIndexField_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryIndexField] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryIndexField] UNIQUE ([TenantId], [DictionaryIndexId], [FieldName])
    );
END
GO

/* cfg.DictionaryLockObject - Lock object for application-level locking */
IF OBJECT_ID(N'cfg.DictionaryLockObject', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryLockObject]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [LockObjectName]     nvarchar(64) NOT NULL,                                                                   -- Lock object name, e.g. E_JOURNAL
        [ShortDescription]   nvarchar(255) NOT NULL,                                                                  -- Description
        [PrimaryTableName]   nvarchar(64) NOT NULL,                                                                   -- Table locked
        [LockMode]           nvarchar(10) NOT NULL,                                                                   -- Exclusive, Shared, ExclusiveNonCumulative
        [LockArgumentFields] nvarchar(255) NOT NULL,                                                                  -- Fields forming the lock argument
        [LockTimeoutSeconds] int NOT NULL,                                                                            -- Timeout before the lock is refused
        [Status]             nvarchar(20) NOT NULL,                                                                   -- Activation status
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryLockObject_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_DictionaryLockObject_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryLockObject] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryLockObject] UNIQUE ([TenantId], [LockObjectName])
    );
END
GO

/* cfg.DictionaryObject - Registry of every dictionary object and its activation status */
IF OBJECT_ID(N'cfg.DictionaryObject', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryObject]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [ObjectType]        nvarchar(20) NOT NULL,                                                                    -- Table, View, Structure, Domain, DataElement, SearchHelp, LockObject, Index
        [ObjectName]        nvarchar(64) NOT NULL,                                                                    -- Object name, e.g. fin.JournalEntryLine, ZCANE_CONTRACT
        [ShortDescription]  nvarchar(255) NOT NULL,                                                                   -- Description shown in the object list
        [Package]           nvarchar(40) NULL,                                                                        -- Development package / module
        [Status]            nvarchar(20) NOT NULL,                                                                    -- Draft, Checked, Active, Inactive, Deprecated
        [IsCustomObject]    bit NOT NULL CONSTRAINT [DF_cfg_DictionaryObject_IsCustomObject] DEFAULT (0),             -- Z* / Y* object
        [NamespacePrefix]   nvarchar(4) NULL,                                                                         -- Z, Y, ZZ
        [ResponsibleUser]   nvarchar(64) NULL,                                                                        -- Owner
        [LastActivatedAt]   datetime2(3) NULL,                                                                        -- Last successful activation (UTC)
        [LastActivatedBy]   nvarchar(64) NULL,                                                                        -- Activating user
        [ActiveVersion]     int NOT NULL,                                                                             -- Version currently active
        [DraftVersion]      int NOT NULL,                                                                             -- Highest draft version
        [DocumentationText] nvarchar(max) NULL,                                                                       -- Object documentation
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryObject_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_cfg_DictionaryObject_IsActive] DEFAULT (1),                   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryObject] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryObject] UNIQUE ([TenantId], [ObjectType], [ObjectName])
    );
END
GO

/* cfg.DictionarySearchHelp - Search help (F4) */
IF OBJECT_ID(N'cfg.DictionarySearchHelp', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionarySearchHelp]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [SearchHelpName]   nvarchar(64) NOT NULL,                                                                     -- Search help name
        [ShortDescription] nvarchar(255) NOT NULL,                                                                    -- Description
        [SearchHelpType]   nvarchar(20) NOT NULL,                                                                     -- Elementary, Collective
        [SelectionMethod]  nvarchar(64) NOT NULL,                                                                     -- Table or view read
        [TextTableName]    nvarchar(64) NULL,                                                                         -- Text table for descriptions
        [DialogType]       nvarchar(20) NOT NULL,                                                                     -- ImmediateDisplay, DialogWithRestriction
        [HotKey]           nvarchar(1) NULL,                                                                          -- Shortcut in collective search helps
        [MaxHits]          int NOT NULL,                                                                              -- Maximum rows returned
        [Status]           nvarchar(20) NOT NULL,                                                                     -- Activation status
        [CreatedAt]        datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionarySearchHelp_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]        nvarchar(64) NOT NULL,                                                                     -- Creating user name
        [ModifiedAt]       datetime2(3) NULL,                                                                         -- Last change timestamp (UTC)
        [ModifiedBy]       nvarchar(64) NULL,                                                                         -- Last changing user name
        [RowVersion]       rowversion NOT NULL,                                                                       -- Optimistic concurrency token
        [IsActive]         bit NOT NULL CONSTRAINT [DF_cfg_DictionarySearchHelp_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionarySearchHelp] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionarySearchHelp] UNIQUE ([TenantId], [SearchHelpName])
    );
END
GO

/* cfg.DictionarySearchHelpParameter - Import / export parameter of a search help */
IF OBJECT_ID(N'cfg.DictionarySearchHelpParameter', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionarySearchHelpParameter]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [DictionarySearchHelpId] bigint NOT NULL,                                                                     -- Owning search help
        [ParameterName]          nvarchar(64) NOT NULL,                                                               -- Parameter name
        [DataElementId]          bigint NULL,                                                                         -- Data element
        [IsImport]               bit NOT NULL CONSTRAINT [DF_cfg_DictionarySearchHelpParameter_IsImport] DEFAULT (0), -- Import parameter
        [IsExport]               bit NOT NULL CONSTRAINT [DF_cfg_DictionarySearchHelpParameter_IsExport] DEFAULT (0), -- Export parameter
        [IsListField]            bit NOT NULL CONSTRAINT [DF_cfg_DictionarySearchHelpParameter_IsListField] DEFAULT (0),-- Shown in the hit list
        [IsSelectionField]       bit NOT NULL CONSTRAINT [DF_cfg_DictionarySearchHelpParameter_IsSelectionField] DEFAULT (0),-- Shown in the restriction dialog
        [ListPosition]           int NULL,                                                                            -- Column order in the hit list
        [SelectionPosition]      int NULL,                                                                            -- Order in the restriction dialog
        [DefaultValue]           nvarchar(255) NULL,                                                                  -- Default value
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionarySearchHelpParameter_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_DictionarySearchHelpParameter_IsActive] DEFAULT (1), -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionarySearchHelpParameter] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionarySearchHelpParameter] UNIQUE ([TenantId], [DictionarySearchHelpId], [ParameterName])
    );
END
GO

/* cfg.DictionaryStructure - Structure (no database table behind it) */
IF OBJECT_ID(N'cfg.DictionaryStructure', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryStructure]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [StructureName]    nvarchar(64) NOT NULL,                                                                     -- Structure name, e.g. #AUDIT
        [ShortDescription] nvarchar(255) NOT NULL,                                                                    -- Description
        [StructureType]    nvarchar(20) NOT NULL,                                                                     -- Include, Dto, ScreenStructure
        [Status]           nvarchar(20) NOT NULL,                                                                     -- Activation status
        [CreatedAt]        datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryStructure_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]        nvarchar(64) NOT NULL,                                                                     -- Creating user name
        [ModifiedAt]       datetime2(3) NULL,                                                                         -- Last change timestamp (UTC)
        [ModifiedBy]       nvarchar(64) NULL,                                                                         -- Last changing user name
        [RowVersion]       rowversion NOT NULL,                                                                       -- Optimistic concurrency token
        [IsActive]         bit NOT NULL CONSTRAINT [DF_cfg_DictionaryStructure_IsActive] DEFAULT (1),                 -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryStructure] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryStructure] UNIQUE ([TenantId], [StructureName])
    );
END
GO

/* cfg.DictionaryStructureField - Field of a structure */
IF OBJECT_ID(N'cfg.DictionaryStructureField', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryStructureField]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [DictionaryStructureId] bigint NOT NULL,                                                                      -- Owning structure
        [FieldName]             nvarchar(64) NOT NULL,                                                                -- Field name
        [FieldPosition]         int NOT NULL,                                                                         -- Order
        [DataElementId]         bigint NULL,                                                                          -- Data element
        [SqlType]               nvarchar(40) NOT NULL,                                                                -- Physical type
        [IsRequired]            bit NOT NULL CONSTRAINT [DF_cfg_DictionaryStructureField_IsRequired] DEFAULT (0),     -- Mandatory
        [ShortDescription]      nvarchar(255) NOT NULL,                                                               -- Description
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryStructureField_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_cfg_DictionaryStructureField_IsActive] DEFAULT (1),       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryStructureField] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryStructureField] UNIQUE ([TenantId], [DictionaryStructureId], [FieldName])
    );
END
GO

/* cfg.DictionaryTable - Table definition and technical settings */
IF OBJECT_ID(N'cfg.DictionaryTable', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryTable]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [SchemaName]             nvarchar(20) NOT NULL,                                                               -- Database schema, e.g. fin
        [TableName]              nvarchar(64) NOT NULL,                                                               -- Table name
        [ShortDescription]       nvarchar(255) NOT NULL,                                                              -- Description
        [TableCategory]          nvarchar(20) NOT NULL,                                                               -- Transparent, Configuration, Master, Transaction, Text, Custom
        [DeliveryClass]          nvarchar(1) NOT NULL,                                                                -- A application, C customizing, S system, L temporary
        [MaintenanceType]        nvarchar(20) NOT NULL,                                                               -- NotAllowed, Display, Maintain
        [IsTenantDependent]      bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTable_IsTenantDependent] DEFAULT (0),      -- Carries TenantId
        [IsCompanyCodeDependent] bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTable_IsCompanyCodeDependent] DEFAULT (0), -- Carries CompanyCodeId
        [SizeCategory]           tinyint NOT NULL,                                                                    -- Expected volume class 0..9
        [BufferingType]          nvarchar(20) NOT NULL,                                                               -- None, FullyBuffered, GenericKey, SingleRecord
        [IsLogged]               bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTable_IsLogged] DEFAULT (0),               -- Table changes written to change documents
        [IsImmutable]            bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTable_IsImmutable] DEFAULT (0),            -- Posted data - updates and deletes rejected
        [AuthorizationGroup]     nvarchar(4) NULL,                                                                    -- Table authorization group for SE16N
        [PrimaryKeyFields]       nvarchar(255) NOT NULL,                                                              -- Comma-separated PK field list
        [Status]                 nvarchar(20) NOT NULL,                                                               -- Activation status
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryTable_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTable_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryTable] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryTable] UNIQUE ([TenantId], [SchemaName], [TableName])
    );
END
GO

/* cfg.DictionaryTableField -  */
IF OBJECT_ID(N'cfg.DictionaryTableField', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryTableField]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [DictionaryTableId]      bigint NOT NULL,                                                                     -- Owning table
        [FieldName]              nvarchar(64) NOT NULL,                                                               -- Field name
        [FieldPosition]          int NOT NULL,                                                                        -- Column order
        [DataElementId]          bigint NULL,                                                                         -- Data element (null for include-generated fields)
        [DomainName]             nvarchar(64) NULL,                                                                   -- Domain resolved from the data element
        [SqlType]                nvarchar(40) NOT NULL,                                                               -- Physical SQL Server type
        [Length]                 int NULL,                                                                            -- Length
        [DecimalPlaces]          tinyint NULL,                                                                        -- Decimals
        [IsKey]                  bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTableField_IsKey] DEFAULT (0),             -- Part of the primary key
        [IsRequired]             bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTableField_IsRequired] DEFAULT (0),        -- NOT NULL
        [IsIdentity]             bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTableField_IsIdentity] DEFAULT (0),        -- Identity / auto-increment
        [DefaultValue]           nvarchar(255) NULL,                                                                  -- Default expression
        [CheckTableName]         nvarchar(64) NULL,                                                                   -- Check table for input validation
        [ForeignKeyId]           bigint NULL,                                                                         -- Foreign key definition
        [SearchHelpId]           bigint NULL,                                                                         -- Field-level search help
        [CurrencyReferenceField] nvarchar(64) NULL,                                                                   -- Field holding the currency of an amount
        [UnitReferenceField]     nvarchar(64) NULL,                                                                   -- Field holding the unit of a quantity
        [IncludeName]            nvarchar(64) NULL,                                                                   -- Include the field came from, e.g. #AUDIT
        [IsCustomField]          bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTableField_IsCustomField] DEFAULT (0),     -- ZZ* customer field
        [IsMasked]               bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTableField_IsMasked] DEFAULT (0),          -- Masked in the table browser and exports
        [ShortDescription]       nvarchar(255) NOT NULL,                                                              -- Field description
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryTableField_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_DictionaryTableField_IsActive] DEFAULT (1),          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryTableField] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryTableField] UNIQUE ([TenantId], [DictionaryTableId], [FieldName])
    );
END
GO

/* cfg.DictionaryView - View definition (join or projection) */
IF OBJECT_ID(N'cfg.DictionaryView', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryView]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [SchemaName]         nvarchar(20) NOT NULL,                                                                   -- Schema
        [ViewName]           nvarchar(64) NOT NULL,                                                                   -- View name
        [ShortDescription]   nvarchar(255) NOT NULL,                                                                  -- Description
        [ViewType]           nvarchar(20) NOT NULL,                                                                   -- Database, Projection, Maintenance, Help
        [BaseTableName]      nvarchar(64) NOT NULL,                                                                   -- Primary table
        [JoinDefinition]     nvarchar(max) NULL,                                                                      -- Join conditions in JSON
        [SelectionCondition] nvarchar(max) NULL,                                                                      -- Fixed WHERE condition
        [IsReadOnly]         bit NOT NULL CONSTRAINT [DF_cfg_DictionaryView_IsReadOnly] DEFAULT (0),                  -- Read-only view
        [GeneratedSql]       nvarchar(max) NULL,                                                                      -- Generated CREATE VIEW statement
        [Status]             nvarchar(20) NOT NULL,                                                                   -- Activation status
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryView_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_DictionaryView_IsActive] DEFAULT (1),                    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryView] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryView] UNIQUE ([TenantId], [SchemaName], [ViewName])
    );
END
GO

/* cfg.DictionaryViewField - Field of a view */
IF OBJECT_ID(N'cfg.DictionaryViewField', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DictionaryViewField]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [DictionaryViewId]  bigint NOT NULL,                                                                          -- Owning view
        [ViewFieldName]     nvarchar(64) NOT NULL,                                                                    -- Field name in the view
        [SourceTableName]   nvarchar(64) NOT NULL,                                                                    -- Source table
        [SourceFieldName]   nvarchar(64) NOT NULL,                                                                    -- Source field
        [FieldPosition]     int NOT NULL,                                                                             -- Order
        [IsKey]             bit NOT NULL CONSTRAINT [DF_cfg_DictionaryViewField_IsKey] DEFAULT (0),                   -- Key field of the view
        [AggregateFunction] nvarchar(10) NULL,                                                                        -- SUM, MIN, MAX, COUNT
        [ShortDescription]  nvarchar(255) NULL,                                                                       -- Description
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DictionaryViewField_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_cfg_DictionaryViewField_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DictionaryViewField] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DictionaryViewField] UNIQUE ([TenantId], [DictionaryViewId], [ViewFieldName])
    );
END
GO

/* cfg.DocumentType - Document type - controls number range and allowed account types (reference: T003) */
IF OBJECT_ID(N'cfg.DocumentType', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DocumentType]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [DocumentType]              nvarchar(2) NOT NULL,                                                             -- Document type key, e.g. SA, KR, DR
        [Name]                      nvarchar(40) NOT NULL,                                                            -- Description
        [NumberRangeObjectId]       bigint NOT NULL,                                                                  -- Number range object used
        [NumberRangeCode]           nvarchar(2) NOT NULL,                                                             -- Number range interval key
        [ReverseDocumentType]       nvarchar(2) NULL,                                                                 -- Document type used for reversals
        [AllowCustomerAccounts]     bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_AllowCustomerAccounts] DEFAULT (0),  -- Account type D permitted
        [AllowVendorAccounts]       bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_AllowVendorAccounts] DEFAULT (0),    -- Account type K permitted
        [AllowGLAccounts]           bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_AllowGLAccounts] DEFAULT (0),        -- Account type S permitted
        [AllowAssetAccounts]        bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_AllowAssetAccounts] DEFAULT (0),     -- Account type A permitted
        [AllowMaterialAccounts]     bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_AllowMaterialAccounts] DEFAULT (0),  -- Account type M permitted
        [IsNetDocumentType]         bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_IsNetDocumentType] DEFAULT (0),      -- Net posting procedure
        [RequireReferenceNumber]    bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_RequireReferenceNumber] DEFAULT (0), -- Reference number mandatory
        [RequireDocumentHeaderText] bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_RequireDocumentHeaderText] DEFAULT (0),-- Header text mandatory
        [IsExternalNumbering]       bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_IsExternalNumbering] DEFAULT (0),    -- Number supplied by the user
        [NegativePostingPermitted]  bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_NegativePostingPermitted] DEFAULT (0),-- Negative postings allowed
        [IsIntercompany]            bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_IsIntercompany] DEFAULT (0),         -- Used for cross-company-code postings
        [SourceModule]              nvarchar(10) NULL,                                                                -- FI, AR, AP, AA, CO, MM, SD
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DocumentType_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_cfg_DocumentType_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DocumentType] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DocumentType] UNIQUE ([TenantId], [DocumentType])
    );
END
GO

/* cfg.DunningLevel - Dunning level settings (reference: T047) */
IF OBJECT_ID(N'cfg.DunningLevel', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DunningLevel]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [DunningProcedureId] bigint NOT NULL,                                                                         -- Dunning procedure
        [DunningLevel]       tinyint NOT NULL,                                                                        -- Level 1..9
        [DaysInArrears]      int NOT NULL,                                                                            -- Days in arrears to reach this level
        [CalculateInterest]  bit NOT NULL CONSTRAINT [DF_cfg_DunningLevel_CalculateInterest] DEFAULT (0),             -- Interest charged at this level
        [PrintAllItems]      bit NOT NULL CONSTRAINT [DF_cfg_DunningLevel_PrintAllItems] DEFAULT (0),                 -- Print all open items
        [AlwaysDun]          bit NOT NULL CONSTRAINT [DF_cfg_DunningLevel_AlwaysDun] DEFAULT (0),                     -- Dun even without new items
        [DunningCharge]      decimal(19,4) NULL,                                                                      -- Fixed dunning charge
        [MinimumAmount]      decimal(19,4) NULL,                                                                      -- Minimum amount to dun
        [FormId]             bigint NULL,                                                                             -- Correspondence form
        [IsLegalDunning]     bit NOT NULL CONSTRAINT [DF_cfg_DunningLevel_IsLegalDunning] DEFAULT (0),                -- Legal dunning procedure
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DunningLevel_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_DunningLevel_IsActive] DEFAULT (1),                      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DunningLevel] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DunningLevel] UNIQUE ([TenantId], [DunningProcedureId], [DunningLevel])
    );
END
GO

/* cfg.DunningProcedure - Dunning procedure (reference: T047A) */
IF OBJECT_ID(N'cfg.DunningProcedure', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[DunningProcedure]
    (
        [Id]                           bigint IDENTITY(1,1) NOT NULL,                                                 -- Surrogate key
        [TenantId]                     int NOT NULL,                                                                  -- Owning tenant - every query is filtered by it
        [DunningProcedure]             nvarchar(4) NOT NULL,                                                          -- Procedure key
        [Name]                         nvarchar(60) NOT NULL,                                                         -- Description
        [DunningIntervalDays]          int NOT NULL,                                                                  -- Minimum days between dunning runs
        [NumberOfDunningLevels]        tinyint NOT NULL,                                                              -- Number of levels
        [GracePeriodDays]              int NOT NULL,                                                                  -- Line item grace days
        [MinimumDaysInArrears]         int NOT NULL,                                                                  -- Minimum arrears before dunning
        [InterestIndicator]            nvarchar(2) NULL,                                                              -- Interest calculation indicator
        [IsStandardTransactionDunning] bit NOT NULL CONSTRAINT [DF_cfg_DunningProcedure_IsStandardTransactionDunning] DEFAULT (0),-- Dun standard transactions
        [CreatedAt]                    datetime2(3) NOT NULL CONSTRAINT [DF_cfg_DunningProcedure_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                    nvarchar(64) NOT NULL,                                                         -- Creating user name
        [ModifiedAt]                   datetime2(3) NULL,                                                             -- Last change timestamp (UTC)
        [ModifiedBy]                   nvarchar(64) NULL,                                                             -- Last changing user name
        [RowVersion]                   rowversion NOT NULL,                                                           -- Optimistic concurrency token
        [IsActive]                     bit NOT NULL CONSTRAINT [DF_cfg_DunningProcedure_IsActive] DEFAULT (1),        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_DunningProcedure] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_DunningProcedure] UNIQUE ([TenantId], [DunningProcedure])
    );
END
GO

/* cfg.ExchangeRate - Exchange rate valid from a date (reference: TCURR) */
IF OBJECT_ID(N'cfg.ExchangeRate', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[ExchangeRate]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [ExchangeRateTypeId] bigint NOT NULL,                                                                         -- Rate type
        [FromCurrencyCode]   nvarchar(5) NOT NULL,                                                                    -- Source currency
        [ToCurrencyCode]     nvarchar(5) NOT NULL,                                                                    -- Target currency
        [ValidFrom]          date NOT NULL,                                                                           -- Valid-from date - latest ? posting date wins
        [Rate]               decimal(23,6) NOT NULL,                                                                  -- Exchange rate
        [FromRatio]          int NOT NULL,                                                                            -- Ratio for the source currency
        [ToRatio]            int NOT NULL,                                                                            -- Ratio for the target currency
        [Source]             nvarchar(20) NULL,                                                                       -- Manual, Import, CentralBank
        [ImportedAt]         datetime2(3) NULL,                                                                       -- Import timestamp (UTC)
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_ExchangeRate_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_ExchangeRate_IsActive] DEFAULT (1),                      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_ExchangeRate] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_ExchangeRate] UNIQUE ([TenantId], [ExchangeRateTypeId], [FromCurrencyCode], [ToCurrencyCode], [ValidFrom])
    );
END
GO

/* cfg.ExchangeRateType - Exchange rate type (reference: TCURV) */
IF OBJECT_ID(N'cfg.ExchangeRateType', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[ExchangeRateType]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [ExchangeRateType]      nvarchar(4) NOT NULL,                                                                 -- Rate type, e.g. M, B, G, EURX
        [Name]                  nvarchar(40) NOT NULL,                                                                -- Description
        [QuotationType]         nvarchar(10) NOT NULL,                                                                -- Direct, Indirect
        [ReferenceCurrencyCode] nvarchar(5) NULL,                                                                     -- Base currency for cross rates
        [IsInversionAllowed]    bit NOT NULL CONSTRAINT [DF_cfg_ExchangeRateType_IsInversionAllowed] DEFAULT (0),     -- Inverted rate may be used when none is found
        [UseFixedRate]          bit NOT NULL CONSTRAINT [DF_cfg_ExchangeRateType_UseFixedRate] DEFAULT (0),           -- Fixed rate (e.g. currency peg)
        [IsBuyingRate]          bit NOT NULL CONSTRAINT [DF_cfg_ExchangeRateType_IsBuyingRate] DEFAULT (0),           -- Bank buying rate
        [IsSellingRate]         bit NOT NULL CONSTRAINT [DF_cfg_ExchangeRateType_IsSellingRate] DEFAULT (0),          -- Bank selling rate
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_cfg_ExchangeRateType_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_cfg_ExchangeRateType_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_ExchangeRateType] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_ExchangeRateType] UNIQUE ([TenantId], [ExchangeRateType])
    );
END
GO

/* cfg.FieldStatusFieldControl - Per-field status inside a field status group (reference: T004F) */
IF OBJECT_ID(N'cfg.FieldStatusFieldControl', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[FieldStatusFieldControl]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [FieldStatusGroupId] bigint NOT NULL,                                                                         -- Owning group
        [FieldName]          nvarchar(64) NOT NULL,                                                                   -- Journal line field controlled
        [FieldGroup]         nvarchar(40) NOT NULL,                                                                   -- Screen group, e.g. Additional account assignments
        [FieldStatus]        nvarchar(10) NOT NULL,                                                                   -- Suppress, Optional, Required, Display
        [DisplayOrder]       int NOT NULL,                                                                            -- Order on the entry screen
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_FieldStatusFieldControl_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_FieldStatusFieldControl_IsActive] DEFAULT (1),           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_FieldStatusFieldControl] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_FieldStatusFieldControl] UNIQUE ([TenantId], [FieldStatusGroupId], [FieldName])
    );
END
GO

/* cfg.FieldStatusGroup - Field status group (reference: T004G) */
IF OBJECT_ID(N'cfg.FieldStatusGroup', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[FieldStatusGroup]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [FieldStatusVariantId] bigint NOT NULL,                                                                       -- Owning variant
        [FieldStatusGroup]     nvarchar(4) NOT NULL,                                                                  -- Group key, e.g. G001
        [Name]                 nvarchar(60) NOT NULL,                                                                 -- Description
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_FieldStatusGroup_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_FieldStatusGroup_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_FieldStatusGroup] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_FieldStatusGroup] UNIQUE ([TenantId], [FieldStatusVariantId], [FieldStatusGroup])
    );
END
GO

/* cfg.FieldStatusVariant - Field status variant (reference: T004V) */
IF OBJECT_ID(N'cfg.FieldStatusVariant', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[FieldStatusVariant]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [FieldStatusVariant] nvarchar(4) NOT NULL,                                                                    -- Variant key
        [Name]               nvarchar(40) NOT NULL,                                                                   -- Description
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_FieldStatusVariant_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_FieldStatusVariant_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_FieldStatusVariant] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_FieldStatusVariant] UNIQUE ([TenantId], [FieldStatusVariant])
    );
END
GO

/* cfg.FinancialStatementNode - Hierarchy node of a financial statement version (reference: T011 structure) */
IF OBJECT_ID(N'cfg.FinancialStatementNode', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[FinancialStatementNode]
    (
        [Id]                          bigint IDENTITY(1,1) NOT NULL,                                                  -- Surrogate key
        [TenantId]                    int NOT NULL,                                                                   -- Owning tenant - every query is filtered by it
        [FinancialStatementVersionId] bigint NOT NULL,                                                                -- Owning version
        [NodeCode]                    nvarchar(20) NOT NULL,                                                          -- Node key
        [ParentNodeId]                bigint NULL,                                                                    -- Parent node
        [NodeText]                    nvarchar(60) NOT NULL,                                                          -- Node description
        [DisplayOrder]                int NOT NULL,                                                                   -- Sort order among siblings
        [NodeType]                    nvarchar(20) NOT NULL,                                                          -- Header, Total, AccountGroup, NotAssigned, PLResult
        [Section]                     nvarchar(20) NOT NULL,                                                          -- Assets, Liabilities, Income, Expense
        [DebitCreditShift]            bit NOT NULL CONSTRAINT [DF_cfg_FinancialStatementNode_DebitCreditShift] DEFAULT (0),-- Move balance to the opposite side when the sign flips
        [TotalNodeCode]               nvarchar(20) NULL,                                                              -- Node receiving the total
        [CreatedAt]                   datetime2(3) NOT NULL CONSTRAINT [DF_cfg_FinancialStatementNode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                   nvarchar(64) NOT NULL,                                                          -- Creating user name
        [ModifiedAt]                  datetime2(3) NULL,                                                              -- Last change timestamp (UTC)
        [ModifiedBy]                  nvarchar(64) NULL,                                                              -- Last changing user name
        [RowVersion]                  rowversion NOT NULL,                                                            -- Optimistic concurrency token
        [IsActive]                    bit NOT NULL CONSTRAINT [DF_cfg_FinancialStatementNode_IsActive] DEFAULT (1),   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_FinancialStatementNode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_FinancialStatementNode] UNIQUE ([TenantId], [FinancialStatementVersionId], [NodeCode])
    );
END
GO

/* cfg.FinancialStatementNodeAccount - Accounts assigned to a statement node (reference: T011 account assignment) */
IF OBJECT_ID(N'cfg.FinancialStatementNodeAccount', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[FinancialStatementNodeAccount]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [FinancialStatementNodeId] bigint NOT NULL,                                                                   -- Node
        [FromGLAccount]            nvarchar(10) NOT NULL,                                                             -- Lower account limit
        [ToGLAccount]              nvarchar(10) NOT NULL,                                                             -- Upper account limit
        [BalanceSide]              nvarchar(10) NOT NULL,                                                             -- Debit, Credit, Both
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_cfg_FinancialStatementNodeAccount_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_cfg_FinancialStatementNodeAccount_IsActive] DEFAULT (1),-- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_FinancialStatementNodeAccount] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_FinancialStatementNodeAccount] UNIQUE ([TenantId], [FinancialStatementNodeId], [FromGLAccount])
    );
END
GO

/* cfg.FinancialStatementVersion - Financial statement version (reference: T011) */
IF OBJECT_ID(N'cfg.FinancialStatementVersion', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[FinancialStatementVersion]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [FinancialStatementVersion] nvarchar(4) NOT NULL,                                                             -- Version key
        [Name]                      nvarchar(60) NOT NULL,                                                            -- Description
        [ChartOfAccountsId]         bigint NULL,                                                                      -- Chart of accounts (null = account-group based)
        [MaintenanceLanguage]       nvarchar(2) NOT NULL,                                                             -- Language of node texts
        [IsGroupAccountVersion]     bit NOT NULL CONSTRAINT [DF_cfg_FinancialStatementVersion_IsGroupAccountVersion] DEFAULT (0),-- Built on group accounts
        [AccountingPrincipleId]     bigint NULL,                                                                      -- Accounting principle presented
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_cfg_FinancialStatementVersion_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_cfg_FinancialStatementVersion_IsActive] DEFAULT (1),  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_FinancialStatementVersion] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_FinancialStatementVersion] UNIQUE ([TenantId], [FinancialStatementVersion])
    );
END
GO

/* cfg.FiscalPeriod -  */
IF OBJECT_ID(N'cfg.FiscalPeriod', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[FiscalPeriod]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [FiscalYearVariantId] bigint NOT NULL,                                                                        -- Fiscal year variant
        [FiscalYear]          smallint NOT NULL,                                                                      -- Fiscal year
        [FiscalPeriod]        tinyint NOT NULL,                                                                       -- Posting period 1..16
        [PeriodStartDate]     date NOT NULL,                                                                          -- First calendar day of the period
        [PeriodEndDate]       date NOT NULL,                                                                          -- Last calendar day of the period
        [IsSpecialPeriod]     bit NOT NULL CONSTRAINT [DF_cfg_FiscalPeriod_IsSpecialPeriod] DEFAULT (0),              -- Special (year-end adjustment) period
        [PeriodStatus]        nvarchar(20) NOT NULL,                                                                  -- Open, ClosedForEntry, Closed
        [ClosedAt]            datetime2(3) NULL,                                                                      -- Period-close timestamp
        [ClosedBy]            nvarchar(64) NULL,                                                                      -- User who closed the period
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_cfg_FiscalPeriod_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_cfg_FiscalPeriod_IsActive] DEFAULT (1),                     -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_FiscalPeriod] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_FiscalPeriod] UNIQUE ([TenantId], [FiscalYearVariantId], [FiscalYear], [FiscalPeriod])
    );
END
GO

/* cfg.FiscalYearVariant - Fiscal year variant (reference: T009) */
IF OBJECT_ID(N'cfg.FiscalYearVariant', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[FiscalYearVariant]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [FiscalYearVariant]      nvarchar(2) NOT NULL,                                                                -- Variant key, e.g. K4
        [Name]                   nvarchar(40) NOT NULL,                                                               -- Description
        [NumberOfPostingPeriods] tinyint NOT NULL,                                                                    -- Normal periods (1-12)
        [NumberOfSpecialPeriods] tinyint NOT NULL,                                                                    -- Special periods (0-4)
        [IsCalendarYear]         bit NOT NULL CONSTRAINT [DF_cfg_FiscalYearVariant_IsCalendarYear] DEFAULT (0),       -- Periods equal calendar months
        [IsYearDependent]        bit NOT NULL CONSTRAINT [DF_cfg_FiscalYearVariant_IsYearDependent] DEFAULT (0),      -- Period boundaries differ per year
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_FiscalYearVariant_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_FiscalYearVariant_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_FiscalYearVariant] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_FiscalYearVariant] UNIQUE ([TenantId], [FiscalYearVariant])
    );
END
GO

/* cfg.FiscalYearVariantPeriod - Period boundaries of a fiscal year variant (reference: T009B) */
IF OBJECT_ID(N'cfg.FiscalYearVariantPeriod', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[FiscalYearVariantPeriod]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [FiscalYearVariantId] bigint NOT NULL,                                                                        -- Owning variant
        [CalendarYear]        smallint NOT NULL,                                                                      -- Year (0000 when year-independent)
        [CalendarMonth]       tinyint NOT NULL,                                                                       -- Month of the period end
        [CalendarDay]         tinyint NOT NULL,                                                                       -- Day of the period end
        [FiscalPeriod]        tinyint NOT NULL,                                                                       -- Resulting posting period
        [YearShift]           smallint NOT NULL,                                                                      -- -1, 0, +1 fiscal year offset
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_cfg_FiscalYearVariantPeriod_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_cfg_FiscalYearVariantPeriod_IsActive] DEFAULT (1),          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_FiscalYearVariantPeriod] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_FiscalYearVariantPeriod] UNIQUE ([TenantId], [FiscalYearVariantId], [CalendarYear], [CalendarMonth], [CalendarDay])
    );
END
GO

/* cfg.Language - Language (reference: T002) */
IF OBJECT_ID(N'cfg.Language', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[Language]
    (
        [Id]            bigint IDENTITY(1,1) NOT NULL,                                                                -- Surrogate key
        [LanguageCode]  nvarchar(2) NOT NULL,                                                                         -- Language key, e.g. EN, KM
        [IsoCode]       nvarchar(5) NOT NULL,                                                                         -- Culture code, e.g. en-US, km-KH
        [Name]          nvarchar(40) NOT NULL,                                                                        -- Language name
        [IsRightToLeft] bit NOT NULL CONSTRAINT [DF_cfg_Language_IsRightToLeft] DEFAULT (0),                          -- Right-to-left script
        [IsActive]      bit NOT NULL CONSTRAINT [DF_cfg_Language_IsActive] DEFAULT (1),                               -- Available in the UI
        [CreatedAt]     datetime2(3) NOT NULL CONSTRAINT [DF_cfg_Language_CreatedAt] DEFAULT (SYSUTCDATETIME()),      -- Creation timestamp (UTC)
        [CreatedBy]     nvarchar(64) NOT NULL,                                                                        -- Creating user name
        [ModifiedAt]    datetime2(3) NULL,                                                                            -- Last change timestamp (UTC)
        [ModifiedBy]    nvarchar(64) NULL,                                                                            -- Last changing user name
        [RowVersion]    rowversion NOT NULL,                                                                          -- Optimistic concurrency token
        CONSTRAINT [PK_cfg_Language] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_Language] UNIQUE ([LanguageCode])
    );
END
GO

/* cfg.Ledger - Ledger of the universal journal (leading + parallel) (reference: T881) */
IF OBJECT_ID(N'cfg.Ledger', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[Ledger]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [Ledger]                nvarchar(2) NOT NULL,                                                                 -- Ledger key, e.g. 0L, 2L
        [Name]                  nvarchar(40) NOT NULL,                                                                -- Description
        [IsLeading]             bit NOT NULL CONSTRAINT [DF_cfg_Ledger_IsLeading] DEFAULT (0),                        -- Leading ledger - exactly one per tenant
        [AccountingPrincipleId] bigint NOT NULL,                                                                      -- Accounting principle represented
        [LedgerType]            nvarchar(10) NOT NULL,                                                                -- Standard, Extension, Appendix
        [IsExtensionLedger]     bit NOT NULL CONSTRAINT [DF_cfg_Ledger_IsExtensionLedger] DEFAULT (0),                -- Postings are deltas on the underlying ledger
        [UnderlyingLedgerId]    bigint NULL,                                                                          -- Base ledger for an extension ledger
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_cfg_Ledger_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_cfg_Ledger_IsActive] DEFAULT (1),                         -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_Ledger] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_Ledger] UNIQUE ([TenantId], [Ledger])
    );
END
GO

/* cfg.LedgerCompanyCode - Company code settings per ledger (reference: T882G) */
IF OBJECT_ID(N'cfg.LedgerCompanyCode', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[LedgerCompanyCode]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [LedgerId]               bigint NOT NULL,                                                                     -- Ledger
        [CompanyCodeId]          bigint NOT NULL,                                                                     -- Company code
        [FiscalYearVariantId]    bigint NOT NULL,                                                                     -- Fiscal year variant for this ledger
        [PostingPeriodVariantId] bigint NOT NULL,                                                                     -- Posting period variant for this ledger
        [Currency1TypeCode]      nvarchar(2) NOT NULL,                                                                -- Currency type of amount 1 (10 local)
        [Currency2TypeCode]      nvarchar(2) NULL,                                                                    -- Currency type of amount 2 (30 group)
        [Currency3TypeCode]      nvarchar(2) NULL,                                                                    -- Currency type of amount 3 (40 hard / 50 index)
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_LedgerCompanyCode_IsActive] DEFAULT (1),             -- Ledger active for this company code
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_LedgerCompanyCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        CONSTRAINT [PK_cfg_LedgerCompanyCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_LedgerCompanyCode] UNIQUE ([TenantId], [LedgerId], [CompanyCodeId])
    );
END
GO

/* cfg.MigrationScript - Reviewed migration generated by activation - executed only by CI/CD */
IF OBJECT_ID(N'cfg.MigrationScript', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[MigrationScript]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [MigrationName]         nvarchar(128) NOT NULL,                                                               -- Migration name, e.g. 20260805_AddZCaneContract
        [DictionaryObjectId]    bigint NULL,                                                                          -- Object that produced the migration
        [CustomObjectRequestId] bigint NULL,                                                                          -- Change request
        [UpScript]              nvarchar(max) NOT NULL,                                                               -- Forward DDL
        [DownScript]            nvarchar(max) NULL,                                                                   -- Rollback DDL
        [GeneratedAt]           datetime2(3) NOT NULL,                                                                -- Generation timestamp (UTC)
        [GeneratedBy]           nvarchar(64) NOT NULL,                                                                -- Generating user
        [ReviewedBy]            nvarchar(64) NULL,                                                                    -- Reviewer
        [ReviewedAt]            datetime2(3) NULL,                                                                    -- Review timestamp (UTC)
        [AppliedEnvironment]    nvarchar(20) NULL,                                                                    -- Environment where it was applied
        [AppliedAt]             datetime2(3) NULL,                                                                    -- Application timestamp (UTC)
        [Checksum]              nvarchar(64) NOT NULL,                                                                -- SHA-256 of the script - tamper detection
        [Status]                nvarchar(20) NOT NULL,                                                                -- Generated, Reviewed, Approved, Applied, Failed, RolledBack
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_cfg_MigrationScript_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_cfg_MigrationScript_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_MigrationScript] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_MigrationScript] UNIQUE ([TenantId], [MigrationName])
    );
END
GO

/* cfg.NumberRangeGap -  */
IF OBJECT_ID(N'cfg.NumberRangeGap', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[NumberRangeGap]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [NumberRangeIntervalId] bigint NOT NULL,                                                                      -- Interval concerned
        [MissingNumber]         bigint NOT NULL,                                                                      -- Number drawn but never committed
        [DetectedAt]            datetime2(3) NOT NULL,                                                                -- Detection timestamp (UTC)
        [Reason]                nvarchar(255) NULL,                                                                   -- Rollback reason if known
        [CorrelationId]         uniqueidentifier NULL,                                                                -- Request that lost the number
        CONSTRAINT [PK_cfg_NumberRangeGap] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* cfg.NumberRangeInterval - Number range interval and its current level (reference: NRIV) */
IF OBJECT_ID(N'cfg.NumberRangeInterval', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[NumberRangeInterval]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [NumberRangeObjectId] bigint NOT NULL,                                                                        -- Number range object
        [NumberRangeCode]     nvarchar(2) NOT NULL,                                                                   -- Interval key, e.g. 01
        [CompanyCodeId]       bigint NULL,                                                                            -- Company code (null = all)
        [FiscalYear]          smallint NOT NULL,                                                                      -- Fiscal year (0 = year-independent)
        [FromNumber]          bigint NOT NULL,                                                                        -- Lower limit
        [ToNumber]            bigint NOT NULL,                                                                        -- Upper limit
        [CurrentNumber]       bigint NOT NULL,                                                                        -- Last number issued - incremented under UPDLOCK
        [IsExternal]          bit NOT NULL CONSTRAINT [DF_cfg_NumberRangeInterval_IsExternal] DEFAULT (0),            -- External numbering (no automatic assignment)
        [IsBlocked]           bit NOT NULL CONSTRAINT [DF_cfg_NumberRangeInterval_IsBlocked] DEFAULT (0),             -- Interval blocked
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_cfg_NumberRangeInterval_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_cfg_NumberRangeInterval_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_NumberRangeInterval] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_NumberRangeInterval] UNIQUE ([TenantId], [NumberRangeObjectId], [NumberRangeCode], [CompanyCodeId], [FiscalYear])
    );
END
GO

/* cfg.NumberRangeObject - Number range object (reference: TNRO) */
IF OBJECT_ID(N'cfg.NumberRangeObject', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[NumberRangeObject]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [NumberRangeObject]      nvarchar(20) NOT NULL,                                                               -- Object key, e.g. RF_BELEG, BP, ASSET
        [Name]                   nvarchar(60) NOT NULL,                                                               -- Description
        [IsYearDependent]        bit NOT NULL CONSTRAINT [DF_cfg_NumberRangeObject_IsYearDependent] DEFAULT (0),      -- Intervals are per fiscal year
        [IsCompanyCodeDependent] bit NOT NULL CONSTRAINT [DF_cfg_NumberRangeObject_IsCompanyCodeDependent] DEFAULT (0),-- Intervals are per company code
        [NumberLength]           tinyint NOT NULL,                                                                    -- Number of digits, zero padded
        [Prefix]                 nvarchar(10) NULL,                                                                   -- Static prefix, e.g. KSS
        [NumberFormat]           nvarchar(60) NULL,                                                                   -- Format mask, e.g. {Prefix}-{Year}-{Type}-{Number:0000000000}
        [WarnPercent]            tinyint NULL,                                                                        -- Warn when this % of the interval is used
        [GapMonitoring]          bit NOT NULL CONSTRAINT [DF_cfg_NumberRangeObject_GapMonitoring] DEFAULT (0),        -- Detect and log number gaps
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_NumberRangeObject_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_NumberRangeObject_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_NumberRangeObject] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_NumberRangeObject] UNIQUE ([TenantId], [NumberRangeObject])
    );
END
GO

/* cfg.PaymentMethod - Payment method (reference: T042Z) */
IF OBJECT_ID(N'cfg.PaymentMethod', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[PaymentMethod]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [CountryCode]          nvarchar(3) NOT NULL,                                                                  -- Country
        [PaymentMethod]        nvarchar(1) NOT NULL,                                                                  -- Method key, e.g. T, C, B
        [Name]                 nvarchar(60) NOT NULL,                                                                 -- Description
        [PaymentType]          nvarchar(20) NOT NULL,                                                                 -- BankTransfer, Check, Cash, Card, Draft
        [IsForOutgoing]        bit NOT NULL CONSTRAINT [DF_cfg_PaymentMethod_IsForOutgoing] DEFAULT (0),              -- Outgoing payments
        [IsForIncoming]        bit NOT NULL CONSTRAINT [DF_cfg_PaymentMethod_IsForIncoming] DEFAULT (0),              -- Incoming payments
        [RequireBankDetails]   bit NOT NULL CONSTRAINT [DF_cfg_PaymentMethod_RequireBankDetails] DEFAULT (0),         -- Bank details mandatory
        [AllowForeignCurrency] bit NOT NULL CONSTRAINT [DF_cfg_PaymentMethod_AllowForeignCurrency] DEFAULT (0),       -- Foreign currency permitted
        [AllowForeignBank]     bit NOT NULL CONSTRAINT [DF_cfg_PaymentMethod_AllowForeignBank] DEFAULT (0),           -- Foreign bank permitted
        [PaymentFileFormat]    nvarchar(20) NULL,                                                                     -- ISO20022, CSV, ACH, Local
        [DocumentTypeId]       bigint NULL,                                                                           -- Document type used by the payment run
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_PaymentMethod_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_PaymentMethod_IsActive] DEFAULT (1),                   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_PaymentMethod] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_PaymentMethod] UNIQUE ([TenantId], [CountryCode], [PaymentMethod])
    );
END
GO

/* cfg.PaymentTerms - Payment terms (reference: T052) */
IF OBJECT_ID(N'cfg.PaymentTerms', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[PaymentTerms]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [PaymentTerms]         nvarchar(4) NOT NULL,                                                                  -- Terms key, e.g. 0001, NT30
        [Name]                 nvarchar(60) NOT NULL,                                                                 -- Description
        [BaselineDateRule]     nvarchar(20) NOT NULL,                                                                 -- DocumentDate, PostingDate, EntryDate, NoDefault
        [AdditionalDays]       int NOT NULL,                                                                          -- Days added to the baseline date
        [FixedDay]             tinyint NULL,                                                                          -- Fixed calendar day for the due date
        [AdditionalMonths]     tinyint NULL,                                                                          -- Months added
        [NetDueDays]           int NOT NULL,                                                                          -- Days until the net amount is due
        [CashDiscount1Days]    int NULL,                                                                              -- Days for discount level 1
        [CashDiscount1Percent] decimal(9,4) NULL,                                                                     -- Discount percentage level 1
        [CashDiscount2Days]    int NULL,                                                                              -- Days for discount level 2
        [CashDiscount2Percent] decimal(9,4) NULL,                                                                     -- Discount percentage level 2
        [IsForCustomer]        bit NOT NULL CONSTRAINT [DF_cfg_PaymentTerms_IsForCustomer] DEFAULT (0),               -- Valid for customers
        [IsForVendor]          bit NOT NULL CONSTRAINT [DF_cfg_PaymentTerms_IsForVendor] DEFAULT (0),                 -- Valid for vendors
        [IsInstallmentPlan]    bit NOT NULL CONSTRAINT [DF_cfg_PaymentTerms_IsInstallmentPlan] DEFAULT (0),           -- Split into instalments
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_PaymentTerms_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_PaymentTerms_IsActive] DEFAULT (1),                    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_PaymentTerms] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_PaymentTerms] UNIQUE ([TenantId], [PaymentTerms])
    );
END
GO

/* cfg.PaymentTermsInstallment - Instalment plan lines (reference: T052S) */
IF OBJECT_ID(N'cfg.PaymentTermsInstallment', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[PaymentTermsInstallment]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [PaymentTermsId]            bigint NOT NULL,                                                                  -- Instalment payment terms
        [InstallmentNumber]         tinyint NOT NULL,                                                                 -- Sequence of the instalment
        [Percentage]                decimal(9,4) NOT NULL,                                                            -- Share of the invoice amount
        [InstallmentPaymentTermsId] bigint NOT NULL,                                                                  -- Payment terms applied to the instalment
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_cfg_PaymentTermsInstallment_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_cfg_PaymentTermsInstallment_IsActive] DEFAULT (1),    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_PaymentTermsInstallment] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_PaymentTermsInstallment] UNIQUE ([TenantId], [PaymentTermsId], [InstallmentNumber])
    );
END
GO

/* cfg.PostingKey - Posting key - debit/credit and account type per line (reference: TBSL) */
IF OBJECT_ID(N'cfg.PostingKey', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[PostingKey]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [PostingKey]           nvarchar(2) NOT NULL,                                                                  -- Posting key, e.g. 40, 50, 01, 31
        [Name]                 nvarchar(40) NOT NULL,                                                                 -- Description
        [DebitCreditIndicator] nvarchar(1) NOT NULL,                                                                  -- S debit, H credit
        [AccountType]          nvarchar(1) NOT NULL,                                                                  -- S, D, K, A, M
        [IsSalesRelated]       bit NOT NULL CONSTRAINT [DF_cfg_PostingKey_IsSalesRelated] DEFAULT (0),                -- Sales-related (updates customer statistics)
        [IsSpecialGLPosting]   bit NOT NULL CONSTRAINT [DF_cfg_PostingKey_IsSpecialGLPosting] DEFAULT (0),            -- Requires a special G/L indicator
        [IsReversalPostingKey] bit NOT NULL CONSTRAINT [DF_cfg_PostingKey_IsReversalPostingKey] DEFAULT (0),          -- Used only by reversals
        [PaymentTransaction]   bit NOT NULL CONSTRAINT [DF_cfg_PostingKey_PaymentTransaction] DEFAULT (0),            -- Payment-relevant
        [FieldStatusGroupId]   bigint NULL,                                                                           -- Field status of the line item
        [IsBlocked]            bit NOT NULL CONSTRAINT [DF_cfg_PostingKey_IsBlocked] DEFAULT (0),                     -- Blocked for entry
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_PostingKey_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_PostingKey_IsActive] DEFAULT (1),                      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_PostingKey] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_PostingKey] UNIQUE ([TenantId], [PostingKey])
    );
END
GO

/* cfg.PostingPeriodControl - Open periods per account type and authorisation group (OB52) (reference: T001B) */
IF OBJECT_ID(N'cfg.PostingPeriodControl', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[PostingPeriodControl]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [PostingPeriodVariantId] bigint NOT NULL,                                                                     -- Posting period variant
        [AccountType]            nvarchar(1) NOT NULL,                                                                -- + all, S G/L, D customer, K vendor, A asset, M material
        [FromAccount]            nvarchar(10) NOT NULL,                                                               -- Lower account limit
        [ToAccount]              nvarchar(10) NOT NULL,                                                               -- Upper account limit
        [FromPeriod1]            tinyint NOT NULL,                                                                    -- First open period, interval 1
        [FromYear1]              smallint NOT NULL,                                                                   -- Fiscal year, interval 1 from
        [ToPeriod1]              tinyint NOT NULL,                                                                    -- Last open period, interval 1
        [ToYear1]                smallint NOT NULL,                                                                   -- Fiscal year, interval 1 to
        [FromPeriod2]            tinyint NULL,                                                                        -- First open period, interval 2 (closing)
        [FromYear2]              smallint NULL,                                                                       -- Fiscal year, interval 2 from
        [ToPeriod2]              tinyint NULL,                                                                        -- Last open period, interval 2
        [ToYear2]                smallint NULL,                                                                       -- Fiscal year, interval 2 to
        [AuthorizationGroup]     nvarchar(4) NULL,                                                                    -- Group allowed to post in interval 2
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_cfg_PostingPeriodControl_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_cfg_PostingPeriodControl_IsActive] DEFAULT (1),          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_PostingPeriodControl] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_PostingPeriodControl] UNIQUE ([TenantId], [PostingPeriodVariantId], [AccountType], [FromAccount])
    );
END
GO

/* cfg.PostingPeriodVariant - Posting period variant (reference: T010O) */
IF OBJECT_ID(N'cfg.PostingPeriodVariant', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[PostingPeriodVariant]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [PostingPeriodVariant] nvarchar(4) NOT NULL,                                                                  -- Variant key
        [Name]                 nvarchar(40) NOT NULL,                                                                 -- Description
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_PostingPeriodVariant_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_PostingPeriodVariant_IsActive] DEFAULT (1),            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_PostingPeriodVariant] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_PostingPeriodVariant] UNIQUE ([TenantId], [PostingPeriodVariant])
    );
END
GO

/* cfg.Region - Region / province / state (reference: T005S) */
IF OBJECT_ID(N'cfg.Region', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[Region]
    (
        [Id]          bigint IDENTITY(1,1) NOT NULL,                                                                  -- Surrogate key
        [TenantId]    int NOT NULL,                                                                                   -- Owning tenant - every query is filtered by it
        [CountryCode] nvarchar(3) NOT NULL,                                                                           -- Country
        [RegionCode]  nvarchar(3) NOT NULL,                                                                           -- Region key
        [Name]        nvarchar(60) NOT NULL,                                                                          -- Region name
        [CreatedAt]   datetime2(3) NOT NULL CONSTRAINT [DF_cfg_Region_CreatedAt] DEFAULT (SYSUTCDATETIME()),          -- Creation timestamp (UTC)
        [CreatedBy]   nvarchar(64) NOT NULL,                                                                          -- Creating user name
        [ModifiedAt]  datetime2(3) NULL,                                                                              -- Last change timestamp (UTC)
        [ModifiedBy]  nvarchar(64) NULL,                                                                              -- Last changing user name
        [RowVersion]  rowversion NOT NULL,                                                                            -- Optimistic concurrency token
        [IsActive]    bit NOT NULL CONSTRAINT [DF_cfg_Region_IsActive] DEFAULT (1),                                   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_Region] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_Region] UNIQUE ([TenantId], [CountryCode], [RegionCode])
    );
END
GO

/* cfg.SpecialGLAccount - Reconciliation account per special G/L indicator (reference: T074) */
IF OBJECT_ID(N'cfg.SpecialGLAccount', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[SpecialGLAccount]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [ChartOfAccountsId]         bigint NOT NULL,                                                                  -- Chart of accounts
        [SpecialGLIndicatorId]      bigint NOT NULL,                                                                  -- Special G/L indicator
        [ReconciliationGLAccountId] bigint NOT NULL,                                                                  -- Normal reconciliation account
        [SpecialGLAccountId]        bigint NOT NULL,                                                                  -- Alternative reconciliation account
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_cfg_SpecialGLAccount_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_cfg_SpecialGLAccount_IsActive] DEFAULT (1),           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_SpecialGLAccount] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_SpecialGLAccount] UNIQUE ([TenantId], [ChartOfAccountsId], [SpecialGLIndicatorId], [ReconciliationGLAccountId])
    );
END
GO

/* cfg.SpecialGLIndicator - Special G/L indicator (down payments, guarantees) (reference: T074U) */
IF OBJECT_ID(N'cfg.SpecialGLIndicator', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[SpecialGLIndicator]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [AccountType]              nvarchar(1) NOT NULL,                                                              -- D customer, K vendor
        [SpecialGLIndicator]       nvarchar(1) NOT NULL,                                                              -- Indicator, e.g. A down payment, F request
        [Name]                     nvarchar(60) NOT NULL,                                                             -- Description
        [SpecialGLType]            nvarchar(20) NOT NULL,                                                             -- DownPayment, BillOfExchange, Other, Noted
        [IsNotedItem]              bit NOT NULL CONSTRAINT [DF_cfg_SpecialGLIndicator_IsNotedItem] DEFAULT (0),       -- Noted item (no G/L update)
        [IsFreeOffsettingEntry]    bit NOT NULL CONSTRAINT [DF_cfg_SpecialGLIndicator_IsFreeOffsettingEntry] DEFAULT (0),-- Free offsetting entry allowed
        [TargetSpecialGLIndicator] nvarchar(1) NULL,                                                                  -- Target indicator on transfer
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_cfg_SpecialGLIndicator_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_cfg_SpecialGLIndicator_IsActive] DEFAULT (1),          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_SpecialGLIndicator] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_SpecialGLIndicator] UNIQUE ([TenantId], [AccountType], [SpecialGLIndicator])
    );
END
GO

/* cfg.TableAuthorizationGroup - Authorization group controlling table access (reference: TBRG) */
IF OBJECT_ID(N'cfg.TableAuthorizationGroup', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[TableAuthorizationGroup]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [AuthorizationGroup] nvarchar(4) NOT NULL,                                                                    -- Group key, e.g. FI01, SEC0
        [Name]               nvarchar(60) NOT NULL,                                                                   -- Description
        [IsSystemProtected]  bit NOT NULL CONSTRAINT [DF_cfg_TableAuthorizationGroup_IsSystemProtected] DEFAULT (0),  -- System/security tables - browser access always denied
        [AllowExport]        bit NOT NULL CONSTRAINT [DF_cfg_TableAuthorizationGroup_AllowExport] DEFAULT (0),        -- Export permitted for this group
        [MaxRowsPerQuery]    int NOT NULL,                                                                            -- Row cap applied to browser queries
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_cfg_TableAuthorizationGroup_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_cfg_TableAuthorizationGroup_IsActive] DEFAULT (1),           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_TableAuthorizationGroup] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_TableAuthorizationGroup] UNIQUE ([TenantId], [AuthorizationGroup])
    );
END
GO

/* cfg.TaxCode - Tax code (reference: T007A) */
IF OBJECT_ID(N'cfg.TaxCode', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[TaxCode]
    (
        [Id]              bigint IDENTITY(1,1) NOT NULL,                                                              -- Surrogate key
        [TenantId]        int NOT NULL,                                                                               -- Owning tenant - every query is filtered by it
        [CountryCode]     nvarchar(3) NOT NULL,                                                                       -- Tax country
        [TaxCode]         nvarchar(2) NOT NULL,                                                                       -- Tax code key, e.g. V1, A1
        [Name]            nvarchar(60) NOT NULL,                                                                      -- Description
        [TaxType]         nvarchar(1) NOT NULL,                                                                       -- V input tax, A output tax
        [TaxCategory]     nvarchar(20) NOT NULL,                                                                      -- VAT, WithholdingTax, SalesTax, Exempt
        [IsReverseCharge] bit NOT NULL CONSTRAINT [DF_cfg_TaxCode_IsReverseCharge] DEFAULT (0),                       -- Reverse charge mechanism
        [IsNonDeductible] bit NOT NULL CONSTRAINT [DF_cfg_TaxCode_IsNonDeductible] DEFAULT (0),                       -- Non-deductible input tax
        [TargetTaxCode]   nvarchar(2) NULL,                                                                           -- Target code for deferred tax
        [CheckIndicator]  nvarchar(1) NULL,                                                                           -- Error / warning on tax deviation
        [IsBlocked]       bit NOT NULL CONSTRAINT [DF_cfg_TaxCode_IsBlocked] DEFAULT (0),                             -- Blocked for posting
        [ValidFrom]       date NOT NULL,                                                                              -- First day the record is valid
        [ValidTo]         date NOT NULL,                                                                              -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]       datetime2(3) NOT NULL CONSTRAINT [DF_cfg_TaxCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),     -- Creation timestamp (UTC)
        [CreatedBy]       nvarchar(64) NOT NULL,                                                                      -- Creating user name
        [ModifiedAt]      datetime2(3) NULL,                                                                          -- Last change timestamp (UTC)
        [ModifiedBy]      nvarchar(64) NULL,                                                                          -- Last changing user name
        [RowVersion]      rowversion NOT NULL,                                                                        -- Optimistic concurrency token
        [IsActive]        bit NOT NULL CONSTRAINT [DF_cfg_TaxCode_IsActive] DEFAULT (1),                              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_TaxCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_TaxCode] UNIQUE ([TenantId], [CountryCode], [TaxCode])
    );
END
GO

/* cfg.TaxCodeRate - Rate per tax code, condition and validity (reference: A003 / KONP) */
IF OBJECT_ID(N'cfg.TaxCodeRate', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[TaxCodeRate]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [TaxCodeId]           bigint NOT NULL,                                                                        -- Tax code
        [ConditionType]       nvarchar(4) NOT NULL,                                                                   -- Condition, e.g. MWVS, MWAS, NAVS
        [TaxJurisdictionCode] nvarchar(15) NULL,                                                                      -- Jurisdiction (when jurisdiction-based)
        [ValidFrom]           date NOT NULL,                                                                          -- Valid from
        [ValidTo]             date NOT NULL,                                                                          -- Valid to
        [RatePercent]         decimal(9,4) NOT NULL,                                                                  -- Tax rate in percent
        [TaxAccountKey]       nvarchar(3) NOT NULL,                                                                   -- Account key for determination, e.g. VST, MWS
        [IsDeductiblePortion] bit NOT NULL CONSTRAINT [DF_cfg_TaxCodeRate_IsDeductiblePortion] DEFAULT (0),           -- Deductible portion of the rate
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_cfg_TaxCodeRate_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_cfg_TaxCodeRate_IsActive] DEFAULT (1),                      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_TaxCodeRate] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_TaxCodeRate] UNIQUE ([TenantId], [TaxCodeId], [ConditionType], [TaxJurisdictionCode], [ValidFrom])
    );
END
GO

/* cfg.TaxJurisdiction - Tax jurisdiction (reference: TTXJ) */
IF OBJECT_ID(N'cfg.TaxJurisdiction', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[TaxJurisdiction]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [TaxJurisdictionCode]  nvarchar(15) NOT NULL,                                                                 -- Jurisdiction code
        [CountryCode]          nvarchar(3) NOT NULL,                                                                  -- Country
        [SchemaCode]           nvarchar(4) NOT NULL,                                                                  -- Jurisdiction schema
        [Name]                 nvarchar(60) NOT NULL,                                                                 -- Description
        [ParentJurisdictionId] bigint NULL,                                                                           -- Higher-level jurisdiction
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_TaxJurisdiction_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_TaxJurisdiction_IsActive] DEFAULT (1),                 -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_TaxJurisdiction] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_TaxJurisdiction] UNIQUE ([TenantId], [TaxJurisdictionCode])
    );
END
GO

/* cfg.ToleranceGroup - Posting and payment tolerances (reference: T043T / T043G) */
IF OBJECT_ID(N'cfg.ToleranceGroup', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[ToleranceGroup]
    (
        [Id]                           bigint IDENTITY(1,1) NOT NULL,                                                 -- Surrogate key
        [TenantId]                     int NOT NULL,                                                                  -- Owning tenant - every query is filtered by it
        [CompanyCodeId]                bigint NOT NULL,                                                               -- Company code
        [ToleranceGroup]               nvarchar(4) NOT NULL,                                                          -- Group key (blank = default)
        [ToleranceType]                nvarchar(20) NOT NULL,                                                         -- User, Customer, Vendor, GL
        [Name]                         nvarchar(60) NULL,                                                             -- Description
        [MaxDocumentAmount]            decimal(19,4) NULL,                                                            -- Maximum amount per document
        [MaxLineItemAmount]            decimal(19,4) NULL,                                                            -- Maximum amount per open item
        [MaxCashDiscountPercent]       decimal(9,4) NULL,                                                             -- Maximum cash discount
        [PaymentDifferenceGainAmount]  decimal(19,4) NULL,                                                            -- Permitted revenue from differences
        [PaymentDifferenceGainPercent] decimal(9,4) NULL,                                                             -- Permitted revenue in percent
        [PaymentDifferenceLossAmount]  decimal(19,4) NULL,                                                            -- Permitted expense from differences
        [PaymentDifferenceLossPercent] decimal(9,4) NULL,                                                             -- Permitted expense in percent
        [CreatedAt]                    datetime2(3) NOT NULL CONSTRAINT [DF_cfg_ToleranceGroup_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                    nvarchar(64) NOT NULL,                                                         -- Creating user name
        [ModifiedAt]                   datetime2(3) NULL,                                                             -- Last change timestamp (UTC)
        [ModifiedBy]                   nvarchar(64) NULL,                                                             -- Last changing user name
        [RowVersion]                   rowversion NOT NULL,                                                           -- Optimistic concurrency token
        [IsActive]                     bit NOT NULL CONSTRAINT [DF_cfg_ToleranceGroup_IsActive] DEFAULT (1),          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_ToleranceGroup] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_ToleranceGroup] UNIQUE ([TenantId], [CompanyCodeId], [ToleranceGroup], [ToleranceType])
    );
END
GO

/* cfg.UnitOfMeasure - Unit of measure (reference: T006) */
IF OBJECT_ID(N'cfg.UnitOfMeasure', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[UnitOfMeasure]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [UnitOfMeasure]    nvarchar(3) NOT NULL,                                                                      -- Internal unit key
        [IsoCode]          nvarchar(3) NULL,                                                                          -- ISO unit code
        [Name]             nvarchar(40) NOT NULL,                                                                     -- Unit name
        [Dimension]        nvarchar(10) NULL,                                                                         -- MASS, VOLUME, TIME, AREA
        [DecimalPlaces]    tinyint NOT NULL,                                                                          -- Decimals used for quantities
        [ConversionFactor] decimal(23,6) NULL,                                                                        -- Factor to the base unit of the dimension
        [CreatedAt]        datetime2(3) NOT NULL CONSTRAINT [DF_cfg_UnitOfMeasure_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]        nvarchar(64) NOT NULL,                                                                     -- Creating user name
        [ModifiedAt]       datetime2(3) NULL,                                                                         -- Last change timestamp (UTC)
        [ModifiedBy]       nvarchar(64) NULL,                                                                         -- Last changing user name
        [RowVersion]       rowversion NOT NULL,                                                                       -- Optimistic concurrency token
        [IsActive]         bit NOT NULL CONSTRAINT [DF_cfg_UnitOfMeasure_IsActive] DEFAULT (1),                       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_UnitOfMeasure] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_UnitOfMeasure] UNIQUE ([TenantId], [UnitOfMeasure])
    );
END
GO

/* cfg.WithholdingTaxCode - Withholding tax code and rate (reference: T059Z) */
IF OBJECT_ID(N'cfg.WithholdingTaxCode', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[WithholdingTaxCode]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [WithholdingTaxTypeId] bigint NOT NULL,                                                                       -- Withholding tax type
        [WithholdingTaxCode]   nvarchar(2) NOT NULL,                                                                  -- Code key
        [Name]                 nvarchar(60) NOT NULL,                                                                 -- Description
        [RatePercent]          decimal(9,4) NOT NULL,                                                                 -- Withholding rate
        [BasePercent]          decimal(9,4) NOT NULL,                                                                 -- Percentage of the base subject to tax
        [MinimumBaseAmount]    decimal(19,4) NULL,                                                                    -- Exemption threshold
        [GLAccountId]          bigint NULL,                                                                           -- Withholding tax G/L account
        [ValidFrom]            date NOT NULL,                                                                         -- First day the record is valid
        [ValidTo]              date NOT NULL,                                                                         -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_WithholdingTaxCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_WithholdingTaxCode_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_WithholdingTaxCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_WithholdingTaxCode] UNIQUE ([TenantId], [WithholdingTaxTypeId], [WithholdingTaxCode])
    );
END
GO

/* cfg.WithholdingTaxType - Withholding tax type (reference: T059P) */
IF OBJECT_ID(N'cfg.WithholdingTaxType', N'U') IS NULL
BEGIN
    CREATE TABLE [cfg].[WithholdingTaxType]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [CountryCode]          nvarchar(3) NOT NULL,                                                                  -- Country
        [WithholdingTaxType]   nvarchar(2) NOT NULL,                                                                  -- Type key
        [Name]                 nvarchar(60) NOT NULL,                                                                 -- Description
        [PostingTime]          nvarchar(20) NOT NULL,                                                                 -- Invoice, Payment
        [BaseAmountType]       nvarchar(20) NOT NULL,                                                                 -- GrossAmount, NetAmount, TaxAmount
        [RoundingRule]         nvarchar(20) NOT NULL,                                                                 -- Commercial, Up, Down
        [IsAccumulationActive] bit NOT NULL CONSTRAINT [DF_cfg_WithholdingTaxType_IsAccumulationActive] DEFAULT (0),  -- Accumulate base amounts per year
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_cfg_WithholdingTaxType_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_cfg_WithholdingTaxType_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_cfg_WithholdingTaxType] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_cfg_WithholdingTaxType] UNIQUE ([TenantId], [CountryCode], [WithholdingTaxType])
    );
END
GO
