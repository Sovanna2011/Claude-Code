/* ============================================================================
   S/4HANA-inspired ERP - secondary indexes
   Indexes for the access paths marked IX in the catalogue

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

/* Primary and business keys are created with the tables. The indexes
   below cover the documented read paths (posting date, document number,
   partner, status, correlation id). Add further indexes for foreign key
   columns once a real workload has been measured - indexing all 838 of
   them up front costs more on write than it returns on read. */

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_Branch_ValidFrom'
               AND object_id = OBJECT_ID(N'org.Branch'))
    CREATE NONCLUSTERED INDEX [IX_org_Branch_ValidFrom]
        ON [org].[Branch] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_Branch_ValidTo'
               AND object_id = OBJECT_ID(N'org.Branch'))
    CREATE NONCLUSTERED INDEX [IX_org_Branch_ValidTo]
        ON [org].[Branch] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_ControllingAreaCompanyCode_ValidFrom'
               AND object_id = OBJECT_ID(N'org.ControllingAreaCompanyCode'))
    CREATE NONCLUSTERED INDEX [IX_org_ControllingAreaCompanyCode_ValidFrom]
        ON [org].[ControllingAreaCompanyCode] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_ControllingAreaCompanyCode_ValidTo'
               AND object_id = OBJECT_ID(N'org.ControllingAreaCompanyCode'))
    CREATE NONCLUSTERED INDEX [IX_org_ControllingAreaCompanyCode_ValidTo]
        ON [org].[ControllingAreaCompanyCode] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_Department_ValidFrom'
               AND object_id = OBJECT_ID(N'org.Department'))
    CREATE NONCLUSTERED INDEX [IX_org_Department_ValidFrom]
        ON [org].[Department] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_Department_ValidTo'
               AND object_id = OBJECT_ID(N'org.Department'))
    CREATE NONCLUSTERED INDEX [IX_org_Department_ValidTo]
        ON [org].[Department] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_OrganizationalAssignment_ValidFrom'
               AND object_id = OBJECT_ID(N'org.OrganizationalAssignment'))
    CREATE NONCLUSTERED INDEX [IX_org_OrganizationalAssignment_ValidFrom]
        ON [org].[OrganizationalAssignment] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_OrganizationalAssignment_ValidTo'
               AND object_id = OBJECT_ID(N'org.OrganizationalAssignment'))
    CREATE NONCLUSTERED INDEX [IX_org_OrganizationalAssignment_ValidTo]
        ON [org].[OrganizationalAssignment] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_Plant_ValidFrom'
               AND object_id = OBJECT_ID(N'org.Plant'))
    CREATE NONCLUSTERED INDEX [IX_org_Plant_ValidFrom]
        ON [org].[Plant] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_org_Plant_ValidTo'
               AND object_id = OBJECT_ID(N'org.Plant'))
    CREATE NONCLUSTERED INDEX [IX_org_Plant_ValidTo]
        ON [org].[Plant] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_AccountDeterminationRule_ValidFrom'
               AND object_id = OBJECT_ID(N'cfg.AccountDeterminationRule'))
    CREATE NONCLUSTERED INDEX [IX_cfg_AccountDeterminationRule_ValidFrom]
        ON [cfg].[AccountDeterminationRule] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_AccountDeterminationRule_ValidTo'
               AND object_id = OBJECT_ID(N'cfg.AccountDeterminationRule'))
    CREATE NONCLUSTERED INDEX [IX_cfg_AccountDeterminationRule_ValidTo]
        ON [cfg].[AccountDeterminationRule] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_BrowserQueryLog_UserId'
               AND object_id = OBJECT_ID(N'cfg.BrowserQueryLog'))
    CREATE NONCLUSTERED INDEX [IX_cfg_BrowserQueryLog_UserId]
        ON [cfg].[BrowserQueryLog] ([TenantId], [UserId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_BrowserQueryLog_ExecutedAt'
               AND object_id = OBJECT_ID(N'cfg.BrowserQueryLog'))
    CREATE NONCLUSTERED INDEX [IX_cfg_BrowserQueryLog_ExecutedAt]
        ON [cfg].[BrowserQueryLog] ([TenantId], [ExecutedAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_BrowserQueryLog_ObjectName'
               AND object_id = OBJECT_ID(N'cfg.BrowserQueryLog'))
    CREATE NONCLUSTERED INDEX [IX_cfg_BrowserQueryLog_ObjectName]
        ON [cfg].[BrowserQueryLog] ([TenantId], [ObjectName]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_DictionaryChangeLog_DictionaryObjectId'
               AND object_id = OBJECT_ID(N'cfg.DictionaryChangeLog'))
    CREATE NONCLUSTERED INDEX [IX_cfg_DictionaryChangeLog_DictionaryObjectId]
        ON [cfg].[DictionaryChangeLog] ([TenantId], [DictionaryObjectId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_DictionaryChangeLog_ChangedAt'
               AND object_id = OBJECT_ID(N'cfg.DictionaryChangeLog'))
    CREATE NONCLUSTERED INDEX [IX_cfg_DictionaryChangeLog_ChangedAt]
        ON [cfg].[DictionaryChangeLog] ([TenantId], [ChangedAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_FiscalPeriod_PeriodStartDate'
               AND object_id = OBJECT_ID(N'cfg.FiscalPeriod'))
    CREATE NONCLUSTERED INDEX [IX_cfg_FiscalPeriod_PeriodStartDate]
        ON [cfg].[FiscalPeriod] ([TenantId], [PeriodStartDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_FiscalPeriod_PeriodEndDate'
               AND object_id = OBJECT_ID(N'cfg.FiscalPeriod'))
    CREATE NONCLUSTERED INDEX [IX_cfg_FiscalPeriod_PeriodEndDate]
        ON [cfg].[FiscalPeriod] ([TenantId], [PeriodEndDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_NumberRangeGap_NumberRangeIntervalId'
               AND object_id = OBJECT_ID(N'cfg.NumberRangeGap'))
    CREATE NONCLUSTERED INDEX [IX_cfg_NumberRangeGap_NumberRangeIntervalId]
        ON [cfg].[NumberRangeGap] ([TenantId], [NumberRangeIntervalId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_TaxCode_ValidFrom'
               AND object_id = OBJECT_ID(N'cfg.TaxCode'))
    CREATE NONCLUSTERED INDEX [IX_cfg_TaxCode_ValidFrom]
        ON [cfg].[TaxCode] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_TaxCode_ValidTo'
               AND object_id = OBJECT_ID(N'cfg.TaxCode'))
    CREATE NONCLUSTERED INDEX [IX_cfg_TaxCode_ValidTo]
        ON [cfg].[TaxCode] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_WithholdingTaxCode_ValidFrom'
               AND object_id = OBJECT_ID(N'cfg.WithholdingTaxCode'))
    CREATE NONCLUSTERED INDEX [IX_cfg_WithholdingTaxCode_ValidFrom]
        ON [cfg].[WithholdingTaxCode] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_cfg_WithholdingTaxCode_ValidTo'
               AND object_id = OBJECT_ID(N'cfg.WithholdingTaxCode'))
    CREATE NONCLUSTERED INDEX [IX_cfg_WithholdingTaxCode_ValidTo]
        ON [cfg].[WithholdingTaxCode] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_Address_City'
               AND object_id = OBJECT_ID(N'mdm.Address'))
    CREATE NONCLUSTERED INDEX [IX_mdm_Address_City]
        ON [mdm].[Address] ([TenantId], [City]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_Address_ValidFrom'
               AND object_id = OBJECT_ID(N'mdm.Address'))
    CREATE NONCLUSTERED INDEX [IX_mdm_Address_ValidFrom]
        ON [mdm].[Address] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_Address_ValidTo'
               AND object_id = OBJECT_ID(N'mdm.Address'))
    CREATE NONCLUSTERED INDEX [IX_mdm_Address_ValidTo]
        ON [mdm].[Address] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartner_Name1'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartner'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartner_Name1]
        ON [mdm].[BusinessPartner] ([TenantId], [Name1]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartner_FullName'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartner'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartner_FullName]
        ON [mdm].[BusinessPartner] ([TenantId], [FullName]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartner_SearchTerm1'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartner'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartner_SearchTerm1]
        ON [mdm].[BusinessPartner] ([TenantId], [SearchTerm1]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartner_SearchTerm2'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartner'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartner_SearchTerm2]
        ON [mdm].[BusinessPartner] ([TenantId], [SearchTerm2]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartner_ValidFrom'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartner'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartner_ValidFrom]
        ON [mdm].[BusinessPartner] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartner_ValidTo'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartner'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartner_ValidTo]
        ON [mdm].[BusinessPartner] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerAddress_ValidFrom'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerAddress'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerAddress_ValidFrom]
        ON [mdm].[BusinessPartnerAddress] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerAddress_ValidTo'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerAddress'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerAddress_ValidTo]
        ON [mdm].[BusinessPartnerAddress] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerAttachment_BusinessPartnerId'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerAttachment'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerAttachment_BusinessPartnerId]
        ON [mdm].[BusinessPartnerAttachment] ([TenantId], [BusinessPartnerId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerBank_ValidFrom'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerBank'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerBank_ValidFrom]
        ON [mdm].[BusinessPartnerBank] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerBank_ValidTo'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerBank'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerBank_ValidTo]
        ON [mdm].[BusinessPartnerBank] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerCommunication_ValidFrom'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerCommunication'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerCommunication_ValidFrom]
        ON [mdm].[BusinessPartnerCommunication] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerCommunication_ValidTo'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerCommunication'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerCommunication_ValidTo]
        ON [mdm].[BusinessPartnerCommunication] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerIdentification_ValidFrom'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerIdentification'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerIdentification_ValidFrom]
        ON [mdm].[BusinessPartnerIdentification] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerIdentification_ValidTo'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerIdentification'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerIdentification_ValidTo]
        ON [mdm].[BusinessPartnerIdentification] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerRelationship_SourceBusinessPartnerId'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerRelationship'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerRelationship_SourceBusinessPartnerId]
        ON [mdm].[BusinessPartnerRelationship] ([TenantId], [SourceBusinessPartnerId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerRelationship_TargetBusinessPartnerId'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerRelationship'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerRelationship_TargetBusinessPartnerId]
        ON [mdm].[BusinessPartnerRelationship] ([TenantId], [TargetBusinessPartnerId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerRelationship_ValidFrom'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerRelationship'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerRelationship_ValidFrom]
        ON [mdm].[BusinessPartnerRelationship] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerRelationship_ValidTo'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerRelationship'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerRelationship_ValidTo]
        ON [mdm].[BusinessPartnerRelationship] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerRoleAssignment_ValidFrom'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerRoleAssignment'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerRoleAssignment_ValidFrom]
        ON [mdm].[BusinessPartnerRoleAssignment] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerRoleAssignment_ValidTo'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerRoleAssignment'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerRoleAssignment_ValidTo]
        ON [mdm].[BusinessPartnerRoleAssignment] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerTaxNumber_ValidFrom'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerTaxNumber'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerTaxNumber_ValidFrom]
        ON [mdm].[BusinessPartnerTaxNumber] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_mdm_BusinessPartnerTaxNumber_ValidTo'
               AND object_id = OBJECT_ID(N'mdm.BusinessPartnerTaxNumber'))
    CREATE NONCLUSTERED INDEX [IX_mdm_BusinessPartnerTaxNumber_ValidTo]
        ON [mdm].[BusinessPartnerTaxNumber] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_Asset_Description'
               AND object_id = OBJECT_ID(N'fin.Asset'))
    CREATE NONCLUSTERED INDEX [IX_fin_Asset_Description]
        ON [fin].[Asset] ([TenantId], [Description]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_Asset_Status'
               AND object_id = OBJECT_ID(N'fin.Asset'))
    CREATE NONCLUSTERED INDEX [IX_fin_Asset_Status]
        ON [fin].[Asset] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_AssetTransaction_PostingDate'
               AND object_id = OBJECT_ID(N'fin.AssetTransaction'))
    CREATE NONCLUSTERED INDEX [IX_fin_AssetTransaction_PostingDate]
        ON [fin].[AssetTransaction] ([TenantId], [PostingDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_AssetTransaction_AssetValueDate'
               AND object_id = OBJECT_ID(N'fin.AssetTransaction'))
    CREATE NONCLUSTERED INDEX [IX_fin_AssetTransaction_AssetValueDate]
        ON [fin].[AssetTransaction] ([TenantId], [AssetValueDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_ClearingDocument_ClearingDate'
               AND object_id = OBJECT_ID(N'fin.ClearingDocument'))
    CREATE NONCLUSTERED INDEX [IX_fin_ClearingDocument_ClearingDate]
        ON [fin].[ClearingDocument] ([TenantId], [ClearingDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_CustomerInvoice_BusinessPartnerId'
               AND object_id = OBJECT_ID(N'fin.CustomerInvoice'))
    CREATE NONCLUSTERED INDEX [IX_fin_CustomerInvoice_BusinessPartnerId]
        ON [fin].[CustomerInvoice] ([TenantId], [BusinessPartnerId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_CustomerInvoice_PostingDate'
               AND object_id = OBJECT_ID(N'fin.CustomerInvoice'))
    CREATE NONCLUSTERED INDEX [IX_fin_CustomerInvoice_PostingDate]
        ON [fin].[CustomerInvoice] ([TenantId], [PostingDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_CustomerInvoice_Status'
               AND object_id = OBJECT_ID(N'fin.CustomerInvoice'))
    CREATE NONCLUSTERED INDEX [IX_fin_CustomerInvoice_Status]
        ON [fin].[CustomerInvoice] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_DepreciationPosting_DepreciationRunId'
               AND object_id = OBJECT_ID(N'fin.DepreciationPosting'))
    CREATE NONCLUSTERED INDEX [IX_fin_DepreciationPosting_DepreciationRunId]
        ON [fin].[DepreciationPosting] ([TenantId], [DepreciationRunId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryHeader_PostingDate'
               AND object_id = OBJECT_ID(N'fin.JournalEntryHeader'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryHeader_PostingDate]
        ON [fin].[JournalEntryHeader] ([TenantId], [PostingDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryHeader_FiscalPeriod'
               AND object_id = OBJECT_ID(N'fin.JournalEntryHeader'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryHeader_FiscalPeriod]
        ON [fin].[JournalEntryHeader] ([TenantId], [FiscalPeriod]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryHeader_ReferenceDocumentNumber'
               AND object_id = OBJECT_ID(N'fin.JournalEntryHeader'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryHeader_ReferenceDocumentNumber]
        ON [fin].[JournalEntryHeader] ([TenantId], [ReferenceDocumentNumber]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryHeader_Status'
               AND object_id = OBJECT_ID(N'fin.JournalEntryHeader'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryHeader_Status]
        ON [fin].[JournalEntryHeader] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryHeader_IntercompanyDocumentNumber'
               AND object_id = OBJECT_ID(N'fin.JournalEntryHeader'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryHeader_IntercompanyDocumentNumber]
        ON [fin].[JournalEntryHeader] ([TenantId], [IntercompanyDocumentNumber]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryHeader_CorrelationId'
               AND object_id = OBJECT_ID(N'fin.JournalEntryHeader'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryHeader_CorrelationId]
        ON [fin].[JournalEntryHeader] ([TenantId], [CorrelationId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_JournalEntryHeaderId'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_JournalEntryHeaderId]
        ON [fin].[JournalEntryLine] ([TenantId], [JournalEntryHeaderId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_PostingDate'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_PostingDate]
        ON [fin].[JournalEntryLine] ([TenantId], [PostingDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_FiscalPeriod'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_FiscalPeriod]
        ON [fin].[JournalEntryLine] ([TenantId], [FiscalPeriod]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_GLAccountId'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_GLAccountId]
        ON [fin].[JournalEntryLine] ([TenantId], [GLAccountId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_GLAccount'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_GLAccount]
        ON [fin].[JournalEntryLine] ([TenantId], [GLAccount]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_BusinessPartnerId'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_BusinessPartnerId]
        ON [fin].[JournalEntryLine] ([TenantId], [BusinessPartnerId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_CostCenterId'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_CostCenterId]
        ON [fin].[JournalEntryLine] ([TenantId], [CostCenterId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_ProfitCenterId'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_ProfitCenterId]
        ON [fin].[JournalEntryLine] ([TenantId], [ProfitCenterId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_InternalOrderId'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_InternalOrderId]
        ON [fin].[JournalEntryLine] ([TenantId], [InternalOrderId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_SegmentId'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_SegmentId]
        ON [fin].[JournalEntryLine] ([TenantId], [SegmentId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_AssignmentReference'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_AssignmentReference]
        ON [fin].[JournalEntryLine] ([TenantId], [AssignmentReference]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_DueDate'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_DueDate]
        ON [fin].[JournalEntryLine] ([TenantId], [DueDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_JournalEntryLine_ClearingDocumentNumber'
               AND object_id = OBJECT_ID(N'fin.JournalEntryLine'))
    CREATE NONCLUSTERED INDEX [IX_fin_JournalEntryLine_ClearingDocumentNumber]
        ON [fin].[JournalEntryLine] ([TenantId], [ClearingDocumentNumber]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_OpenItem_CompanyCodeId'
               AND object_id = OBJECT_ID(N'fin.OpenItem'))
    CREATE NONCLUSTERED INDEX [IX_fin_OpenItem_CompanyCodeId]
        ON [fin].[OpenItem] ([TenantId], [CompanyCodeId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_OpenItem_DocumentNumber'
               AND object_id = OBJECT_ID(N'fin.OpenItem'))
    CREATE NONCLUSTERED INDEX [IX_fin_OpenItem_DocumentNumber]
        ON [fin].[OpenItem] ([TenantId], [DocumentNumber]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_OpenItem_AccountType'
               AND object_id = OBJECT_ID(N'fin.OpenItem'))
    CREATE NONCLUSTERED INDEX [IX_fin_OpenItem_AccountType]
        ON [fin].[OpenItem] ([TenantId], [AccountType]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_OpenItem_BusinessPartnerId'
               AND object_id = OBJECT_ID(N'fin.OpenItem'))
    CREATE NONCLUSTERED INDEX [IX_fin_OpenItem_BusinessPartnerId]
        ON [fin].[OpenItem] ([TenantId], [BusinessPartnerId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_OpenItem_GLAccountId'
               AND object_id = OBJECT_ID(N'fin.OpenItem'))
    CREATE NONCLUSTERED INDEX [IX_fin_OpenItem_GLAccountId]
        ON [fin].[OpenItem] ([TenantId], [GLAccountId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_OpenItem_PostingDate'
               AND object_id = OBJECT_ID(N'fin.OpenItem'))
    CREATE NONCLUSTERED INDEX [IX_fin_OpenItem_PostingDate]
        ON [fin].[OpenItem] ([TenantId], [PostingDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_OpenItem_DueDate'
               AND object_id = OBJECT_ID(N'fin.OpenItem'))
    CREATE NONCLUSTERED INDEX [IX_fin_OpenItem_DueDate]
        ON [fin].[OpenItem] ([TenantId], [DueDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_OpenItem_AssignmentReference'
               AND object_id = OBJECT_ID(N'fin.OpenItem'))
    CREATE NONCLUSTERED INDEX [IX_fin_OpenItem_AssignmentReference]
        ON [fin].[OpenItem] ([TenantId], [AssignmentReference]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_OpenItem_Status'
               AND object_id = OBJECT_ID(N'fin.OpenItem'))
    CREATE NONCLUSTERED INDEX [IX_fin_OpenItem_Status]
        ON [fin].[OpenItem] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_PaymentHeader_BusinessPartnerId'
               AND object_id = OBJECT_ID(N'fin.PaymentHeader'))
    CREATE NONCLUSTERED INDEX [IX_fin_PaymentHeader_BusinessPartnerId]
        ON [fin].[PaymentHeader] ([TenantId], [BusinessPartnerId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_PaymentHeader_PaymentDate'
               AND object_id = OBJECT_ID(N'fin.PaymentHeader'))
    CREATE NONCLUSTERED INDEX [IX_fin_PaymentHeader_PaymentDate]
        ON [fin].[PaymentHeader] ([TenantId], [PaymentDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_PaymentHeader_Status'
               AND object_id = OBJECT_ID(N'fin.PaymentHeader'))
    CREATE NONCLUSTERED INDEX [IX_fin_PaymentHeader_Status]
        ON [fin].[PaymentHeader] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_RecurringEntry_NextRunDate'
               AND object_id = OBJECT_ID(N'fin.RecurringEntry'))
    CREATE NONCLUSTERED INDEX [IX_fin_RecurringEntry_NextRunDate]
        ON [fin].[RecurringEntry] ([TenantId], [NextRunDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_VendorInvoice_BusinessPartnerId'
               AND object_id = OBJECT_ID(N'fin.VendorInvoice'))
    CREATE NONCLUSTERED INDEX [IX_fin_VendorInvoice_BusinessPartnerId]
        ON [fin].[VendorInvoice] ([TenantId], [BusinessPartnerId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_VendorInvoice_VendorInvoiceNumber'
               AND object_id = OBJECT_ID(N'fin.VendorInvoice'))
    CREATE NONCLUSTERED INDEX [IX_fin_VendorInvoice_VendorInvoiceNumber]
        ON [fin].[VendorInvoice] ([TenantId], [VendorInvoiceNumber]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_VendorInvoice_PostingDate'
               AND object_id = OBJECT_ID(N'fin.VendorInvoice'))
    CREATE NONCLUSTERED INDEX [IX_fin_VendorInvoice_PostingDate]
        ON [fin].[VendorInvoice] ([TenantId], [PostingDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_fin_VendorInvoice_Status'
               AND object_id = OBJECT_ID(N'fin.VendorInvoice'))
    CREATE NONCLUSTERED INDEX [IX_fin_VendorInvoice_Status]
        ON [fin].[VendorInvoice] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ActivityType_ValidFrom'
               AND object_id = OBJECT_ID(N'co.ActivityType'))
    CREATE NONCLUSTERED INDEX [IX_co_ActivityType_ValidFrom]
        ON [co].[ActivityType] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ActivityType_ValidTo'
               AND object_id = OBJECT_ID(N'co.ActivityType'))
    CREATE NONCLUSTERED INDEX [IX_co_ActivityType_ValidTo]
        ON [co].[ActivityType] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ControllingPosting_FiscalYear'
               AND object_id = OBJECT_ID(N'co.ControllingPosting'))
    CREATE NONCLUSTERED INDEX [IX_co_ControllingPosting_FiscalYear]
        ON [co].[ControllingPosting] ([TenantId], [FiscalYear]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ControllingPosting_FiscalPeriod'
               AND object_id = OBJECT_ID(N'co.ControllingPosting'))
    CREATE NONCLUSTERED INDEX [IX_co_ControllingPosting_FiscalPeriod]
        ON [co].[ControllingPosting] ([TenantId], [FiscalPeriod]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ControllingPosting_PostingDate'
               AND object_id = OBJECT_ID(N'co.ControllingPosting'))
    CREATE NONCLUSTERED INDEX [IX_co_ControllingPosting_PostingDate]
        ON [co].[ControllingPosting] ([TenantId], [PostingDate]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ControllingPosting_ObjectType'
               AND object_id = OBJECT_ID(N'co.ControllingPosting'))
    CREATE NONCLUSTERED INDEX [IX_co_ControllingPosting_ObjectType]
        ON [co].[ControllingPosting] ([TenantId], [ObjectType]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ControllingPosting_ObjectId'
               AND object_id = OBJECT_ID(N'co.ControllingPosting'))
    CREATE NONCLUSTERED INDEX [IX_co_ControllingPosting_ObjectId]
        ON [co].[ControllingPosting] ([TenantId], [ObjectId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ControllingPosting_CostElementId'
               AND object_id = OBJECT_ID(N'co.ControllingPosting'))
    CREATE NONCLUSTERED INDEX [IX_co_ControllingPosting_CostElementId]
        ON [co].[ControllingPosting] ([TenantId], [CostElementId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_CostCenter_HierarchyNodeId'
               AND object_id = OBJECT_ID(N'co.CostCenter'))
    CREATE NONCLUSTERED INDEX [IX_co_CostCenter_HierarchyNodeId]
        ON [co].[CostCenter] ([TenantId], [HierarchyNodeId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_CostCenter_ProfitCenterId'
               AND object_id = OBJECT_ID(N'co.CostCenter'))
    CREATE NONCLUSTERED INDEX [IX_co_CostCenter_ProfitCenterId]
        ON [co].[CostCenter] ([TenantId], [ProfitCenterId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_CostCenter_ValidFrom'
               AND object_id = OBJECT_ID(N'co.CostCenter'))
    CREATE NONCLUSTERED INDEX [IX_co_CostCenter_ValidFrom]
        ON [co].[CostCenter] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_CostCenter_ValidTo'
               AND object_id = OBJECT_ID(N'co.CostCenter'))
    CREATE NONCLUSTERED INDEX [IX_co_CostCenter_ValidTo]
        ON [co].[CostCenter] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_CostElement_ValidFrom'
               AND object_id = OBJECT_ID(N'co.CostElement'))
    CREATE NONCLUSTERED INDEX [IX_co_CostElement_ValidFrom]
        ON [co].[CostElement] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_CostElement_ValidTo'
               AND object_id = OBJECT_ID(N'co.CostElement'))
    CREATE NONCLUSTERED INDEX [IX_co_CostElement_ValidTo]
        ON [co].[CostElement] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_HierarchyNode_HierarchyPath'
               AND object_id = OBJECT_ID(N'co.HierarchyNode'))
    CREATE NONCLUSTERED INDEX [IX_co_HierarchyNode_HierarchyPath]
        ON [co].[HierarchyNode] ([TenantId], [HierarchyPath]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_HierarchyNode_ValidFrom'
               AND object_id = OBJECT_ID(N'co.HierarchyNode'))
    CREATE NONCLUSTERED INDEX [IX_co_HierarchyNode_ValidFrom]
        ON [co].[HierarchyNode] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_HierarchyNode_ValidTo'
               AND object_id = OBJECT_ID(N'co.HierarchyNode'))
    CREATE NONCLUSTERED INDEX [IX_co_HierarchyNode_ValidTo]
        ON [co].[HierarchyNode] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_InternalOrder_Description'
               AND object_id = OBJECT_ID(N'co.InternalOrder'))
    CREATE NONCLUSTERED INDEX [IX_co_InternalOrder_Description]
        ON [co].[InternalOrder] ([TenantId], [Description]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_InternalOrder_SystemStatus'
               AND object_id = OBJECT_ID(N'co.InternalOrder'))
    CREATE NONCLUSTERED INDEX [IX_co_InternalOrder_SystemStatus]
        ON [co].[InternalOrder] ([TenantId], [SystemStatus]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ProfitCenter_HierarchyNodeId'
               AND object_id = OBJECT_ID(N'co.ProfitCenter'))
    CREATE NONCLUSTERED INDEX [IX_co_ProfitCenter_HierarchyNodeId]
        ON [co].[ProfitCenter] ([TenantId], [HierarchyNodeId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ProfitCenter_SegmentId'
               AND object_id = OBJECT_ID(N'co.ProfitCenter'))
    CREATE NONCLUSTERED INDEX [IX_co_ProfitCenter_SegmentId]
        ON [co].[ProfitCenter] ([TenantId], [SegmentId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ProfitCenter_ValidFrom'
               AND object_id = OBJECT_ID(N'co.ProfitCenter'))
    CREATE NONCLUSTERED INDEX [IX_co_ProfitCenter_ValidFrom]
        ON [co].[ProfitCenter] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ProfitCenter_ValidTo'
               AND object_id = OBJECT_ID(N'co.ProfitCenter'))
    CREATE NONCLUSTERED INDEX [IX_co_ProfitCenter_ValidTo]
        ON [co].[ProfitCenter] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ProfitCenterAssignment_ValidFrom'
               AND object_id = OBJECT_ID(N'co.ProfitCenterAssignment'))
    CREATE NONCLUSTERED INDEX [IX_co_ProfitCenterAssignment_ValidFrom]
        ON [co].[ProfitCenterAssignment] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_ProfitCenterAssignment_ValidTo'
               AND object_id = OBJECT_ID(N'co.ProfitCenterAssignment'))
    CREATE NONCLUSTERED INDEX [IX_co_ProfitCenterAssignment_ValidTo]
        ON [co].[ProfitCenterAssignment] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_co_SettlementDocument_SenderObjectId'
               AND object_id = OBJECT_ID(N'co.SettlementDocument'))
    CREATE NONCLUSTERED INDEX [IX_co_SettlementDocument_SenderObjectId]
        ON [co].[SettlementDocument] ([TenantId], [SenderObjectId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_wf_Notification_RecipientUserId'
               AND object_id = OBJECT_ID(N'wf.Notification'))
    CREATE NONCLUSTERED INDEX [IX_wf_Notification_RecipientUserId]
        ON [wf].[Notification] ([TenantId], [RecipientUserId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_wf_Notification_Status'
               AND object_id = OBJECT_ID(N'wf.Notification'))
    CREATE NONCLUSTERED INDEX [IX_wf_Notification_Status]
        ON [wf].[Notification] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_wf_WorkflowHistory_WorkflowInstanceId'
               AND object_id = OBJECT_ID(N'wf.WorkflowHistory'))
    CREATE NONCLUSTERED INDEX [IX_wf_WorkflowHistory_WorkflowInstanceId]
        ON [wf].[WorkflowHistory] ([TenantId], [WorkflowInstanceId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_wf_WorkflowInstance_ObjectType'
               AND object_id = OBJECT_ID(N'wf.WorkflowInstance'))
    CREATE NONCLUSTERED INDEX [IX_wf_WorkflowInstance_ObjectType]
        ON [wf].[WorkflowInstance] ([TenantId], [ObjectType]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_wf_WorkflowInstance_ObjectId'
               AND object_id = OBJECT_ID(N'wf.WorkflowInstance'))
    CREATE NONCLUSTERED INDEX [IX_wf_WorkflowInstance_ObjectId]
        ON [wf].[WorkflowInstance] ([TenantId], [ObjectId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_wf_WorkflowInstance_Status'
               AND object_id = OBJECT_ID(N'wf.WorkflowInstance'))
    CREATE NONCLUSTERED INDEX [IX_wf_WorkflowInstance_Status]
        ON [wf].[WorkflowInstance] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_wf_WorkflowTask_Status'
               AND object_id = OBJECT_ID(N'wf.WorkflowTask'))
    CREATE NONCLUSTERED INDEX [IX_wf_WorkflowTask_Status]
        ON [wf].[WorkflowTask] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_LoginHistory_UserName'
               AND object_id = OBJECT_ID(N'sec.LoginHistory'))
    CREATE NONCLUSTERED INDEX [IX_sec_LoginHistory_UserName]
        ON [sec].[LoginHistory] ([TenantId], [UserName]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_LoginHistory_AttemptedAt'
               AND object_id = OBJECT_ID(N'sec.LoginHistory'))
    CREATE NONCLUSTERED INDEX [IX_sec_LoginHistory_AttemptedAt]
        ON [sec].[LoginHistory] ([TenantId], [AttemptedAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_Role_ValidFrom'
               AND object_id = OBJECT_ID(N'sec.Role'))
    CREATE NONCLUSTERED INDEX [IX_sec_Role_ValidFrom]
        ON [sec].[Role] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_Role_ValidTo'
               AND object_id = OBJECT_ID(N'sec.Role'))
    CREATE NONCLUSTERED INDEX [IX_sec_Role_ValidTo]
        ON [sec].[Role] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_User_Email'
               AND object_id = OBJECT_ID(N'sec.User'))
    CREATE NONCLUSTERED INDEX [IX_sec_User_Email]
        ON [sec].[User] ([TenantId], [Email]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_User_Status'
               AND object_id = OBJECT_ID(N'sec.User'))
    CREATE NONCLUSTERED INDEX [IX_sec_User_Status]
        ON [sec].[User] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserAuthorization_ValidFrom'
               AND object_id = OBJECT_ID(N'sec.UserAuthorization'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserAuthorization_ValidFrom]
        ON [sec].[UserAuthorization] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserAuthorization_ValidTo'
               AND object_id = OBJECT_ID(N'sec.UserAuthorization'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserAuthorization_ValidTo]
        ON [sec].[UserAuthorization] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserCompanyCode_ValidFrom'
               AND object_id = OBJECT_ID(N'sec.UserCompanyCode'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserCompanyCode_ValidFrom]
        ON [sec].[UserCompanyCode] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserCompanyCode_ValidTo'
               AND object_id = OBJECT_ID(N'sec.UserCompanyCode'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserCompanyCode_ValidTo]
        ON [sec].[UserCompanyCode] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserRole_ValidFrom'
               AND object_id = OBJECT_ID(N'sec.UserRole'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserRole_ValidFrom]
        ON [sec].[UserRole] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserRole_ValidTo'
               AND object_id = OBJECT_ID(N'sec.UserRole'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserRole_ValidTo]
        ON [sec].[UserRole] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserSession_UserId'
               AND object_id = OBJECT_ID(N'sec.UserSession'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserSession_UserId]
        ON [sec].[UserSession] ([TenantId], [UserId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserSession_StartedAt'
               AND object_id = OBJECT_ID(N'sec.UserSession'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserSession_StartedAt]
        ON [sec].[UserSession] ([TenantId], [StartedAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserSubstitution_ValidFrom'
               AND object_id = OBJECT_ID(N'sec.UserSubstitution'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserSubstitution_ValidFrom]
        ON [sec].[UserSubstitution] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_sec_UserSubstitution_ValidTo'
               AND object_id = OBJECT_ID(N'sec.UserSubstitution'))
    CREATE NONCLUSTERED INDEX [IX_sec_UserSubstitution_ValidTo]
        ON [sec].[UserSubstitution] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_AuditLog_OccurredAt'
               AND object_id = OBJECT_ID(N'audit.AuditLog'))
    CREATE NONCLUSTERED INDEX [IX_audit_AuditLog_OccurredAt]
        ON [audit].[AuditLog] ([TenantId], [OccurredAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_AuditLog_UserName'
               AND object_id = OBJECT_ID(N'audit.AuditLog'))
    CREATE NONCLUSTERED INDEX [IX_audit_AuditLog_UserName]
        ON [audit].[AuditLog] ([TenantId], [UserName]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_AuditLog_CompanyCodeId'
               AND object_id = OBJECT_ID(N'audit.AuditLog'))
    CREATE NONCLUSTERED INDEX [IX_audit_AuditLog_CompanyCodeId]
        ON [audit].[AuditLog] ([TenantId], [CompanyCodeId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_AuditLog_Action'
               AND object_id = OBJECT_ID(N'audit.AuditLog'))
    CREATE NONCLUSTERED INDEX [IX_audit_AuditLog_Action]
        ON [audit].[AuditLog] ([TenantId], [Action]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_AuditLog_ObjectType'
               AND object_id = OBJECT_ID(N'audit.AuditLog'))
    CREATE NONCLUSTERED INDEX [IX_audit_AuditLog_ObjectType]
        ON [audit].[AuditLog] ([TenantId], [ObjectType]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_AuditLog_CorrelationId'
               AND object_id = OBJECT_ID(N'audit.AuditLog'))
    CREATE NONCLUSTERED INDEX [IX_audit_AuditLog_CorrelationId]
        ON [audit].[AuditLog] ([TenantId], [CorrelationId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_ChangeDocumentHeader_ObjectClass'
               AND object_id = OBJECT_ID(N'audit.ChangeDocumentHeader'))
    CREATE NONCLUSTERED INDEX [IX_audit_ChangeDocumentHeader_ObjectClass]
        ON [audit].[ChangeDocumentHeader] ([TenantId], [ObjectClass]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_ChangeDocumentHeader_ObjectId'
               AND object_id = OBJECT_ID(N'audit.ChangeDocumentHeader'))
    CREATE NONCLUSTERED INDEX [IX_audit_ChangeDocumentHeader_ObjectId]
        ON [audit].[ChangeDocumentHeader] ([TenantId], [ObjectId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_ChangeDocumentHeader_ChangedAt'
               AND object_id = OBJECT_ID(N'audit.ChangeDocumentHeader'))
    CREATE NONCLUSTERED INDEX [IX_audit_ChangeDocumentHeader_ChangedAt]
        ON [audit].[ChangeDocumentHeader] ([TenantId], [ChangedAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_ChangeDocumentHeader_ChangedBy'
               AND object_id = OBJECT_ID(N'audit.ChangeDocumentHeader'))
    CREATE NONCLUSTERED INDEX [IX_audit_ChangeDocumentHeader_ChangedBy]
        ON [audit].[ChangeDocumentHeader] ([TenantId], [ChangedBy]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_DataAccessLog_AccessedAt'
               AND object_id = OBJECT_ID(N'audit.DataAccessLog'))
    CREATE NONCLUSTERED INDEX [IX_audit_DataAccessLog_AccessedAt]
        ON [audit].[DataAccessLog] ([TenantId], [AccessedAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_audit_DataAccessLog_UserId'
               AND object_id = OBJECT_ID(N'audit.DataAccessLog'))
    CREATE NONCLUSTERED INDEX [IX_audit_DataAccessLog_UserId]
        ON [audit].[DataAccessLog] ([TenantId], [UserId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_rpt_ReportExecutionLog_ReportDefinitionId'
               AND object_id = OBJECT_ID(N'rpt.ReportExecutionLog'))
    CREATE NONCLUSTERED INDEX [IX_rpt_ReportExecutionLog_ReportDefinitionId]
        ON [rpt].[ReportExecutionLog] ([TenantId], [ReportDefinitionId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_rpt_ReportExecutionLog_UserId'
               AND object_id = OBJECT_ID(N'rpt.ReportExecutionLog'))
    CREATE NONCLUSTERED INDEX [IX_rpt_ReportExecutionLog_UserId]
        ON [rpt].[ReportExecutionLog] ([TenantId], [UserId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_rpt_ReportExecutionLog_ExecutedAt'
               AND object_id = OBJECT_ID(N'rpt.ReportExecutionLog'))
    CREATE NONCLUSTERED INDEX [IX_rpt_ReportExecutionLog_ExecutedAt]
        ON [rpt].[ReportExecutionLog] ([TenantId], [ExecutedAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_ApiClient_ValidFrom'
               AND object_id = OBJECT_ID(N'intg.ApiClient'))
    CREATE NONCLUSTERED INDEX [IX_intg_ApiClient_ValidFrom]
        ON [intg].[ApiClient] ([TenantId], [ValidFrom]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_ApiClient_ValidTo'
               AND object_id = OBJECT_ID(N'intg.ApiClient'))
    CREATE NONCLUSTERED INDEX [IX_intg_ApiClient_ValidTo]
        ON [intg].[ApiClient] ([TenantId], [ValidTo]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_BankStatementItem_Status'
               AND object_id = OBJECT_ID(N'intg.BankStatementItem'))
    CREATE NONCLUSTERED INDEX [IX_intg_BankStatementItem_Status]
        ON [intg].[BankStatementItem] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_IdempotencyKey_ExpiresAt'
               AND object_id = OBJECT_ID(N'intg.IdempotencyKey'))
    CREATE NONCLUSTERED INDEX [IX_intg_IdempotencyKey_ExpiresAt]
        ON [intg].[IdempotencyKey] ([TenantId], [ExpiresAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_ImportJob_Status'
               AND object_id = OBJECT_ID(N'intg.ImportJob'))
    CREATE NONCLUSTERED INDEX [IX_intg_ImportJob_Status]
        ON [intg].[ImportJob] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_InboundMessage_SourceSystem'
               AND object_id = OBJECT_ID(N'intg.InboundMessage'))
    CREATE NONCLUSTERED INDEX [IX_intg_InboundMessage_SourceSystem]
        ON [intg].[InboundMessage] ([TenantId], [SourceSystem]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_InboundMessage_ReceivedAt'
               AND object_id = OBJECT_ID(N'intg.InboundMessage'))
    CREATE NONCLUSTERED INDEX [IX_intg_InboundMessage_ReceivedAt]
        ON [intg].[InboundMessage] ([TenantId], [ReceivedAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_InboundMessage_Status'
               AND object_id = OBJECT_ID(N'intg.InboundMessage'))
    CREATE NONCLUSTERED INDEX [IX_intg_InboundMessage_Status]
        ON [intg].[InboundMessage] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_OutboxMessage_EventType'
               AND object_id = OBJECT_ID(N'intg.OutboxMessage'))
    CREATE NONCLUSTERED INDEX [IX_intg_OutboxMessage_EventType]
        ON [intg].[OutboxMessage] ([TenantId], [EventType]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_OutboxMessage_Status'
               AND object_id = OBJECT_ID(N'intg.OutboxMessage'))
    CREATE NONCLUSTERED INDEX [IX_intg_OutboxMessage_Status]
        ON [intg].[OutboxMessage] ([TenantId], [Status]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_OutboxMessage_NextAttemptAt'
               AND object_id = OBJECT_ID(N'intg.OutboxMessage'))
    CREATE NONCLUSTERED INDEX [IX_intg_OutboxMessage_NextAttemptAt]
        ON [intg].[OutboxMessage] ([TenantId], [NextAttemptAt]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_WebhookDelivery_WebhookSubscriptionId'
               AND object_id = OBJECT_ID(N'intg.WebhookDelivery'))
    CREATE NONCLUSTERED INDEX [IX_intg_WebhookDelivery_WebhookSubscriptionId]
        ON [intg].[WebhookDelivery] ([TenantId], [WebhookSubscriptionId]);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_intg_WebhookDelivery_AttemptedAt'
               AND object_id = OBJECT_ID(N'intg.WebhookDelivery'))
    CREATE NONCLUSTERED INDEX [IX_intg_WebhookDelivery_AttemptedAt]
        ON [intg].[WebhookDelivery] ([TenantId], [AttemptedAt]);
GO
