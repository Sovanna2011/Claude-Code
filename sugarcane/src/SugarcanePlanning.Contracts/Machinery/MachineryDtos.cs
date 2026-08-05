using System.ComponentModel.DataAnnotations;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Machinery;

public class TractorDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    public string? EstateName { get; set; }
    public string AssetNo { get; set; } = string.Empty;
    public string Code { get; set; } = string.Empty;
    public string? RegistrationNo { get; set; }
    public string? Brand { get; set; }
    public string? Model { get; set; }
    public int Horsepower { get; set; }
    public int? CurrentFarmId { get; set; }
    public string? CurrentFarmName { get; set; }
    public OwnershipType Ownership { get; set; }
    public decimal DailyCapacityHa { get; set; }
    public decimal FuelConsumptionPerHour { get; set; }
    public decimal FuelConsumptionPerHa { get; set; }
    public AvailabilityStatus Availability { get; set; }
    public MaintenanceStatus Maintenance { get; set; }
    public DateOnly? MaintenanceFromDate { get; set; }
    public DateOnly? MaintenanceToDate { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class TractorUpsertDto
{
    [Required] public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    [Required, StringLength(30)] public string AssetNo { get; set; } = string.Empty;
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [StringLength(30)] public string? RegistrationNo { get; set; }
    [StringLength(60)] public string? Brand { get; set; }
    [StringLength(60)] public string? Model { get; set; }
    [Range(1, 2000)] public int Horsepower { get; set; }
    public int? CurrentFarmId { get; set; }
    public OwnershipType Ownership { get; set; } = OwnershipType.Company;
    [Range(0, 1000)] public decimal DailyCapacityHa { get; set; }
    [Range(0, 1000)] public decimal FuelConsumptionPerHour { get; set; }
    [Range(0, 1000)] public decimal FuelConsumptionPerHa { get; set; }
    public AvailabilityStatus Availability { get; set; } = AvailabilityStatus.Available;
    public MaintenanceStatus Maintenance { get; set; } = MaintenanceStatus.Serviceable;
    public DateOnly? MaintenanceFromDate { get; set; }
    public DateOnly? MaintenanceToDate { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class EquipmentDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    public string? EstateName { get; set; }
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public EquipmentCategory Category { get; set; }
    public int MinimumTractorHp { get; set; }
    public decimal CapacityPerHour { get; set; }
    public decimal CapacityPerDay { get; set; }
    public int? CurrentFarmId { get; set; }
    public string? CurrentFarmName { get; set; }
    public OwnershipType Ownership { get; set; }
    public AvailabilityStatus Availability { get; set; }
    public MaintenanceStatus Maintenance { get; set; }
    public DateOnly? MaintenanceFromDate { get; set; }
    public DateOnly? MaintenanceToDate { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class EquipmentUpsertDto
{
    [Required] public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    public EquipmentCategory Category { get; set; } = EquipmentCategory.Other;
    [Range(0, 2000)] public int MinimumTractorHp { get; set; }
    [Range(0, 1000)] public decimal CapacityPerHour { get; set; }
    [Range(0, 1000)] public decimal CapacityPerDay { get; set; }
    public int? CurrentFarmId { get; set; }
    public OwnershipType Ownership { get; set; } = OwnershipType.Company;
    public AvailabilityStatus Availability { get; set; } = AvailabilityStatus.Available;
    public MaintenanceStatus Maintenance { get; set; } = MaintenanceStatus.Serviceable;
    public DateOnly? MaintenanceFromDate { get; set; }
    public DateOnly? MaintenanceToDate { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class CompatibilityDto
{
    public int Id { get; set; }
    public int TractorId { get; set; }
    public string TractorCode { get; set; } = string.Empty;
    public int TractorHorsepower { get; set; }
    public int EquipmentId { get; set; }
    public string EquipmentCode { get; set; } = string.Empty;
    public string EquipmentName { get; set; } = string.Empty;
    public int MinimumTractorHp { get; set; }
    public bool IsRecommended { get; set; }
    public bool HorsepowerSatisfied => TractorHorsepower >= MinimumTractorHp;
    public string? Remarks { get; set; }
}

public class CompatibilityUpsertDto
{
    [Required] public int TractorId { get; set; }
    [Required] public int EquipmentId { get; set; }
    public bool IsRecommended { get; set; } = true;
    [StringLength(500)] public string? Remarks { get; set; }
}

// ---------------------------------------------------------------- operators

public class OperatorDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public int? FarmId { get; set; }
    public string? FarmName { get; set; }
    public string Code { get; set; } = string.Empty;
    public string FullName { get; set; } = string.Empty;
    public SkillType Skill { get; set; }
    public string? LicenseNo { get; set; }
    public DateOnly? LicenseExpiry { get; set; }
    public int? WorkTeamId { get; set; }
    public string? WorkTeamName { get; set; }
    public decimal StandardHoursPerDay { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class OperatorUpsertDto
{
    [Required] public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    public int? FarmId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string FullName { get; set; } = string.Empty;
    public SkillType Skill { get; set; } = SkillType.GeneralLabor;
    [StringLength(40)] public string? LicenseNo { get; set; }
    public DateOnly? LicenseExpiry { get; set; }
    public int? WorkTeamId { get; set; }
    [Range(1, 24)] public decimal StandardHoursPerDay { get; set; } = 8m;
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class WorkTeamDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public int? FarmId { get; set; }
    public string? FarmName { get; set; }
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? SupervisorName { get; set; }
    public SkillType PrimarySkill { get; set; }
    public int MemberCount { get; set; }
    public decimal StandardHoursPerDay { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class WorkTeamUpsertDto
{
    [Required] public int CompanyId { get; set; }
    public int? FarmId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    [StringLength(150)] public string? SupervisorName { get; set; }
    public SkillType PrimarySkill { get; set; } = SkillType.GeneralLabor;
    [Range(0, 10000)] public int MemberCount { get; set; }
    [Range(1, 24)] public decimal StandardHoursPerDay { get; set; } = 8m;
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}
