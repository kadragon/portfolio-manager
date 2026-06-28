package toss

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestLiveFetchAccountSnapshot(t *testing.T) {
	if os.Getenv("TOSS_LIVE") != "1" {
		t.Skip("set TOSS_LIVE=1 to call Toss Open API")
	}
	clientID := strings.TrimSpace(os.Getenv("TOSS_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("TOSS_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		t.Skip("TOSS_CLIENT_ID/TOSS_CLIENT_SECRET not configured")
	}

	c := NewClient(http.DefaultClient, os.Getenv("TOSS_BASE_URL"), clientID, clientSecret)
	accountSeq := strings.TrimSpace(os.Getenv("TOSS_ACCOUNT_SEQ"))
	if accountSeq == "" {
		token, err := c.accessToken()
		if err != nil {
			t.Fatalf("accessToken: %v", err)
		}
		accountSeq = fetchFirstAccountSeq(t, c, token)
	}

	snapshot, err := c.FetchAccountSnapshot(accountSeq, "")
	if err != nil {
		t.Fatalf("FetchAccountSnapshot: %v", err)
	}
	if snapshot.CashBalance.IsNegative() {
		t.Fatalf("cash balance is negative: %s", snapshot.CashBalance.String())
	}
	for _, h := range snapshot.Holdings {
		if strings.TrimSpace(h.Ticker) == "" {
			t.Fatal("holding ticker is empty")
		}
		if !h.Quantity.IsPositive() {
			t.Fatalf("holding %s quantity is not positive: %s", h.Ticker, h.Quantity.String())
		}
	}
	t.Logf("snapshot ok: holdings=%d cash_present=%t", len(snapshot.Holdings), !snapshot.CashBalance.IsZero())
}

func fetchFirstAccountSeq(t *testing.T, c *Client, token string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/accounts", nil)
	if err != nil {
		t.Fatalf("create accounts request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		t.Fatalf("accounts request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read accounts response: %v", err)
	}
	if resp.StatusCode >= 400 {
		t.Fatalf("accounts HTTP %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Result []struct {
			AccountSeq int64 `json:"accountSeq"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal accounts: %v", err)
	}
	if len(parsed.Result) == 0 || parsed.Result[0].AccountSeq == 0 {
		t.Fatal("no Toss accounts returned")
	}
	return itoa(parsed.Result[0].AccountSeq)
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
