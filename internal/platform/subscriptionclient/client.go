package subscriptionclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const requestTimeout = 5 * time.Second

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: requestTimeout}}
}

func (c *Client) GetEntitlement(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/subscription", c.baseURL), nil)
	if err != nil {
		return "", fmt.Errorf("subscriptionclient: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("subscriptionclient: call subscription-service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("subscriptionclient: unexpected status %d from subscription-service", resp.StatusCode)
	}
	var body struct {
		Entitlement string `json:"entitlement"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("subscriptionclient: decode response: %w", err)
	}
	return body.Entitlement, nil
}
