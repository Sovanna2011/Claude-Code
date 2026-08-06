using ErpS4.Application.Posting;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using Xunit;

namespace ErpS4.IntegrationTests;

/// <summary>
/// The posting engine against a real SQL Server, on the sample dataset the
/// install script loads.
/// </summary>
[Collection(DatabaseCollection.Name)]
public sealed class PostingEngineDatabaseTests(DatabaseFixture fixture)
{
    private const string CompanyCode = "KH01";
    private const string Expense = "500000";
    private const string Cash = "110000";

    // KH01 makes the profit centre mandatory and the expense account demands a
    // cost object, so a document without them is refused - correctly, and it is
    // why the first version of these tests failed.
    private const string CostCenter = "1000";
    private const string ProfitCenter = "PC1000";

    private static readonly DateOnly PostingDate = new(2026, 2, 12);

    private static JournalEntryDraft Draft(decimal amount = 250m) =>
        new JournalEntryDraft(CompanyCode, "SA", PostingDate, PostingDate, "USD")
            {
                HeaderText = "Integration test document",
            }
            .AddLine(new JournalEntryDraftLine("40", Expense, new Money(amount, "USD"))
            {
                Text = "Debit side",
                CostCenter = CostCenter,
                ProfitCenter = ProfitCenter,
            })
            .AddLine(new JournalEntryDraftLine("50", Cash, new Money(-amount, "USD"))
            {
                Text = "Credit side",
                ProfitCenter = ProfitCenter,
            });

    [RequiresDatabaseFact]
    public async Task A_document_posts_and_every_side_effect_lands_in_one_transaction()
    {
        using var scope = fixture.CreateScope();
        var engine = scope.ServiceProvider.GetRequiredService<IPostingEngine>();

        var result = await engine.PostAsync(new PostingRequest(Draft()));

        Assert.True(result.IsSuccess, string.Join("; ", result.Errors.Select(e => e.ToString())));
        Assert.NotNull(result.DocumentNumber);

        // A separate context, so the assertions read what SQL Server stored
        // rather than what the change tracker remembers.
        using var reader = fixture.CreateScope();
        var context = reader.ServiceProvider.GetRequiredService<ErpDbContext>();

        var header = await context.Set<JournalEntryHeader>()
            .SingleAsync(h => h.DocumentNumber == result.DocumentNumber
                              && h.FiscalYear == result.FiscalYear);

        Assert.Equal("Posted", header.Status);
        Assert.Equal(250m, header.TotalDebitAmount);
        Assert.Equal(250m, header.TotalCreditAmount);

        // The audit columns are stamped by the context, not by the caller. A
        // posted header carries no rowversion: nothing ever updates it.
        Assert.Equal("integration", header.CreatedBy);
        Assert.NotEqual(default, header.CreatedAt);

        var lines = await context.Set<JournalEntryLine>()
            .Where(l => l.JournalEntryHeaderId == header.Id)
            .OrderBy(l => l.LineItemNumber)
            .ToListAsync();

        Assert.Equal(2, lines.Count);
        Assert.Equal(250m, lines[0].AmountInDocumentCurrency);
        Assert.Equal(-250m, lines[1].AmountInDocumentCurrency);

        // Signed amounts have to survive the round trip through decimal(19,4).
        Assert.Equal(0m, lines.Sum(l => l.AmountInDocumentCurrency));
        Assert.Equal(0m, lines.Sum(l => l.AmountInLocalCurrency));
    }

    [RequiresDatabaseFact]
    public async Task Each_posting_draws_the_next_number_and_never_repeats_one()
    {
        using var scope = fixture.CreateScope();
        var engine = scope.ServiceProvider.GetRequiredService<IPostingEngine>();

        var first = await engine.PostAsync(new PostingRequest(Draft(11m)));
        var second = await engine.PostAsync(new PostingRequest(Draft(12m)));

        Assert.True(first.IsSuccess);
        Assert.True(second.IsSuccess);
        Assert.NotEqual(first.DocumentNumber, second.DocumentNumber);

        // The mask has to fit the column it is stored in - the reason the
        // sample data collided on its unique key the first time it was loaded.
        Assert.True(first.DocumentNumber!.Length <= NumberRangeService.MaxNumberLength);
    }

    [RequiresDatabaseFact]
    public async Task Concurrent_postings_do_not_receive_the_same_document_number()
    {
        // The number is drawn with a single UPDATE ... OUTPUT. Nothing but a
        // real server can show whether that actually serialises.
        var postings = Enumerable.Range(0, 8).Select(async index =>
        {
            using var scope = fixture.CreateScope();
            var engine = scope.ServiceProvider.GetRequiredService<IPostingEngine>();
            return await engine.PostAsync(new PostingRequest(Draft(100m + index)));
        });

        var results = await Task.WhenAll(postings);

        Assert.All(results, r => Assert.True(
            r.IsSuccess, string.Join("; ", r.Errors.Select(e => e.ToString()))));

        var numbers = results.Select(r => r.DocumentNumber).ToList();
        Assert.Equal(numbers.Count, numbers.Distinct().Count());
    }

    [RequiresDatabaseFact]
    public async Task A_simulation_writes_nothing_at_all()
    {
        using var scope = fixture.CreateScope();
        var engine = scope.ServiceProvider.GetRequiredService<IPostingEngine>();
        var context = scope.ServiceProvider.GetRequiredService<ErpDbContext>();

        var before = await context.Set<JournalEntryHeader>().CountAsync();

        var result = await engine.PostAsync(new PostingRequest(Draft(999m), Simulate: true));

        Assert.True(result.IsSuccess);
        Assert.Null(result.DocumentNumber);

        using var reader = fixture.CreateScope();
        var after = await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<JournalEntryHeader>().CountAsync();

        Assert.Equal(before, after);
    }

    [RequiresDatabaseFact]
    public async Task A_retry_with_the_same_idempotency_key_posts_once()
    {
        var key = Guid.NewGuid();

        using var first = fixture.CreateScope();
        var one = await first.ServiceProvider.GetRequiredService<IPostingEngine>()
            .PostAsync(new PostingRequest(Draft(31m), IdempotencyKey: key));

        using var second = fixture.CreateScope();
        var two = await second.ServiceProvider.GetRequiredService<IPostingEngine>()
            .PostAsync(new PostingRequest(Draft(31m), IdempotencyKey: key));

        Assert.True(one.IsSuccess);
        Assert.True(two.IsSuccess);
        Assert.Equal(one.DocumentNumber, two.DocumentNumber);

        using var reader = fixture.CreateScope();
        var count = await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<JournalEntryHeader>()
            .CountAsync(h => h.IdempotencyKey == key);

        Assert.Equal(1, count);
    }

    [RequiresDatabaseFact]
    public async Task A_reversal_mirrors_the_original_and_points_back_at_it()
    {
        using var scope = fixture.CreateScope();
        var engine = scope.ServiceProvider.GetRequiredService<IPostingEngine>();

        var original = await engine.PostAsync(new PostingRequest(Draft(77m)));
        Assert.True(original.IsSuccess);

        var reversal = await engine.ReverseAsync(new ReversalRequest(
            original.DocumentNumber!, original.FiscalYear, CompanyCode, "01"));

        Assert.True(reversal.IsSuccess,
            string.Join("; ", reversal.Errors.Select(e => e.ToString())));

        using var reader = fixture.CreateScope();
        var context = reader.ServiceProvider.GetRequiredService<ErpDbContext>();

        var reversed = await context.Set<JournalEntryHeader>()
            .SingleAsync(h => h.DocumentNumber == original.DocumentNumber
                              && h.FiscalYear == original.FiscalYear);

        Assert.Equal("Reversed", reversed.Status);
        Assert.Equal(reversal.DocumentNumber, reversed.ReversalDocumentNumber);

        var mirror = await context.Set<JournalEntryLine>()
            .Where(l => l.DocumentNumber == reversal.DocumentNumber
                        && l.FiscalYear == reversal.FiscalYear)
            .ToListAsync();

        Assert.Equal(0m, mirror.Sum(l => l.AmountInDocumentCurrency));
    }

    [RequiresDatabaseFact]
    public async Task A_closed_period_is_refused_by_the_database_configuration()
    {
        using var scope = fixture.CreateScope();
        var engine = scope.ServiceProvider.GetRequiredService<IPostingEngine>();

        // 2020 has no fiscal period configured at all.
        var draft = new JournalEntryDraft(
                CompanyCode, "SA", new DateOnly(2020, 6, 1), new DateOnly(2020, 6, 1), "USD")
            .AddLine(new JournalEntryDraftLine("40", Expense, new Money(5m, "USD"))
            {
                CostCenter = CostCenter, ProfitCenter = ProfitCenter,
            })
            .AddLine(new JournalEntryDraftLine("50", Cash, new Money(-5m, "USD"))
            {
                ProfitCenter = ProfitCenter,
            });

        var result = await engine.PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Null(result.DocumentNumber);
    }

    [RequiresDatabaseFact]
    public async Task A_broken_document_leaves_nothing_behind()
    {
        using var scope = fixture.CreateScope();
        var engine = scope.ServiceProvider.GetRequiredService<IPostingEngine>();
        var context = scope.ServiceProvider.GetRequiredService<ErpDbContext>();

        var before = await context.Set<JournalEntryHeader>().CountAsync();

        // Debits do not equal credits.
        var draft = new JournalEntryDraft(CompanyCode, "SA", PostingDate, PostingDate, "USD")
            .AddLine(new JournalEntryDraftLine("40", Expense, new Money(100m, "USD"))
            {
                CostCenter = CostCenter, ProfitCenter = ProfitCenter,
            })
            .AddLine(new JournalEntryDraftLine("50", Cash, new Money(-90m, "USD"))
            {
                ProfitCenter = ProfitCenter,
            });

        var result = await engine.PostAsync(new PostingRequest(draft));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Errors, e => e.Code == PostingErrorCodes.DocumentNotBalanced);

        using var reader = fixture.CreateScope();
        var after = await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<JournalEntryHeader>().CountAsync();

        Assert.Equal(before, after);
    }
}
