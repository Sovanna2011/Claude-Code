using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>
/// A what-if simulation over an approved projection (section 15). Scenarios are stored
/// separately and never write back to the plan — adopting one is an explicit revision.
/// </summary>
public class PlanningScenario : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ProjectionId { get; set; }
    public PlantingProjection? Projection { get; set; }

    public string Name { get; set; } = string.Empty;
    public string? Description { get; set; }
    public ScenarioStatus Status { get; set; } = ScenarioStatus.Draft;

    public DateTime? SimulatedAtUtc { get; set; }
    public string? SimulatedBy { get; set; }

    /// <summary>Serialized capacity result of the last simulation run, for side-by-side comparison.</summary>
    public string? ResultJson { get; set; }

    public ICollection<ScenarioAdjustment> Adjustments { get; set; } = new List<ScenarioAdjustment>();
}

/// <summary>One lever pulled inside a scenario (section 15).</summary>
public class ScenarioAdjustment : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ScenarioId { get; set; }
    public PlanningScenario? Scenario { get; set; }

    public ScenarioAdjustmentType AdjustmentType { get; set; }

    /// <summary>Numeric magnitude: extra machines, extra hours per day, percent area cut, day shift …</summary>
    public decimal Value { get; set; }

    /// <summary>Optional narrowing of the adjustment to one activity.</summary>
    public int? ActivityId { get; set; }
    public PlantingActivity? Activity { get; set; }

    public string? Remarks { get; set; }
}
