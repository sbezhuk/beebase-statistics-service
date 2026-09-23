package statistics

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sbezhuk/beebase-health/health"

	domainstats "github.com/sbezhuk/beebase-statistics-service/internal/domain/statistics"
)

// Service implements the Dashboard use cases. Every method takes the
// caller's own access token and forwards it to whichever upstream
// services it needs, so each of those services runs its own ownership
// check exactly as it would for a direct request from the caller - this
// service never trusts a userID from anywhere but that verified token.
type Service struct {
	apiaries     ApiaryLister
	hives        HiveLister
	inspections  InspectionLister
	harvests     HarvestLister
	hiveVerifier HiveVerifier
	entitlements EntitlementResolver
	healthFacts  HealthFactsReader
}

var (
	ErrHiveNotFound             = errors.New("hive not found")
	ErrHealthHistoryProRequired = errors.New("health history requires pro")
)

// NewService constructs a Service.
func NewService(apiaries ApiaryLister, hives HiveLister, inspections InspectionLister, harvests HarvestLister, extras ...any) *Service {
	s := &Service{apiaries: apiaries, hives: hives, inspections: inspections, harvests: harvests}
	for _, extra := range extras {
		switch value := extra.(type) {
		case HiveVerifier:
			s.hiveVerifier = value
		case EntitlementResolver:
			s.entitlements = value
		case HealthFactsReader:
			s.healthFacts = value
		}
	}
	return s
}

type HealthHistoryPoint struct {
	Date       time.Time
	Evaluation health.ColonyHealthEvaluation
}

type HealthHistoryInspection struct {
	ID   uuid.UUID
	Date time.Time
	Type health.Type
}

type HealthHistoryResult struct {
	From        time.Time
	To          time.Time
	Points      []HealthHistoryPoint
	Inspections []HealthHistoryInspection
}

func (s *Service) HealthHistory(ctx context.Context, accessToken string, hiveID uuid.UUID, from, to time.Time) (HealthHistoryResult, error) {
	if s.hiveVerifier == nil || s.entitlements == nil || s.healthFacts == nil {
		return HealthHistoryResult{}, fmt.Errorf("statistics: health history dependencies are not configured")
	}
	if err := s.hiveVerifier.Verify(ctx, accessToken, hiveID); err != nil {
		return HealthHistoryResult{}, err
	}
	entitlement, err := s.entitlements.GetEntitlement(ctx, accessToken)
	if err != nil {
		return HealthHistoryResult{}, fmt.Errorf("statistics: resolve entitlement: %w", err)
	}
	if entitlement != EntitlementPro {
		return HealthHistoryResult{}, ErrHealthHistoryProRequired
	}
	facts, err := s.healthFacts.ListHealthFacts(ctx, hiveID, to)
	if err != nil {
		return HealthHistoryResult{}, fmt.Errorf("statistics: list health facts: %w", err)
	}

	evidence := make([]health.HealthEvidence, 0)
	for _, fact := range facts {
		input, err := healthInspection(fact)
		if err != nil {
			return HealthHistoryResult{}, fmt.Errorf("statistics: adapt health fact: %w", err)
		}
		normalized := health.NormalizeInspection(input)
		evidence = append(evidence, normalized.HealthEvidence...)
	}

	points := make([]HealthHistoryPoint, 0, int(to.Sub(from).Hours()/24)+1)
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		evaluation, err := health.CalculateColonyHealth(evidence, day)
		if err != nil {
			return HealthHistoryResult{}, fmt.Errorf("statistics: calculate health history: %w", err)
		}
		points = append(points, HealthHistoryPoint{Date: day, Evaluation: evaluation})
	}

	markers := make([]HealthHistoryInspection, 0)
	for _, fact := range facts {
		if !fact.InspectedAt.Before(from) && !fact.InspectedAt.After(to) {
			markers = append(markers, HealthHistoryInspection{ID: fact.ID, Date: fact.InspectedAt, Type: fact.Type})
		}
	}
	sort.Slice(markers, func(i, j int) bool {
		if !markers[i].Date.Equal(markers[j].Date) {
			return markers[i].Date.Before(markers[j].Date)
		}
		return markers[i].ID.String() < markers[j].ID.String()
	})
	return HealthHistoryResult{From: from, To: to, Points: points, Inspections: markers}, nil
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
// first, capped at limit. Unlike Overview/InspectionStats, it fetches
// only the inspections it needs (via InspectionLister.ListRecent)
// instead of paging through the caller's entire inspection history -
// apiaries and hives are still fetched in full, but those are typically
// few for a single beekeeper (see fetchAll's doc comment).
func (s *Service) RecentActivity(ctx context.Context, accessToken string, limit int) ([]domainstats.ActivityItem, error) {
	apiaries, err := s.apiaries.ListAll(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("statistics: list apiaries: %w", err)
	}
	hives, err := s.hives.ListAll(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("statistics: list hives: %w", err)
	}
	inspections, err := s.inspections.ListRecent(ctx, accessToken, limit)
	if err != nil {
		return nil, fmt.Errorf("statistics: list recent inspections: %w", err)
	}
	return domainstats.ComputeRecentActivity(apiaries, hives, inspections, limit), nil
}

// HarvestStats returns the Dashboard's harvest-focused section by using
// harvest-service's user-scoped global list. It deliberately does not
// fetch hives or fan out one request per hive.
func (s *Service) HarvestStats(ctx context.Context, accessToken string) (domainstats.HarvestStats, error) {
	harvests, err := s.harvests.ListAll(ctx, accessToken)
	if err != nil {
		return domainstats.HarvestStats{}, fmt.Errorf("statistics: list harvests: %w", err)
	}

	return domainstats.ComputeHarvestStats(harvests), nil
}

// NeedsAttention returns the Dashboard's actionable "Needs Attention"
// section. It only needs apiaries and hives (this service's own data)
// plus inspection-service's hive-status (latest InspectedAt per hive
// and the configured threshold) - never the caller's full inspection
// history, unlike Overview/InspectionStats.
func (s *Service) NeedsAttention(ctx context.Context, accessToken string) (domainstats.NeedsAttention, error) {
	apiaries, err := s.apiaries.ListAll(ctx, accessToken)
	if err != nil {
		return domainstats.NeedsAttention{}, fmt.Errorf("statistics: list apiaries: %w", err)
	}
	hives, err := s.hives.ListAll(ctx, accessToken)
	if err != nil {
		return domainstats.NeedsAttention{}, fmt.Errorf("statistics: list hives: %w", err)
	}
	latestByHive, thresholdDays, err := s.inspections.HiveInspectionStatus(ctx, accessToken)
	if err != nil {
		return domainstats.NeedsAttention{}, fmt.Errorf("statistics: get hive inspection status: %w", err)
	}

	return domainstats.ComputeNeedsAttention(apiaries, hives, latestByHive, thresholdDays, time.Now().UTC()), nil
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
