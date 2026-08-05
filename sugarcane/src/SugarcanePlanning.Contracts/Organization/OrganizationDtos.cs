using System.ComponentModel.DataAnnotations;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Organization;

// ------------------------------------------------------------------ Company

public class CompanyDto
{
    public int Id { get; set; }
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? Address { get; set; }
    public string BaseCurrency { get; set; } = "USD";
    public bool IsActive { get; set; } = true;
    public int EstateCount { get; set; }
    public byte[]? RowVersion { get; set; }
}

public class CompanyUpsertDto
{
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    [StringLength(400)] public string? Address { get; set; }
    [Required, StringLength(3, MinimumLength = 3)] public string BaseCurrency { get; set; } = "USD";
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

// ------------------------------------------------------------------- Estate

public class EstateDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public string CompanyName { get; set; } = string.Empty;
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? Location { get; set; }
    public decimal TotalAreaHa { get; set; }
    public bool IsActive { get; set; } = true;
    public int FarmCount { get; set; }
    public byte[]? RowVersion { get; set; }
}

public class EstateUpsertDto
{
    [Required] public int CompanyId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    [StringLength(200)] public string? Location { get; set; }
    [Range(0, 9_999_999)] public decimal TotalAreaHa { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

// --------------------------------------------------------------------- Farm

public class FarmDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public int EstateId { get; set; }
    public string EstateName { get; set; } = string.Empty;
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? ManagerName { get; set; }
    public decimal TotalAreaHa { get; set; }
    public bool IsActive { get; set; } = true;
    public int ZoneCount { get; set; }
    public byte[]? RowVersion { get; set; }
}

public class FarmUpsertDto
{
    [Required] public int EstateId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    [StringLength(150)] public string? ManagerName { get; set; }
    [Range(0, 9_999_999)] public decimal TotalAreaHa { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

// --------------------------------------------------------------------- Zone

public class ZoneDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public int FarmId { get; set; }
    public string FarmName { get; set; } = string.Empty;
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? SupervisorName { get; set; }
    public decimal TotalAreaHa { get; set; }
    public bool IsActive { get; set; } = true;
    public int BlockCount { get; set; }
    public byte[]? RowVersion { get; set; }
}

public class ZoneUpsertDto
{
    [Required] public int FarmId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    [StringLength(150)] public string? SupervisorName { get; set; }
    [Range(0, 9_999_999)] public decimal TotalAreaHa { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

// ------------------------------------------------------------------- Blocks

public class PlantationBlockDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public int ZoneId { get; set; }
    public string ZoneName { get; set; } = string.Empty;
    public string FarmName { get; set; } = string.Empty;
    public string EstateName { get; set; } = string.Empty;
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public decimal TotalAreaHa { get; set; }
    public decimal PlantableAreaHa { get; set; }
    public decimal? Latitude { get; set; }
    public decimal? Longitude { get; set; }
    public string? MapReference { get; set; }
    public SoilType SoilType { get; set; }
    public LandCondition LandCondition { get; set; }
    public IrrigationType Irrigation { get; set; }
    public CropStatus CurrentCropStatus { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class PlantationBlockUpsertDto
{
    [Required] public int ZoneId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    [Range(0, 999_999)] public decimal TotalAreaHa { get; set; }
    [Range(0, 999_999)] public decimal PlantableAreaHa { get; set; }
    [Range(-90, 90)] public decimal? Latitude { get; set; }
    [Range(-180, 180)] public decimal? Longitude { get; set; }
    [StringLength(100)] public string? MapReference { get; set; }
    public SoilType SoilType { get; set; } = SoilType.Unspecified;
    public LandCondition LandCondition { get; set; } = LandCondition.Flat;
    public IrrigationType Irrigation { get; set; } = IrrigationType.None;
    public CropStatus CurrentCropStatus { get; set; } = CropStatus.Fallow;
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

/// <summary>Company → estate → farm → zone → block, for the land-structure tree screen.</summary>
public class LandStructureNodeDto
{
    public string NodeType { get; set; } = string.Empty;   // Company | Estate | Farm | Zone | Block
    public int Id { get; set; }
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public decimal TotalAreaHa { get; set; }
    public decimal PlantableAreaHa { get; set; }
    public bool IsActive { get; set; }
    public List<LandStructureNodeDto> Children { get; set; } = new();
}
