package kis

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// retryServer returns an httptest server that answers the auth refresh endpoint
// and fails the first resource call with EGW00123, then succeeds. okBody is the
// JSON returned on the successful retry.
func retryServer(t *testing.T, okBody string) (*httptest.Server, *TokenManager) {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/oauth2/token") {
			fmt.Fprintf(w, `{"access_token":"refreshed","expires_in":86400}`)
			return
		}
		call++
		if call == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"msg_cd":"EGW00123","msg1":"token expired"}`)
			return
		}
		fmt.Fprint(w, okBody)
	}))
	t.Cleanup(srv.Close)

	store := &MemoryTokenStore{}
	_ = store.Save("old_token", time.Now().Add(24*time.Hour))
	auth := &AuthClient{HTTPClient: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s"}
	return srv, NewTokenManager(store, auth, time.Minute)
}

func TestPostWithRetryTokenExpiry(t *testing.T) {
	srv, mgr := retryServer(t, `{"rt_cd":"0","result":"posted"}`)
	body, err := postWithRetry(context.Background(), srv.Client(), srv.URL+"/order", map[string]string{"PDNO": "005930"},
		BuildHeaders("old_token", "k", "s", "TRID", "P"),
		mgr, "k", "s", "TRID", "P")
	if err != nil {
		t.Fatalf("postWithRetry retry: %v", err)
	}
	if !strings.Contains(string(body), "posted") {
		t.Errorf("body = %q, want 'posted' after retry", body)
	}
}

func TestPostWithRetryHTTP400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"bad request"}`)
	}))
	t.Cleanup(srv.Close)
	mgr := makeManager(t, "tok")
	_, err := postWithRetry(context.Background(), srv.Client(), srv.URL+"/order", map[string]string{"x": "1"},
		BuildHeaders("tok", "k", "s", "TRID", "P"),
		mgr, "k", "s", "TRID", "P")
	if err == nil {
		t.Fatal("expected error on HTTP 400")
	}
}

func TestGetWithRetryFullTokenExpiry(t *testing.T) {
	srv, mgr := retryServer(t, `{"result":"full_after_retry"}`)
	body, hdrs, err := GetWithRetryFull(srv.Client(), srv.URL+"/balance", map[string]string{"q": "1"},
		BuildHeaders("old_token", "k", "s", "TRID", "P"),
		mgr, "k", "s", "TRID", "P")
	if err != nil {
		t.Fatalf("GetWithRetryFull retry: %v", err)
	}
	if !strings.Contains(string(body), "full_after_retry") {
		t.Errorf("body = %q, want 'full_after_retry'", body)
	}
	if hdrs == nil {
		t.Error("expected non-nil response headers")
	}
}

func TestGetWithRetryFullHTTP400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"bad"}`)
	}))
	t.Cleanup(srv.Close)
	mgr := makeManager(t, "tok")
	_, _, err := GetWithRetryFull(srv.Client(), srv.URL+"/balance", nil,
		BuildHeaders("tok", "k", "s", "TRID", "P"),
		mgr, "k", "s", "TRID", "P")
	if err == nil {
		t.Fatal("expected error on HTTP 400")
	}
}

// rateLimitServer fails the first `failures` resource calls with EGW00201
// (per-second request limit exceeded), then answers okBody.
func rateLimitServer(t *testing.T, failures int, okBody string) (*httptest.Server, *int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls <= failures {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"rt_cd":"1","msg_cd":"EGW00201","msg1":"초당 거래건수를 초과하였습니다."}`)
			return
		}
		fmt.Fprint(w, okBody)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func shortenRateLimitBackoff(t *testing.T) {
	t.Helper()
	orig := rateLimitBackoff
	rateLimitBackoff = time.Millisecond
	t.Cleanup(func() { rateLimitBackoff = orig })
}

func TestGetWithRetryRateLimitRecovers(t *testing.T) {
	shortenRateLimitBackoff(t)
	srv, calls := rateLimitServer(t, 2, `{"result":"ok"}`)
	body, err := GetWithRetry(srv.Client(), srv.URL+"/price", nil,
		BuildHeaders("tok", "k", "s", "TRID", "P"),
		nil, "k", "s", "TRID", "P")
	if err != nil {
		t.Fatalf("GetWithRetry after rate limit: %v", err)
	}
	if !strings.Contains(string(body), "ok") {
		t.Errorf("body = %q, want 'ok'", body)
	}
	if *calls != 3 {
		t.Errorf("calls = %d, want 3", *calls)
	}
}

func TestGetWithRetryFullRateLimitRecovers(t *testing.T) {
	shortenRateLimitBackoff(t)
	srv, calls := rateLimitServer(t, 1, `{"result":"ok"}`)
	_, hdrs, err := GetWithRetryFull(srv.Client(), srv.URL+"/balance", nil,
		BuildHeaders("tok", "k", "s", "TRID", "P"),
		nil, "k", "s", "TRID", "P")
	if err != nil {
		t.Fatalf("GetWithRetryFull after rate limit: %v", err)
	}
	if hdrs == nil {
		t.Error("expected non-nil response headers")
	}
	if *calls != 2 {
		t.Errorf("calls = %d, want 2", *calls)
	}
}

func TestGetWithRetryRateLimitGivesUp(t *testing.T) {
	shortenRateLimitBackoff(t)
	srv, calls := rateLimitServer(t, 100, `{"result":"ok"}`)
	_, err := GetWithRetry(srv.Client(), srv.URL+"/price", nil,
		BuildHeaders("tok", "k", "s", "TRID", "P"),
		nil, "k", "s", "TRID", "P")
	if err == nil || !strings.Contains(err.Error(), "EGW00201") {
		t.Fatalf("err = %v, want EGW00201 error after retries exhausted", err)
	}
	if want := rateLimitMaxRetries + 1; *calls != want {
		t.Errorf("calls = %d, want %d", *calls, want)
	}
}
