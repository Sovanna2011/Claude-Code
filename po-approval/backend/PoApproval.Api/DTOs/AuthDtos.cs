using System.ComponentModel.DataAnnotations;

namespace PoApproval.Api.DTOs;

public class LoginRequest
{
    [Required] public string Username { get; set; } = default!;
    [Required] public string Password { get; set; } = default!;
}

public class LoginResponse
{
    public string Token { get; set; } = default!;
    public DateTime ExpiresAt { get; set; }
    public UserInfoDto User { get; set; } = default!;
}

public class UserInfoDto
{
    public int UserId { get; set; }
    public string Username { get; set; } = default!;
    public string DisplayName { get; set; } = default!;
    public string? Email { get; set; }
    public List<ReleaseCodeDto> ReleaseCodes { get; set; } = new();
}

public class ReleaseCodeDto
{
    public string Group { get; set; } = default!;      // FRGGR
    public string Code { get; set; } = default!;       // FRGCO
    public string? Text { get; set; }                  // FRGCT
}
