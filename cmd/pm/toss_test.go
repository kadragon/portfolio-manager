package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/toss"
)

// newTossContainer builds an in-memory-DB container (matching
// newAccountContainer's pattern) wired to a fake Toss server. Pass a nil
// handler to leave c.TossClient unset (for testing the "not configured"
// guard).
func newTossContainer(t *testing.T, handler http.HandlerFunc) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	if handler != nil {
		srv := httptest.NewServer(tossTokenHandler(handler))
		t.Cleanup(srv.Close)
		c.TossClient = toss.NewClient(srv.Client(), srv.URL, "cid", "secret")
	}
	return c
}

// tossTokenHandler serves /oauth2/token before delegating to next, so tests
// only need to handle their endpoint of interest.
func tossTokenHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"Bearer","expires_in":3600}`))
			return
		}
		next(w, r)
	}
}

func TestTossOrderbook(t *testing.T) {
	ctx := context.Background()
	c := newTossContainer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orderbook" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbol"); got != "005930" {
			t.Fatalf("symbol query = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"timestamp":null,"currency":"KRW","asks":[],"bids":[]}}`))
	})

	if err := runToss(ctx, c, []string{"orderbook", "-symbol", "005930"}); err != nil {
		t.Fatalf("toss orderbook: %v", err)
	}
}

func TestTossOrderbookMissingSymbol(t *testing.T) {
	ctx := context.Background()
	c := newTossContainer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s", r.URL.Path)
	})
	if err := runToss(ctx, c, []string{"orderbook"}); err == nil {
		t.Fatal("expected error for missing -symbol")
	}
}

func TestTossClientNotConfigured(t *testing.T) {
	ctx := context.Background()
	c := newTossContainer(t, nil)
	err := runToss(ctx, c, []string{"orderbook", "-symbol", "005930"})
	if err == nil || !strings.Contains(err.Error(), "toss client not configured") {
		t.Fatalf("expected toss-not-configured error, got %v", err)
	}
}

const holdingsOverviewJSON = `{"result":{
	"totalPurchaseAmount":{"krw":"0","usd":null},
	"marketValue":{"amount":{"krw":"0","usd":null},"amountAfterCost":{"krw":"0","usd":null}},
	"profitLoss":{"amount":{"krw":"0","usd":null},"amountAfterCost":{"krw":"0","usd":null},"rate":"0","rateAfterCost":"0"},
	"dailyProfitLoss":{"amount":{"krw":"0","usd":null},"rate":"0"},
	"items":[]
}}`

func TestTossHoldings(t *testing.T) {
	ctx := context.Background()
	c := newTossContainer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/holdings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Tossinvest-Account"); got != "123" {
			t.Fatalf("X-Tossinvest-Account = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(holdingsOverviewJSON))
	})

	acc, err := c.Accounts.Create(ctx, "TOSS", numeric.Zero)
	if err != nil {
		t.Fatalf("Create account: %v", err)
	}
	if err := runAccount(ctx, c, []string{"update", "-id", acc.ID.String(), "-toss-account-seq", "123"}); err != nil {
		t.Fatalf("link toss account: %v", err)
	}

	if err := runToss(ctx, c, []string{"holdings", "-account", "TOSS"}); err != nil {
		t.Fatalf("toss holdings: %v", err)
	}
}

func TestTossHoldingsAccountNotTossLinked(t *testing.T) {
	ctx := context.Background()
	c := newTossContainer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s", r.URL.Path)
	})

	if _, err := c.Accounts.Create(ctx, "ISA", numeric.Zero); err != nil {
		t.Fatalf("Create account: %v", err)
	}

	err := runToss(ctx, c, []string{"holdings", "-account", "ISA"})
	if err == nil || !strings.Contains(err.Error(), "is not linked to a Toss accountSeq") {
		t.Fatalf("expected not-linked error, got %v", err)
	}
}

func TestTossHelpNoArgs(t *testing.T) {
	ctx := context.Background()
	c := newTossContainer(t, nil)
	if err := runToss(ctx, c, nil); err != nil {
		t.Fatalf("toss help (no args): %v", err)
	}
}

func TestTossUnknownVerb(t *testing.T) {
	ctx := context.Background()
	c := newTossContainer(t, nil)
	if err := runToss(ctx, c, []string{"bogus"}); err == nil {
		t.Fatal("expected error for unknown verb")
	}
}
