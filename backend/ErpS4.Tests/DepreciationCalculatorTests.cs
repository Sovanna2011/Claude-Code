using ErpS4.Domain;
using Xunit;

namespace ErpS4.Tests;

/// <summary>
/// Depreciation is the part of asset accounting an auditor recalculates by
/// hand, so it is tested as arithmetic: no database, no configuration, no
/// posting.
/// </summary>
public sealed class DepreciationCalculatorTests
{
    private static readonly DateOnly PeriodStart = new(2026, 3, 1);
    private static readonly DateOnly PeriodEnd = new(2026, 3, 31);

    private static DepreciationBasis Basis(
        decimal acquisition = 120_000m,
        decimal accumulated = 0m,
        decimal scrap = 0m,
        int years = 10,
        DateOnly? start = null,
        DepreciationMethod method = DepreciationMethod.StraightLine,
        decimal factor = 1m) =>
        new(acquisition, accumulated, scrap, years, 0,
            start ?? new DateOnly(2026, 1, 1), 12, method, factor);

    [Fact]
    public void Straight_line_spreads_the_cost_evenly_over_the_useful_life()
    {
        // 120,000 over 10 years is 1,000 a month.
        var charge = DepreciationCalculator.Calculate(Basis(), PeriodStart, PeriodEnd);

        Assert.Equal(1_000m, charge.Amount);
        Assert.Equal(119_000m, charge.NetBookValueAfter);
        Assert.False(charge.IsFinalCharge);
    }

    [Fact]
    public void The_scrap_value_is_a_floor_the_asset_never_goes_below()
    {
        // Only 20,000 of the 120,000 is depreciable.
        var charge = DepreciationCalculator.Calculate(
            Basis(scrap: 100_000m), PeriodStart, PeriodEnd);

        Assert.Equal(166.67m, charge.Amount);

        var nearlyFinished = DepreciationCalculator.Calculate(
            Basis(accumulated: 19_900m, scrap: 100_000m), PeriodStart, PeriodEnd);

        Assert.Equal(100m, nearlyFinished.Amount);
        Assert.True(nearlyFinished.IsFinalCharge);
        Assert.Equal(100_000m, nearlyFinished.NetBookValueAfter);
    }

    [Fact]
    public void A_fully_depreciated_asset_charges_nothing_more()
    {
        var charge = DepreciationCalculator.Calculate(
            Basis(accumulated: 120_000m), PeriodStart, PeriodEnd);

        Assert.Equal(0m, charge.Amount);
        Assert.Contains("scrap value", charge.Basis, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void An_asset_not_yet_in_service_charges_nothing()
    {
        var charge = DepreciationCalculator.Calculate(
            Basis(start: new DateOnly(2026, 6, 1)), PeriodStart, PeriodEnd);

        Assert.Equal(0m, charge.Amount);
        Assert.Equal(120_000m, charge.NetBookValueAfter);
    }

    [Fact]
    public void The_first_period_is_charged_pro_rata_from_the_day_it_enters_service()
    {
        // In service on 16 March: 16 of 31 days.
        var charge = DepreciationCalculator.Calculate(
            Basis(start: new DateOnly(2026, 3, 16)), PeriodStart, PeriodEnd);

        Assert.Equal(Math.Round(1_000m * 16 / 31, 2), charge.Amount);
        Assert.Contains("pro rata", charge.Basis, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void Declining_balance_charges_on_the_book_value_not_the_cost()
    {
        // Double declining over 10 years is 20% a year of the net book value.
        var charge = DepreciationCalculator.Calculate(
            Basis(accumulated: 20_000m, method: DepreciationMethod.DecliningBalance, factor: 2m),
            PeriodStart,
            PeriodEnd);

        Assert.Equal(Math.Round(100_000m * 0.2m / 12, 2), charge.Amount);
    }

    [Fact]
    public void Declining_balance_switches_to_straight_line_once_that_charges_more()
    {
        // Late in life the declining charge falls below the straight line one;
        // without the switch the asset would never finish depreciating.
        var charge = DepreciationCalculator.Calculate(
            Basis(accumulated: 110_000m, method: DepreciationMethod.DecliningBalance, factor: 2m),
            PeriodStart,
            PeriodEnd);

        Assert.Equal(1_000m, charge.Amount);
    }

    [Fact]
    public void An_immediate_key_writes_the_asset_off_in_one_period()
    {
        var charge = DepreciationCalculator.Calculate(
            Basis(acquisition: 400m, method: DepreciationMethod.Immediate), PeriodStart, PeriodEnd);

        Assert.Equal(400m, charge.Amount);
        Assert.True(charge.IsFinalCharge);
        Assert.Equal(0m, charge.NetBookValueAfter);
    }

    [Fact]
    public void A_manual_key_plans_nothing()
    {
        var charge = DepreciationCalculator.Calculate(
            Basis(method: DepreciationMethod.Manual), PeriodStart, PeriodEnd);

        Assert.Equal(0m, charge.Amount);
        Assert.Contains("Manual", charge.Basis, StringComparison.Ordinal);
    }

    [Fact]
    public void Without_a_useful_life_nothing_can_be_planned()
    {
        var charge = DepreciationCalculator.Calculate(Basis(years: 0), PeriodStart, PeriodEnd);

        Assert.Equal(0m, charge.Amount);
        Assert.Contains("useful life", charge.Basis, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void The_last_charge_takes_exactly_what_is_left_over()
    {
        // 1,000 a month would overshoot; only 250 remains.
        var charge = DepreciationCalculator.Calculate(
            Basis(accumulated: 119_750m), PeriodStart, PeriodEnd);

        Assert.Equal(250m, charge.Amount);
        Assert.True(charge.IsFinalCharge);
        Assert.Equal(0m, charge.NetBookValueAfter);
    }

    [Fact]
    public void Rounding_follows_the_currency_precision_given_to_it()
    {
        // A Riel area has no minor unit: 120,000 over 7 years, whole Riel only.
        var charge = DepreciationCalculator.Calculate(
            Basis(years: 7), PeriodStart, PeriodEnd, decimals: 0);

        Assert.Equal(1_429m, charge.Amount);
    }
}
