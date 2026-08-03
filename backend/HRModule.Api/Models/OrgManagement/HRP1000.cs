namespace HRModule.Api.Models.OrgManagement;

/// <summary>HRP1000 - Organizational Management object (O/S/C/P).</summary>
public class HRP1000
{
    public string MANDT { get; set; } = "100";  // Client
    public string PLVAR { get; set; } = "01";    // Plan version
    public string OTYPE { get; set; } = string.Empty; // Object type O/S/C/P
    public int OBJID { get; set; }               // Object ID
    public string ISTAT { get; set; } = "1";     // Planning status
    public DateTime BEGDA { get; set; }
    public DateTime ENDDA { get; set; } = new DateTime(9999, 12, 31);
    public string SEQNR { get; set; } = "000";
    public string LANGU { get; set; } = "E";
    public string? SHORT { get; set; }           // Object abbreviation
    public string? STEXT { get; set; }           // Object name
    public DateTime? AEDTM { get; set; }
    public string? UNAME { get; set; }
}
