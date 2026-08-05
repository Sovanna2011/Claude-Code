using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>Top of the enterprise hierarchy; the tenancy boundary (section 3).</summary>
public class Company : BaseEntity
{
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? Address { get; set; }
    public string BaseCurrency { get; set; } = "USD";
    public bool IsActive { get; set; } = true;

    public ICollection<Estate> Estates { get; set; } = new List<Estate>();
    public ICollection<GrowingSeason> Seasons { get; set; } = new List<GrowingSeason>();
}

/// <summary>Company → <b>Plantation Estate</b> → Farm → Zone → Block.</summary>
public class Estate : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public Company? Company { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? Location { get; set; }
    public decimal TotalAreaHa { get; set; }
    public bool IsActive { get; set; } = true;

    public ICollection<Farm> Farms { get; set; } = new List<Farm>();
}

/// <summary>Estate → <b>Farm</b> → Zone → Block.</summary>
public class Farm : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public int EstateId { get; set; }
    public Estate? Estate { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? ManagerName { get; set; }
    public decimal TotalAreaHa { get; set; }
    public bool IsActive { get; set; } = true;

    public ICollection<Zone> Zones { get; set; } = new List<Zone>();
}

/// <summary>Farm → <b>Zone</b> → Block.</summary>
public class Zone : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public int FarmId { get; set; }
    public Farm? Farm { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? SupervisorName { get; set; }
    public decimal TotalAreaHa { get; set; }
    public bool IsActive { get; set; } = true;

    public ICollection<PlantationBlock> Blocks { get; set; } = new List<PlantationBlock>();
}

/// <summary>The smallest planning unit — a field that gets planted (section 3).</summary>
public class PlantationBlock : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public int ZoneId { get; set; }
    public Zone? Zone { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;

    public decimal TotalAreaHa { get; set; }
    /// <summary>Area that can actually be planted; caps every projection line on this block.</summary>
    public decimal PlantableAreaHa { get; set; }

    public decimal? Latitude { get; set; }
    public decimal? Longitude { get; set; }
    public string? MapReference { get; set; }

    public SoilType SoilType { get; set; } = SoilType.Unspecified;
    public LandCondition LandCondition { get; set; } = LandCondition.Flat;
    public IrrigationType Irrigation { get; set; } = IrrigationType.None;
    public bool HasIrrigation => Irrigation != IrrigationType.None && Irrigation != IrrigationType.RainFed;

    public CropStatus CurrentCropStatus { get; set; } = CropStatus.Fallow;
    public bool IsActive { get; set; } = true;
}
