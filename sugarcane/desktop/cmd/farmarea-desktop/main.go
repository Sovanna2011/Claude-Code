// Command farmarea-desktop is the Windows client of the Sugarcane Planting Planning system.
//
// It shows the same land the browser client does — the KPI cards, the farm → zone → block report
// and the location map — and calculates none of it: every figure comes from the service, which is
// the only thing that reads the database.
//
// Build it for Windows from any machine:
//
//	GOOS=windows GOARCH=amd64 go build -ldflags -H=windowsgui -o farmarea-desktop.exe ./cmd/farmarea-desktop
//
// The -H=windowsgui flag is what stops a console window opening behind the form.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/ui"
)

func main() {
	var opts ui.Options

	flag.StringVar(&opts.BaseURL, "server", envOr("FARMAREA_API_URL", "http://127.0.0.1:8080"),
		"the Farm Area service to sign in to")
	flag.StringVar(&opts.UserName, "user", os.Getenv("FARMAREA_USER"),
		"pre-fill the user name in the sign-in box")
	flag.StringVar(&opts.Password, "password", "",
		"sign in without showing the box; for a developer's own machine, never a deployment")
	flag.Parse()

	if err := ui.Run(opts); err != nil {
		// A window may never have opened — a service that cannot be reached, a Windows call that
		// failed — so the reason goes to the console as well as being shown, for the case where the
		// programme was started from one.
		fmt.Fprintln(os.Stderr, "farmarea-desktop:", err)
		os.Exit(1)
	}
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
