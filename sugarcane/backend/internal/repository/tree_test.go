package repository

import (
	"testing"

	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
)

// buildTree is where farm and zone totals come from. The specification forbids entering them, so
// this is the only place they can be wrong.

func row(farm int, farmCode string, zone *int, zoneCode *string, block *int, blockCode *string,
	total, plantable, nonPlantable, newPlanting, ratoon float64) treeRow {
	return treeRow{
		FarmID: farm, FarmCode: farmCode, FarmName: farmCode + " farm",
		ZoneID: zone, ZoneCode: zoneCode, ZoneName: zoneCode,
		BlockID: block, BlockCode: blockCode, BlockName: blockCode,
		Areas: areasOf(total, plantable, nonPlantable, newPlanting, ratoon),
	}
}

func areasOf(total, plantable, nonPlantable, newPlanting, ratoon float64) domain.Areas {
	return domain.Areas{
		TotalHa:        total,
		PlantableHa:    plantable,
		NonPlantableHa: nonPlantable,
		NewPlantingHa:  newPlanting,
		RatoonHa:       ratoon,
		WithCaneHa:     newPlanting + ratoon,
		AvailableHa:    plantable - newPlanting - ratoon,
	}
}

func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }

func TestBuildTree_sumsBlocksIntoZonesAndFarms(t *testing.T) {
	z1, z1c := ptrInt(11), ptrStr("Z1")
	rows := []treeRow{
		row(1, "F1", z1, z1c, ptrInt(101), ptrStr("B1"), 500, 450, 50, 200, 200),
		row(1, "F1", z1, z1c, ptrInt(102), ptrStr("B2"), 700, 650, 50, 300, 250),
	}

	tree := buildTree(rows)
	if len(tree) != 1 {
		t.Fatalf("expected one farm, got %d", len(tree))
	}
	farm := tree[0]
	zone := farm.Children[0]

	if farm.Areas.TotalHa != 1200 || zone.Areas.TotalHa != 1200 {
		t.Errorf("total: farm %v zone %v, want 1200 each", farm.Areas.TotalHa, zone.Areas.TotalHa)
	}
	if farm.Areas.NewPlantingHa != 500 || farm.Areas.RatoonHa != 450 {
		t.Errorf("new planting %v ratoon %v, want 500 / 450", farm.Areas.NewPlantingHa, farm.Areas.RatoonHa)
	}
	if farm.Areas.WithCaneHa != 950 {
		t.Errorf("with cane = %v, want 950", farm.Areas.WithCaneHa)
	}
	if farm.Areas.AvailableHa != 150 {
		t.Errorf("available = %v, want 150", farm.Areas.AvailableHa)
	}
	if farm.BlockCount != 2 || zone.BlockCount != 2 {
		t.Errorf("block counts: farm %d zone %d, want 2", farm.BlockCount, zone.BlockCount)
	}
}

// The specification's own example has a zone with no blocks. It is part of the estate's structure
// and has to appear, reporting zeros, rather than vanish from the hierarchy.
func TestBuildTree_keepsZonesWithNoBlocks(t *testing.T) {
	rows := []treeRow{
		row(1, "F1", ptrInt(11), ptrStr("Z1"), ptrInt(101), ptrStr("B1"), 500, 450, 50, 200, 100),
		row(1, "F1", ptrInt(12), ptrStr("Z2"), nil, nil, 0, 0, 0, 0, 0),
	}

	farm := buildTree(rows)[0]
	if len(farm.Children) != 2 {
		t.Fatalf("expected both zones, got %d", len(farm.Children))
	}
	empty := farm.Children[1]
	if empty.Code != "Z2" || len(empty.Children) != 0 || empty.Areas.TotalHa != 0 {
		t.Errorf("empty zone = %+v", empty)
	}
	if farm.Areas.TotalHa != 500 {
		t.Errorf("an empty zone must not change the farm total: %v", farm.Areas.TotalHa)
	}
}

// A farm with no zones at all comes back as a single row rather than being dropped.
func TestBuildTree_keepsFarmsWithNoZones(t *testing.T) {
	tree := buildTree([]treeRow{row(2, "F2", nil, nil, nil, nil, 0, 0, 0, 0, 0)})
	if len(tree) != 1 || len(tree[0].Children) != 0 {
		t.Fatalf("expected one childless farm, got %+v", tree)
	}
}

func TestBuildTree_percentagesAreOfTheNodesOwnTotal(t *testing.T) {
	rows := []treeRow{
		row(1, "F1", ptrInt(11), ptrStr("Z1"), ptrInt(101), ptrStr("B1"), 1000, 900, 100, 400, 200),
	}
	farm := buildTree(rows)[0]

	if farm.PercentWithCane != 60 {
		t.Errorf("percent with cane = %v, want 60", farm.PercentWithCane)
	}
	if farm.PercentAvailable != 30 {
		t.Errorf("percent available = %v, want 30", farm.PercentAvailable)
	}
	if farm.PercentCannotPlan != 10 {
		t.Errorf("percent cannot plant = %v, want 10", farm.PercentCannotPlan)
	}
}

func TestBuildTree_returnsAnEmptySliceRatherThanNil(t *testing.T) {
	// The UI5 model binds to an array; a JSON null would leave the table stuck on "no data" with
	// no rows aggregate at all.
	if tree := buildTree(nil); tree == nil || len(tree) != 0 {
		t.Fatalf("expected an empty slice, got %v", tree)
	}
}
