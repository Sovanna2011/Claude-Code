using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>
/// Configurable planting activity master (section 6). Activities are never hard-coded:
/// the activity-plan generator reads whatever rows are active for the crop type.
/// </summary>
public class PlantingActivity : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public ActivityCategory Category { get; set; } = ActivityCategory.LandPreparation;

    /// <summary>Order in which activities are executed within a block.</summary>
    public int SequenceNo { get; set; }

    public CropType ApplicableCropType { get; set; } = CropType.Both;

    /// <summary>Days between the projection line's planting start and this activity's start (may be negative).</summary>
    public int StandardStartDayOffset { get; set; }

    public decimal StandardCapacityPerHour { get; set; }
    public decimal StandardCapacityPerDay { get; set; }
    /// <summary>Standard hours needed to cover one hectare.</summary>
    public decimal StandardDurationPerHa { get; set; }

    public bool IsMandatory { get; set; } = true;
    public bool RequiresTractor { get; set; }
    public bool RequiresEquipment { get; set; }
    public bool RequiresMaterial { get; set; }
    public bool RequiresLabor { get; set; }
    public bool AllowOverlap { get; set; }

    /// <summary>Standard labor-days per hectare, used by the labor projection (section 14).</summary>
    public decimal StandardLaborDaysPerHa { get; set; }

    /// <summary>Equipment category normally used; drives the equipment requirement engine.</summary>
    public EquipmentCategory? DefaultEquipmentCategory { get; set; }

    public bool IsActive { get; set; } = true;

    /// <summary>Activities that must finish before this one starts.</summary>
    public ICollection<ActivityDependency> Predecessors { get; set; } = new List<ActivityDependency>();
    public ICollection<ActivityDependency> Successors { get; set; } = new List<ActivityDependency>();
    public ICollection<ActivityMaterialStandard> MaterialStandards { get; set; } = new List<ActivityMaterialStandard>();

    /// <summary>True when the activity applies to the given crop type.</summary>
    public bool AppliesTo(CropType cropType)
        => ApplicableCropType == CropType.Both || ApplicableCropType == cropType;
}

/// <summary>
/// Directed edge "successor cannot start until predecessor is complete" (section 6).
/// <see cref="LagDays"/> adds a mandatory waiting time after the predecessor ends.
/// </summary>
public class ActivityDependency : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ActivityId { get; set; }
    public PlantingActivity? Activity { get; set; }

    public int PredecessorActivityId { get; set; }
    public PlantingActivity? PredecessorActivity { get; set; }

    public int LagDays { get; set; }
    /// <summary>When false the dependency is advisory and does not block scheduling.</summary>
    public bool IsBlocking { get; set; } = true;
    public string? Remarks { get; set; }
}
