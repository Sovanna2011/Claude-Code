using PoApproval.Api.Models;

namespace PoApproval.Api.Services.Sap;

/// <summary>Filter for the purchasing-document worklist.</summary>
public class PoQueryFilter
{
    public string? Search { get; set; }          // EBELN or vendor name fragment
    public bool OnlyPending { get; set; }         // only POs not yet fully released
    public string? PurchasingGroup { get; set; }
}

/// <summary>
/// Boundary to SAP ECC 6.0 EHP8 purchasing documents (EKKO/EKPO) and the
/// release procedure. This is the single seam through which the application
/// reads and writes purchase orders, so the backing system can be swapped
/// without touching the business services.
///
/// The shipped implementation, <see cref="DbSapEccConnector"/>, uses a SQL
/// Server replica of the SAP tables (the same approach the whole module takes:
/// a SAP-faithful schema on an open stack). A production deployment would add
/// an RFC/BAPI implementation that maps the same POCOs to:
///   * BAPI_PO_GETITEMS / BAPI_PO_GETDETAIL  - read headers and items,
///   * BAPI_PO_RELEASE                        - effect / cancel a release,
///   * BAPI_TRANSACTION_COMMIT                - commit the LUW,
/// connecting via the SAP .NET Connector (NCo 3) to the RFC destination
/// configured in appsettings ("Sap" section). Selecting the implementation is
/// a composition-root decision (see Program.cs / Sap:UseLocalMirror).
/// </summary>
public interface ISapEccConnector
{
    /// <summary>Read purchasing document headers matching the filter.</summary>
    Task<IReadOnlyList<Ekko>> GetHeadersAsync(PoQueryFilter filter, CancellationToken ct = default);

    /// <summary>Read one header. <paramref name="forUpdate"/> tracks it for a later save.</summary>
    Task<Ekko?> GetHeaderAsync(string ebeln, bool forUpdate = false, CancellationToken ct = default);

    /// <summary>Read the items of a purchasing document.</summary>
    Task<IReadOnlyList<Ekpo>> GetItemsAsync(string ebeln, CancellationToken ct = default);

    /// <summary>
    /// Persist an updated header (release-strategy fields) together with the
    /// audit log entry for the action, atomically. In an RFC implementation
    /// this maps to BAPI_PO_RELEASE + BAPI_TRANSACTION_COMMIT.
    /// </summary>
    Task PostReleaseActionAsync(Ekko header, ReleaseLog log, CancellationToken ct = default);
}
