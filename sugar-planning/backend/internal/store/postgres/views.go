package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type savedViews struct{ s *Store }

// SavedViews returns the variant store.
func (s *Store) SavedViews() store.SavedViews { return savedViews{s} }

const savedViewCols = `id, owner, page, name, factory_id, shared, is_default, payload,
	created_at, created_by, updated_at, updated_by, row_version`

func scanSavedView(r scanner) (domain.SavedView, error) {
	var v domain.SavedView
	var factory *string
	err := r.Scan(&v.ID, &v.Owner, &v.Page, &v.Name, &factory, &v.Shared, &v.IsDefault,
		&v.Payload, &v.CreatedAt, &v.CreatedBy, &v.UpdatedAt, &v.UpdatedBy, &v.RowVersion)
	v.FactoryID = ds(factory)
	return v, err
}

// List returns what the caller may see: their own views, plus the shared ones
// from factories within their scope.
//
// The ordering puts a caller's own views first and then sorts by name, so the
// list a variant control renders is stable and the personal ones are where
// somebody looks for them.
func (v savedViews) List(ctx context.Context, f store.SavedViewFilter) ([]domain.SavedView, error) {
	if f.Owner == "" {
		return []domain.SavedView{}, nil
	}

	args := []any{f.Owner}
	clause := ` WHERE (owner = $1 OR (shared AND (factory_id IS NULL`
	if len(f.Factories) > 0 {
		args = append(args, f.Factories)
		clause += fmt.Sprintf(" OR factory_id = ANY($%d)", len(args))
	}
	clause += `)))`
	if f.Page != "" {
		args = append(args, f.Page)
		clause += fmt.Sprintf(" AND page = $%d", len(args))
	}

	rows, err := v.s.q.Query(ctx, "SELECT "+savedViewCols+" FROM saved_views"+clause+
		" ORDER BY (owner <> $1), lower(name)", args...)
	if err != nil {
		return nil, mapError("saved view", err)
	}
	defer rows.Close()

	items := []domain.SavedView{}
	for rows.Next() {
		item, err := scanSavedView(rows)
		if err != nil {
			return nil, mapError("saved view", err)
		}
		items = append(items, item)
	}
	return items, mapError("saved view", rows.Err())
}

func (v savedViews) Get(ctx context.Context, id string) (domain.SavedView, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.SavedView{}, fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
	}
	item, err := scanSavedView(v.s.q.QueryRow(ctx,
		"SELECT "+savedViewCols+" FROM saved_views WHERE id = $1", id))
	if err != nil {
		return domain.SavedView{}, mapError("saved view "+id, err)
	}
	return item, nil
}

// Save inserts, or replaces the caller's view of the same name on the same page.
//
// Saving over a name is what "save" on a variant means, so it is one statement
// rather than a read followed by a decision: two tabs saving the same name at
// once would otherwise both insert, and the unique index would refuse the loser
// with an error nobody asked for.
func (v savedViews) Save(ctx context.Context, item domain.SavedView) (domain.SavedView, error) {
	// The default is cleared first, because the partial unique index would
	// refuse a second default before the upsert had a chance to replace it.
	if item.IsDefault {
		if _, err := v.s.q.Exec(ctx,
			`UPDATE saved_views SET is_default = false, updated_at = now()
			 WHERE owner = $1 AND page = $2 AND is_default`,
			item.Owner, item.Page); err != nil {
			return domain.SavedView{}, mapError("saved view", err)
		}
	}

	saved, err := scanSavedView(v.s.q.QueryRow(ctx, `
		INSERT INTO saved_views
			(owner, page, name, factory_id, shared, is_default, payload,
			 created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7, now(), $1, now(), $1, 1)
		ON CONFLICT (owner, page, lower(name)) DO UPDATE SET
			factory_id  = EXCLUDED.factory_id,
			shared      = EXCLUDED.shared,
			is_default  = EXCLUDED.is_default,
			payload     = EXCLUDED.payload,
			updated_at  = now(),
			updated_by  = EXCLUDED.updated_by,
			row_version = saved_views.row_version + 1
		RETURNING `+savedViewCols,
		item.Owner, item.Page, item.Name, nu(item.FactoryID),
		item.Shared, item.IsDefault, []byte(item.Payload)))
	if err != nil {
		return domain.SavedView{}, mapError("saved view "+item.Name, err)
	}
	return saved, nil
}

// Delete removes one view. The owner is part of the match, so somebody else's
// variant is not found rather than forbidden: who has which variants is not a
// question this answers to a caller.
func (v savedViews) Delete(ctx context.Context, id, owner string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
	}
	tag, err := v.s.q.Exec(ctx,
		"DELETE FROM saved_views WHERE id = $1 AND owner = $2", id, owner)
	if err != nil {
		return mapError("saved view", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
	}
	return nil
}

// SetDefault makes one view the caller's default for its page.
//
// The previous default is cleared first, in its own statement inside the same
// transaction. It was written as one statement with a data-modifying CTE, which
// is wrong: the branches of a CTE all see the same snapshot and run in an
// unspecified order, so the new default could be written before the old one was
// cleared and the partial unique index - checked per row, not at commit - would
// refuse it. A partial unique index cannot be deferred, so the ordering has to
// be real rather than declared.
func (v savedViews) SetDefault(ctx context.Context, id, owner string, on bool) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
	}
	return v.s.InTx(ctx, func(tx store.Store) error {
		q := tx.(*Store).q
		if on {
			if _, err := q.Exec(ctx, `
				UPDATE saved_views SET is_default = false, updated_at = now()
				WHERE owner = $2 AND is_default AND id <> $1
				  AND page = (SELECT page FROM saved_views WHERE id = $1 AND owner = $2)`,
				id, owner); err != nil {
				return mapError("saved view", err)
			}
		}
		tag, err := q.Exec(ctx, `
			UPDATE saved_views SET is_default = $3, updated_at = now(),
				row_version = row_version + 1
			WHERE id = $1 AND owner = $2`, id, owner, on)
		if err != nil {
			return mapError("saved view", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
		}
		return nil
	})
}
