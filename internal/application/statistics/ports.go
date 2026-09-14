// Package statistics orchestrates the Dashboard use cases: fetch a
// caller's apiaries, hives, and inspections from their owning services,
// then delegate to domain/statistics for the actual calculation. It
// depends only on the three ports declared here, never on HTTP directly.
package statistics

import (
	"context"

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
}

// HarvestLister is this service's dependency on harvest-service.
// Unlike the other listers, harvest-service exposes no endpoint that
// lists every harvest a caller owns in one call - only GET
// /hives/{hiveID}/harvest, scoped to a single hive - so ListAllForHives
// takes the caller's hive IDs (already fetched via HiveLister) and fans
// out across them.
type HarvestLister interface {
	// ListAllForHives returns every harvest record across hiveIDs,
	// forwarding accessToken so harvest-service can verify each hive
	// belongs to whoever presented it.
	ListAllForHives(ctx context.Context, accessToken string, hiveIDs []uuid.UUID) ([]domainstats.Harvest, error)
}
