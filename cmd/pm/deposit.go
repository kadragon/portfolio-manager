package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func runDeposit(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm deposit list|get|add|update|delete [flags]")
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return depositList(ctx, c)
	case "get":
		return depositGet(ctx, c, rest)
	case "add":
		return depositAdd(ctx, c, rest)
	case "update":
		return depositUpdate(ctx, c, rest)
	case "delete":
		return depositDelete(ctx, c, rest)
	default:
		return fmt.Errorf("unknown deposit verb %q", verb)
	}
}

func depositGet(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm deposit get", flag.ExitOnError)
	idRaw := fs.String("id", "", "deposit id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	deposit, err := c.Deposits.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get deposit: %w", err)
	}
	if deposit == nil {
		return fmt.Errorf("deposit %s not found", *idRaw)
	}
	return printJSON(deposit)
}

func depositList(ctx context.Context, c *container.Container) error {
	deposits, err := c.Deposits.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list deposits: %w", err)
	}
	return printJSON(deposits)
}

func depositAdd(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm deposit add", flag.ExitOnError)
	amountRaw := fs.String("amount", "", "deposit amount (required)")
	dateRaw := fs.String("date", "", "deposit date YYYY-MM-DD (required)")
	noteRaw := fs.String("note", "", "optional note")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*amountRaw) == "" {
		return fmt.Errorf("-amount is required")
	}
	if strings.TrimSpace(*dateRaw) == "" {
		return fmt.Errorf("-date is required")
	}
	amount, err := numeric.FromString(*amountRaw)
	if err != nil {
		return fmt.Errorf("invalid -amount: %w", err)
	}
	depositDate, err := datex.ParseDate(*dateRaw)
	if err != nil {
		return fmt.Errorf("invalid -date: %w", err)
	}

	noteStr := strings.TrimSpace(*noteRaw)
	note := sql.NullString{}
	var notePtr *sql.NullString
	if noteStr != "" {
		note = sql.NullString{String: noteStr, Valid: true}
		notePtr = &note
	}

	existing, err := c.Deposits.GetByDate(ctx, depositDate)
	if err != nil {
		return fmt.Errorf("get deposit by date: %w", err)
	}
	if existing != nil {
		updated, err := c.Deposits.Update(ctx, existing.ID, amount, depositDate, notePtr)
		if err != nil {
			return fmt.Errorf("update deposit: %w", err)
		}
		return printJSON(updated)
	}

	created, err := c.Deposits.Create(ctx, amount, depositDate, note)
	if err != nil {
		return fmt.Errorf("create deposit: %w", err)
	}
	return printJSON(created)
}

// depositUpdate applies only the flags the caller explicitly passed (via
// fs.Visit) on top of the existing deposit, mirroring the web edit form's
// partial-update and note-sentinel behaviour.
func depositUpdate(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm deposit update", flag.ExitOnError)
	idRaw := fs.String("id", "", "deposit id (required)")
	amountRaw := fs.String("amount", "", "new amount")
	dateRaw := fs.String("date", "", "new deposit date YYYY-MM-DD")
	noteRaw := fs.String("note", "", `new note; "/clear" nulls it`)
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })

	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	existing, err := c.Deposits.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get deposit: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("deposit %s not found", *idRaw)
	}

	amount := existing.Amount
	if seen["amount"] {
		d, err := numeric.FromString(*amountRaw)
		if err != nil {
			return fmt.Errorf("invalid -amount: %w", err)
		}
		amount = d
	}

	depositDate := existing.DepositDate
	if seen["date"] {
		d, err := datex.ParseDate(*dateRaw)
		if err != nil {
			return fmt.Errorf("invalid -date: %w", err)
		}
		depositDate = d
	}

	// Note sentinel (mirrors web handler): unset/empty=keep unchanged,
	// "/clear"=null, else set.
	var notePtr *sql.NullString
	if seen["note"] {
		trimmed := strings.TrimSpace(*noteRaw)
		if trimmed != "" {
			if strings.ToLower(trimmed) == "/clear" {
				ns := sql.NullString{}
				notePtr = &ns
			} else {
				ns := sql.NullString{String: trimmed, Valid: true}
				notePtr = &ns
			}
		}
	}

	updated, err := c.Deposits.Update(ctx, id, amount, depositDate, notePtr)
	if err != nil {
		return fmt.Errorf("update deposit: %w", err)
	}
	return printJSON(updated)
}

func depositDelete(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm deposit delete", flag.ExitOnError)
	idRaw := fs.String("id", "", "deposit id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	existing, err := c.Deposits.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get deposit: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("deposit %s not found", *idRaw)
	}
	if err := c.Deposits.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete deposit: %w", err)
	}
	return printJSON(map[string]string{"status": "deleted", "id": id.String()})
}
