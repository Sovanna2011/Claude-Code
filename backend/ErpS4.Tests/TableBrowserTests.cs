using ErpS4.Application.TableBrowser;
using ErpS4.Database.Entities;
using ErpS4.Tests.TestDoubles;
using Xunit;

namespace ErpS4.Tests;

/// <summary>
/// SE16N is a query builder pointed at the whole database, so nearly every one
/// of these tests is a security test: what it refuses, what it never puts in a
/// statement, and what it records about who looked.
/// </summary>
public sealed class TableBrowserTests
{
    private static BrowserQueryRequest Request(
        string schemaName = "fin",
        string tableName = "JournalEntryHeader") =>
        new(schemaName, tableName);

    private static string Sql(DictionaryScenario scenario) =>
        scenario.Context.RawQueries.Single().Sql;

    [Fact]
    public async Task A_table_that_is_not_in_the_dictionary_is_refused()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(
            Request("fin", "SomethingInvented"));

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.TableUnknown, result.Violations[0].Code);
        Assert.Empty(scenario.Context.RawQueries);
    }

    [Fact]
    public async Task A_security_table_is_refused_however_the_caller_asks()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(Request("sec", "User"));

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.TableProtected, result.Violations[0].Code);
        Assert.Empty(scenario.Context.RawQueries);
    }

    [Fact]
    public async Task A_protected_table_is_absent_from_the_browsable_list()
    {
        var scenario = DictionaryScenario.Create();

        var tables = await scenario.CreateBrowser().GetBrowsableTablesAsync();

        Assert.DoesNotContain(tables, t => t.SchemaName == "sec");
        Assert.Contains(tables, t => t.TableName == "JournalEntryHeader");
    }

    [Fact]
    public async Task A_field_that_does_not_exist_cannot_be_filtered_on()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Filters = [new BrowserFilter("DocumentNumber; DROP TABLE fin.JournalEntryHeader", "EQ", "1")],
        });

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.FieldUnknown, result.Violations[0].Code);
        Assert.Empty(scenario.Context.RawQueries);
    }

    [Fact]
    public async Task An_operator_the_browser_does_not_know_is_refused()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Filters = [new BrowserFilter("DocumentNumber", "OR 1=1 --", "1")],
        });

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.OperatorUnknown, result.Violations[0].Code);
    }

    [Fact]
    public async Task A_filter_value_travels_as_a_parameter_and_never_as_text()
    {
        var scenario = DictionaryScenario.Create();
        const string Hostile = "1000'; DROP TABLE fin.JournalEntryHeader; --";

        await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Filters = [new BrowserFilter("DocumentNumber", BrowserOperators.Equals, Hostile)],
        });

        var query = scenario.Context.RawQueries.Single();

        Assert.DoesNotContain("DROP TABLE", query.Sql, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("[DocumentNumber] = @p", query.Sql, StringComparison.Ordinal);
        Assert.Contains(query.Parameters, p => Equals(p.Value, Hostile));
    }

    [Fact]
    public async Task Wildcards_typed_by_the_user_match_themselves()
    {
        var scenario = DictionaryScenario.Create();

        await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Filters = [new BrowserFilter("HeaderText", BrowserOperators.Contains, "50%_off")],
        });

        var value = Assert.IsType<string>(
            scenario.Context.RawQueries.Single().Parameters.Last(p => p.Value is string).Value);

        Assert.Equal(@"%50\%\_off%", value);
        Assert.Contains(@"ESCAPE '\'", Sql(scenario), StringComparison.Ordinal);
    }

    [Fact]
    public async Task Every_query_is_filtered_by_tenant_whether_the_caller_asked_or_not()
    {
        var scenario = DictionaryScenario.Create();

        await scenario.CreateBrowser().QueryAsync(Request());

        var query = scenario.Context.RawQueries.Single();

        Assert.Contains("[TenantId] = @tenantId", query.Sql, StringComparison.Ordinal);
        Assert.Contains(query.Parameters, p => p.Name == "@tenantId" && Equals(p.Value, 1));
    }

    [Fact]
    public async Task The_row_cap_of_the_authorization_group_beats_the_request()
    {
        var scenario = DictionaryScenario.Create();

        // FINC allows 500; the caller asks for far more.
        var result = await scenario.CreateBrowser().QueryAsync(Request() with { MaxRows = 100_000 });

        Assert.Equal(500, result.RowLimitApplied);

        var limit = scenario.Context.RawQueries.Single().Parameters
            .Single(p => p.Name == "@limit").Value;

        // One row over the cap, so truncation can be detected rather than guessed.
        Assert.Equal(501, limit);
    }

    [Fact]
    public async Task A_result_cut_short_by_the_cap_says_so()
    {
        var scenario = DictionaryScenario.Create();
        var columns = new[] { "Id", "TenantId", "CompanyCodeId", "DocumentNumber", "PostingDate", "HeaderText" };
        scenario.ReturnRows(
            columns,
            [.. Enumerable.Range(0, 3).Select(index => new object?[]
            {
                (long)index, 1, 1L, $"100{index}", new DateTime(2026, 3, 1), "text",
            })]);

        var result = await scenario.CreateBrowser().QueryAsync(Request() with { MaxRows = 2 });

        Assert.True(result.WasTruncated);
        Assert.Equal(2, result.RowCount);
    }

    [Fact]
    public async Task A_masked_column_is_never_read_and_comes_back_as_asterisks()
    {
        var scenario = DictionaryScenario.Create();
        scenario.ReturnRows(
            ["Id", "TenantId", "BusinessPartnerId", "BankAccountNumber", "Iban"],
            new object?[] { 1L, 1, 7L, null, null });

        var result = await scenario.CreateBrowser().QueryAsync(Request("mdm", "BusinessPartnerBank"));

        Assert.True(result.IsSuccess);
        Assert.Contains("NULL AS [Iban]", Sql(scenario), StringComparison.Ordinal);
        Assert.DoesNotContain("[Iban],", Sql(scenario), StringComparison.Ordinal);

        var iban = result.Columns.Select((column, index) => (column, index))
            .Single(pair => pair.column.FieldName == "Iban").index;

        Assert.Equal("********", result.Rows[0][iban]);
    }

    [Fact]
    public async Task A_masked_field_cannot_be_used_as_a_filter()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(
            Request("mdm", "BusinessPartnerBank") with
            {
                Filters = [new BrowserFilter("Iban", BrowserOperators.StartsWith, "KH12")],
            });

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.FieldMasked, result.Violations[0].Code);
        Assert.Empty(scenario.Context.RawQueries);
    }

    [Fact]
    public async Task A_masked_field_cannot_be_sorted_on_either()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(
            Request("mdm", "BusinessPartnerBank") with { SortBy = "Iban" });

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.FieldMasked, result.Violations[0].Code);
    }

    [Fact]
    public async Task Export_is_refused_when_the_group_does_not_allow_it()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(
            Request("mdm", "BusinessPartnerBank") with { IsExport = true });

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.ExportNotAllowed, result.Violations[0].Code);
        Assert.Empty(scenario.Context.RawQueries);
    }

    [Fact]
    public async Task Asking_for_a_company_code_on_a_table_that_has_none_is_reported()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(
            Request("cfg", "Currency") with { CompanyCode = "1000" });

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.CompanyCodeNotApplicable, result.Violations[0].Code);
    }

    [Fact]
    public async Task A_company_code_becomes_a_condition_on_the_surrogate_key()
    {
        var scenario = DictionaryScenario.Create();

        await scenario.CreateBrowser().QueryAsync(Request() with { CompanyCode = "1000" });

        var query = scenario.Context.RawQueries.Single();

        Assert.Contains("[CompanyCodeId] = @companyCodeId", query.Sql, StringComparison.Ordinal);
        Assert.Contains(query.Parameters, p => p.Name == "@companyCodeId");
    }

    [Fact]
    public async Task Paging_is_ordered_by_a_unique_column_as_well_as_the_requested_one()
    {
        var scenario = DictionaryScenario.Create();

        await scenario.CreateBrowser().QueryAsync(Request() with { SortBy = "PostingDate" });

        Assert.Contains(
            "ORDER BY [PostingDate] ASC, [Id] ASC",
            Sql(scenario),
            StringComparison.Ordinal);
    }

    [Fact]
    public async Task Every_query_is_logged_with_who_ran_it_and_what_came_back()
    {
        var scenario = DictionaryScenario.Create();
        scenario.ReturnRows(["Id"], new object?[] { 1L });

        await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Fields = ["DocumentNumber"],
            Filters = [new BrowserFilter("DocumentNumber", BrowserOperators.Equals, "1000000001")],
        });

        var log = Assert.Single(scenario.Context.Set<BrowserQueryLog>());

        Assert.Equal("fin", log.SchemaName);
        Assert.Equal("JournalEntryHeader", log.ObjectName);
        Assert.Equal("DocumentNumber", log.SelectedFields);
        Assert.Contains("1000000001", log.FilterJson);
        Assert.Equal(1, log.RowsReturned);
        Assert.False(log.WasExported);
        Assert.Equal(scenario.Clock.Now.UtcDateTime, log.ExecutedAt);
    }

    [Fact]
    public async Task One_filter_cannot_spend_every_parameter_the_server_allows()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Filters =
            [
                new BrowserFilter("DocumentNumber", BrowserOperators.In)
                {
                    Values = [.. Enumerable.Range(0, TableBrowserService.MaxValuesPerFilter + 1)
                        .Select(index => index.ToString())],
                },
            ],
        });

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.TooManyValues, result.Violations[0].Code);
        Assert.Empty(scenario.Context.RawQueries);
    }

    [Fact]
    public async Task A_value_reaches_the_server_as_the_type_the_column_holds()
    {
        var scenario = DictionaryScenario.Create();

        await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Filters =
            [
                new BrowserFilter("PostingDate", BrowserOperators.GreaterOrEqual, "2026-03-01"),
                new BrowserFilter("CompanyCodeId", BrowserOperators.Equals, "42"),
            ],
        });

        var parameters = scenario.Context.RawQueries.Single().Parameters;

        // Not "2026-03-01" and "42" as text: a date column compared against an
        // nvarchar parameter is at the mercy of the session's DATEFORMAT.
        Assert.Contains(parameters, p => Equals(p.Value, new DateOnly(2026, 3, 1)));
        Assert.Contains(parameters, p => Equals(p.Value, 42L));
    }

    [Fact]
    public async Task A_value_that_is_not_the_columns_type_is_a_violation_not_an_exception()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Filters = [new BrowserFilter("PostingDate", BrowserOperators.Equals, "not a date")],
        });

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.ValueInvalid, result.Violations[0].Code);
        Assert.Empty(scenario.Context.RawQueries);
    }

    [Fact]
    public async Task A_pattern_filter_is_refused_on_a_column_that_is_not_text()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Filters = [new BrowserFilter("PostingDate", BrowserOperators.Contains, "2026")],
        });

        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.PatternOnNonTextField, result.Violations[0].Code);
    }

    [Fact]
    public async Task Every_broken_rule_is_reported_in_one_pass()
    {
        var scenario = DictionaryScenario.Create();

        var result = await scenario.CreateBrowser().QueryAsync(Request() with
        {
            Filters =
            [
                new BrowserFilter("NotAField", BrowserOperators.Equals, "x"),
                new BrowserFilter("DocumentNumber", "XX", "y"),
                new BrowserFilter("DocumentNumber", BrowserOperators.Between, "a"),
            ],
            SortBy = "AlsoNotAField",
        });

        Assert.False(result.IsSuccess);
        Assert.Equal(4, result.Violations.Count);
    }
}
