using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Text;
using HRModule.Api.Models.Security;
using Microsoft.IdentityModel.Tokens;

namespace HRModule.Api.Security;

/// <summary>Options bound from the "Jwt" configuration section.</summary>
public class JwtOptions
{
    public string Key { get; set; } = string.Empty;
    public string Issuer { get; set; } = "HRModule";
    public string Audience { get; set; } = "HRModuleClient";
    public int ExpiryMinutes { get; set; } = 480;
}

/// <summary>Issues signed JWTs carrying the user's role and linked PERNR.</summary>
public class JwtTokenService
{
    public const string PernrClaim = "pernr";
    private readonly JwtOptions _opt;

    public JwtTokenService(JwtOptions opt) => _opt = opt;

    public (string Token, DateTime ExpiresUtc) CreateToken(AppUser user)
    {
        var expires = DateTime.UtcNow.AddMinutes(_opt.ExpiryMinutes);
        var claims = new List<Claim>
        {
            new(JwtRegisteredClaimNames.Sub, user.Username),
            new(ClaimTypes.Name, user.DisplayName ?? user.Username),
            new(ClaimTypes.Role, user.RoleKey),
            new(JwtRegisteredClaimNames.Jti, Guid.NewGuid().ToString())
        };
        if (user.PERNR is int pernr)
            claims.Add(new Claim(PernrClaim, pernr.ToString()));

        var key = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(_opt.Key));
        var creds = new SigningCredentials(key, SecurityAlgorithms.HmacSha256);
        var token = new JwtSecurityToken(
            issuer: _opt.Issuer, audience: _opt.Audience,
            claims: claims, expires: expires, signingCredentials: creds);

        return (new JwtSecurityTokenHandler().WriteToken(token), expires);
    }
}
