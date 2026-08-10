//go:build windows

package ui

import (
	"github.com/lxn/walk"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/api"
	"github.com/sovanna2011/sugarcane-go/desktop/internal/mapimage"
)

// The map is drawn by the mapimage package into an ordinary picture, and this file only puts that
// picture on the screen and turns a click back into a location. Keeping the drawing out of the
// window is what lets the map be checked on a machine with no Windows: the same function that fills
// this control writes the PNG a test looks at.

// setLevel redraws the map by farm, by zone or by block. The whole screen is refreshed rather than
// only the map, because the service groups the same land differently at each level and the figures
// beside it have to agree with what is drawn.
func (f *mainForm) setLevel(level string) {
	if level == f.mapLevel {
		return
	}
	f.mapLevel = level
	f.selected = api.FeatureProperties{}
	f.detailBox.SetText(LocationDetail(api.FeatureProperties{}))
	f.openMaps.SetEnabled(false)
	f.markLevel()
	f.reload()
}

// markLevel shows which of the three buttons is the level being drawn. walk's push button has no
// checked state, so the current one is named in full and the others are not.
func (f *mainForm) markLevel() {
	for level, button := range f.levelButtons {
		if button == nil {
			continue
		}
		if level == f.mapLevel {
			button.SetText("▸ " + level)
		} else {
			button.SetText(level)
		}
	}
}

func (f *mainForm) shapes() []mapimage.Shape {
	if f.rendered == nil {
		return nil
	}
	return f.rendered.Shapes
}

// renderMap draws the current answer at the control's current size. It runs on the window thread
// and is cheap — a few hundred polygons scan-filled — so there is no need to push it off.
func (f *mainForm) renderMap() {
	if f.mapWidget == nil {
		return
	}
	bounds := f.mapWidget.ClientBoundsPixels()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}

	f.rendered = mapimage.Render(f.mapData, mapimage.Options{
		Width:      bounds.Width,
		Height:     bounds.Height,
		SelectedID: f.selected.ID,
		Palette:    systemPalette(),
	})

	bitmap, err := walk.NewBitmapFromImageForDPI(f.rendered.Image, f.mapWidget.DPI())
	if err != nil {
		report(f, "Map", err)
		return
	}
	f.disposeBitmap()
	f.bitmap = bitmap
	f.mapWidget.Invalidate()
}

func (f *mainForm) disposeBitmap() {
	if f.bitmap != nil {
		f.bitmap.Dispose()
		f.bitmap = nil
	}
}

// paintMap puts the rendered picture on the control. The size is checked on every paint because a
// resize has to redraw at the new size rather than stretch the old picture.
func (f *mainForm) paintMap(canvas *walk.Canvas, _ walk.Rectangle) error {
	bounds := f.mapWidget.ClientBoundsPixels()

	if f.bitmap == nil || f.bitmap.Size().Width != bounds.Width || f.bitmap.Size().Height != bounds.Height {
		f.renderMap()
	}
	if f.bitmap == nil {
		return nil
	}
	if err := canvas.DrawImagePixels(f.bitmap, walk.Point{}); err != nil {
		return err
	}

	if f.rendered != nil && f.rendered.Empty {
		return canvas.DrawTextPixels(
			"No "+lower(f.mapLevel)+" has a boundary or coordinates for the current filter.",
			f.Font(), walk.RGB(0x55, 0x6B, 0x82), bounds,
			walk.TextCenter|walk.TextVCenter|walk.TextWordbreak)
	}
	return nil
}

// mapClicked turns a click into the location under it, and selects it. A click that lands on no
// shape clears the selection, which is how a reader gets back to the whole estate.
func (f *mainForm) mapClicked(x, y int, button walk.MouseButton) {
	if button != walk.LeftButton || f.rendered == nil {
		return
	}
	if hit, ok := f.rendered.At(x, y); ok {
		f.selectFeature(hit)
		f.selectRowFor(hit)
		return
	}
	f.selectFeature(api.FeatureProperties{})
}

// selectRowFor moves the hierarchy table to the location just clicked on the map, so the two halves
// of the window describe the same place.
func (f *mainForm) selectRowFor(p api.FeatureProperties) {
	for i, row := range f.rows.rows {
		if row.Node.NodeType == p.Level && row.Node.ID == p.ID {
			_ = f.table.SetCurrentIndex(i)
			return
		}
	}
}

// paintLegend explains whichever fill the current level is using: a block's cane status, or how
// much of a farm or a zone is already planted.
func (f *mainForm) paintLegend(canvas *walk.Canvas, _ walk.Rectangle) error {
	entries := mapimage.Legend(f.mapLevel)
	bounds := f.legendWidget.ClientBoundsPixels()

	x := 0
	swatch := bounds.Height - 10
	if swatch < 6 {
		swatch = 6
	}

	for _, entry := range entries {
		brush, err := walk.NewSolidColorBrush(walk.RGB(entry.Fill.R, entry.Fill.G, entry.Fill.B))
		if err != nil {
			return err
		}
		err = canvas.FillRectanglePixels(brush,
			walk.Rectangle{X: x, Y: (bounds.Height - swatch) / 2, Width: swatch, Height: swatch})
		brush.Dispose()
		if err != nil {
			return err
		}
		x += swatch + 4

		width := 8*len(entry.Label) + 12
		if err := canvas.DrawTextPixels(entry.Label, f.Font(), walk.RGB(0x40, 0x50, 0x60),
			walk.Rectangle{X: x, Y: 0, Width: width, Height: bounds.Height},
			walk.TextLeft|walk.TextVCenter|walk.TextSingleLine); err != nil {
			return err
		}
		x += width
	}
	return nil
}

// systemPalette is the map's colours. They are fixed rather than taken from the Windows theme: the
// fills carry meaning — this block is growing, that farm is barely planted — and a colour that
// changed with the user's theme would change what the map says.
func systemPalette() mapimage.Palette {
	return mapimage.DefaultPalette()
}

func lower(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r >= 'A' && r <= 'Z' {
			out[i] = r + ('a' - 'A')
		}
	}
	return string(out)
}
