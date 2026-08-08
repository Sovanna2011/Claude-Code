using SugarcanePlanning.Contracts.Organization;

namespace SugarcanePlanning.UnitTests;

/// <summary>
/// The geometry behind the block-location screen's built-in plot — the one it draws when no
/// Google Maps key is configured.
/// </summary>
public class BlockMapProjectionTests
{
    private static readonly (int, double, double)[] Estate =
    {
        (1, -15.474, 28.232),      // north-west corner
        (2, -15.474, 28.243),
        (3, -15.485, 28.232),
        (4, -15.485, 28.243)       // south-east corner
    };

    [Fact]
    public void Places_every_block_inside_the_plot()
    {
        var plot = BlockMapProjection.Project(Estate);

        Assert.Equal(4, plot.Points.Count);
        Assert.All(plot.Points, p =>
        {
            Assert.InRange(p.X, 0, plot.Width);
            Assert.InRange(p.Y, 0, plot.Height);
        });
    }

    [Fact]
    public void Puts_north_at_the_top_and_east_to_the_right()
    {
        var plot = BlockMapProjection.Project(Estate).Points.ToDictionary(p => p.BlockId);

        // Block 1 is north of block 3, so it must be drawn above it: a smaller y.
        Assert.True(plot[1].Y < plot[3].Y, "the northern block should be higher up the plot");
        // Block 2 is east of block 1, so it must be drawn to its right.
        Assert.True(plot[2].X > plot[1].X, "the eastern block should be further right");
    }

    [Fact]
    public void Keeps_the_shape_of_the_estate()
    {
        // 0.011 degrees of latitude is about 1.22 km; the same in longitude at 15.5 degrees south
        // is about 1.18 km — so a square of degrees is very slightly wider than it is tall, and
        // the drawn plot has to carry that through rather than come out square.
        var plot = BlockMapProjection.Project(Estate);
        var points = plot.Points.ToDictionary(p => p.BlockId);

        var drawnWidth = points[2].X - points[1].X;
        var drawnHeight = points[3].Y - points[1].Y;

        var expected = 0.011 * Math.Cos(-15.4795 * Math.PI / 180) / 0.011;
        Assert.Equal(expected, drawnWidth / drawnHeight, precision: 2);
    }

    [Fact]
    public void Pads_the_extent_so_markers_never_touch_the_edge()
    {
        var plot = BlockMapProjection.Project(Estate, width: 1000, padFraction: 0.1);
        var xs = plot.Points.Select(p => p.X).ToList();

        // With a tenth of the span added on each side, the extreme blocks sit one twelfth of the
        // width in from each edge: 0.1 / (1 + 0.1 + 0.1).
        Assert.Equal(1000d / 12d, xs.Min(), precision: 1);
        Assert.Equal(1000d - 1000d / 12d, xs.Max(), precision: 1);
    }

    [Fact]
    public void Reports_the_centre_of_the_extent()
    {
        var plot = BlockMapProjection.Project(Estate);

        Assert.Equal(-15.4795, plot.CentreLatitude, precision: 4);
        Assert.Equal(28.2375, plot.CentreLongitude, precision: 4);
    }

    [Fact]
    public void Survives_a_single_block()
    {
        var plot = BlockMapProjection.Project(new[] { (7, -15.4, 28.3) });

        var point = Assert.Single(plot.Points);
        Assert.Equal(7, point.BlockId);
        // Nothing to scale against, so the block belongs in the middle rather than at 0/0 or NaN.
        Assert.Equal(plot.Width / 2, point.X, precision: 1);
        Assert.Equal(plot.Height / 2, point.Y, precision: 1);
    }

    [Fact]
    public void Survives_blocks_that_share_a_point()
    {
        var plot = BlockMapProjection.Project(new[] { (1, -15.4, 28.3), (2, -15.4, 28.3) });

        Assert.All(plot.Points, p =>
        {
            Assert.False(double.IsNaN(p.X));
            Assert.False(double.IsNaN(p.Y));
        });
        Assert.True(plot.Height > 0);
    }

    [Fact]
    public void Returns_an_empty_plot_for_no_blocks()
    {
        var plot = BlockMapProjection.Project(Array.Empty<(int, double, double)>());

        Assert.Empty(plot.Points);
        Assert.True(plot.Width > 0 && plot.Height > 0);
    }

    [Theory]
    [InlineData(0)]
    [InlineData(-1)]
    public void Refuses_a_plot_with_no_width(double width) =>
        Assert.Throws<ArgumentOutOfRangeException>(() => BlockMapProjection.Project(Estate, width));

    [Fact]
    public void Refuses_padding_that_would_swallow_the_extent() =>
        Assert.Throws<ArgumentOutOfRangeException>(() => BlockMapProjection.Project(Estate, 1000, 0.5));

    [Fact]
    public void Scales_the_marker_by_the_square_root_of_area()
    {
        // Four times the area is twice the radius, so the drawn circle's area — what the eye
        // actually compares — is proportional to the hectares.
        var small = BlockMapProjection.Radius(50m, 200m, minimum: 0d, range: 20d);
        var large = BlockMapProjection.Radius(200m, 200m, minimum: 0d, range: 20d);

        Assert.Equal(10d, small, precision: 2);
        Assert.Equal(20d, large, precision: 2);
    }

    [Theory]
    [InlineData(0, 100)]
    [InlineData(100, 0)]
    [InlineData(-5, 100)]
    public void Falls_back_to_the_minimum_radius_without_a_usable_area(decimal area, decimal max) =>
        Assert.Equal(9d, BlockMapProjection.Radius(area, max));

    [Fact]
    public void Builds_a_google_maps_link_that_does_not_depend_on_the_locale()
    {
        var link = BlockMapProjection.GoogleMapsLink(-15.47432m, 28.23328m);

        Assert.Equal("https://www.google.com/maps/search/?api=1&query=-15.47432,28.23328", link);
    }
}
