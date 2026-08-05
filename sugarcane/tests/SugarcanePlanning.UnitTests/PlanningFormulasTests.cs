using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Enums;
using Xunit;

namespace SugarcanePlanning.UnitTests;

/// <summary>Covers every formula in the specification, including its worked example.</summary>
public class PlanningFormulasTests
{
    // ---------------------------------------------------------------- section 5

    [Theory]
    [InlineData(100, 0, 100)]
    [InlineData(100, 5, 95)]
    [InlineData(2600, 12.5, 2275)]
    public void HarvestableArea_applies_the_loss_percentage(decimal area, decimal loss, decimal expected)
        => Assert.Equal(expected, PlanningFormulas.HarvestableArea(area, loss));

    [Fact]
    public void ExpectedCaneProduction_multiplies_harvestable_area_by_yield()
        => Assert.Equal(8_740m, PlanningFormulas.ExpectedCaneProduction(95m, 92m));

    [Theory]
    [InlineData(-1, 5)]
    [InlineData(100, 101)]
    [InlineData(100, -1)]
    public void HarvestableArea_rejects_impossible_inputs(decimal area, decimal loss)
        => Assert.Throws<ArgumentOutOfRangeException>(() => PlanningFormulas.HarvestableArea(area, loss));

    // ---------------------------------------------------------------- section 7

    [Fact]
    public void DailyTarget_divides_area_by_working_days()
        => Assert.Equal(12.5m, PlanningFormulas.DailyTarget(250m, 20));

    [Fact]
    public void DailyTarget_rejects_zero_working_days()
        => Assert.Throws<ArgumentOutOfRangeException>(() => PlanningFormulas.DailyTarget(250m, 0));

    [Fact]
    public void WorkingDays_skips_the_configured_rest_days()
    {
        var monday = new DateOnly(2026, 3, 2);
        var sunday = new DateOnly(2026, 3, 8);

        Assert.Equal(7, PlanningFormulas.WorkingDays(monday, sunday, workOnSaturday: true, workOnSunday: true));
        Assert.Equal(6, PlanningFormulas.WorkingDays(monday, sunday, workOnSaturday: true, workOnSunday: false));
        Assert.Equal(5, PlanningFormulas.WorkingDays(monday, sunday, workOnSaturday: false, workOnSunday: false));
    }

    [Fact]
    public void WorkingDays_excludes_listed_holidays()
    {
        var holidays = new HashSet<DateOnly> { new(2026, 3, 3) };
        Assert.Equal(4, PlanningFormulas.WorkingDays(new DateOnly(2026, 3, 2), new DateOnly(2026, 3, 6),
            workOnSaturday: true, workOnSunday: false, holidays));
    }

    // ---------------------------------------------------------------- section 8

    [Fact]
    public void RequiredTractors_matches_the_worked_example_from_the_specification()
    {
        // 2,600 ha / (4 ha x 90 days) = 7.22 -> 8 tractors.
        Assert.Equal(8, PlanningFormulas.RequiredTractors(2_600m, 4m, 90));
        Assert.Equal(7.2222m, PlanningFormulas.ExactMachineRequirement(2_600m, 4m, 90));
    }

    [Fact]
    public void RequiredTractors_always_rounds_up()
    {
        Assert.Equal(1, PlanningFormulas.RequiredTractors(0.01m, 4m, 90));
        Assert.Equal(2, PlanningFormulas.RequiredTractors(361m, 4m, 90));
        Assert.Equal(1, PlanningFormulas.RequiredTractors(360m, 4m, 90));
    }

    [Fact]
    public void RequiredTractors_is_zero_when_there_is_no_area()
        => Assert.Equal(0, PlanningFormulas.RequiredTractors(0m, 4m, 90));

    [Fact]
    public void RequiredTractors_rejects_zero_capacity_or_days()
    {
        Assert.Throws<ArgumentOutOfRangeException>(() => PlanningFormulas.RequiredTractors(100m, 0m, 90));
        Assert.Throws<ArgumentOutOfRangeException>(() => PlanningFormulas.RequiredTractors(100m, 4m, 0));
    }

    // --------------------------------------------------------------- section 12

    [Fact]
    public void MaterialRequirement_adds_the_waste_percentage_on_top_of_the_base()
    {
        var baseQty = PlanningFormulas.BaseMaterialRequirement(100m, 350m);
        Assert.Equal(35_000m, baseQty);
        Assert.Equal(700m, PlanningFormulas.WasteQuantity(baseQty, 2m));
        Assert.Equal(35_700m, PlanningFormulas.TotalMaterialRequirement(100m, 350m, 2m));
    }

    [Fact]
    public void MaterialRequirement_multiplies_by_the_number_of_applications()
        => Assert.Equal(71_400m, PlanningFormulas.TotalMaterialRequirement(100m, 350m, 2m, numberOfApplications: 2));

    [Fact]
    public void RequiredSeedCane_uses_the_variety_seed_rate()
        => Assert.Equal(1_200m, PlanningFormulas.RequiredSeedCane(150m, 8m));

    [Fact]
    public void RequiredChemical_multiplies_area_rate_and_applications()
        => Assert.Equal(900m, PlanningFormulas.RequiredChemical(150m, 3m, 2));

    // --------------------------------------------------------------- section 13

    [Fact]
    public void NetAvailable_shortage_and_surplus_follow_the_specification()
    {
        var net = PlanningFormulas.NetAvailableQuantity(availableStock: 1_000m, incomingQuantity: 500m, reservedQuantity: 200m);
        Assert.Equal(1_300m, net);

        Assert.Equal(200m, PlanningFormulas.ShortageQuantity(1_500m, net));
        Assert.Equal(0m, PlanningFormulas.SurplusQuantity(1_500m, net));

        Assert.Equal(0m, PlanningFormulas.ShortageQuantity(1_000m, net));
        Assert.Equal(300m, PlanningFormulas.SurplusQuantity(1_000m, net));
    }

    // --------------------------------------------------------------- section 14

    [Fact]
    public void FuelProjection_supports_both_the_area_and_the_hour_basis()
    {
        Assert.Equal(4_500m, PlanningFormulas.ProjectedFuelByArea(100m, 45m));
        Assert.Equal(2_400m, PlanningFormulas.ProjectedFuelByHour(200m, 12m));
    }

    [Fact]
    public void RequiredWorkers_rounds_the_labor_days_up()
    {
        var laborDays = PlanningFormulas.RequiredLaborDays(250m, 2.5m);
        Assert.Equal(625m, laborDays);
        Assert.Equal(32, PlanningFormulas.RequiredWorkers(laborDays, 20));   // 31.25 -> 32
        Assert.Equal(0, PlanningFormulas.RequiredWorkers(0m, 20));
    }

    // --------------------------------------------------------------- section 15

    [Theory]
    [InlineData(0, 0, CapacityStatus.Sufficient)]
    [InlineData(10, 12, CapacityStatus.Sufficient)]
    [InlineData(10, 10, CapacityStatus.Sufficient)]
    [InlineData(10, 9.5, CapacityStatus.AtRisk)]
    [InlineData(10, 5, CapacityStatus.Shortage)]
    [InlineData(10, 0, CapacityStatus.Unavailable)]
    public void EvaluateCapacity_grades_coverage(decimal required, decimal available, CapacityStatus expected)
        => Assert.Equal(expected, PlanningFormulas.EvaluateCapacity(required, available));

    [Fact]
    public void CoveragePercent_is_capped_so_a_huge_surplus_stays_readable()
    {
        Assert.Equal(50m, PlanningFormulas.CoveragePercent(10m, 5m));
        Assert.Equal(999.99m, PlanningFormulas.CoveragePercent(1m, 100_000m));
        Assert.Equal(100m, PlanningFormulas.CoveragePercent(0m, 0m));
    }

    // --------------------------------------------------------------- section 17

    [Fact]
    public void Variances_are_actual_minus_planned()
    {
        Assert.Equal(-15m, PlanningFormulas.AreaVariance(85m, 100m));
        Assert.Equal(120m, PlanningFormulas.FuelVariance(1_120m, 1_000m));
        Assert.Equal(-30m, PlanningFormulas.MaterialVariance(270m, 300m));
    }

    [Fact]
    public void ScheduleVariance_is_expressed_in_days_and_needs_both_dates()
    {
        Assert.Equal(3, PlanningFormulas.ScheduleVarianceDays(new DateOnly(2026, 4, 13), new DateOnly(2026, 4, 10)));
        Assert.Equal(-2, PlanningFormulas.ScheduleVarianceDays(new DateOnly(2026, 4, 8), new DateOnly(2026, 4, 10)));
        Assert.Null(PlanningFormulas.ScheduleVarianceDays(null, new DateOnly(2026, 4, 10)));
    }

    [Fact]
    public void CompletionPercentage_handles_a_zero_plan()
    {
        Assert.Equal(85m, PlanningFormulas.CompletionPercentage(85m, 100m));
        Assert.Equal(0m, PlanningFormulas.CompletionPercentage(85m, 0m));
    }

    // ------------------------------------------------------ units and overlaps

    [Fact]
    public void UnitConversion_round_trips_through_the_base_unit()
    {
        Assert.Equal(500m, PlanningFormulas.ToBaseUnit(10m, 50m));       // 10 bags x 50 kg
        Assert.Equal(10m, PlanningFormulas.ToAlternativeUnit(500m, 50m));
        Assert.Throws<ArgumentOutOfRangeException>(() => PlanningFormulas.ToBaseUnit(10m, 0m));
    }

    [Fact]
    public void DateRangesOverlap_detects_touching_and_disjoint_ranges()
    {
        var a1 = new DateOnly(2026, 3, 1);
        var a2 = new DateOnly(2026, 3, 15);

        Assert.True(PlanningFormulas.DateRangesOverlap(a1, a2, new DateOnly(2026, 3, 15), new DateOnly(2026, 3, 20)));
        Assert.False(PlanningFormulas.DateRangesOverlap(a1, a2, new DateOnly(2026, 3, 16), new DateOnly(2026, 3, 20)));
        Assert.True(PlanningFormulas.DateRangesOverlap(a1, a2, new DateOnly(2026, 2, 20), new DateOnly(2026, 3, 2)));
    }

    [Fact]
    public void TimeRangesOverlap_treats_back_to_back_bookings_as_free()
    {
        var start = new DateTime(2026, 3, 2, 8, 0, 0, DateTimeKind.Utc);
        Assert.False(PlanningFormulas.TimeRangesOverlap(start, start.AddHours(4), start.AddHours(4), start.AddHours(8)));
        Assert.True(PlanningFormulas.TimeRangesOverlap(start, start.AddHours(4), start.AddHours(3), start.AddHours(8)));
    }
}
