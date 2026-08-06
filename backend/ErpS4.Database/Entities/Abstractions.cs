namespace ErpS4.Database.Entities;

/// <summary>
/// Implemented by every tenant-dependent entity. <see cref="ErpDbContext"/>
/// adds a global query filter on <see cref="TenantId"/>, so a query can never
/// silently cross a tenant boundary.
/// </summary>
public interface ITenantScoped
{
    /// <summary>Owning tenant.</summary>
    int TenantId { get; set; }
}

/// <summary>
/// Implemented by insert-only tables - audit trails, change documents and
/// technical logs. <see cref="ErpDbContext"/> refuses to save an update or a
/// delete on them.
/// </summary>
/// <remarks>
/// Posted accounting documents are immutable too, but their rule is richer
/// (clearing fields stay writable, corrections go through reversal), so the
/// posting engine enforces it rather than this marker.
/// </remarks>
public interface IAppendOnly
{
}
