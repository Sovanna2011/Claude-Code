using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Contracts.Organization;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;
using SugarcanePlanning.Infrastructure.Persistence;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// The guarantees only a real SQL Server can prove: that the migration applies, that the
/// multi-step operations survive their explicit transactions, and that the unique indexes,
/// check constraints and row-version tokens in the schema actually bite.
///
/// These caught a fatal defect once already — enabling EF Core's retry-on-failure installs an
/// execution strategy that refuses user-initiated transactions, so every transactional
/// operation threw on SQL Server while passing against the in-memory provider.
/// </summary>
public class SqlServerIntegrationTests : IClassFixture<SqlServerFixture>, IAsyncLifetime
{
    private readonly SqlServerFixture _fixture;
    private SqlServerSeed _seed = null!;

    public SqlServerIntegrationTests(SqlServerFixture fixture) => _fixture = fixture;

    public Task InitializeAsync()
    {
        if (_fixture.Enabled) _seed = _fixture.Seed;
        return Task.CompletedTask;
    }

    public Task DisposeAsync() => Task.CompletedTask;

    /// <summary>A projection on <paramref name="blockId"/>, which each test owns exclusively.</summary>
    private ProjectionCreateDto NewProjection(int blockId, decimal areaHa = 80m) => new()
    {
        EstateId = _seed.EstateId,
        GrowingSeasonId = _seed.SeasonId,
        ProjectionDate = new DateOnly(2026, 2, 1),
        PlanningStartDate = new DateOnly(2026, 3, 1),
        PlanningEndDate = new DateOnly(2026, 8, 31),
        Lines =
        {
            new ProjectionLineUpsertDto
            {
                BlockId = blockId,
                CaneVarietyId = _seed.VarietyId,
                CropType = CropType.NewPlanting,
                ProjectedPlantingAreaHa = areaHa,
                PlannedPlantingStart = new DateOnly(2026, 3, 2),
                PlannedPlantingEnd = new DateOnly(2026, 3, 20),
                ExpectedYieldPerHa = 90m,
                ExpectedLossPercent = 5m
            }
        }
    };

    // ------------------------------------------------------------------ schema

    [SqlServerFact]
    public async Task The_migration_creates_the_whole_schema()
    {
        using var scope = _fixture.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();

        Assert.Empty(await db.Database.GetPendingMigrationsAsync());

        var tables = await db.Database
            .SqlQueryRaw<int>("SELECT COUNT(*) AS Value FROM sys.tables WHERE schema_id = SCHEMA_ID('planning')")
            .SingleAsync();
        var foreignKeys = await db.Database
            .SqlQueryRaw<int>("SELECT COUNT(*) AS Value FROM sys.foreign_keys").SingleAsync();
        var checks = await db.Database
            .SqlQueryRaw<int>("SELECT COUNT(*) AS Value FROM sys.check_constraints").SingleAsync();

        Assert.Equal(36, tables);
        Assert.Equal(58, foreignKeys);
        Assert.Equal(32, checks);
    }

    // ------------------------------------------------ transactional operations

    [SqlServerFact]
    public async Task Creating_a_projection_with_lines_commits_its_transaction()
    {
        using var scope = _fixture.CreateScope();
        var projections = scope.ServiceProvider.GetRequiredService<IProjectionService>();
        var blockId = await _fixture.NewBlockAsync("B-CREATE");

        var created = await projections.CreateAsync(NewProjection(blockId));

        Assert.True(created.Id > 0);
        Assert.Single(created.Lines);
        Assert.Equal(80m, created.TotalProjectedAreaHa);
        Assert.Equal(6_840m, created.TotalExpectedProductionTons);

        // Re-read through a fresh scope: the rows must really be committed.
        using var verify = _fixture.CreateScope();
        var reloaded = await verify.ServiceProvider.GetRequiredService<IProjectionService>()
            .GetProjectionAsync(created.Id);
        Assert.Single(reloaded.Lines);
    }

    [SqlServerFact]
    public async Task Generating_an_activity_plan_commits_its_transaction()
    {
        using var scope = _fixture.CreateScope();
        var projections = scope.ServiceProvider.GetRequiredService<IProjectionService>();
        var plans = scope.ServiceProvider.GetRequiredService<IActivityPlanService>();
        var blockId = await _fixture.NewBlockAsync("B-GEN");

        var projection = await projections.CreateAsync(NewProjection(blockId));
        await projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });

        var result = await plans.GenerateAsync(new GenerateActivityPlanRequest { ProjectionId = projection.Id });

        Assert.Equal(2, result.PlansCreated);

        using var verify = _fixture.CreateScope();
        var db = verify.ServiceProvider.GetRequiredService<AppDbContext>();
        Assert.Equal(2, await db.ActivityPlans.CountAsync(p => p.ProjectionId == projection.Id));
        // The material requirements are written by the same call.
        Assert.True(await db.MaterialRequirements.AnyAsync());
    }

    [SqlServerFact]
    public async Task Revising_an_approved_plan_commits_its_transaction()
    {
        using var scope = _fixture.CreateScope();
        var projections = scope.ServiceProvider.GetRequiredService<IProjectionService>();

        var blockId = await _fixture.NewBlockAsync("B-REVISE");
        var projection = await projections.CreateAsync(NewProjection(blockId, areaHa: 60m));
        await projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });

        var revision = await projections.ReviseAsync(projection.Id,
            new ReviseProjectionDto { RevisionReason = "SQL Server round trip" });

        Assert.Equal(2, revision.Version);

        using var verify = _fixture.CreateScope();
        var original = await verify.ServiceProvider.GetRequiredService<IProjectionService>()
            .GetProjectionAsync(projection.Id);
        Assert.True(original.IsReadOnly);
        Assert.False(original.IsCurrentVersion);
    }

    // ------------------------------------------------------- schema guarantees

    [SqlServerFact]
    public async Task The_unique_index_on_a_company_code_is_enforced_by_the_database()
    {
        using var scope = _fixture.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();

        db.Companies.Add(new Company { Code = "DUP", Name = "First", BaseCurrency = "USD" });
        await db.SaveChangesAsync();

        db.Companies.Add(new Company { Code = "DUP", Name = "Second", BaseCurrency = "USD" });
        var ex = await Assert.ThrowsAsync<DbUpdateException>(() => db.SaveChangesAsync());
        Assert.Contains("unique", ex.InnerException?.Message ?? string.Empty, StringComparison.OrdinalIgnoreCase);
    }

    [SqlServerFact]
    public async Task The_check_constraint_on_a_block_area_is_enforced_by_the_database()
    {
        using var scope = _fixture.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();

        // The service layer rejects this too; here we prove the database is the backstop.
        db.Blocks.Add(new PlantationBlock
        {
            CompanyId = _seed.CompanyId,
            ZoneId = _seed.ZoneId,
            Code = "BAD",
            Name = "Plantable exceeds total",
            TotalAreaHa = 50m,
            PlantableAreaHa = 80m
        });

        var ex = await Assert.ThrowsAsync<DbUpdateException>(() => db.SaveChangesAsync());
        Assert.Contains("CK_Block_Plantable", ex.InnerException?.Message ?? string.Empty);
    }

    [SqlServerFact]
    public async Task A_restricting_foreign_key_stops_an_orphan_row()
    {
        using var scope = _fixture.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();

        db.Farms.Add(new Farm { CompanyId = _seed.CompanyId, EstateId = 987_654, Code = "ORP", Name = "Orphan" });

        var ex = await Assert.ThrowsAsync<DbUpdateException>(() => db.SaveChangesAsync());
        Assert.Contains("FOREIGN KEY", ex.InnerException?.Message ?? string.Empty, StringComparison.OrdinalIgnoreCase);
    }

    // ----------------------------------------------------------- concurrency

    [SqlServerFact]
    public async Task A_stale_row_version_is_rejected_as_a_concurrency_conflict()
    {
        using var first = _fixture.CreateScope();
        var land = first.ServiceProvider.GetRequiredService<ILandStructureService>();

        var created = await land.CreateCompanyAsync(new CompanyUpsertDto
        { Code = "CONC", Name = "Concurrency", BaseCurrency = "USD" });
        Assert.NotNull(created.RowVersion);

        // One user saves first …
        using (var winner = _fixture.CreateScope())
        {
            await winner.ServiceProvider.GetRequiredService<ILandStructureService>()
                .UpdateCompanyAsync(created.Id, new CompanyUpsertDto
                {
                    Code = "CONC",
                    Name = "Renamed by the first user",
                    BaseCurrency = "USD",
                    RowVersion = created.RowVersion
                });
        }

        // … the second, holding the row version read before that save, must be refused.
        using var loser = _fixture.CreateScope();
        await Assert.ThrowsAsync<DbUpdateConcurrencyException>(() =>
            loser.ServiceProvider.GetRequiredService<ILandStructureService>()
                .UpdateCompanyAsync(created.Id, new CompanyUpsertDto
                {
                    Code = "CONC",
                    Name = "Renamed by the second user",
                    BaseCurrency = "USD",
                    RowVersion = created.RowVersion
                }));
    }

    [SqlServerFact]
    public async Task The_row_version_changes_on_every_update()
    {
        using var scope = _fixture.CreateScope();
        var land = scope.ServiceProvider.GetRequiredService<ILandStructureService>();

        var created = await land.CreateCompanyAsync(new CompanyUpsertDto
        { Code = "RV", Name = "Row version", BaseCurrency = "USD" });

        var updated = await land.UpdateCompanyAsync(created.Id, new CompanyUpsertDto
        { Code = "RV", Name = "Row version 2", BaseCurrency = "USD", RowVersion = created.RowVersion });

        Assert.NotEqual(created.RowVersion, updated.RowVersion);
    }

    // ------------------------------------------------------- filters and audit

    [SqlServerFact]
    public async Task Soft_deleted_rows_disappear_from_ordinary_queries()
    {
        using var scope = _fixture.CreateScope();
        var land = scope.ServiceProvider.GetRequiredService<ILandStructureService>();

        var created = await land.CreateCompanyAsync(new CompanyUpsertDto
        { Code = "SOFT", Name = "Soft delete", BaseCurrency = "USD" });
        await land.DeleteCompanyAsync(created.Id);

        await Assert.ThrowsAsync<NotFoundException>(() => land.GetCompanyAsync(created.Id));

        using var verify = _fixture.CreateScope();
        var db = verify.ServiceProvider.GetRequiredService<AppDbContext>();
        Assert.True(await db.Companies.IgnoreQueryFilters().AnyAsync(c => c.Id == created.Id));
    }

    [SqlServerFact]
    public async Task The_audit_interceptor_records_changes_against_real_columns()
    {
        using var scope = _fixture.CreateScope();
        var land = scope.ServiceProvider.GetRequiredService<ILandStructureService>();

        var created = await land.CreateCompanyAsync(new CompanyUpsertDto
        { Code = "AUD", Name = "Audited", BaseCurrency = "USD" });
        await land.UpdateCompanyAsync(created.Id, new CompanyUpsertDto
        { Code = "AUD", Name = "Audited twice", BaseCurrency = "USD", RowVersion = created.RowVersion });

        using var verify = _fixture.CreateScope();
        var db = verify.ServiceProvider.GetRequiredService<AppDbContext>();

        var entries = await db.AuditLogs
            .Where(a => a.TableName == "Companies" && a.RecordId == created.Id.ToString())
            .OrderBy(a => a.Id)
            .ToListAsync();

        Assert.Contains(entries, e => e.Action == AuditAction.Create);
        var update = Assert.Single(entries.Where(e => e.Action == AuditAction.Update));
        Assert.Contains("Name", update.ChangedColumns ?? string.Empty);
        Assert.Contains("Audited twice", update.NewValues ?? string.Empty);
    }

    // ------------------------------------------------------ decimal precision

    [SqlServerFact]
    public async Task Decimal_precision_survives_the_round_trip()
    {
        using var scope = _fixture.CreateScope();
        var projections = scope.ServiceProvider.GetRequiredService<IProjectionService>();

        var blockId = await _fixture.NewBlockAsync("B-DECIMAL");
        var request = NewProjection(blockId);
        request.Lines[0].ProjectedPlantingAreaHa = 12.3456m;
        request.Lines[0].ExpectedLossPercent = 7.25m;
        var created = await projections.CreateAsync(request);

        using var verify = _fixture.CreateScope();
        var reloaded = await verify.ServiceProvider.GetRequiredService<IProjectionService>()
            .GetProjectionAsync(created.Id);

        var line = Assert.Single(reloaded.Lines);
        Assert.Equal(12.3456m, line.ProjectedPlantingAreaHa);
        Assert.Equal(11.4505m, line.HarvestableAreaHa);      // 12.3456 x (1 - 7.25 %)
    }

    // --------------------------------------------------------------- reporting

    [SqlServerFact]
    public async Task The_material_requirement_query_translates_to_SQL()
    {
        using var scope = _fixture.CreateScope();
        var projections = scope.ServiceProvider.GetRequiredService<IProjectionService>();
        var plans = scope.ServiceProvider.GetRequiredService<IActivityPlanService>();
        var requirements = scope.ServiceProvider.GetRequiredService<IMaterialRequirementService>();
        var blockId = await _fixture.NewBlockAsync("B-MRP");

        var projection = await projections.CreateAsync(NewProjection(blockId));
        await projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });
        await plans.GenerateAsync(new GenerateActivityPlanRequest { ProjectionId = projection.Id });

        var rows = await requirements.GetRequirementsAsync(new MaterialRequirementQuery
        {
            ProjectionId = projection.Id,
            GroupBy = "material"
        });

        var seed = Assert.Single(rows);
        Assert.Equal("MAT-SEED", seed.MaterialCode);
        Assert.Equal(672m, seed.TotalRequirement);           // 80 ha x 8 t/ha + 5 % waste
    }
}
