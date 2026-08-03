namespace HRModule.Api.DTOs;

/// <summary>Summary row for the employee list (PA20 header level).</summary>
public class EmployeeSummaryDto
{
    public int Pernr { get; set; }
    public string FullName { get; set; } = string.Empty;
    public string? OrgUnitName { get; set; }
    public string? PositionName { get; set; }
    public string? Email { get; set; }
    public string? EmploymentStatus { get; set; }
    public DateTime? HireDate { get; set; }
}

/// <summary>Full employee master detail across the key infotypes (valid on key date).</summary>
public class EmployeeDetailDto
{
    public int Pernr { get; set; }
    public DateTime KeyDate { get; set; }
    public PersonalDataDto? PersonalData { get; set; }
    public OrgAssignmentDto? OrgAssignment { get; set; }
    public AddressDto? Address { get; set; }
    public WorkingTimeDto? WorkingTime { get; set; }
    public BasicPayDto? BasicPay { get; set; }
    public List<CommunicationDto> Communications { get; set; } = new();
    public List<BankDetailDto> BankDetails { get; set; } = new();

    // Extended infotypes
    public ContractDto? Contract { get; set; }                       // IT0016
    public List<FamilyMemberDto> Family { get; set; } = new();        // IT0021
    public List<EducationDto> Education { get; set; } = new();        // IT0022
    public List<WorkExperienceDto> WorkExperience { get; set; } = new(); // IT0023
    public List<QualificationDto> Qualifications { get; set; } = new();  // IT0024
    public List<AttendanceDto> Attendances { get; set; } = new();       // IT2002
    public List<MonitoringDateDto> MonitoringDates { get; set; } = new();// IT0019
}

/// <summary>IT0016 - Contract Elements.</summary>
public class ContractDto
{
    public string? ContractTypeKey { get; set; }  // CTTYP
    public string? ContractType { get; set; }     // resolved text
    public decimal? ProbationMonths { get; set; } // PRBEZ
    public decimal? NoticeEmployer { get; set; }  // KDGFB
    public decimal? NoticeEmployee { get; set; }  // KDGF2
    public DateTime Begda { get; set; }
    public DateTime Endda { get; set; }
}

/// <summary>IT0021 - Family Member / Dependent.</summary>
public class FamilyMemberDto
{
    public string RelationKey { get; set; } = string.Empty; // SUBTY
    public string? Relation { get; set; }                   // resolved text
    public string? FirstName { get; set; }                  // FAVOR
    public string? LastName { get; set; }                   // FANAM
    public DateTime? BirthDate { get; set; }                // FGBDT
    public string? GenderKey { get; set; }                  // FASEX
    public string? Gender { get; set; }
    public string? BirthCountry { get; set; }               // FGBLD
}

/// <summary>IT0022 - Education.</summary>
public class EducationDto
{
    public string EstablishmentKey { get; set; } = string.Empty; // SUBTY
    public string? Establishment { get; set; }                   // resolved text
    public string? Certificate { get; set; }   // SLABS
    public string? Institute { get; set; }     // INSTI
    public string? Country { get; set; }       // SLAND
    public string? Major { get; set; }         // SFACH
    public string? Grade { get; set; }         // SLGRA
    public DateTime Begda { get; set; }
    public DateTime Endda { get; set; }
}

/// <summary>IT0023 - Previous Employer (work experience).</summary>
public class WorkExperienceDto
{
    public string? Employer { get; set; }   // ARBGB
    public string? Place { get; set; }      // ORT01
    public string? Country { get; set; }    // LAND1
    public string? Task { get; set; }       // TASK
    public string? Industry { get; set; }   // BRANC
    public DateTime Begda { get; set; }
    public DateTime Endda { get; set; }
}

/// <summary>IT0024 - Qualification / Skill.</summary>
public class QualificationDto
{
    public string GroupKey { get; set; } = string.Empty; // SUBTY
    public string? Group { get; set; }                   // resolved text
    public string Qualification { get; set; } = string.Empty; // QUALI
    public int? Proficiency { get; set; }                // AUSPR (0-9)
}

/// <summary>IT2002 - Attendance.</summary>
public class AttendanceDto
{
    public string TypeKey { get; set; } = string.Empty; // AWART
    public string? Type { get; set; }                   // resolved text
    public DateTime Begda { get; set; }
    public DateTime Endda { get; set; }
    public decimal? Days { get; set; }   // ABWTG
    public decimal? Hours { get; set; }  // STDAZ
}

/// <summary>IT0019 - Monitoring of Dates.</summary>
public class MonitoringDateDto
{
    public string TaskKey { get; set; } = string.Empty; // SUBTY
    public string? Task { get; set; }                   // resolved text
    public DateTime Date { get; set; }                  // TERMN
    public DateTime? Reminder { get; set; }             // MNDAT
}

public class PersonalDataDto
{
    public string? FormOfAddress { get; set; }   // ANRED
    public string LastName { get; set; } = string.Empty;   // NACHN
    public string FirstName { get; set; } = string.Empty;  // VORNA
    public string? MiddleName { get; set; }      // MIDNM
    public DateTime? BirthDate { get; set; }     // GBDAT
    public string? BirthPlace { get; set; }      // GBORT
    public string? GenderKey { get; set; }       // GESCH
    public string? Gender { get; set; }          // resolved text
    public string? Nationality { get; set; }     // NATIO
    public string? MaritalStatusKey { get; set; }// FAMST
    public string? MaritalStatus { get; set; }   // resolved text
    public DateTime Begda { get; set; }
    public DateTime Endda { get; set; }
}

public class OrgAssignmentDto
{
    public string? CompanyCode { get; set; }     // BUKRS
    public string? CompanyName { get; set; }
    public string? PersonnelArea { get; set; }   // WERKS
    public string? PersonnelAreaName { get; set; }
    public string? PersonnelSubarea { get; set; }// BTRTL
    public string? EmployeeGroup { get; set; }   // PERSG
    public string? EmployeeGroupName { get; set; }
    public string? EmployeeSubgroup { get; set; }// PERSK
    public string? EmployeeSubgroupName { get; set; }
    public int? OrgUnitId { get; set; }          // ORGEH
    public string? OrgUnitName { get; set; }
    public int? PositionId { get; set; }         // PLANS
    public string? PositionName { get; set; }
    public int? JobId { get; set; }              // STELL
    public string? CostCenter { get; set; }      // KOSTL
    public DateTime Begda { get; set; }
    public DateTime Endda { get; set; }
}

public class AddressDto
{
    public string? Street { get; set; }   // STRAS
    public string? City { get; set; }     // ORT01
    public string? PostalCode { get; set; }// PSTLZ
    public string? Country { get; set; }  // LAND1
    public string? State { get; set; }    // STATE
    public string? Telephone { get; set; }// TELNR
}

public class WorkingTimeDto
{
    public string? WorkScheduleRule { get; set; } // SCHKZ
    public decimal? EmploymentPercent { get; set; }// EMPCT
    public decimal? WeeklyHours { get; set; }      // WOSTD
}

public class BasicPayDto
{
    public string? PayScaleType { get; set; }   // TRFAR
    public string? PayScaleArea { get; set; }   // TRFGB
    public string? PayScaleGroup { get; set; }  // TRFGR
    public decimal? CapacityUtilization { get; set; } // BSGRD
    public string? Currency { get; set; }       // WAERS
    public decimal? AnnualSalary { get; set; }  // ANSAL
    public List<WageTypeDto> WageTypes { get; set; } = new();
}

public class WageTypeDto
{
    public string WageType { get; set; } = string.Empty; // LGART
    public string? WageTypeText { get; set; }
    public decimal? Amount { get; set; }   // BETRG
    public string? Currency { get; set; }  // WAERS
    public decimal? Number { get; set; }   // ANZHL
}

public class CommunicationDto
{
    public string TypeKey { get; set; } = string.Empty; // SUBTY
    public string? TypeText { get; set; }
    public string? Id { get; set; }        // USRID
    public string? LongId { get; set; }    // USRID_LONG
}

public class BankDetailDto
{
    public string? BankDetailsType { get; set; } // SUBTY
    public string? Payee { get; set; }     // EMFTX
    public string? BankCountry { get; set; }// BANKS
    public string? BankKey { get; set; }   // BANKL
    public string? AccountNumber { get; set; }// BANKN
    public string? Currency { get; set; }  // WAERS
}
