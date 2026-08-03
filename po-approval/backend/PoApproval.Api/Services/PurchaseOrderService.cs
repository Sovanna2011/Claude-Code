using Microsoft.EntityFrameworkCore;
using PoApproval.Api.Data;
using PoApproval.Api.DTOs;
using PoApproval.Api.Models;
using PoApproval.Api.Security;
using PoApproval.Api.Services.Interfaces;
using PoApproval.Api.Services.Sap;

namespace PoApproval.Api.Services;

/// <summary>
/// Read model for the approval worklist and PO detail. Purchase-order data is
/// fetched through <see cref="ISapEccConnector"/> (the ECC boundary); the
/// customizing/master texts and the release-strategy steps are resolved from
/// the local tables, exactly as the SAP GUI enriches EKKO with T16Fx texts.
/// </summary>
public class PurchaseOrderService : IPurchaseOrderService
{
    private readonly ISapEccConnector _ecc;
    private readonly PoDbContext _db;

    public PurchaseOrderService(ISapEccConnector ecc, PoDbContext db)
    {
        _ecc = ecc;
        _db = db;
    }

    public async Task<IReadOnlyList<PoSummaryDto>> GetWorklistAsync(
        PoQueryFilter filter, CurrentUser user, CancellationToken ct = default)
    {
        var headers = await _ecc.GetHeadersAsync(filter, ct);
        return await BuildSummariesAsync(headers, user, ct);
    }

    public async Task<IReadOnlyList<PendingStrategyGroupDto>> GetPendingByStrategyAsync(
        bool assignedToMe, CurrentUser user, CancellationToken ct = default)
    {
        var headers = await _ecc.GetHeadersAsync(new PoQueryFilter { OnlyPending = true }, ct);
        var summaries = await BuildSummariesAsync(headers, user, ct);
        var headerByEbeln = headers.ToDictionary(h => h.EBELN);

        var strategies = await _db.T16FS.AsNoTracking().ToListAsync(ct);
        var groupTexts = await _db.T16FG.AsNoTracking().ToDictionaryAsync(g => g.FRGGR, g => g.FRGGT, ct);
        var steps = await LoadStepsAsync(ct);
        var codeTexts = await CodeTextsAsync(ct);

        var result = new List<PendingStrategyGroupDto>();
        foreach (var strat in strategies.OrderBy(s => s.FRGGR).ThenBy(s => s.FRGSX))
        {
            var pos = summaries
                .Where(s => headerByEbeln[s.Ebeln].FRGGR == strat.FRGGR
                         && headerByEbeln[s.Ebeln].FRGSX == strat.FRGSX)
                .ToList();
            if (assignedToMe) pos = pos.Where(p => p.CanCurrentUserRelease).ToList();
            if (pos.Count == 0) continue;

            var codes = steps.GetValueOrDefault((strat.FRGGR, strat.FRGSX), new())
                .Select(st => new ReleaseCodeDto
                {
                    Group = strat.FRGGR,
                    Code = st.FRGCO,
                    Text = codeTexts.GetValueOrDefault((strat.FRGGR, st.FRGCO))
                }).ToList();

            result.Add(new PendingStrategyGroupDto
            {
                ReleaseGroup = strat.FRGGR,
                ReleaseGroupText = groupTexts.GetValueOrDefault(strat.FRGGR),
                Strategy = strat.FRGSX,
                StrategyText = strat.FRGST,
                ValueFrom = strat.ValFrom,
                ValueTo = strat.ValTo,
                Codes = codes,
                Count = pos.Count,
                TotalNetValue = pos.Sum(p => p.NetValue),
                Currency = pos[0].Currency,
                PurchaseOrders = pos
            });
        }
        return result;
    }

    private async Task<List<PoSummaryDto>> BuildSummariesAsync(
        IReadOnlyList<Ekko> headers, CurrentUser user, CancellationToken ct)
    {
        var steps = await LoadStepsAsync(ct);
        var codeTexts = await CodeTextsAsync(ct);
        var strategyTexts = await _db.T16FS.AsNoTracking()
            .ToDictionaryAsync(s => (s.FRGGR, s.FRGSX), s => s.FRGST, ct);
        var groupNames = await _db.T024.AsNoTracking().ToDictionaryAsync(g => g.EKGRP, g => g.EKNAM, ct);
        var indicatorTexts = await IndicatorTextsAsync(ct);
        var userCodes = await UserCodesAsync(user.UserId, ct);

        return headers.Select(k =>
        {
            var (nextCode, _) = NextPending(k, steps);
            return new PoSummaryDto
            {
                Ebeln = k.EBELN,
                DocType = k.BSART,
                VendorId = k.LIFNR,
                VendorName = k.Vendor?.NAME1,
                PurchasingGroup = k.EKGRP,
                PurchasingGroupName = groupNames.GetValueOrDefault(k.EKGRP),
                Currency = k.WAERS,
                NetValue = k.RLWRT,
                DocDate = k.BEDAT,
                Strategy = k.FRGSX,
                StrategyText = k.FRGGR is not null && k.FRGSX is not null
                    ? strategyTexts.GetValueOrDefault((k.FRGGR, k.FRGSX)) : null,
                ReleaseIndicator = k.FRGKE,
                ReleaseIndicatorText = indicatorTexts.GetValueOrDefault(k.FRGKE),
                ReleaseIncomplete = k.FRGRL == "X",
                NextPendingCode = nextCode,
                NextPendingCodeText = nextCode is not null && k.FRGGR is not null
                    ? codeTexts.GetValueOrDefault((k.FRGGR, nextCode)) : null,
                CanCurrentUserRelease = CanRelease(k, nextCode, userCodes)
            };
        }).ToList();
    }

    public async Task<PoDetailDto?> GetDetailAsync(string ebeln, CurrentUser user, CancellationToken ct = default)
    {
        var k = await _ecc.GetHeaderAsync(ebeln, forUpdate: false, ct);
        if (k is null) return null;

        var items = await _ecc.GetItemsAsync(ebeln, ct);
        var steps = await LoadStepsAsync(ct);
        var codeTexts = await CodeTextsAsync(ct);
        var indicatorTexts = await IndicatorTextsAsync(ct);
        var userCodes = await UserCodesAsync(user.UserId, ct);
        var matGroupTexts = await _db.T023T.AsNoTracking().ToDictionaryAsync(m => m.MATKL, m => m.WGBEZ, ct);
        var plantNames = await _db.T001W.AsNoTracking().ToDictionaryAsync(p => p.WERKS, p => p.NAME1, ct);

        var groupName = await _db.T024.Where(g => g.EKGRP == k.EKGRP).Select(g => g.EKNAM).FirstOrDefaultAsync(ct);
        var orgName = await _db.T024E.Where(o => o.EKORG == k.EKORG).Select(o => o.EKOTX).FirstOrDefaultAsync(ct);
        var docTypeText = await _db.T161.Where(t => t.BSTYP == k.BSTYP && t.BSART == k.BSART).Select(t => t.BATXT).FirstOrDefaultAsync(ct);
        var groupText = k.FRGGR is null ? null
            : await _db.T16FG.Where(g => g.FRGGR == k.FRGGR).Select(g => g.FRGGT).FirstOrDefaultAsync(ct);
        var strategyText = (k.FRGGR is null || k.FRGSX is null) ? null
            : await _db.T16FS.Where(s => s.FRGGR == k.FRGGR && s.FRGSX == k.FRGSX).Select(s => s.FRGST).FirstOrDefaultAsync(ct);

        var (nextCode, _) = NextPending(k, steps);

        // Release-step view with per-code audit (last RELEASE per code).
        var logs = await _db.ReleaseLog.AsNoTracking()
            .Where(l => l.EBELN == ebeln)
            .OrderByDescending(l => l.ActedOn)
            .ToListAsync(ct);
        var released = ReleaseCodes.Parse(k.FRGZU);

        var stepList = steps.GetValueOrDefault((k.FRGGR ?? "", k.FRGSX ?? ""), new())
            .Select(st =>
            {
                var isReleased = released.Contains(st.FRGCO);
                var log = logs.FirstOrDefault(l => l.FRGCO == st.FRGCO && l.ActionTyp == "RELEASE");
                return new ReleaseStepDto
                {
                    StepNo = st.StepNo,
                    Code = st.FRGCO,
                    CodeText = k.FRGGR is not null ? codeTexts.GetValueOrDefault((k.FRGGR, st.FRGCO)) : null,
                    IsReleased = isReleased,
                    IsCurrentStep = st.FRGCO == nextCode,
                    ReleasedBy = isReleased ? log?.UNAME : null,
                    ReleasedOn = isReleased ? log?.ActedOn : null
                };
            })
            .ToList();

        var canRelease = CanRelease(k, nextCode, userCodes);

        return new PoDetailDto
        {
            Ebeln = k.EBELN,
            DocType = k.BSART,
            DocTypeText = docTypeText,
            CompanyCode = k.BUKRS,
            PurchasingOrg = k.EKORG,
            PurchasingOrgName = orgName,
            PurchasingGroup = k.EKGRP,
            PurchasingGroupName = groupName,
            Currency = k.WAERS,
            DocDate = k.BEDAT,
            NetValue = k.RLWRT,
            CreatedBy = k.ERNAM,
            Vendor = new VendorDto
            {
                VendorId = k.LIFNR,
                Name = k.Vendor?.NAME1 ?? k.LIFNR,
                City = k.Vendor?.ORT01,
                Country = k.Vendor?.LAND1,
                VatNumber = k.Vendor?.STCEG
            },
            Items = items.Select(i => new PoItemDto
            {
                ItemNo = i.EBELP,
                ShortText = i.TXZ01,
                Material = i.MATNR,
                MaterialGroup = i.MATKL,
                MaterialGroupText = i.MATKL is not null ? matGroupTexts.GetValueOrDefault(i.MATKL) : null,
                Plant = i.WERKS,
                PlantName = i.WERKS is not null ? plantNames.GetValueOrDefault(i.WERKS) : null,
                StorageLocation = i.LGORT,
                Quantity = i.MENGE,
                Unit = i.MEINS,
                NetPrice = i.NETPR,
                PriceUnit = i.PEINH,
                NetValue = i.NETWR,
                GrossValue = i.BRTWR,
                TaxCode = i.MWSKZ,
                DeliveryDate = i.EINDT,
                Deleted = i.LOEKZ == "X"
            }).ToList(),
            ReleaseGroup = k.FRGGR,
            ReleaseGroupText = groupText,
            Strategy = k.FRGSX,
            StrategyText = strategyText,
            ReleaseIndicator = k.FRGKE,
            ReleaseIndicatorText = indicatorTexts.GetValueOrDefault(k.FRGKE),
            ReleaseIncomplete = k.FRGRL == "X",
            ReleaseSteps = stepList,
            ReleaseLog = logs.Select(l => new ReleaseLogDto
            {
                Code = l.FRGCO,
                Action = l.ActionTyp,
                User = l.UNAME,
                ActedOn = l.ActedOn,
                Note = l.Note
            }).ToList(),
            NextPendingCode = nextCode,
            CanCurrentUserRelease = canRelease,
            CanCurrentUserReject = canRelease
        };
    }

    // ---- helpers ------------------------------------------------------------

    /// <summary>Next pending release code for a header, honouring step order.</summary>
    private static (string? code, int? step) NextPending(
        Ekko k, IReadOnlyDictionary<(string, string), List<T16FSCode>> steps)
    {
        if (k.FRGGR is null || k.FRGSX is null || k.FRGKE == "R" || k.FRGKE == " ")
            return (null, null);
        var released = ReleaseCodes.Parse(k.FRGZU);
        var strat = steps.GetValueOrDefault((k.FRGGR, k.FRGSX));
        if (strat is null) return (null, null);
        var next = strat.OrderBy(s => s.StepNo).FirstOrDefault(s => !released.Contains(s.FRGCO));
        return next is null ? (null, null) : (next.FRGCO, next.StepNo);
    }

    private static bool CanRelease(Ekko k, string? nextCode, ISet<string> userCodes)
        => k.FRGKE == "B" && nextCode is not null && k.FRGGR is not null
           && userCodes.Contains($"{k.FRGGR}|{nextCode}");

    private async Task<Dictionary<(string, string), List<T16FSCode>>> LoadStepsAsync(CancellationToken ct)
    {
        var all = await _db.T16FSCode.AsNoTracking().ToListAsync(ct);
        return all.GroupBy(s => (s.FRGGR, s.FRGSX))
                  .ToDictionary(g => g.Key, g => g.OrderBy(s => s.StepNo).ToList());
    }

    private async Task<Dictionary<(string, string), string?>> CodeTextsAsync(CancellationToken ct)
        => await _db.T16FC.AsNoTracking().ToDictionaryAsync(c => (c.FRGGR, c.FRGCO), c => c.FRGCT, ct);

    private async Task<Dictionary<string, string?>> IndicatorTextsAsync(CancellationToken ct)
        => await _db.DomainValues.AsNoTracking()
            .Where(d => d.Domain == "FRGKE")
            .ToDictionaryAsync(d => d.ValueKey, d => d.ValueTxt, ct);

    private async Task<HashSet<string>> UserCodesAsync(int userId, CancellationToken ct)
    {
        var codes = await _db.AppUserReleaseCodes.AsNoTracking()
            .Where(c => c.UserId == userId)
            .Select(c => c.FRGGR + "|" + c.FRGCO)
            .ToListAsync(ct);
        return new HashSet<string>(codes, StringComparer.Ordinal);
    }
}
