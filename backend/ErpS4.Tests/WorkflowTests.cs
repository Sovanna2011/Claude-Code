using ErpS4.Application.Posting;
using ErpS4.Application.Workflow;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using ErpS4.Tests.TestDoubles;
using Microsoft.Extensions.Logging.Abstractions;
using Xunit;

namespace ErpS4.Tests;

/// <summary>
/// Approval rules: a document above the threshold parks instead of posting,
/// the submitter can never approve their own work, and only the last approval
/// gives the document its ledger effect.
/// </summary>
public sealed class WorkflowTests
{
    private static WorkflowService CreateService(PostingScenario scenario, string actingAs) =>
        new(scenario.Context,
            scenario.CreateEngineAs(actingAs),
            new FixedTenantProvider(1),
            new FixedCurrentUser(actingAs),
            scenario.Clock,
            NullLogger<WorkflowService>.Instance);

    private static PostingScenario Configure(PostingScenario scenario, int steps = 1) =>
        WorkflowScenario.Configure(scenario, steps);

    [Fact]
    public async Task A_document_below_the_threshold_needs_no_approval()
    {
        var scenario = new PostingScenario();
        Configure(scenario);

        var decision = await CreateService(scenario, "clerk").EvaluateAsync(
            new WorkflowContext("JournalEntry", PostingScenario.CompanyCodeKey, 500m, "USD"));

        Assert.False(decision.IsRequired);
    }

    [Fact]
    public async Task A_document_above_the_threshold_needs_approval()
    {
        var scenario = new PostingScenario();
        Configure(scenario);

        var decision = await CreateService(scenario, "clerk").EvaluateAsync(
            new WorkflowContext("JournalEntry", PostingScenario.CompanyCodeKey, 5_000m, "USD"));

        Assert.True(decision.IsRequired);
        Assert.Equal("JE_APPROVAL", decision.WorkflowCode);
        Assert.Equal(1, decision.StepCount);
    }

    [Fact]
    public async Task A_parked_document_has_no_ledger_effect_until_it_is_approved()
    {
        var scenario = new PostingScenario();
        Configure(scenario);

        var parked = await scenario.CreateEngineAs("clerk")
            .ParkAsync(new PostingRequest(PostingScenario.BalancedDraft(5_000m)));

        Assert.True(parked.IsSuccess);
        Assert.Equal("PendingApproval", parked.Status);

        // Header and lines exist and can be displayed...
        Assert.Single(scenario.Context.Set<JournalEntryHeader>());
        Assert.Equal(2, scenario.Context.Set<JournalEntryLine>().Count);

        // ...but nothing has moved in the ledger.
        Assert.Empty(scenario.Context.Set<AccountBalance>());
        Assert.Empty(scenario.Context.Set<OpenItem>());
    }

    [Fact]
    public async Task The_submitter_cannot_approve_their_own_document()
    {
        var scenario = new PostingScenario();
        Configure(scenario);
        var submission = await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);

        var result = await CreateService(scenario, "clerk")
            .ApproveAsync(new ApprovalRequest(submission.InstanceNumber!));

        Assert.False(result.IsSuccess);
        Assert.Contains(
            result.Violations, v => v.Code == WorkflowErrorCodes.SelfApprovalForbidden);
    }

    [Fact]
    public async Task The_submitter_never_receives_a_task_for_their_own_document()
    {
        var scenario = new PostingScenario();
        Configure(scenario);

        // The clerk holds the approver role too, but submitted this document.
        var submission = await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);

        Assert.DoesNotContain("clerk", submission.Approvers);
        Assert.Contains("approver", submission.Approvers);
    }

    [Fact]
    public async Task The_final_approval_posts_the_document()
    {
        var scenario = new PostingScenario();
        Configure(scenario);
        var submission = await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);

        var result = await CreateService(scenario, "approver")
            .ApproveAsync(new ApprovalRequest(submission.InstanceNumber!, "Checked the invoice"));

        Assert.True(result.IsSuccess);
        Assert.True(result.IsComplete);
        Assert.Equal("Approved", result.Status);
        Assert.NotNull(result.PostedDocumentNumber);

        var header = Assert.Single(scenario.Context.Set<JournalEntryHeader>());
        Assert.Equal("Posted", header.Status);
        Assert.Equal("approver", header.PostedBy);

        // Now the ledger has moved.
        Assert.NotEmpty(scenario.Context.Set<AccountBalance>());
        Assert.Equal(0m, scenario.Context.Set<AccountBalance>().Sum(b => b.PeriodBalance));
    }

    [Fact]
    public async Task A_rejection_leaves_the_ledger_untouched()
    {
        var scenario = new PostingScenario();
        Configure(scenario);
        var submission = await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);

        var result = await CreateService(scenario, "approver")
            .RejectAsync(new ApprovalRequest(submission.InstanceNumber!, "Wrong cost centre"));

        Assert.True(result.IsSuccess);
        Assert.Equal("Rejected", result.Status);

        var header = Assert.Single(scenario.Context.Set<JournalEntryHeader>());
        Assert.Equal("Rejected", header.Status);
        Assert.Empty(scenario.Context.Set<AccountBalance>());
    }

    [Fact]
    public async Task A_two_step_workflow_posts_only_after_the_second_approval()
    {
        var scenario = new PostingScenario();
        Configure(scenario, steps: 2);
        var submission = await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);

        var first = await CreateService(scenario, "approver")
            .ApproveAsync(new ApprovalRequest(submission.InstanceNumber!));

        Assert.True(first.IsSuccess);
        Assert.False(first.IsComplete);
        Assert.Equal(2, first.NextStep);
        Assert.Equal("PendingApproval",
            scenario.Context.Set<JournalEntryHeader>().Single().Status);
        Assert.Empty(scenario.Context.Set<AccountBalance>());

        var second = await CreateService(scenario, "director")
            .ApproveAsync(new ApprovalRequest(submission.InstanceNumber!));

        Assert.True(second.IsComplete);
        Assert.Equal("Posted", scenario.Context.Set<JournalEntryHeader>().Single().Status);
    }

    [Fact]
    public async Task A_user_without_a_task_cannot_decide()
    {
        var scenario = new PostingScenario();
        Configure(scenario);
        var submission = await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);

        var result = await CreateService(scenario, "stranger")
            .ApproveAsync(new ApprovalRequest(submission.InstanceNumber!));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Violations, v => v.Code == WorkflowErrorCodes.NoTaskForUser);
    }

    [Fact]
    public async Task A_substitute_can_decide_and_the_task_records_who_acted()
    {
        var scenario = new PostingScenario();
        Configure(scenario);
        WorkflowScenario.AddSubstitution(scenario, absent: "approver", substitute: "stranger");
        var submission = await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);

        var result = await CreateService(scenario, "stranger")
            .ApproveAsync(new ApprovalRequest(submission.InstanceNumber!));

        Assert.True(result.IsSuccess);
        var task = scenario.Context.Set<WorkflowTask>().Single(t => t.Status == "Approved");
        Assert.Contains("substitute stranger", task.DelegationReason!, StringComparison.Ordinal);
    }

    [Fact]
    public async Task The_inbox_shows_pending_tasks_and_marks_substitutions()
    {
        var scenario = new PostingScenario();
        Configure(scenario);
        WorkflowScenario.AddSubstitution(scenario, absent: "approver", substitute: "stranger");
        await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);

        var ownInbox = await CreateService(scenario, "approver").GetInboxAsync("approver");
        var substituteInbox = await CreateService(scenario, "stranger").GetInboxAsync("stranger");

        Assert.Single(ownInbox);
        Assert.False(ownInbox[0].IsSubstitution);
        Assert.Single(substituteInbox);
        Assert.True(substituteInbox[0].IsSubstitution);
    }

    [Fact]
    public async Task A_decision_cannot_be_taken_twice()
    {
        var scenario = new PostingScenario();
        Configure(scenario);
        var submission = await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);
        var service = CreateService(scenario, "approver");

        await service.ApproveAsync(new ApprovalRequest(submission.InstanceNumber!));
        var second = await service.ApproveAsync(new ApprovalRequest(submission.InstanceNumber!));

        Assert.False(second.IsSuccess);
        Assert.Contains(second.Violations, v => v.Code == WorkflowErrorCodes.InstanceNotPending);
    }

    [Fact]
    public async Task Approval_after_the_period_closes_escalates_instead_of_posting()
    {
        var scenario = new PostingScenario();
        Configure(scenario);
        var submission = await WorkflowScenario.ParkAndSubmitAsync(scenario, "clerk", 5_000m);

        // The month closes while the document sits in the inbox.
        scenario.Period.PeriodStatus = "Closed";

        var result = await CreateService(scenario, "approver")
            .ApproveAsync(new ApprovalRequest(submission.InstanceNumber!));

        Assert.False(result.IsSuccess);
        Assert.Equal("Escalated", result.Status);
        Assert.Contains(result.PostingErrors, e => e.Code == PostingErrorCodes.PeriodClosed);
        Assert.Empty(scenario.Context.Set<AccountBalance>());
    }
}
