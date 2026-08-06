using ErpS4.Api.Contracts;
using ErpS4.Api.Security;
using ErpS4.Application.Clearing;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Api.Endpoints;

/// <summary>Payment and clearing endpoints - F-28, F-53 and the open item list.</summary>
public static class PaymentEndpoints
{
    public static IEndpointRouteBuilder MapPaymentEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/v1")
            .WithTags("Payments and clearing")
            .RequireAuthorization();

        group.MapPost("/payments", ClearAsync)
            .WithSummary("Post a payment and clear open items (F-28 / F-53)")
            .WithDescription(
                "Builds a payment document, posts it through the central engine and links " +
                "it to the items it settles. Partial payments, residuals and cash discount " +
                "are handled per allocation.")
            .RequireAuthorization(Policies.Permission(Policies.PostJournalEntry))
            .Produces<ClearingResult>(StatusCodes.Status201Created)
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapPost("/clearing/{fiscalYear:int}/{clearingDocument}/reset", ResetAsync)
            .WithSummary("Reset a clearing")
            .WithDescription(
                "Puts the items back to open. The payment document is untouched - reverse " +
                "it separately if it should not exist.")
            .RequireAuthorization(Policies.Permission(Policies.ReverseJournalEntry))
            .Produces<ResetResult>()
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapGet("/open-items", GetOpenItemsAsync)
            .WithSummary("Open items (FBL5N / FBL1N)")
            .RequireAuthorization(Policies.Permission(Policies.ReadFinanceReport))
            .Produces<PagedResult<OpenItemSummary>>();

        return app;
    }

    private static async Task<IResult> ClearAsync(
        [FromBody] ClearingRequest request,
        [FromHeader(Name = "Idempotency-Key")] Guid? idempotencyKey,
        IClearingService clearing,
        IOrganizationalAccessGuard accessGuard,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(request.CompanyCode, "Post", cancellationToken);

        var result = await clearing.ClearAsync(
            request with { IdempotencyKey = idempotencyKey ?? request.IdempotencyKey },
            cancellationToken);

        if (result.IsSuccess)
        {
            return Results.Created(
                $"/api/v1/journal-entries/2026/{result.PaymentDocumentNumber}", result);
        }

        return Results.Problem(
            title: "The payment could not be cleared",
            detail: $"{result.Violations.Count} rule(s) refused this clearing.",
            statusCode: StatusCodes.Status422UnprocessableEntity,
            type: "https://errors.erps4.local/CLR.REJECTED",
            extensions: new Dictionary<string, object?>
            {
                ["errorCode"] = "CLR.REJECTED",
                ["errors"] = result.Violations
                    .Select(v => new { v.Code, v.Message, v.Field, Severity = v.Severity.ToString() })
                    .ToArray(),
                ["postingErrors"] = result.PostingErrors
                    .Select(e => new { e.Code, e.Message, e.Field, e.LineNumber })
                    .ToArray(),
            });
    }

    private static async Task<IResult> ResetAsync(
        int fiscalYear,
        string clearingDocument,
        [FromBody] ResetClearingBody body,
        IClearingService clearing,
        IOrganizationalAccessGuard accessGuard,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(body.CompanyCode, "Post", cancellationToken);

        var result = await clearing.ResetAsync(
            new ResetClearingRequest(
                body.CompanyCode, (short)fiscalYear, clearingDocument, body.Reason),
            cancellationToken);

        return result.IsSuccess
            ? Results.Ok(result)
            : Results.Problem(
                title: "The clearing could not be reset",
                statusCode: StatusCodes.Status422UnprocessableEntity,
                type: "https://errors.erps4.local/CLR.RESET_REJECTED",
                extensions: new Dictionary<string, object?>
                {
                    ["errorCode"] = "CLR.RESET_REJECTED",
                    ["errors"] = result.Violations
                        .Select(v => new { v.Code, v.Message, v.Field })
                        .ToArray(),
                });
    }

    private static async Task<IResult> GetOpenItemsAsync(
        [AsParameters] OpenItemQuery query,
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

        var items =
            from item in context.Query<OpenItem>().AsNoTracking()
            join company in context.Query<CompanyCode>() on item.CompanyCodeId equals company.Id
            join partner in context.Query<BusinessPartner>()
                on item.BusinessPartnerId equals partner.Id into partners
            from partner in partners.DefaultIfEmpty()
            where item.TenantId == tenantId
                  && company.CompanyCodeKey == query.CompanyCode
                  && (query.PartnerNumber == null || partner.PartnerNumber == query.PartnerNumber)
                  && (query.AccountType == null || item.AccountType == query.AccountType)
                  && (query.IncludeCleared == true || item.Status != "Cleared")
                  && (query.DueBy == null || item.DueDate <= query.DueBy)
            select new { item, partner };

        var totalCount = request.IncludeTotalCount
            ? await items.CountAsync(cancellationToken)
            : (int?)null;

        // Oldest due first: that is the order a collections clerk works in.
        var page = await items
            .OrderBy(x => x.item.DueDate)
            .ThenBy(x => x.item.Id)
            .Skip(request.Skip)
            .Take(request.PageSize + 1)
            .Select(x => new OpenItemSummary(
                x.item.DocumentNumber,
                x.item.FiscalYear,
                x.item.LineItemNumber,
                x.item.AccountType,
                x.partner != null ? x.partner.PartnerNumber : null,
                x.partner != null ? x.partner.FullName : null,
                x.item.PostingDate,
                x.item.DueDate,
                x.item.DocumentCurrencyCode,
                x.item.OriginalAmountInDocumentCurrency,
                x.item.OpenAmountInDocumentCurrency,
                x.item.Status,
                x.item.DunningLevel,
                x.item.ClearingDocumentNumber))
            .ToListAsync(cancellationToken);

        var hasNextPage = page.Count > request.PageSize;
        if (hasNextPage)
        {
            page.RemoveAt(page.Count - 1);
        }

        return Results.Ok(PagedResult<OpenItemSummary>.Create(
            page, request, totalCount, hasNextPage));
    }
}

public sealed record ResetClearingBody(string CompanyCode, string Reason);

public sealed record OpenItemQuery
{
    [FromQuery] public required string CompanyCode { get; init; }

    [FromQuery] public string? PartnerNumber { get; init; }

    /// <summary>`D` for customers, `K` for suppliers.</summary>
    [FromQuery] public string? AccountType { get; init; }

    [FromQuery] public DateOnly? DueBy { get; init; }

    [FromQuery] public bool? IncludeCleared { get; init; }

    [FromQuery] public int? PageNumber { get; init; }

    [FromQuery] public int? PageSize { get; init; }

    [FromQuery] public bool? IncludeTotalCount { get; init; }
}

public sealed record OpenItemSummary(
    string DocumentNumber,
    short FiscalYear,
    int LineItemNumber,
    string AccountType,
    string? PartnerNumber,
    string? PartnerName,
    DateOnly PostingDate,
    DateOnly? DueDate,
    string CurrencyCode,
    decimal OriginalAmount,
    decimal OpenAmount,
    string Status,
    byte DunningLevel,
    string? ClearingDocumentNumber);
