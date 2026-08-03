using HRModule.Api.DTOs;

namespace HRModule.Api.Services.Interfaces;

public interface IEmployeeService
{
    Task<IReadOnlyList<EmployeeSummaryDto>> GetEmployeesAsync(string? search, DateTime? keyDate, CancellationToken ct = default);
    Task<EmployeeDetailDto?> GetEmployeeAsync(int pernr, DateTime? keyDate, CancellationToken ct = default);
    Task<HireEmployeeResponse> HireAsync(HireEmployeeRequest request, CancellationToken ct = default);
    Task<bool> UpdatePersonalDataAsync(int pernr, UpdatePersonalDataRequest request, CancellationToken ct = default);
    Task<bool> ReassignAsync(int pernr, ReassignRequest request, CancellationToken ct = default);
    Task<bool> UpdateAddressAsync(int pernr, UpdateAddressRequest request, CancellationToken ct = default);
    Task<bool> AddFamilyMemberAsync(int pernr, FamilyMemberRequest request, CancellationToken ct = default);
}
