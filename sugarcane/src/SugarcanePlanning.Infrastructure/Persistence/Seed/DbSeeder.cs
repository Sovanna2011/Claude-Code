using Microsoft.AspNetCore.Identity;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Execution;
using SugarcanePlanning.Contracts.Scheduling;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;
using SugarcanePlanning.Infrastructure.Identity;

namespace SugarcanePlanning.Infrastructure.Persistence.Seed;

/// <summary>
/// Creates the roles, the demo users and a complete set of sample master and transaction data
/// (section 25.15). Every step is idempotent, so it is safe to run on each start-up.
/// </summary>
public static class DbSeeder
{
    public const string DemoPassword = "Planner#2026";

    public static async Task SeedAsync(IServiceProvider services, CancellationToken ct = default)
    {
        var db = services.GetRequiredService<AppDbContext>();
        var logger = services.GetService<ILoggerFactory>()?.CreateLogger("DbSeeder");

        await SeedRolesAsync(services);

        if (await db.Companies.IgnoreQueryFilters().AnyAsync(ct))
        {
            logger?.LogInformation("Sample data already present; seeding skipped.");
            await SeedUsersAsync(services, await db.Companies.IgnoreQueryFilters().Select(c => c.Id).FirstAsync(ct));
            return;
        }

        var company = new Company
        {
            Code = "SGC",
            Name = "Sugarcane Growers Corporation",
            Address = "1 Mill Road, Cane Valley",
            BaseCurrency = "USD"
        };
        db.Companies.Add(company);
        await db.SaveChangesAsync(ct);

        // ------------------------------------------------------------ land structure

        var estate = new Estate
        {
            CompanyId = company.Id,
            Code = "EST-01",
            Name = "Northern Estate",
            Location = "Cane Valley, North District",
            TotalAreaHa = 12_000m
        };
        db.Estates.Add(estate);
        await db.SaveChangesAsync(ct);

        var farms = new List<Farm>
        {
            new() { CompanyId = company.Id, EstateId = estate.Id, Code = "FRM-01", Name = "Riverside Farm", ManagerName = "J. Almeida", TotalAreaHa = 4_200m },
            new() { CompanyId = company.Id, EstateId = estate.Id, Code = "FRM-02", Name = "Highland Farm", ManagerName = "P. Okonkwo", TotalAreaHa = 3_800m },
            new() { CompanyId = company.Id, EstateId = estate.Id, Code = "FRM-03", Name = "Valley Farm", ManagerName = "S. Ramirez", TotalAreaHa = 4_000m }
        };
        db.Farms.AddRange(farms);
        await db.SaveChangesAsync(ct);

        var zones = new List<Zone>();
        foreach (var farm in farms)
            for (var z = 1; z <= 2; z++)
                zones.Add(new Zone
                {
                    CompanyId = company.Id,
                    FarmId = farm.Id,
                    Code = $"{farm.Code}-Z{z}",
                    Name = $"{farm.Name} Zone {z}",
                    SupervisorName = z == 1 ? "M. Diallo" : "T. Nakamura",
                    TotalAreaHa = farm.TotalAreaHa / 2m
                });
        db.Zones.AddRange(zones);
        await db.SaveChangesAsync(ct);

        var soils = new[] { SoilType.ClayLoam, SoilType.SandyLoam, SoilType.Loam, SoilType.Clay };

        // Each farm is a cluster of fields near Lusaka, its second zone lying south-east of the
        // first and the four blocks of a zone forming a 2 x 2 grid. Deriving the coordinates from
        // the block number alone put the whole estate on one straight diagonal, which is fine for
        // arithmetic but nonsense on a map — the location screen is now the thing that reads them.
        var farmCentres = new Dictionary<string, (decimal Lat, decimal Lng)>
        {
            ["FRM-01"] = (-15.474m, 28.232m),
            ["FRM-02"] = (-15.398m, 28.318m),
            ["FRM-03"] = (-15.322m, 28.404m)
        };
        const decimal blockStep = 0.0115m;      // ≈ 1.3 km between block centres
        var zoneOffsets = new[] { (Lat: 0m, Lng: 0m), (Lat: -0.021m, Lng: 0.028m) };

        var blocks = new List<PlantationBlock>();
        var blockNo = 1;
        foreach (var zone in zones)
        {
            var farm = farms.First(f => f.Id == zone.FarmId);
            var centre = farmCentres.TryGetValue(farm.Code, out var c) ? c : (Lat: -15.4m, Lng: 28.3m);
            var offset = zoneOffsets[(int.Parse(zone.Code[^1..]) - 1) % zoneOffsets.Length];

            for (var i = 1; i <= 4; i++)
            {
                var row = (i - 1) / 2;
                var col = (i - 1) % 2;
                var total = 150m + i * 25m;
                blocks.Add(new PlantationBlock
                {
                    CompanyId = company.Id,
                    ZoneId = zone.Id,
                    Code = $"BLK-{blockNo:D3}",
                    Name = $"{zone.Name} Block {i}",
                    TotalAreaHa = total,
                    PlantableAreaHa = Math.Round(total * 0.92m, 4),
                    SoilType = soils[(blockNo - 1) % soils.Length],
                    LandCondition = blockNo % 3 == 0 ? LandCondition.Undulating : LandCondition.Flat,
                    Irrigation = blockNo % 4 == 0 ? IrrigationType.Drip : IrrigationType.Furrow,
                    CurrentCropStatus = CropStatus.Fallow,
                    Latitude = Math.Round(centre.Lat + offset.Lat - row * blockStep, 6),
                    Longitude = Math.Round(centre.Lng + offset.Lng + col * blockStep, 6),
                    MapReference = $"SHEET-{blockNo:D3}"
                });
                blockNo++;
            }
        }
        db.Blocks.AddRange(blocks);
        await db.SaveChangesAsync(ct);

        // ------------------------------------------------------ season and varieties

        var season = new GrowingSeason
        {
            CompanyId = company.Id,
            Code = "S2026",
            Name = "Season 2026 / 2027",
            StartDate = new DateOnly(2026, 1, 1),
            EndDate = new DateOnly(2027, 12, 31),
            PlannedPlantingStart = new DateOnly(2026, 3, 1),
            PlannedPlantingEnd = new DateOnly(2026, 8, 31),
            ExpectedHarvestStart = new DateOnly(2027, 5, 1),
            ExpectedHarvestEnd = new DateOnly(2027, 11, 30),
            Status = SeasonStatus.Open,
            Remarks = "Main planting campaign for the northern estate."
        };
        db.Seasons.Add(season);

        var varieties = new List<CaneVariety>
        {
            new() { CompanyId = company.Id, Code = "CO-0238", Name = "CO 0238", RecommendedSoilType = SoilType.ClayLoam, SeedRatePerHa = 8.0m, GrowingPeriodMonths = 12, ExpectedYieldPerHa = 92m, ExpectedLossPercent = 5m, RecommendedPlantingStartMonth = 3, RecommendedPlantingEndMonth = 7 },
            new() { CompanyId = company.Id, Code = "NCO-376", Name = "NCo 376", RecommendedSoilType = SoilType.SandyLoam, SeedRatePerHa = 7.5m, GrowingPeriodMonths = 13, ExpectedYieldPerHa = 85m, ExpectedLossPercent = 6m, RecommendedPlantingStartMonth = 4, RecommendedPlantingEndMonth = 8 },
            new() { CompanyId = company.Id, Code = "R-570", Name = "R 570", RecommendedSoilType = SoilType.Loam, SeedRatePerHa = 8.5m, GrowingPeriodMonths = 12, ExpectedYieldPerHa = 88m, ExpectedLossPercent = 4.5m, RecommendedPlantingStartMonth = 3, RecommendedPlantingEndMonth = 6 }
        };
        db.Varieties.AddRange(varieties);
        await db.SaveChangesAsync(ct);

        // --------------------------------------------------------- activity master

        var activities = BuildActivities(company.Id);
        db.Activities.AddRange(activities);
        await db.SaveChangesAsync(ct);

        // Each activity depends on the previous one in sequence, which is how field work runs.
        var ordered = activities.OrderBy(a => a.SequenceNo).ToList();
        for (var i = 1; i < ordered.Count; i++)
            db.ActivityDependencies.Add(new ActivityDependency
            {
                CompanyId = company.Id,
                ActivityId = ordered[i].Id,
                PredecessorActivityId = ordered[i - 1].Id,
                LagDays = ordered[i].Code == "A012" ? 1 : 0,
                IsBlocking = ordered[i].IsMandatory
            });
        await db.SaveChangesAsync(ct);

        // -------------------------------------------------------------- machinery

        var tractors = new List<Tractor>();
        for (var i = 1; i <= 10; i++)
        {
            var hp = i <= 4 ? 180 : i <= 8 ? 120 : 90;
            tractors.Add(new Tractor
            {
                CompanyId = company.Id,
                EstateId = estate.Id,
                AssetNo = $"AST-TR-{i:D3}",
                Code = $"TR-{i:D2}",
                RegistrationNo = $"CV-{4000 + i}",
                Brand = i % 2 == 0 ? "John Deere" : "New Holland",
                Model = hp >= 180 ? "6M-180" : hp >= 120 ? "T6-120" : "T4-90",
                Horsepower = hp,
                CurrentFarmId = farms[(i - 1) % farms.Count].Id,
                Ownership = i > 8 ? OwnershipType.Rental : OwnershipType.Company,
                DailyCapacityHa = hp >= 180 ? 6.0m : hp >= 120 ? 4.0m : 3.0m,
                FuelConsumptionPerHour = hp >= 180 ? 18m : hp >= 120 ? 13m : 9m,
                FuelConsumptionPerHa = hp >= 180 ? 24m : hp >= 120 ? 21m : 18m,
                Availability = i == 9 ? AvailabilityStatus.UnderMaintenance : AvailabilityStatus.Available,
                Maintenance = i == 9 ? MaintenanceStatus.InService : MaintenanceStatus.Serviceable,
                MaintenanceFromDate = i == 9 ? new DateOnly(2026, 3, 1) : null,
                MaintenanceToDate = i == 9 ? new DateOnly(2026, 3, 20) : null
            });
        }
        db.Tractors.AddRange(tractors);

        var equipmentSpecs = new (string Code, string Name, EquipmentCategory Category, int MinHp, decimal PerHour, decimal PerDay)[]
        {
            ("EQ-DP-01", "Disc plough 28\"", EquipmentCategory.DiscPlow, 120, 0.6m, 4.5m),
            ("EQ-DP-02", "Disc plough 28\"", EquipmentCategory.DiscPlow, 120, 0.6m, 4.5m),
            ("EQ-MP-01", "Mouldboard plough", EquipmentCategory.MoldboardPlow, 150, 0.5m, 4.0m),
            ("EQ-HR-01", "Offset harrow", EquipmentCategory.Harrow, 90, 0.9m, 7.0m),
            ("EQ-HR-02", "Offset harrow", EquipmentCategory.Harrow, 90, 0.9m, 7.0m),
            ("EQ-LL-01", "Laser land leveler", EquipmentCategory.LandLeveler, 150, 0.5m, 4.0m),
            ("EQ-FO-01", "Furrow opener 3-row", EquipmentCategory.FurrowOpener, 90, 0.8m, 6.0m),
            ("EQ-CP-01", "Cane planter 2-row", EquipmentCategory.CanePlanter, 120, 0.5m, 4.0m),
            ("EQ-CP-02", "Cane planter 2-row", EquipmentCategory.CanePlanter, 120, 0.5m, 4.0m),
            ("EQ-FS-01", "Fertilizer spreader", EquipmentCategory.FertilizerSpreader, 90, 1.2m, 9.0m),
            ("EQ-SP-01", "Boom sprayer 600 L", EquipmentCategory.ChemicalSprayer, 90, 1.5m, 11.0m),
            ("EQ-WT-01", "Water truck 8,000 L", EquipmentCategory.WaterTruck, 0, 1.0m, 8.0m),
            ("EQ-SC-01", "Seed cane cutter", EquipmentCategory.SeedCutter, 90, 1.0m, 8.0m),
            ("EQ-TR-01", "Tipping trailer 6 t", EquipmentCategory.Trailer, 90, 1.5m, 12.0m)
        };
        var equipment = equipmentSpecs.Select(spec => new EquipmentItem
        {
            CompanyId = company.Id,
            EstateId = estate.Id,
            Code = spec.Code,
            Name = spec.Name,
            Category = spec.Category,
            MinimumTractorHp = spec.MinHp,
            CapacityPerHour = spec.PerHour,
            CapacityPerDay = spec.PerDay,
            CurrentFarmId = farms[0].Id,
            Ownership = OwnershipType.Company
        }).ToList();
        db.Equipment.AddRange(equipment);
        await db.SaveChangesAsync(ct);

        // Compatibility: pair every implement with each tractor that has enough horsepower.
        foreach (var item in equipment)
            foreach (var tractor in tractors.Where(t => t.Horsepower >= item.MinimumTractorHp))
                db.Compatibilities.Add(new TractorEquipmentCompatibility
                {
                    CompanyId = company.Id,
                    TractorId = tractor.Id,
                    EquipmentId = item.Id,
                    IsRecommended = tractor.Horsepower >= item.MinimumTractorHp + 30
                });

        // ------------------------------------------------------------- workforce

        var teams = new List<WorkTeam>
        {
            new() { CompanyId = company.Id, FarmId = farms[0].Id, Code = "TM-01", Name = "Planting crew A", SupervisorName = "L. Fernandes", PrimarySkill = SkillType.Planter, MemberCount = 24 },
            new() { CompanyId = company.Id, FarmId = farms[1].Id, Code = "TM-02", Name = "Planting crew B", SupervisorName = "K. Mensah", PrimarySkill = SkillType.Planter, MemberCount = 22 },
            new() { CompanyId = company.Id, FarmId = farms[2].Id, Code = "TM-03", Name = "Spray crew", SupervisorName = "A. Haddad", PrimarySkill = SkillType.SprayerOperator, MemberCount = 12 }
        };
        db.WorkTeams.AddRange(teams);
        await db.SaveChangesAsync(ct);

        var operatorNames = new[]
        {
            "D. Mwangi", "R. Silva", "H. Patel", "N. Costa", "B. Traoré",
            "V. Kumar", "E. Santos", "O. Adeyemi", "C. Lin", "F. Rossi"
        };
        for (var i = 0; i < operatorNames.Length; i++)
            db.Operators.Add(new Operator
            {
                CompanyId = company.Id,
                EstateId = estate.Id,
                FarmId = farms[i % farms.Count].Id,
                Code = $"OP-{i + 1:D3}",
                FullName = operatorNames[i],
                Skill = i < 6 ? SkillType.TractorOperator : SkillType.EquipmentOperator,
                LicenseNo = $"LIC-{9000 + i}",
                LicenseExpiry = new DateOnly(2028, 12, 31),
                WorkTeamId = teams[i % teams.Count].Id,
                StandardHoursPerDay = 8m
            });
        await db.SaveChangesAsync(ct);

        // -------------------------------------------------------------- materials

        var materials = new List<Material>
        {
            new() { CompanyId = company.Id, Code = "MAT-SEED", Name = "Seed cane", Category = MaterialCategory.SeedCane, BaseUnit = UnitOfMeasure.Ton, StandardRatePerHa = 8m, MinApplicationRate = 6m, MaxApplicationRate = 12m, UnitConversionFactor = 1m },
            new() { CompanyId = company.Id, Code = "MAT-UREA", Name = "Urea 46-0-0", Category = MaterialCategory.Fertilizer, BaseUnit = UnitOfMeasure.Kilogram, AlternativeUnit = UnitOfMeasure.Bag, UnitConversionFactor = 50m, StandardRatePerHa = 250m, MinApplicationRate = 120m, MaxApplicationRate = 400m },
            new() { CompanyId = company.Id, Code = "MAT-NPK", Name = "NPK 12-24-12 basal", Category = MaterialCategory.Fertilizer, BaseUnit = UnitOfMeasure.Kilogram, AlternativeUnit = UnitOfMeasure.Bag, UnitConversionFactor = 50m, StandardRatePerHa = 350m, MinApplicationRate = 200m, MaxApplicationRate = 500m },
            new() { CompanyId = company.Id, Code = "MAT-ATRA", Name = "Atrazine 50 SC", Category = MaterialCategory.Herbicide, BaseUnit = UnitOfMeasure.Liter, AlternativeUnit = UnitOfMeasure.Drum, UnitConversionFactor = 200m, StandardRatePerHa = 4m, MinApplicationRate = 2m, MaxApplicationRate = 6m },
            new() { CompanyId = company.Id, Code = "MAT-24D", Name = "2,4-D amine", Category = MaterialCategory.Herbicide, BaseUnit = UnitOfMeasure.Liter, StandardRatePerHa = 2.5m, MinApplicationRate = 1m, MaxApplicationRate = 4m, UnitConversionFactor = 1m },
            new() { CompanyId = company.Id, Code = "MAT-FUNG", Name = "Seed treatment fungicide", Category = MaterialCategory.Pesticide, BaseUnit = UnitOfMeasure.Liter, StandardRatePerHa = 1.2m, MinApplicationRate = 0.5m, MaxApplicationRate = 2m, UnitConversionFactor = 1m },
            new() { CompanyId = company.Id, Code = "MAT-DIES", Name = "Diesel", Category = MaterialCategory.Fuel, BaseUnit = UnitOfMeasure.Liter, AlternativeUnit = UnitOfMeasure.Drum, UnitConversionFactor = 200m, StandardRatePerHa = 45m, MinApplicationRate = 0m, MaxApplicationRate = 0m },
            new() { CompanyId = company.Id, Code = "MAT-WATR", Name = "Irrigation water", Category = MaterialCategory.Water, BaseUnit = UnitOfMeasure.Liter, StandardRatePerHa = 60_000m, MinApplicationRate = 0m, MaxApplicationRate = 0m, UnitConversionFactor = 1m }
        };
        db.Materials.AddRange(materials);
        await db.SaveChangesAsync(ct);

        Material M(string code) => materials.First(m => m.Code == code);
        PlantingActivity A(string code) => activities.First(a => a.Code == code);

        var standards = new List<ActivityMaterialStandard>
        {
            new() { CompanyId = company.Id, ActivityId = A("A011").Id, MaterialId = M("MAT-FUNG").Id, CropType = CropType.NewPlanting, StandardRatePerHa = 1.2m, NumberOfApplications = 1, WastePercent = 3m, EffectiveFrom = new DateOnly(2026, 1, 1) },
            new() { CompanyId = company.Id, ActivityId = A("A012").Id, MaterialId = M("MAT-SEED").Id, CropType = CropType.NewPlanting, StandardRatePerHa = 8m, NumberOfApplications = 1, WastePercent = 5m, EffectiveFrom = new DateOnly(2026, 1, 1) },
            new() { CompanyId = company.Id, ActivityId = A("A013").Id, MaterialId = M("MAT-NPK").Id, CropType = CropType.Both, StandardRatePerHa = 350m, NumberOfApplications = 1, WastePercent = 2m, EffectiveFrom = new DateOnly(2026, 1, 1) },
            new() { CompanyId = company.Id, ActivityId = A("A014").Id, MaterialId = M("MAT-ATRA").Id, CropType = CropType.Both, StandardRatePerHa = 4m, NumberOfApplications = 1, WastePercent = 4m, EffectiveFrom = new DateOnly(2026, 1, 1) },
            new() { CompanyId = company.Id, ActivityId = A("A015").Id, MaterialId = M("MAT-WATR").Id, CropType = CropType.Both, StandardRatePerHa = 60_000m, NumberOfApplications = 2, WastePercent = 8m, EffectiveFrom = new DateOnly(2026, 1, 1) },
            new() { CompanyId = company.Id, ActivityId = A("A018").Id, MaterialId = M("MAT-UREA").Id, CropType = CropType.Both, StandardRatePerHa = 250m, NumberOfApplications = 1, WastePercent = 2m, EffectiveFrom = new DateOnly(2026, 1, 1) },
            new() { CompanyId = company.Id, ActivityId = A("A019").Id, MaterialId = M("MAT-24D").Id, CropType = CropType.Both, StandardRatePerHa = 2.5m, NumberOfApplications = 2, WastePercent = 4m, EffectiveFrom = new DateOnly(2026, 1, 1) },
            // Heavier basal dose on clay: a more specific standard that wins over the generic row.
            new() { CompanyId = company.Id, ActivityId = A("A013").Id, MaterialId = M("MAT-NPK").Id, CropType = CropType.Both, SoilType = SoilType.Clay, StandardRatePerHa = 400m, NumberOfApplications = 1, WastePercent = 2m, EffectiveFrom = new DateOnly(2026, 1, 1) }
        };
        db.MaterialStandards.AddRange(standards);

        var stockLevels = new (string Code, decimal Available, decimal Reserved, decimal Incoming)[]
        {
            ("MAT-SEED", 9_000m, 500m, 2_000m),
            ("MAT-UREA", 180_000m, 20_000m, 60_000m),
            ("MAT-NPK", 240_000m, 15_000m, 40_000m),
            ("MAT-ATRA", 3_000m, 200m, 1_000m),
            ("MAT-24D", 1_400m, 100m, 600m),
            ("MAT-FUNG", 900m, 50m, 200m),
            ("MAT-DIES", 120_000m, 5_000m, 40_000m),
            ("MAT-WATR", 90_000_000m, 0m, 0m)
        };
        foreach (var (code, available, reserved, incoming) in stockLevels)
        {
            var material = M(code);
            db.MaterialStocks.Add(new MaterialStock
            {
                CompanyId = company.Id,
                EstateId = estate.Id,
                MaterialId = material.Id,
                AvailableStock = available,
                ReservedQuantity = reserved,
                IncomingQuantity = incoming,
                IncomingExpectedDate = new DateOnly(2026, 3, 15),
                Unit = material.BaseUnit,
                LastSyncedUtc = DateTime.UtcNow,
                SourceSystem = "seed"
            });
        }
        await db.SaveChangesAsync(ct);

        // ------------------------------------------------- sample planting projection

        var projection = new PlantingProjection
        {
            CompanyId = company.Id,
            EstateId = estate.Id,
            GrowingSeasonId = season.Id,
            ProjectionNo = "PP-S2026-0001",
            Version = 1,
            ProjectionDate = new DateOnly(2026, 2, 1),
            PlanningStartDate = season.PlannedPlantingStart,
            PlanningEndDate = season.PlannedPlantingEnd,
            Status = ProjectionStatus.Approved,
            PreparedBy = "planner",
            SubmittedBy = "planner",
            SubmittedAtUtc = new DateTime(2026, 2, 3, 9, 0, 0, DateTimeKind.Utc),
            ApprovedBy = "director",
            ApprovedAtUtc = new DateTime(2026, 2, 5, 14, 30, 0, DateTimeKind.Utc),
            IsCurrentVersion = true,
            Remarks = "Baseline plan for the 2026 campaign."
        };

        var zoneById = zones.ToDictionary(z => z.Id);
        var startDate = new DateOnly(2026, 3, 2);

        for (var i = 0; i < 12; i++)
        {
            var block = blocks[i];
            var variety = varieties[i % varieties.Count];
            var zone = zoneById[block.ZoneId];
            var cropType = i % 3 == 2 ? CropType.Ratoon : CropType.NewPlanting;
            var area = Math.Round(block.PlantableAreaHa * 0.8m, 4);

            var line = new ProjectionLine
            {
                CompanyId = company.Id,
                FarmId = zone.FarmId,
                ZoneId = zone.Id,
                BlockId = block.Id,
                CropType = cropType,
                CaneVarietyId = variety.Id,
                AvailableAreaHa = block.PlantableAreaHa,
                ProjectedPlantingAreaHa = area,
                PlannedPlantingStart = startDate.AddDays(i * 7),
                PlannedPlantingEnd = startDate.AddDays(i * 7 + 12),
                ExpectedYieldPerHa = variety.ExpectedYieldPerHa,
                ExpectedLossPercent = variety.ExpectedLossPercent,
                Priority = (i % 5) + 1,
                Remarks = cropType == CropType.Ratoon ? "First ratoon" : null
            };
            line.Recalculate(variety.GrowingPeriodMonths);
            projection.Lines.Add(line);
        }

        projection.RecalculateTotals();
        projection.ApprovalHistory.Add(new ProjectionApprovalHistory
        {
            CompanyId = company.Id,
            Action = ApprovalAction.Submit,
            FromStatus = ProjectionStatus.Draft,
            ToStatus = ProjectionStatus.Submitted,
            ActionBy = "planner",
            ActionAtUtc = projection.SubmittedAtUtc!.Value,
            Comments = "Submitted for management approval."
        });
        projection.ApprovalHistory.Add(new ProjectionApprovalHistory
        {
            CompanyId = company.Id,
            Action = ApprovalAction.Approve,
            FromStatus = ProjectionStatus.Submitted,
            ToStatus = ProjectionStatus.Approved,
            ActionBy = "director",
            ActionAtUtc = projection.ApprovedAtUtc!.Value,
            Comments = "Approved. Proceed with resource scheduling."
        });

        db.Projections.Add(projection);
        await db.SaveChangesAsync(ct);

        var transactions = await SeedTransactionsAsync(services, projection.Id, ct);

        await SeedUsersAsync(services, company.Id);
        logger?.LogInformation(
            "Seeded {Blocks} blocks, {Activities} activities, projection {No}, " +
            "{Plans} activity plans, {Bookings} bookings and {Actuals} progress records.",
            blocks.Count, activities.Count, projection.ProjectionNo,
            transactions.Plans, transactions.Bookings, transactions.Actuals);
    }

    /// <summary>
    /// Sample transaction data (section 25.15): runs the real engines so the demo opens with a
    /// populated Gantt, live bookings, material requirements and part-recorded progress.
    /// Returns the row counts for the start-up log.
    /// </summary>
    public static async Task<(int Plans, int Bookings, int Actuals)> SeedTransactionsAsync(
        IServiceProvider services, int projectionId, CancellationToken ct = default)
    {
        var db = services.GetRequiredService<AppDbContext>();
        var planning = services.GetRequiredService<IActivityPlanService>();
        var scheduling = services.GetRequiredService<ISchedulingService>();
        var execution = services.GetRequiredService<IExecutionService>();

        // 1 — activity plans and, through them, the material requirements.
        var generated = await planning.GenerateAsync(
            new GenerateActivityPlanRequest { ProjectionId = projectionId, Regenerate = true }, ct);

        var plans = await db.ActivityPlans
            .Include(p => p.Activity)
            .Where(p => p.ProjectionId == projectionId)
            .OrderBy(p => p.PlannedStartDate).ThenBy(p => p.SequenceNo)
            .ToListAsync(ct);

        var tractors = await db.Tractors.Where(t => t.Availability == AvailabilityStatus.Available)
            .OrderBy(t => t.Code).ToListAsync(ct);
        var equipment = await db.Equipment.OrderBy(e => e.Code).ToListAsync(ct);
        var operators = await db.Operators.OrderBy(o => o.Code).ToListAsync(ct);

        // 2 — book the first working day of the earliest plans, rotating the fleet so no
        //     machine is ever double-booked. Anything the engine rejects is simply skipped.
        var bookings = 0;
        var index = 0;
        foreach (var plan in plans.Where(p => p.Activity?.RequiresTractor == true).Take(12))
        {
            var tractor = tractors.Count == 0 ? null : tractors[index % tractors.Count];
            var implement = plan.RequiredEquipmentCategory is null
                ? null
                : equipment.FirstOrDefault(e => e.Category == plan.RequiredEquipmentCategory
                                                && (tractor is null || tractor.Horsepower >= e.MinimumTractorHp));
            var op = operators.Count == 0 ? null : operators[index % operators.Count];
            var start = plan.PlannedStartDate.ToDateTime(new TimeOnly(7, 0));

            try
            {
                await scheduling.CreateAsync(new ResourceScheduleUpsertDto
                {
                    ActivityPlanId = plan.Id,
                    ScheduleDate = plan.PlannedStartDate,
                    PlannedStart = start,
                    PlannedEnd = start.AddHours(8),
                    TractorId = tractor?.Id,
                    EquipmentId = implement?.Id,
                    OperatorId = op?.Id,
                    PlannedAreaHa = plan.DailyTargetHa,
                    ExpectedWorkingHours = 8m,
                    SupervisorName = "M. Diallo",
                    Status = ScheduleStatus.Confirmed
                }, ct);
                bookings++;
            }
            catch (BusinessRuleException)
            {
                // A conflicting sample booking is not worth failing start-up over.
            }
            index++;
        }

        // 3 — record progress: the earliest land preparation, and the planting that follows it.
        //     Planting matters because that is what the dashboard measures — it reports area
        //     planted, not land cleared. Seeding land preparation alone leaves the headline
        //     figures, the monthly target-versus-actual table and every variance screen sitting
        //     at zero, which makes the sample tenant look like nothing has happened.
        //     One plan of each kind is left deliberately short so the variance, delay and
        //     shortage screens have content to show.
        var actuals = 0;
        var landPrep = plans
            .Where(p => p.Activity?.Category == ActivityCategory.LandPreparation)
            .Take(6)
            .ToList();
        var planting = plans
            .Where(p => p.Activity?.Category == ActivityCategory.Planting)
            .OrderBy(p => p.PlannedStartDate)
            .Take(5)
            .ToList();

        var completed = landPrep.Concat(planting).ToList();

        for (var i = 0; i < completed.Count; i++)
        {
            var plan = completed[i];
            var isShort = i == landPrep.Count - 1 || i == completed.Count - 1;

            await execution.RecordAsync(new ActivityActualUpsertDto
            {
                ActivityPlanId = plan.Id,
                ActualStartDate = plan.PlannedStartDate,
                ActualCompletionDate = isShort ? null : plan.PlannedEndDate,
                ActualCompletedAreaHa = isShort
                    ? Math.Round(plan.PlannedAreaHa * 0.45m, 4)
                    : plan.PlannedAreaHa,
                ActualWorkingHours = plan.PlannedWorkingHours,
                ActualFuelLiters = Math.Round(plan.PlannedFuelLiters * (isShort ? 0.5m : 1.06m), 4),
                ActualLaborDays = plan.RequiredLaborDays,
                DelayReason = isShort ? "Heavy rain stopped work for four days" : null
            }, ct);
            actuals++;
        }

        // A block whose planting has been recorded is carrying a crop, and the block master has
        // to say so: the land-coverage tree reads CurrentCropStatus for its "area under cane"
        // figure, and leaving every block Fallow made a planted estate report none.
        var plantedBlockIds = planting.Select(p => p.BlockId).Distinct().ToHashSet();

        foreach (var block in await db.Blocks.Where(b => plantedBlockIds.Contains(b.Id)).ToListAsync(ct))
            block.CurrentCropStatus = CropStatus.Growing;

        await db.SaveChangesAsync(ct);

        return (generated.PlansCreated, bookings, actuals);
    }

    /// <summary>The 19 sample activities from section 6, with their standards and requirements.</summary>
    private static List<PlantingActivity> BuildActivities(int companyId)
    {
        // code, name, category, seq, offset, cap/hour, cap/day, hours/ha, labor-days/ha, tractor, equip, material, labor, equipment category
        var rows = new (string Code, string Name, ActivityCategory Category, int Seq, int Offset, decimal PerHour,
            decimal PerDay, decimal HoursPerHa, decimal LaborDays, bool Tractor, bool Equipment, bool Material, bool Labor,
            EquipmentCategory? EquipCategory, CropType Crop)[]
        {
            ("A001", "Land survey", ActivityCategory.Survey, 1, -45, 3.0m, 20m, 0.4m, 0.2m, false, false, false, true, null, CropType.NewPlanting),
            ("A002", "Land clearing", ActivityCategory.LandPreparation, 2, -40, 0.5m, 4m, 2.0m, 1.2m, true, true, false, true, EquipmentCategory.Other, CropType.NewPlanting),
            ("A003", "First plowing", ActivityCategory.LandPreparation, 3, -35, 0.6m, 4.5m, 1.8m, 0.3m, true, true, false, false, EquipmentCategory.DiscPlow, CropType.NewPlanting),
            ("A004", "Second plowing", ActivityCategory.LandPreparation, 4, -30, 0.6m, 4.5m, 1.8m, 0.3m, true, true, false, false, EquipmentCategory.MoldboardPlow, CropType.NewPlanting),
            ("A005", "Harrowing", ActivityCategory.LandPreparation, 5, -25, 0.9m, 7m, 1.2m, 0.2m, true, true, false, false, EquipmentCategory.Harrow, CropType.Both),
            ("A006", "Land leveling", ActivityCategory.LandPreparation, 6, -20, 0.5m, 4m, 2.0m, 0.3m, true, true, false, false, EquipmentCategory.LandLeveler, CropType.NewPlanting),
            ("A007", "Drainage preparation", ActivityCategory.LandPreparation, 7, -16, 0.4m, 3m, 2.6m, 0.8m, true, true, false, true, EquipmentCategory.Other, CropType.NewPlanting),
            ("A008", "Furrow preparation", ActivityCategory.LandPreparation, 8, -12, 0.8m, 6m, 1.4m, 0.2m, true, true, false, false, EquipmentCategory.FurrowOpener, CropType.NewPlanting),
            ("A009", "Seed cane cutting", ActivityCategory.SeedPreparation, 9, -8, 1.0m, 8m, 1.0m, 1.5m, false, true, false, true, EquipmentCategory.SeedCutter, CropType.NewPlanting),
            ("A010", "Seed cane transportation", ActivityCategory.SeedPreparation, 10, -5, 1.5m, 12m, 0.7m, 0.6m, true, true, false, true, EquipmentCategory.Trailer, CropType.NewPlanting),
            ("A011", "Seed treatment", ActivityCategory.SeedPreparation, 11, -3, 1.2m, 9m, 0.9m, 0.5m, false, false, true, true, null, CropType.NewPlanting),
            ("A012", "Planting", ActivityCategory.Planting, 12, 0, 0.5m, 4m, 2.2m, 2.5m, true, true, true, true, EquipmentCategory.CanePlanter, CropType.Both),
            ("A013", "Basal fertilizer application", ActivityCategory.Fertilization, 13, 1, 1.2m, 9m, 0.9m, 0.4m, true, true, true, true, EquipmentCategory.FertilizerSpreader, CropType.Both),
            ("A014", "Pre-emergence herbicide application", ActivityCategory.WeedControl, 14, 3, 1.5m, 11m, 0.8m, 0.3m, true, true, true, true, EquipmentCategory.ChemicalSprayer, CropType.Both),
            ("A015", "Irrigation", ActivityCategory.Irrigation, 15, 5, 1.0m, 8m, 1.0m, 0.5m, true, true, true, true, EquipmentCategory.WaterTruck, CropType.Both),
            ("A016", "Gap filling", ActivityCategory.CropCare, 16, 30, 0.8m, 6m, 1.3m, 1.8m, false, false, true, true, null, CropType.Both),
            ("A017", "First cultivation", ActivityCategory.CropCare, 17, 45, 0.9m, 7m, 1.2m, 0.3m, true, true, false, false, EquipmentCategory.Harrow, CropType.Both),
            ("A018", "First fertilizer application", ActivityCategory.Fertilization, 18, 55, 1.2m, 9m, 0.9m, 0.4m, true, true, true, true, EquipmentCategory.FertilizerSpreader, CropType.Both),
            ("A019", "Post-emergence herbicide application", ActivityCategory.WeedControl, 19, 65, 1.5m, 11m, 0.8m, 0.3m, true, true, true, true, EquipmentCategory.ChemicalSprayer, CropType.Both)
        };

        return rows.Select(r => new PlantingActivity
        {
            CompanyId = companyId,
            Code = r.Code,
            Name = r.Name,
            Category = r.Category,
            SequenceNo = r.Seq,
            ApplicableCropType = r.Crop,
            StandardStartDayOffset = r.Offset,
            StandardCapacityPerHour = r.PerHour,
            StandardCapacityPerDay = r.PerDay,
            StandardDurationPerHa = r.HoursPerHa,
            StandardLaborDaysPerHa = r.LaborDays,
            IsMandatory = r.Code is not ("A016" or "A007"),
            RequiresTractor = r.Tractor,
            RequiresEquipment = r.Equipment,
            RequiresMaterial = r.Material,
            RequiresLabor = r.Labor,
            AllowOverlap = r.Code is "A001" or "A016",
            DefaultEquipmentCategory = r.EquipCategory
        }).ToList();
    }

    // ---------------------------------------------------------------- identity

    private static async Task SeedRolesAsync(IServiceProvider services)
    {
        var roleManager = services.GetRequiredService<RoleManager<AppRole>>();
        foreach (var role in AppRoles.All)
            if (!await roleManager.RoleExistsAsync(role))
                await roleManager.CreateAsync(new AppRole(role) { Description = $"{role} (seeded)" });
    }

    /// <summary>Demo accounts, one per major role, all sharing <see cref="DemoPassword"/>.</summary>
    private static async Task SeedUsersAsync(IServiceProvider services, int companyId)
    {
        var userManager = services.GetRequiredService<UserManager<AppUser>>();

        var demoUsers = new (string UserName, string FullName, string Role)[]
        {
            ("admin", "System Administrator", AppRoles.SystemAdministrator),
            ("director", "Plantation Director", AppRoles.PlantationDirector),
            ("manager", "Plantation Manager", AppRoles.PlantationManager),
            ("farmmanager", "Farm Manager", AppRoles.FarmManager),
            ("planner", "Agricultural Planner", AppRoles.AgriculturalPlanner),
            ("machinery", "Machinery Manager", AppRoles.MachineryManager),
            ("materials", "Material Planner", AppRoles.MaterialPlanner),
            ("supervisor", "Field Supervisor", AppRoles.FieldSupervisor),
            ("approver", "Management Approver", AppRoles.ManagementApprover),
            ("viewer", "Report Viewer", AppRoles.ReportViewer)
        };

        foreach (var (userName, fullName, role) in demoUsers)
        {
            var user = await userManager.FindByNameAsync(userName);
            if (user is null)
            {
                user = new AppUser
                {
                    UserName = userName,
                    Email = $"{userName}@sugarcane.example",
                    EmailConfirmed = true,
                    FullName = fullName,
                    CompanyId = companyId,
                    IsActive = true
                };
                var created = await userManager.CreateAsync(user, DemoPassword);
                if (!created.Succeeded)
                    throw new InvalidOperationException(
                        $"Could not create demo user '{userName}': {string.Join("; ", created.Errors.Select(e => e.Description))}");
            }

            if (!await userManager.IsInRoleAsync(user, role))
                await userManager.AddToRoleAsync(user, role);
        }
    }
}
