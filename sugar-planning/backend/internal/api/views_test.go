package api_test

import (
	"net/http"
	"strings"
	"testing"
)

// Saved views over HTTP. What matters is whose variant is whose: this is the
// one place a caller's data belongs to them rather than to a factory, and the
// failure that would matter is one person's filters reaching another.

func TestAViewBelongsToWhoeverMadeIt(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "planner", http.MethodPut, "/api/v1/views", map[string]any{
		"page": "board", "name": "My fortnight",
		"payload": map[string]any{"from": "2026-12-01", "to": "2026-12-14", "kind": "cane"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	mine := decode(t, rec)
	if mine["owner"] != "planner" {
		t.Errorf("owner = %v, want the caller", mine["owner"])
	}
	// The factory comes from the caller's scope, not from the request: letting
	// a client name one would let somebody publish into a plant they cannot see.
	if mine["factoryId"] != ts.seeded.FactoryID {
		t.Errorf("factoryId = %v, want the caller's factory", mine["factoryId"])
	}
	// The payload is stored and not interpreted.
	payload := mine["payload"].(map[string]any)
	if payload["from"] != "2026-12-01" || payload["kind"] != "cane" {
		t.Errorf("the payload was altered: %v", payload)
	}

	// Somebody else saves one on the same page.
	rec = ts.do(t, "supervisor", http.MethodPut, "/api/v1/views", map[string]any{
		"page": "board", "name": "Night shift", "payload": map[string]any{"kind": "production"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("save as supervisor: %d %s", rec.Code, rec.Body.String())
	}
	theirs := decode(t, rec)

	// A personal view is invisible to everyone else.
	rec = ts.do(t, "planner", http.MethodGet, "/api/v1/views?page=board", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	for _, item := range decode(t, rec)["value"].([]any) {
		if item.(map[string]any)["name"] == "Night shift" {
			t.Error("somebody else's personal view must not be listed")
		}
	}

	// And cannot be deleted. Not found rather than forbidden: who has which
	// variants is not a question this answers to a caller.
	rec = ts.do(t, "planner", http.MethodDelete, "/api/v1/views/"+theirs["id"].(string), nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("deleting somebody else's view: %d, want 404", rec.Code)
	}

	// A shared one is offered to colleagues at the same factory, and still only
	// its owner may change it.
	rec = ts.do(t, "supervisor", http.MethodPut, "/api/v1/views", map[string]any{
		"page": "board", "name": "Morning review", "shared": true,
		"payload": map[string]any{"kind": "cane"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("save shared: %d %s", rec.Code, rec.Body.String())
	}
	shared := decode(t, rec)

	rec = ts.do(t, "planner", http.MethodGet, "/api/v1/views?page=board", nil)
	found := false
	for _, item := range decode(t, rec)["value"].([]any) {
		if item.(map[string]any)["name"] == "Morning review" {
			found = true
		}
	}
	if !found {
		t.Error("a shared view must be offered to colleagues at the same factory")
	}
	rec = ts.do(t, "planner", http.MethodDelete, "/api/v1/views/"+shared["id"].(string), nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("deleting somebody else's shared view: %d, want 404", rec.Code)
	}

	// Another factory's people see none of it.
	rec = ts.do(t, "outsider", http.MethodGet, "/api/v1/views?page=board", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("outsider list: %d %s", rec.Code, rec.Body.String())
	}
	if n := len(decode(t, rec)["value"].([]any)); n != 0 {
		t.Errorf("a shared view must not cross factories, the outsider sees %d", n)
	}

	// My own I may delete.
	rec = ts.do(t, "planner", http.MethodDelete, "/api/v1/views/"+mine["id"].(string), nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("deleting my own: %d %s", rec.Code, rec.Body.String())
	}
}

func TestSavingOverANameReplacesIt(t *testing.T) {
	ts := newTestServer(t)
	save := func(kind string) map[string]any {
		rec := ts.do(t, "planner", http.MethodPut, "/api/v1/views", map[string]any{
			"page": "board", "name": "Working set",
			"payload": map[string]any{"kind": kind},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
		}
		return decode(t, rec)
	}

	first := save("cane")
	second := save("storage")
	if first["id"] != second["id"] {
		t.Error("saving over a name must replace it rather than adding a second entry")
	}

	rec := ts.do(t, "planner", http.MethodGet, "/api/v1/views?page=board", nil)
	if n := len(decode(t, rec)["value"].([]any)); n != 1 {
		t.Errorf("one view of that name, got %d", n)
	}
	if second["payload"].(map[string]any)["kind"] != "storage" {
		t.Errorf("the payload was not replaced: %v", second["payload"])
	}
}

func TestAtMostOneDefaultPerPersonPerPage(t *testing.T) {
	ts := newTestServer(t)
	save := func(name string, isDefault bool) string {
		rec := ts.do(t, "planner", http.MethodPut, "/api/v1/views", map[string]any{
			"page": "board", "name": name, "isDefault": isDefault,
			"payload": map[string]any{"kind": "cane"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("save %s: %d %s", name, rec.Code, rec.Body.String())
		}
		return decode(t, rec)["id"].(string)
	}

	first := save("First", true)
	second := save("Second", false)

	rec := ts.do(t, "planner", http.MethodPost, "/api/v1/views/"+second+"/default", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("set default: %d %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "planner", http.MethodGet, "/api/v1/views?page=board", nil)
	defaults := []string{}
	for _, item := range decode(t, rec)["value"].([]any) {
		v := item.(map[string]any)
		if v["isDefault"] == true {
			defaults = append(defaults, v["name"].(string))
		}
	}
	if len(defaults) != 1 || defaults[0] != "Second" {
		t.Errorf("defaults = %v, want only Second", defaults)
	}
	_ = first

	// Sending false clears it, so a screen can be returned to opening unfiltered.
	rec = ts.do(t, "planner", http.MethodPost, "/api/v1/views/"+second+"/default",
		map[string]any{"default": false})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("clear default: %d %s", rec.Code, rec.Body.String())
	}
	rec = ts.do(t, "planner", http.MethodGet, "/api/v1/views?page=board", nil)
	for _, item := range decode(t, rec)["value"].([]any) {
		if item.(map[string]any)["isDefault"] == true {
			t.Errorf("the default was not cleared: %v", item)
		}
	}
}

func TestAViewIsRefusedWhenItCouldNotBeApplied(t *testing.T) {
	ts := newTestServer(t)

	bad := []struct {
		why  string
		body map[string]any
		want string
	}{
		{"no name", map[string]any{"page": "board", "payload": map[string]any{"a": 1}}, "name"},
		{"a screen that holds no views",
			map[string]any{"page": "launchpad", "name": "x", "payload": map[string]any{"a": 1}}, "page"},
		{"nothing in it", map[string]any{"page": "board", "name": "x"}, "payload"},
	}
	for _, c := range bad {
		rec := ts.do(t, "planner", http.MethodPut, "/api/v1/views", c.body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", c.why, rec.Code, rec.Body.String())
			continue
		}
		// The message is addressed to the field, so a MessagePopover can put it
		// on the control that has to change.
		errs, _ := decode(t, rec)["errors"].([]any)
		if len(errs) == 0 {
			t.Errorf("%s: the refusal must name a field", c.why)
			continue
		}
		if field := errs[0].(map[string]any)["field"]; field != c.want {
			t.Errorf("%s: field = %v, want %s", c.why, field, c.want)
		}
	}

	// A view is not a place to keep arbitrary data.
	rec := ts.do(t, "planner", http.MethodPut, "/api/v1/views", map[string]any{
		"page": "board", "name": "Too big",
		"payload": map[string]any{"blob": strings.Repeat("x", 20<<10)},
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("an oversized payload: %d", rec.Code)
	}
}

func TestPersonalizationNeedsNoPermissionBeyondSigningIn(t *testing.T) {
	ts := newTestServer(t)

	// An executive viewer may change nothing in this system. They may still
	// remember how they like to look at a screen: gating that behind a
	// permission would mean a role that can open a page cannot personalise it.
	rec := ts.do(t, "executive", http.MethodPut, "/api/v1/views", map[string]any{
		"page": "board", "name": "How I read it", "payload": map[string]any{"kind": "cane"},
	})
	if rec.Code != http.StatusOK {
		t.Errorf("an executive viewer saving a view: %d %s", rec.Code, rec.Body.String())
	}

	// Signing in is still required.
	rec = ts.do(t, "", http.MethodGet, "/api/v1/views?page=board", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("an anonymous caller: %d, want 401", rec.Code)
	}
}
