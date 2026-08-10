package geo_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/geo"
)

func TestPolygonYieldsItsRings(t *testing.T) {
	rings := geo.Rings(json.RawMessage(
		`{"type":"Polygon","coordinates":[[[28.0,-15.0],[28.1,-15.0],[28.1,-15.1],[28.0,-15.1],[28.0,-15.0]]]}`))

	if len(rings) != 1 {
		t.Fatalf("expected one ring, got %d", len(rings))
	}
	if len(rings[0]) != 5 {
		t.Fatalf("expected five points, got %d", len(rings[0]))
	}
	if rings[0][0].Lng != 28.0 || rings[0][0].Lat != -15.0 {
		t.Errorf("longitude and latitude are the wrong way round: %+v", rings[0][0])
	}
}

func TestMultiPolygonIsFlattenedToEveryRing(t *testing.T) {
	rings := geo.Rings(json.RawMessage(`{"type":"MultiPolygon","coordinates":[
		[[[28.0,-15.0],[28.1,-15.0],[28.1,-15.1],[28.0,-15.0]]],
		[[[28.2,-15.0],[28.3,-15.0],[28.3,-15.1],[28.2,-15.0]]]]}`))

	if len(rings) != 2 {
		t.Fatalf("expected two rings, got %d", len(rings))
	}
}

func TestAPointBecomesASquareSoAnUnsurveyedBlockIsStillDrawn(t *testing.T) {
	rings := geo.Rings(json.RawMessage(`{"type":"Point","coordinates":[28.25,-15.5]}`))

	if len(rings) != 1 {
		t.Fatalf("expected one ring, got %d", len(rings))
	}
	centre := rings[0].Centroid()
	if math.Abs(centre.Lng-28.25) > 1e-9 || math.Abs(centre.Lat+15.5) > 1e-9 {
		t.Errorf("the square is not centred on the block: %+v", centre)
	}
	if !rings[0].Contains(geo.Point{Lng: 28.25, Lat: -15.5}) {
		t.Error("the block's own coordinates fall outside the square drawn for it")
	}
}

func TestAGeometryTheMapCannotDrawYieldsNothing(t *testing.T) {
	for _, raw := range []string{
		`{"type":"LineString","coordinates":[[28.0,-15.0],[28.1,-15.1]]}`,
		`{"type":"Polygon"}`,
		`{"type":"Polygon","coordinates":[[[28.0,-15.0],[28.1,-15.0]]]}`, // too few points to enclose anything
		`{"nonsense":true}`,
		`null`,
		``,
	} {
		if rings := geo.Rings(json.RawMessage(raw)); len(rings) != 0 {
			t.Errorf("%s drew %d ring(s), expected none", raw, len(rings))
		}
	}
}

func TestARingKnowsWhatIsInsideIt(t *testing.T) {
	ring := geo.Ring{{28.0, -15.0}, {28.1, -15.0}, {28.1, -15.1}, {28.0, -15.1}, {28.0, -15.0}}

	cases := []struct {
		point geo.Point
		want  bool
	}{
		{geo.Point{Lng: 28.05, Lat: -15.05}, true},
		{geo.Point{Lng: 28.2, Lat: -15.05}, false},
		{geo.Point{Lng: 28.05, Lat: -14.9}, false},
	}
	for _, c := range cases {
		if got := ring.Contains(c.point); got != c.want {
			t.Errorf("Contains(%+v) = %v, want %v", c.point, got, c.want)
		}
	}
}

func TestTheLargestRingIsTheOneWithRoomForALabel(t *testing.T) {
	small := geo.Ring{{28.0, -15.0}, {28.01, -15.0}, {28.01, -15.01}, {28.0, -15.0}}
	large := geo.Ring{{28.0, -15.0}, {28.5, -15.0}, {28.5, -15.5}, {28.0, -15.0}}

	ring, ok := geo.Largest([]geo.Ring{small, large, small})
	if !ok {
		t.Fatal("no ring was chosen")
	}
	if ring.Area() != large.Area() {
		t.Error("the label would be written on a sliver rather than the main body")
	}
	if _, ok := geo.Largest(nil); ok {
		t.Error("a shape with no rings should offer nowhere to write")
	}
}

func TestProjectingAPixelBackGivesTheSamePlace(t *testing.T) {
	p := geo.NewProjection(geo.Bounds{MinLng: 28.0, MinLat: -15.6, MaxLng: 28.5, MaxLat: -15.3}, 1000, 600, 0.06)

	original := geo.Point{Lng: 28.25, Lat: -15.45}
	x, y := p.ToPixel(original)
	round := p.ToGeo(x, y)

	if math.Abs(round.Lng-original.Lng) > 1e-9 || math.Abs(round.Lat-original.Lat) > 1e-9 {
		t.Errorf("a click would land at %+v rather than %+v", round, original)
	}
}

func TestNorthIsUpAndEastIsRight(t *testing.T) {
	p := geo.NewProjection(geo.Bounds{MinLng: 28.0, MinLat: -15.6, MaxLng: 28.5, MaxLat: -15.3}, 1000, 600, 0.06)

	westX, _ := p.ToPixel(geo.Point{Lng: 28.0, Lat: -15.45})
	eastX, _ := p.ToPixel(geo.Point{Lng: 28.5, Lat: -15.45})
	_, southY := p.ToPixel(geo.Point{Lng: 28.25, Lat: -15.6})
	_, northY := p.ToPixel(geo.Point{Lng: 28.25, Lat: -15.3})

	if !(eastX > westX) {
		t.Error("east is drawn to the left of west")
	}
	if !(northY < southY) {
		t.Error("north is drawn below south")
	}
}

func TestASingleBlockStillProjects(t *testing.T) {
	// One block's bounds are a single point; without a floor on the span this divides by nought.
	p := geo.NewProjection(geo.Bounds{MinLng: 28.25, MinLat: -15.5, MaxLng: 28.25, MaxLat: -15.5}, 400, 400, 0.06)

	x, y := p.ToPixel(geo.Point{Lng: 28.25, Lat: -15.5})

	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		t.Fatalf("the block projected to (%v, %v)", x, y)
	}
	if x < 0 || x > 400 || y < 0 || y > 400 {
		t.Errorf("the block landed outside the map at (%v, %v)", x, y)
	}
}

func TestEmptyBoundsAreReportedRatherThanProjected(t *testing.T) {
	b := geo.BoundsOf()
	if !b.Empty() {
		t.Error("bounds over nothing should be empty")
	}
	p := geo.NewProjection(b, 300, 300, 0.06)
	x, y := p.ToPixel(geo.Point{Lng: 28, Lat: -15})
	if math.IsNaN(x) || math.IsNaN(y) {
		t.Errorf("projecting against empty bounds gave (%v, %v)", x, y)
	}
}

func TestBoundsCoverEveryRingGivenToThem(t *testing.T) {
	b := geo.BoundsOf(
		[]geo.Ring{{{28.0, -15.0}, {28.1, -15.2}, {28.0, -15.0}}},
		[]geo.Ring{{{28.4, -15.1}, {28.5, -14.9}, {28.4, -15.1}}},
	)

	if b.MinLng != 28.0 || b.MaxLng != 28.5 || b.MinLat != -15.2 || b.MaxLat != -14.9 {
		t.Errorf("bounds %+v do not cover both sets of rings", b)
	}
}
