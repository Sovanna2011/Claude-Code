using System.Globalization;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Options;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Labor;
using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>Fuel and workforce projections with conflict-aware availability (section 14).</summary>
public class FuelLaborService : ServiceBase, IFuelLaborService
{
    private readonly PlanningOptions _options;

    public FuelLaborService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock,
        IOptions<PlanningOptions> options) : base(db, user, clock) => _options = options.Value;

    private IQueryable<ActivityPlan> FilteredPlans(FuelLaborQuery query)
    {
        var plans = Db.ActivityPlans
            .Include(p => p.Activity)
            .Include(p => p.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .AsNoTracking()
            .Where(p => p.Status != ActivityStatus.Cancelled);

        if (query.ProjectionId is not null) plans = plans.Where(p => p.ProjectionId == query.ProjectionId);
        if (query.SeasonId is not null) plans = plans.Where(p => p.Projection!.GrowingSeasonId == query.SeasonId);
        if (query.EstateId is not null) plans = plans.Where(p => p.Block!.Zone!.Farm!.EstateId == query.EstateId);
        if (query.FarmId is not null) plans = plans.Where(p => p.FarmId == query.FarmId);
        if (query.ActivityId is not null) plans = plans.Where(p => p.ActivityId == query.ActivityId);
        if (query.FromDate is not null) plans = plans.Where(p => p.PlannedEndDate >= query.FromDate);
        if (query.ToDate is not null) plans = plans.Where(p => p.PlannedStartDate <= query.ToDate);
        return plans;
    }

    // -------------------------------------------------------------------- fuel

    public async Task<IReadOnlyList<FuelProjectionDto>> GetFuelProjectionAsync(FuelLaborQuery query, CancellationToken ct = default)
    {
        var plans = await FilteredPlans(query).ToListAsync(ct);
        var groupBy = (query.GroupBy ?? "activity").ToLowerInvariant();

        // Fuel per hour comes from the tractors actually booked; otherwise the fleet average.
        var schedules = await Db.Schedules.AsNoTracking()
            .Include(s => s.Tractor)
            .Where(s => s.Status != ScheduleStatus.Cancelled)
            .Select(s => new { s.ActivityPlanId, s.TractorId, s.ExpectedWorkingHours, s.ScheduleDate, s.Tractor!.FuelConsumptionPerHour, s.Tractor.FuelConsumptionPerHa, s.Tractor.Code })
            .ToListAsync(ct);

        var fleetLitersPerHour = await Db.Tractors.AsNoTracking()
            .Where(t => t.IsActive && t.FuelConsumptionPerHour > 0)
            .Select(t => (decimal?)t.FuelConsumptionPerHour).AverageAsync(ct) ?? 0m;
        var fleetLitersPerHa = await Db.Tractors.AsNoTracking()
            .Where(t => t.IsActive && t.FuelConsumptionPerHa > 0)
            .Select(t => (decimal?)t.FuelConsumptionPerHa).AverageAsync(ct) ?? 0m;

        if (groupBy == "tractor")
        {
            return schedules
                .Where(s => s.TractorId is not null)
                .GroupBy(s => new { s.TractorId, s.Code })
                .Select(g =>
                {
                    var hours = g.Sum(s => s.ExpectedWorkingHours);
                    var area = plans.Where(p => g.Select(x => x.ActivityPlanId).Contains(p.Id)).Sum(p => p.PlannedAreaHa);
                    var byHour = PlanningFormulas.ProjectedFuelByHour(hours, g.First().FuelConsumptionPerHour);
                    var byArea = PlanningFormulas.ProjectedFuelByArea(area, g.First().FuelConsumptionPerHa);
                    return new FuelProjectionDto
                    {
                        GroupBy = "tractor",
                        GroupKey = g.Key.TractorId!.Value.ToString(),
                        GroupName = g.Key.Code,
                        EntityId = g.Key.TractorId,
                        PlannedAreaHa = area,
                        PlannedWorkingHours = hours,
                        FuelByAreaLiters = byArea,
                        FuelByHourLiters = byHour,
                        ProjectedFuelLiters = Math.Max(byArea, byHour)
                    };
                })
                .OrderBy(r => r.GroupName)
                .ToList();
        }

        IEnumerable<IGrouping<string, ActivityPlan>> groups = groupBy switch
        {
            "farm" => plans.GroupBy(p => p.Block?.Zone?.Farm?.Name ?? "(no farm)"),
            "day" => plans.GroupBy(p => p.PlannedStartDate.ToString("yyyy-MM-dd")),
            "week" => plans.GroupBy(p => IsoWeekLabel(p.PlannedStartDate)),
            "month" => plans.GroupBy(p => p.PlannedStartDate.ToString("yyyy-MM")),
            _ => plans.GroupBy(p => p.Activity?.Name ?? "(no activity)")
        };

        return groups.Select(g =>
        {
            var area = g.Sum(p => p.PlannedAreaHa);
            var hours = g.Sum(p => p.PlannedWorkingHours);
            var planIds = g.Select(p => p.Id).ToHashSet();
            var booked = schedules.Where(s => planIds.Contains(s.ActivityPlanId) && s.TractorId is not null).ToList();

            var byHour = booked.Count > 0
                ? Math.Round(booked.Sum(s => s.ExpectedWorkingHours * s.FuelConsumptionPerHour), PlanningFormulas.QuantityScale)
                : PlanningFormulas.ProjectedFuelByHour(hours, fleetLitersPerHour);
            var byArea = g.Sum(p => p.PlannedFuelLiters) > 0
                ? Math.Round(g.Sum(p => p.PlannedFuelLiters), PlanningFormulas.QuantityScale)
                : PlanningFormulas.ProjectedFuelByArea(area, fleetLitersPerHa);

            return new FuelProjectionDto
            {
                GroupBy = groupBy,
                GroupKey = g.Key,
                GroupName = g.Key,
                EntityId = groupBy == "activity" ? g.First().ActivityId : null,
                PeriodStart = g.Min(p => p.PlannedStartDate),
                PeriodEnd = g.Max(p => p.PlannedEndDate),
                PlannedAreaHa = area,
                PlannedWorkingHours = hours,
                FuelByAreaLiters = byArea,
                FuelByHourLiters = byHour,
                ProjectedFuelLiters = Math.Max(byArea, byHour)
            };
        }).OrderBy(r => r.GroupName).ToList();
    }

    private static string IsoWeekLabel(DateOnly date)
    {
        var dt = date.ToDateTime(TimeOnly.MinValue);
        var week = ISOWeek.GetWeekOfYear(dt);
        return $"{ISOWeek.GetYear(dt)}-W{week:D2}";
    }

    // ------------------------------------------------------------------- labor

    public async Task<IReadOnlyList<LaborProjectionDto>> GetLaborProjectionAsync(FuelLaborQuery query, CancellationToken ct = default)
    {
        var plans = await FilteredPlans(query).Where(p => p.Activity!.RequiresLabor).ToListAsync(ct);
        var groupBy = (query.GroupBy ?? "activity").ToLowerInvariant();

        var availableWorkers = await Db.Operators.AsNoTracking().CountAsync(o => o.IsActive, ct)
                               + await Db.WorkTeams.AsNoTracking().Where(t => t.IsActive).SumAsync(t => t.MemberCount, ct);

        IEnumerable<IGrouping<string, ActivityPlan>> groups = groupBy switch
        {
            "farm" => plans.GroupBy(p => p.Block?.Zone?.Farm?.Name ?? "(no farm)"),
            "day" => plans.GroupBy(p => p.PlannedStartDate.ToString("yyyy-MM-dd")),
            "week" => plans.GroupBy(p => IsoWeekLabel(p.PlannedStartDate)),
            "month" => plans.GroupBy(p => p.PlannedStartDate.ToString("yyyy-MM")),
            "block" => plans.GroupBy(p => p.Block is null ? "(no block)" : $"{p.Block.Code} — {p.Block.Name}"),
            _ => plans.GroupBy(p => p.Activity?.Name ?? "(no activity)")
        };

        var result = new List<LaborProjectionDto>();
        foreach (var g in groups)
        {
            var area = g.Sum(p => p.PlannedAreaHa);
            var laborDays = Math.Round(g.Sum(p => p.RequiredLaborDays), PlanningFormulas.QuantityScale);
            var start = g.Min(p => p.PlannedStartDate);
            var end = g.Max(p => p.PlannedEndDate);
            var workingDays = Math.Max(1, PlanningFormulas.WorkingDays(start, end, _options.WorkOnSaturday, _options.WorkOnSunday));
            var required = PlanningFormulas.RequiredWorkers(laborDays, workingDays);

            result.Add(new LaborProjectionDto
            {
                GroupBy = groupBy,
                GroupKey = g.Key,
                GroupName = g.Key,
                EntityId = groupBy == "activity" ? g.First().ActivityId : null,
                PeriodStart = start,
                PeriodEnd = end,
                PlannedAreaHa = area,
                StandardLaborDaysPerHa = area <= 0 ? 0m : Math.Round(laborDays / area, PlanningFormulas.QuantityScale),
                RequiredLaborDays = laborDays,
                WorkingDays = workingDays,
                RequiredWorkers = required,
                AvailableWorkers = availableWorkers,
                Status = PlanningFormulas.EvaluateCapacity(required, availableWorkers, _options.AtRiskThreshold),
                SupervisorName = g.Select(p => p.SupervisorName).FirstOrDefault(s => !string.IsNullOrWhiteSpace(s))
            });
        }

        return result.OrderBy(r => r.GroupName).ToList();
    }
}
