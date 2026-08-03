using System.ComponentModel.DataAnnotations;

namespace PoApproval.Api.DTOs;

/// <summary>Release (approve) the next pending step of a PO.</summary>
public class ReleaseActionRequest
{
    /// <summary>
    /// Release code to effect. Optional - if omitted the API uses the user's
    /// eligible code for the next pending step.
    /// </summary>
    public string? Code { get; set; }
    public string? Note { get; set; }
}

/// <summary>Reject / cancel a release; resets the strategy from the given code.</summary>
public class RejectActionRequest
{
    public string? Code { get; set; }
    [Required] public string Note { get; set; } = default!;
}

/// <summary>Value-help item (key + text).</summary>
public class ValueHelpItemDto
{
    public string Key { get; set; } = default!;
    public string? Text { get; set; }
}
