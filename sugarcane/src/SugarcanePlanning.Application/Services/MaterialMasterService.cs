using System.Linq.Expressions;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;

namespace SugarcanePlanning.Application.Services;

/// <summary>Material master, activity material standards and the read-only stock feed (sections 11-13).</summary>
public class MaterialMasterService : ServiceBase, IMaterialMasterService
{
    private readonly IAuditService _audit;

    public MaterialMasterService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock, IAuditService audit)
        : base(db, user, clock) => _audit = audit;

    // ---------------------------------------------------------------- materials

    public async Task<PagedResult<MaterialDto>> GetMaterialsAsync(QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Materials.AsNoTracking().AsQueryable();
        if (!q.IncludeInactive) query = query.Where(m => m.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(m => m.Code.Contains(q.Search) || m.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<Material, object>>>
        {
            ["code"] = m => m.Code,
            ["name"] = m => m.Name,
            ["category"] = m => m.Category
        };
        return await query.ApplySort(q, sorts, m => m.Code).ToPagedResultAsync(q, m => m.ToDto(), ct);
    }

    public async Task<MaterialDto> GetMaterialAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Materials.AsNoTracking(), id, "Material", ct)).ToDto();

    public async Task<MaterialDto> CreateMaterialAsync(MaterialUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.Materials.AnyAsync(m => m.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Material code '{code}' is already in use.");
        ValidateRates(dto);

        var entity = new Material();
        dto.ApplyTo(entity);
        entity.CompanyId = User.CompanyId ?? dto.CompanyId;
        Db.Materials.Add(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task<MaterialDto> UpdateMaterialAsync(int id, MaterialUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Materials, id, "Material", ct);
        var code = dto.Code.Trim();
        if (await Db.Materials.AnyAsync(m => m.Code == code && m.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Material code '{code}' is already in use.");
        ValidateRates(dto);

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task DeleteMaterialAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Materials, id, "Material", ct);
        if (await Db.MaterialStandards.AnyAsync(s => s.MaterialId == id, ct))
            throw new BusinessRuleException("IN_USE", "Remove the activity material standards for this material first.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }

    private static void ValidateRates(MaterialUpsertDto dto)
    {
        if (dto.MaxApplicationRate > 0 && dto.MinApplicationRate > dto.MaxApplicationRate)
            throw new BusinessRuleException("RATE_BAND", "Minimum application rate cannot exceed the maximum.");
        if (dto.UnitConversionFactor <= 0)
            throw new BusinessRuleException("CONVERSION", "The unit conversion factor must be greater than zero.");
        if (dto.AlternativeUnit is not null && dto.AlternativeUnit == dto.BaseUnit)
            throw new BusinessRuleException("UNIT", "The alternative unit must differ from the base unit.");
    }

    // ---------------------------------------------------------------- standards

    public async Task<PagedResult<ActivityMaterialStandardDto>> GetStandardsAsync(int? activityId, QueryParameters q,
        CancellationToken ct = default)
    {
        var query = Db.MaterialStandards
            .Include(s => s.Activity).Include(s => s.Material).Include(s => s.CaneVariety)
            .AsNoTracking().AsQueryable();
        if (activityId is not null) query = query.Where(s => s.ActivityId == activityId);
        if (!q.IncludeInactive) query = query.Where(s => s.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(s => s.Material!.Code.Contains(q.Search) || s.Material.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<ActivityMaterialStandard, object>>>
        {
            ["activity"] = s => s.Activity!.SequenceNo,
            ["material"] = s => s.Material!.Code,
            ["effectiveFrom"] = s => s.EffectiveFrom
        };
        return await query.ApplySort(q, sorts, s => s.Activity!.SequenceNo).ToPagedResultAsync(q, s => s.ToDto(), ct);
    }

    public async Task<ActivityMaterialStandardDto> CreateStandardAsync(ActivityMaterialStandardUpsertDto dto,
        CancellationToken ct = default)
    {
        var activity = await RequireAsync(Db.Activities, dto.ActivityId, "Planting activity", ct);
        var material = await RequireAsync(Db.Materials, dto.MaterialId, "Material", ct);
        ValidateStandard(dto, material);
        await RequireNoOverlappingStandardAsync(dto, 0, ct);

        var entity = new ActivityMaterialStandard { CompanyId = activity.CompanyId };
        dto.ApplyTo(entity);
        entity.CompanyId = activity.CompanyId;
        if (entity.EffectiveFrom == default) entity.EffectiveFrom = Clock.Today;
        Db.MaterialStandards.Add(entity);
        await Db.SaveChangesAsync(ct);

        await _audit.LogAsync(Domain.Enums.AuditAction.Create, nameof(ActivityMaterialStandard), entity.Id.ToString(),
            null, $"{material.Code} @ {entity.StandardRatePerHa}/ha", null, ct);

        return await LoadStandardAsync(entity.Id, ct);
    }

    public async Task<ActivityMaterialStandardDto> UpdateStandardAsync(int id, ActivityMaterialStandardUpsertDto dto,
        CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.MaterialStandards, id, "Activity material standard", ct);
        var material = await RequireAsync(Db.Materials, dto.MaterialId, "Material", ct);
        ValidateStandard(dto, material);
        await RequireNoOverlappingStandardAsync(dto, id, ct);

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var old = $"{entity.StandardRatePerHa}/ha x{entity.NumberOfApplications} waste {entity.WastePercent}%";
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);

        await _audit.LogAsync(Domain.Enums.AuditAction.Update, nameof(ActivityMaterialStandard), id.ToString(),
            old, $"{entity.StandardRatePerHa}/ha x{entity.NumberOfApplications} waste {entity.WastePercent}%", null, ct);

        return await LoadStandardAsync(id, ct);
    }

    public async Task DeleteStandardAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.MaterialStandards, id, "Activity material standard", ct);
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }

    private void ValidateStandard(ActivityMaterialStandardUpsertDto dto, Material material)
    {
        if (dto.EffectiveTo is not null && dto.EffectiveTo < dto.EffectiveFrom)
            throw new BusinessRuleException("DATE_RANGE", "The effective-to date must not be before the effective-from date.");
        if (!material.IsRateWithinBand(dto.StandardRatePerHa))
            throw new BusinessRuleException("RATE_BAND",
                $"Rate {dto.StandardRatePerHa} is outside the allowed band for {material.Code} " +
                $"({material.MinApplicationRate} … {material.MaxApplicationRate} {material.BaseUnit}/ha).");
    }

    /// <summary>Two standards with the same specificity may not be effective at the same time.</summary>
    private async Task RequireNoOverlappingStandardAsync(ActivityMaterialStandardUpsertDto dto, int excludeId,
        CancellationToken ct)
    {
        var clash = await Db.MaterialStandards.AsNoTracking().AnyAsync(s =>
            s.Id != excludeId
            && s.IsActive
            && s.ActivityId == dto.ActivityId
            && s.MaterialId == dto.MaterialId
            && s.CropType == dto.CropType
            && s.CaneVarietyId == dto.CaneVarietyId
            && s.SoilType == dto.SoilType
            && s.EffectiveFrom <= (dto.EffectiveTo ?? DateOnly.MaxValue)
            && dto.EffectiveFrom <= (s.EffectiveTo ?? DateOnly.MaxValue), ct);

        if (clash)
            throw new BusinessRuleException("STANDARD_OVERLAP",
                "An equally specific standard for this activity and material already covers that period.");
    }

    private async Task<ActivityMaterialStandardDto> LoadStandardAsync(int id, CancellationToken ct)
        => (await Db.MaterialStandards
            .Include(s => s.Activity).Include(s => s.Material).Include(s => s.CaneVariety)
            .AsNoTracking().FirstAsync(s => s.Id == id, ct)).ToDto();

    // -------------------------------------------------------------------- stock

    public async Task<IReadOnlyList<MaterialStockDto>> GetStocksAsync(int? estateId, CancellationToken ct = default)
    {
        var query = Db.MaterialStocks.Include(s => s.Material).Include(s => s.Estate).AsNoTracking().AsQueryable();
        if (estateId is not null) query = query.Where(s => s.EstateId == estateId || s.EstateId == null);
        var rows = await query.OrderBy(s => s.Material!.Code).ToListAsync(ct);
        return rows.Select(s => s.ToDto()).ToList();
    }

    /// <summary>
    /// Stock interface (section 13): upserts the positions delivered by the ERP. The planning
    /// system stores what it is told and never posts movements back.
    /// </summary>
    public async Task<int> SyncStockAsync(IReadOnlyList<MaterialStockSyncDto> rows, CancellationToken ct = default)
    {
        if (rows.Count == 0) return 0;

        var codes = rows.Select(r => r.MaterialCode.Trim()).Distinct().ToList();
        var materials = await Db.Materials.Where(m => codes.Contains(m.Code)).ToDictionaryAsync(m => m.Code, ct);
        var estateCodes = rows.Where(r => !string.IsNullOrWhiteSpace(r.EstateCode))
            .Select(r => r.EstateCode!.Trim()).Distinct().ToList();
        var estates = await Db.Estates.Where(e => estateCodes.Contains(e.Code)).ToDictionaryAsync(e => e.Code, ct);

        var applied = 0;
        foreach (var row in rows)
        {
            if (!materials.TryGetValue(row.MaterialCode.Trim(), out var material))
                throw new NotFoundException($"Material '{row.MaterialCode}' does not exist.");

            int? estateId = null;
            if (!string.IsNullOrWhiteSpace(row.EstateCode))
            {
                if (!estates.TryGetValue(row.EstateCode.Trim(), out var estate))
                    throw new NotFoundException($"Estate '{row.EstateCode}' does not exist.");
                estateId = estate.Id;
            }

            var stock = await Db.MaterialStocks
                .FirstOrDefaultAsync(s => s.MaterialId == material.Id && s.EstateId == estateId, ct);
            if (stock is null)
            {
                stock = new MaterialStock
                {
                    CompanyId = material.CompanyId,
                    MaterialId = material.Id,
                    EstateId = estateId,
                    Unit = material.BaseUnit
                };
                Db.MaterialStocks.Add(stock);
            }

            stock.AvailableStock = row.AvailableStock;
            stock.ReservedQuantity = row.ReservedQuantity;
            stock.IncomingQuantity = row.IncomingQuantity;
            stock.IncomingExpectedDate = row.IncomingExpectedDate;
            stock.LastSyncedUtc = Clock.UtcNow;
            stock.SourceSystem = row.SourceSystem ?? "interface";
            applied++;
        }

        await Db.SaveChangesAsync(ct);
        await _audit.LogAsync(Domain.Enums.AuditAction.Update, nameof(MaterialStock), "interface",
            null, $"{applied} stock position(s) synchronised", null, ct);
        return applied;
    }
}
