using ErpS4.Domain;
using Xunit;

namespace ErpS4.Tests;

public sealed class MoneyTests
{
    [Fact]
    public void Rounding_follows_the_currency_not_a_global_default()
    {
        // KHR has no minor unit, so rounding to two decimals would invent
        // precision the currency does not have.
        Assert.Equal(4101m, new Money(4100.5m, "KHR").Round(0).Amount);
        Assert.Equal(1000.13m, new Money(1000.125m, "USD").Round(2).Amount);
    }

    [Fact]
    public void Rounding_goes_away_from_zero_on_both_signs()
    {
        Assert.Equal(2.35m, new Money(2.345m, "USD").Round(2).Amount);
        Assert.Equal(-2.35m, new Money(-2.345m, "USD").Round(2).Amount);
    }

    [Fact]
    public void Amounts_in_different_currencies_cannot_be_added()
    {
        var exception = Assert.Throws<InvalidOperationException>(
            () => new Money(1m, "USD") + new Money(1m, "KHR"));

        Assert.Contains("USD", exception.Message, StringComparison.Ordinal);
        Assert.Contains("KHR", exception.Message, StringComparison.Ordinal);
    }

    [Fact]
    public void Sign_decides_debit_and_credit()
    {
        Assert.True(new Money(1m, "USD").IsDebit);
        Assert.True(new Money(-1m, "USD").IsCredit);
        Assert.True(Money.Zero("USD").IsZero);
    }

    [Fact]
    public void Currency_is_required()
    {
        Assert.Throws<ArgumentException>(() => new Money(1m, " "));
    }
}

public sealed class JournalEntryDraftTests
{
    private static readonly DateOnly Date = new(2026, 1, 20);

    private static JournalEntryDraft Draft(string currency = "USD") =>
        new("KH01", "SA", Date, Date, currency);

    [Fact]
    public void A_balanced_document_passes()
    {
        var draft = Draft()
            .AddLine(new JournalEntryDraftLine("40", "500000", new Money(100m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", "400000", new Money(-100m, "USD")));

        Assert.Empty(draft.Validate(2));
        Assert.Equal(100m, draft.TotalDebit.Amount);
        Assert.Equal(100m, draft.TotalCredit.Amount);
        Assert.True(draft.Difference.IsZero);
    }

    [Fact]
    public void An_unbalanced_document_reports_the_difference()
    {
        var draft = Draft()
            .AddLine(new JournalEntryDraftLine("40", "500000", new Money(100m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", "400000", new Money(-90m, "USD")));

        var error = Assert.Single(draft.Validate(2));
        Assert.Equal(PostingErrorCodes.DocumentNotBalanced, error.Code);
        Assert.Equal(10m, draft.Difference.Amount);
    }

    [Fact]
    public void A_document_needs_at_least_two_lines()
    {
        var draft = Draft()
            .AddLine(new JournalEntryDraftLine("40", "500000", new Money(100m, "USD")));

        Assert.Contains(draft.Validate(2), e => e.Code == PostingErrorCodes.TooFewLines);
    }

    [Fact]
    public void A_zero_line_is_refused()
    {
        var draft = Draft()
            .AddLine(new JournalEntryDraftLine("40", "500000", new Money(0m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", "400000", new Money(0m, "USD")));

        Assert.Contains(draft.Validate(2), e => e.Code == PostingErrorCodes.LineAmountZero);
    }

    [Fact]
    public void Lines_have_to_be_in_the_document_currency()
    {
        var draft = Draft()
            .AddLine(new JournalEntryDraftLine("40", "500000", new Money(100m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", "400000", new Money(-100m, "KHR")));

        Assert.Contains(draft.Validate(2), e => e.Code == PostingErrorCodes.CurrencyMixedOnDocument);
    }

    [Fact]
    public void Line_numbers_are_assigned_in_order()
    {
        var draft = Draft()
            .AddLine(new JournalEntryDraftLine("40", "500000", new Money(100m, "USD")))
            .AddLine(new JournalEntryDraftLine("50", "400000", new Money(-100m, "USD")));

        Assert.Equal(new[] { 1, 2 }, draft.Lines.Select(l => l.LineNumber).ToArray());
    }

    [Fact]
    public void Rounding_uses_the_document_currency_precision()
    {
        // Two Riel lines that differ by half a Riel: at KHR precision the
        // document does not balance, and the engine has to say so.
        var draft = Draft("KHR")
            .AddLine(new JournalEntryDraftLine("40", "500000", new Money(4100.5m, "KHR")))
            .AddLine(new JournalEntryDraftLine("50", "400000", new Money(-4100m, "KHR")));

        Assert.Contains(draft.Validate(0), e => e.Code == PostingErrorCodes.DocumentNotBalanced);
    }
}
