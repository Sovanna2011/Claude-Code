package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Views is the saved-filter service behind variant management.
//
// It is the one place in this system where a caller's data belongs to *them*
// rather than to a factory, and that shapes the whole of it. There is no
// permission to read somebody else's views, because there is no such operation:
// a view is addressed to its owner the way a notification is addressed to a
// role. What a caller may see is their own plus what colleagues have chosen to
// share, and what they may change is only their own.
type Views struct {
	store store.Store
}

// NewViews builds the service.
func NewViews(s store.Store) *Views { return &Views{store: s} }

// List returns the views a caller may use on one page.
//
// A caller who has signed in may always read their own views: personalization
// is not a privilege, and gating it behind a permission would mean a role that
// can see a screen could not remember how they like to look at it.
func (v *Views) List(ctx context.Context, page string) ([]domain.SavedView, error) {
	caller := auth.FromContext(ctx)
	if caller.Username == "" {
		return []domain.SavedView{}, nil
	}
	if page != "" && !domain.ValidViewPage(page) {
		return nil, fmt.Errorf("%w: %q is not a screen that holds saved views",
			domain.ErrValidation, page)
	}
	return v.store.SavedViews().List(ctx, store.SavedViewFilter{
		Owner: caller.Username, Page: page, Factories: caller.Factories,
	})
}

// SaveViewRequest is one variant being stored.
//
// Payload is json.RawMessage rather than a map: this layer stores it and does
// not read it, and decoding it into a map only to encode it again would lose
// key order and turn every integer into a float for no purpose.
type SaveViewRequest struct {
	Page      string          `json:"page"`
	Name      string          `json:"name"`
	Shared    bool            `json:"shared"`
	IsDefault bool            `json:"isDefault"`
	Payload   json.RawMessage `json:"payload"`
}

// Save stores a variant under the caller's name.
//
// Saving over a name replaces it, which is what "save" means on a variant
// control. The factory is taken from the caller's scope rather than from the
// request: a shared view is offered to people at a factory, and letting a
// client name one would let somebody publish a variant into a plant they cannot
// see.
func (v *Views) Save(ctx context.Context, req SaveViewRequest) (domain.SavedView, error) {
	caller := auth.FromContext(ctx)
	if caller.Username == "" {
		return domain.SavedView{}, fmt.Errorf("%w: a view belongs to somebody", domain.ErrForbidden)
	}

	item := domain.SavedView{
		Owner:     caller.Username,
		Page:      strings.TrimSpace(req.Page),
		Name:      strings.TrimSpace(req.Name),
		Shared:    req.Shared,
		IsDefault: req.IsDefault,
		Payload:   req.Payload,
		FactoryID: soleFactory(caller),
	}
	if err := item.Validate(); err != nil {
		return domain.SavedView{}, err
	}
	// Sharing a view with a factory nobody can name would put it in front of
	// everybody, which is not what the person pressing "share" meant.
	if item.Shared && item.FactoryID == "" {
		return domain.SavedView{}, &domain.ValidationError{Errors: []domain.FieldError{{
			Field: "shared", Code: "NO_FACTORY_SCOPE",
			Message: "a shared view is published to one factory, and your account is not " +
				"scoped to exactly one; save it as a personal view instead",
		}}}
	}
	return v.store.SavedViews().Save(ctx, item)
}

// Delete removes one of the caller's own views.
func (v *Views) Delete(ctx context.Context, id string) error {
	caller := auth.FromContext(ctx)
	if caller.Username == "" {
		return fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
	}
	return v.store.SavedViews().Delete(ctx, id, caller.Username)
}

// SetDefault marks one of the caller's own views as the one to apply on
// arrival, or clears it.
func (v *Views) SetDefault(ctx context.Context, id string, on bool) error {
	caller := auth.FromContext(ctx)
	if caller.Username == "" {
		return fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
	}
	return v.store.SavedViews().SetDefault(ctx, id, caller.Username, on)
}

// soleFactory is the factory a view belongs to.
//
// Somebody scoped to one plant gets their views scoped to it, so that working
// across two factories does not mean seeing the other one's filters offered
// here. Somebody scoped to several - or to all of them - gets a view with no
// factory, which is honest: it is not about a particular plant.
func soleFactory(caller auth.Principal) string {
	if len(caller.Factories) == 1 {
		return caller.Factories[0]
	}
	return ""
}
