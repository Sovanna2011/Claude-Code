using HRModule.Api.Data;
using HRModule.Api.DTOs;
using HRModule.Api.Models;
using HRModule.Api.Models.Infotypes;
using HRModule.Api.Services.Interfaces;
using Microsoft.EntityFrameworkCore;

namespace HRModule.Api.Services;

/// <summary>
/// Personnel Administration service. Reads the record of each infotype valid on
/// a key date (SAP time-constraint 1 read) and performs the standard actions
/// (Hiring, Personal Data change, Organizational reassignment) using the SAP
/// "delimit + insert new time slice" pattern.
/// </summary>
public class EmployeeService : IEmployeeService
{
    private static readonly DateTime HighDate = new(9999, 12, 31);
    private readonly HRDbContext _db;

    public EmployeeService(HRDbContext db) => _db = db;

    public async Task<IReadOnlyList<EmployeeSummaryDto>> GetEmployeesAsync(
        string? search, DateTime? keyDate, CancellationToken ct = default)
    {
        var key = keyDate?.Date ?? DateTime.Today;

        // Personal data valid on the key date, one per PERNR (latest BEGDA).
        var query =
            from em in _db.Employees
            join p2 in _db.PA0002 on em.PERNR equals p2.PERNR
            where p2.BEGDA <= key && p2.ENDDA >= key
            select new { em, p2 };

        if (!string.IsNullOrWhiteSpace(search))
        {
            var s = search.Trim();
            query = query.Where(x =>
                x.p2.NACHN.Contains(s) || x.p2.VORNA.Contains(s) ||
                x.em.PERNR.ToString().Contains(s));
        }

        var rows = await query.AsNoTracking().ToListAsync(ct);

        // Enrich with org assignment + email valid on the key date.
        var pernrs = rows.Select(r => r.em.PERNR).ToList();
        var orgAssignments = await _db.PA0001.AsNoTracking()
            .Where(x => pernrs.Contains(x.PERNR) && x.BEGDA <= key && x.ENDDA >= key)
            .ToListAsync(ct);
        var statuses = await _db.PA0000.AsNoTracking()
            .Where(x => pernrs.Contains(x.PERNR) && x.BEGDA <= key && x.ENDDA >= key)
            .ToListAsync(ct);
        var emails = await _db.PA0105.AsNoTracking()
            .Where(x => pernrs.Contains(x.PERNR) && x.SUBTY == "0010" && x.BEGDA <= key && x.ENDDA >= key)
            .ToListAsync(ct);

        var orgTexts = await LoadOrgTextsAsync(key, ct);
        var statusTexts = await LoadDomainDictAsync("STAT2", ct);

        return rows.Select(r =>
        {
            var org = orgAssignments.Where(a => a.PERNR == r.em.PERNR).OrderByDescending(a => a.BEGDA).FirstOrDefault();
            var st = statuses.Where(a => a.PERNR == r.em.PERNR).OrderByDescending(a => a.BEGDA).FirstOrDefault();
            var mail = emails.Where(a => a.PERNR == r.em.PERNR).OrderByDescending(a => a.BEGDA).FirstOrDefault();
            return new EmployeeSummaryDto
            {
                Pernr = r.em.PERNR,
                FullName = $"{r.p2.VORNA} {r.p2.NACHN}",
                OrgUnitName = org?.ORGEH is int o ? orgTexts.GetValueOrDefault(("O", o)) : null,
                PositionName = org?.PLANS is int p ? orgTexts.GetValueOrDefault(("S", p)) : null,
                Email = mail?.USRID_LONG ?? mail?.USRID,
                EmploymentStatus = st?.STAT2 is string s2 ? statusTexts.GetValueOrDefault(s2) : null,
                HireDate = r.em.HireDate
            };
        })
        .OrderBy(x => x.Pernr)
        .ToList();
    }

    public async Task<EmployeeDetailDto?> GetEmployeeAsync(int pernr, DateTime? keyDate, CancellationToken ct = default)
    {
        var key = keyDate?.Date ?? DateTime.Today;
        var exists = await _db.Employees.AnyAsync(e => e.PERNR == pernr, ct);
        if (!exists) return null;

        var dto = new EmployeeDetailDto { Pernr = pernr, KeyDate = key };

        var p2 = await ValidOn(_db.PA0002, pernr, key, ct);
        if (p2 is not null)
        {
            var gender = await ResolveDomainAsync("GESCH", p2.GESCH, ct);
            var marital = await ResolveDomainAsync("FAMST", p2.FAMST, ct);
            dto.PersonalData = new PersonalDataDto
            {
                FormOfAddress = p2.ANRED, LastName = p2.NACHN, FirstName = p2.VORNA,
                MiddleName = p2.MIDNM, BirthDate = p2.GBDAT, BirthPlace = p2.GBORT,
                GenderKey = p2.GESCH, Gender = gender, Nationality = p2.NATIO,
                MaritalStatusKey = p2.FAMST, MaritalStatus = marital,
                Begda = p2.BEGDA, Endda = p2.ENDDA
            };
        }

        var p1 = await ValidOn(_db.PA0001, pernr, key, ct);
        if (p1 is not null)
        {
            var orgTexts = await LoadOrgTextsAsync(key, ct);
            dto.OrgAssignment = new OrgAssignmentDto
            {
                CompanyCode = p1.BUKRS,
                CompanyName = await _db.T001.Where(t => t.BUKRS == p1.BUKRS).Select(t => t.BUTXT).FirstOrDefaultAsync(ct),
                PersonnelArea = p1.WERKS,
                PersonnelAreaName = await _db.T500P.Where(t => t.WERKS == p1.WERKS).Select(t => t.NAME1).FirstOrDefaultAsync(ct),
                PersonnelSubarea = p1.BTRTL,
                EmployeeGroup = p1.PERSG,
                EmployeeGroupName = await _db.T501.Where(t => t.PERSG == p1.PERSG).Select(t => t.PTEXT).FirstOrDefaultAsync(ct),
                EmployeeSubgroup = p1.PERSK,
                EmployeeSubgroupName = await _db.T503K.Where(t => t.PERSK == p1.PERSK).Select(t => t.PTEXT).FirstOrDefaultAsync(ct),
                OrgUnitId = p1.ORGEH,
                OrgUnitName = p1.ORGEH is int o ? orgTexts.GetValueOrDefault(("O", o)) : null,
                PositionId = p1.PLANS,
                PositionName = p1.PLANS is int p ? orgTexts.GetValueOrDefault(("S", p)) : null,
                JobId = p1.STELL,
                CostCenter = p1.KOSTL,
                Begda = p1.BEGDA, Endda = p1.ENDDA
            };
        }

        var p6 = await ValidOn(_db.PA0006, pernr, key, ct);
        if (p6 is not null)
            dto.Address = new AddressDto
            {
                Street = p6.STRAS, City = p6.ORT01, PostalCode = p6.PSTLZ,
                Country = p6.LAND1, State = p6.STATE, Telephone = p6.TELNR
            };

        var p7 = await ValidOn(_db.PA0007, pernr, key, ct);
        if (p7 is not null)
            dto.WorkingTime = new WorkingTimeDto
            {
                WorkScheduleRule = p7.SCHKZ, EmploymentPercent = p7.EMPCT, WeeklyHours = p7.WOSTD
            };

        var p8 = await _db.PA0008.Include(x => x.WageTypes).AsNoTracking()
            .Where(x => x.PERNR == pernr && x.BEGDA <= key && x.ENDDA >= key)
            .OrderByDescending(x => x.BEGDA).FirstOrDefaultAsync(ct);
        if (p8 is not null)
        {
            var wtTexts = await _db.T512T.AsNoTracking().ToDictionaryAsync(t => t.LGART, t => t.LGTXT, ct);
            dto.BasicPay = new BasicPayDto
            {
                PayScaleType = p8.TRFAR, PayScaleArea = p8.TRFGB, PayScaleGroup = p8.TRFGR,
                CapacityUtilization = p8.BSGRD, Currency = p8.WAERS, AnnualSalary = p8.ANSAL,
                WageTypes = p8.WageTypes.Select(w => new WageTypeDto
                {
                    WageType = w.LGART, WageTypeText = wtTexts.GetValueOrDefault(w.LGART),
                    Amount = w.BETRG, Currency = w.WAERS, Number = w.ANZHL
                }).ToList()
            };
        }

        var commTexts = await LoadDomainDictAsync("USRTY", ct);
        dto.Communications = await _db.PA0105.AsNoTracking()
            .Where(x => x.PERNR == pernr && x.BEGDA <= key && x.ENDDA >= key)
            .Select(x => new CommunicationDto { TypeKey = x.SUBTY, Id = x.USRID, LongId = x.USRID_LONG })
            .ToListAsync(ct);
        foreach (var c in dto.Communications) c.TypeText = commTexts.GetValueOrDefault(c.TypeKey);

        dto.BankDetails = await _db.PA0009.AsNoTracking()
            .Where(x => x.PERNR == pernr && x.BEGDA <= key && x.ENDDA >= key)
            .Select(x => new BankDetailDto
            {
                BankDetailsType = x.SUBTY, Payee = x.EMFTX, BankCountry = x.BANKS,
                BankKey = x.BANKL, AccountNumber = x.BANKN, Currency = x.WAERS
            })
            .ToListAsync(ct);

        await PopulateExtendedAsync(dto, pernr, key, ct);
        return dto;
    }

    /// <summary>Populates the extended infotypes (IT0016/0019/0021/0022/0023/0024/2002).</summary>
    private async Task PopulateExtendedAsync(EmployeeDetailDto dto, int pernr, DateTime key, CancellationToken ct)
    {
        var famTexts = await LoadDomainDictAsync("FAMSA", ct);
        var eduTexts = await LoadDomainDictAsync("SLART", ct);
        var qualTexts = await LoadDomainDictAsync("QUALG", ct);
        var taskTexts = await LoadDomainDictAsync("TMART", ct);
        var genderTexts = await LoadDomainDictAsync("GESCH", ct);

        var c16 = await ValidOn(_db.PA0016, pernr, key, ct);
        if (c16 is not null)
            dto.Contract = new ContractDto
            {
                ContractTypeKey = c16.CTTYP,
                ContractType = await _db.T547T.Where(t => t.CTTYP == c16.CTTYP).Select(t => t.CTTXT).FirstOrDefaultAsync(ct),
                ProbationMonths = c16.PRBEZ, NoticeEmployer = c16.KDGFB, NoticeEmployee = c16.KDGF2,
                Begda = c16.BEGDA, Endda = c16.ENDDA
            };

        dto.Family = (await _db.PA0021.AsNoTracking()
            .Where(x => x.PERNR == pernr && x.BEGDA <= key && x.ENDDA >= key).ToListAsync(ct))
            .Select(x => new FamilyMemberDto
            {
                RelationKey = x.SUBTY, Relation = famTexts.GetValueOrDefault(x.SUBTY),
                FirstName = x.FAVOR, LastName = x.FANAM, BirthDate = x.FGBDT,
                GenderKey = x.FASEX, Gender = x.FASEX is null ? null : genderTexts.GetValueOrDefault(x.FASEX),
                BirthCountry = x.FGBLD
            }).ToList();

        dto.Education = (await _db.PA0022.AsNoTracking()
            .Where(x => x.PERNR == pernr).OrderBy(x => x.BEGDA).ToListAsync(ct))
            .Select(x => new EducationDto
            {
                EstablishmentKey = x.SUBTY, Establishment = eduTexts.GetValueOrDefault(x.SUBTY),
                Certificate = x.SLABS, Institute = x.INSTI, Country = x.SLAND,
                Major = x.SFACH, Grade = x.SLGRA, Begda = x.BEGDA, Endda = x.ENDDA
            }).ToList();

        dto.WorkExperience = await _db.PA0023.AsNoTracking()
            .Where(x => x.PERNR == pernr).OrderBy(x => x.BEGDA)
            .Select(x => new WorkExperienceDto
            {
                Employer = x.ARBGB, Place = x.ORT01, Country = x.LAND1,
                Task = x.TASK, Industry = x.BRANC, Begda = x.BEGDA, Endda = x.ENDDA
            }).ToListAsync(ct);

        dto.Qualifications = (await _db.PA0024.AsNoTracking()
            .Where(x => x.PERNR == pernr && x.BEGDA <= key && x.ENDDA >= key).ToListAsync(ct))
            .Select(x => new QualificationDto
            {
                GroupKey = x.SUBTY, Group = qualTexts.GetValueOrDefault(x.SUBTY),
                Qualification = x.QUALI, Proficiency = x.AUSPR
            }).ToList();

        var attTexts = await _db.T554S.AsNoTracking().ToDictionaryAsync(t => t.AWART, t => t.ATEXT, ct);
        dto.Attendances = (await _db.PA2002.AsNoTracking()
            .Where(x => x.PERNR == pernr).OrderByDescending(x => x.BEGDA).ToListAsync(ct))
            .Select(x => new AttendanceDto
            {
                TypeKey = x.AWART, Type = attTexts.GetValueOrDefault(x.AWART),
                Begda = x.BEGDA, Endda = x.ENDDA, Days = x.ABWTG, Hours = x.STDAZ
            }).ToList();

        dto.MonitoringDates = (await _db.PA0019.AsNoTracking()
            .Where(x => x.PERNR == pernr && x.ENDDA >= key).OrderBy(x => x.TERMN).ToListAsync(ct))
            .Select(x => new MonitoringDateDto
            {
                TaskKey = x.SUBTY, Task = taskTexts.GetValueOrDefault(x.SUBTY),
                Date = x.TERMN, Reminder = x.MNDAT
            }).ToList();
    }

    public async Task<bool> UpdateAddressAsync(int pernr, UpdateAddressRequest r, CancellationToken ct = default)
    {
        if (!await _db.Employees.AnyAsync(e => e.PERNR == pernr, ct)) return false;

        await using var tx = await _db.Database.BeginTransactionAsync(ct);

        var sub = string.IsNullOrEmpty(r.SubType) ? "1" : r.SubType;
        var current = await _db.PA0006
            .Where(x => x.PERNR == pernr && x.SUBTY == sub && x.ENDDA >= r.Begda && x.BEGDA < r.Begda)
            .OrderByDescending(x => x.BEGDA).FirstOrDefaultAsync(ct);
        if (current is not null)
        {
            current.ENDDA = r.Begda.AddDays(-1);
            current.AEDTM = DateTime.Today;
            current.UNAME = r.ChangedBy;
        }

        _db.PA0006.Add(new PA0006
        {
            PERNR = pernr, SUBTY = sub, BEGDA = r.Begda, ENDDA = HighDate,
            STRAS = r.Street, ORT01 = r.City, PSTLZ = r.PostalCode,
            LAND1 = r.Country, STATE = r.State, TELNR = r.Telephone,
            AEDTM = DateTime.Today, UNAME = r.ChangedBy
        });

        await _db.SaveChangesAsync(ct);
        await tx.CommitAsync(ct);
        return true;
    }

    public async Task<bool> AddFamilyMemberAsync(int pernr, FamilyMemberRequest r, CancellationToken ct = default)
    {
        if (!await _db.Employees.AnyAsync(e => e.PERNR == pernr, ct)) return false;

        // Next object identification (OBJPS) within the relation subtype.
        var maxObjps = await _db.PA0021
            .Where(x => x.PERNR == pernr && x.SUBTY == r.RelationType)
            .Select(x => x.OBJPS).ToListAsync(ct);
        var next = (maxObjps.Select(o => int.TryParse(o, out var n) ? n : 0).DefaultIfEmpty(0).Max() + 1)
            .ToString("D2");

        _db.PA0021.Add(new PA0021
        {
            PERNR = pernr, SUBTY = r.RelationType, OBJPS = next,
            BEGDA = r.Begda ?? DateTime.Today, ENDDA = HighDate,
            FANAM = r.LastName, FAVOR = r.FirstName, FGBDT = r.BirthDate,
            FASEX = r.Gender, FGBLD = r.BirthCountry,
            AEDTM = DateTime.Today, UNAME = r.ChangedBy
        });

        await _db.SaveChangesAsync(ct);
        return true;
    }

    public async Task<HireEmployeeResponse> HireAsync(HireEmployeeRequest r, CancellationToken ct = default)
    {
        await using var tx = await _db.Database.BeginTransactionAsync(ct);

        var pernr = await NextNumberAsync("PERNR", ct);

        _db.Employees.Add(new EmployeeMaster { PERNR = pernr, HireDate = r.HireDate, IsActive = true, CreatedOn = DateTime.Now });

        _db.PA0000.Add(new PA0000
        {
            PERNR = pernr, BEGDA = r.HireDate, ENDDA = HighDate,
            MASSN = "01", MASSG = "01", STAT2 = "3",
            AEDTM = DateTime.Today, UNAME = r.ChangedBy
        });
        _db.PA0001.Add(new PA0001
        {
            PERNR = pernr, BEGDA = r.HireDate, ENDDA = HighDate,
            BUKRS = r.CompanyCode, WERKS = r.PersonnelArea,
            PERSG = r.EmployeeGroup, PERSK = r.EmployeeSubgroup,
            ORGEH = r.OrgUnit, PLANS = r.Position,
            AEDTM = DateTime.Today, UNAME = r.ChangedBy
        });
        _db.PA0002.Add(new PA0002
        {
            PERNR = pernr, BEGDA = r.HireDate, ENDDA = HighDate,
            NACHN = r.LastName, VORNA = r.FirstName,
            GBDAT = r.BirthDate, GESCH = r.Gender,
            AEDTM = DateTime.Today, UNAME = r.ChangedBy
        });
        if (!string.IsNullOrWhiteSpace(r.Email))
            _db.PA0105.Add(new PA0105
            {
                PERNR = pernr, SUBTY = "0010", BEGDA = r.HireDate, ENDDA = HighDate,
                USRID = r.Email, USRID_LONG = r.Email, AEDTM = DateTime.Today, UNAME = r.ChangedBy
            });

        await _db.SaveChangesAsync(ct);
        await tx.CommitAsync(ct);

        return new HireEmployeeResponse { Pernr = pernr, Message = $"Employee {pernr} hired successfully." };
    }

    public async Task<bool> UpdatePersonalDataAsync(int pernr, UpdatePersonalDataRequest r, CancellationToken ct = default)
    {
        if (!await _db.Employees.AnyAsync(e => e.PERNR == pernr, ct)) return false;

        await using var tx = await _db.Database.BeginTransactionAsync(ct);

        // Delimit the record open on/after the new begin date (time constraint 1).
        var current = await _db.PA0002
            .Where(x => x.PERNR == pernr && x.ENDDA >= r.Begda && x.BEGDA < r.Begda)
            .OrderByDescending(x => x.BEGDA).FirstOrDefaultAsync(ct);
        if (current is not null)
        {
            current.ENDDA = r.Begda.AddDays(-1);
            current.AEDTM = DateTime.Today;
            current.UNAME = r.ChangedBy;
        }

        _db.PA0002.Add(new PA0002
        {
            PERNR = pernr, BEGDA = r.Begda, ENDDA = HighDate,
            ANRED = r.FormOfAddress, NACHN = r.LastName, VORNA = r.FirstName,
            MIDNM = r.MiddleName, GBDAT = r.BirthDate, GESCH = r.Gender,
            NATIO = r.Nationality, FAMST = r.MaritalStatus,
            AEDTM = DateTime.Today, UNAME = r.ChangedBy
        });

        await _db.SaveChangesAsync(ct);
        await tx.CommitAsync(ct);
        return true;
    }

    public async Task<bool> ReassignAsync(int pernr, ReassignRequest r, CancellationToken ct = default)
    {
        if (!await _db.Employees.AnyAsync(e => e.PERNR == pernr, ct)) return false;

        await using var tx = await _db.Database.BeginTransactionAsync(ct);

        var current = await _db.PA0001
            .Where(x => x.PERNR == pernr && x.ENDDA >= r.Begda && x.BEGDA < r.Begda)
            .OrderByDescending(x => x.BEGDA).FirstOrDefaultAsync(ct);

        // Carry forward fields not being changed.
        var newRec = new PA0001
        {
            PERNR = pernr, BEGDA = r.Begda, ENDDA = HighDate,
            BUKRS = current?.BUKRS, WERKS = current?.WERKS, BTRTL = current?.BTRTL,
            PERSG = current?.PERSG, PERSK = current?.PERSK,
            ORGEH = r.OrgUnit ?? current?.ORGEH,
            PLANS = r.Position ?? current?.PLANS,
            STELL = current?.STELL,
            KOSTL = r.CostCenter ?? current?.KOSTL,
            AEDTM = DateTime.Today, UNAME = r.ChangedBy
        };

        if (current is not null)
        {
            current.ENDDA = r.Begda.AddDays(-1);
            current.AEDTM = DateTime.Today;
            current.UNAME = r.ChangedBy;
        }
        _db.PA0001.Add(newRec);

        await _db.SaveChangesAsync(ct);
        await tx.CommitAsync(ct);
        return true;
    }

    // ---- helpers ------------------------------------------------------------

    private static async Task<T?> ValidOn<T>(DbSet<T> set, int pernr, DateTime key, CancellationToken ct)
        where T : InfotypeBase
        => await set.AsNoTracking()
            .Where(x => x.PERNR == pernr && x.BEGDA <= key && x.ENDDA >= key)
            .OrderByDescending(x => x.BEGDA).FirstOrDefaultAsync(ct);

    private async Task<int> NextNumberAsync(string rangeObject, CancellationToken ct)
    {
        var range = await _db.NumberRanges.FirstOrDefaultAsync(x => x.RangeObject == rangeObject, ct)
                    ?? throw new InvalidOperationException($"Number range '{rangeObject}' is not defined.");
        range.CurrentNumber += 1;
        if (range.CurrentNumber > range.ToNumber)
            throw new InvalidOperationException($"Number range '{rangeObject}' is exhausted.");
        await _db.SaveChangesAsync(ct);
        return (int)range.CurrentNumber;
    }

    private async Task<Dictionary<(string, int), string?>> LoadOrgTextsAsync(DateTime key, CancellationToken ct)
        => await _db.HRP1000.AsNoTracking()
            .Where(o => o.PLVAR == "01" && o.BEGDA <= key && o.ENDDA >= key)
            .ToDictionaryAsync(o => (o.OTYPE, o.OBJID), o => o.STEXT, ct);

    private async Task<Dictionary<string, string?>> LoadDomainDictAsync(string domain, CancellationToken ct)
        => await _db.DomainValues.AsNoTracking()
            .Where(d => d.Domain == domain)
            .ToDictionaryAsync(d => d.ValueKey, d => d.ValueTxt, ct);

    private async Task<string?> ResolveDomainAsync(string domain, string? key, CancellationToken ct)
        => key is null ? null
            : await _db.DomainValues.AsNoTracking()
                .Where(d => d.Domain == domain && d.ValueKey == key)
                .Select(d => d.ValueTxt).FirstOrDefaultAsync(ct);
}
