namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 0001 - Organizational Assignment.</summary>
public class PA0001 : InfotypeBase
{
    public string? BUKRS { get; set; }   // Company code
    public string? WERKS { get; set; }   // Personnel area
    public string? BTRTL { get; set; }   // Personnel subarea
    public string? PERSG { get; set; }   // Employee group
    public string? PERSK { get; set; }   // Employee subgroup
    public int? ORGEH { get; set; }      // Organizational unit (HRP1000 O)
    public int? PLANS { get; set; }      // Position (HRP1000 S)
    public int? STELL { get; set; }      // Job (HRP1000 C)
    public string? KOSTL { get; set; }   // Cost center
    public string? ABKRS { get; set; }   // Payroll area
    public string? SACHZ { get; set; }   // Administrator
}
