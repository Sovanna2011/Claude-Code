using System.Text.Json;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Application.Common;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Mapping;
using SugarcanePlanning.Contracts.Capacity;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;

namespace SugarcanePlanning.Application.Services;

/// <summary>
/// What-if scenarios (section 15). A simulation reruns the capacity engine with the scenario's
/// levers applied in memory; the approved plan is never modified.
/// </summary>
public class ScenarioService : ServiceBase, IScenarioService
{
    private readonly CapacityService _capacity;

    public ScenarioService(IAppDbContext db, ICurrentUser user, IDateTimeProvider clock, ICapacityService capacity)
        : base(db, user, clock)
        => _capacity = capacity as CapacityService
            ?? throw new InvalidOperationException("Scenario simulation requires the built-in capacity service.");

    public async Task<IReadOnlyList<ScenarioDto>> GetScenariosAsync(int projectionId, CancellationToken ct = default)
    {
        var rows = await Db.Scenarios
            .Include(s => s.Projection)
            .Include(s => s.Adjustments).ThenInclude(a => a.Activity)
            .AsNoTracking()
            .Where(s => s.ProjectionId == projectionId)
            .OrderBy(s => s.Name)
            .ToListAsync(ct);
        return rows.Select(s => s.ToDto()).ToList();
    }

    public async Task<ScenarioDto> CreateAsync(ScenarioUpsertDto dto, CancellationToken ct = default)
    {
        var projection = await RequireAsync(Db.Projections, dto.ProjectionId, "Planting projection", ct);

        var entity = new PlanningScenario
        {
            CompanyId = projection.CompanyId,
            ProjectionId = projection.Id,
            Name = dto.Name.Trim(),
            Description = dto.Description,
            Status = ScenarioStatus.Draft
        };
        foreach (var adj in dto.Adjustments)
            entity.Adjustments.Add(new ScenarioAdjustment
            {
                CompanyId = projection.CompanyId,
                AdjustmentType = adj.AdjustmentType,
                Value = adj.Value,
                ActivityId = adj.ActivityId,
                Remarks = adj.Remarks
            });

        Db.Scenarios.Add(entity);
        await Db.SaveChangesAsync(ct);
        return await LoadDtoAsync(entity.Id, ct);
    }

    public async Task<ScenarioDto> UpdateAsync(int id, ScenarioUpsertDto dto, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Scenarios.Include(s => s.Adjustments), id, "Scenario", ct);
        if (entity.Status == ScenarioStatus.Adopted)
            throw new BusinessRuleException("ADOPTED", "An adopted scenario can no longer be changed.");

        ApplyConcurrencyToken(entity, dto.RowVersion);
        entity.Name = dto.Name.Trim();
        entity.Description = dto.Description;

        Db.ScenarioAdjustments.RemoveRange(entity.Adjustments);
        entity.Adjustments.Clear();
        foreach (var adj in dto.Adjustments)
            entity.Adjustments.Add(new ScenarioAdjustment
            {
                CompanyId = entity.CompanyId,
                ScenarioId = entity.Id,
                AdjustmentType = adj.AdjustmentType,
                Value = adj.Value,
                ActivityId = adj.ActivityId,
                Remarks = adj.Remarks
            });

        entity.Status = ScenarioStatus.Draft;
        entity.ResultJson = null;
        await Db.SaveChangesAsync(ct);
        return await LoadDtoAsync(id, ct);
    }

    public async Task DeleteAsync(int id, CancellationToken ct = default)
    {
        var entity = await RequireAsync(Db.Scenarios, id, "Scenario", ct);
        SoftDelete(entity);
        entity.Status = ScenarioStatus.Discarded;
        await Db.SaveChangesAsync(ct);
    }

    // -------------------------------------------------------------- simulation

    public async Task<ScenarioResultDto> SimulateAsync(int id, CancellationToken ct = default)
    {
        var scenario = await RequireAsync(
            Db.Scenarios.Include(s => s.Adjustments).Include(s => s.Projection), id, "Scenario", ct);

        var levers = BuildLevers(scenario.Adjustments, await EquipmentCategoriesAsync(scenario.ProjectionId, ct));

        var baseline = await _capacity.AnalyzeAsync(scenario.ProjectionId, ScenarioLevers.None, ct);
        var simulated = await _capacity.AnalyzeAsync(scenario.ProjectionId, levers, ct);

        var result = new ScenarioResultDto
        {
            ScenarioId = scenario.Id,
            ScenarioName = scenario.Name,
            Baseline = baseline,
            Simulated = simulated
        };

        foreach (var simLine in simulated.Lines)
        {
            var baseLine = baseline.Lines.FirstOrDefault(l =>
                l.ResourceType == simLine.ResourceType && l.ResourceName == simLine.ResourceName);
            result.Deltas.Add(new ScenarioDeltaDto
            {
                ResourceType = simLine.ResourceType,
                ResourceName = simLine.ResourceName,
                BaselineGap = baseLine?.Gap ?? 0m,
                SimulatedGap = simLine.Gap,
                BaselineStatus = baseLine?.Status ?? CapacityStatus.Sufficient,
                SimulatedStatus = simLine.Status
            });
        }

        scenario.Status = ScenarioStatus.Simulated;
        scenario.SimulatedAtUtc = Clock.UtcNow;
        scenario.SimulatedBy = User.UserName;
        scenario.ResultJson = JsonSerializer.Serialize(result.Deltas);
        await Db.SaveChangesAsync(ct);

        return result;
    }

    /// <summary>Equipment categories actually used by the projection, so "add equipment" has a target.</summary>
    private async Task<IReadOnlyList<EquipmentCategory>> EquipmentCategoriesAsync(int projectionId, CancellationToken ct)
        => await Db.ActivityPlans.AsNoTracking()
            .Where(p => p.ProjectionId == projectionId && p.RequiredEquipmentCategory != null)
            .Select(p => p.RequiredEquipmentCategory!.Value)
            .Distinct()
            .ToListAsync(ct);

    /// <summary>Folds the stored adjustments into the lever object consumed by the capacity engine.</summary>
    internal static ScenarioLevers BuildLevers(IEnumerable<ScenarioAdjustment> adjustments,
        IReadOnlyList<EquipmentCategory> categories)
    {
        var levers = new ScenarioLevers();
        foreach (var adj in adjustments)
        {
            switch (adj.AdjustmentType)
            {
                case ScenarioAdjustmentType.AdditionalRentalTractors:
                    levers.AdditionalTractors += adj.Value;
                    break;
                case ScenarioAdjustmentType.AdditionalEquipment:
                    // Spread the extra units across the categories the plan actually needs.
                    foreach (var category in categories)
                        levers.AdditionalEquipmentByCategory[category] =
                            levers.AdditionalEquipmentByCategory.GetValueOrDefault(category) + adj.Value;
                    break;
                case ScenarioAdjustmentType.ExtendedWorkingHours:
                    levers.ExtraHoursPerDay += adj.Value;
                    break;
                case ScenarioAdjustmentType.ReducedPlantingArea:
                    levers.AreaReductionPercent += adj.Value;
                    break;
                case ScenarioAdjustmentType.ChangedPlantingDates:
                    levers.DateShiftDays += (int)adj.Value;
                    break;
                case ScenarioAdjustmentType.ChangedActivityDuration:
                    levers.DurationChangeDays += (int)adj.Value;
                    break;
                case ScenarioAdjustmentType.AdditionalWorkers:
                    levers.AdditionalWorkers += adj.Value;
                    break;
            }
        }
        return levers;
    }

    private async Task<ScenarioDto> LoadDtoAsync(int id, CancellationToken ct)
        => (await Db.Scenarios
            .Include(s => s.Projection)
            .Include(s => s.Adjustments).ThenInclude(a => a.Activity)
            .AsNoTracking().FirstAsync(s => s.Id == id, ct)).ToDto();
}
