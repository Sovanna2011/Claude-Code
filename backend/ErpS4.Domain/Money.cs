namespace ErpS4.Domain;

/// <summary>
/// An amount in one currency. Debits are positive, credits negative - the same
/// convention the universal journal stores.
/// </summary>
/// <remarks>
/// Rounding is a currency property, not a global one: KHR and JPY have no minor
/// unit, so rounding a Riel amount to two decimals would invent precision that
/// the currency does not have.
/// </remarks>
public readonly record struct Money
{
    public Money(decimal amount, string currency)
    {
        if (string.IsNullOrWhiteSpace(currency))
        {
            throw new ArgumentException("Currency is required.", nameof(currency));
        }

        Amount = amount;
        Currency = currency;
    }

    public decimal Amount { get; }

    public string Currency { get; }

    public bool IsDebit => Amount > 0m;

    public bool IsCredit => Amount < 0m;

    public bool IsZero => Amount == 0m;

    public static Money Zero(string currency) => new(0m, currency);

    /// <summary>Rounds half away from zero, the commercial rule.</summary>
    public Money Round(int decimalPlaces) =>
        new(Math.Round(Amount, decimalPlaces, MidpointRounding.AwayFromZero), Currency);

    public Money Negate() => new(-Amount, Currency);

    public Money Abs() => new(Math.Abs(Amount), Currency);

    public static Money operator +(Money left, Money right)
    {
        EnsureSameCurrency(left, right);
        return new Money(left.Amount + right.Amount, left.Currency);
    }

    public static Money operator -(Money left, Money right)
    {
        EnsureSameCurrency(left, right);
        return new Money(left.Amount - right.Amount, left.Currency);
    }

    public static Money operator *(Money value, decimal factor) =>
        new(value.Amount * factor, value.Currency);

    private static void EnsureSameCurrency(Money left, Money right)
    {
        if (!string.Equals(left.Currency, right.Currency, StringComparison.Ordinal))
        {
            throw new InvalidOperationException(
                $"Cannot combine {left.Currency} and {right.Currency}: convert first.");
        }
    }

    public override string ToString() => $"{Amount:0.####} {Currency}";
}
