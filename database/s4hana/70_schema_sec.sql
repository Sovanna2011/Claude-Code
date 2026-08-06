/* ============================================================================
   S/4HANA-inspired ERP - schema [sec]
   Users and security (19 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* sec.AuthorizationField - Field of an authorization object (reference: TOBJT/AUTHX) */
IF OBJECT_ID(N'sec.AuthorizationField', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[AuthorizationField]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [AuthorizationObjectId] bigint NOT NULL,                                                                      -- Authorization object
        [FieldName]             nvarchar(20) NOT NULL,                                                                -- Field, e.g. BUKRS, ACTVT, KOSTL
        [Name]                  nvarchar(60) NOT NULL,                                                                -- Description
        [DataElementId]         bigint NULL,                                                                          -- Data element defining the value domain
        [CheckTableName]        nvarchar(64) NULL,                                                                    -- Table of permitted values
        [IsOrganizationalLevel] bit NOT NULL CONSTRAINT [DF_sec_AuthorizationField_IsOrganizationalLevel] DEFAULT (0),-- Organisational level (derived roles fill it)
        [FieldPosition]         int NOT NULL,                                                                         -- Order
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_sec_AuthorizationField_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_sec_AuthorizationField_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_AuthorizationField] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_AuthorizationField] UNIQUE ([TenantId], [AuthorizationObjectId], [FieldName])
    );
END
GO

/* sec.AuthorizationObject - Authorization object (a set of fields checked together) (reference: TOBJ) */
IF OBJECT_ID(N'sec.AuthorizationObject', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[AuthorizationObject]
    (
        [Id]          bigint IDENTITY(1,1) NOT NULL,                                                                  -- Surrogate key
        [TenantId]    int NOT NULL,                                                                                   -- Owning tenant - every query is filtered by it
        [ObjectCode]  nvarchar(20) NOT NULL,                                                                          -- Object key, e.g. F_BKPF_BUK
        [Name]        nvarchar(60) NOT NULL,                                                                          -- Description
        [ObjectClass] nvarchar(20) NOT NULL,                                                                          -- Class, e.g. FI, CO, BC
        [Description] nvarchar(255) NULL,                                                                             -- Purpose of the object
        [CreatedAt]   datetime2(3) NOT NULL CONSTRAINT [DF_sec_AuthorizationObject_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]   nvarchar(64) NOT NULL,                                                                          -- Creating user name
        [ModifiedAt]  datetime2(3) NULL,                                                                              -- Last change timestamp (UTC)
        [ModifiedBy]  nvarchar(64) NULL,                                                                              -- Last changing user name
        [RowVersion]  rowversion NOT NULL,                                                                            -- Optimistic concurrency token
        [IsActive]    bit NOT NULL CONSTRAINT [DF_sec_AuthorizationObject_IsActive] DEFAULT (1),                      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_AuthorizationObject] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_AuthorizationObject] UNIQUE ([TenantId], [ObjectCode])
    );
END
GO

/* sec.AuthorizationValue - Value or range granted for an authorization field in a role */
IF OBJECT_ID(N'sec.AuthorizationValue', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[AuthorizationValue]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [RoleId]                bigint NOT NULL,                                                                      -- Role
        [AuthorizationObjectId] bigint NOT NULL,                                                                      -- Authorization object
        [AuthorizationFieldId]  bigint NOT NULL,                                                                      -- Field
        [LineNumber]            int NOT NULL,                                                                         -- Value line
        [SignIndicator]         nvarchar(1) NOT NULL,                                                                 -- I include, E exclude
        [Operator]              nvarchar(10) NOT NULL,                                                                -- EQ, BT, CP, NE
        [LowValue]              nvarchar(60) NOT NULL,                                                                -- Value or interval start (* = all)
        [HighValue]             nvarchar(60) NULL,                                                                    -- Interval end
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_sec_AuthorizationValue_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_sec_AuthorizationValue_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_AuthorizationValue] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_AuthorizationValue] UNIQUE ([TenantId], [RoleId], [AuthorizationObjectId], [AuthorizationFieldId], [LineNumber])
    );
END
GO

/* sec.LoginHistory - Logon attempt log */
IF OBJECT_ID(N'sec.LoginHistory', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[LoginHistory]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [UserName]             nvarchar(64) NOT NULL,                                                                 -- Logon name used (kept even if unknown)
        [UserId]               bigint NULL,                                                                           -- User, when resolved
        [AttemptedAt]          datetime2(3) NOT NULL,                                                                 -- Attempt timestamp (UTC)
        [IsSuccessful]         bit NOT NULL CONSTRAINT [DF_sec_LoginHistory_IsSuccessful] DEFAULT (0),                -- Logon succeeded
        [FailureReason]        nvarchar(40) NULL,                                                                     -- InvalidPassword, UserLocked, UserExpired, MfaFailed, UnknownUser
        [AuthenticationMethod] nvarchar(20) NOT NULL,                                                                 -- Password, Oidc, ApiKey, Certificate
        [IpAddress]            nvarchar(45) NOT NULL,                                                                 -- Client IP
        [UserAgent]            nvarchar(255) NULL,                                                                    -- Client user agent
        [SessionId]            uniqueidentifier NULL,                                                                 -- Session created
        [CorrelationId]        uniqueidentifier NULL,                                                                 -- Request correlation id
        CONSTRAINT [PK_sec_LoginHistory] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* sec.PasswordHistory - Previous password hashes, to enforce reuse rules */
IF OBJECT_ID(N'sec.PasswordHistory', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[PasswordHistory]
    (
        [Id]           bigint IDENTITY(1,1) NOT NULL,                                                                 -- Surrogate key
        [TenantId]     int NOT NULL,                                                                                  -- Owning tenant - every query is filtered by it
        [UserId]       bigint NOT NULL,                                                                               -- User
        [ChangedAt]    datetime2(3) NOT NULL,                                                                         -- Change timestamp (UTC)
        [PasswordHash] nvarchar(255) NOT NULL,                                                                        -- Historic hash (never logged, never exported)
        [PasswordSalt] nvarchar(64) NOT NULL,                                                                         -- Historic salt
        [ChangedBy]    nvarchar(64) NOT NULL,                                                                         -- User or administrator who changed it
        [ChangeReason] nvarchar(40) NOT NULL,                                                                         -- UserChange, AdminReset, Expired, ForcedPolicy
        CONSTRAINT [PK_sec_PasswordHistory] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_PasswordHistory] UNIQUE ([TenantId], [UserId], [ChangedAt])
    );
END
GO

/* sec.Permission - Atomic permission */
IF OBJECT_ID(N'sec.Permission', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[Permission]
    (
        [Id]             bigint IDENTITY(1,1) NOT NULL,                                                               -- Surrogate key
        [TenantId]       int NOT NULL,                                                                                -- Owning tenant - every query is filtered by it
        [PermissionCode] nvarchar(60) NOT NULL,                                                                       -- Permission key, e.g. Finance.JournalEntry.Post
        [Name]           nvarchar(60) NOT NULL,                                                                       -- Permission name
        [Module]         nvarchar(20) NOT NULL,                                                                       -- Owning module
        [ObjectType]     nvarchar(40) NOT NULL,                                                                       -- Protected object
        [Action]         nvarchar(20) NOT NULL,                                                                       -- Create, Read, Update, Delete, Post, Reverse, Approve, Export, Execute
        [IsCritical]     bit NOT NULL CONSTRAINT [DF_sec_Permission_IsCritical] DEFAULT (0),                          -- Critical permission - always audited
        [CreatedAt]      datetime2(3) NOT NULL CONSTRAINT [DF_sec_Permission_CreatedAt] DEFAULT (SYSUTCDATETIME()),   -- Creation timestamp (UTC)
        [CreatedBy]      nvarchar(64) NOT NULL,                                                                       -- Creating user name
        [ModifiedAt]     datetime2(3) NULL,                                                                           -- Last change timestamp (UTC)
        [ModifiedBy]     nvarchar(64) NULL,                                                                           -- Last changing user name
        [RowVersion]     rowversion NOT NULL,                                                                         -- Optimistic concurrency token
        [IsActive]       bit NOT NULL CONSTRAINT [DF_sec_Permission_IsActive] DEFAULT (1),                            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_Permission] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_Permission] UNIQUE ([TenantId], [PermissionCode])
    );
END
GO

/* sec.Role - Role (collection of permissions and authorizations) */
IF OBJECT_ID(N'sec.Role', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[Role]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [RoleCode]                 nvarchar(40) NOT NULL,                                                             -- Role key, e.g. FI_ACCOUNTANT
        [Name]                     nvarchar(60) NOT NULL,                                                             -- Role name
        [Description]              nvarchar(255) NULL,                                                                -- Description
        [RoleType]                 nvarchar(20) NOT NULL,                                                             -- Single, Composite, Derived
        [ParentRoleId]             bigint NULL,                                                                       -- Reference role for derived roles
        [IsCriticalRole]           bit NOT NULL CONSTRAINT [DF_sec_Role_IsCriticalRole] DEFAULT (0),                  -- Requires extra approval to assign
        [RequiresApprovalToAssign] bit NOT NULL CONSTRAINT [DF_sec_Role_RequiresApprovalToAssign] DEFAULT (0),        -- Assignment goes through workflow
        [IsSystemRole]             bit NOT NULL CONSTRAINT [DF_sec_Role_IsSystemRole] DEFAULT (0),                    -- Delivered role - cannot be deleted
        [ValidFrom]                date NOT NULL,                                                                     -- First day the record is valid
        [ValidTo]                  date NOT NULL,                                                                     -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_sec_Role_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_sec_Role_IsActive] DEFAULT (1),                        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_Role] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_Role] UNIQUE ([TenantId], [RoleCode])
    );
END
GO

/* sec.RolePermission - Permission granted by a role */
IF OBJECT_ID(N'sec.RolePermission', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[RolePermission]
    (
        [Id]           bigint IDENTITY(1,1) NOT NULL,                                                                 -- Surrogate key
        [TenantId]     int NOT NULL,                                                                                  -- Owning tenant - every query is filtered by it
        [RoleId]       bigint NOT NULL,                                                                               -- Role
        [PermissionId] bigint NOT NULL,                                                                               -- Permission
        [IsGranted]    bit NOT NULL CONSTRAINT [DF_sec_RolePermission_IsGranted] DEFAULT (0),                         -- Granted (1) or explicitly denied (0)
        [CreatedAt]    datetime2(3) NOT NULL CONSTRAINT [DF_sec_RolePermission_CreatedAt] DEFAULT (SYSUTCDATETIME()), -- Creation timestamp (UTC)
        [CreatedBy]    nvarchar(64) NOT NULL,                                                                         -- Creating user name
        [ModifiedAt]   datetime2(3) NULL,                                                                             -- Last change timestamp (UTC)
        [ModifiedBy]   nvarchar(64) NULL,                                                                             -- Last changing user name
        [RowVersion]   rowversion NOT NULL,                                                                           -- Optimistic concurrency token
        [IsActive]     bit NOT NULL CONSTRAINT [DF_sec_RolePermission_IsActive] DEFAULT (1),                          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_RolePermission] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_RolePermission] UNIQUE ([TenantId], [RoleId], [PermissionId])
    );
END
GO

/* sec.RoleTransactionCode - T-codes a role may start */
IF OBJECT_ID(N'sec.RoleTransactionCode', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[RoleTransactionCode]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [RoleId]            bigint NOT NULL,                                                                          -- Role
        [TransactionCodeId] bigint NOT NULL,                                                                          -- Transaction code
        [IsGranted]         bit NOT NULL CONSTRAINT [DF_sec_RoleTransactionCode_IsGranted] DEFAULT (0),               -- Granted or denied
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_sec_RoleTransactionCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_sec_RoleTransactionCode_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_RoleTransactionCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_RoleTransactionCode] UNIQUE ([TenantId], [RoleId], [TransactionCodeId])
    );
END
GO

/* sec.SegregationOfDutiesRule - Conflicting duty combination */
IF OBJECT_ID(N'sec.SegregationOfDutiesRule', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[SegregationOfDutiesRule]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [RuleCode]          nvarchar(20) NOT NULL,                                                                    -- Rule key, e.g. SOD_AP_01
        [Name]              nvarchar(60) NOT NULL,                                                                    -- Rule name
        [Description]       nvarchar(500) NULL,                                                                       -- Risk described by the rule
        [RiskLevel]         nvarchar(10) NOT NULL,                                                                    -- Low, Medium, High, Critical
        [ConflictType]      nvarchar(20) NOT NULL,                                                                    -- PermissionPair, RolePair, TransactionPair
        [FirstObjectCode]   nvarchar(60) NOT NULL,                                                                    -- First conflicting permission / role / T-code
        [SecondObjectCode]  nvarchar(60) NOT NULL,                                                                    -- Second conflicting object
        [EnforcementMode]   nvarchar(20) NOT NULL,                                                                    -- Block, WarnAndLog, LogOnly
        [MitigationControl] nvarchar(500) NULL,                                                                       -- Compensating control if the conflict is accepted
        [IsActive]          bit NOT NULL CONSTRAINT [DF_sec_SegregationOfDutiesRule_IsActive] DEFAULT (1),            -- Rule active
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_sec_SegregationOfDutiesRule_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        CONSTRAINT [PK_sec_SegregationOfDutiesRule] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_SegregationOfDutiesRule] UNIQUE ([TenantId], [RuleCode])
    );
END
GO

/* sec.SegregationOfDutiesViolation - Detected or accepted conflict for a user */
IF OBJECT_ID(N'sec.SegregationOfDutiesViolation', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[SegregationOfDutiesViolation]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [SegregationOfDutiesRuleId] bigint NOT NULL,                                                                  -- Rule violated
        [UserId]                    bigint NOT NULL,                                                                  -- User concerned
        [DetectedAt]                datetime2(3) NOT NULL,                                                            -- Detection timestamp (UTC)
        [Status]                    nvarchar(20) NOT NULL,                                                            -- Open, Mitigated, Accepted, Resolved
        [MitigationNote]            nvarchar(500) NULL,                                                               -- Mitigation applied
        [AcceptedBy]                nvarchar(64) NULL,                                                                -- Approver of the exception
        [AcceptedAt]                datetime2(3) NULL,                                                                -- Acceptance timestamp (UTC)
        [ReviewDueDate]             date NULL,                                                                        -- Next review
        [ResolvedAt]                datetime2(3) NULL,                                                                -- Resolution timestamp (UTC)
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_sec_SegregationOfDutiesViolation_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_sec_SegregationOfDutiesViolation_IsActive] DEFAULT (1),-- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_SegregationOfDutiesViolation] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_SegregationOfDutiesViolation] UNIQUE ([TenantId], [SegregationOfDutiesRuleId], [UserId])
    );
END
GO

/* sec.TransactionCode - T-code navigation alias (reference: TSTC) */
IF OBJECT_ID(N'sec.TransactionCode', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[TransactionCode]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [TransactionCode]       nvarchar(20) NOT NULL,                                                                -- T-code, e.g. FB50, SE16N
        [Name]                  nvarchar(60) NOT NULL,                                                                -- Description
        [Module]                nvarchar(20) NOT NULL,                                                                -- Owning module
        [RoutePath]             nvarchar(255) NOT NULL,                                                               -- Frontend route, e.g. /finance/journal-entry/new
        [RequiredPermissionId]  bigint NULL,                                                                          -- Permission checked before navigation
        [AuthorizationObjectId] bigint NULL,                                                                          -- Authorization object checked
        [IconName]              nvarchar(40) NULL,                                                                    -- Icon
        [Category]              nvarchar(40) NULL,                                                                    -- Menu category
        [IsFavoriteEligible]    bit NOT NULL CONSTRAINT [DF_sec_TransactionCode_IsFavoriteEligible] DEFAULT (0),      -- May be added to favourites
        [IsBlocked]             bit NOT NULL CONSTRAINT [DF_sec_TransactionCode_IsBlocked] DEFAULT (0),               -- Blocked for all users
        [HelpText]              nvarchar(500) NULL,                                                                   -- Help text
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_sec_TransactionCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_sec_TransactionCode_IsActive] DEFAULT (1),                -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_TransactionCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_TransactionCode] UNIQUE ([TenantId], [TransactionCode])
    );
END
GO

/* sec.User - Application user */
IF OBJECT_ID(N'sec.User', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[User]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [UserName]                 nvarchar(64) NOT NULL,                                                             -- Logon name
        [EmployeeNumber]           nvarchar(20) NULL,                                                                 -- Personnel number
        [BusinessPartnerId]        bigint NULL,                                                                       -- Business partner (employee role)
        [FirstName]                nvarchar(40) NULL,                                                                 -- First name
        [LastName]                 nvarchar(40) NULL,                                                                 -- Last name
        [DisplayName]              nvarchar(100) NOT NULL,                                                            -- Display name
        [Email]                    nvarchar(255) NOT NULL,                                                            -- Email address
        [MobileNumber]             nvarchar(30) NULL,                                                                 -- Mobile number
        [UserType]                 nvarchar(20) NOT NULL,                                                             -- Dialog, Service, Integration, Api, Background, Auditor
        [PasswordHash]             nvarchar(255) NULL,                                                                -- Password hash (never logged)
        [PasswordSalt]             nvarchar(64) NULL,                                                                 -- Password salt
        [PasswordChangedAt]        datetime2(3) NULL,                                                                 -- Last password change (UTC)
        [MustChangePassword]       bit NOT NULL CONSTRAINT [DF_sec_User_MustChangePassword] DEFAULT (0),              -- Password change forced at next logon
        [ExternalIdentityProvider] nvarchar(40) NULL,                                                                 -- OIDC provider name
        [ExternalSubjectId]        nvarchar(255) NULL,                                                                -- Subject id at the provider
        [IsMfaEnabled]             bit NOT NULL CONSTRAINT [DF_sec_User_IsMfaEnabled] DEFAULT (0),                    -- Multi-factor authentication active
        [MfaSecretReference]       nvarchar(255) NULL,                                                                -- Reference to the MFA secret in the secret store
        [LanguageCode]             nvarchar(2) NOT NULL,                                                              -- UI language
        [TimeZoneId]               nvarchar(64) NOT NULL,                                                             -- Display time zone
        [DateFormat]               nvarchar(10) NULL,                                                                 -- Preferred date format
        [DecimalNotation]          nvarchar(10) NULL,                                                                 -- Preferred decimal notation
        [DefaultCompanyCodeId]     bigint NULL,                                                                       -- Default company code
        [DefaultControllingAreaId] bigint NULL,                                                                       -- Default controlling area
        [DefaultCurrencyCode]      nvarchar(5) NULL,                                                                  -- Default display currency
        [DefaultTransactionCode]   nvarchar(20) NULL,                                                                 -- Start transaction
        [Status]                   nvarchar(20) NOT NULL,                                                             -- Active, Locked, Expired, Disabled
        [IsLocked]                 bit NOT NULL CONSTRAINT [DF_sec_User_IsLocked] DEFAULT (0),                        -- Administratively locked
        [LockReason]               nvarchar(255) NULL,                                                                -- Lock reason
        [FailedLoginCount]         int NOT NULL,                                                                      -- Consecutive failed logons
        [LastFailedLoginAt]        datetime2(3) NULL,                                                                 -- Last failed logon (UTC)
        [LastLoginAt]              datetime2(3) NULL,                                                                 -- Last successful logon (UTC)
        [LastLoginIpAddress]       nvarchar(45) NULL,                                                                 -- Last logon origin
        [ValidFrom]                date NOT NULL,                                                                     -- Account valid from
        [ValidTo]                  date NOT NULL,                                                                     -- Account valid to
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_sec_User_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_sec_User_IsActive] DEFAULT (1),                        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_User] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_User] UNIQUE ([TenantId], [UserName])
    );
END
GO

/* sec.UserAuthorization - Authorization granted directly to a user (exception to role-based access) */
IF OBJECT_ID(N'sec.UserAuthorization', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[UserAuthorization]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [UserId]                bigint NOT NULL,                                                                      -- User
        [AuthorizationObjectId] bigint NOT NULL,                                                                      -- Authorization object
        [AuthorizationFieldId]  bigint NOT NULL,                                                                      -- Field
        [LineNumber]            int NOT NULL,                                                                         -- Value line
        [SignIndicator]         nvarchar(1) NOT NULL,                                                                 -- I include, E exclude
        [LowValue]              nvarchar(60) NOT NULL,                                                                -- Value or interval start
        [HighValue]             nvarchar(60) NULL,                                                                    -- Interval end
        [GrantReason]           nvarchar(255) NOT NULL,                                                               -- Justification - required for direct grants
        [ApprovedBy]            nvarchar(64) NOT NULL,                                                                -- Approver
        [ApprovedAt]            datetime2(3) NOT NULL,                                                                -- Approval timestamp (UTC)
        [ValidFrom]             date NOT NULL,                                                                        -- First day the record is valid
        [ValidTo]               date NOT NULL,                                                                        -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_sec_UserAuthorization_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_sec_UserAuthorization_IsActive] DEFAULT (1),              -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_UserAuthorization] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_UserAuthorization] UNIQUE ([TenantId], [UserId], [AuthorizationObjectId], [AuthorizationFieldId], [LineNumber])
    );
END
GO

/* sec.UserCompanyCode - Company codes a user may work in */
IF OBJECT_ID(N'sec.UserCompanyCode', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[UserCompanyCode]
    (
        [Id]            bigint IDENTITY(1,1) NOT NULL,                                                                -- Surrogate key
        [TenantId]      int NOT NULL,                                                                                 -- Owning tenant - every query is filtered by it
        [UserId]        bigint NOT NULL,                                                                              -- User
        [CompanyCodeId] bigint NOT NULL,                                                                              -- Company code
        [IsDefault]     bit NOT NULL CONSTRAINT [DF_sec_UserCompanyCode_IsDefault] DEFAULT (0),                       -- Default company code
        [AccessLevel]   nvarchar(20) NOT NULL,                                                                        -- Read, Write, Post, Approve
        [ValidFrom]     date NOT NULL,                                                                                -- First day the record is valid
        [ValidTo]       date NOT NULL,                                                                                -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]     datetime2(3) NOT NULL CONSTRAINT [DF_sec_UserCompanyCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]     nvarchar(64) NOT NULL,                                                                        -- Creating user name
        [ModifiedAt]    datetime2(3) NULL,                                                                            -- Last change timestamp (UTC)
        [ModifiedBy]    nvarchar(64) NULL,                                                                            -- Last changing user name
        [RowVersion]    rowversion NOT NULL,                                                                          -- Optimistic concurrency token
        [IsActive]      bit NOT NULL CONSTRAINT [DF_sec_UserCompanyCode_IsActive] DEFAULT (1),                        -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_UserCompanyCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_UserCompanyCode] UNIQUE ([TenantId], [UserId], [CompanyCodeId])
    );
END
GO

/* sec.UserProfile - Personalisation settings of a user */
IF OBJECT_ID(N'sec.UserProfile', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[UserProfile]
    (
        [Id]             bigint IDENTITY(1,1) NOT NULL,                                                               -- Surrogate key
        [TenantId]       int NOT NULL,                                                                                -- Owning tenant - every query is filtered by it
        [UserId]         bigint NOT NULL,                                                                             -- User
        [ParameterId]    nvarchar(40) NOT NULL,                                                                       -- Parameter key, e.g. BUK company code default
        [ParameterValue] nvarchar(255) NOT NULL,                                                                      -- Parameter value
        [Category]       nvarchar(20) NOT NULL,                                                                       -- Default, Layout, Theme, Favorite, Notification
        [CreatedAt]      datetime2(3) NOT NULL CONSTRAINT [DF_sec_UserProfile_CreatedAt] DEFAULT (SYSUTCDATETIME()),  -- Creation timestamp (UTC)
        [CreatedBy]      nvarchar(64) NOT NULL,                                                                       -- Creating user name
        [ModifiedAt]     datetime2(3) NULL,                                                                           -- Last change timestamp (UTC)
        [ModifiedBy]     nvarchar(64) NULL,                                                                           -- Last changing user name
        [RowVersion]     rowversion NOT NULL,                                                                         -- Optimistic concurrency token
        [IsActive]       bit NOT NULL CONSTRAINT [DF_sec_UserProfile_IsActive] DEFAULT (1),                           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_UserProfile] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_UserProfile] UNIQUE ([TenantId], [UserId], [ParameterId])
    );
END
GO

/* sec.UserRole - Role assignment to a user */
IF OBJECT_ID(N'sec.UserRole', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[UserRole]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [UserId]           bigint NOT NULL,                                                                           -- User
        [RoleId]           bigint NOT NULL,                                                                           -- Role
        [AssignedBy]       nvarchar(64) NOT NULL,                                                                     -- Assigning administrator
        [AssignedAt]       datetime2(3) NOT NULL,                                                                     -- Assignment timestamp (UTC)
        [ApprovedBy]       nvarchar(64) NULL,                                                                         -- Approver for critical roles
        [ApprovedAt]       datetime2(3) NULL,                                                                         -- Approval timestamp (UTC)
        [AssignmentReason] nvarchar(255) NULL,                                                                        -- Business justification
        [ValidFrom]        date NOT NULL,                                                                             -- First day the record is valid
        [ValidTo]          date NOT NULL,                                                                             -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]        datetime2(3) NOT NULL CONSTRAINT [DF_sec_UserRole_CreatedAt] DEFAULT (SYSUTCDATETIME()),   -- Creation timestamp (UTC)
        [CreatedBy]        nvarchar(64) NOT NULL,                                                                     -- Creating user name
        [ModifiedAt]       datetime2(3) NULL,                                                                         -- Last change timestamp (UTC)
        [ModifiedBy]       nvarchar(64) NULL,                                                                         -- Last changing user name
        [RowVersion]       rowversion NOT NULL,                                                                       -- Optimistic concurrency token
        [IsActive]         bit NOT NULL CONSTRAINT [DF_sec_UserRole_IsActive] DEFAULT (1),                            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_sec_UserRole] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_UserRole] UNIQUE ([TenantId], [UserId], [RoleId])
    );
END
GO

/* sec.UserSession - Active or ended user session */
IF OBJECT_ID(N'sec.UserSession', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[UserSession]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [SessionId]           uniqueidentifier NOT NULL,                                                              -- Session identifier
        [UserId]              bigint NOT NULL,                                                                        -- User
        [StartedAt]           datetime2(3) NOT NULL,                                                                  -- Session start (UTC)
        [LastActivityAt]      datetime2(3) NOT NULL,                                                                  -- Last request (UTC)
        [ExpiresAt]           datetime2(3) NOT NULL,                                                                  -- Expiry (UTC)
        [EndedAt]             datetime2(3) NULL,                                                                      -- Session end (UTC)
        [EndReason]           nvarchar(20) NULL,                                                                      -- Logout, Timeout, AdminTerminated, TokenRevoked
        [IpAddress]           nvarchar(45) NOT NULL,                                                                  -- Client IP
        [UserAgent]           nvarchar(255) NULL,                                                                     -- Client user agent
        [DeviceType]          nvarchar(20) NULL,                                                                      -- Desktop, Tablet, Mobile, Api
        [ActiveCompanyCodeId] bigint NULL,                                                                            -- Company code selected in the session
        [RefreshTokenHash]    nvarchar(255) NULL,                                                                     -- Hash of the refresh token
        [IsMfaVerified]       bit NOT NULL CONSTRAINT [DF_sec_UserSession_IsMfaVerified] DEFAULT (0),                 -- MFA completed in this session
        CONSTRAINT [PK_sec_UserSession] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_UserSession] UNIQUE ([TenantId], [SessionId])
    );
END
GO

/* sec.UserSubstitution - Substitute who may act for a user during an absence */
IF OBJECT_ID(N'sec.UserSubstitution', N'U') IS NULL
BEGIN
    CREATE TABLE [sec].[UserSubstitution]
    (
        [Id]               bigint IDENTITY(1,1) NOT NULL,                                                             -- Surrogate key
        [TenantId]         int NOT NULL,                                                                              -- Owning tenant - every query is filtered by it
        [UserId]           bigint NOT NULL,                                                                           -- Absent user
        [SubstituteUserId] bigint NOT NULL,                                                                           -- Substitute
        [SubstitutionType] nvarchar(20) NOT NULL,                                                                     -- Approval, FullAccess, ReadOnly
        [ScopeRoleId]      bigint NULL,                                                                               -- Restrict the substitution to one role
        [CompanyCodeId]    bigint NULL,                                                                               -- Restrict to one company code
        [MaximumAmount]    decimal(19,4) NULL,                                                                        -- Approval limit of the substitute
        [CurrencyCode]     nvarchar(5) NULL,                                                                          -- Currency of the limit
        [IsActive]         bit NOT NULL CONSTRAINT [DF_sec_UserSubstitution_IsActive] DEFAULT (1),                    -- Substitution currently effective
        [Reason]           nvarchar(255) NULL,                                                                        -- Reason (leave, travel)
        [ApprovedBy]       nvarchar(64) NULL,                                                                         -- Approver
        [ValidFrom]        date NOT NULL,                                                                             -- First day the record is valid
        [ValidTo]          date NOT NULL,                                                                             -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]        datetime2(3) NOT NULL CONSTRAINT [DF_sec_UserSubstitution_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]        nvarchar(64) NOT NULL,                                                                     -- Creating user name
        [ModifiedAt]       datetime2(3) NULL,                                                                         -- Last change timestamp (UTC)
        [ModifiedBy]       nvarchar(64) NULL,                                                                         -- Last changing user name
        [RowVersion]       rowversion NOT NULL,                                                                       -- Optimistic concurrency token
        CONSTRAINT [PK_sec_UserSubstitution] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_sec_UserSubstitution] UNIQUE ([TenantId], [UserId], [SubstituteUserId])
    );
END
GO
