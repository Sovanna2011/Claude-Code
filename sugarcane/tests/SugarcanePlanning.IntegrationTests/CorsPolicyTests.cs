using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// The Blazor client is served from a different origin to the API, so anything the browser
/// must read off a response has to be listed in <c>Access-Control-Expose-Headers</c>. Without
/// it every exported report is saved as "report.pdf" instead of its real name — a defect no
/// server-side test can see, because the header is present on the wire either way.
/// </summary>
public class CorsPolicyTests : IClassFixture<CorsPolicyTests.ApiFactory>
{
    private readonly ApiFactory _factory;

    public CorsPolicyTests(ApiFactory factory) => _factory = factory;

    [Fact]
    public async Task Content_Disposition_is_exposed_to_the_browser()
    {
        var client = _factory.CreateClient();

        // An unauthenticated call is enough: the CORS middleware runs before authorisation and
        // never touches the database, so no SQL Server is needed to prove the policy.
        using var request = new HttpRequestMessage(HttpMethod.Get, "/api/reports/PlantingProjection/export?format=Pdf");
        request.Headers.Add("Origin", "http://localhost:5150");

        var response = await client.SendAsync(request);

        Assert.True(response.Headers.TryGetValues("Access-Control-Expose-Headers", out var exposed),
            "the CORS policy must expose headers to the client");
        Assert.Contains(exposed!, value => value.Contains("Content-Disposition", StringComparison.OrdinalIgnoreCase));
    }

    [Fact]
    public async Task A_request_from_an_unknown_origin_is_not_granted_access()
    {
        var client = _factory.CreateClient();

        using var request = new HttpRequestMessage(HttpMethod.Get, "/api/companies");
        request.Headers.Add("Origin", "https://not-our-client.example");

        var response = await client.SendAsync(request);

        Assert.False(response.Headers.Contains("Access-Control-Allow-Origin"));
    }

    [Fact]
    public async Task The_preflight_allows_the_client_origin()
    {
        var client = _factory.CreateClient();

        using var request = new HttpRequestMessage(HttpMethod.Options, "/api/companies");
        request.Headers.Add("Origin", "http://localhost:5150");
        request.Headers.Add("Access-Control-Request-Method", "GET");
        request.Headers.Add("Access-Control-Request-Headers", "authorization");

        var response = await client.SendAsync(request);

        Assert.True(response.Headers.TryGetValues("Access-Control-Allow-Origin", out var origins));
        Assert.Contains("http://localhost:5150", origins!);
    }

    /// <summary>Hosts the real API without touching a database: no migration, no seeding.</summary>
    public class ApiFactory : WebApplicationFactory<Program>
    {
        protected override IHost CreateHost(IHostBuilder builder)
        {
            builder.ConfigureHostConfiguration(config => config.AddInMemoryCollection(new Dictionary<string, string?>
            {
                ["Database:MigrateOnStartup"] = "false",
                ["Database:SeedSampleData"] = "false",
                ["Jwt:SigningKey"] = "cors-test-signing-key-long-enough-1234567890",
                ["ConnectionStrings:SugarcanePlanning"] = "Server=(unused);Database=None;Trusted_Connection=True"
            }));
            return base.CreateHost(builder);
        }
    }
}
