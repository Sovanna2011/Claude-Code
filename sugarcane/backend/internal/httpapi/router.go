package httpapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/sovanna2011/sugarcane-go/backend/internal/auth"
	"github.com/sovanna2011/sugarcane-go/backend/internal/database"
	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
	"github.com/sovanna2011/sugarcane-go/backend/internal/repository"
	"github.com/sovanna2011/sugarcane-go/backend/internal/service"
)

type API struct {
	svc     *service.Container
	tokens  *auth.Tokens
	support *repository.SupportRepository
	db      *database.DB
	log     *slog.Logger
}

func New(svc *service.Container, tokens *auth.Tokens, support *repository.SupportRepository,
	db *database.DB, log *slog.Logger) *API {
	return &API{svc: svc, tokens: tokens, support: support, db: db, log: log}
}

// Routes wires every endpoint of section 13. Go's own ServeMux matches on method and path pattern,
// so the service needs no routing dependency.
func (a *API) Routes(allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()

	// Open: the health probe and the login.
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("POST /api/auth/login", a.login)

	protected := http.NewServeMux()

	// Master data
	protected.HandleFunc("GET /api/farms", a.listFarms)
	protected.HandleFunc("POST /api/farms", a.createFarm)
	protected.HandleFunc("GET /api/farms/{id}", a.getFarm)
	protected.HandleFunc("PUT /api/farms/{id}", a.updateFarm)
	protected.HandleFunc("GET /api/farms/{id}/zones", a.zonesOfFarm)
	protected.HandleFunc("GET /api/farms/tree", a.tree)

	protected.HandleFunc("GET /api/zones", a.listZones)
	protected.HandleFunc("POST /api/zones", a.createZone)
	protected.HandleFunc("GET /api/zones/{id}", a.getZone)
	protected.HandleFunc("PUT /api/zones/{id}", a.updateZone)
	protected.HandleFunc("GET /api/zones/{id}/blocks", a.blocksOfZone)

	protected.HandleFunc("GET /api/blocks", a.listBlocks)
	protected.HandleFunc("POST /api/blocks", a.createBlock)
	protected.HandleFunc("GET /api/blocks/{id}", a.getBlock)
	protected.HandleFunc("PUT /api/blocks/{id}", a.updateBlock)
	protected.HandleFunc("GET /api/blocks/{id}/geometry", a.getGeometry)
	protected.HandleFunc("PUT /api/blocks/{id}/geometry", a.putGeometry)

	// Planning master data: seasons, varieties, the activity chain
	protected.HandleFunc("GET /api/seasons", a.listSeasons)
	protected.HandleFunc("POST /api/seasons", a.createSeason)
	protected.HandleFunc("GET /api/seasons/{id}", a.getSeason)
	protected.HandleFunc("PUT /api/seasons/{id}", a.updateSeason)

	protected.HandleFunc("GET /api/varieties", a.listVarieties)
	protected.HandleFunc("POST /api/varieties", a.createVariety)
	protected.HandleFunc("GET /api/varieties/{id}", a.getVariety)
	protected.HandleFunc("PUT /api/varieties/{id}", a.updateVariety)

	protected.HandleFunc("GET /api/activities", a.listActivities)
	protected.HandleFunc("POST /api/activities", a.createActivity)
	protected.HandleFunc("GET /api/activities/chain", a.activityChain)
	protected.HandleFunc("GET /api/activities/{id}", a.getActivity)
	protected.HandleFunc("PUT /api/activities/{id}", a.updateActivity)
	protected.HandleFunc("POST /api/activities/dependencies", a.addDependency)
	protected.HandleFunc("DELETE /api/activities/dependencies/{id}", a.deleteDependency)

	// Planting information
	protected.HandleFunc("GET /api/planting", a.listPlanting)
	protected.HandleFunc("POST /api/planting", a.savePlanting)
	protected.HandleFunc("DELETE /api/planting/{id}", a.deletePlanting)

	// Summaries
	protected.HandleFunc("GET /api/summaries/farm-area", a.farmAreaSummary)
	protected.HandleFunc("GET /api/summaries/zone-area", a.zoneAreaSummary)
	protected.HandleFunc("GET /api/summaries/block-area", a.blockAreaSummary)

	// Dashboard
	protected.HandleFunc("GET /api/dashboard/farm-area", a.dashboardKPIs)
	protected.HandleFunc("GET /api/dashboard/farm-area/map", a.dashboardMap)
	protected.HandleFunc("GET /api/dashboard/charts", a.dashboardCharts)

	// Reports
	protected.HandleFunc("GET /api/reports/farm-area-tree", a.tree)
	protected.HandleFunc("GET /api/reports/farm-area-tree.xlsx", a.treeExcel)
	protected.HandleFunc("GET /api/reports/planting-plan-vs-actual", a.planVsActual)

	// Reference data and the audit trail
	protected.HandleFunc("GET /api/lookups/{kind}", a.lookup)
	protected.HandleFunc("GET /api/land-classification", a.landClassification)
	protected.HandleFunc("GET /api/audit", a.audit)
	protected.HandleFunc("GET /api/auth/me", a.me)

	mux.Handle("/api/", Authenticate(a.tokens, a.log)(protected))

	var handler http.Handler = mux
	handler = RequestLog(a.log)(handler)
	handler = CORS(allowedOrigins)(handler)
	handler = Recover(a.log)(handler)
	return handler
}

// ---------------------------------------------------------------- health and identity

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if err := a.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unhealthy", "database": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "healthy", "time": time.Now().UTC()})
}

type loginRequest struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, r, err, a.log)
		return
	}

	user, hash, err := a.support.FindUser(r.Context(), strings.TrimSpace(req.UserName))
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	if !auth.CheckPassword(hash, req.Password) {
		writeError(w, r, domain.Unauthorized("Invalid user name or password."), a.log)
		return
	}

	token, expires, err := a.tokens.Issue(user)
	if err != nil {
		writeError(w, r, domain.Internal(err), a.log)
		return
	}
	a.support.TouchLogin(r.Context(), user.ID)

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token, "expiresAt": expires, "user": user,
	})
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userOf(r))
}

// ---------------------------------------------------------------- farms

func (a *API) listFarms(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Master.ListFarms(r.Context(), f, pageFrom(r))
	a.respond(w, r, result, err)
}

func (a *API) getFarm(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	farm, err := a.svc.Master.GetFarm(r.Context(), id)
	a.respond(w, r, farm, err)
}

func (a *API) createFarm(w http.ResponseWriter, r *http.Request) {
	var in repository.FarmInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	farm, err := a.svc.Master.SaveFarm(r.Context(), userOf(r), remoteOf(r), nil, in)
	a.respondCreated(w, r, farm, err)
}

func (a *API) updateFarm(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var in repository.FarmInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	farm, err := a.svc.Master.SaveFarm(r.Context(), userOf(r), remoteOf(r), &id, in)
	a.respond(w, r, farm, err)
}

func (a *API) zonesOfFarm(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	f.FarmID = &id
	result, err := a.svc.Master.ListZones(r.Context(), f, pageFrom(r))
	a.respond(w, r, result, err)
}

// ---------------------------------------------------------------- zones

func (a *API) listZones(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Master.ListZones(r.Context(), f, pageFrom(r))
	a.respond(w, r, result, err)
}

func (a *API) getZone(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	zone, err := a.svc.Master.GetZone(r.Context(), id)
	a.respond(w, r, zone, err)
}

func (a *API) createZone(w http.ResponseWriter, r *http.Request) {
	var in repository.ZoneInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	zone, err := a.svc.Master.SaveZone(r.Context(), userOf(r), remoteOf(r), nil, in)
	a.respondCreated(w, r, zone, err)
}

func (a *API) updateZone(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var in repository.ZoneInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	zone, err := a.svc.Master.SaveZone(r.Context(), userOf(r), remoteOf(r), &id, in)
	a.respond(w, r, zone, err)
}

func (a *API) blocksOfZone(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	f.ZoneID = &id
	result, err := a.svc.Master.ListBlocks(r.Context(), f, pageFrom(r))
	a.respond(w, r, result, err)
}

// ---------------------------------------------------------------- blocks

func (a *API) listBlocks(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Master.ListBlocks(r.Context(), f, pageFrom(r))
	a.respond(w, r, result, err)
}

func (a *API) getBlock(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	block, err := a.svc.Master.GetBlock(r.Context(), id)
	a.respond(w, r, block, err)
}

func (a *API) createBlock(w http.ResponseWriter, r *http.Request) {
	var req service.BlockSaveRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	block, err := a.svc.Master.SaveBlock(r.Context(), userOf(r), remoteOf(r), nil, req)
	a.respondCreated(w, r, block, err)
}

func (a *API) updateBlock(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var req service.BlockSaveRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	block, err := a.svc.Master.SaveBlock(r.Context(), userOf(r), remoteOf(r), &id, req)
	a.respond(w, r, block, err)
}

func (a *API) getGeometry(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	geometry, areaHa, err := a.svc.Master.Geometry(r.Context(), id)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"blockId": id, "geometry": geometry, "boundaryAreaHa": areaHa,
	})
}

type geometryRequest struct {
	Geometry json.RawMessage `json:"geometry"`
}

func (a *API) putGeometry(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var req geometryRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	areaHa, err := a.svc.Master.SetGeometry(r.Context(), userOf(r), remoteOf(r), id, req.Geometry)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"blockId": id, "boundaryAreaHa": areaHa})
}

// ---------------------------------------------------------------- planting

func (a *API) listPlanting(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	rows, err := a.svc.Master.ListPlantings(r.Context(), f)
	a.respond(w, r, rows, err)
}

func (a *API) savePlanting(w http.ResponseWriter, r *http.Request) {
	var in repository.PlantingInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	saved, err := a.svc.Master.SavePlanting(r.Context(), userOf(r), remoteOf(r), in)
	a.respond(w, r, saved, err)
}

func (a *API) deletePlanting(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	if err := a.svc.Master.DeletePlanting(r.Context(), userOf(r), remoteOf(r), id); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------- planning master data

func (a *API) listSeasons(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Activity.ListSeasons(r.Context(), f, pageFrom(r))
	a.respond(w, r, result, err)
}

func (a *API) getSeason(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	season, err := a.svc.Activity.GetSeason(r.Context(), id)
	a.respond(w, r, season, err)
}

func (a *API) createSeason(w http.ResponseWriter, r *http.Request) {
	var in domain.SeasonInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	season, err := a.svc.Activity.SaveSeason(r.Context(), userOf(r), remoteOf(r), nil, in)
	a.respondCreated(w, r, season, err)
}

func (a *API) updateSeason(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var in domain.SeasonInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	season, err := a.svc.Activity.SaveSeason(r.Context(), userOf(r), remoteOf(r), &id, in)
	a.respond(w, r, season, err)
}

func (a *API) listVarieties(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Activity.ListVarieties(r.Context(), f, pageFrom(r))
	a.respond(w, r, result, err)
}

func (a *API) getVariety(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	variety, err := a.svc.Activity.GetVariety(r.Context(), id)
	a.respond(w, r, variety, err)
}

func (a *API) createVariety(w http.ResponseWriter, r *http.Request) {
	var in domain.VarietyInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	variety, err := a.svc.Activity.SaveVariety(r.Context(), userOf(r), remoteOf(r), nil, in)
	a.respondCreated(w, r, variety, err)
}

func (a *API) updateVariety(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var in domain.VarietyInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	variety, err := a.svc.Activity.SaveVariety(r.Context(), userOf(r), remoteOf(r), &id, in)
	a.respond(w, r, variety, err)
}

func (a *API) listActivities(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Activity.ListActivities(r.Context(), f, pageFrom(r))
	a.respond(w, r, result, err)
}

func (a *API) getActivity(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	activity, err := a.svc.Activity.GetActivity(r.Context(), id)
	a.respond(w, r, activity, err)
}

func (a *API) createActivity(w http.ResponseWriter, r *http.Request) {
	var in domain.ActivityInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	activity, err := a.svc.Activity.SaveActivity(r.Context(), userOf(r), remoteOf(r), nil, in)
	a.respondCreated(w, r, activity, err)
}

func (a *API) updateActivity(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	var in domain.ActivityInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	activity, err := a.svc.Activity.SaveActivity(r.Context(), userOf(r), remoteOf(r), &id, in)
	a.respond(w, r, activity, err)
}

// The chain is the activity master in the order the engine walks it, for one crop type.
func (a *API) activityChain(w http.ResponseWriter, r *http.Request) {
	companyID := 1
	if v := optionalInt(r, "companyId"); v != nil {
		companyID = *v
	}
	chain, err := a.svc.Activity.Chain(r.Context(), companyID,
		strings.TrimSpace(r.URL.Query().Get("cropType")))
	a.respond(w, r, chain, err)
}

func (a *API) addDependency(w http.ResponseWriter, r *http.Request) {
	var in domain.DependencyInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	activity, err := a.svc.Activity.AddDependency(r.Context(), userOf(r), remoteOf(r), in)
	a.respondCreated(w, r, activity, err)
}

func (a *API) deleteDependency(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(r, "id")
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	if err := a.svc.Activity.DeleteDependency(r.Context(), userOf(r), remoteOf(r), id); err != nil {
		writeError(w, r, err, a.log)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------- dashboard and reports

func (a *API) dashboardKPIs(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Dashboard.KPIs(r.Context(), f)
	a.respond(w, r, result, err)
}

func (a *API) dashboardMap(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Dashboard.MapData(r.Context(), f)
	a.respond(w, r, result, err)
}

func (a *API) dashboardCharts(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	result, err := a.svc.Dashboard.Charts(r.Context(), f)
	a.respond(w, r, result, err)
}

func (a *API) tree(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	nodes, err := a.svc.Dashboard.Tree(r.Context(), f)
	a.respond(w, r, nodes, err)
}

func (a *API) treeExcel(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	book, err := a.svc.Report.TreeWorkbook(r.Context(), f)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	name := fmt.Sprintf("farm-area-tree-%s.xlsx", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(book)
}

func (a *API) planVsActual(w http.ResponseWriter, r *http.Request) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	level := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("level")))
	result, err := a.svc.Dashboard.PlanVsActual(r.Context(), f, level)
	a.respond(w, r, result, err)
}

// The three summary endpoints are the tree at one level each, which is what a caller integrating
// with the service usually wants rather than the whole hierarchy.
func (a *API) farmAreaSummary(w http.ResponseWriter, r *http.Request)  { a.summary(w, r, "Farm") }
func (a *API) zoneAreaSummary(w http.ResponseWriter, r *http.Request)  { a.summary(w, r, "Zone") }
func (a *API) blockAreaSummary(w http.ResponseWriter, r *http.Request) { a.summary(w, r, "Block") }

func (a *API) summary(w http.ResponseWriter, r *http.Request, level string) {
	f, err := filterFrom(r)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	nodes, err := a.svc.Dashboard.Tree(r.Context(), f)
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}

	flat := []*domain.TreeNode{}
	var walk func([]*domain.TreeNode)
	walk = func(list []*domain.TreeNode) {
		for _, n := range list {
			if n.NodeType == level {
				copied := *n
				copied.Children = nil
				flat = append(flat, &copied)
			}
			walk(n.Children)
		}
	}
	walk(nodes)
	writeJSON(w, http.StatusOK, flat)
}

// ---------------------------------------------------------------- reference data

func (a *API) lookup(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.Support.Lookup(r.Context(), r.PathValue("kind"), optionalInt(r, "parentId"))
	a.respond(w, r, items, err)
}

func (a *API) landClassification(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.Support.Lookup(r.Context(), "reasons", nil)
	a.respond(w, r, items, err)
}

func (a *API) audit(w http.ResponseWriter, r *http.Request) {
	result, err := a.svc.Support.Audit(r.Context(), userOf(r),
		strings.TrimSpace(r.URL.Query().Get("entity")), pageFrom(r))
	a.respond(w, r, result, err)
}

// ---------------------------------------------------------------- response helpers

func (a *API) respond(w http.ResponseWriter, r *http.Request, body any, err error) {
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (a *API) respondCreated(w http.ResponseWriter, r *http.Request, body any, err error) {
	if err != nil {
		writeError(w, r, err, a.log)
		return
	}
	writeJSON(w, http.StatusCreated, body)
}
