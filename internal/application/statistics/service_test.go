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

	// recentByToken/recentErr back ListRecent independently of
	// byToken/err (which back ListAll), so a test can make one fail
	// while the other succeeds - proving which one RecentActivity
	// actually calls.
	recentByToken map[string][]domainstats.Inspection
	recentErr     error
	// lastLimit records the limit ListRecent was last called with, so a
	// test can assert it was forwarded correctly.
	lastLimit int

	// statusByToken/thresholdDays/statusErr back HiveInspectionStatus,
	// independently of the other fields above.
	statusByToken map[string]map[uuid.UUID]time.Time
	thresholdDays int
	statusErr     error
}

func (f *fakeInspectionLister) ListAll(_ context.Context, accessToken string) ([]domainstats.Inspection, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byToken[accessToken], nil
}

// ListRecent mimics inspectionclient.Client.ListRecent's contract: at
// most limit items, already newest-first.
func (f *fakeInspectionLister) ListRecent(_ context.Context, accessToken string, limit int) ([]domainstats.Inspection, error) {
	f.lastLimit = limit
	if f.recentErr != nil {
		return nil, f.recentErr
	}
	items := f.recentByToken[accessToken]
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// HiveInspectionStatus mimics inspectionclient.Client.HiveInspectionStatus's
// contract: the latest InspectedAt per hive (only for hives with at
// least one), plus the configured threshold.
func (f *fakeInspectionLister) HiveInspectionStatus(_ context.Context, accessToken string) (map[uuid.UUID]time.Time, int, error) {
	if f.statusErr != nil {
		return nil, 0, f.statusErr
	}
	return f.statusByToken[accessToken], f.thresholdDays, nil
}

type fakeHarvestLister struct {
	byToken map[string][]domainstats.Harvest
	err     error
}

func (f *fakeHarvestLister) ListAllForHives(_ context.Context, accessToken string, _ []uuid.UUID) ([]domainstats.Harvest, error) {
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
		&fakeHarvestLister{},
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
		&fakeHarvestLister{},
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
		&fakeHarvestLister{err: errors.New("ApiaryStats must not fetch harvests")},
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
	// Already newest-first, matching ListRecent's real contract.
	inspections := []domainstats.Inspection{
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now},
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-time.Hour)},
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-2 * time.Hour)},
	}

	inspectionLister := &fakeInspectionLister{recentByToken: map[string][]domainstats.Inspection{token: inspections}}
	svc := appstatistics.NewService(
		&fakeApiaryLister{byToken: map[string][]domainstats.Apiary{token: {apiary}}},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		inspectionLister,
		&fakeHarvestLister{},
	)

	items, err := svc.RecentActivity(context.Background(), token, 2)
	if err != nil {
		t.Fatalf("RecentActivity: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if inspectionLister.lastLimit != 2 {
		t.Errorf("ListRecent called with limit = %d, want 2", inspectionLister.lastLimit)
	}
}

func TestRecentActivity_DoesNotFetchEntireInspectionHistory(t *testing.T) {
	token := "user-token"
	hive := domainstats.Hive{ID: uuid.New(), Name: "Hive 1"}
	insp := domainstats.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: time.Now().UTC()}

	svc := appstatistics.NewService(
		&fakeApiaryLister{},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{
			// If RecentActivity ever calls ListAll instead of ListRecent,
			// this makes the test fail loudly instead of silently
			// returning the full history.
			err:           errors.New("RecentActivity must not fetch the entire inspection history"),
			recentByToken: map[string][]domainstats.Inspection{token: {insp}},
		},
		&fakeHarvestLister{},
	)

	items, err := svc.RecentActivity(context.Background(), token, 10)
	if err != nil {
		t.Fatalf("RecentActivity: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}

func TestRecentActivity_FewerItemsWhenFewerExist(t *testing.T) {
	token := "user-token"
	hive := domainstats.Hive{ID: uuid.New(), Name: "Hive 1"}
	insp := domainstats.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: time.Now().UTC()}

	svc := appstatistics.NewService(
		&fakeApiaryLister{},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{recentByToken: map[string][]domainstats.Inspection{token: {insp}}},
		&fakeHarvestLister{},
	)

	items, err := svc.RecentActivity(context.Background(), token, 10)
	if err != nil {
		t.Fatalf("RecentActivity: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1 (only one inspection exists, limit was 10)", len(items))
	}
}

func TestRecentActivity_LimitOneReturnsOnlyTheLatest(t *testing.T) {
	token := "user-token"
	hive := domainstats.Hive{ID: uuid.New(), Name: "Hive 1"}
	now := time.Now().UTC()
	newest := domainstats.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now, Notes: "newest"}
	older := domainstats.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-time.Hour), Notes: "older"}

	svc := appstatistics.NewService(
		&fakeApiaryLister{},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{recentByToken: map[string][]domainstats.Inspection{token: {newest, older}}},
		&fakeHarvestLister{},
	)

	items, err := svc.RecentActivity(context.Background(), token, 1)
	if err != nil {
		t.Fatalf("RecentActivity: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].InspectionID != newest.ID {
		t.Errorf("items[0] = %s, want the newest inspection %s", items[0].InspectionID, newest.ID)
	}
}

func TestRecentActivity_UpstreamErrorPropagates(t *testing.T) {
	token := "user-token"
	hive := domainstats.Hive{ID: uuid.New(), Name: "Hive 1"}

	svc := appstatistics.NewService(
		&fakeApiaryLister{},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{recentErr: errors.New("inspection-service unreachable")},
		&fakeHarvestLister{},
	)

	if _, err := svc.RecentActivity(context.Background(), token, 10); err == nil {
		t.Fatal("RecentActivity: got nil error, want upstream failure to propagate")
	}
}

func TestHarvestStats_ForwardsTokenAndComputesFromFetchedData(t *testing.T) {
	token := "user-token"
	hive := domainstats.Hive{ID: uuid.New(), Name: "Hive 1"}
	harvest := domainstats.Harvest{ID: uuid.New(), Product: "HONEY", Amount: 2, Unit: "kg", HarvestedAt: time.Now().UTC()}

	svc := appstatistics.NewService(
		&fakeApiaryLister{},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{},
		&fakeHarvestLister{byToken: map[string][]domainstats.Harvest{token: {harvest}}},
	)

	stats, err := svc.HarvestStats(context.Background(), token)
	if err != nil {
		t.Fatalf("HarvestStats: %v", err)
	}
	if stats.TotalHarvests != 1 {
		t.Errorf("TotalHarvests = %d, want 1", stats.TotalHarvests)
	}
}

func TestHarvestStats_DoesNotFetchApiariesOrInspections(t *testing.T) {
	token := "user-token"
	hive := domainstats.Hive{ID: uuid.New(), Name: "Hive 1"}

	svc := appstatistics.NewService(
		// If HarvestStats ever calls ApiaryLister.ListAll, this makes the
		// test fail loudly instead of silently returning nothing.
		&fakeApiaryLister{err: errors.New("HarvestStats must not fetch apiaries")},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{err: errors.New("HarvestStats must not fetch inspections")},
		&fakeHarvestLister{},
	)

	if _, err := svc.HarvestStats(context.Background(), token); err != nil {
		t.Fatalf("HarvestStats: %v", err)
	}
}

func TestHarvestStats_UpstreamErrorPropagates(t *testing.T) {
	token := "user-token"
	hive := domainstats.Hive{ID: uuid.New(), Name: "Hive 1"}

	svc := appstatistics.NewService(
		&fakeApiaryLister{},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{},
		&fakeHarvestLister{err: errors.New("harvest-service unreachable")},
	)

	if _, err := svc.HarvestStats(context.Background(), token); err == nil {
		t.Fatal("HarvestStats: got nil error, want upstream failure to propagate")
	}
}

func TestNeedsAttention_ForwardsTokenAndComputesFromFetchedData(t *testing.T) {
	token := "user-token"
	apiaryEmpty := domainstats.Apiary{ID: uuid.New(), Name: "Empty"}
	apiaryWithHive := domainstats.Apiary{ID: uuid.New(), Name: "Has a hive"}
	hive := domainstats.Hive{ID: uuid.New(), ApiaryID: apiaryWithHive.ID, Name: "Hive 1"}

	svc := appstatistics.NewService(
		&fakeApiaryLister{byToken: map[string][]domainstats.Apiary{token: {apiaryEmpty, apiaryWithHive}}},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{
			statusByToken: map[string]map[uuid.UUID]time.Time{token: {}}, // never inspected
			thresholdDays: 14,
		},
		&fakeHarvestLister{},
	)

	na, err := svc.NeedsAttention(context.Background(), token)
	if err != nil {
		t.Fatalf("NeedsAttention: %v", err)
	}
	if na.ApiariesWithoutHives != 1 {
		t.Errorf("ApiariesWithoutHives = %d, want 1", na.ApiariesWithoutHives)
	}
	if na.HivesNeedingInspection != 1 {
		t.Errorf("HivesNeedingInspection = %d, want 1 (never inspected)", na.HivesNeedingInspection)
	}
	if na.InspectionWarningThresholdDays != 14 {
		t.Errorf("InspectionWarningThresholdDays = %d, want 14", na.InspectionWarningThresholdDays)
	}
}

func TestNeedsAttention_DoesNotFetchEntireInspectionHistory(t *testing.T) {
	token := "user-token"
	hive := domainstats.Hive{ID: uuid.New(), Name: "Hive 1"}

	svc := appstatistics.NewService(
		&fakeApiaryLister{},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{
			// If NeedsAttention ever calls ListAll/ListRecent instead of
			// HiveInspectionStatus, this makes the test fail loudly
			// instead of silently returning the full history.
			err:           errors.New("NeedsAttention must not fetch the full inspection history"),
			recentErr:     errors.New("NeedsAttention must not call ListRecent"),
			statusByToken: map[string]map[uuid.UUID]time.Time{token: {hive.ID: time.Now().UTC()}},
			thresholdDays: 14,
		},
		&fakeHarvestLister{},
	)

	na, err := svc.NeedsAttention(context.Background(), token)
	if err != nil {
		t.Fatalf("NeedsAttention: %v", err)
	}
	if na.HivesNeedingInspection != 0 {
		t.Errorf("HivesNeedingInspection = %d, want 0 (recently inspected)", na.HivesNeedingInspection)
	}
}

func TestNeedsAttention_UpstreamErrorPropagates(t *testing.T) {
	token := "user-token"
	hive := domainstats.Hive{ID: uuid.New(), Name: "Hive 1"}

	svc := appstatistics.NewService(
		&fakeApiaryLister{},
		&fakeHiveLister{byToken: map[string][]domainstats.Hive{token: {hive}}},
		&fakeInspectionLister{statusErr: errors.New("inspection-service unreachable")},
		&fakeHarvestLister{},
	)

	if _, err := svc.NeedsAttention(context.Background(), token); err == nil {
		t.Fatal("NeedsAttention: got nil error, want upstream failure to propagate")
	}
}
