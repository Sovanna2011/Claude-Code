package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Integration is the interface layer: the dispatcher that publishes what the
// outbox holds, and the two inbound adapters a sugar mill actually has - the
// weighbridge at the gate and the laboratory system.
//
// The inbound adapters deliberately go through the ordinary services rather
// than writing to the store themselves. A ticket that arrives from the
// weighbridge is subject to the same validation, the same authorisation and the
// same audit trail as a tonnage somebody types in, because an interface that
// can bypass a rule is a rule that is not enforced.
type Integration struct {
	store     store.Store
	execution *Execution
	planning  *Planning
	publisher Publisher
	now       func() time.Time
}

// Publisher delivers one envelope to whatever is on the other side. The
// implementations live in internal/integration; the interface is here because
// the dispatcher is what depends on it.
type Publisher interface {
	Publish(ctx context.Context, e domain.EventEnvelope) error
	// Describe names the destination for the start-up log.
	Describe() string
}

// NewIntegration builds the service.
func NewIntegration(s store.Store, execution *Execution, planning *Planning,
	publisher Publisher, now func() time.Time,
) *Integration {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Integration{store: s, execution: execution, planning: planning,
		publisher: publisher, now: now}
}

// ---------------------------------------------------------------------------
// Outbound: the dispatcher
// ---------------------------------------------------------------------------

// DispatchResult reports what one pass of the dispatcher did.
type DispatchResult struct {
	Considered int `json:"considered"`
	Published  int `json:"published"`
	Failed     int `json:"failed"`
	// Exhausted counts the events that reached the retry ceiling on this pass
	// and now need a person. It is what an alert should be raised on.
	Exhausted int      `json:"exhausted"`
	Errors    []string `json:"errors,omitempty"`
}

// DefaultDispatchBatch is how many events one pass takes. Small enough that a
// slow consumer cannot hold a transaction open for a minute, large enough that
// a backlog drains in reasonable time.
const DefaultDispatchBatch = 50

// Dispatch publishes the events whose backoff has elapsed.
//
// It is called by the scheduler on a timer and by an operator on demand. Each
// event is marked inside its own transaction, immediately after its delivery,
// so a crash in the middle of a batch loses at most the marking of the event in
// flight - which is a duplicate delivery, which is what the envelope id is for.
func (i *Integration) Dispatch(ctx context.Context, limit int) (DispatchResult, error) {
	if i.publisher == nil {
		return DispatchResult{}, fmt.Errorf(
			"%w: no publisher is configured, so nothing can be delivered", domain.ErrValidation)
	}
	if limit <= 0 {
		limit = DefaultDispatchBatch
	}

	now := i.now()
	due, err := i.store.Outbox().Due(ctx, now, limit)
	if err != nil {
		return DispatchResult{}, err
	}

	result := DispatchResult{Considered: len(due)}
	for _, event := range due {
		if err := i.deliver(ctx, event); err != nil {
			result.Failed++
			if len(result.Errors) < 10 {
				result.Errors = append(result.Errors,
					fmt.Sprintf("%s %s: %s", event.Topic, event.ID, err))
			}
			// Attempts is the count before this one, so reaching the ceiling
			// with this attempt is attempts+1.
			if event.Attempts+1 >= domain.MaxOutboxAttempts {
				result.Exhausted++
			}
			continue
		}
		result.Published++
	}
	return result, nil
}

// deliver publishes one event and records the outcome.
func (i *Integration) deliver(ctx context.Context, event domain.OutboxEvent) error {
	envelope := domain.EventEnvelope{
		ID: event.ID, Topic: event.Topic, OccurredAt: event.CreatedAt,
		CorrelationID: event.CorrelationID, Payload: event.Payload,
		Attempt: event.Attempts + 1,
	}

	publishErr := i.publisher.Publish(ctx, envelope)
	// The marking uses a background context so that a cancelled request cannot
	// leave a delivered event looking undelivered - which would republish it on
	// the next pass for no reason.
	marking, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	if publishErr != nil {
		if err := i.store.Outbox().MarkFailed(marking, event.ID, i.now(), publishErr.Error()); err != nil {
			return fmt.Errorf("%v (and recording the failure failed: %w)", publishErr, err)
		}
		return publishErr
	}
	return i.store.Outbox().MarkPublished(marking, event.ID, i.now())
}

// ListEvents returns the outbox for the operations screen.
func (i *Integration) ListEvents(ctx context.Context, f store.OutboxFilter) (store.Page[domain.OutboxEvent], error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermIntegrationRead); err != nil {
		return store.Page[domain.OutboxEvent]{}, err
	}
	if f.Topic != "" && !domain.ValidTopic(domain.Topic(f.Topic)) {
		return store.Page[domain.OutboxEvent]{}, fmt.Errorf(
			"%w: %q is not a topic this system publishes", domain.ErrValidation, f.Topic)
	}
	return i.store.Outbox().List(ctx, f)
}

// RetryEvent delivers one event now, whatever its backoff says.
//
// This is what an operator does after fixing the far end: an event that has
// given up after twenty-five attempts is not lost, and the fix should not have
// to wait for a timer. Success publishes it; failure records another attempt
// and the new error, which is the more useful outcome when the far end is still
// broken.
func (i *Integration) RetryEvent(ctx context.Context, id string) (domain.OutboxEvent, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermIntegrationRead); err != nil {
		return domain.OutboxEvent{}, err
	}
	if i.publisher == nil {
		return domain.OutboxEvent{}, fmt.Errorf(
			"%w: no publisher is configured, so nothing can be delivered", domain.ErrValidation)
	}

	event, err := i.store.Outbox().Get(ctx, id)
	if err != nil {
		return domain.OutboxEvent{}, err
	}
	if event.IsPublished() {
		return domain.OutboxEvent{}, fmt.Errorf(
			"%w: event %s was already published at %s",
			domain.ErrValidation, event.ID, event.PublishedAt.Format(time.RFC3339))
	}
	if err := i.deliver(ctx, event); err != nil {
		// The attempt was recorded, so the caller is shown the current state
		// together with why it still is not delivered.
		latest, getErr := i.store.Outbox().Get(ctx, id)
		if getErr != nil {
			return domain.OutboxEvent{}, err
		}
		return latest, fmt.Errorf("%w: %s", domain.ErrUpstream, err)
	}
	return i.store.Outbox().Get(ctx, id)
}

// ListJobs reports what each background job last did.
//
// "Is the dispatcher running?" is a question somebody asks at eight in the
// evening, and the honest answer needs a timestamp rather than an assurance.
func (i *Integration) ListJobs(ctx context.Context) ([]domain.JobRun, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermIntegrationRead); err != nil {
		return nil, err
	}
	return i.store.Jobs().List(ctx)
}

// ---------------------------------------------------------------------------
// Inbound: the weighbridge
// ---------------------------------------------------------------------------

// WeighbridgeRequest is a batch of tickets from the gate. Batching matters: the
// terminal buffers when the network drops and sends a shift's worth at once.
type WeighbridgeRequest struct {
	Tickets []domain.WeighbridgeTicket `json:"tickets"`
	// Partial accepts the sound tickets and reports the rest. A terminal
	// replaying a buffer wants this; a single ticket keyed by hand does not.
	Partial bool `json:"partial,omitempty"`
}

// WeighbridgeResult reports what the tickets did to the actuals.
type WeighbridgeResult struct {
	Accepted int        `json:"accepted"`
	Rejected int        `json:"rejected"`
	Issues   []RowIssue `json:"issues,omitempty"`
	// Days lists the actual cane rows that were written, so the terminal can
	// show the gate keeper the running total for the shift.
	Days []WeighbridgeDay `json:"days"`
}

// WeighbridgeDay is one day's accumulated deliveries at one factory.
type WeighbridgeDay struct {
	FactoryID     string              `json:"factoryId"`
	BusinessDate  domain.BusinessDate `json:"businessDate"`
	VersionID     string              `json:"versionId"`
	Tickets       int                 `json:"tickets"`
	CaneDelivered domain.Dec          `json:"caneDelivered"`
	CaneAccepted  domain.Dec          `json:"caneAccepted"`
	CaneRejected  domain.Dec          `json:"caneRejected"`
}

// IngestWeighbridge records gate tickets as actual cane.
//
// Tickets are added to the day rather than replacing it: the gate weighs a
// lorry at a time, and each message carries what has arrived since the last
// one. A day already holding a hand-entered figure therefore grows by what the
// gate reports, which is the behaviour a supervisor expects and the reason the
// ticket number is carried into the note - the same ticket sent twice can be
// found and, if it was double counted, corrected.
func (i *Integration) IngestWeighbridge(ctx context.Context, req WeighbridgeRequest) (WeighbridgeResult, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermIntegrationWrite); err != nil {
		return WeighbridgeResult{}, err
	}
	if len(req.Tickets) == 0 {
		return WeighbridgeResult{}, fmt.Errorf(
			"%w: the message carries no tickets", domain.ErrValidation)
	}

	result := WeighbridgeResult{Days: []WeighbridgeDay{}}
	// Tickets are grouped by factory and day, because that is the grain the
	// actuals are held at. Fifty lorries on one day become one row, not fifty.
	type key struct {
		factory string
		date    domain.BusinessDate
	}
	groups := map[key]*WeighbridgeDay{}
	var order []key

	for row, ticket := range req.Tickets {
		if ticket.FactoryID == "" {
			result.Issues = append(result.Issues, RowIssue{Row: row, Field: "factoryId",
				Code: "REQUIRED", Message: "a ticket must say which factory weighed it"})
			continue
		}
		if err := ticket.Validate(); err != nil {
			var verr *domain.ValidationError
			if errors.As(err, &verr) {
				for _, e := range verr.Errors {
					result.Issues = append(result.Issues, RowIssue{Row: row, Field: e.Field,
						Code: e.Code, Message: e.Message})
				}
				continue
			}
			return WeighbridgeResult{}, err
		}
		if err := caller.RequireFactory(ticket.FactoryID); err != nil {
			result.Issues = append(result.Issues, RowIssue{Row: row, Field: "factoryId",
				Code: "FORBIDDEN", Message: err.Error()})
			continue
		}

		k := key{ticket.FactoryID, ticket.BusinessDate}
		day, ok := groups[k]
		if !ok {
			day = &WeighbridgeDay{FactoryID: ticket.FactoryID, BusinessDate: ticket.BusinessDate}
			groups[k], order = day, append(order, k)
		}
		day.Tickets++
		day.CaneDelivered = domain.RoundQty(day.CaneDelivered.Add(ticket.DeliveredTons()))
		day.CaneAccepted = domain.RoundQty(day.CaneAccepted.Add(ticket.NetTons()))
		day.CaneRejected = domain.RoundQty(day.CaneRejected.Add(ticket.RejectedTons()))
		result.Accepted++
	}

	result.Rejected = len(req.Tickets) - result.Accepted
	if result.Rejected > 0 && !req.Partial {
		// All or nothing is the default, so a terminal that sent a bad batch
		// resends the whole batch rather than working out what got through.
		result.Accepted, result.Days = 0, []WeighbridgeDay{}
		return result, fmt.Errorf("%w: %d of %d tickets were refused",
			domain.ErrValidation, result.Rejected, len(req.Tickets))
	}
	if result.Accepted == 0 {
		return result, nil
	}

	for _, k := range order {
		day := groups[k]
		version, err := i.actualsVersion(ctx, day.FactoryID, day.BusinessDate)
		if err != nil {
			return WeighbridgeResult{}, err
		}
		day.VersionID = version.ID
		if err := i.addCane(ctx, version, *day); err != nil {
			return WeighbridgeResult{}, err
		}
		result.Days = append(result.Days, *day)
	}
	return result, nil
}

// addCane adds a day's deliveries to the actual cane row, reading what is there
// and writing the sum inside one transaction so two terminals sending at once
// cannot lose a lorry between them.
func (i *Integration) addCane(ctx context.Context, version domain.PlanVersion, day WeighbridgeDay) error {
	caller := auth.FromContext(ctx)
	return i.store.InTx(ctx, func(tx store.Store) error {
		existing, err := tx.Planning().ListCane(ctx, store.PlanFilter{
			VersionIDs: []string{version.ID}, FactoryID: day.FactoryID,
			From: day.BusinessDate, To: day.BusinessDate, Series: domain.SeriesActual,
		})
		if err != nil {
			return err
		}

		row := domain.DailyCanePlan{
			VersionID: version.ID, FactoryID: day.FactoryID, BusinessDate: day.BusinessDate,
			Series: domain.SeriesActual,
		}
		for _, r := range existing {
			if r.ShiftID == "" {
				row = r
				break
			}
		}
		row.CaneDelivered = domain.RoundQty(row.CaneDelivered.Add(day.CaneDelivered))
		row.CaneAccepted = domain.RoundQty(row.CaneAccepted.Add(day.CaneAccepted))
		row.CaneRejected = domain.RoundQty(row.CaneRejected.Add(day.CaneRejected))
		// Accepted cane is what the gate lets in, not what the mill crushed.
		// The crushed figure stays where it is: it comes from the mill, and a
		// gate reading is not evidence about it.
		if row.CaneAvailable.LessThan(row.CaneAccepted) {
			row.CaneAvailable = row.CaneAccepted
		}
		row.Note = fmt.Sprintf("%d weighbridge tickets", day.Tickets)

		if _, err := tx.Planning().UpsertCane(ctx, []domain.DailyCanePlan{row}, caller.Username); err != nil {
			return err
		}
		return writeAudit(ctx, tx, i.now, auditEntry{
			action: "WEIGHBRIDGE", entity: "daily_cane_plan", entityID: row.ID,
			after: day, reason: fmt.Sprintf("%d tickets, %s t accepted at the gate",
				day.Tickets, day.CaneAccepted),
		})
	})
}

// actualsVersion finds the container the actuals of a factory's season are held
// in. An interface cannot be asked which plan version to write to - the
// weighbridge does not know what a plan version is - so the season covering the
// date decides, and a date outside every season is refused rather than guessed
// at.
func (i *Integration) actualsVersion(ctx context.Context, factoryID string,
	on domain.BusinessDate,
) (domain.PlanVersion, error) {

	seasons, err := i.store.Planning().ListSeasons(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return domain.PlanVersion{}, err
	}
	var season domain.Season
	for _, s := range seasons.Items {
		if s.FactoryID != factoryID || on < s.StartDate || on > s.EndDate {
			continue
		}
		if season.ID == "" || s.StartDate > season.StartDate {
			season = s
		}
	}
	if season.ID == "" {
		return domain.PlanVersion{}, fmt.Errorf(
			"%w: %s is not inside any season of this factory, so there is nothing to record it against",
			domain.ErrValidation, on)
	}

	versions, err := i.store.Planning().ListVersions(ctx, season.ID, store.ListOptions{Top: 1000})
	if err != nil {
		return domain.PlanVersion{}, err
	}
	for _, v := range versions.Items {
		if v.PlanType == domain.PlanTypeActual {
			return v, nil
		}
	}
	return domain.PlanVersion{}, fmt.Errorf(
		"%w: season %s has no actuals container to record against",
		domain.ErrValidation, season.Code)
}

// ---------------------------------------------------------------------------
// Inbound: the laboratory system
// ---------------------------------------------------------------------------

// IngestLabResults records a sheet of instrument readings against the sample
// whose number is printed on the bottle.
//
// It resolves the sample and the parameter codes and then calls the ordinary
// quality service, so an instrument reading is judged against the same
// specifications, produces the same verdict and places the same hold as a
// reading a technician types in.
func (i *Integration) IngestLabResults(ctx context.Context, msg domain.LabResultMessage) (ResultsOutcome, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermIntegrationWrite); err != nil {
		return ResultsOutcome{}, err
	}
	if err := msg.Validate(); err != nil {
		return ResultsOutcome{}, err
	}

	sample, err := i.findSample(ctx, msg)
	if err != nil {
		return ResultsOutcome{}, err
	}

	parameters, err := i.store.Execution().ListParameters(ctx)
	if err != nil {
		return ResultsOutcome{}, err
	}
	byCode := map[string]domain.QualityParameter{}
	for _, p := range parameters {
		byCode[strings.ToUpper(p.Code)] = p
	}

	verr := &domain.ValidationError{}
	results := make([]ResultInput, 0, len(msg.Readings))
	for row, reading := range msg.Readings {
		parameter, ok := byCode[strings.ToUpper(reading.ParameterCode)]
		if !ok {
			verr.AddRow(row, "parameterCode", "NOT_FOUND", fmt.Sprintf(
				"%q is not a quality parameter in this system", reading.ParameterCode))
			continue
		}
		if !parameter.Active {
			verr.AddRow(row, "parameterCode", "INACTIVE", fmt.Sprintf(
				"parameter %s is not in use", parameter.Code))
			continue
		}
		results = append(results, ResultInput{
			ParameterID: parameter.ID, Value: reading.Value, UOM: reading.UOM,
			Comment: instrumentNote(msg),
		})
	}
	if err := verr.OrNil(); err != nil {
		return ResultsOutcome{}, err
	}

	return i.execution.RecordResults(ctx, sample.ID, ResultsRequest{
		Results: results, Complete: msg.Complete,
	})
}

func instrumentNote(msg domain.LabResultMessage) string {
	parts := make([]string, 0, 2)
	if msg.Instrument != "" {
		parts = append(parts, msg.Instrument)
	}
	if !msg.MeasuredAt.IsZero() {
		parts = append(parts, msg.MeasuredAt.UTC().Format(time.RFC3339))
	}
	if len(parts) == 0 {
		return "laboratory interface"
	}
	return "laboratory interface: " + strings.Join(parts, " at ")
}

// findSample resolves the number on the bottle to a sample.
//
// A number that matches nothing is refused rather than creating a sample: a
// reading with no sample is a reading of nothing in particular, and inventing
// one would put a verdict against material nobody sampled.
func (i *Integration) findSample(ctx context.Context, msg domain.LabResultMessage) (domain.QualitySample, error) {
	page, err := i.store.Execution().ListSamples(ctx, store.ExecutionFilter{
		Number: msg.SampleNo, FactoryID: msg.FactoryID, Top: 10,
	})
	if err != nil {
		return domain.QualitySample{}, err
	}
	matches := page.Items
	if len(matches) == 0 {
		return domain.QualitySample{}, fmt.Errorf(
			"%w: no sample is numbered %s", domain.ErrNotFound, msg.SampleNo)
	}
	if len(matches) > 1 {
		// Sample numbers are unique per factory, so this only happens when a
		// message from a shared laboratory omits the factory. Naming the
		// candidates is more use than picking one.
		sort.Slice(matches, func(a, b int) bool { return matches[a].FactoryID < matches[b].FactoryID })
		factories := make([]string, 0, len(matches))
		for _, m := range matches {
			factories = append(factories, m.FactoryID)
		}
		return domain.QualitySample{}, fmt.Errorf(
			"%w: sample %s exists at more than one factory (%s); name the factory in the message",
			domain.ErrValidation, msg.SampleNo, strings.Join(factories, ", "))
	}
	return matches[0], nil
}
