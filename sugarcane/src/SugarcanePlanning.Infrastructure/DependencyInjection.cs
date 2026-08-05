using Microsoft.AspNetCore.Identity;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Infrastructure.Identity;
using SugarcanePlanning.Infrastructure.Persistence;
using SugarcanePlanning.Infrastructure.Persistence.Interceptors;
using SugarcanePlanning.Infrastructure.Reporting;
using SugarcanePlanning.Infrastructure.Services;

namespace SugarcanePlanning.Infrastructure;

/// <summary>Wires the database, Identity, the audit interceptor and the report exporters.</summary>
public static class DependencyInjection
{
    public static IServiceCollection AddInfrastructure(this IServiceCollection services, IConfiguration configuration)
    {
        services.Configure<PlanningOptions>(configuration.GetSection(PlanningOptions.SectionName));

        services.AddScoped<IDateTimeProvider, SystemDateTimeProvider>();
        services.AddScoped<ICurrentUser, HttpCurrentUser>();
        services.AddScoped<IAuditService, AuditService>();
        services.AddScoped<AuditSaveChangesInterceptor>();

        var connectionString = configuration.GetConnectionString("SugarcanePlanning")
                               ?? "Server=localhost;Database=SugarcanePlanning;Trusted_Connection=True;TrustServerCertificate=True";

        services.AddDbContext<AppDbContext>((provider, options) =>
        {
            options.UseSqlServer(connectionString, sql =>
            {
                sql.MigrationsHistoryTable("__EFMigrationsHistory", "planning");
                sql.EnableRetryOnFailure(3);
            });
            options.AddInterceptors(provider.GetRequiredService<AuditSaveChangesInterceptor>());
        });

        services.AddScoped<IAppDbContext>(provider => provider.GetRequiredService<AppDbContext>());

        services.AddIdentityCore<AppUser>(options =>
            {
                options.Password.RequiredLength = 8;
                options.Password.RequireDigit = true;
                options.Password.RequireUppercase = true;
                options.Password.RequireNonAlphanumeric = false;
                options.User.RequireUniqueEmail = true;
                options.Lockout.MaxFailedAccessAttempts = 5;
            })
            .AddRoles<AppRole>()
            .AddEntityFrameworkStores<AppDbContext>()
            .AddDefaultTokenProviders();

        services.AddScoped<IReportExporter, PdfReportExporter>();
        services.AddScoped<IReportExporter, ExcelReportExporter>();
        services.AddScoped<IReportExporter, JsonReportExporter>();

        return services;
    }
}
