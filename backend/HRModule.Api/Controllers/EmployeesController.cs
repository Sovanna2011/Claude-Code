using System.Security.Claims;
using HRModule.Api.DTOs;
using HRModule.Api.Security;
using HRModule.Api.Services.Interfaces;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace HRModule.Api.Controllers;

/// <summary>
/// Personnel Administration endpoints (SAP PA20/PA30/PA40 equivalents).
/// Requires authentication; write actions are gated by role policy.
/// </summary>
[ApiController]
[Route("api/[controller]")]
[Produces("application/json")]
[Authorize]
public class EmployeesController : ControllerBase
{
    private readonly IEmployeeService _service;
    private readonly ITimeService _time;

    public EmployeesController(IEmployeeService service, ITimeService time)
    {
        _service = service;
        _time = time;
    }

    /// <summary>List employees (PA20 header list). Admin / Manager only.</summary>
    [HttpGet]
    [Authorize(Policy = Policies.TimeKeepers)]
    public async Task<ActionResult<IReadOnlyList<EmployeeSummaryDto>>> GetAll(
        [FromQuery] string? search, [FromQuery] DateTime? keyDate, CancellationToken ct)
        => Ok(await _service.GetEmployeesAsync(search, keyDate, ct));

    /// <summary>Full master data for one employee. Employees may read only their own record.</summary>
    [HttpGet("{pernr:int}")]
    public async Task<ActionResult<EmployeeDetailDto>> Get(int pernr, [FromQuery] DateTime? keyDate, CancellationToken ct)
    {
        if (!CanAccess(pernr)) return Forbid();
        var dto = await _service.GetEmployeeAsync(pernr, keyDate, ct);
        return dto is null ? NotFound(new { message = $"Employee {pernr} not found." }) : Ok(dto);
    }

    /// <summary>Self-service guard: an EMPLOYEE may only access their own PERNR.</summary>
    private bool CanAccess(int pernr)
    {
        if (User.IsInRole(Roles.Admin) || User.IsInRole(Roles.Manager)) return true;
        var own = User.FindFirstValue(JwtTokenService.PernrClaim);
        return own is not null && int.TryParse(own, out var p) && p == pernr;
    }

    /// <summary>Hiring action (IT0000/0001/0002) - creates a new PERNR.</summary>
    [HttpPost("hire")]
    [Authorize(Policy = Policies.AdminOnly)]
    public async Task<ActionResult<HireEmployeeResponse>> Hire([FromBody] HireEmployeeRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var result = await _service.HireAsync(request, ct);
        return CreatedAtAction(nameof(Get), new { pernr = result.Pernr }, result);
    }

    /// <summary>Update Personal Data (IT0002) - creates a new time slice.</summary>
    [HttpPut("{pernr:int}/personaldata")]
    [Authorize(Policy = Policies.AdminOnly)]
    public async Task<IActionResult> UpdatePersonalData(int pernr, [FromBody] UpdatePersonalDataRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _service.UpdatePersonalDataAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Organizational reassignment (delimits IT0001).</summary>
    [HttpPut("{pernr:int}/reassign")]
    [Authorize(Policy = Policies.AdminOnly)]
    public async Task<IActionResult> Reassign(int pernr, [FromBody] ReassignRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _service.ReassignAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Maintain an address (IT0006) - creates a new time slice.</summary>
    [HttpPut("{pernr:int}/address")]
    [Authorize(Policy = Policies.AdminOnly)]
    public async Task<IActionResult> UpdateAddress(int pernr, [FromBody] UpdateAddressRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _service.UpdateAddressAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Add a family member / dependent (IT0021).</summary>
    [HttpPost("{pernr:int}/family")]
    [Authorize(Policy = Policies.AdminOnly)]
    public async Task<IActionResult> AddFamilyMember(int pernr, [FromBody] FamilyMemberRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _service.AddFamilyMemberAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Record an attendance (IT2002).</summary>
    [HttpPost("{pernr:int}/attendances")]
    [Authorize(Policy = Policies.TimeKeepers)]
    public async Task<IActionResult> RecordAttendance(int pernr, [FromBody] AttendanceRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _time.RecordAttendanceAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Leave balances (IT2006). Employees may read only their own.</summary>
    [HttpGet("{pernr:int}/leave-balances")]
    public async Task<ActionResult<IReadOnlyList<LeaveBalanceDto>>> LeaveBalances(int pernr, CancellationToken ct)
        => CanAccess(pernr) ? Ok(await _time.GetLeaveBalancesAsync(pernr, ct)) : Forbid();

    /// <summary>Record an absence (IT2001) and deduct from the matching quota.</summary>
    [HttpPost("{pernr:int}/absences")]
    [Authorize(Policy = Policies.TimeKeepers)]
    public async Task<IActionResult> RecordAbsence(int pernr, [FromBody] AbsenceRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _time.RecordAbsenceAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }
}
