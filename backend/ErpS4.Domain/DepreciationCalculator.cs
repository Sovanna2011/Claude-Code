namespace ErpS4.Domain;

/// <summary>How the depreciable amount is spread over the useful life.</summary>
public enum DepreciationMethod
{
    StraightLine = 0,
    DecliningBalance = 1,

    /// <summary>Written off in full in the period it is capitalised (low value assets).</summary>
    Immediate = 2,

    /// <summary>The system plans nothing; a person posts each amount.</summary>
    Manual = 3,
}

/// <summary>Everything needed to plan one period of depreciation for one area.</summary>
/// <param name="AcquisitionValue">Capitalised cost, including subsequent acquisitions.</param>
/// <param name="AccumulatedDepreciation">Posted so far, as a positive number.</param>
/// <param name="ScrapValue">Floor: depreciation stops here, never at zero.</param>
/// <param name="DepreciationStart">
/// First day the asset depreciates. Before this date the planned amount is
/// zero, and in the period containing it the charge is pro rata.
/// </param>
/// <param name="DeclineFactor">Multiplier on the straight line rate, e.g. 2 for double declining.</param>
/// <param name="SwitchToStraightLine">
/// Declining balance eventually charges less than straight line would; when
/// this is set, the calculation takes whichever is larger, which is what stops
/// an asset depreciating forever.
/// </param>
public sealed record DepreciationBasis(
    decimal AcquisitionValue,
    decimal AccumulatedDepreciation,
    decimal ScrapValue,
    int UsefulLifeYears,
    int UsefulLifePeriods,
    DateOnly DepreciationStart,
    byte PeriodsPerYear = 12,
    DepreciationMethod Method = DepreciationMethod.StraightLine,
    decimal DeclineFactor = 1m,
    bool SwitchToStraightLine = true)
{
    /// <summary>Total periods of life; zero means the asset never depreciates.</summary>
    public int TotalLifePeriods => (UsefulLifeYears * PeriodsPerYear) + UsefulLifePeriods;

    public decimal NetBookValue => AcquisitionValue - AccumulatedDepreciation;

    /// <summary>What is still available to write off before hitting the floor.</summary>
    public decimal RemainingDepreciable => NetBookValue - ScrapValue;
}

/// <param name="Amount">Charge for the period, never negative.</param>
/// <param name="NetBookValueAfter">Net book value once this charge is posted.</param>
/// <param name="IsFinalCharge">True when this charge takes the asset to its scrap value.</param>
/// <param name="Basis">Short explanation, shown on the depreciation preview.</param>
public sealed record PeriodDepreciation(
    decimal Amount,
    decimal NetBookValueAfter,
    bool IsFinalCharge,
    string Basis);

/// <summary>
/// Plans depreciation for one asset, one area, one period.
/// </summary>
/// <remarks>
/// Pure arithmetic on purpose: no database, no configuration lookup, no
/// posting. Depreciation is the part of asset accounting an auditor
/// recalculates by hand, so it has to be verifiable in isolation.
/// </remarks>
public static class DepreciationCalculator
{
    /// <summary>
    /// Charge for the period ending on <paramref name="periodEnd"/>.
    /// </summary>
    /// <param name="periodStart">First day of the fiscal period.</param>
    /// <param name="periodEnd">Last day of the fiscal period.</param>
    /// <param name="decimals">Decimal places of the area currency.</param>
    public static PeriodDepreciation Calculate(
        DepreciationBasis basis,
        DateOnly periodStart,
        DateOnly periodEnd,
        int decimals = 2)
    {
        if (basis.Method == DepreciationMethod.Manual)
        {
            return None(basis, "Manual depreciation key: nothing is planned.");
        }

        if (basis.DepreciationStart > periodEnd)
        {
            return None(basis, "The asset is not yet in service in this period.");
        }

        if (basis.RemainingDepreciable <= 0m)
        {
            return None(basis, "The asset has reached its scrap value.");
        }

        if (basis.Method == DepreciationMethod.Immediate)
        {
            return Charge(basis, basis.RemainingDepreciable, decimals, "Immediate write-off.");
        }

        if (basis.TotalLifePeriods <= 0)
        {
            return None(basis, "No useful life is maintained, so nothing can be planned.");
        }

        var straightLine = (basis.AcquisitionValue - basis.ScrapValue) / basis.TotalLifePeriods;

        var amount = basis.Method switch
        {
            DepreciationMethod.DecliningBalance => Declining(basis, straightLine),
            _ => straightLine,
        };

        // Period control "pro rata temporis": in the period the asset enters
        // service it earns only the days it was actually in service.
        var proRata = ProRataFactor(basis.DepreciationStart, periodStart, periodEnd);
        amount *= proRata;

        var reason = basis.Method == DepreciationMethod.DecliningBalance
            ? $"Declining balance at {basis.DeclineFactor:0.##}x"
            : "Straight line";

        if (proRata < 1m)
        {
            reason += $", pro rata {proRata:P0} from {basis.DepreciationStart:yyyy-MM-dd}";
        }

        return Charge(basis, amount, decimals, reason + ".");
    }

    /// <summary>
    /// Declining balance on the net book value, switching to straight line on
    /// the remaining life once that charges more.
    /// </summary>
    private static decimal Declining(DepreciationBasis basis, decimal straightLine)
    {
        var annualRate = basis.DeclineFactor / basis.UsefulLifeYears;
        var declining = basis.NetBookValue * annualRate / basis.PeriodsPerYear;

        return basis.SwitchToStraightLine && straightLine > declining ? straightLine : declining;
    }

    /// <summary>
    /// 1 for a full period; the share of the period the asset was in service
    /// when depreciation starts inside it.
    /// </summary>
    private static decimal ProRataFactor(DateOnly start, DateOnly periodStart, DateOnly periodEnd)
    {
        if (start <= periodStart)
        {
            return 1m;
        }

        var daysInPeriod = periodEnd.DayNumber - periodStart.DayNumber + 1;
        var daysInService = periodEnd.DayNumber - start.DayNumber + 1;

        if (daysInPeriod <= 0 || daysInService <= 0)
        {
            return 0m;
        }

        return (decimal)daysInService / daysInPeriod;
    }

    private static PeriodDepreciation Charge(
        DepreciationBasis basis,
        decimal amount,
        int decimals,
        string reason)
    {
        var rounded = Math.Round(amount, decimals, MidpointRounding.AwayFromZero);

        // The last charge is whatever is left: rounding must not leave a few
        // cents on the books forever, nor take the asset below its floor.
        var capped = Math.Min(rounded, Math.Round(basis.RemainingDepreciable, decimals));
        var isFinal = capped >= Math.Round(basis.RemainingDepreciable, decimals);

        return new PeriodDepreciation(
            capped,
            basis.NetBookValue - capped,
            isFinal,
            isFinal ? reason + " Final charge: the remaining book value." : reason);
    }

    private static PeriodDepreciation None(DepreciationBasis basis, string reason) =>
        new(0m, basis.NetBookValue, false, reason);
}
