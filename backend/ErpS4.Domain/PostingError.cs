namespace ErpS4.Domain;

/// <summary>
/// Stable error codes the posting engine reports. The API maps them to RFC 7807
/// Problem Details; the UI maps them to messages beside the offending field.
/// They are part of the contract, so never renumber or reuse one.
/// </summary>
public static class PostingErrorCodes
{
    public const string DocumentNotBalanced = "POST.NOT_BALANCED";
    public const string PeriodClosed = "POST.PERIOD_CLOSED";
    public const string PeriodNotDefined = "POST.PERIOD_NOT_DEFINED";
    public const string CompanyCodeUnknown = "POST.COMPANY_CODE_UNKNOWN";
    public const string CompanyCodeNotValidOnDate = "POST.COMPANY_CODE_NOT_VALID_ON_DATE";
    public const string DocumentTypeUnknown = "POST.DOCUMENT_TYPE_UNKNOWN";
    public const string AccountTypeNotAllowed = "POST.ACCOUNT_TYPE_NOT_ALLOWED";
    public const string PostingKeyUnknown = "POST.POSTING_KEY_UNKNOWN";
    public const string PostingKeyBlocked = "POST.POSTING_KEY_BLOCKED";
    public const string PostingKeyAccountTypeMismatch = "POST.POSTING_KEY_ACCOUNT_TYPE_MISMATCH";
    public const string PostingKeySignMismatch = "POST.POSTING_KEY_SIGN_MISMATCH";
    public const string AccountUnknown = "POST.ACCOUNT_UNKNOWN";
    public const string AccountNotOpenInCompanyCode = "POST.ACCOUNT_NOT_IN_COMPANY_CODE";
    public const string AccountBlockedForPosting = "POST.ACCOUNT_BLOCKED";
    public const string ReconciliationAccountDirectPosting = "POST.RECONCILIATION_ACCOUNT_DIRECT";
    public const string ReconciliationAccountMismatch = "POST.RECONCILIATION_ACCOUNT_MISMATCH";
    public const string BusinessPartnerRequired = "POST.BUSINESS_PARTNER_REQUIRED";
    public const string BusinessPartnerBlocked = "POST.BUSINESS_PARTNER_BLOCKED";
    public const string BusinessPartnerRoleMissing = "POST.BUSINESS_PARTNER_ROLE_MISSING";
    public const string CostCenterRequired = "POST.COST_CENTER_REQUIRED";
    public const string ProfitCenterRequired = "POST.PROFIT_CENTER_REQUIRED";
    public const string CostObjectNotValidOnDate = "POST.COST_OBJECT_NOT_VALID_ON_DATE";
    public const string CurrencyUnknown = "POST.CURRENCY_UNKNOWN";
    public const string ExchangeRateMissing = "POST.EXCHANGE_RATE_MISSING";
    public const string TaxCodeUnknown = "POST.TAX_CODE_UNKNOWN";
    public const string TaxAmountInconsistent = "POST.TAX_AMOUNT_INCONSISTENT";
    public const string LineAmountZero = "POST.LINE_AMOUNT_ZERO";
    public const string TooFewLines = "POST.TOO_FEW_LINES";
    public const string CurrencyMixedOnDocument = "POST.CURRENCY_MIXED";
    public const string NumberRangeExhausted = "POST.NUMBER_RANGE_EXHAUSTED";
    public const string DocumentAlreadyReversed = "POST.ALREADY_REVERSED";
    public const string DocumentNotPosted = "POST.NOT_POSTED";
    public const string ReversalPeriodClosed = "POST.REVERSAL_PERIOD_CLOSED";
    public const string NotAuthorized = "POST.NOT_AUTHORIZED";
    public const string SelfApprovalForbidden = "POST.SELF_APPROVAL_FORBIDDEN";
}

/// <summary>One reason a document was refused, tied to the field that caused it.</summary>
/// <param name="Code">Stable code from <see cref="PostingErrorCodes"/>.</param>
/// <param name="Message">Message for the user, already resolved to their language.</param>
/// <param name="Field">Path of the offending field, e.g. <c>Lines[2].GLAccount</c>.</param>
/// <param name="LineNumber">Line the error belongs to, null for header errors.</param>
public sealed record PostingError(string Code, string Message, string? Field = null, int? LineNumber = null)
{
    public override string ToString() =>
        LineNumber is null ? $"{Code}: {Message}" : $"{Code} (line {LineNumber}): {Message}";
}

/// <summary>
/// Thrown when a caller ignores <see cref="PostingError"/> results and asks the
/// engine to commit anyway. Callers should prefer inspecting the result.
/// </summary>
public sealed class PostingRejectedException : Exception
{
    public PostingRejectedException(IReadOnlyList<PostingError> errors)
        : base(BuildMessage(errors))
    {
        Errors = errors;
    }

    public IReadOnlyList<PostingError> Errors { get; }

    private static string BuildMessage(IReadOnlyList<PostingError> errors) =>
        errors.Count == 1
            ? errors[0].ToString()
            : $"The document was rejected by {errors.Count} rules: " +
              string.Join("; ", errors.Select(e => e.ToString()));
}
