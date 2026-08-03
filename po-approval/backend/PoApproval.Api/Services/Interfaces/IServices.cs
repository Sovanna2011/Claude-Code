using PoApproval.Api.DTOs;
using PoApproval.Api.Security;
using PoApproval.Api.Services.Sap;

namespace PoApproval.Api.Services.Interfaces;

public interface IAuthService
{
    Task<LoginResponse?> LoginAsync(LoginRequest request, CancellationToken ct = default);
    Task<UserInfoDto?> GetUserInfoAsync(int userId, CancellationToken ct = default);
}

public interface IPurchaseOrderService
{
    Task<IReadOnlyList<PoSummaryDto>> GetWorklistAsync(PoQueryFilter filter, CurrentUser user, CancellationToken ct = default);
    Task<PoDetailDto?> GetDetailAsync(string ebeln, CurrentUser user, CancellationToken ct = default);
}

public interface IReleaseService
{
    Task ReleaseAsync(string ebeln, ReleaseActionRequest request, CurrentUser user, CancellationToken ct = default);
    Task RejectAsync(string ebeln, RejectActionRequest request, CurrentUser user, CancellationToken ct = default);
}

public interface IValueHelpService
{
    Task<IReadOnlyList<ValueHelpItemDto>> GetAsync(string name, CancellationToken ct = default);
}
