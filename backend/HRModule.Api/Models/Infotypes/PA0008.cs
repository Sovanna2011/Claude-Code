namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 0008 - Basic Pay, with wage type sub-records.</summary>
public class PA0008 : InfotypeBase
{
    public string? TRFAR { get; set; }    // Pay scale type
    public string? TRFGB { get; set; }    // Pay scale area
    public string? TRFGR { get; set; }    // Pay scale group
    public string? TRFST { get; set; }    // Pay scale level
    public decimal? BSGRD { get; set; }   // Capacity utilization level (%)
    public decimal? DIVGV { get; set; }   // Working hours per pay period
    public string? WAERS { get; set; }    // Currency key
    public decimal? ANSAL { get; set; }   // Annual salary

    public List<PA0008WageType> WageTypes { get; set; } = new();
}

/// <summary>Wage type line of Basic Pay (SAP fields LGART/BETRG/ANZHL).</summary>
public class PA0008WageType
{
    public int PERNR { get; set; }
    public DateTime ENDDA { get; set; }
    public int SEQNR { get; set; }
    public int LineNo { get; set; }
    public string LGART { get; set; } = string.Empty;  // Wage type
    public decimal? BETRG { get; set; }                // Amount
    public string? WAERS { get; set; }                 // Currency
    public decimal? ANZHL { get; set; }                // Number / quantity
}
