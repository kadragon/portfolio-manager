package handlers

import (
	"database/sql"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/kadragon/portfolio-manager/internal/web/templates"
)

// AccountHandler handles /accounts routes.
type AccountHandler struct {
	c *container.Container
}

// NewAccountHandler creates a handler using the given container.
func NewAccountHandler(c *container.Container) *AccountHandler {
	return &AccountHandler{c: c}
}

// Register wires all account routes onto e.
func (h *AccountHandler) Register(e *echo.Echo) {
	e.GET("/accounts", h.list)
	e.POST("/accounts", h.create)
	e.PUT("/accounts/bulk-cash", h.bulkCash)
	e.GET("/accounts/:id", h.row)
	e.GET("/accounts/:id/edit", h.editForm)
	e.PUT("/accounts/:id", h.update)
	e.DELETE("/accounts/:id", h.delete)
	e.POST("/accounts/:id/sync", h.syncAccount)
	e.POST("/accounts/classify-stocks", h.classifyStocks)
}

func (h *AccountHandler) list(c echo.Context) error {
	accounts, err := h.c.Accounts.ListAll(c.Request().Context())
	if err != nil {
		return err
	}
	return templates.AccountsPage(accounts).Render(c.Request().Context(), c.Response().Writer)
}

func (h *AccountHandler) create(c echo.Context) error {
	ctx := c.Request().Context()
	name := c.FormValue("name")
	cashBalance := parseCashBalance(c.FormValue("cash_balance"))
	account, err := h.c.Accounts.Create(ctx, name, cashBalance)
	if err != nil {
		return err
	}
	return templates.AccountRow(account).Render(ctx, c.Response().Writer)
}

func (h *AccountHandler) bulkCash(c echo.Context) error {
	ctx := c.Request().Context()
	accounts, err := h.c.Accounts.ListAll(ctx)
	if err != nil {
		return err
	}

	type pending struct {
		id          uuidx.UUID
		name        string
		cashBalance numeric.Decimal
	}
	updates := make([]pending, 0, len(accounts))

	for _, account := range accounts {
		field := "cash_" + account.ID.String()
		raw := c.FormValue(field)
		escapedName := html.EscapeString(account.Name)
		if raw == "" {
			c.Response().WriteHeader(http.StatusUnprocessableEntity)
			_, _ = c.Response().Write([]byte("'" + escapedName + "' 예수금을 입력하세요."))
			return nil
		}
		d, err2 := decimal.NewFromString(raw)
		if err2 != nil {
			c.Response().WriteHeader(http.StatusUnprocessableEntity)
			_, _ = c.Response().Write([]byte("'" + escapedName + "' 예수금 형식이 올바르지 않습니다."))
			return nil //nolint:nilerr // intentional: response written manually; nil tells Echo not to re-handle
		}
		updates = append(updates, pending{
			id:          account.ID,
			name:        account.Name,
			cashBalance: numeric.Wrap(d),
		})
	}

	for _, u := range updates {
		if _, err2 := h.c.Accounts.UpdateNameCash(ctx, u.id, u.name, u.cashBalance); err2 != nil {
			return err2
		}
	}

	c.Response().Header().Set("HX-Refresh", "true")
	return c.NoContent(http.StatusOK)
}

func (h *AccountHandler) row(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	account, err := h.c.Accounts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if account == nil {
		return c.NoContent(http.StatusNotFound)
	}
	return templates.AccountRow(*account).Render(ctx, c.Response().Writer)
}

func (h *AccountHandler) editForm(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	account, err := h.c.Accounts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if account == nil {
		return c.NoContent(http.StatusNotFound)
	}
	// available_kis_key_ids: empty until Phase 8
	return templates.AccountForm(*account, nil, "").Render(ctx, c.Response().Writer)
}

func (h *AccountHandler) update(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	account, err := h.c.Accounts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if account == nil {
		return c.NoContent(http.StatusNotFound)
	}

	name := c.FormValue("name")
	cashBalance := parseCashBalance(c.FormValue("cash_balance"))

	// kis_account_no: always sent by the edit form; empty string → NULL
	rawKis := c.FormValue("kis_account_no")
	kisAccountNo := sql.NullString{}
	if stripped := strings.TrimSpace(rawKis); stripped != "" {
		kisAccountNo = sql.NullString{String: stripped, Valid: true}
	}

	// kis_api_key_id: only set when kis_account_no is non-null; otherwise preserve existing
	kisAPIKeyID := sql.NullInt64{}
	if kisAccountNo.Valid {
		rawKeyID := c.FormValue("kis_api_key_id")
		if rawKeyID != "" {
			var keyID int64
			if _, err2 := fmt.Sscanf(rawKeyID, "%d", &keyID); err2 == nil {
				kisAPIKeyID = sql.NullInt64{Int64: keyID, Valid: true}
			}
		} else {
			// selector not submitted → preserve existing
			if account.KisAPIKeyID != nil {
				kisAPIKeyID = sql.NullInt64{Int64: *account.KisAPIKeyID, Valid: true}
			}
		}
	}

	// KIS validation deferred to Phase 8 — skip

	// account_type: edit form sends one of brokerage/irp/pension/isa; empty → NULL (unclassified)
	accountType := sql.NullString{}
	if t := strings.TrimSpace(c.FormValue("account_type")); models.ValidAccountType(t) {
		accountType = sql.NullString{String: t, Valid: true}
	}

	updated, err := h.c.Accounts.Update(ctx, id, name, cashBalance, kisAccountNo, kisAPIKeyID, accountType)
	if err != nil {
		return err
	}
	return templates.AccountRow(updated).Render(ctx, c.Response().Writer)
}

func (h *AccountHandler) delete(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	if err := h.c.Accounts.DeleteWithHoldings(ctx, id); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}

func (h *AccountHandler) syncAccount(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return renderSyncResult(c, id.String(), false, "잘못된 계좌 ID입니다.", nil, false)
	}

	if h.c.AccountSync == nil && len(h.c.AccountSyncByKeyID) == 0 {
		return renderSyncResult(c, id.String(), false,
			"KIS 계좌 동기화 서비스가 설정되지 않았습니다. (.env에 KIS_CANO/KIS_ACNT_PRDT_CD 확인)", nil, false)
	}

	confirmEmpty := strings.ToLower(c.FormValue("confirm_empty")) == "true"

	account, err := h.c.Accounts.GetByID(ctx, id)
	if err != nil || account == nil {
		return renderSyncResult(c, id.String(), false, "계좌를 찾을 수 없습니다.", nil, false)
	}

	syncSvc := h.c.SyncServiceForKeyID(account.KisAPIKeyID)
	if syncSvc == nil {
		return renderSyncResult(c, id.String(), false,
			"KIS 계좌 동기화 서비스가 설정되지 않았습니다. (.env에 KIS_CANO/KIS_ACNT_PRDT_CD 확인)", nil, false)
	}

	var cano, acntPrdtCd string
	if account.KisAccountNo != nil && *account.KisAccountNo != "" {
		cano, acntPrdtCd, err = normalizeKisAccountNo(*account.KisAccountNo)
		if err != nil {
			return renderSyncResult(c, id.String(), false, err.Error(), nil, false)
		}
	} else if h.c.KisCano != "" {
		cano = h.c.KisCano
		acntPrdtCd = h.c.KisAcntPrdtCd
	} else {
		return renderSyncResult(c, id.String(), false, "이 계좌에는 KIS 계좌번호가 설정되지 않았습니다.", nil, false)
	}

	syncResult, err := syncSvc.SyncAccount(ctx, *account, cano, acntPrdtCd, confirmEmpty)
	if err != nil {
		if services.IsKisEmptySnapshotError(err) {
			return renderSyncResult(c, id.String(), false,
				"동기화 중단: "+err.Error(), nil, !confirmEmpty)
		}
		msg := fmt.Sprintf("동기화 실패: %s", html.EscapeString(err.Error()))
		return renderSyncResult(c, id.String(), false, msg, nil, false)
	}
	return renderSyncResult(c, id.String(), true, "KIS 계좌 동기화 완료", &syncResult, false)
}

// classifyStocks backfills asset_class (ETF/stock) for unclassified stocks via KIS.
// Needed so IRP/연금 accounts can be recommended domestic-listed ETFs.
func (h *AccountHandler) classifyStocks(c echo.Context) error {
	ctx := c.Request().Context()
	if h.c.StockClassification == nil || !h.c.StockClassification.Enabled() {
		return templates.ClassifyResultPartial(false,
			"KIS 자산구분 분류 서비스가 설정되지 않았습니다. (.env에 KIS_APP_KEY/KIS_APP_SECRET 확인)").
			Render(ctx, c.Response().Writer)
	}
	res, err := h.c.StockClassification.ClassifyAll(ctx)
	if err != nil {
		return templates.ClassifyResultPartial(false,
			"자산구분 분류 실패: "+html.EscapeString(err.Error())).Render(ctx, c.Response().Writer)
	}
	status := "완료"
	if res.Failed > 0 {
		status = "부분 완료" // some stocks could not be classified — surface, don't claim done
	}
	msg := fmt.Sprintf("자산구분 분류 %s — 전체 %d · 신규분류 %d · 건너뜀 %d · 실패 %d",
		status, res.Total, res.Classified, res.Skipped, res.Failed)
	return templates.ClassifyResultPartial(res.Failed == 0, msg).Render(ctx, c.Response().Writer)
}

func renderSyncResult(c echo.Context, accountIDStr string, success bool, message string, result *models.KisAccountSyncResult, showConfirmEmpty bool) error {
	return templates.SyncResultPartial(accountIDStr, success, message, result, showConfirmEmpty).Render(c.Request().Context(), c.Response().Writer)
}

// normalizeKisAccountNo extracts the 8-digit account number and 2-digit product code
// from a KIS account number string (e.g. "12345678-01" or "1234567801").
func normalizeKisAccountNo(s string) (cano, acntPrdtCd string, err error) {
	var digits strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			digits.WriteRune(ch)
		}
	}
	d := digits.String()
	if len(d) != 10 {
		return "", "", fmt.Errorf("KIS 계좌번호 형식이 올바르지 않습니다 (8자리-2자리)")
	}
	return d[:8], d[8:], nil
}

func parseCashBalance(s string) numeric.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return numeric.Zero
	}
	return numeric.Wrap(d)
}
