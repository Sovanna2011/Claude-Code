using ErpS4.Api.Contracts;
using ErpS4.Api.Middleware;
using ErpS4.Api.Security;
using ErpS4.Application.Posting;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Api.Endpoints;

/// <summary>Journal entry endpoints - FB50, FB03, FB08 and the line item list.</summary>
public static class JournalEntryEndpoints
{
    /// <summary>Fields a caller may sort the line item list by.</summary>
    private static readonly HashSet<string> SortableLineFields = new(StringComparer.OrdinalIgnoreCase)
    {
        "PostingDate", "DocumentNumber", "GLAccount", "AmountInLocalCurrency", "DueDate",
    };

    public static IEndpointRouteBuilder MapJournalEntryEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/v1/journal-entries")
            .WithTags("Journal entries")
            .RequireAuthorization();

        group.MapPost("/", PostAsync)
            .WithSummary("Post a journal entry (FB50)")
            .WithDescription(
                "Validates and posts. Send an Idempotency-Key header: a retry with the " +
                "same key returns the document the first call created.")
            .RequireAuthorization(Policies.Permission(Policies.PostJournalEntry))
            .Produces<PostingResponse>(StatusCodes.Status201Created)
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapPost("/simulate", SimulateAsync)
            .WithSummary("Simulate a journal entry")
            .WithDescription("Runs every rule and returns the accounting impact without posting.")
            .RequireAuthorization(Policies.Permission(Policies.CreateJournalEntry))
            .Produces<PostingResponse>()
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapPost("/{fiscalYear:int}/{documentNumber}/reversal", ReverseAsync)
            .WithSummary("Reverse a posted document (FB08)")
            .RequireAuthorization(Policies.Permission(Policies.ReverseJournalEntry))
            .Produces<PostingResponse>(StatusCodes.Status201Created)
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapGet("/{fiscalYear:int}/{documentNumber}", GetAsync)
            .WithSummary("Display a document (FB03)")
            .RequireAuthorization(Policies.Permission(Policies.ReadFinanceReport))
            .Produces<JournalEntryDocument>()
            .ProducesProblem(StatusCodes.Status404NotFound);

        group.MapGet("/line-items", GetLineItemsAsync)
            .WithSummary("G/L line items (FBL3N)")
            .WithDescription("Server-side paged; sorting is restricted to an allow list.")
            .RequireAuthorization(Policies.Permission(Policies.ReadFinanceReport))
            .Produces<PagedResult<JournalEntryLineItem>>();

        return app;
    }

    private static async Task<IResult> PostAsync(
        [FromBody] PostJournalEntryCommand command,
        [FromHeader(Name = "Idempotency-Key")] Guid? idempotencyKey,
        PostJournalEntryHandler handler,
        IOrganizationalAccessGuard accessGuard,
        HttpContext context,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(command.CompanyCode, "Post", cancellationToken);

        var result = await handler.HandleAsync(
            command with { IdempotencyKey = idempotencyKey ?? command.IdempotencyKey, Simulate = false },
            cancellationToken);

        if (!result.IsSuccess)
        {
            return RejectionProblem(result, context);
        }

        var response = PostingResponse.From(result);

        // A retry is not a creation: 200 tells the caller the work was already
        // done, 201 that this call did it.
        return result.WasAlreadyPosted
            ? Results.Ok(response)
            : Results.Created(
                $"/api/v1/journal-entries/{result.FiscalYear}/{result.DocumentNumber}", response);
    }

    private static async Task<IResult> SimulateAsync(
        [FromBody] PostJournalEntryCommand command,
        PostJournalEntryHandler handler,
        IOrganizationalAccessGuard accessGuard,
        HttpContext context,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(command.CompanyCode, "Write", cancellationToken);

        var result = await handler.HandleAsync(
            command with { Simulate = true }, cancellationToken);

        return result.IsSuccess
            ? Results.Ok(PostingResponse.From(result))
            : RejectionProblem(result, context);
    }

    private static async Task<IResult> ReverseAsync(
        int fiscalYear,
        string documentNumber,
        [FromBody] ReverseDocumentRequest request,
        [FromHeader(Name = "Idempotency-Key")] Guid? idempotencyKey,
        IPostingEngine postingEngine,
        IOrganizationalAccessGuard accessGuard,
        HttpContext context,
        CancellationToken cancellationToken)
    {
        await accessGuard.EnsureCompanyCodeAsync(request.CompanyCode, "Post", cancellationToken);

        var result = await postingEngine.ReverseAsync(
            new ReversalRequest(
                documentNumber,
                (short)fiscalYear,
                request.CompanyCode,
                request.ReasonCode,
                request.PostingDate,
                idempotencyKey),
            cancellationToken);

        return result.IsSuccess
            ? Results.Created(
                $"/api/v1/journal-entries/{result.FiscalYear}/{result.DocumentNumber}",
                PostingResponse.From(result))
            : RejectionProblem(result, context);
    }

    private static async Task<IResult> GetAsync(
        int fiscalYear,
        string documentNumber,
        IErpDataContext context,
        ITenantProvider tenantProvider,
        IOrganizationalAccessGuard accessGuard,
        CancellationToken cancellationToken)
    {
        var tenantId = tenantProvider.TenantId;

        var header = await (
            from h in context.Query<JournalEntryHeader>().AsNoTracking()
            join company in context.Query<CompanyCode>() on h.CompanyCodeId equals company.Id
            where h.TenantId == tenantId
                  && h.FiscalYear == fiscalYear
                  && h.DocumentNumber == documentNumber
            select new { Header = h, company.CompanyCodeKey }
        ).FirstOrDefaultAsync(cancellationToken);

        if (header is null)
        {
            return Results.NotFound();
        }

        await accessGuard.EnsureCompanyCodeAsync(header.CompanyCodeKey, "Read", cancellationToken);

        var lines = await context.Query<JournalEntryLine>()
            .AsNoTracking()
            .Where(l => l.TenantId == tenantId && l.JournalEntryHeaderId == header.Header.Id)
            .OrderBy(l => l.LineItemNumber)
            .Select(l => new JournalEntryLineItem(
                l.DocumentNumber,
                l.FiscalYear,
                l.LineItemNumber,
                l.PostingDate,
                l.PostingKey,
                l.DebitCreditIndicator,
                l.GLAccount,
                l.AmountInDocumentCurrency,
                l.DocumentCurrencyCode,
                l.AmountInLocalCurrency,
                l.LocalCurrencyCode,
                l.AssignmentReference,
                l.LineItemText,
                l.ClearingStatus,
                l.DueDate))
            .ToListAsync(cancellationToken);

        return Results.Ok(new JournalEntryDocument(
            header.Header.DocumentNumber,
            header.Header.FiscalYear,
            header.CompanyCodeKey,
            header.Header.PostingDate,
            header.Header.DocumentDate,
            header.Header.DocumentCurrencyCode,
            header.Header.TotalDebitAmount,
            header.Header.Status,
            header.Header.IsReversed,
            header.Header.ReversalDocumentNumber,
            header.Header.PostedBy,
            lines));
    }

    private static async Task<IResult> GetLineItemsAsync(
        [AsParameters] LineItemQuery query,
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
            SortBy = query.SortBy,
            SortDescending = query.SortDescending ?? false,
            IncludeTotalCount = query.IncludeTotalCount ?? false,
        };

        if (request.SortBy is not null && !SortableLineFields.Contains(request.SortBy))
        {
            throw new ArgumentException(
                $"Cannot sort by '{request.SortBy}'. Allowed: " +
                string.Join(", ", SortableLineFields.Order()));
        }

        var lines =
            from l in context.Query<JournalEntryLine>().AsNoTracking()
            join company in context.Query<CompanyCode>() on l.CompanyCodeId equals company.Id
            where l.TenantId == tenantId
                  && company.CompanyCodeKey == query.CompanyCode
                  && (query.GLAccount == null || l.GLAccount == query.GLAccount)
                  && (query.FromDate == null || l.PostingDate >= query.FromDate)
                  && (query.ToDate == null || l.PostingDate <= query.ToDate)
                  && (query.OpenItemsOnly != true || l.ClearingStatus == "Open")
            select l;

        var totalCount = request.IncludeTotalCount
            ? await lines.CountAsync(cancellationToken)
            : (int?)null;

        // Sorting always ends on the key: without a unique tie-break, two pages
        // can return the same row and skip another.
        var sorted = (request.SortBy?.ToUpperInvariant(), request.SortDescending) switch
        {
            ("DOCUMENTNUMBER", false) => lines.OrderBy(l => l.DocumentNumber).ThenBy(l => l.Id),
            ("DOCUMENTNUMBER", true) => lines.OrderByDescending(l => l.DocumentNumber).ThenBy(l => l.Id),
            ("GLACCOUNT", false) => lines.OrderBy(l => l.GLAccount).ThenBy(l => l.Id),
            ("GLACCOUNT", true) => lines.OrderByDescending(l => l.GLAccount).ThenBy(l => l.Id),
            ("AMOUNTINLOCALCURRENCY", false) => lines.OrderBy(l => l.AmountInLocalCurrency).ThenBy(l => l.Id),
            ("AMOUNTINLOCALCURRENCY", true) => lines.OrderByDescending(l => l.AmountInLocalCurrency).ThenBy(l => l.Id),
            ("DUEDATE", false) => lines.OrderBy(l => l.DueDate).ThenBy(l => l.Id),
            ("DUEDATE", true) => lines.OrderByDescending(l => l.DueDate).ThenBy(l => l.Id),
            (_, true) => lines.OrderByDescending(l => l.PostingDate).ThenBy(l => l.Id),
            _ => lines.OrderBy(l => l.PostingDate).ThenBy(l => l.Id),
        };

        // One row more than the page tells us whether a next page exists
        // without counting the whole result set.
        var page = await sorted
            .Skip(request.Skip)
            .Take(request.PageSize + 1)
            .Select(l => new JournalEntryLineItem(
                l.DocumentNumber,
                l.FiscalYear,
                l.LineItemNumber,
                l.PostingDate,
                l.PostingKey,
                l.DebitCreditIndicator,
                l.GLAccount,
                l.AmountInDocumentCurrency,
                l.DocumentCurrencyCode,
                l.AmountInLocalCurrency,
                l.LocalCurrencyCode,
                l.AssignmentReference,
                l.LineItemText,
                l.ClearingStatus,
                l.DueDate))
            .ToListAsync(cancellationToken);

        var hasNextPage = page.Count > request.PageSize;
        if (hasNextPage)
        {
            page.RemoveAt(page.Count - 1);
        }

        return Results.Ok(PagedResult<JournalEntryLineItem>.Create(
            page, request, totalCount, hasNextPage));
    }

    /// <summary>
    /// A rejected document is a 422 with the complete rule list, so the UI can
    /// put each message beside the field that caused it.
    /// </summary>
    private static IResult RejectionProblem(PostingResult result, HttpContext context)
    {
        var problem = new ProblemDetails
        {
            Status = StatusCodes.Status422UnprocessableEntity,
            Title = "The document was rejected",
            Type = "https://errors.erps4.local/POSTING.REJECTED",
            Detail = $"{result.Errors.Count} rule(s) refused this document.",
            Instance = context.Request.Path,
        };

        problem.Extensions["errorCode"] = "POSTING.REJECTED";
        problem.Extensions["traceId"] = context.TraceIdentifier;
        problem.Extensions["correlationId"] =
            context.Items[ProblemDetailsMiddleware.CorrelationHeader]?.ToString();
        problem.Extensions["errors"] = result.Errors
            .Select(e => new { e.Code, e.Message, e.Field, e.LineNumber })
            .ToArray();

        return Results.Problem(problem);
    }
}

/// <param name="ReasonCode">Reversal reason.</param>
/// <param name="PostingDate">Defaults to the original posting date.</param>
public sealed record ReverseDocumentRequest(
    string CompanyCode,
    string ReasonCode,
    DateOnly? PostingDate = null);

/// <summary>Query string of the line item list.</summary>
public sealed record LineItemQuery
{
    [FromQuery] public required string CompanyCode { get; init; }

    [FromQuery] public string? GLAccount { get; init; }

    [FromQuery] public DateOnly? FromDate { get; init; }

    [FromQuery] public DateOnly? ToDate { get; init; }

    [FromQuery] public bool? OpenItemsOnly { get; init; }

    [FromQuery] public int? PageNumber { get; init; }

    [FromQuery] public int? PageSize { get; init; }

    [FromQuery] public string? SortBy { get; init; }

    [FromQuery] public bool? SortDescending { get; init; }

    [FromQuery] public bool? IncludeTotalCount { get; init; }
}

public sealed record PostingResponse(
    string? DocumentNumber,
    short FiscalYear,
    byte FiscalPeriod,
    string? Status,
    bool WasSimulated,
    bool WasAlreadyPosted,
    IReadOnlyList<PostingResponseLine> Lines)
{
    public static PostingResponse From(PostingResult result) =>
        new(result.DocumentNumber,
            result.FiscalYear,
            result.FiscalPeriod,
            result.Status,
            result.WasSimulated,
            result.WasAlreadyPosted,
            result.Lines.Select(PostingResponseLine.From).ToList());
}

public sealed record PostingResponseLine(
    int LineNumber,
    string PostingKey,
    string DebitCreditIndicator,
    string Account,
    decimal DocumentAmount,
    string DocumentCurrency,
    decimal LocalAmount,
    string LocalCurrency,
    decimal? GroupAmount,
    string? GroupCurrency,
    string? BusinessPartner,
    string? CostCenter,
    string? ProfitCenter,
    string? Text)
{
    public static PostingResponseLine From(SimulatedLine line) =>
        new(line.LineNumber,
            line.PostingKey,
            line.DebitCreditIndicator,
            line.Account,
            line.DocumentAmount.Amount,
            line.DocumentAmount.Currency,
            line.LocalAmount.Amount,
            line.LocalAmount.Currency,
            line.GroupAmount?.Amount,
            line.GroupAmount?.Currency,
            line.BusinessPartner,
            line.CostCenter,
            line.ProfitCenter,
            line.Text);
}

public sealed record JournalEntryDocument(
    string DocumentNumber,
    short FiscalYear,
    string CompanyCode,
    DateOnly PostingDate,
    DateOnly DocumentDate,
    string DocumentCurrency,
    decimal TotalDebitAmount,
    string Status,
    bool IsReversed,
    string? ReversalDocumentNumber,
    string? PostedBy,
    IReadOnlyList<JournalEntryLineItem> Lines);

public sealed record JournalEntryLineItem(
    string DocumentNumber,
    short FiscalYear,
    int LineItemNumber,
    DateOnly PostingDate,
    string PostingKey,
    string DebitCreditIndicator,
    string GLAccount,
    decimal AmountInDocumentCurrency,
    string DocumentCurrencyCode,
    decimal AmountInLocalCurrency,
    string LocalCurrencyCode,
    string? AssignmentReference,
    string? LineItemText,
    string ClearingStatus,
    DateOnly? DueDate);
