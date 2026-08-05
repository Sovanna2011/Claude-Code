using System.Linq.Expressions;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Seasons;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>Growing seasons and sugarcane varieties (section 4).</summary>
public class SeasonService : ServiceBase, ISeasonService
{
    public SeasonService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock) : base(db, user, clock) { }

    // ------------------------------------------------------------------ seasons

    public async Task<PagedResult<GrowingSeasonDto>> GetSeasonsAsync(QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Seasons.AsNoTracking().AsQueryable();
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(s => s.Code.Contains(q.Search) || s.Name.Contains(q.Search));
        if (!q.IncludeInactive) query = query.Where(s => s.Status != SeasonStatus.Closed);

        var sorts = new Dictionary<string, Expression<Func<GrowingSeason, object>>>
        {
            ["code"] = s => s.Code,
            ["name"] = s => s.Name,
            ["startDate"] = s => s.StartDate
        };
        var result = await query.ApplySort(q, sorts, s => s.StartDate).ToPagedResultAsync(q, s => s.ToDto(), ct);

        // Attach the projection counts without pulling the whole graph.
        var ids = result.Items.Select(i => i.Id).ToList();
        var counts = await Db.Projections.AsNoTracking()
            .Where(p => ids.Contains(p.GrowingSeasonId))
            .GroupBy(p => p.GrowingSeasonId)
            .Select(g => new { SeasonId = g.Key, Count = g.Count() })
            .ToListAsync(ct);
        foreach (var item in result.Items)
            item.ProjectionCount = counts.FirstOrDefault(c => c.SeasonId == item.Id)?.Count ?? 0;
        return result;
    }

    public async Task<GrowingSeasonDto> GetSeasonAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Seasons.AsNoTracking(), id, "Growing season", ct)).ToDto();

    public async Task<GrowingSeasonDto> CreateSeasonAsync(GrowingSeasonUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.Seasons.AnyAsync(s => s.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Season code '{code}' is already in use.");
        ValidateSeasonDates(dto);

        var entity = new GrowingSeason();
        dto.ApplyTo(entity);
        entity.CompanyId = User.CompanyId ?? dto.CompanyId;
        Db.Seasons.Add(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task<GrowingSeasonDto> UpdateSeasonAsync(int id, GrowingSeasonUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Seasons, id, "Growing season", ct);
        if (entity.Status == SeasonStatus.Closed)
            throw new BusinessRuleException("SEASON_CLOSED", "A closed season can no longer be changed.");
        var code = dto.Code.Trim();
        if (await Db.Seasons.AnyAsync(s => s.Code == code && s.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Season code '{code}' is already in use.");
        ValidateSeasonDates(dto);

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task DeleteSeasonAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Seasons, id, "Growing season", ct);
        if (await Db.Projections.AnyAsync(p => p.GrowingSeasonId == id, ct))
            throw new BusinessRuleException("IN_USE", "Projections exist for this season; it cannot be deleted.");
        SoftDelete(entity);
        await Db.SaveChangesAsync(ct);
    }

    private static void ValidateSeasonDates(GrowingSeasonUpsertDto dto)
    {
        if (dto.EndDate < dto.StartDate)
            throw new BusinessRuleException("DATE_RANGE", "Season end date must not be before the start date.");
        if (dto.PlannedPlantingEnd < dto.PlannedPlantingStart)
            throw new BusinessRuleException("DATE_RANGE", "Planting end date must not be before the planting start date.");
        if (dto.PlannedPlantingStart < dto.StartDate || dto.PlannedPlantingEnd > dto.EndDate)
            throw new BusinessRuleException("DATE_RANGE", "The planting window must sit inside the season.");
        if (dto.ExpectedHarvestEnd < dto.ExpectedHarvestStart)
            throw new BusinessRuleException("DATE_RANGE", "Harvest end date must not be before the harvest start date.");
    }

    // ---------------------------------------------------------------- varieties

    public async Task<PagedResult<CaneVarietyDto>> GetVarietiesAsync(QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Varieties.AsNoTracking().AsQueryable();
        if (!q.IncludeInactive) query = query.Where(v => v.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(v => v.Code.Contains(q.Search) || v.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<CaneVariety, object>>>
        {
            ["code"] = v => v.Code,
            ["name"] = v => v.Name,
            ["yield"] = v => v.ExpectedYieldPerHa
        };
        return await query.ApplySort(q, sorts, v => v.Code).ToPagedResultAsync(q, v => v.ToDto(), ct);
    }

    public async Task<CaneVarietyDto> GetVarietyAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Varieties.AsNoTracking(), id, "Sugarcane variety", ct)).ToDto();

    public async Task<CaneVarietyDto> CreateVarietyAsync(CaneVarietyUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.Varieties.AnyAsync(v => v.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Variety code '{code}' is already in use.");

        var entity = new CaneVariety();
        dto.ApplyTo(entity);
        entity.CompanyId = User.CompanyId ?? dto.CompanyId;
        Db.Varieties.Add(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task<CaneVarietyDto> UpdateVarietyAsync(int id, CaneVarietyUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Varieties, id, "Sugarcane variety", ct);
        var code = dto.Code.Trim();
        if (await Db.Varieties.AnyAsync(v => v.Code == code && v.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Variety code '{code}' is already in use.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task DeleteVarietyAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Varieties, id, "Sugarcane variety", ct);
        if (await Db.ProjectionLines.AnyAsync(l => l.CaneVarietyId == id, ct))
            throw new BusinessRuleException("IN_USE", "This variety is used by a projection line and cannot be deleted.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }
}
