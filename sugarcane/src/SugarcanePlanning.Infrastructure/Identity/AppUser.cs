using Microsoft.AspNetCore.Identity;

namespace SugarcanePlanning.Infrastructure.Identity;

/// <summary>Application user; carries the company that scopes every request (section 21).</summary>
public class AppUser : IdentityUser
{
    public string FullName { get; set; } = string.Empty;
    public int CompanyId { get; set; }
    public bool IsActive { get; set; } = true;
    public DateTime CreatedAtUtc { get; set; } = DateTime.UtcNow;
    public DateTime? LastLoginUtc { get; set; }
}

/// <summary>Role with a human-readable description for the administration screen.</summary>
public class AppRole : IdentityRole
{
    public AppRole() { }
    public AppRole(string name) : base(name) { }
    public string? Description { get; set; }
}
