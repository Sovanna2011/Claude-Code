using System.Text.Json.Serialization;
using Microsoft.AspNetCore.Authentication;
using Microsoft.OpenApi.Models;
using Microsoft.EntityFrameworkCore;
using PoApproval.Api.Data;
using PoApproval.Api.Middleware;
using PoApproval.Api.Security;
using PoApproval.Api.Services;
using PoApproval.Api.Services.Interfaces;
using PoApproval.Api.Services.Sap;

var builder = WebApplication.CreateBuilder(args);

// ---- Services --------------------------------------------------------------
builder.Services.AddControllers()
    .AddJsonOptions(o =>
    {
        o.JsonSerializerOptions.PropertyNamingPolicy = System.Text.Json.JsonNamingPolicy.CamelCase;
        o.JsonSerializerOptions.DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull;
        o.JsonSerializerOptions.ReferenceHandler = ReferenceHandler.IgnoreCycles;
    });

builder.Services.AddDbContext<PoDbContext>(opt =>
    opt.UseSqlServer(builder.Configuration.GetConnectionString("POApproval")));

// Security primitives.
builder.Services.AddSingleton<PasswordHasher>();
builder.Services.AddSingleton<TokenService>();

// SAP ECC boundary. The default is the SQL replica; register an RFC/NCo
// implementation here when Sap:UseLocalMirror is false (see docs/architecture.md).
var useMirror = builder.Configuration.GetValue<bool?>("Sap:UseLocalMirror") ?? true;
if (useMirror)
    builder.Services.AddScoped<ISapEccConnector, DbSapEccConnector>();
else
    throw new InvalidOperationException(
        "Sap:UseLocalMirror=false requires an RFC-based ISapEccConnector implementation. " +
        "See docs/architecture.md for wiring the SAP .NET Connector (NCo 3).");

// Application services.
builder.Services.AddScoped<IAuthService, AuthService>();
builder.Services.AddScoped<IPurchaseOrderService, PurchaseOrderService>();
builder.Services.AddScoped<IReleaseService, ReleaseService>();
builder.Services.AddScoped<IValueHelpService, ValueHelpService>();

// Authentication / authorization (custom bearer token scheme).
builder.Services.AddAuthentication(TokenService.SchemeName)
    .AddScheme<AuthenticationSchemeOptions, TokenAuthenticationHandler>(TokenService.SchemeName, null);
builder.Services.AddAuthorization();

builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen(c =>
{
    c.SwaggerDoc("v1", new() { Title = "PO Approval API", Version = "v1",
        Description = "SAP ECC 6.0 EHP8-style Purchase Order release/approval - C# / SQL Server backend for SAPUI5." });

    var scheme = new OpenApiSecurityScheme
    {
        Name = "Authorization",
        Type = SecuritySchemeType.Http,
        Scheme = "bearer",
        BearerFormat = "Token",
        In = ParameterLocation.Header,
        Description = "Paste the token returned by POST /api/auth/login.",
        Reference = new OpenApiReference { Type = ReferenceType.SecurityScheme, Id = "Bearer" }
    };
    c.AddSecurityDefinition("Bearer", scheme);
    c.AddSecurityRequirement(new OpenApiSecurityRequirement { [scheme] = Array.Empty<string>() });
});

var corsOrigins = builder.Configuration.GetSection("Cors:AllowedOrigins").Get<string[]>()
                  ?? new[] { "http://localhost:8080" };
builder.Services.AddCors(o => o.AddPolicy("ui5", p =>
    p.WithOrigins(corsOrigins).AllowAnyHeader().AllowAnyMethod()));

var app = builder.Build();

// ---- Pipeline --------------------------------------------------------------
app.UseMiddleware<ExceptionHandlingMiddleware>();

if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI(c => c.SwaggerEndpoint("/swagger/v1/swagger.json", "PO Approval API v1"));
}

app.UseCors("ui5");

// Serve the bundled SAPUI5 app from wwwroot if present.
app.UseDefaultFiles();
app.UseStaticFiles();

app.UseAuthentication();
app.UseAuthorization();

app.MapControllers();
app.MapGet("/health", () => Results.Ok(new { status = "UP", module = "MM-PUR", time = DateTime.UtcNow }));

app.Run();
