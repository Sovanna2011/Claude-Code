using ErpS4.Application.Clearing;
using ErpS4.Application.Posting;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using ErpS4.Tests.TestDoubles;
using Microsoft.Extensions.Logging.Abstractions;
using Xunit;

namespace ErpS4.Tests;

/// <summary>
/// Clearing rules: an item is settled once, never more than it owes, and a
/// payment posts through the same engine a manual document does.
/// </summary>
public sealed class ClearingTests
{
    private static readonly DateOnly InvoiceDate = new(2026, 1, 20);
    private static readonly DateOnly PaymentDate = new(2026, 1, 28);

    private static ClearingService CreateService(PostingScenario scenario) =>
        new(scenario.Context,
            scenario.CreateEngine(),
            scenario.NumberRanges,
            new ErpS4.Database.FixedTenantProvider(1),
            new ErpS4.Database.FixedCurrentUser("tester"),
            scenario.Clock,
            NullLogger<ClearingService>.Instance);

    /// <summary>Posts a customer invoice and returns its document number.</summary>
    private static async Task<string> PostInvoiceAsync(PostingScenario scenario, decimal amount)
    {
        var draft = new JournalEntryDraft(
                PostingScenario.CompanyCodeKey, "SA", InvoiceDate, InvoiceDate, "USD")
            .AddLine(new JournalEntryDraftLine(
                "01", PostingScenario.Receivable, new Money(amount, "USD"))
            {
                BusinessPartner = PostingScenario.Customer,
                BaselineDate = InvoiceDate,
                Assignment = "INV-1",
            })
            .AddLine(new JournalEntryDraftLine(
                "50", PostingScenario.Revenue, new Money(-amount, "USD")));

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));
        Assert.True(result.IsSuccess);
        return result.DocumentNumber!;
    }

    private static ClearingRequest Request(
        string documentNumber,
        decimal applied,
        decimal discount = 0m,
        DifferenceHandling handling = DifferenceHandling.PartialPayment) =>
        new(PostingScenario.CompanyCodeKey,
            PaymentDirection.Incoming,
            PostingScenario.Customer,
            PostingScenario.Bank,
            PaymentDate,
            "USD",
            [
                new ClearingAllocation(documentNumber, 2026, 1, applied)
                {
                    CashDiscountAmount = discount,
                    DifferenceHandling = handling,
                },
            ])
        {
            CashDiscountAccount = discount > 0m ? PostingScenario.Expense : null,
            Reference = "BANK-1",
        };

    [Fact]
    public async Task A_full_payment_clears_the_item_and_posts_the_bank_movement()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);

        var result = await CreateService(scenario).ClearAsync(Request(invoice, 1000m));

        Assert.True(result.IsSuccess);
        Assert.Equal(1000m, result.TotalCleared);

        var item = scenario.Context.Set<OpenItem>().Single(i => i.DocumentNumber == invoice);
        Assert.Equal("Cleared", item.Status);
        Assert.Equal(0m, item.OpenAmountInDocumentCurrency);
        Assert.Equal(1000m, item.ClearedAmountInDocumentCurrency);
        Assert.Equal(result.ClearingDocumentNumber, item.ClearingDocumentNumber);

        // The payment's own receivable line is an open item too, and the same
        // clearing settles it. Left open it would sit on the customer account
        // as a credit nobody can explain.
        var paymentItem = scenario.Context.Set<OpenItem>()
            .Single(i => i.DocumentNumber == result.PaymentDocumentNumber);
        Assert.Equal("Cleared", paymentItem.Status);
        Assert.Equal(0m, paymentItem.OpenAmountInDocumentCurrency);
        Assert.Equal(result.ClearingDocumentNumber, paymentItem.ClearingDocumentNumber);

        // The invoice line records the clearing without its amounts changing.
        var invoiceLine = scenario.Context.Set<JournalEntryLine>()
            .Single(l => l.DocumentNumber == invoice && l.LineItemNumber == 1);
        Assert.Equal("Cleared", invoiceLine.ClearingStatus);
        Assert.Equal(1000m, invoiceLine.AmountInDocumentCurrency);

        // Bank debit and receivable credit, posted through the engine.
        var payment = scenario.Context.Set<JournalEntryLine>()
            .Where(l => l.DocumentNumber == result.PaymentDocumentNumber)
            .ToList();
        Assert.Equal(2, payment.Count);
        Assert.Equal(1000m, payment.Single(l => l.GLAccount == PostingScenario.Bank)
            .AmountInDocumentCurrency);
        Assert.Equal(-1000m, payment.Single(l => l.GLAccount == PostingScenario.Receivable)
            .AmountInDocumentCurrency);
    }

    [Fact]
    public async Task A_partial_payment_leaves_the_remainder_open_on_the_same_item()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);

        var result = await CreateService(scenario).ClearAsync(Request(invoice, 400m));

        Assert.True(result.IsSuccess);

        var item = scenario.Context.Set<OpenItem>().Single(i => i.DocumentNumber == invoice);
        Assert.Equal("PartiallyCleared", item.Status);
        Assert.Equal(600m, item.OpenAmountInDocumentCurrency);
        Assert.Null(item.ClearingDocumentNumber);

        // The payment itself is fully applied even though the invoice is not.
        Assert.Equal("Cleared", scenario.Context.Set<OpenItem>()
            .Single(i => i.DocumentNumber == result.PaymentDocumentNumber).Status);
    }

    [Fact]
    public async Task A_residual_closes_the_invoice_and_opens_a_new_item_dated_from_the_payment()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);

        var result = await CreateService(scenario)
            .ClearAsync(Request(invoice, 400m, handling: DifferenceHandling.Residual));

        Assert.True(result.IsSuccess);

        var items = scenario.Context.Set<OpenItem>();

        var original = items.Single(i => i.DocumentNumber == invoice);
        Assert.Equal("Cleared", original.Status);
        Assert.Equal(0m, original.OpenAmountInDocumentCurrency);

        // Three items on the payment document's side of the ledger: the
        // payment's own line, settled, and the residual, which is the point.
        var residual = items.Single(
            i => i.DocumentNumber == result.PaymentDocumentNumber && i.Status == "Open");
        Assert.Equal(600m, residual.OpenAmountInDocumentCurrency);

        Assert.Equal(600m, items.Sum(i => i.OpenAmountInDocumentCurrency));

        // The remainder ages from the payment, not from the original invoice.
        Assert.Equal(PaymentDate, residual.BaselineDate);
        Assert.Equal(invoice, residual.ReferenceDocumentNumber);
    }

    [Fact]
    public async Task A_cash_discount_settles_the_item_and_posts_to_its_own_account()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);

        // 980 in the bank, 20 discount: the invoice is fully settled.
        var result = await CreateService(scenario).ClearAsync(Request(invoice, 980m, discount: 20m));

        Assert.True(result.IsSuccess);
        Assert.Equal(20m, result.TotalCashDiscount);

        var item = scenario.Context.Set<OpenItem>().Single(i => i.DocumentNumber == invoice);
        Assert.Equal("Cleared", item.Status);

        var payment = scenario.Context.Set<JournalEntryLine>()
            .Where(l => l.DocumentNumber == result.PaymentDocumentNumber)
            .ToList();

        Assert.Equal(980m, payment.Single(l => l.GLAccount == PostingScenario.Bank)
            .AmountInDocumentCurrency);
        Assert.Equal(20m, payment.Single(l => l.GLAccount == PostingScenario.Expense)
            .AmountInDocumentCurrency);
        Assert.Equal(-1000m, payment.Single(l => l.GLAccount == PostingScenario.Receivable)
            .AmountInDocumentCurrency);
    }

    [Fact]
    public async Task A_discount_without_an_account_is_refused()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);

        var request = Request(invoice, 980m, discount: 20m) with { CashDiscountAccount = null };
        var result = await CreateService(scenario).ClearAsync(request);

        Assert.False(result.IsSuccess);
        Assert.Contains(
            result.Violations, v => v.Code == ClearingErrorCodes.CashDiscountAccountRequired);
    }

    [Fact]
    public async Task Paying_more_than_is_open_is_refused()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);

        var result = await CreateService(scenario).ClearAsync(Request(invoice, 1500m));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Violations, v => v.Code == ClearingErrorCodes.OverAllocation);
        Assert.Equal("Open", scenario.Context.Set<OpenItem>().Single().Status);
    }

    [Fact]
    public async Task An_item_cannot_be_cleared_twice()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);
        var service = CreateService(scenario);

        await service.ClearAsync(Request(invoice, 1000m));
        var second = await service.ClearAsync(Request(invoice, 1000m));

        Assert.False(second.IsSuccess);
        Assert.Contains(second.Violations, v => v.Code == ClearingErrorCodes.ItemAlreadyCleared);
    }

    [Fact]
    public async Task An_outgoing_payment_cannot_settle_a_customer_item()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);

        var request = Request(invoice, 1000m) with { Direction = PaymentDirection.Outgoing };
        var result = await CreateService(scenario).ClearAsync(request);

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Violations, v => v.Code == ClearingErrorCodes.DirectionMismatch);
    }

    [Fact]
    public async Task Clearing_in_another_currency_is_refused_rather_than_guessed()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);

        var request = Request(invoice, 1000m) with { CurrencyCode = "KHR" };
        var result = await CreateService(scenario).ClearAsync(request);

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Violations, v => v.Code == ClearingErrorCodes.CurrencyMismatch);
    }

    [Fact]
    public async Task An_unknown_item_is_reported()
    {
        var scenario = new PostingScenario();
        await PostInvoiceAsync(scenario, 1000m);

        var result = await CreateService(scenario).ClearAsync(Request("NOT-A-DOCUMENT", 100m));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Violations, v => v.Code == ClearingErrorCodes.ItemUnknown);
    }

    [Fact]
    public async Task A_reset_puts_the_item_back_where_it_was()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);
        var service = CreateService(scenario);

        var cleared = await service.ClearAsync(Request(invoice, 1000m));

        var reset = await service.ResetAsync(new ResetClearingRequest(
            PostingScenario.CompanyCodeKey, 2026, cleared.ClearingDocumentNumber!, "Wrong invoice"));

        Assert.True(reset.IsSuccess);
        // Both sides go back to open: the invoice and the payment's own line.
        Assert.Equal(2, reset.ItemsReopened);

        var item = scenario.Context.Set<OpenItem>().Single(i => i.DocumentNumber == invoice);
        Assert.Equal("Open", item.Status);
        Assert.Equal(1000m, item.OpenAmountInDocumentCurrency);
        Assert.Equal(0m, item.ClearedAmountInDocumentCurrency);
        Assert.Null(item.ClearingDocumentNumber);
    }

    [Fact]
    public async Task A_clearing_cannot_be_reset_twice()
    {
        var scenario = new PostingScenario();
        var invoice = await PostInvoiceAsync(scenario, 1000m);
        var service = CreateService(scenario);

        var cleared = await service.ClearAsync(Request(invoice, 1000m));
        var request = new ResetClearingRequest(
            PostingScenario.CompanyCodeKey, 2026, cleared.ClearingDocumentNumber!, "Mistake");

        await service.ResetAsync(request);
        var second = await service.ResetAsync(request);

        Assert.False(second.IsSuccess);
        Assert.Contains(second.Violations, v => v.Code == ClearingErrorCodes.ClearingAlreadyReset);
    }

    [Fact]
    public async Task Two_invoices_can_be_settled_by_one_payment()
    {
        var scenario = new PostingScenario();
        var first = await PostInvoiceAsync(scenario, 300m);
        var second = await PostInvoiceAsync(scenario, 700m);

        var result = await CreateService(scenario).ClearAsync(new ClearingRequest(
            PostingScenario.CompanyCodeKey,
            PaymentDirection.Incoming,
            PostingScenario.Customer,
            PostingScenario.Bank,
            PaymentDate,
            "USD",
            [
                new ClearingAllocation(first, 2026, 1, 300m),
                new ClearingAllocation(second, 2026, 1, 700m),
            ]));

        Assert.True(result.IsSuccess);
        Assert.Equal(1000m, result.TotalCleared);
        Assert.Equal(2, result.ClearedItems.Count);
        Assert.All(scenario.Context.Set<OpenItem>(), i => Assert.Equal("Cleared", i.Status));

        // One bank line, two receivable lines: a single payment document.
        var payment = scenario.Context.Set<JournalEntryLine>()
            .Where(l => l.DocumentNumber == result.PaymentDocumentNumber)
            .ToList();
        Assert.Equal(3, payment.Count);
    }
}
