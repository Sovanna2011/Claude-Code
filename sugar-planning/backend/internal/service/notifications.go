package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Notifications is the inbox and the job that fills it.
//
// The distinction the whole thing rests on: an **alert** is calculated - it is
// true of the plan at the moment somebody looks at the dashboard - while a
// **notification** is a fact about a person. This was raised, they were told,
// they have or have not read it. A notification survives the alert ceasing to be
// true, which is what makes an inbox an account of what happened rather than a
// second dashboard.
type Notifications struct {
	store     store.Store
	analytics *Analytics
	now       func() time.Time
}

// NewNotifications builds the service.
func NewNotifications(s store.Store, analytics *Analytics, now func() time.Time) *Notifications {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Notifications{store: s, analytics: analytics, now: now}
}

// ---------------------------------------------------------------------------
// The inbox
// ---------------------------------------------------------------------------

// inboxOf is the filter for whoever is asking: the roles they hold and the
// factories they may see. It is built in one place so the badge and the list
// cannot disagree about what is in somebody's inbox.
func inboxOf(caller auth.Principal, unreadOnly bool, skip, top int) store.NotificationFilter {
	return store.NotificationFilter{
		Recipients: caller.Roles, Factories: caller.Factories,
		UnreadOnly: unreadOnly, Skip: skip, Top: top,
	}
}

// List returns what the caller's roles entitle them to see.
//
// There is no permission for reading somebody else's inbox, because there is no
// such thing here: a notification is addressed to a role at a factory, and
// holding that role there is what makes it yours. Reading the alerts themselves
// is a different act, and anybody with plan:read can do it on the dashboard.
func (n *Notifications) List(ctx context.Context, unreadOnly bool, skip, top int) (store.Page[domain.Notification], error) {
	caller := auth.FromContext(ctx)
	if len(caller.Roles) == 0 {
		return store.Page[domain.Notification]{Items: []domain.Notification{}}, nil
	}
	return n.store.Notifications().List(ctx, inboxOf(caller, unreadOnly, skip, top))
}

// Unread counts what is waiting, for the badge in the shell.
func (n *Notifications) Unread(ctx context.Context) (int, error) {
	caller := auth.FromContext(ctx)
	if len(caller.Roles) == 0 {
		return 0, nil
	}
	return n.store.Notifications().Unread(ctx, inboxOf(caller, true, 0, 0))
}

// MarkRead marks one notification read.
//
// A notification is read once, by whoever gets to it first: it is addressed to a
// role, and the point of that is that somebody deals with it - not that everyone
// holding the role ticks it off separately.
func (n *Notifications) MarkRead(ctx context.Context, id string) error {
	caller := auth.FromContext(ctx)
	if len(caller.Roles) == 0 {
		return fmt.Errorf("%w: notification %s", domain.ErrNotFound, id)
	}
	return n.store.Notifications().MarkRead(ctx, id, caller.Roles, n.now())
}

// ---------------------------------------------------------------------------
// Raising them
// ---------------------------------------------------------------------------

// RaiseAgainAfter is how long an alert stays quiet once it has been raised.
//
// Reading a notification means "I know", not "remind me at the next tick", so
// the suppression counts read notifications too. A day is long enough that an
// alert which stays true for a fortnight does not fill an inbox with a fortnight
// of copies of itself, and short enough that one nobody has acted on comes back
// rather than being forgotten.
const RaiseAgainAfter = 24 * time.Hour

// EvaluateResult reports what one pass of the alert job did.
type EvaluateResult struct {
	Seasons int `json:"seasons"`
	Alerts  int `json:"alerts"`
	Raised  int `json:"raised"`
	// Suppressed counts the alerts already raised within the last day. It is the
	// figure that shows the deduplication working: without it every tick would
	// raise the same warning again.
	Suppressed int `json:"suppressed"`
}

// Evaluate calculates the alerts for every open season and puts the new ones in
// the right inboxes.
//
// It runs as a scheduled job rather than on a page load because that is the
// point: a store that will fill on 1 January should reach somebody in November,
// not the next time a planner happens to open the dashboard.
func (n *Notifications) Evaluate(ctx context.Context) (EvaluateResult, error) {
	result := EvaluateResult{}

	seasons, err := n.store.Planning().ListSeasons(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return result, err
	}

	for _, season := range seasons.Items {
		if strings.EqualFold(season.Status, "CLOSED") {
			continue
		}
		result.Seasons++

		dash, err := n.analytics.Dashboard(ctx, DashboardRequest{SeasonID: season.ID})
		if err != nil {
			// A season with no released plan cannot be measured, which is not a
			// failure of the job: the others still need evaluating.
			continue
		}
		result.Alerts += len(dash.Alerts)

		for _, alert := range dash.Alerts {
			raised, err := n.raise(ctx, season, alert)
			if err != nil {
				return result, err
			}
			if raised {
				result.Raised++
				continue
			}
			result.Suppressed++
		}
	}
	return result, nil
}

// raise puts one alert in the inboxes of the people whose job it is, unless it
// is already sitting unread in them.
func (n *Notifications) raise(ctx context.Context, season domain.Season,
	alert domain.Alert,
) (bool, error) {

	// Only what somebody has to act on is sent. An informational alert belongs
	// on the dashboard, and an inbox that fills with them is an inbox nobody
	// reads - which costs the warnings that do matter.
	if alert.Severity != domain.SeverityError && alert.Severity != domain.SeverityWarning {
		return false, nil
	}

	raised := false
	for _, role := range rolesFor(alert) {
		item := domain.Notification{
			Recipient: role, FactoryID: season.FactoryID,
			Severity: alert.Severity, Code: alert.Code,
			Title: alert.Title, Detail: alert.Detail,
			Entity: alert.Entity, EntityID: alert.EntityID,
			CreatedAt: n.now(),
		}
		if item.Entity == "" {
			// An alert about the season as a whole is still about something, and
			// without this every such alert would share one deduplication key.
			item.Entity, item.EntityID = "season", season.ID
		}

		exists, err := n.store.Notifications().ExistsSince(ctx, item.DedupeKey(),
			n.now().Add(-RaiseAgainAfter))
		if err != nil {
			return false, err
		}
		if exists {
			continue
		}
		if _, err := n.store.Notifications().Save(ctx, item); err != nil {
			return false, err
		}
		raised = true
	}
	return raised, nil
}

// rolesFor works out whose job an alert is.
//
// It is a routing decision rather than an authorisation one: everybody named
// here could already see the alert on the dashboard, and what this decides is
// whose inbox it lands in. Sending everything to everybody would be easier and
// would mean nobody owned any of it.
func rolesFor(alert domain.Alert) []string {
	switch {
	case strings.HasPrefix(alert.Code, "CAPACITY"), strings.HasPrefix(alert.Code, "STORAGE"):
		// A store about to overflow is a shipment problem before it is a
		// planning one: the way out is to ship faster.
		return []string{auth.RoleShipmentPlanner, auth.RoleProductionPlanner}
	case strings.HasPrefix(alert.Code, "MATERIAL"), strings.HasPrefix(alert.Code, "SHORTAGE"):
		return []string{auth.RoleProductionPlanner}
	case strings.HasPrefix(alert.Code, "QUALITY"):
		return []string{auth.RoleQualityUser}
	case strings.HasPrefix(alert.Code, "DOWNTIME"), strings.HasPrefix(alert.Code, "CRUSH"):
		return []string{auth.RoleShiftSupervisor, auth.RoleProductionPlanner}
	}
	// Anything else is the plan's problem, and the approver is the person who
	// has to decide what to do about it.
	return []string{auth.RoleProductionPlanner, auth.RoleApprover}
}
