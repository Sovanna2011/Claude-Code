using Microsoft.EntityFrameworkCore;
using PoApproval.Api.Data;
using PoApproval.Api.DTOs;
using PoApproval.Api.Models;
using PoApproval.Api.Security;
using PoApproval.Api.Services.Interfaces;
using PoApproval.Api.Services.Sap;

namespace PoApproval.Api.Services;

/// <summary>
/// Release-procedure business logic (SAP ME29N equivalent). Enforces:
///  * sequential sign-off - only the next pending code may be released,
///  * authorization - the user must hold the release code (M_EINK_FRG),
///  * completion - when the last step signs off the PO becomes released.
/// Persistence goes through <see cref="ISapEccConnector"/> so the same rules
/// apply whether the backing store is the SQL replica or a live ECC via RFC.
/// </summary>
public class ReleaseService : IReleaseService
{
    private const string HighFrgke = "R";   // released
    private readonly ISapEccConnector _ecc;
    private readonly PoDbContext _db;

    public ReleaseService(ISapEccConnector ecc, PoDbContext db)
    {
        _ecc = ecc;
        _db = db;
    }

    public async Task ReleaseAsync(string ebeln, ReleaseActionRequest request, CurrentUser user, CancellationToken ct = default)
    {
        var k = await _ecc.GetHeaderAsync(ebeln, forUpdate: true, ct)
                ?? throw new KeyNotFoundException($"Purchase order {ebeln} not found.");

        if (k.FRGSX is null || k.FRGGR is null)
            throw new InvalidOperationException("Purchase order is not subject to release.");
        if (k.FRGKE == HighFrgke)
            throw new InvalidOperationException("Purchase order is already fully released.");

        var steps = await StepsAsync(k.FRGGR, k.FRGSX, ct);
        var released = ReleaseCodes.Parse(k.FRGZU);
        var next = steps.FirstOrDefault(s => !released.Contains(s.FRGCO))
                   ?? throw new InvalidOperationException("No pending release step.");

        var code = request.Code ?? next.FRGCO;
        if (code != next.FRGCO)
            throw new InvalidOperationException($"Release code {code} is not the next pending step ({next.FRGCO}).");

        await EnsureUserHoldsCodeAsync(user, k.FRGGR, code, ct);

        // Effect the release.
        k.FRGZU += code;
        var remaining = steps.Count(s => !ReleaseCodes.Parse(k.FRGZU).Contains(s.FRGCO));
        k.FRGKE = remaining == 0 ? HighFrgke : "B";
        k.FRGRL = remaining == 0 ? " " : "X";
        k.AEDAT = DateTime.Today;

        await _ecc.PostReleaseActionAsync(k, new ReleaseLog
        {
            EBELN = ebeln, FRGCO = code, ActionTyp = "RELEASE",
            UNAME = user.UserName, ActedOn = DateTime.Now, Note = request.Note
        }, ct);
    }

    public async Task RejectAsync(string ebeln, RejectActionRequest request, CurrentUser user, CancellationToken ct = default)
    {
        var k = await _ecc.GetHeaderAsync(ebeln, forUpdate: true, ct)
                ?? throw new KeyNotFoundException($"Purchase order {ebeln} not found.");

        if (k.FRGSX is null || k.FRGGR is null)
            throw new InvalidOperationException("Purchase order is not subject to release.");

        var steps = await StepsAsync(k.FRGGR, k.FRGSX, ct);
        var released = ReleaseCodes.Parse(k.FRGZU);
        var nextPending = steps.FirstOrDefault(s => !released.Contains(s.FRGCO));

        // Default to rejecting the current pending step.
        var code = request.Code ?? nextPending?.FRGCO
                   ?? throw new InvalidOperationException("Nothing to reject; the PO is fully released.");
        var step = steps.FirstOrDefault(s => s.FRGCO == code)
                   ?? throw new InvalidOperationException($"Release code {code} is not part of the strategy.");

        await EnsureUserHoldsCodeAsync(user, k.FRGGR, code, ct);

        // Reset: keep only codes of steps preceding the rejected step.
        k.FRGZU = string.Concat(steps.Where(s => s.StepNo < step.StepNo).Select(s => s.FRGCO));
        k.FRGKE = "B";
        k.FRGRL = "X";
        k.AEDAT = DateTime.Today;

        await _ecc.PostReleaseActionAsync(k, new ReleaseLog
        {
            EBELN = ebeln, FRGCO = code, ActionTyp = "REJECT",
            UNAME = user.UserName, ActedOn = DateTime.Now, Note = request.Note
        }, ct);
    }

    private async Task<List<T16FSCode>> StepsAsync(string frggr, string frgsx, CancellationToken ct)
        => await _db.T16FSCode.AsNoTracking()
            .Where(s => s.FRGGR == frggr && s.FRGSX == frgsx)
            .OrderBy(s => s.StepNo)
            .ToListAsync(ct);

    private async Task EnsureUserHoldsCodeAsync(CurrentUser user, string frggr, string code, CancellationToken ct)
    {
        var holds = await _db.AppUserReleaseCodes.AsNoTracking()
            .AnyAsync(c => c.UserId == user.UserId && c.FRGGR == frggr && c.FRGCO == code, ct);
        if (!holds)
            throw new UnauthorizedAccessException(
                $"You do not hold release code {code} in group {frggr}.");
    }
}
