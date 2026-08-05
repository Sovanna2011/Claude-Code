using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Diagnostics;
using Microsoft.Extensions.Options;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Services;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;
using SugarcanePlanning.Infrastructure.Persistence;
using SugarcanePlanning.Infrastructure.Services;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// Spins up the real application services over an isolated in-memory database and seeds a small
/// but complete data set: one estate, two blocks, a season, a variety, three activities,
/// tractors, implements, an operator and two materials with their standards and stock.
/// </summary>
public sealed class PlanningTestHost : IDisposable
{
    public AppDbContext Db { get; }
    public TestCurrentUser User { get; }
    public FixedClock Clock { get; }
    public PlanningOptions Options { get; }

    public ILandStructureService Land { get; }
    public ISeasonService Seasons { get; }
    public IActivityMasterService ActivityMaster { get; }
    public IMachineryService Machinery { get; }
    public IMaterialMasterService MaterialMaster { get; }
    public IProjectionService Projections { get; }
    public IActivityPlanService ActivityPlans { get; }
    public ISchedulingService Scheduling { get; }
    public IMaterialRequirementService MaterialRequirements { get; }
    public IFuelLaborService FuelLabor { get; }
    public ICapacityService Capacity { get; }
    public IScenarioService Scenarios { get; }
    public IExecutionService Execution { get; }
    public IDashboardService Dashboard { get; }
    public IReportService Reports { get; }
    public IAuditQueryService Audit { get; }
    public ILookupService Lookups { get; }

    // Seeded keys, handy for arranging tests.
    public int CompanyId { get; private set; }
    public int EstateId { get; private set; }
    public int FarmId { get; private set; }
    public int ZoneId { get; private set; }
    public int BlockAId { get; private set; }
    public int BlockBId { get; private set; }
    public int SeasonId { get; private set; }
    public int VarietyId { get; private set; }
    public int PloughActivityId { get; private set; }
    public int PlantActivityId { get; private set; }
    public int FertiliseActivityId { get; private set; }
    public int TractorAId { get; private set; }
    public int TractorBId { get; private set; }
    public int PlanterId { get; private set; }
    public int OperatorId { get; private set; }
    public int SeedMaterialId { get; private set; }
    public int FertiliserMaterialId { get; private set; }

    public PlanningTestHost()
    {
        var options = new DbContextOptionsBuilder<AppDbContext>()
            .UseInMemoryDatabase($"planning-{Guid.NewGuid():N}")
            // The in-memory provider has no transactions; the services open them deliberately.
            .ConfigureWarnings(w => w.Ignore(InMemoryEventId.TransactionIgnoredWarning))
            .Options;

        User = new TestCurrentUser();
        Clock = new FixedClock(new DateTime(2026, 3, 1, 6, 0, 0, DateTimeKind.Utc));
        Options = new PlanningOptions { WorkOnSaturday = true, WorkOnSunday = false };
        var wrapped = Microsoft.Extensions.Options.Options.Create(Options);

        Db = new AppDbContext(options, User);
        var audit = new AuditService(Db, User, Clock);

        Land = new LandStructureService(Db, User, Clock);
        Seasons = new SeasonService(Db, User, Clock);
        ActivityMaster = new ActivityMasterService(Db, User, Clock);
        Machinery = new MachineryService(Db, User, Clock);
        MaterialMaster = new MaterialMasterService(Db, User, Clock, audit);
        Projections = new ProjectionService(Db, User, Clock, audit);
        MaterialRequirements = new MaterialRequirementService(Db, User, Clock, wrapped);
        ActivityPlans = new ActivityPlanService(Db, User, Clock, wrapped, MaterialRequirements);
        Scheduling = new SchedulingService(Db, User, Clock, audit);
        FuelLabor = new FuelLaborService(Db, User, Clock, wrapped);
        Capacity = new CapacityService(Db, User, Clock, wrapped, MaterialRequirements);
        Scenarios = new ScenarioService(Db, User, Clock, Capacity);
        Execution = new ExecutionService(Db, User, Clock, audit);
        Dashboard = new DashboardService(Db, User, Clock, MaterialRequirements, Capacity, wrapped);
        Reports = new ReportService(Db, User, Clock, MaterialRequirements, FuelLabor, Capacity, Projections, Execution);
        Audit = new AuditQueryService(Db, User, Clock);
        Lookups = new LookupService(Db, User, Clock);

        Seed();
    }

    private void Seed()
    {
        var company = new Company { Code = "TST", Name = "Test Sugar", BaseCurrency = "USD" };
        Db.Companies.Add(company);
        Db.SaveChanges();
        CompanyId = company.Id;
        User.CompanyId = company.Id;

        var estate = new Estate { CompanyId = company.Id, Code = "E1", Name = "Test Estate", TotalAreaHa = 1_000m };
        Db.Estates.Add(estate);
        Db.SaveChanges();
        EstateId = estate.Id;

        var farm = new Farm { CompanyId = company.Id, EstateId = estate.Id, Code = "F1", Name = "Test Farm", TotalAreaHa = 500m };
        Db.Farms.Add(farm);
        Db.SaveChanges();
        FarmId = farm.Id;

        var zone = new Zone { CompanyId = company.Id, FarmId = farm.Id, Code = "Z1", Name = "Test Zone", TotalAreaHa = 300m };
        Db.Zones.Add(zone);
        Db.SaveChanges();
        ZoneId = zone.Id;

        var blockA = new PlantationBlock
        {
            CompanyId = company.Id, ZoneId = zone.Id, Code = "B1", Name = "Block one",
            TotalAreaHa = 120m, PlantableAreaHa = 100m, SoilType = SoilType.ClayLoam
        };
        var blockB = new PlantationBlock
        {
            CompanyId = company.Id, ZoneId = zone.Id, Code = "B2", Name = "Block two",
            TotalAreaHa = 60m, PlantableAreaHa = 50m, SoilType = SoilType.Clay
        };
        Db.Blocks.AddRange(blockA, blockB);
        Db.SaveChanges();
        BlockAId = blockA.Id;
        BlockBId = blockB.Id;

        var season = new GrowingSeason
        {
            CompanyId = company.Id,
            Code = "S26",
            Name = "Season 2026",
            StartDate = new DateOnly(2026, 1, 1),
            EndDate = new DateOnly(2027, 12, 31),
            PlannedPlantingStart = new DateOnly(2026, 3, 1),
            PlannedPlantingEnd = new DateOnly(2026, 8, 31),
            ExpectedHarvestStart = new DateOnly(2027, 5, 1),
            ExpectedHarvestEnd = new DateOnly(2027, 11, 30),
            Status = SeasonStatus.Open
        };
        Db.Seasons.Add(season);

        var variety = new CaneVariety
        {
            CompanyId = company.Id, Code = "V1", Name = "Test variety",
            SeedRatePerHa = 8m, GrowingPeriodMonths = 12, ExpectedYieldPerHa = 90m, ExpectedLossPercent = 5m
        };
        Db.Varieties.Add(variety);
        Db.SaveChanges();
        SeasonId = season.Id;
        VarietyId = variety.Id;

        var plough = new PlantingActivity
        {
            CompanyId = company.Id, Code = "A003", Name = "First plowing", Category = ActivityCategory.LandPreparation,
            SequenceNo = 3, StandardStartDayOffset = -20, StandardCapacityPerDay = 4m, StandardDurationPerHa = 1.8m,
            StandardLaborDaysPerHa = 0.3m, RequiresTractor = true, RequiresEquipment = true,
            DefaultEquipmentCategory = EquipmentCategory.DiscPlow, ApplicableCropType = CropType.Both
        };
        var plant = new PlantingActivity
        {
            CompanyId = company.Id, Code = "A012", Name = "Planting", Category = ActivityCategory.Planting,
            SequenceNo = 12, StandardStartDayOffset = 0, StandardCapacityPerDay = 4m, StandardDurationPerHa = 2.2m,
            StandardLaborDaysPerHa = 2.5m, RequiresTractor = true, RequiresEquipment = true, RequiresMaterial = true,
            RequiresLabor = true, DefaultEquipmentCategory = EquipmentCategory.CanePlanter, ApplicableCropType = CropType.Both
        };
        var fertilise = new PlantingActivity
        {
            CompanyId = company.Id, Code = "A013", Name = "Basal fertilizer", Category = ActivityCategory.Fertilization,
            SequenceNo = 13, StandardStartDayOffset = 1, StandardCapacityPerDay = 9m, StandardDurationPerHa = 0.9m,
            StandardLaborDaysPerHa = 0.4m, RequiresTractor = true, RequiresEquipment = true, RequiresMaterial = true,
            RequiresLabor = true, DefaultEquipmentCategory = EquipmentCategory.FertilizerSpreader, ApplicableCropType = CropType.Both
        };
        Db.Activities.AddRange(plough, plant, fertilise);
        Db.SaveChanges();
        PloughActivityId = plough.Id;
        PlantActivityId = plant.Id;
        FertiliseActivityId = fertilise.Id;

        Db.ActivityDependencies.AddRange(
            new ActivityDependency { CompanyId = company.Id, ActivityId = plant.Id, PredecessorActivityId = plough.Id, IsBlocking = true },
            new ActivityDependency { CompanyId = company.Id, ActivityId = fertilise.Id, PredecessorActivityId = plant.Id, IsBlocking = true, LagDays = 1 });

        var tractorA = new Tractor
        {
            CompanyId = company.Id, EstateId = estate.Id, CurrentFarmId = farm.Id, AssetNo = "AST-1", Code = "TR-01",
            Horsepower = 180, DailyCapacityHa = 6m, FuelConsumptionPerHour = 18m, FuelConsumptionPerHa = 24m
        };
        var tractorB = new Tractor
        {
            CompanyId = company.Id, EstateId = estate.Id, CurrentFarmId = farm.Id, AssetNo = "AST-2", Code = "TR-02",
            Horsepower = 90, DailyCapacityHa = 3m, FuelConsumptionPerHour = 9m, FuelConsumptionPerHa = 18m
        };
        Db.Tractors.AddRange(tractorA, tractorB);

        var planter = new EquipmentItem
        {
            CompanyId = company.Id, EstateId = estate.Id, CurrentFarmId = farm.Id, Code = "EQ-CP", Name = "Cane planter",
            Category = EquipmentCategory.CanePlanter, MinimumTractorHp = 120, CapacityPerDay = 4m
        };
        Db.Equipment.Add(planter);

        var op = new Operator
        {
            CompanyId = company.Id, EstateId = estate.Id, FarmId = farm.Id, Code = "OP-1",
            FullName = "Test Operator", Skill = SkillType.TractorOperator
        };
        Db.Operators.Add(op);
        Db.SaveChanges();
        TractorAId = tractorA.Id;
        TractorBId = tractorB.Id;
        PlanterId = planter.Id;
        OperatorId = op.Id;

        var seed = new Material
        {
            CompanyId = company.Id, Code = "MAT-SEED", Name = "Seed cane", Category = MaterialCategory.SeedCane,
            BaseUnit = UnitOfMeasure.Ton, StandardRatePerHa = 8m, UnitConversionFactor = 1m
        };
        var fertiliser = new Material
        {
            CompanyId = company.Id, Code = "MAT-NPK", Name = "NPK basal", Category = MaterialCategory.Fertilizer,
            BaseUnit = UnitOfMeasure.Kilogram, StandardRatePerHa = 350m, UnitConversionFactor = 1m
        };
        Db.Materials.AddRange(seed, fertiliser);
        Db.SaveChanges();
        SeedMaterialId = seed.Id;
        FertiliserMaterialId = fertiliser.Id;

        Db.MaterialStandards.AddRange(
            new ActivityMaterialStandard
            {
                CompanyId = company.Id, ActivityId = plant.Id, MaterialId = seed.Id, CropType = CropType.NewPlanting,
                StandardRatePerHa = 8m, NumberOfApplications = 1, WastePercent = 5m, EffectiveFrom = new DateOnly(2026, 1, 1)
            },
            new ActivityMaterialStandard
            {
                CompanyId = company.Id, ActivityId = fertilise.Id, MaterialId = fertiliser.Id, CropType = CropType.Both,
                StandardRatePerHa = 350m, NumberOfApplications = 1, WastePercent = 2m, EffectiveFrom = new DateOnly(2026, 1, 1)
            });

        Db.MaterialStocks.AddRange(
            new MaterialStock
            {
                CompanyId = company.Id, EstateId = estate.Id, MaterialId = seed.Id,
                AvailableStock = 500m, ReservedQuantity = 0m, IncomingQuantity = 0m, Unit = UnitOfMeasure.Ton
            },
            new MaterialStock
            {
                CompanyId = company.Id, EstateId = estate.Id, MaterialId = fertiliser.Id,
                AvailableStock = 10_000m, ReservedQuantity = 500m, IncomingQuantity = 5_000m, Unit = UnitOfMeasure.Kilogram
            });

        Db.SaveChanges();
    }

    /// <summary>Creates a draft projection with one line on block A.</summary>
    public Contracts.Projections.ProjectionCreateDto NewProjection(decimal areaHa = 80m, int? blockId = null) => new()
    {
        EstateId = EstateId,
        GrowingSeasonId = SeasonId,
        ProjectionDate = Clock.Today,
        PlanningStartDate = new DateOnly(2026, 3, 1),
        PlanningEndDate = new DateOnly(2026, 8, 31),
        Lines =
        {
            new Contracts.Projections.ProjectionLineUpsertDto
            {
                BlockId = blockId ?? BlockAId,
                CaneVarietyId = VarietyId,
                CropType = CropType.NewPlanting,
                ProjectedPlantingAreaHa = areaHa,
                PlannedPlantingStart = new DateOnly(2026, 3, 2),
                PlannedPlantingEnd = new DateOnly(2026, 3, 20),
                ExpectedYieldPerHa = 90m,
                ExpectedLossPercent = 5m
            }
        }
    };

    public void Dispose() => Db.Dispose();
}

/// <summary>Deterministic clock so date-driven behaviour is reproducible.</summary>
public class FixedClock : IDateTimeProvider
{
    public FixedClock(DateTime utcNow) => UtcNow = utcNow;
    public DateTime UtcNow { get; set; }
    public DateOnly Today => DateOnly.FromDateTime(UtcNow);
}

/// <summary>Caller identity for tests; roles can be narrowed to check authorisation paths.</summary>
public class TestCurrentUser : ICurrentUser
{
    public string UserName { get; set; } = "tester";
    public string? FullName => "Integration Tester";
    public int? CompanyId { get; set; }
    public bool IsAuthenticated => true;
    public IReadOnlyCollection<string> Roles { get; set; } = AppRoles.All;
    public string? IpAddress => "127.0.0.1";
    public string? DeviceInfo => "xunit";

    public bool IsInRole(string role) => Roles.Contains(role);

    public bool HasPolicy(string policy)
        => Policies.RoleMap.TryGetValue(policy, out var allowed) && allowed.Any(IsInRole);
}
