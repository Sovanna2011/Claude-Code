using Microsoft.AspNetCore.StaticFiles;

// Static file server for the published Blazor WebAssembly client. It deliberately runs on its
// own port, so the test stack reproduces the real topology: the client and the API are separate
// origins, and every cross-origin rule — including the exposed Content-Disposition header that
// gives exported reports their real file names — is exercised rather than bypassed.

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

// A .wasm served as application/octet-stream fails to instantiate in the browser, and the
// runtime's own provider does not map every extension Blazor publishes.
var contentTypes = new FileExtensionContentTypeProvider();
contentTypes.Mappings[".wasm"] = "application/wasm";
contentTypes.Mappings[".blat"] = "application/octet-stream";
contentTypes.Mappings[".dat"] = "application/octet-stream";

// index.html names which build the browser loads, so caching it would keep serving the previous
// build's assets after a rebuild.
const string NoStore = "no-cache, no-store, must-revalidate";

// Everything else is revalidated rather than cached outright. The published assemblies keep
// stable names (dotnet.js, SugarcanePlanning.Client.wasm — see the client csproj for why
// fingerprinting is off), so an immutable long max-age would keep serving the previous
// deployment's code. "no-cache" still allows a 304 on an unchanged file, so a warm browser
// re-downloads nothing; it just always asks.
const string Revalidate = "no-cache";

static bool IsEntryPoint(string path) =>
    path.EndsWith("/index.html", StringComparison.OrdinalIgnoreCase) || path == "/";

app.UseDefaultFiles();
app.UseStaticFiles(new StaticFileOptions
{
    ContentTypeProvider = contentTypes,
    OnPrepareResponse = context => context.Context.Response.Headers.CacheControl =
        IsEntryPoint(context.Context.Request.Path.Value ?? string.Empty) ? NoStore : Revalidate
});

// The client owns its routes: /projections and /schedule are not files on disk, so anything not
// found has to fall back to index.html or a refresh returns 404. The fallback needs its own
// options — it serves the file through its own endpoint, so the settings above do not reach it,
// which is exactly how index.html ends up with no cache header at all if this is left default.
app.MapFallbackToFile("index.html", new StaticFileOptions
{
    ContentTypeProvider = contentTypes,
    OnPrepareResponse = context => context.Context.Response.Headers.CacheControl = NoStore
});

app.Run();
