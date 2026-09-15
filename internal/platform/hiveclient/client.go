// Package hiveclient implements application/statistics.HiveLister against
// the real hive-service over HTTP.
package hiveclient

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
	// pageLimit is hive-service's own pagination.MaxLimit - the largest
	// page size it accepts, so ListAll pages through as few requests as
	// possible.
	pageLimit = 100
)

// Client lists every hive the caller owns from hive-service, forwarding
// the caller's own access token so hive-service scopes the result to the
// same user this service verified via their token.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a Client that calls hive-service at baseURL (e.g.
// "http://hive-service:8080").
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: requestTimeout},
	}
}

type hiveItem struct {
	ID       uuid.UUID `json:"id"`
	ApiaryID uuid.UUID `json:"apiaryId"`
	Name     string    `json:"name"`
}

type hivePage struct {
	Items      []hiveItem `json:"items"`
	Pagination struct {
		TotalPages int `json:"totalPages"`
	} `json:"pagination"`
}

// ListAll implements application/statistics.HiveLister by paging through
// GET /api/v1/hives until every hive the caller owns has been collected.
func (c *Client) ListAll(ctx context.Context, accessToken string) ([]domainstats.Hive, error) {
	var out []domainstats.Hive

	for page := 1; ; page++ {
		body, err := c.fetchPage(ctx, accessToken, page)
		if err != nil {
			return nil, err
		}

		for _, item := range body.Items {
			out = append(out, domainstats.Hive{ID: item.ID, ApiaryID: item.ApiaryID, Name: item.Name})
		}

		if page >= body.Pagination.TotalPages || len(body.Items) == 0 {
			break
		}
	}

	return out, nil
}

func (c *Client) fetchPage(ctx context.Context, accessToken string, page int) (hivePage, error) {
	u := fmt.Sprintf("%s/api/v1/hives?page=%d&limit=%d", c.baseURL, page, pageLimit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return hivePage{}, fmt.Errorf("hiveclient: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return hivePage{}, fmt.Errorf("hiveclient: call hive-service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return hivePage{}, fmt.Errorf("hiveclient: unexpected status %d from hive-service", resp.StatusCode)
	}

	var body hivePage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return hivePage{}, fmt.Errorf("hiveclient: decode response: %w", err)
	}

	return body, nil
}
