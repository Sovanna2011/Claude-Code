namespace PoApproval.Api.Models;

/// <summary>
/// Application login account. Passwords are stored only as PBKDF2 hashes
/// (see Security/PasswordHasher). Release codes granted to the user reproduce
/// the SAP authorization object M_EINK_FRG (FRGGR / FRGCO).
/// </summary>
public class AppUser
{
    public int UserId { get; set; }
    public string UserName { get; set; } = default!;
    public string DisplayName { get; set; } = default!;
    public string? Email { get; set; }
    public string PwdHash { get; set; } = default!;
    public bool IsActive { get; set; } = true;
    public DateTime CreatedOn { get; set; }

    public List<AppUserReleaseCode> ReleaseCodes { get; set; } = new();
}

/// <summary>Release code granted to a user (auth object M_EINK_FRG).</summary>
public class AppUserReleaseCode
{
    public int UserId { get; set; }
    public string FRGGR { get; set; } = default!;
    public string FRGCO { get; set; } = default!;
}
