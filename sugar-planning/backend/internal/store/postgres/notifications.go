package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type notifications struct{ s *Store }

// Notifications returns the inbox.
func (s *Store) Notifications() store.Notifications { return notifications{s} }

const notificationCols = `id, recipient, factory_id, severity, code, title, detail,
	entity, entity_id, read_at, created_at`

func scanNotification(r scanner) (domain.Notification, error) {
	var n domain.Notification
	var severity string
	var factory *string
	err := r.Scan(&n.ID, &n.Recipient, &factory, &severity, &n.Code, &n.Title, &n.Detail,
		&n.Entity, &n.EntityID, &n.ReadAt, &n.CreatedAt)
	n.Severity, n.FactoryID = domain.Severity(severity), ds(factory)
	return n, err
}

func (n notifications) List(ctx context.Context, f store.NotificationFilter) (store.Page[domain.Notification], error) {
	w := inboxWhere(f)
	clause := w.sql()

	var total int
	if err := n.s.q.QueryRow(ctx, "SELECT count(*) FROM notifications"+clause, w.args...).
		Scan(&total); err != nil {
		return store.Page[domain.Notification]{}, mapError("notification", err)
	}
	limit := w.limit(store.ExecutionFilter{Skip: f.Skip, Top: f.Top})

	rows, err := n.s.q.Query(ctx, "SELECT "+notificationCols+" FROM notifications"+clause+
		" ORDER BY created_at DESC"+limit, w.args...)
	if err != nil {
		return store.Page[domain.Notification]{}, mapError("notification", err)
	}
	defer rows.Close()

	items := []domain.Notification{}
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return store.Page[domain.Notification]{}, mapError("notification", err)
		}
		items = append(items, item)
	}
	return store.Page[domain.Notification]{Items: items, Count: total},
		mapError("notification", rows.Err())
}

// inboxWhere is the one place the inbox's addressing is decided, so the count in
// the badge and the list below it cannot disagree.
func inboxWhere(f store.NotificationFilter) *execWhere {
	w := &execWhere{}
	// An unaddressed inbox is empty rather than everybody's.
	w.in("recipient", f.Recipients)
	if len(f.Recipients) == 0 {
		w.raw("false")
	}
	if len(f.Factories) > 0 {
		// A notification with no factory is about the system rather than a
		// plant, and everybody addressed by its role sees it.
		w.args = append(w.args, f.Factories)
		w.raw(fmt.Sprintf("(factory_id IS NULL OR factory_id = ANY($%d))", len(w.args)))
	}
	if f.UnreadOnly {
		w.raw("read_at IS NULL")
	}
	return w
}

func (n notifications) Save(ctx context.Context, item domain.Notification) (domain.Notification, error) {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = nowUTC()
	}
	if item.Severity == "" {
		item.Severity = domain.SeverityInfo
	}
	saved, err := scanNotification(n.s.q.QueryRow(ctx, `INSERT INTO notifications
		(id, recipient, factory_id, severity, code, title, detail, entity, entity_id,
		 read_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING `+notificationCols,
		item.ID, item.Recipient, nu(item.FactoryID), string(item.Severity), item.Code,
		item.Title, item.Detail, item.Entity, item.EntityID, item.ReadAt, item.CreatedAt))
	return saved, mapError("notification", err)
}

func (n notifications) MarkRead(ctx context.Context, id string, roles []string, at time.Time) error {
	if _, err := uuid.Parse(id); err != nil || len(roles) == 0 {
		return fmt.Errorf("%w: notification %s", domain.ErrNotFound, id)
	}
	// The reader's roles are part of the WHERE rather than checked afterwards:
	// whose inbox holds what is not a question this system answers to a caller.
	tag, err := n.s.q.Exec(ctx, `UPDATE notifications SET read_at = $1
		WHERE id = $2 AND recipient = ANY($3) AND read_at IS NULL`, at, id, roles)
	if err != nil {
		return mapError("notification", err)
	}
	if tag.RowsAffected() == 0 {
		// Either it is not theirs, it does not exist, or it was already read.
		// Already-read is not an error, so the distinction is made here.
		var exists bool
		if err := n.s.q.QueryRow(ctx,
			`SELECT true FROM notifications WHERE id = $1 AND recipient = ANY($2)`,
			id, roles).Scan(&exists); err != nil {
			return fmt.Errorf("%w: notification %s", domain.ErrNotFound, id)
		}
		return nil
	}
	return nil
}

func (n notifications) Unread(ctx context.Context, f store.NotificationFilter) (int, error) {
	f.UnreadOnly = true
	w := inboxWhere(f)
	var count int
	err := n.s.q.QueryRow(ctx, "SELECT count(*) FROM notifications"+w.sql(), w.args...).Scan(&count)
	return count, mapError("notification", err)
}

// ExistsSince asks the database rather than reading the inbox out, because the
// alert evaluator asks this once per alert per tick.
func (n notifications) ExistsSince(ctx context.Context, key string, since time.Time) (bool, error) {
	var exists bool
	err := n.s.q.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM notifications
		WHERE created_at >= $2
		  AND recipient || '|' || coalesce(factory_id::text, '') || '|' || code
		      || '|' || entity || '|' || entity_id = $1)`,
		key, since).Scan(&exists)
	return exists, mapError("notification", err)
}
