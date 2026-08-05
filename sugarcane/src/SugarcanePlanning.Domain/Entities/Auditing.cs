using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>
/// Immutable audit record (section 22). Written by the save-changes interceptor for data
/// changes and by the services for workflow events.
/// </summary>
public class AuditLog : BaseEntity
{
    public int? CompanyId { get; set; }

    public string UserName { get; set; } = "system";
    public DateTime TimestampUtc { get; set; }
    public AuditAction Action { get; set; }

    public string TableName { get; set; } = string.Empty;
    public string RecordId { get; set; } = string.Empty;

    /// <summary>JSON snapshot of the changed columns before the change.</summary>
    public string? OldValues { get; set; }
    /// <summary>JSON snapshot of the changed columns after the change.</summary>
    public string? NewValues { get; set; }
    public string? ChangedColumns { get; set; }

    public string? IpAddress { get; set; }
    public string? DeviceInfo { get; set; }
    public string? Remarks { get; set; }
}
