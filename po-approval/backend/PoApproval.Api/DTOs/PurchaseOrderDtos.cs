namespace PoApproval.Api.DTOs;

/// <summary>Row in the approval worklist (ME28-style).</summary>
public class PoSummaryDto
{
    public string Ebeln { get; set; } = default!;
    public string DocType { get; set; } = default!;
    public string VendorId { get; set; } = default!;
    public string? VendorName { get; set; }
    public string PurchasingGroup { get; set; } = default!;
    public string? PurchasingGroupName { get; set; }
    public string Currency { get; set; } = default!;
    public decimal NetValue { get; set; }
    public DateTime DocDate { get; set; }
    public string? Strategy { get; set; }
    public string? StrategyText { get; set; }
    public string ReleaseIndicator { get; set; } = default!;   // FRGKE
    public string? ReleaseIndicatorText { get; set; }
    public bool ReleaseIncomplete { get; set; }                // FRGRL == 'X'
    public string? NextPendingCode { get; set; }               // FRGCO awaiting sign-off
    public string? NextPendingCodeText { get; set; }
    /// <summary>True when the signed-in user holds the next pending code.</summary>
    public bool CanCurrentUserRelease { get; set; }
}

/// <summary>Full detail for one PO (BAPI_PO_GETDETAIL-style).</summary>
public class PoDetailDto
{
    public string Ebeln { get; set; } = default!;
    public string DocType { get; set; } = default!;
    public string? DocTypeText { get; set; }
    public string CompanyCode { get; set; } = default!;
    public string PurchasingOrg { get; set; } = default!;
    public string? PurchasingOrgName { get; set; }
    public string PurchasingGroup { get; set; } = default!;
    public string? PurchasingGroupName { get; set; }
    public string Currency { get; set; } = default!;
    public DateTime DocDate { get; set; }
    public decimal NetValue { get; set; }
    public string? CreatedBy { get; set; }

    public VendorDto Vendor { get; set; } = default!;
    public List<PoItemDto> Items { get; set; } = new();

    // Release strategy
    public string? ReleaseGroup { get; set; }
    public string? ReleaseGroupText { get; set; }
    public string? Strategy { get; set; }
    public string? StrategyText { get; set; }
    public string ReleaseIndicator { get; set; } = default!;
    public string? ReleaseIndicatorText { get; set; }
    public bool ReleaseIncomplete { get; set; }
    public List<ReleaseStepDto> ReleaseSteps { get; set; } = new();
    public List<ReleaseLogDto> ReleaseLog { get; set; } = new();

    // Convenience flags for the signed-in user
    public string? NextPendingCode { get; set; }
    public bool CanCurrentUserRelease { get; set; }
    public bool CanCurrentUserReject { get; set; }
}

public class VendorDto
{
    public string VendorId { get; set; } = default!;
    public string Name { get; set; } = default!;
    public string? City { get; set; }
    public string? Country { get; set; }
    public string? VatNumber { get; set; }
}

public class PoItemDto
{
    public int ItemNo { get; set; }
    public string ShortText { get; set; } = default!;
    public string? Material { get; set; }
    public string? MaterialGroup { get; set; }
    public string? MaterialGroupText { get; set; }
    public string? Plant { get; set; }
    public string? PlantName { get; set; }
    public string? StorageLocation { get; set; }
    public decimal Quantity { get; set; }
    public string Unit { get; set; } = default!;
    public decimal NetPrice { get; set; }
    public decimal PriceUnit { get; set; }
    public decimal NetValue { get; set; }
    public decimal? GrossValue { get; set; }
    public string? TaxCode { get; set; }
    public DateTime? DeliveryDate { get; set; }
    public bool Deleted { get; set; }
}

/// <summary>
/// A release strategy together with the purchasing documents currently pending
/// under it (SAP ME28-style "release by strategy" worklist).
/// </summary>
public class PendingStrategyGroupDto
{
    public string ReleaseGroup { get; set; } = default!;
    public string? ReleaseGroupText { get; set; }
    public string Strategy { get; set; } = default!;
    public string? StrategyText { get; set; }
    public decimal ValueFrom { get; set; }
    public decimal ValueTo { get; set; }
    public List<ReleaseCodeDto> Codes { get; set; } = new();
    public int Count { get; set; }
    public decimal TotalNetValue { get; set; }
    public string Currency { get; set; } = "EUR";
    public List<PoSummaryDto> PurchaseOrders { get; set; } = new();
}

public class ReleaseStepDto
{
    public int StepNo { get; set; }
    public string Code { get; set; } = default!;         // FRGCO
    public string? CodeText { get; set; }
    public bool IsReleased { get; set; }
    public bool IsCurrentStep { get; set; }              // next pending step
    public string? ReleasedBy { get; set; }
    public DateTime? ReleasedOn { get; set; }
}

public class ReleaseLogDto
{
    public string? Code { get; set; }
    public string Action { get; set; } = default!;
    public string User { get; set; } = default!;
    public DateTime ActedOn { get; set; }
    public string? Note { get; set; }
}
