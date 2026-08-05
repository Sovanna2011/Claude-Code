using ErpS4.Api.Security;
using ErpS4.Application.TableBrowser;
using Microsoft.AspNetCore.Mvc;

namespace ErpS4.Api.Endpoints;

/// <summary>SE16N - the table browser.</summary>
public static class TableBrowserEndpoints
{
    public static IEndpointRouteBuilder MapTableBrowserEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/v1/table-browser")
            .WithTags("Table browser")
            .RequireAuthorization(Policies.Permission(Policies.ReadTableBrowser));

        group.MapGet("/tables", TablesAsync)
            .WithSummary("Tables this user may browse")
            .WithDescription("System-protected tables are absent from the list, not merely refused.")
            .Produces<IReadOnlyList<BrowsableTable>>();

        group.MapPost("/query", QueryAsync)
            .WithSummary("Run a browser query")
            .WithDescription(
                "Table and field names are resolved through the data dictionary and every " +
                "value the caller supplies becomes a parameter, so a filter cannot become " +
                "part of the statement. Results are capped and every query is logged.")
            .Produces<BrowserQueryResult>()
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapPost("/export", ExportAsync)
            .WithSummary("Run a browser query for export")
            .WithDescription(
                "Taking data out is a separate act from looking at it: it needs its own " +
                "permission and is logged as an export.")
            .RequireAuthorization(Policies.Permission(Policies.ExportTableBrowser))
            .Produces<BrowserQueryResult>()
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        return app;
    }

    private static async Task<IResult> TablesAsync(
        ITableBrowserService browser,
        [FromQuery] string? search,
        CancellationToken cancellationToken) =>
        Results.Ok(await browser.GetBrowsableTablesAsync(search, cancellationToken));

    private static async Task<IResult> QueryAsync(
        [FromBody] BrowserQueryRequest request,
        ITableBrowserService browser,
        IOrganizationalAccessGuard guard,
        CancellationToken cancellationToken) =>
        await RunAsync(request with { IsExport = false }, browser, guard, cancellationToken);

    private static async Task<IResult> ExportAsync(
        [FromBody] BrowserQueryRequest request,
        ITableBrowserService browser,
        IOrganizationalAccessGuard guard,
        CancellationToken cancellationToken) =>
        await RunAsync(request with { IsExport = true }, browser, guard, cancellationToken);

    private static async Task<IResult> RunAsync(
        BrowserQueryRequest request,
        ITableBrowserService browser,
        IOrganizationalAccessGuard guard,
        CancellationToken cancellationToken)
    {
        // Narrowing to a company code still has to be a company code the user
        // may see - otherwise the browser would hand out what the postings
        // endpoints refuse.
        if (!string.IsNullOrWhiteSpace(request.CompanyCode))
        {
            await guard.EnsureCompanyCodeAsync(request.CompanyCode, "Read", cancellationToken);
        }

        var result = await browser.QueryAsync(request, cancellationToken);

        return result.IsSuccess
            ? Results.Ok(result)
            : Results.Problem(
                title: "The query was refused",
                detail: result.Violations.FirstOrDefault()?.Message,
                statusCode: StatusCodes.Status422UnprocessableEntity,
                type: "https://errors.erps4.local/"
                      + (result.Violations.FirstOrDefault()?.Code ?? "SE16N.REFUSED"),
                extensions: new Dictionary<string, object?>
                {
                    ["errorCode"] = result.Violations.FirstOrDefault()?.Code,
                    ["errors"] = result.Violations
                        .Select(v => new { v.Code, v.Message, v.Field }).ToArray(),
                });
    }
}
