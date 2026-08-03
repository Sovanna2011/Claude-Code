namespace HRModule.Api.Models.Security;

/// <summary>Application role (simplified SAP authorization role).</summary>
public class AppRole
{
    public string RoleKey { get; set; } = string.Empty;   // HR_ADMIN, HR_MANAGER, EMPLOYEE
    public string RoleName { get; set; } = string.Empty;
}

/// <summary>Application user for web front-end authentication.</summary>
public class AppUser
{
    public int UserId { get; set; }
    public string Username { get; set; } = string.Empty;
    public string? DisplayName { get; set; }
    public string PasswordHash { get; set; } = string.Empty;  // base64 PBKDF2
    public string PasswordSalt { get; set; } = string.Empty;  // base64 salt
    public string RoleKey { get; set; } = string.Empty;
    public int? PERNR { get; set; }                            // linked employee
    public bool IsActive { get; set; } = true;
    public DateTime CreatedOn { get; set; }
    public DateTime? LastLogin { get; set; }
}
