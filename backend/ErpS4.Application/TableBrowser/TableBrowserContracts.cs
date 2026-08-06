using ErpS4.Domain;

namespace ErpS4.Application.TableBrowser;

/// <summary>
/// SE16N-style table browser (design prompt, section 11): read-only, authorised
/// querying of the tables a user is allowed to see.
/// </summary>
/// <remarks>
/// The security posture is the feature. Table and field names come from the
/// data dictionary, never from the request; every value the caller supplies
/// travels as a parameter; tenant isolation is added by the service, not asked
/// for by the caller; results are capped; masked fields stay masked; and every
/// query is logged with who ran it and what came back.
/// </remarks>
public interface ITableBrowserService
{
    /// <summary>Runs a browser query, or explains why it will not.</summary>
    Task<BrowserQueryResult> QueryAsync(
        BrowserQueryRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>Tables the signed-in user may browse.</summary>
    Task<IReadOnlyList<BrowsableTable>> GetBrowsableTablesAsync(
        string? search = null,
        CancellationToken cancellationToken = default);
}

/// <param name="Fields">Columns to return; empty means every readable column.</param>
/// <param name="MaxRows">Requested cap, itself capped by the authorization group.</param>
/// <param name="IsExport">
/// Export needs its own permission and is logged separately: taking data out is
/// a different act from looking at it.
/// </param>
public sealed record BrowserQueryRequest(string SchemaName, string TableName)
{
    public IReadOnlyList<string> Fields { get; init; } = [];

    public IReadOnlyList<BrowserFilter> Filters { get; init; } = [];

    public string? SortBy { get; init; }

    public bool SortDescending { get; init; }

    public int PageNumber { get; init; } = 1;

    public int MaxRows { get; init; } = 100;

    public bool IsExport { get; init; }

    /// <summary>Restrict to one company code, on top of the user's own access.</summary>
    public string? CompanyCode { get; init; }
}

/// <param name="Operator">One of <see cref="BrowserOperators"/>.</param>
/// <param name="Exclude">Turns the condition into NOT (…), the SE16N "exclude" flag.</param>
public sealed record BrowserFilter(string Field, string Operator, string? Value = null)
{
    public string? HighValue { get; init; }

    public IReadOnlyList<string> Values { get; init; } = [];

    public bool Exclude { get; init; }
}

/// <summary>The operators the browser understands. Anything else is refused.</summary>
public static class BrowserOperators
{
    public const string Equals = "EQ";
    public const string NotEquals = "NE";
    public const string GreaterThan = "GT";
    public const string GreaterOrEqual = "GE";
    public const string LessThan = "LT";
    public const string LessOrEqual = "LE";
    public const string Between = "BT";
    public const string Contains = "CP";
    public const string StartsWith = "SW";
    public const string EndsWith = "EW";
    public const string IsEmpty = "NULL";
    public const string IsNotEmpty = "NN";
    public const string In = "IN";
    public const string NotIn = "NI";

    public static readonly IReadOnlySet<string> All = new HashSet<string>(StringComparer.Ordinal)
    {
        Equals, NotEquals, GreaterThan, GreaterOrEqual, LessThan, LessOrEqual,
        Between, Contains, StartsWith, EndsWith, IsEmpty, IsNotEmpty, In, NotIn,
    };

    /// <summary>Operators that need no value at all.</summary>
    public static bool IsUnary(string @operator) =>
        @operator is IsEmpty or IsNotEmpty;
}

public sealed record BrowserQueryResult
{
    public bool IsSuccess => Violations.All(v => !v.IsBlocking);

    public string SchemaName { get; init; } = string.Empty;

    public string TableName { get; init; } = string.Empty;

    public IReadOnlyList<BrowserColumn> Columns { get; init; } = [];

    public IReadOnlyList<IReadOnlyList<string?>> Rows { get; init; } = [];

    public int RowCount => Rows.Count;

    /// <summary>True when the row cap cut the result short.</summary>
    public bool WasTruncated { get; init; }

    public int RowLimitApplied { get; init; }

    public int DurationMilliseconds { get; init; }

    /// <summary>The statement that ran, with parameter markers - never values.</summary>
    public string? ExecutedSql { get; init; }

    public IReadOnlyList<RuleViolation> Violations { get; init; } = [];
}

/// <param name="IsMasked">Masked columns are returned as asterisks, whoever asks.</param>
public sealed record BrowserColumn(
    string FieldName,
    string DataType,
    string? Description,
    bool IsKey,
    bool IsMasked);

public sealed record BrowsableTable(
    string SchemaName,
    string TableName,
    string Description,
    string TableCategory,
    string? AuthorizationGroup,
    int MaxRowsPerQuery,
    bool AllowExport);

public static class BrowserErrorCodes
{
    public const string TableUnknown = "SE16N.TABLE_UNKNOWN";
    public const string UserUnknown = "SE16N.USER_UNKNOWN";
    public const string TableProtected = "SE16N.TABLE_PROTECTED";
    public const string NotAuthorizedForTable = "SE16N.TABLE_NOT_AUTHORIZED";
    public const string FieldUnknown = "SE16N.FIELD_UNKNOWN";
    public const string FieldMasked = "SE16N.FIELD_MASKED";
    public const string OperatorUnknown = "SE16N.OPERATOR_UNKNOWN";
    public const string ValueRequired = "SE16N.VALUE_REQUIRED";
    public const string TooManyValues = "SE16N.TOO_MANY_VALUES";
    public const string ValueInvalid = "SE16N.VALUE_INVALID";
    public const string PatternOnNonTextField = "SE16N.PATTERN_ON_NON_TEXT_FIELD";
    public const string ExportNotAllowed = "SE16N.EXPORT_NOT_ALLOWED";
    public const string SortFieldUnknown = "SE16N.SORT_FIELD_UNKNOWN";
    public const string CompanyCodeNotApplicable = "SE16N.COMPANY_CODE_NOT_APPLICABLE";
    public const string CompanyCodeUnknown = "SE16N.COMPANY_CODE_UNKNOWN";
}
