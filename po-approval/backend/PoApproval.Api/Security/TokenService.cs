using System.Security.Claims;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using PoApproval.Api.Models;

namespace PoApproval.Api.Security;

/// <summary>
/// Issues and validates stateless bearer tokens signed with HMAC-SHA256.
/// A token is  base64url(payload) "." base64url(signature)  where the payload
/// is a small JSON document { sub, uid, name, exp }. This keeps the module free
/// of external JWT dependencies while providing tamper-proof, expiring tokens
/// for the UI5 user-level authentication.
/// </summary>
public class TokenService
{
    public const string SchemeName = "PoToken";

    private readonly byte[] _key;
    private readonly int _lifetimeMinutes;

    public TokenService(IConfiguration config)
    {
        var secret = config["Auth:SigningKey"];
        if (string.IsNullOrWhiteSpace(secret) || secret.Length < 16)
            throw new InvalidOperationException(
                "Auth:SigningKey is missing or too short (>= 16 chars required).");
        _key = Encoding.UTF8.GetBytes(secret);
        _lifetimeMinutes = config.GetValue<int?>("Auth:TokenLifetimeMinutes") ?? 480;
    }

    public (string Token, DateTime ExpiresAt) Create(AppUser user)
    {
        var expires = DateTime.UtcNow.AddMinutes(_lifetimeMinutes);
        var payload = new TokenPayload
        {
            sub = user.UserName,
            uid = user.UserId,
            name = user.DisplayName,
            exp = new DateTimeOffset(expires).ToUnixTimeSeconds()
        };
        var json = JsonSerializer.SerializeToUtf8Bytes(payload);
        var body = Base64Url(json);
        var sig = Base64Url(Sign(body));
        return ($"{body}.{sig}", expires);
    }

    public ClaimsPrincipal? Validate(string token)
    {
        if (string.IsNullOrWhiteSpace(token)) return null;
        var parts = token.Split('.');
        if (parts.Length != 2) return null;

        var expectedSig = Base64Url(Sign(parts[0]));
        if (!CryptographicOperations.FixedTimeEquals(
                Encoding.UTF8.GetBytes(expectedSig), Encoding.UTF8.GetBytes(parts[1])))
            return null;

        TokenPayload? payload;
        try
        {
            payload = JsonSerializer.Deserialize<TokenPayload>(FromBase64Url(parts[0]));
        }
        catch (JsonException)
        {
            return null;
        }
        if (payload is null) return null;
        if (DateTimeOffset.FromUnixTimeSeconds(payload.exp) < DateTimeOffset.UtcNow) return null;

        var identity = new ClaimsIdentity(SchemeName);
        identity.AddClaim(new Claim(ClaimTypes.NameIdentifier, payload.uid.ToString()));
        identity.AddClaim(new Claim(ClaimTypes.Name, payload.sub));
        identity.AddClaim(new Claim("displayName", payload.name ?? payload.sub));
        return new ClaimsPrincipal(identity);
    }

    private byte[] Sign(string body)
    {
        using var hmac = new HMACSHA256(_key);
        return hmac.ComputeHash(Encoding.UTF8.GetBytes(body));
    }

    private static string Base64Url(byte[] data) =>
        Convert.ToBase64String(data).TrimEnd('=').Replace('+', '-').Replace('/', '_');

    private static byte[] FromBase64Url(string s)
    {
        var b64 = s.Replace('-', '+').Replace('_', '/');
        b64 = (b64.Length % 4) switch { 2 => b64 + "==", 3 => b64 + "=", _ => b64 };
        return Convert.FromBase64String(b64);
    }

    private sealed class TokenPayload
    {
        public string sub { get; set; } = default!;
        public int uid { get; set; }
        public string? name { get; set; }
        public long exp { get; set; }
    }
}
