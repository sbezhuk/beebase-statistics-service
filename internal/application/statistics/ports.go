// Package statistics orchestrates the Dashboard use cases: fetch a
// caller's apiaries, hives, and inspections from their owning services,
// then delegate to domain/statistics for the actual calculation. It
// depends only on the three ports declared here, never on HTTP directly.
package statistics

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainstats "github.com/sbezhuk/beebase-statistics-service/internal/domain/statistics"
)

// ApiaryLister is this service's dependency on apiary-service.
type ApiaryLister interface {
	// ListAll returns every apiary belonging to whoever presented
	// accessToken.
	ListAll(ctx context.Context, accessToken string) ([]domainstats.Apiary, error)
}

// HiveLister is this service's dependency on hive-service.
type HiveLister interface {
	// ListAll returns every hive belonging to whoever presented
	// accessToken.
	ListAll(ctx context.Context, accessToken string) ([]domainstats.Hive, error)
}

// InspectionLister is this service's dependency on inspection-service.
type InspectionLister interface {
	// ListAll returns every inspection belonging to whoever presented
	// accessToken, across all of their hives.
	ListAll(ctx context.Context, accessToken string) ([]domainstats.Inspection, error)
	// ListRecent returns at most limit of whoever presented accessToken's
	// most recent inspections, newest first by InspectedAt, without
	// paging through their entire inspection history to get there.
	ListRecent(ctx context.Context, accessToken string, limit int) ([]domainstats.Inspection, error)
	// HiveInspectionStatus returns the latest InspectedAt for every hive
	// whoever presented accessToken has ever inspected (a hive absent
	// from the map has never been inspected), and the currently
	// configured inspection warning threshold in days - the same single
	// source of truth hive-service's own needs_inspection filter reads,
	// so the Dashboard's NeedsAttention count and hive-service's filter
	// never disagree.
	HiveInspectionStatus(ctx context.Context, accessToken string) (latestByHive map[uuid.UUID]time.Time, thresholdDays int, err error)
}

// HarvestLister is this service's dependency on harvest-service.
// Harvest-service scopes its global list by the caller's authenticated
// identity and pages through all owned harvests in one resource API.
type HarvestLister interface {
	ListAll(ctx context.Context, accessToken string) ([]domainstats.Harvest, error)
}
