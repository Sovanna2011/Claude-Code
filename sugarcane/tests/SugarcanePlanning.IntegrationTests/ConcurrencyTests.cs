using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Contracts.Scheduling;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;
using SugarcanePlanning.Infrastructure.Persistence;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// The conflict checks and the "no overlapping plan for a block" rule are read-then-write:
/// they query, decide, then insert. Under concurrency that is a time-of-check/time-of-use race
/// unless the database holds a lock across both steps. These tests fire genuinely simultaneous
/// requests through separate scopes — one per connection, exactly as the API would — and assert
/// that only one of them wins.
/// </summary>
public class ConcurrencyTests : IClassFixture<SqlServerFixture>
{
    private readonly SqlServerFixture _fixture;

    public ConcurrencyTests(SqlServerFixture fixture) => _fixture = fixture;

    /// <summary>Approves a projection and generates its plans, returning the planting plan.</summary>
    private async Task<(int PlanId, DateOnly Date)> ArrangePlanAsync(string blockCode)
    {
        var blockId = await _fixture.NewBlockAsync(blockCode);

        using var scope = _fixture.CreateScope();
        var projections = scope.ServiceProvider.GetRequiredService<IProjectionService>();
        var plans = scope.ServiceProvider.GetRequiredService<IActivityPlanService>();

        var projection = await projections.CreateAsync(new ProjectionCreateDto
        {
            EstateId = _fixture.Seed.EstateId,
            GrowingSeasonId = _fixture.Seed.SeasonId,
            ProjectionDate = new DateOnly(2026, 2, 1),
            PlanningStartDate = new DateOnly(2026, 3, 1),
            PlanningEndDate = new DateOnly(2026, 8, 31),
            Lines =
            {
                new ProjectionLineUpsertDto
                {
                    BlockId = blockId,
                    CaneVarietyId = _fixture.Seed.VarietyId,
                    CropType = CropType.NewPlanting,
                    ProjectedPlantingAreaHa = 40m,
                    PlannedPlantingStart = new DateOnly(2026, 3, 2),
                    PlannedPlantingEnd = new DateOnly(2026, 3, 20),
                    ExpectedYieldPerHa = 90m,
                    ExpectedLossPercent = 5m
                }
            }
        });

        await projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });
        await plans.GenerateAsync(new GenerateActivityPlanRequest { ProjectionId = projection.Id });

        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
        var plan = await db.ActivityPlans
            .Where(p => p.ProjectionId == projection.Id && p.ActivityId == _fixture.Seed.PloughActivityId)
            .SingleAsync();

        return (plan.Id, plan.PlannedStartDate);
    }

    private ResourceScheduleUpsertDto Booking(int planId, DateOnly date) => new()
    {
        ActivityPlanId = planId,
        ScheduleDate = date,
        PlannedStart = date.ToDateTime(new TimeOnly(7, 0)),
        PlannedEnd = date.ToDateTime(new TimeOnly(15, 0)),
        TractorId = _fixture.Seed.TractorId,
        PlannedAreaHa = 4m,
        ExpectedWorkingHours = 8m
    };

    /// <summary>Runs the same operation from several scopes at once, collecting the outcomes.</summary>
    private async Task<(int Succeeded, int Rejected, List<string> Errors)> RaceAsync(
        int attempts, Func<IServiceScope, Task> operation)
    {
        var gate = new TaskCompletionSource();
        var runs = Enumerable.Range(0, attempts).Select(async _ =>
        {
            using var scope = _fixture.CreateScope();
            await gate.Task;                       // release every attempt at the same moment
            try
            {
                await operation(scope);
                return (Ok: true, Error: (string?)null);
            }
            catch (Exception ex)
            {
                // The stable rule code lives on the exception, not in the message, and it is the
                // code the API turns into the 422 body — so that is what the assertions check.
                var code = ex is BusinessRuleException rule ? rule.Code : ex.GetType().Name;
                return (Ok: false, Error: $"{code}: {ex.Message}");
            }
        }).ToArray();

        gate.SetResult();
        var outcomes = await Task.WhenAll(runs);

        return (outcomes.Count(o => o.Ok),
                outcomes.Count(o => !o.Ok),
                outcomes.Where(o => !o.Ok).Select(o => o.Error!).ToList());
    }

    [SqlServerFact]
    public async Task Simultaneous_bookings_of_one_tractor_cannot_both_win()
    {
        var (planId, date) = await ArrangePlanAsync("B-RACE1");

        // Book once first, so the plan is already Scheduled. Otherwise the racers collide on the
        // plan's row version instead of on the booking, and the real guard is never exercised.
        using (var warmUp = _fixture.CreateScope())
        {
            var first = Booking(planId, date);
            first.PlannedStart = date.ToDateTime(new TimeOnly(4, 0));
            first.PlannedEnd = date.ToDateTime(new TimeOnly(5, 0));
            first.TractorId = null;
            await warmUp.ServiceProvider.GetRequiredService<ISchedulingService>().CreateAsync(first);
        }

        var (succeeded, rejected, errors) = await RaceAsync(6, async scope =>
            await scope.ServiceProvider.GetRequiredService<ISchedulingService>()
                .CreateAsync(Booking(planId, date)));

        using var verify = _fixture.CreateScope();
        var db = verify.ServiceProvider.GetRequiredService<AppDbContext>();
        var stored = await db.Schedules.CountAsync(s =>
            s.ActivityPlanId == planId && s.TractorId == _fixture.Seed.TractorId
            && s.Status != ScheduleStatus.Cancelled);

        Assert.True(succeeded >= 1, "at least one booking must be accepted");
        Assert.Equal(1, stored);
        Assert.Equal(1, succeeded);
        Assert.Equal(5, rejected);
        Assert.All(errors, e => Assert.Contains("SCHEDULE_CONFLICT", e, StringComparison.OrdinalIgnoreCase));
    }

    [SqlServerFact]
    public async Task Simultaneous_projections_on_one_block_cannot_both_be_stored()
    {
        var blockId = await _fixture.NewBlockAsync("B-RACE2");

        ProjectionCreateDto Request() => new()
        {
            EstateId = _fixture.Seed.EstateId,
            GrowingSeasonId = _fixture.Seed.SeasonId,
            ProjectionDate = new DateOnly(2026, 2, 1),
            PlanningStartDate = new DateOnly(2026, 3, 1),
            PlanningEndDate = new DateOnly(2026, 8, 31),
            Lines =
            {
                new ProjectionLineUpsertDto
                {
                    BlockId = blockId,
                    CaneVarietyId = _fixture.Seed.VarietyId,
                    CropType = CropType.NewPlanting,
                    ProjectedPlantingAreaHa = 90m,     // two of these would exceed the 100 ha block
                    PlannedPlantingStart = new DateOnly(2026, 4, 1),
                    PlannedPlantingEnd = new DateOnly(2026, 4, 20),
                    ExpectedYieldPerHa = 90m,
                    ExpectedLossPercent = 5m
                }
            }
        };

        // Two drafts on the same block are deliberately legal — planners work in parallel. It is
        // the submission that commits the block, so both drafts are written first and only the
        // submissions race. Neither draft can see the other as committed at the moment it starts.
        var draftIds = new List<int>();
        using (var scope = _fixture.CreateScope())
        {
            var projections = scope.ServiceProvider.GetRequiredService<IProjectionService>();
            foreach (var _ in Enumerable.Range(0, 2))
                draftIds.Add((await projections.CreateAsync(Request())).Id);
        }

        var next = 0;
        var (succeeded, rejected, errors) = await RaceAsync(2, async scope =>
        {
            var draftId = draftIds[Interlocked.Increment(ref next) - 1];
            await scope.ServiceProvider.GetRequiredService<IProjectionService>()
                .ExecuteWorkflowAsync(draftId, new WorkflowActionDto { Action = ApprovalAction.Submit });
        });

        using var verify = _fixture.CreateScope();
        var db = verify.ServiceProvider.GetRequiredService<AppDbContext>();
        var overlapping = await db.ProjectionLines
            .Include(l => l.Projection)
            .Where(l => l.BlockId == blockId && l.Projection!.Status == ProjectionStatus.Submitted)
            .CountAsync();

        // The second submit must have been refused: the block is already committed for that window.
        Assert.Equal(1, overlapping);
        Assert.Equal(1, succeeded);
        Assert.Equal(1, rejected);
        Assert.All(errors, e => Assert.Contains("BLOCK_OVERLAP", e, StringComparison.OrdinalIgnoreCase));
    }

    [SqlServerFact]
    public async Task Simultaneous_projection_numbers_stay_unique()
    {
        var blocks = new List<int>();
        for (var i = 0; i < 4; i++) blocks.Add(await _fixture.NewBlockAsync($"B-NUM{i}"));

        var index = 0;
        var (succeeded, _, errors) = await RaceAsync(4, async scope =>
        {
            var blockId = blocks[Interlocked.Increment(ref index) - 1];
            await scope.ServiceProvider.GetRequiredService<IProjectionService>().CreateAsync(new ProjectionCreateDto
            {
                EstateId = _fixture.Seed.EstateId,
                GrowingSeasonId = _fixture.Seed.SeasonId,
                ProjectionDate = new DateOnly(2026, 2, 1),
                PlanningStartDate = new DateOnly(2026, 3, 1),
                PlanningEndDate = new DateOnly(2026, 8, 31),
                Lines =
                {
                    new ProjectionLineUpsertDto
                    {
                        BlockId = blockId,
                        CaneVarietyId = _fixture.Seed.VarietyId,
                        CropType = CropType.NewPlanting,
                        ProjectedPlantingAreaHa = 10m,
                        PlannedPlantingStart = new DateOnly(2026, 6, 1),
                        PlannedPlantingEnd = new DateOnly(2026, 6, 10),
                        ExpectedYieldPerHa = 90m,
                        ExpectedLossPercent = 5m
                    }
                }
            });
        });

        using var verify = _fixture.CreateScope();
        var db = verify.ServiceProvider.GetRequiredService<AppDbContext>();
        var numbers = await db.Projections
            .Where(p => p.EstateId == _fixture.Seed.EstateId)
            .Select(p => new { p.ProjectionNo, p.Version })
            .ToListAsync();

        // Whatever the outcome of the race, the stored numbers must never collide.
        Assert.Equal(numbers.Count, numbers.Distinct().Count());
        Assert.True(succeeded >= 1, $"at least one create must succeed; errors: {string.Join(" | ", errors)}");
    }

    [SqlServerFact]
    public async Task Simultaneous_edits_of_one_record_do_not_silently_overwrite_each_other()
    {
        using var setup = _fixture.CreateScope();
        var land = setup.ServiceProvider.GetRequiredService<ILandStructureService>();
        var created = await land.CreateCompanyAsync(new Contracts.Organization.CompanyUpsertDto
        { Code = "RACE", Name = "Race", BaseCurrency = "USD" });

        var attempt = 0;
        var (succeeded, rejected, _) = await RaceAsync(5, async scope =>
        {
            var n = Interlocked.Increment(ref attempt);
            await scope.ServiceProvider.GetRequiredService<ILandStructureService>()
                .UpdateCompanyAsync(created.Id, new Contracts.Organization.CompanyUpsertDto
                {
                    Code = "RACE",
                    Name = $"Renamed by writer {n}",
                    BaseCurrency = "USD",
                    RowVersion = created.RowVersion          // every writer read the same version
                });
        });

        // The row version means exactly one writer may win; the rest must be told to reload.
        Assert.Equal(1, succeeded);
        Assert.Equal(4, rejected);
    }
}
