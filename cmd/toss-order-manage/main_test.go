package main

import (
	"strings"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/toss"
)

func TestApplyCreateConditionalDefaults(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 15, 23, 30, 0, 0, time.FixedZone("KST", 9*60*60))
	tests := []struct {
		name           string
		action         string
		condType       string
		orderType      string
		expireDate     string
		wantOrderType  string
		wantExpireDate string
	}{
		{
			name:           "defaults conditional create for SINGLE",
			action:         "create-conditional",
			condType:       "SINGLE",
			wantOrderType:  "MARKET",
			wantExpireDate: "2026-07-16",
		},
		{
			name:           "defaults conditional create for OCO to LIMIT",
			action:         "create-conditional",
			condType:       "OCO",
			wantOrderType:  "LIMIT",
			wantExpireDate: "2026-07-16",
		},
		{
			name:           "defaults conditional create for OTO to LIMIT",
			action:         "create-conditional",
			condType:       "OTO",
			wantOrderType:  "LIMIT",
			wantExpireDate: "2026-07-16",
		},
		{
			name:           "preserves explicit create values",
			action:         "create-conditional",
			condType:       "SINGLE",
			orderType:      "LIMIT",
			expireDate:     "2026-07-31",
			wantOrderType:  "LIMIT",
			wantExpireDate: "2026-07-31",
		},
		{
			name:           "does not default conditional modify",
			action:         "modify-conditional",
			condType:       "SINGLE",
			wantOrderType:  "",
			wantExpireDate: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotOrderType, gotExpireDate := applyCreateConditionalDefaults(
				tt.action,
				tt.condType,
				tt.orderType,
				tt.expireDate,
				now,
			)
			if gotOrderType != tt.wantOrderType {
				t.Errorf("orderType = %q, want %q", gotOrderType, tt.wantOrderType)
			}
			if gotExpireDate != tt.wantExpireDate {
				t.Errorf("expireDate = %q, want %q", gotExpireDate, tt.wantExpireDate)
			}
		})
	}
}

func TestBuildAmountOrderRequest(t *testing.T) {
	t.Parallel()

	req, err := buildAmountOrderRequest(" schd ", " buy ", "23.30", "amount-1", false)
	if err != nil {
		t.Fatalf("buildAmountOrderRequest: %v", err)
	}
	if req.Symbol != "SCHD" || req.Side != "BUY" || req.OrderType != "MARKET" {
		t.Fatalf("request routing fields = %+v", req)
	}
	if req.OrderAmount != "23.30" || req.Quantity != "" || req.Price != "" {
		t.Fatalf("request amount fields = %+v", req)
	}
	if req.ClientOrderID != "amount-1" {
		t.Errorf("clientOrderId = %q, want amount-1", req.ClientOrderID)
	}
}

func TestBuildAmountOrderRequestRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		symbol  string
		side    string
		amount  string
		wantErr string
	}{
		{name: "missing symbol", side: "BUY", amount: "10", wantErr: "-symbol is required"},
		{name: "invalid side", symbol: "SCHD", side: "hold", amount: "10", wantErr: "-side must be BUY or SELL"},
		{name: "missing amount", symbol: "SCHD", side: "BUY", wantErr: "-order-amount is required"},
		{name: "negative amount", symbol: "SCHD", side: "BUY", amount: "-1", wantErr: "positive decimal"},
		{name: "zero amount", symbol: "SCHD", side: "BUY", amount: "0", wantErr: "greater than zero"},
		{name: "exponent amount", symbol: "SCHD", side: "BUY", amount: "1e2", wantErr: "positive decimal"},
		{name: "too long", symbol: "SCHD", side: "BUY", amount: strings.Repeat("1", 31), wantErr: "at most 30 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := buildAmountOrderRequest(tt.symbol, tt.side, tt.amount, "", false)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAmountOrderStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		symbol  string
		stocks  []toss.StockInfo
		wantErr string
	}{
		{
			name:   "active USD stock",
			symbol: "SCHD",
			stocks: []toss.StockInfo{
				{Symbol: "SCHD", Currency: "USD", Market: "AMEX", Status: "ACTIVE"},
			},
		},
		{
			name:   "domestic stock",
			symbol: "0052D0",
			stocks: []toss.StockInfo{
				{Symbol: "0052D0", Currency: "KRW", Market: "KOSPI", Status: "ACTIVE"},
			},
			wantErr: "USD stock",
		},
		{
			name:   "inactive stock",
			symbol: "SCHD",
			stocks: []toss.StockInfo{
				{Symbol: "SCHD", Currency: "USD", Market: "AMEX", Status: "DELISTED"},
			},
			wantErr: "not active",
		},
		{name: "missing stock", symbol: "SCHD", wantErr: "did not return"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateAmountOrderStock(tt.stocks, tt.symbol)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("validateAmountOrderStock: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
