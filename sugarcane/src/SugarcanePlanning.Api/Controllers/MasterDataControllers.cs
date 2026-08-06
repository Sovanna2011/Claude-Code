using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Organization;
using SugarcanePlanning.Contracts.Seasons;

namespace SugarcanePlanning.Api.Controllers;

/// <summary>Base class: JSON only, authenticated, standard response documentation.</summary>
[ApiController]
[Authorize]
[Produces("application/json")]
[ProducesResponseType(typeof(ApiErrorDto), StatusCodes.Status401Unauthorized)]
[ProducesResponseType(typeof(ApiErrorDto), StatusCodes.Status403Forbidden)]
[ProducesResponseType(typeof(ApiErrorDto), StatusCodes.Status422UnprocessableEntity)]
public abstract class ApiControllerBase : ControllerBase
{
}

/// <summary>Companies (section 3).</summary>
[Route("api/companies")]
public class CompaniesController : ApiControllerBase
{
    private readonly ILandStructureService _service;
    public CompaniesController(ILandStructureService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<CompanyDto>>> GetAll([FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetCompaniesAsync(q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<CompanyDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetCompanyAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.Administer)]
    public async Task<ActionResult<CompanyDto>> Create([FromBody] CompanyUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateCompanyAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.Administer)]
    public async Task<ActionResult<CompanyDto>> Update(int id, [FromBody] CompanyUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateCompanyAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Administer)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteCompanyAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Estates (section 3).</summary>
[Route("api/estates")]
public class EstatesController : ApiControllerBase
{
    private readonly ILandStructureService _service;
    public EstatesController(ILandStructureService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<EstateDto>>> GetAll([FromQuery] int? companyId,
        [FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetEstatesAsync(companyId, q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<EstateDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetEstateAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<EstateDto>> Create([FromBody] EstateUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateEstateAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<EstateDto>> Update(int id, [FromBody] EstateUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateEstateAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteEstateAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Farms (section 3).</summary>
[Route("api/farms")]
public class FarmsController : ApiControllerBase
{
    private readonly ILandStructureService _service;
    public FarmsController(ILandStructureService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<FarmDto>>> GetAll([FromQuery] int? estateId,
        [FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetFarmsAsync(estateId, q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<FarmDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetFarmAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<FarmDto>> Create([FromBody] FarmUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateFarmAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<FarmDto>> Update(int id, [FromBody] FarmUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateFarmAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteFarmAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Zones (section 3).</summary>
[Route("api/zones")]
public class ZonesController : ApiControllerBase
{
    private readonly ILandStructureService _service;
    public ZonesController(ILandStructureService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<ZoneDto>>> GetAll([FromQuery] int? farmId,
        [FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetZonesAsync(farmId, q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<ZoneDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetZoneAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<ZoneDto>> Create([FromBody] ZoneUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateZoneAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<ZoneDto>> Update(int id, [FromBody] ZoneUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateZoneAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteZoneAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Plantation blocks and the land-structure tree (section 3).</summary>
[Route("api/blocks")]
public class BlocksController : ApiControllerBase
{
    private readonly ILandStructureService _service;
    public BlocksController(ILandStructureService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<PlantationBlockDto>>> GetAll([FromQuery] int? zoneId,
        [FromQuery] int? farmId, [FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetBlocksAsync(zoneId, farmId, q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PlantationBlockDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetBlockAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<PlantationBlockDto>> Create([FromBody] PlantationBlockUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateBlockAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<PlantationBlockDto>> Update(int id, [FromBody] PlantationBlockUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateBlockAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteBlockAsync(id, ct);
        return NoContent();
    }

    /// <summary>Company → estate → farm → zone → block, for the land-structure screen.</summary>
    [HttpGet("/api/land-structure"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<LandStructureNodeDto>>> Structure([FromQuery] int? companyId, CancellationToken ct)
        => Ok(await _service.GetStructureAsync(companyId, ct));
}

/// <summary>Growing seasons (section 4).</summary>
[Route("api/seasons")]
public class SeasonsController : ApiControllerBase
{
    private readonly ISeasonService _service;
    public SeasonsController(ISeasonService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<GrowingSeasonDto>>> GetAll([FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetSeasonsAsync(q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<GrowingSeasonDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetSeasonAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<GrowingSeasonDto>> Create([FromBody] GrowingSeasonUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateSeasonAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<GrowingSeasonDto>> Update(int id, [FromBody] GrowingSeasonUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateSeasonAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteSeasonAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Sugarcane varieties (section 4).</summary>
[Route("api/varieties")]
public class VarietiesController : ApiControllerBase
{
    private readonly ISeasonService _service;
    public VarietiesController(ISeasonService service) => _service = service;

    [HttpGet, Authorize(Policy = Policies.View)]
    public async Task<ActionResult<PagedResult<CaneVarietyDto>>> GetAll([FromQuery] QueryParameters q, CancellationToken ct)
        => Ok(await _service.GetVarietiesAsync(q, ct));

    [HttpGet("{id:int}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<CaneVarietyDto>> Get(int id, CancellationToken ct)
        => Ok(await _service.GetVarietyAsync(id, ct));

    [HttpPost, Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<CaneVarietyDto>> Create([FromBody] CaneVarietyUpsertDto dto, CancellationToken ct)
    {
        var created = await _service.CreateVarietyAsync(dto, ct);
        return CreatedAtAction(nameof(Get), new { id = created.Id }, created);
    }

    [HttpPut("{id:int}"), Authorize(Policy = Policies.ManageMasterData)]
    public async Task<ActionResult<CaneVarietyDto>> Update(int id, [FromBody] CaneVarietyUpsertDto dto, CancellationToken ct)
        => Ok(await _service.UpdateVarietyAsync(id, dto, ct));

    [HttpDelete("{id:int}"), Authorize(Policy = Policies.Delete)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _service.DeleteVarietyAsync(id, ct);
        return NoContent();
    }
}

/// <summary>Value helps for cascading dropdowns.</summary>
[Route("api/lookups")]
public class LookupsController : ApiControllerBase
{
    private readonly ILookupService _service;
    public LookupsController(ILookupService service) => _service = service;

    /// <summary>kind = companies | estates | farms | zones | blocks | seasons | varieties | activities | materials | tractors | equipment | operators | workteams | projections.</summary>
    [HttpGet("{kind}"), Authorize(Policy = Policies.View)]
    public async Task<ActionResult<IReadOnlyList<LookupDto>>> Get(string kind, [FromQuery] int? parentId, CancellationToken ct)
        => Ok(await _service.GetAsync(kind, parentId, ct));
}
