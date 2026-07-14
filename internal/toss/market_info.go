package toss

import (
	"context"
	"time"
)

// ExchangeRateResponse is the result of GetExchangeRate.
type ExchangeRateResponse struct {
	BaseCurrency   string    `json:"baseCurrency"`
	QuoteCurrency  string    `json:"quoteCurrency"`
	Rate           string    `json:"rate"`
	MidRate        string    `json:"midRate"`
	BasisPoint     string    `json:"basisPoint"`
	RateChangeType string    `json:"rateChangeType"`
	ValidFrom      time.Time `json:"validFrom"`
	ValidUntil     time.Time `json:"validUntil"`
}

// GetExchangeRate fetches the exchange rate between base and quote
// currencies. A zero at omits the dateTime query param, returning the
// current rate.
func (c *Client) GetExchangeRate(ctx context.Context, base, quote string, at time.Time) (ExchangeRateResponse, error) {
	query := map[string]string{
		"baseCurrency":  base,
		"quoteCurrency": quote,
	}
	if !at.IsZero() {
		query["dateTime"] = at.Format(time.RFC3339)
	}
	return doGet[ExchangeRateResponse](ctx, c, "toss exchange-rate", "/api/v1/exchange-rate", query, nil)
}

// PreMarketSession is a KR pre-market trading window.
type PreMarketSession struct {
	StartTime                   time.Time  `json:"startTime"`
	SinglePriceAuctionStartTime *time.Time `json:"singlePriceAuctionStartTime"`
	EndTime                     time.Time  `json:"endTime"`
}

// RegularMarketSession is a KR regular trading window.
type RegularMarketSession struct {
	StartTime                   time.Time  `json:"startTime"`
	SinglePriceAuctionStartTime *time.Time `json:"singlePriceAuctionStartTime"`
	EndTime                     time.Time  `json:"endTime"`
}

// AfterMarketSession is a KR after-market trading window.
type AfterMarketSession struct {
	StartTime                 time.Time  `json:"startTime"`
	SinglePriceAuctionEndTime *time.Time `json:"singlePriceAuctionEndTime"`
	EndTime                   time.Time  `json:"endTime"`
}

// IntegratedHour bundles a KR trading day's sessions.
type IntegratedHour struct {
	PreMarket     *PreMarketSession     `json:"preMarket"`
	RegularMarket *RegularMarketSession `json:"regularMarket"`
	AfterMarket   *AfterMarketSession   `json:"afterMarket"`
}

// KrMarketDay is one day's KR market schedule.
type KrMarketDay struct {
	Date       string          `json:"date"`
	Integrated *IntegratedHour `json:"integrated"`
}

// KrMarketCalendarResponse is the result of GetKrMarketCalendar.
type KrMarketCalendarResponse struct {
	Today               KrMarketDay `json:"today"`
	PreviousBusinessDay KrMarketDay `json:"previousBusinessDay"`
	NextBusinessDay     KrMarketDay `json:"nextBusinessDay"`
}

// GetKrMarketCalendar fetches KR market operating hours for date (optional,
// YYYY-MM-DD; empty omits the query param and returns today's schedule).
func (c *Client) GetKrMarketCalendar(ctx context.Context, date string) (KrMarketCalendarResponse, error) {
	query := map[string]string{"date": date}
	return doGet[KrMarketCalendarResponse](ctx, c, "toss market-calendar-kr", "/api/v1/market-calendar/KR", query, nil)
}

// UsMarketSession is a US trading window. All four US session types
// (day/pre/regular/after market) share this identical {startTime, endTime}
// shape per the spec, unlike KR's sessions which each carry a differently
// named single-price-auction field.
type UsMarketSession struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// UsMarketDay is one day's US market schedule.
type UsMarketDay struct {
	Date          string           `json:"date"`
	DayMarket     *UsMarketSession `json:"dayMarket"`
	PreMarket     *UsMarketSession `json:"preMarket"`
	RegularMarket *UsMarketSession `json:"regularMarket"`
	AfterMarket   *UsMarketSession `json:"afterMarket"`
}

// UsMarketCalendarResponse is the result of GetUsMarketCalendar.
type UsMarketCalendarResponse struct {
	Today               UsMarketDay `json:"today"`
	PreviousBusinessDay UsMarketDay `json:"previousBusinessDay"`
	NextBusinessDay     UsMarketDay `json:"nextBusinessDay"`
}

// GetUsMarketCalendar fetches US market operating hours for date (optional,
// YYYY-MM-DD; empty omits the query param and returns today's schedule).
func (c *Client) GetUsMarketCalendar(ctx context.Context, date string) (UsMarketCalendarResponse, error) {
	query := map[string]string{"date": date}
	return doGet[UsMarketCalendarResponse](ctx, c, "toss market-calendar-us", "/api/v1/market-calendar/US", query, nil)
}
