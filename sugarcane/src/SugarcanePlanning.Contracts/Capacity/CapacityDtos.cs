using System.ComponentModel.DataAnnotations;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Capacity;

/// <summary>One requirement-versus-availability comparison line (section 15).</summary>
public class CapacityLineDto
{
    /// <summary>Tractor | Equipment | Labor | Material | DailyCapacity | Completion.</summary>
    public string ResourceType { get; set; } = string.Empty;
    public int? ResourceId { get; set; }
    public string ResourceName { get; set; } = string.Empty;
    public string? Category { get; set; }
    public string Unit { get; set; } = string.Empty;

    public decimal Required { get; set; }
    public decimal Available { get; set; }
    public decimal Gap => Math.Round(Available - Required, 4);
    public decimal CoveragePercent { get; set; }
    public CapacityStatus Status { get; set; }
    public string? Recommendation { get; set; }
}

public class CapacityAnalysisDto
{
    public int ProjectionId { get; set; }
    public string ProjectionNo { get; set; } = string.Empty;
    public DateOnly PeriodStart { get; set; }
    public DateOnly PeriodEnd { get; set; }
    public int WorkingDays { get; set; }
    public decimal TotalPlannedAreaHa { get; set; }
    public decimal DailyCapacityRequiredHa { get; set; }
    public decimal DailyCapacityAvailableHa { get; set; }
    public DateOnly? PlannedCompletionDate { get; set; }
    public DateOnly? RequiredCompletionDate { get; set; }
    public int? CompletionVarianceDays { get; set; }
    public CapacityStatus OverallStatus { get; set; }
    public List<CapacityLineDto> Lines { get; set; } = new();
}

// ------------------------------------------------------------------ scenarios

public class ScenarioDto
{
    public int Id { get; set; }
    public int ProjectionId { get; set; }
    public string ProjectionNo { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? Description { get; set; }
    public ScenarioStatus Status { get; set; }
    public DateTime? SimulatedAtUtc { get; set; }
    public string? SimulatedBy { get; set; }
    public List<ScenarioAdjustmentDto> Adjustments { get; set; } = new();
    public byte[]? RowVersion { get; set; }
}

public class ScenarioAdjustmentDto
{
    public int Id { get; set; }
    public ScenarioAdjustmentType AdjustmentType { get; set; }
    public decimal Value { get; set; }
    public int? ActivityId { get; set; }
    public string? ActivityName { get; set; }
    public string? Remarks { get; set; }
}

public class ScenarioUpsertDto
{
    [Required] public int ProjectionId { get; set; }
    [Required, StringLength(150)] public string Name { get; set; } = string.Empty;
    [StringLength(1000)] public string? Description { get; set; }
    public List<ScenarioAdjustmentUpsertDto> Adjustments { get; set; } = new();
    public byte[]? RowVersion { get; set; }
}

public class ScenarioAdjustmentUpsertDto
{
    public ScenarioAdjustmentType AdjustmentType { get; set; }
    [Range(-10000, 100000)] public decimal Value { get; set; }
    public int? ActivityId { get; set; }
    [StringLength(500)] public string? Remarks { get; set; }
}

/// <summary>Baseline versus scenario, side by side. The approved plan is never written to.</summary>
public class ScenarioResultDto
{
    public int ScenarioId { get; set; }
    public string ScenarioName { get; set; } = string.Empty;
    public CapacityAnalysisDto Baseline { get; set; } = new();
    public CapacityAnalysisDto Simulated { get; set; } = new();
    public List<ScenarioDeltaDto> Deltas { get; set; } = new();
}

public class ScenarioDeltaDto
{
    public string ResourceType { get; set; } = string.Empty;
    public string ResourceName { get; set; } = string.Empty;
    public decimal BaselineGap { get; set; }
    public decimal SimulatedGap { get; set; }
    public decimal Improvement => Math.Round(SimulatedGap - BaselineGap, 4);
    public CapacityStatus BaselineStatus { get; set; }
    public CapacityStatus SimulatedStatus { get; set; }
}
