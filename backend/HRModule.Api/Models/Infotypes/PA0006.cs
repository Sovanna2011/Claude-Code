namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 0006 - Addresses. SUBTY = address type (T591A).</summary>
public class PA0006 : InfotypeBase
{
    public string? STRAS { get; set; }   // Street and house number
    public string? ORT01 { get; set; }   // City
    public string? ORT02 { get; set; }   // District
    public string? PSTLZ { get; set; }   // Postal code
    public string? LAND1 { get; set; }   // Country key
    public string? STATE { get; set; }   // Region / state
    public string? TELNR { get; set; }   // Telephone number
}
