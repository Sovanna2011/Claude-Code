using ErpS4.Api.Security;
using ErpS4.Application.DataDictionary;
using Microsoft.AspNetCore.Mvc;

namespace ErpS4.Api.Endpoints;

/// <summary>SE11 - the data dictionary, read only.</summary>
public static class DictionaryEndpoints
{
    public static IEndpointRouteBuilder MapDictionaryEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/v1/dictionary")
            .WithTags("Data dictionary")
            .RequireAuthorization(Policies.Permission(Policies.ReadDictionary));

        group.MapGet("/objects", SearchAsync)
            .WithSummary("Repository browser")
            .WithDescription("Dictionary objects, optionally filtered by type and name.")
            .Produces<IReadOnlyList<DictionaryObjectSummary>>();

        group.MapGet("/tables/{schemaName}/{tableName}", GetTableAsync)
            .WithSummary("Table definition")
            .WithDescription("Fields, keys, indexes and foreign keys, as SE11 shows them.")
            .Produces<TableDefinition>()
            .ProducesProblem(StatusCodes.Status404NotFound);

        group.MapGet("/tables/{schemaName}/{tableName}/ddl", GetDdlAsync)
            .WithSummary("Generated DDL")
            .WithDescription(
                "The CREATE TABLE the dictionary describes, rendered for reading. It is " +
                "generated here and executed nowhere: changing the database is a migration, " +
                "not a preview.")
            .Produces<string>(contentType: "text/plain")
            .ProducesProblem(StatusCodes.Status404NotFound);

        group.MapGet("/tables/{schemaName}/{tableName}/where-used", GetWhereUsedAsync)
            .WithSummary("Where-used list")
            .WithDescription("The foreign keys that point at this table.")
            .Produces<IReadOnlyList<UsageReference>>();

        group.MapGet("/domains/{domainName}", GetDomainAsync)
            .WithSummary("Domain and its fixed values")
            .Produces<DomainDefinition>()
            .ProducesProblem(StatusCodes.Status404NotFound);

        group.MapGet("/data-elements/{dataElementName}", GetDataElementAsync)
            .WithSummary("Data element and the domain behind it")
            .Produces<DataElementDefinition>()
            .ProducesProblem(StatusCodes.Status404NotFound);

        return app;
    }

    private static async Task<IResult> SearchAsync(
        IDictionaryService dictionary,
        [FromQuery] string? objectType,
        [FromQuery] string? search,
        [FromQuery] int? maxResults,
        CancellationToken cancellationToken) =>
        Results.Ok(await dictionary.SearchObjectsAsync(
            objectType, search, maxResults ?? 200, cancellationToken));

    private static async Task<IResult> GetTableAsync(
        string schemaName,
        string tableName,
        IDictionaryService dictionary,
        CancellationToken cancellationToken) =>
        await dictionary.GetTableAsync(schemaName, tableName, cancellationToken) is { } table
            ? Results.Ok(table)
            : NotFound($"{schemaName}.{tableName}");

    private static async Task<IResult> GetDdlAsync(
        string schemaName,
        string tableName,
        IDictionaryService dictionary,
        CancellationToken cancellationToken) =>
        await dictionary.GenerateCreateTableSqlAsync(schemaName, tableName, cancellationToken)
                is { } sql
            ? Results.Text(sql, "text/plain")
            : NotFound($"{schemaName}.{tableName}");

    private static async Task<IResult> GetWhereUsedAsync(
        string schemaName,
        string tableName,
        IDictionaryService dictionary,
        CancellationToken cancellationToken) =>
        Results.Ok(await dictionary.GetWhereUsedAsync(schemaName, tableName, cancellationToken));

    private static async Task<IResult> GetDomainAsync(
        string domainName,
        IDictionaryService dictionary,
        CancellationToken cancellationToken) =>
        await dictionary.GetDomainAsync(domainName, cancellationToken) is { } domain
            ? Results.Ok(domain)
            : NotFound(domainName);

    private static async Task<IResult> GetDataElementAsync(
        string dataElementName,
        IDictionaryService dictionary,
        CancellationToken cancellationToken) =>
        await dictionary.GetDataElementAsync(dataElementName, cancellationToken) is { } element
            ? Results.Ok(element)
            : NotFound(dataElementName);

    private static IResult NotFound(string name) =>
        Results.Problem(
            title: "Not in the data dictionary",
            detail: $"{name} does not exist in the dictionary for this tenant.",
            statusCode: StatusCodes.Status404NotFound,
            type: "https://errors.erps4.local/SE11.NOT_FOUND");
}
