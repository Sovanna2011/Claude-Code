using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Common;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>
/// Actual field execution recorded against an activity plan; the basis of every
/// projection-versus-actual figure (section 17).
/// </summary>
public class ActivityActual : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ActivityPlanId { get; set; }
    public ActivityPlan? ActivityPlan { get; set; }

    public DateOnly? ActualStartDate { get; set; }
    public DateOnly? ActualCompletionDate { get; set; }
    public decimal ActualCompletedAreaHa { get; set; }

    public int? ActualTractorId { get; set; }
    public Tractor? ActualTractor { get; set; }
    public int? ActualEquipmentId { get; set; }
    public EquipmentItem? ActualEquipment { get; set; }
    public int? ActualOperatorId { get; set; }
    public Operator? ActualOperator { get; set; }

    public decimal ActualWorkingHours { get; set; }
    public decimal ActualFuelLiters { get; set; }
    public decimal ActualLaborDays { get; set; }

    // Derived variances (section 17), persisted so reports do not recompute them.
    public decimal AreaVariance { get; set; }
    public decimal FuelVariance { get; set; }
    public int? ScheduleVarianceDays { get; set; }
    public decimal CompletionPercent { get; set; }

    public string? DelayReason { get; set; }
    public string? Remarks { get; set; }

    public ICollection<ActualMaterialUsage> MaterialUsages { get; set; } = new List<ActualMaterialUsage>();

    /// <summary>Recomputes the variance columns from the linked plan.</summary>
    public void Recalculate(ActivityPlan plan)
    {
        AreaVariance = PlanningFormulas.AreaVariance(ActualCompletedAreaHa, plan.PlannedAreaHa);
        FuelVariance = PlanningFormulas.FuelVariance(ActualFuelLiters, plan.PlannedFuelLiters);
        ScheduleVarianceDays = PlanningFormulas.ScheduleVarianceDays(ActualCompletionDate, plan.PlannedEndDate);
        CompletionPercent = PlanningFormulas.CompletionPercentage(ActualCompletedAreaHa, plan.PlannedAreaHa);
    }
}

/// <summary>Material actually consumed, compared against the planned requirement (section 17).</summary>
public class ActualMaterialUsage : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ActivityActualId { get; set; }
    public ActivityActual? ActivityActual { get; set; }

    public int MaterialId { get; set; }
    public Material? Material { get; set; }

    public decimal PlannedQuantity { get; set; }
    public decimal ActualQuantity { get; set; }
    public decimal Variance { get; set; }
    public Enums.UnitOfMeasure Unit { get; set; }
    public string? Remarks { get; set; }

    /// <summary>Material Variance = Actual Material - Planned Material.</summary>
    public void Recalculate() => Variance = PlanningFormulas.MaterialVariance(ActualQuantity, PlannedQuantity);
}
