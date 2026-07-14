package toss

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetKrMarketCalendar(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market-calendar/KR" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"today": map[string]any{
				"date": "2026-07-14",
				"integrated": map[string]any{
					"preMarket": map[string]any{
						"startTime": "2026-07-14T08:00:00+09:00",
						"endTime":   "2026-07-14T09:00:00+09:00",
					},
					"regularMarket": map[string]any{
						"startTime":                   "2026-07-14T09:00:00+09:00",
						"singlePriceAuctionStartTime": "2026-07-14T15:20:00+09:00",
						"endTime":                     "2026-07-14T15:30:00+09:00",
					},
					"afterMarket": nil,
				},
			},
			"previousBusinessDay": map[string]any{"date": "2026-07-13"},
			"nextBusinessDay":     map[string]any{"date": "2026-07-15"},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetKrMarketCalendar(context.Background(), "2026-07-14")
	if err != nil {
		t.Fatalf("GetKrMarketCalendar: %v", err)
	}
	if gotQuery != "date=2026-07-14" {
		t.Fatalf("query = %q, want date=2026-07-14", gotQuery)
	}
	if got.Today.Date != "2026-07-14" {
		t.Fatalf("today.date = %q", got.Today.Date)
	}
	if got.Today.Integrated == nil || got.Today.Integrated.PreMarket == nil {
		t.Fatal("expected integrated.preMarket to be present")
	}
	if got.Today.Integrated.AfterMarket != nil {
		t.Fatalf("expected afterMarket nil, got %+v", got.Today.Integrated.AfterMarket)
	}
	if got.Today.Integrated.RegularMarket == nil || got.Today.Integrated.RegularMarket.SinglePriceAuctionStartTime == nil {
		t.Fatal("expected regularMarket.singlePriceAuctionStartTime to be present")
	}
	if got.PreviousBusinessDay.Date != "2026-07-13" || got.NextBusinessDay.Date != "2026-07-15" {
		t.Fatalf("adjacent days = %+v / %+v", got.PreviousBusinessDay, got.NextBusinessDay)
	}
}

func TestGetKrMarketCalendarOmitsEmptyDate(t *testing.T) {
	var gotQuery string
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"today":               map[string]any{"date": "2026-07-14"},
			"previousBusinessDay": map[string]any{"date": "2026-07-13"},
			"nextBusinessDay":     map[string]any{"date": "2026-07-15"},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	if _, err := client.GetKrMarketCalendar(context.Background(), ""); err != nil {
		t.Fatalf("GetKrMarketCalendar: %v", err)
	}
	if gotQuery != "" {
		t.Fatalf("query = %q, want empty (date omitted)", gotQuery)
	}
}

func TestGetUsMarketCalendar(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market-calendar/US" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("date") != "2026-07-14" {
			t.Fatalf("date query = %q", r.URL.Query().Get("date"))
		}
		writeJSON(t, w, map[string]any{"result": map[string]any{
			"today": map[string]any{
				"date": "2026-07-14",
				"dayMarket": map[string]any{
					"startTime": "2026-07-14T04:00:00-04:00",
					"endTime":   "2026-07-14T20:00:00-04:00",
				},
				"preMarket": nil,
				"regularMarket": map[string]any{
					"startTime": "2026-07-14T09:30:00-04:00",
					"endTime":   "2026-07-14T16:00:00-04:00",
				},
				"afterMarket": nil,
			},
			"previousBusinessDay": map[string]any{"date": "2026-07-13"},
			"nextBusinessDay":     map[string]any{"date": "2026-07-15"},
		}})
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	got, err := client.GetUsMarketCalendar(context.Background(), "2026-07-14")
	if err != nil {
		t.Fatalf("GetUsMarketCalendar: %v", err)
	}
	if got.Today.DayMarket == nil || got.Today.RegularMarket == nil {
		t.Fatal("expected dayMarket and regularMarket to be present")
	}
	if got.Today.PreMarket != nil || got.Today.AfterMarket != nil {
		t.Fatalf("expected preMarket/afterMarket nil, got %+v / %+v", got.Today.PreMarket, got.Today.AfterMarket)
	}
}

func TestGetMarketCalendarHTTPError(t *testing.T) {
	srv := tokenOnlyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid-date","message":"invalid date","requestId":"req-1"}}`))
	})
	t.Cleanup(srv.Close)

	client := NewClient(srv.Client(), srv.URL, "cid", "secret")
	if _, err := client.GetKrMarketCalendar(context.Background(), "bad-date"); err == nil || !strings.Contains(err.Error(), "invalid-date") {
		t.Fatalf("expected invalid-date error, got %v", err)
	}
	if _, err := client.GetUsMarketCalendar(context.Background(), "bad-date"); err == nil || !strings.Contains(err.Error(), "invalid-date") {
		t.Fatalf("expected invalid-date error, got %v", err)
	}
}
