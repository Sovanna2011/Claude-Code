using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Contracts.Auditing;

public class AuditLogDto
{
    public int Id { get; set; }
    public int? CompanyId { get; set; }
    public string UserName { get; set; } = string.Empty;
    public DateTime TimestampUtc { get; set; }
    public AuditAction Action { get; set; }
    public string TableName { get; set; } = string.Empty;
    public string RecordId { get; set; } = string.Empty;
    public string? OldValues { get; set; }
    public string? NewValues { get; set; }
    public string? ChangedColumns { get; set; }
    public string? IpAddress { get; set; }
    public string? DeviceInfo { get; set; }
    public string? Remarks { get; set; }
}

public class AuditLogQuery
{
    public string? UserName { get; set; }
    public string? TableName { get; set; }
    public string? RecordId { get; set; }
    public AuditAction? Action { get; set; }
    public DateOnly? FromDate { get; set; }
    public DateOnly? ToDate { get; set; }
    public int Page { get; set; } = 1;
    public int PageSize { get; set; } = 50;

    public string ToQueryString()
    {
        var parts = new List<string> { $"page={Page}", $"pageSize={PageSize}" };
        void Add(string k, object? v) { if (v is not null) parts.Add($"{k}={Uri.EscapeDataString(v.ToString()!)}"); }
        Add("userName", UserName);
        Add("tableName", TableName);
        Add("recordId", RecordId);
        Add("action", Action);
        Add("fromDate", FromDate?.ToString("yyyy-MM-dd"));
        Add("toDate", ToDate?.ToString("yyyy-MM-dd"));
        return string.Join('&', parts);
    }
}
