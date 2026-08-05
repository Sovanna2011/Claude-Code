using Microsoft.AspNetCore.Components.Authorization;
using Microsoft.AspNetCore.Components.Web;
using Microsoft.AspNetCore.Components.WebAssembly.Hosting;
using SugarcanePlanning.Client;
using SugarcanePlanning.Client.Services;

var builder = WebAssemblyHostBuilder.CreateDefault(args);
builder.RootComponents.Add<App>("#app");
builder.RootComponents.Add<HeadOutlet>("head::after");

// Base address of the Web API; override it in wwwroot/appsettings.json per environment.
var apiBaseUrl = builder.Configuration["ApiBaseUrl"] ?? "https://localhost:7100/";
builder.Services.AddScoped(_ => new HttpClient { BaseAddress = new Uri(apiBaseUrl) });

builder.Services.AddScoped<TokenStore>();
builder.Services.AddScoped<ApiClient>();
builder.Services.AddScoped<PlanningApi>();
builder.Services.AddScoped<AuthService>();

builder.Services.AddScoped<AuthenticationStateProvider, JwtAuthenticationStateProvider>();
builder.Services.AddScoped(sp => (JwtAuthenticationStateProvider)sp.GetRequiredService<AuthenticationStateProvider>());
builder.Services.AddAuthorizationCore();

await builder.Build().RunAsync();
