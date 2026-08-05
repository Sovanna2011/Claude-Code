using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Text.Json;
using Microsoft.AspNetCore.Components.Authorization;
using Microsoft.JSInterop;
using SugarcanePlanning.Contracts.Auth;

namespace SugarcanePlanning.Client.Services;

/// <summary>Persists the bearer token and the signed-in profile in browser local storage.</summary>
public class TokenStore
{
    private const string TokenKey = "sgc.token";
    private const string ProfileKey = "sgc.profile";

    private readonly IJSRuntime _js;
    private string? _cachedToken;
    private LoginResponse? _cachedProfile;

    public TokenStore(IJSRuntime js) => _js = js;

    public async Task<string?> GetTokenAsync()
    {
        if (_cachedToken is not null) return _cachedToken;
        _cachedToken = await GetItemAsync(TokenKey);
        return _cachedToken;
    }

    public async Task<LoginResponse?> GetProfileAsync()
    {
        if (_cachedProfile is not null) return _cachedProfile;
        var raw = await GetItemAsync(ProfileKey);
        if (string.IsNullOrWhiteSpace(raw)) return null;
        _cachedProfile = JsonSerializer.Deserialize<LoginResponse>(raw, ApiClient.JsonOptions);
        return _cachedProfile;
    }

    public async Task SaveAsync(LoginResponse login)
    {
        _cachedToken = login.Token;
        _cachedProfile = login;
        await SetItemAsync(TokenKey, login.Token);
        await SetItemAsync(ProfileKey, JsonSerializer.Serialize(login, ApiClient.JsonOptions));
    }

    public async Task ClearAsync()
    {
        _cachedToken = null;
        _cachedProfile = null;
        await _js.InvokeVoidAsync("localStorage.removeItem", TokenKey);
        await _js.InvokeVoidAsync("localStorage.removeItem", ProfileKey);
    }

    private async Task<string?> GetItemAsync(string key)
    {
        try
        {
            return await _js.InvokeAsync<string?>("localStorage.getItem", key);
        }
        catch (JSException)
        {
            return null;      // storage blocked (private mode); the user simply signs in again
        }
    }

    private Task SetItemAsync(string key, string value) => _js.InvokeVoidAsync("localStorage.setItem", key, value).AsTask();
}

/// <summary>Builds the <see cref="ClaimsPrincipal"/> from the stored JWT, honouring its expiry.</summary>
public class JwtAuthenticationStateProvider : AuthenticationStateProvider
{
    private readonly TokenStore _tokens;

    public JwtAuthenticationStateProvider(TokenStore tokens) => _tokens = tokens;

    public override async Task<AuthenticationState> GetAuthenticationStateAsync()
    {
        var token = await _tokens.GetTokenAsync();
        if (string.IsNullOrWhiteSpace(token)) return Anonymous();

        JwtSecurityToken jwt;
        try
        {
            jwt = new JwtSecurityTokenHandler().ReadJwtToken(token);
        }
        catch (Exception)
        {
            await _tokens.ClearAsync();
            return Anonymous();
        }

        if (jwt.ValidTo <= DateTime.UtcNow)
        {
            await _tokens.ClearAsync();
            return Anonymous();
        }

        // The role claim type in the token is the full URI; map it so IsInRole works.
        var claims = jwt.Claims.Select(c => c.Type == "role" ? new Claim(ClaimTypes.Role, c.Value) : c).ToList();
        var identity = new ClaimsIdentity(claims, "jwt", ClaimTypes.Name, ClaimTypes.Role);
        return new AuthenticationState(new ClaimsPrincipal(identity));
    }

    public void NotifyAuthenticationStateChanged()
        => NotifyAuthenticationStateChanged(GetAuthenticationStateAsync());

    private static AuthenticationState Anonymous() => new(new ClaimsPrincipal(new ClaimsIdentity()));
}

/// <summary>Sign-in / sign-out plus the permission checks the navigation menu uses.</summary>
public class AuthService
{
    private readonly ApiClient _api;
    private readonly TokenStore _tokens;
    private readonly JwtAuthenticationStateProvider _stateProvider;

    public AuthService(ApiClient api, TokenStore tokens, AuthenticationStateProvider stateProvider)
    {
        _api = api;
        _tokens = tokens;
        _stateProvider = (JwtAuthenticationStateProvider)stateProvider;
    }

    public async Task<LoginResponse> LoginAsync(LoginRequest request)
    {
        var response = await _api.PostAsync<LoginResponse>("api/auth/login", request)
                       ?? throw new InvalidOperationException("The server returned an empty login response.");
        await _tokens.SaveAsync(response);
        _stateProvider.NotifyAuthenticationStateChanged();
        return response;
    }

    public async Task LogoutAsync()
    {
        await _tokens.ClearAsync();
        _stateProvider.NotifyAuthenticationStateChanged();
    }

    public Task<LoginResponse?> GetProfileAsync() => _tokens.GetProfileAsync();

    /// <summary>True when the signed-in user's roles satisfy the policy (section 21).</summary>
    public async Task<bool> HasPermissionAsync(string policy)
    {
        var profile = await _tokens.GetProfileAsync();
        return profile is not null && profile.Permissions.Contains(policy);
    }
}
