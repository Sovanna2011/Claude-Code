namespace ErpS4.Domain;

/// <summary>
/// A document on its way to the ledger: the caller's intent, before any
/// configuration has been read. It carries only the rules that hold whatever
/// the configuration says - line count, non-zero amounts, one document
/// currency, and debits equal to credits.
/// </summary>
/// <remarks>
/// Everything that needs master data or configuration (period open, account
/// postable, reconciliation account rules, currency conversion) belongs to the
/// posting engine, not here. That split is what keeps this class testable
/// without a database.
/// </remarks>
public sealed class JournalEntryDraft
{
    private readonly List<JournalEntryDraftLine> _lines = [];

    public JournalEntryDraft(
        string companyCode,
        string documentType,
        DateOnly documentDate,
        DateOnly postingDate,
        string documentCurrency)
    {
        CompanyCode = companyCode;
        DocumentType = documentType;
        DocumentDate = documentDate;
        PostingDate = postingDate;
        DocumentCurrency = documentCurrency;
    }

    public string CompanyCode { get; }

    public string DocumentType { get; }

    public DateOnly DocumentDate { get; }

    public DateOnly PostingDate { get; }

    public string DocumentCurrency { get; }

    public string? HeaderText { get; init; }

    public string? ReferenceDocumentNumber { get; init; }

    /// <summary>Set when this document reverses another one.</summary>
    public ReversalReference? Reverses { get; init; }

    public IReadOnlyList<JournalEntryDraftLine> Lines => _lines;

    public JournalEntryDraft AddLine(JournalEntryDraftLine line)
    {
        _lines.Add(line with { LineNumber = _lines.Count + 1 });
        return this;
    }

    public Money TotalDebit =>
        _lines.Where(l => l.Amount.IsDebit)
              .Aggregate(Money.Zero(DocumentCurrency), (total, l) => total + l.Amount);

    public Money TotalCredit =>
        _lines.Where(l => l.Amount.IsCredit)
              .Aggregate(Money.Zero(DocumentCurrency), (total, l) => total + l.Amount)
              .Negate();

    /// <summary>Difference between debits and credits; zero on a valid document.</summary>
    public Money Difference => TotalDebit - TotalCredit;

    /// <summary>
    /// Structural validation. The engine adds the configuration-dependent rules
    /// on top; both sets are returned to the caller together so a user fixes
    /// every problem in one pass instead of one per round trip.
    /// </summary>
    public IReadOnlyList<PostingError> Validate(int currencyDecimals)
    {
        var errors = new List<PostingError>();

        if (_lines.Count < 2)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.TooFewLines,
                "A document needs at least a debit and a credit line.",
                nameof(Lines)));
        }

        foreach (var line in _lines)
        {
            if (line.Amount.IsZero)
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.LineAmountZero,
                    "A line amount cannot be zero.",
                    $"Lines[{line.LineNumber}].Amount",
                    line.LineNumber));
            }

            if (!string.Equals(line.Amount.Currency, DocumentCurrency, StringComparison.Ordinal))
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.CurrencyMixedOnDocument,
                    $"Line is in {line.Amount.Currency} but the document is in {DocumentCurrency}.",
                    $"Lines[{line.LineNumber}].Amount",
                    line.LineNumber));
            }
        }

        // Only worth asking whether the document balances once every line is in
        // the same currency. Adding USD to KHR throws by design, and the mixed
        // currency has already been reported above - reporting "not balanced"
        // on top of it would be a second message about the same mistake.
        var isSingleCurrency = _lines.All(line =>
            string.Equals(line.Amount.Currency, DocumentCurrency, StringComparison.Ordinal));

        if (!isSingleCurrency)
        {
            return errors;
        }

        var difference = Difference.Round(currencyDecimals);
        if (!difference.IsZero)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.DocumentNotBalanced,
                $"Debits {TotalDebit.Round(currencyDecimals)} do not equal credits " +
                $"{TotalCredit.Round(currencyDecimals)}; difference {difference}.",
                nameof(Lines)));
        }

        return errors;
    }
}

/// <summary>One line of a draft document.</summary>
/// <param name="PostingKey">Posting key; decides debit/credit and account type.</param>
/// <param name="Account">G/L account, or the reconciliation account for a partner line.</param>
/// <param name="Amount">Signed amount in the document currency.</param>
public sealed record JournalEntryDraftLine(string PostingKey, string Account, Money Amount)
{
    /// <summary>Assigned by <see cref="JournalEntryDraft.AddLine"/>.</summary>
    public int LineNumber { get; init; }

    public string? BusinessPartner { get; init; }

    public string? CostCenter { get; init; }

    public string? ProfitCenter { get; init; }

    public string? InternalOrder { get; init; }

    public string? Segment { get; init; }

    public string? BusinessArea { get; init; }

    public string? FunctionalArea { get; init; }

    public string? Asset { get; init; }

    public int? AssetSubNumber { get; init; }

    public string? TaxCode { get; init; }

    public Money? TaxAmount { get; init; }

    public string? Assignment { get; init; }

    public string? Text { get; init; }

    public string? PaymentTerms { get; init; }

    public DateOnly? BaselineDate { get; init; }

    public string? PartnerCompanyCode { get; init; }
}

/// <summary>Identifies the document a reversal cancels.</summary>
public sealed record ReversalReference(string DocumentNumber, short FiscalYear, string ReasonCode);
