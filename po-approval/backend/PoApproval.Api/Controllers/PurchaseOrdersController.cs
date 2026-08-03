using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using PoApproval.Api.DTOs;
using PoApproval.Api.Security;
using PoApproval.Api.Services.Interfaces;
using PoApproval.Api.Services.Sap;

namespace PoApproval.Api.Controllers;

/// <summary>
/// Purchase order approval endpoints (SAP ME28 worklist / ME29N release).
/// All endpoints require an authenticated user.
/// </summary>
[ApiController]
[Authorize]
[Route("api/purchase-orders")]
[Produces("application/json")]
public class PurchaseOrdersController : ControllerBase
{
    private readonly IPurchaseOrderService _po;
    private readonly IReleaseService _release;

    public PurchaseOrdersController(IPurchaseOrderService po, IReleaseService release)
    {
        _po = po;
        _release = release;
    }

    /// <summary>Approval worklist, optionally filtered.</summary>
    [HttpGet]
    public async Task<ActionResult<IReadOnlyList<PoSummaryDto>>> GetAll(
        [FromQuery] string? search,
        [FromQuery] bool onlyPending,
        [FromQuery] string? purchasingGroup,
        CancellationToken ct)
    {
        var filter = new PoQueryFilter { Search = search, OnlyPending = onlyPending, PurchasingGroup = purchasingGroup };
        return Ok(await _po.GetWorklistAsync(filter, CurrentUser.From(User), ct));
    }

    /// <summary>
    /// Purchase orders pending release, grouped by release strategy (ME28-style).
    /// Set <c>assignedToMe=true</c> to see only the POs awaiting the signed-in
    /// user's release code.
    /// </summary>
    [HttpGet("pending-by-strategy")]
    public async Task<ActionResult<IReadOnlyList<PendingStrategyGroupDto>>> PendingByStrategy(
        [FromQuery] bool assignedToMe, CancellationToken ct)
        => Ok(await _po.GetPendingByStrategyAsync(assignedToMe, CurrentUser.From(User), ct));

    /// <summary>Full detail of one PO, including the release strategy state.</summary>
    [HttpGet("{ebeln}")]
    public async Task<ActionResult<PoDetailDto>> Get(string ebeln, CancellationToken ct)
    {
        var dto = await _po.GetDetailAsync(ebeln, CurrentUser.From(User), ct);
        return dto is null ? NotFound(new { message = $"Purchase order {ebeln} not found." }) : Ok(dto);
    }

    /// <summary>Release (approve) the next pending step of the PO.</summary>
    [HttpPost("{ebeln}/release")]
    public async Task<ActionResult<PoDetailDto>> Release(string ebeln, [FromBody] ReleaseActionRequest request, CancellationToken ct)
    {
        var user = CurrentUser.From(User);
        await _release.ReleaseAsync(ebeln, request ?? new ReleaseActionRequest(), user, ct);
        return Ok(await _po.GetDetailAsync(ebeln, user, ct));
    }

    /// <summary>Reject / cancel the release, resetting the strategy from that code.</summary>
    [HttpPost("{ebeln}/reject")]
    public async Task<ActionResult<PoDetailDto>> Reject(string ebeln, [FromBody] RejectActionRequest request, CancellationToken ct)
    {
        if (!ModelState.IsValid) return BadRequest(ModelState);
        var user = CurrentUser.From(User);
        await _release.RejectAsync(ebeln, request, user, ct);
        return Ok(await _po.GetDetailAsync(ebeln, user, ct));
    }
}
