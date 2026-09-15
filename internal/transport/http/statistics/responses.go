package statistics

import (
	"time"

	"github.com/google/uuid"

	domainstats "github.com/sbezhuk/beebase-statistics-service/internal/domain/statistics"
)

// OverviewResponse is the public representation of the Dashboard's
// top-level summary.
type OverviewResponse struct {
	TotalApiaries        int        `json:"totalApiaries"`
	TotalHives           int        `json:"totalHives"`
	TotalInspections     int        `json:"totalInspections"`
	InspectionsLast7Days int        `json:"inspectionsLast7Days"`
	InspectionsThisMonth int        `json:"inspectionsThisMonth"`
	InspectionsThisYear  int        `json:"inspectionsThisYear"`
	ApiariesWithoutHives int        `json:"apiariesWithoutHives"`
	LatestInspectionAt   *time.Time `json:"latestInspectionAt"`
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
	ApiaryID  uuid.UUID `json:"apiaryId"`
	Name      string    `json:"name"`
	HiveCount int       `json:"hiveCount"`
}

func newApiaryHiveCountResponse(c domainstats.ApiaryHiveCount) ApiaryHiveCountResponse {
	return ApiaryHiveCountResponse{ApiaryID: c.ApiaryID, Name: c.Name, HiveCount: c.HiveCount}
}

// ApiaryStatsResponse is the public representation of the Dashboard's
// apiary-focused section.
type ApiaryStatsResponse struct {
	TotalApiaries        int                       `json:"totalApiaries"`
	ApiariesWithoutHives int                       `json:"apiariesWithoutHives"`
	ApiaryWithMostHives  *ApiaryHiveCountResponse  `json:"apiaryWithMostHives"`
	HiveDistribution     []ApiaryHiveCountResponse `json:"hiveDistribution"`
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
	HiveID          uuid.UUID `json:"hiveId"`
	HiveName        string    `json:"hiveName"`
	ApiaryID        uuid.UUID `json:"apiaryId"`
	ApiaryName      string    `json:"apiaryName"`
	InspectionCount int       `json:"inspectionCount"`
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
	TotalInspections        int                          `json:"totalInspections"`
	InspectionsLast7Days    int                          `json:"inspectionsLast7Days"`
	InspectionsThisMonth    int                          `json:"inspectionsThisMonth"`
	InspectionsThisYear     int                          `json:"inspectionsThisYear"`
	HiveWithMostInspections *HiveInspectionCountResponse `json:"hiveWithMostInspections"`
	LatestInspectionAt      *time.Time                   `json:"latestInspectionAt"`
	ActivityLast30Days      []DayCountResponse           `json:"activityLast30Days"`
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
	InspectionID uuid.UUID `json:"inspectionId"`
	InspectedAt  time.Time `json:"inspectedAt"`
	HiveID       uuid.UUID `json:"hiveId"`
	HiveName     string    `json:"hiveName"`
	ApiaryID     uuid.UUID `json:"apiaryId"`
	ApiaryName   string    `json:"apiaryName"`
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
	TotalHarvests     int                           `json:"totalHarvests"`
	TotalAmountByUnit []HarvestAmountByUnitResponse `json:"totalAmountByUnit"`
	LatestHarvestedAt *time.Time                    `json:"latestHarvestedAt"`
	LatestProduct     *string                       `json:"latestProduct"`
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
	ApiariesWithoutHives           int `json:"apiariesWithoutHives"`
	HivesNeedingInspection         int `json:"hivesNeedingInspection"`
	InspectionWarningThresholdDays int `json:"inspectionWarningThresholdDays"`
}

func newNeedsAttentionResponse(s domainstats.NeedsAttention) NeedsAttentionResponse {
	return NeedsAttentionResponse{
		ApiariesWithoutHives:           s.ApiariesWithoutHives,
		HivesNeedingInspection:         s.HivesNeedingInspection,
		InspectionWarningThresholdDays: s.InspectionWarningThresholdDays,
	}
}
