package toss

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// TestDoGetEmptyBodyOnlyTreatedAsSuccessFor204 covers a review finding:
// doRequest previously treated ANY empty response body on a non-error status
// as success with a zero-value result, not just the one documented 204 (from
// CancelConditionalOrder). A truncated/malformed 200 response would then
// silently report a zero-value result instead of erroring.
func TestDoGetEmptyBodyOnlyTreatedAsSuccessFor204(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Body intentionally left empty to simulate a truncated/malformed
		// 200 response.
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	_, err := doGet[BuyingPowerResponse](context.Background(), client, "toss test", "/whatever", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("expected a json-unmarshal error for an empty 200 body, got %v", err)
	}
}

func TestDoGetEmptyBodyOn204IsSuccess(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	if _, err := doGet[BuyingPowerResponse](context.Background(), client, "toss test", "/whatever", nil, nil); err != nil {
		t.Fatalf("204 should be treated as success, got %v", err)
	}
}
