/* ============================================================================
   S/4HANA-inspired ERP - foreign keys
   837 constraints, added after all tables exist

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

USE [ErpS4];
GO

-- org.Branch.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Branch_TenantId')
    ALTER TABLE [org].[Branch] WITH CHECK
        ADD CONSTRAINT [FK_org_Branch_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.Branch.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Branch_CompanyCodeId')
    ALTER TABLE [org].[Branch] WITH CHECK
        ADD CONSTRAINT [FK_org_Branch_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- org.Branch.LocationId -> org.Location (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Branch_LocationId')
    ALTER TABLE [org].[Branch] WITH CHECK
        ADD CONSTRAINT [FK_org_Branch_LocationId] FOREIGN KEY ([LocationId])
        REFERENCES [org].[Location] ([Id]);
GO

-- org.Branch.BusinessAreaId -> org.BusinessArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Branch_BusinessAreaId')
    ALTER TABLE [org].[Branch] WITH CHECK
        ADD CONSTRAINT [FK_org_Branch_BusinessAreaId] FOREIGN KEY ([BusinessAreaId])
        REFERENCES [org].[BusinessArea] ([Id]);
GO

-- org.Branch.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Branch_ProfitCenterId')
    ALTER TABLE [org].[Branch] WITH CHECK
        ADD CONSTRAINT [FK_org_Branch_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- org.BusinessArea.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_BusinessArea_TenantId')
    ALTER TABLE [org].[BusinessArea] WITH CHECK
        ADD CONSTRAINT [FK_org_BusinessArea_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.Company.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Company_TenantId')
    ALTER TABLE [org].[Company] WITH CHECK
        ADD CONSTRAINT [FK_org_Company_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.Company.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Company_CountryCode')
    ALTER TABLE [org].[Company] WITH CHECK
        ADD CONSTRAINT [FK_org_Company_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- org.Company.GroupCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Company_GroupCurrencyCode')
    ALTER TABLE [org].[Company] WITH CHECK
        ADD CONSTRAINT [FK_org_Company_GroupCurrencyCode] FOREIGN KEY ([TenantId], [GroupCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- org.Company.LanguageCode -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Company_LanguageCode')
    ALTER TABLE [org].[Company] WITH CHECK
        ADD CONSTRAINT [FK_org_Company_LanguageCode] FOREIGN KEY ([LanguageCode])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- org.CompanyCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_TenantId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.CompanyCode.CompanyId -> org.Company (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_CompanyId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_CompanyId] FOREIGN KEY ([CompanyId])
        REFERENCES [org].[Company] ([Id]);
GO

-- org.CompanyCode.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_CountryCode')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- org.CompanyCode.LocalCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_LocalCurrencyCode')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_LocalCurrencyCode] FOREIGN KEY ([TenantId], [LocalCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- org.CompanyCode.GroupCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_GroupCurrencyCode')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_GroupCurrencyCode] FOREIGN KEY ([TenantId], [GroupCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- org.CompanyCode.HardCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_HardCurrencyCode')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_HardCurrencyCode] FOREIGN KEY ([TenantId], [HardCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- org.CompanyCode.IndexCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_IndexCurrencyCode')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_IndexCurrencyCode] FOREIGN KEY ([TenantId], [IndexCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- org.CompanyCode.LanguageCode -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_LanguageCode')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_LanguageCode] FOREIGN KEY ([LanguageCode])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- org.CompanyCode.ChartOfAccountsId -> cfg.ChartOfAccounts (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_ChartOfAccountsId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_ChartOfAccountsId] FOREIGN KEY ([ChartOfAccountsId])
        REFERENCES [cfg].[ChartOfAccounts] ([Id]);
GO

-- org.CompanyCode.CountryChartOfAccountsId -> cfg.ChartOfAccounts (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_CountryChartOfAccountsId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_CountryChartOfAccountsId] FOREIGN KEY ([CountryChartOfAccountsId])
        REFERENCES [cfg].[ChartOfAccounts] ([Id]);
GO

-- org.CompanyCode.FiscalYearVariantId -> cfg.FiscalYearVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_FiscalYearVariantId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_FiscalYearVariantId] FOREIGN KEY ([FiscalYearVariantId])
        REFERENCES [cfg].[FiscalYearVariant] ([Id]);
GO

-- org.CompanyCode.PostingPeriodVariantId -> cfg.PostingPeriodVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_PostingPeriodVariantId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_PostingPeriodVariantId] FOREIGN KEY ([PostingPeriodVariantId])
        REFERENCES [cfg].[PostingPeriodVariant] ([Id]);
GO

-- org.CompanyCode.FieldStatusVariantId -> cfg.FieldStatusVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_FieldStatusVariantId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_FieldStatusVariantId] FOREIGN KEY ([FieldStatusVariantId])
        REFERENCES [cfg].[FieldStatusVariant] ([Id]);
GO

-- org.CompanyCode.CreditControlAreaId -> org.CreditControlArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_CreditControlAreaId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_CreditControlAreaId] FOREIGN KEY ([CreditControlAreaId])
        REFERENCES [org].[CreditControlArea] ([Id]);
GO

-- org.CompanyCode.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_ControllingAreaId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- org.CompanyCode.AddressId -> mdm.Address (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CompanyCode_AddressId')
    ALTER TABLE [org].[CompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_CompanyCode_AddressId] FOREIGN KEY ([AddressId])
        REFERENCES [mdm].[Address] ([Id]);
GO

-- org.ControllingArea.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_ControllingArea_TenantId')
    ALTER TABLE [org].[ControllingArea] WITH CHECK
        ADD CONSTRAINT [FK_org_ControllingArea_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.ControllingArea.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_ControllingArea_CurrencyCode')
    ALTER TABLE [org].[ControllingArea] WITH CHECK
        ADD CONSTRAINT [FK_org_ControllingArea_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- org.ControllingArea.ChartOfAccountsId -> cfg.ChartOfAccounts (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_ControllingArea_ChartOfAccountsId')
    ALTER TABLE [org].[ControllingArea] WITH CHECK
        ADD CONSTRAINT [FK_org_ControllingArea_ChartOfAccountsId] FOREIGN KEY ([ChartOfAccountsId])
        REFERENCES [cfg].[ChartOfAccounts] ([Id]);
GO

-- org.ControllingArea.FiscalYearVariantId -> cfg.FiscalYearVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_ControllingArea_FiscalYearVariantId')
    ALTER TABLE [org].[ControllingArea] WITH CHECK
        ADD CONSTRAINT [FK_org_ControllingArea_FiscalYearVariantId] FOREIGN KEY ([FiscalYearVariantId])
        REFERENCES [cfg].[FiscalYearVariant] ([Id]);
GO

-- org.ControllingArea.OperatingConcernId -> org.OperatingConcern (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_ControllingArea_OperatingConcernId')
    ALTER TABLE [org].[ControllingArea] WITH CHECK
        ADD CONSTRAINT [FK_org_ControllingArea_OperatingConcernId] FOREIGN KEY ([OperatingConcernId])
        REFERENCES [org].[OperatingConcern] ([Id]);
GO

-- org.ControllingAreaCompanyCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_ControllingAreaCompanyCode_TenantId')
    ALTER TABLE [org].[ControllingAreaCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_ControllingAreaCompanyCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.ControllingAreaCompanyCode.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_ControllingAreaCompanyCode_ControllingAreaId')
    ALTER TABLE [org].[ControllingAreaCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_ControllingAreaCompanyCode_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- org.ControllingAreaCompanyCode.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_ControllingAreaCompanyCode_CompanyCodeId')
    ALTER TABLE [org].[ControllingAreaCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_org_ControllingAreaCompanyCode_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- org.CreditControlArea.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CreditControlArea_TenantId')
    ALTER TABLE [org].[CreditControlArea] WITH CHECK
        ADD CONSTRAINT [FK_org_CreditControlArea_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.CreditControlArea.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_CreditControlArea_CurrencyCode')
    ALTER TABLE [org].[CreditControlArea] WITH CHECK
        ADD CONSTRAINT [FK_org_CreditControlArea_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- org.Department.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Department_TenantId')
    ALTER TABLE [org].[Department] WITH CHECK
        ADD CONSTRAINT [FK_org_Department_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.Department.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Department_CompanyCodeId')
    ALTER TABLE [org].[Department] WITH CHECK
        ADD CONSTRAINT [FK_org_Department_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- org.Department.ParentDepartmentId -> org.Department (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Department_ParentDepartmentId')
    ALTER TABLE [org].[Department] WITH CHECK
        ADD CONSTRAINT [FK_org_Department_ParentDepartmentId] FOREIGN KEY ([ParentDepartmentId])
        REFERENCES [org].[Department] ([Id]);
GO

-- org.Department.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Department_CostCenterId')
    ALTER TABLE [org].[Department] WITH CHECK
        ADD CONSTRAINT [FK_org_Department_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- org.Department.ManagerBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Department_ManagerBusinessPartnerId')
    ALTER TABLE [org].[Department] WITH CHECK
        ADD CONSTRAINT [FK_org_Department_ManagerBusinessPartnerId] FOREIGN KEY ([ManagerBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- org.DistributionChannel.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_DistributionChannel_TenantId')
    ALTER TABLE [org].[DistributionChannel] WITH CHECK
        ADD CONSTRAINT [FK_org_DistributionChannel_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.Division.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Division_TenantId')
    ALTER TABLE [org].[Division] WITH CHECK
        ADD CONSTRAINT [FK_org_Division_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.FactoryCalendar.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_FactoryCalendar_TenantId')
    ALTER TABLE [org].[FactoryCalendar] WITH CHECK
        ADD CONSTRAINT [FK_org_FactoryCalendar_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.FactoryCalendar.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_FactoryCalendar_CountryCode')
    ALTER TABLE [org].[FactoryCalendar] WITH CHECK
        ADD CONSTRAINT [FK_org_FactoryCalendar_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- org.FactoryCalendarHoliday.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_FactoryCalendarHoliday_TenantId')
    ALTER TABLE [org].[FactoryCalendarHoliday] WITH CHECK
        ADD CONSTRAINT [FK_org_FactoryCalendarHoliday_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.FactoryCalendarHoliday.FactoryCalendarId -> org.FactoryCalendar (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_FactoryCalendarHoliday_FactoryCalendarId')
    ALTER TABLE [org].[FactoryCalendarHoliday] WITH CHECK
        ADD CONSTRAINT [FK_org_FactoryCalendarHoliday_FactoryCalendarId] FOREIGN KEY ([FactoryCalendarId])
        REFERENCES [org].[FactoryCalendar] ([Id]);
GO

-- org.FunctionalArea.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_FunctionalArea_TenantId')
    ALTER TABLE [org].[FunctionalArea] WITH CHECK
        ADD CONSTRAINT [FK_org_FunctionalArea_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.Location.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Location_TenantId')
    ALTER TABLE [org].[Location] WITH CHECK
        ADD CONSTRAINT [FK_org_Location_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.Location.AddressId -> mdm.Address (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Location_AddressId')
    ALTER TABLE [org].[Location] WITH CHECK
        ADD CONSTRAINT [FK_org_Location_AddressId] FOREIGN KEY ([AddressId])
        REFERENCES [mdm].[Address] ([Id]);
GO

-- org.Location.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Location_CountryCode')
    ALTER TABLE [org].[Location] WITH CHECK
        ADD CONSTRAINT [FK_org_Location_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- org.Location.RegionCode -> cfg.Region (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Location_RegionCode')
    ALTER TABLE [org].[Location] WITH CHECK
        ADD CONSTRAINT [FK_org_Location_RegionCode] FOREIGN KEY ([TenantId], [CountryCode], [RegionCode])
        REFERENCES [cfg].[Region] ([TenantId], [CountryCode], [RegionCode]);
GO

-- org.OperatingConcern.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_OperatingConcern_TenantId')
    ALTER TABLE [org].[OperatingConcern] WITH CHECK
        ADD CONSTRAINT [FK_org_OperatingConcern_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.OperatingConcern.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_OperatingConcern_CurrencyCode')
    ALTER TABLE [org].[OperatingConcern] WITH CHECK
        ADD CONSTRAINT [FK_org_OperatingConcern_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- org.OperatingConcern.FiscalYearVariantId -> cfg.FiscalYearVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_OperatingConcern_FiscalYearVariantId')
    ALTER TABLE [org].[OperatingConcern] WITH CHECK
        ADD CONSTRAINT [FK_org_OperatingConcern_FiscalYearVariantId] FOREIGN KEY ([FiscalYearVariantId])
        REFERENCES [cfg].[FiscalYearVariant] ([Id]);
GO

-- org.OrganizationalAssignment.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_OrganizationalAssignment_TenantId')
    ALTER TABLE [org].[OrganizationalAssignment] WITH CHECK
        ADD CONSTRAINT [FK_org_OrganizationalAssignment_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.Plant.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Plant_TenantId')
    ALTER TABLE [org].[Plant] WITH CHECK
        ADD CONSTRAINT [FK_org_Plant_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.Plant.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Plant_CompanyCodeId')
    ALTER TABLE [org].[Plant] WITH CHECK
        ADD CONSTRAINT [FK_org_Plant_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- org.Plant.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Plant_CountryCode')
    ALTER TABLE [org].[Plant] WITH CHECK
        ADD CONSTRAINT [FK_org_Plant_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- org.Plant.AddressId -> mdm.Address (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Plant_AddressId')
    ALTER TABLE [org].[Plant] WITH CHECK
        ADD CONSTRAINT [FK_org_Plant_AddressId] FOREIGN KEY ([AddressId])
        REFERENCES [mdm].[Address] ([Id]);
GO

-- org.Plant.PurchasingOrganizationId -> org.PurchasingOrganization (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Plant_PurchasingOrganizationId')
    ALTER TABLE [org].[Plant] WITH CHECK
        ADD CONSTRAINT [FK_org_Plant_PurchasingOrganizationId] FOREIGN KEY ([PurchasingOrganizationId])
        REFERENCES [org].[PurchasingOrganization] ([Id]);
GO

-- org.Plant.FactoryCalendarId -> org.FactoryCalendar (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Plant_FactoryCalendarId')
    ALTER TABLE [org].[Plant] WITH CHECK
        ADD CONSTRAINT [FK_org_Plant_FactoryCalendarId] FOREIGN KEY ([FactoryCalendarId])
        REFERENCES [org].[FactoryCalendar] ([Id]);
GO

-- org.PurchasingOrganization.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_PurchasingOrganization_TenantId')
    ALTER TABLE [org].[PurchasingOrganization] WITH CHECK
        ADD CONSTRAINT [FK_org_PurchasingOrganization_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.PurchasingOrganization.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_PurchasingOrganization_CompanyCodeId')
    ALTER TABLE [org].[PurchasingOrganization] WITH CHECK
        ADD CONSTRAINT [FK_org_PurchasingOrganization_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- org.SalesArea.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_SalesArea_TenantId')
    ALTER TABLE [org].[SalesArea] WITH CHECK
        ADD CONSTRAINT [FK_org_SalesArea_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.SalesArea.SalesOrganizationId -> org.SalesOrganization (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_SalesArea_SalesOrganizationId')
    ALTER TABLE [org].[SalesArea] WITH CHECK
        ADD CONSTRAINT [FK_org_SalesArea_SalesOrganizationId] FOREIGN KEY ([SalesOrganizationId])
        REFERENCES [org].[SalesOrganization] ([Id]);
GO

-- org.SalesArea.DistributionChannelId -> org.DistributionChannel (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_SalesArea_DistributionChannelId')
    ALTER TABLE [org].[SalesArea] WITH CHECK
        ADD CONSTRAINT [FK_org_SalesArea_DistributionChannelId] FOREIGN KEY ([DistributionChannelId])
        REFERENCES [org].[DistributionChannel] ([Id]);
GO

-- org.SalesArea.DivisionId -> org.Division (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_SalesArea_DivisionId')
    ALTER TABLE [org].[SalesArea] WITH CHECK
        ADD CONSTRAINT [FK_org_SalesArea_DivisionId] FOREIGN KEY ([DivisionId])
        REFERENCES [org].[Division] ([Id]);
GO

-- org.SalesOrganization.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_SalesOrganization_TenantId')
    ALTER TABLE [org].[SalesOrganization] WITH CHECK
        ADD CONSTRAINT [FK_org_SalesOrganization_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- org.SalesOrganization.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_SalesOrganization_CompanyCodeId')
    ALTER TABLE [org].[SalesOrganization] WITH CHECK
        ADD CONSTRAINT [FK_org_SalesOrganization_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- org.SalesOrganization.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_SalesOrganization_CurrencyCode')
    ALTER TABLE [org].[SalesOrganization] WITH CHECK
        ADD CONSTRAINT [FK_org_SalesOrganization_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- org.SalesOrganization.AddressId -> mdm.Address (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_SalesOrganization_AddressId')
    ALTER TABLE [org].[SalesOrganization] WITH CHECK
        ADD CONSTRAINT [FK_org_SalesOrganization_AddressId] FOREIGN KEY ([AddressId])
        REFERENCES [mdm].[Address] ([Id]);
GO

-- org.Segment.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_org_Segment_TenantId')
    ALTER TABLE [org].[Segment] WITH CHECK
        ADD CONSTRAINT [FK_org_Segment_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.AccountDeterminationRule.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountDeterminationRule_TenantId')
    ALTER TABLE [cfg].[AccountDeterminationRule] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountDeterminationRule_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.AccountDeterminationRule.ChartOfAccountsId -> cfg.ChartOfAccounts (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountDeterminationRule_ChartOfAccountsId')
    ALTER TABLE [cfg].[AccountDeterminationRule] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountDeterminationRule_ChartOfAccountsId] FOREIGN KEY ([ChartOfAccountsId])
        REFERENCES [cfg].[ChartOfAccounts] ([Id]);
GO

-- cfg.AccountDeterminationRule.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountDeterminationRule_CompanyCodeId')
    ALTER TABLE [cfg].[AccountDeterminationRule] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountDeterminationRule_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- cfg.AccountDeterminationRule.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountDeterminationRule_CurrencyCode')
    ALTER TABLE [cfg].[AccountDeterminationRule] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountDeterminationRule_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- cfg.AccountDeterminationRule.DebitGLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountDeterminationRule_DebitGLAccountId')
    ALTER TABLE [cfg].[AccountDeterminationRule] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountDeterminationRule_DebitGLAccountId] FOREIGN KEY ([DebitGLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- cfg.AccountDeterminationRule.CreditGLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountDeterminationRule_CreditGLAccountId')
    ALTER TABLE [cfg].[AccountDeterminationRule] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountDeterminationRule_CreditGLAccountId] FOREIGN KEY ([CreditGLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- cfg.AccountGroup.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountGroup_TenantId')
    ALTER TABLE [cfg].[AccountGroup] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountGroup_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.AccountGroup.ChartOfAccountsId -> cfg.ChartOfAccounts (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountGroup_ChartOfAccountsId')
    ALTER TABLE [cfg].[AccountGroup] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountGroup_ChartOfAccountsId] FOREIGN KEY ([ChartOfAccountsId])
        REFERENCES [cfg].[ChartOfAccounts] ([Id]);
GO

-- cfg.AccountGroup.FieldStatusGroupId -> cfg.FieldStatusGroup (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountGroup_FieldStatusGroupId')
    ALTER TABLE [cfg].[AccountGroup] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountGroup_FieldStatusGroupId] FOREIGN KEY ([FieldStatusGroupId])
        REFERENCES [cfg].[FieldStatusGroup] ([Id]);
GO

-- cfg.AccountingPrinciple.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_AccountingPrinciple_TenantId')
    ALTER TABLE [cfg].[AccountingPrinciple] WITH CHECK
        ADD CONSTRAINT [FK_cfg_AccountingPrinciple_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.BrowserQueryLog.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_BrowserQueryLog_TenantId')
    ALTER TABLE [cfg].[BrowserQueryLog] WITH CHECK
        ADD CONSTRAINT [FK_cfg_BrowserQueryLog_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.BrowserQueryLog.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_BrowserQueryLog_UserId')
    ALTER TABLE [cfg].[BrowserQueryLog] WITH CHECK
        ADD CONSTRAINT [FK_cfg_BrowserQueryLog_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- cfg.BrowserVariant.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_BrowserVariant_TenantId')
    ALTER TABLE [cfg].[BrowserVariant] WITH CHECK
        ADD CONSTRAINT [FK_cfg_BrowserVariant_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.BrowserVariant.OwnerUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_BrowserVariant_OwnerUserId')
    ALTER TABLE [cfg].[BrowserVariant] WITH CHECK
        ADD CONSTRAINT [FK_cfg_BrowserVariant_OwnerUserId] FOREIGN KEY ([OwnerUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- cfg.BrowserVariantField.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_BrowserVariantField_TenantId')
    ALTER TABLE [cfg].[BrowserVariantField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_BrowserVariantField_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.BrowserVariantField.BrowserVariantId -> cfg.BrowserVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_BrowserVariantField_BrowserVariantId')
    ALTER TABLE [cfg].[BrowserVariantField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_BrowserVariantField_BrowserVariantId] FOREIGN KEY ([BrowserVariantId])
        REFERENCES [cfg].[BrowserVariant] ([Id]);
GO

-- cfg.BrowserVariantFilter.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_BrowserVariantFilter_TenantId')
    ALTER TABLE [cfg].[BrowserVariantFilter] WITH CHECK
        ADD CONSTRAINT [FK_cfg_BrowserVariantFilter_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.BrowserVariantFilter.BrowserVariantId -> cfg.BrowserVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_BrowserVariantFilter_BrowserVariantId')
    ALTER TABLE [cfg].[BrowserVariantFilter] WITH CHECK
        ADD CONSTRAINT [FK_cfg_BrowserVariantFilter_BrowserVariantId] FOREIGN KEY ([BrowserVariantId])
        REFERENCES [cfg].[BrowserVariant] ([Id]);
GO

-- cfg.ChartOfAccounts.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ChartOfAccounts_TenantId')
    ALTER TABLE [cfg].[ChartOfAccounts] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ChartOfAccounts_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.ChartOfAccounts.MaintenanceLanguage -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ChartOfAccounts_MaintenanceLanguage')
    ALTER TABLE [cfg].[ChartOfAccounts] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ChartOfAccounts_MaintenanceLanguage] FOREIGN KEY ([MaintenanceLanguage])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- cfg.ChartOfAccounts.GroupChartOfAccountsId -> cfg.ChartOfAccounts (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ChartOfAccounts_GroupChartOfAccountsId')
    ALTER TABLE [cfg].[ChartOfAccounts] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ChartOfAccounts_GroupChartOfAccountsId] FOREIGN KEY ([GroupChartOfAccountsId])
        REFERENCES [cfg].[ChartOfAccounts] ([Id]);
GO

-- cfg.CorrespondenceForm.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CorrespondenceForm_TenantId')
    ALTER TABLE [cfg].[CorrespondenceForm] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CorrespondenceForm_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CorrespondenceForm.LanguageCode -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CorrespondenceForm_LanguageCode')
    ALTER TABLE [cfg].[CorrespondenceForm] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CorrespondenceForm_LanguageCode] FOREIGN KEY ([LanguageCode])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- cfg.Country.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_Country_TenantId')
    ALTER TABLE [cfg].[Country] WITH CHECK
        ADD CONSTRAINT [FK_cfg_Country_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.Country.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_Country_CurrencyCode')
    ALTER TABLE [cfg].[Country] WITH CHECK
        ADD CONSTRAINT [FK_cfg_Country_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- cfg.Country.LanguageCode -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_Country_LanguageCode')
    ALTER TABLE [cfg].[Country] WITH CHECK
        ADD CONSTRAINT [FK_cfg_Country_LanguageCode] FOREIGN KEY ([LanguageCode])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- cfg.Currency.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_Currency_TenantId')
    ALTER TABLE [cfg].[Currency] WITH CHECK
        ADD CONSTRAINT [FK_cfg_Currency_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CurrencyDecimal.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CurrencyDecimal_TenantId')
    ALTER TABLE [cfg].[CurrencyDecimal] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CurrencyDecimal_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CurrencyDecimal.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CurrencyDecimal_CurrencyCode')
    ALTER TABLE [cfg].[CurrencyDecimal] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CurrencyDecimal_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- cfg.CurrencyTranslationRatio.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CurrencyTranslationRatio_TenantId')
    ALTER TABLE [cfg].[CurrencyTranslationRatio] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CurrencyTranslationRatio_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CurrencyTranslationRatio.ExchangeRateTypeId -> cfg.ExchangeRateType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CurrencyTranslationRatio_ExchangeRateTypeId')
    ALTER TABLE [cfg].[CurrencyTranslationRatio] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CurrencyTranslationRatio_ExchangeRateTypeId] FOREIGN KEY ([ExchangeRateTypeId])
        REFERENCES [cfg].[ExchangeRateType] ([Id]);
GO

-- cfg.CurrencyTranslationRatio.FromCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CurrencyTranslationRatio_FromCurrencyCode')
    ALTER TABLE [cfg].[CurrencyTranslationRatio] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CurrencyTranslationRatio_FromCurrencyCode] FOREIGN KEY ([TenantId], [FromCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- cfg.CurrencyTranslationRatio.ToCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CurrencyTranslationRatio_ToCurrencyCode')
    ALTER TABLE [cfg].[CurrencyTranslationRatio] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CurrencyTranslationRatio_ToCurrencyCode] FOREIGN KEY ([TenantId], [ToCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- cfg.CustomFieldDefinition.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomFieldDefinition_TenantId')
    ALTER TABLE [cfg].[CustomFieldDefinition] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomFieldDefinition_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CustomFieldDefinition.DataElementId -> cfg.DictionaryDataElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomFieldDefinition_DataElementId')
    ALTER TABLE [cfg].[CustomFieldDefinition] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomFieldDefinition_DataElementId] FOREIGN KEY ([DataElementId])
        REFERENCES [cfg].[DictionaryDataElement] ([Id]);
GO

-- cfg.CustomFieldDefinition.SearchHelpId -> cfg.DictionarySearchHelp (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomFieldDefinition_SearchHelpId')
    ALTER TABLE [cfg].[CustomFieldDefinition] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomFieldDefinition_SearchHelpId] FOREIGN KEY ([SearchHelpId])
        REFERENCES [cfg].[DictionarySearchHelp] ([Id]);
GO

-- cfg.CustomFieldDefinition.AuthorizationGroup -> cfg.TableAuthorizationGroup (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomFieldDefinition_AuthorizationGroup')
    ALTER TABLE [cfg].[CustomFieldDefinition] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomFieldDefinition_AuthorizationGroup] FOREIGN KEY ([TenantId], [AuthorizationGroup])
        REFERENCES [cfg].[TableAuthorizationGroup] ([TenantId], [AuthorizationGroup]);
GO

-- cfg.CustomFieldValue.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomFieldValue_TenantId')
    ALTER TABLE [cfg].[CustomFieldValue] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomFieldValue_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CustomFieldValue.CustomFieldDefinitionId -> cfg.CustomFieldDefinition (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomFieldValue_CustomFieldDefinitionId')
    ALTER TABLE [cfg].[CustomFieldValue] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomFieldValue_CustomFieldDefinitionId] FOREIGN KEY ([CustomFieldDefinitionId])
        REFERENCES [cfg].[CustomFieldDefinition] ([Id]);
GO

-- cfg.CustomObjectRequest.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomObjectRequest_TenantId')
    ALTER TABLE [cfg].[CustomObjectRequest] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomObjectRequest_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CustomObjectRequestItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomObjectRequestItem_TenantId')
    ALTER TABLE [cfg].[CustomObjectRequestItem] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomObjectRequestItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CustomObjectRequestItem.CustomObjectRequestId -> cfg.CustomObjectRequest (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomObjectRequestItem_CustomObjectRequestId')
    ALTER TABLE [cfg].[CustomObjectRequestItem] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomObjectRequestItem_CustomObjectRequestId] FOREIGN KEY ([CustomObjectRequestId])
        REFERENCES [cfg].[CustomObjectRequest] ([Id]);
GO

-- cfg.CustomTable.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomTable_TenantId')
    ALTER TABLE [cfg].[CustomTable] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomTable_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CustomTable.AuthorizationGroup -> cfg.TableAuthorizationGroup (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomTable_AuthorizationGroup')
    ALTER TABLE [cfg].[CustomTable] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomTable_AuthorizationGroup] FOREIGN KEY ([TenantId], [AuthorizationGroup])
        REFERENCES [cfg].[TableAuthorizationGroup] ([TenantId], [AuthorizationGroup]);
GO

-- cfg.CustomTable.NumberRangeObjectId -> cfg.NumberRangeObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomTable_NumberRangeObjectId')
    ALTER TABLE [cfg].[CustomTable] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomTable_NumberRangeObjectId] FOREIGN KEY ([NumberRangeObjectId])
        REFERENCES [cfg].[NumberRangeObject] ([Id]);
GO

-- cfg.CustomTable.DictionaryObjectId -> cfg.DictionaryObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomTable_DictionaryObjectId')
    ALTER TABLE [cfg].[CustomTable] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomTable_DictionaryObjectId] FOREIGN KEY ([DictionaryObjectId])
        REFERENCES [cfg].[DictionaryObject] ([Id]);
GO

-- cfg.CustomTableField.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomTableField_TenantId')
    ALTER TABLE [cfg].[CustomTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomTableField_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.CustomTableField.CustomTableId -> cfg.CustomTable (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomTableField_CustomTableId')
    ALTER TABLE [cfg].[CustomTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomTableField_CustomTableId] FOREIGN KEY ([CustomTableId])
        REFERENCES [cfg].[CustomTable] ([Id]);
GO

-- cfg.CustomTableField.DataElementId -> cfg.DictionaryDataElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomTableField_DataElementId')
    ALTER TABLE [cfg].[CustomTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomTableField_DataElementId] FOREIGN KEY ([DataElementId])
        REFERENCES [cfg].[DictionaryDataElement] ([Id]);
GO

-- cfg.CustomTableField.DictionaryDomainId -> cfg.DictionaryDomain (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomTableField_DictionaryDomainId')
    ALTER TABLE [cfg].[CustomTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomTableField_DictionaryDomainId] FOREIGN KEY ([DictionaryDomainId])
        REFERENCES [cfg].[DictionaryDomain] ([Id]);
GO

-- cfg.CustomTableField.SearchHelpId -> cfg.DictionarySearchHelp (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_CustomTableField_SearchHelpId')
    ALTER TABLE [cfg].[CustomTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_CustomTableField_SearchHelpId] FOREIGN KEY ([SearchHelpId])
        REFERENCES [cfg].[DictionarySearchHelp] ([Id]);
GO

-- cfg.DictionaryChangeLog.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryChangeLog_TenantId')
    ALTER TABLE [cfg].[DictionaryChangeLog] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryChangeLog_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryChangeLog.DictionaryObjectId -> cfg.DictionaryObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryChangeLog_DictionaryObjectId')
    ALTER TABLE [cfg].[DictionaryChangeLog] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryChangeLog_DictionaryObjectId] FOREIGN KEY ([DictionaryObjectId])
        REFERENCES [cfg].[DictionaryObject] ([Id]);
GO

-- cfg.DictionaryChangeLog.MigrationScriptId -> cfg.MigrationScript (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryChangeLog_MigrationScriptId')
    ALTER TABLE [cfg].[DictionaryChangeLog] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryChangeLog_MigrationScriptId] FOREIGN KEY ([MigrationScriptId])
        REFERENCES [cfg].[MigrationScript] ([Id]);
GO

-- cfg.DictionaryChangeLog.TransportRequestId -> cfg.CustomObjectRequest (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryChangeLog_TransportRequestId')
    ALTER TABLE [cfg].[DictionaryChangeLog] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryChangeLog_TransportRequestId] FOREIGN KEY ([TransportRequestId])
        REFERENCES [cfg].[CustomObjectRequest] ([Id]);
GO

-- cfg.DictionaryDataElement.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryDataElement_TenantId')
    ALTER TABLE [cfg].[DictionaryDataElement] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryDataElement_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryDataElement.DictionaryDomainId -> cfg.DictionaryDomain (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryDataElement_DictionaryDomainId')
    ALTER TABLE [cfg].[DictionaryDataElement] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryDataElement_DictionaryDomainId] FOREIGN KEY ([DictionaryDomainId])
        REFERENCES [cfg].[DictionaryDomain] ([Id]);
GO

-- cfg.DictionaryDataElement.SearchHelpId -> cfg.DictionarySearchHelp (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryDataElement_SearchHelpId')
    ALTER TABLE [cfg].[DictionaryDataElement] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryDataElement_SearchHelpId] FOREIGN KEY ([SearchHelpId])
        REFERENCES [cfg].[DictionarySearchHelp] ([Id]);
GO

-- cfg.DictionaryDomain.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryDomain_TenantId')
    ALTER TABLE [cfg].[DictionaryDomain] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryDomain_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryDomainValue.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryDomainValue_TenantId')
    ALTER TABLE [cfg].[DictionaryDomainValue] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryDomainValue_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryDomainValue.DictionaryDomainId -> cfg.DictionaryDomain (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryDomainValue_DictionaryDomainId')
    ALTER TABLE [cfg].[DictionaryDomainValue] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryDomainValue_DictionaryDomainId] FOREIGN KEY ([DictionaryDomainId])
        REFERENCES [cfg].[DictionaryDomain] ([Id]);
GO

-- cfg.DictionaryDomainValue.LanguageCode -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryDomainValue_LanguageCode')
    ALTER TABLE [cfg].[DictionaryDomainValue] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryDomainValue_LanguageCode] FOREIGN KEY ([LanguageCode])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- cfg.DictionaryForeignKey.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryForeignKey_TenantId')
    ALTER TABLE [cfg].[DictionaryForeignKey] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryForeignKey_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryForeignKeyField.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryForeignKeyField_TenantId')
    ALTER TABLE [cfg].[DictionaryForeignKeyField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryForeignKeyField_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryForeignKeyField.DictionaryForeignKeyId -> cfg.DictionaryForeignKey (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryForeignKeyField_DictionaryForeignKeyId')
    ALTER TABLE [cfg].[DictionaryForeignKeyField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryForeignKeyField_DictionaryForeignKeyId] FOREIGN KEY ([DictionaryForeignKeyId])
        REFERENCES [cfg].[DictionaryForeignKey] ([Id]);
GO

-- cfg.DictionaryIndex.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryIndex_TenantId')
    ALTER TABLE [cfg].[DictionaryIndex] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryIndex_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryIndex.DictionaryTableId -> cfg.DictionaryTable (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryIndex_DictionaryTableId')
    ALTER TABLE [cfg].[DictionaryIndex] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryIndex_DictionaryTableId] FOREIGN KEY ([DictionaryTableId])
        REFERENCES [cfg].[DictionaryTable] ([Id]);
GO

-- cfg.DictionaryIndexField.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryIndexField_TenantId')
    ALTER TABLE [cfg].[DictionaryIndexField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryIndexField_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryIndexField.DictionaryIndexId -> cfg.DictionaryIndex (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryIndexField_DictionaryIndexId')
    ALTER TABLE [cfg].[DictionaryIndexField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryIndexField_DictionaryIndexId] FOREIGN KEY ([DictionaryIndexId])
        REFERENCES [cfg].[DictionaryIndex] ([Id]);
GO

-- cfg.DictionaryLockObject.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryLockObject_TenantId')
    ALTER TABLE [cfg].[DictionaryLockObject] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryLockObject_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryObject.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryObject_TenantId')
    ALTER TABLE [cfg].[DictionaryObject] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryObject_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionarySearchHelp.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionarySearchHelp_TenantId')
    ALTER TABLE [cfg].[DictionarySearchHelp] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionarySearchHelp_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionarySearchHelpParameter.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionarySearchHelpParameter_TenantId')
    ALTER TABLE [cfg].[DictionarySearchHelpParameter] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionarySearchHelpParameter_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionarySearchHelpParameter.DictionarySearchHelpId -> cfg.DictionarySearchHelp (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionarySearchHelpParameter_DictionarySearchHelpId')
    ALTER TABLE [cfg].[DictionarySearchHelpParameter] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionarySearchHelpParameter_DictionarySearchHelpId] FOREIGN KEY ([DictionarySearchHelpId])
        REFERENCES [cfg].[DictionarySearchHelp] ([Id]);
GO

-- cfg.DictionarySearchHelpParameter.DataElementId -> cfg.DictionaryDataElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionarySearchHelpParameter_DataElementId')
    ALTER TABLE [cfg].[DictionarySearchHelpParameter] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionarySearchHelpParameter_DataElementId] FOREIGN KEY ([DataElementId])
        REFERENCES [cfg].[DictionaryDataElement] ([Id]);
GO

-- cfg.DictionaryStructure.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryStructure_TenantId')
    ALTER TABLE [cfg].[DictionaryStructure] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryStructure_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryStructureField.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryStructureField_TenantId')
    ALTER TABLE [cfg].[DictionaryStructureField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryStructureField_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryStructureField.DictionaryStructureId -> cfg.DictionaryStructure (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryStructureField_DictionaryStructureId')
    ALTER TABLE [cfg].[DictionaryStructureField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryStructureField_DictionaryStructureId] FOREIGN KEY ([DictionaryStructureId])
        REFERENCES [cfg].[DictionaryStructure] ([Id]);
GO

-- cfg.DictionaryStructureField.DataElementId -> cfg.DictionaryDataElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryStructureField_DataElementId')
    ALTER TABLE [cfg].[DictionaryStructureField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryStructureField_DataElementId] FOREIGN KEY ([DataElementId])
        REFERENCES [cfg].[DictionaryDataElement] ([Id]);
GO

-- cfg.DictionaryTable.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryTable_TenantId')
    ALTER TABLE [cfg].[DictionaryTable] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryTable_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryTableField.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryTableField_TenantId')
    ALTER TABLE [cfg].[DictionaryTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryTableField_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryTableField.DictionaryTableId -> cfg.DictionaryTable (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryTableField_DictionaryTableId')
    ALTER TABLE [cfg].[DictionaryTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryTableField_DictionaryTableId] FOREIGN KEY ([DictionaryTableId])
        REFERENCES [cfg].[DictionaryTable] ([Id]);
GO

-- cfg.DictionaryTableField.DataElementId -> cfg.DictionaryDataElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryTableField_DataElementId')
    ALTER TABLE [cfg].[DictionaryTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryTableField_DataElementId] FOREIGN KEY ([DataElementId])
        REFERENCES [cfg].[DictionaryDataElement] ([Id]);
GO

-- cfg.DictionaryTableField.ForeignKeyId -> cfg.DictionaryForeignKey (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryTableField_ForeignKeyId')
    ALTER TABLE [cfg].[DictionaryTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryTableField_ForeignKeyId] FOREIGN KEY ([ForeignKeyId])
        REFERENCES [cfg].[DictionaryForeignKey] ([Id]);
GO

-- cfg.DictionaryTableField.SearchHelpId -> cfg.DictionarySearchHelp (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryTableField_SearchHelpId')
    ALTER TABLE [cfg].[DictionaryTableField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryTableField_SearchHelpId] FOREIGN KEY ([SearchHelpId])
        REFERENCES [cfg].[DictionarySearchHelp] ([Id]);
GO

-- cfg.DictionaryView.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryView_TenantId')
    ALTER TABLE [cfg].[DictionaryView] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryView_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryViewField.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryViewField_TenantId')
    ALTER TABLE [cfg].[DictionaryViewField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryViewField_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DictionaryViewField.DictionaryViewId -> cfg.DictionaryView (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DictionaryViewField_DictionaryViewId')
    ALTER TABLE [cfg].[DictionaryViewField] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DictionaryViewField_DictionaryViewId] FOREIGN KEY ([DictionaryViewId])
        REFERENCES [cfg].[DictionaryView] ([Id]);
GO

-- cfg.DocumentType.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DocumentType_TenantId')
    ALTER TABLE [cfg].[DocumentType] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DocumentType_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DocumentType.NumberRangeObjectId -> cfg.NumberRangeObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DocumentType_NumberRangeObjectId')
    ALTER TABLE [cfg].[DocumentType] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DocumentType_NumberRangeObjectId] FOREIGN KEY ([NumberRangeObjectId])
        REFERENCES [cfg].[NumberRangeObject] ([Id]);
GO

-- cfg.DunningLevel.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DunningLevel_TenantId')
    ALTER TABLE [cfg].[DunningLevel] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DunningLevel_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.DunningLevel.DunningProcedureId -> cfg.DunningProcedure (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DunningLevel_DunningProcedureId')
    ALTER TABLE [cfg].[DunningLevel] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DunningLevel_DunningProcedureId] FOREIGN KEY ([DunningProcedureId])
        REFERENCES [cfg].[DunningProcedure] ([Id]);
GO

-- cfg.DunningLevel.FormId -> cfg.CorrespondenceForm (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DunningLevel_FormId')
    ALTER TABLE [cfg].[DunningLevel] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DunningLevel_FormId] FOREIGN KEY ([FormId])
        REFERENCES [cfg].[CorrespondenceForm] ([Id]);
GO

-- cfg.DunningProcedure.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_DunningProcedure_TenantId')
    ALTER TABLE [cfg].[DunningProcedure] WITH CHECK
        ADD CONSTRAINT [FK_cfg_DunningProcedure_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.ExchangeRate.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ExchangeRate_TenantId')
    ALTER TABLE [cfg].[ExchangeRate] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ExchangeRate_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.ExchangeRate.ExchangeRateTypeId -> cfg.ExchangeRateType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ExchangeRate_ExchangeRateTypeId')
    ALTER TABLE [cfg].[ExchangeRate] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ExchangeRate_ExchangeRateTypeId] FOREIGN KEY ([ExchangeRateTypeId])
        REFERENCES [cfg].[ExchangeRateType] ([Id]);
GO

-- cfg.ExchangeRate.FromCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ExchangeRate_FromCurrencyCode')
    ALTER TABLE [cfg].[ExchangeRate] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ExchangeRate_FromCurrencyCode] FOREIGN KEY ([TenantId], [FromCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- cfg.ExchangeRate.ToCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ExchangeRate_ToCurrencyCode')
    ALTER TABLE [cfg].[ExchangeRate] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ExchangeRate_ToCurrencyCode] FOREIGN KEY ([TenantId], [ToCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- cfg.ExchangeRateType.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ExchangeRateType_TenantId')
    ALTER TABLE [cfg].[ExchangeRateType] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ExchangeRateType_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.ExchangeRateType.ReferenceCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ExchangeRateType_ReferenceCurrencyCode')
    ALTER TABLE [cfg].[ExchangeRateType] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ExchangeRateType_ReferenceCurrencyCode] FOREIGN KEY ([TenantId], [ReferenceCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- cfg.FieldStatusFieldControl.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FieldStatusFieldControl_TenantId')
    ALTER TABLE [cfg].[FieldStatusFieldControl] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FieldStatusFieldControl_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.FieldStatusFieldControl.FieldStatusGroupId -> cfg.FieldStatusGroup (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FieldStatusFieldControl_FieldStatusGroupId')
    ALTER TABLE [cfg].[FieldStatusFieldControl] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FieldStatusFieldControl_FieldStatusGroupId] FOREIGN KEY ([FieldStatusGroupId])
        REFERENCES [cfg].[FieldStatusGroup] ([Id]);
GO

-- cfg.FieldStatusGroup.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FieldStatusGroup_TenantId')
    ALTER TABLE [cfg].[FieldStatusGroup] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FieldStatusGroup_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.FieldStatusGroup.FieldStatusVariantId -> cfg.FieldStatusVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FieldStatusGroup_FieldStatusVariantId')
    ALTER TABLE [cfg].[FieldStatusGroup] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FieldStatusGroup_FieldStatusVariantId] FOREIGN KEY ([FieldStatusVariantId])
        REFERENCES [cfg].[FieldStatusVariant] ([Id]);
GO

-- cfg.FieldStatusVariant.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FieldStatusVariant_TenantId')
    ALTER TABLE [cfg].[FieldStatusVariant] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FieldStatusVariant_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.FinancialStatementNode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FinancialStatementNode_TenantId')
    ALTER TABLE [cfg].[FinancialStatementNode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FinancialStatementNode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.FinancialStatementNode.FinancialStatementVersionId -> cfg.FinancialStatementVersion (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FinancialStatementNode_FinancialStatementVersionId')
    ALTER TABLE [cfg].[FinancialStatementNode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FinancialStatementNode_FinancialStatementVersionId] FOREIGN KEY ([FinancialStatementVersionId])
        REFERENCES [cfg].[FinancialStatementVersion] ([Id]);
GO

-- cfg.FinancialStatementNode.ParentNodeId -> cfg.FinancialStatementNode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FinancialStatementNode_ParentNodeId')
    ALTER TABLE [cfg].[FinancialStatementNode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FinancialStatementNode_ParentNodeId] FOREIGN KEY ([ParentNodeId])
        REFERENCES [cfg].[FinancialStatementNode] ([Id]);
GO

-- cfg.FinancialStatementNodeAccount.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FinancialStatementNodeAccount_TenantId')
    ALTER TABLE [cfg].[FinancialStatementNodeAccount] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FinancialStatementNodeAccount_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.FinancialStatementNodeAccount.FinancialStatementNodeId -> cfg.FinancialStatementNode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FinancialStatementNodeAccount_FinancialStatementNodeId')
    ALTER TABLE [cfg].[FinancialStatementNodeAccount] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FinancialStatementNodeAccount_FinancialStatementNodeId] FOREIGN KEY ([FinancialStatementNodeId])
        REFERENCES [cfg].[FinancialStatementNode] ([Id]);
GO

-- cfg.FinancialStatementVersion.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FinancialStatementVersion_TenantId')
    ALTER TABLE [cfg].[FinancialStatementVersion] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FinancialStatementVersion_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.FinancialStatementVersion.ChartOfAccountsId -> cfg.ChartOfAccounts (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FinancialStatementVersion_ChartOfAccountsId')
    ALTER TABLE [cfg].[FinancialStatementVersion] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FinancialStatementVersion_ChartOfAccountsId] FOREIGN KEY ([ChartOfAccountsId])
        REFERENCES [cfg].[ChartOfAccounts] ([Id]);
GO

-- cfg.FinancialStatementVersion.MaintenanceLanguage -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FinancialStatementVersion_MaintenanceLanguage')
    ALTER TABLE [cfg].[FinancialStatementVersion] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FinancialStatementVersion_MaintenanceLanguage] FOREIGN KEY ([MaintenanceLanguage])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- cfg.FinancialStatementVersion.AccountingPrincipleId -> cfg.AccountingPrinciple (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FinancialStatementVersion_AccountingPrincipleId')
    ALTER TABLE [cfg].[FinancialStatementVersion] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FinancialStatementVersion_AccountingPrincipleId] FOREIGN KEY ([AccountingPrincipleId])
        REFERENCES [cfg].[AccountingPrinciple] ([Id]);
GO

-- cfg.FiscalPeriod.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FiscalPeriod_TenantId')
    ALTER TABLE [cfg].[FiscalPeriod] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FiscalPeriod_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.FiscalPeriod.FiscalYearVariantId -> cfg.FiscalYearVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FiscalPeriod_FiscalYearVariantId')
    ALTER TABLE [cfg].[FiscalPeriod] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FiscalPeriod_FiscalYearVariantId] FOREIGN KEY ([FiscalYearVariantId])
        REFERENCES [cfg].[FiscalYearVariant] ([Id]);
GO

-- cfg.FiscalYearVariant.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FiscalYearVariant_TenantId')
    ALTER TABLE [cfg].[FiscalYearVariant] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FiscalYearVariant_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.FiscalYearVariantPeriod.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FiscalYearVariantPeriod_TenantId')
    ALTER TABLE [cfg].[FiscalYearVariantPeriod] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FiscalYearVariantPeriod_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.FiscalYearVariantPeriod.FiscalYearVariantId -> cfg.FiscalYearVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_FiscalYearVariantPeriod_FiscalYearVariantId')
    ALTER TABLE [cfg].[FiscalYearVariantPeriod] WITH CHECK
        ADD CONSTRAINT [FK_cfg_FiscalYearVariantPeriod_FiscalYearVariantId] FOREIGN KEY ([FiscalYearVariantId])
        REFERENCES [cfg].[FiscalYearVariant] ([Id]);
GO

-- cfg.Ledger.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_Ledger_TenantId')
    ALTER TABLE [cfg].[Ledger] WITH CHECK
        ADD CONSTRAINT [FK_cfg_Ledger_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.Ledger.AccountingPrincipleId -> cfg.AccountingPrinciple (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_Ledger_AccountingPrincipleId')
    ALTER TABLE [cfg].[Ledger] WITH CHECK
        ADD CONSTRAINT [FK_cfg_Ledger_AccountingPrincipleId] FOREIGN KEY ([AccountingPrincipleId])
        REFERENCES [cfg].[AccountingPrinciple] ([Id]);
GO

-- cfg.Ledger.UnderlyingLedgerId -> cfg.Ledger (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_Ledger_UnderlyingLedgerId')
    ALTER TABLE [cfg].[Ledger] WITH CHECK
        ADD CONSTRAINT [FK_cfg_Ledger_UnderlyingLedgerId] FOREIGN KEY ([UnderlyingLedgerId])
        REFERENCES [cfg].[Ledger] ([Id]);
GO

-- cfg.LedgerCompanyCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_LedgerCompanyCode_TenantId')
    ALTER TABLE [cfg].[LedgerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_LedgerCompanyCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.LedgerCompanyCode.LedgerId -> cfg.Ledger (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_LedgerCompanyCode_LedgerId')
    ALTER TABLE [cfg].[LedgerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_LedgerCompanyCode_LedgerId] FOREIGN KEY ([LedgerId])
        REFERENCES [cfg].[Ledger] ([Id]);
GO

-- cfg.LedgerCompanyCode.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_LedgerCompanyCode_CompanyCodeId')
    ALTER TABLE [cfg].[LedgerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_LedgerCompanyCode_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- cfg.LedgerCompanyCode.FiscalYearVariantId -> cfg.FiscalYearVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_LedgerCompanyCode_FiscalYearVariantId')
    ALTER TABLE [cfg].[LedgerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_LedgerCompanyCode_FiscalYearVariantId] FOREIGN KEY ([FiscalYearVariantId])
        REFERENCES [cfg].[FiscalYearVariant] ([Id]);
GO

-- cfg.LedgerCompanyCode.PostingPeriodVariantId -> cfg.PostingPeriodVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_LedgerCompanyCode_PostingPeriodVariantId')
    ALTER TABLE [cfg].[LedgerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_LedgerCompanyCode_PostingPeriodVariantId] FOREIGN KEY ([PostingPeriodVariantId])
        REFERENCES [cfg].[PostingPeriodVariant] ([Id]);
GO

-- cfg.MigrationScript.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_MigrationScript_TenantId')
    ALTER TABLE [cfg].[MigrationScript] WITH CHECK
        ADD CONSTRAINT [FK_cfg_MigrationScript_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.MigrationScript.DictionaryObjectId -> cfg.DictionaryObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_MigrationScript_DictionaryObjectId')
    ALTER TABLE [cfg].[MigrationScript] WITH CHECK
        ADD CONSTRAINT [FK_cfg_MigrationScript_DictionaryObjectId] FOREIGN KEY ([DictionaryObjectId])
        REFERENCES [cfg].[DictionaryObject] ([Id]);
GO

-- cfg.MigrationScript.CustomObjectRequestId -> cfg.CustomObjectRequest (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_MigrationScript_CustomObjectRequestId')
    ALTER TABLE [cfg].[MigrationScript] WITH CHECK
        ADD CONSTRAINT [FK_cfg_MigrationScript_CustomObjectRequestId] FOREIGN KEY ([CustomObjectRequestId])
        REFERENCES [cfg].[CustomObjectRequest] ([Id]);
GO

-- cfg.NumberRangeGap.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_NumberRangeGap_TenantId')
    ALTER TABLE [cfg].[NumberRangeGap] WITH CHECK
        ADD CONSTRAINT [FK_cfg_NumberRangeGap_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.NumberRangeGap.NumberRangeIntervalId -> cfg.NumberRangeInterval (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_NumberRangeGap_NumberRangeIntervalId')
    ALTER TABLE [cfg].[NumberRangeGap] WITH CHECK
        ADD CONSTRAINT [FK_cfg_NumberRangeGap_NumberRangeIntervalId] FOREIGN KEY ([NumberRangeIntervalId])
        REFERENCES [cfg].[NumberRangeInterval] ([Id]);
GO

-- cfg.NumberRangeInterval.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_NumberRangeInterval_TenantId')
    ALTER TABLE [cfg].[NumberRangeInterval] WITH CHECK
        ADD CONSTRAINT [FK_cfg_NumberRangeInterval_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.NumberRangeInterval.NumberRangeObjectId -> cfg.NumberRangeObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_NumberRangeInterval_NumberRangeObjectId')
    ALTER TABLE [cfg].[NumberRangeInterval] WITH CHECK
        ADD CONSTRAINT [FK_cfg_NumberRangeInterval_NumberRangeObjectId] FOREIGN KEY ([NumberRangeObjectId])
        REFERENCES [cfg].[NumberRangeObject] ([Id]);
GO

-- cfg.NumberRangeInterval.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_NumberRangeInterval_CompanyCodeId')
    ALTER TABLE [cfg].[NumberRangeInterval] WITH CHECK
        ADD CONSTRAINT [FK_cfg_NumberRangeInterval_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- cfg.NumberRangeObject.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_NumberRangeObject_TenantId')
    ALTER TABLE [cfg].[NumberRangeObject] WITH CHECK
        ADD CONSTRAINT [FK_cfg_NumberRangeObject_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.PaymentMethod.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PaymentMethod_TenantId')
    ALTER TABLE [cfg].[PaymentMethod] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PaymentMethod_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.PaymentMethod.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PaymentMethod_CountryCode')
    ALTER TABLE [cfg].[PaymentMethod] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PaymentMethod_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- cfg.PaymentMethod.DocumentTypeId -> cfg.DocumentType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PaymentMethod_DocumentTypeId')
    ALTER TABLE [cfg].[PaymentMethod] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PaymentMethod_DocumentTypeId] FOREIGN KEY ([DocumentTypeId])
        REFERENCES [cfg].[DocumentType] ([Id]);
GO

-- cfg.PaymentTerms.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PaymentTerms_TenantId')
    ALTER TABLE [cfg].[PaymentTerms] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PaymentTerms_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.PaymentTermsInstallment.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PaymentTermsInstallment_TenantId')
    ALTER TABLE [cfg].[PaymentTermsInstallment] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PaymentTermsInstallment_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.PaymentTermsInstallment.PaymentTermsId -> cfg.PaymentTerms (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PaymentTermsInstallment_PaymentTermsId')
    ALTER TABLE [cfg].[PaymentTermsInstallment] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PaymentTermsInstallment_PaymentTermsId] FOREIGN KEY ([PaymentTermsId])
        REFERENCES [cfg].[PaymentTerms] ([Id]);
GO

-- cfg.PaymentTermsInstallment.InstallmentPaymentTermsId -> cfg.PaymentTerms (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PaymentTermsInstallment_InstallmentPaymentTermsId')
    ALTER TABLE [cfg].[PaymentTermsInstallment] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PaymentTermsInstallment_InstallmentPaymentTermsId] FOREIGN KEY ([InstallmentPaymentTermsId])
        REFERENCES [cfg].[PaymentTerms] ([Id]);
GO

-- cfg.PostingKey.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PostingKey_TenantId')
    ALTER TABLE [cfg].[PostingKey] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PostingKey_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.PostingKey.FieldStatusGroupId -> cfg.FieldStatusGroup (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PostingKey_FieldStatusGroupId')
    ALTER TABLE [cfg].[PostingKey] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PostingKey_FieldStatusGroupId] FOREIGN KEY ([FieldStatusGroupId])
        REFERENCES [cfg].[FieldStatusGroup] ([Id]);
GO

-- cfg.PostingPeriodControl.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PostingPeriodControl_TenantId')
    ALTER TABLE [cfg].[PostingPeriodControl] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PostingPeriodControl_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.PostingPeriodControl.PostingPeriodVariantId -> cfg.PostingPeriodVariant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PostingPeriodControl_PostingPeriodVariantId')
    ALTER TABLE [cfg].[PostingPeriodControl] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PostingPeriodControl_PostingPeriodVariantId] FOREIGN KEY ([PostingPeriodVariantId])
        REFERENCES [cfg].[PostingPeriodVariant] ([Id]);
GO

-- cfg.PostingPeriodVariant.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_PostingPeriodVariant_TenantId')
    ALTER TABLE [cfg].[PostingPeriodVariant] WITH CHECK
        ADD CONSTRAINT [FK_cfg_PostingPeriodVariant_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.Region.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_Region_TenantId')
    ALTER TABLE [cfg].[Region] WITH CHECK
        ADD CONSTRAINT [FK_cfg_Region_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.Region.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_Region_CountryCode')
    ALTER TABLE [cfg].[Region] WITH CHECK
        ADD CONSTRAINT [FK_cfg_Region_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- cfg.SpecialGLAccount.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_SpecialGLAccount_TenantId')
    ALTER TABLE [cfg].[SpecialGLAccount] WITH CHECK
        ADD CONSTRAINT [FK_cfg_SpecialGLAccount_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.SpecialGLAccount.ChartOfAccountsId -> cfg.ChartOfAccounts (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_SpecialGLAccount_ChartOfAccountsId')
    ALTER TABLE [cfg].[SpecialGLAccount] WITH CHECK
        ADD CONSTRAINT [FK_cfg_SpecialGLAccount_ChartOfAccountsId] FOREIGN KEY ([ChartOfAccountsId])
        REFERENCES [cfg].[ChartOfAccounts] ([Id]);
GO

-- cfg.SpecialGLAccount.SpecialGLIndicatorId -> cfg.SpecialGLIndicator (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_SpecialGLAccount_SpecialGLIndicatorId')
    ALTER TABLE [cfg].[SpecialGLAccount] WITH CHECK
        ADD CONSTRAINT [FK_cfg_SpecialGLAccount_SpecialGLIndicatorId] FOREIGN KEY ([SpecialGLIndicatorId])
        REFERENCES [cfg].[SpecialGLIndicator] ([Id]);
GO

-- cfg.SpecialGLAccount.ReconciliationGLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_SpecialGLAccount_ReconciliationGLAccountId')
    ALTER TABLE [cfg].[SpecialGLAccount] WITH CHECK
        ADD CONSTRAINT [FK_cfg_SpecialGLAccount_ReconciliationGLAccountId] FOREIGN KEY ([ReconciliationGLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- cfg.SpecialGLAccount.SpecialGLAccountId -> cfg.SpecialGLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_SpecialGLAccount_SpecialGLAccountId')
    ALTER TABLE [cfg].[SpecialGLAccount] WITH CHECK
        ADD CONSTRAINT [FK_cfg_SpecialGLAccount_SpecialGLAccountId] FOREIGN KEY ([SpecialGLAccountId])
        REFERENCES [cfg].[SpecialGLAccount] ([Id]);
GO

-- cfg.SpecialGLIndicator.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_SpecialGLIndicator_TenantId')
    ALTER TABLE [cfg].[SpecialGLIndicator] WITH CHECK
        ADD CONSTRAINT [FK_cfg_SpecialGLIndicator_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.TableAuthorizationGroup.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_TableAuthorizationGroup_TenantId')
    ALTER TABLE [cfg].[TableAuthorizationGroup] WITH CHECK
        ADD CONSTRAINT [FK_cfg_TableAuthorizationGroup_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.TaxCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_TaxCode_TenantId')
    ALTER TABLE [cfg].[TaxCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_TaxCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.TaxCode.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_TaxCode_CountryCode')
    ALTER TABLE [cfg].[TaxCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_TaxCode_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- cfg.TaxCodeRate.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_TaxCodeRate_TenantId')
    ALTER TABLE [cfg].[TaxCodeRate] WITH CHECK
        ADD CONSTRAINT [FK_cfg_TaxCodeRate_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.TaxCodeRate.TaxCodeId -> cfg.TaxCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_TaxCodeRate_TaxCodeId')
    ALTER TABLE [cfg].[TaxCodeRate] WITH CHECK
        ADD CONSTRAINT [FK_cfg_TaxCodeRate_TaxCodeId] FOREIGN KEY ([TaxCodeId])
        REFERENCES [cfg].[TaxCode] ([Id]);
GO

-- cfg.TaxJurisdiction.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_TaxJurisdiction_TenantId')
    ALTER TABLE [cfg].[TaxJurisdiction] WITH CHECK
        ADD CONSTRAINT [FK_cfg_TaxJurisdiction_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.TaxJurisdiction.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_TaxJurisdiction_CountryCode')
    ALTER TABLE [cfg].[TaxJurisdiction] WITH CHECK
        ADD CONSTRAINT [FK_cfg_TaxJurisdiction_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- cfg.TaxJurisdiction.ParentJurisdictionId -> cfg.TaxJurisdiction (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_TaxJurisdiction_ParentJurisdictionId')
    ALTER TABLE [cfg].[TaxJurisdiction] WITH CHECK
        ADD CONSTRAINT [FK_cfg_TaxJurisdiction_ParentJurisdictionId] FOREIGN KEY ([ParentJurisdictionId])
        REFERENCES [cfg].[TaxJurisdiction] ([Id]);
GO

-- cfg.ToleranceGroup.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ToleranceGroup_TenantId')
    ALTER TABLE [cfg].[ToleranceGroup] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ToleranceGroup_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.ToleranceGroup.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_ToleranceGroup_CompanyCodeId')
    ALTER TABLE [cfg].[ToleranceGroup] WITH CHECK
        ADD CONSTRAINT [FK_cfg_ToleranceGroup_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- cfg.UnitOfMeasure.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_UnitOfMeasure_TenantId')
    ALTER TABLE [cfg].[UnitOfMeasure] WITH CHECK
        ADD CONSTRAINT [FK_cfg_UnitOfMeasure_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.WithholdingTaxCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_WithholdingTaxCode_TenantId')
    ALTER TABLE [cfg].[WithholdingTaxCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_WithholdingTaxCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.WithholdingTaxCode.WithholdingTaxTypeId -> cfg.WithholdingTaxType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_WithholdingTaxCode_WithholdingTaxTypeId')
    ALTER TABLE [cfg].[WithholdingTaxCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_WithholdingTaxCode_WithholdingTaxTypeId] FOREIGN KEY ([WithholdingTaxTypeId])
        REFERENCES [cfg].[WithholdingTaxType] ([Id]);
GO

-- cfg.WithholdingTaxCode.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_WithholdingTaxCode_GLAccountId')
    ALTER TABLE [cfg].[WithholdingTaxCode] WITH CHECK
        ADD CONSTRAINT [FK_cfg_WithholdingTaxCode_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- cfg.WithholdingTaxType.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_WithholdingTaxType_TenantId')
    ALTER TABLE [cfg].[WithholdingTaxType] WITH CHECK
        ADD CONSTRAINT [FK_cfg_WithholdingTaxType_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- cfg.WithholdingTaxType.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_cfg_WithholdingTaxType_CountryCode')
    ALTER TABLE [cfg].[WithholdingTaxType] WITH CHECK
        ADD CONSTRAINT [FK_cfg_WithholdingTaxType_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- mdm.Address.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_Address_TenantId')
    ALTER TABLE [mdm].[Address] WITH CHECK
        ADD CONSTRAINT [FK_mdm_Address_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.Address.RegionCode -> cfg.Region (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_Address_RegionCode')
    ALTER TABLE [mdm].[Address] WITH CHECK
        ADD CONSTRAINT [FK_mdm_Address_RegionCode] FOREIGN KEY ([TenantId], [CountryCode], [RegionCode])
        REFERENCES [cfg].[Region] ([TenantId], [CountryCode], [RegionCode]);
GO

-- mdm.Address.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_Address_CountryCode')
    ALTER TABLE [mdm].[Address] WITH CHECK
        ADD CONSTRAINT [FK_mdm_Address_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- mdm.Address.LanguageCode -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_Address_LanguageCode')
    ALTER TABLE [mdm].[Address] WITH CHECK
        ADD CONSTRAINT [FK_mdm_Address_LanguageCode] FOREIGN KEY ([LanguageCode])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- mdm.Bank.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_Bank_TenantId')
    ALTER TABLE [mdm].[Bank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_Bank_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.Bank.BankCountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_Bank_BankCountryCode')
    ALTER TABLE [mdm].[Bank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_Bank_BankCountryCode] FOREIGN KEY ([TenantId], [BankCountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- mdm.BusinessPartner.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartner_TenantId')
    ALTER TABLE [mdm].[BusinessPartner] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartner_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartner.BusinessPartnerGroupId -> mdm.BusinessPartnerGroup (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartner_BusinessPartnerGroupId')
    ALTER TABLE [mdm].[BusinessPartner] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartner_BusinessPartnerGroupId] FOREIGN KEY ([BusinessPartnerGroupId])
        REFERENCES [mdm].[BusinessPartnerGroup] ([Id]);
GO

-- mdm.BusinessPartner.NationalityCountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartner_NationalityCountryCode')
    ALTER TABLE [mdm].[BusinessPartner] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartner_NationalityCountryCode] FOREIGN KEY ([TenantId], [NationalityCountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- mdm.BusinessPartner.LanguageCode -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartner_LanguageCode')
    ALTER TABLE [mdm].[BusinessPartner] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartner_LanguageCode] FOREIGN KEY ([LanguageCode])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- mdm.BusinessPartner.RegistrationCountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartner_RegistrationCountryCode')
    ALTER TABLE [mdm].[BusinessPartner] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartner_RegistrationCountryCode] FOREIGN KEY ([TenantId], [RegistrationCountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- mdm.BusinessPartner.AnnualRevenueCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartner_AnnualRevenueCurrencyCode')
    ALTER TABLE [mdm].[BusinessPartner] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartner_AnnualRevenueCurrencyCode] FOREIGN KEY ([TenantId], [AnnualRevenueCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- mdm.BusinessPartner.DefaultAddressId -> mdm.Address (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartner_DefaultAddressId')
    ALTER TABLE [mdm].[BusinessPartner] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartner_DefaultAddressId] FOREIGN KEY ([DefaultAddressId])
        REFERENCES [mdm].[Address] ([Id]);
GO

-- mdm.BusinessPartner.TradingPartnerCompany -> org.Company (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartner_TradingPartnerCompany')
    ALTER TABLE [mdm].[BusinessPartner] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartner_TradingPartnerCompany] FOREIGN KEY ([TenantId], [TradingPartnerCompany])
        REFERENCES [org].[Company] ([TenantId], [CompanyCodeGroup]);
GO

-- mdm.BusinessPartnerAddress.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerAddress_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerAddress] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerAddress_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerAddress.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerAddress_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerAddress] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerAddress_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerAddress.AddressId -> mdm.Address (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerAddress_AddressId')
    ALTER TABLE [mdm].[BusinessPartnerAddress] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerAddress_AddressId] FOREIGN KEY ([AddressId])
        REFERENCES [mdm].[Address] ([Id]);
GO

-- mdm.BusinessPartnerAttachment.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerAttachment_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerAttachment] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerAttachment_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerAttachment.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerAttachment_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerAttachment] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerAttachment_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerBank.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerBank_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerBank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerBank_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerBank.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerBank_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerBank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerBank_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerBank.BankCountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerBank_BankCountryCode')
    ALTER TABLE [mdm].[BusinessPartnerBank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerBank_BankCountryCode] FOREIGN KEY ([TenantId], [BankCountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- mdm.BusinessPartnerBank.BankKey -> mdm.Bank (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerBank_BankKey')
    ALTER TABLE [mdm].[BusinessPartnerBank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerBank_BankKey] FOREIGN KEY ([TenantId], [BankCountryCode], [BankKey])
        REFERENCES [mdm].[Bank] ([TenantId], [BankCountryCode], [BankKey]);
GO

-- mdm.BusinessPartnerBank.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerBank_CurrencyCode')
    ALTER TABLE [mdm].[BusinessPartnerBank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerBank_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- mdm.BusinessPartnerCommunication.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCommunication_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerCommunication] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCommunication_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerCommunication.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCommunication_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerCommunication] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCommunication_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerCommunication.AddressId -> mdm.Address (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCommunication_AddressId')
    ALTER TABLE [mdm].[BusinessPartnerCommunication] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCommunication_AddressId] FOREIGN KEY ([AddressId])
        REFERENCES [mdm].[Address] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_CompanyCodeId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.ReconciliationGLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_ReconciliationGLAccountId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_ReconciliationGLAccountId] FOREIGN KEY ([ReconciliationGLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.AlternativePayerPayeeId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_AlternativePayerPayeeId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_AlternativePayerPayeeId] FOREIGN KEY ([AlternativePayerPayeeId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.HeadOfficePartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_HeadOfficePartnerId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_HeadOfficePartnerId] FOREIGN KEY ([HeadOfficePartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.PaymentTermsId -> cfg.PaymentTerms (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_PaymentTermsId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_PaymentTermsId] FOREIGN KEY ([PaymentTermsId])
        REFERENCES [cfg].[PaymentTerms] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.HouseBankId -> mdm.HouseBank (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_HouseBankId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_HouseBankId] FOREIGN KEY ([HouseBankId])
        REFERENCES [mdm].[HouseBank] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.ToleranceGroupId -> cfg.ToleranceGroup (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_ToleranceGroupId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_ToleranceGroupId] FOREIGN KEY ([ToleranceGroupId])
        REFERENCES [cfg].[ToleranceGroup] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.DunningProcedureId -> cfg.DunningProcedure (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_DunningProcedureId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_DunningProcedureId] FOREIGN KEY ([DunningProcedureId])
        REFERENCES [cfg].[DunningProcedure] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.DunningRecipientPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_DunningRecipientPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_DunningRecipientPartnerId] FOREIGN KEY ([DunningRecipientPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.WithholdingTaxCodeId -> cfg.WithholdingTaxCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_WithholdingTaxCodeId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_WithholdingTaxCodeId] FOREIGN KEY ([WithholdingTaxCodeId])
        REFERENCES [cfg].[WithholdingTaxCode] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.ClearingPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_ClearingPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_ClearingPartnerId] FOREIGN KEY ([ClearingPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerCompanyCode.LocalCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCompanyCode_LocalCurrencyCode')
    ALTER TABLE [mdm].[BusinessPartnerCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCompanyCode_LocalCurrencyCode] FOREIGN KEY ([TenantId], [LocalCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- mdm.BusinessPartnerCreditProfile.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCreditProfile_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerCreditProfile] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCreditProfile_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerCreditProfile.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCreditProfile_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerCreditProfile] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCreditProfile_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerCreditProfile.CreditControlAreaId -> org.CreditControlArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCreditProfile_CreditControlAreaId')
    ALTER TABLE [mdm].[BusinessPartnerCreditProfile] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCreditProfile_CreditControlAreaId] FOREIGN KEY ([CreditControlAreaId])
        REFERENCES [org].[CreditControlArea] ([Id]);
GO

-- mdm.BusinessPartnerCreditProfile.CreditLimitCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCreditProfile_CreditLimitCurrencyCode')
    ALTER TABLE [mdm].[BusinessPartnerCreditProfile] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCreditProfile_CreditLimitCurrencyCode] FOREIGN KEY ([TenantId], [CreditLimitCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- mdm.BusinessPartnerCustomer.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCustomer_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerCustomer] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCustomer_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerCustomer.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCustomer_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerCustomer] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCustomer_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerCustomer.CustomerAccountGroupId -> cfg.AccountGroup (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerCustomer_CustomerAccountGroupId')
    ALTER TABLE [mdm].[BusinessPartnerCustomer] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerCustomer_CustomerAccountGroupId] FOREIGN KEY ([CustomerAccountGroupId])
        REFERENCES [cfg].[AccountGroup] ([Id]);
GO

-- mdm.BusinessPartnerGroup.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerGroup_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerGroup] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerGroup_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerGroup.NumberRangeObjectId -> cfg.NumberRangeObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerGroup_NumberRangeObjectId')
    ALTER TABLE [mdm].[BusinessPartnerGroup] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerGroup_NumberRangeObjectId] FOREIGN KEY ([NumberRangeObjectId])
        REFERENCES [cfg].[NumberRangeObject] ([Id]);
GO

-- mdm.BusinessPartnerIdentification.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerIdentification_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerIdentification] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerIdentification_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerIdentification.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerIdentification_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerIdentification] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerIdentification_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerIdentification.IssuingCountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerIdentification_IssuingCountryCode')
    ALTER TABLE [mdm].[BusinessPartnerIdentification] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerIdentification_IssuingCountryCode] FOREIGN KEY ([TenantId], [IssuingCountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- mdm.BusinessPartnerPurchasingOrganization.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerPurchasingOrganization_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerPurchasingOrganization] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerPurchasingOrganization_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerPurchasingOrganization.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerPurchasingOrganization_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerPurchasingOrganization] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerPurchasingOrganization_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerPurchasingOrganization.PurchasingOrganizationId -> org.PurchasingOrganization (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerPurchasingOrganization_PurchasingOrganizationId')
    ALTER TABLE [mdm].[BusinessPartnerPurchasingOrganization] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerPurchasingOrganization_PurchasingOrganizationId] FOREIGN KEY ([PurchasingOrganizationId])
        REFERENCES [org].[PurchasingOrganization] ([Id]);
GO

-- mdm.BusinessPartnerPurchasingOrganization.OrderCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerPurchasingOrganization_OrderCurrencyCode')
    ALTER TABLE [mdm].[BusinessPartnerPurchasingOrganization] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerPurchasingOrganization_OrderCurrencyCode] FOREIGN KEY ([TenantId], [OrderCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- mdm.BusinessPartnerPurchasingOrganization.PaymentTermsId -> cfg.PaymentTerms (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerPurchasingOrganization_PaymentTermsId')
    ALTER TABLE [mdm].[BusinessPartnerPurchasingOrganization] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerPurchasingOrganization_PaymentTermsId] FOREIGN KEY ([PaymentTermsId])
        REFERENCES [cfg].[PaymentTerms] ([Id]);
GO

-- mdm.BusinessPartnerRelationship.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerRelationship_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerRelationship] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerRelationship_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerRelationship.SourceBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerRelationship_SourceBusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerRelationship] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerRelationship_SourceBusinessPartnerId] FOREIGN KEY ([SourceBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerRelationship.TargetBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerRelationship_TargetBusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerRelationship] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerRelationship_TargetBusinessPartnerId] FOREIGN KEY ([TargetBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerRole.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerRole_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerRole] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerRole_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerRoleAssignment.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerRoleAssignment_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerRoleAssignment] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerRoleAssignment_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerRoleAssignment.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerRoleAssignment_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerRoleAssignment] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerRoleAssignment_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerRoleAssignment.BusinessPartnerRoleId -> mdm.BusinessPartnerRole (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerRoleAssignment_BusinessPartnerRoleId')
    ALTER TABLE [mdm].[BusinessPartnerRoleAssignment] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerRoleAssignment_BusinessPartnerRoleId] FOREIGN KEY ([BusinessPartnerRoleId])
        REFERENCES [mdm].[BusinessPartnerRole] ([Id]);
GO

-- mdm.BusinessPartnerSalesArea.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerSalesArea_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerSalesArea] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerSalesArea_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerSalesArea.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerSalesArea_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerSalesArea] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerSalesArea_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerSalesArea.SalesAreaId -> org.SalesArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerSalesArea_SalesAreaId')
    ALTER TABLE [mdm].[BusinessPartnerSalesArea] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerSalesArea_SalesAreaId] FOREIGN KEY ([SalesAreaId])
        REFERENCES [org].[SalesArea] ([Id]);
GO

-- mdm.BusinessPartnerSalesArea.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerSalesArea_CurrencyCode')
    ALTER TABLE [mdm].[BusinessPartnerSalesArea] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerSalesArea_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- mdm.BusinessPartnerSalesArea.PaymentTermsId -> cfg.PaymentTerms (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerSalesArea_PaymentTermsId')
    ALTER TABLE [mdm].[BusinessPartnerSalesArea] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerSalesArea_PaymentTermsId] FOREIGN KEY ([PaymentTermsId])
        REFERENCES [cfg].[PaymentTerms] ([Id]);
GO

-- mdm.BusinessPartnerSalesArea.DeliveringPlantId -> org.Plant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerSalesArea_DeliveringPlantId')
    ALTER TABLE [mdm].[BusinessPartnerSalesArea] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerSalesArea_DeliveringPlantId] FOREIGN KEY ([DeliveringPlantId])
        REFERENCES [org].[Plant] ([Id]);
GO

-- mdm.BusinessPartnerTaxNumber.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerTaxNumber_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerTaxNumber] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerTaxNumber_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerTaxNumber.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerTaxNumber_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerTaxNumber] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerTaxNumber_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerTaxNumber.CountryCode -> cfg.Country (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerTaxNumber_CountryCode')
    ALTER TABLE [mdm].[BusinessPartnerTaxNumber] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerTaxNumber_CountryCode] FOREIGN KEY ([TenantId], [CountryCode])
        REFERENCES [cfg].[Country] ([TenantId], [CountryCode]);
GO

-- mdm.BusinessPartnerVendor.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerVendor_TenantId')
    ALTER TABLE [mdm].[BusinessPartnerVendor] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerVendor_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.BusinessPartnerVendor.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerVendor_BusinessPartnerId')
    ALTER TABLE [mdm].[BusinessPartnerVendor] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerVendor_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- mdm.BusinessPartnerVendor.VendorAccountGroupId -> cfg.AccountGroup (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_BusinessPartnerVendor_VendorAccountGroupId')
    ALTER TABLE [mdm].[BusinessPartnerVendor] WITH CHECK
        ADD CONSTRAINT [FK_mdm_BusinessPartnerVendor_VendorAccountGroupId] FOREIGN KEY ([VendorAccountGroupId])
        REFERENCES [cfg].[AccountGroup] ([Id]);
GO

-- mdm.GLAccount.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccount_TenantId')
    ALTER TABLE [mdm].[GLAccount] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccount_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.GLAccount.ChartOfAccountsId -> cfg.ChartOfAccounts (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccount_ChartOfAccountsId')
    ALTER TABLE [mdm].[GLAccount] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccount_ChartOfAccountsId] FOREIGN KEY ([ChartOfAccountsId])
        REFERENCES [cfg].[ChartOfAccounts] ([Id]);
GO

-- mdm.GLAccount.AccountGroupId -> cfg.AccountGroup (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccount_AccountGroupId')
    ALTER TABLE [mdm].[GLAccount] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccount_AccountGroupId] FOREIGN KEY ([AccountGroupId])
        REFERENCES [cfg].[AccountGroup] ([Id]);
GO

-- mdm.GLAccount.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccount_FunctionalAreaId')
    ALTER TABLE [mdm].[GLAccount] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccount_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- mdm.GLAccountCompanyCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountCompanyCode_TenantId')
    ALTER TABLE [mdm].[GLAccountCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountCompanyCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.GLAccountCompanyCode.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountCompanyCode_GLAccountId')
    ALTER TABLE [mdm].[GLAccountCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountCompanyCode_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- mdm.GLAccountCompanyCode.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountCompanyCode_CompanyCodeId')
    ALTER TABLE [mdm].[GLAccountCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountCompanyCode_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- mdm.GLAccountCompanyCode.AccountCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountCompanyCode_AccountCurrencyCode')
    ALTER TABLE [mdm].[GLAccountCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountCompanyCode_AccountCurrencyCode] FOREIGN KEY ([TenantId], [AccountCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- mdm.GLAccountCompanyCode.FieldStatusGroupId -> cfg.FieldStatusGroup (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountCompanyCode_FieldStatusGroupId')
    ALTER TABLE [mdm].[GLAccountCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountCompanyCode_FieldStatusGroupId] FOREIGN KEY ([FieldStatusGroupId])
        REFERENCES [cfg].[FieldStatusGroup] ([Id]);
GO

-- mdm.GLAccountCompanyCode.HouseBankId -> mdm.HouseBank (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountCompanyCode_HouseBankId')
    ALTER TABLE [mdm].[GLAccountCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountCompanyCode_HouseBankId] FOREIGN KEY ([HouseBankId])
        REFERENCES [mdm].[HouseBank] ([Id]);
GO

-- mdm.GLAccountCompanyCode.HouseBankAccountId -> mdm.HouseBankAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountCompanyCode_HouseBankAccountId')
    ALTER TABLE [mdm].[GLAccountCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountCompanyCode_HouseBankAccountId] FOREIGN KEY ([HouseBankAccountId])
        REFERENCES [mdm].[HouseBankAccount] ([Id]);
GO

-- mdm.GLAccountText.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountText_TenantId')
    ALTER TABLE [mdm].[GLAccountText] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountText_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.GLAccountText.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountText_GLAccountId')
    ALTER TABLE [mdm].[GLAccountText] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountText_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- mdm.GLAccountText.LanguageCode -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_GLAccountText_LanguageCode')
    ALTER TABLE [mdm].[GLAccountText] WITH CHECK
        ADD CONSTRAINT [FK_mdm_GLAccountText_LanguageCode] FOREIGN KEY ([LanguageCode])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- mdm.HouseBank.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_HouseBank_TenantId')
    ALTER TABLE [mdm].[HouseBank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_HouseBank_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.HouseBank.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_HouseBank_CompanyCodeId')
    ALTER TABLE [mdm].[HouseBank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_HouseBank_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- mdm.HouseBank.BankId -> mdm.Bank (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_HouseBank_BankId')
    ALTER TABLE [mdm].[HouseBank] WITH CHECK
        ADD CONSTRAINT [FK_mdm_HouseBank_BankId] FOREIGN KEY ([BankId])
        REFERENCES [mdm].[Bank] ([Id]);
GO

-- mdm.HouseBankAccount.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_HouseBankAccount_TenantId')
    ALTER TABLE [mdm].[HouseBankAccount] WITH CHECK
        ADD CONSTRAINT [FK_mdm_HouseBankAccount_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- mdm.HouseBankAccount.HouseBankId -> mdm.HouseBank (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_HouseBankAccount_HouseBankId')
    ALTER TABLE [mdm].[HouseBankAccount] WITH CHECK
        ADD CONSTRAINT [FK_mdm_HouseBankAccount_HouseBankId] FOREIGN KEY ([HouseBankId])
        REFERENCES [mdm].[HouseBank] ([Id]);
GO

-- mdm.HouseBankAccount.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_HouseBankAccount_CurrencyCode')
    ALTER TABLE [mdm].[HouseBankAccount] WITH CHECK
        ADD CONSTRAINT [FK_mdm_HouseBankAccount_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- mdm.HouseBankAccount.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_HouseBankAccount_GLAccountId')
    ALTER TABLE [mdm].[HouseBankAccount] WITH CHECK
        ADD CONSTRAINT [FK_mdm_HouseBankAccount_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- mdm.HouseBankAccount.ClearingGLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_mdm_HouseBankAccount_ClearingGLAccountId')
    ALTER TABLE [mdm].[HouseBankAccount] WITH CHECK
        ADD CONSTRAINT [FK_mdm_HouseBankAccount_ClearingGLAccountId] FOREIGN KEY ([ClearingGLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.AccountBalance.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_TenantId')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.AccountBalance.LedgerId -> cfg.Ledger (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_LedgerId')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_LedgerId] FOREIGN KEY ([LedgerId])
        REFERENCES [cfg].[Ledger] ([Id]);
GO

-- fin.AccountBalance.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_CompanyCodeId')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.AccountBalance.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_GLAccountId')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.AccountBalance.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_BusinessPartnerId')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.AccountBalance.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_ProfitCenterId')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- fin.AccountBalance.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_SegmentId')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- fin.AccountBalance.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_FunctionalAreaId')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- fin.AccountBalance.BusinessAreaId -> org.BusinessArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_BusinessAreaId')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_BusinessAreaId] FOREIGN KEY ([BusinessAreaId])
        REFERENCES [org].[BusinessArea] ([Id]);
GO

-- fin.AccountBalance.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AccountBalance_CurrencyCode')
    ALTER TABLE [fin].[AccountBalance] WITH CHECK
        ADD CONSTRAINT [FK_fin_AccountBalance_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.Asset.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_TenantId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.Asset.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_CompanyCodeId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.Asset.AssetClassId -> fin.AssetClass (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_AssetClassId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_AssetClassId] FOREIGN KEY ([AssetClassId])
        REFERENCES [fin].[AssetClass] ([Id]);
GO

-- fin.Asset.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_UnitOfMeasure')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- fin.Asset.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_CostCenterId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- fin.Asset.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_ProfitCenterId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- fin.Asset.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_SegmentId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- fin.Asset.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_FunctionalAreaId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- fin.Asset.BusinessAreaId -> org.BusinessArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_BusinessAreaId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_BusinessAreaId] FOREIGN KEY ([BusinessAreaId])
        REFERENCES [org].[BusinessArea] ([Id]);
GO

-- fin.Asset.InternalOrderId -> co.InternalOrder (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_InternalOrderId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_InternalOrderId] FOREIGN KEY ([InternalOrderId])
        REFERENCES [co].[InternalOrder] ([Id]);
GO

-- fin.Asset.PlantId -> org.Plant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_PlantId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_PlantId] FOREIGN KEY ([PlantId])
        REFERENCES [org].[Plant] ([Id]);
GO

-- fin.Asset.LocationId -> org.Location (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_LocationId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_LocationId] FOREIGN KEY ([LocationId])
        REFERENCES [org].[Location] ([Id]);
GO

-- fin.Asset.ResponsiblePersonPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_ResponsiblePersonPartnerId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_ResponsiblePersonPartnerId] FOREIGN KEY ([ResponsiblePersonPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.Asset.VendorBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_Asset_VendorBusinessPartnerId')
    ALTER TABLE [fin].[Asset] WITH CHECK
        ADD CONSTRAINT [FK_fin_Asset_VendorBusinessPartnerId] FOREIGN KEY ([VendorBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.AssetClass.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetClass_TenantId')
    ALTER TABLE [fin].[AssetClass] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetClass_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.AssetClass.NumberRangeObjectId -> cfg.NumberRangeObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetClass_NumberRangeObjectId')
    ALTER TABLE [fin].[AssetClass] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetClass_NumberRangeObjectId] FOREIGN KEY ([NumberRangeObjectId])
        REFERENCES [cfg].[NumberRangeObject] ([Id]);
GO

-- fin.AssetClassDepreciationArea.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetClassDepreciationArea_TenantId')
    ALTER TABLE [fin].[AssetClassDepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetClassDepreciationArea_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.AssetClassDepreciationArea.AssetClassId -> fin.AssetClass (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetClassDepreciationArea_AssetClassId')
    ALTER TABLE [fin].[AssetClassDepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetClassDepreciationArea_AssetClassId] FOREIGN KEY ([AssetClassId])
        REFERENCES [fin].[AssetClass] ([Id]);
GO

-- fin.AssetClassDepreciationArea.DepreciationAreaId -> fin.DepreciationArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetClassDepreciationArea_DepreciationAreaId')
    ALTER TABLE [fin].[AssetClassDepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetClassDepreciationArea_DepreciationAreaId] FOREIGN KEY ([DepreciationAreaId])
        REFERENCES [fin].[DepreciationArea] ([Id]);
GO

-- fin.AssetClassDepreciationArea.DepreciationKeyId -> fin.DepreciationKey (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetClassDepreciationArea_DepreciationKeyId')
    ALTER TABLE [fin].[AssetClassDepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetClassDepreciationArea_DepreciationKeyId] FOREIGN KEY ([DepreciationKeyId])
        REFERENCES [fin].[DepreciationKey] ([Id]);
GO

-- fin.AssetDepreciationArea.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetDepreciationArea_TenantId')
    ALTER TABLE [fin].[AssetDepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetDepreciationArea_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.AssetDepreciationArea.AssetId -> fin.Asset (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetDepreciationArea_AssetId')
    ALTER TABLE [fin].[AssetDepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetDepreciationArea_AssetId] FOREIGN KEY ([AssetId])
        REFERENCES [fin].[Asset] ([Id]);
GO

-- fin.AssetDepreciationArea.DepreciationAreaId -> fin.DepreciationArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetDepreciationArea_DepreciationAreaId')
    ALTER TABLE [fin].[AssetDepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetDepreciationArea_DepreciationAreaId] FOREIGN KEY ([DepreciationAreaId])
        REFERENCES [fin].[DepreciationArea] ([Id]);
GO

-- fin.AssetDepreciationArea.DepreciationKeyId -> fin.DepreciationKey (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetDepreciationArea_DepreciationKeyId')
    ALTER TABLE [fin].[AssetDepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetDepreciationArea_DepreciationKeyId] FOREIGN KEY ([DepreciationKeyId])
        REFERENCES [fin].[DepreciationKey] ([Id]);
GO

-- fin.AssetTimeDependent.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTimeDependent_TenantId')
    ALTER TABLE [fin].[AssetTimeDependent] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTimeDependent_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.AssetTimeDependent.AssetId -> fin.Asset (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTimeDependent_AssetId')
    ALTER TABLE [fin].[AssetTimeDependent] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTimeDependent_AssetId] FOREIGN KEY ([AssetId])
        REFERENCES [fin].[Asset] ([Id]);
GO

-- fin.AssetTimeDependent.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTimeDependent_CostCenterId')
    ALTER TABLE [fin].[AssetTimeDependent] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTimeDependent_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- fin.AssetTimeDependent.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTimeDependent_ProfitCenterId')
    ALTER TABLE [fin].[AssetTimeDependent] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTimeDependent_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- fin.AssetTimeDependent.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTimeDependent_SegmentId')
    ALTER TABLE [fin].[AssetTimeDependent] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTimeDependent_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- fin.AssetTimeDependent.InternalOrderId -> co.InternalOrder (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTimeDependent_InternalOrderId')
    ALTER TABLE [fin].[AssetTimeDependent] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTimeDependent_InternalOrderId] FOREIGN KEY ([InternalOrderId])
        REFERENCES [co].[InternalOrder] ([Id]);
GO

-- fin.AssetTimeDependent.PlantId -> org.Plant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTimeDependent_PlantId')
    ALTER TABLE [fin].[AssetTimeDependent] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTimeDependent_PlantId] FOREIGN KEY ([PlantId])
        REFERENCES [org].[Plant] ([Id]);
GO

-- fin.AssetTimeDependent.LocationId -> org.Location (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTimeDependent_LocationId')
    ALTER TABLE [fin].[AssetTimeDependent] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTimeDependent_LocationId] FOREIGN KEY ([LocationId])
        REFERENCES [org].[Location] ([Id]);
GO

-- fin.AssetTransaction.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransaction_TenantId')
    ALTER TABLE [fin].[AssetTransaction] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransaction_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.AssetTransaction.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransaction_CompanyCodeId')
    ALTER TABLE [fin].[AssetTransaction] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransaction_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.AssetTransaction.AssetId -> fin.Asset (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransaction_AssetId')
    ALTER TABLE [fin].[AssetTransaction] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransaction_AssetId] FOREIGN KEY ([AssetId])
        REFERENCES [fin].[Asset] ([Id]);
GO

-- fin.AssetTransaction.DepreciationAreaId -> fin.DepreciationArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransaction_DepreciationAreaId')
    ALTER TABLE [fin].[AssetTransaction] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransaction_DepreciationAreaId] FOREIGN KEY ([DepreciationAreaId])
        REFERENCES [fin].[DepreciationArea] ([Id]);
GO

-- fin.AssetTransaction.TransactionTypeId -> fin.AssetTransactionType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransaction_TransactionTypeId')
    ALTER TABLE [fin].[AssetTransaction] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransaction_TransactionTypeId] FOREIGN KEY ([TransactionTypeId])
        REFERENCES [fin].[AssetTransactionType] ([Id]);
GO

-- fin.AssetTransaction.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransaction_CurrencyCode')
    ALTER TABLE [fin].[AssetTransaction] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransaction_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.AssetTransaction.PartnerBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransaction_PartnerBusinessPartnerId')
    ALTER TABLE [fin].[AssetTransaction] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransaction_PartnerBusinessPartnerId] FOREIGN KEY ([PartnerBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.AssetTransaction.TargetAssetId -> fin.Asset (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransaction_TargetAssetId')
    ALTER TABLE [fin].[AssetTransaction] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransaction_TargetAssetId] FOREIGN KEY ([TargetAssetId])
        REFERENCES [fin].[Asset] ([Id]);
GO

-- fin.AssetTransaction.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransaction_JournalEntryHeaderId')
    ALTER TABLE [fin].[AssetTransaction] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransaction_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.AssetTransactionType.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetTransactionType_TenantId')
    ALTER TABLE [fin].[AssetTransactionType] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetTransactionType_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.AssetValue.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetValue_TenantId')
    ALTER TABLE [fin].[AssetValue] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetValue_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.AssetValue.AssetId -> fin.Asset (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetValue_AssetId')
    ALTER TABLE [fin].[AssetValue] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetValue_AssetId] FOREIGN KEY ([AssetId])
        REFERENCES [fin].[Asset] ([Id]);
GO

-- fin.AssetValue.DepreciationAreaId -> fin.DepreciationArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetValue_DepreciationAreaId')
    ALTER TABLE [fin].[AssetValue] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetValue_DepreciationAreaId] FOREIGN KEY ([DepreciationAreaId])
        REFERENCES [fin].[DepreciationArea] ([Id]);
GO

-- fin.AssetValue.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_AssetValue_CurrencyCode')
    ALTER TABLE [fin].[AssetValue] WITH CHECK
        ADD CONSTRAINT [FK_fin_AssetValue_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.BalanceCarryForward.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_BalanceCarryForward_TenantId')
    ALTER TABLE [fin].[BalanceCarryForward] WITH CHECK
        ADD CONSTRAINT [FK_fin_BalanceCarryForward_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.BalanceCarryForward.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_BalanceCarryForward_CompanyCodeId')
    ALTER TABLE [fin].[BalanceCarryForward] WITH CHECK
        ADD CONSTRAINT [FK_fin_BalanceCarryForward_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.BalanceCarryForward.LedgerId -> cfg.Ledger (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_BalanceCarryForward_LedgerId')
    ALTER TABLE [fin].[BalanceCarryForward] WITH CHECK
        ADD CONSTRAINT [FK_fin_BalanceCarryForward_LedgerId] FOREIGN KEY ([LedgerId])
        REFERENCES [cfg].[Ledger] ([Id]);
GO

-- fin.BalanceCarryForward.RetainedEarningsGLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_BalanceCarryForward_RetainedEarningsGLAccountId')
    ALTER TABLE [fin].[BalanceCarryForward] WITH CHECK
        ADD CONSTRAINT [FK_fin_BalanceCarryForward_RetainedEarningsGLAccountId] FOREIGN KEY ([RetainedEarningsGLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.BalanceCarryForward.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_BalanceCarryForward_JournalEntryHeaderId')
    ALTER TABLE [fin].[BalanceCarryForward] WITH CHECK
        ADD CONSTRAINT [FK_fin_BalanceCarryForward_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.ClearingDocument.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ClearingDocument_TenantId')
    ALTER TABLE [fin].[ClearingDocument] WITH CHECK
        ADD CONSTRAINT [FK_fin_ClearingDocument_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.ClearingDocument.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ClearingDocument_CompanyCodeId')
    ALTER TABLE [fin].[ClearingDocument] WITH CHECK
        ADD CONSTRAINT [FK_fin_ClearingDocument_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.ClearingDocument.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ClearingDocument_JournalEntryHeaderId')
    ALTER TABLE [fin].[ClearingDocument] WITH CHECK
        ADD CONSTRAINT [FK_fin_ClearingDocument_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.ClearingDocument.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ClearingDocument_CurrencyCode')
    ALTER TABLE [fin].[ClearingDocument] WITH CHECK
        ADD CONSTRAINT [FK_fin_ClearingDocument_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.ClearingItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ClearingItem_TenantId')
    ALTER TABLE [fin].[ClearingItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_ClearingItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.ClearingItem.ClearingDocumentId -> fin.ClearingDocument (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ClearingItem_ClearingDocumentId')
    ALTER TABLE [fin].[ClearingItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_ClearingItem_ClearingDocumentId] FOREIGN KEY ([ClearingDocumentId])
        REFERENCES [fin].[ClearingDocument] ([Id]);
GO

-- fin.ClearingItem.OpenItemId -> fin.OpenItem (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ClearingItem_OpenItemId')
    ALTER TABLE [fin].[ClearingItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_ClearingItem_OpenItemId] FOREIGN KEY ([OpenItemId])
        REFERENCES [fin].[OpenItem] ([Id]);
GO

-- fin.ClearingItem.ResidualOpenItemId -> fin.OpenItem (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ClearingItem_ResidualOpenItemId')
    ALTER TABLE [fin].[ClearingItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_ClearingItem_ResidualOpenItemId] FOREIGN KEY ([ResidualOpenItemId])
        REFERENCES [fin].[OpenItem] ([Id]);
GO

-- fin.CustomerInvoice.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_TenantId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.CustomerInvoice.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_CompanyCodeId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.CustomerInvoice.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_BusinessPartnerId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.CustomerInvoice.PayerBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_PayerBusinessPartnerId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_PayerBusinessPartnerId] FOREIGN KEY ([PayerBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.CustomerInvoice.BillToBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_BillToBusinessPartnerId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_BillToBusinessPartnerId] FOREIGN KEY ([BillToBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.CustomerInvoice.ShipToBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_ShipToBusinessPartnerId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_ShipToBusinessPartnerId] FOREIGN KEY ([ShipToBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.CustomerInvoice.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_CurrencyCode')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.CustomerInvoice.PaymentTermsId -> cfg.PaymentTerms (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_PaymentTermsId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_PaymentTermsId] FOREIGN KEY ([PaymentTermsId])
        REFERENCES [cfg].[PaymentTerms] ([Id]);
GO

-- fin.CustomerInvoice.SalesAreaId -> org.SalesArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_SalesAreaId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_SalesAreaId] FOREIGN KEY ([SalesAreaId])
        REFERENCES [org].[SalesArea] ([Id]);
GO

-- fin.CustomerInvoice.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_JournalEntryHeaderId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.CustomerInvoice.WorkflowInstanceId -> wf.WorkflowInstance (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoice_WorkflowInstanceId')
    ALTER TABLE [fin].[CustomerInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoice_WorkflowInstanceId] FOREIGN KEY ([WorkflowInstanceId])
        REFERENCES [wf].[WorkflowInstance] ([Id]);
GO

-- fin.CustomerInvoiceItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_TenantId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.CustomerInvoiceItem.CustomerInvoiceId -> fin.CustomerInvoice (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_CustomerInvoiceId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_CustomerInvoiceId] FOREIGN KEY ([CustomerInvoiceId])
        REFERENCES [fin].[CustomerInvoice] ([Id]);
GO

-- fin.CustomerInvoiceItem.RevenueGLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_RevenueGLAccountId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_RevenueGLAccountId] FOREIGN KEY ([RevenueGLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.CustomerInvoiceItem.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_UnitOfMeasure')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- fin.CustomerInvoiceItem.TaxCodeId -> cfg.TaxCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_TaxCodeId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_TaxCodeId] FOREIGN KEY ([TaxCodeId])
        REFERENCES [cfg].[TaxCode] ([Id]);
GO

-- fin.CustomerInvoiceItem.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_CostCenterId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- fin.CustomerInvoiceItem.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_ProfitCenterId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- fin.CustomerInvoiceItem.InternalOrderId -> co.InternalOrder (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_InternalOrderId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_InternalOrderId] FOREIGN KEY ([InternalOrderId])
        REFERENCES [co].[InternalOrder] ([Id]);
GO

-- fin.CustomerInvoiceItem.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_SegmentId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- fin.CustomerInvoiceItem.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_FunctionalAreaId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- fin.CustomerInvoiceItem.BusinessAreaId -> org.BusinessArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_CustomerInvoiceItem_BusinessAreaId')
    ALTER TABLE [fin].[CustomerInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_CustomerInvoiceItem_BusinessAreaId] FOREIGN KEY ([BusinessAreaId])
        REFERENCES [org].[BusinessArea] ([Id]);
GO

-- fin.DepreciationArea.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationArea_TenantId')
    ALTER TABLE [fin].[DepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationArea_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.DepreciationArea.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationArea_CompanyCodeId')
    ALTER TABLE [fin].[DepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationArea_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.DepreciationArea.LedgerId -> cfg.Ledger (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationArea_LedgerId')
    ALTER TABLE [fin].[DepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationArea_LedgerId] FOREIGN KEY ([LedgerId])
        REFERENCES [cfg].[Ledger] ([Id]);
GO

-- fin.DepreciationArea.AccountingPrincipleId -> cfg.AccountingPrinciple (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationArea_AccountingPrincipleId')
    ALTER TABLE [fin].[DepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationArea_AccountingPrincipleId] FOREIGN KEY ([AccountingPrincipleId])
        REFERENCES [cfg].[AccountingPrinciple] ([Id]);
GO

-- fin.DepreciationArea.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationArea_CurrencyCode')
    ALTER TABLE [fin].[DepreciationArea] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationArea_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.DepreciationKey.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationKey_TenantId')
    ALTER TABLE [fin].[DepreciationKey] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationKey_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.DepreciationPosting.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationPosting_TenantId')
    ALTER TABLE [fin].[DepreciationPosting] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationPosting_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.DepreciationPosting.DepreciationRunId -> fin.DepreciationRun (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationPosting_DepreciationRunId')
    ALTER TABLE [fin].[DepreciationPosting] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationPosting_DepreciationRunId] FOREIGN KEY ([DepreciationRunId])
        REFERENCES [fin].[DepreciationRun] ([Id]);
GO

-- fin.DepreciationPosting.AssetId -> fin.Asset (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationPosting_AssetId')
    ALTER TABLE [fin].[DepreciationPosting] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationPosting_AssetId] FOREIGN KEY ([AssetId])
        REFERENCES [fin].[Asset] ([Id]);
GO

-- fin.DepreciationPosting.DepreciationAreaId -> fin.DepreciationArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationPosting_DepreciationAreaId')
    ALTER TABLE [fin].[DepreciationPosting] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationPosting_DepreciationAreaId] FOREIGN KEY ([DepreciationAreaId])
        REFERENCES [fin].[DepreciationArea] ([Id]);
GO

-- fin.DepreciationPosting.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationPosting_CurrencyCode')
    ALTER TABLE [fin].[DepreciationPosting] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationPosting_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.DepreciationPosting.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationPosting_CostCenterId')
    ALTER TABLE [fin].[DepreciationPosting] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationPosting_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- fin.DepreciationPosting.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationPosting_ProfitCenterId')
    ALTER TABLE [fin].[DepreciationPosting] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationPosting_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- fin.DepreciationPosting.InternalOrderId -> co.InternalOrder (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationPosting_InternalOrderId')
    ALTER TABLE [fin].[DepreciationPosting] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationPosting_InternalOrderId] FOREIGN KEY ([InternalOrderId])
        REFERENCES [co].[InternalOrder] ([Id]);
GO

-- fin.DepreciationPosting.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationPosting_JournalEntryHeaderId')
    ALTER TABLE [fin].[DepreciationPosting] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationPosting_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.DepreciationRun.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationRun_TenantId')
    ALTER TABLE [fin].[DepreciationRun] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationRun_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.DepreciationRun.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DepreciationRun_CompanyCodeId')
    ALTER TABLE [fin].[DepreciationRun] WITH CHECK
        ADD CONSTRAINT [FK_fin_DepreciationRun_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.DocumentAttachment.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DocumentAttachment_TenantId')
    ALTER TABLE [fin].[DocumentAttachment] WITH CHECK
        ADD CONSTRAINT [FK_fin_DocumentAttachment_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.DunningNotice.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningNotice_TenantId')
    ALTER TABLE [fin].[DunningNotice] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningNotice_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.DunningNotice.DunningRunId -> fin.DunningRun (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningNotice_DunningRunId')
    ALTER TABLE [fin].[DunningNotice] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningNotice_DunningRunId] FOREIGN KEY ([DunningRunId])
        REFERENCES [fin].[DunningRun] ([Id]);
GO

-- fin.DunningNotice.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningNotice_BusinessPartnerId')
    ALTER TABLE [fin].[DunningNotice] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningNotice_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.DunningNotice.DunningProcedureId -> cfg.DunningProcedure (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningNotice_DunningProcedureId')
    ALTER TABLE [fin].[DunningNotice] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningNotice_DunningProcedureId] FOREIGN KEY ([DunningProcedureId])
        REFERENCES [cfg].[DunningProcedure] ([Id]);
GO

-- fin.DunningNotice.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningNotice_CurrencyCode')
    ALTER TABLE [fin].[DunningNotice] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningNotice_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.DunningNoticeItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningNoticeItem_TenantId')
    ALTER TABLE [fin].[DunningNoticeItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningNoticeItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.DunningNoticeItem.DunningNoticeId -> fin.DunningNotice (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningNoticeItem_DunningNoticeId')
    ALTER TABLE [fin].[DunningNoticeItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningNoticeItem_DunningNoticeId] FOREIGN KEY ([DunningNoticeId])
        REFERENCES [fin].[DunningNotice] ([Id]);
GO

-- fin.DunningNoticeItem.OpenItemId -> fin.OpenItem (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningNoticeItem_OpenItemId')
    ALTER TABLE [fin].[DunningNoticeItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningNoticeItem_OpenItemId] FOREIGN KEY ([OpenItemId])
        REFERENCES [fin].[OpenItem] ([Id]);
GO

-- fin.DunningRun.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningRun_TenantId')
    ALTER TABLE [fin].[DunningRun] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningRun_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.DunningRun.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_DunningRun_CompanyCodeId')
    ALTER TABLE [fin].[DunningRun] WITH CHECK
        ADD CONSTRAINT [FK_fin_DunningRun_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.ForeignCurrencyValuation.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ForeignCurrencyValuation_TenantId')
    ALTER TABLE [fin].[ForeignCurrencyValuation] WITH CHECK
        ADD CONSTRAINT [FK_fin_ForeignCurrencyValuation_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.ForeignCurrencyValuation.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ForeignCurrencyValuation_CompanyCodeId')
    ALTER TABLE [fin].[ForeignCurrencyValuation] WITH CHECK
        ADD CONSTRAINT [FK_fin_ForeignCurrencyValuation_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.ForeignCurrencyValuation.OpenItemId -> fin.OpenItem (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ForeignCurrencyValuation_OpenItemId')
    ALTER TABLE [fin].[ForeignCurrencyValuation] WITH CHECK
        ADD CONSTRAINT [FK_fin_ForeignCurrencyValuation_OpenItemId] FOREIGN KEY ([OpenItemId])
        REFERENCES [fin].[OpenItem] ([Id]);
GO

-- fin.ForeignCurrencyValuation.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ForeignCurrencyValuation_GLAccountId')
    ALTER TABLE [fin].[ForeignCurrencyValuation] WITH CHECK
        ADD CONSTRAINT [FK_fin_ForeignCurrencyValuation_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.ForeignCurrencyValuation.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ForeignCurrencyValuation_JournalEntryHeaderId')
    ALTER TABLE [fin].[ForeignCurrencyValuation] WITH CHECK
        ADD CONSTRAINT [FK_fin_ForeignCurrencyValuation_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.ForeignCurrencyValuation.ReversalJournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_ForeignCurrencyValuation_ReversalJournalEntryHeaderId')
    ALTER TABLE [fin].[ForeignCurrencyValuation] WITH CHECK
        ADD CONSTRAINT [FK_fin_ForeignCurrencyValuation_ReversalJournalEntryHeaderId] FOREIGN KEY ([ReversalJournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.JournalEntryHeader.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryHeader_TenantId')
    ALTER TABLE [fin].[JournalEntryHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryHeader_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.JournalEntryHeader.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryHeader_CompanyCodeId')
    ALTER TABLE [fin].[JournalEntryHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryHeader_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.JournalEntryHeader.LedgerId -> cfg.Ledger (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryHeader_LedgerId')
    ALTER TABLE [fin].[JournalEntryHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryHeader_LedgerId] FOREIGN KEY ([LedgerId])
        REFERENCES [cfg].[Ledger] ([Id]);
GO

-- fin.JournalEntryHeader.DocumentTypeId -> cfg.DocumentType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryHeader_DocumentTypeId')
    ALTER TABLE [fin].[JournalEntryHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryHeader_DocumentTypeId] FOREIGN KEY ([DocumentTypeId])
        REFERENCES [cfg].[DocumentType] ([Id]);
GO

-- fin.JournalEntryHeader.DocumentCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryHeader_DocumentCurrencyCode')
    ALTER TABLE [fin].[JournalEntryHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryHeader_DocumentCurrencyCode] FOREIGN KEY ([TenantId], [DocumentCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.JournalEntryHeader.LocalCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryHeader_LocalCurrencyCode')
    ALTER TABLE [fin].[JournalEntryHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryHeader_LocalCurrencyCode] FOREIGN KEY ([TenantId], [LocalCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.JournalEntryHeader.GroupCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryHeader_GroupCurrencyCode')
    ALTER TABLE [fin].[JournalEntryHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryHeader_GroupCurrencyCode] FOREIGN KEY ([TenantId], [GroupCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.JournalEntryHeader.WorkflowInstanceId -> wf.WorkflowInstance (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryHeader_WorkflowInstanceId')
    ALTER TABLE [fin].[JournalEntryHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryHeader_WorkflowInstanceId] FOREIGN KEY ([WorkflowInstanceId])
        REFERENCES [wf].[WorkflowInstance] ([Id]);
GO

-- fin.JournalEntryLine.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_TenantId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.JournalEntryLine.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_JournalEntryHeaderId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.JournalEntryLine.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_CompanyCodeId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.JournalEntryLine.LedgerId -> cfg.Ledger (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_LedgerId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_LedgerId] FOREIGN KEY ([LedgerId])
        REFERENCES [cfg].[Ledger] ([Id]);
GO

-- fin.JournalEntryLine.PostingKey -> cfg.PostingKey (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_PostingKey')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_PostingKey] FOREIGN KEY ([TenantId], [PostingKey])
        REFERENCES [cfg].[PostingKey] ([TenantId], [PostingKey]);
GO

-- fin.JournalEntryLine.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_GLAccountId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.JournalEntryLine.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_BusinessPartnerId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.JournalEntryLine.AssetId -> fin.Asset (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_AssetId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_AssetId] FOREIGN KEY ([AssetId])
        REFERENCES [fin].[Asset] ([Id]);
GO

-- fin.JournalEntryLine.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_CostCenterId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- fin.JournalEntryLine.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_ProfitCenterId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- fin.JournalEntryLine.PartnerProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_PartnerProfitCenterId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_PartnerProfitCenterId] FOREIGN KEY ([PartnerProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- fin.JournalEntryLine.InternalOrderId -> co.InternalOrder (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_InternalOrderId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_InternalOrderId] FOREIGN KEY ([InternalOrderId])
        REFERENCES [co].[InternalOrder] ([Id]);
GO

-- fin.JournalEntryLine.CostElementId -> co.CostElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_CostElementId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_CostElementId] FOREIGN KEY ([CostElementId])
        REFERENCES [co].[CostElement] ([Id]);
GO

-- fin.JournalEntryLine.ActivityTypeId -> co.ActivityType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_ActivityTypeId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_ActivityTypeId] FOREIGN KEY ([ActivityTypeId])
        REFERENCES [co].[ActivityType] ([Id]);
GO

-- fin.JournalEntryLine.BusinessAreaId -> org.BusinessArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_BusinessAreaId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_BusinessAreaId] FOREIGN KEY ([BusinessAreaId])
        REFERENCES [org].[BusinessArea] ([Id]);
GO

-- fin.JournalEntryLine.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_FunctionalAreaId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- fin.JournalEntryLine.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_SegmentId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- fin.JournalEntryLine.PartnerSegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_PartnerSegmentId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_PartnerSegmentId] FOREIGN KEY ([PartnerSegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- fin.JournalEntryLine.PartnerCompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_PartnerCompanyCodeId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_PartnerCompanyCodeId] FOREIGN KEY ([PartnerCompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.JournalEntryLine.PlantId -> org.Plant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_PlantId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_PlantId] FOREIGN KEY ([PlantId])
        REFERENCES [org].[Plant] ([Id]);
GO

-- fin.JournalEntryLine.BranchId -> org.Branch (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_BranchId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_BranchId] FOREIGN KEY ([BranchId])
        REFERENCES [org].[Branch] ([Id]);
GO

-- fin.JournalEntryLine.DocumentCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_DocumentCurrencyCode')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_DocumentCurrencyCode] FOREIGN KEY ([TenantId], [DocumentCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.JournalEntryLine.LocalCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_LocalCurrencyCode')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_LocalCurrencyCode] FOREIGN KEY ([TenantId], [LocalCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.JournalEntryLine.GroupCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_GroupCurrencyCode')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_GroupCurrencyCode] FOREIGN KEY ([TenantId], [GroupCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.JournalEntryLine.ControllingAreaCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_ControllingAreaCurrencyCode')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_ControllingAreaCurrencyCode] FOREIGN KEY ([TenantId], [ControllingAreaCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.JournalEntryLine.HardCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_HardCurrencyCode')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_HardCurrencyCode] FOREIGN KEY ([TenantId], [HardCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.JournalEntryLine.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_UnitOfMeasure')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- fin.JournalEntryLine.TaxCodeId -> cfg.TaxCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_TaxCodeId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_TaxCodeId] FOREIGN KEY ([TaxCodeId])
        REFERENCES [cfg].[TaxCode] ([Id]);
GO

-- fin.JournalEntryLine.WithholdingTaxCodeId -> cfg.WithholdingTaxCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_WithholdingTaxCodeId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_WithholdingTaxCodeId] FOREIGN KEY ([WithholdingTaxCodeId])
        REFERENCES [cfg].[WithholdingTaxCode] ([Id]);
GO

-- fin.JournalEntryLine.PaymentTermsId -> cfg.PaymentTerms (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_PaymentTermsId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_PaymentTermsId] FOREIGN KEY ([PaymentTermsId])
        REFERENCES [cfg].[PaymentTerms] ([Id]);
GO

-- fin.JournalEntryLine.HouseBankId -> mdm.HouseBank (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryLine_HouseBankId')
    ALTER TABLE [fin].[JournalEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryLine_HouseBankId] FOREIGN KEY ([HouseBankId])
        REFERENCES [mdm].[HouseBank] ([Id]);
GO

-- fin.JournalEntryTax.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryTax_TenantId')
    ALTER TABLE [fin].[JournalEntryTax] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryTax_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.JournalEntryTax.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryTax_CompanyCodeId')
    ALTER TABLE [fin].[JournalEntryTax] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryTax_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.JournalEntryTax.TaxCodeId -> cfg.TaxCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryTax_TaxCodeId')
    ALTER TABLE [fin].[JournalEntryTax] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryTax_TaxCodeId] FOREIGN KEY ([TaxCodeId])
        REFERENCES [cfg].[TaxCode] ([Id]);
GO

-- fin.JournalEntryTax.TaxGLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_JournalEntryTax_TaxGLAccountId')
    ALTER TABLE [fin].[JournalEntryTax] WITH CHECK
        ADD CONSTRAINT [FK_fin_JournalEntryTax_TaxGLAccountId] FOREIGN KEY ([TaxGLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.OpenItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_OpenItem_TenantId')
    ALTER TABLE [fin].[OpenItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_OpenItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.OpenItem.JournalEntryLineId -> fin.JournalEntryLine (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_OpenItem_JournalEntryLineId')
    ALTER TABLE [fin].[OpenItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_OpenItem_JournalEntryLineId] FOREIGN KEY ([JournalEntryLineId])
        REFERENCES [fin].[JournalEntryLine] ([Id]);
GO

-- fin.OpenItem.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_OpenItem_CompanyCodeId')
    ALTER TABLE [fin].[OpenItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_OpenItem_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.OpenItem.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_OpenItem_BusinessPartnerId')
    ALTER TABLE [fin].[OpenItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_OpenItem_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.OpenItem.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_OpenItem_GLAccountId')
    ALTER TABLE [fin].[OpenItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_OpenItem_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.OpenItem.DocumentCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_OpenItem_DocumentCurrencyCode')
    ALTER TABLE [fin].[OpenItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_OpenItem_DocumentCurrencyCode] FOREIGN KEY ([TenantId], [DocumentCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.OpenItem.PaymentTermsId -> cfg.PaymentTerms (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_OpenItem_PaymentTermsId')
    ALTER TABLE [fin].[OpenItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_OpenItem_PaymentTermsId] FOREIGN KEY ([PaymentTermsId])
        REFERENCES [cfg].[PaymentTerms] ([Id]);
GO

-- fin.PaymentAllocation.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentAllocation_TenantId')
    ALTER TABLE [fin].[PaymentAllocation] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentAllocation_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.PaymentAllocation.PaymentHeaderId -> fin.PaymentHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentAllocation_PaymentHeaderId')
    ALTER TABLE [fin].[PaymentAllocation] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentAllocation_PaymentHeaderId] FOREIGN KEY ([PaymentHeaderId])
        REFERENCES [fin].[PaymentHeader] ([Id]);
GO

-- fin.PaymentAllocation.OpenItemId -> fin.OpenItem (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentAllocation_OpenItemId')
    ALTER TABLE [fin].[PaymentAllocation] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentAllocation_OpenItemId] FOREIGN KEY ([OpenItemId])
        REFERENCES [fin].[OpenItem] ([Id]);
GO

-- fin.PaymentAllocation.ResidualOpenItemId -> fin.OpenItem (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentAllocation_ResidualOpenItemId')
    ALTER TABLE [fin].[PaymentAllocation] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentAllocation_ResidualOpenItemId] FOREIGN KEY ([ResidualOpenItemId])
        REFERENCES [fin].[OpenItem] ([Id]);
GO

-- fin.PaymentHeader.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_TenantId')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.PaymentHeader.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_CompanyCodeId')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.PaymentHeader.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_BusinessPartnerId')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.PaymentHeader.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_CurrencyCode')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.PaymentHeader.HouseBankId -> mdm.HouseBank (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_HouseBankId')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_HouseBankId] FOREIGN KEY ([HouseBankId])
        REFERENCES [mdm].[HouseBank] ([Id]);
GO

-- fin.PaymentHeader.HouseBankAccountId -> mdm.HouseBankAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_HouseBankAccountId')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_HouseBankAccountId] FOREIGN KEY ([HouseBankAccountId])
        REFERENCES [mdm].[HouseBankAccount] ([Id]);
GO

-- fin.PaymentHeader.PaymentRunId -> fin.PaymentRun (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_PaymentRunId')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_PaymentRunId] FOREIGN KEY ([PaymentRunId])
        REFERENCES [fin].[PaymentRun] ([Id]);
GO

-- fin.PaymentHeader.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_JournalEntryHeaderId')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.PaymentHeader.ClearingDocumentId -> fin.ClearingDocument (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_ClearingDocumentId')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_ClearingDocumentId] FOREIGN KEY ([ClearingDocumentId])
        REFERENCES [fin].[ClearingDocument] ([Id]);
GO

-- fin.PaymentHeader.WorkflowInstanceId -> wf.WorkflowInstance (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentHeader_WorkflowInstanceId')
    ALTER TABLE [fin].[PaymentHeader] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentHeader_WorkflowInstanceId] FOREIGN KEY ([WorkflowInstanceId])
        REFERENCES [wf].[WorkflowInstance] ([Id]);
GO

-- fin.PaymentProposalItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentProposalItem_TenantId')
    ALTER TABLE [fin].[PaymentProposalItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentProposalItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.PaymentProposalItem.PaymentRunId -> fin.PaymentRun (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentProposalItem_PaymentRunId')
    ALTER TABLE [fin].[PaymentProposalItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentProposalItem_PaymentRunId] FOREIGN KEY ([PaymentRunId])
        REFERENCES [fin].[PaymentRun] ([Id]);
GO

-- fin.PaymentProposalItem.OpenItemId -> fin.OpenItem (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentProposalItem_OpenItemId')
    ALTER TABLE [fin].[PaymentProposalItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentProposalItem_OpenItemId] FOREIGN KEY ([OpenItemId])
        REFERENCES [fin].[OpenItem] ([Id]);
GO

-- fin.PaymentProposalItem.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentProposalItem_BusinessPartnerId')
    ALTER TABLE [fin].[PaymentProposalItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentProposalItem_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.PaymentProposalItem.HouseBankId -> mdm.HouseBank (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentProposalItem_HouseBankId')
    ALTER TABLE [fin].[PaymentProposalItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentProposalItem_HouseBankId] FOREIGN KEY ([HouseBankId])
        REFERENCES [mdm].[HouseBank] ([Id]);
GO

-- fin.PaymentProposalItem.PaymentHeaderId -> fin.PaymentHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentProposalItem_PaymentHeaderId')
    ALTER TABLE [fin].[PaymentProposalItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentProposalItem_PaymentHeaderId] FOREIGN KEY ([PaymentHeaderId])
        REFERENCES [fin].[PaymentHeader] ([Id]);
GO

-- fin.PaymentRun.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentRun_TenantId')
    ALTER TABLE [fin].[PaymentRun] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentRun_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.PaymentRun.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentRun_CompanyCodeId')
    ALTER TABLE [fin].[PaymentRun] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentRun_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.PaymentRun.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_PaymentRun_CurrencyCode')
    ALTER TABLE [fin].[PaymentRun] WITH CHECK
        ADD CONSTRAINT [FK_fin_PaymentRun_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.RecurringEntry.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntry_TenantId')
    ALTER TABLE [fin].[RecurringEntry] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntry_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.RecurringEntry.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntry_CompanyCodeId')
    ALTER TABLE [fin].[RecurringEntry] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntry_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.RecurringEntry.DocumentTypeId -> cfg.DocumentType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntry_DocumentTypeId')
    ALTER TABLE [fin].[RecurringEntry] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntry_DocumentTypeId] FOREIGN KEY ([DocumentTypeId])
        REFERENCES [cfg].[DocumentType] ([Id]);
GO

-- fin.RecurringEntry.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntry_CurrencyCode')
    ALTER TABLE [fin].[RecurringEntry] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntry_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.RecurringEntryExecution.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryExecution_TenantId')
    ALTER TABLE [fin].[RecurringEntryExecution] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryExecution_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.RecurringEntryExecution.RecurringEntryId -> fin.RecurringEntry (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryExecution_RecurringEntryId')
    ALTER TABLE [fin].[RecurringEntryExecution] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryExecution_RecurringEntryId] FOREIGN KEY ([RecurringEntryId])
        REFERENCES [fin].[RecurringEntry] ([Id]);
GO

-- fin.RecurringEntryExecution.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryExecution_JournalEntryHeaderId')
    ALTER TABLE [fin].[RecurringEntryExecution] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryExecution_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.RecurringEntryLine.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_TenantId')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.RecurringEntryLine.RecurringEntryId -> fin.RecurringEntry (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_RecurringEntryId')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_RecurringEntryId] FOREIGN KEY ([RecurringEntryId])
        REFERENCES [fin].[RecurringEntry] ([Id]);
GO

-- fin.RecurringEntryLine.PostingKey -> cfg.PostingKey (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_PostingKey')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_PostingKey] FOREIGN KEY ([TenantId], [PostingKey])
        REFERENCES [cfg].[PostingKey] ([TenantId], [PostingKey]);
GO

-- fin.RecurringEntryLine.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_GLAccountId')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.RecurringEntryLine.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_BusinessPartnerId')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.RecurringEntryLine.TaxCodeId -> cfg.TaxCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_TaxCodeId')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_TaxCodeId] FOREIGN KEY ([TaxCodeId])
        REFERENCES [cfg].[TaxCode] ([Id]);
GO

-- fin.RecurringEntryLine.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_CostCenterId')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- fin.RecurringEntryLine.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_ProfitCenterId')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- fin.RecurringEntryLine.InternalOrderId -> co.InternalOrder (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_InternalOrderId')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_InternalOrderId] FOREIGN KEY ([InternalOrderId])
        REFERENCES [co].[InternalOrder] ([Id]);
GO

-- fin.RecurringEntryLine.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_RecurringEntryLine_SegmentId')
    ALTER TABLE [fin].[RecurringEntryLine] WITH CHECK
        ADD CONSTRAINT [FK_fin_RecurringEntryLine_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- fin.VendorInvoice.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_TenantId')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.VendorInvoice.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_CompanyCodeId')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- fin.VendorInvoice.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_BusinessPartnerId')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.VendorInvoice.PayeeBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_PayeeBusinessPartnerId')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_PayeeBusinessPartnerId] FOREIGN KEY ([PayeeBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- fin.VendorInvoice.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_CurrencyCode')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- fin.VendorInvoice.PaymentTermsId -> cfg.PaymentTerms (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_PaymentTermsId')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_PaymentTermsId] FOREIGN KEY ([PaymentTermsId])
        REFERENCES [cfg].[PaymentTerms] ([Id]);
GO

-- fin.VendorInvoice.HouseBankId -> mdm.HouseBank (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_HouseBankId')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_HouseBankId] FOREIGN KEY ([HouseBankId])
        REFERENCES [mdm].[HouseBank] ([Id]);
GO

-- fin.VendorInvoice.PurchasingOrganizationId -> org.PurchasingOrganization (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_PurchasingOrganizationId')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_PurchasingOrganizationId] FOREIGN KEY ([PurchasingOrganizationId])
        REFERENCES [org].[PurchasingOrganization] ([Id]);
GO

-- fin.VendorInvoice.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_JournalEntryHeaderId')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- fin.VendorInvoice.WorkflowInstanceId -> wf.WorkflowInstance (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoice_WorkflowInstanceId')
    ALTER TABLE [fin].[VendorInvoice] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoice_WorkflowInstanceId] FOREIGN KEY ([WorkflowInstanceId])
        REFERENCES [wf].[WorkflowInstance] ([Id]);
GO

-- fin.VendorInvoiceItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_TenantId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- fin.VendorInvoiceItem.VendorInvoiceId -> fin.VendorInvoice (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_VendorInvoiceId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_VendorInvoiceId] FOREIGN KEY ([VendorInvoiceId])
        REFERENCES [fin].[VendorInvoice] ([Id]);
GO

-- fin.VendorInvoiceItem.ExpenseGLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_ExpenseGLAccountId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_ExpenseGLAccountId] FOREIGN KEY ([ExpenseGLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- fin.VendorInvoiceItem.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_UnitOfMeasure')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- fin.VendorInvoiceItem.TaxCodeId -> cfg.TaxCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_TaxCodeId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_TaxCodeId] FOREIGN KEY ([TaxCodeId])
        REFERENCES [cfg].[TaxCode] ([Id]);
GO

-- fin.VendorInvoiceItem.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_CostCenterId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- fin.VendorInvoiceItem.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_ProfitCenterId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- fin.VendorInvoiceItem.InternalOrderId -> co.InternalOrder (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_InternalOrderId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_InternalOrderId] FOREIGN KEY ([InternalOrderId])
        REFERENCES [co].[InternalOrder] ([Id]);
GO

-- fin.VendorInvoiceItem.AssetId -> fin.Asset (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_AssetId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_AssetId] FOREIGN KEY ([AssetId])
        REFERENCES [fin].[Asset] ([Id]);
GO

-- fin.VendorInvoiceItem.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_SegmentId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- fin.VendorInvoiceItem.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_FunctionalAreaId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- fin.VendorInvoiceItem.BusinessAreaId -> org.BusinessArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_BusinessAreaId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_BusinessAreaId] FOREIGN KEY ([BusinessAreaId])
        REFERENCES [org].[BusinessArea] ([Id]);
GO

-- fin.VendorInvoiceItem.PlantId -> org.Plant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_fin_VendorInvoiceItem_PlantId')
    ALTER TABLE [fin].[VendorInvoiceItem] WITH CHECK
        ADD CONSTRAINT [FK_fin_VendorInvoiceItem_PlantId] FOREIGN KEY ([PlantId])
        REFERENCES [org].[Plant] ([Id]);
GO

-- co.ActivityPrice.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ActivityPrice_TenantId')
    ALTER TABLE [co].[ActivityPrice] WITH CHECK
        ADD CONSTRAINT [FK_co_ActivityPrice_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.ActivityPrice.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ActivityPrice_ControllingAreaId')
    ALTER TABLE [co].[ActivityPrice] WITH CHECK
        ADD CONSTRAINT [FK_co_ActivityPrice_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.ActivityPrice.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ActivityPrice_CostCenterId')
    ALTER TABLE [co].[ActivityPrice] WITH CHECK
        ADD CONSTRAINT [FK_co_ActivityPrice_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- co.ActivityPrice.ActivityTypeId -> co.ActivityType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ActivityPrice_ActivityTypeId')
    ALTER TABLE [co].[ActivityPrice] WITH CHECK
        ADD CONSTRAINT [FK_co_ActivityPrice_ActivityTypeId] FOREIGN KEY ([ActivityTypeId])
        REFERENCES [co].[ActivityType] ([Id]);
GO

-- co.ActivityPrice.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ActivityPrice_CurrencyCode')
    ALTER TABLE [co].[ActivityPrice] WITH CHECK
        ADD CONSTRAINT [FK_co_ActivityPrice_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.ActivityType.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ActivityType_TenantId')
    ALTER TABLE [co].[ActivityType] WITH CHECK
        ADD CONSTRAINT [FK_co_ActivityType_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.ActivityType.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ActivityType_ControllingAreaId')
    ALTER TABLE [co].[ActivityType] WITH CHECK
        ADD CONSTRAINT [FK_co_ActivityType_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.ActivityType.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ActivityType_UnitOfMeasure')
    ALTER TABLE [co].[ActivityType] WITH CHECK
        ADD CONSTRAINT [FK_co_ActivityType_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- co.ActivityType.AllocationCostElementId -> co.CostElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ActivityType_AllocationCostElementId')
    ALTER TABLE [co].[ActivityType] WITH CHECK
        ADD CONSTRAINT [FK_co_ActivityType_AllocationCostElementId] FOREIGN KEY ([AllocationCostElementId])
        REFERENCES [co].[CostElement] ([Id]);
GO

-- co.AllocationCycle.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationCycle_TenantId')
    ALTER TABLE [co].[AllocationCycle] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationCycle_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.AllocationCycle.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationCycle_ControllingAreaId')
    ALTER TABLE [co].[AllocationCycle] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationCycle_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.AllocationCycleReceiver.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationCycleReceiver_TenantId')
    ALTER TABLE [co].[AllocationCycleReceiver] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationCycleReceiver_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.AllocationCycleReceiver.AllocationCycleSegmentId -> co.AllocationCycleSegment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationCycleReceiver_AllocationCycleSegmentId')
    ALTER TABLE [co].[AllocationCycleReceiver] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationCycleReceiver_AllocationCycleSegmentId] FOREIGN KEY ([AllocationCycleSegmentId])
        REFERENCES [co].[AllocationCycleSegment] ([Id]);
GO

-- co.AllocationCycleSegment.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationCycleSegment_TenantId')
    ALTER TABLE [co].[AllocationCycleSegment] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationCycleSegment_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.AllocationCycleSegment.AllocationCycleId -> co.AllocationCycle (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationCycleSegment_AllocationCycleId')
    ALTER TABLE [co].[AllocationCycleSegment] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationCycleSegment_AllocationCycleId] FOREIGN KEY ([AllocationCycleId])
        REFERENCES [co].[AllocationCycle] ([Id]);
GO

-- co.AllocationCycleSegment.StatisticalKeyFigureId -> co.StatisticalKeyFigure (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationCycleSegment_StatisticalKeyFigureId')
    ALTER TABLE [co].[AllocationCycleSegment] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationCycleSegment_StatisticalKeyFigureId] FOREIGN KEY ([StatisticalKeyFigureId])
        REFERENCES [co].[StatisticalKeyFigure] ([Id]);
GO

-- co.AllocationCycleSegment.AssessmentCostElementId -> co.CostElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationCycleSegment_AssessmentCostElementId')
    ALTER TABLE [co].[AllocationCycleSegment] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationCycleSegment_AssessmentCostElementId] FOREIGN KEY ([AssessmentCostElementId])
        REFERENCES [co].[CostElement] ([Id]);
GO

-- co.AllocationRun.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationRun_TenantId')
    ALTER TABLE [co].[AllocationRun] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationRun_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.AllocationRun.AllocationCycleId -> co.AllocationCycle (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationRun_AllocationCycleId')
    ALTER TABLE [co].[AllocationRun] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationRun_AllocationCycleId] FOREIGN KEY ([AllocationCycleId])
        REFERENCES [co].[AllocationCycle] ([Id]);
GO

-- co.AllocationRun.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_AllocationRun_CurrencyCode')
    ALTER TABLE [co].[AllocationRun] WITH CHECK
        ADD CONSTRAINT [FK_co_AllocationRun_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.Commitment.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_Commitment_TenantId')
    ALTER TABLE [co].[Commitment] WITH CHECK
        ADD CONSTRAINT [FK_co_Commitment_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.Commitment.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_Commitment_ControllingAreaId')
    ALTER TABLE [co].[Commitment] WITH CHECK
        ADD CONSTRAINT [FK_co_Commitment_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.Commitment.CostElementId -> co.CostElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_Commitment_CostElementId')
    ALTER TABLE [co].[Commitment] WITH CHECK
        ADD CONSTRAINT [FK_co_Commitment_CostElementId] FOREIGN KEY ([CostElementId])
        REFERENCES [co].[CostElement] ([Id]);
GO

-- co.Commitment.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_Commitment_CurrencyCode')
    ALTER TABLE [co].[Commitment] WITH CHECK
        ADD CONSTRAINT [FK_co_Commitment_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.ControllingPosting.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_TenantId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.ControllingPosting.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_ControllingAreaId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.ControllingPosting.CostElementId -> co.CostElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_CostElementId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_CostElementId] FOREIGN KEY ([CostElementId])
        REFERENCES [co].[CostElement] ([Id]);
GO

-- co.ControllingPosting.ActivityTypeId -> co.ActivityType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_ActivityTypeId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_ActivityTypeId] FOREIGN KEY ([ActivityTypeId])
        REFERENCES [co].[ActivityType] ([Id]);
GO

-- co.ControllingPosting.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_CompanyCodeId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- co.ControllingPosting.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_ProfitCenterId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- co.ControllingPosting.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_FunctionalAreaId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- co.ControllingPosting.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_SegmentId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- co.ControllingPosting.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_CurrencyCode')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.ControllingPosting.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_UnitOfMeasure')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- co.ControllingPosting.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_JournalEntryHeaderId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- co.ControllingPosting.AllocationRunId -> co.AllocationRun (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_AllocationRunId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_AllocationRunId] FOREIGN KEY ([AllocationRunId])
        REFERENCES [co].[AllocationRun] ([Id]);
GO

-- co.ControllingPosting.SettlementDocumentId -> co.SettlementDocument (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingPosting_SettlementDocumentId')
    ALTER TABLE [co].[ControllingPosting] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingPosting_SettlementDocumentId] FOREIGN KEY ([SettlementDocumentId])
        REFERENCES [co].[SettlementDocument] ([Id]);
GO

-- co.ControllingTotal.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingTotal_TenantId')
    ALTER TABLE [co].[ControllingTotal] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingTotal_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.ControllingTotal.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingTotal_ControllingAreaId')
    ALTER TABLE [co].[ControllingTotal] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingTotal_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.ControllingTotal.CostElementId -> co.CostElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingTotal_CostElementId')
    ALTER TABLE [co].[ControllingTotal] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingTotal_CostElementId] FOREIGN KEY ([CostElementId])
        REFERENCES [co].[CostElement] ([Id]);
GO

-- co.ControllingTotal.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ControllingTotal_CurrencyCode')
    ALTER TABLE [co].[ControllingTotal] WITH CHECK
        ADD CONSTRAINT [FK_co_ControllingTotal_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.CostCenter.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_TenantId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.CostCenter.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_ControllingAreaId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.CostCenter.CostCenterCategoryId -> co.CostCenterCategory (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_CostCenterCategoryId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_CostCenterCategoryId] FOREIGN KEY ([CostCenterCategoryId])
        REFERENCES [co].[CostCenterCategory] ([Id]);
GO

-- co.CostCenter.HierarchyNodeId -> co.HierarchyNode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_HierarchyNodeId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_HierarchyNodeId] FOREIGN KEY ([HierarchyNodeId])
        REFERENCES [co].[HierarchyNode] ([Id]);
GO

-- co.CostCenter.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_CompanyCodeId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- co.CostCenter.BusinessAreaId -> org.BusinessArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_BusinessAreaId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_BusinessAreaId] FOREIGN KEY ([BusinessAreaId])
        REFERENCES [org].[BusinessArea] ([Id]);
GO

-- co.CostCenter.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_ProfitCenterId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- co.CostCenter.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_FunctionalAreaId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- co.CostCenter.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_SegmentId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- co.CostCenter.PlantId -> org.Plant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_PlantId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_PlantId] FOREIGN KEY ([PlantId])
        REFERENCES [org].[Plant] ([Id]);
GO

-- co.CostCenter.ResponsiblePersonPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_ResponsiblePersonPartnerId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_ResponsiblePersonPartnerId] FOREIGN KEY ([ResponsiblePersonPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- co.CostCenter.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_CurrencyCode')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.CostCenter.AddressId -> mdm.Address (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenter_AddressId')
    ALTER TABLE [co].[CostCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenter_AddressId] FOREIGN KEY ([AddressId])
        REFERENCES [mdm].[Address] ([Id]);
GO

-- co.CostCenterCategory.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostCenterCategory_TenantId')
    ALTER TABLE [co].[CostCenterCategory] WITH CHECK
        ADD CONSTRAINT [FK_co_CostCenterCategory_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.CostElement.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostElement_TenantId')
    ALTER TABLE [co].[CostElement] WITH CHECK
        ADD CONSTRAINT [FK_co_CostElement_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.CostElement.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostElement_ControllingAreaId')
    ALTER TABLE [co].[CostElement] WITH CHECK
        ADD CONSTRAINT [FK_co_CostElement_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.CostElement.GLAccountId -> mdm.GLAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostElement_GLAccountId')
    ALTER TABLE [co].[CostElement] WITH CHECK
        ADD CONSTRAINT [FK_co_CostElement_GLAccountId] FOREIGN KEY ([GLAccountId])
        REFERENCES [mdm].[GLAccount] ([Id]);
GO

-- co.CostElement.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostElement_FunctionalAreaId')
    ALTER TABLE [co].[CostElement] WITH CHECK
        ADD CONSTRAINT [FK_co_CostElement_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- co.CostElement.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostElement_UnitOfMeasure')
    ALTER TABLE [co].[CostElement] WITH CHECK
        ADD CONSTRAINT [FK_co_CostElement_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- co.CostElement.DefaultCostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostElement_DefaultCostCenterId')
    ALTER TABLE [co].[CostElement] WITH CHECK
        ADD CONSTRAINT [FK_co_CostElement_DefaultCostCenterId] FOREIGN KEY ([DefaultCostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- co.CostElement.DefaultInternalOrderId -> co.InternalOrder (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_CostElement_DefaultInternalOrderId')
    ALTER TABLE [co].[CostElement] WITH CHECK
        ADD CONSTRAINT [FK_co_CostElement_DefaultInternalOrderId] FOREIGN KEY ([DefaultInternalOrderId])
        REFERENCES [co].[InternalOrder] ([Id]);
GO

-- co.HierarchyNode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_HierarchyNode_TenantId')
    ALTER TABLE [co].[HierarchyNode] WITH CHECK
        ADD CONSTRAINT [FK_co_HierarchyNode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.HierarchyNode.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_HierarchyNode_ControllingAreaId')
    ALTER TABLE [co].[HierarchyNode] WITH CHECK
        ADD CONSTRAINT [FK_co_HierarchyNode_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.HierarchyNode.ParentNodeId -> co.HierarchyNode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_HierarchyNode_ParentNodeId')
    ALTER TABLE [co].[HierarchyNode] WITH CHECK
        ADD CONSTRAINT [FK_co_HierarchyNode_ParentNodeId] FOREIGN KEY ([ParentNodeId])
        REFERENCES [co].[HierarchyNode] ([Id]);
GO

-- co.InternalOrder.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_TenantId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.InternalOrder.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_ControllingAreaId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.InternalOrder.OrderTypeId -> co.InternalOrderType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_OrderTypeId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_OrderTypeId] FOREIGN KEY ([OrderTypeId])
        REFERENCES [co].[InternalOrderType] ([Id]);
GO

-- co.InternalOrder.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_CompanyCodeId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- co.InternalOrder.BusinessAreaId -> org.BusinessArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_BusinessAreaId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_BusinessAreaId] FOREIGN KEY ([BusinessAreaId])
        REFERENCES [org].[BusinessArea] ([Id]);
GO

-- co.InternalOrder.PlantId -> org.Plant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_PlantId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_PlantId] FOREIGN KEY ([PlantId])
        REFERENCES [org].[Plant] ([Id]);
GO

-- co.InternalOrder.ResponsibleCostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_ResponsibleCostCenterId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_ResponsibleCostCenterId] FOREIGN KEY ([ResponsibleCostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- co.InternalOrder.RequestingCostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_RequestingCostCenterId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_RequestingCostCenterId] FOREIGN KEY ([RequestingCostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- co.InternalOrder.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_ProfitCenterId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- co.InternalOrder.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_SegmentId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- co.InternalOrder.FunctionalAreaId -> org.FunctionalArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_FunctionalAreaId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_FunctionalAreaId] FOREIGN KEY ([FunctionalAreaId])
        REFERENCES [org].[FunctionalArea] ([Id]);
GO

-- co.InternalOrder.ResponsiblePersonPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_ResponsiblePersonPartnerId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_ResponsiblePersonPartnerId] FOREIGN KEY ([ResponsiblePersonPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- co.InternalOrder.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_CurrencyCode')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.InternalOrder.AssetId -> fin.Asset (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrder_AssetId')
    ALTER TABLE [co].[InternalOrder] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrder_AssetId] FOREIGN KEY ([AssetId])
        REFERENCES [fin].[Asset] ([Id]);
GO

-- co.InternalOrderBudget.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrderBudget_TenantId')
    ALTER TABLE [co].[InternalOrderBudget] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrderBudget_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.InternalOrderBudget.InternalOrderId -> co.InternalOrder (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrderBudget_InternalOrderId')
    ALTER TABLE [co].[InternalOrderBudget] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrderBudget_InternalOrderId] FOREIGN KEY ([InternalOrderId])
        REFERENCES [co].[InternalOrder] ([Id]);
GO

-- co.InternalOrderBudget.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrderBudget_CurrencyCode')
    ALTER TABLE [co].[InternalOrderBudget] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrderBudget_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.InternalOrderType.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrderType_TenantId')
    ALTER TABLE [co].[InternalOrderType] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrderType_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.InternalOrderType.NumberRangeObjectId -> cfg.NumberRangeObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_InternalOrderType_NumberRangeObjectId')
    ALTER TABLE [co].[InternalOrderType] WITH CHECK
        ADD CONSTRAINT [FK_co_InternalOrderType_NumberRangeObjectId] FOREIGN KEY ([NumberRangeObjectId])
        REFERENCES [cfg].[NumberRangeObject] ([Id]);
GO

-- co.PlanEntry.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_PlanEntry_TenantId')
    ALTER TABLE [co].[PlanEntry] WITH CHECK
        ADD CONSTRAINT [FK_co_PlanEntry_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.PlanEntry.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_PlanEntry_ControllingAreaId')
    ALTER TABLE [co].[PlanEntry] WITH CHECK
        ADD CONSTRAINT [FK_co_PlanEntry_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.PlanEntry.CostElementId -> co.CostElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_PlanEntry_CostElementId')
    ALTER TABLE [co].[PlanEntry] WITH CHECK
        ADD CONSTRAINT [FK_co_PlanEntry_CostElementId] FOREIGN KEY ([CostElementId])
        REFERENCES [co].[CostElement] ([Id]);
GO

-- co.PlanEntry.ActivityTypeId -> co.ActivityType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_PlanEntry_ActivityTypeId')
    ALTER TABLE [co].[PlanEntry] WITH CHECK
        ADD CONSTRAINT [FK_co_PlanEntry_ActivityTypeId] FOREIGN KEY ([ActivityTypeId])
        REFERENCES [co].[ActivityType] ([Id]);
GO

-- co.PlanEntry.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_PlanEntry_CurrencyCode')
    ALTER TABLE [co].[PlanEntry] WITH CHECK
        ADD CONSTRAINT [FK_co_PlanEntry_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.PlanEntry.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_PlanEntry_UnitOfMeasure')
    ALTER TABLE [co].[PlanEntry] WITH CHECK
        ADD CONSTRAINT [FK_co_PlanEntry_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- co.ProfitCenter.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenter_TenantId')
    ALTER TABLE [co].[ProfitCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenter_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.ProfitCenter.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenter_ControllingAreaId')
    ALTER TABLE [co].[ProfitCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenter_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.ProfitCenter.HierarchyNodeId -> co.HierarchyNode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenter_HierarchyNodeId')
    ALTER TABLE [co].[ProfitCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenter_HierarchyNodeId] FOREIGN KEY ([HierarchyNodeId])
        REFERENCES [co].[HierarchyNode] ([Id]);
GO

-- co.ProfitCenter.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenter_CompanyCodeId')
    ALTER TABLE [co].[ProfitCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenter_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- co.ProfitCenter.SegmentId -> org.Segment (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenter_SegmentId')
    ALTER TABLE [co].[ProfitCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenter_SegmentId] FOREIGN KEY ([SegmentId])
        REFERENCES [org].[Segment] ([Id]);
GO

-- co.ProfitCenter.BusinessAreaId -> org.BusinessArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenter_BusinessAreaId')
    ALTER TABLE [co].[ProfitCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenter_BusinessAreaId] FOREIGN KEY ([BusinessAreaId])
        REFERENCES [org].[BusinessArea] ([Id]);
GO

-- co.ProfitCenter.ResponsiblePersonPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenter_ResponsiblePersonPartnerId')
    ALTER TABLE [co].[ProfitCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenter_ResponsiblePersonPartnerId] FOREIGN KEY ([ResponsiblePersonPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- co.ProfitCenter.AddressId -> mdm.Address (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenter_AddressId')
    ALTER TABLE [co].[ProfitCenter] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenter_AddressId] FOREIGN KEY ([AddressId])
        REFERENCES [mdm].[Address] ([Id]);
GO

-- co.ProfitCenterAssignment.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenterAssignment_TenantId')
    ALTER TABLE [co].[ProfitCenterAssignment] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenterAssignment_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.ProfitCenterAssignment.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenterAssignment_ProfitCenterId')
    ALTER TABLE [co].[ProfitCenterAssignment] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenterAssignment_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- co.ProfitCenterAssignment.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_ProfitCenterAssignment_CompanyCodeId')
    ALTER TABLE [co].[ProfitCenterAssignment] WITH CHECK
        ADD CONSTRAINT [FK_co_ProfitCenterAssignment_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- co.SettlementDocument.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementDocument_TenantId')
    ALTER TABLE [co].[SettlementDocument] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementDocument_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.SettlementDocument.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementDocument_ControllingAreaId')
    ALTER TABLE [co].[SettlementDocument] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementDocument_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.SettlementDocument.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementDocument_CurrencyCode')
    ALTER TABLE [co].[SettlementDocument] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementDocument_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- co.SettlementDocument.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementDocument_JournalEntryHeaderId')
    ALTER TABLE [co].[SettlementDocument] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementDocument_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- co.SettlementDocumentItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementDocumentItem_TenantId')
    ALTER TABLE [co].[SettlementDocumentItem] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementDocumentItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.SettlementDocumentItem.SettlementDocumentId -> co.SettlementDocument (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementDocumentItem_SettlementDocumentId')
    ALTER TABLE [co].[SettlementDocumentItem] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementDocumentItem_SettlementDocumentId] FOREIGN KEY ([SettlementDocumentId])
        REFERENCES [co].[SettlementDocument] ([Id]);
GO

-- co.SettlementDocumentItem.SettlementRuleId -> co.SettlementRule (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementDocumentItem_SettlementRuleId')
    ALTER TABLE [co].[SettlementDocumentItem] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementDocumentItem_SettlementRuleId] FOREIGN KEY ([SettlementRuleId])
        REFERENCES [co].[SettlementRule] ([Id]);
GO

-- co.SettlementDocumentItem.CostElementId -> co.CostElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementDocumentItem_CostElementId')
    ALTER TABLE [co].[SettlementDocumentItem] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementDocumentItem_CostElementId] FOREIGN KEY ([CostElementId])
        REFERENCES [co].[CostElement] ([Id]);
GO

-- co.SettlementRule.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementRule_TenantId')
    ALTER TABLE [co].[SettlementRule] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementRule_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.SettlementRule.SettlementCostElementId -> co.CostElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_SettlementRule_SettlementCostElementId')
    ALTER TABLE [co].[SettlementRule] WITH CHECK
        ADD CONSTRAINT [FK_co_SettlementRule_SettlementCostElementId] FOREIGN KEY ([SettlementCostElementId])
        REFERENCES [co].[CostElement] ([Id]);
GO

-- co.StatisticalKeyFigure.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_StatisticalKeyFigure_TenantId')
    ALTER TABLE [co].[StatisticalKeyFigure] WITH CHECK
        ADD CONSTRAINT [FK_co_StatisticalKeyFigure_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.StatisticalKeyFigure.ControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_StatisticalKeyFigure_ControllingAreaId')
    ALTER TABLE [co].[StatisticalKeyFigure] WITH CHECK
        ADD CONSTRAINT [FK_co_StatisticalKeyFigure_ControllingAreaId] FOREIGN KEY ([ControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- co.StatisticalKeyFigure.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_StatisticalKeyFigure_UnitOfMeasure')
    ALTER TABLE [co].[StatisticalKeyFigure] WITH CHECK
        ADD CONSTRAINT [FK_co_StatisticalKeyFigure_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- co.StatisticalKeyFigureValue.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_StatisticalKeyFigureValue_TenantId')
    ALTER TABLE [co].[StatisticalKeyFigureValue] WITH CHECK
        ADD CONSTRAINT [FK_co_StatisticalKeyFigureValue_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- co.StatisticalKeyFigureValue.StatisticalKeyFigureId -> co.StatisticalKeyFigure (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_StatisticalKeyFigureValue_StatisticalKeyFigureId')
    ALTER TABLE [co].[StatisticalKeyFigureValue] WITH CHECK
        ADD CONSTRAINT [FK_co_StatisticalKeyFigureValue_StatisticalKeyFigureId] FOREIGN KEY ([StatisticalKeyFigureId])
        REFERENCES [co].[StatisticalKeyFigure] ([Id]);
GO

-- co.StatisticalKeyFigureValue.UnitOfMeasure -> cfg.UnitOfMeasure (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_co_StatisticalKeyFigureValue_UnitOfMeasure')
    ALTER TABLE [co].[StatisticalKeyFigureValue] WITH CHECK
        ADD CONSTRAINT [FK_co_StatisticalKeyFigureValue_UnitOfMeasure] FOREIGN KEY ([TenantId], [UnitOfMeasure])
        REFERENCES [cfg].[UnitOfMeasure] ([TenantId], [UnitOfMeasure]);
GO

-- wf.Notification.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_Notification_TenantId')
    ALTER TABLE [wf].[Notification] WITH CHECK
        ADD CONSTRAINT [FK_wf_Notification_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- wf.Notification.RecipientUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_Notification_RecipientUserId')
    ALTER TABLE [wf].[Notification] WITH CHECK
        ADD CONSTRAINT [FK_wf_Notification_RecipientUserId] FOREIGN KEY ([RecipientUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- wf.Notification.WorkflowTaskId -> wf.WorkflowTask (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_Notification_WorkflowTaskId')
    ALTER TABLE [wf].[Notification] WITH CHECK
        ADD CONSTRAINT [FK_wf_Notification_WorkflowTaskId] FOREIGN KEY ([WorkflowTaskId])
        REFERENCES [wf].[WorkflowTask] ([Id]);
GO

-- wf.WorkflowDefinition.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowDefinition_TenantId')
    ALTER TABLE [wf].[WorkflowDefinition] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowDefinition_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- wf.WorkflowHistory.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowHistory_TenantId')
    ALTER TABLE [wf].[WorkflowHistory] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowHistory_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- wf.WorkflowHistory.WorkflowInstanceId -> wf.WorkflowInstance (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowHistory_WorkflowInstanceId')
    ALTER TABLE [wf].[WorkflowHistory] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowHistory_WorkflowInstanceId] FOREIGN KEY ([WorkflowInstanceId])
        REFERENCES [wf].[WorkflowInstance] ([Id]);
GO

-- wf.WorkflowHistory.PerformedByUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowHistory_PerformedByUserId')
    ALTER TABLE [wf].[WorkflowHistory] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowHistory_PerformedByUserId] FOREIGN KEY ([PerformedByUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- wf.WorkflowInstance.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowInstance_TenantId')
    ALTER TABLE [wf].[WorkflowInstance] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowInstance_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- wf.WorkflowInstance.WorkflowDefinitionId -> wf.WorkflowDefinition (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowInstance_WorkflowDefinitionId')
    ALTER TABLE [wf].[WorkflowInstance] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowInstance_WorkflowDefinitionId] FOREIGN KEY ([WorkflowDefinitionId])
        REFERENCES [wf].[WorkflowDefinition] ([Id]);
GO

-- wf.WorkflowInstance.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowInstance_CompanyCodeId')
    ALTER TABLE [wf].[WorkflowInstance] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowInstance_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- wf.WorkflowInstance.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowInstance_CurrencyCode')
    ALTER TABLE [wf].[WorkflowInstance] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowInstance_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- wf.WorkflowRule.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowRule_TenantId')
    ALTER TABLE [wf].[WorkflowRule] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowRule_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- wf.WorkflowRule.WorkflowDefinitionId -> wf.WorkflowDefinition (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowRule_WorkflowDefinitionId')
    ALTER TABLE [wf].[WorkflowRule] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowRule_WorkflowDefinitionId] FOREIGN KEY ([WorkflowDefinitionId])
        REFERENCES [wf].[WorkflowDefinition] ([Id]);
GO

-- wf.WorkflowRule.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowRule_CompanyCodeId')
    ALTER TABLE [wf].[WorkflowRule] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowRule_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- wf.WorkflowRule.DocumentTypeId -> cfg.DocumentType (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowRule_DocumentTypeId')
    ALTER TABLE [wf].[WorkflowRule] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowRule_DocumentTypeId] FOREIGN KEY ([DocumentTypeId])
        REFERENCES [cfg].[DocumentType] ([Id]);
GO

-- wf.WorkflowRule.CostCenterId -> co.CostCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowRule_CostCenterId')
    ALTER TABLE [wf].[WorkflowRule] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowRule_CostCenterId] FOREIGN KEY ([CostCenterId])
        REFERENCES [co].[CostCenter] ([Id]);
GO

-- wf.WorkflowRule.ProfitCenterId -> co.ProfitCenter (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowRule_ProfitCenterId')
    ALTER TABLE [wf].[WorkflowRule] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowRule_ProfitCenterId] FOREIGN KEY ([ProfitCenterId])
        REFERENCES [co].[ProfitCenter] ([Id]);
GO

-- wf.WorkflowRule.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowRule_CurrencyCode')
    ALTER TABLE [wf].[WorkflowRule] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowRule_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- wf.WorkflowStep.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowStep_TenantId')
    ALTER TABLE [wf].[WorkflowStep] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowStep_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- wf.WorkflowStep.WorkflowDefinitionId -> wf.WorkflowDefinition (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowStep_WorkflowDefinitionId')
    ALTER TABLE [wf].[WorkflowStep] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowStep_WorkflowDefinitionId] FOREIGN KEY ([WorkflowDefinitionId])
        REFERENCES [wf].[WorkflowDefinition] ([Id]);
GO

-- wf.WorkflowStep.ApproverRoleId -> sec.Role (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowStep_ApproverRoleId')
    ALTER TABLE [wf].[WorkflowStep] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowStep_ApproverRoleId] FOREIGN KEY ([ApproverRoleId])
        REFERENCES [sec].[Role] ([Id]);
GO

-- wf.WorkflowStep.ApproverUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowStep_ApproverUserId')
    ALTER TABLE [wf].[WorkflowStep] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowStep_ApproverUserId] FOREIGN KEY ([ApproverUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- wf.WorkflowStep.EscalationRoleId -> sec.Role (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowStep_EscalationRoleId')
    ALTER TABLE [wf].[WorkflowStep] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowStep_EscalationRoleId] FOREIGN KEY ([EscalationRoleId])
        REFERENCES [sec].[Role] ([Id]);
GO

-- wf.WorkflowTask.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowTask_TenantId')
    ALTER TABLE [wf].[WorkflowTask] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowTask_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- wf.WorkflowTask.WorkflowInstanceId -> wf.WorkflowInstance (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowTask_WorkflowInstanceId')
    ALTER TABLE [wf].[WorkflowTask] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowTask_WorkflowInstanceId] FOREIGN KEY ([WorkflowInstanceId])
        REFERENCES [wf].[WorkflowInstance] ([Id]);
GO

-- wf.WorkflowTask.AssignedUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowTask_AssignedUserId')
    ALTER TABLE [wf].[WorkflowTask] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowTask_AssignedUserId] FOREIGN KEY ([AssignedUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- wf.WorkflowTask.AssignedRoleId -> sec.Role (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowTask_AssignedRoleId')
    ALTER TABLE [wf].[WorkflowTask] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowTask_AssignedRoleId] FOREIGN KEY ([AssignedRoleId])
        REFERENCES [sec].[Role] ([Id]);
GO

-- wf.WorkflowTask.DecidedByUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowTask_DecidedByUserId')
    ALTER TABLE [wf].[WorkflowTask] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowTask_DecidedByUserId] FOREIGN KEY ([DecidedByUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- wf.WorkflowTask.DelegatedToUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_wf_WorkflowTask_DelegatedToUserId')
    ALTER TABLE [wf].[WorkflowTask] WITH CHECK
        ADD CONSTRAINT [FK_wf_WorkflowTask_DelegatedToUserId] FOREIGN KEY ([DelegatedToUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.AuthorizationField.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_AuthorizationField_TenantId')
    ALTER TABLE [sec].[AuthorizationField] WITH CHECK
        ADD CONSTRAINT [FK_sec_AuthorizationField_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.AuthorizationField.AuthorizationObjectId -> sec.AuthorizationObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_AuthorizationField_AuthorizationObjectId')
    ALTER TABLE [sec].[AuthorizationField] WITH CHECK
        ADD CONSTRAINT [FK_sec_AuthorizationField_AuthorizationObjectId] FOREIGN KEY ([AuthorizationObjectId])
        REFERENCES [sec].[AuthorizationObject] ([Id]);
GO

-- sec.AuthorizationField.DataElementId -> cfg.DictionaryDataElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_AuthorizationField_DataElementId')
    ALTER TABLE [sec].[AuthorizationField] WITH CHECK
        ADD CONSTRAINT [FK_sec_AuthorizationField_DataElementId] FOREIGN KEY ([DataElementId])
        REFERENCES [cfg].[DictionaryDataElement] ([Id]);
GO

-- sec.AuthorizationObject.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_AuthorizationObject_TenantId')
    ALTER TABLE [sec].[AuthorizationObject] WITH CHECK
        ADD CONSTRAINT [FK_sec_AuthorizationObject_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.AuthorizationValue.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_AuthorizationValue_TenantId')
    ALTER TABLE [sec].[AuthorizationValue] WITH CHECK
        ADD CONSTRAINT [FK_sec_AuthorizationValue_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.AuthorizationValue.RoleId -> sec.Role (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_AuthorizationValue_RoleId')
    ALTER TABLE [sec].[AuthorizationValue] WITH CHECK
        ADD CONSTRAINT [FK_sec_AuthorizationValue_RoleId] FOREIGN KEY ([RoleId])
        REFERENCES [sec].[Role] ([Id]);
GO

-- sec.AuthorizationValue.AuthorizationObjectId -> sec.AuthorizationObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_AuthorizationValue_AuthorizationObjectId')
    ALTER TABLE [sec].[AuthorizationValue] WITH CHECK
        ADD CONSTRAINT [FK_sec_AuthorizationValue_AuthorizationObjectId] FOREIGN KEY ([AuthorizationObjectId])
        REFERENCES [sec].[AuthorizationObject] ([Id]);
GO

-- sec.AuthorizationValue.AuthorizationFieldId -> sec.AuthorizationField (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_AuthorizationValue_AuthorizationFieldId')
    ALTER TABLE [sec].[AuthorizationValue] WITH CHECK
        ADD CONSTRAINT [FK_sec_AuthorizationValue_AuthorizationFieldId] FOREIGN KEY ([AuthorizationFieldId])
        REFERENCES [sec].[AuthorizationField] ([Id]);
GO

-- sec.LoginHistory.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_LoginHistory_TenantId')
    ALTER TABLE [sec].[LoginHistory] WITH CHECK
        ADD CONSTRAINT [FK_sec_LoginHistory_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.LoginHistory.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_LoginHistory_UserId')
    ALTER TABLE [sec].[LoginHistory] WITH CHECK
        ADD CONSTRAINT [FK_sec_LoginHistory_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.PasswordHistory.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_PasswordHistory_TenantId')
    ALTER TABLE [sec].[PasswordHistory] WITH CHECK
        ADD CONSTRAINT [FK_sec_PasswordHistory_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.PasswordHistory.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_PasswordHistory_UserId')
    ALTER TABLE [sec].[PasswordHistory] WITH CHECK
        ADD CONSTRAINT [FK_sec_PasswordHistory_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.Permission.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_Permission_TenantId')
    ALTER TABLE [sec].[Permission] WITH CHECK
        ADD CONSTRAINT [FK_sec_Permission_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.Role.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_Role_TenantId')
    ALTER TABLE [sec].[Role] WITH CHECK
        ADD CONSTRAINT [FK_sec_Role_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.Role.ParentRoleId -> sec.Role (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_Role_ParentRoleId')
    ALTER TABLE [sec].[Role] WITH CHECK
        ADD CONSTRAINT [FK_sec_Role_ParentRoleId] FOREIGN KEY ([ParentRoleId])
        REFERENCES [sec].[Role] ([Id]);
GO

-- sec.RolePermission.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_RolePermission_TenantId')
    ALTER TABLE [sec].[RolePermission] WITH CHECK
        ADD CONSTRAINT [FK_sec_RolePermission_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.RolePermission.RoleId -> sec.Role (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_RolePermission_RoleId')
    ALTER TABLE [sec].[RolePermission] WITH CHECK
        ADD CONSTRAINT [FK_sec_RolePermission_RoleId] FOREIGN KEY ([RoleId])
        REFERENCES [sec].[Role] ([Id]);
GO

-- sec.RolePermission.PermissionId -> sec.Permission (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_RolePermission_PermissionId')
    ALTER TABLE [sec].[RolePermission] WITH CHECK
        ADD CONSTRAINT [FK_sec_RolePermission_PermissionId] FOREIGN KEY ([PermissionId])
        REFERENCES [sec].[Permission] ([Id]);
GO

-- sec.RoleTransactionCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_RoleTransactionCode_TenantId')
    ALTER TABLE [sec].[RoleTransactionCode] WITH CHECK
        ADD CONSTRAINT [FK_sec_RoleTransactionCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.RoleTransactionCode.RoleId -> sec.Role (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_RoleTransactionCode_RoleId')
    ALTER TABLE [sec].[RoleTransactionCode] WITH CHECK
        ADD CONSTRAINT [FK_sec_RoleTransactionCode_RoleId] FOREIGN KEY ([RoleId])
        REFERENCES [sec].[Role] ([Id]);
GO

-- sec.RoleTransactionCode.TransactionCodeId -> sec.TransactionCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_RoleTransactionCode_TransactionCodeId')
    ALTER TABLE [sec].[RoleTransactionCode] WITH CHECK
        ADD CONSTRAINT [FK_sec_RoleTransactionCode_TransactionCodeId] FOREIGN KEY ([TransactionCodeId])
        REFERENCES [sec].[TransactionCode] ([Id]);
GO

-- sec.SegregationOfDutiesRule.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_SegregationOfDutiesRule_TenantId')
    ALTER TABLE [sec].[SegregationOfDutiesRule] WITH CHECK
        ADD CONSTRAINT [FK_sec_SegregationOfDutiesRule_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.SegregationOfDutiesViolation.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_SegregationOfDutiesViolation_TenantId')
    ALTER TABLE [sec].[SegregationOfDutiesViolation] WITH CHECK
        ADD CONSTRAINT [FK_sec_SegregationOfDutiesViolation_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.SegregationOfDutiesViolation.SegregationOfDutiesRuleId -> sec.SegregationOfDutiesRule (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_SegregationOfDutiesViolation_SegregationOfDutiesRuleId')
    ALTER TABLE [sec].[SegregationOfDutiesViolation] WITH CHECK
        ADD CONSTRAINT [FK_sec_SegregationOfDutiesViolation_SegregationOfDutiesRuleId] FOREIGN KEY ([SegregationOfDutiesRuleId])
        REFERENCES [sec].[SegregationOfDutiesRule] ([Id]);
GO

-- sec.SegregationOfDutiesViolation.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_SegregationOfDutiesViolation_UserId')
    ALTER TABLE [sec].[SegregationOfDutiesViolation] WITH CHECK
        ADD CONSTRAINT [FK_sec_SegregationOfDutiesViolation_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.TransactionCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_TransactionCode_TenantId')
    ALTER TABLE [sec].[TransactionCode] WITH CHECK
        ADD CONSTRAINT [FK_sec_TransactionCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.TransactionCode.RequiredPermissionId -> sec.Permission (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_TransactionCode_RequiredPermissionId')
    ALTER TABLE [sec].[TransactionCode] WITH CHECK
        ADD CONSTRAINT [FK_sec_TransactionCode_RequiredPermissionId] FOREIGN KEY ([RequiredPermissionId])
        REFERENCES [sec].[Permission] ([Id]);
GO

-- sec.TransactionCode.AuthorizationObjectId -> sec.AuthorizationObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_TransactionCode_AuthorizationObjectId')
    ALTER TABLE [sec].[TransactionCode] WITH CHECK
        ADD CONSTRAINT [FK_sec_TransactionCode_AuthorizationObjectId] FOREIGN KEY ([AuthorizationObjectId])
        REFERENCES [sec].[AuthorizationObject] ([Id]);
GO

-- sec.User.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_User_TenantId')
    ALTER TABLE [sec].[User] WITH CHECK
        ADD CONSTRAINT [FK_sec_User_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.User.BusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_User_BusinessPartnerId')
    ALTER TABLE [sec].[User] WITH CHECK
        ADD CONSTRAINT [FK_sec_User_BusinessPartnerId] FOREIGN KEY ([BusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- sec.User.LanguageCode -> cfg.Language (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_User_LanguageCode')
    ALTER TABLE [sec].[User] WITH CHECK
        ADD CONSTRAINT [FK_sec_User_LanguageCode] FOREIGN KEY ([LanguageCode])
        REFERENCES [cfg].[Language] ([LanguageCode]);
GO

-- sec.User.DefaultCompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_User_DefaultCompanyCodeId')
    ALTER TABLE [sec].[User] WITH CHECK
        ADD CONSTRAINT [FK_sec_User_DefaultCompanyCodeId] FOREIGN KEY ([DefaultCompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- sec.User.DefaultControllingAreaId -> org.ControllingArea (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_User_DefaultControllingAreaId')
    ALTER TABLE [sec].[User] WITH CHECK
        ADD CONSTRAINT [FK_sec_User_DefaultControllingAreaId] FOREIGN KEY ([DefaultControllingAreaId])
        REFERENCES [org].[ControllingArea] ([Id]);
GO

-- sec.User.DefaultCurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_User_DefaultCurrencyCode')
    ALTER TABLE [sec].[User] WITH CHECK
        ADD CONSTRAINT [FK_sec_User_DefaultCurrencyCode] FOREIGN KEY ([TenantId], [DefaultCurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- sec.UserAuthorization.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserAuthorization_TenantId')
    ALTER TABLE [sec].[UserAuthorization] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserAuthorization_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.UserAuthorization.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserAuthorization_UserId')
    ALTER TABLE [sec].[UserAuthorization] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserAuthorization_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.UserAuthorization.AuthorizationObjectId -> sec.AuthorizationObject (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserAuthorization_AuthorizationObjectId')
    ALTER TABLE [sec].[UserAuthorization] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserAuthorization_AuthorizationObjectId] FOREIGN KEY ([AuthorizationObjectId])
        REFERENCES [sec].[AuthorizationObject] ([Id]);
GO

-- sec.UserAuthorization.AuthorizationFieldId -> sec.AuthorizationField (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserAuthorization_AuthorizationFieldId')
    ALTER TABLE [sec].[UserAuthorization] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserAuthorization_AuthorizationFieldId] FOREIGN KEY ([AuthorizationFieldId])
        REFERENCES [sec].[AuthorizationField] ([Id]);
GO

-- sec.UserCompanyCode.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserCompanyCode_TenantId')
    ALTER TABLE [sec].[UserCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserCompanyCode_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.UserCompanyCode.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserCompanyCode_UserId')
    ALTER TABLE [sec].[UserCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserCompanyCode_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.UserCompanyCode.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserCompanyCode_CompanyCodeId')
    ALTER TABLE [sec].[UserCompanyCode] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserCompanyCode_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- sec.UserProfile.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserProfile_TenantId')
    ALTER TABLE [sec].[UserProfile] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserProfile_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.UserProfile.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserProfile_UserId')
    ALTER TABLE [sec].[UserProfile] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserProfile_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.UserRole.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserRole_TenantId')
    ALTER TABLE [sec].[UserRole] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserRole_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.UserRole.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserRole_UserId')
    ALTER TABLE [sec].[UserRole] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserRole_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.UserRole.RoleId -> sec.Role (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserRole_RoleId')
    ALTER TABLE [sec].[UserRole] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserRole_RoleId] FOREIGN KEY ([RoleId])
        REFERENCES [sec].[Role] ([Id]);
GO

-- sec.UserSession.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserSession_TenantId')
    ALTER TABLE [sec].[UserSession] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserSession_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.UserSession.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserSession_UserId')
    ALTER TABLE [sec].[UserSession] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserSession_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.UserSession.ActiveCompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserSession_ActiveCompanyCodeId')
    ALTER TABLE [sec].[UserSession] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserSession_ActiveCompanyCodeId] FOREIGN KEY ([ActiveCompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- sec.UserSubstitution.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserSubstitution_TenantId')
    ALTER TABLE [sec].[UserSubstitution] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserSubstitution_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- sec.UserSubstitution.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserSubstitution_UserId')
    ALTER TABLE [sec].[UserSubstitution] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserSubstitution_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.UserSubstitution.SubstituteUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserSubstitution_SubstituteUserId')
    ALTER TABLE [sec].[UserSubstitution] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserSubstitution_SubstituteUserId] FOREIGN KEY ([SubstituteUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- sec.UserSubstitution.ScopeRoleId -> sec.Role (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserSubstitution_ScopeRoleId')
    ALTER TABLE [sec].[UserSubstitution] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserSubstitution_ScopeRoleId] FOREIGN KEY ([ScopeRoleId])
        REFERENCES [sec].[Role] ([Id]);
GO

-- sec.UserSubstitution.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserSubstitution_CompanyCodeId')
    ALTER TABLE [sec].[UserSubstitution] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserSubstitution_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- sec.UserSubstitution.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_sec_UserSubstitution_CurrencyCode')
    ALTER TABLE [sec].[UserSubstitution] WITH CHECK
        ADD CONSTRAINT [FK_sec_UserSubstitution_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- audit.AuditLog.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_audit_AuditLog_TenantId')
    ALTER TABLE [audit].[AuditLog] WITH CHECK
        ADD CONSTRAINT [FK_audit_AuditLog_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- audit.AuditLog.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_audit_AuditLog_UserId')
    ALTER TABLE [audit].[AuditLog] WITH CHECK
        ADD CONSTRAINT [FK_audit_AuditLog_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- audit.AuditLog.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_audit_AuditLog_CompanyCodeId')
    ALTER TABLE [audit].[AuditLog] WITH CHECK
        ADD CONSTRAINT [FK_audit_AuditLog_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- audit.ChangeDocumentHeader.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_audit_ChangeDocumentHeader_TenantId')
    ALTER TABLE [audit].[ChangeDocumentHeader] WITH CHECK
        ADD CONSTRAINT [FK_audit_ChangeDocumentHeader_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- audit.ChangeDocumentItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_audit_ChangeDocumentItem_TenantId')
    ALTER TABLE [audit].[ChangeDocumentItem] WITH CHECK
        ADD CONSTRAINT [FK_audit_ChangeDocumentItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- audit.ChangeDocumentItem.ChangeDocumentHeaderId -> audit.ChangeDocumentHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_audit_ChangeDocumentItem_ChangeDocumentHeaderId')
    ALTER TABLE [audit].[ChangeDocumentItem] WITH CHECK
        ADD CONSTRAINT [FK_audit_ChangeDocumentItem_ChangeDocumentHeaderId] FOREIGN KEY ([ChangeDocumentHeaderId])
        REFERENCES [audit].[ChangeDocumentHeader] ([Id]);
GO

-- audit.DataAccessLog.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_audit_DataAccessLog_TenantId')
    ALTER TABLE [audit].[DataAccessLog] WITH CHECK
        ADD CONSTRAINT [FK_audit_DataAccessLog_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- audit.DataAccessLog.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_audit_DataAccessLog_UserId')
    ALTER TABLE [audit].[DataAccessLog] WITH CHECK
        ADD CONSTRAINT [FK_audit_DataAccessLog_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- audit.RetentionPolicy.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_audit_RetentionPolicy_TenantId')
    ALTER TABLE [audit].[RetentionPolicy] WITH CHECK
        ADD CONSTRAINT [FK_audit_RetentionPolicy_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- rpt.ReportDefinition.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportDefinition_TenantId')
    ALTER TABLE [rpt].[ReportDefinition] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportDefinition_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- rpt.ReportDefinition.RequiredPermissionId -> sec.Permission (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportDefinition_RequiredPermissionId')
    ALTER TABLE [rpt].[ReportDefinition] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportDefinition_RequiredPermissionId] FOREIGN KEY ([RequiredPermissionId])
        REFERENCES [sec].[Permission] ([Id]);
GO

-- rpt.ReportExecutionLog.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportExecutionLog_TenantId')
    ALTER TABLE [rpt].[ReportExecutionLog] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportExecutionLog_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- rpt.ReportExecutionLog.ReportDefinitionId -> rpt.ReportDefinition (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportExecutionLog_ReportDefinitionId')
    ALTER TABLE [rpt].[ReportExecutionLog] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportExecutionLog_ReportDefinitionId] FOREIGN KEY ([ReportDefinitionId])
        REFERENCES [rpt].[ReportDefinition] ([Id]);
GO

-- rpt.ReportExecutionLog.UserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportExecutionLog_UserId')
    ALTER TABLE [rpt].[ReportExecutionLog] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportExecutionLog_UserId] FOREIGN KEY ([UserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- rpt.ReportLayout.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportLayout_TenantId')
    ALTER TABLE [rpt].[ReportLayout] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportLayout_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- rpt.ReportLayout.ReportDefinitionId -> rpt.ReportDefinition (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportLayout_ReportDefinitionId')
    ALTER TABLE [rpt].[ReportLayout] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportLayout_ReportDefinitionId] FOREIGN KEY ([ReportDefinitionId])
        REFERENCES [rpt].[ReportDefinition] ([Id]);
GO

-- rpt.ReportLayout.OwnerUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportLayout_OwnerUserId')
    ALTER TABLE [rpt].[ReportLayout] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportLayout_OwnerUserId] FOREIGN KEY ([OwnerUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- rpt.ReportParameter.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportParameter_TenantId')
    ALTER TABLE [rpt].[ReportParameter] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportParameter_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- rpt.ReportParameter.ReportDefinitionId -> rpt.ReportDefinition (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportParameter_ReportDefinitionId')
    ALTER TABLE [rpt].[ReportParameter] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportParameter_ReportDefinitionId] FOREIGN KEY ([ReportDefinitionId])
        REFERENCES [rpt].[ReportDefinition] ([Id]);
GO

-- rpt.ReportParameter.DataElementId -> cfg.DictionaryDataElement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportParameter_DataElementId')
    ALTER TABLE [rpt].[ReportParameter] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportParameter_DataElementId] FOREIGN KEY ([DataElementId])
        REFERENCES [cfg].[DictionaryDataElement] ([Id]);
GO

-- rpt.ReportParameter.SearchHelpId -> cfg.DictionarySearchHelp (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportParameter_SearchHelpId')
    ALTER TABLE [rpt].[ReportParameter] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportParameter_SearchHelpId] FOREIGN KEY ([SearchHelpId])
        REFERENCES [cfg].[DictionarySearchHelp] ([Id]);
GO

-- rpt.ReportVariant.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportVariant_TenantId')
    ALTER TABLE [rpt].[ReportVariant] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportVariant_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- rpt.ReportVariant.ReportDefinitionId -> rpt.ReportDefinition (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportVariant_ReportDefinitionId')
    ALTER TABLE [rpt].[ReportVariant] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportVariant_ReportDefinitionId] FOREIGN KEY ([ReportDefinitionId])
        REFERENCES [rpt].[ReportDefinition] ([Id]);
GO

-- rpt.ReportVariant.OwnerUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_rpt_ReportVariant_OwnerUserId')
    ALTER TABLE [rpt].[ReportVariant] WITH CHECK
        ADD CONSTRAINT [FK_rpt_ReportVariant_OwnerUserId] FOREIGN KEY ([OwnerUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- intg.ApiClient.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_ApiClient_TenantId')
    ALTER TABLE [intg].[ApiClient] WITH CHECK
        ADD CONSTRAINT [FK_intg_ApiClient_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.ApiClient.ServiceUserId -> sec.User (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_ApiClient_ServiceUserId')
    ALTER TABLE [intg].[ApiClient] WITH CHECK
        ADD CONSTRAINT [FK_intg_ApiClient_ServiceUserId] FOREIGN KEY ([ServiceUserId])
        REFERENCES [sec].[User] ([Id]);
GO

-- intg.BankStatement.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatement_TenantId')
    ALTER TABLE [intg].[BankStatement] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatement_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.BankStatement.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatement_CompanyCodeId')
    ALTER TABLE [intg].[BankStatement] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatement_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- intg.BankStatement.HouseBankId -> mdm.HouseBank (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatement_HouseBankId')
    ALTER TABLE [intg].[BankStatement] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatement_HouseBankId] FOREIGN KEY ([HouseBankId])
        REFERENCES [mdm].[HouseBank] ([Id]);
GO

-- intg.BankStatement.HouseBankAccountId -> mdm.HouseBankAccount (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatement_HouseBankAccountId')
    ALTER TABLE [intg].[BankStatement] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatement_HouseBankAccountId] FOREIGN KEY ([HouseBankAccountId])
        REFERENCES [mdm].[HouseBankAccount] ([Id]);
GO

-- intg.BankStatement.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatement_CurrencyCode')
    ALTER TABLE [intg].[BankStatement] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatement_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- intg.BankStatement.ImportJobId -> intg.ImportJob (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatement_ImportJobId')
    ALTER TABLE [intg].[BankStatement] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatement_ImportJobId] FOREIGN KEY ([ImportJobId])
        REFERENCES [intg].[ImportJob] ([Id]);
GO

-- intg.BankStatementItem.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatementItem_TenantId')
    ALTER TABLE [intg].[BankStatementItem] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatementItem_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.BankStatementItem.BankStatementId -> intg.BankStatement (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatementItem_BankStatementId')
    ALTER TABLE [intg].[BankStatementItem] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatementItem_BankStatementId] FOREIGN KEY ([BankStatementId])
        REFERENCES [intg].[BankStatement] ([Id]);
GO

-- intg.BankStatementItem.CurrencyCode -> cfg.Currency (business key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatementItem_CurrencyCode')
    ALTER TABLE [intg].[BankStatementItem] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatementItem_CurrencyCode] FOREIGN KEY ([TenantId], [CurrencyCode])
        REFERENCES [cfg].[Currency] ([TenantId], [CurrencyCode]);
GO

-- intg.BankStatementItem.MatchedBusinessPartnerId -> mdm.BusinessPartner (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatementItem_MatchedBusinessPartnerId')
    ALTER TABLE [intg].[BankStatementItem] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatementItem_MatchedBusinessPartnerId] FOREIGN KEY ([MatchedBusinessPartnerId])
        REFERENCES [mdm].[BusinessPartner] ([Id]);
GO

-- intg.BankStatementItem.MatchedOpenItemId -> fin.OpenItem (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatementItem_MatchedOpenItemId')
    ALTER TABLE [intg].[BankStatementItem] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatementItem_MatchedOpenItemId] FOREIGN KEY ([MatchedOpenItemId])
        REFERENCES [fin].[OpenItem] ([Id]);
GO

-- intg.BankStatementItem.JournalEntryHeaderId -> fin.JournalEntryHeader (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatementItem_JournalEntryHeaderId')
    ALTER TABLE [intg].[BankStatementItem] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatementItem_JournalEntryHeaderId] FOREIGN KEY ([JournalEntryHeaderId])
        REFERENCES [fin].[JournalEntryHeader] ([Id]);
GO

-- intg.BankStatementItem.ClearingDocumentId -> fin.ClearingDocument (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_BankStatementItem_ClearingDocumentId')
    ALTER TABLE [intg].[BankStatementItem] WITH CHECK
        ADD CONSTRAINT [FK_intg_BankStatementItem_ClearingDocumentId] FOREIGN KEY ([ClearingDocumentId])
        REFERENCES [fin].[ClearingDocument] ([Id]);
GO

-- intg.IdempotencyKey.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_IdempotencyKey_TenantId')
    ALTER TABLE [intg].[IdempotencyKey] WITH CHECK
        ADD CONSTRAINT [FK_intg_IdempotencyKey_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.IdempotencyKey.ApiClientId -> intg.ApiClient (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_IdempotencyKey_ApiClientId')
    ALTER TABLE [intg].[IdempotencyKey] WITH CHECK
        ADD CONSTRAINT [FK_intg_IdempotencyKey_ApiClientId] FOREIGN KEY ([ApiClientId])
        REFERENCES [intg].[ApiClient] ([Id]);
GO

-- intg.ImportJob.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_ImportJob_TenantId')
    ALTER TABLE [intg].[ImportJob] WITH CHECK
        ADD CONSTRAINT [FK_intg_ImportJob_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.ImportJob.CompanyCodeId -> org.CompanyCode (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_ImportJob_CompanyCodeId')
    ALTER TABLE [intg].[ImportJob] WITH CHECK
        ADD CONSTRAINT [FK_intg_ImportJob_CompanyCodeId] FOREIGN KEY ([CompanyCodeId])
        REFERENCES [org].[CompanyCode] ([Id]);
GO

-- intg.ImportJobError.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_ImportJobError_TenantId')
    ALTER TABLE [intg].[ImportJobError] WITH CHECK
        ADD CONSTRAINT [FK_intg_ImportJobError_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.ImportJobError.ImportJobId -> intg.ImportJob (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_ImportJobError_ImportJobId')
    ALTER TABLE [intg].[ImportJobError] WITH CHECK
        ADD CONSTRAINT [FK_intg_ImportJobError_ImportJobId] FOREIGN KEY ([ImportJobId])
        REFERENCES [intg].[ImportJob] ([Id]);
GO

-- intg.InboundMessage.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_InboundMessage_TenantId')
    ALTER TABLE [intg].[InboundMessage] WITH CHECK
        ADD CONSTRAINT [FK_intg_InboundMessage_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.InboundMessage.ApiClientId -> intg.ApiClient (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_InboundMessage_ApiClientId')
    ALTER TABLE [intg].[InboundMessage] WITH CHECK
        ADD CONSTRAINT [FK_intg_InboundMessage_ApiClientId] FOREIGN KEY ([ApiClientId])
        REFERENCES [intg].[ApiClient] ([Id]);
GO

-- intg.IntegrationEndpoint.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_IntegrationEndpoint_TenantId')
    ALTER TABLE [intg].[IntegrationEndpoint] WITH CHECK
        ADD CONSTRAINT [FK_intg_IntegrationEndpoint_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.OutboxMessage.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_OutboxMessage_TenantId')
    ALTER TABLE [intg].[OutboxMessage] WITH CHECK
        ADD CONSTRAINT [FK_intg_OutboxMessage_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.WebhookDelivery.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_WebhookDelivery_TenantId')
    ALTER TABLE [intg].[WebhookDelivery] WITH CHECK
        ADD CONSTRAINT [FK_intg_WebhookDelivery_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.WebhookDelivery.WebhookSubscriptionId -> intg.WebhookSubscription (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_WebhookDelivery_WebhookSubscriptionId')
    ALTER TABLE [intg].[WebhookDelivery] WITH CHECK
        ADD CONSTRAINT [FK_intg_WebhookDelivery_WebhookSubscriptionId] FOREIGN KEY ([WebhookSubscriptionId])
        REFERENCES [intg].[WebhookSubscription] ([Id]);
GO

-- intg.WebhookDelivery.OutboxMessageId -> intg.OutboxMessage (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_WebhookDelivery_OutboxMessageId')
    ALTER TABLE [intg].[WebhookDelivery] WITH CHECK
        ADD CONSTRAINT [FK_intg_WebhookDelivery_OutboxMessageId] FOREIGN KEY ([OutboxMessageId])
        REFERENCES [intg].[OutboxMessage] ([Id]);
GO

-- intg.WebhookSubscription.TenantId -> org.Tenant (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_WebhookSubscription_TenantId')
    ALTER TABLE [intg].[WebhookSubscription] WITH CHECK
        ADD CONSTRAINT [FK_intg_WebhookSubscription_TenantId] FOREIGN KEY ([TenantId])
        REFERENCES [org].[Tenant] ([Id]);
GO

-- intg.WebhookSubscription.ApiClientId -> intg.ApiClient (surrogate key)
IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'FK_intg_WebhookSubscription_ApiClientId')
    ALTER TABLE [intg].[WebhookSubscription] WITH CHECK
        ADD CONSTRAINT [FK_intg_WebhookSubscription_ApiClientId] FOREIGN KEY ([ApiClientId])
        REFERENCES [intg].[ApiClient] ([Id]);
GO

/* References the application enforces instead of the database:
     org.CompanyCode.TaxJurisdictionSchemaId - jurisdiction schema key, not a jurisdiction row
     org.Tenant.DefaultCurrencyCode - cfg.Currency is tenant dependent, org.Tenant is not
     fin.JournalEntryLine.ProjectId - project / WBS element belongs to a future module
     co.SettlementDocument.SenderObjectId - polymorphic: qualified by SenderObjectType
     co.SettlementRule.SenderObjectId - polymorphic: qualified by SenderObjectType
 */
