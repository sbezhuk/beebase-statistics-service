package statistics

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sbezhuk/beebase-health/health"
)

func TestHealthInspectionPreservesCanonicalAssessmentShape(t *testing.T) {
	emptyStages := []health.BroodStage{}
	emptyPests := []health.PestSign{}
	populatedWarnings := []health.HealthWarningSign{health.HealthWarningOther}
	fact := HealthFact{
		ID: uuid.New(), HiveID: uuid.New(), InspectedAt: time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC), Type: health.TypeSeasonal,
		Assessment: &HealthFactAssessment{
			Version:     health.AssessmentVersion,
			BroodStages: &emptyStages, PestSigns: &emptyPests, HealthWarningSigns: &populatedWarnings,
			SeasonalReadiness: ptr(health.SeasonalReadinessReady),
		},
	}

	got, err := healthInspection(fact)
	if err != nil {
		t.Fatalf("healthInspection: %v", err)
	}
	if got.ID != fact.ID || got.HiveID != fact.HiveID || !got.InspectedAt.Equal(fact.InspectedAt) || got.Type != fact.Type {
		t.Fatalf("identity/date/type = %+v", got)
	}
	if got.Assessment == nil || got.Assessment.BroodStages == nil || len(*got.Assessment.BroodStages) != 0 {
		t.Fatalf("empty BroodStages was not preserved")
	}
	if got.Assessment.PestSigns == nil || len(*got.Assessment.PestSigns) != 0 {
		t.Fatalf("empty PestSigns was not preserved")
	}
	if got.Assessment.HealthWarningSigns == nil || len(*got.Assessment.HealthWarningSigns) != 1 {
		t.Fatalf("populated HealthWarningSigns was not preserved")
	}
	if got.Assessment.SeasonalConcerns != nil {
		t.Fatalf("nil SeasonalConcerns became non-nil")
	}
}

func TestHealthInspectionSupportsAllSixTypes(t *testing.T) {
	for _, typ := range []health.Type{health.TypeRoutine, health.TypeQueen, health.TypeBrood, health.TypeHealth, health.TypeFeeding, health.TypeSeasonal} {
		t.Run(string(typ), func(t *testing.T) {
			got, err := healthInspection(HealthFact{ID: uuid.New(), HiveID: uuid.New(), InspectedAt: time.Now().UTC(), Type: typ})
			if err != nil || got.Type != typ {
				t.Fatalf("got type/error = %s/%v", got.Type, err)
			}
		})
	}
}

func ptr[T any](value T) *T { return &value }
