package kis

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
)

// failingPriceServer answers auth and fails every resource call, standing in for
// an outage or a rate-limit rejection.
func failingPriceServer(t *testing.T) (*httptest.Server, *TokenManager) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/oauth2/token") {
			fmt.Fprint(w, `{"access_token":"tok","expires_in":86400}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"msg_cd":"EGW00201","msg1":"rate limit exceeded"}`)
	}))
	t.Cleanup(srv.Close)

	store := &MemoryTokenStore{}
	_ = store.Save("tok", time.Now().Add(24*time.Hour))
	auth := &AuthClient{HTTPClient: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s"}
	return srv, NewTokenManager(store, auth, time.Minute)
}

// A failed call must not look like a closed market. Price sync walks back over a
// (0, nil) result, so collapsing an outage into one makes it store an earlier
// close for a date that traded normally — a wrong benchmark return reported as
// if it were real.
func TestGetHistoricalCloseReportsDomesticFailureAsError(t *testing.T) {
	srv, mgr := failingPriceServer(t)
	c := &UnifiedPriceClient{
		Domestic: &DomesticPriceClient{
			HTTP: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s",
			CustType: "P", Env: "real", Manager: mgr,
		},
	}

	price, err := c.GetHistoricalClose("005930", datex.New(2025, time.July, 8), "")
	if err == nil {
		t.Errorf("GetHistoricalClose returned (%v, nil) on a failing venue, want an error", price)
	}
	if price != 0 {
		t.Errorf("price = %v on failure, want 0", price)
	}
}

func TestGetHistoricalCloseReportsOverseasFailureAsError(t *testing.T) {
	srv, mgr := failingPriceServer(t)
	c := &UnifiedPriceClient{
		Overseas: &OverseasPriceClient{
			HTTP: srv.Client(), BaseURL: srv.URL, AppKey: "k", AppSecret: "s",
			CustType: "P", Env: "real", Manager: mgr,
		},
	}

	price, err := c.GetHistoricalClose("SPY", datex.New(2025, time.July, 8), "AMEX")
	if err == nil {
		t.Errorf("GetHistoricalClose returned (%v, nil) with every exchange failing, want an error", price)
	}
	if price != 0 {
		t.Errorf("price = %v on failure, want 0", price)
	}
}
