namespace HRModule.Api.Models.Customizing;

/// <summary>T001 - Company Codes.</summary>
public class T001
{
    public string BUKRS { get; set; } = string.Empty;
    public string? BUTXT { get; set; }
    public string? LAND1 { get; set; }
    public string? WAERS { get; set; }
}

/// <summary>T500P - Personnel Areas.</summary>
public class T500P
{
    public string WERKS { get; set; } = string.Empty;
    public string? NAME1 { get; set; }
    public string? BUKRS { get; set; }
    public string? MOLGA { get; set; }
}

/// <summary>T001P - Personnel Subareas.</summary>
public class T001P
{
    public string WERKS { get; set; } = string.Empty;
    public string BTRTL { get; set; } = string.Empty;
    public string? BTEXT { get; set; }
}

/// <summary>T501 - Employee Group.</summary>
public class T501
{
    public string PERSG { get; set; } = string.Empty;
    public string? PTEXT { get; set; }
}

/// <summary>T503K - Employee Subgroup.</summary>
public class T503K
{
    public string PERSK { get; set; } = string.Empty;
    public string? PTEXT { get; set; }
}

/// <summary>T528T - Position texts.</summary>
public class T528T
{
    public int PLANS { get; set; }
    public string? PLSTX { get; set; }
}

/// <summary>T529A - Personnel action types.</summary>
public class T529A
{
    public string MASSN { get; set; } = string.Empty;
    public string? MNTXT { get; set; }
}

/// <summary>T530 - Reasons for action.</summary>
public class T530
{
    public string MASSN { get; set; } = string.Empty;
    public string MASSG { get; set; } = string.Empty;
    public string? MGTXT { get; set; }
}

/// <summary>T554S - Absence / Attendance types.</summary>
public class T554S
{
    public string MOABW { get; set; } = "01";
    public string AWART { get; set; } = string.Empty;
    public string? ATEXT { get; set; }
    public string? KENNZ { get; set; }
}

/// <summary>T005 - Countries.</summary>
public class T005
{
    public string LAND1 { get; set; } = string.Empty;
    public string? LANDX { get; set; }
    public string? WAERS { get; set; }
}

/// <summary>T512T - Wage type texts.</summary>
public class T512T
{
    public string LGART { get; set; } = string.Empty;
    public string? LGTXT { get; set; }
}

/// <summary>T547T - Contract type texts.</summary>
public class T547T
{
    public string CTTYP { get; set; } = string.Empty;
    public string? CTTXT { get; set; }
}

/// <summary>Generic domain fixed-value table (GESCH, FAMST, ANRED, USRTY, STAT2).</summary>
public class DomainValue
{
    public string Domain { get; set; } = string.Empty;
    public string ValueKey { get; set; } = string.Empty;
    public string? ValueTxt { get; set; }
}

/// <summary>Number range object state (emulates SAP SNRO).</summary>
public class NumberRange
{
    public string RangeObject { get; set; } = string.Empty;
    public long FromNumber { get; set; }
    public long ToNumber { get; set; }
    public long CurrentNumber { get; set; }
}
