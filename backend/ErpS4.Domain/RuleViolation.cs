namespace ErpS4.Domain;

/// <summary>
/// A master-data rule that was broken, reported rather than thrown so a caller
/// receives the complete list in one response.
/// </summary>
/// <param name="Code">Stable code, e.g. <c>BP.ROLE_UNKNOWN</c>.</param>
/// <param name="Message">Message for the user.</param>
/// <param name="Field">Field path the message belongs beside.</param>
/// <param name="Severity">Whether this blocks the operation or only warns.</param>
public sealed record RuleViolation(
    string Code,
    string Message,
    string? Field = null,
    ViolationSeverity Severity = ViolationSeverity.Error)
{
    public bool IsBlocking => Severity == ViolationSeverity.Error;

    public override string ToString() => $"{Code}: {Message}";
}

public enum ViolationSeverity
{
    /// <summary>Refuses the operation.</summary>
    Error = 0,

    /// <summary>Lets the operation through, but the user should see it.</summary>
    Warning = 1,

    /// <summary>Observation only, e.g. a field left empty on purpose.</summary>
    Information = 2,
}
