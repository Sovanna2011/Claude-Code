using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>Planning-side material master (section 11). No purchasing or warehousing here.</summary>
public class Material : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public MaterialCategory Category { get; set; } = MaterialCategory.Other;

    public UnitOfMeasure BaseUnit { get; set; } = UnitOfMeasure.Kilogram;
    public UnitOfMeasure? AlternativeUnit { get; set; }
    /// <summary>base quantity = alternative quantity x factor (e.g. 1 Bag = 50 kg → 50).</summary>
    public decimal UnitConversionFactor { get; set; } = 1m;

    public decimal StandardRatePerHa { get; set; }
    public decimal MinApplicationRate { get; set; }
    public decimal MaxApplicationRate { get; set; }

    public bool IsActive { get; set; } = true;

    public ICollection<ActivityMaterialStandard> Standards { get; set; } = new List<ActivityMaterialStandard>();
    public ICollection<MaterialStock> Stocks { get; set; } = new List<MaterialStock>();

    /// <summary>Guards the min/max band from the master data (section 11).</summary>
    public bool IsRateWithinBand(decimal rate)
        => (MinApplicationRate <= 0 || rate >= MinApplicationRate)
        && (MaxApplicationRate <= 0 || rate <= MaxApplicationRate);
}

/// <summary>
/// Consumption standard per activity, refined by crop type / variety / soil type (section 12).
/// The most specific effective row wins.
/// </summary>
public class ActivityMaterialStandard : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ActivityId { get; set; }
    public PlantingActivity? Activity { get; set; }

    public int MaterialId { get; set; }
    public Material? Material { get; set; }

    public CropType CropType { get; set; } = CropType.Both;
    public int? CaneVarietyId { get; set; }
    public CaneVariety? CaneVariety { get; set; }
    public SoilType? SoilType { get; set; }

    public decimal StandardRatePerHa { get; set; }
    public int NumberOfApplications { get; set; } = 1;
    public decimal WastePercent { get; set; }

    public DateOnly EffectiveFrom { get; set; }
    public DateOnly? EffectiveTo { get; set; }
    public bool IsActive { get; set; } = true;

    /// <summary>Higher score = more specific match; used to pick one standard among candidates.</summary>
    public int SpecificityScore()
    {
        var score = 0;
        if (CropType != CropType.Both) score += 4;
        if (CaneVarietyId is not null) score += 2;
        if (SoilType is not null) score += 1;
        return score;
    }

    /// <summary>True when the row is effective on the given date.</summary>
    public bool IsEffectiveOn(DateOnly date)
        => IsActive && EffectiveFrom <= date && (EffectiveTo is null || EffectiveTo >= date);
}

/// <summary>
/// Calculated material requirement for one activity plan and one material (section 13).
/// Rows are regenerated whenever the plan changes.
/// </summary>
public class ActivityMaterialRequirement : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ActivityPlanId { get; set; }
    public ActivityPlan? ActivityPlan { get; set; }

    public int MaterialId { get; set; }
    public Material? Material { get; set; }

    public int? ActivityMaterialStandardId { get; set; }

    public decimal PlannedAreaHa { get; set; }
    public decimal StandardRatePerHa { get; set; }
    public int NumberOfApplications { get; set; } = 1;
    public decimal WastePercent { get; set; }

    public decimal BaseRequirement { get; set; }
    public decimal WasteQuantity { get; set; }
    public decimal TotalRequirement { get; set; }
    public UnitOfMeasure Unit { get; set; }

    public DateOnly RequiredDeliveryDate { get; set; }
}

/// <summary>
/// Stock position fed in from the ERP interface (section 13). The planning system only reads it.
/// </summary>
public class MaterialStock : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    public Estate? Estate { get; set; }

    public int MaterialId { get; set; }
    public Material? Material { get; set; }

    public decimal AvailableStock { get; set; }
    public decimal ReservedQuantity { get; set; }
    public decimal IncomingQuantity { get; set; }
    public DateOnly? IncomingExpectedDate { get; set; }

    public UnitOfMeasure Unit { get; set; }
    public DateTime LastSyncedUtc { get; set; }
    public string? SourceSystem { get; set; }
}
