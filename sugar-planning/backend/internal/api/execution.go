package api

import (
	"net/http"
	"strings"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// This file is the HTTP surface of the daily factory: stock, postings,
// production orders, confirmations, the laboratory and the maintenance
// calendar. As everywhere else in this package the handlers only decode,
// delegate and encode; every rule lives in the service and the domain.

// executionFilter reads the allow-listed query parameters of an execution
// endpoint. Nothing here is passed through to SQL: each field names a column
// the repository knows about.
func executionFilter(r *http.Request) store.ExecutionFilter {
	q := r.URL.Query()
	return store.ExecutionFilter{
		FactoryID:   q.Get("factoryId"),
		VersionID:   q.Get("versionId"),
		From:        domain.BusinessDate(q.Get("from")),
		To:          domain.BusinessDate(q.Get("to")),
		ProductIDs:  splitCSV(q.Get("productId")),
		WarehouseID: q.Get("warehouseId"),
		LineID:      q.Get("lineId"),
		OrderID:     q.Get("orderId"),
		Statuses:    upperCSV(q.Get("status")),
		OpenOnly:    q.Get("openOnly") == "true",
		Skip:        atoiOr(q.Get("$skip"), 0),
		Top:         atoiOr(q.Get("$top"), 0),
	}
}

func upperCSV(s string) []string {
	values := splitCSV(s)
	for i := range values {
		values[i] = strings.ToUpper(values[i])
	}
	return values
}

// ---------------------------------------------------------------------------
// Stock and postings
// ---------------------------------------------------------------------------

func (s *Server) handleStock(w http.ResponseWriter, r *http.Request) {
	lines, err := s.execution.Stock(r.Context(), executionFilter(r))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[service.StockLine]{Value: lines, Count: len(lines)})
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	page, err := s.execution.ListDocuments(r.Context(), executionFilter(r))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, page)
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	document, err := s.execution.GetDocument(r.Context(), r.PathValue("id"))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, document)
}

// handlePostDocument moves stock.
//
// A posting is the one request in the application that must never happen twice
// on a retry, so it honours the idempotency key: a repeated key replays the
// first response instead of moving the balance again.
func (s *Server) handlePostDocument(w http.ResponseWriter, r *http.Request) {
	var req service.PostingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	s.postOnce(w, r, func() (any, error) {
		return s.execution.Post(r.Context(), req)
	})
}

func (s *Server) handleReverseDocument(w http.ResponseWriter, r *http.Request) {
	var req service.ReversalRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	s.postOnce(w, r, func() (any, error) {
		return s.execution.Reverse(r.Context(), r.PathValue("id"), req)
	})
}

// ---------------------------------------------------------------------------
// Production orders
// ---------------------------------------------------------------------------

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	page, err := s.execution.ListOrders(r.Context(), executionFilter(r))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, page)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	detail, err := s.execution.GetOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(detail.Order.RowVersion))
	writeJSON(w, detail)
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req service.OrderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	order, err := s.execution.CreateOrder(r.Context(), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(order.RowVersion))
	writeJSONStatus(w, http.StatusCreated, order)
}

func (s *Server) handleOrdersFromPlan(w http.ResponseWriter, r *http.Request) {
	var req service.OrdersFromPlanRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	s.postOnce(w, r, func() (any, error) {
		return s.execution.CreateOrdersFromPlan(r.Context(), r.PathValue("id"), req)
	})
}

func (s *Server) handleOrderAction(w http.ResponseWriter, r *http.Request) {
	var req service.OrderActionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	if v, err := ifMatch(r); err != nil {
		writeProblem(w, r, err)
		return
	} else if v != 0 {
		req.RowVersion = v
	}
	order, err := s.execution.ActOnOrder(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(order.RowVersion))
	writeJSON(w, order)
}

func (s *Server) handleConfirmOrder(w http.ResponseWriter, r *http.Request) {
	var req service.ConfirmRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	s.postOnce(w, r, func() (any, error) {
		return s.execution.Confirm(r.Context(), r.PathValue("id"), req)
	})
}

func (s *Server) handleReverseConfirmation(w http.ResponseWriter, r *http.Request) {
	var req service.ReversalRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	s.postOnce(w, r, func() (any, error) {
		return s.execution.ReverseConfirmation(r.Context(), r.PathValue("id"), req)
	})
}

// ---------------------------------------------------------------------------
// Quality
// ---------------------------------------------------------------------------

func (s *Server) handleListQualityParameters(w http.ResponseWriter, r *http.Request) {
	parameters, err := s.execution.ListParameters(r.Context())
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.QualityParameter]{Value: parameters, Count: len(parameters)})
}

func (s *Server) handleSaveQualityParameter(w http.ResponseWriter, r *http.Request) {
	var p domain.QualityParameter
	if err := decodeJSON(w, r, &p); err != nil {
		writeProblem(w, r, err)
		return
	}
	saved, err := s.execution.SaveParameter(r.Context(), p)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, saved)
}

func (s *Server) handleListQualitySpecs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	on := domain.BusinessDate(q.Get("on"))
	if on == "" {
		on = domain.NewBusinessDate(s.now())
	}
	specs, err := s.execution.SpecsFor(r.Context(), q.Get("productId"), on)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.QualitySpec]{Value: specs, Count: len(specs)})
}

func (s *Server) handleSaveQualitySpec(w http.ResponseWriter, r *http.Request) {
	var spec domain.QualitySpec
	if err := decodeJSON(w, r, &spec); err != nil {
		writeProblem(w, r, err)
		return
	}
	saved, err := s.execution.SaveSpec(r.Context(), spec)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, saved)
}

func (s *Server) handleListSamples(w http.ResponseWriter, r *http.Request) {
	page, err := s.execution.ListSamples(r.Context(), executionFilter(r))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, page)
}

func (s *Server) handleGetSample(w http.ResponseWriter, r *http.Request) {
	sample, err := s.execution.GetSample(r.Context(), r.PathValue("id"))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, sample)
}

func (s *Server) handleCreateSample(w http.ResponseWriter, r *http.Request) {
	var req service.SampleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	sample, err := s.execution.CreateSample(r.Context(), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSONStatus(w, http.StatusCreated, sample)
}

func (s *Server) handleRecordResults(w http.ResponseWriter, r *http.Request) {
	var req service.ResultsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	outcome, err := s.execution.RecordResults(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, outcome)
}

func (s *Server) handleListHolds(w http.ResponseWriter, r *http.Request) {
	holds, err := s.execution.ListHolds(r.Context(), executionFilter(r))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.QualityHold]{Value: holds, Count: len(holds)})
}

func (s *Server) handlePlaceHold(w http.ResponseWriter, r *http.Request) {
	var req service.HoldRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	s.postOnce(w, r, func() (any, error) {
		return s.execution.PlaceHold(r.Context(), req)
	})
}

func (s *Server) handleReleaseHold(w http.ResponseWriter, r *http.Request) {
	var req service.ReleaseRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	if v, err := ifMatch(r); err != nil {
		writeProblem(w, r, err)
		return
	} else if v != 0 {
		req.RowVersion = v
	}
	hold, err := s.execution.ReleaseHold(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(hold.RowVersion))
	writeJSON(w, hold)
}

// ---------------------------------------------------------------------------
// Maintenance
// ---------------------------------------------------------------------------

func (s *Server) handleListMaintenance(w http.ResponseWriter, r *http.Request) {
	windows, err := s.execution.ListMaintenance(r.Context(), executionFilter(r))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.MaintenanceWindow]{Value: windows, Count: len(windows)})
}

func (s *Server) handleSaveMaintenance(w http.ResponseWriter, r *http.Request) {
	var m domain.MaintenanceWindow
	if err := decodeJSON(w, r, &m); err != nil {
		writeProblem(w, r, err)
		return
	}
	if v, err := ifMatch(r); err != nil {
		writeProblem(w, r, err)
		return
	} else if v != 0 {
		m.RowVersion = v
	}
	saved, err := s.execution.SaveMaintenance(r.Context(), m)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSON(w, saved)
}
