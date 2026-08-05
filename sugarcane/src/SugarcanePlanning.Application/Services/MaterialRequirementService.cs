using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Options;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Materials;
using SugarcanePlanning.Domain.Calculations;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>
/// Material requirement planning (sections 12 and 13): resolves the most specific standard for
/// every activity plan, applies the waste formula, then nets the requirement against stock.
/// </summary>
public class MaterialRequirementService : ServiceBase, IMaterialRequirementService
{
    private readonly PlanningOptions _options;

    public MaterialRequirementService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock,
        IOptions<PlanningOptions> options) : base(db, user, clock) => _options = options.Value;

    // ------------------------------------------------------------ recalculation

    public async Task<int> RecalculateAsync(int projectionId, CancellationToken ct = default)
    {
        var projection = await RequireAsync(Db.Projections, projectionId, "Planting projection", ct);

        var plans = await Db.ActivityPlans
            .Include(p => p.Activity)
            .Include(p => p.ProjectionLine)
            .Include(p => p.Block)
            .Where(p => p.ProjectionId == projectionId)
            .ToListAsync(ct);

        var planIds = plans.Select(p => p.Id).ToList();
        var existing = await Db.MaterialRequirements.Where(r => planIds.Contains(r.ActivityPlanId)).ToListAsync(ct);
        Db.MaterialRequirements.RemoveRange(existing);

        var standards = await Db.MaterialStandards
            .Include(s => s.Material)
            .AsNoTracking()
            .Where(s => s.IsActive)
            .ToListAsync(ct);

        var varieties = await Db.Varieties.AsNoTracking().ToDictionaryAsync(v => v.Id, ct);
        var created = 0;

        foreach (var plan in plans)
        {
            if (plan.Activity is null || !plan.Activity.RequiresMaterial) continue;
            var line = plan.ProjectionLine;
            if (line is null) continue;

            var matches = ResolveStandards(standards, plan.ActivityId, line.CropType, line.CaneVarietyId,
                plan.Block?.SoilType, plan.PlannedStartDate);

            foreach (var standard in matches)
            {
                // Seed cane is driven by the variety's seed rate rather than a fixed standard.
                var ratePerHa = standard.Material?.Category == MaterialCategory.SeedCane
                                && varieties.TryGetValue(line.CaneVarietyId, out var variety)
                                && variety.SeedRatePerHa > 0
                    ? variety.SeedRatePerHa
                    : standard.StandardRatePerHa;

                var baseQty = PlanningFormulas.BaseMaterialRequirement(plan.PlannedAreaHa, ratePerHa)
                              * standard.NumberOfApplications;
                var waste = PlanningFormulas.WasteQuantity(baseQty, standard.WastePercent);

                Db.MaterialRequirements.Add(new ActivityMaterialRequirement
                {
                    CompanyId = projection.CompanyId,
                    ActivityPlanId = plan.Id,
                    MaterialId = standard.MaterialId,
                    ActivityMaterialStandardId = standard.Id,
                    PlannedAreaHa = plan.PlannedAreaHa,
                    StandardRatePerHa = ratePerHa,
                    NumberOfApplications = standard.NumberOfApplications,
                    WastePercent = standard.WastePercent,
                    BaseRequirement = baseQty,
                    WasteQuantity = waste,
                    TotalRequirement = Math.Round(baseQty + waste, PlanningFormulas.QuantityScale),
                    Unit = standard.Material?.BaseUnit ?? UnitOfMeasure.Kilogram,
                    RequiredDeliveryDate = plan.PlannedStartDate.AddDays(-_options.MaterialDeliveryLeadDays)
                });
                created++;
            }
        }

        await Db.SaveChangesAsync(ct);
        return created;
    }

    /// <summary>
    /// Picks one standard per material: among the rows effective on the date and matching the
    /// crop type / variety / soil type, the most specific one wins (section 12).
    /// </summary>
    internal static List<ActivityMaterialStandard> ResolveStandards(IReadOnlyList<ActivityMaterialStandard> all,
        int activityId, CropType cropType, int varietyId, SoilType? soilType, DateOnly onDate)
        => all
            .Where(s => s.ActivityId == activityId
                        && s.IsEffectiveOn(onDate)
                        && (s.CropType == CropType.Both || s.CropType == cropType)
                        && (s.CaneVarietyId is null || s.CaneVarietyId == varietyId)
                        && (s.SoilType is null || soilType is null || s.SoilType == soilType))
            .GroupBy(s => s.MaterialId)
            .Select(g => g.OrderByDescending(s => s.SpecificityScore())
                          .ThenByDescending(s => s.EffectiveFrom)
                          .First())
            .ToList();

    // ------------------------------------------------------------------ queries

    public async Task<IReadOnlyList<MaterialRequirementDto>> GetRequirementsAsync(MaterialRequirementQuery query,
        CancellationToken ct = default)
    {
        var rows = Db.MaterialRequirements
            .Include(r => r.Material)
            .Include(r => r.ActivityPlan!).ThenInclude(p => p.Activity)
            .Include(r => r.ActivityPlan!).ThenInclude(p => p.ProjectionLine)
            .Include(r => r.ActivityPlan!).ThenInclude(p => p.Block!).ThenInclude(b => b.Zone!).ThenInclude(z => z.Farm!).ThenInclude(f => f.Estate)
            .AsNoTracking()
            .AsQueryable();

        if (query.ProjectionId is not null) rows = rows.Where(r => r.ActivityPlan!.ProjectionId == query.ProjectionId);
        if (query.SeasonId is not null) rows = rows.Where(r => r.ActivityPlan!.Projection!.GrowingSeasonId == query.SeasonId);
        if (query.EstateId is not null) rows = rows.Where(r => r.ActivityPlan!.Block!.Zone!.Farm!.EstateId == query.EstateId);
        if (query.FarmId is not null) rows = rows.Where(r => r.ActivityPlan!.FarmId == query.FarmId);
        if (query.ZoneId is not null) rows = rows.Where(r => r.ActivityPlan!.ZoneId == query.ZoneId);
        if (query.BlockId is not null) rows = rows.Where(r => r.ActivityPlan!.BlockId == query.BlockId);
        if (query.ActivityId is not null) rows = rows.Where(r => r.ActivityPlan!.ActivityId == query.ActivityId);
        if (query.MaterialId is not null) rows = rows.Where(r => r.MaterialId == query.MaterialId);
        if (query.CaneVarietyId is not null) rows = rows.Where(r => r.ActivityPlan!.ProjectionLine!.CaneVarietyId == query.CaneVarietyId);
        if (query.Category is not null) rows = rows.Where(r => r.Material!.Category == query.Category);
        if (query.FromDate is not null) rows = rows.Where(r => r.ActivityPlan!.PlannedStartDate >= query.FromDate);
        if (query.ToDate is not null) rows = rows.Where(r => r.ActivityPlan!.PlannedStartDate <= query.ToDate);

        var data = await rows.ToListAsync(ct);

        var stocks = await Db.MaterialStocks.AsNoTracking()
            .Where(s => query.EstateId == null || s.EstateId == null || s.EstateId == query.EstateId)
            .ToListAsync(ct);

        var grouped = (query.GroupBy ?? "material").ToLowerInvariant() switch
        {
            "activity" => data.GroupBy(r => new GroupKey(r.MaterialId, r.ActivityPlan!.ActivityId, null, null, null, null, null)),
            "block" => data.GroupBy(r => new GroupKey(r.MaterialId, null, r.ActivityPlan!.BlockId, null, null, null, null)),
            "farm" => data.GroupBy(r => new GroupKey(r.MaterialId, null, null, r.ActivityPlan!.FarmId, null, null, null)),
            "month" => data.GroupBy(r => new GroupKey(r.MaterialId, null, null, null,
                r.ActivityPlan!.PlannedStartDate.Year, r.ActivityPlan.PlannedStartDate.Month, null)),
            "variety" => data.GroupBy(r => new GroupKey(r.MaterialId, null, null, null, null, null,
                r.ActivityPlan!.ProjectionLine!.CaneVarietyId)),
            _ => data.GroupBy(r => new GroupKey(r.MaterialId, null, null, null, null, null, null))
        };

        var varieties = await Db.Varieties.AsNoTracking().ToDictionaryAsync(v => v.Id, v => v.Name, ct);
        var result = new List<MaterialRequirementDto>();

        foreach (var group in grouped)
        {
            var first = group.First();
            var material = first.Material!;
            var total = Math.Round(group.Sum(r => r.TotalRequirement), PlanningFormulas.QuantityScale);

            var stockRows = stocks.Where(s => s.MaterialId == group.Key.MaterialId).ToList();
            var available = stockRows.Sum(s => s.AvailableStock);
            var reserved = stockRows.Sum(s => s.ReservedQuantity);
            var incoming = stockRows.Sum(s => s.IncomingQuantity);
            var net = PlanningFormulas.NetAvailableQuantity(available, incoming, reserved);

            var dto = new MaterialRequirementDto
            {
                MaterialId = material.Id,
                MaterialCode = material.Code,
                MaterialName = material.Name,
                Category = material.Category,
                Unit = material.BaseUnit,
                PlannedAreaHa = Math.Round(group.Sum(r => r.PlannedAreaHa), PlanningFormulas.QuantityScale),
                BaseRequirement = Math.Round(group.Sum(r => r.BaseRequirement), PlanningFormulas.QuantityScale),
                WasteQuantity = Math.Round(group.Sum(r => r.WasteQuantity), PlanningFormulas.QuantityScale),
                TotalRequirement = total,
                AvailableStock = available,
                ReservedQuantity = reserved,
                IncomingQuantity = incoming,
                NetAvailableQuantity = net,
                ShortageQuantity = PlanningFormulas.ShortageQuantity(total, net),
                SurplusQuantity = PlanningFormulas.SurplusQuantity(total, net),
                RequiredDeliveryDate = group.Min(r => r.RequiredDeliveryDate),
                Status = PlanningFormulas.EvaluateCapacity(total, net, _options.AtRiskThreshold)
            };

            if (group.Key.ActivityId is not null)
            {
                dto.ActivityId = group.Key.ActivityId;
                dto.ActivityName = first.ActivityPlan?.Activity?.Name;
            }
            if (group.Key.BlockId is not null)
            {
                dto.BlockId = group.Key.BlockId;
                dto.BlockName = first.ActivityPlan?.Block is null
                    ? null : $"{first.ActivityPlan.Block.Code} — {first.ActivityPlan.Block.Name}";
                dto.ZoneId = first.ActivityPlan?.ZoneId;
                dto.ZoneName = first.ActivityPlan?.Block?.Zone?.Name;
            }
            if (group.Key.FarmId is not null)
            {
                dto.FarmId = group.Key.FarmId;
                dto.FarmName = first.ActivityPlan?.Block?.Zone?.Farm?.Name;
            }
            if (group.Key.Year is not null)
            {
                dto.PlanningYear = group.Key.Year;
                dto.PlanningMonth = group.Key.Month;
            }
            if (group.Key.VarietyId is not null)
            {
                dto.CaneVarietyId = group.Key.VarietyId;
                dto.VarietyName = varieties.TryGetValue(group.Key.VarietyId.Value, out var name) ? name : null;
            }
            dto.EstateId = first.ActivityPlan?.Block?.Zone?.Farm?.EstateId;
            dto.EstateName = first.ActivityPlan?.Block?.Zone?.Farm?.Estate?.Name;

            result.Add(dto);
        }

        if (query.ShortagesOnly) result = result.Where(r => r.ShortageQuantity > 0).ToList();

        return result
            .OrderBy(r => r.Category)
            .ThenBy(r => r.MaterialCode)
            .ThenBy(r => r.PlanningYear).ThenBy(r => r.PlanningMonth)
            .ToList();
    }

    /// <summary>Composite grouping key; unused members stay null for the selected grouping.</summary>
    private readonly record struct GroupKey(int MaterialId, int? ActivityId, int? BlockId, int? FarmId,
        int? Year, int? Month, int? VarietyId);
}
