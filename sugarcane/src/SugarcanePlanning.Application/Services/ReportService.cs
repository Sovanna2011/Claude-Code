using System.Globalization;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Labor;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Contracts.Reports;
using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>
/// The 22 reports of section 20. Every report is produced as the same column/row structure so
/// the grid, the print preview, the PDF writer and the Excel writer share one path.
/// </summary>
public class ReportService : ServiceBase, IReportService
{
    private readonly IMaterialRequirementService _materials;
    private readonly IFuelLaborService _fuelLabor;
    private readonly ICapacityService _capacity;
    private readonly IProjectionService _projections;
    private readonly IExecutionService _execution;

    public ReportService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock,
        IMaterialRequirementService materials, IFuelLaborService fuelLabor, ICapacityService capacity,
        IProjectionService projections, IExecutionService execution)
        : base(db, user, clock)
    {
        _materials = materials;
        _fuelLabor = fuelLabor;
        _capacity = capacity;
        _projections = projections;
        _execution = execution;
    }

    public async Task<ReportResultDto> GenerateAsync(ReportKey key, ReportRequest request, CancellationToken ct = default)
    {
        request.Report = key;
        var report = key switch
        {
            ReportKey.PlantingProjection => await PlantingProjectionAsync(request, ct),
            ReportKey.MonthlyPlan => await MonthlyPlanAsync(request, ct),
            ReportKey.FarmZoneBlockPlan => await FarmZoneBlockAsync(request, ct),
            ReportKey.NewPlantingAndRatoon => await NewPlantingRatoonAsync(request, ct),
            ReportKey.ActivitySchedule => await ActivityScheduleAsync(request, ct),
            ReportKey.ActivityCalendar => await ActivityCalendarAsync(request, ct),
            ReportKey.TractorRequirement => await MachineRequirementAsync(request, isTractor: true, ct),
            ReportKey.EquipmentRequirement => await MachineRequirementAsync(request, isTractor: false, ct),
            ReportKey.TractorUtilization => await UtilizationAsync(request, isTractor: true, ct),
            ReportKey.EquipmentUtilization => await UtilizationAsync(request, isTractor: false, ct),
            ReportKey.TractorShortage => await MachineShortageAsync(request, "Tractor", ct),
            ReportKey.EquipmentShortage => await MachineShortageAsync(request, "Equipment", ct),
            ReportKey.SeedCane => await MaterialCategoryAsync(request, MaterialCategory.SeedCane, "Seed cane requirement", ct),
            ReportKey.Fertilizer => await MaterialCategoryAsync(request, MaterialCategory.Fertilizer, "Fertilizer requirement", ct),
            ReportKey.Chemicals => await ChemicalsAsync(request, ct),
            ReportKey.MaterialShortage => await MaterialShortageAsync(request, ct),
            ReportKey.Fuel => await FuelAsync(request, ct),
            ReportKey.Labor => await LaborAsync(request, ct),
            ReportKey.CapacityAnalysis => await CapacityAsync(request, ct),
            ReportKey.DelayedActivities => await DelayedAsync(request, ct),
            ReportKey.RevisionComparison => await RevisionComparisonAsync(request, ct),
            ReportKey.ProjectionVsActual => await ProjectionVsActualAsync(request, ct),
            _ => throw new ArgumentOutOfRangeException(nameof(key), key, "Unknown report.")
        };

        report.Report = key;
        report.GeneratedAtUtc = Clock.UtcNow;
        report.GeneratedBy = User.UserName;
        ApplyFilterCaptions(report, request);
        ApplySort(report, request);
        return report;
    }

    // ------------------------------------------------------------------ helpers

    private static ReportColumnDto Col(string key, string header, string type = "text", string? format = null)
        => new() { Key = key, Header = header, DataType = type, Format = format };

    private static ReportRowDto Row(params (string Key, string? Value)[] cells)
    {
        var row = new ReportRowDto();
        foreach (var (k, v) in cells) row.Values[k] = v;
        return row;
    }

    private static string N(decimal value, int decimals = 2) => value.ToString($"N{decimals}", CultureInfo.InvariantCulture);
    private static string D(DateOnly? value) => value?.ToString("yyyy-MM-dd") ?? string.Empty;

    private static void ApplyFilterCaptions(ReportResultDto report, ReportRequest request)
    {
        void Add(string label, object? value) { if (value is not null) report.Filters[label] = value.ToString()!; }
        Add("Projection", request.ProjectionId);
        Add("Season", request.SeasonId);
        Add("Estate", request.EstateId);
        Add("Farm", request.FarmId);
        Add("Zone", request.ZoneId);
        Add("Block", request.BlockId);
        Add("Activity", request.ActivityId);
        Add("Material", request.MaterialId);
        Add("From", request.FromDate?.ToString("yyyy-MM-dd"));
        Add("To", request.ToDate?.ToString("yyyy-MM-dd"));
        Add("Group by", request.GroupBy);
    }

    /// <summary>Sorts the generic rows by any column, numerically when the column is numeric.</summary>
    private static void ApplySort(ReportResultDto report, ReportRequest request)
    {
        if (string.IsNullOrWhiteSpace(request.SortBy)) return;
        var column = report.Columns.FirstOrDefault(c => c.Key == request.SortBy);
        if (column is null) return;

        Comparison<ReportRowDto> comparison = column.IsNumeric
            ? (a, b) => ParseNumber(a.Values.GetValueOrDefault(column.Key))
                .CompareTo(ParseNumber(b.Values.GetValueOrDefault(column.Key)))
            : (a, b) => string.Compare(a.Values.GetValueOrDefault(column.Key),
                b.Values.GetValueOrDefault(column.Key), StringComparison.OrdinalIgnoreCase);

        var rows = report.Rows.ToList();
        rows.Sort(comparison);
        if (request.SortDescending) rows.Reverse();
        report.Rows = rows;
    }

    private static decimal ParseNumber(string? text)
        => decimal.TryParse(text, NumberStyles.Any, CultureInfo.InvariantCulture, out var value) ? value : 0m;

    private IQueryable<ActivityPlan> PlanQuery(ReportRequest r)
    {
        var q = Db.ActivityPlans
            .Include(p => p.Activity)
            .Include(p => p.Projection)
            .Include(p => p.ProjectionLine!).ThenInclude(l => l.CaneVariety)
            .Include(p => p.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .AsNoTracking()
            .Where(p => p.Status != ActivityStatus.Cancelled);
        if (r.ProjectionId is not null) q = q.Where(p => p.ProjectionId == r.ProjectionId);
        if (r.SeasonId is not null) q = q.Where(p => p.Projection!.GrowingSeasonId == r.SeasonId);
        if (r.EstateId is not null) q = q.Where(p => p.Block!.Zone!.Farm!.EstateId == r.EstateId);
        if (r.FarmId is not null) q = q.Where(p => p.FarmId == r.FarmId);
        if (r.ZoneId is not null) q = q.Where(p => p.ZoneId == r.ZoneId);
        if (r.BlockId is not null) q = q.Where(p => p.BlockId == r.BlockId);
        if (r.ActivityId is not null) q = q.Where(p => p.ActivityId == r.ActivityId);
        if (r.FromDate is not null) q = q.Where(p => p.PlannedEndDate >= r.FromDate);
        if (r.ToDate is not null) q = q.Where(p => p.PlannedStartDate <= r.ToDate);
        return q;
    }

    // ------------------------------------------------------------- 1 projection

    private async Task<ReportResultDto> PlantingProjectionAsync(ReportRequest r, CancellationToken ct)
    {
        var lines = await Db.ProjectionLines
            .Include(l => l.Projection!).ThenInclude(p => p.GrowingSeason)
            .Include(l => l.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .Include(l => l.CaneVariety)
            .AsNoTracking()
            .Where(l => (r.ProjectionId == null || l.ProjectionId == r.ProjectionId)
                        && (r.SeasonId == null || l.Projection!.GrowingSeasonId == r.SeasonId)
                        && (r.FarmId == null || l.FarmId == r.FarmId)
                        && (r.BlockId == null || l.BlockId == r.BlockId))
            .OrderBy(l => l.Block!.Zone!.Farm!.Code).ThenBy(l => l.Block!.Code)
            .ToListAsync(ct);

        var report = new ReportResultDto
        {
            Title = "Planting projection",
            Columns =
            {
                Col("projection", "Projection"), Col("farm", "Farm"), Col("zone", "Zone"), Col("block", "Block"),
                Col("crop", "Crop type"), Col("variety", "Variety"),
                Col("area", "Projected area (ha)", "number"), Col("start", "Planting start", "date"),
                Col("end", "Planting end", "date"), Col("harvest", "Expected harvest", "date"),
                Col("yield", "Yield (t/ha)", "number"), Col("loss", "Loss %", "percent"),
                Col("harvestable", "Harvestable (ha)", "number"), Col("production", "Expected production (t)", "number")
            }
        };

        foreach (var l in lines)
            report.Rows.Add(Row(
                ("projection", l.Projection?.ProjectionNo),
                ("farm", l.Block?.Zone?.Farm?.Name),
                ("zone", l.Block?.Zone?.Name),
                ("block", $"{l.Block?.Code} — {l.Block?.Name}"),
                ("crop", l.CropType.ToString()),
                ("variety", l.CaneVariety?.Name),
                ("area", N(l.ProjectedPlantingAreaHa)),
                ("start", D(l.PlannedPlantingStart)),
                ("end", D(l.PlannedPlantingEnd)),
                ("harvest", D(l.ExpectedHarvestDate)),
                ("yield", N(l.ExpectedYieldPerHa)),
                ("loss", N(l.ExpectedLossPercent)),
                ("harvestable", N(l.HarvestableAreaHa)),
                ("production", N(l.ExpectedCaneProductionTons))));

        report.Totals["area"] = Math.Round(lines.Sum(l => l.ProjectedPlantingAreaHa), 2);
        report.Totals["harvestable"] = Math.Round(lines.Sum(l => l.HarvestableAreaHa), 2);
        report.Totals["production"] = Math.Round(lines.Sum(l => l.ExpectedCaneProductionTons), 2);
        return report;
    }

    // ----------------------------------------------------------- 2 monthly plan

    private async Task<ReportResultDto> MonthlyPlanAsync(ReportRequest r, CancellationToken ct)
    {
        var plans = await PlanQuery(r).ToListAsync(ct);
        var report = new ReportResultDto
        {
            Title = "Monthly planting plan",
            Columns =
            {
                Col("period", "Month"), Col("activities", "Activities", "number"),
                Col("blocks", "Blocks", "number"), Col("area", "Planned area (ha)", "number"),
                Col("hours", "Planned hours", "number"), Col("fuel", "Planned fuel (L)", "number"),
                Col("workers", "Peak workers", "number")
            }
        };

        foreach (var g in plans.GroupBy(p => p.PlannedStartDate.ToString("yyyy-MM")).OrderBy(g => g.Key))
            report.Rows.Add(Row(
                ("period", g.Key),
                ("activities", g.Count().ToString()),
                ("blocks", g.Select(p => p.BlockId).Distinct().Count().ToString()),
                ("area", N(g.Sum(p => p.PlannedAreaHa))),
                ("hours", N(g.Sum(p => p.PlannedWorkingHours))),
                ("fuel", N(g.Sum(p => p.PlannedFuelLiters))),
                ("workers", g.Count() == 0 ? "0" : g.Max(p => p.RequiredWorkers).ToString())));

        report.Totals["area"] = Math.Round(plans.Sum(p => p.PlannedAreaHa), 2);
        report.Totals["fuel"] = Math.Round(plans.Sum(p => p.PlannedFuelLiters), 2);
        return report;
    }

    // -------------------------------------------------- 3 farm / zone / block

    private async Task<ReportResultDto> FarmZoneBlockAsync(ReportRequest r, CancellationToken ct)
    {
        var lines = await Db.ProjectionLines
            .Include(l => l.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .AsNoTracking()
            .Where(l => (r.ProjectionId == null || l.ProjectionId == r.ProjectionId)
                        && (r.SeasonId == null || l.Projection!.GrowingSeasonId == r.SeasonId))
            .ToListAsync(ct);

        var report = new ReportResultDto
        {
            Title = "Farm / zone / block plan",
            Columns =
            {
                Col("farm", "Farm"), Col("zone", "Zone"), Col("block", "Block"),
                Col("plantable", "Plantable (ha)", "number"), Col("planned", "Planned (ha)", "number"),
                Col("utilization", "Utilization %", "percent"), Col("production", "Expected production (t)", "number")
            }
        };

        foreach (var g in lines
            .GroupBy(l => new { Farm = l.Block?.Zone?.Farm?.Name ?? "", Zone = l.Block?.Zone?.Name ?? "", Block = l.Block?.Code ?? "" })
            .OrderBy(g => g.Key.Farm).ThenBy(g => g.Key.Zone).ThenBy(g => g.Key.Block))
        {
            var plantable = g.First().AvailableAreaHa;
            var planned = g.Sum(l => l.ProjectedPlantingAreaHa);
            report.Rows.Add(Row(
                ("farm", g.Key.Farm), ("zone", g.Key.Zone), ("block", g.Key.Block),
                ("plantable", N(plantable)), ("planned", N(planned)),
                ("utilization", N(plantable <= 0 ? 0 : planned / plantable * 100m)),
                ("production", N(g.Sum(l => l.ExpectedCaneProductionTons)))));
            report.Rows[^1].Group = g.Key.Farm;
        }

        report.Totals["planned"] = Math.Round(lines.Sum(l => l.ProjectedPlantingAreaHa), 2);
        return report;
    }

    // ------------------------------------------------- 4 new planting / ratoon

    private async Task<ReportResultDto> NewPlantingRatoonAsync(ReportRequest r, CancellationToken ct)
    {
        var lines = await Db.ProjectionLines
            .Include(l => l.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .AsNoTracking()
            .Where(l => (r.ProjectionId == null || l.ProjectionId == r.ProjectionId)
                        && (r.SeasonId == null || l.Projection!.GrowingSeasonId == r.SeasonId))
            .ToListAsync(ct);

        var report = new ReportResultDto
        {
            Title = "New planting versus ratoon",
            Columns =
            {
                Col("farm", "Farm"), Col("new", "New planting (ha)", "number"), Col("ratoon", "Ratoon (ha)", "number"),
                Col("total", "Total (ha)", "number"), Col("newShare", "New planting %", "percent"),
                Col("production", "Expected production (t)", "number")
            }
        };

        foreach (var g in lines.GroupBy(l => l.Block?.Zone?.Farm?.Name ?? "(no farm)").OrderBy(g => g.Key))
        {
            var newArea = g.Where(l => l.CropType == CropType.NewPlanting).Sum(l => l.ProjectedPlantingAreaHa);
            var ratoon = g.Where(l => l.CropType == CropType.Ratoon).Sum(l => l.ProjectedPlantingAreaHa);
            var total = newArea + ratoon;
            report.Rows.Add(Row(
                ("farm", g.Key), ("new", N(newArea)), ("ratoon", N(ratoon)), ("total", N(total)),
                ("newShare", N(total <= 0 ? 0 : newArea / total * 100m)),
                ("production", N(g.Sum(l => l.ExpectedCaneProductionTons)))));
        }

        report.Totals["new"] = Math.Round(lines.Where(l => l.CropType == CropType.NewPlanting).Sum(l => l.ProjectedPlantingAreaHa), 2);
        report.Totals["ratoon"] = Math.Round(lines.Where(l => l.CropType == CropType.Ratoon).Sum(l => l.ProjectedPlantingAreaHa), 2);
        return report;
    }

    // -------------------------------------------------- 5 / 6 activity schedule

    private async Task<ReportResultDto> ActivityScheduleAsync(ReportRequest r, CancellationToken ct)
    {
        var plans = await PlanQuery(r).OrderBy(p => p.PlannedStartDate).ThenBy(p => p.SequenceNo).ToListAsync(ct);
        var report = new ReportResultDto
        {
            Title = "Activity schedule",
            Columns =
            {
                Col("block", "Block"), Col("activity", "Activity"), Col("seq", "Seq", "number"),
                Col("start", "Planned start", "date"), Col("end", "Planned end", "date"),
                Col("days", "Working days", "number"), Col("area", "Planned area (ha)", "number"),
                Col("target", "Daily target (ha)", "number"), Col("tractors", "Tractors", "number"),
                Col("workers", "Workers", "number"), Col("status", "Status", "status")
            }
        };

        foreach (var p in plans)
            report.Rows.Add(Row(
                ("block", $"{p.Block?.Code} — {p.Block?.Name}"),
                ("activity", p.Activity?.Name),
                ("seq", p.SequenceNo.ToString()),
                ("start", D(p.PlannedStartDate)), ("end", D(p.PlannedEndDate)),
                ("days", p.WorkingDays.ToString()),
                ("area", N(p.PlannedAreaHa)), ("target", N(p.DailyTargetHa)),
                ("tractors", p.RequiredTractorCount.ToString()),
                ("workers", p.RequiredWorkers.ToString()),
                ("status", p.Status.ToString())));

        report.Totals["area"] = Math.Round(plans.Sum(p => p.PlannedAreaHa), 2);
        return report;
    }

    private async Task<ReportResultDto> ActivityCalendarAsync(ReportRequest r, CancellationToken ct)
    {
        var plans = await PlanQuery(r).ToListAsync(ct);
        var report = new ReportResultDto
        {
            Title = "Activity calendar",
            Columns =
            {
                Col("date", "Date", "date"), Col("activities", "Activities", "number"),
                Col("blocks", "Blocks", "number"), Col("area", "Area in progress (ha)", "number"),
                Col("detail", "Activities on this day")
            }
        };

        if (plans.Count == 0) return report;

        var from = r.FromDate ?? plans.Min(p => p.PlannedStartDate);
        var to = r.ToDate ?? plans.Max(p => p.PlannedEndDate);

        for (var d = from; d <= to; d = d.AddDays(1))
        {
            var active = plans.Where(p => p.PlannedStartDate <= d && d <= p.PlannedEndDate).ToList();
            if (active.Count == 0) continue;
            report.Rows.Add(Row(
                ("date", D(d)),
                ("activities", active.Count.ToString()),
                ("blocks", active.Select(p => p.BlockId).Distinct().Count().ToString()),
                ("area", N(active.Sum(p => p.DailyTargetHa))),
                ("detail", string.Join(", ", active.Select(p => p.Activity?.Name).Distinct().Take(6)))));
        }
        return report;
    }

    // --------------------------------------------- 7-12 machinery requirement

    private async Task<ReportResultDto> MachineRequirementAsync(ReportRequest r, bool isTractor, CancellationToken ct)
    {
        var plans = await PlanQuery(r)
            .Where(p => isTractor ? p.Activity!.RequiresTractor : p.Activity!.RequiresEquipment)
            .ToListAsync(ct);

        var report = new ReportResultDto
        {
            Title = isTractor ? "Tractor requirement" : "Equipment requirement",
            Columns =
            {
                Col("activity", "Activity"), Col("category", isTractor ? "Tractor type" : "Equipment category"),
                Col("area", "Planned area (ha)", "number"), Col("days", "Working days", "number"),
                Col("capacity", "Capacity/unit/day (ha)", "number"), Col("exact", "Exact requirement", "number"),
                Col("required", "Units required", "number")
            }
        };

        foreach (var g in plans.GroupBy(p => p.Activity!.Name).OrderBy(g => g.Key))
        {
            var area = g.Sum(p => p.PlannedAreaHa);
            var days = Math.Max(1, PlanningFormulas.WorkingDays(g.Min(p => p.PlannedStartDate), g.Max(p => p.PlannedEndDate)));
            var capacity = g.First().Activity!.StandardCapacityPerDay;
            var required = capacity <= 0 ? 0 : PlanningFormulas.RequiredTractors(area, capacity, days);

            report.Rows.Add(Row(
                ("activity", g.Key),
                ("category", isTractor
                    ? (g.First().RequiredTractorType ?? "any")
                    : (g.First().RequiredEquipmentCategory?.ToString() ?? "any")),
                ("area", N(area)), ("days", days.ToString()), ("capacity", N(capacity)),
                ("exact", N(PlanningFormulas.ExactMachineRequirement(area, capacity, days))),
                ("required", required.ToString())));
        }
        return report;
    }

    private async Task<ReportResultDto> UtilizationAsync(ReportRequest r, bool isTractor, CancellationToken ct)
    {
        var q = Db.Schedules
            .Include(s => s.Tractor).Include(s => s.Equipment)
            .AsNoTracking()
            .Where(s => s.Status != ScheduleStatus.Cancelled);
        if (r.FromDate is not null) q = q.Where(s => s.ScheduleDate >= r.FromDate);
        if (r.ToDate is not null) q = q.Where(s => s.ScheduleDate <= r.ToDate);
        q = isTractor ? q.Where(s => s.TractorId != null) : q.Where(s => s.EquipmentId != null);

        var bookings = await q.ToListAsync(ct);
        var windowStart = r.FromDate ?? (bookings.Count > 0 ? bookings.Min(b => b.ScheduleDate) : Clock.Today);
        var windowEnd = r.ToDate ?? (bookings.Count > 0 ? bookings.Max(b => b.ScheduleDate) : Clock.Today);
        var windowDays = Math.Max(1, PlanningFormulas.WorkingDays(windowStart, windowEnd));

        var report = new ReportResultDto
        {
            Title = isTractor ? "Tractor utilization" : "Equipment utilization",
            Subtitle = $"{windowStart:yyyy-MM-dd} … {windowEnd:yyyy-MM-dd} ({windowDays} working days)",
            Columns =
            {
                Col("code", "Code"), Col("name", "Description"), Col("bookings", "Bookings", "number"),
                Col("days", "Days used", "number"), Col("hours", "Hours", "number"),
                Col("area", "Area (ha)", "number"), Col("utilization", "Utilization %", "percent")
            }
        };

        foreach (var g in bookings
            .GroupBy(b => isTractor
                ? (Id: b.TractorId!.Value, Code: b.Tractor?.Code ?? "", Name: $"{b.Tractor?.Brand} {b.Tractor?.Model}".Trim())
                : (Id: b.EquipmentId!.Value, Code: b.Equipment?.Code ?? "", Name: b.Equipment?.Name ?? ""))
            .OrderBy(g => g.Key.Code))
        {
            var days = g.Select(b => b.ScheduleDate).Distinct().Count();
            report.Rows.Add(Row(
                ("code", g.Key.Code), ("name", g.Key.Name),
                ("bookings", g.Count().ToString()), ("days", days.ToString()),
                ("hours", N(g.Sum(b => b.ExpectedWorkingHours))),
                ("area", N(g.Sum(b => b.PlannedAreaHa))),
                ("utilization", N(days / (decimal)windowDays * 100m))));
        }
        return report;
    }

    private async Task<ReportResultDto> MachineShortageAsync(ReportRequest r, string resourceType, CancellationToken ct)
    {
        if (r.ProjectionId is null)
            throw new Domain.Common.BusinessRuleException("PROJECTION_REQUIRED",
                "A projection must be selected for the shortage report.");

        var analysis = await _capacity.AnalyzeAsync(r.ProjectionId.Value, ct);
        var report = new ReportResultDto
        {
            Title = $"{resourceType} shortage",
            Subtitle = $"Projection {analysis.ProjectionNo} — {analysis.WorkingDays} working days",
            Columns =
            {
                Col("resource", resourceType), Col("required", "Required", "number"),
                Col("available", "Available", "number"), Col("gap", "Gap", "number"),
                Col("coverage", "Coverage %", "percent"), Col("status", "Status", "status"),
                Col("action", "Recommended action")
            }
        };

        foreach (var line in analysis.Lines.Where(l => l.ResourceType == resourceType))
            report.Rows.Add(Row(
                ("resource", line.ResourceName), ("required", N(line.Required)), ("available", N(line.Available)),
                ("gap", N(line.Gap)), ("coverage", N(line.CoveragePercent)),
                ("status", line.Status.ToString()), ("action", line.Recommendation)));
        return report;
    }

    // -------------------------------------------------------- 13-16 materials

    private async Task<ReportResultDto> MaterialCategoryAsync(ReportRequest r, MaterialCategory category, string title,
        CancellationToken ct)
    {
        var rows = await _materials.GetRequirementsAsync(new MaterialRequirementQuery
        {
            ProjectionId = r.ProjectionId,
            SeasonId = r.SeasonId,
            EstateId = r.EstateId,
            FarmId = r.FarmId,
            BlockId = r.BlockId,
            ActivityId = r.ActivityId,
            MaterialId = r.MaterialId,
            Category = category,
            GroupBy = r.GroupBy ?? "material"
        }, ct);

        return BuildMaterialReport(title, rows);
    }

    private async Task<ReportResultDto> ChemicalsAsync(ReportRequest r, CancellationToken ct)
    {
        var all = new List<MaterialRequirementDto>();
        foreach (var category in new[] { MaterialCategory.Herbicide, MaterialCategory.Pesticide })
            all.AddRange(await _materials.GetRequirementsAsync(new MaterialRequirementQuery
            {
                ProjectionId = r.ProjectionId,
                SeasonId = r.SeasonId,
                EstateId = r.EstateId,
                FarmId = r.FarmId,
                Category = category,
                GroupBy = r.GroupBy ?? "material"
            }, ct));
        return BuildMaterialReport("Chemical requirement", all);
    }

    private async Task<ReportResultDto> MaterialShortageAsync(ReportRequest r, CancellationToken ct)
    {
        var rows = await _materials.GetRequirementsAsync(new MaterialRequirementQuery
        {
            ProjectionId = r.ProjectionId,
            SeasonId = r.SeasonId,
            EstateId = r.EstateId,
            GroupBy = "material",
            ShortagesOnly = true
        }, ct);
        return BuildMaterialReport("Material shortage", rows);
    }

    private static ReportResultDto BuildMaterialReport(string title, IReadOnlyList<MaterialRequirementDto> rows)
    {
        var report = new ReportResultDto
        {
            Title = title,
            Columns =
            {
                Col("material", "Material"), Col("category", "Category"), Col("unit", "Unit"),
                Col("area", "Planned area (ha)", "number"), Col("base", "Base requirement", "number"),
                Col("waste", "Waste", "number"), Col("total", "Total requirement", "number"),
                Col("stock", "Available stock", "number"), Col("reserved", "Reserved", "number"),
                Col("incoming", "Incoming", "number"), Col("net", "Net available", "number"),
                Col("shortage", "Shortage", "number"), Col("surplus", "Surplus", "number"),
                Col("delivery", "Required delivery", "date"), Col("status", "Status", "status")
            }
        };

        foreach (var m in rows)
            report.Rows.Add(Row(
                ("material", $"{m.MaterialCode} — {m.MaterialName}"),
                ("category", m.Category.ToString()), ("unit", m.Unit.ToString()),
                ("area", N(m.PlannedAreaHa)), ("base", N(m.BaseRequirement)), ("waste", N(m.WasteQuantity)),
                ("total", N(m.TotalRequirement)), ("stock", N(m.AvailableStock)), ("reserved", N(m.ReservedQuantity)),
                ("incoming", N(m.IncomingQuantity)), ("net", N(m.NetAvailableQuantity)),
                ("shortage", N(m.ShortageQuantity)), ("surplus", N(m.SurplusQuantity)),
                ("delivery", D(m.RequiredDeliveryDate)), ("status", m.Status.ToString())));

        report.Totals["total"] = Math.Round(rows.Sum(m => m.TotalRequirement), 2);
        report.Totals["shortage"] = Math.Round(rows.Sum(m => m.ShortageQuantity), 2);
        return report;
    }

    // ---------------------------------------------------------- 17-18 fuel/labor

    private async Task<ReportResultDto> FuelAsync(ReportRequest r, CancellationToken ct)
    {
        var rows = await _fuelLabor.GetFuelProjectionAsync(new FuelLaborQuery
        {
            ProjectionId = r.ProjectionId,
            SeasonId = r.SeasonId,
            EstateId = r.EstateId,
            FarmId = r.FarmId,
            ActivityId = r.ActivityId,
            FromDate = r.FromDate,
            ToDate = r.ToDate,
            GroupBy = r.GroupBy ?? "activity"
        }, ct);

        var report = new ReportResultDto
        {
            Title = "Fuel projection",
            Subtitle = $"Grouped by {r.GroupBy ?? "activity"}",
            Columns =
            {
                Col("group", "Group"), Col("area", "Planned area (ha)", "number"),
                Col("hours", "Planned hours", "number"), Col("byArea", "Fuel by area (L)", "number"),
                Col("byHour", "Fuel by hour (L)", "number"), Col("projected", "Projected fuel (L)", "number")
            }
        };

        foreach (var f in rows)
            report.Rows.Add(Row(
                ("group", f.GroupName), ("area", N(f.PlannedAreaHa)), ("hours", N(f.PlannedWorkingHours)),
                ("byArea", N(f.FuelByAreaLiters)), ("byHour", N(f.FuelByHourLiters)),
                ("projected", N(f.ProjectedFuelLiters))));

        report.Totals["projected"] = Math.Round(rows.Sum(f => f.ProjectedFuelLiters), 2);
        return report;
    }

    private async Task<ReportResultDto> LaborAsync(ReportRequest r, CancellationToken ct)
    {
        var rows = await _fuelLabor.GetLaborProjectionAsync(new FuelLaborQuery
        {
            ProjectionId = r.ProjectionId,
            SeasonId = r.SeasonId,
            EstateId = r.EstateId,
            FarmId = r.FarmId,
            ActivityId = r.ActivityId,
            FromDate = r.FromDate,
            ToDate = r.ToDate,
            GroupBy = r.GroupBy ?? "activity"
        }, ct);

        var report = new ReportResultDto
        {
            Title = "Labor projection",
            Subtitle = $"Grouped by {r.GroupBy ?? "activity"}",
            Columns =
            {
                Col("group", "Group"), Col("area", "Planned area (ha)", "number"),
                Col("perHa", "Labor-days per ha", "number"), Col("laborDays", "Required labor-days", "number"),
                Col("days", "Working days", "number"), Col("required", "Required workers", "number"),
                Col("available", "Available workers", "number"), Col("gap", "Gap", "number"),
                Col("status", "Status", "status")
            }
        };

        foreach (var l in rows)
            report.Rows.Add(Row(
                ("group", l.GroupName), ("area", N(l.PlannedAreaHa)), ("perHa", N(l.StandardLaborDaysPerHa, 4)),
                ("laborDays", N(l.RequiredLaborDays)), ("days", l.WorkingDays.ToString()),
                ("required", l.RequiredWorkers.ToString()), ("available", l.AvailableWorkers.ToString()),
                ("gap", l.WorkerGap.ToString()), ("status", l.Status.ToString())));

        report.Totals["laborDays"] = Math.Round(rows.Sum(l => l.RequiredLaborDays), 2);
        return report;
    }

    // ------------------------------------------------------------- 19 capacity

    private async Task<ReportResultDto> CapacityAsync(ReportRequest r, CancellationToken ct)
    {
        if (r.ProjectionId is null)
            throw new Domain.Common.BusinessRuleException("PROJECTION_REQUIRED",
                "A projection must be selected for the capacity report.");

        var analysis = await _capacity.AnalyzeAsync(r.ProjectionId.Value, ct);
        var report = new ReportResultDto
        {
            Title = "Capacity analysis",
            Subtitle = $"Projection {analysis.ProjectionNo} — {analysis.PeriodStart:yyyy-MM-dd} … {analysis.PeriodEnd:yyyy-MM-dd} " +
                       $"({analysis.WorkingDays} working days), overall {analysis.OverallStatus}",
            Columns =
            {
                Col("type", "Resource type"), Col("resource", "Resource"), Col("unit", "Unit"),
                Col("required", "Required", "number"), Col("available", "Available", "number"),
                Col("gap", "Gap", "number"), Col("coverage", "Coverage %", "percent"),
                Col("status", "Status", "status"), Col("action", "Recommendation")
            }
        };

        foreach (var line in analysis.Lines)
        {
            report.Rows.Add(Row(
                ("type", line.ResourceType), ("resource", line.ResourceName), ("unit", line.Unit),
                ("required", N(line.Required)), ("available", N(line.Available)), ("gap", N(line.Gap)),
                ("coverage", N(line.CoveragePercent)), ("status", line.Status.ToString()),
                ("action", line.Recommendation)));
            report.Rows[^1].Group = line.ResourceType;
        }
        return report;
    }

    // -------------------------------------------------------------- 20 delayed

    private async Task<ReportResultDto> DelayedAsync(ReportRequest r, CancellationToken ct)
    {
        var plans = await PlanQuery(r).ToListAsync(ct);
        var planIds = plans.Select(p => p.Id).ToList();
        var actuals = await Db.ActivityActuals.AsNoTracking()
            .Where(a => planIds.Contains(a.ActivityPlanId)).ToListAsync(ct);
        var byPlan = actuals.ToDictionary(a => a.ActivityPlanId);

        var report = new ReportResultDto
        {
            Title = "Delayed activities",
            Columns =
            {
                Col("farm", "Farm"), Col("block", "Block"), Col("activity", "Activity"),
                Col("plannedEnd", "Planned end", "date"), Col("actualEnd", "Actual end", "date"),
                Col("daysLate", "Days late", "number"), Col("area", "Planned area (ha)", "number"),
                Col("completion", "Completion %", "percent"), Col("reason", "Delay reason")
            }
        };

        foreach (var p in plans
            .Where(p => p.Status == ActivityStatus.Delayed
                        || (p.Status != ActivityStatus.Completed && p.PlannedEndDate < Clock.Today))
            .OrderByDescending(p => Clock.Today.DayNumber - p.PlannedEndDate.DayNumber))
        {
            var actual = byPlan.GetValueOrDefault(p.Id);
            var reference = actual?.ActualCompletionDate ?? Clock.Today;
            report.Rows.Add(Row(
                ("farm", p.Block?.Zone?.Farm?.Name), ("block", $"{p.Block?.Code} — {p.Block?.Name}"),
                ("activity", p.Activity?.Name), ("plannedEnd", D(p.PlannedEndDate)),
                ("actualEnd", D(actual?.ActualCompletionDate)),
                ("daysLate", Math.Max(0, reference.DayNumber - p.PlannedEndDate.DayNumber).ToString()),
                ("area", N(p.PlannedAreaHa)), ("completion", N(actual?.CompletionPercent ?? 0m)),
                ("reason", actual?.DelayReason)));
        }
        return report;
    }

    // ----------------------------------------------------------- 21 comparison

    private async Task<ReportResultDto> RevisionComparisonAsync(ReportRequest r, CancellationToken ct)
    {
        if (r.ProjectionId is null || r.CompareToProjectionId is null)
            throw new Domain.Common.BusinessRuleException("VERSIONS_REQUIRED",
                "Two projection versions must be selected for the revision comparison.");

        var comparison = await _projections.CompareVersionsAsync(r.ProjectionId.Value, r.CompareToProjectionId.Value, ct);
        var report = new ReportResultDto
        {
            Title = "Revision comparison",
            Subtitle = $"Version {comparison.FromVersion} → {comparison.ToVersion}: " +
                       $"{comparison.FromTotalAreaHa:N2} ha → {comparison.ToTotalAreaHa:N2} ha",
            Columns =
            {
                Col("scope", "Scope"), Col("block", "Block"), Col("field", "Field"),
                Col("old", "Old value"), Col("new", "New value"), Col("change", "Change type")
            }
        };

        foreach (var d in comparison.Differences)
            report.Rows.Add(Row(
                ("scope", d.Scope), ("block", d.BlockCode), ("field", d.Field),
                ("old", d.OldValue), ("new", d.NewValue), ("change", d.ChangeType)));
        return report;
    }

    // -------------------------------------------------- 22 projection vs actual

    private async Task<ReportResultDto> ProjectionVsActualAsync(ReportRequest r, CancellationToken ct)
    {
        if (r.ProjectionId is null)
            throw new Domain.Common.BusinessRuleException("PROJECTION_REQUIRED",
                "A projection must be selected for the projection-versus-actual report.");

        var rows = await _execution.CompareAsync(r.ProjectionId.Value, r.GroupBy ?? "block", ct);
        var report = new ReportResultDto
        {
            Title = "Projection versus actual",
            Subtitle = $"Grouped by {r.GroupBy ?? "block"}",
            Columns =
            {
                Col("group", "Group"), Col("plannedArea", "Planned area (ha)", "number"),
                Col("actualArea", "Actual area (ha)", "number"), Col("areaVar", "Area variance (ha)", "number"),
                Col("plannedFuel", "Planned fuel (L)", "number"), Col("actualFuel", "Actual fuel (L)", "number"),
                Col("fuelVar", "Fuel variance (L)", "number"), Col("plannedLabor", "Planned labor-days", "number"),
                Col("actualLabor", "Actual labor-days", "number"), Col("completion", "Completion %", "percent"),
                Col("delayed", "Delayed activities", "number"), Col("scheduleVar", "Avg schedule variance (days)", "number")
            }
        };

        foreach (var row in rows)
            report.Rows.Add(Row(
                ("group", row.GroupName), ("plannedArea", N(row.PlannedAreaHa)), ("actualArea", N(row.ActualAreaHa)),
                ("areaVar", N(row.AreaVariance)), ("plannedFuel", N(row.PlannedFuelLiters)),
                ("actualFuel", N(row.ActualFuelLiters)), ("fuelVar", N(row.FuelVariance)),
                ("plannedLabor", N(row.PlannedLaborDays)), ("actualLabor", N(row.ActualLaborDays)),
                ("completion", N(row.CompletionPercent)), ("delayed", row.PlansDelayed.ToString()),
                ("scheduleVar", row.AverageScheduleVarianceDays?.ToString() ?? "")));

        report.Totals["plannedArea"] = Math.Round(rows.Sum(x => x.PlannedAreaHa), 2);
        report.Totals["actualArea"] = Math.Round(rows.Sum(x => x.ActualAreaHa), 2);
        return report;
    }
}
