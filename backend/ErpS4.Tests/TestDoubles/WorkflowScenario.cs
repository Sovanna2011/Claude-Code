using ErpS4.Application.Posting;
using ErpS4.Application.Workflow;
using ErpS4.Database;
using ErpS4.Database.Entities;
using Microsoft.Extensions.Logging.Abstractions;

namespace ErpS4.Tests.TestDoubles;

/// <summary>
/// Adds users, roles and an approval workflow to a <see cref="PostingScenario"/>:
/// documents of 1,000 or more need approval, and maker-checker is on.
/// </summary>
public static class WorkflowScenario
{
    public const string WorkflowCode = "JE_APPROVAL";
    public const decimal Threshold = 1_000m;

    /// <param name="steps">1 for a single approval, 2 to add a director step.</param>
    public static PostingScenario Configure(PostingScenario scenario, int steps = 1)
    {
        var approverRole = new Role
        {
            TenantId = 1, RoleCode = "FI_APPROVER", Name = "Financial approver",
            RoleType = "Single", ValidFrom = new DateOnly(2000, 1, 1),
            ValidTo = new DateOnly(9999, 12, 31),
        };
        var directorRole = new Role
        {
            TenantId = 1, RoleCode = "FI_DIRECTOR", Name = "Finance director",
            RoleType = "Single", ValidFrom = new DateOnly(2000, 1, 1),
            ValidTo = new DateOnly(9999, 12, 31),
        };
        scenario.Context.Seed(approverRole, directorRole);

        var clerk = SeedUser(scenario, "clerk", "Sok Dara");
        var approver = SeedUser(scenario, "approver", "Chan Sopheak");
        var director = SeedUser(scenario, "director", "Ly Vanna");
        SeedUser(scenario, "stranger", "Not involved");

        // The clerk also holds the approver role: maker-checker, not a missing
        // authorisation, is what keeps them off their own document.
        Assign(scenario, clerk, approverRole);
        Assign(scenario, approver, approverRole);
        Assign(scenario, director, directorRole);

        var definition = new WorkflowDefinition
        {
            TenantId = 1,
            WorkflowCode = WorkflowCode,
            Version = 1,
            Name = "Journal entry approval",
            ObjectType = "JournalEntry",
            ApprovalMode = "Sequential",
            IsMakerCheckerEnforced = true,
            AllowDelegation = true,
            AllowResubmission = true,
            IsActive = true,
            EffectiveFrom = new DateOnly(2000, 1, 1),
        };
        scenario.Context.Seed(definition);

        scenario.Context.Seed(new WorkflowRule
        {
            TenantId = 1,
            WorkflowDefinitionId = definition.Id,
            RuleSequence = 10,
            Name = "Journal entries of 1,000 or more",
            MinimumAmount = Threshold,
            IsActive = true,
        });

        scenario.Context.Seed(new WorkflowStep
        {
            TenantId = 1,
            WorkflowDefinitionId = definition.Id,
            StepNumber = 1,
            Name = "Financial approval",
            StepType = "Approval",
            ApproverDeterminationType = "Role",
            ApproverRoleId = approverRole.Id,
            RequiredApprovals = 1,
            OnRejectAction = "ReturnToSubmitter",
        });

        if (steps > 1)
        {
            scenario.Context.Seed(new WorkflowStep
            {
                TenantId = 1,
                WorkflowDefinitionId = definition.Id,
                StepNumber = 2,
                Name = "Director approval",
                StepType = "Approval",
                ApproverDeterminationType = "Role",
                ApproverRoleId = directorRole.Id,
                RequiredApprovals = 1,
                OnRejectAction = "ReturnToSubmitter",
            });
        }

        return scenario;
    }

    /// <summary>Parks a document as <paramref name="submitter"/> and submits it.</summary>
    public static async Task<SubmissionResult> ParkAndSubmitAsync(
        PostingScenario scenario,
        string submitter,
        decimal amount)
    {
        var parked = await scenario.CreateEngineAs(submitter)
            .ParkAsync(new PostingRequest(PostingScenario.BalancedDraft(amount)));

        var header = scenario.Context.Set<JournalEntryHeader>()
            .Single(h => h.DocumentNumber == parked.DocumentNumber);

        var service = new WorkflowService(
            scenario.Context,
            scenario.CreateEngineAs(submitter),
            new FixedTenantProvider(1),
            new FixedCurrentUser(submitter),
            scenario.Clock,
            NullLogger<WorkflowService>.Instance);

        return await service.SubmitAsync(new SubmitForApprovalRequest(
            "JournalEntry",
            header.Id,
            PostingScenario.CompanyCodeKey,
            amount,
            "USD")
        {
            DocumentNumber = parked.DocumentNumber,
            ObjectDescription = parked.DocumentNumber,
            FiscalYear = parked.FiscalYear,
        });
    }

    public static void AddSubstitution(PostingScenario scenario, string absent, string substitute)
    {
        var absentUser = scenario.Context.Set<User>().Single(u => u.UserName == absent);
        var substituteUser = scenario.Context.Set<User>().Single(u => u.UserName == substitute);

        scenario.Context.Seed(new UserSubstitution
        {
            TenantId = 1,
            UserId = absentUser.Id,
            SubstituteUserId = substituteUser.Id,
            SubstitutionType = "Approval",
            IsActive = true,
            Reason = "Annual leave",
            ValidFrom = new DateOnly(2026, 1, 1),
            ValidTo = new DateOnly(2026, 12, 31),
        });
    }

    private static User SeedUser(PostingScenario scenario, string userName, string displayName)
    {
        var user = new User
        {
            TenantId = 1,
            UserName = userName,
            DisplayName = displayName,
            Email = $"{userName}@example.com",
            UserType = "Dialog",
            LanguageCode = "EN",
            TimeZoneId = "Asia/Phnom_Penh",
            Status = "Active",
            IsLocked = false,
            FailedLoginCount = 0,
            ValidFrom = new DateOnly(2000, 1, 1),
            ValidTo = new DateOnly(9999, 12, 31),
        };

        scenario.Context.Seed(user);
        return user;
    }

    private static void Assign(PostingScenario scenario, User user, Role role) =>
        scenario.Context.Seed(new UserRole
        {
            TenantId = 1,
            UserId = user.Id,
            RoleId = role.Id,
            AssignedBy = "setup",
            AssignedAt = new DateTime(2026, 1, 1, 0, 0, 0, DateTimeKind.Utc),
            ValidFrom = new DateOnly(2000, 1, 1),
            ValidTo = new DateOnly(9999, 12, 31),
        });
}
