package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SavedView is a named set of filters, sorts and column choices for one screen.
//
// Section 15 asks for variant management, saved views and personalization.
// Stored, all three are this: what a page looked like when somebody had it the
// way they wanted, under a name they can find again.
//
// The Payload is opaque on purpose. What a view holds is a property of the
// screen it belongs to - the planning board saves a date range and a series, the
// order list saves a status filter - and modelling it here would mean changing
// the domain every time a screen grew a filter. The server stores what the page
// sent and hands it back; the page is the only thing that can interpret it.
type SavedView struct {
	ID string `json:"id"`
	// Owner is the username from the token. There is no user table: identity
	// lives in the identity provider, and a view belongs to whoever made it.
	Owner string `json:"owner"`
	// Page is the route name, so a view cannot be offered on a screen that
	// could not apply it.
	Page      string `json:"page"`
	Name      string `json:"name"`
	FactoryID string `json:"factoryId,omitempty"`
	// Shared publishes the view to everyone at the same factory. Only the owner
	// may change or delete it: a variant anybody can edit is a variant nobody
	// can rely on.
	Shared bool `json:"shared"`
	// IsDefault applies the view when the page opens. At most one per person
	// per page.
	IsDefault bool            `json:"isDefault"`
	Payload   json.RawMessage `json:"payload"`
	AuditFields
}

// MaxViewPayloadBytes bounds what one view may hold.
//
// A filter set is a few hundred bytes. The limit is generous enough that no
// real screen will meet it and small enough that the endpoint cannot be used to
// store arbitrary data per user.
const MaxViewPayloadBytes = 16 << 10

// ViewPages is the set of screens a view may be saved against.
//
// It is a list rather than free text because the page name decides which screen
// offers the view, and a typo would produce a variant that is stored, listed
// nowhere, and quietly lost.
var ViewPages = []string{
	"board", "downtime", "orders", "quality", "stock", "shipments", "warehouse",
	"materials", "audit", "interfaces", "import", "reports",
}

// ValidViewPage reports whether a screen may hold saved views.
func ValidViewPage(page string) bool {
	for _, p := range ViewPages {
		if p == page {
			return true
		}
	}
	return false
}

// Validate checks a view before it is stored.
//
// The payload is checked for being well-formed JSON and for its size, and for
// nothing else: the server does not know what a board's filters mean, and
// pretending to validate them would only reject the next filter somebody adds.
func (v SavedView) Validate() error {
	verr := &ValidationError{}
	if strings.TrimSpace(v.Name) == "" {
		verr.Add("name", "REQUIRED", "a view needs a name to be found again by")
	}
	if len([]rune(v.Name)) > 60 {
		verr.Add("name", "TOO_LONG", "a view name is at most 60 characters")
	}
	if !ValidViewPage(v.Page) {
		verr.Add("page", "UNKNOWN_PAGE", fmt.Sprintf(
			"%q is not a screen that holds saved views; it is one of %s",
			v.Page, strings.Join(ViewPages, ", ")))
	}
	switch {
	case len(v.Payload) == 0:
		verr.Add("payload", "REQUIRED", "a view with nothing in it would restore nothing")
	case len(v.Payload) > MaxViewPayloadBytes:
		verr.Add("payload", "TOO_LARGE", fmt.Sprintf(
			"a view holds at most %d bytes; this one is %d",
			MaxViewPayloadBytes, len(v.Payload)))
	case !json.Valid(v.Payload):
		verr.Add("payload", "INVALID_JSON", "the view payload is not valid JSON")
	}
	return verr.OrNil()
}
