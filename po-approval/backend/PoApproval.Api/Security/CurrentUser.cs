using System.Security.Claims;

namespace PoApproval.Api.Security;

/// <summary>The authenticated principal, projected from the token claims.</summary>
public record CurrentUser(int UserId, string UserName, string DisplayName)
{
    public static CurrentUser From(ClaimsPrincipal principal)
    {
        var uid = principal.FindFirstValue(ClaimTypes.NameIdentifier);
        var name = principal.FindFirstValue(ClaimTypes.Name);
        var display = principal.FindFirstValue("displayName") ?? name;
        if (uid is null || name is null)
            throw new InvalidOperationException("Authenticated user claims are incomplete.");
        return new CurrentUser(int.Parse(uid), name, display ?? name);
    }
}
