using ErpS4.Api.Contracts;
using ErpS4.Api.Security;
using ErpS4.Application.BusinessPartners;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Api.Endpoints;

/// <summary>Business partner endpoints - BP, BP_ROLE, BP_SYNC, BP_CHECK.</summary>
public static class BusinessPartnerEndpoints
{
    private static readonly HashSet<string> SortableFields = new(StringComparer.OrdinalIgnoreCase)
    {
        "PartnerNumber", "FullName", "SearchTerm1",
    };

    public static IEndpointRouteBuilder MapBusinessPartnerEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/v1/business-partners")
            .WithTags("Business partners")
            .RequireAuthorization();

        group.MapGet("/", SearchAsync)
            .WithSummary("Search business partners (BP)")
            .RequireAuthorization(Policies.Permission(Policies.ReadBusinessPartner))
            .Produces<PagedResult<BusinessPartnerSummary>>();

        group.MapGet("/{partnerNumber}", GetAsync)
            .WithSummary("Display a business partner with its roles")
            .RequireAuthorization(Policies.Permission(Policies.ReadBusinessPartner))
            .Produces<BusinessPartnerDetail>()
            .ProducesProblem(StatusCodes.Status404NotFound);

        group.MapPost("/{partnerNumber}/roles", AssignRoleAsync)
            .WithSummary("Assign a role (BP_ROLE)")
            .WithDescription(
                "Creates the data the role needs - customer, supplier or company code " +
                "segment - without duplicating the identity.")
            .RequireAuthorization(Policies.Permission(Policies.UpdateBusinessPartner))
            .Produces<RoleAssignmentResult>()
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapPost("/{partnerNumber}/synchronize", SynchronizeAsync)
            .WithSummary("Re-run role synchronisation (BP_SYNC)")
            .RequireAuthorization(Policies.Permission(Policies.UpdateBusinessPartner))
            .Produces<SynchronizationResult>();

        group.MapGet("/{partnerNumber}/consistency-check", CheckAsync)
            .WithSummary("Check role and company code data (BP_CHECK)")
            .RequireAuthorization(Policies.Permission(Policies.ReadBusinessPartner))
            .Produces<ConsistencyReport>();

        return app;
    }

    private static async Task<IResult> AssignRoleAsync(
        string partnerNumber,
        [FromBody] AssignRoleBody body,
        IBusinessPartnerSyncService service,
        CancellationToken cancellationToken)
    {
        var result = await service.AssignRoleAsync(
            new AssignRoleRequest(
                partnerNumber,
                body.RoleCode,
                body.CompanyCode,
                body.ReconciliationAccount,
                body.AccountGroup,
                body.PaymentTerms,
                body.ValidFrom,
                body.ValidTo),
            cancellationToken);

        if (result.IsSuccess)
        {
            return Results.Ok(result);
        }

        // The violations are the useful part of the response, so they travel in
        // the problem document rather than being flattened into one message.
        return Results.Problem(
            title: "The role could not be assigned",
            detail: $"{result.Violations.Count} rule(s) refused this assignment.",
            statusCode: StatusCodes.Status422UnprocessableEntity,
            type: "https://errors.erps4.local/BP.ROLE_REJECTED",
            extensions: new Dictionary<string, object?>
            {
                ["errorCode"] = "BP.ROLE_REJECTED",
                ["errors"] = result.Violations
                    .Select(v => new { v.Code, v.Message, v.Field, Severity = v.Severity.ToString() })
                    .ToArray(),
            });
    }

    private static async Task<IResult> SynchronizeAsync(
        string partnerNumber,
        IBusinessPartnerSyncService service,
        CancellationToken cancellationToken) =>
        Results.Ok(await service.SynchronizeAsync(partnerNumber, cancellationToken));

    private static async Task<IResult> CheckAsync(
        string partnerNumber,
        IBusinessPartnerSyncService service,
        CancellationToken cancellationToken) =>
        Results.Ok(await service.CheckAsync(partnerNumber, cancellationToken));

    private static async Task<IResult> GetAsync(
        string partnerNumber,
        IErpDataContext context,
        ITenantProvider tenantProvider,
        CancellationToken cancellationToken)
    {
        var tenantId = tenantProvider.TenantId;

        var partner = await context.Query<BusinessPartner>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                p => p.TenantId == tenantId && p.PartnerNumber == partnerNumber,
                cancellationToken);

        if (partner is null)
        {
            return Results.NotFound();
        }

        var roles = await (
            from assignment in context.Query<BusinessPartnerRoleAssignment>().AsNoTracking()
            join role in context.Query<BusinessPartnerRole>()
                on assignment.BusinessPartnerRoleId equals role.Id
            where assignment.TenantId == tenantId && assignment.BusinessPartnerId == partner.Id
            orderby role.DisplayOrder
            select new BusinessPartnerRoleSummary(
                role.RoleCode,
                role.Name,
                role.RoleCategory,
                assignment.SyncStatus,
                assignment.ValidFrom,
                assignment.ValidTo)
        ).ToListAsync(cancellationToken);

        var segments = await (
            from segment in context.Query<BusinessPartnerCompanyCode>().AsNoTracking()
            join company in context.Query<CompanyCode>() on segment.CompanyCodeId equals company.Id
            join account in context.Query<GLAccount>()
                on segment.ReconciliationGLAccountId equals account.Id
            where segment.TenantId == tenantId && segment.BusinessPartnerId == partner.Id
            select new BusinessPartnerCompanyCodeSummary(
                company.CompanyCodeKey,
                segment.RoleCategory,
                account.GLAccountCode,
                segment.IsPostingBlocked)
        ).ToListAsync(cancellationToken);

        return Results.Ok(new BusinessPartnerDetail(
            partner.PartnerNumber,
            partner.PartnerCategory,
            partner.FullName,
            partner.SearchTerm1,
            partner.Status,
            partner.IsCentralBlocked,
            partner.IsIntercompany,
            roles,
            segments));
    }

    private static async Task<IResult> SearchAsync(
        [AsParameters] BusinessPartnerQuery query,
        IErpDataContext context,
        ITenantProvider tenantProvider,
        CancellationToken cancellationToken)
    {
        var tenantId = tenantProvider.TenantId;
        var request = new PageRequest
        {
            PageNumber = query.PageNumber ?? 1,
            PageSize = query.PageSize ?? PageRequest.DefaultPageSize,
            SortBy = query.SortBy,
            SortDescending = query.SortDescending ?? false,
            IncludeTotalCount = query.IncludeTotalCount ?? false,
        };

        if (request.SortBy is not null && !SortableFields.Contains(request.SortBy))
        {
            throw new ArgumentException(
                $"Cannot sort by '{request.SortBy}'. Allowed: " +
                string.Join(", ", SortableFields.Order()));
        }

        var partners = context.Query<BusinessPartner>().AsNoTracking()
            .Where(p => p.TenantId == tenantId);

        if (!string.IsNullOrWhiteSpace(query.Search))
        {
            var term = query.Search.Trim();
            partners = partners.Where(p =>
                p.PartnerNumber.Contains(term)
                || p.FullName.Contains(term)
                || (p.SearchTerm1 != null && p.SearchTerm1.Contains(term)));
        }

        if (query.RoleCode is not null)
        {
            partners =
                from p in partners
                join assignment in context.Query<BusinessPartnerRoleAssignment>()
                    on p.Id equals assignment.BusinessPartnerId
                join role in context.Query<BusinessPartnerRole>()
                    on assignment.BusinessPartnerRoleId equals role.Id
                where role.RoleCode == query.RoleCode
                select p;
        }

        var totalCount = request.IncludeTotalCount
            ? await partners.CountAsync(cancellationToken)
            : (int?)null;

        var sorted = (request.SortBy?.ToUpperInvariant(), request.SortDescending) switch
        {
            ("FULLNAME", false) => partners.OrderBy(p => p.FullName).ThenBy(p => p.Id),
            ("FULLNAME", true) => partners.OrderByDescending(p => p.FullName).ThenBy(p => p.Id),
            ("SEARCHTERM1", false) => partners.OrderBy(p => p.SearchTerm1).ThenBy(p => p.Id),
            ("SEARCHTERM1", true) => partners.OrderByDescending(p => p.SearchTerm1).ThenBy(p => p.Id),
            (_, true) => partners.OrderByDescending(p => p.PartnerNumber).ThenBy(p => p.Id),
            _ => partners.OrderBy(p => p.PartnerNumber).ThenBy(p => p.Id),
        };

        var page = await sorted
            .Skip(request.Skip)
            .Take(request.PageSize + 1)
            .Select(p => new BusinessPartnerSummary(
                p.PartnerNumber, p.FullName, p.PartnerCategory, p.SearchTerm1, p.Status))
            .ToListAsync(cancellationToken);

        var hasNextPage = page.Count > request.PageSize;
        if (hasNextPage)
        {
            page.RemoveAt(page.Count - 1);
        }

        return Results.Ok(PagedResult<BusinessPartnerSummary>.Create(
            page, request, totalCount, hasNextPage));
    }
}

public sealed record AssignRoleBody(
    string RoleCode,
    string? CompanyCode = null,
    string? ReconciliationAccount = null,
    string? AccountGroup = null,
    string? PaymentTerms = null,
    DateOnly? ValidFrom = null,
    DateOnly? ValidTo = null);

public sealed record BusinessPartnerQuery
{
    [FromQuery] public string? Search { get; init; }

    [FromQuery] public string? RoleCode { get; init; }

    [FromQuery] public int? PageNumber { get; init; }

    [FromQuery] public int? PageSize { get; init; }

    [FromQuery] public string? SortBy { get; init; }

    [FromQuery] public bool? SortDescending { get; init; }

    [FromQuery] public bool? IncludeTotalCount { get; init; }
}

public sealed record BusinessPartnerSummary(
    string PartnerNumber,
    string FullName,
    string PartnerCategory,
    string? SearchTerm1,
    string Status);

public sealed record BusinessPartnerDetail(
    string PartnerNumber,
    string PartnerCategory,
    string FullName,
    string? SearchTerm1,
    string Status,
    bool IsCentralBlocked,
    bool IsIntercompany,
    IReadOnlyList<BusinessPartnerRoleSummary> Roles,
    IReadOnlyList<BusinessPartnerCompanyCodeSummary> CompanyCodes);

public sealed record BusinessPartnerRoleSummary(
    string RoleCode,
    string Name,
    string RoleCategory,
    string SyncStatus,
    DateOnly ValidFrom,
    DateOnly ValidTo);

public sealed record BusinessPartnerCompanyCodeSummary(
    string CompanyCode,
    string RoleCategory,
    string ReconciliationAccount,
    bool IsPostingBlocked);
