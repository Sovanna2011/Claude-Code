using System.Text;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Application.DataDictionary;

/// <inheritdoc />
public sealed class DictionaryService(
    IErpDataContext context,
    ITenantProvider tenantProvider) : IDictionaryService
{
    private int TenantId => tenantProvider.TenantId;

    public async Task<IReadOnlyList<DictionaryObjectSummary>> SearchObjectsAsync(
        string? objectType = null,
        string? search = null,
        int maxResults = 200,
        CancellationToken cancellationToken = default)
    {
        var take = Math.Clamp(maxResults, 1, 1_000);

        return await context.Query<DictionaryObject>()
            .AsNoTracking()
            .Where(o => o.TenantId == TenantId
                        && (objectType == null || o.ObjectType == objectType)
                        && (search == null
                            || o.ObjectName.Contains(search)
                            || o.ShortDescription.Contains(search)))
            .OrderBy(o => o.ObjectType)
            .ThenBy(o => o.ObjectName)
            .Take(take)
            .Select(o => new DictionaryObjectSummary(
                o.ObjectType,
                o.ObjectName,
                o.ShortDescription,
                o.Status,
                o.IsCustomObject,
                o.Package,
                o.ResponsibleUser,
                o.LastActivatedAt,
                o.ActiveVersion))
            .ToListAsync(cancellationToken);
    }

    public async Task<TableDefinition?> GetTableAsync(
        string schemaName,
        string tableName,
        CancellationToken cancellationToken = default)
    {
        var table = await context.Query<DictionaryTable>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                t => t.TenantId == TenantId
                     && t.SchemaName == schemaName
                     && t.TableName == tableName,
                cancellationToken);

        if (table is null)
        {
            return null;
        }

        var fields = await LoadFieldsAsync(table.Id, cancellationToken);
        var indexes = await LoadIndexesAsync(table.Id, cancellationToken);
        var foreignKeys = await LoadForeignKeysAsync(schemaName, tableName, cancellationToken);

        return new TableDefinition(
            table.SchemaName,
            table.TableName,
            table.ShortDescription,
            table.TableCategory,
            table.DeliveryClass,
            table.MaintenanceType,
            table.IsTenantDependent,
            table.IsCompanyCodeDependent,
            table.IsLogged,
            table.IsImmutable,
            table.AuthorizationGroup,
            table.PrimaryKeyFields,
            table.Status,
            fields,
            indexes,
            foreignKeys);
    }

    /// <summary>
    /// Fields with the data element and domain names resolved, because a field
    /// row on its own says <c>nvarchar(4)</c> where a reader wants to know it is
    /// a company code.
    /// </summary>
    private async Task<IReadOnlyList<FieldDefinition>> LoadFieldsAsync(
        long tableId,
        CancellationToken cancellationToken)
    {
        var fields = await context.Query<DictionaryTableField>()
            .AsNoTracking()
            .Where(f => f.TenantId == TenantId && f.DictionaryTableId == tableId)
            .OrderBy(f => f.FieldPosition)
            .ToListAsync(cancellationToken);

        // Looked up by id rather than left-joined. A group join whose key
        // selector reads through the optional side (dataElement.DomainId) is
        // fine once EF turns it into SQL and a null reference the moment the
        // same query runs over objects - the sort of difference that shows up
        // only in one of the two.
        var dataElementIds = fields
            .Where(f => f.DataElementId is not null)
            .Select(f => f.DataElementId!.Value)
            .Distinct()
            .ToList();

        var dataElements = new Dictionary<long, DictionaryDataElement>();
        if (dataElementIds.Count > 0)
        {
            dataElements = await context.Query<DictionaryDataElement>()
                .AsNoTracking()
                .Where(e => e.TenantId == TenantId && dataElementIds.Contains(e.Id))
                .ToDictionaryAsync(e => e.Id, cancellationToken);
        }

        var domainIds = dataElements.Values
            .Select(e => e.DictionaryDomainId)
            .Distinct()
            .ToList();

        var domains = new Dictionary<long, DictionaryDomain>();
        if (domainIds.Count > 0)
        {
            domains = await context.Query<DictionaryDomain>()
                .AsNoTracking()
                .Where(d => d.TenantId == TenantId && domainIds.Contains(d.Id))
                .ToDictionaryAsync(d => d.Id, cancellationToken);
        }

        return fields.Select(field =>
        {
            DictionaryDataElement? dataElement = null;
            if (field.DataElementId is { } dataElementId)
            {
                dataElements.TryGetValue(dataElementId, out dataElement);
            }

            string? domainName = null;
            if (dataElement is not null
                && domains.TryGetValue(dataElement.DictionaryDomainId, out var domain))
            {
                domainName = domain.DomainName;
            }

            return new FieldDefinition(
                field.FieldPosition,
                field.FieldName,
                field.SqlType,
                field.Length,
                field.DecimalPlaces,
                field.IsKey,
                field.IsRequired,
                field.IsIdentity,
                field.IsMasked,
                field.IsCustomField,
                field.DefaultValue,
                dataElement?.DataElementName,

                // The field may name a domain directly; otherwise it inherits
                // the one behind its data element.
                field.DomainName ?? domainName,
                field.CheckTableName,
                field.CurrencyReferenceField,
                field.UnitReferenceField,
                field.IncludeName,
                field.ShortDescription);
        }).ToList();
    }

    private async Task<IReadOnlyList<IndexDefinition>> LoadIndexesAsync(
        long tableId,
        CancellationToken cancellationToken)
    {
        var indexes = await context.Query<DictionaryIndex>()
            .AsNoTracking()
            .Where(i => i.TenantId == TenantId && i.DictionaryTableId == tableId)
            .OrderBy(i => i.IndexName)
            .ToListAsync(cancellationToken);

        if (indexes.Count == 0)
        {
            return [];
        }

        var indexIds = indexes.Select(i => i.Id).ToList();

        var fields = await context.Query<DictionaryIndexField>()
            .AsNoTracking()
            .Where(f => f.TenantId == TenantId && indexIds.Contains(f.DictionaryIndexId))
            .OrderBy(f => f.FieldPosition)
            .ToListAsync(cancellationToken);

        var byIndex = fields.GroupBy(f => f.DictionaryIndexId)
            .ToDictionary(group => group.Key, group => group.ToList());

        return indexes.Select(index => new IndexDefinition(
            index.IndexName,
            index.ShortDescription,
            index.IsUnique,
            index.IsClustered,
            index.IsColumnStore,
            index.FilterPredicate,
            index.IncludedColumns,
            byIndex.TryGetValue(index.Id, out var indexFields)
                ? indexFields
                    .Select(f => new IndexFieldDefinition(
                        f.FieldPosition, f.FieldName, f.SortDirection))
                    .ToList()
                : [])).ToList();
    }

    private async Task<IReadOnlyList<ForeignKeyDefinition>> LoadForeignKeysAsync(
        string schemaName,
        string tableName,
        CancellationToken cancellationToken)
    {
        var foreignKeys = await context.Query<DictionaryForeignKey>()
            .AsNoTracking()
            .Where(fk => fk.TenantId == TenantId
                         && fk.SourceSchemaName == schemaName
                         && fk.SourceTableName == tableName)
            .OrderBy(fk => fk.ForeignKeyName)
            .ToListAsync(cancellationToken);

        if (foreignKeys.Count == 0)
        {
            return [];
        }

        var foreignKeyIds = foreignKeys.Select(fk => fk.Id).ToList();

        var fields = await context.Query<DictionaryForeignKeyField>()
            .AsNoTracking()
            .Where(f => f.TenantId == TenantId
                        && foreignKeyIds.Contains(f.DictionaryForeignKeyId))
            .OrderBy(f => f.FieldPosition)
            .ToListAsync(cancellationToken);

        var byForeignKey = fields.GroupBy(f => f.DictionaryForeignKeyId)
            .ToDictionary(group => group.Key, group => group.ToList());

        return foreignKeys.Select(fk => new ForeignKeyDefinition(
            fk.ForeignKeyName,
            fk.TargetSchemaName,
            fk.TargetTableName,
            fk.Cardinality,
            fk.ForeignKeyType,
            fk.CheckRequired,
            fk.OnDeleteAction,
            byForeignKey.TryGetValue(fk.Id, out var keyFields)
                ? keyFields
                    .Select(f => new ForeignKeyFieldDefinition(
                        f.FieldPosition, f.SourceFieldName, f.TargetFieldName, f.ConstantValue))
                    .ToList()
                : [])).ToList();
    }

    public async Task<IReadOnlyList<UsageReference>> GetWhereUsedAsync(
        string schemaName,
        string tableName,
        CancellationToken cancellationToken = default)
    {
        var foreignKeys = await context.Query<DictionaryForeignKey>()
            .AsNoTracking()
            .Where(fk => fk.TenantId == TenantId
                         && fk.TargetSchemaName == schemaName
                         && fk.TargetTableName == tableName)
            .OrderBy(fk => fk.SourceSchemaName)
            .ThenBy(fk => fk.SourceTableName)
            .ToListAsync(cancellationToken);

        if (foreignKeys.Count == 0)
        {
            return [];
        }

        var foreignKeyIds = foreignKeys.Select(fk => fk.Id).ToList();

        var fields = await context.Query<DictionaryForeignKeyField>()
            .AsNoTracking()
            .Where(f => f.TenantId == TenantId
                        && foreignKeyIds.Contains(f.DictionaryForeignKeyId))
            .OrderBy(f => f.FieldPosition)
            .ToListAsync(cancellationToken);

        var byForeignKey = fields.GroupBy(f => f.DictionaryForeignKeyId)
            .ToDictionary(group => group.Key, group => group.ToList());

        return foreignKeys.Select(fk => new UsageReference(
            fk.ForeignKeyName,
            fk.SourceSchemaName,
            fk.SourceTableName,
            fk.Cardinality,
            byForeignKey.TryGetValue(fk.Id, out var keyFields)
                ? keyFields.Select(f => f.SourceFieldName).ToList()
                : [])).ToList();
    }

    public async Task<DomainDefinition?> GetDomainAsync(
        string domainName,
        CancellationToken cancellationToken = default)
    {
        var domain = await context.Query<DictionaryDomain>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                d => d.TenantId == TenantId && d.DomainName == domainName,
                cancellationToken);

        return domain is null ? null : await MapDomainAsync(domain, cancellationToken);
    }

    private async Task<DomainDefinition> MapDomainAsync(
        DictionaryDomain domain,
        CancellationToken cancellationToken)
    {
        var fixedValues = domain.HasFixedValues
            ? await context.Query<DictionaryDomainValue>()
                .AsNoTracking()
                .Where(v => v.TenantId == TenantId && v.DictionaryDomainId == domain.Id)
                .OrderBy(v => v.DisplayOrder)
                .ThenBy(v => v.LowValue)
                .Select(v => new DomainFixedValue(
                    v.LowValue, v.HighValue, v.Description, v.IsDefault, v.DisplayOrder))
                .ToListAsync(cancellationToken)
            : [];

        return new DomainDefinition(
            domain.DomainName,
            domain.ShortDescription,
            domain.DataType,
            domain.SqlType,
            domain.Length,
            domain.DecimalPlaces,
            domain.IsSigned,
            domain.CaseSensitive,
            domain.ValueTableName,
            domain.LowerLimit,
            domain.UpperLimit,
            domain.Status,
            fixedValues);
    }

    public async Task<DataElementDefinition?> GetDataElementAsync(
        string dataElementName,
        CancellationToken cancellationToken = default)
    {
        var dataElement = await context.Query<DictionaryDataElement>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                e => e.TenantId == TenantId && e.DataElementName == dataElementName,
                cancellationToken);

        if (dataElement is null)
        {
            return null;
        }

        var domain = await context.Query<DictionaryDomain>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                d => d.TenantId == TenantId && d.Id == dataElement.DictionaryDomainId,
                cancellationToken);

        return new DataElementDefinition(
            dataElement.DataElementName,
            dataElement.ShortDescription,
            dataElement.ShortLabel,
            dataElement.MediumLabel,
            dataElement.LongLabel,
            dataElement.HeaderLabel,
            dataElement.IsChangeDocumentRelevant,
            dataElement.IsPersonalData,
            dataElement.DocumentationText,
            dataElement.Status,
            domain is null ? null : await MapDomainAsync(domain, cancellationToken));
    }

    public async Task<string?> GenerateCreateTableSqlAsync(
        string schemaName,
        string tableName,
        CancellationToken cancellationToken = default)
    {
        var table = await GetTableAsync(schemaName, tableName, cancellationToken);

        if (table is null || table.Fields.Count == 0)
        {
            return null;
        }

        var sql = new StringBuilder();

        sql.AppendLine($"/* {table.SchemaName}.{table.TableName} - {table.ShortDescription} */");
        sql.AppendLine($"CREATE TABLE [{table.SchemaName}].[{table.TableName}]");
        sql.AppendLine("(");

        var width = table.Fields.Max(f => f.FieldName.Length) + 3;

        // Kept as (definition, trailing comment) pairs so the comma that
        // separates members can be dropped from the last one without having to
        // find it again inside a comment.
        var members = new List<(string Definition, string Comment)>();

        foreach (var field in table.Fields)
        {
            var column = $"[{field.FieldName}]".PadRight(width);
            var nullability = field.IsRequired || field.IsKey ? "NOT NULL" : "NULL";
            var identity = field.IsIdentity ? " IDENTITY(1,1)" : string.Empty;
            var @default = field.DefaultValue is null
                ? string.Empty
                : $" CONSTRAINT [DF_{table.SchemaName}_{table.TableName}_{field.FieldName}] " +
                  $"DEFAULT ({field.DefaultValue})";

            // rowversion is assigned by the engine, so it carries neither a
            // nullability clause of its own choosing nor a default.
            var definition = field.SqlType.Equals("rowversion", StringComparison.OrdinalIgnoreCase)
                ? $"    {column}rowversion NOT NULL"
                : $"    {column}{field.SqlType}{identity} {nullability}{@default}";

            members.Add((definition, Comment(field)));
        }

        var keyFields = table.Fields.Where(f => f.IsKey).Select(f => f.FieldName).ToList();

        if (keyFields.Count > 0)
        {
            members.Add((
                $"    CONSTRAINT [PK_{table.SchemaName}_{table.TableName}] PRIMARY KEY CLUSTERED " +
                $"({string.Join(", ", keyFields.Select(name => $"[{name}]"))})",
                string.Empty));
        }

        var rendered = members.Select((member, index) =>
            member.Definition + (index < members.Count - 1 ? "," : string.Empty) + member.Comment);

        sql.AppendLine(string.Join(Environment.NewLine, rendered));
        sql.AppendLine(");");

        foreach (var index in table.Indexes)
        {
            // A clustered columnstore index covers the whole table and takes no
            // column list; every other kind needs one.
            var isClusteredColumnStore = index.IsColumnStore && index.IsClustered;

            if (index.Fields.Count == 0 && !isClusteredColumnStore)
            {
                continue;
            }

            sql.AppendLine();
            sql.Append("CREATE ");
            sql.Append(index.IsUnique && !index.IsColumnStore ? "UNIQUE " : string.Empty);
            sql.Append(index.IsClustered ? "CLUSTERED " : "NONCLUSTERED ");
            sql.Append(index.IsColumnStore ? "COLUMNSTORE " : string.Empty);
            sql.Append($"INDEX [{index.IndexName}] ");
            sql.Append($"ON [{table.SchemaName}].[{table.TableName}]");

            if (!isClusteredColumnStore)
            {
                // A columnstore index lists columns without a sort direction:
                // it has no order to speak of.
                sql.Append($" ({string.Join(", ", index.Fields.Select(f =>
                    index.IsColumnStore ? $"[{f.FieldName}]" : $"[{f.FieldName}] {f.SortDirection}"))})");
            }

            if (!string.IsNullOrWhiteSpace(index.IncludedColumns) && !index.IsColumnStore)
            {
                sql.Append($" INCLUDE ({index.IncludedColumns})");
            }

            if (!string.IsNullOrWhiteSpace(index.FilterPredicate))
            {
                sql.Append($" WHERE {index.FilterPredicate}");
            }

            sql.AppendLine(";");
        }

        foreach (var foreignKey in table.ForeignKeys.Where(fk => fk.Fields.Count > 0))
        {
            sql.AppendLine();
            sql.AppendLine($"ALTER TABLE [{table.SchemaName}].[{table.TableName}]");
            sql.AppendLine($"    ADD CONSTRAINT [{foreignKey.ForeignKeyName}] FOREIGN KEY " +
                           $"({string.Join(", ", foreignKey.Fields.Select(f => $"[{f.SourceFieldName}]"))})");
            sql.Append($"    REFERENCES [{foreignKey.TargetSchemaName}].[{foreignKey.TargetTableName}] " +
                       $"({string.Join(", ", foreignKey.Fields.Select(f => $"[{f.TargetFieldName}]"))})");
            sql.AppendLine(foreignKey.OnDeleteAction is "NO ACTION" or ""
                ? ";"
                : $" ON DELETE {foreignKey.OnDeleteAction};");
        }

        return sql.ToString();
    }

    private static string Comment(FieldDefinition field) =>
        string.IsNullOrWhiteSpace(field.ShortDescription)
            ? string.Empty
            : $"  -- {field.ShortDescription}";
}
