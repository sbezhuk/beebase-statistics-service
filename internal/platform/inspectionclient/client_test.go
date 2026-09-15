package inspectionclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
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
					"id":          id.String(),
					"hiveId":      hiveID.String(),
					"inspectedAt": inspectedAt.Format(time.RFC3339),
					"notes":       "queen seen",
				},
			},
			Pagination: map[string]any{"totalPages": 1},
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

// newInspectionServer starts a fake inspection-service holding total
// inspections, oldest to newest (item i was inspected i hours after a
// fixed base time, with notes "item-<i>"), and serves GET
// /api/v1/inspections honoring page/limit exactly like the real
// service's default order (InspectedAt ascending). It returns the
// server and a counter of how many requests it has received, so a test
// can assert ListRecent stays bounded regardless of total.
func newInspectionServer(t *testing.T, total int) (*httptest.Server, *int32) {
	t.Helper()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ids := make([]uuid.UUID, total)
	for i := range ids {
		ids[i] = uuid.New()
	}

	var requests int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		offset := (page - 1) * limit
		end := offset + limit
		if end > total {
			end = total
		}
		if offset > total {
			offset = total
		}

		items := []map[string]any{}
		for i := offset; i < end; i++ {
			items = append(items, map[string]any{
				"id":          ids[i].String(),
				"hiveId":      uuid.Nil.String(),
				"inspectedAt": base.Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
				"notes":       fmt.Sprintf("item-%d", i),
			})
		}

		totalPages := 0
		if total > 0 && limit > 0 {
			totalPages = int(math.Ceil(float64(total) / float64(limit)))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakePage{
			Items: items,
			Pagination: map[string]any{
				"total":      total,
				"totalPages": totalPages,
			},
		})
	}))
	t.Cleanup(srv.Close)

	return srv, &requests
}

func TestClient_ListRecent_FewerThanLimitReturnsEverythingNewestFirst(t *testing.T) {
	srv, _ := newInspectionServer(t, 3)

	client := inspectionclient.New(srv.URL)
	got, err := client.ListRecent(context.Background(), "token", 10)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3 (fewer than limit exist)", len(got))
	}
	if got[0].Notes != "item-2" || got[1].Notes != "item-1" || got[2].Notes != "item-0" {
		t.Errorf("order = [%s, %s, %s], want newest-first [item-2, item-1, item-0]", got[0].Notes, got[1].Notes, got[2].Notes)
	}
}

func TestClient_ListRecent_ExactPageBoundary(t *testing.T) {
	srv, requests := newInspectionServer(t, 20)

	client := inspectionclient.New(srv.URL)
	got, err := client.ListRecent(context.Background(), "token", 10)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("len(got) = %d, want 10", len(got))
	}
	if got[0].Notes != "item-19" || got[9].Notes != "item-10" {
		t.Errorf("got[0]/got[9] = %s/%s, want item-19/item-10", got[0].Notes, got[9].Notes)
	}
	// Probe (page=1,limit=1) + the single aligned last page: no merge needed.
	if n := atomic.LoadInt32(requests); n != 2 {
		t.Errorf("requests made = %d, want 2 (probe + one page, boundary is aligned)", n)
	}
}

func TestClient_ListRecent_UnalignedBoundaryMergesTwoPages(t *testing.T) {
	srv, requests := newInspectionServer(t, 23)

	client := inspectionclient.New(srv.URL)
	got, err := client.ListRecent(context.Background(), "token", 10)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("len(got) = %d, want 10", len(got))
	}
	// The last 10 of 23 items (indices 13..22), newest (item-22) first.
	if got[0].Notes != "item-22" || got[9].Notes != "item-13" {
		t.Errorf("got[0]/got[9] = %s/%s, want item-22/item-13", got[0].Notes, got[9].Notes)
	}
	for i := 0; i < len(got)-1; i++ {
		if !got[i].InspectedAt.After(got[i+1].InspectedAt) {
			t.Fatalf("got[%d..%d] not strictly newest-first: %v then %v", i, i+1, got[i].InspectedAt, got[i+1].InspectedAt)
		}
	}
	// Probe + last page (3 items) + previous page to fill the remaining 7.
	if n := atomic.LoadInt32(requests); n != 3 {
		t.Errorf("requests made = %d, want 3 (probe + two pages, boundary is unaligned)", n)
	}
}

func TestClient_ListRecent_LimitOneReturnsOnlyTheLatest(t *testing.T) {
	srv, _ := newInspectionServer(t, 5)

	client := inspectionclient.New(srv.URL)
	got, err := client.ListRecent(context.Background(), "token", 1)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Notes != "item-4" {
		t.Errorf("got[0].Notes = %q, want %q (the latest of 5)", got[0].Notes, "item-4")
	}
}

func TestClient_ListRecent_NoInspectionsMakesOneRequest(t *testing.T) {
	srv, requests := newInspectionServer(t, 0)

	client := inspectionclient.New(srv.URL)
	got, err := client.ListRecent(context.Background(), "token", 10)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
	if n := atomic.LoadInt32(requests); n != 1 {
		t.Errorf("requests made = %d, want 1 (just the probe)", n)
	}
}

func TestClient_ListRecent_ZeroOrNegativeLimitMakesNoRequests(t *testing.T) {
	srv, requests := newInspectionServer(t, 5)
	client := inspectionclient.New(srv.URL)

	for _, limit := range []int{0, -1} {
		got, err := client.ListRecent(context.Background(), "token", limit)
		if err != nil {
			t.Fatalf("ListRecent(limit=%d): %v", limit, err)
		}
		if len(got) != 0 {
			t.Fatalf("ListRecent(limit=%d): len(got) = %d, want 0", limit, len(got))
		}
	}
	if n := atomic.LoadInt32(requests); n != 0 {
		t.Errorf("requests made = %d, want 0", n)
	}
}

func TestClient_ListRecent_StaysBoundedRegardlessOfTotalSize(t *testing.T) {
	srv, requests := newInspectionServer(t, 1000)

	client := inspectionclient.New(srv.URL)
	got, err := client.ListRecent(context.Background(), "token", 5)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("len(got) = %d, want 5", len(got))
	}
	if got[0].Notes != "item-999" || got[4].Notes != "item-995" {
		t.Errorf("got[0]/got[4] = %s/%s, want item-999/item-995", got[0].Notes, got[4].Notes)
	}
	// However many of the 1000 inspections exist, ListRecent must never
	// page through more than a small, bounded number of requests.
	if n := atomic.LoadInt32(requests); n > 3 {
		t.Errorf("requests made = %d, want at most 3 regardless of total=1000", n)
	}
}

func TestClient_ListRecent_UnexpectedStatusFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := inspectionclient.New(srv.URL)
	if _, err := client.ListRecent(context.Background(), "token", 10); err == nil {
		t.Fatal("ListRecent against a 500: got nil error, want a failure")
	}
}

func TestClient_ListAll_AcceptsDateOnlyInspectedAt(t *testing.T) {
	id := uuid.New()
	hiveID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"id":"` + id.String() + `","hiveId":"` + hiveID.String() + `","inspectedAt":"2026-09-15","notes":"new"}],"pagination":{"totalPages":1}}`))
	}))
	defer srv.Close()

	client := inspectionclient.New(srv.URL)
	got, err := client.ListAll(context.Background(), "token")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != 1 || got[0].ID != id || got[0].HiveID != hiveID {
		t.Fatalf("got = %+v, want one inspection with matching ids", got)
	}
	want := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	if !got[0].InspectedAt.Equal(want) {
		t.Errorf("InspectedAt = %v, want %v", got[0].InspectedAt, want)
	}
}

func TestClient_ListRecent_UnreachableServer(t *testing.T) {
	client := inspectionclient.New("http://127.0.0.1:1") // nothing listens here
	if _, err := client.ListRecent(context.Background(), "token", 10); err == nil {
		t.Fatal("ListRecent against an unreachable server: got nil error, want a failure")
	}
}

func TestClient_HiveInspectionStatus_Success(t *testing.T) {
	hiveID := uuid.New()
	latest := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good-token" {
			t.Errorf("Authorization header = %q, want forwarded bearer token", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/v1/inspections/hive-status" {
			t.Errorf("path = %q, want /api/v1/inspections/hive-status", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"thresholdDays": 14,
			"hives": []map[string]any{
				{"hiveId": hiveID.String(), "latestInspectedAt": latest.Format(time.RFC3339)},
			},
		})
	}))
	defer srv.Close()

	client := inspectionclient.New(srv.URL)
	latestByHive, thresholdDays, err := client.HiveInspectionStatus(context.Background(), "good-token")
	if err != nil {
		t.Fatalf("HiveInspectionStatus: %v", err)
	}
	if thresholdDays != 14 {
		t.Errorf("thresholdDays = %d, want 14", thresholdDays)
	}
	got, ok := latestByHive[hiveID]
	if !ok || !got.Equal(latest) {
		t.Errorf("latestByHive[hiveID] = %v, ok=%v, want %v", got, ok, latest)
	}
}

func TestClient_HiveInspectionStatus_EmptyHivesYieldsEmptyMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"thresholdDays": 14, "hives": []any{}})
	}))
	defer srv.Close()

	client := inspectionclient.New(srv.URL)
	latestByHive, thresholdDays, err := client.HiveInspectionStatus(context.Background(), "token")
	if err != nil {
		t.Fatalf("HiveInspectionStatus: %v", err)
	}
	if len(latestByHive) != 0 {
		t.Errorf("latestByHive = %+v, want empty", latestByHive)
	}
	if thresholdDays != 14 {
		t.Errorf("thresholdDays = %d, want 14", thresholdDays)
	}
}

func TestClient_HiveInspectionStatus_UnexpectedStatusFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := inspectionclient.New(srv.URL)
	if _, _, err := client.HiveInspectionStatus(context.Background(), "token"); err == nil {
		t.Fatal("HiveInspectionStatus against a 500: got nil error, want a failure")
	}
}

func TestClient_HiveInspectionStatus_UnreachableServer(t *testing.T) {
	client := inspectionclient.New("http://127.0.0.1:1") // nothing listens here
	if _, _, err := client.HiveInspectionStatus(context.Background(), "token"); err == nil {
		t.Fatal("HiveInspectionStatus against an unreachable server: got nil error, want a failure")
	}
}
