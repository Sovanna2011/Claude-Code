/* ============================================================================
   S/4HANA-inspired ERP - schema [mdm]
   Business Partner and master data (24 tables)

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* mdm.Address - Central address record (reference: ADRC) */
IF OBJECT_ID(N'mdm.Address', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[Address]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [AddressNumber]     nvarchar(10) NOT NULL,                                                                    -- Address number
        [Title]             nvarchar(4) NULL,                                                                         -- Form of address
        [Name1]             nvarchar(40) NULL,                                                                        -- Name line 1
        [Name2]             nvarchar(40) NULL,                                                                        -- Name line 2
        [Street]            nvarchar(60) NULL,                                                                        -- Street
        [HouseNumber]       nvarchar(10) NULL,                                                                        -- House number
        [StreetSupplement1] nvarchar(40) NULL,                                                                        -- Street line 2
        [StreetSupplement2] nvarchar(40) NULL,                                                                        -- Street line 3
        [District]          nvarchar(40) NULL,                                                                        -- District / khan / sangkat
        [City]              nvarchar(40) NULL,                                                                        -- City
        [PostalCode]        nvarchar(10) NULL,                                                                        -- Postal code
        [PoBox]             nvarchar(10) NULL,                                                                        -- PO box
        [PoBoxPostalCode]   nvarchar(10) NULL,                                                                        -- PO box postal code
        [RegionCode]        nvarchar(3) NULL,                                                                         -- Region / province
        [CountryCode]       nvarchar(3) NOT NULL,                                                                     -- Country
        [LanguageCode]      nvarchar(2) NOT NULL,                                                                     -- Address language
        [TimeZoneId]        nvarchar(64) NULL,                                                                        -- Time zone
        [Latitude]          decimal(9,6) NULL,                                                                        -- Geo latitude
        [Longitude]         decimal(9,6) NULL,                                                                        -- Geo longitude
        [FormattedAddress]  nvarchar(500) NULL,                                                                       -- Address rendered in the country format
        [ValidFrom]         date NOT NULL,                                                                            -- First day the record is valid
        [ValidTo]           date NOT NULL,                                                                            -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_mdm_Address_CreatedAt] DEFAULT (SYSUTCDATETIME()),   -- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_mdm_Address_IsActive] DEFAULT (1),                            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_Address] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_Address] UNIQUE ([TenantId], [AddressNumber])
    );
END
GO

/* mdm.Bank - Bank master (reference: BNKA) */
IF OBJECT_ID(N'mdm.Bank', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[Bank]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [BankCountryCode]     nvarchar(3) NOT NULL,                                                                   -- Bank country
        [BankKey]             nvarchar(15) NOT NULL,                                                                  -- Bank key / routing number
        [BankName]            nvarchar(60) NOT NULL,                                                                  -- Bank name
        [BankBranch]          nvarchar(40) NULL,                                                                      -- Branch
        [Street]              nvarchar(60) NULL,                                                                      -- Street
        [City]                nvarchar(40) NULL,                                                                      -- City
        [PostalCode]          nvarchar(10) NULL,                                                                      -- Postal code
        [SwiftCode]           nvarchar(11) NULL,                                                                      -- SWIFT / BIC
        [BankGroup]           nvarchar(2) NULL,                                                                       -- Bank group
        [IsBlocked]           bit NOT NULL CONSTRAINT [DF_mdm_Bank_IsBlocked] DEFAULT (0),                            -- Blocked
        [IsMarkedForDeletion] bit NOT NULL CONSTRAINT [DF_mdm_Bank_IsMarkedForDeletion] DEFAULT (0),                  -- Deletion flag
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_mdm_Bank_CreatedAt] DEFAULT (SYSUTCDATETIME()),    -- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_mdm_Bank_IsActive] DEFAULT (1),                             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_Bank] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_Bank] UNIQUE ([TenantId], [BankCountryCode], [BankKey])
    );
END
GO

/* mdm.BusinessPartner - Central business partner (general data) (reference: BUT000) */
IF OBJECT_ID(N'mdm.BusinessPartner', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartner]
    (
        [Id]                        bigint IDENTITY(1,1) NOT NULL,                                                    -- Surrogate key
        [TenantId]                  int NOT NULL,                                                                     -- Owning tenant - every query is filtered by it
        [PartnerNumber]             nvarchar(10) NOT NULL,                                                            -- Business partner number
        [PartnerCategory]           nvarchar(1) NOT NULL,                                                             -- 1 Person, 2 Organization, 3 Group
        [BusinessPartnerGroupId]    bigint NOT NULL,                                                                  -- BP grouping - drives the number range
        [PartnerType]               nvarchar(4) NULL,                                                                 -- Additional classification
        [Title]                     nvarchar(4) NULL,                                                                 -- Form of address key
        [Name1]                     nvarchar(40) NULL,                                                                -- Organization name 1 / last name
        [Name2]                     nvarchar(40) NULL,                                                                -- Organization name 2 / first name
        [Name3]                     nvarchar(40) NULL,                                                                -- Name line 3
        [Name4]                     nvarchar(40) NULL,                                                                -- Name line 4
        [FullName]                  nvarchar(160) NOT NULL,                                                           -- Formatted display name (computed on save)
        [SearchTerm1]               nvarchar(20) NULL,                                                                -- Search term 1
        [SearchTerm2]               nvarchar(20) NULL,                                                                -- Search term 2
        [FirstName]                 nvarchar(40) NULL,                                                                -- First name (person)
        [LastName]                  nvarchar(40) NULL,                                                                -- Last name (person)
        [MiddleName]                nvarchar(40) NULL,                                                                -- Middle name (person)
        [NickName]                  nvarchar(40) NULL,                                                                -- Nickname
        [DateOfBirth]               date NULL,                                                                        -- Date of birth (person)
        [PlaceOfBirth]              nvarchar(40) NULL,                                                                -- Place of birth
        [Gender]                    nvarchar(1) NULL,                                                                 -- 1 male, 2 female, 3 not specified
        [MaritalStatus]             nvarchar(1) NULL,                                                                 -- Marital status key
        [NationalityCountryCode]    nvarchar(3) NULL,                                                                 -- Nationality
        [LanguageCode]              nvarchar(2) NOT NULL,                                                             -- Correspondence language
        [IndustrySector]            nvarchar(10) NULL,                                                                -- Industry key
        [LegalForm]                 nvarchar(2) NULL,                                                                 -- Legal form of the organisation
        [LegalEntityCode]           nvarchar(2) NULL,                                                                 -- Legal entity classification
        [RegistrationNumber]        nvarchar(20) NULL,                                                                -- Commercial register number
        [RegistrationCountryCode]   nvarchar(3) NULL,                                                                 -- Country of registration
        [RegistrationDate]          date NULL,                                                                        -- Date of registration
        [FoundationDate]            date NULL,                                                                        -- Foundation / establishment date
        [LiquidationDate]           date NULL,                                                                        -- Liquidation date
        [EmployeeCount]             int NULL,                                                                         -- Number of employees
        [AnnualRevenue]             decimal(19,4) NULL,                                                               -- Annual revenue
        [AnnualRevenueCurrencyCode] nvarchar(5) NULL,                                                                 -- Currency of the revenue figure
        [DefaultAddressId]          bigint NULL,                                                                      -- Standard address
        [Status]                    nvarchar(20) NOT NULL,                                                            -- Active, Blocked, MarkedForDeletion, Archived
        [IsCentralBlocked]          bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartner_IsCentralBlocked] DEFAULT (0),    -- Central posting block
        [IsMarkedForDeletion]       bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartner_IsMarkedForDeletion] DEFAULT (0), -- Central deletion flag
        [IsIntercompany]            bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartner_IsIntercompany] DEFAULT (0),      -- Group company partner
        [TradingPartnerCompany]     nvarchar(6) NULL,                                                                 -- Company key for intercompany elimination
        [ExternalPartnerNumber]     nvarchar(20) NULL,                                                                -- Number in a legacy or external system
        [ValidFrom]                 date NOT NULL,                                                                    -- First day the record is valid
        [ValidTo]                   date NOT NULL,                                                                    -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]                 datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartner_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                 nvarchar(64) NOT NULL,                                                            -- Creating user name
        [ModifiedAt]                datetime2(3) NULL,                                                                -- Last change timestamp (UTC)
        [ModifiedBy]                nvarchar(64) NULL,                                                                -- Last changing user name
        [RowVersion]                rowversion NOT NULL,                                                              -- Optimistic concurrency token
        [IsActive]                  bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartner_IsActive] DEFAULT (1),            -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartner] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartner] UNIQUE ([TenantId], [PartnerNumber])
    );
END
GO

/* mdm.BusinessPartnerAddress - Address usage of a partner (reference: BUT020) */
IF OBJECT_ID(N'mdm.BusinessPartnerAddress', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerAddress]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [BusinessPartnerId] bigint NOT NULL,                                                                          -- Business partner
        [AddressId]         bigint NOT NULL,                                                                          -- Address
        [AddressUsage]      nvarchar(10) NOT NULL,                                                                    -- STANDARD, BILLTO, SHIPTO, DELIVERY, HOME, WORK
        [IsStandardAddress] bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerAddress_IsStandardAddress] DEFAULT (0),    -- Standard address for the usage
        [ValidFrom]         date NOT NULL,                                                                            -- First day the record is valid
        [ValidTo]           date NOT NULL,                                                                            -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerAddress_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerAddress_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerAddress] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerAddress] UNIQUE ([TenantId], [BusinessPartnerId], [AddressId], [AddressUsage])
    );
END
GO

/* mdm.BusinessPartnerAttachment - Document attached to a partner */
IF OBJECT_ID(N'mdm.BusinessPartnerAttachment', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerAttachment]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [BusinessPartnerId] bigint NOT NULL,                                                                          -- Business partner
        [FileName]          nvarchar(255) NOT NULL,                                                                   -- Original file name
        [ContentType]       nvarchar(100) NOT NULL,                                                                   -- MIME type
        [FileSizeBytes]     bigint NOT NULL,                                                                          -- Size in bytes
        [StorageUri]        nvarchar(500) NOT NULL,                                                                   -- Location in the document store
        [DocumentCategory]  nvarchar(40) NULL,                                                                        -- Contract, TaxCertificate, Registration, Other
        [Checksum]          nvarchar(64) NOT NULL,                                                                    -- SHA-256 of the content
        [UploadedAt]        datetime2(3) NOT NULL,                                                                    -- Upload timestamp (UTC)
        [UploadedBy]        nvarchar(64) NOT NULL,                                                                    -- Uploading user
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerAttachment_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerAttachment_IsActive] DEFAULT (1),          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerAttachment] PRIMARY KEY CLUSTERED ([Id])
    );
END
GO

/* mdm.BusinessPartnerBank - Bank details of a partner (reference: BUT0BK) */
IF OBJECT_ID(N'mdm.BusinessPartnerBank', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerBank]
    (
        [Id]                       bigint IDENTITY(1,1) NOT NULL,                                                     -- Surrogate key
        [TenantId]                 int NOT NULL,                                                                      -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]        bigint NOT NULL,                                                                   -- Business partner
        [BankDetailId]             nvarchar(4) NOT NULL,                                                              -- Bank details id, e.g. 0001
        [BankCountryCode]          nvarchar(3) NOT NULL,                                                              -- Bank country
        [BankKey]                  nvarchar(15) NOT NULL,                                                             -- Bank key / routing code
        [BankAccountNumber]        nvarchar(18) NULL,                                                                 -- Account number (masked in the browser)
        [BankAccountHolder]        nvarchar(60) NULL,                                                                 -- Account holder name
        [Iban]                     nvarchar(34) NULL,                                                                 -- IBAN
        [SwiftCode]                nvarchar(11) NULL,                                                                 -- SWIFT / BIC
        [BankControlKey]           nvarchar(2) NULL,                                                                  -- Bank control key
        [CurrencyCode]             nvarchar(5) NULL,                                                                  -- Account currency
        [IsDefaultForPayment]      bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerBank_IsDefaultForPayment] DEFAULT (0),-- Default account for payments
        [PaymentMethodRestriction] nvarchar(10) NULL,                                                                 -- Payment methods allowed on this account
        [IsVerified]               bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerBank_IsVerified] DEFAULT (0),       -- Verified by a second user (maker-checker)
        [VerifiedBy]               nvarchar(64) NULL,                                                                 -- Verifying user
        [VerifiedAt]               datetime2(3) NULL,                                                                 -- Verification timestamp (UTC)
        [ValidFrom]                date NOT NULL,                                                                     -- First day the record is valid
        [ValidTo]                  date NOT NULL,                                                                     -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]                datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerBank_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                nvarchar(64) NOT NULL,                                                             -- Creating user name
        [ModifiedAt]               datetime2(3) NULL,                                                                 -- Last change timestamp (UTC)
        [ModifiedBy]               nvarchar(64) NULL,                                                                 -- Last changing user name
        [RowVersion]               rowversion NOT NULL,                                                               -- Optimistic concurrency token
        [IsActive]                 bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerBank_IsActive] DEFAULT (1),         -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerBank] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerBank] UNIQUE ([TenantId], [BusinessPartnerId], [BankDetailId])
    );
END
GO

/* mdm.BusinessPartnerCommunication - Communication channel of a partner or address (reference: ADR2 / ADR6) */
IF OBJECT_ID(N'mdm.BusinessPartnerCommunication', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerCommunication]
    (
        [Id]                 bigint IDENTITY(1,1) NOT NULL,                                                           -- Surrogate key
        [TenantId]           int NOT NULL,                                                                            -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]  bigint NOT NULL,                                                                         -- Business partner
        [AddressId]          bigint NULL,                                                                             -- Address the channel belongs to
        [CommunicationType]  nvarchar(10) NOT NULL,                                                                   -- PHONE, MOBILE, FAX, EMAIL, WEB, TELEX
        [SequenceNumber]     int NOT NULL,                                                                            -- Sequence within the type
        [CountryDialCode]    nvarchar(5) NULL,                                                                        -- Country dialling code
        [Value]              nvarchar(241) NOT NULL,                                                                  -- Number, address or URL
        [Extension]          nvarchar(10) NULL,                                                                       -- Telephone extension
        [IsDefault]          bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCommunication_IsDefault] DEFAULT (0),     -- Standard channel of the type
        [IsMarketingAllowed] bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCommunication_IsMarketingAllowed] DEFAULT (0),-- Consent for marketing contact
        [Notes]              nvarchar(255) NULL,                                                                      -- Notes
        [ValidFrom]          date NOT NULL,                                                                           -- First day the record is valid
        [ValidTo]            date NOT NULL,                                                                           -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]          datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCommunication_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]          nvarchar(64) NOT NULL,                                                                   -- Creating user name
        [ModifiedAt]         datetime2(3) NULL,                                                                       -- Last change timestamp (UTC)
        [ModifiedBy]         nvarchar(64) NULL,                                                                       -- Last changing user name
        [RowVersion]         rowversion NOT NULL,                                                                     -- Optimistic concurrency token
        [IsActive]           bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCommunication_IsActive] DEFAULT (1),      -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerCommunication] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerCommunication] UNIQUE ([TenantId], [BusinessPartnerId], [CommunicationType], [SequenceNumber])
    );
END
GO

/* mdm.BusinessPartnerCompanyCode - Company code segment (customer and vendor) (reference: KNB1 / LFB1) */
IF OBJECT_ID(N'mdm.BusinessPartnerCompanyCode', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerCompanyCode]
    (
        [Id]                             bigint IDENTITY(1,1) NOT NULL,                                               -- Surrogate key
        [TenantId]                       int NOT NULL,                                                                -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]              bigint NOT NULL,                                                             -- Business partner
        [CompanyCodeId]                  bigint NOT NULL,                                                             -- Company code
        [RoleCategory]                   nvarchar(20) NOT NULL,                                                       -- Customer or Vendor - one segment per side
        [ReconciliationGLAccountId]      bigint NOT NULL,                                                             -- Reconciliation account - must be flagged as such
        [AlternativePayerPayeeId]        bigint NULL,                                                                 -- Alternative payer / payee partner
        [HeadOfficePartnerId]            bigint NULL,                                                                 -- Head office for branch accounting
        [SortKey]                        nvarchar(3) NULL,                                                            -- Allocation (assignment) field rule
        [PlanningGroup]                  nvarchar(10) NULL,                                                           -- Cash management planning group
        [PaymentTermsId]                 bigint NULL,                                                                 -- Payment terms
        [PaymentMethods]                 nvarchar(10) NULL,                                                           -- Permitted payment methods
        [PaymentBlockReason]             nvarchar(1) NULL,                                                            -- Payment block key
        [IsPostingBlocked]               bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCompanyCode_IsPostingBlocked] DEFAULT (0),-- Posting block for this company code
        [IsMarkedForDeletion]            bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCompanyCode_IsMarkedForDeletion] DEFAULT (0),-- Deletion flag for this company code
        [HouseBankId]                    bigint NULL,                                                                 -- House bank used for payments
        [PaymentGroupingKey]             nvarchar(2) NULL,                                                            -- Grouping key for the payment run
        [IsIndividualPayment]            bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCompanyCode_IsIndividualPayment] DEFAULT (0),-- Pay each item separately
        [ToleranceGroupId]               bigint NULL,                                                                 -- Tolerance group
        [DunningProcedureId]             bigint NULL,                                                                 -- Dunning procedure
        [DunningRecipientPartnerId]      bigint NULL,                                                                 -- Alternative dunning recipient
        [DunningBlockReason]             nvarchar(1) NULL,                                                            -- Dunning block
        [LastDunningDate]                date NULL,                                                                   -- Date of the last dunning run
        [DunningLevel]                   tinyint NULL,                                                                -- Current dunning level
        [DunningClerk]                   nvarchar(2) NULL,                                                            -- Dunning clerk
        [AccountingClerk]                nvarchar(2) NULL,                                                            -- Accounting clerk
        [WithholdingTaxCodeId]           bigint NULL,                                                                 -- Withholding tax code
        [IsWithholdingTaxExempt]         bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCompanyCode_IsWithholdingTaxExempt] DEFAULT (0),-- Exempt from withholding tax
        [WithholdingTaxExemptionNumber]  nvarchar(20) NULL,                                                           -- Exemption certificate number
        [WithholdingTaxExemptionValidTo] date NULL,                                                                   -- Certificate expiry
        [IsClearingWithVendorAllowed]    bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCompanyCode_IsClearingWithVendorAllowed] DEFAULT (0),-- Clearing between customer and vendor side allowed
        [ClearingPartnerId]              bigint NULL,                                                                 -- Partner used for cross-clearing
        [InterestCalculationIndicator]   nvarchar(2) NULL,                                                            -- Interest calculation indicator
        [LastInterestRunDate]            date NULL,                                                                   -- Last interest calculation
        [CorrespondenceType]             nvarchar(20) NULL,                                                           -- Default correspondence type
        [LocalCurrencyCode]              nvarchar(5) NULL,                                                            -- Account currency, if not the company code currency
        [CreatedAt]                      datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCompanyCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                      nvarchar(64) NOT NULL,                                                       -- Creating user name
        [ModifiedAt]                     datetime2(3) NULL,                                                           -- Last change timestamp (UTC)
        [ModifiedBy]                     nvarchar(64) NULL,                                                           -- Last changing user name
        [RowVersion]                     rowversion NOT NULL,                                                         -- Optimistic concurrency token
        [IsActive]                       bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCompanyCode_IsActive] DEFAULT (1),-- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerCompanyCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerCompanyCode] UNIQUE ([TenantId], [BusinessPartnerId], [CompanyCodeId], [RoleCategory])
    );
END
GO

/* mdm.BusinessPartnerCreditProfile - Credit management segment (reference: UKMBP_CMS_SGM) */
IF OBJECT_ID(N'mdm.BusinessPartnerCreditProfile', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerCreditProfile]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]       bigint NOT NULL,                                                                    -- Business partner
        [CreditControlAreaId]     bigint NOT NULL,                                                                    -- Credit control area
        [CreditLimit]             decimal(19,4) NOT NULL,                                                             -- Credit limit
        [CreditLimitCurrencyCode] nvarchar(5) NOT NULL,                                                               -- Limit currency
        [CreditExposure]          decimal(19,4) NOT NULL,                                                             -- Current exposure (open items + orders)
        [CreditLimitUsedPercent]  decimal(9,4) NOT NULL,                                                              -- Utilisation in percent
        [RiskCategory]            nvarchar(3) NULL,                                                                   -- Risk category
        [CreditRating]            nvarchar(10) NULL,                                                                  -- Internal or external rating
        [RatingAgency]            nvarchar(20) NULL,                                                                  -- Rating source
        [CreditLimitValidTo]      date NULL,                                                                          -- Limit expiry
        [LastReviewDate]          date NULL,                                                                          -- Last credit review
        [NextReviewDate]          date NULL,                                                                          -- Next scheduled review
        [IsCreditBlocked]         bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCreditProfile_IsCreditBlocked] DEFAULT (0),-- Credit block active
        [BlockReason]             nvarchar(255) NULL,                                                                 -- Reason for the block
        [PaymentBehaviourIndex]   decimal(9,4) NULL,                                                                  -- Average days beyond terms
        [HighestDunningLevel]     tinyint NULL,                                                                       -- Highest dunning level reached
        [OldestOpenItemDate]      date NULL,                                                                          -- Date of the oldest open item
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCreditProfile_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCreditProfile_IsActive] DEFAULT (1), -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerCreditProfile] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerCreditProfile] UNIQUE ([TenantId], [BusinessPartnerId], [CreditControlAreaId])
    );
END
GO

/* mdm.BusinessPartnerCustomer - Customer role - general customer data (reference: KNA1) */
IF OBJECT_ID(N'mdm.BusinessPartnerCustomer', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerCustomer]
    (
        [Id]                     bigint IDENTITY(1,1) NOT NULL,                                                       -- Surrogate key
        [TenantId]               int NOT NULL,                                                                        -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]      bigint NOT NULL,                                                                     -- Business partner
        [CustomerNumber]         nvarchar(10) NOT NULL,                                                               -- Customer account number
        [CustomerAccountGroupId] bigint NOT NULL,                                                                     -- Customer account group
        [CustomerClassification] nvarchar(2) NULL,                                                                    -- Customer classification
        [IndustryKey]            nvarchar(4) NULL,                                                                    -- Industry
        [CustomerGroup]          nvarchar(2) NULL,                                                                    -- Customer group for statistics
        [NielsenIndicator]       nvarchar(2) NULL,                                                                    -- Regional market indicator
        [TaxClassification]      nvarchar(1) NULL,                                                                    -- Tax classification of the customer
        [VatRegistrationNumber]  nvarchar(20) NULL,                                                                   -- VAT registration number
        [IsOneTimeCustomer]      bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCustomer_IsOneTimeCustomer] DEFAULT (0),-- One-time account
        [IsOrderBlocked]         bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCustomer_IsOrderBlocked] DEFAULT (0), -- Order block
        [IsDeliveryBlocked]      bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCustomer_IsDeliveryBlocked] DEFAULT (0),-- Delivery block
        [IsBillingBlocked]       bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCustomer_IsBillingBlocked] DEFAULT (0),-- Billing block
        [IsPostingBlocked]       bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCustomer_IsPostingBlocked] DEFAULT (0),-- Central posting block
        [IsMarkedForDeletion]    bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCustomer_IsMarkedForDeletion] DEFAULT (0),-- Deletion flag
        [TransportationZone]     nvarchar(10) NULL,                                                                   -- Transportation zone
        [AuthorizationGroup]     nvarchar(4) NULL,                                                                    -- Authorization group
        [CreatedAt]              datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCustomer_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]              nvarchar(64) NOT NULL,                                                               -- Creating user name
        [ModifiedAt]             datetime2(3) NULL,                                                                   -- Last change timestamp (UTC)
        [ModifiedBy]             nvarchar(64) NULL,                                                                   -- Last changing user name
        [RowVersion]             rowversion NOT NULL,                                                                 -- Optimistic concurrency token
        [IsActive]               bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerCustomer_IsActive] DEFAULT (1),       -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerCustomer] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerCustomer] UNIQUE ([TenantId], [BusinessPartnerId], [CustomerNumber])
    );
END
GO

/* mdm.BusinessPartnerGroup - BP grouping - number range and screen defaults (reference: TB001 / T077D) */
IF OBJECT_ID(N'mdm.BusinessPartnerGroup', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerGroup]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [GroupCode]           nvarchar(4) NOT NULL,                                                                   -- Grouping key
        [Name]                nvarchar(60) NOT NULL,                                                                  -- Description
        [NumberRangeObjectId] bigint NOT NULL,                                                                        -- Number range object
        [NumberRangeCode]     nvarchar(2) NOT NULL,                                                                   -- Number range interval
        [IsExternalNumbering] bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerGroup_IsExternalNumbering] DEFAULT (0),  -- Number entered by the user
        [PartnerCategory]     nvarchar(1) NULL,                                                                       -- Restricted to this category
        [IsOneTimeAccount]    bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerGroup_IsOneTimeAccount] DEFAULT (0),     -- One-time account (CpD)
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerGroup_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        [IsActive]            bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerGroup_IsActive] DEFAULT (1),             -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerGroup] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerGroup] UNIQUE ([TenantId], [GroupCode])
    );
END
GO

/* mdm.BusinessPartnerIdentification - Identification numbers (passport, licence, registration) (reference: BUT0ID) */
IF OBJECT_ID(N'mdm.BusinessPartnerIdentification', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerIdentification]
    (
        [Id]                   bigint IDENTITY(1,1) NOT NULL,                                                         -- Surrogate key
        [TenantId]             int NOT NULL,                                                                          -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]    bigint NOT NULL,                                                                       -- Business partner
        [IdentificationType]   nvarchar(6) NOT NULL,                                                                  -- PASSPORT, NATID, DRVLIC, BUSLIC
        [IdentificationNumber] nvarchar(60) NOT NULL,                                                                 -- Number (masked for unauthorised users)
        [IssuingInstitution]   nvarchar(40) NULL,                                                                     -- Issuing authority
        [IssuingCountryCode]   nvarchar(3) NULL,                                                                      -- Country of issue
        [IssuingRegionCode]    nvarchar(3) NULL,                                                                      -- Region of issue
        [IssueDate]            date NULL,                                                                             -- Date of issue
        [ExpiryDate]           date NULL,                                                                             -- Expiry date
        [IsPersonalData]       bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerIdentification_IsPersonalData] DEFAULT (0),-- Subject to masking and retention rules
        [ValidFrom]            date NOT NULL,                                                                         -- First day the record is valid
        [ValidTo]              date NOT NULL,                                                                         -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]            datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerIdentification_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]            nvarchar(64) NOT NULL,                                                                 -- Creating user name
        [ModifiedAt]           datetime2(3) NULL,                                                                     -- Last change timestamp (UTC)
        [ModifiedBy]           nvarchar(64) NULL,                                                                     -- Last changing user name
        [RowVersion]           rowversion NOT NULL,                                                                   -- Optimistic concurrency token
        [IsActive]             bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerIdentification_IsActive] DEFAULT (1),   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerIdentification] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerIdentification] UNIQUE ([TenantId], [BusinessPartnerId], [IdentificationType], [IdentificationNumber])
    );
END
GO

/* mdm.BusinessPartnerPurchasingOrganization - Vendor purchasing organisation data (reference: LFM1) */
IF OBJECT_ID(N'mdm.BusinessPartnerPurchasingOrganization', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerPurchasingOrganization]
    (
        [Id]                                     bigint IDENTITY(1,1) NOT NULL,                                       -- Surrogate key
        [TenantId]                               int NOT NULL,                                                        -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]                      bigint NOT NULL,                                                     -- Business partner
        [PurchasingOrganizationId]               bigint NOT NULL,                                                     -- Purchasing organisation
        [PurchasingGroup]                        nvarchar(3) NULL,                                                    -- Purchasing group
        [OrderCurrencyCode]                      nvarchar(5) NOT NULL,                                                -- Order currency
        [PaymentTermsId]                         bigint NULL,                                                         -- Purchasing payment terms
        [Incoterms1]                             nvarchar(3) NULL,                                                    -- Incoterms part 1
        [Incoterms2]                             nvarchar(28) NULL,                                                   -- Incoterms part 2 (location)
        [MinimumOrderValue]                      decimal(19,4) NULL,                                                  -- Minimum order value
        [IsGoodsReceiptBasedInvoiceVerification] bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_IsGoodsReceiptBasedInvoiceVerification] DEFAULT (0),-- GR-based invoice verification
        [IsGoodsReceiptRequired]                 bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_IsGoodsReceiptRequired] DEFAULT (0),-- Goods receipt expected
        [IsInvoiceReceiptRequired]               bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_IsInvoiceReceiptRequired] DEFAULT (0),-- Invoice receipt expected
        [IsEvaluatedReceiptSettlement]           bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_IsEvaluatedReceiptSettlement] DEFAULT (0),-- ERS (self-billing)
        [IsAutomaticPurchaseOrderAllowed]        bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_IsAutomaticPurchaseOrderAllowed] DEFAULT (0),-- Automatic PO generation allowed
        [IsReturnsVendor]                        bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_IsReturnsVendor] DEFAULT (0),-- Returns vendor
        [SchemaGroup]                            nvarchar(2) NULL,                                                    -- Pricing schema group
        [PlannedDeliveryDays]                    int NULL,                                                            -- Planned delivery time
        [IsPurchasingBlocked]                    bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_IsPurchasingBlocked] DEFAULT (0),-- Block for this purchasing organisation
        [IsMarkedForDeletion]                    bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_IsMarkedForDeletion] DEFAULT (0),-- Deletion flag
        [CreatedAt]                              datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                              nvarchar(64) NOT NULL,                                               -- Creating user name
        [ModifiedAt]                             datetime2(3) NULL,                                                   -- Last change timestamp (UTC)
        [ModifiedBy]                             nvarchar(64) NULL,                                                   -- Last changing user name
        [RowVersion]                             rowversion NOT NULL,                                                 -- Optimistic concurrency token
        [IsActive]                               bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerPurchasingOrganization_IsActive] DEFAULT (1),-- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerPurchasingOrganization] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerPurchasingOrganization] UNIQUE ([TenantId], [BusinessPartnerId], [PurchasingOrganizationId])
    );
END
GO

/* mdm.BusinessPartnerRelationship - Validity-dated relationship between two partners (reference: BUT050) */
IF OBJECT_ID(N'mdm.BusinessPartnerRelationship', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerRelationship]
    (
        [Id]                      bigint IDENTITY(1,1) NOT NULL,                                                      -- Surrogate key
        [TenantId]                int NOT NULL,                                                                       -- Owning tenant - every query is filtered by it
        [RelationshipNumber]      nvarchar(12) NOT NULL,                                                              -- Relationship number
        [RelationshipCategory]    nvarchar(6) NOT NULL,                                                               -- PARENT, SUBSID, CONTACT, EMPLOY, SOLDTO, SHIPTO, BILLTO, PAYER, SUPPL, RELCO, ICOMP
        [SourceBusinessPartnerId] bigint NOT NULL,                                                                    -- Partner 1
        [TargetBusinessPartnerId] bigint NOT NULL,                                                                    -- Partner 2
        [IsDirectional]           bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRelationship_IsDirectional] DEFAULT (0),-- Relationship has a direction
        [RelationshipRole]        nvarchar(40) NULL,                                                                  -- Role of the target in the relationship
        [ShareholdingPercent]     decimal(9,4) NULL,                                                                  -- Ownership percentage
        [IsStandard]              bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRelationship_IsStandard] DEFAULT (0),-- Default partner of this category
        [DepartmentText]          nvarchar(40) NULL,                                                                  -- Department (contact person)
        [FunctionText]            nvarchar(40) NULL,                                                                  -- Function (contact person)
        [Notes]                   nvarchar(255) NULL,                                                                 -- Notes
        [ValidFrom]               date NOT NULL,                                                                      -- First day the record is valid
        [ValidTo]                 date NOT NULL,                                                                      -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]               datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRelationship_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]               nvarchar(64) NOT NULL,                                                              -- Creating user name
        [ModifiedAt]              datetime2(3) NULL,                                                                  -- Last change timestamp (UTC)
        [ModifiedBy]              nvarchar(64) NULL,                                                                  -- Last changing user name
        [RowVersion]              rowversion NOT NULL,                                                                -- Optimistic concurrency token
        [IsActive]                bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRelationship_IsActive] DEFAULT (1),  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerRelationship] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerRelationship] UNIQUE ([TenantId], [RelationshipNumber])
    );
END
GO

/* mdm.BusinessPartnerRole - Catalogue of available BP roles (reference: TB003) */
IF OBJECT_ID(N'mdm.BusinessPartnerRole', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerRole]
    (
        [Id]                             bigint IDENTITY(1,1) NOT NULL,                                               -- Surrogate key
        [TenantId]                       int NOT NULL,                                                                -- Owning tenant - every query is filtered by it
        [RoleCode]                       nvarchar(6) NOT NULL,                                                        -- Role key, e.g. 000000 general, FLCU00/FLCU01 customer, FLVN00/FLVN01 vendor
        [Name]                           nvarchar(60) NOT NULL,                                                       -- Role name
        [RoleCategory]                   nvarchar(20) NOT NULL,                                                       -- General, Customer, FICustomer, Vendor, FIVendor, Employee, ContactPerson, Bank, Intercompany
        [RequiresCompanyCodeData]        bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRole_RequiresCompanyCodeData] DEFAULT (0),-- Company code segment required
        [RequiresSalesArea]              bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRole_RequiresSalesArea] DEFAULT (0),-- Sales area segment required
        [RequiresPurchasingOrganization] bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRole_RequiresPurchasingOrganization] DEFAULT (0),-- Purchasing segment required
        [SyncTargetEntity]               nvarchar(40) NULL,                                                           -- Entity created by role synchronisation
        [IsStandardRole]                 bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRole_IsStandardRole] DEFAULT (0),-- Delivered role (not customer-defined)
        [DisplayOrder]                   int NOT NULL,                                                                -- Order in the role list
        [CreatedAt]                      datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRole_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                      nvarchar(64) NOT NULL,                                                       -- Creating user name
        [ModifiedAt]                     datetime2(3) NULL,                                                           -- Last change timestamp (UTC)
        [ModifiedBy]                     nvarchar(64) NULL,                                                           -- Last changing user name
        [RowVersion]                     rowversion NOT NULL,                                                         -- Optimistic concurrency token
        [IsActive]                       bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRole_IsActive] DEFAULT (1),   -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerRole] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerRole] UNIQUE ([TenantId], [RoleCode])
    );
END
GO

/* mdm.BusinessPartnerRoleAssignment - Roles held by a partner, with validity (reference: BUT100) */
IF OBJECT_ID(N'mdm.BusinessPartnerRoleAssignment', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerRoleAssignment]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]     bigint NOT NULL,                                                                      -- Business partner
        [BusinessPartnerRoleId] bigint NOT NULL,                                                                      -- Role
        [IsSynchronized]        bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRoleAssignment_IsSynchronized] DEFAULT (0),-- Role-specific data has been created
        [SynchronizedAt]        datetime2(3) NULL,                                                                    -- Last synchronisation (UTC)
        [SyncStatus]            nvarchar(20) NOT NULL,                                                                -- Pending, Completed, Failed
        [SyncMessage]           nvarchar(255) NULL,                                                                   -- Last synchronisation message
        [ValidFrom]             date NOT NULL,                                                                        -- First day the record is valid
        [ValidTo]               date NOT NULL,                                                                        -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRoleAssignment_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerRoleAssignment_IsActive] DEFAULT (1),  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerRoleAssignment] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerRoleAssignment] UNIQUE ([TenantId], [BusinessPartnerId], [BusinessPartnerRoleId])
    );
END
GO

/* mdm.BusinessPartnerSalesArea - Customer sales area data (reference: KNVV) */
IF OBJECT_ID(N'mdm.BusinessPartnerSalesArea', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerSalesArea]
    (
        [Id]                         bigint IDENTITY(1,1) NOT NULL,                                                   -- Surrogate key
        [TenantId]                   int NOT NULL,                                                                    -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]          bigint NOT NULL,                                                                 -- Business partner
        [SalesAreaId]                bigint NOT NULL,                                                                 -- Sales organisation / channel / division
        [CustomerGroup]              nvarchar(2) NULL,                                                                -- Customer group
        [SalesDistrict]              nvarchar(6) NULL,                                                                -- Sales district
        [SalesOffice]                nvarchar(4) NULL,                                                                -- Sales office
        [SalesGroup]                 nvarchar(3) NULL,                                                                -- Sales group
        [PriceGroup]                 nvarchar(2) NULL,                                                                -- Price group
        [PriceListType]              nvarchar(2) NULL,                                                                -- Price list type
        [CustomerPricingProcedure]   nvarchar(1) NULL,                                                                -- Pricing procedure determination
        [CurrencyCode]               nvarchar(5) NOT NULL,                                                            -- Order currency
        [PaymentTermsId]             bigint NULL,                                                                     -- Sales payment terms
        [Incoterms1]                 nvarchar(3) NULL,                                                                -- Incoterms part 1
        [Incoterms2]                 nvarchar(28) NULL,                                                               -- Incoterms part 2 (location)
        [ShippingConditions]         nvarchar(2) NULL,                                                                -- Shipping conditions
        [DeliveryPriority]           nvarchar(2) NULL,                                                                -- Delivery priority
        [DeliveringPlantId]          bigint NULL,                                                                     -- Default delivering plant
        [IsCompleteDeliveryRequired] bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerSalesArea_IsCompleteDeliveryRequired] DEFAULT (0),-- Complete delivery required
        [PartialDeliveryPerItem]     nvarchar(1) NULL,                                                                -- Partial delivery indicator
        [MaxPartialDeliveries]       tinyint NULL,                                                                    -- Maximum partial deliveries
        [OrderCombinationAllowed]    bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerSalesArea_OrderCombinationAllowed] DEFAULT (0),-- Order combination permitted
        [AccountAssignmentGroup]     nvarchar(2) NULL,                                                                -- Account assignment group for revenue determination
        [TaxClassification]          nvarchar(1) NULL,                                                                -- Tax classification
        [IsOrderBlocked]             bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerSalesArea_IsOrderBlocked] DEFAULT (0),-- Sales area order block
        [IsMarkedForDeletion]        bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerSalesArea_IsMarkedForDeletion] DEFAULT (0),-- Deletion flag
        [CreatedAt]                  datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerSalesArea_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                  nvarchar(64) NOT NULL,                                                           -- Creating user name
        [ModifiedAt]                 datetime2(3) NULL,                                                               -- Last change timestamp (UTC)
        [ModifiedBy]                 nvarchar(64) NULL,                                                               -- Last changing user name
        [RowVersion]                 rowversion NOT NULL,                                                             -- Optimistic concurrency token
        [IsActive]                   bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerSalesArea_IsActive] DEFAULT (1),  -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerSalesArea] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerSalesArea] UNIQUE ([TenantId], [BusinessPartnerId], [SalesAreaId])
    );
END
GO

/* mdm.BusinessPartnerTaxNumber - Tax numbers of a partner (reference: BUT0TX) */
IF OBJECT_ID(N'mdm.BusinessPartnerTaxNumber', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerTaxNumber]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [BusinessPartnerId] bigint NOT NULL,                                                                          -- Business partner
        [TaxNumberCategory] nvarchar(6) NOT NULL,                                                                     -- Category, e.g. KH0 VAT TIN, EU0 VAT id
        [TaxNumber]         nvarchar(20) NOT NULL,                                                                    -- Tax number
        [CountryCode]       nvarchar(3) NOT NULL,                                                                     -- Issuing country
        [IsNaturalPerson]   bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerTaxNumber_IsNaturalPerson] DEFAULT (0),    -- Natural person indicator
        [IsValidated]       bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerTaxNumber_IsValidated] DEFAULT (0),        -- Format validated against the country rule
        [ValidatedAt]       datetime2(3) NULL,                                                                        -- Validation timestamp (UTC)
        [ValidFrom]         date NOT NULL,                                                                            -- First day the record is valid
        [ValidTo]           date NOT NULL,                                                                            -- Last day valid (9999-12-31 = open ended)
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerTaxNumber_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerTaxNumber_IsActive] DEFAULT (1),           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerTaxNumber] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerTaxNumber] UNIQUE ([TenantId], [BusinessPartnerId], [TaxNumberCategory], [TaxNumber])
    );
END
GO

/* mdm.BusinessPartnerVendor - Vendor / supplier role - general vendor data (reference: LFA1) */
IF OBJECT_ID(N'mdm.BusinessPartnerVendor', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[BusinessPartnerVendor]
    (
        [Id]                    bigint IDENTITY(1,1) NOT NULL,                                                        -- Surrogate key
        [TenantId]              int NOT NULL,                                                                         -- Owning tenant - every query is filtered by it
        [BusinessPartnerId]     bigint NOT NULL,                                                                      -- Business partner
        [VendorNumber]          nvarchar(10) NOT NULL,                                                                -- Vendor account number
        [VendorAccountGroupId]  bigint NOT NULL,                                                                      -- Vendor account group
        [VendorClassification]  nvarchar(2) NULL,                                                                     -- Vendor classification
        [IndustryKey]           nvarchar(4) NULL,                                                                     -- Industry
        [VatRegistrationNumber] nvarchar(20) NULL,                                                                    -- VAT registration number
        [TaxNumberType]         nvarchar(2) NULL,                                                                     -- Tax number type
        [IsOneTimeVendor]       bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerVendor_IsOneTimeVendor] DEFAULT (0),   -- One-time account
        [IsPurchasingBlocked]   bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerVendor_IsPurchasingBlocked] DEFAULT (0),-- Purchasing block
        [IsPostingBlocked]      bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerVendor_IsPostingBlocked] DEFAULT (0),  -- Central posting block
        [IsPaymentBlocked]      bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerVendor_IsPaymentBlocked] DEFAULT (0),  -- Central payment block
        [IsMarkedForDeletion]   bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerVendor_IsMarkedForDeletion] DEFAULT (0),-- Deletion flag
        [IsServiceProvider]     bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerVendor_IsServiceProvider] DEFAULT (0), -- Service provider (withholding tax relevance)
        [IsSubcontractor]       bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerVendor_IsSubcontractor] DEFAULT (0),   -- Subcontractor
        [QualityRating]         nvarchar(2) NULL,                                                                     -- Supplier quality rating
        [AuthorizationGroup]    nvarchar(4) NULL,                                                                     -- Authorization group
        [CreatedAt]             datetime2(3) NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerVendor_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]             nvarchar(64) NOT NULL,                                                                -- Creating user name
        [ModifiedAt]            datetime2(3) NULL,                                                                    -- Last change timestamp (UTC)
        [ModifiedBy]            nvarchar(64) NULL,                                                                    -- Last changing user name
        [RowVersion]            rowversion NOT NULL,                                                                  -- Optimistic concurrency token
        [IsActive]              bit NOT NULL CONSTRAINT [DF_mdm_BusinessPartnerVendor_IsActive] DEFAULT (1),          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_BusinessPartnerVendor] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_BusinessPartnerVendor] UNIQUE ([TenantId], [BusinessPartnerId], [VendorNumber])
    );
END
GO

/* mdm.GLAccount - G/L account - chart of accounts area (reference: SKA1) */
IF OBJECT_ID(N'mdm.GLAccount', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[GLAccount]
    (
        [Id]                         bigint IDENTITY(1,1) NOT NULL,                                                   -- Surrogate key
        [TenantId]                   int NOT NULL,                                                                    -- Owning tenant - every query is filtered by it
        [ChartOfAccountsId]          bigint NOT NULL,                                                                 -- Chart of accounts
        [GLAccount]                  nvarchar(10) NOT NULL,                                                           -- G/L account number
        [AccountGroupId]             bigint NOT NULL,                                                                 -- Account group
        [AccountType]                nvarchar(20) NOT NULL,                                                           -- BalanceSheet, PrimaryCost, SecondaryCost, NonOperatingExpenseRevenue, CashAccount
        [IsBalanceSheetAccount]      bit NOT NULL CONSTRAINT [DF_mdm_GLAccount_IsBalanceSheetAccount] DEFAULT (0),    -- Balance sheet account
        [IsProfitAndLossAccount]     bit NOT NULL CONSTRAINT [DF_mdm_GLAccount_IsProfitAndLossAccount] DEFAULT (0),   -- P&L account
        [IsReconciliationAccount]    bit NOT NULL CONSTRAINT [DF_mdm_GLAccount_IsReconciliationAccount] DEFAULT (0),  -- Reconciliation account - no direct posting
        [ReconciliationAccountType]  nvarchar(1) NULL,                                                                -- D customer, K vendor, A asset
        [RetainedEarningsAccountKey] nvarchar(2) NULL,                                                                -- P&L statement account type for carry-forward
        [GroupAccountNumber]         nvarchar(10) NULL,                                                               -- Group chart account
        [TradingPartnerRequired]     bit NOT NULL CONSTRAINT [DF_mdm_GLAccount_TradingPartnerRequired] DEFAULT (0),   -- Trading partner mandatory
        [IsBlockedForCreation]       bit NOT NULL CONSTRAINT [DF_mdm_GLAccount_IsBlockedForCreation] DEFAULT (0),     -- Blocked for company-code creation
        [IsBlockedForPosting]        bit NOT NULL CONSTRAINT [DF_mdm_GLAccount_IsBlockedForPosting] DEFAULT (0),      -- Blocked for posting
        [IsBlockedForPlanning]       bit NOT NULL CONSTRAINT [DF_mdm_GLAccount_IsBlockedForPlanning] DEFAULT (0),     -- Blocked for planning
        [IsMarkedForDeletion]        bit NOT NULL CONSTRAINT [DF_mdm_GLAccount_IsMarkedForDeletion] DEFAULT (0),      -- Deletion flag
        [SampleAccountNumber]        nvarchar(10) NULL,                                                               -- Sample account used as a template
        [FunctionalAreaId]           bigint NULL,                                                                     -- Default functional area
        [CostElementCategory]        nvarchar(2) NULL,                                                                -- CO cost element category (1, 11, 21, 41, 42, 43)
        [CreatedAt]                  datetime2(3) NOT NULL CONSTRAINT [DF_mdm_GLAccount_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                  nvarchar(64) NOT NULL,                                                           -- Creating user name
        [ModifiedAt]                 datetime2(3) NULL,                                                               -- Last change timestamp (UTC)
        [ModifiedBy]                 nvarchar(64) NULL,                                                               -- Last changing user name
        [RowVersion]                 rowversion NOT NULL,                                                             -- Optimistic concurrency token
        [IsActive]                   bit NOT NULL CONSTRAINT [DF_mdm_GLAccount_IsActive] DEFAULT (1),                 -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_GLAccount] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_GLAccount] UNIQUE ([TenantId], [ChartOfAccountsId], [GLAccount])
    );
END
GO

/* mdm.GLAccountCompanyCode - G/L account - company code area (reference: SKB1) */
IF OBJECT_ID(N'mdm.GLAccountCompanyCode', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[GLAccountCompanyCode]
    (
        [Id]                                    bigint IDENTITY(1,1) NOT NULL,                                        -- Surrogate key
        [TenantId]                              int NOT NULL,                                                         -- Owning tenant - every query is filtered by it
        [GLAccountId]                           bigint NOT NULL,                                                      -- G/L account
        [CompanyCodeId]                         bigint NOT NULL,                                                      -- Company code
        [AccountCurrencyCode]                   nvarchar(5) NOT NULL,                                                 -- Account currency
        [IsOnlyBalancesInLocalCurrency]         bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsOnlyBalancesInLocalCurrency] DEFAULT (0),-- Only local currency balances
        [TaxCategory]                           nvarchar(2) NULL,                                                     -- Tax category (-, +, *, tax code)
        [IsPostingWithoutTaxAllowed]            bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsPostingWithoutTaxAllowed] DEFAULT (0),-- Posting without tax code allowed
        [IsOpenItemManaged]                     bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsOpenItemManaged] DEFAULT (0),-- Open item management
        [IsLineItemDisplay]                     bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsLineItemDisplay] DEFAULT (0),-- Line item display
        [SortKey]                               nvarchar(3) NULL,                                                     -- Rule filling the allocation field
        [FieldStatusGroupId]                    bigint NOT NULL,                                                      -- Field status group
        [IsPostAutomaticallyOnly]               bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsPostAutomaticallyOnly] DEFAULT (0),-- Post automatically only
        [IsReconciliationAccountForAccountType] nvarchar(1) NULL,                                                     -- Reconciliation account type in this company code
        [IsRelevantToCashFlow]                  bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsRelevantToCashFlow] DEFAULT (0),-- Relevant to cash flow
        [HouseBankId]                           bigint NULL,                                                          -- House bank of a bank account
        [HouseBankAccountId]                    bigint NULL,                                                          -- House bank account
        [InterestCalculationIndicator]          nvarchar(2) NULL,                                                     -- Interest indicator
        [PlanningGroup]                         nvarchar(10) NULL,                                                    -- Cash management planning group
        [IsBlockedForPosting]                   bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsBlockedForPosting] DEFAULT (0),-- Blocked for posting in this company code
        [IsMarkedForDeletion]                   bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsMarkedForDeletion] DEFAULT (0),-- Deletion flag
        [AuthorizationGroup]                    nvarchar(4) NULL,                                                     -- Authorization group
        [CostCenterRequired]                    bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_CostCenterRequired] DEFAULT (0),-- Cost centre mandatory on postings
        [ProfitCenterRequired]                  bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_ProfitCenterRequired] DEFAULT (0),-- Profit centre mandatory on postings
        [IsInflationRelevant]                   bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsInflationRelevant] DEFAULT (0),-- Subject to inflation adjustment
        [ValuationGroup]                        nvarchar(4) NULL,                                                     -- Foreign currency valuation group
        [CreatedAt]                             datetime2(3) NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]                             nvarchar(64) NOT NULL,                                                -- Creating user name
        [ModifiedAt]                            datetime2(3) NULL,                                                    -- Last change timestamp (UTC)
        [ModifiedBy]                            nvarchar(64) NULL,                                                    -- Last changing user name
        [RowVersion]                            rowversion NOT NULL,                                                  -- Optimistic concurrency token
        [IsActive]                              bit NOT NULL CONSTRAINT [DF_mdm_GLAccountCompanyCode_IsActive] DEFAULT (1),-- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_GLAccountCompanyCode] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_GLAccountCompanyCode] UNIQUE ([TenantId], [GLAccountId], [CompanyCodeId])
    );
END
GO

/* mdm.GLAccountText - G/L account descriptions per language (reference: SKAT) */
IF OBJECT_ID(N'mdm.GLAccountText', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[GLAccountText]
    (
        [Id]           bigint IDENTITY(1,1) NOT NULL,                                                                 -- Surrogate key
        [TenantId]     int NOT NULL,                                                                                  -- Owning tenant - every query is filtered by it
        [GLAccountId]  bigint NOT NULL,                                                                               -- G/L account
        [LanguageCode] nvarchar(2) NOT NULL,                                                                          -- Language
        [ShortText]    nvarchar(20) NOT NULL,                                                                         -- Short text
        [LongText]     nvarchar(50) NOT NULL,                                                                         -- Long text
        [CreatedAt]    datetime2(3) NOT NULL CONSTRAINT [DF_mdm_GLAccountText_CreatedAt] DEFAULT (SYSUTCDATETIME()),  -- Creation timestamp (UTC)
        [CreatedBy]    nvarchar(64) NOT NULL,                                                                         -- Creating user name
        [ModifiedAt]   datetime2(3) NULL,                                                                             -- Last change timestamp (UTC)
        [ModifiedBy]   nvarchar(64) NULL,                                                                             -- Last changing user name
        [RowVersion]   rowversion NOT NULL,                                                                           -- Optimistic concurrency token
        [IsActive]     bit NOT NULL CONSTRAINT [DF_mdm_GLAccountText_IsActive] DEFAULT (1),                           -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_GLAccountText] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_GLAccountText] UNIQUE ([TenantId], [GLAccountId], [LanguageCode])
    );
END
GO

/* mdm.HouseBank - House bank of a company code (reference: T012) */
IF OBJECT_ID(N'mdm.HouseBank', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[HouseBank]
    (
        [Id]                bigint IDENTITY(1,1) NOT NULL,                                                            -- Surrogate key
        [TenantId]          int NOT NULL,                                                                             -- Owning tenant - every query is filtered by it
        [CompanyCodeId]     bigint NOT NULL,                                                                          -- Company code
        [HouseBankCode]     nvarchar(5) NOT NULL,                                                                     -- House bank key
        [BankId]            bigint NOT NULL,                                                                          -- Bank master record
        [Description]       nvarchar(60) NULL,                                                                        -- Description
        [PaymentFileFormat] nvarchar(20) NULL,                                                                        -- Payment file format used
        [IsDefault]         bit NOT NULL CONSTRAINT [DF_mdm_HouseBank_IsDefault] DEFAULT (0),                         -- Default house bank
        [CreatedAt]         datetime2(3) NOT NULL CONSTRAINT [DF_mdm_HouseBank_CreatedAt] DEFAULT (SYSUTCDATETIME()), -- Creation timestamp (UTC)
        [CreatedBy]         nvarchar(64) NOT NULL,                                                                    -- Creating user name
        [ModifiedAt]        datetime2(3) NULL,                                                                        -- Last change timestamp (UTC)
        [ModifiedBy]        nvarchar(64) NULL,                                                                        -- Last changing user name
        [RowVersion]        rowversion NOT NULL,                                                                      -- Optimistic concurrency token
        [IsActive]          bit NOT NULL CONSTRAINT [DF_mdm_HouseBank_IsActive] DEFAULT (1),                          -- Soft-delete / active flag (master + config only)
        CONSTRAINT [PK_mdm_HouseBank] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_HouseBank] UNIQUE ([TenantId], [CompanyCodeId], [HouseBankCode])
    );
END
GO

/* mdm.HouseBankAccount - Bank account of a house bank (reference: T012K) */
IF OBJECT_ID(N'mdm.HouseBankAccount', N'U') IS NULL
BEGIN
    CREATE TABLE [mdm].[HouseBankAccount]
    (
        [Id]                  bigint IDENTITY(1,1) NOT NULL,                                                          -- Surrogate key
        [TenantId]            int NOT NULL,                                                                           -- Owning tenant - every query is filtered by it
        [HouseBankId]         bigint NOT NULL,                                                                        -- House bank
        [AccountId]           nvarchar(5) NOT NULL,                                                                   -- Account id
        [BankAccountNumber]   nvarchar(18) NULL,                                                                      -- Account number
        [Iban]                nvarchar(34) NULL,                                                                      -- IBAN
        [CurrencyCode]        nvarchar(5) NOT NULL,                                                                   -- Account currency
        [GLAccountId]         bigint NOT NULL,                                                                        -- Bank G/L account
        [ClearingGLAccountId] bigint NULL,                                                                            -- Bank clearing account
        [AccountHolder]       nvarchar(60) NULL,                                                                      -- Account holder
        [OverdraftLimit]      decimal(19,4) NULL,                                                                     -- Overdraft limit
        [IsActive]            bit NOT NULL CONSTRAINT [DF_mdm_HouseBankAccount_IsActive] DEFAULT (1),                 -- Active
        [CreatedAt]           datetime2(3) NOT NULL CONSTRAINT [DF_mdm_HouseBankAccount_CreatedAt] DEFAULT (SYSUTCDATETIME()),-- Creation timestamp (UTC)
        [CreatedBy]           nvarchar(64) NOT NULL,                                                                  -- Creating user name
        [ModifiedAt]          datetime2(3) NULL,                                                                      -- Last change timestamp (UTC)
        [ModifiedBy]          nvarchar(64) NULL,                                                                      -- Last changing user name
        [RowVersion]          rowversion NOT NULL,                                                                    -- Optimistic concurrency token
        CONSTRAINT [PK_mdm_HouseBankAccount] PRIMARY KEY CLUSTERED ([Id]),
        CONSTRAINT [UQ_mdm_HouseBankAccount] UNIQUE ([TenantId], [HouseBankId], [AccountId])
    );
END
GO
