//go:build windows

package ui

import (
	"context"
	"os/exec"
	"strconv"

	"github.com/lxn/walk"
	d "github.com/lxn/walk/declarative"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/api"
	"github.com/sovanna2011/sugarcane-go/desktop/internal/mapimage"
)

// mainForm is the window. It owns the controls and the last answer from the service, and nothing
// else: every rule about what the figures mean lives in the service, and every rule about how the
// map looks lives in the mapimage package.
type mainForm struct {
	*walk.MainWindow
	client *api.Client

	// the filter bar
	cropYearBox, farmBox, zoneBox, blockBox *walk.ComboBox
	searchBox                               *walk.LineEdit
	cropYears, farms, zones, blocks         []api.LookupItem

	// the cards
	cardValues []*walk.Label
	cardShares []*walk.Label

	// the hierarchy
	rows      *hierarchyModel
	table     *walk.TableView
	treeLabel *walk.Label

	// the map
	mapLevel     string
	levelButtons map[string]*walk.PushButton
	mapWidget    *walk.CustomWidget
	legendWidget *walk.CustomWidget
	mapData      api.MapData
	rendered     *mapimage.Rendered
	bitmap       *walk.Bitmap
	selected     api.FeatureProperties
	detailBox    *walk.TextEdit
	openMaps     *walk.PushButton

	statusFilter *walk.StatusBarItem
	statusUser   *walk.StatusBarItem

	// busy guards against a second refresh starting while the first is in flight, which would have
	// two answers race to fill the same controls.
	busy bool
}

// cardOrder is the six figures across the top, in the order the service returns them.
var cardOrder = []string{"total", "newPlanting", "ratoon", "withCane", "available", "cannotPlant"}

func showMain(client *api.Client) error {
	f := &mainForm{
		client:       client,
		mapLevel:     api.LevelBlock,
		levelButtons: map[string]*walk.PushButton{},
		rows:         &hierarchyModel{},
	}

	cards := make([]d.Widget, 0, len(cardOrder))
	f.cardValues = make([]*walk.Label, len(cardOrder))
	f.cardShares = make([]*walk.Label, len(cardOrder))
	for i := range cardOrder {
		cards = append(cards, d.GroupBox{
			Title:  " ",
			Layout: d.VBox{MarginsZero: true},
			Children: []d.Widget{
				d.Label{AssignTo: &f.cardValues[i], Text: "—", Font: d.Font{PointSize: 14, Bold: true}},
				d.Label{AssignTo: &f.cardShares[i], Text: "", TextColor: walk.RGB(0x55, 0x6B, 0x82)},
			},
		})
	}

	// The three buttons that choose which location level the map draws. AssignTo is filled in when
	// the window is created, so the pointers are collected into a slice now and put in the map by
	// level once Create has returned — see below.
	buttons := make([]*walk.PushButton, len(api.MapLevels))
	levelButtons := make([]d.Widget, 0, len(api.MapLevels))
	for i, level := range api.MapLevels {
		level := level
		levelButtons = append(levelButtons, d.PushButton{
			AssignTo:  &buttons[i],
			Text:      level,
			MaxSize:   d.Size{Width: 90},
			OnClicked: func() { f.setLevel(level) },
		})
	}

	err := d.MainWindow{
		AssignTo: &f.MainWindow,
		Title:    "Farm Area Monitoring — Sugarcane Planting Planning",
		MinSize:  d.Size{Width: 1100, Height: 720},
		Layout:   d.VBox{},
		MenuItems: []d.MenuItem{
			d.Menu{
				Text: "&File",
				Items: []d.MenuItem{
					d.Action{Text: "&Refresh\tF5", Shortcut: d.Shortcut{Key: walk.KeyF5},
						OnTriggered: func() { f.reload() }},
					d.Separator{},
					d.Action{Text: "&Sign out", OnTriggered: f.signOut},
					d.Action{Text: "E&xit", OnTriggered: func() { f.Close() }},
				},
			},
			d.Menu{
				Text: "&Help",
				Items: []d.MenuItem{
					d.Action{Text: "&About", OnTriggered: f.about},
				},
			},
		},
		StatusBarItems: []d.StatusBarItem{
			{AssignTo: &f.statusFilter, Text: "", Width: 420},
			{AssignTo: &f.statusUser, Text: ""},
		},
		Children: []d.Widget{
			// ---------------------------------------------------------------- filters
			d.GroupBox{
				Title:  "Filters",
				Layout: d.Grid{Columns: 8},
				Children: []d.Widget{
					d.Label{Text: "Crop year"},
					d.ComboBox{AssignTo: &f.cropYearBox, MinSize: d.Size{Width: 110},
						OnCurrentIndexChanged: func() { f.reload() }},

					d.Label{Text: "Farm"},
					d.ComboBox{AssignTo: &f.farmBox, MinSize: d.Size{Width: 180},
						OnCurrentIndexChanged: f.farmChanged},

					d.Label{Text: "Zone"},
					d.ComboBox{AssignTo: &f.zoneBox, MinSize: d.Size{Width: 200},
						OnCurrentIndexChanged: f.zoneChanged},

					d.Label{Text: "Block"},
					d.ComboBox{AssignTo: &f.blockBox, MinSize: d.Size{Width: 220},
						OnCurrentIndexChanged: func() { f.reload() }},

					d.Label{Text: "Search"},
					d.LineEdit{AssignTo: &f.searchBox, ColumnSpan: 3,
						CueBanner: "farm, zone or block", OnEditingFinished: func() { f.reload() }},

					d.PushButton{Text: "Refresh", OnClicked: func() { f.reload() }},
					d.PushButton{Text: "Clear filters", OnClicked: f.clearFilters},
					d.HSpacer{ColumnSpan: 2},
				},
			},

			// ---------------------------------------------------------------- cards
			d.Composite{Layout: d.HBox{MarginsZero: true}, Children: cards},

			// ---------------------------------------------------------------- hierarchy and map
			d.HSplitter{
				Children: []d.Widget{
					d.Composite{
						Layout: d.VBox{},
						Children: []d.Widget{
							d.Label{Text: "Farm → zone → block", Font: d.Font{Bold: true}},
							d.TableView{
								AssignTo:              &f.table,
								Model:                 f.rows,
								AlternatingRowBG:      true,
								ColumnsOrderable:      false,
								MultiSelection:        false,
								OnCurrentIndexChanged: func() { f.rowSelected() },
								Columns: []d.TableViewColumn{
									{Title: "Hierarchy", Width: 260},
									{Title: "Total (ha)", Width: 90, Alignment: d.AlignFar},
									{Title: "New planting", Width: 100, Alignment: d.AlignFar},
									{Title: "Ratoon", Width: 90, Alignment: d.AlignFar},
									{Title: "With cane", Width: 90, Alignment: d.AlignFar},
									{Title: "Available", Width: 90, Alignment: d.AlignFar},
									{Title: "Cannot plant", Width: 100, Alignment: d.AlignFar},
									{Title: "Blocks", Width: 60, Alignment: d.AlignFar},
								},
							},
							d.Label{AssignTo: &f.treeLabel, Text: "", TextColor: walk.RGB(0x55, 0x6B, 0x82)},
						},
					},
					d.Composite{
						Layout: d.VBox{},
						Children: []d.Widget{
							d.Composite{
								Layout: d.HBox{MarginsZero: true},
								Children: append(
									append([]d.Widget{d.Label{Text: "Draw by"}}, levelButtons...),
									d.HSpacer{},
									d.PushButton{AssignTo: &f.openMaps, Text: "Open in Google Maps",
										Enabled: false, OnClicked: f.openInGoogleMaps},
								),
							},
							d.CustomWidget{
								AssignTo:            &f.mapWidget,
								ClearsBackground:    true,
								InvalidatesOnResize: true,
								Paint:               f.paintMap,
								OnMouseDown:         f.mapClicked,
								MinSize:             d.Size{Width: 360, Height: 300},
							},
							d.CustomWidget{
								AssignTo:            &f.legendWidget,
								ClearsBackground:    true,
								InvalidatesOnResize: true,
								Paint:               f.paintLegend,
								MinSize:             d.Size{Height: 24},
								MaxSize:             d.Size{Height: 24},
							},
							d.TextEdit{AssignTo: &f.detailBox, ReadOnly: true, VScroll: true,
								MinSize: d.Size{Height: 220},
								Font:    d.Font{Family: "Consolas", PointSize: 9}},
						},
					},
				},
			},
		},
	}.Create()
	if err != nil {
		return err
	}

	for i, level := range api.MapLevels {
		f.levelButtons[level] = buttons[i]
	}

	f.markLevel()
	f.detailBox.SetText(LocationDetail(api.FeatureProperties{}))
	f.statusUser.SetText(client.User().FullName + " · " + client.User().Role + " · " + client.BaseURL())

	// The dropdowns are filled before the first refresh, so the crop year the estate is being
	// managed in is chosen rather than every year at once.
	f.loadLookups()

	f.Run()
	f.disposeBitmap()
	return nil
}

// ---------------------------------------------------------------- the filter

func (f *mainForm) filter() api.Filter {
	return api.Filter{
		CropYear: intOrNil(f.cropYearBox.Text()),
		FarmID:   selectedID(f.farmBox, f.farms),
		ZoneID:   selectedID(f.zoneBox, f.zones),
		BlockID:  selectedID(f.blockBox, f.blocks),
		Search:   f.searchBox.Text(),
	}
}

// loadLookups fills the four dropdowns and then refreshes. Choosing a farm narrows the zones and
// choosing a zone narrows the blocks, for the same reason the browser client does it: a filter bar
// offering another farm's zones invites an empty report.
func (f *mainForm) loadLookups() {
	go func() {
		years, yearsErr := call(func(ctx context.Context) ([]api.LookupItem, error) {
			return f.client.Lookup(ctx, "cropYears", nil)
		})
		farms, farmsErr := call(func(ctx context.Context) ([]api.LookupItem, error) {
			return f.client.Lookup(ctx, "farms", nil)
		})
		zones, zonesErr := call(func(ctx context.Context) ([]api.LookupItem, error) {
			return f.client.Lookup(ctx, "zones", nil)
		})
		blocks, blocksErr := call(func(ctx context.Context) ([]api.LookupItem, error) {
			return f.client.Lookup(ctx, "blocks", nil)
		})

		f.Synchronize(func() {
			for _, err := range []error{yearsErr, farmsErr, zonesErr, blocksErr} {
				if err != nil {
					report(f, "Reference data", err)
					return
				}
			}

			f.cropYears = years
			yearLabels := make([]string, 0, len(years)+1)
			yearLabels = append(yearLabels, "")
			for _, year := range years {
				yearLabels = append(yearLabels, strconv.Itoa(year.ID))
			}
			_ = f.cropYearBox.SetModel(yearLabels)

			f.setLookup(f.farmBox, &f.farms, farms)
			f.setLookup(f.zoneBox, &f.zones, zones)
			f.setLookup(f.blockBox, &f.blocks, blocks)

			// The most recent crop year rather than every year at once: a plantation's dashboard is
			// about the season being managed.
			if len(yearLabels) > 1 {
				_ = f.cropYearBox.SetCurrentIndex(1)
			} else {
				f.reload()
			}
		})
	}()
}

func (f *mainForm) setLookup(box *walk.ComboBox, into *[]api.LookupItem, items []api.LookupItem) {
	labels, values := blankFirst(items)
	*into = values
	_ = box.SetModel(labels)
	_ = box.SetCurrentIndex(0)
}

func (f *mainForm) farmChanged() {
	farmID := selectedID(f.farmBox, f.farms)
	go func() {
		zones, err := call(func(ctx context.Context) ([]api.LookupItem, error) {
			return f.client.Lookup(ctx, "zones", farmID)
		})
		blocks, blocksErr := call(func(ctx context.Context) ([]api.LookupItem, error) {
			return f.client.Lookup(ctx, "blocks", nil)
		})
		f.Synchronize(func() {
			if err != nil || blocksErr != nil {
				report(f, "Zones", cmp(err, blocksErr))
				return
			}
			f.setLookup(f.zoneBox, &f.zones, zones)
			f.setLookup(f.blockBox, &f.blocks, blocks)
			f.reload()
		})
	}()
}

func (f *mainForm) zoneChanged() {
	zoneID := selectedID(f.zoneBox, f.zones)
	go func() {
		blocks, err := call(func(ctx context.Context) ([]api.LookupItem, error) {
			return f.client.Lookup(ctx, "blocks", zoneID)
		})
		f.Synchronize(func() {
			if err != nil {
				report(f, "Blocks", err)
				return
			}
			f.setLookup(f.blockBox, &f.blocks, blocks)
			f.reload()
		})
	}()
}

func (f *mainForm) clearFilters() {
	_ = f.cropYearBox.SetCurrentIndex(0)
	_ = f.farmBox.SetCurrentIndex(0)
	_ = f.zoneBox.SetCurrentIndex(0)
	_ = f.blockBox.SetCurrentIndex(0)
	_ = f.searchBox.SetText("")
	f.selected = api.FeatureProperties{}
	f.reload()
}

// ---------------------------------------------------------------- refreshing

// reload fetches the cards, the hierarchy and the map with one filter, in one pass, so they cannot
// end up describing different land.
func (f *mainForm) reload() {
	if f.busy || f.client == nil || !f.client.SignedIn() {
		return
	}
	f.busy = true

	filter := f.filter()
	level := f.mapLevel
	f.statusFilter.SetText(FilterSummary(filter) + " — loading…")

	go func() {
		board, boardErr := call(func(ctx context.Context) (api.Dashboard, error) {
			return f.client.Dashboard(ctx, filter)
		})
		tree, treeErr := call(func(ctx context.Context) ([]*api.TreeNode, error) {
			return f.client.Tree(ctx, filter)
		})
		drawn, mapErr := call(func(ctx context.Context) (api.MapData, error) {
			return f.client.Map(ctx, filter, level)
		})

		f.Synchronize(func() {
			f.busy = false
			f.statusFilter.SetText(FilterSummary(filter))

			if err := cmp(boardErr, treeErr, mapErr); err != nil {
				if api.IsUnauthorized(err) {
					walk.MsgBox(f, "Session expired",
						"Your session has expired. Sign in again.", walk.MsgBoxIconWarning)
					f.signOut()
					return
				}
				report(f, "Refresh", err)
				return
			}

			f.showCards(board)
			f.rows.setRows(FlattenTree(tree))
			f.treeLabel.SetText(TreeSummary(tree, board.BlockCount, board.Areas.TotalHa))

			f.mapData = drawn
			f.mapLevel = drawn.Level
			f.markLevel()
			f.renderMap()
			f.legendWidget.Invalidate()
		})
	}()
}

func (f *mainForm) showCards(board api.Dashboard) {
	byKey := map[string]api.KPI{}
	for _, kpi := range board.KPIs {
		byKey[kpi.Key] = kpi
	}
	for i, key := range cardOrder {
		kpi, ok := byKey[key]
		if !ok {
			f.cardValues[i].SetText("—")
			f.cardShares[i].SetText("")
			continue
		}
		f.cardValues[i].SetText(Hectares(kpi.AreaHa) + " ha")
		f.cardShares[i].SetText(kpi.Label + " · " + Percent(kpi.PercentTotal) + " of total")
	}
}

// ---------------------------------------------------------------- the hierarchy table

type hierarchyModel struct {
	walk.TableModelBase
	rows []Row
}

func (m *hierarchyModel) RowCount() int { return len(m.rows) }

func (m *hierarchyModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.rows) {
		return ""
	}
	r := m.rows[row]
	switch col {
	case 0:
		return r.Label()
	case 1:
		return Hectares(r.Node.Areas.TotalHa)
	case 2:
		return Hectares(r.Node.Areas.NewPlantingHa)
	case 3:
		return Hectares(r.Node.Areas.RatoonHa)
	case 4:
		return Hectares(r.Node.Areas.WithCaneHa)
	case 5:
		return Hectares(r.Node.Areas.AvailableHa)
	case 6:
		return Hectares(r.Node.Areas.NonPlantableHa)
	case 7:
		return r.Node.BlockCount
	}
	return ""
}

func (m *hierarchyModel) setRows(rows []Row) {
	m.rows = rows
	m.PublishRowsReset()
}

// rowSelected keeps the map and the table in step: choosing a row highlights that location when the
// map is drawing that level, and fills the detail panel either way.
func (f *mainForm) rowSelected() {
	index := f.table.CurrentIndex()
	if index < 0 || index >= len(f.rows.rows) {
		return
	}
	node := f.rows.rows[index].Node

	for _, shape := range f.shapes() {
		if shape.Properties.Level == node.NodeType && shape.Properties.ID == node.ID {
			f.selectFeature(shape.Properties)
			return
		}
	}

	// The map is drawing another level, so it has nothing to say about this row; the panel still
	// describes it from what the hierarchy knows.
	f.selectFeature(api.FeatureProperties{
		Level: node.NodeType, ID: node.ID, Code: node.Code, Name: node.Name,
		TotalAreaHa: node.Areas.TotalHa, NewPlantingAreaHa: node.Areas.NewPlantingHa,
		RatoonAreaHa: node.Areas.RatoonHa, AreaWithCaneHa: node.Areas.WithCaneHa,
		AvailableAreaHa: node.Areas.AvailableHa, NonPlantableAreaHa: node.Areas.NonPlantableHa,
		PlantedPercent: node.PercentWithCane, BlockCount: node.BlockCount,
		CaneStatus: node.CaneStatus, LandStatus: node.LandStatus,
		CaneVarietyName: node.CaneVarietyName,
		Latitude:        node.Latitude, Longitude: node.Longitude, MapURL: node.MapURL,
	})
}

// ---------------------------------------------------------------- selection

func (f *mainForm) selectFeature(p api.FeatureProperties) {
	f.selected = p
	f.detailBox.SetText(LocationDetail(p))
	f.openMaps.SetEnabled(p.MapURL != "")
	f.renderMap()
}

func (f *mainForm) openInGoogleMaps() {
	if f.selected.MapURL == "" {
		return
	}
	// rundll32 rather than "start": it needs no shell, so nothing in the URL can be read as a
	// command, and the URL the service built is the only thing handed over.
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", f.selected.MapURL)
	if err := cmd.Start(); err != nil {
		report(f, "Open in Google Maps", err)
	}
}

func (f *mainForm) signOut() {
	f.client.SignOut()
	f.Close()
}

func (f *mainForm) about() {
	walk.MsgBox(f,
		"About",
		"Farm Area Monitoring\r\n\r\n"+
			"The Windows client of the Sugarcane Planting Planning system.\r\n"+
			"Every figure it shows is calculated by the service from block-level data;\r\n"+
			"nothing is entered or worked out here.\r\n\r\n"+
			"Service: "+f.client.BaseURL(),
		walk.MsgBoxIconInformation)
}

// cmp returns the first error that is not nil, so several calls can be checked in one line.
func cmp(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
