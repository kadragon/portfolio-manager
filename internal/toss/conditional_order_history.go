package toss

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ConditionalOrderListParams are the query filters for GetConditionalOrders.
type ConditionalOrderListParams struct {
	Status string // required: OPEN or CLOSED
	Symbol string
	Cursor string
	Limit  int
}

// ConditionalOrderCondition is one watch leg (First/Second) of a
// ConditionalOrderDetailResponse.
type ConditionalOrderCondition struct {
	Type             string  `json:"type"`   // STOP/PROFIT_RATE
	Status           string  `json:"status"` // WATCHING/HOLDING/PAUSED/ORDERING/ORDERED/COMPLETED/EXPIRED/CANCELED
	TriggerPrice     *string `json:"triggerPrice"`
	TargetProfitRate *string `json:"targetProfitRate"`
	OrderPrice       *string `json:"orderPrice"`
	TriggeredOrderID *string `json:"triggeredOrderId"`
}

// ConditionalOrderDetailResponse is one conditional order, returned both by
// GetConditionalOrders (list items) and GetConditionalOrder (single detail).
// Second is present only for OCO/OTO; SINGLE orders leave it nil.
type ConditionalOrderDetailResponse struct {
	ConditionalOrderID string                     `json:"conditionalOrderId"`
	Type               string                     `json:"type"`   // SINGLE/OCO/OTO
	Status             string                     `json:"status"` // WATCHING/PAUSED/ORDERING/ORDERED/COMPLETED/EXPIRED
	Symbol             string                     `json:"symbol"`
	Market             string                     `json:"market"` // KR/US
	Quantity           string                     `json:"quantity"`
	OrderType          string                     `json:"orderType"`
	ExpireDate         string                     `json:"expireDate"`
	First              ConditionalOrderCondition  `json:"first"`
	Second             *ConditionalOrderCondition `json:"second,omitempty"`
	CreatedAt          time.Time                  `json:"createdAt"`
}

// PaginatedConditionalOrderResponse is the result of GetConditionalOrders.
type PaginatedConditionalOrderResponse struct {
	ConditionalOrders []ConditionalOrderDetailResponse `json:"conditionalOrders"`
	NextCursor        *string                          `json:"nextCursor"`
	HasNext           bool                             `json:"hasNext"`
}

// GetConditionalOrders lists conditional orders for accountSeq. Status is
// required (OPEN or CLOSED); Symbol, Cursor, and Limit are optional filters.
// Limit defaults to 20 (max 100) when <= 0, per the API.
func (c *Client) GetConditionalOrders(ctx context.Context, accountSeq string, params ConditionalOrderListParams) (PaginatedConditionalOrderResponse, error) {
	if params.Status != "OPEN" && params.Status != "CLOSED" {
		return PaginatedConditionalOrderResponse{}, fmt.Errorf("toss conditional orders: status must be OPEN or CLOSED, got %q", params.Status)
	}
	query := map[string]string{
		"status": params.Status,
		"symbol": params.Symbol,
		"cursor": params.Cursor,
	}
	if params.Limit > 0 {
		query["limit"] = strconv.Itoa(params.Limit)
	}
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	return doGet[PaginatedConditionalOrderResponse](ctx, c, "toss conditional orders", "/api/v1/conditional-orders", query, headers)
}

// GetConditionalOrder returns the detail of one conditional order,
// identified by conditionalOrderID. Covers both open and closed orders.
func (c *Client) GetConditionalOrder(ctx context.Context, accountSeq, conditionalOrderID string) (ConditionalOrderDetailResponse, error) {
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	path := "/api/v1/conditional-orders/" + url.PathEscape(conditionalOrderID)
	return doGet[ConditionalOrderDetailResponse](ctx, c, "toss conditional order", path, nil, headers)
}
