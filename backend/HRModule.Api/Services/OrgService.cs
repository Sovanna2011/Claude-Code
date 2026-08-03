using HRModule.Api.Data;
using HRModule.Api.DTOs;
using HRModule.Api.Services.Interfaces;
using Microsoft.EntityFrameworkCore;

namespace HRModule.Api.Services;

/// <summary>
/// Organizational Management service. Reconstructs the org hierarchy by walking
/// HRP1001 relationships (evaluation paths), mirroring SAP transaction PPOME.
///   A 002  org unit reports to org unit
///   A 003  position belongs to org unit
///   B 008  position is held by person
///   B 012  org unit is managed by chief position
/// </summary>
public class OrgService : IOrgService
{
    private readonly HRDbContext _db;
    public OrgService(HRDbContext db) => _db = db;

    public async Task<IReadOnlyList<OrgUnitNodeDto>> GetOrgUnitsFlatAsync(DateTime? keyDate, CancellationToken ct = default)
    {
        var key = keyDate?.Date ?? DateTime.Today;
        var units = await _db.HRP1000.AsNoTracking()
            .Where(o => o.OTYPE == "O" && o.PLVAR == "01" && o.BEGDA <= key && o.ENDDA >= key)
            .Select(o => new OrgUnitNodeDto { OrgUnitId = o.OBJID, OrgUnitName = o.STEXT, ShortText = o.SHORT })
            .ToListAsync(ct);

        var parentRels = await _db.HRP1001.AsNoTracking()
            .Where(r => r.OTYPE == "O" && r.SCLAS == "O" && r.RSIGN == "A" && r.RELAT == "002"
                        && r.PLVAR == "01" && r.BEGDA <= key && r.ENDDA >= key)
            .ToListAsync(ct);

        foreach (var u in units)
        {
            var rel = parentRels.FirstOrDefault(r => r.OBJID == u.OrgUnitId);
            if (rel is not null && int.TryParse(rel.SOBID, out var parent))
                u.ParentOrgId = parent;
        }
        return units.OrderBy(u => u.OrgUnitId).ToList();
    }

    public async Task<OrgUnitNodeDto?> GetOrgStructureAsync(int rootOrgId, DateTime? keyDate, CancellationToken ct = default)
    {
        var key = keyDate?.Date ?? DateTime.Today;
        var flat = (await GetOrgUnitsFlatAsync(key, ct)).ToList();
        var byId = flat.ToDictionary(u => u.OrgUnitId);
        if (!byId.TryGetValue(rootOrgId, out var root)) return null;

        // Head counts (employees assigned to each org unit on the key date).
        // Materialise first, then group in memory to stay clear of EF Core's
        // limitations translating Distinct().Count() inside a GroupBy projection.
        var assignments = await _db.PA0001.AsNoTracking()
            .Where(a => a.ORGEH != null && a.BEGDA <= key && a.ENDDA >= key)
            .Select(a => new { OrgId = a.ORGEH!.Value, a.PERNR })
            .ToListAsync(ct);
        var countMap = assignments
            .GroupBy(a => a.OrgId)
            .ToDictionary(g => g.Key, g => g.Select(x => x.PERNR).Distinct().Count());

        // Chief position (manager) per org unit.
        var chiefs = await _db.HRP1001.AsNoTracking()
            .Where(r => r.OTYPE == "O" && r.RSIGN == "B" && r.RELAT == "012" && r.SCLAS == "S"
                        && r.PLVAR == "01" && r.BEGDA <= key && r.ENDDA >= key)
            .ToListAsync(ct);
        var chiefMap = new Dictionary<int, string?>();
        foreach (var c in chiefs)
            if (int.TryParse(c.SOBID, out var posId))
            {
                var name = await _db.HRP1000.AsNoTracking()
                    .Where(p => p.OTYPE == "S" && p.OBJID == posId && p.PLVAR == "01"
                                && p.BEGDA <= key && p.ENDDA >= key)
                    .Select(p => p.STEXT).FirstOrDefaultAsync(ct);
                chiefMap[c.OBJID] = name;
            }

        foreach (var u in flat)
        {
            u.HeadCount = countMap.GetValueOrDefault(u.OrgUnitId);
            u.ManagerPositionName = chiefMap.GetValueOrDefault(u.OrgUnitId);
        }

        // Build the tree.
        foreach (var u in flat)
            if (u.ParentOrgId is int pid && byId.TryGetValue(pid, out var parent) && pid != u.OrgUnitId)
                parent.Children.Add(u);

        return root;
    }

    public async Task<IReadOnlyList<PositionDto>> GetPositionsAsync(int? orgUnitId, DateTime? keyDate, CancellationToken ct = default)
    {
        var key = keyDate?.Date ?? DateTime.Today;

        var positions = await _db.HRP1000.AsNoTracking()
            .Where(p => p.OTYPE == "S" && p.PLVAR == "01" && p.BEGDA <= key && p.ENDDA >= key)
            .Select(p => new PositionDto { PositionId = p.OBJID, PositionName = p.STEXT })
            .ToListAsync(ct);

        // position -> org unit (A 003)
        var toOrg = await _db.HRP1001.AsNoTracking()
            .Where(r => r.OTYPE == "S" && r.RSIGN == "A" && r.RELAT == "003" && r.SCLAS == "O"
                        && r.PLVAR == "01" && r.BEGDA <= key && r.ENDDA >= key)
            .ToListAsync(ct);
        // position -> job (A 007)
        var toJob = await _db.HRP1001.AsNoTracking()
            .Where(r => r.OTYPE == "S" && r.RSIGN == "A" && r.RELAT == "007" && r.SCLAS == "C"
                        && r.PLVAR == "01" && r.BEGDA <= key && r.ENDDA >= key)
            .ToListAsync(ct);

        var orgNames = await _db.HRP1000.AsNoTracking()
            .Where(o => o.OTYPE == "O" && o.PLVAR == "01" && o.BEGDA <= key && o.ENDDA >= key)
            .ToDictionaryAsync(o => o.OBJID, o => o.STEXT, ct);
        var jobNames = await _db.HRP1000.AsNoTracking()
            .Where(o => o.OTYPE == "C" && o.PLVAR == "01" && o.BEGDA <= key && o.ENDDA >= key)
            .ToDictionaryAsync(o => o.OBJID, o => o.STEXT, ct);

        // holder: person assigned to the position via PA0001 (PLANS)
        var holders = await (
            from a in _db.PA0001.AsNoTracking()
            join p2 in _db.PA0002.AsNoTracking()
                on a.PERNR equals p2.PERNR
            where a.PLANS != null && a.BEGDA <= key && a.ENDDA >= key
                  && p2.BEGDA <= key && p2.ENDDA >= key
            select new { a.PLANS, a.PERNR, p2.VORNA, p2.NACHN }
        ).ToListAsync(ct);

        foreach (var pos in positions)
        {
            var org = toOrg.FirstOrDefault(r => r.OBJID == pos.PositionId);
            if (org is not null && int.TryParse(org.SOBID, out var oId))
            { pos.OrgUnitId = oId; pos.OrgUnitName = orgNames.GetValueOrDefault(oId); }

            var job = toJob.FirstOrDefault(r => r.OBJID == pos.PositionId);
            if (job is not null && int.TryParse(job.SOBID, out var jId))
            { pos.JobId = jId; pos.JobName = jobNames.GetValueOrDefault(jId); }

            var holder = holders.FirstOrDefault(h => h.PLANS == pos.PositionId);
            if (holder is not null)
            { pos.HolderPernr = holder.PERNR; pos.HolderName = $"{holder.VORNA} {holder.NACHN}"; }
        }

        if (orgUnitId is int filter)
            positions = positions.Where(p => p.OrgUnitId == filter).ToList();

        return positions.OrderBy(p => p.PositionId).ToList();
    }
}
