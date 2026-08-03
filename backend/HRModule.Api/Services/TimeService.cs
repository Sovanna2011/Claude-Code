using HRModule.Api.Data;
using HRModule.Api.DTOs;
using HRModule.Api.Models.Infotypes;
using HRModule.Api.Services.Interfaces;
using Microsoft.EntityFrameworkCore;

namespace HRModule.Api.Services;

/// <summary>
/// Time Management service. Records absences (IT2001) and deducts them from the
/// matching absence quota (IT2006), reproducing the SAP quota-deduction link.
/// </summary>
public class TimeService : ITimeService
{
    private readonly HRDbContext _db;
    public TimeService(HRDbContext db) => _db = db;

    public async Task<IReadOnlyList<LeaveBalanceDto>> GetLeaveBalancesAsync(int pernr, CancellationToken ct = default)
    {
        var quotaTexts = await _db.T554S.AsNoTracking()
            .ToDictionaryAsync(t => t.AWART, t => t.ATEXT, ct);

        var balances = await _db.PA2006.AsNoTracking()
            .Where(q => q.PERNR == pernr)
            .Select(q => new LeaveBalanceDto
            {
                Pernr = q.PERNR, QuotaType = q.KTART,
                Begda = q.BEGDA, Endda = q.ENDDA,
                Entitlement = q.ANZHL, Deducted = q.KVERB,
                Remaining = q.ANZHL - q.KVERB
            })
            .ToListAsync(ct);

        foreach (var b in balances) b.QuotaText = quotaTexts.GetValueOrDefault(b.QuotaType);
        return balances;
    }

    public async Task<bool> RecordAbsenceAsync(int pernr, AbsenceRequest r, CancellationToken ct = default)
    {
        if (!await _db.Employees.AnyAsync(e => e.PERNR == pernr, ct)) return false;
        if (r.Endda < r.Begda) throw new ArgumentException("End date must not be before start date.");

        await using var tx = await _db.Database.BeginTransactionAsync(ct);

        var days = r.Days ?? (decimal)((r.Endda - r.Begda).Days + 1);

        _db.PA2001.Add(new PA2001
        {
            PERNR = pernr, SUBTY = r.AbsenceType, AWART = r.AbsenceType,
            BEGDA = r.Begda, ENDDA = r.Endda, ABWTG = days,
            APPROVED = false, AEDTM = DateTime.Today, UNAME = r.ChangedBy
        });

        // Deduct from a matching quota valid over the absence start date.
        var quota = await _db.PA2006
            .Where(q => q.PERNR == pernr && q.KTART == r.AbsenceType
                        && q.BEGDA <= r.Begda && q.ENDDA >= r.Begda)
            .OrderByDescending(q => q.BEGDA).FirstOrDefaultAsync(ct);
        if (quota is not null)
        {
            if (quota.ANZHL - quota.KVERB < days)
                throw new InvalidOperationException("Insufficient leave quota for this absence.");
            quota.KVERB += days;
            quota.AEDTM = DateTime.Today;
            quota.UNAME = r.ChangedBy;
        }

        await _db.SaveChangesAsync(ct);
        await tx.CommitAsync(ct);
        return true;
    }

    public async Task<bool> RecordAttendanceAsync(int pernr, AttendanceRequest r, CancellationToken ct = default)
    {
        if (!await _db.Employees.AnyAsync(e => e.PERNR == pernr, ct)) return false;
        if (r.Endda < r.Begda) throw new ArgumentException("End date must not be before start date.");

        var days = r.Days ?? (decimal)((r.Endda - r.Begda).Days + 1);
        _db.PA2002.Add(new PA2002
        {
            PERNR = pernr, SUBTY = r.AttendanceType, AWART = r.AttendanceType,
            BEGDA = r.Begda, ENDDA = r.Endda, ABWTG = days, STDAZ = r.Hours,
            AEDTM = DateTime.Today, UNAME = r.ChangedBy
        });
        await _db.SaveChangesAsync(ct);
        return true;
    }
}
