package domain

import (
	"fmt"
	"time"
)

// This file is the integration model of section 22: the events this system
// publishes to the ERP and the MES, and the readings it accepts from the
// weighbridge and the laboratory.
//
// Everything outbound goes through an outbox rather than a direct call. A
// factory network drops, an ERP is restarted mid-shift, and a message posted
// during either must not be lost - nor must a confirmation be rolled back
// because the ERP was down when somebody pressed the button. Writing the event
// to the same database, in the same transaction as the change it describes, is
// what decouples the two: the change commits or it does not, and the event
// commits with it.

// ---------------------------------------------------------------------------
// Outbound
// ---------------------------------------------------------------------------

// Topic names an integration event. A consumer subscribes to topics, so the
// names are part of the contract and are not rearranged casually.
type Topic string

const (
	TopicPlanReleased       Topic = "plan.released"
	TopicProductionConfirm  Topic = "production.confirmed"
	TopicStockPosted        Topic = "stock.posted"
	TopicStockReversed      Topic = "stock.reversed"
	TopicQualityFailed      Topic = "quality.failed"
	TopicQualityReleased    Topic = "quality.released"
	TopicShipmentDispatched Topic = "shipment.dispatched"
	TopicCostRunCompleted   Topic = "cost.run.completed"
)

// ValidTopic reports whether the value is a topic this system publishes.
func ValidTopic(t Topic) bool {
	switch t {
	case TopicPlanReleased, TopicProductionConfirm, TopicStockPosted, TopicStockReversed,
		TopicQualityFailed, TopicQualityReleased, TopicShipmentDispatched, TopicCostRunCompleted:
		return true
	}
	return false
}

// OutboxEvent is one message waiting to be published.
//
// It is written in the same transaction as the change it describes. A publisher
// then reads it, delivers it and marks it published; a crash between delivering
// and marking means the message is delivered twice, which is why every payload
// carries an id a consumer can deduplicate on. At-least-once with a
// deduplication key is a promise this system can actually keep; exactly-once
// across two systems is not.
type OutboxEvent struct {
	ID            string     `json:"id"`
	Topic         Topic      `json:"topic"`
	Payload       string     `json:"payload"`
	CreatedAt     time.Time  `json:"createdAt"`
	PublishedAt   *time.Time `json:"publishedAt,omitempty"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"lastError,omitempty"`
	CorrelationID string     `json:"correlationId,omitempty"`
}

// IsPublished reports whether the event has been delivered.
func (e OutboxEvent) IsPublished() bool { return e.PublishedAt != nil }

// RetryDelay is how long to wait before attempting an event again.
//
// The backoff doubles to a ceiling of an hour. An ERP that is down for a
// morning should not be hammered every second, and an event that has failed
// thirty times is waiting for a person, not for another attempt.
func RetryDelay(attempts int) time.Duration {
	if attempts <= 0 {
		return 0
	}
	delay := time.Second
	for i := 1; i < attempts && delay < time.Hour; i++ {
		delay *= 2
	}
	if delay > time.Hour {
		delay = time.Hour
	}
	return delay
}

// MaxOutboxAttempts is where automatic retrying stops and somebody has to look.
// The event is not discarded: it stays in the outbox with its last error, which
// is the whole point of having one.
const MaxOutboxAttempts = 25

// IsExhausted reports whether an event has stopped being retried.
func (e OutboxEvent) IsExhausted() bool {
	return !e.IsPublished() && e.Attempts >= MaxOutboxAttempts
}

// DueAt is when an event may next be attempted.
func (e OutboxEvent) DueAt() time.Time {
	return e.CreatedAt.Add(RetryDelay(e.Attempts))
}

// ---------------------------------------------------------------------------
// Background jobs
// ---------------------------------------------------------------------------

// JobRun is what a scheduled job last did.
//
// It is kept because "is the dispatcher running?" is a question somebody asks
// at eight in the evening, and the honest answer needs a timestamp rather than
// an assurance.
type JobRun struct {
	Name       string     `json:"name"`
	Owner      string     `json:"owner,omitempty"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	// Status is RUNNING, OK or FAILED; empty means it has never run here.
	Status   string `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Runs     int64  `json:"runs"`
	Failures int64  `json:"failures"`
}

// Job statuses.
const (
	JobRunning = "RUNNING"
	JobOK      = "OK"
	JobFailed  = "FAILED"
)

// IsStale reports whether a job has not finished within the time it should
// have, which is what an operations screen colours red. A job that has never
// run is not stale - it may simply be switched off.
func (j JobRun) IsStale(every time.Duration, now time.Time) bool {
	if j.FinishedAt == nil || every <= 0 {
		return false
	}
	// Three intervals of grace: one missed tick is a slow batch, three is a
	// scheduler that has stopped.
	return now.Sub(*j.FinishedAt) > 3*every
}

// ---------------------------------------------------------------------------
// Inbound: weighbridge
// ---------------------------------------------------------------------------

// WeighbridgeTicket is one cane delivery weighed at the gate.
//
// The gross and tare are both carried rather than only the net, because a
// dispute about a delivery is settled by the two weights, and a system that
// kept only their difference could not settle it.
type WeighbridgeTicket struct {
	TicketNo     string       `json:"ticketNo"`
	FactoryID    string       `json:"factoryId"`
	BusinessDate BusinessDate `json:"businessDate"`
	WeighedAt    time.Time    `json:"weighedAt"`
	VehicleNo    string       `json:"vehicleNo,omitempty"`
	GrowerCode   string       `json:"growerCode,omitempty"`
	FieldCode    string       `json:"fieldCode,omitempty"`
	GrossKg      Dec          `json:"grossKg"`
	TareKg       Dec          `json:"tareKg"`
	// RejectedKg is cane refused at the gate: burnt, stale or foreign matter.
	RejectedKg Dec    `json:"rejectedKg"`
	ReasonCode string `json:"reasonCode,omitempty"`
}

// NetTons is the accepted delivery in tons.
//
//	net = (gross - tare - rejected) / 1000
func (t WeighbridgeTicket) NetTons() Dec {
	net := t.GrossKg.Sub(t.TareKg).Sub(t.RejectedKg)
	return RoundQty(ClampNonNegative(net).Div(DI(1000)))
}

// DeliveredTons is everything that arrived, before anything was refused.
func (t WeighbridgeTicket) DeliveredTons() Dec {
	return RoundQty(ClampNonNegative(t.GrossKg.Sub(t.TareKg)).Div(DI(1000)))
}

// RejectedTons is what was refused at the gate.
func (t WeighbridgeTicket) RejectedTons() Dec {
	return RoundQty(ClampNonNegative(t.RejectedKg).Div(DI(1000)))
}

// Validate checks a ticket before it is allowed to move a tonnage.
func (t WeighbridgeTicket) Validate() error {
	verr := &ValidationError{}
	if t.TicketNo == "" {
		verr.Add("ticketNo", "REQUIRED", "a ticket needs its weighbridge number")
	}
	if !t.BusinessDate.Valid() {
		verr.Add("businessDate", "INVALID_DATE", "a ticket needs a valid business date")
	}
	if t.GrossKg.LessThanOrEqual(Zero) {
		verr.Add("grossKg", "NOT_POSITIVE", "the gross weight must be more than zero")
	}
	if t.TareKg.IsNegative() {
		verr.Add("tareKg", "NEGATIVE", "the tare weight cannot be negative")
	}
	if t.TareKg.GreaterThanOrEqual(t.GrossKg) {
		verr.Add("tareKg", "OUT_OF_RANGE",
			"the tare weight is not less than the gross weight, so the load weighs nothing")
	}
	if t.RejectedKg.IsNegative() {
		verr.Add("rejectedKg", "NEGATIVE", "the rejected weight cannot be negative")
	}
	if t.RejectedKg.GreaterThan(t.GrossKg.Sub(t.TareKg)) {
		verr.Add("rejectedKg", "OUT_OF_RANGE",
			"more was rejected than was delivered")
	}
	if t.RejectedKg.GreaterThan(Zero) && t.ReasonCode == "" {
		verr.Add("reasonCode", "REQUIRED",
			"cane refused at the gate needs a reason; it is what the grower is shown")
	}
	return verr.OrNil()
}

// ---------------------------------------------------------------------------
// Inbound: laboratory information system
// ---------------------------------------------------------------------------

// LabReading is one measurement an instrument produced.
type LabReading struct {
	ParameterCode string `json:"parameterCode"`
	Value         Dec    `json:"value"`
	UOM           string `json:"uom,omitempty"`
}

// LabResultMessage is a sheet of readings from the laboratory system.
//
// It names the sample by number rather than by id: the instrument knows the
// number printed on the bottle, and asking a laboratory technician to key a
// uuid would guarantee the interface went unused.
type LabResultMessage struct {
	SampleNo     string       `json:"sampleNo"`
	FactoryID    string       `json:"factoryId,omitempty"`
	BusinessDate BusinessDate `json:"businessDate,omitempty"`
	Instrument   string       `json:"instrument,omitempty"`
	MeasuredAt   time.Time    `json:"measuredAt,omitempty"`
	Readings     []LabReading `json:"readings"`
	// Complete closes the sample and produces a verdict. An instrument that
	// reports one parameter at a time leaves it false until the last message.
	Complete bool `json:"complete"`
}

// Validate checks a laboratory message before it is allowed to judge anything.
func (m LabResultMessage) Validate() error {
	verr := &ValidationError{}
	if m.SampleNo == "" {
		verr.Add("sampleNo", "REQUIRED",
			"the message must name the sample, by the number printed on the bottle")
	}
	if len(m.Readings) == 0 {
		verr.Add("readings", "EMPTY", "a result message with no readings judges nothing")
	}
	for i, r := range m.Readings {
		if r.ParameterCode == "" {
			verr.AddRow(i, "parameterCode", "REQUIRED", "a reading needs a parameter code")
		}
	}
	return verr.OrNil()
}

// ---------------------------------------------------------------------------
// Delivery
// ---------------------------------------------------------------------------

// EventEnvelope is what a consumer receives. It is deliberately flat and
// self-describing: a consumer should not need this system's schema to route a
// message, and the id is what it deduplicates on.
type EventEnvelope struct {
	ID            string    `json:"id"`
	Topic         Topic     `json:"topic"`
	OccurredAt    time.Time `json:"occurredAt"`
	CorrelationID string    `json:"correlationId,omitempty"`
	Source        string    `json:"source"`
	// Payload is the event body as it was written, already JSON.
	Payload string `json:"payload"`
	// Attempt is which delivery this is, from one. A consumer seeing an attempt
	// above one knows a duplicate is possible.
	Attempt int `json:"attempt"`
}

// Describe renders an envelope for a log line.
func (e EventEnvelope) Describe() string {
	return fmt.Sprintf("%s %s (attempt %d)", e.Topic, e.ID, e.Attempt)
}
