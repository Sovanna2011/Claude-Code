using System.ComponentModel.DataAnnotations;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Seasons;

public class GrowingSeasonDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public DateOnly StartDate { get; set; }
    public DateOnly EndDate { get; set; }
    public DateOnly PlannedPlantingStart { get; set; }
    public DateOnly PlannedPlantingEnd { get; set; }
    public DateOnly ExpectedHarvestStart { get; set; }
    public DateOnly ExpectedHarvestEnd { get; set; }
    public SeasonStatus Status { get; set; }
    public string? Remarks { get; set; }
    public int ProjectionCount { get; set; }
    public byte[]? RowVersion { get; set; }
}

public class GrowingSeasonUpsertDto : IValidatableObject
{
    [Required] public int CompanyId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    public DateOnly StartDate { get; set; }
    public DateOnly EndDate { get; set; }
    public DateOnly PlannedPlantingStart { get; set; }
    public DateOnly PlannedPlantingEnd { get; set; }
    public DateOnly ExpectedHarvestStart { get; set; }
    public DateOnly ExpectedHarvestEnd { get; set; }
    public SeasonStatus Status { get; set; } = SeasonStatus.Draft;
    [StringLength(1000)] public string? Remarks { get; set; }
    public byte[]? RowVersion { get; set; }

    public IEnumerable<ValidationResult> Validate(ValidationContext validationContext)
    {
        if (EndDate < StartDate)
            yield return new ValidationResult("Season end date must not be before the start date.", new[] { nameof(EndDate) });
        if (PlannedPlantingEnd < PlannedPlantingStart)
            yield return new ValidationResult("Planting end date must not be before the planting start date.", new[] { nameof(PlannedPlantingEnd) });
        if (PlannedPlantingStart < StartDate || PlannedPlantingEnd > EndDate)
            yield return new ValidationResult("The planting window must sit inside the season.", new[] { nameof(PlannedPlantingStart) });
        if (ExpectedHarvestEnd < ExpectedHarvestStart)
            yield return new ValidationResult("Harvest end date must not be before the harvest start date.", new[] { nameof(ExpectedHarvestEnd) });
    }
}

public class CaneVarietyDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public SoilType RecommendedSoilType { get; set; }
    public decimal SeedRatePerHa { get; set; }
    public int GrowingPeriodMonths { get; set; }
    public decimal ExpectedYieldPerHa { get; set; }
    public decimal ExpectedLossPercent { get; set; }
    public int RecommendedPlantingStartMonth { get; set; }
    public int RecommendedPlantingEndMonth { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class CaneVarietyUpsertDto
{
    [Required] public int CompanyId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    public SoilType RecommendedSoilType { get; set; } = SoilType.Unspecified;
    [Range(0.01, 1000)] public decimal SeedRatePerHa { get; set; } = 8m;
    [Range(1, 36)] public int GrowingPeriodMonths { get; set; } = 12;
    [Range(0, 1000)] public decimal ExpectedYieldPerHa { get; set; } = 80m;
    [Range(0, 100)] public decimal ExpectedLossPercent { get; set; }
    [Range(1, 12)] public int RecommendedPlantingStartMonth { get; set; } = 1;
    [Range(1, 12)] public int RecommendedPlantingEndMonth { get; set; } = 12;
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}
