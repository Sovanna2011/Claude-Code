using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>
/// One scheduled activity on one block, generated from an approved projection line (section 7).
/// This is what the Gantt view, the resource engines and the actual-progress screens work on.
/// </summary>
public class ActivityPlan : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ProjectionId { get; set; }
    public PlantingProjection? Projection { get; set; }
    public int ProjectionLineId { get; set; }
    public ProjectionLine? ProjectionLine { get; set; }

    public int FarmId { get; set; }
    public int ZoneId { get; set; }
    public int BlockId { get; set; }
    public PlantationBlock? Block { get; set; }

    public int ActivityId { get; set; }
    public PlantingActivity? Activity { get; set; }
    public int SequenceNo { get; set; }

    public decimal PlannedAreaHa { get; set; }
    public DateOnly PlannedStartDate { get; set; }
    public DateOnly PlannedEndDate { get; set; }
    public int WorkingDays { get; set; }
    public decimal DailyTargetHa { get; set; }

    /// <summary>Planned machine hours = planned area x standard duration per hectare.</summary>
    public decimal PlannedWorkingHours { get; set; }

    public string? RequiredTractorType { get; set; }
    public EquipmentCategory? RequiredEquipmentCategory { get; set; }
    public int RequiredTractorCount { get; set; }
    public int RequiredEquipmentCount { get; set; }
    public decimal RequiredLaborDays { get; set; }
    public int RequiredWorkers { get; set; }
    public decimal PlannedFuelLiters { get; set; }

    public string? SupervisorName { get; set; }
    public ActivityStatus Status { get; set; } = ActivityStatus.Planned;
    public string? Remarks { get; set; }

    public ICollection<ResourceSchedule> Schedules { get; set; } = new List<ResourceSchedule>();
    public ICollection<ActivityMaterialRequirement> MaterialRequirements { get; set; } = new List<ActivityMaterialRequirement>();
    public ICollection<ActivityActual> Actuals { get; set; } = new List<ActivityActual>();

    /// <summary>Recomputes working days, daily target, hours, labor and fuel from the current dates and area.</summary>
    public void Recalculate(PlantingActivity activity, decimal? litersPerHectare = null,
        bool workOnSaturday = true, bool workOnSunday = false)
    {
        WorkingDays = Math.Max(1, PlanningFormulas.WorkingDays(PlannedStartDate, PlannedEndDate, workOnSaturday, workOnSunday));
        DailyTargetHa = PlanningFormulas.DailyTarget(PlannedAreaHa, WorkingDays);
        PlannedWorkingHours = Math.Round(PlannedAreaHa * activity.StandardDurationPerHa, PlanningFormulas.QuantityScale);
        RequiredLaborDays = PlanningFormulas.RequiredLaborDays(PlannedAreaHa, activity.StandardLaborDaysPerHa);
        RequiredWorkers = activity.RequiresLabor ? PlanningFormulas.RequiredWorkers(RequiredLaborDays, WorkingDays) : 0;
        if (activity.RequiresTractor && activity.StandardCapacityPerDay > 0)
            RequiredTractorCount = PlanningFormulas.RequiredTractors(PlannedAreaHa, activity.StandardCapacityPerDay, WorkingDays);
        if (activity.RequiresEquipment && activity.StandardCapacityPerDay > 0)
            RequiredEquipmentCount = PlanningFormulas.RequiredEquipment(PlannedAreaHa, activity.StandardCapacityPerDay, WorkingDays);
        if (litersPerHectare is > 0)
            PlannedFuelLiters = PlanningFormulas.ProjectedFuelByArea(PlannedAreaHa, litersPerHectare.Value);
    }
}
