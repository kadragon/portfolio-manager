package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/toss"
)

// runToss dispatches "pm toss <verb>" subcommands. Only read-only Toss Open
// API queries are exposed here; write/mutating operations (order placement,
// modification, cancellation) are intentionally out of scope for pm.
func runToss(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return tossHelp()
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "help", "-h", "--help":
		return tossHelp()
	case "orderbook":
		return tossOrderbook(ctx, c, rest)
	case "prices":
		return tossPrices(ctx, c, rest)
	case "trades":
		return tossTrades(ctx, c, rest)
	case "price-limit":
		return tossPriceLimit(ctx, c, rest)
	case "candles":
		return tossCandles(ctx, c, rest)
	case "stocks":
		return tossStocks(ctx, c, rest)
	case "stock-warnings":
		return tossStockWarnings(ctx, c, rest)
	case "exchange-rate":
		return tossExchangeRate(ctx, c, rest)
	case "market-calendar-kr":
		return tossMarketCalendarKR(ctx, c, rest)
	case "market-calendar-us":
		return tossMarketCalendarUS(ctx, c, rest)
	case "rankings":
		return tossRankings(ctx, c, rest)
	case "indicator-prices":
		return tossIndicatorPrices(ctx, c, rest)
	case "indicator-candles":
		return tossIndicatorCandles(ctx, c, rest)
	case "investor-trading":
		return tossInvestorTrading(ctx, c, rest)
	case "accounts":
		return tossAccounts(ctx, c, rest)
	case "holdings":
		return tossHoldings(ctx, c, rest)
	case "orders":
		return tossOrders(ctx, c, rest)
	case "order":
		return tossOrder(ctx, c, rest)
	case "conditional-orders":
		return tossConditionalOrders(ctx, c, rest)
	case "conditional-order":
		return tossConditionalOrder(ctx, c, rest)
	case "buying-power":
		return tossBuyingPower(ctx, c, rest)
	case "sellable-quantity":
		return tossSellableQuantity(ctx, c, rest)
	case "commissions":
		return tossCommissions(ctx, c, rest)
	default:
		return fmt.Errorf("unknown toss verb %q", verb)
	}
}

func tossHelp() error {
	fmt.Println(`usage: pm toss <verb> [flags]

market data (no account required):
  orderbook          -symbol T
  prices             -symbols T1,T2,...
  trades             -symbol T [-count N]
  price-limit        -symbol T
  candles            -symbol T -interval 1m|1d [-count N] [-before RFC3339] [-adjusted true|false]
  stocks             -symbols T1,T2,...
  stock-warnings     -symbol T
  exchange-rate      -base CCY -quote CCY [-at RFC3339]
  market-calendar-kr [-date YYYY-MM-DD]
  market-calendar-us [-date YYYY-MM-DD]
  rankings           -type TYPE -market KR|US -duration D [-exclude-caution] [-count N]
  indicator-prices   -symbols S1,S2,...
  indicator-candles  -symbol S -interval 1m|1d [-count N] [-before RFC3339]
  investor-trading   -symbol KOSPI|KOSDAQ -interval 1d|1w|1mo|1y [-count N] [-until YYYY-MM-DD]

account/order data (require -account NAME, resolved to a Toss-linked account):
  accounts
  holdings            -account NAME [-symbol T]
  orders              -account NAME -status OPEN|CLOSED [-symbol T] [-from DATE] [-to DATE] [-cursor C] [-limit N]
  order               -account NAME -order-id ID
  conditional-orders  -account NAME -status OPEN|CLOSED [-symbol T] [-cursor C] [-limit N]
  conditional-order   -account NAME -conditional-order-id ID
  buying-power        -account NAME -currency KRW|USD
  sellable-quantity   -account NAME -symbol T
  commissions         -account NAME

Every subcommand prints indented JSON to stdout and exits non-zero on error.`)
	return nil
}

// requireToss returns c.TossClient, or an error if Toss isn't configured.
func requireToss(c *container.Container) (*toss.Client, error) {
	if c.TossClient == nil {
		return nil, fmt.Errorf("toss client not configured (.env TOSS_CLIENT_ID/TOSS_CLIENT_SECRET)")
	}
	return c.TossClient, nil
}

// tossAccountSeq resolves name to an account and returns its Toss accountSeq
// as a string, or an error if the account isn't Toss-linked.
func tossAccountSeq(ctx context.Context, c *container.Container, name string) (string, error) {
	acct, err := resolveAccountByName(ctx, c, name)
	if err != nil {
		return "", err
	}
	if acct.TossAccountSeq == nil {
		return "", fmt.Errorf("account %q is not linked to a Toss accountSeq", acct.Name)
	}
	return strconv.FormatInt(*acct.TossAccountSeq, 10), nil
}

// splitSymbols splits a comma-separated flag value into a trimmed,
// non-empty-entry slice.
func splitSymbols(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseTimeFlag parses raw as RFC3339, returning the zero time for an empty
// string (meaning "omit this parameter").
func parseTimeFlag(name, raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s: %w", name, err)
	}
	return t, nil
}

func tossOrderbook(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss orderbook", flag.ExitOnError)
	symbol := fs.String("symbol", "", "symbol (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*symbol) == "" {
		return fmt.Errorf("-symbol is required")
	}
	result, err := client.GetOrderbook(ctx, *symbol)
	if err != nil {
		return fmt.Errorf("toss orderbook: %w", err)
	}
	return printJSON(result)
}

func tossPrices(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss prices", flag.ExitOnError)
	symbols := fs.String("symbols", "", "comma-separated symbols (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	syms := splitSymbols(*symbols)
	if len(syms) == 0 {
		return fmt.Errorf("-symbols is required")
	}
	result, err := client.GetPrices(ctx, syms)
	if err != nil {
		return fmt.Errorf("toss prices: %w", err)
	}
	return printJSON(result)
}

func tossTrades(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss trades", flag.ExitOnError)
	symbol := fs.String("symbol", "", "symbol (required)")
	count := fs.Int("count", 0, "max trades to return (0 = server default)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*symbol) == "" {
		return fmt.Errorf("-symbol is required")
	}
	result, err := client.GetTrades(ctx, *symbol, *count)
	if err != nil {
		return fmt.Errorf("toss trades: %w", err)
	}
	return printJSON(result)
}

func tossPriceLimit(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss price-limit", flag.ExitOnError)
	symbol := fs.String("symbol", "", "symbol (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*symbol) == "" {
		return fmt.Errorf("-symbol is required")
	}
	result, err := client.GetPriceLimit(ctx, *symbol)
	if err != nil {
		return fmt.Errorf("toss price-limit: %w", err)
	}
	return printJSON(result)
}

func tossCandles(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss candles", flag.ExitOnError)
	symbol := fs.String("symbol", "", "symbol (required)")
	interval := fs.String("interval", "", "1m or 1d (required)")
	count := fs.Int("count", 0, "max candles to return (0 = server default)")
	before := fs.String("before", "", "RFC3339 pagination cursor (omit for most recent page)")
	// GetCandles's nil-adjusted path only exists to omit the query param and
	// let the server default (also true) apply; always passing &value with a
	// true default is behaviorally equivalent and simpler for a CLI flag.
	adjusted := fs.Bool("adjusted", true, "apply split/dividend adjustment")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*symbol) == "" {
		return fmt.Errorf("-symbol is required")
	}
	if strings.TrimSpace(*interval) == "" {
		return fmt.Errorf("-interval is required")
	}
	beforeTime, err := parseTimeFlag("-before", *before)
	if err != nil {
		return err
	}
	result, err := client.GetCandles(ctx, *symbol, *interval, *count, beforeTime, adjusted)
	if err != nil {
		return fmt.Errorf("toss candles: %w", err)
	}
	return printJSON(result)
}

func tossStocks(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss stocks", flag.ExitOnError)
	symbols := fs.String("symbols", "", "comma-separated symbols (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	syms := splitSymbols(*symbols)
	if len(syms) == 0 {
		return fmt.Errorf("-symbols is required")
	}
	result, err := client.GetStocks(ctx, syms)
	if err != nil {
		return fmt.Errorf("toss stocks: %w", err)
	}
	return printJSON(result)
}

func tossStockWarnings(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss stock-warnings", flag.ExitOnError)
	symbol := fs.String("symbol", "", "symbol (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*symbol) == "" {
		return fmt.Errorf("-symbol is required")
	}
	result, err := client.GetStockWarnings(ctx, *symbol)
	if err != nil {
		return fmt.Errorf("toss stock-warnings: %w", err)
	}
	return printJSON(result)
}

func tossExchangeRate(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss exchange-rate", flag.ExitOnError)
	base := fs.String("base", "", "base currency (required)")
	quote := fs.String("quote", "", "quote currency (required)")
	at := fs.String("at", "", "RFC3339 timestamp (omit for current rate)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*base) == "" || strings.TrimSpace(*quote) == "" {
		return fmt.Errorf("-base and -quote are required")
	}
	atTime, err := parseTimeFlag("-at", *at)
	if err != nil {
		return err
	}
	result, err := client.GetExchangeRate(ctx, *base, *quote, atTime)
	if err != nil {
		return fmt.Errorf("toss exchange-rate: %w", err)
	}
	return printJSON(result)
}

func tossMarketCalendarKR(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss market-calendar-kr", flag.ExitOnError)
	date := fs.String("date", "", "YYYY-MM-DD (omit for today)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	result, err := client.GetKrMarketCalendar(ctx, *date)
	if err != nil {
		return fmt.Errorf("toss market-calendar-kr: %w", err)
	}
	return printJSON(result)
}

func tossMarketCalendarUS(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss market-calendar-us", flag.ExitOnError)
	date := fs.String("date", "", "YYYY-MM-DD (omit for today)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	result, err := client.GetUsMarketCalendar(ctx, *date)
	if err != nil {
		return fmt.Errorf("toss market-calendar-us: %w", err)
	}
	return printJSON(result)
}

func tossRankings(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss rankings", flag.ExitOnError)
	rankType := fs.String("type", "", "ranking type (required)")
	market := fs.String("market", "", "KR or US (required)")
	duration := fs.String("duration", "", "ranking window (required)")
	excludeCaution := fs.Bool("exclude-caution", false, "exclude investment-caution stocks")
	count := fs.Int("count", 0, "max results (0 = server default)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*rankType) == "" || strings.TrimSpace(*market) == "" || strings.TrimSpace(*duration) == "" {
		return fmt.Errorf("-type, -market, and -duration are required")
	}
	result, err := client.GetRankings(ctx, toss.RankingParams{
		Type:                     *rankType,
		MarketCountry:            *market,
		Duration:                 *duration,
		ExcludeInvestmentCaution: *excludeCaution,
		Count:                    *count,
	})
	if err != nil {
		return fmt.Errorf("toss rankings: %w", err)
	}
	return printJSON(result)
}

func tossIndicatorPrices(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss indicator-prices", flag.ExitOnError)
	symbols := fs.String("symbols", "", "comma-separated indicator symbols (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	syms := splitSymbols(*symbols)
	if len(syms) == 0 {
		return fmt.Errorf("-symbols is required")
	}
	result, err := client.GetMarketIndicatorPrices(ctx, syms)
	if err != nil {
		return fmt.Errorf("toss indicator-prices: %w", err)
	}
	return printJSON(result)
}

func tossIndicatorCandles(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss indicator-candles", flag.ExitOnError)
	symbol := fs.String("symbol", "", "indicator symbol (required)")
	interval := fs.String("interval", "", "1m or 1d (required)")
	count := fs.Int("count", 0, "max candles to return (0 = server default)")
	before := fs.String("before", "", "RFC3339 pagination cursor (omit for most recent page)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*symbol) == "" {
		return fmt.Errorf("-symbol is required")
	}
	if strings.TrimSpace(*interval) == "" {
		return fmt.Errorf("-interval is required")
	}
	beforeTime, err := parseTimeFlag("-before", *before)
	if err != nil {
		return err
	}
	result, err := client.GetMarketIndicatorCandles(ctx, *symbol, *interval, *count, beforeTime)
	if err != nil {
		return fmt.Errorf("toss indicator-candles: %w", err)
	}
	return printJSON(result)
}

func tossInvestorTrading(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss investor-trading", flag.ExitOnError)
	symbol := fs.String("symbol", "", "KOSPI or KOSDAQ (required)")
	interval := fs.String("interval", "", "1d, 1w, 1mo, or 1y (required)")
	count := fs.Int("count", 0, "max records to return (0 = server default)")
	until := fs.String("until", "", "YYYY-MM-DD pagination cursor (omit for most recent records)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*symbol) == "" {
		return fmt.Errorf("-symbol is required")
	}
	if strings.TrimSpace(*interval) == "" {
		return fmt.Errorf("-interval is required")
	}
	result, err := client.GetMarketIndicatorInvestorTrading(ctx, *symbol, *interval, *count, *until)
	if err != nil {
		return fmt.Errorf("toss investor-trading: %w", err)
	}
	return printJSON(result)
}

func tossAccounts(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss accounts", flag.ExitOnError)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	result, err := client.GetAccounts(ctx)
	if err != nil {
		return fmt.Errorf("toss accounts: %w", err)
	}
	return printJSON(result)
}

func tossHoldings(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss holdings", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	symbol := fs.String("symbol", "", "filter to a single symbol")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	accountSeq, err := tossAccountSeq(ctx, c, *account)
	if err != nil {
		return err
	}
	result, err := client.GetHoldings(ctx, accountSeq, *symbol)
	if err != nil {
		return fmt.Errorf("toss holdings: %w", err)
	}
	return printJSON(result)
}

func tossOrders(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss orders", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	status := fs.String("status", "", "OPEN or CLOSED (required)")
	symbol := fs.String("symbol", "", "filter to a single symbol")
	from := fs.String("from", "", "YYYY-MM-DD range start")
	to := fs.String("to", "", "YYYY-MM-DD range end")
	cursor := fs.String("cursor", "", "pagination cursor")
	limit := fs.Int("limit", 0, "max results (0 = server default)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*status) == "" {
		return fmt.Errorf("-status is required")
	}
	accountSeq, err := tossAccountSeq(ctx, c, *account)
	if err != nil {
		return err
	}
	result, err := client.GetOrders(ctx, accountSeq, toss.OrderListParams{
		Status: *status,
		Symbol: *symbol,
		From:   *from,
		To:     *to,
		Cursor: *cursor,
		Limit:  *limit,
	})
	if err != nil {
		return fmt.Errorf("toss orders: %w", err)
	}
	return printJSON(result)
}

func tossOrder(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss order", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	orderID := fs.String("order-id", "", "order id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*orderID) == "" {
		return fmt.Errorf("-order-id is required")
	}
	accountSeq, err := tossAccountSeq(ctx, c, *account)
	if err != nil {
		return err
	}
	result, err := client.GetOrder(ctx, accountSeq, *orderID)
	if err != nil {
		return fmt.Errorf("toss order: %w", err)
	}
	return printJSON(result)
}

func tossConditionalOrders(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss conditional-orders", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	status := fs.String("status", "", "OPEN or CLOSED (required)")
	symbol := fs.String("symbol", "", "filter to a single symbol")
	cursor := fs.String("cursor", "", "pagination cursor")
	limit := fs.Int("limit", 0, "max results (0 = server default)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*status) == "" {
		return fmt.Errorf("-status is required")
	}
	accountSeq, err := tossAccountSeq(ctx, c, *account)
	if err != nil {
		return err
	}
	result, err := client.GetConditionalOrders(ctx, accountSeq, toss.ConditionalOrderListParams{
		Status: *status,
		Symbol: *symbol,
		Cursor: *cursor,
		Limit:  *limit,
	})
	if err != nil {
		return fmt.Errorf("toss conditional-orders: %w", err)
	}
	return printJSON(result)
}

func tossConditionalOrder(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss conditional-order", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	conditionalOrderID := fs.String("conditional-order-id", "", "conditional order id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*conditionalOrderID) == "" {
		return fmt.Errorf("-conditional-order-id is required")
	}
	accountSeq, err := tossAccountSeq(ctx, c, *account)
	if err != nil {
		return err
	}
	result, err := client.GetConditionalOrder(ctx, accountSeq, *conditionalOrderID)
	if err != nil {
		return fmt.Errorf("toss conditional-order: %w", err)
	}
	return printJSON(result)
}

func tossBuyingPower(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss buying-power", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	currency := fs.String("currency", "", "KRW or USD (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*currency) == "" {
		return fmt.Errorf("-currency is required")
	}
	accountSeq, err := tossAccountSeq(ctx, c, *account)
	if err != nil {
		return err
	}
	result, err := client.GetBuyingPower(ctx, accountSeq, *currency)
	if err != nil {
		return fmt.Errorf("toss buying-power: %w", err)
	}
	return printJSON(result)
}

func tossSellableQuantity(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss sellable-quantity", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	symbol := fs.String("symbol", "", "symbol (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*symbol) == "" {
		return fmt.Errorf("-symbol is required")
	}
	accountSeq, err := tossAccountSeq(ctx, c, *account)
	if err != nil {
		return err
	}
	result, err := client.GetSellableQuantity(ctx, accountSeq, *symbol)
	if err != nil {
		return fmt.Errorf("toss sellable-quantity: %w", err)
	}
	return printJSON(result)
}

func tossCommissions(ctx context.Context, c *container.Container, args []string) error {
	client, err := requireToss(c)
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("pm toss commissions", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	accountSeq, err := tossAccountSeq(ctx, c, *account)
	if err != nil {
		return err
	}
	result, err := client.GetCommissions(ctx, accountSeq)
	if err != nil {
		return fmt.Errorf("toss commissions: %w", err)
	}
	return printJSON(result)
}
