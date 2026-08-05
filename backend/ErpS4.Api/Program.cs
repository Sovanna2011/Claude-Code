using System.Text;
using ErpS4.Api.Endpoints;
using ErpS4.Api.Middleware;
using ErpS4.Api.Security;
using ErpS4.Application;
using ErpS4.Database;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.AspNetCore.Authorization;
using Microsoft.IdentityModel.Tokens;
using Serilog;

var builder = WebApplication.CreateBuilder(args);

// Serilog from configuration: technical logs go here, business audit goes to
// audit.AuditLog. The two are deliberately separate stores.
builder.Host.UseSerilog((context, configuration) =>
    configuration.ReadFrom.Configuration(context.Configuration));

builder.Services.AddErpDatabase(
    builder.Configuration.GetConnectionString("ErpS4")
    ?? throw new InvalidOperationException("ConnectionStrings:ErpS4 is not configured."));
builder.Services.AddErpApplication();

builder.Services.AddHttpContextAccessor();
builder.Services.AddScoped<ITenantProvider, HttpTenantProvider>();
builder.Services.AddScoped<HttpCurrentUser>();
builder.Services.AddScoped<ICurrentUser>(p => p.GetRequiredService<HttpCurrentUser>());
builder.Services.AddScoped<ICurrentUserAccessor>(p => p.GetRequiredService<HttpCurrentUser>());
builder.Services.AddScoped<IOrganizationalAccessGuard, OrganizationalAccessGuard>();

builder.Services
    .AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options =>
    {
        var jwt = builder.Configuration.GetSection("Jwt");
        options.TokenValidationParameters = new TokenValidationParameters
        {
            ValidateIssuer = true,
            ValidateAudience = true,
            ValidateLifetime = true,
            ValidateIssuerSigningKey = true,
            ValidIssuer = jwt["Issuer"],
            ValidAudience = jwt["Audience"],
            IssuerSigningKey = new SymmetricSecurityKey(
                Encoding.UTF8.GetBytes(jwt["SigningKey"]
                    ?? throw new InvalidOperationException("Jwt:SigningKey is not configured."))),
            ClockSkew = TimeSpan.FromMinutes(1),
        };
    });

// Deny by default: an endpoint has to opt out of authorization explicitly.
builder.Services.AddAuthorization(options =>
    options.FallbackPolicy = new AuthorizationPolicyBuilder()
        .RequireAuthenticatedUser()
        .Build());

builder.Services.AddSingleton<IAuthorizationPolicyProvider, PermissionPolicyProvider>();
builder.Services.AddScoped<IAuthorizationHandler, PermissionAuthorizationHandler>();

builder.Services.AddOpenApi();
builder.Services.AddProblemDetails();

var app = builder.Build();

app.UseSerilogRequestLogging();
app.UseMiddleware<CorrelationIdMiddleware>();
app.UseMiddleware<ProblemDetailsMiddleware>();

if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
}

app.UseAuthentication();
app.UseAuthorization();

app.MapJournalEntryEndpoints();
app.MapBusinessPartnerEndpoints();
app.MapPaymentEndpoints();
app.MapAssetEndpoints();
app.MapApprovalEndpoints();
app.MapDictionaryEndpoints();
app.MapTableBrowserEndpoints();

app.MapGet("/health", () => Results.Ok(new { status = "ok" }))
    .AllowAnonymous()
    .WithTags("Diagnostics")
    .ExcludeFromDescription();

app.Run();

/// <summary>Exposed so integration tests can build the same host.</summary>
public partial class Program;
