using System.ComponentModel.DataAnnotations;

namespace HRModule.Api.DTOs;

public class LoginRequest
{
    [Required] public string Username { get; set; } = string.Empty;
    [Required] public string Password { get; set; } = string.Empty;
}

public class LoginResponse
{
    public string Token { get; set; } = string.Empty;
    public DateTime ExpiresUtc { get; set; }
    public UserInfoDto User { get; set; } = new();
}

/// <summary>Current user info (from /auth/me or login).</summary>
public class UserInfoDto
{
    public string Username { get; set; } = string.Empty;
    public string? DisplayName { get; set; }
    public string RoleKey { get; set; } = string.Empty;
    public string? RoleName { get; set; }
    public int? Pernr { get; set; }
}
