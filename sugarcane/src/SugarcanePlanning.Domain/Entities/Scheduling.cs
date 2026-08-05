using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>
/// A booking of a tractor / implement / operator against an activity plan for a time window
/// (section 10). The scheduling engine refuses to persist one that raises a conflict.
/// </summary>
public class ResourceSchedule : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ActivityPlanId { get; set; }
    public ActivityPlan? ActivityPlan { get; set; }

    public int BlockId { get; set; }
    public PlantationBlock? Block { get; set; }
    public int ActivityId { get; set; }
    public PlantingActivity? Activity { get; set; }

    public DateOnly ScheduleDate { get; set; }
    public DateTime PlannedStart { get; set; }
    public DateTime PlannedEnd { get; set; }

    public int? TractorId { get; set; }
    public Tractor? Tractor { get; set; }
    public int? EquipmentId { get; set; }
    public EquipmentItem? Equipment { get; set; }
    public int? OperatorId { get; set; }
    public Operator? Operator { get; set; }
    public int? WorkTeamId { get; set; }
    public WorkTeam? WorkTeam { get; set; }

    public decimal PlannedAreaHa { get; set; }
    public decimal DailyTargetHa { get; set; }
    public decimal ExpectedWorkingHours { get; set; }

    public string? SupervisorName { get; set; }
    public ScheduleStatus Status { get; set; } = ScheduleStatus.Planned;
    public string? Remarks { get; set; }

    /// <summary>Set when a manager overrode a blocking dependency (section 6).</summary>
    public bool DependencyOverrideApproved { get; set; }
    public string? DependencyOverrideBy { get; set; }
    public string? DependencyOverrideReason { get; set; }
}
