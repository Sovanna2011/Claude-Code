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

namespace SugarcanePlanning.Client.Services;

/// <summary>
/// One typed entry point per REST endpoint, so pages never build URLs by hand.
/// </summary>
public class PlanningApi
{
    private readonly ApiClient _api;
    public PlanningApi(ApiClient api) => _api = api;

    private static string Q(QueryParameters q) => q.ToQueryString();
    private static PagedResult<T> Empty<T>() => new(Array.Empty<T>(), 0, 1, 25);

    // ---------------------------------------------------------------- lookups

    public async Task<IReadOnlyList<LookupDto>> LookupAsync(string kind, int? parentId = null)
        => await _api.GetAsync<List<LookupDto>>($"api/lookups/{kind}" + (parentId is null ? "" : $"?parentId={parentId}"))
           ?? new List<LookupDto>();

    // ------------------------------------------------------------------- land

    public async Task<PagedResult<CompanyDto>> GetCompaniesAsync(QueryParameters q)
        => await _api.GetAsync<PagedResult<CompanyDto>>($"api/companies?{Q(q)}") ?? Empty<CompanyDto>();
    public Task<CompanyDto?> CreateCompanyAsync(CompanyUpsertDto dto) => _api.PostAsync<CompanyDto>("api/companies", dto);
    public Task<CompanyDto?> UpdateCompanyAsync(int id, CompanyUpsertDto dto) => _api.PutAsync<CompanyDto>($"api/companies/{id}", dto);
    public Task DeleteCompanyAsync(int id) => _api.DeleteAsync($"api/companies/{id}");

    public async Task<PagedResult<EstateDto>> GetEstatesAsync(int? companyId, QueryParameters q)
        => await _api.GetAsync<PagedResult<EstateDto>>($"api/estates?{Q(q)}" + (companyId is null ? "" : $"&companyId={companyId}"))
           ?? Empty<EstateDto>();
    public Task<EstateDto?> CreateEstateAsync(EstateUpsertDto dto) => _api.PostAsync<EstateDto>("api/estates", dto);
    public Task<EstateDto?> UpdateEstateAsync(int id, EstateUpsertDto dto) => _api.PutAsync<EstateDto>($"api/estates/{id}", dto);
    public Task DeleteEstateAsync(int id) => _api.DeleteAsync($"api/estates/{id}");

    public async Task<PagedResult<FarmDto>> GetFarmsAsync(int? estateId, QueryParameters q)
        => await _api.GetAsync<PagedResult<FarmDto>>($"api/farms?{Q(q)}" + (estateId is null ? "" : $"&estateId={estateId}"))
           ?? Empty<FarmDto>();
    public Task<FarmDto?> CreateFarmAsync(FarmUpsertDto dto) => _api.PostAsync<FarmDto>("api/farms", dto);
    public Task<FarmDto?> UpdateFarmAsync(int id, FarmUpsertDto dto) => _api.PutAsync<FarmDto>($"api/farms/{id}", dto);
    public Task DeleteFarmAsync(int id) => _api.DeleteAsync($"api/farms/{id}");

    public async Task<PagedResult<ZoneDto>> GetZonesAsync(int? farmId, QueryParameters q)
        => await _api.GetAsync<PagedResult<ZoneDto>>($"api/zones?{Q(q)}" + (farmId is null ? "" : $"&farmId={farmId}"))
           ?? Empty<ZoneDto>();
    public Task<ZoneDto?> CreateZoneAsync(ZoneUpsertDto dto) => _api.PostAsync<ZoneDto>("api/zones", dto);
    public Task<ZoneDto?> UpdateZoneAsync(int id, ZoneUpsertDto dto) => _api.PutAsync<ZoneDto>($"api/zones/{id}", dto);
    public Task DeleteZoneAsync(int id) => _api.DeleteAsync($"api/zones/{id}");

    public async Task<PagedResult<PlantationBlockDto>> GetBlocksAsync(int? zoneId, int? farmId, QueryParameters q)
        => await _api.GetAsync<PagedResult<PlantationBlockDto>>(
               $"api/blocks?{Q(q)}" + (zoneId is null ? "" : $"&zoneId={zoneId}") + (farmId is null ? "" : $"&farmId={farmId}"))
           ?? Empty<PlantationBlockDto>();
    public Task<PlantationBlockDto?> CreateBlockAsync(PlantationBlockUpsertDto dto) => _api.PostAsync<PlantationBlockDto>("api/blocks", dto);
    public Task<PlantationBlockDto?> UpdateBlockAsync(int id, PlantationBlockUpsertDto dto) => _api.PutAsync<PlantationBlockDto>($"api/blocks/{id}", dto);
    public Task DeleteBlockAsync(int id) => _api.DeleteAsync($"api/blocks/{id}");

    public async Task<IReadOnlyList<LandStructureNodeDto>> GetLandStructureAsync(int? companyId = null)
        => await _api.GetAsync<List<LandStructureNodeDto>>("api/land-structure" + (companyId is null ? "" : $"?companyId={companyId}"))
           ?? new List<LandStructureNodeDto>();

    // ---------------------------------------------------------------- seasons

    public async Task<PagedResult<GrowingSeasonDto>> GetSeasonsAsync(QueryParameters q)
        => await _api.GetAsync<PagedResult<GrowingSeasonDto>>($"api/seasons?{Q(q)}") ?? Empty<GrowingSeasonDto>();
    public Task<GrowingSeasonDto?> CreateSeasonAsync(GrowingSeasonUpsertDto dto) => _api.PostAsync<GrowingSeasonDto>("api/seasons", dto);
    public Task<GrowingSeasonDto?> UpdateSeasonAsync(int id, GrowingSeasonUpsertDto dto) => _api.PutAsync<GrowingSeasonDto>($"api/seasons/{id}", dto);
    public Task DeleteSeasonAsync(int id) => _api.DeleteAsync($"api/seasons/{id}");

    public async Task<PagedResult<CaneVarietyDto>> GetVarietiesAsync(QueryParameters q)
        => await _api.GetAsync<PagedResult<CaneVarietyDto>>($"api/varieties?{Q(q)}") ?? Empty<CaneVarietyDto>();
    public Task<CaneVarietyDto?> CreateVarietyAsync(CaneVarietyUpsertDto dto) => _api.PostAsync<CaneVarietyDto>("api/varieties", dto);
    public Task<CaneVarietyDto?> UpdateVarietyAsync(int id, CaneVarietyUpsertDto dto) => _api.PutAsync<CaneVarietyDto>($"api/varieties/{id}", dto);
    public Task DeleteVarietyAsync(int id) => _api.DeleteAsync($"api/varieties/{id}");

    // ------------------------------------------------------------- activities

    public async Task<PagedResult<PlantingActivityDto>> GetActivitiesAsync(QueryParameters q)
        => await _api.GetAsync<PagedResult<PlantingActivityDto>>($"api/activities?{Q(q)}") ?? Empty<PlantingActivityDto>();
    public Task<PlantingActivityDto?> CreateActivityAsync(PlantingActivityUpsertDto dto) => _api.PostAsync<PlantingActivityDto>("api/activities", dto);
    public Task<PlantingActivityDto?> UpdateActivityAsync(int id, PlantingActivityUpsertDto dto) => _api.PutAsync<PlantingActivityDto>($"api/activities/{id}", dto);
    public Task DeleteActivityAsync(int id) => _api.DeleteAsync($"api/activities/{id}");

    public async Task<IReadOnlyList<ActivityDependencyDto>> GetDependenciesAsync(int? activityId = null)
        => await _api.GetAsync<List<ActivityDependencyDto>>("api/activities/dependencies" + (activityId is null ? "" : $"?activityId={activityId}"))
           ?? new List<ActivityDependencyDto>();
    public Task<ActivityDependencyDto?> AddDependencyAsync(ActivityDependencyUpsertDto dto) => _api.PostAsync<ActivityDependencyDto>("api/activities/dependencies", dto);
    public Task RemoveDependencyAsync(int id) => _api.DeleteAsync($"api/activities/dependencies/{id}");

    // ------------------------------------------------------------ projections

    public async Task<PagedResult<ProjectionSummaryDto>> GetProjectionsAsync(int? seasonId, int? estateId, QueryParameters q)
        => await _api.GetAsync<PagedResult<ProjectionSummaryDto>>(
               $"api/projections?{Q(q)}" + (seasonId is null ? "" : $"&seasonId={seasonId}") + (estateId is null ? "" : $"&estateId={estateId}"))
           ?? Empty<ProjectionSummaryDto>();
    public Task<ProjectionDetailDto?> GetProjectionAsync(int id) => _api.GetAsync<ProjectionDetailDto>($"api/projections/{id}");
    public Task<ProjectionDetailDto?> CreateProjectionAsync(ProjectionCreateDto dto) => _api.PostAsync<ProjectionDetailDto>("api/projections", dto);
    public Task<ProjectionDetailDto?> UpdateProjectionAsync(int id, ProjectionUpdateDto dto) => _api.PutAsync<ProjectionDetailDto>($"api/projections/{id}", dto);
    public Task DeleteProjectionAsync(int id) => _api.DeleteAsync($"api/projections/{id}");
    public Task<ProjectionLineDto?> AddLineAsync(int id, ProjectionLineUpsertDto dto) => _api.PostAsync<ProjectionLineDto>($"api/projections/{id}/lines", dto);
    public Task<ProjectionLineDto?> UpdateLineAsync(int id, int lineId, ProjectionLineUpsertDto dto) => _api.PutAsync<ProjectionLineDto>($"api/projections/{id}/lines/{lineId}", dto);
    public Task RemoveLineAsync(int id, int lineId) => _api.DeleteAsync($"api/projections/{id}/lines/{lineId}");
    public Task<ProjectionDetailDto?> WorkflowAsync(int id, WorkflowActionDto action) => _api.PostAsync<ProjectionDetailDto>($"api/projections/{id}/workflow", action);
    public Task<ProjectionDetailDto?> ReviseAsync(int id, ReviseProjectionDto dto) => _api.PostAsync<ProjectionDetailDto>($"api/projections/{id}/revise", dto);
    public async Task<IReadOnlyList<ProjectionSummaryDto>> GetVersionsAsync(int id)
        => await _api.GetAsync<List<ProjectionSummaryDto>>($"api/projections/{id}/versions") ?? new List<ProjectionSummaryDto>();
    public Task<VersionComparisonDto?> CompareVersionsAsync(int fromId, int toId)
        => _api.GetAsync<VersionComparisonDto>($"api/projections/compare?fromId={fromId}&toId={toId}");

    // --------------------------------------------------------- activity plans

    public async Task<PagedResult<ActivityPlanDto>> GetActivityPlansAsync(int? projectionId, int? blockId, int? activityId, QueryParameters q)
        => await _api.GetAsync<PagedResult<ActivityPlanDto>>($"api/activity-plans?{Q(q)}"
               + (projectionId is null ? "" : $"&projectionId={projectionId}")
               + (blockId is null ? "" : $"&blockId={blockId}")
               + (activityId is null ? "" : $"&activityId={activityId}"))
           ?? Empty<ActivityPlanDto>();
    public Task<GenerateActivityPlanResultDto?> GenerateActivityPlansAsync(GenerateActivityPlanRequest request)
        => _api.PostAsync<GenerateActivityPlanResultDto>("api/activity-plans/generate", request);
    public Task<ActivityPlanDto?> UpdateActivityPlanAsync(int id, ActivityPlanUpdateDto dto)
        => _api.PutAsync<ActivityPlanDto>($"api/activity-plans/{id}", dto);
    public Task<GanttViewDto?> GetGanttAsync(int projectionId, int? farmId = null)
        => _api.GetAsync<GanttViewDto>($"api/activity-plans/gantt?projectionId={projectionId}" + (farmId is null ? "" : $"&farmId={farmId}"));

    // -------------------------------------------------------------- machinery

    public async Task<PagedResult<TractorDto>> GetTractorsAsync(QueryParameters q)
        => await _api.GetAsync<PagedResult<TractorDto>>($"api/tractors?{Q(q)}") ?? Empty<TractorDto>();
    public Task<TractorDto?> CreateTractorAsync(TractorUpsertDto dto) => _api.PostAsync<TractorDto>("api/tractors", dto);
    public Task<TractorDto?> UpdateTractorAsync(int id, TractorUpsertDto dto) => _api.PutAsync<TractorDto>($"api/tractors/{id}", dto);
    public Task DeleteTractorAsync(int id) => _api.DeleteAsync($"api/tractors/{id}");

    public async Task<PagedResult<EquipmentDto>> GetEquipmentAsync(QueryParameters q)
        => await _api.GetAsync<PagedResult<EquipmentDto>>($"api/equipment?{Q(q)}") ?? Empty<EquipmentDto>();
    public Task<EquipmentDto?> CreateEquipmentAsync(EquipmentUpsertDto dto) => _api.PostAsync<EquipmentDto>("api/equipment", dto);
    public Task<EquipmentDto?> UpdateEquipmentAsync(int id, EquipmentUpsertDto dto) => _api.PutAsync<EquipmentDto>($"api/equipment/{id}", dto);
    public Task DeleteEquipmentAsync(int id) => _api.DeleteAsync($"api/equipment/{id}");

    public async Task<IReadOnlyList<CompatibilityDto>> GetCompatibilityAsync(int? tractorId = null, int? equipmentId = null)
        => await _api.GetAsync<List<CompatibilityDto>>("api/equipment/compatibility?"
               + (tractorId is null ? "" : $"tractorId={tractorId}&")
               + (equipmentId is null ? "" : $"equipmentId={equipmentId}"))
           ?? new List<CompatibilityDto>();
    public Task<CompatibilityDto?> AddCompatibilityAsync(CompatibilityUpsertDto dto) => _api.PostAsync<CompatibilityDto>("api/equipment/compatibility", dto);
    public Task RemoveCompatibilityAsync(int id) => _api.DeleteAsync($"api/equipment/compatibility/{id}");

    public async Task<PagedResult<OperatorDto>> GetOperatorsAsync(QueryParameters q)
        => await _api.GetAsync<PagedResult<OperatorDto>>($"api/workforce/operators?{Q(q)}") ?? Empty<OperatorDto>();
    public Task<OperatorDto?> CreateOperatorAsync(OperatorUpsertDto dto) => _api.PostAsync<OperatorDto>("api/workforce/operators", dto);
    public Task<OperatorDto?> UpdateOperatorAsync(int id, OperatorUpsertDto dto) => _api.PutAsync<OperatorDto>($"api/workforce/operators/{id}", dto);
    public Task DeleteOperatorAsync(int id) => _api.DeleteAsync($"api/workforce/operators/{id}");

    public async Task<PagedResult<WorkTeamDto>> GetWorkTeamsAsync(QueryParameters q)
        => await _api.GetAsync<PagedResult<WorkTeamDto>>($"api/workforce/teams?{Q(q)}") ?? Empty<WorkTeamDto>();
    public Task<WorkTeamDto?> CreateWorkTeamAsync(WorkTeamUpsertDto dto) => _api.PostAsync<WorkTeamDto>("api/workforce/teams", dto);
    public Task<WorkTeamDto?> UpdateWorkTeamAsync(int id, WorkTeamUpsertDto dto) => _api.PutAsync<WorkTeamDto>($"api/workforce/teams/{id}", dto);
    public Task DeleteWorkTeamAsync(int id) => _api.DeleteAsync($"api/workforce/teams/{id}");

    // ------------------------------------------------------------- scheduling

    public async Task<PagedResult<ResourceScheduleDto>> GetSchedulesAsync(DateOnly? from, DateOnly? to,
        int? tractorId, int? equipmentId, int? operatorId, QueryParameters q)
        => await _api.GetAsync<PagedResult<ResourceScheduleDto>>($"api/schedules?{Q(q)}"
               + (from is null ? "" : $"&from={from:yyyy-MM-dd}")
               + (to is null ? "" : $"&to={to:yyyy-MM-dd}")
               + (tractorId is null ? "" : $"&tractorId={tractorId}")
               + (equipmentId is null ? "" : $"&equipmentId={equipmentId}")
               + (operatorId is null ? "" : $"&operatorId={operatorId}"))
           ?? Empty<ResourceScheduleDto>();
    public Task<ScheduleBoardDto?> GetScheduleBoardAsync(DateOnly from, DateOnly to, string groupBy, int? farmId = null, int? blockId = null)
        => _api.GetAsync<ScheduleBoardDto>($"api/schedules/board?from={from:yyyy-MM-dd}&to={to:yyyy-MM-dd}&groupBy={groupBy}"
               + (farmId is null ? "" : $"&farmId={farmId}") + (blockId is null ? "" : $"&blockId={blockId}"));
    public Task<ScheduleValidationResultDto?> ValidateScheduleAsync(ResourceScheduleUpsertDto dto, int? scheduleId = null)
        => _api.PostAsync<ScheduleValidationResultDto>("api/schedules/validate" + (scheduleId is null ? "" : $"?scheduleId={scheduleId}"), dto);
    public Task<ResourceScheduleDto?> CreateScheduleAsync(ResourceScheduleUpsertDto dto) => _api.PostAsync<ResourceScheduleDto>("api/schedules", dto);
    public Task<ResourceScheduleDto?> UpdateScheduleAsync(int id, ResourceScheduleUpsertDto dto) => _api.PutAsync<ResourceScheduleDto>($"api/schedules/{id}", dto);
    public Task CancelScheduleAsync(int id) => _api.DeleteAsync($"api/schedules/{id}");

    // -------------------------------------------------------------- materials

    public async Task<PagedResult<MaterialDto>> GetMaterialsAsync(QueryParameters q)
        => await _api.GetAsync<PagedResult<MaterialDto>>($"api/materials?{Q(q)}") ?? Empty<MaterialDto>();
    public Task<MaterialDto?> CreateMaterialAsync(MaterialUpsertDto dto) => _api.PostAsync<MaterialDto>("api/materials", dto);
    public Task<MaterialDto?> UpdateMaterialAsync(int id, MaterialUpsertDto dto) => _api.PutAsync<MaterialDto>($"api/materials/{id}", dto);
    public Task DeleteMaterialAsync(int id) => _api.DeleteAsync($"api/materials/{id}");

    public async Task<PagedResult<ActivityMaterialStandardDto>> GetStandardsAsync(int? activityId, QueryParameters q)
        => await _api.GetAsync<PagedResult<ActivityMaterialStandardDto>>(
               $"api/materials/standards?{Q(q)}" + (activityId is null ? "" : $"&activityId={activityId}"))
           ?? Empty<ActivityMaterialStandardDto>();
    public Task<ActivityMaterialStandardDto?> CreateStandardAsync(ActivityMaterialStandardUpsertDto dto)
        => _api.PostAsync<ActivityMaterialStandardDto>("api/materials/standards", dto);
    public Task<ActivityMaterialStandardDto?> UpdateStandardAsync(int id, ActivityMaterialStandardUpsertDto dto)
        => _api.PutAsync<ActivityMaterialStandardDto>($"api/materials/standards/{id}", dto);
    public Task DeleteStandardAsync(int id) => _api.DeleteAsync($"api/materials/standards/{id}");

    public async Task<IReadOnlyList<MaterialStockDto>> GetStockAsync(int? estateId = null)
        => await _api.GetAsync<List<MaterialStockDto>>("api/materials/stock" + (estateId is null ? "" : $"?estateId={estateId}"))
           ?? new List<MaterialStockDto>();

    public async Task<IReadOnlyList<MaterialRequirementDto>> GetMaterialRequirementsAsync(MaterialRequirementQuery query)
        => await _api.GetAsync<List<MaterialRequirementDto>>($"api/material-requirements?{query.ToQueryString()}")
           ?? new List<MaterialRequirementDto>();
    public Task<object?> RecalculateRequirementsAsync(int projectionId)
        => _api.PostAsync<object>($"api/material-requirements/recalculate?projectionId={projectionId}", null);

    // ------------------------------------------------------------ fuel & labor

    public async Task<IReadOnlyList<FuelProjectionDto>> GetFuelProjectionAsync(FuelLaborQuery query)
        => await _api.GetAsync<List<FuelProjectionDto>>($"api/projections-resources/fuel?{query.ToQueryString()}")
           ?? new List<FuelProjectionDto>();
    public async Task<IReadOnlyList<LaborProjectionDto>> GetLaborProjectionAsync(FuelLaborQuery query)
        => await _api.GetAsync<List<LaborProjectionDto>>($"api/projections-resources/labor?{query.ToQueryString()}")
           ?? new List<LaborProjectionDto>();

    // --------------------------------------------------- capacity & scenarios

    public Task<CapacityAnalysisDto?> GetCapacityAsync(int projectionId) => _api.GetAsync<CapacityAnalysisDto>($"api/capacity/{projectionId}");
    public async Task<IReadOnlyList<ScenarioDto>> GetScenariosAsync(int projectionId)
        => await _api.GetAsync<List<ScenarioDto>>($"api/capacity/scenarios?projectionId={projectionId}") ?? new List<ScenarioDto>();
    public Task<ScenarioDto?> CreateScenarioAsync(ScenarioUpsertDto dto) => _api.PostAsync<ScenarioDto>("api/capacity/scenarios", dto);
    public Task<ScenarioDto?> UpdateScenarioAsync(int id, ScenarioUpsertDto dto) => _api.PutAsync<ScenarioDto>($"api/capacity/scenarios/{id}", dto);
    public Task DeleteScenarioAsync(int id) => _api.DeleteAsync($"api/capacity/scenarios/{id}");
    public Task<ScenarioResultDto?> SimulateScenarioAsync(int id) => _api.PostAsync<ScenarioResultDto>($"api/capacity/scenarios/{id}/simulate", null);

    // -------------------------------------------------------------- execution

    public Task<ActivityActualDto?> GetActualAsync(int activityPlanId) => _api.GetAsync<ActivityActualDto>($"api/execution/{activityPlanId}");
    public Task<ActivityActualDto?> RecordActualAsync(ActivityActualUpsertDto dto) => _api.PostAsync<ActivityActualDto>("api/execution", dto);
    public async Task<IReadOnlyList<ProjectionVsActualDto>> CompareActualAsync(int projectionId, string groupBy)
        => await _api.GetAsync<List<ProjectionVsActualDto>>($"api/execution/compare/{projectionId}?groupBy={groupBy}")
           ?? new List<ProjectionVsActualDto>();

    // ------------------------------------------------------ dashboard, reports

    public Task<DashboardDto?> GetDashboardAsync(int? seasonId = null, int? estateId = null)
        => _api.GetAsync<DashboardDto>("api/dashboard?" + (seasonId is null ? "" : $"seasonId={seasonId}&") + (estateId is null ? "" : $"estateId={estateId}"));

    public Task<ReportResultDto?> GetReportAsync(ReportKey key, ReportRequest request)
        => _api.GetAsync<ReportResultDto>($"api/reports/{key}?{request.ToQueryString()}");

    public Task<(byte[] Content, string FileName, string ContentType)> ExportReportAsync(ReportKey key,
        ExportFormat format, ReportRequest request)
        => _api.DownloadAsync($"api/reports/{key}/export?format={format}&{request.ToQueryString()}");

    // ----------------------------------------------------------------- audit

    public async Task<PagedResult<AuditLogDto>> GetAuditAsync(AuditLogQuery query)
        => await _api.GetAsync<PagedResult<AuditLogDto>>($"api/audit?{query.ToQueryString()}") ?? Empty<AuditLogDto>();
}
