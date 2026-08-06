using System.Net;
using System.Net.Http.Json;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.Data.SqlClient;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.AspNetCore.Identity;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Infrastructure.Identity;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// Sign-in is the one endpoint an unauthenticated attacker can call as often as they like, so
/// the lockout is the control that stops a password being guessed. It needs the real Identity
/// stores and therefore a real database: <c>UserManager.CheckPasswordAsync</c> answers correctly
/// against any provider, but only a real store records the failure count that locks the account.
/// </summary>
public class LoginLockoutTests : IClassFixture<LoginLockoutTests.ApiFactory>
{
    private readonly ApiFactory _factory;

    public LoginLockoutTests(ApiFactory factory) => _factory = factory;

    private const string Password = "Planner#2026";

    private static async Task<(HttpStatusCode Status, string? Code)> AttemptAsync(
        HttpClient client, string user, string password)
    {
        var response = await client.PostAsJsonAsync("/api/auth/login",
            new LoginRequest { UserName = user, Password = password });

        if (response.IsSuccessStatusCode) return (response.StatusCode, null);

        var error = await response.Content.ReadFromJsonAsync<ApiErrorDto>();
        return (response.StatusCode, error?.Code);
    }

    [SqlServerFact]
    public async Task Repeated_wrong_passwords_lock_the_account_and_the_right_one_stops_working()
    {
        var client = _factory.CreateClient();

        // The demo tenant is seeded on start-up; use an account no other test signs in with.
        const string user = "approver";

        Assert.Equal(HttpStatusCode.OK, (await AttemptAsync(client, user, Password)).Status);

        // Five is the configured threshold, so the fifth failure is what locks it.
        for (var attempt = 1; attempt <= 5; attempt++)
        {
            var (status, code) = await AttemptAsync(client, user, $"WrongGuess{attempt}");
            Assert.Equal(HttpStatusCode.Unauthorized, status);
            Assert.Equal("INVALID_CREDENTIALS", code);
        }

        // The correct password must now be refused too — otherwise the lockout is decorative
        // and the password can be guessed indefinitely.
        var locked = await AttemptAsync(client, user, Password);
        Assert.Equal(HttpStatusCode.Unauthorized, locked.Status);
        Assert.Equal("ACCOUNT_LOCKED", locked.Code);

        using var scope = _factory.Services.CreateScope();
        var users = scope.ServiceProvider.GetRequiredService<UserManager<AppUser>>();
        var locked_user = await users.FindByNameAsync(user);
        Assert.NotNull(locked_user);
        Assert.True(await users.IsLockedOutAsync(locked_user!));

        // Once the lock is lifted the account works again and the counter has been cleared.
        await users.SetLockoutEndDateAsync(locked_user!, null);
        await users.ResetAccessFailedCountAsync(locked_user!);

        Assert.Equal(HttpStatusCode.OK, (await AttemptAsync(client, user, Password)).Status);
        Assert.Equal(0, await users.GetAccessFailedCountAsync(
            (await users.FindByNameAsync(user))!));
    }

    [SqlServerFact]
    public async Task A_successful_sign_in_clears_earlier_failures()
    {
        var client = _factory.CreateClient();
        const string user = "materials";

        // Four failures is one short of the threshold.
        for (var attempt = 1; attempt <= 4; attempt++)
            Assert.Equal(HttpStatusCode.Unauthorized, (await AttemptAsync(client, user, "Nope!" + attempt)).Status);

        Assert.Equal(HttpStatusCode.OK, (await AttemptAsync(client, user, Password)).Status);

        using var scope = _factory.Services.CreateScope();
        var users = scope.ServiceProvider.GetRequiredService<UserManager<AppUser>>();
        Assert.Equal(0, await users.GetAccessFailedCountAsync((await users.FindByNameAsync(user))!));

        // Four more failures must therefore still not lock it: the count restarted.
        for (var attempt = 1; attempt <= 4; attempt++)
            Assert.Equal(HttpStatusCode.Unauthorized, (await AttemptAsync(client, user, "Nope!" + attempt)).Status);
        Assert.Equal(HttpStatusCode.OK, (await AttemptAsync(client, user, Password)).Status);
    }

    [SqlServerFact]
    public async Task An_unknown_user_and_a_wrong_password_are_indistinguishable()
    {
        var client = _factory.CreateClient();

        var unknown = await AttemptAsync(client, "no-such-person", Password);
        var wrong = await AttemptAsync(client, "supervisor", "definitely-not-it");

        // Different answers here would let an attacker enumerate valid user names.
        Assert.Equal(unknown.Status, wrong.Status);
        Assert.Equal(unknown.Code, wrong.Code);
        Assert.Equal("INVALID_CREDENTIALS", unknown.Code);
    }

    /// <summary>
    /// Hosts the real API against a throw-away SQL Server database, migrated and seeded exactly
    /// as it is on start-up, so the Identity stores are the real ones.
    /// </summary>
    public class ApiFactory : WebApplicationFactory<Program>, IAsyncLifetime
    {
        private readonly string _databaseName = $"SugarcaneLogin_{Guid.NewGuid():N}";
        private string _connectionString = string.Empty;

        public Task InitializeAsync()
        {
            if (string.IsNullOrWhiteSpace(SqlServerFixture.BaseConnectionString)) return Task.CompletedTask;
            _connectionString = new SqlConnectionStringBuilder(SqlServerFixture.BaseConnectionString)
            { InitialCatalog = _databaseName }.ConnectionString;
            return Task.CompletedTask;
        }

        protected override IHost CreateHost(IHostBuilder builder)
        {
            builder.ConfigureHostConfiguration(config => config.AddInMemoryCollection(new Dictionary<string, string?>
            {
                ["ConnectionStrings:SugarcanePlanning"] = _connectionString,
                ["Database:MigrateOnStartup"] = "true",
                ["Database:SeedSampleData"] = "true",
                ["Jwt:SigningKey"] = "login-lockout-test-signing-key-long-enough-1234567890"
            }));
            return base.CreateHost(builder);
        }

        async Task IAsyncLifetime.DisposeAsync()
        {
            await base.DisposeAsync();
            if (string.IsNullOrWhiteSpace(SqlServerFixture.BaseConnectionString)) return;

            var master = new SqlConnectionStringBuilder(SqlServerFixture.BaseConnectionString)
            { InitialCatalog = "master" }.ConnectionString;
            await using var connection = new SqlConnection(master);
            await connection.OpenAsync();
            await using var command = connection.CreateCommand();
            command.CommandText = $"IF DB_ID('{_databaseName}') IS NOT NULL BEGIN " +
                                  $"ALTER DATABASE [{_databaseName}] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; " +
                                  $"DROP DATABASE [{_databaseName}]; END";
            await command.ExecuteNonQueryAsync();
        }
    }
}
