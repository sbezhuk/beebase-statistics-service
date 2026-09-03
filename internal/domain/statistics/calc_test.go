package statistics_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	statistics "github.com/sbezhuk/beebase-statistics-service/internal/domain/statistics"
)

func fixedNow() time.Time {
	// A Wednesday, well clear of month/year boundaries.
	return time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
}

func TestComputeOverview_EmptyInputs(t *testing.T) {
	o := statistics.ComputeOverview(nil, nil, nil, fixedNow())

	if o.TotalApiaries != 0 || o.TotalHives != 0 || o.TotalInspections != 0 {
		t.Fatalf("totals = %+v, want all zero", o)
	}
	if o.AvgHivesPerApiary != 0 {
		t.Errorf("AvgHivesPerApiary = %v, want 0 (no apiaries)", o.AvgHivesPerApiary)
	}
	if o.AvgInspectionsPerHive != 0 {
		t.Errorf("AvgInspectionsPerHive = %v, want 0 (no hives)", o.AvgInspectionsPerHive)
	}
	if o.LatestInspectionAt != nil {
		t.Errorf("LatestInspectionAt = %v, want nil", o.LatestInspectionAt)
	}
}

func TestComputeOverview_CountsAndAverages(t *testing.T) {
	apiaryA := statistics.Apiary{ID: uuid.New(), Name: "Apiary A"}
	apiaryB := statistics.Apiary{ID: uuid.New(), Name: "Apiary B (no hives)"}

	hive1 := statistics.Hive{ID: uuid.New(), ApiaryID: apiaryA.ID, Name: "Hive 1"}
	hive2 := statistics.Hive{ID: uuid.New(), ApiaryID: apiaryA.ID, Name: "Hive 2 (no inspections)"}

	now := fixedNow()
	insp := statistics.Inspection{ID: uuid.New(), HiveID: hive1.ID, InspectedAt: now.Add(-2 * 24 * time.Hour)}

	o := statistics.ComputeOverview(
		[]statistics.Apiary{apiaryA, apiaryB},
		[]statistics.Hive{hive1, hive2},
		[]statistics.Inspection{insp},
		now,
	)

	if o.TotalApiaries != 2 {
		t.Errorf("TotalApiaries = %d, want 2", o.TotalApiaries)
	}
	if o.TotalHives != 2 {
		t.Errorf("TotalHives = %d, want 2", o.TotalHives)
	}
	if o.TotalInspections != 1 {
		t.Errorf("TotalInspections = %d, want 1", o.TotalInspections)
	}
	if o.ApiariesWithoutHives != 1 {
		t.Errorf("ApiariesWithoutHives = %d, want 1", o.ApiariesWithoutHives)
	}
	if o.HivesWithoutInspections != 1 {
		t.Errorf("HivesWithoutInspections = %d, want 1", o.HivesWithoutInspections)
	}
	if o.AvgHivesPerApiary != 1 {
		t.Errorf("AvgHivesPerApiary = %v, want 1 (2 hives / 2 apiaries)", o.AvgHivesPerApiary)
	}
	if o.AvgInspectionsPerHive != 0.5 {
		t.Errorf("AvgInspectionsPerHive = %v, want 0.5 (1 inspection / 2 hives)", o.AvgInspectionsPerHive)
	}
	if o.LatestInspectionAt == nil || !o.LatestInspectionAt.Equal(insp.InspectedAt) {
		t.Errorf("LatestInspectionAt = %v, want %v", o.LatestInspectionAt, insp.InspectedAt)
	}
}

func TestComputeOverview_TimeWindows(t *testing.T) {
	now := fixedNow() // 2026-03-18

	hive := statistics.Hive{ID: uuid.New(), ApiaryID: uuid.New()}

	within7Days := statistics.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-3 * 24 * time.Hour)}
	beyond7DaysThisMonth := statistics.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
	earlierThisYear := statistics.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)}
	lastYear := statistics.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC)}

	o := statistics.ComputeOverview(
		nil,
		[]statistics.Hive{hive},
		[]statistics.Inspection{within7Days, beyond7DaysThisMonth, earlierThisYear, lastYear},
		now,
	)

	if o.InspectionsLast7Days != 1 {
		t.Errorf("InspectionsLast7Days = %d, want 1", o.InspectionsLast7Days)
	}
	if o.InspectionsThisMonth != 2 {
		t.Errorf("InspectionsThisMonth = %d, want 2 (within7Days + beyond7DaysThisMonth)", o.InspectionsThisMonth)
	}
	if o.InspectionsThisYear != 3 {
		t.Errorf("InspectionsThisYear = %d, want 3 (everything but lastYear)", o.InspectionsThisYear)
	}
}

func TestComputeApiaryStats_DistributionSortedDescending(t *testing.T) {
	apiaryFew := statistics.Apiary{ID: uuid.New(), Name: "Few"}
	apiaryMany := statistics.Apiary{ID: uuid.New(), Name: "Many"}
	apiaryNone := statistics.Apiary{ID: uuid.New(), Name: "None"}

	hives := []statistics.Hive{
		{ID: uuid.New(), ApiaryID: apiaryMany.ID},
		{ID: uuid.New(), ApiaryID: apiaryMany.ID},
		{ID: uuid.New(), ApiaryID: apiaryMany.ID},
		{ID: uuid.New(), ApiaryID: apiaryFew.ID},
	}

	stats := statistics.ComputeApiaryStats([]statistics.Apiary{apiaryFew, apiaryMany, apiaryNone}, hives)

	if stats.TotalApiaries != 3 {
		t.Errorf("TotalApiaries = %d, want 3", stats.TotalApiaries)
	}
	if stats.ApiariesWithoutHives != 1 {
		t.Errorf("ApiariesWithoutHives = %d, want 1", stats.ApiariesWithoutHives)
	}
	if len(stats.HiveDistribution) != 3 {
		t.Fatalf("HiveDistribution has %d entries, want 3", len(stats.HiveDistribution))
	}
	if stats.HiveDistribution[0].ApiaryID != apiaryMany.ID || stats.HiveDistribution[0].HiveCount != 3 {
		t.Errorf("HiveDistribution[0] = %+v, want apiaryMany with count 3", stats.HiveDistribution[0])
	}
	if stats.HiveDistribution[1].ApiaryID != apiaryFew.ID || stats.HiveDistribution[1].HiveCount != 1 {
		t.Errorf("HiveDistribution[1] = %+v, want apiaryFew with count 1", stats.HiveDistribution[1])
	}
	if stats.HiveDistribution[2].ApiaryID != apiaryNone.ID || stats.HiveDistribution[2].HiveCount != 0 {
		t.Errorf("HiveDistribution[2] = %+v, want apiaryNone with count 0", stats.HiveDistribution[2])
	}
	if stats.ApiaryWithMostHives == nil || stats.ApiaryWithMostHives.ApiaryID != apiaryMany.ID {
		t.Errorf("ApiaryWithMostHives = %v, want apiaryMany", stats.ApiaryWithMostHives)
	}
}

func TestComputeApiaryStats_NoHivesAtAll(t *testing.T) {
	apiary := statistics.Apiary{ID: uuid.New(), Name: "Solo"}

	stats := statistics.ComputeApiaryStats([]statistics.Apiary{apiary}, nil)

	if stats.ApiaryWithMostHives != nil {
		t.Errorf("ApiaryWithMostHives = %v, want nil when no hives exist", stats.ApiaryWithMostHives)
	}
	if stats.ApiariesWithoutHives != 1 {
		t.Errorf("ApiariesWithoutHives = %d, want 1", stats.ApiariesWithoutHives)
	}
}

func TestComputeInspectionStats_HiveWithMostInspections(t *testing.T) {
	apiary := statistics.Apiary{ID: uuid.New(), Name: "Home Apiary"}
	hiveBusy := statistics.Hive{ID: uuid.New(), ApiaryID: apiary.ID, Name: "Busy Hive"}
	hiveQuiet := statistics.Hive{ID: uuid.New(), ApiaryID: apiary.ID, Name: "Quiet Hive"}

	now := fixedNow()
	inspections := []statistics.Inspection{
		{ID: uuid.New(), HiveID: hiveBusy.ID, InspectedAt: now.Add(-24 * time.Hour)},
		{ID: uuid.New(), HiveID: hiveBusy.ID, InspectedAt: now.Add(-48 * time.Hour)},
		{ID: uuid.New(), HiveID: hiveQuiet.ID, InspectedAt: now.Add(-72 * time.Hour)},
	}

	stats := statistics.ComputeInspectionStats(
		[]statistics.Apiary{apiary},
		[]statistics.Hive{hiveBusy, hiveQuiet},
		inspections,
		now,
	)

	if stats.TotalInspections != 3 {
		t.Errorf("TotalInspections = %d, want 3", stats.TotalInspections)
	}
	if stats.HiveWithMostInspections == nil {
		t.Fatal("HiveWithMostInspections = nil, want hiveBusy")
	}
	if stats.HiveWithMostInspections.HiveID != hiveBusy.ID {
		t.Errorf("HiveWithMostInspections.HiveID = %s, want %s", stats.HiveWithMostInspections.HiveID, hiveBusy.ID)
	}
	if stats.HiveWithMostInspections.ApiaryName != apiary.Name {
		t.Errorf("HiveWithMostInspections.ApiaryName = %q, want %q", stats.HiveWithMostInspections.ApiaryName, apiary.Name)
	}
	if stats.HiveWithMostInspections.InspectionCount != 2 {
		t.Errorf("HiveWithMostInspections.InspectionCount = %d, want 2", stats.HiveWithMostInspections.InspectionCount)
	}
}

func TestComputeInspectionStats_NoInspections(t *testing.T) {
	stats := statistics.ComputeInspectionStats(nil, nil, nil, fixedNow())

	if stats.HiveWithMostInspections != nil {
		t.Errorf("HiveWithMostInspections = %v, want nil", stats.HiveWithMostInspections)
	}
	if stats.LatestInspectionAt != nil {
		t.Errorf("LatestInspectionAt = %v, want nil", stats.LatestInspectionAt)
	}
	if len(stats.ActivityLast30Days) != 30 {
		t.Fatalf("ActivityLast30Days has %d entries, want 30", len(stats.ActivityLast30Days))
	}
	for _, d := range stats.ActivityLast30Days {
		if d.Count != 0 {
			t.Errorf("day %s has count %d, want 0", d.Date, d.Count)
		}
	}
}

func TestComputeInspectionStats_ActivityLast30Days_ZeroFilledAndBounded(t *testing.T) {
	now := fixedNow() // 2026-03-18
	hive := statistics.Hive{ID: uuid.New()}

	inspections := []statistics.Inspection{
		// Today: two inspections on the same day should collapse into one bucket of 2.
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now},
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-1 * time.Hour)},
		// Exactly 29 days ago: the oldest day still in range.
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.AddDate(0, 0, -29)},
		// 31 days ago: out of range, must not appear anywhere in the 30 buckets.
		{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.AddDate(0, 0, -31)},
	}

	stats := statistics.ComputeInspectionStats(nil, []statistics.Hive{hive}, inspections, now)

	activity := stats.ActivityLast30Days
	if len(activity) != 30 {
		t.Fatalf("ActivityLast30Days has %d entries, want 30", len(activity))
	}

	wantToday := now.Format("2006-01-02")
	if activity[29].Date != wantToday {
		t.Errorf("last bucket date = %q, want %q (today)", activity[29].Date, wantToday)
	}
	if activity[29].Count != 2 {
		t.Errorf("today's count = %d, want 2", activity[29].Count)
	}

	wantOldest := now.AddDate(0, 0, -29).Format("2006-01-02")
	if activity[0].Date != wantOldest {
		t.Errorf("first bucket date = %q, want %q", activity[0].Date, wantOldest)
	}
	if activity[0].Count != 1 {
		t.Errorf("oldest-in-range count = %d, want 1", activity[0].Count)
	}

	total := 0
	for _, d := range activity {
		total += d.Count
	}
	if total != 3 {
		t.Errorf("total inspections across 30-day window = %d, want 3 (the 31-days-ago one must be excluded)", total)
	}
}

func TestComputeRecentActivity_NewestFirstAndLimited(t *testing.T) {
	apiary := statistics.Apiary{ID: uuid.New(), Name: "Home"}
	hive := statistics.Hive{ID: uuid.New(), ApiaryID: apiary.ID, Name: "Hive 1"}

	now := fixedNow()
	oldest := statistics.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-72 * time.Hour), Notes: "oldest"}
	middle := statistics.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-48 * time.Hour), Notes: "middle"}
	newest := statistics.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-24 * time.Hour), Notes: "newest"}

	items := statistics.ComputeRecentActivity(
		[]statistics.Apiary{apiary},
		[]statistics.Hive{hive},
		[]statistics.Inspection{oldest, middle, newest},
		2,
	)

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2 (limited)", len(items))
	}
	if items[0].InspectionID != newest.ID {
		t.Errorf("items[0] = %s, want newest %s", items[0].InspectionID, newest.ID)
	}
	if items[1].InspectionID != middle.ID {
		t.Errorf("items[1] = %s, want middle %s", items[1].InspectionID, middle.ID)
	}
	if items[0].HiveName != hive.Name || items[0].ApiaryName != apiary.Name {
		t.Errorf("items[0] hive/apiary names = %q/%q, want %q/%q", items[0].HiveName, items[0].ApiaryName, hive.Name, apiary.Name)
	}
}

func TestComputeRecentActivity_NoLimitReturnsEverything(t *testing.T) {
	hive := statistics.Hive{ID: uuid.New()}
	now := fixedNow()

	inspections := make([]statistics.Inspection, 5)
	for i := range inspections {
		inspections[i] = statistics.Inspection{ID: uuid.New(), HiveID: hive.ID, InspectedAt: now.Add(-time.Duration(i) * time.Hour)}
	}

	items := statistics.ComputeRecentActivity(nil, []statistics.Hive{hive}, inspections, 0)
	if len(items) != 5 {
		t.Fatalf("len(items) = %d, want 5 (limit<=0 means unbounded)", len(items))
	}
}
