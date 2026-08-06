using ErpS4.Application;
using ErpS4.Database;
using Microsoft.Extensions.DependencyInjection;
using Xunit;

namespace ErpS4.IntegrationTests;

/// <summary>
/// A real <see cref="ErpDbContext"/> over a real SQL Server.
/// </summary>
/// <remarks>
/// The unit tests run the same rules over in-memory lists, which is fast and
/// says nothing about whether the model maps. Only a database can fail on a
/// column that is too narrow, a query filter that does not translate, a
/// <c>decimal</c> that rounds differently in T-SQL, or a transaction that
/// deadlocks - so these tests exist to fail in those ways before a user does.
/// </remarks>
public sealed class DatabaseFixture : IDisposable
{
    public const string ConnectionVariable = "ERPS4_TEST_CONNECTION";

    private readonly ServiceProvider? _provider;

    public DatabaseFixture()
    {
        ConnectionString = Environment.GetEnvironmentVariable(ConnectionVariable);

        if (string.IsNullOrWhiteSpace(ConnectionString))
        {
            return;
        }

        var services = new ServiceCollection();
        services.AddErpDatabase(ConnectionString);
        services.AddErpApplication();
        services.AddSingleton<ITenantProvider>(new FixedTenantProvider(1));
        services.AddSingleton<ICurrentUser>(new FixedCurrentUser("integration"));
        services.AddLogging();

        _provider = services.BuildServiceProvider();
    }

    public string? ConnectionString { get; }

    /// <summary>True when a server was configured for this run.</summary>
    public bool IsAvailable => _provider is not null;

    /// <summary>A fresh scope, i.e. a fresh DbContext, as a request would get.</summary>
    public IServiceScope CreateScope() =>
        (_provider ?? throw new InvalidOperationException(
            $"No database configured. Set {ConnectionVariable}."))
        .CreateScope();

    public void Dispose() => _provider?.Dispose();
}

[CollectionDefinition(Name)]
public sealed class DatabaseCollection : ICollectionFixture<DatabaseFixture>
{
    public const string Name = "sql-server";
}

/// <summary>
/// Skips rather than fails when no server is configured, so the suite stays
/// runnable on a machine that has no SQL Server without pretending it passed.
/// </summary>
public sealed class RequiresDatabaseFactAttribute : FactAttribute
{
    public RequiresDatabaseFactAttribute()
    {
        if (string.IsNullOrWhiteSpace(
                Environment.GetEnvironmentVariable(DatabaseFixture.ConnectionVariable)))
        {
            Skip = $"Set {DatabaseFixture.ConnectionVariable} to run integration tests.";
        }
    }
}
