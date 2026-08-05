using ErpS4.Application.Posting;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;

namespace ErpS4.Application.Workflow;

/// <inheritdoc />
public sealed class WorkflowService(
    IErpDataContext context,
    IPostingEngine postingEngine,
    ITenantProvider tenantProvider,
    ICurrentUser currentUser,
    TimeProvider timeProvider,
    ILogger<WorkflowService> logger) : IWorkflowService
{
    private int TenantId => tenantProvider.TenantId;

    public async Task<WorkflowDecision> EvaluateAsync(
        WorkflowContext workflowContext,
        CancellationToken cancellationToken = default)
    {
        var match = await MatchAsync(workflowContext, cancellationToken);

        if (match is null)
        {
            return WorkflowDecision.NotRequired;
        }

        var stepCount = await context.Query<WorkflowStep>()
            .CountAsync(
                s => s.TenantId == TenantId && s.WorkflowDefinitionId == match.Definition.Id,
                cancellationToken);

        return new WorkflowDecision(
            true, match.Definition.WorkflowCode, match.Rule.Name, stepCount);
    }

    public async Task<SubmissionResult> SubmitAsync(
        SubmitForApprovalRequest request,
        CancellationToken cancellationToken = default)
    {
        var workflowContext = new WorkflowContext(
            request.ObjectType, request.CompanyCode, request.Amount, request.CurrencyCode)
        {
            DocumentType = request.DocumentType,
            CostCenter = request.CostCenter,
        };

        var match = await MatchAsync(workflowContext, cancellationToken);
        if (match is null)
        {
            // No rule matched: nothing is created, and the caller is free to
            // post. Approval that nobody configured is not approval.
            return new SubmissionResult { ApprovalRequired = false };
        }

        var steps = await context.Query<WorkflowStep>()
            .AsNoTracking()
            .Where(s => s.TenantId == TenantId && s.WorkflowDefinitionId == match.Definition.Id)
            .OrderBy(s => s.StepNumber)
            .ToListAsync(cancellationToken);

        if (steps.Count == 0)
        {
            return new SubmissionResult
            {
                ApprovalRequired = true,
                Violations = [
                    new RuleViolation(
                        WorkflowErrorCodes.StepsMissing,
                        $"Workflow {match.Definition.WorkflowCode} has no steps, so nobody " +
                        "would ever see this document."),
                ],
            };
        }

        var now = timeProvider.GetUtcNow().UtcDateTime;
        var user = currentUser.UserName;
        var companyCode = await context.Query<CompanyCode>()
            .AsNoTracking()
            .FirstAsync(c => c.TenantId == TenantId && c.CompanyCodeKey == request.CompanyCode,
                cancellationToken);

        var instanceNumber = $"WF-{request.FiscalYear:0000}-{Guid.NewGuid().ToString("N")[..10].ToUpperInvariant()}";

        var instance = new WorkflowInstance
        {
            TenantId = TenantId,
            InstanceNumber = instanceNumber,
            WorkflowDefinitionId = match.Definition.Id,
            ObjectType = request.ObjectType,
            ObjectId = request.ObjectId,
            ObjectDescription = request.ObjectDescription ?? request.DocumentNumber,
            CompanyCodeId = companyCode.Id,
            CurrencyCode = request.CurrencyCode,
            Amount = request.Amount,
            CurrentStepNumber = steps[0].StepNumber,
            Status = "PendingApproval",
            SubmittedBy = user,
            SubmittedAt = now,
            ResubmissionCount = 0,
            CreatedAt = now,
            CreatedBy = user,
        };

        context.Add(instance);
        await context.SaveChangesAsync(cancellationToken);

        var approvers = await CreateTasksAsync(
            instance, match.Definition, steps[0], now, user, cancellationToken);

        AddHistory(instance, "Submitted", steps[0].StepNumber, null, "PendingApproval",
            request.Comment, now, user);

        await context.SaveChangesAsync(cancellationToken);

        logger.LogInformation(
            "Workflow {Instance} started for {ObjectType} {Description} with {Count} approvers",
            instanceNumber, request.ObjectType, instance.ObjectDescription, approvers.Count);

        return new SubmissionResult
        {
            ApprovalRequired = true,
            InstanceNumber = instanceNumber,
            CurrentStep = steps[0].StepNumber,
            Approvers = approvers,
            Violations = approvers.Count == 0
                ? [
                    new RuleViolation(
                        WorkflowErrorCodes.ApproverNotResolved,
                        "No approver could be resolved for the first step; the document would " +
                        "sit unapproved forever.",
                        nameof(WorkflowStep.ApproverDeterminationType)),
                ]
                : [],
        };
    }

    public Task<DecisionResult> ApproveAsync(
        ApprovalRequest request,
        CancellationToken cancellationToken = default) =>
        DecideAsync(request, approve: true, cancellationToken);

    public Task<DecisionResult> RejectAsync(
        ApprovalRequest request,
        CancellationToken cancellationToken = default) =>
        DecideAsync(request, approve: false, cancellationToken);

    public async Task<IReadOnlyList<InboxItem>> GetInboxAsync(
        string userName,
        CancellationToken cancellationToken = default)
    {
        var today = DateOnly.FromDateTime(timeProvider.GetUtcNow().UtcDateTime);

        // A substitute sees the tasks of whoever they stand in for, marked as
        // such: acting for someone else is never invisible.
        var substitutedUserIds = await (
            from substitution in context.Query<UserSubstitution>().AsNoTracking()
            join substitute in context.Query<User>() on substitution.SubstituteUserId equals substitute.Id
            where substitution.TenantId == TenantId
                  && substitute.UserName == userName
                  && substitution.IsActive
                  && substitution.ValidFrom <= today
                  && substitution.ValidTo >= today
            select substitution.UserId
        ).ToListAsync(cancellationToken);

        var items = await (
            from task in context.Query<WorkflowTask>().AsNoTracking()
            join instance in context.Query<WorkflowInstance>()
                on task.WorkflowInstanceId equals instance.Id
            join assignee in context.Query<User>() on task.AssignedUserId equals assignee.Id
            join company in context.Query<CompanyCode>()
                on instance.CompanyCodeId equals company.Id
            where task.TenantId == TenantId
                  && task.Status == "Pending"
                  && (assignee.UserName == userName || substitutedUserIds.Contains(assignee.Id))
            orderby task.AssignedAt
            select new InboxItem(
                instance.InstanceNumber,
                instance.ObjectType,
                instance.ObjectDescription,
                company.CompanyCodeKey,
                instance.Amount,
                instance.CurrencyCode,
                task.StepNumber,
                instance.SubmittedBy,
                instance.SubmittedAt,
                task.DueAt,
                assignee.UserName != userName)
        ).ToListAsync(cancellationToken);

        return items;
    }

    private async Task<DecisionResult> DecideAsync(
        ApprovalRequest request,
        bool approve,
        CancellationToken cancellationToken)
    {
        var instance = await context.Query<WorkflowInstance>()
            .FirstOrDefaultAsync(
                i => i.TenantId == TenantId && i.InstanceNumber == request.InstanceNumber,
                cancellationToken);

        if (instance is null)
        {
            return Failed(request.InstanceNumber, WorkflowErrorCodes.InstanceUnknown,
                $"Workflow {request.InstanceNumber} does not exist.");
        }

        if (instance.Status is not ("PendingApproval" or "Submitted" or "Escalated"))
        {
            return Failed(request.InstanceNumber, WorkflowErrorCodes.InstanceNotPending,
                $"Workflow {request.InstanceNumber} is {instance.Status}.");
        }

        var definition = await context.Query<WorkflowDefinition>()
            .AsNoTracking()
            .FirstAsync(d => d.Id == instance.WorkflowDefinitionId, cancellationToken);

        var user = currentUser.UserName;

        // Maker-checker: whoever submitted a document may not be the one who
        // approves it, however senior they are.
        if (definition.IsMakerCheckerEnforced
            && string.Equals(instance.SubmittedBy, user, StringComparison.OrdinalIgnoreCase))
        {
            return Failed(request.InstanceNumber, WorkflowErrorCodes.SelfApprovalForbidden,
                "You submitted this document, so you cannot decide on it.");
        }

        var (task, actingFor) = await FindTaskAsync(instance, user, cancellationToken);
        if (task is null)
        {
            return Failed(request.InstanceNumber, WorkflowErrorCodes.NoTaskForUser,
                $"{user} has no open task on workflow {request.InstanceNumber}.");
        }

        var now = timeProvider.GetUtcNow().UtcDateTime;
        var deciderId = await context.Query<User>()
            .Where(u => u.TenantId == TenantId && u.UserName == user)
            .Select(u => (long?)u.Id)
            .FirstOrDefaultAsync(cancellationToken);

        task.Status = approve ? "Approved" : "Rejected";
        task.Decision = approve ? "Approve" : "Reject";
        task.DecidedAt = now;
        task.DecidedByUserId = deciderId;
        task.Comment = request.Comment;
        task.ModifiedAt = now;
        task.ModifiedBy = user;

        if (actingFor is not null)
        {
            task.DelegationReason = $"Decided by substitute {user} for {actingFor}.";
        }

        AddHistory(instance, approve ? "Approved" : "Rejected", task.StepNumber,
            instance.Status, approve ? "Approved" : "Rejected", request.Comment, now, user);

        if (!approve)
        {
            instance.Status = "Rejected";
            instance.FinalDecision = "Rejected";
            instance.RejectionReason = request.Comment;
            instance.CompletedAt = now;
            await MarkDocumentAsync(instance, "Rejected", now, user, cancellationToken);
            await context.SaveChangesAsync(cancellationToken);

            logger.LogInformation(
                "Workflow {Instance} rejected by {User}", request.InstanceNumber, user);

            return new DecisionResult
            {
                InstanceNumber = request.InstanceNumber,
                Status = "Rejected",
                IsComplete = true,
            };
        }

        // Sibling tasks on the same step: the step is done when enough of them
        // have approved.
        var stepTasks = await context.Query<WorkflowTask>()
            .Where(t => t.TenantId == TenantId
                        && t.WorkflowInstanceId == instance.Id
                        && t.StepNumber == task.StepNumber)
            .ToListAsync(cancellationToken);

        var step = await context.Query<WorkflowStep>()
            .AsNoTracking()
            .FirstAsync(
                s => s.WorkflowDefinitionId == instance.WorkflowDefinitionId
                     && s.StepNumber == task.StepNumber,
                cancellationToken);

        var approvals = stepTasks.Count(t => t.Status == "Approved");
        if (approvals < step.RequiredApprovals)
        {
            await context.SaveChangesAsync(cancellationToken);

            return new DecisionResult
            {
                InstanceNumber = request.InstanceNumber,
                Status = "PendingApproval",
                NextStep = task.StepNumber,
            };
        }

        // Withdraw the other open tasks on this step: the decision is made.
        foreach (var sibling in stepTasks.Where(t => t.Status == "Pending"))
        {
            sibling.Status = "Withdrawn";
            sibling.ModifiedAt = now;
            sibling.ModifiedBy = user;
        }

        var nextStep = await context.Query<WorkflowStep>()
            .AsNoTracking()
            .Where(s => s.WorkflowDefinitionId == instance.WorkflowDefinitionId
                        && s.StepNumber > task.StepNumber)
            .OrderBy(s => s.StepNumber)
            .FirstOrDefaultAsync(cancellationToken);

        if (nextStep is not null)
        {
            instance.CurrentStepNumber = nextStep.StepNumber;
            var approvers = await CreateTasksAsync(
                instance, definition, nextStep, now, user, cancellationToken);

            AddHistory(instance, "Assigned", nextStep.StepNumber, "PendingApproval",
                "PendingApproval", null, now, user);

            await context.SaveChangesAsync(cancellationToken);

            logger.LogInformation(
                "Workflow {Instance} advanced to step {Step} with {Count} approvers",
                request.InstanceNumber, nextStep.StepNumber, approvers.Count);

            return new DecisionResult
            {
                InstanceNumber = request.InstanceNumber,
                Status = "PendingApproval",
                NextStep = nextStep.StepNumber,
            };
        }

        instance.Status = "Approved";
        instance.FinalDecision = "Approved";
        instance.CompletedAt = now;
        AddHistory(instance, "Completed", task.StepNumber, "PendingApproval", "Approved",
            null, now, user);

        await context.SaveChangesAsync(cancellationToken);

        // The last approval is what gives the document its ledger effect.
        var posted = await CompleteDocumentAsync(instance, now, user, cancellationToken);

        return posted;
    }

    /// <summary>
    /// Posts the parked document the instance was created for. Any other object
    /// type simply completes: approving a master data change does not post
    /// anything.
    /// </summary>
    private async Task<DecisionResult> CompleteDocumentAsync(
        WorkflowInstance instance,
        DateTime now,
        string user,
        CancellationToken cancellationToken)
    {
        if (instance.ObjectType != "JournalEntry")
        {
            return new DecisionResult
            {
                InstanceNumber = instance.InstanceNumber,
                Status = "Approved",
                IsComplete = true,
            };
        }

        var header = await context.Query<JournalEntryHeader>()
            .AsNoTracking()
            .FirstOrDefaultAsync(h => h.Id == instance.ObjectId, cancellationToken);

        if (header is null)
        {
            return new DecisionResult
            {
                InstanceNumber = instance.InstanceNumber,
                Status = "Approved",
                IsComplete = true,
                Violations = [
                    new RuleViolation(
                        WorkflowErrorCodes.PostingFailed,
                        "The approved document no longer exists."),
                ],
            };
        }

        var companyCode = await context.Query<CompanyCode>()
            .AsNoTracking()
            .Where(c => c.Id == header.CompanyCodeId)
            .Select(c => c.CompanyCodeKey)
            .FirstAsync(cancellationToken);

        var result = await postingEngine.PostParkedAsync(
            companyCode, header.FiscalYear, header.DocumentNumber, cancellationToken);

        if (!result.IsSuccess)
        {
            // The document stays approved but unposted: the period closed under
            // it, or a rule now refuses it. Someone has to look.
            instance.Status = "Escalated";
            await context.SaveChangesAsync(cancellationToken);

            return new DecisionResult
            {
                InstanceNumber = instance.InstanceNumber,
                Status = "Escalated",
                IsComplete = false,
                PostingErrors = result.Errors,
                Violations = [
                    new RuleViolation(
                        WorkflowErrorCodes.PostingFailed,
                        "Approved, but the document could not be posted; see the posting errors."),
                ],
            };
        }

        logger.LogInformation(
            "Workflow {Instance} approved and posted document {Document}",
            instance.InstanceNumber, header.DocumentNumber);

        return new DecisionResult
        {
            InstanceNumber = instance.InstanceNumber,
            Status = "Approved",
            IsComplete = true,
            PostedDocumentNumber = header.DocumentNumber,
        };
    }

    /// <summary>The first rule that matches, in the order the rules define.</summary>
    private async Task<RuleMatch?> MatchAsync(
        WorkflowContext workflowContext,
        CancellationToken cancellationToken)
    {
        var candidates = await (
            from rule in context.Query<WorkflowRule>().AsNoTracking()
            join definition in context.Query<WorkflowDefinition>()
                on rule.WorkflowDefinitionId equals definition.Id
            where rule.TenantId == TenantId
                  && rule.IsActive
                  && definition.IsActive
                  && definition.ObjectType == workflowContext.ObjectType
            orderby rule.RuleSequence
            select new { rule, definition }
        ).ToListAsync(cancellationToken);

        if (candidates.Count == 0)
        {
            return null;
        }

        var companyCodeId = await context.Query<CompanyCode>()
            .AsNoTracking()
            .Where(c => c.TenantId == TenantId && c.CompanyCodeKey == workflowContext.CompanyCode)
            .Select(c => (long?)c.Id)
            .FirstOrDefaultAsync(cancellationToken);

        var amount = Math.Abs(workflowContext.Amount);

        foreach (var candidate in candidates)
        {
            var rule = candidate.rule;

            if (rule.CompanyCodeId is not null && rule.CompanyCodeId != companyCodeId)
            {
                continue;
            }

            if (rule.MinimumAmount is { } minimum && amount < minimum)
            {
                continue;
            }

            if (rule.MaximumAmount is { } maximum && amount > maximum)
            {
                continue;
            }

            if (rule.SourceModule is not null && rule.SourceModule != workflowContext.SourceModule)
            {
                continue;
            }

            return new RuleMatch(rule, candidate.definition);
        }

        return null;
    }

    /// <summary>
    /// Creates one task per resolved approver. A role step gives every holder
    /// of the role a task; the first decision withdraws the rest.
    /// </summary>
    private async Task<List<string>> CreateTasksAsync(
        WorkflowInstance instance,
        WorkflowDefinition definition,
        WorkflowStep step,
        DateTime now,
        string user,
        CancellationToken cancellationToken)
    {
        var today = DateOnly.FromDateTime(now);
        List<User> approvers;

        if (step.ApproverDeterminationType == "User" && step.ApproverUserId is { } userId)
        {
            approvers = await context.Query<User>()
                .AsNoTracking()
                .Where(u => u.Id == userId && u.Status == "Active")
                .ToListAsync(cancellationToken);
        }
        else if (step.ApproverRoleId is { } roleId)
        {
            approvers = await (
                from assignment in context.Query<UserRole>().AsNoTracking()
                join candidate in context.Query<User>() on assignment.UserId equals candidate.Id
                where assignment.TenantId == TenantId
                      && assignment.RoleId == roleId
                      && assignment.ValidFrom <= today && assignment.ValidTo >= today
                      && candidate.Status == "Active" && !candidate.IsLocked
                select candidate
            ).ToListAsync(cancellationToken);
        }
        else
        {
            approvers = [];
        }

        // Maker-checker again, one layer earlier: never put the document in the
        // submitter's own inbox.
        if (definition.IsMakerCheckerEnforced)
        {
            approvers = approvers
                .Where(a => !string.Equals(a.UserName, instance.SubmittedBy,
                    StringComparison.OrdinalIgnoreCase))
                .ToList();
        }

        foreach (var approver in approvers)
        {
            context.Add(new WorkflowTask
            {
                TenantId = TenantId,
                WorkflowInstanceId = instance.Id,
                StepNumber = step.StepNumber,
                AssignedUserId = approver.Id,
                AssignedRoleId = step.ApproverRoleId,
                Status = "Pending",
                AssignedAt = now,
                DueAt = step.EscalationHours is { } hours ? now.AddHours(hours) : null,
                ReminderCount = 0,
                IsReadOnly = false,
                CreatedAt = now,
                CreatedBy = user,
            });

            context.Add(new Notification
            {
                TenantId = TenantId,
                RecipientUserId = approver.Id,
                NotificationType = "ApprovalRequest",
                Channel = "InApp",
                Subject = $"Approval needed: {instance.ObjectDescription}",
                Body = $"{instance.SubmittedBy} submitted {instance.ObjectDescription} " +
                       $"for {instance.Amount} {instance.CurrencyCode}.",
                ObjectType = instance.ObjectType,
                ObjectId = instance.ObjectId,
                NavigationUrl = $"/approvals/{instance.InstanceNumber}",
                Priority = "Normal",
                Status = "Queued",
                RetryCount = 0,
            });
        }

        return approvers.Select(a => a.UserName).ToList();
    }

    /// <summary>Finds the caller's task, or one they stand in for.</summary>
    private async Task<(WorkflowTask? Task, string? ActingFor)> FindTaskAsync(
        WorkflowInstance instance,
        string userName,
        CancellationToken cancellationToken)
    {
        var own = await (
            from task in context.Query<WorkflowTask>()
            join assignee in context.Query<User>() on task.AssignedUserId equals assignee.Id
            where task.TenantId == TenantId
                  && task.WorkflowInstanceId == instance.Id
                  && task.StepNumber == instance.CurrentStepNumber
                  && task.Status == "Pending"
                  && assignee.UserName == userName
            select task
        ).FirstOrDefaultAsync(cancellationToken);

        if (own is not null)
        {
            return (own, null);
        }

        var today = DateOnly.FromDateTime(timeProvider.GetUtcNow().UtcDateTime);

        var substituted = await (
            from task in context.Query<WorkflowTask>()
            join assignee in context.Query<User>() on task.AssignedUserId equals assignee.Id
            join substitution in context.Query<UserSubstitution>()
                on assignee.Id equals substitution.UserId
            join substitute in context.Query<User>()
                on substitution.SubstituteUserId equals substitute.Id
            where task.TenantId == TenantId
                  && task.WorkflowInstanceId == instance.Id
                  && task.StepNumber == instance.CurrentStepNumber
                  && task.Status == "Pending"
                  && substitute.UserName == userName
                  && substitution.IsActive
                  && substitution.ValidFrom <= today && substitution.ValidTo >= today
            select new { task, assignee.UserName }
        ).FirstOrDefaultAsync(cancellationToken);

        return substituted is null ? (null, null) : (substituted.task, substituted.UserName);
    }

    private async Task MarkDocumentAsync(
        WorkflowInstance instance,
        string status,
        DateTime now,
        string user,
        CancellationToken cancellationToken)
    {
        if (instance.ObjectType != "JournalEntry")
        {
            return;
        }

        var header = await context.Query<JournalEntryHeader>()
            .FirstOrDefaultAsync(h => h.Id == instance.ObjectId, cancellationToken);

        if (header is null)
        {
            return;
        }

        header.Status = status;
        header.ModifiedAt = now;
        header.ModifiedBy = user;
    }

    private void AddHistory(
        WorkflowInstance instance,
        string eventType,
        int? stepNumber,
        string? fromStatus,
        string? toStatus,
        string? comment,
        DateTime now,
        string user)
    {
        context.Add(new WorkflowHistory
        {
            TenantId = TenantId,
            WorkflowInstanceId = instance.Id,
            EventSequence = 0,
            EventType = eventType,
            StepNumber = stepNumber,
            PerformedAt = now,
            FromStatus = fromStatus,
            ToStatus = toStatus,
            Comment = comment,
        });
    }

    private static DecisionResult Failed(string instanceNumber, string code, string message) =>
        new()
        {
            InstanceNumber = instanceNumber,
            Status = "Unchanged",
            Violations = [new RuleViolation(code, message)],
        };

    private sealed record RuleMatch(WorkflowRule Rule, WorkflowDefinition Definition);
}
