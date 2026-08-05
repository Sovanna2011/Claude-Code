/* ============================================================================
   S/4HANA-inspired ERP - schema [intg]
   Integration and API management (11 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* intg.ApiClient - Registered API consumer */
IF OBJECT_ID(N'intg.ApiClient', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[ApiClient]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [ClientCode]          nvarchar(40) NOT NULL,                                                                  -- Client key
        [Name]                nvarchar(60) NOT NULL,                                                                  -- Client name
        [Description]         nvarchar(255) NULL,                                                                     -- Purpose of the integration
        [ClientType]          nvarchar(20) NOT NULL,                                                                  -- Machine, Partner, Internal, Mobile
        [ServiceUserId]       bigint NOT NULL,                                                                        -- Service user the client acts as
        [ApiKeyHash]          nvarchar(255) NULL,                                                                     -- Hash of the API key (never stored in clear)
        [ApiKeyLastRotatedAt] datetime2(3) NULL,                                                                      -- Last key rotation (UTC)
        [OAuthClientId]       nvarchar(100) NULL,                                                                     -- OAuth / OIDC client id
        [AllowedScopes]       nvarchar(500) NULL,                                                                     -- Granted scopes
        [AllowedIpRanges]     nvarchar(500) NULL,                                                                     -- Permitted source IP ranges
        [RateLimitPerMinute]  int NOT NULL,                                                                           -- Requests allowed per minute
        [IsActive]            bit NOT NULL CONSTRAINT [DF_intg_ApiClient_IsActive] DEFAULT (1),                       -- Client active
        [ContactEmail]        nvarchar(255) NULL,                                                                     -- Technical contact
        [ValidFrom]           date NOT NULL,                                                                          -- First day the record is valid
        [ValidTo]             date NOT NULL,                                                                          -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_intg_ApiClient_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        CONSTRAINT [PK_intg_ApiClient] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_ApiClient] UNIQUE ([TenantId], [ClientCode])
    );
END
GO

/* intg.BankStatement - Imported electronic bank statement header (reference: FEBKO) */
IF OBJECT_ID(N'intg.BankStatement', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[BankStatement]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [CompanyCodeId]      bigint NOT NULL,                                                                         -- Company code
        [StatementNumber]    nvarchar(20) NOT NULL,                                                                   -- Statement number
        [StatementDate]      date NOT NULL,                                                                           -- Statement date
        [HouseBankId]        bigint NOT NULL,                                                                         -- House bank
        [HouseBankAccountId] bigint NOT NULL,                                                                         -- Bank account
        [CurrencyCode]       nvarchar(5) NOT NULL,                                                                    -- Statement currency
        [OpeningBalance]     decimal(19,4) NOT NULL,                                                                  -- Opening balance
        [ClosingBalance]     decimal(19,4) NOT NULL,                                                                  -- Closing balance
        [TotalDebitAmount]   decimal(19,4) NOT NULL,                                                                  -- Total debits
        [TotalCreditAmount]  decimal(19,4) NOT NULL,                                                                  -- Total credits
        [ItemCount]          int NOT NULL,                                                                            -- Number of items
        [FileFormat]         nvarchar(20) NOT NULL,                                                                   -- CAMT053, MT940, CSV
        [ImportJobId]        bigint NULL,                                                                             -- Import job
        [Status]             nvarchar(20) NOT NULL,                                                                   -- Imported, PartiallyPosted, Posted, Failed
        [PostedAt]           datetime2(3) NULL,                                                                       -- Posting timestamp (UTC)
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_intg_BankStatement_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_intg_BankStatement_IsActive] DEFAULT (1),                    -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_intg_BankStatement] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_BankStatement] UNIQUE ([TenantId], [CompanyCodeId], [StatementNumber], [StatementDate])
    );
END
GO

/* intg.BankStatementItem - Bank statement line and its clearing result (reference: FEBEP) */
IF OBJECT_ID(N'intg.BankStatementItem', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[BankStatementItem]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [BankStatementId]          bigint NOT NULL,                                                                   -- Statement
        [ItemNumber]               int NOT NULL,                                                                      -- Item number
        [ValueDate]                date NOT NULL,                                                                     -- Value date
        [BookingDate]              date NOT NULL,                                                                     -- Booking date
        [Amount]                   decimal(19,4) NOT NULL,                                                            -- Amount (signed)
        [CurrencyCode]             nvarchar(5) NOT NULL,                                                              -- Currency
        [DebitCreditIndicator]     nvarchar(1) NOT NULL,                                                              -- S debit, H credit
        [BankTransactionCode]      nvarchar(10) NULL,                                                                 -- Bank transaction code
        [PaymentReference]         nvarchar(40) NULL,                                                                 -- End-to-end reference
        [PartnerName]              nvarchar(60) NULL,                                                                 -- Counterparty name
        [PartnerIban]              nvarchar(34) NULL,                                                                 -- Counterparty IBAN
        [PurposeText]              nvarchar(500) NULL,                                                                -- Remittance information
        [MatchedBusinessPartnerId] bigint NULL,                                                                       -- Partner identified
        [MatchedOpenItemId]        bigint NULL,                                                                       -- Open item identified
        [MatchConfidencePercent]   decimal(9,4) NULL,                                                                 -- Confidence of the automatic match
        [Status]                   nvarchar(20) NOT NULL,                                                             -- Unprocessed, Matched, PartiallyMatched, Posted, Rejected, Manual
        [JournalEntryHeaderId]     bigint NULL,                                                                       -- Accounting document created
        [ClearingDocumentId]       bigint NULL,                                                                       -- Clearing document created
        [ProcessedBy]              nvarchar(64) NULL,                                                                 -- User who processed the item
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_intg_BankStatementItem_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_intg_BankStatementItem_IsActive] DEFAULT (1),          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_intg_BankStatementItem] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_BankStatementItem] UNIQUE ([TenantId], [BankStatementId], [ItemNumber])
    );
END
GO

/* intg.IdempotencyKey - Recorded idempotency key and its stored response */
IF OBJECT_ID(N'intg.IdempotencyKey', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[IdempotencyKey]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [IdempotencyKey]     uniqueidentifier NOT NULL,                                                               -- Key supplied by the caller
        [Endpoint]           nvarchar(255) NOT NULL,                                                                  -- Endpoint the key applies to
        [RequestHash]        nvarchar(64) NOT NULL,                                                                   -- SHA-256 of the request body - mismatch is rejected
        [Status]             nvarchar(20) NOT NULL,                                                                   -- InProgress, Completed, Failed
        [ResponseStatusCode] int NULL,                                                                                -- Stored HTTP status
        [ResponseBody]       nvarchar(max) NULL,                                                                      -- Stored response
        [ResultObjectType]   nvarchar(40) NULL,                                                                       -- Object created by the request
        [ResultObjectId]     bigint NULL,                                                                             -- Key of the created object
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_intg_IdempotencyKey_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- First seen (UTC)
        [CompletedAt]        datetime2(3) NULL,                                                                       -- Completion timestamp (UTC)
        [ExpiresAt]          datetime2(3) NOT NULL,                                                                   -- Expiry of the key
        [ApiClientId]        bigint NULL,                                                                             -- Calling client
        CONSTRAINT [PK_intg_IdempotencyKey] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_IdempotencyKey] UNIQUE ([TenantId], [IdempotencyKey], [Endpoint])
    );
END
GO

/* intg.ImportJob - CSV / Excel import run */
IF OBJECT_ID(N'intg.ImportJob', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[ImportJob]
    (
        [Id]              bigint IDENTITY(1,1) NOT NULL,                                                              -- Surrogate key
        [TenantId]        int NOT NULL,                                                                               -- Owning tenant - every query is filtered by it
        [JobNumber]       nvarchar(20) NOT NULL,                                                                      -- Job number
        [ImportType]      nvarchar(40) NOT NULL,                                                                      -- BusinessPartner, GLAccount, JournalEntry, ExchangeRate, AssetLegacy, OpeningBalance
        [FileName]        nvarchar(255) NOT NULL,                                                                     -- Uploaded file name
        [FileUri]         nvarchar(500) NOT NULL,                                                                     -- Stored file location
        [Checksum]        nvarchar(64) NOT NULL,                                                                      -- SHA-256 of the file
        [CompanyCodeId]   bigint NULL,                                                                                -- Target company code
        [Status]          nvarchar(20) NOT NULL,                                                                      -- Uploaded, Validating, Validated, Importing, Completed, CompletedWithErrors, Failed, Cancelled
        [IsTestRun]       bit NOT NULL CONSTRAINT [DF_intg_ImportJob_IsTestRun] DEFAULT (0),                          -- Validation only
        [TotalRowCount]   int NULL,                                                                                   -- Rows in the file
        [SuccessRowCount] int NULL,                                                                                   -- Rows imported
        [ErrorRowCount]   int NULL,                                                                                   -- Rows rejected
        [StartedAt]       datetime2(3) NULL,                                                                          -- Start (UTC)
        [CompletedAt]     datetime2(3) NULL,                                                                          -- Completion (UTC)
        [ExecutedBy]      nvarchar(64) NOT NULL,                                                                      -- Executing user
        [CorrelationId]   uniqueidentifier NULL,                                                                      -- Correlation id
        [CreatedAt]       datetime2(3) NOT NULL CONSTRAINT [DF_intg_ImportJob_CreatedAt] DEFAULT (SYSUTCDATETIME()),  -- Creation timestamp (UTC)
        [CreatedBy]       nvarchar(64) NOT NULL,                                                                      -- Creating user name
        [ModifiedAt]      datetime2(3) NULL,                                                                          -- Last change timestamp (UTC)
        [ModifiedBy]      nvarchar(64) NULL,                                                                          -- Last changing user name
        [RowVersion]      rowversion NOT NULL,                                                                        -- Optimistic concurrency token
        [IsActive]        bit NOT NULL CONSTRAINT [DF_intg_ImportJob_IsActive] DEFAULT (1),                           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_intg_ImportJob] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_ImportJob] UNIQUE ([TenantId], [JobNumber])
    );
END
GO

/* intg.ImportJobError - Rejected row of an import job */
IF OBJECT_ID(N'intg.ImportJobError', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[ImportJobError]
    (
        [Id]           bigint IDENTITY(1,1) NOT NULL,                                                                 -- Surrogate key
        [TenantId]     int NOT NULL,                                                                                  -- Owning tenant - every query is filtered by it
        [ImportJobId]  bigint NOT NULL,                                                                               -- Import job
        [RowNumber]    int NOT NULL,                                                                                  -- Row in the source file
        [FieldName]    nvarchar(64) NULL,                                                                             -- Field that failed
        [RawValue]     nvarchar(500) NULL,                                                                            -- Value supplied
        [ErrorCode]    nvarchar(40) NOT NULL,                                                                         -- Stable application error code
        [ErrorMessage] nvarchar(500) NOT NULL,                                                                        -- Message shown to the user
        [Severity]     nvarchar(10) NOT NULL,                                                                         -- Error, Warning
        CONSTRAINT [PK_intg_ImportJobError] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_ImportJobError] UNIQUE ([TenantId], [ImportJobId], [RowNumber])
    );
END
GO

/* intg.InboundMessage - Received message, stored before processing */
IF OBJECT_ID(N'intg.InboundMessage', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[InboundMessage]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [MessageId]        uniqueidentifier NOT NULL,                                                                 -- Message identifier
        [SourceSystem]     nvarchar(40) NOT NULL,                                                                     -- Sending system
        [MessageType]      nvarchar(80) NOT NULL,                                                                     -- Message type
        [PayloadJson]      nvarchar(max) NOT NULL,                                                                    -- Raw payload
        [ReceivedAt]       datetime2(3) NOT NULL,                                                                     -- Receipt timestamp (UTC)
        [Status]           nvarchar(20) NOT NULL,                                                                     -- Received, Processing, Processed, Failed, Rejected, Duplicate
        [ProcessedAt]      datetime2(3) NULL,                                                                         -- Processing timestamp (UTC)
        [AttemptCount]     int NOT NULL,                                                                              -- Processing attempts
        [ResultObjectType] nvarchar(40) NULL,                                                                         -- Object created
        [ResultObjectId]   bigint NULL,                                                                               -- Key of the created object
        [ErrorMessage]     nvarchar(500) NULL,                                                                        -- Error message
        [ApiClientId]      bigint NULL,                                                                               -- Sending client
        [CorrelationId]    uniqueidentifier NULL,                                                                     -- Correlation id
        CONSTRAINT [PK_intg_InboundMessage] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_InboundMessage] UNIQUE ([TenantId], [MessageId])
    );
END
GO

/* intg.IntegrationEndpoint - Outbound endpoint configuration (payroll, bank, logistics) */
IF OBJECT_ID(N'intg.IntegrationEndpoint', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[IntegrationEndpoint]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [EndpointCode]        nvarchar(40) NOT NULL,                                                                  -- Endpoint key
        [Name]                nvarchar(60) NOT NULL,                                                                  -- Description
        [Direction]           nvarchar(10) NOT NULL,                                                                  -- Inbound, Outbound
        [Protocol]            nvarchar(20) NOT NULL,                                                                  -- Https, Sftp, FileShare, MessageQueue
        [TargetUrl]           nvarchar(500) NULL,                                                                     -- Endpoint address
        [AuthenticationType]  nvarchar(20) NOT NULL,                                                                  -- None, Basic, ApiKey, OAuth2, Certificate
        [CredentialReference] nvarchar(255) NULL,                                                                     -- Reference to the secret store - never the secret itself
        [PayloadFormat]       nvarchar(20) NOT NULL,                                                                  -- Json, Xml, Csv, Fixed
        [ScheduleCron]        nvarchar(40) NULL,                                                                      -- Schedule for polling or sending
        [TimeoutSeconds]      int NOT NULL,                                                                           -- Request timeout
        [MaxRetries]          int NOT NULL,                                                                           -- Retry limit
        [IsActive]            bit NOT NULL CONSTRAINT [DF_intg_IntegrationEndpoint_IsActive] DEFAULT (1),             -- Endpoint active
        [LastRunAt]           datetime2(3) NULL,                                                                      -- Last run (UTC)
        [LastRunStatus]       nvarchar(20) NULL,                                                                      -- Result of the last run
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_intg_IntegrationEndpoint_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        CONSTRAINT [PK_intg_IntegrationEndpoint] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_IntegrationEndpoint] UNIQUE ([TenantId], [EndpointCode])
    );
END
GO

/* intg.OutboxMessage - Transactional outbox - events written with the business transaction */
IF OBJECT_ID(N'intg.OutboxMessage', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[OutboxMessage]
    (
        [Id]            bigint IDENTITY(1,1) NOT NULL,                                                                -- Surrogate key
        [TenantId]      int NOT NULL,                                                                                 -- Owning tenant - every query is filtered by it
        [MessageId]     uniqueidentifier NOT NULL,                                                                    -- Message identifier
        [EventType]     nvarchar(80) NOT NULL,                                                                        -- e.g. JournalEntryPosted, BusinessPartnerChanged
        [AggregateType] nvarchar(40) NOT NULL,                                                                        -- Source aggregate
        [AggregateId]   bigint NOT NULL,                                                                              -- Source key
        [PayloadJson]   nvarchar(max) NOT NULL,                                                                       -- Event payload
        [Headers]       nvarchar(max) NULL,                                                                           -- Message headers
        [OccurredAt]    datetime2(3) NOT NULL,                                                                        -- Business event timestamp (UTC)
        [Status]        nvarchar(20) NOT NULL,                                                                        -- Pending, Publishing, Published, Failed, DeadLettered
        [PublishedAt]   datetime2(3) NULL,                                                                            -- Publication timestamp (UTC)
        [AttemptCount]  int NOT NULL,                                                                                 -- Delivery attempts
        [NextAttemptAt] datetime2(3) NULL,                                                                            -- Next retry (UTC)
        [LastError]     nvarchar(500) NULL,                                                                           -- Last error
        [CorrelationId] uniqueidentifier NULL,                                                                        -- Originating request
        [CreatedBy]     nvarchar(64) NOT NULL,                                                                        -- Creating user or job
        CONSTRAINT [PK_intg_OutboxMessage] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_OutboxMessage] UNIQUE ([TenantId], [MessageId])
    );
END
GO

/* intg.WebhookDelivery - Delivery attempt of a webhook */
IF OBJECT_ID(N'intg.WebhookDelivery', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[WebhookDelivery]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [WebhookSubscriptionId] bigint NOT NULL,                                                                      -- Subscription
        [OutboxMessageId]       bigint NOT NULL,                                                                      -- Message delivered
        [AttemptNumber]         int NOT NULL,                                                                         -- Attempt number
        [AttemptedAt]           datetime2(3) NOT NULL,                                                                -- Attempt timestamp (UTC)
        [ResponseStatusCode]    int NULL,                                                                             -- HTTP status returned
        [ResponseBody]          nvarchar(1000) NULL,                                                                  -- Truncated response body
        [DurationMs]            int NULL,                                                                             -- Duration in milliseconds
        [IsSuccessful]          bit NOT NULL CONSTRAINT [DF_intg_WebhookDelivery_IsSuccessful] DEFAULT (0),           -- Delivery accepted
        [ErrorMessage]          nvarchar(500) NULL,                                                                   -- Error message
        CONSTRAINT [PK_intg_WebhookDelivery] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* intg.WebhookSubscription - Subscription of an external system to events */
IF OBJECT_ID(N'intg.WebhookSubscription', N'U') IS NULL
BEGIN
    CREATE TABLE [intg].[WebhookSubscription]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [SubscriptionCode]        nvarchar(40) NOT NULL,                                                              -- Subscription key
        [ApiClientId]             bigint NOT NULL,                                                                    -- Subscribing client
        [EventTypes]              nvarchar(500) NOT NULL,                                                             -- Subscribed event types
        [TargetUrl]               nvarchar(500) NOT NULL,                                                             -- Delivery endpoint
        [SecretReference]         nvarchar(255) NULL,                                                                 -- Reference to the signing secret in the secret store
        [SignatureAlgorithm]      nvarchar(20) NOT NULL,                                                              -- HMACSHA256
        [FilterExpression]        nvarchar(500) NULL,                                                                 -- Additional filter on the payload
        [MaxRetries]              int NOT NULL,                                                                       -- Maximum delivery attempts
        [TimeoutSeconds]          int NOT NULL,                                                                       -- Request timeout
        [IsActive]                bit NOT NULL CONSTRAINT [DF_intg_WebhookSubscription_IsActive] DEFAULT (1),         -- Subscription active
        [LastDeliveryAt]          datetime2(3) NULL,                                                                  -- Last successful delivery (UTC)
        [ConsecutiveFailureCount] int NOT NULL,                                                                       -- Consecutive failures - disables at the threshold
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_intg_WebhookSubscription_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        CONSTRAINT [PK_intg_WebhookSubscription] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_intg_WebhookSubscription] UNIQUE ([TenantId], [SubscriptionCode])
    );
END
GO
