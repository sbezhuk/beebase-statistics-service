package statistics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sbezhuk/beebase-health/health"

	appstatistics "github.com/sbezhuk/beebase-statistics-service/internal/application/statistics"
)

type historyVerifier struct {
	err   error
	calls int
}

func (f *historyVerifier) Verify(context.Context, string, uuid.UUID) error { f.calls++; return f.err }

type historyEntitlement struct {
	value string
	err   error
	calls int
}

func (f *historyEntitlement) GetEntitlement(context.Context, string) (string, error) {
	f.calls++
	return f.value, f.err
}

type historyFacts struct {
	facts []appstatistics.HealthFact
	err   error
	calls int
	to    time.Time
}

func (f *historyFacts) ListHealthFacts(_ context.Context, _ uuid.UUID, to time.Time) ([]appstatistics.HealthFact, error) {
	f.calls++
	f.to = to
	return f.facts, f.err
}

func newHistoryService(v *historyVerifier, e *historyEntitlement, f *historyFacts) *appstatistics.Service {
	return appstatistics.NewService(&fakeApiaryLister{}, &fakeHiveLister{}, &fakeInspectionLister{}, &fakeHarvestLister{}, v, e, f)
}

func TestHealthHistoryAuthorizationOrder(t *testing.T) {
	v := &historyVerifier{err: appstatistics.ErrHiveNotFound}
	e := &historyEntitlement{value: appstatistics.EntitlementPro}
	f := &historyFacts{}
	svc := newHistoryService(v, e, f)
	_, err := svc.HealthHistory(context.Background(), "token", uuid.New(), day(10), day(10))
	if !errors.Is(err, appstatistics.ErrHiveNotFound) || e.calls != 0 || f.calls != 0 {
		t.Fatalf("err/calls = %v/%d/%d/%d, want ownership failure before entitlement/facts", err, v.calls, e.calls, f.calls)
	}
}

func TestHealthHistoryFreeOwnerRejectedBeforeFacts(t *testing.T) {
	v := &historyVerifier{}
	e := &historyEntitlement{value: "free"}
	f := &historyFacts{}
	_, err := newHistoryService(v, e, f).HealthHistory(context.Background(), "token", uuid.New(), day(10), day(10))
	if !errors.Is(err, appstatistics.ErrHealthHistoryProRequired) || v.calls != 1 || e.calls != 1 || f.calls != 0 {
		t.Fatalf("err/calls = %v/%d/%d/%d", err, v.calls, e.calls, f.calls)
	}
}

func TestHealthHistoryPreservesPreWindowEvidenceAndMarkers(t *testing.T) {
	hiveID := uuid.New()
	preWindow := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	inRange := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	future := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	strength := health.ColonyStrengthStrong
	f := &historyFacts{facts: []appstatistics.HealthFact{
		{ID: preWindow, HiveID: hiveID, InspectedAt: day(1), Type: health.TypeRoutine, Assessment: &appstatistics.HealthFactAssessment{ColonyStrength: &strength}},
		{ID: inRange, HiveID: hiveID, InspectedAt: day(10), Type: health.TypeQueen},
		{ID: future, HiveID: hiveID, InspectedAt: day(20), Type: health.TypeHealth},
	}}
	result, err := newHistoryService(&historyVerifier{}, &historyEntitlement{value: appstatistics.EntitlementPro}, f).HealthHistory(context.Background(), "token", hiveID, day(10), day(12))
	if err != nil {
		t.Fatalf("HealthHistory: %v", err)
	}
	if !f.to.Equal(day(12)) {
		t.Fatalf("facts upper bound = %v, want %v", f.to, day(12))
	}
	if len(result.Points) != 3 || len(result.Inspections) != 1 || result.Inspections[0].ID != inRange {
		t.Fatalf("points/markers = %d/%+v", len(result.Points), result.Inspections)
	}
}

func day(dayOfMonth int) time.Time { return time.Date(2026, 9, dayOfMonth, 0, 0, 0, 0, time.UTC) }
