using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Text;
using Microsoft.IdentityModel.Tokens;
using SugarcanePlanning.Infrastructure.Identity;
using SugarcanePlanning.Infrastructure.Services;

namespace SugarcanePlanning.Api.Security;

/// <summary>JWT settings bound from <c>appsettings.json</c>.</summary>
public class JwtOptions
{
    public const string SectionName = "Jwt";
    public string Issuer { get; set; } = "SugarcanePlanning";
    public string Audience { get; set; } = "SugarcanePlanningClient";
    /// <summary>Signing key; override it in production via configuration or a secret store.</summary>
    public string SigningKey { get; set; } = "change-this-signing-key-to-at-least-32-characters";
    public int ExpiryMinutes { get; set; } = 480;
}

/// <summary>Issues the bearer tokens the Blazor client sends on every call.</summary>
public class JwtTokenService
{
    private readonly JwtOptions _options;

    public JwtTokenService(Microsoft.Extensions.Options.IOptions<JwtOptions> options) => _options = options.Value;

    public (string Token, DateTime ExpiresAtUtc) CreateToken(AppUser user, IEnumerable<string> roles)
    {
        var expires = DateTime.UtcNow.AddMinutes(_options.ExpiryMinutes);

        var claims = new List<Claim>
        {
            new(JwtRegisteredClaimNames.Sub, user.Id),
            new(JwtRegisteredClaimNames.Jti, Guid.NewGuid().ToString()),
            new(ClaimTypes.Name, user.UserName ?? string.Empty),
            new(ClaimTypes.NameIdentifier, user.Id),
            new(HttpCurrentUser.CompanyClaim, user.CompanyId.ToString()),
            new(HttpCurrentUser.FullNameClaim, user.FullName)
        };
        claims.AddRange(roles.Select(r => new Claim(ClaimTypes.Role, r)));

        var key = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(_options.SigningKey));
        var token = new JwtSecurityToken(
            issuer: _options.Issuer,
            audience: _options.Audience,
            claims: claims,
            notBefore: DateTime.UtcNow,
            expires: expires,
            signingCredentials: new SigningCredentials(key, SecurityAlgorithms.HmacSha256));

        return (new JwtSecurityTokenHandler().WriteToken(token), expires);
    }
}
