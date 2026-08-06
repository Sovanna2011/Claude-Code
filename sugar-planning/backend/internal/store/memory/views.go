package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type savedViews struct{ s *Store }

// SavedViews returns the variant store.
func (s *Store) SavedViews() store.SavedViews { return savedViews{s} }

func (v savedViews) List(_ context.Context, f store.SavedViewFilter) ([]domain.SavedView, error) {
	v.s.lock()
	defer v.s.unlock()

	if f.Owner == "" {
		return []domain.SavedView{}, nil
	}
	items := []domain.SavedView{}
	for _, item := range v.s.d.savedViews {
		if f.Page != "" && item.Page != f.Page {
			continue
		}
		if !visibleView(item, f) {
			continue
		}
		items = append(items, item)
	}
	sortViews(items)
	return items, nil
}

// visibleView is the one place a view's visibility is decided.
//
// Mine always; somebody else's only if they shared it and it belongs to a
// factory I can see. A shared view with no factory is about the screen rather
// than a plant, so everybody gets it.
func visibleView(item domain.SavedView, f store.SavedViewFilter) bool {
	if item.Owner == f.Owner {
		return true
	}
	if !item.Shared {
		return false
	}
	if len(f.Factories) == 0 || item.FactoryID == "" {
		return true
	}
	return contains(f.Factories, item.FactoryID)
}

// sortViews puts a caller's own views first and orders each group by name, so
// the list a variant control renders is stable and the personal ones are where
// somebody looks for them.
func sortViews(items []domain.SavedView) {
	sort.Slice(items, func(a, b int) bool {
		if items[a].Shared != items[b].Shared {
			return !items[a].Shared
		}
		return strings.ToLower(items[a].Name) < strings.ToLower(items[b].Name)
	})
}

func (v savedViews) Get(_ context.Context, id string) (domain.SavedView, error) {
	v.s.lock()
	defer v.s.unlock()

	item, ok := v.s.d.savedViews[id]
	if !ok {
		return domain.SavedView{}, fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
	}
	return item, nil
}

func (v savedViews) Save(_ context.Context, item domain.SavedView) (domain.SavedView, error) {
	v.s.lock()
	defer v.s.unlock()

	now := nowUTC()
	// Saving over a name replaces it, which is what "save" on a variant means.
	// Finding the existing row here rather than in the service keeps the two
	// store implementations agreeing about what a duplicate name is.
	for _, existing := range v.s.d.savedViews {
		if existing.Owner == item.Owner && existing.Page == item.Page &&
			strings.EqualFold(existing.Name, item.Name) && existing.ID != item.ID {
			item.ID = existing.ID
			item.CreatedAt, item.CreatedBy = existing.CreatedAt, existing.CreatedBy
			item.RowVersion = existing.RowVersion
			break
		}
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
		item.CreatedAt, item.CreatedBy = now, item.Owner
		item.RowVersion = 0
	}
	item.UpdatedAt, item.UpdatedBy = now, item.Owner
	item.RowVersion++
	if item.CreatedAt.IsZero() {
		item.CreatedAt, item.CreatedBy = now, item.Owner
	}

	if item.IsDefault {
		v.clearDefault(item.Owner, item.Page, item.ID)
	}
	v.s.d.savedViews[item.ID] = item
	return item, nil
}

func (v savedViews) Delete(_ context.Context, id, owner string) error {
	v.s.lock()
	defer v.s.unlock()

	item, ok := v.s.d.savedViews[id]
	// The owner is part of the match, so somebody else's variant is not found
	// rather than forbidden: who has which variants is not a question this
	// answers to a caller.
	if !ok || item.Owner != owner {
		return fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
	}
	delete(v.s.d.savedViews, id)
	return nil
}

func (v savedViews) SetDefault(_ context.Context, id, owner string, on bool) error {
	v.s.lock()
	defer v.s.unlock()

	item, ok := v.s.d.savedViews[id]
	if !ok || item.Owner != owner {
		return fmt.Errorf("%w: saved view %s", domain.ErrNotFound, id)
	}
	if on {
		v.clearDefault(owner, item.Page, id)
	}
	item.IsDefault = on
	item.UpdatedAt = nowUTC()
	item.RowVersion++
	v.s.d.savedViews[id] = item
	return nil
}

// clearDefault unsets any other default for this owner and page. The caller
// holds the lock.
func (v savedViews) clearDefault(owner, page, keep string) {
	for id, other := range v.s.d.savedViews {
		if id == keep || other.Owner != owner || other.Page != page || !other.IsDefault {
			continue
		}
		other.IsDefault = false
		v.s.d.savedViews[id] = other
	}
}
