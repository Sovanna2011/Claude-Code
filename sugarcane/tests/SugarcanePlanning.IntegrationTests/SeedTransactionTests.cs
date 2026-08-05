using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Diagnostics;
using Microsoft.Extensions.DependencyInjection;
using SugarcanePlanning.Application;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;
using SugarcanePlanning.Infrastructure.Persistence;
using SugarcanePlanning.Infrastructure.Persistence.Seed;
using SugarcanePlanning.Infrastructure.Services;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// Executes <see cref="DbSeeder.SeedTransactionsAsync"/> itself over a real service provider.
/// Start-up seeding otherwise runs only against SQL Server, where it cannot be exercised here.
/// </summary>
public class SeedTransactionTests : IDisposable
{
    private readonly ServiceProvider _provider;
    private readonly AppDbContext _db;
    private readonly int _projectionId;

    public SeedTransactionTests()
    {
        var databaseName = $"seed-{Guid.NewGuid():N}";
        var services = new ServiceCollection();

        services.AddLogging();
        services.AddSingleton<ICurrentUser>(new SystemCurrentUser());
        services.AddSingleton<IDateTimeProvider>(new FixedClock(new DateTime(2026, 3, 1, 6, 0, 0, DateTimeKind.Utc)));
        services.Configure<PlanningOptions>(_ => { });

        services.AddDbContext<AppDbContext>(options => options
            .UseInMemoryDatabase(databaseName)
            .ConfigureWarnings(w => w.Ignore(InMemoryEventId.TransactionIgnoredWarning)));
        services.AddScoped<IAppDbContext>(sp => sp.GetRequiredService<AppDbContext>());
        services.AddScoped<IAuditService, AuditService>();
        services.AddApplication();

        _provider = services.BuildServiceProvider();
        _db = _provider.GetRequiredService<AppDbContext>();

        _projectionId = ArrangeApprovedProjection();
    }

    /// <summary>Minimal but complete master data plus one approved projection.</summary>
    private int ArrangeApprovedProjection()
    {
        var company = new Company { Code = "SEED", Name = "Seed Co", BaseCurrency = "USD" };
        _db.Companies.Add(company);
        _db.SaveChanges();

        var estate = new Estate { CompanyId = company.Id, Code = "E1", Name = "Estate", TotalAreaHa = 500m };
        _db.Estates.Add(estate);
        _db.SaveChanges();

        var farm = new Farm { CompanyId = company.Id, EstateId = estate.Id, Code = "F1", Name = "Farm", TotalAreaHa = 300m };
        _db.Farms.Add(farm);
        _db.SaveChanges();

        var zone = new Zone { CompanyId = company.Id, FarmId = farm.Id, Code = "Z1", Name = "Zone", TotalAreaHa = 200m };
        _db.Zones.Add(zone);
        _db.SaveChanges();

        var block = new PlantationBlock
        {
            CompanyId = company.Id, ZoneId = zone.Id, Code = "B1", Name = "Block",
            TotalAreaHa = 120m, PlantableAreaHa = 100m, SoilType = SoilType.Loam
        };
        _db.Blocks.Add(block);

        var season = new GrowingSeason
        {
            CompanyId = company.Id, Code = "S26", Name = "Season 2026",
            StartDate = new DateOnly(2026, 1, 1), EndDate = new DateOnly(2027, 12, 31),
            PlannedPlantingStart = new DateOnly(2026, 3, 1), PlannedPlantingEnd = new DateOnly(2026, 8, 31),
            ExpectedHarvestStart = new DateOnly(2027, 5, 1), ExpectedHarvestEnd = new DateOnly(2027, 11, 30),
            Status = SeasonStatus.Open
        };
        _db.Seasons.Add(season);

        var variety = new CaneVariety
        {
            CompanyId = company.Id, Code = "V1", Name = "Variety",
            SeedRatePerHa = 8m, GrowingPeriodMonths = 12, ExpectedYieldPerHa = 90m, ExpectedLossPercent = 5m
        };
        _db.Varieties.Add(variety);
        _db.SaveChanges();

        var plough = new PlantingActivity
        {
            CompanyId = company.Id, Code = "A003", Name = "First plowing",
            Category = ActivityCategory.LandPreparation, SequenceNo = 3, StandardStartDayOffset = -20,
            StandardCapacityPerDay = 4m, StandardDurationPerHa = 1.8m, StandardLaborDaysPerHa = 0.3m,
            RequiresTractor = true, RequiresEquipment = true,
            DefaultEquipmentCategory = EquipmentCategory.DiscPlow
        };
        var harrow = new PlantingActivity
        {
            CompanyId = company.Id, Code = "A005", Name = "Harrowing",
            Category = ActivityCategory.LandPreparation, SequenceNo = 5, StandardStartDayOffset = -12,
            StandardCapacityPerDay = 7m, StandardDurationPerHa = 1.2m, StandardLaborDaysPerHa = 0.2m,
            RequiresTractor = true, RequiresEquipment = true,
            DefaultEquipmentCategory = EquipmentCategory.Harrow
        };
        var plant = new PlantingActivity
        {
            CompanyId = company.Id, Code = "A012", Name = "Planting",
            Category = ActivityCategory.Planting, SequenceNo = 12, StandardStartDayOffset = 0,
            StandardCapacityPerDay = 4m, StandardDurationPerHa = 2.2m, StandardLaborDaysPerHa = 2.5m,
            RequiresTractor = true, RequiresEquipment = true, RequiresMaterial = true, RequiresLabor = true,
            DefaultEquipmentCategory = EquipmentCategory.CanePlanter
        };
        _db.Activities.AddRange(plough, harrow, plant);
        _db.SaveChanges();

        _db.ActivityDependencies.AddRange(
            new ActivityDependency { CompanyId = company.Id, ActivityId = harrow.Id, PredecessorActivityId = plough.Id, IsBlocking = true },
            new ActivityDependency { CompanyId = company.Id, ActivityId = plant.Id, PredecessorActivityId = harrow.Id, IsBlocking = true });

        _db.Tractors.AddRange(
            new Tractor { CompanyId = company.Id, EstateId = estate.Id, CurrentFarmId = farm.Id, AssetNo = "A1", Code = "TR-01", Horsepower = 180, DailyCapacityHa = 6m, FuelConsumptionPerHa = 24m },
            new Tractor { CompanyId = company.Id, EstateId = estate.Id, CurrentFarmId = farm.Id, AssetNo = "A2", Code = "TR-02", Horsepower = 150, DailyCapacityHa = 5m, FuelConsumptionPerHa = 22m });

        _db.Equipment.AddRange(
            new EquipmentItem { CompanyId = company.Id, EstateId = estate.Id, CurrentFarmId = farm.Id, Code = "EQ-DP", Name = "Disc plough", Category = EquipmentCategory.DiscPlow, MinimumTractorHp = 120, CapacityPerDay = 4.5m },
            new EquipmentItem { CompanyId = company.Id, EstateId = estate.Id, CurrentFarmId = farm.Id, Code = "EQ-HR", Name = "Harrow", Category = EquipmentCategory.Harrow, MinimumTractorHp = 90, CapacityPerDay = 7m },
            new EquipmentItem { CompanyId = company.Id, EstateId = estate.Id, CurrentFarmId = farm.Id, Code = "EQ-CP", Name = "Cane planter", Category = EquipmentCategory.CanePlanter, MinimumTractorHp = 120, CapacityPerDay = 4m });

        _db.Operators.AddRange(
            new Operator { CompanyId = company.Id, FarmId = farm.Id, Code = "OP-1", FullName = "Operator one", Skill = SkillType.TractorOperator },
            new Operator { CompanyId = company.Id, FarmId = farm.Id, Code = "OP-2", FullName = "Operator two", Skill = SkillType.TractorOperator });

        var material = new Material
        {
            CompanyId = company.Id, Code = "MAT-SEED", Name = "Seed cane",
            Category = MaterialCategory.SeedCane, BaseUnit = UnitOfMeasure.Ton, UnitConversionFactor = 1m
        };
        _db.Materials.Add(material);
        _db.SaveChanges();

        _db.MaterialStandards.Add(new ActivityMaterialStandard
        {
            CompanyId = company.Id, ActivityId = plant.Id, MaterialId = material.Id,
            CropType = CropType.NewPlanting, StandardRatePerHa = 8m, NumberOfApplications = 1,
            WastePercent = 5m, EffectiveFrom = new DateOnly(2026, 1, 1)
        });
        _db.SaveChanges();

        var projection = new PlantingProjection
        {
            CompanyId = company.Id,
            EstateId = estate.Id,
            GrowingSeasonId = season.Id,
            ProjectionNo = "PP-S26-0001",
            Version = 1,
            ProjectionDate = new DateOnly(2026, 2, 1),
            PlanningStartDate = new DateOnly(2026, 3, 1),
            PlanningEndDate = new DateOnly(2026, 8, 31),
            Status = ProjectionStatus.Approved,
            IsCurrentVersion = true
        };

        var line = new ProjectionLine
        {
            CompanyId = company.Id,
            FarmId = farm.Id,
            ZoneId = zone.Id,
            BlockId = block.Id,
            CropType = CropType.NewPlanting,
            CaneVarietyId = variety.Id,
            AvailableAreaHa = 100m,
            ProjectedPlantingAreaHa = 80m,
            PlannedPlantingStart = new DateOnly(2026, 3, 2),
            PlannedPlantingEnd = new DateOnly(2026, 3, 20),
            ExpectedYieldPerHa = 90m,
            ExpectedLossPercent = 5m
        };
        line.Recalculate(variety.GrowingPeriodMonths);
        projection.Lines.Add(line);
        projection.RecalculateTotals();

        _db.Projections.Add(projection);
        _db.SaveChanges();

        return projection.Id;
    }

    [Fact]
    public async Task The_seeder_produces_plans_bookings_and_progress()
    {
        var (plans, bookings, actuals) = await DbSeeder.SeedTransactionsAsync(_provider, _projectionId);

        Assert.Equal(3, plans);              // plough, harrow, plant
        Assert.True(bookings > 0, "the seeder should leave at least one live booking");
        Assert.True(actuals > 0, "the seeder should record some progress");

        Assert.Equal(plans, await _db.ActivityPlans.CountAsync(p => p.ProjectionId == _projectionId));
        Assert.Equal(bookings, await _db.Schedules.CountAsync(s => s.Status != ScheduleStatus.Cancelled));
        Assert.Equal(actuals, await _db.ActivityActuals.CountAsync());
    }

    [Fact]
    public async Task The_seeded_bookings_are_free_of_conflicts()
    {
        await DbSeeder.SeedTransactionsAsync(_provider, _projectionId);

        var bookings = await _db.Schedules.Where(s => s.Status != ScheduleStatus.Cancelled).ToListAsync();

        foreach (var a in bookings)
            foreach (var b in bookings.Where(x => x.Id != a.Id))
            {
                var overlaps = a.PlannedStart < b.PlannedEnd && b.PlannedStart < a.PlannedEnd;
                if (!overlaps) continue;

                Assert.False(a.TractorId is not null && a.TractorId == b.TractorId, "a tractor was double-booked");
                Assert.False(a.EquipmentId is not null && a.EquipmentId == b.EquipmentId, "equipment was double-booked");
                Assert.False(a.OperatorId is not null && a.OperatorId == b.OperatorId, "an operator was double-booked");
            }
    }

    [Fact]
    public async Task The_seeded_bookings_respect_the_horsepower_rule()
    {
        await DbSeeder.SeedTransactionsAsync(_provider, _projectionId);

        var bookings = await _db.Schedules
            .Include(s => s.Tractor).Include(s => s.Equipment)
            .Where(s => s.TractorId != null && s.EquipmentId != null)
            .ToListAsync();

        Assert.All(bookings, b => Assert.True(b.Tractor!.Horsepower >= b.Equipment!.MinimumTractorHp));
    }

    [Fact]
    public async Task Seeding_leaves_the_dashboard_with_real_content()
    {
        await DbSeeder.SeedTransactionsAsync(_provider, _projectionId);

        var dashboard = _provider.GetRequiredService<Application.Interfaces.IDashboardService>();
        var view = await dashboard.GetAsync(null, null);

        Assert.Equal(80m, view.TotalProjectedAreaHa);
        Assert.NotEmpty(view.MonthlyTargets);
        Assert.NotEmpty(view.AreaByFarm);
        Assert.True(view.MaterialShortageCount >= 0);
    }

    [Fact]
    public async Task Material_requirements_are_generated_alongside_the_plans()
    {
        await DbSeeder.SeedTransactionsAsync(_provider, _projectionId);

        var requirements = _provider.GetRequiredService<Application.Interfaces.IMaterialRequirementService>();
        var rows = await requirements.GetRequirementsAsync(
            new Contracts.Materials.MaterialRequirementQuery { ProjectionId = _projectionId });

        var seed = Assert.Single(rows);
        Assert.Equal("MAT-SEED", seed.MaterialCode);
        Assert.Equal(672m, seed.TotalRequirement);        // 80 ha x 8 t/ha + 5 % waste
    }

    [Fact]
    public async Task One_activity_is_left_part_finished_so_the_variance_screens_have_content()
    {
        await DbSeeder.SeedTransactionsAsync(_provider, _projectionId);

        var actuals = await _db.ActivityActuals.ToListAsync();

        Assert.Contains(actuals, a => a.CompletionPercent >= 100m);
        Assert.Contains(actuals, a => a.CompletionPercent < 100m && a.DelayReason != null);
    }

    [Fact]
    public async Task Running_the_transaction_seeder_twice_does_not_duplicate_plans()
    {
        await DbSeeder.SeedTransactionsAsync(_provider, _projectionId);
        var firstPass = await _db.ActivityPlans.CountAsync(p => p.ProjectionId == _projectionId);

        // A second pass regenerates rather than appends; live bookings are the only thing that
        // would stop it, and the engine reports that as a business rule rather than duplicating.
        try
        {
            await DbSeeder.SeedTransactionsAsync(_provider, _projectionId);
        }
        catch (Domain.Common.BusinessRuleException ex)
        {
            Assert.Equal("PLANS_SCHEDULED", ex.Code);
        }

        Assert.Equal(firstPass, await _db.ActivityPlans.CountAsync(p => p.ProjectionId == _projectionId));
    }

    public void Dispose()
    {
        _db.Dispose();
        _provider.Dispose();
    }
}
