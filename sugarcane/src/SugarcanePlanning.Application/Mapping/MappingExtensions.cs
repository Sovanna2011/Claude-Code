using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Auditing;
using SugarcanePlanning.Contracts.Capacity;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Execution;
using SugarcanePlanning.Contracts.Machinery;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Contracts.Organization;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Contracts.Scheduling;
using SugarcanePlanning.Contracts.Seasons;
using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Entities;

namespace SugarcanePlanning.Application.Mapping;

/// <summary>
/// Hand-written entity ↔ DTO mapping. Explicit code keeps the projections translatable by
/// EF Core and makes every mapped field visible at review time.
/// </summary>
public static class MappingExtensions
{
    // ------------------------------------------------------------- organization

    public static CompanyDto ToDto(this Company e) => new()
    {
        Id = e.Id,
        Code = e.Code,
        Name = e.Name,
        Address = e.Address,
        BaseCurrency = e.BaseCurrency,
        IsActive = e.IsActive,
        EstateCount = e.Estates?.Count ?? 0,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this CompanyUpsertDto dto, Company e)
    {
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.Address = dto.Address;
        e.BaseCurrency = dto.BaseCurrency.ToUpperInvariant();
        e.IsActive = dto.IsActive;
    }

    public static EstateDto ToDto(this Estate e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        CompanyName = e.Company?.Name ?? string.Empty,
        Code = e.Code,
        Name = e.Name,
        Location = e.Location,
        TotalAreaHa = e.TotalAreaHa,
        IsActive = e.IsActive,
        FarmCount = e.Farms?.Count ?? 0,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this EstateUpsertDto dto, Estate e)
    {
        e.CompanyId = dto.CompanyId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.Location = dto.Location;
        e.TotalAreaHa = dto.TotalAreaHa;
        e.IsActive = dto.IsActive;
    }

    public static FarmDto ToDto(this Farm e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        EstateId = e.EstateId,
        EstateName = e.Estate?.Name ?? string.Empty,
        Code = e.Code,
        Name = e.Name,
        ManagerName = e.ManagerName,
        TotalAreaHa = e.TotalAreaHa,
        IsActive = e.IsActive,
        ZoneCount = e.Zones?.Count ?? 0,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this FarmUpsertDto dto, Farm e)
    {
        e.EstateId = dto.EstateId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.ManagerName = dto.ManagerName;
        e.TotalAreaHa = dto.TotalAreaHa;
        e.IsActive = dto.IsActive;
    }

    public static ZoneDto ToDto(this Zone e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        FarmId = e.FarmId,
        FarmName = e.Farm?.Name ?? string.Empty,
        Code = e.Code,
        Name = e.Name,
        SupervisorName = e.SupervisorName,
        TotalAreaHa = e.TotalAreaHa,
        IsActive = e.IsActive,
        BlockCount = e.Blocks?.Count ?? 0,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this ZoneUpsertDto dto, Zone e)
    {
        e.FarmId = dto.FarmId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.SupervisorName = dto.SupervisorName;
        e.TotalAreaHa = dto.TotalAreaHa;
        e.IsActive = dto.IsActive;
    }

    public static PlantationBlockDto ToDto(this PlantationBlock e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        ZoneId = e.ZoneId,
        ZoneName = e.Zone?.Name ?? string.Empty,
        FarmName = e.Zone?.Farm?.Name ?? string.Empty,
        EstateName = e.Zone?.Farm?.Estate?.Name ?? string.Empty,
        Code = e.Code,
        Name = e.Name,
        TotalAreaHa = e.TotalAreaHa,
        PlantableAreaHa = e.PlantableAreaHa,
        Latitude = e.Latitude,
        Longitude = e.Longitude,
        MapReference = e.MapReference,
        SoilType = e.SoilType,
        LandCondition = e.LandCondition,
        Irrigation = e.Irrigation,
        CurrentCropStatus = e.CurrentCropStatus,
        IsActive = e.IsActive,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this PlantationBlockUpsertDto dto, PlantationBlock e)
    {
        e.ZoneId = dto.ZoneId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.TotalAreaHa = dto.TotalAreaHa;
        e.PlantableAreaHa = dto.PlantableAreaHa;
        e.Latitude = dto.Latitude;
        e.Longitude = dto.Longitude;
        e.MapReference = dto.MapReference;
        e.SoilType = dto.SoilType;
        e.LandCondition = dto.LandCondition;
        e.Irrigation = dto.Irrigation;
        e.CurrentCropStatus = dto.CurrentCropStatus;
        e.IsActive = dto.IsActive;
    }

    // ------------------------------------------------------------------ seasons

    public static GrowingSeasonDto ToDto(this GrowingSeason e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        Code = e.Code,
        Name = e.Name,
        StartDate = e.StartDate,
        EndDate = e.EndDate,
        PlannedPlantingStart = e.PlannedPlantingStart,
        PlannedPlantingEnd = e.PlannedPlantingEnd,
        ExpectedHarvestStart = e.ExpectedHarvestStart,
        ExpectedHarvestEnd = e.ExpectedHarvestEnd,
        Status = e.Status,
        Remarks = e.Remarks,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this GrowingSeasonUpsertDto dto, GrowingSeason e)
    {
        e.CompanyId = dto.CompanyId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.StartDate = dto.StartDate;
        e.EndDate = dto.EndDate;
        e.PlannedPlantingStart = dto.PlannedPlantingStart;
        e.PlannedPlantingEnd = dto.PlannedPlantingEnd;
        e.ExpectedHarvestStart = dto.ExpectedHarvestStart;
        e.ExpectedHarvestEnd = dto.ExpectedHarvestEnd;
        e.Status = dto.Status;
        e.Remarks = dto.Remarks;
    }

    public static CaneVarietyDto ToDto(this CaneVariety e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        Code = e.Code,
        Name = e.Name,
        RecommendedSoilType = e.RecommendedSoilType,
        SeedRatePerHa = e.SeedRatePerHa,
        GrowingPeriodMonths = e.GrowingPeriodMonths,
        ExpectedYieldPerHa = e.ExpectedYieldPerHa,
        ExpectedLossPercent = e.ExpectedLossPercent,
        RecommendedPlantingStartMonth = e.RecommendedPlantingStartMonth,
        RecommendedPlantingEndMonth = e.RecommendedPlantingEndMonth,
        IsActive = e.IsActive,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this CaneVarietyUpsertDto dto, CaneVariety e)
    {
        e.CompanyId = dto.CompanyId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.RecommendedSoilType = dto.RecommendedSoilType;
        e.SeedRatePerHa = dto.SeedRatePerHa;
        e.GrowingPeriodMonths = dto.GrowingPeriodMonths;
        e.ExpectedYieldPerHa = dto.ExpectedYieldPerHa;
        e.ExpectedLossPercent = dto.ExpectedLossPercent;
        e.RecommendedPlantingStartMonth = dto.RecommendedPlantingStartMonth;
        e.RecommendedPlantingEndMonth = dto.RecommendedPlantingEndMonth;
        e.IsActive = dto.IsActive;
    }

    // --------------------------------------------------------------- activities

    public static PlantingActivityDto ToDto(this PlantingActivity e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        Code = e.Code,
        Name = e.Name,
        Category = e.Category,
        SequenceNo = e.SequenceNo,
        ApplicableCropType = e.ApplicableCropType,
        StandardStartDayOffset = e.StandardStartDayOffset,
        StandardCapacityPerHour = e.StandardCapacityPerHour,
        StandardCapacityPerDay = e.StandardCapacityPerDay,
        StandardDurationPerHa = e.StandardDurationPerHa,
        StandardLaborDaysPerHa = e.StandardLaborDaysPerHa,
        IsMandatory = e.IsMandatory,
        RequiresTractor = e.RequiresTractor,
        RequiresEquipment = e.RequiresEquipment,
        RequiresMaterial = e.RequiresMaterial,
        RequiresLabor = e.RequiresLabor,
        AllowOverlap = e.AllowOverlap,
        DefaultEquipmentCategory = e.DefaultEquipmentCategory,
        IsActive = e.IsActive,
        Predecessors = e.Predecessors?.Select(d => d.ToDto()).ToList() ?? new List<ActivityDependencyDto>(),
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this PlantingActivityUpsertDto dto, PlantingActivity e)
    {
        e.CompanyId = dto.CompanyId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.Category = dto.Category;
        e.SequenceNo = dto.SequenceNo;
        e.ApplicableCropType = dto.ApplicableCropType;
        e.StandardStartDayOffset = dto.StandardStartDayOffset;
        e.StandardCapacityPerHour = dto.StandardCapacityPerHour;
        e.StandardCapacityPerDay = dto.StandardCapacityPerDay;
        e.StandardDurationPerHa = dto.StandardDurationPerHa;
        e.StandardLaborDaysPerHa = dto.StandardLaborDaysPerHa;
        e.IsMandatory = dto.IsMandatory;
        e.RequiresTractor = dto.RequiresTractor;
        e.RequiresEquipment = dto.RequiresEquipment;
        e.RequiresMaterial = dto.RequiresMaterial;
        e.RequiresLabor = dto.RequiresLabor;
        e.AllowOverlap = dto.AllowOverlap;
        e.DefaultEquipmentCategory = dto.DefaultEquipmentCategory;
        e.IsActive = dto.IsActive;
    }

    public static ActivityDependencyDto ToDto(this ActivityDependency e) => new()
    {
        Id = e.Id,
        ActivityId = e.ActivityId,
        ActivityName = e.Activity?.Name ?? string.Empty,
        PredecessorActivityId = e.PredecessorActivityId,
        PredecessorActivityName = e.PredecessorActivity?.Name ?? string.Empty,
        LagDays = e.LagDays,
        IsBlocking = e.IsBlocking,
        Remarks = e.Remarks
    };

    public static ActivityPlanDto ToDto(this ActivityPlan e, decimal completionPercent = 0m) => new()
    {
        Id = e.Id,
        ProjectionId = e.ProjectionId,
        ProjectionNo = e.Projection?.ProjectionNo ?? string.Empty,
        ProjectionLineId = e.ProjectionLineId,
        FarmId = e.FarmId,
        FarmName = e.Block?.Zone?.Farm?.Name ?? string.Empty,
        ZoneId = e.ZoneId,
        ZoneName = e.Block?.Zone?.Name ?? string.Empty,
        BlockId = e.BlockId,
        BlockName = e.Block is null ? string.Empty : $"{e.Block.Code} — {e.Block.Name}",
        ActivityId = e.ActivityId,
        ActivityCode = e.Activity?.Code ?? string.Empty,
        ActivityName = e.Activity?.Name ?? string.Empty,
        SequenceNo = e.SequenceNo,
        PlannedAreaHa = e.PlannedAreaHa,
        PlannedStartDate = e.PlannedStartDate,
        PlannedEndDate = e.PlannedEndDate,
        WorkingDays = e.WorkingDays,
        DailyTargetHa = e.DailyTargetHa,
        PlannedWorkingHours = e.PlannedWorkingHours,
        RequiredTractorType = e.RequiredTractorType,
        RequiredEquipmentCategory = e.RequiredEquipmentCategory,
        RequiredTractorCount = e.RequiredTractorCount,
        RequiredEquipmentCount = e.RequiredEquipmentCount,
        RequiredLaborDays = e.RequiredLaborDays,
        RequiredWorkers = e.RequiredWorkers,
        PlannedFuelLiters = e.PlannedFuelLiters,
        SupervisorName = e.SupervisorName,
        Status = e.Status,
        Remarks = e.Remarks,
        CompletionPercent = completionPercent,
        RowVersion = e.RowVersion
    };

    // -------------------------------------------------------------- projections

    public static ProjectionSummaryDto ToSummaryDto(this PlantingProjection e) => new()
    {
        Id = e.Id,
        ProjectionNo = e.ProjectionNo,
        Version = e.Version,
        CompanyId = e.CompanyId,
        EstateId = e.EstateId,
        EstateName = e.Estate?.Name ?? string.Empty,
        GrowingSeasonId = e.GrowingSeasonId,
        SeasonName = e.GrowingSeason?.Name ?? string.Empty,
        ProjectionDate = e.ProjectionDate,
        PlanningStartDate = e.PlanningStartDate,
        PlanningEndDate = e.PlanningEndDate,
        TotalProjectedAreaHa = e.TotalProjectedAreaHa,
        TotalExpectedProductionTons = e.TotalExpectedProductionTons,
        Status = e.Status,
        IsCurrentVersion = e.IsCurrentVersion,
        IsReadOnly = e.IsReadOnly,
        PreparedBy = e.PreparedBy,
        ApprovedBy = e.ApprovedBy,
        ApprovedAtUtc = e.ApprovedAtUtc,
        LineCount = e.Lines?.Count ?? 0
    };

    public static ProjectionDetailDto ToDetailDto(this PlantingProjection e)
    {
        var dto = new ProjectionDetailDto
        {
            Id = e.Id,
            ProjectionNo = e.ProjectionNo,
            Version = e.Version,
            CompanyId = e.CompanyId,
            EstateId = e.EstateId,
            EstateName = e.Estate?.Name ?? string.Empty,
            GrowingSeasonId = e.GrowingSeasonId,
            SeasonName = e.GrowingSeason?.Name ?? string.Empty,
            ProjectionDate = e.ProjectionDate,
            PlanningStartDate = e.PlanningStartDate,
            PlanningEndDate = e.PlanningEndDate,
            TotalProjectedAreaHa = e.TotalProjectedAreaHa,
            TotalExpectedProductionTons = e.TotalExpectedProductionTons,
            Status = e.Status,
            IsCurrentVersion = e.IsCurrentVersion,
            IsReadOnly = e.IsReadOnly,
            PreparedBy = e.PreparedBy,
            SubmittedBy = e.SubmittedBy,
            SubmittedAtUtc = e.SubmittedAtUtc,
            ReviewedBy = e.ReviewedBy,
            ReviewedAtUtc = e.ReviewedAtUtc,
            ApprovedBy = e.ApprovedBy,
            ApprovedAtUtc = e.ApprovedAtUtc,
            RejectedBy = e.RejectedBy,
            RejectedAtUtc = e.RejectedAtUtc,
            RejectionReason = e.RejectionReason,
            RevisedFromProjectionId = e.RevisedFromProjectionId,
            RevisionReason = e.RevisionReason,
            Remarks = e.Remarks,
            LineCount = e.Lines?.Count ?? 0,
            RowVersion = e.RowVersion
        };
        if (e.Lines is not null)
            dto.Lines = e.Lines.OrderBy(l => l.Priority).ThenBy(l => l.Id).Select(l => l.ToDto()).ToList();
        if (e.ApprovalHistory is not null)
            dto.ApprovalHistory = e.ApprovalHistory.OrderByDescending(h => h.ActionAtUtc).Select(h => h.ToDto()).ToList();
        return dto;
    }

    public static ProjectionLineDto ToDto(this ProjectionLine e) => new()
    {
        Id = e.Id,
        ProjectionId = e.ProjectionId,
        FarmId = e.FarmId,
        FarmName = e.Farm?.Name ?? e.Block?.Zone?.Farm?.Name ?? string.Empty,
        ZoneId = e.ZoneId,
        ZoneName = e.Zone?.Name ?? e.Block?.Zone?.Name ?? string.Empty,
        BlockId = e.BlockId,
        BlockCode = e.Block?.Code ?? string.Empty,
        BlockName = e.Block?.Name ?? string.Empty,
        CropType = e.CropType,
        CaneVarietyId = e.CaneVarietyId,
        VarietyName = e.CaneVariety?.Name ?? string.Empty,
        AvailableAreaHa = e.AvailableAreaHa,
        ProjectedPlantingAreaHa = e.ProjectedPlantingAreaHa,
        PlannedPlantingStart = e.PlannedPlantingStart,
        PlannedPlantingEnd = e.PlannedPlantingEnd,
        ExpectedHarvestDate = e.ExpectedHarvestDate,
        ExpectedYieldPerHa = e.ExpectedYieldPerHa,
        ExpectedLossPercent = e.ExpectedLossPercent,
        HarvestableAreaHa = e.HarvestableAreaHa,
        ExpectedCaneProductionTons = e.ExpectedCaneProductionTons,
        Priority = e.Priority,
        Remarks = e.Remarks
    };

    public static ApprovalHistoryDto ToDto(this ProjectionApprovalHistory e) => new()
    {
        Id = e.Id,
        Action = e.Action,
        FromStatus = e.FromStatus,
        ToStatus = e.ToStatus,
        ActionBy = e.ActionBy,
        ActionAtUtc = e.ActionAtUtc,
        Comments = e.Comments
    };

    // ---------------------------------------------------------------- machinery

    public static TractorDto ToDto(this Tractor e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        EstateId = e.EstateId,
        EstateName = e.Estate?.Name,
        AssetNo = e.AssetNo,
        Code = e.Code,
        RegistrationNo = e.RegistrationNo,
        Brand = e.Brand,
        Model = e.Model,
        Horsepower = e.Horsepower,
        CurrentFarmId = e.CurrentFarmId,
        CurrentFarmName = e.CurrentFarm?.Name,
        Ownership = e.Ownership,
        DailyCapacityHa = e.DailyCapacityHa,
        FuelConsumptionPerHour = e.FuelConsumptionPerHour,
        FuelConsumptionPerHa = e.FuelConsumptionPerHa,
        Availability = e.Availability,
        Maintenance = e.Maintenance,
        MaintenanceFromDate = e.MaintenanceFromDate,
        MaintenanceToDate = e.MaintenanceToDate,
        IsActive = e.IsActive,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this TractorUpsertDto dto, Tractor e)
    {
        e.CompanyId = dto.CompanyId;
        e.EstateId = dto.EstateId;
        e.AssetNo = dto.AssetNo.Trim();
        e.Code = dto.Code.Trim();
        e.RegistrationNo = dto.RegistrationNo;
        e.Brand = dto.Brand;
        e.Model = dto.Model;
        e.Horsepower = dto.Horsepower;
        e.CurrentFarmId = dto.CurrentFarmId;
        e.Ownership = dto.Ownership;
        e.DailyCapacityHa = dto.DailyCapacityHa;
        e.FuelConsumptionPerHour = dto.FuelConsumptionPerHour;
        e.FuelConsumptionPerHa = dto.FuelConsumptionPerHa;
        e.Availability = dto.Availability;
        e.Maintenance = dto.Maintenance;
        e.MaintenanceFromDate = dto.MaintenanceFromDate;
        e.MaintenanceToDate = dto.MaintenanceToDate;
        e.IsActive = dto.IsActive;
    }

    public static EquipmentDto ToDto(this EquipmentItem e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        EstateId = e.EstateId,
        EstateName = e.Estate?.Name,
        Code = e.Code,
        Name = e.Name,
        Category = e.Category,
        MinimumTractorHp = e.MinimumTractorHp,
        CapacityPerHour = e.CapacityPerHour,
        CapacityPerDay = e.CapacityPerDay,
        CurrentFarmId = e.CurrentFarmId,
        CurrentFarmName = e.CurrentFarm?.Name,
        Ownership = e.Ownership,
        Availability = e.Availability,
        Maintenance = e.Maintenance,
        MaintenanceFromDate = e.MaintenanceFromDate,
        MaintenanceToDate = e.MaintenanceToDate,
        IsActive = e.IsActive,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this EquipmentUpsertDto dto, EquipmentItem e)
    {
        e.CompanyId = dto.CompanyId;
        e.EstateId = dto.EstateId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.Category = dto.Category;
        e.MinimumTractorHp = dto.MinimumTractorHp;
        e.CapacityPerHour = dto.CapacityPerHour;
        e.CapacityPerDay = dto.CapacityPerDay;
        e.CurrentFarmId = dto.CurrentFarmId;
        e.Ownership = dto.Ownership;
        e.Availability = dto.Availability;
        e.Maintenance = dto.Maintenance;
        e.MaintenanceFromDate = dto.MaintenanceFromDate;
        e.MaintenanceToDate = dto.MaintenanceToDate;
        e.IsActive = dto.IsActive;
    }

    public static CompatibilityDto ToDto(this TractorEquipmentCompatibility e) => new()
    {
        Id = e.Id,
        TractorId = e.TractorId,
        TractorCode = e.Tractor?.Code ?? string.Empty,
        TractorHorsepower = e.Tractor?.Horsepower ?? 0,
        EquipmentId = e.EquipmentId,
        EquipmentCode = e.Equipment?.Code ?? string.Empty,
        EquipmentName = e.Equipment?.Name ?? string.Empty,
        MinimumTractorHp = e.Equipment?.MinimumTractorHp ?? 0,
        IsRecommended = e.IsRecommended,
        Remarks = e.Remarks
    };

    public static OperatorDto ToDto(this Operator e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        FarmId = e.FarmId,
        FarmName = e.Farm?.Name,
        Code = e.Code,
        FullName = e.FullName,
        Skill = e.Skill,
        LicenseNo = e.LicenseNo,
        LicenseExpiry = e.LicenseExpiry,
        WorkTeamId = e.WorkTeamId,
        WorkTeamName = e.WorkTeam?.Name,
        StandardHoursPerDay = e.StandardHoursPerDay,
        IsActive = e.IsActive,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this OperatorUpsertDto dto, Operator e)
    {
        e.CompanyId = dto.CompanyId;
        e.EstateId = dto.EstateId;
        e.FarmId = dto.FarmId;
        e.Code = dto.Code.Trim();
        e.FullName = dto.FullName.Trim();
        e.Skill = dto.Skill;
        e.LicenseNo = dto.LicenseNo;
        e.LicenseExpiry = dto.LicenseExpiry;
        e.WorkTeamId = dto.WorkTeamId;
        e.StandardHoursPerDay = dto.StandardHoursPerDay;
        e.IsActive = dto.IsActive;
    }

    public static WorkTeamDto ToDto(this WorkTeam e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        FarmId = e.FarmId,
        FarmName = e.Farm?.Name,
        Code = e.Code,
        Name = e.Name,
        SupervisorName = e.SupervisorName,
        PrimarySkill = e.PrimarySkill,
        MemberCount = e.MemberCount,
        StandardHoursPerDay = e.StandardHoursPerDay,
        IsActive = e.IsActive,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this WorkTeamUpsertDto dto, WorkTeam e)
    {
        e.CompanyId = dto.CompanyId;
        e.FarmId = dto.FarmId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.SupervisorName = dto.SupervisorName;
        e.PrimarySkill = dto.PrimarySkill;
        e.MemberCount = dto.MemberCount;
        e.StandardHoursPerDay = dto.StandardHoursPerDay;
        e.IsActive = dto.IsActive;
    }

    // --------------------------------------------------------------- scheduling

    public static ResourceScheduleDto ToDto(this ResourceSchedule e) => new()
    {
        Id = e.Id,
        ActivityPlanId = e.ActivityPlanId,
        BlockId = e.BlockId,
        BlockCode = e.Block?.Code ?? string.Empty,
        BlockName = e.Block?.Name ?? string.Empty,
        FarmName = e.Block?.Zone?.Farm?.Name ?? string.Empty,
        ActivityId = e.ActivityId,
        ActivityName = e.Activity?.Name ?? string.Empty,
        ScheduleDate = e.ScheduleDate,
        PlannedStart = e.PlannedStart,
        PlannedEnd = e.PlannedEnd,
        TractorId = e.TractorId,
        TractorCode = e.Tractor?.Code,
        EquipmentId = e.EquipmentId,
        EquipmentCode = e.Equipment?.Code,
        OperatorId = e.OperatorId,
        OperatorName = e.Operator?.FullName,
        WorkTeamId = e.WorkTeamId,
        WorkTeamName = e.WorkTeam?.Name,
        PlannedAreaHa = e.PlannedAreaHa,
        DailyTargetHa = e.DailyTargetHa,
        ExpectedWorkingHours = e.ExpectedWorkingHours,
        SupervisorName = e.SupervisorName,
        Status = e.Status,
        Remarks = e.Remarks,
        DependencyOverrideApproved = e.DependencyOverrideApproved,
        DependencyOverrideReason = e.DependencyOverrideReason,
        RowVersion = e.RowVersion
    };

    // ---------------------------------------------------------------- materials

    public static MaterialDto ToDto(this Material e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        Code = e.Code,
        Name = e.Name,
        Category = e.Category,
        BaseUnit = e.BaseUnit,
        AlternativeUnit = e.AlternativeUnit,
        UnitConversionFactor = e.UnitConversionFactor,
        StandardRatePerHa = e.StandardRatePerHa,
        MinApplicationRate = e.MinApplicationRate,
        MaxApplicationRate = e.MaxApplicationRate,
        IsActive = e.IsActive,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this MaterialUpsertDto dto, Material e)
    {
        e.CompanyId = dto.CompanyId;
        e.Code = dto.Code.Trim();
        e.Name = dto.Name.Trim();
        e.Category = dto.Category;
        e.BaseUnit = dto.BaseUnit;
        e.AlternativeUnit = dto.AlternativeUnit;
        e.UnitConversionFactor = dto.UnitConversionFactor;
        e.StandardRatePerHa = dto.StandardRatePerHa;
        e.MinApplicationRate = dto.MinApplicationRate;
        e.MaxApplicationRate = dto.MaxApplicationRate;
        e.IsActive = dto.IsActive;
    }

    public static ActivityMaterialStandardDto ToDto(this ActivityMaterialStandard e) => new()
    {
        Id = e.Id,
        ActivityId = e.ActivityId,
        ActivityName = e.Activity?.Name ?? string.Empty,
        MaterialId = e.MaterialId,
        MaterialCode = e.Material?.Code ?? string.Empty,
        MaterialName = e.Material?.Name ?? string.Empty,
        Unit = e.Material?.BaseUnit ?? Domain.Enums.UnitOfMeasure.Kilogram,
        CropType = e.CropType,
        CaneVarietyId = e.CaneVarietyId,
        VarietyName = e.CaneVariety?.Name,
        SoilType = e.SoilType,
        StandardRatePerHa = e.StandardRatePerHa,
        NumberOfApplications = e.NumberOfApplications,
        WastePercent = e.WastePercent,
        EffectiveFrom = e.EffectiveFrom,
        EffectiveTo = e.EffectiveTo,
        IsActive = e.IsActive,
        RowVersion = e.RowVersion
    };

    public static void ApplyTo(this ActivityMaterialStandardUpsertDto dto, ActivityMaterialStandard e)
    {
        e.ActivityId = dto.ActivityId;
        e.MaterialId = dto.MaterialId;
        e.CropType = dto.CropType;
        e.CaneVarietyId = dto.CaneVarietyId;
        e.SoilType = dto.SoilType;
        e.StandardRatePerHa = dto.StandardRatePerHa;
        e.NumberOfApplications = dto.NumberOfApplications;
        e.WastePercent = dto.WastePercent;
        e.EffectiveFrom = dto.EffectiveFrom;
        e.EffectiveTo = dto.EffectiveTo;
        e.IsActive = dto.IsActive;
    }

    public static MaterialStockDto ToDto(this MaterialStock e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        EstateId = e.EstateId,
        EstateName = e.Estate?.Name,
        MaterialId = e.MaterialId,
        MaterialCode = e.Material?.Code ?? string.Empty,
        MaterialName = e.Material?.Name ?? string.Empty,
        AvailableStock = e.AvailableStock,
        ReservedQuantity = e.ReservedQuantity,
        IncomingQuantity = e.IncomingQuantity,
        IncomingExpectedDate = e.IncomingExpectedDate,
        NetAvailableQuantity = PlanningFormulas.NetAvailableQuantity(e.AvailableStock, e.IncomingQuantity, e.ReservedQuantity),
        Unit = e.Unit,
        LastSyncedUtc = e.LastSyncedUtc,
        SourceSystem = e.SourceSystem
    };

    // ---------------------------------------------------------------- execution

    public static ActivityActualDto ToDto(this ActivityActual e) => new()
    {
        Id = e.Id,
        ActivityPlanId = e.ActivityPlanId,
        BlockName = e.ActivityPlan?.Block is null ? string.Empty : $"{e.ActivityPlan.Block.Code} — {e.ActivityPlan.Block.Name}",
        ActivityName = e.ActivityPlan?.Activity?.Name ?? string.Empty,
        PlannedStartDate = e.ActivityPlan?.PlannedStartDate ?? default,
        PlannedEndDate = e.ActivityPlan?.PlannedEndDate ?? default,
        PlannedAreaHa = e.ActivityPlan?.PlannedAreaHa ?? 0m,
        PlannedFuelLiters = e.ActivityPlan?.PlannedFuelLiters ?? 0m,
        ActualStartDate = e.ActualStartDate,
        ActualCompletionDate = e.ActualCompletionDate,
        ActualCompletedAreaHa = e.ActualCompletedAreaHa,
        ActualTractorId = e.ActualTractorId,
        ActualTractorCode = e.ActualTractor?.Code,
        ActualEquipmentId = e.ActualEquipmentId,
        ActualEquipmentCode = e.ActualEquipment?.Code,
        ActualOperatorId = e.ActualOperatorId,
        ActualOperatorName = e.ActualOperator?.FullName,
        ActualWorkingHours = e.ActualWorkingHours,
        ActualFuelLiters = e.ActualFuelLiters,
        ActualLaborDays = e.ActualLaborDays,
        AreaVariance = e.AreaVariance,
        FuelVariance = e.FuelVariance,
        ScheduleVarianceDays = e.ScheduleVarianceDays,
        CompletionPercent = e.CompletionPercent,
        DelayReason = e.DelayReason,
        Remarks = e.Remarks,
        MaterialUsages = e.MaterialUsages?.Select(u => u.ToDto()).ToList() ?? new List<ActualMaterialUsageDto>(),
        RowVersion = e.RowVersion
    };

    public static ActualMaterialUsageDto ToDto(this ActualMaterialUsage e) => new()
    {
        Id = e.Id,
        MaterialId = e.MaterialId,
        MaterialCode = e.Material?.Code ?? string.Empty,
        MaterialName = e.Material?.Name ?? string.Empty,
        PlannedQuantity = e.PlannedQuantity,
        ActualQuantity = e.ActualQuantity,
        Variance = e.Variance,
        Unit = e.Unit,
        Remarks = e.Remarks
    };

    // ---------------------------------------------------------------- scenarios

    public static ScenarioDto ToDto(this PlanningScenario e) => new()
    {
        Id = e.Id,
        ProjectionId = e.ProjectionId,
        ProjectionNo = e.Projection?.ProjectionNo ?? string.Empty,
        Name = e.Name,
        Description = e.Description,
        Status = e.Status,
        SimulatedAtUtc = e.SimulatedAtUtc,
        SimulatedBy = e.SimulatedBy,
        Adjustments = e.Adjustments?.Select(a => a.ToDto()).ToList() ?? new List<ScenarioAdjustmentDto>(),
        RowVersion = e.RowVersion
    };

    public static ScenarioAdjustmentDto ToDto(this ScenarioAdjustment e) => new()
    {
        Id = e.Id,
        AdjustmentType = e.AdjustmentType,
        Value = e.Value,
        ActivityId = e.ActivityId,
        ActivityName = e.Activity?.Name,
        Remarks = e.Remarks
    };

    // ----------------------------------------------------------------- auditing

    public static AuditLogDto ToDto(this AuditLog e) => new()
    {
        Id = e.Id,
        CompanyId = e.CompanyId,
        UserName = e.UserName,
        TimestampUtc = e.TimestampUtc,
        Action = e.Action,
        TableName = e.TableName,
        RecordId = e.RecordId,
        OldValues = e.OldValues,
        NewValues = e.NewValues,
        ChangedColumns = e.ChangedColumns,
        IpAddress = e.IpAddress,
        DeviceInfo = e.DeviceInfo,
        Remarks = e.Remarks
    };

    // ------------------------------------------------------------------ lookups

    public static LookupDto ToLookup(this Estate e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, ParentId = e.CompanyId, Value = e.TotalAreaHa };
    public static LookupDto ToLookup(this Farm e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, ParentId = e.EstateId, Value = e.TotalAreaHa };
    public static LookupDto ToLookup(this Zone e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, ParentId = e.FarmId, Value = e.TotalAreaHa };
    public static LookupDto ToLookup(this PlantationBlock e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, ParentId = e.ZoneId, Value = e.PlantableAreaHa };
    public static LookupDto ToLookup(this GrowingSeason e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, ParentId = e.CompanyId };
    public static LookupDto ToLookup(this CaneVariety e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, Value = e.ExpectedYieldPerHa };
    public static LookupDto ToLookup(this PlantingActivity e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, Value = e.SequenceNo };
    public static LookupDto ToLookup(this Material e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, Value = e.StandardRatePerHa };
    public static LookupDto ToLookup(this Tractor e) => new() { Id = e.Id, Code = e.Code, Name = $"{e.Brand} {e.Model}".Trim(), ParentId = e.CurrentFarmId, Value = e.Horsepower };
    public static LookupDto ToLookup(this EquipmentItem e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, ParentId = e.CurrentFarmId, Value = e.MinimumTractorHp };
    public static LookupDto ToLookup(this Operator e) => new() { Id = e.Id, Code = e.Code, Name = e.FullName, ParentId = e.FarmId };
    public static LookupDto ToLookup(this WorkTeam e) => new() { Id = e.Id, Code = e.Code, Name = e.Name, ParentId = e.FarmId, Value = e.MemberCount };
}
