namespace HRModule.Api.Models.OrgManagement;

/// <summary>HRP1001 - Relationships between OM objects.</summary>
public class HRP1001
{
    public string MANDT { get; set; } = "100";
    public string PLVAR { get; set; } = "01";
    public string OTYPE { get; set; } = string.Empty; // Source object type
    public int OBJID { get; set; }                     // Source object ID
    public string ISTAT { get; set; } = "1";
    public DateTime BEGDA { get; set; }
    public DateTime ENDDA { get; set; } = new DateTime(9999, 12, 31);
    public string SEQNR { get; set; } = "000";
    public string RSIGN { get; set; } = string.Empty;  // A / B
    public string RELAT { get; set; } = string.Empty;  // 002,003,007,008,012...
    public string SCLAS { get; set; } = string.Empty;  // Related object type
    public string SOBID { get; set; } = string.Empty;  // Related object ID
    public string? PRIOX { get; set; }
    public DateTime? AEDTM { get; set; }
    public string? UNAME { get; set; }
}
