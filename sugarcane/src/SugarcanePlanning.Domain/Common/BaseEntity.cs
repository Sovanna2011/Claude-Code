namespace SugarcanePlanning.Domain.Common;

/// <summary>
/// Root of every persisted entity. Carries the surrogate key, the created/modified
/// audit stamps, the soft-delete flag and the optimistic-concurrency token.
/// </summary>
public abstract class BaseEntity
{
    public int Id { get; set; }

    // --- Audit fields (populated by the AuditSaveChangesInterceptor) -------------
    public DateTime CreatedAtUtc { get; set; }
    public string? CreatedBy { get; set; }
    public DateTime? ModifiedAtUtc { get; set; }
    public string? ModifiedBy { get; set; }

    // --- Soft deletion ----------------------------------------------------------
    public bool IsDeleted { get; set; }
    public DateTime? DeletedAtUtc { get; set; }
    public string? DeletedBy { get; set; }

    /// <summary>
    /// SQL Server <c>rowversion</c>. Any update sends the value read by the client;
    /// a mismatch raises <see cref="Microsoft.EntityFrameworkCore.DbUpdateConcurrencyException"/>.
    /// </summary>
    public byte[]? RowVersion { get; set; }
}

/// <summary>
/// Marks an entity that belongs to exactly one company. The DbContext applies a
/// global query filter on <see cref="CompanyId"/> so tenants never see each other's rows.
/// </summary>
public interface ICompanyScoped
{
    int CompanyId { get; set; }
}
