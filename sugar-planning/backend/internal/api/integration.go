package api

import (
	"net/http"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// The HTTP surface of the interfaces: what this system sends out, and the two
// readings it takes in.
//
// The inbound endpoints are idempotent through the same Idempotency-Key
// mechanism as every other posting. A weighbridge terminal that lost the
// network mid-send retries the batch, and the retry replays the first answer
// rather than weighing the same lorries twice.

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.OutboxFilter{
		Topic:       q.Get("topic"),
		Unpublished: q.Get("unpublished") == "true",
		Exhausted:   q.Get("exhausted") == "true",
		Skip:        atoiOr(q.Get("$skip"), 0),
		Top:         atoiOr(q.Get("$top"), 0),
	}
	page, err := s.integration.ListEvents(r.Context(), f)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.OutboxEvent]{
		Value: page.Items, Count: page.Count, Skip: f.Skip, Top: f.Top,
	})
}

func (s *Server) handleRetryEvent(w http.ResponseWriter, r *http.Request) {
	event, err := s.integration.RetryEvent(r.Context(), r.PathValue("id"))
	if err != nil {
		// The attempt was recorded whether or not it delivered, so a failed
		// retry answers 502 with the far end's own words in the detail - which
		// is what the operator pressed the button to find out.
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, event)
}

func (s *Server) handleDispatch(w http.ResponseWriter, r *http.Request) {
	if err := requirePermission(r, domain.PermIntegrationRead); err != nil {
		writeProblem(w, r, err)
		return
	}
	result, err := s.integration.Dispatch(r.Context(), atoiOr(r.URL.Query().Get("limit"), 0))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, result)
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	runs, err := s.integration.ListJobs(r.Context())
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, map[string]any{"value": runs, "count": len(runs)})
}

func (s *Server) handleWeighbridge(w http.ResponseWriter, r *http.Request) {
	var req service.WeighbridgeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	s.postOnce(w, r, func() (any, error) {
		return s.integration.IngestWeighbridge(r.Context(), req)
	})
}

func (s *Server) handleLabResults(w http.ResponseWriter, r *http.Request) {
	var msg domain.LabResultMessage
	if err := decodeJSON(w, r, &msg); err != nil {
		writeProblem(w, r, err)
		return
	}
	s.postOnce(w, r, func() (any, error) {
		return s.integration.IngestLabResults(r.Context(), msg)
	})
}
