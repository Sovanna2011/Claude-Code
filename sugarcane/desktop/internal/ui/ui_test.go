package ui_test

import (
	"math"
	"strings"
	"testing"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/api"
	"github.com/sovanna2011/sugarcane-go/desktop/internal/ui"
)

// The window's own logic: how a figure is written, and how the hierarchy is laid out as rows. It is
// outside the //go:build windows files on purpose, so the part of the form that can be wrong in a
// way a reader would notice is the part that is tested.

func TestFiguresAreWrittenOneWay(t *testing.T) {
	cases := []struct {
		value float64
		want  string
	}{
		{0, "0.0"},
		{130.9, "130.9"},
		{2618.75, "2,618.8"}, // rounded to one decimal, thousands separated
		{1234567.89, "1,234,567.9"},
		{-91.7, "-91.7"},
		{-1234.5, "-1,234.5"},
	}
	for _, c := range cases {
		if got := ui.Hectares(c.value); got != c.want {
			t.Errorf("Hectares(%v) = %q, want %q", c.value, got, c.want)
		}
	}
}

func TestAFigureThatIsNotANumberIsNotWrittenAsOne(t *testing.T) {
	// A percentage of nought hectares is not nought per cent, it is nothing to report.
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if got := ui.Percent(v); got != "—" {
			t.Errorf("Percent(%v) = %q, want an em dash", v, got)
		}
		if got := ui.Hectares(v); got != "—" {
			t.Errorf("Hectares(%v) = %q, want an em dash", v, got)
		}
	}
	if got := ui.Percent(31.37); got != "31.4%" {
		t.Errorf("Percent(31.37) = %q", got)
	}
}

func TestTheStatusBarSaysWhatLandIsBeingDescribed(t *testing.T) {
	if got := ui.FilterSummary(api.Filter{}); !strings.Contains(got, "whole estate") {
		t.Errorf("with nothing set the status bar says %q", got)
	}
	if got := ui.FilterSummary(api.Filter{FarmID: api.Int(1)}); got != "1 filter applied" {
		t.Errorf("with one filter set the status bar says %q", got)
	}
	if got := ui.FilterSummary(api.Filter{FarmID: api.Int(1), CropYear: api.Int(2026)}); got != "2 filters applied" {
		t.Errorf("with two filters set the status bar says %q", got)
	}
}

func sample() []*api.TreeNode {
	block := func(id int, code string) *api.TreeNode {
		return &api.TreeNode{NodeType: "Block", ID: id, Code: code, Name: code + " name",
			Areas: api.Areas{TotalHa: 100}, BlockCount: 1}
	}
	return []*api.TreeNode{{
		NodeType: "Farm", ID: 1, Code: "FRM-01", Name: "Riverside Farm",
		Areas: api.Areas{TotalHa: 400}, BlockCount: 4,
		Children: []*api.TreeNode{
			{NodeType: "Zone", ID: 1, Code: "FRM-01-Z1", Name: "Zone 1",
				Areas: api.Areas{TotalHa: 200}, BlockCount: 2,
				Children: []*api.TreeNode{block(1, "BLK-001"), block(2, "BLK-002")}},
			{NodeType: "Zone", ID: 2, Code: "FRM-01-Z2", Name: "Zone 2",
				Areas: api.Areas{TotalHa: 200}, BlockCount: 2,
				Children: []*api.TreeNode{block(3, "BLK-003"), block(4, "BLK-004")}},
		},
	}}
}

func TestTheHierarchyIsLaidOutDepthFirstAndIndented(t *testing.T) {
	rows := ui.FlattenTree(sample())

	want := []struct {
		depth int
		code  string
	}{
		{0, "FRM-01"},
		{1, "FRM-01-Z1"},
		{2, "BLK-001"},
		{2, "BLK-002"},
		{1, "FRM-01-Z2"},
		{2, "BLK-003"},
		{2, "BLK-004"},
	}
	if len(rows) != len(want) {
		t.Fatalf("laid out %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		if rows[i].Depth != w.depth || rows[i].Node.Code != w.code {
			t.Errorf("row %d is %s at depth %d, want %s at %d",
				i, rows[i].Node.Code, rows[i].Depth, w.code, w.depth)
		}
	}
	if !strings.HasPrefix(rows[2].Label(), "        BLK-001") {
		t.Errorf("a block is not indented under its zone: %q", rows[2].Label())
	}
	if !strings.HasPrefix(rows[0].Label(), "FRM-01") {
		t.Errorf("a farm is indented: %q", rows[0].Label())
	}
}

func TestAnEmptyHierarchyLaysOutNothing(t *testing.T) {
	if rows := ui.FlattenTree(nil); len(rows) != 0 {
		t.Errorf("laid out %d rows from nothing", len(rows))
	}
}

func TestTheHierarchyPanelCountsWhatIsInIt(t *testing.T) {
	got := ui.TreeSummary(sample(), 4, 400)

	for _, want := range []string{"1 farm(s)", "2 zone(s)", "4 block(s)", "400.0 ha"} {
		if !strings.Contains(got, want) {
			t.Errorf("the summary %q is missing %q", got, want)
		}
	}
}

func TestTheDetailPanelShowsWhatTheLevelHas(t *testing.T) {
	block := api.FeatureProperties{
		Level: api.LevelBlock, Code: "BLK-001", Name: "Block 1",
		FarmName: "Riverside Farm", ZoneName: "Zone 1", CaneStatus: "Growing",
		TotalAreaHa: 130.9, CaneVarietyName: "NCo 376", PlantingDate: "2026-03-13",
	}
	farm := api.FeatureProperties{
		Level: api.LevelFarm, Code: "FRM-01", Name: "Riverside Farm",
		BlockCount: 7, PlantedPercent: 31.37, TotalAreaHa: 916.3,
	}

	blockText := ui.LocationDetail(block)
	for _, want := range []string{"BLK-001", "Riverside Farm", "Growing", "NCo 376", "2026-03-13", "130.9 ha"} {
		if !strings.Contains(blockText, want) {
			t.Errorf("the block panel is missing %q:\n%s", want, blockText)
		}
	}
	if strings.Contains(blockText, "Share planted") {
		t.Error("a block is shown a farm's figure")
	}

	farmText := ui.LocationDetail(farm)
	for _, want := range []string{"FRM-01", "Blocks", "7", "31.4%", "916.3 ha"} {
		if !strings.Contains(farmText, want) {
			t.Errorf("the farm panel is missing %q:\n%s", want, farmText)
		}
	}
	// A farm has no single variety, status or planting date, so it is not offered empty ones.
	for _, absent := range []string{"Variety", "Cane status", "Planted on"} {
		if strings.Contains(farmText, absent) {
			t.Errorf("the farm panel offers %q, which a farm does not have:\n%s", absent, farmText)
		}
	}
}

func TestNothingSelectedSaysSo(t *testing.T) {
	got := ui.LocationDetail(api.FeatureProperties{})
	if !strings.Contains(got, "Select a location") {
		t.Errorf("with nothing selected the panel says %q", got)
	}
}
