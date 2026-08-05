using ErpS4.Domain;

namespace ErpS4.Application.Workflow;

/// <summary>
/// Approval workflow (design prompt, section 17): rules by company code,
/// document type and amount; sequential and multi-level approval; maker-checker;
/// substitution; rejection and resubmission.
/// </summary>
public interface IWorkflowService
{
    /// <summary>
    /// Whether this document needs approval, and under which definition. A
    /// caller that gets <see cref="WorkflowDecision.NotRequired"/> posts
    /// directly.
    /// </summary>
    Task<WorkflowDecision> EvaluateAsync(
        WorkflowContext context,
        CancellationToken cancellationToken = default);

    /// <summary>Starts an approval process for a parked document.</summary>
    Task<SubmissionResult> SubmitAsync(
        SubmitForApprovalRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Approves one step. When the last step is approved the document is
    /// posted, so approval is what gives it its ledger effect.
    /// </summary>
    Task<DecisionResult> ApproveAsync(
        ApprovalRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>Rejects the document; it keeps its number and posts nothing.</summary>
    Task<DecisionResult> RejectAsync(
        ApprovalRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>Tasks waiting for a user, including those they substitute for.</summary>
    Task<IReadOnlyList<InboxItem>> GetInboxAsync(
        string userName,
        CancellationToken cancellationToken = default);
}

/// <param name="Amount">Absolute document amount the rules compare against.</param>
public sealed record WorkflowContext(
    string ObjectType,
    string CompanyCode,
    decimal Amount,
    string CurrencyCode)
{
    public string? DocumentType { get; init; }

    public string? CostCenter { get; init; }

    public string? SourceModule { get; init; }
}

/// <param name="IsRequired">False when no rule matches: the caller may post directly.</param>
/// <param name="WorkflowCode">Definition that matched.</param>
/// <param name="RuleName">Rule that matched, for the audit trail and the UI.</param>
public sealed record WorkflowDecision(
    bool IsRequired,
    string? WorkflowCode = null,
    string? RuleName = null,
    int StepCount = 0)
{
    public static WorkflowDecision NotRequired { get; } = new(false);
}

public sealed record SubmitForApprovalRequest(
    string ObjectType,
    long ObjectId,
    string CompanyCode,
    decimal Amount,
    string CurrencyCode)
{
    /// <summary>Document number, so the approver sees what they are approving.</summary>
    public string? ObjectDescription { get; init; }

    public string? DocumentNumber { get; init; }

    public short FiscalYear { get; init; }

    public string? DocumentType { get; init; }

    public string? CostCenter { get; init; }

    public string? Comment { get; init; }
}

public sealed record SubmissionResult
{
    public bool IsSuccess => Violations.All(v => !v.IsBlocking);

    /// <summary>False when no rule matched; nothing was created.</summary>
    public bool ApprovalRequired { get; init; }

    public string? InstanceNumber { get; init; }

    public int CurrentStep { get; init; }

    public IReadOnlyList<string> Approvers { get; init; } = [];

    public IReadOnlyList<RuleViolation> Violations { get; init; } = [];
}

public sealed record ApprovalRequest(string InstanceNumber, string? Comment = null);

public sealed record DecisionResult
{
    public bool IsSuccess => Violations.All(v => !v.IsBlocking);

    public required string InstanceNumber { get; init; }

    public string Status { get; init; } = string.Empty;

    /// <summary>True when this decision completed the process.</summary>
    public bool IsComplete { get; init; }

    /// <summary>Set when the final approval posted the document.</summary>
    public string? PostedDocumentNumber { get; init; }

    public int? NextStep { get; init; }

    public IReadOnlyList<RuleViolation> Violations { get; init; } = [];

    public IReadOnlyList<PostingError> PostingErrors { get; init; } = [];
}

public sealed record InboxItem(
    string InstanceNumber,
    string ObjectType,
    string? ObjectDescription,
    string CompanyCode,
    decimal? Amount,
    string? CurrencyCode,
    int StepNumber,
    string SubmittedBy,
    DateTime SubmittedAt,
    DateTime? DueAt,
    bool IsSubstitution);

public static class WorkflowErrorCodes
{
    public const string InstanceUnknown = "WF.INSTANCE_UNKNOWN";
    public const string InstanceNotPending = "WF.INSTANCE_NOT_PENDING";
    public const string NoTaskForUser = "WF.NO_TASK_FOR_USER";
    public const string SelfApprovalForbidden = "WF.SELF_APPROVAL_FORBIDDEN";
    public const string DefinitionMissing = "WF.DEFINITION_MISSING";
    public const string StepsMissing = "WF.STEPS_MISSING";
    public const string ApproverNotResolved = "WF.APPROVER_NOT_RESOLVED";
    public const string PostingFailed = "WF.POSTING_FAILED";
}
