package harvestclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sbezhuk/beebase-statistics-service/internal/platform/harvestclient"
)

type fakePage struct {
	Items      []map[string]any `json:"items"`
	Pagination map[string]any   `json:"pagination"`
}

func TestClient_ListAllForHives_SingleHiveSinglePage(t *testing.T) {
	id := uuid.New()
	hiveID := uuid.New()
	harvestedAt := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good-token" {
			t.Errorf("Authorization header = %q, want forwarded bearer token", r.Header.Get("Authorization"))
		}
		wantPath := fmt.Sprintf("/api/v1/hives/%s/harvest", hiveID)
		if r.URL.Path != wantPath {
			t.Errorf("path = %q, want %q", r.URL.Path, wantPath)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakePage{
			Items: []map[string]any{
				{
					"id":           id.String(),
					"product":      "HONEY",
					"amount":       2.5,
					"unit":         "kg",
					"harvested_at": harvestedAt.Format(time.RFC3339),
				},
			},
			Pagination: map[string]any{"total_pages": 1},
		})
	}))
	defer srv.Close()

	client := harvestclient.New(srv.URL)
	got, err := client.ListAllForHives(context.Background(), "good-token", []uuid.UUID{hiveID})
	if err != nil {
		t.Fatalf("ListAllForHives: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].ID != id || got[0].Product != "HONEY" || got[0].Unit != "kg" || got[0].Amount != 2.5 {
		t.Errorf("got[0] = %+v, want id/product/unit/amount to match", got[0])
	}
	if !got[0].HarvestedAt.Equal(harvestedAt) {
		t.Errorf("HarvestedAt = %v, want %v", got[0].HarvestedAt, harvestedAt)
	}
}

func TestClient_ListAllForHives_FansOutAcrossHives(t *testing.T) {
	hiveA := uuid.New()
	hiveB := uuid.New()
	requestedPaths := map[string]int{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths[r.URL.Path]++

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakePage{
			Items: []map[string]any{
				{
					"id":           uuid.New().String(),
					"product":      "WAX",
					"amount":       10.0,
					"unit":         "g",
					"harvested_at": time.Now().UTC().Format(time.RFC3339),
				},
			},
			Pagination: map[string]any{"total_pages": 1},
		})
	}))
	defer srv.Close()

	client := harvestclient.New(srv.URL)
	got, err := client.ListAllForHives(context.Background(), "token", []uuid.UUID{hiveA, hiveB})
	if err != nil {
		t.Fatalf("ListAllForHives: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2 (one per hive)", len(got))
	}
	if requestedPaths[fmt.Sprintf("/api/v1/hives/%s/harvest", hiveA)] != 1 {
		t.Errorf("hive A requested %d times, want 1", requestedPaths[fmt.Sprintf("/api/v1/hives/%s/harvest", hiveA)])
	}
	if requestedPaths[fmt.Sprintf("/api/v1/hives/%s/harvest", hiveB)] != 1 {
		t.Errorf("hive B requested %d times, want 1", requestedPaths[fmt.Sprintf("/api/v1/hives/%s/harvest", hiveB)])
	}
}

func TestClient_ListAllForHives_NoHivesMakesNoRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	client := harvestclient.New(srv.URL)
	got, err := client.ListAllForHives(context.Background(), "token", nil)
	if err != nil {
		t.Fatalf("ListAllForHives: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
}

func TestClient_ListAllForHives_UnexpectedStatusFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := harvestclient.New(srv.URL)
	if _, err := client.ListAllForHives(context.Background(), "token", []uuid.UUID{uuid.New()}); err == nil {
		t.Fatal("ListAllForHives against a 500: got nil error, want a failure")
	}
}

func TestClient_ListAllForHives_UnreachableServer(t *testing.T) {
	client := harvestclient.New("http://127.0.0.1:1") // nothing listens here
	if _, err := client.ListAllForHives(context.Background(), "token", []uuid.UUID{uuid.New()}); err == nil {
		t.Fatal("ListAllForHives against an unreachable server: got nil error, want a failure")
	}
}
