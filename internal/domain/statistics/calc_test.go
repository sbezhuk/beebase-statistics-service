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
	if o.LatestInspectionAt != nil {
		t.Errorf("LatestInspectionAt = %v, want nil", o.LatestInspectionAt)
	}
}

func TestComputeOverview_Counts(t *testing.T) {
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

func TestComputeHarvestStats_NoHarvests(t *testing.T) {
	stats := statistics.ComputeHarvestStats(nil)

	if stats.TotalHarvests != 0 {
		t.Errorf("TotalHarvests = %d, want 0", stats.TotalHarvests)
	}
	if len(stats.TotalAmountByUnit) != 0 {
		t.Errorf("TotalAmountByUnit = %+v, want empty", stats.TotalAmountByUnit)
	}
	if stats.LatestHarvestedAt != nil {
		t.Errorf("LatestHarvestedAt = %v, want nil", stats.LatestHarvestedAt)
	}
	if stats.LatestProduct != nil {
		t.Errorf("LatestProduct = %v, want nil", stats.LatestProduct)
	}
}

func TestComputeHarvestStats_TotalsGroupedByUnitNotMixed(t *testing.T) {
	now := fixedNow()

	harvests := []statistics.Harvest{
		{ID: uuid.New(), Product: "HONEY", Amount: 2, Unit: "kg", HarvestedAt: now.Add(-48 * time.Hour)},
		{ID: uuid.New(), Product: "HONEY", Amount: 1.5, Unit: "kg", HarvestedAt: now.Add(-24 * time.Hour)},
		{ID: uuid.New(), Product: "WAX", Amount: 50, Unit: "g", HarvestedAt: now},
	}

	stats := statistics.ComputeHarvestStats(harvests)

	if stats.TotalHarvests != 3 {
		t.Errorf("TotalHarvests = %d, want 3", stats.TotalHarvests)
	}
	if len(stats.TotalAmountByUnit) != 2 {
		t.Fatalf("TotalAmountByUnit has %d entries, want 2 (kg, g)", len(stats.TotalAmountByUnit))
	}
	// Sorted by unit: "g" before "kg".
	if stats.TotalAmountByUnit[0].Unit != "g" || stats.TotalAmountByUnit[0].Total != 50 {
		t.Errorf("TotalAmountByUnit[0] = %+v, want {g 50}", stats.TotalAmountByUnit[0])
	}
	if stats.TotalAmountByUnit[1].Unit != "kg" || stats.TotalAmountByUnit[1].Total != 3.5 {
		t.Errorf("TotalAmountByUnit[1] = %+v, want {kg 3.5}", stats.TotalAmountByUnit[1])
	}
}

func TestComputeHarvestStats_LatestByHarvestedAt(t *testing.T) {
	now := fixedNow()

	older := statistics.Harvest{ID: uuid.New(), Product: "HONEY", Amount: 1, Unit: "kg", HarvestedAt: now.Add(-24 * time.Hour)}
	newest := statistics.Harvest{ID: uuid.New(), Product: "POLLEN", Amount: 100, Unit: "g", HarvestedAt: now}

	stats := statistics.ComputeHarvestStats([]statistics.Harvest{older, newest})

	if stats.LatestHarvestedAt == nil || !stats.LatestHarvestedAt.Equal(newest.HarvestedAt) {
		t.Errorf("LatestHarvestedAt = %v, want %v", stats.LatestHarvestedAt, newest.HarvestedAt)
	}
	if stats.LatestProduct == nil || *stats.LatestProduct != newest.Product {
		t.Errorf("LatestProduct = %v, want %q", stats.LatestProduct, newest.Product)
	}
}

func TestComputeNeedsAttention_ApiaryWithZeroHivesNeedsAttention(t *testing.T) {
	apiaryEmpty := statistics.Apiary{ID: uuid.New(), Name: "Empty"}
	apiaryWithHive := statistics.Apiary{ID: uuid.New(), Name: "Has a hive"}
	hive := statistics.Hive{ID: uuid.New(), ApiaryID: apiaryWithHive.ID}

	na := statistics.ComputeNeedsAttention(
		[]statistics.Apiary{apiaryEmpty, apiaryWithHive},
		[]statistics.Hive{hive},
		map[uuid.UUID]time.Time{hive.ID: fixedNow()},
		14,
		fixedNow(),
	)

	if na.ApiariesWithoutHives != 1 {
		t.Errorf("ApiariesWithoutHives = %d, want 1 (apiaryEmpty only)", na.ApiariesWithoutHives)
	}
}

func TestComputeNeedsAttention_ApiaryWithOneOrMoreHivesDoesNotNeedAttention(t *testing.T) {
	apiary := statistics.Apiary{ID: uuid.New(), Name: "Has hives"}
	hive1 := statistics.Hive{ID: uuid.New(), ApiaryID: apiary.ID}
	hive2 := statistics.Hive{ID: uuid.New(), ApiaryID: apiary.ID}

	na := statistics.ComputeNeedsAttention(
		[]statistics.Apiary{apiary},
		[]statistics.Hive{hive1, hive2},
		map[uuid.UUID]time.Time{hive1.ID: fixedNow(), hive2.ID: fixedNow()},
		14,
		fixedNow(),
	)

	if na.ApiariesWithoutHives != 0 {
		t.Errorf("ApiariesWithoutHives = %d, want 0", na.ApiariesWithoutHives)
	}
}

func TestComputeNeedsAttention_HiveWithNoInspectionsNeedsInspection(t *testing.T) {
	hive := statistics.Hive{ID: uuid.New()}

	na := statistics.ComputeNeedsAttention(nil, []statistics.Hive{hive}, map[uuid.UUID]time.Time{}, 14, fixedNow())

	if na.HivesNeedingInspection != 1 {
		t.Errorf("HivesNeedingInspection = %d, want 1 (never inspected)", na.HivesNeedingInspection)
	}
}

func TestComputeNeedsAttention_HiveInspectedTodayIsOK(t *testing.T) {
	now := fixedNow()
	hive := statistics.Hive{ID: uuid.New()}

	na := statistics.ComputeNeedsAttention(nil, []statistics.Hive{hive}, map[uuid.UUID]time.Time{hive.ID: now}, 14, now)

	if na.HivesNeedingInspection != 0 {
		t.Errorf("HivesNeedingInspection = %d, want 0 (inspected today)", na.HivesNeedingInspection)
	}
}

func TestComputeNeedsAttention_HiveInspectedSevenDaysAgoIsOK(t *testing.T) {
	now := fixedNow()
	hive := statistics.Hive{ID: uuid.New()}

	na := statistics.ComputeNeedsAttention(nil, []statistics.Hive{hive}, map[uuid.UUID]time.Time{hive.ID: now.AddDate(0, 0, -7)}, 14, now)

	if na.HivesNeedingInspection != 0 {
		t.Errorf("HivesNeedingInspection = %d, want 0 (7 days ago, within 14-day threshold)", na.HivesNeedingInspection)
	}
}

func TestComputeNeedsAttention_HiveInspectedExactlyAtThresholdIsOK(t *testing.T) {
	now := fixedNow()
	hive := statistics.Hive{ID: uuid.New()}

	na := statistics.ComputeNeedsAttention(nil, []statistics.Hive{hive}, map[uuid.UUID]time.Time{hive.ID: now.AddDate(0, 0, -14)}, 14, now)

	if na.HivesNeedingInspection != 0 {
		t.Errorf("HivesNeedingInspection = %d, want 0 (exactly 14 days ago is still OK)", na.HivesNeedingInspection)
	}
}

func TestComputeNeedsAttention_HiveInspectedPastThresholdNeedsInspection(t *testing.T) {
	now := fixedNow()
	hive := statistics.Hive{ID: uuid.New()}

	na := statistics.ComputeNeedsAttention(nil, []statistics.Hive{hive}, map[uuid.UUID]time.Time{hive.ID: now.AddDate(0, 0, -15)}, 14, now)

	if na.HivesNeedingInspection != 1 {
		t.Errorf("HivesNeedingInspection = %d, want 1 (15 days ago, past the 14-day threshold)", na.HivesNeedingInspection)
	}
}

func TestComputeNeedsAttention_MultipleHivesUsesLatestInspectionDatePerHive(t *testing.T) {
	// latestInspectionByHive already holds only the latest date per hive
	// (inspection-service's own aggregation) - this proves
	// ComputeNeedsAttention trusts that value directly rather than
	// somehow re-deriving it.
	now := fixedNow()
	stale := statistics.Hive{ID: uuid.New()}
	fresh := statistics.Hive{ID: uuid.New()}

	na := statistics.ComputeNeedsAttention(
		nil,
		[]statistics.Hive{stale, fresh},
		map[uuid.UUID]time.Time{
			stale.ID: now.AddDate(0, 0, -30), // an older inspection is NOT what's passed in for "fresh"
			fresh.ID: now,
		},
		14,
		now,
	)

	if na.HivesNeedingInspection != 1 {
		t.Errorf("HivesNeedingInspection = %d, want 1 (only stale)", na.HivesNeedingInspection)
	}
}

func TestComputeNeedsAttention_ConfiguredThresholdIsUsed(t *testing.T) {
	now := fixedNow()
	hive := statistics.Hive{ID: uuid.New()}
	latest := now.AddDate(0, 0, -10)

	strict := statistics.ComputeNeedsAttention(nil, []statistics.Hive{hive}, map[uuid.UUID]time.Time{hive.ID: latest}, 5, now)
	if strict.HivesNeedingInspection != 1 {
		t.Errorf("with threshold=5, 10 days ago should need inspection")
	}
	if strict.InspectionWarningThresholdDays != 5 {
		t.Errorf("InspectionWarningThresholdDays = %d, want 5 (echoed back)", strict.InspectionWarningThresholdDays)
	}

	lenient := statistics.ComputeNeedsAttention(nil, []statistics.Hive{hive}, map[uuid.UUID]time.Time{hive.ID: latest}, 30, now)
	if lenient.HivesNeedingInspection != 0 {
		t.Errorf("with threshold=30, 10 days ago should still be OK")
	}
}

func TestComputeNeedsAttention_DefaultThresholdIsUsedWhenPassedThrough(t *testing.T) {
	// ComputeNeedsAttention itself has no default - it always uses
	// whatever thresholdDays the caller passes (inspection-service is
	// the one that falls back to inspectionwarning.DefaultThresholdDays
	// when unconfigured). This just confirms 14 flows through unchanged.
	na := statistics.ComputeNeedsAttention(nil, nil, map[uuid.UUID]time.Time{}, 14, fixedNow())
	if na.InspectionWarningThresholdDays != 14 {
		t.Errorf("InspectionWarningThresholdDays = %d, want 14", na.InspectionWarningThresholdDays)
	}
}
