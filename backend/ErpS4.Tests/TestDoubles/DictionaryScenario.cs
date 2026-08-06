using ErpS4.Application.DataDictionary;
using ErpS4.Application.TableBrowser;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.Extensions.Logging.Abstractions;

namespace ErpS4.Tests.TestDoubles;

/// <summary>
/// A small data dictionary: one financial table, one master-data table with a
/// masked column, and one security table that must never be browsable.
/// </summary>
/// <remarks>
/// Deliberately not the real 228-table catalogue. The rules under test are
/// about what the browser refuses and what its SQL looks like, and four tables
/// exercise all of them.
/// </remarks>
public sealed class DictionaryScenario
{
    public const string Actor = "auditor";

    public InMemoryDataContext Context { get; } = new();

    public FixedTimeProvider Clock { get; } = new(new DateTimeOffset(2026, 3, 1, 9, 0, 0, TimeSpan.Zero));

    public long JournalHeaderTableId { get; private set; }

    public long PartnerBankTableId { get; private set; }

    public static DictionaryScenario Create()
    {
        var scenario = new DictionaryScenario();
        scenario.Seed();
        return scenario;
    }

    public TableBrowserService CreateBrowser() =>
        new(Context,
            new FixedTenantProvider(1),
            new FixedCurrentUser(Actor),
            Clock,
            NullLogger<TableBrowserService>.Instance);

    public DictionaryService CreateDictionary() =>
        new(Context, new FixedTenantProvider(1));

    private void Seed()
    {
        Context.Seed(new User
        {
            TenantId = 1,
            UserName = Actor,
            DisplayName = "Internal audit",
            Email = "audit@example.test",
            UserType = "Dialog",
            Status = "Active",
            LanguageCode = "EN",
            TimeZoneId = "Asia/Phnom_Penh",
            ValidFrom = new DateOnly(2000, 1, 1),
            ValidTo = new DateOnly(9999, 12, 31),
        });

        Context.Seed(
            Group("FINC", protectedGroup: false, allowExport: true, maxRows: 500),
            Group("MAST", protectedGroup: false, allowExport: false, maxRows: 500),
            Group("SECU", protectedGroup: true, allowExport: false, maxRows: 0));

        Context.Seed(new CompanyCode
        {
            TenantId = 1,
            CompanyCodeKey = "1000",
            Name = "Cambodia Trading",
            CompanyId = 1,
            CountryCode = "KH",
            LocalCurrencyCode = "USD",
            LanguageCode = "EN",
            ChartOfAccountsId = 1,
            FiscalYearVariantId = 1,
            PostingPeriodVariantId = 1,
            FieldStatusVariantId = 1,
            ValidFrom = new DateOnly(2000, 1, 1),
            ValidTo = new DateOnly(9999, 12, 31),
        });

        JournalHeaderTableId = SeedTable(
            "fin", "JournalEntryHeader", "Journal entry header", "Transaction", "FINC",
            [
                Field("Id", "bigint", key: true, identity: true),
                Field("TenantId", "int", required: true),
                Field("CompanyCodeId", "bigint", required: true),
                Field("DocumentNumber", "nvarchar(10)", required: true),
                Field("PostingDate", "date", required: true),
                Field("HeaderText", "nvarchar(25)"),
            ]);

        PartnerBankTableId = SeedTable(
            "mdm", "BusinessPartnerBank", "Bank details of a business partner", "Master", "MAST",
            [
                Field("Id", "bigint", key: true, identity: true),
                Field("TenantId", "int", required: true),
                Field("BusinessPartnerId", "bigint", required: true),
                Field("BankAccountNumber", "nvarchar(35)", masked: true),
                Field("Iban", "nvarchar(34)", masked: true),
            ]);

        // Protected by its group, and in the sec schema besides: either alone
        // has to be enough to refuse it.
        SeedTable(
            "sec", "User", "Application user", "Master", "SECU",
            [
                Field("Id", "bigint", key: true, identity: true),
                Field("TenantId", "int", required: true),
                Field("UserName", "nvarchar(64)", required: true),
                Field("PasswordHash", "nvarchar(255)", masked: true),
            ]);

        // Not company-code dependent, and no authorization group at all.
        SeedTable(
            "cfg", "Currency", "Currency", "Configuration", null,
            [
                Field("Id", "bigint", key: true, identity: true),
                Field("TenantId", "int", required: true),
                Field("CurrencyCode", "nvarchar(3)", required: true),
                Field("DecimalPlaces", "tinyint", required: true),
            ]);

        SeedForeignKey(
            "FK_fin_JournalEntryHeader_CompanyCodeId",
            "fin", "JournalEntryHeader", "org", "CompanyCode",
            "CompanyCodeId", "Id");
    }

    private static TableAuthorizationGroup Group(
        string code, bool protectedGroup, bool allowExport, int maxRows) =>
        new()
        {
            TenantId = 1,
            AuthorizationGroup = code,
            Name = code,
            IsSystemProtected = protectedGroup,
            AllowExport = allowExport,
            MaxRowsPerQuery = maxRows,
        };

    private long SeedTable(
        string schemaName,
        string tableName,
        string description,
        string category,
        string? authorizationGroup,
        IReadOnlyList<DictionaryTableField> fields)
    {
        var table = new DictionaryTable
        {
            TenantId = 1,
            SchemaName = schemaName,
            TableName = tableName,
            ShortDescription = description,
            TableCategory = category,
            DeliveryClass = "A",
            MaintenanceType = "Maintain",
            IsTenantDependent = true,
            IsCompanyCodeDependent = fields.Any(f => f.FieldName == "CompanyCodeId"),
            BufferingType = "None",
            IsLogged = true,
            AuthorizationGroup = authorizationGroup,
            PrimaryKeyFields = "Id",
            Status = "Active",
        };
        Context.Seed(table);

        for (var index = 0; index < fields.Count; index++)
        {
            fields[index].TenantId = 1;
            fields[index].DictionaryTableId = table.Id;
            fields[index].FieldPosition = index + 1;
            Context.Seed(fields[index]);
        }

        Context.Seed(new DictionaryObject
        {
            TenantId = 1,
            ObjectType = "Table",
            ObjectName = $"{schemaName}.{tableName}",
            ShortDescription = description,
            Status = "Active",
            ActiveVersion = 1,
        });

        return table.Id;
    }

    private static DictionaryTableField Field(
        string name,
        string sqlType,
        bool key = false,
        bool required = false,
        bool identity = false,
        bool masked = false) =>
        new()
        {
            FieldName = name,
            SqlType = sqlType,
            IsKey = key,
            IsRequired = required || key,
            IsIdentity = identity,
            IsMasked = masked,
            ShortDescription = name,
        };

    private void SeedForeignKey(
        string name,
        string sourceSchema,
        string sourceTable,
        string targetSchema,
        string targetTable,
        string sourceField,
        string targetField)
    {
        var foreignKey = new DictionaryForeignKey
        {
            TenantId = 1,
            ForeignKeyName = name,
            SourceSchemaName = sourceSchema,
            SourceTableName = sourceTable,
            TargetSchemaName = targetSchema,
            TargetTableName = targetTable,
            Cardinality = "N:1",
            ForeignKeyType = "Key",
            CheckRequired = true,
            OnDeleteAction = "NO ACTION",
            Status = "Active",
        };
        Context.Seed(foreignKey);

        Context.Seed(new DictionaryForeignKeyField
        {
            TenantId = 1,
            DictionaryForeignKeyId = foreignKey.Id,
            SourceFieldName = sourceField,
            TargetFieldName = targetField,
            FieldPosition = 1,
        });
    }

    /// <summary>Makes <see cref="InMemoryDataContext.QueryRawAsync"/> return rows.</summary>
    public void ReturnRows(IReadOnlyList<string> columns, params object?[][] rows) =>
        Context.RawResult = new RawQueryResult(columns, rows);
}
