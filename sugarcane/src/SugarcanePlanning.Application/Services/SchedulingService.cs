using System.Linq.Expressions;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Scheduling;
using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>
/// Resource scheduling with the eight conflict checks of section 10. Every booking is
/// validated before it is written; a blocking conflict aborts the save.
/// </summary>
public class SchedulingService : ServiceBase, ISchedulingService
{
    private readonly IAuditService _audit;

    public SchedulingService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock, IAuditService audit)
        : base(db, user, clock) => _audit = audit;

    // -------------------------------------------------------------- validation

    public async Task<ScheduleValidationResultDto> ValidateAsync(ResourceScheduleUpsertDto dto, int? existingScheduleId,
        CancellationToken ct = default)
    {
        var result = new ScheduleValidationResultDto();

        var plan = await Db.ActivityPlans
            .Include(p => p.Activity)
            .Include(p => p.Projection)
            .Include(p => p.Block!).ThenInclude(b => b.Zone)
            .AsNoTracking()
            .FirstOrDefaultAsync(p => p.Id == dto.ActivityPlanId, ct)
            ?? throw new NotFoundException("Activity plan", dto.ActivityPlanId);

        if (dto.PlannedEnd <= dto.PlannedStart)
            throw new BusinessRuleException("TIME_RANGE", "The booking end time must be after the start time.");

        var excludeId = existingScheduleId ?? 0;

        // 8 — scheduling outside the approved planting-plan period.
        // Land preparation legitimately runs before the planting window (negative day offsets),
        // so the permitted window is the projection period widened by this plan's own dates.
        var projection = plan.Projection;
        if (projection is not null)
        {
            var windowStart = plan.PlannedStartDate < projection.PlanningStartDate
                ? plan.PlannedStartDate : projection.PlanningStartDate;
            var windowEnd = plan.PlannedEndDate > projection.PlanningEndDate
                ? plan.PlannedEndDate : projection.PlanningEndDate;

            if (dto.ScheduleDate < windowStart || dto.ScheduleDate > windowEnd)
                result.Conflicts.Add(new ScheduleConflictDto
                {
                    Type = ConflictType.OutsidePlanPeriod,
                    Message = $"{dto.ScheduleDate:yyyy-MM-dd} lies outside the approved plan period " +
                              $"({windowStart:yyyy-MM-dd} … {windowEnd:yyyy-MM-dd}).",
                    IsBlocking = true
                });
        }

        Tractor? tractor = null;
        EquipmentItem? equipment = null;

        // 1 / 4 — tractor double-booking and maintenance
        if (dto.TractorId is not null)
        {
            tractor = await Db.Tractors.Include(t => t.CurrentFarm).AsNoTracking()
                .FirstOrDefaultAsync(t => t.Id == dto.TractorId, ct)
                ?? throw new NotFoundException("Tractor", dto.TractorId);

            if (!tractor.IsBookableOn(dto.ScheduleDate))
                result.Conflicts.Add(new ScheduleConflictDto
                {
                    Type = ConflictType.MachineUnderMaintenance,
                    ResourceCode = tractor.Code,
                    Message = $"Tractor {tractor.Code} is not available on {dto.ScheduleDate:yyyy-MM-dd} ({tractor.Availability}).",
                    IsBlocking = true
                });

            var clash = await FindOverlapAsync(s => s.TractorId == dto.TractorId, dto, excludeId, ct);
            if (clash is not null)
                result.Conflicts.Add(Overlap(ConflictType.TractorDoubleBooking, tractor.Code,
                    $"Tractor {tractor.Code} is already booked", clash));

            // 6 — location conflict: the machine is parked at another farm that day
            var planFarmId = plan.FarmId;
            if (tractor.CurrentFarmId is not null && tractor.CurrentFarmId != planFarmId)
            {
                var sameDayElsewhere = await Db.Schedules.AsNoTracking().AnyAsync(
                    s => s.TractorId == dto.TractorId && s.Id != excludeId
                         && s.ScheduleDate == dto.ScheduleDate
                         && s.Status != ScheduleStatus.Cancelled
                         && s.ActivityPlan!.FarmId != planFarmId, ct);
                result.Conflicts.Add(new ScheduleConflictDto
                {
                    Type = ConflictType.LocationConflict,
                    ResourceCode = tractor.Code,
                    Message = $"Tractor {tractor.Code} is stationed at {tractor.CurrentFarm?.Name ?? "another farm"}; " +
                              $"a transfer is needed for this booking.",
                    IsBlocking = sameDayElsewhere
                });
            }
        }

        // 2 / 4 — equipment double-booking and maintenance
        if (dto.EquipmentId is not null)
        {
            equipment = await Db.Equipment.Include(e => e.CurrentFarm).AsNoTracking()
                .FirstOrDefaultAsync(e => e.Id == dto.EquipmentId, ct)
                ?? throw new NotFoundException("Equipment", dto.EquipmentId);

            if (!equipment.IsBookableOn(dto.ScheduleDate))
                result.Conflicts.Add(new ScheduleConflictDto
                {
                    Type = ConflictType.MachineUnderMaintenance,
                    ResourceCode = equipment.Code,
                    Message = $"Equipment {equipment.Code} is not available on {dto.ScheduleDate:yyyy-MM-dd} ({equipment.Availability}).",
                    IsBlocking = true
                });

            var clash = await FindOverlapAsync(s => s.EquipmentId == dto.EquipmentId, dto, excludeId, ct);
            if (clash is not null)
                result.Conflicts.Add(Overlap(ConflictType.EquipmentDoubleBooking, equipment.Code,
                    $"Equipment {equipment.Code} is already booked", clash));
        }

        // 5 — horsepower, and the explicit compatibility matrix
        if (tractor is not null && equipment is not null)
        {
            if (tractor.Horsepower < equipment.MinimumTractorHp)
                result.Conflicts.Add(new ScheduleConflictDto
                {
                    Type = ConflictType.InsufficientHorsepower,
                    ResourceCode = tractor.Code,
                    Message = $"Tractor {tractor.Code} delivers {tractor.Horsepower} hp but {equipment.Name} needs at least {equipment.MinimumTractorHp} hp.",
                    IsBlocking = true
                });

            var hasRules = await Db.Compatibilities.AsNoTracking().AnyAsync(c => c.EquipmentId == equipment.Id, ct);
            if (hasRules)
            {
                var paired = await Db.Compatibilities.AsNoTracking()
                    .AnyAsync(c => c.EquipmentId == equipment.Id && c.TractorId == tractor.Id, ct);
                if (!paired)
                    result.Conflicts.Add(new ScheduleConflictDto
                    {
                        Type = ConflictType.IncompatibleEquipment,
                        ResourceCode = tractor.Code,
                        Message = $"Tractor {tractor.Code} is not on the compatibility list of {equipment.Name}.",
                        IsBlocking = true
                    });
            }
        }

        // 3 — operator double-booking
        if (dto.OperatorId is not null)
        {
            var op = await Db.Operators.AsNoTracking().FirstOrDefaultAsync(o => o.Id == dto.OperatorId, ct)
                     ?? throw new NotFoundException("Operator", dto.OperatorId);
            if (!op.IsActive)
                result.Conflicts.Add(new ScheduleConflictDto
                {
                    Type = ConflictType.OperatorDoubleBooking,
                    ResourceCode = op.Code,
                    Message = $"Operator {op.FullName} is inactive.",
                    IsBlocking = true
                });

            var clash = await FindOverlapAsync(s => s.OperatorId == dto.OperatorId, dto, excludeId, ct);
            if (clash is not null)
                result.Conflicts.Add(Overlap(ConflictType.OperatorDoubleBooking, op.Code,
                    $"Operator {op.FullName} is already booked", clash));
        }

        // 7 — activity dependency
        var dependencyConflicts = await CheckDependenciesAsync(plan, dto.ScheduleDate, ct);
        foreach (var conflict in dependencyConflicts)
        {
            // A manager with the override policy may schedule anyway, with a recorded reason.
            if (dto.OverrideDependency && User.HasPolicy(Policies.OverrideDependency)
                && !string.IsNullOrWhiteSpace(dto.DependencyOverrideReason))
                conflict.IsBlocking = false;
            result.Conflicts.Add(conflict);
        }

        return result;
    }

    private static ScheduleConflictDto Overlap(ConflictType type, string code, string message, ResourceSchedule clash) => new()
    {
        Type = type,
        ResourceCode = code,
        ConflictingScheduleId = clash.Id,
        ConflictStart = clash.PlannedStart,
        ConflictEnd = clash.PlannedEnd,
        Message = $"{message} from {clash.PlannedStart:yyyy-MM-dd HH:mm} to {clash.PlannedEnd:HH:mm}.",
        IsBlocking = true
    };

    /// <summary>Finds an active booking of the same resource whose time window overlaps.</summary>
    private async Task<ResourceSchedule?> FindOverlapAsync(Expression<Func<ResourceSchedule, bool>> resourceFilter,
        ResourceScheduleUpsertDto dto, int excludeId, CancellationToken ct)
        => await Db.Schedules.AsNoTracking()
            .Where(resourceFilter)
            .Where(s => s.Id != excludeId
                        && s.Status != ScheduleStatus.Cancelled
                        && s.PlannedStart < dto.PlannedEnd
                        && dto.PlannedStart < s.PlannedEnd)
            .FirstOrDefaultAsync(ct);

    /// <summary>
    /// A dependent activity may not start before its blocking predecessors on the same block
    /// are complete (section 6).
    /// </summary>
    private async Task<List<ScheduleConflictDto>> CheckDependenciesAsync(ActivityPlan plan, DateOnly scheduleDate,
        CancellationToken ct)
    {
        var conflicts = new List<ScheduleConflictDto>();
        var deps = await Db.ActivityDependencies.AsNoTracking()
            .Include(d => d.PredecessorActivity)
            .Where(d => d.ActivityId == plan.ActivityId && d.IsBlocking)
            .ToListAsync(ct);
        if (deps.Count == 0) return conflicts;

        foreach (var dep in deps)
        {
            var predecessor = await Db.ActivityPlans.AsNoTracking()
                .Include(p => p.Activity)
                .FirstOrDefaultAsync(p => p.BlockId == plan.BlockId
                                          && p.ProjectionId == plan.ProjectionId
                                          && p.ActivityId == dep.PredecessorActivityId, ct);
            if (predecessor is null) continue;

            if (predecessor.Status != ActivityStatus.Completed)
            {
                var earliest = predecessor.PlannedEndDate.AddDays(dep.LagDays + 1);
                if (scheduleDate < earliest)
                    conflicts.Add(new ScheduleConflictDto
                    {
                        Type = ConflictType.ActivityDependency,
                        Message = $"{plan.Activity?.Name} cannot start on {scheduleDate:yyyy-MM-dd}: " +
                                  $"{dep.PredecessorActivity?.Name} is {predecessor.Status} and finishes {predecessor.PlannedEndDate:yyyy-MM-dd}" +
                                  (dep.LagDays > 0 ? $" plus {dep.LagDays} lag day(s)" : string.Empty) + ".",
                        IsBlocking = true
                    });
            }
        }
        return conflicts;
    }

    // ------------------------------------------------------------------ writes

    /// <summary>
    /// The keys a booking must hold while it is validated and written. The double-booking checks
    /// are read-then-write, so without a lock two simultaneous requests both read "free" and both
    /// insert. Every resource the booking touches is locked, plus the activity plan itself,
    /// because the first booking moves it from Planned to Scheduled.
    /// </summary>
    private static IEnumerable<string> LockKeys(int activityPlanId, int? tractorId, int? equipmentId,
        int? operatorId, int? workTeamId = null)
    {
        yield return $"activity-plan:{activityPlanId}";
        if (tractorId is not null) yield return $"tractor:{tractorId}";
        if (equipmentId is not null) yield return $"equipment:{equipmentId}";
        if (operatorId is not null) yield return $"operator:{operatorId}";
        if (workTeamId is not null) yield return $"work-team:{workTeamId}";
    }

    private static IEnumerable<string> LockKeys(ResourceScheduleUpsertDto dto)
        => LockKeys(dto.ActivityPlanId, dto.TractorId, dto.EquipmentId, dto.OperatorId, dto.WorkTeamId);

    public async Task<ResourceScheduleDto> CreateAsync(ResourceScheduleUpsertDto dto, CancellationToken ct = default)
    {
        await using var tx = await Db.BeginTransactionAsync(ct);
        await Db.LockAsync(LockKeys(dto), ct);

        var validation = await ValidateAsync(dto, null, ct);
        if (!validation.IsValid)
            throw new BusinessRuleException("SCHEDULE_CONFLICT",
                string.Join(" ", validation.Conflicts.Where(c => c.IsBlocking).Select(c => c.Message)));

        var plan = await RequireAsync(Db.ActivityPlans.Include(p => p.Activity), dto.ActivityPlanId, "Activity plan", ct);

        var entity = new ResourceSchedule
        {
            CompanyId = plan.CompanyId,
            ActivityPlanId = plan.Id,
            BlockId = plan.BlockId,
            ActivityId = plan.ActivityId,
            ScheduleDate = dto.ScheduleDate,
            PlannedStart = dto.PlannedStart,
            PlannedEnd = dto.PlannedEnd,
            TractorId = dto.TractorId,
            EquipmentId = dto.EquipmentId,
            OperatorId = dto.OperatorId,
            WorkTeamId = dto.WorkTeamId,
            PlannedAreaHa = dto.PlannedAreaHa,
            DailyTargetHa = plan.DailyTargetHa,
            ExpectedWorkingHours = dto.ExpectedWorkingHours,
            SupervisorName = dto.SupervisorName ?? plan.SupervisorName,
            Status = dto.Status,
            Remarks = dto.Remarks
        };

        if (dto.OverrideDependency && !string.IsNullOrWhiteSpace(dto.DependencyOverrideReason))
        {
            entity.DependencyOverrideApproved = true;
            entity.DependencyOverrideBy = User.UserName;
            entity.DependencyOverrideReason = dto.DependencyOverrideReason;
        }

        Db.Schedules.Add(entity);

        if (plan.Status == ActivityStatus.Planned) plan.Status = ActivityStatus.Scheduled;
        await Db.SaveChangesAsync(ct);

        await _audit.LogAsync(AuditAction.Schedule, nameof(ResourceSchedule), entity.Id.ToString(),
            null, $"plan {plan.Id} on {entity.ScheduleDate:yyyy-MM-dd}", entity.DependencyOverrideReason, ct);

        await tx.CommitAsync(ct);
        return await LoadDtoAsync(entity.Id, ct);
    }

    public async Task<ResourceScheduleDto> UpdateAsync(int id, ResourceScheduleUpsertDto dto, CancellationToken ct = default)
    {
        await using var tx = await Db.BeginTransactionAsync(ct);

        var entity = await RequireAsync(Db.Schedules, id, "Resource schedule", ct);
        if (entity.Status == ScheduleStatus.Completed)
            throw new BusinessRuleException("COMPLETED", "A completed booking can no longer be changed.");

        // Both sides of a reassignment: the resources being released and the ones being taken.
        await Db.LockAsync(LockKeys(dto).Concat(
            LockKeys(entity.ActivityPlanId, entity.TractorId, entity.EquipmentId, entity.OperatorId, entity.WorkTeamId)), ct);

        var validation = await ValidateAsync(dto, id, ct);
        if (!validation.IsValid)
            throw new BusinessRuleException("SCHEDULE_CONFLICT",
                string.Join(" ", validation.Conflicts.Where(c => c.IsBlocking).Select(c => c.Message)));

        ApplyConcurrencyToken(entity, dto.RowVersion);

        var oldResources = $"tractor={entity.TractorId}, equipment={entity.EquipmentId}, operator={entity.OperatorId}";

        entity.ScheduleDate = dto.ScheduleDate;
        entity.PlannedStart = dto.PlannedStart;
        entity.PlannedEnd = dto.PlannedEnd;
        entity.TractorId = dto.TractorId;
        entity.EquipmentId = dto.EquipmentId;
        entity.OperatorId = dto.OperatorId;
        entity.WorkTeamId = dto.WorkTeamId;
        entity.PlannedAreaHa = dto.PlannedAreaHa;
        entity.ExpectedWorkingHours = dto.ExpectedWorkingHours;
        entity.SupervisorName = dto.SupervisorName;
        entity.Status = dto.Status;
        entity.Remarks = dto.Remarks;

        if (dto.OverrideDependency && !string.IsNullOrWhiteSpace(dto.DependencyOverrideReason))
        {
            entity.DependencyOverrideApproved = true;
            entity.DependencyOverrideBy = User.UserName;
            entity.DependencyOverrideReason = dto.DependencyOverrideReason;
        }

        await Db.SaveChangesAsync(ct);
        await _audit.LogAsync(AuditAction.Reassign, nameof(ResourceSchedule), entity.Id.ToString(),
            oldResources, $"tractor={entity.TractorId}, equipment={entity.EquipmentId}, operator={entity.OperatorId}", null, ct);

        await tx.CommitAsync(ct);
        return await LoadDtoAsync(id, ct);
    }

    public async Task CancelAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Schedules, id, "Resource schedule", ct);
        entity.Status = ScheduleStatus.Cancelled;
        await Db.SaveChangesAsync(ct);
        await _audit.LogAsync(AuditAction.Update, nameof(ResourceSchedule), id.ToString(), null, "Cancelled", null, ct);
    }

    private async Task<ResourceScheduleDto> LoadDtoAsync(int id, CancellationToken ct)
        => (await BaseScheduleQuery().FirstAsync(s => s.Id == id, ct)).ToDto();

    private IQueryable<ResourceSchedule> BaseScheduleQuery() => Db.Schedules
        .Include(s => s.Activity)
        .Include(s => s.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm)
        .Include(s => s.Tractor)
        .Include(s => s.Equipment)
        .Include(s => s.Operator)
        .Include(s => s.WorkTeam)
        .AsNoTracking();

    // ------------------------------------------------------------------- reads

    public async Task<PagedResult<ResourceScheduleDto>> GetSchedulesAsync(DateOnly? from, DateOnly? to, int? tractorId,
        int? equipmentId, int? operatorId, QueryParameters q, CancellationToken ct = default)
    {
        var query = BaseScheduleQuery();
        if (from is not null) query = query.Where(s => s.ScheduleDate >= from);
        if (to is not null) query = query.Where(s => s.ScheduleDate <= to);
        if (tractorId is not null) query = query.Where(s => s.TractorId == tractorId);
        if (equipmentId is not null) query = query.Where(s => s.EquipmentId == equipmentId);
        if (operatorId is not null) query = query.Where(s => s.OperatorId == operatorId);
        if (!q.IncludeInactive) query = query.Where(s => s.Status != ScheduleStatus.Cancelled);

        var sorts = new Dictionary<string, Expression<Func<ResourceSchedule, object>>>
        {
            ["date"] = s => s.ScheduleDate,
            ["block"] = s => s.Block!.Code,
            ["status"] = s => s.Status
        };
        return await query.ApplySort(q, sorts, s => s.ScheduleDate).ToPagedResultAsync(q, s => s.ToDto(), ct);
    }

    public async Task<ScheduleBoardDto> GetBoardAsync(DateOnly from, DateOnly to, string groupBy, int? farmId,
        int? blockId, CancellationToken ct = default)
    {
        if (to < from) (from, to) = (to, from);

        var query = BaseScheduleQuery()
            .Where(s => s.ScheduleDate >= from && s.ScheduleDate <= to && s.Status != ScheduleStatus.Cancelled);
        if (farmId is not null) query = query.Where(s => s.Block!.Zone!.FarmId == farmId);
        if (blockId is not null) query = query.Where(s => s.BlockId == blockId);

        var bookings = await query.OrderBy(s => s.ScheduleDate).ThenBy(s => s.PlannedStart).ToListAsync(ct);
        var dtos = bookings.Select(b => b.ToDto()).ToList();

        var board = new ScheduleBoardDto
        {
            RangeStart = from,
            RangeEnd = to,
            GroupBy = string.IsNullOrWhiteSpace(groupBy) ? "day" : groupBy.ToLowerInvariant()
        };

        for (var d = from; d <= to; d = d.AddDays(1))
        {
            var dayBookings = dtos.Where(b => b.ScheduleDate == d).ToList();
            board.Days.Add(new ScheduleCalendarDayDto
            {
                Date = d,
                PlannedAreaHa = dayBookings.Sum(b => b.PlannedAreaHa),
                BookingCount = dayBookings.Count,
                TractorsBooked = dayBookings.Where(b => b.TractorId is not null).Select(b => b.TractorId).Distinct().Count(),
                EquipmentBooked = dayBookings.Where(b => b.EquipmentId is not null).Select(b => b.EquipmentId).Distinct().Count(),
                OperatorsBooked = dayBookings.Where(b => b.OperatorId is not null).Select(b => b.OperatorId).Distinct().Count(),
                Bookings = dayBookings
            });
        }

        var totalDays = PlanningFormulas.WorkingDays(from, to);
        board.Rows = board.GroupBy switch
        {
            "tractor" => BuildRows(dtos, "Tractor", b => b.TractorId, b => b.TractorCode ?? string.Empty, totalDays),
            "equipment" => BuildRows(dtos, "Equipment", b => b.EquipmentId, b => b.EquipmentCode ?? string.Empty, totalDays),
            "operator" => BuildRows(dtos, "Operator", b => b.OperatorId, b => b.OperatorName ?? string.Empty, totalDays),
            "farm" => BuildRows(dtos, "Farm", _ => null, b => b.FarmName, totalDays),
            "block" => BuildRows(dtos, "Block", b => b.BlockId, b => $"{b.BlockCode} — {b.BlockName}", totalDays),
            _ => new List<ScheduleResourceRowDto>()
        };

        return board;
    }

    private static List<ScheduleResourceRowDto> BuildRows(IReadOnlyList<ResourceScheduleDto> bookings, string type,
        Func<ResourceScheduleDto, int?> idSelector, Func<ResourceScheduleDto, string> nameSelector, int totalWorkingDays)
        => bookings
            .Where(b => !string.IsNullOrWhiteSpace(nameSelector(b)))
            .GroupBy(nameSelector)
            .Select(g =>
            {
                var days = g.Select(b => b.ScheduleDate).Distinct().Count();
                return new ScheduleResourceRowDto
                {
                    ResourceType = type,
                    ResourceId = idSelector(g.First()) ?? 0,
                    ResourceCode = g.Key,
                    ResourceName = g.Key,
                    TotalHours = g.Sum(b => b.ExpectedWorkingHours),
                    TotalAreaHa = g.Sum(b => b.PlannedAreaHa),
                    UtilizedDays = days,
                    UtilizationPercent = totalWorkingDays <= 0 ? 0m
                        : Math.Round(days / (decimal)totalWorkingDays * 100m, 2),
                    Bookings = g.OrderBy(b => b.PlannedStart).ToList()
                };
            })
            .OrderBy(r => r.ResourceCode)
            .ToList();
}
