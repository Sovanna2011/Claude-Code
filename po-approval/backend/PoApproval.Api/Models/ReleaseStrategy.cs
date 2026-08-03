namespace PoApproval.Api.Models;

/// <summary>T16FG - Release Group.</summary>
public class T16FG
{
    public string FRGGR { get; set; } = default!;    // Release group
    public string? FRGGT { get; set; }               // Description
    public string? CLASS { get; set; }               // Classification class
}

/// <summary>
/// T16FS - Release Strategy. The determination band (ValFrom..ValTo) emulates
/// the classification condition on the net order value CEKKO-GNETW.
/// </summary>
public class T16FS
{
    public string FRGGR { get; set; } = default!;    // Release group
    public string FRGSX { get; set; } = default!;    // Release strategy
    public string? FRGST { get; set; }               // Strategy description
    public string WAERS { get; set; } = "EUR";       // Condition currency
    public decimal ValFrom { get; set; }             // Min net value (inclusive)
    public decimal ValTo { get; set; }               // Max net value (inclusive)
}

/// <summary>T16FC - Release Code (per release group).</summary>
public class T16FC
{
    public string FRGGR { get; set; } = default!;    // Release group
    public string FRGCO { get; set; } = default!;    // Release code
    public string? FRGCT { get; set; }               // Description
}

/// <summary>Ordered release code that makes up a strategy (SAP FRGC1..FRGC8).</summary>
public class T16FSCode
{
    public string FRGGR { get; set; } = default!;    // Release group
    public string FRGSX { get; set; } = default!;    // Release strategy
    public int StepNo { get; set; }                  // Sign-off sequence (1..n)
    public string FRGCO { get; set; } = default!;    // Required release code
}
