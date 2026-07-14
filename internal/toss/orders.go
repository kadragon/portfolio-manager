package toss

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/models"
)

// OrderCreateRequest is the body for CreateOrder. Exactly one of Quantity or
// OrderAmount must be set (quantity-based vs. amount-based orders); OrderType
// LIMIT requires Price, MARKET forbids it.
type OrderCreateRequest struct {
	ClientOrderID         string `json:"clientOrderId,omitempty"`
	Symbol                string `json:"symbol"`
	Side                  string `json:"side"`      // BUY/SELL
	OrderType             string `json:"orderType"` // LIMIT/MARKET
	TimeInForce           string `json:"timeInForce,omitempty"`
	Quantity              string `json:"quantity,omitempty"`
	Price                 string `json:"price,omitempty"`
	OrderAmount           string `json:"orderAmount,omitempty"`
	ConfirmHighValueOrder bool   `json:"confirmHighValueOrder,omitempty"`
}

// OrderResponse is the result of CreateOrder.
type OrderResponse struct {
	OrderID       string  `json:"orderId"`
	ClientOrderID *string `json:"clientOrderId"`
}

// CreateOrder submits a new order for accountSeq.
func (c *Client) CreateOrder(ctx context.Context, accountSeq string, req OrderCreateRequest) (OrderResponse, error) {
	if err := req.validate(); err != nil {
		return OrderResponse{}, err
	}
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	resp, err := doPost[OrderResponse](ctx, c, "toss order", "/api/v1/orders", req, headers)
	if err != nil {
		return OrderResponse{}, err
	}
	if resp.OrderID == "" {
		return OrderResponse{}, fmt.Errorf("toss order: missing orderId")
	}
	return resp, nil
}

// validate enforces the OrderCreateRequest oneOf shape from the Toss OpenAPI
// spec (components.schemas.OrderCreateRequest): exactly one of Quantity
// (quantity-based) or OrderAmount (amount-based, US MARKET only) must be
// set, and Price is required for LIMIT / forbidden for MARKET.
func (r OrderCreateRequest) validate() error {
	hasQuantity := strings.TrimSpace(r.Quantity) != ""
	hasAmount := strings.TrimSpace(r.OrderAmount) != ""
	switch {
	case hasQuantity && hasAmount:
		return fmt.Errorf("toss order: quantity and orderAmount are mutually exclusive, both were set")
	case !hasQuantity && !hasAmount:
		return fmt.Errorf("toss order: exactly one of quantity or orderAmount must be set")
	}
	if hasAmount && r.OrderType != "MARKET" {
		return fmt.Errorf("toss order: orderAmount-based orders require orderType MARKET, got %q", r.OrderType)
	}

	hasPrice := strings.TrimSpace(r.Price) != ""
	switch r.OrderType {
	case "MARKET":
		if hasPrice {
			return fmt.Errorf("toss order: orderType MARKET forbids price")
		}
	case "LIMIT":
		if !hasPrice {
			return fmt.Errorf("toss order: orderType LIMIT requires price")
		}
	}
	return nil
}

// OrderModifyRequest is the body for ModifyOrder.
type OrderModifyRequest struct {
	OrderType             string `json:"orderType"` // LIMIT/MARKET
	Quantity              string `json:"quantity,omitempty"`
	Price                 string `json:"price,omitempty"`
	ConfirmHighValueOrder bool   `json:"confirmHighValueOrder,omitempty"`
}

// OrderOperationResponse is the result of ModifyOrder/CancelOrder. OrderID is
// a newly issued identifier for the modify/cancel operation itself — it is
// distinct from the original order's orderId.
type OrderOperationResponse struct {
	OrderID string `json:"orderId"`
}

// ModifyOrder amends the price and/or quantity of an existing order.
func (c *Client) ModifyOrder(ctx context.Context, accountSeq, orderID string, req OrderModifyRequest) (OrderOperationResponse, error) {
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	path := "/api/v1/orders/" + url.PathEscape(orderID) + "/modify"
	return doPost[OrderOperationResponse](ctx, c, "toss order modify", path, req, headers)
}

// CancelOrder cancels an existing order. The endpoint takes no meaningful
// request body; an explicit empty object is sent to match the spec's
// documented `{}` example (cancelOrder requestBody schema is `{"type":
// "object"}`).
func (c *Client) CancelOrder(ctx context.Context, accountSeq, orderID string) (OrderOperationResponse, error) {
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	path := "/api/v1/orders/" + url.PathEscape(orderID) + "/cancel"
	return doPost[OrderOperationResponse](ctx, c, "toss order cancel", path, struct{}{}, headers)
}

// PlaceOrder creates a market quantity order for the given Toss accountSeq.
// Kept for source/behavior compatibility with existing callers
// (services.TossOrderClient, cmd/rebalance-order); new code should call
// CreateOrder directly for LIMIT/amount-based orders.
func (c *Client) PlaceOrder(ctx context.Context, accountSeq string, intent models.OrderIntent) (map[string]any, error) {
	accountSeq = strings.TrimSpace(accountSeq)
	if accountSeq == "" {
		return nil, fmt.Errorf("toss order: accountSeq is required")
	}
	if intent.Quantity <= 0 {
		return nil, fmt.Errorf("toss order: quantity must be positive")
	}
	side, err := tossOrderSide(intent.Side)
	if err != nil {
		return nil, err
	}

	resp, err := c.CreateOrder(ctx, accountSeq, OrderCreateRequest{
		Symbol:    strings.ToUpper(strings.TrimSpace(intent.Ticker)),
		Side:      side,
		OrderType: "MARKET",
		Quantity:  fmt.Sprintf("%d", intent.Quantity),
	})
	if err != nil {
		return nil, err
	}

	result := map[string]any{"orderId": resp.OrderID}
	if resp.ClientOrderID != nil {
		result["clientOrderId"] = *resp.ClientOrderID
	} else {
		result["clientOrderId"] = nil
	}
	return map[string]any{"result": result}, nil
}

func tossOrderSide(side string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy":
		return "BUY", nil
	case "sell":
		return "SELL", nil
	default:
		return "", fmt.Errorf("toss order: unsupported side %q", side)
	}
}
