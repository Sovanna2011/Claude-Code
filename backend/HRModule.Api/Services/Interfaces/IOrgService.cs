using HRModule.Api.DTOs;

namespace HRModule.Api.Services.Interfaces;

public interface IOrgService
{
    Task<IReadOnlyList<OrgUnitNodeDto>> GetOrgUnitsFlatAsync(DateTime? keyDate, CancellationToken ct = default);
    Task<OrgUnitNodeDto?> GetOrgStructureAsync(int rootOrgId, DateTime? keyDate, CancellationToken ct = default);
    Task<IReadOnlyList<PositionDto>> GetPositionsAsync(int? orgUnitId, DateTime? keyDate, CancellationToken ct = default);
}
