package toss

import (
	"context"
	"fmt"
	"net/url"
)

// ConditionRequest describes one watch leg of a conditional order: "when the
// price reaches TriggerPrice, place an OrderSide order." Quantity and
// order type are shared at the group level (see ConditionalOrderCreateRequest
// / ConditionalOrderModifyRequest), not per leg.
type ConditionRequest struct {
	OrderSide    string `json:"orderSide"` // BUY/SELL
	TriggerPrice string `json:"triggerPrice"`
	OrderPrice   string `json:"orderPrice,omitempty"` // required iff group OrderType=LIMIT; omit for MARKET
}

// ConditionalOrderCreateRequest is the body for CreateConditionalOrder. Type
// governs the shape of First/Second:
//   - SINGLE: only First is monitored; Second must be nil.
//   - OCO (One-Cancels-the-Other): First and Second are monitored
//     simultaneously; fulfilling one auto-cancels the other. Both legs must
//     be SELL, and the group OrderType must be LIMIT.
//   - OTO (One-Triggers-the-Other): once First fills, Second starts being
//     monitored. First must be BUY, Second must be SELL, and the group
//     OrderType must be LIMIT.
type ConditionalOrderCreateRequest struct {
	Symbol                string            `json:"symbol"`
	Type                  string            `json:"type"` // SINGLE/OCO/OTO
	Quantity              string            `json:"quantity"`
	OrderType             string            `json:"orderType"` // LIMIT/MARKET
	ClientOrderID         string            `json:"clientOrderId,omitempty"`
	ExpireDate            string            `json:"expireDate"` // YYYY-MM-DD, required
	First                 ConditionRequest  `json:"first"`
	Second                *ConditionRequest `json:"second,omitempty"`
	ConfirmHighValueOrder bool              `json:"confirmHighValueOrder,omitempty"`
}

// ConditionalOrderModifyRequest is the body for ModifyConditionalOrder. It
// re-specifies the whole conditional order group (Symbol is fixed by the
// target conditionalOrderId, so it isn't part of the body); the same
// structural rules as ConditionalOrderCreateRequest apply.
type ConditionalOrderModifyRequest struct {
	Type                  string            `json:"type"`
	Quantity              string            `json:"quantity"`
	OrderType             string            `json:"orderType"`
	ExpireDate            string            `json:"expireDate"`
	First                 ConditionRequest  `json:"first"`
	Second                *ConditionRequest `json:"second,omitempty"`
	ConfirmHighValueOrder bool              `json:"confirmHighValueOrder,omitempty"`
}

// ConditionalOrderCreateResponse is the result of CreateConditionalOrder.
type ConditionalOrderCreateResponse struct {
	ConditionalOrderID string  `json:"conditionalOrderId"`
	ClientOrderID      *string `json:"clientOrderId"`
}

// ConditionalOrderResponse is the result of ModifyConditionalOrder: only the
// (possibly new) conditionalOrderId. Modifying a conditional order cancels
// the existing one and creates a new one, so the returned ID replaces the
// original for subsequent get/modify/cancel calls.
type ConditionalOrderResponse struct {
	ConditionalOrderID string `json:"conditionalOrderId"`
}

// CreateConditionalOrder creates a conditional order that watches Symbol's
// price and automatically places a BUY/SELL order once the watch
// condition(s) are met.
func (c *Client) CreateConditionalOrder(ctx context.Context, accountSeq string, req ConditionalOrderCreateRequest) (ConditionalOrderCreateResponse, error) {
	if err := validateConditionalOrderLegs(req.Type, req.OrderType, req.First, req.Second); err != nil {
		return ConditionalOrderCreateResponse{}, err
	}
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	return doPost[ConditionalOrderCreateResponse](ctx, c, "toss conditional order", "/api/v1/conditional-orders", req, headers)
}

// ModifyConditionalOrder replaces the conditional order identified by
// conditionalOrderID with req. Per the API, this cancels the existing order
// and creates a new one; use the conditionalOrderId in the response for
// subsequent calls.
func (c *Client) ModifyConditionalOrder(ctx context.Context, accountSeq, conditionalOrderID string, req ConditionalOrderModifyRequest) (ConditionalOrderResponse, error) {
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	path := "/api/v1/conditional-orders/" + url.PathEscape(conditionalOrderID) + "/modify"
	return doPost[ConditionalOrderResponse](ctx, c, "toss conditional order modify", path, req, headers)
}

// CancelConditionalOrder cancels the conditional order identified by
// conditionalOrderID. The API responds 204 No Content on success, so there
// is no result payload to return.
func (c *Client) CancelConditionalOrder(ctx context.Context, accountSeq, conditionalOrderID string) error {
	headers := map[string]string{"X-Tossinvest-Account": accountSeq}
	path := "/api/v1/conditional-orders/" + url.PathEscape(conditionalOrderID)
	_, err := doDelete[struct{}](ctx, c, "toss conditional order cancel", path, headers)
	return err
}

// validateConditionalOrderLegs enforces the structural rules the API
// documents for each conditional order type, so callers get a clear local
// error instead of a 400/422 round trip.
func validateConditionalOrderLegs(orderType, groupOrderType string, first ConditionRequest, second *ConditionRequest) error {
	switch orderType {
	case "SINGLE":
		if second != nil {
			return fmt.Errorf("toss conditional order: type SINGLE must not set second")
		}
	case "OCO":
		if second == nil {
			return fmt.Errorf("toss conditional order: type OCO requires second")
		}
		if groupOrderType != "LIMIT" {
			return fmt.Errorf("toss conditional order: type OCO requires orderType LIMIT, got %q", groupOrderType)
		}
		if first.OrderSide != "SELL" || second.OrderSide != "SELL" {
			return fmt.Errorf("toss conditional order: type OCO requires both legs SELL")
		}
	case "OTO":
		if second == nil {
			return fmt.Errorf("toss conditional order: type OTO requires second")
		}
		if groupOrderType != "LIMIT" {
			return fmt.Errorf("toss conditional order: type OTO requires orderType LIMIT, got %q", groupOrderType)
		}
		if first.OrderSide != "BUY" {
			return fmt.Errorf("toss conditional order: type OTO requires first leg BUY")
		}
		if second.OrderSide != "SELL" {
			return fmt.Errorf("toss conditional order: type OTO requires second leg SELL")
		}
	default:
		return fmt.Errorf("toss conditional order: unsupported type %q", orderType)
	}
	return nil
}
