using ErpS4.Api.Contracts;
using ErpS4.Api.Security;
using ErpS4.Application.Assets;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Api.Endpoints;

/// <summary>Asset accounting endpoints - F-90, ABAVN, AFAB, AS03.</summary>
public static class AssetEndpoints
{
    public static IEndpointRouteBuilder MapAssetEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/v1")
            .WithTags("Asset accounting")
            .RequireAuthorization();

        group.MapPost("/assets/{assetNumber}/acquisitions", AcquireAsync)
            .WithSummary("Capitalise an asset (F-90)")
            .RequireAuthorization(Policies.Permission(Policies.PostAsset))
            .Produces<AssetPostingResult>(StatusCodes.Status201Created)
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapPost("/assets/{assetNumber}/retirement", RetireAsync)
            .WithSummary("Retire an asset, with or without revenue (ABAVN)")
            .RequireAuthorization(Policies.Permission(Policies.PostAsset))
            .Produces<AssetPostingResult>(StatusCodes.Status201Created)
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapPost("/depreciation-runs/preview", PreviewAsync)
            .WithSummary("Plan depreciation without posting")
            .WithDescription("The same calculation the run uses, so the preview cannot disagree.")
            .RequireAuthorization(Policies.Permission(Policies.ReadAsset))
            .Produces<DepreciationPlan>();

        group.MapPost("/depreciation-runs", RunAsync)
            .WithSummary("Run and post depreciation (AFAB)")
            .WithDescription("A planned run happens once per period; re-running needs RunType 'Repeat'.")
            .RequireAuthorization(Policies.Permission(Policies.PostAsset))
            .Produces<DepreciationRunResult>(StatusCodes.Status201Created)
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapGet("/assets", ListAsync)
            .WithSummary("Asset register (AS03)")
            .RequireAuthorization(Policies.Permission(Policies.ReadAsset))
            .Produces<PagedResult<AssetSummary>>();

        return app;
    }

    private static async Task<IResult> AcquireAsync(
        string assetNumber,
        [FromBody] AssetAcquisitionBody body,
        [FromHeader(Name = "Idempotency-Key")] Guid? idempotencyKey,
        IAssetService assets,
        IOrganizationalAccessGuard accessGuard,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(body.CompanyCode, "Post", cancellationToken);

        var result = await assets.AcquireAsync(
            new AssetAcquisitionRequest(
                body.CompanyCode,
                assetNumber,
                body.AssetSubNumber,
                body.Amount,
                body.CurrencyCode,
                body.PostingDate,
                body.OffsettingAccount)
            {
                VendorPartnerNumber = body.VendorPartnerNumber,
                AssetValueDate = body.AssetValueDate,
                Reference = body.Reference,
                Text = body.Text,
                IdempotencyKey = idempotencyKey,
            },
            cancellationToken);

        return AssetResponse(result, $"/api/v1/assets?companyCode={body.CompanyCode}");
    }

    private static async Task<IResult> RetireAsync(
        string assetNumber,
        [FromBody] AssetRetirementBody body,
        [FromHeader(Name = "Idempotency-Key")] Guid? idempotencyKey,
        IAssetService assets,
        IOrganizationalAccessGuard accessGuard,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(body.CompanyCode, "Post", cancellationToken);

        var result = await assets.RetireAsync(
            new AssetRetirementRequest(
                body.CompanyCode,
                assetNumber,
                body.AssetSubNumber,
                body.PostingDate,
                body.RevenueAmount,
                body.CurrencyCode)
            {
                CustomerPartnerNumber = body.CustomerPartnerNumber,
                RevenueOffsettingAccount = body.RevenueOffsettingAccount,
                Text = body.Text,
                IdempotencyKey = idempotencyKey,
            },
            cancellationToken);

        return AssetResponse(result, $"/api/v1/assets?companyCode={body.CompanyCode}");
    }

    private static async Task<IResult> PreviewAsync(
        [FromBody] DepreciationRunBody body,
        IAssetService assets,
        IOrganizationalAccessGuard accessGuard,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(body.CompanyCode, "Read", cancellationToken);

        return Results.Ok(await assets.PlanDepreciationAsync(
            new DepreciationRunRequest(body.CompanyCode, body.FiscalYear, body.FiscalPeriod)
            {
                AssetNumber = body.AssetNumber,
            },
            cancellationToken));
    }

    private static async Task<IResult> RunAsync(
        [FromBody] DepreciationRunBody body,
        [FromHeader(Name = "Idempotency-Key")] Guid? idempotencyKey,
        IAssetService assets,
        IOrganizationalAccessGuard accessGuard,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(body.CompanyCode, "Post", cancellationToken);

        var result = await assets.RunDepreciationAsync(
            new DepreciationRunRequest(body.CompanyCode, body.FiscalYear, body.FiscalPeriod)
            {
                RunType = body.RunType ?? "Planned",
                AssetNumber = body.AssetNumber,
                IdempotencyKey = idempotencyKey,
            },
            cancellationToken);

        if (result.IsSuccess)
        {
            return Results.Created("/api/v1/depreciation-runs", result);
        }

        return Results.Problem(
            title: "The depreciation run was refused",
            statusCode: StatusCodes.Status422UnprocessableEntity,
            type: "https://errors.erps4.local/AA.RUN_REJECTED",
            extensions: new Dictionary<string, object?>
            {
                ["errorCode"] = "AA.RUN_REJECTED",
                ["errors"] = result.Violations.Select(v => new { v.Code, v.Message, v.Field }).ToArray(),
                ["postingErrors"] = result.PostingErrors
                    .Select(e => new { e.Code, e.Message, e.Field, e.LineNumber }).ToArray(),
            });
    }

    private static async Task<IResult> ListAsync(
        [AsParameters] AssetQuery query,
        IErpDataContext context,
        ITenantProvider tenantProvider,
        IOrganizationalAccessGuard accessGuard,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(query.CompanyCode, "Read", cancellationToken);

        var tenantId = tenantProvider.TenantId;
        var request = new PageRequest
        {
            PageNumber = query.PageNumber ?? 1,
            PageSize = query.PageSize ?? PageRequest.DefaultPageSize,
            IncludeTotalCount = query.IncludeTotalCount ?? false,
        };

        var assets =
            from asset in context.Query<Asset>().AsNoTracking()
            join company in context.Query<CompanyCode>() on asset.CompanyCodeId equals company.Id
            join assetClass in context.Query<AssetClass>() on asset.AssetClassId equals assetClass.Id
            where asset.TenantId == tenantId
                  && company.CompanyCodeKey == query.CompanyCode
                  && (query.AssetClass == null || assetClass.AssetClassCode == query.AssetClass)
                  && (query.IncludeRetired == true
                      || (asset.Status != "Retired" && asset.Status != "Sold"
                          && asset.Status != "Scrapped"))
            select new { asset, assetClass };

        var totalCount = request.IncludeTotalCount
            ? await assets.CountAsync(cancellationToken)
            : (int?)null;

        var page = await assets
            .OrderBy(x => x.asset.AssetNumber)
            .ThenBy(x => x.asset.AssetSubNumber)
            .Skip(request.Skip)
            .Take(request.PageSize + 1)
            .Select(x => new AssetSummary(
                x.asset.AssetNumber,
                x.asset.AssetSubNumber,
                x.assetClass.AssetClassCode,
                x.asset.Description,
                x.asset.CapitalizationDate,
                x.asset.Status,
                context.Query<AssetValue>()
                    .Where(v => v.AssetId == x.asset.Id && v.FiscalYear == query.FiscalYear)
                    .Select(v => (decimal?)v.NetBookValue)
                    .FirstOrDefault()))
            .ToListAsync(cancellationToken);

        var hasNextPage = page.Count > request.PageSize;
        if (hasNextPage)
        {
            page.RemoveAt(page.Count - 1);
        }

        return Results.Ok(PagedResult<AssetSummary>.Create(page, request, totalCount, hasNextPage));
    }

    private static IResult AssetResponse(AssetPostingResult result, string location) =>
        result.IsSuccess
            ? Results.Created(location, result)
            : Results.Problem(
                title: "The asset posting was refused",
                statusCode: StatusCodes.Status422UnprocessableEntity,
                type: "https://errors.erps4.local/AA.REJECTED",
                extensions: new Dictionary<string, object?>
                {
                    ["errorCode"] = "AA.REJECTED",
                    ["errors"] = result.Violations
                        .Select(v => new { v.Code, v.Message, v.Field }).ToArray(),
                    ["postingErrors"] = result.PostingErrors
                        .Select(e => new { e.Code, e.Message, e.Field, e.LineNumber }).ToArray(),
                });
}

public sealed record AssetAcquisitionBody(
    string CompanyCode,
    decimal Amount,
    string CurrencyCode,
    DateOnly PostingDate,
    string OffsettingAccount)
{
    public int AssetSubNumber { get; init; }

    public string? VendorPartnerNumber { get; init; }

    public DateOnly? AssetValueDate { get; init; }

    public string? Reference { get; init; }

    public string? Text { get; init; }
}

public sealed record AssetRetirementBody(
    string CompanyCode,
    DateOnly PostingDate,
    decimal RevenueAmount,
    string CurrencyCode)
{
    public int AssetSubNumber { get; init; }

    public string? CustomerPartnerNumber { get; init; }

    public string? RevenueOffsettingAccount { get; init; }

    public string? Text { get; init; }
}

public sealed record DepreciationRunBody(
    string CompanyCode,
    short FiscalYear,
    byte FiscalPeriod)
{
    public string? RunType { get; init; }

    public string? AssetNumber { get; init; }
}

public sealed record AssetQuery
{
    [FromQuery] public required string CompanyCode { get; init; }

    [FromQuery] public short FiscalYear { get; init; }

    [FromQuery] public string? AssetClass { get; init; }

    [FromQuery] public bool? IncludeRetired { get; init; }

    [FromQuery] public int? PageNumber { get; init; }

    [FromQuery] public int? PageSize { get; init; }

    [FromQuery] public bool? IncludeTotalCount { get; init; }
}

public sealed record AssetSummary(
    string AssetNumber,
    int AssetSubNumber,
    string AssetClass,
    string Description,
    DateOnly? CapitalizationDate,
    string Status,
    decimal? NetBookValue);
