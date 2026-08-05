using ErpS4.Domain;

namespace ErpS4.Application.BusinessPartners;

/// <summary>
/// Business partner role synchronisation (design prompt, section 6; T-codes
/// BP, BP_ROLE, BP_SYNC, BP_CHECK).
/// </summary>
/// <remarks>
/// The rule the whole model rests on: a customer and a supplier are roles on
/// one partner, never two identities. Assigning a role creates or updates the
/// data that role needs and touches nothing about the general identity, so the
/// same partner can be both without its name, address or tax number being
/// stored twice.
/// </remarks>
public interface IBusinessPartnerSyncService
{
    /// <summary>Assigns a role and creates the data that role requires.</summary>
    Task<RoleAssignmentResult> AssignRoleAsync(
        AssignRoleRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Re-runs synchronisation for every role the partner holds (BP_SYNC).
    /// Used after an import, or to repair a partner whose role data was created
    /// before a rule changed.
    /// </summary>
    Task<SynchronizationResult> SynchronizeAsync(
        string partnerNumber,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Reports inconsistencies without changing anything (BP_CHECK).
    /// </summary>
    Task<ConsistencyReport> CheckAsync(
        string partnerNumber,
        CancellationToken cancellationToken = default);
}

/// <param name="PartnerNumber">Partner that receives the role.</param>
/// <param name="RoleCode">Role to assign, e.g. <c>FLCU00</c>.</param>
/// <param name="CompanyCode">
/// Required for the FI roles: the company code segment carries the
/// reconciliation account, payment terms and dunning data.
/// </param>
/// <param name="ReconciliationAccount">
/// Reconciliation account for that company code segment. Must be flagged as a
/// reconciliation account of the matching type.
/// </param>
/// <param name="AccountGroup">Customer or vendor account group.</param>
public sealed record AssignRoleRequest(
    string PartnerNumber,
    string RoleCode,
    string? CompanyCode = null,
    string? ReconciliationAccount = null,
    string? AccountGroup = null,
    string? PaymentTerms = null,
    DateOnly? ValidFrom = null,
    DateOnly? ValidTo = null);

public sealed record RoleAssignmentResult
{
    public bool IsSuccess => Violations.All(v => !v.IsBlocking);

    public required string PartnerNumber { get; init; }

    public required string RoleCode { get; init; }

    /// <summary>True when the partner already held this role - the call was a no-op.</summary>
    public bool WasAlreadyAssigned { get; init; }

    /// <summary>Role-specific records this call created.</summary>
    public IReadOnlyList<string> Created { get; init; } = [];

    /// <summary>`Completed` or `Pending` when data is still missing.</summary>
    public string SyncStatus { get; init; } = "Completed";

    public IReadOnlyList<RuleViolation> Violations { get; init; } = [];
}

public sealed record SynchronizationResult
{
    public required string PartnerNumber { get; init; }

    public required IReadOnlyList<RoleAssignmentResult> Roles { get; init; }

    public bool IsSuccess => Roles.All(r => r.IsSuccess);
}

/// <summary>What BP_CHECK found. An empty report is a healthy partner.</summary>
public sealed record ConsistencyReport
{
    public required string PartnerNumber { get; init; }

    public required IReadOnlyList<RuleViolation> Findings { get; init; }

    public bool IsConsistent => Findings.Count == 0;

    public int ErrorCount => Findings.Count(f => f.Severity == ViolationSeverity.Error);

    public int WarningCount => Findings.Count(f => f.Severity == ViolationSeverity.Warning);
}

/// <summary>Stable codes for business partner rules.</summary>
public static class BusinessPartnerErrorCodes
{
    public const string PartnerUnknown = "BP.PARTNER_UNKNOWN";
    public const string PartnerNotActive = "BP.PARTNER_NOT_ACTIVE";
    public const string RoleUnknown = "BP.ROLE_UNKNOWN";
    public const string CompanyCodeRequired = "BP.COMPANY_CODE_REQUIRED";
    public const string CompanyCodeUnknown = "BP.COMPANY_CODE_UNKNOWN";
    public const string ReconciliationAccountRequired = "BP.RECON_ACCOUNT_REQUIRED";
    public const string ReconciliationAccountUnknown = "BP.RECON_ACCOUNT_UNKNOWN";
    public const string ReconciliationAccountWrongType = "BP.RECON_ACCOUNT_WRONG_TYPE";
    public const string AccountGroupUnknown = "BP.ACCOUNT_GROUP_UNKNOWN";
    public const string RoleDataMissing = "BP.ROLE_DATA_MISSING";
    public const string RoleMissingForData = "BP.ROLE_MISSING_FOR_DATA";
    public const string SegmentWithoutRole = "BP.SEGMENT_WITHOUT_ROLE";
    public const string NumberMismatch = "BP.NUMBER_MISMATCH";
    public const string BlockInconsistent = "BP.BLOCK_INCONSISTENT";
}
