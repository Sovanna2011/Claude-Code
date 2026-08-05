namespace SugarcanePlanning.Contracts.Common;

/// <summary>One page of results plus the paging metadata the grids need.</summary>
public class PagedResult<T>
{
    public IReadOnlyList<T> Items { get; set; } = Array.Empty<T>();
    public int TotalCount { get; set; }
    public int Page { get; set; } = 1;
    public int PageSize { get; set; } = 25;
    public int TotalPages => PageSize <= 0 ? 0 : (int)Math.Ceiling(TotalCount / (double)PageSize);
    public bool HasPrevious => Page > 1;
    public bool HasNext => Page < TotalPages;

    public PagedResult() { }

    public PagedResult(IReadOnlyList<T> items, int totalCount, int page, int pageSize)
    {
        Items = items;
        TotalCount = totalCount;
        Page = page;
        PageSize = pageSize;
    }
}

/// <summary>Standard paging / filtering / sorting query string for every list endpoint.</summary>
public class QueryParameters
{
    private const int MaxPageSize = 500;
    private int _pageSize = 25;

    public int Page { get; set; } = 1;

    public int PageSize
    {
        get => _pageSize;
        set => _pageSize = value switch { <= 0 => 25, > MaxPageSize => MaxPageSize, _ => value };
    }

    /// <summary>Free-text filter applied to the entity's code and name.</summary>
    public string? Search { get; set; }

    /// <summary>Property name to sort by; unknown names fall back to the entity default.</summary>
    public string? SortBy { get; set; }
    public bool SortDescending { get; set; }

    /// <summary>When false, soft-deleted-inactive rows are hidden.</summary>
    public bool IncludeInactive { get; set; }

    public string ToQueryString()
    {
        var parts = new List<string> { $"page={Page}", $"pageSize={PageSize}" };
        if (!string.IsNullOrWhiteSpace(Search)) parts.Add($"search={Uri.EscapeDataString(Search)}");
        if (!string.IsNullOrWhiteSpace(SortBy)) parts.Add($"sortBy={Uri.EscapeDataString(SortBy)}");
        if (SortDescending) parts.Add("sortDescending=true");
        if (IncludeInactive) parts.Add("includeInactive=true");
        return string.Join('&', parts);
    }
}

/// <summary>Problem payload returned by the global exception handler.</summary>
public class ApiErrorDto
{
    public string Code { get; set; } = "ERROR";
    public string Message { get; set; } = string.Empty;
    public string? TraceId { get; set; }
    public Dictionary<string, string[]>? ValidationErrors { get; set; }
}

/// <summary>Id/text pair for dropdowns and value helps.</summary>
public class LookupDto
{
    public int Id { get; set; }
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string Display => string.IsNullOrWhiteSpace(Code) ? Name : $"{Code} — {Name}";
    /// <summary>Parent key, so cascading dropdowns can filter client-side.</summary>
    public int? ParentId { get; set; }
    /// <summary>Free numeric payload (e.g. plantable area of a block, horsepower of a tractor).</summary>
    public decimal? Value { get; set; }
}
