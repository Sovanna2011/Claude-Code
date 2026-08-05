using System.Text;
using System.Text.Json.Serialization;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.EntityFrameworkCore;
using Microsoft.IdentityModel.Tokens;
using Microsoft.OpenApi;
using SugarcanePlanning.Api.Middleware;
using SugarcanePlanning.Api.Security;
using SugarcanePlanning.Application;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Infrastructure;
using SugarcanePlanning.Infrastructure.Persistence;
using SugarcanePlanning.Infrastructure.Persistence.Seed;

var builder = WebApplication.CreateBuilder(args);

// ------------------------------------------------------------------- services

builder.Services.AddHttpContextAccessor();
builder.Services.AddApplication();
builder.Services.AddInfrastructure(builder.Configuration);

builder.Services.Configure<JwtOptions>(builder.Configuration.GetSection(JwtOptions.SectionName));
builder.Services.AddScoped<JwtTokenService>();

builder.Services.AddControllers()
    .AddJsonOptions(options =>
    {
        // Enums travel as names so the Blazor client and integrations stay readable.
        options.JsonSerializerOptions.Converters.Add(new JsonStringEnumConverter());
        options.JsonSerializerOptions.DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull;
    });

var jwt = builder.Configuration.GetSection(JwtOptions.SectionName).Get<JwtOptions>() ?? new JwtOptions();

builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options =>
    {
        options.TokenValidationParameters = new TokenValidationParameters
        {
            ValidateIssuer = true,
            ValidateAudience = true,
            ValidateLifetime = true,
            ValidateIssuerSigningKey = true,
            ValidIssuer = jwt.Issuer,
            ValidAudience = jwt.Audience,
            IssuerSigningKey = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(jwt.SigningKey)),
            ClockSkew = TimeSpan.FromMinutes(1)
        };
    });

// One policy per permission of section 21, satisfied by the roles in the role map.
builder.Services.AddAuthorization(options =>
{
    foreach (var policy in Policies.All)
    {
        var roles = Policies.RoleMap.TryGetValue(policy, out var allowed) ? allowed : Array.Empty<string>();
        options.AddPolicy(policy, p => p.RequireAuthenticatedUser().RequireRole(roles));
    }
});

builder.Services.AddCors(options => options.AddPolicy("BlazorClient", policy => policy
    .WithOrigins(builder.Configuration.GetSection("Cors:AllowedOrigins").Get<string[]>()
                 ?? new[] { "https://localhost:7150", "http://localhost:5150" })
    .AllowAnyHeader()
    .AllowAnyMethod()
    // Without this the browser hides Content-Disposition from the Blazor client, and every
    // exported report is saved as "report.pdf" instead of its real name.
    .WithExposedHeaders("Content-Disposition")));

builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen(options =>
{
    options.SwaggerDoc("v1", new OpenApiInfo
    {
        Title = "Sugarcane Planting Planning Management System",
        Version = "v1",
        Description = "Planning and control of sugarcane planting operations by activity, tractor, " +
                      "equipment, material, workforce, location and schedule."
    });

    options.AddSecurityDefinition("Bearer", new OpenApiSecurityScheme
    {
        Name = "Authorization",
        Type = SecuritySchemeType.Http,
        Scheme = "bearer",
        BearerFormat = "JWT",
        In = ParameterLocation.Header,
        Description = "Paste the token returned by POST /api/auth/login."
    });
    options.AddSecurityRequirement(_ => new OpenApiSecurityRequirement
    {
        [new OpenApiSecuritySchemeReference("Bearer")] = new List<string>()
    });

    var xml = Path.Combine(AppContext.BaseDirectory, "SugarcanePlanning.Api.xml");
    if (File.Exists(xml)) options.IncludeXmlComments(xml);
});

builder.Services.AddHealthChecks().AddDbContextCheck<AppDbContext>();

var app = builder.Build();

// -------------------------------------------------------------------- pipeline

app.UseMiddleware<ExceptionHandlingMiddleware>();

if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI(o => o.SwaggerEndpoint("/swagger/v1/swagger.json", "Sugarcane Planning API v1"));
}
else
{
    app.UseHsts();
    app.UseHttpsRedirection();
}

app.UseCors("BlazorClient");
app.UseAuthentication();
app.UseAuthorization();

app.MapControllers();
app.MapHealthChecks("/health");

// Applies pending migrations and seeds the sample data when configured to.
if (builder.Configuration.GetValue("Database:MigrateOnStartup", true))
{
    using var scope = app.Services.CreateScope();
    var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
    await db.Database.MigrateAsync();

    if (builder.Configuration.GetValue("Database:SeedSampleData", true))
        await DbSeeder.SeedAsync(scope.ServiceProvider);
}

app.Run();

/// <summary>Exposed so the integration tests can host the API with <c>WebApplicationFactory</c>.</summary>
public partial class Program { }
