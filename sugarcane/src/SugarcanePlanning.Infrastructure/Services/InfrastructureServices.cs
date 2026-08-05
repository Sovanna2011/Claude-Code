using System.Security.Claims;
using Microsoft.AspNetCore.Http;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Infrastructure.Services;

/// <summary>System clock. Injected everywhere so date logic is testable.</summary>
public class SystemDateTimeProvider : IDateTimeProvider
{
    public DateTime UtcNow => DateTime.UtcNow;
    public DateOnly Today => DateOnly.FromDateTime(DateTime.UtcNow);
}

/// <summary>Reads the caller's identity, company and roles from the JWT claims on the request.</summary>
public class HttpCurrentUser : ICurrentUser
{
    /// <summary>Claim carrying the user's company; the tenant filter keys off it.</summary>
    public const string CompanyClaim = "company_id";
    public const string FullNameClaim = "full_name";

    private readonly IHttpContextAccessor _accessor;

    public HttpCurrentUser(IHttpContextAccessor accessor) => _accessor = accessor;

    private ClaimsPrincipal? Principal => _accessor.HttpContext?.User;

    public string UserName => Principal?.Identity?.Name
                              ?? Principal?.FindFirstValue(ClaimTypes.NameIdentifier)
                              ?? "anonymous";

    public string? FullName => Principal?.FindFirstValue(FullNameClaim);

    public int? CompanyId
    {
        get
        {
            var raw = Principal?.FindFirstValue(CompanyClaim);
            return int.TryParse(raw, out var id) ? id : null;
        }
    }

    public bool IsAuthenticated => Principal?.Identity?.IsAuthenticated == true;

    public IReadOnlyCollection<string> Roles =>
        Principal?.FindAll(ClaimTypes.Role).Select(c => c.Value).ToList() ?? new List<string>();

    public string? IpAddress => _accessor.HttpContext?.Connection.RemoteIpAddress?.ToString();

    public string? DeviceInfo
    {
        get
        {
            var agent = _accessor.HttpContext?.Request.Headers.UserAgent.ToString();
            return string.IsNullOrWhiteSpace(agent) ? null : agent[..Math.Min(agent.Length, 300)];
        }
    }

    public bool IsInRole(string role) => Principal?.IsInRole(role) == true;

    /// <summary>Evaluates the role map of section 21 without a round trip to the authorization service.</summary>
    public bool HasPolicy(string policy)
        => Policies.RoleMap.TryGetValue(policy, out var roles) && roles.Any(IsInRole);
}

/// <summary>Fixed identity used by seeding, migrations and background jobs.</summary>
public class SystemCurrentUser : ICurrentUser
{
    public SystemCurrentUser(int? companyId = null, string userName = "system")
    {
        CompanyId = companyId;
        UserName = userName;
    }

    public string UserName { get; }
    public string? FullName => "System";
    public int? CompanyId { get; }
    public bool IsAuthenticated => true;
    public IReadOnlyCollection<string> Roles { get; } = AppRoles.All;
    public string? IpAddress => null;
    public string? DeviceInfo => "system";
    public bool IsInRole(string role) => true;
    public bool HasPolicy(string policy) => true;
}

/// <summary>Writes workflow / export / login events straight to the audit table (section 22).</summary>
public class AuditService : IAuditService
{
    private readonly IAppDbContext _db;
    private readonly ICurrentUser _user;
    private readonly IDateTimeProvider _clock;

    public AuditService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock)
    {
        _db = db;
        _user = user;
        _clock = clock;
    }

    public async Task LogAsync(AuditAction action, string tableName, string recordId,
        string? oldValues = null, string? newValues = null, string? remarks = null, CancellationToken ct = default)
    {
        _db.AuditLogs.Add(new AuditLog
        {
            CompanyId = _user.CompanyId,
            UserName = _user.IsAuthenticated ? _user.UserName : "system",
            TimestampUtc = _clock.UtcNow,
            Action = action,
            TableName = tableName,
            RecordId = recordId,
            OldValues = oldValues,
            NewValues = newValues,
            Remarks = remarks,
            IpAddress = _user.IpAddress,
            DeviceInfo = _user.DeviceInfo
        });
        await _db.SaveChangesAsync(ct);
    }
}
