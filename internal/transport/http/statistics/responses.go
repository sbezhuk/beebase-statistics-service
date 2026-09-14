package statistics

import (
	"time"

	"github.com/google/uuid"

	domainstats "github.com/sbezhuk/beebase-statistics-service/internal/domain/statistics"
)

// OverviewResponse is the public representation of the Dashboard's
// top-level summary.
type OverviewResponse struct {
	TotalApiaries        int        `json:"total_apiaries"`
	TotalHives           int        `json:"total_hives"`
	TotalInspections     int        `json:"total_inspections"`
	InspectionsLast7Days int        `json:"inspections_last_7_days"`
	InspectionsThisMonth int        `json:"inspections_this_month"`
	InspectionsThisYear  int        `json:"inspections_this_year"`
	ApiariesWithoutHives int        `json:"apiaries_without_hives"`
	LatestInspectionAt   *time.Time `json:"latest_inspection_at"`
}

func newOverviewResponse(o domainstats.Overview) OverviewResponse {
	return OverviewResponse{
		TotalApiaries:        o.TotalApiaries,
		TotalHives:           o.TotalHives,
		TotalInspections:     o.TotalInspections,
		InspectionsLast7Days: o.InspectionsLast7Days,
		InspectionsThisMonth: o.InspectionsThisMonth,
		InspectionsThisYear:  o.InspectionsThisYear,
		ApiariesWithoutHives: o.ApiariesWithoutHives,
		LatestInspectionAt:   o.LatestInspectionAt,
	}
}

// ApiaryHiveCountResponse is one apiary's hive count.
type ApiaryHiveCountResponse struct {
	ApiaryID  uuid.UUID `json:"apiary_id"`
	Name      string    `json:"name"`
	HiveCount int       `json:"hive_count"`
}

func newApiaryHiveCountResponse(c domainstats.ApiaryHiveCount) ApiaryHiveCountResponse {
	return ApiaryHiveCountResponse{ApiaryID: c.ApiaryID, Name: c.Name, HiveCount: c.HiveCount}
}

// ApiaryStatsResponse is the public representation of the Dashboard's
// apiary-focused section.
type ApiaryStatsResponse struct {
	TotalApiaries        int                       `json:"total_apiaries"`
	ApiariesWithoutHives int                       `json:"apiaries_without_hives"`
	ApiaryWithMostHives  *ApiaryHiveCountResponse  `json:"apiary_with_most_hives"`
	HiveDistribution     []ApiaryHiveCountResponse `json:"hive_distribution"`
}

func newApiaryStatsResponse(s domainstats.ApiaryStats) ApiaryStatsResponse {
	dist := make([]ApiaryHiveCountResponse, len(s.HiveDistribution))
	for i, c := range s.HiveDistribution {
		dist[i] = newApiaryHiveCountResponse(c)
	}

	var most *ApiaryHiveCountResponse
	if s.ApiaryWithMostHives != nil {
		m := newApiaryHiveCountResponse(*s.ApiaryWithMostHives)
		most = &m
	}

	return ApiaryStatsResponse{
		TotalApiaries:        s.TotalApiaries,
		ApiariesWithoutHives: s.ApiariesWithoutHives,
		ApiaryWithMostHives:  most,
		HiveDistribution:     dist,
	}
}

// HiveInspectionCountResponse is one hive's inspection count, including
// the apiary it belongs to.
type HiveInspectionCountResponse struct {
	HiveID          uuid.UUID `json:"hive_id"`
	HiveName        string    `json:"hive_name"`
	ApiaryID        uuid.UUID `json:"apiary_id"`
	ApiaryName      string    `json:"apiary_name"`
	InspectionCount int       `json:"inspection_count"`
}

// DayCountResponse is the number of inspections performed on one
// calendar day.
type DayCountResponse struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// InspectionStatsResponse is the public representation of the
// Dashboard's inspection-focused section.
type InspectionStatsResponse struct {
	TotalInspections        int                          `json:"total_inspections"`
	InspectionsLast7Days    int                          `json:"inspections_last_7_days"`
	InspectionsThisMonth    int                          `json:"inspections_this_month"`
	InspectionsThisYear     int                          `json:"inspections_this_year"`
	HiveWithMostInspections *HiveInspectionCountResponse `json:"hive_with_most_inspections"`
	LatestInspectionAt      *time.Time                   `json:"latest_inspection_at"`
	ActivityLast30Days      []DayCountResponse           `json:"activity_last_30_days"`
}

func newInspectionStatsResponse(s domainstats.InspectionStats) InspectionStatsResponse {
	activity := make([]DayCountResponse, len(s.ActivityLast30Days))
	for i, d := range s.ActivityLast30Days {
		activity[i] = DayCountResponse{Date: d.Date, Count: d.Count}
	}

	var most *HiveInspectionCountResponse
	if s.HiveWithMostInspections != nil {
		c := s.HiveWithMostInspections
		most = &HiveInspectionCountResponse{
			HiveID:          c.HiveID,
			HiveName:        c.HiveName,
			ApiaryID:        c.ApiaryID,
			ApiaryName:      c.ApiaryName,
			InspectionCount: c.InspectionCount,
		}
	}

	return InspectionStatsResponse{
		TotalInspections:        s.TotalInspections,
		InspectionsLast7Days:    s.InspectionsLast7Days,
		InspectionsThisMonth:    s.InspectionsThisMonth,
		InspectionsThisYear:     s.InspectionsThisYear,
		HiveWithMostInspections: most,
		LatestInspectionAt:      s.LatestInspectionAt,
		ActivityLast30Days:      activity,
	}
}

// ActivityItemResponse is one inspection as shown in the Dashboard's
// Recent Activity feed.
type ActivityItemResponse struct {
	InspectionID uuid.UUID `json:"inspection_id"`
	InspectedAt  time.Time `json:"inspected_at"`
	HiveID       uuid.UUID `json:"hive_id"`
	HiveName     string    `json:"hive_name"`
	ApiaryID     uuid.UUID `json:"apiary_id"`
	ApiaryName   string    `json:"apiary_name"`
	Notes        string    `json:"notes"`
}

// ActivityResponse is the public representation of the Dashboard's
// Recent Activity feed.
type ActivityResponse struct {
	Items []ActivityItemResponse `json:"items"`
}

func newActivityResponse(items []domainstats.ActivityItem) ActivityResponse {
	out := make([]ActivityItemResponse, len(items))
	for i, item := range items {
		out[i] = ActivityItemResponse{
			InspectionID: item.InspectionID,
			InspectedAt:  item.InspectedAt,
			HiveID:       item.HiveID,
			HiveName:     item.HiveName,
			ApiaryID:     item.ApiaryID,
			ApiaryName:   item.ApiaryName,
			Notes:        item.Notes,
		}
	}
	return ActivityResponse{Items: out}
}

// HarvestAmountByUnitResponse is the total harvested amount recorded in
// one unit.
type HarvestAmountByUnitResponse struct {
	Unit  string  `json:"unit"`
	Total float64 `json:"total"`
}

// HarvestStatsResponse is the public representation of the Dashboard's
// harvest-focused section. A caller with no harvest records yet gets a
// valid response with TotalHarvests 0, an empty TotalAmountByUnit, and
// null latest fields - not an error.
type HarvestStatsResponse struct {
	TotalHarvests     int                           `json:"total_harvests"`
	TotalAmountByUnit []HarvestAmountByUnitResponse `json:"total_amount_by_unit"`
	LatestHarvestedAt *time.Time                    `json:"latest_harvested_at"`
	LatestProduct     *string                       `json:"latest_product"`
}

func newHarvestStatsResponse(s domainstats.HarvestStats) HarvestStatsResponse {
	byUnit := make([]HarvestAmountByUnitResponse, len(s.TotalAmountByUnit))
	for i, u := range s.TotalAmountByUnit {
		byUnit[i] = HarvestAmountByUnitResponse{Unit: u.Unit, Total: u.Total}
	}

	return HarvestStatsResponse{
		TotalHarvests:     s.TotalHarvests,
		TotalAmountByUnit: byUnit,
		LatestHarvestedAt: s.LatestHarvestedAt,
		LatestProduct:     s.LatestProduct,
	}
}

// NeedsAttentionResponse is the public representation of the Dashboard's
// actionable "Needs Attention" section. The client reads
// InspectionWarningThresholdDays from here rather than hardcoding or
// duplicating it - the backend is the only source of truth for that
// value (see beebase-common/inspectionwarning).
type NeedsAttentionResponse struct {
	ApiariesWithoutHives           int `json:"apiaries_without_hives"`
	HivesNeedingInspection         int `json:"hives_needing_inspection"`
	InspectionWarningThresholdDays int `json:"inspection_warning_threshold_days"`
}

func newNeedsAttentionResponse(s domainstats.NeedsAttention) NeedsAttentionResponse {
	return NeedsAttentionResponse{
		ApiariesWithoutHives:           s.ApiariesWithoutHives,
		HivesNeedingInspection:         s.HivesNeedingInspection,
		InspectionWarningThresholdDays: s.InspectionWarningThresholdDays,
	}
}
