using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Labor;

/// <summary>Fuel projection row; the query decides the grouping (section 14).</summary>
public class FuelProjectionDto
{
    /// <summary>activity | tractor | farm | day | week | month.</summary>
    public string GroupBy { get; set; } = "activity";
    public string GroupKey { get; set; } = string.Empty;
    public string GroupName { get; set; } = string.Empty;
    public int? EntityId { get; set; }
    public DateOnly? PeriodStart { get; set; }
    public DateOnly? PeriodEnd { get; set; }

    public decimal PlannedAreaHa { get; set; }
    public decimal PlannedWorkingHours { get; set; }
    public decimal FuelByAreaLiters { get; set; }
    public decimal FuelByHourLiters { get; set; }
    /// <summary>Figure used for procurement: the larger of the two projections.</summary>
    public decimal ProjectedFuelLiters { get; set; }
}

/// <summary>Labor projection row (section 14).</summary>
public class LaborProjectionDto
{
    public string GroupBy { get; set; } = "activity";
    public string GroupKey { get; set; } = string.Empty;
    public string GroupName { get; set; } = string.Empty;
    public int? EntityId { get; set; }
    public DateOnly? PeriodStart { get; set; }
    public DateOnly? PeriodEnd { get; set; }

    public SkillType? Skill { get; set; }
    public decimal PlannedAreaHa { get; set; }
    public decimal StandardLaborDaysPerHa { get; set; }
    public decimal RequiredLaborDays { get; set; }
    public int WorkingDays { get; set; }
    public int RequiredWorkers { get; set; }
    public int AvailableWorkers { get; set; }
    public int WorkerGap => AvailableWorkers - RequiredWorkers;
    public CapacityStatus Status { get; set; }
    public string? SupervisorName { get; set; }
}

public class FuelLaborQuery
{
    public int? ProjectionId { get; set; }
    public int? SeasonId { get; set; }
    public int? EstateId { get; set; }
    public int? FarmId { get; set; }
    public int? ActivityId { get; set; }
    public DateOnly? FromDate { get; set; }
    public DateOnly? ToDate { get; set; }
    /// <summary>activity | tractor | farm | day | week | month.</summary>
    public string GroupBy { get; set; } = "activity";

    public string ToQueryString()
    {
        var parts = new List<string>();
        void Add(string k, object? v) { if (v is not null) parts.Add($"{k}={Uri.EscapeDataString(v.ToString()!)}"); }
        Add("projectionId", ProjectionId);
        Add("seasonId", SeasonId);
        Add("estateId", EstateId);
        Add("farmId", FarmId);
        Add("activityId", ActivityId);
        Add("fromDate", FromDate?.ToString("yyyy-MM-dd"));
        Add("toDate", ToDate?.ToString("yyyy-MM-dd"));
        Add("groupBy", GroupBy);
        return string.Join('&', parts);
    }
}
