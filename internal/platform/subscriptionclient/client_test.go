package subscriptionclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sbezhuk/beebase-statistics-service/internal/platform/subscriptionclient"
)

func TestClient_GetEntitlementForwardsCallerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer caller-token" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"entitlement": "pro"})
	}))
	defer server.Close()
	got, err := subscriptionclient.New(server.URL).GetEntitlement(context.Background(), "caller-token")
	if err != nil || got != "pro" {
		t.Fatalf("entitlement/error = %q/%v", got, err)
	}
}
