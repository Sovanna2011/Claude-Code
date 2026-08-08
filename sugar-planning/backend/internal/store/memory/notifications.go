package memory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type notifications struct{ s *Store }

// Notifications returns the inbox.
func (s *Store) Notifications() store.Notifications { return notifications{s} }

func (n notifications) List(_ context.Context, f store.NotificationFilter) (store.Page[domain.Notification], error) {
	n.s.lock()
	defer n.s.unlock()

	var items []domain.Notification
	for _, item := range n.s.d.notifications {
		if !matchesInbox(item, f) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(a, b int) bool { return items[a].CreatedAt.After(items[b].CreatedAt) })
	return paginate(items, store.ListOptions{Skip: f.Skip, Top: orDefaultTop(f.Top)}), nil
}

func (n notifications) Save(_ context.Context, item domain.Notification) (domain.Notification, error) {
	n.s.lock()
	defer n.s.unlock()

	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = nowUTC()
	}
	n.s.d.notifications[item.ID] = item
	return item, nil
}

func (n notifications) MarkRead(_ context.Context, id string, roles []string, at time.Time) error {
	n.s.lock()
	defer n.s.unlock()

	item, ok := n.s.d.notifications[id]
	if !ok || !contains(roles, item.Recipient) {
		return fmt.Errorf("%w: notification %s", domain.ErrNotFound, id)
	}
	if item.IsRead() {
		return nil
	}
	read := at
	item.ReadAt = &read
	n.s.d.notifications[id] = item
	return nil
}

func (n notifications) Unread(_ context.Context, f store.NotificationFilter) (int, error) {
	n.s.lock()
	defer n.s.unlock()

	f.UnreadOnly = true
	count := 0
	for _, item := range n.s.d.notifications {
		if matchesInbox(item, f) {
			count++
		}
	}
	return count, nil
}

// matchesInbox is the one place the inbox's addressing is decided, so the count
// in the badge and the list below it cannot disagree.
func matchesInbox(item domain.Notification, f store.NotificationFilter) bool {
	if f.UnreadOnly && item.IsRead() {
		return false
	}
	if !contains(f.Recipients, item.Recipient) {
		return false
	}
	// An empty scope is a caller entitled to every factory; a notification with
	// no factory is about the system rather than a plant, and everybody
	// addressed by its role sees it.
	if len(f.Factories) > 0 && item.FactoryID != "" && !contains(f.Factories, item.FactoryID) {
		return false
	}
	return true
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func (n notifications) ExistsSince(_ context.Context, key string, since time.Time) (bool, error) {
	n.s.lock()
	defer n.s.unlock()

	for _, item := range n.s.d.notifications {
		if item.DedupeKey() == key && !item.CreatedAt.Before(since) {
			return true, nil
		}
	}
	return false, nil
}
