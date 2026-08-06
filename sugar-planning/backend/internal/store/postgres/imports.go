package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type imports struct{ s *Store }

// Imports returns the mapping templates and the staging area.
func (s *Store) Imports() store.Imports { return imports{s} }

// ---------------------------------------------------------------------------
// Mappings
// ---------------------------------------------------------------------------

const mappingCols = `id, code, name, kind, columns, header_row, first_data_row,
	delimiter, date_format, decimal_comma, note, valid_from, valid_to, active,
	created_at, created_by, updated_at, updated_by, row_version`

func scanMapping(r scanner) (domain.ImportMapping, error) {
	var m domain.ImportMapping
	var kind string
	var columns []byte
	var v validityCols
	err := r.Scan(&m.ID, &m.Code, &m.Name, &kind, &columns, &m.HeaderRow, &m.FirstDataRow,
		&m.Delimiter, &m.DateFormat, &m.DecimalComma, &m.Note,
		&v.from, &v.to, &v.active,
		&m.CreatedAt, &m.CreatedBy, &m.UpdatedAt, &m.UpdatedBy, &m.RowVersion)
	if err != nil {
		return m, err
	}
	m.Kind, m.Validity = domain.ImportKind(kind), v.toDomain()
	if len(columns) > 0 {
		if err := json.Unmarshal(columns, &m.Columns); err != nil {
			return m, fmt.Errorf("the stored column mapping of %s is not readable: %w", m.Code, err)
		}
	}
	return m, nil
}

func (i imports) ListMappings(ctx context.Context, kind string) ([]domain.ImportMapping, error) {
	query := "SELECT " + mappingCols + " FROM import_mappings"
	args := []any{}
	if kind != "" {
		query += " WHERE kind = $1"
		args = append(args, kind)
	}
	query += " ORDER BY code"

	rows, err := i.s.q.Query(ctx, query, args...)
	if err != nil {
		return nil, mapError("import mapping", err)
	}
	defer rows.Close()

	out := []domain.ImportMapping{}
	for rows.Next() {
		m, err := scanMapping(rows)
		if err != nil {
			return nil, mapError("import mapping", err)
		}
		out = append(out, m)
	}
	return out, mapError("import mapping", rows.Err())
}

func (i imports) GetMapping(ctx context.Context, id string) (domain.ImportMapping, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.ImportMapping{}, fmt.Errorf("%w: import mapping %s", domain.ErrNotFound, id)
	}
	m, err := scanMapping(i.s.q.QueryRow(ctx,
		"SELECT "+mappingCols+" FROM import_mappings WHERE id = $1", id))
	return m, mapError("import mapping "+id, err)
}

func (i imports) MappingByCode(ctx context.Context, code string) (domain.ImportMapping, error) {
	m, err := scanMapping(i.s.q.QueryRow(ctx,
		"SELECT "+mappingCols+" FROM import_mappings WHERE code = $1", code))
	return m, mapError("import mapping "+code, err)
}

func (i imports) SaveMapping(ctx context.Context, m domain.ImportMapping, actor string) (domain.ImportMapping, error) {
	now := nowUTC()
	columns, err := json.Marshal(m.Columns)
	if err != nil {
		return domain.ImportMapping{}, fmt.Errorf("the column mapping could not be stored: %w", err)
	}

	if m.ID == "" {
		m.ID = uuid.NewString()
		saved, err := scanMapping(i.s.q.QueryRow(ctx, `INSERT INTO import_mappings
			(id, code, name, kind, columns, header_row, first_data_row, delimiter,
			 date_format, decimal_comma, note, valid_from, valid_to, active,
			 created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$15,$16,1)
			RETURNING `+mappingCols,
			m.ID, m.Code, m.Name, string(m.Kind), columns, m.HeaderRow, m.FirstDataRow,
			m.Delimiter, m.DateFormat, m.DecimalComma, m.Note,
			nd(m.ValidFrom), nd(m.ValidTo), m.Active, now, actor))
		return saved, mapError("import mapping "+m.Code, err)
	}

	saved, err := scanMapping(i.s.q.QueryRow(ctx, `UPDATE import_mappings SET
			code=$1, name=$2, kind=$3, columns=$4, header_row=$5, first_data_row=$6,
			delimiter=$7, date_format=$8, decimal_comma=$9, note=$10,
			valid_from=$11, valid_to=$12, active=$13,
			updated_at=$14, updated_by=$15, row_version = row_version + 1
		WHERE id=$16 AND ($17 = 0 OR row_version=$17)
		RETURNING `+mappingCols,
		m.Code, m.Name, string(m.Kind), columns, m.HeaderRow, m.FirstDataRow,
		m.Delimiter, m.DateFormat, m.DecimalComma, m.Note,
		nd(m.ValidFrom), nd(m.ValidTo), m.Active, now, actor, m.ID, m.RowVersion))
	if isNoRows(err) {
		return domain.ImportMapping{}, versionConflict(ctx, i.s,
			"import_mappings", "import mapping", m.ID, m.RowVersion)
	}
	return saved, mapError("import mapping "+m.Code, err)
}

// ---------------------------------------------------------------------------
// Jobs and their staged rows
// ---------------------------------------------------------------------------

const importJobCols = `id, kind, mapping_id, version_id, factory_id, file_name, status,
	total_rows, valid_rows, error_rows, errors, kind_note, committed_at, committed_by,
	created_at, created_by`

func scanImportJob(r scanner) (domain.ImportJob, error) {
	var j domain.ImportJob
	var kind string
	var mapping, version, factory *string
	var errs []byte
	err := r.Scan(&j.ID, &kind, &mapping, &version, &factory, &j.FileName, &j.Status,
		&j.TotalRows, &j.ValidRows, &j.ErrorRows, &errs, &j.Note,
		&j.CommittedAt, &j.CommittedBy, &j.CreatedAt, &j.CreatedBy)
	if err != nil {
		return j, err
	}
	j.Kind = domain.ImportKind(kind)
	j.MappingID, j.VersionID, j.FactoryID = ds(mapping), ds(version), ds(factory)
	if len(errs) > 0 {
		if err := json.Unmarshal(errs, &j.Errors); err != nil {
			return j, fmt.Errorf("the stored errors of import %s are not readable: %w", j.ID, err)
		}
	}
	return j, nil
}

func (i imports) ListJobs(ctx context.Context, f store.ImportFilter) (store.Page[domain.ImportJob], error) {
	w := &execWhere{}
	if f.Kind != "" {
		w.eq("kind", f.Kind)
	}
	if f.Status != "" {
		w.eq("status", f.Status)
	}
	if f.VersionID != "" {
		w.eq("version_id", f.VersionID)
	}
	clause := w.sql()

	var total int
	if err := i.s.q.QueryRow(ctx, "SELECT count(*) FROM import_jobs"+clause, w.args...).
		Scan(&total); err != nil {
		return store.Page[domain.ImportJob]{}, mapError("import job", err)
	}
	limit := w.limit(store.ExecutionFilter{Skip: f.Skip, Top: f.Top})

	rows, err := i.s.q.Query(ctx, "SELECT "+importJobCols+" FROM import_jobs"+clause+
		" ORDER BY created_at DESC"+limit, w.args...)
	if err != nil {
		return store.Page[domain.ImportJob]{}, mapError("import job", err)
	}
	defer rows.Close()

	items := []domain.ImportJob{}
	for rows.Next() {
		j, err := scanImportJob(rows)
		if err != nil {
			return store.Page[domain.ImportJob]{}, mapError("import job", err)
		}
		items = append(items, j)
	}
	return store.Page[domain.ImportJob]{Items: items, Count: total}, mapError("import job", rows.Err())
}

func (i imports) GetJob(ctx context.Context, id string) (domain.ImportJob, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.ImportJob{}, fmt.Errorf("%w: import job %s", domain.ErrNotFound, id)
	}
	j, err := scanImportJob(i.s.q.QueryRow(ctx,
		"SELECT "+importJobCols+" FROM import_jobs WHERE id = $1", id))
	return j, mapError("import job "+id, err)
}

func (i imports) SaveJob(ctx context.Context, j domain.ImportJob, actor string) (domain.ImportJob, error) {
	errs, err := json.Marshal(orEmptyErrors(j.Errors))
	if err != nil {
		return domain.ImportJob{}, fmt.Errorf("the import's errors could not be stored: %w", err)
	}

	if j.ID == "" {
		j.ID = uuid.NewString()
		saved, err := scanImportJob(i.s.q.QueryRow(ctx, `INSERT INTO import_jobs
			(id, kind, mapping_id, version_id, factory_id, file_name, status,
			 total_rows, valid_rows, error_rows, errors, kind_note,
			 committed_at, committed_by, created_at, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
			RETURNING `+importJobCols,
			j.ID, string(j.Kind), nu(j.MappingID), nu(j.VersionID), nu(j.FactoryID),
			j.FileName, j.Status, j.TotalRows, j.ValidRows, j.ErrorRows, errs, j.Note,
			j.CommittedAt, j.CommittedBy, nowUTC(), actor))
		return saved, mapError("import job", err)
	}

	saved, err := scanImportJob(i.s.q.QueryRow(ctx, `UPDATE import_jobs SET
			kind=$1, mapping_id=$2, version_id=$3, factory_id=$4, file_name=$5, status=$6,
			total_rows=$7, valid_rows=$8, error_rows=$9, errors=$10, kind_note=$11,
			committed_at=$12, committed_by=$13
		WHERE id=$14
		RETURNING `+importJobCols,
		string(j.Kind), nu(j.MappingID), nu(j.VersionID), nu(j.FactoryID), j.FileName,
		j.Status, j.TotalRows, j.ValidRows, j.ErrorRows, errs, j.Note,
		j.CommittedAt, j.CommittedBy, j.ID))
	if isNoRows(err) {
		return domain.ImportJob{}, fmt.Errorf("%w: import job %s", domain.ErrNotFound, j.ID)
	}
	return saved, mapError("import job", err)
}

func orEmptyErrors(e []domain.FieldError) []domain.FieldError {
	if e == nil {
		return []domain.FieldError{}
	}
	return e
}

func (i imports) SaveRows(ctx context.Context, jobID string, rows []domain.ImportRow) error {
	if _, err := i.s.q.Exec(ctx, "DELETE FROM import_rows WHERE job_id = $1", jobID); err != nil {
		return mapError("import row", err)
	}
	for _, r := range rows {
		values, err := json.Marshal(r.Values)
		if err != nil {
			return fmt.Errorf("row %d could not be stored: %w", r.RowNo, err)
		}
		errs, err := json.Marshal(orEmptyErrors(r.Errors))
		if err != nil {
			return fmt.Errorf("row %d could not be stored: %w", r.RowNo, err)
		}
		if _, err := i.s.q.Exec(ctx, `INSERT INTO import_rows
			(id, job_id, row_no, values, errors, duplicate, replaces)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			uuid.NewString(), jobID, r.RowNo, values, errs, r.Duplicate, r.Replaces); err != nil {
			return mapError("import row", err)
		}
	}
	return nil
}

func (i imports) Rows(ctx context.Context, jobID string, errorsOnly bool, skip, top int) (store.Page[domain.ImportRow], error) {
	where := " WHERE job_id = $1"
	if errorsOnly {
		where += " AND jsonb_array_length(errors) > 0"
	}

	var total int
	if err := i.s.q.QueryRow(ctx, "SELECT count(*) FROM import_rows"+where, jobID).
		Scan(&total); err != nil {
		return store.Page[domain.ImportRow]{}, mapError("import row", err)
	}
	if top <= 0 || top > 5000 {
		top = 5000
	}

	rows, err := i.s.q.Query(ctx, `SELECT row_no, values, errors, duplicate, replaces
		FROM import_rows`+where+` ORDER BY row_no LIMIT $2 OFFSET $3`, jobID, top, skip)
	if err != nil {
		return store.Page[domain.ImportRow]{}, mapError("import row", err)
	}
	defer rows.Close()

	items := []domain.ImportRow{}
	for rows.Next() {
		var r domain.ImportRow
		var values, errs []byte
		if err := rows.Scan(&r.RowNo, &values, &errs, &r.Duplicate, &r.Replaces); err != nil {
			return store.Page[domain.ImportRow]{}, mapError("import row", err)
		}
		if err := json.Unmarshal(values, &r.Values); err != nil {
			return store.Page[domain.ImportRow]{}, fmt.Errorf(
				"the stored values of row %d are not readable: %w", r.RowNo, err)
		}
		if err := json.Unmarshal(errs, &r.Errors); err != nil {
			return store.Page[domain.ImportRow]{}, fmt.Errorf(
				"the stored errors of row %d are not readable: %w", r.RowNo, err)
		}
		items = append(items, r)
	}
	return store.Page[domain.ImportRow]{Items: items, Count: total}, mapError("import row", rows.Err())
}
