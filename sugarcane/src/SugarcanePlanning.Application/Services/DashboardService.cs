using System.Globalization;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Options;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Dashboard;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>Aggregates every KPI, chart series and watch list on the dashboard (section 18).</summary>
public class DashboardService : ServiceBase, IDashboardService
{
    private readonly IMaterialRequirementService _materials;
    private readonly ICapacityService _capacity;
    private readonly PlanningOptions _options;

    public DashboardService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock,
        IMaterialRequirementService materials, ICapacityService capacity, IOptions<PlanningOptions> options)
        : base(db, user, clock)
    {
        _materials = materials;
        _capacity = capacity;
        _options = options.Value;
    }

    public async Task<DashboardDto> GetAsync(int? seasonId, int? estateId, CancellationToken ct = default)
    {
        var season = seasonId is not null
            ? await Db.Seasons.AsNoTracking().FirstOrDefaultAsync(s => s.Id == seasonId, ct)
            : await Db.Seasons.AsNoTracking().Where(s => s.Status == SeasonStatus.Open)
                .OrderByDescending(s => s.StartDate).FirstOrDefaultAsync(ct);

        var dto = new DashboardDto
        {
            SeasonId = season?.Id,
            SeasonName = season?.Name ?? "(no open season)",
            GeneratedOn = Clock.Today
        };
        if (season is null) return dto;

        var projections = await Db.Projections.AsNoTracking()
            .Where(p => p.GrowingSeasonId == season.Id && p.IsCurrentVersion
                        && (estateId == null || p.EstateId == estateId)
                        && p.Status != ProjectionStatus.Rejected)
            .ToListAsync(ct);
        var projectionIds = projections.Select(p => p.Id).ToList();
        if (projectionIds.Count == 0) return dto;

        var lines = await Db.ProjectionLines.AsNoTracking()
            .Include(l => l.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .Where(l => projectionIds.Contains(l.ProjectionId))
            .ToListAsync(ct);

        dto.TotalProjectedAreaHa = Math.Round(lines.Sum(l => l.ProjectedPlantingAreaHa), PlanningFormulas.QuantityScale);
        dto.NewPlantingAreaHa = Math.Round(lines.Where(l => l.CropType == CropType.NewPlanting).Sum(l => l.ProjectedPlantingAreaHa), PlanningFormulas.QuantityScale);
        dto.RatoonAreaHa = Math.Round(lines.Where(l => l.CropType == CropType.Ratoon).Sum(l => l.ProjectedPlantingAreaHa), PlanningFormulas.QuantityScale);
        dto.TotalExpectedProductionTons = Math.Round(lines.Sum(l => l.ExpectedCaneProductionTons), PlanningFormulas.QuantityScale);

        var plans = await Db.ActivityPlans.AsNoTracking()
            .Include(p => p.Activity)
            .Include(p => p.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .Where(p => projectionIds.Contains(p.ProjectionId) && p.Status != ActivityStatus.Cancelled)
            .ToListAsync(ct);

        var planIds = plans.Select(p => p.Id).ToList();
        var actuals = await Db.ActivityActuals.AsNoTracking()
            .Where(a => planIds.Contains(a.ActivityPlanId))
            .ToListAsync(ct);
        var actualByPlan = actuals.ToDictionary(a => a.ActivityPlanId);

        // Progress is measured on the planting activity itself, not on land preparation.
        var plantingPlans = plans.Where(p => p.Activity?.Category == ActivityCategory.Planting).ToList();
        if (plantingPlans.Count == 0) plantingPlans = plans;

        dto.PlannedAreaToDateHa = Math.Round(plantingPlans
            .Where(p => p.PlannedEndDate <= Clock.Today).Sum(p => p.PlannedAreaHa), PlanningFormulas.QuantityScale);
        dto.ActualPlantedAreaHa = Math.Round(plantingPlans
            .Sum(p => actualByPlan.GetValueOrDefault(p.Id)?.ActualCompletedAreaHa ?? 0m), PlanningFormulas.QuantityScale);
        dto.OverallCompletionPercent = PlanningFormulas.CompletionPercentage(
            dto.ActualPlantedAreaHa, plantingPlans.Sum(p => p.PlannedAreaHa));

        // ------------------------------------------------------------ breakdowns

        dto.AreaByFarm = BuildBreakdown(lines, plantingPlans, actualByPlan,
            l => l.Block?.Zone?.FarmId ?? 0, l => l.Block?.Zone?.Farm?.Name ?? "(no farm)",
            p => p.FarmId);
        dto.AreaByZone = BuildBreakdown(lines, plantingPlans, actualByPlan,
            l => l.ZoneId, l => l.Block?.Zone?.Name ?? "(no zone)", p => p.ZoneId);
        dto.AreaByBlock = BuildBreakdown(lines, plantingPlans, actualByPlan,
            l => l.BlockId, l => l.Block is null ? "(no block)" : $"{l.Block.Code} — {l.Block.Name}", p => p.BlockId);

        dto.MonthlyTargets = BuildPeriodTargets(plantingPlans, actualByPlan, monthly: true);
        dto.WeeklyTargets = BuildPeriodTargets(plantingPlans, actualByPlan, monthly: false);

        // -------------------------------------------------------- resource gaps

        foreach (var projectionId in projectionIds)
        {
            var analysis = await _capacity.AnalyzeAsync(projectionId, ct);
            foreach (var lineDto in analysis.Lines.Where(l => l.ResourceType is "Tractor" or "Equipment" or "Labor"))
            {
                var existing = dto.ResourceGaps.FirstOrDefault(g =>
                    g.ResourceType == lineDto.ResourceType && g.Name == lineDto.ResourceName);
                if (existing is null)
                {
                    dto.ResourceGaps.Add(new ResourceGapDto
                    {
                        ResourceType = lineDto.ResourceType,
                        Name = lineDto.ResourceName,
                        Required = lineDto.Required,
                        Available = lineDto.Available,
                        Shortage = Math.Max(lineDto.Required - lineDto.Available, 0m),
                        Status = lineDto.Status
                    });
                }
                else
                {
                    existing.Required += lineDto.Required;
                    existing.Shortage = Math.Max(existing.Required - existing.Available, 0m);
                    existing.Status = PlanningFormulas.EvaluateCapacity(existing.Required, existing.Available, _options.AtRiskThreshold);
                }
            }
        }

        // ---------------------------------------------------------- materials

        var requirements = await _materials.GetRequirementsAsync(
            new MaterialRequirementQuery { SeasonId = season.Id, EstateId = estateId, GroupBy = "material" }, ct);

        dto.MaterialSummary = requirements
            .GroupBy(r => r.Category)
            .Select(g => new MaterialSummaryDto
            {
                Category = g.Key,
                CategoryName = g.Key.ToString(),
                Unit = g.First().Unit.ToString(),
                TotalRequirement = Math.Round(g.Sum(r => r.TotalRequirement), PlanningFormulas.QuantityScale),
                NetAvailable = Math.Round(g.Sum(r => r.NetAvailableQuantity), PlanningFormulas.QuantityScale),
                Shortage = Math.Round(g.Sum(r => r.ShortageQuantity), PlanningFormulas.QuantityScale),
                Status = PlanningFormulas.EvaluateCapacity(g.Sum(r => r.TotalRequirement),
                    g.Sum(r => r.NetAvailableQuantity), _options.AtRiskThreshold)
            })
            .OrderBy(s => s.CategoryName)
            .ToList();
        dto.MaterialShortageCount = requirements.Count(r => r.ShortageQuantity > 0);

        // ------------------------------------------------- delayed & at risk

        dto.DelayedActivities = plans
            .Where(p => p.Status == ActivityStatus.Delayed
                        || (p.Status != ActivityStatus.Completed && p.PlannedEndDate < Clock.Today))
            .Select(p =>
            {
                var actual = actualByPlan.GetValueOrDefault(p.Id);
                return new DelayedActivityDto
                {
                    ActivityPlanId = p.Id,
                    BlockName = p.Block is null ? string.Empty : $"{p.Block.Code} — {p.Block.Name}",
                    FarmName = p.Block?.Zone?.Farm?.Name ?? string.Empty,
                    ActivityName = p.Activity?.Name ?? string.Empty,
                    PlannedEndDate = p.PlannedEndDate,
                    DaysLate = Math.Max(0, Clock.Today.DayNumber - p.PlannedEndDate.DayNumber),
                    PlannedAreaHa = p.PlannedAreaHa,
                    CompletionPercent = actual?.CompletionPercent ?? 0m,
                    DelayReason = actual?.DelayReason
                };
            })
            .OrderByDescending(d => d.DaysLate)
            .Take(50)
            .ToList();
        dto.DelayedActivityCount = dto.DelayedActivities.Count;

        dto.AtRiskBlocks = plans
            .GroupBy(p => p.BlockId)
            .Select(g =>
            {
                var first = g.First();
                var plannedArea = g.Where(p => p.Activity?.Category == ActivityCategory.Planting).Sum(p => p.PlannedAreaHa);
                if (plannedArea <= 0) plannedArea = g.Max(p => p.PlannedAreaHa);
                var done = g.Sum(p => actualByPlan.GetValueOrDefault(p.Id)?.ActualCompletedAreaHa ?? 0m);
                var late = g.Count(p => p.Status != ActivityStatus.Completed && p.PlannedEndDate < Clock.Today);
                var completion = PlanningFormulas.CompletionPercentage(done, g.Sum(p => p.PlannedAreaHa));

                return new AtRiskBlockDto
                {
                    BlockId = g.Key,
                    BlockCode = first.Block?.Code ?? string.Empty,
                    BlockName = first.Block?.Name ?? string.Empty,
                    FarmName = first.Block?.Zone?.Farm?.Name ?? string.Empty,
                    PlannedAreaHa = Math.Round(plannedArea, PlanningFormulas.QuantityScale),
                    CompletionPercent = completion,
                    RiskReason = late > 0
                        ? $"{late} activity(ies) past their planned end date"
                        : "on schedule",
                    Status = late switch
                    {
                        0 => CapacityStatus.Sufficient,
                        <= 2 => CapacityStatus.AtRisk,
                        _ => CapacityStatus.Shortage
                    }
                };
            })
            .Where(b => b.Status != CapacityStatus.Sufficient)
            .OrderByDescending(b => b.Status)
            .ThenBy(b => b.BlockCode)
            .ToList();
        dto.AtRiskBlockCount = dto.AtRiskBlocks.Count;

        return dto;
    }

    private static List<AreaBreakdownDto> BuildBreakdown(
        IReadOnlyList<ProjectionLine> lines,
        IReadOnlyList<ActivityPlan> plans,
        IReadOnlyDictionary<int, ActivityActual> actuals,
        Func<ProjectionLine, int> lineKey,
        Func<ProjectionLine, string> lineName,
        Func<ActivityPlan, int> planKey)
        => lines.GroupBy(lineKey)
            .Select(g =>
            {
                var groupPlans = plans.Where(p => planKey(p) == g.Key).ToList();
                var plannedArea = groupPlans.Sum(p => p.PlannedAreaHa);
                var actualArea = groupPlans.Sum(p => actuals.GetValueOrDefault(p.Id)?.ActualCompletedAreaHa ?? 0m);
                return new AreaBreakdownDto
                {
                    Id = g.Key,
                    Name = lineName(g.First()),
                    PlannedAreaHa = Math.Round(g.Sum(l => l.ProjectedPlantingAreaHa), PlanningFormulas.QuantityScale),
                    NewPlantingAreaHa = Math.Round(g.Where(l => l.CropType == CropType.NewPlanting).Sum(l => l.ProjectedPlantingAreaHa), PlanningFormulas.QuantityScale),
                    RatoonAreaHa = Math.Round(g.Where(l => l.CropType == CropType.Ratoon).Sum(l => l.ProjectedPlantingAreaHa), PlanningFormulas.QuantityScale),
                    ActualAreaHa = Math.Round(actualArea, PlanningFormulas.QuantityScale),
                    CompletionPercent = PlanningFormulas.CompletionPercentage(actualArea, plannedArea)
                };
            })
            .OrderByDescending(b => b.PlannedAreaHa)
            .ToList();

    private static List<PeriodTargetDto> BuildPeriodTargets(IReadOnlyList<ActivityPlan> plans,
        IReadOnlyDictionary<int, ActivityActual> actuals, bool monthly)
        => plans.GroupBy(p => monthly
                ? (Year: p.PlannedStartDate.Year, Period: p.PlannedStartDate.Month)
                : (Year: ISOWeek.GetYear(p.PlannedStartDate.ToDateTime(TimeOnly.MinValue)),
                   Period: ISOWeek.GetWeekOfYear(p.PlannedStartDate.ToDateTime(TimeOnly.MinValue))))
            .Select(g =>
            {
                var target = g.Sum(p => p.PlannedAreaHa);
                var actual = g.Sum(p => actuals.GetValueOrDefault(p.Id)?.ActualCompletedAreaHa ?? 0m);
                return new PeriodTargetDto
                {
                    Year = g.Key.Year,
                    Period = g.Key.Period,
                    Label = monthly
                        ? $"{g.Key.Year}-{g.Key.Period:D2}"
                        : $"{g.Key.Year}-W{g.Key.Period:D2}",
                    PeriodStart = g.Min(p => p.PlannedStartDate),
                    PeriodEnd = g.Max(p => p.PlannedEndDate),
                    TargetAreaHa = Math.Round(target, PlanningFormulas.QuantityScale),
                    ActualAreaHa = Math.Round(actual, PlanningFormulas.QuantityScale),
                    CompletionPercent = PlanningFormulas.CompletionPercentage(actual, target)
                };
            })
            .OrderBy(p => p.Year).ThenBy(p => p.Period)
            .ToList();
}
