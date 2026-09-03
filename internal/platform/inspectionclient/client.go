// Package inspectionclient implements
// application/statistics.InspectionLister against the real
// inspection-service over HTTP.
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
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

// ListAll implements application/statistics.InspectionLister by paging
// through GET /api/v1/inspections until every inspection the caller owns
// has been collected.
func (c *Client) ListAll(ctx context.Context, accessToken string) ([]domainstats.Inspection, error) {
	var out []domainstats.Inspection

	for page := 1; ; page++ {
		body, err := c.fetchPage(ctx, accessToken, page)
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

func (c *Client) fetchPage(ctx context.Context, accessToken string, page int) (inspectionPage, error) {
	u := fmt.Sprintf("%s/api/v1/inspections?page=%d&limit=%d", c.baseURL, page, pageLimit)

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
