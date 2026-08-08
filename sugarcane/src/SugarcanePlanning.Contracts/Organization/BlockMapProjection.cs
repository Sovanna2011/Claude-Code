namespace SugarcanePlanning.Contracts.Organization;

/// <summary>One block placed on the plot, in the plot's own coordinate space.</summary>
public readonly record struct BlockMapPoint(int BlockId, double X, double Y);

/// <summary>The result of projecting a set of blocks onto a flat drawing surface.</summary>
public sealed class BlockMapPlot
{
    public required double Width { get; init; }
    public required double Height { get; init; }
    public required IReadOnlyList<BlockMapPoint> Points { get; init; }

    /// <summary>Centre of the plotted extent — what "open the estate in Google Maps" points at.</summary>
    public required double CentreLatitude { get; init; }
    public required double CentreLongitude { get; init; }
}

/// <summary>
/// Turns block latitudes and longitudes into plot coordinates for the fallback map the client
/// draws itself when no Google Maps key is configured.
///
/// It lives in Contracts rather than in the Blazor project so it can be unit-tested without the
/// test project taking a dependency on a WebAssembly application.
///
/// The projection is equirectangular with longitude scaled by the cosine of the centre latitude.
/// Over an estate — tens of kilometres — that is accurate to well under a metre of relative
/// position, and unlike Web Mercator it keeps the arithmetic legible.
/// </summary>
public static class BlockMapProjection
{
    /// <summary>Span assumed when every block sits at the same point, so the plot is never degenerate.</summary>
    private const double MinimumSpanDegrees = 0.01;

    public static BlockMapPlot Project(
        IEnumerable<(int BlockId, double Latitude, double Longitude)> blocks,
        double width = 1000d,
        double padFraction = 0.08d)
    {
        ArgumentNullException.ThrowIfNull(blocks);
        if (width <= 0) throw new ArgumentOutOfRangeException(nameof(width), "The plot must have a positive width.");
        if (padFraction is < 0 or >= 0.5) throw new ArgumentOutOfRangeException(nameof(padFraction),
            "The padding must be a fraction of the extent below one half.");

        var list = blocks.ToList();
        if (list.Count == 0)
        {
            return new BlockMapPlot
            {
                Width = width,
                Height = Math.Round(width * 0.6d, 2),
                Points = Array.Empty<BlockMapPoint>(),
                CentreLatitude = 0,
                CentreLongitude = 0
            };
        }

        var minLat = list.Min(b => b.Latitude);
        var maxLat = list.Max(b => b.Latitude);
        var minLng = list.Min(b => b.Longitude);
        var maxLng = list.Max(b => b.Longitude);

        var centreLat = (minLat + maxLat) / 2d;
        var centreLng = (minLng + maxLng) / 2d;

        // One degree of longitude is shorter than one of latitude everywhere but the equator;
        // without this the estate comes out stretched east to west.
        var kx = Math.Cos(centreLat * Math.PI / 180d);

        var xs = list.Select(b => b.Longitude * kx).ToList();
        var ys = list.Select(b => -b.Latitude).ToList();      // north is up, so latitude is inverted

        var spanX = Math.Max(xs.Max() - xs.Min(), MinimumSpanDegrees);
        var spanY = Math.Max(ys.Max() - ys.Min(), MinimumSpanDegrees);

        var padX = spanX * padFraction;
        var padY = spanY * padFraction;

        // The same padding on both axes would distort the shape; the box is grown symmetrically
        // around the mid-point of each axis so a north-south estate stays north-south.
        var midX = (xs.Max() + xs.Min()) / 2d;
        var midY = (ys.Max() + ys.Min()) / 2d;

        var halfX = spanX / 2d + padX;
        var halfY = spanY / 2d + padY;

        var height = Math.Round(width * (halfY / halfX), 2);
        var points = new List<BlockMapPoint>(list.Count);
        for (var i = 0; i < list.Count; i++)
        {
            var x = (xs[i] - (midX - halfX)) / (2d * halfX) * width;
            var y = (ys[i] - (midY - halfY)) / (2d * halfY) * height;
            points.Add(new BlockMapPoint(list[i].BlockId, Math.Round(x, 2), Math.Round(y, 2)));
        }

        return new BlockMapPlot
        {
            Width = width,
            Height = height,
            Points = points,
            CentreLatitude = centreLat,
            CentreLongitude = centreLng
        };
    }

    /// <summary>
    /// Marker radius for a block, scaled by the square root of its area so that the drawn circle's
    /// area — not its width — is what the eye compares.
    /// </summary>
    public static double Radius(decimal areaHa, decimal maxAreaHa, double minimum = 9d, double range = 13d)
    {
        if (maxAreaHa <= 0 || areaHa <= 0) return minimum;
        var ratio = Math.Min(1d, (double)(areaHa / maxAreaHa));
        return Math.Round(minimum + range * Math.Sqrt(ratio), 2);
    }

    /// <summary>A Google Maps search link for a point — the one thing that works with no API key.</summary>
    public static string GoogleMapsLink(decimal latitude, decimal longitude) =>
        FormattableString.Invariant($"https://www.google.com/maps/search/?api=1&query={latitude},{longitude}");
}
