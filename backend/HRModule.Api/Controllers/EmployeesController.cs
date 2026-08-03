using HRModule.Api.DTOs;
using HRModule.Api.Services.Interfaces;
using Microsoft.AspNetCore.Mvc;

namespace HRModule.Api.Controllers;

/// <summary>
/// Personnel Administration endpoints (SAP PA20/PA30/PA40 equivalents).
/// </summary>
[ApiController]
[Route("api/[controller]")]
[Produces("application/json")]
public class EmployeesController : ControllerBase
{
    private readonly IEmployeeService _service;
    private readonly ITimeService _time;

    public EmployeesController(IEmployeeService service, ITimeService time)
    {
        _service = service;
        _time = time;
    }

    /// <summary>List employees (PA20 header list), optionally filtered by name/PERNR.</summary>
    [HttpGet]
    public async Task<ActionResult<IReadOnlyList<EmployeeSummaryDto>>> GetAll(
        [FromQuery] string? search, [FromQuery] DateTime? keyDate, CancellationToken ct)
        => Ok(await _service.GetEmployeesAsync(search, keyDate, ct));

    /// <summary>Full master data for one employee valid on a key date.</summary>
    [HttpGet("{pernr:int}")]
    public async Task<ActionResult<EmployeeDetailDto>> Get(int pernr, [FromQuery] DateTime? keyDate, CancellationToken ct)
    {
        var dto = await _service.GetEmployeeAsync(pernr, keyDate, ct);
        return dto is null ? NotFound(new { message = $"Employee {pernr} not found." }) : Ok(dto);
    }

    /// <summary>Hiring action (IT0000/0001/0002) - creates a new PERNR.</summary>
    [HttpPost("hire")]
    public async Task<ActionResult<HireEmployeeResponse>> Hire([FromBody] HireEmployeeRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var result = await _service.HireAsync(request, ct);
        return CreatedAtAction(nameof(Get), new { pernr = result.Pernr }, result);
    }

    /// <summary>Update Personal Data (IT0002) - creates a new time slice.</summary>
    [HttpPut("{pernr:int}/personaldata")]
    public async Task<IActionResult> UpdatePersonalData(int pernr, [FromBody] UpdatePersonalDataRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _service.UpdatePersonalDataAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Organizational reassignment (delimits IT0001).</summary>
    [HttpPut("{pernr:int}/reassign")]
    public async Task<IActionResult> Reassign(int pernr, [FromBody] ReassignRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _service.ReassignAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Maintain an address (IT0006) - creates a new time slice.</summary>
    [HttpPut("{pernr:int}/address")]
    public async Task<IActionResult> UpdateAddress(int pernr, [FromBody] UpdateAddressRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _service.UpdateAddressAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Add a family member / dependent (IT0021).</summary>
    [HttpPost("{pernr:int}/family")]
    public async Task<IActionResult> AddFamilyMember(int pernr, [FromBody] FamilyMemberRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _service.AddFamilyMemberAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Record an attendance (IT2002).</summary>
    [HttpPost("{pernr:int}/attendances")]
    public async Task<IActionResult> RecordAttendance(int pernr, [FromBody] AttendanceRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _time.RecordAttendanceAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }

    /// <summary>Leave balances (IT2006) for the employee.</summary>
    [HttpGet("{pernr:int}/leave-balances")]
    public async Task<ActionResult<IReadOnlyList<LeaveBalanceDto>>> LeaveBalances(int pernr, CancellationToken ct)
        => Ok(await _time.GetLeaveBalancesAsync(pernr, ct));

    /// <summary>Record an absence (IT2001) and deduct from the matching quota.</summary>
    [HttpPost("{pernr:int}/absences")]
    public async Task<IActionResult> RecordAbsence(int pernr, [FromBody] AbsenceRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var ok = await _time.RecordAbsenceAsync(pernr, request, ct);
        return ok ? NoContent() : NotFound(new { message = $"Employee {pernr} not found." });
    }
}
