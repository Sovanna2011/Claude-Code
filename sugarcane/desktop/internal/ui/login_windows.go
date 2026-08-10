//go:build windows

package ui

import (
	"context"

	"github.com/lxn/walk"
	d "github.com/lxn/walk/declarative"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/api"
)

// showLogin puts the sign-in box up and returns a signed-in client, or nil if the user closed it.
//
// The box stays open on a refusal, with the service's own message beneath the fields, rather than
// throwing the typing away behind a message box. The request runs off the window thread and the
// answer is applied back on it: signing in against a service that is not there takes as long as the
// connection takes to fail, and a frozen window for that long looks like a crash.
func showLogin(opts Options) (*api.Client, error) {
	var (
		dialog             *walk.Dialog
		serverBox, userBox *walk.LineEdit
		passwordBox        *walk.LineEdit
		message            *walk.Label
		signIn, cancel     *walk.PushButton
		client             *api.Client
	)

	attempt := func() {
		server, user, password := serverBox.Text(), userBox.Text(), passwordBox.Text()
		if server == "" || user == "" || password == "" {
			message.SetText("The server, the user name and the password are all needed.")
			return
		}

		// The address may have been edited, so a client is made for whatever it now says.
		attempted := api.New(server)
		signIn.SetEnabled(false)
		message.SetText("Signing in…")

		go func() {
			_, err := call(func(ctx context.Context) (api.User, error) {
				return attempted.SignIn(ctx, user, password)
			})
			dialog.Synchronize(func() {
				signIn.SetEnabled(true)
				if err != nil {
					message.SetText(err.Error())
					passwordBox.SetFocus()
					passwordBox.SetTextSelection(0, -1)
					return
				}
				client = attempted
				dialog.Accept()
			})
		}()
	}

	err := d.Dialog{
		AssignTo:      &dialog,
		Title:         "Farm Area Monitoring — sign in",
		DefaultButton: &signIn,
		CancelButton:  &cancel,
		MinSize:       d.Size{Width: 460, Height: 250},
		Layout:        d.VBox{},
		Children: []d.Widget{
			d.Label{
				Text:      "Sign in to monitor land utilisation across the plantation.",
				TextColor: walk.RGB(0x55, 0x6B, 0x82),
			},
			d.Composite{
				Layout: d.Grid{Columns: 2},
				Children: []d.Widget{
					d.Label{Text: "Server"},
					d.LineEdit{AssignTo: &serverBox, Text: opts.BaseURL},

					d.Label{Text: "User name"},
					d.LineEdit{AssignTo: &userBox, Text: opts.UserName},

					d.Label{Text: "Password"},
					d.LineEdit{AssignTo: &passwordBox, PasswordMode: true},
				},
			},
			d.Label{AssignTo: &message, Text: "", TextColor: walk.RGB(0xAA, 0x08, 0x08)},
			d.VSpacer{},
			d.Composite{
				Layout: d.HBox{},
				Children: []d.Widget{
					d.HSpacer{},
					d.PushButton{AssignTo: &signIn, Text: "Sign in", OnClicked: attempt},
					d.PushButton{AssignTo: &cancel, Text: "Cancel", OnClicked: func() { dialog.Cancel() }},
				},
			},
		},
	}.Create(nil)
	if err != nil {
		return nil, err
	}

	if userBox.Text() == "" {
		userBox.SetFocus()
	} else {
		passwordBox.SetFocus()
	}

	dialog.Run()
	return client, nil
}
