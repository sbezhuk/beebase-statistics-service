package statistics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestParseLimit locks down Activity's existing ?limit= validation and
// default, unchanged by the ListRecent efficiency work: still 1-50,
// still defaulting to 10 when omitted.
func TestParseLimit(t *testing.T) {
	h := &Handler{}

	tests := []struct {
		name      string
		query     string
		wantLimit int
		wantOK    bool
	}{
		{name: "default when omitted", query: "", wantLimit: defaultActivityLimit, wantOK: true},
		{name: "compact dashboard-sized limit", query: "limit=5", wantLimit: 5, wantOK: true},
		{name: "minimum bound", query: "limit=1", wantLimit: 1, wantOK: true},
		{name: "maximum bound", query: "limit=50", wantLimit: maxActivityLimit, wantOK: true},
		{name: "zero rejected", query: "limit=0", wantOK: false},
		{name: "above max rejected", query: "limit=51", wantOK: false},
		{name: "negative rejected", query: "limit=-1", wantOK: false},
		{name: "non-numeric rejected", query: "limit=abc", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/statistics/activity?"+tt.query, nil)
			rec := httptest.NewRecorder()

			got, ok := h.parseLimit(rec, r)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
				}
				return
			}
			if got != tt.wantLimit {
				t.Errorf("limit = %d, want %d", got, tt.wantLimit)
			}
		})
	}
}
