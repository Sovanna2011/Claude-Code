using System.Linq.Expressions;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Organization;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;

namespace SugarcanePlanning.Application.Services;

/// <summary>Company → estate → farm → zone → block master data (section 3).</summary>
public class LandStructureService : ServiceBase, ILandStructureService
{
    public LandStructureService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock) : base(db, user, clock) { }

    // ------------------------------------------------------------------ company

    public async Task<PagedResult<CompanyDto>> GetCompaniesAsync(QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Companies.Include(c => c.Estates).AsNoTracking().AsQueryable();
        if (!q.IncludeInactive) query = query.Where(c => c.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(c => c.Code.Contains(q.Search) || c.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<Company, object>>>
        {
            ["code"] = c => c.Code,
            ["name"] = c => c.Name
        };
        return await query.ApplySort(q, sorts, c => c.Code).ToPagedResultAsync(q, c => c.ToDto(), ct);
    }

    public async Task<CompanyDto> GetCompanyAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Companies.Include(c => c.Estates).AsNoTracking(), id, "Company", ct)).ToDto();

    public async Task<CompanyDto> CreateCompanyAsync(CompanyUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.Companies.AnyAsync(c => c.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Company code '{code}' is already in use.");

        var entity = new Company();
        dto.ApplyTo(entity);
        Db.Companies.Add(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task<CompanyDto> UpdateCompanyAsync(int id, CompanyUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Companies, id, "Company", ct);
        var code = dto.Code.Trim();
        if (await Db.Companies.AnyAsync(c => c.Code == code && c.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Company code '{code}' is already in use.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        dto.ApplyTo(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task DeleteCompanyAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Companies, id, "Company", ct);
        if (await Db.Estates.AnyAsync(e => e.CompanyId == id, ct))
            throw new BusinessRuleException("IN_USE", "Deactivate or remove the estates of this company first.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }

    // ------------------------------------------------------------------- estate

    public async Task<PagedResult<EstateDto>> GetEstatesAsync(int? companyId, QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Estates.Include(e => e.Company).Include(e => e.Farms).AsNoTracking().AsQueryable();
        if (companyId is not null) query = query.Where(e => e.CompanyId == companyId);
        if (!q.IncludeInactive) query = query.Where(e => e.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(e => e.Code.Contains(q.Search) || e.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<Estate, object>>>
        {
            ["code"] = e => e.Code,
            ["name"] = e => e.Name,
            ["area"] = e => e.TotalAreaHa
        };
        return await query.ApplySort(q, sorts, e => e.Code).ToPagedResultAsync(q, e => e.ToDto(), ct);
    }

    public async Task<EstateDto> GetEstateAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Estates.Include(e => e.Company).Include(e => e.Farms).AsNoTracking(), id, "Estate", ct)).ToDto();

    public async Task<EstateDto> CreateEstateAsync(EstateUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.Estates.AnyAsync(e => e.CompanyId == dto.CompanyId && e.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Estate code '{code}' already exists in this company.");

        var entity = new Estate();
        dto.ApplyTo(entity);
        entity.CompanyId = User.CompanyId ?? dto.CompanyId;
        Db.Estates.Add(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task<EstateDto> UpdateEstateAsync(int id, EstateUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Estates, id, "Estate", ct);
        var code = dto.Code.Trim();
        if (await Db.Estates.AnyAsync(e => e.CompanyId == entity.CompanyId && e.Code == code && e.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Estate code '{code}' already exists in this company.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;                       // tenancy is never moved by an update
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task DeleteEstateAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Estates, id, "Estate", ct);
        if (await Db.Farms.AnyAsync(f => f.EstateId == id, ct))
            throw new BusinessRuleException("IN_USE", "Remove the farms of this estate first.");
        if (await Db.Projections.AnyAsync(p => p.EstateId == id, ct))
            throw new BusinessRuleException("IN_USE", "Projections exist for this estate; it cannot be deleted.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }

    // --------------------------------------------------------------------- farm

    public async Task<PagedResult<FarmDto>> GetFarmsAsync(int? estateId, QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Farms.Include(f => f.Estate).Include(f => f.Zones).AsNoTracking().AsQueryable();
        if (estateId is not null) query = query.Where(f => f.EstateId == estateId);
        if (!q.IncludeInactive) query = query.Where(f => f.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(f => f.Code.Contains(q.Search) || f.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<Farm, object>>>
        {
            ["code"] = f => f.Code,
            ["name"] = f => f.Name,
            ["area"] = f => f.TotalAreaHa
        };
        return await query.ApplySort(q, sorts, f => f.Code).ToPagedResultAsync(q, f => f.ToDto(), ct);
    }

    public async Task<FarmDto> GetFarmAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Farms.Include(f => f.Estate).Include(f => f.Zones).AsNoTracking(), id, "Farm", ct)).ToDto();

    public async Task<FarmDto> CreateFarmAsync(FarmUpsertDto dto, CancellationToken ct = default)
    {
        var estate = await RequireAsync(Db.Estates, dto.EstateId, "Estate", ct);
        var code = dto.Code.Trim();
        if (await Db.Farms.AnyAsync(f => f.EstateId == dto.EstateId && f.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Farm code '{code}' already exists in this estate.");

        var entity = new Farm { CompanyId = estate.CompanyId };
        dto.ApplyTo(entity);
        entity.CompanyId = estate.CompanyId;
        Db.Farms.Add(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task<FarmDto> UpdateFarmAsync(int id, FarmUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Farms, id, "Farm", ct);
        var code = dto.Code.Trim();
        if (await Db.Farms.AnyAsync(f => f.EstateId == dto.EstateId && f.Code == code && f.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Farm code '{code}' already exists in this estate.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task DeleteFarmAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Farms, id, "Farm", ct);
        if (await Db.Zones.AnyAsync(z => z.FarmId == id, ct))
            throw new BusinessRuleException("IN_USE", "Remove the zones of this farm first.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }

    // --------------------------------------------------------------------- zone

    public async Task<PagedResult<ZoneDto>> GetZonesAsync(int? farmId, QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Zones.Include(z => z.Farm).Include(z => z.Blocks).AsNoTracking().AsQueryable();
        if (farmId is not null) query = query.Where(z => z.FarmId == farmId);
        if (!q.IncludeInactive) query = query.Where(z => z.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(z => z.Code.Contains(q.Search) || z.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<Zone, object>>>
        {
            ["code"] = z => z.Code,
            ["name"] = z => z.Name,
            ["area"] = z => z.TotalAreaHa
        };
        return await query.ApplySort(q, sorts, z => z.Code).ToPagedResultAsync(q, z => z.ToDto(), ct);
    }

    public async Task<ZoneDto> GetZoneAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Zones.Include(z => z.Farm).Include(z => z.Blocks).AsNoTracking(), id, "Zone", ct)).ToDto();

    public async Task<ZoneDto> CreateZoneAsync(ZoneUpsertDto dto, CancellationToken ct = default)
    {
        var farm = await RequireAsync(Db.Farms, dto.FarmId, "Farm", ct);
        var code = dto.Code.Trim();
        if (await Db.Zones.AnyAsync(z => z.FarmId == dto.FarmId && z.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Zone code '{code}' already exists in this farm.");

        var entity = new Zone();
        dto.ApplyTo(entity);
        entity.CompanyId = farm.CompanyId;
        Db.Zones.Add(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task<ZoneDto> UpdateZoneAsync(int id, ZoneUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Zones, id, "Zone", ct);
        var code = dto.Code.Trim();
        if (await Db.Zones.AnyAsync(z => z.FarmId == dto.FarmId && z.Code == code && z.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Zone code '{code}' already exists in this farm.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task DeleteZoneAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Zones, id, "Zone", ct);
        if (await Db.Blocks.AnyAsync(b => b.ZoneId == id, ct))
            throw new BusinessRuleException("IN_USE", "Remove the blocks of this zone first.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }

    // -------------------------------------------------------------------- block

    public async Task<PagedResult<PlantationBlockDto>> GetBlocksAsync(int? zoneId, int? farmId, QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Blocks
            .Include(b => b.Zone!).ThenInclude(z => z.Farm!).ThenInclude(f => f.Estate)
            .AsNoTracking().AsQueryable();
        if (zoneId is not null) query = query.Where(b => b.ZoneId == zoneId);
        if (farmId is not null) query = query.Where(b => b.Zone!.FarmId == farmId);
        if (!q.IncludeInactive) query = query.Where(b => b.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(b => b.Code.Contains(q.Search) || b.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<PlantationBlock, object>>>
        {
            ["code"] = b => b.Code,
            ["name"] = b => b.Name,
            ["area"] = b => b.PlantableAreaHa
        };
        return await query.ApplySort(q, sorts, b => b.Code).ToPagedResultAsync(q, b => b.ToDto(), ct);
    }

    public async Task<PlantationBlockDto> GetBlockAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Blocks
            .Include(b => b.Zone!).ThenInclude(z => z.Farm!).ThenInclude(f => f.Estate)
            .AsNoTracking(), id, "Plantation block", ct)).ToDto();

    public async Task<PlantationBlockDto> CreateBlockAsync(PlantationBlockUpsertDto dto, CancellationToken ct = default)
    {
        var zone = await RequireAsync(Db.Zones, dto.ZoneId, "Zone", ct);
        var code = dto.Code.Trim();
        if (await Db.Blocks.AnyAsync(b => b.ZoneId == dto.ZoneId && b.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Block code '{code}' already exists in this zone.");
        if (dto.PlantableAreaHa > dto.TotalAreaHa)
            throw new BusinessRuleException("AREA_INVALID", "Plantable area cannot exceed the total area of the block.");

        var entity = new PlantationBlock();
        dto.ApplyTo(entity);
        entity.CompanyId = zone.CompanyId;
        Db.Blocks.Add(entity);
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task<PlantationBlockDto> UpdateBlockAsync(int id, PlantationBlockUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Blocks, id, "Plantation block", ct);
        var code = dto.Code.Trim();
        if (await Db.Blocks.AnyAsync(b => b.ZoneId == dto.ZoneId && b.Code == code && b.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Block code '{code}' already exists in this zone.");
        if (dto.PlantableAreaHa > dto.TotalAreaHa)
            throw new BusinessRuleException("AREA_INVALID", "Plantable area cannot exceed the total area of the block.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return entity.ToDto();
    }

    public async Task DeleteBlockAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Blocks, id, "Plantation block", ct);
        if (await Db.ProjectionLines.AnyAsync(l => l.BlockId == id, ct))
            throw new BusinessRuleException("IN_USE", "This block is used by a planting projection and cannot be deleted.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }

    // ---------------------------------------------------------------- structure

    public async Task<IReadOnlyList<LandStructureNodeDto>> GetStructureAsync(int? companyId, CancellationToken ct = default)
    {
        var companies = await Db.Companies.AsNoTracking()
            .Where(c => companyId == null || c.Id == companyId)
            .OrderBy(c => c.Code).ToListAsync(ct);
        var estates = await Db.Estates.AsNoTracking().OrderBy(e => e.Code).ToListAsync(ct);
        var farms = await Db.Farms.AsNoTracking().OrderBy(f => f.Code).ToListAsync(ct);
        var zones = await Db.Zones.AsNoTracking().OrderBy(z => z.Code).ToListAsync(ct);
        var blocks = await Db.Blocks.AsNoTracking().OrderBy(b => b.Code).ToListAsync(ct);

        return companies.Select(c => new LandStructureNodeDto
        {
            NodeType = "Company",
            Id = c.Id,
            Code = c.Code,
            Name = c.Name,
            IsActive = c.IsActive,
            Children = estates.Where(e => e.CompanyId == c.Id).Select(e => new LandStructureNodeDto
            {
                NodeType = "Estate",
                Id = e.Id,
                Code = e.Code,
                Name = e.Name,
                TotalAreaHa = e.TotalAreaHa,
                IsActive = e.IsActive,
                Children = farms.Where(f => f.EstateId == e.Id).Select(f => new LandStructureNodeDto
                {
                    NodeType = "Farm",
                    Id = f.Id,
                    Code = f.Code,
                    Name = f.Name,
                    TotalAreaHa = f.TotalAreaHa,
                    IsActive = f.IsActive,
                    Children = zones.Where(z => z.FarmId == f.Id).Select(z => new LandStructureNodeDto
                    {
                        NodeType = "Zone",
                        Id = z.Id,
                        Code = z.Code,
                        Name = z.Name,
                        TotalAreaHa = z.TotalAreaHa,
                        IsActive = z.IsActive,
                        Children = blocks.Where(b => b.ZoneId == z.Id).Select(b => new LandStructureNodeDto
                        {
                            NodeType = "Block",
                            Id = b.Id,
                            Code = b.Code,
                            Name = b.Name,
                            TotalAreaHa = b.TotalAreaHa,
                            PlantableAreaHa = b.PlantableAreaHa,
                            IsActive = b.IsActive
                        }).ToList()
                    }).ToList()
                }).ToList()
            }).ToList()
        }).ToList();
    }
}
