// Package inspectionclient implements
// application/statistics.InspectionLister (ListAll, ListRecent, and
// HiveInspectionStatus) against the real inspection-service over HTTP.
package inspectionclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	domainstats "github.com/sbezhuk/beebase-statistics-service/internal/domain/statistics"
)

const (
	requestTimeout = 5 * time.Second
	// pageLimit is inspection-service's own pagination.MaxLimit - the
	// largest page size it accepts, so ListAll pages through as few
	// requests as possible.
	pageLimit = 100
)

// Client lists every inspection the caller owns, across all of their
// hives, from inspection-service, forwarding the caller's own access
// token so inspection-service scopes the result to the same user this
// service verified via their token.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a Client that calls inspection-service at baseURL (e.g.
// "http://inspection-service:8080").
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: requestTimeout},
	}
}

type inspectionItem struct {
	ID          uuid.UUID `json:"id"`
	HiveID      uuid.UUID `json:"hive_id"`
	InspectedAt time.Time `json:"inspected_at"`
	Notes       string    `json:"notes"`
}

type inspectionPage struct {
	Items      []inspectionItem `json:"items"`
	Pagination struct {
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

// ListAll implements application/statistics.InspectionLister by paging
// through GET /api/v1/inspections until every inspection the caller owns
// has been collected.
func (c *Client) ListAll(ctx context.Context, accessToken string) ([]domainstats.Inspection, error) {
	var out []domainstats.Inspection

	for page := 1; ; page++ {
		body, err := c.fetchPage(ctx, accessToken, page, pageLimit)
		if err != nil {
			return nil, err
		}

		for _, item := range body.Items {
			out = append(out, domainstats.Inspection{
				ID:          item.ID,
				HiveID:      item.HiveID,
				InspectedAt: item.InspectedAt,
				Notes:       item.Notes,
			})
		}

		if page >= body.Pagination.TotalPages || len(body.Items) == 0 {
			break
		}
	}

	return out, nil
}

// ListRecent implements application/statistics.InspectionLister's bounded
// query. inspection-service's GET /api/v1/inspections has no "newest
// first" order of its own - its default order is InspectedAt ascending
// (oldest first, ties broken by id ascending), and its only other
// supported order is by creation date, not InspectedAt - so instead of
// paging through everything and sorting client-side, this fetches only
// the tail of that ascending order: one minimal probe request to learn
// the total record count, then the last limit-sized page (and, when the
// caller's total isn't a multiple of limit, the page before it too, to
// fill in the remainder), reversed to newest-first. At most 3 requests,
// each transferring at most limit rows, regardless of how many
// inspections the caller has.
func (c *Client) ListRecent(ctx context.Context, accessToken string, limit int) ([]domainstats.Inspection, error) {
	if limit <= 0 {
		return nil, nil
	}

	probe, err := c.fetchPage(ctx, accessToken, 1, 1)
	if err != nil {
		return nil, err
	}
	total := probe.Pagination.Total
	if total == 0 {
		return nil, nil
	}

	lastPage := (total + limit - 1) / limit

	last, err := c.fetchPage(ctx, accessToken, lastPage, limit)
	if err != nil {
		return nil, err
	}
	items := last.Items

	if len(items) < limit && lastPage > 1 {
		prev, err := c.fetchPage(ctx, accessToken, lastPage-1, limit)
		if err != nil {
			return nil, err
		}
		need := limit - len(items)
		start := len(prev.Items) - need
		if start < 0 {
			start = 0
		}
		items = append(append([]inspectionItem{}, prev.Items[start:]...), items...)
	}

	// items is ascending (oldest -> newest); reverse for newest-first.
	out := make([]domainstats.Inspection, len(items))
	for i, item := range items {
		out[len(items)-1-i] = domainstats.Inspection{
			ID:          item.ID,
			HiveID:      item.HiveID,
			InspectedAt: item.InspectedAt,
			Notes:       item.Notes,
		}
	}
	return out, nil
}

type hiveInspectionStatusItem struct {
	HiveID            uuid.UUID `json:"hive_id"`
	LatestInspectedAt time.Time `json:"latest_inspected_at"`
}

type hiveInspectionStatusResponse struct {
	ThresholdDays int                        `json:"threshold_days"`
	Hives         []hiveInspectionStatusItem `json:"hives"`
}

// HiveInspectionStatus implements application/statistics.InspectionLister
// by calling GET /api/v1/inspections/hive-status.
func (c *Client) HiveInspectionStatus(ctx context.Context, accessToken string) (map[uuid.UUID]time.Time, int, error) {
	u := fmt.Sprintf("%s/api/v1/inspections/hive-status", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("inspectionclient: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("inspectionclient: call inspection-service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("inspectionclient: unexpected status %d from inspection-service", resp.StatusCode)
	}

	var body hiveInspectionStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, 0, fmt.Errorf("inspectionclient: decode response: %w", err)
	}

	latestByHive := make(map[uuid.UUID]time.Time, len(body.Hives))
	for _, item := range body.Hives {
		latestByHive[item.HiveID] = item.LatestInspectedAt
	}

	return latestByHive, body.ThresholdDays, nil
}

func (c *Client) fetchPage(ctx context.Context, accessToken string, page, limit int) (inspectionPage, error) {
	u := fmt.Sprintf("%s/api/v1/inspections?page=%d&limit=%d", c.baseURL, page, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return inspectionPage{}, fmt.Errorf("inspectionclient: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return inspectionPage{}, fmt.Errorf("inspectionclient: call inspection-service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return inspectionPage{}, fmt.Errorf("inspectionclient: unexpected status %d from inspection-service", resp.StatusCode)
	}

	var body inspectionPage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return inspectionPage{}, fmt.Errorf("inspectionclient: decode response: %w", err)
	}

	return body, nil
}
