using System.ComponentModel.DataAnnotations;

namespace SugarcanePlanning.Contracts.Auth;

public class LoginRequest
{
    [Required, StringLength(100)] public string UserName { get; set; } = string.Empty;
    [Required, StringLength(100)] public string Password { get; set; } = string.Empty;
}

public class LoginResponse
{
    public string Token { get; set; } = string.Empty;
    public DateTime ExpiresAtUtc { get; set; }
    public string UserName { get; set; } = string.Empty;
    public string FullName { get; set; } = string.Empty;
    public int CompanyId { get; set; }
    public string CompanyName { get; set; } = string.Empty;
    public List<string> Roles { get; set; } = new();
    public List<string> Permissions { get; set; } = new();
}

public class RegisterUserRequest
{
    [Required, StringLength(100)] public string UserName { get; set; } = string.Empty;
    [Required, EmailAddress, StringLength(200)] public string Email { get; set; } = string.Empty;
    [Required, StringLength(100, MinimumLength = 8)] public string Password { get; set; } = string.Empty;
    [Required, StringLength(150)] public string FullName { get; set; } = string.Empty;
    [Required] public int CompanyId { get; set; }
    public List<string> Roles { get; set; } = new();
}

public class ChangePasswordRequest
{
    [Required] public string CurrentPassword { get; set; } = string.Empty;
    [Required, StringLength(100, MinimumLength = 8)] public string NewPassword { get; set; } = string.Empty;
}

public class UserDto
{
    public string Id { get; set; } = string.Empty;
    public string UserName { get; set; } = string.Empty;
    public string Email { get; set; } = string.Empty;
    public string FullName { get; set; } = string.Empty;
    public int CompanyId { get; set; }
    public string CompanyName { get; set; } = string.Empty;
    public bool IsActive { get; set; }
    public List<string> Roles { get; set; } = new();
}

/// <summary>The ten roles from section 21.</summary>
public static class AppRoles
{
    public const string SystemAdministrator = "System Administrator";
    public const string PlantationDirector = "Plantation Director";
    public const string PlantationManager = "Plantation Manager";
    public const string FarmManager = "Farm Manager";
    public const string AgriculturalPlanner = "Agricultural Planner";
    public const string MachineryManager = "Machinery Manager";
    public const string MaterialPlanner = "Material Planner";
    public const string FieldSupervisor = "Field Supervisor";
    public const string ManagementApprover = "Management Approver";
    public const string ReportViewer = "Report Viewer";

    public static readonly string[] All =
    {
        SystemAdministrator, PlantationDirector, PlantationManager, FarmManager,
        AgriculturalPlanner, MachineryManager, MaterialPlanner, FieldSupervisor,
        ManagementApprover, ReportViewer
    };
}

/// <summary>
/// Authorization policy names (section 21). Each maps to the set of roles allowed to
/// perform View / Create / Edit / Delete / Submit / Approve / Reject / Revise / Close / Export.
/// </summary>
public static class Policies
{
    public const string View = "perm:view";
    public const string Create = "perm:create";
    public const string Edit = "perm:edit";
    public const string Delete = "perm:delete";
    public const string Submit = "perm:submit";
    public const string Approve = "perm:approve";
    public const string Reject = "perm:reject";
    public const string Revise = "perm:revise";
    public const string Close = "perm:close";
    public const string Export = "perm:export";
    public const string ManageMasterData = "perm:manage-master-data";
    public const string ManageMachinery = "perm:manage-machinery";
    public const string ManageMaterials = "perm:manage-materials";
    public const string Schedule = "perm:schedule";
    public const string RecordActuals = "perm:record-actuals";
    public const string OverrideDependency = "perm:override-dependency";
    public const string Administer = "perm:administer";

    public static readonly string[] All =
    {
        View, Create, Edit, Delete, Submit, Approve, Reject, Revise, Close, Export,
        ManageMasterData, ManageMachinery, ManageMaterials, Schedule, RecordActuals,
        OverrideDependency, Administer
    };

    /// <summary>Single source of truth for which roles satisfy which policy.</summary>
    public static IReadOnlyDictionary<string, string[]> RoleMap { get; } = new Dictionary<string, string[]>
    {
        [View] = AppRoles.All,
        [Export] = AppRoles.All,
        [Create] = new[] { AppRoles.SystemAdministrator, AppRoles.PlantationDirector, AppRoles.PlantationManager, AppRoles.FarmManager, AppRoles.AgriculturalPlanner, AppRoles.MachineryManager, AppRoles.MaterialPlanner },
        [Edit] = new[] { AppRoles.SystemAdministrator, AppRoles.PlantationDirector, AppRoles.PlantationManager, AppRoles.FarmManager, AppRoles.AgriculturalPlanner, AppRoles.MachineryManager, AppRoles.MaterialPlanner },
        [Delete] = new[] { AppRoles.SystemAdministrator, AppRoles.PlantationManager },
        [Submit] = new[] { AppRoles.SystemAdministrator, AppRoles.AgriculturalPlanner, AppRoles.FarmManager, AppRoles.PlantationManager },
        [Approve] = new[] { AppRoles.SystemAdministrator, AppRoles.PlantationManager, AppRoles.PlantationDirector, AppRoles.ManagementApprover },
        [Reject] = new[] { AppRoles.SystemAdministrator, AppRoles.PlantationManager, AppRoles.PlantationDirector, AppRoles.ManagementApprover },
        [Revise] = new[] { AppRoles.SystemAdministrator, AppRoles.AgriculturalPlanner, AppRoles.PlantationManager },
        [Close] = new[] { AppRoles.SystemAdministrator, AppRoles.PlantationManager, AppRoles.PlantationDirector },
        [ManageMasterData] = new[] { AppRoles.SystemAdministrator, AppRoles.PlantationManager, AppRoles.AgriculturalPlanner },
        [ManageMachinery] = new[] { AppRoles.SystemAdministrator, AppRoles.MachineryManager, AppRoles.PlantationManager },
        [ManageMaterials] = new[] { AppRoles.SystemAdministrator, AppRoles.MaterialPlanner, AppRoles.PlantationManager },
        [Schedule] = new[] { AppRoles.SystemAdministrator, AppRoles.MachineryManager, AppRoles.FarmManager, AppRoles.FieldSupervisor, AppRoles.PlantationManager, AppRoles.AgriculturalPlanner },
        [RecordActuals] = new[] { AppRoles.SystemAdministrator, AppRoles.FieldSupervisor, AppRoles.FarmManager, AppRoles.PlantationManager },
        [OverrideDependency] = new[] { AppRoles.SystemAdministrator, AppRoles.PlantationManager, AppRoles.PlantationDirector },
        [Administer] = new[] { AppRoles.SystemAdministrator }
    };
}
