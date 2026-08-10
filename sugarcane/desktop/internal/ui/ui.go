// Package ui is the desktop window: a Win32 form built with lxn/walk, over the api package.
//
// Everything Windows-specific is in files behind a //go:build windows tag, and everything worth
// testing — the client, the geometry, the map renderer, the formatting below — is outside them. The
// window is assembly: it arranges controls and calls the client, and holds no rule of its own.
package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/api"
)

// Options is what the window needs to start.
type Options struct {
	// BaseURL is the service the sign-in box opens with.
	BaseURL string
	// UserName pre-fills the sign-in box, for a shortcut during development.
	UserName string
	// Password signs in without showing the box at all. Only ever set from a command-line flag on
	// a developer's own machine; there is no setting for it and nothing writes it down.
	Password string
}

// ---------------------------------------------------------------- formatting
//
// The window shows figures in one style throughout: hectares to one decimal with thousands
// separated, and a percentage to one. These are here rather than inline so the same number never
// appears two ways on one screen.

// Hectares formats an area the way every figure in the window is written.
func Hectares(v float64) string { return Thousands(v, 1) }

// Percent formats a share.
func Percent(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + "%"
}

// Thousands formats a number to the given number of decimals with thousands separated by commas.
func Thousands(v float64, decimals int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	text := strconv.FormatFloat(v, 'f', decimals, 64)

	sign := ""
	if strings.HasPrefix(text, "-") {
		sign, text = "-", text[1:]
	}
	whole, fraction, _ := strings.Cut(text, ".")

	var out strings.Builder
	for i, digit := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(digit)
	}
	if fraction != "" {
		out.WriteByte('.')
		out.WriteString(fraction)
	}
	return sign + out.String()
}

// FilterSummary is the line the status bar shows, so a figure is never read without knowing what
// land it describes.
func FilterSummary(f api.Filter) string {
	switch n := f.Applied(); n {
	case 0:
		return "No filter applied — the whole estate"
	case 1:
		return "1 filter applied"
	default:
		return fmt.Sprintf("%d filters applied", n)
	}
}

// TreeSummary is what the hierarchy panel reports beneath itself.
func TreeSummary(tree []*api.TreeNode, blockCount int, totalHa float64) string {
	zones := 0
	for _, farm := range tree {
		zones += len(farm.Children)
	}
	return fmt.Sprintf("%d farm(s) · %d zone(s) · %d block(s) · %s ha total",
		len(tree), zones, blockCount, Hectares(totalHa))
}

// LocationDetail is what the panel beside the map says about the location last selected. A block
// carries its own status, variety and dates; a farm or a zone has none of those — it has a count of
// blocks and how much of it is planted, which is the question a whole location answers.
func LocationDetail(p api.FeatureProperties) string {
	if p.Code == "" {
		return "Select a location on the map to see its detail."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\r\n\r\n", p.Code, p.Name)

	line := func(label, value string) {
		if value != "" {
			fmt.Fprintf(&b, "%-22s %s\r\n", label, value)
		}
	}

	if p.Level == api.LevelBlock {
		line("Farm", p.FarmName)
		line("Zone", p.ZoneName)
		line("Cane status", p.CaneStatus)
		line("Land status", p.LandStatus)
	} else {
		line("Blocks", strconv.Itoa(p.BlockCount))
		line("Share planted", Percent(p.PlantedPercent))
	}

	line("Total area", Hectares(p.TotalAreaHa)+" ha")
	line("New planting", Hectares(p.NewPlantingAreaHa)+" ha")
	line("Ratoon", Hectares(p.RatoonAreaHa)+" ha")
	line("Area with cane", Hectares(p.AreaWithCaneHa)+" ha")
	line("Available", Hectares(p.AvailableAreaHa)+" ha")
	line("Cannot be planted", Hectares(p.NonPlantableAreaHa)+" ha")

	if p.Level == api.LevelBlock {
		line("Variety", p.CaneVarietyName)
		line("Planted on", p.PlantingDate)
		line("Expected harvest", p.ExpectedHarvestDate)
	}
	if p.Latitude != nil && p.Longitude != nil {
		line("Coordinates", fmt.Sprintf("%.5f, %.5f", *p.Latitude, *p.Longitude))
	}
	return b.String()
}

// FlattenTree lays the hierarchy out as rows, depth first, so it can be shown in a table with a
// column for each figure. walk's tree control carries no columns, and the figures beside each
// level are the point of the report.
type Row struct {
	Depth int
	Node  *api.TreeNode
}

// Label is the first column: the code and name, indented to its level.
func (r Row) Label() string {
	return strings.Repeat("    ", r.Depth) + r.Node.Code + " — " + r.Node.Name
}

// FlattenTree walks the hierarchy into rows.
func FlattenTree(nodes []*api.TreeNode) []Row {
	var rows []Row
	var walk func(nodes []*api.TreeNode, depth int)
	walk = func(nodes []*api.TreeNode, depth int) {
		for _, node := range nodes {
			rows = append(rows, Row{Depth: depth, Node: node})
			walk(node.Children, depth+1)
		}
	}
	walk(nodes, 0)
	return rows
}
