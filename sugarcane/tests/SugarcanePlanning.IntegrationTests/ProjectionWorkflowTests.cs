using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>Section 5 validation rules and the section 16 approval workflow, end to end.</summary>
public class ProjectionValidationTests : IDisposable
{
    private readonly PlanningTestHost _host = new();
    public void Dispose() => _host.Dispose();

    [Fact]
    public async Task Creating_a_projection_derives_the_number_and_the_header_totals()
    {
        var created = await _host.Projections.CreateAsync(_host.NewProjection(areaHa: 80m));

        Assert.Equal("PP-S26-0001", created.ProjectionNo);
        Assert.Equal(1, created.Version);
        Assert.Equal(ProjectionStatus.Draft, created.Status);
        Assert.Equal(80m, created.TotalProjectedAreaHa);

        // 80 ha x (1 - 5%) = 76 ha harvestable x 90 t/ha = 6,840 t
        Assert.Equal(6_840m, created.TotalExpectedProductionTons);
        Assert.Equal(76m, created.Lines[0].HarvestableAreaHa);
        Assert.Equal(new DateOnly(2027, 3, 20), created.Lines[0].ExpectedHarvestDate);
    }

    [Fact]
    public async Task Projected_area_may_not_exceed_the_plantable_area_of_the_block()
    {
        var request = _host.NewProjection(areaHa: 150m);       // block A has 100 ha plantable

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() => _host.Projections.CreateAsync(request));
        Assert.Equal("AREA_EXCEEDS_BLOCK", ex.Code);
    }

    [Fact]
    public async Task Planting_dates_must_lie_inside_the_season_planting_window()
    {
        var request = _host.NewProjection();
        request.Lines[0].PlannedPlantingStart = new DateOnly(2026, 2, 1);
        request.Lines[0].PlannedPlantingEnd = new DateOnly(2026, 2, 20);

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() => _host.Projections.CreateAsync(request));
        Assert.Equal("OUTSIDE_SEASON", ex.Code);
    }

    [Fact]
    public async Task Two_lines_on_the_same_block_may_not_overlap_in_time()
    {
        var created = await _host.Projections.CreateAsync(_host.NewProjection(areaHa: 40m));

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() => _host.Projections.AddLineAsync(created.Id,
            new ProjectionLineUpsertDto
            {
                BlockId = _host.BlockAId,
                CaneVarietyId = _host.VarietyId,
                ProjectedPlantingAreaHa = 30m,
                PlannedPlantingStart = new DateOnly(2026, 3, 15),   // overlaps 3-02 … 3-20
                PlannedPlantingEnd = new DateOnly(2026, 4, 5)
            }));

        Assert.Equal("BLOCK_OVERLAP", ex.Code);
    }

    [Fact]
    public async Task The_sum_of_lines_on_one_block_may_not_exceed_its_plantable_area()
    {
        var created = await _host.Projections.CreateAsync(_host.NewProjection(areaHa: 80m));

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() => _host.Projections.AddLineAsync(created.Id,
            new ProjectionLineUpsertDto
            {
                BlockId = _host.BlockAId,
                CaneVarietyId = _host.VarietyId,
                ProjectedPlantingAreaHa = 30m,                      // 80 + 30 > 100
                PlannedPlantingStart = new DateOnly(2026, 5, 1),
                PlannedPlantingEnd = new DateOnly(2026, 5, 20)
            }));

        Assert.Equal("AREA_EXCEEDS_BLOCK", ex.Code);
    }

    [Fact]
    public async Task A_second_block_can_be_added_and_the_header_totals_follow()
    {
        var created = await _host.Projections.CreateAsync(_host.NewProjection(areaHa: 80m));

        await _host.Projections.AddLineAsync(created.Id, new ProjectionLineUpsertDto
        {
            BlockId = _host.BlockBId,
            CaneVarietyId = _host.VarietyId,
            ProjectedPlantingAreaHa = 40m,
            PlannedPlantingStart = new DateOnly(2026, 4, 1),
            PlannedPlantingEnd = new DateOnly(2026, 4, 20)
        });

        var reloaded = await _host.Projections.GetProjectionAsync(created.Id);
        Assert.Equal(2, reloaded.Lines.Count);
        Assert.Equal(120m, reloaded.TotalProjectedAreaHa);
    }

    [Fact]
    public async Task Removing_a_line_recalculates_the_header()
    {
        var created = await _host.Projections.CreateAsync(_host.NewProjection(areaHa: 80m));
        await _host.Projections.RemoveLineAsync(created.Id, created.Lines[0].Id);

        var reloaded = await _host.Projections.GetProjectionAsync(created.Id);
        Assert.Empty(reloaded.Lines);
        Assert.Equal(0m, reloaded.TotalProjectedAreaHa);
    }

    [Fact]
    public async Task An_unknown_block_is_reported_as_not_found()
    {
        var request = _host.NewProjection();
        request.Lines[0].BlockId = 987_654;

        await Assert.ThrowsAsync<NotFoundException>(() => _host.Projections.CreateAsync(request));
    }
}

public class ProjectionWorkflowTests : IDisposable
{
    private readonly PlanningTestHost _host = new();
    public void Dispose() => _host.Dispose();

    private Task<ProjectionDetailDto> CreateAsync() => _host.Projections.CreateAsync(_host.NewProjection());

    [Fact]
    public async Task Draft_submitted_approved_is_the_happy_path()
    {
        var projection = await CreateAsync();

        projection = await _host.Projections.ExecuteWorkflowAsync(projection.Id,
            new WorkflowActionDto { Action = ApprovalAction.Submit, Comments = "Ready for review" });
        Assert.Equal(ProjectionStatus.Submitted, projection.Status);
        Assert.Equal("tester", projection.SubmittedBy);

        projection = await _host.Projections.ExecuteWorkflowAsync(projection.Id,
            new WorkflowActionDto { Action = ApprovalAction.Review });
        Assert.Equal(ProjectionStatus.UnderReview, projection.Status);

        projection = await _host.Projections.ExecuteWorkflowAsync(projection.Id,
            new WorkflowActionDto { Action = ApprovalAction.Approve, Comments = "Approved" });
        Assert.Equal(ProjectionStatus.Approved, projection.Status);
        Assert.NotNull(projection.ApprovedAtUtc);
        Assert.Equal(3, projection.ApprovalHistory.Count);
    }

    [Fact]
    public async Task An_illegal_transition_is_refused()
    {
        var projection = await CreateAsync();

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() =>
            _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve }));

        Assert.Equal("INVALID_TRANSITION", ex.Code);
    }

    [Fact]
    public async Task A_rejection_needs_a_reason()
    {
        var projection = await CreateAsync();
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() =>
            _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Reject }));

        Assert.Equal("REASON_REQUIRED", ex.Code);
    }

    [Fact]
    public async Task A_rejected_projection_becomes_editable_again()
    {
        var projection = await CreateAsync();
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        projection = await _host.Projections.ExecuteWorkflowAsync(projection.Id,
            new WorkflowActionDto { Action = ApprovalAction.Reject, Comments = "Area too ambitious" });

        Assert.Equal(ProjectionStatus.Rejected, projection.Status);
        Assert.Equal("Area too ambitious", projection.RejectionReason);

        // Editing is allowed again, then it can be resubmitted.
        await _host.Projections.UpdateHeaderAsync(projection.Id, new ProjectionUpdateDto
        {
            ProjectionDate = projection.ProjectionDate,
            PlanningStartDate = projection.PlanningStartDate,
            PlanningEndDate = projection.PlanningEndDate,
            Remarks = "Reduced scope"
        });

        projection = await _host.Projections.ExecuteWorkflowAsync(projection.Id,
            new WorkflowActionDto { Action = ApprovalAction.Submit });
        Assert.Equal(ProjectionStatus.Submitted, projection.Status);
    }

    [Fact]
    public async Task An_approved_projection_can_no_longer_be_edited()
    {
        var projection = await CreateAsync();
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() =>
            _host.Projections.AddLineAsync(projection.Id, new ProjectionLineUpsertDto
            {
                BlockId = _host.BlockBId,
                CaneVarietyId = _host.VarietyId,
                ProjectedPlantingAreaHa = 10m,
                PlannedPlantingStart = new DateOnly(2026, 5, 1),
                PlannedPlantingEnd = new DateOnly(2026, 5, 10)
            }));

        Assert.Equal("NOT_EDITABLE", ex.Code);
    }

    [Fact]
    public async Task Submitting_an_empty_projection_is_refused()
    {
        var request = _host.NewProjection();
        request.Lines.Clear();
        var projection = await _host.Projections.CreateAsync(request);

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() =>
            _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit }));

        Assert.Equal("NO_LINES", ex.Code);
    }

    [Fact]
    public async Task A_user_without_the_approve_policy_cannot_approve()
    {
        var projection = await CreateAsync();
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });

        _host.User.Roles = new[] { AppRoles.ReportViewer };

        await Assert.ThrowsAsync<ForbiddenException>(() =>
            _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve }));
    }

    // ------------------------------------------------------------- versioning

    [Fact]
    public async Task Revising_an_approved_plan_copies_it_and_freezes_the_previous_version()
    {
        var projection = await CreateAsync();
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });

        var revision = await _host.Projections.ReviseAsync(projection.Id,
            new ReviseProjectionDto { RevisionReason = "Rain delayed land preparation" });

        Assert.Equal(2, revision.Version);
        Assert.Equal(ProjectionStatus.Revised, revision.Status);
        Assert.Equal(projection.Id, revision.RevisedFromProjectionId);
        Assert.True(revision.IsCurrentVersion);
        Assert.Equal(projection.Lines.Count, revision.Lines.Count);
        Assert.Equal(projection.TotalProjectedAreaHa, revision.TotalProjectedAreaHa);

        var original = await _host.Projections.GetProjectionAsync(projection.Id);
        Assert.True(original.IsReadOnly);
        Assert.False(original.IsCurrentVersion);
        Assert.Equal(ProjectionStatus.Approved, original.Status);
    }

    [Fact]
    public async Task Only_an_approved_current_version_can_be_revised()
    {
        var projection = await CreateAsync();

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() =>
            _host.Projections.ReviseAsync(projection.Id, new ReviseProjectionDto { RevisionReason = "too early" }));

        Assert.Equal("NOT_APPROVED", ex.Code);
    }

    [Fact]
    public async Task Comparing_two_versions_lists_the_changed_fields()
    {
        var projection = await CreateAsync();
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });

        var revision = await _host.Projections.ReviseAsync(projection.Id,
            new ReviseProjectionDto { RevisionReason = "Reduce the area" });

        await _host.Projections.UpdateLineAsync(revision.Id, revision.Lines[0].Id, new ProjectionLineUpsertDto
        {
            BlockId = _host.BlockAId,
            CaneVarietyId = _host.VarietyId,
            ProjectedPlantingAreaHa = 60m,                       // was 80
            PlannedPlantingStart = new DateOnly(2026, 3, 2),
            PlannedPlantingEnd = new DateOnly(2026, 3, 20),
            ExpectedYieldPerHa = 90m,
            ExpectedLossPercent = 5m
        });

        var comparison = await _host.Projections.CompareVersionsAsync(projection.Id, revision.Id);

        Assert.Equal(1, comparison.FromVersion);
        Assert.Equal(2, comparison.ToVersion);
        Assert.Equal(80m, comparison.FromTotalAreaHa);
        Assert.Equal(60m, comparison.ToTotalAreaHa);
        Assert.Contains(comparison.Differences, d => d.Scope == "Line" && d.Field == "Projected area (ha)");
        Assert.Contains(comparison.Differences, d => d.Scope == "Header" && d.Field == "Total projected area (ha)");
    }

    [Fact]
    public async Task The_version_chain_lists_every_version_in_order()
    {
        var projection = await CreateAsync();
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });
        await _host.Projections.ReviseAsync(projection.Id, new ReviseProjectionDto { RevisionReason = "v2" });

        var versions = await _host.Projections.GetVersionsAsync(projection.Id);

        Assert.Equal(2, versions.Count);
        Assert.Equal(new[] { 1, 2 }, versions.Select(v => v.Version));
    }

    [Fact]
    public async Task The_workflow_transitions_are_written_to_the_audit_trail()
    {
        var projection = await CreateAsync();
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });

        var log = await _host.Audit.QueryAsync(new Contracts.Auditing.AuditLogQuery
        {
            TableName = "PlantingProjection",
            Action = AuditAction.Submit
        });

        Assert.NotEmpty(log.Items);
        Assert.Equal("tester", log.Items[0].UserName);
    }

    [Fact]
    public async Task The_projection_list_hides_superseded_versions_by_default()
    {
        var projection = await CreateAsync();
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });
        await _host.Projections.ReviseAsync(projection.Id, new ReviseProjectionDto { RevisionReason = "v2" });

        var current = await _host.Projections.GetProjectionsAsync(null, null, new QueryParameters());
        Assert.Single(current.Items);

        var all = await _host.Projections.GetProjectionsAsync(null, null, new QueryParameters { IncludeInactive = true });
        Assert.Equal(2, all.TotalCount);
    }
}
