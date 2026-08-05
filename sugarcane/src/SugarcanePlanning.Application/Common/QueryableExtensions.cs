using System.Linq.Expressions;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Contracts.Common;

namespace SugarcanePlanning.Application.Common;

/// <summary>Paging and dynamic-sorting helpers shared by every list endpoint.</summary>
public static class QueryableExtensions
{
    /// <summary>Runs the count and the page query and packages both into a <see cref="PagedResult{T}"/>.</summary>
    public static async Task<PagedResult<TDto>> ToPagedResultAsync<TEntity, TDto>(
        this IQueryable<TEntity> query,
        QueryParameters parameters,
        Func<TEntity, TDto> projector,
        CancellationToken ct = default)
    {
        var total = await query.CountAsync(ct);
        var page = parameters.Page < 1 ? 1 : parameters.Page;
        var items = await query
            .Skip((page - 1) * parameters.PageSize)
            .Take(parameters.PageSize)
            .ToListAsync(ct);
        return new PagedResult<TDto>(items.Select(projector).ToList(), total, page, parameters.PageSize);
    }

    /// <summary>
    /// Applies <c>sortBy</c>/<c>sortDescending</c> using a whitelist of allowed property
    /// expressions, falling back to <paramref name="defaultSort"/> for unknown keys.
    /// </summary>
    public static IQueryable<TEntity> ApplySort<TEntity>(
        this IQueryable<TEntity> query,
        QueryParameters parameters,
        IReadOnlyDictionary<string, Expression<Func<TEntity, object>>> allowed,
        Expression<Func<TEntity, object>> defaultSort)
    {
        var key = parameters.SortBy;
        var selector = key is not null && allowed.TryGetValue(key, out var expr) ? expr : defaultSort;
        return parameters.SortDescending ? query.OrderByDescending(selector) : query.OrderBy(selector);
    }
}
