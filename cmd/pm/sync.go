package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// normalizeKisAccountNo extracts the 8-digit CANO and 2-digit account-product
// code from a KIS account number string (e.g. "12345678-01" or "1234567801").
// Duplicated from internal/services.normalizeKisAccountNo (unexported there),
// matching this codebase's existing convention of small local re-derivations
// (see that function's own doc comment, and container.go's loadKISAccount).
func normalizeKisAccountNo(s string) (cano, acntPrdtCd string, err error) {
	var digits strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			digits.WriteRune(ch)
		}
	}
	d := digits.String()
	if len(d) != 10 {
		return "", "", fmt.Errorf("invalid KIS account number format (need 8+2 digits): %q", s)
	}
	return d[:8], d[8:], nil
}

// runSync syncs one account's holdings/cash from its linked broker (KIS or
// Toss, chosen by which link the account has), mirroring
// AccountHandler.syncAccount / syncTossAccount.
func runSync(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm sync", flag.ExitOnError)
	account := fs.String("account", "", "account name, exact or unique substring match (required)")
	confirmEmpty := fs.Bool("confirm-empty", false, "allow the sync to wipe holdings when the broker snapshot is empty")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	acct, err := resolveAccountByName(ctx, c, *account)
	if err != nil {
		return err
	}

	switch {
	case acct.TossAccountSeq != nil:
		if c.TossAccountSync == nil {
			return fmt.Errorf("toss sync service not configured (.env TOSS_CLIENT_ID/TOSS_CLIENT_SECRET)")
		}
		accountSeq := fmt.Sprintf("%d", *acct.TossAccountSeq)
		result, err := c.TossAccountSync.SyncAccount(ctx, acct, accountSeq, "", *confirmEmpty)
		if err != nil {
			return syncErr(err)
		}
		return printJSON(result)

	case acct.KisAccountNo != nil && strings.TrimSpace(*acct.KisAccountNo) != "":
		syncSvc := c.SyncServiceForKeyID(acct.KisAPIKeyID)
		if syncSvc == nil {
			return fmt.Errorf("KIS sync service not configured (.env KIS_CANO/KIS_ACNT_PRDT_CD)")
		}
		cano, acntPrdtCd, err := normalizeKisAccountNo(*acct.KisAccountNo)
		if err != nil {
			return err
		}
		result, err := syncSvc.SyncAccount(ctx, acct, cano, acntPrdtCd, *confirmEmpty)
		if err != nil {
			return syncErr(err)
		}
		return printJSON(result)

	default:
		return fmt.Errorf("account %q has no KIS account number or Toss accountSeq linked", acct.Name)
	}
}

// syncErr surfaces the "empty snapshot" guard as an actionable message
// pointing at -confirm-empty, matching the web UI's confirmation prompt.
func syncErr(err error) error {
	if services.IsKisEmptySnapshotError(err) {
		return fmt.Errorf("sync stopped: %w (rerun with -confirm-empty if the account is really empty)", err)
	}
	return fmt.Errorf("sync failed: %w", err)
}

func runClassifyStocks(ctx context.Context, c *container.Container, _ []string) error {
	if c.StockClassification == nil || !c.StockClassification.Enabled() {
		return fmt.Errorf("KIS stock classification service not configured (.env KIS_APP_KEY/KIS_APP_SECRET)")
	}
	res, err := c.StockClassification.ClassifyAll(ctx)
	if err != nil {
		return fmt.Errorf("classify stocks: %w", err)
	}
	return printJSON(res)
}
