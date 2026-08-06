using ErpS4.Application.Posting;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;

namespace ErpS4.Application.Clearing;

/// <inheritdoc />
public sealed class ClearingService(
    IErpDataContext context,
    IPostingEngine postingEngine,
    INumberRangeService numberRanges,
    ITenantProvider tenantProvider,
    ICurrentUser currentUser,
    TimeProvider timeProvider,
    ILogger<ClearingService> logger) : IClearingService
{
    private int TenantId => tenantProvider.TenantId;

    public async Task<ClearingResult> ClearAsync(
        ClearingRequest request,
        CancellationToken cancellationToken = default)
    {
        var violations = ValidateRequest(request);
        if (violations.Count > 0)
        {
            return new ClearingResult { Violations = violations };
        }

        var items = await LoadItemsAsync(request, violations, cancellationToken);
        if (violations.Count > 0)
        {
            return new ClearingResult { Violations = violations };
        }

        var draft = BuildDraft(request, items, violations);
        if (violations.Count > 0)
        {
            return new ClearingResult { Violations = violations };
        }

        var posted = await postingEngine.PostAsync(
            new PostingRequest(
                draft,
                request.Direction == PaymentDirection.Incoming ? "AR" : "AP",
                request.Direction == PaymentDirection.Incoming ? "F-28" : "F-53",
                request.IdempotencyKey),
            cancellationToken);

        if (!posted.IsSuccess)
        {
            return new ClearingResult
            {
                PostingErrors = posted.Errors,
                Violations = [
                    new RuleViolation(
                        ClearingErrorCodes.PostingFailed,
                        "The payment document was refused; see the posting errors."),
                ],
            };
        }

        return await RecordClearingAsync(request, items, posted, cancellationToken);
    }

    public async Task<ResetResult> ResetAsync(
        ResetClearingRequest request,
        CancellationToken cancellationToken = default)
    {
        var clearing = await context.Query<ClearingDocument>()
            .FirstOrDefaultAsync(
                c => c.TenantId == TenantId
                     && c.FiscalYear == request.FiscalYear
                     && c.ClearingDocumentNumber == request.ClearingDocumentNumber,
                cancellationToken);

        if (clearing is null)
        {
            return new ResetResult
            {
                ClearingDocumentNumber = request.ClearingDocumentNumber,
                Violations = [
                    new RuleViolation(
                        ClearingErrorCodes.ClearingUnknown,
                        $"Clearing document {request.ClearingDocumentNumber} does not exist."),
                ],
            };
        }

        if (clearing.IsReset)
        {
            return new ResetResult
            {
                ClearingDocumentNumber = request.ClearingDocumentNumber,
                Violations = [
                    new RuleViolation(
                        ClearingErrorCodes.ClearingAlreadyReset,
                        "This clearing was already reset."),
                ],
            };
        }

        var now = timeProvider.GetUtcNow().UtcDateTime;
        var user = currentUser.UserName;

        var clearingItems = await context.Query<ClearingItem>()
            .Where(i => i.TenantId == TenantId && i.ClearingDocumentId == clearing.Id)
            .ToListAsync(cancellationToken);

        var reopened = 0;
        foreach (var clearingItem in clearingItems)
        {
            var openItem = await context.Query<OpenItem>()
                .FirstOrDefaultAsync(o => o.Id == clearingItem.OpenItemId, cancellationToken);

            if (openItem is null)
            {
                continue;
            }

            openItem.OpenAmountInDocumentCurrency += clearingItem.ClearedAmountInDocumentCurrency;
            openItem.OpenAmountInLocalCurrency += clearingItem.ClearedAmountInLocalCurrency;
            openItem.ClearedAmountInDocumentCurrency -= clearingItem.ClearedAmountInDocumentCurrency;
            openItem.Status =
                openItem.ClearedAmountInDocumentCurrency > 0 ? "PartiallyCleared" : "Open";
            openItem.ClearingDocumentNumber = null;
            openItem.ClearingDate = null;
            openItem.ModifiedAt = now;
            openItem.ModifiedBy = user;
            reopened++;
        }

        // A clearing document records its own reversal in ResetAt / ResetBy and
        // carries no modification columns: nothing else about it ever changes.
        clearing.IsReset = true;
        clearing.ResetAt = now;
        clearing.ResetBy = user;

        await context.SaveChangesAsync(cancellationToken);

        logger.LogInformation(
            "Clearing {Clearing} reset by {User}: {Count} items reopened. Reason: {Reason}",
            request.ClearingDocumentNumber, user, reopened, request.Reason);

        return new ResetResult
        {
            ClearingDocumentNumber = request.ClearingDocumentNumber,
            ItemsReopened = reopened,
        };
    }

    private static List<RuleViolation> ValidateRequest(ClearingRequest request)
    {
        var violations = new List<RuleViolation>();

        if (request.Allocations.Count == 0)
        {
            violations.Add(new RuleViolation(
                ClearingErrorCodes.NoAllocations,
                "A clearing needs at least one open item.",
                nameof(request.Allocations)));
            return violations;
        }

        for (var index = 0; index < request.Allocations.Count; index++)
        {
            var allocation = request.Allocations[index];
            if (allocation.AppliedAmount <= 0m)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.AmountNotPositive,
                    "The applied amount has to be positive; use the direction to say which " +
                    "way the money moved.",
                    $"Allocations[{index}].AppliedAmount"));
            }

            if (allocation.CashDiscountAmount < 0m)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.AmountNotPositive,
                    "A cash discount cannot be negative.",
                    $"Allocations[{index}].CashDiscountAmount"));
            }
        }

        if (request.Allocations.Any(a => a.CashDiscountAmount > 0m)
            && string.IsNullOrWhiteSpace(request.CashDiscountAccount))
        {
            violations.Add(new RuleViolation(
                ClearingErrorCodes.CashDiscountAccountRequired,
                "A cash discount needs its own account: it is an expense, not a quiet " +
                "reduction of revenue.",
                nameof(request.CashDiscountAccount)));
        }

        return violations;
    }

    /// <summary>
    /// Loads every allocated item and checks it belongs to this partner, this
    /// company code and this currency, and still has enough left open.
    /// </summary>
    private async Task<List<LoadedItem>> LoadItemsAsync(
        ClearingRequest request,
        List<RuleViolation> violations,
        CancellationToken cancellationToken)
    {
        var loaded = new List<LoadedItem>();

        var partner = await context.Query<BusinessPartner>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                p => p.TenantId == TenantId && p.PartnerNumber == request.PartnerNumber,
                cancellationToken);

        if (partner is null)
        {
            violations.Add(new RuleViolation(
                ClearingErrorCodes.PartnerUnknown,
                $"Business partner {request.PartnerNumber} does not exist.",
                nameof(request.PartnerNumber)));
            return loaded;
        }

        var companyCode = await context.Query<CompanyCode>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                c => c.TenantId == TenantId && c.CompanyCodeKey == request.CompanyCode,
                cancellationToken);

        if (companyCode is null)
        {
            violations.Add(new RuleViolation(
                ClearingErrorCodes.ItemWrongCompanyCode,
                $"Company code {request.CompanyCode} does not exist.",
                nameof(request.CompanyCode)));
            return loaded;
        }

        var expectedAccountType = request.Direction == PaymentDirection.Incoming ? "D" : "K";

        for (var index = 0; index < request.Allocations.Count; index++)
        {
            var allocation = request.Allocations[index];
            var field = $"Allocations[{index}]";

            var item = await context.Query<OpenItem>()
                .FirstOrDefaultAsync(
                    o => o.TenantId == TenantId
                         && o.DocumentNumber == allocation.DocumentNumber
                         && o.FiscalYear == allocation.FiscalYear
                         && o.LineItemNumber == allocation.LineItemNumber,
                    cancellationToken);

            if (item is null)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.ItemUnknown,
                    $"Open item {allocation.DocumentNumber}/{allocation.FiscalYear} line " +
                    $"{allocation.LineItemNumber} does not exist.",
                    field));
                continue;
            }

            if (item.Status == "Cleared")
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.ItemAlreadyCleared,
                    $"Item {allocation.DocumentNumber} line {allocation.LineItemNumber} was " +
                    $"already cleared by {item.ClearingDocumentNumber}.",
                    field));
                continue;
            }

            if (item.CompanyCodeId != companyCode.Id)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.ItemWrongCompanyCode,
                    $"Item {allocation.DocumentNumber} belongs to another company code; " +
                    "clear it there.",
                    field));
                continue;
            }

            if (item.BusinessPartnerId != partner.Id)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.ItemWrongPartner,
                    $"Item {allocation.DocumentNumber} belongs to another partner.",
                    field));
                continue;
            }

            if (item.AccountType != expectedAccountType)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.DirectionMismatch,
                    $"Item {allocation.DocumentNumber} is a {item.AccountType} item; an " +
                    $"{request.Direction.ToString().ToLowerInvariant()} payment settles " +
                    $"{expectedAccountType} items.",
                    field));
                continue;
            }

            if (item.DocumentCurrencyCode != request.CurrencyCode)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.CurrencyMismatch,
                    $"Item {allocation.DocumentNumber} is in {item.DocumentCurrencyCode} but " +
                    $"the payment is in {request.CurrencyCode}.",
                    field));
                continue;
            }

            // Foreign currency clearing realises an exchange difference, which
            // needs the receivable credited at the original rate while the bank
            // is debited at today's - a per-line rate override the posting
            // engine does not offer yet. Refusing is better than posting a
            // silently wrong local amount.
            if (item.DocumentCurrencyCode != companyCode.LocalCurrencyCode)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.ForeignCurrencyNotSupported,
                    $"Item {allocation.DocumentNumber} is in {item.DocumentCurrencyCode}, not " +
                    $"the local currency {companyCode.LocalCurrencyCode}. Foreign currency " +
                    "clearing needs the realised exchange difference, which is not implemented " +
                    "yet.",
                    field));
                continue;
            }

            var settled = allocation.AppliedAmount + allocation.CashDiscountAmount;
            if (settled > item.OpenAmountInDocumentCurrency)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.OverAllocation,
                    $"{settled} exceeds the {item.OpenAmountInDocumentCurrency} still open on " +
                    $"item {allocation.DocumentNumber} line {allocation.LineItemNumber}.",
                    field));
                continue;
            }

            // The line has to post to the item's own reconciliation account:
            // the posting engine refuses any other, and rightly so.
            var reconciliationAccount = await context.Query<GLAccount>()
                .AsNoTracking()
                .Where(a => a.Id == item.GLAccountId)
                .Select(a => a.GLAccountCode)
                .FirstOrDefaultAsync(cancellationToken);

            if (reconciliationAccount is null)
            {
                violations.Add(new RuleViolation(
                    ClearingErrorCodes.ItemUnknown,
                    $"Item {allocation.DocumentNumber} points at an account that no longer exists.",
                    field));
                continue;
            }

            loaded.Add(new LoadedItem(item, allocation, partner, reconciliationAccount));
        }

        return loaded;
    }

    /// <summary>
    /// Bank against the reconciliation account, plus the discount line when one
    /// was granted. The draft goes through the posting engine like any other
    /// document.
    /// </summary>
    private JournalEntryDraft BuildDraft(
        ClearingRequest request,
        IReadOnlyList<LoadedItem> items,
        List<RuleViolation> violations)
    {
        var incoming = request.Direction == PaymentDirection.Incoming;
        var currency = request.CurrencyCode;
        var documentDate = request.DocumentDate ?? request.PostingDate;

        var paid = items.Sum(i => i.Allocation.AppliedAmount);
        var discount = items.Sum(i => i.Allocation.CashDiscountAmount);

        var draft = new JournalEntryDraft(
            request.CompanyCode,
            incoming ? "DZ" : "KZ",
            documentDate,
            request.PostingDate,
            currency)
        {
            HeaderText = request.Text
                ?? (incoming ? "Incoming payment" : "Outgoing payment"),
            ReferenceDocumentNumber = request.Reference,
        };

        // Bank moves by the amount actually transferred; the discount never
        // touches the bank.
        draft.AddLine(new JournalEntryDraftLine(
            incoming ? "40" : "50",
            request.BankAccount,
            new Money(incoming ? paid : -paid, currency))
        {
            Assignment = request.Reference,
            Text = draft.HeaderText,
        });

        if (discount > 0m && request.CashDiscountAccount is not null)
        {
            draft.AddLine(new JournalEntryDraftLine(
                incoming ? "40" : "50",
                request.CashDiscountAccount,
                new Money(incoming ? discount : -discount, currency))
            {
                Assignment = request.Reference,
                Text = incoming ? "Cash discount granted" : "Cash discount received",
            });
        }

        foreach (var item in items)
        {
            var settled = item.Allocation.AppliedAmount + item.Allocation.CashDiscountAmount;

            draft.AddLine(new JournalEntryDraftLine(
                incoming ? "15" : "25",
                item.ReconciliationAccount,
                new Money(incoming ? -settled : settled, currency))
            {
                BusinessPartner = item.Partner.PartnerNumber,
                Assignment = item.OpenItem.AssignmentReference,
                Text = $"Settles {item.Allocation.DocumentNumber}",
            });
        }

        return draft;
    }

    private async Task<ClearingResult> RecordClearingAsync(
        ClearingRequest request,
        IReadOnlyList<LoadedItem> items,
        PostingResult posted,
        CancellationToken cancellationToken)
    {
        var now = timeProvider.GetUtcNow().UtcDateTime;
        var user = currentUser.UserName;
        var companyCodeId = items[0].OpenItem.CompanyCodeId;

        var clearingNumber = await numberRanges.NextAsync(
            "RF_BELEG",
            "01",
            companyCodeId,
            posted.FiscalYear,
            "AB",
            cancellationToken);

        var paid = items.Sum(i => i.Allocation.AppliedAmount);
        var discount = items.Sum(i => i.Allocation.CashDiscountAmount);

        var clearing = new ClearingDocument
        {
            TenantId = TenantId,
            CompanyCodeId = companyCodeId,
            FiscalYear = posted.FiscalYear,
            ClearingDocumentNumber = clearingNumber,
            ClearingDate = request.PostingDate,
            PostingDate = request.PostingDate,
            ClearingType = request.Direction == PaymentDirection.Incoming
                ? "IncomingPayment"
                : "OutgoingPayment",
            CurrencyCode = request.CurrencyCode,
            TotalClearedAmount = paid + discount,
            DifferenceAmount = 0m,
            DifferenceHandling = items.Any(i => i.IsPartial)
                ? items.Any(i => i.Allocation.DifferenceHandling == DifferenceHandling.Residual)
                    ? "Residual"
                    : "PartialPayment"
                : null,
            IsReset = false,
            CreatedAt = now,
            CreatedBy = user,
        };

        context.Add(clearing);
        await context.SaveChangesAsync(cancellationToken);

        // The partner lines of the payment document are what a residual item
        // hangs off: the remainder belongs to the payment, not to the invoice.
        var paymentLines = await context.Query<JournalEntryLine>()
            .Where(l => l.TenantId == TenantId
                        && l.DocumentNumber == posted.DocumentNumber
                        && l.FiscalYear == posted.FiscalYear
                        && l.BusinessPartnerId != null)
            .OrderBy(l => l.LineItemNumber)
            .ToListAsync(cancellationToken);

        // Captured before the loop below, because a residual item is created
        // out of the same payment line and must stay open. Identifying the
        // payment's own items afterwards would sweep the residual up with them.
        var paymentLineIds = paymentLines.Select(l => l.Id).ToList();
        var paymentItemIds = await context.Query<OpenItem>()
            .Where(o => o.TenantId == TenantId
                        && o.DocumentNumber == posted.DocumentNumber
                        && o.FiscalYear == posted.FiscalYear
                        && paymentLineIds.Contains(o.JournalEntryLineId))
            .Select(o => o.Id)
            .ToListAsync(cancellationToken);

        var cleared = new List<ClearedItem>(items.Count);

        for (var index = 0; index < items.Count; index++)
        {
            var item = items[index];
            var openItem = item.OpenItem;
            var settled = item.Allocation.AppliedAmount + item.Allocation.CashDiscountAmount;

            context.Add(new ClearingItem
            {
                TenantId = TenantId,
                ClearingDocumentId = clearing.Id,
                OpenItemId = openItem.Id,
                ClearedAmountInDocumentCurrency = settled,
                ClearedAmountInLocalCurrency = settled,
                CashDiscountTaken = item.Allocation.CashDiscountAmount,
                IsPartialClearing = item.IsPartial,
                CreatedAt = now,
                CreatedBy = user,
            });

            openItem.OpenAmountInDocumentCurrency -= settled;
            openItem.OpenAmountInLocalCurrency -= settled;
            openItem.ClearedAmountInDocumentCurrency += settled;
            openItem.ModifiedAt = now;
            openItem.ModifiedBy = user;

            string? residualDocument = null;
            var residual = item.IsPartial
                           && item.Allocation.DifferenceHandling == DifferenceHandling.Residual;

            if (!item.IsPartial || residual)
            {
                var remainder = openItem.OpenAmountInDocumentCurrency;

                openItem.Status = "Cleared";
                openItem.OpenAmountInDocumentCurrency = 0m;
                openItem.OpenAmountInLocalCurrency = 0m;
                openItem.ClearingDocumentNumber = clearingNumber;
                openItem.ClearingDate = request.PostingDate;

                if (residual && remainder > 0m)
                {
                    residualDocument = posted.DocumentNumber;
                    AddResidualItem(
                        openItem, paymentLines, index, remainder, request, posted, now, user);
                }
            }
            else
            {
                openItem.Status = "PartiallyCleared";
            }

            await UpdateSourceLineAsync(openItem, clearingNumber, request.PostingDate, cancellationToken);

            cleared.Add(new ClearedItem(
                item.Allocation.DocumentNumber,
                item.Allocation.FiscalYear,
                item.Allocation.LineItemNumber,
                item.Allocation.AppliedAmount,
                item.Allocation.CashDiscountAmount,
                openItem.OpenAmountInDocumentCurrency,
                openItem.Status,
                residualDocument));
        }

        await ClearPaymentItemsAsync(
            paymentItemIds, clearing, clearingNumber, request, now, user, cancellationToken);

        await context.SaveChangesAsync(cancellationToken);

        logger.LogInformation(
            "Clearing {Clearing} settled {Count} items with payment {Payment}",
            clearingNumber, items.Count, posted.DocumentNumber);

        return new ClearingResult
        {
            PaymentDocumentNumber = posted.DocumentNumber,
            ClearingDocumentNumber = clearingNumber,
            TotalCleared = paid + discount,
            TotalCashDiscount = discount,
            ClearedItems = cleared,
        };
    }

    /// <summary>
    /// Clears the payment document's own partner lines against the same
    /// clearing document.
    /// </summary>
    /// <remarks>
    /// The payment posts a credit to the reconciliation account, and the
    /// posting engine opens an item for it like any other partner line. Left
    /// alone it would sit on the customer account forever as a credit nobody
    /// can explain, and the account would never age to nil however many
    /// invoices were settled. A residual item is the exception: it was created
    /// above out of the same payment line precisely to stay open.
    /// </remarks>
    private async Task ClearPaymentItemsAsync(
        IReadOnlyList<long> paymentItemIds,
        ClearingDocument clearing,
        string clearingNumber,
        ClearingRequest request,
        DateTime now,
        string user,
        CancellationToken cancellationToken)
    {
        if (paymentItemIds.Count == 0)
        {
            return;
        }

        var paymentItems = await context.Query<OpenItem>()
            .Where(o => paymentItemIds.Contains(o.Id) && o.Status == "Open")
            .ToListAsync(cancellationToken);

        foreach (var paymentItem in paymentItems)
        {
            context.Add(new ClearingItem
            {
                TenantId = TenantId,
                ClearingDocumentId = clearing.Id,
                OpenItemId = paymentItem.Id,
                ClearedAmountInDocumentCurrency = paymentItem.OpenAmountInDocumentCurrency,
                ClearedAmountInLocalCurrency = paymentItem.OpenAmountInLocalCurrency,
                CashDiscountTaken = 0m,
                IsPartialClearing = false,
                CreatedAt = now,
                CreatedBy = user,
            });

            paymentItem.ClearedAmountInDocumentCurrency +=
                paymentItem.OpenAmountInDocumentCurrency;
            paymentItem.OpenAmountInDocumentCurrency = 0m;
            paymentItem.OpenAmountInLocalCurrency = 0m;
            paymentItem.Status = "Cleared";
            paymentItem.ClearingDocumentNumber = clearingNumber;
            paymentItem.ClearingDate = request.PostingDate;
            paymentItem.ModifiedAt = now;
            paymentItem.ModifiedBy = user;

            await UpdateSourceLineAsync(
                paymentItem, clearingNumber, request.PostingDate, cancellationToken);
        }
    }

    private void AddResidualItem(
        OpenItem original,
        IReadOnlyList<JournalEntryLine> paymentLines,
        int index,
        decimal remainder,
        ClearingRequest request,
        PostingResult posted,
        DateTime now,
        string user)
    {
        var line = index < paymentLines.Count ? paymentLines[index] : paymentLines[^1];

        context.Add(new OpenItem
        {
            TenantId = TenantId,
            JournalEntryLineId = line.Id,
            CompanyCodeId = original.CompanyCodeId,
            FiscalYear = posted.FiscalYear,
            DocumentNumber = posted.DocumentNumber!,
            LineItemNumber = line.LineItemNumber,
            AccountType = original.AccountType,
            BusinessPartnerId = original.BusinessPartnerId,
            GLAccountId = original.GLAccountId,
            PostingDate = request.PostingDate,
            DocumentDate = request.DocumentDate ?? request.PostingDate,

            // The residual starts its own life: the baseline date is the
            // payment, so ageing and dunning restart from the agreement.
            BaselineDate = request.PostingDate,
            DueDate = request.PostingDate,
            DocumentCurrencyCode = original.DocumentCurrencyCode,
            OriginalAmountInDocumentCurrency = remainder,
            OpenAmountInDocumentCurrency = remainder,
            OriginalAmountInLocalCurrency = remainder,
            OpenAmountInLocalCurrency = remainder,
            ClearedAmountInDocumentCurrency = 0m,
            DebitCreditIndicator = original.DebitCreditIndicator,
            PaymentTermsId = original.PaymentTermsId,
            DunningLevel = 0,
            AssignmentReference = original.AssignmentReference,
            ReferenceDocumentNumber = original.DocumentNumber,
            LineItemText = $"Residual from {original.DocumentNumber}",
            Status = "Open",
            CreatedAt = now,
            CreatedBy = user,
        });
    }

    /// <summary>
    /// Mirrors the clearing state onto the journal line. The line's amounts are
    /// never touched - a posted document is immutable; only the clearing
    /// fields, which exist for exactly this purpose, change.
    /// </summary>
    private async Task UpdateSourceLineAsync(
        OpenItem openItem,
        string clearingNumber,
        DateOnly clearingDate,
        CancellationToken cancellationToken)
    {
        var line = await context.Query<JournalEntryLine>()
            .FirstOrDefaultAsync(l => l.Id == openItem.JournalEntryLineId, cancellationToken);

        if (line is null)
        {
            return;
        }

        line.ClearingStatus = openItem.Status;

        if (openItem.Status == "Cleared")
        {
            line.ClearingDocumentNumber = clearingNumber;
            line.ClearingDate = clearingDate;
        }
    }

    /// <param name="ReconciliationAccount">
    /// The item's own reconciliation account - never a different one.
    /// </param>
    private sealed record LoadedItem(
        OpenItem OpenItem,
        ClearingAllocation Allocation,
        BusinessPartner Partner,
        string ReconciliationAccount)
    {
        /// <summary>True when this clearing leaves something open on the item.</summary>
        public bool IsPartial =>
            Allocation.AppliedAmount + Allocation.CashDiscountAmount
            < OpenItem.OpenAmountInDocumentCurrency;
    }
}
