package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func runHolding(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm holding list|add|add-by-ticker|bulk|update|delete [flags]")
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return holdingList(ctx, c, rest)
	case "add":
		return holdingAdd(ctx, c, rest)
	case "add-by-ticker":
		return holdingAddByTicker(ctx, c, rest)
	case "bulk":
		return holdingBulk(ctx, c, rest)
	case "update":
		return holdingUpdate(ctx, c, rest)
	case "delete":
		return holdingDelete(ctx, c, rest)
	default:
		return fmt.Errorf("unknown holding verb %q", verb)
	}
}

func holdingList(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm holding list", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	acc, err := resolveAccountByName(ctx, c, *account)
	if err != nil {
		return err
	}
	holdings, err := c.Holdings.ListByAccount(ctx, acc.ID)
	if err != nil {
		return fmt.Errorf("list holdings: %w", err)
	}
	return printJSON(holdings)
}

func holdingAdd(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm holding add", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	stockRaw := fs.String("stock", "", "stock id (required)")
	qtyRaw := fs.String("qty", "", "quantity (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	acc, err := resolveAccountByName(ctx, c, *account)
	if err != nil {
		return err
	}
	stockID, err := uuidx.Parse(*stockRaw)
	if err != nil {
		return fmt.Errorf("invalid -stock: %w", err)
	}
	qty, err := numeric.FromString(*qtyRaw)
	if err != nil {
		return fmt.Errorf("invalid -qty: %w", err)
	}
	holding, err := c.Holdings.Create(ctx, acc.ID, stockID, qty)
	if err != nil {
		return fmt.Errorf("create holding: %w", err)
	}
	return printJSON(holding)
}

// holdingAddByTicker mirrors the web handler's createByTicker: normalize the
// ticker (trim + uppercase) and look it up. The CLI exposes no group input, so
// a ticker that does not yet exist cannot be auto-created (the web flow requires
// a group selection) and is reported as an error.
func holdingAddByTicker(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm holding add-by-ticker", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	tickerRaw := fs.String("ticker", "", "stock ticker (required)")
	qtyRaw := fs.String("qty", "", "quantity (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	acc, err := resolveAccountByName(ctx, c, *account)
	if err != nil {
		return err
	}
	ticker := strings.TrimSpace(strings.ToUpper(*tickerRaw))
	if ticker == "" {
		return fmt.Errorf("-ticker is required")
	}
	stock, err := c.Stocks.GetByTicker(ctx, ticker)
	if err != nil {
		return fmt.Errorf("get stock by ticker: %w", err)
	}
	if stock == nil {
		return fmt.Errorf("stock with ticker %q not found; create it first via 'pm stock add'", ticker)
	}
	qty, err := numeric.FromString(*qtyRaw)
	if err != nil {
		return fmt.Errorf("invalid -qty: %w", err)
	}
	holding, err := c.Holdings.Create(ctx, acc.ID, stock.ID, qty)
	if err != nil {
		return fmt.Errorf("create holding: %w", err)
	}
	return printJSON(holding)
}

func holdingBulk(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm holding bulk", flag.ExitOnError)
	account := fs.String("account", "", "account name (required)")
	updatesRaw := fs.String("updates", "", `comma-separated "id:qty" pairs (required)`)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	acc, err := resolveAccountByName(ctx, c, *account)
	if err != nil {
		return err
	}
	updates, err := parseHoldingUpdates(*updatesRaw)
	if err != nil {
		return err
	}
	if err := c.Holdings.BulkUpdateByAccount(ctx, acc.ID, updates); err != nil {
		return fmt.Errorf("bulk update holdings: %w", err)
	}
	return printJSON(map[string]any{"status": "updated", "count": len(updates)})
}

func parseHoldingUpdates(raw string) ([]repositories.HoldingUpdate, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("-updates is required")
	}
	pairs := strings.Split(raw, ",")
	updates := make([]repositories.HoldingUpdate, 0, len(pairs))
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idPart, qtyPart, ok := strings.Cut(pair, ":")
		if !ok {
			return nil, fmt.Errorf("invalid update %q: expected \"id:qty\"", pair)
		}
		id, err := uuidx.Parse(strings.TrimSpace(idPart))
		if err != nil {
			return nil, fmt.Errorf("invalid update %q: %w", pair, err)
		}
		qty, err := numeric.FromString(strings.TrimSpace(qtyPart))
		if err != nil {
			return nil, fmt.Errorf("invalid update %q: %w", pair, err)
		}
		updates = append(updates, repositories.HoldingUpdate{ID: id, Quantity: qty})
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("-updates is required")
	}
	return updates, nil
}

func holdingUpdate(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm holding update", flag.ExitOnError)
	idRaw := fs.String("id", "", "holding id (required)")
	qtyRaw := fs.String("qty", "", "quantity (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	qty, err := numeric.FromString(*qtyRaw)
	if err != nil {
		return fmt.Errorf("invalid -qty: %w", err)
	}
	holding, err := c.Holdings.Update(ctx, id, qty)
	if err != nil {
		return fmt.Errorf("update holding: %w", err)
	}
	return printJSON(holding)
}

func holdingDelete(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm holding delete", flag.ExitOnError)
	idRaw := fs.String("id", "", "holding id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	if err := c.Holdings.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete holding: %w", err)
	}
	return printJSON(map[string]string{"status": "deleted", "id": id.String()})
}
