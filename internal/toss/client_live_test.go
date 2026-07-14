package toss

import (
	"context"
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
		accountSeq = fetchFirstAccountSeq(t, c)
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

func fetchFirstAccountSeq(t *testing.T, c *Client) string {
	t.Helper()
	accounts, err := c.GetAccounts(context.Background())
	if err != nil {
		t.Fatalf("GetAccounts: %v", err)
	}
	if len(accounts) == 0 || accounts[0].AccountSeq == 0 {
		t.Fatal("no Toss accounts returned")
	}
	return strconv.FormatInt(accounts[0].AccountSeq, 10)
}
