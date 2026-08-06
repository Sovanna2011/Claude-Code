/* ============================================================================
   S/4HANA-inspired ERP - schema [co]
   Controlling (24 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* co.ActivityPrice - Planned or actual price of an activity type per cost centre and period */
IF OBJECT_ID(N'co.ActivityPrice', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[ActivityPrice]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [ControllingAreaId] bigint NOT NULL,                                                                          -- Controlling area
        [CostCenterId]      bigint NOT NULL,                                                                          -- Sending cost centre
        [ActivityTypeId]    bigint NOT NULL,                                                                          -- Activity type
        [FiscalYear]        smallint NOT NULL,                                                                        -- Fiscal year
        [FiscalPeriod]      tinyint NOT NULL,                                                                         -- Period
        [PlanVersion]       nvarchar(3) NOT NULL,                                                                     -- Plan version, e.g. 000
        [FixedPrice]        decimal(19,4) NOT NULL,                                                                   -- Fixed portion of the price
        [VariablePrice]     decimal(19,4) NOT NULL,                                                                   -- Variable portion of the price
        [PriceUnit]         int NOT NULL,                                                                             -- Price unit (price per n units)
        [CurrencyCode]      nvarchar(5) NOT NULL,                                                                     -- Currency
        [PlannedQuantity]   decimal(23,6) NULL,                                                                       -- Planned activity quantity
        [CapacityQuantity]  decimal(23,6) NULL,                                                                       -- Capacity
        [IsActualPrice]     bit NOT NULL CONSTRAINT [DF_co_ActivityPrice_IsActualPrice] DEFAULT (0),                  -- Actual price calculated at period end
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_co_ActivityPrice_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_co_ActivityPrice_IsActive] DEFAULT (1),                       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_ActivityPrice] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_ActivityPrice] UNIQUE ([TenantId], [ControllingAreaId], [CostCenterId], [ActivityTypeId], [FiscalYear], [FiscalPeriod], [PlanVersion])
    );
END
GO

/* co.ActivityType - Activity type (reference: CSLA) */
IF OBJECT_ID(N'co.ActivityType', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[ActivityType]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [ControllingAreaId]       bigint NOT NULL,                                                                    -- Controlling area
        [ActivityType]            nvarchar(6) NOT NULL,                                                               -- Activity type key
        [Name]                    nvarchar(40) NOT NULL,                                                              -- Description
        [UnitOfMeasure]           nvarchar(3) NOT NULL,                                                               -- Activity unit
        [ActivityTypeCategory]    nvarchar(1) NOT NULL,                                                               -- 1 manual entry / manual allocation, 2 indirect determination, 3 manual entry / no allocation
        [AllocationCostElementId] bigint NOT NULL,                                                                    -- Secondary cost element used for allocation
        [PriceIndicator]          nvarchar(1) NOT NULL,                                                               -- 1 plan price automatic, 2 plan price political, 3 manual
        [IsBlocked]               bit NOT NULL CONSTRAINT [DF_co_ActivityType_IsBlocked] DEFAULT (0),                 -- Blocked
        [ValidFrom]               date NOT NULL,                                                                      -- First day the record is valid
        [ValidTo]                 date NOT NULL,                                                                      -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_co_ActivityType_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_co_ActivityType_IsActive] DEFAULT (1),                  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_ActivityType] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_ActivityType] UNIQUE ([TenantId], [ControllingAreaId], [ActivityType])
    );
END
GO

/* co.AllocationCycle - Distribution or assessment cycle (reference: T811C) */
IF OBJECT_ID(N'co.AllocationCycle', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[AllocationCycle]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [ControllingAreaId]    bigint NOT NULL,                                                                       -- Controlling area
        [CycleCode]            nvarchar(6) NOT NULL,                                                                  -- Cycle key
        [StartDate]            date NOT NULL,                                                                         -- Cycle start date
        [Name]                 nvarchar(60) NOT NULL,                                                                 -- Description
        [AllocationType]       nvarchar(20) NOT NULL,                                                                 -- Distribution, Assessment, PeriodicReposting, IndirectActivityAllocation
        [IsPlan]               bit NOT NULL CONSTRAINT [DF_co_AllocationCycle_IsPlan] DEFAULT (0),                    -- Plan allocation
        [PlanVersion]          nvarchar(3) NULL,                                                                      -- Plan version
        [EndDate]              date NOT NULL,                                                                         -- Cycle end date
        [IterationAllowed]     bit NOT NULL CONSTRAINT [DF_co_AllocationCycle_IterationAllowed] DEFAULT (0),          -- Iterative processing allowed
        [CumulativeProcessing] bit NOT NULL CONSTRAINT [DF_co_AllocationCycle_CumulativeProcessing] DEFAULT (0),      -- Cumulative allocation
        [Status]               nvarchar(20) NOT NULL,                                                                 -- Draft, Active, Blocked
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_co_AllocationCycle_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_co_AllocationCycle_IsActive] DEFAULT (1),                  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_AllocationCycle] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_AllocationCycle] UNIQUE ([TenantId], [ControllingAreaId], [CycleCode], [StartDate])
    );
END
GO

/* co.AllocationCycleReceiver - Fixed receiver share of an allocation segment */
IF OBJECT_ID(N'co.AllocationCycleReceiver', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[AllocationCycleReceiver]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [AllocationCycleSegmentId] bigint NOT NULL,                                                                   -- Owning segment
        [ReceiverObjectType]       nvarchar(20) NOT NULL,                                                             -- Receiver type
        [ReceiverObjectId]         bigint NOT NULL,                                                                   -- Receiver object
        [FiscalYear]               smallint NOT NULL,                                                                 -- Fiscal year
        [FiscalPeriod]             tinyint NOT NULL,                                                                  -- Period
        [Percentage]               decimal(9,4) NULL,                                                                 -- Fixed percentage
        [Portion]                  decimal(23,6) NULL,                                                                -- Fixed portion
        [FixedAmount]              decimal(19,4) NULL,                                                                -- Fixed amount
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_co_AllocationCycleReceiver_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_co_AllocationCycleReceiver_IsActive] DEFAULT (1),      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_AllocationCycleReceiver] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_AllocationCycleReceiver] UNIQUE ([TenantId], [AllocationCycleSegmentId], [ReceiverObjectType], [ReceiverObjectId], [FiscalYear], [FiscalPeriod])
    );
END
GO

/* co.AllocationCycleSegment - Segment of an allocation cycle: senders, receivers and the tracing factor (reference: T811S) */
IF OBJECT_ID(N'co.AllocationCycleSegment', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[AllocationCycleSegment]
    (
        [Id]                            bigint IDENTITY(1,1) NOT NULL,                                                -- Surrogate key
        [TenantId]                      int NOT NULL,                                                                 -- Owning tenant - every query is filtered by it
        [AllocationCycleId]             bigint NOT NULL,                                                              -- Owning cycle
        [SegmentCode]                   nvarchar(6) NOT NULL,                                                         -- Segment key
        [Name]                          nvarchar(60) NOT NULL,                                                        -- Description
        [SegmentOrder]                  int NOT NULL,                                                                 -- Processing order
        [SenderObjectType]              nvarchar(20) NOT NULL,                                                        -- Sender object type
        [SenderSelection]               nvarchar(max) NOT NULL,                                                       -- Sender selection (JSON: groups, intervals)
        [SenderCostElementSelection]    nvarchar(max) NULL,                                                           -- Cost elements allocated
        [SenderRule]                    nvarchar(20) NOT NULL,                                                        -- PostedAmounts, FixedAmounts, FixedRates
        [SenderPercent]                 decimal(9,4) NULL,                                                            -- Percentage of the sender to allocate
        [ReceiverObjectType]            nvarchar(20) NOT NULL,                                                        -- Receiver object type
        [ReceiverSelection]             nvarchar(max) NOT NULL,                                                       -- Receiver selection (JSON)
        [ReceiverRule]                  nvarchar(20) NOT NULL,                                                        -- VariablePortions, FixedPercentages, FixedPortions, FixedAmounts
        [TracingFactorType]             nvarchar(20) NULL,                                                            -- StatisticalKeyFigure, PostedCosts, ActivityQuantity, Percentage
        [StatisticalKeyFigureId]        bigint NULL,                                                                  -- Key figure used as the tracing factor
        [AssessmentCostElementId]       bigint NULL,                                                                  -- Assessment cost element
        [ScalingNegativeTracingFactors] nvarchar(20) NULL,                                                            -- Handling of negative tracing factors
        [CreatedAt]                     datetime2(3) NOT NULL CONSTRAINT [DF_co_AllocationCycleSegment_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                     nvarchar(64) NOT NULL,                                                        -- Creating user name
        [ModifiedAt]                    datetime2(3) NULL,                                                            -- Last change timestamp (UTC)
        [ModifiedBy]                    nvarchar(64) NULL,                                                            -- Last changing user name
        [RowVersion]                    rowversion NOT NULL,                                                          -- Optimistic concurrency token
        [IsActive]                      bit NOT NULL CONSTRAINT [DF_co_AllocationCycleSegment_IsActive] DEFAULT (1),  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_AllocationCycleSegment] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_AllocationCycleSegment] UNIQUE ([TenantId], [AllocationCycleId], [SegmentCode])
    );
END
GO

/* co.AllocationRun - Execution of an allocation cycle */
IF OBJECT_ID(N'co.AllocationRun', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[AllocationRun]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [AllocationCycleId]         bigint NOT NULL,                                                                  -- Cycle executed
        [FiscalYear]                smallint NOT NULL,                                                                -- Fiscal year
        [FiscalPeriod]              tinyint NOT NULL,                                                                 -- Period
        [RunSequence]               int NOT NULL,                                                                     -- Sequence when repeated
        [IsTestRun]                 bit NOT NULL CONSTRAINT [DF_co_AllocationRun_IsTestRun] DEFAULT (0),              -- Test run
        [PostingDate]               date NOT NULL,                                                                    -- Posting date
        [Status]                    nvarchar(20) NOT NULL,                                                            -- Running, Completed, Failed, Reversed
        [SenderCount]               int NULL,                                                                         -- Number of senders processed
        [ReceiverCount]             int NULL,                                                                         -- Number of receivers credited
        [TotalAllocatedAmount]      decimal(19,4) NULL,                                                               -- Total allocated
        [CurrencyCode]              nvarchar(5) NULL,                                                                 -- Currency
        [ControllingDocumentNumber] nvarchar(30) NULL,                                                                -- CO document produced
        [ExecutedAt]                datetime2(3) NOT NULL,                                                            -- Execution timestamp (UTC)
        [ExecutedBy]                nvarchar(64) NOT NULL,                                                            -- Executing user or job
        [LogText]                   nvarchar(max) NULL,                                                               -- Run log
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_co_AllocationRun_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_co_AllocationRun_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_AllocationRun] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_AllocationRun] UNIQUE ([TenantId], [AllocationCycleId], [FiscalYear], [FiscalPeriod], [RunSequence])
    );
END
GO

/* co.Commitment - Open commitment from purchase requisitions and orders (reference: COOI) */
IF OBJECT_ID(N'co.Commitment', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[Commitment]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [ControllingAreaId]    bigint NOT NULL,                                                                       -- Controlling area
        [ObjectType]           nvarchar(20) NOT NULL,                                                                 -- CostCenter, InternalOrder
        [ObjectId]             bigint NOT NULL,                                                                       -- Account assignment object
        [SourceDocumentType]   nvarchar(20) NOT NULL,                                                                 -- PurchaseRequisition, PurchaseOrder, Contract
        [SourceDocumentNumber] nvarchar(20) NOT NULL,                                                                 -- Source document
        [SourceDocumentItem]   int NOT NULL,                                                                          -- Source item
        [CostElementId]        bigint NOT NULL,                                                                       -- Cost element
        [FiscalYear]           smallint NOT NULL,                                                                     -- Fiscal year
        [FiscalPeriod]         tinyint NOT NULL,                                                                      -- Period
        [CurrencyCode]         nvarchar(5) NOT NULL,                                                                  -- Currency
        [CommitmentAmount]     decimal(19,4) NOT NULL,                                                                -- Committed amount
        [ReducedAmount]        decimal(19,4) NOT NULL,                                                                -- Amount already reduced by actuals
        [OpenCommitmentAmount] decimal(19,4) NOT NULL,                                                                -- Remaining commitment
        [Status]               nvarchar(20) NOT NULL,                                                                 -- Open, PartiallyReduced, Closed
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_co_Commitment_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_co_Commitment_IsActive] DEFAULT (1),                       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_Commitment] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_Commitment] UNIQUE ([TenantId], [ControllingAreaId], [ObjectType], [ObjectId], [SourceDocumentType], [SourceDocumentNumber], [SourceDocumentItem])
    );
END
GO

/* co.ControllingPosting - CO line item - actual and plan (reference: COEP) */
IF OBJECT_ID(N'co.ControllingPosting', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[ControllingPosting]
    (
        [Id]                              bigint IDENTITY(1,1) NOT NULL,                                              -- Surrogate key
        [TenantId]                        int NOT NULL,                                                               -- Owning tenant - every query is filtered by it
        [ControllingAreaId]               bigint NOT NULL,                                                            -- Controlling area
        [ControllingDocumentNumber]       nvarchar(30) NOT NULL,                                                      -- CO document number - the FI number plus a -CO### suffix, so it needs more room than the FI number itself
        [LineItemNumber]                  int NOT NULL,                                                               -- Line number
        [FiscalYear]                      smallint NOT NULL,                                                          -- Fiscal year
        [FiscalPeriod]                    tinyint NOT NULL,                                                           -- Period
        [PostingDate]                     date NOT NULL,                                                              -- Posting date
        [PlanVersion]                     nvarchar(3) NOT NULL,                                                       -- Version (000 = actual)
        [IsPlan]                          bit NOT NULL CONSTRAINT [DF_co_ControllingPosting_IsPlan] DEFAULT (0),      -- Plan record
        [ValueType]                       nvarchar(2) NOT NULL,                                                       -- 04 actual, 01 plan, 11 statistical actual, 21 commitment
        [ObjectType]                      nvarchar(20) NOT NULL,                                                      -- CostCenter, InternalOrder, ProfitCenter, Asset, Project
        [ObjectId]                        bigint NOT NULL,                                                            -- Account assignment object
        [PartnerObjectType]               nvarchar(20) NULL,                                                          -- Partner object type (sender/receiver)
        [PartnerObjectId]                 bigint NULL,                                                                -- Partner object
        [CostElementId]                   bigint NOT NULL,                                                            -- Cost element
        [ActivityTypeId]                  bigint NULL,                                                                -- Activity type
        [CompanyCodeId]                   bigint NOT NULL,                                                            -- Company code
        [ProfitCenterId]                  bigint NULL,                                                                -- Profit centre
        [FunctionalAreaId]                bigint NULL,                                                                -- Functional area
        [SegmentId]                       bigint NULL,                                                                -- Segment
        [DebitCreditIndicator]            nvarchar(1) NOT NULL,                                                       -- S debit, H credit
        [CurrencyCode]                    nvarchar(5) NOT NULL,                                                       -- Transaction currency
        [AmountInTransactionCurrency]     decimal(19,4) NOT NULL,                                                     -- Amount in transaction currency
        [AmountInControllingAreaCurrency] decimal(19,4) NOT NULL,                                                     -- Amount in controlling area currency
        [AmountInCompanyCodeCurrency]     decimal(19,4) NOT NULL,                                                     -- Amount in company code currency
        [FixedAmount]                     decimal(19,4) NULL,                                                         -- Fixed cost portion
        [Quantity]                        decimal(23,6) NULL,                                                         -- Quantity
        [UnitOfMeasure]                   nvarchar(3) NULL,                                                           -- Unit of measure
        [TransactionType]                 nvarchar(20) NOT NULL,                                                      -- Primary, Distribution, Assessment, ActivityAllocation, Settlement, Reposting, Surcharge
        [ReferenceDocumentNumber]         nvarchar(20) NULL,                                                          -- Source document
        [JournalEntryHeaderId]            bigint NULL,                                                                -- Related FI document
        [AllocationRunId]                 bigint NULL,                                                                -- Allocation run that produced the line
        [SettlementDocumentId]            bigint NULL,                                                                -- Settlement document
        [LineItemText]                    nvarchar(255) NULL,                                                         -- Item text
        [IsReversed]                      bit NOT NULL CONSTRAINT [DF_co_ControllingPosting_IsReversed] DEFAULT (0),  -- Reversed
        [CreatedAt]                       datetime2(3) NOT NULL CONSTRAINT [DF_co_ControllingPosting_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                       nvarchar(64) NOT NULL,                                                      -- Creating user or job
        CONSTRAINT [PK_co_ControllingPosting] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_ControllingPosting] UNIQUE ([TenantId], [ControllingAreaId], [ControllingDocumentNumber], [LineItemNumber])
    );
END
GO

/* co.ControllingTotal - Period totals per object, cost element and version (reference: COSP / COSS) */
IF OBJECT_ID(N'co.ControllingTotal', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[ControllingTotal]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [ControllingAreaId] bigint NOT NULL,                                                                          -- Controlling area
        [ObjectType]        nvarchar(20) NOT NULL,                                                                    -- Object type
        [ObjectId]          bigint NOT NULL,                                                                          -- Object
        [CostElementId]     bigint NOT NULL,                                                                          -- Cost element
        [FiscalYear]        smallint NOT NULL,                                                                        -- Fiscal year
        [FiscalPeriod]      tinyint NOT NULL,                                                                         -- Period
        [PlanVersion]       nvarchar(3) NOT NULL,                                                                     -- Version
        [ValueType]         nvarchar(2) NOT NULL,                                                                     -- Actual / plan / commitment
        [CurrencyCode]      nvarchar(5) NOT NULL,                                                                     -- Currency
        [TotalAmount]       decimal(19,4) NOT NULL,                                                                   -- Total amount
        [FixedAmount]       decimal(19,4) NOT NULL,                                                                   -- Fixed portion
        [TotalQuantity]     decimal(23,6) NULL,                                                                       -- Total quantity
        [LastUpdatedAt]     datetime2(3) NOT NULL,                                                                    -- Last update (UTC)
        [RowVersion]        rowversion NOT NULL,                                                                      -- Concurrency token
        CONSTRAINT [PK_co_ControllingTotal] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_ControllingTotal] UNIQUE ([TenantId], [ControllingAreaId], [ObjectType], [ObjectId], [CostElementId], [FiscalYear], [FiscalPeriod], [PlanVersion], [ValueType])
    );
END
GO

/* co.CostCenter - Cost centre master (reference: CSKS) */
IF OBJECT_ID(N'co.CostCenter', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[CostCenter]
    (
        [Id]                              bigint IDENTITY(1,1) NOT NULL,                                              -- Surrogate key
        [TenantId]                        int NOT NULL,                                                               -- Owning tenant - every query is filtered by it
        [ControllingAreaId]               bigint NOT NULL,                                                            -- Controlling area
        [CostCenter]                      nvarchar(10) NOT NULL,                                                      -- Cost centre number
        [Name]                            nvarchar(40) NOT NULL,                                                      -- Short name
        [Description]                     nvarchar(60) NULL,                                                          -- Description
        [CostCenterCategoryId]            bigint NOT NULL,                                                            -- Category
        [HierarchyNodeId]                 bigint NOT NULL,                                                            -- Node in the standard hierarchy
        [CompanyCodeId]                   bigint NOT NULL,                                                            -- Company code
        [BusinessAreaId]                  bigint NULL,                                                                -- Business area
        [ProfitCenterId]                  bigint NULL,                                                                -- Profit centre
        [FunctionalAreaId]                bigint NULL,                                                                -- Functional area
        [SegmentId]                       bigint NULL,                                                                -- Segment
        [PlantId]                         bigint NULL,                                                                -- Plant
        [DepartmentCode]                  nvarchar(10) NULL,                                                          -- Department
        [ResponsiblePersonPartnerId]      bigint NULL,                                                                -- Person responsible (employee BP)
        [ResponsibleUserName]             nvarchar(64) NULL,                                                          -- Responsible system user
        [CurrencyCode]                    nvarchar(5) NOT NULL,                                                       -- Cost centre currency
        [IsLockedForActualPrimaryCosts]   bit NOT NULL CONSTRAINT [DF_co_CostCenter_IsLockedForActualPrimaryCosts] DEFAULT (0),-- Lock actual primary postings
        [IsLockedForActualSecondaryCosts] bit NOT NULL CONSTRAINT [DF_co_CostCenter_IsLockedForActualSecondaryCosts] DEFAULT (0),-- Lock actual secondary postings
        [IsLockedForActualRevenues]       bit NOT NULL CONSTRAINT [DF_co_CostCenter_IsLockedForActualRevenues] DEFAULT (0),-- Lock actual revenues
        [IsLockedForPlanning]             bit NOT NULL CONSTRAINT [DF_co_CostCenter_IsLockedForPlanning] DEFAULT (0), -- Lock planning
        [IsLockedForCommitments]          bit NOT NULL CONSTRAINT [DF_co_CostCenter_IsLockedForCommitments] DEFAULT (0),-- Lock commitments
        [RecordQuantity]                  bit NOT NULL CONSTRAINT [DF_co_CostCenter_RecordQuantity] DEFAULT (0),      -- Quantities recorded
        [AddressId]                       bigint NULL,                                                                -- Address
        [IsMarkedForDeletion]             bit NOT NULL CONSTRAINT [DF_co_CostCenter_IsMarkedForDeletion] DEFAULT (0), -- Deletion flag
        [ValidFrom]                       date NOT NULL,                                                              -- First day the record is valid
        [ValidTo]                         date NOT NULL,                                                              -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]                       datetime2(3) NOT NULL CONSTRAINT [DF_co_CostCenter_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                       nvarchar(64) NOT NULL,                                                      -- Creating user name
        [ModifiedAt]                      datetime2(3) NULL,                                                          -- Last change timestamp (UTC)
        [ModifiedBy]                      nvarchar(64) NULL,                                                          -- Last changing user name
        [RowVersion]                      rowversion NOT NULL,                                                        -- Optimistic concurrency token
        [IsActive]                        bit NOT NULL CONSTRAINT [DF_co_CostCenter_IsActive] DEFAULT (1),            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_CostCenter] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_CostCenter] UNIQUE ([TenantId], [ControllingAreaId], [CostCenter])
    );
END
GO

/* co.CostCenterCategory - Cost centre category (reference: TKA05) */
IF OBJECT_ID(N'co.CostCenterCategory', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[CostCenterCategory]
    (
        [Id]                           bigint IDENTITY(1,1) NOT NULL,                                                 -- Surrogate key
        [TenantId]                     int NOT NULL,                                                                  -- Owning tenant - every query is filtered by it
        [CategoryCode]                 nvarchar(1) NOT NULL,                                                          -- Category key, e.g. F production, V sales, H service
        [Name]                         nvarchar(40) NOT NULL,                                                         -- Description
        [IsActualPrimaryCostsLocked]   bit NOT NULL CONSTRAINT [DF_co_CostCenterCategory_IsActualPrimaryCostsLocked] DEFAULT (0),-- Lock actual primary costs
        [IsActualSecondaryCostsLocked] bit NOT NULL CONSTRAINT [DF_co_CostCenterCategory_IsActualSecondaryCostsLocked] DEFAULT (0),-- Lock actual secondary costs
        [IsActualRevenuesLocked]       bit NOT NULL CONSTRAINT [DF_co_CostCenterCategory_IsActualRevenuesLocked] DEFAULT (0),-- Lock actual revenues
        [IsCommitmentUpdateLocked]     bit NOT NULL CONSTRAINT [DF_co_CostCenterCategory_IsCommitmentUpdateLocked] DEFAULT (0),-- Lock commitment update
        [IsPlanPrimaryCostsLocked]     bit NOT NULL CONSTRAINT [DF_co_CostCenterCategory_IsPlanPrimaryCostsLocked] DEFAULT (0),-- Lock plan primary costs
        [CreatedAt]                    datetime2(3) NOT NULL CONSTRAINT [DF_co_CostCenterCategory_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                    nvarchar(64) NOT NULL,                                                         -- Creating user name
        [ModifiedAt]                   datetime2(3) NULL,                                                             -- Last change timestamp (UTC)
        [ModifiedBy]                   nvarchar(64) NULL,                                                             -- Last changing user name
        [RowVersion]                   rowversion NOT NULL,                                                           -- Optimistic concurrency token
        [IsActive]                     bit NOT NULL CONSTRAINT [DF_co_CostCenterCategory_IsActive] DEFAULT (1),       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_CostCenterCategory] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_CostCenterCategory] UNIQUE ([TenantId], [CategoryCode])
    );
END
GO

/* co.CostElement - Cost element (primary = G/L expense account, secondary = internal) (reference: CSKB) */
IF OBJECT_ID(N'co.CostElement', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[CostElement]
    (
        [Id]                           bigint IDENTITY(1,1) NOT NULL,                                                 -- Surrogate key
        [TenantId]                     int NOT NULL,                                                                  -- Owning tenant - every query is filtered by it
        [ControllingAreaId]            bigint NOT NULL,                                                               -- Controlling area
        [CostElement]                  nvarchar(10) NOT NULL,                                                         -- Cost element number
        [CostElementCategory]          nvarchar(2) NOT NULL,                                                          -- 1 primary, 11 revenue, 12 sales deduction, 21 internal settlement, 41 overhead, 42 assessment, 43 activity allocation
        [IsPrimary]                    bit NOT NULL CONSTRAINT [DF_co_CostElement_IsPrimary] DEFAULT (0),             -- Primary cost element (has a G/L account)
        [GLAccountId]                  bigint NULL,                                                                   -- G/L account for primary cost elements
        [Name]                         nvarchar(40) NOT NULL,                                                         -- Description
        [Description]                  nvarchar(60) NULL,                                                             -- Long description
        [FunctionalAreaId]             bigint NULL,                                                                   -- Default functional area
        [IsRecordQuantity]             bit NOT NULL CONSTRAINT [DF_co_CostElement_IsRecordQuantity] DEFAULT (0),      -- Quantity recorded with the posting
        [UnitOfMeasure]                nvarchar(3) NULL,                                                              -- Unit for the quantity
        [DefaultAccountAssignmentType] nvarchar(20) NULL,                                                             -- CostCenter, InternalOrder, None
        [DefaultCostCenterId]          bigint NULL,                                                                   -- Default cost centre
        [DefaultInternalOrderId]       bigint NULL,                                                                   -- Default internal order
        [ValidFrom]                    date NOT NULL,                                                                 -- First day the record is valid
        [ValidTo]                      date NOT NULL,                                                                 -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]                    datetime2(3) NOT NULL CONSTRAINT [DF_co_CostElement_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                    nvarchar(64) NOT NULL,                                                         -- Creating user name
        [ModifiedAt]                   datetime2(3) NULL,                                                             -- Last change timestamp (UTC)
        [ModifiedBy]                   nvarchar(64) NULL,                                                             -- Last changing user name
        [RowVersion]                   rowversion NOT NULL,                                                           -- Optimistic concurrency token
        [IsActive]                     bit NOT NULL CONSTRAINT [DF_co_CostElement_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_CostElement] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_CostElement] UNIQUE ([TenantId], [ControllingAreaId], [CostElement])
    );
END
GO

/* co.HierarchyNode - Node of a standard cost-centre or profit-centre hierarchy (reference: SETNODE) */
IF OBJECT_ID(N'co.HierarchyNode', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[HierarchyNode]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [ControllingAreaId]   bigint NOT NULL,                                                                        -- Controlling area
        [HierarchyType]       nvarchar(20) NOT NULL,                                                                  -- CostCenter, ProfitCenter, CostElement, InternalOrder
        [HierarchyId]         nvarchar(12) NOT NULL,                                                                  -- Hierarchy id
        [NodeCode]            nvarchar(20) NOT NULL,                                                                  -- Node key
        [ParentNodeId]        bigint NULL,                                                                            -- Parent node
        [NodeName]            nvarchar(60) NOT NULL,                                                                  -- Node description
        [NodeLevel]           int NOT NULL,                                                                           -- Depth in the hierarchy
        [DisplayOrder]        int NOT NULL,                                                                           -- Order among siblings
        [HierarchyPath]       nvarchar(500) NOT NULL,                                                                 -- Materialised path for fast roll-ups
        [IsStandardHierarchy] bit NOT NULL CONSTRAINT [DF_co_HierarchyNode_IsStandardHierarchy] DEFAULT (0),          -- Node of the standard hierarchy
        [IsLeaf]              bit NOT NULL CONSTRAINT [DF_co_HierarchyNode_IsLeaf] DEFAULT (0),                       -- Objects may be assigned to this node
        [ValidFrom]           date NOT NULL,                                                                          -- First day the record is valid
        [ValidTo]             date NOT NULL,                                                                          -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_co_HierarchyNode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_co_HierarchyNode_IsActive] DEFAULT (1),                     -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_HierarchyNode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_HierarchyNode] UNIQUE ([TenantId], [ControllingAreaId], [HierarchyType], [HierarchyId], [NodeCode])
    );
END
GO

/* co.InternalOrder - Internal order master (reference: AUFK) */
IF OBJECT_ID(N'co.InternalOrder', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[InternalOrder]
    (
        [Id]                         bigint IDENTITY(1,1) NOT NULL,                                                   -- Surrogate key
        [TenantId]                   int NOT NULL,                                                                    -- Owning tenant - every query is filtered by it
        [ControllingAreaId]          bigint NOT NULL,                                                                 -- Controlling area
        [OrderNumber]                nvarchar(12) NOT NULL,                                                           -- Internal order number (not an internal counter)
        [OrderTypeId]                bigint NOT NULL,                                                                 -- Order type
        [Description]                nvarchar(60) NOT NULL,                                                           -- Order description
        [LongText]                   nvarchar(max) NULL,                                                              -- Long text
        [CompanyCodeId]              bigint NOT NULL,                                                                 -- Company code
        [BusinessAreaId]             bigint NULL,                                                                     -- Business area
        [PlantId]                    bigint NULL,                                                                     -- Plant
        [ResponsibleCostCenterId]    bigint NOT NULL,                                                                 -- Responsible cost centre
        [RequestingCostCenterId]     bigint NULL,                                                                     -- Requesting cost centre
        [ProfitCenterId]             bigint NULL,                                                                     -- Profit centre
        [SegmentId]                  bigint NULL,                                                                     -- Segment
        [FunctionalAreaId]           bigint NULL,                                                                     -- Functional area
        [ResponsiblePersonPartnerId] bigint NULL,                                                                     -- Person responsible
        [ApplicantName]              nvarchar(60) NULL,                                                               -- Applicant
        [CurrencyCode]               nvarchar(5) NOT NULL,                                                            -- Order currency
        [IsStatistical]              bit NOT NULL CONSTRAINT [DF_co_InternalOrder_IsStatistical] DEFAULT (0),         -- Statistical order - no settlement
        [IsRevenueBearing]           bit NOT NULL CONSTRAINT [DF_co_InternalOrder_IsRevenueBearing] DEFAULT (0),      -- Revenue postings allowed
        [SystemStatus]               nvarchar(20) NOT NULL,                                                           -- Created, Released, TechnicallyCompleted, Closed, Locked, MarkedForDeletion
        [UserStatus]                 nvarchar(20) NULL,                                                               -- User status
        [WorkStartDate]              date NULL,                                                                       -- Planned start
        [WorkEndDate]                date NULL,                                                                       -- Planned finish
        [ActualStartDate]            date NULL,                                                                       -- Actual start
        [ActualEndDate]              date NULL,                                                                       -- Actual finish
        [EstimatedTotalCost]         decimal(19,4) NULL,                                                              -- Estimated cost
        [InvestmentReason]           nvarchar(2) NULL,                                                                -- Investment reason
        [AssetId]                    bigint NULL,                                                                     -- Asset under construction linked to the order
        [IsBudgetControlActive]      bit NOT NULL CONSTRAINT [DF_co_InternalOrder_IsBudgetControlActive] DEFAULT (0), -- Availability control active
        [IsMarkedForDeletion]        bit NOT NULL CONSTRAINT [DF_co_InternalOrder_IsMarkedForDeletion] DEFAULT (0),   -- Deletion flag
        [CreatedAt]                  datetime2(3) NOT NULL CONSTRAINT [DF_co_InternalOrder_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                  nvarchar(64) NOT NULL,                                                           -- Creating user name
        [ModifiedAt]                 datetime2(3) NULL,                                                               -- Last change timestamp (UTC)
        [ModifiedBy]                 nvarchar(64) NULL,                                                               -- Last changing user name
        [RowVersion]                 rowversion NOT NULL,                                                             -- Optimistic concurrency token
        [IsActive]                   bit NOT NULL CONSTRAINT [DF_co_InternalOrder_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_InternalOrder] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_InternalOrder] UNIQUE ([TenantId], [ControllingAreaId], [OrderNumber])
    );
END
GO

/* co.InternalOrderBudget - Annual and overall budget with availability control (reference: BPJA) */
IF OBJECT_ID(N'co.InternalOrderBudget', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[InternalOrderBudget]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [InternalOrderId]         bigint NOT NULL,                                                                    -- Internal order
        [FiscalYear]              smallint NOT NULL,                                                                  -- Fiscal year (0 = overall budget)
        [BudgetVersion]           nvarchar(3) NOT NULL,                                                               -- Budget version
        [CurrencyCode]            nvarchar(5) NOT NULL,                                                               -- Currency
        [OriginalBudget]          decimal(19,4) NOT NULL,                                                             -- Original budget
        [SupplementAmount]        decimal(19,4) NOT NULL,                                                             -- Supplements
        [ReturnAmount]            decimal(19,4) NOT NULL,                                                             -- Returns
        [CurrentBudget]           decimal(19,4) NOT NULL,                                                             -- Current budget
        [AssignedAmount]          decimal(19,4) NOT NULL,                                                             -- Actuals + commitments
        [AvailableAmount]         decimal(19,4) NOT NULL,                                                             -- Remaining budget
        [UsagePercent]            decimal(9,4) NOT NULL,                                                              -- Utilisation
        [WarningThresholdPercent] decimal(9,4) NULL,                                                                  -- Threshold for a warning
        [ErrorThresholdPercent]   decimal(9,4) NULL,                                                                  -- Threshold that blocks postings
        [IsBudgetExceeded]        bit NOT NULL CONSTRAINT [DF_co_InternalOrderBudget_IsBudgetExceeded] DEFAULT (0),   -- Budget exceeded
        [ReleasedAmount]          decimal(19,4) NULL,                                                                 -- Budget released for use
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_co_InternalOrderBudget_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_co_InternalOrderBudget_IsActive] DEFAULT (1),           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_InternalOrderBudget] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_InternalOrderBudget] UNIQUE ([TenantId], [InternalOrderId], [FiscalYear], [BudgetVersion])
    );
END
GO

/* co.InternalOrderType - Internal order type (reference: T003O) */
IF OBJECT_ID(N'co.InternalOrderType', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[InternalOrderType]
    (
        [Id]                           bigint IDENTITY(1,1) NOT NULL,                                                 -- Surrogate key
        [TenantId]                     int NOT NULL,                                                                  -- Owning tenant - every query is filtered by it
        [OrderType]                    nvarchar(4) NOT NULL,                                                          -- Order type key
        [Name]                         nvarchar(40) NOT NULL,                                                         -- Description
        [OrderCategory]                nvarchar(20) NOT NULL,                                                         -- Overhead, Investment, Accrual, RevenueBearing, Production
        [NumberRangeObjectId]          bigint NOT NULL,                                                               -- Number range object
        [NumberRangeCode]              nvarchar(2) NOT NULL,                                                          -- Number range interval
        [IsStatisticalOnly]            bit NOT NULL CONSTRAINT [DF_co_InternalOrderType_IsStatisticalOnly] DEFAULT (0),-- Statistical orders only
        [IsRevenuePostingAllowed]      bit NOT NULL CONSTRAINT [DF_co_InternalOrderType_IsRevenuePostingAllowed] DEFAULT (0),-- Revenue postings allowed
        [IsCommitmentManagementActive] bit NOT NULL CONSTRAINT [DF_co_InternalOrderType_IsCommitmentManagementActive] DEFAULT (0),-- Commitment management active
        [IsBudgetControlActive]        bit NOT NULL CONSTRAINT [DF_co_InternalOrderType_IsBudgetControlActive] DEFAULT (0),-- Availability control active
        [SettlementProfile]            nvarchar(6) NULL,                                                              -- Settlement profile
        [PlanningProfile]              nvarchar(6) NULL,                                                              -- Planning profile
        [BudgetProfile]                nvarchar(6) NULL,                                                              -- Budget profile
        [StatusProfile]                nvarchar(8) NULL,                                                              -- Status profile
        [IsMasterDataFieldsRequired]   bit NOT NULL CONSTRAINT [DF_co_InternalOrderType_IsMasterDataFieldsRequired] DEFAULT (0),-- Responsible cost centre mandatory
        [CreatedAt]                    datetime2(3) NOT NULL CONSTRAINT [DF_co_InternalOrderType_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                    nvarchar(64) NOT NULL,                                                         -- Creating user name
        [ModifiedAt]                   datetime2(3) NULL,                                                             -- Last change timestamp (UTC)
        [ModifiedBy]                   nvarchar(64) NULL,                                                             -- Last changing user name
        [RowVersion]                   rowversion NOT NULL,                                                           -- Optimistic concurrency token
        [IsActive]                     bit NOT NULL CONSTRAINT [DF_co_InternalOrderType_IsActive] DEFAULT (1),        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_InternalOrderType] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_InternalOrderType] UNIQUE ([TenantId], [OrderType])
    );
END
GO

/* co.PlanEntry - Cost centre / order / profit centre planning line */
IF OBJECT_ID(N'co.PlanEntry', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[PlanEntry]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [ControllingAreaId]     bigint NOT NULL,                                                                      -- Controlling area
        [PlanVersion]           nvarchar(3) NOT NULL,                                                                 -- Plan version
        [ObjectType]            nvarchar(20) NOT NULL,                                                                -- CostCenter, InternalOrder, ProfitCenter
        [ObjectId]              bigint NOT NULL,                                                                      -- Planned object
        [CostElementId]         bigint NOT NULL,                                                                      -- Cost element
        [FiscalYear]            smallint NOT NULL,                                                                    -- Fiscal year
        [FiscalPeriod]          tinyint NOT NULL,                                                                     -- Period
        [ActivityTypeId]        bigint NULL,                                                                          -- Activity type
        [CurrencyCode]          nvarchar(5) NOT NULL,                                                                 -- Currency
        [PlannedFixedAmount]    decimal(19,4) NOT NULL,                                                               -- Planned fixed costs
        [PlannedVariableAmount] decimal(19,4) NOT NULL,                                                               -- Planned variable costs
        [PlannedTotalAmount]    decimal(19,4) NOT NULL,                                                               -- Total planned amount
        [PlannedQuantity]       decimal(23,6) NULL,                                                                   -- Planned quantity
        [UnitOfMeasure]         nvarchar(3) NULL,                                                                     -- Unit
        [PlanningMethod]        nvarchar(20) NOT NULL,                                                                -- Manual, CopyFromActual, CopyFromPlan, Distribution, Formula
        [IsLocked]              bit NOT NULL CONSTRAINT [DF_co_PlanEntry_IsLocked] DEFAULT (0),                       -- Plan line locked
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_co_PlanEntry_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_co_PlanEntry_IsActive] DEFAULT (1),                       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_PlanEntry] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_PlanEntry] UNIQUE ([TenantId], [ControllingAreaId], [PlanVersion], [ObjectType], [ObjectId], [CostElementId], [FiscalYear], [FiscalPeriod], [ActivityTypeId])
    );
END
GO

/* co.ProfitCenter - Profit centre master (reference: CEPC) */
IF OBJECT_ID(N'co.ProfitCenter', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[ProfitCenter]
    (
        [Id]                         bigint IDENTITY(1,1) NOT NULL,                                                   -- Surrogate key
        [TenantId]                   int NOT NULL,                                                                    -- Owning tenant - every query is filtered by it
        [ControllingAreaId]          bigint NOT NULL,                                                                 -- Controlling area
        [ProfitCenter]               nvarchar(10) NOT NULL,                                                           -- Profit centre number
        [Name]                       nvarchar(40) NOT NULL,                                                           -- Short name
        [Description]                nvarchar(60) NULL,                                                               -- Description
        [HierarchyNodeId]            bigint NOT NULL,                                                                 -- Node in the standard hierarchy
        [CompanyCodeId]              bigint NULL,                                                                     -- Company code (blank = cross-company)
        [SegmentId]                  bigint NULL,                                                                     -- Segment derived from this profit centre
        [BusinessAreaId]             bigint NULL,                                                                     -- Business area
        [ResponsiblePersonPartnerId] bigint NULL,                                                                     -- Person responsible
        [ResponsibleUserName]        nvarchar(64) NULL,                                                               -- Responsible system user
        [DepartmentCode]             nvarchar(10) NULL,                                                               -- Department
        [IsLockedForPosting]         bit NOT NULL CONSTRAINT [DF_co_ProfitCenter_IsLockedForPosting] DEFAULT (0),     -- Locked for postings
        [IsLockedForPlanning]        bit NOT NULL CONSTRAINT [DF_co_ProfitCenter_IsLockedForPlanning] DEFAULT (0),    -- Locked for planning
        [AddressId]                  bigint NULL,                                                                     -- Address
        [IsMarkedForDeletion]        bit NOT NULL CONSTRAINT [DF_co_ProfitCenter_IsMarkedForDeletion] DEFAULT (0),    -- Deletion flag
        [ValidFrom]                  date NOT NULL,                                                                   -- First day the record is valid
        [ValidTo]                    date NOT NULL,                                                                   -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]                  datetime2(3) NOT NULL CONSTRAINT [DF_co_ProfitCenter_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                  nvarchar(64) NOT NULL,                                                           -- Creating user name
        [ModifiedAt]                 datetime2(3) NULL,                                                               -- Last change timestamp (UTC)
        [ModifiedBy]                 nvarchar(64) NULL,                                                               -- Last changing user name
        [RowVersion]                 rowversion NOT NULL,                                                             -- Optimistic concurrency token
        [IsActive]                   bit NOT NULL CONSTRAINT [DF_co_ProfitCenter_IsActive] DEFAULT (1),               -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_ProfitCenter] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_ProfitCenter] UNIQUE ([TenantId], [ControllingAreaId], [ProfitCenter])
    );
END
GO

/* co.ProfitCenterAssignment - Assignment of an object to a profit centre with validity */
IF OBJECT_ID(N'co.ProfitCenterAssignment', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[ProfitCenterAssignment]
    (
        [Id]             bigint IDENTITY(1,1) NOT NULL,                                                               -- Surrogate key
        [TenantId]       int NOT NULL,                                                                                -- Owning tenant - every query is filtered by it
        [ObjectType]     nvarchar(20) NOT NULL,                                                                       -- CostCenter, InternalOrder, Asset, GLAccount, Plant
        [ObjectId]       bigint NOT NULL,                                                                             -- Assigned object
        [ProfitCenterId] bigint NOT NULL,                                                                             -- Profit centre
        [CompanyCodeId]  bigint NULL,                                                                                 -- Company code the assignment applies to
        [ValidFrom]      date NOT NULL,                                                                               -- First day the record is valid
        [ValidTo]        date NOT NULL,                                                                               -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]      datetime2(3) NOT NULL CONSTRAINT [DF_co_ProfitCenterAssignment_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]      nvarchar(64) NOT NULL,                                                                       -- Creating user name
        [ModifiedAt]     datetime2(3) NULL,                                                                           -- Last change timestamp (UTC)
        [ModifiedBy]     nvarchar(64) NULL,                                                                           -- Last changing user name
        [RowVersion]     rowversion NOT NULL,                                                                         -- Optimistic concurrency token
        [IsActive]       bit NOT NULL CONSTRAINT [DF_co_ProfitCenterAssignment_IsActive] DEFAULT (1),                 -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_ProfitCenterAssignment] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_ProfitCenterAssignment] UNIQUE ([TenantId], [ObjectType], [ObjectId], [ProfitCenterId])
    );
END
GO

/* co.SettlementDocument - Executed settlement (reference: settlement documents from KO88) */
IF OBJECT_ID(N'co.SettlementDocument', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[SettlementDocument]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [ControllingAreaId]        bigint NOT NULL,                                                                   -- Controlling area
        [SettlementDocumentNumber] nvarchar(20) NOT NULL,                                                             -- Settlement document number
        [SenderObjectType]         nvarchar(20) NOT NULL,                                                             -- Sender object type
        [SenderObjectId]           bigint NOT NULL,                                                                   -- Sender object
        [FiscalYear]               smallint NOT NULL,                                                                 -- Fiscal year
        [FiscalPeriod]             tinyint NOT NULL,                                                                  -- Settlement period
        [PostingDate]              date NOT NULL,                                                                     -- Posting date
        [SettlementType]           nvarchar(3) NOT NULL,                                                              -- PER, FUL
        [CurrencyCode]             nvarchar(5) NOT NULL,                                                              -- Currency
        [TotalSettledAmount]       decimal(19,4) NOT NULL,                                                            -- Total settled
        [IsTestRun]                bit NOT NULL CONSTRAINT [DF_co_SettlementDocument_IsTestRun] DEFAULT (0),          -- Test run
        [IsReversed]               bit NOT NULL CONSTRAINT [DF_co_SettlementDocument_IsReversed] DEFAULT (0),         -- Reversed
        [JournalEntryHeaderId]     bigint NULL,                                                                       -- Accounting document created
        [Status]                   nvarchar(20) NOT NULL,                                                             -- Simulated, Posted, Reversed, Failed
        [ExecutedAt]               datetime2(3) NOT NULL,                                                             -- Execution timestamp (UTC)
        [ExecutedBy]               nvarchar(64) NOT NULL,                                                             -- Executing user or job
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_co_SettlementDocument_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_co_SettlementDocument_IsActive] DEFAULT (1),           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_SettlementDocument] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_SettlementDocument] UNIQUE ([TenantId], [ControllingAreaId], [SettlementDocumentNumber])
    );
END
GO

/* co.SettlementDocumentItem - Amount settled to one receiver */
IF OBJECT_ID(N'co.SettlementDocumentItem', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[SettlementDocumentItem]
    (
        [Id]                           bigint IDENTITY(1,1) NOT NULL,                                                 -- Surrogate key
        [TenantId]                     int NOT NULL,                                                                  -- Owning tenant - every query is filtered by it
        [SettlementDocumentId]         bigint NOT NULL,                                                               -- Settlement document
        [ItemNumber]                   int NOT NULL,                                                                  -- Item number
        [SettlementRuleId]             bigint NULL,                                                                   -- Rule applied
        [ReceiverType]                 nvarchar(20) NOT NULL,                                                         -- Receiver type
        [ReceiverObjectId]             bigint NOT NULL,                                                               -- Receiver key
        [CostElementId]                bigint NOT NULL,                                                               -- Settlement cost element
        [SettledAmount]                decimal(19,4) NOT NULL,                                                        -- Amount settled
        [SettledAmountInLocalCurrency] decimal(19,4) NOT NULL,                                                        -- Amount in local currency
        [Quantity]                     decimal(23,6) NULL,                                                            -- Quantity settled
        [CreatedAt]                    datetime2(3) NOT NULL CONSTRAINT [DF_co_SettlementDocumentItem_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                    nvarchar(64) NOT NULL,                                                         -- Creating user name
        [ModifiedAt]                   datetime2(3) NULL,                                                             -- Last change timestamp (UTC)
        [ModifiedBy]                   nvarchar(64) NULL,                                                             -- Last changing user name
        [RowVersion]                   rowversion NOT NULL,                                                           -- Optimistic concurrency token
        [IsActive]                     bit NOT NULL CONSTRAINT [DF_co_SettlementDocumentItem_IsActive] DEFAULT (1),   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_SettlementDocumentItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_SettlementDocumentItem] UNIQUE ([TenantId], [SettlementDocumentId], [ItemNumber])
    );
END
GO

/* co.SettlementRule - Settlement rule of a sender object (reference: COBRB) */
IF OBJECT_ID(N'co.SettlementRule', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[SettlementRule]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [SenderObjectType]        nvarchar(20) NOT NULL,                                                              -- InternalOrder, AssetUnderConstruction, Project
        [SenderObjectId]          bigint NOT NULL,                                                                    -- Sender object
        [RuleNumber]              int NOT NULL,                                                                       -- Distribution rule number
        [ReceiverType]            nvarchar(20) NOT NULL,                                                              -- CostCenter, GLAccount, Asset, InternalOrder, ProfitCenter, Project
        [ReceiverObjectId]        bigint NOT NULL,                                                                    -- Receiver key
        [SettlementType]          nvarchar(3) NOT NULL,                                                               -- PER periodic, FUL full settlement
        [Percentage]              decimal(9,4) NULL,                                                                  -- Share in percent
        [EquivalenceNumber]       int NULL,                                                                           -- Equivalence number for proportional split
        [AmountLimit]             decimal(19,4) NULL,                                                                 -- Maximum amount to settle
        [SettlementCostElementId] bigint NULL,                                                                        -- Settlement cost element
        [ValidFromPeriod]         tinyint NULL,                                                                       -- First period the rule applies
        [ValidFromYear]           smallint NULL,                                                                      -- First year the rule applies
        [ValidToPeriod]           tinyint NULL,                                                                       -- Last period
        [ValidToYear]             smallint NULL,                                                                      -- Last year
        [IsBlocked]               bit NOT NULL CONSTRAINT [DF_co_SettlementRule_IsBlocked] DEFAULT (0),               -- Rule blocked
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_co_SettlementRule_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_co_SettlementRule_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_SettlementRule] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_SettlementRule] UNIQUE ([TenantId], [SenderObjectType], [SenderObjectId], [RuleNumber])
    );
END
GO

/* co.StatisticalKeyFigure - Statistical key figure (reference: TKA03) */
IF OBJECT_ID(N'co.StatisticalKeyFigure', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[StatisticalKeyFigure]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [ControllingAreaId]    bigint NOT NULL,                                                                       -- Controlling area
        [StatisticalKeyFigure] nvarchar(6) NOT NULL,                                                                  -- Key figure code, e.g. HEADCNT, AREA
        [Name]                 nvarchar(40) NOT NULL,                                                                 -- Description
        [UnitOfMeasure]        nvarchar(3) NOT NULL,                                                                  -- Unit
        [KeyFigureCategory]    nvarchar(1) NOT NULL,                                                                  -- 1 fixed value, 2 total value
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_co_StatisticalKeyFigure_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_co_StatisticalKeyFigure_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_StatisticalKeyFigure] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_StatisticalKeyFigure] UNIQUE ([TenantId], [ControllingAreaId], [StatisticalKeyFigure])
    );
END
GO

/* co.StatisticalKeyFigureValue - Posted statistical key figure value */
IF OBJECT_ID(N'co.StatisticalKeyFigureValue', N'U') IS NULL
BEGIN
    CREATE TABLE [co].[StatisticalKeyFigureValue]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [StatisticalKeyFigureId] bigint NOT NULL,                                                                     -- Key figure
        [ObjectType]             nvarchar(20) NOT NULL,                                                               -- CostCenter, InternalOrder, ProfitCenter
        [ObjectId]               bigint NOT NULL,                                                                     -- Receiving object
        [FiscalYear]             smallint NOT NULL,                                                                   -- Fiscal year
        [FiscalPeriod]           tinyint NOT NULL,                                                                    -- Period
        [PlanVersion]            nvarchar(3) NOT NULL,                                                                -- Plan version (000 = actual)
        [IsPlan]                 bit NOT NULL CONSTRAINT [DF_co_StatisticalKeyFigureValue_IsPlan] DEFAULT (0),        -- Plan or actual value
        [Quantity]               decimal(23,6) NOT NULL,                                                              -- Value
        [UnitOfMeasure]          nvarchar(3) NOT NULL,                                                                -- Unit
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_co_StatisticalKeyFigureValue_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_co_StatisticalKeyFigureValue_IsActive] DEFAULT (1),      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_co_StatisticalKeyFigureValue] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_co_StatisticalKeyFigureValue] UNIQUE ([TenantId], [StatisticalKeyFigureId], [ObjectType], [ObjectId], [FiscalYear], [FiscalPeriod], [PlanVersion], [IsPlan])
    );
END
GO
