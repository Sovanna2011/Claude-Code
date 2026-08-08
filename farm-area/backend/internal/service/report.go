package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/sovanna2011/farm-area/backend/internal/domain"
)

// ReportService renders the tree report as a workbook. It reads through the dashboard service
// rather than issuing its own queries, so the exported file and the screen can never disagree —
// the export is the same data, in a different container.
type ReportService struct {
	dashboard *DashboardService
}

const sheetName = "Farm area"

func (s *ReportService) TreeWorkbook(ctx context.Context, f domain.Filter) ([]byte, error) {
	tree, err := s.dashboard.Tree(ctx, f)
	if err != nil {
		return nil, err
	}
	kpis, err := s.dashboard.KPIs(ctx, f)
	if err != nil {
		return nil, err
	}

	file := excelize.NewFile()
	defer file.Close()
	index, err := file.NewSheet(sheetName)
	if err != nil {
		return nil, err
	}
	file.SetActiveSheet(index)
	file.DeleteSheet("Sheet1")

	header, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"354A5F"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	number, err := file.NewStyle(&excelize.Style{CustomNumFmt: strPtr("#,##0.00")})
	if err != nil {
		return nil, err
	}
	percent, err := file.NewStyle(&excelize.Style{CustomNumFmt: strPtr(`0.0"%"`)})
	if err != nil {
		return nil, err
	}
	farmStyle, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EAF0F5"}},
	})
	if err != nil {
		return nil, err
	}

	columns := []struct {
		title string
		width float64
	}{
		{"Level", 10}, {"Code", 16}, {"Name", 34},
		{"Total area (ha)", 16}, {"Plantable (ha)", 16},
		{"New planting (ha)", 18}, {"Ratoon (ha)", 14},
		{"Area with cane (ha)", 19}, {"Available (ha)", 16}, {"Cannot plant (ha)", 18},
		{"% with cane", 13}, {"% available", 13}, {"% cannot plant", 15},
		{"Land status", 14}, {"Cane status", 14}, {"Variety", 20}, {"Google Maps", 46},
	}
	for i, c := range columns {
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = file.SetColWidth(sheetName, col, col, c.width)
		cell := fmt.Sprintf("%s1", col)
		_ = file.SetCellStr(sheetName, cell, c.title)
		_ = file.SetCellStyle(sheetName, cell, cell, header)
	}
	_ = file.SetRowHeight(sheetName, 1, 28)
	if err := file.AutoFilter(sheetName, "A1:Q1", nil); err != nil {
		return nil, err
	}
	// The header stays visible while a reader scrolls a hundred blocks.
	if err := file.SetPanes(sheetName, &excelize.Panes{
		Freeze: true, Split: false, XSplit: 3, YSplit: 1, TopLeftCell: "D2", ActivePane: "bottomRight",
	}); err != nil {
		return nil, err
	}

	row := 2
	var write func(nodes []*domain.TreeNode, depth int) error
	write = func(nodes []*domain.TreeNode, depth int) error {
		for _, n := range nodes {
			indent := ""
			for i := 0; i < depth; i++ {
				indent += "    "
			}
			values := []any{
				n.NodeType, indent + n.Code, n.Name,
				n.Areas.TotalHa, n.Areas.PlantableHa, n.Areas.NewPlantingHa, n.Areas.RatoonHa,
				n.Areas.WithCaneHa, n.Areas.AvailableHa, n.Areas.NonPlantableHa,
				n.PercentWithCane, n.PercentAvailable, n.PercentCannotPlan,
				n.LandStatus, n.CaneStatus, n.VarietyName, n.MapURL,
			}
			for i, v := range values {
				col, _ := excelize.ColumnNumberToName(i + 1)
				cell := fmt.Sprintf("%s%d", col, row)
				if err := file.SetCellValue(sheetName, cell, v); err != nil {
					return err
				}
			}
			_ = file.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("J%d", row), number)
			_ = file.SetCellStyle(sheetName, fmt.Sprintf("K%d", row), fmt.Sprintf("M%d", row), percent)
			if depth == 0 {
				_ = file.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("Q%d", row), farmStyle)
			}
			if n.MapURL != "" {
				display, tooltip := "Open in Google Maps", "Open this "+n.NodeType+" in Google Maps"
				_ = file.SetCellHyperLink(sheetName, fmt.Sprintf("Q%d", row), n.MapURL, "External",
					excelize.HyperlinkOpts{Display: &display, Tooltip: &tooltip})
			}
			// Excel's own outline, so the workbook expands and collapses like the screen does.
			if depth > 0 {
				_ = file.SetRowOutlineLevel(sheetName, row, uint8(depth))
			}
			row++
			if err := write(n.Children, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := write(tree, 0); err != nil {
		return nil, err
	}

	// The estate total goes at the bottom, from the same KPI figures the cards show.
	totals := []any{"Estate", "", fmt.Sprintf("Total — %d block(s)", kpis.BlockCount),
		kpis.Areas.TotalHa, kpis.Areas.PlantableHa, kpis.Areas.NewPlantingHa, kpis.Areas.RatoonHa,
		kpis.Areas.WithCaneHa, kpis.Areas.AvailableHa, kpis.Areas.NonPlantableHa,
		domain.PercentOfTotal(kpis.Areas.WithCaneHa, kpis.Areas.TotalHa),
		domain.PercentOfTotal(kpis.Areas.AvailableHa, kpis.Areas.TotalHa),
		domain.PercentOfTotal(kpis.Areas.NonPlantableHa, kpis.Areas.TotalHa),
	}
	for i, v := range totals {
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = file.SetCellValue(sheetName, fmt.Sprintf("%s%d", col, row), v)
	}
	_ = file.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("Q%d", row), header)
	_ = file.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("J%d", row), number)

	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, fmt.Errorf("write workbook: %w", err)
	}
	return buf.Bytes(), nil
}

func strPtr(s string) *string { return &s }
