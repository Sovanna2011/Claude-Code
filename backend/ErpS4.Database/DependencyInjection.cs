using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;

namespace ErpS4.Database;

/// <summary>Registration for the ERP database module.</summary>
public static class DependencyInjection
{
    /// <summary>
    /// Registers <see cref="ErpDbContext"/> as a scoped dependency, together
    /// with connection resilience and a command timeout. The caller supplies
    /// <see cref="ITenantProvider"/> and <see cref="ICurrentUser"/>, which are
    /// request scoped in a web host.
    /// </summary>
    public static IServiceCollection AddErpDatabase(
        this IServiceCollection services,
        string connectionString,
        int commandTimeoutSeconds = 60)
    {
        services.AddDbContext<ErpDbContext>(options =>
            options.UseSqlServer(connectionString, sqlServer =>
            {
                sqlServer.EnableRetryOnFailure(
                    maxRetryCount: 5,
                    maxRetryDelay: TimeSpan.FromSeconds(10),
                    errorNumbersToAdd: null);
                sqlServer.CommandTimeout(commandTimeoutSeconds);
            }));

        return services;
    }
}
