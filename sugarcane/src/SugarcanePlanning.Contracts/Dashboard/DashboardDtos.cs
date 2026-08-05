using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Dashboard;

/// <summary>Everything the dashboard screen renders (section 18).</summary>
public class DashboardDto
{
    public int? SeasonId { get; set; }
    public string SeasonName { get; set; } = string.Empty;
    public DateOnly GeneratedOn { get; set; }

    // KPI cards
    public decimal TotalProjectedAreaHa { get; set; }
    public decimal NewPlantingAreaHa { get; set; }
    public decimal RatoonAreaHa { get; set; }
    public decimal TotalExpectedProductionTons { get; set; }
    public decimal PlannedAreaToDateHa { get; set; }
    public decimal ActualPlantedAreaHa { get; set; }
    public decimal OverallCompletionPercent { get; set; }
    public int DelayedActivityCount { get; set; }
    public int AtRiskBlockCount { get; set; }
    public int MaterialShortageCount { get; set; }

    public List<AreaBreakdownDto> AreaByFarm { get; set; } = new();
    public List<AreaBreakdownDto> AreaByZone { get; set; } = new();
    public List<AreaBreakdownDto> AreaByBlock { get; set; } = new();
    public List<PeriodTargetDto> MonthlyTargets { get; set; } = new();
    public List<PeriodTargetDto> WeeklyTargets { get; set; } = new();
    public List<ResourceGapDto> ResourceGaps { get; set; } = new();
    public List<MaterialSummaryDto> MaterialSummary { get; set; } = new();
    public List<DelayedActivityDto> DelayedActivities { get; set; } = new();
    public List<AtRiskBlockDto> AtRiskBlocks { get; set; } = new();
}

public class AreaBreakdownDto
{
    public int Id { get; set; }
    public string Name { get; set; } = string.Empty;
    public decimal PlannedAreaHa { get; set; }
    public decimal ActualAreaHa { get; set; }
    public decimal NewPlantingAreaHa { get; set; }
    public decimal RatoonAreaHa { get; set; }
    public decimal CompletionPercent { get; set; }
}

public class PeriodTargetDto
{
    public int Year { get; set; }
    public int Period { get; set; }              // month 1-12 or ISO week
    public string Label { get; set; } = string.Empty;
    public DateOnly PeriodStart { get; set; }
    public DateOnly PeriodEnd { get; set; }
    public decimal TargetAreaHa { get; set; }
    public decimal ActualAreaHa { get; set; }
    public decimal CompletionPercent { get; set; }
}

public class ResourceGapDto
{
    public string ResourceType { get; set; } = string.Empty;   // Tractor | Equipment | Labor
    public string Name { get; set; } = string.Empty;
    public decimal Required { get; set; }
    public decimal Available { get; set; }
    public decimal Shortage { get; set; }
    public CapacityStatus Status { get; set; }
}

public class MaterialSummaryDto
{
    public MaterialCategory Category { get; set; }
    public string CategoryName { get; set; } = string.Empty;
    public string Unit { get; set; } = string.Empty;
    public decimal TotalRequirement { get; set; }
    public decimal NetAvailable { get; set; }
    public decimal Shortage { get; set; }
    public CapacityStatus Status { get; set; }
}

public class DelayedActivityDto
{
    public int ActivityPlanId { get; set; }
    public string BlockName { get; set; } = string.Empty;
    public string FarmName { get; set; } = string.Empty;
    public string ActivityName { get; set; } = string.Empty;
    public DateOnly PlannedEndDate { get; set; }
    public int DaysLate { get; set; }
    public decimal PlannedAreaHa { get; set; }
    public decimal CompletionPercent { get; set; }
    public string? DelayReason { get; set; }
}

public class AtRiskBlockDto
{
    public int BlockId { get; set; }
    public string BlockCode { get; set; } = string.Empty;
    public string BlockName { get; set; } = string.Empty;
    public string FarmName { get; set; } = string.Empty;
    public decimal PlannedAreaHa { get; set; }
    public decimal CompletionPercent { get; set; }
    public string RiskReason { get; set; } = string.Empty;
    public CapacityStatus Status { get; set; }
}
