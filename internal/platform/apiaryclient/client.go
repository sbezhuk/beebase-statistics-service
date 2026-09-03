// Package apiaryclient implements application/statistics.ApiaryLister
// against the real apiary-service over HTTP.
package apiaryclient

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
	// pageLimit is apiary-service's own pagination.MaxLimit - the largest
	// page size it accepts, so ListAll pages through as few requests as
	// possible.
	pageLimit = 100
)

// Client lists every apiary the caller owns from apiary-service,
// forwarding the caller's own access token so apiary-service scopes the
// result to the same user this service verified via their token.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a Client that calls apiary-service at baseURL (e.g.
// "http://apiary-service:8080").
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: requestTimeout},
	}
}

type apiaryItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type apiaryPage struct {
	Items      []apiaryItem `json:"items"`
	Pagination struct {
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

// ListAll implements application/statistics.ApiaryLister by paging
// through GET /api/v1/apiaries until every apiary the caller owns has
// been collected.
func (c *Client) ListAll(ctx context.Context, accessToken string) ([]domainstats.Apiary, error) {
	var out []domainstats.Apiary

	for page := 1; ; page++ {
		body, err := c.fetchPage(ctx, accessToken, page)
		if err != nil {
			return nil, err
		}

		for _, item := range body.Items {
			out = append(out, domainstats.Apiary{ID: item.ID, Name: item.Name})
		}

		if page >= body.Pagination.TotalPages || len(body.Items) == 0 {
			break
		}
	}

	return out, nil
}

func (c *Client) fetchPage(ctx context.Context, accessToken string, page int) (apiaryPage, error) {
	u := fmt.Sprintf("%s/api/v1/apiaries?page=%d&limit=%d", c.baseURL, page, pageLimit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return apiaryPage{}, fmt.Errorf("apiaryclient: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return apiaryPage{}, fmt.Errorf("apiaryclient: call apiary-service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return apiaryPage{}, fmt.Errorf("apiaryclient: unexpected status %d from apiary-service", resp.StatusCode)
	}

	var body apiaryPage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return apiaryPage{}, fmt.Errorf("apiaryclient: decode response: %w", err)
	}

	return body, nil
}
