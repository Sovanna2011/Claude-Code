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

    async Task<RawQueryResult> IErpDataContext.QueryRawAsync(
        RawQuery query,
        CancellationToken cancellationToken)
    {
        // EnableRetryOnFailure only covers commands EF itself issues. ADO.NET
        // run directly would sail past it, so a transient SQL Server or Azure
        // SQL error would surface as a 500 on a query that is safe to repeat.
        var strategy = Database.CreateExecutionStrategy();

        return await strategy.ExecuteAsync(async token =>
        {
            var connection = Database.GetDbConnection();

            // If the context already had the connection open - inside a
            // transaction, say - it is not this method's to close.
            var opened = connection.State != System.Data.ConnectionState.Open;
            if (opened)
            {
                await connection.OpenAsync(token);
            }

            try
            {
                return await ReadAsync(connection, query, token);
            }
            finally
            {
                if (opened)
                {
                    await connection.CloseAsync();
                }
            }
        }, cancellationToken);
    }

    private static async Task<RawQueryResult> ReadAsync(
        System.Data.Common.DbConnection connection,
        RawQuery query,
        CancellationToken cancellationToken)
    {
        await using var command = connection.CreateCommand();
        command.CommandText = query.Sql;
        command.CommandTimeout = query.CommandTimeoutSeconds;

        foreach (var parameter in query.Parameters)
        {
            var dbParameter = command.CreateParameter();
            dbParameter.ParameterName = parameter.Name;
            dbParameter.Value = parameter.Value ?? DBNull.Value;
            command.Parameters.Add(dbParameter);
        }

        await using var reader = await command.ExecuteReaderAsync(cancellationToken);

        var columns = Enumerable.Range(0, reader.FieldCount).Select(reader.GetName).ToList();
        var rows = new List<object?[]>();

        while (await reader.ReadAsync(cancellationToken))
        {
            var values = new object?[reader.FieldCount];
            reader.GetValues(values!);

            for (var index = 0; index < values.Length; index++)
            {
                if (values[index] == DBNull.Value)
                {
                    values[index] = null;
                }
            }

            rows.Add(values);
        }

        return new RawQueryResult(columns, rows);
    }

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
