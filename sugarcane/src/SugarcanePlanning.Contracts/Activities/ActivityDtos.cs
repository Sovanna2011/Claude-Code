using System.ComponentModel.DataAnnotations;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Activities;

public class PlantingActivityDto
{
    public int Id { get; set; }
    public int CompanyId { get; set; }
    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public ActivityCategory Category { get; set; }
    public int SequenceNo { get; set; }
    public CropType ApplicableCropType { get; set; }
    public int StandardStartDayOffset { get; set; }
    public decimal StandardCapacityPerHour { get; set; }
    public decimal StandardCapacityPerDay { get; set; }
    public decimal StandardDurationPerHa { get; set; }
    public decimal StandardLaborDaysPerHa { get; set; }
    public bool IsMandatory { get; set; }
    public bool RequiresTractor { get; set; }
    public bool RequiresEquipment { get; set; }
    public bool RequiresMaterial { get; set; }
    public bool RequiresLabor { get; set; }
    public bool AllowOverlap { get; set; }
    public EquipmentCategory? DefaultEquipmentCategory { get; set; }
    public bool IsActive { get; set; } = true;
    public List<ActivityDependencyDto> Predecessors { get; set; } = new();
    public byte[]? RowVersion { get; set; }
}

public class PlantingActivityUpsertDto
{
    [Required] public int CompanyId { get; set; }
    [Required, StringLength(20)] public string Code { get; set; } = string.Empty;
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    public ActivityCategory Category { get; set; } = ActivityCategory.LandPreparation;
    [Range(1, 999)] public int SequenceNo { get; set; } = 1;
    public CropType ApplicableCropType { get; set; } = CropType.Both;
    [Range(-365, 365)] public int StandardStartDayOffset { get; set; }
    [Range(0, 10000)] public decimal StandardCapacityPerHour { get; set; }
    [Range(0, 10000)] public decimal StandardCapacityPerDay { get; set; }
    [Range(0, 10000)] public decimal StandardDurationPerHa { get; set; }
    [Range(0, 10000)] public decimal StandardLaborDaysPerHa { get; set; }
    public bool IsMandatory { get; set; } = true;
    public bool RequiresTractor { get; set; }
    public bool RequiresEquipment { get; set; }
    public bool RequiresMaterial { get; set; }
    public bool RequiresLabor { get; set; }
    public bool AllowOverlap { get; set; }
    public EquipmentCategory? DefaultEquipmentCategory { get; set; }
    public bool IsActive { get; set; } = true;
    public byte[]? RowVersion { get; set; }
}

public class ActivityDependencyDto
{
    public int Id { get; set; }
    public int ActivityId { get; set; }
    public string ActivityName { get; set; } = string.Empty;
    public int PredecessorActivityId { get; set; }
    public string PredecessorActivityName { get; set; } = string.Empty;
    public int LagDays { get; set; }
    public bool IsBlocking { get; set; } = true;
    public string? Remarks { get; set; }
}

public class ActivityDependencyUpsertDto
{
    [Required] public int ActivityId { get; set; }
    [Required] public int PredecessorActivityId { get; set; }
    [Range(0, 365)] public int LagDays { get; set; }
    public bool IsBlocking { get; set; } = true;
    [StringLength(500)] public string? Remarks { get; set; }
}

// ------------------------------------------------------------- activity plan

public class ActivityPlanDto
{
    public int Id { get; set; }
    public int ProjectionId { get; set; }
    public string ProjectionNo { get; set; } = string.Empty;
    public int ProjectionLineId { get; set; }
    public int FarmId { get; set; }
    public string FarmName { get; set; } = string.Empty;
    public int ZoneId { get; set; }
    public string ZoneName { get; set; } = string.Empty;
    public int BlockId { get; set; }
    public string BlockName { get; set; } = string.Empty;
    public int ActivityId { get; set; }
    public string ActivityCode { get; set; } = string.Empty;
    public string ActivityName { get; set; } = string.Empty;
    public int SequenceNo { get; set; }
    public decimal PlannedAreaHa { get; set; }
    public DateOnly PlannedStartDate { get; set; }
    public DateOnly PlannedEndDate { get; set; }
    public int WorkingDays { get; set; }
    public decimal DailyTargetHa { get; set; }
    public decimal PlannedWorkingHours { get; set; }
    public string? RequiredTractorType { get; set; }
    public EquipmentCategory? RequiredEquipmentCategory { get; set; }
    public int RequiredTractorCount { get; set; }
    public int RequiredEquipmentCount { get; set; }
    public decimal RequiredLaborDays { get; set; }
    public int RequiredWorkers { get; set; }
    public decimal PlannedFuelLiters { get; set; }
    public string? SupervisorName { get; set; }
    public ActivityStatus Status { get; set; }
    public string? Remarks { get; set; }
    public decimal CompletionPercent { get; set; }
    public byte[]? RowVersion { get; set; }
}

/// <summary>Editable subset of an activity plan (dates, area, supervisor, status).</summary>
public class ActivityPlanUpdateDto
{
    public DateOnly PlannedStartDate { get; set; }
    public DateOnly PlannedEndDate { get; set; }
    [Range(0, 999_999)] public decimal PlannedAreaHa { get; set; }
    [StringLength(150)] public string? SupervisorName { get; set; }
    public ActivityStatus Status { get; set; }
    [StringLength(1000)] public string? Remarks { get; set; }
    public byte[]? RowVersion { get; set; }
}

/// <summary>Bar for the Gantt / timeline view.</summary>
public class GanttBarDto
{
    public int ActivityPlanId { get; set; }
    public string BlockName { get; set; } = string.Empty;
    public string ActivityName { get; set; } = string.Empty;
    public int SequenceNo { get; set; }
    public DateOnly Start { get; set; }
    public DateOnly End { get; set; }
    public ActivityStatus Status { get; set; }
    public decimal PlannedAreaHa { get; set; }
    public decimal CompletionPercent { get; set; }
    public List<int> PredecessorPlanIds { get; set; } = new();
}

public class GanttViewDto
{
    public DateOnly RangeStart { get; set; }
    public DateOnly RangeEnd { get; set; }
    public List<GanttBarDto> Bars { get; set; } = new();
}

/// <summary>Request that turns an approved projection into activity plans (section 7).</summary>
public class GenerateActivityPlanRequest
{
    [Required] public int ProjectionId { get; set; }
    /// <summary>Replace any plans generated earlier for this projection.</summary>
    public bool Regenerate { get; set; } = true;
    public bool WorkOnSaturday { get; set; } = true;
    public bool WorkOnSunday { get; set; }
}

public class GenerateActivityPlanResultDto
{
    public int ProjectionId { get; set; }
    public int PlansCreated { get; set; }
    public int PlansRemoved { get; set; }
    public DateOnly? EarliestStart { get; set; }
    public DateOnly? LatestEnd { get; set; }
    public List<string> Warnings { get; set; } = new();
}
