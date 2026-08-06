using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Common;

/// <summary>
/// The persistence surface the application layer is allowed to see. Infrastructure supplies
/// the EF Core implementation; tests can supply an in-memory or SQLite-backed one.
/// </summary>
public interface IAppDbContext
{
    DbSet<Company> Companies { get; }
    DbSet<Estate> Estates { get; }
    DbSet<Farm> Farms { get; }
    DbSet<Zone> Zones { get; }
    DbSet<PlantationBlock> Blocks { get; }

    DbSet<GrowingSeason> Seasons { get; }
    DbSet<CaneVariety> Varieties { get; }

    DbSet<PlantingActivity> Activities { get; }
    DbSet<ActivityDependency> ActivityDependencies { get; }

    DbSet<PlantingProjection> Projections { get; }
    DbSet<ProjectionLine> ProjectionLines { get; }
    DbSet<ProjectionApprovalHistory> ApprovalHistory { get; }
    DbSet<ActivityPlan> ActivityPlans { get; }

    DbSet<Tractor> Tractors { get; }
    DbSet<EquipmentItem> Equipment { get; }
    DbSet<TractorEquipmentCompatibility> Compatibilities { get; }
    DbSet<Operator> Operators { get; }
    DbSet<WorkTeam> WorkTeams { get; }
    DbSet<ResourceSchedule> Schedules { get; }

    DbSet<Material> Materials { get; }
    DbSet<ActivityMaterialStandard> MaterialStandards { get; }
    DbSet<ActivityMaterialRequirement> MaterialRequirements { get; }
    DbSet<MaterialStock> MaterialStocks { get; }

    DbSet<ActivityActual> ActivityActuals { get; }
    DbSet<ActualMaterialUsage> ActualMaterialUsages { get; }

    DbSet<PlanningScenario> Scenarios { get; }
    DbSet<ScenarioAdjustment> ScenarioAdjustments { get; }
    DbSet<AuditLog> AuditLogs { get; }

    Task<int> SaveChangesAsync(CancellationToken cancellationToken = default);

    /// <summary>Opens an explicit transaction so multi-step operations commit atomically.</summary>
    Task<IAppTransaction> BeginTransactionAsync(CancellationToken cancellationToken = default);

    /// <summary>
    /// Takes an exclusive lock on each logical key, held until the surrounding transaction ends.
    /// This is what makes the read-then-write rules that no constraint can express — "this
    /// tractor has no overlapping booking", "this block has no approved plan in this window" —
    /// safe under simultaneous requests: without it two callers both read "free" and both write.
    /// Callers must already be inside a transaction. Keys are taken in a fixed order, so two
    /// callers asking for overlapping sets cannot deadlock against each other.
    /// </summary>
    Task LockAsync(IEnumerable<string> keys, CancellationToken cancellationToken = default);

    /// <summary>Change-tracker entry; used to seed the original row version for concurrency checks.</summary>
    Microsoft.EntityFrameworkCore.ChangeTracking.EntityEntry Entry(object entity);
}

/// <summary>Provider-agnostic transaction handle.</summary>
public interface IAppTransaction : IAsyncDisposable
{
    Task CommitAsync(CancellationToken cancellationToken = default);
    Task RollbackAsync(CancellationToken cancellationToken = default);
}

/// <summary>Identity of the caller, resolved from the JWT on every request.</summary>
public interface ICurrentUser
{
    string UserName { get; }
    string? FullName { get; }
    /// <summary>Company the caller belongs to; every query is filtered by it (section 21).</summary>
    int? CompanyId { get; }
    bool IsAuthenticated { get; }
    IReadOnlyCollection<string> Roles { get; }
    string? IpAddress { get; }
    string? DeviceInfo { get; }
    bool IsInRole(string role);
    bool HasPolicy(string policy);
}

/// <summary>Clock abstraction so date-driven logic is testable.</summary>
public interface IDateTimeProvider
{
    DateTime UtcNow { get; }
    DateOnly Today { get; }
}

/// <summary>Writes workflow / export / login events to the audit trail (section 22).</summary>
public interface IAuditService
{
    Task LogAsync(AuditAction action, string tableName, string recordId,
        string? oldValues = null, string? newValues = null, string? remarks = null,
        CancellationToken ct = default);
}

/// <summary>
/// Company working-calendar settings used by every duration calculation.
/// Kept as options so a deployment can switch to a six- or seven-day week.
/// </summary>
public class PlanningOptions
{
    public const string SectionName = "Planning";
    public bool WorkOnSaturday { get; set; } = true;
    public bool WorkOnSunday { get; set; }
    /// <summary>Standard machine hours in one working day.</summary>
    public decimal StandardWorkingHoursPerDay { get; set; } = 8m;
    /// <summary>Coverage below this ratio is a shortage rather than "at risk".</summary>
    public decimal AtRiskThreshold { get; set; } = 0.90m;
    /// <summary>Lead time subtracted from an activity start to get the material delivery date.</summary>
    public int MaterialDeliveryLeadDays { get; set; } = 7;
}
