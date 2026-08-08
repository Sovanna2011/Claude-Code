// Package integration delivers outbound events to the systems that consume
// them.
//
// The dispatcher that decides what to deliver and when lives in the service
// layer, because retrying, backing off and giving up are business decisions
// about a factory's interfaces. What lives here is the last step: putting one
// envelope on the wire.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/kss/sugarplan/internal/domain"
)

// HTTPPublisher posts envelopes to a webhook.
//
// A webhook is the interface a mill actually gets: the ERP team exposes an
// endpoint, or an integration platform does it for them, and this system posts
// to it. A message broker would be a better fit for a large landscape, and the
// Publisher interface is where one would be added - but requiring a broker to
// be installed before a single event can flow would be the wrong default for a
// single plant with one ERP.
type HTTPPublisher struct {
	client   *http.Client
	byTopic  map[domain.Topic]string
	fallback string
	// header is the fixed authorisation header sent with every delivery, if
	// one is configured. It is never logged.
	header string
	value  string
	source string
}

// HTTPOptions configures the webhook publisher.
type HTTPOptions struct {
	// Endpoint receives every topic that has no more specific endpoint.
	Endpoint string
	// ByTopic overrides the endpoint per topic, so an ERP and an MES can be fed
	// from the same outbox without either seeing the other's traffic.
	ByTopic map[domain.Topic]string
	// AuthHeader and AuthValue are sent with every delivery, e.g.
	// "Authorization: Bearer ...". Both empty means no header.
	AuthHeader string
	AuthValue  string
	Timeout    time.Duration
	// Source names this system in the envelope.
	Source string
}

// NewHTTP builds a webhook publisher. It returns nil when nothing is
// configured, which is how the caller knows to fall back to the log publisher
// rather than starting a dispatcher that has nowhere to deliver.
func NewHTTP(opts HTTPOptions) *HTTPPublisher {
	if opts.Endpoint == "" && len(opts.ByTopic) == 0 {
		return nil
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	if opts.Source == "" {
		opts.Source = "sugarplan"
	}
	return &HTTPPublisher{
		client:   &http.Client{Timeout: opts.Timeout},
		byTopic:  opts.ByTopic,
		fallback: opts.Endpoint,
		header:   opts.AuthHeader,
		value:    opts.AuthValue,
		source:   opts.Source,
	}
}

// Publish delivers one envelope.
//
// Any 2xx is a delivery. Everything else is an error, which puts the event back
// in the outbox for the backoff to retry: a 500 because the ERP is restarting
// and a 400 because it disliked the payload are both worth retrying a few
// times, and an event that keeps being refused ends up in the exhausted list
// where somebody sees it rather than being silently dropped.
func (p *HTTPPublisher) Publish(ctx context.Context, e domain.EventEnvelope) error {
	endpoint := p.endpointFor(e.Topic)
	if endpoint == "" {
		return fmt.Errorf("no endpoint is configured for topic %s", e.Topic)
	}
	e.Source = p.source

	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// The event id is repeated in a header so a consumer can deduplicate at its
	// gateway without parsing the body.
	req.Header.Set("X-Event-Id", e.ID)
	req.Header.Set("X-Event-Topic", string(e.Topic))
	if e.CorrelationID != "" {
		req.Header.Set("X-Correlation-Id", e.CorrelationID)
	}
	if p.header != "" {
		req.Header.Set(p.header, p.value)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("post to %s: %w", redact(endpoint), err)
	}
	defer resp.Body.Close()
	// The response body is read and bounded so the connection can be reused and
	// so a chatty error page cannot fill the outbox's error column.
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))

	if resp.StatusCode/100 == 2 {
		return nil
	}
	text := strings.TrimSpace(string(snippet))
	if text == "" {
		return fmt.Errorf("%s answered %s", redact(endpoint), resp.Status)
	}
	return fmt.Errorf("%s answered %s: %s", redact(endpoint), resp.Status, text)
}

func (p *HTTPPublisher) endpointFor(topic domain.Topic) string {
	if endpoint, ok := p.byTopic[topic]; ok && endpoint != "" {
		return endpoint
	}
	return p.fallback
}

// Describe names the destinations, for the start-up log and the health page.
func (p *HTTPPublisher) Describe() string {
	parts := make([]string, 0, len(p.byTopic)+1)
	if p.fallback != "" {
		parts = append(parts, "* -> "+redact(p.fallback))
	}
	for topic, endpoint := range p.byTopic {
		parts = append(parts, string(topic)+" -> "+redact(endpoint))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// redact removes any credentials somebody put in the URL, so an endpoint can be
// logged without leaking one.
func redact(endpoint string) string {
	at := strings.Index(endpoint, "@")
	slashes := strings.Index(endpoint, "//")
	if at < 0 || slashes < 0 || at < slashes {
		return endpoint
	}
	return endpoint[:slashes+2] + "***@" + endpoint[at+1:]
}

// LogPublisher writes envelopes to the application log.
//
// This is the default when no endpoint is configured, and it is a real
// destination rather than a pretend one: the events appear in the log the
// operator already collects, which is where a plant without an ERP interface
// yet would look for them. What it is not is a silent success - every delivery
// is visible, and switching to a webhook is a matter of setting one variable.
type LogPublisher struct {
	log    *slog.Logger
	source string
}

// NewLog builds the logging publisher.
func NewLog(l *slog.Logger, source string) *LogPublisher {
	if l == nil {
		l = slog.Default()
	}
	if source == "" {
		source = "sugarplan"
	}
	return &LogPublisher{log: l, source: source}
}

// Publish records the envelope.
func (p *LogPublisher) Publish(_ context.Context, e domain.EventEnvelope) error {
	e.Source = p.source
	p.log.Info("integration event published to the log",
		"eventId", e.ID, "topic", string(e.Topic), "source", e.Source, "attempt", e.Attempt,
		"correlationId", e.CorrelationID, "payload", e.Payload)
	return nil
}

// Describe names the destination.
func (p *LogPublisher) Describe() string { return "application log" }
