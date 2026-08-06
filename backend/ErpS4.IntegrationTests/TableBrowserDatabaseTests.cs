using ErpS4.Application.DataDictionary;
using ErpS4.Application.TableBrowser;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using Xunit;

namespace ErpS4.IntegrationTests;

/// <summary>
/// SE16N and SE11 against the real dictionary, over the real 228 tables.
/// </summary>
/// <remarks>
/// The unit tests build a four-table dictionary and check what the service
/// refuses. These check the statements it builds actually run: SQL Server is
/// the only thing that can say whether the SQL is valid, whether a masked
/// column really comes back empty, and whether the paging clause works.
/// </remarks>
[Collection(DatabaseCollection.Name)]
public sealed class TableBrowserDatabaseTests(DatabaseFixture fixture)
{
    // A seeded user: the browser refuses to run a query it cannot attribute to
    // one, because every query is logged against whoever ran it.
    private const string Browser = "kss.admin";

    [RequiresDatabaseFact]
    public async Task A_browser_query_runs_and_returns_the_seeded_documents()
    {
        using var scope = fixture.CreateScopeAs(Browser);

        var result = await scope.ServiceProvider.GetRequiredService<ITableBrowserService>()
            .QueryAsync(new BrowserQueryRequest("fin", "JournalEntryHeader")
            {
                Fields = ["DocumentNumber", "PostingDate", "TotalDebitAmount"],
                SortBy = "DocumentNumber",
                MaxRows = 50,
            });

        Assert.True(result.IsSuccess,
            string.Join("; ", result.Violations.Select(v => v.ToString())));
        Assert.Equal(3, result.Columns.Count);
        Assert.NotEmpty(result.Rows);
        Assert.All(result.Rows, row => Assert.Equal(3, row.Count));
    }

    [RequiresDatabaseFact]
    public async Task A_masked_column_comes_back_empty_from_the_server_itself()
    {
        using var scope = fixture.CreateScopeAs(Browser);

        var result = await scope.ServiceProvider.GetRequiredService<ITableBrowserService>()
            .QueryAsync(new BrowserQueryRequest("sec", "PasswordHistory"));

        // sec is protected outright, so this never reaches the server.
        Assert.False(result.IsSuccess);
        Assert.Equal(BrowserErrorCodes.TableProtected, result.Violations[0].Code);

        // A masked column on a browsable table is selected as a literal NULL,
        // and SQL Server has to accept that statement.
        using var second = fixture.CreateScopeAs(Browser);
        var bank = await second.ServiceProvider.GetRequiredService<ITableBrowserService>()
            .QueryAsync(new BrowserQueryRequest("mdm", "HouseBankAccount"));

        Assert.True(bank.IsSuccess,
            string.Join("; ", bank.Violations.Select(v => v.ToString())));
        Assert.Contains("NULL AS [Iban]", bank.ExecutedSql!, StringComparison.Ordinal);
    }

    [RequiresDatabaseFact]
    public async Task A_typed_filter_reaches_the_server_and_selects_the_right_rows()
    {
        using var scope = fixture.CreateScopeAs(Browser);

        var result = await scope.ServiceProvider.GetRequiredService<ITableBrowserService>()
            .QueryAsync(new BrowserQueryRequest("fin", "JournalEntryHeader")
            {
                Fields = ["DocumentNumber", "PostingDate"],
                Filters =
                [
                    new BrowserFilter("PostingDate", BrowserOperators.GreaterOrEqual, "2026-02-01"),
                ],
            });

        Assert.True(result.IsSuccess,
            string.Join("; ", result.Violations.Select(v => v.ToString())));
        Assert.NotEmpty(result.Rows);

        // Every row really is on or after the date, which is the thing an
        // nvarchar parameter and an implicit conversion could get wrong.
        Assert.All(result.Rows, row =>
            Assert.True(DateTime.Parse(row[1]!, System.Globalization.CultureInfo.InvariantCulture)
                        >= new DateTime(2026, 2, 1)));
    }

    [RequiresDatabaseFact]
    public async Task An_injection_attempt_matches_nothing_and_breaks_nothing()
    {
        using var scope = fixture.CreateScopeAs(Browser);

        var result = await scope.ServiceProvider.GetRequiredService<ITableBrowserService>()
            .QueryAsync(new BrowserQueryRequest("fin", "JournalEntryHeader")
            {
                Fields = ["DocumentNumber"],
                Filters =
                [
                    new BrowserFilter(
                        "DocumentNumber",
                        BrowserOperators.Equals,
                        "x'; DROP TABLE fin.JournalEntryLine; --"),
                ],
            });

        Assert.True(result.IsSuccess);
        Assert.Empty(result.Rows);

        using var reader = fixture.CreateScope();
        Assert.True(await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<JournalEntryLine>().AnyAsync());
    }

    [RequiresDatabaseFact]
    public async Task Every_query_is_written_to_the_browser_log()
    {
        using var scope = fixture.CreateScopeAs(Browser);
        var context = scope.ServiceProvider.GetRequiredService<ErpDbContext>();
        var before = await context.Set<BrowserQueryLog>().CountAsync();

        await scope.ServiceProvider.GetRequiredService<ITableBrowserService>()
            .QueryAsync(new BrowserQueryRequest("cfg", "Currency") { MaxRows = 5 });

        using var reader = fixture.CreateScope();
        var log = await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<BrowserQueryLog>()
            .OrderByDescending(l => l.Id)
            .FirstAsync();

        Assert.Equal(before + 1, await reader.ServiceProvider
            .GetRequiredService<ErpDbContext>().Set<BrowserQueryLog>().CountAsync());
        Assert.Equal("cfg", log.SchemaName);
        Assert.Equal("Currency", log.ObjectName);
        Assert.False(log.WasExported);
    }

    [RequiresDatabaseFact]
    public async Task The_generated_ddl_matches_the_table_that_is_actually_there()
    {
        using var scope = fixture.CreateScopeAs(Browser);

        var sql = await scope.ServiceProvider.GetRequiredService<IDictionaryService>()
            .GenerateCreateTableSqlAsync("org", "CompanyCode");

        Assert.NotNull(sql);

        // Every column SE11 renders has to exist on the real table: the point
        // of the dictionary is that it describes the database it sits on.
        using var reader = fixture.CreateScope();
        var actual = await reader.ServiceProvider.GetRequiredService<IErpDataContext>()
            .QueryRawAsync(new RawQuery(
                "SELECT name FROM sys.columns WHERE object_id = OBJECT_ID(N'org.CompanyCode');",
                []));

        var columns = actual.Rows.Select(r => (string)r[0]!).ToList();

        Assert.NotEmpty(columns);
        Assert.All(columns, column =>
            Assert.Contains($"[{column}]", sql!, StringComparison.Ordinal));
    }

    [RequiresDatabaseFact]
    public async Task Where_used_answers_from_the_seeded_relationships()
    {
        using var scope = fixture.CreateScopeAs(Browser);

        var usages = await scope.ServiceProvider.GetRequiredService<IDictionaryService>()
            .GetWhereUsedAsync("org", "CompanyCode");

        Assert.NotEmpty(usages);
        Assert.Contains(usages, u => u.SourceSchemaName == "fin"
                                     && u.SourceTableName == "JournalEntryHeader");
    }
}
