package toss

import (
	"context"
	"fmt"
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
