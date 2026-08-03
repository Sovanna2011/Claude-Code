namespace HRModule.Api.Models.Infotypes;

/// <summary>Infotype 0016 - Contract Elements.</summary>
public class PA0016 : InfotypeBase
{
    public string? CTTYP { get; set; }    // Contract type (T547T)
    public decimal? PRBEZ { get; set; }   // Probation period (months)
    public decimal? KDGFB { get; set; }   // Notice period, employer (months)
    public decimal? KDGF2 { get; set; }   // Notice period, employee (months)
}

/// <summary>Infotype 0019 - Monitoring of Dates. SUBTY = task type (TMART).</summary>
public class PA0019 : InfotypeBase
{
    public DateTime TERMN { get; set; }   // Date of task / deadline
    public DateTime? MNDAT { get; set; }  // Reminder date
    public bool REMINDED { get; set; }
}

/// <summary>Infotype 0021 - Family Members / Dependents. SUBTY = family type (FAMSA).</summary>
public class PA0021 : InfotypeBase
{
    public string? FANAM { get; set; }    // Last name
    public string? FAVOR { get; set; }    // First name
    public DateTime? FGBDT { get; set; }  // Date of birth
    public string? FASEX { get; set; }    // Gender
    public string? FGBLD { get; set; }    // Country of birth
    public string? FGBOT { get; set; }    // Place of birth
}

/// <summary>Infotype 0022 - Education. SUBTY = establishment type (SLART).</summary>
public class PA0022 : InfotypeBase
{
    public string? SLABS { get; set; }    // Certificate / degree
    public string? INSTI { get; set; }    // Institute / school
    public string? SLAND { get; set; }    // Country
    public string? SFACH { get; set; }    // Branch of study / major
    public string? SLGRA { get; set; }    // Final grade
}

/// <summary>Infotype 0023 - Other/Previous Employers (work experience).</summary>
public class PA0023 : InfotypeBase
{
    public string? ARBGB { get; set; }    // Previous employer
    public string? ORT01 { get; set; }    // Place
    public string? LAND1 { get; set; }    // Country
    public string? TASK { get; set; }     // Activity / job title
    public string? BRANC { get; set; }    // Industry
}

/// <summary>Infotype 0024 - Qualifications / Skills. SUBTY = qualification group.</summary>
public class PA0024 : InfotypeBase
{
    public string QUALI { get; set; } = string.Empty; // Qualification / skill
    public int? AUSPR { get; set; }                    // Proficiency (0-9)
}

/// <summary>Infotype 2002 - Attendances. SUBTY = attendance type (AWART).</summary>
public class PA2002 : InfotypeBase
{
    public string AWART { get; set; } = string.Empty;  // Attendance type
    public decimal? ABWTG { get; set; }   // Attendance days
    public decimal? STDAZ { get; set; }   // Attendance hours
    public TimeSpan? BEGUZ { get; set; }  // Start time
    public TimeSpan? ENDUZ { get; set; }  // End time
}
