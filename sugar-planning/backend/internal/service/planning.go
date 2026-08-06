// Package service holds the application services: the use cases that sit
// between the HTTP transport and the repositories.
//
// A service is responsible for authorisation, validation, orchestration across
// repositories, transactions and the audit trail. It contains no SQL and no
// HTTP, and it delegates every calculation to the domain package.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Planning is the season and plan-version service.
type Planning struct {
	store store.Store
	now   func() time.Time
}

// NewPlanning builds the service. The clock is injected so tests are
// deterministic.
func NewPlanning(s store.Store, now func() time.Time) *Planning {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Planning{store: s, now: now}
}

// ---------------------------------------------------------------------------
// Seasons
// ---------------------------------------------------------------------------

// ListSeasons returns the seasons the caller may see.
func (p *Planning) ListSeasons(ctx context.Context, opts store.ListOptions) (store.Page[domain.Season], error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return store.Page[domain.Season]{}, err
	}
	page, err := p.store.Planning().ListSeasons(ctx, opts)
	if err != nil {
		return page, err
	}
	// The data scope is applied here rather than in the repository, so that a
	// repository can never be the thing that leaks another factory's plan.
	filtered := page.Items[:0]
	for _, s := range page.Items {
		if caller.CanSeeFactory(s.FactoryID) {
			filtered = append(filtered, s)
		}
	}
	page.Items = filtered
	page.Count = len(filtered)
	return page, nil
}

// GetSeason reads one season.
func (p *Planning) GetSeason(ctx context.Context, id string) (domain.Season, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return domain.Season{}, err
	}
	s, err := p.store.Planning().GetSeason(ctx, id)
	if err != nil {
		return domain.Season{}, err
	}
	if err := caller.RequireFactory(s.FactoryID); err != nil {
		return domain.Season{}, err
	}
	return s, nil
}

// SaveSeason creates or updates a season and opens its actuals container.
func (p *Planning) SaveSeason(ctx context.Context, s domain.Season) (domain.Season, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return domain.Season{}, err
	}
	if err := caller.RequireFactory(s.FactoryID); err != nil {
		return domain.Season{}, err
	}

	verr := &domain.ValidationError{}
	if s.Code == "" {
		verr.Add("code", "REQUIRED", "the season needs a code, for example 2026-2027")
	}
	if !s.StartDate.Valid() {
		verr.Add("startDate", "INVALID_DATE", "the start date must be an ISO date")
	}
	if s.EndDate != "" && s.EndDate < s.StartDate {
		verr.Add("endDate", "OUT_OF_RANGE", "the end date cannot be before the start date")
	}
	if s.PlannedDays < 0 {
		verr.Add("plannedDays", "NEGATIVE", "the planned length cannot be negative")
	}
	if s.Status == "" {
		s.Status = "OPEN"
	}
	if err := verr.OrNil(); err != nil {
		return domain.Season{}, err
	}

	isNew := s.ID == ""
	var saved domain.Season
	err := p.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Planning().SaveSeason(ctx, s, caller.Username)
		if err != nil {
			return err
		}
		if isNew {
			// Every season gets exactly one actuals container, created here so
			// that operators always have somewhere to post to.
			if _, err := tx.Planning().SaveVersion(ctx, domain.PlanVersion{
				SeasonID: saved.ID, Code: "ACTUAL", Description: "Recorded actuals",
				PlanType: domain.PlanTypeActual, Status: domain.StatusReleased,
				Owner: caller.Username,
			}, caller.Username); err != nil {
				return err
			}
		}
		return p.audit(ctx, tx, auditEntry{
			action: actionFor(isNew), entity: "season", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// ---------------------------------------------------------------------------
// Versions
// ---------------------------------------------------------------------------

// VersionDetail is a version with everything a planner needs to work on it.
type VersionDetail struct {
	Version        domain.PlanVersion       `json:"version"`
	Assumptions    []domain.PlanAssumption  `json:"assumptions"`
	Mix            []domain.ProductMixEntry `json:"productMix"`
	AllowedActions []domain.PlanAction      `json:"allowedActions"`
	Editable       bool                     `json:"editable"`
}

// ListVersions returns the versions of a season.
func (p *Planning) ListVersions(ctx context.Context, seasonID string, opts store.ListOptions) (store.Page[domain.PlanVersion], error) {
	if _, err := p.GetSeason(ctx, seasonID); err != nil {
		return store.Page[domain.PlanVersion]{}, err
	}
	return p.store.Planning().ListVersions(ctx, seasonID, opts)
}

// GetVersion reads a version with its assumptions and mix.
func (p *Planning) GetVersion(ctx context.Context, id string) (VersionDetail, error) {
	caller := auth.FromContext(ctx)
	v, err := p.versionInScope(ctx, id)
	if err != nil {
		return VersionDetail{}, err
	}
	assumptions, err := p.store.Planning().ListAssumptions(ctx, id)
	if err != nil {
		return VersionDetail{}, err
	}
	mix, err := p.store.Planning().ListMix(ctx, id)
	if err != nil {
		return VersionDetail{}, err
	}

	// Only offer actions the caller could actually complete, so the UI does not
	// present a button that the backend will refuse.
	var allowed []domain.PlanAction
	for _, a := range domain.AllowedActions(v.Status, v.PlanType) {
		t, err := domain.LookupTransition(v.Status, a)
		if err == nil && caller.Can(t.Permission) {
			allowed = append(allowed, a)
		}
	}
	return VersionDetail{
		Version: v, Assumptions: assumptions, Mix: mix,
		AllowedActions: allowed,
		Editable:       v.IsEditable() && caller.Can(domain.PermPlanWrite),
	}, nil
}

// versionInScope loads a version and checks read permission and data scope.
func (p *Planning) versionInScope(ctx context.Context, id string) (domain.PlanVersion, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return domain.PlanVersion{}, err
	}
	v, err := p.store.Planning().GetVersion(ctx, id)
	if err != nil {
		return domain.PlanVersion{}, err
	}
	season, err := p.store.Planning().GetSeason(ctx, v.SeasonID)
	if err != nil {
		return domain.PlanVersion{}, err
	}
	if err := caller.RequireFactory(season.FactoryID); err != nil {
		return domain.PlanVersion{}, err
	}
	return v, nil
}

// SaveVersion creates or updates a version header.
func (p *Planning) SaveVersion(ctx context.Context, v domain.PlanVersion) (domain.PlanVersion, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return domain.PlanVersion{}, err
	}
	season, err := p.store.Planning().GetSeason(ctx, v.SeasonID)
	if err != nil {
		return domain.PlanVersion{}, err
	}
	if err := caller.RequireFactory(season.FactoryID); err != nil {
		return domain.PlanVersion{}, err
	}

	verr := &domain.ValidationError{}
	if v.Code == "" {
		verr.Add("code", "REQUIRED", "the version needs a code")
	}
	if !domain.ValidPlanType(v.PlanType) {
		verr.Add("planType", "INVALID", fmt.Sprintf("%q is not a known plan type", v.PlanType))
	}
	if v.PlanType == domain.PlanTypeActual && v.ID == "" {
		verr.Add("planType", "RESERVED",
			"the actuals container is created with the season and cannot be added by hand")
	}
	if err := verr.OrNil(); err != nil {
		return domain.PlanVersion{}, err
	}

	if v.ID != "" {
		existing, err := p.store.Planning().GetVersion(ctx, v.ID)
		if err != nil {
			return domain.PlanVersion{}, err
		}
		if !existing.IsEditable() {
			return domain.PlanVersion{}, fmt.Errorf(
				"%w: version %s is %s and can only be changed through a workflow action",
				domain.ErrLocked, existing.Code, existing.Status)
		}
		// Status is owned by the workflow, not by an ordinary edit.
		v.Status = existing.Status
	} else if v.Status == "" {
		v.Status = domain.StatusDraft
	}
	if v.Owner == "" {
		v.Owner = caller.Username
	}

	isNew := v.ID == ""
	var saved domain.PlanVersion
	err = p.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Planning().SaveVersion(ctx, v, caller.Username)
		if err != nil {
			return err
		}
		return p.audit(ctx, tx, auditEntry{
			action: actionFor(isNew), entity: "plan_version", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// CopyRequest describes a scenario copy.
type CopyRequest struct {
	SourceVersionID string          `json:"sourceVersionId"`
	Code            string          `json:"code"`
	Description     string          `json:"description"`
	PlanType        domain.PlanType `json:"planType"`
	// AssumptionOverrides replace individual assumptions in the copy, which is
	// how a what-if scenario is created: copy the baseline, change the recovery
	// rate, regenerate.
	AssumptionOverrides map[string]domain.Dec `json:"assumptionOverrides,omitempty"`
	// CopyDailyRows carries the generated daily rows across. A what-if that
	// will be regenerated does not need them.
	CopyDailyRows bool `json:"copyDailyRows"`
}

// CopyVersion creates a new scenario from an existing one.
func (p *Planning) CopyVersion(ctx context.Context, req CopyRequest) (domain.PlanVersion, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return domain.PlanVersion{}, err
	}
	source, err := p.versionInScope(ctx, req.SourceVersionID)
	if err != nil {
		return domain.PlanVersion{}, err
	}
	if req.Code == "" {
		return domain.PlanVersion{}, fmt.Errorf("%w: the new version needs a code", domain.ErrValidation)
	}
	if req.PlanType == "" {
		req.PlanType = domain.PlanTypeWhatIf
	}
	if !domain.ValidPlanType(req.PlanType) {
		return domain.PlanVersion{}, fmt.Errorf("%w: %q is not a known plan type",
			domain.ErrValidation, req.PlanType)
	}
	if req.PlanType == domain.PlanTypeActual {
		return domain.PlanVersion{}, fmt.Errorf(
			"%w: a season has exactly one actuals container and it cannot be copied", domain.ErrValidation)
	}

	var created domain.PlanVersion
	err = p.store.InTx(ctx, func(tx store.Store) error {
		var err error
		created, err = tx.Planning().SaveVersion(ctx, domain.PlanVersion{
			SeasonID: source.SeasonID, Code: req.Code, Description: req.Description,
			PlanType: req.PlanType, Status: domain.StatusDraft, Owner: caller.Username,
			SourceVersion: source.ID, EffectiveFrom: source.EffectiveFrom,
			EffectiveTo: source.EffectiveTo,
		}, caller.Username)
		if err != nil {
			return err
		}

		assumptions, err := tx.Planning().ListAssumptions(ctx, source.ID)
		if err != nil {
			return err
		}
		for _, a := range assumptions {
			if override, ok := req.AssumptionOverrides[a.Code]; ok {
				a.Value = override
			}
			a.ID, a.VersionID, a.RowVersion = "", created.ID, 0
			if _, err := tx.Planning().SaveAssumption(ctx, a, caller.Username); err != nil {
				return err
			}
		}
		// An override for an assumption the source did not have is still applied,
		// so a scenario can introduce a new lever.
		for code, value := range req.AssumptionOverrides {
			if hasAssumption(assumptions, code) {
				continue
			}
			if _, err := tx.Planning().SaveAssumption(ctx, domain.PlanAssumption{
				VersionID: created.ID, Code: code, Value: value,
				Description: "Added by scenario copy",
			}, caller.Username); err != nil {
				return err
			}
		}

		mix, err := tx.Planning().ListMix(ctx, source.ID)
		if err != nil {
			return err
		}
		for _, m := range mix {
			m.ID, m.VersionID, m.RowVersion = "", created.ID, 0
			if _, err := tx.Planning().SaveMix(ctx, m, caller.Username); err != nil {
				return err
			}
		}

		if req.CopyDailyRows {
			if err := copyDailyRows(ctx, tx, source.ID, created.ID, caller.Username); err != nil {
				return err
			}
		}
		return p.audit(ctx, tx, auditEntry{
			action: "COPY", entity: "plan_version", entityID: created.ID,
			before: source, after: created,
			reason: fmt.Sprintf("copied from %s", source.Code),
		})
	})
	return created, err
}

func hasAssumption(list []domain.PlanAssumption, code string) bool {
	for _, a := range list {
		if a.Code == code {
			return true
		}
	}
	return false
}

func copyDailyRows(ctx context.Context, tx store.Store, fromID, toID, actor string) error {
	pl := tx.Planning()
	f := store.PlanFilter{VersionIDs: []string{fromID}}

	cane, err := pl.ListCane(ctx, f)
	if err != nil {
		return err
	}
	for i := range cane {
		cane[i].ID, cane[i].VersionID, cane[i].RowVersion = "", toID, 0
	}
	if _, err := pl.UpsertCane(ctx, cane, actor); err != nil {
		return err
	}

	products, err := pl.ListProducts(ctx, f)
	if err != nil {
		return err
	}
	for i := range products {
		products[i].ID, products[i].VersionID, products[i].RowVersion = "", toID, 0
	}
	if _, err := pl.UpsertProducts(ctx, products, actor); err != nil {
		return err
	}

	storageRows, err := pl.ListStorage(ctx, f)
	if err != nil {
		return err
	}
	for i := range storageRows {
		storageRows[i].ID, storageRows[i].VersionID, storageRows[i].RowVersion = "", toID, 0
	}
	if _, err := pl.UpsertStorage(ctx, storageRows, actor); err != nil {
		return err
	}

	shipments, err := pl.ListShipments(ctx, f)
	if err != nil {
		return err
	}
	for i := range shipments {
		shipments[i].ID, shipments[i].VersionID, shipments[i].RowVersion = "", toID, 0
	}
	_, err = pl.UpsertShipments(ctx, shipments, actor)
	return err
}

// ---------------------------------------------------------------------------
// Assumptions and mix
// ---------------------------------------------------------------------------

// SaveAssumption stores one planning assumption.
func (p *Planning) SaveAssumption(ctx context.Context, a domain.PlanAssumption) (domain.PlanAssumption, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return domain.PlanAssumption{}, err
	}
	v, err := p.versionInScope(ctx, a.VersionID)
	if err != nil {
		return domain.PlanAssumption{}, err
	}
	if !v.IsEditable() {
		return domain.PlanAssumption{}, fmt.Errorf("%w: version %s is %s",
			domain.ErrLocked, v.Code, v.Status)
	}
	if a.Code == "" {
		return domain.PlanAssumption{}, fmt.Errorf("%w: the assumption needs a code", domain.ErrValidation)
	}

	var saved domain.PlanAssumption
	err = p.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Planning().SaveAssumption(ctx, a, caller.Username)
		if err != nil {
			return err
		}
		return p.audit(ctx, tx, auditEntry{
			action: "UPDATE", entity: "plan_assumption", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// SaveMixEntry stores one product mix line.
func (p *Planning) SaveMixEntry(ctx context.Context, m domain.ProductMixEntry) (domain.ProductMixEntry, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return domain.ProductMixEntry{}, err
	}
	v, err := p.versionInScope(ctx, m.VersionID)
	if err != nil {
		return domain.ProductMixEntry{}, err
	}
	if !v.IsEditable() {
		return domain.ProductMixEntry{}, fmt.Errorf("%w: version %s is %s",
			domain.ErrLocked, v.Code, v.Status)
	}
	verr := &domain.ValidationError{}
	if m.ProductID == "" {
		verr.Add("productId", "REQUIRED", "the mix line needs a product")
	}
	if m.SeasonTons.IsNegative() {
		verr.Add("seasonTons", "NEGATIVE", "the season tonnage cannot be negative")
	}
	if m.DailyRateTons.IsNegative() {
		verr.Add("dailyRateTons", "NEGATIVE", "the daily rate cannot be negative")
	}
	if err := verr.OrNil(); err != nil {
		return domain.ProductMixEntry{}, err
	}

	var saved domain.ProductMixEntry
	err = p.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Planning().SaveMix(ctx, m, caller.Username)
		if err != nil {
			return err
		}
		return p.audit(ctx, tx, auditEntry{
			action: "UPDATE", entity: "product_mix", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// DeleteMixEntry removes a mix line.
func (p *Planning) DeleteMixEntry(ctx context.Context, versionID, id string) error {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return err
	}
	v, err := p.versionInScope(ctx, versionID)
	if err != nil {
		return err
	}
	if !v.IsEditable() {
		return fmt.Errorf("%w: version %s is %s", domain.ErrLocked, v.Code, v.Status)
	}
	return p.store.InTx(ctx, func(tx store.Store) error {
		if err := tx.Planning().DeleteMix(ctx, id); err != nil {
			return err
		}
		return p.audit(ctx, tx, auditEntry{action: "DELETE", entity: "product_mix", entityID: id})
	})
}

// ---------------------------------------------------------------------------
// Workflow
// ---------------------------------------------------------------------------

// TransitionRequest applies a workflow action to a version.
type TransitionRequest struct {
	Action domain.PlanAction `json:"action"`
	Reason string            `json:"reason,omitempty"`
	// LockThrough is set when releasing: dates up to and including this date
	// become read-only. Empty locks the whole version.
	LockThrough domain.BusinessDate `json:"lockThrough,omitempty"`
	RowVersion  int64               `json:"rowVersion"`
}

// Transition runs a workflow action, enforcing the state machine, the
// permission that guards it and the reason requirement.
func (p *Planning) Transition(ctx context.Context, versionID string, req TransitionRequest) (domain.PlanVersion, error) {
	caller := auth.FromContext(ctx)
	v, err := p.versionInScope(ctx, versionID)
	if err != nil {
		return domain.PlanVersion{}, err
	}
	if req.RowVersion != 0 && req.RowVersion != v.RowVersion {
		return domain.PlanVersion{}, fmt.Errorf("%w: version %s is at row version %d",
			domain.ErrConflict, v.Code, v.RowVersion)
	}

	transition, err := domain.ApplyTransition(v, req.Action, req.Reason)
	if err != nil {
		return domain.PlanVersion{}, err
	}
	if err := caller.Require(transition.Permission); err != nil {
		return domain.PlanVersion{}, err
	}
	// Separation of duties: the owner of a plan cannot approve their own work.
	if req.Action == domain.ActionApprove && v.Owner == caller.Username {
		return domain.PlanVersion{}, fmt.Errorf(
			"%w: %s submitted this plan and cannot also approve it", domain.ErrForbidden, caller.Username)
	}

	before := v
	now := p.now()
	v.Status = transition.To
	switch req.Action {
	case domain.ActionSubmit:
		v.SubmittedAt = &now
	case domain.ActionApprove:
		v.ApprovedAt, v.ApprovedBy = &now, caller.Username
	case domain.ActionRelease:
		v.ReleasedAt = &now
		v.LockedThrough = req.LockThrough
	case domain.ActionReopen:
		// Reopening clears the approval trail: the plan has to earn it again.
		v.ApprovedAt, v.ApprovedBy, v.ReleasedAt, v.SubmittedAt = nil, "", nil, nil
		v.LockedThrough = ""
	}
	if req.Reason != "" {
		v.Comment = req.Reason
	}

	var saved domain.PlanVersion
	err = p.store.InTx(ctx, func(tx store.Store) error {
		// Releasing supersedes the version that was released before, so a season
		// only ever has one live baseline.
		if req.Action == domain.ActionRelease {
			others, err := tx.Planning().ListVersions(ctx, v.SeasonID, store.ListOptions{Top: 1000})
			if err != nil {
				return err
			}
			for _, other := range others.Items {
				if other.ID == v.ID || other.Status != domain.StatusReleased {
					continue
				}
				other.Status = domain.StatusSuperseded
				other.Comment = fmt.Sprintf("superseded by %s", v.Code)
				if _, err := tx.Planning().SaveVersion(ctx, other, caller.Username); err != nil {
					return err
				}
				if err := p.audit(ctx, tx, auditEntry{
					action: string(domain.ActionSupersede), entity: "plan_version", entityID: other.ID,
					reason: fmt.Sprintf("superseded by %s", v.Code),
				}); err != nil {
					return err
				}
			}
		}

		var err error
		saved, err = tx.Planning().SaveVersion(ctx, v, caller.Username)
		if err != nil {
			return err
		}
		if err := p.audit(ctx, tx, auditEntry{
			action: string(req.Action), entity: "plan_version", entityID: saved.ID,
			before: before, after: saved, reason: req.Reason,
		}); err != nil {
			return err
		}

		// Release is the only transition worth telling another system about.
		// A plan in review is this department's business; a released plan is
		// what the factory and the ERP are expected to work to.
		if req.Action != domain.ActionRelease {
			return nil
		}
		event := PlanReleasedEvent{
			VersionID: saved.ID, VersionNo: saved.VersionNo, SeasonID: saved.SeasonID,
			PlanType:    saved.PlanType,
			Description: saved.Description, ReleasedBy: caller.Username,
		}
		if season, err := tx.Planning().GetSeason(ctx, saved.SeasonID); err == nil {
			event.SeasonCode, event.FactoryID = season.Code, season.FactoryID
			event.ValidFrom, event.ValidTo = season.StartDate, season.EndDate
		}
		return emit(ctx, tx, domain.TopicPlanReleased, event)
	})
	return saved, err
}

// ---------------------------------------------------------------------------
// Audit helper
// ---------------------------------------------------------------------------

type auditEntry struct {
	action   string
	entity   string
	entityID string
	before   any
	after    any
	reason   string
}

// audit writes one audit record inside the caller's transaction, so the trail
// is committed with the change it describes or not at all.
func (p *Planning) audit(ctx context.Context, tx store.Store, e auditEntry) error {
	return writeAudit(ctx, tx, p.now, e)
}

// writeAudit is the shared implementation, used by every service. It is a free
// function rather than a method so that a new service cannot accidentally grow
// its own idea of what an audit record looks like.
func writeAudit(ctx context.Context, tx store.Store, now func() time.Time, e auditEntry) error {
	caller := auth.FromContext(ctx)
	return tx.Audit().Append(ctx, domain.AuditEvent{
		OccurredAt: now(), Actor: caller.Username, Action: e.action,
		Entity: e.entity, EntityID: e.entityID,
		Before: encodeState(e.before), After: encodeState(e.after),
		Reason: e.reason, CorrelationID: CorrelationFromContext(ctx),
		SourceIP: SourceIPFromContext(ctx),
	})
}

func encodeState(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func actionFor(isNew bool) string {
	if isNew {
		return "CREATE"
	}
	return "UPDATE"
}

// sortedDates returns the distinct dates of a set of rows, in order.
func sortedDates(dates map[domain.BusinessDate]bool) []domain.BusinessDate {
	out := make([]domain.BusinessDate, 0, len(dates))
	for d := range dates {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
