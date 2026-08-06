using ErpS4.Application.BusinessPartners;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using ErpS4.Tests.TestDoubles;
using Xunit;

namespace ErpS4.Tests;

/// <summary>
/// The central business partner rule: customer and supplier are roles on one
/// identity, never two records. These tests are what stops a future change
/// from quietly reintroducing separate customer and vendor masters.
/// </summary>
public sealed class BusinessPartnerSyncTests
{
    [Fact]
    public async Task Assigning_the_customer_role_creates_customer_data_on_the_same_identity()
    {
        var scenario = new BusinessPartnerScenario();

        var result = await scenario.CreateService().AssignRoleAsync(
            new AssignRoleRequest(BusinessPartnerScenario.PartnerNumber, "FLCU00"));

        Assert.True(result.IsSuccess);
        Assert.Equal("Completed", result.SyncStatus);
        Assert.Contains(nameof(BusinessPartnerCustomer), result.Created);

        // One partner, one customer record, and the same number on both.
        Assert.Single(scenario.Context.Set<BusinessPartner>());
        var customer = Assert.Single(scenario.Context.Set<BusinessPartnerCustomer>());
        Assert.Equal(scenario.Partner.Id, customer.BusinessPartnerId);
        Assert.Equal(BusinessPartnerScenario.PartnerNumber, customer.CustomerNumber);
    }

    [Fact]
    public async Task One_partner_can_be_customer_and_supplier_at_once()
    {
        var scenario = new BusinessPartnerScenario();
        var service = scenario.CreateService();

        await service.AssignRoleAsync(
            new AssignRoleRequest(BusinessPartnerScenario.PartnerNumber, "FLCU00"));
        await service.AssignRoleAsync(
            new AssignRoleRequest(BusinessPartnerScenario.PartnerNumber, "FLVN00"));

        Assert.Single(scenario.Context.Set<BusinessPartner>());
        Assert.Single(scenario.Context.Set<BusinessPartnerCustomer>());
        Assert.Single(scenario.Context.Set<BusinessPartnerVendor>());
        Assert.Equal(2, scenario.Context.Set<BusinessPartnerRoleAssignment>().Count);

        // The identity is untouched: the name is stored once, not twice.
        Assert.Equal("Sokha Trading Co., Ltd", scenario.Partner.FullName);
    }

    [Fact]
    public async Task Assigning_a_role_twice_changes_nothing()
    {
        var scenario = new BusinessPartnerScenario();
        var service = scenario.CreateService();
        var request = new AssignRoleRequest(BusinessPartnerScenario.PartnerNumber, "FLCU00");

        var first = await service.AssignRoleAsync(request);
        var second = await service.AssignRoleAsync(request);

        Assert.False(first.WasAlreadyAssigned);
        Assert.True(second.WasAlreadyAssigned);
        Assert.Empty(second.Created);
        Assert.Single(scenario.Context.Set<BusinessPartnerCustomer>());
        Assert.Single(scenario.Context.Set<BusinessPartnerRoleAssignment>());
    }

    [Fact]
    public async Task An_fi_role_without_company_code_data_is_reported_not_silently_skipped()
    {
        var scenario = new BusinessPartnerScenario();

        var result = await scenario.CreateService().AssignRoleAsync(
            new AssignRoleRequest(BusinessPartnerScenario.PartnerNumber, "FLCU01"));

        Assert.False(result.IsSuccess);
        Assert.Equal("Failed", result.SyncStatus);
        Assert.Contains(
            result.Violations,
            v => v.Code == BusinessPartnerErrorCodes.CompanyCodeRequired);
    }

    [Fact]
    public async Task An_fi_role_creates_the_company_code_segment()
    {
        var scenario = new BusinessPartnerScenario();

        var result = await scenario.CreateService().AssignRoleAsync(
            new AssignRoleRequest(
                BusinessPartnerScenario.PartnerNumber,
                "FLCU01",
                BusinessPartnerScenario.CompanyCodeKey,
                BusinessPartnerScenario.CustomerRecon));

        Assert.True(result.IsSuccess);
        var segment = Assert.Single(scenario.Context.Set<BusinessPartnerCompanyCode>());
        Assert.Equal("Customer", segment.RoleCategory);
        Assert.Equal(scenario.CustomerReconAccount.Id, segment.ReconciliationGLAccountId);
    }

    [Fact]
    public async Task The_reconciliation_account_has_to_match_the_role()
    {
        var scenario = new BusinessPartnerScenario();

        // 400000 is an ordinary account, and 200000 reconciles suppliers - both
        // would make every posting to this customer fail later.
        var plain = await scenario.CreateService().AssignRoleAsync(
            new AssignRoleRequest(
                BusinessPartnerScenario.PartnerNumber, "FLCU01",
                BusinessPartnerScenario.CompanyCodeKey, BusinessPartnerScenario.PlainAccount));

        var wrongSide = await scenario.CreateService().AssignRoleAsync(
            new AssignRoleRequest(
                BusinessPartnerScenario.PartnerNumber, "FLCU01",
                BusinessPartnerScenario.CompanyCodeKey, BusinessPartnerScenario.VendorRecon));

        Assert.Contains(
            plain.Violations,
            v => v.Code == BusinessPartnerErrorCodes.ReconciliationAccountWrongType);
        Assert.Contains(
            wrongSide.Violations,
            v => v.Code == BusinessPartnerErrorCodes.ReconciliationAccountWrongType);
        Assert.Empty(scenario.Context.Set<BusinessPartnerCompanyCode>());
    }

    [Fact]
    public async Task An_unknown_partner_or_role_is_refused()
    {
        var scenario = new BusinessPartnerScenario();
        var service = scenario.CreateService();

        var unknownPartner = await service.AssignRoleAsync(
            new AssignRoleRequest("9999999999", "FLCU00"));
        var unknownRole = await service.AssignRoleAsync(
            new AssignRoleRequest(BusinessPartnerScenario.PartnerNumber, "ZZZZZZ"));

        Assert.Contains(
            unknownPartner.Violations, v => v.Code == BusinessPartnerErrorCodes.PartnerUnknown);
        Assert.Contains(
            unknownRole.Violations, v => v.Code == BusinessPartnerErrorCodes.RoleUnknown);
    }

    [Fact]
    public async Task A_blocked_partner_receives_no_new_roles()
    {
        var scenario = new BusinessPartnerScenario();
        scenario.Partner.Status = "Blocked";

        var result = await scenario.CreateService().AssignRoleAsync(
            new AssignRoleRequest(BusinessPartnerScenario.PartnerNumber, "FLCU00"));

        Assert.False(result.IsSuccess);
        Assert.Contains(
            result.Violations, v => v.Code == BusinessPartnerErrorCodes.PartnerNotActive);
    }

    [Fact]
    public async Task Check_reports_a_role_whose_data_was_never_created()
    {
        var scenario = new BusinessPartnerScenario();

        // A role assignment written by an import, without the role data.
        scenario.Context.Seed(new BusinessPartnerRoleAssignment
        {
            TenantId = 1,
            BusinessPartnerId = scenario.Partner.Id,
            BusinessPartnerRoleId = scenario.Context.Set<BusinessPartnerRole>()
                .Single(r => r.RoleCode == "FLCU00").Id,
            SyncStatus = "Pending",
            ValidFrom = new DateOnly(2026, 1, 1),
            ValidTo = new DateOnly(9999, 12, 31),
        });

        var report = await scenario.CreateService()
            .CheckAsync(BusinessPartnerScenario.PartnerNumber);

        Assert.False(report.IsConsistent);
        Assert.Contains(
            report.Findings, f => f.Code == BusinessPartnerErrorCodes.RoleDataMissing);
    }

    [Fact]
    public async Task Check_reports_customer_data_left_behind_by_a_removed_role()
    {
        var scenario = new BusinessPartnerScenario();

        scenario.Context.Seed(new BusinessPartnerCustomer
        {
            TenantId = 1,
            BusinessPartnerId = scenario.Partner.Id,
            CustomerNumber = BusinessPartnerScenario.PartnerNumber,
            CustomerAccountGroupId = scenario.CustomerGroup.Id,
        });

        var report = await scenario.CreateService()
            .CheckAsync(BusinessPartnerScenario.PartnerNumber);

        var finding = Assert.Single(report.Findings);
        Assert.Equal(BusinessPartnerErrorCodes.RoleMissingForData, finding.Code);
        Assert.Equal(ViolationSeverity.Warning, finding.Severity);
    }

    [Fact]
    public async Task Check_is_quiet_when_the_partner_is_healthy()
    {
        var scenario = new BusinessPartnerScenario();
        var service = scenario.CreateService();

        await service.AssignRoleAsync(
            new AssignRoleRequest(BusinessPartnerScenario.PartnerNumber, "FLCU00"));
        await service.AssignRoleAsync(
            new AssignRoleRequest(
                BusinessPartnerScenario.PartnerNumber, "FLCU01",
                BusinessPartnerScenario.CompanyCodeKey, BusinessPartnerScenario.CustomerRecon));

        var report = await service.CheckAsync(BusinessPartnerScenario.PartnerNumber);

        Assert.True(report.IsConsistent);
        Assert.Equal(0, report.ErrorCount);
    }

    [Fact]
    public async Task Synchronize_repairs_role_data_that_is_missing()
    {
        var scenario = new BusinessPartnerScenario();

        scenario.Context.Seed(new BusinessPartnerRoleAssignment
        {
            TenantId = 1,
            BusinessPartnerId = scenario.Partner.Id,
            BusinessPartnerRoleId = scenario.Context.Set<BusinessPartnerRole>()
                .Single(r => r.RoleCode == "FLVN00").Id,
            SyncStatus = "Pending",
            ValidFrom = new DateOnly(2026, 1, 1),
            ValidTo = new DateOnly(9999, 12, 31),
        });

        var result = await scenario.CreateService()
            .SynchronizeAsync(BusinessPartnerScenario.PartnerNumber);

        Assert.True(result.IsSuccess);
        Assert.Single(scenario.Context.Set<BusinessPartnerVendor>());

        var report = await scenario.CreateService()
            .CheckAsync(BusinessPartnerScenario.PartnerNumber);
        Assert.True(report.IsConsistent);
    }

    [Fact]
    public async Task Check_reports_an_unknown_partner()
    {
        var scenario = new BusinessPartnerScenario();

        var report = await scenario.CreateService().CheckAsync("9999999999");

        Assert.False(report.IsConsistent);
        Assert.Contains(
            report.Findings, f => f.Code == BusinessPartnerErrorCodes.PartnerUnknown);
    }
}
