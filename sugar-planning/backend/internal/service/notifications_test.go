package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
)

// The inbox is what turns an alert somebody could have seen into one somebody
// was actually told about. These cover the difference.

func notificationHarness(t *testing.T) (*harness, *service.Notifications) {
	t.Helper()
	h := newHarness(t, 14)
	return h, service.NewNotifications(h.store, h.analytics, fixedClock())
}

func TestTheAlertsOfTheSeasonReachTheRightInboxes(t *testing.T) {
	h, notes := notificationHarness(t)
	system := h.as(auth.RoleExecutiveViewer)

	result, err := notes.Evaluate(system)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.Seasons != 1 {
		t.Fatalf("one open season, got %d", result.Seasons)
	}
	// The reference scenario has real capacity problems, which is the whole
	// reason this job exists.
	if result.Alerts == 0 || result.Raised == 0 {
		t.Fatalf("the seeded plan has alerts to raise: %+v", result)
	}

	// A capacity warning is the shipment planner's problem before it is
	// anybody else's.
	page, err := notes.List(h.as(auth.RoleShipmentPlanner), false, 0, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) == 0 {
		t.Fatal("the shipment planner must have been told about the stores filling")
	}
	for _, item := range page.Items {
		if item.Severity != domain.SeverityError && item.Severity != domain.SeverityWarning {
			t.Errorf("only what somebody must act on belongs in an inbox: %+v", item)
		}
		if item.FactoryID != h.seeded.FactoryID {
			t.Errorf("a notification must say which factory it is about: %+v", item)
		}
	}

	// Somebody who holds none of the addressed roles has an empty inbox rather
	// than everybody's.
	page, err = notes.List(h.as(auth.RoleAuditor), false, 0, 100)
	if err != nil {
		t.Fatalf("list as auditor: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("an auditor is addressed by none of these, got %d", len(page.Items))
	}
}

func TestTheSameAlertIsNotRaisedTwice(t *testing.T) {
	h, notes := notificationHarness(t)
	system := h.as(auth.RoleExecutiveViewer)

	first, err := notes.Evaluate(system)
	if err != nil {
		t.Fatalf("first pass: %v", err)
	}
	if first.Raised == 0 {
		t.Fatal("the first pass must raise something")
	}

	second, err := notes.Evaluate(system)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	// Without this every tick would repeat the same warning until nobody read
	// any of them.
	if second.Raised != 0 {
		t.Errorf("the second pass must raise nothing new, raised %d", second.Raised)
	}
	if second.Suppressed == 0 {
		t.Error("the suppressed count is what shows the deduplication working")
	}

	planner := h.as(auth.RoleShipmentPlanner)
	before, err := notes.Unread(planner)
	if err != nil {
		t.Fatalf("unread: %v", err)
	}
	if before == 0 {
		t.Fatal("the planner has unread notifications")
	}

	page, _ := notes.List(planner, true, 0, 100)
	if err := notes.MarkRead(planner, page.Items[0].ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	after, err := notes.Unread(planner)
	if err != nil {
		t.Fatalf("unread again: %v", err)
	}
	if after != before-1 {
		t.Errorf("unread went from %d to %d", before, after)
	}

	// Reading it does not bring it straight back. "I know" is not "remind me at
	// the next tick", and an alert that stays true for a fortnight must not fill
	// an inbox with a fortnight of copies of itself.
	third, err := notes.Evaluate(system)
	if err != nil {
		t.Fatalf("third pass: %v", err)
	}
	if third.Raised != 0 {
		t.Errorf("a dismissed alert must stay quiet for a day, raised %d", third.Raised)
	}

	// A day later it comes back, because an alert nobody acted on should not be
	// forgotten either.
	tomorrow := service.NewNotifications(h.store, h.analytics, func() time.Time {
		return fixedClock()().Add(service.RaiseAgainAfter + time.Hour)
	})
	fourth, err := tomorrow.Evaluate(system)
	if err != nil {
		t.Fatalf("fourth pass: %v", err)
	}
	if fourth.Raised == 0 {
		t.Error("an alert still true a day later must be raised again")
	}
}

func TestAnInboxBelongsToTheRolesTheCallerHolds(t *testing.T) {
	h, notes := notificationHarness(t)
	if _, err := notes.Evaluate(h.as(auth.RoleExecutiveViewer)); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	planner := h.as(auth.RoleShipmentPlanner)
	page, err := notes.List(planner, false, 0, 100)
	if err != nil || len(page.Items) == 0 {
		t.Fatalf("list: %v %d", err, len(page.Items))
	}
	id := page.Items[0].ID

	// Somebody who does not hold the role cannot mark it read, and is told it
	// does not exist rather than that it is not theirs.
	err = notes.MarkRead(h.as(auth.RoleAuditor), id)
	if err == nil {
		t.Error("an auditor must not be able to clear the shipment planner's inbox")
	}

	// A caller with no roles at all has an empty inbox rather than an error.
	empty := auth.WithPrincipal(context.Background(),
		auth.NewPrincipal("nobody", "nobody", "Nobody", "", nil, nil, nil))
	page, err = notes.List(empty, false, 0, 100)
	if err != nil {
		t.Fatalf("list with no roles: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("an unaddressed inbox is empty, got %d", len(page.Items))
	}
}
