using System.Text.Encodings.Web;
using Microsoft.AspNetCore.Authentication;
using Microsoft.Extensions.Options;

namespace PoApproval.Api.Security;

/// <summary>
/// Authentication handler that reads a bearer token from the Authorization
/// header and validates it with <see cref="TokenService"/>. Registered under
/// the scheme <see cref="TokenService.SchemeName"/> and used by [Authorize].
/// </summary>
public class TokenAuthenticationHandler : AuthenticationHandler<AuthenticationSchemeOptions>
{
    private readonly TokenService _tokens;

    public TokenAuthenticationHandler(
        IOptionsMonitor<AuthenticationSchemeOptions> options,
        ILoggerFactory logger,
        UrlEncoder encoder,
        TokenService tokens)
        : base(options, logger, encoder)
    {
        _tokens = tokens;
    }

    protected override Task<AuthenticateResult> HandleAuthenticateAsync()
    {
        if (!Request.Headers.TryGetValue("Authorization", out var header))
            return Task.FromResult(AuthenticateResult.NoResult());

        var raw = header.ToString();
        var token = raw.StartsWith("Bearer ", StringComparison.OrdinalIgnoreCase)
            ? raw["Bearer ".Length..].Trim()
            : raw.Trim();

        var principal = _tokens.Validate(token);
        if (principal is null)
            return Task.FromResult(AuthenticateResult.Fail("Invalid or expired token."));

        var ticket = new AuthenticationTicket(principal, Scheme.Name);
        return Task.FromResult(AuthenticateResult.Success(ticket));
    }
}
