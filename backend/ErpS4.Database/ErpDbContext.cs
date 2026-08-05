using System.Linq.Expressions;
using ErpS4.Database.Entities;
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.ChangeTracking;

namespace ErpS4.Database;

/// <summary>
/// EF Core context over the S/4HANA-inspired ERP schema.
/// </summary>
/// <remarks>
/// <para>
/// The SQL scripts under <c>database/s4hana</c> own the physical schema; this
/// model is mapped onto it and is deliberately not the source of migrations.
/// Both are generated from <c>docs/s4hana/table_catalogue.csv</c>, so they
/// describe the same tables.
/// </para>
/// <para>
/// The DbSet properties live in the generated <c>ErpDbContext.Sets.cs</c>.
/// </para>
/// </remarks>
public sealed partial class ErpDbContext : DbContext
{
    private readonly ITenantProvider _tenantProvider;
    private readonly ICurrentUser _currentUser;
    private readonly TimeProvider _timeProvider;

    public ErpDbContext(
        DbContextOptions<ErpDbContext> options,
        ITenantProvider tenantProvider,
        ICurrentUser currentUser,
        TimeProvider? timeProvider = null)
        : base(options)
    {
        _tenantProvider = tenantProvider;
        _currentUser = currentUser;
        _timeProvider = timeProvider ?? TimeProvider.System;
    }

    /// <summary>Tenant every query in this context is filtered by.</summary>
    public int CurrentTenantId => _tenantProvider.TenantId;

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.ApplyConfigurationsFromAssembly(typeof(ErpDbContext).Assembly);
        ApplyTenantFilters(modelBuilder);
        base.OnModelCreating(modelBuilder);
    }

    /// <summary>
    /// Adds <c>WHERE TenantId = @currentTenant</c> to every tenant-dependent
    /// entity. Use <c>IgnoreQueryFilters()</c> only in administration code that
    /// genuinely works across tenants.
    /// </summary>
    private void ApplyTenantFilters(ModelBuilder modelBuilder)
    {
        var currentTenant = Expression.Property(
            Expression.Constant(this), nameof(CurrentTenantId));

        foreach (var entityType in modelBuilder.Model.GetEntityTypes())
        {
            if (!typeof(ITenantScoped).IsAssignableFrom(entityType.ClrType))
            {
                continue;
            }

            var parameter = Expression.Parameter(entityType.ClrType, "e");
            var tenantId = Expression.Call(
                typeof(EF),
                nameof(EF.Property),
                new[] { typeof(int) },
                parameter,
                Expression.Constant(nameof(ITenantScoped.TenantId)));

            modelBuilder
                .Entity(entityType.ClrType)
                .HasQueryFilter(
                    Expression.Lambda(Expression.Equal(tenantId, currentTenant), parameter));
        }
    }

    public override int SaveChanges(bool acceptAllChangesOnSuccess)
    {
        ApplyWritePolicies();
        return base.SaveChanges(acceptAllChangesOnSuccess);
    }

    public override Task<int> SaveChangesAsync(
        bool acceptAllChangesOnSuccess,
        CancellationToken cancellationToken = default)
    {
        ApplyWritePolicies();
        return base.SaveChangesAsync(acceptAllChangesOnSuccess, cancellationToken);
    }

    /// <summary>
    /// Stamps the audit columns, defaults the tenant on new rows, protects
    /// creation data from later edits, and rejects writes to append-only
    /// tables.
    /// </summary>
    private void ApplyWritePolicies()
    {
        var now = _timeProvider.GetUtcNow().UtcDateTime;
        var user = _currentUser.UserName;

        foreach (var entry in ChangeTracker.Entries())
        {
            if (entry.State is EntityState.Detached or EntityState.Unchanged)
            {
                continue;
            }

            if (entry.Entity is IAppendOnly && entry.State != EntityState.Added)
            {
                throw new InvalidOperationException(
                    $"{entry.Metadata.GetSchemaQualifiedTableName()} is append only: " +
                    $"{entry.State} is not permitted. Write a new row instead.");
            }

            switch (entry.State)
            {
                case EntityState.Added:
                    if (entry.Entity is ITenantScoped added && added.TenantId == 0)
                    {
                        added.TenantId = _tenantProvider.TenantId;
                    }

                    SetIfMapped(entry, "CreatedAt", now);
                    SetIfMapped(entry, "CreatedBy", user);
                    break;

                case EntityState.Modified:
                    SetIfMapped(entry, "ModifiedAt", now);
                    SetIfMapped(entry, "ModifiedBy", user);
                    Freeze(entry, "CreatedAt");
                    Freeze(entry, "CreatedBy");
                    Freeze(entry, "TenantId");
                    break;
            }
        }
    }

    private static void SetIfMapped(EntityEntry entry, string propertyName, object value)
    {
        if (entry.Metadata.FindProperty(propertyName) is not null)
        {
            entry.Property(propertyName).CurrentValue = value;
        }
    }

    private static void Freeze(EntityEntry entry, string propertyName)
    {
        if (entry.Metadata.FindProperty(propertyName) is not null)
        {
            entry.Property(propertyName).IsModified = false;
        }
    }
}
