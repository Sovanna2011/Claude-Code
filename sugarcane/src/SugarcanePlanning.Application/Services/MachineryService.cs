using System.Linq.Expressions;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Machinery;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>Tractors, implements, the compatibility matrix, operators and teams (sections 8, 9, 14).</summary>
public class MachineryService : ServiceBase, IMachineryService
{
    public MachineryService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock) : base(db, user, clock) { }

    // ----------------------------------------------------------------- tractors

    public async Task<PagedResult<TractorDto>> GetTractorsAsync(QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Tractors.Include(t => t.Estate).Include(t => t.CurrentFarm).AsNoTracking().AsQueryable();
        if (!q.IncludeInactive) query = query.Where(t => t.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(t => t.Code.Contains(q.Search) || t.AssetNo.Contains(q.Search)
                                     || (t.Brand ?? "").Contains(q.Search) || (t.Model ?? "").Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<Tractor, object>>>
        {
            ["code"] = t => t.Code,
            ["assetNo"] = t => t.AssetNo,
            ["hp"] = t => t.Horsepower,
            ["capacity"] = t => t.DailyCapacityHa,
            ["availability"] = t => t.Availability
        };
        return await query.ApplySort(q, sorts, t => t.Code).ToPagedResultAsync(q, t => t.ToDto(), ct);
    }

    public async Task<TractorDto> GetTractorAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Tractors.Include(t => t.Estate).Include(t => t.CurrentFarm).AsNoTracking(), id, "Tractor", ct)).ToDto();

    public async Task<TractorDto> CreateTractorAsync(TractorUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.Tractors.AnyAsync(t => t.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Tractor code '{code}' is already in use.");
        ValidateMaintenanceWindow(dto.MaintenanceFromDate, dto.MaintenanceToDate);

        var entity = new Tractor();
        dto.ApplyTo(entity);
        entity.CompanyId = User.CompanyId ?? dto.CompanyId;
        Db.Tractors.Add(entity);
        await Db.SaveChangesAsync(ct);
        return await GetTractorAsync(entity.Id, ct);
    }

    public async Task<TractorDto> UpdateTractorAsync(int id, TractorUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Tractors, id, "Tractor", ct);
        var code = dto.Code.Trim();
        if (await Db.Tractors.AnyAsync(t => t.Code == code && t.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Tractor code '{code}' is already in use.");
        ValidateMaintenanceWindow(dto.MaintenanceFromDate, dto.MaintenanceToDate);

        // Putting a machine into maintenance must not orphan live bookings.
        if (dto.Availability is AvailabilityStatus.UnderMaintenance or AvailabilityStatus.Breakdown
            && dto.MaintenanceFromDate is not null && dto.MaintenanceToDate is not null)
        {
            var clash = await Db.Schedules.AsNoTracking().AnyAsync(
                s => s.TractorId == id && s.Status != ScheduleStatus.Cancelled
                     && s.ScheduleDate >= dto.MaintenanceFromDate && s.ScheduleDate <= dto.MaintenanceToDate, ct);
            if (clash)
                throw new BusinessRuleException("BOOKED_IN_WINDOW",
                    "This tractor has bookings inside the maintenance window. Reassign them first.");
        }

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return await GetTractorAsync(id, ct);
    }

    public async Task DeleteTractorAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Tractors, id, "Tractor", ct);
        if (await Db.Schedules.AnyAsync(s => s.TractorId == id && s.Status != ScheduleStatus.Cancelled, ct))
            throw new BusinessRuleException("IN_USE", "This tractor has active bookings and cannot be deleted.");
        SoftDelete(entity);
        entity.IsActive = false;
        entity.Availability = AvailabilityStatus.Inactive;
        await Db.SaveChangesAsync(ct);
    }

    private static void ValidateMaintenanceWindow(DateOnly? from, DateOnly? to)
    {
        if (from is not null && to is not null && to < from)
            throw new BusinessRuleException("DATE_RANGE", "Maintenance end date must not be before the start date.");
    }

    // ---------------------------------------------------------------- equipment

    public async Task<PagedResult<EquipmentDto>> GetEquipmentAsync(QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Equipment.Include(e => e.Estate).Include(e => e.CurrentFarm).AsNoTracking().AsQueryable();
        if (!q.IncludeInactive) query = query.Where(e => e.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(e => e.Code.Contains(q.Search) || e.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<EquipmentItem, object>>>
        {
            ["code"] = e => e.Code,
            ["name"] = e => e.Name,
            ["category"] = e => e.Category,
            ["capacity"] = e => e.CapacityPerDay
        };
        return await query.ApplySort(q, sorts, e => e.Code).ToPagedResultAsync(q, e => e.ToDto(), ct);
    }

    public async Task<EquipmentDto> GetEquipmentItemAsync(int id, CancellationToken ct = default)
        => (await RequireAsync(Db.Equipment.Include(e => e.Estate).Include(e => e.CurrentFarm).AsNoTracking(), id, "Equipment", ct)).ToDto();

    public async Task<EquipmentDto> CreateEquipmentAsync(EquipmentUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.Equipment.AnyAsync(e => e.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Equipment code '{code}' is already in use.");
        ValidateMaintenanceWindow(dto.MaintenanceFromDate, dto.MaintenanceToDate);

        var entity = new EquipmentItem();
        dto.ApplyTo(entity);
        entity.CompanyId = User.CompanyId ?? dto.CompanyId;
        Db.Equipment.Add(entity);
        await Db.SaveChangesAsync(ct);
        return await GetEquipmentItemAsync(entity.Id, ct);
    }

    public async Task<EquipmentDto> UpdateEquipmentAsync(int id, EquipmentUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Equipment, id, "Equipment", ct);
        var code = dto.Code.Trim();
        if (await Db.Equipment.AnyAsync(e => e.Code == code && e.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Equipment code '{code}' is already in use.");
        ValidateMaintenanceWindow(dto.MaintenanceFromDate, dto.MaintenanceToDate);

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return await GetEquipmentItemAsync(id, ct);
    }

    public async Task DeleteEquipmentAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Equipment, id, "Equipment", ct);
        if (await Db.Schedules.AnyAsync(s => s.EquipmentId == id && s.Status != ScheduleStatus.Cancelled, ct))
            throw new BusinessRuleException("IN_USE", "This equipment has active bookings and cannot be deleted.");
        SoftDelete(entity);
        entity.IsActive = false;
        entity.Availability = AvailabilityStatus.Inactive;
        await Db.SaveChangesAsync(ct);
    }

    // ----------------------------------------------------------- compatibility

    public async Task<IReadOnlyList<CompatibilityDto>> GetCompatibilitiesAsync(int? tractorId, int? equipmentId,
        CancellationToken ct = default)
    {
        var query = Db.Compatibilities.Include(c => c.Tractor).Include(c => c.Equipment).AsNoTracking().AsQueryable();
        if (tractorId is not null) query = query.Where(c => c.TractorId == tractorId);
        if (equipmentId is not null) query = query.Where(c => c.EquipmentId == equipmentId);
        var rows = await query.OrderBy(c => c.Tractor!.Code).ThenBy(c => c.Equipment!.Code).ToListAsync(ct);
        return rows.Select(c => c.ToDto()).ToList();
    }

    public async Task<CompatibilityDto> AddCompatibilityAsync(CompatibilityUpsertDto dto, CancellationToken ct = default)
    {
        var tractor = await RequireAsync(Db.Tractors, dto.TractorId, "Tractor", ct);
        var equipment = await RequireAsync(Db.Equipment, dto.EquipmentId, "Equipment", ct);

        if (tractor.Horsepower < equipment.MinimumTractorHp)
            throw new BusinessRuleException("INSUFFICIENT_HP",
                $"Tractor {tractor.Code} delivers {tractor.Horsepower} hp but {equipment.Name} needs at least {equipment.MinimumTractorHp} hp.");
        if (await Db.Compatibilities.AnyAsync(c => c.TractorId == dto.TractorId && c.EquipmentId == dto.EquipmentId, ct))
            throw new BusinessRuleException("DUPLICATE", "That tractor/equipment pairing already exists.");

        var entity = new TractorEquipmentCompatibility
        {
            CompanyId = tractor.CompanyId,
            TractorId = dto.TractorId,
            EquipmentId = dto.EquipmentId,
            IsRecommended = dto.IsRecommended,
            Remarks = dto.Remarks
        };
        Db.Compatibilities.Add(entity);
        await Db.SaveChangesAsync(ct);

        return (await Db.Compatibilities.Include(c => c.Tractor).Include(c => c.Equipment)
            .AsNoTracking().FirstAsync(c => c.Id == entity.Id, ct)).ToDto();
    }

    public async Task RemoveCompatibilityAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Compatibilities, id, "Compatibility rule", ct);
        Db.Compatibilities.Remove(entity);
        await Db.SaveChangesAsync(ct);
    }

    // ---------------------------------------------------------------- operators

    public async Task<PagedResult<OperatorDto>> GetOperatorsAsync(QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.Operators.Include(o => o.Farm).Include(o => o.WorkTeam).AsNoTracking().AsQueryable();
        if (!q.IncludeInactive) query = query.Where(o => o.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(o => o.Code.Contains(q.Search) || o.FullName.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<Operator, object>>>
        {
            ["code"] = o => o.Code,
            ["name"] = o => o.FullName,
            ["skill"] = o => o.Skill
        };
        return await query.ApplySort(q, sorts, o => o.Code).ToPagedResultAsync(q, o => o.ToDto(), ct);
    }

    public async Task<OperatorDto> CreateOperatorAsync(OperatorUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.Operators.AnyAsync(o => o.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Operator code '{code}' is already in use.");

        var entity = new Operator();
        dto.ApplyTo(entity);
        entity.CompanyId = User.CompanyId ?? dto.CompanyId;
        Db.Operators.Add(entity);
        await Db.SaveChangesAsync(ct);
        await SyncTeamHeadcountAsync(entity.WorkTeamId, ct);
        return (await Db.Operators.Include(o => o.Farm).Include(o => o.WorkTeam)
            .AsNoTracking().FirstAsync(o => o.Id == entity.Id, ct)).ToDto();
    }

    public async Task<OperatorDto> UpdateOperatorAsync(int id, OperatorUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Operators, id, "Operator", ct);
        var code = dto.Code.Trim();
        if (await Db.Operators.AnyAsync(o => o.Code == code && o.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Operator code '{code}' is already in use.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var previousTeam = entity.WorkTeamId;
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);

        await SyncTeamHeadcountAsync(previousTeam, ct);
        await SyncTeamHeadcountAsync(entity.WorkTeamId, ct);

        return (await Db.Operators.Include(o => o.Farm).Include(o => o.WorkTeam)
            .AsNoTracking().FirstAsync(o => o.Id == id, ct)).ToDto();
    }

    public async Task DeleteOperatorAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Operators, id, "Operator", ct);
        if (await Db.Schedules.AnyAsync(s => s.OperatorId == id && s.Status != ScheduleStatus.Cancelled, ct))
            throw new BusinessRuleException("IN_USE", "This operator has active bookings and cannot be deleted.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
        await SyncTeamHeadcountAsync(entity.WorkTeamId, ct);
    }

    /// <summary>Keeps <see cref="WorkTeam.MemberCount"/> equal to the number of active members.</summary>
    private async Task SyncTeamHeadcountAsync(int? teamId, CancellationToken ct)
    {
        if (teamId is null) return;
        var team = await Db.WorkTeams.FirstOrDefaultAsync(t => t.Id == teamId, ct);
        if (team is null) return;
        team.MemberCount = await Db.Operators.CountAsync(o => o.WorkTeamId == teamId && o.IsActive, ct);
        await Db.SaveChangesAsync(ct);
    }

    // ---------------------------------------------------------------- work teams

    public async Task<PagedResult<WorkTeamDto>> GetWorkTeamsAsync(QueryParameters q, CancellationToken ct = default)
    {
        var query = Db.WorkTeams.Include(t => t.Farm).AsNoTracking().AsQueryable();
        if (!q.IncludeInactive) query = query.Where(t => t.IsActive);
        if (!string.IsNullOrWhiteSpace(q.Search))
            query = query.Where(t => t.Code.Contains(q.Search) || t.Name.Contains(q.Search));

        var sorts = new Dictionary<string, Expression<Func<WorkTeam, object>>>
        {
            ["code"] = t => t.Code,
            ["name"] = t => t.Name,
            ["members"] = t => t.MemberCount
        };
        return await query.ApplySort(q, sorts, t => t.Code).ToPagedResultAsync(q, t => t.ToDto(), ct);
    }

    public async Task<WorkTeamDto> CreateWorkTeamAsync(WorkTeamUpsertDto dto, CancellationToken ct = default)
    {
        var code = dto.Code.Trim();
        if (await Db.WorkTeams.AnyAsync(t => t.Code == code, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Work team code '{code}' is already in use.");

        var entity = new WorkTeam();
        dto.ApplyTo(entity);
        entity.CompanyId = User.CompanyId ?? dto.CompanyId;
        Db.WorkTeams.Add(entity);
        await Db.SaveChangesAsync(ct);
        return (await Db.WorkTeams.Include(t => t.Farm).AsNoTracking().FirstAsync(t => t.Id == entity.Id, ct)).ToDto();
    }

    public async Task<WorkTeamDto> UpdateWorkTeamAsync(int id, WorkTeamUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.WorkTeams, id, "Work team", ct);
        var code = dto.Code.Trim();
        if (await Db.WorkTeams.AnyAsync(t => t.Code == code && t.Id != id, ct))
            throw new BusinessRuleException("DUPLICATE_CODE", $"Work team code '{code}' is already in use.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        var companyId = entity.CompanyId;
        dto.ApplyTo(entity);
        entity.CompanyId = companyId;
        await Db.SaveChangesAsync(ct);
        return (await Db.WorkTeams.Include(t => t.Farm).AsNoTracking().FirstAsync(t => t.Id == id, ct)).ToDto();
    }

    public async Task DeleteWorkTeamAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.WorkTeams, id, "Work team", ct);
        if (await Db.Schedules.AnyAsync(s => s.WorkTeamId == id && s.Status != ScheduleStatus.Cancelled, ct))
            throw new BusinessRuleException("IN_USE", "This work team has active bookings and cannot be deleted.");
        SoftDelete(entity);
        entity.IsActive = false;
        await Db.SaveChangesAsync(ct);
    }
}
