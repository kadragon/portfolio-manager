package main

import (
	"testing"
	"time"
)

func TestApplyCreateConditionalDefaults(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 15, 23, 30, 0, 0, time.FixedZone("KST", 9*60*60))
	tests := []struct {
		name           string
		action         string
		orderType      string
		expireDate     string
		wantOrderType  string
		wantExpireDate string
	}{
		{
			name:           "defaults conditional create",
			action:         "create-conditional",
			wantOrderType:  "MARKET",
			wantExpireDate: "2026-07-16",
		},
		{
			name:           "preserves explicit create values",
			action:         "create-conditional",
			orderType:      "LIMIT",
			expireDate:     "2026-07-31",
			wantOrderType:  "LIMIT",
			wantExpireDate: "2026-07-31",
		},
		{
			name:           "does not default conditional modify",
			action:         "modify-conditional",
			wantOrderType:  "",
			wantExpireDate: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotOrderType, gotExpireDate := applyCreateConditionalDefaults(
				tt.action,
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
