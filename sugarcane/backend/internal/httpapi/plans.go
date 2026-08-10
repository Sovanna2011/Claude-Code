package httpapi

import (
	"net/http"

	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
)

// The activity plan endpoints. Generating is a POST to the collection rather than a PUT on a
// field: it is an action with a result, not a value being set.

func (a *API) listPlans(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Plan.List(r.Context(), f, pageFrom(r), userOf(r))
	a.respond(w, r, result, err)
}

func (a *API) getPlan(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	// The same filter the rest of the system takes, so the Gantt can ask for one farm's rows.
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	plan, err := a.svc.Plan.Get(r.Context(), id, f, userOf(r))
	a.respond(w, r, plan, err)
}

func (a *API) generatePlan(w http.ResponseWriter, r *http.Request) {
	var in domain.PlanInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	plan, err := a.svc.Plan.Generate(r.Context(), userOf(r), remoteOf(r), in)
	a.respond(w, r, plan, err)
}

func (a *API) planAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := intParam(r, "id")
		if err != nil {
			writeError(w, r, err, a.log)
			return
		}
		req := domain.WorkflowRequest{}
		if r.ContentLength > 0 {
			if err := decodeBody(r, &req); err != nil {
				writeError(w, r, err, a.log)
				return
			}
		}
		plan, err := a.svc.Plan.Act(r.Context(), userOf(r), remoteOf(r), id, action, req)
		a.respond(w, r, plan, err)
	}
}

func (a *API) deletePlan(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	if err := a.svc.Plan.Delete(r.Context(), userOf(r), remoteOf(r), id); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
