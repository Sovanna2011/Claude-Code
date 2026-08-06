# 05 — Financial Accounting (`fin`)

`fin.JournalEntryLine` is the **universal journal**: one line table carrying
G/L, AR, AP, asset and controlling account assignments together with all
currency amounts. Every report in this system reconciles to it, because no
subledger keeps its own parallel balance.

Immutability: `fin.JournalEntryHeader` and `fin.JournalEntryLine` are
insert-only. The only mutable columns are the clearing fields
(`ClearingDocumentNumber`, `ClearingDate`) and the reversal pointer, all
written by dedicated engine operations that also insert their own audit rows.

---

## 5.1 Universal journal

### `fin.JournalEntryHeader`
**Accounting document header** · reference: `BKPF`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `DocumentNumber` | `nvarchar(20)` | AK | no | Document number, e.g. `KSS-2026-SA-0000000123` |
| `LedgerId` | `bigint` | AK,FK | no | Ledger — the leading ledger for standard postings |
| `DocumentTypeId` | `bigint` | FK | no | Document type |
| `DocumentDate` | `date` | | no | Document date |
| `PostingDate` | `date` | IX | no | Posting date — determines period |
| `EntryDate` | `date` | | no | Date the document was entered |
| `EntryTime` | `time(0)` | | no | Time of entry |
| `FiscalPeriod` | `tinyint` | IX | no | Posting period derived from the posting date |
| `TranslationDate` | `date` | | yes | Date used for currency translation |
| `DocumentCurrencyCode` | `nvarchar(5)` | FK | no | Document (transaction) currency |
| `LocalCurrencyCode` | `nvarchar(5)` | FK | no | Company code currency |
| `GroupCurrencyCode` | `nvarchar(5)` | FK | yes | Group currency |
| `ExchangeRate` | `decimal(23,6)` | | yes | Document → local exchange rate |
| `ExchangeRateType` | `nvarchar(4)` | | yes | Rate type used |
| `GroupExchangeRate` | `decimal(23,6)` | | yes | Document → group exchange rate |
| `TotalDebitAmount` | `decimal(19,4)` | | no | Sum of debits in document currency |
| `TotalCreditAmount` | `decimal(19,4)` | | no | Sum of credits in document currency |
| `ReferenceDocumentNumber` | `nvarchar(20)` | IX | yes | External reference (invoice number) |
| `DocumentHeaderText` | `nvarchar(255)` | | yes | Header text |
| `Status` | `nvarchar(20)` | IX | no | `Draft`, `Held`, `Parked`, `Submitted`, `PendingApproval`, `Approved`, `Posted`, `Rejected`, `Reversed`, `Cancelled` |
| `PostedAt` | `datetime2(3)` | | yes | Posting timestamp (UTC) |
| `PostedBy` | `nvarchar(64)` | | yes | Posting user |
| `ParkedBy` | `nvarchar(64)` | | yes | User who parked the document |
| `ApprovedBy` | `nvarchar(64)` | | yes | Approving user — must differ from `PostedBy` under maker-checker |
| `ApprovedAt` | `datetime2(3)` | | yes | Approval timestamp (UTC) |
| `WorkflowInstanceId` | `bigint` | FK | yes | Approval workflow instance |
| `IsReversed` | `bit` | | no | Document has been reversed |
| `ReversalDocumentNumber` | `nvarchar(20)` | | yes | Reversing document |
| `ReversalReasonCode` | `nvarchar(2)` | | yes | Reversal reason |
| `ReversedDocumentNumber` | `nvarchar(20)` | | yes | Original document, if this is a reversal |
| `ReversalDate` | `date` | | yes | Posting date of the reversal |
| `IsIntercompany` | `bit` | | no | Part of a cross-company-code transaction |
| `IntercompanyDocumentNumber` | `nvarchar(20)` | IX | yes | Cross-company-code document number |
| `SourceModule` | `nvarchar(10)` | | no | `FI`, `AR`, `AP`, `AA`, `CO`, `MM`, `SD`, `PY` |
| `SourceDocumentType` | `nvarchar(20)` | | yes | Originating object type |
| `SourceDocumentId` | `bigint` | | yes | Originating object key |
| `TransactionCode` | `nvarchar(20)` | | yes | T-code used, e.g. `FB50` |
| `IdempotencyKey` | `uniqueidentifier` | AK | yes | Prevents duplicate posting on retry |
| `CorrelationId` | `uniqueidentifier` | IX | yes | Request correlation id |
| `BatchInputSession` | `nvarchar(20)` | | yes | Import / batch session name |
| `IsSimulation` | `bit` | | no | Simulated document, never balanced into totals |
| `CreatedAt` | `datetime2(3)` | | no | Creation timestamp (UTC) |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user |
| `ModifiedAt` | `datetime2(3)` | | yes | Last change before posting only |
| `ModifiedBy` | `nvarchar(64)` | | yes | Last changing user before posting |

---

### `fin.JournalEntryLine`
**Universal journal line item — the single source of accounting truth** · reference: `ACDOCA` / `BSEG`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `JournalEntryHeaderId` | `bigint` | FK,IX | no | Owning document header |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `DocumentNumber` | `nvarchar(20)` | AK | no | Document number |
| `LineItemNumber` | `int` | AK | no | Line number within the document |
| `LedgerId` | `bigint` | AK,FK | no | Ledger |
| `PostingDate` | `date` | IX | no | Posting date (denormalised for reporting) |
| `FiscalPeriod` | `tinyint` | IX | no | Posting period |
| `PostingKey` | `nvarchar(2)` | FK | no | Posting key |
| `DebitCreditIndicator` | `nvarchar(1)` | | no | `S` debit, `H` credit |
| `AccountType` | `nvarchar(1)` | | no | `S`, `D`, `K`, `A`, `M` |
| `GLAccountId` | `bigint` | FK,IX | no | G/L account — reconciliation account for D/K lines |
| `GLAccount` | `nvarchar(10)` | IX | no | G/L account number (denormalised) |
| `BusinessPartnerId` | `bigint` | FK,IX | yes | Customer / vendor partner |
| `BusinessPartnerRoleCategory` | `nvarchar(20)` | | yes | `Customer` or `Vendor` — which role posted |
| `SpecialGLIndicator` | `nvarchar(1)` | | yes | Special G/L indicator |
| `AssetId` | `bigint` | FK | yes | Asset master record |
| `AssetSubNumber` | `int` | | yes | Asset sub-number |
| `AssetTransactionType` | `nvarchar(3)` | | yes | Asset transaction type |
| `CostCenterId` | `bigint` | FK,IX | yes | Cost centre |
| `ProfitCenterId` | `bigint` | FK,IX | yes | Profit centre |
| `PartnerProfitCenterId` | `bigint` | FK | yes | Partner profit centre |
| `InternalOrderId` | `bigint` | FK,IX | yes | Internal order |
| `CostElementId` | `bigint` | FK | yes | Cost element |
| `ActivityTypeId` | `bigint` | FK | yes | Activity type |
| `BusinessAreaId` | `bigint` | FK | yes | Business area |
| `FunctionalAreaId` | `bigint` | FK | yes | Functional area |
| `SegmentId` | `bigint` | FK,IX | yes | Segment |
| `PartnerSegmentId` | `bigint` | FK | yes | Partner segment |
| `PartnerCompanyCodeId` | `bigint` | FK | yes | Partner company code (intercompany) |
| `TradingPartnerCompany` | `nvarchar(6)` | | yes | Trading partner for consolidation |
| `PlantId` | `bigint` | FK | yes | Plant |
| `BranchId` | `bigint` | FK | yes | Branch |
| `ProjectId` | `bigint` | FK | yes | Project / WBS element (future module) |
| `DocumentCurrencyCode` | `nvarchar(5)` | FK | no | Document currency |
| `AmountInDocumentCurrency` | `decimal(19,4)` | | no | Amount in document currency (signed) |
| `LocalCurrencyCode` | `nvarchar(5)` | FK | no | Company code currency |
| `AmountInLocalCurrency` | `decimal(19,4)` | | no | Amount in local currency (signed) |
| `GroupCurrencyCode` | `nvarchar(5)` | FK | yes | Group currency |
| `AmountInGroupCurrency` | `decimal(19,4)` | | yes | Amount in group currency (signed) |
| `ControllingAreaCurrencyCode` | `nvarchar(5)` | FK | yes | Controlling area currency |
| `AmountInControllingAreaCurrency` | `decimal(19,4)` | | yes | Amount in controlling area currency |
| `HardCurrencyCode` | `nvarchar(5)` | FK | yes | Hard currency |
| `AmountInHardCurrency` | `decimal(19,4)` | | yes | Amount in hard currency |
| `ExchangeRate` | `decimal(23,6)` | | yes | Rate document → local |
| `Quantity` | `decimal(23,6)` | | yes | Quantity |
| `UnitOfMeasure` | `nvarchar(3)` | FK | yes | Unit of measure |
| `TaxCodeId` | `bigint` | FK | yes | Tax code |
| `TaxBaseAmountInDocumentCurrency` | `decimal(19,4)` | | yes | Tax base amount |
| `TaxAmountInDocumentCurrency` | `decimal(19,4)` | | yes | Tax amount in document currency |
| `TaxAmountInLocalCurrency` | `decimal(19,4)` | | yes | Tax amount in local currency |
| `TaxJurisdictionCode` | `nvarchar(15)` | | yes | Tax jurisdiction |
| `IsTaxLine` | `bit` | | no | Automatically generated tax line |
| `WithholdingTaxCodeId` | `bigint` | FK | yes | Withholding tax code |
| `WithholdingTaxBaseAmount` | `decimal(19,4)` | | yes | Withholding tax base |
| `WithholdingTaxAmount` | `decimal(19,4)` | | yes | Withholding tax amount |
| `AssignmentReference` | `nvarchar(18)` | IX | yes | Allocation field (sort key result) |
| `LineItemText` | `nvarchar(255)` | | yes | Item text |
| `ReferenceKey1` | `nvarchar(20)` | | yes | Reference key 1 |
| `ReferenceKey2` | `nvarchar(20)` | | yes | Reference key 2 |
| `ReferenceKey3` | `nvarchar(20)` | | yes | Reference key 3 |
| `BaselineDate` | `date` | | yes | Baseline date for payment terms |
| `PaymentTermsId` | `bigint` | FK | yes | Payment terms |
| `DueDate` | `date` | IX | yes | Net due date |
| `CashDiscount1Percent` | `decimal(9,4)` | | yes | Cash discount percentage 1 |
| `CashDiscount1Days` | `int` | | yes | Cash discount days 1 |
| `CashDiscountBaseAmount` | `decimal(19,4)` | | yes | Amount eligible for discount |
| `PaymentMethod` | `nvarchar(1)` | | yes | Payment method |
| `PaymentBlockReason` | `nvarchar(1)` | | yes | Payment block |
| `HouseBankId` | `bigint` | FK | yes | House bank for payment |
| `PartnerBankDetailId` | `nvarchar(4)` | | yes | Partner bank details id |
| `IsOpenItemManaged` | `bit` | | no | Line creates an open item |
| `ClearingStatus` | `nvarchar(20)` | | no | `Open`, `PartiallyCleared`, `Cleared` |
| `ClearingDocumentNumber` | `nvarchar(20)` | IX | yes | Clearing document |
| `ClearingDate` | `date` | | yes | Clearing date |
| `InvoiceReferenceDocumentNumber` | `nvarchar(20)` | | yes | Invoice a credit memo or payment refers to |
| `InvoiceReferenceFiscalYear` | `smallint` | | yes | Fiscal year of the referenced invoice |
| `InvoiceReferenceLineItem` | `int` | | yes | Line of the referenced invoice |
| `IsReversalLine` | `bit` | | no | Line belongs to a reversal document |
| `IsNegativePosting` | `bit` | | no | Negative posting (reduces the same side) |
| `IsStatistical` | `bit` | | no | Statistical posting (noted item, statistical order) |
| `SourceModule` | `nvarchar(10)` | | no | Originating module |
| `ControllingDocumentNumber` | `nvarchar(30)` | | yes | Related CO document (FI number plus a `-CO###` suffix) |
| `AssetDocumentNumber` | `nvarchar(20)` | | yes | Related asset document |
| `MaterialNumber` | `nvarchar(40)` | | yes | Material (future inventory module) |
| `CreatedAt` | `datetime2(3)` | | no | Creation timestamp (UTC) |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user |

---

### `fin.JournalEntryTax`
**Tax breakdown per document, code and jurisdiction** · reference: `BSET`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `DocumentNumber` | `nvarchar(20)` | AK | no | Document number |
| `TaxLineNumber` | `int` | AK | no | Tax line sequence |
| `TaxCodeId` | `bigint` | FK | no | Tax code |
| `ConditionType` | `nvarchar(4)` | | no | Condition type |
| `TaxJurisdictionCode` | `nvarchar(15)` | | yes | Tax jurisdiction |
| `TaxRatePercent` | `decimal(9,4)` | | no | Applied rate |
| `TaxBaseAmountInLocalCurrency` | `decimal(19,4)` | | no | Base amount in local currency |
| `TaxAmountInLocalCurrency` | `decimal(19,4)` | | no | Tax amount in local currency |
| `TaxBaseAmountInDocumentCurrency` | `decimal(19,4)` | | no | Base amount in document currency |
| `TaxAmountInDocumentCurrency` | `decimal(19,4)` | | no | Tax amount in document currency |
| `NonDeductibleAmount` | `decimal(19,4)` | | yes | Non-deductible portion |
| `TaxGLAccountId` | `bigint` | FK | no | Tax account posted to |
| `IsOutputTax` | `bit` | | no | Output tax (`A`) rather than input tax (`V`) |
| `ReportingDate` | `date` | | yes | Date used in the tax return |
| `CreatedAt` | `datetime2(3)` | | no | Creation timestamp (UTC) |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user |

---

### `fin.OpenItem`
**Open item index for customers, vendors and open-item-managed G/L accounts** · reference: `BSID`/`BSIK`/`BSIS`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `JournalEntryLineId` | `bigint` | AK,FK | no | Originating journal line |
| `CompanyCodeId` | `bigint` | FK,IX | no | Company code |
| `FiscalYear` | `smallint` | | no | Fiscal year |
| `DocumentNumber` | `nvarchar(20)` | IX | no | Document number |
| `LineItemNumber` | `int` | | no | Line number |
| `AccountType` | `nvarchar(1)` | IX | no | `D`, `K`, `S` |
| `BusinessPartnerId` | `bigint` | FK,IX | yes | Customer / vendor |
| `GLAccountId` | `bigint` | FK,IX | no | Reconciliation or G/L account |
| `SpecialGLIndicator` | `nvarchar(1)` | | yes | Special G/L indicator |
| `PostingDate` | `date` | IX | no | Posting date |
| `DocumentDate` | `date` | | no | Document date |
| `BaselineDate` | `date` | | yes | Baseline date |
| `DueDate` | `date` | IX | yes | Net due date |
| `DaysInArrears` | `int` | | yes | Days overdue at the last aging run |
| `DocumentCurrencyCode` | `nvarchar(5)` | FK | no | Document currency |
| `OriginalAmountInDocumentCurrency` | `decimal(19,4)` | | no | Original amount |
| `OpenAmountInDocumentCurrency` | `decimal(19,4)` | | no | Remaining open amount |
| `OriginalAmountInLocalCurrency` | `decimal(19,4)` | | no | Original amount in local currency |
| `OpenAmountInLocalCurrency` | `decimal(19,4)` | | no | Remaining open amount in local currency |
| `ClearedAmountInDocumentCurrency` | `decimal(19,4)` | | no | Amount already cleared |
| `DebitCreditIndicator` | `nvarchar(1)` | | no | `S` debit, `H` credit |
| `PaymentTermsId` | `bigint` | FK | yes | Payment terms |
| `CashDiscountAmount` | `decimal(19,4)` | | yes | Cash discount still available |
| `CashDiscountDueDate` | `date` | | yes | Last day for the cash discount |
| `PaymentMethod` | `nvarchar(1)` | | yes | Payment method |
| `PaymentBlockReason` | `nvarchar(1)` | | yes | Payment block |
| `DunningLevel` | `tinyint` | | no | Current dunning level |
| `LastDunningDate` | `date` | | yes | Last dunning date |
| `DunningBlockReason` | `nvarchar(1)` | | yes | Dunning block |
| `AssignmentReference` | `nvarchar(18)` | IX | yes | Allocation field |
| `ReferenceDocumentNumber` | `nvarchar(20)` | | yes | External reference |
| `LineItemText` | `nvarchar(255)` | | yes | Item text |
| `Status` | `nvarchar(20)` | IX | no | `Open`, `PartiallyCleared`, `Cleared` |
| `ClearingDocumentNumber` | `nvarchar(20)` | | yes | Clearing document |
| `ClearingDate` | `date` | | yes | Clearing date |
| `IsDisputed` | `bit` | | no | Item under dispute |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.ClearingDocument`
**Clearing transaction header** · reference: clearing part of `BSEG`/`BKPF`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `ClearingDocumentNumber` | `nvarchar(20)` | AK | no | Clearing document number |
| `ClearingDate` | `date` | IX | no | Clearing date |
| `PostingDate` | `date` | | no | Posting date of the clearing document |
| `ClearingType` | `nvarchar(20)` | | no | `IncomingPayment`, `OutgoingPayment`, `Transfer`, `Manual`, `Automatic`, `Reset` |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Accounting document created by clearing |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Clearing currency |
| `TotalClearedAmount` | `decimal(19,4)` | | no | Total amount cleared |
| `DifferenceAmount` | `decimal(19,4)` | | no | Residual / tolerance difference |
| `DifferenceHandling` | `nvarchar(20)` | | yes | `Residual`, `PartialPayment`, `WriteOff`, `OnAccount` |
| `IsReset` | `bit` | | no | Clearing has been reset |
| `ResetAt` | `datetime2(3)` | | yes | Reset timestamp (UTC) |
| `ResetBy` | `nvarchar(64)` | | yes | User who reset the clearing |
| `CreatedAt` | `datetime2(3)` | | no | Creation timestamp (UTC) |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user |

---

### `fin.ClearingItem`
**Open item cleared by a clearing document**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ClearingDocumentId` | `bigint` | AK,FK | no | Clearing document |
| `OpenItemId` | `bigint` | AK,FK | no | Open item cleared |
| `ClearedAmountInDocumentCurrency` | `decimal(19,4)` | | no | Amount cleared in document currency |
| `ClearedAmountInLocalCurrency` | `decimal(19,4)` | | no | Amount cleared in local currency |
| `CashDiscountTaken` | `decimal(19,4)` | | yes | Cash discount granted |
| `ExchangeRateDifference` | `decimal(19,4)` | | yes | Realised exchange rate gain/loss |
| `IsPartialClearing` | `bit` | | no | Partial clearing |
| `ResidualOpenItemId` | `bigint` | FK | yes | Residual item created |
| `CreatedAt` | `datetime2(3)` | | no | Creation timestamp (UTC) |
| `CreatedBy` | `nvarchar(64)` | | no | Creating user |

---

## 5.2 Accounts receivable

### `fin.CustomerInvoice`
**Customer invoice / credit memo header** · reference: FI invoice via `VBRK`/`BKPF`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `InvoiceNumber` | `nvarchar(20)` | AK | no | Invoice number |
| `InvoiceType` | `nvarchar(20)` | | no | `Invoice`, `CreditMemo`, `DownPaymentRequest`, `ProForma` |
| `BusinessPartnerId` | `bigint` | FK,IX | no | Customer (BP with customer role) |
| `PayerBusinessPartnerId` | `bigint` | FK | yes | Alternative payer |
| `BillToBusinessPartnerId` | `bigint` | FK | yes | Bill-to party |
| `ShipToBusinessPartnerId` | `bigint` | FK | yes | Ship-to party |
| `InvoiceDate` | `date` | | no | Invoice date |
| `PostingDate` | `date` | IX | no | Posting date |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Invoice currency |
| `ExchangeRate` | `decimal(23,6)` | | yes | Exchange rate to local currency |
| `NetAmount` | `decimal(19,4)` | | no | Net amount |
| `TaxAmount` | `decimal(19,4)` | | no | Tax amount |
| `GrossAmount` | `decimal(19,4)` | | no | Gross amount |
| `WithholdingTaxAmount` | `decimal(19,4)` | | yes | Withholding tax |
| `PaymentTermsId` | `bigint` | FK | yes | Payment terms |
| `BaselineDate` | `date` | | yes | Baseline date |
| `DueDate` | `date` | | yes | Due date |
| `PaymentMethod` | `nvarchar(1)` | | yes | Payment method |
| `SalesAreaId` | `bigint` | FK | yes | Sales area |
| `ReferenceDocumentNumber` | `nvarchar(20)` | | yes | Customer reference / PO number |
| `HeaderText` | `nvarchar(255)` | | yes | Header text |
| `Status` | `nvarchar(20)` | IX | no | `Draft`, `Parked`, `PendingApproval`, `Posted`, `PartiallyCleared`, `Cleared`, `Reversed`, `Cancelled` |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Accounting document created |
| `PaidAmount` | `decimal(19,4)` | | no | Amount received so far |
| `OpenAmount` | `decimal(19,4)` | | no | Amount still open |
| `IsDunningBlocked` | `bit` | | no | Dunning block |
| `WorkflowInstanceId` | `bigint` | FK | yes | Approval workflow |
| `IdempotencyKey` | `uniqueidentifier` | | yes | Duplicate protection |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.CustomerInvoiceItem`
**Customer invoice line**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CustomerInvoiceId` | `bigint` | AK,FK | no | Owning invoice |
| `ItemNumber` | `int` | AK | no | Item number |
| `RevenueGLAccountId` | `bigint` | FK | no | Revenue account |
| `Description` | `nvarchar(255)` | | no | Item description |
| `Quantity` | `decimal(23,6)` | | yes | Quantity |
| `UnitOfMeasure` | `nvarchar(3)` | FK | yes | Unit of measure |
| `UnitPrice` | `decimal(19,4)` | | yes | Unit price |
| `NetAmount` | `decimal(19,4)` | | no | Net amount |
| `DiscountAmount` | `decimal(19,4)` | | yes | Item discount |
| `TaxCodeId` | `bigint` | FK | yes | Tax code |
| `TaxAmount` | `decimal(19,4)` | | yes | Tax amount |
| `CostCenterId` | `bigint` | FK | yes | Cost centre |
| `ProfitCenterId` | `bigint` | FK | yes | Profit centre |
| `InternalOrderId` | `bigint` | FK | yes | Internal order |
| `SegmentId` | `bigint` | FK | yes | Segment |
| `FunctionalAreaId` | `bigint` | FK | yes | Functional area |
| `BusinessAreaId` | `bigint` | FK | yes | Business area |
| `MaterialNumber` | `nvarchar(40)` | | yes | Material (future module) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 5.3 Accounts payable

### `fin.VendorInvoice`
**Vendor invoice / credit memo header** · reference: `RBKP`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `InvoiceNumber` | `nvarchar(20)` | AK | no | Internal invoice number |
| `InvoiceType` | `nvarchar(20)` | | no | `Invoice`, `CreditMemo`, `DownPaymentRequest`, `SubsequentDebit` |
| `BusinessPartnerId` | `bigint` | FK,IX | no | Vendor (BP with vendor role) |
| `PayeeBusinessPartnerId` | `bigint` | FK | yes | Alternative payee |
| `VendorInvoiceNumber` | `nvarchar(20)` | IX | no | Vendor's own invoice number — duplicate check |
| `InvoiceDate` | `date` | | no | Invoice date |
| `PostingDate` | `date` | IX | no | Posting date |
| `ReceiptDate` | `date` | | yes | Date the invoice was received |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Invoice currency |
| `ExchangeRate` | `decimal(23,6)` | | yes | Exchange rate |
| `NetAmount` | `decimal(19,4)` | | no | Net amount |
| `TaxAmount` | `decimal(19,4)` | | no | Tax amount |
| `GrossAmount` | `decimal(19,4)` | | no | Gross amount |
| `WithholdingTaxAmount` | `decimal(19,4)` | | yes | Withholding tax |
| `PaymentTermsId` | `bigint` | FK | yes | Payment terms |
| `BaselineDate` | `date` | | yes | Baseline date |
| `DueDate` | `date` | | yes | Due date |
| `PaymentMethod` | `nvarchar(1)` | | yes | Payment method |
| `PaymentBlockReason` | `nvarchar(1)` | | yes | Payment block |
| `HouseBankId` | `bigint` | FK | yes | House bank for payment |
| `PartnerBankDetailId` | `nvarchar(4)` | | yes | Vendor bank details used |
| `PurchasingOrganizationId` | `bigint` | FK | yes | Purchasing organisation |
| `PurchaseOrderNumber` | `nvarchar(20)` | | yes | Purchase order reference (future module) |
| `HeaderText` | `nvarchar(255)` | | yes | Header text |
| `Status` | `nvarchar(20)` | IX | no | `Draft`, `Parked`, `PendingApproval`, `Approved`, `Posted`, `PartiallyCleared`, `Cleared`, `Reversed`, `Cancelled` |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Accounting document created |
| `PaidAmount` | `decimal(19,4)` | | no | Amount paid so far |
| `OpenAmount` | `decimal(19,4)` | | no | Amount still open |
| `WorkflowInstanceId` | `bigint` | FK | yes | Approval workflow |
| `IdempotencyKey` | `uniqueidentifier` | | yes | Duplicate protection |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.VendorInvoiceItem`
**Vendor invoice line** · reference: `RSEG`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `VendorInvoiceId` | `bigint` | AK,FK | no | Owning invoice |
| `ItemNumber` | `int` | AK | no | Item number |
| `ExpenseGLAccountId` | `bigint` | FK | no | Expense / balance sheet account |
| `Description` | `nvarchar(255)` | | no | Item description |
| `Quantity` | `decimal(23,6)` | | yes | Quantity |
| `UnitOfMeasure` | `nvarchar(3)` | FK | yes | Unit of measure |
| `UnitPrice` | `decimal(19,4)` | | yes | Unit price |
| `NetAmount` | `decimal(19,4)` | | no | Net amount |
| `TaxCodeId` | `bigint` | FK | yes | Tax code |
| `TaxAmount` | `decimal(19,4)` | | yes | Tax amount |
| `IsNonDeductibleTax` | `bit` | | no | Tax not deductible — added to the expense |
| `CostCenterId` | `bigint` | FK | yes | Cost centre |
| `ProfitCenterId` | `bigint` | FK | yes | Profit centre |
| `InternalOrderId` | `bigint` | FK | yes | Internal order |
| `AssetId` | `bigint` | FK | yes | Asset for vendor acquisition |
| `SegmentId` | `bigint` | FK | yes | Segment |
| `FunctionalAreaId` | `bigint` | FK | yes | Functional area |
| `BusinessAreaId` | `bigint` | FK | yes | Business area |
| `PlantId` | `bigint` | FK | yes | Plant |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 5.4 Payments and dunning

### `fin.PaymentHeader`
**Incoming or outgoing payment** · reference: `REGUH`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `PaymentNumber` | `nvarchar(20)` | AK | no | Payment document number |
| `PaymentDirection` | `nvarchar(10)` | | no | `Incoming`, `Outgoing` |
| `PaymentType` | `nvarchar(20)` | | no | `Manual`, `AutomaticRun`, `DownPayment`, `OnAccount`, `Refund` |
| `BusinessPartnerId` | `bigint` | FK,IX | no | Customer or vendor paid |
| `PaymentDate` | `date` | IX | no | Value date |
| `PostingDate` | `date` | | no | Posting date |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Payment currency |
| `ExchangeRate` | `decimal(23,6)` | | yes | Exchange rate |
| `PaymentAmount` | `decimal(19,4)` | | no | Gross payment amount |
| `CashDiscountAmount` | `decimal(19,4)` | | yes | Cash discount taken |
| `WithholdingTaxAmount` | `decimal(19,4)` | | yes | Withholding tax deducted |
| `BankChargeAmount` | `decimal(19,4)` | | yes | Bank charges |
| `AllocatedAmount` | `decimal(19,4)` | | no | Amount allocated to open items |
| `UnallocatedAmount` | `decimal(19,4)` | | no | On-account (unapplied) amount |
| `PaymentMethod` | `nvarchar(1)` | | no | Payment method |
| `HouseBankId` | `bigint` | FK | yes | House bank |
| `HouseBankAccountId` | `bigint` | FK | yes | House bank account |
| `PartnerBankDetailId` | `nvarchar(4)` | | yes | Partner bank details |
| `CheckNumber` | `nvarchar(20)` | | yes | Cheque number |
| `PaymentReference` | `nvarchar(40)` | | yes | Payment reference / end-to-end id |
| `PaymentRunId` | `bigint` | FK | yes | Automatic payment run that created it |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Accounting document created |
| `ClearingDocumentId` | `bigint` | FK | yes | Clearing document created |
| `Status` | `nvarchar(20)` | IX | no | `Draft`, `PendingApproval`, `Approved`, `Posted`, `Sent`, `Confirmed`, `Reversed`, `Rejected` |
| `WorkflowInstanceId` | `bigint` | FK | yes | Approval workflow |
| `IdempotencyKey` | `uniqueidentifier` | | yes | Duplicate protection |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.PaymentAllocation`
**Allocation of a payment to an open item** · reference: `REGUP`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PaymentHeaderId` | `bigint` | AK,FK | no | Payment |
| `OpenItemId` | `bigint` | AK,FK | no | Open item settled |
| `AllocatedAmount` | `decimal(19,4)` | | no | Amount applied |
| `AllocatedAmountInLocalCurrency` | `decimal(19,4)` | | no | Amount applied in local currency |
| `CashDiscountAmount` | `decimal(19,4)` | | yes | Discount granted on this item |
| `ExchangeRateDifference` | `decimal(19,4)` | | yes | Realised exchange difference |
| `AllocationType` | `nvarchar(20)` | | no | `Full`, `Partial`, `Residual`, `OnAccount` |
| `ResidualOpenItemId` | `bigint` | FK | yes | Residual item created |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.PaymentRun`
**Automatic payment run (F110)** · reference: `REGUV`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RunDate` | `date` | AK | no | Run date |
| `RunIdentifier` | `nvarchar(6)` | AK | no | Identification of the run |
| `CompanyCodeId` | `bigint` | FK | no | Company code |
| `PostingDate` | `date` | | no | Posting date of the payments |
| `DocumentDate` | `date` | | no | Document date |
| `NextPaymentDate` | `date` | | no | Date of the next planned run |
| `PaymentMethods` | `nvarchar(10)` | | no | Payment methods considered |
| `BusinessPartnerFrom` | `nvarchar(10)` | | yes | Partner range start |
| `BusinessPartnerTo` | `nvarchar(10)` | | yes | Partner range end |
| `CurrencyCode` | `nvarchar(5)` | FK | yes | Restrict to one currency |
| `MinimumPaymentAmount` | `decimal(19,4)` | | yes | Minimum amount per payment |
| `Status` | `nvarchar(20)` | | no | `Scheduled`, `ProposalRunning`, `ProposalReady`, `ProposalEdited`, `PaymentRunning`, `Posted`, `FileCreated`, `Cancelled` |
| `ProposalCreatedAt` | `datetime2(3)` | | yes | Proposal timestamp (UTC) |
| `PaymentPostedAt` | `datetime2(3)` | | yes | Posting timestamp (UTC) |
| `TotalPaymentAmount` | `decimal(19,4)` | | yes | Total paid |
| `PaymentCount` | `int` | | yes | Number of payments created |
| `ExceptionCount` | `int` | | yes | Number of blocked / exception items |
| `PaymentFileUri` | `nvarchar(500)` | | yes | Generated bank file |
| `ApprovedBy` | `nvarchar(64)` | | yes | Approver of the run |
| `ApprovedAt` | `datetime2(3)` | | yes | Approval timestamp (UTC) |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.PaymentProposalItem`
**Open item selected by a payment proposal**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `PaymentRunId` | `bigint` | AK,FK | no | Payment run |
| `OpenItemId` | `bigint` | AK,FK | no | Open item |
| `BusinessPartnerId` | `bigint` | FK | no | Partner |
| `ProposedAmount` | `decimal(19,4)` | | no | Amount proposed |
| `CashDiscountAmount` | `decimal(19,4)` | | yes | Discount proposed |
| `PaymentMethod` | `nvarchar(1)` | | no | Payment method proposed |
| `HouseBankId` | `bigint` | FK | yes | House bank proposed |
| `IsBlocked` | `bit` | | no | Excluded from the run |
| `BlockReason` | `nvarchar(255)` | | yes | Reason for exclusion |
| `IsEdited` | `bit` | | no | Changed during proposal editing |
| `PaymentHeaderId` | `bigint` | FK | yes | Payment created from this item |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.DunningRun`
**Dunning run** · reference: `MHNK` run level

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RunDate` | `date` | AK | no | Run date |
| `RunIdentifier` | `nvarchar(6)` | AK | no | Identification of the run |
| `CompanyCodeId` | `bigint` | FK | no | Company code |
| `DunningDate` | `date` | | no | Date printed on the letters |
| `DocumentsPostedUntil` | `date` | | no | Documents considered up to this date |
| `Status` | `nvarchar(20)` | | no | `Scheduled`, `ProposalReady`, `Edited`, `Printed`, `Cancelled` |
| `NoticeCount` | `int` | | yes | Number of notices produced |
| `TotalDunnedAmount` | `decimal(19,4)` | | yes | Total amount dunned |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.DunningNotice`
**Dunning notice per partner and level** · reference: `MHND`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DunningRunId` | `bigint` | AK,FK | no | Dunning run |
| `BusinessPartnerId` | `bigint` | AK,FK | no | Customer dunned |
| `DunningLevel` | `tinyint` | | no | Dunning level reached |
| `DunningProcedureId` | `bigint` | FK | no | Dunning procedure |
| `TotalOverdueAmount` | `decimal(19,4)` | | no | Total overdue |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `InterestAmount` | `decimal(19,4)` | | yes | Interest charged |
| `DunningChargeAmount` | `decimal(19,4)` | | yes | Dunning charge |
| `ItemCount` | `int` | | no | Number of dunned items |
| `IsPrinted` | `bit` | | no | Notice issued |
| `PrintedAt` | `datetime2(3)` | | yes | Print timestamp (UTC) |
| `DocumentUri` | `nvarchar(500)` | | yes | Generated PDF |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.DunningNoticeItem`
**Open item on a dunning notice**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `DunningNoticeId` | `bigint` | AK,FK | no | Dunning notice |
| `OpenItemId` | `bigint` | AK,FK | no | Dunned open item |
| `DunningLevel` | `tinyint` | | no | Level applied to the item |
| `OverdueAmount` | `decimal(19,4)` | | no | Overdue amount |
| `DaysInArrears` | `int` | | no | Days overdue |
| `InterestAmount` | `decimal(19,4)` | | yes | Interest on this item |
| *include* | `#AUDIT` | | | Standard audit columns |

---

## 5.5 Recurring entries, valuation, balances

### `fin.RecurringEntry`
**Recurring entry template** · reference: `BKDF`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `RecurringDocumentNumber` | `nvarchar(20)` | AK | no | Recurring document number |
| `DocumentTypeId` | `bigint` | FK | no | Document type used on execution |
| `Description` | `nvarchar(255)` | | no | Description |
| `FirstRunDate` | `date` | | no | First run date |
| `LastRunDate` | `date` | | no | Last run date |
| `NextRunDate` | `date` | IX | yes | Next scheduled run |
| `RunSchedule` | `nvarchar(20)` | | no | `Monthly`, `Quarterly`, `Yearly`, `RunDates` |
| `IntervalMonths` | `tinyint` | | yes | Months between runs |
| `RunDayOfMonth` | `tinyint` | | yes | Day of the month |
| `CurrencyCode` | `nvarchar(5)` | FK | no | Currency |
| `TotalAmount` | `decimal(19,4)` | | no | Document amount |
| `IsCopyAmountFromLastRun` | `bit` | | no | Reuse the last executed amount |
| `NumberOfRunsExecuted` | `int` | | no | Executions so far |
| `IsDeletionFlagged` | `bit` | | no | Marked for deletion |
| `Status` | `nvarchar(20)` | | no | `Active`, `Suspended`, `Completed`, `Deleted` |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.RecurringEntryLine`
**Line of a recurring entry template**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RecurringEntryId` | `bigint` | AK,FK | no | Owning template |
| `LineItemNumber` | `int` | AK | no | Line number |
| `PostingKey` | `nvarchar(2)` | FK | no | Posting key |
| `GLAccountId` | `bigint` | FK | yes | G/L account |
| `BusinessPartnerId` | `bigint` | FK | yes | Customer / vendor |
| `Amount` | `decimal(19,4)` | | no | Amount |
| `TaxCodeId` | `bigint` | FK | yes | Tax code |
| `CostCenterId` | `bigint` | FK | yes | Cost centre |
| `ProfitCenterId` | `bigint` | FK | yes | Profit centre |
| `InternalOrderId` | `bigint` | FK | yes | Internal order |
| `SegmentId` | `bigint` | FK | yes | Segment |
| `AssignmentReference` | `nvarchar(18)` | | yes | Allocation field |
| `LineItemText` | `nvarchar(255)` | | yes | Item text |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.RecurringEntryExecution`
**Log of each recurring-entry execution**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `RecurringEntryId` | `bigint` | AK,FK | no | Template executed |
| `ExecutionDate` | `date` | AK | no | Execution date |
| `PostingDate` | `date` | | no | Posting date used |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Document created |
| `Status` | `nvarchar(20)` | | no | `Success`, `Failed`, `Skipped` |
| `Message` | `nvarchar(500)` | | yes | Result message |
| `ExecutedAt` | `datetime2(3)` | | no | Execution timestamp (UTC) |
| `ExecutedBy` | `nvarchar(64)` | | no | Executing user or job |

---

### `fin.ForeignCurrencyValuation`
**Result of a period-end foreign currency valuation**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period valued |
| `ValuationAreaCode` | `nvarchar(4)` | AK | no | Valuation area / accounting principle |
| `OpenItemId` | `bigint` | AK,FK | yes | Valued open item (null for account balances) |
| `GLAccountId` | `bigint` | FK | no | Account valued |
| `ValuationDate` | `date` | | no | Key date of the valuation |
| `ExchangeRateUsed` | `decimal(23,6)` | | no | Rate used |
| `AmountInDocumentCurrency` | `decimal(19,4)` | | no | Original foreign currency amount |
| `BookValueInLocalCurrency` | `decimal(19,4)` | | no | Book value before valuation |
| `ValuatedAmountInLocalCurrency` | `decimal(19,4)` | | no | Value at the valuation rate |
| `UnrealizedGainLossAmount` | `decimal(19,4)` | | no | Unrealised difference |
| `IsReversalPosted` | `bit` | | no | Valuation reversed in the following period |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Valuation posting |
| `ReversalJournalEntryHeaderId` | `bigint` | FK | yes | Reversal posting |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.AccountBalance`
**Period balances per account and dimension — maintained by the posting engine** · reference: `GLT0` / `FAGLFLEXT`

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `LedgerId` | `bigint` | AK,FK | no | Ledger |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `FiscalYear` | `smallint` | AK | no | Fiscal year |
| `FiscalPeriod` | `tinyint` | AK | no | Period (`0` = carry-forward) |
| `GLAccountId` | `bigint` | AK,FK | no | G/L account |
| `BusinessPartnerId` | `bigint` | AK,FK | yes | Customer / vendor for subledger balances |
| `ProfitCenterId` | `bigint` | AK,FK | yes | Profit centre |
| `SegmentId` | `bigint` | AK,FK | yes | Segment |
| `FunctionalAreaId` | `bigint` | AK,FK | yes | Functional area |
| `BusinessAreaId` | `bigint` | AK,FK | yes | Business area |
| `CurrencyType` | `nvarchar(2)` | AK | no | `10` local, `30` group, `00` document |
| `CurrencyCode` | `nvarchar(5)` | AK,FK | no | Currency of the amounts |
| `DebitTotal` | `decimal(19,4)` | | no | Total debits in the period |
| `CreditTotal` | `decimal(19,4)` | | no | Total credits in the period |
| `PeriodBalance` | `decimal(19,4)` | | no | Debits − credits |
| `CumulativeBalance` | `decimal(19,4)` | | no | Balance including carry-forward |
| `LastUpdatedAt` | `datetime2(3)` | | no | Last update (UTC) |
| `RowVersion` | `rowversion` | | no | Concurrency token |

---

### `fin.BalanceCarryForward`
**Year-end carry-forward run and its results**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `CompanyCodeId` | `bigint` | AK,FK | no | Company code |
| `LedgerId` | `bigint` | AK,FK | no | Ledger |
| `FromFiscalYear` | `smallint` | AK | no | Year closed |
| `ToFiscalYear` | `smallint` | AK | no | Year opened |
| `RetainedEarningsGLAccountId` | `bigint` | FK | no | Retained earnings account |
| `CarriedForwardBalanceSheetAmount` | `decimal(19,4)` | | no | Balance sheet total carried forward |
| `CarriedForwardProfitLossAmount` | `decimal(19,4)` | | no | P&L result transferred |
| `AccountsProcessed` | `int` | | no | Number of accounts processed |
| `Status` | `nvarchar(20)` | | no | `Running`, `Completed`, `Failed`, `Repeated` |
| `ExecutedAt` | `datetime2(3)` | | no | Execution timestamp (UTC) |
| `ExecutedBy` | `nvarchar(64)` | | no | Executing user |
| `JournalEntryHeaderId` | `bigint` | FK | yes | Carry-forward document |
| *include* | `#AUDIT` | | | Standard audit columns |

---

### `fin.DocumentAttachment`
**File attached to an accounting document, invoice or payment**

| Field Name | Data Type | Key | Null | Description |
|---|---|---|---|---|
| `Id` | `bigint` | PK | no | Surrogate key |
| *include* | `#TENANT` | | | Tenant discriminator |
| `ObjectType` | `nvarchar(40)` | AK | no | `JournalEntry`, `CustomerInvoice`, `VendorInvoice`, `Payment`, `Asset` |
| `ObjectId` | `bigint` | AK | no | Key of the object |
| `FileName` | `nvarchar(255)` | | no | File name |
| `ContentType` | `nvarchar(100)` | | no | MIME type |
| `FileSizeBytes` | `bigint` | | no | Size in bytes |
| `StorageUri` | `nvarchar(500)` | | no | Location in the document store |
| `Checksum` | `nvarchar(64)` | | no | SHA-256 of the content |
| `DocumentCategory` | `nvarchar(40)` | | yes | `SourceDocument`, `Approval`, `BankAdvice`, `Other` |
| `UploadedAt` | `datetime2(3)` | | no | Upload timestamp (UTC) |
| `UploadedBy` | `nvarchar(64)` | | no | Uploading user |
| `IsImmutable` | `bit` | | no | Attached to a posted document — cannot be removed |
