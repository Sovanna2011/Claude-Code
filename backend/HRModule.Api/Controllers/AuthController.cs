using System.Security.Claims;
using HRModule.Api.Data;
using HRModule.Api.DTOs;
using HRModule.Api.Security;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace HRModule.Api.Controllers;

/// <summary>Authentication endpoints (login, current user).</summary>
[ApiController]
[Route("api/[controller]")]
[Produces("application/json")]
public class AuthController : ControllerBase
{
    private readonly HRDbContext _db;
    private readonly JwtTokenService _jwt;

    public AuthController(HRDbContext db, JwtTokenService jwt)
    {
        _db = db;
        _jwt = jwt;
    }

    /// <summary>Authenticates a user and returns a JWT.</summary>
    [HttpPost("login")]
    [AllowAnonymous]
    public async Task<ActionResult<LoginResponse>> Login([FromBody] LoginRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);

        var user = await _db.AppUsers.FirstOrDefaultAsync(
            u => u.Username == request.Username && u.IsActive, ct);

        // Same generic message whether the user is unknown or the password is wrong.
        if (user is null || !PasswordHasher.Verify(request.Password, user.PasswordHash, user.PasswordSalt))
            return Unauthorized(new { message = "Invalid username or password." });

        user.LastLogin = DateTime.UtcNow;
        await _db.SaveChangesAsync(ct);

        var (token, expires) = _jwt.CreateToken(user);
        var roleName = await _db.AppRoles.Where(r => r.RoleKey == user.RoleKey)
            .Select(r => r.RoleName).FirstOrDefaultAsync(ct);

        return Ok(new LoginResponse
        {
            Token = token,
            ExpiresUtc = expires,
            User = new UserInfoDto
            {
                Username = user.Username, DisplayName = user.DisplayName,
                RoleKey = user.RoleKey, RoleName = roleName, Pernr = user.PERNR
            }
        });
    }

    /// <summary>Returns the currently authenticated user (from the token).</summary>
    [HttpGet("me")]
    [Authorize]
    public async Task<ActionResult<UserInfoDto>> Me(CancellationToken ct)
    {
        var username = User.FindFirstValue(System.IdentityModel.Tokens.Jwt.JwtRegisteredClaimNames.Sub)
                       ?? User.Identity?.Name;
        var user = await _db.AppUsers.FirstOrDefaultAsync(u => u.Username == username, ct);
        if (user is null) return Unauthorized();

        var roleName = await _db.AppRoles.Where(r => r.RoleKey == user.RoleKey)
            .Select(r => r.RoleName).FirstOrDefaultAsync(ct);

        return Ok(new UserInfoDto
        {
            Username = user.Username, DisplayName = user.DisplayName,
            RoleKey = user.RoleKey, RoleName = roleName, Pernr = user.PERNR
        });
    }
}
