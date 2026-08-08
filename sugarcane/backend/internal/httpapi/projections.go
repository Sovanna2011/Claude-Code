package httpapi

import (
	"net/http"

	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
)

// The planting projection endpoints. The header and its lines are separate resources, and every
// step of the workflow is its own verb — POST /api/projections/{id}/approve rather than a status
// field a caller could set to anything.

func (a *API) listProjections(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Projection.List(r.Context(), f, pageFrom(r), userOf(r))
	a.respond(w, r, result, err)
}

func (a *API) getProjection(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	projection, err := a.svc.Projection.Get(r.Context(), id, userOf(r))
	a.respond(w, r, projection, err)
}

func (a *API) createProjection(w http.ResponseWriter, r *http.Request) {
	var in domain.ProjectionInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	projection, err := a.svc.Projection.Save(r.Context(), userOf(r), remoteOf(r), nil, in)
	a.respondCreated(w, r, projection, err)
}

func (a *API) updateProjection(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var in domain.ProjectionInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	projection, err := a.svc.Projection.Save(r.Context(), userOf(r), remoteOf(r), &id, in)
	a.respond(w, r, projection, err)
}

// Writing a line answers with the whole projection, not the line: the header totals have moved and
// the caller would otherwise have to fetch them separately to redraw the screen.

func (a *API) createProjectionLine(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var in domain.ProjectionLineInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	projection, err := a.svc.Projection.SaveLine(r.Context(), userOf(r), remoteOf(r), id, nil, in)
	a.respondCreated(w, r, projection, err)
}

func (a *API) updateProjectionLine(w http.ResponseWriter, r *http.Request) {
	id, lineID, err := twoIntParams(r, "id", "lineId")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var in domain.ProjectionLineInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	projection, err := a.svc.Projection.SaveLine(r.Context(), userOf(r), remoteOf(r), id, &lineID, in)
	a.respond(w, r, projection, err)
}

func (a *API) deleteProjectionLine(w http.ResponseWriter, r *http.Request) {
	id, lineID, err := twoIntParams(r, "id", "lineId")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	projection, err := a.svc.Projection.DeleteLine(r.Context(), userOf(r), remoteOf(r), id, lineID)
	a.respond(w, r, projection, err)
}

// workflowHandler builds the handler for one action, so the six steps are registered from the same
// graph the service enforces rather than written out six times.
func (a *API) workflowHandler(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := intParam(r, "id")
		if err != nil {
			writeError(w, r, err, a.log)
			return
		}
		req := domain.WorkflowRequest{}
		// The body is optional: approving needs nothing said, rejecting does, and the service is
		// what decides which. An empty body is not a malformed one.
		if r.ContentLength > 0 {
			if err := decodeBody(r, &req); err != nil {
				writeError(w, r, err, a.log)
				return
			}
		}
		projection, err := a.svc.Projection.Act(r.Context(), userOf(r), remoteOf(r), id, action, req)
		a.respond(w, r, projection, err)
	}
}

// projectionWorkflow serves the transition graph itself, so a client can render the workflow — and
// explain why a button is missing — without hard-coding a copy of it.
func (a *API) projectionWorkflow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"statuses":    domain.ValidProjectionStatuses,
		"transitions": domain.ProjectionWorkflow,
	})
}

func twoIntParams(r *http.Request, first, second string) (int, int, error) {
	a, err := intParam(r, first)
	if err != nil {
		return 0, 0, err
	}
	b, err := intParam(r, second)
	if err != nil {
		return 0, 0, err
	}
	return a, b, nil
}
