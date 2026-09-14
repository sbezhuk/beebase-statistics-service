// Package harvestclient implements application/statistics.HarvestLister
// against the real harvest-service over HTTP.
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
	pageLimit      = 100
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: requestTimeout}}
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

// ListAll implements application/statistics.HarvestLister by paging through
// GET /api/v1/harvests until all records are loaded.
func (c *Client) ListAll(ctx context.Context, accessToken string) ([]domainstats.Harvest, error) {
	var out []domainstats.Harvest
	for page := 1; ; page++ {
		body, err := c.fetchPage(ctx, accessToken, page)
		if err != nil {
			return nil, err
		}
		for _, item := range body.Items {
			out = append(out, domainstats.Harvest{ID: item.ID, Product: item.Product, Amount: item.Amount, Unit: item.Unit, HarvestedAt: item.HarvestedAt})
		}
		if page >= body.Pagination.TotalPages || len(body.Items) == 0 {
			break
		}
	}
	return out, nil
}

func (c *Client) fetchPage(ctx context.Context, accessToken string, page int) (harvestPage, error) {
	return c.fetch(ctx, accessToken, fmt.Sprintf("%s/api/v1/harvests?page=%d&limit=%d", c.baseURL, page, pageLimit))
}

func (c *Client) fetch(ctx context.Context, accessToken, u string) (harvestPage, error) {
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
	var body harvestPage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return harvestPage{}, fmt.Errorf("harvestclient: decode response: %w", err)
	}
	return body, nil
}
