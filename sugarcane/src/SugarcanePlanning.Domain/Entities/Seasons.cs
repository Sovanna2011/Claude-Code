using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>A planting/harvest campaign. All projections hang off a season (section 4).</summary>
public class GrowingSeason : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public Company? Company { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;

    public DateOnly StartDate { get; set; }
    public DateOnly EndDate { get; set; }
    public DateOnly PlannedPlantingStart { get; set; }
    public DateOnly PlannedPlantingEnd { get; set; }
    public DateOnly ExpectedHarvestStart { get; set; }
    public DateOnly ExpectedHarvestEnd { get; set; }

    public SeasonStatus Status { get; set; } = SeasonStatus.Draft;
    public string? Remarks { get; set; }

    /// <summary>True when the given planting window sits inside the season's planting period.</summary>
    public bool ContainsPlantingWindow(DateOnly start, DateOnly end)
        => start >= PlannedPlantingStart && end <= PlannedPlantingEnd;
}

/// <summary>Sugarcane variety master with the agronomic standards used by the engines (section 4).</summary>
public class CaneVariety : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;

    public SoilType RecommendedSoilType { get; set; } = SoilType.Unspecified;
    /// <summary>Standard seed rate in tons of seed cane per hectare.</summary>
    public decimal SeedRatePerHa { get; set; }
    public int GrowingPeriodMonths { get; set; } = 12;
    /// <summary>Expected cane yield in tons per hectare.</summary>
    public decimal ExpectedYieldPerHa { get; set; }
    public decimal ExpectedLossPercent { get; set; }

    /// <summary>Recommended planting window expressed as month numbers (1-12).</summary>
    public int RecommendedPlantingStartMonth { get; set; } = 1;
    public int RecommendedPlantingEndMonth { get; set; } = 12;

    public bool IsActive { get; set; } = true;

    /// <summary>True when the month falls inside the recommended window (wrapping over year end).</summary>
    public bool IsRecommendedForMonth(int month)
    {
        if (RecommendedPlantingStartMonth <= RecommendedPlantingEndMonth)
            return month >= RecommendedPlantingStartMonth && month <= RecommendedPlantingEndMonth;
        return month >= RecommendedPlantingStartMonth || month <= RecommendedPlantingEndMonth;
    }
}
