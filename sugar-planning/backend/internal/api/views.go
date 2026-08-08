package api

import (
	"net/http"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
)

// Saved views: the filter somebody set up last Tuesday, under a name they can
// find again.
//
// These endpoints carry no permission check, and that is deliberate. A view
// belongs to whoever made it, the way a notification is addressed to a role;
// there is no operation here that reaches anybody else's, so there is nothing to
// authorise beyond having signed in. Gating personalization behind a permission
// would mean a role that can open a screen could not remember how they like to
// look at it.

func (s *Server) handleListViews(w http.ResponseWriter, r *http.Request) {
	views, err := s.views.List(r.Context(), r.URL.Query().Get("page"))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.SavedView]{Value: views, Count: len(views)})
}

func (s *Server) handleSaveView(w http.ResponseWriter, r *http.Request) {
	var req service.SaveViewRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	saved, err := s.views.Save(r.Context(), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSON(w, saved)
}

func (s *Server) handleDeleteView(w http.ResponseWriter, r *http.Request) {
	if err := s.views.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeProblem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSetDefaultView marks a view as the one to apply on arrival, or clears
// it. It is a separate command rather than a field on the save because setting a
// default is a different act from changing a filter, and doing it should not
// require re-sending a payload the caller may not still have.
func (s *Server) handleSetDefaultView(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Default *bool `json:"default"`
	}
	// A body at all is optional here: pressing "set as default" on a variant
	// control has nothing to say beyond which view, so a POST with no body is
	// the natural request and must not be an error.
	if r.ContentLength != 0 {
		if err := decodeJSON(w, r, &body); err != nil {
			writeProblem(w, r, err)
			return
		}
	}
	// Omitting the field means "make this the default"; sending false clears it.
	on := body.Default == nil || *body.Default
	if err := s.views.SetDefault(r.Context(), r.PathValue("id"), on); err != nil {
		writeProblem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
