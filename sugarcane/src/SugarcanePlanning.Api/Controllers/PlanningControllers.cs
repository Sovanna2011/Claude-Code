using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Capacity;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Dashboard;
using SugarcanePlanning.Contracts.Execution;
using SugarcanePlanning.Contracts.Labor;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Contracts.Scheduling;

namespace SugarcanePlanning.Api.Controllers;

/// <summary>Activity master and dependency setup (section 6).</summary>
[Route("api/activities")]
public class ActivitiesController : ApiControllerBase
{
    private readonly IActivityMasterService _service;
    public ActivitiesController(IActivityMasterService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<PlantingActivityDto>>> GetAll([FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetActivitiesAsync(q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PlantingActivityDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetActivityAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<PlantingActivityDto>> Create([FromBody] PlantingActivityUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateActivityAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<PlantingActivityDto>> Update(int id, [FromBody] PlantingActivityUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateActivityAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteActivityAsync(id, ct);
        return NoContent();
    }

    [HttpGet("dependencies"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<ActivityDependencyDto>>> Dependencies([FromQuery] int? activityId, CancellationToken ct)
        => Ok(await _service.GetDependenciesAsync(activityId, ct));

    [HttpPost("dependencies"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<ActivityDependencyDto>> AddDependency([FromBody] ActivityDependencyUpsertDto dto, CancellationToken ct)
        => Ok(await _service.AddDependencyAsync(dto, ct));

    [HttpDelete("dependencies/{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<IActionResult> RemoveDependency(int id, CancellationToken ct)
    {
        await _service.RemoveDependencyAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Planting projections, workflow and versioning (sections 5, 16).</summary>
[Route("api/projections")]
public class ProjectionsController : ApiControllerBase
{
    private readonly IProjectionService _service;
    public ProjectionsController(IProjectionService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<ProjectionSummaryDto>>> GetAll([FromQuery] int? seasonId,
        [FromQuery] int? estateId, [FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetProjectionsAsync(seasonId, estateId, q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<ProjectionDetailDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetProjectionAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.Create)]
    public async Task<ActionResult<ProjectionDetailDto>> Create([FromBody] ProjectionCreateDto dto, CancellationToken ct)
    {
        var created = await _service.CreateAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.Edit)]
    public async Task<ActionResult<ProjectionDetailDto>> Update(int id, [FromBody] ProjectionUpdateDto dto, CancellationToken ct)
        => Ok(await _service.UpdateHeaderAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteAsync(id, ct);
        return NoContent();
    }

    [HttpPost("{id:int}/lines"), Authorize(Policy = Policies.Edit)]
    public async Task<ActionResult<ProjectionLineDto>> AddLine(int id, [FromBody] ProjectionLineUpsertDto dto, CancellationToken ct)
        => Ok(await _service.AddLineAsync(id, dto, ct));

    [HttpPut("{id:int}/lines/{lineId:int}"), Authorize(Policy = Policies.Edit)]
    public async Task<ActionResult<ProjectionLineDto>> UpdateLine(int id, int lineId,
        [FromBody] ProjectionLineUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateLineAsync(id, lineId, dto, ct));

    [HttpDelete("{id:int}/lines/{lineId:int}"), Authorize(Policy = Policies.Edit)]
    public async Task<IActionResult> RemoveLine(int id, int lineId, CancellationToken ct)
    {
        await _service.RemoveLineAsync(id, lineId, ct);
        return NoContent();
    }

    /// <summary>
    /// Submit / review / approve / reject / return / close (section 16). This is the one action
    /// without a policy attribute: the permission required depends on the action in the body —
    /// Submit, Approve, Reject, Revise and Close each map to a different one — so the check lives
    /// in <c>ProjectionService.ExecuteWorkflowAsync</c>, which refuses before reading anything.
    /// </summary>
    [HttpPost("{id:int}/workflow")]
    public async Task<ActionResult<ProjectionDetailDto>> Workflow(int id, [FromBody] WorkflowActionDto action, CancellationToken ct)
        => Ok(await _service.ExecuteWorkflowAsync(id, action, ct));

    [HttpPost("{id:int}/revise"), Authorize(Policy = Policies.Revise)]
    public async Task<ActionResult<ProjectionDetailDto>> Revise(int id, [FromBody] ReviseProjectionDto dto, CancellationToken ct)
        => Ok(await _service.ReviseAsync(id, dto, ct));

    [HttpGet("{id:int}/versions"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<ProjectionSummaryDto>>> Versions(int id, CancellationToken ct)
        => Ok(await _service.GetVersionsAsync(id, ct));

    [HttpGet("compare"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<VersionComparisonDto>> Compare([FromQuery] int fromId, [FromQuery] int toId, CancellationToken ct)
        => Ok(await _service.CompareVersionsAsync(fromId, toId, ct));
}

/// <summary>Activity planning and the Gantt view (section 7).</summary>
[Route("api/activity-plans")]
public class ActivityPlansController : ApiControllerBase
{
    private readonly IActivityPlanService _service;
    public ActivityPlansController(IActivityPlanService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<ActivityPlanDto>>> GetAll([FromQuery] int? projectionId,
        [FromQuery] int? blockId, [FromQuery] int? activityId, [FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetPlansAsync(projectionId, blockId, activityId, q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<ActivityPlanDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetPlanAsync(id, ct));

    /// <summary>Generates the activity plans of an approved projection (section 7).</summary>
    [HttpPost("generate"), Authorize(Policy = Policies.Create)]
    public async Task<ActionResult<GenerateActivityPlanResultDto>> Generate(
        [FromBody] GenerateActivityPlanRequest request, CancellationToken ct)
        => Ok(await _service.GenerateAsync(request, ct));

    [HttpPut("{id:int}"), Authorize(Policy = Policies.Edit)]
    public async Task<ActionResult<ActivityPlanDto>> Update(int id, [FromBody] ActivityPlanUpdateDto dto, CancellationToken ct)
        => Ok(await _service.UpdatePlanAsync(id, dto, ct));

    [HttpGet("gantt"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<GanttViewDto>> Gantt([FromQuery] int projectionId, [FromQuery] int? farmId, CancellationToken ct)
        => Ok(await _service.GetGanttAsync(projectionId, farmId, ct));
}

/// <summary>Tractor, equipment and operator scheduling with conflict detection (section 10).</summary>
[Route("api/schedules")]
public class SchedulesController : ApiControllerBase
{
    private readonly ISchedulingService _service;
    public SchedulesController(ISchedulingService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<ResourceScheduleDto>>> GetAll([FromQuery] DateOnly? from,
        [FromQuery] DateOnly? to, [FromQuery] int? tractorId, [FromQuery] int? equipmentId,
        [FromQuery] int? operatorId, [FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetSchedulesAsync(from, to, tractorId, equipmentId, operatorId, q, ct));

    /// <summary>Daily / weekly / monthly / tractor / equipment / farm / block board.</summary>
    [HttpGet("board"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<ScheduleBoardDto>> Board([FromQuery] DateOnly from, [FromQuery] DateOnly to,
        [FromQuery] string groupBy = "day", [FromQuery] int? farmId = null, [FromQuery] int? blockId = null,
        CancellationToken ct = default)
        => Ok(await _service.GetBoardAsync(from, to, groupBy, farmId, blockId, ct));

    /// <summary>Dry-run of every conflict check without saving.</summary>
    [HttpPost("validate"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<ScheduleValidationResultDto>> Validate([FromBody] ResourceScheduleUpsertDto dto,
        [FromQuery] int? scheduleId, CancellationToken ct)
        => Ok(await _service.ValidateAsync(dto, scheduleId, ct));

    [HttpPost, Authorize(Policy = Policies.Schedule)]
    public async Task<ActionResult<ResourceScheduleDto>> Create([FromBody] ResourceScheduleUpsertDto dto, CancellationToken ct)
        => Ok(await _service.CreateAsync(dto, ct));

    [HttpPut("{id:int}"), Authorize(Policy = Policies.Schedule)]
    public async Task<ActionResult<ResourceScheduleDto>> Update(int id, [FromBody] ResourceScheduleUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Schedule)]
    public async Task<IActionResult> Cancel(int id, CancellationToken ct)
    {
        await _service.CancelAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Material requirement planning (sections 12, 13).</summary>
[Route("api/material-requirements")]
public class MaterialRequirementsController : ApiControllerBase
{
    private readonly IMaterialRequirementService _service;
    public MaterialRequirementsController(IMaterialRequirementService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<MaterialRequirementDto>>> Get(
        [FromQuery] MaterialRequirementQuery query, CancellationToken ct)
        => Ok(await _service.GetRequirementsAsync(query, ct));

    /// <summary>Recomputes the stored requirement rows of a projection.</summary>
    [HttpPost("recalculate"), Authorize(Policy = Policies.ManageMaterials)]
    public async Task<ActionResult<object>> Recalculate([FromQuery] int projectionId, CancellationToken ct)
        => Ok(new { projectionId, rowsCreated = await _service.RecalculateAsync(projectionId, ct) });
}

/// <summary>Fuel and labor projections (section 14).</summary>
[Route("api/projections-resources")]
public class FuelLaborController : ApiControllerBase
{
    private readonly IFuelLaborService _service;
    public FuelLaborController(IFuelLaborService service) => _service = service;

    [HttpGet("fuel"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<FuelProjectionDto>>> Fuel([FromQuery] FuelLaborQuery query, CancellationToken ct)
        => Ok(await _service.GetFuelProjectionAsync(query, ct));

    [HttpGet("labor"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<LaborProjectionDto>>> Labor([FromQuery] FuelLaborQuery query, CancellationToken ct)
        => Ok(await _service.GetLaborProjectionAsync(query, ct));
}

/// <summary>Capacity analysis and scenario planning (section 15).</summary>
[Route("api/capacity")]
public class CapacityController : ApiControllerBase
{
    private readonly ICapacityService _capacity;
    private readonly IScenarioService _scenarios;

    public CapacityController(ICapacityService capacity, IScenarioService scenarios)
    {
        _capacity = capacity;
        _scenarios = scenarios;
    }

    [HttpGet("{projectionId:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<CapacityAnalysisDto>> Analyze(int projectionId, CancellationToken ct)
        => Ok(await _capacity.AnalyzeAsync(projectionId, ct));

    [HttpGet("scenarios"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<ScenarioDto>>> Scenarios([FromQuery] int projectionId, CancellationToken ct)
        => Ok(await _scenarios.GetScenariosAsync(projectionId, ct));

    [HttpPost("scenarios"), Authorize(Policy = Policies.Create)]
    public async Task<ActionResult<ScenarioDto>> CreateScenario([FromBody] ScenarioUpsertDto dto, CancellationToken ct)
        => Ok(await _scenarios.CreateAsync(dto, ct));

    [HttpPut("scenarios/{id:int}"), Authorize(Policy = Policies.Edit)]
    public async Task<ActionResult<ScenarioDto>> UpdateScenario(int id, [FromBody] ScenarioUpsertDto dto, CancellationToken ct)
        => Ok(await _scenarios.UpdateAsync(id, dto, ct));

    [HttpDelete("scenarios/{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> DeleteScenario(int id, CancellationToken ct)
    {
        await _scenarios.DeleteAsync(id, ct);
        return NoContent();
    }

    /// <summary>Runs the simulation. The approved plan is never modified.</summary>
    [HttpPost("scenarios/{id:int}/simulate"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<ScenarioResultDto>> Simulate(int id, CancellationToken ct)
        => Ok(await _scenarios.SimulateAsync(id, ct));
}

/// <summary>Actual execution and projection-versus-actual (section 17).</summary>
[Route("api/execution")]
public class ExecutionController : ApiControllerBase
{
    private readonly IExecutionService _service;
    public ExecutionController(IExecutionService service) => _service = service;

    [HttpGet("{activityPlanId:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<ActivityActualDto>> Get(int activityPlanId, CancellationToken ct)
        => Ok(await _service.GetActualAsync(activityPlanId, ct));

    [HttpPost, Authorize(Policy = Policies.RecordActuals)]
    public async Task<ActionResult<ActivityActualDto>> Record([FromBody] ActivityActualUpsertDto dto, CancellationToken ct)
        => Ok(await _service.RecordAsync(dto, ct));

    [HttpGet("compare/{projectionId:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<ProjectionVsActualDto>>> Compare(int projectionId,
        [FromQuery] string groupBy = "block", CancellationToken ct = default)
        => Ok(await _service.CompareAsync(projectionId, groupBy, ct));
}

/// <summary>Dashboard aggregation (section 18).</summary>
[Route("api/dashboard")]
public class DashboardController : ApiControllerBase
{
    private readonly IDashboardService _service;
    public DashboardController(IDashboardService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<DashboardDto>> Get([FromQuery] int? seasonId, [FromQuery] int? estateId, CancellationToken ct)
        => Ok(await _service.GetAsync(seasonId, estateId, ct));
}
