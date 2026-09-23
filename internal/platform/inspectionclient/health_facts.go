package inspectionclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/sbezhuk/beebase-health/health"

	appstatistics "github.com/sbezhuk/beebase-statistics-service/internal/application/statistics"
)

type healthFactsInspection struct {
	ID          uuid.UUID              `json:"id"`
	HiveID      uuid.UUID              `json:"hiveId"`
	InspectedAt string                 `json:"inspectedAt"`
	Type        health.Type            `json:"type"`
	Assessment  *healthFactsAssessment `json:"assessment,omitempty"`
}

type healthFactsAssessment struct {
	Version                int                            `json:"version"`
	ColonyStrength         *health.ColonyStrength         `json:"colonyStrength,omitempty"`
	QueenStatus            *health.QueenStatus            `json:"queenStatus,omitempty"`
	BroodStatus            *health.BroodStatus            `json:"broodStatus,omitempty"`
	FoodStores             *health.FoodStores             `json:"foodStores,omitempty"`
	HealthConcerns         *health.HealthConcerns         `json:"healthConcerns,omitempty"`
	QueenObserved          *health.QueenObserved          `json:"queenObserved,omitempty"`
	EggsObserved           *health.EggsObserved           `json:"eggsObserved,omitempty"`
	QueenCells             *health.QueenCells             `json:"queenCells,omitempty"`
	QueenCondition         *health.QueenCondition         `json:"queenCondition,omitempty"`
	BroodAmount            *health.BroodAmount            `json:"broodAmount,omitempty"`
	BroodPattern           *health.BroodPattern           `json:"broodPattern,omitempty"`
	BroodStages            *[]health.BroodStage           `json:"broodStages,omitempty"`
	BroodConcerns          *health.BroodConcerns          `json:"broodConcerns,omitempty"`
	HealthOverallCondition *health.HealthOverallCondition `json:"healthOverallCondition,omitempty"`
	PestSigns              *[]health.PestSign             `json:"pestSigns,omitempty"`
	HealthWarningSigns     *[]health.HealthWarningSign    `json:"healthWarningSigns,omitempty"`
	HealthConcernLevel     *health.HealthConcernLevel     `json:"healthConcernLevel,omitempty"`
	FeedingNeed            *health.FeedingNeed            `json:"feedingNeed,omitempty"`
	FeedingPerformed       *health.FeedingPerformed       `json:"feedingPerformed,omitempty"`
	FeedTypes              *[]health.FeedType             `json:"feedTypes,omitempty"`
	Season                 *health.SeasonalPhase          `json:"season,omitempty"`
	SeasonalStoreReadiness *health.SeasonalStoreReadiness `json:"seasonalStoreReadiness,omitempty"`
	SeasonalReadiness      *health.SeasonalReadiness      `json:"seasonalReadiness,omitempty"`
	SeasonalConcerns       *[]health.SeasonalConcern      `json:"seasonalConcerns,omitempty"`
}

type healthFactsResponse struct {
	HiveID      uuid.UUID               `json:"hiveId"`
	Inspections []healthFactsInspection `json:"inspections"`
}

func NewWithInternalToken(baseURL, internalToken string) *Client {
	client := New(baseURL)
	client.internalToken = internalToken
	return client
}

func (c *Client) ListHealthFacts(ctx context.Context, hiveID uuid.UUID, to time.Time) ([]appstatistics.HealthFact, error) {
	base, err := url.Parse(fmt.Sprintf("%s/internal/api/v1/hives/%s/health-facts", c.baseURL, hiveID))
	if err != nil {
		return nil, fmt.Errorf("inspectionclient: build health facts request: %w", err)
	}
	query := base.Query()
	query.Set("to", to.Format("2006-01-02"))
	base.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("inspectionclient: build health facts request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.internalToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inspectionclient: call internal health facts: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inspectionclient: unexpected health facts status %d", resp.StatusCode)
	}
	var body healthFactsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("inspectionclient: decode health facts: %w", err)
	}
	out := make([]appstatistics.HealthFact, len(body.Inspections))
	for i, item := range body.Inspections {
		date, err := time.Parse("2006-01-02", item.InspectedAt)
		if err != nil {
			return nil, fmt.Errorf("inspectionclient: parse health fact date: %w", err)
		}
		out[i] = appstatistics.HealthFact{ID: item.ID, HiveID: item.HiveID, InspectedAt: date, Type: item.Type, Assessment: convertAssessment(item.Assessment)}
	}
	return out, nil
}

func convertAssessment(input *healthFactsAssessment) *appstatistics.HealthFactAssessment {
	if input == nil {
		return nil
	}
	return &appstatistics.HealthFactAssessment{
		Version: input.Version, ColonyStrength: input.ColonyStrength, QueenStatus: input.QueenStatus,
		BroodStatus: input.BroodStatus, FoodStores: input.FoodStores, HealthConcerns: input.HealthConcerns,
		QueenObserved: input.QueenObserved, EggsObserved: input.EggsObserved, QueenCells: input.QueenCells,
		QueenCondition: input.QueenCondition, BroodAmount: input.BroodAmount, BroodPattern: input.BroodPattern,
		BroodStages: input.BroodStages, BroodConcerns: input.BroodConcerns, HealthOverallCondition: input.HealthOverallCondition,
		PestSigns: input.PestSigns, HealthWarningSigns: input.HealthWarningSigns, HealthConcernLevel: input.HealthConcernLevel,
		FeedingNeed: input.FeedingNeed, FeedingPerformed: input.FeedingPerformed, FeedTypes: input.FeedTypes,
		Season: input.Season, SeasonalStoreReadiness: input.SeasonalStoreReadiness, SeasonalReadiness: input.SeasonalReadiness,
		SeasonalConcerns: input.SeasonalConcerns,
	}
}
