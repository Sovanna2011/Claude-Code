using System.ComponentModel.DataAnnotations;

namespace HRModule.Api.DTOs;

/// <summary>Payload for the SAP "Hiring" action (creates PERNR + IT0000/0001/0002).</summary>
public class HireEmployeeRequest
{
    [Required] public DateTime HireDate { get; set; }
    [Required, MaxLength(40)] public string LastName { get; set; } = string.Empty;
    [Required, MaxLength(40)] public string FirstName { get; set; } = string.Empty;
    public string? Gender { get; set; }          // GESCH
    public DateTime? BirthDate { get; set; }     // GBDAT
    public string CompanyCode { get; set; } = "1000";
    public string PersonnelArea { get; set; } = "1000";
    public string EmployeeGroup { get; set; } = "1";
    public string EmployeeSubgroup { get; set; } = "DU";
    public int? OrgUnit { get; set; }            // ORGEH
    public int? Position { get; set; }           // PLANS
    public string? Email { get; set; }
    public string ChangedBy { get; set; } = "WEBUI";
}

/// <summary>Result of a hiring action.</summary>
public class HireEmployeeResponse
{
    public int Pernr { get; set; }
    public string Message { get; set; } = string.Empty;
}

/// <summary>Payload to update Personal Data (IT0002) - creates a new time slice.</summary>
public class UpdatePersonalDataRequest
{
    [Required] public DateTime Begda { get; set; }
    public string? FormOfAddress { get; set; }
    [Required] public string LastName { get; set; } = string.Empty;
    [Required] public string FirstName { get; set; } = string.Empty;
    public string? MiddleName { get; set; }
    public DateTime? BirthDate { get; set; }
    public string? Gender { get; set; }
    public string? Nationality { get; set; }
    public string? MaritalStatus { get; set; }
    public string ChangedBy { get; set; } = "WEBUI";
}

/// <summary>Payload for an organizational reassignment (delimits IT0001).</summary>
public class ReassignRequest
{
    [Required] public DateTime Begda { get; set; }
    public int? OrgUnit { get; set; }
    public int? Position { get; set; }
    public string? CostCenter { get; set; }
    public string ChangedBy { get; set; } = "WEBUI";
}

/// <summary>Payload to record an absence (IT2001) and deduct from a quota.</summary>
public class AbsenceRequest
{
    [Required] public string AbsenceType { get; set; } = string.Empty; // AWART
    [Required] public DateTime Begda { get; set; }
    [Required] public DateTime Endda { get; set; }
    public decimal? Days { get; set; }
    public string ChangedBy { get; set; } = "WEBUI";
}
