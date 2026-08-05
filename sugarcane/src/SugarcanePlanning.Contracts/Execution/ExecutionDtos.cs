using System.ComponentModel.DataAnnotations;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Execution;

public class ActivityActualDto
{
    public int Id { get; set; }
    public int ActivityPlanId { get; set; }
    public string BlockName { get; set; } = string.Empty;
    public string ActivityName { get; set; } = string.Empty;

    public DateOnly PlannedStartDate { get; set; }
    public DateOnly PlannedEndDate { get; set; }
    public decimal PlannedAreaHa { get; set; }
    public decimal PlannedFuelLiters { get; set; }

    public DateOnly? ActualStartDate { get; set; }
    public DateOnly? ActualCompletionDate { get; set; }
    public decimal ActualCompletedAreaHa { get; set; }
    public int? ActualTractorId { get; set; }
    public string? ActualTractorCode { get; set; }
    public int? ActualEquipmentId { get; set; }
    public string? ActualEquipmentCode { get; set; }
    public int? ActualOperatorId { get; set; }
    public string? ActualOperatorName { get; set; }
    public decimal ActualWorkingHours { get; set; }
    public decimal ActualFuelLiters { get; set; }
    public decimal ActualLaborDays { get; set; }

    public decimal AreaVariance { get; set; }
    public decimal FuelVariance { get; set; }
    public int? ScheduleVarianceDays { get; set; }
    public decimal CompletionPercent { get; set; }

    public string? DelayReason { get; set; }
    public string? Remarks { get; set; }
    public List<ActualMaterialUsageDto> MaterialUsages { get; set; } = new();
    public byte[]? RowVersion { get; set; }
}

public class ActivityActualUpsertDto
{
    [Required] public int ActivityPlanId { get; set; }
    public DateOnly? ActualStartDate { get; set; }
    public DateOnly? ActualCompletionDate { get; set; }
    [Range(0, 999_999)] public decimal ActualCompletedAreaHa { get; set; }
    public int? ActualTractorId { get; set; }
    public int? ActualEquipmentId { get; set; }
    public int? ActualOperatorId { get; set; }
    [Range(0, 100_000)] public decimal ActualWorkingHours { get; set; }
    [Range(0, 1_000_000)] public decimal ActualFuelLiters { get; set; }
    [Range(0, 1_000_000)] public decimal ActualLaborDays { get; set; }
    [StringLength(500)] public string? DelayReason { get; set; }
    [StringLength(1000)] public string? Remarks { get; set; }
    public List<ActualMaterialUsageUpsertDto> MaterialUsages { get; set; } = new();
    public byte[]? RowVersion { get; set; }
}

public class ActualMaterialUsageDto
{
    public int Id { get; set; }
    public int MaterialId { get; set; }
    public string MaterialCode { get; set; } = string.Empty;
    public string MaterialName { get; set; } = string.Empty;
    public decimal PlannedQuantity { get; set; }
    public decimal ActualQuantity { get; set; }
    public decimal Variance { get; set; }
    public UnitOfMeasure Unit { get; set; }
    public string? Remarks { get; set; }
}

public class ActualMaterialUsageUpsertDto
{
    [Required] public int MaterialId { get; set; }
    [Range(0, 10_000_000)] public decimal ActualQuantity { get; set; }
    [StringLength(500)] public string? Remarks { get; set; }
}

/// <summary>Projection-versus-actual comparison row (section 17).</summary>
public class ProjectionVsActualDto
{
    public int? BlockId { get; set; }
    public string GroupName { get; set; } = string.Empty;
    public string GroupBy { get; set; } = "block";      // block | farm | activity | month
    public decimal PlannedAreaHa { get; set; }
    public decimal ActualAreaHa { get; set; }
    public decimal AreaVariance { get; set; }
    public decimal PlannedFuelLiters { get; set; }
    public decimal ActualFuelLiters { get; set; }
    public decimal FuelVariance { get; set; }
    public decimal PlannedLaborDays { get; set; }
    public decimal ActualLaborDays { get; set; }
    public decimal CompletionPercent { get; set; }
    public int PlansTotal { get; set; }
    public int PlansCompleted { get; set; }
    public int PlansDelayed { get; set; }
    public int? AverageScheduleVarianceDays { get; set; }
}
