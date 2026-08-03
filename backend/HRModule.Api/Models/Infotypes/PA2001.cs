namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 2001 - Absences. SUBTY = absence type (AWART, T554S).</summary>
public class PA2001 : InfotypeBase
{
    public string AWART { get; set; } = string.Empty; // Attendance/absence type
    public decimal? ABWTG { get; set; }   // Absence days
    public decimal? STDAZ { get; set; }   // Absence hours
    public TimeSpan? BEGUZ { get; set; }  // Start time
    public TimeSpan? ENDUZ { get; set; }  // End time
    public bool APPROVED { get; set; }
}
