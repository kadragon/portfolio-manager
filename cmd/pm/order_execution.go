package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// orderExecutionOutput omits RawResponse because broker payloads can contain
// account metadata that should not be emitted by a general-purpose list command.
type orderExecutionOutput struct {
	ID        uuidx.UUID
	Ticker    string
	Side      string
	Quantity  int
	Currency  string
	Exchange  string
	Status    string
	Message   string
	OrderType string
	Price     *numeric.Decimal
	CreatedAt ktime.Time
}

func runOrderExecution(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm order-execution list [flags]")
	}
	switch args[0] {
	case "list":
		return orderExecutionList(ctx, c, args[1:])
	default:
		return fmt.Errorf("unknown order-execution verb %q", args[0])
	}
}

func orderExecutionList(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm order-execution list", flag.ExitOnError)
	limit := fs.Int("limit", 20, "maximum rows to return")
	ticker := fs.String("ticker", "", "exact ticker filter")
	status := fs.String("status", "", "exact status filter")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if *limit <= 0 {
		return fmt.Errorf("-limit must be positive")
	}
	records, err := c.OrderExecutions.List(ctx,
		strings.ToUpper(strings.TrimSpace(*ticker)),
		strings.ToLower(strings.TrimSpace(*status)),
		*limit,
	)
	if err != nil {
		return fmt.Errorf("list order executions: %w", err)
	}
	return printJSON(toOrderExecutionOutputs(records))
}

func toOrderExecutionOutputs(records []models.OrderExecutionRecord) []orderExecutionOutput {
	output := make([]orderExecutionOutput, len(records))
	for i, record := range records {
		output[i] = orderExecutionOutput{
			ID:        record.ID,
			Ticker:    record.Ticker,
			Side:      record.Side,
			Quantity:  record.Quantity,
			Currency:  record.Currency,
			Exchange:  record.Exchange,
			Status:    record.Status,
			Message:   record.Message,
			OrderType: record.OrderType,
			Price:     record.Price,
			CreatedAt: record.CreatedAt,
		}
	}
	return output
}
