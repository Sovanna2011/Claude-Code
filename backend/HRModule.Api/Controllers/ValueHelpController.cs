using HRModule.Api.DTOs;
using HRModule.Api.Security;
using HRModule.Api.Services.Interfaces;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace HRModule.Api.Controllers;

/// <summary>Value help (F4) endpoints backing SAPUI5 dropdowns / selects.</summary>
[ApiController]
[Route("api/[controller]")]
[Produces("application/json")]
[Authorize]
public class ValueHelpController : ControllerBase
{
    private readonly IValueHelpService _service;
    public ValueHelpController(IValueHelpService service) => _service = service;

    [HttpGet("company-codes")]
    public async Task<ActionResult<IReadOnlyList<ValueHelpDto>>> CompanyCodes(CancellationToken ct)
        => Ok(await _service.GetCompanyCodesAsync(ct));

    [HttpGet("personnel-areas")]
    public async Task<ActionResult<IReadOnlyList<ValueHelpDto>>> PersonnelAreas(CancellationToken ct)
        => Ok(await _service.GetPersonnelAreasAsync(ct));

    [HttpGet("employee-groups")]
    public async Task<ActionResult<IReadOnlyList<ValueHelpDto>>> EmployeeGroups(CancellationToken ct)
        => Ok(await _service.GetEmployeeGroupsAsync(ct));

    [HttpGet("employee-subgroups")]
    public async Task<ActionResult<IReadOnlyList<ValueHelpDto>>> EmployeeSubgroups(CancellationToken ct)
        => Ok(await _service.GetEmployeeSubgroupsAsync(ct));

    [HttpGet("absence-types")]
    public async Task<ActionResult<IReadOnlyList<ValueHelpDto>>> AbsenceTypes(CancellationToken ct)
        => Ok(await _service.GetAbsenceTypesAsync(ct));

    /// <summary>Generic domain fixed values (e.g. GESCH, FAMST, ANRED, STAT2).</summary>
    [HttpGet("domain/{domain}")]
    public async Task<ActionResult<IReadOnlyList<ValueHelpDto>>> Domain(string domain, CancellationToken ct)
        => Ok(await _service.GetDomainAsync(domain, ct));
}
