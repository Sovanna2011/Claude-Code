# 08 — Workflow, Security, Audit (`wf`, `sec`, `audit`)

---

## 8.1 Workflow and approval (`wf`)

### `wf.WorkflowDefinition`
**Approval process definition**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `WorkflowCode` | `nvarchar(20)` | AK | no | Workflow key, e.g. `JE_APPROVAL` |
| `Version` | `int` | AK | no | Definition version — instances pin their version |
| `Name` | `nvarchar(60)` | | no | Description |
| `ObjectType` | `nvarchar(40)` | | no | `JournalEntry`, `VendorInvoice`, `Payment`, `PaymentRun`, `BusinessPartner`, `CustomObjectRequest` |
| `ApprovalMode` | `nvarchar(20)` | | no | `Sequential`, `Parallel`, `MultiLevel` |
| `IsMakerCheckerEnforced` | `bit` | | no | Submitter may not approve |
| `AllowDelegation` | `bit` | | no | Delegation permitted |
| `AllowResubmission` | `bit` | | no | Rejected documents may be resubmitted |
| `EscalationHours` | `int` | | yes | Hours before escalation |
| `ReminderHours` | `int` | | yes | Hours before a reminder is sent |
| `IsActive` | `bit` | | no | Active version |
| `EffectiveFrom` | `date` | | no | First day this version applies |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `wf.WorkflowRule`
**Condition selecting a workflow and its approval level**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `WorkflowDefinitionId` | `bigint` | AK,FK | no | Workflow definition |
| `RuleSequence` | `int` | AK | no | Evaluation order — first match wins |
| `Name` | `nvarchar(60)` | | no | Rule description |
| `CompanyCodeId` | `bigint` | FK | yes | Applies to this company code |
| `DocumentTypeId` | `bigint` | FK | yes | Applies to this document type |
| `DepartmentCode` | `nvarchar(10)` | | yes | Applies to this department |
| `CostCenterId` | `bigint` | FK | yes | Applies to this cost centre |
| `ProfitCenterId` | `bigint` | FK | yes | Applies to this profit centre |
| `CurrencyCode` | `nvarchar(5)` | FK | yes | Currency of the amount limits |
| `MinimumAmount` | `decimal(19,4)` | | yes | Lower amount limit |
| `MaximumAmount` | `decimal(19,4)` | | yes | Upper amount limit |
| `SourceModule` | `nvarchar(10)` | | yes | Applies to this module |
| `ConditionExpression` | `nvarchar(max)` | | yes | Additional condition (expression tree JSON) |
| `IsActive` | `bit` | | no | Rule active |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `wf.WorkflowStep`
**Step of a workflow definition**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `WorkflowDefinitionId` | `bigint` | AK,FK | no | Workflow definition |
| `StepNumber` | `int` | AK | no | Step sequence |
| `Name` | `nvarchar(60)` | | no | Step description |
| `StepType` | `nvarchar(20)` | | no | `Approval`, `Review`, `Notification`, `AutomaticCheck`, `Posting` |
| `ApproverDeterminationType` | `nvarchar(20)` | | no | `Role`, `User`, `CostCenterManager`, `DepartmentHead`, `LineManager`, `Expression` |
| `ApproverRoleId` | `bigint` | FK | yes | Approving role |
| `ApproverUserId` | `bigint` | FK | yes | Named approver |
| `ApproverExpression` | `nvarchar(255)` | | yes | Expression resolving the approver |
| `RequiredApprovals` | `int` | | no | Approvals needed at this step |
| `IsParallel` | `bit` | | no | Approvers work in parallel |
| `IsOptional` | `bit` | | no | Step may be skipped |
| `MinimumAmount` | `decimal(19,4)` | | yes | Step applies from this amount |
| `EscalationRoleId` | `bigint` | FK | yes | Role notified on escalation |
| `EscalationHours` | `int` | | yes | Hours before escalation |
| `OnRejectAction` | `nvarchar(20)` | | no | `ReturnToSubmitter`, `ReturnToPreviousStep`, `Terminate` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `wf.WorkflowInstance`
**Running or finished approval process for one document**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `InstanceNumber` | `nvarchar(20)` | AK | no | Instance number |
| `WorkflowDefinitionId` | `bigint` | FK | no | Definition and version used |
| `ObjectType` | `nvarchar(40)` | IX | no | Object under approval |
| `ObjectId` | `bigint` | IX | no | Object key |
| `ObjectDescription` | `nvarchar(255)` | | yes | Text shown in the inbox |
| `CompanyCodeId` | `bigint` | FK | no | Company code |
| `CurrencyCode` | `nvarchar(5)` | FK | yes | Currency of the amount |
| `Amount` | `decimal(19,4)` | | yes | Amount that drove rule selection |
| `CurrentStepNumber` | `int` | | no | Step currently active |
| `Status` | `nvarchar(20)` | IX | no | `Submitted`, `PendingApproval`, `Approved`, `Rejected`, `Cancelled`, `Escalated`, `Completed` |
| `SubmittedBy` | `nvarchar(64)` | | no | Submitting user |
| `SubmittedAt` | `datetime2(3)` | | no | Submission timestamp (UTC) |
| `CompletedAt` | `datetime2(3)` | | yes | Completion timestamp (UTC) |
| `DueAt` | `datetime2(3)` | | yes | Deadline for the current step |
| `FinalDecision` | `nvarchar(20)` | | yes | `Approved`, `Rejected` |
| `RejectionReason` | `nvarchar(500)` | | yes | Reason given on rejection |
| `ResubmissionCount` | `int` | | no | Number of resubmissions |
| `CorrelationId` | `uniqueidentifier` | | yes | Request correlation id |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `wf.WorkflowTask`
**Individual approval task in an inbox**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `WorkflowInstanceId` | `bigint` | AK,FK | no | Workflow instance |
| `StepNumber` | `int` | AK | no | Step |
| `AssignedUserId` | `bigint` | AK,FK | no | Assigned approver |
| `AssignedRoleId` | `bigint` | FK | yes | Role the task was routed through |
| `Status` | `nvarchar(20)` | IX | no | `Pending`, `Approved`, `Rejected`, `Delegated`, `Escalated`, `Withdrawn`, `Expired` |
| `AssignedAt` | `datetime2(3)` | | no | Assignment timestamp (UTC) |
| `DueAt` | `datetime2(3)` | | yes | Due timestamp (UTC) |
| `DecidedAt` | `datetime2(3)` | | yes | Decision timestamp (UTC) |
| `DecidedByUserId` | `bigint` | FK | yes | User who decided (may be a substitute) |
| `Decision` | `nvarchar(20)` | | yes | `Approve`, `Reject`, `RequestInformation` |
| `Comment` | `nvarchar(1000)` | | yes | Approver comment |
| `DelegatedToUserId` | `bigint` | FK | yes | Delegate |
| `DelegationReason` | `nvarchar(255)` | | yes | Reason for delegation |
| `ReminderCount` | `int` | | no | Reminders sent |
| `IsReadOnly` | `bit` | | no | Informational task |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `wf.WorkflowHistory`
**Immutable event log of a workflow instance**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `WorkflowInstanceId` | `bigint` | FK,IX | no | Workflow instance |
| `EventSequence` | `int` | | no | Event order |
| `EventType` | `nvarchar(30)` | | no | `Submitted`, `Assigned`, `Approved`, `Rejected`, `Delegated`, `Escalated`, `Reminded`, `Cancelled`, `Resubmitted`, `Completed` |
| `StepNumber` | `int` | | yes | Step concerned |
| `PerformedByUserId` | `bigint` | FK | yes | Acting user |
| `PerformedAt` | `datetime2(3)` | | no | Event timestamp (UTC) |
| `FromStatus` | `nvarchar(20)` | | yes | Previous status |
| `ToStatus` | `nvarchar(20)` | | yes | New status |
| `Comment` | `nvarchar(1000)` | | yes | Comment |
| `CorrelationId` | `uniqueidentifier` | | yes | Request correlation id |

---

### `wf.Notification`
**Notification queued or delivered to a user**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RecipientUserId` | `bigint` | FK,IX | no | Recipient |
| `NotificationType` | `nvarchar(30)` | | no | `ApprovalRequest`, `Reminder`, `Escalation`, `Decision`, `SystemAlert` |
| `Channel` | `nvarchar(20)` | | no | `InApp`, `Email`, `Push`, `SignalR` |
| `Subject` | `nvarchar(255)` | | no | Subject |
| `Body` | `nvarchar(max)` | | yes | Message body |
| `ObjectType` | `nvarchar(40)` | | yes | Related object type |
| `ObjectId` | `bigint` | | yes | Related object key |
| `WorkflowTaskId` | `bigint` | FK | yes | Related approval task |
| `NavigationUrl` | `nvarchar(500)` | | yes | Deep link into the application |
| `Priority` | `nvarchar(10)` | | no | `Low`, `Normal`, `High` |
| `Status` | `nvarchar(20)` | IX | no | `Queued`, `Sent`, `Delivered`, `Failed`, `Read` |
| `SentAt` | `datetime2(3)` | | yes | Send timestamp (UTC) |
| `ReadAt` | `datetime2(3)` | | yes | Read timestamp (UTC) |
| `RetryCount` | `int` | | no | Delivery attempts |
| `ErrorMessage` | `nvarchar(500)` | | yes | Last delivery error |

---

## 8.2 Users and security (`sec`)

### `sec.User`
**Application user**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UserName` | `nvarchar(64)` | AK | no | Logon name |
| `EmployeeNumber` | `nvarchar(20)` | | yes | Personnel number |
| `BusinessPartnerId` | `bigint` | FK | yes | Business partner (employee role) |
| `FirstName` | `nvarchar(40)` | | yes | First name |
| `LastName` | `nvarchar(40)` | | yes | Last name |
| `DisplayName` | `nvarchar(100)` | | no | Display name |
| `Email` | `nvarchar(255)` | IX | no | Email address |
| `MobileNumber` | `nvarchar(30)` | | yes | Mobile number |
| `UserType` | `nvarchar(20)` | | no | `Dialog`, `Service`, `Integration`, `Api`, `Background`, `Auditor` |
| `PasswordHash` | `nvarchar(255)` | | yes | Password hash (never logged) |
| `PasswordSalt` | `nvarchar(64)` | | yes | Password salt |
| `PasswordChangedAt` | `datetime2(3)` | | yes | Last password change (UTC) |
| `MustChangePassword` | `bit` | | no | Password change forced at next logon |
| `ExternalIdentityProvider` | `nvarchar(40)` | | yes | OIDC provider name |
| `ExternalSubjectId` | `nvarchar(255)` | | yes | Subject id at the provider |
| `IsMfaEnabled` | `bit` | | no | Multi-factor authentication active |
| `MfaSecretReference` | `nvarchar(255)` | | yes | Reference to the MFA secret in the secret store |
| `LanguageCode` | `nvarchar(2)` | FK | no | UI language |
| `TimeZoneId` | `nvarchar(64)` | | no | Display time zone |
| `DateFormat` | `nvarchar(10)` | | yes | Preferred date format |
| `DecimalNotation` | `nvarchar(10)` | | yes | Preferred decimal notation |
| `DefaultCompanyCodeId` | `bigint` | FK | yes | Default company code |
| `DefaultControllingAreaId` | `bigint` | FK | yes | Default controlling area |
| `DefaultCurrencyCode` | `nvarchar(5)` | FK | yes | Default display currency |
| `DefaultTransactionCode` | `nvarchar(20)` | | yes | Start transaction |
| `Status` | `nvarchar(20)` | IX | no | `Active`, `Locked`, `Expired`, `Disabled` |
| `IsLocked` | `bit` | | no | Administratively locked |
| `LockReason` | `nvarchar(255)` | | yes | Lock reason |
| `FailedLoginCount` | `int` | | no | Consecutive failed logons |
| `LastFailedLoginAt` | `datetime2(3)` | | yes | Last failed logon (UTC) |
| `LastLoginAt` | `datetime2(3)` | | yes | Last successful logon (UTC) |
| `LastLoginIpAddress` | `nvarchar(45)` | | yes | Last logon origin |
| `ValidFrom` | `date` | | no | Account valid from |
| `ValidTo` | `date` | | no | Account valid to |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.UserProfile`
**Personalisation settings of a user**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UserId` | `bigint` | AK,FK | no | User |
| `ParameterId` | `nvarchar(40)` | AK | no | Parameter key, e.g. `BUK` company code default |
| `ParameterValue` | `nvarchar(255)` | | no | Parameter value |
| `Category` | `nvarchar(20)` | | no | `Default`, `Layout`, `Theme`, `Favorite`, `Notification` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.UserCompanyCode`
**Company codes a user may work in**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UserId` | `bigint` | AK,FK | no | User |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `IsDefault` | `bit` | | no | Default company code |
| `AccessLevel` | `nvarchar(20)` | | no | `Read`, `Write`, `Post`, `Approve` |
| *include* | `#VALIDITY` | | | Assignment validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.Role`
**Role (collection of permissions and authorizations)**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RoleCode` | `nvarchar(40)` | AK | no | Role key, e.g. `FI_ACCOUNTANT` |
| `Name` | `nvarchar(60)` | | no | Role name |
| `Description` | `nvarchar(255)` | | yes | Description |
| `RoleType` | `nvarchar(20)` | | no | `Single`, `Composite`, `Derived` |
| `ParentRoleId` | `bigint` | FK | yes | Reference role for derived roles |
| `IsCriticalRole` | `bit` | | no | Requires extra approval to assign |
| `RequiresApprovalToAssign` | `bit` | | no | Assignment goes through workflow |
| `IsSystemRole` | `bit` | | no | Delivered role — cannot be deleted |
| *include* | `#VALIDITY` | | | Role validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.Permission`
**Atomic permission**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PermissionCode` | `nvarchar(60)` | AK | no | Permission key, e.g. `Finance.JournalEntry.Post` |
| `Name` | `nvarchar(60)` | | no | Permission name |
| `Module` | `nvarchar(20)` | | no | Owning module |
| `ObjectType` | `nvarchar(40)` | | no | Protected object |
| `Action` | `nvarchar(20)` | | no | `Create`, `Read`, `Update`, `Delete`, `Post`, `Reverse`, `Approve`, `Export`, `Execute` |
| `IsCritical` | `bit` | | no | Critical permission — always audited |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.UserRole`
**Role assignment to a user**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UserId` | `bigint` | AK,FK | no | User |
| `RoleId` | `bigint` | AK,FK | no | Role |
| `AssignedBy` | `nvarchar(64)` | | no | Assigning administrator |
| `AssignedAt` | `datetime2(3)` | | no | Assignment timestamp (UTC) |
| `ApprovedBy` | `nvarchar(64)` | | yes | Approver for critical roles |
| `ApprovedAt` | `datetime2(3)` | | yes | Approval timestamp (UTC) |
| `AssignmentReason` | `nvarchar(255)` | | yes | Business justification |
| *include* | `#VALIDITY` | | | Assignment validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.RolePermission`
**Permission granted by a role**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RoleId` | `bigint` | AK,FK | no | Role |
| `PermissionId` | `bigint` | AK,FK | no | Permission |
| `IsGranted` | `bit` | | no | Granted (`1`) or explicitly denied (`0`) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.TransactionCode`
**T-code navigation alias** · reference: `TSTC`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `TransactionCode` | `nvarchar(20)` | AK | no | T-code, e.g. `FB50`, `SE16N` |
| `Name` | `nvarchar(60)` | | no | Description |
| `Module` | `nvarchar(20)` | | no | Owning module |
| `RoutePath` | `nvarchar(255)` | | no | Frontend route, e.g. `/finance/journal-entry/new` |
| `RequiredPermissionId` | `bigint` | FK | yes | Permission checked before navigation |
| `AuthorizationObjectId` | `bigint` | FK | yes | Authorization object checked |
| `IconName` | `nvarchar(40)` | | yes | Icon |
| `Category` | `nvarchar(40)` | | yes | Menu category |
| `IsFavoriteEligible` | `bit` | | no | May be added to favourites |
| `IsBlocked` | `bit` | | no | Blocked for all users |
| `HelpText` | `nvarchar(500)` | | yes | Help text |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.RoleTransactionCode`
**T-codes a role may start**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RoleId` | `bigint` | AK,FK | no | Role |
| `TransactionCodeId` | `bigint` | AK,FK | no | Transaction code |
| `IsGranted` | `bit` | | no | Granted or denied |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.AuthorizationObject`
**Authorization object (a set of fields checked together)** · reference: `TOBJ`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ObjectCode` | `nvarchar(20)` | AK | no | Object key, e.g. `F_BKPF_BUK` |
| `Name` | `nvarchar(60)` | | no | Description |
| `ObjectClass` | `nvarchar(20)` | | no | Class, e.g. `FI`, `CO`, `BC` |
| `Description` | `nvarchar(255)` | | yes | Purpose of the object |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.AuthorizationField`
**Field of an authorization object** · reference: `TOBJT`/`AUTHX`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AuthorizationObjectId` | `bigint` | AK,FK | no | Authorization object |
| `FieldName` | `nvarchar(20)` | AK | no | Field, e.g. `BUKRS`, `ACTVT`, `KOSTL` |
| `Name` | `nvarchar(60)` | | no | Description |
| `DataElementId` | `bigint` | FK | yes | Data element defining the value domain |
| `CheckTableName` | `nvarchar(64)` | | yes | Table of permitted values |
| `IsOrganizationalLevel` | `bit` | | no | Organisational level (derived roles fill it) |
| `FieldPosition` | `int` | | no | Order |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.AuthorizationValue`
**Value or range granted for an authorization field in a role**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RoleId` | `bigint` | AK,FK | no | Role |
| `AuthorizationObjectId` | `bigint` | AK,FK | no | Authorization object |
| `AuthorizationFieldId` | `bigint` | AK,FK | no | Field |
| `LineNumber` | `int` | AK | no | Value line |
| `SignIndicator` | `nvarchar(1)` | | no | `I` include, `E` exclude |
| `Operator` | `nvarchar(10)` | | no | `EQ`, `BT`, `CP`, `NE` |
| `LowValue` | `nvarchar(60)` | | no | Value or interval start (`*` = all) |
| `HighValue` | `nvarchar(60)` | | yes | Interval end |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.UserAuthorization`
**Authorization granted directly to a user (exception to role-based access)**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UserId` | `bigint` | AK,FK | no | User |
| `AuthorizationObjectId` | `bigint` | AK,FK | no | Authorization object |
| `AuthorizationFieldId` | `bigint` | AK,FK | no | Field |
| `LineNumber` | `int` | AK | no | Value line |
| `SignIndicator` | `nvarchar(1)` | | no | `I` include, `E` exclude |
| `LowValue` | `nvarchar(60)` | | no | Value or interval start |
| `HighValue` | `nvarchar(60)` | | yes | Interval end |
| `GrantReason` | `nvarchar(255)` | | no | Justification — required for direct grants |
| `ApprovedBy` | `nvarchar(64)` | | no | Approver |
| `ApprovedAt` | `datetime2(3)` | | no | Approval timestamp (UTC) |
| *include* | `#VALIDITY` | | | Grant validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.UserSession`
**Active or ended user session**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SessionId` | `uniqueidentifier` | AK | no | Session identifier |
| `UserId` | `bigint` | FK,IX | no | User |
| `StartedAt` | `datetime2(3)` | IX | no | Session start (UTC) |
| `LastActivityAt` | `datetime2(3)` | | no | Last request (UTC) |
| `ExpiresAt` | `datetime2(3)` | | no | Expiry (UTC) |
| `EndedAt` | `datetime2(3)` | | yes | Session end (UTC) |
| `EndReason` | `nvarchar(20)` | | yes | `Logout`, `Timeout`, `AdminTerminated`, `TokenRevoked` |
| `IpAddress` | `nvarchar(45)` | | no | Client IP |
| `UserAgent` | `nvarchar(255)` | | yes | Client user agent |
| `DeviceType` | `nvarchar(20)` | | yes | `Desktop`, `Tablet`, `Mobile`, `Api` |
| `ActiveCompanyCodeId` | `bigint` | FK | yes | Company code selected in the session |
| `RefreshTokenHash` | `nvarchar(255)` | | yes | Hash of the refresh token |
| `IsMfaVerified` | `bit` | | no | MFA completed in this session |

---

### `sec.LoginHistory`
**Logon attempt log**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UserName` | `nvarchar(64)` | IX | no | Logon name used (kept even if unknown) |
| `UserId` | `bigint` | FK | yes | User, when resolved |
| `AttemptedAt` | `datetime2(3)` | IX | no | Attempt timestamp (UTC) |
| `IsSuccessful` | `bit` | | no | Logon succeeded |
| `FailureReason` | `nvarchar(40)` | | yes | `InvalidPassword`, `UserLocked`, `UserExpired`, `MfaFailed`, `UnknownUser` |
| `AuthenticationMethod` | `nvarchar(20)` | | no | `Password`, `Oidc`, `ApiKey`, `Certificate` |
| `IpAddress` | `nvarchar(45)` | | no | Client IP |
| `UserAgent` | `nvarchar(255)` | | yes | Client user agent |
| `SessionId` | `uniqueidentifier` | | yes | Session created |
| `CorrelationId` | `uniqueidentifier` | | yes | Request correlation id |

---

### `sec.PasswordHistory`
**Previous password hashes, to enforce reuse rules**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UserId` | `bigint` | AK,FK | no | User |
| `ChangedAt` | `datetime2(3)` | AK | no | Change timestamp (UTC) |
| `PasswordHash` | `nvarchar(255)` | | no | Historic hash (never logged, never exported) |
| `PasswordSalt` | `nvarchar(64)` | | no | Historic salt |
| `ChangedBy` | `nvarchar(64)` | | no | User or administrator who changed it |
| `ChangeReason` | `nvarchar(40)` | | no | `UserChange`, `AdminReset`, `Expired`, `ForcedPolicy` |

---

### `sec.UserSubstitution`
**Substitute who may act for a user during an absence**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `UserId` | `bigint` | AK,FK | no | Absent user |
| `SubstituteUserId` | `bigint` | AK,FK | no | Substitute |
| `SubstitutionType` | `nvarchar(20)` | | no | `Approval`, `FullAccess`, `ReadOnly` |
| `ScopeRoleId` | `bigint` | FK | yes | Restrict the substitution to one role |
| `CompanyCodeId` | `bigint` | FK | yes | Restrict to one company code |
| `MaximumAmount` | `decimal(19,4)` | | yes | Approval limit of the substitute |
| `CurrencyCode` | `nvarchar(5)` | FK | yes | Currency of the limit |
| `IsActive` | `bit` | | no | Substitution currently effective |
| `Reason` | `nvarchar(255)` | | yes | Reason (leave, travel) |
| `ApprovedBy` | `nvarchar(64)` | | yes | Approver |
| *include* | `#VALIDITY` | | | Substitution period |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.SegregationOfDutiesRule`
**Conflicting duty combination**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RuleCode` | `nvarchar(20)` | AK | no | Rule key, e.g. `SOD_AP_01` |
| `Name` | `nvarchar(60)` | | no | Rule name |
| `Description` | `nvarchar(500)` | | yes | Risk described by the rule |
| `RiskLevel` | `nvarchar(10)` | | no | `Low`, `Medium`, `High`, `Critical` |
| `ConflictType` | `nvarchar(20)` | | no | `PermissionPair`, `RolePair`, `TransactionPair` |
| `FirstObjectCode` | `nvarchar(60)` | | no | First conflicting permission / role / T-code |
| `SecondObjectCode` | `nvarchar(60)` | | no | Second conflicting object |
| `EnforcementMode` | `nvarchar(20)` | | no | `Block`, `WarnAndLog`, `LogOnly` |
| `MitigationControl` | `nvarchar(500)` | | yes | Compensating control if the conflict is accepted |
| `IsActive` | `bit` | | no | Rule active |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `sec.SegregationOfDutiesViolation`
**Detected or accepted conflict for a user**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SegregationOfDutiesRuleId` | `bigint` | AK,FK | no | Rule violated |
| `UserId` | `bigint` | AK,FK | no | User concerned |
| `DetectedAt` | `datetime2(3)` | | no | Detection timestamp (UTC) |
| `Status` | `nvarchar(20)` | | no | `Open`, `Mitigated`, `Accepted`, `Resolved` |
| `MitigationNote` | `nvarchar(500)` | | yes | Mitigation applied |
| `AcceptedBy` | `nvarchar(64)` | | yes | Approver of the exception |
| `AcceptedAt` | `datetime2(3)` | | yes | Acceptance timestamp (UTC) |
| `ReviewDueDate` | `date` | | yes | Next review |
| `ResolvedAt` | `datetime2(3)` | | yes | Resolution timestamp (UTC) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 8.3 Audit (`audit`)

### `audit.AuditLog`
**Business audit trail — one row per auditable action**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `OccurredAt` | `datetime2(3)` | IX | no | Action timestamp (UTC) |
| `UserName` | `nvarchar(64)` | IX | no | Acting user |
| `UserId` | `bigint` | FK | yes | User |
| `CompanyCodeId` | `bigint` | FK,IX | yes | Company code context |
| `Action` | `nvarchar(40)` | IX | no | `Create`, `Change`, `Delete`, `Post`, `Reverse`, `Approve`, `Reject`, `Export`, `Login`, `ConfigChange` |
| `ObjectType` | `nvarchar(40)` | IX | no | Object type acted on |
| `ObjectId` | `bigint` | | yes | Object key |
| `ObjectKeyText` | `nvarchar(255)` | | yes | Readable key, e.g. document number |
| `SourceType` | `nvarchar(20)` | | no | `Screen`, `Api`, `BackgroundJob`, `Import`, `System` |
| `SourceName` | `nvarchar(100)` | | yes | Screen, endpoint or job name |
| `TransactionCode` | `nvarchar(20)` | | yes | T-code used |
| `Result` | `nvarchar(20)` | | no | `Success`, `Failure`, `Denied` |
| `MessageText` | `nvarchar(500)` | | yes | Result or error message |
| `CorrelationId` | `uniqueidentifier` | IX | yes | Request correlation id |
| `SessionId` | `uniqueidentifier` | | yes | Session |
| `IpAddress` | `nvarchar(45)` | | yes | Request origin |
| `UserAgent` | `nvarchar(255)` | | yes | Client user agent |
| `IsSensitiveAction` | `bit` | | no | Critical action — extended retention |
| `HashChainValue` | `nvarchar(64)` | | no | SHA-256 over the row and the previous hash — tamper detection |
| `PreviousHashValue` | `nvarchar(64)` | | yes | Hash of the previous audit row |

---

### `audit.ChangeDocumentHeader`
**Change document for configuration and master data** · reference: `CDHDR`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ChangeDocumentNumber` | `nvarchar(20)` | AK | no | Change document number |
| `ObjectClass` | `nvarchar(40)` | IX | no | Object class, e.g. `BUSINESS_PARTNER`, `GL_ACCOUNT`, `COST_CENTER` |
| `ObjectId` | `bigint` | IX | no | Changed object |
| `ObjectKeyText` | `nvarchar(255)` | | no | Readable object key |
| `ChangedAt` | `datetime2(3)` | IX | no | Change timestamp (UTC) |
| `ChangedBy` | `nvarchar(64)` | IX | no | Changing user |
| `ChangeType` | `nvarchar(1)` | | no | `I` insert, `U` update, `D` delete |
| `TransactionCode` | `nvarchar(20)` | | yes | T-code used |
| `SourceType` | `nvarchar(20)` | | no | `Screen`, `Api`, `Import`, `Job` |
| `CorrelationId` | `uniqueidentifier` | | yes | Request correlation id |
| `PlannedChangeDate` | `date` | | yes | Effective date for planned changes |

---

### `audit.ChangeDocumentItem`
**Field-level before/after values** · reference: `CDPOS`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ChangeDocumentHeaderId` | `bigint` | AK,FK | no | Change document |
| `ItemNumber` | `int` | AK | no | Item sequence |
| `TableName` | `nvarchar(64)` | | no | Changed table |
| `TableKeyText` | `nvarchar(255)` | | no | Key of the changed row |
| `FieldName` | `nvarchar(64)` | | no | Changed field |
| `ChangeIndicator` | `nvarchar(1)` | | no | `I` insert, `U` update, `D` delete |
| `OldValue` | `nvarchar(500)` | | yes | Value before the change (masked if sensitive) |
| `NewValue` | `nvarchar(500)` | | yes | Value after the change (masked if sensitive) |
| `IsSensitiveField` | `bit` | | no | Sensitive field — values masked |
| `TextValueChanged` | `bit` | | no | Long text changed (values not stored inline) |

---

### `audit.DataAccessLog`
**Read access to personal or restricted data**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `AccessedAt` | `datetime2(3)` | IX | no | Access timestamp (UTC) |
| `UserId` | `bigint` | FK,IX | no | Accessing user |
| `ObjectType` | `nvarchar(40)` | | no | Object type read |
| `ObjectId` | `bigint` | | yes | Object key |
| `AccessType` | `nvarchar(20)` | | no | `Display`, `Search`, `Report`, `Export`, `Api` |
| `FieldsAccessed` | `nvarchar(500)` | | yes | Sensitive fields returned |
| `RecordCount` | `int` | | no | Number of records returned |
| `Purpose` | `nvarchar(255)` | | yes | Stated purpose, when required |
| `CorrelationId` | `uniqueidentifier` | | yes | Request correlation id |
| `IpAddress` | `nvarchar(45)` | | yes | Request origin |

---

### `audit.RetentionPolicy`
**Retention and archiving rule per audit or business object**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ObjectType` | `nvarchar(40)` | AK | no | Object the policy applies to |
| `RetentionYears` | `int` | | no | Years the data must be kept |
| `ArchiveAfterYears` | `int` | | yes | Years before the data is archived |
| `LegalBasis` | `nvarchar(255)` | | yes | Legal or regulatory basis |
| `DeletionMode` | `nvarchar(20)` | | no | `Block`, `Anonymize`, `Delete`, `ArchiveOnly` |
| `IsActive` | `bit` | | no | Policy active |
| `LastExecutedAt` | `datetime2(3)` | | yes | Last execution (UTC) |
| *include* | `#AUDIT` | | | Standard audit columns |
