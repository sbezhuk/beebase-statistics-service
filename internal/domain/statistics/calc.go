package statistics

import (
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/sbezhuk/beebase-common/inspectionwarning"
)

// Overview is the Dashboard's top-level summary, combining apiary, hive,
// and inspection data.
type Overview struct {
	TotalApiaries        int
	TotalHives           int
	TotalInspections     int
	InspectionsLast7Days int
	InspectionsThisMonth int
	InspectionsThisYear  int
	ApiariesWithoutHives int
	// LatestInspectionAt is nil when the caller has no inspections yet.
	LatestInspectionAt *time.Time
}

// ComputeOverview builds Overview from apiaries, hives, and inspections
// as they stood at now. now is a parameter (rather than time.Now()
// inside) so callers can test day/month/year boundaries deterministically.
func ComputeOverview(apiaries []Apiary, hives []Hive, inspections []Inspection, now time.Time) Overview {
	last7, thisMonth, thisYear, latest := inspectionWindowCounts(inspections, now)

	return Overview{
		TotalApiaries:        len(apiaries),
		TotalHives:           len(hives),
		TotalInspections:     len(inspections),
		InspectionsLast7Days: last7,
		InspectionsThisMonth: thisMonth,
		InspectionsThisYear:  thisYear,
		ApiariesWithoutHives: apiariesWithoutHivesCount(apiaries, hives),
		LatestInspectionAt:   latest,
	}
}

// ApiaryHiveCount is one apiary's hive count, as reported in
// ApiaryStats.HiveDistribution.
type ApiaryHiveCount struct {
	ApiaryID  uuid.UUID
	Name      string
	HiveCount int
}

// ApiaryStats is the Dashboard's apiary-focused section.
type ApiaryStats struct {
	TotalApiaries        int
	ApiariesWithoutHives int
	// ApiaryWithMostHives is nil when the caller has no hives at all.
	ApiaryWithMostHives *ApiaryHiveCount
	// HiveDistribution covers every apiary, sorted by hive count
	// descending (ties broken by ApiaryID for determinism) - ready to
	// render as-is for the hive distribution chart.
	HiveDistribution []ApiaryHiveCount
}

// ComputeApiaryStats builds ApiaryStats from apiaries and hives as they
// stood at the time both were fetched.
func ComputeApiaryStats(apiaries []Apiary, hives []Hive) ApiaryStats {
	hivesByApiary := countHivesByApiary(hives)

	dist := make([]ApiaryHiveCount, 0, len(apiaries))
	for _, a := range apiaries {
		dist = append(dist, ApiaryHiveCount{ApiaryID: a.ID, Name: a.Name, HiveCount: hivesByApiary[a.ID]})
	}
	sortByCountThenID(dist)

	var most *ApiaryHiveCount
	if len(dist) > 0 && dist[0].HiveCount > 0 {
		m := dist[0]
		most = &m
	}

	return ApiaryStats{
		TotalApiaries:        len(apiaries),
		ApiariesWithoutHives: apiariesWithoutHivesCount(apiaries, hives),
		ApiaryWithMostHives:  most,
		HiveDistribution:     dist,
	}
}

// HiveInspectionCount is one hive's inspection count, including the
// apiary it belongs to.
type HiveInspectionCount struct {
	HiveID          uuid.UUID
	HiveName        string
	ApiaryID        uuid.UUID
	ApiaryName      string
	InspectionCount int
}

// DayCount is the number of inspections performed on one calendar day
// (UTC), as reported in InspectionStats.ActivityLast30Days.
type DayCount struct {
	Date  string // YYYY-MM-DD, UTC
	Count int
}

// InspectionStats is the Dashboard's inspection-focused section.
type InspectionStats struct {
	TotalInspections     int
	InspectionsLast7Days int
	InspectionsThisMonth int
	InspectionsThisYear  int
	// HiveWithMostInspections is nil when the caller has no inspections.
	HiveWithMostInspections *HiveInspectionCount
	// LatestInspectionAt is nil when the caller has no inspections yet.
	LatestInspectionAt *time.Time
	// ActivityLast30Days always has exactly 30 entries, oldest to
	// newest ending today (UTC), zero-filled for days with no
	// inspections - ready to render as-is for the activity chart.
	ActivityLast30Days []DayCount
}

// ComputeInspectionStats builds InspectionStats from apiaries, hives, and
// inspections as they stood at now.
func ComputeInspectionStats(apiaries []Apiary, hives []Hive, inspections []Inspection, now time.Time) InspectionStats {
	last7, thisMonth, thisYear, latest := inspectionWindowCounts(inspections, now)

	hiveByID := indexHivesByID(hives)
	apiaryByID := indexApiariesByID(apiaries)

	most := hiveWithMostInspections(inspections, hiveByID, apiaryByID)

	return InspectionStats{
		TotalInspections:        len(inspections),
		InspectionsLast7Days:    last7,
		InspectionsThisMonth:    thisMonth,
		InspectionsThisYear:     thisYear,
		HiveWithMostInspections: most,
		LatestInspectionAt:      latest,
		ActivityLast30Days:      activityLast30Days(inspections, now),
	}
}

// ActivityItem is one inspection as shown in the Dashboard's Recent
// Activity feed.
type ActivityItem struct {
	InspectionID uuid.UUID
	InspectedAt  time.Time
	HiveID       uuid.UUID
	HiveName     string
	ApiaryID     uuid.UUID
	ApiaryName   string
	Notes        string
}

// ComputeRecentActivity returns the caller's most recent inspections,
// newest first, capped at limit (no cap when limit <= 0).
func ComputeRecentActivity(apiaries []Apiary, hives []Hive, inspections []Inspection, limit int) []ActivityItem {
	hiveByID := indexHivesByID(hives)
	apiaryByID := indexApiariesByID(apiaries)

	sorted := make([]Inspection, len(inspections))
	copy(sorted, inspections)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].InspectedAt.Equal(sorted[j].InspectedAt) {
			return sorted[i].InspectedAt.After(sorted[j].InspectedAt)
		}
		return sorted[i].ID.String() < sorted[j].ID.String()
	})

	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}

	items := make([]ActivityItem, len(sorted))
	for idx, ins := range sorted {
		h := hiveByID[ins.HiveID]
		a := apiaryByID[h.ApiaryID]
		items[idx] = ActivityItem{
			InspectionID: ins.ID,
			InspectedAt:  ins.InspectedAt,
			HiveID:       h.ID,
			HiveName:     h.Name,
			ApiaryID:     a.ID,
			ApiaryName:   a.Name,
			Notes:        ins.Notes,
		}
	}
	return items
}

// HarvestAmountByUnit is the total harvested amount recorded in one unit,
// as reported in HarvestStats.TotalAmountByUnit. Amounts are only ever
// summed within the same unit: harvest-service allows different products
// (and even the same product) to be recorded in different units, so a
// single combined "total amount" across units would be meaningless.
type HarvestAmountByUnit struct {
	Unit  string
	Total float64
}

// HarvestStats is the Dashboard's harvest-focused section.
type HarvestStats struct {
	TotalHarvests int
	// TotalAmountByUnit covers every unit present in the caller's harvest
	// records, sorted by Unit for determinism. Empty when TotalHarvests
	// is 0.
	TotalAmountByUnit []HarvestAmountByUnit
	// LatestHarvestedAt is nil when the caller has no harvest records yet
	// - a valid, non-error result, not a zero value standing in for one.
	LatestHarvestedAt *time.Time
	// LatestProduct is the product of the most recent harvest record (by
	// HarvestedAt, ties broken by ID for determinism), nil when the
	// caller has no harvest records yet.
	LatestProduct *string
}

// ComputeHarvestStats builds HarvestStats from every harvest record the
// caller owns, across all of their hives. An empty/nil harvests is a
// valid input - not an error - and yields a zero-valued HarvestStats.
func ComputeHarvestStats(harvests []Harvest) HarvestStats {
	if len(harvests) == 0 {
		return HarvestStats{}
	}

	totals := make(map[string]float64, len(harvests))
	for _, h := range harvests {
		totals[h.Unit] += h.Amount
	}

	units := make([]string, 0, len(totals))
	for u := range totals {
		units = append(units, u)
	}
	sort.Strings(units)

	byUnit := make([]HarvestAmountByUnit, len(units))
	for i, u := range units {
		byUnit[i] = HarvestAmountByUnit{Unit: u, Total: totals[u]}
	}

	latest := latestHarvest(harvests)
	latestAt := latest.HarvestedAt
	latestProduct := latest.Product

	return HarvestStats{
		TotalHarvests:     len(harvests),
		TotalAmountByUnit: byUnit,
		LatestHarvestedAt: &latestAt,
		LatestProduct:     &latestProduct,
	}
}

// NeedsAttention is the Dashboard's actionable "Needs Attention"
// section: counts of apiaries and hives that currently need the
// caller's attention, plus the threshold behind the hive count, so the
// client never has to know or duplicate that business rule itself.
type NeedsAttention struct {
	ApiariesWithoutHives int
	// HivesNeedingInspection counts hives that have never been
	// inspected, or whose latest inspection is older than
	// InspectionWarningThresholdDays (see beebase-common/
	// inspectionwarning) - the same rule, and the same threshold, that
	// GET /api/v1/hives?needs_inspection=true on hive-service applies.
	HivesNeedingInspection         int
	InspectionWarningThresholdDays int
}

// ComputeNeedsAttention builds NeedsAttention from apiaries and hives as
// they stood at the time both were fetched, latestInspectionByHive (the
// latest InspectedAt per hive id, from inspection-service's hive-status
// - a hive absent from the map has never been inspected), the currently
// configured threshold, and now.
func ComputeNeedsAttention(apiaries []Apiary, hives []Hive, latestInspectionByHive map[uuid.UUID]time.Time, thresholdDays int, now time.Time) NeedsAttention {
	needing := 0
	for _, h := range hives {
		var latest *time.Time
		if t, ok := latestInspectionByHive[h.ID]; ok {
			latest = &t
		}
		if inspectionwarning.NeedsInspection(latest, thresholdDays, now) {
			needing++
		}
	}

	return NeedsAttention{
		ApiariesWithoutHives:           apiariesWithoutHivesCount(apiaries, hives),
		HivesNeedingInspection:         needing,
		InspectionWarningThresholdDays: thresholdDays,
	}
}

// latestHarvest returns the harvest with the most recent HarvestedAt,
// ties broken by ID for determinism. Callers must ensure harvests is
// non-empty.
func latestHarvest(harvests []Harvest) Harvest {
	best := harvests[0]
	for _, h := range harvests[1:] {
		if h.HarvestedAt.After(best.HarvestedAt) {
			best = h
		} else if h.HarvestedAt.Equal(best.HarvestedAt) && h.ID.String() < best.ID.String() {
			best = h
		}
	}
	return best
}

// apiariesWithoutHivesCount returns the number of apiaries with zero
// hives - no grace period based on the apiary's age, an apiary qualifies
// the instant its last hive is gone. Shared by ComputeOverview,
// ComputeApiaryStats, and ComputeNeedsAttention so the three never
// disagree.
func apiariesWithoutHivesCount(apiaries []Apiary, hives []Hive) int {
	hivesByApiary := countHivesByApiary(hives)
	count := 0
	for _, a := range apiaries {
		if hivesByApiary[a.ID] == 0 {
			count++
		}
	}
	return count
}

func countHivesByApiary(hives []Hive) map[uuid.UUID]int {
	m := make(map[uuid.UUID]int, len(hives))
	for _, h := range hives {
		m[h.ApiaryID]++
	}
	return m
}

func countInspectionsByHive(inspections []Inspection) map[uuid.UUID]int {
	m := make(map[uuid.UUID]int, len(inspections))
	for _, i := range inspections {
		m[i.HiveID]++
	}
	return m
}

func indexHivesByID(hives []Hive) map[uuid.UUID]Hive {
	m := make(map[uuid.UUID]Hive, len(hives))
	for _, h := range hives {
		m[h.ID] = h
	}
	return m
}

func indexApiariesByID(apiaries []Apiary) map[uuid.UUID]Apiary {
	m := make(map[uuid.UUID]Apiary, len(apiaries))
	for _, a := range apiaries {
		m[a.ID] = a
	}
	return m
}

// inspectionWindowCounts scans inspections once for every rolling/
// calendar window Overview and InspectionStats need, plus the latest
// InspectedAt seen. All windows are evaluated in UTC, matching how this
// codebase always stores timestamps (time.Now().UTC()).
func inspectionWindowCounts(inspections []Inspection, now time.Time) (last7, thisMonth, thisYear int, latest *time.Time) {
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	var latestTime time.Time
	hasLatest := false

	for _, ins := range inspections {
		if !ins.InspectedAt.Before(sevenDaysAgo) {
			last7++
		}
		if !ins.InspectedAt.Before(monthStart) {
			thisMonth++
		}
		if !ins.InspectedAt.Before(yearStart) {
			thisYear++
		}
		if !hasLatest || ins.InspectedAt.After(latestTime) {
			latestTime = ins.InspectedAt
			hasLatest = true
		}
	}

	if hasLatest {
		latest = &latestTime
	}
	return last7, thisMonth, thisYear, latest
}

// activityLast30Days buckets inspections by calendar day (UTC) across the
// 30 days ending today, zero-filling any day with no inspections.
func activityLast30Days(inspections []Inspection, now time.Time) []DayCount {
	const days = 30
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := today.AddDate(0, 0, -(days - 1))

	counts := make(map[string]int, days)
	for _, ins := range inspections {
		d := time.Date(ins.InspectedAt.Year(), ins.InspectedAt.Month(), ins.InspectedAt.Day(), 0, 0, 0, 0, time.UTC)
		if d.Before(start) || d.After(today) {
			continue
		}
		counts[d.Format("2006-01-02")]++
	}

	out := make([]DayCount, days)
	for i := 0; i < days; i++ {
		key := start.AddDate(0, 0, i).Format("2006-01-02")
		out[i] = DayCount{Date: key, Count: counts[key]}
	}
	return out
}

// hiveWithMostInspections returns the hive with the highest inspection
// count, ties broken by HiveID for determinism, or nil if inspections is
// empty.
func hiveWithMostInspections(inspections []Inspection, hiveByID map[uuid.UUID]Hive, apiaryByID map[uuid.UUID]Apiary) *HiveInspectionCount {
	counts := countInspectionsByHive(inspections)
	if len(counts) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })

	bestID := ids[0]
	bestCount := counts[bestID]
	for _, id := range ids[1:] {
		if counts[id] > bestCount {
			bestID = id
			bestCount = counts[id]
		}
	}

	h := hiveByID[bestID]
	a := apiaryByID[h.ApiaryID]
	return &HiveInspectionCount{
		HiveID:          h.ID,
		HiveName:        h.Name,
		ApiaryID:        a.ID,
		ApiaryName:      a.Name,
		InspectionCount: bestCount,
	}
}

func sortByCountThenID(dist []ApiaryHiveCount) {
	sort.Slice(dist, func(i, j int) bool {
		if dist[i].HiveCount != dist[j].HiveCount {
			return dist[i].HiveCount > dist[j].HiveCount
		}
		return dist[i].ApiaryID.String() < dist[j].ApiaryID.String()
	})
}
