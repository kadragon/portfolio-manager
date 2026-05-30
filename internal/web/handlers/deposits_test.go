package handlers_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/web/handlers"
)

func setupDeposits(t *testing.T) (*echo.Echo, *container.Container) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewDepositHandler(c).Register(e)
	return e, c
}

func seedDeposit(t *testing.T, c *container.Container, dateStr string, amountStr string, note string) models.Deposit {
	t.Helper()
	amt, _ := numeric.FromString(amountStr)
	d, err := datex.ParseDate(dateStr)
	if err != nil {
		t.Fatalf("parse date %s: %v", dateStr, err)
	}
	ns := sql.NullString{}
	if note != "" {
		ns = sql.NullString{String: note, Valid: true}
	}
	dep, err := c.Deposits.Create(context.Background(), amt, d, ns)
	if err != nil {
		t.Fatalf("seed deposit: %v", err)
	}
	return dep
}

// --- list ---

func TestDepositsListOK(t *testing.T) {
	e, c := setupDeposits(t)
	seedDeposit(t, c, "2026-01-01", "1000000", "월급")
	seedDeposit(t, c, "2026-02-01", "500000", "")

	rec := do(e, http.MethodGet, "/deposits", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "입금 내역") {
		t.Error("page title missing")
	}
	if !strings.Contains(body, "총투자원금") {
		t.Error("total stat missing")
	}
	if !strings.Contains(body, "₩1,500,000") {
		t.Errorf("total amount wrong, body=%s", body)
	}
	if !strings.Contains(body, "월급") {
		t.Error("note missing from list")
	}
}

func TestDepositsListEmpty(t *testing.T) {
	e, _ := setupDeposits(t)
	rec := do(e, http.MethodGet, "/deposits", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "입금 내역이 없습니다") {
		t.Error("empty state missing")
	}
}

// --- row ---

func TestDepositRowOK(t *testing.T) {
	e, c := setupDeposits(t)
	dep := seedDeposit(t, c, "2026-03-01", "300000", "보너스")

	rec := do(e, http.MethodGet, "/deposits/"+dep.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "2026-03-01") {
		t.Error("date missing")
	}
	if !strings.Contains(body, "보너스") {
		t.Error("note missing")
	}
}

func TestDepositRowNotFound(t *testing.T) {
	e, _ := setupDeposits(t)
	rec := do(e, http.MethodGet, "/deposits/00000000000000000000000000000000", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- create ---

func TestDepositCreateOK(t *testing.T) {
	e, c := setupDeposits(t)

	rec := do(e, http.MethodPost, "/deposits", url.Values{
		"deposit_date": {"2026-04-01"},
		"amount":       {"2000000"},
		"note":         {"세뱃돈"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "2026-04-01") {
		t.Error("date missing from created row")
	}

	all, _ := c.Deposits.ListAll(context.Background())
	if len(all) != 1 {
		t.Fatalf("want 1 deposit, got %d", len(all))
	}
}

func TestDepositCreateUpsertDuplicateDate(t *testing.T) {
	e, c := setupDeposits(t)
	seedDeposit(t, c, "2026-05-01", "100000", "")

	// Same date → upsert (update existing)
	rec := do(e, http.MethodPost, "/deposits", url.Values{
		"deposit_date": {"2026-05-01"},
		"amount":       {"999000"},
		"note":         {"수정됨"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("HX-Refresh") != "true" {
		t.Error("HX-Refresh header missing on upsert")
	}

	all, _ := c.Deposits.ListAll(context.Background())
	if len(all) != 1 {
		t.Fatalf("upsert should not create new row, got %d rows", len(all))
	}
	if all[0].Amount.String() != "999000" {
		t.Errorf("amount not updated: %s", all[0].Amount.String())
	}
}

func TestDepositCreateNoteStripped(t *testing.T) {
	e, c := setupDeposits(t)

	// Empty note stored as NULL
	rec := do(e, http.MethodPost, "/deposits", url.Values{
		"deposit_date": {"2026-06-01"},
		"amount":       {"50000"},
		"note":         {""},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	all, _ := c.Deposits.ListAll(context.Background())
	if len(all) != 1 {
		t.Fatalf("want 1 deposit")
	}
	if all[0].Note.Valid {
		t.Errorf("empty note should be NULL, got %+v", all[0].Note)
	}
}

// --- edit form ---

func TestDepositEditFormOK(t *testing.T) {
	e, c := setupDeposits(t)
	dep := seedDeposit(t, c, "2026-07-01", "400000", "여름 보너스")

	rec := do(e, http.MethodGet, "/deposits/"+dep.ID.String()+"/edit", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "여름 보너스") {
		t.Error("note missing from edit form")
	}
	if !strings.Contains(body, "저장") {
		t.Error("submit button missing")
	}
}

func TestDepositEditFormNotFound(t *testing.T) {
	e, _ := setupDeposits(t)
	rec := do(e, http.MethodGet, "/deposits/00000000000000000000000000000000/edit", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- update ---

func TestDepositUpdateOK(t *testing.T) {
	e, c := setupDeposits(t)
	dep := seedDeposit(t, c, "2026-08-01", "100000", "초기")

	rec := do(e, http.MethodPut, "/deposits/"+dep.ID.String(), url.Values{
		"deposit_date": {"2026-08-02"},
		"amount":       {"200000"},
		"note":         {"수정"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "수정") {
		t.Error("updated note missing")
	}
}

func TestDepositUpdateEmptyNoteKeepsExisting(t *testing.T) {
	e, c := setupDeposits(t)
	dep := seedDeposit(t, c, "2026-09-01", "100000", "유지될 메모")

	do(e, http.MethodPut, "/deposits/"+dep.ID.String(), url.Values{
		"deposit_date": {"2026-09-01"},
		"amount":       {"100000"},
		"note":         {""},
	})

	updated, _ := c.Deposits.GetByID(context.Background(), dep.ID)
	if !updated.Note.Valid || updated.Note.String != "유지될 메모" {
		t.Errorf("note changed on empty update: %+v", updated.Note)
	}
}

func TestDepositUpdateClearNote(t *testing.T) {
	e, c := setupDeposits(t)
	dep := seedDeposit(t, c, "2026-10-01", "100000", "삭제될 메모")

	do(e, http.MethodPut, "/deposits/"+dep.ID.String(), url.Values{
		"deposit_date": {"2026-10-01"},
		"amount":       {"100000"},
		"note":         {"/clear"},
	})

	updated, _ := c.Deposits.GetByID(context.Background(), dep.ID)
	if updated.Note.Valid {
		t.Errorf("note should be NULL after /clear, got %+v", updated.Note)
	}
}

// --- delete ---

func TestDepositDeleteOK(t *testing.T) {
	e, c := setupDeposits(t)
	dep := seedDeposit(t, c, "2026-11-01", "100000", "")

	rec := do(e, http.MethodDelete, "/deposits/"+dep.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	all, _ := c.Deposits.ListAll(context.Background())
	if len(all) != 0 {
		t.Fatalf("deposit not deleted, count=%d", len(all))
	}
}
