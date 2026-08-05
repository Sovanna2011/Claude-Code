using ErpS4.Application.BusinessPartners;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.Extensions.Logging.Abstractions;

namespace ErpS4.Tests.TestDoubles;

/// <summary>
/// One company code, a customer and a supplier account group, the three
/// reconciliation accounts the tests need, the standard role catalogue and one
/// partner. Everything else a test needs, it sets itself.
/// </summary>
public sealed class BusinessPartnerScenario
{
    public const string CompanyCodeKey = "KH01";
    public const string PartnerNumber = "1000000003";
    public const string CustomerRecon = "120000";
    public const string VendorRecon = "200000";
    public const string PlainAccount = "400000";

    public BusinessPartnerScenario()
    {
        Context = new InMemoryDataContext();

        ChartOfAccounts = new ChartOfAccounts
        {
            TenantId = 1, ChartOfAccountsCode = "INT", Name = "International",
            MaintenanceLanguage = "EN", AccountNumberLength = 6, ChartType = "Operational",
        };
        Context.Seed(ChartOfAccounts);

        CompanyCode = new CompanyCode
        {
            TenantId = 1, CompanyCodeKey = CompanyCodeKey, CompanyId = 1, Name = "Test company",
            CountryCode = "KH", LocalCurrencyCode = "USD", LanguageCode = "EN",
            ChartOfAccountsId = ChartOfAccounts.Id, FiscalYearVariantId = 1,
            PostingPeriodVariantId = 1, FieldStatusVariantId = 1,
            ValidFrom = new DateOnly(2000, 1, 1), ValidTo = new DateOnly(9999, 12, 31),
        };
        Context.Seed(CompanyCode);

        CustomerGroup = new AccountGroup
        {
            TenantId = 1, ChartOfAccountsId = ChartOfAccounts.Id, AccountGroupCode = "CUST",
            Name = "Customers", FromAccount = "0000000001", ToAccount = "0000999999",
            AppliesTo = "Customer",
        };
        VendorGroup = new AccountGroup
        {
            TenantId = 1, ChartOfAccountsId = ChartOfAccounts.Id, AccountGroupCode = "VEND",
            Name = "Suppliers", FromAccount = "0001000000", ToAccount = "0001999999",
            AppliesTo = "Vendor",
        };
        Context.Seed(CustomerGroup, VendorGroup);

        CustomerReconAccount = SeedAccount(CustomerRecon, "D");
        VendorReconAccount = SeedAccount(VendorRecon, "K");
        RevenueAccount = SeedAccount(PlainAccount, null);

        Context.Seed(
            new BusinessPartnerRole { TenantId = 1, RoleCode = "000000", Name = "General", RoleCategory = "General", DisplayOrder = 10 },
            new BusinessPartnerRole { TenantId = 1, RoleCode = "FLCU00", Name = "Customer", RoleCategory = "Customer", DisplayOrder = 20 },
            new BusinessPartnerRole { TenantId = 1, RoleCode = "FLCU01", Name = "FI Customer", RoleCategory = "FICustomer", RequiresCompanyCodeData = true, DisplayOrder = 30 },
            new BusinessPartnerRole { TenantId = 1, RoleCode = "FLVN00", Name = "Supplier", RoleCategory = "Vendor", DisplayOrder = 40 },
            new BusinessPartnerRole { TenantId = 1, RoleCode = "FLVN01", Name = "FI Supplier", RoleCategory = "FIVendor", RequiresCompanyCodeData = true, DisplayOrder = 50 });

        Partner = new BusinessPartner
        {
            TenantId = 1, PartnerNumber = PartnerNumber, PartnerCategory = "2",
            BusinessPartnerGroupId = 1, FullName = "Sokha Trading Co., Ltd",
            Name1 = "Sokha Trading Co., Ltd", LanguageCode = "EN", Status = "Active",
            ValidFrom = new DateOnly(2000, 1, 1), ValidTo = new DateOnly(9999, 12, 31),
        };
        Context.Seed(Partner);

        Clock = new FixedTimeProvider(new DateTimeOffset(2026, 1, 20, 9, 0, 0, TimeSpan.Zero));
    }

    public InMemoryDataContext Context { get; }

    public ChartOfAccounts ChartOfAccounts { get; }

    public CompanyCode CompanyCode { get; }

    public AccountGroup CustomerGroup { get; }

    public AccountGroup VendorGroup { get; }

    public GLAccount CustomerReconAccount { get; }

    public GLAccount VendorReconAccount { get; }

    public GLAccount RevenueAccount { get; }

    public BusinessPartner Partner { get; }

    public FixedTimeProvider Clock { get; }

    public BusinessPartnerSyncService CreateService() =>
        new(Context,
            new FixedTenantProvider(1),
            new FixedCurrentUser("tester"),
            Clock,
            NullLogger<BusinessPartnerSyncService>.Instance);

    private GLAccount SeedAccount(string number, string? reconciliationType)
    {
        var account = new GLAccount
        {
            TenantId = 1,
            ChartOfAccountsId = ChartOfAccounts.Id,
            GLAccountCode = number,
            AccountGroupId = 1,
            AccountType = "BalanceSheet",
            IsReconciliationAccount = reconciliationType is not null,
            ReconciliationAccountType = reconciliationType,
        };

        Context.Seed(account);
        return account;
    }
}
