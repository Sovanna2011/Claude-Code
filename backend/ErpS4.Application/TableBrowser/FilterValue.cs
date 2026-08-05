using System.Globalization;

namespace ErpS4.Application.TableBrowser;

/// <summary>
/// Turns the text a user typed into the CLR type the column actually holds.
/// </summary>
/// <remarks>
/// Without this every filter value would reach SQL Server as an
/// <c>nvarchar</c> parameter and rely on implicit conversion. That mostly
/// works, and fails in two ways that matter: a value that is not a date or a
/// number raises a conversion error (message 241 or 8114) as a
/// <c>SqlException</c>, which is a 500 for what is really a typing mistake;
/// and the parsing follows the session's language and DATEFORMAT settings, so
/// the same filter can mean different days on different servers. Parsing here,
/// invariantly, makes a bad value a reported violation and a good one
/// unambiguous.
/// </remarks>
public static class FilterValue
{
    /// <summary>Text-valued column types, the only ones LIKE makes sense on.</summary>
    public static bool IsText(string sqlType) =>
        BaseType(sqlType) is "nvarchar" or "varchar" or "nchar" or "char" or "text" or "ntext";

    /// <summary>
    /// Parses <paramref name="text"/> for a column of <paramref name="sqlType"/>.
    /// Returns false when the text cannot be that type.
    /// </summary>
    public static bool TryConvert(string sqlType, string? text, out object? value)
    {
        value = null;

        if (text is null)
        {
            return true;
        }

        var invariant = CultureInfo.InvariantCulture;

        switch (BaseType(sqlType))
        {
            case "nvarchar" or "varchar" or "nchar" or "char" or "text" or "ntext":
                value = text;
                return true;

            case "bigint":
                if (long.TryParse(text, NumberStyles.Integer, invariant, out var asLong))
                {
                    value = asLong;
                    return true;
                }

                return false;

            case "int":
                if (int.TryParse(text, NumberStyles.Integer, invariant, out var asInt))
                {
                    value = asInt;
                    return true;
                }

                return false;

            case "smallint":
                if (short.TryParse(text, NumberStyles.Integer, invariant, out var asShort))
                {
                    value = asShort;
                    return true;
                }

                return false;

            case "tinyint":
                if (byte.TryParse(text, NumberStyles.Integer, invariant, out var asByte))
                {
                    value = asByte;
                    return true;
                }

                return false;

            case "bit":
                // "1" and "0" as well as "true" and "false": a checkbox column
                // is written both ways depending on who is asking.
                if (bool.TryParse(text, out var asBool))
                {
                    value = asBool;
                    return true;
                }

                if (text is "1" or "0")
                {
                    value = text == "1";
                    return true;
                }

                return false;

            case "decimal" or "numeric" or "money" or "smallmoney":
                if (decimal.TryParse(text, NumberStyles.Number, invariant, out var asDecimal))
                {
                    value = asDecimal;
                    return true;
                }

                return false;

            case "float" or "real":
                if (double.TryParse(text, NumberStyles.Float, invariant, out var asDouble))
                {
                    value = asDouble;
                    return true;
                }

                return false;

            case "date":
                if (DateOnly.TryParse(text, invariant, DateTimeStyles.None, out var asDate))
                {
                    value = asDate;
                    return true;
                }

                return false;

            case "datetime2" or "datetime" or "smalldatetime" or "datetimeoffset":
                if (DateTime.TryParse(
                        text, invariant, DateTimeStyles.RoundtripKind, out var asDateTime))
                {
                    value = asDateTime;
                    return true;
                }

                return false;

            case "time":
                if (TimeOnly.TryParse(text, invariant, DateTimeStyles.None, out var asTime))
                {
                    value = asTime;
                    return true;
                }

                return false;

            case "uniqueidentifier":
                if (Guid.TryParse(text, out var asGuid))
                {
                    value = asGuid;
                    return true;
                }

                return false;

            case "rowversion" or "timestamp" or "varbinary" or "binary":
                try
                {
                    value = Convert.FromHexString(text.StartsWith("0x", StringComparison.OrdinalIgnoreCase)
                        ? text[2..]
                        : text);
                    return true;
                }
                catch (FormatException)
                {
                    return false;
                }

            default:
                // An unrecognised type is passed through as text and left to
                // the server, which is what happened before this existed.
                value = text;
                return true;
        }
    }

    private static string BaseType(string sqlType)
    {
        var parenthesis = sqlType.IndexOf('(');
        var bare = parenthesis < 0 ? sqlType : sqlType[..parenthesis];
        return bare.Trim().ToLowerInvariant();
    }
}
