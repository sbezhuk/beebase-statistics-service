// Package statistics orchestrates the Dashboard use cases: fetch a
// caller's apiaries, hives, and inspections from their owning services,
// then delegate to domain/statistics for the actual calculation. It
// depends only on the three ports declared here, never on HTTP directly.
package statistics

import (
	"context"

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
