using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;

namespace ErpS4.Application.BusinessPartners;

/// <inheritdoc />
public sealed class BusinessPartnerSyncService(
    IErpDataContext context,
    ITenantProvider tenantProvider,
    ICurrentUser currentUser,
    TimeProvider timeProvider,
    ILogger<BusinessPartnerSyncService> logger) : IBusinessPartnerSyncService
{
    private static readonly DateOnly OpenEnded = new(9999, 12, 31);

    private int TenantId => tenantProvider.TenantId;

    public async Task<RoleAssignmentResult> AssignRoleAsync(
        AssignRoleRequest request,
        CancellationToken cancellationToken = default)
    {
        var violations = new List<RuleViolation>();

        var partner = await context.Query<BusinessPartner>()
            .FirstOrDefaultAsync(
                p => p.TenantId == TenantId && p.PartnerNumber == request.PartnerNumber,
                cancellationToken);

        if (partner is null)
        {
            return Rejected(request, new RuleViolation(
                BusinessPartnerErrorCodes.PartnerUnknown,
                $"Business partner {request.PartnerNumber} does not exist.",
                nameof(request.PartnerNumber)));
        }

        if (partner.Status != "Active")
        {
            return Rejected(request, new RuleViolation(
                BusinessPartnerErrorCodes.PartnerNotActive,
                $"Business partner {request.PartnerNumber} is {partner.Status.ToLowerInvariant()}.",
                nameof(request.PartnerNumber)));
        }

        var role = await context.Query<BusinessPartnerRole>()
            .FirstOrDefaultAsync(
                r => r.TenantId == TenantId && r.RoleCode == request.RoleCode,
                cancellationToken);

        if (role is null)
        {
            return Rejected(request, new RuleViolation(
                BusinessPartnerErrorCodes.RoleUnknown,
                $"Role {request.RoleCode} does not exist.",
                nameof(request.RoleCode)));
        }

        var existing = await context.Query<BusinessPartnerRoleAssignment>()
            .FirstOrDefaultAsync(
                a => a.TenantId == TenantId
                     && a.BusinessPartnerId == partner.Id
                     && a.BusinessPartnerRoleId == role.Id,
                cancellationToken);

        var now = timeProvider.GetUtcNow().UtcDateTime;
        var user = currentUser.UserName;
        var created = new List<string>();

        var assignment = existing;
        if (assignment is null)
        {
            assignment = new BusinessPartnerRoleAssignment
            {
                TenantId = TenantId,
                BusinessPartnerId = partner.Id,
                BusinessPartnerRoleId = role.Id,
                IsSynchronized = false,
                SyncStatus = "Pending",
                ValidFrom = request.ValidFrom ?? DateOnly.FromDateTime(now),
                ValidTo = request.ValidTo ?? OpenEnded,
                CreatedAt = now,
                CreatedBy = user,
            };

            context.Add(assignment);
            await context.SaveChangesAsync(cancellationToken);
            created.Add(nameof(BusinessPartnerRoleAssignment));
        }

        // The identity is never touched here: only the data the role needs.
        var roleData = await EnsureRoleDataAsync(partner, role, request, now, user, cancellationToken);
        created.AddRange(roleData.Created);
        violations.AddRange(roleData.Violations);

        var blocking = violations.Any(v => v.IsBlocking);
        assignment.IsSynchronized = !blocking;
        assignment.SyncStatus = blocking ? "Failed" : "Completed";
        assignment.SynchronizedAt = now;
        assignment.SyncMessage = violations.Count == 0
            ? null
            : string.Join("; ", violations.Select(v => v.Message));
        assignment.ModifiedAt = now;
        assignment.ModifiedBy = user;

        await context.SaveChangesAsync(cancellationToken);

        logger.LogInformation(
            "Role {Role} on partner {Partner}: created {Created}, status {Status}",
            role.RoleCode, partner.PartnerNumber, created.Count, assignment.SyncStatus);

        return new RoleAssignmentResult
        {
            PartnerNumber = partner.PartnerNumber,
            RoleCode = role.RoleCode,
            WasAlreadyAssigned = existing is not null && roleData.Created.Count == 0,
            Created = created,
            SyncStatus = assignment.SyncStatus,
            Violations = violations,
        };
    }

    public async Task<SynchronizationResult> SynchronizeAsync(
        string partnerNumber,
        CancellationToken cancellationToken = default)
    {
        var roles = await (
            from assignment in context.Query<BusinessPartnerRoleAssignment>()
            join partner in context.Query<BusinessPartner>()
                on assignment.BusinessPartnerId equals partner.Id
            join role in context.Query<BusinessPartnerRole>()
                on assignment.BusinessPartnerRoleId equals role.Id
            where assignment.TenantId == TenantId && partner.PartnerNumber == partnerNumber
            select role.RoleCode
        ).ToListAsync(cancellationToken);

        var results = new List<RoleAssignmentResult>(roles.Count);
        foreach (var roleCode in roles)
        {
            results.Add(await AssignRoleAsync(
                new AssignRoleRequest(partnerNumber, roleCode), cancellationToken));
        }

        return new SynchronizationResult { PartnerNumber = partnerNumber, Roles = results };
    }

    public async Task<ConsistencyReport> CheckAsync(
        string partnerNumber,
        CancellationToken cancellationToken = default)
    {
        var findings = new List<RuleViolation>();

        var partner = await context.Query<BusinessPartner>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                p => p.TenantId == TenantId && p.PartnerNumber == partnerNumber,
                cancellationToken);

        if (partner is null)
        {
            return new ConsistencyReport
            {
                PartnerNumber = partnerNumber,
                Findings = [
                    new RuleViolation(
                        BusinessPartnerErrorCodes.PartnerUnknown,
                        $"Business partner {partnerNumber} does not exist."),
                ],
            };
        }

        var roleCategories = await (
            from assignment in context.Query<BusinessPartnerRoleAssignment>()
            join role in context.Query<BusinessPartnerRole>()
                on assignment.BusinessPartnerRoleId equals role.Id
            where assignment.TenantId == TenantId && assignment.BusinessPartnerId == partner.Id
            select role.RoleCategory
        ).ToListAsync(cancellationToken);

        var customer = await context.Query<BusinessPartnerCustomer>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                c => c.TenantId == TenantId && c.BusinessPartnerId == partner.Id,
                cancellationToken);

        var vendor = await context.Query<BusinessPartnerVendor>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                v => v.TenantId == TenantId && v.BusinessPartnerId == partner.Id,
                cancellationToken);

        var hasCustomerRole = roleCategories.Contains("Customer") || roleCategories.Contains("FICustomer");
        var hasVendorRole = roleCategories.Contains("Vendor") || roleCategories.Contains("FIVendor");

        if (hasCustomerRole && customer is null)
        {
            findings.Add(new RuleViolation(
                BusinessPartnerErrorCodes.RoleDataMissing,
                "The partner holds a customer role but has no customer data. Run BP_SYNC.",
                nameof(BusinessPartnerCustomer)));
        }

        if (hasVendorRole && vendor is null)
        {
            findings.Add(new RuleViolation(
                BusinessPartnerErrorCodes.RoleDataMissing,
                "The partner holds a supplier role but has no supplier data. Run BP_SYNC.",
                nameof(BusinessPartnerVendor)));
        }

        if (customer is not null && !hasCustomerRole)
        {
            findings.Add(new RuleViolation(
                BusinessPartnerErrorCodes.RoleMissingForData,
                "Customer data exists without a customer role; the role was probably removed.",
                nameof(BusinessPartnerCustomer),
                ViolationSeverity.Warning));
        }

        if (vendor is not null && !hasVendorRole)
        {
            findings.Add(new RuleViolation(
                BusinessPartnerErrorCodes.RoleMissingForData,
                "Supplier data exists without a supplier role; the role was probably removed.",
                nameof(BusinessPartnerVendor),
                ViolationSeverity.Warning));
        }

        // In this design the customer and supplier numbers are the partner
        // number: one identity, one number, whatever role it is playing.
        if (customer is not null && customer.CustomerNumber != partner.PartnerNumber)
        {
            findings.Add(new RuleViolation(
                BusinessPartnerErrorCodes.NumberMismatch,
                $"Customer number {customer.CustomerNumber} differs from partner number " +
                $"{partner.PartnerNumber}.",
                nameof(BusinessPartnerCustomer),
                ViolationSeverity.Warning));
        }

        if (vendor is not null && vendor.VendorNumber != partner.PartnerNumber)
        {
            findings.Add(new RuleViolation(
                BusinessPartnerErrorCodes.NumberMismatch,
                $"Supplier number {vendor.VendorNumber} differs from partner number " +
                $"{partner.PartnerNumber}.",
                nameof(BusinessPartnerVendor),
                ViolationSeverity.Warning));
        }

        var segments = await (
            from segment in context.Query<BusinessPartnerCompanyCode>().AsNoTracking()
            join company in context.Query<CompanyCode>() on segment.CompanyCodeId equals company.Id
            join account in context.Query<GLAccount>()
                on segment.ReconciliationGLAccountId equals account.Id
            where segment.TenantId == TenantId && segment.BusinessPartnerId == partner.Id
            select new
            {
                segment.RoleCategory,
                company.CompanyCodeKey,
                account.GLAccountCode,
                account.IsReconciliationAccount,
                account.ReconciliationAccountType,
                segment.IsPostingBlocked,
            }).ToListAsync(cancellationToken);

        foreach (var segment in segments)
        {
            var expectedType = segment.RoleCategory == "Customer" ? "D" : "K";

            if (!segment.IsReconciliationAccount)
            {
                findings.Add(new RuleViolation(
                    BusinessPartnerErrorCodes.ReconciliationAccountWrongType,
                    $"Account {segment.GLAccountCode} in {segment.CompanyCodeKey} is not a " +
                    "reconciliation account; postings to this partner will be refused.",
                    nameof(BusinessPartnerCompanyCode)));
            }
            else if (segment.ReconciliationAccountType != expectedType)
            {
                findings.Add(new RuleViolation(
                    BusinessPartnerErrorCodes.ReconciliationAccountWrongType,
                    $"Account {segment.GLAccountCode} in {segment.CompanyCodeKey} reconciles " +
                    $"{segment.ReconciliationAccountType} accounts, but the segment is " +
                    $"{segment.RoleCategory}.",
                    nameof(BusinessPartnerCompanyCode)));
            }

            var hasRole = segment.RoleCategory == "Customer" ? hasCustomerRole : hasVendorRole;
            if (!hasRole)
            {
                findings.Add(new RuleViolation(
                    BusinessPartnerErrorCodes.SegmentWithoutRole,
                    $"Company code {segment.CompanyCodeKey} has {segment.RoleCategory} data, " +
                    "but the partner does not hold that role.",
                    nameof(BusinessPartnerCompanyCode),
                    ViolationSeverity.Warning));
            }

            if (partner.IsCentralBlocked && !segment.IsPostingBlocked)
            {
                findings.Add(new RuleViolation(
                    BusinessPartnerErrorCodes.BlockInconsistent,
                    $"The partner is blocked centrally, but {segment.CompanyCodeKey} is not - " +
                    "the central block already stops postings, so this is informational.",
                    nameof(BusinessPartnerCompanyCode),
                    ViolationSeverity.Information));
            }
        }

        return new ConsistencyReport { PartnerNumber = partnerNumber, Findings = findings };
    }

    /// <summary>
    /// Creates whatever the role needs and nothing else. Called by both
    /// AssignRole and Synchronize, so a repair run and a first assignment take
    /// exactly the same path.
    /// </summary>
    private async Task<(List<string> Created, List<RuleViolation> Violations)> EnsureRoleDataAsync(
        BusinessPartner partner,
        BusinessPartnerRole role,
        AssignRoleRequest request,
        DateTime now,
        string user,
        CancellationToken cancellationToken)
    {
        var created = new List<string>();
        var violations = new List<RuleViolation>();

        switch (role.RoleCategory)
        {
            case "Customer":
                if (await EnsureCustomerAsync(partner, request, now, user, violations, cancellationToken))
                {
                    created.Add(nameof(BusinessPartnerCustomer));
                }

                break;

            case "Vendor":
                if (await EnsureVendorAsync(partner, request, now, user, violations, cancellationToken))
                {
                    created.Add(nameof(BusinessPartnerVendor));
                }

                break;

            case "FICustomer":
            case "FIVendor":
                var roleCategory = role.RoleCategory == "FICustomer" ? "Customer" : "Vendor";
                if (await EnsureCompanyCodeSegmentAsync(
                        partner, roleCategory, request, now, user, violations, cancellationToken))
                {
                    created.Add(nameof(BusinessPartnerCompanyCode));
                }

                break;
        }

        return (created, violations);
    }

    private async Task<bool> EnsureCustomerAsync(
        BusinessPartner partner,
        AssignRoleRequest request,
        DateTime now,
        string user,
        List<RuleViolation> violations,
        CancellationToken cancellationToken)
    {
        var existing = await context.Query<BusinessPartnerCustomer>()
            .FirstOrDefaultAsync(
                c => c.TenantId == TenantId && c.BusinessPartnerId == partner.Id,
                cancellationToken);

        if (existing is not null)
        {
            return false;
        }

        var accountGroup = await ResolveAccountGroupAsync(
            request.AccountGroup, "Customer", violations, cancellationToken);

        if (accountGroup is null)
        {
            return false;
        }

        context.Add(new BusinessPartnerCustomer
        {
            TenantId = TenantId,
            BusinessPartnerId = partner.Id,

            // One identity, one number: the customer number is the partner
            // number, which is what makes the dual role unambiguous.
            CustomerNumber = partner.PartnerNumber,
            CustomerAccountGroupId = accountGroup.Id,
            CreatedAt = now,
            CreatedBy = user,
        });

        return true;
    }

    private async Task<bool> EnsureVendorAsync(
        BusinessPartner partner,
        AssignRoleRequest request,
        DateTime now,
        string user,
        List<RuleViolation> violations,
        CancellationToken cancellationToken)
    {
        var existing = await context.Query<BusinessPartnerVendor>()
            .FirstOrDefaultAsync(
                v => v.TenantId == TenantId && v.BusinessPartnerId == partner.Id,
                cancellationToken);

        if (existing is not null)
        {
            return false;
        }

        var accountGroup = await ResolveAccountGroupAsync(
            request.AccountGroup, "Vendor", violations, cancellationToken);

        if (accountGroup is null)
        {
            return false;
        }

        context.Add(new BusinessPartnerVendor
        {
            TenantId = TenantId,
            BusinessPartnerId = partner.Id,
            VendorNumber = partner.PartnerNumber,
            VendorAccountGroupId = accountGroup.Id,
            CreatedAt = now,
            CreatedBy = user,
        });

        return true;
    }

    private async Task<bool> EnsureCompanyCodeSegmentAsync(
        BusinessPartner partner,
        string roleCategory,
        AssignRoleRequest request,
        DateTime now,
        string user,
        List<RuleViolation> violations,
        CancellationToken cancellationToken)
    {
        if (request.CompanyCode is null)
        {
            violations.Add(new RuleViolation(
                BusinessPartnerErrorCodes.CompanyCodeRequired,
                "This role needs company code data; supply a company code and a " +
                "reconciliation account.",
                nameof(request.CompanyCode)));
            return false;
        }

        var companyCode = await context.Query<CompanyCode>()
            .FirstOrDefaultAsync(
                c => c.TenantId == TenantId && c.CompanyCodeKey == request.CompanyCode,
                cancellationToken);

        if (companyCode is null)
        {
            violations.Add(new RuleViolation(
                BusinessPartnerErrorCodes.CompanyCodeUnknown,
                $"Company code {request.CompanyCode} does not exist.",
                nameof(request.CompanyCode)));
            return false;
        }

        var existing = await context.Query<BusinessPartnerCompanyCode>()
            .FirstOrDefaultAsync(
                s => s.TenantId == TenantId
                     && s.BusinessPartnerId == partner.Id
                     && s.CompanyCodeId == companyCode.Id
                     && s.RoleCategory == roleCategory,
                cancellationToken);

        if (existing is not null)
        {
            return false;
        }

        if (request.ReconciliationAccount is null)
        {
            violations.Add(new RuleViolation(
                BusinessPartnerErrorCodes.ReconciliationAccountRequired,
                "A company code segment needs a reconciliation account.",
                nameof(request.ReconciliationAccount)));
            return false;
        }

        var account = await context.Query<GLAccount>()
            .FirstOrDefaultAsync(
                a => a.TenantId == TenantId
                     && a.ChartOfAccountsId == companyCode.ChartOfAccountsId
                     && a.GLAccountCode == request.ReconciliationAccount,
                cancellationToken);

        if (account is null)
        {
            violations.Add(new RuleViolation(
                BusinessPartnerErrorCodes.ReconciliationAccountUnknown,
                $"Account {request.ReconciliationAccount} does not exist in the chart of " +
                $"accounts of {request.CompanyCode}.",
                nameof(request.ReconciliationAccount)));
            return false;
        }

        var expectedType = roleCategory == "Customer" ? "D" : "K";

        // The posting engine refuses a partner line whose account is not the
        // reconciliation account of that partner, so a wrong account here would
        // only surface as a failed posting weeks later.
        if (!account.IsReconciliationAccount || account.ReconciliationAccountType != expectedType)
        {
            violations.Add(new RuleViolation(
                BusinessPartnerErrorCodes.ReconciliationAccountWrongType,
                $"Account {request.ReconciliationAccount} is not a reconciliation account for " +
                $"{roleCategory.ToLowerInvariant()}s (expected type {expectedType}).",
                nameof(request.ReconciliationAccount)));
            return false;
        }

        long? paymentTermsId = null;
        if (request.PaymentTerms is not null)
        {
            paymentTermsId = await context.Query<PaymentTerms>()
                .Where(t => t.TenantId == TenantId && t.PaymentTermsCode == request.PaymentTerms)
                .Select(t => (long?)t.Id)
                .FirstOrDefaultAsync(cancellationToken);
        }

        context.Add(new BusinessPartnerCompanyCode
        {
            TenantId = TenantId,
            BusinessPartnerId = partner.Id,
            CompanyCodeId = companyCode.Id,
            RoleCategory = roleCategory,
            ReconciliationGLAccountId = account.Id,
            PaymentTermsId = paymentTermsId,
            SortKey = "001",
            CreatedAt = now,
            CreatedBy = user,
        });

        return true;
    }

    private async Task<AccountGroup?> ResolveAccountGroupAsync(
        string? accountGroupCode,
        string appliesTo,
        List<RuleViolation> violations,
        CancellationToken cancellationToken)
    {
        var query = context.Query<AccountGroup>()
            .Where(g => g.TenantId == TenantId && g.AppliesTo == appliesTo);

        if (accountGroupCode is not null)
        {
            query = query.Where(g => g.AccountGroupCode == accountGroupCode);
        }

        var group = await query.OrderBy(g => g.AccountGroupCode)
            .FirstOrDefaultAsync(cancellationToken);

        if (group is null)
        {
            violations.Add(new RuleViolation(
                BusinessPartnerErrorCodes.AccountGroupUnknown,
                accountGroupCode is null
                    ? $"No {appliesTo.ToLowerInvariant()} account group is configured."
                    : $"Account group {accountGroupCode} does not exist for " +
                      $"{appliesTo.ToLowerInvariant()}s.",
                nameof(AssignRoleRequest.AccountGroup)));
        }

        return group;
    }

    private static RoleAssignmentResult Rejected(AssignRoleRequest request, RuleViolation violation) =>
        new()
        {
            PartnerNumber = request.PartnerNumber,
            RoleCode = request.RoleCode,
            SyncStatus = "Failed",
            Violations = [violation],
        };
}
