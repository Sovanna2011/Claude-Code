namespace SugarcanePlanning.Contracts.Reports;

/// <summary>Every report the system can produce (section 20).</summary>
public enum ReportKey
{
    PlantingProjection = 0,
    MonthlyPlan = 1,
    FarmZoneBlockPlan = 2,
    NewPlantingAndRatoon = 3,
    ActivitySchedule = 4,
    ActivityCalendar = 5,
    TractorRequirement = 6,
    TractorUtilization = 7,
    TractorShortage = 8,
    EquipmentRequirement = 9,
    EquipmentUtilization = 10,
    EquipmentShortage = 11,
    SeedCane = 12,
    Fertilizer = 13,
    Chemicals = 14,
    MaterialShortage = 15,
    Fuel = 16,
    Labor = 17,
    CapacityAnalysis = 18,
    DelayedActivities = 19,
    RevisionComparison = 20,
    ProjectionVsActual = 21
}

public enum ExportFormat { Json = 0, Pdf = 1, Excel = 2 }

/// <summary>Filter / sort / group parameters shared by every report (section 20).</summary>
public class ReportRequest
{
    public ReportKey Report { get; set; }
    public int? ProjectionId { get; set; }
    public int? SeasonId { get; set; }
    public int? EstateId { get; set; }
    public int? FarmId { get; set; }
    public int? ZoneId { get; set; }
    public int? BlockId { get; set; }
    public int? ActivityId { get; set; }
    public int? MaterialId { get; set; }
    public int? TractorId { get; set; }
    public int? EquipmentId { get; set; }
    public int? CompareToProjectionId { get; set; }
    public DateOnly? FromDate { get; set; }
    public DateOnly? ToDate { get; set; }
    public string? GroupBy { get; set; }
    public string? SortBy { get; set; }
    public bool SortDescending { get; set; }

    public string ToQueryString()
    {
        var parts = new List<string>();
        void Add(string k, object? v) { if (v is not null) parts.Add($"{k}={Uri.EscapeDataString(v.ToString()!)}"); }
        Add("projectionId", ProjectionId);
        Add("seasonId", SeasonId);
        Add("estateId", EstateId);
        Add("farmId", FarmId);
        Add("zoneId", ZoneId);
        Add("blockId", BlockId);
        Add("activityId", ActivityId);
        Add("materialId", MaterialId);
        Add("tractorId", TractorId);
        Add("equipmentId", EquipmentId);
        Add("compareToProjectionId", CompareToProjectionId);
        Add("fromDate", FromDate?.ToString("yyyy-MM-dd"));
        Add("toDate", ToDate?.ToString("yyyy-MM-dd"));
        Add("groupBy", GroupBy);
        Add("sortBy", SortBy);
        if (SortDescending) Add("sortDescending", true);
        return string.Join('&', parts);
    }
}

/// <summary>
/// Generic tabular report payload. The same shape feeds the on-screen grid, the print
/// preview, the PDF writer and the Excel writer.
/// </summary>
public class ReportResultDto
{
    public ReportKey Report { get; set; }
    public string Title { get; set; } = string.Empty;
    public string? Subtitle { get; set; }
    public DateTime GeneratedAtUtc { get; set; }
    public string GeneratedBy { get; set; } = string.Empty;
    public List<ReportColumnDto> Columns { get; set; } = new();
    public List<ReportRowDto> Rows { get; set; } = new();
    public Dictionary<string, string> Filters { get; set; } = new();
    public Dictionary<string, decimal> Totals { get; set; } = new();
}

public class ReportColumnDto
{
    public string Key { get; set; } = string.Empty;
    public string Header { get; set; } = string.Empty;
    /// <summary>text | number | date | percent | status.</summary>
    public string DataType { get; set; } = "text";
    public string? Format { get; set; }
    public bool IsNumeric => DataType is "number" or "percent";
    public int Width { get; set; } = 1;
}

public class ReportRowDto
{
    public Dictionary<string, string?> Values { get; set; } = new();
    /// <summary>Grouping caption; rows sharing a group are rendered under one header.</summary>
    public string? Group { get; set; }
    public bool IsSubtotal { get; set; }
}
