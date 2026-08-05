using System.ComponentModel.DataAnnotations;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Scheduling;

public class ResourceScheduleDto
{
    public int Id { get; set; }
    public int ActivityPlanId { get; set; }
    public int BlockId { get; set; }
    public string BlockCode { get; set; } = string.Empty;
    public string BlockName { get; set; } = string.Empty;
    public string FarmName { get; set; } = string.Empty;
    public int ActivityId { get; set; }
    public string ActivityName { get; set; } = string.Empty;
    public DateOnly ScheduleDate { get; set; }
    public DateTime PlannedStart { get; set; }
    public DateTime PlannedEnd { get; set; }
    public int? TractorId { get; set; }
    public string? TractorCode { get; set; }
    public int? EquipmentId { get; set; }
    public string? EquipmentCode { get; set; }
    public int? OperatorId { get; set; }
    public string? OperatorName { get; set; }
    public int? WorkTeamId { get; set; }
    public string? WorkTeamName { get; set; }
    public decimal PlannedAreaHa { get; set; }
    public decimal DailyTargetHa { get; set; }
    public decimal ExpectedWorkingHours { get; set; }
    public string? SupervisorName { get; set; }
    public ScheduleStatus Status { get; set; }
    public string? Remarks { get; set; }
    public bool DependencyOverrideApproved { get; set; }
    public string? DependencyOverrideReason { get; set; }
    public byte[]? RowVersion { get; set; }
}

public class ResourceScheduleUpsertDto
{
    [Required] public int ActivityPlanId { get; set; }
    public DateOnly ScheduleDate { get; set; }
    public DateTime PlannedStart { get; set; }
    public DateTime PlannedEnd { get; set; }
    public int? TractorId { get; set; }
    public int? EquipmentId { get; set; }
    public int? OperatorId { get; set; }
    public int? WorkTeamId { get; set; }
    [Range(0, 999_999)] public decimal PlannedAreaHa { get; set; }
    [Range(0, 24)] public decimal ExpectedWorkingHours { get; set; } = 8m;
    [StringLength(150)] public string? SupervisorName { get; set; }
    public ScheduleStatus Status { get; set; } = ScheduleStatus.Planned;
    [StringLength(1000)] public string? Remarks { get; set; }

    /// <summary>Manager override for a blocking activity dependency (section 6).</summary>
    public bool OverrideDependency { get; set; }
    [StringLength(500)] public string? DependencyOverrideReason { get; set; }

    public byte[]? RowVersion { get; set; }
}

/// <summary>One reason a booking cannot be saved (section 10).</summary>
public class ScheduleConflictDto
{
    public ConflictType Type { get; set; }
    public string Message { get; set; } = string.Empty;
    public int? ConflictingScheduleId { get; set; }
    public string? ResourceCode { get; set; }
    public DateTime? ConflictStart { get; set; }
    public DateTime? ConflictEnd { get; set; }
    /// <summary>Blocking conflicts stop the save; non-blocking ones are warnings.</summary>
    public bool IsBlocking { get; set; } = true;
}

public class ScheduleValidationResultDto
{
    public bool IsValid => Conflicts.All(c => !c.IsBlocking);
    public List<ScheduleConflictDto> Conflicts { get; set; } = new();
}

/// <summary>Calendar cell for the daily / weekly / monthly scheduling boards.</summary>
public class ScheduleCalendarDayDto
{
    public DateOnly Date { get; set; }
    public decimal PlannedAreaHa { get; set; }
    public int BookingCount { get; set; }
    public int TractorsBooked { get; set; }
    public int EquipmentBooked { get; set; }
    public int OperatorsBooked { get; set; }
    public List<ResourceScheduleDto> Bookings { get; set; } = new();
}

public class ScheduleBoardDto
{
    public DateOnly RangeStart { get; set; }
    public DateOnly RangeEnd { get; set; }
    public string GroupBy { get; set; } = "day";     // day | tractor | equipment | operator | farm | block
    public List<ScheduleCalendarDayDto> Days { get; set; } = new();
    public List<ScheduleResourceRowDto> Rows { get; set; } = new();
}

/// <summary>A resource lane (one tractor / implement / operator) with its bookings.</summary>
public class ScheduleResourceRowDto
{
    public string ResourceType { get; set; } = string.Empty;
    public int ResourceId { get; set; }
    public string ResourceCode { get; set; } = string.Empty;
    public string ResourceName { get; set; } = string.Empty;
    public decimal TotalHours { get; set; }
    public decimal TotalAreaHa { get; set; }
    public int UtilizedDays { get; set; }
    public decimal UtilizationPercent { get; set; }
    public List<ResourceScheduleDto> Bookings { get; set; } = new();
}
