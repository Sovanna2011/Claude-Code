using Microsoft.EntityFrameworkCore;
using PoApproval.Api.Data;
using PoApproval.Api.DTOs;
using PoApproval.Api.Security;
using PoApproval.Api.Services.Interfaces;

namespace PoApproval.Api.Services;

/// <summary>
/// User-level authentication. Verifies credentials against the PO.AppUser store
/// and issues a signed bearer token carrying the user's identity. The token is
/// consumed by the UI5 front end and validated on every protected API call.
/// </summary>
public class AuthService : IAuthService
{
    private readonly PoDbContext _db;
    private readonly PasswordHasher _hasher;
    private readonly TokenService _tokens;
    private readonly string _dummyHash;

    public AuthService(PoDbContext db, PasswordHasher hasher, TokenService tokens)
    {
        _db = db;
        _hasher = hasher;
        _tokens = tokens;
        // A well-formed hash verified when the user does not exist, so the
        // response time does not reveal whether a user name is valid.
        _dummyHash = hasher.Hash("\0nonexistent\0");
    }

    public async Task<LoginResponse?> LoginAsync(LoginRequest request, CancellationToken ct = default)
    {
        var user = await _db.AppUsers
            .Include(u => u.ReleaseCodes)
            .FirstOrDefaultAsync(u => u.UserName == request.Username && u.IsActive, ct);

        // Verify even when the user is missing, to keep timing uniform.
        var ok = _hasher.Verify(request.Password, user?.PwdHash ?? _dummyHash);
        if (user is null || !ok) return null;

        var (token, expires) = _tokens.Create(user);
        return new LoginResponse
        {
            Token = token,
            ExpiresAt = expires,
            User = await BuildUserInfoAsync(user.UserId, ct) ?? throw new InvalidOperationException("User vanished.")
        };
    }

    public Task<UserInfoDto?> GetUserInfoAsync(int userId, CancellationToken ct = default)
        => BuildUserInfoAsync(userId, ct);

    private async Task<UserInfoDto?> BuildUserInfoAsync(int userId, CancellationToken ct)
    {
        var user = await _db.AppUsers.AsNoTracking()
            .Include(u => u.ReleaseCodes)
            .FirstOrDefaultAsync(u => u.UserId == userId, ct);
        if (user is null) return null;

        var codeTexts = await _db.T16FC.AsNoTracking()
            .ToDictionaryAsync(c => (c.FRGGR, c.FRGCO), c => c.FRGCT, ct);

        return new UserInfoDto
        {
            UserId = user.UserId,
            Username = user.UserName,
            DisplayName = user.DisplayName,
            Email = user.Email,
            ReleaseCodes = user.ReleaseCodes
                .Select(rc => new ReleaseCodeDto
                {
                    Group = rc.FRGGR,
                    Code = rc.FRGCO,
                    Text = codeTexts.GetValueOrDefault((rc.FRGGR, rc.FRGCO))
                })
                .OrderBy(c => c.Group).ThenBy(c => c.Code)
                .ToList()
        };
    }
}
