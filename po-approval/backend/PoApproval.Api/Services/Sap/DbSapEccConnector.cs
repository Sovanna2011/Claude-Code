using Microsoft.EntityFrameworkCore;
using PoApproval.Api.Data;
using PoApproval.Api.Models;

namespace PoApproval.Api.Services.Sap;

/// <summary>
/// <see cref="ISapEccConnector"/> backed by the SQL Server replica of the SAP
/// purchasing tables. This is the default, self-contained implementation used
/// when Sap:UseLocalMirror is true. Swap in an NCo/RFC implementation to talk
/// to a live ECC system without changing the business services.
/// </summary>
public class DbSapEccConnector : ISapEccConnector
{
    private readonly PoDbContext _db;

    public DbSapEccConnector(PoDbContext db) => _db = db;

    public async Task<IReadOnlyList<Ekko>> GetHeadersAsync(PoQueryFilter filter, CancellationToken ct = default)
    {
        var q = _db.Ekko.AsNoTracking().Include(k => k.Vendor).AsQueryable();

        if (filter.OnlyPending)
            q = q.Where(k => k.FRGRL == "X");   // release not yet complete

        if (!string.IsNullOrWhiteSpace(filter.PurchasingGroup))
            q = q.Where(k => k.EKGRP == filter.PurchasingGroup);

        if (!string.IsNullOrWhiteSpace(filter.Search))
        {
            var s = filter.Search.Trim();
            q = q.Where(k => k.EBELN.Contains(s) ||
                             (k.Vendor != null && k.Vendor.NAME1.Contains(s)));
        }

        return await q.OrderByDescending(k => k.BEDAT).ThenBy(k => k.EBELN).ToListAsync(ct);
    }

    public async Task<Ekko?> GetHeaderAsync(string ebeln, bool forUpdate = false, CancellationToken ct = default)
    {
        var q = _db.Ekko.Include(k => k.Vendor).AsQueryable();
        if (!forUpdate) q = q.AsNoTracking();
        return await q.FirstOrDefaultAsync(k => k.EBELN == ebeln, ct);
    }

    public async Task<IReadOnlyList<Ekpo>> GetItemsAsync(string ebeln, CancellationToken ct = default)
        => await _db.Ekpo.AsNoTracking()
            .Where(i => i.EBELN == ebeln)
            .OrderBy(i => i.EBELP)
            .ToListAsync(ct);

    public async Task PostReleaseActionAsync(Ekko header, ReleaseLog log, CancellationToken ct = default)
    {
        // header is a tracked entity from GetHeaderAsync(forUpdate: true); EF
        // writes the modified release-strategy fields. The log row and the
        // header change commit in one transaction (one SaveChanges call).
        _db.ReleaseLog.Add(log);
        await _db.SaveChangesAsync(ct);
    }
}
