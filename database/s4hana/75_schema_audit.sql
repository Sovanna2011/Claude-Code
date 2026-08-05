/* ============================================================================
   S/4HANA-inspired ERP - schema [audit]
   Audit and change documents (5 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* audit.AuditLog - Business audit trail - one row per auditable action */
IF OBJECT_ID(N'audit.AuditLog', N'U') IS NULL
BEGIN
    CREATE TABLE [audit].[AuditLog]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [OccurredAt]        datetime2(3) NOT NULL,                                                                    -- Action timestamp (UTC)
        [UserName]          nvarchar(64) NOT NULL,                                                                    -- Acting user
        [UserId]            bigint NULL,                                                                              -- User
        [CompanyCodeId]     bigint NULL,                                                                              -- Company code context
        [Action]            nvarchar(40) NOT NULL,                                                                    -- Create, Change, Delete, Post, Reverse, Approve, Reject, Export, Login, ConfigChange
        [ObjectType]        nvarchar(40) NOT NULL,                                                                    -- Object type acted on
        [ObjectId]          bigint NULL,                                                                              -- Object key
        [ObjectKeyText]     nvarchar(255) NULL,                                                                       -- Readable key, e.g. document number
        [SourceType]        nvarchar(20) NOT NULL,                                                                    -- Screen, Api, BackgroundJob, Import, System
        [SourceName]        nvarchar(100) NULL,                                                                       -- Screen, endpoint or job name
        [TransactionCode]   nvarchar(20) NULL,                                                                        -- T-code used
        [Result]            nvarchar(20) NOT NULL,                                                                    -- Success, Failure, Denied
        [MessageText]       nvarchar(500) NULL,                                                                       -- Result or error message
        [CorrelationId]     uniqueidentifier NULL,                                                                    -- Request correlation id
        [SessionId]         uniqueidentifier NULL,                                                                    -- Session
        [IpAddress]         nvarchar(45) NULL,                                                                        -- Request origin
        [UserAgent]         nvarchar(255) NULL,                                                                       -- Client user agent
        [IsSensitiveAction] bit NOT NULL CONSTRAINT [DF_audit_AuditLog_IsSensitiveAction] DEFAULT (0),                -- Critical action - extended retention
        [HashChainValue]    nvarchar(64) NOT NULL,                                                                    -- SHA-256 over the row and the previous hash - tamper detection
        [PreviousHashValue] nvarchar(64) NULL,                                                                        -- Hash of the previous audit row
        CONSTRAINT [PK_audit_AuditLog] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* audit.ChangeDocumentHeader - Change document for configuration and master data (reference: CDHDR) */
IF OBJECT_ID(N'audit.ChangeDocumentHeader', N'U') IS NULL
BEGIN
    CREATE TABLE [audit].[ChangeDocumentHeader]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [ChangeDocumentNumber] nvarchar(20) NOT NULL,                                                                 -- Change document number
        [ObjectClass]          nvarchar(40) NOT NULL,                                                                 -- Object class, e.g. BUSINESS_PARTNER, GL_ACCOUNT, COST_CENTER
        [ObjectId]             bigint NOT NULL,                                                                       -- Changed object
        [ObjectKeyText]        nvarchar(255) NOT NULL,                                                                -- Readable object key
        [ChangedAt]            datetime2(3) NOT NULL,                                                                 -- Change timestamp (UTC)
        [ChangedBy]            nvarchar(64) NOT NULL,                                                                 -- Changing user
        [ChangeType]           nvarchar(1) NOT NULL,                                                                  -- I insert, U update, D delete
        [TransactionCode]      nvarchar(20) NULL,                                                                     -- T-code used
        [SourceType]           nvarchar(20) NOT NULL,                                                                 -- Screen, Api, Import, Job
        [CorrelationId]        uniqueidentifier NULL,                                                                 -- Request correlation id
        [PlannedChangeDate]    date NULL,                                                                             -- Effective date for planned changes
        CONSTRAINT [PK_audit_ChangeDocumentHeader] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_audit_ChangeDocumentHeader] UNIQUE ([TenantId], [ChangeDocumentNumber])
    );
END
GO

/* audit.ChangeDocumentItem - Field-level before/after values (reference: CDPOS) */
IF OBJECT_ID(N'audit.ChangeDocumentItem', N'U') IS NULL
BEGIN
    CREATE TABLE [audit].[ChangeDocumentItem]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [ChangeDocumentHeaderId] bigint NOT NULL,                                                                     -- Change document
        [ItemNumber]             int NOT NULL,                                                                        -- Item sequence
        [TableName]              nvarchar(64) NOT NULL,                                                               -- Changed table
        [TableKeyText]           nvarchar(255) NOT NULL,                                                              -- Key of the changed row
        [FieldName]              nvarchar(64) NOT NULL,                                                               -- Changed field
        [ChangeIndicator]        nvarchar(1) NOT NULL,                                                                -- I insert, U update, D delete
        [OldValue]               nvarchar(500) NULL,                                                                  -- Value before the change (masked if sensitive)
        [NewValue]               nvarchar(500) NULL,                                                                  -- Value after the change (masked if sensitive)
        [IsSensitiveField]       bit NOT NULL CONSTRAINT [DF_audit_ChangeDocumentItem_IsSensitiveField] DEFAULT (0),  -- Sensitive field - values masked
        [TextValueChanged]       bit NOT NULL CONSTRAINT [DF_audit_ChangeDocumentItem_TextValueChanged] DEFAULT (0),  -- Long text changed (values not stored inline)
        CONSTRAINT [PK_audit_ChangeDocumentItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_audit_ChangeDocumentItem] UNIQUE ([TenantId], [ChangeDocumentHeaderId], [ItemNumber])
    );
END
GO

/* audit.DataAccessLog - Read access to personal or restricted data */
IF OBJECT_ID(N'audit.DataAccessLog', N'U') IS NULL
BEGIN
    CREATE TABLE [audit].[DataAccessLog]
    (
        [Id]             bigint IDENTITY(1,1) NOT NULL,                                                               -- Surrogate key
        [TenantId]       int NOT NULL,                                                                                -- Owning tenant - every query is filtered by it
        [AccessedAt]     datetime2(3) NOT NULL,                                                                       -- Access timestamp (UTC)
        [UserId]         bigint NOT NULL,                                                                             -- Accessing user
        [ObjectType]     nvarchar(40) NOT NULL,                                                                       -- Object type read
        [ObjectId]       bigint NULL,                                                                                 -- Object key
        [AccessType]     nvarchar(20) NOT NULL,                                                                       -- Display, Search, Report, Export, Api
        [FieldsAccessed] nvarchar(500) NULL,                                                                          -- Sensitive fields returned
        [RecordCount]    int NOT NULL,                                                                                -- Number of records returned
        [Purpose]        nvarchar(255) NULL,                                                                          -- Stated purpose, when required
        [CorrelationId]  uniqueidentifier NULL,                                                                       -- Request correlation id
        [IpAddress]      nvarchar(45) NULL,                                                                           -- Request origin
        CONSTRAINT [PK_audit_DataAccessLog] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* audit.RetentionPolicy - Retention and archiving rule per audit or business object */
IF OBJECT_ID(N'audit.RetentionPolicy', N'U') IS NULL
BEGIN
    CREATE TABLE [audit].[RetentionPolicy]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [ObjectType]        nvarchar(40) NOT NULL,                                                                    -- Object the policy applies to
        [RetentionYears]    int NOT NULL,                                                                             -- Years the data must be kept
        [ArchiveAfterYears] int NULL,                                                                                 -- Years before the data is archived
        [LegalBasis]        nvarchar(255) NULL,                                                                       -- Legal or regulatory basis
        [DeletionMode]      nvarchar(20) NOT NULL,                                                                    -- Block, Anonymize, Delete, ArchiveOnly
        [IsActive]          bit NOT NULL CONSTRAINT [DF_audit_RetentionPolicy_IsActive] DEFAULT (1),                  -- Policy active
        [LastExecutedAt]    datetime2(3) NULL,                                                                        -- Last execution (UTC)
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_audit_RetentionPolicy_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        CONSTRAINT [PK_audit_RetentionPolicy] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_audit_RetentionPolicy] UNIQUE ([TenantId], [ObjectType])
    );
END
GO
