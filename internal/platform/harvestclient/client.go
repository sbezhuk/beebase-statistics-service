// Package harvestclient implements
// application/statistics.HarvestLister against the real harvest-service
// over HTTP.
package harvestclient

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
	// pageLimit is harvest-service's own pagination.MaxLimit - the
	// largest page size it accepts, so ListAllForHives pages through as
	// few requests as possible.
	pageLimit = 100
)

// Client lists every harvest record across a set of hives from
// harvest-service, forwarding the caller's own access token so
// harvest-service can verify each hive belongs to whoever presented it -
// the same check it runs for a direct request from the caller.
//
// Unlike apiary/hive/inspection-service, harvest-service has no endpoint
// that lists every harvest a caller owns in one call - only GET
// /hives/{hiveID}/harvest, scoped to a single hive - so this client fans
// out across the given hive IDs instead of a duplicated, dashboard-only
// data-access path.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a Client that calls harvest-service at baseURL (e.g.
// "http://harvest-service:8080").
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: requestTimeout},
	}
}

type harvestItem struct {
	ID          uuid.UUID `json:"id"`
	Product     string    `json:"product"`
	Amount      float64   `json:"amount"`
	Unit        string    `json:"unit"`
	HarvestedAt time.Time `json:"harvested_at"`
}

type harvestPage struct {
	Items      []harvestItem `json:"items"`
	Pagination struct {
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

// ListAllForHives implements application/statistics.HarvestLister by
// paging through GET /api/v1/hives/{hiveID}/harvest for every hive in
// hiveIDs.
func (c *Client) ListAllForHives(ctx context.Context, accessToken string, hiveIDs []uuid.UUID) ([]domainstats.Harvest, error) {
	var out []domainstats.Harvest

	for _, hiveID := range hiveIDs {
		for page := 1; ; page++ {
			body, err := c.fetchPage(ctx, accessToken, hiveID, page)
			if err != nil {
				return nil, err
			}

			for _, item := range body.Items {
				out = append(out, domainstats.Harvest{
					ID:          item.ID,
					Product:     item.Product,
					Amount:      item.Amount,
					Unit:        item.Unit,
					HarvestedAt: item.HarvestedAt,
				})
			}

			if page >= body.Pagination.TotalPages || len(body.Items) == 0 {
				break
			}
		}
	}

	return out, nil
}

func (c *Client) fetchPage(ctx context.Context, accessToken string, hiveID uuid.UUID, page int) (harvestPage, error) {
	u := fmt.Sprintf("%s/api/v1/hives/%s/harvest?page=%d&limit=%d", c.baseURL, hiveID, page, pageLimit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return harvestPage{}, fmt.Errorf("harvestclient: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return harvestPage{}, fmt.Errorf("harvestclient: call harvest-service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return harvestPage{}, fmt.Errorf("harvestclient: unexpected status %d from harvest-service", resp.StatusCode)
	}

	var respBody harvestPage
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return harvestPage{}, fmt.Errorf("harvestclient: decode response: %w", err)
	}

	return respBody, nil
}
