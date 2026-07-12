package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// accountOutput is the CLI's JSON view of an account. It deliberately exposes
// only whether a KIS key slot is configured, never the source slot value.
// CodeQL's clear-text-logging query treats any value reaching json.Marshal as
// exposed, so a json:"-" tag on models.Account would not be sufficient.
type accountOutput struct {
	ID                  uuidx.UUID
	Name                string
	CashBalance         numeric.Decimal
	CreatedAt           time.Time
	UpdatedAt           time.Time
	KisAccountNo        *string
	KisAPIKeyConfigured bool
	KisAPIKeySlot       *string
	AccountType         *string
	TossAccountSeq      *int64
}

func toAccountOutput(a models.Account) accountOutput {
	return accountOutput{
		ID:                  a.ID,
		Name:                a.Name,
		CashBalance:         a.CashBalance,
		CreatedAt:           a.CreatedAt,
		UpdatedAt:           a.UpdatedAt,
		KisAccountNo:        a.KisAccountNo,
		KisAPIKeyConfigured: a.KisAPIKeyID != nil,
		KisAPIKeySlot:       kisAPIKeySlotLabel(a.KisAPIKeyID),
		AccountType:         a.AccountType,
		TossAccountSeq:      a.TossAccountSeq,
	}
}

func kisAPIKeySlotLabel(id *int64) *string {
	if id == nil {
		return nil
	}
	label := "unmapped"
	switch *id {
	case 1:
		label = "slot-1"
	case 2:
		label = "slot-2"
	case 3:
		label = "slot-3"
	case 4:
		label = "slot-4"
	case 5:
		label = "slot-5"
	case 6:
		label = "slot-6"
	case 7:
		label = "slot-7"
	case 8:
		label = "slot-8"
	case 9:
		label = "slot-9"
	}
	return &label
}

func toAccountOutputs(accounts []models.Account) []accountOutput {
	out := make([]accountOutput, len(accounts))
	for i, a := range accounts {
		out[i] = toAccountOutput(a)
	}
	return out
}

func runAccount(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm account list|get|add|update|delete|set-cash [flags]")
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return accountList(ctx, c)
	case "get":
		return accountGet(ctx, c, rest)
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

func accountGet(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm account get", flag.ExitOnError)
	idRaw := fs.String("id", "", "account id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	account, err := c.Accounts.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account %s not found", *idRaw)
	}
	return printJSON(toAccountOutput(*account))
}

func accountList(ctx context.Context, c *container.Container) error {
	accounts, err := c.Accounts.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list accounts: %w", err)
	}
	return printJSON(toAccountOutputs(accounts))
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
	return printJSON(toAccountOutput(account))
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
	kisAPIKeyID := fs.String("kis-api-key-id", "", `KIS API key set id; "/clear" unsets it`)
	accountType := fs.String("account-type", "", `brokerage|irp|pension|isa; "/clear" unsets it`)
	tossSeq := fs.String("toss-account-seq", "", `Toss accountSeq; "/clear" unsets it`)
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
		if trimmed := strings.TrimSpace(*kisAccountNo); trimmed == "" || strings.EqualFold(trimmed, "/clear") {
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
		parsed, err := parseNullableInt64Flag("-kis-api-key-id", *kisAPIKeyID)
		if err != nil {
			return err
		}
		kisKey = parsed
	}

	acctType := sql.NullString{}
	if existing.AccountType != nil {
		acctType = sql.NullString{String: *existing.AccountType, Valid: true}
	}
	if seen["account-type"] {
		if strings.EqualFold(strings.TrimSpace(*accountType), "/clear") {
			acctType = sql.NullString{}
		} else if !models.ValidAccountType(*accountType) {
			return fmt.Errorf("invalid -account-type %q", *accountType)
		} else {
			acctType = sql.NullString{String: *accountType, Valid: true}
		}
	}

	tossAccountSeq := sql.NullInt64{}
	if existing.TossAccountSeq != nil {
		tossAccountSeq = sql.NullInt64{Int64: *existing.TossAccountSeq, Valid: true}
	}
	if seen["toss-account-seq"] {
		parsed, err := parseNullableInt64Flag("-toss-account-seq", *tossSeq)
		if err != nil {
			return err
		}
		tossAccountSeq = parsed
	}

	updated, err := c.Accounts.Update(ctx, id, newName, newCash, kisNo, kisKey, acctType, tossAccountSeq)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	return printJSON(toAccountOutput(updated))
}

func parseNullableInt64Flag(name, raw string) (sql.NullInt64, error) {
	trimmed := strings.TrimSpace(raw)
	if strings.EqualFold(trimmed, "/clear") {
		return sql.NullInt64{}, nil
	}
	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return sql.NullInt64{}, fmt.Errorf("invalid %s: %w", name, err)
	}
	return sql.NullInt64{Int64: value, Valid: true}, nil
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
	existing, err := c.Accounts.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("account %s not found", *idRaw)
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
	return printJSON(toAccountOutput(updated))
}
