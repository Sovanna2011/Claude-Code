namespace ErpS4.Application.DataDictionary;

/// <summary>
/// SE11-style data dictionary display (design prompt, section 10).
/// </summary>
/// <remarks>
/// Read-only. The dictionary describes the database, so a change to it is a
/// change to the database - that belongs behind the migration and approval flow
/// described in section 10.4, not behind a display service. What this offers is
/// what SE11 offers before you press change: what exists, what a table is made
/// of, and what the DDL for it would look like.
/// </remarks>
public interface IDictionaryService
{
    /// <summary>Repository browser: dictionary objects, optionally filtered.</summary>
    Task<IReadOnlyList<DictionaryObjectSummary>> SearchObjectsAsync(
        string? objectType = null,
        string? search = null,
        int maxResults = 200,
        CancellationToken cancellationToken = default);

    /// <summary>Full definition of one table: fields, keys, indexes, foreign keys.</summary>
    Task<TableDefinition?> GetTableAsync(
        string schemaName,
        string tableName,
        CancellationToken cancellationToken = default);

    /// <summary>A domain and its fixed values.</summary>
    Task<DomainDefinition?> GetDomainAsync(
        string domainName,
        CancellationToken cancellationToken = default);

    /// <summary>A data element with the domain it rests on.</summary>
    Task<DataElementDefinition?> GetDataElementAsync(
        string dataElementName,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// The DDL the dictionary describes, rendered for reading.
    /// </summary>
    /// <remarks>
    /// Generated, never executed here. A dictionary that cannot show its own
    /// DDL is a dictionary nobody trusts; a dictionary that runs it is a way to
    /// change the database without a migration.
    /// </remarks>
    Task<string?> GenerateCreateTableSqlAsync(
        string schemaName,
        string tableName,
        CancellationToken cancellationToken = default);

    /// <summary>Where a table is used: the foreign keys that point at it.</summary>
    Task<IReadOnlyList<UsageReference>> GetWhereUsedAsync(
        string schemaName,
        string tableName,
        CancellationToken cancellationToken = default);
}

public sealed record DictionaryObjectSummary(
    string ObjectType,
    string ObjectName,
    string ShortDescription,
    string Status,
    bool IsCustomObject,
    string? Package,
    string? ResponsibleUser,
    DateTime? LastActivatedAt,
    int ActiveVersion);

/// <param name="IsTenantDependent">True when every row carries a TenantId.</param>
/// <param name="AuthorizationGroup">Governs who may browse it in SE16N.</param>
public sealed record TableDefinition(
    string SchemaName,
    string TableName,
    string ShortDescription,
    string TableCategory,
    string DeliveryClass,
    string MaintenanceType,
    bool IsTenantDependent,
    bool IsCompanyCodeDependent,
    bool IsLogged,
    bool IsImmutable,
    string? AuthorizationGroup,
    string PrimaryKeyFields,
    string Status,
    IReadOnlyList<FieldDefinition> Fields,
    IReadOnlyList<IndexDefinition> Indexes,
    IReadOnlyList<ForeignKeyDefinition> ForeignKeys);

/// <param name="IsMasked">Masked fields are hidden by the browser and by exports.</param>
public sealed record FieldDefinition(
    int Position,
    string FieldName,
    string SqlType,
    int? Length,
    byte? DecimalPlaces,
    bool IsKey,
    bool IsRequired,
    bool IsIdentity,
    bool IsMasked,
    bool IsCustomField,
    string? DefaultValue,
    string? DataElementName,
    string? DomainName,
    string? CheckTableName,
    string? CurrencyReferenceField,
    string? UnitReferenceField,
    string? IncludeName,
    string ShortDescription);

public sealed record IndexDefinition(
    string IndexName,
    string? ShortDescription,
    bool IsUnique,
    bool IsClustered,
    bool IsColumnStore,
    string? FilterPredicate,
    string? IncludedColumns,
    IReadOnlyList<IndexFieldDefinition> Fields);

public sealed record IndexFieldDefinition(
    int Position,
    string FieldName,
    string SortDirection);

public sealed record ForeignKeyDefinition(
    string ForeignKeyName,
    string TargetSchemaName,
    string TargetTableName,
    string Cardinality,
    string ForeignKeyType,
    bool CheckRequired,
    string OnDeleteAction,
    IReadOnlyList<ForeignKeyFieldDefinition> Fields);

public sealed record ForeignKeyFieldDefinition(
    int Position,
    string SourceFieldName,
    string TargetFieldName,
    string? ConstantValue);

public sealed record DomainDefinition(
    string DomainName,
    string ShortDescription,
    string DataType,
    string SqlType,
    int? Length,
    byte? DecimalPlaces,
    bool IsSigned,
    bool CaseSensitive,
    string? ValueTableName,
    string? LowerLimit,
    string? UpperLimit,
    string Status,
    IReadOnlyList<DomainFixedValue> FixedValues);

public sealed record DomainFixedValue(
    string LowValue,
    string? HighValue,
    string Description,
    bool IsDefault,
    int DisplayOrder);

/// <param name="IsPersonalData">Drives the masking and the retention rules.</param>
public sealed record DataElementDefinition(
    string DataElementName,
    string ShortDescription,
    string? ShortLabel,
    string? MediumLabel,
    string? LongLabel,
    string? HeaderLabel,
    bool IsChangeDocumentRelevant,
    bool IsPersonalData,
    string? DocumentationText,
    string Status,
    DomainDefinition? Domain);

/// <param name="SourceFieldNames">The columns on the referencing side.</param>
public sealed record UsageReference(
    string ForeignKeyName,
    string SourceSchemaName,
    string SourceTableName,
    string Cardinality,
    IReadOnlyList<string> SourceFieldNames);
