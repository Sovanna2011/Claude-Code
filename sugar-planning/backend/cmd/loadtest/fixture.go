package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The fixture: ten years of daily history, written straight into the tables.
//
// Everywhere else in this repository, data is produced by driving the services,
// because a figure written past the rules is a figure that does not reconcile.
// Here that rule is deliberately broken, and it is worth saying why: the point
// of this fixture is *volume*, not correctness. Ten seasons through the posting
// engine one transaction at a time would take hours and would prove nothing the
// service tests do not already prove. What a query planner cares about is how
// many rows there are, how they are distributed, and whether the statistics
// reflect that - and a COPY gives all three in seconds.
//
// The consequence to keep in mind: these rows are not guaranteed to reconcile
// with each other the way seeded data does. Nothing here should be read as a
// business figure. It exists to be selected, filtered and aggregated over.

// scale describes the shape of one generated season.
type scale struct {
	Days        int // crushing days
	Products    int // finished products with a daily row
	Warehouses  int // stores with a daily ledger row
	DocsPerDay  int // inventory documents posted each day
	ItemsPerDoc int
}

var defaultScale = scale{
	Days: 137, Products: 4, Warehouses: 5, DocsPerDay: 40, ItemsPerDoc: 3,
}

// rowEstimate is what a run will add, so the operator can decide before it
// starts rather than discover halfway through.
func (s scale) rowEstimate(seasons int) map[string]int {
	perSeason := map[string]int{
		"daily_cane_plans":         s.Days * 2, // plan and actual
		"daily_product_plans":      s.Days * s.Products * 2,
		"daily_storage_plans":      s.Days * s.Warehouses * s.Products,
		"inventory_documents":      s.Days * s.DocsPerDay,
		"inventory_document_items": s.Days * s.DocsPerDay * s.ItemsPerDoc,
		"audit_events":             s.Days * s.DocsPerDay,
	}
	out := map[string]int{}
	total := 0
	for k, v := range perSeason {
		out[k] = v * seasons
		total += v * seasons
	}
	out["TOTAL"] = total
	return out
}

// reference holds the ids the fixture hangs off: it extends the seeded factory
// rather than inventing an organisation, so the rows land where the API will
// look for them.
type reference struct {
	CompanyID  string
	FactoryID  string
	Products   []string
	Warehouses []string
}

func loadReference(ctx context.Context, db *pgxpool.Pool, factoryCode string) (reference, error) {
	var ref reference
	err := db.QueryRow(ctx,
		`SELECT f.id, f.company_id FROM factories f WHERE f.code = $1`, factoryCode).
		Scan(&ref.FactoryID, &ref.CompanyID)
	if err != nil {
		return ref, fmt.Errorf("factory %s: %w (run the migrations and the seed first)",
			factoryCode, err)
	}

	rows, err := db.Query(ctx,
		`SELECT id FROM products WHERE stage IN ('FINISHED','REFINED') OR true ORDER BY code LIMIT 8`)
	if err != nil {
		return ref, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return ref, err
		}
		ref.Products = append(ref.Products, id)
	}
	rows.Close()

	rows, err = db.Query(ctx,
		`SELECT id FROM warehouses WHERE factory_id = $1 ORDER BY code`, ref.FactoryID)
	if err != nil {
		return ref, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return ref, err
		}
		ref.Warehouses = append(ref.Warehouses, id)
	}
	rows.Close()

	if len(ref.Products) == 0 || len(ref.Warehouses) == 0 {
		return ref, fmt.Errorf("the factory has no products or no warehouses; " +
			"load the reference scenario first")
	}
	return ref, nil
}

// generate writes `seasons` historical seasons, oldest first, ending the year
// before the seeded one so the reference season stays the current campaign.
func generate(ctx context.Context, db *pgxpool.Pool, ref reference, seasons int, s scale,
	progress func(string)) error {

	for i := 0; i < seasons; i++ {
		// 2016-2017 backwards from the seeded 2026-2027.
		startYear := 2026 - seasons + i
		code := fmt.Sprintf("%d-%d", startYear, startYear+1)

		var exists bool
		if err := db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM seasons WHERE factory_id = $1 AND code = $2)`,
			ref.FactoryID, code).Scan(&exists); err != nil {
			return err
		}
		if exists {
			progress(fmt.Sprintf("season %s already present, left alone", code))
			continue
		}

		start := time.Date(startYear, 12, 1, 0, 0, 0, 0, time.UTC)
		if err := generateSeason(ctx, db, ref, code, start, s); err != nil {
			return fmt.Errorf("season %s: %w", code, err)
		}
		progress(fmt.Sprintf("season %s written", code))
	}

	// Without this the planner is working from statistics that predate the
	// rows, and the first measurement measures a bad plan rather than the
	// schema. It is also what a real bulk load would do.
	progress("analysing")
	_, err := db.Exec(ctx, `ANALYZE`)
	return err
}

func generateSeason(ctx context.Context, db *pgxpool.Pool, ref reference,
	code string, start time.Time, s scale) error {

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	seasonID := uuid.NewString()
	end := start.AddDate(0, 0, s.Days-1)
	if _, err := tx.Exec(ctx,
		`INSERT INTO seasons (id, company_id, factory_id, code, name, start_date, end_date,
		    planned_days, status, created_by, updated_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'CLOSED','loadtest','loadtest')`,
		seasonID, ref.CompanyID, ref.FactoryID, code, "Crushing season "+code,
		start, end, s.Days); err != nil {
		return err
	}

	planID, actualID := uuid.NewString(), uuid.NewString()
	for _, v := range []struct {
		id, code, planType, status string
		no                         int
	}{
		{planID, "V1", "BUDGET", "RELEASED", 1},
		{actualID, "ACTUAL", "ACTUAL", "CLOSED", 2},
	} {
		if _, err := tx.Exec(ctx,
			`INSERT INTO plan_versions (id, season_id, version_no, code, description, plan_type,
			    status, owner, effective_from, created_by, updated_by)
			 VALUES ($1,$2,$3,$4,'load fixture',$5,$6,'loadtest',$7,'loadtest','loadtest')`,
			v.id, seasonID, v.no, v.code, v.planType, v.status, start); err != nil {
			return err
		}
	}

	if err := copyCane(ctx, tx, ref, planID, actualID, start, s); err != nil {
		return fmt.Errorf("cane: %w", err)
	}
	if err := copyProduction(ctx, tx, ref, planID, actualID, start, s); err != nil {
		return fmt.Errorf("production: %w", err)
	}
	if err := copyStorage(ctx, tx, ref, actualID, start, s); err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	if err := copyDocuments(ctx, tx, ref, start, s); err != nil {
		return fmt.Errorf("documents: %w", err)
	}
	return tx.Commit(ctx)
}

// A deterministic wobble, so the data is not a flat line - a planner that sees
// one value per column can choose an index a real distribution would not
// justify. No randomness: the same fixture every run, so two measurements are
// comparable.
func wobble(day, spread int) float64 {
	return float64((day*37)%spread) - float64(spread)/2
}

func copyCane(ctx context.Context, tx pgx.Tx, ref reference, planID, actualID string,
	start time.Time, s scale) error {

	rows := make([][]any, 0, s.Days*2)
	for d := 0; d < s.Days; d++ {
		date := start.AddDate(0, 0, d)
		target := 16788.0
		actual := target + wobble(d, 4000)
		for _, v := range []struct {
			id     string
			series string
			qty    float64
		}{{planID, "PLAN", target}, {actualID, "ACTUAL", actual}} {
			rows = append(rows, []any{
				uuid.NewString(), v.id, ref.FactoryID, date, nil, v.series,
				qty(v.qty * 1.02), qty(v.qty * 1.01), qty(v.qty), qty(v.qty * 0.01), qty(0),
				qty(v.qty), qty(v.qty / 22), 24.0, 2.0, nil, "load fixture",
				"loadtest", "loadtest",
			})
		}
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"daily_cane_plans"},
		[]string{"id", "version_id", "factory_id", "business_date", "shift_id", "series",
			"cane_available", "cane_delivered", "cane_accepted", "cane_rejected",
			"cane_diverted", "cane_crushed", "crush_rate_tph", "available_hrs",
			"stoppage_hrs", "reason_code", "note", "created_by", "updated_by"},
		pgx.CopyFromRows(rows))
	return err
}

func copyProduction(ctx context.Context, tx pgx.Tx, ref reference, planID, actualID string,
	start time.Time, s scale) error {

	products := take(ref.Products, s.Products)
	rows := make([][]any, 0, s.Days*len(products)*2)
	for d := 0; d < s.Days; d++ {
		date := start.AddDate(0, 0, d)
		for _, p := range products {
			target := 440.0
			actual := target + wobble(d, 120)
			for _, v := range []struct {
				id     string
				series string
				q      float64
			}{{planID, "PLAN", target}, {actualID, "ACTUAL", actual}} {
				rows = append(rows, []any{
					uuid.NewString(), v.id, ref.FactoryID, nil, date, nil, p, nil, v.series,
					qty(v.q), qty(v.q * 1.05), qty(v.q * 0.01), qty(0), qty(0), qty(0),
					nil, "load fixture", "loadtest", "loadtest",
				})
			}
		}
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"daily_product_plans"},
		[]string{"id", "version_id", "factory_id", "line_id", "business_date", "shift_id",
			"product_id", "packaging_id", "series", "quantity", "remelt_input",
			"process_loss", "rework", "rejected", "hold_qty", "reason_code", "note",
			"created_by", "updated_by"},
		pgx.CopyFromRows(rows))
	return err
}

func copyStorage(ctx context.Context, tx pgx.Tx, ref reference, versionID string,
	start time.Time, s scale) error {

	products := take(ref.Products, s.Products)
	stores := take(ref.Warehouses, s.Warehouses)
	rows := make([][]any, 0, s.Days*len(stores)*len(products))
	for d := 0; d < s.Days; d++ {
		date := start.AddDate(0, 0, d)
		for _, w := range stores {
			for _, p := range products {
				open := 1000.0 + float64(d)*12
				in, out := 440.0, 400.0
				rows = append(rows, []any{
					uuid.NewString(), versionID, w, p, date, "ACTUAL",
					qty(open), qty(in), qty(0), qty(0), qty(0), qty(0), qty(0),
					qty(out), qty(0), qty(0), qty(0), qty(open + in - out), nil,
					"loadtest", "loadtest",
				})
			}
		}
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"daily_storage_plans"},
		[]string{"id", "version_id", "warehouse_id", "product_id", "business_date", "series",
			"beginning_balance", "production_receipt", "transfer_in", "transfer_out",
			"repack_in", "repack_out", "remelt_issue", "shipment_qty", "adjustment",
			"process_loss", "hold_qty", "ending_balance", "physical_balance",
			"created_by", "updated_by"},
		pgx.CopyFromRows(rows))
	return err
}

// copyDocuments writes the transaction volume: the posting ledger and the audit
// trail that goes with it, which are the two tables the specification singles
// out as high volume.
func copyDocuments(ctx context.Context, tx pgx.Tx, ref reference, start time.Time, s scale) error {
	products := take(ref.Products, s.Products)
	stores := take(ref.Warehouses, s.Warehouses)

	docs := make([][]any, 0, s.Days*s.DocsPerDay)
	items := make([][]any, 0, s.Days*s.DocsPerDay*s.ItemsPerDoc)
	audits := make([][]any, 0, s.Days*s.DocsPerDay)

	seq := 0
	for d := 0; d < s.Days; d++ {
		date := start.AddDate(0, 0, d)
		postedAt := date.Add(6 * time.Hour)
		for n := 0; n < s.DocsPerDay; n++ {
			seq++
			id := uuid.NewString()
			docs = append(docs, []any{
				id, fmt.Sprintf("LD-%d-%06d", start.Year(), seq), "RECEIPT", date,
				postedAt, ref.FactoryID, "load fixture", nil, false, nil, "",
				"loadtest",
			})
			for line := 0; line < s.ItemsPerDoc; line++ {
				items = append(items, []any{
					uuid.NewString(), id, line + 1,
					stores[(n+line)%len(stores)], products[(n+line)%len(products)],
					nil, qty(120.5), "TON", nil, "loadtest",
				})
			}
			audits = append(audits, []any{
				uuid.NewString(), postedAt, "loadtest", "POST_RECEIPT",
				"inventory_document", id, nil, `{"docType":"RECEIPT"}`, "",
				uuid.NewString(), "",
			})
		}
	}

	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"inventory_documents"},
		[]string{"id", "document_no", "doc_type", "business_date", "posted_at", "factory_id",
			"reference", "reversal_of", "reversed", "reason_code", "note", "created_by"},
		pgx.CopyFromRows(docs)); err != nil {
		return err
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"inventory_document_items"},
		[]string{"id", "document_id", "line_no", "warehouse_id", "product_id", "batch_id",
			"quantity", "uom", "to_warehouse", "created_by"},
		pgx.CopyFromRows(items)); err != nil {
		return err
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"audit_events"},
		[]string{"id", "occurred_at", "actor", "action", "entity", "entity_id",
			"before_state", "after_state", "reason", "correlation_id", "source_ip"},
		pgx.CopyFromRows(audits))
	return err
}

func take(all []string, n int) []string {
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// qty rounds to the three decimals the columns store, so COPY never has to
// round and a sum is the sum of what was written.
func qty(v float64) string {
	if v < 0 {
		v = 0
	}
	return fmt.Sprintf("%.3f", v)
}
