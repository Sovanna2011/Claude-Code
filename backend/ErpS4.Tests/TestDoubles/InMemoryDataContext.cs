using System.Collections;
using System.Linq.Expressions;
using System.Reflection;
using ErpS4.Database;
using Microsoft.EntityFrameworkCore.Query;

namespace ErpS4.Tests.TestDoubles;

/// <summary>
/// An <see cref="IErpDataContext"/> backed by lists.
/// </summary>
/// <remarks>
/// The posting engine is worth testing without a database: these tests run in
/// milliseconds and assert on rules, not on SQL. The query provider below is
/// the standard shim that lets EF's async operators run over LINQ to Objects.
/// </remarks>
public sealed class InMemoryDataContext : IErpDataContext
{
    private readonly Dictionary<Type, IList> _sets = [];
    private long _nextId;

    public int SaveChangesCallCount { get; private set; }

    public int TransactionCount { get; private set; }

    public List<TEntity> Set<TEntity>() where TEntity : class
    {
        if (!_sets.TryGetValue(typeof(TEntity), out var set))
        {
            set = new List<TEntity>();
            _sets[typeof(TEntity)] = set;
        }

        return (List<TEntity>)set;
    }

    /// <summary>Adds rows as if they were already in the database.</summary>
    public InMemoryDataContext Seed<TEntity>(params TEntity[] entities) where TEntity : class
    {
        foreach (var entity in entities)
        {
            AssignIdentity(entity);
            Set<TEntity>().Add(entity);
        }

        return this;
    }

    public IQueryable<TEntity> Query<TEntity>() where TEntity : class =>
        new TestAsyncEnumerable<TEntity>(Set<TEntity>());

    public void Add<TEntity>(TEntity entity) where TEntity : class
    {
        AssignIdentity(entity);
        Set<TEntity>().Add(entity);
    }

    public void AddRange<TEntity>(IEnumerable<TEntity> entities) where TEntity : class
    {
        foreach (var entity in entities)
        {
            Add(entity);
        }
    }

    public Task<int> SaveChangesAsync(CancellationToken cancellationToken = default)
    {
        SaveChangesCallCount++;
        return Task.FromResult(0);
    }

    /// <summary>Every raw query the code under test built, in order.</summary>
    public List<RawQuery> RawQueries { get; } = [];

    /// <summary>What <see cref="QueryRawAsync"/> hands back. Empty by default.</summary>
    public RawQueryResult RawResult { get; set; } = new([], []);

    /// <summary>
    /// Records the statement instead of executing it. The point of the table
    /// browser tests is what the SQL looks like and what travelled as a
    /// parameter, which is exactly what this captures.
    /// </summary>
    public Task<RawQueryResult> QueryRawAsync(
        RawQuery query,
        CancellationToken cancellationToken = default)
    {
        RawQueries.Add(query);
        return Task.FromResult(RawResult);
    }

    public Task<TResult> ExecuteInTransactionAsync<TResult>(
        Func<CancellationToken, Task<TResult>> operation,
        CancellationToken cancellationToken = default)
    {
        TransactionCount++;
        return operation(cancellationToken);
    }

    /// <summary>Stands in for IDENTITY, so code that reads Id after Add works.</summary>
    private void AssignIdentity<TEntity>(TEntity entity) where TEntity : class
    {
        var property = typeof(TEntity).GetProperty("Id", BindingFlags.Public | BindingFlags.Instance);
        if (property is null || !property.CanWrite)
        {
            return;
        }

        if (property.PropertyType == typeof(long) && (long)(property.GetValue(entity) ?? 0L) == 0L)
        {
            property.SetValue(entity, ++_nextId);
        }
        else if (property.PropertyType == typeof(int) && (int)(property.GetValue(entity) ?? 0) == 0)
        {
            property.SetValue(entity, (int)++_nextId);
        }
    }
}

internal sealed class TestAsyncEnumerable<T>(IEnumerable<T> enumerable)
    : EnumerableQuery<T>(enumerable), IAsyncEnumerable<T>, IQueryable<T>
{
    public TestAsyncEnumerable(Expression expression)
        : this(new EnumerableQuery<T>(expression))
    {
    }

    public IAsyncEnumerator<T> GetAsyncEnumerator(CancellationToken cancellationToken = default) =>
        new TestAsyncEnumerator<T>(this.AsEnumerable().GetEnumerator());

    IQueryProvider IQueryable.Provider => new TestAsyncQueryProvider<T>(this);
}

internal sealed class TestAsyncEnumerator<T>(IEnumerator<T> inner) : IAsyncEnumerator<T>
{
    public T Current => inner.Current;

    public ValueTask<bool> MoveNextAsync() => ValueTask.FromResult(inner.MoveNext());

    public ValueTask DisposeAsync()
    {
        inner.Dispose();
        return ValueTask.CompletedTask;
    }
}

internal sealed class TestAsyncQueryProvider<TEntity>(IQueryProvider inner) : IAsyncQueryProvider
{
    public IQueryable CreateQuery(Expression expression) =>
        new TestAsyncEnumerable<TEntity>(expression);

    public IQueryable<TElement> CreateQuery<TElement>(Expression expression) =>
        new TestAsyncEnumerable<TElement>(expression);

    public object? Execute(Expression expression) => inner.Execute(expression);

    public TResult Execute<TResult>(Expression expression) => inner.Execute<TResult>(expression);

    public TResult ExecuteAsync<TResult>(Expression expression, CancellationToken cancellationToken)
    {
        // EF calls this for FirstOrDefaultAsync and friends: TResult is Task<T>,
        // so run the query synchronously and wrap the result.
        var resultType = typeof(TResult).GetGenericArguments()[0];

        var execute = typeof(IQueryProvider)
            .GetMethods()
            .First(m => m.Name == nameof(IQueryProvider.Execute) && m.IsGenericMethod)
            .MakeGenericMethod(resultType);

        var result = execute.Invoke(inner, [expression]);

        return (TResult)typeof(Task)
            .GetMethod(nameof(Task.FromResult))!
            .MakeGenericMethod(resultType)
            .Invoke(null, [result])!;
    }
}
