using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Calculations;

/// <summary>
/// Every planning formula in the specification, implemented once as pure functions so
/// the API, the requirement engines, the reports and the unit tests all agree.
/// Nothing here touches the database.
/// </summary>
public static class PlanningFormulas
{
    /// <summary>Rounding used for every stored quantity (4 decimals = 1 m² on a hectare).</summary>
    public const int QuantityScale = 4;

    private static decimal Round(decimal value) => Math.Round(value, QuantityScale, MidpointRounding.AwayFromZero);

    // ---------------------------------------------------------------- section 5

    /// <summary>Harvestable Area = Projected Planting Area x (1 - Expected Loss % / 100).</summary>
    public static decimal HarvestableArea(decimal projectedPlantingArea, decimal expectedLossPercent)
    {
        if (projectedPlantingArea < 0) throw new ArgumentOutOfRangeException(nameof(projectedPlantingArea));
        if (expectedLossPercent is < 0 or > 100) throw new ArgumentOutOfRangeException(nameof(expectedLossPercent));
        return Round(projectedPlantingArea * (1m - expectedLossPercent / 100m));
    }

    /// <summary>Expected Cane Production = Harvestable Area x Expected Yield per Hectare.</summary>
    public static decimal ExpectedCaneProduction(decimal harvestableArea, decimal expectedYieldPerHectare)
    {
        if (harvestableArea < 0) throw new ArgumentOutOfRangeException(nameof(harvestableArea));
        if (expectedYieldPerHectare < 0) throw new ArgumentOutOfRangeException(nameof(expectedYieldPerHectare));
        return Round(harvestableArea * expectedYieldPerHectare);
    }

    // ---------------------------------------------------------------- section 7

    /// <summary>Daily Target = Planned Activity Area / Available Working Days.</summary>
    public static decimal DailyTarget(decimal plannedArea, int availableWorkingDays)
    {
        if (plannedArea < 0) throw new ArgumentOutOfRangeException(nameof(plannedArea));
        if (availableWorkingDays <= 0)
            throw new ArgumentOutOfRangeException(nameof(availableWorkingDays), "Available working days must be greater than zero.");
        return Round(plannedArea / availableWorkingDays);
    }

    /// <summary>
    /// Working days between two inclusive dates. When <paramref name="workOnSaturday"/> /
    /// <paramref name="workOnSunday"/> are false those weekdays are skipped.
    /// </summary>
    public static int WorkingDays(DateOnly start, DateOnly end, bool workOnSaturday = true, bool workOnSunday = false,
        IReadOnlySet<DateOnly>? holidays = null)
    {
        if (end < start) return 0;
        var days = 0;
        for (var d = start; d <= end; d = d.AddDays(1))
        {
            if (d.DayOfWeek == DayOfWeek.Saturday && !workOnSaturday) continue;
            if (d.DayOfWeek == DayOfWeek.Sunday && !workOnSunday) continue;
            if (holidays is not null && holidays.Contains(d)) continue;
            days++;
        }
        return days;
    }

    // ---------------------------------------------------------------- section 8

    /// <summary>
    /// Required Tractors = Planned Area / (Capacity per Tractor per Day x Available Working Days),
    /// always rounded <b>up</b>. Example: 2,600 / (4 x 90) = 7.22 → 8 tractors.
    /// </summary>
    public static int RequiredTractors(decimal plannedArea, decimal capacityPerTractorPerDay, int availableWorkingDays)
    {
        if (plannedArea <= 0) return 0;
        if (capacityPerTractorPerDay <= 0)
            throw new ArgumentOutOfRangeException(nameof(capacityPerTractorPerDay), "Capacity per tractor per day must be greater than zero.");
        if (availableWorkingDays <= 0)
            throw new ArgumentOutOfRangeException(nameof(availableWorkingDays), "Available working days must be greater than zero.");
        return (int)Math.Ceiling(plannedArea / (capacityPerTractorPerDay * availableWorkingDays));
    }

    /// <summary>Same arithmetic as <see cref="RequiredTractors"/>, applied to an implement.</summary>
    public static int RequiredEquipment(decimal plannedArea, decimal capacityPerUnitPerDay, int availableWorkingDays)
        => RequiredTractors(plannedArea, capacityPerUnitPerDay, availableWorkingDays);

    /// <summary>Un-rounded requirement, kept for reporting the exact ratio next to the rounded figure.</summary>
    public static decimal ExactMachineRequirement(decimal plannedArea, decimal capacityPerUnitPerDay, int availableWorkingDays)
    {
        if (plannedArea <= 0) return 0m;
        if (capacityPerUnitPerDay <= 0 || availableWorkingDays <= 0) return 0m;
        return Round(plannedArea / (capacityPerUnitPerDay * availableWorkingDays));
    }

    // --------------------------------------------------------------- section 12

    /// <summary>Base Material Requirement = Planned Area x Standard Rate per Hectare.</summary>
    public static decimal BaseMaterialRequirement(decimal plannedArea, decimal standardRatePerHectare)
        => Round(plannedArea * standardRatePerHectare);

    /// <summary>Waste Quantity = Base Requirement x Waste % / 100.</summary>
    public static decimal WasteQuantity(decimal baseRequirement, decimal wastePercent)
    {
        if (wastePercent < 0) throw new ArgumentOutOfRangeException(nameof(wastePercent));
        return Round(baseRequirement * wastePercent / 100m);
    }

    /// <summary>Total Material Requirement = Base Requirement + Waste Quantity.</summary>
    public static decimal TotalMaterialRequirement(decimal plannedArea, decimal standardRatePerHectare, decimal wastePercent,
        int numberOfApplications = 1)
    {
        if (numberOfApplications < 1) numberOfApplications = 1;
        var baseQty = BaseMaterialRequirement(plannedArea, standardRatePerHectare) * numberOfApplications;
        return Round(baseQty + WasteQuantity(baseQty, wastePercent));
    }

    /// <summary>Required Seed Cane = New Planting Area x Seed Rate per Hectare.</summary>
    public static decimal RequiredSeedCane(decimal newPlantingArea, decimal seedRatePerHectare)
        => Round(newPlantingArea * seedRatePerHectare);

    /// <summary>Required Chemical = Treatment Area x Application Rate x Number of Applications.</summary>
    public static decimal RequiredChemical(decimal treatmentArea, decimal applicationRate, int numberOfApplications)
    {
        if (numberOfApplications < 1) numberOfApplications = 1;
        return Round(treatmentArea * applicationRate * numberOfApplications);
    }

    // --------------------------------------------------------------- section 13

    /// <summary>Net Available = Available Stock + Incoming - Reserved.</summary>
    public static decimal NetAvailableQuantity(decimal availableStock, decimal incomingQuantity, decimal reservedQuantity)
        => Round(availableStock + incomingQuantity - reservedQuantity);

    /// <summary>Shortage = max(Total Requirement - Net Available, 0).</summary>
    public static decimal ShortageQuantity(decimal totalRequirement, decimal netAvailable)
        => Round(Math.Max(totalRequirement - netAvailable, 0m));

    /// <summary>Surplus = max(Net Available - Total Requirement, 0).</summary>
    public static decimal SurplusQuantity(decimal totalRequirement, decimal netAvailable)
        => Round(Math.Max(netAvailable - totalRequirement, 0m));

    // --------------------------------------------------------------- section 14

    /// <summary>Projected Fuel by Area = Planned Area x Standard Liters per Hectare.</summary>
    public static decimal ProjectedFuelByArea(decimal plannedArea, decimal litersPerHectare)
        => Round(plannedArea * litersPerHectare);

    /// <summary>Projected Fuel by Hour = Planned Working Hours x Standard Liters per Hour.</summary>
    public static decimal ProjectedFuelByHour(decimal plannedWorkingHours, decimal litersPerHour)
        => Round(plannedWorkingHours * litersPerHour);

    /// <summary>Required Labor-Days = Planned Area x Standard Labor-Days per Hectare.</summary>
    public static decimal RequiredLaborDays(decimal plannedArea, decimal laborDaysPerHectare)
        => Round(plannedArea * laborDaysPerHectare);

    /// <summary>Required Workers = Required Labor-Days / Available Working Days, rounded up.</summary>
    public static int RequiredWorkers(decimal requiredLaborDays, int availableWorkingDays)
    {
        if (requiredLaborDays <= 0) return 0;
        if (availableWorkingDays <= 0)
            throw new ArgumentOutOfRangeException(nameof(availableWorkingDays), "Available working days must be greater than zero.");
        return (int)Math.Ceiling(requiredLaborDays / availableWorkingDays);
    }

    // --------------------------------------------------------------- section 15

    /// <summary>
    /// Grades a requirement against availability: everything covered is
    /// <see cref="CapacityStatus.Sufficient"/>, nothing available at all is
    /// <see cref="CapacityStatus.Unavailable"/>, a coverage below
    /// <paramref name="atRiskThreshold"/> (default 90 %) is a shortage, otherwise at risk.
    /// </summary>
    public static CapacityStatus EvaluateCapacity(decimal required, decimal available, decimal atRiskThreshold = 0.90m)
    {
        if (required <= 0) return CapacityStatus.Sufficient;
        if (available <= 0) return CapacityStatus.Unavailable;
        if (available >= required) return CapacityStatus.Sufficient;
        var coverage = available / required;
        return coverage >= atRiskThreshold ? CapacityStatus.AtRisk : CapacityStatus.Shortage;
    }

    /// <summary>Coverage ratio expressed as a percentage (available / required x 100), capped at 999.99.</summary>
    public static decimal CoveragePercent(decimal required, decimal available)
    {
        if (required <= 0) return 100m;
        return Round(Math.Min(available / required * 100m, 999.99m));
    }

    // --------------------------------------------------------------- section 17

    /// <summary>Area Variance = Actual Area - Planned Area.</summary>
    public static decimal AreaVariance(decimal actualArea, decimal plannedArea) => Round(actualArea - plannedArea);

    /// <summary>Material Variance = Actual Material - Planned Material.</summary>
    public static decimal MaterialVariance(decimal actual, decimal planned) => Round(actual - planned);

    /// <summary>Fuel Variance = Actual Fuel - Planned Fuel.</summary>
    public static decimal FuelVariance(decimal actual, decimal planned) => Round(actual - planned);

    /// <summary>Schedule Variance in days = Actual Completion Date - Planned Completion Date.</summary>
    public static int? ScheduleVarianceDays(DateOnly? actualCompletion, DateOnly? plannedCompletion)
    {
        if (actualCompletion is null || plannedCompletion is null) return null;
        return actualCompletion.Value.DayNumber - plannedCompletion.Value.DayNumber;
    }

    /// <summary>Completion % = Actual Completed Area / Planned Area x 100.</summary>
    public static decimal CompletionPercentage(decimal actualCompletedArea, decimal plannedArea)
    {
        if (plannedArea <= 0) return 0m;
        return Round(actualCompletedArea / plannedArea * 100m);
    }

    // --------------------------------------------------------- unit conversion

    /// <summary>
    /// Converts a quantity expressed in a material's alternative unit into its base unit
    /// using the master-data conversion factor (base = alternative x factor).
    /// </summary>
    public static decimal ToBaseUnit(decimal quantityInAlternativeUnit, decimal conversionFactor)
    {
        if (conversionFactor <= 0)
            throw new ArgumentOutOfRangeException(nameof(conversionFactor), "Unit conversion factor must be greater than zero.");
        return Round(quantityInAlternativeUnit * conversionFactor);
    }

    /// <summary>Inverse of <see cref="ToBaseUnit"/>.</summary>
    public static decimal ToAlternativeUnit(decimal quantityInBaseUnit, decimal conversionFactor)
    {
        if (conversionFactor <= 0)
            throw new ArgumentOutOfRangeException(nameof(conversionFactor), "Unit conversion factor must be greater than zero.");
        return Round(quantityInBaseUnit / conversionFactor);
    }

    // ------------------------------------------------------------ date helpers

    /// <summary>True when two inclusive date ranges share at least one day.</summary>
    public static bool DateRangesOverlap(DateOnly startA, DateOnly endA, DateOnly startB, DateOnly endB)
        => startA <= endB && startB <= endA;

    /// <summary>True when two time windows on the same day share at least one minute.</summary>
    public static bool TimeRangesOverlap(DateTime startA, DateTime endA, DateTime startB, DateTime endB)
        => startA < endB && startB < endA;
}
