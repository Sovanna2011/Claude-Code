using System.Globalization;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using ErpS4.Application.Services;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;

namespace ErpS4.Application.Posting;

/// <summary>
/// The one place a document becomes accounting fact.
/// </summary>
/// <remarks>
/// <para>
/// Every module posts through here - FI, AR, AP, asset accounting, controlling,
/// and later logistics - so the rules in section 15 of the design are enforced
/// once instead of once per module: tenant, company code, period, accounts,
/// posting keys, currencies, tax, controlling derivation, reconciliation
/// account protection, and balanced debits and credits.
/// </para>
/// <para>
/// Header, lines, open items, controlling documents, balances, the audit record
/// and the integration event are written in one transaction. If any mandatory
/// step fails the whole document is rolled back; a drawn document number is not
/// reused, and the gap is recorded.
/// </para>
/// </remarks>
public sealed partial class PostingEngine(
    IErpDataContext context,
    INumberRangeService numberRanges,
    ICurrencyConverter currencyConverter,
    ITenantProvider tenantProvider,
    ICurrentUser currentUser,
    TimeProvider timeProvider,
    ILogger<PostingEngine> logger) : IPostingEngine
{
    private int TenantId => tenantProvider.TenantId;

    public async Task<PostingResult> PostAsync(
        PostingRequest request,
        CancellationToken cancellationToken = default)
    {
        var draft = request.Draft;

        if (request.IdempotencyKey is { } key && !request.Simulate)
        {
            var existing = await context.Query<JournalEntryHeader>()
                .AsNoTracking()
                .Where(h => h.TenantId == TenantId && h.IdempotencyKey == key)
                .Select(h => new { h.DocumentNumber, h.FiscalYear, h.FiscalPeriod, h.Status })
                .FirstOrDefaultAsync(cancellationToken);

            if (existing is not null)
            {
                logger.LogInformation(
                    "Idempotency key {Key} already produced document {Document}/{Year}",
                    key, existing.DocumentNumber, existing.FiscalYear);

                return PostingResult.Posted(
                    existing.DocumentNumber,
                    existing.FiscalYear,
                    existing.FiscalPeriod,
                    existing.Status,
                    [],
                    alreadyPosted: true);
            }
        }

        var (configuration, loadErrors) = await LoadConfigurationAsync(draft, cancellationToken);
        if (configuration is null)
        {
            return PostingResult.Rejected(loadErrors);
        }

        var errors = new List<PostingError>(loadErrors);
        errors.AddRange(draft.Validate(configuration.DocumentCurrencyDecimals));
        errors.AddRange(ValidatePeriod(draft, configuration));
        errors.AddRange(ValidateLines(draft, configuration));

        var (lines, conversionErrors) = await ConvertAsync(draft, configuration, cancellationToken);
        errors.AddRange(conversionErrors);

        if (errors.Count > 0)
        {
            logger.LogInformation(
                "Document for company code {CompanyCode} rejected by {Count} rules",
                draft.CompanyCode, errors.Count);
            return PostingResult.Rejected(errors);
        }

        // Local currency has to balance as well: rounding each line separately
        // can leave a residue that the document currency does not show.
        var localDifference = lines
            .Aggregate(0m, (total, line) => total + line.LocalAmount.Amount);
        if (Math.Round(localDifference, configuration.LocalCurrencyDecimals) != 0m)
        {
            return PostingResult.Rejected([
                new PostingError(
                    PostingErrorCodes.DocumentNotBalanced,
                    $"Converted amounts differ by {localDifference} {configuration.LocalCurrency}; " +
                    "adjust a line or post the rounding difference explicitly.",
                    nameof(JournalEntryDraft.Lines)),
            ]);
        }

        if (request.Simulate)
        {
            return PostingResult.Simulated(
                (short)configuration.Period.FiscalYear,
                configuration.Period.FiscalPeriodCode,
                lines);
        }

        return await context.ExecuteInTransactionAsync(
            token => CommitAsync(request, configuration, lines, token),
            cancellationToken);
    }

    public async Task<PostingResult> ReverseAsync(
        ReversalRequest request,
        CancellationToken cancellationToken = default)
    {
        var original = await context.Query<JournalEntryHeader>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                h => h.TenantId == TenantId
                     && h.DocumentNumber == request.DocumentNumber
                     && h.FiscalYear == request.FiscalYear,
                cancellationToken);

        if (original is null)
        {
            return PostingResult.Rejected([
                new PostingError(
                    PostingErrorCodes.DocumentNotPosted,
                    $"Document {request.DocumentNumber}/{request.FiscalYear} does not exist.",
                    nameof(ReversalRequest.DocumentNumber)),
            ]);
        }

        if (original.Status != "Posted")
        {
            return PostingResult.Rejected([
                new PostingError(
                    PostingErrorCodes.DocumentNotPosted,
                    $"Only a posted document can be reversed; this one is {original.Status}.",
                    nameof(ReversalRequest.DocumentNumber)),
            ]);
        }

        if (original.IsReversed)
        {
            return PostingResult.Rejected([
                new PostingError(
                    PostingErrorCodes.DocumentAlreadyReversed,
                    $"Document {request.DocumentNumber} was already reversed by " +
                    $"{original.ReversalDocumentNumber}.",
                    nameof(ReversalRequest.DocumentNumber)),
            ]);
        }

        var originalLines = await context.Query<JournalEntryLine>()
            .AsNoTracking()
            .Where(l => l.TenantId == TenantId && l.JournalEntryHeaderId == original.Id)
            .OrderBy(l => l.LineItemNumber)
            .ToListAsync(cancellationToken);

        var postingDate = request.PostingDate ?? original.PostingDate;

        // The reversal mirrors the original: same accounts and assignments,
        // opposite posting keys. Nothing about the original document changes
        // except the pointer to its reversal.
        var draft = new JournalEntryDraft(
            request.CompanyCode,
            original.DocumentTypeCodeOrDefault(),
            postingDate,
            postingDate,
            original.DocumentCurrencyCode)
        {
            HeaderText = $"Reversal of {original.DocumentNumber}",
            ReferenceDocumentNumber = original.DocumentNumber,
            Reverses = new ReversalReference(
                original.DocumentNumber, original.FiscalYear, request.ReasonCode),
        };

        foreach (var line in originalLines)
        {
            draft.AddLine(new JournalEntryDraftLine(
                MirrorPostingKey(line.PostingKey),
                line.GLAccount,
                new Money(-line.AmountInDocumentCurrency, line.DocumentCurrencyCode))
            {
                BusinessPartner = null,
                CostCenter = null,
                ProfitCenter = null,
                Text = $"Reversal: {line.LineItemText}",
                Assignment = line.AssignmentReference,
            });
        }

        var result = await PostAsync(
            new PostingRequest(draft, original.SourceModule, "FB08", request.IdempotencyKey),
            cancellationToken);

        if (result.IsSuccess && !result.WasAlreadyPosted)
        {
            var tracked = await context.Query<JournalEntryHeader>()
                .FirstAsync(h => h.Id == original.Id, cancellationToken);

            tracked.IsReversed = true;
            tracked.ReversalDocumentNumber = result.DocumentNumber;
            tracked.ReversalReasonCode = request.ReasonCode;
            tracked.ReversalDate = postingDate;
            await context.SaveChangesAsync(cancellationToken);
        }

        return result;
    }

    /// <summary>
    /// Debit becomes credit on the same account type: 40 &lt;-&gt; 50,
    /// 01 &lt;-&gt; 11, 21 &lt;-&gt; 31, 70 &lt;-&gt; 75.
    /// </summary>
    public static string MirrorPostingKey(string postingKey) => postingKey switch
    {
        "40" => "50",
        "50" => "40",
        "01" => "11",
        "11" => "01",
        "15" => "01",
        "21" => "31",
        "31" => "21",
        "25" => "31",
        "70" => "75",
        "75" => "70",
        _ => postingKey,
    };

    private async Task<PostingResult> CommitAsync(
        PostingRequest request,
        PostingConfiguration configuration,
        IReadOnlyList<SimulatedLine> lines,
        CancellationToken cancellationToken)
    {
        var draft = request.Draft;
        var now = timeProvider.GetUtcNow().UtcDateTime;
        var user = currentUser.UserName;
        var fiscalYear = (short)configuration.Period.FiscalYear;
        var period = configuration.Period.FiscalPeriodCode;

        var documentNumber = await numberRanges.NextAsync(
            "RF_BELEG",
            configuration.DocumentType.NumberRangeCode,
            configuration.CompanyCode.Id,
            fiscalYear,
            configuration.DocumentType.DocumentTypeCode,
            cancellationToken);

        var totalDebit = lines.Where(l => l.DocumentAmount.IsDebit)
            .Sum(l => l.DocumentAmount.Amount);

        var header = new JournalEntryHeader
        {
            TenantId = TenantId,
            CompanyCodeId = configuration.CompanyCode.Id,
            FiscalYear = fiscalYear,
            DocumentNumber = documentNumber,
            LedgerId = configuration.Ledger.Id,
            DocumentTypeId = configuration.DocumentType.Id,
            DocumentDate = draft.DocumentDate,
            PostingDate = draft.PostingDate,
            EntryDate = DateOnly.FromDateTime(now),
            EntryTime = TimeOnly.FromDateTime(now),
            FiscalPeriod = period,
            DocumentCurrencyCode = draft.DocumentCurrency,
            LocalCurrencyCode = configuration.LocalCurrency,
            GroupCurrencyCode = configuration.GroupCurrency,
            TotalDebitAmount = totalDebit,
            TotalCreditAmount = totalDebit,
            ReferenceDocumentNumber = draft.ReferenceDocumentNumber,
            DocumentHeaderText = draft.HeaderText,
            Status = "Posted",
            PostedAt = now,
            PostedBy = user,
            IsReversed = false,
            ReversedDocumentNumber = draft.Reverses?.DocumentNumber,
            ReversalReasonCode = draft.Reverses?.ReasonCode,
            IsIntercompany = draft.Lines.Any(l => l.PartnerCompanyCode is not null),
            SourceModule = request.SourceModule,
            TransactionCode = request.TransactionCode,
            IdempotencyKey = request.IdempotencyKey,
            IsSimulation = false,
            CreatedAt = now,
            CreatedBy = user,
        };

        context.Add(header);
        await context.SaveChangesAsync(cancellationToken);

        var entityLines = BuildLines(draft, configuration, lines, header, now, user);
        context.AddRange(entityLines);
        await context.SaveChangesAsync(cancellationToken);

        AddOpenItems(entityLines, header, now, user);
        AddControllingPostings(configuration, entityLines, now, user);
        await UpdateBalancesAsync(configuration, entityLines, cancellationToken);
        await AddAuditTrailAsync(header, now, user, cancellationToken);
        AddOutboxEvent(header, entityLines, now, user);

        await context.SaveChangesAsync(cancellationToken);

        logger.LogInformation(
            "Posted {Document}/{Year} in {CompanyCode} with {LineCount} lines",
            documentNumber, fiscalYear, draft.CompanyCode, entityLines.Count);

        return PostingResult.Posted(documentNumber, fiscalYear, period, "Posted", lines);
    }

    private List<JournalEntryLine> BuildLines(
        JournalEntryDraft draft,
        PostingConfiguration configuration,
        IReadOnlyList<SimulatedLine> converted,
        JournalEntryHeader header,
        DateTime now,
        string user)
    {
        var lines = new List<JournalEntryLine>(draft.Lines.Count);

        foreach (var line in draft.Lines)
        {
            var amounts = converted.First(c => c.LineNumber == line.LineNumber);
            var postingKey = configuration.PostingKeys[line.PostingKey];
            var account = configuration.Accounts[line.Account];
            var partner = line.BusinessPartner is null
                ? null
                : configuration.Partners[line.BusinessPartner];
            var accountSegment = configuration.AccountSegments.GetValueOrDefault(account.Id);

            lines.Add(new JournalEntryLine
            {
                TenantId = TenantId,
                JournalEntryHeaderId = header.Id,
                CompanyCodeId = header.CompanyCodeId,
                FiscalYear = header.FiscalYear,
                DocumentNumber = header.DocumentNumber,
                LineItemNumber = line.LineNumber,
                LedgerId = header.LedgerId,
                PostingDate = header.PostingDate,
                FiscalPeriod = header.FiscalPeriod,
                PostingKey = postingKey.PostingKeyCode,
                DebitCreditIndicator = amounts.DebitCreditIndicator,
                AccountType = postingKey.AccountType,
                GLAccountId = account.Id,
                GLAccount = account.GLAccountCode,
                BusinessPartnerId = partner?.Id,
                BusinessPartnerRoleCategory = partner is null
                    ? null
                    : postingKey.AccountType == "D" ? "Customer" : "Vendor",
                CostCenterId = line.CostCenter is null
                    ? null
                    : configuration.CostCenters[line.CostCenter].Id,
                ProfitCenterId = line.ProfitCenter is null
                    ? null
                    : configuration.ProfitCenters[line.ProfitCenter].Id,
                InternalOrderId = line.InternalOrder is null
                    ? null
                    : configuration.InternalOrders[line.InternalOrder].Id,
                SegmentId = line.Segment is null
                    ? null
                    : configuration.Segments[line.Segment].Id,
                DocumentCurrencyCode = amounts.DocumentAmount.Currency,
                AmountInDocumentCurrency = amounts.DocumentAmount.Amount,
                LocalCurrencyCode = amounts.LocalAmount.Currency,
                AmountInLocalCurrency = amounts.LocalAmount.Amount,
                GroupCurrencyCode = amounts.GroupAmount?.Currency,
                AmountInGroupCurrency = amounts.GroupAmount?.Amount,
                TaxCodeId = line.TaxCode is null ? null : configuration.TaxCodes[line.TaxCode].Id,
                TaxAmountInDocumentCurrency = line.TaxAmount?.Amount,
                IsTaxLine = false,
                AssignmentReference = line.Assignment,
                LineItemText = line.Text,
                BaselineDate = line.BaselineDate,
                DueDate = line.BaselineDate,
                IsOpenItemManaged =
                    postingKey.AccountType is "D" or "K"
                    || (accountSegment?.IsOpenItemManaged ?? false),
                ClearingStatus = "Open",
                IsReversalLine = draft.Reverses is not null,
                SourceModule = header.SourceModule,
                CreatedAt = now,
                CreatedBy = user,
            });
        }

        return lines;
    }

    private void AddOpenItems(
        IReadOnlyList<JournalEntryLine> lines,
        JournalEntryHeader header,
        DateTime now,
        string user)
    {
        foreach (var line in lines.Where(l => l.IsOpenItemManaged))
        {
            context.Add(new OpenItem
            {
                TenantId = TenantId,
                JournalEntryLineId = line.Id,
                CompanyCodeId = line.CompanyCodeId,
                FiscalYear = line.FiscalYear,
                DocumentNumber = line.DocumentNumber,
                LineItemNumber = line.LineItemNumber,
                AccountType = line.AccountType,
                BusinessPartnerId = line.BusinessPartnerId,
                GLAccountId = line.GLAccountId,
                PostingDate = line.PostingDate,
                DocumentDate = header.DocumentDate,
                BaselineDate = line.BaselineDate,
                DueDate = line.DueDate,
                DocumentCurrencyCode = line.DocumentCurrencyCode,
                OriginalAmountInDocumentCurrency = Math.Abs(line.AmountInDocumentCurrency),
                OpenAmountInDocumentCurrency = Math.Abs(line.AmountInDocumentCurrency),
                OriginalAmountInLocalCurrency = Math.Abs(line.AmountInLocalCurrency),
                OpenAmountInLocalCurrency = Math.Abs(line.AmountInLocalCurrency),
                ClearedAmountInDocumentCurrency = 0m,
                DebitCreditIndicator = line.DebitCreditIndicator,
                DunningLevel = 0,
                AssignmentReference = line.AssignmentReference,
                ReferenceDocumentNumber = header.ReferenceDocumentNumber,
                LineItemText = line.LineItemText,
                Status = "Open",
                CreatedAt = now,
                CreatedBy = user,
            });
        }
    }

    private void AddControllingPostings(
        PostingConfiguration configuration,
        IReadOnlyList<JournalEntryLine> lines,
        DateTime now,
        string user)
    {
        var relevant = lines
            .Where(l => l.CostCenterId is not null || l.InternalOrderId is not null)
            .Where(l => configuration.CostElementsByAccount.ContainsKey(l.GLAccountId))
            .ToList();

        var sequence = 0;
        foreach (var line in relevant)
        {
            sequence++;
            var costElement = configuration.CostElementsByAccount[line.GLAccountId];
            var isCostCenter = line.CostCenterId is not null;

            context.Add(new ControllingPosting
            {
                TenantId = TenantId,
                ControllingAreaId = configuration.CompanyCode.ControllingAreaId
                    ?? throw new InvalidOperationException(
                        "A controlling posting needs a controlling area on the company code."),
                ControllingDocumentNumber =
                    $"{line.DocumentNumber}-CO{sequence.ToString("000", CultureInfo.InvariantCulture)}",
                LineItemNumber = 1,
                FiscalYear = line.FiscalYear,
                FiscalPeriod = line.FiscalPeriod,
                PostingDate = line.PostingDate,
                PlanVersion = "000",
                IsPlan = false,
                ValueType = "04",
                ObjectType = isCostCenter ? "CostCenter" : "InternalOrder",
                ObjectId = isCostCenter ? line.CostCenterId!.Value : line.InternalOrderId!.Value,
                CostElementId = costElement.Id,
                CompanyCodeId = line.CompanyCodeId,
                ProfitCenterId = line.ProfitCenterId,
                SegmentId = line.SegmentId,
                DebitCreditIndicator = line.DebitCreditIndicator,
                CurrencyCode = line.DocumentCurrencyCode,
                AmountInTransactionCurrency = line.AmountInDocumentCurrency,
                AmountInControllingAreaCurrency = line.AmountInLocalCurrency,
                AmountInCompanyCodeCurrency = line.AmountInLocalCurrency,
                TransactionType = "Primary",
                ReferenceDocumentNumber = line.DocumentNumber,
                JournalEntryHeaderId = line.JournalEntryHeaderId,
                LineItemText = line.LineItemText,
                IsReversed = false,
                CreatedAt = now,
                CreatedBy = user,
            });
        }
    }

    private async Task UpdateBalancesAsync(
        PostingConfiguration configuration,
        IReadOnlyList<JournalEntryLine> lines,
        CancellationToken cancellationToken)
    {
        var groups = lines
            .GroupBy(l => (l.LedgerId, l.CompanyCodeId, l.FiscalYear, l.FiscalPeriod, l.GLAccountId))
            .ToList();

        foreach (var group in groups)
        {
            var key = group.Key;
            var debit = group.Where(l => l.AmountInLocalCurrency > 0)
                .Sum(l => l.AmountInLocalCurrency);
            var credit = group.Where(l => l.AmountInLocalCurrency < 0)
                .Sum(l => -l.AmountInLocalCurrency);

            var balance = await context.Query<AccountBalance>()
                .FirstOrDefaultAsync(
                    b => b.TenantId == TenantId
                         && b.LedgerId == key.LedgerId
                         && b.CompanyCodeId == key.CompanyCodeId
                         && b.FiscalYear == key.FiscalYear
                         && b.FiscalPeriod == key.FiscalPeriod
                         && b.GLAccountId == key.GLAccountId
                         && b.CurrencyType == "10",
                    cancellationToken);

            if (balance is null)
            {
                context.Add(new AccountBalance
                {
                    TenantId = TenantId,
                    LedgerId = key.LedgerId,
                    CompanyCodeId = key.CompanyCodeId,
                    FiscalYear = key.FiscalYear,
                    FiscalPeriod = key.FiscalPeriod,
                    GLAccountId = key.GLAccountId,
                    CurrencyType = "10",
                    CurrencyCode = configuration.LocalCurrency,
                    DebitTotal = debit,
                    CreditTotal = credit,
                    PeriodBalance = debit - credit,
                    CumulativeBalance = debit - credit,
                    LastUpdatedAt = timeProvider.GetUtcNow().UtcDateTime,
                });
            }
            else
            {
                balance.DebitTotal += debit;
                balance.CreditTotal += credit;
                balance.PeriodBalance += debit - credit;
                balance.CumulativeBalance += debit - credit;
                balance.LastUpdatedAt = timeProvider.GetUtcNow().UtcDateTime;
            }
        }
    }

    private async Task AddAuditTrailAsync(
        JournalEntryHeader header,
        DateTime now,
        string user,
        CancellationToken cancellationToken)
    {
        var previousHash = await context.Query<AuditLog>()
            .AsNoTracking()
            .Where(a => a.TenantId == TenantId)
            .OrderByDescending(a => a.Id)
            .Select(a => a.HashChainValue)
            .FirstOrDefaultAsync(cancellationToken);

        var payload =
            $"{now:O}|{user}|Post|JournalEntry|{header.DocumentNumber}|{header.FiscalYear}|" +
            $"{header.TotalDebitAmount}|{previousHash}";

        context.Add(new AuditLog
        {
            TenantId = TenantId,
            OccurredAt = now,
            UserName = user,
            CompanyCodeId = header.CompanyCodeId,
            Action = "Post",
            ObjectType = "JournalEntry",
            ObjectId = header.Id,
            ObjectKeyText = $"{header.DocumentNumber}/{header.FiscalYear}",
            SourceType = "Api",
            SourceName = nameof(PostingEngine),
            TransactionCode = header.TransactionCode,
            Result = "Success",
            IsSensitiveAction = true,
            PreviousHashValue = previousHash,
            HashChainValue = Sha256(payload),
        });
    }

    private void AddOutboxEvent(
        JournalEntryHeader header,
        IReadOnlyList<JournalEntryLine> lines,
        DateTime now,
        string user)
    {
        // Written inside the posting transaction: an event can never describe a
        // document that was rolled back, and a committed document always has
        // its event waiting for the dispatcher.
        var payload = JsonSerializer.Serialize(new
        {
            header.DocumentNumber,
            header.FiscalYear,
            header.CompanyCodeId,
            header.PostingDate,
            header.DocumentCurrencyCode,
            header.TotalDebitAmount,
            LineCount = lines.Count,
            Accounts = lines.Select(l => l.GLAccount).Distinct().ToArray(),
        });

        context.Add(new OutboxMessage
        {
            TenantId = TenantId,
            MessageId = Guid.NewGuid(),
            EventType = "JournalEntryPosted",
            AggregateType = nameof(JournalEntryHeader),
            AggregateId = header.Id,
            PayloadJson = payload,
            OccurredAt = now,
            Status = "Pending",
            AttemptCount = 0,
            CreatedBy = user,
        });
    }

    private static string Sha256(string value) =>
        Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(value)));
}

/// <summary>Small helpers that keep the engine readable.</summary>
internal static class PostingEngineExtensions
{
    /// <summary>
    /// The document type code carried on a stored header. The header keeps the
    /// id; the code is what a draft is expressed in.
    /// </summary>
    public static string DocumentTypeCodeOrDefault(this JournalEntryHeader header) =>
        header.SourceModule switch
        {
            "AR" => "DR",
            "AP" => "KR",
            "AA" => "AA",
            _ => "SA",
        };
}
