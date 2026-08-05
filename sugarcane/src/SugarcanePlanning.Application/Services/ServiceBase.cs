using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Domain.Common;

namespace SugarcanePlanning.Application.Services;

/// <summary>
/// Shared plumbing for every application service: tenant resolution, lookup-or-throw,
/// optimistic-concurrency seeding, soft deletion and code-uniqueness checks.
/// </summary>
public abstract class ServiceBase
{
    protected readonly IAppDbContext Db;
    protected readonly ICurrentUser User;
    protected readonly IDateTimeProvider Clock;

    protected ServiceBase(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock)
    {
        Db = db;
        User = user;
        Clock = clock;
    }

    /// <summary>The caller's company. Every write is stamped with it (section 21).</summary>
    protected int CompanyId => User.CompanyId
        ?? throw new ForbiddenException("The signed-in user is not assigned to a company.");

    /// <summary>Loads an entity by key or throws <see cref="NotFoundException"/>.</summary>
    protected static async Task<TEntity> RequireAsync<TEntity>(IQueryable<TEntity> source, int id, string entityName,
        CancellationToken ct) where TEntity : BaseEntity
    {
        var entity = await source.FirstOrDefaultAsync(e => e.Id == id, ct);
        return entity ?? throw new NotFoundException(entityName, id);
    }

    /// <summary>
    /// Feeds the client's row version into the change tracker so a concurrent update
    /// raises <see cref="DbUpdateConcurrencyException"/> instead of silently winning.
    /// </summary>
    protected void ApplyConcurrencyToken(BaseEntity entity, byte[]? clientRowVersion)
    {
        if (clientRowVersion is null || clientRowVersion.Length == 0) return;
        Db.Entry(entity).Property(nameof(BaseEntity.RowVersion)).OriginalValue = clientRowVersion;
    }

    /// <summary>Marks a record deleted without removing the row (section 23).</summary>
    protected void SoftDelete(BaseEntity entity)
    {
        entity.IsDeleted = true;
        entity.DeletedAtUtc = Clock.UtcNow;
        entity.DeletedBy = User.UserName;
    }

    /// <summary>Throws when another live row in the same company already uses the code.</summary>
    protected static async Task RequireUniqueCodeAsync<TEntity>(IQueryable<TEntity> siblings, string code, int excludeId,
        string entityName, CancellationToken ct) where TEntity : BaseEntity
    {
        var exists = await siblings.AnyAsync(e => e.Id != excludeId, ct);
        if (exists) throw new BusinessRuleException("DUPLICATE_CODE", $"{entityName} code '{code}' is already in use.");
    }
}
