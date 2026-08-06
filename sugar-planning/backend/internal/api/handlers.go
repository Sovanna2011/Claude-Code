package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// ---------------------------------------------------------------------------
// Master data
// ---------------------------------------------------------------------------

// registerMasterData wires the four standard routes for one master entity.
// Writing this once means every master entity has identical paging, ETag and
// permission behaviour, and a new entity is one line of registration.
func registerMasterData[T any](mux *http.ServeMux, path string, repo store.Repo[T]) {
	base := "/api/v1/master/" + path

	mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
		if err := requirePermission(r, domain.PermMasterDataRead); err != nil {
			writeProblem(w, r, err)
			return
		}
		opts := listOptions(r)
		page, err := repo.List(r.Context(), opts)
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		writeJSON(w, pageResponse[T]{Value: page.Items, Count: page.Count, Skip: opts.Skip, Top: opts.Top})
	})

	mux.HandleFunc("GET "+base+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := requirePermission(r, domain.PermMasterDataRead); err != nil {
			writeProblem(w, r, err)
			return
		}
		entity, err := repo.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		if v, ok := any(entity).(interface{ Version() int64 }); ok {
			w.Header().Set("ETag", etagFor(v.Version()))
		}
		writeJSON(w, entity)
	})

	mux.HandleFunc("POST "+base, func(w http.ResponseWriter, r *http.Request) {
		if err := requirePermission(r, domain.PermMasterDataWrite); err != nil {
			writeProblem(w, r, err)
			return
		}
		var entity T
		if err := decodeJSON(w, r, &entity); err != nil {
			writeProblem(w, r, err)
			return
		}
		saved, err := repo.Save(r.Context(), entity, auth.FromContext(r.Context()).Username)
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		writeJSONStatus(w, http.StatusCreated, saved)
	})

	mux.HandleFunc("PUT "+base+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := requirePermission(r, domain.PermMasterDataWrite); err != nil {
			writeProblem(w, r, err)
			return
		}
		var entity T
		if err := decodeJSON(w, r, &entity); err != nil {
			writeProblem(w, r, err)
			return
		}
		saved, err := repo.Save(r.Context(), entity, auth.FromContext(r.Context()).Username)
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		writeJSON(w, saved)
	})

	mux.HandleFunc("DELETE "+base+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := requirePermission(r, domain.PermMasterDataWrite); err != nil {
			writeProblem(w, r, err)
			return
		}
		// Deactivation is destructive enough to insist on the row version, so a
		// stale browser tab cannot deactivate a record somebody else just edited.
		rowVersion, err := requireIfMatch(r)
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		if err := repo.Deactivate(r.Context(), r.PathValue("id"), rowVersion,
			auth.FromContext(r.Context()).Username); err != nil {
			writeProblem(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// ---------------------------------------------------------------------------
// Seasons
// ---------------------------------------------------------------------------

func (s *Server) handleListSeasons(w http.ResponseWriter, r *http.Request) {
	opts := listOptions(r)
	page, err := s.planning.ListSeasons(r.Context(), opts)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.Season]{
		Value: page.Items, Count: page.Count, Skip: opts.Skip, Top: opts.Top,
	})
}

func (s *Server) handleGetSeason(w http.ResponseWriter, r *http.Request) {
	season, err := s.planning.GetSeason(r.Context(), r.PathValue("id"))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(season.RowVersion))
	writeJSON(w, season)
}

func (s *Server) handleCreateSeason(w http.ResponseWriter, r *http.Request) {
	var season domain.Season
	if err := decodeJSON(w, r, &season); err != nil {
		writeProblem(w, r, err)
		return
	}
	season.ID = ""
	saved, err := s.planning.SaveSeason(r.Context(), season)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSONStatus(w, http.StatusCreated, saved)
}

func (s *Server) handleUpdateSeason(w http.ResponseWriter, r *http.Request) {
	var season domain.Season
	if err := decodeJSON(w, r, &season); err != nil {
		writeProblem(w, r, err)
		return
	}
	season.ID = r.PathValue("id")
	if v, err := ifMatch(r); err != nil {
		writeProblem(w, r, err)
		return
	} else if v != 0 {
		season.RowVersion = v
	}
	saved, err := s.planning.SaveSeason(r.Context(), season)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSON(w, saved)
}

// ---------------------------------------------------------------------------
// Versions
// ---------------------------------------------------------------------------

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	opts := listOptions(r)
	page, err := s.planning.ListVersions(r.Context(), r.PathValue("id"), opts)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.PlanVersion]{
		Value: page.Items, Count: page.Count, Skip: opts.Skip, Top: opts.Top,
	})
}

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	detail, err := s.planning.GetVersion(r.Context(), r.PathValue("id"))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(detail.Version.RowVersion))
	writeJSON(w, detail)
}

func (s *Server) handleCreateVersion(w http.ResponseWriter, r *http.Request) {
	var version domain.PlanVersion
	if err := decodeJSON(w, r, &version); err != nil {
		writeProblem(w, r, err)
		return
	}
	version.ID, version.SeasonID = "", r.PathValue("id")
	saved, err := s.planning.SaveVersion(r.Context(), version)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSONStatus(w, http.StatusCreated, saved)
}

func (s *Server) handleUpdateVersion(w http.ResponseWriter, r *http.Request) {
	var version domain.PlanVersion
	if err := decodeJSON(w, r, &version); err != nil {
		writeProblem(w, r, err)
		return
	}
	version.ID = r.PathValue("id")
	if v, err := ifMatch(r); err != nil {
		writeProblem(w, r, err)
		return
	} else if v != 0 {
		version.RowVersion = v
	}
	saved, err := s.planning.SaveVersion(r.Context(), version)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSON(w, saved)
}

func (s *Server) handleCopyVersion(w http.ResponseWriter, r *http.Request) {
	var req service.CopyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	req.SourceVersionID = r.PathValue("id")
	created, err := s.planning.CopyVersion(r.Context(), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSONStatus(w, http.StatusCreated, created)
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req service.GenerateRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &req); err != nil {
			writeProblem(w, r, err)
			return
		}
	}
	// Generating a season plan is expensive and repeatable; an idempotency key
	// makes a retried request return the first result instead of rebuilding.
	s.postOnce(w, r, func() (any, error) {
		return s.planning.Generate(r.Context(), r.PathValue("id"), req)
	})
}

// postOnce runs a write, honouring the Idempotency-Key header.
//
// The key is claimed before the work starts, so two concurrent retries of the
// same request cannot both do it; the response is attached afterwards, so a
// later retry replays the document the first request produced rather than a
// bare acknowledgement. A request without the header is simply run.
func (s *Server) postOnce(w http.ResponseWriter, r *http.Request, run func() (any, error)) {
	key := r.Header.Get("Idempotency-Key")
	if key != "" {
		fresh, previous, err := s.store.Idempotency().Remember(r.Context(), key, r.URL.Path, nil)
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		if !fresh {
			w.Header().Set("Idempotent-Replay", "true")
			if len(previous) > 0 {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_, _ = w.Write(previous)
				return
			}
			writeJSONStatus(w, http.StatusOK, map[string]string{
				"status": "the request with this idempotency key was already processed",
			})
			return
		}
	}

	result, err := run()
	if err != nil {
		writeProblem(w, r, err)
		return
	}

	body, marshalErr := json.Marshal(result)
	if key != "" && marshalErr == nil {
		// Failing to record the response is not a reason to fail a posting that
		// has already been committed. The key stays claimed either way, so the
		// retry is still refused; it just replays the acknowledgement.
		if err := s.store.Idempotency().Complete(r.Context(), key, r.URL.Path, body); err != nil {
			logError(r, err)
		}
	}
	if marshalErr != nil {
		writeProblem(w, r, marshalErr)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(body)
}

func (s *Server) handleTransition(w http.ResponseWriter, r *http.Request) {
	var req service.TransitionRequest
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
	saved, err := s.planning.Transition(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSON(w, saved)
}

func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	var req service.CompareRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	result, err := s.planning.Compare(r.Context(), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, result)
}

func (s *Server) handleSaveAssumption(w http.ResponseWriter, r *http.Request) {
	var a domain.PlanAssumption
	if err := decodeJSON(w, r, &a); err != nil {
		writeProblem(w, r, err)
		return
	}
	a.VersionID = r.PathValue("id")
	saved, err := s.planning.SaveAssumption(r.Context(), a)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, saved)
}

func (s *Server) handleSaveMix(w http.ResponseWriter, r *http.Request) {
	var m domain.ProductMixEntry
	if err := decodeJSON(w, r, &m); err != nil {
		writeProblem(w, r, err)
		return
	}
	m.VersionID = r.PathValue("id")
	saved, err := s.planning.SaveMixEntry(r.Context(), m)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, saved)
}

func (s *Server) handleDeleteMix(w http.ResponseWriter, r *http.Request) {
	if err := s.planning.DeleteMixEntry(r.Context(), r.PathValue("id"), r.PathValue("mixId")); err != nil {
		writeProblem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Daily plan rows
// ---------------------------------------------------------------------------

func (s *Server) handleListCane(w http.ResponseWriter, r *http.Request) {
	versionID := r.PathValue("id")
	if _, err := s.planning.GetVersion(r.Context(), versionID); err != nil {
		writeProblem(w, r, err)
		return
	}
	rows, err := s.store.Planning().ListCane(r.Context(), planFilter(r, versionID))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.DailyCanePlan]{Value: rows, Count: len(rows)})
}

func (s *Server) handleListProduction(w http.ResponseWriter, r *http.Request) {
	versionID := r.PathValue("id")
	if _, err := s.planning.GetVersion(r.Context(), versionID); err != nil {
		writeProblem(w, r, err)
		return
	}
	rows, err := s.store.Planning().ListProducts(r.Context(), planFilter(r, versionID))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.DailyProductPlan]{Value: rows, Count: len(rows)})
}

func (s *Server) handleListStorage(w http.ResponseWriter, r *http.Request) {
	versionID := r.PathValue("id")
	if _, err := s.planning.GetVersion(r.Context(), versionID); err != nil {
		writeProblem(w, r, err)
		return
	}
	rows, err := s.store.Planning().ListStorage(r.Context(), planFilter(r, versionID))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.DailyStoragePlan]{Value: rows, Count: len(rows)})
}

func (s *Server) handleListShipments(w http.ResponseWriter, r *http.Request) {
	versionID := r.PathValue("id")
	if _, err := s.planning.GetVersion(r.Context(), versionID); err != nil {
		writeProblem(w, r, err)
		return
	}
	rows, err := s.store.Planning().ListShipments(r.Context(), planFilter(r, versionID))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.DailyShipmentPlan]{Value: rows, Count: len(rows)})
}

// bulkRequest is the shape of every daily-row write. The rows are typed per
// endpoint; the options are common.
type bulkRequest[T any] struct {
	Rows    []T  `json:"rows"`
	Partial bool `json:"partial,omitempty"`
}

func handleBulk[T any](w http.ResponseWriter, r *http.Request,
	apply func(versionID string, rows []T, opts service.UpsertOptions) (service.UpsertResult, error)) {

	var req bulkRequest[T]
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, r, err)
		return
	}
	if len(req.Rows) == 0 {
		writeProblem(w, r, wrapValidation("the request contains no rows"))
		return
	}
	const maxRows = 20000
	if len(req.Rows) > maxRows {
		writeProblem(w, r, wrapValidation(
			fmt.Sprintf("a single request carries at most %d rows; send the plan in batches", maxRows)))
		return
	}

	result, err := apply(r.PathValue("id"), req.Rows, service.UpsertOptions{Partial: req.Partial})
	if err != nil {
		// A validation failure still returns the row-level detail, which is what
		// the planning grid needs to highlight the offending cells.
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, result)
}

func (s *Server) handleUpsertCane(w http.ResponseWriter, r *http.Request) {
	handleBulk(w, r, func(id string, rows []domain.DailyCanePlan, o service.UpsertOptions) (service.UpsertResult, error) {
		return s.planning.UpsertCane(r.Context(), id, rows, o)
	})
}

func (s *Server) handleUpsertProduction(w http.ResponseWriter, r *http.Request) {
	handleBulk(w, r, func(id string, rows []domain.DailyProductPlan, o service.UpsertOptions) (service.UpsertResult, error) {
		return s.planning.UpsertProduction(r.Context(), id, rows, o)
	})
}

func (s *Server) handleUpsertStorage(w http.ResponseWriter, r *http.Request) {
	handleBulk(w, r, func(id string, rows []domain.DailyStoragePlan, o service.UpsertOptions) (service.UpsertResult, error) {
		return s.planning.UpsertStorage(r.Context(), id, rows, o)
	})
}

func (s *Server) handleUpsertShipments(w http.ResponseWriter, r *http.Request) {
	handleBulk(w, r, func(id string, rows []domain.DailyShipmentPlan, o service.UpsertOptions) (service.UpsertResult, error) {
		return s.planning.UpsertShipments(r.Context(), id, rows, o)
	})
}

// ---------------------------------------------------------------------------
// Analytics, materials, downtime and audit
// ---------------------------------------------------------------------------

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := service.DashboardRequest{
		SeasonID:      q.Get("seasonId"),
		PlanVersionID: q.Get("versionId"),
		From:          domain.BusinessDate(q.Get("from")),
		To:            domain.BusinessDate(q.Get("to")),
		AsOf:          domain.BusinessDate(q.Get("asOf")),
		RollingWindow: atoiOr(q.Get("rollingWindow"), 7),
	}
	if req.SeasonID == "" {
		writeProblem(w, r, wrapValidation("seasonId is required"))
		return
	}
	dash, err := s.analytics.Dashboard(r.Context(), req)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, dash)
}

func (s *Server) handleMaterialRequirements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	reqs, err := s.materials.Requirements(r.Context(), service.RequirementsRequest{
		VersionID: r.PathValue("id"),
		From:      domain.BusinessDate(q.Get("from")),
		To:        domain.BusinessDate(q.Get("to")),
	})
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[service.MaterialRequirement]{Value: reqs, Count: len(reqs)})
}

func (s *Server) handleListDowntime(w http.ResponseWriter, r *http.Request) {
	if err := requirePermission(r, domain.PermPlanRead); err != nil {
		writeProblem(w, r, err)
		return
	}
	events, err := s.store.Planning().ListDowntime(r.Context(), planFilter(r, ""))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.DowntimeEvent]{Value: events, Count: len(events)})
}

func (s *Server) handleSaveDowntime(w http.ResponseWriter, r *http.Request) {
	if err := requirePermission(r, domain.PermDowntimeWrite); err != nil {
		writeProblem(w, r, err)
		return
	}
	var e domain.DowntimeEvent
	if err := decodeJSON(w, r, &e); err != nil {
		writeProblem(w, r, err)
		return
	}
	caller := auth.FromContext(r.Context())
	if err := caller.RequireFactory(e.FactoryID); err != nil {
		writeProblem(w, r, err)
		return
	}
	if e.EndAt.Before(e.StartAt) {
		writeProblem(w, r, wrapValidation("the downtime end time cannot be before the start time"))
		return
	}
	if e.DurationHrs.IsZero() && !e.EndAt.IsZero() {
		// Derive the duration exactly from whole minutes rather than through a
		// float, so 8 hours 20 minutes is 8.333 and not 8.333333333333334.
		minutes := int64(e.EndAt.Sub(e.StartAt) / time.Minute)
		e.DurationHrs = domain.RoundRate(domain.DI(minutes).Div(domain.DI(60)))
	}
	saved, err := s.store.Planning().SaveDowntime(r.Context(), e, caller.Username)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSONStatus(w, http.StatusCreated, saved)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if err := requirePermission(r, domain.PermAuditRead); err != nil {
		writeProblem(w, r, err)
		return
	}
	q := r.URL.Query()
	page, err := s.store.Audit().List(r.Context(), store.AuditFilter{
		Entity: q.Get("entity"), EntityID: q.Get("entityId"),
		Actor: q.Get("actor"), Action: q.Get("action"),
		Skip: atoiOr(q.Get("$skip"), 0), Top: atoiOr(q.Get("$top"), 100),
	})
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.AuditEvent]{
		Value: page.Items, Count: page.Count,
		Skip: atoiOr(q.Get("$skip"), 0), Top: atoiOr(q.Get("$top"), 100),
	})
}
