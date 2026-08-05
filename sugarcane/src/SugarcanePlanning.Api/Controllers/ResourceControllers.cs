using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Auditing;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Machinery;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Contracts.Reports;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Api.Controllers;

/// <summary>Tractor master (section 8).</summary>
[Route("api/tractors")]
public class TractorsController : ApiControllerBase
{
    private readonly IMachineryService _service;
    public TractorsController(IMachineryService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<TractorDto>>> GetAll([FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetTractorsAsync(q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<TractorDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetTractorAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMachinery)]
    public async Task<ActionResult<TractorDto>> Create([FromBody] TractorUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateTractorAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMachinery)]
    public async Task<ActionResult<TractorDto>> Update(int id, [FromBody] TractorUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateTractorAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteTractorAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Equipment master and the tractor compatibility matrix (section 9).</summary>
[Route("api/equipment")]
public class EquipmentController : ApiControllerBase
{
    private readonly IMachineryService _service;
    public EquipmentController(IMachineryService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<EquipmentDto>>> GetAll([FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetEquipmentAsync(q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<EquipmentDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetEquipmentItemAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMachinery)]
    public async Task<ActionResult<EquipmentDto>> Create([FromBody] EquipmentUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateEquipmentAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMachinery)]
    public async Task<ActionResult<EquipmentDto>> Update(int id, [FromBody] EquipmentUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateEquipmentAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteEquipmentAsync(id, ct);
        return NoContent();
    }

    [HttpGet("compatibility"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<CompatibilityDto>>> Compatibility([FromQuery] int? tractorId,
        [FromQuery] int? equipmentId, CancellationToken ct)
        => Ok(await _service.GetCompatibilitiesAsync(tractorId, equipmentId, ct));

    [HttpPost("compatibility"), Authorize(Policy = Policies.ManageMachinery)]
    public async Task<ActionResult<CompatibilityDto>> AddCompatibility([FromBody] CompatibilityUpsertDto dto, CancellationToken ct)
        => Ok(await _service.AddCompatibilityAsync(dto, ct));

    [HttpDelete("compatibility/{id:int}"), Authorize(Policy = Policies.ManageMachinery)]
    public async Task<IActionResult> RemoveCompatibility(int id, CancellationToken ct)
    {
        await _service.RemoveCompatibilityAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Operators and work teams (section 14).</summary>
[Route("api/workforce")]
public class WorkforceController : ApiControllerBase
{
    private readonly IMachineryService _service;
    public WorkforceController(IMachineryService service) => _service = service;

    [HttpGet("operators"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<OperatorDto>>> Operators([FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetOperatorsAsync(q, ct));

    [HttpPost("operators"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<OperatorDto>> CreateOperator([FromBody] OperatorUpsertDto dto, CancellationToken ct)
        => Ok(await _service.CreateOperatorAsync(dto, ct));

    [HttpPut("operators/{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<OperatorDto>> UpdateOperator(int id, [FromBody] OperatorUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateOperatorAsync(id, dto, ct));

    [HttpDelete("operators/{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> DeleteOperator(int id, CancellationToken ct)
    {
        await _service.DeleteOperatorAsync(id, ct);
        return NoContent();
    }

    [HttpGet("teams"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<WorkTeamDto>>> Teams([FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetWorkTeamsAsync(q, ct));

    [HttpPost("teams"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<WorkTeamDto>> CreateTeam([FromBody] WorkTeamUpsertDto dto, CancellationToken ct)
        => Ok(await _service.CreateWorkTeamAsync(dto, ct));

    [HttpPut("teams/{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<WorkTeamDto>> UpdateTeam(int id, [FromBody] WorkTeamUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateWorkTeamAsync(id, dto, ct));

    [HttpDelete("teams/{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> DeleteTeam(int id, CancellationToken ct)
    {
        await _service.DeleteWorkTeamAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Material master, activity standards and the stock interface (sections 11-13).</summary>
[Route("api/materials")]
public class MaterialsController : ApiControllerBase
{
    private readonly IMaterialMasterService _service;
    public MaterialsController(IMaterialMasterService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<MaterialDto>>> GetAll([FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetMaterialsAsync(q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<MaterialDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetMaterialAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMaterials)]
    public async Task<ActionResult<MaterialDto>> Create([FromBody] MaterialUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateMaterialAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMaterials)]
    public async Task<ActionResult<MaterialDto>> Update(int id, [FromBody] MaterialUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateMaterialAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteMaterialAsync(id, ct);
        return NoContent();
    }

    [HttpGet("standards"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<ActivityMaterialStandardDto>>> Standards([FromQuery] int? activityId,
        [FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetStandardsAsync(activityId, q, ct));

    [HttpPost("standards"), Authorize(Policy = Policies.ManageMaterials)]
    public async Task<ActionResult<ActivityMaterialStandardDto>> CreateStandard(
        [FromBody] ActivityMaterialStandardUpsertDto dto, CancellationToken ct)
        => Ok(await _service.CreateStandardAsync(dto, ct));

    [HttpPut("standards/{id:int}"), Authorize(Policy = Policies.ManageMaterials)]
    public async Task<ActionResult<ActivityMaterialStandardDto>> UpdateStandard(int id,
        [FromBody] ActivityMaterialStandardUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateStandardAsync(id, dto, ct));

    [HttpDelete("standards/{id:int}"), Authorize(Policy = Policies.ManageMaterials)]
    public async Task<IActionResult> DeleteStandard(int id, CancellationToken ct)
    {
        await _service.DeleteStandardAsync(id, ct);
        return NoContent();
    }

    [HttpGet("stock"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<MaterialStockDto>>> Stock([FromQuery] int? estateId, CancellationToken ct)
        => Ok(await _service.GetStocksAsync(estateId, ct));

    /// <summary>Inbound stock interface (section 13). This system never posts movements back.</summary>
    [HttpPost("stock/sync"), Authorize(Policy = Policies.ManageMaterials)]
    public async Task<ActionResult<object>> SyncStock([FromBody] List<MaterialStockSyncDto> rows, CancellationToken ct)
        => Ok(new { applied = await _service.SyncStockAsync(rows, ct) });
}

/// <summary>Reports with print, PDF and Excel export (section 20).</summary>
[Route("api/reports")]
public class ReportsController : ApiControllerBase
{
    private readonly IReportService _reports;
    private readonly IEnumerable<IReportExporter> _exporters;
    private readonly IAuditService _audit;

    public ReportsController(IReportService reports, IEnumerable<IReportExporter> exporters, IAuditService audit)
    {
        _reports = reports;
        _exporters = exporters;
        _audit = audit;
    }

    /// <summary>Report data for the on-screen grid and the print preview.</summary>
    [HttpGet("{key}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<ReportResultDto>> Get(ReportKey key, [FromQuery] ReportRequest request, CancellationToken ct)
        => Ok(await _reports.GenerateAsync(key, request, ct));

    /// <summary>Same report as a downloadable PDF or Excel file.</summary>
    [HttpGet("{key}/export"), Authorize(Policy = Policies.Export)]
    public async Task<IActionResult> Export(ReportKey key, [FromQuery] ExportFormat format,
        [FromQuery] ReportRequest request, CancellationToken ct)
    {
        var exporter = _exporters.FirstOrDefault(e => e.Format == format)
                       ?? throw new Domain.Common.NotFoundException($"No exporter for format '{format}'.");

        var report = await _reports.GenerateAsync(key, request, ct);
        var bytes = exporter.Export(report);

        await _audit.LogAsync(AuditAction.Export, "Report", key.ToString(), null, format.ToString(), null, ct);

        var fileName = $"{key}-{DateTime.UtcNow:yyyyMMdd-HHmm}.{exporter.FileExtension}";
        return File(bytes, exporter.ContentType, fileName);
    }
}

/// <summary>Audit trail (section 22).</summary>
[Route("api/audit")]
public class AuditController : ApiControllerBase
{
    private readonly IAuditQueryService _service;
    public AuditController(IAuditQueryService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.Administer)]
    public async Task<ActionResult<PagedResult<AuditLogDto>>> Query([FromQuery] AuditLogQuery query, CancellationToken ct)
        => Ok(await _service.QueryAsync(query, ct));
}
