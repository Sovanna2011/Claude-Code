using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Domain.Common;

namespace SugarcanePlanning.Application.Services;

/// <summary>Value helps for the cascading dropdowns used across the UI.</summary>
public class LookupService : ServiceBase, ILookupService
{
    public LookupService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock) : base(db, user, clock) { }

    public async Task<IReadOnlyList<LookupDto>> GetAsync(string kind, int? parentId, CancellationToken ct = default)
    {
        kind = (kind ?? string.Empty).ToLowerInvariant();
        return kind switch
        {
            "companies" => (await Db.Companies.AsNoTracking().Where(c => c.IsActive)
                .OrderBy(c => c.Code).ToListAsync(ct))
                .Select(c => new LookupDto { Id = c.Id, Code = c.Code, Name = c.Name }).ToList(),

            "estates" => (await Db.Estates.AsNoTracking()
                .Where(e => e.IsActive && (parentId == null || e.CompanyId == parentId))
                .OrderBy(e => e.Code).ToListAsync(ct)).Select(e => e.ToLookup()).ToList(),

            "farms" => (await Db.Farms.AsNoTracking()
                .Where(f => f.IsActive && (parentId == null || f.EstateId == parentId))
                .OrderBy(f => f.Code).ToListAsync(ct)).Select(f => f.ToLookup()).ToList(),

            "zones" => (await Db.Zones.AsNoTracking()
                .Where(z => z.IsActive && (parentId == null || z.FarmId == parentId))
                .OrderBy(z => z.Code).ToListAsync(ct)).Select(z => z.ToLookup()).ToList(),

            "blocks" => (await Db.Blocks.AsNoTracking()
                .Where(b => b.IsActive && (parentId == null || b.ZoneId == parentId))
                .OrderBy(b => b.Code).ToListAsync(ct)).Select(b => b.ToLookup()).ToList(),

            "seasons" => (await Db.Seasons.AsNoTracking()
                .Where(s => parentId == null || s.CompanyId == parentId)
                .OrderByDescending(s => s.StartDate).ToListAsync(ct)).Select(s => s.ToLookup()).ToList(),

            "varieties" => (await Db.Varieties.AsNoTracking().Where(v => v.IsActive)
                .OrderBy(v => v.Code).ToListAsync(ct)).Select(v => v.ToLookup()).ToList(),

            "activities" => (await Db.Activities.AsNoTracking().Where(a => a.IsActive)
                .OrderBy(a => a.SequenceNo).ToListAsync(ct)).Select(a => a.ToLookup()).ToList(),

            "materials" => (await Db.Materials.AsNoTracking().Where(m => m.IsActive)
                .OrderBy(m => m.Code).ToListAsync(ct)).Select(m => m.ToLookup()).ToList(),

            "tractors" => (await Db.Tractors.AsNoTracking()
                .Where(t => t.IsActive && (parentId == null || t.CurrentFarmId == parentId))
                .OrderBy(t => t.Code).ToListAsync(ct)).Select(t => t.ToLookup()).ToList(),

            "equipment" => (await Db.Equipment.AsNoTracking()
                .Where(e => e.IsActive && (parentId == null || e.CurrentFarmId == parentId))
                .OrderBy(e => e.Code).ToListAsync(ct)).Select(e => e.ToLookup()).ToList(),

            "operators" => (await Db.Operators.AsNoTracking()
                .Where(o => o.IsActive && (parentId == null || o.FarmId == parentId))
                .OrderBy(o => o.Code).ToListAsync(ct)).Select(o => o.ToLookup()).ToList(),

            "workteams" => (await Db.WorkTeams.AsNoTracking()
                .Where(t => t.IsActive && (parentId == null || t.FarmId == parentId))
                .OrderBy(t => t.Code).ToListAsync(ct)).Select(t => t.ToLookup()).ToList(),

            "projections" => (await Db.Projections.AsNoTracking()
                .Where(p => p.IsCurrentVersion && (parentId == null || p.GrowingSeasonId == parentId))
                .OrderByDescending(p => p.ProjectionDate).ToListAsync(ct))
                .Select(p => new LookupDto
                {
                    Id = p.Id,
                    Code = p.ProjectionNo,
                    Name = $"v{p.Version} · {p.Status}",
                    ParentId = p.GrowingSeasonId,
                    Value = p.TotalProjectedAreaHa
                }).ToList(),

            _ => throw new NotFoundException($"Unknown lookup kind '{kind}'.")
        };
    }
}
