# Data model

All tables live in the `planning` schema of the `SugarcanePlanning` database and share the
same audit and concurrency columns.

## Common columns

| Column | Type | Purpose |
|--------|------|---------|
| `Id` | `int identity` | primary key |
| `CompanyId` | `int` | tenancy; global query filter (except `AuditLogs`, where it is nullable) |
| `CreatedAtUtc` / `CreatedBy` | `datetime2` / `nvarchar(100)` | insert stamp |
| `ModifiedAtUtc` / `ModifiedBy` | `datetime2` / `nvarchar(100)` | update stamp |
| `IsDeleted` / `DeletedAtUtc` / `DeletedBy` | `bit` / … | soft deletion |
| `RowVersion` | `rowversion` | optimistic concurrency |

## Entity-relationship overview

```
Company ─┬─< Estate ─< Farm ─< Zone ─< PlantationBlock
         ├─< GrowingSeason
         └─< CaneVariety

GrowingSeason ─< PlantingProjection ─┬─< ProjectionLine >── PlantationBlock
                                     │                  >── CaneVariety
                                     ├─< ProjectionApprovalHistory
                                     ├─< ActivityPlan ─┬─< ResourceSchedule
                                     │                 ├─< ActivityMaterialRequirement >── Material
                                     │                 └─< ActivityActual ─< ActualMaterialUsage
                                     └─< PlanningScenario ─< ScenarioAdjustment

PlantingActivity ─┬─< ActivityDependency (self-referencing predecessor graph)
                  ├─< ActivityMaterialStandard >── Material, CaneVariety
                  └─< ActivityPlan

Tractor ─< TractorEquipmentCompatibility >─ Equipment
Tractor, Equipment, Operator, WorkTeam ─< ResourceSchedule
Material ─< MaterialStock >── Estate
```

## Tables

### Enterprise and land structure (section 3)

| Table | Key columns | Constraints |
|-------|-------------|-------------|
| `Companies` | `Code` unique, `Name`, `BaseCurrency`, `IsActive` | |
| `Estates` | `(CompanyId, Code)` unique, `Location`, `TotalAreaHa` | `TotalAreaHa >= 0` |
| `Farms` | `(EstateId, Code)` unique, `ManagerName`, `TotalAreaHa` | `TotalAreaHa >= 0` |
| `Zones` | `(FarmId, Code)` unique, `SupervisorName`, `TotalAreaHa` | `TotalAreaHa >= 0` |
| `PlantationBlocks` | `(ZoneId, Code)` unique, `TotalAreaHa`, `PlantableAreaHa`, `Latitude`, `Longitude`, `MapReference`, `SoilType`, `LandCondition`, `Irrigation`, `CurrentCropStatus` | `PlantableAreaHa <= TotalAreaHa` |

### Season and variety (section 4)

| Table | Key columns | Constraints |
|-------|-------------|-------------|
| `GrowingSeasons` | `(CompanyId, Code)` unique, planting and harvest windows, `Status` | `EndDate >= StartDate` |
| `CaneVarieties` | `(CompanyId, Code)` unique, `SeedRatePerHa`, `GrowingPeriodMonths`, `ExpectedYieldPerHa`, `ExpectedLossPercent`, recommended planting months | loss 0–100, months 1–12 |

### Activities (section 6)

| Table | Key columns | Constraints |
|-------|-------------|-------------|
| `PlantingActivities` | `(CompanyId, Code)` unique, `SequenceNo`, `Category`, `ApplicableCropType`, `StandardStartDayOffset`, capacities, `StandardDurationPerHa`, `StandardLaborDaysPerHa`, the four *Requires* flags, `AllowOverlap` | `SequenceNo > 0` |
| `ActivityDependencies` | `(ActivityId, PredecessorActivityId)` unique, `LagDays`, `IsBlocking` | `ActivityId <> PredecessorActivityId`; cycles rejected in the service |

### Projections (sections 5 and 16)

| Table | Key columns | Constraints |
|-------|-------------|-------------|
| `PlantingProjections` | `(EstateId, ProjectionNo, Version)` unique, `Status`, prepared/submitted/reviewed/approved/rejected by and at, `RevisedFromProjectionId`, `RevisionReason`, `IsReadOnly`, `IsCurrentVersion`, totals | `Version > 0`, `PlanningEndDate >= PlanningStartDate` |
| `ProjectionLines` | `BlockId`, `CaneVarietyId`, `CropType`, `AvailableAreaHa`, `ProjectedPlantingAreaHa`, planting window, yield, loss, derived `HarvestableAreaHa` and `ExpectedCaneProductionTons`, `Priority` | area > 0, loss 0–100, dates ordered, priority 1–9 |
| `ProjectionApprovalHistory` | `Action`, `FromStatus`, `ToStatus`, `ActionBy`, `ActionAtUtc`, `Comments` | |

### Activity plans (section 7)

`ActivityPlans` — `ProjectionId`, `ProjectionLineId`, `FarmId`, `ZoneId`, `BlockId`,
`ActivityId`, `SequenceNo`, `PlannedAreaHa`, planned start/end, `WorkingDays`,
`DailyTargetHa`, `PlannedWorkingHours`, required tractor type and counts, `RequiredLaborDays`,
`RequiredWorkers`, `PlannedFuelLiters`, `SupervisorName`, `Status`.
Indexed on `(ProjectionId, BlockId, ActivityId)`, on the date range and on `Status`.

### Machinery and workforce (sections 8, 9, 14)

| Table | Key columns | Constraints |
|-------|-------------|-------------|
| `Tractors` | `(CompanyId, Code)` and `(CompanyId, AssetNo)` unique, `Horsepower`, `DailyCapacityHa`, fuel per hour and per hectare, `Availability`, `Maintenance` + window | `Horsepower > 0`, `DailyCapacityHa >= 0` |
| `Equipment` | `(CompanyId, Code)` unique, `Category`, `MinimumTractorHp`, capacities, availability | `MinimumTractorHp >= 0` |
| `TractorEquipmentCompatibility` | `(TractorId, EquipmentId)` unique, `IsRecommended` | horsepower validated on insert |
| `Operators` | `(CompanyId, Code)` unique, `Skill`, licence, `WorkTeamId`, `StandardHoursPerDay` | hours > 0 |
| `WorkTeams` | `(CompanyId, Code)` unique, `PrimarySkill`, `MemberCount` | `MemberCount >= 0` |

### Scheduling (section 10)

`ResourceSchedules` — `ActivityPlanId`, `BlockId`, `ActivityId`, `ScheduleDate`,
`PlannedStart`/`PlannedEnd`, optional `TractorId`, `EquipmentId`, `OperatorId`, `WorkTeamId`,
`PlannedAreaHa`, `DailyTargetHa`, `ExpectedWorkingHours`, `Status`, and the dependency-override
trio. Check constraint `PlannedEnd > PlannedStart`. Indexed on
`(TractorId, PlannedStart, PlannedEnd)` and the same for equipment and operator, so each
double-booking check is an index seek.

### Materials (sections 11–13)

| Table | Key columns | Constraints |
|-------|-------------|-------------|
| `Materials` | `(CompanyId, Code)` unique, `Category`, `BaseUnit`, `AlternativeUnit`, `UnitConversionFactor`, standard and min/max rates | factor > 0, rates >= 0 |
| `ActivityMaterialStandards` | `(ActivityId, MaterialId, CropType, EffectiveFrom)` indexed, optional `CaneVarietyId` and `SoilType`, `StandardRatePerHa`, `NumberOfApplications`, `WastePercent`, effective dates | rate > 0, applications >= 1, waste 0–100 |
| `ActivityMaterialRequirements` | `(ActivityPlanId, MaterialId)` unique, base / waste / total quantities, `RequiredDeliveryDate` | regenerated from the plan |
| `MaterialStocks` | `(MaterialId, EstateId)` unique, available / reserved / incoming, `LastSyncedUtc`, `SourceSystem` | quantities >= 0; written only by the interface |

### Execution (section 17)

| Table | Key columns |
|-------|-------------|
| `ActivityActuals` | `ActivityPlanId` unique, actual dates, area, machine and operator used, hours, fuel, labor-days, derived `AreaVariance`, `FuelVariance`, `ScheduleVarianceDays`, `CompletionPercent`, `DelayReason` |
| `ActualMaterialUsages` | `(ActivityActualId, MaterialId)` unique, planned, actual, `Variance` |

### Scenarios and audit (sections 15, 22)

| Table | Key columns |
|-------|-------------|
| `PlanningScenarios` | `(ProjectionId, Name)` unique, `Status`, `SimulatedAtUtc`, `ResultJson` |
| `ScenarioAdjustments` | `AdjustmentType`, `Value`, optional `ActivityId` |
| `AuditLogs` | `UserName`, `TimestampUtc`, `Action`, `TableName`, `RecordId`, `OldValues`, `NewValues`, `ChangedColumns`, `IpAddress`, `DeviceInfo`; indexed on timestamp, `(TableName, RecordId)` and user |

### Identity

`Users`, `Roles`, `UserRoles`, `UserClaims`, `UserLogins`, `UserTokens`, `RoleClaims` —
ASP.NET Core Identity, with `Users.CompanyId` and `Users.FullName` added so the JWT can carry
the tenant.

## Generating the DDL

```bash
dotnet ef migrations script -p src/SugarcanePlanning.Infrastructure \
                            -s src/SugarcanePlanning.Api \
                            -o database/01_schema.sql --idempotent
```

The checked-in `database/01_schema.sql` is that output: 36 tables with their primary keys,
foreign keys, unique indexes, check constraints and performance indexes.
