# 09 — Reporting and Integration (`rpt`, `intg`)

Reporting stores **definitions, variants and layouts** only — figures always
come from `fin.JournalEntryLine`, `fin.AccountBalance` and `co.ControllingTotal`
so no report can drift from the universal journal.

Integration uses a **transactional outbox**: business events are written in the
same transaction as the posting, then published by a background dispatcher.

---

## 9.1 Reporting (`rpt`)

### `rpt.ReportDefinition`
**Registered report**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ReportCode` | `nvarchar(40)` | AK | no | Report key, e.g. `FI_TRIAL_BALANCE` |
| `Name` | `nvarchar(60)` | | no | Report name |
| `Description` | `nvarchar(255)` | | yes | Description |
| `Module` | `nvarchar(20)` | | no | Owning module |
| `ReportCategory` | `nvarchar(30)` | | no | `GeneralLedger`, `Receivables`, `Payables`, `Assets`, `Controlling`, `Tax`, `Audit`, `Dashboard` |
| `DataSourceView` | `nvarchar(64)` | | no | View or read model queried |
| `RequiredPermissionId` | `bigint` | FK | yes | Permission required to run |
| `TransactionCode` | `nvarchar(20)` | | yes | T-code alias |
| `SupportsDrillDown` | `bit` | | no | Drill-down to line items |
| `DrillDownTarget` | `nvarchar(40)` | | yes | Target object of the drill-down |
| `DefaultCurrencyType` | `nvarchar(2)` | | yes | `10` local, `30` group, `00` document |
| `MaxRows` | `int` | | no | Row cap |
| `IsExportAllowed` | `bit` | | no | Export permitted |
| `IsScheduleAllowed` | `bit` | | no | May be scheduled as a background job |
| `IsActive` | `bit` | | no | Active |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `rpt.ReportParameter`
**Selection parameter of a report**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ReportDefinitionId` | `bigint` | AK,FK | no | Report |
| `ParameterName` | `nvarchar(64)` | AK | no | Parameter name |
| `Label` | `nvarchar(60)` | | no | Screen label |
| `DataElementId` | `bigint` | FK | yes | Data element |
| `SqlType` | `nvarchar(40)` | | no | Physical type |
| `IsRequired` | `bit` | | no | Mandatory |
| `IsRange` | `bit` | | no | Range (from / to) parameter |
| `IsMultiSelect` | `bit` | | no | Multiple values allowed |
| `DefaultValue` | `nvarchar(255)` | | yes | Default |
| `SearchHelpId` | `bigint` | FK | yes | Search help |
| `DisplayOrder` | `int` | | no | Order on the selection screen |
| `IsAuthorizationRelevant` | `bit` | | no | Value restricted by the user's authorizations |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `rpt.ReportVariant`
**Saved selection values for a report**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ReportDefinitionId` | `bigint` | AK,FK | no | Report |
| `VariantName` | `nvarchar(40)` | AK | no | Variant name |
| `OwnerUserId` | `bigint` | AK,FK | no | Owner |
| `Description` | `nvarchar(255)` | | yes | Description |
| `ParameterValuesJson` | `nvarchar(max)` | | no | Stored parameter values |
| `IsShared` | `bit` | | no | Visible to other users |
| `IsDefault` | `bit` | | no | Loaded by default |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `rpt.ReportLayout`
**Column layout of a report or list**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ReportDefinitionId` | `bigint` | AK,FK | no | Report |
| `LayoutName` | `nvarchar(40)` | AK | no | Layout name |
| `OwnerUserId` | `bigint` | AK,FK | no | Owner |
| `ColumnDefinitionJson` | `nvarchar(max)` | | no | Columns, order, width, visibility |
| `SortDefinition` | `nvarchar(255)` | | yes | Sort fields |
| `GroupByFields` | `nvarchar(255)` | | yes | Grouping fields |
| `SubtotalFields` | `nvarchar(255)` | | yes | Subtotal fields |
| `IsShared` | `bit` | | no | Visible to other users |
| `IsDefault` | `bit` | | no | Default layout |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `rpt.ReportExecutionLog`
**Execution record for auditing and performance analysis**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ReportDefinitionId` | `bigint` | FK,IX | no | Report |
| `UserId` | `bigint` | FK,IX | no | Executing user |
| `ExecutedAt` | `datetime2(3)` | IX | no | Execution timestamp (UTC) |
| `ParameterValuesJson` | `nvarchar(max)` | | yes | Parameters used |
| `RowCount` | `int` | | no | Rows returned |
| `DurationMs` | `int` | | no | Duration in milliseconds |
| `OutputFormat` | `nvarchar(10)` | | yes | `Screen`, `XLSX`, `PDF`, `CSV` |
| `WasExported` | `bit` | | no | Result exported |
| `WasTruncated` | `bit` | | no | Row cap reached |
| `Status` | `nvarchar(20)` | | no | `Success`, `Failed`, `Cancelled`, `Timeout` |
| `ErrorMessage` | `nvarchar(500)` | | yes | Error message |
| `CorrelationId` | `uniqueidentifier` | | yes | Request correlation id |

---

## 9.2 Integration and API management (`intg`)

### `intg.ApiClient`
**Registered API consumer**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ClientCode` | `nvarchar(40)` | AK | no | Client key |
| `Name` | `nvarchar(60)` | | no | Client name |
| `Description` | `nvarchar(255)` | | yes | Purpose of the integration |
| `ClientType` | `nvarchar(20)` | | no | `Machine`, `Partner`, `Internal`, `Mobile` |
| `ServiceUserId` | `bigint` | FK | no | Service user the client acts as |
| `ApiKeyHash` | `nvarchar(255)` | | yes | Hash of the API key (never stored in clear) |
| `ApiKeyLastRotatedAt` | `datetime2(3)` | | yes | Last key rotation (UTC) |
| `OAuthClientId` | `nvarchar(100)` | | yes | OAuth / OIDC client id |
| `AllowedScopes` | `nvarchar(500)` | | yes | Granted scopes |
| `AllowedIpRanges` | `nvarchar(500)` | | yes | Permitted source IP ranges |
| `RateLimitPerMinute` | `int` | | no | Requests allowed per minute |
| `IsActive` | `bit` | | no | Client active |
| `ContactEmail` | `nvarchar(255)` | | yes | Technical contact |
| *include* | `#VALIDITY` | | | Client validity |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `intg.IdempotencyKey`
**Recorded idempotency key and its stored response**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `IdempotencyKey` | `uniqueidentifier` | AK | no | Key supplied by the caller |
| `Endpoint` | `nvarchar(255)` | AK | no | Endpoint the key applies to |
| `RequestHash` | `nvarchar(64)` | | no | SHA-256 of the request body — mismatch is rejected |
| `Status` | `nvarchar(20)` | | no | `InProgress`, `Completed`, `Failed` |
| `ResponseStatusCode` | `int` | | yes | Stored HTTP status |
| `ResponseBody` | `nvarchar(max)` | | yes | Stored response |
| `ResultObjectType` | `nvarchar(40)` | | yes | Object created by the request |
| `ResultObjectId` | `bigint` | | yes | Key of the created object |
| `CreatedAt` | `datetime2(3)` | | no | First seen (UTC) |
| `CompletedAt` | `datetime2(3)` | | yes | Completion timestamp (UTC) |
| `ExpiresAt` | `datetime2(3)` | IX | no | Expiry of the key |
| `ApiClientId` | `bigint` | FK | yes | Calling client |

---

### `intg.OutboxMessage`
**Transactional outbox — events written with the business transaction**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `MessageId` | `uniqueidentifier` | AK | no | Message identifier |
| `EventType` | `nvarchar(80)` | IX | no | e.g. `JournalEntryPosted`, `BusinessPartnerChanged` |
| `AggregateType` | `nvarchar(40)` | | no | Source aggregate |
| `AggregateId` | `bigint` | | no | Source key |
| `PayloadJson` | `nvarchar(max)` | | no | Event payload |
| `Headers` | `nvarchar(max)` | | yes | Message headers |
| `OccurredAt` | `datetime2(3)` | | no | Business event timestamp (UTC) |
| `Status` | `nvarchar(20)` | IX | no | `Pending`, `Publishing`, `Published`, `Failed`, `DeadLettered` |
| `PublishedAt` | `datetime2(3)` | | yes | Publication timestamp (UTC) |
| `AttemptCount` | `int` | | no | Delivery attempts |
| `NextAttemptAt` | `datetime2(3)` | IX | yes | Next retry (UTC) |
| `LastError` | `nvarchar(500)` | | yes | Last error |
| `CorrelationId` | `uniqueidentifier` | | yes | Originating request |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user or job |

---

### `intg.InboundMessage`
**Received message, stored before processing**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `MessageId` | `uniqueidentifier` | AK | no | Message identifier |
| `SourceSystem` | `nvarchar(40)` | IX | no | Sending system |
| `MessageType` | `nvarchar(80)` | | no | Message type |
| `PayloadJson` | `nvarchar(max)` | | no | Raw payload |
| `ReceivedAt` | `datetime2(3)` | IX | no | Receipt timestamp (UTC) |
| `Status` | `nvarchar(20)` | IX | no | `Received`, `Processing`, `Processed`, `Failed`, `Rejected`, `Duplicate` |
| `ProcessedAt` | `datetime2(3)` | | yes | Processing timestamp (UTC) |
| `AttemptCount` | `int` | | no | Processing attempts |
| `ResultObjectType` | `nvarchar(40)` | | yes | Object created |
| `ResultObjectId` | `bigint` | | yes | Key of the created object |
| `ErrorMessage` | `nvarchar(500)` | | yes | Error message |
| `ApiClientId` | `bigint` | FK | yes | Sending client |
| `CorrelationId` | `uniqueidentifier` | | yes | Correlation id |

---

### `intg.WebhookSubscription`
**Subscription of an external system to events**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `SubscriptionCode` | `nvarchar(40)` | AK | no | Subscription key |
| `ApiClientId` | `bigint` | FK | no | Subscribing client |
| `EventTypes` | `nvarchar(500)` | | no | Subscribed event types |
| `TargetUrl` | `nvarchar(500)` | | no | Delivery endpoint |
| `SecretReference` | `nvarchar(255)` | | yes | Reference to the signing secret in the secret store |
| `SignatureAlgorithm` | `nvarchar(20)` | | no | `HMACSHA256` |
| `FilterExpression` | `nvarchar(500)` | | yes | Additional filter on the payload |
| `MaxRetries` | `int` | | no | Maximum delivery attempts |
| `TimeoutSeconds` | `int` | | no | Request timeout |
| `IsActive` | `bit` | | no | Subscription active |
| `LastDeliveryAt` | `datetime2(3)` | | yes | Last successful delivery (UTC) |
| `ConsecutiveFailureCount` | `int` | | no | Consecutive failures — disables at the threshold |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `intg.WebhookDelivery`
**Delivery attempt of a webhook**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `WebhookSubscriptionId` | `bigint` | FK,IX | no | Subscription |
| `OutboxMessageId` | `bigint` | FK | no | Message delivered |
| `AttemptNumber` | `int` | | no | Attempt number |
| `AttemptedAt` | `datetime2(3)` | IX | no | Attempt timestamp (UTC) |
| `ResponseStatusCode` | `int` | | yes | HTTP status returned |
| `ResponseBody` | `nvarchar(1000)` | | yes | Truncated response body |
| `DurationMs` | `int` | | yes | Duration in milliseconds |
| `IsSuccessful` | `bit` | | no | Delivery accepted |
| `ErrorMessage` | `nvarchar(500)` | | yes | Error message |

---

### `intg.ImportJob`
**CSV / Excel import run**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `JobNumber` | `nvarchar(20)` | AK | no | Job number |
| `ImportType` | `nvarchar(40)` | | no | `BusinessPartner`, `GLAccount`, `JournalEntry`, `ExchangeRate`, `AssetLegacy`, `OpeningBalance` |
| `FileName` | `nvarchar(255)` | | no | Uploaded file name |
| `FileUri` | `nvarchar(500)` | | no | Stored file location |
| `Checksum` | `nvarchar(64)` | | no | SHA-256 of the file |
| `CompanyCodeId` | `bigint` | FK | yes | Target company code |
| `Status` | `nvarchar(20)` | IX | no | `Uploaded`, `Validating`, `Validated`, `Importing`, `Completed`, `CompletedWithErrors`, `Failed`, `Cancelled` |
| `IsTestRun` | `bit` | | no | Validation only |
| `TotalRowCount` | `int` | | yes | Rows in the file |
| `SuccessRowCount` | `int` | | yes | Rows imported |
| `ErrorRowCount` | `int` | | yes | Rows rejected |
| `StartedAt` | `datetime2(3)` | | yes | Start (UTC) |
| `CompletedAt` | `datetime2(3)` | | yes | Completion (UTC) |
| `ExecutedBy` | `nvarchar(64)` | | no | Executing user |
| `CorrelationId` | `uniqueidentifier` | | yes | Correlation id |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `intg.ImportJobError`
**Rejected row of an import job**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ImportJobId` | `bigint` | AK,FK | no | Import job |
| `RowNumber` | `int` | AK | no | Row in the source file |
| `FieldName` | `nvarchar(64)` | | yes | Field that failed |
| `RawValue` | `nvarchar(500)` | | yes | Value supplied |
| `ErrorCode` | `nvarchar(40)` | | no | Stable application error code |
| `ErrorMessage` | `nvarchar(500)` | | no | Message shown to the user |
| `Severity` | `nvarchar(10)` | | no | `Error`, `Warning` |

---

### `intg.BankStatement`
**Imported electronic bank statement header** · reference: `FEBKO`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `StatementNumber` | `nvarchar(20)` | AK | no | Statement number |
| `StatementDate` | `date` | AK | no | Statement date |
| `HouseBankId` | `bigint` | FK | no | House bank |
| `HouseBankAccountId` | `bigint` | FK | no | Bank account |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Statement currency |
| `OpeningBalance` | `decimal(19,4)` | | no | Opening balance |
| `ClosingBalance` | `decimal(19,4)` | | no | Closing balance |
| `TotalDebitAmount` | `decimal(19,4)` | | no | Total debits |
| `TotalCreditAmount` | `decimal(19,4)` | | no | Total credits |
| `ItemCount` | `int` | | no | Number of items |
| `FileFormat` | `nvarchar(20)` | | no | `CAMT053`, `MT940`, `CSV` |
| `ImportJobId` | `bigint` | FK | yes | Import job |
| `Status` | `nvarchar(20)` | | no | `Imported`, `PartiallyPosted`, `Posted`, `Failed` |
| `PostedAt` | `datetime2(3)` | | yes | Posting timestamp (UTC) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `intg.BankStatementItem`
**Bank statement line and its clearing result** · reference: `FEBEP`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `BankStatementId` | `bigint` | AK,FK | no | Statement |
| `ItemNumber` | `int` | AK | no | Item number |
| `ValueDate` | `date` | | no | Value date |
| `BookingDate` | `date` | | no | Booking date |
| `Amount` | `decimal(19,4)` | | no | Amount (signed) |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `DebitCreditIndicator` | `nvarchar(1)` | | no | `S` debit, `H` credit |
| `BankTransactionCode` | `nvarchar(10)` | | yes | Bank transaction code |
| `PaymentReference` | `nvarchar(40)` | | yes | End-to-end reference |
| `PartnerName` | `nvarchar(60)` | | yes | Counterparty name |
| `PartnerIban` | `nvarchar(34)` | | yes | Counterparty IBAN |
| `PurposeText` | `nvarchar(500)` | | yes | Remittance information |
| `MatchedBusinessPartnerId` | `bigint` | FK | yes | Partner identified |
| `MatchedOpenItemId` | `bigint` | FK | yes | Open item identified |
| `MatchConfidencePercent` | `decimal(9,4)` | | yes | Confidence of the automatic match |
| `Status` | `nvarchar(20)` | IX | no | `Unprocessed`, `Matched`, `PartiallyMatched`, `Posted`, `Rejected`, `Manual` |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Accounting document created |
| `ClearingDocumentId` | `bigint` | FK | yes | Clearing document created |
| `ProcessedBy` | `nvarchar(64)` | | yes | User who processed the item |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `intg.IntegrationEndpoint`
**Outbound endpoint configuration (payroll, bank, logistics)**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `EndpointCode` | `nvarchar(40)` | AK | no | Endpoint key |
| `Name` | `nvarchar(60)` | | no | Description |
| `Direction` | `nvarchar(10)` | | no | `Inbound`, `Outbound` |
| `Protocol` | `nvarchar(20)` | | no | `Https`, `Sftp`, `FileShare`, `MessageQueue` |
| `TargetUrl` | `nvarchar(500)` | | yes | Endpoint address |
| `AuthenticationType` | `nvarchar(20)` | | no | `None`, `Basic`, `ApiKey`, `OAuth2`, `Certificate` |
| `CredentialReference` | `nvarchar(255)` | | yes | Reference to the secret store — never the secret itself |
| `PayloadFormat` | `nvarchar(20)` | | no | `Json`, `Xml`, `Csv`, `Fixed` |
| `ScheduleCron` | `nvarchar(40)` | | yes | Schedule for polling or sending |
| `TimeoutSeconds` | `int` | | no | Request timeout |
| `MaxRetries` | `int` | | no | Retry limit |
| `IsActive` | `bit` | | no | Endpoint active |
| `LastRunAt` | `datetime2(3)` | | yes | Last run (UTC) |
| `LastRunStatus` | `nvarchar(20)` | | yes | Result of the last run |
| *include* | `#AUDIT` | | | Standard audit columns |
