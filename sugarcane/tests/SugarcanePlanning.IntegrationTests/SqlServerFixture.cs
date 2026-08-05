using Microsoft.Data.SqlClient;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using SugarcanePlanning.Application;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Domain.Enums;
using SugarcanePlanning.Infrastructure.Persistence;
using SugarcanePlanning.Infrastructure.Persistence.Interceptors;
using SugarcanePlanning.Infrastructure.Services;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// Runs a fact only when a SQL Server is configured, so <c>dotnet test</c> stays green on a
/// machine without one. Point <c>SUGARCANE_TEST_SQLSERVER</c> at a server to enable them, e.g.
/// <c>Server=127.0.0.1,1433;User Id=sa;Password=…;TrustServerCertificate=True</c>.
/// </summary>
public sealed class SqlServerFactAttribute : FactAttribute
{
    public SqlServerFactAttribute()
    {
        if (string.IsNullOrWhiteSpace(SqlServerFixture.BaseConnectionString))
            Skip = "Set SUGARCANE_TEST_SQLSERVER to run the SQL Server tests.";
    }
}

/// <summary>
/// Creates a throw-away database, applies the real EF Core migration to it and exposes the
/// service graph wired to SQL Server. These tests exercise what the in-memory provider cannot:
/// transactions, unique indexes, check constraints and <c>rowversion</c> concurrency.
/// </summary>
public class SqlServerFixture : IAsyncLifetime
{
    public const string EnvironmentVariable = "SUGARCANE_TEST_SQLSERVER";

    public static string? BaseConnectionString =>
        Environment.GetEnvironmentVariable(EnvironmentVariable);

    private readonly string _databaseName = $"SugarcaneTest_{Guid.NewGuid():N}";
    private ServiceProvider? _provider;

    public string ConnectionString { get; private set; } = string.Empty;
    public bool Enabled => !string.IsNullOrWhiteSpace(BaseConnectionString);

    /// <summary>
    /// Master data shared by every test in the class. xUnit builds a new test-class instance per
    /// method, so this is created once here rather than per test — otherwise the second test
    /// would trip the unique index on the company code.
    /// </summary>
    public SqlServerSeed Seed { get; private set; } = null!;

    public IServiceProvider Services => _provider
        ?? throw new InvalidOperationException("The SQL Server fixture is not initialised.");

    /// <summary>A fresh scope, mirroring the per-request lifetime the API uses.</summary>
    public IServiceScope CreateScope() => Services.CreateScope();

    public async Task InitializeAsync()
    {
        if (!Enabled) return;

        var builder = new SqlConnectionStringBuilder(BaseConnectionString) { InitialCatalog = "master" };
        await using (var connection = new SqlConnection(builder.ConnectionString))
        {
            await connection.OpenAsync();
            await using var command = connection.CreateCommand();
            command.CommandText = $"CREATE DATABASE [{_databaseName}]";
            await command.ExecuteNonQueryAsync();
        }

        builder.InitialCatalog = _databaseName;
        ConnectionString = builder.ConnectionString;

        var services = new ServiceCollection();
        services.AddLogging();
        services.AddSingleton<ICurrentUser>(new SystemCurrentUser());
        services.AddSingleton<IDateTimeProvider>(new FixedClock(new DateTime(2026, 3, 1, 6, 0, 0, DateTimeKind.Utc)));
        services.Configure<PlanningOptions>(_ => { });
        services.AddScoped<IAuditService, AuditService>();
        services.AddScoped<AuditSaveChangesInterceptor>();

        services.AddDbContext<AppDbContext>((sp, options) =>
        {
            options.UseSqlServer(ConnectionString, sql =>
                sql.MigrationsHistoryTable("__EFMigrationsHistory", "planning"));
            options.AddInterceptors(sp.GetRequiredService<AuditSaveChangesInterceptor>());
        });
        services.AddScoped<IAppDbContext>(sp => sp.GetRequiredService<AppDbContext>());
        services.AddApplication();

        _provider = services.BuildServiceProvider();

        using (var scope = _provider.CreateScope())
            await scope.ServiceProvider.GetRequiredService<AppDbContext>().Database.MigrateAsync();

        Seed = await ArrangeMasterDataAsync();
    }

    /// <summary>
    /// A block of its own for a test that creates a projection. Sharing one block would trip the
    /// "approved plans for the same block cannot overlap" rule between tests.
    /// </summary>
    public async Task<int> NewBlockAsync(string code)
    {
        using var scope = CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();

        var block = new Domain.Entities.PlantationBlock
        {
            CompanyId = Seed.CompanyId,
            ZoneId = Seed.ZoneId,
            Code = code.Length <= 20 ? code : code[..20],
            Name = $"Block for {code}",
            TotalAreaHa = 120m,
            PlantableAreaHa = 100m,
            SoilType = SoilType.Loam
        };
        db.Blocks.Add(block);
        await db.SaveChangesAsync();
        return block.Id;
    }

    public async Task DisposeAsync()
    {
        if (_provider is not null) await _provider.DisposeAsync();
        if (!Enabled) return;

        var builder = new SqlConnectionStringBuilder(BaseConnectionString) { InitialCatalog = "master" };
        await using var connection = new SqlConnection(builder.ConnectionString);
        await connection.OpenAsync();
        await using var command = connection.CreateCommand();
        command.CommandText = $"ALTER DATABASE [{_databaseName}] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; " +
                              $"DROP DATABASE [{_databaseName}]";
        await command.ExecuteNonQueryAsync();
    }

    private async Task<SqlServerSeed> ArrangeMasterDataAsync()
    {
        using var scope = CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();

        var company = new Domain.Entities.Company { Code = "SQLT", Name = "SQL Test Co", BaseCurrency = "USD" };
        db.Companies.Add(company);
        await db.SaveChangesAsync();

        var estate = new Domain.Entities.Estate { CompanyId = company.Id, Code = "E1", Name = "Estate", TotalAreaHa = 500m };
        db.Estates.Add(estate);
        await db.SaveChangesAsync();

        var farm = new Domain.Entities.Farm { CompanyId = company.Id, EstateId = estate.Id, Code = "F1", Name = "Farm", TotalAreaHa = 300m };
        db.Farms.Add(farm);
        await db.SaveChangesAsync();

        var zone = new Domain.Entities.Zone { CompanyId = company.Id, FarmId = farm.Id, Code = "Z1", Name = "Zone", TotalAreaHa = 200m };
        db.Zones.Add(zone);
        await db.SaveChangesAsync();

        var block = new Domain.Entities.PlantationBlock
        {
            CompanyId = company.Id, ZoneId = zone.Id, Code = "B1", Name = "Block",
            TotalAreaHa = 120m, PlantableAreaHa = 100m, SoilType = SoilType.Loam
        };
        db.Blocks.Add(block);

        var season = new Domain.Entities.GrowingSeason
        {
            CompanyId = company.Id, Code = "S26", Name = "Season 2026",
            StartDate = new DateOnly(2026, 1, 1), EndDate = new DateOnly(2027, 12, 31),
            PlannedPlantingStart = new DateOnly(2026, 3, 1), PlannedPlantingEnd = new DateOnly(2026, 8, 31),
            ExpectedHarvestStart = new DateOnly(2027, 5, 1), ExpectedHarvestEnd = new DateOnly(2027, 11, 30),
            Status = SeasonStatus.Open
        };
        db.Seasons.Add(season);

        var variety = new Domain.Entities.CaneVariety
        {
            CompanyId = company.Id, Code = "V1", Name = "Variety", SeedRatePerHa = 8m,
            GrowingPeriodMonths = 12, ExpectedYieldPerHa = 90m, ExpectedLossPercent = 5m
        };
        db.Varieties.Add(variety);
        await db.SaveChangesAsync();

        var plough = new Domain.Entities.PlantingActivity
        {
            CompanyId = company.Id, Code = "A003", Name = "First plowing",
            Category = ActivityCategory.LandPreparation, SequenceNo = 3, StandardStartDayOffset = -20,
            StandardCapacityPerDay = 4m, StandardDurationPerHa = 1.8m, StandardLaborDaysPerHa = 0.3m,
            RequiresTractor = true, RequiresEquipment = true, DefaultEquipmentCategory = EquipmentCategory.DiscPlow
        };
        var plant = new Domain.Entities.PlantingActivity
        {
            CompanyId = company.Id, Code = "A012", Name = "Planting",
            Category = ActivityCategory.Planting, SequenceNo = 12, StandardStartDayOffset = 0,
            StandardCapacityPerDay = 4m, StandardDurationPerHa = 2.2m, StandardLaborDaysPerHa = 2.5m,
            RequiresTractor = true, RequiresEquipment = true, RequiresMaterial = true, RequiresLabor = true,
            DefaultEquipmentCategory = EquipmentCategory.CanePlanter
        };
        db.Activities.AddRange(plough, plant);
        await db.SaveChangesAsync();

        db.ActivityDependencies.Add(new Domain.Entities.ActivityDependency
        {
            CompanyId = company.Id, ActivityId = plant.Id, PredecessorActivityId = plough.Id, IsBlocking = true
        });

        var tractor = new Domain.Entities.Tractor
        {
            CompanyId = company.Id, EstateId = estate.Id, CurrentFarmId = farm.Id,
            AssetNo = "A1", Code = "TR-01", Horsepower = 180, DailyCapacityHa = 6m, FuelConsumptionPerHa = 24m
        };
        db.Tractors.Add(tractor);

        var material = new Domain.Entities.Material
        {
            CompanyId = company.Id, Code = "MAT-SEED", Name = "Seed cane",
            Category = MaterialCategory.SeedCane, BaseUnit = UnitOfMeasure.Ton, UnitConversionFactor = 1m
        };
        db.Materials.Add(material);
        await db.SaveChangesAsync();

        db.MaterialStandards.Add(new Domain.Entities.ActivityMaterialStandard
        {
            CompanyId = company.Id, ActivityId = plant.Id, MaterialId = material.Id,
            CropType = CropType.NewPlanting, StandardRatePerHa = 8m, NumberOfApplications = 1,
            WastePercent = 5m, EffectiveFrom = new DateOnly(2026, 1, 1)
        });
        await db.SaveChangesAsync();

        return new SqlServerSeed(company.Id, estate.Id, farm.Id, zone.Id, block.Id, season.Id,
            variety.Id, plough.Id, plant.Id, tractor.Id, material.Id);
    }
}

/// <summary>Keys of the master data created by <see cref="SqlServerFixture.ArrangeMasterDataAsync"/>.</summary>
public record SqlServerSeed(int CompanyId, int EstateId, int FarmId, int ZoneId, int BlockId,
    int SeasonId, int VarietyId, int PloughActivityId, int PlantActivityId, int TractorId, int MaterialId);
