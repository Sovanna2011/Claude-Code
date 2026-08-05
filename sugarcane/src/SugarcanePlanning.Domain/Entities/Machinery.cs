using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>Tractor master (section 8).</summary>
public class Tractor : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    public Estate? Estate { get; set; }

    public string AssetNo { get; set; } = string.Empty;
    public string Code { get; set; } = string.Empty;
    public string? RegistrationNo { get; set; }

    public string? Brand { get; set; }
    public string? Model { get; set; }
    public int Horsepower { get; set; }

    /// <summary>Farm the machine is currently parked at; used by the location-conflict check.</summary>
    public int? CurrentFarmId { get; set; }
    public Farm? CurrentFarm { get; set; }

    public OwnershipType Ownership { get; set; } = OwnershipType.Company;

    /// <summary>Hectares this tractor can cover in one working day.</summary>
    public decimal DailyCapacityHa { get; set; }
    public decimal FuelConsumptionPerHour { get; set; }
    public decimal FuelConsumptionPerHa { get; set; }

    public AvailabilityStatus Availability { get; set; } = AvailabilityStatus.Available;
    public MaintenanceStatus Maintenance { get; set; } = MaintenanceStatus.Serviceable;
    public DateOnly? MaintenanceFromDate { get; set; }
    public DateOnly? MaintenanceToDate { get; set; }

    public bool IsActive { get; set; } = true;

    public ICollection<ResourceSchedule> Schedules { get; set; } = new List<ResourceSchedule>();

    /// <summary>A machine can be booked only when it is available and not in a maintenance window.</summary>
    public bool IsBookableOn(DateOnly date)
    {
        if (!IsActive) return false;
        if (Availability is AvailabilityStatus.UnderMaintenance or AvailabilityStatus.Breakdown or AvailabilityStatus.Inactive)
            return false;
        if (MaintenanceFromDate is not null && MaintenanceToDate is not null)
            return !(date >= MaintenanceFromDate.Value && date <= MaintenanceToDate.Value);
        return true;
    }
}

/// <summary>Implement master — plows, harrows, planters, sprayers, trailers … (section 9).</summary>
public class EquipmentItem : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    public Estate? Estate { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public EquipmentCategory Category { get; set; } = EquipmentCategory.Other;

    /// <summary>Minimum tractor horsepower needed to pull this implement.</summary>
    public int MinimumTractorHp { get; set; }

    public decimal CapacityPerHour { get; set; }
    public decimal CapacityPerDay { get; set; }

    public int? CurrentFarmId { get; set; }
    public Farm? CurrentFarm { get; set; }

    public OwnershipType Ownership { get; set; } = OwnershipType.Company;
    public AvailabilityStatus Availability { get; set; } = AvailabilityStatus.Available;
    public MaintenanceStatus Maintenance { get; set; } = MaintenanceStatus.Serviceable;
    public DateOnly? MaintenanceFromDate { get; set; }
    public DateOnly? MaintenanceToDate { get; set; }

    public bool IsActive { get; set; } = true;

    public ICollection<TractorEquipmentCompatibility> Compatibilities { get; set; } = new List<TractorEquipmentCompatibility>();
    public ICollection<ResourceSchedule> Schedules { get; set; } = new List<ResourceSchedule>();

    /// <inheritdoc cref="Tractor.IsBookableOn"/>
    public bool IsBookableOn(DateOnly date)
    {
        if (!IsActive) return false;
        if (Availability is AvailabilityStatus.UnderMaintenance or AvailabilityStatus.Breakdown or AvailabilityStatus.Inactive)
            return false;
        if (MaintenanceFromDate is not null && MaintenanceToDate is not null)
            return !(date >= MaintenanceFromDate.Value && date <= MaintenanceToDate.Value);
        return true;
    }
}

/// <summary>
/// Explicit tractor/implement pairing rule (section 9). When rows exist for an implement the
/// pairing must be listed <em>and</em> the horsepower check must pass.
/// </summary>
public class TractorEquipmentCompatibility : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int TractorId { get; set; }
    public Tractor? Tractor { get; set; }

    public int EquipmentId { get; set; }
    public EquipmentItem? Equipment { get; set; }

    public bool IsRecommended { get; set; } = true;
    public string? Remarks { get; set; }
}
