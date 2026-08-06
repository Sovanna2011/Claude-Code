package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// These tests cover the interfaces: what leaves this system when something
// happens, and what a weighbridge terminal or a laboratory instrument can do to
// it. The reference scenario is seeded, so the tonnages are the real ones.

// recorder is a publisher that remembers what it was asked to deliver and can
// be told to fail, which is the only way to exercise a backoff without waiting
// for one.
type recorder struct {
	mu        sync.Mutex
	delivered []domain.EventEnvelope
	failWith  error
}

func (r *recorder) Publish(_ context.Context, e domain.EventEnvelope) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failWith != nil {
		return r.failWith
	}
	r.delivered = append(r.delivered, e)
	return nil
}

func (r *recorder) Describe() string { return "test recorder" }

func (r *recorder) topics() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.delivered))
	for _, e := range r.delivered {
		out = append(out, string(e.Topic))
	}
	return out
}

func integrationHarness(t *testing.T) (*harness, *service.Execution, *service.Integration, *recorder) {
	t.Helper()
	h, exec := execHarness(t)
	pub := &recorder{}
	return h, exec, service.NewIntegration(h.store, exec, h.planning, pub, fixedClock()), pub
}

// gate is the machine account the weighbridge and the laboratory sign in as.
func (h *harness) gate() context.Context { return h.as(auth.RoleIntegration) }

// ---------------------------------------------------------------------------
// Outbound
// ---------------------------------------------------------------------------

func TestPostingStockWritesAnEventAndTheDispatcherDeliversIt(t *testing.T) {
	h, exec, integ, pub := integrationHarness(t)
	ctx := h.keeper()

	posted, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: h.seeded.Warehouses["FG-WH1"], ProductID: h.seeded.Products["REF"],
			Quantity: domain.D("500"),
		}},
	})
	if err != nil {
		t.Fatalf("post receipt: %v", err)
	}

	admin := h.as(auth.RoleSystemAdmin)
	page, err := integ.ListEvents(admin, store.OutboxFilter{Unpublished: true})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Topic != domain.TopicStockPosted {
		t.Fatalf("posting stock must write one stock.posted event, got %v", page.Items)
	}

	var event service.StockEvent
	if err := json.Unmarshal([]byte(page.Items[0].Payload), &event); err != nil {
		t.Fatalf("the payload must be readable JSON: %v", err)
	}
	if event.DocumentNo != posted.DocumentNo {
		t.Errorf("the event must name the document: %s, want %s", event.DocumentNo, posted.DocumentNo)
	}
	if len(event.Lines) != 1 || !event.Lines[0].Quantity.Equal(domain.D("500")) {
		t.Errorf("the event must carry the movement, got %+v", event.Lines)
	}

	result, err := integ.Dispatch(admin, 0)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if result.Published != 1 || result.Failed != 0 {
		t.Errorf("one event should have been published, got %+v", result)
	}
	if got := pub.topics(); len(got) != 1 || got[0] != "stock.posted" {
		t.Errorf("the publisher should have seen the event, got %v", got)
	}

	// A second pass has nothing to do: a published event is finished with.
	result, err = integ.Dispatch(admin, 0)
	if err != nil {
		t.Fatalf("dispatch again: %v", err)
	}
	if result.Considered != 0 {
		t.Errorf("a published event must not be delivered twice, got %+v", result)
	}
}

func TestAFailedPostingPublishesNothing(t *testing.T) {
	h, exec, integ, _ := integrationHarness(t)
	ctx := h.keeper()

	// Issuing from an empty store fails, and the event written alongside it must
	// fail with it. This is the whole reason for an outbox rather than a call.
	_, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocIssue, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: h.seeded.Warehouses["FG-WH1"], ProductID: h.seeded.Products["REF"],
			Quantity: domain.D("500"),
		}},
	})
	if err == nil {
		t.Fatal("issuing stock that is not there must be refused")
	}

	page, err := integ.ListEvents(h.as(auth.RoleSystemAdmin), store.OutboxFilter{})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("a rolled back posting must publish nothing, got %v", page.Items)
	}
}

func TestADeliveryThatFailsIsRetriedWithItsErrorRecorded(t *testing.T) {
	h, exec, integ, pub := integrationHarness(t)
	ctx := h.keeper()
	admin := h.as(auth.RoleSystemAdmin)

	if _, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: h.seeded.Warehouses["FG-WH1"], ProductID: h.seeded.Products["REF"],
			Quantity: domain.D("500"),
		}},
	}); err != nil {
		t.Fatalf("post receipt: %v", err)
	}

	pub.failWith = errors.New("the ERP answered 503")
	result, err := integ.Dispatch(admin, 0)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if result.Failed != 1 || result.Published != 0 {
		t.Fatalf("the delivery failed, so the pass should report it: %+v", result)
	}
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0], "503") {
		t.Errorf("the pass must quote what went wrong, got %v", result.Errors)
	}

	page, err := integ.ListEvents(admin, store.OutboxFilter{Unpublished: true})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("the event must still be waiting, got %v", page.Items)
	}
	if page.Items[0].Attempts != 1 || !strings.Contains(page.Items[0].LastError, "503") {
		t.Errorf("the failure must be recorded on the event, got %+v", page.Items[0])
	}

	// The backoff holds it back from the very next pass.
	result, err = integ.Dispatch(admin, 0)
	if err != nil {
		t.Fatalf("dispatch again: %v", err)
	}
	if result.Considered != 0 {
		t.Errorf("an event that just failed is not due again immediately, got %+v", result)
	}

	// Retrying by hand ignores the backoff, which is what an operator does once
	// the far end is fixed.
	pub.failWith = nil
	event, err := integ.RetryEvent(admin, page.Items[0].ID)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !event.IsPublished() {
		t.Errorf("a successful retry must publish the event, got %+v", event)
	}
}

func TestARetryThatStillFailsSaysSo(t *testing.T) {
	h, exec, integ, pub := integrationHarness(t)
	admin := h.as(auth.RoleSystemAdmin)

	if _, err := exec.Post(h.keeper(), service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: h.seeded.Warehouses["FG-WH1"], ProductID: h.seeded.Products["REF"],
			Quantity: domain.D("10"),
		}},
	}); err != nil {
		t.Fatalf("post receipt: %v", err)
	}
	page, err := integ.ListEvents(admin, store.OutboxFilter{Unpublished: true})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}

	pub.failWith = errors.New("connection refused")
	event, err := integ.RetryEvent(admin, page.Items[0].ID)
	if !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("a far end that is still broken is an upstream failure, got %v", err)
	}
	if event.Attempts != 1 || event.IsPublished() {
		t.Errorf("the failed attempt must be recorded and the event left waiting, got %+v", event)
	}
}

func TestReleasingAPlanAndFailingASamplePublish(t *testing.T) {
	h, exec, integ, _ := integrationHarness(t)
	admin := h.as(auth.RoleSystemAdmin)

	// A failed sample.
	lab := h.as(auth.RoleQualityUser)
	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: h.seeded.Products["REF"], FactoryID: h.seeded.FactoryID,
		BusinessDate: "2026-12-05",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	parameters, err := exec.ListParameters(lab)
	if err != nil {
		t.Fatalf("list parameters: %v", err)
	}
	pol := parameterByCode(t, parameters, "POL")
	outcome, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results:  []service.ResultInput{{ParameterID: pol, Value: domain.D("90")}},
		Complete: true,
	})
	if err != nil {
		t.Fatalf("record results: %v", err)
	}
	if outcome.Verdict != domain.QualityFail {
		t.Fatalf("a pol of 90 must fail the refined specification, got %s", outcome.Verdict)
	}

	page, err := integ.ListEvents(admin, store.OutboxFilter{Topic: string(domain.TopicQualityFailed)})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("a failed sample must publish, got %v", page.Items)
	}
	var quality service.QualityEvent
	if err := json.Unmarshal([]byte(page.Items[0].Payload), &quality); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if quality.SampleNo != sample.SampleNo || quality.Verdict != domain.QualityFail {
		t.Errorf("the event must name the sample and its verdict, got %+v", quality)
	}
}

func TestAnInterimSheetPublishesNothing(t *testing.T) {
	h, exec, integ, _ := integrationHarness(t)
	lab := h.as(auth.RoleQualityUser)

	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: h.seeded.Products["REF"], FactoryID: h.seeded.FactoryID,
		BusinessDate: "2026-12-05",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	parameters, err := exec.ListParameters(lab)
	if err != nil {
		t.Fatalf("list parameters: %v", err)
	}
	if _, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results: []service.ResultInput{{
			ParameterID: parameterByCode(t, parameters, "POL"), Value: domain.D("90"),
		}},
	}); err != nil {
		t.Fatalf("record interim results: %v", err)
	}

	page, err := integ.ListEvents(h.as(auth.RoleSystemAdmin),
		store.OutboxFilter{Topic: string(domain.TopicQualityFailed)})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("an unfinished sheet is not a verdict and must not publish, got %v", page.Items)
	}
}

func TestOnlyAnOperatorMaySeeTheOutbox(t *testing.T) {
	h, _, integ, _ := integrationHarness(t)
	if _, err := integ.ListEvents(h.keeper(), store.OutboxFilter{}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a warehouse operator must not read the outbox, got %v", err)
	}
	if _, err := integ.ListEvents(h.as(auth.RoleSystemAdmin),
		store.OutboxFilter{Topic: "nonsense"}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("an unknown topic must be refused, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Inbound: the weighbridge
// ---------------------------------------------------------------------------

func ticket(no string, factory string, gross, tare, rejected string) domain.WeighbridgeTicket {
	t := domain.WeighbridgeTicket{
		TicketNo: no, FactoryID: factory, BusinessDate: "2026-12-05",
		GrossKg: domain.D(gross), TareKg: domain.D(tare), RejectedKg: domain.D(rejected),
	}
	if t.RejectedKg.GreaterThan(domain.Zero) {
		t.ReasonCode = "BURNT"
	}
	return t
}

func TestWeighbridgeTicketsBecomeActualCane(t *testing.T) {
	h, _, integ, _ := integrationHarness(t)
	ctx := h.gate()

	result, err := integ.IngestWeighbridge(ctx, service.WeighbridgeRequest{
		Tickets: []domain.WeighbridgeTicket{
			ticket("T-1", h.seeded.FactoryID, "42000", "14000", "0"),
			ticket("T-2", h.seeded.FactoryID, "40000", "14000", "1000"),
		},
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if result.Accepted != 2 || result.Rejected != 0 {
		t.Fatalf("both tickets are sound: %+v", result)
	}
	if len(result.Days) != 1 {
		t.Fatalf("two tickets on one day are one row, got %d", len(result.Days))
	}

	day := result.Days[0]
	// 28.000 + 26.000 delivered, of which 1.000 was refused at the gate.
	if !day.CaneDelivered.Equal(domain.D("54")) {
		t.Errorf("delivered = %s, want 54.000", day.CaneDelivered)
	}
	if !day.CaneAccepted.Equal(domain.D("53")) {
		t.Errorf("accepted = %s, want 53.000", day.CaneAccepted)
	}
	if !day.CaneRejected.Equal(domain.D("1")) {
		t.Errorf("rejected = %s, want 1.000", day.CaneRejected)
	}

	// The figures are on the actuals row, where the dashboard reads them.
	rows, err := h.store.Planning().ListCane(context.Background(), store.PlanFilter{
		VersionIDs: []string{day.VersionID}, From: "2026-12-05", To: "2026-12-05",
		Series: domain.SeriesActual,
	})
	if err != nil {
		t.Fatalf("list cane: %v", err)
	}
	if len(rows) != 1 || !rows[0].CaneAccepted.Equal(domain.D("53")) {
		t.Fatalf("the actual row must hold the accepted tonnage, got %v", rows)
	}
	// The mill reports what it crushed; the gate does not.
	if !rows[0].CaneCrushed.IsZero() {
		t.Errorf("a gate reading must not invent a crushed figure, got %s", rows[0].CaneCrushed)
	}

	// A second batch adds to the day rather than replacing it, because the gate
	// sends what has arrived since the last message.
	if _, err := integ.IngestWeighbridge(ctx, service.WeighbridgeRequest{
		Tickets: []domain.WeighbridgeTicket{ticket("T-3", h.seeded.FactoryID, "30000", "10000", "0")},
	}); err != nil {
		t.Fatalf("ingest second batch: %v", err)
	}
	rows, err = h.store.Planning().ListCane(context.Background(), store.PlanFilter{
		VersionIDs: []string{day.VersionID}, From: "2026-12-05", To: "2026-12-05",
		Series: domain.SeriesActual,
	})
	if err != nil {
		t.Fatalf("list cane: %v", err)
	}
	if len(rows) != 1 || !rows[0].CaneAccepted.Equal(domain.D("73")) {
		t.Fatalf("the second batch must add to the day, got %v", rows)
	}
}

func TestABadTicketRefusesTheWholeBatchUnlessAskedOtherwise(t *testing.T) {
	h, _, integ, _ := integrationHarness(t)
	ctx := h.gate()

	// A tare heavier than the gross means the load weighs nothing.
	batch := []domain.WeighbridgeTicket{
		ticket("T-1", h.seeded.FactoryID, "42000", "14000", "0"),
		ticket("T-2", h.seeded.FactoryID, "12000", "14000", "0"),
	}

	result, err := integ.IngestWeighbridge(ctx, service.WeighbridgeRequest{Tickets: batch})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a batch with a bad ticket is refused whole, got %v", err)
	}
	if result.Accepted != 0 {
		t.Errorf("nothing may be recorded from a refused batch, got %+v", result)
	}
	if len(result.Issues) == 0 || result.Issues[0].Row != 1 {
		t.Errorf("the issue must name the row that was wrong, got %+v", result.Issues)
	}

	// A terminal replaying a buffer asks for the sound tickets to be taken.
	result, err = integ.IngestWeighbridge(ctx, service.WeighbridgeRequest{
		Tickets: batch, Partial: true,
	})
	if err != nil {
		t.Fatalf("partial ingest: %v", err)
	}
	if result.Accepted != 1 || result.Rejected != 1 {
		t.Errorf("one ticket in, one reported: %+v", result)
	}
	if len(result.Days) != 1 || !result.Days[0].CaneAccepted.Equal(domain.D("28")) {
		t.Errorf("only the sound ticket may count, got %+v", result.Days)
	}
}

func TestRejectedCaneNeedsAReasonAndADateNeedsASeason(t *testing.T) {
	h, _, integ, _ := integrationHarness(t)
	ctx := h.gate()

	noReason := ticket("T-1", h.seeded.FactoryID, "42000", "14000", "2000")
	noReason.ReasonCode = ""
	result, err := integ.IngestWeighbridge(ctx, service.WeighbridgeRequest{
		Tickets: []domain.WeighbridgeTicket{noReason},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("cane refused with no reason must be refused, got %v", err)
	}
	if len(result.Issues) == 0 || result.Issues[0].Field != "reasonCode" {
		t.Errorf("the issue must point at the missing reason, got %+v", result.Issues)
	}

	outside := ticket("T-2", h.seeded.FactoryID, "42000", "14000", "0")
	outside.BusinessDate = "2030-06-01"
	if _, err := integ.IngestWeighbridge(ctx, service.WeighbridgeRequest{
		Tickets: []domain.WeighbridgeTicket{outside},
	}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a date outside every season has nothing to record against, got %v", err)
	}
}

func TestTheWeighbridgeAccountMayDoNothingElse(t *testing.T) {
	h, exec, integ, _ := integrationHarness(t)
	ctx := h.gate()

	if _, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: h.seeded.Warehouses["FG-WH1"], ProductID: h.seeded.Products["REF"],
			Quantity: domain.D("500"),
		}},
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("the gate terminal must not be able to post stock, got %v", err)
	}

	// And a person who is not the machine account cannot feed the interface.
	if _, err := integ.IngestWeighbridge(h.keeper(), service.WeighbridgeRequest{
		Tickets: []domain.WeighbridgeTicket{ticket("T-1", h.seeded.FactoryID, "42000", "14000", "0")},
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("only the integration account may post gate tickets, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Inbound: the laboratory
// ---------------------------------------------------------------------------

func TestLabResultsAreJudgedLikeAnyOther(t *testing.T) {
	h, exec, integ, _ := integrationHarness(t)
	lab := h.as(auth.RoleQualityUser)

	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: h.seeded.Products["REF"], FactoryID: h.seeded.FactoryID,
		BusinessDate: "2026-12-05",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}

	outcome, err := integ.IngestLabResults(h.gate(), domain.LabResultMessage{
		SampleNo:   sample.SampleNo,
		Instrument: "POLARIMETER-2",
		MeasuredAt: time.Date(2026, 12, 5, 6, 30, 0, 0, time.UTC),
		// A pol of 90 is far below the refined specification.
		Readings: []domain.LabReading{{ParameterCode: "pol", Value: domain.D("90")}},
		Complete: true,
	})
	if err != nil {
		t.Fatalf("ingest lab results: %v", err)
	}
	if outcome.Verdict != domain.QualityFail {
		t.Errorf("an instrument reading must reach the same verdict as a typed one, got %s",
			outcome.Verdict)
	}
	if len(outcome.Sample.Results) != 1 {
		t.Fatalf("the reading must be stored, got %d results", len(outcome.Sample.Results))
	}
	if !strings.Contains(outcome.Sample.Results[0].Comment, "POLARIMETER-2") {
		t.Errorf("the instrument must be recorded against the reading, got %q",
			outcome.Sample.Results[0].Comment)
	}
}

func TestAnUnknownSampleOrParameterIsRefused(t *testing.T) {
	h, exec, integ, _ := integrationHarness(t)
	ctx := h.gate()

	if _, err := integ.IngestLabResults(ctx, domain.LabResultMessage{
		SampleNo: "QS-F1-2026-99999",
		Readings: []domain.LabReading{{ParameterCode: "POL", Value: domain.D("99")}},
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("a reading with no sample judges nothing and must be refused, got %v", err)
	}

	sample, err := exec.CreateSample(h.as(auth.RoleQualityUser), service.SampleRequest{
		ProductID: h.seeded.Products["REF"], FactoryID: h.seeded.FactoryID,
		BusinessDate: "2026-12-05",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	_, err = integ.IngestLabResults(ctx, domain.LabResultMessage{
		SampleNo: sample.SampleNo,
		Readings: []domain.LabReading{{ParameterCode: "VISCOSITY", Value: domain.D("3")}},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("an unknown parameter must be refused, got %v", err)
	}
	if !strings.Contains(err.Error(), "VISCOSITY") {
		t.Errorf("the message must name the parameter it did not recognise, got %v", err)
	}
}

func parameterByCode(t *testing.T, parameters []domain.QualityParameter, code string) string {
	t.Helper()
	for _, p := range parameters {
		if strings.EqualFold(p.Code, code) {
			return p.ID
		}
	}
	t.Fatalf("the seeded catalogue has no %s parameter", code)
	return ""
}
