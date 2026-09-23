package hiveclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	appstatistics "github.com/sbezhuk/beebase-statistics-service/internal/application/statistics"
)

func (c *Client) Verify(ctx context.Context, accessToken string, hiveID uuid.UUID) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/hives/%s", c.baseURL, hiveID), nil)
	if err != nil {
		return fmt.Errorf("hiveclient: build verify request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("hiveclient: verify hive: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
		var ignored json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&ignored); err != nil {
			return fmt.Errorf("hiveclient: decode verify response: %w", err)
		}
		return nil
	case http.StatusNotFound:
		return appstatistics.ErrHiveNotFound
	default:
		return fmt.Errorf("hiveclient: unexpected verify status %d", resp.StatusCode)
	}
}
