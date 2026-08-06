/* ============================================================================
   S/4HANA-inspired ERP - schema [fin]
   Financial accounting and asset accounting (36 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* fin.AccountBalance - Period balances per account and dimension - maintained by the posting engine (reference: GLT0 / FAGLFLEXT) */
IF OBJECT_ID(N'fin.AccountBalance', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[AccountBalance]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [LedgerId]          bigint NOT NULL,                                                                          -- Ledger
        [CompanyCodeId]     bigint NOT NULL,                                                                          -- Company code
        [FiscalYear]        smallint NOT NULL,                                                                        -- Fiscal year
        [FiscalPeriod]      tinyint NOT NULL,                                                                         -- Period (0 = carry-forward)
        [GLAccountId]       bigint NOT NULL,                                                                          -- G/L account
        [BusinessPartnerId] bigint NULL,                                                                              -- Customer / vendor for subledger balances
        [ProfitCenterId]    bigint NULL,                                                                              -- Profit centre
        [SegmentId]         bigint NULL,                                                                              -- Segment
        [FunctionalAreaId]  bigint NULL,                                                                              -- Functional area
        [BusinessAreaId]    bigint NULL,                                                                              -- Business area
        [CurrencyType]      nvarchar(2) NOT NULL,                                                                     -- 10 local, 30 group, 00 document
        [CurrencyCode]      nvarchar(5) NOT NULL,                                                                     -- Currency of the amounts
        [DebitTotal]        decimal(19,4) NOT NULL,                                                                   -- Total debits in the period
        [CreditTotal]       decimal(19,4) NOT NULL,                                                                   -- Total credits in the period
        [PeriodBalance]     decimal(19,4) NOT NULL,                                                                   -- Debits ? credits
        [CumulativeBalance] decimal(19,4) NOT NULL,                                                                   -- Balance including carry-forward
        [LastUpdatedAt]     datetime2(3) NOT NULL,                                                                    -- Last update (UTC)
        [RowVersion]        rowversion NOT NULL,                                                                      -- Concurrency token
        CONSTRAINT [PK_fin_AccountBalance] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_AccountBalance] UNIQUE ([TenantId], [LedgerId], [CompanyCodeId], [FiscalYear], [FiscalPeriod], [GLAccountId], [BusinessPartnerId], [ProfitCenterId], [SegmentId], [FunctionalAreaId], [BusinessAreaId], [CurrencyType], [CurrencyCode])
    );
END
GO

/* fin.Asset - Asset master record (reference: ANLA) */
IF OBJECT_ID(N'fin.Asset', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[Asset]
    (
        [Id]                         bigint IDENTITY(1,1) NOT NULL,                                                   -- Surrogate key
        [TenantId]                   int NOT NULL,                                                                    -- Owning tenant - every query is filtered by it
        [CompanyCodeId]              bigint NOT NULL,                                                                 -- Company code
        [AssetNumber]                nvarchar(12) NOT NULL,                                                           -- Main asset number
        [AssetSubNumber]             int NOT NULL,                                                                    -- Sub-number (0 = main asset)
        [AssetClassId]               bigint NOT NULL,                                                                 -- Asset class
        [Description]                nvarchar(60) NOT NULL,                                                           -- Asset description
        [Description2]               nvarchar(60) NULL,                                                               -- Additional description
        [SerialNumber]               nvarchar(40) NULL,                                                               -- Serial number
        [InventoryNumber]            nvarchar(25) NULL,                                                               -- Inventory number
        [Quantity]                   decimal(23,6) NULL,                                                              -- Quantity
        [UnitOfMeasure]              nvarchar(3) NULL,                                                                -- Unit of measure
        [CapitalizationDate]         date NULL,                                                                       -- Capitalisation date - starts depreciation
        [AcquisitionDate]            date NULL,                                                                       -- First acquisition date
        [InServiceDate]              date NULL,                                                                       -- Date placed in service
        [DeactivationDate]           date NULL,                                                                       -- Deactivation / retirement date
        [PlannedRetirementDate]      date NULL,                                                                       -- Planned retirement
        [CostCenterId]               bigint NULL,                                                                     -- Responsible cost centre
        [ProfitCenterId]             bigint NULL,                                                                     -- Profit centre
        [SegmentId]                  bigint NULL,                                                                     -- Segment
        [FunctionalAreaId]           bigint NULL,                                                                     -- Functional area
        [BusinessAreaId]             bigint NULL,                                                                     -- Business area
        [InternalOrderId]            bigint NULL,                                                                     -- Investment order
        [PlantId]                    bigint NULL,                                                                     -- Plant
        [LocationId]                 bigint NULL,                                                                     -- Location
        [RoomNumber]                 nvarchar(20) NULL,                                                               -- Room
        [ResponsiblePersonPartnerId] bigint NULL,                                                                     -- Person responsible (employee BP)
        [VendorBusinessPartnerId]    bigint NULL,                                                                     -- Supplying vendor
        [ManufacturerName]           nvarchar(60) NULL,                                                               -- Manufacturer
        [LicensePlateNumber]         nvarchar(20) NULL,                                                               -- Licence plate (vehicles)
        [IsAssetUnderConstruction]   bit NOT NULL CONSTRAINT [DF_fin_Asset_IsAssetUnderConstruction] DEFAULT (0),     -- Asset under construction
        [SettlementProfile]          nvarchar(6) NULL,                                                                -- Settlement profile for AuC
        [IsLowValueAsset]            bit NOT NULL CONSTRAINT [DF_fin_Asset_IsLowValueAsset] DEFAULT (0),              -- Low value asset
        [IsInvestmentSupport]        bit NOT NULL CONSTRAINT [DF_fin_Asset_IsInvestmentSupport] DEFAULT (0),          -- Investment support asset
        [Status]                     nvarchar(20) NOT NULL,                                                           -- Created, Capitalized, Active, Retired, Sold, Scrapped, Blocked, MarkedForDeletion
        [IsPostingBlocked]           bit NOT NULL CONSTRAINT [DF_fin_Asset_IsPostingBlocked] DEFAULT (0),             -- Blocked for postings
        [IsMarkedForDeletion]        bit NOT NULL CONSTRAINT [DF_fin_Asset_IsMarkedForDeletion] DEFAULT (0),          -- Deletion flag
        [LegacyAssetNumber]          nvarchar(20) NULL,                                                               -- Number in the legacy system
        [CreatedAt]                  datetime2(3) NOT NULL CONSTRAINT [DF_fin_Asset_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                  nvarchar(64) NOT NULL,                                                           -- Creating user name
        [ModifiedAt]                 datetime2(3) NULL,                                                               -- Last change timestamp (UTC)
        [ModifiedBy]                 nvarchar(64) NULL,                                                               -- Last changing user name
        [RowVersion]                 rowversion NOT NULL,                                                             -- Optimistic concurrency token
        [IsActive]                   bit NOT NULL CONSTRAINT [DF_fin_Asset_IsActive] DEFAULT (1),                     -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_Asset] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_Asset] UNIQUE ([TenantId], [CompanyCodeId], [AssetNumber], [AssetSubNumber])
    );
END
GO

/* fin.AssetClass - Asset class - defaults, number range and account determination (reference: ANKA) */
IF OBJECT_ID(N'fin.AssetClass', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[AssetClass]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [AssetClass]               nvarchar(8) NOT NULL,                                                              -- Asset class key, e.g. 2000 machinery
        [Name]                     nvarchar(60) NOT NULL,                                                             -- Description
        [AccountDeterminationKey]  nvarchar(8) NOT NULL,                                                              -- Account determination key
        [NumberRangeObjectId]      bigint NOT NULL,                                                                   -- Number range object
        [NumberRangeCode]          nvarchar(2) NOT NULL,                                                              -- Number range interval
        [IsExternalNumbering]      bit NOT NULL CONSTRAINT [DF_fin_AssetClass_IsExternalNumbering] DEFAULT (0),       -- Asset number entered by the user
        [ScreenLayoutKey]          nvarchar(4) NULL,                                                                  -- Screen layout for the master record
        [IsAssetUnderConstruction] bit NOT NULL CONSTRAINT [DF_fin_AssetClass_IsAssetUnderConstruction] DEFAULT (0),  -- Assets under construction class
        [IsLowValueAsset]          bit NOT NULL CONSTRAINT [DF_fin_AssetClass_IsLowValueAsset] DEFAULT (0),           -- Low value asset class
        [LowValueAmountLimit]      decimal(19,4) NULL,                                                                -- Low value threshold
        [DefaultUsefulLifeYears]   int NULL,                                                                          -- Default useful life in years
        [DefaultUsefulLifePeriods] int NULL,                                                                          -- Default useful life in periods
        [AllowSubNumbers]          bit NOT NULL CONSTRAINT [DF_fin_AssetClass_AllowSubNumbers] DEFAULT (0),           -- Sub-assets permitted
        [IsBlocked]                bit NOT NULL CONSTRAINT [DF_fin_AssetClass_IsBlocked] DEFAULT (0),                 -- Blocked for new assets
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_fin_AssetClass_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_fin_AssetClass_IsActive] DEFAULT (1),                  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_AssetClass] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_AssetClass] UNIQUE ([TenantId], [AssetClass])
    );
END
GO

/* fin.AssetClassDepreciationArea - Default depreciation settings per asset class and area (reference: ANKB) */
IF OBJECT_ID(N'fin.AssetClassDepreciationArea', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[AssetClassDepreciationArea]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [AssetClassId]       bigint NOT NULL,                                                                         -- Asset class
        [DepreciationAreaId] bigint NOT NULL,                                                                         -- Depreciation area
        [DepreciationKeyId]  bigint NOT NULL,                                                                         -- Default depreciation key
        [UsefulLifeYears]    int NULL,                                                                                -- Default useful life in years
        [UsefulLifePeriods]  int NULL,                                                                                -- Default useful life in periods
        [IsAreaDeactivated]  bit NOT NULL CONSTRAINT [DF_fin_AssetClassDepreciationArea_IsAreaDeactivated] DEFAULT (0),-- Area not used for this class
        [ScrapValuePercent]  decimal(9,4) NULL,                                                                       -- Default scrap value percentage
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_fin_AssetClassDepreciationArea_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_fin_AssetClassDepreciationArea_IsActive] DEFAULT (1),        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_AssetClassDepreciationArea] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_AssetClassDepreciationArea] UNIQUE ([TenantId], [AssetClassId], [DepreciationAreaId])
    );
END
GO

/* fin.AssetDepreciationArea - Depreciation parameters of an asset per area (reference: ANLB) */
IF OBJECT_ID(N'fin.AssetDepreciationArea', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[AssetDepreciationArea]
    (
        [Id]                           bigint IDENTITY(1,1) NOT NULL,                                                 -- Surrogate key
        [TenantId]                     int NOT NULL,                                                                  -- Owning tenant - every query is filtered by it
        [AssetId]                      bigint NOT NULL,                                                               -- Asset
        [DepreciationAreaId]           bigint NOT NULL,                                                               -- Depreciation area
        [DepreciationKeyId]            bigint NOT NULL,                                                               -- Depreciation key
        [UsefulLifeYears]              int NOT NULL,                                                                  -- Useful life in years
        [UsefulLifePeriods]            int NOT NULL,                                                                  -- Additional periods
        [ExpiredUsefulLifeYears]       int NOT NULL,                                                                  -- Expired years
        [ExpiredUsefulLifePeriods]     int NOT NULL,                                                                  -- Expired periods
        [DepreciationStartDate]        date NULL,                                                                     -- Ordinary depreciation start
        [SpecialDepreciationStartDate] date NULL,                                                                     -- Special depreciation start
        [ScrapValue]                   decimal(19,4) NULL,                                                            -- Scrap value
        [ScrapValuePercent]            decimal(9,4) NULL,                                                             -- Scrap value percentage
        [IsDeactivated]                bit NOT NULL CONSTRAINT [DF_fin_AssetDepreciationArea_IsDeactivated] DEFAULT (0),-- Area deactivated for this asset
        [IsManualDepreciation]         bit NOT NULL CONSTRAINT [DF_fin_AssetDepreciationArea_IsManualDepreciation] DEFAULT (0),-- Depreciation entered manually
        [IndexSeries]                  nvarchar(4) NULL,                                                              -- Index series for replacement values
        [CreatedAt]                    datetime2(3) NOT NULL CONSTRAINT [DF_fin_AssetDepreciationArea_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                    nvarchar(64) NOT NULL,                                                         -- Creating user name
        [ModifiedAt]                   datetime2(3) NULL,                                                             -- Last change timestamp (UTC)
        [ModifiedBy]                   nvarchar(64) NULL,                                                             -- Last changing user name
        [RowVersion]                   rowversion NOT NULL,                                                           -- Optimistic concurrency token
        [IsActive]                     bit NOT NULL CONSTRAINT [DF_fin_AssetDepreciationArea_IsActive] DEFAULT (1),   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_AssetDepreciationArea] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_AssetDepreciationArea] UNIQUE ([TenantId], [AssetId], [DepreciationAreaId])
    );
END
GO

/* fin.AssetTimeDependent - Time-dependent asset assignments (reference: ANLZ) */
IF OBJECT_ID(N'fin.AssetTimeDependent', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[AssetTimeDependent]
    (
        [Id]              bigint IDENTITY(1,1) NOT NULL,                                                              -- Surrogate key
        [TenantId]        int NOT NULL,                                                                               -- Owning tenant - every query is filtered by it
        [AssetId]         bigint NOT NULL,                                                                            -- Asset
        [ValidFrom]       date NOT NULL,                                                                              -- Valid from
        [ValidTo]         date NOT NULL,                                                                              -- Valid to
        [CostCenterId]    bigint NULL,                                                                                -- Cost centre in this interval
        [ProfitCenterId]  bigint NULL,                                                                                -- Profit centre
        [SegmentId]       bigint NULL,                                                                                -- Segment
        [InternalOrderId] bigint NULL,                                                                                -- Internal order
        [PlantId]         bigint NULL,                                                                                -- Plant
        [LocationId]      bigint NULL,                                                                                -- Location
        [IsShutdown]      bit NOT NULL CONSTRAINT [DF_fin_AssetTimeDependent_IsShutdown] DEFAULT (0),                 -- Asset shut down - depreciation suspended
        [ShiftFactor]     decimal(9,4) NULL,                                                                          -- Multiple-shift factor
        [CreatedAt]       datetime2(3) NOT NULL CONSTRAINT [DF_fin_AssetTimeDependent_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]       nvarchar(64) NOT NULL,                                                                      -- Creating user name
        [ModifiedAt]      datetime2(3) NULL,                                                                          -- Last change timestamp (UTC)
        [ModifiedBy]      nvarchar(64) NULL,                                                                          -- Last changing user name
        [RowVersion]      rowversion NOT NULL,                                                                        -- Optimistic concurrency token
        [IsActive]        bit NOT NULL CONSTRAINT [DF_fin_AssetTimeDependent_IsActive] DEFAULT (1),                   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_AssetTimeDependent] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_AssetTimeDependent] UNIQUE ([TenantId], [AssetId], [ValidFrom])
    );
END
GO

/* fin.AssetTransaction - Asset transaction (acquisition, retirement, transfer, write-up) (reference: ANEP) */
IF OBJECT_ID(N'fin.AssetTransaction', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[AssetTransaction]
    (
        [Id]                             bigint IDENTITY(1,1) NOT NULL,                                               -- Surrogate key
        [TenantId]                       int NOT NULL,                                                                -- Owning tenant - every query is filtered by it
        [CompanyCodeId]                  bigint NOT NULL,                                                             -- Company code
        [AssetId]                        bigint NOT NULL,                                                             -- Asset
        [FiscalYear]                     smallint NOT NULL,                                                           -- Fiscal year
        [AssetDocumentNumber]            nvarchar(20) NOT NULL,                                                       -- Asset document number
        [LineItemNumber]                 int NOT NULL,                                                                -- Line number
        [DepreciationAreaId]             bigint NOT NULL,                                                             -- Depreciation area
        [TransactionTypeId]              bigint NOT NULL,                                                             -- Transaction type
        [PostingDate]                    date NOT NULL,                                                               -- Posting date
        [DocumentDate]                   date NOT NULL,                                                               -- Document date
        [AssetValueDate]                 date NOT NULL,                                                               -- Asset value date - drives period control
        [FiscalPeriod]                   tinyint NOT NULL,                                                            -- Period
        [CurrencyCode]                   nvarchar(5) NOT NULL,                                                        -- Transaction currency
        [TransactionAmount]              decimal(19,4) NOT NULL,                                                      -- Amount in transaction currency
        [AmountInLocalCurrency]          decimal(19,4) NOT NULL,                                                      -- Amount in local currency
        [AmountInAreaCurrency]           decimal(19,4) NOT NULL,                                                      -- Amount in the area currency
        [Quantity]                       decimal(23,6) NULL,                                                          -- Quantity retired or acquired
        [AccumulatedDepreciationRetired] decimal(19,4) NULL,                                                          -- Accumulated depreciation removed on retirement
        [RevenueAmount]                  decimal(19,4) NULL,                                                          -- Sale revenue
        [GainLossAmount]                 decimal(19,4) NULL,                                                          -- Gain or loss on retirement
        [PercentRetired]                 decimal(9,4) NULL,                                                           -- Percentage retired
        [PartnerBusinessPartnerId]       bigint NULL,                                                                 -- Vendor or customer involved
        [TargetAssetId]                  bigint NULL,                                                                 -- Receiving asset on transfer
        [JournalEntryHeaderId]           bigint NOT NULL,                                                             -- Accounting document created
        [ReferenceDocumentNumber]        nvarchar(20) NULL,                                                           -- Reference document
        [Text]                           nvarchar(255) NULL,                                                          -- Transaction text
        [IsReversed]                     bit NOT NULL CONSTRAINT [DF_fin_AssetTransaction_IsReversed] DEFAULT (0),    -- Reversed
        [ReversalAssetDocumentNumber]    nvarchar(20) NULL,                                                           -- Reversing asset document
        [CreatedAt]                      datetime2(3) NOT NULL CONSTRAINT [DF_fin_AssetTransaction_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                      nvarchar(64) NOT NULL,                                                       -- Creating user
        CONSTRAINT [PK_fin_AssetTransaction] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_AssetTransaction] UNIQUE ([TenantId], [CompanyCodeId], [AssetId], [FiscalYear], [AssetDocumentNumber], [LineItemNumber], [DepreciationAreaId])
    );
END
GO

/* fin.AssetTransactionType - Asset transaction type (reference: TABW) */
IF OBJECT_ID(N'fin.AssetTransactionType', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[AssetTransactionType]
    (
        [Id]                             bigint IDENTITY(1,1) NOT NULL,                                               -- Surrogate key
        [TenantId]                       int NOT NULL,                                                                -- Owning tenant - every query is filtered by it
        [TransactionType]                nvarchar(3) NOT NULL,                                                        -- Type key, e.g. 100 acquisition, 200 retirement
        [Name]                           nvarchar(60) NOT NULL,                                                       -- Description
        [TransactionCategory]            nvarchar(20) NOT NULL,                                                       -- Acquisition, Retirement, Transfer, WriteUp, PostCapitalization, Depreciation, Impairment
        [DebitCreditIndicator]           nvarchar(1) NOT NULL,                                                        -- S debit, H credit
        [AffectsAcquisitionValue]        bit NOT NULL CONSTRAINT [DF_fin_AssetTransactionType_AffectsAcquisitionValue] DEFAULT (0),-- Updates the acquisition value
        [AffectsAccumulatedDepreciation] bit NOT NULL CONSTRAINT [DF_fin_AssetTransactionType_AffectsAccumulatedDepreciation] DEFAULT (0),-- Updates accumulated depreciation
        [IsPriorYearAcquisition]         bit NOT NULL CONSTRAINT [DF_fin_AssetTransactionType_IsPriorYearAcquisition] DEFAULT (0),-- Refers to a prior-year acquisition
        [RequiresRevenueAccount]         bit NOT NULL CONSTRAINT [DF_fin_AssetTransactionType_RequiresRevenueAccount] DEFAULT (0),-- Requires a revenue account (sale)
        [IsRetirementWithRevenue]        bit NOT NULL CONSTRAINT [DF_fin_AssetTransactionType_IsRetirementWithRevenue] DEFAULT (0),-- Retirement with revenue
        [DepreciationAreaRestriction]    nvarchar(255) NULL,                                                          -- Areas the type may post to
        [CreatedAt]                      datetime2(3) NOT NULL CONSTRAINT [DF_fin_AssetTransactionType_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                      nvarchar(64) NOT NULL,                                                       -- Creating user name
        [ModifiedAt]                     datetime2(3) NULL,                                                           -- Last change timestamp (UTC)
        [ModifiedBy]                     nvarchar(64) NULL,                                                           -- Last changing user name
        [RowVersion]                     rowversion NOT NULL,                                                         -- Optimistic concurrency token
        [IsActive]                       bit NOT NULL CONSTRAINT [DF_fin_AssetTransactionType_IsActive] DEFAULT (1),  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_AssetTransactionType] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_AssetTransactionType] UNIQUE ([TenantId], [TransactionType])
    );
END
GO

/* fin.AssetValue - Cumulative asset values per year and area (reference: ANLC) */
IF OBJECT_ID(N'fin.AssetValue', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[AssetValue]
    (
        [Id]                                    bigint IDENTITY(1,1) NOT NULL,                                        -- Surrogate key
        [TenantId]                              int NOT NULL,                                                         -- Owning tenant - every query is filtered by it
        [AssetId]                               bigint NOT NULL,                                                      -- Asset
        [DepreciationAreaId]                    bigint NOT NULL,                                                      -- Depreciation area
        [FiscalYear]                            smallint NOT NULL,                                                    -- Fiscal year
        [CurrencyCode]                          nvarchar(5) NOT NULL,                                                 -- Currency of the values
        [AcquisitionValueBroughtForward]        decimal(19,4) NOT NULL,                                               -- Opening acquisition value
        [AccumulatedDepreciationBroughtForward] decimal(19,4) NOT NULL,                                               -- Opening accumulated depreciation
        [CurrentYearAcquisitions]               decimal(19,4) NOT NULL,                                               -- Acquisitions in the year
        [CurrentYearRetirements]                decimal(19,4) NOT NULL,                                               -- Retirements in the year
        [CurrentYearTransfers]                  decimal(19,4) NOT NULL,                                               -- Transfers in the year
        [OrdinaryDepreciationPosted]            decimal(19,4) NOT NULL,                                               -- Ordinary depreciation posted
        [SpecialDepreciationPosted]             decimal(19,4) NOT NULL,                                               -- Special depreciation posted
        [UnplannedDepreciationPosted]           decimal(19,4) NOT NULL,                                               -- Unplanned depreciation posted
        [ImpairmentPosted]                      decimal(19,4) NOT NULL,                                               -- Impairment posted
        [WriteUpPosted]                         decimal(19,4) NOT NULL,                                               -- Write-ups posted
        [PlannedDepreciationYear]               decimal(19,4) NOT NULL,                                               -- Planned depreciation for the year
        [NetBookValue]                          decimal(19,4) NOT NULL,                                               -- Net book value at the end of the year
        [ScrapValue]                            decimal(19,4) NULL,                                                   -- Scrap value
        [ReplacementValue]                      decimal(19,4) NULL,                                                   -- Replacement value
        [LastUpdatedAt]                         datetime2(3) NOT NULL,                                                -- Last update (UTC)
        [RowVersion]                            rowversion NOT NULL,                                                  -- Concurrency token
        CONSTRAINT [PK_fin_AssetValue] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_AssetValue] UNIQUE ([TenantId], [AssetId], [DepreciationAreaId], [FiscalYear])
    );
END
GO

/* fin.BalanceCarryForward - Year-end carry-forward run and its results */
IF OBJECT_ID(N'fin.BalanceCarryForward', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[BalanceCarryForward]
    (
        [Id]                               bigint IDENTITY(1,1) NOT NULL,                                             -- Surrogate key
        [TenantId]                         int NOT NULL,                                                              -- Owning tenant - every query is filtered by it
        [CompanyCodeId]                    bigint NOT NULL,                                                           -- Company code
        [LedgerId]                         bigint NOT NULL,                                                           -- Ledger
        [FromFiscalYear]                   smallint NOT NULL,                                                         -- Year closed
        [ToFiscalYear]                     smallint NOT NULL,                                                         -- Year opened
        [RetainedEarningsGLAccountId]      bigint NOT NULL,                                                           -- Retained earnings account
        [CarriedForwardBalanceSheetAmount] decimal(19,4) NOT NULL,                                                    -- Balance sheet total carried forward
        [CarriedForwardProfitLossAmount]   decimal(19,4) NOT NULL,                                                    -- P&L result transferred
        [AccountsProcessed]                int NOT NULL,                                                              -- Number of accounts processed
        [Status]                           nvarchar(20) NOT NULL,                                                     -- Running, Completed, Failed, Repeated
        [ExecutedAt]                       datetime2(3) NOT NULL,                                                     -- Execution timestamp (UTC)
        [ExecutedBy]                       nvarchar(64) NOT NULL,                                                     -- Executing user
        [JournalEntryHeaderId]             bigint NULL,                                                               -- Carry-forward document
        [CreatedAt]                        datetime2(3) NOT NULL CONSTRAINT [DF_fin_BalanceCarryForward_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                        nvarchar(64) NOT NULL,                                                     -- Creating user name
        [ModifiedAt]                       datetime2(3) NULL,                                                         -- Last change timestamp (UTC)
        [ModifiedBy]                       nvarchar(64) NULL,                                                         -- Last changing user name
        [RowVersion]                       rowversion NOT NULL,                                                       -- Optimistic concurrency token
        [IsActive]                         bit NOT NULL CONSTRAINT [DF_fin_BalanceCarryForward_IsActive] DEFAULT (1), -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_BalanceCarryForward] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_BalanceCarryForward] UNIQUE ([TenantId], [CompanyCodeId], [LedgerId], [FromFiscalYear], [ToFiscalYear])
    );
END
GO

/* fin.ClearingDocument - Clearing transaction header (reference: clearing part of BSEG/BKPF) */
IF OBJECT_ID(N'fin.ClearingDocument', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[ClearingDocument]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [CompanyCodeId]          bigint NOT NULL,                                                                     -- Company code
        [FiscalYear]             smallint NOT NULL,                                                                   -- Fiscal year
        [ClearingDocumentNumber] nvarchar(20) NOT NULL,                                                               -- Clearing document number
        [ClearingDate]           date NOT NULL,                                                                       -- Clearing date
        [PostingDate]            date NOT NULL,                                                                       -- Posting date of the clearing document
        [ClearingType]           nvarchar(20) NOT NULL,                                                               -- IncomingPayment, OutgoingPayment, Transfer, Manual, Automatic, Reset
        [JournalEntryHeaderId]   bigint NULL,                                                                         -- Accounting document created by clearing
        [CurrencyCode]           nvarchar(5) NOT NULL,                                                                -- Clearing currency
        [TotalClearedAmount]     decimal(19,4) NOT NULL,                                                              -- Total amount cleared
        [DifferenceAmount]       decimal(19,4) NOT NULL,                                                              -- Residual / tolerance difference
        [DifferenceHandling]     nvarchar(20) NULL,                                                                   -- Residual, PartialPayment, WriteOff, OnAccount
        [IsReset]                bit NOT NULL CONSTRAINT [DF_fin_ClearingDocument_IsReset] DEFAULT (0),               -- Clearing has been reset
        [ResetAt]                datetime2(3) NULL,                                                                   -- Reset timestamp (UTC)
        [ResetBy]                nvarchar(64) NULL,                                                                   -- User who reset the clearing
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_fin_ClearingDocument_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user
        CONSTRAINT [PK_fin_ClearingDocument] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_ClearingDocument] UNIQUE ([TenantId], [CompanyCodeId], [FiscalYear], [ClearingDocumentNumber])
    );
END
GO

/* fin.ClearingItem - Open item cleared by a clearing document */
IF OBJECT_ID(N'fin.ClearingItem', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[ClearingItem]
    (
        [Id]                              bigint IDENTITY(1,1) NOT NULL,                                              -- Surrogate key
        [TenantId]                        int NOT NULL,                                                               -- Owning tenant - every query is filtered by it
        [ClearingDocumentId]              bigint NOT NULL,                                                            -- Clearing document
        [OpenItemId]                      bigint NOT NULL,                                                            -- Open item cleared
        [ClearedAmountInDocumentCurrency] decimal(19,4) NOT NULL,                                                     -- Amount cleared in document currency
        [ClearedAmountInLocalCurrency]    decimal(19,4) NOT NULL,                                                     -- Amount cleared in local currency
        [CashDiscountTaken]               decimal(19,4) NULL,                                                         -- Cash discount granted
        [ExchangeRateDifference]          decimal(19,4) NULL,                                                         -- Realised exchange rate gain/loss
        [IsPartialClearing]               bit NOT NULL CONSTRAINT [DF_fin_ClearingItem_IsPartialClearing] DEFAULT (0),-- Partial clearing
        [ResidualOpenItemId]              bigint NULL,                                                                -- Residual item created
        [CreatedAt]                       datetime2(3) NOT NULL CONSTRAINT [DF_fin_ClearingItem_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                       nvarchar(64) NOT NULL,                                                      -- Creating user
        CONSTRAINT [PK_fin_ClearingItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_ClearingItem] UNIQUE ([TenantId], [ClearingDocumentId], [OpenItemId])
    );
END
GO

/* fin.CustomerInvoice - Customer invoice / credit memo header (reference: FI invoice via VBRK/BKPF) */
IF OBJECT_ID(N'fin.CustomerInvoice', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[CustomerInvoice]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [CompanyCodeId]           bigint NOT NULL,                                                                    -- Company code
        [FiscalYear]              smallint NOT NULL,                                                                  -- Fiscal year
        [InvoiceNumber]           nvarchar(20) NOT NULL,                                                              -- Invoice number
        [InvoiceType]             nvarchar(20) NOT NULL,                                                              -- Invoice, CreditMemo, DownPaymentRequest, ProForma
        [BusinessPartnerId]       bigint NOT NULL,                                                                    -- Customer (BP with customer role)
        [PayerBusinessPartnerId]  bigint NULL,                                                                        -- Alternative payer
        [BillToBusinessPartnerId] bigint NULL,                                                                        -- Bill-to party
        [ShipToBusinessPartnerId] bigint NULL,                                                                        -- Ship-to party
        [InvoiceDate]             date NOT NULL,                                                                      -- Invoice date
        [PostingDate]             date NOT NULL,                                                                      -- Posting date
        [CurrencyCode]            nvarchar(5) NOT NULL,                                                               -- Invoice currency
        [ExchangeRate]            decimal(23,6) NULL,                                                                 -- Exchange rate to local currency
        [NetAmount]               decimal(19,4) NOT NULL,                                                             -- Net amount
        [TaxAmount]               decimal(19,4) NOT NULL,                                                             -- Tax amount
        [GrossAmount]             decimal(19,4) NOT NULL,                                                             -- Gross amount
        [WithholdingTaxAmount]    decimal(19,4) NULL,                                                                 -- Withholding tax
        [PaymentTermsId]          bigint NULL,                                                                        -- Payment terms
        [BaselineDate]            date NULL,                                                                          -- Baseline date
        [DueDate]                 date NULL,                                                                          -- Due date
        [PaymentMethod]           nvarchar(1) NULL,                                                                   -- Payment method
        [SalesAreaId]             bigint NULL,                                                                        -- Sales area
        [ReferenceDocumentNumber] nvarchar(20) NULL,                                                                  -- Customer reference / PO number
        [HeaderText]              nvarchar(255) NULL,                                                                 -- Header text
        [Status]                  nvarchar(20) NOT NULL,                                                              -- Draft, Parked, PendingApproval, Posted, PartiallyCleared, Cleared, Reversed, Cancelled
        [JournalEntryHeaderId]    bigint NULL,                                                                        -- Accounting document created
        [PaidAmount]              decimal(19,4) NOT NULL,                                                             -- Amount received so far
        [OpenAmount]              decimal(19,4) NOT NULL,                                                             -- Amount still open
        [IsDunningBlocked]        bit NOT NULL CONSTRAINT [DF_fin_CustomerInvoice_IsDunningBlocked] DEFAULT (0),      -- Dunning block
        [WorkflowInstanceId]      bigint NULL,                                                                        -- Approval workflow
        [IdempotencyKey]          uniqueidentifier NULL,                                                              -- Duplicate protection
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_fin_CustomerInvoice_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_fin_CustomerInvoice_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_CustomerInvoice] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_CustomerInvoice] UNIQUE ([TenantId], [CompanyCodeId], [FiscalYear], [InvoiceNumber])
    );
END
GO

/* fin.CustomerInvoiceItem - Customer invoice line */
IF OBJECT_ID(N'fin.CustomerInvoiceItem', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[CustomerInvoiceItem]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [CustomerInvoiceId]  bigint NOT NULL,                                                                         -- Owning invoice
        [ItemNumber]         int NOT NULL,                                                                            -- Item number
        [RevenueGLAccountId] bigint NOT NULL,                                                                         -- Revenue account
        [Description]        nvarchar(255) NOT NULL,                                                                  -- Item description
        [Quantity]           decimal(23,6) NULL,                                                                      -- Quantity
        [UnitOfMeasure]      nvarchar(3) NULL,                                                                        -- Unit of measure
        [UnitPrice]          decimal(19,4) NULL,                                                                      -- Unit price
        [NetAmount]          decimal(19,4) NOT NULL,                                                                  -- Net amount
        [DiscountAmount]     decimal(19,4) NULL,                                                                      -- Item discount
        [TaxCodeId]          bigint NULL,                                                                             -- Tax code
        [TaxAmount]          decimal(19,4) NULL,                                                                      -- Tax amount
        [CostCenterId]       bigint NULL,                                                                             -- Cost centre
        [ProfitCenterId]     bigint NULL,                                                                             -- Profit centre
        [InternalOrderId]    bigint NULL,                                                                             -- Internal order
        [SegmentId]          bigint NULL,                                                                             -- Segment
        [FunctionalAreaId]   bigint NULL,                                                                             -- Functional area
        [BusinessAreaId]     bigint NULL,                                                                             -- Business area
        [MaterialNumber]     nvarchar(40) NULL,                                                                       -- Material (future module)
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_fin_CustomerInvoiceItem_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_fin_CustomerInvoiceItem_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_CustomerInvoiceItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_CustomerInvoiceItem] UNIQUE ([TenantId], [CustomerInvoiceId], [ItemNumber])
    );
END
GO

/* fin.DepreciationArea - Depreciation area (book, tax, group, cost accounting) (reference: T093) */
IF OBJECT_ID(N'fin.DepreciationArea', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[DepreciationArea]
    (
        [Id]                         bigint IDENTITY(1,1) NOT NULL,                                                   -- Surrogate key
        [TenantId]                   int NOT NULL,                                                                    -- Owning tenant - every query is filtered by it
        [CompanyCodeId]              bigint NOT NULL,                                                                 -- Company code
        [DepreciationArea]           nvarchar(2) NOT NULL,                                                            -- Area key, e.g. 01 book, 15 tax, 30 group
        [Name]                       nvarchar(60) NOT NULL,                                                           -- Description
        [AreaType]                   nvarchar(20) NOT NULL,                                                           -- Book, Tax, Group, CostAccounting, Derived
        [LedgerId]                   bigint NULL,                                                                     -- Ledger the area posts to
        [AccountingPrincipleId]      bigint NULL,                                                                     -- Accounting principle
        [CurrencyCode]               nvarchar(5) NOT NULL,                                                            -- Area currency
        [PostsToGeneralLedger]       nvarchar(20) NOT NULL,                                                           -- RealTime, Periodic, NoPosting, DepreciationOnly
        [IsRealDepreciationArea]     bit NOT NULL CONSTRAINT [DF_fin_DepreciationArea_IsRealDepreciationArea] DEFAULT (0),-- Real (not derived) area
        [DerivedFromArea1]           nvarchar(2) NULL,                                                                -- First area of a derived area
        [DerivedFromArea2]           nvarchar(2) NULL,                                                                -- Second area of a derived area
        [AcquisitionValueRule]       nvarchar(20) NOT NULL,                                                           -- AllValuesAllowed, PositiveOnly, NegativeOnly, ZeroOnly
        [NetBookValueRule]           nvarchar(20) NOT NULL,                                                           -- Allowed net book value sign
        [IsAreaForParallelValuation] bit NOT NULL CONSTRAINT [DF_fin_DepreciationArea_IsAreaForParallelValuation] DEFAULT (0),-- Parallel valuation area
        [IsActive]                   bit NOT NULL CONSTRAINT [DF_fin_DepreciationArea_IsActive] DEFAULT (1),          -- Active
        [CreatedAt]                  datetime2(3) NOT NULL CONSTRAINT [DF_fin_DepreciationArea_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                  nvarchar(64) NOT NULL,                                                           -- Creating user name
        [ModifiedAt]                 datetime2(3) NULL,                                                               -- Last change timestamp (UTC)
        [ModifiedBy]                 nvarchar(64) NULL,                                                               -- Last changing user name
        [RowVersion]                 rowversion NOT NULL,                                                             -- Optimistic concurrency token
        CONSTRAINT [PK_fin_DepreciationArea] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_DepreciationArea] UNIQUE ([TenantId], [CompanyCodeId], [DepreciationArea])
    );
END
GO

/* fin.DepreciationKey - Depreciation key - method, base and period control (reference: T090NA) */
IF OBJECT_ID(N'fin.DepreciationKey', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[DepreciationKey]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [DepreciationKey]           nvarchar(4) NOT NULL,                                                             -- Key, e.g. LINR, DG20
        [Name]                      nvarchar(60) NOT NULL,                                                            -- Description
        [DepreciationMethod]        nvarchar(20) NOT NULL,                                                            -- StraightLine, DecliningBalance, SumOfYearsDigits, UnitOfProduction, Manual, Immediate
        [BaseValueRule]             nvarchar(20) NOT NULL,                                                            -- AcquisitionValue, NetBookValue, ReplacementValue, HalfAcquisitionValue
        [DeclineFactor]             decimal(9,4) NULL,                                                                -- Multiplier for declining balance
        [PeriodControlAcquisition]  nvarchar(3) NOT NULL,                                                             -- Period control for acquisitions, e.g. 01 pro rata
        [PeriodControlAddition]     nvarchar(3) NOT NULL,                                                             -- Period control for subsequent acquisitions
        [PeriodControlRetirement]   nvarchar(3) NOT NULL,                                                             -- Period control for retirements
        [PeriodControlTransfer]     nvarchar(3) NOT NULL,                                                             -- Period control for transfers
        [ChangeMethodAtEnd]         nvarchar(20) NULL,                                                                -- Change to straight line when it is more favourable
        [IsScrapValueConsidered]    bit NOT NULL CONSTRAINT [DF_fin_DepreciationKey_IsScrapValueConsidered] DEFAULT (0),-- Depreciation stops at the scrap value
        [AllowNegativeDepreciation] bit NOT NULL CONSTRAINT [DF_fin_DepreciationKey_AllowNegativeDepreciation] DEFAULT (0),-- Write-ups permitted
        [IsShutdownRelevant]        bit NOT NULL CONSTRAINT [DF_fin_DepreciationKey_IsShutdownRelevant] DEFAULT (0),  -- Shutdown periods suspend depreciation
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_fin_DepreciationKey_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_fin_DepreciationKey_IsActive] DEFAULT (1),            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_DepreciationKey] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_DepreciationKey] UNIQUE ([TenantId], [DepreciationKey])
    );
END
GO

/* fin.DepreciationPosting - Depreciation posted per asset, area and period (reference: ANLP) */
IF OBJECT_ID(N'fin.DepreciationPosting', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[DepreciationPosting]
    (
        [Id]                          bigint IDENTITY(1,1) NOT NULL,                                                  -- Surrogate key
        [TenantId]                    int NOT NULL,                                                                   -- Owning tenant - every query is filtered by it
        [DepreciationRunId]           bigint NOT NULL,                                                                -- Run that produced the posting
        [AssetId]                     bigint NOT NULL,                                                                -- Asset
        [DepreciationAreaId]          bigint NOT NULL,                                                                -- Depreciation area
        [FiscalYear]                  smallint NOT NULL,                                                              -- Fiscal year
        [FiscalPeriod]                tinyint NOT NULL,                                                               -- Period
        [PostingDate]                 date NOT NULL,                                                                  -- Posting date
        [CurrencyCode]                nvarchar(5) NOT NULL,                                                           -- Currency
        [OrdinaryDepreciationAmount]  decimal(19,4) NOT NULL,                                                         -- Ordinary depreciation
        [SpecialDepreciationAmount]   decimal(19,4) NOT NULL,                                                         -- Special depreciation
        [UnplannedDepreciationAmount] decimal(19,4) NOT NULL,                                                         -- Unplanned depreciation
        [ImpairmentAmount]            decimal(19,4) NOT NULL,                                                         -- Impairment
        [WriteUpAmount]               decimal(19,4) NOT NULL,                                                         -- Write-up
        [TotalPostedAmount]           decimal(19,4) NOT NULL,                                                         -- Total posted in the period
        [NetBookValueAfterPosting]    decimal(19,4) NOT NULL,                                                         -- Net book value after the posting
        [CostCenterId]                bigint NULL,                                                                    -- Cost centre charged
        [ProfitCenterId]              bigint NULL,                                                                    -- Profit centre charged
        [InternalOrderId]             bigint NULL,                                                                    -- Internal order charged
        [JournalEntryHeaderId]        bigint NULL,                                                                    -- Accounting document created
        [IsReversed]                  bit NOT NULL CONSTRAINT [DF_fin_DepreciationPosting_IsReversed] DEFAULT (0),    -- Reversed by a repeat run
        [CreatedAt]                   datetime2(3) NOT NULL CONSTRAINT [DF_fin_DepreciationPosting_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                   nvarchar(64) NOT NULL,                                                          -- Creating user or job
        CONSTRAINT [PK_fin_DepreciationPosting] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_DepreciationPosting] UNIQUE ([TenantId], [AssetId], [DepreciationAreaId], [FiscalYear], [FiscalPeriod])
    );
END
GO

/* fin.DepreciationRun - Depreciation posting run (AFAB) */
IF OBJECT_ID(N'fin.DepreciationRun', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[DepreciationRun]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [CompanyCodeId]           bigint NOT NULL,                                                                    -- Company code
        [FiscalYear]              smallint NOT NULL,                                                                  -- Fiscal year
        [FiscalPeriod]            tinyint NOT NULL,                                                                   -- Period posted
        [RunNumber]               int NOT NULL,                                                                       -- 1 for the planned run, then 2, 3 ? for each repeat. Without it the key would allow one run per period and forbid the repeats this table defines
        [RunType]                 nvarchar(20) NOT NULL,                                                              -- Planned, Repeat, Restart, Unplanned
        [IsTestRun]               bit NOT NULL CONSTRAINT [DF_fin_DepreciationRun_IsTestRun] DEFAULT (0),             -- Test run - no postings created
        [Status]                  nvarchar(20) NOT NULL,                                                              -- Scheduled, Running, Completed, Failed, Cancelled
        [AssetsProcessed]         int NULL,                                                                           -- Number of assets processed
        [TotalDepreciationAmount] decimal(19,4) NULL,                                                                 -- Total depreciation posted
        [ErrorCount]              int NULL,                                                                           -- Number of errors
        [StartedAt]               datetime2(3) NOT NULL,                                                              -- Start timestamp (UTC)
        [CompletedAt]             datetime2(3) NULL,                                                                  -- Completion timestamp (UTC)
        [ExecutedBy]              nvarchar(64) NOT NULL,                                                              -- Executing user or job
        [LogText]                 nvarchar(max) NULL,                                                                 -- Run log
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_fin_DepreciationRun_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_fin_DepreciationRun_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_DepreciationRun] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_DepreciationRun] UNIQUE ([TenantId], [CompanyCodeId], [FiscalYear], [FiscalPeriod], [RunNumber])
    );
END
GO

/* fin.DocumentAttachment - File attached to an accounting document, invoice or payment */
IF OBJECT_ID(N'fin.DocumentAttachment', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[DocumentAttachment]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [ObjectType]       nvarchar(40) NOT NULL,                                                                     -- JournalEntry, CustomerInvoice, VendorInvoice, Payment, Asset
        [ObjectId]         bigint NOT NULL,                                                                           -- Key of the object
        [FileName]         nvarchar(255) NOT NULL,                                                                    -- File name
        [ContentType]      nvarchar(100) NOT NULL,                                                                    -- MIME type
        [FileSizeBytes]    bigint NOT NULL,                                                                           -- Size in bytes
        [StorageUri]       nvarchar(500) NOT NULL,                                                                    -- Location in the document store
        [Checksum]         nvarchar(64) NOT NULL,                                                                     -- SHA-256 of the content
        [DocumentCategory] nvarchar(40) NULL,                                                                         -- SourceDocument, Approval, BankAdvice, Other
        [UploadedAt]       datetime2(3) NOT NULL,                                                                     -- Upload timestamp (UTC)
        [UploadedBy]       nvarchar(64) NOT NULL,                                                                     -- Uploading user
        [IsImmutable]      bit NOT NULL CONSTRAINT [DF_fin_DocumentAttachment_IsImmutable] DEFAULT (0),               -- Attached to a posted document - cannot be removed
        CONSTRAINT [PK_fin_DocumentAttachment] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_DocumentAttachment] UNIQUE ([TenantId], [ObjectType], [ObjectId])
    );
END
GO

/* fin.DunningNotice - Dunning notice per partner and level (reference: MHND) */
IF OBJECT_ID(N'fin.DunningNotice', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[DunningNotice]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [DunningRunId]        bigint NOT NULL,                                                                        -- Dunning run
        [BusinessPartnerId]   bigint NOT NULL,                                                                        -- Customer dunned
        [DunningLevel]        tinyint NOT NULL,                                                                       -- Dunning level reached
        [DunningProcedureId]  bigint NOT NULL,                                                                        -- Dunning procedure
        [TotalOverdueAmount]  decimal(19,4) NOT NULL,                                                                 -- Total overdue
        [CurrencyCode]        nvarchar(5) NOT NULL,                                                                   -- Currency
        [InterestAmount]      decimal(19,4) NULL,                                                                     -- Interest charged
        [DunningChargeAmount] decimal(19,4) NULL,                                                                     -- Dunning charge
        [ItemCount]           int NOT NULL,                                                                           -- Number of dunned items
        [IsPrinted]           bit NOT NULL CONSTRAINT [DF_fin_DunningNotice_IsPrinted] DEFAULT (0),                   -- Notice issued
        [PrintedAt]           datetime2(3) NULL,                                                                      -- Print timestamp (UTC)
        [DocumentUri]         nvarchar(500) NULL,                                                                     -- Generated PDF
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_fin_DunningNotice_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_fin_DunningNotice_IsActive] DEFAULT (1),                    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_DunningNotice] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_DunningNotice] UNIQUE ([TenantId], [DunningRunId], [BusinessPartnerId])
    );
END
GO

/* fin.DunningNoticeItem - Open item on a dunning notice */
IF OBJECT_ID(N'fin.DunningNoticeItem', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[DunningNoticeItem]
    (
        [Id]              bigint IDENTITY(1,1) NOT NULL,                                                              -- Surrogate key
        [TenantId]        int NOT NULL,                                                                               -- Owning tenant - every query is filtered by it
        [DunningNoticeId] bigint NOT NULL,                                                                            -- Dunning notice
        [OpenItemId]      bigint NOT NULL,                                                                            -- Dunned open item
        [DunningLevel]    tinyint NOT NULL,                                                                           -- Level applied to the item
        [OverdueAmount]   decimal(19,4) NOT NULL,                                                                     -- Overdue amount
        [DaysInArrears]   int NOT NULL,                                                                               -- Days overdue
        [InterestAmount]  decimal(19,4) NULL,                                                                         -- Interest on this item
        [CreatedAt]       datetime2(3) NOT NULL CONSTRAINT [DF_fin_DunningNoticeItem_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]       nvarchar(64) NOT NULL,                                                                      -- Creating user name
        [ModifiedAt]      datetime2(3) NULL,                                                                          -- Last change timestamp (UTC)
        [ModifiedBy]      nvarchar(64) NULL,                                                                          -- Last changing user name
        [RowVersion]      rowversion NOT NULL,                                                                        -- Optimistic concurrency token
        [IsActive]        bit NOT NULL CONSTRAINT [DF_fin_DunningNoticeItem_IsActive] DEFAULT (1),                    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_DunningNoticeItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_DunningNoticeItem] UNIQUE ([TenantId], [DunningNoticeId], [OpenItemId])
    );
END
GO

/* fin.DunningRun - Dunning run (reference: MHNK run level) */
IF OBJECT_ID(N'fin.DunningRun', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[DunningRun]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [RunDate]              date NOT NULL,                                                                         -- Run date
        [RunIdentifier]        nvarchar(6) NOT NULL,                                                                  -- Identification of the run
        [CompanyCodeId]        bigint NOT NULL,                                                                       -- Company code
        [DunningDate]          date NOT NULL,                                                                         -- Date printed on the letters
        [DocumentsPostedUntil] date NOT NULL,                                                                         -- Documents considered up to this date
        [Status]               nvarchar(20) NOT NULL,                                                                 -- Scheduled, ProposalReady, Edited, Printed, Cancelled
        [NoticeCount]          int NULL,                                                                              -- Number of notices produced
        [TotalDunnedAmount]    decimal(19,4) NULL,                                                                    -- Total amount dunned
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_fin_DunningRun_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_fin_DunningRun_IsActive] DEFAULT (1),                      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_DunningRun] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_DunningRun] UNIQUE ([TenantId], [RunDate], [RunIdentifier])
    );
END
GO

/* fin.ForeignCurrencyValuation - Result of a period-end foreign currency valuation */
IF OBJECT_ID(N'fin.ForeignCurrencyValuation', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[ForeignCurrencyValuation]
    (
        [Id]                            bigint IDENTITY(1,1) NOT NULL,                                                -- Surrogate key
        [TenantId]                      int NOT NULL,                                                                 -- Owning tenant - every query is filtered by it
        [CompanyCodeId]                 bigint NOT NULL,                                                              -- Company code
        [FiscalYear]                    smallint NOT NULL,                                                            -- Fiscal year
        [FiscalPeriod]                  tinyint NOT NULL,                                                             -- Period valued
        [ValuationAreaCode]             nvarchar(4) NOT NULL,                                                         -- Valuation area / accounting principle
        [OpenItemId]                    bigint NULL,                                                                  -- Valued open item (null for account balances)
        [GLAccountId]                   bigint NOT NULL,                                                              -- Account valued
        [ValuationDate]                 date NOT NULL,                                                                -- Key date of the valuation
        [ExchangeRateUsed]              decimal(23,6) NOT NULL,                                                       -- Rate used
        [AmountInDocumentCurrency]      decimal(19,4) NOT NULL,                                                       -- Original foreign currency amount
        [BookValueInLocalCurrency]      decimal(19,4) NOT NULL,                                                       -- Book value before valuation
        [ValuatedAmountInLocalCurrency] decimal(19,4) NOT NULL,                                                       -- Value at the valuation rate
        [UnrealizedGainLossAmount]      decimal(19,4) NOT NULL,                                                       -- Unrealised difference
        [IsReversalPosted]              bit NOT NULL CONSTRAINT [DF_fin_ForeignCurrencyValuation_IsReversalPosted] DEFAULT (0),-- Valuation reversed in the following period
        [JournalEntryHeaderId]          bigint NULL,                                                                  -- Valuation posting
        [ReversalJournalEntryHeaderId]  bigint NULL,                                                                  -- Reversal posting
        [CreatedAt]                     datetime2(3) NOT NULL CONSTRAINT [DF_fin_ForeignCurrencyValuation_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                     nvarchar(64) NOT NULL,                                                        -- Creating user name
        [ModifiedAt]                    datetime2(3) NULL,                                                            -- Last change timestamp (UTC)
        [ModifiedBy]                    nvarchar(64) NULL,                                                            -- Last changing user name
        [RowVersion]                    rowversion NOT NULL,                                                          -- Optimistic concurrency token
        [IsActive]                      bit NOT NULL CONSTRAINT [DF_fin_ForeignCurrencyValuation_IsActive] DEFAULT (1),-- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_ForeignCurrencyValuation] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_ForeignCurrencyValuation] UNIQUE ([TenantId], [CompanyCodeId], [FiscalYear], [FiscalPeriod], [ValuationAreaCode], [OpenItemId])
    );
END
GO

/* fin.JournalEntryHeader - Accounting document header (reference: BKPF) */
IF OBJECT_ID(N'fin.JournalEntryHeader', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[JournalEntryHeader]
    (
        [Id]                         bigint IDENTITY(1,1) NOT NULL,                                                   -- Surrogate key
        [TenantId]                   int NOT NULL,                                                                    -- Owning tenant - every query is filtered by it
        [CompanyCodeId]              bigint NOT NULL,                                                                 -- Company code
        [FiscalYear]                 smallint NOT NULL,                                                               -- Fiscal year
        [DocumentNumber]             nvarchar(20) NOT NULL,                                                           -- Document number, e.g. KSS-2026-SA-0000000123
        [LedgerId]                   bigint NOT NULL,                                                                 -- Ledger - the leading ledger for standard postings
        [DocumentTypeId]             bigint NOT NULL,                                                                 -- Document type
        [DocumentDate]               date NOT NULL,                                                                   -- Document date
        [PostingDate]                date NOT NULL,                                                                   -- Posting date - determines period
        [EntryDate]                  date NOT NULL,                                                                   -- Date the document was entered
        [EntryTime]                  time(0) NOT NULL,                                                                -- Time of entry
        [FiscalPeriod]               tinyint NOT NULL,                                                                -- Posting period derived from the posting date
        [TranslationDate]            date NULL,                                                                       -- Date used for currency translation
        [DocumentCurrencyCode]       nvarchar(5) NOT NULL,                                                            -- Document (transaction) currency
        [LocalCurrencyCode]          nvarchar(5) NOT NULL,                                                            -- Company code currency
        [GroupCurrencyCode]          nvarchar(5) NULL,                                                                -- Group currency
        [ExchangeRate]               decimal(23,6) NULL,                                                              -- Document -> local exchange rate
        [ExchangeRateType]           nvarchar(4) NULL,                                                                -- Rate type used
        [GroupExchangeRate]          decimal(23,6) NULL,                                                              -- Document -> group exchange rate
        [TotalDebitAmount]           decimal(19,4) NOT NULL,                                                          -- Sum of debits in document currency
        [TotalCreditAmount]          decimal(19,4) NOT NULL,                                                          -- Sum of credits in document currency
        [ReferenceDocumentNumber]    nvarchar(20) NULL,                                                               -- External reference (invoice number)
        [DocumentHeaderText]         nvarchar(255) NULL,                                                              -- Header text
        [Status]                     nvarchar(20) NOT NULL,                                                           -- Draft, Held, Parked, Submitted, PendingApproval, Approved, Posted, Rejected, Reversed, Cancelled
        [PostedAt]                   datetime2(3) NULL,                                                               -- Posting timestamp (UTC)
        [PostedBy]                   nvarchar(64) NULL,                                                               -- Posting user
        [ParkedBy]                   nvarchar(64) NULL,                                                               -- User who parked the document
        [ApprovedBy]                 nvarchar(64) NULL,                                                               -- Approving user - must differ from PostedBy under maker-checker
        [ApprovedAt]                 datetime2(3) NULL,                                                               -- Approval timestamp (UTC)
        [WorkflowInstanceId]         bigint NULL,                                                                     -- Approval workflow instance
        [IsReversed]                 bit NOT NULL CONSTRAINT [DF_fin_JournalEntryHeader_IsReversed] DEFAULT (0),      -- Document has been reversed
        [ReversalDocumentNumber]     nvarchar(20) NULL,                                                               -- Reversing document
        [ReversalReasonCode]         nvarchar(2) NULL,                                                                -- Reversal reason
        [ReversedDocumentNumber]     nvarchar(20) NULL,                                                               -- Original document, if this is a reversal
        [ReversalDate]               date NULL,                                                                       -- Posting date of the reversal
        [IsIntercompany]             bit NOT NULL CONSTRAINT [DF_fin_JournalEntryHeader_IsIntercompany] DEFAULT (0),  -- Part of a cross-company-code transaction
        [IntercompanyDocumentNumber] nvarchar(20) NULL,                                                               -- Cross-company-code document number
        [SourceModule]               nvarchar(10) NOT NULL,                                                           -- FI, AR, AP, AA, CO, MM, SD, PY
        [SourceDocumentType]         nvarchar(20) NULL,                                                               -- Originating object type
        [SourceDocumentId]           bigint NULL,                                                                     -- Originating object key
        [TransactionCode]            nvarchar(20) NULL,                                                               -- T-code used, e.g. FB50
        [IdempotencyKey]             uniqueidentifier NULL,                                                           -- Prevents duplicate posting on retry
        [CorrelationId]              uniqueidentifier NULL,                                                           -- Request correlation id
        [BatchInputSession]          nvarchar(20) NULL,                                                               -- Import / batch session name
        [IsSimulation]               bit NOT NULL CONSTRAINT [DF_fin_JournalEntryHeader_IsSimulation] DEFAULT (0),    -- Simulated document, never balanced into totals
        [CreatedAt]                  datetime2(3) NOT NULL CONSTRAINT [DF_fin_JournalEntryHeader_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                  nvarchar(64) NOT NULL,                                                           -- Creating user
        [ModifiedAt]                 datetime2(3) NULL,                                                               -- Last change before posting only
        [ModifiedBy]                 nvarchar(64) NULL,                                                               -- Last changing user before posting
        CONSTRAINT [PK_fin_JournalEntryHeader] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_JournalEntryHeader] UNIQUE ([TenantId], [CompanyCodeId], [FiscalYear], [DocumentNumber], [LedgerId], [IdempotencyKey])
    );
END
GO

/* fin.JournalEntryLine - Universal journal line item - the single source of accounting truth (reference: ACDOCA / BSEG) */
IF OBJECT_ID(N'fin.JournalEntryLine', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[JournalEntryLine]
    (
        [Id]                              bigint IDENTITY(1,1) NOT NULL,                                              -- Surrogate key
        [TenantId]                        int NOT NULL,                                                               -- Owning tenant - every query is filtered by it
        [JournalEntryHeaderId]            bigint NOT NULL,                                                            -- Owning document header
        [CompanyCodeId]                   bigint NOT NULL,                                                            -- Company code
        [FiscalYear]                      smallint NOT NULL,                                                          -- Fiscal year
        [DocumentNumber]                  nvarchar(20) NOT NULL,                                                      -- Document number
        [LineItemNumber]                  int NOT NULL,                                                               -- Line number within the document
        [LedgerId]                        bigint NOT NULL,                                                            -- Ledger
        [PostingDate]                     date NOT NULL,                                                              -- Posting date (denormalised for reporting)
        [FiscalPeriod]                    tinyint NOT NULL,                                                           -- Posting period
        [PostingKey]                      nvarchar(2) NOT NULL,                                                       -- Posting key
        [DebitCreditIndicator]            nvarchar(1) NOT NULL,                                                       -- S debit, H credit
        [AccountType]                     nvarchar(1) NOT NULL,                                                       -- S, D, K, A, M
        [GLAccountId]                     bigint NOT NULL,                                                            -- G/L account - reconciliation account for D/K lines
        [GLAccount]                       nvarchar(10) NOT NULL,                                                      -- G/L account number (denormalised)
        [BusinessPartnerId]               bigint NULL,                                                                -- Customer / vendor partner
        [BusinessPartnerRoleCategory]     nvarchar(20) NULL,                                                          -- Customer or Vendor - which role posted
        [SpecialGLIndicator]              nvarchar(1) NULL,                                                           -- Special G/L indicator
        [AssetId]                         bigint NULL,                                                                -- Asset master record
        [AssetSubNumber]                  int NULL,                                                                   -- Asset sub-number
        [AssetTransactionType]            nvarchar(3) NULL,                                                           -- Asset transaction type
        [CostCenterId]                    bigint NULL,                                                                -- Cost centre
        [ProfitCenterId]                  bigint NULL,                                                                -- Profit centre
        [PartnerProfitCenterId]           bigint NULL,                                                                -- Partner profit centre
        [InternalOrderId]                 bigint NULL,                                                                -- Internal order
        [CostElementId]                   bigint NULL,                                                                -- Cost element
        [ActivityTypeId]                  bigint NULL,                                                                -- Activity type
        [BusinessAreaId]                  bigint NULL,                                                                -- Business area
        [FunctionalAreaId]                bigint NULL,                                                                -- Functional area
        [SegmentId]                       bigint NULL,                                                                -- Segment
        [PartnerSegmentId]                bigint NULL,                                                                -- Partner segment
        [PartnerCompanyCodeId]            bigint NULL,                                                                -- Partner company code (intercompany)
        [TradingPartnerCompany]           nvarchar(6) NULL,                                                           -- Trading partner for consolidation
        [PlantId]                         bigint NULL,                                                                -- Plant
        [BranchId]                        bigint NULL,                                                                -- Branch
        [ProjectId]                       bigint NULL,                                                                -- Project / WBS element (future module)
        [DocumentCurrencyCode]            nvarchar(5) NOT NULL,                                                       -- Document currency
        [AmountInDocumentCurrency]        decimal(19,4) NOT NULL,                                                     -- Amount in document currency (signed)
        [LocalCurrencyCode]               nvarchar(5) NOT NULL,                                                       -- Company code currency
        [AmountInLocalCurrency]           decimal(19,4) NOT NULL,                                                     -- Amount in local currency (signed)
        [GroupCurrencyCode]               nvarchar(5) NULL,                                                           -- Group currency
        [AmountInGroupCurrency]           decimal(19,4) NULL,                                                         -- Amount in group currency (signed)
        [ControllingAreaCurrencyCode]     nvarchar(5) NULL,                                                           -- Controlling area currency
        [AmountInControllingAreaCurrency] decimal(19,4) NULL,                                                         -- Amount in controlling area currency
        [HardCurrencyCode]                nvarchar(5) NULL,                                                           -- Hard currency
        [AmountInHardCurrency]            decimal(19,4) NULL,                                                         -- Amount in hard currency
        [ExchangeRate]                    decimal(23,6) NULL,                                                         -- Rate document -> local
        [Quantity]                        decimal(23,6) NULL,                                                         -- Quantity
        [UnitOfMeasure]                   nvarchar(3) NULL,                                                           -- Unit of measure
        [TaxCodeId]                       bigint NULL,                                                                -- Tax code
        [TaxBaseAmountInDocumentCurrency] decimal(19,4) NULL,                                                         -- Tax base amount
        [TaxAmountInDocumentCurrency]     decimal(19,4) NULL,                                                         -- Tax amount in document currency
        [TaxAmountInLocalCurrency]        decimal(19,4) NULL,                                                         -- Tax amount in local currency
        [TaxJurisdictionCode]             nvarchar(15) NULL,                                                          -- Tax jurisdiction
        [IsTaxLine]                       bit NOT NULL CONSTRAINT [DF_fin_JournalEntryLine_IsTaxLine] DEFAULT (0),    -- Automatically generated tax line
        [WithholdingTaxCodeId]            bigint NULL,                                                                -- Withholding tax code
        [WithholdingTaxBaseAmount]        decimal(19,4) NULL,                                                         -- Withholding tax base
        [WithholdingTaxAmount]            decimal(19,4) NULL,                                                         -- Withholding tax amount
        [AssignmentReference]             nvarchar(18) NULL,                                                          -- Allocation field (sort key result)
        [LineItemText]                    nvarchar(255) NULL,                                                         -- Item text
        [ReferenceKey1]                   nvarchar(20) NULL,                                                          -- Reference key 1
        [ReferenceKey2]                   nvarchar(20) NULL,                                                          -- Reference key 2
        [ReferenceKey3]                   nvarchar(20) NULL,                                                          -- Reference key 3
        [BaselineDate]                    date NULL,                                                                  -- Baseline date for payment terms
        [PaymentTermsId]                  bigint NULL,                                                                -- Payment terms
        [DueDate]                         date NULL,                                                                  -- Net due date
        [CashDiscount1Percent]            decimal(9,4) NULL,                                                          -- Cash discount percentage 1
        [CashDiscount1Days]               int NULL,                                                                   -- Cash discount days 1
        [CashDiscountBaseAmount]          decimal(19,4) NULL,                                                         -- Amount eligible for discount
        [PaymentMethod]                   nvarchar(1) NULL,                                                           -- Payment method
        [PaymentBlockReason]              nvarchar(1) NULL,                                                           -- Payment block
        [HouseBankId]                     bigint NULL,                                                                -- House bank for payment
        [PartnerBankDetailId]             nvarchar(4) NULL,                                                           -- Partner bank details id
        [IsOpenItemManaged]               bit NOT NULL CONSTRAINT [DF_fin_JournalEntryLine_IsOpenItemManaged] DEFAULT (0),-- Line creates an open item
        [ClearingStatus]                  nvarchar(20) NOT NULL,                                                      -- Open, PartiallyCleared, Cleared
        [ClearingDocumentNumber]          nvarchar(20) NULL,                                                          -- Clearing document
        [ClearingDate]                    date NULL,                                                                  -- Clearing date
        [InvoiceReferenceDocumentNumber]  nvarchar(20) NULL,                                                          -- Invoice a credit memo or payment refers to
        [InvoiceReferenceFiscalYear]      smallint NULL,                                                              -- Fiscal year of the referenced invoice
        [InvoiceReferenceLineItem]        int NULL,                                                                   -- Line of the referenced invoice
        [IsReversalLine]                  bit NOT NULL CONSTRAINT [DF_fin_JournalEntryLine_IsReversalLine] DEFAULT (0),-- Line belongs to a reversal document
        [IsNegativePosting]               bit NOT NULL CONSTRAINT [DF_fin_JournalEntryLine_IsNegativePosting] DEFAULT (0),-- Negative posting (reduces the same side)
        [IsStatistical]                   bit NOT NULL CONSTRAINT [DF_fin_JournalEntryLine_IsStatistical] DEFAULT (0),-- Statistical posting (noted item, statistical order)
        [SourceModule]                    nvarchar(10) NOT NULL,                                                      -- Originating module
        [ControllingDocumentNumber]       nvarchar(30) NULL,                                                          -- Related CO document (FI number plus a -CO### suffix)
        [AssetDocumentNumber]             nvarchar(20) NULL,                                                          -- Related asset document
        [MaterialNumber]                  nvarchar(40) NULL,                                                          -- Material (future inventory module)
        [CreatedAt]                       datetime2(3) NOT NULL CONSTRAINT [DF_fin_JournalEntryLine_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                       nvarchar(64) NOT NULL,                                                      -- Creating user
        CONSTRAINT [PK_fin_JournalEntryLine] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_JournalEntryLine] UNIQUE ([TenantId], [CompanyCodeId], [FiscalYear], [DocumentNumber], [LineItemNumber], [LedgerId])
    );
END
GO

/* fin.JournalEntryTax - Tax breakdown per document, code and jurisdiction (reference: BSET) */
IF OBJECT_ID(N'fin.JournalEntryTax', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[JournalEntryTax]
    (
        [Id]                              bigint IDENTITY(1,1) NOT NULL,                                              -- Surrogate key
        [TenantId]                        int NOT NULL,                                                               -- Owning tenant - every query is filtered by it
        [CompanyCodeId]                   bigint NOT NULL,                                                            -- Company code
        [FiscalYear]                      smallint NOT NULL,                                                          -- Fiscal year
        [DocumentNumber]                  nvarchar(20) NOT NULL,                                                      -- Document number
        [TaxLineNumber]                   int NOT NULL,                                                               -- Tax line sequence
        [TaxCodeId]                       bigint NOT NULL,                                                            -- Tax code
        [ConditionType]                   nvarchar(4) NOT NULL,                                                       -- Condition type
        [TaxJurisdictionCode]             nvarchar(15) NULL,                                                          -- Tax jurisdiction
        [TaxRatePercent]                  decimal(9,4) NOT NULL,                                                      -- Applied rate
        [TaxBaseAmountInLocalCurrency]    decimal(19,4) NOT NULL,                                                     -- Base amount in local currency
        [TaxAmountInLocalCurrency]        decimal(19,4) NOT NULL,                                                     -- Tax amount in local currency
        [TaxBaseAmountInDocumentCurrency] decimal(19,4) NOT NULL,                                                     -- Base amount in document currency
        [TaxAmountInDocumentCurrency]     decimal(19,4) NOT NULL,                                                     -- Tax amount in document currency
        [NonDeductibleAmount]             decimal(19,4) NULL,                                                         -- Non-deductible portion
        [TaxGLAccountId]                  bigint NOT NULL,                                                            -- Tax account posted to
        [IsOutputTax]                     bit NOT NULL CONSTRAINT [DF_fin_JournalEntryTax_IsOutputTax] DEFAULT (0),   -- Output tax (A) rather than input tax (V)
        [ReportingDate]                   date NULL,                                                                  -- Date used in the tax return
        [CreatedAt]                       datetime2(3) NOT NULL CONSTRAINT [DF_fin_JournalEntryTax_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                       nvarchar(64) NOT NULL,                                                      -- Creating user
        CONSTRAINT [PK_fin_JournalEntryTax] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_JournalEntryTax] UNIQUE ([TenantId], [CompanyCodeId], [FiscalYear], [DocumentNumber], [TaxLineNumber])
    );
END
GO

/* fin.OpenItem - Open item index for customers, vendors and open-item-managed G/L accounts (reference: BSID/BSIK/BSIS) */
IF OBJECT_ID(N'fin.OpenItem', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[OpenItem]
    (
        [Id]                               bigint IDENTITY(1,1) NOT NULL,                                             -- Surrogate key
        [TenantId]                         int NOT NULL,                                                              -- Owning tenant - every query is filtered by it
        [JournalEntryLineId]               bigint NOT NULL,                                                           -- Originating journal line
        [CompanyCodeId]                    bigint NOT NULL,                                                           -- Company code
        [FiscalYear]                       smallint NOT NULL,                                                         -- Fiscal year
        [DocumentNumber]                   nvarchar(20) NOT NULL,                                                     -- Document number
        [LineItemNumber]                   int NOT NULL,                                                              -- Line number
        [AccountType]                      nvarchar(1) NOT NULL,                                                      -- D, K, S
        [BusinessPartnerId]                bigint NULL,                                                               -- Customer / vendor
        [GLAccountId]                      bigint NOT NULL,                                                           -- Reconciliation or G/L account
        [SpecialGLIndicator]               nvarchar(1) NULL,                                                          -- Special G/L indicator
        [PostingDate]                      date NOT NULL,                                                             -- Posting date
        [DocumentDate]                     date NOT NULL,                                                             -- Document date
        [BaselineDate]                     date NULL,                                                                 -- Baseline date
        [DueDate]                          date NULL,                                                                 -- Net due date
        [DaysInArrears]                    int NULL,                                                                  -- Days overdue at the last aging run
        [DocumentCurrencyCode]             nvarchar(5) NOT NULL,                                                      -- Document currency
        [OriginalAmountInDocumentCurrency] decimal(19,4) NOT NULL,                                                    -- Original amount
        [OpenAmountInDocumentCurrency]     decimal(19,4) NOT NULL,                                                    -- Remaining open amount
        [OriginalAmountInLocalCurrency]    decimal(19,4) NOT NULL,                                                    -- Original amount in local currency
        [OpenAmountInLocalCurrency]        decimal(19,4) NOT NULL,                                                    -- Remaining open amount in local currency
        [ClearedAmountInDocumentCurrency]  decimal(19,4) NOT NULL,                                                    -- Amount already cleared
        [DebitCreditIndicator]             nvarchar(1) NOT NULL,                                                      -- S debit, H credit
        [PaymentTermsId]                   bigint NULL,                                                               -- Payment terms
        [CashDiscountAmount]               decimal(19,4) NULL,                                                        -- Cash discount still available
        [CashDiscountDueDate]              date NULL,                                                                 -- Last day for the cash discount
        [PaymentMethod]                    nvarchar(1) NULL,                                                          -- Payment method
        [PaymentBlockReason]               nvarchar(1) NULL,                                                          -- Payment block
        [DunningLevel]                     tinyint NOT NULL,                                                          -- Current dunning level
        [LastDunningDate]                  date NULL,                                                                 -- Last dunning date
        [DunningBlockReason]               nvarchar(1) NULL,                                                          -- Dunning block
        [AssignmentReference]              nvarchar(18) NULL,                                                         -- Allocation field
        [ReferenceDocumentNumber]          nvarchar(20) NULL,                                                         -- External reference
        [LineItemText]                     nvarchar(255) NULL,                                                        -- Item text
        [Status]                           nvarchar(20) NOT NULL,                                                     -- Open, PartiallyCleared, Cleared
        [ClearingDocumentNumber]           nvarchar(20) NULL,                                                         -- Clearing document
        [ClearingDate]                     date NULL,                                                                 -- Clearing date
        [IsDisputed]                       bit NOT NULL CONSTRAINT [DF_fin_OpenItem_IsDisputed] DEFAULT (0),          -- Item under dispute
        [CreatedAt]                        datetime2(3) NOT NULL CONSTRAINT [DF_fin_OpenItem_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                        nvarchar(64) NOT NULL,                                                     -- Creating user name
        [ModifiedAt]                       datetime2(3) NULL,                                                         -- Last change timestamp (UTC)
        [ModifiedBy]                       nvarchar(64) NULL,                                                         -- Last changing user name
        [RowVersion]                       rowversion NOT NULL,                                                       -- Optimistic concurrency token
        [IsActive]                         bit NOT NULL CONSTRAINT [DF_fin_OpenItem_IsActive] DEFAULT (1),            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_OpenItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_OpenItem] UNIQUE ([TenantId], [JournalEntryLineId])
    );
END
GO

/* fin.PaymentAllocation - Allocation of a payment to an open item (reference: REGUP) */
IF OBJECT_ID(N'fin.PaymentAllocation', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[PaymentAllocation]
    (
        [Id]                             bigint IDENTITY(1,1) NOT NULL,                                               -- Surrogate key
        [TenantId]                       int NOT NULL,                                                                -- Owning tenant - every query is filtered by it
        [PaymentHeaderId]                bigint NOT NULL,                                                             -- Payment
        [OpenItemId]                     bigint NOT NULL,                                                             -- Open item settled
        [AllocatedAmount]                decimal(19,4) NOT NULL,                                                      -- Amount applied
        [AllocatedAmountInLocalCurrency] decimal(19,4) NOT NULL,                                                      -- Amount applied in local currency
        [CashDiscountAmount]             decimal(19,4) NULL,                                                          -- Discount granted on this item
        [ExchangeRateDifference]         decimal(19,4) NULL,                                                          -- Realised exchange difference
        [AllocationType]                 nvarchar(20) NOT NULL,                                                       -- Full, Partial, Residual, OnAccount
        [ResidualOpenItemId]             bigint NULL,                                                                 -- Residual item created
        [CreatedAt]                      datetime2(3) NOT NULL CONSTRAINT [DF_fin_PaymentAllocation_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                      nvarchar(64) NOT NULL,                                                       -- Creating user name
        [ModifiedAt]                     datetime2(3) NULL,                                                           -- Last change timestamp (UTC)
        [ModifiedBy]                     nvarchar(64) NULL,                                                           -- Last changing user name
        [RowVersion]                     rowversion NOT NULL,                                                         -- Optimistic concurrency token
        [IsActive]                       bit NOT NULL CONSTRAINT [DF_fin_PaymentAllocation_IsActive] DEFAULT (1),     -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_PaymentAllocation] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_PaymentAllocation] UNIQUE ([TenantId], [PaymentHeaderId], [OpenItemId])
    );
END
GO

/* fin.PaymentHeader - Incoming or outgoing payment (reference: REGUH) */
IF OBJECT_ID(N'fin.PaymentHeader', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[PaymentHeader]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [CompanyCodeId]        bigint NOT NULL,                                                                       -- Company code
        [FiscalYear]           smallint NOT NULL,                                                                     -- Fiscal year
        [PaymentNumber]        nvarchar(20) NOT NULL,                                                                 -- Payment document number
        [PaymentDirection]     nvarchar(10) NOT NULL,                                                                 -- Incoming, Outgoing
        [PaymentType]          nvarchar(20) NOT NULL,                                                                 -- Manual, AutomaticRun, DownPayment, OnAccount, Refund
        [BusinessPartnerId]    bigint NOT NULL,                                                                       -- Customer or vendor paid
        [PaymentDate]          date NOT NULL,                                                                         -- Value date
        [PostingDate]          date NOT NULL,                                                                         -- Posting date
        [CurrencyCode]         nvarchar(5) NOT NULL,                                                                  -- Payment currency
        [ExchangeRate]         decimal(23,6) NULL,                                                                    -- Exchange rate
        [PaymentAmount]        decimal(19,4) NOT NULL,                                                                -- Gross payment amount
        [CashDiscountAmount]   decimal(19,4) NULL,                                                                    -- Cash discount taken
        [WithholdingTaxAmount] decimal(19,4) NULL,                                                                    -- Withholding tax deducted
        [BankChargeAmount]     decimal(19,4) NULL,                                                                    -- Bank charges
        [AllocatedAmount]      decimal(19,4) NOT NULL,                                                                -- Amount allocated to open items
        [UnallocatedAmount]    decimal(19,4) NOT NULL,                                                                -- On-account (unapplied) amount
        [PaymentMethod]        nvarchar(1) NOT NULL,                                                                  -- Payment method
        [HouseBankId]          bigint NULL,                                                                           -- House bank
        [HouseBankAccountId]   bigint NULL,                                                                           -- House bank account
        [PartnerBankDetailId]  nvarchar(4) NULL,                                                                      -- Partner bank details
        [CheckNumber]          nvarchar(20) NULL,                                                                     -- Cheque number
        [PaymentReference]     nvarchar(40) NULL,                                                                     -- Payment reference / end-to-end id
        [PaymentRunId]         bigint NULL,                                                                           -- Automatic payment run that created it
        [JournalEntryHeaderId] bigint NULL,                                                                           -- Accounting document created
        [ClearingDocumentId]   bigint NULL,                                                                           -- Clearing document created
        [Status]               nvarchar(20) NOT NULL,                                                                 -- Draft, PendingApproval, Approved, Posted, Sent, Confirmed, Reversed, Rejected
        [WorkflowInstanceId]   bigint NULL,                                                                           -- Approval workflow
        [IdempotencyKey]       uniqueidentifier NULL,                                                                 -- Duplicate protection
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_fin_PaymentHeader_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_fin_PaymentHeader_IsActive] DEFAULT (1),                   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_PaymentHeader] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_PaymentHeader] UNIQUE ([TenantId], [CompanyCodeId], [FiscalYear], [PaymentNumber])
    );
END
GO

/* fin.PaymentProposalItem - Open item selected by a payment proposal */
IF OBJECT_ID(N'fin.PaymentProposalItem', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[PaymentProposalItem]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [PaymentRunId]       bigint NOT NULL,                                                                         -- Payment run
        [OpenItemId]         bigint NOT NULL,                                                                         -- Open item
        [BusinessPartnerId]  bigint NOT NULL,                                                                         -- Partner
        [ProposedAmount]     decimal(19,4) NOT NULL,                                                                  -- Amount proposed
        [CashDiscountAmount] decimal(19,4) NULL,                                                                      -- Discount proposed
        [PaymentMethod]      nvarchar(1) NOT NULL,                                                                    -- Payment method proposed
        [HouseBankId]        bigint NULL,                                                                             -- House bank proposed
        [IsBlocked]          bit NOT NULL CONSTRAINT [DF_fin_PaymentProposalItem_IsBlocked] DEFAULT (0),              -- Excluded from the run
        [BlockReason]        nvarchar(255) NULL,                                                                      -- Reason for exclusion
        [IsEdited]           bit NOT NULL CONSTRAINT [DF_fin_PaymentProposalItem_IsEdited] DEFAULT (0),               -- Changed during proposal editing
        [PaymentHeaderId]    bigint NULL,                                                                             -- Payment created from this item
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_fin_PaymentProposalItem_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_fin_PaymentProposalItem_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_PaymentProposalItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_PaymentProposalItem] UNIQUE ([TenantId], [PaymentRunId], [OpenItemId])
    );
END
GO

/* fin.PaymentRun - Automatic payment run (F110) (reference: REGUV) */
IF OBJECT_ID(N'fin.PaymentRun', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[PaymentRun]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [RunDate]              date NOT NULL,                                                                         -- Run date
        [RunIdentifier]        nvarchar(6) NOT NULL,                                                                  -- Identification of the run
        [CompanyCodeId]        bigint NOT NULL,                                                                       -- Company code
        [PostingDate]          date NOT NULL,                                                                         -- Posting date of the payments
        [DocumentDate]         date NOT NULL,                                                                         -- Document date
        [NextPaymentDate]      date NOT NULL,                                                                         -- Date of the next planned run
        [PaymentMethods]       nvarchar(10) NOT NULL,                                                                 -- Payment methods considered
        [BusinessPartnerFrom]  nvarchar(10) NULL,                                                                     -- Partner range start
        [BusinessPartnerTo]    nvarchar(10) NULL,                                                                     -- Partner range end
        [CurrencyCode]         nvarchar(5) NULL,                                                                      -- Restrict to one currency
        [MinimumPaymentAmount] decimal(19,4) NULL,                                                                    -- Minimum amount per payment
        [Status]               nvarchar(20) NOT NULL,                                                                 -- Scheduled, ProposalRunning, ProposalReady, ProposalEdited, PaymentRunning, Posted, FileCreated, Cancelled
        [ProposalCreatedAt]    datetime2(3) NULL,                                                                     -- Proposal timestamp (UTC)
        [PaymentPostedAt]      datetime2(3) NULL,                                                                     -- Posting timestamp (UTC)
        [TotalPaymentAmount]   decimal(19,4) NULL,                                                                    -- Total paid
        [PaymentCount]         int NULL,                                                                              -- Number of payments created
        [ExceptionCount]       int NULL,                                                                              -- Number of blocked / exception items
        [PaymentFileUri]       nvarchar(500) NULL,                                                                    -- Generated bank file
        [ApprovedBy]           nvarchar(64) NULL,                                                                     -- Approver of the run
        [ApprovedAt]           datetime2(3) NULL,                                                                     -- Approval timestamp (UTC)
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_fin_PaymentRun_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_fin_PaymentRun_IsActive] DEFAULT (1),                      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_PaymentRun] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_PaymentRun] UNIQUE ([TenantId], [RunDate], [RunIdentifier])
    );
END
GO

/* fin.RecurringEntry - Recurring entry template (reference: BKDF) */
IF OBJECT_ID(N'fin.RecurringEntry', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[RecurringEntry]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [CompanyCodeId]           bigint NOT NULL,                                                                    -- Company code
        [RecurringDocumentNumber] nvarchar(20) NOT NULL,                                                              -- Recurring document number
        [DocumentTypeId]          bigint NOT NULL,                                                                    -- Document type used on execution
        [Description]             nvarchar(255) NOT NULL,                                                             -- Description
        [FirstRunDate]            date NOT NULL,                                                                      -- First run date
        [LastRunDate]             date NOT NULL,                                                                      -- Last run date
        [NextRunDate]             date NULL,                                                                          -- Next scheduled run
        [RunSchedule]             nvarchar(20) NOT NULL,                                                              -- Monthly, Quarterly, Yearly, RunDates
        [IntervalMonths]          tinyint NULL,                                                                       -- Months between runs
        [RunDayOfMonth]           tinyint NULL,                                                                       -- Day of the month
        [CurrencyCode]            nvarchar(5) NOT NULL,                                                               -- Currency
        [TotalAmount]             decimal(19,4) NOT NULL,                                                             -- Document amount
        [IsCopyAmountFromLastRun] bit NOT NULL CONSTRAINT [DF_fin_RecurringEntry_IsCopyAmountFromLastRun] DEFAULT (0),-- Reuse the last executed amount
        [NumberOfRunsExecuted]    int NOT NULL,                                                                       -- Executions so far
        [IsDeletionFlagged]       bit NOT NULL CONSTRAINT [DF_fin_RecurringEntry_IsDeletionFlagged] DEFAULT (0),      -- Marked for deletion
        [Status]                  nvarchar(20) NOT NULL,                                                              -- Active, Suspended, Completed, Deleted
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_fin_RecurringEntry_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_fin_RecurringEntry_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_RecurringEntry] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_RecurringEntry] UNIQUE ([TenantId], [CompanyCodeId], [RecurringDocumentNumber])
    );
END
GO

/* fin.RecurringEntryExecution - Log of each recurring-entry execution */
IF OBJECT_ID(N'fin.RecurringEntryExecution', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[RecurringEntryExecution]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [RecurringEntryId]     bigint NOT NULL,                                                                       -- Template executed
        [ExecutionDate]        date NOT NULL,                                                                         -- Execution date
        [PostingDate]          date NOT NULL,                                                                         -- Posting date used
        [JournalEntryHeaderId] bigint NULL,                                                                           -- Document created
        [Status]               nvarchar(20) NOT NULL,                                                                 -- Success, Failed, Skipped
        [Message]              nvarchar(500) NULL,                                                                    -- Result message
        [ExecutedAt]           datetime2(3) NOT NULL,                                                                 -- Execution timestamp (UTC)
        [ExecutedBy]           nvarchar(64) NOT NULL,                                                                 -- Executing user or job
        CONSTRAINT [PK_fin_RecurringEntryExecution] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_RecurringEntryExecution] UNIQUE ([TenantId], [RecurringEntryId], [ExecutionDate])
    );
END
GO

/* fin.RecurringEntryLine - Line of a recurring entry template */
IF OBJECT_ID(N'fin.RecurringEntryLine', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[RecurringEntryLine]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [RecurringEntryId]    bigint NOT NULL,                                                                        -- Owning template
        [LineItemNumber]      int NOT NULL,                                                                           -- Line number
        [PostingKey]          nvarchar(2) NOT NULL,                                                                   -- Posting key
        [GLAccountId]         bigint NULL,                                                                            -- G/L account
        [BusinessPartnerId]   bigint NULL,                                                                            -- Customer / vendor
        [Amount]              decimal(19,4) NOT NULL,                                                                 -- Amount
        [TaxCodeId]           bigint NULL,                                                                            -- Tax code
        [CostCenterId]        bigint NULL,                                                                            -- Cost centre
        [ProfitCenterId]      bigint NULL,                                                                            -- Profit centre
        [InternalOrderId]     bigint NULL,                                                                            -- Internal order
        [SegmentId]           bigint NULL,                                                                            -- Segment
        [AssignmentReference] nvarchar(18) NULL,                                                                      -- Allocation field
        [LineItemText]        nvarchar(255) NULL,                                                                     -- Item text
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_fin_RecurringEntryLine_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_fin_RecurringEntryLine_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_RecurringEntryLine] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_RecurringEntryLine] UNIQUE ([TenantId], [RecurringEntryId], [LineItemNumber])
    );
END
GO

/* fin.VendorInvoice - Vendor invoice / credit memo header (reference: RBKP) */
IF OBJECT_ID(N'fin.VendorInvoice', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[VendorInvoice]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [CompanyCodeId]            bigint NOT NULL,                                                                   -- Company code
        [FiscalYear]               smallint NOT NULL,                                                                 -- Fiscal year
        [InvoiceNumber]            nvarchar(20) NOT NULL,                                                             -- Internal invoice number
        [InvoiceType]              nvarchar(20) NOT NULL,                                                             -- Invoice, CreditMemo, DownPaymentRequest, SubsequentDebit
        [BusinessPartnerId]        bigint NOT NULL,                                                                   -- Vendor (BP with vendor role)
        [PayeeBusinessPartnerId]   bigint NULL,                                                                       -- Alternative payee
        [VendorInvoiceNumber]      nvarchar(20) NOT NULL,                                                             -- Vendor's own invoice number - duplicate check
        [InvoiceDate]              date NOT NULL,                                                                     -- Invoice date
        [PostingDate]              date NOT NULL,                                                                     -- Posting date
        [ReceiptDate]              date NULL,                                                                         -- Date the invoice was received
        [CurrencyCode]             nvarchar(5) NOT NULL,                                                              -- Invoice currency
        [ExchangeRate]             decimal(23,6) NULL,                                                                -- Exchange rate
        [NetAmount]                decimal(19,4) NOT NULL,                                                            -- Net amount
        [TaxAmount]                decimal(19,4) NOT NULL,                                                            -- Tax amount
        [GrossAmount]              decimal(19,4) NOT NULL,                                                            -- Gross amount
        [WithholdingTaxAmount]     decimal(19,4) NULL,                                                                -- Withholding tax
        [PaymentTermsId]           bigint NULL,                                                                       -- Payment terms
        [BaselineDate]             date NULL,                                                                         -- Baseline date
        [DueDate]                  date NULL,                                                                         -- Due date
        [PaymentMethod]            nvarchar(1) NULL,                                                                  -- Payment method
        [PaymentBlockReason]       nvarchar(1) NULL,                                                                  -- Payment block
        [HouseBankId]              bigint NULL,                                                                       -- House bank for payment
        [PartnerBankDetailId]      nvarchar(4) NULL,                                                                  -- Vendor bank details used
        [PurchasingOrganizationId] bigint NULL,                                                                       -- Purchasing organisation
        [PurchaseOrderNumber]      nvarchar(20) NULL,                                                                 -- Purchase order reference (future module)
        [HeaderText]               nvarchar(255) NULL,                                                                -- Header text
        [Status]                   nvarchar(20) NOT NULL,                                                             -- Draft, Parked, PendingApproval, Approved, Posted, PartiallyCleared, Cleared, Reversed, Cancelled
        [JournalEntryHeaderId]     bigint NULL,                                                                       -- Accounting document created
        [PaidAmount]               decimal(19,4) NOT NULL,                                                            -- Amount paid so far
        [OpenAmount]               decimal(19,4) NOT NULL,                                                            -- Amount still open
        [WorkflowInstanceId]       bigint NULL,                                                                       -- Approval workflow
        [IdempotencyKey]           uniqueidentifier NULL,                                                             -- Duplicate protection
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_fin_VendorInvoice_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_fin_VendorInvoice_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_VendorInvoice] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_VendorInvoice] UNIQUE ([TenantId], [CompanyCodeId], [FiscalYear], [InvoiceNumber])
    );
END
GO

/* fin.VendorInvoiceItem - Vendor invoice line (reference: RSEG) */
IF OBJECT_ID(N'fin.VendorInvoiceItem', N'U') IS NULL
BEGIN
    CREATE TABLE [fin].[VendorInvoiceItem]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [VendorInvoiceId]    bigint NOT NULL,                                                                         -- Owning invoice
        [ItemNumber]         int NOT NULL,                                                                            -- Item number
        [ExpenseGLAccountId] bigint NOT NULL,                                                                         -- Expense / balance sheet account
        [Description]        nvarchar(255) NOT NULL,                                                                  -- Item description
        [Quantity]           decimal(23,6) NULL,                                                                      -- Quantity
        [UnitOfMeasure]      nvarchar(3) NULL,                                                                        -- Unit of measure
        [UnitPrice]          decimal(19,4) NULL,                                                                      -- Unit price
        [NetAmount]          decimal(19,4) NOT NULL,                                                                  -- Net amount
        [TaxCodeId]          bigint NULL,                                                                             -- Tax code
        [TaxAmount]          decimal(19,4) NULL,                                                                      -- Tax amount
        [IsNonDeductibleTax] bit NOT NULL CONSTRAINT [DF_fin_VendorInvoiceItem_IsNonDeductibleTax] DEFAULT (0),       -- Tax not deductible - added to the expense
        [CostCenterId]       bigint NULL,                                                                             -- Cost centre
        [ProfitCenterId]     bigint NULL,                                                                             -- Profit centre
        [InternalOrderId]    bigint NULL,                                                                             -- Internal order
        [AssetId]            bigint NULL,                                                                             -- Asset for vendor acquisition
        [SegmentId]          bigint NULL,                                                                             -- Segment
        [FunctionalAreaId]   bigint NULL,                                                                             -- Functional area
        [BusinessAreaId]     bigint NULL,                                                                             -- Business area
        [PlantId]            bigint NULL,                                                                             -- Plant
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_fin_VendorInvoiceItem_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_fin_VendorInvoiceItem_IsActive] DEFAULT (1),                 -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_fin_VendorInvoiceItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_fin_VendorInvoiceItem] UNIQUE ([TenantId], [VendorInvoiceId], [ItemNumber])
    );
END
GO
