using System.Globalization;
using System.Linq.Expressions;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>
/// Planting projections: line validation (section 5), the approval workflow and the
/// revision/version chain (section 16).
/// </summary>
public class ProjectionService : ServiceBase, IProjectionService
{
    private readonly IAuditService _audit;

    public ProjectionService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock, IAuditService audit)
        : base(db, user, clock) => _audit = audit;

    // -------------------------------------------------------------------- reads

    public async Task<PagedResult<ProjectionSummaryDto>> GetProjectionsAsync(int? seasonId, int? estateId,
        QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Projections
            .Include(p => p.Estate).Include(p => p.GrowingSeason).Include(p => p.Lines)
            .AsNoTracking().AsQueryable();
        if (seasonId is not null) query = query.Where(p => p.GrowingSeasonId == seasonId);
        if (estateId is not null) query = query.Where(p => p.EstateId == estateId);
        if (!q.IncludeInactive) query = query.Where(p => p.IsCurrentVersion);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(p => p.ProjectionNo.Contains(q.Search) || (p.Remarks ?? "").Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<PlantingProjection, object>>>
        {
            ["projectionNo"] = p => p.ProjectionNo,
            ["date"] = p => p.ProjectionDate,
            ["area"] = p => p.TotalProjectedAreaHa,
            ["status"] = p => p.Status
        };
        return await query.ApplySort(q, sorts, p => p.ProjectionDate).ToPagedResultAsync(q, p => p.ToSummaryDto(), ct);
    }

    public async Task<ProjectionDetailDto> GetProjectionAsync(int id, CancellationToken ct = default)
        => (await LoadFullAsync(id, tracking: false, ct)).ToDetailDto();

    private async Task<PlantingProjection> LoadFullAsync(int id, bool tracking, CancellationToken ct)
    {
        var query = Db.Projections
            .Include(p => p.Estate)
            .Include(p => p.GrowingSeason)
            .Include(p => p.ApprovalHistory)
            .Include(p => p.Lines).ThenInclude(l => l.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .Include(p => p.Lines).ThenInclude(l => l.CaneVariety)
            .AsQueryable();
        if (!tracking) query = query.AsNoTracking();
        return await RequireAsync(query, id, "Planting projection", ct);
    }

    // ------------------------------------------------------------------- create

    public async Task<ProjectionDetailDto> CreateAsync(ProjectionCreateDto dto, CancellationToken ct = default)
    {
        var estate = await RequireAsync(Db.Estates, dto.EstateId, "Estate", ct);
        var season = await RequireAsync(Db.Seasons, dto.GrowingSeasonId, "Growing season", ct);
        if (season.Status == SeasonStatus.Closed)
            throw new BusinessRuleException("SEASON_CLOSED", "Projections cannot be created in a closed season.");

        await using var tx = await Db.BeginTransactionAsync(ct);

        var projection = new PlantingProjection
        {
            CompanyId = estate.CompanyId,
            EstateId = estate.Id,
            GrowingSeasonId = season.Id,
            ProjectionNo = await NextProjectionNumberAsync(season, ct),
            Version = 1,
            ProjectionDate = dto.ProjectionDate == default ? Clock.Today : dto.ProjectionDate,
            PlanningStartDate = dto.PlanningStartDate == default ? season.PlannedPlantingStart : dto.PlanningStartDate,
            PlanningEndDate = dto.PlanningEndDate == default ? season.PlannedPlantingEnd : dto.PlanningEndDate,
            Status = ProjectionStatus.Draft,
            PreparedBy = User.UserName,
            Remarks = dto.Remarks,
            IsCurrentVersion = true
        };

        Db.Projections.Add(projection);
        await Db.SaveChangesAsync(ct);

        foreach (var lineDto in dto.Lines)
            projection.Lines.Add(await BuildLineAsync(projection, season, lineDto, null, ct));

        projection.RecalculateTotals();
        await Db.SaveChangesAsync(ct);
        await tx.CommitAsync(ct);

        return (await LoadFullAsync(projection.Id, tracking: false, ct)).ToDetailDto();
    }

    /// <summary>Projection number: <c>PP-{season code}-{sequence}</c>, unique per season.</summary>
    private async Task<string> NextProjectionNumberAsync(GrowingSeason season, CancellationToken ct)
    {
        var count = await Db.Projections.CountAsync(p => p.GrowingSeasonId == season.Id && p.Version == 1, ct);
        return $"PP-{season.Code}-{(count + 1).ToString("D4", CultureInfo.InvariantCulture)}";
    }

    // -------------------------------------------------------------------- lines

    public async Task<ProjectionLineDto> AddLineAsync(int projectionId, ProjectionLineUpsertDto dto, CancellationToken ct = default)
    {
        var projection = await LoadFullAsync(projectionId, tracking: true, ct);
        RequireEditable(projection);
        var season = await RequireAsync(Db.Seasons, projection.GrowingSeasonId, "Growing season", ct);

        var line = await BuildLineAsync(projection, season, dto, null, ct);
        projection.Lines.Add(line);
        projection.RecalculateTotals();
        await Db.SaveChangesAsync(ct);

        return (await Db.ProjectionLines
            .Include(l => l.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .Include(l => l.CaneVariety)
            .AsNoTracking().FirstAsync(l => l.Id == line.Id, ct)).ToDto();
    }

    public async Task<ProjectionLineDto> UpdateLineAsync(int projectionId, int lineId, ProjectionLineUpsertDto dto,
        CancellationToken ct = default)
    {
        var projection = await LoadFullAsync(projectionId, tracking: true, ct);
        RequireEditable(projection);
        var season = await RequireAsync(Db.Seasons, projection.GrowingSeasonId, "Growing season", ct);

        var line = projection.Lines.FirstOrDefault(l => l.Id == lineId)
                   ?? throw new NotFoundException("Projection line", lineId);
        await BuildLineAsync(projection, season, dto, line, ct);
        projection.RecalculateTotals();
        await Db.SaveChangesAsync(ct);

        return (await Db.ProjectionLines
            .Include(l => l.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
            .Include(l => l.CaneVariety)
            .AsNoTracking().FirstAsync(l => l.Id == lineId, ct)).ToDto();
    }

    public async Task RemoveLineAsync(int projectionId, int lineId, CancellationToken ct = default)
    {
        var projection = await LoadFullAsync(projectionId, tracking: true, ct);
        RequireEditable(projection);
        var line = projection.Lines.FirstOrDefault(l => l.Id == lineId)
                   ?? throw new NotFoundException("Projection line", lineId);
        Db.ProjectionLines.Remove(line);
        projection.Lines.Remove(line);
        projection.RecalculateTotals();
        await Db.SaveChangesAsync(ct);
    }

    /// <summary>
    /// Applies every section-5 validation rule and the derived quantities to a line.
    /// Passing <paramref name="existing"/> updates in place; passing null builds a new line.
    /// </summary>
    private async Task<ProjectionLine> BuildLineAsync(PlantingProjection projection, GrowingSeason season,
        ProjectionLineUpsertDto dto, ProjectionLine? existing, CancellationToken ct)
    {
        var block = await RequireAsync(
            Db.Blocks.Include(b => b.Zone!).ThenInclude(z => z.Farm), dto.BlockId, "Plantation block", ct);
        var variety = await RequireAsync(Db.Varieties, dto.CaneVarietyId, "Sugarcane variety", ct);

        if (dto.ProjectedPlantingAreaHa <= 0)
            throw new BusinessRuleException("AREA_INVALID", "Projected planting area must be greater than zero.");
        if (dto.ProjectedPlantingAreaHa > block.PlantableAreaHa)
            throw new BusinessRuleException("AREA_EXCEEDS_BLOCK",
                $"Projected area {dto.ProjectedPlantingAreaHa:N2} ha exceeds the plantable area of block {block.Code} ({block.PlantableAreaHa:N2} ha).");
        if (dto.PlannedPlantingEnd < dto.PlannedPlantingStart)
            throw new BusinessRuleException("DATE_RANGE", "Planting end date must not be before the planting start date.");
        if (!season.ContainsPlantingWindow(dto.PlannedPlantingStart, dto.PlannedPlantingEnd))
            throw new BusinessRuleException("OUTSIDE_SEASON",
                $"Planting dates must fall inside the season planting window ({season.PlannedPlantingStart:yyyy-MM-dd} … {season.PlannedPlantingEnd:yyyy-MM-dd}).");

        await RequireNoOverlappingApprovedPlanAsync(projection, dto, existing?.Id ?? 0, block, ct);

        var line = existing ?? new ProjectionLine { CompanyId = projection.CompanyId, ProjectionId = projection.Id };
        line.BlockId = block.Id;
        line.ZoneId = block.ZoneId;
        line.FarmId = block.Zone?.FarmId ?? line.FarmId;
        line.CropType = dto.CropType;
        line.CaneVarietyId = variety.Id;
        line.AvailableAreaHa = block.PlantableAreaHa;
        line.ProjectedPlantingAreaHa = dto.ProjectedPlantingAreaHa;
        line.PlannedPlantingStart = dto.PlannedPlantingStart;
        line.PlannedPlantingEnd = dto.PlannedPlantingEnd;
        line.ExpectedYieldPerHa = dto.ExpectedYieldPerHa > 0 ? dto.ExpectedYieldPerHa : variety.ExpectedYieldPerHa;
        line.ExpectedLossPercent = dto.ExpectedLossPercent > 0 ? dto.ExpectedLossPercent : variety.ExpectedLossPercent;
        line.Priority = dto.Priority;
        line.Remarks = dto.Remarks;
        line.Recalculate(variety.GrowingPeriodMonths);
        return line;
    }

    /// <summary>
    /// Section 5: "Approved plans for the same block cannot overlap." Checks both other
    /// approved projections and the sibling lines of the document being edited.
    /// </summary>
    private async Task RequireNoOverlappingApprovedPlanAsync(PlantingProjection projection, ProjectionLineUpsertDto dto,
        int excludeLineId, PlantationBlock block, CancellationToken ct)
    {
        var approvedStatuses = new[] { ProjectionStatus.Approved, ProjectionStatus.Submitted, ProjectionStatus.UnderReview };

        var clash = await Db.ProjectionLines.AsNoTracking()
            .Include(l => l.Projection)
            .Where(l => l.BlockId == dto.BlockId
                        && l.Id != excludeLineId
                        && l.ProjectionId != projection.Id
                        && l.Projection!.IsCurrentVersion
                        && approvedStatuses.Contains(l.Projection.Status)
                        && l.PlannedPlantingStart <= dto.PlannedPlantingEnd
                        && dto.PlannedPlantingStart <= l.PlannedPlantingEnd)
            .Select(l => new { l.Projection!.ProjectionNo, l.PlannedPlantingStart, l.PlannedPlantingEnd })
            .FirstOrDefaultAsync(ct);

        if (clash is not null)
            throw new BusinessRuleException("BLOCK_OVERLAP",
                $"Block {block.Code} is already planned in projection {clash.ProjectionNo} between {clash.PlannedPlantingStart:yyyy-MM-dd} and {clash.PlannedPlantingEnd:yyyy-MM-dd}.");

        var siblingClash = projection.Lines
            .Where(l => l.Id != excludeLineId && l.BlockId == dto.BlockId && !l.IsDeleted)
            .Any(l => l.PlannedPlantingStart <= dto.PlannedPlantingEnd && dto.PlannedPlantingStart <= l.PlannedPlantingEnd);
        if (siblingClash)
            throw new BusinessRuleException("BLOCK_OVERLAP",
                $"Block {block.Code} already has an overlapping line in this projection.");

        // The sum over all lines of this block must still fit the plantable area.
        var otherArea = projection.Lines
            .Where(l => l.Id != excludeLineId && l.BlockId == dto.BlockId && !l.IsDeleted)
            .Sum(l => l.ProjectedPlantingAreaHa);
        if (otherArea + dto.ProjectedPlantingAreaHa > block.PlantableAreaHa)
            throw new BusinessRuleException("AREA_EXCEEDS_BLOCK",
                $"Total projected area for block {block.Code} ({otherArea + dto.ProjectedPlantingAreaHa:N2} ha) exceeds its plantable area ({block.PlantableAreaHa:N2} ha).");
    }

    /// <summary>
    /// Re-runs the block-overlap rule over the stored lines at the moment a document is committed.
    /// The check in <see cref="BuildLineAsync"/> only sees documents that were already submitted;
    /// two drafts written on the same block are legal, and it is the submission that decides which
    /// of them takes the block. Callers hold a lock on each block, so of two simultaneous
    /// submissions exactly one gets past this.
    /// </summary>
    private async Task RequireBlockStillFreeAsync(PlantingProjection projection, CancellationToken ct)
    {
        var committedStatuses = new[]
        {
            ProjectionStatus.Submitted, ProjectionStatus.UnderReview, ProjectionStatus.Approved
        };

        foreach (var line in projection.Lines.Where(l => !l.IsDeleted))
        {
            var blockId = line.BlockId;
            var start = line.PlannedPlantingStart;
            var end = line.PlannedPlantingEnd;

            var clash = await Db.ProjectionLines.AsNoTracking()
                .Where(l => l.BlockId == blockId
                            && l.ProjectionId != projection.Id
                            && l.Projection!.IsCurrentVersion
                            && committedStatuses.Contains(l.Projection.Status)
                            && l.PlannedPlantingStart <= end
                            && start <= l.PlannedPlantingEnd)
                .Select(l => new
                {
                    l.Projection!.ProjectionNo,
                    BlockCode = l.Block!.Code,
                    l.PlannedPlantingStart,
                    l.PlannedPlantingEnd
                })
                .FirstOrDefaultAsync(ct);

            if (clash is not null)
                throw new BusinessRuleException("BLOCK_OVERLAP",
                    $"Block {clash.BlockCode} was committed to projection {clash.ProjectionNo} between " +
                    $"{clash.PlannedPlantingStart:yyyy-MM-dd} and {clash.PlannedPlantingEnd:yyyy-MM-dd}; " +
                    "change the dates or the block before submitting.");
        }
    }

    // ------------------------------------------------------------------- header

    public async Task<ProjectionDetailDto> UpdateHeaderAsync(int id, ProjectionUpdateDto dto, CancellationToken ct = default)
    {
        var projection = await LoadFullAsync(id, tracking: true, ct);
        RequireEditable(projection);
        ApplyConcurrencyToken(projection, dto.RowVersion);

        if (dto.PlanningEndDate < dto.PlanningStartDate)
            throw new BusinessRuleException("DATE_RANGE", "Planning end date must not be before the planning start date.");

        projection.ProjectionDate = dto.ProjectionDate;
        projection.PlanningStartDate = dto.PlanningStartDate;
        projection.PlanningEndDate = dto.PlanningEndDate;
        projection.Remarks = dto.Remarks;
        projection.RecalculateTotals();
        await Db.SaveChangesAsync(ct);
        return (await LoadFullAsync(id, tracking: false, ct)).ToDetailDto();
    }

    public async Task DeleteAsync(int id, CancellationToken ct = default)
    {
        var projection = await RequireAsync(Db.Projections, id, "Planting projection", ct);
        if (projection.Status is not (ProjectionStatus.Draft or ProjectionStatus.Rejected))
            throw new BusinessRuleException("NOT_DELETABLE", "Only draft or rejected projections can be deleted.");
        SoftDelete(projection);
        await Db.SaveChangesAsync(ct);
    }

    private static void RequireEditable(PlantingProjection projection)
    {
        if (!projection.IsEditable)
            throw new BusinessRuleException("NOT_EDITABLE",
                $"Projection {projection.ProjectionNo} is {projection.Status} and can no longer be edited.");
    }

    // ----------------------------------------------------------------- workflow

    /// <summary>Legal transitions of the Draft → Submitted → Review → Approval workflow (section 16).</summary>
    private static readonly Dictionary<(ProjectionStatus, ApprovalAction), ProjectionStatus> Transitions = new()
    {
        [(ProjectionStatus.Draft, ApprovalAction.Submit)] = ProjectionStatus.Submitted,
        [(ProjectionStatus.Revised, ApprovalAction.Submit)] = ProjectionStatus.Submitted,
        [(ProjectionStatus.Rejected, ApprovalAction.Submit)] = ProjectionStatus.Submitted,
        [(ProjectionStatus.Submitted, ApprovalAction.Review)] = ProjectionStatus.UnderReview,
        [(ProjectionStatus.Submitted, ApprovalAction.Approve)] = ProjectionStatus.Approved,
        [(ProjectionStatus.UnderReview, ApprovalAction.Approve)] = ProjectionStatus.Approved,
        [(ProjectionStatus.Submitted, ApprovalAction.Reject)] = ProjectionStatus.Rejected,
        [(ProjectionStatus.UnderReview, ApprovalAction.Reject)] = ProjectionStatus.Rejected,
        [(ProjectionStatus.Submitted, ApprovalAction.ReturnForCorrection)] = ProjectionStatus.Draft,
        [(ProjectionStatus.UnderReview, ApprovalAction.ReturnForCorrection)] = ProjectionStatus.Draft,
        [(ProjectionStatus.Approved, ApprovalAction.Close)] = ProjectionStatus.Closed
    };

    /// <summary>Policy that a caller must hold to perform each action.</summary>
    private static readonly Dictionary<ApprovalAction, string> ActionPolicies = new()
    {
        [ApprovalAction.Submit] = Policies.Submit,
        [ApprovalAction.Review] = Policies.Approve,
        [ApprovalAction.Approve] = Policies.Approve,
        [ApprovalAction.Reject] = Policies.Reject,
        [ApprovalAction.ReturnForCorrection] = Policies.Reject,
        [ApprovalAction.Revise] = Policies.Revise,
        [ApprovalAction.Close] = Policies.Close
    };

    public async Task<ProjectionDetailDto> ExecuteWorkflowAsync(int id, WorkflowActionDto action, CancellationToken ct = default)
    {
        // Authorisation first: the permission depends only on the requested action, so an
        // unauthorised caller is refused before any document is read. Checking it after the load
        // would let them probe for ids — a missing projection answers 404 instead of 403.
        if (ActionPolicies.TryGetValue(action.Action, out var policy) && !User.HasPolicy(policy))
            throw new ForbiddenException($"Your roles do not allow the action '{action.Action}'.");

        var projection = await LoadFullAsync(id, tracking: true, ct);
        if (projection.IsReadOnly)
            throw new BusinessRuleException("READ_ONLY", "This version is read-only; work on the current version instead.");

        if (!Transitions.TryGetValue((projection.Status, action.Action), out var newStatus))
            throw new BusinessRuleException("INVALID_TRANSITION",
                $"'{action.Action}' is not allowed while the projection is {projection.Status}.");

        if (action.Action == ApprovalAction.Submit && projection.Lines.Count == 0)
            throw new BusinessRuleException("NO_LINES", "A projection cannot be submitted without projection lines.");
        if (action.Action == ApprovalAction.Reject && string.IsNullOrWhiteSpace(action.Comments))
            throw new BusinessRuleException("REASON_REQUIRED", "A rejection reason is required.");

        ApplyConcurrencyToken(projection, action.RowVersion);

        await using var tx = await Db.BeginTransactionAsync(ct);

        if (action.Action == ApprovalAction.Submit)
        {
            // Submitting is what commits a block to this document, so the overlap rule is checked
            // here as well as at line entry. The lock makes the check and the status change one
            // step: without it two drafts written at the same time could both be submitted.
            await Db.LockAsync(projection.Lines.Where(l => !l.IsDeleted).Select(l => $"block:{l.BlockId}"), ct);
            await RequireBlockStillFreeAsync(projection, ct);
        }

        var from = projection.Status;
        var now = Clock.UtcNow;
        projection.Status = newStatus;

        switch (action.Action)
        {
            case ApprovalAction.Submit:
                projection.SubmittedBy = User.UserName;
                projection.SubmittedAtUtc = now;
                projection.RejectionReason = null;
                break;
            case ApprovalAction.Review:
                projection.ReviewedBy = User.UserName;
                projection.ReviewedAtUtc = now;
                break;
            case ApprovalAction.Approve:
                projection.ApprovedBy = User.UserName;
                projection.ApprovedAtUtc = now;
                break;
            case ApprovalAction.Reject:
                projection.RejectedBy = User.UserName;
                projection.RejectedAtUtc = now;
                projection.RejectionReason = action.Comments;
                break;
        }

        projection.ApprovalHistory.Add(new ProjectionApprovalHistory
        {
            CompanyId = projection.CompanyId,
            ProjectionId = projection.Id,
            Action = action.Action,
            FromStatus = from,
            ToStatus = newStatus,
            ActionBy = User.UserName,
            ActionAtUtc = now,
            Comments = action.Comments
        });

        await Db.SaveChangesAsync(ct);
        await _audit.LogAsync(MapAuditAction(action.Action), nameof(PlantingProjection), projection.Id.ToString(),
            from.ToString(), newStatus.ToString(), action.Comments, ct);

        await tx.CommitAsync(ct);
        return (await LoadFullAsync(id, tracking: false, ct)).ToDetailDto();
    }

    private static AuditAction MapAuditAction(ApprovalAction action) => action switch
    {
        ApprovalAction.Submit => AuditAction.Submit,
        ApprovalAction.Approve => AuditAction.Approve,
        ApprovalAction.Reject => AuditAction.Reject,
        ApprovalAction.ReturnForCorrection => AuditAction.Return,
        ApprovalAction.Revise => AuditAction.Revise,
        ApprovalAction.Close => AuditAction.Close,
        _ => AuditAction.Update
    };

    // ----------------------------------------------------------------- revision

    public async Task<ProjectionDetailDto> ReviseAsync(int id, ReviseProjectionDto dto, CancellationToken ct = default)
    {
        if (!User.HasPolicy(Policies.Revise))
            throw new ForbiddenException("Your roles do not allow revising a projection.");

        var source = await LoadFullAsync(id, tracking: true, ct);
        if (source.Status != ProjectionStatus.Approved)
            throw new BusinessRuleException("NOT_APPROVED", "Only an approved projection can be revised.");
        if (!source.IsCurrentVersion)
            throw new BusinessRuleException("NOT_CURRENT", "Only the current version can be revised.");

        await using var tx = await Db.BeginTransactionAsync(ct);

        var revision = new PlantingProjection
        {
            CompanyId = source.CompanyId,
            EstateId = source.EstateId,
            GrowingSeasonId = source.GrowingSeasonId,
            ProjectionNo = source.ProjectionNo,
            Version = source.Version + 1,
            ProjectionDate = Clock.Today,
            PlanningStartDate = source.PlanningStartDate,
            PlanningEndDate = source.PlanningEndDate,
            Status = ProjectionStatus.Revised,
            PreparedBy = User.UserName,
            RevisedFromProjectionId = source.Id,
            RevisionReason = dto.RevisionReason,
            Remarks = source.Remarks,
            IsCurrentVersion = true
        };

        foreach (var line in source.Lines.Where(l => !l.IsDeleted))
        {
            revision.Lines.Add(new ProjectionLine
            {
                CompanyId = line.CompanyId,
                FarmId = line.FarmId,
                ZoneId = line.ZoneId,
                BlockId = line.BlockId,
                CropType = line.CropType,
                CaneVarietyId = line.CaneVarietyId,
                AvailableAreaHa = line.AvailableAreaHa,
                ProjectedPlantingAreaHa = line.ProjectedPlantingAreaHa,
                PlannedPlantingStart = line.PlannedPlantingStart,
                PlannedPlantingEnd = line.PlannedPlantingEnd,
                ExpectedHarvestDate = line.ExpectedHarvestDate,
                ExpectedYieldPerHa = line.ExpectedYieldPerHa,
                ExpectedLossPercent = line.ExpectedLossPercent,
                HarvestableAreaHa = line.HarvestableAreaHa,
                ExpectedCaneProductionTons = line.ExpectedCaneProductionTons,
                Priority = line.Priority,
                Remarks = line.Remarks
            });
        }
        revision.RecalculateTotals();

        // The approved version stays intact but becomes read-only history.
        source.IsCurrentVersion = false;
        source.IsReadOnly = true;

        revision.ApprovalHistory.Add(new ProjectionApprovalHistory
        {
            CompanyId = source.CompanyId,
            Action = ApprovalAction.Revise,
            FromStatus = ProjectionStatus.Approved,
            ToStatus = ProjectionStatus.Revised,
            ActionBy = User.UserName,
            ActionAtUtc = Clock.UtcNow,
            Comments = dto.RevisionReason
        });

        Db.Projections.Add(revision);
        await Db.SaveChangesAsync(ct);
        await tx.CommitAsync(ct);

        await _audit.LogAsync(AuditAction.Revise, nameof(PlantingProjection), revision.Id.ToString(),
            $"v{source.Version}", $"v{revision.Version}", dto.RevisionReason, ct);

        return (await LoadFullAsync(revision.Id, tracking: false, ct)).ToDetailDto();
    }

    public async Task<IReadOnlyList<ProjectionSummaryDto>> GetVersionsAsync(int id, CancellationToken ct = default)
    {
        var projection = await RequireAsync(Db.Projections.AsNoTracking(), id, "Planting projection", ct);
        var chain = await Db.Projections
            .Include(p => p.Estate).Include(p => p.GrowingSeason).Include(p => p.Lines)
            .AsNoTracking()
            .Where(p => p.ProjectionNo == projection.ProjectionNo && p.EstateId == projection.EstateId)
            .OrderBy(p => p.Version)
            .ToListAsync(ct);
        return chain.Select(p => p.ToSummaryDto()).ToList();
    }

    public async Task<VersionComparisonDto> CompareVersionsAsync(int fromId, int toId, CancellationToken ct = default)
    {
        var from = await LoadFullAsync(fromId, tracking: false, ct);
        var to = await LoadFullAsync(toId, tracking: false, ct);

        var result = new VersionComparisonDto
        {
            FromProjectionId = from.Id,
            FromVersion = from.Version,
            ToProjectionId = to.Id,
            ToVersion = to.Version,
            FromTotalAreaHa = from.TotalProjectedAreaHa,
            ToTotalAreaHa = to.TotalProjectedAreaHa,
            FromTotalProductionTons = from.TotalExpectedProductionTons,
            ToTotalProductionTons = to.TotalExpectedProductionTons
        };

        void CompareHeader(string field, string? a, string? b)
        {
            if (a == b) return;
            result.Differences.Add(new VersionDifferenceDto
            { Scope = "Header", Field = field, OldValue = a, NewValue = b });
        }

        CompareHeader("Planning start", from.PlanningStartDate.ToString("yyyy-MM-dd"), to.PlanningStartDate.ToString("yyyy-MM-dd"));
        CompareHeader("Planning end", from.PlanningEndDate.ToString("yyyy-MM-dd"), to.PlanningEndDate.ToString("yyyy-MM-dd"));
        CompareHeader("Total projected area (ha)", from.TotalProjectedAreaHa.ToString("N4"), to.TotalProjectedAreaHa.ToString("N4"));
        CompareHeader("Total expected production (t)", from.TotalExpectedProductionTons.ToString("N4"), to.TotalExpectedProductionTons.ToString("N4"));
        CompareHeader("Status", from.Status.ToString(), to.Status.ToString());
        CompareHeader("Remarks", from.Remarks, to.Remarks);

        var fromLines = from.Lines.ToDictionary(l => l.BlockId);
        var toLines = to.Lines.ToDictionary(l => l.BlockId);

        foreach (var (blockId, oldLine) in fromLines)
        {
            var blockCode = oldLine.Block?.Code ?? blockId.ToString();
            if (!toLines.TryGetValue(blockId, out var newLine))
            {
                result.Differences.Add(new VersionDifferenceDto
                {
                    Scope = "Line",
                    BlockCode = blockCode,
                    Field = "Projected area (ha)",
                    OldValue = oldLine.ProjectedPlantingAreaHa.ToString("N4"),
                    NewValue = null,
                    ChangeType = "Removed"
                });
                continue;
            }

            void CompareLine(string field, string? a, string? b)
            {
                if (a == b) return;
                result.Differences.Add(new VersionDifferenceDto
                { Scope = "Line", BlockCode = blockCode, Field = field, OldValue = a, NewValue = b });
            }

            CompareLine("Projected area (ha)", oldLine.ProjectedPlantingAreaHa.ToString("N4"), newLine.ProjectedPlantingAreaHa.ToString("N4"));
            CompareLine("Crop type", oldLine.CropType.ToString(), newLine.CropType.ToString());
            CompareLine("Variety", oldLine.CaneVarietyId.ToString(), newLine.CaneVarietyId.ToString());
            CompareLine("Planting start", oldLine.PlannedPlantingStart.ToString("yyyy-MM-dd"), newLine.PlannedPlantingStart.ToString("yyyy-MM-dd"));
            CompareLine("Planting end", oldLine.PlannedPlantingEnd.ToString("yyyy-MM-dd"), newLine.PlannedPlantingEnd.ToString("yyyy-MM-dd"));
            CompareLine("Yield per ha (t)", oldLine.ExpectedYieldPerHa.ToString("N4"), newLine.ExpectedYieldPerHa.ToString("N4"));
            CompareLine("Loss %", oldLine.ExpectedLossPercent.ToString("N2"), newLine.ExpectedLossPercent.ToString("N2"));
            CompareLine("Expected production (t)", oldLine.ExpectedCaneProductionTons.ToString("N4"), newLine.ExpectedCaneProductionTons.ToString("N4"));
        }

        foreach (var (blockId, newLine) in toLines.Where(kv => !fromLines.ContainsKey(kv.Key)))
        {
            result.Differences.Add(new VersionDifferenceDto
            {
                Scope = "Line",
                BlockCode = newLine.Block?.Code ?? blockId.ToString(),
                Field = "Projected area (ha)",
                OldValue = null,
                NewValue = newLine.ProjectedPlantingAreaHa.ToString("N4"),
                ChangeType = "Added"
            });
        }

        return result;
    }
}
