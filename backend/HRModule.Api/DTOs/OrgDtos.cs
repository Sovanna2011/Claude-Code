namespace HRModule.Api.DTOs;

/// <summary>Node in the organizational structure (org unit).</summary>
public class OrgUnitNodeDto
{
    public int OrgUnitId { get; set; }
    public string? OrgUnitName { get; set; }
    public string? ShortText { get; set; }
    public int? ParentOrgId { get; set; }
    public int Depth { get; set; }
    public string? ManagerPositionName { get; set; }
    public int HeadCount { get; set; }
    public List<OrgUnitNodeDto> Children { get; set; } = new();
}

/// <summary>Position with holder information.</summary>
public class PositionDto
{
    public int PositionId { get; set; }
    public string? PositionName { get; set; }
    public int? OrgUnitId { get; set; }
    public string? OrgUnitName { get; set; }
    public int? JobId { get; set; }
    public string? JobName { get; set; }
    public int? HolderPernr { get; set; }
    public string? HolderName { get; set; }
    public bool IsVacant => HolderPernr is null;
}

/// <summary>Remaining leave balance per quota type.</summary>
public class LeaveBalanceDto
{
    public int Pernr { get; set; }
    public string QuotaType { get; set; } = string.Empty;
    public string? QuotaText { get; set; }
    public DateTime Begda { get; set; }
    public DateTime Endda { get; set; }
    public decimal Entitlement { get; set; }
    public decimal Deducted { get; set; }
    public decimal Remaining { get; set; }
}

/// <summary>Generic key/text pair used for value helps (F4).</summary>
public class ValueHelpDto
{
    public string Key { get; set; } = string.Empty;
    public string? Text { get; set; }
}
