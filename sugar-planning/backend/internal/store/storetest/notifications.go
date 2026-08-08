package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// An inbox addressed to a role at a factory has to mean the same thing in both
// stores, or the badge in the shell and the list under it come from different
// rules.

func testNotifications(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	inbox := s.Notifications()
	now := time.Date(2027, 1, 5, 6, 0, 0, 0, time.UTC)

	planner, err := inbox.Save(ctx, domain.Notification{
		Recipient: "SHIPMENT_PLANNER", FactoryID: f.factory,
		Severity: domain.SeverityError, Code: "CAPACITY_FULL",
		Title: "FG-WH1 fills on 1 January", Detail: "844 t/day would keep it inside capacity",
		Entity: "warehouse", EntityID: f.warehouse, CreatedAt: now,
	})
	must(t, err, "save")
	if planner.ID == "" {
		t.Fatal("a saved notification needs an id")
	}

	// One for a role nobody in this test holds, and one for another factory.
	_, err = inbox.Save(ctx, domain.Notification{
		Recipient: "QUALITY_USER", FactoryID: f.factory, Severity: domain.SeverityWarning,
		Code: "QUALITY_FAIL", Title: "A sample failed", CreatedAt: now,
	})
	must(t, err, "save another role")
	_, err = inbox.Save(ctx, domain.Notification{
		Recipient: "SHIPMENT_PLANNER", FactoryID: "", Severity: domain.SeverityWarning,
		Code: "SYSTEM", Title: "About the system rather than a plant", CreatedAt: now,
	})
	must(t, err, "save with no factory")

	mine := store.NotificationFilter{
		Recipients: []string{"SHIPMENT_PLANNER"}, Factories: []string{f.factory},
	}
	page, err := inbox.List(ctx, mine)
	must(t, err, "list")
	// The one addressed to another role is not mine; the one with no factory is,
	// because it is about the system rather than a plant.
	if len(page.Items) != 2 {
		t.Fatalf("two notifications for this role, got %d", len(page.Items))
	}

	count, err := inbox.Unread(ctx, mine)
	must(t, err, "unread")
	if count != 2 {
		t.Errorf("unread = %d, want 2", count)
	}

	// An unaddressed inbox is empty rather than everybody's.
	page, err = inbox.List(ctx, store.NotificationFilter{})
	must(t, err, "list with no recipients")
	if len(page.Items) != 0 {
		t.Errorf("a caller with no roles sees nothing, got %d", len(page.Items))
	}

	// Somebody scoped to another factory does not see this one's.
	page, err = inbox.List(ctx, store.NotificationFilter{
		Recipients: []string{"SHIPMENT_PLANNER"}, Factories: []string{newID(7)},
	})
	must(t, err, "list for another factory")
	if len(page.Items) != 1 || page.Items[0].Code != "SYSTEM" {
		t.Errorf("only the factory-less one crosses factories, got %v", page.Items)
	}

	// The deduplication key is what stops the same alert arriving at every tick.
	exists, err := inbox.ExistsSince(ctx, planner.DedupeKey(), now.Add(-time.Hour))
	must(t, err, "exists since")
	if !exists {
		t.Error("a notification raised within the window must be found by its key")
	}
	// One raised before the window has stopped suppressing.
	exists, err = inbox.ExistsSince(ctx, planner.DedupeKey(), now.Add(time.Hour))
	must(t, err, "exists since a later moment")
	if exists {
		t.Error("a notification older than the window must not suppress the next one")
	}

	must(t, inbox.MarkRead(ctx, planner.ID, []string{"SHIPMENT_PLANNER"}, now.Add(time.Hour)),
		"mark read")
	count, err = inbox.Unread(ctx, mine)
	must(t, err, "unread after reading")
	if count != 1 {
		t.Errorf("unread = %d, want 1", count)
	}
	// Reading it does not make the alert raisable again: "I know" is not
	// "remind me at the next tick".
	exists, err = inbox.ExistsSince(ctx, planner.DedupeKey(), now.Add(-time.Hour))
	must(t, err, "exists since, after reading")
	if !exists {
		t.Error("a read notification within the window must still suppress the next one")
	}

	// Reading it twice is not an error; the second time is a no-op.
	must(t, inbox.MarkRead(ctx, planner.ID, []string{"SHIPMENT_PLANNER"}, now.Add(2*time.Hour)),
		"mark read twice")

	// A role the caller does not hold cannot clear somebody else's inbox, and
	// is told it does not exist rather than that it is not theirs.
	if err := inbox.MarkRead(ctx, planner.ID, []string{"AUDITOR"}, now); err == nil {
		t.Error("marking somebody else's notification read must be refused")
	}

	page, err = inbox.List(ctx, store.NotificationFilter{
		Recipients: []string{"SHIPMENT_PLANNER"}, Factories: []string{f.factory},
		UnreadOnly: true,
	})
	must(t, err, "list unread")
	if len(page.Items) != 1 || page.Items[0].Code != "SYSTEM" {
		t.Errorf("one unread left, got %v", page.Items)
	}
}
