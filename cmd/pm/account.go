package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func runAccount(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm account list|add|update|delete|set-cash [flags]")
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return accountList(ctx, c)
	case "add":
		return accountAdd(ctx, c, rest)
	case "update":
		return accountUpdate(ctx, c, rest)
	case "delete":
		return accountDelete(ctx, c, rest)
	case "set-cash":
		return accountSetCash(ctx, c, rest)
	default:
		return fmt.Errorf("unknown account verb %q", verb)
	}
}

func accountList(ctx context.Context, c *container.Container) error {
	accounts, err := c.Accounts.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list accounts: %w", err)
	}
	return printJSON(accounts)
}

func accountAdd(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm account add", flag.ExitOnError)
	name := fs.String("name", "", "account name (required)")
	cash := fs.String("cash", "0", "initial cash balance")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("-name is required")
	}
	cashDec, err := numeric.FromString(*cash)
	if err != nil {
		return fmt.Errorf("invalid -cash: %w", err)
	}
	account, err := c.Accounts.Create(ctx, *name, cashDec)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	return printJSON(account)
}

// accountUpdate applies only the flags the caller explicitly passed
// (via fs.Visit), preserving every other field on the existing account —
// mirroring the web edit form's partial-update behaviour.
func accountUpdate(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm account update", flag.ExitOnError)
	idRaw := fs.String("id", "", "account id (required)")
	name := fs.String("name", "", "new name")
	cash := fs.String("cash", "", "new cash balance")
	kisAccountNo := fs.String("kis-account-no", "", "KIS account number (8+2 digits); empty clears it")
	kisAPIKeyID := fs.Int64("kis-api-key-id", 0, "KIS API key set id")
	accountType := fs.String("account-type", "", "brokerage|irp|pension|isa")
	tossSeq := fs.Int64("toss-account-seq", 0, "Toss accountSeq")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })

	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	existing, err := c.Accounts.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("account %s not found", *idRaw)
	}

	newName := existing.Name
	if seen["name"] {
		newName = *name
	}

	newCash := existing.CashBalance
	if seen["cash"] {
		d, err := numeric.FromString(*cash)
		if err != nil {
			return fmt.Errorf("invalid -cash: %w", err)
		}
		newCash = d
	}

	kisNo := sql.NullString{}
	if existing.KisAccountNo != nil {
		kisNo = sql.NullString{String: *existing.KisAccountNo, Valid: true}
	}
	if seen["kis-account-no"] {
		if trimmed := strings.TrimSpace(*kisAccountNo); trimmed == "" {
			kisNo = sql.NullString{}
		} else {
			kisNo = sql.NullString{String: trimmed, Valid: true}
		}
	}

	kisKey := sql.NullInt64{}
	if existing.KisAPIKeyID != nil {
		kisKey = sql.NullInt64{Int64: *existing.KisAPIKeyID, Valid: true}
	}
	if seen["kis-api-key-id"] {
		kisKey = sql.NullInt64{Int64: *kisAPIKeyID, Valid: true}
	}

	acctType := sql.NullString{}
	if existing.AccountType != nil {
		acctType = sql.NullString{String: *existing.AccountType, Valid: true}
	}
	if seen["account-type"] {
		if !models.ValidAccountType(*accountType) {
			return fmt.Errorf("invalid -account-type %q", *accountType)
		}
		acctType = sql.NullString{String: *accountType, Valid: true}
	}

	tossAccountSeq := sql.NullInt64{}
	if existing.TossAccountSeq != nil {
		tossAccountSeq = sql.NullInt64{Int64: *existing.TossAccountSeq, Valid: true}
	}
	if seen["toss-account-seq"] {
		tossAccountSeq = sql.NullInt64{Int64: *tossSeq, Valid: true}
	}

	updated, err := c.Accounts.Update(ctx, id, newName, newCash, kisNo, kisKey, acctType, tossAccountSeq)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	return printJSON(updated)
}

func accountDelete(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm account delete", flag.ExitOnError)
	idRaw := fs.String("id", "", "account id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	if err := c.Accounts.DeleteWithHoldings(ctx, id); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return printJSON(map[string]string{"status": "deleted", "id": id.String()})
}

func accountSetCash(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm account set-cash", flag.ExitOnError)
	idRaw := fs.String("id", "", "account id (required)")
	cash := fs.String("cash", "", "new cash balance (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	existing, err := c.Accounts.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("account %s not found", *idRaw)
	}
	d, err := numeric.FromString(*cash)
	if err != nil {
		return fmt.Errorf("invalid -cash: %w", err)
	}
	updated, err := c.Accounts.UpdateNameCash(ctx, id, existing.Name, d)
	if err != nil {
		return fmt.Errorf("update account cash: %w", err)
	}
	return printJSON(updated)
}
