package api

import (
	"net/http"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// The HTTP surface of the costing: the cost structure, its rates, the currency
// quotations, and the runs that turn them into a cost per ton.

func costFilter(r *http.Request) store.CostFilter {
	q := r.URL.Query()
	return store.CostFilter{
		FactoryID: q.Get("factoryId"),
		SeasonID:  q.Get("seasonId"),
		VersionID: q.Get("versionId"),
		ElementID: q.Get("elementId"),
		RateType:  q.Get("rateType"),
		On:        domain.BusinessDate(q.Get("on")),
		Skip:      atoiOr(q.Get("$skip"), 0),
		Top:       atoiOr(q.Get("$top"), 0),
	}
}

func (s *Server) handleListCostElements(w http.ResponseWriter, r *http.Request) {
	opts := listOptions(r)
	page, err := s.costing.ListElements(r.Context(), opts)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.CostElement]{
		Value: page.Items, Count: page.Count, Skip: opts.Skip, Top: opts.Top,
	})
}

func (s *Server) handleSaveCostElement(w http.ResponseWriter, r *http.Request) {
	var e domain.CostElement
	if err := decodeJSON(w, r, &e); err != nil {
		writeProblem(w, r, err)
		return
	}
	saved, err := s.costing.SaveElement(r.Context(), e)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSON(w, saved)
}

func (s *Server) handleListCostRates(w http.ResponseWriter, r *http.Request) {
	rates, err := s.costing.ListRates(r.Context(), costFilter(r))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.CostRate]{Value: rates, Count: len(rates)})
}

func (s *Server) handleSaveCostRate(w http.ResponseWriter, r *http.Request) {
	var rate domain.CostRate
	if err := decodeJSON(w, r, &rate); err != nil {
		writeProblem(w, r, err)
		return
	}
	saved, err := s.costing.SaveRate(r.Context(), rate)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSON(w, saved)
}

func (s *Server) handleDeleteCostRate(w http.ResponseWriter, r *http.Request) {
	if err := s.costing.DeleteRate(r.Context(), r.PathValue("id")); err != nil {
		writeProblem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListExchangeRates(w http.ResponseWriter, r *http.Request) {
	rates, err := s.costing.ListExchangeRates(r.Context())
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.ExchangeRate]{Value: rates, Count: len(rates)})
}

func (s *Server) handleSaveExchangeRate(w http.ResponseWriter, r *http.Request) {
	var rate domain.ExchangeRate
	if err := decodeJSON(w, r, &rate); err != nil {
		writeProblem(w, r, err)
		return
	}
	saved, err := s.costing.SaveExchangeRate(r.Context(), rate)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, saved)
}

// handleCostRun costs a period.
//
// It is a POST because it takes a request body rather than because it changes
// anything: a run without `save` writes nothing, and the idempotency key
// applies only to the saving form.
func (s *Server) handleCostRun(w http.ResponseWriter, r *http.Request) {
	var req service.CostRunRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	if !req.Save {
		result, err := s.costing.Run(r.Context(), req)
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		writeJSON(w, result)
		return
	}
	s.postOnce(w, r, func() (any, error) {
		return s.costing.Run(r.Context(), req)
	})
}

func (s *Server) handleListCostRuns(w http.ResponseWriter, r *http.Request) {
	page, err := s.costing.ListRuns(r.Context(), costFilter(r))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, page)
}

func (s *Server) handleGetCostRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.costing.GetRun(r.Context(), r.PathValue("id"))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(run.RowVersion))
	writeJSON(w, run)
}
