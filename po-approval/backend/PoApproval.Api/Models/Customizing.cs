namespace PoApproval.Api.Models;

/// <summary>T001 - Company code.</summary>
public class T001
{
    public string BUKRS { get; set; } = default!;
    public string? BUTXT { get; set; }
    public string? LAND1 { get; set; }
    public string? WAERS { get; set; }
}

/// <summary>T024 - Purchasing group.</summary>
public class T024
{
    public string EKGRP { get; set; } = default!;
    public string? EKNAM { get; set; }
    public string? EKTEL { get; set; }
}

/// <summary>T024E - Purchasing organization.</summary>
public class T024E
{
    public string EKORG { get; set; } = default!;
    public string? EKOTX { get; set; }
    public string? BUKRS { get; set; }
}

/// <summary>T161 - Purchasing document type.</summary>
public class T161
{
    public string BSTYP { get; set; } = default!;
    public string BSART { get; set; } = default!;
    public string? BATXT { get; set; }
}

/// <summary>Generic domain fixed value / short text.</summary>
public class DomainValue
{
    public string Domain { get; set; } = default!;
    public string ValueKey { get; set; } = default!;
    public string? ValueTxt { get; set; }
}

/// <summary>Number range (SAP number range object, e.g. EBELN).</summary>
public class NumberRange
{
    public string RangeObject { get; set; } = default!;
    public long FromNumber { get; set; }
    public long ToNumber { get; set; }
    public long CurrentNumber { get; set; }
}
