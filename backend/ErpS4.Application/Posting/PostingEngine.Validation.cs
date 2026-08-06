using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Application.Posting;

/// <summary>
/// The rule set. Every method collects errors instead of throwing, so one round
/// trip tells the user everything that is wrong with the document.
/// </summary>
public sealed partial class PostingEngine
{
    private async Task<(PostingConfiguration? Configuration, List<PostingError> Errors)>
        LoadConfigurationAsync(JournalEntryDraft draft, CancellationToken cancellationToken)
    {
        var errors = new List<PostingError>();

        var companyCode = await context.Query<CompanyCode>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                c => c.TenantId == TenantId && c.CompanyCodeKey == draft.CompanyCode,
                cancellationToken);

        if (companyCode is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.CompanyCodeUnknown,
                $"Company code {draft.CompanyCode} does not exist.",
                nameof(JournalEntryDraft.CompanyCode)));
            return (null, errors);
        }

        if (draft.PostingDate < companyCode.ValidFrom || draft.PostingDate > companyCode.ValidTo)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.CompanyCodeNotValidOnDate,
                $"Company code {draft.CompanyCode} is valid from {companyCode.ValidFrom} " +
                $"to {companyCode.ValidTo}; posting date is {draft.PostingDate}.",
                nameof(JournalEntryDraft.PostingDate)));
        }

        var documentType = await context.Query<DocumentType>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                d => d.TenantId == TenantId && d.DocumentTypeCode == draft.DocumentType,
                cancellationToken);

        if (documentType is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.DocumentTypeUnknown,
                $"Document type {draft.DocumentType} does not exist.",
                nameof(JournalEntryDraft.DocumentType)));
            return (null, errors);
        }

        var ledger = await context.Query<Ledger>()
            .AsNoTracking()
            .FirstOrDefaultAsync(l => l.TenantId == TenantId && l.IsLeading, cancellationToken);

        if (ledger is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.DocumentTypeUnknown,
                "No leading ledger is configured for this tenant.",
                nameof(JournalEntryDraft.CompanyCode)));
            return (null, errors);
        }

        var period = await context.Query<FiscalPeriod>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                p => p.TenantId == TenantId
                     && p.FiscalYearVariantId == companyCode.FiscalYearVariantId
                     && p.PeriodStartDate <= draft.PostingDate
                     && p.PeriodEndDate >= draft.PostingDate,
                cancellationToken);

        if (period is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.PeriodNotDefined,
                $"No fiscal period covers {draft.PostingDate}. Maintain the fiscal " +
                "year variant before posting into this year.",
                nameof(JournalEntryDraft.PostingDate)));
            return (null, errors);
        }

        var accountNumbers = draft.Lines.Select(l => l.Account).Distinct().ToList();
        var postingKeyCodes = draft.Lines.Select(l => l.PostingKey).Distinct().ToList();
        var partnerNumbers = draft.Lines.Where(l => l.BusinessPartner is not null)
            .Select(l => l.BusinessPartner!).Distinct().ToList();
        var costCenterCodes = draft.Lines.Where(l => l.CostCenter is not null)
            .Select(l => l.CostCenter!).Distinct().ToList();
        var profitCenterCodes = draft.Lines.Where(l => l.ProfitCenter is not null)
            .Select(l => l.ProfitCenter!).Distinct().ToList();
        var orderNumbers = draft.Lines.Where(l => l.InternalOrder is not null)
            .Select(l => l.InternalOrder!).Distinct().ToList();
        var segmentCodes = draft.Lines.Where(l => l.Segment is not null)
            .Select(l => l.Segment!).Distinct().ToList();
        var taxCodes = draft.Lines.Where(l => l.TaxCode is not null)
            .Select(l => l.TaxCode!).Distinct().ToList();
        var assetNumbers = draft.Lines.Where(l => l.Asset is not null)
            .Select(l => l.Asset!).Distinct().ToList();

        var accounts = await context.Query<GLAccount>()
            .AsNoTracking()
            .Where(a => a.TenantId == TenantId
                        && a.ChartOfAccountsId == companyCode.ChartOfAccountsId
                        && accountNumbers.Contains(a.GLAccountCode))
            .ToListAsync(cancellationToken);

        var accountIds = accounts.Select(a => a.Id).ToList();

        var accountSegments = await context.Query<GLAccountCompanyCode>()
            .AsNoTracking()
            .Where(s => s.TenantId == TenantId
                        && s.CompanyCodeId == companyCode.Id
                        && accountIds.Contains(s.GLAccountId))
            .ToListAsync(cancellationToken);

        var partners = await context.Query<BusinessPartner>()
            .AsNoTracking()
            .Where(p => p.TenantId == TenantId && partnerNumbers.Contains(p.PartnerNumber))
            .ToListAsync(cancellationToken);

        var partnerIds = partners.Select(p => p.Id).ToList();

        var partnerSegments = await context.Query<BusinessPartnerCompanyCode>()
            .AsNoTracking()
            .Where(s => s.TenantId == TenantId
                        && s.CompanyCodeId == companyCode.Id
                        && partnerIds.Contains(s.BusinessPartnerId))
            .ToListAsync(cancellationToken);

        var costElements = await context.Query<CostElement>()
            .AsNoTracking()
            .Where(e => e.TenantId == TenantId
                        && e.IsPrimary
                        && e.GLAccountId != null
                        && accountIds.Contains(e.GLAccountId!.Value)
                        && e.ValidFrom <= draft.PostingDate
                        && e.ValidTo >= draft.PostingDate)
            .ToListAsync(cancellationToken);

        var configuration = new PostingConfiguration
        {
            CompanyCode = companyCode,
            DocumentType = documentType,
            Ledger = ledger,
            Period = period,
            PeriodControls = await context.Query<PostingPeriodControl>()
                .AsNoTracking()
                .Where(c => c.TenantId == TenantId
                            && c.PostingPeriodVariantId == companyCode.PostingPeriodVariantId)
                .ToListAsync(cancellationToken),
            PostingKeys = (await context.Query<PostingKey>()
                    .AsNoTracking()
                    .Where(k => k.TenantId == TenantId && postingKeyCodes.Contains(k.PostingKeyCode))
                    .ToListAsync(cancellationToken))
                .ToDictionary(k => k.PostingKeyCode, StringComparer.Ordinal),
            Accounts = accounts.ToDictionary(a => a.GLAccountCode, StringComparer.Ordinal),
            AccountSegments = accountSegments.ToDictionary(s => s.GLAccountId),
            Partners = partners.ToDictionary(p => p.PartnerNumber, StringComparer.Ordinal),
            PartnerSegments = partnerSegments,
            CostCenters = (await context.Query<CostCenter>()
                    .AsNoTracking()
                    .Where(c => c.TenantId == TenantId
                                && costCenterCodes.Contains(c.CostCenterCode)
                                && c.ValidFrom <= draft.PostingDate
                                && c.ValidTo >= draft.PostingDate)
                    .ToListAsync(cancellationToken))
                .ToDictionary(c => c.CostCenterCode, StringComparer.Ordinal),
            ProfitCenters = (await context.Query<ProfitCenter>()
                    .AsNoTracking()
                    .Where(p => p.TenantId == TenantId
                                && profitCenterCodes.Contains(p.ProfitCenterCode)
                                && p.ValidFrom <= draft.PostingDate
                                && p.ValidTo >= draft.PostingDate)
                    .ToListAsync(cancellationToken))
                .ToDictionary(p => p.ProfitCenterCode, StringComparer.Ordinal),
            InternalOrders = (await context.Query<InternalOrder>()
                    .AsNoTracking()
                    .Where(o => o.TenantId == TenantId && orderNumbers.Contains(o.OrderNumber))
                    .ToListAsync(cancellationToken))
                .ToDictionary(o => o.OrderNumber, StringComparer.Ordinal),
            Segments = (await context.Query<Segment>()
                    .AsNoTracking()
                    .Where(s => s.TenantId == TenantId && segmentCodes.Contains(s.SegmentCode))
                    .ToListAsync(cancellationToken))
                .ToDictionary(s => s.SegmentCode, StringComparer.Ordinal),
            TaxCodes = (await context.Query<TaxCode>()
                    .AsNoTracking()
                    .Where(t => t.TenantId == TenantId
                                && t.CountryCode == companyCode.CountryCode
                                && taxCodes.Contains(t.TaxCodeKey))
                    .ToListAsync(cancellationToken))
                .ToDictionary(t => t.TaxCodeKey, StringComparer.Ordinal),
            CostElementsByAccount = costElements.ToDictionary(e => e.GLAccountId!.Value),
            Assets = (await context.Query<Asset>()
                    .AsNoTracking()
                    .Where(a => a.TenantId == TenantId
                                && a.CompanyCodeId == companyCode.Id
                                && assetNumbers.Contains(a.AssetNumber))
                    .ToListAsync(cancellationToken))
                .ToDictionary(a => a.AssetNumber, StringComparer.Ordinal),
            DocumentCurrencyDecimals =
                await currencyConverter.GetDecimalsAsync(draft.DocumentCurrency, cancellationToken),
            LocalCurrencyDecimals =
                await currencyConverter.GetDecimalsAsync(companyCode.LocalCurrencyCode, cancellationToken),
            GroupCurrencyDecimals = companyCode.GroupCurrencyCode is null
                ? 2
                : await currencyConverter.GetDecimalsAsync(companyCode.GroupCurrencyCode, cancellationToken),
            ControllingAreaId = companyCode.ControllingAreaId
                ?? await context.Query<ControllingAreaCompanyCode>()
                    .AsNoTracking()
                    .Where(a => a.TenantId == TenantId && a.CompanyCodeId == companyCode.Id)
                    .Select(a => (long?)a.ControllingAreaId)
                    .FirstOrDefaultAsync(cancellationToken),
        };

        return (configuration, errors);
    }

    /// <summary>Period status and the OB52 window for the whole document.</summary>
    private static List<PostingError> ValidatePeriod(
        JournalEntryDraft draft,
        PostingConfiguration configuration)
    {
        var errors = new List<PostingError>();

        if (configuration.Period.PeriodStatus != "Open")
        {
            errors.Add(new PostingError(
                PostingErrorCodes.PeriodClosed,
                $"Period {configuration.Period.FiscalPeriodCode}/{configuration.Period.FiscalYear} " +
                $"is {configuration.Period.PeriodStatus}.",
                nameof(JournalEntryDraft.PostingDate)));
        }

        return errors;
    }

    /// <summary>
    /// Whether the OB52 window for an account type contains this period. Both
    /// intervals are honoured; the second one is what lets a closing team keep
    /// posting while everyone else is locked out.
    /// </summary>
    private static bool IsPeriodOpenFor(
        PostingPeriodControl? control,
        short fiscalYear,
        byte period)
    {
        if (control is null)
        {
            return false;
        }

        static bool Within(short fromYear, byte fromPeriod, short toYear, byte toPeriod,
            short year, byte period) =>
            (year > fromYear || (year == fromYear && period >= fromPeriod))
            && (year < toYear || (year == toYear && period <= toPeriod));

        if (Within(control.FromYear1, control.FromPeriod1, control.ToYear1, control.ToPeriod1,
                fiscalYear, period))
        {
            return true;
        }

        return control.FromPeriod2 is { } fromPeriod2
               && control.FromYear2 is { } fromYear2
               && control.ToPeriod2 is { } toPeriod2
               && control.ToYear2 is { } toYear2
               && Within(fromYear2, fromPeriod2, toYear2, toPeriod2, fiscalYear, period);
    }

    private List<PostingError> ValidateLines(
        JournalEntryDraft draft,
        PostingConfiguration configuration)
    {
        var errors = new List<PostingError>();
        var fiscalYear = (short)configuration.Period.FiscalYear;
        var period = configuration.Period.FiscalPeriodCode;

        foreach (var line in draft.Lines)
        {
            var field = $"Lines[{line.LineNumber}]";

            if (!configuration.PostingKeys.TryGetValue(line.PostingKey, out var postingKey))
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.PostingKeyUnknown,
                    $"Posting key {line.PostingKey} does not exist.",
                    $"{field}.PostingKey", line.LineNumber));
                continue;
            }

            if (postingKey.IsBlocked)
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.PostingKeyBlocked,
                    $"Posting key {line.PostingKey} is blocked for entry.",
                    $"{field}.PostingKey", line.LineNumber));
            }

            var expectedDebit = postingKey.DebitCreditIndicator == "S";
            if (expectedDebit != line.Amount.IsDebit)
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.PostingKeySignMismatch,
                    $"Posting key {line.PostingKey} posts a " +
                    $"{(expectedDebit ? "debit" : "credit")} but the amount is " +
                    $"{(line.Amount.IsDebit ? "positive" : "negative")}.",
                    $"{field}.Amount", line.LineNumber));
            }

            if (!IsAccountTypeAllowed(configuration.DocumentType, postingKey.AccountType))
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.AccountTypeNotAllowed,
                    $"Document type {configuration.DocumentType.DocumentTypeCode} does not allow " +
                    $"account type {postingKey.AccountType}.",
                    $"{field}.PostingKey", line.LineNumber));
            }

            if (!IsPeriodOpenFor(configuration.ControlFor(postingKey.AccountType), fiscalYear, period))
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.PeriodClosed,
                    $"Period {period}/{fiscalYear} is not open for account type " +
                    $"{postingKey.AccountType}.",
                    $"{field}.PostingKey", line.LineNumber));
            }

            if (!configuration.Accounts.TryGetValue(line.Account, out var account))
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.AccountUnknown,
                    $"Account {line.Account} does not exist in the chart of accounts.",
                    $"{field}.Account", line.LineNumber));
                continue;
            }

            if (account.IsBlockedForPosting)
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.AccountBlockedForPosting,
                    $"Account {line.Account} is blocked for posting.",
                    $"{field}.Account", line.LineNumber));
            }

            if (!configuration.AccountSegments.TryGetValue(account.Id, out var segment))
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.AccountNotOpenInCompanyCode,
                    $"Account {line.Account} is not open in company code {draft.CompanyCode}.",
                    $"{field}.Account", line.LineNumber));
                continue;
            }

            if (segment.IsBlockedForPosting)
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.AccountBlockedForPosting,
                    $"Account {line.Account} is blocked in company code {draft.CompanyCode}.",
                    $"{field}.Account", line.LineNumber));
            }

            errors.AddRange(ValidateReconciliation(line, postingKey, account, configuration, field));
            errors.AddRange(ValidateAccountAssignment(line, postingKey, segment, configuration, field));

            errors.AddRange(ValidateAsset(line, postingKey, configuration, field));

            if (line.TaxCode is not null && !configuration.TaxCodes.ContainsKey(line.TaxCode))
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.TaxCodeUnknown,
                    $"Tax code {line.TaxCode} does not exist for country " +
                    $"{configuration.CompanyCode.CountryCode}.",
                    $"{field}.TaxCode", line.LineNumber));
            }

            if (line.InternalOrder is not null
                && configuration.InternalOrders.TryGetValue(line.InternalOrder, out var order)
                && order.SystemStatus is "Created" or "Closed" or "Locked")
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.CostObjectNotValidOnDate,
                    $"Internal order {line.InternalOrder} is {order.SystemStatus} and cannot " +
                    "receive postings.",
                    $"{field}.InternalOrder", line.LineNumber));
            }
        }

        return errors;
    }

    /// <summary>
    /// An asset line has to name an asset that exists, is not blocked, and
    /// belongs to this company code - an asset account posted without an asset
    /// leaves the register and the ledger disagreeing.
    /// </summary>
    private static List<PostingError> ValidateAsset(
        JournalEntryDraftLine line,
        PostingKey postingKey,
        PostingConfiguration configuration,
        string field)
    {
        var errors = new List<PostingError>();

        if (postingKey.AccountType != "A")
        {
            return errors;
        }

        if (line.Asset is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.AssetRequired,
                $"Posting key {postingKey.PostingKeyCode} posts to an asset account and needs " +
                "an asset number.",
                $"{field}.Asset", line.LineNumber));
            return errors;
        }

        if (!configuration.Assets.TryGetValue(line.Asset, out var asset))
        {
            errors.Add(new PostingError(
                PostingErrorCodes.AssetUnknown,
                $"Asset {line.Asset} does not exist in company code " +
                $"{configuration.CompanyCode.CompanyCodeKey}.",
                $"{field}.Asset", line.LineNumber));
            return errors;
        }

        if (asset.IsPostingBlocked)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.AssetBlocked,
                $"Asset {line.Asset} is blocked for postings.",
                $"{field}.Asset", line.LineNumber));
        }

        return errors;
    }

    /// <summary>
    /// A reconciliation account is only ever posted through its subledger. This
    /// is the rule that keeps the subledgers reconciled to the general ledger,
    /// so it is checked from both sides: a G/L line may not touch one, and a
    /// partner line must use exactly the account the partner's company code
    /// segment points at.
    /// </summary>
    private static List<PostingError> ValidateReconciliation(
        JournalEntryDraftLine line,
        PostingKey postingKey,
        GLAccount account,
        PostingConfiguration configuration,
        string field)
    {
        var errors = new List<PostingError>();

        if (postingKey.AccountType == "S" && account.IsReconciliationAccount)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.ReconciliationAccountDirectPosting,
                $"Account {account.GLAccountCode} is a reconciliation account for " +
                $"{account.ReconciliationAccountType}; post through the subledger instead.",
                $"{field}.Account", line.LineNumber));
            return errors;
        }

        if (postingKey.AccountType is not ("D" or "K"))
        {
            return errors;
        }

        if (line.BusinessPartner is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.BusinessPartnerRequired,
                $"Posting key {postingKey.PostingKeyCode} needs a business partner.",
                $"{field}.BusinessPartner", line.LineNumber));
            return errors;
        }

        if (!configuration.Partners.TryGetValue(line.BusinessPartner, out var partner))
        {
            errors.Add(new PostingError(
                PostingErrorCodes.BusinessPartnerRequired,
                $"Business partner {line.BusinessPartner} does not exist.",
                $"{field}.BusinessPartner", line.LineNumber));
            return errors;
        }

        if (partner.IsCentralBlocked || partner.Status != "Active")
        {
            errors.Add(new PostingError(
                PostingErrorCodes.BusinessPartnerBlocked,
                $"Business partner {line.BusinessPartner} is {partner.Status.ToLowerInvariant()}.",
                $"{field}.BusinessPartner", line.LineNumber));
        }

        var roleCategory = postingKey.AccountType == "D" ? "Customer" : "Vendor";
        var partnerSegment = configuration.PartnerSegments.FirstOrDefault(
            s => s.BusinessPartnerId == partner.Id && s.RoleCategory == roleCategory);

        if (partnerSegment is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.BusinessPartnerRoleMissing,
                $"Business partner {line.BusinessPartner} has no {roleCategory.ToLowerInvariant()} " +
                $"data in company code {configuration.CompanyCode.CompanyCodeKey}.",
                $"{field}.BusinessPartner", line.LineNumber));
            return errors;
        }

        if (partnerSegment.IsPostingBlocked)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.BusinessPartnerBlocked,
                $"Business partner {line.BusinessPartner} is blocked for posting in " +
                $"{configuration.CompanyCode.CompanyCodeKey}.",
                $"{field}.BusinessPartner", line.LineNumber));
        }

        if (partnerSegment.ReconciliationGLAccountId != account.Id)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.ReconciliationAccountMismatch,
                $"Account {account.GLAccountCode} is not the reconciliation account of " +
                $"{line.BusinessPartner} in {configuration.CompanyCode.CompanyCodeKey}.",
                $"{field}.Account", line.LineNumber));
        }

        return errors;
    }

    private static List<PostingError> ValidateAccountAssignment(
        JournalEntryDraftLine line,
        PostingKey postingKey,
        GLAccountCompanyCode segment,
        PostingConfiguration configuration,
        string field)
    {
        var errors = new List<PostingError>();

        if (segment.CostCenterRequired && line.CostCenter is null && line.InternalOrder is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.CostCenterRequired,
                "This account needs a cost centre or an internal order.",
                $"{field}.CostCenter", line.LineNumber));
        }

        // A cost object means a controlling document, and a controlling
        // document needs a controlling area. Checked here rather than at the
        // point of writing it, where an unassigned company code used to throw
        // out of the middle of a transaction instead of being reported.
        if ((line.CostCenter is not null || line.InternalOrder is not null)
            && configuration.ControllingAreaId is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.ControllingAreaMissing,
                $"Company code {configuration.CompanyCode.CompanyCodeKey} is not assigned to " +
                "a controlling area, so a line with a cost object cannot be posted.",
                $"{field}.CostCenter", line.LineNumber));
        }

        if (line.CostCenter is not null)
        {
            if (!configuration.CostCenters.TryGetValue(line.CostCenter, out var costCenter))
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.CostObjectNotValidOnDate,
                    $"Cost centre {line.CostCenter} does not exist or is not valid on the " +
                    "posting date.",
                    $"{field}.CostCenter", line.LineNumber));
            }
            else if (costCenter.IsLockedForActualPrimaryCosts)
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.CostObjectNotValidOnDate,
                    $"Cost centre {line.CostCenter} is locked for actual primary costs.",
                    $"{field}.CostCenter", line.LineNumber));
            }
        }

        var profitCenterRequired =
            configuration.CompanyCode.ProfitCenterMandatory || segment.ProfitCenterRequired;

        if (profitCenterRequired && line.ProfitCenter is null && line.CostCenter is null)
        {
            errors.Add(new PostingError(
                PostingErrorCodes.ProfitCenterRequired,
                "This company code requires a profit centre on every line; supply one " +
                "directly or through a cost centre.",
                $"{field}.ProfitCenter", line.LineNumber));
        }

        if (line.ProfitCenter is not null
            && !configuration.ProfitCenters.ContainsKey(line.ProfitCenter))
        {
            errors.Add(new PostingError(
                PostingErrorCodes.CostObjectNotValidOnDate,
                $"Profit centre {line.ProfitCenter} does not exist or is not valid on the " +
                "posting date.",
                $"{field}.ProfitCenter", line.LineNumber));
        }

        if (line.Segment is not null && !configuration.Segments.ContainsKey(line.Segment))
        {
            errors.Add(new PostingError(
                PostingErrorCodes.CostObjectNotValidOnDate,
                $"Segment {line.Segment} does not exist.",
                $"{field}.Segment", line.LineNumber));
        }

        return errors;
    }

    private static bool IsAccountTypeAllowed(DocumentType documentType, string accountType) =>
        accountType switch
        {
            "S" => documentType.AllowGLAccounts,
            "D" => documentType.AllowCustomerAccounts,
            "K" => documentType.AllowVendorAccounts,
            "A" => documentType.AllowAssetAccounts,
            "M" => documentType.AllowMaterialAccounts,
            _ => false,
        };

    /// <summary>
    /// Converts every line into local and group currency. Conversion happens
    /// per line, not on the document total, so a rounding difference shows up
    /// here rather than in a report three months later.
    /// </summary>
    private async Task<(List<SimulatedLine> Lines, List<PostingError> Errors)> ConvertAsync(
        JournalEntryDraft draft,
        PostingConfiguration configuration,
        CancellationToken cancellationToken)
    {
        var lines = new List<SimulatedLine>(draft.Lines.Count);
        var errors = new List<PostingError>();

        foreach (var line in draft.Lines)
        {
            var local = await currencyConverter.ConvertAsync(
                line.Amount, configuration.LocalCurrency, draft.PostingDate,
                cancellationToken: cancellationToken);

            if (local is null)
            {
                errors.Add(new PostingError(
                    PostingErrorCodes.ExchangeRateMissing,
                    $"No exchange rate from {draft.DocumentCurrency} to " +
                    $"{configuration.LocalCurrency} on {draft.PostingDate}.",
                    $"Lines[{line.LineNumber}].Amount", line.LineNumber));
                continue;
            }

            Money? group = null;
            if (configuration.GroupCurrency is { } groupCurrency)
            {
                var converted = await currencyConverter.ConvertAsync(
                    line.Amount, groupCurrency, draft.PostingDate,
                    cancellationToken: cancellationToken);

                if (converted is null)
                {
                    errors.Add(new PostingError(
                        PostingErrorCodes.ExchangeRateMissing,
                        $"No exchange rate from {draft.DocumentCurrency} to {groupCurrency} " +
                        $"on {draft.PostingDate}.",
                        $"Lines[{line.LineNumber}].Amount", line.LineNumber));
                }
                else
                {
                    group = converted.Amount.Round(configuration.GroupCurrencyDecimals);
                }
            }

            configuration.Accounts.TryGetValue(line.Account, out var account);

            lines.Add(new SimulatedLine(
                line.LineNumber,
                line.PostingKey,
                line.Amount.IsDebit ? "S" : "H",
                line.Account,
                account?.AccountType,
                line.Amount.Round(configuration.DocumentCurrencyDecimals),
                local.Amount.Round(configuration.LocalCurrencyDecimals),
                group,
                line.BusinessPartner,
                line.CostCenter,
                line.ProfitCenter,
                line.Segment,
                line.TaxCode,
                line.Text));
        }

        return (lines, errors);
    }
}
