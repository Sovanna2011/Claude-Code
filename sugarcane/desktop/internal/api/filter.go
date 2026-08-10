package api

import (
	"net/url"
	"strconv"
	"strings"
)

// Filter is the one filter every read takes. It is the desktop's copy of the service's own filter,
// so the cards, the hierarchy and the map cannot end up describing different land: one value is
// built from the filter bar and handed to every call of a refresh.
//
// The pointer fields are the ones where "not set" and "nought" are different answers.
type Filter struct {
	CompanyID    *int
	PlantationID *int
	FarmID       *int
	ZoneID       *int
	BlockID      *int
	CropYear     *int
	PlantingYear *int
	SeasonID     *int
	VarietyID    *int

	PlantingType     string
	LandStatus       string
	CaneStatus       string
	ProjectionStatus string
	Search           string

	IncludeInactive bool
}

// Param is one extra query parameter a particular endpoint takes, such as the map's level.
type Param struct {
	Name  string
	Value string
}

// Query renders the filter as a query string, leading "?" included, or "" when nothing is set.
// Values go through url.Values, so a search term containing an ampersand narrows the search rather
// than becoming another parameter.
func (f Filter) Query(extra ...Param) string {
	values := url.Values{}
	for _, p := range f.pairs() {
		values.Set(p.Name, p.Value)
	}
	if f.IncludeInactive {
		values.Set("includeInactive", "true")
	}
	for _, p := range extra {
		if p.Value != "" {
			values.Set(p.Name, p.Value)
		}
	}
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}

// Applied is how many filters narrow the query, which is what the status bar reports.
//
// IncludeInactive is deliberately not counted: "show the retired land too" widens the query rather
// than narrowing it, and counting it would have the status bar say land is being hidden at the
// moment more of it is being shown.
func (f Filter) Applied() int { return len(f.pairs()) }

func (f Filter) pairs() []Param {
	out := make([]Param, 0, 14)
	add := func(name string, v *int) {
		if v != nil {
			out = append(out, Param{name, strconv.Itoa(*v)})
		}
	}
	addText := func(name, v string) {
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, Param{name, s})
		}
	}

	add("companyId", f.CompanyID)
	add("plantationId", f.PlantationID)
	add("farmId", f.FarmID)
	add("zoneId", f.ZoneID)
	add("blockId", f.BlockID)
	add("cropYear", f.CropYear)
	add("plantingYear", f.PlantingYear)
	add("seasonId", f.SeasonID)
	add("varietyId", f.VarietyID)
	addText("plantingType", f.PlantingType)
	addText("landStatus", f.LandStatus)
	addText("caneStatus", f.CaneStatus)
	addText("projectionStatus", f.ProjectionStatus)
	addText("search", f.Search)
	return out
}

// Int is a shorthand for the pointer fields, so a caller can write api.Int(2026) inline.
func Int(v int) *int { return &v }
