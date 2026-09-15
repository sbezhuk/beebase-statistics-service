package apiaryclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/sbezhuk/beebase-statistics-service/internal/platform/apiaryclient"
)

type fakePage struct {
	Items      []map[string]any `json:"items"`
	Pagination map[string]any   `json:"pagination"`
}

func TestClient_ListAll_SinglePage(t *testing.T) {
	id := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good-token" {
			t.Errorf("Authorization header = %q, want forwarded bearer token", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/v1/apiaries" {
			t.Errorf("path = %q, want /api/v1/apiaries", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakePage{
			Items: []map[string]any{
				{"id": id.String(), "name": "Home Apiary"},
			},
			Pagination: map[string]any{"page": 1, "limit": 100, "total": 1, "totalPages": 1},
		})
	}))
	defer srv.Close()

	client := apiaryclient.New(srv.URL)
	got, err := client.ListAll(context.Background(), "good-token")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].ID != id || got[0].Name != "Home Apiary" {
		t.Errorf("got[0] = %+v, want {%s, Home Apiary}", got[0], id)
	}
}

func TestClient_ListAll_PagesThroughEverything(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	var requestedPages []int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			t.Fatalf("parse page query param: %v", err)
		}
		requestedPages = append(requestedPages, page)

		idx := page - 1
		var items []map[string]any
		if idx < len(ids) {
			items = append(items, map[string]any{"id": ids[idx].String(), "name": fmt.Sprintf("Apiary %d", idx)})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakePage{
			Items:      items,
			Pagination: map[string]any{"totalPages": len(ids)},
		})
	}))
	defer srv.Close()

	client := apiaryclient.New(srv.URL)
	got, err := client.ListAll(context.Background(), "token")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != len(ids) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(ids))
	}
	if len(requestedPages) != len(ids) {
		t.Fatalf("requested %d pages, want %d", len(requestedPages), len(ids))
	}
}

func TestClient_ListAll_UnexpectedStatusFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := apiaryclient.New(srv.URL)
	if _, err := client.ListAll(context.Background(), "token"); err == nil {
		t.Fatal("ListAll against a 500: got nil error, want a failure")
	}
}

func TestClient_ListAll_UnreachableServer(t *testing.T) {
	client := apiaryclient.New("http://127.0.0.1:1") // nothing listens here
	if _, err := client.ListAll(context.Background(), "token"); err == nil {
		t.Fatal("ListAll against an unreachable server: got nil error, want a failure")
	}
}
