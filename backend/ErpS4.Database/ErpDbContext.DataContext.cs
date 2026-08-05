using Microsoft.EntityFrameworkCore;

namespace ErpS4.Database;

/// <summary>
/// <see cref="IErpDataContext"/> implementation. Kept in its own file so the
/// generated DbSet partial and the hand-written context stay readable.
/// </summary>
public sealed partial class ErpDbContext : IErpDataContext
{
    IQueryable<TEntity> IErpDataContext.Query<TEntity>() => Set<TEntity>();

    void IErpDataContext.Add<TEntity>(TEntity entity) => Set<TEntity>().Add(entity);

    void IErpDataContext.AddRange<TEntity>(IEnumerable<TEntity> entities) =>
        Set<TEntity>().AddRange(entities);

    Task<int> IErpDataContext.SaveChangesAsync(CancellationToken cancellationToken) =>
        SaveChangesAsync(cancellationToken);

    async Task<TResult> IErpDataContext.ExecuteInTransactionAsync<TResult>(
        Func<CancellationToken, Task<TResult>> operation,
        CancellationToken cancellationToken)
    {
        // EnableRetryOnFailure means a transaction has to be opened inside the
        // execution strategy, otherwise a retry would resume a dead transaction.
        var strategy = Database.CreateExecutionStrategy();

        return await strategy.ExecuteAsync(async token =>
        {
            await using var transaction = await Database.BeginTransactionAsync(token);
            var result = await operation(token);
            await transaction.CommitAsync(token);
            return result;
        }, cancellationToken);
    }
}
