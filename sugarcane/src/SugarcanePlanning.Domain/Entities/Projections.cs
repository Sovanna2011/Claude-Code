using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>Planting projection header — the versioned planning document (section 5 / 16).</summary>
public class PlantingProjection : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public int EstateId { get; set; }
    public Estate? Estate { get; set; }
    public int GrowingSeasonId { get; set; }
    public GrowingSeason? GrowingSeason { get; set; }

    public string ProjectionNo { get; set; } = string.Empty;
    public int Version { get; set; } = 1;
    public DateOnly ProjectionDate { get; set; }

    public DateOnly PlanningStartDate { get; set; }
    public DateOnly PlanningEndDate { get; set; }

    public decimal TotalProjectedAreaHa { get; set; }
    public decimal TotalExpectedProductionTons { get; set; }

    public ProjectionStatus Status { get; set; } = ProjectionStatus.Draft;

    public string? PreparedBy { get; set; }
    public string? SubmittedBy { get; set; }
    public DateTime? SubmittedAtUtc { get; set; }
    public string? ReviewedBy { get; set; }
    public DateTime? ReviewedAtUtc { get; set; }
    public string? ApprovedBy { get; set; }
    public DateTime? ApprovedAtUtc { get; set; }
    public string? RejectedBy { get; set; }
    public DateTime? RejectedAtUtc { get; set; }
    public string? RejectionReason { get; set; }

    /// <summary>Set on a revision; points at the version this document was copied from.</summary>
    public int? RevisedFromProjectionId { get; set; }
    public PlantingProjection? RevisedFrom { get; set; }
    public string? RevisionReason { get; set; }

    /// <summary>Read-only versions are superseded revisions: they can never be edited again.</summary>
    public bool IsReadOnly { get; set; }
    /// <summary>Marks the newest version in a revision chain.</summary>
    public bool IsCurrentVersion { get; set; } = true;

    public string? Remarks { get; set; }

    public ICollection<ProjectionLine> Lines { get; set; } = new List<ProjectionLine>();
    public ICollection<ProjectionApprovalHistory> ApprovalHistory { get; set; } = new List<ProjectionApprovalHistory>();
    public ICollection<ActivityPlan> ActivityPlans { get; set; } = new List<ActivityPlan>();

    /// <summary>Header totals are always derived from the lines (section 5).</summary>
    public void RecalculateTotals()
    {
        TotalProjectedAreaHa = Math.Round(Lines.Where(l => !l.IsDeleted).Sum(l => l.ProjectedPlantingAreaHa), PlanningFormulas.QuantityScale);
        TotalExpectedProductionTons = Math.Round(Lines.Where(l => !l.IsDeleted).Sum(l => l.ExpectedCaneProductionTons), PlanningFormulas.QuantityScale);
    }

    /// <summary>Only draft and returned documents accept edits.</summary>
    public bool IsEditable => !IsReadOnly && Status is ProjectionStatus.Draft or ProjectionStatus.Rejected or ProjectionStatus.Revised;
}

/// <summary>One block's worth of planting inside a projection (section 5).</summary>
public class ProjectionLine : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ProjectionId { get; set; }
    public PlantingProjection? Projection { get; set; }

    public int FarmId { get; set; }
    public Farm? Farm { get; set; }
    public int ZoneId { get; set; }
    public Zone? Zone { get; set; }
    public int BlockId { get; set; }
    public PlantationBlock? Block { get; set; }

    public CropType CropType { get; set; } = CropType.NewPlanting;

    public int CaneVarietyId { get; set; }
    public CaneVariety? CaneVariety { get; set; }

    /// <summary>Plantable area of the block at the time the line was created.</summary>
    public decimal AvailableAreaHa { get; set; }
    public decimal ProjectedPlantingAreaHa { get; set; }

    public DateOnly PlannedPlantingStart { get; set; }
    public DateOnly PlannedPlantingEnd { get; set; }
    public DateOnly? ExpectedHarvestDate { get; set; }

    public decimal ExpectedYieldPerHa { get; set; }
    public decimal ExpectedLossPercent { get; set; }

    // Derived — recomputed by Recalculate() and persisted for reporting.
    public decimal HarvestableAreaHa { get; set; }
    public decimal ExpectedCaneProductionTons { get; set; }

    public int Priority { get; set; } = 5;
    public string? Remarks { get; set; }

    public ICollection<ActivityPlan> ActivityPlans { get; set; } = new List<ActivityPlan>();

    /// <summary>Applies the section-5 formulas and derives the expected harvest date from the variety.</summary>
    public void Recalculate(int? growingPeriodMonths = null)
    {
        HarvestableAreaHa = PlanningFormulas.HarvestableArea(ProjectedPlantingAreaHa, ExpectedLossPercent);
        ExpectedCaneProductionTons = PlanningFormulas.ExpectedCaneProduction(HarvestableAreaHa, ExpectedYieldPerHa);
        if (growingPeriodMonths is > 0)
            ExpectedHarvestDate = PlannedPlantingEnd.AddMonths(growingPeriodMonths.Value);
    }
}

/// <summary>Every workflow transition a projection went through (section 16 / 22).</summary>
public class ProjectionApprovalHistory : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }

    public int ProjectionId { get; set; }
    public PlantingProjection? Projection { get; set; }

    public ApprovalAction Action { get; set; }
    public ProjectionStatus FromStatus { get; set; }
    public ProjectionStatus ToStatus { get; set; }

    public string ActionBy { get; set; } = string.Empty;
    public DateTime ActionAtUtc { get; set; }
    public string? Comments { get; set; }
}
