package storetest

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// The outbox is the one piece of the system whose whole value is a promise
// about failure: an event is written with the change it describes, survives a
// crash, and is retried on a widening backoff until it is delivered or until a
// person is asked to look. Both stores have to keep that promise identically,
// so the promise is tested here rather than against one implementation.

func testOutbox(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	now := time.Date(2027, 1, 12, 6, 0, 0, 0, time.UTC)

	append := func(o store.Outbox, id string, topic domain.Topic, at time.Time) {
		t.Helper()
		must(t, o.Append(ctx, domain.OutboxEvent{
			ID: id, Topic: topic, Payload: `{"documentNo":"MD-1"}`,
			CreatedAt: at, CorrelationID: "corr-" + id,
		}), "append "+id)
	}

	append(s.Outbox(), newID(1), domain.TopicStockPosted, now)
	append(s.Outbox(), newID(2), domain.TopicQualityFailed, now.Add(time.Minute))

	// A brand new event is due at once: nobody waits a second to hear that a
	// shipment left.
	due, err := s.Outbox().Due(ctx, now.Add(time.Minute), 10)
	must(t, err, "due")
	if len(due) != 2 {
		t.Fatalf("both new events are due, got %d", len(due))
	}
	if due[0].ID != newID(1) {
		t.Errorf("a backlog drains oldest first, got %s", due[0].ID)
	}
	// The payload survives as JSON rather than byte for byte: PostgreSQL stores
	// it as jsonb, which validates it and normalises the whitespace. That is a
	// trade worth making - a malformed payload is caught when it is written
	// rather than by a consumer at three in the morning - and it costs nothing,
	// because what a consumer deduplicates on is the envelope id, not the bytes.
	if !sameJSON(due[0].Payload, `{"documentNo":"MD-1"}`) ||
		due[0].CorrelationID != "corr-"+newID(1) {
		t.Errorf("the payload and correlation id must survive the round trip: %+v", due[0])
	}

	one, err := s.Outbox().Get(ctx, newID(1))
	must(t, err, "get one event")
	if one.Topic != domain.TopicStockPosted {
		t.Errorf("reading one event must return it whole, got %+v", one)
	}
	if _, err := s.Outbox().Get(ctx, newID(8)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an event that does not exist must be not found, got %v", err)
	}

	// A delivery that failed is not retried immediately.
	must(t, s.Outbox().MarkFailed(ctx, newID(1), now.Add(time.Minute), "connection refused"),
		"mark failed")
	due, err = s.Outbox().Due(ctx, now.Add(time.Minute), 10)
	must(t, err, "due after failure")
	if len(due) != 1 || due[0].ID != newID(2) {
		t.Fatalf("an event that just failed is not due again yet, got %v", ids(due))
	}

	// It is due once its backoff has elapsed, and it carries what went wrong.
	due, err = s.Outbox().Due(ctx, now.Add(time.Minute).Add(2*time.Second), 10)
	must(t, err, "due after the backoff")
	if len(due) != 2 {
		t.Fatalf("the backoff has elapsed, so both are due, got %v", ids(due))
	}
	failed := find(due, newID(1))
	if failed.Attempts != 1 || failed.LastError != "connection refused" {
		t.Errorf("the failure must be recorded: attempts=%d error=%q",
			failed.Attempts, failed.LastError)
	}

	// A published event is finished with.
	published := now.Add(time.Hour)
	must(t, s.Outbox().MarkPublished(ctx, newID(2), published), "mark published")
	due, err = s.Outbox().Due(ctx, published, 10)
	must(t, err, "due after publishing")
	if len(due) != 1 || due[0].ID != newID(1) {
		t.Fatalf("a published event is never due again, got %v", ids(due))
	}
	if err := s.Outbox().MarkPublished(ctx, newID(2), published); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("publishing an already published event must not silently succeed, got %v", err)
	}

	// limit is honoured, because a dispatcher takes a batch rather than a season.
	append(s.Outbox(), newID(3), domain.TopicPlanReleased, now)
	due, err = s.Outbox().Due(ctx, published, 1)
	must(t, err, "due with a limit")
	if len(due) != 1 {
		t.Fatalf("the limit must be honoured, got %d", len(due))
	}

	// An event that has failed its way to the ceiling stops being retried and
	// waits for a person. It is not discarded: the last error is the reason
	// anybody can say what went wrong.
	at := published
	for i := 0; i < domain.MaxOutboxAttempts; i++ {
		at = at.Add(2 * time.Hour)
		must(t, s.Outbox().MarkFailed(ctx, newID(3), at, "ERP rejected the document"),
			"exhaust the retries")
	}
	due, err = s.Outbox().Due(ctx, at.Add(24*time.Hour), 10)
	must(t, err, "due after exhaustion")
	if find(due, newID(3)).ID != "" {
		t.Error("an exhausted event must stop being retried automatically")
	}

	page, err := s.Outbox().List(ctx, store.OutboxFilter{Exhausted: true})
	must(t, err, "list exhausted")
	if len(page.Items) != 1 || page.Items[0].ID != newID(3) {
		t.Fatalf("the exhausted event must still be listed for a person, got %v", ids(page.Items))
	}
	if page.Items[0].LastError != "ERP rejected the document" {
		t.Errorf("the last error is what a person reads, got %q", page.Items[0].LastError)
	}

	page, err = s.Outbox().List(ctx, store.OutboxFilter{Unpublished: true})
	must(t, err, "list unpublished")
	if len(page.Items) != 2 {
		t.Errorf("two events are still unpublished, got %v", ids(page.Items))
	}
	page, err = s.Outbox().List(ctx, store.OutboxFilter{Topic: string(domain.TopicQualityFailed)})
	must(t, err, "list by topic")
	if len(page.Items) != 1 || page.Items[0].ID != newID(2) {
		t.Errorf("filtering by topic must select one event, got %v", ids(page.Items))
	}
}

// testOutboxRollback is the reason the outbox exists at all: an event describing
// a change that was rolled back must roll back with it.
func testOutboxRollback(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	boom := errors.New("the posting failed after the event was written")

	err := s.InTx(ctx, func(tx store.Store) error {
		if err := tx.Outbox().Append(ctx, domain.OutboxEvent{
			ID: newID(9), Topic: domain.TopicStockPosted, Payload: `{"documentNo":"MD-9"}`,
		}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("the transaction must fail with the caller's error, got %v", err)
	}

	page, err := s.Outbox().List(ctx, store.OutboxFilter{})
	must(t, err, "list after rollback")
	if len(page.Items) != 0 {
		t.Fatalf("an event whose change was rolled back must not be published, got %v",
			ids(page.Items))
	}

	// And the committed case, so the test proves a rollback rather than a store
	// that never wrote anything.
	must(t, s.InTx(ctx, func(tx store.Store) error {
		return tx.Outbox().Append(ctx, domain.OutboxEvent{
			ID: newID(10), Topic: domain.TopicStockPosted, Payload: `{"documentNo":"MD-10"}`,
		})
	}), "commit an event")

	page, err = s.Outbox().List(ctx, store.OutboxFilter{})
	must(t, err, "list after commit")
	if len(page.Items) != 1 || page.Items[0].ID != newID(10) {
		t.Fatalf("a committed event must be there to publish, got %v", ids(page.Items))
	}
}

func sameJSON(a, b string) bool {
	var x, y any
	if json.Unmarshal([]byte(a), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

func ids(events []domain.OutboxEvent) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, e.ID)
	}
	return out
}

func find(events []domain.OutboxEvent, id string) domain.OutboxEvent {
	for _, e := range events {
		if e.ID == id {
			return e
		}
	}
	return domain.OutboxEvent{}
}

// newID makes a stable uuid for the nth event, so a failure names the same
// event every run. The columns are uuid-typed, so a readable string will not do.
func newID(n int) string {
	return "00000000-0000-4000-8000-0000000000" + string(rune('0'+n/10)) + string(rune('0'+n%10))
}
