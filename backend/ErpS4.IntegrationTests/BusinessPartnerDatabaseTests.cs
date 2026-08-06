using ErpS4.Application.BusinessPartners;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using Xunit;

namespace ErpS4.IntegrationTests;

/// <summary>
/// Business Partner roles against a real server, on the seeded partners.
/// </summary>
/// <remarks>
/// The sample data gives 1000000001 the customer roles, 1000000002 the supplier
/// ones, 1000000003 both, and 1000000005 the intercompany pair. 1000000004 is
/// the employee partner and holds no company code segment, so it is the one
/// these tests extend.
/// </remarks>
[Collection(DatabaseCollection.Name)]
public sealed class BusinessPartnerDatabaseTests(DatabaseFixture fixture)
{
    private const string CompanyCode = "KH01";
    private const string Customer = "1000000001";
    private const string DualRole = "1000000003";
    private const string Employee = "1000000004";
    private const string Vendor = "1000000002";
    private const string CustomerReconciliation = "120000";

    [RequiresDatabaseFact]
    public async Task A_seeded_partner_is_consistent()
    {
        using var scope = fixture.CreateScope();

        var report = await scope.ServiceProvider
            .GetRequiredService<IBusinessPartnerSyncService>()
            .CheckAsync(Customer);

        Assert.Equal(Customer, report.PartnerNumber);
        Assert.Equal(0, report.ErrorCount);
    }

    [RequiresDatabaseFact]
    public async Task One_identity_holds_the_customer_and_the_supplier_role()
    {
        using var scope = fixture.CreateScope();
        var context = scope.ServiceProvider.GetRequiredService<ErpDbContext>();

        var partnerId = await context.Set<BusinessPartner>()
            .Where(p => p.PartnerNumber == DualRole)
            .Select(p => p.Id)
            .SingleAsync();

        var roles = await (
            from assignment in context.Set<BusinessPartnerRoleAssignment>()
            join role in context.Set<BusinessPartnerRole>()
                on assignment.BusinessPartnerRoleId equals role.Id
            where assignment.BusinessPartnerId == partnerId
            select role.RoleCode
        ).ToListAsync();

        // One partner number, both sides of the ledger - the whole point of
        // the Business Partner model over separate customer and vendor masters.
        Assert.Contains("FLCU01", roles);
        Assert.Contains("FLVN01", roles);

        var report = await scope.ServiceProvider
            .GetRequiredService<IBusinessPartnerSyncService>()
            .CheckAsync(DualRole);

        Assert.Equal(0, report.ErrorCount);
    }

    [RequiresDatabaseFact]
    public async Task Assigning_a_role_creates_only_what_that_role_needs()
    {
        using var scope = fixture.CreateScope();
        var service = scope.ServiceProvider.GetRequiredService<IBusinessPartnerSyncService>();

        var result = await service.AssignRoleAsync(new AssignRoleRequest(
            Employee, "FLCU01", CompanyCode, CustomerReconciliation, "KUNA"));

        Assert.True(result.IsSuccess,
            string.Join("; ", result.Violations.Select(v => v.ToString())));

        using var reader = fixture.CreateScope();
        var context = reader.ServiceProvider.GetRequiredService<ErpDbContext>();

        var partnerId = await context.Set<BusinessPartner>()
            .Where(p => p.PartnerNumber == Employee)
            .Select(p => p.Id)
            .SingleAsync();

        // The company code segment is what the role needed, and it points at
        // the reconciliation account that was asked for.
        var segment = await context.Set<BusinessPartnerCompanyCode>()
            .SingleAsync(s => s.BusinessPartnerId == partnerId);

        Assert.Equal(CustomerReconciliation, await context.Set<GLAccount>()
            .Where(a => a.Id == segment.ReconciliationGLAccountId)
            .Select(a => a.GLAccountCode)
            .SingleAsync());

        // Repeating it changes nothing: assignment is idempotent.
        using var again = fixture.CreateScope();
        var repeated = await again.ServiceProvider
            .GetRequiredService<IBusinessPartnerSyncService>()
            .AssignRoleAsync(new AssignRoleRequest(
                Employee, "FLCU01", CompanyCode, CustomerReconciliation, "KUNA"));

        Assert.True(repeated.IsSuccess);
        Assert.True(repeated.WasAlreadyAssigned);

        Assert.Equal(1, await context.Set<BusinessPartnerCompanyCode>()
            .CountAsync(s => s.BusinessPartnerId == partnerId));
    }

    [RequiresDatabaseFact]
    public async Task A_customer_role_is_refused_a_supplier_reconciliation_account()
    {
        using var scope = fixture.CreateScope();

        // 150000 is the asset reconciliation account, not a customer one, and
        // the partner has no customer segment yet - so the account is actually
        // used rather than found already in place. Catching it here is what
        // stops the failure surfacing months later as a posting nobody can
        // explain.
        var result = await scope.ServiceProvider
            .GetRequiredService<IBusinessPartnerSyncService>()
            .AssignRoleAsync(new AssignRoleRequest(
                Vendor, "FLCU01", CompanyCode, "150000", "KUNA"));

        Assert.False(result.IsSuccess);
    }

    [RequiresDatabaseFact]
    public async Task Synchronisation_reports_every_role_it_looked_at()
    {
        using var scope = fixture.CreateScope();

        var result = await scope.ServiceProvider
            .GetRequiredService<IBusinessPartnerSyncService>()
            .SynchronizeAsync(DualRole);

        Assert.Equal(DualRole, result.PartnerNumber);
        Assert.NotEmpty(result.Roles);
        Assert.True(result.IsSuccess,
            string.Join("; ", result.Roles.SelectMany(r => r.Violations)
                .Select(v => v.ToString())));
    }
}
