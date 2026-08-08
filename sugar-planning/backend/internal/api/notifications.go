package api

import (
	"net/http"

	"github.com/kss/sugarplan/internal/domain"
)

// The inbox: what has been raised for the roles the caller holds, at the
// factories they may see.

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	skip, top := atoiOr(q.Get("$skip"), 0), atoiOr(q.Get("$top"), 0)
	page, err := s.notifications.List(r.Context(), q.Get("unreadOnly") == "true", skip, top)
	if err != nil {
		writeProblem(w, r, err)
		return
	}

	// The unread count comes back with the list so the badge in the shell can
	// be right without a second request.
	unread, err := s.notifications.Unread(r.Context())
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, map[string]any{
		"value": page.Items, "count": page.Count, "unread": unread,
		"skip": skip, "top": top,
	})
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	if err := s.notifications.MarkRead(r.Context(), r.PathValue("id")); err != nil {
		writeProblem(w, r, err)
		return
	}
	unread, err := s.notifications.Unread(r.Context())
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, map[string]any{"unread": unread})
}

// handleEvaluateAlerts runs the alert job on demand, which is what somebody does
// after fixing a plan rather than waiting for the next tick.
func (s *Server) handleEvaluateAlerts(w http.ResponseWriter, r *http.Request) {
	if err := requirePermission(r, domain.PermPlanRead); err != nil {
		writeProblem(w, r, err)
		return
	}
	result, err := s.notifications.Evaluate(r.Context())
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, result)
}
