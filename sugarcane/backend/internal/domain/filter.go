package domain

import (
	"strconv"
	"strings"
)

// Filter is the dashboard filter bar, in one struct. Every read path takes the same value, which
// is what makes the KPI cards, the charts, the tree report and the map agree with one another:
// they are not four queries that happen to be filtered alike, they are four queries given the
// same filter.
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
	PlantingType *string

	LandStatus *string
	CaneStatus *string

	ActivityID       *int
	ActivityCategory *string

	// Projections. CurrentOnly hides the superseded versions of a revised plan, which is what a
	// list screen wants by default — the history is reachable from the plan itself.
	ProjectionStatus *string
	CurrentOnly      bool

	// Free text across farm, zone and block code and name.
	Search string

	IncludeInactive bool
}

// Page is the pagination and sorting a list endpoint accepts.
type Page struct {
	Number   int
	Size     int
	SortBy   string
	SortDesc bool
}

const (
	defaultPageSize = 50
	maxPageSize     = 500
)

// Normalise clamps whatever arrived on the query string into something safe to put in a LIMIT.
func (p Page) Normalise() Page {
	if p.Number < 1 {
		p.Number = 1
	}
	switch {
	case p.Size <= 0:
		p.Size = defaultPageSize
	case p.Size > maxPageSize:
		p.Size = maxPageSize
	}
	return p
}

func (p Page) Offset() int { return (p.Number - 1) * p.Size }

// PagedResult is the envelope every list endpoint returns.
type PagedResult[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`
}

func NewPagedResult[T any](items []T, page Page, total int) PagedResult[T] {
	if items == nil {
		items = []T{}
	}
	pages := 0
	if page.Size > 0 {
		pages = (total + page.Size - 1) / page.Size
	}
	return PagedResult[T]{Items: items, Page: page.Number, PageSize: page.Size, TotalCount: total, TotalPages: pages}
}

// ValidPlantingTypes and the status vocabularies, so a filter cannot smuggle an unknown enum value
// into a query and have Postgres reject it with a type error the caller cannot act on.
var (
	ValidPlantingTypes = []string{"NewPlanting", "Ratoon"}
	ValidLandStatuses  = []string{"Active", "Reserved", "Retired"}
	ValidCaneStatuses  = []string{"Fallow", "Prepared", "Planted", "Growing", "ReadyForHarvest", "Harvested"}
)

func IsOneOf(value string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(a, value) {
			return true
		}
	}
	return false
}

// Canonical returns the vocabulary's own spelling of a value, so "ratoon" from a query string
// becomes the "Ratoon" the enum expects.
func Canonical(value string, allowed []string) (string, bool) {
	for _, a := range allowed {
		if strings.EqualFold(a, value) {
			return a, true
		}
	}
	return "", false
}

// GoogleMapsURL is the one place the link format lives. It is a plain search URL, which needs no
// API key and works from any browser — the interactive map in the dashboard is a separate concern.
func GoogleMapsURL(lat, lng *float64) string {
	if lat == nil || lng == nil {
		return ""
	}
	return "https://www.google.com/maps/search/?api=1&query=" +
		trimFloat(*lat) + "," + trimFloat(*lng)
}

func trimFloat(v float64) string {
	s := strings.TrimRight(strings.TrimRight(formatFloat(v), "0"), ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}
