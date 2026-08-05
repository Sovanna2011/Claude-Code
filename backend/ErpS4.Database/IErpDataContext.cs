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
    /// Runs one financial business transaction atomically. Under a retrying
    /// execution strategy the whole delegate is replayed, so it must not depend
    /// on state built up before the call.
    /// </summary>
    Task<TResult> ExecuteInTransactionAsync<TResult>(
        Func<CancellationToken, Task<TResult>> operation,
        CancellationToken cancellationToken = default);
}
