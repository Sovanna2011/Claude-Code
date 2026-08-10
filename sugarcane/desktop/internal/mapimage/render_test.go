package mapimage_test

import (
	"context"
	"encoding/json"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/api"
	"github.com/sovanna2011/sugarcane-go/desktop/internal/mapimage"
)

// twoBlocks is a hand-made answer in the service's shape: two square blocks side by side, one
// growing and one fallow, inside a farm outline that covers both.
func twoBlocks() api.MapData {
	feature := func(id int, code, status string, west float64, planted float64) api.Feature[api.FeatureProperties] {
		geometry := json.RawMessage(`{"type":"Polygon","coordinates":[[` +
			pt(west, -15.10) + "," + pt(west+0.03, -15.10) + "," +
			pt(west+0.03, -15.07) + "," + pt(west, -15.07) + "," + pt(west, -15.10) + `]]}`)
		return api.Feature[api.FeatureProperties]{
			Geometry: geometry,
			Properties: api.FeatureProperties{
				Level: api.LevelBlock, ID: id, Code: code, Name: code + " name",
				TotalAreaHa: 100, CaneStatus: status, PlantedPercent: planted,
			},
		}
	}

	return api.MapData{
		Level: api.LevelBlock,
		Features: api.FeatureCollection[api.FeatureProperties]{
			Type: "FeatureCollection",
			Features: []api.Feature[api.FeatureProperties]{
				feature(1, "BLK-001", "Growing", 28.00, 90),
				feature(2, "BLK-002", "Fallow", 28.05, 0),
			},
		},
		Outlines: api.FeatureCollection[api.OutlineProperties]{
			Type: "FeatureCollection",
			Features: []api.Feature[api.OutlineProperties]{{
				Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[` +
					pt(27.99, -15.11) + "," + pt(28.09, -15.11) + "," +
					pt(28.09, -15.06) + "," + pt(27.99, -15.06) + "," + pt(27.99, -15.11) + `]]}`),
				Properties: api.OutlineProperties{Level: api.LevelFarm, ID: 1, Code: "FRM-01"},
			}},
		},
	}
}

func pt(lng, lat float64) string {
	b, _ := json.Marshal([]float64{lng, lat})
	return string(b)
}

func TestTheMapIsDrawnAtTheSizeAskedFor(t *testing.T) {
	out := mapimage.Render(twoBlocks(), mapimage.Options{Width: 400, Height: 300})

	if out.Image.Bounds().Dx() != 400 || out.Image.Bounds().Dy() != 300 {
		t.Fatalf("drew %v, want 400×300", out.Image.Bounds())
	}
	if out.Empty {
		t.Error("two blocks were given but the map reports itself empty")
	}
	if len(out.Shapes) != 2 {
		t.Errorf("kept %d shapes, want 2", len(out.Shapes))
	}
}

func TestAClickLandsOnTheBlockUnderIt(t *testing.T) {
	out := mapimage.Render(twoBlocks(), mapimage.Options{Width: 400, Height: 300})

	// The centre of each block, projected, has to answer with that block.
	for _, shape := range out.Shapes {
		centre := shape.Rings[0].Centroid()
		x, y := out.Projection.ToPixel(centre)

		hit, ok := out.At(int(x), int(y))
		if !ok {
			t.Fatalf("the centre of %s hit nothing", shape.Properties.Code)
		}
		if hit.ID != shape.Properties.ID {
			t.Errorf("the centre of %s answered with %s", shape.Properties.Code, hit.Code)
		}
	}

	// A corner of the picture is padding, and belongs to no block.
	if _, ok := out.At(1, 1); ok {
		t.Error("the margin answered with a block")
	}
}

func TestABlockIsFilledByWhatIsGrowingOnIt(t *testing.T) {
	out := mapimage.Render(twoBlocks(), mapimage.Options{Width: 400, Height: 300})

	growing := colourAtCentreOf(t, out, "BLK-001")
	fallow := colourAtCentreOf(t, out, "BLK-002")

	if growing == fallow {
		t.Error("a growing block and a fallow one are filled the same")
	}
	// Growing is the positive green, so its green channel leads.
	if !(growing.G > growing.R && growing.G > growing.B) {
		t.Errorf("a growing block is not green: %+v", growing)
	}
}

func TestAFarmIsFilledByHowMuchOfItIsPlanted(t *testing.T) {
	data := twoBlocks()
	data.Level = api.LevelFarm
	for i := range data.Features.Features {
		data.Features.Features[i].Properties.Level = api.LevelFarm
		data.Features.Features[i].Properties.CaneStatus = "" // a farm has no single status
	}

	out := mapimage.Render(data, mapimage.Options{Width: 400, Height: 300})

	// The one at 90% planted must be darker than the one at nought.
	mostly := colourAtCentreOf(t, out, "BLK-001")
	none := colourAtCentreOf(t, out, "BLK-002")
	if luminance(mostly) >= luminance(none) {
		t.Errorf("a farm 90%% planted (%+v) is not darker than one with nothing on it (%+v)", mostly, none)
	}
}

func TestTheBandsCoverEveryShareWithoutOverlapping(t *testing.T) {
	cases := []struct {
		percent float64
		want    string
	}{
		{0, "Nothing planted"},
		{0.5, "Nothing planted"},
		{1, "Under 40% planted"},
		{39.9, "Under 40% planted"},
		{40, "40 to 80% planted"},
		{79.9, "40 to 80% planted"},
		{80, "Over 80% planted"},
		{100, "Over 80% planted"},
		{140, "Over 80% planted"}, // more land planted than registered: still the top band, not a panic
	}
	for _, c := range cases {
		if got := mapimage.BandFor(c.percent).Label; got != c.want {
			t.Errorf("BandFor(%v) = %q, want %q", c.percent, got, c.want)
		}
	}
}

func TestTheLegendFollowsTheLevel(t *testing.T) {
	block := mapimage.Legend(api.LevelBlock)
	farm := mapimage.Legend(api.LevelFarm)

	if block[0].Label != "Growing" {
		t.Errorf("the block legend starts %q", block[0].Label)
	}
	if len(farm) != len(mapimage.Bands) || farm[0].Label != "Nothing planted" {
		t.Errorf("the farm legend is not the planted-share ramp: %+v", farm)
	}
}

func TestAnEmptyAnswerIsReportedRatherThanDrawnBlank(t *testing.T) {
	out := mapimage.Render(api.MapData{Level: api.LevelBlock}, mapimage.Options{Width: 200, Height: 150})

	if !out.Empty {
		t.Error("nothing was given but the map does not report itself empty")
	}
	if out.Image == nil || out.Image.Bounds().Dx() != 200 {
		t.Error("an empty map still has to be a picture of the right size")
	}
	if _, ok := out.At(100, 75); ok {
		t.Error("an empty map answered a click with a location")
	}
}

func TestASelectedBlockIsDrawnDifferently(t *testing.T) {
	plain := mapimage.Render(twoBlocks(), mapimage.Options{Width: 400, Height: 300})
	picked := mapimage.Render(twoBlocks(), mapimage.Options{Width: 400, Height: 300, SelectedID: 1})

	if sameImage(plain, picked) {
		t.Error("selecting a block changed nothing on the map")
	}
}

// TestRenderTheRealEstate draws the live service's own data and writes it out, so the picture can
// be looked at rather than assumed. It skips without a service, like the client's own tests.
func TestRenderTheRealEstate(t *testing.T) {
	url := os.Getenv("FARMAREA_API_URL")
	if url == "" {
		t.Skip("set FARMAREA_API_URL to render the real estate")
	}

	client := api.New(url)
	if _, err := client.SignIn(context.Background(), "admin", "Farm#2026"); err != nil {
		t.Fatalf("sign in: %v", err)
	}

	for _, level := range api.MapLevels {
		data, err := client.Map(context.Background(), api.Filter{CropYear: api.Int(2026)}, level)
		if err != nil {
			t.Fatalf("map by %s: %v", level, err)
		}

		out := mapimage.Render(data, mapimage.Options{Width: 900, Height: 620})
		if out.Empty {
			t.Fatalf("%s level drew nothing", level)
		}
		if len(out.Shapes) != len(data.Features.Features) {
			t.Errorf("%s level: the service sent %d features, %d were drawn — one has a geometry "+
				"the map cannot read", level, len(data.Features.Features), len(out.Shapes))
		}

		// Every drawn location has to answer a click at its own centre, or the map is a picture
		// rather than a control.
		for _, shape := range out.Shapes {
			ring := shape.Rings[0]
			x, y := out.Projection.ToPixel(ring.Centroid())
			if hit, ok := out.At(int(x), int(y)); !ok || hit.ID != shape.Properties.ID {
				// A concave shape's centroid can fall outside it, which is not a fault; only a
				// click that answers with the wrong location is.
				if ok && hit.ID != shape.Properties.ID {
					t.Errorf("%s level: the centre of %s answered with %s",
						level, shape.Properties.Code, hit.Code)
				}
			}
		}

		if dir := os.Getenv("FARMAREA_MAP_OUT"); dir != "" {
			write(t, filepath.Join(dir, "map-"+level+".png"), out)
		}
	}
}

func write(t *testing.T, path string, out *mapimage.Rendered) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer file.Close()
	if err := png.Encode(file, out.Image); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	t.Logf("wrote %s", path)
}

func colourAtCentreOf(t *testing.T, out *mapimage.Rendered, code string) color.RGBA {
	t.Helper()
	for _, shape := range out.Shapes {
		if shape.Properties.Code != code {
			continue
		}
		// A little off centre, so the label's own pixels are not sampled.
		centre := shape.Rings[0].Centroid()
		x, y := out.Projection.ToPixel(centre)
		return out.Image.RGBAAt(int(x), int(y)+20)
	}
	t.Fatalf("%s was not drawn", code)
	return color.RGBA{}
}

func sameImage(a, b *mapimage.Rendered) bool {
	if a.Image.Bounds() != b.Image.Bounds() {
		return false
	}
	for i := range a.Image.Pix {
		if a.Image.Pix[i] != b.Image.Pix[i] {
			return false
		}
	}
	return true
}

func luminance(c color.RGBA) int {
	return int((299*uint32(c.R) + 587*uint32(c.G) + 114*uint32(c.B)) / 1000)
}
