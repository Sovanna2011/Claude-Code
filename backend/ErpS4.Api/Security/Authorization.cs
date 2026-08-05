using System.Security.Claims;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.AspNetCore.Authorization;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Api.Security;

/// <summary>Claim types this API issues and reads.</summary>
public static class ErpClaims
{
    public const string TenantId = "erp:tenant";
    public const string UserName = "erp:user";
}

/// <summary>Tenant from the token. Nothing else may set it.</summary>
public sealed class HttpTenantProvider(IHttpContextAccessor accessor) : ITenantProvider
{
    public int TenantId =>
        int.TryParse(
            accessor.HttpContext?.User.FindFirstValue(ErpClaims.TenantId),
            out var tenantId)
            ? tenantId
            : throw new UnauthorizedAccessException("The token carries no tenant.");
}

/// <summary>Acting user for the audit columns.</summary>
public sealed class HttpCurrentUser(IHttpContextAccessor accessor) : ICurrentUser
{
    public string UserName =>
        accessor.HttpContext?.User.FindFirstValue(ErpClaims.UserName)
        ?? accessor.HttpContext?.User.Identity?.Name
        ?? "anonymous";
}

/// <summary>Requires one permission code, e.g. <c>Finance.JournalEntry.Post</c>.</summary>
public sealed class PermissionRequirement(string permission) : IAuthorizationRequirement
{
    public string Permission { get; } = permission;
}

/// <summary>
/// Resolves permissions from the security tables rather than from the token.
/// </summary>
/// <remarks>
/// Roles change while a token is alive, and a revoked role has to take effect
/// on the next request, not at the next login. The lookup is cached for the
/// request only.
/// </remarks>
public sealed class PermissionAuthorizationHandler(
    IErpDataContext context,
    ITenantProvider tenantProvider,
    ILogger<PermissionAuthorizationHandler> logger)
    : AuthorizationHandler<PermissionRequirement>
{
    private HashSet<string>? _permissions;

    protected override async Task HandleRequirementAsync(
        AuthorizationHandlerContext authorizationContext,
        PermissionRequirement requirement)
    {
        var userName = authorizationContext.User.FindFirstValue(ErpClaims.UserName);
        if (string.IsNullOrEmpty(userName))
        {
            return;   // unauthenticated: leave the requirement unmet
        }

        _permissions ??= await LoadAsync(userName);

        if (_permissions.Contains(requirement.Permission))
        {
            authorizationContext.Succeed(requirement);
        }
        else
        {
            logger.LogWarning(
                "User {User} lacks permission {Permission}", userName, requirement.Permission);
        }
    }

    private async Task<HashSet<string>> LoadAsync(string userName)
    {
        var tenantId = tenantProvider.TenantId;
        var today = DateOnly.FromDateTime(DateTime.UtcNow);

        var permissions = await (
            from user in context.Query<User>()
            join userRole in context.Query<UserRole>() on user.Id equals userRole.UserId
            join rolePermission in context.Query<RolePermission>()
                on userRole.RoleId equals rolePermission.RoleId
            join permission in context.Query<Permission>()
                on rolePermission.PermissionId equals permission.Id
            where user.TenantId == tenantId
                  && user.UserName == userName
                  && user.Status == "Active"
                  && !user.IsLocked
                  && user.ValidFrom <= today && user.ValidTo >= today
                  && userRole.ValidFrom <= today && userRole.ValidTo >= today
                  && rolePermission.IsGranted
            select permission.PermissionCode
        ).Distinct().ToListAsync();

        return permissions.ToHashSet(StringComparer.Ordinal);
    }
}

/// <summary>
/// Creates a policy per permission on demand, so an endpoint can ask for
/// <c>RequireAuthorization(Policies.Permission("Finance.JournalEntry.Post"))</c>
/// without every permission being registered at startup.
/// </summary>
public sealed class PermissionPolicyProvider(
    Microsoft.Extensions.Options.IOptions<AuthorizationOptions> options)
    : DefaultAuthorizationPolicyProvider(options)
{
    public const string Prefix = "perm:";

    public override async Task<AuthorizationPolicy?> GetPolicyAsync(string policyName)
    {
        if (!policyName.StartsWith(Prefix, StringComparison.Ordinal))
        {
            return await base.GetPolicyAsync(policyName);
        }

        return new AuthorizationPolicyBuilder()
            .RequireAuthenticatedUser()
            .AddRequirements(new PermissionRequirement(policyName[Prefix.Length..]))
            .Build();
    }
}

public static class Policies
{
    public static string Permission(string permissionCode) =>
        PermissionPolicyProvider.Prefix + permissionCode;

    public const string PostJournalEntry = "Finance.JournalEntry.Post";
    public const string CreateJournalEntry = "Finance.JournalEntry.Create";
    public const string ReverseJournalEntry = "Finance.JournalEntry.Reverse";
    public const string ReadFinanceReport = "Finance.Report.Read";
    public const string ReadBusinessPartner = "Master.BusinessPartner.Read";
    public const string UpdateBusinessPartner = "Master.BusinessPartner.Update";
}

/// <summary>
/// Checks organisational access - which company codes a user may work in, and
/// at what level. Endpoints call this once the request body is known, because
/// the company code is in the body, not in the route.
/// </summary>
public interface IOrganizationalAccessGuard
{
    /// <exception cref="UnauthorizedAccessException">Access is not granted.</exception>
    Task EnsureCompanyCodeAsync(
        string companyCode,
        string accessLevel,
        CancellationToken cancellationToken = default);
}

/// <inheritdoc />
public sealed class OrganizationalAccessGuard(
    IErpDataContext context,
    ITenantProvider tenantProvider,
    ICurrentUser currentUser) : IOrganizationalAccessGuard
{
    // Ordered from least to most: a user with Approve also has Post.
    private static readonly string[] Levels = ["Read", "Write", "Post", "Approve"];

    public async Task EnsureCompanyCodeAsync(
        string companyCode,
        string accessLevel,
        CancellationToken cancellationToken = default)
    {
        var tenantId = tenantProvider.TenantId;
        var today = DateOnly.FromDateTime(DateTime.UtcNow);

        var granted = await (
            from assignment in context.Query<UserCompanyCode>()
            join user in context.Query<User>() on assignment.UserId equals user.Id
            join company in context.Query<CompanyCode>()
                on assignment.CompanyCodeId equals company.Id
            where assignment.TenantId == tenantId
                  && user.UserName == currentUser.UserName
                  && company.CompanyCodeKey == companyCode
                  && assignment.ValidFrom <= today && assignment.ValidTo >= today
            select assignment.AccessLevel
        ).ToListAsync(cancellationToken);

        var required = Array.IndexOf(Levels, accessLevel);

        if (granted.Count == 0 || granted.Max(level => Array.IndexOf(Levels, level)) < required)
        {
            throw new UnauthorizedAccessException(
                $"User {currentUser.UserName} has no {accessLevel} access to company code " +
                $"{companyCode}.");
        }
    }
}
