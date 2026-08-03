namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 0105 - Communication. SUBTY = communication type (USRTY).</summary>
public class PA0105 : InfotypeBase
{
    public string? USRID { get; set; }        // Communication ID / value (short)
    public string? USRID_LONG { get; set; }   // Long form (e.g. email address)
}
