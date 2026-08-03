using HRModule.Api.Data;
using HRModule.Api.DTOs;
using HRModule.Api.Services.Interfaces;
using Microsoft.EntityFrameworkCore;

namespace HRModule.Api.Services;

/// <summary>Supplies value help (F4) lists sourced from the customizing tables.</summary>
public class ValueHelpService : IValueHelpService
{
    private readonly HRDbContext _db;
    public ValueHelpService(HRDbContext db) => _db = db;

    public async Task<IReadOnlyList<ValueHelpDto>> GetCompanyCodesAsync(CancellationToken ct = default)
        => await _db.T001.AsNoTracking()
            .Select(x => new ValueHelpDto { Key = x.BUKRS, Text = x.BUTXT })
            .OrderBy(x => x.Key).ToListAsync(ct);

    public async Task<IReadOnlyList<ValueHelpDto>> GetPersonnelAreasAsync(CancellationToken ct = default)
        => await _db.T500P.AsNoTracking()
            .Select(x => new ValueHelpDto { Key = x.WERKS, Text = x.NAME1 })
            .OrderBy(x => x.Key).ToListAsync(ct);

    public async Task<IReadOnlyList<ValueHelpDto>> GetEmployeeGroupsAsync(CancellationToken ct = default)
        => await _db.T501.AsNoTracking()
            .Select(x => new ValueHelpDto { Key = x.PERSG, Text = x.PTEXT })
            .OrderBy(x => x.Key).ToListAsync(ct);

    public async Task<IReadOnlyList<ValueHelpDto>> GetEmployeeSubgroupsAsync(CancellationToken ct = default)
        => await _db.T503K.AsNoTracking()
            .Select(x => new ValueHelpDto { Key = x.PERSK, Text = x.PTEXT })
            .OrderBy(x => x.Key).ToListAsync(ct);

    public async Task<IReadOnlyList<ValueHelpDto>> GetAbsenceTypesAsync(CancellationToken ct = default)
        => await _db.T554S.AsNoTracking()
            .Select(x => new ValueHelpDto { Key = x.AWART, Text = x.ATEXT })
            .OrderBy(x => x.Key).ToListAsync(ct);

    public async Task<IReadOnlyList<ValueHelpDto>> GetDomainAsync(string domain, CancellationToken ct = default)
        => await _db.DomainValues.AsNoTracking()
            .Where(x => x.Domain == domain)
            .Select(x => new ValueHelpDto { Key = x.ValueKey, Text = x.ValueTxt })
            .OrderBy(x => x.Key).ToListAsync(ct);
}
