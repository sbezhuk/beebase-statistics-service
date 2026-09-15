package hiveclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sbezhuk/beebase-statistics-service/internal/platform/hiveclient"
)

type fakePage struct {
	Items      []map[string]any `json:"items"`
	Pagination map[string]any   `json:"pagination"`
}

func TestClient_ListAll_SinglePage(t *testing.T) {
	id := uuid.New()
	apiaryID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good-token" {
			t.Errorf("Authorization header = %q, want forwarded bearer token", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/v1/hives" {
			t.Errorf("path = %q, want /api/v1/hives", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakePage{
			Items: []map[string]any{
				{"id": id.String(), "apiaryId": apiaryID.String(), "name": "Hive 1"},
			},
			Pagination: map[string]any{"totalPages": 1},
		})
	}))
	defer srv.Close()

	client := hiveclient.New(srv.URL)
	got, err := client.ListAll(context.Background(), "good-token")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].ID != id || got[0].ApiaryID != apiaryID || got[0].Name != "Hive 1" {
		t.Errorf("got[0] = %+v, want {%s, %s, Hive 1}", got[0], id, apiaryID)
	}
}

func TestClient_ListAll_UnexpectedStatusFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := hiveclient.New(srv.URL)
	if _, err := client.ListAll(context.Background(), "token"); err == nil {
		t.Fatal("ListAll against a 500: got nil error, want a failure")
	}
}

func TestClient_ListAll_UnreachableServer(t *testing.T) {
	client := hiveclient.New("http://127.0.0.1:1") // nothing listens here
	if _, err := client.ListAll(context.Background(), "token"); err == nil {
		t.Fatal("ListAll against an unreachable server: got nil error, want a failure")
	}
}
