namespace HRModule.Api.Security;

/// <summary>Role keys and authorization policy names used across the API.</summary>
public static class Roles
{
    public const string Admin = "HR_ADMIN";
    public const string Manager = "HR_MANAGER";
    public const string Employee = "EMPLOYEE";
}

public static class Policies
{
    /// <summary>Full maintenance actions (hiring, master data, org).</summary>
    public const string AdminOnly = "AdminOnly";

    /// <summary>Time recording (absences, attendances) - admin or manager.</summary>
    public const string TimeKeepers = "TimeKeepers";

    /// <summary>Any authenticated user (all roles).</summary>
    public const string AllStaff = "AllStaff";
}
