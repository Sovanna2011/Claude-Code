using ErpS4.Domain;

namespace ErpS4.Application.Assets;

/// <summary>
/// Asset accounting (design prompt, section 7; T-codes F-90, ABAVN, AFAB).
/// </summary>
/// <remarks>
/// Every asset movement posts through the central engine, so the asset register
/// and the general ledger cannot disagree: the same document that updates
/// fin.AssetTransaction updates fin.JournalEntryLine.
/// </remarks>
public interface IAssetService
{
    /// <summary>Capitalises an asset, from a supplier invoice or a clearing account.</summary>
    Task<AssetPostingResult> AcquireAsync(
        AssetAcquisitionRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>Retires an asset, with or without revenue, and books the gain or loss.</summary>
    Task<AssetPostingResult> RetireAsync(
        AssetRetirementRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Plans depreciation for a period without posting it - the preview a
    /// controller checks before the run.
    /// </summary>
    Task<DepreciationPlan> PlanDepreciationAsync(
        DepreciationRunRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>Runs and posts depreciation for a period (AFAB).</summary>
    Task<DepreciationRunResult> RunDepreciationAsync(
        DepreciationRunRequest request,
        CancellationToken cancellationToken = default);
}

/// <param name="OffsettingAccount">
/// Where the credit goes: the supplier reconciliation account for a vendor
/// acquisition, or a clearing account when the invoice is posted separately.
/// </param>
public sealed record AssetAcquisitionRequest(
    string CompanyCode,
    string AssetNumber,
    int AssetSubNumber,
    decimal Amount,
    string CurrencyCode,
    DateOnly PostingDate,
    string OffsettingAccount)
{
    public string? VendorPartnerNumber { get; init; }

    public DateOnly? AssetValueDate { get; init; }

    public string? Reference { get; init; }

    public string? Text { get; init; }

    public Guid? IdempotencyKey { get; init; }
}

/// <param name="RevenueAmount">Sale proceeds; zero for a scrapping.</param>
public sealed record AssetRetirementRequest(
    string CompanyCode,
    string AssetNumber,
    int AssetSubNumber,
    DateOnly PostingDate,
    decimal RevenueAmount,
    string CurrencyCode)
{
    public string? CustomerPartnerNumber { get; init; }

    public string? RevenueOffsettingAccount { get; init; }

    public string? Text { get; init; }

    public Guid? IdempotencyKey { get; init; }
}

public sealed record DepreciationRunRequest(
    string CompanyCode,
    short FiscalYear,
    byte FiscalPeriod)
{
    /// <summary>`Planned`, `Repeat` or `Unplanned`.</summary>
    public string RunType { get; init; } = "Planned";

    /// <summary>Limit the run to one asset, for a correction.</summary>
    public string? AssetNumber { get; init; }

    public Guid? IdempotencyKey { get; init; }
}

public sealed record AssetPostingResult
{
    public bool IsSuccess => Violations.All(v => !v.IsBlocking) && DocumentNumber is not null;

    public string? DocumentNumber { get; init; }

    public string? AssetDocumentNumber { get; init; }

    public decimal NetBookValueAfter { get; init; }

    /// <summary>Gain (positive) or loss (negative) realised on a retirement.</summary>
    public decimal? GainOrLoss { get; init; }

    public IReadOnlyList<RuleViolation> Violations { get; init; } = [];

    public IReadOnlyList<PostingError> PostingErrors { get; init; } = [];
}

/// <summary>What a run would post, per asset and area.</summary>
public sealed record DepreciationPlan
{
    public required short FiscalYear { get; init; }

    public required byte FiscalPeriod { get; init; }

    public required IReadOnlyList<PlannedDepreciation> Items { get; init; }

    public decimal Total => Items.Sum(i => i.Amount);

    public IReadOnlyList<RuleViolation> Violations { get; init; } = [];
}

public sealed record PlannedDepreciation(
    string AssetNumber,
    int AssetSubNumber,
    string DepreciationArea,
    decimal Amount,
    decimal NetBookValueAfter,
    bool IsFinalCharge,
    string Basis);

public sealed record DepreciationRunResult
{
    public bool IsSuccess => Violations.All(v => !v.IsBlocking);

    public required short FiscalYear { get; init; }

    public required byte FiscalPeriod { get; init; }

    public string? DocumentNumber { get; init; }

    public int AssetsProcessed { get; init; }

    public decimal TotalDepreciation { get; init; }

    public IReadOnlyList<PlannedDepreciation> Postings { get; init; } = [];

    public IReadOnlyList<RuleViolation> Violations { get; init; } = [];

    public IReadOnlyList<PostingError> PostingErrors { get; init; } = [];
}

public static class AssetErrorCodes
{
    public const string AssetUnknown = "AA.ASSET_UNKNOWN";
    public const string AssetBlocked = "AA.ASSET_BLOCKED";
    public const string AssetRetired = "AA.ASSET_RETIRED";
    public const string AssetNotCapitalized = "AA.ASSET_NOT_CAPITALIZED";
    public const string AreaMissing = "AA.DEPRECIATION_AREA_MISSING";
    public const string AccountDeterminationMissing = "AA.ACCOUNT_DETERMINATION_MISSING";
    public const string PeriodUnknown = "AA.PERIOD_UNKNOWN";
    public const string RunAlreadyExecuted = "AA.RUN_ALREADY_EXECUTED";
    public const string AmountNotPositive = "AA.AMOUNT_NOT_POSITIVE";
    public const string PostingFailed = "AA.POSTING_FAILED";
}

/// <summary>
/// Account keys asset accounting derives its postings from, resolved through
/// <c>cfg.AccountDeterminationRule</c> with the asset class's determination key
/// as the modifier.
/// </summary>
public static class AssetTransactionKeys
{
    /// <summary>Balance sheet account for acquisition and production cost.</summary>
    public const string AcquisitionValue = "ANL";

    /// <summary>Accumulated depreciation.</summary>
    public const string AccumulatedDepreciation = "AFA";

    /// <summary>Depreciation expense.</summary>
    public const string DepreciationExpense = "AFX";

    /// <summary>Gain on retirement.</summary>
    public const string RetirementGain = "AAV";

    /// <summary>Loss on retirement, and the clearing of net book value on scrapping.</summary>
    public const string RetirementLoss = "AAL";
}
