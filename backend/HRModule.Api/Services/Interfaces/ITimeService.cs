using HRModule.Api.DTOs;

namespace HRModule.Api.Services.Interfaces;

public interface ITimeService
{
    Task<IReadOnlyList<LeaveBalanceDto>> GetLeaveBalancesAsync(int pernr, CancellationToken ct = default);
    Task<bool> RecordAbsenceAsync(int pernr, AbsenceRequest request, CancellationToken ct = default);
    Task<bool> RecordAttendanceAsync(int pernr, AttendanceRequest request, CancellationToken ct = default);
}
