using ErpS4.Application.Clearing;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using Xunit;

namespace ErpS4.IntegrationTests;

/// <summary>
/// Clearing against the seeded customer invoice, on a real server.
/// </summary>
/// <remarks>
/// The sample data leaves <c>KSS-2026-DR-00000001</c> partially cleared with
/// 6 000 USD open, which is the item these tests settle. Each one takes a
/// different slice of it, so they do not depend on the order they run in.
/// </remarks>
[Collection(DatabaseCollection.Name)]
public sealed class ClearingDatabaseTests(DatabaseFixture fixture)
{
    private const string CompanyCode = "KH01";
    private const string Customer = "1000000001";
    private const string Bank = "110000";
    private const string Invoice = "KSS-2026-DR-00000001";

    private static readonly DateOnly PaymentDate = new(2026, 2, 20);

    private static ClearingRequest Payment(
        decimal applied,
        DifferenceHandling handling = DifferenceHandling.PartialPayment) =>
        new(CompanyCode,
            PaymentDirection.Incoming,
            Customer,
            Bank,
            PaymentDate,
            "USD",
            [
                new ClearingAllocation(Invoice, 2026, 1, applied)
                {
                    DifferenceHandling = handling,
                },
            ])
        {
            Reference = "IT-BANK",
        };

    /// <summary>
    /// Both error lists, because a clearing that fails inside the posting
    /// engine reports "see the posting errors" and puts them in the other one.
    /// </summary>
    private static string Explain(ClearingResult result) =>
        string.Join("; ",
            result.Violations.Select(v => v.ToString())
                .Concat(result.PostingErrors.Select(e => e.ToString())));

    private static async Task<decimal> OpenAmountAsync(DatabaseFixture fixture)
    {
        using var scope = fixture.CreateScope();
        return await scope.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<OpenItem>()
            .Where(o => o.DocumentNumber == Invoice && o.LineItemNumber == 1)
            .Select(o => o.OpenAmountInDocumentCurrency)
            .SingleAsync();
    }

    [RequiresDatabaseFact]
    public async Task A_partial_payment_posts_a_document_and_reduces_what_is_open()
    {
        var before = await OpenAmountAsync(fixture);
        Assert.True(before >= 100m, $"The seeded invoice has only {before} open.");

        using var scope = fixture.CreateScope();
        var result = await scope.ServiceProvider.GetRequiredService<IClearingService>()
            .ClearAsync(Payment(100m));

        Assert.True(result.IsSuccess, Explain(result));
        Assert.Equal(100m, result.TotalCleared);
        Assert.NotNull(result.PaymentDocumentNumber);
        Assert.NotNull(result.ClearingDocumentNumber);

        Assert.Equal(before - 100m, await OpenAmountAsync(fixture));

        using var reader = fixture.CreateScope();
        var context = reader.ServiceProvider.GetRequiredService<ErpDbContext>();

        // The payment goes through the posting engine, so it is an ordinary
        // document: bank debit, receivable credit, balanced.
        var lines = await context.Set<JournalEntryLine>()
            .Where(l => l.DocumentNumber == result.PaymentDocumentNumber)
            .ToListAsync();

        Assert.Equal(2, lines.Count);
        Assert.Equal(0m, lines.Sum(l => l.AmountInDocumentCurrency));
        Assert.Equal(100m, lines.Single(l => l.GLAccount == Bank).AmountInDocumentCurrency);

        // The clearing document and its items are written alongside it.
        var clearing = await context.Set<ClearingDocument>()
            .SingleAsync(c => c.ClearingDocumentNumber == result.ClearingDocumentNumber);

        Assert.Equal(100m, clearing.TotalClearedAmount);
        Assert.False(clearing.IsReset);

        var items = await context.Set<ClearingItem>()
            .Where(i => i.ClearingDocumentId == clearing.Id)
            .ToListAsync();

        // The invoice item and the payment's own item: both sides are settled.
        Assert.Equal(2, items.Count);
    }

    [RequiresDatabaseFact]
    public async Task Paying_more_than_is_open_is_refused_and_writes_nothing()
    {
        var before = await OpenAmountAsync(fixture);

        using var scope = fixture.CreateScope();
        var context = scope.ServiceProvider.GetRequiredService<ErpDbContext>();
        var documentsBefore = await context.Set<JournalEntryHeader>().CountAsync();

        var result = await scope.ServiceProvider.GetRequiredService<IClearingService>()
            .ClearAsync(Payment(before + 1_000m));

        Assert.False(result.IsSuccess);
        Assert.Contains(result.Violations, v => v.Code == ClearingErrorCodes.OverAllocation);
        Assert.Equal(before, await OpenAmountAsync(fixture));

        using var reader = fixture.CreateScope();
        Assert.Equal(
            documentsBefore,
            await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
                .Set<JournalEntryHeader>().CountAsync());
    }

    [RequiresDatabaseFact]
    public async Task A_reset_reopens_both_sides_of_the_clearing()
    {
        using var scope = fixture.CreateScope();
        var service = scope.ServiceProvider.GetRequiredService<IClearingService>();

        var before = await OpenAmountAsync(fixture);
        var cleared = await service.ClearAsync(Payment(50m));
        Assert.True(cleared.IsSuccess, Explain(cleared));
        Assert.Equal(before - 50m, await OpenAmountAsync(fixture));

        using var resetScope = fixture.CreateScope();
        var reset = await resetScope.ServiceProvider.GetRequiredService<IClearingService>()
            .ResetAsync(new ResetClearingRequest(
                CompanyCode, 2026, cleared.ClearingDocumentNumber!, "Posted in error"));

        Assert.True(reset.IsSuccess,
            string.Join("; ", reset.Violations.Select(v => v.ToString())));

        // The invoice and the payment's own item both go back to open.
        Assert.Equal(2, reset.ItemsReopened);
        Assert.Equal(before, await OpenAmountAsync(fixture));

        using var reader = fixture.CreateScope();
        var clearing = await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<ClearingDocument>()
            .SingleAsync(c => c.ClearingDocumentNumber == cleared.ClearingDocumentNumber);

        Assert.True(clearing.IsReset);
        Assert.NotNull(clearing.ResetAt);
        Assert.Equal("integration", clearing.ResetBy);
    }

    [RequiresDatabaseFact]
    public async Task A_clearing_cannot_be_reset_twice()
    {
        using var scope = fixture.CreateScope();
        var service = scope.ServiceProvider.GetRequiredService<IClearingService>();

        var cleared = await service.ClearAsync(Payment(25m));
        Assert.True(cleared.IsSuccess, Explain(cleared));

        var request = new ResetClearingRequest(
            CompanyCode, 2026, cleared.ClearingDocumentNumber!, "Twice");

        using var firstScope = fixture.CreateScope();
        var first = await firstScope.ServiceProvider.GetRequiredService<IClearingService>()
            .ResetAsync(request);
        Assert.True(first.IsSuccess);

        using var secondScope = fixture.CreateScope();
        var second = await secondScope.ServiceProvider.GetRequiredService<IClearingService>()
            .ResetAsync(request);

        Assert.False(second.IsSuccess);
    }
}
