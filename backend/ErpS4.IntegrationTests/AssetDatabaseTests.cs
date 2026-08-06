using ErpS4.Application.Assets;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using Xunit;

namespace ErpS4.IntegrationTests;

/// <summary>
/// Asset accounting against the seeded boiler, on a real server.
/// </summary>
[Collection(DatabaseCollection.Name)]
public sealed class AssetDatabaseTests(DatabaseFixture fixture)
{
    private const string CompanyCode = "KH01";
    private const string Asset = "100000000001";
    private const string Vendor = "1000000002";
    private const string Payables = "200000";

    [RequiresDatabaseFact]
    public async Task A_subsequent_acquisition_raises_the_asset_and_posts_a_document()
    {
        using var scope = fixture.CreateScope();
        var service = scope.ServiceProvider.GetRequiredService<IAssetService>();

        // One depreciation area, not the sum of them: an acquisition raises
        // the value in every area the asset has, so summing would expect the
        // amount once and find it several times.
        var areaId = await BookAreaIdAsync();
        var before = await CurrentYearAcquisitionsAsync();

        var result = await service.AcquireAsync(new AssetAcquisitionRequest(
            CompanyCode, Asset, 0, 5_000m, "USD", new DateOnly(2026, 2, 18), Payables)
        {
            VendorPartnerNumber = Vendor,
            Text = "Integration acquisition",
        });

        Assert.True(result.IsSuccess, Explain(result));
        Assert.NotNull(result.DocumentNumber);

        Assert.Equal(before + 5_000m, await CurrentYearAcquisitionsAsync());

        using var reader = fixture.CreateScope();
        var context = reader.ServiceProvider.GetRequiredService<ErpDbContext>();

        // The document goes through the posting engine and balances like any
        // other, and the asset line carries the asset it belongs to.
        var lines = await context.Set<JournalEntryLine>()
            .Where(l => l.DocumentNumber == result.DocumentNumber)
            .ToListAsync();

        Assert.NotEmpty(lines);
        Assert.Equal(0m, lines.Sum(l => l.AmountInDocumentCurrency));
        Assert.Contains(lines, l => l.AssetId is not null);

        // Asset transactions record the movement in their own right, keyed by
        // the asset document number rather than the FI one, and written once
        // per depreciation area.
        Assert.NotNull(result.AssetDocumentNumber);

        var transactions = await context.Set<AssetTransaction>()
            .Where(t => t.AssetDocumentNumber == result.AssetDocumentNumber)
            .ToListAsync();

        Assert.NotEmpty(transactions);
        Assert.All(transactions, t => Assert.Equal(5_000m, t.AmountInLocalCurrency));
        Assert.All(transactions, t => Assert.Equal(result.DocumentNumber, t.ReferenceDocumentNumber));

        // What the asset gained this year, summed across its depreciation
        // areas - the balance an acquisition is supposed to move.
        async Task<decimal> CurrentYearAcquisitionsAsync()
        {
            using var valueScope = fixture.CreateScope();
            var valueContext = valueScope.ServiceProvider.GetRequiredService<ErpDbContext>();

            var assetId = await valueContext.Set<Asset>()
                .Where(a => a.AssetNumber == Asset)
                .Select(a => a.Id)
                .SingleAsync();

            return await valueContext.Set<AssetValue>()
                .Where(v => v.AssetId == assetId
                            && v.FiscalYear == 2026
                            && v.DepreciationAreaId == areaId)
                .SumAsync(v => v.CurrentYearAcquisitions);
        }
    }

    [RequiresDatabaseFact]
    public async Task A_depreciation_run_can_be_previewed_without_posting_anything()
    {
        using var scope = fixture.CreateScope();
        var service = scope.ServiceProvider.GetRequiredService<IAssetService>();
        var context = scope.ServiceProvider.GetRequiredService<ErpDbContext>();

        var before = await context.Set<DepreciationRun>().CountAsync();

        var plan = await service.PlanDepreciationAsync(
            new DepreciationRunRequest(CompanyCode, 2026, 3));

        Assert.Empty(plan.Violations);
        Assert.NotEmpty(plan.Items);
        Assert.All(plan.Items, i => Assert.True(i.Amount >= 0m));
        Assert.Equal(plan.Items.Sum(i => i.Amount), plan.Total);

        using var reader = fixture.CreateScope();
        Assert.Equal(
            before,
            await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
                .Set<DepreciationRun>().CountAsync());
    }

    [RequiresDatabaseFact]
    public async Task A_depreciation_run_posts_once_and_refuses_to_run_again()
    {
        using var scope = fixture.CreateScope();

        // A period nothing has run yet, so the test does not depend on being
        // the first thing to touch this database. Re-running the suite against
        // the same server used to fail here for exactly that reason.
        var period = await NextUnusedPeriodAsync();
        var request = new DepreciationRunRequest(CompanyCode, 2026, period);

        var first = await scope.ServiceProvider.GetRequiredService<IAssetService>()
            .RunDepreciationAsync(request);

        Assert.True(first.IsSuccess, string.Join("; ",
            first.Violations.Select(v => v.ToString())
                .Concat(first.PostingErrors.Select(e => e.ToString()))));
        Assert.True(first.AssetsProcessed > 0);

        using var again = fixture.CreateScope();
        var second = await again.ServiceProvider.GetRequiredService<IAssetService>()
            .RunDepreciationAsync(request);

        Assert.False(second.IsSuccess);

        using var reader = fixture.CreateScope();
        var runs = await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<DepreciationRun>()
            .CountAsync(r => r.FiscalYear == 2026 && r.FiscalPeriod == period);

        Assert.Equal(1, runs);
    }

    /// <summary>The asset's lowest depreciation area - the book one.</summary>
    private async Task<long> BookAreaIdAsync()
    {
        using var scope = fixture.CreateScope();
        var context = scope.ServiceProvider.GetRequiredService<ErpDbContext>();

        var assetId = await context.Set<Asset>()
            .Where(a => a.AssetNumber == Asset)
            .Select(a => a.Id)
            .SingleAsync();

        return await context.Set<AssetValue>()
            .Where(v => v.AssetId == assetId && v.FiscalYear == 2026)
            .Select(v => v.DepreciationAreaId)
            .OrderBy(id => id)
            .FirstAsync();
    }

    /// <summary>A 2026 period no depreciation run has used yet.</summary>
    private async Task<byte> NextUnusedPeriodAsync()
    {
        using var scope = fixture.CreateScope();
        var used = await scope.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<DepreciationRun>()
            .Where(r => r.FiscalYear == 2026)
            .Select(r => r.FiscalPeriod)
            .ToListAsync();

        for (byte period = 1; period <= 12; period++)
        {
            if (!used.Contains(period))
            {
                return period;
            }
        }

        throw new InvalidOperationException(
            "Every 2026 period has been run; reset the database with " +
            "tools/start_test_database.sh --reset.");
    }

    private static string Explain(AssetPostingResult result) =>
        string.Join("; ",
            result.Violations.Select(v => v.ToString())
                .Concat(result.PostingErrors.Select(e => e.ToString())));
}
