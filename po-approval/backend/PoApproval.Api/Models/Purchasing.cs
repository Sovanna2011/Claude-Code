namespace PoApproval.Api.Models;

/// <summary>
/// EKKO - Purchasing Document Header (SAP MM-PUR). Carries the release-strategy
/// control fields (FRGGR/FRGSX/FRGZU/FRGKE/FRGRL) that drive the approval flow.
/// </summary>
public class Ekko
{
    public string EBELN { get; set; } = default!;   // Purchasing document number
    public string BSTYP { get; set; } = "F";         // Category ('F' = PO)
    public string BSART { get; set; } = default!;    // Document type (T161)
    public string LIFNR { get; set; } = default!;    // Vendor (LFA1)
    public string EKORG { get; set; } = default!;    // Purchasing organization (T024E)
    public string EKGRP { get; set; } = default!;    // Purchasing group (T024)
    public string BUKRS { get; set; } = default!;    // Company code (T001)
    public string WAERS { get; set; } = default!;    // Currency
    public DateTime BEDAT { get; set; }              // Document date
    public decimal RLWRT { get; set; }               // Total net order value (CEKKO-GNETW)

    // Release strategy control
    public string? FRGGR { get; set; }               // Release group (T16FG)
    public string? FRGSX { get; set; }               // Release strategy (T16FS)
    public string FRGZU { get; set; } = "";          // Release status (codes effected)
    public string FRGKE { get; set; } = " ";         // Release indicator (' '/'B'/'R')
    public string FRGRL { get; set; } = " ";         // Release incomplete ('X')

    // Administrative
    public string? ERNAM { get; set; }               // Created by
    public DateTime? AEDAT { get; set; }             // Changed on
    public string MEMORY { get; set; } = " ";        // Held/incomplete ('X')

    public Lfa1? Vendor { get; set; }
    public List<Ekpo> Items { get; set; } = new();
}

/// <summary>EKPO - Purchasing Document Item.</summary>
public class Ekpo
{
    public string EBELN { get; set; } = default!;    // Purchasing document number
    public int EBELP { get; set; }                   // Item number
    public string TXZ01 { get; set; } = default!;    // Short text
    public string? MATNR { get; set; }               // Material number
    public string? MATKL { get; set; }               // Material group
    public string? WERKS { get; set; }               // Plant
    public decimal MENGE { get; set; }               // Quantity
    public string MEINS { get; set; } = "EA";        // Order unit
    public decimal NETPR { get; set; }               // Net price
    public decimal PEINH { get; set; } = 1;          // Price unit
    public decimal NETWR { get; set; }               // Net value
    public decimal? BRTWR { get; set; }              // Gross value (incl. tax)
    public string? MWSKZ { get; set; }               // Tax code
    public string? LGORT { get; set; }               // Storage location
    public DateTime? EINDT { get; set; }             // Item delivery date
    public string LOEKZ { get; set; } = " ";         // Deletion indicator
}

/// <summary>LFA1 - Vendor master (general section).</summary>
public class Lfa1
{
    public string LIFNR { get; set; } = default!;
    public string NAME1 { get; set; } = default!;
    public string? ORT01 { get; set; }
    public string? PSTLZ { get; set; }
    public string? LAND1 { get; set; }
    public string? STCEG { get; set; }
    public string? WAERS { get; set; }
}

/// <summary>Application audit trail of release / reject / reset actions.</summary>
public class ReleaseLog
{
    public long LogId { get; set; }
    public string EBELN { get; set; } = default!;
    public string? FRGCO { get; set; }
    public string ActionTyp { get; set; } = default!;   // RELEASE | REJECT | RESET
    public string UNAME { get; set; } = default!;
    public DateTime ActedOn { get; set; }
    public string? Note { get; set; }
}
