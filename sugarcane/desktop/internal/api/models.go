// Package api is the desktop client's whole conversation with the Farm Area service. Nothing else
// in the application speaks HTTP, so the address, the bearer token and the shape of an error are
// each dealt with in one place.
//
// It has no dependency on the window toolkit, which is what lets it be tested on any machine —
// including the Linux one that builds the service — rather than only on Windows.
package api

import (
	"encoding/json"
	"time"
)

// User is who is signed in.
type User struct {
	ID       int    `json:"id"`
	UserName string `json:"userName"`
	FullName string `json:"fullName"`
	Role     string `json:"role"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      User      `json:"user"`
}

// Areas is the six figures every level of the hierarchy reports, in hectares.
type Areas struct {
	TotalHa        float64 `json:"totalAreaHa"`
	PlantableHa    float64 `json:"plantableAreaHa"`
	NonPlantableHa float64 `json:"nonPlantableAreaHa"`
	NewPlantingHa  float64 `json:"newPlantingAreaHa"`
	RatoonHa       float64 `json:"ratoonAreaHa"`
	WithCaneHa     float64 `json:"areaWithCaneHa"`
	AvailableHa    float64 `json:"availableAreaHa"`
}

// KPI is one card at the top of the dashboard.
type KPI struct {
	Key          string  `json:"key"`
	Label        string  `json:"label"`
	AreaHa       float64 `json:"areaHa"`
	PercentTotal float64 `json:"percentOfTotalArea"`
	DrillTo      string  `json:"drillTo"`
}

// Dashboard is what the cards at the top of the window show.
type Dashboard struct {
	Areas      Areas  `json:"areas"`
	KPIs       []KPI  `json:"kpis"`
	BlockCount int    `json:"blockCount"`
	Unit       string `json:"unit"`
}

// TreeNode is one row of the farm → zone → block report. NodeType is Farm, Zone or Block.
type TreeNode struct {
	NodeType           string      `json:"nodeType"`
	ID                 int         `json:"id"`
	Code               string      `json:"code"`
	Name               string      `json:"name"`
	Areas              Areas       `json:"areas"`
	PercentWithCane    float64     `json:"percentWithCane"`
	PercentAvailable   float64     `json:"percentAvailable"`
	PercentCannotPlant float64     `json:"percentCannotPlant"`
	Latitude           *float64    `json:"latitude"`
	Longitude          *float64    `json:"longitude"`
	MapURL             string      `json:"mapUrl"`
	BlockCount         int         `json:"blockCount"`
	LandStatus         string      `json:"landStatus"`
	CaneStatus         string      `json:"caneStatus"`
	CaneVarietyName    string      `json:"caneVarietyName"`
	Children           []*TreeNode `json:"children"`
}

// LookupItem is one entry of a filter dropdown.
type LookupItem struct {
	ID       int    `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	ParentID *int   `json:"parentId"`
	Display  string `json:"display"`
}

// Label is what a combo box shows. A lookup always carries a display string from the service; the
// fallback is only for the blank entry the filter bar adds itself so a choice can be cleared.
func (l LookupItem) Label() string {
	if l.Display != "" {
		return l.Display
	}
	return l.Name
}

// ---------------------------------------------------------------- the map

// MapLevel is which location level the map draws. The strings are the service's own vocabulary.
const (
	LevelFarm  = "Farm"
	LevelZone  = "Zone"
	LevelBlock = "Block"
)

// MapLevels is the order the level buttons appear in, coarsest first.
var MapLevels = []string{LevelFarm, LevelZone, LevelBlock}

// FeatureProperties is what the service hangs on a map feature. The first four are on every feature
// at every level; the rest belong to one level or the other and are zero elsewhere — a farm has no
// cane status, a block has no block count.
type FeatureProperties struct {
	Level string `json:"level"`
	ID    int    `json:"id"`
	Code  string `json:"code"`
	Name  string `json:"name"`

	TotalAreaHa        float64 `json:"totalAreaHa"`
	PlantableAreaHa    float64 `json:"plantableAreaHa"`
	NonPlantableAreaHa float64 `json:"nonPlantableAreaHa"`
	NewPlantingAreaHa  float64 `json:"newPlantingAreaHa"`
	RatoonAreaHa       float64 `json:"ratoonAreaHa"`
	AreaWithCaneHa     float64 `json:"areaWithCaneHa"`
	AvailableAreaHa    float64 `json:"availableAreaHa"`
	PlantedPercent     float64 `json:"plantedPercent"`

	BlockCount          int      `json:"blockCount"`
	CaneStatus          string   `json:"caneStatus"`
	LandStatus          string   `json:"landStatus"`
	FarmName            string   `json:"farmName"`
	ZoneName            string   `json:"zoneName"`
	CaneVarietyName     string   `json:"caneVarietyName"`
	PlantingDate        string   `json:"plantingDate"`
	ExpectedHarvestDate string   `json:"expectedHarvestDate"`
	Latitude            *float64 `json:"latitude"`
	Longitude           *float64 `json:"longitude"`
	MapURL              string   `json:"mapUrl"`
}

// OutlineProperties is the registered boundary of a farm or a zone, drawn under the filled features.
type OutlineProperties struct {
	Level string `json:"level"`
	ID    int    `json:"id"`
	Code  string `json:"code"`
	Name  string `json:"name"`
}

// Feature is one GeoJSON feature. The geometry stays raw: a Point, a Polygon and a MultiPolygon
// nest their coordinates to different depths, and geo.Rings is the one place that is dealt with.
type Feature[P any] struct {
	Geometry   json.RawMessage `json:"geometry"`
	Properties P               `json:"properties"`
}

// FeatureCollection is a GeoJSON collection of one property shape.
type FeatureCollection[P any] struct {
	Type     string       `json:"type"`
	Features []Feature[P] `json:"features"`
}

// MapData draws the whole map in one request. The level travels back with it because the caller may
// have asked for a spelling the service did not accept, and the fills and the legend have to
// describe the features actually received rather than the ones that were asked for.
type MapData struct {
	Level    string                               `json:"level"`
	Features FeatureCollection[FeatureProperties] `json:"features"`
	Outlines FeatureCollection[OutlineProperties] `json:"outlines"`
}

// ---------------------------------------------------------------- projections

// Page is one page of a list, with the count the pager needs.
type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`
}

// Projection is one row of the planting-projection list.
type Projection struct {
	ID                          int       `json:"id"`
	ProjectionNo                string    `json:"projectionNo"`
	Description                 string    `json:"description"`
	Status                      string    `json:"status"`
	CropYear                    int       `json:"cropYear"`
	SeasonID                    int       `json:"seasonId"`
	SeasonName                  string    `json:"seasonName"`
	Version                     int       `json:"version"`
	TotalProjectedAreaHa        float64   `json:"totalProjectedAreaHa"`
	TotalExpectedProductionTons float64   `json:"totalExpectedProductionTons"`
	LineCount                   int       `json:"lineCount"`
	Actions                     []string  `json:"actions"`
	CreatedBy                   string    `json:"createdBy"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedBy                   string    `json:"updatedBy"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}
