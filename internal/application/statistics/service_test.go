package statistics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	appstatistics "github.com/sbezhuk/beebase-statistics-service/internal/application/statistics"
	domainstats "github.com/sbezhuk/beebase-statistics-service/internal/domain/statistics"
)

// --- fake ports ---

type fakeApiaryLister struct {
	byToken map[string][]domainstats.Apiary
	err     error
}

func (f *fakeApiaryLister) ListAll(_ context.Context, accessToken string) ([]domainstats.Apiary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byToken[accessToken], nil
}

type fakeHiveLister struct {
	byToken map[string][]domainstats.Hive
	err     error
}

func (f *fakeHiveLister) ListAll(_ context.Context, accessToken string) ([]domainstats.Hive, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byToken[accessToken], nil
}

type fakeInspectionLister struct {
	byToken map[string][]domainstats.Inspection
	err     error
}

func (f *fakeInspectionLister) ListAll(_ context.Context, accessToken string) ([]domainstats.Inspection, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byToken[accessToken], nil
}

// --- tests ---

func TestOverview_ForwardsTokenAndComputesFromFetchedData(t *testing.T) {
	token := "user-token"
	apiary := domainstats.Apiary{ID: uuid.New(), Name: "Home"}
	hive := domainstats.Hive{ID: uuid.New(), ApiaryID: apiary.ID, Name: "Hive 1"}
	insp := domainstats.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: time.Now().UTC()}

	svc := appstatistics.NewService(
		&fakeApiaryLister{byToken: map[string][]domainstats.Apiary{token: {apiary}}},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{byToken: map[string][]domainstats.Inspection{token: {insp}}},
	)

	overview, err := svc.Overview(context.Background(), token)
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if overview.TotalApiaries != 1 || overview.TotalHives != 1 || overview.TotalInspections != 1 {
		t.Errorf("overview = %+v, want totals of 1/1/1", overview)
	}
}

func TestOverview_UpstreamErrorPropagates(t *testing.T) {
	upstreamErr := errors.New("apiary-service unreachable")
	svc := appstatistics.NewService(
		&fakeApiaryLister{err: upstreamErr},
		&fakeHiveLister{},
		&fakeInspectionLister{},
	)

	_, err := svc.Overview(context.Background(), "some-token")
	if err == nil {
		t.Fatal("Overview: got nil error, want upstream failure to propagate")
	}
}

func TestApiaryStats_DoesNotFetchInspections(t *testing.T) {
	token := "user-token"
	apiary := domainstats.Apiary{ID: uuid.New(), Name: "Home"}

	svc := appstatistics.NewService(
		&fakeApiaryLister{byToken: map[string][]domainstats.Apiary{token: {apiary}}},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{}},
		// If ApiaryStats ever calls InspectionLister.ListAll, this makes
		// the test fail loudly instead of silently returning nothing.
		&fakeInspectionLister{err: errors.New("ApiaryStats must not fetch inspections")},
	)

	stats, err := svc.ApiaryStats(context.Background(), token)
	if err != nil {
		t.Fatalf("ApiaryStats: %v", err)
	}
	if stats.TotalApiaries != 1 {
		t.Errorf("TotalApiaries = %d, want 1", stats.TotalApiaries)
	}
}

func TestRecentActivity_ForwardsLimit(t *testing.T) {
	token := "user-token"
	apiary := domainstats.Apiary{ID: uuid.New(), Name: "Home"}
	hive := domainstats.Hive{ID: uuid.New(), ApiaryID: apiary.ID, Name: "Hive 1"}
	now := time.Now().UTC()
	inspections := []domainstats.Inspection{
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now},
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-time.Hour)},
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-2 * time.Hour)},
	}

	svc := appstatistics.NewService(
		&fakeApiaryLister{byToken: map[string][]domainstats.Apiary{token: {apiary}}},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{byToken: map[string][]domainstats.Inspection{token: inspections}},
	)

	items, err := svc.RecentActivity(context.Background(), token, 2)
	if err != nil {
		t.Fatalf("RecentActivity: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}
