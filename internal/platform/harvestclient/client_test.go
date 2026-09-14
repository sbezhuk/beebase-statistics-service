package harvestclient_test

import (
	"context"
	"encoding/json"
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

func TestClient_ListAll_SinglePage(t *testing.T) {
	id := uuid.New()
	harvestedAt := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good-token" {
			t.Errorf("Authorization header = %q, want forwarded bearer token", r.Header.Get("Authorization"))
		}
		wantPath := "/api/v1/harvests"
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
	got, err := client.ListAll(context.Background(), "good-token")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
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

func TestClient_ListAll_PagesGlobalEndpoint(t *testing.T) {
	requestedPages := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/harvests" {
			t.Errorf("path = %q, want global harvest path", r.URL.Path)
		}
		requestedPages++

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
	got, err := client.ListAll(context.Background(), "token")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != 1 || requestedPages != 1 {
		t.Fatalf("got %d records in %d requests, want 1 in 1 request", len(got), requestedPages)
	}
}

func TestClient_ListAll_EmptyPageMakesOneRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/harvests" {
			t.Errorf("unexpected request to %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(fakePage{Pagination: map[string]any{"total_pages": 1}})
	}))
	defer srv.Close()

	client := harvestclient.New(srv.URL)
	got, err := client.ListAll(context.Background(), "token")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
}

func TestClient_ListAll_UnexpectedStatusFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := harvestclient.New(srv.URL)
	if _, err := client.ListAll(context.Background(), "token"); err == nil {
		t.Fatal("ListAll against a 500: got nil error, want a failure")
	}
}

func TestClient_ListAll_UnreachableServer(t *testing.T) {
	client := harvestclient.New("http://127.0.0.1:1") // nothing listens here
	if _, err := client.ListAll(context.Background(), "token"); err == nil {
		t.Fatal("ListAll against an unreachable server: got nil error, want a failure")
	}
}
