/* ============================================================================
   S/4HANA-inspired ERP - schema [rpt]
   Reporting metadata (5 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* rpt.ReportDefinition - Registered report */
IF OBJECT_ID(N'rpt.ReportDefinition', N'U') IS NULL
BEGIN
    CREATE TABLE [rpt].[ReportDefinition]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [ReportCode]           nvarchar(40) NOT NULL,                                                                 -- Report key, e.g. FI_TRIAL_BALANCE
        [Name]                 nvarchar(60) NOT NULL,                                                                 -- Report name
        [Description]          nvarchar(255) NULL,                                                                    -- Description
        [Module]               nvarchar(20) NOT NULL,                                                                 -- Owning module
        [ReportCategory]       nvarchar(30) NOT NULL,                                                                 -- GeneralLedger, Receivables, Payables, Assets, Controlling, Tax, Audit, Dashboard
        [DataSourceView]       nvarchar(64) NOT NULL,                                                                 -- View or read model queried
        [RequiredPermissionId] bigint NULL,                                                                           -- Permission required to run
        [TransactionCode]      nvarchar(20) NULL,                                                                     -- T-code alias
        [SupportsDrillDown]    bit NOT NULL CONSTRAINT [DF_rpt_ReportDefinition_SupportsDrillDown] DEFAULT (0),       -- Drill-down to line items
        [DrillDownTarget]      nvarchar(40) NULL,                                                                     -- Target object of the drill-down
        [DefaultCurrencyType]  nvarchar(2) NULL,                                                                      -- 10 local, 30 group, 00 document
        [MaxRows]              int NOT NULL,                                                                          -- Row cap
        [IsExportAllowed]      bit NOT NULL CONSTRAINT [DF_rpt_ReportDefinition_IsExportAllowed] DEFAULT (0),         -- Export permitted
        [IsScheduleAllowed]    bit NOT NULL CONSTRAINT [DF_rpt_ReportDefinition_IsScheduleAllowed] DEFAULT (0),       -- May be scheduled as a background job
        [IsActive]             bit NOT NULL CONSTRAINT [DF_rpt_ReportDefinition_IsActive] DEFAULT (1),                -- Active
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_rpt_ReportDefinition_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        CONSTRAINT [PK_rpt_ReportDefinition] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_rpt_ReportDefinition] UNIQUE ([TenantId], [ReportCode])
    );
END
GO

/* rpt.ReportExecutionLog - Execution record for auditing and performance analysis */
IF OBJECT_ID(N'rpt.ReportExecutionLog', N'U') IS NULL
BEGIN
    CREATE TABLE [rpt].[ReportExecutionLog]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [ReportDefinitionId]  bigint NOT NULL,                                                                        -- Report
        [UserId]              bigint NOT NULL,                                                                        -- Executing user
        [ExecutedAt]          datetime2(3) NOT NULL,                                                                  -- Execution timestamp (UTC)
        [ParameterValuesJson] nvarchar(max) NULL,                                                                     -- Parameters used
        [RowCount]            int NOT NULL,                                                                           -- Rows returned
        [DurationMs]          int NOT NULL,                                                                           -- Duration in milliseconds
        [OutputFormat]        nvarchar(10) NULL,                                                                      -- Screen, XLSX, PDF, CSV
        [WasExported]         bit NOT NULL CONSTRAINT [DF_rpt_ReportExecutionLog_WasExported] DEFAULT (0),            -- Result exported
        [WasTruncated]        bit NOT NULL CONSTRAINT [DF_rpt_ReportExecutionLog_WasTruncated] DEFAULT (0),           -- Row cap reached
        [Status]              nvarchar(20) NOT NULL,                                                                  -- Success, Failed, Cancelled, Timeout
        [ErrorMessage]        nvarchar(500) NULL,                                                                     -- Error message
        [CorrelationId]       uniqueidentifier NULL,                                                                  -- Request correlation id
        CONSTRAINT [PK_rpt_ReportExecutionLog] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* rpt.ReportLayout - Column layout of a report or list */
IF OBJECT_ID(N'rpt.ReportLayout', N'U') IS NULL
BEGIN
    CREATE TABLE [rpt].[ReportLayout]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [ReportDefinitionId]   bigint NOT NULL,                                                                       -- Report
        [LayoutName]           nvarchar(40) NOT NULL,                                                                 -- Layout name
        [OwnerUserId]          bigint NOT NULL,                                                                       -- Owner
        [ColumnDefinitionJson] nvarchar(max) NOT NULL,                                                                -- Columns, order, width, visibility
        [SortDefinition]       nvarchar(255) NULL,                                                                    -- Sort fields
        [GroupByFields]        nvarchar(255) NULL,                                                                    -- Grouping fields
        [SubtotalFields]       nvarchar(255) NULL,                                                                    -- Subtotal fields
        [IsShared]             bit NOT NULL CONSTRAINT [DF_rpt_ReportLayout_IsShared] DEFAULT (0),                    -- Visible to other users
        [IsDefault]            bit NOT NULL CONSTRAINT [DF_rpt_ReportLayout_IsDefault] DEFAULT (0),                   -- Default layout
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_rpt_ReportLayout_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_rpt_ReportLayout_IsActive] DEFAULT (1),                    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_rpt_ReportLayout] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_rpt_ReportLayout] UNIQUE ([TenantId], [ReportDefinitionId], [LayoutName], [OwnerUserId])
    );
END
GO

/* rpt.ReportParameter - Selection parameter of a report */
IF OBJECT_ID(N'rpt.ReportParameter', N'U') IS NULL
BEGIN
    CREATE TABLE [rpt].[ReportParameter]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [ReportDefinitionId]      bigint NOT NULL,                                                                    -- Report
        [ParameterName]           nvarchar(64) NOT NULL,                                                              -- Parameter name
        [Label]                   nvarchar(60) NOT NULL,                                                              -- Screen label
        [DataElementId]           bigint NULL,                                                                        -- Data element
        [SqlType]                 nvarchar(40) NOT NULL,                                                              -- Physical type
        [IsRequired]              bit NOT NULL CONSTRAINT [DF_rpt_ReportParameter_IsRequired] DEFAULT (0),            -- Mandatory
        [IsRange]                 bit NOT NULL CONSTRAINT [DF_rpt_ReportParameter_IsRange] DEFAULT (0),               -- Range (from / to) parameter
        [IsMultiSelect]           bit NOT NULL CONSTRAINT [DF_rpt_ReportParameter_IsMultiSelect] DEFAULT (0),         -- Multiple values allowed
        [DefaultValue]            nvarchar(255) NULL,                                                                 -- Default
        [SearchHelpId]            bigint NULL,                                                                        -- Search help
        [DisplayOrder]            int NOT NULL,                                                                       -- Order on the selection screen
        [IsAuthorizationRelevant] bit NOT NULL CONSTRAINT [DF_rpt_ReportParameter_IsAuthorizationRelevant] DEFAULT (0),-- Value restricted by the user's authorizations
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_rpt_ReportParameter_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_rpt_ReportParameter_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_rpt_ReportParameter] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_rpt_ReportParameter] UNIQUE ([TenantId], [ReportDefinitionId], [ParameterName])
    );
END
GO

/* rpt.ReportVariant - Saved selection values for a report */
IF OBJECT_ID(N'rpt.ReportVariant', N'U') IS NULL
BEGIN
    CREATE TABLE [rpt].[ReportVariant]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [ReportDefinitionId]  bigint NOT NULL,                                                                        -- Report
        [VariantName]         nvarchar(40) NOT NULL,                                                                  -- Variant name
        [OwnerUserId]         bigint NOT NULL,                                                                        -- Owner
        [Description]         nvarchar(255) NULL,                                                                     -- Description
        [ParameterValuesJson] nvarchar(max) NOT NULL,                                                                 -- Stored parameter values
        [IsShared]            bit NOT NULL CONSTRAINT [DF_rpt_ReportVariant_IsShared] DEFAULT (0),                    -- Visible to other users
        [IsDefault]           bit NOT NULL CONSTRAINT [DF_rpt_ReportVariant_IsDefault] DEFAULT (0),                   -- Loaded by default
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_rpt_ReportVariant_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_rpt_ReportVariant_IsActive] DEFAULT (1),                    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_rpt_ReportVariant] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_rpt_ReportVariant] UNIQUE ([TenantId], [ReportDefinitionId], [VariantName], [OwnerUserId])
    );
END
GO
