using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Execution;
using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>Actual execution capture and the projection-versus-actual comparison (section 17).</summary>
public class ExecutionService : ServiceBase, IExecutionService
{
    private readonly IAuditService _audit;

    public ExecutionService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock, IAuditService audit)
        : base(db, user, clock) => _audit = audit;

    public async Task<ActivityActualDto> GetActualAsync(int activityPlanId, CancellationToken ct = default)
    {
        var actual = await BaseQuery().FirstOrDefaultAsync(a => a.ActivityPlanId == activityPlanId, ct);
        if (actual is not null) return actual.ToDto();

        // Nothing recorded yet: return an empty shell carrying the planned figures.
        var plan = await RequireAsync(
            Db.ActivityPlans.Include(p => p.Activity).Include(p => p.Block).AsNoTracking(),
            activityPlanId, "Activity plan", ct);

        return new ActivityActualDto
        {
            ActivityPlanId = plan.Id,
            BlockName = plan.Block is null ? string.Empty : $"{plan.Block.Code} — {plan.Block.Name}",
            ActivityName = plan.Activity?.Name ?? string.Empty,
            PlannedStartDate = plan.PlannedStartDate,
            PlannedEndDate = plan.PlannedEndDate,
            PlannedAreaHa = plan.PlannedAreaHa,
            PlannedFuelLiters = plan.PlannedFuelLiters
        };
    }

    private IQueryable<ActivityActual> BaseQuery() => Db.ActivityActuals
        .Include(a => a.ActivityPlan!).ThenInclude(p => p.Activity)
        .Include(a => a.ActivityPlan!).ThenInclude(p => p.Block)
        .Include(a => a.ActualTractor)
        .Include(a => a.ActualEquipment)
        .Include(a => a.ActualOperator)
        .Include(a => a.MaterialUsages).ThenInclude(u => u.Material)
        .AsNoTracking();

    public async Task<ActivityActualDto> RecordAsync(ActivityActualUpsertDto dto, CancellationToken ct = default)
    {
        var plan = await RequireAsync(Db.ActivityPlans.Include(p => p.Activity), dto.ActivityPlanId, "Activity plan", ct);

        if (dto.ActualCompletionDate is not null && dto.ActualStartDate is not null
            && dto.ActualCompletionDate < dto.ActualStartDate)
            throw new BusinessRuleException("DATE_RANGE", "Actual completion date must not be before the actual start date.");
        if (dto.ActualCompletedAreaHa < 0)
            throw new BusinessRuleException("AREA_INVALID", "Actual completed area cannot be negative.");

        var actual = await Db.ActivityActuals
            .Include(a => a.MaterialUsages)
            .FirstOrDefaultAsync(a => a.ActivityPlanId == dto.ActivityPlanId, ct);

        var isNew = actual is null;
        actual ??= new ActivityActual { CompanyId = plan.CompanyId, ActivityPlanId = plan.Id };
        if (!isNew) ApplyConcurrencyToken(actual, dto.RowVersion);

        actual.ActualStartDate = dto.ActualStartDate;
        actual.ActualCompletionDate = dto.ActualCompletionDate;
        actual.ActualCompletedAreaHa = dto.ActualCompletedAreaHa;
        actual.ActualTractorId = dto.ActualTractorId;
        actual.ActualEquipmentId = dto.ActualEquipmentId;
        actual.ActualOperatorId = dto.ActualOperatorId;
        actual.ActualWorkingHours = dto.ActualWorkingHours;
        actual.ActualFuelLiters = dto.ActualFuelLiters;
        actual.ActualLaborDays = dto.ActualLaborDays;
        actual.DelayReason = dto.DelayReason;
        actual.Remarks = dto.Remarks;
        actual.Recalculate(plan);

        // Material usage: compare each recorded consumption with the planned requirement.
        var planned = await Db.MaterialRequirements.AsNoTracking()
            .Include(r => r.Material)
            .Where(r => r.ActivityPlanId == plan.Id)
            .ToListAsync(ct);

        Db.ActualMaterialUsages.RemoveRange(actual.MaterialUsages);
        actual.MaterialUsages.Clear();

        foreach (var usage in dto.MaterialUsages)
        {
            var requirement = planned.FirstOrDefault(p => p.MaterialId == usage.MaterialId);
            var row = new ActualMaterialUsage
            {
                CompanyId = plan.CompanyId,
                MaterialId = usage.MaterialId,
                PlannedQuantity = requirement?.TotalRequirement ?? 0m,
                ActualQuantity = usage.ActualQuantity,
                Unit = requirement?.Unit ?? UnitOfMeasure.Kilogram,
                Remarks = usage.Remarks
            };
            row.Recalculate();
            actual.MaterialUsages.Add(row);
        }

        if (isNew) Db.ActivityActuals.Add(actual);

        // Derive the plan status from progress and timing.
        plan.Status = DeriveStatus(plan, actual, Clock.Today);

        await Db.SaveChangesAsync(ct);
        await _audit.LogAsync(AuditAction.Execute, nameof(ActivityActual), actual.Id.ToString(),
            null, $"{actual.ActualCompletedAreaHa:N2} ha ({actual.CompletionPercent:N1}%)", dto.DelayReason, ct);

        return (await BaseQuery().FirstAsync(a => a.Id == actual.Id, ct)).ToDto();
    }

    /// <summary>Completed at 100 %, delayed when past the planned end, otherwise in progress.</summary>
    internal static ActivityStatus DeriveStatus(ActivityPlan plan, ActivityActual actual, DateOnly today)
    {
        if (actual.CompletionPercent >= 100m && actual.ActualCompletionDate is not null) return ActivityStatus.Completed;
        if (actual.ActualStartDate is null) return plan.Status;
        var overdue = (actual.ActualCompletionDate ?? today) > plan.PlannedEndDate;
        return overdue ? ActivityStatus.Delayed : ActivityStatus.InProgress;
    }

    // ------------------------------------------------------ projection vs actual

    public async Task<IReadOnlyList<ProjectionVsActualDto>> CompareAsync(int projectionId, string groupBy,
        CancellationToken ct = default)
    {
        var plans = await Db.ActivityPlans
            .Include(p => p.Activity)
            .Include(p => p.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .AsNoTracking()
            .Where(p => p.ProjectionId == projectionId && p.Status != ActivityStatus.Cancelled)
            .ToListAsync(ct);

        var planIds = plans.Select(p => p.Id).ToList();
        var actuals = await Db.ActivityActuals.AsNoTracking()
            .Where(a => planIds.Contains(a.ActivityPlanId))
            .ToListAsync(ct);
        var actualByPlan = actuals.ToDictionary(a => a.ActivityPlanId);

        groupBy = string.IsNullOrWhiteSpace(groupBy) ? "block" : groupBy.ToLowerInvariant();

        IEnumerable<IGrouping<string, ActivityPlan>> groups = groupBy switch
        {
            "farm" => plans.GroupBy(p => p.Block?.Zone?.Farm?.Name ?? "(no farm)"),
            "activity" => plans.GroupBy(p => p.Activity?.Name ?? "(no activity)"),
            "month" => plans.GroupBy(p => p.PlannedStartDate.ToString("yyyy-MM")),
            _ => plans.GroupBy(p => p.Block is null ? "(no block)" : $"{p.Block.Code} — {p.Block.Name}")
        };

        return groups.Select(g =>
        {
            var groupActuals = g.Select(p => actualByPlan.GetValueOrDefault(p.Id)).Where(a => a is not null).ToList();
            var plannedArea = g.Sum(p => p.PlannedAreaHa);
            var actualArea = groupActuals.Sum(a => a!.ActualCompletedAreaHa);
            var variances = groupActuals.Where(a => a!.ScheduleVarianceDays is not null)
                .Select(a => a!.ScheduleVarianceDays!.Value).ToList();

            return new ProjectionVsActualDto
            {
                GroupBy = groupBy,
                GroupName = g.Key,
                BlockId = groupBy == "block" ? g.First().BlockId : null,
                PlannedAreaHa = Math.Round(plannedArea, PlanningFormulas.QuantityScale),
                ActualAreaHa = Math.Round(actualArea, PlanningFormulas.QuantityScale),
                AreaVariance = PlanningFormulas.AreaVariance(actualArea, plannedArea),
                PlannedFuelLiters = Math.Round(g.Sum(p => p.PlannedFuelLiters), PlanningFormulas.QuantityScale),
                ActualFuelLiters = Math.Round(groupActuals.Sum(a => a!.ActualFuelLiters), PlanningFormulas.QuantityScale),
                FuelVariance = PlanningFormulas.FuelVariance(groupActuals.Sum(a => a!.ActualFuelLiters), g.Sum(p => p.PlannedFuelLiters)),
                PlannedLaborDays = Math.Round(g.Sum(p => p.RequiredLaborDays), PlanningFormulas.QuantityScale),
                ActualLaborDays = Math.Round(groupActuals.Sum(a => a!.ActualLaborDays), PlanningFormulas.QuantityScale),
                CompletionPercent = PlanningFormulas.CompletionPercentage(actualArea, plannedArea),
                PlansTotal = g.Count(),
                PlansCompleted = g.Count(p => p.Status == ActivityStatus.Completed),
                PlansDelayed = g.Count(p => p.Status == ActivityStatus.Delayed),
                AverageScheduleVarianceDays = variances.Count == 0 ? null : (int)Math.Round(variances.Average())
            };
        }).OrderBy(r => r.GroupName).ToList();
    }
}
