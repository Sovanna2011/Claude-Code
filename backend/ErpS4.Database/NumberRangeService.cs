using ErpS4.Database.Entities;
using Microsoft.Data.SqlClient;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Database;

/// <summary>Draws the next number from a number range interval.</summary>
public interface INumberRangeService
{
    /// <summary>
    /// Reserves the next number and returns it formatted, e.g.
    /// <c>KSS-2026-SA-0000000123</c>.
    /// </summary>
    /// <exception cref="NumberRangeExhaustedException">The interval is used up.</exception>
    Task<string> NextAsync(
        string numberRangeObject,
        string numberRangeCode,
        long? companyCodeId,
        short fiscalYear,
        string? documentType = null,
        CancellationToken cancellationToken = default);
}

public sealed class NumberRangeExhaustedException(string numberRangeObject, string numberRangeCode)
    : Exception($"Number range {numberRangeObject}/{numberRangeCode} is exhausted.")
{
    public string NumberRangeObject { get; } = numberRangeObject;

    public string NumberRangeCode { get; } = numberRangeCode;
}

/// <summary>
/// SQL Server implementation. The number is drawn with a single UPDATE ...
/// OUTPUT so two concurrent posts can never receive the same value: the row
/// lock is held for the duration of that statement and released immediately,
/// which keeps the serialisation point as short as possible.
/// </summary>
/// <remarks>
/// A number drawn by a transaction that later rolls back is lost, which is
/// correct - gaps are legal, duplicates are not. The gap is recorded in
/// <c>cfg.NumberRangeGap</c> by the posting engine so an auditor can see why a
/// number is missing.
/// </remarks>
public sealed class NumberRangeService(ErpDbContext context, ITenantProvider tenantProvider)
    : INumberRangeService
{
    private const string DrawSql = """
        UPDATE  i
        SET     i.CurrentNumber = i.CurrentNumber + 1
        OUTPUT  inserted.CurrentNumber AS Value
        FROM    cfg.NumberRangeInterval AS i
        WHERE   i.Id = @intervalId
                AND i.IsBlocked = 0
                AND i.CurrentNumber < i.ToNumber;
        """;

    public async Task<string> NextAsync(
        string numberRangeObject,
        string numberRangeCode,
        long? companyCodeId,
        short fiscalYear,
        string? documentType = null,
        CancellationToken cancellationToken = default)
    {
        var tenantId = tenantProvider.TenantId;

        var definition = await (
            from o in context.NumberRangeObject
            join i in context.NumberRangeInterval on o.Id equals i.NumberRangeObjectId
            where o.TenantId == tenantId
                  && o.NumberRangeObjectCode == numberRangeObject
                  && i.NumberRangeCode == numberRangeCode
                  && (i.CompanyCodeId == companyCodeId || i.CompanyCodeId == null)
                  && (i.FiscalYear == fiscalYear || i.FiscalYear == 0)
            orderby i.CompanyCodeId descending, i.FiscalYear descending
            select new
            {
                IntervalId = i.Id,
                o.Prefix,
                o.NumberFormat,
                o.NumberLength,
                i.IsExternal,
            }).FirstOrDefaultAsync(cancellationToken);

        if (definition is null)
        {
            throw new InvalidOperationException(
                $"No number range interval {numberRangeObject}/{numberRangeCode} for " +
                $"company code {companyCodeId} and fiscal year {fiscalYear}.");
        }

        if (definition.IsExternal)
        {
            throw new InvalidOperationException(
                $"Number range {numberRangeObject}/{numberRangeCode} is external: " +
                "the caller has to supply the number.");
        }

        var drawn = await context.Database
            .SqlQueryRaw<long>(DrawSql, new SqlParameter("@intervalId", definition.IntervalId))
            .ToListAsync(cancellationToken);

        if (drawn.Count == 0)
        {
            throw new NumberRangeExhaustedException(numberRangeObject, numberRangeCode);
        }

        return Format(
            drawn[0],
            definition.Prefix,
            definition.NumberFormat,
            definition.NumberLength,
            fiscalYear,
            documentType);
    }

    /// <summary>
    /// Applies the format mask, e.g.
    /// <c>{Prefix}-{Year}-{Type}-{Number:0000000000}</c>. Without a mask the
    /// number is simply padded to the configured length.
    /// </summary>
    public static string Format(
        long number,
        string? prefix,
        string? format,
        byte numberLength,
        short fiscalYear,
        string? documentType)
    {
        var padded = number.ToString(new string('0', Math.Max(numberLength, (byte)1)));

        if (string.IsNullOrWhiteSpace(format))
        {
            return string.IsNullOrWhiteSpace(prefix) ? padded : $"{prefix}{padded}";
        }

        return format
            .Replace("{Prefix}", prefix ?? string.Empty, StringComparison.Ordinal)
            .Replace("{Year}", fiscalYear.ToString("0000"), StringComparison.Ordinal)
            .Replace("{Type}", documentType ?? string.Empty, StringComparison.Ordinal)
            .Replace($"{{Number:{new string('0', numberLength)}}}", padded, StringComparison.Ordinal)
            .Replace("{Number}", padded, StringComparison.Ordinal);
    }
}
