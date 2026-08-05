using Microsoft.AspNetCore.Identity.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.ChangeTracking;
using Microsoft.EntityFrameworkCore.Storage;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Infrastructure.Identity;

namespace SugarcanePlanning.Infrastructure.Persistence;

/// <summary>
/// EF Core context for the planning schema plus ASP.NET Core Identity. Applies the soft-delete
/// and company-isolation query filters and the SQL Server row-version concurrency token.
/// </summary>
public class AppDbContext : IdentityDbContext<AppUser, AppRole, string>, IAppDbContext
{
    private readonly ICurrentUser? _currentUser;

    public AppDbContext(DbContextOptions<AppDbContext> options, ICurrentUser? currentUser = null) : base(options)
        => _currentUser = currentUser;

    public DbSet<Company> Companies => Set<Company>();
    public DbSet<Estate> Estates => Set<Estate>();
    public DbSet<Farm> Farms => Set<Farm>();
    public DbSet<Zone> Zones => Set<Zone>();
    public DbSet<PlantationBlock> Blocks => Set<PlantationBlock>();

    public DbSet<GrowingSeason> Seasons => Set<GrowingSeason>();
    public DbSet<CaneVariety> Varieties => Set<CaneVariety>();

    public DbSet<PlantingActivity> Activities => Set<PlantingActivity>();
    public DbSet<ActivityDependency> ActivityDependencies => Set<ActivityDependency>();

    public DbSet<PlantingProjection> Projections => Set<PlantingProjection>();
    public DbSet<ProjectionLine> ProjectionLines => Set<ProjectionLine>();
    public DbSet<ProjectionApprovalHistory> ApprovalHistory => Set<ProjectionApprovalHistory>();
    public DbSet<ActivityPlan> ActivityPlans => Set<ActivityPlan>();

    public DbSet<Tractor> Tractors => Set<Tractor>();
    public DbSet<EquipmentItem> Equipment => Set<EquipmentItem>();
    public DbSet<TractorEquipmentCompatibility> Compatibilities => Set<TractorEquipmentCompatibility>();
    public DbSet<Operator> Operators => Set<Operator>();
    public DbSet<WorkTeam> WorkTeams => Set<WorkTeam>();
    public DbSet<ResourceSchedule> Schedules => Set<ResourceSchedule>();

    public DbSet<Material> Materials => Set<Material>();
    public DbSet<ActivityMaterialStandard> MaterialStandards => Set<ActivityMaterialStandard>();
    public DbSet<ActivityMaterialRequirement> MaterialRequirements => Set<ActivityMaterialRequirement>();
    public DbSet<MaterialStock> MaterialStocks => Set<MaterialStock>();

    public DbSet<ActivityActual> ActivityActuals => Set<ActivityActual>();
    public DbSet<ActualMaterialUsage> ActualMaterialUsages => Set<ActualMaterialUsage>();

    public DbSet<PlanningScenario> Scenarios => Set<PlanningScenario>();
    public DbSet<ScenarioAdjustment> ScenarioAdjustments => Set<ScenarioAdjustment>();
    public DbSet<AuditLog> AuditLogs => Set<AuditLog>();

    /// <summary>Company of the signed-in user; null disables the tenant filter (migrations, seeding, jobs).</summary>
    public int? TenantCompanyId => _currentUser?.CompanyId;

    public async Task<IAppTransaction> BeginTransactionAsync(CancellationToken cancellationToken = default)
    {
        // A nested call reuses the ambient transaction instead of failing.
        if (Database.CurrentTransaction is not null)
            return new NoOpTransaction();
        var tx = await Database.BeginTransactionAsync(cancellationToken);
        return new EfTransaction(tx);
    }

    protected override void OnModelCreating(ModelBuilder builder)
    {
        base.OnModelCreating(builder);

        builder.HasDefaultSchema("planning");
        builder.ApplyConfigurationsFromAssembly(typeof(AppDbContext).Assembly);

        var isSqlServer = Database.IsSqlServer();

        foreach (var entityType in builder.Model.GetEntityTypes())
        {
            var clrType = entityType.ClrType;
            if (!typeof(BaseEntity).IsAssignableFrom(clrType)) continue;

            // rowversion is a SQL Server native type; other providers get a plain concurrency token.
            var rowVersion = builder.Entity(clrType).Property(nameof(BaseEntity.RowVersion));
            if (isSqlServer) rowVersion.IsRowVersion();
            else rowVersion.IsConcurrencyToken();

            builder.Entity(clrType).Property(nameof(BaseEntity.CreatedBy)).HasMaxLength(100);
            builder.Entity(clrType).Property(nameof(BaseEntity.ModifiedBy)).HasMaxLength(100);
            builder.Entity(clrType).Property(nameof(BaseEntity.DeletedBy)).HasMaxLength(100);
            builder.Entity(clrType).HasIndex(nameof(BaseEntity.IsDeleted));

            ApplyQueryFilter(builder, clrType);
        }

        // Identity tables live in the same schema with readable names.
        builder.Entity<AppUser>().ToTable("Users");
        builder.Entity<AppRole>().ToTable("Roles");
        builder.Entity<Microsoft.AspNetCore.Identity.IdentityUserRole<string>>().ToTable("UserRoles");
        builder.Entity<Microsoft.AspNetCore.Identity.IdentityUserClaim<string>>().ToTable("UserClaims");
        builder.Entity<Microsoft.AspNetCore.Identity.IdentityUserLogin<string>>().ToTable("UserLogins");
        builder.Entity<Microsoft.AspNetCore.Identity.IdentityUserToken<string>>().ToTable("UserTokens");
        builder.Entity<Microsoft.AspNetCore.Identity.IdentityRoleClaim<string>>().ToTable("RoleClaims");
        builder.Entity<AppUser>().Property(u => u.FullName).HasMaxLength(150);
    }

    /// <summary>
    /// Builds <c>e =&gt; !e.IsDeleted &amp;&amp; (TenantCompanyId == null || e.CompanyId == TenantCompanyId)</c>
    /// for every entity, so no query can leak another company's rows (section 21/23).
    /// </summary>
    private void ApplyQueryFilter(ModelBuilder builder, Type clrType)
    {
        var parameter = System.Linq.Expressions.Expression.Parameter(clrType, "e");

        System.Linq.Expressions.Expression body = System.Linq.Expressions.Expression.Not(
            System.Linq.Expressions.Expression.Property(parameter, nameof(BaseEntity.IsDeleted)));

        if (typeof(ICompanyScoped).IsAssignableFrom(clrType))
        {
            var tenantProperty = System.Linq.Expressions.Expression.Property(
                System.Linq.Expressions.Expression.Constant(this), nameof(TenantCompanyId));
            var companyProperty = System.Linq.Expressions.Expression.Property(parameter, nameof(ICompanyScoped.CompanyId));

            var noTenant = System.Linq.Expressions.Expression.Equal(
                tenantProperty, System.Linq.Expressions.Expression.Constant(null, typeof(int?)));
            var sameTenant = System.Linq.Expressions.Expression.Equal(
                System.Linq.Expressions.Expression.Convert(companyProperty, typeof(int?)), tenantProperty);

            body = System.Linq.Expressions.Expression.AndAlso(body,
                System.Linq.Expressions.Expression.OrElse(noTenant, sameTenant));
        }

        builder.Entity(clrType).HasQueryFilter(System.Linq.Expressions.Expression.Lambda(body, parameter));
    }
}

/// <summary>Adapter over an EF Core transaction.</summary>
internal sealed class EfTransaction : IAppTransaction
{
    private readonly IDbContextTransaction _tx;
    public EfTransaction(IDbContextTransaction tx) => _tx = tx;
    public Task CommitAsync(CancellationToken cancellationToken = default) => _tx.CommitAsync(cancellationToken);
    public Task RollbackAsync(CancellationToken cancellationToken = default) => _tx.RollbackAsync(cancellationToken);
    public ValueTask DisposeAsync() => _tx.DisposeAsync();
}

/// <summary>Returned when a transaction is already open; commit/rollback belong to the outer scope.</summary>
internal sealed class NoOpTransaction : IAppTransaction
{
    public Task CommitAsync(CancellationToken cancellationToken = default) => Task.CompletedTask;
    public Task RollbackAsync(CancellationToken cancellationToken = default) => Task.CompletedTask;
    public ValueTask DisposeAsync() => ValueTask.CompletedTask;
}
