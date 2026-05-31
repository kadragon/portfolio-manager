package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/numeric"
)

// TestDashboardIndexPopulated seeds a full portfolio (group → stock → account →
// holding) and renders the dashboard, exercising the portfolio-summary and
// holdings-by-group assembly paths plus the populated template branch.
func TestDashboardIndexPopulated(t *testing.T) {
	e, c := setupDashboard(t)
	ctx := context.Background()

	g, err := c.Groups.Create(ctx, "국내성장", 100.0)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	st, err := c.Stocks.Create(ctx, "005930", g.ID)
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}
	acc, err := c.Accounts.Create(ctx, "메인", numeric.FromInt(1000000))
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if _, err := c.Holdings.Create(ctx, acc.ID, st.ID, numeric.FromInt(10)); err != nil {
		t.Fatalf("create holding: %v", err)
	}

	rec := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	e.ServeHTTP(rr, rec)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
}
