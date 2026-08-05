using ErpS4.Domain;

namespace ErpS4.Application.Posting;

/// <summary>The central posting engine (design prompt, section 15).</summary>
public interface IPostingEngine
{
    /// <summary>
    /// Validates a draft and, when <paramref name="request"/> asks for it,
    /// commits it. Simulation runs every rule and reports the complete
    /// accounting impact without writing anything.
    /// </summary>
    Task<PostingResult> PostAsync(PostingRequest request, CancellationToken cancellationToken = default);

    /// <summary>
    /// Reverses a posted document with a new document that references it.
    /// Posted documents are never changed or deleted.
    /// </summary>
    Task<PostingResult> ReverseAsync(ReversalRequest request, CancellationToken cancellationToken = default);

    /// <summary>
    /// Validates a document, gives it a number and stores its lines, but leaves
    /// the ledger untouched: no open items, no controlling documents, no
    /// balances. This is what a document waiting for approval looks like - it
    /// can be displayed and audited, and it changes no figure until approved.
    /// </summary>
    Task<PostingResult> ParkAsync(PostingRequest request, CancellationToken cancellationToken = default);

    /// <summary>
    /// Completes a parked document once approval is in. Rules are checked again
    /// first: the period may have closed while the document sat in an inbox.
    /// </summary>
    Task<PostingResult> PostParkedAsync(
        string companyCode,
        short fiscalYear,
        string documentNumber,
        CancellationToken cancellationToken = default);
}

/// <param name="Draft">What the caller wants to post.</param>
/// <param name="SourceModule">FI, AR, AP, AA, CO, MM, SD, PY.</param>
/// <param name="TransactionCode">T-code the user came from, for the audit trail.</param>
/// <param name="IdempotencyKey">
/// Supplied by the caller. A retry with the same key returns the document the
/// first call created instead of posting a second one.
/// </param>
/// <param name="Simulate">Validate and calculate, but do not write.</param>
public sealed record PostingRequest(
    JournalEntryDraft Draft,
    string SourceModule = "FI",
    string? TransactionCode = null,
    Guid? IdempotencyKey = null,
    bool Simulate = false);

/// <param name="DocumentNumber">Document to reverse.</param>
/// <param name="FiscalYear">Fiscal year of that document.</param>
/// <param name="CompanyCode">Company code of that document.</param>
/// <param name="ReasonCode">Reversal reason.</param>
/// <param name="PostingDate">
/// Posting date of the reversal. Defaults to the original posting date, which
/// is only accepted while that period is still open.
/// </param>
public sealed record ReversalRequest(
    string DocumentNumber,
    short FiscalYear,
    string CompanyCode,
    string ReasonCode,
    DateOnly? PostingDate = null,
    Guid? IdempotencyKey = null);

/// <summary>
/// Outcome of a posting attempt. A failed attempt carries every rule that was
/// broken, not just the first, so the user fixes the document in one pass.
/// </summary>
public sealed record PostingResult
{
    private PostingResult()
    {
    }

    public bool IsSuccess { get; private init; }

    public string? DocumentNumber { get; private init; }

    public short FiscalYear { get; private init; }

    public byte FiscalPeriod { get; private init; }

    public string? Status { get; private init; }

    /// <summary>True when the request was a simulation and nothing was written.</summary>
    public bool WasSimulated { get; private init; }

    /// <summary>True when an earlier call with the same idempotency key produced this document.</summary>
    public bool WasAlreadyPosted { get; private init; }

    /// <summary>The lines as they would be stored, including derived amounts.</summary>
    public IReadOnlyList<SimulatedLine> Lines { get; private init; } = [];

    public IReadOnlyList<PostingError> Errors { get; private init; } = [];

    public static PostingResult Posted(
        string documentNumber,
        short fiscalYear,
        byte fiscalPeriod,
        string status,
        IReadOnlyList<SimulatedLine> lines,
        bool alreadyPosted = false) =>
        new()
        {
            IsSuccess = true,
            DocumentNumber = documentNumber,
            FiscalYear = fiscalYear,
            FiscalPeriod = fiscalPeriod,
            Status = status,
            Lines = lines,
            WasAlreadyPosted = alreadyPosted,
        };

    public static PostingResult Simulated(
        short fiscalYear,
        byte fiscalPeriod,
        IReadOnlyList<SimulatedLine> lines) =>
        new()
        {
            IsSuccess = true,
            FiscalYear = fiscalYear,
            FiscalPeriod = fiscalPeriod,
            Status = "Simulated",
            WasSimulated = true,
            Lines = lines,
        };

    public static PostingResult Rejected(IReadOnlyList<PostingError> errors) =>
        new() { IsSuccess = false, Errors = errors };
}

/// <summary>A line as the engine calculated it, shown by the simulation screen.</summary>
public sealed record SimulatedLine(
    int LineNumber,
    string PostingKey,
    string DebitCreditIndicator,
    string Account,
    string? AccountName,
    Money DocumentAmount,
    Money LocalAmount,
    Money? GroupAmount,
    string? BusinessPartner,
    string? CostCenter,
    string? ProfitCenter,
    string? Segment,
    string? TaxCode,
    string? Text);
