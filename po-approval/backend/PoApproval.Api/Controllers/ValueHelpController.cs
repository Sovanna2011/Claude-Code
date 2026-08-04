using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using PoApproval.Api.DTOs;
using PoApproval.Api.Services.Interfaces;

namespace PoApproval.Api.Controllers;

/// <summary>Reference-data value helps (F4) for the UI5 front end.</summary>
[ApiController]
[Authorize]
[Route("api/valuehelp")]
[Produces("application/json")]
public class ValueHelpController : ControllerBase
{
    private readonly IValueHelpService _service;

    public ValueHelpController(IValueHelpService service) => _service = service;

    /// <summary>
    /// Named value help, e.g. purchasing-groups, purchasing-orgs, doc-types,
    /// release-groups, release-strategies, release-indicators, vendors.
    /// </summary>
    [HttpGet("{name}")]
    public async Task<ActionResult<IReadOnlyList<ValueHelpItemDto>>> Get(string name, CancellationToken ct)
        => Ok(await _service.GetAsync(name, ct));
}
