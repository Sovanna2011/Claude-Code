using System.ComponentModel.DataAnnotations;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Projections;

public class ProjectionSummaryDto
{
    public int Id { get; set; }
    public string ProjectionNo { get; set; } = string.Empty;
    public int Version { get; set; }
    public int CompanyId { get; set; }
    public int EstateId { get; set; }
    public string EstateName { get; set; } = string.Empty;
    public int GrowingSeasonId { get; set; }
    public string SeasonName { get; set; } = string.Empty;
    public DateOnly ProjectionDate { get; set; }
    public DateOnly PlanningStartDate { get; set; }
    public DateOnly PlanningEndDate { get; set; }
    public decimal TotalProjectedAreaHa { get; set; }
    public decimal TotalExpectedProductionTons { get; set; }
    public ProjectionStatus Status { get; set; }
    public bool IsCurrentVersion { get; set; }
    public bool IsReadOnly { get; set; }
    public string? PreparedBy { get; set; }
    public string? ApprovedBy { get; set; }
    public DateTime? ApprovedAtUtc { get; set; }
    public int LineCount { get; set; }
}

public class ProjectionDetailDto : ProjectionSummaryDto
{
    public string? SubmittedBy { get; set; }
    public DateTime? SubmittedAtUtc { get; set; }
    public string? ReviewedBy { get; set; }
    public DateTime? ReviewedAtUtc { get; set; }
    public string? RejectedBy { get; set; }
    public DateTime? RejectedAtUtc { get; set; }
    public string? RejectionReason { get; set; }
    public int? RevisedFromProjectionId { get; set; }
    public string? RevisionReason { get; set; }
    public string? Remarks { get; set; }
    public List<ProjectionLineDto> Lines { get; set; } = new();
    public List<ApprovalHistoryDto> ApprovalHistory { get; set; } = new();
    public byte[]? RowVersion { get; set; }
}

public class ProjectionLineDto
{
    public int Id { get; set; }
    public int ProjectionId { get; set; }
    public int FarmId { get; set; }
    public string FarmName { get; set; } = string.Empty;
    public int ZoneId { get; set; }
    public string ZoneName { get; set; } = string.Empty;
    public int BlockId { get; set; }
    public string BlockCode { get; set; } = string.Empty;
    public string BlockName { get; set; } = string.Empty;
    public CropType CropType { get; set; }
    public int CaneVarietyId { get; set; }
    public string VarietyName { get; set; } = string.Empty;
    public decimal AvailableAreaHa { get; set; }
    public decimal ProjectedPlantingAreaHa { get; set; }
    public DateOnly PlannedPlantingStart { get; set; }
    public DateOnly PlannedPlantingEnd { get; set; }
    public DateOnly? ExpectedHarvestDate { get; set; }
    public decimal ExpectedYieldPerHa { get; set; }
    public decimal ExpectedLossPercent { get; set; }
    public decimal HarvestableAreaHa { get; set; }
    public decimal ExpectedCaneProductionTons { get; set; }
    public int Priority { get; set; }
    public string? Remarks { get; set; }
}

public class ProjectionCreateDto
{
    [Required] public int EstateId { get; set; }
    [Required] public int GrowingSeasonId { get; set; }
    public DateOnly ProjectionDate { get; set; }
    public DateOnly PlanningStartDate { get; set; }
    public DateOnly PlanningEndDate { get; set; }
    [StringLength(1000)] public string? Remarks { get; set; }
    public List<ProjectionLineUpsertDto> Lines { get; set; } = new();
}

public class ProjectionUpdateDto
{
    public DateOnly ProjectionDate { get; set; }
    public DateOnly PlanningStartDate { get; set; }
    public DateOnly PlanningEndDate { get; set; }
    [StringLength(1000)] public string? Remarks { get; set; }
    public byte[]? RowVersion { get; set; }
}

public class ProjectionLineUpsertDto
{
    public int Id { get; set; }
    [Required] public int BlockId { get; set; }
    public CropType CropType { get; set; } = CropType.NewPlanting;
    [Required] public int CaneVarietyId { get; set; }
    [Range(0.0001, 999_999)] public decimal ProjectedPlantingAreaHa { get; set; }
    public DateOnly PlannedPlantingStart { get; set; }
    public DateOnly PlannedPlantingEnd { get; set; }
    [Range(0, 1000)] public decimal ExpectedYieldPerHa { get; set; }
    [Range(0, 100)] public decimal ExpectedLossPercent { get; set; }
    [Range(1, 9)] public int Priority { get; set; } = 5;
    [StringLength(500)] public string? Remarks { get; set; }
}

public class ApprovalHistoryDto
{
    public int Id { get; set; }
    public ApprovalAction Action { get; set; }
    public ProjectionStatus FromStatus { get; set; }
    public ProjectionStatus ToStatus { get; set; }
    public string ActionBy { get; set; } = string.Empty;
    public DateTime ActionAtUtc { get; set; }
    public string? Comments { get; set; }
}

/// <summary>Submit / approve / reject / return / close payload (section 16).</summary>
public class WorkflowActionDto
{
    [Required] public ApprovalAction Action { get; set; }
    [StringLength(1000)] public string? Comments { get; set; }
    public byte[]? RowVersion { get; set; }
}

public class ReviseProjectionDto
{
    [Required, StringLength(1000)] public string RevisionReason { get; set; } = string.Empty;
}

/// <summary>Field-by-field difference between two versions (section 16).</summary>
public class VersionComparisonDto
{
    public int FromProjectionId { get; set; }
    public int FromVersion { get; set; }
    public int ToProjectionId { get; set; }
    public int ToVersion { get; set; }
    public decimal FromTotalAreaHa { get; set; }
    public decimal ToTotalAreaHa { get; set; }
    public decimal FromTotalProductionTons { get; set; }
    public decimal ToTotalProductionTons { get; set; }
    public List<VersionDifferenceDto> Differences { get; set; } = new();
}

public class VersionDifferenceDto
{
    public string Scope { get; set; } = string.Empty;      // Header | Line
    public string BlockCode { get; set; } = string.Empty;
    public string Field { get; set; } = string.Empty;
    public string? OldValue { get; set; }
    public string? NewValue { get; set; }
    public string ChangeType { get; set; } = "Changed";     // Added | Removed | Changed
}
