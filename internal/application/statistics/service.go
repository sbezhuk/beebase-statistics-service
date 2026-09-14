package statistics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domainstats "github.com/sbezhuk/beebase-statistics-service/internal/domain/statistics"
)

// Service implements the Dashboard use cases. Every method takes the
// caller's own access token and forwards it to whichever upstream
// services it needs, so each of those services runs its own ownership
// check exactly as it would for a direct request from the caller - this
// service never trusts a userID from anywhere but that verified token.
type Service struct {
	apiaries    ApiaryLister
	hives       HiveLister
	inspections InspectionLister
	harvests    HarvestLister
}

// NewService constructs a Service.
func NewService(apiaries ApiaryLister, hives HiveLister, inspections InspectionLister, harvests HarvestLister) *Service {
	return &Service{apiaries: apiaries, hives: hives, inspections: inspections, harvests: harvests}
}

// Overview returns the Dashboard's top-level summary.
func (s *Service) Overview(ctx context.Context, accessToken string) (domainstats.Overview, error) {
	apiaries, hives, inspections, err := s.fetchAll(ctx, accessToken)
	if err != nil {
		return domainstats.Overview{}, err
	}
	return domainstats.ComputeOverview(apiaries, hives, inspections, time.Now().UTC()), nil
}

// ApiaryStats returns the Dashboard's apiary-focused section. It only
// needs apiaries and hives, so it skips fetching inspections entirely.
func (s *Service) ApiaryStats(ctx context.Context, accessToken string) (domainstats.ApiaryStats, error) {
	apiaries, err := s.apiaries.ListAll(ctx, accessToken)
	if err != nil {
		return domainstats.ApiaryStats{}, fmt.Errorf("statistics: list apiaries: %w", err)
	}
	hives, err := s.hives.ListAll(ctx, accessToken)
	if err != nil {
		return domainstats.ApiaryStats{}, fmt.Errorf("statistics: list hives: %w", err)
	}
	return domainstats.ComputeApiaryStats(apiaries, hives), nil
}

// InspectionStats returns the Dashboard's inspection-focused section.
func (s *Service) InspectionStats(ctx context.Context, accessToken string) (domainstats.InspectionStats, error) {
	apiaries, hives, inspections, err := s.fetchAll(ctx, accessToken)
	if err != nil {
		return domainstats.InspectionStats{}, err
	}
	return domainstats.ComputeInspectionStats(apiaries, hives, inspections, time.Now().UTC()), nil
}

// RecentActivity returns the caller's most recent inspections, newest
// first, capped at limit.
func (s *Service) RecentActivity(ctx context.Context, accessToken string, limit int) ([]domainstats.ActivityItem, error) {
	apiaries, hives, inspections, err := s.fetchAll(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return domainstats.ComputeRecentActivity(apiaries, hives, inspections, limit), nil
}

// HarvestStats returns the Dashboard's harvest-focused section. It only
// needs hives (to know which hives to ask harvest-service for), so it
// skips fetching apiaries and inspections entirely.
func (s *Service) HarvestStats(ctx context.Context, accessToken string) (domainstats.HarvestStats, error) {
	hives, err := s.hives.ListAll(ctx, accessToken)
	if err != nil {
		return domainstats.HarvestStats{}, fmt.Errorf("statistics: list hives: %w", err)
	}

	hiveIDs := make([]uuid.UUID, len(hives))
	for i, h := range hives {
		hiveIDs[i] = h.ID
	}

	harvests, err := s.harvests.ListAllForHives(ctx, accessToken, hiveIDs)
	if err != nil {
		return domainstats.HarvestStats{}, fmt.Errorf("statistics: list harvests: %w", err)
	}

	return domainstats.ComputeHarvestStats(harvests), nil
}

// fetchAll fetches every apiary, hive, and inspection belonging to
// whoever presented accessToken. Apiaries and hives are typically few
// for a single beekeeper; inspections can be large for a long-running
// account - a full page-through on every call is an accepted tradeoff for
// this personal-scale app, not something worth caching or bounding here.
func (s *Service) fetchAll(ctx context.Context, accessToken string) ([]domainstats.Apiary, []domainstats.Hive, []domainstats.Inspection, error) {
	apiaries, err := s.apiaries.ListAll(ctx, accessToken)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("statistics: list apiaries: %w", err)
	}
	hives, err := s.hives.ListAll(ctx, accessToken)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("statistics: list hives: %w", err)
	}
	inspections, err := s.inspections.ListAll(ctx, accessToken)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("statistics: list inspections: %w", err)
	}
	return apiaries, hives, inspections, nil
}
