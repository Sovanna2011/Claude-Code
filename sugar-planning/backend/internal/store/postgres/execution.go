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

type execution struct{ s *Store }

// Execution returns the execution repository.
func (s *Store) Execution() store.Execution { return execution{s} }

// execWhere builds the WHERE clause for an execution filter, on the same
// allow-listed basis as the planning filters: named columns, bound parameters,
// no client-supplied SQL.
type execWhere struct {
	clauses []string
	args    []any
}

func (w *execWhere) eq(col string, value any) {
	w.args = append(w.args, value)
	w.clauses = append(w.clauses, fmt.Sprintf("%s = $%d", col, len(w.args)))
}

func (w *execWhere) in(col string, values []string) {
	if len(values) == 0 {
		return
	}
	w.args = append(w.args, values)
	w.clauses = append(w.clauses, fmt.Sprintf("%s = ANY($%d)", col, len(w.args)))
}

func (w *execWhere) dateRange(col string, from, to domain.BusinessDate) {
	if from != "" {
		w.args = append(w.args, nd(from))
		w.clauses = append(w.clauses, fmt.Sprintf("%s >= $%d", col, len(w.args)))
	}
	if to != "" {
		w.args = append(w.args, nd(to))
		w.clauses = append(w.clauses, fmt.Sprintf("%s <= $%d", col, len(w.args)))
	}
}

func (w *execWhere) raw(clause string) { w.clauses = append(w.clauses, clause) }

func (w *execWhere) sql() string {
	if len(w.clauses) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w.clauses, " AND ")
}

func (w *execWhere) limit(f store.ExecutionFilter) string {
	top := f.Top
	if top <= 0 || top > 5000 {
		top = 500
	}
	w.args = append(w.args, top, f.Skip)
	return fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(w.args)-1, len(w.args))
}

// ---------------------------------------------------------------------------
// Production orders
// ---------------------------------------------------------------------------

const orderCols = `id, order_no, company_id, factory_id, line_id, version_id, business_date,
	shift_id, product_id, packaging_id, planned_qty, confirmed_qty, batch_id, bom_version,
	planned_start, planned_end, priority, status, team, variance_reason,
	created_at, created_by, updated_at, updated_by, row_version`

func scanOrder(r scanner) (domain.ProductionOrder, error) {
	var o domain.ProductionOrder
	var date time.Time
	var line, version, shift, packaging, batch, variance *string
	var status string
	err := r.Scan(&o.ID, &o.OrderNo, &o.CompanyID, &o.FactoryID, &line, &version, &date,
		&shift, &o.ProductID, &packaging, &o.PlannedQty, &o.ConfirmedQty, &batch, &o.BOMVersion,
		&o.PlannedStart, &o.PlannedEnd, &o.Priority, &status, &o.Team, &variance,
		&o.CreatedAt, &o.CreatedBy, &o.UpdatedAt, &o.UpdatedBy, &o.RowVersion)
	o.BusinessDate, o.Status = mustDate(date), domain.OrderStatus(status)
	o.LineID, o.VersionID, o.ShiftID = ds(line), ds(version), ds(shift)
	o.PackagingID, o.BatchID, o.VarianceReason = ds(packaging), ds(batch), ds(variance)
	return o, err
}

func (e execution) ListOrders(ctx context.Context, f store.ExecutionFilter) (store.Page[domain.ProductionOrder], error) {
	w := &execWhere{}
	if f.FactoryID != "" {
		w.eq("factory_id", f.FactoryID)
	}
	if f.VersionID != "" {
		w.eq("version_id", f.VersionID)
	}
	if f.LineID != "" {
		w.eq("line_id", f.LineID)
	}
	w.in("product_id", f.ProductIDs)
	w.in("status", f.Statuses)
	w.dateRange("business_date", f.From, f.To)
	if f.OpenOnly {
		w.raw("status IN ('PLANNED','RELEASED','IN_PROCESS','PARTIALLY_CONFIRMED')")
	}
	clause := w.sql()

	var total int
	if err := e.s.q.QueryRow(ctx, "SELECT count(*) FROM production_orders"+clause, w.args...).
		Scan(&total); err != nil {
		return store.Page[domain.ProductionOrder]{}, mapError("production order", err)
	}
	limit := w.limit(f)

	rows, err := e.s.q.Query(ctx,
		"SELECT "+orderCols+" FROM production_orders"+clause+" ORDER BY order_no"+limit, w.args...)
	if err != nil {
		return store.Page[domain.ProductionOrder]{}, mapError("production order", err)
	}
	defer rows.Close()

	items := []domain.ProductionOrder{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return store.Page[domain.ProductionOrder]{}, mapError("production order", err)
		}
		items = append(items, o)
	}
	return store.Page[domain.ProductionOrder]{Items: items, Count: total},
		mapError("production order", rows.Err())
}

func (e execution) GetOrder(ctx context.Context, id string) (domain.ProductionOrder, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.ProductionOrder{}, fmt.Errorf("%w: production order %s", domain.ErrNotFound, id)
	}
	o, err := scanOrder(e.s.q.QueryRow(ctx,
		"SELECT "+orderCols+" FROM production_orders WHERE id = $1", id))
	return o, mapError("production order "+id, err)
}

func (e execution) SaveOrder(ctx context.Context, o domain.ProductionOrder, actor string) (domain.ProductionOrder, error) {
	now := nowUTC()
	if o.Priority == 0 {
		o.Priority = 5
	}
	if o.ID == "" {
		o.ID = uuid.NewString()
		o.CreatedAt, o.CreatedBy, o.UpdatedAt, o.UpdatedBy, o.RowVersion = now, actor, now, actor, 1
		_, err := e.s.q.Exec(ctx, `INSERT INTO production_orders
			(id, order_no, company_id, factory_id, line_id, version_id, business_date, shift_id,
			 product_id, packaging_id, planned_qty, confirmed_qty, batch_id, bom_version,
			 planned_start, planned_end, priority, status, team, variance_reason,
			 created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			        $21,$22,$21,$22,1)`,
			o.ID, o.OrderNo, o.CompanyID, o.FactoryID, nu(o.LineID), nu(o.VersionID),
			nd(o.BusinessDate), nu(o.ShiftID), o.ProductID, nu(o.PackagingID),
			o.PlannedQty, o.ConfirmedQty, nu(o.BatchID), o.BOMVersion,
			o.PlannedStart, o.PlannedEnd, o.Priority, string(o.Status), o.Team,
			nu(o.VarianceReason), now, actor)
		if err != nil {
			return domain.ProductionOrder{}, mapError("production order "+o.OrderNo, err)
		}
		return o, nil
	}

	var created time.Time
	var createdBy string
	var version int64
	err := e.s.q.QueryRow(ctx, `UPDATE production_orders SET
			line_id=$1, business_date=$2, shift_id=$3, product_id=$4, packaging_id=$5,
			planned_qty=$6, confirmed_qty=$7, batch_id=$8, bom_version=$9,
			planned_start=$10, planned_end=$11, priority=$12, status=$13, team=$14,
			variance_reason=$15, updated_at=$16, updated_by=$17, row_version = row_version + 1
		WHERE id=$18 AND row_version=$19
		RETURNING created_at, created_by, row_version`,
		nu(o.LineID), nd(o.BusinessDate), nu(o.ShiftID), o.ProductID, nu(o.PackagingID),
		o.PlannedQty, o.ConfirmedQty, nu(o.BatchID), o.BOMVersion,
		o.PlannedStart, o.PlannedEnd, o.Priority, string(o.Status), o.Team,
		nu(o.VarianceReason), now, actor, o.ID, o.RowVersion).
		Scan(&created, &createdBy, &version)
	if err != nil {
		if isNoRows(err) {
			return domain.ProductionOrder{}, versionConflict(ctx, e.s,
				"production_orders", "production order", o.ID, o.RowVersion)
		}
		return domain.ProductionOrder{}, mapError("production order "+o.OrderNo, err)
	}
	o.CreatedAt, o.CreatedBy, o.UpdatedAt, o.UpdatedBy, o.RowVersion = created, createdBy, now, actor, version
	return o, nil
}

// numberSeries maps a series to the table and column its numbers live in. It
// is an allow list, not a lookup: the table name goes into the SQL text, so it
// may only ever come from here.
var numberSeries = map[string][2]string{
	store.SeriesOrder:        {"production_orders", "order_no"},
	store.SeriesConfirmation: {"production_confirmations", "confirmation_no"},
	store.SeriesDocument:     {"inventory_documents", "document_no"},
	store.SeriesSample:       {"quality_samples", "sample_no"},
}

func (e execution) NextNumber(ctx context.Context, series, factoryCode string, year int) (string, error) {
	target, ok := numberSeries[series]
	if !ok {
		return "", fmt.Errorf("%w: %q is not a known document series", domain.ErrValidation, series)
	}
	table, column := target[0], target[1]

	prefix := fmt.Sprintf("%s-%s-%d-", series, factoryCode, year)
	var highest *int
	// The numbers are matched on the prefix and the suffix read back off the
	// end, so a number somebody typed by hand in another format is ignored
	// rather than crashing the cast.
	err := e.s.q.QueryRow(ctx, fmt.Sprintf(`
		SELECT max(substring(%s from '([0-9]+)$')::int)
		FROM %s WHERE %s LIKE $1`, column, table, column), prefix+"%").Scan(&highest)
	if err != nil {
		return "", mapError("document number for series "+series, err)
	}
	next := 1
	if highest != nil {
		next = *highest + 1
	}
	return fmt.Sprintf("%s%05d", prefix, next), nil
}

// ---------------------------------------------------------------------------
// Confirmations
// ---------------------------------------------------------------------------

const confirmationCols = `id, order_id, confirmation_no, business_date, shift_id, yield_qty,
	scrap_qty, rework_qty, labour_hours, machine_hours, batch_id, reversal_of, reversed,
	reason_code, created_at, created_by, updated_at, updated_by, row_version`

func scanConfirmation(r scanner) (domain.ProductionConfirmation, error) {
	var c domain.ProductionConfirmation
	var date time.Time
	var shift, batch, reversalOf, reason *string
	err := r.Scan(&c.ID, &c.OrderID, &c.ConfirmationNo, &date, &shift, &c.YieldQty,
		&c.ScrapQty, &c.ReworkQty, &c.LabourHours, &c.MachineHours, &batch, &reversalOf,
		&c.Reversed, &reason, &c.CreatedAt, &c.CreatedBy, &c.UpdatedAt, &c.UpdatedBy, &c.RowVersion)
	c.BusinessDate = mustDate(date)
	c.ShiftID, c.BatchID, c.ReversalOf, c.ReasonCode = ds(shift), ds(batch), ds(reversalOf), ds(reason)
	return c, err
}

func (e execution) ListConfirmations(ctx context.Context, orderID string) ([]domain.ProductionConfirmation, error) {
	clause, args := "", []any{}
	if orderID != "" {
		clause, args = " WHERE order_id = $1", []any{orderID}
	}
	rows, err := e.s.q.Query(ctx,
		"SELECT "+confirmationCols+" FROM production_confirmations"+clause+" ORDER BY confirmation_no", args...)
	if err != nil {
		return nil, mapError("confirmation", err)
	}
	defer rows.Close()

	var out []domain.ProductionConfirmation
	for rows.Next() {
		c, err := scanConfirmation(rows)
		if err != nil {
			return nil, mapError("confirmation", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError("confirmation", err)
	}
	for i := range out {
		consumptions, err := e.consumptionsFor(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Consumptions = consumptions
	}
	return out, nil
}

func (e execution) consumptionsFor(ctx context.Context, confirmationID string) ([]domain.MaterialConsumption, error) {
	rows, err := e.s.q.Query(ctx, `SELECT id, confirmation_id, material_id, quantity, uom,
		created_at, created_by FROM material_consumptions WHERE confirmation_id = $1
		ORDER BY material_id`, confirmationID)
	if err != nil {
		return nil, mapError("material consumption", err)
	}
	defer rows.Close()

	var out []domain.MaterialConsumption
	for rows.Next() {
		var m domain.MaterialConsumption
		if err := rows.Scan(&m.ID, &m.ConfirmationID, &m.MaterialID, &m.Quantity, &m.UOM,
			&m.CreatedAt, &m.CreatedBy); err != nil {
			return nil, mapError("material consumption", err)
		}
		out = append(out, m)
	}
	return out, mapError("material consumption", rows.Err())
}

func (e execution) GetConfirmation(ctx context.Context, id string) (domain.ProductionConfirmation, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.ProductionConfirmation{}, fmt.Errorf("%w: confirmation %s", domain.ErrNotFound, id)
	}
	c, err := scanConfirmation(e.s.q.QueryRow(ctx,
		"SELECT "+confirmationCols+" FROM production_confirmations WHERE id = $1", id))
	if err != nil {
		return domain.ProductionConfirmation{}, mapError("confirmation "+id, err)
	}
	c.Consumptions, err = e.consumptionsFor(ctx, c.ID)
	return c, err
}

func (e execution) SaveConfirmation(ctx context.Context, c domain.ProductionConfirmation, actor string) (domain.ProductionConfirmation, error) {
	now := nowUTC()
	if c.ID == "" {
		c.ID = uuid.NewString()
		c.CreatedAt, c.CreatedBy, c.UpdatedAt, c.UpdatedBy, c.RowVersion = now, actor, now, actor, 1
		_, err := e.s.q.Exec(ctx, `INSERT INTO production_confirmations
			(id, order_id, confirmation_no, business_date, shift_id, yield_qty, scrap_qty,
			 rework_qty, labour_hours, machine_hours, batch_id, reversal_of, reversed, reason_code,
			 created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$15,$16,1)`,
			c.ID, c.OrderID, c.ConfirmationNo, nd(c.BusinessDate), nu(c.ShiftID),
			c.YieldQty, c.ScrapQty, c.ReworkQty, c.LabourHours, c.MachineHours,
			nu(c.BatchID), nu(c.ReversalOf), c.Reversed, nu(c.ReasonCode), now, actor)
		if err != nil {
			return domain.ProductionConfirmation{}, mapError("confirmation "+c.ConfirmationNo, err)
		}
		for i := range c.Consumptions {
			c.Consumptions[i].ID = uuid.NewString()
			c.Consumptions[i].ConfirmationID = c.ID
			c.Consumptions[i].CreatedAt, c.Consumptions[i].CreatedBy = now, actor
			if _, err := e.s.q.Exec(ctx, `INSERT INTO material_consumptions
				(id, confirmation_id, material_id, quantity, uom, created_at, created_by)
				VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				c.Consumptions[i].ID, c.ID, c.Consumptions[i].MaterialID,
				c.Consumptions[i].Quantity, c.Consumptions[i].UOM, now, actor); err != nil {
				return domain.ProductionConfirmation{}, mapError("material consumption", err)
			}
		}
		return c, nil
	}

	// The only update a confirmation ever receives is being flagged as
	// reversed; its quantities are history.
	_, err := e.s.q.Exec(ctx, `UPDATE production_confirmations
		SET reversed = $1, updated_at = $2, updated_by = $3, row_version = row_version + 1
		WHERE id = $4`, c.Reversed, now, actor, c.ID)
	if err != nil {
		return domain.ProductionConfirmation{}, mapError("confirmation "+c.ConfirmationNo, err)
	}
	c.UpdatedAt, c.UpdatedBy, c.RowVersion = now, actor, c.RowVersion+1
	return c, nil
}

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

// documentColumns is the select list of inventory_documents, in the order
// scanDocument reads them. It is a list rather than a string so that the
// aliased form used by the join in ListDocuments is built by prefixing each
// name, not by patching the text - "factory_id" contains "id", and a textual
// substitution turns it into nonsense.
var documentColumns = []string{
	"id", "document_no", "doc_type", "business_date", "posted_at", "factory_id",
	"reference", "reversal_of", "reversed", "reason_code", "note", "created_at", "created_by",
}

// documentCols is the unqualified select list.
var documentCols = strings.Join(documentColumns, ", ")

// qualify prefixes every column with a table alias.
func qualify(alias string, cols []string) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = alias + "." + c
	}
	return strings.Join(out, ", ")
}

func scanDocument(r scanner) (domain.InventoryDocument, error) {
	var d domain.InventoryDocument
	var date time.Time
	var docType string
	var reversalOf, reason *string
	err := r.Scan(&d.ID, &d.DocumentNo, &docType, &date, &d.PostedAt, &d.FactoryID,
		&d.Reference, &reversalOf, &d.Reversed, &reason, &d.Note, &d.CreatedAt, &d.CreatedBy)
	d.BusinessDate, d.DocType = mustDate(date), domain.DocType(docType)
	d.ReversalOf, d.ReasonCode = ds(reversalOf), ds(reason)
	return d, err
}

// PostDocument writes the document, its items and the resulting balances.
//
// The balance update is an upsert, so the first movement into a store needs no
// special case, and it happens in the same statement sequence as the items:
// callers wrap the whole thing in InTx, so a failure leaves no trace.
func (e execution) PostDocument(ctx context.Context, d domain.InventoryDocument, actor string) (domain.InventoryDocument, error) {
	now := nowUTC()
	d.ID = uuid.NewString()
	d.CreatedAt, d.CreatedBy = now, actor
	if d.PostedAt.IsZero() {
		d.PostedAt = now
	}

	if _, err := e.s.q.Exec(ctx, `INSERT INTO inventory_documents
		(id, document_no, doc_type, business_date, posted_at, factory_id, reference,
		 reversal_of, reversed, reason_code, note, created_at, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		d.ID, d.DocumentNo, string(d.DocType), nd(d.BusinessDate), d.PostedAt, d.FactoryID,
		d.Reference, nu(d.ReversalOf), d.Reversed, nu(d.ReasonCode), d.Note, now, actor); err != nil {
		return domain.InventoryDocument{}, mapError("inventory document "+d.DocumentNo, err)
	}

	for i := range d.Items {
		item := &d.Items[i]
		item.ID = uuid.NewString()
		item.DocumentID = d.ID
		item.CreatedAt, item.CreatedBy = now, actor

		if _, err := e.s.q.Exec(ctx, `INSERT INTO inventory_document_items
			(id, document_id, line_no, warehouse_id, product_id, batch_id, quantity, uom,
			 to_warehouse, created_at, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			item.ID, d.ID, item.LineNo, item.WarehouseID, item.ProductID, nu(item.BatchID),
			item.Quantity, item.UOM, nu(item.ToWarehouse), now, actor); err != nil {
			return domain.InventoryDocument{}, mapError(
				fmt.Sprintf("inventory line %d of %s", item.LineNo, d.DocumentNo), err)
		}

		// Hold and release move the held quantity; everything else moves the
		// balance itself.
		quantityDelta, holdDelta := item.Quantity, domain.Zero
		switch d.DocType {
		case domain.DocHold:
			quantityDelta, holdDelta = domain.Zero, item.Quantity
		case domain.DocRelease:
			quantityDelta, holdDelta = domain.Zero, item.Quantity.Neg()
		}

		// The balance is moved in two statements rather than one upsert.
		//
		// PostgreSQL checks the table constraints against the tuple an INSERT
		// proposes before it discovers the conflict, so an upsert carrying the
		// deltas is judged as if it were the whole balance: a hold of 100 t
		// against a stock of 380 t arrives as "quantity 0, hold 100" and trips
		// the rule that nothing may be held that is not there. Creating the row
		// at zero and then adding the deltas states the same intent without
		// ever presenting a tuple that is not a real balance.
		if _, err := e.s.q.Exec(ctx, `INSERT INTO stock_balances
			(warehouse_id, product_id, quantity, hold_quantity, updated_at, updated_by, row_version)
			VALUES ($1,$2,0,0,$3,$4,0)
			ON CONFLICT (warehouse_id, product_id) DO NOTHING`,
			item.WarehouseID, item.ProductID, now, actor); err != nil {
			return domain.InventoryDocument{}, mapError(
				fmt.Sprintf("stock balance for line %d of %s", item.LineNo, d.DocumentNo), err)
		}
		if _, err := e.s.q.Exec(ctx, `UPDATE stock_balances SET
				quantity = quantity + $1, hold_quantity = hold_quantity + $2,
				updated_at = $3, updated_by = $4, row_version = row_version + 1
			WHERE warehouse_id = $5 AND product_id = $6`,
			quantityDelta, holdDelta, now, actor, item.WarehouseID, item.ProductID); err != nil {
			return domain.InventoryDocument{}, mapError(
				fmt.Sprintf("stock balance for line %d of %s", item.LineNo, d.DocumentNo), err)
		}
	}
	return d, nil
}

func (e execution) GetDocument(ctx context.Context, id string) (domain.InventoryDocument, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.InventoryDocument{}, fmt.Errorf("%w: inventory document %s", domain.ErrNotFound, id)
	}
	d, err := scanDocument(e.s.q.QueryRow(ctx,
		"SELECT "+documentCols+" FROM inventory_documents WHERE id = $1", id))
	if err != nil {
		return domain.InventoryDocument{}, mapError("inventory document "+id, err)
	}
	d.Items, err = e.itemsFor(ctx, d.ID)
	return d, err
}

func (e execution) itemsFor(ctx context.Context, documentID string) ([]domain.InventoryDocumentItem, error) {
	rows, err := e.s.q.Query(ctx, `SELECT id, document_id, line_no, warehouse_id, product_id,
		batch_id, quantity, uom, to_warehouse, created_at, created_by
		FROM inventory_document_items WHERE document_id = $1 ORDER BY line_no`, documentID)
	if err != nil {
		return nil, mapError("inventory document item", err)
	}
	defer rows.Close()

	var out []domain.InventoryDocumentItem
	for rows.Next() {
		var item domain.InventoryDocumentItem
		var batch, toWarehouse *string
		if err := rows.Scan(&item.ID, &item.DocumentID, &item.LineNo, &item.WarehouseID,
			&item.ProductID, &batch, &item.Quantity, &item.UOM, &toWarehouse,
			&item.CreatedAt, &item.CreatedBy); err != nil {
			return nil, mapError("inventory document item", err)
		}
		item.BatchID, item.ToWarehouse = ds(batch), ds(toWarehouse)
		out = append(out, item)
	}
	return out, mapError("inventory document item", rows.Err())
}

func (e execution) ListDocuments(ctx context.Context, f store.ExecutionFilter) (store.Page[domain.InventoryDocument], error) {
	w := &execWhere{}
	if f.FactoryID != "" {
		w.eq("d.factory_id", f.FactoryID)
	}
	w.dateRange("d.business_date", f.From, f.To)
	if f.WarehouseID != "" {
		w.args = append(w.args, f.WarehouseID)
		w.raw(fmt.Sprintf(
			"EXISTS (SELECT 1 FROM inventory_document_items i WHERE i.document_id = d.id AND i.warehouse_id = $%d)",
			len(w.args)))
	}
	clause := w.sql()

	var total int
	if err := e.s.q.QueryRow(ctx,
		"SELECT count(*) FROM inventory_documents d"+clause, w.args...).Scan(&total); err != nil {
		return store.Page[domain.InventoryDocument]{}, mapError("inventory document", err)
	}
	limit := w.limit(f)

	rows, err := e.s.q.Query(ctx, "SELECT "+qualify("d", documentColumns)+" FROM inventory_documents d"+clause+
		" ORDER BY d.business_date DESC, d.document_no"+limit, w.args...)
	if err != nil {
		return store.Page[domain.InventoryDocument]{}, mapError("inventory document", err)
	}
	defer rows.Close()

	items := []domain.InventoryDocument{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return store.Page[domain.InventoryDocument]{}, mapError("inventory document", err)
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return store.Page[domain.InventoryDocument]{}, mapError("inventory document", err)
	}
	for i := range items {
		if items[i].Items, err = e.itemsFor(ctx, items[i].ID); err != nil {
			return store.Page[domain.InventoryDocument]{}, err
		}
	}
	return store.Page[domain.InventoryDocument]{Items: items, Count: total}, nil
}

func (e execution) MarkReversed(ctx context.Context, documentID, actor string) error {
	if _, err := uuid.Parse(documentID); err != nil {
		return fmt.Errorf("%w: inventory document %s", domain.ErrNotFound, documentID)
	}
	var documentNo string
	err := e.s.q.QueryRow(ctx, `UPDATE inventory_documents SET reversed = true
		WHERE id = $1 AND reversed = false RETURNING document_no`, documentID).Scan(&documentNo)
	if err == nil {
		return nil
	}
	if !isNoRows(err) {
		return mapError("inventory document", err)
	}
	// No row was updated: either the document is not there at all, or somebody
	// has already reversed it. The caller needs to be able to tell those apart.
	if err := e.s.q.QueryRow(ctx,
		"SELECT document_no FROM inventory_documents WHERE id = $1", documentID).
		Scan(&documentNo); err != nil {
		return mapError("inventory document "+documentID, err)
	}
	return fmt.Errorf("%w: document %s has already been reversed",
		domain.ErrValidation, documentNo)
}

func (e execution) Positions(ctx context.Context, pairs [][2]string) (map[string]domain.StockPosition, error) {
	out := map[string]domain.StockPosition{}
	if len(pairs) == 0 {
		return out, nil
	}

	warehouses := make([]string, 0, len(pairs))
	products := make([]string, 0, len(pairs))
	for _, p := range pairs {
		warehouses = append(warehouses, p[0])
		products = append(products, p[1])
		// A pair that has never been posted has a zero position, not a missing
		// one; the query below overwrites the ones that exist.
		out[p[0]+"|"+p[1]] = domain.StockPosition{
			WarehouseID: p[0], ProductID: p[1],
			Quantity: domain.Zero, HoldQuantity: domain.Zero,
		}
	}

	rows, err := e.s.q.Query(ctx, `SELECT warehouse_id, product_id, quantity, hold_quantity,
		updated_at, updated_by, row_version FROM stock_balances
		WHERE warehouse_id = ANY($1) AND product_id = ANY($2)`, warehouses, products)
	if err != nil {
		return nil, mapError("stock balance", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p domain.StockPosition
		if err := rows.Scan(&p.WarehouseID, &p.ProductID, &p.Quantity, &p.HoldQuantity,
			&p.UpdatedAt, &p.UpdatedBy, &p.RowVersion); err != nil {
			return nil, mapError("stock balance", err)
		}
		key := p.WarehouseID + "|" + p.ProductID
		if _, wanted := out[key]; wanted {
			out[key] = p
		}
	}
	return out, mapError("stock balance", rows.Err())
}

func (e execution) ListPositions(ctx context.Context, f store.ExecutionFilter) ([]domain.StockPosition, error) {
	w := &execWhere{}
	if f.WarehouseID != "" {
		w.eq("warehouse_id", f.WarehouseID)
	}
	w.in("product_id", f.ProductIDs)

	rows, err := e.s.q.Query(ctx, `SELECT warehouse_id, product_id, quantity, hold_quantity,
		updated_at, updated_by, row_version FROM stock_balances`+w.sql()+
		" ORDER BY warehouse_id, product_id", w.args...)
	if err != nil {
		return nil, mapError("stock balance", err)
	}
	defer rows.Close()

	var out []domain.StockPosition
	for rows.Next() {
		var p domain.StockPosition
		if err := rows.Scan(&p.WarehouseID, &p.ProductID, &p.Quantity, &p.HoldQuantity,
			&p.UpdatedAt, &p.UpdatedBy, &p.RowVersion); err != nil {
			return nil, mapError("stock balance", err)
		}
		out = append(out, p)
	}
	return out, mapError("stock balance", rows.Err())
}

// ---------------------------------------------------------------------------
// Quality
// ---------------------------------------------------------------------------

func (e execution) ListParameters(ctx context.Context) ([]domain.QualityParameter, error) {
	rows, err := e.s.q.Query(ctx, `SELECT id, code, name, uom, test_method, active,
		created_at, created_by, updated_at, updated_by, row_version
		FROM quality_parameters ORDER BY code`)
	if err != nil {
		return nil, mapError("quality parameter", err)
	}
	defer rows.Close()

	var out []domain.QualityParameter
	for rows.Next() {
		var p domain.QualityParameter
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.UOM, &p.TestMethod, &p.Active,
			&p.CreatedAt, &p.CreatedBy, &p.UpdatedAt, &p.UpdatedBy, &p.RowVersion); err != nil {
			return nil, mapError("quality parameter", err)
		}
		out = append(out, p)
	}
	return out, mapError("quality parameter", rows.Err())
}

func (e execution) SaveParameter(ctx context.Context, p domain.QualityParameter, actor string) (domain.QualityParameter, error) {
	now := nowUTC()
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	err := e.s.q.QueryRow(ctx, `INSERT INTO quality_parameters
		(id, code, name, uom, test_method, active, created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$7,$8,1)
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, uom = EXCLUDED.uom,
			test_method = EXCLUDED.test_method, active = EXCLUDED.active,
			updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by,
			row_version = quality_parameters.row_version + 1
		RETURNING id, created_at, created_by, row_version`,
		p.ID, p.Code, p.Name, p.UOM, p.TestMethod, p.Active, now, actor).
		Scan(&p.ID, &p.CreatedAt, &p.CreatedBy, &p.RowVersion)
	if err != nil {
		return domain.QualityParameter{}, mapError("quality parameter "+p.Code, err)
	}
	p.UpdatedAt, p.UpdatedBy = now, actor
	return p, nil
}

const specCols = `id, product_id, parameter_id, lower_limit, upper_limit, warn_lower, warn_upper,
	valid_from, valid_to, created_at, created_by, updated_at, updated_by, row_version`

func scanSpec(r scanner) (domain.QualitySpec, error) {
	var s domain.QualitySpec
	var from time.Time
	var to *time.Time
	err := r.Scan(&s.ID, &s.ProductID, &s.ParameterID, &s.LowerLimit, &s.UpperLimit,
		&s.WarnLower, &s.WarnUpper, &from, &to,
		&s.CreatedAt, &s.CreatedBy, &s.UpdatedAt, &s.UpdatedBy, &s.RowVersion)
	s.ValidFrom, s.ValidTo = mustDate(from), bd(to)
	return s, err
}

func (e execution) SpecsFor(ctx context.Context, productID string, on domain.BusinessDate) ([]domain.QualitySpec, error) {
	rows, err := e.s.q.Query(ctx, "SELECT "+specCols+` FROM quality_specs
		WHERE product_id = $1 AND valid_from <= $2 AND (valid_to IS NULL OR valid_to >= $2)
		ORDER BY parameter_id`, productID, nd(on))
	if err != nil {
		return nil, mapError("quality specification", err)
	}
	defer rows.Close()

	var out []domain.QualitySpec
	for rows.Next() {
		s, err := scanSpec(rows)
		if err != nil {
			return nil, mapError("quality specification", err)
		}
		out = append(out, s)
	}
	return out, mapError("quality specification", rows.Err())
}

func (e execution) SaveSpec(ctx context.Context, s domain.QualitySpec, actor string) (domain.QualitySpec, error) {
	now := nowUTC()
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	err := e.s.q.QueryRow(ctx, `INSERT INTO quality_specs
		(id, product_id, parameter_id, lower_limit, upper_limit, warn_lower, warn_upper,
		 valid_from, valid_to, created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$10,$11,1)
		ON CONFLICT (product_id, parameter_id, valid_from) DO UPDATE SET
			lower_limit = EXCLUDED.lower_limit, upper_limit = EXCLUDED.upper_limit,
			warn_lower = EXCLUDED.warn_lower, warn_upper = EXCLUDED.warn_upper,
			valid_to = EXCLUDED.valid_to, updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by, row_version = quality_specs.row_version + 1
		RETURNING id, created_at, created_by, row_version`,
		s.ID, s.ProductID, s.ParameterID, s.LowerLimit, s.UpperLimit, s.WarnLower, s.WarnUpper,
		nd(s.ValidFrom), nd(s.ValidTo), now, actor).
		Scan(&s.ID, &s.CreatedAt, &s.CreatedBy, &s.RowVersion)
	if err != nil {
		return domain.QualitySpec{}, mapError("quality specification", err)
	}
	s.UpdatedAt, s.UpdatedBy = now, actor
	return s, nil
}

const sampleCols = `id, sample_no, product_id, batch_id, factory_id, business_date, shift_id,
	taken_at, lab_user, status, comment, created_at, created_by, updated_at, updated_by, row_version`

func scanSample(r scanner) (domain.QualitySample, error) {
	var s domain.QualitySample
	var date time.Time
	var batch, shift *string
	err := r.Scan(&s.ID, &s.SampleNo, &s.ProductID, &batch, &s.FactoryID, &date, &shift,
		&s.TakenAt, &s.LabUser, &s.Status, &s.Comment,
		&s.CreatedAt, &s.CreatedBy, &s.UpdatedAt, &s.UpdatedBy, &s.RowVersion)
	s.BusinessDate, s.BatchID, s.ShiftID = mustDate(date), ds(batch), ds(shift)
	return s, err
}

func (e execution) ListSamples(ctx context.Context, f store.ExecutionFilter) (store.Page[domain.QualitySample], error) {
	w := &execWhere{}
	if f.FactoryID != "" {
		w.eq("factory_id", f.FactoryID)
	}
	w.in("product_id", f.ProductIDs)
	w.dateRange("business_date", f.From, f.To)
	if f.Number != "" {
		w.eq("sample_no", f.Number)
	}
	clause := w.sql()

	var total int
	if err := e.s.q.QueryRow(ctx, "SELECT count(*) FROM quality_samples"+clause, w.args...).
		Scan(&total); err != nil {
		return store.Page[domain.QualitySample]{}, mapError("quality sample", err)
	}
	limit := w.limit(f)

	rows, err := e.s.q.Query(ctx, "SELECT "+sampleCols+" FROM quality_samples"+clause+
		" ORDER BY business_date DESC, sample_no DESC"+limit, w.args...)
	if err != nil {
		return store.Page[domain.QualitySample]{}, mapError("quality sample", err)
	}
	defer rows.Close()

	items := []domain.QualitySample{}
	for rows.Next() {
		s, err := scanSample(rows)
		if err != nil {
			return store.Page[domain.QualitySample]{}, mapError("quality sample", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return store.Page[domain.QualitySample]{}, mapError("quality sample", err)
	}
	for i := range items {
		if items[i].Results, err = e.resultsFor(ctx, items[i].ID); err != nil {
			return store.Page[domain.QualitySample]{}, err
		}
	}
	return store.Page[domain.QualitySample]{Items: items, Count: total}, nil
}

func (e execution) resultsFor(ctx context.Context, sampleID string) ([]domain.QualityResult, error) {
	rows, err := e.s.q.Query(ctx, `SELECT id, sample_id, parameter_id, result_value, uom,
		lower_limit, upper_limit, status, comment, created_at, created_by
		FROM quality_results WHERE sample_id = $1 ORDER BY parameter_id`, sampleID)
	if err != nil {
		return nil, mapError("quality result", err)
	}
	defer rows.Close()

	var out []domain.QualityResult
	for rows.Next() {
		var r domain.QualityResult
		var status string
		if err := rows.Scan(&r.ID, &r.SampleID, &r.ParameterID, &r.Value, &r.UOM,
			&r.LowerLimit, &r.UpperLimit, &status, &r.Comment,
			&r.CreatedAt, &r.CreatedBy); err != nil {
			return nil, mapError("quality result", err)
		}
		r.Status = domain.QualityStatus(status)
		out = append(out, r)
	}
	return out, mapError("quality result", rows.Err())
}

func (e execution) GetSample(ctx context.Context, id string) (domain.QualitySample, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.QualitySample{}, fmt.Errorf("%w: quality sample %s", domain.ErrNotFound, id)
	}
	s, err := scanSample(e.s.q.QueryRow(ctx,
		"SELECT "+sampleCols+" FROM quality_samples WHERE id = $1", id))
	if err != nil {
		return domain.QualitySample{}, mapError("quality sample "+id, err)
	}
	s.Results, err = e.resultsFor(ctx, s.ID)
	return s, err
}

func (e execution) SaveSample(ctx context.Context, s domain.QualitySample, actor string) (domain.QualitySample, error) {
	now := nowUTC()
	if s.Status == "" {
		s.Status = "OPEN"
	}
	if s.TakenAt.IsZero() {
		s.TakenAt = now
	}
	if s.ID == "" {
		s.ID = uuid.NewString()
		s.CreatedAt, s.CreatedBy, s.UpdatedAt, s.UpdatedBy, s.RowVersion = now, actor, now, actor, 1
		_, err := e.s.q.Exec(ctx, `INSERT INTO quality_samples
			(id, sample_no, product_id, batch_id, factory_id, business_date, shift_id, taken_at,
			 lab_user, status, comment, created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$12,$13,1)`,
			s.ID, s.SampleNo, s.ProductID, nu(s.BatchID), s.FactoryID, nd(s.BusinessDate),
			nu(s.ShiftID), s.TakenAt, s.LabUser, s.Status, s.Comment, now, actor)
		if err != nil {
			return domain.QualitySample{}, mapError("quality sample "+s.SampleNo, err)
		}
		return s, nil
	}
	_, err := e.s.q.Exec(ctx, `UPDATE quality_samples SET status = $1, comment = $2,
		lab_user = $3, updated_at = $4, updated_by = $5, row_version = row_version + 1
		WHERE id = $6`, s.Status, s.Comment, s.LabUser, now, actor, s.ID)
	if err != nil {
		return domain.QualitySample{}, mapError("quality sample "+s.SampleNo, err)
	}
	s.UpdatedAt, s.UpdatedBy, s.RowVersion = now, actor, s.RowVersion+1
	return s, nil
}

// SaveResults replaces the results of a sample.
//
// A re-entered result set is the whole set, not an addition to it: a parameter
// the laboratory has dropped from the sheet has to disappear from the record
// too, otherwise a stale measurement keeps voting in the sample's verdict.
func (e execution) SaveResults(ctx context.Context, sampleID string, results []domain.QualityResult, actor string) error {
	if _, err := uuid.Parse(sampleID); err != nil {
		return fmt.Errorf("%w: quality sample %s", domain.ErrNotFound, sampleID)
	}
	var exists string
	if err := e.s.q.QueryRow(ctx, "SELECT id FROM quality_samples WHERE id = $1", sampleID).
		Scan(&exists); err != nil {
		return mapError("quality sample "+sampleID, err)
	}

	keep := make([]string, 0, len(results))
	for _, r := range results {
		keep = append(keep, r.ParameterID)
	}
	if _, err := e.s.q.Exec(ctx,
		"DELETE FROM quality_results WHERE sample_id = $1 AND parameter_id::text <> ALL($2::text[])",
		sampleID, keep); err != nil {
		return mapError("quality result", err)
	}

	now := nowUTC()
	for _, r := range results {
		id := r.ID
		if id == "" {
			id = uuid.NewString()
		}
		if _, err := e.s.q.Exec(ctx, `INSERT INTO quality_results
			(id, sample_id, parameter_id, result_value, uom, lower_limit, upper_limit,
			 status, comment, created_at, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			ON CONFLICT (sample_id, parameter_id) DO UPDATE SET
				result_value = EXCLUDED.result_value, uom = EXCLUDED.uom,
				lower_limit = EXCLUDED.lower_limit, upper_limit = EXCLUDED.upper_limit,
				status = EXCLUDED.status, comment = EXCLUDED.comment`,
			id, sampleID, r.ParameterID, r.Value, r.UOM, r.LowerLimit, r.UpperLimit,
			string(r.Status), r.Comment, now, actor); err != nil {
			return mapError("quality result", err)
		}
	}
	return nil
}

const holdCols = `id, warehouse_id, product_id, batch_id, sample_id, quantity, placed_on,
	released_on, released_by, reason, created_at, created_by, updated_at, updated_by, row_version`

func scanHold(r scanner) (domain.QualityHold, error) {
	var h domain.QualityHold
	var placed time.Time
	var released *time.Time
	var batch, sample, releasedBy *string
	err := r.Scan(&h.ID, &h.WarehouseID, &h.ProductID, &batch, &sample, &h.Quantity,
		&placed, &released, &releasedBy, &h.Reason,
		&h.CreatedAt, &h.CreatedBy, &h.UpdatedAt, &h.UpdatedBy, &h.RowVersion)
	h.PlacedOn, h.ReleasedOn = mustDate(placed), bd(released)
	h.BatchID, h.SampleID, h.ReleasedBy = ds(batch), ds(sample), ds(releasedBy)
	return h, err
}

func (e execution) ListHolds(ctx context.Context, f store.ExecutionFilter) ([]domain.QualityHold, error) {
	w := &execWhere{}
	if f.WarehouseID != "" {
		w.eq("warehouse_id", f.WarehouseID)
	}
	w.in("product_id", f.ProductIDs)
	if f.OpenOnly {
		w.raw("released_on IS NULL")
	}
	rows, err := e.s.q.Query(ctx, "SELECT "+holdCols+" FROM quality_holds"+w.sql()+
		" ORDER BY placed_on DESC", w.args...)
	if err != nil {
		return nil, mapError("quality hold", err)
	}
	defer rows.Close()

	var out []domain.QualityHold
	for rows.Next() {
		h, err := scanHold(rows)
		if err != nil {
			return nil, mapError("quality hold", err)
		}
		out = append(out, h)
	}
	return out, mapError("quality hold", rows.Err())
}

func (e execution) GetHold(ctx context.Context, id string) (domain.QualityHold, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.QualityHold{}, fmt.Errorf("%w: quality hold %s", domain.ErrNotFound, id)
	}
	h, err := scanHold(e.s.q.QueryRow(ctx, "SELECT "+holdCols+" FROM quality_holds WHERE id = $1", id))
	return h, mapError("quality hold "+id, err)
}

func (e execution) SaveHold(ctx context.Context, h domain.QualityHold, actor string) (domain.QualityHold, error) {
	now := nowUTC()
	if h.ID == "" {
		h.ID = uuid.NewString()
		h.CreatedAt, h.CreatedBy, h.UpdatedAt, h.UpdatedBy, h.RowVersion = now, actor, now, actor, 1
		_, err := e.s.q.Exec(ctx, `INSERT INTO quality_holds
			(id, warehouse_id, product_id, batch_id, sample_id, quantity, placed_on,
			 released_on, released_by, reason, created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$11,$12,1)`,
			h.ID, h.WarehouseID, h.ProductID, nu(h.BatchID), nu(h.SampleID), h.Quantity,
			nd(h.PlacedOn), nd(h.ReleasedOn), nu(h.ReleasedBy), h.Reason, now, actor)
		if err != nil {
			return domain.QualityHold{}, mapError("quality hold", err)
		}
		return h, nil
	}
	var created time.Time
	var createdBy string
	var version int64
	err := e.s.q.QueryRow(ctx, `UPDATE quality_holds SET released_on = $1, released_by = $2,
			reason = $3, updated_at = $4, updated_by = $5, row_version = row_version + 1
		WHERE id = $6 AND row_version = $7
		RETURNING created_at, created_by, row_version`,
		nd(h.ReleasedOn), nu(h.ReleasedBy), h.Reason, now, actor, h.ID, h.RowVersion).
		Scan(&created, &createdBy, &version)
	if err != nil {
		if isNoRows(err) {
			return domain.QualityHold{}, versionConflict(ctx, e.s,
				"quality_holds", "quality hold", h.ID, h.RowVersion)
		}
		return domain.QualityHold{}, mapError("quality hold", err)
	}
	h.CreatedAt, h.CreatedBy, h.UpdatedAt, h.UpdatedBy, h.RowVersion = created, createdBy, now, actor, version
	return h, nil
}

// ---------------------------------------------------------------------------
// Maintenance
// ---------------------------------------------------------------------------

const maintenanceCols = `id, factory_id, line_id, equipment_id, start_date, end_date,
	description, status, approved_by, created_at, created_by, updated_at, updated_by, row_version`

func scanMaintenance(r scanner) (domain.MaintenanceWindow, error) {
	var m domain.MaintenanceWindow
	var start, end time.Time
	var line, equipment, approvedBy *string
	err := r.Scan(&m.ID, &m.FactoryID, &line, &equipment, &start, &end,
		&m.Description, &m.Status, &approvedBy,
		&m.CreatedAt, &m.CreatedBy, &m.UpdatedAt, &m.UpdatedBy, &m.RowVersion)
	m.StartDate, m.EndDate = mustDate(start), mustDate(end)
	m.LineID, m.EquipmentID, m.ApprovedBy = ds(line), ds(equipment), ds(approvedBy)
	return m, err
}

func (e execution) ListMaintenance(ctx context.Context, f store.ExecutionFilter) ([]domain.MaintenanceWindow, error) {
	w := &execWhere{}
	if f.FactoryID != "" {
		w.eq("factory_id", f.FactoryID)
	}
	w.in("status", f.Statuses)
	// A window overlaps the range when it starts before the end of it and ends
	// after the start of it.
	if f.To != "" {
		w.args = append(w.args, nd(f.To))
		w.raw(fmt.Sprintf("start_date <= $%d", len(w.args)))
	}
	if f.From != "" {
		w.args = append(w.args, nd(f.From))
		w.raw(fmt.Sprintf("end_date >= $%d", len(w.args)))
	}

	rows, err := e.s.q.Query(ctx, "SELECT "+maintenanceCols+" FROM maintenance_windows"+w.sql()+
		" ORDER BY start_date", w.args...)
	if err != nil {
		return nil, mapError("maintenance window", err)
	}
	defer rows.Close()

	var out []domain.MaintenanceWindow
	for rows.Next() {
		m, err := scanMaintenance(rows)
		if err != nil {
			return nil, mapError("maintenance window", err)
		}
		out = append(out, m)
	}
	return out, mapError("maintenance window", rows.Err())
}

func (e execution) SaveMaintenance(ctx context.Context, m domain.MaintenanceWindow, actor string) (domain.MaintenanceWindow, error) {
	now := nowUTC()
	if m.Status == "" {
		m.Status = domain.MaintenancePlanned
	}
	if m.ID == "" {
		m.ID = uuid.NewString()
		m.CreatedAt, m.CreatedBy, m.UpdatedAt, m.UpdatedBy, m.RowVersion = now, actor, now, actor, 1
		_, err := e.s.q.Exec(ctx, `INSERT INTO maintenance_windows
			(id, factory_id, line_id, equipment_id, start_date, end_date, description, status,
			 approved_by, created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$10,$11,1)`,
			m.ID, m.FactoryID, nu(m.LineID), nu(m.EquipmentID), nd(m.StartDate), nd(m.EndDate),
			m.Description, m.Status, nu(m.ApprovedBy), now, actor)
		if err != nil {
			return domain.MaintenanceWindow{}, mapError("maintenance window", err)
		}
		return m, nil
	}
	var created time.Time
	var createdBy string
	var version int64
	err := e.s.q.QueryRow(ctx, `UPDATE maintenance_windows SET line_id=$1, equipment_id=$2,
			start_date=$3, end_date=$4, description=$5, status=$6, approved_by=$7,
			updated_at=$8, updated_by=$9, row_version = row_version + 1
		WHERE id=$10 AND row_version=$11
		RETURNING created_at, created_by, row_version`,
		nu(m.LineID), nu(m.EquipmentID), nd(m.StartDate), nd(m.EndDate), m.Description,
		m.Status, nu(m.ApprovedBy), now, actor, m.ID, m.RowVersion).
		Scan(&created, &createdBy, &version)
	if err != nil {
		if isNoRows(err) {
			return domain.MaintenanceWindow{}, versionConflict(ctx, e.s,
				"maintenance_windows", "maintenance window", m.ID, m.RowVersion)
		}
		return domain.MaintenanceWindow{}, mapError("maintenance window", err)
	}
	m.CreatedAt, m.CreatedBy, m.UpdatedAt, m.UpdatedBy, m.RowVersion = created, createdBy, now, actor, version
	return m, nil
}
