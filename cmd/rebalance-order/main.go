// Command rebalance-order places a single market order for one account
// (KIS or Toss, whichever the account is linked to) and records the outcome
// in the order_executions table. It is invoked by the execute-rebalance-plan
// Claude Code skill, one order at a time, after the user confirms each order
// in conversation — this binary has no interactive confirmation of its own
// beyond the -yes flag, which is a defense-in-depth guard against accidental
// invocation, not a substitute for the skill asking the user first.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/services"
)

func main() {
	account := flag.String("account", "", "account name, exact or unique substring match (e.g. ISA, TOSS)")
	ticker := flag.String("ticker", "", "ticker symbol")
	side := flag.String("side", "", "buy or sell")
	qty := flag.Int("qty", 0, "share quantity (market order, no price)")
	exchange := flag.String("exchange", "", "overseas exchange order-code (NASD/NYSE/AMEX); leave empty for domestic")
	currency := flag.String("currency", "KRW", "currency recorded on the execution log (KRW/USD)")
	price := flag.String("price", "", "limit price; empty places a market order. A limit buy reserves exactly price×qty, unlike a market buy which reserves at the daily upper limit (KIS only; not supported for Toss)")
	yes := flag.Bool("yes", false, "actually place the order; without this flag, only prints a dry-run preview")
	flag.Parse()

	normSide := strings.ToLower(strings.TrimSpace(*side))
	if *account == "" || *ticker == "" || *qty <= 0 || (normSide != "buy" && normSide != "sell") {
		fmt.Fprintln(os.Stderr, "usage: rebalance-order -account NAME -ticker TICKER -side buy|sell -qty N [-exchange NASD] [-currency KRW] [-yes]")
		os.Exit(2)
	}

	env := strings.ToLower(strings.TrimSpace(os.Getenv("KIS_ENV")))
	if env == "" {
		env = "real"
	}
	fmt.Fprintf(os.Stderr, "KIS_ENV=%q — %s\n", env, envWarning(env))

	orderType := "market"
	if strings.TrimSpace(*price) != "" {
		orderType = "limit @ " + strings.TrimSpace(*price)
	}

	if !*yes {
		fmt.Printf("[DRY RUN] account=%s ticker=%s side=%s qty=%d exchange=%s currency=%s type=%s\n(rerun with -yes to actually place this order)\n",
			*account, *ticker, *side, *qty, *exchange, *currency, orderType)
		if strings.TrimSpace(*price) != "" {
			// The dry-run runs before account resolution, so it can't tell KIS
			// from Toss; warn that a limit price is KIS-only and -yes will reject
			// it for a Toss account, rather than letting the limit preview imply
			// the real run would go through.
			fmt.Println("(note: -price is KIS-only; a Toss-linked account will reject this on -yes)")
		}
		return
	}

	c, err := container.New("")
	if err != nil {
		log.Fatalf("init container: %v", err)
	}
	defer func() { _ = c.Close() }()

	var tossClient services.TossOrderClient
	if c.TossClient != nil {
		tossClient = c.TossClient
	}

	svc := services.NewOrderExecutionService(
		c.Accounts,
		c.OrderExecutions,
		func(keyID *int64, cano, acntPrdtCd string) (services.KISOrderClient, error) {
			return c.BuildKISOrderClient(keyID, cano, acntPrdtCd)
		},
		tossClient,
	)

	record, err := svc.PlaceOrder(context.Background(), *account, *ticker, *side, *qty, *exchange, *currency, strings.TrimSpace(*price))
	if err != nil {
		log.Fatalf("order not placed: %v", err)
	}

	out, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		log.Fatalf("marshal result: %v", err)
	}
	fmt.Println(string(out))

	if record.Status != "success" {
		os.Exit(1)
	}
}

func envWarning(env string) string {
	switch env {
	case "real":
		return "REAL trading — this places a live order with real money"
	case "demo", "vps", "paper":
		return "paper/demo trading — this will NOT place a live market order"
	default:
		return "UNRECOGNIZED value — KIS silently routes unknown KIS_ENV values to the REAL production API (see AGENTS.md)"
	}
}
