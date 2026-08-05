using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Capacity;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Execution;
using SugarcanePlanning.Contracts.Labor;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Contracts.Reports;
using SugarcanePlanning.Contracts.Scheduling;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// Drives the whole process of section 24: approve a projection, generate the activity plan,
/// calculate materials, schedule resources, analyse capacity and record actual progress.
/// </summary>
public class PlanningEngineTests : IDisposable
{
    private readonly PlanningTestHost _host = new();
    public void Dispose() => _host.Dispose();

    /// <summary>Creates and approves a projection so the downstream engines have something to work on.</summary>
    private async Task<ProjectionDetailDto> ApprovedProjectionAsync(decimal areaHa = 80m)
    {
        var projection = await _host.Projections.CreateAsync(_host.NewProjection(areaHa));
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        return await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });
    }

    private async Task<ProjectionDetailDto> PlannedProjectionAsync(decimal areaHa = 80m)
    {
        var projection = await ApprovedProjectionAsync(areaHa);
        await _host.ActivityPlans.GenerateAsync(new GenerateActivityPlanRequest { ProjectionId = projection.Id });
        return projection;
    }

    // ------------------------------------------------------- activity planning

    [Fact]
    public async Task Generating_the_plan_creates_one_row_per_activity_and_block()
    {
        var projection = await ApprovedProjectionAsync();

        var result = await _host.ActivityPlans.GenerateAsync(new GenerateActivityPlanRequest { ProjectionId = projection.Id });

        Assert.Equal(3, result.PlansCreated);                 // plough, plant, fertilise
        Assert.Equal(0, result.PlansRemoved);
        Assert.NotNull(result.EarliestStart);

        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());
        Assert.Equal(3, plans.TotalCount);
    }

    [Fact]
    public async Task The_start_day_offset_places_land_preparation_before_planting()
    {
        var projection = await PlannedProjectionAsync();
        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());

        var plough = plans.Items.Single(p => p.ActivityId == _host.PloughActivityId);
        var plant = plans.Items.Single(p => p.ActivityId == _host.PlantActivityId);

        // Planting starts on the line's planting start; ploughing has a −20 day offset.
        Assert.Equal(new DateOnly(2026, 2, 10), plough.PlannedStartDate);
        Assert.True(plough.PlannedEndDate < plant.PlannedStartDate);
    }

    [Fact]
    public async Task A_blocking_dependency_pushes_the_successor_past_its_predecessor()
    {
        var projection = await PlannedProjectionAsync();
        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());

        var plant = plans.Items.Single(p => p.ActivityId == _host.PlantActivityId);
        var fertilise = plans.Items.Single(p => p.ActivityId == _host.FertiliseActivityId);

        // Fertilising waits for planting plus one lag day.
        Assert.True(fertilise.PlannedStartDate > plant.PlannedEndDate);
    }

    [Fact]
    public async Task Derived_figures_follow_the_formulas_of_section_7_and_8()
    {
        var projection = await PlannedProjectionAsync(areaHa: 80m);
        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());
        var plant = plans.Items.Single(p => p.ActivityId == _host.PlantActivityId);

        Assert.Equal(80m, plant.PlannedAreaHa);
        Assert.Equal(176m, plant.PlannedWorkingHours);            // 80 ha x 2.2 h/ha
        Assert.Equal(200m, plant.RequiredLaborDays);              // 80 ha x 2.5 labor-days/ha
        Assert.True(plant.WorkingDays > 0);
        Assert.Equal(Math.Round(80m / plant.WorkingDays, 4), plant.DailyTargetHa);
        Assert.True(plant.RequiredTractorCount >= 1);
    }

    [Fact]
    public async Task Regenerating_replaces_the_previous_plan()
    {
        var projection = await PlannedProjectionAsync();

        var result = await _host.ActivityPlans.GenerateAsync(new GenerateActivityPlanRequest
        {
            ProjectionId = projection.Id,
            Regenerate = true
        });

        Assert.Equal(3, result.PlansRemoved);
        Assert.Equal(3, result.PlansCreated);
    }

    [Fact]
    public async Task A_draft_projection_cannot_be_turned_into_an_activity_plan()
    {
        var projection = await _host.Projections.CreateAsync(_host.NewProjection());

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() =>
            _host.ActivityPlans.GenerateAsync(new GenerateActivityPlanRequest { ProjectionId = projection.Id }));

        Assert.Equal("NOT_APPROVED", ex.Code);
    }

    [Fact]
    public async Task The_gantt_view_carries_the_dependency_arrows()
    {
        var projection = await PlannedProjectionAsync();
        var gantt = await _host.ActivityPlans.GetGanttAsync(projection.Id, null);

        Assert.Equal(3, gantt.Bars.Count);
        Assert.True(gantt.RangeEnd >= gantt.RangeStart);

        var fertilise = gantt.Bars.Single(b => b.ActivityName == "Basal fertilizer");
        Assert.Single(fertilise.PredecessorPlanIds);
    }

    // ------------------------------------------------------ material planning

    [Fact]
    public async Task Material_requirements_apply_the_rate_applications_and_waste()
    {
        var projection = await PlannedProjectionAsync(areaHa: 80m);

        var rows = await _host.MaterialRequirements.GetRequirementsAsync(
            new MaterialRequirementQuery { ProjectionId = projection.Id, GroupBy = "material" });

        var seed = rows.Single(r => r.MaterialCode == "MAT-SEED");
        Assert.Equal(640m, seed.BaseRequirement);                 // 80 ha x 8 t/ha
        Assert.Equal(32m, seed.WasteQuantity);                    // 5 % waste
        Assert.Equal(672m, seed.TotalRequirement);

        var fertiliser = rows.Single(r => r.MaterialCode == "MAT-NPK");
        Assert.Equal(28_000m, fertiliser.BaseRequirement);         // 80 ha x 350 kg/ha
        Assert.Equal(560m, fertiliser.WasteQuantity);              // 2 % waste
        Assert.Equal(28_560m, fertiliser.TotalRequirement);
    }

    [Fact]
    public async Task Requirements_are_netted_against_stock_and_flag_a_shortage()
    {
        var projection = await PlannedProjectionAsync(areaHa: 80m);

        var rows = await _host.MaterialRequirements.GetRequirementsAsync(
            new MaterialRequirementQuery { ProjectionId = projection.Id, GroupBy = "material" });

        // Seed cane: 500 available, no incoming, none reserved -> 172 t short of 672 t.
        var seed = rows.Single(r => r.MaterialCode == "MAT-SEED");
        Assert.Equal(500m, seed.NetAvailableQuantity);
        Assert.Equal(172m, seed.ShortageQuantity);
        Assert.Equal(0m, seed.SurplusQuantity);
        Assert.Equal(CapacityStatus.Shortage, seed.Status);

        // Fertiliser: 10,000 + 5,000 − 500 = 14,500 kg net, comfortably below the 28,560 kg need.
        var fertiliser = rows.Single(r => r.MaterialCode == "MAT-NPK");
        Assert.Equal(14_500m, fertiliser.NetAvailableQuantity);
        Assert.Equal(14_060m, fertiliser.ShortageQuantity);
    }

    [Fact]
    public async Task The_shortage_filter_narrows_the_result()
    {
        var projection = await PlannedProjectionAsync(areaHa: 10m);   // small enough to be covered

        var all = await _host.MaterialRequirements.GetRequirementsAsync(
            new MaterialRequirementQuery { ProjectionId = projection.Id });
        var shortages = await _host.MaterialRequirements.GetRequirementsAsync(
            new MaterialRequirementQuery { ProjectionId = projection.Id, ShortagesOnly = true });

        Assert.Equal(2, all.Count);
        Assert.True(shortages.Count < all.Count);
    }

    [Fact]
    public async Task Requirements_can_be_grouped_by_activity()
    {
        var projection = await PlannedProjectionAsync();

        var rows = await _host.MaterialRequirements.GetRequirementsAsync(
            new MaterialRequirementQuery { ProjectionId = projection.Id, GroupBy = "activity" });

        Assert.All(rows, row => Assert.NotNull(row.ActivityName));
        Assert.Contains(rows, row => row.ActivityName == "Planting");
        Assert.Contains(rows, row => row.ActivityName == "Basal fertilizer");
    }

    [Fact]
    public async Task The_stock_interface_updates_the_net_available_quantity()
    {
        var projection = await PlannedProjectionAsync(areaHa: 80m);

        await _host.MaterialMaster.SyncStockAsync(new List<MaterialStockSyncDto>
        {
            new() { MaterialCode = "MAT-SEED", EstateCode = "E1", AvailableStock = 900m, IncomingQuantity = 100m, SourceSystem = "ERP" }
        });

        var rows = await _host.MaterialRequirements.GetRequirementsAsync(
            new MaterialRequirementQuery { ProjectionId = projection.Id });

        var seed = rows.Single(r => r.MaterialCode == "MAT-SEED");
        Assert.Equal(1_000m, seed.NetAvailableQuantity);
        Assert.Equal(0m, seed.ShortageQuantity);
        Assert.Equal(328m, seed.SurplusQuantity);
    }

    // ------------------------------------------------------------- fuel & labor

    [Fact]
    public async Task The_fuel_projection_reports_both_bases_and_takes_the_larger()
    {
        var projection = await PlannedProjectionAsync();

        var rows = await _host.FuelLabor.GetFuelProjectionAsync(new FuelLaborQuery
        {
            ProjectionId = projection.Id,
            GroupBy = "activity"
        });

        Assert.NotEmpty(rows);
        Assert.All(rows, row => Assert.Equal(Math.Max(row.FuelByAreaLiters, row.FuelByHourLiters), row.ProjectedFuelLiters));
    }

    [Fact]
    public async Task The_labor_projection_grades_the_workforce_gap()
    {
        var projection = await PlannedProjectionAsync();

        var rows = await _host.FuelLabor.GetLaborProjectionAsync(new FuelLaborQuery
        {
            ProjectionId = projection.Id,
            GroupBy = "activity"
        });

        var planting = rows.Single(r => r.GroupName == "Planting");
        Assert.Equal(200m, planting.RequiredLaborDays);
        Assert.True(planting.RequiredWorkers > 0);
        // One seeded operator against a heavy planting crew requirement.
        Assert.Equal(CapacityStatus.Shortage, planting.Status);
    }

    // ---------------------------------------------------------------- capacity

    [Fact]
    public async Task Capacity_analysis_covers_machines_labor_and_materials()
    {
        var projection = await PlannedProjectionAsync();

        var analysis = await _host.Capacity.AnalyzeAsync(projection.Id);

        Assert.Contains(analysis.Lines, l => l.ResourceType == "Tractor");
        Assert.Contains(analysis.Lines, l => l.ResourceType == "Equipment");
        Assert.Contains(analysis.Lines, l => l.ResourceType == "Labor");
        Assert.Contains(analysis.Lines, l => l.ResourceType == "Material");
        Assert.Contains(analysis.Lines, l => l.ResourceType == "DailyCapacity");
        Assert.True(analysis.WorkingDays > 0);
        Assert.NotEqual(CapacityStatus.Sufficient, analysis.OverallStatus);   // seed cane is short
    }

    [Fact]
    public async Task A_scenario_improves_the_gap_without_touching_the_plan()
    {
        var projection = await PlannedProjectionAsync();

        var scenario = await _host.Scenarios.CreateAsync(new ScenarioUpsertDto
        {
            ProjectionId = projection.Id,
            Name = "Add rentals and hours",
            Adjustments =
            {
                new ScenarioAdjustmentUpsertDto { AdjustmentType = ScenarioAdjustmentType.AdditionalRentalTractors, Value = 4 },
                new ScenarioAdjustmentUpsertDto { AdjustmentType = ScenarioAdjustmentType.AdditionalWorkers, Value = 200 },
                new ScenarioAdjustmentUpsertDto { AdjustmentType = ScenarioAdjustmentType.ReducedPlantingArea, Value = 20 }
            }
        });

        var result = await _host.Scenarios.SimulateAsync(scenario.Id);

        Assert.NotEmpty(result.Deltas);
        var labor = result.Deltas.Single(d => d.ResourceType == "Labor");
        Assert.True(labor.SimulatedGap > labor.BaselineGap);

        // The stored plan is untouched: the baseline still reports the original figures.
        var baseline = await _host.Capacity.AnalyzeAsync(projection.Id);
        Assert.Equal(result.Baseline.TotalPlannedAreaHa, baseline.TotalPlannedAreaHa);

        var reloaded = await _host.Scenarios.GetScenariosAsync(projection.Id);
        Assert.Equal(ScenarioStatus.Simulated, reloaded.Single().Status);
    }

    // -------------------------------------------------------------- execution

    [Fact]
    public async Task Recording_actuals_derives_the_variances_and_the_plan_status()
    {
        var projection = await PlannedProjectionAsync(areaHa: 80m);
        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());
        var plant = plans.Items.Single(p => p.ActivityId == _host.PlantActivityId);

        var actual = await _host.Execution.RecordAsync(new ActivityActualUpsertDto
        {
            ActivityPlanId = plant.Id,
            ActualStartDate = plant.PlannedStartDate,
            ActualCompletionDate = plant.PlannedEndDate.AddDays(3),
            ActualCompletedAreaHa = 72m,
            ActualWorkingHours = 190m,
            ActualFuelLiters = 2_100m,
            ActualLaborDays = 210m,
            DelayReason = "Rain",
            MaterialUsages = { new ActualMaterialUsageUpsertDto { MaterialId = _host.SeedMaterialId, ActualQuantity = 700m } }
        });

        Assert.Equal(-8m, actual.AreaVariance);                 // 72 − 80
        Assert.Equal(90m, actual.CompletionPercent);
        Assert.Equal(3, actual.ScheduleVarianceDays);
        Assert.Equal(28m, actual.MaterialUsages.Single().Variance);   // 700 − 672 planned

        var reloaded = await _host.ActivityPlans.GetPlanAsync(plant.Id);
        Assert.Equal(ActivityStatus.Delayed, reloaded.Status);
        Assert.Equal(90m, reloaded.CompletionPercent);
    }

    [Fact]
    public async Task Projection_versus_actual_aggregates_by_block()
    {
        var projection = await PlannedProjectionAsync(areaHa: 80m);
        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());
        var plant = plans.Items.Single(p => p.ActivityId == _host.PlantActivityId);

        await _host.Execution.RecordAsync(new ActivityActualUpsertDto
        {
            ActivityPlanId = plant.Id,
            ActualStartDate = plant.PlannedStartDate,
            ActualCompletionDate = plant.PlannedEndDate,
            ActualCompletedAreaHa = 80m,
            ActualFuelLiters = 1_800m
        });

        var comparison = await _host.Execution.CompareAsync(projection.Id, "block");

        var row = Assert.Single(comparison);
        Assert.Equal(80m, row.ActualAreaHa);
        Assert.Equal(3, row.PlansTotal);
        Assert.Equal(1, row.PlansCompleted);
    }

    [Fact]
    public async Task Recording_actuals_twice_updates_the_same_row()
    {
        var projection = await PlannedProjectionAsync();
        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());
        var plant = plans.Items.Single(p => p.ActivityId == _host.PlantActivityId);

        await _host.Execution.RecordAsync(new ActivityActualUpsertDto
        {
            ActivityPlanId = plant.Id,
            ActualStartDate = plant.PlannedStartDate,
            ActualCompletedAreaHa = 40m
        });
        var second = await _host.Execution.RecordAsync(new ActivityActualUpsertDto
        {
            ActivityPlanId = plant.Id,
            ActualStartDate = plant.PlannedStartDate,
            ActualCompletedAreaHa = 60m
        });

        Assert.Equal(60m, second.ActualCompletedAreaHa);
        Assert.Equal(1, _host.Db.ActivityActuals.Count(a => a.ActivityPlanId == plant.Id));
    }

    [Fact]
    public async Task An_actual_completion_before_the_start_is_refused()
    {
        var projection = await PlannedProjectionAsync();
        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() => _host.Execution.RecordAsync(new ActivityActualUpsertDto
        {
            ActivityPlanId = plans.Items[0].Id,
            ActualStartDate = new DateOnly(2026, 4, 10),
            ActualCompletionDate = new DateOnly(2026, 4, 1)
        }));

        Assert.Equal("DATE_RANGE", ex.Code);
    }

    // --------------------------------------------------------------- dashboard

    [Fact]
    public async Task The_dashboard_summarises_the_season()
    {
        var projection = await PlannedProjectionAsync(areaHa: 80m);
        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());
        var plant = plans.Items.Single(p => p.ActivityId == _host.PlantActivityId);

        await _host.Execution.RecordAsync(new ActivityActualUpsertDto
        {
            ActivityPlanId = plant.Id,
            ActualStartDate = plant.PlannedStartDate,
            ActualCompletionDate = plant.PlannedEndDate,
            ActualCompletedAreaHa = 40m
        });

        var dashboard = await _host.Dashboard.GetAsync(_host.SeasonId, null);

        Assert.Equal(80m, dashboard.TotalProjectedAreaHa);
        Assert.Equal(80m, dashboard.NewPlantingAreaHa);
        Assert.Equal(0m, dashboard.RatoonAreaHa);
        Assert.Equal(40m, dashboard.ActualPlantedAreaHa);
        Assert.Equal(50m, dashboard.OverallCompletionPercent);
        Assert.NotEmpty(dashboard.AreaByFarm);
        Assert.NotEmpty(dashboard.MonthlyTargets);
        Assert.True(dashboard.MaterialShortageCount > 0);
    }

    // ----------------------------------------------------------------- reports

    [Theory]
    [InlineData(ReportKey.PlantingProjection)]
    [InlineData(ReportKey.MonthlyPlan)]
    [InlineData(ReportKey.FarmZoneBlockPlan)]
    [InlineData(ReportKey.NewPlantingAndRatoon)]
    [InlineData(ReportKey.ActivitySchedule)]
    [InlineData(ReportKey.ActivityCalendar)]
    [InlineData(ReportKey.TractorRequirement)]
    [InlineData(ReportKey.EquipmentRequirement)]
    [InlineData(ReportKey.TractorUtilization)]
    [InlineData(ReportKey.EquipmentUtilization)]
    [InlineData(ReportKey.TractorShortage)]
    [InlineData(ReportKey.EquipmentShortage)]
    [InlineData(ReportKey.SeedCane)]
    [InlineData(ReportKey.Fertilizer)]
    [InlineData(ReportKey.Chemicals)]
    [InlineData(ReportKey.MaterialShortage)]
    [InlineData(ReportKey.Fuel)]
    [InlineData(ReportKey.Labor)]
    [InlineData(ReportKey.CapacityAnalysis)]
    [InlineData(ReportKey.DelayedActivities)]
    [InlineData(ReportKey.ProjectionVsActual)]
    public async Task Every_report_produces_columns_and_a_title(ReportKey key)
    {
        var projection = await PlannedProjectionAsync();

        var report = await _host.Reports.GenerateAsync(key, new ReportRequest { ProjectionId = projection.Id });

        Assert.Equal(key, report.Report);
        Assert.False(string.IsNullOrWhiteSpace(report.Title));
        Assert.NotEmpty(report.Columns);
        Assert.Equal("tester", report.GeneratedBy);
    }

    [Fact]
    public async Task The_revision_comparison_report_needs_two_versions()
    {
        var projection = await PlannedProjectionAsync();

        await Assert.ThrowsAsync<BusinessRuleException>(() => _host.Reports.GenerateAsync(
            ReportKey.RevisionComparison, new ReportRequest { ProjectionId = projection.Id }));

        var revision = await _host.Projections.ReviseAsync(projection.Id,
            new ReviseProjectionDto { RevisionReason = "compare me" });

        var report = await _host.Reports.GenerateAsync(ReportKey.RevisionComparison,
            new ReportRequest { ProjectionId = projection.Id, CompareToProjectionId = revision.Id });

        Assert.NotEmpty(report.Columns);
    }

    [Fact]
    public async Task A_report_can_be_sorted_by_any_column()
    {
        var projection = await PlannedProjectionAsync();

        var ascending = await _host.Reports.GenerateAsync(ReportKey.ActivitySchedule,
            new ReportRequest { ProjectionId = projection.Id, SortBy = "area", SortDescending = false });
        var descending = await _host.Reports.GenerateAsync(ReportKey.ActivitySchedule,
            new ReportRequest { ProjectionId = projection.Id, SortBy = "area", SortDescending = true });

        Assert.Equal(ascending.Rows.Count, descending.Rows.Count);
        Assert.Equal(ascending.Rows.First().Values["activity"], descending.Rows.Last().Values["activity"]);
    }

    // ----------------------------------------------------------------- lookups

    [Fact]
    public async Task Lookups_return_the_seeded_master_data()
    {
        Assert.NotEmpty(await _host.Lookups.GetAsync("blocks", null));
        Assert.NotEmpty(await _host.Lookups.GetAsync("varieties", null));
        Assert.NotEmpty(await _host.Lookups.GetAsync("activities", null));
        Assert.NotEmpty(await _host.Lookups.GetAsync("tractors", null));
        await Assert.ThrowsAsync<NotFoundException>(() => _host.Lookups.GetAsync("nonsense", null));
    }

    [Fact]
    public async Task Blocks_can_be_filtered_by_their_parent_zone()
    {
        var blocks = await _host.Lookups.GetAsync("blocks", _host.ZoneId);
        Assert.Equal(2, blocks.Count);
        Assert.All(blocks, b => Assert.Equal(_host.ZoneId, b.ParentId));
    }
}
