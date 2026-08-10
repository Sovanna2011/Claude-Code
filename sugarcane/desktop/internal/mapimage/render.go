// Package mapimage draws the estate map into an ordinary image.
//
// It is deliberately not a Windows control. The window hands it a size and the service's features
// and blits the result; everything about how the map looks lives here, in code that runs anywhere.
// That is what makes the map testable: the same function that fills the window on Windows renders
// a PNG on the build machine, and the picture can be checked rather than assumed.
package mapimage

import (
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/api"
	"github.com/sovanna2011/sugarcane-go/desktop/internal/geo"
)

// Palette is the colours the map draws with, so the window can hand in the system's own.
type Palette struct {
	Background color.RGBA
	Outline    color.RGBA
	Selected   color.RGBA
	Label      color.RGBA
	Edge       color.RGBA
}

// DefaultPalette matches the browser client's Fiori Horizon colours closely enough that the two
// applications are recognisably the same system.
func DefaultPalette() Palette {
	return Palette{
		Background: color.RGBA{0xF5, 0xF6, 0xF7, 0xFF},
		Outline:    color.RGBA{0x83, 0x96, 0xA8, 0xFF},
		Selected:   color.RGBA{0x00, 0x64, 0xD9, 0xFF},
		Label:      color.RGBA{0x40, 0x50, 0x60, 0xFF},
		Edge:       color.RGBA{0xFF, 0xFF, 0xFF, 0xFF},
	}
}

// statusColours are a block's fill: what is growing on it, which is the question asked of a block.
var statusColours = map[string]color.RGBA{
	"Growing":         {0x25, 0x6F, 0x3A, 0xFF},
	"Planted":         {0x25, 0x6F, 0x3A, 0xFF},
	"ReadyForHarvest": {0xE7, 0x65, 0x00, 0xFF},
	"Harvested":       {0x78, 0x8F, 0xA6, 0xFF},
	"Prepared":        {0x00, 0x70, 0xF2, 0xFF},
	"Fallow":          {0x78, 0x8F, 0xA6, 0xFF},
}

var neutral = color.RGBA{0x78, 0x8F, 0xA6, 0xFF}

// Band is one entry of the planted-share ramp used at farm and zone level. A whole farm has no
// single status — it is part planted, and how much is the point. Four bands rather than a
// continuous scale: four fills can be told apart, 55% from 62% cannot.
type Band struct {
	UpTo  float64
	Label string
	Fill  color.RGBA
}

// Bands is the ramp, lightest first.
var Bands = []Band{
	{1, "Nothing planted", color.RGBA{0xC4, 0xCF, 0xDA, 0xFF}},
	{40, "Under 40% planted", color.RGBA{0xB6, 0xD7, 0xBE, 0xFF}},
	{80, "40 to 80% planted", color.RGBA{0x6D, 0xAE, 0x82, 0xFF}},
	{math.Inf(1), "Over 80% planted", color.RGBA{0x25, 0x6F, 0x3A, 0xFF}},
}

// BandFor is the band a planted share falls in.
func BandFor(percent float64) Band {
	for _, b := range Bands {
		if percent < b.UpTo {
			return b
		}
	}
	return Bands[len(Bands)-1]
}

// Legend is what the window lists beneath the map for the level being drawn.
func Legend(level string) []Band {
	if level != api.LevelBlock {
		return Bands
	}
	return []Band{
		{0, "Growing", statusColours["Growing"]},
		{0, "Prepared", statusColours["Prepared"]},
		{0, "Ready for harvest", statusColours["ReadyForHarvest"]},
		{0, "Fallow", statusColours["Fallow"]},
	}
}

// Shape is one drawn location, kept after rendering so a click can be answered without projecting
// everything again.
type Shape struct {
	Properties api.FeatureProperties
	Rings      []geo.Ring
}

// Rendered is the picture and what is in it.
type Rendered struct {
	Image      *image.RGBA
	Shapes     []Shape
	Projection *geo.Projection
	Level      string
	// Empty is true when the filter admitted no location with a boundary, so the window can say so
	// rather than showing a blank rectangle and leaving the reader to guess.
	Empty bool
}

// At returns the location drawn under a pixel, topmost first, and whether there was one.
func (r *Rendered) At(x, y int) (api.FeatureProperties, bool) {
	if r.Projection == nil {
		return api.FeatureProperties{}, false
	}
	point := r.Projection.ToGeo(float64(x), float64(y))
	// Last drawn is topmost, so the search runs backwards.
	for i := len(r.Shapes) - 1; i >= 0; i-- {
		for _, ring := range r.Shapes[i].Rings {
			if ring.Contains(point) {
				return r.Shapes[i].Properties, true
			}
		}
	}
	return api.FeatureProperties{}, false
}

// supersample is the factor the map is drawn at before being scaled down. Two is enough to take the
// staircase off a boundary; four costs four times the memory for a difference nobody sees.
const supersample = 2

// Options is what a caller varies between one drawing and the next.
type Options struct {
	Width, Height int
	SelectedID    int
	Palette       Palette
}

// Render draws the map and returns the picture with what is in it.
func Render(data api.MapData, opts Options) *Rendered {
	if opts.Width < 16 {
		opts.Width = 16
	}
	if opts.Height < 16 {
		opts.Height = 16
	}
	if opts.Palette == (Palette{}) {
		opts.Palette = DefaultPalette()
	}

	level := data.Level
	if level == "" {
		level = api.LevelBlock
	}

	shapes := make([]Shape, 0, len(data.Features.Features))
	var featureRings []geo.Ring
	for _, f := range data.Features.Features {
		rings := geo.Rings(f.Geometry)
		if len(rings) == 0 {
			continue
		}
		shapes = append(shapes, Shape{Properties: f.Properties, Rings: rings})
		featureRings = append(featureRings, rings...)
	}

	var outlineRings []geo.Ring
	for _, o := range data.Outlines.Features {
		outlineRings = append(outlineRings, geo.Rings(o.Geometry)...)
	}

	out := &Rendered{Shapes: shapes, Level: level, Empty: len(shapes) == 0}

	// The picture is drawn at a multiple of the requested size and scaled down, which is the whole
	// of the antialiasing: the rasteriser itself only has to decide in or out.
	w, h := opts.Width*supersample, opts.Height*supersample
	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	fill(canvas, opts.Palette.Background)

	// The outlines are included in the bounds so a farm's registered boundary is not clipped by the
	// smaller union of its blocks.
	bounds := geo.BoundsOf(featureRings, outlineRings)
	projection := geo.NewProjection(bounds, float64(w), float64(h), 0.06)
	out.Projection = geo.NewProjection(bounds, float64(opts.Width), float64(opts.Height), 0.06)

	if out.Empty && len(outlineRings) == 0 {
		out.Image = downsample(canvas, opts.Width, opts.Height)
		return out
	}

	// Registered boundaries first, underneath: a location shows both its full extent and the part
	// of it the filter admits.
	for _, ring := range outlineRings {
		strokeRing(canvas, projection, ring, opts.Palette.Outline, 2*supersample)
	}

	for _, shape := range shapes {
		selected := shape.Properties.ID == opts.SelectedID && opts.SelectedID != 0
		fillColour := fillOf(level, shape.Properties)
		edge, thickness := opts.Palette.Edge, 1*supersample
		if selected {
			edge, thickness = opts.Palette.Selected, 3*supersample
		}
		for _, ring := range shape.Rings {
			fillRing(canvas, projection, ring, fillColour)
			strokeRing(canvas, projection, ring, edge, thickness)
		}
	}

	// One label per location, on its largest piece — see geo.Largest. A code that will not fit on
	// the shape it names is left off: written anyway it lands on the neighbours, and a map where
	// every code overlaps two others names nothing at all.
	for _, shape := range shapes {
		ring, ok := geo.Largest(shape.Rings)
		if !ok {
			continue
		}
		if !fits(projection, ring, shape.Properties.Code) {
			continue
		}
		x, y := projection.ToPixel(ring.Centroid())
		drawLabel(canvas, int(x), int(y), shape.Properties.Code, opts.Palette.Label)
	}

	out.Image = downsample(canvas, opts.Width, opts.Height)
	return out
}

// labelOverhang is how far a code may run past the shape it names, as a fraction of its own width.
// Demanding an exact fit loses the code on a block that misses by a pixel — which, at estate scale,
// is most of them — and a letter over the edge reads far better than no name at all.
const labelOverhang = 0.15

// fits reports whether a code can be written across a ring without burying its neighbours.
func fits(p *geo.Projection, ring geo.Ring, code string) bool {
	pts := project(p, ring)
	minX, maxX := pts[0].X, pts[0].X
	minY, maxY := pts[0].Y, pts[0].Y
	for _, pt := range pts {
		minX = math.Min(minX, pt.X)
		maxX = math.Max(maxX, pt.X)
		minY = math.Min(minY, pt.Y)
		maxY = math.Max(maxY, pt.Y)
	}
	return maxX-minX >= float64(labelWidth(code))*(1-labelOverhang) &&
		maxY-minY >= float64(labelHeight())*1.5
}

// fillOf is how one feature is filled, which depends on the level being drawn.
func fillOf(level string, p api.FeatureProperties) color.RGBA {
	if level == api.LevelBlock {
		if c, ok := statusColours[p.CaneStatus]; ok {
			return c
		}
		return neutral
	}
	return BandFor(p.PlantedPercent).Fill
}

// ---------------------------------------------------------------- rasterising

func fill(img *image.RGBA, c color.RGBA) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// fillRing paints the inside of a ring by scanline, using the even-odd rule — the same rule the
// hit test uses, so what is painted and what answers a click are the same region.
func fillRing(img *image.RGBA, p *geo.Projection, ring geo.Ring, c color.RGBA) {
	if len(ring) < 3 {
		return
	}
	pts := project(p, ring)
	minY, maxY := pts[0].Y, pts[0].Y
	for _, pt := range pts {
		minY = math.Min(minY, pt.Y)
		maxY = math.Max(maxY, pt.Y)
	}
	bounds := img.Bounds()
	top := clamp(int(math.Floor(minY)), bounds.Min.Y, bounds.Max.Y-1)
	bottom := clamp(int(math.Ceil(maxY)), bounds.Min.Y, bounds.Max.Y-1)

	crossings := make([]float64, 0, len(pts))
	for y := top; y <= bottom; y++ {
		scan := float64(y) + 0.5
		crossings = crossings[:0]
		for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
			a, b := pts[i], pts[j]
			if (a.Y > scan) == (b.Y > scan) {
				continue
			}
			crossings = append(crossings, a.X+(scan-a.Y)*(b.X-a.X)/(b.Y-a.Y))
		}
		if len(crossings) < 2 {
			continue
		}
		sort.Float64s(crossings)
		for i := 0; i+1 < len(crossings); i += 2 {
			from := clamp(int(math.Ceil(crossings[i]-0.5)), bounds.Min.X, bounds.Max.X-1)
			to := clamp(int(math.Floor(crossings[i+1]-0.5)), bounds.Min.X, bounds.Max.X-1)
			for x := from; x <= to; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func strokeRing(img *image.RGBA, p *geo.Projection, ring geo.Ring, c color.RGBA, thickness int) {
	pts := project(p, ring)
	for i := range pts {
		a := pts[i]
		b := pts[(i+1)%len(pts)]
		line(img, a, b, c, thickness)
	}
}

type pixel struct{ X, Y float64 }

func project(p *geo.Projection, ring geo.Ring) []pixel {
	pts := make([]pixel, len(ring))
	for i, point := range ring {
		x, y := p.ToPixel(point)
		pts[i] = pixel{x, y}
	}
	return pts
}

// line is Bresenham widened into a square brush, which is all a boundary needs.
func line(img *image.RGBA, a, b pixel, c color.RGBA, thickness int) {
	if thickness < 1 {
		thickness = 1
	}
	x0, y0 := int(math.Round(a.X)), int(math.Round(a.Y))
	x1, y1 := int(math.Round(b.X)), int(math.Round(b.Y))

	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := step(x0, x1), step(y0, y1)
	err := dx + dy

	for {
		dot(img, x0, y0, c, thickness)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func dot(img *image.RGBA, x, y int, c color.RGBA, thickness int) {
	half := thickness / 2
	bounds := img.Bounds()
	for dy := -half; dy <= half; dy++ {
		for dx := -half; dx <= half; dx++ {
			px, py := x+dx, y+dy
			if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

// downsample averages each supersample × supersample square into one pixel.
func downsample(src *image.RGBA, width, height int) *image.RGBA {
	if supersample == 1 {
		return src
	}
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	n := supersample * supersample
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var r, g, b uint32
			for dy := 0; dy < supersample; dy++ {
				for dx := 0; dx < supersample; dx++ {
					c := src.RGBAAt(x*supersample+dx, y*supersample+dy)
					r += uint32(c.R)
					g += uint32(c.G)
					b += uint32(c.B)
				}
			}
			out.SetRGBA(x, y, color.RGBA{uint8(r / uint32(n)), uint8(g / uint32(n)), uint8(b / uint32(n)), 0xFF})
		}
	}
	return out
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func step(from, to int) int {
	if from < to {
		return 1
	}
	return -1
}
