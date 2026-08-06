using ErpS4.Application.Posting;
using ErpS4.Application.Workflow;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using Xunit;

namespace ErpS4.IntegrationTests;

/// <summary>
/// Park, approve and post against a real server, using the approval workflow
/// the sample dataset configures: two sequential steps above 10 000 USD.
/// </summary>
[Collection(DatabaseCollection.Name)]
public sealed class WorkflowDatabaseTests(DatabaseFixture fixture)
{
    private const string CompanyCode = "KH01";
    private const string Expense = "500000";
    private const string Cash = "110000";
    private const string CostCenter = "1000";
    private const string ProfitCenter = "PC1000";

    private static readonly DateOnly PostingDate = new(2026, 2, 24);

    private static JournalEntryDraft Draft(decimal amount) =>
        new JournalEntryDraft(CompanyCode, "SA", PostingDate, PostingDate, "USD")
            {
                HeaderText = "Workflow integration document",
            }
            .AddLine(new JournalEntryDraftLine("40", Expense, new Money(amount, "USD"))
            {
                CostCenter = CostCenter,
                ProfitCenter = ProfitCenter,
            })
            .AddLine(new JournalEntryDraftLine("50", Cash, new Money(-amount, "USD"))
            {
                ProfitCenter = ProfitCenter,
            });

    [RequiresDatabaseFact]
    public async Task The_seeded_rule_decides_whether_approval_is_needed()
    {
        using var scope = fixture.CreateScope();
        var workflow = scope.ServiceProvider.GetRequiredService<IWorkflowService>();

        // The seeded rule filters on SourceModule, so it has to be supplied:
        // a rule filter nobody can satisfy is a rule that never fires.
        var small = await workflow.EvaluateAsync(
            new WorkflowContext("JournalEntry", CompanyCode, 500m, "USD")
            {
                SourceModule = "FI",
            });
        var large = await workflow.EvaluateAsync(
            new WorkflowContext("JournalEntry", CompanyCode, 25_000m, "USD")
            {
                SourceModule = "FI",
            });

        Assert.False(small.IsRequired);
        Assert.True(large.IsRequired);
        Assert.Equal("JE_APPROVAL", large.WorkflowCode);
        Assert.Equal(2, large.StepCount);
    }

    [RequiresDatabaseFact]
    public async Task A_parked_document_has_a_number_and_lines_but_no_ledger_effect()
    {
        using var scope = fixture.CreateScope();
        var engine = scope.ServiceProvider.GetRequiredService<IPostingEngine>();

        var parked = await engine.ParkAsync(new PostingRequest(Draft(25_000m)));

        Assert.True(parked.IsSuccess,
            string.Join("; ", parked.Errors.Select(e => e.ToString())));
        Assert.NotNull(parked.DocumentNumber);

        using var reader = fixture.CreateScope();
        var context = reader.ServiceProvider.GetRequiredService<ErpDbContext>();

        var header = await context.Set<JournalEntryHeader>()
            .SingleAsync(h => h.DocumentNumber == parked.DocumentNumber);

        Assert.Equal("Parked", header.Status);

        // Lines exist; the things that make a document count do not.
        Assert.NotEmpty(await context.Set<JournalEntryLine>()
            .Where(l => l.JournalEntryHeaderId == header.Id).ToListAsync());

        Assert.Empty(await context.Set<OpenItem>()
            .Where(o => o.DocumentNumber == parked.DocumentNumber).ToListAsync());
    }

    [RequiresDatabaseFact]
    public async Task Only_the_last_approval_gives_the_document_its_ledger_effect()
    {
        var parked = await ParkAndSubmitAsync(30_000m);

        // Step one: the approver. The document is still parked afterwards.
        using var first = CreateScopeAs("kss.approver");
        var stepOne = await first.ServiceProvider.GetRequiredService<IWorkflowService>()
            .ApproveAsync(new ApprovalRequest(parked.InstanceNumber!, "Checked"));

        Assert.True(stepOne.IsSuccess, Explain(stepOne));
        Assert.Equal("PendingApproval", await StatusAsync(parked.DocumentNumber));

        // Step two: the director. Only now does it post.
        using var second = CreateScopeAs("kss.admin");   // holds FI_DIRECTOR
        var stepTwo = await second.ServiceProvider.GetRequiredService<IWorkflowService>()
            .ApproveAsync(new ApprovalRequest(parked.InstanceNumber!, "Approved"));

        Assert.True(stepTwo.IsSuccess, Explain(stepTwo));
        Assert.Equal("Posted", await StatusAsync(parked.DocumentNumber));

        // Posting is what creates the balances, and it happened once.
        using var reader = fixture.CreateScope();
        var lines = await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<JournalEntryLine>()
            .Where(l => l.DocumentNumber == parked.DocumentNumber)
            .ToListAsync();

        Assert.Equal(2, lines.Count);
        Assert.Equal(0m, lines.Sum(l => l.AmountInDocumentCurrency));
    }

    [RequiresDatabaseFact]
    public async Task The_submitter_cannot_approve_their_own_document()
    {
        // Submitted by someone who holds the approver role themselves, so
        // what refuses them is maker-checker, not a missing authorisation.
        var parked = await ParkAndSubmitAsync(40_000m, submitter: "kss.approver");

        using var scope = CreateScopeAs("kss.approver");
        var refused = await scope.ServiceProvider.GetRequiredService<IWorkflowService>()
            .ApproveAsync(new ApprovalRequest(parked.InstanceNumber!));

        Assert.False(refused.IsSuccess);
        Assert.Equal("PendingApproval", await StatusAsync(parked.DocumentNumber));

        // The other holder of the same role can, which is what makes the
        // refusal maker-checker rather than a deadlock.
        using var colleague = CreateScopeAs("kss.approver2");
        var accepted = await colleague.ServiceProvider.GetRequiredService<IWorkflowService>()
            .ApproveAsync(new ApprovalRequest(parked.InstanceNumber!));

        Assert.True(accepted.IsSuccess, Explain(accepted));
    }

    [RequiresDatabaseFact]
    public async Task A_rejection_leaves_the_ledger_untouched()
    {
        var parked = await ParkAndSubmitAsync(35_000m);

        using var scope = CreateScopeAs("kss.approver");
        var decision = await scope.ServiceProvider.GetRequiredService<IWorkflowService>()
            .RejectAsync(new ApprovalRequest(parked.InstanceNumber!, "Wrong cost centre"));

        Assert.True(decision.IsSuccess, Explain(decision));
        Assert.NotEqual("Posted", await StatusAsync(parked.DocumentNumber));

        using var reader = fixture.CreateScope();
        Assert.Empty(await reader.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<OpenItem>()
            .Where(o => o.DocumentNumber == parked.DocumentNumber)
            .ToListAsync());
    }

    /// <summary>A scope whose <see cref="ICurrentUser"/> is the named user.</summary>
    private IServiceScope CreateScopeAs(string userName) => fixture.CreateScopeAs(userName);

    private async Task<(string? DocumentNumber, string? InstanceNumber)> ParkAndSubmitAsync(
        decimal amount,
        string submitter = "kss.accountant")
    {
        using var scope = CreateScopeAs(submitter);

        var parked = await scope.ServiceProvider.GetRequiredService<IPostingEngine>()
            .ParkAsync(new PostingRequest(Draft(amount)));

        Assert.True(parked.IsSuccess,
            string.Join("; ", parked.Errors.Select(e => e.ToString())));

        var header = await scope.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<JournalEntryHeader>()
            .SingleAsync(h => h.DocumentNumber == parked.DocumentNumber);

        var submission = await scope.ServiceProvider.GetRequiredService<IWorkflowService>()
            .SubmitAsync(new SubmitForApprovalRequest(
                "JournalEntry", header.Id, CompanyCode, amount, "USD")
            {
                DocumentNumber = parked.DocumentNumber,
                FiscalYear = parked.FiscalYear,
                SourceModule = "FI",
            });

        Assert.True(submission.IsSuccess,
            string.Join("; ", submission.Violations.Select(v => v.ToString())));
        Assert.True(submission.ApprovalRequired);

        return (parked.DocumentNumber, submission.InstanceNumber);
    }

    private async Task<string> StatusAsync(string? documentNumber)
    {
        using var scope = fixture.CreateScope();
        return await scope.ServiceProvider.GetRequiredService<ErpDbContext>()
            .Set<JournalEntryHeader>()
            .Where(h => h.DocumentNumber == documentNumber)
            .Select(h => h.Status)
            .SingleAsync();
    }

    private static string Explain(DecisionResult result) =>
        string.Join("; ",
            result.Violations.Select(v => v.ToString())
                .Concat(result.PostingErrors.Select(e => e.ToString())));
}
