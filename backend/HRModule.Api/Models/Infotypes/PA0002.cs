namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 0002 - Personal Data.</summary>
public class PA0002 : InfotypeBase
{
    public string? ANRED { get; set; }   // Form-of-address key
    public string NACHN { get; set; } = string.Empty; // Last name
    public string VORNA { get; set; } = string.Empty; // First name
    public string? MIDNM { get; set; }   // Middle name
    public string? RUFNM { get; set; }   // Known-as name
    public string? TITEL { get; set; }   // Title
    public DateTime? GBDAT { get; set; } // Date of birth
    public string? GBORT { get; set; }   // Place of birth
    public string? GESCH { get; set; }   // Gender key
    public string? NATIO { get; set; }   // Nationality
    public string? FAMST { get; set; }   // Marital status
    public string? SPRSL { get; set; }   // Language key
}
