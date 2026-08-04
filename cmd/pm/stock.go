package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func runStock(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm stock list|get|add|update|move|delete [flags]")
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return stockList(ctx, c, rest)
	case "get":
		return stockGet(ctx, c, rest)
	case "add":
		return stockAdd(ctx, c, rest)
	case "update":
		return stockUpdate(ctx, c, rest)
	case "move":
		return stockMove(ctx, c, rest)
	case "delete":
		return stockDelete(ctx, c, rest)
	default:
		return fmt.Errorf("unknown stock verb %q", verb)
	}
}

func stockGet(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm stock get", flag.ExitOnError)
	idRaw := fs.String("id", "", "stock id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	stock, err := c.Stocks.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get stock: %w", err)
	}
	if stock == nil {
		return fmt.Errorf("stock %s not found", *idRaw)
	}
	return printJSON(stock)
}

func stockList(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm stock list", flag.ExitOnError)
	group := fs.String("group", "", "group UUID or name; omit to list all")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*group) != "" {
		g, err := resolveGroupRef(ctx, c, *group)
		if err != nil {
			return err
		}
		stocks, err := c.Stocks.ListByGroup(ctx, g.ID)
		if err != nil {
			return fmt.Errorf("list stocks by group: %w", err)
		}
		return printJSON(stocks)
	}
	stocks, err := c.Stocks.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list stocks: %w", err)
	}
	return printJSON(stocks)
}

func stockAdd(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm stock add", flag.ExitOnError)
	group := fs.String("group", "", "group UUID or name (required)")
	ticker := fs.String("ticker", "", "ticker symbol (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	g, err := resolveGroupRef(ctx, c, *group)
	if err != nil {
		return err
	}
	tick := strings.ToUpper(strings.TrimSpace(*ticker))
	if tick == "" {
		return fmt.Errorf("-ticker is required")
	}
	s, err := c.Stocks.Create(ctx, tick, g.ID)
	if err != nil {
		return fmt.Errorf("create stock: %w", err)
	}
	return printJSON(s)
}

func stockUpdate(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm stock update", flag.ExitOnError)
	idRaw := fs.String("id", "", "stock id (required)")
	ticker := fs.String("ticker", "", "new ticker")
	exchange := fs.String("exchange", "", "new exchange")
	name := fs.String("name", "", "new name")
	assetClass := fs.String("asset-class", "", "new asset class")
	securityGroup := fs.String("security-group", "", "new security group")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })

	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}

	if !seen["ticker"] && !seen["exchange"] && !seen["name"] && !seen["asset-class"] && !seen["security-group"] {
		return fmt.Errorf("no fields to update")
	}

	normalizedSecurityGroup := ""
	unknownSecurityGroup := false
	if seen["security-group"] {
		normalizedSecurityGroup = strings.ToUpper(strings.TrimSpace(*securityGroup))
		if normalizedSecurityGroup != "" && !models.ValidSecurityGroup(normalizedSecurityGroup) {
			// Unknown codes are accepted so a code KIS adds later is not rejected;
			// only malformed input is refused.
			if !models.WellFormedSecurityGroup(normalizedSecurityGroup) {
				return fmt.Errorf("invalid -security-group %q: expected a two-letter code", *securityGroup)
			}
			unknownSecurityGroup = true
		}
	}

	if seen["ticker"] {
		tick := strings.ToUpper(strings.TrimSpace(*ticker))
		if tick == "" {
			return fmt.Errorf("-ticker cannot be empty")
		}
		if _, err := c.Stocks.UpdateTicker(ctx, id, tick); err != nil {
			return fmt.Errorf("update stock ticker: %w", err)
		}
	}
	if seen["exchange"] {
		if _, err := c.Stocks.UpdateExchange(ctx, id, *exchange); err != nil {
			return fmt.Errorf("update stock exchange: %w", err)
		}
	}
	if seen["name"] {
		if _, err := c.Stocks.UpdateName(ctx, id, *name); err != nil {
			return fmt.Errorf("update stock name: %w", err)
		}
	}
	if seen["asset-class"] {
		if _, err := c.Stocks.UpdateAssetClass(ctx, id, *assetClass); err != nil {
			return fmt.Errorf("update stock asset class: %w", err)
		}
	}
	if seen["security-group"] {
		if _, err := c.Stocks.UpdateSecurityGroup(ctx, id, normalizedSecurityGroup); err != nil {
			return fmt.Errorf("update stock security group: %w", err)
		}
		if unknownSecurityGroup {
			fmt.Fprintf(os.Stderr, "warning: unknown KIS security group %q — accepted\n", normalizedSecurityGroup)
		}
	}

	s, err := c.Stocks.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get stock: %w", err)
	}
	return printJSON(s)
}

func stockMove(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm stock move", flag.ExitOnError)
	idRaw := fs.String("id", "", "stock id (required)")
	group := fs.String("group", "", "group UUID or name (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	g, err := resolveGroupRef(ctx, c, *group)
	if err != nil {
		return err
	}
	s, err := c.Stocks.UpdateGroup(ctx, id, g.ID)
	if err != nil {
		return fmt.Errorf("move stock: %w", err)
	}
	return printJSON(s)
}

func stockDelete(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm stock delete", flag.ExitOnError)
	idRaw := fs.String("id", "", "stock id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	existing, err := c.Stocks.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get stock: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("stock %s not found", *idRaw)
	}
	if err := c.Stocks.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete stock: %w", err)
	}
	return printJSON(map[string]string{"status": "deleted", "id": id.String()})
}
