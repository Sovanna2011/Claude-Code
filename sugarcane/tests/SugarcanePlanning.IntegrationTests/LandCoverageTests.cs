using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Contracts.Organization;
using SugarcanePlanning.Domain.Entities;
using SugarcanePlanning.Domain.Enums;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// The land-coverage tree the dashboard reports from: farm → zone → block, with total,
/// plantable, new-planting, ratoon, standing-cane and unplantable area at every level.
/// </summary>
public class LandCoverageTests
{
    /// <summary>The test host seeds one farm with one zone: block A 120/100 ha, block B 60/50 ha.</summary>
    private const decimal TotalA = 120m, PlantableA = 100m, TotalB = 60m, PlantableB = 50m;

    [Fact]
    public async Task Returns_a_farm_zone_block_tree()
    {
        using var host = new PlanningTestHost();

        var tree = await host.Land.GetLandCoverageAsync(null, null);

        var farm = Assert.Single(tree);
        Assert.Equal("Farm", farm.NodeType);
        var zone = Assert.Single(farm.Children);
        Assert.Equal("Zone", zone.NodeType);
        Assert.Equal(2, zone.Children.Count);
        Assert.All(zone.Children, b => Assert.Equal("Block", b.NodeType));
    }

    [Fact]
    public async Task Rolls_every_area_up_from_the_blocks()
    {
        using var host = new PlanningTestHost();

        var farm = (await host.Land.GetLandCoverageAsync(null, null)).Single();
        var zone = farm.Children.Single();

        // The farm's own TotalAreaHa column says 500 ha, but the tree reports what the blocks
        // beneath it actually add up to — a header total is derived, never entered.
        Assert.Equal(TotalA + TotalB, farm.TotalAreaHa);
        Assert.Equal(PlantableA + PlantableB, farm.PlantableAreaHa);
        Assert.Equal(farm.TotalAreaHa, zone.TotalAreaHa);
        Assert.Equal(farm.PlantableAreaHa, zone.PlantableAreaHa);
    }

    [Fact]
    public async Task Not_plantable_is_total_less_plantable()
    {
        using var host = new PlanningTestHost();

        var farm = (await host.Land.GetLandCoverageAsync(null, null)).Single();
        var blocks = farm.Children.Single().Children;

        Assert.Equal(TotalA - PlantableA, blocks.Single(b => b.Id == host.BlockAId).NotPlantableAreaHa);
        Assert.Equal(TotalB - PlantableB, blocks.Single(b => b.Id == host.BlockBId).NotPlantableAreaHa);
        Assert.Equal(farm.TotalAreaHa - farm.PlantableAreaHa, farm.NotPlantableAreaHa);
    }

    [Fact]
    public async Task Counts_only_approved_projection_lines()
    {
        using var host = new PlanningTestHost();

        // A draft on block A. Nothing is committed yet, so nothing should be reported.
        var draft = await host.Projections.CreateAsync(host.NewProjection(80m), default);
        var before = (await host.Land.GetLandCoverageAsync(null, null)).Single();
        Assert.Equal(0m, before.NewPlantingAreaHa);

        await Approve(host, draft.Id);

        var after = (await host.Land.GetLandCoverageAsync(null, null)).Single();
        Assert.Equal(80m, after.NewPlantingAreaHa);
        Assert.Equal(80m, after.Children.Single().Children.Single(b => b.Id == host.BlockAId).NewPlantingAreaHa);
    }

    [Fact]
    public async Task Separates_new_planting_from_ratoon()
    {
        using var host = new PlanningTestHost();

        var dto = host.NewProjection(80m);
        dto.Lines.Add(new Contracts.Projections.ProjectionLineUpsertDto
        {
            BlockId = host.BlockBId,
            CaneVarietyId = host.VarietyId,
            CropType = CropType.Ratoon,
            ProjectedPlantingAreaHa = 40m,
            PlannedPlantingStart = new DateOnly(2026, 3, 2),
            PlannedPlantingEnd = new DateOnly(2026, 3, 20),
            ExpectedYieldPerHa = 85m,
            ExpectedLossPercent = 5m
        });

        var projection = await host.Projections.CreateAsync(dto, default);
        await Approve(host, projection.Id);

        var farm = (await host.Land.GetLandCoverageAsync(null, null)).Single();
        Assert.Equal(80m, farm.NewPlantingAreaHa);
        Assert.Equal(40m, farm.RatoonAreaHa);

        var blocks = farm.Children.Single().Children;
        Assert.Equal(40m, blocks.Single(b => b.Id == host.BlockBId).RatoonAreaHa);
        Assert.Equal(0m, blocks.Single(b => b.Id == host.BlockBId).NewPlantingAreaHa);
    }

    [Theory]
    [InlineData(CropStatus.Planted, true)]
    [InlineData(CropStatus.Growing, true)]
    [InlineData(CropStatus.ReadyForHarvest, true)]
    [InlineData(CropStatus.Fallow, false)]
    [InlineData(CropStatus.Prepared, false)]
    [InlineData(CropStatus.Harvested, false)]
    public async Task Area_under_cane_follows_the_block_crop_status(CropStatus status, bool counted)
    {
        using var host = new PlanningTestHost();
        var block = await host.Db.Blocks.FirstAsync(b => b.Id == host.BlockAId);
        block.CurrentCropStatus = status;
        await host.Db.SaveChangesAsync();

        var farm = (await host.Land.GetLandCoverageAsync(null, null)).Single();

        Assert.Equal(counted ? PlantableA : 0m, farm.AreaUnderCaneHa);
    }

    [Fact]
    public async Task Gives_every_row_a_map_link_when_the_blocks_have_coordinates()
    {
        using var host = new PlanningTestHost();
        var blocks = await host.Db.Blocks.OrderBy(b => b.Id).ToListAsync();
        blocks[0].Latitude = -15.40m; blocks[0].Longitude = 28.30m;
        blocks[1].Latitude = -15.50m; blocks[1].Longitude = 28.40m;
        await host.Db.SaveChangesAsync();

        var farm = (await host.Land.GetLandCoverageAsync(null, null)).Single();

        // A farm row points at the centre of its blocks, so the link works at every level.
        Assert.Equal(-15.45m, farm.Latitude);
        Assert.Equal(28.35m, farm.Longitude);
        Assert.Equal("https://www.google.com/maps/search/?api=1&query=-15.45,28.35", farm.MapUrl);
        Assert.All(farm.Children.Single().Children, b => Assert.NotNull(b.MapUrl));
    }

    [Fact]
    public async Task Leaves_the_map_link_empty_when_no_block_has_coordinates()
    {
        using var host = new PlanningTestHost();

        var farm = (await host.Land.GetLandCoverageAsync(null, null)).Single();

        Assert.Null(farm.MapUrl);
        Assert.All(farm.Children.Single().Children, b => Assert.Null(b.MapUrl));
    }

    [Fact]
    public async Task Filters_by_farm_and_by_season()
    {
        using var host = new PlanningTestHost();
        var projection = await host.Projections.CreateAsync(host.NewProjection(80m), default);
        await Approve(host, projection.Id);

        Assert.Empty(await host.Land.GetLandCoverageAsync(null, farmId: host.FarmId + 999));
        Assert.Single(await host.Land.GetLandCoverageAsync(null, farmId: host.FarmId));

        // A season the projection does not belong to reports the land but none of its planting.
        var other = (await host.Land.GetLandCoverageAsync(host.SeasonId + 999, null)).Single();
        Assert.Equal(TotalA + TotalB, other.TotalAreaHa);
        Assert.Equal(0m, other.NewPlantingAreaHa);

        var own = (await host.Land.GetLandCoverageAsync(host.SeasonId, null)).Single();
        Assert.Equal(80m, own.NewPlantingAreaHa);
    }

    private static async Task Approve(PlanningTestHost host, int projectionId)
    {
        foreach (var action in new[] { ApprovalAction.Submit, ApprovalAction.Review, ApprovalAction.Approve })
        {
            await host.Projections.ExecuteWorkflowAsync(projectionId,
                new Contracts.Projections.WorkflowActionDto { Action = action }, default);
        }

        Assert.Equal(ProjectionStatus.Approved,
            (await host.Db.Projections.AsNoTracking().FirstAsync(p => p.Id == projectionId)).Status);
    }
}
