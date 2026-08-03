namespace PoApproval.Api.Services;

/// <summary>
/// Helpers for interpreting the SAP release status FRGZU, which is a plain
/// concatenation of the two-character release codes that have been effected
/// (e.g. "0102" = codes 01 and 02 released).
/// </summary>
public static class ReleaseCodes
{
    /// <summary>Splits FRGZU into the set of effected two-character codes.</summary>
    public static HashSet<string> Parse(string? frgzu)
    {
        var set = new HashSet<string>(StringComparer.Ordinal);
        if (string.IsNullOrEmpty(frgzu)) return set;
        for (var i = 0; i + 2 <= frgzu.Length; i += 2)
            set.Add(frgzu.Substring(i, 2));
        return set;
    }

    /// <summary>True when the code has already been effected in FRGZU.</summary>
    public static bool IsReleased(string? frgzu, string code) => Parse(frgzu).Contains(code);
}
