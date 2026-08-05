using ErpS4.Tests.TestDoubles;
using Xunit;

namespace ErpS4.Tests;

/// <summary>
/// SE11 describes the database it sits on. These tests check that the
/// description matches what the dictionary actually holds, and that the DDL it
/// renders is the DDL those rows imply.
/// </summary>
public sealed class DictionaryTests
{
    [Fact]
    public async Task A_table_comes_back_with_its_fields_in_position_order()
    {
        var scenario = DictionaryScenario.Create();

        var table = await scenario.CreateDictionary().GetTableAsync("fin", "JournalEntryHeader");

        Assert.NotNull(table);
        Assert.Equal("Journal entry header", table.ShortDescription);
        Assert.Equal(
            new[] { "Id", "TenantId", "CompanyCodeId", "DocumentNumber", "PostingDate", "HeaderText" },
            table.Fields.Select(f => f.FieldName));
        Assert.Equal(new[] { 1, 2, 3, 4, 5, 6 }, table.Fields.Select(f => f.Position));
    }

    [Fact]
    public async Task A_table_that_does_not_exist_is_null_rather_than_empty()
    {
        var scenario = DictionaryScenario.Create();

        Assert.Null(await scenario.CreateDictionary().GetTableAsync("fin", "Invented"));
    }

    [Fact]
    public async Task The_dictionary_reports_a_masked_field_as_masked()
    {
        var scenario = DictionaryScenario.Create();

        var table = await scenario.CreateDictionary().GetTableAsync("mdm", "BusinessPartnerBank");

        Assert.NotNull(table);
        Assert.True(table.Fields.Single(f => f.FieldName == "Iban").IsMasked);
        Assert.False(table.Fields.Single(f => f.FieldName == "BusinessPartnerId").IsMasked);
    }

    [Fact]
    public async Task A_table_carries_the_foreign_keys_that_leave_it()
    {
        var scenario = DictionaryScenario.Create();

        var table = await scenario.CreateDictionary().GetTableAsync("fin", "JournalEntryHeader");

        var foreignKey = Assert.Single(table!.ForeignKeys);

        Assert.Equal("org", foreignKey.TargetSchemaName);
        Assert.Equal("CompanyCode", foreignKey.TargetTableName);
        Assert.Equal("CompanyCodeId", Assert.Single(foreignKey.Fields).SourceFieldName);
    }

    [Fact]
    public async Task Where_used_answers_from_the_other_end_of_the_same_key()
    {
        var scenario = DictionaryScenario.Create();

        var usages = await scenario.CreateDictionary().GetWhereUsedAsync("org", "CompanyCode");

        var usage = Assert.Single(usages);

        Assert.Equal("fin", usage.SourceSchemaName);
        Assert.Equal("JournalEntryHeader", usage.SourceTableName);
        Assert.Equal(new[] { "CompanyCodeId" }, usage.SourceFieldNames);
    }

    [Fact]
    public async Task The_generated_ddl_is_the_one_the_dictionary_rows_describe()
    {
        var scenario = DictionaryScenario.Create();

        var sql = await scenario.CreateDictionary()
            .GenerateCreateTableSqlAsync("fin", "JournalEntryHeader");

        Assert.NotNull(sql);
        Assert.Contains("CREATE TABLE [fin].[JournalEntryHeader]", sql, StringComparison.Ordinal);
        Assert.Contains("bigint IDENTITY(1,1) NOT NULL", sql, StringComparison.Ordinal);
        Assert.Contains("[DocumentNumber]", sql, StringComparison.Ordinal);

        // A field that is neither key nor required is nullable, and says so.
        Assert.Contains("nvarchar(25) NULL", sql, StringComparison.Ordinal);

        Assert.Contains(
            "CONSTRAINT [PK_fin_JournalEntryHeader] PRIMARY KEY CLUSTERED ([Id])",
            sql,
            StringComparison.Ordinal);

        Assert.Contains(
            "REFERENCES [org].[CompanyCode] ([Id]);",
            sql,
            StringComparison.Ordinal);
    }

    [Fact]
    public async Task The_generated_ddl_separates_members_with_commas_but_not_the_last_one()
    {
        var scenario = DictionaryScenario.Create();

        var sql = await scenario.CreateDictionary()
            .GenerateCreateTableSqlAsync("cfg", "Currency");

        Assert.NotNull(sql);

        // Comments trail each column, so the separator is checked on the
        // definition rather than on the whole line.
        var members = sql.Split(Environment.NewLine)
            .SkipWhile(line => line != "(")
            .Skip(1)
            .TakeWhile(line => !line.StartsWith(");", StringComparison.Ordinal))
            .Select(line => line.Split("  --")[0].TrimEnd())
            .ToArray();

        Assert.All(members[..^1], line => Assert.EndsWith(",", line, StringComparison.Ordinal));
        Assert.EndsWith("PRIMARY KEY CLUSTERED ([Id])", members[^1], StringComparison.Ordinal);
    }

    [Fact]
    public async Task The_ddl_for_a_table_that_does_not_exist_is_null()
    {
        var scenario = DictionaryScenario.Create();

        Assert.Null(await scenario.CreateDictionary()
            .GenerateCreateTableSqlAsync("fin", "Invented"));
    }

    [Fact]
    public async Task The_repository_browser_filters_by_type_and_name()
    {
        var scenario = DictionaryScenario.Create();
        var dictionary = scenario.CreateDictionary();

        Assert.Equal(4, (await dictionary.SearchObjectsAsync("Table")).Count);
        Assert.Empty(await dictionary.SearchObjectsAsync("Domain"));

        var match = Assert.Single(await dictionary.SearchObjectsAsync(search: "JournalEntry"));

        Assert.Equal("fin.JournalEntryHeader", match.ObjectName);
    }

    [Fact]
    public async Task The_repository_browser_caps_what_it_returns()
    {
        var scenario = DictionaryScenario.Create();

        var objects = await scenario.CreateDictionary().SearchObjectsAsync(maxResults: 2);

        Assert.Equal(2, objects.Count);
    }
}
