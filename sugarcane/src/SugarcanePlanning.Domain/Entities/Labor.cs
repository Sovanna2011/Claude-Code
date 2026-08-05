using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Domain.Entities;

/// <summary>A field worker or machine operator available for scheduling (section 14).</summary>
public class Operator : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public int? EstateId { get; set; }
    public int? FarmId { get; set; }
    public Farm? Farm { get; set; }

    public string Code { get; set; } = string.Empty;
    public string FullName { get; set; } = string.Empty;
    public SkillType Skill { get; set; } = SkillType.GeneralLabor;

    /// <summary>Licence/certification reference for tractor and equipment operators.</summary>
    public string? LicenseNo { get; set; }
    public DateOnly? LicenseExpiry { get; set; }

    public int? WorkTeamId { get; set; }
    public WorkTeam? WorkTeam { get; set; }

    public decimal StandardHoursPerDay { get; set; } = 8m;
    public bool IsActive { get; set; } = true;

    public ICollection<ResourceSchedule> Schedules { get; set; } = new List<ResourceSchedule>();
}

/// <summary>A crew that can be booked as a unit (section 14).</summary>
public class WorkTeam : BaseEntity, ICompanyScoped
{
    public int CompanyId { get; set; }
    public int? FarmId { get; set; }
    public Farm? Farm { get; set; }

    public string Code { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? SupervisorName { get; set; }
    public SkillType PrimarySkill { get; set; } = SkillType.GeneralLabor;

    /// <summary>Head count available for capacity comparison.</summary>
    public int MemberCount { get; set; }
    public decimal StandardHoursPerDay { get; set; } = 8m;
    public bool IsActive { get; set; } = true;

    public ICollection<Operator> Members { get; set; } = new List<Operator>();
}
