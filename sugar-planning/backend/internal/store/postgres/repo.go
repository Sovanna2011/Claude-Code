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

// pgSpec describes how one master-data entity maps to its table. Everything
// entity-specific lives here; the repository below is written once.
type pgSpec[T any] struct {
	name      string
	table     string
	cols      []string // business columns, in the order used by scan and values
	scan      func(scanner) (T, error)
	values    func(T) []any
	id        func(*T) *string
	code      func(T) string
	codeCol   string
	audit     func(*T) *domain.AuditFields
	parentCol string   // column used by ListOptions.ParentID
	textCols  []string // columns searched by ListOptions.Search
	activeCol string   // empty when the entity cannot be deactivated
}

type pgRepo[T any] struct {
	s  *Store
	sp pgSpec[T]
}

func (r pgRepo[T]) selectSQL() string {
	return fmt.Sprintf("SELECT id, %s, %s FROM %s",
		strings.Join(r.sp.cols, ", "), auditCols, r.sp.table)
}

func (r pgRepo[T]) List(ctx context.Context, opts store.ListOptions) (store.Page[T], error) {
	opts = opts.Normalise()
	var where []string
	var args []any

	if opts.Active != nil && r.sp.activeCol != "" {
		args = append(args, *opts.Active)
		where = append(where, fmt.Sprintf("%s = $%d", r.sp.activeCol, len(args)))
	}
	if opts.ParentID != "" && r.sp.parentCol != "" {
		args = append(args, opts.ParentID)
		where = append(where, fmt.Sprintf("%s = $%d", r.sp.parentCol, len(args)))
	}
	if opts.Search != "" && len(r.sp.textCols) > 0 {
		args = append(args, "%"+strings.ToLower(opts.Search)+"%")
		var ors []string
		for _, c := range r.sp.textCols {
			ors = append(ors, fmt.Sprintf("lower(%s) LIKE $%d", c, len(args)))
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	countSQL := fmt.Sprintf("SELECT count(*) FROM %s%s", r.sp.table, clause)
	if err := r.s.q.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return store.Page[T]{}, mapError(r.sp.name, err)
	}

	args = append(args, opts.Top, opts.Skip)
	listSQL := fmt.Sprintf("%s%s ORDER BY %s LIMIT $%d OFFSET $%d",
		r.selectSQL(), clause, r.sp.codeCol, len(args)-1, len(args))

	rows, err := r.s.q.Query(ctx, listSQL, args...)
	if err != nil {
		return store.Page[T]{}, mapError(r.sp.name, err)
	}
	defer rows.Close()

	items := []T{}
	for rows.Next() {
		v, err := r.sp.scan(rows)
		if err != nil {
			return store.Page[T]{}, mapError(r.sp.name, err)
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return store.Page[T]{}, mapError(r.sp.name, err)
	}
	return store.Page[T]{Items: items, Count: total}, nil
}

func (r pgRepo[T]) Get(ctx context.Context, id string) (T, error) {
	var zero T
	if _, err := uuid.Parse(id); err != nil {
		return zero, fmt.Errorf("%w: %s %s", domain.ErrNotFound, r.sp.name, id)
	}
	row := r.s.q.QueryRow(ctx, r.selectSQL()+" WHERE id = $1", id)
	v, err := r.sp.scan(row)
	if err != nil {
		return zero, mapError(r.sp.name+" "+id, err)
	}
	return v, nil
}

func (r pgRepo[T]) GetByCode(ctx context.Context, code string) (T, error) {
	var zero T
	row := r.s.q.QueryRow(ctx, fmt.Sprintf("%s WHERE %s = $1", r.selectSQL(), r.sp.codeCol), code)
	v, err := r.sp.scan(row)
	if err != nil {
		return zero, mapError(r.sp.name+" "+code, err)
	}
	return v, nil
}

func (r pgRepo[T]) Save(ctx context.Context, entity T, actor string) (T, error) {
	var zero T
	idPtr := r.sp.id(&entity)
	aud := r.sp.audit(&entity)
	now := time.Now().UTC()

	if *idPtr == "" {
		*idPtr = uuid.NewString()
		aud.CreatedAt, aud.CreatedBy = now, actor
		aud.UpdatedAt, aud.UpdatedBy, aud.RowVersion = now, actor, 1

		args := append([]any{*idPtr}, r.sp.values(entity)...)
		args = append(args, now, actor, now, actor, int64(1))
		sql := fmt.Sprintf("INSERT INTO %s (id, %s, %s) VALUES (%s)",
			r.sp.table, strings.Join(r.sp.cols, ", "), auditCols, placeholders(len(args)))
		if _, err := r.s.q.Exec(ctx, sql, args...); err != nil {
			*idPtr = ""
			return zero, mapError(r.sp.name+" "+r.sp.code(entity), err)
		}
		return entity, nil
	}

	// Update guarded by row_version: the WHERE clause is the concurrency check.
	values := r.sp.values(entity)
	args := append([]any{}, values...)
	args = append(args, now, actor, *idPtr, aud.RowVersion)
	sql := fmt.Sprintf(
		"UPDATE %s SET %s, updated_at = $%d, updated_by = $%d, row_version = row_version + 1 "+
			"WHERE id = $%d AND row_version = $%d RETURNING row_version, created_at, created_by",
		r.sp.table, assignments(r.sp.cols, 1), len(values)+1, len(values)+2, len(values)+3, len(values)+4)

	var newVersion int64
	var createdAt time.Time
	var createdBy string
	err := r.s.q.QueryRow(ctx, sql, args...).Scan(&newVersion, &createdAt, &createdBy)
	if err != nil {
		if isNoRows(err) {
			return zero, r.conflictOrMissing(ctx, *idPtr, aud.RowVersion)
		}
		return zero, mapError(r.sp.name+" "+r.sp.code(entity), err)
	}
	aud.RowVersion, aud.CreatedAt, aud.CreatedBy = newVersion, createdAt, createdBy
	aud.UpdatedAt, aud.UpdatedBy = now, actor
	return entity, nil
}

func (r pgRepo[T]) Deactivate(ctx context.Context, id string, rowVersion int64, actor string) error {
	if r.sp.activeCol == "" {
		return fmt.Errorf("%w: %s cannot be deactivated", domain.ErrValidation, r.sp.name)
	}
	sql := fmt.Sprintf(
		"UPDATE %s SET %s = false, updated_at = $1, updated_by = $2, row_version = row_version + 1 "+
			"WHERE id = $3 AND row_version = $4", r.sp.table, r.sp.activeCol)
	tag, err := r.s.q.Exec(ctx, sql, time.Now().UTC(), actor, id, rowVersion)
	if err != nil {
		return mapError(r.sp.name, err)
	}
	if tag.RowsAffected() == 0 {
		return r.conflictOrMissing(ctx, id, rowVersion)
	}
	return nil
}

// conflictOrMissing distinguishes "the row is gone" from "somebody else
// changed it", so the client gets 404 or 412 rather than a bare failure.
func (r pgRepo[T]) conflictOrMissing(ctx context.Context, id string, expected int64) error {
	var current int64
	var updatedBy string
	err := r.s.q.QueryRow(ctx,
		fmt.Sprintf("SELECT row_version, updated_by FROM %s WHERE id = $1", r.sp.table), id).
		Scan(&current, &updatedBy)
	if err != nil {
		if isNoRows(err) {
			return fmt.Errorf("%w: %s %s", domain.ErrNotFound, r.sp.name, id)
		}
		return mapError(r.sp.name, err)
	}
	return fmt.Errorf("%w: %s %s was changed by %s (version %d, you have %d)",
		domain.ErrConflict, r.sp.name, id, updatedBy, current, expected)
}

func isNoRows(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no rows in result set")
}
