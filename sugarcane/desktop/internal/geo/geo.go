// Package geo turns the service's GeoJSON into rings, and rings into pixels.
//
// None of it touches the window toolkit, which is what lets the map be rendered and checked on a
// machine with no Windows and no screen.
package geo

import (
	"encoding/json"
	"math"
)

// Point is a longitude and a latitude in degrees, in that order — the order GeoJSON uses.
type Point struct {
	Lng float64
	Lat float64
}

// Ring is one closed ring of a polygon.
type Ring []Point

// PointMarkerHalfWidth is half the side of the square drawn for a location that has coordinates but
// has not been surveyed, in degrees. Roughly forty metres, which is visible without dominating.
const PointMarkerHalfWidth = 0.0004

// Rings reads the rings out of a GeoJSON geometry. Polygon and MultiPolygon differ only in how
// deeply their coordinates nest; a Point becomes a small square, so a location with coordinates but
// no survey is still on the map rather than silently absent. Anything else yields nothing, because
// a map that panics on an unexpected geometry is worse than one that leaves a shape out.
func Rings(geometry json.RawMessage) []Ring {
	if len(geometry) == 0 {
		return nil
	}
	var shape struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal(geometry, &shape); err != nil || len(shape.Coordinates) == 0 {
		return nil
	}

	switch shape.Type {
	case "Polygon":
		var polygon []Ring
		if err := json.Unmarshal(shape.Coordinates, &polygon); err != nil {
			return nil
		}
		return valid(polygon)

	case "MultiPolygon":
		var multi [][]Ring
		if err := json.Unmarshal(shape.Coordinates, &multi); err != nil {
			return nil
		}
		var out []Ring
		for _, polygon := range multi {
			out = append(out, valid(polygon)...)
		}
		return out

	case "Point":
		var centre Point
		if err := json.Unmarshal(shape.Coordinates, &centre); err != nil {
			return nil
		}
		d := PointMarkerHalfWidth
		return []Ring{{
			{centre.Lng - d, centre.Lat - d},
			{centre.Lng + d, centre.Lat - d},
			{centre.Lng + d, centre.Lat + d},
			{centre.Lng - d, centre.Lat + d},
			{centre.Lng - d, centre.Lat - d},
		}}
	}
	return nil
}

// UnmarshalJSON reads the [longitude, latitude] pair GeoJSON writes a position as.
func (p *Point) UnmarshalJSON(raw []byte) error {
	var pair []float64
	if err := json.Unmarshal(raw, &pair); err != nil {
		return err
	}
	if len(pair) >= 2 {
		p.Lng, p.Lat = pair[0], pair[1]
	}
	return nil
}

func valid(rings []Ring) []Ring {
	out := rings[:0]
	for _, ring := range rings {
		if len(ring) >= 3 {
			out = append(out, ring)
		}
	}
	return out
}

// Area is the shoelace area in degrees squared. It is only ever compared against another ring of
// the same location — to find the largest piece of a multi-part shape, which is the one with room
// for a label — so the unit does not matter and no projection is needed.
func (r Ring) Area() float64 {
	var sum float64
	for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
		sum += r[j].Lng*r[i].Lat - r[i].Lng*r[j].Lat
	}
	return math.Abs(sum) / 2
}

// Centroid is the average of the ring's points, which is where its label goes.
//
// A GeoJSON ring is closed by repeating its first point, and counting that point twice pulls the
// average towards it — enough to sit a label off-centre, and worse the fewer points the ring has.
func (r Ring) Centroid() Point {
	points := r
	if n := len(points); n > 1 && points[n-1] == points[0] {
		points = points[:n-1]
	}
	if len(points) == 0 {
		return Point{}
	}
	var x, y float64
	for _, p := range points {
		x += p.Lng
		y += p.Lat
	}
	return Point{x / float64(len(points)), y / float64(len(points))}
}

// Contains reports whether a point falls inside the ring, by ray casting. This is what turns a
// click on the map into a location.
func (r Ring) Contains(p Point) bool {
	inside := false
	for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
		a, b := r[i], r[j]
		if (a.Lat > p.Lat) != (b.Lat > p.Lat) &&
			p.Lng < (b.Lng-a.Lng)*(p.Lat-a.Lat)/(b.Lat-a.Lat)+a.Lng {
			inside = !inside
		}
	}
	return inside
}

// Bounds is the box everything the map draws has to fit into.
type Bounds struct {
	MinLng, MinLat, MaxLng, MaxLat float64
}

// Empty reports whether the bounds cover nothing, which happens when there is no geometry at all.
func (b Bounds) Empty() bool { return b.MinLng > b.MaxLng || b.MinLat > b.MaxLat }

// BoundsOf is the box covering every ring given to it.
func BoundsOf(rings ...[]Ring) Bounds {
	b := Bounds{
		MinLng: math.Inf(1), MinLat: math.Inf(1),
		MaxLng: math.Inf(-1), MaxLat: math.Inf(-1),
	}
	for _, set := range rings {
		for _, ring := range set {
			for _, p := range ring {
				b.MinLng = math.Min(b.MinLng, p.Lng)
				b.MaxLng = math.Max(b.MaxLng, p.Lng)
				b.MinLat = math.Min(b.MinLat, p.Lat)
				b.MaxLat = math.Max(b.MaxLat, p.Lat)
			}
		}
	}
	return b
}

// Projection turns degrees into pixels and back.
//
// Equirectangular, with longitude scaled by the cosine of the centre latitude. Over an estate that
// is accurate to well under a metre of relative position, and unlike Web Mercator the arithmetic
// stays legible. It is the same projection the browser client uses, so the two draw one picture.
type Projection struct {
	width, height float64
	minX, minLat  float64
	spanX, spanY  float64
	scaleLng      float64
	padding       float64
}

// minimumSpan stops a single block, whose bounds are one point, from dividing by nought.
const minimumSpan = 0.0005

// NewProjection fits bounds into a width by height rectangle, leaving a margin of padding on each
// side as a fraction of the span.
func NewProjection(b Bounds, width, height, padding float64) *Projection {
	var midLat float64
	if !b.Empty() {
		midLat = (b.MinLat + b.MaxLat) / 2
	}
	scaleLng := math.Cos(midLat * math.Pi / 180)

	p := &Projection{
		width: width, height: height,
		scaleLng: scaleLng, padding: padding,
		spanX: minimumSpan, spanY: minimumSpan,
	}
	if !b.Empty() {
		p.minX = b.MinLng * scaleLng
		p.minLat = b.MinLat
		p.spanX = math.Max((b.MaxLng-b.MinLng)*scaleLng, minimumSpan)
		p.spanY = math.Max(b.MaxLat-b.MinLat, minimumSpan)
	}
	return p
}

// Size is the rectangle the projection fits into.
func (p *Projection) Size() (width, height float64) { return p.width, p.height }

// ToPixel places a point. Latitude grows northwards and y downwards, so y is inverted.
func (p *Projection) ToPixel(pt Point) (x, y float64) {
	x = ((pt.Lng * p.scaleLng) - p.minX + p.spanX*p.padding) / (p.spanX * (1 + p.padding*2)) * p.width
	y = p.height - ((pt.Lat-p.minLat+p.spanY*p.padding)/(p.spanY*(1+p.padding*2)))*p.height
	return x, y
}

// ToGeo places a pixel, which is what turns a click into a place.
func (p *Projection) ToGeo(x, y float64) Point {
	lng := ((x / p.width * (p.spanX * (1 + p.padding*2))) - p.spanX*p.padding + p.minX) / p.scaleLng
	lat := ((p.height-y)/p.height*(p.spanY*(1+p.padding*2)) - p.spanY*p.padding) + p.minLat
	return Point{lng, lat}
}

// Largest is the ring of a shape with the most room for a label. A farm drawn as the union of a
// dozen disjoint blocks would otherwise have its code written a dozen times, once per piece.
func Largest(rings []Ring) (Ring, bool) {
	var best Ring
	bestArea := -1.0
	for _, ring := range rings {
		if a := ring.Area(); a > bestArea {
			best, bestArea = ring, a
		}
	}
	return best, bestArea >= 0
}
