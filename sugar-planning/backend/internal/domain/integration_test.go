package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/domain"
)

func TestWeighbridgeNetIsGrossLessTareLessRejected(t *testing.T) {
	ticket := domain.WeighbridgeTicket{
		TicketNo: "WB-1001", BusinessDate: "2026-12-05",
		GrossKg: domain.D("42150"), TareKg: domain.D("14200"),
		RejectedKg: domain.D("950"), ReasonCode: "DT-RAIN",
	}
	if err := ticket.Validate(); err != nil {
		t.Fatalf("the ticket is valid: %v", err)
	}
	// (42,150 - 14,200 - 950) / 1000
	if !ticket.NetTons().Equal(domain.D("27")) {
		t.Errorf("net = %s t, want 27", ticket.NetTons())
	}
	// The delivered figure is before anything was refused, because the grower
	// is paid on one and the mill crushes the other.
	if !ticket.DeliveredTons().Equal(domain.D("27.95")) {
		t.Errorf("delivered = %s t, want 27.95", ticket.DeliveredTons())
	}
	if !ticket.RejectedTons().Equal(domain.D("0.95")) {
		t.Errorf("rejected = %s t, want 0.95", ticket.RejectedTons())
	}
}

func TestAnImpossibleTicketIsRefusedWithEveryReason(t *testing.T) {
	bad := domain.WeighbridgeTicket{
		BusinessDate: "not-a-date",
		GrossKg:      domain.D("-1"), TareKg: domain.D("-5"), RejectedKg: domain.D("-2"),
	}
	err := bad.Validate()
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("an impossible ticket must be refused, got %v", err)
	}
	var verr *domain.ValidationError
	if !errors.As(err, &verr) || len(verr.Errors) < 4 {
		t.Errorf("every problem must be reported at once, got %+v", err)
	}

	// A tare at or above the gross means the load weighs nothing.
	empty := domain.WeighbridgeTicket{
		TicketNo: "WB-1", BusinessDate: "2026-12-05",
		GrossKg: domain.D("14000"), TareKg: domain.D("14000"),
	}
	if err := empty.Validate(); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a load weighing nothing must be refused, got %v", err)
	}

	// Refusing cane without saying why leaves a grower with no answer.
	unexplained := domain.WeighbridgeTicket{
		TicketNo: "WB-2", BusinessDate: "2026-12-05",
		GrossKg: domain.D("40000"), TareKg: domain.D("14000"), RejectedKg: domain.D("500"),
	}
	if err := unexplained.Validate(); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a rejection needs a reason, got %v", err)
	}
}

func TestRetryBackoffDoublesToACeiling(t *testing.T) {
	if got := domain.RetryDelay(0); got != 0 {
		t.Errorf("an unattempted event waits %s, want none", got)
	}
	if got := domain.RetryDelay(1); got != time.Second {
		t.Errorf("the first retry waits %s, want a second", got)
	}
	if got := domain.RetryDelay(4); got != 8*time.Second {
		t.Errorf("the fourth retry waits %s, want 8s", got)
	}
	// An ERP down for a morning must not be hammered every second.
	if got := domain.RetryDelay(50); got != time.Hour {
		t.Errorf("the backoff is capped at %s, want an hour", got)
	}
}

func TestAnExhaustedEventStaysInTheOutbox(t *testing.T) {
	event := domain.OutboxEvent{
		Topic: domain.TopicStockPosted, Attempts: domain.MaxOutboxAttempts,
		CreatedAt: time.Now(), LastError: "connection refused",
	}
	if !event.IsExhausted() {
		t.Error("an event past the attempt limit has stopped being retried")
	}
	if event.IsPublished() {
		t.Error("it is not published; it is waiting for a person")
	}
	// The point of an outbox: the message and its last error are still there.
	if event.LastError == "" {
		t.Error("the last error must survive, or nobody can diagnose it")
	}

	published := time.Now()
	done := domain.OutboxEvent{PublishedAt: &published, Attempts: 3}
	if !done.IsPublished() || done.IsExhausted() {
		t.Error("a delivered event is published and not exhausted")
	}
}

func TestALabMessageMustNameItsSampleAndCarryReadings(t *testing.T) {
	good := domain.LabResultMessage{
		SampleNo: "QS-F1-2026-00001", Complete: true,
		Readings: []domain.LabReading{{ParameterCode: "POL", Value: domain.D("99.8")}},
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("the message is valid: %v", err)
	}

	for _, bad := range []domain.LabResultMessage{
		{Readings: []domain.LabReading{{ParameterCode: "POL", Value: domain.D("99.8")}}},
		{SampleNo: "QS-1"},
		{SampleNo: "QS-1", Readings: []domain.LabReading{{Value: domain.D("1")}}},
	} {
		if err := bad.Validate(); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("%+v must be refused, got %v", bad, err)
		}
	}
}

func TestOnlyKnownTopicsArePublished(t *testing.T) {
	for _, topic := range []domain.Topic{
		domain.TopicPlanReleased, domain.TopicProductionConfirm, domain.TopicStockPosted,
		domain.TopicQualityFailed, domain.TopicCostRunCompleted,
	} {
		if !domain.ValidTopic(topic) {
			t.Errorf("%s must be a known topic", topic)
		}
	}
	if domain.ValidTopic("something.invented") {
		t.Error("an unknown topic must not be publishable; the names are a contract")
	}
}
