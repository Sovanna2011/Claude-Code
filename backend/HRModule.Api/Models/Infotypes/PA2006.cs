namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 2006 - Absence Quotas. SUBTY = quota type (KTART, T556A).</summary>
public class PA2006 : InfotypeBase
{
    public string KTART { get; set; } = string.Empty; // Absence quota type
    public decimal ANZHL { get; set; }    // Quota entitlement number
    public decimal KVERB { get; set; }    // Deducted amount
    public DateTime? DESTA { get; set; }  // Deduction from
    public DateTime? DEEND { get; set; }  // Deduction to
}
