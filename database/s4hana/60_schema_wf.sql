/* ============================================================================
   S/4HANA-inspired ERP - schema [wf]
   Workflow and approval (7 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* wf.Notification - Notification queued or delivered to a user */
IF OBJECT_ID(N'wf.Notification', N'U') IS NULL
BEGIN
    CREATE TABLE [wf].[Notification]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [RecipientUserId]  bigint NOT NULL,                                                                           -- Recipient
        [NotificationType] nvarchar(30) NOT NULL,                                                                     -- ApprovalRequest, Reminder, Escalation, Decision, SystemAlert
        [Channel]          nvarchar(20) NOT NULL,                                                                     -- InApp, Email, Push, SignalR
        [Subject]          nvarchar(255) NOT NULL,                                                                    -- Subject
        [Body]             nvarchar(max) NULL,                                                                        -- Message body
        [ObjectType]       nvarchar(40) NULL,                                                                         -- Related object type
        [ObjectId]         bigint NULL,                                                                               -- Related object key
        [WorkflowTaskId]   bigint NULL,                                                                               -- Related approval task
        [NavigationUrl]    nvarchar(500) NULL,                                                                        -- Deep link into the application
        [Priority]         nvarchar(10) NOT NULL,                                                                     -- Low, Normal, High
        [Status]           nvarchar(20) NOT NULL,                                                                     -- Queued, Sent, Delivered, Failed, Read
        [SentAt]           datetime2(3) NULL,                                                                         -- Send timestamp (UTC)
        [ReadAt]           datetime2(3) NULL,                                                                         -- Read timestamp (UTC)
        [RetryCount]       int NOT NULL,                                                                              -- Delivery attempts
        [ErrorMessage]     nvarchar(500) NULL,                                                                        -- Last delivery error
        CONSTRAINT [PK_wf_Notification] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* wf.WorkflowDefinition - Approval process definition */
IF OBJECT_ID(N'wf.WorkflowDefinition', N'U') IS NULL
BEGIN
    CREATE TABLE [wf].[WorkflowDefinition]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [WorkflowCode]           nvarchar(20) NOT NULL,                                                               -- Workflow key, e.g. JE_APPROVAL
        [Version]                int NOT NULL,                                                                        -- Definition version - instances pin their version
        [Name]                   nvarchar(60) NOT NULL,                                                               -- Description
        [ObjectType]             nvarchar(40) NOT NULL,                                                               -- JournalEntry, VendorInvoice, Payment, PaymentRun, BusinessPartner, CustomObjectRequest
        [ApprovalMode]           nvarchar(20) NOT NULL,                                                               -- Sequential, Parallel, MultiLevel
        [IsMakerCheckerEnforced] bit NOT NULL CONSTRAINT [DF_wf_WorkflowDefinition_IsMakerCheckerEnforced] DEFAULT (0),-- Submitter may not approve
        [AllowDelegation]        bit NOT NULL CONSTRAINT [DF_wf_WorkflowDefinition_AllowDelegation] DEFAULT (0),      -- Delegation permitted
        [AllowResubmission]      bit NOT NULL CONSTRAINT [DF_wf_WorkflowDefinition_AllowResubmission] DEFAULT (0),    -- Rejected documents may be resubmitted
        [EscalationHours]        int NULL,                                                                            -- Hours before escalation
        [ReminderHours]          int NULL,                                                                            -- Hours before a reminder is sent
        [IsActive]               bit NOT NULL CONSTRAINT [DF_wf_WorkflowDefinition_IsActive] DEFAULT (1),             -- Active version
        [EffectiveFrom]          date NOT NULL,                                                                       -- First day this version applies
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_wf_WorkflowDefinition_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        CONSTRAINT [PK_wf_WorkflowDefinition] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_wf_WorkflowDefinition] UNIQUE ([TenantId], [WorkflowCode], [Version])
    );
END
GO

/* wf.WorkflowHistory - Immutable event log of a workflow instance */
IF OBJECT_ID(N'wf.WorkflowHistory', N'U') IS NULL
BEGIN
    CREATE TABLE [wf].[WorkflowHistory]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [WorkflowInstanceId] bigint NOT NULL,                                                                         -- Workflow instance
        [EventSequence]      int NOT NULL,                                                                            -- Event order
        [EventType]          nvarchar(30) NOT NULL,                                                                   -- Submitted, Assigned, Approved, Rejected, Delegated, Escalated, Reminded, Cancelled, Resubmitted, Completed
        [StepNumber]         int NULL,                                                                                -- Step concerned
        [PerformedByUserId]  bigint NULL,                                                                             -- Acting user
        [PerformedAt]        datetime2(3) NOT NULL,                                                                   -- Event timestamp (UTC)
        [FromStatus]         nvarchar(20) NULL,                                                                       -- Previous status
        [ToStatus]           nvarchar(20) NULL,                                                                       -- New status
        [Comment]            nvarchar(1000) NULL,                                                                     -- Comment
        [CorrelationId]      uniqueidentifier NULL,                                                                   -- Request correlation id
        CONSTRAINT [PK_wf_WorkflowHistory] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* wf.WorkflowInstance - Running or finished approval process for one document */
IF OBJECT_ID(N'wf.WorkflowInstance', N'U') IS NULL
BEGIN
    CREATE TABLE [wf].[WorkflowInstance]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [InstanceNumber]       nvarchar(20) NOT NULL,                                                                 -- Instance number
        [WorkflowDefinitionId] bigint NOT NULL,                                                                       -- Definition and version used
        [ObjectType]           nvarchar(40) NOT NULL,                                                                 -- Object under approval
        [ObjectId]             bigint NOT NULL,                                                                       -- Object key
        [ObjectDescription]    nvarchar(255) NULL,                                                                    -- Text shown in the inbox
        [CompanyCodeId]        bigint NOT NULL,                                                                       -- Company code
        [CurrencyCode]         nvarchar(5) NULL,                                                                      -- Currency of the amount
        [Amount]               decimal(19,4) NULL,                                                                    -- Amount that drove rule selection
        [CurrentStepNumber]    int NOT NULL,                                                                          -- Step currently active
        [Status]               nvarchar(20) NOT NULL,                                                                 -- Submitted, PendingApproval, Approved, Rejected, Cancelled, Escalated, Completed
        [SubmittedBy]          nvarchar(64) NOT NULL,                                                                 -- Submitting user
        [SubmittedAt]          datetime2(3) NOT NULL,                                                                 -- Submission timestamp (UTC)
        [CompletedAt]          datetime2(3) NULL,                                                                     -- Completion timestamp (UTC)
        [DueAt]                datetime2(3) NULL,                                                                     -- Deadline for the current step
        [FinalDecision]        nvarchar(20) NULL,                                                                     -- Approved, Rejected
        [RejectionReason]      nvarchar(500) NULL,                                                                    -- Reason given on rejection
        [ResubmissionCount]    int NOT NULL,                                                                          -- Number of resubmissions
        [CorrelationId]        uniqueidentifier NULL,                                                                 -- Request correlation id
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_wf_WorkflowInstance_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_wf_WorkflowInstance_IsActive] DEFAULT (1),                 -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_wf_WorkflowInstance] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_wf_WorkflowInstance] UNIQUE ([TenantId], [InstanceNumber])
    );
END
GO

/* wf.WorkflowRule - Condition selecting a workflow and its approval level */
IF OBJECT_ID(N'wf.WorkflowRule', N'U') IS NULL
BEGIN
    CREATE TABLE [wf].[WorkflowRule]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [WorkflowDefinitionId] bigint NOT NULL,                                                                       -- Workflow definition
        [RuleSequence]         int NOT NULL,                                                                          -- Evaluation order - first match wins
        [Name]                 nvarchar(60) NOT NULL,                                                                 -- Rule description
        [CompanyCodeId]        bigint NULL,                                                                           -- Applies to this company code
        [DocumentTypeId]       bigint NULL,                                                                           -- Applies to this document type
        [DepartmentCode]       nvarchar(10) NULL,                                                                     -- Applies to this department
        [CostCenterId]         bigint NULL,                                                                           -- Applies to this cost centre
        [ProfitCenterId]       bigint NULL,                                                                           -- Applies to this profit centre
        [CurrencyCode]         nvarchar(5) NULL,                                                                      -- Currency of the amount limits
        [MinimumAmount]        decimal(19,4) NULL,                                                                    -- Lower amount limit
        [MaximumAmount]        decimal(19,4) NULL,                                                                    -- Upper amount limit
        [SourceModule]         nvarchar(10) NULL,                                                                     -- Applies to this module
        [ConditionExpression]  nvarchar(max) NULL,                                                                    -- Additional condition (expression tree JSON)
        [IsActive]             bit NOT NULL CONSTRAINT [DF_wf_WorkflowRule_IsActive] DEFAULT (1),                     -- Rule active
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_wf_WorkflowRule_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        CONSTRAINT [PK_wf_WorkflowRule] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_wf_WorkflowRule] UNIQUE ([TenantId], [WorkflowDefinitionId], [RuleSequence])
    );
END
GO

/* wf.WorkflowStep - Step of a workflow definition */
IF OBJECT_ID(N'wf.WorkflowStep', N'U') IS NULL
BEGIN
    CREATE TABLE [wf].[WorkflowStep]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [WorkflowDefinitionId]      bigint NOT NULL,                                                                  -- Workflow definition
        [StepNumber]                int NOT NULL,                                                                     -- Step sequence
        [Name]                      nvarchar(60) NOT NULL,                                                            -- Step description
        [StepType]                  nvarchar(20) NOT NULL,                                                            -- Approval, Review, Notification, AutomaticCheck, Posting
        [ApproverDeterminationType] nvarchar(20) NOT NULL,                                                            -- Role, User, CostCenterManager, DepartmentHead, LineManager, Expression
        [ApproverRoleId]            bigint NULL,                                                                      -- Approving role
        [ApproverUserId]            bigint NULL,                                                                      -- Named approver
        [ApproverExpression]        nvarchar(255) NULL,                                                               -- Expression resolving the approver
        [RequiredApprovals]         int NOT NULL,                                                                     -- Approvals needed at this step
        [IsParallel]                bit NOT NULL CONSTRAINT [DF_wf_WorkflowStep_IsParallel] DEFAULT (0),              -- Approvers work in parallel
        [IsOptional]                bit NOT NULL CONSTRAINT [DF_wf_WorkflowStep_IsOptional] DEFAULT (0),              -- Step may be skipped
        [MinimumAmount]             decimal(19,4) NULL,                                                               -- Step applies from this amount
        [EscalationRoleId]          bigint NULL,                                                                      -- Role notified on escalation
        [EscalationHours]           int NULL,                                                                         -- Hours before escalation
        [OnRejectAction]            nvarchar(20) NOT NULL,                                                            -- ReturnToSubmitter, ReturnToPreviousStep, Terminate
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_wf_WorkflowStep_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_wf_WorkflowStep_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_wf_WorkflowStep] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_wf_WorkflowStep] UNIQUE ([TenantId], [WorkflowDefinitionId], [StepNumber])
    );
END
GO

/* wf.WorkflowTask - Individual approval task in an inbox */
IF OBJECT_ID(N'wf.WorkflowTask', N'U') IS NULL
BEGIN
    CREATE TABLE [wf].[WorkflowTask]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [WorkflowInstanceId] bigint NOT NULL,                                                                         -- Workflow instance
        [StepNumber]         int NOT NULL,                                                                            -- Step
        [AssignedUserId]     bigint NOT NULL,                                                                         -- Assigned approver
        [AssignedRoleId]     bigint NULL,                                                                             -- Role the task was routed through
        [Status]             nvarchar(20) NOT NULL,                                                                   -- Pending, Approved, Rejected, Delegated, Escalated, Withdrawn, Expired
        [AssignedAt]         datetime2(3) NOT NULL,                                                                   -- Assignment timestamp (UTC)
        [DueAt]              datetime2(3) NULL,                                                                       -- Due timestamp (UTC)
        [DecidedAt]          datetime2(3) NULL,                                                                       -- Decision timestamp (UTC)
        [DecidedByUserId]    bigint NULL,                                                                             -- User who decided (may be a substitute)
        [Decision]           nvarchar(20) NULL,                                                                       -- Approve, Reject, RequestInformation
        [Comment]            nvarchar(1000) NULL,                                                                     -- Approver comment
        [DelegatedToUserId]  bigint NULL,                                                                             -- Delegate
        [DelegationReason]   nvarchar(255) NULL,                                                                      -- Reason for delegation
        [ReminderCount]      int NOT NULL,                                                                            -- Reminders sent
        [IsReadOnly]         bit NOT NULL CONSTRAINT [DF_wf_WorkflowTask_IsReadOnly] DEFAULT (0),                     -- Informational task
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_wf_WorkflowTask_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_wf_WorkflowTask_IsActive] DEFAULT (1),                       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_wf_WorkflowTask] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_wf_WorkflowTask] UNIQUE ([TenantId], [WorkflowInstanceId], [StepNumber], [AssignedUserId])
    );
END
GO
