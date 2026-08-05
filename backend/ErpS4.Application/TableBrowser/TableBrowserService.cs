using System.Diagnostics;
using System.Text;
using System.Text.Json;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;

namespace ErpS4.Application.TableBrowser;

/// <inheritdoc />
public sealed class TableBrowserService(
    IErpDataContext context,
    ITenantProvider tenantProvider,
    ICurrentUser currentUser,
    TimeProvider timeProvider,
    ILogger<TableBrowserService> logger) : ITableBrowserService
{
    /// <summary>Nothing may return more than this, whatever the caller asks for.</summary>
    public const int AbsoluteRowLimit = 5_000;

    /// <summary>Values one IN or NOT IN filter may carry.</summary>
    public const int MaxValuesPerFilter = 500;

    private const string Mask = "********";

    private int TenantId => tenantProvider.TenantId;

    public async Task<IReadOnlyList<BrowsableTable>> GetBrowsableTablesAsync(
        string? search = null,
        CancellationToken cancellationToken = default)
    {
        var tables = await (
            from table in context.Query<DictionaryTable>().AsNoTracking()
            join authorizationGroup in context.Query<TableAuthorizationGroup>()
                on table.AuthorizationGroup equals authorizationGroup.AuthorizationGroup
                into groups
            from authorizationGroup in groups.DefaultIfEmpty()
            where table.TenantId == TenantId
                  && table.Status == "Active"
                  && (authorizationGroup == null || !authorizationGroup.IsSystemProtected)
                  && (search == null
                      || table.TableName.Contains(search)
                      || table.ShortDescription.Contains(search))
            orderby table.SchemaName, table.TableName
            select new BrowsableTable(
                table.SchemaName,
                table.TableName,
                table.ShortDescription,
                table.TableCategory,
                table.AuthorizationGroup,
                authorizationGroup == null ? 1_000 : authorizationGroup.MaxRowsPerQuery,
                authorizationGroup != null && authorizationGroup.AllowExport)
        ).ToListAsync(cancellationToken);

        return tables;
    }

    public async Task<BrowserQueryResult> QueryAsync(
        BrowserQueryRequest request,
        CancellationToken cancellationToken = default)
    {
        var stopwatch = Stopwatch.StartNew();

        var table = await context.Query<DictionaryTable>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                t => t.TenantId == TenantId
                     && t.SchemaName == request.SchemaName
                     && t.TableName == request.TableName,
                cancellationToken);

        if (table is null)
        {
            return Refused(request, BrowserErrorCodes.TableUnknown,
                $"{request.SchemaName}.{request.TableName} is not in the data dictionary.");
        }

        var authorizationGroup = table.AuthorizationGroup is null
            ? null
            : await context.Query<TableAuthorizationGroup>()
                .AsNoTracking()
                .FirstOrDefaultAsync(
                    g => g.TenantId == TenantId
                         && g.AuthorizationGroup == table.AuthorizationGroup,
                    cancellationToken);

        // Security and system tables are never browsable, whatever a user's
        // permissions say: the browser is not a back door into sec or audit.
        if (authorizationGroup?.IsSystemProtected == true || table.SchemaName == "sec")
        {
            return Refused(request, BrowserErrorCodes.TableProtected,
                $"{request.SchemaName}.{request.TableName} is protected and cannot be browsed.");
        }

        if (request.IsExport && authorizationGroup is not null && !authorizationGroup.AllowExport)
        {
            return Refused(request, BrowserErrorCodes.ExportNotAllowed,
                $"Authorization group {table.AuthorizationGroup} does not permit export.");
        }

        var fields = await context.Query<DictionaryTableField>()
            .AsNoTracking()
            .Where(f => f.TenantId == TenantId && f.DictionaryTableId == table.Id)
            .OrderBy(f => f.FieldPosition)
            .ToListAsync(cancellationToken);

        if (fields.Count == 0)
        {
            return Refused(request, BrowserErrorCodes.TableUnknown,
                $"{request.SchemaName}.{request.TableName} has no fields in the dictionary.");
        }

        var byName = fields.ToDictionary(f => f.FieldName, StringComparer.OrdinalIgnoreCase);
        var violations = new List<RuleViolation>();

        var selected = ResolveFields(request, fields, byName, violations);
        var filters = ValidateFilters(request, byName, violations);
        var sort = ResolveSort(request, byName, fields, violations);
        var companyCodeId = await ResolveCompanyCodeAsync(
            request, byName, violations, cancellationToken);

        if (violations.Count > 0)
        {
            return new BrowserQueryResult
            {
                SchemaName = request.SchemaName,
                TableName = request.TableName,
                Violations = violations,
            };
        }

        var limit = Math.Min(
            Math.Min(request.MaxRows <= 0 ? 100 : request.MaxRows, AbsoluteRowLimit),
            authorizationGroup?.MaxRowsPerQuery ?? AbsoluteRowLimit);

        var query = BuildQuery(request, table, selected, filters, sort, limit, companyCodeId);

        var result = await context.QueryRawAsync(query, cancellationToken);
        stopwatch.Stop();

        var wasTruncated = result.Rows.Count > limit;
        var rows = result.Rows
            .Take(limit)
            .Select(row => Format(row, selected))
            .ToList();

        await LogAsync(request, table, selected, filters, rows.Count, limit, wasTruncated,
            (int)stopwatch.ElapsedMilliseconds, cancellationToken);

        logger.LogInformation(
            "Browser query on {Schema}.{Table} by {User} returned {Rows} rows in {Duration} ms",
            request.SchemaName, request.TableName, currentUser.UserName, rows.Count,
            stopwatch.ElapsedMilliseconds);

        return new BrowserQueryResult
        {
            SchemaName = request.SchemaName,
            TableName = request.TableName,
            Columns = selected
                .Select(f => new BrowserColumn(
                    f.FieldName, f.SqlType, f.ShortDescription, f.IsKey, f.IsMasked))
                .ToList(),
            Rows = rows,
            WasTruncated = wasTruncated,
            RowLimitApplied = limit,
            DurationMilliseconds = (int)stopwatch.ElapsedMilliseconds,
            ExecutedSql = query.Sql,
        };
    }

    /// <summary>
    /// Builds the statement. Identifiers come from the dictionary rows loaded
    /// above and are bracket-quoted; every caller value becomes a parameter.
    /// There is no path by which request text reaches the statement.
    /// </summary>
    private RawQuery BuildQuery(
        BrowserQueryRequest request,
        DictionaryTable table,
        IReadOnlyList<DictionaryTableField> selected,
        IReadOnlyList<ValidatedFilter> filters,
        SortSpecification sort,
        int limit,
        long? companyCodeId)
    {
        var parameters = new List<RawParameter>();
        var sql = new StringBuilder();

        // A masked column is never read at all - it leaves the database as a
        // literal NULL. Fetching it and blanking it afterwards would still put
        // the value on the wire and in every buffer between here and there.
        sql.Append("SELECT ");
        sql.AppendJoin(", ", selected.Select(f =>
            f.IsMasked ? $"NULL AS [{f.FieldName}]" : $"[{f.FieldName}]"));

        // No NOLOCK: this is a finance ledger, and a browser that shows
        // half-written documents is worse than one that waits.
        sql.Append($" FROM [{table.SchemaName}].[{table.TableName}]");

        var conditions = new List<string>();

        // Tenant isolation is added here, not requested by the caller: a
        // browser query cannot be written that crosses tenants.
        if (table.IsTenantDependent)
        {
            parameters.Add(new RawParameter("@tenantId", TenantId));
            conditions.Add("[TenantId] = @tenantId");
        }

        if (companyCodeId is not null)
        {
            parameters.Add(new RawParameter("@companyCodeId", companyCodeId.Value));
            conditions.Add("[CompanyCodeId] = @companyCodeId");
        }

        foreach (var filter in filters)
        {
            conditions.Add(Condition(filter, parameters));
        }

        if (conditions.Count > 0)
        {
            sql.Append(" WHERE ").AppendJoin(" AND ", conditions);
        }

        sql.Append($" ORDER BY [{sort.Field}] {(sort.Descending ? "DESC" : "ASC")}");
        if (!string.Equals(sort.Field, sort.TieBreak, StringComparison.OrdinalIgnoreCase))
        {
            sql.Append($", [{sort.TieBreak}] ASC");
        }

        var offset = Math.Max(request.PageNumber - 1, 0) * limit;
        parameters.Add(new RawParameter("@offset", offset));

        // One row more than the limit, so the caller can be told the result was
        // cut short instead of quietly seeing a partial answer.
        parameters.Add(new RawParameter("@limit", limit + 1));
        sql.Append(" OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY;");

        return new RawQuery(sql.ToString(), parameters);
    }

    private static string Condition(ValidatedFilter filter, List<RawParameter> parameters)
    {
        var column = $"[{filter.Field.FieldName}]";

        string Parameter(string? value)
        {
            var name = $"@p{parameters.Count}";
            parameters.Add(new RawParameter(name, value));
            return name;
        }

        var condition = filter.Operator switch
        {
            BrowserOperators.Equals => $"{column} = {Parameter(filter.Value)}",
            BrowserOperators.NotEquals => $"{column} <> {Parameter(filter.Value)}",
            BrowserOperators.GreaterThan => $"{column} > {Parameter(filter.Value)}",
            BrowserOperators.GreaterOrEqual => $"{column} >= {Parameter(filter.Value)}",
            BrowserOperators.LessThan => $"{column} < {Parameter(filter.Value)}",
            BrowserOperators.LessOrEqual => $"{column} <= {Parameter(filter.Value)}",
            BrowserOperators.Between =>
                $"{column} BETWEEN {Parameter(filter.Value)} AND {Parameter(filter.HighValue)}",

            // The wildcards are added to the parameter value, so a percent sign
            // typed by the user is matched literally rather than expanding.
            BrowserOperators.Contains => $"{column} LIKE {Parameter($"%{Escape(filter.Value)}%")} ESCAPE '\\'",
            BrowserOperators.StartsWith => $"{column} LIKE {Parameter($"{Escape(filter.Value)}%")} ESCAPE '\\'",
            BrowserOperators.EndsWith => $"{column} LIKE {Parameter($"%{Escape(filter.Value)}")} ESCAPE '\\'",
            BrowserOperators.IsEmpty => $"{column} IS NULL",
            BrowserOperators.IsNotEmpty => $"{column} IS NOT NULL",
            BrowserOperators.In or BrowserOperators.NotIn =>
                $"{column} {(filter.Operator == BrowserOperators.In ? "IN" : "NOT IN")} " +
                $"({string.Join(", ", filter.Values.Select(Parameter))})",
            _ => throw new InvalidOperationException($"Operator {filter.Operator} slipped through validation."),
        };

        return filter.Exclude ? $"NOT ({condition})" : condition;
    }

    /// <summary>Escapes the LIKE wildcards so they match themselves.</summary>
    private static string Escape(string? value) =>
        (value ?? string.Empty)
            .Replace("\\", "\\\\", StringComparison.Ordinal)
            .Replace("%", "\\%", StringComparison.Ordinal)
            .Replace("_", "\\_", StringComparison.Ordinal)
            .Replace("[", "\\[", StringComparison.Ordinal);

    private static List<DictionaryTableField> ResolveFields(
        BrowserQueryRequest request,
        List<DictionaryTableField> fields,
        Dictionary<string, DictionaryTableField> byName,
        List<RuleViolation> violations)
    {
        if (request.Fields.Count == 0)
        {
            return fields;
        }

        var selected = new List<DictionaryTableField>();

        foreach (var name in request.Fields)
        {
            if (byName.TryGetValue(name, out var field))
            {
                selected.Add(field);
            }
            else
            {
                violations.Add(new RuleViolation(
                    BrowserErrorCodes.FieldUnknown,
                    $"{request.SchemaName}.{request.TableName} has no field {name}.",
                    nameof(request.Fields)));
            }
        }

        return selected.Count > 0 ? selected : fields;
    }

    private static List<ValidatedFilter> ValidateFilters(
        BrowserQueryRequest request,
        Dictionary<string, DictionaryTableField> byName,
        List<RuleViolation> violations)
    {
        var validated = new List<ValidatedFilter>();

        for (var index = 0; index < request.Filters.Count; index++)
        {
            var filter = request.Filters[index];
            var field = $"Filters[{index}]";

            if (!byName.TryGetValue(filter.Field, out var dictionaryField))
            {
                violations.Add(new RuleViolation(
                    BrowserErrorCodes.FieldUnknown,
                    $"Cannot filter on {filter.Field}: the field does not exist.",
                    $"{field}.Field"));
                continue;
            }

            // Filtering on a masked field would defeat the mask: repeated
            // equality probes recover the value the display refuses to show.
            if (dictionaryField.IsMasked)
            {
                violations.Add(new RuleViolation(
                    BrowserErrorCodes.FieldMasked,
                    $"{filter.Field} is masked and cannot be used as a filter.",
                    $"{field}.Field"));
                continue;
            }

            if (!BrowserOperators.All.Contains(filter.Operator))
            {
                violations.Add(new RuleViolation(
                    BrowserErrorCodes.OperatorUnknown,
                    $"{filter.Operator} is not an operator this browser accepts.",
                    $"{field}.Operator"));
                continue;
            }

            if (!BrowserOperators.IsUnary(filter.Operator))
            {
                var hasValue = filter.Operator is BrowserOperators.In or BrowserOperators.NotIn
                    ? filter.Values.Count > 0
                    : filter.Value is not null;

                if (!hasValue)
                {
                    violations.Add(new RuleViolation(
                        BrowserErrorCodes.ValueRequired,
                        $"Operator {filter.Operator} needs a value.",
                        $"{field}.Value"));
                    continue;
                }
            }

            // SQL Server takes 2,100 parameters per command, and one filter must
            // not be able to spend them all. Refused with a number rather than
            // silently truncated, so nobody reads a short list as the answer.
            if (filter.Operator is BrowserOperators.In or BrowserOperators.NotIn
                && filter.Values.Count > MaxValuesPerFilter)
            {
                violations.Add(new RuleViolation(
                    BrowserErrorCodes.TooManyValues,
                    $"An {filter.Operator} filter takes at most {MaxValuesPerFilter} values; " +
                    $"this one has {filter.Values.Count}.",
                    $"{field}.Values"));
                continue;
            }

            if (filter.Operator == BrowserOperators.Between && filter.HighValue is null)
            {
                violations.Add(new RuleViolation(
                    BrowserErrorCodes.ValueRequired,
                    "A between filter needs both ends of the interval.",
                    $"{field}.HighValue"));
                continue;
            }

            validated.Add(new ValidatedFilter(
                dictionaryField, filter.Operator, filter.Value, filter.HighValue,
                filter.Values, filter.Exclude));
        }

        return validated;
    }

    /// <summary>
    /// Turns the requested company code into the surrogate key the table
    /// actually stores. Asking for a company code on a table that has none is
    /// reported rather than ignored: silently returning every company's rows to
    /// someone who asked for one company's rows is the wrong kind of surprise.
    /// </summary>
    private async Task<long?> ResolveCompanyCodeAsync(
        BrowserQueryRequest request,
        Dictionary<string, DictionaryTableField> byName,
        List<RuleViolation> violations,
        CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(request.CompanyCode))
        {
            return null;
        }

        if (!byName.ContainsKey("CompanyCodeId"))
        {
            violations.Add(new RuleViolation(
                BrowserErrorCodes.CompanyCodeNotApplicable,
                $"{request.SchemaName}.{request.TableName} is not company-code dependent.",
                nameof(request.CompanyCode)));

            return null;
        }

        var companyCodeId = await context.Query<CompanyCode>()
            .AsNoTracking()
            .Where(c => c.TenantId == TenantId && c.CompanyCodeKey == request.CompanyCode)
            .Select(c => (long?)c.Id)
            .FirstOrDefaultAsync(cancellationToken);

        if (companyCodeId is null)
        {
            violations.Add(new RuleViolation(
                BrowserErrorCodes.CompanyCodeUnknown,
                $"Company code {request.CompanyCode} does not exist.",
                nameof(request.CompanyCode)));
        }

        return companyCodeId;
    }

    private static SortSpecification ResolveSort(
        BrowserQueryRequest request,
        Dictionary<string, DictionaryTableField> byName,
        List<DictionaryTableField> fields,
        List<RuleViolation> violations)
    {
        // Every result is ordered by a unique column as well, so paging cannot
        // repeat one row and skip another.
        var tieBreak = fields.FirstOrDefault(f => f.IsKey && !f.IsMasked)?.FieldName
                       ?? fields.First(f => !f.IsMasked).FieldName;

        if (request.SortBy is null)
        {
            return new SortSpecification(tieBreak, request.SortDescending, tieBreak);
        }

        if (!byName.TryGetValue(request.SortBy, out var sortField))
        {
            violations.Add(new RuleViolation(
                BrowserErrorCodes.SortFieldUnknown,
                $"Cannot sort by {request.SortBy}: the field does not exist.",
                nameof(request.SortBy)));

            return new SortSpecification(tieBreak, request.SortDescending, tieBreak);
        }

        // Sorting by a masked column orders the rows by the hidden value, which
        // is most of the way to reading it.
        if (sortField.IsMasked)
        {
            violations.Add(new RuleViolation(
                BrowserErrorCodes.FieldMasked,
                $"{request.SortBy} is masked and cannot be sorted on.",
                nameof(request.SortBy)));

            return new SortSpecification(tieBreak, request.SortDescending, tieBreak);
        }

        return new SortSpecification(sortField.FieldName, request.SortDescending, tieBreak);
    }

    private static List<string?> Format(
        IReadOnlyList<object?> row,
        IReadOnlyList<DictionaryTableField> fields)
    {
        var formatted = new List<string?>(fields.Count);

        for (var index = 0; index < fields.Count; index++)
        {
            if (fields[index].IsMasked)
            {
                // Masked means masked: the browser never returns the value, so
                // an export cannot carry it out either.
                formatted.Add(Mask);
                continue;
            }

            var value = index < row.Count ? row[index] : null;
            formatted.Add(value switch
            {
                null => null,
                DateTime dateTime => dateTime.ToString("O"),
                DateOnly date => date.ToString("yyyy-MM-dd"),
                byte[] bytes => Convert.ToHexString(bytes),
                IFormattable formattable => formattable.ToString(null, System.Globalization.CultureInfo.InvariantCulture),
                _ => value.ToString(),
            });
        }

        return formatted;
    }

    private async Task LogAsync(
        BrowserQueryRequest request,
        DictionaryTable table,
        IReadOnlyList<DictionaryTableField> selected,
        IReadOnlyList<ValidatedFilter> filters,
        int rowsReturned,
        int limit,
        bool wasTruncated,
        int durationMilliseconds,
        CancellationToken cancellationToken)
    {
        var userId = await context.Query<User>()
            .AsNoTracking()
            .Where(u => u.TenantId == TenantId && u.UserName == currentUser.UserName)
            .Select(u => (long?)u.Id)
            .FirstOrDefaultAsync(cancellationToken);

        // The filter is logged with masked values redacted: the log must not
        // become the leak the masking prevented.
        var filterJson = JsonSerializer.Serialize(filters.Select(f => new
        {
            f.Field.FieldName,
            f.Operator,
            Value = f.Field.IsMasked ? Mask : f.Value,
            HighValue = f.Field.IsMasked ? Mask : f.HighValue,
            f.Exclude,
        }));

        context.Add(new BrowserQueryLog
        {
            TenantId = TenantId,
            UserId = userId ?? 0,
            ExecutedAt = timeProvider.GetUtcNow().UtcDateTime,
            SchemaName = table.SchemaName,
            ObjectName = table.TableName,
            SelectedFields = string.Join(",", selected.Select(f => f.FieldName)),
            FilterJson = filterJson,
            RowsReturned = rowsReturned,
            RowLimitApplied = limit,
            WasExported = request.IsExport,
            ExportFormat = request.IsExport ? "XLSX" : null,
            DurationMs = durationMilliseconds,
            WasTruncated = wasTruncated,
        });

        await context.SaveChangesAsync(cancellationToken);
    }

    private static BrowserQueryResult Refused(
        BrowserQueryRequest request,
        string code,
        string message) =>
        new()
        {
            SchemaName = request.SchemaName,
            TableName = request.TableName,
            Violations = [new RuleViolation(code, message)],
        };

    private sealed record ValidatedFilter(
        DictionaryTableField Field,
        string Operator,
        string? Value,
        string? HighValue,
        IReadOnlyList<string> Values,
        bool Exclude);

    private sealed record SortSpecification(string Field, bool Descending, string TieBreak);
}
