using Microsoft.EntityFrameworkCore;
using PoApproval.Api.Data;
using PoApproval.Api.DTOs;
using PoApproval.Api.Services.Interfaces;

namespace PoApproval.Api.Services;

/// <summary>Reference-data value helps (F4) for the UI5 front end.</summary>
public class ValueHelpService : IValueHelpService
{
    private readonly PoDbContext _db;

    public ValueHelpService(PoDbContext db) => _db = db;

    public async Task<IReadOnlyList<ValueHelpItemDto>> GetAsync(string name, CancellationToken ct = default)
        => name.ToLowerInvariant() switch
        {
            "purchasing-groups" => await _db.T024.AsNoTracking()
                .Select(x => new ValueHelpItemDto { Key = x.EKGRP, Text = x.EKNAM })
                .OrderBy(x => x.Key).ToListAsync(ct),

            "purchasing-orgs" => await _db.T024E.AsNoTracking()
                .Select(x => new ValueHelpItemDto { Key = x.EKORG, Text = x.EKOTX })
                .OrderBy(x => x.Key).ToListAsync(ct),

            "doc-types" => await _db.T161.AsNoTracking()
                .Where(x => x.BSTYP == "F")
                .Select(x => new ValueHelpItemDto { Key = x.BSART, Text = x.BATXT })
                .OrderBy(x => x.Key).ToListAsync(ct),

            "release-groups" => await _db.T16FG.AsNoTracking()
                .Select(x => new ValueHelpItemDto { Key = x.FRGGR, Text = x.FRGGT })
                .OrderBy(x => x.Key).ToListAsync(ct),

            "release-strategies" => await _db.T16FS.AsNoTracking()
                .Select(x => new ValueHelpItemDto { Key = x.FRGSX, Text = x.FRGST })
                .OrderBy(x => x.Key).ToListAsync(ct),

            "release-indicators" => await _db.DomainValues.AsNoTracking()
                .Where(x => x.Domain == "FRGKE")
                .Select(x => new ValueHelpItemDto { Key = x.ValueKey, Text = x.ValueTxt })
                .OrderBy(x => x.Key).ToListAsync(ct),

            "vendors" => await _db.Lfa1.AsNoTracking()
                .Select(x => new ValueHelpItemDto { Key = x.LIFNR, Text = x.NAME1 })
                .OrderBy(x => x.Key).ToListAsync(ct),

            _ => throw new ArgumentException($"Unknown value help '{name}'.")
        };
}
