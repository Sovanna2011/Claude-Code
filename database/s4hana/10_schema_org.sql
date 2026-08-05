/* ============================================================================
   S/4HANA-inspired ERP - schema [org]
   Enterprise structure (22 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* org.Branch - Branch / office of a company code (reference: J_1BBRANCH) */
IF OBJECT_ID(N'org.Branch', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[Branch]
    (
        [Id]             bigint IDENTITY(1,1) NOT NULL,                                                               -- Surrogate key
        [TenantId]       int NOT NULL,                                                                                -- Owning tenant - every query is filtered by it
        [Branch]         nvarchar(4) NOT NULL,                                                                        -- Branch key
        [CompanyCodeId]  bigint NOT NULL,                                                                             -- Owning company code
        [Name]           nvarchar(60) NOT NULL,                                                                       -- Branch name
        [LocationId]     bigint NULL,                                                                                 -- Physical location
        [BusinessAreaId] bigint NULL,                                                                                 -- Default business area
        [ProfitCenterId] bigint NULL,                                                                                 -- Default profit centre
        [TaxNumber]      nvarchar(20) NULL,                                                                           -- Branch tax registration
        [IsHeadOffice]   bit NOT NULL CONSTRAINT [DF_org_Branch_IsHeadOffice] DEFAULT (0),                            -- Head-office branch flag
        [ValidFrom]      date NOT NULL,                                                                               -- First day the record is valid
        [ValidTo]        date NOT NULL,                                                                               -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]      datetime2(3) NOT NULL CONSTRAINT [DF_org_Branch_CreatedAt] DEFAULT (SYSUTCDATETIME()),       -- Creation timestamp (UTC)
        [CreatedBy]      nvarchar(64) NOT NULL,                                                                       -- Creating user name
        [ModifiedAt]     datetime2(3) NULL,                                                                           -- Last change timestamp (UTC)
        [ModifiedBy]     nvarchar(64) NULL,                                                                           -- Last changing user name
        [RowVersion]     rowversion NOT NULL,                                                                         -- Optimistic concurrency token
        [IsActive]       bit NOT NULL CONSTRAINT [DF_org_Branch_IsActive] DEFAULT (1),                                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_Branch] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_Branch] UNIQUE ([TenantId], [Branch])
    );
END
GO

/* org.BusinessArea - Cross-company-code reporting unit (reference: TGSB) */
IF OBJECT_ID(N'org.BusinessArea', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[BusinessArea]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [BusinessArea]              nvarchar(4) NOT NULL,                                                             -- Business area key
        [Name]                      nvarchar(40) NOT NULL,                                                            -- Description
        [ConsolidationBusinessArea] nvarchar(4) NULL,                                                                 -- Consolidation business area
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_org_BusinessArea_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_org_BusinessArea_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_BusinessArea] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_BusinessArea] UNIQUE ([TenantId], [BusinessArea])
    );
END
GO

/* org.Company - Legal or consolidation group entity (reference: T880) */
IF OBJECT_ID(N'org.Company', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[Company]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [CompanyCodeGroup]      nvarchar(6) NOT NULL,                                                                 -- Company key (trading partner id)
        [Name]                  nvarchar(60) NOT NULL,                                                                -- Company name
        [Name2]                 nvarchar(60) NULL,                                                                    -- Name line 2
        [CountryCode]           nvarchar(3) NOT NULL,                                                                 -- Country of registration
        [GroupCurrencyCode]     nvarchar(5) NOT NULL,                                                                 -- Consolidation (group) currency
        [LanguageCode]          nvarchar(2) NOT NULL,                                                                 -- Correspondence language
        [RegistrationNumber]    nvarchar(20) NULL,                                                                    -- Commercial register number
        [TaxNumber]             nvarchar(20) NULL,                                                                    -- Group tax number
        [AccountingStandard]    nvarchar(10) NULL,                                                                    -- IFRS, LOCAL, USGAAP
        [ConsolidationRelevant] bit NOT NULL CONSTRAINT [DF_org_Company_ConsolidationRelevant] DEFAULT (0),           -- Included in consolidation
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_org_Company_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_org_Company_IsActive] DEFAULT (1),                        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_Company] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_Company] UNIQUE ([TenantId], [CompanyCodeGroup])
    );
END
GO

/* org.CompanyCode - Smallest unit producing a complete set of books (reference: T001) */
IF OBJECT_ID(N'org.CompanyCode', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[CompanyCode]
    (
        [Id]                              bigint IDENTITY(1,1) NOT NULL,                                              -- Surrogate key
        [TenantId]                        int NOT NULL,                                                               -- Owning tenant - every query is filtered by it
        [CompanyCode]                     nvarchar(4) NOT NULL,                                                       -- Company code key (e.g. KH01)
        [CompanyId]                       bigint NOT NULL,                                                            -- Owning company
        [Name]                            nvarchar(60) NOT NULL,                                                      -- Company code name
        [Name2]                           nvarchar(60) NULL,                                                          -- Additional name
        [City]                            nvarchar(40) NULL,                                                          -- Registered city
        [CountryCode]                     nvarchar(3) NOT NULL,                                                       -- Country - drives tax and address format
        [LocalCurrencyCode]               nvarchar(5) NOT NULL,                                                       -- Company code (local) currency
        [GroupCurrencyCode]               nvarchar(5) NULL,                                                           -- Group currency, defaulted from company
        [HardCurrencyCode]                nvarchar(5) NULL,                                                           -- Hard currency for high-inflation countries
        [IndexCurrencyCode]               nvarchar(5) NULL,                                                           -- Index-based currency
        [LanguageCode]                    nvarchar(2) NOT NULL,                                                       -- Correspondence language
        [ChartOfAccountsId]               bigint NOT NULL,                                                            -- Operational chart of accounts
        [CountryChartOfAccountsId]        bigint NULL,                                                                -- Country-specific chart of accounts
        [FiscalYearVariantId]             bigint NOT NULL,                                                            -- Fiscal year variant
        [PostingPeriodVariantId]          bigint NOT NULL,                                                            -- Posting period variant
        [FieldStatusVariantId]            bigint NOT NULL,                                                            -- Field status variant
        [CreditControlAreaId]             bigint NULL,                                                                -- Default credit control area
        [ControllingAreaId]               bigint NULL,                                                                -- Assigned controlling area
        [TaxJurisdictionSchemaId]         bigint NULL,                                                                -- Tax jurisdiction schema
        [VatRegistrationNumber]           nvarchar(20) NULL,                                                          -- VAT registration number
        [AddressId]                       bigint NULL,                                                                -- Registered address
        [MaxExchangeRateDeviationPercent] decimal(9,4) NULL,                                                          -- Warning threshold on manual rates
        [ProposeFiscalYear]               bit NOT NULL CONSTRAINT [DF_org_CompanyCode_ProposeFiscalYear] DEFAULT (0), -- Propose fiscal year on entry screens
        [NegativePostingAllowed]          bit NOT NULL CONSTRAINT [DF_org_CompanyCode_NegativePostingAllowed] DEFAULT (0),-- Allow negative postings on reversal
        [BusinessAreaFinancialStatements] bit NOT NULL CONSTRAINT [DF_org_CompanyCode_BusinessAreaFinancialStatements] DEFAULT (0),-- Business-area balance sheets required
        [ProfitCenterMandatory]           bit NOT NULL CONSTRAINT [DF_org_CompanyCode_ProfitCenterMandatory] DEFAULT (0),-- Profit centre required on every line
        [SegmentMandatory]                bit NOT NULL CONSTRAINT [DF_org_CompanyCode_SegmentMandatory] DEFAULT (0),  -- Segment required on every line
        [DocumentEntryScreenVariant]      nvarchar(4) NULL,                                                           -- Entry screen variant
        [IsProductive]                    bit NOT NULL CONSTRAINT [DF_org_CompanyCode_IsProductive] DEFAULT (0),      -- Productive - deletion of test data blocked
        [ValidFrom]                       date NOT NULL,                                                              -- First day the company code may post
        [ValidTo]                         date NOT NULL,                                                              -- Last day the company code may post
        [CreatedAt]                       datetime2(3) NOT NULL CONSTRAINT [DF_org_CompanyCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                       nvarchar(64) NOT NULL,                                                      -- Creating user name
        [ModifiedAt]                      datetime2(3) NULL,                                                          -- Last change timestamp (UTC)
        [ModifiedBy]                      nvarchar(64) NULL,                                                          -- Last changing user name
        [RowVersion]                      rowversion NOT NULL,                                                        -- Optimistic concurrency token
        [IsActive]                        bit NOT NULL CONSTRAINT [DF_org_CompanyCode_IsActive] DEFAULT (1),          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_CompanyCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_CompanyCode] UNIQUE ([TenantId], [CompanyCode])
    );
END
GO

/* org.ControllingArea - Controlling area - the CO boundary (reference: TKA01) */
IF OBJECT_ID(N'org.ControllingArea', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[ControllingArea]
    (
        [Id]                            bigint IDENTITY(1,1) NOT NULL,                                                -- Surrogate key
        [TenantId]                      int NOT NULL,                                                                 -- Owning tenant - every query is filtered by it
        [ControllingArea]               nvarchar(4) NOT NULL,                                                         -- Controlling area key
        [Name]                          nvarchar(40) NOT NULL,                                                        -- Description
        [CurrencyCode]                  nvarchar(5) NOT NULL,                                                         -- Controlling area currency
        [CurrencyTypeCode]              nvarchar(2) NOT NULL,                                                         -- 10 company code, 20 controlling area, 30 group
        [ChartOfAccountsId]             bigint NOT NULL,                                                              -- Chart of accounts - must match assigned company codes
        [FiscalYearVariantId]           bigint NOT NULL,                                                              -- Fiscal year variant
        [CostCenterStandardHierarchy]   nvarchar(12) NOT NULL,                                                        -- Standard cost centre hierarchy id
        [ProfitCenterStandardHierarchy] nvarchar(12) NULL,                                                            -- Standard profit centre hierarchy id
        [AssignmentControl]             nvarchar(1) NOT NULL,                                                         -- 1 one company code, 2 cross-company-code
        [OperatingConcernId]            bigint NULL,                                                                  -- Assigned operating concern
        [ReconciliationLedgerActive]    bit NOT NULL CONSTRAINT [DF_org_ControllingArea_ReconciliationLedgerActive] DEFAULT (0),-- Reconciliation ledger active
        [ProfitCenterAccountingActive]  bit NOT NULL CONSTRAINT [DF_org_ControllingArea_ProfitCenterAccountingActive] DEFAULT (0),-- Profit centre accounting active
        [ActivateCommitmentManagement]  bit NOT NULL CONSTRAINT [DF_org_ControllingArea_ActivateCommitmentManagement] DEFAULT (0),-- Commitment management active
        [ValidFromYear]                 smallint NOT NULL,                                                            -- First fiscal year the area is usable
        [CreatedAt]                     datetime2(3) NOT NULL CONSTRAINT [DF_org_ControllingArea_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                     nvarchar(64) NOT NULL,                                                        -- Creating user name
        [ModifiedAt]                    datetime2(3) NULL,                                                            -- Last change timestamp (UTC)
        [ModifiedBy]                    nvarchar(64) NULL,                                                            -- Last changing user name
        [RowVersion]                    rowversion NOT NULL,                                                          -- Optimistic concurrency token
        [IsActive]                      bit NOT NULL CONSTRAINT [DF_org_ControllingArea_IsActive] DEFAULT (1),        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_ControllingArea] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_ControllingArea] UNIQUE ([TenantId], [ControllingArea])
    );
END
GO

/* org.ControllingAreaCompanyCode - Company codes assigned to a controlling area (reference: TKA02) */
IF OBJECT_ID(N'org.ControllingAreaCompanyCode', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[ControllingAreaCompanyCode]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [ControllingAreaId] bigint NOT NULL,                                                                          -- Controlling area
        [CompanyCodeId]     bigint NOT NULL,                                                                          -- Company code
        [ValidFrom]         date NOT NULL,                                                                            -- First day the record is valid
        [ValidTo]           date NOT NULL,                                                                            -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_org_ControllingAreaCompanyCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_org_ControllingAreaCompanyCode_IsActive] DEFAULT (1),         -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_ControllingAreaCompanyCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_ControllingAreaCompanyCode] UNIQUE ([TenantId], [ControllingAreaId], [CompanyCodeId])
    );
END
GO

/* org.CreditControlArea - Credit control area (reference: T014) */
IF OBJECT_ID(N'org.CreditControlArea', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[CreditControlArea]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [CreditControlArea]       nvarchar(4) NOT NULL,                                                               -- Credit control area key
        [Name]                    nvarchar(40) NOT NULL,                                                              -- Description
        [CurrencyCode]            nvarchar(5) NOT NULL,                                                               -- Credit limit currency
        [DefaultCreditLimit]      decimal(19,4) NULL,                                                                 -- Default limit for new customers
        [RiskCategory]            nvarchar(3) NULL,                                                                   -- Default risk category
        [UpdateGroup]             nvarchar(6) NULL,                                                                   -- Credit exposure update group
        [AllOrganizationsAllowed] bit NOT NULL CONSTRAINT [DF_org_CreditControlArea_AllOrganizationsAllowed] DEFAULT (0),-- May be used by any company code
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_org_CreditControlArea_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_org_CreditControlArea_IsActive] DEFAULT (1),            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_CreditControlArea] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_CreditControlArea] UNIQUE ([TenantId], [CreditControlArea])
    );
END
GO

/* org.Department - Internal department, used for approval routing (reference: T527X) */
IF OBJECT_ID(N'org.Department', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[Department]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [DepartmentCode]           nvarchar(10) NOT NULL,                                                             -- Department key
        [CompanyCodeId]            bigint NOT NULL,                                                                   -- Owning company code
        [ParentDepartmentId]       bigint NULL,                                                                       -- Parent department (hierarchy)
        [Name]                     nvarchar(60) NOT NULL,                                                             -- Department name
        [CostCenterId]             bigint NULL,                                                                       -- Default cost centre
        [ManagerBusinessPartnerId] bigint NULL,                                                                       -- Head of department (BP, employee role)
        [ValidFrom]                date NOT NULL,                                                                     -- First day the record is valid
        [ValidTo]                  date NOT NULL,                                                                     -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_org_Department_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_org_Department_IsActive] DEFAULT (1),                  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_Department] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_Department] UNIQUE ([TenantId], [DepartmentCode])
    );
END
GO

/* org.DistributionChannel - Distribution channel (reference: TVTW) */
IF OBJECT_ID(N'org.DistributionChannel', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[DistributionChannel]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [DistributionChannel] nvarchar(2) NOT NULL,                                                                   -- Distribution channel key
        [Name]                nvarchar(40) NOT NULL,                                                                  -- Description
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_org_DistributionChannel_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_org_DistributionChannel_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_DistributionChannel] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_DistributionChannel] UNIQUE ([TenantId], [DistributionChannel])
    );
END
GO

/* org.Division - Product division (reference: TSPA) */
IF OBJECT_ID(N'org.Division', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[Division]
    (
        [Id]         bigint IDENTITY(1,1) NOT NULL,                                                                   -- Surrogate key
        [TenantId]   int NOT NULL,                                                                                    -- Owning tenant - every query is filtered by it
        [Division]   nvarchar(2) NOT NULL,                                                                            -- Division key
        [Name]       nvarchar(40) NOT NULL,                                                                           -- Description
        [CreatedAt]  datetime2(3) NOT NULL CONSTRAINT [DF_org_Division_CreatedAt] DEFAULT (SYSUTCDATETIME()),         -- Creation timestamp (UTC)
        [CreatedBy]  nvarchar(64) NOT NULL,                                                                           -- Creating user name
        [ModifiedAt] datetime2(3) NULL,                                                                               -- Last change timestamp (UTC)
        [ModifiedBy] nvarchar(64) NULL,                                                                               -- Last changing user name
        [RowVersion] rowversion NOT NULL,                                                                             -- Optimistic concurrency token
        [IsActive]   bit NOT NULL CONSTRAINT [DF_org_Division_IsActive] DEFAULT (1),                                  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_Division] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_Division] UNIQUE ([TenantId], [Division])
    );
END
GO

/* org.FactoryCalendar - Working-day calendar for due-date and depreciation calculation (reference: TFACD) */
IF OBJECT_ID(N'org.FactoryCalendar', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[FactoryCalendar]
    (
        [Id]              bigint IDENTITY(1,1) NOT NULL,                                                              -- Surrogate key
        [TenantId]        int NOT NULL,                                                                               -- Owning tenant - every query is filtered by it
        [CalendarCode]    nvarchar(2) NOT NULL,                                                                       -- Factory calendar key
        [Name]            nvarchar(40) NOT NULL,                                                                      -- Description
        [CountryCode]     nvarchar(3) NULL,                                                                           -- Public holiday country
        [WorkingDaysMask] nvarchar(7) NOT NULL,                                                                       -- 7 chars Mon..Sun, X = working day
        [ValidFromYear]   smallint NOT NULL,                                                                          -- First year covered
        [ValidToYear]     smallint NOT NULL,                                                                          -- Last year covered
        [CreatedAt]       datetime2(3) NOT NULL CONSTRAINT [DF_org_FactoryCalendar_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]       nvarchar(64) NOT NULL,                                                                      -- Creating user name
        [ModifiedAt]      datetime2(3) NULL,                                                                          -- Last change timestamp (UTC)
        [ModifiedBy]      nvarchar(64) NULL,                                                                          -- Last changing user name
        [RowVersion]      rowversion NOT NULL,                                                                        -- Optimistic concurrency token
        [IsActive]        bit NOT NULL CONSTRAINT [DF_org_FactoryCalendar_IsActive] DEFAULT (1),                      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_FactoryCalendar] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_FactoryCalendar] UNIQUE ([TenantId], [CalendarCode])
    );
END
GO

/* org.FactoryCalendarHoliday - Non-working day of a factory calendar (reference: THOC) */
IF OBJECT_ID(N'org.FactoryCalendarHoliday', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[FactoryCalendarHoliday]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [FactoryCalendarId] bigint NOT NULL,                                                                          -- Owning calendar
        [HolidayDate]       date NOT NULL,                                                                            -- Non-working date
        [Name]              nvarchar(40) NOT NULL,                                                                    -- Holiday name
        [IsHalfDay]         bit NOT NULL CONSTRAINT [DF_org_FactoryCalendarHoliday_IsHalfDay] DEFAULT (0),            -- Half working day
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_org_FactoryCalendarHoliday_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_org_FactoryCalendarHoliday_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_FactoryCalendarHoliday] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_FactoryCalendarHoliday] UNIQUE ([TenantId], [FactoryCalendarId], [HolidayDate])
    );
END
GO

/* org.FunctionalArea - Cost-of-sales classification (reference: TFKB) */
IF OBJECT_ID(N'org.FunctionalArea', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[FunctionalArea]
    (
        [Id]             bigint IDENTITY(1,1) NOT NULL,                                                               -- Surrogate key
        [TenantId]       int NOT NULL,                                                                                -- Owning tenant - every query is filtered by it
        [FunctionalArea] nvarchar(16) NOT NULL,                                                                       -- Functional area key
        [Name]           nvarchar(40) NOT NULL,                                                                       -- Description
        [CreatedAt]      datetime2(3) NOT NULL CONSTRAINT [DF_org_FunctionalArea_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]      nvarchar(64) NOT NULL,                                                                       -- Creating user name
        [ModifiedAt]     datetime2(3) NULL,                                                                           -- Last change timestamp (UTC)
        [ModifiedBy]     nvarchar(64) NULL,                                                                           -- Last changing user name
        [RowVersion]     rowversion NOT NULL,                                                                         -- Optimistic concurrency token
        [IsActive]       bit NOT NULL CONSTRAINT [DF_org_FunctionalArea_IsActive] DEFAULT (1),                        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_FunctionalArea] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_FunctionalArea] UNIQUE ([TenantId], [FunctionalArea])
    );
END
GO

/* org.Location - Physical location (address anchor) (reference: TLOC) */
IF OBJECT_ID(N'org.Location', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[Location]
    (
        [Id]           bigint IDENTITY(1,1) NOT NULL,                                                                 -- Surrogate key
        [TenantId]     int NOT NULL,                                                                                  -- Owning tenant - every query is filtered by it
        [LocationCode] nvarchar(10) NOT NULL,                                                                         -- Location key
        [Name]         nvarchar(60) NOT NULL,                                                                         -- Location name
        [AddressId]    bigint NULL,                                                                                   -- Address
        [CountryCode]  nvarchar(3) NOT NULL,                                                                          -- Country
        [RegionCode]   nvarchar(3) NULL,                                                                              -- Region / province
        [Latitude]     decimal(9,6) NULL,                                                                             -- Geo latitude
        [Longitude]    decimal(9,6) NULL,                                                                             -- Geo longitude
        [CreatedAt]    datetime2(3) NOT NULL CONSTRAINT [DF_org_Location_CreatedAt] DEFAULT (SYSUTCDATETIME()),       -- Creation timestamp (UTC)
        [CreatedBy]    nvarchar(64) NOT NULL,                                                                         -- Creating user name
        [ModifiedAt]   datetime2(3) NULL,                                                                             -- Last change timestamp (UTC)
        [ModifiedBy]   nvarchar(64) NULL,                                                                             -- Last changing user name
        [RowVersion]   rowversion NOT NULL,                                                                           -- Optimistic concurrency token
        [IsActive]     bit NOT NULL CONSTRAINT [DF_org_Location_IsActive] DEFAULT (1),                                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_Location] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_Location] UNIQUE ([TenantId], [LocationCode])
    );
END
GO

/* org.OperatingConcern - Operating concern for profitability analysis (reference: TKEB) */
IF OBJECT_ID(N'org.OperatingConcern', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[OperatingConcern]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [OperatingConcern]    nvarchar(4) NOT NULL,                                                                   -- Operating concern key
        [Name]                nvarchar(40) NOT NULL,                                                                  -- Description
        [CurrencyCode]        nvarchar(5) NOT NULL,                                                                   -- Operating concern currency
        [FiscalYearVariantId] bigint NOT NULL,                                                                        -- Fiscal year variant
        [IsCostingBased]      bit NOT NULL CONSTRAINT [DF_org_OperatingConcern_IsCostingBased] DEFAULT (0),           -- Costing-based profitability analysis active
        [IsAccountBased]      bit NOT NULL CONSTRAINT [DF_org_OperatingConcern_IsAccountBased] DEFAULT (0),           -- Account-based profitability analysis active
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_org_OperatingConcern_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_org_OperatingConcern_IsActive] DEFAULT (1),                 -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_OperatingConcern] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_OperatingConcern] UNIQUE ([TenantId], [OperatingConcern])
    );
END
GO

/* org.OrganizationalAssignment - Validity-dated assignment between any two organisational objects (reference: assignment views in SPRO) */
IF OBJECT_ID(N'org.OrganizationalAssignment', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[OrganizationalAssignment]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [SourceObjectType] nvarchar(20) NOT NULL,                                                                     -- e.g. Plant, SalesOrganization
        [SourceObjectId]   bigint NOT NULL,                                                                           -- Id of the source object
        [TargetObjectType] nvarchar(20) NOT NULL,                                                                     -- e.g. CompanyCode, ControllingArea
        [TargetObjectId]   bigint NOT NULL,                                                                           -- Id of the target object
        [AssignmentType]   nvarchar(20) NOT NULL,                                                                     -- AssignedTo, SuppliesTo, ReportsTo
        [IsPrimary]        bit NOT NULL CONSTRAINT [DF_org_OrganizationalAssignment_IsPrimary] DEFAULT (0),           -- Primary assignment when several exist
        [ValidFrom]        date NOT NULL,                                                                             -- First day the record is valid
        [ValidTo]          date NOT NULL,                                                                             -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]        datetime2(3) NOT NULL CONSTRAINT [DF_org_OrganizationalAssignment_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]        nvarchar(64) NOT NULL,                                                                     -- Creating user name
        [ModifiedAt]       datetime2(3) NULL,                                                                         -- Last change timestamp (UTC)
        [ModifiedBy]       nvarchar(64) NULL,                                                                         -- Last changing user name
        [RowVersion]       rowversion NOT NULL,                                                                       -- Optimistic concurrency token
        [IsActive]         bit NOT NULL CONSTRAINT [DF_org_OrganizationalAssignment_IsActive] DEFAULT (1),            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_OrganizationalAssignment] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_OrganizationalAssignment] UNIQUE ([TenantId], [SourceObjectType], [SourceObjectId], [TargetObjectType], [TargetObjectId], [AssignmentType])
    );
END
GO

/* org.Plant - Operational site / production or storage location (reference: T001W) */
IF OBJECT_ID(N'org.Plant', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[Plant]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [Plant]                    nvarchar(4) NOT NULL,                                                              -- Plant key
        [CompanyCodeId]            bigint NOT NULL,                                                                   -- Owning company code
        [Name]                     nvarchar(60) NOT NULL,                                                             -- Plant name
        [CountryCode]              nvarchar(3) NOT NULL,                                                              -- Country
        [City]                     nvarchar(40) NULL,                                                                 -- City
        [AddressId]                bigint NULL,                                                                       -- Plant address
        [PurchasingOrganizationId] bigint NULL,                                                                       -- Default purchasing organisation
        [ValuationArea]            nvarchar(4) NULL,                                                                  -- Valuation area (future inventory module)
        [FactoryCalendarId]        bigint NULL,                                                                       -- Factory calendar
        [TaxJurisdictionCode]      nvarchar(15) NULL,                                                                 -- Tax jurisdiction
        [ValidFrom]                date NOT NULL,                                                                     -- First day the record is valid
        [ValidTo]                  date NOT NULL,                                                                     -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_org_Plant_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_org_Plant_IsActive] DEFAULT (1),                       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_Plant] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_Plant] UNIQUE ([TenantId], [Plant])
    );
END
GO

/* org.PurchasingOrganization - Purchasing organisation (reference: T024E) */
IF OBJECT_ID(N'org.PurchasingOrganization', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[PurchasingOrganization]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [PurchasingOrganization] nvarchar(4) NOT NULL,                                                                -- Purchasing organisation key
        [CompanyCodeId]          bigint NULL,                                                                         -- Assigned company code (blank = cross-company)
        [Name]                   nvarchar(60) NOT NULL,                                                               -- Description
        [IsCrossCompany]         bit NOT NULL CONSTRAINT [DF_org_PurchasingOrganization_IsCrossCompany] DEFAULT (0),  -- May purchase for several company codes
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_org_PurchasingOrganization_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_org_PurchasingOrganization_IsActive] DEFAULT (1),        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_PurchasingOrganization] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_PurchasingOrganization] UNIQUE ([TenantId], [PurchasingOrganization])
    );
END
GO

/* org.SalesArea - Permitted sales organisation / channel / division combination (reference: TVTA) */
IF OBJECT_ID(N'org.SalesArea', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[SalesArea]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [SalesOrganizationId]   bigint NOT NULL,                                                                      -- Sales organisation
        [DistributionChannelId] bigint NOT NULL,                                                                      -- Distribution channel
        [DivisionId]            bigint NOT NULL,                                                                      -- Division
        [IsActive]              bit NOT NULL CONSTRAINT [DF_org_SalesArea_IsActive] DEFAULT (1),                      -- Combination allowed
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_org_SalesArea_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        CONSTRAINT [PK_org_SalesArea] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_SalesArea] UNIQUE ([TenantId], [SalesOrganizationId], [DistributionChannelId], [DivisionId])
    );
END
GO

/* org.SalesOrganization - Sales organisation (reference: TVKO) */
IF OBJECT_ID(N'org.SalesOrganization', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[SalesOrganization]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [SalesOrganization] nvarchar(4) NOT NULL,                                                                     -- Sales organisation key
        [CompanyCodeId]     bigint NOT NULL,                                                                          -- Assigned company code
        [Name]              nvarchar(60) NOT NULL,                                                                    -- Description
        [CurrencyCode]      nvarchar(5) NOT NULL,                                                                     -- Statistics currency
        [AddressId]         bigint NULL,                                                                              -- Address
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_org_SalesOrganization_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_org_SalesOrganization_IsActive] DEFAULT (1),                  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_SalesOrganization] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_SalesOrganization] UNIQUE ([TenantId], [SalesOrganization])
    );
END
GO

/* org.Segment - Segment for segment reporting (IFRS 8) (reference: FAGL_SEGM) */
IF OBJECT_ID(N'org.Segment', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[Segment]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [Segment]            nvarchar(10) NOT NULL,                                                                   -- Segment key
        [Name]               nvarchar(40) NOT NULL,                                                                   -- Description
        [DerivationPriority] int NOT NULL,                                                                            -- Order in which derivation rules are evaluated
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_org_Segment_CreatedAt] DEFAULT (SYSUTCDATETIME()),  -- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_org_Segment_IsActive] DEFAULT (1),                           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_Segment] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_Segment] UNIQUE ([TenantId], [Segment])
    );
END
GO

/* org.Tenant - Client / tenant - the top isolation boundary (reference: T000) */
IF OBJECT_ID(N'org.Tenant', N'U') IS NULL
BEGIN
    CREATE TABLE [org].[Tenant]
    (
        [Id]                      int IDENTITY(1,1) NOT NULL,                                                         -- Tenant id, used as TenantId everywhere else
        [TenantCode]              nvarchar(4) NOT NULL,                                                               -- Short tenant key (SAP client number equivalent)
        [Name]                    nvarchar(60) NOT NULL,                                                              -- Tenant name
        [LogicalSystem]           nvarchar(20) NULL,                                                                  -- Logical system name for integration
        [DefaultLanguage]         nvarchar(2) NOT NULL,                                                               -- Default UI language (EN, KM)
        [DefaultCurrencyCode]     nvarchar(5) NULL,                                                                   -- Default currency proposal
        [TimeZoneId]              nvarchar(64) NOT NULL,                                                              -- IANA time zone used for display
        [IsProduction]            bit NOT NULL CONSTRAINT [DF_org_Tenant_IsProduction] DEFAULT (0),                   -- Production client - blocks unreviewed DDL and test postings
        [AllowCustomizingChanges] bit NOT NULL CONSTRAINT [DF_org_Tenant_AllowCustomizingChanges] DEFAULT (0),        -- Whether configuration may be changed directly
        [ValidFrom]               date NOT NULL,                                                                      -- Tenant activation date
        [ValidTo]                 date NOT NULL,                                                                      -- Tenant expiry date
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_org_Tenant_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_org_Tenant_IsActive] DEFAULT (1),                       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_org_Tenant] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_org_Tenant] UNIQUE ([TenantCode])
    );
END
GO
