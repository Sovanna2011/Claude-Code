using HRModule.Api.DTOs;

namespace HRModule.Api.Services.Interfaces;

public interface IValueHelpService
{
    Task<IReadOnlyList<ValueHelpDto>> GetCompanyCodesAsync(CancellationToken ct = default);
    Task<IReadOnlyList<ValueHelpDto>> GetPersonnelAreasAsync(CancellationToken ct = default);
    Task<IReadOnlyList<ValueHelpDto>> GetEmployeeGroupsAsync(CancellationToken ct = default);
    Task<IReadOnlyList<ValueHelpDto>> GetEmployeeSubgroupsAsync(CancellationToken ct = default);
    Task<IReadOnlyList<ValueHelpDto>> GetAbsenceTypesAsync(CancellationToken ct = default);
    Task<IReadOnlyList<ValueHelpDto>> GetDomainAsync(string domain, CancellationToken ct = default);
}
