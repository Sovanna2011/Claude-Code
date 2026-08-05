using SugarcanePlanning.Application.Services;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;
using Xunit;

namespace SugarcanePlanning.UnitTests;

/// <summary>Unit-level checks of the planning engines that do not need a database.</summary>
public class ActivityDependencyGraphTests
{
    [Fact]
    public void A_new_edge_that_closes_a_loop_is_rejected()
    {
        // 2 waits for 1, 3 waits for 2. Making 1 wait for 3 would close the loop.
        var edges = new List<(int ActivityId, int PredecessorId)> { (2, 1), (3, 2) };

        Assert.True(ActivityMasterService.CreatesCycle(edges, activityId: 1, predecessorId: 3));
        Assert.False(ActivityMasterService.CreatesCycle(edges, activityId: 4, predecessorId: 3));
    }

    [Fact]
    public void An_unrelated_branch_does_not_count_as_a_cycle()
    {
        var edges = new List<(int, int)> { (2, 1), (3, 1), (5, 4) };
        Assert.False(ActivityMasterService.CreatesCycle(edges, activityId: 4, predecessorId: 3));
    }
}

public class ActivityPlanSchedulingTests
{
    private static PlantingActivity Activity(decimal capacityPerDay) => new()
    {
        Code = "A012",
        Name = "Planting",
        StandardCapacityPerDay = capacityPerDay,
        StandardDurationPerHa = 2m,
        StandardLaborDaysPerHa = 2.5m,
        RequiresTractor = true,
        RequiresLabor = true
    };

    [Fact]
    public void Duration_is_area_divided_by_daily_capacity_rounded_up()
    {
        Assert.Equal(25, ActivityPlanService.EstimateDurationDays(100m, Activity(4m)));
        Assert.Equal(26, ActivityPlanService.EstimateDurationDays(101m, Activity(4m)));
    }

    [Fact]
    public void An_activity_without_a_capacity_figure_takes_a_single_day()
        => Assert.Equal(1, ActivityPlanService.EstimateDurationDays(100m, Activity(0m)));

    [Fact]
    public void AddWorkingDays_skips_the_rest_days()
    {
        var monday = new DateOnly(2026, 3, 2);

        // Five working days starting Monday, resting on Sunday only, ends on Friday.
        Assert.Equal(new DateOnly(2026, 3, 6),
            ActivityPlanService.AddWorkingDays(monday, 5, saturday: true, sunday: false));

        // With Saturday off too, six working days reach the following Monday.
        Assert.Equal(new DateOnly(2026, 3, 9),
            ActivityPlanService.AddWorkingDays(monday, 6, saturday: false, sunday: false));
    }

    [Fact]
    public void A_start_on_a_rest_day_moves_to_the_next_working_day()
    {
        var sunday = new DateOnly(2026, 3, 8);
        Assert.Equal(new DateOnly(2026, 3, 9),
            ActivityPlanService.AddWorkingDays(sunday, 1, saturday: true, sunday: false));
    }

    [Fact]
    public void Recalculate_derives_target_hours_labor_and_machine_counts()
    {
        var activity = Activity(4m);
        var plan = new ActivityPlan
        {
            PlannedAreaHa = 100m,
            PlannedStartDate = new DateOnly(2026, 3, 2),
            PlannedEndDate = new DateOnly(2026, 3, 31)
        };

        plan.Recalculate(activity, litersPerHectare: 20m, workOnSaturday: true, workOnSunday: false);

        Assert.Equal(26, plan.WorkingDays);                    // March 2-31 without Sundays
        Assert.Equal(3.8462m, plan.DailyTargetHa);
        Assert.Equal(200m, plan.PlannedWorkingHours);          // 100 ha x 2 h/ha
        Assert.Equal(250m, plan.RequiredLaborDays);            // 100 ha x 2.5 labor-days/ha
        Assert.Equal(10, plan.RequiredWorkers);                // ceil(250 / 26)
        Assert.Equal(1, plan.RequiredTractorCount);            // ceil(100 / (4 x 26))
        Assert.Equal(2_000m, plan.PlannedFuelLiters);
    }
}

public class MaterialStandardResolutionTests
{
    private static ActivityMaterialStandard Standard(int id, int materialId, CropType crop,
        int? varietyId = null, SoilType? soil = null, decimal rate = 100m) => new()
    {
        Id = id,
        ActivityId = 1,
        MaterialId = materialId,
        CropType = crop,
        CaneVarietyId = varietyId,
        SoilType = soil,
        StandardRatePerHa = rate,
        IsActive = true,
        EffectiveFrom = new DateOnly(2026, 1, 1)
    };

    [Fact]
    public void The_most_specific_standard_wins_for_each_material()
    {
        var all = new List<ActivityMaterialStandard>
        {
            Standard(1, materialId: 10, CropType.Both, rate: 350m),
            Standard(2, materialId: 10, CropType.Both, soil: SoilType.Clay, rate: 400m),
            Standard(3, materialId: 10, CropType.NewPlanting, varietyId: 7, soil: SoilType.Clay, rate: 450m)
        };

        var chosen = MaterialRequirementService.ResolveStandards(all, activityId: 1, CropType.NewPlanting,
            varietyId: 7, soilType: SoilType.Clay, onDate: new DateOnly(2026, 3, 1));

        Assert.Single(chosen);
        Assert.Equal(450m, chosen[0].StandardRatePerHa);
    }

    [Fact]
    public void A_standard_for_another_crop_type_is_ignored()
    {
        var all = new List<ActivityMaterialStandard>
        {
            Standard(1, materialId: 10, CropType.Ratoon, rate: 200m),
            Standard(2, materialId: 10, CropType.Both, rate: 350m)
        };

        var chosen = MaterialRequirementService.ResolveStandards(all, 1, CropType.NewPlanting, 7,
            SoilType.Loam, new DateOnly(2026, 3, 1));

        Assert.Single(chosen);
        Assert.Equal(350m, chosen[0].StandardRatePerHa);
    }

    [Fact]
    public void A_standard_outside_its_effective_period_is_ignored()
    {
        var expired = Standard(1, 10, CropType.Both);
        expired.EffectiveTo = new DateOnly(2026, 2, 28);

        var chosen = MaterialRequirementService.ResolveStandards(new[] { expired }, 1, CropType.NewPlanting,
            7, SoilType.Loam, new DateOnly(2026, 3, 1));

        Assert.Empty(chosen);
    }

    [Fact]
    public void Each_material_contributes_exactly_one_row()
    {
        var all = new List<ActivityMaterialStandard>
        {
            Standard(1, materialId: 10, CropType.Both, rate: 350m),
            Standard(2, materialId: 11, CropType.Both, rate: 4m)
        };

        var chosen = MaterialRequirementService.ResolveStandards(all, 1, CropType.NewPlanting, 7,
            SoilType.Loam, new DateOnly(2026, 3, 1));

        Assert.Equal(2, chosen.Count);
        Assert.Equal(new[] { 10, 11 }, chosen.Select(s => s.MaterialId).OrderBy(x => x));
    }
}

public class ScenarioLeverTests
{
    [Fact]
    public void Adjustments_are_folded_into_the_lever_object()
    {
        var adjustments = new List<ScenarioAdjustment>
        {
            new() { AdjustmentType = ScenarioAdjustmentType.AdditionalRentalTractors, Value = 3 },
            new() { AdjustmentType = ScenarioAdjustmentType.ExtendedWorkingHours, Value = 2 },
            new() { AdjustmentType = ScenarioAdjustmentType.ReducedPlantingArea, Value = 10 },
            new() { AdjustmentType = ScenarioAdjustmentType.ChangedPlantingDates, Value = -7 },
            new() { AdjustmentType = ScenarioAdjustmentType.AdditionalWorkers, Value = 25 },
            new() { AdjustmentType = ScenarioAdjustmentType.AdditionalEquipment, Value = 2 }
        };

        var levers = ScenarioService.BuildLevers(adjustments,
            new[] { EquipmentCategory.DiscPlow, EquipmentCategory.CanePlanter });

        Assert.Equal(3m, levers.AdditionalTractors);
        Assert.Equal(2m, levers.ExtraHoursPerDay);
        Assert.Equal(10m, levers.AreaReductionPercent);
        Assert.Equal(-7, levers.DateShiftDays);
        Assert.Equal(25m, levers.AdditionalWorkers);
        Assert.Equal(2m, levers.AdditionalEquipmentByCategory[EquipmentCategory.DiscPlow]);
        Assert.Equal(2m, levers.AdditionalEquipmentByCategory[EquipmentCategory.CanePlanter]);
    }

    [Fact]
    public void Repeated_adjustments_of_the_same_type_accumulate()
    {
        var levers = ScenarioService.BuildLevers(new List<ScenarioAdjustment>
        {
            new() { AdjustmentType = ScenarioAdjustmentType.AdditionalRentalTractors, Value = 2 },
            new() { AdjustmentType = ScenarioAdjustmentType.AdditionalRentalTractors, Value = 3 }
        }, Array.Empty<EquipmentCategory>());

        Assert.Equal(5m, levers.AdditionalTractors);
    }
}

public class ExecutionStatusTests
{
    private static ActivityPlan Plan() => new()
    {
        PlannedAreaHa = 100m,
        PlannedStartDate = new DateOnly(2026, 3, 2),
        PlannedEndDate = new DateOnly(2026, 3, 20),
        Status = ActivityStatus.Scheduled
    };

    [Fact]
    public void A_finished_activity_becomes_Completed()
    {
        var plan = Plan();
        var actual = new ActivityActual
        {
            ActualStartDate = new DateOnly(2026, 3, 2),
            ActualCompletionDate = new DateOnly(2026, 3, 18),
            ActualCompletedAreaHa = 100m
        };
        actual.Recalculate(plan);

        Assert.Equal(100m, actual.CompletionPercent);
        Assert.Equal(-2, actual.ScheduleVarianceDays);
        Assert.Equal(ActivityStatus.Completed,
            ExecutionService.DeriveStatus(plan, actual, new DateOnly(2026, 3, 18)));
    }

    [Fact]
    public void An_activity_past_its_planned_end_becomes_Delayed()
    {
        var plan = Plan();
        var actual = new ActivityActual
        {
            ActualStartDate = new DateOnly(2026, 3, 2),
            ActualCompletedAreaHa = 40m
        };
        actual.Recalculate(plan);

        Assert.Equal(40m, actual.CompletionPercent);
        Assert.Equal(ActivityStatus.Delayed,
            ExecutionService.DeriveStatus(plan, actual, new DateOnly(2026, 3, 25)));
    }

    [Fact]
    public void An_activity_still_inside_its_window_is_InProgress()
    {
        var plan = Plan();
        var actual = new ActivityActual
        {
            ActualStartDate = new DateOnly(2026, 3, 2),
            ActualCompletedAreaHa = 40m
        };
        actual.Recalculate(plan);

        Assert.Equal(ActivityStatus.InProgress,
            ExecutionService.DeriveStatus(plan, actual, new DateOnly(2026, 3, 10)));
    }

    [Fact]
    public void Nothing_recorded_leaves_the_plan_status_untouched()
    {
        var plan = Plan();
        var actual = new ActivityActual();
        actual.Recalculate(plan);

        Assert.Equal(ActivityStatus.Scheduled,
            ExecutionService.DeriveStatus(plan, actual, new DateOnly(2026, 3, 10)));
    }
}

public class DomainEntityTests
{
    [Fact]
    public void ProjectionLine_Recalculate_derives_area_production_and_harvest_date()
    {
        var line = new ProjectionLine
        {
            ProjectedPlantingAreaHa = 120m,
            ExpectedLossPercent = 5m,
            ExpectedYieldPerHa = 90m,
            PlannedPlantingEnd = new DateOnly(2026, 4, 30)
        };

        line.Recalculate(growingPeriodMonths: 12);

        Assert.Equal(114m, line.HarvestableAreaHa);
        Assert.Equal(10_260m, line.ExpectedCaneProductionTons);
        Assert.Equal(new DateOnly(2027, 4, 30), line.ExpectedHarvestDate);
    }

    [Fact]
    public void Projection_totals_are_the_sum_of_its_live_lines()
    {
        var projection = new PlantingProjection();
        projection.Lines.Add(new ProjectionLine { ProjectedPlantingAreaHa = 100m, ExpectedCaneProductionTons = 8_000m });
        projection.Lines.Add(new ProjectionLine { ProjectedPlantingAreaHa = 50m, ExpectedCaneProductionTons = 4_000m });
        projection.Lines.Add(new ProjectionLine { ProjectedPlantingAreaHa = 25m, ExpectedCaneProductionTons = 2_000m, IsDeleted = true });

        projection.RecalculateTotals();

        Assert.Equal(150m, projection.TotalProjectedAreaHa);
        Assert.Equal(12_000m, projection.TotalExpectedProductionTons);
    }

    [Theory]
    [InlineData(ProjectionStatus.Draft, true)]
    [InlineData(ProjectionStatus.Rejected, true)]
    [InlineData(ProjectionStatus.Revised, true)]
    [InlineData(ProjectionStatus.Submitted, false)]
    [InlineData(ProjectionStatus.Approved, false)]
    [InlineData(ProjectionStatus.Closed, false)]
    public void Only_draft_rejected_and_revised_documents_are_editable(ProjectionStatus status, bool expected)
        => Assert.Equal(expected, new PlantingProjection { Status = status }.IsEditable);

    [Fact]
    public void A_read_only_version_is_never_editable()
        => Assert.False(new PlantingProjection { Status = ProjectionStatus.Draft, IsReadOnly = true }.IsEditable);

    [Fact]
    public void A_machine_in_its_maintenance_window_cannot_be_booked()
    {
        var tractor = new Tractor
        {
            IsActive = true,
            Availability = AvailabilityStatus.Available,
            MaintenanceFromDate = new DateOnly(2026, 3, 10),
            MaintenanceToDate = new DateOnly(2026, 3, 20)
        };

        Assert.False(tractor.IsBookableOn(new DateOnly(2026, 3, 15)));
        Assert.True(tractor.IsBookableOn(new DateOnly(2026, 3, 21)));

        tractor.Availability = AvailabilityStatus.Breakdown;
        Assert.False(tractor.IsBookableOn(new DateOnly(2026, 3, 21)));
    }

    [Fact]
    public void The_season_planting_window_bounds_every_projection_line()
    {
        var season = new GrowingSeason
        {
            PlannedPlantingStart = new DateOnly(2026, 3, 1),
            PlannedPlantingEnd = new DateOnly(2026, 8, 31)
        };

        Assert.True(season.ContainsPlantingWindow(new DateOnly(2026, 3, 1), new DateOnly(2026, 8, 31)));
        Assert.False(season.ContainsPlantingWindow(new DateOnly(2026, 2, 28), new DateOnly(2026, 4, 1)));
        Assert.False(season.ContainsPlantingWindow(new DateOnly(2026, 8, 1), new DateOnly(2026, 9, 1)));
    }

    [Fact]
    public void A_variety_planting_window_can_wrap_across_the_year_end()
    {
        var variety = new CaneVariety { RecommendedPlantingStartMonth = 10, RecommendedPlantingEndMonth = 2 };

        Assert.True(variety.IsRecommendedForMonth(11));
        Assert.True(variety.IsRecommendedForMonth(1));
        Assert.False(variety.IsRecommendedForMonth(6));
    }

    [Fact]
    public void The_material_rate_band_is_enforced_only_where_limits_are_set()
    {
        var material = new Material { MinApplicationRate = 100m, MaxApplicationRate = 400m };
        Assert.True(material.IsRateWithinBand(250m));
        Assert.False(material.IsRateWithinBand(50m));
        Assert.False(material.IsRateWithinBand(500m));

        var unbounded = new Material();
        Assert.True(unbounded.IsRateWithinBand(9_999m));
    }

    [Fact]
    public void Specificity_score_orders_standards_from_generic_to_exact()
    {
        Assert.Equal(0, new ActivityMaterialStandard { CropType = CropType.Both }.SpecificityScore());
        Assert.Equal(4, new ActivityMaterialStandard { CropType = CropType.NewPlanting }.SpecificityScore());
        Assert.Equal(7, new ActivityMaterialStandard
        {
            CropType = CropType.NewPlanting,
            CaneVarietyId = 1,
            SoilType = SoilType.Clay
        }.SpecificityScore());
    }
}
