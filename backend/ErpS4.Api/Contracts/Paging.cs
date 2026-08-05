namespace ErpS4.Api.Contracts;

/// <summary>
/// Server-side paging contract, used by every list, report and browser query.
/// </summary>
/// <remarks>
/// Page size is capped rather than trusted: an unbounded list is how a
/// reporting screen takes a production database down.
/// </remarks>
public sealed record PageRequest
{
    public const int DefaultPageSize = 50;
    public const int MaxPageSize = 500;

    private readonly int _pageNumber = 1;
    private readonly int _pageSize = DefaultPageSize;

    /// <summary>One-based page number.</summary>
    public int PageNumber
    {
        get => _pageNumber;
        init => _pageNumber = value < 1 ? 1 : value;
    }

    public int PageSize
    {
        get => _pageSize;
        init => _pageSize = value switch
        {
            < 1 => DefaultPageSize,
            > MaxPageSize => MaxPageSize,
            _ => value,
        };
    }

    /// <summary>Field to sort by; the endpoint checks it against its allow list.</summary>
    public string? SortBy { get; init; }

    public bool SortDescending { get; init; }

    /// <summary>
    /// Total count costs a second query, so it is opt-in. Screens that only
    /// need "is there a next page" should leave it off.
    /// </summary>
    public bool IncludeTotalCount { get; init; }

    public int Skip => (PageNumber - 1) * PageSize;
}

/// <summary>One page of results plus the navigation the client needs.</summary>
public sealed record PagedResult<T>
{
    public required IReadOnlyList<T> Items { get; init; }

    public required int PageNumber { get; init; }

    public required int PageSize { get; init; }

    /// <summary>Null unless the caller asked for it.</summary>
    public int? TotalCount { get; init; }

    public bool HasNextPage { get; init; }

    public bool HasPreviousPage => PageNumber > 1;

    public static PagedResult<T> Create(
        IReadOnlyList<T> items,
        PageRequest request,
        int? totalCount,
        bool hasNextPage) =>
        new()
        {
            Items = items,
            PageNumber = request.PageNumber,
            PageSize = request.PageSize,
            TotalCount = totalCount,
            HasNextPage = hasNextPage,
        };
}
