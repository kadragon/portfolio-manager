package main

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func newAccountContainer(t *testing.T) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return container.NewWithQueries(sqlDB, q)
}

func TestAccountListEmpty(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	if err := runAccount(ctx, c, []string{"list"}); err != nil {
		t.Fatalf("account list (empty): %v", err)
	}
}

func TestAccountGet(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	acc, err := c.Accounts.Create(ctx, "ISA", numeric.Zero)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runAccount(ctx, c, []string{"get", "-id", acc.ID.String()}); err != nil {
		t.Fatalf("account get: %v", err)
	}
	if err := runAccount(ctx, c, []string{"get", "-id", "00000000000000000000000000000000"}); err == nil {
		t.Fatal("expected error for unknown account id")
	}
}

func TestAccountOutputIncludesCurrencyCashBalances(t *testing.T) {
	krw, _ := numeric.FromString("1000")
	usd, _ := numeric.FromString("2.5")
	out := toAccountOutput(models.Account{CashBalanceKRW: &krw, CashBalanceUSD: &usd})
	if out.CashBalanceKRW == nil || out.CashBalanceKRW.String() != "1000" ||
		out.CashBalanceUSD == nil || out.CashBalanceUSD.String() != "2.5" {
		t.Fatalf("currency cash = KRW %v USD %v", out.CashBalanceKRW, out.CashBalanceUSD)
	}
}

func TestAccountOutputPreservesUnknownCurrencyCashBalances(t *testing.T) {
	out := toAccountOutput(models.Account{})
	if out.CashBalanceKRW != nil || out.CashBalanceUSD != nil {
		t.Fatalf("unknown currency cash = KRW %v USD %v; want nil, nil",
			out.CashBalanceKRW, out.CashBalanceUSD)
	}
}

func TestAccountAdd(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)

	if err := runAccount(ctx, c, []string{"add", "-name", "ISA", "-cash", "10000"}); err != nil {
		t.Fatalf("account add: %v", err)
	}
	accounts, err := c.Accounts.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Name != "ISA" {
		t.Fatalf("expected one ISA account, got %+v", accounts)
	}
	if accounts[0].CashBalance.String() != "10000" {
		t.Fatalf("expected cash 10000, got %s", accounts[0].CashBalance.String())
	}
}

func TestAccountAddMissingName(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	if err := runAccount(ctx, c, []string{"add", "-cash", "100"}); err == nil {
		t.Fatal("expected error for missing -name")
	}
}

func TestAccountUpdatePartial(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	acc, err := c.Accounts.Create(ctx, "TOSS", numeric.FromInt(0))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := runAccount(ctx, c, []string{"update", "-id", acc.ID.String(), "-cash", "500"}); err != nil {
		t.Fatalf("account update -cash: %v", err)
	}
	updated, err := c.Accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Name != "TOSS" {
		t.Fatalf("expected name unchanged (TOSS), got %s", updated.Name)
	}
	if updated.CashBalance.String() != "500" {
		t.Fatalf("expected cash 500, got %s", updated.CashBalance.String())
	}
	if updated.CashBalanceKRW == nil || updated.CashBalanceKRW.String() != "500" ||
		updated.CashBalanceUSD == nil || !updated.CashBalanceUSD.IsZero() {
		t.Fatalf("manual currency cash = KRW %v USD %v", updated.CashBalanceKRW, updated.CashBalanceUSD)
	}

	if err := runAccount(ctx, c, []string{"update", "-id", acc.ID.String(), "-name", "TOSS renamed"}); err != nil {
		t.Fatalf("account update -name: %v", err)
	}
	updated, err = c.Accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Name != "TOSS renamed" {
		t.Fatalf("expected name updated, got %s", updated.Name)
	}
	if updated.CashBalance.String() != "500" {
		t.Fatalf("expected cash unchanged (500), got %s", updated.CashBalance.String())
	}
}

func TestAccountUpdateNamePreservesBrokerCashBreakdown(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	acc, err := c.Accounts.Create(ctx, "TOSS", numeric.Zero)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	total, _ := numeric.FromString("4500")
	krw, _ := numeric.FromString("1000")
	usd, _ := numeric.FromString("2.5")
	if _, err := c.Accounts.UpdateCashBalances(ctx, acc.ID, acc.Name, total, krw, usd); err != nil {
		t.Fatalf("seed broker cash: %v", err)
	}

	if err := runAccount(ctx, c, []string{"update", "-id", acc.ID.String(), "-name", "TOSS renamed"}); err != nil {
		t.Fatalf("account update -name: %v", err)
	}
	updated, err := c.Accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.CashBalance.String() != "4500" || updated.CashBalanceKRW == nil ||
		updated.CashBalanceKRW.String() != "1000" || updated.CashBalanceUSD == nil ||
		updated.CashBalanceUSD.String() != "2.5" {
		t.Fatalf("broker cash changed: total %s KRW %v USD %v",
			updated.CashBalance.String(), updated.CashBalanceKRW, updated.CashBalanceUSD)
	}
}

func TestAccountUpdateUnknownID(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	unknown := "00000000000000000000000000000000"
	if err := runAccount(ctx, c, []string{"update", "-id", unknown, "-name", "x"}); err == nil {
		t.Fatal("expected error for unknown account id")
	}
}

func TestAccountUpdateClearsNullableLinkage(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	acc, err := c.Accounts.Create(ctx, "TOSS", numeric.Zero)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runAccount(ctx, c, []string{
		"update", "-id", acc.ID.String(),
		"-kis-account-no", "1234567801",
		"-kis-api-key-id", "2",
		"-account-type", "isa",
		"-toss-account-seq", "123",
	}); err != nil {
		t.Fatalf("set linkage: %v", err)
	}
	seeded, err := c.Accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetByID seeded: %v", err)
	}
	if seeded.KisAPIKeyID == nil || *seeded.KisAPIKeyID != 2 || seeded.TossAccountSeq == nil || *seeded.TossAccountSeq != 123 {
		t.Fatalf("numeric linkage not stored: %+v", seeded)
	}

	if err := runAccount(ctx, c, []string{
		"update", "-id", acc.ID.String(),
		"-kis-account-no", "/clear",
		"-kis-api-key-id", "/clear",
		"-account-type", "/clear",
		"-toss-account-seq", "/clear",
	}); err != nil {
		t.Fatalf("clear linkage: %v", err)
	}
	got, err := c.Accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.KisAccountNo != nil || got.KisAPIKeyID != nil || got.AccountType != nil || got.TossAccountSeq != nil {
		t.Fatalf("linkage not cleared: %+v", got)
	}
}

func TestAccountUpdateClearingKisAccountNoCascadesKeyID(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	acc, err := c.Accounts.Create(ctx, "ISA", numeric.Zero)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runAccount(ctx, c, []string{
		"update", "-id", acc.ID.String(),
		"-kis-account-no", "1234567801",
		"-kis-api-key-id", "2",
	}); err != nil {
		t.Fatalf("set linkage: %v", err)
	}

	// Clearing kis-account-no without mentioning kis-api-key-id must cascade,
	// leaving no orphaned key id behind.
	if err := runAccount(ctx, c, []string{
		"update", "-id", acc.ID.String(),
		"-kis-account-no", "/clear",
	}); err != nil {
		t.Fatalf("clear kis-account-no: %v", err)
	}
	got, err := c.Accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.KisAccountNo != nil {
		t.Fatalf("kis_account_no not cleared: %+v", got.KisAccountNo)
	}
	if got.KisAPIKeyID != nil {
		t.Fatalf("kis_api_key_id left orphaned after clearing kis-account-no: %+v", got.KisAPIKeyID)
	}
}

func TestAccountUpdateExplicitKeyIDWinsOverCascade(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	acc, err := c.Accounts.Create(ctx, "ISA", numeric.Zero)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runAccount(ctx, c, []string{
		"update", "-id", acc.ID.String(),
		"-kis-account-no", "1234567801",
		"-kis-api-key-id", "2",
	}); err != nil {
		t.Fatalf("set linkage: %v", err)
	}

	// An explicit -kis-api-key-id in the same call wins over the cascade.
	if err := runAccount(ctx, c, []string{
		"update", "-id", acc.ID.String(),
		"-kis-account-no", "/clear",
		"-kis-api-key-id", "5",
	}); err != nil {
		t.Fatalf("clear kis-account-no with explicit key id: %v", err)
	}
	got, err := c.Accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.KisAPIKeyID == nil || *got.KisAPIKeyID != 5 {
		t.Fatalf("explicit kis_api_key_id not applied: %+v", got.KisAPIKeyID)
	}
}

func TestAccountDelete(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	acc, err := c.Accounts.Create(ctx, "ISA", numeric.FromInt(0))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runAccount(ctx, c, []string{"delete", "-id", acc.ID.String()}); err != nil {
		t.Fatalf("account delete: %v", err)
	}
	got, err := c.Accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected account deleted, still found: %+v", got)
	}
}

func TestAccountDeleteUnknownID(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	if err := runAccount(ctx, c, []string{"delete", "-id", "00000000000000000000000000000000"}); err == nil {
		t.Fatal("expected error for unknown account id")
	}
}

func TestAccountSetCash(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	acc, err := c.Accounts.Create(ctx, "ISA", numeric.FromInt(0))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := runAccount(ctx, c, []string{"set-cash", "-id", acc.ID.String(), "-cash", "999"}); err != nil {
		t.Fatalf("account set-cash: %v", err)
	}
	updated, err := c.Accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.CashBalance.String() != "999" {
		t.Fatalf("expected cash 999, got %s", updated.CashBalance.String())
	}
	if updated.CashBalanceKRW == nil || updated.CashBalanceKRW.String() != "999" ||
		updated.CashBalanceUSD == nil || !updated.CashBalanceUSD.IsZero() {
		t.Fatalf("manual currency cash = KRW %v USD %v", updated.CashBalanceKRW, updated.CashBalanceUSD)
	}
}

func TestAccountUnknownVerb(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	if err := runAccount(ctx, c, []string{"bogus"}); err == nil {
		t.Fatal("expected error for unknown verb")
	}
}

func TestAccountNoArgs(t *testing.T) {
	ctx := context.Background()
	c := newAccountContainer(t)
	if err := runAccount(ctx, c, nil); err == nil {
		t.Fatal("expected usage error for no args")
	}
}
