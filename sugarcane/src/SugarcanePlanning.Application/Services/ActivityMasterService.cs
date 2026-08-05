using System.Linq.Expressions;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;

namespace SugarcanePlanning.Application.Services;

/// <summary>Activity master plus the dependency graph, including cycle prevention (section 6).</summary>
public class ActivityMasterService : ServiceBase, IActivityMasterService
{
    public ActivityMasterService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock) : base(db, user, clock) { }

    public async Task<PagedResult<PlantingActivityDto>> GetActivitiesAsync(QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Activities
            .Include(a => a.Predecessors).ThenInclude(d => d.PredecessorActivity)
            .AsNoTracking().AsQueryable();
        if (!q.IncludeInactive) query = query.Where(a => a.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(a => a.Code.Contains(q.Search) || a.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<PlantingActivity, object>>>
        {
            ["code"] = a => a.Code,
            ["name"] = a => a.Name,
            ["sequence"] = a => a.SequenceNo
        };
        return await query.ApplySort(q, sorts, a => a.SequenceNo).ToPagedResultAsync(q, a => a.ToDto(), ct);
    }

    public async Task<PlantingActivityDto> GetActivityAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Activities
            .Include(a => a.Predecessors).ThenInclude(d => d.PredecessorActivity)
            .AsNoTracking(), id, "Planting activity", ct)).ToDto();

    public async Task<PlantingActivityDto> CreateActivityAsync(PlantingActivityUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.Activities.AnyAsync(a => a.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Activity code '{code}' is already in use.");

        var entity = new PlantingActivity();
        dto.ApplyTo(entity);
        entity.CompanyId = User.CompanyId ?? dto.CompanyId;
        Db.Activities.Add(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task<PlantingActivityDto> UpdateActivityAsync(int id, PlantingActivityUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Activities, id, "Planting activity", ct);
        var code = dto.Code.Trim();
        if (await Db.Activities.AnyAsync(a => a.Code == code && a.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Activity code '{code}' is already in use.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task DeleteActivityAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Activities, id, "Planting activity", ct);
        if (await Db.ActivityPlans.AnyAsync(p => p.ActivityId == id, ct))
            throw new BusinessRuleException("IN_USE", "This activity is used by an activity plan and cannot be deleted.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }

    // -------------------------------------------------------------- dependencies

    public async Task<IReadOnlyList<ActivityDependencyDto>> GetDependenciesAsync(int? activityId, CancellationToken ct = default)
    {
        var query = Db.ActivityDependencies
            .Include(d => d.Activity).Include(d => d.PredecessorActivity)
            .AsNoTracking().AsQueryable();
        if (activityId is not null) query = query.Where(d => d.ActivityId == activityId);
        var rows = await query.OrderBy(d => d.Activity!.SequenceNo).ToListAsync(ct);
        return rows.Select(d => d.ToDto()).ToList();
    }

    public async Task<ActivityDependencyDto> AddDependencyAsync(ActivityDependencyUpsertDto dto, CancellationToken ct = default)
    {
        if (dto.ActivityId == dto.PredecessorActivityId)
            throw new BusinessRuleException("SELF_DEPENDENCY", "An activity cannot depend on itself.");

        var activity = await RequireAsync(Db.Activities, dto.ActivityId, "Planting activity", ct);
        _ = await RequireAsync(Db.Activities, dto.PredecessorActivityId, "Predecessor activity", ct);

        if (await Db.ActivityDependencies.AnyAsync(
                d => d.ActivityId == dto.ActivityId && d.PredecessorActivityId == dto.PredecessorActivityId, ct))
            throw new BusinessRuleException("DUPLICATE", "That dependency already exists.");

        var edges = await Db.ActivityDependencies.AsNoTracking()
            .Select(d => new { d.ActivityId, d.PredecessorActivityId }).ToListAsync(ct);
        if (CreatesCycle(edges.Select(e => (e.ActivityId, e.PredecessorActivityId)).ToList(),
                dto.ActivityId, dto.PredecessorActivityId))
            throw new BusinessRuleException("CYCLIC_DEPENDENCY",
                "That dependency would create a cycle in the activity graph.");

        var entity = new ActivityDependency
        {
            CompanyId = activity.CompanyId,
            ActivityId = dto.ActivityId,
            PredecessorActivityId = dto.PredecessorActivityId,
            LagDays = dto.LagDays,
            IsBlocking = dto.IsBlocking,
            Remarks = dto.Remarks
        };
        Db.ActivityDependencies.Add(entity);
        await Db.SaveChangesAsync(ct);

        return (await Db.ActivityDependencies
            .Include(d => d.Activity).Include(d => d.PredecessorActivity)
            .AsNoTracking().FirstAsync(d => d.Id == entity.Id, ct)).ToDto();
    }

    public async Task RemoveDependencyAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.ActivityDependencies, id, "Activity dependency", ct);
        Db.ActivityDependencies.Remove(entity);
        await Db.SaveChangesAsync(ct);
    }

    /// <summary>
    /// Walks the existing predecessor edges from <paramref name="predecessorId"/>; if the walk
    /// reaches <paramref name="activityId"/>, adding the new edge would close a loop.
    /// </summary>
    internal static bool CreatesCycle(IReadOnlyList<(int ActivityId, int PredecessorId)> edges,
        int activityId, int predecessorId)
    {
        var visited = new HashSet<int>();
        var stack = new Stack<int>();
        stack.Push(predecessorId);
        while (stack.Count > 0)
        {
            var current = stack.Pop();
            if (current == activityId) return true;
            if (!visited.Add(current)) continue;
            foreach (var edge in edges.Where(e => e.ActivityId == current))
                stack.Push(edge.PredecessorId);
        }
        return false;
    }
}
