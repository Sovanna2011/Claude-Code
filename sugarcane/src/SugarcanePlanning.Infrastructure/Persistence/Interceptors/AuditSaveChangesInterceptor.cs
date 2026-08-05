using System.Text.Json;
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.ChangeTracking;
using Microsoft.EntityFrameworkCore.Diagnostics;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Infrastructure.Persistence.Interceptors;

/// <summary>
/// Stamps the created/modified audit fields and writes one <see cref="AuditLog"/> row per changed
/// entity, capturing old and new values, the user, the timestamp and the device (section 22).
/// </summary>
public class AuditSaveChangesInterceptor : SaveChangesInterceptor
{
    private readonly ICurrentUser _user;
    private readonly IDateTimeProvider _clock;

    /// <summary>
    /// Audit rows for inserts, paired with the entity they describe. A store-generated key is
    /// still zero while <c>SavingChanges</c> runs, so the record id is filled in afterwards.
    /// </summary>
    private readonly List<(AuditLog Log, BaseEntity Entity)> _pendingKeys = new();

    public AuditSaveChangesInterceptor(ICurrentUser user, IDateTimeProvider clock)
    {
        _user = user;
        _clock = clock;
    }

    public override InterceptionResult<int> SavingChanges(DbContextEventData eventData, InterceptionResult<int> result)
    {
        if (eventData.Context is not null) Apply(eventData.Context);
        return base.SavingChanges(eventData, result);
    }

    public override ValueTask<InterceptionResult<int>> SavingChangesAsync(DbContextEventData eventData,
        InterceptionResult<int> result, CancellationToken cancellationToken = default)
    {
        if (eventData.Context is not null) Apply(eventData.Context);
        return base.SavingChangesAsync(eventData, result, cancellationToken);
    }

    public override int SavedChanges(SaveChangesCompletedEventData eventData, int result)
    {
        if (ResolvePendingKeys() && eventData.Context is not null) eventData.Context.SaveChanges();
        return base.SavedChanges(eventData, result);
    }

    public override async ValueTask<int> SavedChangesAsync(SaveChangesCompletedEventData eventData, int result,
        CancellationToken cancellationToken = default)
    {
        if (ResolvePendingKeys() && eventData.Context is not null)
            await eventData.Context.SaveChangesAsync(cancellationToken);
        return await base.SavedChangesAsync(eventData, result, cancellationToken);
    }

    public override void SaveChangesFailed(DbContextErrorEventData eventData)
    {
        _pendingKeys.Clear();
        base.SaveChangesFailed(eventData);
    }

    public override Task SaveChangesFailedAsync(DbContextErrorEventData eventData,
        CancellationToken cancellationToken = default)
    {
        _pendingKeys.Clear();
        return base.SaveChangesFailedAsync(eventData, cancellationToken);
    }

    /// <summary>
    /// Copies the now-known keys onto the insert audit rows. Returns true when a further save is
    /// needed; the second pass finds only <see cref="AuditLog"/> changes, which are not audited,
    /// so it cannot recurse.
    /// </summary>
    private bool ResolvePendingKeys()
    {
        if (_pendingKeys.Count == 0) return false;

        foreach (var (log, entity) in _pendingKeys) log.RecordId = entity.Id.ToString();
        _pendingKeys.Clear();
        return true;
    }

    private void Apply(DbContext context)
    {
        var now = _clock.UtcNow;
        var user = _user.IsAuthenticated ? _user.UserName : "system";
        var logs = new List<AuditLog>();

        foreach (var entry in context.ChangeTracker.Entries<BaseEntity>().ToList())
        {
            // The audit log itself is never audited — that would recurse.
            if (entry.Entity is AuditLog) continue;

            switch (entry.State)
            {
                case EntityState.Added:
                    entry.Entity.CreatedAtUtc = now;
                    entry.Entity.CreatedBy = user;
                    var insertLog = BuildLog(entry, AuditAction.Create, user, now);
                    logs.Add(insertLog);
                    _pendingKeys.Add((insertLog, entry.Entity));
                    break;

                case EntityState.Modified:
                    entry.Entity.ModifiedAtUtc = now;
                    entry.Entity.ModifiedBy = user;
                    var action = entry.Entity.IsDeleted && WasJustDeleted(entry) ? AuditAction.Delete : AuditAction.Update;
                    logs.Add(BuildLog(entry, action, user, now));
                    break;

                case EntityState.Deleted:
                    logs.Add(BuildLog(entry, AuditAction.Delete, user, now));
                    break;
            }
        }

        if (logs.Count > 0) context.Set<AuditLog>().AddRange(logs);
    }

    private static bool WasJustDeleted(EntityEntry<BaseEntity> entry)
    {
        var property = entry.Property(nameof(BaseEntity.IsDeleted));
        return property.IsModified && property.OriginalValue is false;
    }

    private AuditLog BuildLog(EntityEntry<BaseEntity> entry, AuditAction action, string user, DateTime now)
    {
        var changed = new List<string>();
        var oldValues = new Dictionary<string, object?>();
        var newValues = new Dictionary<string, object?>();

        foreach (var property in entry.Properties)
        {
            var name = property.Metadata.Name;
            if (name is nameof(BaseEntity.RowVersion)) continue;
            // The key is carried by RecordId; on an insert it is still zero here.
            if (name is nameof(BaseEntity.Id)) continue;

            switch (entry.State)
            {
                case EntityState.Added:
                    newValues[name] = property.CurrentValue;
                    break;
                case EntityState.Deleted:
                    oldValues[name] = property.OriginalValue;
                    break;
                case EntityState.Modified when property.IsModified:
                    changed.Add(name);
                    oldValues[name] = property.OriginalValue;
                    newValues[name] = property.CurrentValue;
                    break;
            }
        }

        return new AuditLog
        {
            CompanyId = entry.Entity is ICompanyScoped scoped ? scoped.CompanyId : _user.CompanyId,
            UserName = user,
            TimestampUtc = now,
            Action = action,
            TableName = entry.Metadata.GetTableName() ?? entry.Metadata.ClrType.Name,
            RecordId = entry.Entity.Id.ToString(),
            OldValues = oldValues.Count == 0 ? null : JsonSerializer.Serialize(oldValues),
            NewValues = newValues.Count == 0 ? null : JsonSerializer.Serialize(newValues),
            ChangedColumns = changed.Count == 0 ? null : string.Join(',', changed),
            IpAddress = _user.IpAddress,
            DeviceInfo = _user.DeviceInfo,
            CreatedAtUtc = now,
            CreatedBy = user
        };
    }
}
