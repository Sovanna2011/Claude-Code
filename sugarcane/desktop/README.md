# Farm Area Monitoring — the Windows client

A native Windows desktop application, written in Go, over the same REST service the browser client
uses. It is a **window form** in the Win32 sense: real menus, a real status bar, real list and
combo controls, drawn by Windows itself through [lxn/walk](https://github.com/lxn/walk).

It calculates nothing. Every figure it shows comes from the service, which is the only thing that
reads the database — so the desktop, the browser and `curl` cannot disagree about how much land is
under cane.

```
desktop/
├── cmd/farmarea-desktop/
│   ├── main.go                        flags, then hand over to the window
│   └── farmarea-desktop.exe.manifest  copy beside the .exe — see below
├── internal/
│   ├── api/        the whole conversation with the service: client, models, filter
│   ├── geo/        GeoJSON to rings, rings to pixels, pixels back to a place
│   ├── mapimage/   the map, drawn into an ordinary picture
│   └── ui/         the window — only these files are Windows-only
└── go.mod
```

## Building

The window is Win32, but nothing about **building** it needs Windows. Cross-compile from Linux,
macOS or Windows alike, with no C compiler:

```bash
cd sugarcane/desktop
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags "-H=windowsgui -s -w" -o farmarea-desktop.exe ./cmd/farmarea-desktop
```

`-H=windowsgui` is what stops a black console window opening behind the form. `-s -w` drops the
symbol table, which takes the executable from about 15 MB to about 10.

Copy `farmarea-desktop.exe.manifest` next to the executable, keeping that exact name. Without it the
window still opens, but with the grey controls of comctl32 version 5 and no idea of the monitor's
scaling. The file itself explains how to compile it into the executable instead.

## Running

```
farmarea-desktop.exe                                   sign-in box, pointed at 127.0.0.1:8080
farmarea-desktop.exe -server http://estate-01:8080     a service on the network
farmarea-desktop.exe -user planner                     pre-fill the user name
```

`FARMAREA_API_URL` and `FARMAREA_USER` set the same two defaults. There is a `-password` flag for a
developer signing in over and over on their own machine; it is not a deployment setting and the
token it obtains is held in memory and never written down.

Sign in with **admin / Farm#2026**, or `manager`, `planner`, `viewer` — the same accounts as the
browser client, and the same rules about what each may do.

## What the window shows

**Filters** — crop year, farm, zone, block and a search box. Choosing a farm narrows the zone list
and choosing a zone narrows the blocks, so the bar never offers another farm's zones.

**Six cards** — total area, new planting, ratoon, area with cane, available for planting, cannot be
planted, each with its share of the total.

**Farm → zone → block** — the hierarchy as a table, indented, with a column for every figure. walk's
tree control carries no columns, and the figures beside each level are the point of the report.

**The location map** — the estate's own PostGIS boundaries, drawn by farm, by zone or by block. A
block is filled by what is growing on it; a farm or a zone by how much of it is already planted,
because a whole farm has no single status. Clicking a shape selects it, fills the panel beneath and
moves the table to the same location; selecting a row in the table works the other way round.

One filter feeds the cards, the table and the map in a single refresh, so they cannot end up
describing different land.

## Tests

```bash
cd sugarcane/desktop
go test ./...                                  # 36 tests; the service tests skip

export FARMAREA_API_URL="http://127.0.0.1:8080"
go test ./...                                  # 49 tests
```

The tests run on any platform, because everything worth testing is outside the Windows-only files:
the client, the geometry, the map renderer and the window's own formatting. What is left inside them
is assembly — arranging controls and calling the client.

The service tests are the ones that matter. A hand-written stub would only prove the client agrees
with somebody's idea of the JSON, which is exactly the thing that goes wrong; these check it reads
what the Go service actually sends, that the cards add up, that the hierarchy and the map describe
the same land as the cards at all three levels, and that a bad filter value comes back as a code the
window can branch on rather than a 500.

The map has its own way of being checked without a screen. `mapimage.Render` draws into an ordinary
`image.RGBA`, so a test renders the live estate and writes it out:

```bash
FARMAREA_API_URL="http://127.0.0.1:8080" FARMAREA_MAP_OUT=/tmp go test ./internal/mapimage/ -run RealEstate
```

That produces `map-Farm.png`, `map-Zone.png` and `map-Block.png` — the same picture the window
shows, which can be looked at rather than assumed. It has already earned its keep: it caught labels
drawn four times too large for the blocks they named, and a centroid pulled off-centre because a
GeoJSON ring repeats its first point.
