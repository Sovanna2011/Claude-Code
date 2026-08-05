using System.ComponentModel.DataAnnotations;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Materials;

public class MaterialDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public MaterialCategory Category { get; set; }
    public UnitOfMeasure BaseUnit { get; set; }
    public UnitOfMeasure? AlternativeUnit { get; set; }
    public decimal UnitConversionFactor { get; set; }
    public decimal StandardRatePerHa { get; set; }
    public decimal MinApplicationRate { get; set; }
    public decimal MaxApplicationRate { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class MaterialUpsertDto : IValidatableObject
{
    [Required] public int CompanyId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    public MaterialCategory Category { get; set; } = MaterialCategory.Other;
    public UnitOfMeasure BaseUnit { get; set; } = UnitOfMeasure.Kilogram;
    public UnitOfMeasure? AlternativeUnit { get; set; }
    [Range(0.0001, 1_000_000)] public decimal UnitConversionFactor { get; set; } = 1m;
    [Range(0, 1_000_000)] public decimal StandardRatePerHa { get; set; }
    [Range(0, 1_000_000)] public decimal MinApplicationRate { get; set; }
    [Range(0, 1_000_000)] public decimal MaxApplicationRate { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }

    public IEnumerable<ValidationResult> Validate(ValidationContext validationContext)
    {
        if (MaxApplicationRate > 0 && MinApplicationRate > MaxApplicationRate)
            yield return new ValidationResult("Minimum application rate cannot exceed the maximum.",
                new[] { nameof(MinApplicationRate) });
        if (AlternativeUnit is not null && AlternativeUnit == BaseUnit)
            yield return new ValidationResult("The alternative unit must differ from the base unit.",
                new[] { nameof(AlternativeUnit) });
    }
}

public class ActivityMaterialStandardDto
{
    public int Id { get; set; }
    public int ActivityId { get; set; }
    public string ActivityName { get; set; } = string.Empty;
    public int MaterialId { get; set; }
    public string MaterialCode { get; set; } = string.Empty;
    public string MaterialName { get; set; } = string.Empty;
    public UnitOfMeasure Unit { get; set; }
    public CropType CropType { get; set; }
    public int? CaneVarietyId { get; set; }
    public string? VarietyName { get; set; }
    public SoilType? SoilType { get; set; }
    public decimal StandardRatePerHa { get; set; }
    public int NumberOfApplications { get; set; }
    public decimal WastePercent { get; set; }
    public DateOnly EffectiveFrom { get; set; }
    public DateOnly? EffectiveTo { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class ActivityMaterialStandardUpsertDto
{
    [Required] public int ActivityId { get; set; }
    [Required] public int MaterialId { get; set; }
    public CropType CropType { get; set; } = CropType.Both;
    public int? CaneVarietyId { get; set; }
    public SoilType? SoilType { get; set; }
    [Range(0.0001, 1_000_000)] public decimal StandardRatePerHa { get; set; }
    [Range(1, 20)] public int NumberOfApplications { get; set; } = 1;
    [Range(0, 100)] public decimal WastePercent { get; set; }
    public DateOnly EffectiveFrom { get; set; }
    public DateOnly? EffectiveTo { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class MaterialStockDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    public string? EstateName { get; set; }
    public int MaterialId { get; set; }
    public string MaterialCode { get; set; } = string.Empty;
    public string MaterialName { get; set; } = string.Empty;
    public decimal AvailableStock { get; set; }
    public decimal ReservedQuantity { get; set; }
    public decimal IncomingQuantity { get; set; }
    public DateOnly? IncomingExpectedDate { get; set; }
    public decimal NetAvailableQuantity { get; set; }
    public UnitOfMeasure Unit { get; set; }
    public DateTime LastSyncedUtc { get; set; }
    public string? SourceSystem { get; set; }
}

/// <summary>Payload accepted by the stock interface endpoint (section 13).</summary>
public class MaterialStockSyncDto
{
    [Required] public string MaterialCode { get; set; } = string.Empty;
    public string? EstateCode { get; set; }
    [Range(0, 100_000_000)] public decimal AvailableStock { get; set; }
    [Range(0, 100_000_000)] public decimal ReservedQuantity { get; set; }
    [Range(0, 100_000_000)] public decimal IncomingQuantity { get; set; }
    public DateOnly? IncomingExpectedDate { get; set; }
    [StringLength(50)] public string? SourceSystem { get; set; }
}

/// <summary>One consolidated MRP row (section 13).</summary>
public class MaterialRequirementDto
{
    public int MaterialId { get; set; }
    public string MaterialCode { get; set; } = string.Empty;
    public string MaterialName { get; set; } = string.Empty;
    public MaterialCategory Category { get; set; }
    public UnitOfMeasure Unit { get; set; }

    public int? EstateId { get; set; }
    public string? EstateName { get; set; }
    public int? FarmId { get; set; }
    public string? FarmName { get; set; }
    public int? ZoneId { get; set; }
    public string? ZoneName { get; set; }
    public int? BlockId { get; set; }
    public string? BlockName { get; set; }
    public int? ActivityId { get; set; }
    public string? ActivityName { get; set; }
    public int? CaneVarietyId { get; set; }
    public string? VarietyName { get; set; }
    public int? PlanningYear { get; set; }
    public int? PlanningMonth { get; set; }

    public decimal PlannedAreaHa { get; set; }
    public decimal BaseRequirement { get; set; }
    public decimal WasteQuantity { get; set; }
    public decimal TotalRequirement { get; set; }

    public decimal AvailableStock { get; set; }
    public decimal ReservedQuantity { get; set; }
    public decimal IncomingQuantity { get; set; }
    public decimal NetAvailableQuantity { get; set; }
    public decimal ShortageQuantity { get; set; }
    public decimal SurplusQuantity { get; set; }
    public DateOnly? RequiredDeliveryDate { get; set; }
    public CapacityStatus Status { get; set; }
}

/// <summary>Filter for the MRP screen; every level of the hierarchy is optional.</summary>
public class MaterialRequirementQuery
{
    public int? ProjectionId { get; set; }
    public int? SeasonId { get; set; }
    public int? EstateId { get; set; }
    public int? FarmId { get; set; }
    public int? ZoneId { get; set; }
    public int? BlockId { get; set; }
    public int? ActivityId { get; set; }
    public int? MaterialId { get; set; }
    public int? CaneVarietyId { get; set; }
    public MaterialCategory? Category { get; set; }
    public DateOnly? FromDate { get; set; }
    public DateOnly? ToDate { get; set; }
    /// <summary>Grouping level: material | activity | block | farm | month | variety.</summary>
    public string GroupBy { get; set; } = "material";
    public bool ShortagesOnly { get; set; }

    public string ToQueryString()
    {
        var parts = new List<string>();
        void Add(string key, object? value) { if (value is not null) parts.Add($"{key}={Uri.EscapeDataString(value.ToString()!)}"); }
        Add("projectionId", ProjectionId);
        Add("seasonId", SeasonId);
        Add("estateId", EstateId);
        Add("farmId", FarmId);
        Add("zoneId", ZoneId);
        Add("blockId", BlockId);
        Add("activityId", ActivityId);
        Add("materialId", MaterialId);
        Add("caneVarietyId", CaneVarietyId);
        Add("category", Category);
        Add("fromDate", FromDate?.ToString("yyyy-MM-dd"));
        Add("toDate", ToDate?.ToString("yyyy-MM-dd"));
        Add("groupBy", GroupBy);
        if (ShortagesOnly) Add("shortagesOnly", true);
        return string.Join('&', parts);
    }
}
