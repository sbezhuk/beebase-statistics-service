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

type fakePage struct {
	Items      []map[string]any `json:"items"`
	Pagination map[string]any   `json:"pagination"`
}

func TestClient_ListAll_SinglePage(t *testing.T) {
	id := uuid.New()
	hiveID := uuid.New()
	inspectedAt := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good-token" {
			t.Errorf("Authorization header = %q, want forwarded bearer token", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/v1/inspections" {
			t.Errorf("path = %q, want /api/v1/inspections", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakePage{
			Items: []map[string]any{
				{
					"id":           id.String(),
					"hive_id":      hiveID.String(),
					"inspected_at": inspectedAt.Format(time.RFC3339),
					"notes":        "queen seen",
				},
			},
			Pagination: map[string]any{"total_pages": 1},
		})
	}))
	defer srv.Close()

	client := inspectionclient.New(srv.URL)
	got, err := client.ListAll(context.Background(), "good-token")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].ID != id || got[0].HiveID != hiveID || got[0].Notes != "queen seen" {
		t.Errorf("got[0] = %+v, want id/hive_id/notes to match", got[0])
	}
	if !got[0].InspectedAt.Equal(inspectedAt) {
		t.Errorf("InspectedAt = %v, want %v", got[0].InspectedAt, inspectedAt)
	}
}

func TestClient_ListAll_UnexpectedStatusFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := inspectionclient.New(srv.URL)
	if _, err := client.ListAll(context.Background(), "token"); err == nil {
		t.Fatal("ListAll against a 500: got nil error, want a failure")
	}
}

func TestClient_ListAll_UnreachableServer(t *testing.T) {
	client := inspectionclient.New("http://127.0.0.1:1") // nothing listens here
	if _, err := client.ListAll(context.Background(), "token"); err == nil {
		t.Fatal("ListAll against an unreachable server: got nil error, want a failure")
	}
}
