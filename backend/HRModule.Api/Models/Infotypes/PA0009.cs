namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 0009 - Bank Details. SUBTY = bank details type (0=main).</summary>
public class PA0009 : InfotypeBase
{
    public string? BNKSA { get; set; }   // Bank details type
    public string? EMFTX { get; set; }   // Payee name
    public string? BANKS { get; set; }   // Bank country key
    public string? BANKL { get; set; }   // Bank key / routing number
    public string? BANKN { get; set; }   // Bank account number / IBAN
    public string? ZLSCH { get; set; }   // Payment method
    public string? WAERS { get; set; }   // Currency
}
