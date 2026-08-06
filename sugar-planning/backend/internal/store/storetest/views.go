package storetest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Saved views: whose variant is whose, and what "save" over a name means. Both
// stores have to agree, because a variant list that differs between them is a
// personalization somebody loses on the day the deployment changes.

func testSavedViews(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	views := s.SavedViews()

	view := func(owner, page, name string, shared bool, payload string) domain.SavedView {
		return domain.SavedView{
			Owner: owner, Page: page, Name: name, Shared: shared,
			FactoryID: f.factory, Payload: json.RawMessage(payload),
		}
	}

	mine, err := views.Save(ctx, view("sokha", "board", "My fortnight", false,
		`{"from":"2026-12-01","to":"2026-12-14","kind":"cane"}`))
	must(t, err, "save mine")
	if mine.ID == "" {
		t.Fatal("a saved view needs an id")
	}
	if mine.RowVersion != 1 {
		t.Errorf("row version = %d, want 1 on insert", mine.RowVersion)
	}

	// Saving over a name replaces it, which is what "save" on a variant means.
	// A second row of the same name would give somebody two identical entries
	// in the list and no way to tell which one they were editing.
	again, err := views.Save(ctx, view("sokha", "board", "My fortnight", false,
		`{"from":"2027-01-01","to":"2027-01-14","kind":"production"}`))
	must(t, err, "save over the same name")
	if again.ID != mine.ID {
		t.Errorf("saving over a name must replace it, got a new id %s", again.ID)
	}
	if again.RowVersion <= mine.RowVersion {
		t.Errorf("row version = %d, want more than %d", again.RowVersion, mine.RowVersion)
	}
	if !sameJSON(string(again.Payload), `{"from":"2027-01-01","to":"2027-01-14","kind":"production"}`) {
		t.Errorf("the payload was not replaced: %s", again.Payload)
	}

	// The same name on a different page is a different view: the two screens
	// cannot apply each other's filters anyway.
	other, err := views.Save(ctx, view("sokha", "downtime", "My fortnight", false, `{"lineId":""}`))
	must(t, err, "same name, other page")
	if other.ID == again.ID {
		t.Error("a name is unique per page, not across the application")
	}

	// Somebody else's personal view is invisible; their shared one is not.
	_, err = views.Save(ctx, view("dara", "board", "Dara's private", false, `{"kind":"storage"}`))
	must(t, err, "save another owner")
	shared, err := views.Save(ctx, view("dara", "board", "Morning review", true, `{"kind":"cane"}`))
	must(t, err, "save shared")

	list, err := views.List(ctx, store.SavedViewFilter{
		Owner: "sokha", Page: "board", Factories: []string{f.factory},
	})
	must(t, err, "list")
	if len(list) != 2 {
		t.Fatalf("mine plus one shared, got %d: %v", len(list), names(list))
	}
	// Own views first, so the personal ones are where somebody looks for them.
	if list[0].Owner != "sokha" {
		t.Errorf("a caller's own views come first, got %v", names(list))
	}
	for _, item := range list {
		if item.Name == "Dara's private" {
			t.Error("somebody else's personal view must not be listed")
		}
	}

	// A shared view from a factory outside the caller's scope is not offered.
	elsewhere, err := views.List(ctx, store.SavedViewFilter{
		Owner: "sokha", Page: "board", Factories: []string{newID(9)},
	})
	must(t, err, "list from another scope")
	if len(elsewhere) != 1 || elsewhere[0].Owner != "sokha" {
		t.Errorf("only my own view belongs to another factory's scope, got %v", names(elsewhere))
	}

	// A caller who is nobody has no views. Empty rather than everything: this
	// is the failure mode that would leak one person's filters to another.
	nobody, err := views.List(ctx, store.SavedViewFilter{Page: "board"})
	must(t, err, "list with no owner")
	if len(nobody) != 0 {
		t.Errorf("a caller with no username has no views, got %d", len(nobody))
	}

	// --- defaults -----------------------------------------------------------

	must(t, views.SetDefault(ctx, again.ID, "sokha", true), "set default")
	stored, err := views.Get(ctx, again.ID)
	must(t, err, "get after default")
	if !stored.IsDefault {
		t.Error("the view was not marked default")
	}

	// At most one default per person per page. Setting a second must clear the
	// first rather than leaving two views both claiming to open the screen.
	second, err := views.Save(ctx, view("sokha", "board", "Whole season", false, `{"kind":"cane"}`))
	must(t, err, "save second")
	must(t, views.SetDefault(ctx, second.ID, "sokha", true), "set second default")

	list, err = views.List(ctx, store.SavedViewFilter{
		Owner: "sokha", Page: "board", Factories: []string{f.factory},
	})
	must(t, err, "list after defaults")
	defaults := 0
	for _, item := range list {
		if item.IsDefault {
			defaults++
			if item.ID != second.ID {
				t.Errorf("the default is %s, want the one just set", item.Name)
			}
		}
	}
	if defaults != 1 {
		t.Errorf("%d views claim to be the default; at most one may", defaults)
	}

	// Saving a view as the default has the same effect as the command, so a
	// variant control that does it in one step cannot end up with two.
	third, err := views.Save(ctx, domain.SavedView{
		Owner: "sokha", Page: "board", Name: "Last week", FactoryID: f.factory,
		IsDefault: true, Payload: json.RawMessage(`{"kind":"cane"}`),
	})
	must(t, err, "save as default")
	list, err = views.List(ctx, store.SavedViewFilter{
		Owner: "sokha", Page: "board", Factories: []string{f.factory},
	})
	must(t, err, "list after saving a default")
	defaults = 0
	for _, item := range list {
		if item.IsDefault {
			defaults++
			if item.ID != third.ID {
				t.Errorf("the default is %s, want the one just saved", item.Name)
			}
		}
	}
	if defaults != 1 {
		t.Errorf("%d defaults after saving one; at most one may", defaults)
	}

	must(t, views.SetDefault(ctx, third.ID, "sokha", false), "clear default")
	stored, err = views.Get(ctx, third.ID)
	must(t, err, "get after clearing")
	if stored.IsDefault {
		t.Error("the default was not cleared")
	}

	// --- ownership ----------------------------------------------------------

	// Somebody else's view is not found rather than forbidden: who has which
	// variants is not a question this answers to a caller.
	err = views.Delete(ctx, shared.ID, "sokha")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("deleting a shared view I do not own: %v, want not found", err)
	}
	err = views.SetDefault(ctx, shared.ID, "sokha", true)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("defaulting somebody else's view: %v, want not found", err)
	}
	// And it is still there.
	if _, err := views.Get(ctx, shared.ID); err != nil {
		t.Errorf("the shared view must survive a refused delete: %v", err)
	}

	must(t, views.Delete(ctx, again.ID, "sokha"), "delete my own")
	if _, err := views.Get(ctx, again.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("after deleting: %v, want not found", err)
	}
	if err := views.Delete(ctx, again.ID, "sokha"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("deleting twice: %v, want not found", err)
	}
}

func names(items []domain.SavedView) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Owner + "/" + item.Name
	}
	return out
}
