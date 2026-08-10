package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/sovanna2011/sugarcane-go/backend/internal/database"
	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
)

// LookupItem is a value the filter bar offers. Everything the user can filter on is served from
// the database rather than hard-coded in the front end, so a new variety appears in the dropdown
// the moment it is created.
type LookupItem struct {
	ID      int    `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Parent  *int   `json:"parentId,omitempty"`
	Display string `json:"display"`
}

type SupportRepository struct{ db *database.DB }

func NewSupportRepository(db *database.DB) *SupportRepository { return &SupportRepository{db: db} }

// lookupQueries is the whitelist. A kind that is not in this map is rejected rather than
// interpolated into SQL.
var lookupQueries = map[string]string{
	"companies":     `SELECT id, code, name, NULL::int FROM company WHERE active ORDER BY code`,
	"plantations":   `SELECT id, code, name, company_id FROM plantation WHERE active AND ($1::int IS NULL OR company_id = $1) ORDER BY code`,
	"farms":         `SELECT f.id, f.code, f.name, f.plantation_id FROM farm f WHERE f.active AND ($1::int IS NULL OR f.plantation_id = $1) ORDER BY f.code`,
	"zones":         `SELECT z.id, z.code, z.name, z.farm_id FROM zone z WHERE z.active AND ($1::int IS NULL OR z.farm_id = $1) ORDER BY z.code`,
	"blocks":        `SELECT b.id, b.code, b.name, b.zone_id FROM block b WHERE b.active AND ($1::int IS NULL OR b.zone_id = $1) ORDER BY b.code`,
	"seasons":       `SELECT id, code, name, crop_year FROM crop_season WHERE active ORDER BY crop_year DESC`,
	"varieties":     `SELECT id, code, name, NULL::int FROM cane_variety WHERE active ORDER BY code`,
	"reasons":       `SELECT 0, code, name, NULL::int FROM non_plantable_reason WHERE active ORDER BY sort_order`,
	"cropYears":     `SELECT DISTINCT crop_year, crop_year::text, crop_year::text, NULL::int FROM block_planting ORDER BY 1 DESC`,
	"plantingYears": `SELECT DISTINCT planting_year, planting_year::text, planting_year::text, NULL::int FROM block_planting ORDER BY 1 DESC`,
}

func (r *SupportRepository) Lookup(ctx context.Context, kind string, parentID *int) ([]LookupItem, error) {
	query, ok := lookupQueries[kind]
	if !ok {
		return nil, domain.BadRequest("UNKNOWN_LOOKUP", fmt.Sprintf("There is no lookup called %q.", kind))
	}

	var rows pgx.Rows
	var err error
	if wantsParent(query) {
		rows, err = r.db.Pool().Query(ctx, query, parentID)
	} else {
		rows, err = r.db.Pool().Query(ctx, query)
	}
	if err != nil {
		return nil, fmt.Errorf("lookup %s: %w", kind, err)
	}
	defer rows.Close()

	items := []LookupItem{}
	for rows.Next() {
		var it LookupItem
		if err := rows.Scan(&it.ID, &it.Code, &it.Name, &it.Parent); err != nil {
			return nil, err
		}
		it.Display = it.Name
		if it.Code != "" && it.Code != it.Name {
			it.Display = it.Code + " — " + it.Name
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func wantsParent(query string) bool {
	return len(query) > 0 && containsPlaceholder(query)
}

func containsPlaceholder(q string) bool {
	for i := 0; i+1 < len(q); i++ {
		if q[i] == '$' && q[i+1] == '1' {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- users

func (r *SupportRepository) FindUser(ctx context.Context, username string) (domain.User, string, error) {
	var u domain.User
	var hash string
	var active bool
	err := r.db.Pool().QueryRow(ctx,
		`SELECT id, username, full_name, role, password_hash, active FROM app_user WHERE username = $1`,
		username).Scan(&u.ID, &u.Username, &u.FullName, &u.Role, &hash, &active)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !active) {
		// An unknown user and a disabled one answer identically, so the endpoint cannot be used
		// to discover which accounts exist.
		return domain.User{}, "", domain.Unauthorized("Invalid user name or password.")
	}
	return u, hash, err
}

func (r *SupportRepository) TouchLogin(ctx context.Context, id int) {
	_, _ = r.db.Pool().Exec(ctx, `UPDATE app_user SET last_login_at = now() WHERE id = $1`, id)
}

// ---------------------------------------------------------------- audit

// AuditEntry is one line of the functional trail: who changed what, when, from where.
type AuditEntry struct {
	ID         int64  `json:"id"`
	At         string `json:"at"`
	Actor      string `json:"actor"`
	Action     string `json:"action"`
	Entity     string `json:"entity"`
	EntityID   string `json:"entityId"`
	Before     any    `json:"before,omitempty"`
	After      any    `json:"after,omitempty"`
	RemoteAddr string `json:"remoteAddr,omitempty"`
}

// Write records a change. It takes the transaction, so the audit row commits or rolls back with
// the change it describes — an audit trail that survives a rolled-back write is a lie.
func (r *SupportRepository) Write(ctx context.Context, tx database.Querier, actor, action, entity, entityID string, before, after any, remote string) error {
	beforeJSON, err := marshalOrNil(before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalOrNil(after)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_log(actor, action, entity, entity_id, before_data, after_data, remote_addr)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		actor, action, entity, entityID, beforeJSON, afterJSON, nullIfEmpty(remote))
	return err
}

func (r *SupportRepository) ListAudit(ctx context.Context, entity string, page domain.Page) ([]AuditEntry, int, error) {
	b := &builder{}
	if entity != "" {
		b.eq("entity", entity)
	}

	var total int
	if err := r.db.Pool().QueryRow(ctx, "SELECT count(*) FROM audit_log"+b.whereSQL(), b.args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, at::text, actor, action, entity, COALESCE(entity_id, ''),
	                 before_data, after_data, COALESCE(remote_addr, '')
	            FROM audit_log` + b.whereSQL() +
		fmt.Sprintf(" ORDER BY at DESC, id DESC LIMIT %s OFFSET %s", b.add(page.Size), b.add(page.Offset()))

	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var before, after []byte
		if err := rows.Scan(&e.ID, &e.At, &e.Actor, &e.Action, &e.Entity, &e.EntityID,
			&before, &after, &e.RemoteAddr); err != nil {
			return nil, 0, err
		}
		if len(before) > 0 {
			_ = json.Unmarshal(before, &e.Before)
		}
		if len(after) > 0 {
			_ = json.Unmarshal(after, &e.After)
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func marshalOrNil(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// unmarshalDependencies reads the blocking predecessors the planning query aggregates into JSON.
// Doing it in one query rather than one per activity keeps the generator's input to a single round
// trip, which matters when the master runs to nineteen activities.
func unmarshalDependencies(raw []byte, into *[]domain.PlanDependency) error {
	if len(raw) == 0 {
		return nil
	}
	var parsed []struct {
		DependsOnID int  `json:"dependsOnId"`
		LagDays     int  `json:"lagDays"`
		IsBlocking  bool `json:"isBlocking"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("activity dependencies: %w", err)
	}
	for _, p := range parsed {
		*into = append(*into, domain.PlanDependency{
			DependsOnID: p.DependsOnID, LagDays: p.LagDays, IsBlocking: p.IsBlocking})
	}
	return nil
}

// sortStrings keeps the holiday list in a stable order, so two plans with the same calendar store
// it identically and a diff of the two rows shows nothing.
func sortStrings(values []string) {
	sort.Strings(values)
}
