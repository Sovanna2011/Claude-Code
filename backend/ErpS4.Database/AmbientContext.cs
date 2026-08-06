namespace ErpS4.Database;

/// <summary>Supplies the tenant the current request works in.</summary>
public interface ITenantProvider
{
    /// <summary>Tenant id, resolved from the token, host name or job context.</summary>
    int TenantId { get; }
}

/// <summary>Supplies the acting user for the audit columns.</summary>
public interface ICurrentUser
{
    /// <summary>Logon name, or a job name for background work.</summary>
    string UserName { get; }
}

/// <summary>Fixed tenant, for background jobs, tests and seeding.</summary>
public sealed class FixedTenantProvider(int tenantId) : ITenantProvider
{
    public int TenantId { get; } = tenantId;
}

/// <summary>Fixed user name, for background jobs, tests and seeding.</summary>
public sealed class FixedCurrentUser(string userName) : ICurrentUser
{
    public string UserName { get; } = userName;
}
