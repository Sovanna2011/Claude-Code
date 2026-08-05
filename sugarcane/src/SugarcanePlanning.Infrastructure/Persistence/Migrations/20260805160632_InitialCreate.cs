using System;
using Microsoft.EntityFrameworkCore.Migrations;

#nullable disable

namespace SugarcanePlanning.Infrastructure.Persistence.Migrations
{
    /// <inheritdoc />
    public partial class InitialCreate : Migration
    {
        /// <inheritdoc />
        protected override void Up(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.EnsureSchema(
                name: "planning");

            migrationBuilder.CreateTable(
                name: "AuditLogs",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: true),
                    UserName = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: false),
                    TimestampUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    Action = table.Column<int>(type: "int", nullable: false),
                    TableName = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: false),
                    RecordId = table.Column<string>(type: "nvarchar(60)", maxLength: 60, nullable: false),
                    OldValues = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    NewValues = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    ChangedColumns = table.Column<string>(type: "nvarchar(2000)", maxLength: 2000, nullable: true),
                    IpAddress = table.Column<string>(type: "nvarchar(60)", maxLength: 60, nullable: true),
                    DeviceInfo = table.Column<string>(type: "nvarchar(300)", maxLength: 300, nullable: true),
                    Remarks = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_AuditLogs", x => x.Id);
                });

            migrationBuilder.CreateTable(
                name: "CaneVarieties",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    RecommendedSoilType = table.Column<int>(type: "int", nullable: false),
                    SeedRatePerHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    GrowingPeriodMonths = table.Column<int>(type: "int", nullable: false),
                    ExpectedYieldPerHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ExpectedLossPercent = table.Column<decimal>(type: "decimal(9,4)", precision: 9, scale: 4, nullable: false),
                    RecommendedPlantingStartMonth = table.Column<int>(type: "int", nullable: false),
                    RecommendedPlantingEndMonth = table.Column<int>(type: "int", nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_CaneVarieties", x => x.Id);
                    table.CheckConstraint("CK_Variety_Loss", "[ExpectedLossPercent] >= 0 AND [ExpectedLossPercent] <= 100");
                    table.CheckConstraint("CK_Variety_Months", "[RecommendedPlantingStartMonth] BETWEEN 1 AND 12 AND [RecommendedPlantingEndMonth] BETWEEN 1 AND 12");
                });

            migrationBuilder.CreateTable(
                name: "Companies",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    Address = table.Column<string>(type: "nvarchar(400)", maxLength: 400, nullable: true),
                    BaseCurrency = table.Column<string>(type: "nvarchar(3)", maxLength: 3, nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Companies", x => x.Id);
                });

            migrationBuilder.CreateTable(
                name: "Materials",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    Category = table.Column<int>(type: "int", nullable: false),
                    BaseUnit = table.Column<int>(type: "int", nullable: false),
                    AlternativeUnit = table.Column<int>(type: "int", nullable: true),
                    UnitConversionFactor = table.Column<decimal>(type: "decimal(18,6)", precision: 18, scale: 6, nullable: false),
                    StandardRatePerHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    MinApplicationRate = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    MaxApplicationRate = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Materials", x => x.Id);
                    table.CheckConstraint("CK_Material_Conversion", "[UnitConversionFactor] > 0");
                    table.CheckConstraint("CK_Material_Rates", "[MinApplicationRate] >= 0 AND [MaxApplicationRate] >= 0");
                });

            migrationBuilder.CreateTable(
                name: "PlantingActivities",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    Category = table.Column<int>(type: "int", nullable: false),
                    SequenceNo = table.Column<int>(type: "int", nullable: false),
                    ApplicableCropType = table.Column<int>(type: "int", nullable: false),
                    StandardStartDayOffset = table.Column<int>(type: "int", nullable: false),
                    StandardCapacityPerHour = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    StandardCapacityPerDay = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    StandardDurationPerHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    IsMandatory = table.Column<bool>(type: "bit", nullable: false),
                    RequiresTractor = table.Column<bool>(type: "bit", nullable: false),
                    RequiresEquipment = table.Column<bool>(type: "bit", nullable: false),
                    RequiresMaterial = table.Column<bool>(type: "bit", nullable: false),
                    RequiresLabor = table.Column<bool>(type: "bit", nullable: false),
                    AllowOverlap = table.Column<bool>(type: "bit", nullable: false),
                    StandardLaborDaysPerHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    DefaultEquipmentCategory = table.Column<int>(type: "int", nullable: true),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_PlantingActivities", x => x.Id);
                    table.CheckConstraint("CK_Activity_Sequence", "[SequenceNo] > 0");
                });

            migrationBuilder.CreateTable(
                name: "Roles",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    Description = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    Name = table.Column<string>(type: "nvarchar(256)", maxLength: 256, nullable: true),
                    NormalizedName = table.Column<string>(type: "nvarchar(256)", maxLength: 256, nullable: true),
                    ConcurrencyStamp = table.Column<string>(type: "nvarchar(max)", nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Roles", x => x.Id);
                });

            migrationBuilder.CreateTable(
                name: "Users",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    FullName = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    LastLoginUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    UserName = table.Column<string>(type: "nvarchar(256)", maxLength: 256, nullable: true),
                    NormalizedUserName = table.Column<string>(type: "nvarchar(256)", maxLength: 256, nullable: true),
                    Email = table.Column<string>(type: "nvarchar(256)", maxLength: 256, nullable: true),
                    NormalizedEmail = table.Column<string>(type: "nvarchar(256)", maxLength: 256, nullable: true),
                    EmailConfirmed = table.Column<bool>(type: "bit", nullable: false),
                    PasswordHash = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    SecurityStamp = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    ConcurrencyStamp = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    PhoneNumber = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    PhoneNumberConfirmed = table.Column<bool>(type: "bit", nullable: false),
                    TwoFactorEnabled = table.Column<bool>(type: "bit", nullable: false),
                    LockoutEnd = table.Column<DateTimeOffset>(type: "datetimeoffset", nullable: true),
                    LockoutEnabled = table.Column<bool>(type: "bit", nullable: false),
                    AccessFailedCount = table.Column<int>(type: "int", nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Users", x => x.Id);
                });

            migrationBuilder.CreateTable(
                name: "Estates",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    Location = table.Column<string>(type: "nvarchar(200)", maxLength: 200, nullable: true),
                    TotalAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Estates", x => x.Id);
                    table.CheckConstraint("CK_Estate_Area", "[TotalAreaHa] >= 0");
                    table.ForeignKey(
                        name: "FK_Estates_Companies_CompanyId",
                        column: x => x.CompanyId,
                        principalSchema: "planning",
                        principalTable: "Companies",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "GrowingSeasons",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    StartDate = table.Column<DateOnly>(type: "date", nullable: false),
                    EndDate = table.Column<DateOnly>(type: "date", nullable: false),
                    PlannedPlantingStart = table.Column<DateOnly>(type: "date", nullable: false),
                    PlannedPlantingEnd = table.Column<DateOnly>(type: "date", nullable: false),
                    ExpectedHarvestStart = table.Column<DateOnly>(type: "date", nullable: false),
                    ExpectedHarvestEnd = table.Column<DateOnly>(type: "date", nullable: false),
                    Status = table.Column<int>(type: "int", nullable: false),
                    Remarks = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_GrowingSeasons", x => x.Id);
                    table.CheckConstraint("CK_Season_Dates", "[EndDate] >= [StartDate]");
                    table.ForeignKey(
                        name: "FK_GrowingSeasons_Companies_CompanyId",
                        column: x => x.CompanyId,
                        principalSchema: "planning",
                        principalTable: "Companies",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "ActivityDependencies",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ActivityId = table.Column<int>(type: "int", nullable: false),
                    PredecessorActivityId = table.Column<int>(type: "int", nullable: false),
                    LagDays = table.Column<int>(type: "int", nullable: false),
                    IsBlocking = table.Column<bool>(type: "bit", nullable: false),
                    Remarks = table.Column<string>(type: "nvarchar(500)", maxLength: 500, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ActivityDependencies", x => x.Id);
                    table.CheckConstraint("CK_Dependency_NotSelf", "[ActivityId] <> [PredecessorActivityId]");
                    table.ForeignKey(
                        name: "FK_ActivityDependencies_PlantingActivities_ActivityId",
                        column: x => x.ActivityId,
                        principalSchema: "planning",
                        principalTable: "PlantingActivities",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_ActivityDependencies_PlantingActivities_PredecessorActivityId",
                        column: x => x.PredecessorActivityId,
                        principalSchema: "planning",
                        principalTable: "PlantingActivities",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "ActivityMaterialStandards",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ActivityId = table.Column<int>(type: "int", nullable: false),
                    MaterialId = table.Column<int>(type: "int", nullable: false),
                    CropType = table.Column<int>(type: "int", nullable: false),
                    CaneVarietyId = table.Column<int>(type: "int", nullable: true),
                    SoilType = table.Column<int>(type: "int", nullable: true),
                    StandardRatePerHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    NumberOfApplications = table.Column<int>(type: "int", nullable: false),
                    WastePercent = table.Column<decimal>(type: "decimal(9,4)", precision: 9, scale: 4, nullable: false),
                    EffectiveFrom = table.Column<DateOnly>(type: "date", nullable: false),
                    EffectiveTo = table.Column<DateOnly>(type: "date", nullable: true),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ActivityMaterialStandards", x => x.Id);
                    table.CheckConstraint("CK_Standard_Applications", "[NumberOfApplications] >= 1");
                    table.CheckConstraint("CK_Standard_Rate", "[StandardRatePerHa] > 0");
                    table.CheckConstraint("CK_Standard_Waste", "[WastePercent] >= 0 AND [WastePercent] <= 100");
                    table.ForeignKey(
                        name: "FK_ActivityMaterialStandards_CaneVarieties_CaneVarietyId",
                        column: x => x.CaneVarietyId,
                        principalSchema: "planning",
                        principalTable: "CaneVarieties",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ActivityMaterialStandards_Materials_MaterialId",
                        column: x => x.MaterialId,
                        principalSchema: "planning",
                        principalTable: "Materials",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ActivityMaterialStandards_PlantingActivities_ActivityId",
                        column: x => x.ActivityId,
                        principalSchema: "planning",
                        principalTable: "PlantingActivities",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "RoleClaims",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    RoleId = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    ClaimType = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    ClaimValue = table.Column<string>(type: "nvarchar(max)", nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_RoleClaims", x => x.Id);
                    table.ForeignKey(
                        name: "FK_RoleClaims_Roles_RoleId",
                        column: x => x.RoleId,
                        principalSchema: "planning",
                        principalTable: "Roles",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "UserClaims",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    UserId = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    ClaimType = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    ClaimValue = table.Column<string>(type: "nvarchar(max)", nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_UserClaims", x => x.Id);
                    table.ForeignKey(
                        name: "FK_UserClaims_Users_UserId",
                        column: x => x.UserId,
                        principalSchema: "planning",
                        principalTable: "Users",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "UserLogins",
                schema: "planning",
                columns: table => new
                {
                    LoginProvider = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    ProviderKey = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    ProviderDisplayName = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    UserId = table.Column<string>(type: "nvarchar(450)", nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_UserLogins", x => new { x.LoginProvider, x.ProviderKey });
                    table.ForeignKey(
                        name: "FK_UserLogins_Users_UserId",
                        column: x => x.UserId,
                        principalSchema: "planning",
                        principalTable: "Users",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "UserRoles",
                schema: "planning",
                columns: table => new
                {
                    UserId = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    RoleId = table.Column<string>(type: "nvarchar(450)", nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_UserRoles", x => new { x.UserId, x.RoleId });
                    table.ForeignKey(
                        name: "FK_UserRoles_Roles_RoleId",
                        column: x => x.RoleId,
                        principalSchema: "planning",
                        principalTable: "Roles",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_UserRoles_Users_UserId",
                        column: x => x.UserId,
                        principalSchema: "planning",
                        principalTable: "Users",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "UserTokens",
                schema: "planning",
                columns: table => new
                {
                    UserId = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    LoginProvider = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    Name = table.Column<string>(type: "nvarchar(450)", nullable: false),
                    Value = table.Column<string>(type: "nvarchar(max)", nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_UserTokens", x => new { x.UserId, x.LoginProvider, x.Name });
                    table.ForeignKey(
                        name: "FK_UserTokens_Users_UserId",
                        column: x => x.UserId,
                        principalSchema: "planning",
                        principalTable: "Users",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "Farms",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    EstateId = table.Column<int>(type: "int", nullable: false),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    ManagerName = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: true),
                    TotalAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Farms", x => x.Id);
                    table.CheckConstraint("CK_Farm_Area", "[TotalAreaHa] >= 0");
                    table.ForeignKey(
                        name: "FK_Farms_Estates_EstateId",
                        column: x => x.EstateId,
                        principalSchema: "planning",
                        principalTable: "Estates",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "MaterialStocks",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    EstateId = table.Column<int>(type: "int", nullable: true),
                    MaterialId = table.Column<int>(type: "int", nullable: false),
                    AvailableStock = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ReservedQuantity = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    IncomingQuantity = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    IncomingExpectedDate = table.Column<DateOnly>(type: "date", nullable: true),
                    Unit = table.Column<int>(type: "int", nullable: false),
                    LastSyncedUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    SourceSystem = table.Column<string>(type: "nvarchar(50)", maxLength: 50, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_MaterialStocks", x => x.Id);
                    table.CheckConstraint("CK_Stock_Quantities", "[AvailableStock] >= 0 AND [ReservedQuantity] >= 0 AND [IncomingQuantity] >= 0");
                    table.ForeignKey(
                        name: "FK_MaterialStocks_Estates_EstateId",
                        column: x => x.EstateId,
                        principalSchema: "planning",
                        principalTable: "Estates",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_MaterialStocks_Materials_MaterialId",
                        column: x => x.MaterialId,
                        principalSchema: "planning",
                        principalTable: "Materials",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "PlantingProjections",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    EstateId = table.Column<int>(type: "int", nullable: false),
                    GrowingSeasonId = table.Column<int>(type: "int", nullable: false),
                    ProjectionNo = table.Column<string>(type: "nvarchar(30)", maxLength: 30, nullable: false),
                    Version = table.Column<int>(type: "int", nullable: false),
                    ProjectionDate = table.Column<DateOnly>(type: "date", nullable: false),
                    PlanningStartDate = table.Column<DateOnly>(type: "date", nullable: false),
                    PlanningEndDate = table.Column<DateOnly>(type: "date", nullable: false),
                    TotalProjectedAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    TotalExpectedProductionTons = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    Status = table.Column<int>(type: "int", nullable: false),
                    PreparedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    SubmittedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    SubmittedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ReviewedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ReviewedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ApprovedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ApprovedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    RejectedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RejectedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    RejectionReason = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    RevisedFromProjectionId = table.Column<int>(type: "int", nullable: true),
                    RevisionReason = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    IsReadOnly = table.Column<bool>(type: "bit", nullable: false),
                    IsCurrentVersion = table.Column<bool>(type: "bit", nullable: false),
                    Remarks = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_PlantingProjections", x => x.Id);
                    table.CheckConstraint("CK_Projection_Dates", "[PlanningEndDate] >= [PlanningStartDate]");
                    table.CheckConstraint("CK_Projection_Version", "[Version] > 0");
                    table.ForeignKey(
                        name: "FK_PlantingProjections_Estates_EstateId",
                        column: x => x.EstateId,
                        principalSchema: "planning",
                        principalTable: "Estates",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_PlantingProjections_GrowingSeasons_GrowingSeasonId",
                        column: x => x.GrowingSeasonId,
                        principalSchema: "planning",
                        principalTable: "GrowingSeasons",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_PlantingProjections_PlantingProjections_RevisedFromProjectionId",
                        column: x => x.RevisedFromProjectionId,
                        principalSchema: "planning",
                        principalTable: "PlantingProjections",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "Equipment",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    EstateId = table.Column<int>(type: "int", nullable: true),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    Category = table.Column<int>(type: "int", nullable: false),
                    MinimumTractorHp = table.Column<int>(type: "int", nullable: false),
                    CapacityPerHour = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    CapacityPerDay = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    CurrentFarmId = table.Column<int>(type: "int", nullable: true),
                    Ownership = table.Column<int>(type: "int", nullable: false),
                    Availability = table.Column<int>(type: "int", nullable: false),
                    Maintenance = table.Column<int>(type: "int", nullable: false),
                    MaintenanceFromDate = table.Column<DateOnly>(type: "date", nullable: true),
                    MaintenanceToDate = table.Column<DateOnly>(type: "date", nullable: true),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Equipment", x => x.Id);
                    table.CheckConstraint("CK_Equipment_Hp", "[MinimumTractorHp] >= 0");
                    table.ForeignKey(
                        name: "FK_Equipment_Estates_EstateId",
                        column: x => x.EstateId,
                        principalSchema: "planning",
                        principalTable: "Estates",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.SetNull);
                    table.ForeignKey(
                        name: "FK_Equipment_Farms_CurrentFarmId",
                        column: x => x.CurrentFarmId,
                        principalSchema: "planning",
                        principalTable: "Farms",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.SetNull);
                });

            migrationBuilder.CreateTable(
                name: "Tractors",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    EstateId = table.Column<int>(type: "int", nullable: true),
                    AssetNo = table.Column<string>(type: "nvarchar(30)", maxLength: 30, nullable: false),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    RegistrationNo = table.Column<string>(type: "nvarchar(30)", maxLength: 30, nullable: true),
                    Brand = table.Column<string>(type: "nvarchar(60)", maxLength: 60, nullable: true),
                    Model = table.Column<string>(type: "nvarchar(60)", maxLength: 60, nullable: true),
                    Horsepower = table.Column<int>(type: "int", nullable: false),
                    CurrentFarmId = table.Column<int>(type: "int", nullable: true),
                    Ownership = table.Column<int>(type: "int", nullable: false),
                    DailyCapacityHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    FuelConsumptionPerHour = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    FuelConsumptionPerHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    Availability = table.Column<int>(type: "int", nullable: false),
                    Maintenance = table.Column<int>(type: "int", nullable: false),
                    MaintenanceFromDate = table.Column<DateOnly>(type: "date", nullable: true),
                    MaintenanceToDate = table.Column<DateOnly>(type: "date", nullable: true),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Tractors", x => x.Id);
                    table.CheckConstraint("CK_Tractor_Capacity", "[DailyCapacityHa] >= 0");
                    table.CheckConstraint("CK_Tractor_Hp", "[Horsepower] > 0");
                    table.ForeignKey(
                        name: "FK_Tractors_Estates_EstateId",
                        column: x => x.EstateId,
                        principalSchema: "planning",
                        principalTable: "Estates",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.SetNull);
                    table.ForeignKey(
                        name: "FK_Tractors_Farms_CurrentFarmId",
                        column: x => x.CurrentFarmId,
                        principalSchema: "planning",
                        principalTable: "Farms",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.SetNull);
                });

            migrationBuilder.CreateTable(
                name: "WorkTeams",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    FarmId = table.Column<int>(type: "int", nullable: true),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    SupervisorName = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: true),
                    PrimarySkill = table.Column<int>(type: "int", nullable: false),
                    MemberCount = table.Column<int>(type: "int", nullable: false),
                    StandardHoursPerDay = table.Column<decimal>(type: "decimal(9,2)", precision: 9, scale: 2, nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_WorkTeams", x => x.Id);
                    table.CheckConstraint("CK_Team_Members", "[MemberCount] >= 0");
                    table.ForeignKey(
                        name: "FK_WorkTeams_Farms_FarmId",
                        column: x => x.FarmId,
                        principalSchema: "planning",
                        principalTable: "Farms",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.SetNull);
                });

            migrationBuilder.CreateTable(
                name: "Zones",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    FarmId = table.Column<int>(type: "int", nullable: false),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    SupervisorName = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: true),
                    TotalAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Zones", x => x.Id);
                    table.CheckConstraint("CK_Zone_Area", "[TotalAreaHa] >= 0");
                    table.ForeignKey(
                        name: "FK_Zones_Farms_FarmId",
                        column: x => x.FarmId,
                        principalSchema: "planning",
                        principalTable: "Farms",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "PlanningScenarios",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ProjectionId = table.Column<int>(type: "int", nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    Description = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    Status = table.Column<int>(type: "int", nullable: false),
                    SimulatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    SimulatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ResultJson = table.Column<string>(type: "nvarchar(max)", nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_PlanningScenarios", x => x.Id);
                    table.ForeignKey(
                        name: "FK_PlanningScenarios_PlantingProjections_ProjectionId",
                        column: x => x.ProjectionId,
                        principalSchema: "planning",
                        principalTable: "PlantingProjections",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "ProjectionApprovalHistory",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ProjectionId = table.Column<int>(type: "int", nullable: false),
                    Action = table.Column<int>(type: "int", nullable: false),
                    FromStatus = table.Column<int>(type: "int", nullable: false),
                    ToStatus = table.Column<int>(type: "int", nullable: false),
                    ActionBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: false),
                    ActionAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    Comments = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ProjectionApprovalHistory", x => x.Id);
                    table.ForeignKey(
                        name: "FK_ProjectionApprovalHistory_PlantingProjections_ProjectionId",
                        column: x => x.ProjectionId,
                        principalSchema: "planning",
                        principalTable: "PlantingProjections",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "TractorEquipmentCompatibility",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    TractorId = table.Column<int>(type: "int", nullable: false),
                    EquipmentId = table.Column<int>(type: "int", nullable: false),
                    IsRecommended = table.Column<bool>(type: "bit", nullable: false),
                    Remarks = table.Column<string>(type: "nvarchar(500)", maxLength: 500, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_TractorEquipmentCompatibility", x => x.Id);
                    table.ForeignKey(
                        name: "FK_TractorEquipmentCompatibility_Equipment_EquipmentId",
                        column: x => x.EquipmentId,
                        principalSchema: "planning",
                        principalTable: "Equipment",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_TractorEquipmentCompatibility_Tractors_TractorId",
                        column: x => x.TractorId,
                        principalSchema: "planning",
                        principalTable: "Tractors",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                });

            migrationBuilder.CreateTable(
                name: "Operators",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    EstateId = table.Column<int>(type: "int", nullable: true),
                    FarmId = table.Column<int>(type: "int", nullable: true),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    FullName = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    Skill = table.Column<int>(type: "int", nullable: false),
                    LicenseNo = table.Column<string>(type: "nvarchar(40)", maxLength: 40, nullable: true),
                    LicenseExpiry = table.Column<DateOnly>(type: "date", nullable: true),
                    WorkTeamId = table.Column<int>(type: "int", nullable: true),
                    StandardHoursPerDay = table.Column<decimal>(type: "decimal(9,2)", precision: 9, scale: 2, nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Operators", x => x.Id);
                    table.CheckConstraint("CK_Operator_Hours", "[StandardHoursPerDay] > 0");
                    table.ForeignKey(
                        name: "FK_Operators_Farms_FarmId",
                        column: x => x.FarmId,
                        principalSchema: "planning",
                        principalTable: "Farms",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.SetNull);
                    table.ForeignKey(
                        name: "FK_Operators_WorkTeams_WorkTeamId",
                        column: x => x.WorkTeamId,
                        principalSchema: "planning",
                        principalTable: "WorkTeams",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.SetNull);
                });

            migrationBuilder.CreateTable(
                name: "PlantationBlocks",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ZoneId = table.Column<int>(type: "int", nullable: false),
                    Code = table.Column<string>(type: "nvarchar(20)", maxLength: 20, nullable: false),
                    Name = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: false),
                    TotalAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    PlantableAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    Latitude = table.Column<decimal>(type: "decimal(9,6)", precision: 9, scale: 6, nullable: true),
                    Longitude = table.Column<decimal>(type: "decimal(9,6)", precision: 9, scale: 6, nullable: true),
                    MapReference = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    SoilType = table.Column<int>(type: "int", nullable: false),
                    LandCondition = table.Column<int>(type: "int", nullable: false),
                    Irrigation = table.Column<int>(type: "int", nullable: false),
                    CurrentCropStatus = table.Column<int>(type: "int", nullable: false),
                    IsActive = table.Column<bool>(type: "bit", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_PlantationBlocks", x => x.Id);
                    table.CheckConstraint("CK_Block_Area", "[TotalAreaHa] >= 0 AND [PlantableAreaHa] >= 0");
                    table.CheckConstraint("CK_Block_Plantable", "[PlantableAreaHa] <= [TotalAreaHa]");
                    table.ForeignKey(
                        name: "FK_PlantationBlocks_Zones_ZoneId",
                        column: x => x.ZoneId,
                        principalSchema: "planning",
                        principalTable: "Zones",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "ScenarioAdjustments",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ScenarioId = table.Column<int>(type: "int", nullable: false),
                    AdjustmentType = table.Column<int>(type: "int", nullable: false),
                    Value = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ActivityId = table.Column<int>(type: "int", nullable: true),
                    Remarks = table.Column<string>(type: "nvarchar(500)", maxLength: 500, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ScenarioAdjustments", x => x.Id);
                    table.ForeignKey(
                        name: "FK_ScenarioAdjustments_PlanningScenarios_ScenarioId",
                        column: x => x.ScenarioId,
                        principalSchema: "planning",
                        principalTable: "PlanningScenarios",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_ScenarioAdjustments_PlantingActivities_ActivityId",
                        column: x => x.ActivityId,
                        principalSchema: "planning",
                        principalTable: "PlantingActivities",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "ProjectionLines",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ProjectionId = table.Column<int>(type: "int", nullable: false),
                    FarmId = table.Column<int>(type: "int", nullable: false),
                    ZoneId = table.Column<int>(type: "int", nullable: false),
                    BlockId = table.Column<int>(type: "int", nullable: false),
                    CropType = table.Column<int>(type: "int", nullable: false),
                    CaneVarietyId = table.Column<int>(type: "int", nullable: false),
                    AvailableAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ProjectedPlantingAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    PlannedPlantingStart = table.Column<DateOnly>(type: "date", nullable: false),
                    PlannedPlantingEnd = table.Column<DateOnly>(type: "date", nullable: false),
                    ExpectedHarvestDate = table.Column<DateOnly>(type: "date", nullable: true),
                    ExpectedYieldPerHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ExpectedLossPercent = table.Column<decimal>(type: "decimal(9,4)", precision: 9, scale: 4, nullable: false),
                    HarvestableAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ExpectedCaneProductionTons = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    Priority = table.Column<int>(type: "int", nullable: false),
                    Remarks = table.Column<string>(type: "nvarchar(500)", maxLength: 500, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ProjectionLines", x => x.Id);
                    table.CheckConstraint("CK_Line_Area", "[ProjectedPlantingAreaHa] > 0");
                    table.CheckConstraint("CK_Line_Dates", "[PlannedPlantingEnd] >= [PlannedPlantingStart]");
                    table.CheckConstraint("CK_Line_Loss", "[ExpectedLossPercent] >= 0 AND [ExpectedLossPercent] <= 100");
                    table.CheckConstraint("CK_Line_Priority", "[Priority] BETWEEN 1 AND 9");
                    table.ForeignKey(
                        name: "FK_ProjectionLines_CaneVarieties_CaneVarietyId",
                        column: x => x.CaneVarietyId,
                        principalSchema: "planning",
                        principalTable: "CaneVarieties",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ProjectionLines_Farms_FarmId",
                        column: x => x.FarmId,
                        principalSchema: "planning",
                        principalTable: "Farms",
                        principalColumn: "Id");
                    table.ForeignKey(
                        name: "FK_ProjectionLines_PlantationBlocks_BlockId",
                        column: x => x.BlockId,
                        principalSchema: "planning",
                        principalTable: "PlantationBlocks",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ProjectionLines_PlantingProjections_ProjectionId",
                        column: x => x.ProjectionId,
                        principalSchema: "planning",
                        principalTable: "PlantingProjections",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_ProjectionLines_Zones_ZoneId",
                        column: x => x.ZoneId,
                        principalSchema: "planning",
                        principalTable: "Zones",
                        principalColumn: "Id");
                });

            migrationBuilder.CreateTable(
                name: "ActivityPlans",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ProjectionId = table.Column<int>(type: "int", nullable: false),
                    ProjectionLineId = table.Column<int>(type: "int", nullable: false),
                    FarmId = table.Column<int>(type: "int", nullable: false),
                    ZoneId = table.Column<int>(type: "int", nullable: false),
                    BlockId = table.Column<int>(type: "int", nullable: false),
                    ActivityId = table.Column<int>(type: "int", nullable: false),
                    SequenceNo = table.Column<int>(type: "int", nullable: false),
                    PlannedAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    PlannedStartDate = table.Column<DateOnly>(type: "date", nullable: false),
                    PlannedEndDate = table.Column<DateOnly>(type: "date", nullable: false),
                    WorkingDays = table.Column<int>(type: "int", nullable: false),
                    DailyTargetHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    PlannedWorkingHours = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    RequiredTractorType = table.Column<string>(type: "nvarchar(60)", maxLength: 60, nullable: true),
                    RequiredEquipmentCategory = table.Column<int>(type: "int", nullable: true),
                    RequiredTractorCount = table.Column<int>(type: "int", nullable: false),
                    RequiredEquipmentCount = table.Column<int>(type: "int", nullable: false),
                    RequiredLaborDays = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    RequiredWorkers = table.Column<int>(type: "int", nullable: false),
                    PlannedFuelLiters = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    SupervisorName = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: true),
                    Status = table.Column<int>(type: "int", nullable: false),
                    Remarks = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ActivityPlans", x => x.Id);
                    table.CheckConstraint("CK_Plan_Area", "[PlannedAreaHa] >= 0");
                    table.CheckConstraint("CK_Plan_Dates", "[PlannedEndDate] >= [PlannedStartDate]");
                    table.ForeignKey(
                        name: "FK_ActivityPlans_PlantationBlocks_BlockId",
                        column: x => x.BlockId,
                        principalSchema: "planning",
                        principalTable: "PlantationBlocks",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ActivityPlans_PlantingActivities_ActivityId",
                        column: x => x.ActivityId,
                        principalSchema: "planning",
                        principalTable: "PlantingActivities",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ActivityPlans_PlantingProjections_ProjectionId",
                        column: x => x.ProjectionId,
                        principalSchema: "planning",
                        principalTable: "PlantingProjections",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_ActivityPlans_ProjectionLines_ProjectionLineId",
                        column: x => x.ProjectionLineId,
                        principalSchema: "planning",
                        principalTable: "ProjectionLines",
                        principalColumn: "Id");
                });

            migrationBuilder.CreateTable(
                name: "ActivityActuals",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ActivityPlanId = table.Column<int>(type: "int", nullable: false),
                    ActualStartDate = table.Column<DateOnly>(type: "date", nullable: true),
                    ActualCompletionDate = table.Column<DateOnly>(type: "date", nullable: true),
                    ActualCompletedAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ActualTractorId = table.Column<int>(type: "int", nullable: true),
                    ActualEquipmentId = table.Column<int>(type: "int", nullable: true),
                    ActualOperatorId = table.Column<int>(type: "int", nullable: true),
                    ActualWorkingHours = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ActualFuelLiters = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ActualLaborDays = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    AreaVariance = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    FuelVariance = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ScheduleVarianceDays = table.Column<int>(type: "int", nullable: true),
                    CompletionPercent = table.Column<decimal>(type: "decimal(9,4)", precision: 9, scale: 4, nullable: false),
                    DelayReason = table.Column<string>(type: "nvarchar(500)", maxLength: 500, nullable: true),
                    Remarks = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ActivityActuals", x => x.Id);
                    table.CheckConstraint("CK_Actual_Area", "[ActualCompletedAreaHa] >= 0");
                    table.ForeignKey(
                        name: "FK_ActivityActuals_ActivityPlans_ActivityPlanId",
                        column: x => x.ActivityPlanId,
                        principalSchema: "planning",
                        principalTable: "ActivityPlans",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_ActivityActuals_Equipment_ActualEquipmentId",
                        column: x => x.ActualEquipmentId,
                        principalSchema: "planning",
                        principalTable: "Equipment",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ActivityActuals_Operators_ActualOperatorId",
                        column: x => x.ActualOperatorId,
                        principalSchema: "planning",
                        principalTable: "Operators",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ActivityActuals_Tractors_ActualTractorId",
                        column: x => x.ActualTractorId,
                        principalSchema: "planning",
                        principalTable: "Tractors",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "ActivityMaterialRequirements",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ActivityPlanId = table.Column<int>(type: "int", nullable: false),
                    MaterialId = table.Column<int>(type: "int", nullable: false),
                    ActivityMaterialStandardId = table.Column<int>(type: "int", nullable: true),
                    PlannedAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    StandardRatePerHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    NumberOfApplications = table.Column<int>(type: "int", nullable: false),
                    WastePercent = table.Column<decimal>(type: "decimal(9,4)", precision: 9, scale: 4, nullable: false),
                    BaseRequirement = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    WasteQuantity = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    TotalRequirement = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    Unit = table.Column<int>(type: "int", nullable: false),
                    RequiredDeliveryDate = table.Column<DateOnly>(type: "date", nullable: false),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ActivityMaterialRequirements", x => x.Id);
                    table.ForeignKey(
                        name: "FK_ActivityMaterialRequirements_ActivityPlans_ActivityPlanId",
                        column: x => x.ActivityPlanId,
                        principalSchema: "planning",
                        principalTable: "ActivityPlans",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_ActivityMaterialRequirements_Materials_MaterialId",
                        column: x => x.MaterialId,
                        principalSchema: "planning",
                        principalTable: "Materials",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "ResourceSchedules",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ActivityPlanId = table.Column<int>(type: "int", nullable: false),
                    BlockId = table.Column<int>(type: "int", nullable: false),
                    ActivityId = table.Column<int>(type: "int", nullable: false),
                    ScheduleDate = table.Column<DateOnly>(type: "date", nullable: false),
                    PlannedStart = table.Column<DateTime>(type: "datetime2", nullable: false),
                    PlannedEnd = table.Column<DateTime>(type: "datetime2", nullable: false),
                    TractorId = table.Column<int>(type: "int", nullable: true),
                    EquipmentId = table.Column<int>(type: "int", nullable: true),
                    OperatorId = table.Column<int>(type: "int", nullable: true),
                    WorkTeamId = table.Column<int>(type: "int", nullable: true),
                    PlannedAreaHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    DailyTargetHa = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ExpectedWorkingHours = table.Column<decimal>(type: "decimal(9,2)", precision: 9, scale: 2, nullable: false),
                    SupervisorName = table.Column<string>(type: "nvarchar(150)", maxLength: 150, nullable: true),
                    Status = table.Column<int>(type: "int", nullable: false),
                    Remarks = table.Column<string>(type: "nvarchar(1000)", maxLength: 1000, nullable: true),
                    DependencyOverrideApproved = table.Column<bool>(type: "bit", nullable: false),
                    DependencyOverrideBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    DependencyOverrideReason = table.Column<string>(type: "nvarchar(500)", maxLength: 500, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ResourceSchedules", x => x.Id);
                    table.CheckConstraint("CK_Schedule_Area", "[PlannedAreaHa] >= 0");
                    table.CheckConstraint("CK_Schedule_Time", "[PlannedEnd] > [PlannedStart]");
                    table.ForeignKey(
                        name: "FK_ResourceSchedules_ActivityPlans_ActivityPlanId",
                        column: x => x.ActivityPlanId,
                        principalSchema: "planning",
                        principalTable: "ActivityPlans",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_ResourceSchedules_Equipment_EquipmentId",
                        column: x => x.EquipmentId,
                        principalSchema: "planning",
                        principalTable: "Equipment",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ResourceSchedules_Operators_OperatorId",
                        column: x => x.OperatorId,
                        principalSchema: "planning",
                        principalTable: "Operators",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ResourceSchedules_PlantationBlocks_BlockId",
                        column: x => x.BlockId,
                        principalSchema: "planning",
                        principalTable: "PlantationBlocks",
                        principalColumn: "Id");
                    table.ForeignKey(
                        name: "FK_ResourceSchedules_PlantingActivities_ActivityId",
                        column: x => x.ActivityId,
                        principalSchema: "planning",
                        principalTable: "PlantingActivities",
                        principalColumn: "Id");
                    table.ForeignKey(
                        name: "FK_ResourceSchedules_Tractors_TractorId",
                        column: x => x.TractorId,
                        principalSchema: "planning",
                        principalTable: "Tractors",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                    table.ForeignKey(
                        name: "FK_ResourceSchedules_WorkTeams_WorkTeamId",
                        column: x => x.WorkTeamId,
                        principalSchema: "planning",
                        principalTable: "WorkTeams",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateTable(
                name: "ActualMaterialUsages",
                schema: "planning",
                columns: table => new
                {
                    Id = table.Column<int>(type: "int", nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    CompanyId = table.Column<int>(type: "int", nullable: false),
                    ActivityActualId = table.Column<int>(type: "int", nullable: false),
                    MaterialId = table.Column<int>(type: "int", nullable: false),
                    PlannedQuantity = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    ActualQuantity = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    Variance = table.Column<decimal>(type: "decimal(18,4)", precision: 18, scale: 4, nullable: false),
                    Unit = table.Column<int>(type: "int", nullable: false),
                    Remarks = table.Column<string>(type: "nvarchar(500)", maxLength: 500, nullable: true),
                    CreatedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: false),
                    CreatedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    ModifiedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    ModifiedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    IsDeleted = table.Column<bool>(type: "bit", nullable: false),
                    DeletedAtUtc = table.Column<DateTime>(type: "datetime2", nullable: true),
                    DeletedBy = table.Column<string>(type: "nvarchar(100)", maxLength: 100, nullable: true),
                    RowVersion = table.Column<byte[]>(type: "rowversion", rowVersion: true, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_ActualMaterialUsages", x => x.Id);
                    table.ForeignKey(
                        name: "FK_ActualMaterialUsages_ActivityActuals_ActivityActualId",
                        column: x => x.ActivityActualId,
                        principalSchema: "planning",
                        principalTable: "ActivityActuals",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Cascade);
                    table.ForeignKey(
                        name: "FK_ActualMaterialUsages_Materials_MaterialId",
                        column: x => x.MaterialId,
                        principalSchema: "planning",
                        principalTable: "Materials",
                        principalColumn: "Id",
                        onDelete: ReferentialAction.Restrict);
                });

            migrationBuilder.CreateIndex(
                name: "IX_ActivityActuals_ActivityPlanId",
                schema: "planning",
                table: "ActivityActuals",
                column: "ActivityPlanId",
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_ActivityActuals_ActualEquipmentId",
                schema: "planning",
                table: "ActivityActuals",
                column: "ActualEquipmentId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityActuals_ActualOperatorId",
                schema: "planning",
                table: "ActivityActuals",
                column: "ActualOperatorId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityActuals_ActualTractorId",
                schema: "planning",
                table: "ActivityActuals",
                column: "ActualTractorId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityActuals_IsDeleted",
                schema: "planning",
                table: "ActivityActuals",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityDependencies_ActivityId_PredecessorActivityId",
                schema: "planning",
                table: "ActivityDependencies",
                columns: new[] { "ActivityId", "PredecessorActivityId" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_ActivityDependencies_IsDeleted",
                schema: "planning",
                table: "ActivityDependencies",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityDependencies_PredecessorActivityId",
                schema: "planning",
                table: "ActivityDependencies",
                column: "PredecessorActivityId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityMaterialRequirements_ActivityPlanId_MaterialId",
                schema: "planning",
                table: "ActivityMaterialRequirements",
                columns: new[] { "ActivityPlanId", "MaterialId" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_ActivityMaterialRequirements_IsDeleted",
                schema: "planning",
                table: "ActivityMaterialRequirements",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityMaterialRequirements_MaterialId",
                schema: "planning",
                table: "ActivityMaterialRequirements",
                column: "MaterialId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityMaterialRequirements_RequiredDeliveryDate",
                schema: "planning",
                table: "ActivityMaterialRequirements",
                column: "RequiredDeliveryDate");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityMaterialStandards_ActivityId_MaterialId_CropType_EffectiveFrom",
                schema: "planning",
                table: "ActivityMaterialStandards",
                columns: new[] { "ActivityId", "MaterialId", "CropType", "EffectiveFrom" });

            migrationBuilder.CreateIndex(
                name: "IX_ActivityMaterialStandards_CaneVarietyId",
                schema: "planning",
                table: "ActivityMaterialStandards",
                column: "CaneVarietyId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityMaterialStandards_IsDeleted",
                schema: "planning",
                table: "ActivityMaterialStandards",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityMaterialStandards_MaterialId",
                schema: "planning",
                table: "ActivityMaterialStandards",
                column: "MaterialId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityPlans_ActivityId",
                schema: "planning",
                table: "ActivityPlans",
                column: "ActivityId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityPlans_BlockId",
                schema: "planning",
                table: "ActivityPlans",
                column: "BlockId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityPlans_IsDeleted",
                schema: "planning",
                table: "ActivityPlans",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityPlans_PlannedStartDate_PlannedEndDate",
                schema: "planning",
                table: "ActivityPlans",
                columns: new[] { "PlannedStartDate", "PlannedEndDate" });

            migrationBuilder.CreateIndex(
                name: "IX_ActivityPlans_ProjectionId_BlockId_ActivityId",
                schema: "planning",
                table: "ActivityPlans",
                columns: new[] { "ProjectionId", "BlockId", "ActivityId" });

            migrationBuilder.CreateIndex(
                name: "IX_ActivityPlans_ProjectionLineId",
                schema: "planning",
                table: "ActivityPlans",
                column: "ProjectionLineId");

            migrationBuilder.CreateIndex(
                name: "IX_ActivityPlans_Status",
                schema: "planning",
                table: "ActivityPlans",
                column: "Status");

            migrationBuilder.CreateIndex(
                name: "IX_ActualMaterialUsages_ActivityActualId_MaterialId",
                schema: "planning",
                table: "ActualMaterialUsages",
                columns: new[] { "ActivityActualId", "MaterialId" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_ActualMaterialUsages_IsDeleted",
                schema: "planning",
                table: "ActualMaterialUsages",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ActualMaterialUsages_MaterialId",
                schema: "planning",
                table: "ActualMaterialUsages",
                column: "MaterialId");

            migrationBuilder.CreateIndex(
                name: "IX_AuditLogs_IsDeleted",
                schema: "planning",
                table: "AuditLogs",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_AuditLogs_TableName_RecordId",
                schema: "planning",
                table: "AuditLogs",
                columns: new[] { "TableName", "RecordId" });

            migrationBuilder.CreateIndex(
                name: "IX_AuditLogs_TimestampUtc",
                schema: "planning",
                table: "AuditLogs",
                column: "TimestampUtc");

            migrationBuilder.CreateIndex(
                name: "IX_AuditLogs_UserName",
                schema: "planning",
                table: "AuditLogs",
                column: "UserName");

            migrationBuilder.CreateIndex(
                name: "IX_CaneVarieties_CompanyId_Code",
                schema: "planning",
                table: "CaneVarieties",
                columns: new[] { "CompanyId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_CaneVarieties_IsDeleted",
                schema: "planning",
                table: "CaneVarieties",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_Companies_Code",
                schema: "planning",
                table: "Companies",
                column: "Code",
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Companies_IsDeleted",
                schema: "planning",
                table: "Companies",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_Equipment_Category",
                schema: "planning",
                table: "Equipment",
                column: "Category");

            migrationBuilder.CreateIndex(
                name: "IX_Equipment_CompanyId_Code",
                schema: "planning",
                table: "Equipment",
                columns: new[] { "CompanyId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Equipment_CurrentFarmId",
                schema: "planning",
                table: "Equipment",
                column: "CurrentFarmId");

            migrationBuilder.CreateIndex(
                name: "IX_Equipment_EstateId",
                schema: "planning",
                table: "Equipment",
                column: "EstateId");

            migrationBuilder.CreateIndex(
                name: "IX_Equipment_IsDeleted",
                schema: "planning",
                table: "Equipment",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_Estates_CompanyId_Code",
                schema: "planning",
                table: "Estates",
                columns: new[] { "CompanyId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Estates_IsDeleted",
                schema: "planning",
                table: "Estates",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_Farms_CompanyId",
                schema: "planning",
                table: "Farms",
                column: "CompanyId");

            migrationBuilder.CreateIndex(
                name: "IX_Farms_EstateId_Code",
                schema: "planning",
                table: "Farms",
                columns: new[] { "EstateId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Farms_IsDeleted",
                schema: "planning",
                table: "Farms",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_GrowingSeasons_CompanyId_Code",
                schema: "planning",
                table: "GrowingSeasons",
                columns: new[] { "CompanyId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_GrowingSeasons_IsDeleted",
                schema: "planning",
                table: "GrowingSeasons",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_Materials_Category",
                schema: "planning",
                table: "Materials",
                column: "Category");

            migrationBuilder.CreateIndex(
                name: "IX_Materials_CompanyId_Code",
                schema: "planning",
                table: "Materials",
                columns: new[] { "CompanyId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Materials_IsDeleted",
                schema: "planning",
                table: "Materials",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_MaterialStocks_EstateId",
                schema: "planning",
                table: "MaterialStocks",
                column: "EstateId");

            migrationBuilder.CreateIndex(
                name: "IX_MaterialStocks_IsDeleted",
                schema: "planning",
                table: "MaterialStocks",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_MaterialStocks_MaterialId_EstateId",
                schema: "planning",
                table: "MaterialStocks",
                columns: new[] { "MaterialId", "EstateId" },
                unique: true,
                filter: "[EstateId] IS NOT NULL");

            migrationBuilder.CreateIndex(
                name: "IX_Operators_CompanyId_Code",
                schema: "planning",
                table: "Operators",
                columns: new[] { "CompanyId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Operators_FarmId",
                schema: "planning",
                table: "Operators",
                column: "FarmId");

            migrationBuilder.CreateIndex(
                name: "IX_Operators_IsDeleted",
                schema: "planning",
                table: "Operators",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_Operators_WorkTeamId",
                schema: "planning",
                table: "Operators",
                column: "WorkTeamId");

            migrationBuilder.CreateIndex(
                name: "IX_PlanningScenarios_IsDeleted",
                schema: "planning",
                table: "PlanningScenarios",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_PlanningScenarios_ProjectionId_Name",
                schema: "planning",
                table: "PlanningScenarios",
                columns: new[] { "ProjectionId", "Name" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_PlantationBlocks_CompanyId",
                schema: "planning",
                table: "PlantationBlocks",
                column: "CompanyId");

            migrationBuilder.CreateIndex(
                name: "IX_PlantationBlocks_IsDeleted",
                schema: "planning",
                table: "PlantationBlocks",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_PlantationBlocks_ZoneId_Code",
                schema: "planning",
                table: "PlantationBlocks",
                columns: new[] { "ZoneId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_PlantingActivities_CompanyId_Code",
                schema: "planning",
                table: "PlantingActivities",
                columns: new[] { "CompanyId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_PlantingActivities_IsDeleted",
                schema: "planning",
                table: "PlantingActivities",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_PlantingActivities_SequenceNo",
                schema: "planning",
                table: "PlantingActivities",
                column: "SequenceNo");

            migrationBuilder.CreateIndex(
                name: "IX_PlantingProjections_EstateId_ProjectionNo_Version",
                schema: "planning",
                table: "PlantingProjections",
                columns: new[] { "EstateId", "ProjectionNo", "Version" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_PlantingProjections_GrowingSeasonId_Status",
                schema: "planning",
                table: "PlantingProjections",
                columns: new[] { "GrowingSeasonId", "Status" });

            migrationBuilder.CreateIndex(
                name: "IX_PlantingProjections_IsDeleted",
                schema: "planning",
                table: "PlantingProjections",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_PlantingProjections_RevisedFromProjectionId",
                schema: "planning",
                table: "PlantingProjections",
                column: "RevisedFromProjectionId");

            migrationBuilder.CreateIndex(
                name: "IX_ProjectionApprovalHistory_IsDeleted",
                schema: "planning",
                table: "ProjectionApprovalHistory",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ProjectionApprovalHistory_ProjectionId_ActionAtUtc",
                schema: "planning",
                table: "ProjectionApprovalHistory",
                columns: new[] { "ProjectionId", "ActionAtUtc" });

            migrationBuilder.CreateIndex(
                name: "IX_ProjectionLines_BlockId_PlannedPlantingStart_PlannedPlantingEnd",
                schema: "planning",
                table: "ProjectionLines",
                columns: new[] { "BlockId", "PlannedPlantingStart", "PlannedPlantingEnd" });

            migrationBuilder.CreateIndex(
                name: "IX_ProjectionLines_CaneVarietyId",
                schema: "planning",
                table: "ProjectionLines",
                column: "CaneVarietyId");

            migrationBuilder.CreateIndex(
                name: "IX_ProjectionLines_FarmId",
                schema: "planning",
                table: "ProjectionLines",
                column: "FarmId");

            migrationBuilder.CreateIndex(
                name: "IX_ProjectionLines_IsDeleted",
                schema: "planning",
                table: "ProjectionLines",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ProjectionLines_ProjectionId",
                schema: "planning",
                table: "ProjectionLines",
                column: "ProjectionId");

            migrationBuilder.CreateIndex(
                name: "IX_ProjectionLines_ZoneId",
                schema: "planning",
                table: "ProjectionLines",
                column: "ZoneId");

            migrationBuilder.CreateIndex(
                name: "IX_ResourceSchedules_ActivityId",
                schema: "planning",
                table: "ResourceSchedules",
                column: "ActivityId");

            migrationBuilder.CreateIndex(
                name: "IX_ResourceSchedules_ActivityPlanId",
                schema: "planning",
                table: "ResourceSchedules",
                column: "ActivityPlanId");

            migrationBuilder.CreateIndex(
                name: "IX_ResourceSchedules_BlockId",
                schema: "planning",
                table: "ResourceSchedules",
                column: "BlockId");

            migrationBuilder.CreateIndex(
                name: "IX_ResourceSchedules_EquipmentId_PlannedStart_PlannedEnd",
                schema: "planning",
                table: "ResourceSchedules",
                columns: new[] { "EquipmentId", "PlannedStart", "PlannedEnd" });

            migrationBuilder.CreateIndex(
                name: "IX_ResourceSchedules_IsDeleted",
                schema: "planning",
                table: "ResourceSchedules",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ResourceSchedules_OperatorId_PlannedStart_PlannedEnd",
                schema: "planning",
                table: "ResourceSchedules",
                columns: new[] { "OperatorId", "PlannedStart", "PlannedEnd" });

            migrationBuilder.CreateIndex(
                name: "IX_ResourceSchedules_ScheduleDate",
                schema: "planning",
                table: "ResourceSchedules",
                column: "ScheduleDate");

            migrationBuilder.CreateIndex(
                name: "IX_ResourceSchedules_TractorId_PlannedStart_PlannedEnd",
                schema: "planning",
                table: "ResourceSchedules",
                columns: new[] { "TractorId", "PlannedStart", "PlannedEnd" });

            migrationBuilder.CreateIndex(
                name: "IX_ResourceSchedules_WorkTeamId",
                schema: "planning",
                table: "ResourceSchedules",
                column: "WorkTeamId");

            migrationBuilder.CreateIndex(
                name: "IX_RoleClaims_RoleId",
                schema: "planning",
                table: "RoleClaims",
                column: "RoleId");

            migrationBuilder.CreateIndex(
                name: "RoleNameIndex",
                schema: "planning",
                table: "Roles",
                column: "NormalizedName",
                unique: true,
                filter: "[NormalizedName] IS NOT NULL");

            migrationBuilder.CreateIndex(
                name: "IX_ScenarioAdjustments_ActivityId",
                schema: "planning",
                table: "ScenarioAdjustments",
                column: "ActivityId");

            migrationBuilder.CreateIndex(
                name: "IX_ScenarioAdjustments_IsDeleted",
                schema: "planning",
                table: "ScenarioAdjustments",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_ScenarioAdjustments_ScenarioId",
                schema: "planning",
                table: "ScenarioAdjustments",
                column: "ScenarioId");

            migrationBuilder.CreateIndex(
                name: "IX_TractorEquipmentCompatibility_EquipmentId",
                schema: "planning",
                table: "TractorEquipmentCompatibility",
                column: "EquipmentId");

            migrationBuilder.CreateIndex(
                name: "IX_TractorEquipmentCompatibility_IsDeleted",
                schema: "planning",
                table: "TractorEquipmentCompatibility",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_TractorEquipmentCompatibility_TractorId_EquipmentId",
                schema: "planning",
                table: "TractorEquipmentCompatibility",
                columns: new[] { "TractorId", "EquipmentId" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Tractors_Availability",
                schema: "planning",
                table: "Tractors",
                column: "Availability");

            migrationBuilder.CreateIndex(
                name: "IX_Tractors_CompanyId_AssetNo",
                schema: "planning",
                table: "Tractors",
                columns: new[] { "CompanyId", "AssetNo" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Tractors_CompanyId_Code",
                schema: "planning",
                table: "Tractors",
                columns: new[] { "CompanyId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Tractors_CurrentFarmId",
                schema: "planning",
                table: "Tractors",
                column: "CurrentFarmId");

            migrationBuilder.CreateIndex(
                name: "IX_Tractors_EstateId",
                schema: "planning",
                table: "Tractors",
                column: "EstateId");

            migrationBuilder.CreateIndex(
                name: "IX_Tractors_IsDeleted",
                schema: "planning",
                table: "Tractors",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_UserClaims_UserId",
                schema: "planning",
                table: "UserClaims",
                column: "UserId");

            migrationBuilder.CreateIndex(
                name: "IX_UserLogins_UserId",
                schema: "planning",
                table: "UserLogins",
                column: "UserId");

            migrationBuilder.CreateIndex(
                name: "IX_UserRoles_RoleId",
                schema: "planning",
                table: "UserRoles",
                column: "RoleId");

            migrationBuilder.CreateIndex(
                name: "EmailIndex",
                schema: "planning",
                table: "Users",
                column: "NormalizedEmail");

            migrationBuilder.CreateIndex(
                name: "UserNameIndex",
                schema: "planning",
                table: "Users",
                column: "NormalizedUserName",
                unique: true,
                filter: "[NormalizedUserName] IS NOT NULL");

            migrationBuilder.CreateIndex(
                name: "IX_WorkTeams_CompanyId_Code",
                schema: "planning",
                table: "WorkTeams",
                columns: new[] { "CompanyId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_WorkTeams_FarmId",
                schema: "planning",
                table: "WorkTeams",
                column: "FarmId");

            migrationBuilder.CreateIndex(
                name: "IX_WorkTeams_IsDeleted",
                schema: "planning",
                table: "WorkTeams",
                column: "IsDeleted");

            migrationBuilder.CreateIndex(
                name: "IX_Zones_CompanyId",
                schema: "planning",
                table: "Zones",
                column: "CompanyId");

            migrationBuilder.CreateIndex(
                name: "IX_Zones_FarmId_Code",
                schema: "planning",
                table: "Zones",
                columns: new[] { "FarmId", "Code" },
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Zones_IsDeleted",
                schema: "planning",
                table: "Zones",
                column: "IsDeleted");
        }

        /// <inheritdoc />
        protected override void Down(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.DropTable(
                name: "ActivityDependencies",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "ActivityMaterialRequirements",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "ActivityMaterialStandards",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "ActualMaterialUsages",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "AuditLogs",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "MaterialStocks",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "ProjectionApprovalHistory",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "ResourceSchedules",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "RoleClaims",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "ScenarioAdjustments",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "TractorEquipmentCompatibility",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "UserClaims",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "UserLogins",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "UserRoles",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "UserTokens",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "ActivityActuals",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Materials",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "PlanningScenarios",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Roles",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Users",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "ActivityPlans",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Equipment",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Operators",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Tractors",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "PlantingActivities",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "ProjectionLines",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "WorkTeams",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "CaneVarieties",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "PlantationBlocks",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "PlantingProjections",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Zones",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "GrowingSeasons",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Farms",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Estates",
                schema: "planning");

            migrationBuilder.DropTable(
                name: "Companies",
                schema: "planning");
        }
    }
}
