//go:build !windows

package ui

import "errors"

// Run reports that there is no window to open. The rest of the desktop — the client, the geometry,
// the map renderer — builds and is tested on every platform; only the form itself is Win32, so this
// keeps `go build ./...` and `go test ./...` honest on the machine that builds the service.
func Run(Options) error {
	return errors.New("the Farm Area desktop window runs on Windows; " +
		"build it with GOOS=windows go build ./cmd/farmarea-desktop")
}
