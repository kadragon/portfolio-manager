package toss

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// newLiveClient skips the test unless TOSS_LIVE=1 and credentials are
// configured, then returns a client and a resolved accountSeq (from
// TOSS_ACCOUNT_SEQ, or the first account GetAccounts returns).
func newLiveClient(t *testing.T) (*Client, string) {
	t.Helper()
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
	return c, accountSeq
}

func TestLiveFetchAccountSnapshot(t *testing.T) {
	c, accountSeq := newLiveClient(t)

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

// TestLiveTossReadEndpoints chains a handful of cheap, side-effect-free reads
// across most of the client's endpoint groups in a single test, rather than
// one live test per endpoint — TOSS_LIVE token issuance is rate-limited to
// ~1/min, so a per-endpoint sweep would be slow and likely to hit that limit.
// It only proves the real API's response shapes decode into our structs;
// write operations (CreateOrder, ModifyOrder, CancelOrder, conditional-order
// writes) must never run here or in any other TOSS_LIVE test — placing or
// altering a real order is not something a test suite gets to do.
func TestLiveTossReadEndpoints(t *testing.T) {
	c, accountSeq := newLiveClient(t)
	ctx := context.Background()

	if _, err := c.GetAccounts(ctx); err != nil {
		t.Fatalf("GetAccounts: %v", err)
	}

	if _, err := c.GetHoldings(ctx, accountSeq, ""); err != nil {
		t.Fatalf("GetHoldings: %v", err)
	}

	if _, err := c.GetBuyingPower(ctx, accountSeq, "KRW"); err != nil {
		t.Fatalf("GetBuyingPower: %v", err)
	}

	if _, err := c.GetOrders(ctx, accountSeq, OrderListParams{Status: "OPEN"}); err != nil {
		t.Fatalf("GetOrders: %v", err)
	}

	if _, err := c.GetPrices(ctx, []string{"005930"}); err != nil {
		t.Fatalf("GetPrices: %v", err)
	}

	if _, err := c.GetExchangeRate(ctx, "USD", "KRW", time.Time{}); err != nil {
		t.Fatalf("GetExchangeRate: %v", err)
	}

	t.Log("live read sweep ok: accounts, holdings, buying-power, orders, prices, exchange-rate")
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
