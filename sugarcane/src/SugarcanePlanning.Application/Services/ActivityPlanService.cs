using System.Linq.Expressions;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Options;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>
/// The scheduling engine of section 7: turns each approved projection line into a chain of
/// activity plans, honouring sequence, day offsets, dependency lag and the working calendar.
/// </summary>
public class ActivityPlanService : ServiceBase, IActivityPlanService
{
    private readonly PlanningOptions _options;
    private readonly IMaterialRequirementService _materials;

    public ActivityPlanService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock,
        IOptions<PlanningOptions> options, IMaterialRequirementService materials)
        : base(db, user, clock)
    {
        _options = options.Value;
        _materials = materials;
    }

    // ------------------------------------------------------------- generation

    public async Task<GenerateActivityPlanResultDto> GenerateAsync(GenerateActivityPlanRequest request, CancellationToken ct = default)
    {
        var projection = await RequireAsync(
            Db.Projections.Include(p => p.Lines).ThenInclude(l => l.Block!).ThenInclude(b => b.Zone),
            request.ProjectionId, "Planting projection", ct);

        if (projection.Status is not (ProjectionStatus.Approved or ProjectionStatus.Submitted or ProjectionStatus.UnderReview))
            throw new BusinessRuleException("NOT_APPROVED",
                "Activity plans are generated from a submitted or approved projection.");

        var activities = await Db.Activities.AsNoTracking()
            .Where(a => a.IsActive)
            .OrderBy(a => a.SequenceNo)
            .ToListAsync(ct);
        if (activities.Count == 0)
            throw new BusinessRuleException("NO_ACTIVITIES", "No active planting activities are configured.");

        var dependencies = await Db.ActivityDependencies.AsNoTracking().ToListAsync(ct);
        var tractorFuelPerHa = await Db.Tractors.AsNoTracking()
            .Where(t => t.IsActive && t.FuelConsumptionPerHa > 0)
            .Select(t => (decimal?)t.FuelConsumptionPerHa)
            .AverageAsync(ct) ?? 0m;

        var result = new GenerateActivityPlanResultDto { ProjectionId = projection.Id };

        await using var tx = await Db.BeginTransactionAsync(ct);

        if (request.Regenerate)
        {
            var existing = await Db.ActivityPlans.Where(p => p.ProjectionId == projection.Id).ToListAsync(ct);
            var scheduled = await Db.Schedules.AsNoTracking()
                .Where(s => existing.Select(e => e.Id).Contains(s.ActivityPlanId) && s.Status != ScheduleStatus.Cancelled)
                .CountAsync(ct);
            if (scheduled > 0)
                throw new BusinessRuleException("PLANS_SCHEDULED",
                    $"{scheduled} resource bookings exist for this projection. Cancel them before regenerating the plan.");

            Db.ActivityPlans.RemoveRange(existing);
            result.PlansRemoved = existing.Count;
            await Db.SaveChangesAsync(ct);
        }

        foreach (var line in projection.Lines.Where(l => !l.IsDeleted))
        {
            var applicable = activities.Where(a => a.AppliesTo(line.CropType)).OrderBy(a => a.SequenceNo).ToList();
            if (applicable.Count == 0)
            {
                result.Warnings.Add($"No activities are configured for crop type {line.CropType}; block {line.Block?.Code} was skipped.");
                continue;
            }

            // Earliest finish per activity, so a dependent activity can be pushed out.
            var finishByActivity = new Dictionary<int, DateOnly>();

            foreach (var activity in applicable)
            {
                var start = line.PlannedPlantingStart.AddDays(activity.StandardStartDayOffset);

                // Respect blocking predecessors: start no earlier than predecessor end + lag + 1.
                foreach (var dep in dependencies.Where(d => d.ActivityId == activity.Id && d.IsBlocking))
                {
                    if (finishByActivity.TryGetValue(dep.PredecessorActivityId, out var predEnd))
                    {
                        var earliest = predEnd.AddDays(dep.LagDays + 1);
                        if (earliest > start) start = earliest;
                    }
                }

                var duration = EstimateDurationDays(line.ProjectedPlantingAreaHa, activity);
                var end = AddWorkingDays(start, duration, request.WorkOnSaturday, request.WorkOnSunday);

                var plan = new ActivityPlan
                {
                    CompanyId = projection.CompanyId,
                    ProjectionId = projection.Id,
                    ProjectionLineId = line.Id,
                    FarmId = line.FarmId,
                    ZoneId = line.ZoneId,
                    BlockId = line.BlockId,
                    ActivityId = activity.Id,
                    SequenceNo = activity.SequenceNo,
                    PlannedAreaHa = line.ProjectedPlantingAreaHa,
                    PlannedStartDate = start,
                    PlannedEndDate = end,
                    RequiredEquipmentCategory = activity.DefaultEquipmentCategory,
                    Status = ActivityStatus.Planned
                };
                plan.Recalculate(activity, activity.RequiresTractor ? tractorFuelPerHa : null,
                    request.WorkOnSaturday, request.WorkOnSunday);

                Db.ActivityPlans.Add(plan);
                finishByActivity[activity.Id] = end;

                result.PlansCreated++;
                result.EarliestStart = result.EarliestStart is null || start < result.EarliestStart ? start : result.EarliestStart;
                result.LatestEnd = result.LatestEnd is null || end > result.LatestEnd ? end : result.LatestEnd;

                if (end > projection.PlanningEndDate)
                    result.Warnings.Add($"{activity.Name} on block {line.Block?.Code} ends {end:yyyy-MM-dd}, after the plan period ({projection.PlanningEndDate:yyyy-MM-dd}).");
            }
        }

        await Db.SaveChangesAsync(ct);
        await tx.CommitAsync(ct);

        // Material requirements follow directly from the freshly generated plans.
        await _materials.RecalculateAsync(projection.Id, ct);

        return result;
    }

    /// <summary>
    /// Working days needed for one activity on one block: area ÷ daily capacity, at least one day.
    /// When the activity has no capacity figure it is treated as a single-day task.
    /// </summary>
    internal static int EstimateDurationDays(decimal areaHa, PlantingActivity activity)
    {
        if (activity.StandardCapacityPerDay <= 0) return 1;
        return Math.Max(1, (int)Math.Ceiling(areaHa / activity.StandardCapacityPerDay));
    }

    /// <summary>Adds <paramref name="workingDays"/> working days to a start date (inclusive of the start).</summary>
    internal static DateOnly AddWorkingDays(DateOnly start, int workingDays, bool saturday, bool sunday)
    {
        var remaining = Math.Max(1, workingDays);
        var current = start;
        while (!IsWorkingDay(current, saturday, sunday)) current = current.AddDays(1);
        while (remaining > 1)
        {
            current = current.AddDays(1);
            if (IsWorkingDay(current, saturday, sunday)) remaining--;
        }
        return current;
    }

    private static bool IsWorkingDay(DateOnly d, bool saturday, bool sunday) => d.DayOfWeek switch
    {
        DayOfWeek.Saturday => saturday,
        DayOfWeek.Sunday => sunday,
        _ => true
    };

    // ------------------------------------------------------------------- reads

    public async Task<PagedResult<ActivityPlanDto>> GetPlansAsync(int? projectionId, int? blockId, int? activityId,
        QueryParameters q, CancellationToken ct = default)
    {
        var query = BasePlanQuery().AsQueryable();
        if (projectionId is not null) query = query.Where(p => p.ProjectionId == projectionId);
        if (blockId is not null) query = query.Where(p => p.BlockId == blockId);
        if (activityId is not null) query = query.Where(p => p.ActivityId == activityId);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(p => p.Activity!.Name.Contains(q.Search) || p.Block!.Code.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<ActivityPlan, object>>>
        {
            ["start"] = p => p.PlannedStartDate,
            ["sequence"] = p => p.SequenceNo,
            ["area"] = p => p.PlannedAreaHa,
            ["status"] = p => p.Status
        };
        var paged = await query.ApplySort(q, sorts, p => p.PlannedStartDate)
            .ToPagedResultAsync(q, p => p.ToDto(), ct);

        await AttachCompletionAsync(paged.Items, ct);
        return paged;
    }

    public async Task<ActivityPlanDto> GetPlanAsync(int id, CancellationToken ct = default)
    {
        var plan = await RequireAsync(BasePlanQuery(), id, "Activity plan", ct);
        var dto = plan.ToDto();
        await AttachCompletionAsync(new[] { dto }, ct);
        return dto;
    }

    private IQueryable<ActivityPlan> BasePlanQuery() => Db.ActivityPlans
        .Include(p => p.Activity)
        .Include(p => p.Projection)
        .Include(p => p.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
        .AsNoTracking();

    private async Task AttachCompletionAsync(IReadOnlyList<ActivityPlanDto> items, CancellationToken ct)
    {
        if (items.Count == 0) return;
        var ids = items.Select(i => i.Id).ToList();
        var actuals = await Db.ActivityActuals.AsNoTracking()
            .Where(a => ids.Contains(a.ActivityPlanId))
            .Select(a => new { a.ActivityPlanId, a.CompletionPercent })
            .ToListAsync(ct);
        foreach (var item in items)
            item.CompletionPercent = actuals.FirstOrDefault(a => a.ActivityPlanId == item.Id)?.CompletionPercent ?? 0m;
    }

    // ------------------------------------------------------------------ update

    public async Task<ActivityPlanDto> UpdatePlanAsync(int id, ActivityPlanUpdateDto dto, CancellationToken ct = default)
    {
        var plan = await RequireAsync(Db.ActivityPlans.Include(p => p.Activity), id, "Activity plan", ct);
        if (dto.PlannedEndDate < dto.PlannedStartDate)
            throw new BusinessRuleException("DATE_RANGE", "Planned end date must not be before the planned start date.");

        var line = await RequireAsync(Db.ProjectionLines.AsNoTracking(), plan.ProjectionLineId, "Projection line", ct);
        if (dto.PlannedAreaHa > line.ProjectedPlantingAreaHa)
            throw new BusinessRuleException("AREA_EXCEEDS_LINE",
                $"Planned activity area cannot exceed the projected planting area of the line ({line.ProjectedPlantingAreaHa:N2} ha).");

        ApplyConcurrencyToken(plan, dto.RowVersion);

        plan.PlannedStartDate = dto.PlannedStartDate;
        plan.PlannedEndDate = dto.PlannedEndDate;
        plan.PlannedAreaHa = dto.PlannedAreaHa;
        plan.SupervisorName = dto.SupervisorName;
        plan.Status = dto.Status;
        plan.Remarks = dto.Remarks;
        plan.Recalculate(plan.Activity!, null, _options.WorkOnSaturday, _options.WorkOnSunday);

        await Db.SaveChangesAsync(ct);
        return await GetPlanAsync(id, ct);
    }

    // ------------------------------------------------------------------- gantt

    public async Task<GanttViewDto> GetGanttAsync(int projectionId, int? farmId, CancellationToken ct = default)
    {
        var query = BasePlanQuery().Where(p => p.ProjectionId == projectionId);
        if (farmId is not null) query = query.Where(p => p.FarmId == farmId);
        var plans = await query.OrderBy(p => p.BlockId).ThenBy(p => p.SequenceNo).ToListAsync(ct);

        var dependencies = await Db.ActivityDependencies.AsNoTracking().ToListAsync(ct);
        var actuals = await Db.ActivityActuals.AsNoTracking()
            .Where(a => plans.Select(p => p.Id).Contains(a.ActivityPlanId))
            .Select(a => new { a.ActivityPlanId, a.CompletionPercent })
            .ToListAsync(ct);

        var bars = plans.Select(p => new GanttBarDto
        {
            ActivityPlanId = p.Id,
            BlockName = p.Block is null ? string.Empty : $"{p.Block.Code} — {p.Block.Name}",
            ActivityName = p.Activity?.Name ?? string.Empty,
            SequenceNo = p.SequenceNo,
            Start = p.PlannedStartDate,
            End = p.PlannedEndDate,
            Status = p.Status,
            PlannedAreaHa = p.PlannedAreaHa,
            CompletionPercent = actuals.FirstOrDefault(a => a.ActivityPlanId == p.Id)?.CompletionPercent ?? 0m,
            // Dependency arrows only make sense between plans on the same block.
            PredecessorPlanIds = dependencies
                .Where(d => d.ActivityId == p.ActivityId)
                .SelectMany(d => plans.Where(o => o.BlockId == p.BlockId && o.ActivityId == d.PredecessorActivityId))
                .Select(o => o.Id)
                .ToList()
        }).ToList();

        return new GanttViewDto
        {
            RangeStart = bars.Count == 0 ? Clock.Today : bars.Min(b => b.Start),
            RangeEnd = bars.Count == 0 ? Clock.Today : bars.Max(b => b.End),
            Bars = bars
        };
    }
}
