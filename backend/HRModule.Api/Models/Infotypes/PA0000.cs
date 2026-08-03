namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 0000 - Actions (Massnahmen).</summary>
public class PA0000 : InfotypeBase
{
    /// <summary>Action type (MASSN, check table T529A).</summary>
    public string MASSN { get; set; } = string.Empty;

    /// <summary>Reason for action (MASSG, check table T530).</summary>
    public string? MASSG { get; set; }

    /// <summary>Employment status (STAT2): 0=withdrawn,1=inactive,2=retiree,3=active.</summary>
    public string? STAT2 { get; set; }
}
