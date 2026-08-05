using ErpS4.Application.Posting;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using ErpS4.Tests.TestDoubles;
using Xunit;

namespace ErpS4.Tests;

/// <summary>
/// The rules section 23 of the design calls mandatory. Each one is a test the
/// build refuses to go green without: debits equal credits, closed periods
/// reject postings, reconciliation accounts are not posted directly, posted
/// documents are never changed, reversals reference their original, and a
/// retried request does not post twice.
/// </summary>
public sealed class PostingEngineTests
{
    private static readonly DateOnly PostingDate = new(2026, 1, 20);

    [Fact]
    public async Task Balanced_document_is_posted()
    {
        var scenario = new PostingScenario();

        var result = await scenario.CreateEngine()
            .PostAsync(new PostingRequest(PostingScenario.BalancedDraft()));

        Assert.True(result.IsSuccess);
        Assert.Equal("Posted", result.Status);
        Assert.Equal("TST-2026-SA-0000000001", result.DocumentNumber);
        Assert.Equal((short)2026, result.FiscalYear);
        Assert.Equal((byte)1, result.FiscalPeriod);

        var header = Assert.Single(scenario.Context.Set<JournalEntryHeader>());
        Assert.Equal("Posted", header.Status);
        Assert.Equal(1000m, header.TotalDebitAmount);
        Assert.Equal(1000m, header.TotalCreditAmount);
        Assert.Equal(2, scenario.Context.Set<JournalEntryLine>().Count);
        Assert.Equal(1, scenario.Context.TransactionCount);
    }

    [Fact]
    public async Task Unbalanced_document_is_rejected()
    {
        var scenario = new PostingScenario();

        var draft = new JournalEntryDraft(
                PostingScenario.CompanyCodeKey, "SA", PostingDate, PostingDate, "USD")
            .AddLine(new JournalEntryDraftLine("40", PostingScenario.Expense, new Money(1000m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", PostingScenario.Revenue, new Money(-900m, "USD")));

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.DocumentNotBalanced);
        Assert.Empty(scenario.Context.Set<JournalEntryHeader>());
    }

    [Fact]
    public async Task Closed_period_rejects_the_posting()
    {
        var scenario = new PostingScenario();
        scenario.Period.PeriodStatus = "Closed";

        var result = await scenario.CreateEngine()
            .PostAsync(new PostingRequest(PostingScenario.BalancedDraft()));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.PeriodClosed);
        Assert.Empty(scenario.Context.Set<JournalEntryHeader>());
    }

    [Fact]
    public async Task Period_outside_the_open_window_rejects_the_posting()
    {
        var scenario = new PostingScenario();
        scenario.PeriodControl.FromPeriod1 = 2;   // period 1 is no longer open

        var result = await scenario.CreateEngine()
            .PostAsync(new PostingRequest(PostingScenario.BalancedDraft()));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.PeriodClosed);
    }

    [Fact]
    public async Task Reconciliation_account_cannot_be_posted_directly()
    {
        var scenario = new PostingScenario();

        var draft = new JournalEntryDraft(
                PostingScenario.CompanyCodeKey, "SA", PostingDate, PostingDate, "USD")
            .AddLine(new JournalEntryDraftLine("40", PostingScenario.Receivable, new Money(500m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", PostingScenario.Revenue, new Money(-500m, "USD")));

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Contains(
            result.Errors,
            e => e.Code == PostingErrorCodes.ReconciliationAccountDirectPosting);
    }

    [Fact]
    public async Task Customer_line_must_use_the_partners_reconciliation_account()
    {
        var scenario = new PostingScenario();

        // Posting key 01 is a customer line, but the account is revenue rather
        // than the reconciliation account the partner points at.
        var draft = new JournalEntryDraft(
                PostingScenario.CompanyCodeKey, "SA", PostingDate, PostingDate, "USD")
            .AddLine(new JournalEntryDraftLine("01", PostingScenario.Revenue, new Money(500m, "USD"))
            {
                BusinessPartner = PostingScenario.Customer,
            })
            .AddLine(new JournalEntryDraftLine("50", PostingScenario.Revenue, new Money(-500m, "USD")));

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Contains(
            result.Errors,
            e => e.Code == PostingErrorCodes.ReconciliationAccountMismatch);
    }

    [Fact]
    public async Task Customer_invoice_creates_an_open_item()
    {
        var scenario = new PostingScenario();

        var draft = new JournalEntryDraft(
                PostingScenario.CompanyCodeKey, "SA", PostingDate, PostingDate, "USD")
            .AddLine(new JournalEntryDraftLine("01", PostingScenario.Receivable, new Money(500m, "USD"))
            {
                BusinessPartner = PostingScenario.Customer,
                BaselineDate = PostingDate,
            })
            .AddLine(new JournalEntryDraftLine("50", PostingScenario.Revenue, new Money(-500m, "USD")));

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.True(result.IsSuccess);

        var openItem = Assert.Single(scenario.Context.Set<OpenItem>());
        Assert.Equal("Open", openItem.Status);
        Assert.Equal(500m, openItem.OpenAmountInDocumentCurrency);
        Assert.Equal("D", openItem.AccountType);
        Assert.Equal(scenario.Partner.Id, openItem.BusinessPartnerId);
    }

    [Fact]
    public async Task Blocked_partner_is_rejected()
    {
        var scenario = new PostingScenario();
        scenario.Partner.IsCentralBlocked = true;

        var draft = new JournalEntryDraft(
                PostingScenario.CompanyCodeKey, "SA", PostingDate, PostingDate, "USD")
            .AddLine(new JournalEntryDraftLine("01", PostingScenario.Receivable, new Money(500m, "USD"))
            {
                BusinessPartner = PostingScenario.Customer,
            })
            .AddLine(new JournalEntryDraftLine("50", PostingScenario.Revenue, new Money(-500m, "USD")));

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.BusinessPartnerBlocked);
    }

    [Fact]
    public async Task Posting_key_sign_must_match_the_amount()
    {
        var scenario = new PostingScenario();

        // 40 is a debit key, so a negative amount contradicts it.
        var draft = new JournalEntryDraft(
                PostingScenario.CompanyCodeKey, "SA", PostingDate, PostingDate, "USD")
            .AddLine(new JournalEntryDraftLine("40", PostingScenario.Expense, new Money(-500m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", PostingScenario.Revenue, new Money(500m, "USD")));

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.PostingKeySignMismatch);
    }

    [Fact]
    public async Task Foreign_currency_document_is_converted_to_local_currency()
    {
        var scenario = new PostingScenario();

        // 4,100,000 KHR at 4,100 KHR/USD is 1,000 USD in the company code.
        var draft = PostingScenario.BalancedDraft(4_100_000m, "KHR");

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.True(result.IsSuccess);

        var debit = result.Lines.Single(l => l.DocumentAmount.IsDebit);
        Assert.Equal(4_100_000m, debit.DocumentAmount.Amount);
        Assert.Equal("KHR", debit.DocumentAmount.Currency);
        Assert.Equal(1_000m, debit.LocalAmount.Amount);
        Assert.Equal("USD", debit.LocalAmount.Currency);
    }

    [Fact]
    public async Task Missing_exchange_rate_is_reported_not_thrown()
    {
        var scenario = new PostingScenario();
        var draft = PostingScenario.BalancedDraft(1000m, "THB");   // no THB rate configured

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.ExchangeRateMissing);
    }

    [Fact]
    public async Task Simulation_calculates_without_writing_anything()
    {
        var scenario = new PostingScenario();

        var result = await scenario.CreateEngine()
            .PostAsync(new PostingRequest(PostingScenario.BalancedDraft(), Simulate: true));

        Assert.True(result.IsSuccess);
        Assert.True(result.WasSimulated);
        Assert.Equal(2, result.Lines.Count);
        Assert.Empty(scenario.Context.Set<JournalEntryHeader>());
        Assert.Empty(scenario.NumberRanges.Issued);
        Assert.Equal(0, scenario.Context.TransactionCount);
    }

    [Fact]
    public async Task Retrying_with_the_same_idempotency_key_does_not_post_twice()
    {
        var scenario = new PostingScenario();
        var engine = scenario.CreateEngine();
        var key = Guid.NewGuid();

        var first = await engine.PostAsync(
            new PostingRequest(PostingScenario.BalancedDraft(), IdempotencyKey: key));
        var second = await engine.PostAsync(
            new PostingRequest(PostingScenario.BalancedDraft(), IdempotencyKey: key));

        Assert.True(first.IsSuccess);
        Assert.True(second.IsSuccess);
        Assert.False(first.WasAlreadyPosted);
        Assert.True(second.WasAlreadyPosted);
        Assert.Equal(first.DocumentNumber, second.DocumentNumber);
        Assert.Single(scenario.Context.Set<JournalEntryHeader>());
        Assert.Single(scenario.NumberRanges.Issued);
    }

    [Fact]
    public async Task Posting_writes_the_audit_record_and_the_integration_event()
    {
        var scenario = new PostingScenario();

        await scenario.CreateEngine().PostAsync(new PostingRequest(PostingScenario.BalancedDraft()));

        var audit = Assert.Single(scenario.Context.Set<AuditLog>());
        Assert.Equal("Post", audit.Action);
        Assert.Equal("JournalEntry", audit.ObjectType);
        Assert.False(string.IsNullOrEmpty(audit.HashChainValue));

        var outbox = Assert.Single(scenario.Context.Set<OutboxMessage>());
        Assert.Equal("JournalEntryPosted", outbox.EventType);
        Assert.Equal("Pending", outbox.Status);
    }

    [Fact]
    public async Task Posting_updates_the_period_balance_of_each_account()
    {
        var scenario = new PostingScenario();
        var engine = scenario.CreateEngine();

        await engine.PostAsync(new PostingRequest(PostingScenario.BalancedDraft(1000m)));
        await engine.PostAsync(new PostingRequest(PostingScenario.BalancedDraft(250m)));

        var expense = scenario.Context.Set<AccountBalance>()
            .Single(b => b.GLAccountId == scenario.ExpenseAccount.Id);

        Assert.Equal(1250m, expense.DebitTotal);
        Assert.Equal(0m, expense.CreditTotal);
        Assert.Equal(1250m, expense.PeriodBalance);

        // The two balance rows are equal and opposite: the ledger stays square.
        Assert.Equal(0m, scenario.Context.Set<AccountBalance>().Sum(b => b.PeriodBalance));
    }

    [Fact]
    public async Task Reversal_creates_a_new_document_that_points_at_the_original()
    {
        var scenario = new PostingScenario();
        var engine = scenario.CreateEngine();

        var posted = await engine.PostAsync(new PostingRequest(PostingScenario.BalancedDraft()));

        var reversal = await engine.ReverseAsync(new ReversalRequest(
            posted.DocumentNumber!, posted.FiscalYear, PostingScenario.CompanyCodeKey, "01"));

        Assert.True(reversal.IsSuccess);
        Assert.NotEqual(posted.DocumentNumber, reversal.DocumentNumber);

        var original = scenario.Context.Set<JournalEntryHeader>()
            .Single(h => h.DocumentNumber == posted.DocumentNumber);

        Assert.True(original.IsReversed);
        Assert.Equal(reversal.DocumentNumber, original.ReversalDocumentNumber);
        Assert.Equal("01", original.ReversalReasonCode);

        var reversalHeader = scenario.Context.Set<JournalEntryHeader>()
            .Single(h => h.DocumentNumber == reversal.DocumentNumber);

        Assert.Equal(posted.DocumentNumber, reversalHeader.ReversedDocumentNumber);

        // The reversal mirrors every line: debit becomes credit, same accounts.
        var reversalLines = scenario.Context.Set<JournalEntryLine>()
            .Where(l => l.DocumentNumber == reversal.DocumentNumber)
            .OrderBy(l => l.LineItemNumber)
            .ToList();

        Assert.Equal(2, reversalLines.Count);
        Assert.Equal("50", reversalLines[0].PostingKey);
        Assert.Equal(PostingScenario.Expense, reversalLines[0].GLAccount);
        Assert.Equal(-1000m, reversalLines[0].AmountInDocumentCurrency);
    }

    [Fact]
    public async Task A_document_cannot_be_reversed_twice()
    {
        var scenario = new PostingScenario();
        var engine = scenario.CreateEngine();

        var posted = await engine.PostAsync(new PostingRequest(PostingScenario.BalancedDraft()));
        var request = new ReversalRequest(
            posted.DocumentNumber!, posted.FiscalYear, PostingScenario.CompanyCodeKey, "01");

        await engine.ReverseAsync(request);
        var second = await engine.ReverseAsync(request);

        Assert.False(second.IsSuccess);
        Assert.Contains(second.Errors, e => e.Code == PostingErrorCodes.DocumentAlreadyReversed);
    }

    [Fact]
    public async Task Unknown_company_code_is_reported_without_touching_the_database()
    {
        var scenario = new PostingScenario();

        var draft = new JournalEntryDraft("XX99", "SA", PostingDate, PostingDate, "USD")
            .AddLine(new JournalEntryDraftLine("40", PostingScenario.Expense, new Money(10m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", PostingScenario.Revenue, new Money(-10m, "USD")));

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.CompanyCodeUnknown);
        Assert.Equal(0, scenario.Context.TransactionCount);
    }

    [Fact]
    public async Task Every_broken_rule_is_reported_in_one_pass()
    {
        var scenario = new PostingScenario();
        scenario.Period.PeriodStatus = "Closed";

        // Closed period, unknown posting key and an unbalanced document at once.
        var draft = new JournalEntryDraft(
                PostingScenario.CompanyCodeKey, "SA", PostingDate, PostingDate, "USD")
            .AddLine(new JournalEntryDraftLine("99", PostingScenario.Expense, new Money(1000m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", PostingScenario.Revenue, new Money(-800m, "USD")));

        var result = await scenario.CreateEngine().PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.PeriodClosed);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.PostingKeyUnknown);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.DocumentNotBalanced);
    }

    [Theory]
    [InlineData("40", "50")]
    [InlineData("50", "40")]
    [InlineData("01", "11")]
    [InlineData("31", "21")]
    [InlineData("70", "75")]
    public void Reversal_mirrors_the_posting_key(string original, string mirrored) =>
        Assert.Equal(mirrored, PostingEngine.MirrorPostingKey(original));
}
