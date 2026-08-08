using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Auditing;
using SugarcanePlanning.Contracts.Capacity;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Dashboard;
using SugarcanePlanning.Contracts.Execution;
using SugarcanePlanning.Contracts.Labor;
using SugarcanePlanning.Contracts.Machinery;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Contracts.Organization;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Contracts.Reports;
using SugarcanePlanning.Contracts.Scheduling;
using SugarcanePlanning.Contracts.Seasons;

namespace SugarcanePlanning.Application.Interfaces;

/// <summary>Company / estate / farm / zone / block master data (section 3).</summary>
public interface ILandStructureService
{
    Task<PagedResult<CompanyDto>> GetCompaniesAsync(QueryParameters q, CancellationToken ct = default);
    Task<CompanyDto> GetCompanyAsync(int id, CancellationToken ct = default);
    Task<CompanyDto> CreateCompanyAsync(CompanyUpsertDto dto, CancellationToken ct = default);
    Task<CompanyDto> UpdateCompanyAsync(int id, CompanyUpsertDto dto, CancellationToken ct = default);
    Task DeleteCompanyAsync(int id, CancellationToken ct = default);

    Task<PagedResult<EstateDto>> GetEstatesAsync(int? companyId, QueryParameters q, CancellationToken ct = default);
    Task<EstateDto> GetEstateAsync(int id, CancellationToken ct = default);
    Task<EstateDto> CreateEstateAsync(EstateUpsertDto dto, CancellationToken ct = default);
    Task<EstateDto> UpdateEstateAsync(int id, EstateUpsertDto dto, CancellationToken ct = default);
    Task DeleteEstateAsync(int id, CancellationToken ct = default);

    Task<PagedResult<FarmDto>> GetFarmsAsync(int? estateId, QueryParameters q, CancellationToken ct = default);
    Task<FarmDto> GetFarmAsync(int id, CancellationToken ct = default);
    Task<FarmDto> CreateFarmAsync(FarmUpsertDto dto, CancellationToken ct = default);
    Task<FarmDto> UpdateFarmAsync(int id, FarmUpsertDto dto, CancellationToken ct = default);
    Task DeleteFarmAsync(int id, CancellationToken ct = default);

    Task<PagedResult<ZoneDto>> GetZonesAsync(int? farmId, QueryParameters q, CancellationToken ct = default);
    Task<ZoneDto> GetZoneAsync(int id, CancellationToken ct = default);
    Task<ZoneDto> CreateZoneAsync(ZoneUpsertDto dto, CancellationToken ct = default);
    Task<ZoneDto> UpdateZoneAsync(int id, ZoneUpsertDto dto, CancellationToken ct = default);
    Task DeleteZoneAsync(int id, CancellationToken ct = default);

    Task<PagedResult<PlantationBlockDto>> GetBlocksAsync(int? zoneId, int? farmId, QueryParameters q, CancellationToken ct = default);
    Task<PlantationBlockDto> GetBlockAsync(int id, CancellationToken ct = default);
    Task<PlantationBlockDto> CreateBlockAsync(PlantationBlockUpsertDto dto, CancellationToken ct = default);
    Task<PlantationBlockDto> UpdateBlockAsync(int id, PlantationBlockUpsertDto dto, CancellationToken ct = default);
    Task DeleteBlockAsync(int id, CancellationToken ct = default);

    Task<IReadOnlyList<LandStructureNodeDto>> GetStructureAsync(int? companyId, CancellationToken ct = default);

    /// <summary>Farm → zone → block with the areas the dashboard tree reports, rolled up at every level.</summary>
    Task<IReadOnlyList<LandCoverageNodeDto>> GetLandCoverageAsync(int? seasonId, int? farmId, CancellationToken ct = default);
}

/// <summary>Growing seasons and cane varieties (section 4).</summary>
public interface ISeasonService
{
    Task<PagedResult<GrowingSeasonDto>> GetSeasonsAsync(QueryParameters q, CancellationToken ct = default);
    Task<GrowingSeasonDto> GetSeasonAsync(int id, CancellationToken ct = default);
    Task<GrowingSeasonDto> CreateSeasonAsync(GrowingSeasonUpsertDto dto, CancellationToken ct = default);
    Task<GrowingSeasonDto> UpdateSeasonAsync(int id, GrowingSeasonUpsertDto dto, CancellationToken ct = default);
    Task DeleteSeasonAsync(int id, CancellationToken ct = default);

    Task<PagedResult<CaneVarietyDto>> GetVarietiesAsync(QueryParameters q, CancellationToken ct = default);
    Task<CaneVarietyDto> GetVarietyAsync(int id, CancellationToken ct = default);
    Task<CaneVarietyDto> CreateVarietyAsync(CaneVarietyUpsertDto dto, CancellationToken ct = default);
    Task<CaneVarietyDto> UpdateVarietyAsync(int id, CaneVarietyUpsertDto dto, CancellationToken ct = default);
    Task DeleteVarietyAsync(int id, CancellationToken ct = default);
}

/// <summary>Activity master and dependency graph (section 6).</summary>
public interface IActivityMasterService
{
    Task<PagedResult<PlantingActivityDto>> GetActivitiesAsync(QueryParameters q, CancellationToken ct = default);
    Task<PlantingActivityDto> GetActivityAsync(int id, CancellationToken ct = default);
    Task<PlantingActivityDto> CreateActivityAsync(PlantingActivityUpsertDto dto, CancellationToken ct = default);
    Task<PlantingActivityDto> UpdateActivityAsync(int id, PlantingActivityUpsertDto dto, CancellationToken ct = default);
    Task DeleteActivityAsync(int id, CancellationToken ct = default);

    Task<IReadOnlyList<ActivityDependencyDto>> GetDependenciesAsync(int? activityId, CancellationToken ct = default);
    Task<ActivityDependencyDto> AddDependencyAsync(ActivityDependencyUpsertDto dto, CancellationToken ct = default);
    Task RemoveDependencyAsync(int id, CancellationToken ct = default);
}

/// <summary>Tractors, implements, compatibility, operators and teams (sections 8, 9, 14).</summary>
public interface IMachineryService
{
    Task<PagedResult<TractorDto>> GetTractorsAsync(QueryParameters q, CancellationToken ct = default);
    Task<TractorDto> GetTractorAsync(int id, CancellationToken ct = default);
    Task<TractorDto> CreateTractorAsync(TractorUpsertDto dto, CancellationToken ct = default);
    Task<TractorDto> UpdateTractorAsync(int id, TractorUpsertDto dto, CancellationToken ct = default);
    Task DeleteTractorAsync(int id, CancellationToken ct = default);

    Task<PagedResult<EquipmentDto>> GetEquipmentAsync(QueryParameters q, CancellationToken ct = default);
    Task<EquipmentDto> GetEquipmentItemAsync(int id, CancellationToken ct = default);
    Task<EquipmentDto> CreateEquipmentAsync(EquipmentUpsertDto dto, CancellationToken ct = default);
    Task<EquipmentDto> UpdateEquipmentAsync(int id, EquipmentUpsertDto dto, CancellationToken ct = default);
    Task DeleteEquipmentAsync(int id, CancellationToken ct = default);

    Task<IReadOnlyList<CompatibilityDto>> GetCompatibilitiesAsync(int? tractorId, int? equipmentId, CancellationToken ct = default);
    Task<CompatibilityDto> AddCompatibilityAsync(CompatibilityUpsertDto dto, CancellationToken ct = default);
    Task RemoveCompatibilityAsync(int id, CancellationToken ct = default);

    Task<PagedResult<OperatorDto>> GetOperatorsAsync(QueryParameters q, CancellationToken ct = default);
    Task<OperatorDto> CreateOperatorAsync(OperatorUpsertDto dto, CancellationToken ct = default);
    Task<OperatorDto> UpdateOperatorAsync(int id, OperatorUpsertDto dto, CancellationToken ct = default);
    Task DeleteOperatorAsync(int id, CancellationToken ct = default);

    Task<PagedResult<WorkTeamDto>> GetWorkTeamsAsync(QueryParameters q, CancellationToken ct = default);
    Task<WorkTeamDto> CreateWorkTeamAsync(WorkTeamUpsertDto dto, CancellationToken ct = default);
    Task<WorkTeamDto> UpdateWorkTeamAsync(int id, WorkTeamUpsertDto dto, CancellationToken ct = default);
    Task DeleteWorkTeamAsync(int id, CancellationToken ct = default);
}

/// <summary>Material master, activity standards and the stock interface (sections 11-13).</summary>
public interface IMaterialMasterService
{
    Task<PagedResult<MaterialDto>> GetMaterialsAsync(QueryParameters q, CancellationToken ct = default);
    Task<MaterialDto> GetMaterialAsync(int id, CancellationToken ct = default);
    Task<MaterialDto> CreateMaterialAsync(MaterialUpsertDto dto, CancellationToken ct = default);
    Task<MaterialDto> UpdateMaterialAsync(int id, MaterialUpsertDto dto, CancellationToken ct = default);
    Task DeleteMaterialAsync(int id, CancellationToken ct = default);

    Task<PagedResult<ActivityMaterialStandardDto>> GetStandardsAsync(int? activityId, QueryParameters q, CancellationToken ct = default);
    Task<ActivityMaterialStandardDto> CreateStandardAsync(ActivityMaterialStandardUpsertDto dto, CancellationToken ct = default);
    Task<ActivityMaterialStandardDto> UpdateStandardAsync(int id, ActivityMaterialStandardUpsertDto dto, CancellationToken ct = default);
    Task DeleteStandardAsync(int id, CancellationToken ct = default);

    Task<IReadOnlyList<MaterialStockDto>> GetStocksAsync(int? estateId, CancellationToken ct = default);
    Task<int> SyncStockAsync(IReadOnlyList<MaterialStockSyncDto> rows, CancellationToken ct = default);
}

/// <summary>Planting projections, validation, workflow and versioning (sections 5, 16).</summary>
public interface IProjectionService
{
    Task<PagedResult<ProjectionSummaryDto>> GetProjectionsAsync(int? seasonId, int? estateId, QueryParameters q, CancellationToken ct = default);
    Task<ProjectionDetailDto> GetProjectionAsync(int id, CancellationToken ct = default);
    Task<ProjectionDetailDto> CreateAsync(ProjectionCreateDto dto, CancellationToken ct = default);
    Task<ProjectionDetailDto> UpdateHeaderAsync(int id, ProjectionUpdateDto dto, CancellationToken ct = default);
    Task DeleteAsync(int id, CancellationToken ct = default);

    Task<ProjectionLineDto> AddLineAsync(int projectionId, ProjectionLineUpsertDto dto, CancellationToken ct = default);
    Task<ProjectionLineDto> UpdateLineAsync(int projectionId, int lineId, ProjectionLineUpsertDto dto, CancellationToken ct = default);
    Task RemoveLineAsync(int projectionId, int lineId, CancellationToken ct = default);

    Task<ProjectionDetailDto> ExecuteWorkflowAsync(int id, WorkflowActionDto action, CancellationToken ct = default);
    Task<ProjectionDetailDto> ReviseAsync(int id, ReviseProjectionDto dto, CancellationToken ct = default);
    Task<IReadOnlyList<ProjectionSummaryDto>> GetVersionsAsync(int id, CancellationToken ct = default);
    Task<VersionComparisonDto> CompareVersionsAsync(int fromId, int toId, CancellationToken ct = default);
}

/// <summary>Activity-plan generation and the Gantt view (section 7).</summary>
public interface IActivityPlanService
{
    Task<GenerateActivityPlanResultDto> GenerateAsync(GenerateActivityPlanRequest request, CancellationToken ct = default);
    Task<PagedResult<ActivityPlanDto>> GetPlansAsync(int? projectionId, int? blockId, int? activityId, QueryParameters q, CancellationToken ct = default);
    Task<ActivityPlanDto> GetPlanAsync(int id, CancellationToken ct = default);
    Task<ActivityPlanDto> UpdatePlanAsync(int id, ActivityPlanUpdateDto dto, CancellationToken ct = default);
    Task<GanttViewDto> GetGanttAsync(int projectionId, int? farmId, CancellationToken ct = default);
}

/// <summary>Resource booking with full conflict detection (section 10).</summary>
public interface ISchedulingService
{
    Task<ScheduleValidationResultDto> ValidateAsync(ResourceScheduleUpsertDto dto, int? existingScheduleId, CancellationToken ct = default);
    Task<ResourceScheduleDto> CreateAsync(ResourceScheduleUpsertDto dto, CancellationToken ct = default);
    Task<ResourceScheduleDto> UpdateAsync(int id, ResourceScheduleUpsertDto dto, CancellationToken ct = default);
    Task CancelAsync(int id, CancellationToken ct = default);
    Task<ScheduleBoardDto> GetBoardAsync(DateOnly from, DateOnly to, string groupBy, int? farmId, int? blockId, CancellationToken ct = default);
    Task<PagedResult<ResourceScheduleDto>> GetSchedulesAsync(DateOnly? from, DateOnly? to, int? tractorId, int? equipmentId, int? operatorId, QueryParameters q, CancellationToken ct = default);
}

/// <summary>Material requirement planning (sections 12, 13).</summary>
public interface IMaterialRequirementService
{
    /// <summary>Recomputes and stores the per-activity-plan requirement rows for a projection.</summary>
    Task<int> RecalculateAsync(int projectionId, CancellationToken ct = default);
    Task<IReadOnlyList<MaterialRequirementDto>> GetRequirementsAsync(MaterialRequirementQuery query, CancellationToken ct = default);
}

/// <summary>Fuel and labor projections (section 14).</summary>
public interface IFuelLaborService
{
    Task<IReadOnlyList<FuelProjectionDto>> GetFuelProjectionAsync(FuelLaborQuery query, CancellationToken ct = default);
    Task<IReadOnlyList<LaborProjectionDto>> GetLaborProjectionAsync(FuelLaborQuery query, CancellationToken ct = default);
}

/// <summary>Capacity comparison and what-if scenarios (section 15).</summary>
public interface ICapacityService
{
    Task<CapacityAnalysisDto> AnalyzeAsync(int projectionId, CancellationToken ct = default);
}

public interface IScenarioService
{
    Task<IReadOnlyList<ScenarioDto>> GetScenariosAsync(int projectionId, CancellationToken ct = default);
    Task<ScenarioDto> CreateAsync(ScenarioUpsertDto dto, CancellationToken ct = default);
    Task<ScenarioDto> UpdateAsync(int id, ScenarioUpsertDto dto, CancellationToken ct = default);
    Task DeleteAsync(int id, CancellationToken ct = default);
    Task<ScenarioResultDto> SimulateAsync(int id, CancellationToken ct = default);
}

/// <summary>Actual execution capture and projection-versus-actual (section 17).</summary>
public interface IExecutionService
{
    Task<ActivityActualDto> GetActualAsync(int activityPlanId, CancellationToken ct = default);
    Task<ActivityActualDto> RecordAsync(ActivityActualUpsertDto dto, CancellationToken ct = default);
    Task<IReadOnlyList<ProjectionVsActualDto>> CompareAsync(int projectionId, string groupBy, CancellationToken ct = default);
}

/// <summary>Dashboard aggregation (section 18).</summary>
public interface IDashboardService
{
    Task<DashboardDto> GetAsync(int? seasonId, int? estateId, CancellationToken ct = default);
}

/// <summary>Report generation and export (section 20).</summary>
public interface IReportService
{
    Task<ReportResultDto> GenerateAsync(ReportKey key, ReportRequest request, CancellationToken ct = default);
}

/// <summary>Turns a report result into a downloadable file.</summary>
public interface IReportExporter
{
    ExportFormat Format { get; }
    string ContentType { get; }
    string FileExtension { get; }
    byte[] Export(ReportResultDto report);
}

/// <summary>Read side of the audit trail (section 22).</summary>
public interface IAuditQueryService
{
    Task<PagedResult<AuditLogDto>> QueryAsync(AuditLogQuery query, CancellationToken ct = default);
}

/// <summary>Value helps for every dropdown in the UI.</summary>
public interface ILookupService
{
    Task<IReadOnlyList<LookupDto>> GetAsync(string kind, int? parentId, CancellationToken ct = default);
}
