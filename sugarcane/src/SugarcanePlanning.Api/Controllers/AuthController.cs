using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Api.Security;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Domain.Enums;
using SugarcanePlanning.Infrastructure.Identity;
using SugarcanePlanning.Infrastructure.Persistence;

namespace SugarcanePlanning.Api.Controllers;

/// <summary>Sign-in, user administration and the caller's own profile (section 21).</summary>
[ApiController]
[Route("api/auth")]
[Produces("application/json")]
public class AuthController : ControllerBase
{
    private readonly UserManager<AppUser> _users;
    private readonly JwtTokenService _tokens;
    private readonly AppDbContext _db;
    private readonly IAuditService _audit;
    private readonly IDateTimeProvider _clock;

    public AuthController(UserManager<AppUser> users, JwtTokenService tokens, AppDbContext db,
        IAuditService audit, IDateTimeProvider clock)
    {
        _users = users;
        _tokens = tokens;
        _db = db;
        _audit = audit;
        _clock = clock;
    }

    /// <summary>Exchanges credentials for a bearer token carrying the company and role claims.</summary>
    [HttpPost("login")]
    [AllowAnonymous]
    [ProducesResponseType(typeof(LoginResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ApiErrorDto), StatusCodes.Status401Unauthorized)]
    public async Task<ActionResult<LoginResponse>> Login([FromBody] LoginRequest request, CancellationToken ct)
    {
        var user = await _users.FindByNameAsync(request.UserName);

        // An unknown user and a wrong password answer identically, so the endpoint cannot be
        // used to discover which user names exist.
        if (user is null || !user.IsActive)
            return Unauthorized(new ApiErrorDto { Code = "INVALID_CREDENTIALS", Message = "Invalid user name or password." });

        // A locked account is told so plainly: the person needs to know why their correct
        // password stopped working, and by this point they have already proved the name exists.
        if (await _users.IsLockedOutAsync(user))
        {
            await _audit.LogAsync(AuditAction.Login, "Users", user.Id, null, user.UserName, "Locked out", ct);
            return Unauthorized(new ApiErrorDto
            {
                Code = "ACCOUNT_LOCKED",
                Message = "This account is locked after too many failed sign-in attempts. Try again later."
            });
        }

        if (!await _users.CheckPasswordAsync(user, request.Password))
        {
            // CheckPasswordAsync alone neither counts failures nor locks anything, so the
            // configured MaxFailedAccessAttempts would never take effect and the password could
            // be guessed indefinitely. AccessFailedAsync is what applies the lockout.
            await _users.AccessFailedAsync(user);
            return Unauthorized(new ApiErrorDto { Code = "INVALID_CREDENTIALS", Message = "Invalid user name or password." });
        }

        await _users.ResetAccessFailedCountAsync(user);

        var roles = await _users.GetRolesAsync(user);
        var (token, expires) = _tokens.CreateToken(user, roles);

        user.LastLoginUtc = _clock.UtcNow;
        await _users.UpdateAsync(user);

        var company = await _db.Companies.IgnoreQueryFilters().AsNoTracking()
            .FirstOrDefaultAsync(c => c.Id == user.CompanyId, ct);

        await _audit.LogAsync(AuditAction.Login, "Users", user.Id, null, user.UserName, null, ct);

        return Ok(new LoginResponse
        {
            Token = token,
            ExpiresAtUtc = expires,
            UserName = user.UserName ?? string.Empty,
            FullName = user.FullName,
            CompanyId = user.CompanyId,
            CompanyName = company?.Name ?? string.Empty,
            Roles = roles.ToList(),
            Permissions = Policies.All.Where(p =>
                Policies.RoleMap.TryGetValue(p, out var allowed) && allowed.Intersect(roles).Any()).ToList()
        });
    }

    /// <summary>Profile of the signed-in user; the client uses it to rebuild its menu after a refresh.</summary>
    [HttpGet("me")]
    [Authorize]
    public async Task<ActionResult<UserDto>> Me(CancellationToken ct)
    {
        var user = await _users.FindByNameAsync(User.Identity?.Name ?? string.Empty);
        if (user is null) return Unauthorized();

        var roles = await _users.GetRolesAsync(user);
        var company = await _db.Companies.IgnoreQueryFilters().AsNoTracking()
            .FirstOrDefaultAsync(c => c.Id == user.CompanyId, ct);

        return Ok(new UserDto
        {
            Id = user.Id,
            UserName = user.UserName ?? string.Empty,
            Email = user.Email ?? string.Empty,
            FullName = user.FullName,
            CompanyId = user.CompanyId,
            CompanyName = company?.Name ?? string.Empty,
            IsActive = user.IsActive,
            Roles = roles.ToList()
        });
    }

    [HttpGet("users")]
    [Authorize(Policy = Policies.Administer)]
    public async Task<ActionResult<IReadOnlyList<UserDto>>> Users(CancellationToken ct)
    {
        var users = await _users.Users.AsNoTracking().OrderBy(u => u.UserName).ToListAsync(ct);
        var result = new List<UserDto>();
        foreach (var user in users)
            result.Add(new UserDto
            {
                Id = user.Id,
                UserName = user.UserName ?? string.Empty,
                Email = user.Email ?? string.Empty,
                FullName = user.FullName,
                CompanyId = user.CompanyId,
                IsActive = user.IsActive,
                Roles = (await _users.GetRolesAsync(user)).ToList()
            });
        return Ok(result);
    }

    [HttpPost("users")]
    [Authorize(Policy = Policies.Administer)]
    public async Task<ActionResult<UserDto>> Register([FromBody] RegisterUserRequest request, CancellationToken ct)
    {
        var user = new AppUser
        {
            UserName = request.UserName,
            Email = request.Email,
            EmailConfirmed = true,
            FullName = request.FullName,
            CompanyId = request.CompanyId,
            IsActive = true
        };

        var created = await _users.CreateAsync(user, request.Password);
        if (!created.Succeeded)
            return UnprocessableEntity(new ApiErrorDto
            {
                Code = "USER_CREATE_FAILED",
                Message = string.Join("; ", created.Errors.Select(e => e.Description))
            });

        var roles = request.Roles.Where(r => AppRoles.All.Contains(r)).ToList();
        if (roles.Count > 0) await _users.AddToRolesAsync(user, roles);

        await _audit.LogAsync(AuditAction.Create, "Users", user.Id, null, user.UserName, null, ct);

        return Ok(new UserDto
        {
            Id = user.Id,
            UserName = user.UserName ?? string.Empty,
            Email = user.Email ?? string.Empty,
            FullName = user.FullName,
            CompanyId = user.CompanyId,
            IsActive = true,
            Roles = roles
        });
    }

    [HttpPost("change-password")]
    [Authorize]
    public async Task<IActionResult> ChangePassword([FromBody] ChangePasswordRequest request)
    {
        var user = await _users.FindByNameAsync(User.Identity?.Name ?? string.Empty);
        if (user is null) return Unauthorized();

        var result = await _users.ChangePasswordAsync(user, request.CurrentPassword, request.NewPassword);
        if (!result.Succeeded)
            return UnprocessableEntity(new ApiErrorDto
            {
                Code = "PASSWORD_CHANGE_FAILED",
                Message = string.Join("; ", result.Errors.Select(e => e.Description))
            });

        return NoContent();
    }

    /// <summary>The ten roles of section 21, for the user-administration screen.</summary>
    [HttpGet("roles")]
    [Authorize(Policy = Policies.Administer)]
    public ActionResult<IReadOnlyList<string>> Roles() => Ok(AppRoles.All);
}
