//go:build windows

package ui

import (
	"context"
	"strconv"
	"time"

	"github.com/lxn/walk"
	d "github.com/lxn/walk/declarative"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/api"
)

// callTimeout bounds every call to the service. A window that has been "Loading…" for a minute
// because a server is wedged is worse than one that says it could not be reached.
const callTimeout = 30 * time.Second

// Run signs the user in and opens the main window. It returns when the window is closed.
func Run(opts Options) error {
	if opts.BaseURL == "" {
		opts.BaseURL = "http://127.0.0.1:8080"
	}

	var client *api.Client

	// A password on the command line signs in without showing the box; anything else asks.
	if opts.Password != "" {
		client = api.New(opts.BaseURL)
		if _, err := call(func(ctx context.Context) (api.User, error) {
			return client.SignIn(ctx, opts.UserName, opts.Password)
		}); err != nil {
			return err
		}
	} else {
		signedIn, err := showLogin(opts)
		if err != nil {
			return err
		}
		if signedIn == nil {
			// The user closed the sign-in box. That is not a failure.
			return nil
		}
		client = signedIn
	}

	return showMain(client)
}

// call runs one request against the service with the standard timeout.
func call[T any](do func(context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return do(ctx)
}

// report puts an error in front of the user in the service's own words. A refusal names the rule
// that answered; anything else says what actually happened rather than "the request failed".
func report(owner walk.Form, title string, err error) {
	if err == nil {
		return
	}
	walk.MsgBox(owner, title, err.Error(), walk.MsgBoxIconError)
}

func intOrNil(text string) *int {
	if text == "" {
		return nil
	}
	v, err := strconv.Atoi(text)
	if err != nil {
		return nil
	}
	return &v
}

// blankFirst is the entry every filter dropdown opens with, so a choice can be cleared as easily as
// it was made.
func blankFirst(items []api.LookupItem) ([]string, []api.LookupItem) {
	labels := make([]string, 0, len(items)+1)
	values := make([]api.LookupItem, 0, len(items)+1)

	labels = append(labels, "")
	values = append(values, api.LookupItem{})
	for _, item := range items {
		labels = append(labels, item.Label())
		values = append(values, item)
	}
	return labels, values
}

// selectedID is the id behind the current entry of a filter dropdown, or nil for the blank one.
func selectedID(box *walk.ComboBox, values []api.LookupItem) *int {
	index := box.CurrentIndex()
	if index <= 0 || index >= len(values) || values[index].ID == 0 {
		return nil
	}
	id := values[index].ID
	return &id
}

// declarative aliases, so the layout below reads as a form rather than as a package path.
type (
	composite  = d.Composite
	hBox       = d.HBox
	vBox       = d.VBox
	grid       = d.Grid
	label      = d.Label
	pushButton = d.PushButton
	comboBox   = d.ComboBox
	lineEdit   = d.LineEdit
	textEdit   = d.TextEdit
	tableView  = d.TableView
	tableCol   = d.TableViewColumn
	custom     = d.CustomWidget
	splitter   = d.HSplitter
	spacer     = d.HSpacer
)
