using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Auditing;
using SugarcanePlanning.Contracts.Common;

namespace SugarcanePlanning.Application.Services;

/// <summary>Read side of the audit trail (section 22). The log itself is append-only.</summary>
public class AuditQueryService : ServiceBase, IAuditQueryService
{
    public AuditQueryService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock) : base(db, user, clock) { }

    public async Task<PagedResult<AuditLogDto>> QueryAsync(AuditLogQuery query, CancellationToken ct = default)
    {
        var rows = Db.AuditLogs.AsNoTracking().AsQueryable();

        if (!string.IsNullOrWhiteSpace(query.UserName)) rows = rows.Where(a => a.UserName.Contains(query.UserName));
        if (!string.IsNullOrWhiteSpace(query.TableName)) rows = rows.Where(a => a.TableName == query.TableName);
        if (!string.IsNullOrWhiteSpace(query.RecordId)) rows = rows.Where(a => a.RecordId == query.RecordId);
        if (query.Action is not null) rows = rows.Where(a => a.Action == query.Action);
        if (query.FromDate is not null)
            rows = rows.Where(a => a.TimestampUtc >= query.FromDate.Value.ToDateTime(TimeOnly.MinValue));
        if (query.ToDate is not null)
            rows = rows.Where(a => a.TimestampUtc <= query.ToDate.Value.ToDateTime(TimeOnly.MaxValue));

        var page = query.Page < 1 ? 1 : query.Page;
        var size = query.PageSize is < 1 or > 500 ? 50 : query.PageSize;

        var total = await rows.CountAsync(ct);
        var items = await rows
            .OrderByDescending(a => a.TimestampUtc)
            .Skip((page - 1) * size)
            .Take(size)
            .ToListAsync(ct);

        return new PagedResult<AuditLogDto>(items.Select(a => a.ToDto()).ToList(), total, page, size);
    }
}
