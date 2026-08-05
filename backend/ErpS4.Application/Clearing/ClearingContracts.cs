using ErpS4.Domain;

namespace ErpS4.Application.Clearing;

/// <summary>
/// Open item clearing: incoming and outgoing payments, partial payments,
/// residual items, cash discount and the reset (design prompt, section 7;
/// T-codes F-28, F-53).
/// </summary>
/// <remarks>
/// Clearing does not post on its own. It builds a draft and hands it to the
/// central posting engine, so a payment goes through exactly the rules a
/// manual journal entry does - period, accounts, reconciliation, balance - and
/// there is no second posting path to keep in step.
/// </remarks>
public interface IClearingService
{
    /// <summary>Posts a payment and clears the open items it settles.</summary>
    Task<ClearingResult> ClearAsync(
        ClearingRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Resets a clearing: the items go back to open. The payment document
    /// itself is untouched - reverse it through the posting engine if it should
    /// not exist at all.
    /// </summary>
    Task<ResetResult> ResetAsync(
        ResetClearingRequest request,
        CancellationToken cancellationToken = default);
}

public enum PaymentDirection
{
    /// <summary>Customer pays us: bank is debited, the receivable is credited.</summary>
    Incoming = 0,

    /// <summary>We pay a supplier: the payable is debited, bank is credited.</summary>
    Outgoing = 1,
}

/// <summary>How a shortfall is treated when a payment does not settle an item.</summary>
public enum DifferenceHandling
{
    /// <summary>Leave the remainder open on the original item (partial payment).</summary>
    PartialPayment = 0,

    /// <summary>
    /// Close the original item and open a new one for the remainder, dated from
    /// the payment. The item's age restarts, which is the point: a residual is
    /// a renegotiated balance, not an overdue invoice.
    /// </summary>
    Residual = 1,
}

/// <param name="BankAccount">G/L account of the bank the money moved through.</param>
/// <param name="CashDiscountAccount">
/// Required when any allocation grants a discount: the discount is an expense
/// (incoming) or income (outgoing), not a silent reduction of revenue.
/// </param>
public sealed record ClearingRequest(
    string CompanyCode,
    PaymentDirection Direction,
    string PartnerNumber,
    string BankAccount,
    DateOnly PostingDate,
    string CurrencyCode,
    IReadOnlyList<ClearingAllocation> Allocations)
{
    public DateOnly? DocumentDate { get; init; }

    public string? CashDiscountAccount { get; init; }

    public string? Reference { get; init; }

    public string? Text { get; init; }

    public Guid? IdempotencyKey { get; init; }
}

/// <param name="AppliedAmount">Positive amount taken off this item.</param>
/// <param name="CashDiscountAmount">Discount granted; settles the item alongside the payment.</param>
public sealed record ClearingAllocation(
    string DocumentNumber,
    short FiscalYear,
    int LineItemNumber,
    decimal AppliedAmount)
{
    public decimal CashDiscountAmount { get; init; }

    public DifferenceHandling DifferenceHandling { get; init; } = DifferenceHandling.PartialPayment;
}

public sealed record ClearingResult
{
    public bool IsSuccess => Violations.All(v => !v.IsBlocking) && ClearingDocumentNumber is not null;

    /// <summary>The payment document the posting engine created.</summary>
    public string? PaymentDocumentNumber { get; init; }

    /// <summary>The clearing document that links payment and items.</summary>
    public string? ClearingDocumentNumber { get; init; }

    public decimal TotalCleared { get; init; }

    public decimal TotalCashDiscount { get; init; }

    public IReadOnlyList<ClearedItem> ClearedItems { get; init; } = [];

    public IReadOnlyList<RuleViolation> Violations { get; init; } = [];

    /// <summary>Posting errors, when the engine refused the payment document.</summary>
    public IReadOnlyList<PostingError> PostingErrors { get; init; } = [];
}

/// <param name="RemainingOpenAmount">What is still open after this clearing.</param>
/// <param name="ResidualDocumentNumber">Set when a residual item was created.</param>
public sealed record ClearedItem(
    string DocumentNumber,
    short FiscalYear,
    int LineItemNumber,
    decimal AppliedAmount,
    decimal CashDiscountAmount,
    decimal RemainingOpenAmount,
    string Status,
    string? ResidualDocumentNumber = null);

public sealed record ResetClearingRequest(
    string CompanyCode,
    short FiscalYear,
    string ClearingDocumentNumber,
    string Reason);

public sealed record ResetResult
{
    public bool IsSuccess => Violations.All(v => !v.IsBlocking);

    public required string ClearingDocumentNumber { get; init; }

    public int ItemsReopened { get; init; }

    public IReadOnlyList<RuleViolation> Violations { get; init; } = [];
}

public static class ClearingErrorCodes
{
    public const string NoAllocations = "CLR.NO_ALLOCATIONS";
    public const string AmountNotPositive = "CLR.AMOUNT_NOT_POSITIVE";
    public const string ItemUnknown = "CLR.ITEM_UNKNOWN";
    public const string ItemAlreadyCleared = "CLR.ITEM_ALREADY_CLEARED";
    public const string ItemWrongPartner = "CLR.ITEM_WRONG_PARTNER";
    public const string ItemWrongCompanyCode = "CLR.ITEM_WRONG_COMPANY_CODE";
    public const string CurrencyMismatch = "CLR.CURRENCY_MISMATCH";
    public const string ForeignCurrencyNotSupported = "CLR.FOREIGN_CURRENCY_UNSUPPORTED";
    public const string OverAllocation = "CLR.OVER_ALLOCATION";
    public const string CashDiscountAccountRequired = "CLR.DISCOUNT_ACCOUNT_REQUIRED";
    public const string PartnerUnknown = "CLR.PARTNER_UNKNOWN";
    public const string DirectionMismatch = "CLR.DIRECTION_MISMATCH";
    public const string ClearingUnknown = "CLR.CLEARING_UNKNOWN";
    public const string ClearingAlreadyReset = "CLR.ALREADY_RESET";
    public const string PostingFailed = "CLR.POSTING_FAILED";
}
