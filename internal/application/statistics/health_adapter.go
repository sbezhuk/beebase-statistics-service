package statistics

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/sbezhuk/beebase-health/health"
)

func healthInspection(input HealthFact) (health.Inspection, error) {
	if input.ID == uuid.Nil || input.HiveID == uuid.Nil {
		return health.Inspection{}, fmt.Errorf("inspection and hive IDs are required")
	}
	if input.InspectedAt.IsZero() {
		return health.Inspection{}, fmt.Errorf("inspected date is required")
	}
	return health.Inspection{
		ID: input.ID, HiveID: input.HiveID, InspectedAt: input.InspectedAt,
		Type: input.Type, Assessment: healthAssessment(input.Assessment),
	}, nil
}

func healthAssessment(input *HealthFactAssessment) *health.Assessment {
	if input == nil {
		return nil
	}
	return &health.Assessment{
		Version:        input.Version,
		ColonyStrength: input.ColonyStrength, QueenStatus: input.QueenStatus,
		BroodStatus: input.BroodStatus, FoodStores: input.FoodStores,
		HealthConcerns: input.HealthConcerns, QueenObserved: input.QueenObserved,
		EggsObserved: input.EggsObserved, QueenCells: input.QueenCells,
		QueenCondition: input.QueenCondition, BroodAmount: input.BroodAmount,
		BroodPattern: input.BroodPattern, BroodStages: input.BroodStages,
		BroodConcerns: input.BroodConcerns, HealthOverallCondition: input.HealthOverallCondition,
		PestSigns: input.PestSigns, HealthWarningSigns: input.HealthWarningSigns,
		HealthConcernLevel: input.HealthConcernLevel, FeedingNeed: input.FeedingNeed,
		FeedingPerformed: input.FeedingPerformed, FeedTypes: input.FeedTypes,
		Season: input.Season, SeasonalStoreReadiness: input.SeasonalStoreReadiness,
		SeasonalReadiness: input.SeasonalReadiness, SeasonalConcerns: input.SeasonalConcerns,
	}
}
