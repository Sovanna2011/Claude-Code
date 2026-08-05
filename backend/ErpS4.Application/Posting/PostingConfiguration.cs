using ErpS4.Database.Entities;

namespace ErpS4.Application.Posting;

/// <summary>
/// Everything the engine needs to judge one document, loaded in a handful of
/// queries before validation starts.
/// </summary>
/// <remarks>
/// Loading up front, rather than querying inside the per-line loop, is what
/// keeps a 200-line document to a fixed number of round trips.
/// </remarks>
internal sealed class PostingConfiguration
{
    public required CompanyCode CompanyCode { get; init; }

    public required DocumentType DocumentType { get; init; }

    public required Ledger Ledger { get; init; }

    public required FiscalPeriod Period { get; init; }

    public required IReadOnlyList<PostingPeriodControl> PeriodControls { get; init; }

    public required IReadOnlyDictionary<string, PostingKey> PostingKeys { get; init; }

    public required IReadOnlyDictionary<string, GLAccount> Accounts { get; init; }

    public required IReadOnlyDictionary<long, GLAccountCompanyCode> AccountSegments { get; init; }

    public required IReadOnlyDictionary<string, BusinessPartner> Partners { get; init; }

    public required IReadOnlyList<BusinessPartnerCompanyCode> PartnerSegments { get; init; }

    public required IReadOnlyDictionary<string, CostCenter> CostCenters { get; init; }

    public required IReadOnlyDictionary<string, ProfitCenter> ProfitCenters { get; init; }

    public required IReadOnlyDictionary<string, InternalOrder> InternalOrders { get; init; }

    public required IReadOnlyDictionary<string, Segment> Segments { get; init; }

    public required IReadOnlyDictionary<string, TaxCode> TaxCodes { get; init; }

    public required IReadOnlyDictionary<long, CostElement> CostElementsByAccount { get; init; }

    public required int DocumentCurrencyDecimals { get; init; }

    public required int LocalCurrencyDecimals { get; init; }

    public required int GroupCurrencyDecimals { get; init; }

    public string LocalCurrency => CompanyCode.LocalCurrencyCode;

    public string? GroupCurrency => CompanyCode.GroupCurrencyCode;

    /// <summary>
    /// The open-period window for an account type, falling back to the
    /// <c>+</c> entry that covers every type.
    /// </summary>
    public PostingPeriodControl? ControlFor(string accountType) =>
        PeriodControls.FirstOrDefault(c => c.AccountType == accountType)
        ?? PeriodControls.FirstOrDefault(c => c.AccountType == "+");
}
