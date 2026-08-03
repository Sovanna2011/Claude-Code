using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using PoApproval.Api.DTOs;
using PoApproval.Api.Security;
using PoApproval.Api.Services.Interfaces;

namespace PoApproval.Api.Controllers;

/// <summary>User-level authentication endpoints.</summary>
[ApiController]
[Route("api/[controller]")]
[Produces("application/json")]
public class AuthController : ControllerBase
{
    private readonly IAuthService _auth;

    public AuthController(IAuthService auth) => _auth = auth;

    /// <summary>Authenticate with username + password and receive a bearer token.</summary>
    [AllowAnonymous]
    [HttpPost("login")]
    public async Task<ActionResult<LoginResponse>> Login([FromBody] LoginRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var result = await _auth.LoginAsync(request, ct);
        return result is null
            ? Unauthorized(new { message = "Invalid user name or password." })
            : Ok(result);
    }

    /// <summary>Return the signed-in user's profile and release codes.</summary>
    [Authorize]
    [HttpGet("me")]
    public async Task<ActionResult<UserInfoDto>> Me(CancellationToken ct)
    {
        var user = CurrentUser.From(User);
        var info = await _auth.GetUserInfoAsync(user.UserId, ct);
        return info is null ? Unauthorized() : Ok(info);
    }
}
