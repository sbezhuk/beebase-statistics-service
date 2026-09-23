package inspectionclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sbezhuk/beebase-statistics-service/internal/platform/inspectionclient"
)

func TestClient_ListHealthFactsUsesInternalTokenAndDateOnly(t *testing.T) {
	hiveID := uuid.New()
	inspectionID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/api/v1/hives/"+hiveID.String()+"/health-facts" || r.URL.Query().Get("to") != "2026-09-12" {
			t.Errorf("request path/query = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer internal-secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hiveId": hiveID,
			"inspections": []any{map[string]any{
				"id": inspectionID, "hiveId": hiveID, "inspectedAt": "2026-09-01", "type": "ROUTINE",
				"assessment": map[string]any{"version": 1, "broodStages": []any{}, "pestSigns": nil},
			}},
		})
	}))
	defer server.Close()

	client := inspectionclient.NewWithInternalToken(server.URL, "internal-secret")
	got, err := client.ListHealthFacts(context.Background(), hiveID, time.Date(2026, 9, 12, 23, 0, 0, 0, time.FixedZone("x", 3600)))
	if err != nil {
		t.Fatalf("ListHealthFacts: %v", err)
	}
	if len(got) != 1 || got[0].ID != inspectionID || got[0].InspectedAt.Format("2006-01-02") != "2026-09-01" {
		t.Fatalf("facts = %+v", got)
	}
	if got[0].Assessment == nil || got[0].Assessment.BroodStages == nil || len(*got[0].Assessment.BroodStages) != 0 || got[0].Assessment.PestSigns != nil {
		t.Fatalf("nil/empty assessment slices were not preserved: %+v", got[0].Assessment)
	}
}
