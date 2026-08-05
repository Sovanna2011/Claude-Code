using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Options;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Capacity;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>
/// Capacity comparison of section 15: required versus available tractors, implements,
/// workers, materials, daily hectares and completion date.
/// </summary>
public class CapacityService : ServiceBase, ICapacityService
{
    private readonly PlanningOptions _options;
    private readonly IMaterialRequirementService _materials;

    public CapacityService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock,
        IOptions<PlanningOptions> options, IMaterialRequirementService materials)
        : base(db, user, clock)
    {
        _options = options.Value;
        _materials = materials;
    }

    public Task<CapacityAnalysisDto> AnalyzeAsync(int projectionId, CancellationToken ct = default)
        => AnalyzeAsync(projectionId, ScenarioLevers.None, ct);

    /// <summary>
    /// Shared implementation. The scenario engine calls it with non-default levers to model
    /// extra machines, longer days or a reduced area without touching the stored plan.
    /// </summary>
    internal async Task<CapacityAnalysisDto> AnalyzeAsync(int projectionId, ScenarioLevers levers, CancellationToken ct)
    {
        var projection = await RequireAsync(Db.Projections.AsNoTracking(), projectionId, "Planting projection", ct);

        var plans = await Db.ActivityPlans
            .Include(p => p.Activity)
            .AsNoTracking()
            .Where(p => p.ProjectionId == projectionId && p.Status != ActivityStatus.Cancelled)
            .ToListAsync(ct);

        var areaFactor = 1m - levers.AreaReductionPercent / 100m;
        if (areaFactor < 0m) areaFactor = 0m;

        var periodStart = plans.Count > 0 ? plans.Min(p => p.PlannedStartDate) : projection.PlanningStartDate;
        var periodEnd = plans.Count > 0 ? plans.Max(p => p.PlannedEndDate) : projection.PlanningEndDate;
        periodStart = periodStart.AddDays(levers.DateShiftDays);
        periodEnd = periodEnd.AddDays(levers.DateShiftDays + levers.DurationChangeDays);

        var workingDays = Math.Max(1, PlanningFormulas.WorkingDays(periodStart, periodEnd,
            _options.WorkOnSaturday, _options.WorkOnSunday));

        // Extra hours per day scale a machine's daily hectare capacity proportionally.
        var hourFactor = _options.StandardWorkingHoursPerDay <= 0
            ? 1m
            : (_options.StandardWorkingHoursPerDay + levers.ExtraHoursPerDay) / _options.StandardWorkingHoursPerDay;

        var result = new CapacityAnalysisDto
        {
            ProjectionId = projection.Id,
            ProjectionNo = projection.ProjectionNo,
            PeriodStart = periodStart,
            PeriodEnd = periodEnd,
            WorkingDays = workingDays,
            TotalPlannedAreaHa = Math.Round(plans.Sum(p => p.PlannedAreaHa) * areaFactor, PlanningFormulas.QuantityScale)
        };

        await AddTractorLinesAsync(result, plans, workingDays, areaFactor, hourFactor, levers, ct);
        await AddEquipmentLinesAsync(result, plans, workingDays, areaFactor, hourFactor, levers, ct);
        await AddLaborLineAsync(result, plans, workingDays, areaFactor, levers, ct);
        await AddMaterialLinesAsync(result, projectionId, areaFactor, ct);
        AddThroughputLines(result, projection, plans, workingDays, areaFactor, hourFactor, levers);

        result.OverallStatus = result.Lines.Count == 0
            ? CapacityStatus.Sufficient
            : result.Lines.Max(l => l.Status);

        return result;
    }

    // ------------------------------------------------------------------ tractors

    private async Task AddTractorLinesAsync(CapacityAnalysisDto result, List<ActivityPlan> plans, int workingDays,
        decimal areaFactor, decimal hourFactor, ScenarioLevers levers, CancellationToken ct)
    {
        var tractorPlans = plans.Where(p => p.Activity?.RequiresTractor == true).ToList();
        if (tractorPlans.Count == 0) return;

        var tractors = await Db.Tractors.AsNoTracking().Where(t => t.IsActive).ToListAsync(ct);
        var usable = tractors.Count(t => t.Availability is AvailabilityStatus.Available or AvailabilityStatus.Assigned)
                     + (int)levers.AdditionalTractors;

        var avgCapacity = tractors.Where(t => t.DailyCapacityHa > 0).Select(t => t.DailyCapacityHa).DefaultIfEmpty(0m).Average();

        var totalArea = Math.Round(tractorPlans.Sum(p => p.PlannedAreaHa) * areaFactor, PlanningFormulas.QuantityScale);
        var effectiveCapacity = avgCapacity * hourFactor;

        var required = effectiveCapacity <= 0
            ? 0
            : PlanningFormulas.RequiredTractors(totalArea, effectiveCapacity, workingDays);

        result.Lines.Add(new CapacityLineDto
        {
            ResourceType = "Tractor",
            ResourceName = "Tractor fleet",
            Unit = "units",
            Required = required,
            Available = usable,
            CoveragePercent = PlanningFormulas.CoveragePercent(required, usable),
            Status = PlanningFormulas.EvaluateCapacity(required, usable, _options.AtRiskThreshold),
            Recommendation = required > usable
                ? $"Add {required - usable} tractor(s) or extend the schedule; " +
                  $"{totalArea:N0} ha at {effectiveCapacity:N1} ha/day over {workingDays} working days."
                : null
        });
    }

    // ----------------------------------------------------------------- equipment

    private async Task AddEquipmentLinesAsync(CapacityAnalysisDto result, List<ActivityPlan> plans, int workingDays,
        decimal areaFactor, decimal hourFactor, ScenarioLevers levers, CancellationToken ct)
    {
        var equipmentPlans = plans.Where(p => p.Activity?.RequiresEquipment == true && p.RequiredEquipmentCategory is not null).ToList();
        if (equipmentPlans.Count == 0) return;

        var fleet = await Db.Equipment.AsNoTracking().Where(e => e.IsActive).ToListAsync(ct);

        foreach (var group in equipmentPlans.GroupBy(p => p.RequiredEquipmentCategory!.Value))
        {
            var units = fleet.Where(e => e.Category == group.Key).ToList();
            var usable = units.Count(e => e.Availability is AvailabilityStatus.Available or AvailabilityStatus.Assigned)
                         + (int)(levers.AdditionalEquipmentByCategory.TryGetValue(group.Key, out var extra) ? extra : 0m);

            var avgCapacity = units.Where(e => e.CapacityPerDay > 0).Select(e => e.CapacityPerDay).DefaultIfEmpty(0m).Average();
            var effectiveCapacity = avgCapacity * hourFactor;
            var totalArea = Math.Round(group.Sum(p => p.PlannedAreaHa) * areaFactor, PlanningFormulas.QuantityScale);

            var required = effectiveCapacity <= 0
                ? group.Max(p => p.RequiredEquipmentCount)
                : PlanningFormulas.RequiredEquipment(totalArea, effectiveCapacity, workingDays);

            result.Lines.Add(new CapacityLineDto
            {
                ResourceType = "Equipment",
                ResourceName = group.Key.ToString(),
                Category = group.Key.ToString(),
                Unit = "units",
                Required = required,
                Available = usable,
                CoveragePercent = PlanningFormulas.CoveragePercent(required, usable),
                Status = PlanningFormulas.EvaluateCapacity(required, usable, _options.AtRiskThreshold),
                Recommendation = required > usable ? $"Hire or transfer {required - usable} more {group.Key}." : null
            });
        }
    }

    // --------------------------------------------------------------------- labor

    private async Task AddLaborLineAsync(CapacityAnalysisDto result, List<ActivityPlan> plans, int workingDays,
        decimal areaFactor, ScenarioLevers levers, CancellationToken ct)
    {
        var laborPlans = plans.Where(p => p.Activity?.RequiresLabor == true).ToList();
        if (laborPlans.Count == 0) return;

        var laborDays = Math.Round(laborPlans.Sum(p => p.RequiredLaborDays) * areaFactor, PlanningFormulas.QuantityScale);
        var required = PlanningFormulas.RequiredWorkers(laborDays, workingDays);

        var available = await Db.Operators.AsNoTracking().CountAsync(o => o.IsActive, ct)
                        + await Db.WorkTeams.AsNoTracking().Where(t => t.IsActive).SumAsync(t => t.MemberCount, ct)
                        + (int)levers.AdditionalWorkers;

        result.Lines.Add(new CapacityLineDto
        {
            ResourceType = "Labor",
            ResourceName = "Field workforce",
            Unit = "workers",
            Required = required,
            Available = available,
            CoveragePercent = PlanningFormulas.CoveragePercent(required, available),
            Status = PlanningFormulas.EvaluateCapacity(required, available, _options.AtRiskThreshold),
            Recommendation = required > available
                ? $"{laborDays:N0} labor-days over {workingDays} working days need {required} workers; {required - available} short."
                : null
        });
    }

    // ----------------------------------------------------------------- materials

    private async Task AddMaterialLinesAsync(CapacityAnalysisDto result, int projectionId, decimal areaFactor, CancellationToken ct)
    {
        var requirements = await _materials.GetRequirementsAsync(
            new MaterialRequirementQuery { ProjectionId = projectionId, GroupBy = "material" }, ct);

        foreach (var r in requirements)
        {
            var required = Math.Round(r.TotalRequirement * areaFactor, PlanningFormulas.QuantityScale);
            result.Lines.Add(new CapacityLineDto
            {
                ResourceType = "Material",
                ResourceId = r.MaterialId,
                ResourceName = $"{r.MaterialCode} — {r.MaterialName}",
                Category = r.Category.ToString(),
                Unit = r.Unit.ToString(),
                Required = required,
                Available = r.NetAvailableQuantity,
                CoveragePercent = PlanningFormulas.CoveragePercent(required, r.NetAvailableQuantity),
                Status = PlanningFormulas.EvaluateCapacity(required, r.NetAvailableQuantity, _options.AtRiskThreshold),
                Recommendation = required > r.NetAvailableQuantity
                    ? $"Order {required - r.NetAvailableQuantity:N2} {r.Unit} by {r.RequiredDeliveryDate:yyyy-MM-dd}."
                    : null
            });
        }
    }

    // -------------------------------------------------- throughput & completion

    private void AddThroughputLines(CapacityAnalysisDto result, PlantingProjection projection, List<ActivityPlan> plans,
        int workingDays, decimal areaFactor, decimal hourFactor, ScenarioLevers levers)
    {
        var plantedArea = Math.Round(plans
            .Where(p => p.Activity?.Category == ActivityCategory.Planting)
            .Sum(p => p.PlannedAreaHa) * areaFactor, PlanningFormulas.QuantityScale);
        if (plantedArea <= 0) plantedArea = result.TotalPlannedAreaHa;

        result.DailyCapacityRequiredHa = workingDays <= 0 ? 0m
            : Math.Round(plantedArea / workingDays, PlanningFormulas.QuantityScale);

        var dailyAvailable = Math.Round(plans
            .Where(p => p.Status != ActivityStatus.Completed)
            .Sum(p => p.DailyTargetHa) * hourFactor, PlanningFormulas.QuantityScale);
        result.DailyCapacityAvailableHa = dailyAvailable;

        result.Lines.Add(new CapacityLineDto
        {
            ResourceType = "DailyCapacity",
            ResourceName = "Planned hectares per day",
            Unit = "ha/day",
            Required = result.DailyCapacityRequiredHa,
            Available = dailyAvailable,
            CoveragePercent = PlanningFormulas.CoveragePercent(result.DailyCapacityRequiredHa, dailyAvailable),
            Status = PlanningFormulas.EvaluateCapacity(result.DailyCapacityRequiredHa, dailyAvailable, _options.AtRiskThreshold)
        });

        result.PlannedCompletionDate = plans.Count == 0 ? null
            : plans.Max(p => p.PlannedEndDate).AddDays(levers.DateShiftDays + levers.DurationChangeDays);
        result.RequiredCompletionDate = projection.PlanningEndDate;
        result.CompletionVarianceDays = PlanningFormulas.ScheduleVarianceDays(result.PlannedCompletionDate, result.RequiredCompletionDate);

        if (result.PlannedCompletionDate is not null)
        {
            var late = result.CompletionVarianceDays > 0;
            result.Lines.Add(new CapacityLineDto
            {
                ResourceType = "Completion",
                ResourceName = "Completion date",
                Unit = "days",
                Required = 0,
                Available = result.CompletionVarianceDays ?? 0,
                CoveragePercent = late ? 0m : 100m,
                Status = late
                    ? (result.CompletionVarianceDays > 14 ? CapacityStatus.Shortage : CapacityStatus.AtRisk)
                    : CapacityStatus.Sufficient,
                Recommendation = late
                    ? $"Planned completion {result.PlannedCompletionDate:yyyy-MM-dd} is {result.CompletionVarianceDays} day(s) after the required date."
                    : null
            });
        }
    }
}

/// <summary>
/// The simulation levers of section 15, resolved from a scenario's adjustments.
/// The default instance means "run the baseline exactly as planned".
/// </summary>
internal sealed class ScenarioLevers
{
    public static ScenarioLevers None { get; } = new();

    public decimal AdditionalTractors { get; set; }
    public decimal AdditionalWorkers { get; set; }
    public decimal ExtraHoursPerDay { get; set; }
    public decimal AreaReductionPercent { get; set; }
    public int DateShiftDays { get; set; }
    public int DurationChangeDays { get; set; }
    public Dictionary<EquipmentCategory, decimal> AdditionalEquipmentByCategory { get; } = new();
}
