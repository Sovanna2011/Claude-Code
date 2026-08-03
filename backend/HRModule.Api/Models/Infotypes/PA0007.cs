namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 0007 - Planned Working Time.</summary>
public class PA0007 : InfotypeBase
{
    public string? SCHKZ { get; set; }    // Work schedule rule
    public string? ZTERF { get; set; }    // Time management status
    public decimal? EMPCT { get; set; }   // Employment percentage
    public decimal? WOSTD { get; set; }   // Weekly working hours
    public decimal? MOSTD { get; set; }   // Monthly working hours
    public decimal? JRSTD { get; set; }   // Annual working hours
}
