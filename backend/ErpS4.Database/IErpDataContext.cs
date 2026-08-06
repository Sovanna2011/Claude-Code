namespace ErpS4.Database;

/// <summary>
/// The persistence surface the application layer uses.
/// </summary>
/// <remarks>
/// Deliberately expressed with <see cref="IQueryable{T}"/> rather than
/// <c>DbSet</c>: the posting engine can then be unit tested against in-memory
/// lists, without an EF provider and without a database. <see cref="ErpDbContext"/>
/// is the production implementation.
/// </remarks>
public interface IErpDataContext
{
    /// <summary>Queryable, change-tracked access to an entity set.</summary>
    IQueryable<TEntity> Query<TEntity>() where TEntity : class;

    void Add<TEntity>(TEntity entity) where TEntity : class;

    void AddRange<TEntity>(IEnumerable<TEntity> entities) where TEntity : class;

    Task<int> SaveChangesAsync(CancellationToken cancellationToken = default);

    /// <summary>
    /// Runs a read-only query built by the table browser.
    /// </summary>
    /// <remarks>
    /// The only place in the system that executes SQL text. Everything the
    /// caller supplies travels as a parameter; the identifiers are taken from
    /// the data dictionary, never from the request.
    /// </remarks>
    Task<RawQueryResult> QueryRawAsync(RawQuery query, CancellationToken cancellationToken = default);

    /// <summary>
    /// Runs one financial business transaction atomically. Under a retrying
    /// execution strategy the whole delegate is replayed, so it must not depend
    /// on state built up before the call.
    /// </summary>
    Task<TResult> ExecuteInTransactionAsync<TResult>(
        Func<CancellationToken, Task<TResult>> operation,
        CancellationToken cancellationToken = default);
}

/// <param name="Sql">Statement text, built from dictionary metadata only.</param>
/// <param name="Parameters">Every value the caller supplied.</param>
/// <param name="CommandTimeoutSeconds">A browser query never runs unbounded.</param>
public sealed record RawQuery(
    string Sql,
    IReadOnlyList<RawParameter> Parameters,
    int CommandTimeoutSeconds = 30);

public sealed record RawParameter(string Name, object? Value);

public sealed record RawQueryResult(
    IReadOnlyList<string> Columns,
    IReadOnlyList<object?[]> Rows);
