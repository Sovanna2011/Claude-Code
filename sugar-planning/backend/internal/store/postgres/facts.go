package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// factWhere builds the WHERE clause for a plan filter. Only the columns the
// table actually has are considered, and every value is a bound parameter -
// there is no string concatenation of client input anywhere.
type factWhere struct {
	clauses []string
	args    []any
}

func (w *factWhere) add(col string, value any) {
	w.args = append(w.args, value)
	w.clauses = append(w.clauses, fmt.Sprintf("%s = $%d", col, len(w.args)))
}

func (w *factWhere) addIn(col string, values []string) {
	if len(values) == 0 {
		return
	}
	// A blank id is not a uuid, and PostgreSQL says so with a 22P02 rather than
	// returning nothing. Drop the blanks; if that empties the list, match
	// nothing rather than everything, because a filter that was asked for and
	// cannot be applied must not quietly widen the query.
	wanted := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" {
			wanted = append(wanted, v)
		}
	}
	if len(wanted) == 0 {
		w.clauses = append(w.clauses, "false")
		return
	}
	w.args = append(w.args, wanted)
	w.clauses = append(w.clauses, fmt.Sprintf("%s = ANY($%d)", col, len(w.args)))
}

func (w *factWhere) addRange(col string, from, to domain.BusinessDate) {
	if from != "" {
		w.args = append(w.args, nd(from))
		w.clauses = append(w.clauses, fmt.Sprintf("%s >= $%d", col, len(w.args)))
	}
	if to != "" {
		w.args = append(w.args, nd(to))
		w.clauses = append(w.clauses, fmt.Sprintf("%s <= $%d", col, len(w.args)))
	}
}

func (w *factWhere) sql() string {
	if len(w.clauses) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w.clauses, " AND ")
}

func (w *factWhere) limit(f store.PlanFilter) string {
	top := f.Top
	if top <= 0 || top > 100000 {
		top = 100000 // a full season of daily rows for one dimension
	}
	w.args = append(w.args, top, f.Skip)
	return fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(w.args)-1, len(w.args))
}

// ---------------------------------------------------------------------------
// Cane
// ---------------------------------------------------------------------------

const caneCols = `id, version_id, factory_id, business_date, shift_id, series,
	cane_available, cane_delivered, cane_accepted, cane_rejected, cane_diverted, cane_crushed,
	crush_rate_tph, available_hrs, stoppage_hrs, reason_code, note,
	created_at, created_by, updated_at, updated_by, row_version`

func (p planning) ListCane(ctx context.Context, f store.PlanFilter) ([]domain.DailyCanePlan, error) {
	w := &factWhere{}
	w.addIn("version_id", f.VersionIDs)
	if f.FactoryID != "" {
		w.add("factory_id", f.FactoryID)
	}
	if f.Series != "" {
		w.add("series", string(f.Series))
	}
	w.addRange("business_date", f.From, f.To)
	clause := w.sql()
	limit := w.limit(f)

	rows, err := p.s.q.Query(ctx,
		"SELECT "+caneCols+" FROM daily_cane_plans"+clause+" ORDER BY business_date, shift_id"+limit, w.args...)
	if err != nil {
		return nil, mapError("daily cane plan", err)
	}
	defer rows.Close()

	var out []domain.DailyCanePlan
	for rows.Next() {
		var r domain.DailyCanePlan
		var date time.Time
		var shift, reason *string
		var series string
		if err := rows.Scan(&r.ID, &r.VersionID, &r.FactoryID, &date, &shift, &series,
			&r.CaneAvailable, &r.CaneDelivered, &r.CaneAccepted, &r.CaneRejected, &r.CaneDiverted,
			&r.CaneCrushed, &r.CrushRateTPH, &r.AvailableHrs, &r.StoppageHrs, &reason, &r.Note,
			&r.CreatedAt, &r.CreatedBy, &r.UpdatedAt, &r.UpdatedBy, &r.RowVersion); err != nil {
			return nil, mapError("daily cane plan", err)
		}
		r.BusinessDate, r.ShiftID, r.ReasonCode, r.Series = mustDate(date), ds(shift), ds(reason), domain.Series(series)
		out = append(out, r)
	}
	return out, mapError("daily cane plan", rows.Err())
}

func (p planning) UpsertCane(ctx context.Context, rows []domain.DailyCanePlan, actor string) (int, error) {
	now := nowUTC()
	const stmt = `INSERT INTO daily_cane_plans
		(id, version_id, factory_id, business_date, shift_id, series,
		 cane_available, cane_delivered, cane_accepted, cane_rejected, cane_diverted, cane_crushed,
		 crush_rate_tph, available_hrs, stoppage_hrs, reason_code, note,
		 created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$18,$19,1)
		ON CONFLICT (version_id, factory_id, business_date,
			COALESCE(shift_id, '00000000-0000-0000-0000-000000000000'::uuid), series)
		DO UPDATE SET
			cane_available = EXCLUDED.cane_available, cane_delivered = EXCLUDED.cane_delivered,
			cane_accepted = EXCLUDED.cane_accepted, cane_rejected = EXCLUDED.cane_rejected,
			cane_diverted = EXCLUDED.cane_diverted, cane_crushed = EXCLUDED.cane_crushed,
			crush_rate_tph = EXCLUDED.crush_rate_tph, available_hrs = EXCLUDED.available_hrs,
			stoppage_hrs = EXCLUDED.stoppage_hrs, reason_code = EXCLUDED.reason_code,
			note = EXCLUDED.note, updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by,
			row_version = daily_cane_plans.row_version + 1`

	for i, r := range rows {
		if r.ID == "" {
			r.ID = uuid.NewString()
		}
		if _, err := p.s.q.Exec(ctx, stmt, r.ID, r.VersionID, r.FactoryID, nd(r.BusinessDate),
			nu(r.ShiftID), string(r.Series), r.CaneAvailable, r.CaneDelivered, r.CaneAccepted,
			r.CaneRejected, r.CaneDiverted, r.CaneCrushed, r.CrushRateTPH, r.AvailableHrs,
			r.StoppageHrs, nu(r.ReasonCode), r.Note, now, actor); err != nil {
			return i, mapError(fmt.Sprintf("daily cane plan for %s", r.BusinessDate), err)
		}
	}
	return len(rows), nil
}

// ---------------------------------------------------------------------------
// Products
// ---------------------------------------------------------------------------

const productPlanCols = `id, version_id, factory_id, line_id, business_date, shift_id, product_id,
	packaging_id, series, quantity, remelt_input, process_loss, rework, rejected, hold_qty,
	reason_code, note, created_at, created_by, updated_at, updated_by, row_version`

func (p planning) ListProducts(ctx context.Context, f store.PlanFilter) ([]domain.DailyProductPlan, error) {
	w := &factWhere{}
	w.addIn("version_id", f.VersionIDs)
	w.addIn("product_id", f.ProductIDs)
	w.addIn("line_id", f.LineIDs)
	if f.FactoryID != "" {
		w.add("factory_id", f.FactoryID)
	}
	if f.Series != "" {
		w.add("series", string(f.Series))
	}
	w.addRange("business_date", f.From, f.To)
	clause := w.sql()
	limit := w.limit(f)

	rows, err := p.s.q.Query(ctx,
		"SELECT "+productPlanCols+" FROM daily_product_plans"+clause+
			" ORDER BY business_date, product_id"+limit, w.args...)
	if err != nil {
		return nil, mapError("daily product plan", err)
	}
	defer rows.Close()

	var out []domain.DailyProductPlan
	for rows.Next() {
		var r domain.DailyProductPlan
		var date time.Time
		var line, shift, pack, reason *string
		var series string
		if err := rows.Scan(&r.ID, &r.VersionID, &r.FactoryID, &line, &date, &shift, &r.ProductID,
			&pack, &series, &r.Quantity, &r.RemeltInput, &r.ProcessLoss, &r.Rework, &r.Rejected,
			&r.HoldQty, &reason, &r.Note,
			&r.CreatedAt, &r.CreatedBy, &r.UpdatedAt, &r.UpdatedBy, &r.RowVersion); err != nil {
			return nil, mapError("daily product plan", err)
		}
		r.BusinessDate, r.LineID, r.ShiftID = mustDate(date), ds(line), ds(shift)
		r.PackagingID, r.ReasonCode, r.Series = ds(pack), ds(reason), domain.Series(series)
		out = append(out, r)
	}
	return out, mapError("daily product plan", rows.Err())
}

func (p planning) UpsertProducts(ctx context.Context, rows []domain.DailyProductPlan, actor string) (int, error) {
	now := nowUTC()
	const stmt = `INSERT INTO daily_product_plans
		(id, version_id, factory_id, line_id, business_date, shift_id, product_id, packaging_id,
		 series, quantity, remelt_input, process_loss, rework, rejected, hold_qty, reason_code, note,
		 created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$18,$19,1)
		ON CONFLICT (version_id, factory_id,
			COALESCE(line_id, '00000000-0000-0000-0000-000000000000'::uuid), business_date,
			COALESCE(shift_id, '00000000-0000-0000-0000-000000000000'::uuid), product_id,
			COALESCE(packaging_id, '00000000-0000-0000-0000-000000000000'::uuid), series)
		DO UPDATE SET
			quantity = EXCLUDED.quantity, remelt_input = EXCLUDED.remelt_input,
			process_loss = EXCLUDED.process_loss, rework = EXCLUDED.rework,
			rejected = EXCLUDED.rejected, hold_qty = EXCLUDED.hold_qty,
			reason_code = EXCLUDED.reason_code, note = EXCLUDED.note,
			updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by,
			row_version = daily_product_plans.row_version + 1`

	for i, r := range rows {
		if r.ID == "" {
			r.ID = uuid.NewString()
		}
		if _, err := p.s.q.Exec(ctx, stmt, r.ID, r.VersionID, r.FactoryID, nu(r.LineID),
			nd(r.BusinessDate), nu(r.ShiftID), r.ProductID, nu(r.PackagingID), string(r.Series),
			r.Quantity, r.RemeltInput, r.ProcessLoss, r.Rework, r.Rejected, r.HoldQty,
			nu(r.ReasonCode), r.Note, now, actor); err != nil {
			return i, mapError(fmt.Sprintf("daily product plan for %s", r.BusinessDate), err)
		}
	}
	return len(rows), nil
}

// ---------------------------------------------------------------------------
// Storage
// ---------------------------------------------------------------------------

const storageCols = `id, version_id, warehouse_id, product_id, business_date, series,
	beginning_balance, production_receipt, transfer_in, transfer_out, repack_in, repack_out,
	remelt_issue, shipment_qty, adjustment, process_loss, hold_qty, ending_balance, physical_balance,
	created_at, created_by, updated_at, updated_by, row_version`

func (p planning) ListStorage(ctx context.Context, f store.PlanFilter) ([]domain.DailyStoragePlan, error) {
	w := &factWhere{}
	w.addIn("version_id", f.VersionIDs)
	w.addIn("product_id", f.ProductIDs)
	w.addIn("warehouse_id", f.WarehouseIDs)
	if f.Series != "" {
		w.add("series", string(f.Series))
	}
	w.addRange("business_date", f.From, f.To)
	clause := w.sql()
	limit := w.limit(f)

	rows, err := p.s.q.Query(ctx,
		"SELECT "+storageCols+" FROM daily_storage_plans"+clause+
			" ORDER BY warehouse_id, product_id, business_date"+limit, w.args...)
	if err != nil {
		return nil, mapError("daily storage plan", err)
	}
	defer rows.Close()

	var out []domain.DailyStoragePlan
	for rows.Next() {
		var r domain.DailyStoragePlan
		var date time.Time
		var series string
		var physical *domain.Dec
		if err := rows.Scan(&r.ID, &r.VersionID, &r.WarehouseID, &r.ProductID, &date, &series,
			&r.BeginningBalance, &r.ProductionReceipt, &r.TransferIn, &r.TransferOut,
			&r.RepackIn, &r.RepackOut, &r.RemeltIssue, &r.ShipmentQty, &r.Adjustment,
			&r.ProcessLoss, &r.HoldQty, &r.EndingBalance, &physical,
			&r.CreatedAt, &r.CreatedBy, &r.UpdatedAt, &r.UpdatedBy, &r.RowVersion); err != nil {
			return nil, mapError("daily storage plan", err)
		}
		r.BusinessDate, r.Series, r.PhysicalBalance = mustDate(date), domain.Series(series), physical
		out = append(out, r)
	}
	return out, mapError("daily storage plan", rows.Err())
}

func (p planning) UpsertStorage(ctx context.Context, rows []domain.DailyStoragePlan, actor string) (int, error) {
	now := nowUTC()
	const stmt = `INSERT INTO daily_storage_plans
		(id, version_id, warehouse_id, product_id, business_date, series, beginning_balance,
		 production_receipt, transfer_in, transfer_out, repack_in, repack_out, remelt_issue,
		 shipment_qty, adjustment, process_loss, hold_qty, ending_balance, physical_balance,
		 created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$20,$21,1)
		ON CONFLICT (version_id, warehouse_id, product_id, business_date, series)
		DO UPDATE SET
			beginning_balance = EXCLUDED.beginning_balance,
			production_receipt = EXCLUDED.production_receipt, transfer_in = EXCLUDED.transfer_in,
			transfer_out = EXCLUDED.transfer_out, repack_in = EXCLUDED.repack_in,
			repack_out = EXCLUDED.repack_out, remelt_issue = EXCLUDED.remelt_issue,
			shipment_qty = EXCLUDED.shipment_qty, adjustment = EXCLUDED.adjustment,
			process_loss = EXCLUDED.process_loss, hold_qty = EXCLUDED.hold_qty,
			ending_balance = EXCLUDED.ending_balance, physical_balance = EXCLUDED.physical_balance,
			updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by,
			row_version = daily_storage_plans.row_version + 1`

	for i, r := range rows {
		if r.ID == "" {
			r.ID = uuid.NewString()
		}
		var physical any
		if r.PhysicalBalance != nil {
			physical = *r.PhysicalBalance
		}
		if _, err := p.s.q.Exec(ctx, stmt, r.ID, r.VersionID, r.WarehouseID, r.ProductID,
			nd(r.BusinessDate), string(r.Series), r.BeginningBalance, r.ProductionReceipt,
			r.TransferIn, r.TransferOut, r.RepackIn, r.RepackOut, r.RemeltIssue, r.ShipmentQty,
			r.Adjustment, r.ProcessLoss, r.HoldQty, r.EndingBalance, physical, now, actor); err != nil {
			return i, mapError(fmt.Sprintf("daily storage plan for %s", r.BusinessDate), err)
		}
	}
	return len(rows), nil
}

// ---------------------------------------------------------------------------
// Shipments
// ---------------------------------------------------------------------------

const shipmentCols = `id, version_id, warehouse_id, product_id, channel_id, business_date, series,
	quantity, note, created_at, created_by, updated_at, updated_by, row_version`

func (p planning) ListShipments(ctx context.Context, f store.PlanFilter) ([]domain.DailyShipmentPlan, error) {
	w := &factWhere{}
	w.addIn("version_id", f.VersionIDs)
	w.addIn("product_id", f.ProductIDs)
	w.addIn("warehouse_id", f.WarehouseIDs)
	w.addIn("channel_id", f.ChannelIDs)
	if f.Series != "" {
		w.add("series", string(f.Series))
	}
	w.addRange("business_date", f.From, f.To)
	clause := w.sql()
	limit := w.limit(f)

	rows, err := p.s.q.Query(ctx,
		"SELECT "+shipmentCols+" FROM daily_shipment_plans"+clause+
			" ORDER BY business_date, channel_id"+limit, w.args...)
	if err != nil {
		return nil, mapError("daily shipment plan", err)
	}
	defer rows.Close()

	var out []domain.DailyShipmentPlan
	for rows.Next() {
		var r domain.DailyShipmentPlan
		var date time.Time
		var wh *string
		var series string
		if err := rows.Scan(&r.ID, &r.VersionID, &wh, &r.ProductID, &r.ChannelID, &date, &series,
			&r.Quantity, &r.Note,
			&r.CreatedAt, &r.CreatedBy, &r.UpdatedAt, &r.UpdatedBy, &r.RowVersion); err != nil {
			return nil, mapError("daily shipment plan", err)
		}
		r.BusinessDate, r.WarehouseID, r.Series = mustDate(date), ds(wh), domain.Series(series)
		out = append(out, r)
	}
	return out, mapError("daily shipment plan", rows.Err())
}

func (p planning) UpsertShipments(ctx context.Context, rows []domain.DailyShipmentPlan, actor string) (int, error) {
	now := nowUTC()
	const stmt = `INSERT INTO daily_shipment_plans
		(id, version_id, warehouse_id, product_id, channel_id, business_date, series, quantity, note,
		 created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$10,$11,1)
		ON CONFLICT (version_id, COALESCE(warehouse_id, '00000000-0000-0000-0000-000000000000'::uuid),
			product_id, channel_id, business_date, series)
		DO UPDATE SET quantity = EXCLUDED.quantity, note = EXCLUDED.note,
			updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by,
			row_version = daily_shipment_plans.row_version + 1`

	for i, r := range rows {
		if r.ID == "" {
			r.ID = uuid.NewString()
		}
		if _, err := p.s.q.Exec(ctx, stmt, r.ID, r.VersionID, nu(r.WarehouseID), r.ProductID,
			r.ChannelID, nd(r.BusinessDate), string(r.Series), r.Quantity, r.Note, now, actor); err != nil {
			return i, mapError(fmt.Sprintf("daily shipment plan for %s", r.BusinessDate), err)
		}
	}
	return len(rows), nil
}

func (p planning) DeleteVersionRows(ctx context.Context, versionID string) error {
	for _, table := range []string{
		"daily_cane_plans", "daily_product_plans", "daily_storage_plans", "daily_shipment_plans",
	} {
		if _, err := p.s.q.Exec(ctx, "DELETE FROM "+table+" WHERE version_id = $1", versionID); err != nil {
			return mapError(table, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Downtime
// ---------------------------------------------------------------------------

func (p planning) ListDowntime(ctx context.Context, f store.PlanFilter) ([]domain.DowntimeEvent, error) {
	w := &factWhere{}
	if f.FactoryID != "" {
		w.add("factory_id", f.FactoryID)
	}
	w.addIn("line_id", f.LineIDs)
	w.addRange("business_date", f.From, f.To)
	clause := w.sql()
	limit := w.limit(f)

	rows, err := p.s.q.Query(ctx, `SELECT id, factory_id, line_id, business_date, shift_id,
		start_at, end_at, duration_hrs, planned, reason_code, root_cause, team, corrective_action,
		created_at, created_by, updated_at, updated_by, row_version
		FROM downtime_events`+clause+" ORDER BY business_date, start_at"+limit, w.args...)
	if err != nil {
		return nil, mapError("downtime event", err)
	}
	defer rows.Close()

	var out []domain.DowntimeEvent
	for rows.Next() {
		var e domain.DowntimeEvent
		var date time.Time
		var line, shift *string
		if err := rows.Scan(&e.ID, &e.FactoryID, &line, &date, &shift, &e.StartAt, &e.EndAt,
			&e.DurationHrs, &e.Planned, &e.ReasonCode, &e.RootCause, &e.Team, &e.Action,
			&e.CreatedAt, &e.CreatedBy, &e.UpdatedAt, &e.UpdatedBy, &e.RowVersion); err != nil {
			return nil, mapError("downtime event", err)
		}
		e.BusinessDate, e.LineID, e.ShiftID = mustDate(date), ds(line), ds(shift)
		out = append(out, e)
	}
	return out, mapError("downtime event", rows.Err())
}

func (p planning) SaveDowntime(ctx context.Context, e domain.DowntimeEvent, actor string) (domain.DowntimeEvent, error) {
	now := nowUTC()
	if e.ID == "" {
		e.ID = uuid.NewString()
		e.CreatedAt, e.CreatedBy, e.UpdatedAt, e.UpdatedBy, e.RowVersion = now, actor, now, actor, 1
		_, err := p.s.q.Exec(ctx, `INSERT INTO downtime_events
			(id, factory_id, line_id, business_date, shift_id, start_at, end_at, duration_hrs,
			 planned, reason_code, root_cause, team, corrective_action,
			 created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$14,$15,1)`,
			e.ID, e.FactoryID, nu(e.LineID), nd(e.BusinessDate), nu(e.ShiftID), e.StartAt, e.EndAt,
			e.DurationHrs, e.Planned, e.ReasonCode, e.RootCause, e.Team, e.Action, now, actor)
		if err != nil {
			return domain.DowntimeEvent{}, mapError("downtime event", err)
		}
		return e, nil
	}
	var created time.Time
	var createdBy string
	var version int64
	err := p.s.q.QueryRow(ctx, `UPDATE downtime_events SET
			line_id=$1, business_date=$2, shift_id=$3, start_at=$4, end_at=$5, duration_hrs=$6,
			planned=$7, reason_code=$8, root_cause=$9, team=$10, corrective_action=$11,
			updated_at=$12, updated_by=$13, row_version = row_version + 1
		WHERE id=$14 AND row_version=$15 RETURNING created_at, created_by, row_version`,
		nu(e.LineID), nd(e.BusinessDate), nu(e.ShiftID), e.StartAt, e.EndAt, e.DurationHrs,
		e.Planned, e.ReasonCode, e.RootCause, e.Team, e.Action, now, actor, e.ID, e.RowVersion).
		Scan(&created, &createdBy, &version)
	if err != nil {
		if isNoRows(err) {
			return domain.DowntimeEvent{}, versionConflict(ctx, p.s, "downtime_events", "downtime event", e.ID, e.RowVersion)
		}
		return domain.DowntimeEvent{}, mapError("downtime event", err)
	}
	e.CreatedAt, e.CreatedBy, e.UpdatedAt, e.UpdatedBy, e.RowVersion = created, createdBy, now, actor, version
	return e, nil
}
