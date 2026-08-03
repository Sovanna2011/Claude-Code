using HRModule.Api.DTOs;
using HRModule.Api.Security;
using HRModule.Api.Services.Interfaces;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace HRModule.Api.Controllers;

/// <summary>Organizational Management endpoints (SAP PPOME/PPOSE equivalents).</summary>
[ApiController]
[Route("api/[controller]")]
[Produces("application/json")]
[Authorize(Policy = Policies.TimeKeepers)]
public class OrgUnitsController : ControllerBase
{
    private readonly IOrgService _service;
    public OrgUnitsController(IOrgService service) => _service = service;

    /// <summary>Flat list of org units.</summary>
    [HttpGet]
    public async Task<ActionResult<IReadOnlyList<OrgUnitNodeDto>>> GetAll([FromQuery] DateTime? keyDate, CancellationToken ct)
        => Ok(await _service.GetOrgUnitsFlatAsync(keyDate, ct));

    /// <summary>Org hierarchy starting at the given root org unit.</summary>
    [HttpGet("{rootOrgId:int}/structure")]
    public async Task<ActionResult<OrgUnitNodeDto>> GetStructure(int rootOrgId, [FromQuery] DateTime? keyDate, CancellationToken ct)
    {
        var node = await _service.GetOrgStructureAsync(rootOrgId, keyDate, ct);
        return node is null ? NotFound(new { message = $"Org unit {rootOrgId} not found." }) : Ok(node);
    }

    /// <summary>Positions, optionally filtered by org unit, with holder info.</summary>
    [HttpGet("positions")]
    public async Task<ActionResult<IReadOnlyList<PositionDto>>> GetPositions(
        [FromQuery] int? orgUnitId, [FromQuery] DateTime? keyDate, CancellationToken ct)
        => Ok(await _service.GetPositionsAsync(orgUnitId, keyDate, ct));
}
