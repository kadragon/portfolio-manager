// Command toss-order-manage modifies or cancels an existing Toss order, and
// creates/modifies/cancels Toss conditional orders (SINGLE/OCO/OTO).
//
// Conditional orders are a deliberate, scoped exception to this codebase's
// "no unattended/scheduled execution" guarantee: once created, Toss's
// servers watch the trigger price and fire the order on their own, later,
// with no human present, bounded only by the order's expireDate. Per
// docs/adr/0001-account-type-aware-tax-optimal-rebalancing.md's 2026-07-14
// revision, this is the only such exception in this codebase, and only for
// Toss-linked accounts (KIS has no conditional-order support to parallel
// it). The mitigation is the same dry-run-by-default + explicit -yes gate as
// cmd/rebalance-order, plus (for conditional orders) a full preview of the
// exact trigger/expire conditions before the human confirms — the
// unattended window is bounded to what they explicitly approved and for how
// long they approved it, never open-ended.
//
// This binary intentionally stays out of cmd/pm (the read/mutate-DB data
// CLI) and does not touch cmd/rebalance-order (kept single-purpose: place
// one market order).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/toss"
)

const usage = `usage:
  toss-order-manage -account NAME -action modify -order-id ID -order-type LIMIT|MARKET [-quantity N] [-price P] [-confirm-high-value-order] [-yes]
  toss-order-manage -account NAME -action cancel -order-id ID [-yes]
  toss-order-manage -account NAME -action create-conditional -symbol T -type SINGLE|OCO|OTO -quantity N [-order-type LIMIT|MARKET] [-expire-date YYYY-MM-DD] -first-side BUY|SELL -first-trigger-price P [-first-order-price P] [-second-side BUY|SELL] [-second-trigger-price P] [-second-order-price P] [-client-order-id ID] [-confirm-high-value-order] [-yes]
  toss-order-manage -account NAME -action modify-conditional -conditional-order-id ID -type SINGLE|OCO|OTO -quantity N -order-type LIMIT|MARKET -expire-date YYYY-MM-DD -first-side BUY|SELL -first-trigger-price P [-first-order-price P] [-second-side BUY|SELL] [-second-trigger-price P] [-second-order-price P] [-confirm-high-value-order] [-yes]
  toss-order-manage -account NAME -action cancel-conditional -conditional-order-id ID [-yes]

create-conditional defaults: order-type=MARKET, expire-date=tomorrow in KST
`

var koreaStandardTime = time.FixedZone("KST", 9*60*60)

func main() {
	account := flag.String("account", "", "account name, exact or unique substring match; must be linked to a Toss accountSeq")
	action := flag.String("action", "", "modify|cancel|create-conditional|modify-conditional|cancel-conditional")
	orderID := flag.String("order-id", "", "existing order ID (modify/cancel)")
	conditionalOrderID := flag.String("conditional-order-id", "", "existing conditional order ID (modify-conditional/cancel-conditional)")
	orderType := flag.String("order-type", "", "LIMIT or MARKET (create-conditional default: MARKET)")
	quantity := flag.String("quantity", "", "share quantity")
	price := flag.String("price", "", "limit price (modify only)")
	confirmHighValueOrder := flag.Bool("confirm-high-value-order", false, "acknowledge Toss's high-value-order confirmation requirement")
	symbol := flag.String("symbol", "", "ticker symbol (create-conditional)")
	condType := flag.String("type", "", "SINGLE|OCO|OTO (conditional orders)")
	expireDate := flag.String("expire-date", "", "YYYY-MM-DD (create-conditional default: tomorrow in KST)")
	clientOrderID := flag.String("client-order-id", "", "optional client-supplied order ID (create-conditional)")
	firstSide := flag.String("first-side", "", "BUY|SELL (conditional orders, first leg)")
	firstTriggerPrice := flag.String("first-trigger-price", "", "trigger price for the first leg")
	firstOrderPrice := flag.String("first-order-price", "", "order price for the first leg (required iff order-type LIMIT)")
	secondSide := flag.String("second-side", "", "BUY|SELL (OCO/OTO second leg)")
	secondTriggerPrice := flag.String("second-trigger-price", "", "trigger price for the second leg (OCO/OTO)")
	secondOrderPrice := flag.String("second-order-price", "", "order price for the second leg (OCO/OTO, required iff order-type LIMIT)")
	yes := flag.Bool("yes", false, "actually call the Toss API; without this flag, only prints a dry-run preview")
	flag.Parse()

	*action = strings.ToLower(strings.TrimSpace(*action))
	switch *action {
	case "modify", "cancel", "create-conditional", "modify-conditional", "cancel-conditional":
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	*orderType, *expireDate = applyCreateConditionalDefaults(*action, *orderType, *expireDate, time.Now())

	if strings.TrimSpace(*account) == "" {
		fail("-account is required")
	}

	second := buildSecondLeg(*secondSide, *secondTriggerPrice, *secondOrderPrice)
	if err := validateAction(*action, *orderID, conditionalOrderID, *orderType, *quantity, *symbol, *condType,
		*expireDate, *firstSide, *firstTriggerPrice, second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	c, err := container.New("")
	if err != nil {
		log.Fatalf("init container: %v", err)
	}
	defer func() { _ = c.Close() }()

	if c.TossClient == nil {
		log.Fatalf("Toss is not configured (.env TOSS_CLIENT_ID/TOSS_CLIENT_SECRET)")
	}
	fmt.Fprintf(os.Stderr, "Toss base URL=%q — this places a LIVE order-management action against real money\n", c.TossClient.BaseURL)

	ctx := context.Background()
	acct, err := resolveTossAccount(ctx, c, *account)
	if err != nil {
		log.Fatalf("resolve account: %v", err)
	}
	accountSeq := fmt.Sprintf("%d", *acct.TossAccountSeq)

	first := toss.ConditionRequest{
		OrderSide:    strings.ToUpper(strings.TrimSpace(*firstSide)),
		TriggerPrice: strings.TrimSpace(*firstTriggerPrice),
		OrderPrice:   strings.TrimSpace(*firstOrderPrice),
	}

	switch *action {
	case "modify":
		// Fetch the order being changed so the preview shows what it IS, not
		// just what it will become — without this, a human confirming -yes
		// has nothing to compare the new quantity/price against and can't
		// catch a fat-finger typo (e.g. 5 vs 50). Fetching also fails fast if
		// orderId doesn't exist, before ever calling ModifyOrder.
		current, err := c.TossClient.GetOrder(ctx, accountSeq, *orderID)
		if err != nil {
			log.Fatalf("look up current order before modify: %v", err)
		}
		req := toss.OrderModifyRequest{
			OrderType:             strings.ToUpper(strings.TrimSpace(*orderType)),
			Quantity:              strings.TrimSpace(*quantity),
			Price:                 strings.TrimSpace(*price),
			ConfirmHighValueOrder: *confirmHighValueOrder,
		}
		if !*yes {
			printDryRun("modify", *account, accountSeq, map[string]any{"orderId": *orderID, "currentOrder": current, "proposedChange": req})
			return
		}
		resp, err := c.TossClient.ModifyOrder(ctx, accountSeq, *orderID, req)
		if err != nil {
			log.Fatalf("modify order: %v", err)
		}
		printJSON(resp)

	case "cancel":
		// Same rationale as modify: show what's actually being canceled
		// (symbol/side/quantity/status) rather than a bare opaque orderId,
		// and fail fast if it doesn't exist.
		current, err := c.TossClient.GetOrder(ctx, accountSeq, *orderID)
		if err != nil {
			log.Fatalf("look up current order before cancel: %v", err)
		}
		if !*yes {
			printDryRun("cancel", *account, accountSeq, map[string]any{"orderId": *orderID, "currentOrder": current})
			return
		}
		resp, err := c.TossClient.CancelOrder(ctx, accountSeq, *orderID)
		if err != nil {
			log.Fatalf("cancel order: %v", err)
		}
		printJSON(resp)

	case "create-conditional":
		req := toss.ConditionalOrderCreateRequest{
			Symbol:                strings.ToUpper(strings.TrimSpace(*symbol)),
			Type:                  strings.ToUpper(strings.TrimSpace(*condType)),
			Quantity:              strings.TrimSpace(*quantity),
			OrderType:             strings.ToUpper(strings.TrimSpace(*orderType)),
			ClientOrderID:         strings.TrimSpace(*clientOrderID),
			ExpireDate:            strings.TrimSpace(*expireDate),
			First:                 first,
			Second:                second,
			ConfirmHighValueOrder: *confirmHighValueOrder,
		}
		if !*yes {
			printDryRun("create-conditional", *account, accountSeq, map[string]any{"request": req})
			return
		}
		resp, err := c.TossClient.CreateConditionalOrder(ctx, accountSeq, req)
		if err != nil {
			log.Fatalf("create conditional order: %v", err)
		}
		printJSON(resp)

	case "modify-conditional":
		// Same rationale as "modify": show the existing watch/trigger state
		// before overwriting it, and fail fast if the ID doesn't exist.
		current, err := c.TossClient.GetConditionalOrder(ctx, accountSeq, *conditionalOrderID)
		if err != nil {
			log.Fatalf("look up current conditional order before modify: %v", err)
		}
		req := toss.ConditionalOrderModifyRequest{
			Type:                  strings.ToUpper(strings.TrimSpace(*condType)),
			Quantity:              strings.TrimSpace(*quantity),
			OrderType:             strings.ToUpper(strings.TrimSpace(*orderType)),
			ExpireDate:            strings.TrimSpace(*expireDate),
			First:                 first,
			Second:                second,
			ConfirmHighValueOrder: *confirmHighValueOrder,
		}
		if !*yes {
			printDryRun("modify-conditional", *account, accountSeq, map[string]any{"conditionalOrderId": *conditionalOrderID, "currentConditionalOrder": current, "proposedChange": req})
			return
		}
		resp, err := c.TossClient.ModifyConditionalOrder(ctx, accountSeq, *conditionalOrderID, req)
		if err != nil {
			log.Fatalf("modify conditional order: %v", err)
		}
		printJSON(resp)

	case "cancel-conditional":
		// Same rationale as "cancel": show what's actually being canceled
		// (symbol/type/trigger conditions/status), not a bare opaque ID.
		current, err := c.TossClient.GetConditionalOrder(ctx, accountSeq, *conditionalOrderID)
		if err != nil {
			log.Fatalf("look up current conditional order before cancel: %v", err)
		}
		if !*yes {
			printDryRun("cancel-conditional", *account, accountSeq, map[string]any{"conditionalOrderId": *conditionalOrderID, "currentConditionalOrder": current})
			return
		}
		if err := c.TossClient.CancelConditionalOrder(ctx, accountSeq, *conditionalOrderID); err != nil {
			log.Fatalf("cancel conditional order: %v", err)
		}
		fmt.Printf("conditional order %s canceled\n", *conditionalOrderID)
	}
}

func applyCreateConditionalDefaults(action, orderType, expireDate string, now time.Time) (string, string) {
	if action != "create-conditional" {
		return orderType, expireDate
	}
	if strings.TrimSpace(orderType) == "" {
		orderType = "MARKET"
	}
	if strings.TrimSpace(expireDate) == "" {
		expireDate = now.In(koreaStandardTime).AddDate(0, 0, 1).Format(time.DateOnly)
	}
	return orderType, expireDate
}

// buildSecondLeg builds the second ConditionRequest leg from flags, or
// returns nil if none of the second-leg flags were set (SINGLE orders have
// no second leg).
func buildSecondLeg(side, triggerPrice, orderPrice string) *toss.ConditionRequest {
	if strings.TrimSpace(side) == "" && strings.TrimSpace(triggerPrice) == "" && strings.TrimSpace(orderPrice) == "" {
		return nil
	}
	return &toss.ConditionRequest{
		OrderSide:    strings.ToUpper(strings.TrimSpace(side)),
		TriggerPrice: strings.TrimSpace(triggerPrice),
		OrderPrice:   strings.TrimSpace(orderPrice),
	}
}

// validateAction checks the required flags for the given action are present,
// giving a clear local usage error before touching the container or the
// Toss API. This mirrors (but is deliberately thinner than) the client's own
// structural validation in internal/toss/conditional_orders.go
// (validateConditionalOrderLegs) — that function is still the source of
// truth for leg-shape rules (SINGLE/OCO/OTO side and orderType constraints);
// this just catches missing-flag typos early.
func validateAction(
	action, orderID string, conditionalOrderID *string, orderType, quantity, symbol, condType,
	expireDate, firstSide, firstTriggerPrice string, second *toss.ConditionRequest,
) error {
	switch action {
	case "modify":
		if strings.TrimSpace(orderID) == "" {
			return fmt.Errorf("-order-id is required for -action modify")
		}
		if !isOrderType(orderType) {
			return fmt.Errorf("-order-type must be LIMIT or MARKET, got %q", orderType)
		}
	case "cancel":
		if strings.TrimSpace(orderID) == "" {
			return fmt.Errorf("-order-id is required for -action cancel")
		}
	case "create-conditional", "modify-conditional":
		if action == "create-conditional" && strings.TrimSpace(symbol) == "" {
			return fmt.Errorf("-symbol is required for -action create-conditional")
		}
		if action == "modify-conditional" && strings.TrimSpace(*conditionalOrderID) == "" {
			return fmt.Errorf("-conditional-order-id is required for -action modify-conditional")
		}
		normType := strings.ToUpper(strings.TrimSpace(condType))
		if normType != "SINGLE" && normType != "OCO" && normType != "OTO" {
			return fmt.Errorf("-type must be SINGLE, OCO, or OTO, got %q", condType)
		}
		if strings.TrimSpace(quantity) == "" {
			return fmt.Errorf("-quantity is required")
		}
		if !isOrderType(orderType) {
			return fmt.Errorf("-order-type must be LIMIT or MARKET, got %q", orderType)
		}
		if strings.TrimSpace(expireDate) == "" {
			return fmt.Errorf("-expire-date is required")
		}
		if strings.TrimSpace(firstSide) == "" {
			return fmt.Errorf("-first-side is required")
		}
		if strings.TrimSpace(firstTriggerPrice) == "" {
			return fmt.Errorf("-first-trigger-price is required")
		}
		switch normType {
		case "SINGLE":
			if second != nil {
				return fmt.Errorf("-type SINGLE must not set any -second-* flags")
			}
		case "OCO", "OTO":
			if second == nil {
				return fmt.Errorf("-type %s requires -second-side, -second-trigger-price (and -second-order-price for LIMIT)", normType)
			}
		}
	case "cancel-conditional":
		if strings.TrimSpace(*conditionalOrderID) == "" {
			return fmt.Errorf("-conditional-order-id is required for -action cancel-conditional")
		}
	}
	return nil
}

func isOrderType(s string) bool {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "LIMIT", "MARKET":
		return true
	default:
		return false
	}
}

// resolveTossAccount finds an account by exact name, falling back to a
// unique case-insensitive substring match — mirroring
// cmd/pm/helpers.go's resolveAccountByName (unexported there, not
// importable), matching this codebase's existing convention of small local
// re-derivations across cmd/* binaries (see normalizeKisAccountNo's doc
// comment in cmd/pm/sync.go). Requires the resolved account to be linked to
// a Toss accountSeq.
func resolveTossAccount(ctx context.Context, c *container.Container, name string) (models.Account, error) {
	name = strings.TrimSpace(name)
	accounts, err := c.Accounts.ListAll(ctx)
	if err != nil {
		return models.Account{}, fmt.Errorf("list accounts: %w", err)
	}

	lower := strings.ToLower(name)
	for _, a := range accounts {
		if strings.EqualFold(a.Name, name) {
			return validateTossLinked(a)
		}
	}

	var matches []models.Account
	for _, a := range accounts {
		if strings.Contains(strings.ToLower(a.Name), lower) {
			matches = append(matches, a)
		}
	}
	switch len(matches) {
	case 0:
		return models.Account{}, fmt.Errorf("no account matches %q", name)
	case 1:
		return validateTossLinked(matches[0])
	default:
		names := make([]string, len(matches))
		for i, a := range matches {
			names[i] = a.Name
		}
		return models.Account{}, fmt.Errorf("ambiguous account name %q matches multiple accounts: %s", name, strings.Join(names, ", "))
	}
}

func validateTossLinked(a models.Account) (models.Account, error) {
	if a.TossAccountSeq == nil {
		return models.Account{}, fmt.Errorf("account %q is not linked to a Toss accountSeq", a.Name)
	}
	return a, nil
}

func printDryRun(action, account, accountSeq string, fields map[string]any) {
	out, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		log.Fatalf("marshal dry-run preview: %v", err)
	}
	fmt.Printf("[DRY RUN] action=%s account=%s accountSeq=%s\n%s\n(rerun with -yes to actually call the Toss API)\n", action, account, accountSeq, out)
}

func printJSON(v any) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("marshal result: %v", err)
	}
	fmt.Println(string(out))
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	fmt.Fprint(os.Stderr, usage)
	os.Exit(2)
}
