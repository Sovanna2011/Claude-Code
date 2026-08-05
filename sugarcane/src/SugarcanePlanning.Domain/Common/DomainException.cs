namespace SugarcanePlanning.Domain.Common;

/// <summary>Raised when an invariant of the planning domain is violated.</summary>
public class BusinessRuleException : Exception
{
    public string Code { get; }

    public BusinessRuleException(string message) : base(message) => Code = "BUSINESS_RULE";

    public BusinessRuleException(string code, string message) : base(message) => Code = code;
}

/// <summary>Raised when a referenced record does not exist (or is filtered out by tenancy).</summary>
public class NotFoundException : Exception
{
    public NotFoundException(string entity, object key)
        : base($"{entity} '{key}' was not found.") { }

    public NotFoundException(string message) : base(message) { }
}

/// <summary>Raised when the caller may authenticate but is not allowed to touch the record.</summary>
public class ForbiddenException : Exception
{
    public ForbiddenException(string message) : base(message) { }
}
