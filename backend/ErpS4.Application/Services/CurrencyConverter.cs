using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Application.Services;

/// <summary>Converts an amount between currencies using the configured rates.</summary>
public interface ICurrencyConverter
{
    /// <summary>
    /// Converts at the rate valid on <paramref name="date"/>. Returns null when
    /// no rate is configured - the caller turns that into a posting error with
    /// the currency pair in the message, rather than throwing.
    /// </summary>
    Task<CurrencyConversion?> ConvertAsync(
        Money amount,
        string toCurrency,
        DateOnly date,
        string rateType = "M",
        CancellationToken cancellationToken = default);

    /// <summary>Decimal places of a currency; 0 for KHR and JPY, 2 for most.</summary>
    Task<int> GetDecimalsAsync(string currency, CancellationToken cancellationToken = default);
}

/// <param name="Amount">The converted amount, unrounded.</param>
/// <param name="Rate">Rate applied, before ratios.</param>
/// <param name="WasInverted">True when the inverse of a configured rate was used.</param>
public sealed record CurrencyConversion(Money Amount, decimal Rate, bool WasInverted);

/// <inheritdoc />
public sealed class CurrencyConverter(IErpDataContext context, ITenantProvider tenantProvider)
    : ICurrencyConverter
{
    private readonly Dictionary<string, int> _decimalCache = new(StringComparer.Ordinal);

    public async Task<CurrencyConversion?> ConvertAsync(
        Money amount,
        string toCurrency,
        DateOnly date,
        string rateType = "M",
        CancellationToken cancellationToken = default)
    {
        if (string.Equals(amount.Currency, toCurrency, StringComparison.Ordinal))
        {
            return new CurrencyConversion(amount, 1m, WasInverted: false);
        }

        var tenantId = tenantProvider.TenantId;

        var direct = await (
            from rate in context.Query<ExchangeRate>()
            join type in context.Query<ExchangeRateType>()
                on rate.ExchangeRateTypeId equals type.Id
            where rate.TenantId == tenantId
                  && type.ExchangeRateTypeCode == rateType
                  && rate.FromCurrencyCode == amount.Currency
                  && rate.ToCurrencyCode == toCurrency
                  && rate.ValidFrom <= date
            orderby rate.ValidFrom descending
            select new { rate.Rate, rate.FromRatio, rate.ToRatio }
        ).FirstOrDefaultAsync(cancellationToken);

        if (direct is not null)
        {
            var converted = amount.Amount * direct.Rate * direct.ToRatio / direct.FromRatio;
            return new CurrencyConversion(
                new Money(converted, toCurrency), direct.Rate, WasInverted: false);
        }

        // Falling back to the inverse is only legitimate when the rate type says
        // so; some types are one-directional on purpose (buying versus selling).
        var inverse = await (
            from rate in context.Query<ExchangeRate>()
            join type in context.Query<ExchangeRateType>()
                on rate.ExchangeRateTypeId equals type.Id
            where rate.TenantId == tenantId
                  && type.ExchangeRateTypeCode == rateType
                  && type.IsInversionAllowed
                  && rate.FromCurrencyCode == toCurrency
                  && rate.ToCurrencyCode == amount.Currency
                  && rate.ValidFrom <= date
            orderby rate.ValidFrom descending
            select new { rate.Rate, rate.FromRatio, rate.ToRatio }
        ).FirstOrDefaultAsync(cancellationToken);

        if (inverse is null || inverse.Rate == 0m)
        {
            return null;
        }

        var invertedAmount = amount.Amount / inverse.Rate * inverse.FromRatio / inverse.ToRatio;
        return new CurrencyConversion(
            new Money(invertedAmount, toCurrency), 1m / inverse.Rate, WasInverted: true);
    }

    public async Task<int> GetDecimalsAsync(
        string currency,
        CancellationToken cancellationToken = default)
    {
        if (_decimalCache.TryGetValue(currency, out var cached))
        {
            return cached;
        }

        var decimals = await context.Query<Currency>()
            .AsNoTracking()
            .Where(c => c.TenantId == tenantProvider.TenantId && c.CurrencyCode == currency)
            .Select(c => (int?)c.DecimalPlaces)
            .FirstOrDefaultAsync(cancellationToken);

        var result = decimals ?? 2;
        _decimalCache[currency] = result;
        return result;
    }
}
