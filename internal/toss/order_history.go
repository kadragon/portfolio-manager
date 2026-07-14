package toss

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

// OrderListParams filters GetOrders. Status is required (OPEN or CLOSED); the
// rest are optional.
type OrderListParams struct {
	Status string // required: OPEN or CLOSED
	Symbol string
	From   string // YYYY-MM-DD
	To     string
	Cursor string
	Limit  int
}

// OrderExecution is the fill result embedded in an Order. Nullable fields are
// null until (or unless) a fill occurs.
type OrderExecution struct {
	FilledQuantity     string     `json:"filledQuantity"`
	AverageFilledPrice *string    `json:"averageFilledPrice"`
	FilledAmount       *string    `json:"filledAmount"`
	Commission         *string    `json:"commission"`
	Tax                *string    `json:"tax"`
	FilledAt           *time.Time `json:"filledAt"`
	SettlementDate     *string    `json:"settlementDate"`
}

// Order is a single order as returned by GetOrders/GetOrder.
type Order struct {
	OrderID     string         `json:"orderId"`
	Symbol      string         `json:"symbol"`
	Side        string         `json:"side"`
	OrderType   string         `json:"orderType"`
	TimeInForce string         `json:"timeInForce"` // DAY/CLS/OPG
	Status      string         `json:"status"`      // PENDING/PENDING_CANCEL/PENDING_REPLACE/PARTIAL_FILLED/FILLED/CANCELED/REJECTED/CANCEL_REJECTED/REPLACE_REJECTED/REPLACED — open string, clients must tolerate unknown future values
	Price       *string        `json:"price"`
	Quantity    string         `json:"quantity"`
	OrderAmount *string        `json:"orderAmount"`
	Currency    string         `json:"currency"`
	OrderedAt   time.Time      `json:"orderedAt"`
	CanceledAt  *time.Time     `json:"canceledAt"`
	Execution   OrderExecution `json:"execution"`
}

// PaginatedOrderResponse is the result of GetOrders.
type PaginatedOrderResponse struct {
	Orders     []Order `json:"orders"`
	NextCursor *string `json:"nextCursor"`
	HasNext    bool    `json:"hasNext"`
}

// GetOrders lists orders for accountSeq filtered by params.Status (required).
//
// Per the API spec, status=OPEN returns the full open-order list (limit and
// cursor are ignored); status=CLOSED currently returns a 400
// "closed-not-supported" error despite otherwise-documented pagination
// params — this is expected server behavior today, not a bug in this client.
func (c *Client) GetOrders(ctx context.Context, accountSeq string, params OrderListParams) (PaginatedOrderResponse, error) {
	query := map[string]string{
		"status": params.Status,
		"symbol": params.Symbol,
		"from":   params.From,
		"to":     params.To,
		"cursor": params.Cursor,
	}
	if params.Limit > 0 {
		query["limit"] = strconv.Itoa(params.Limit)
	}
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	return doGet[PaginatedOrderResponse](ctx, c, "toss orders", "/api/v1/orders", query, headers)
}

// GetOrder returns the detail of a single order for accountSeq. orderId is an
// opaque server-issued token; it is URL-escaped defensively even though it is
// documented to only contain URL-safe characters.
func (c *Client) GetOrder(ctx context.Context, accountSeq, orderID string) (Order, error) {
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	path := "/api/v1/orders/" + url.PathEscape(orderID)
	return doGet[Order](ctx, c, "toss order", path, nil, headers)
}
