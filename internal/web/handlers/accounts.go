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
	"github.com/kadragon/portfolio-manager/internal/numeric"
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
			return nil
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

	updated, err := h.c.Accounts.Update(ctx, id, name, cashBalance, kisAccountNo, kisAPIKeyID)
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

func parseCashBalance(s string) numeric.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return numeric.Zero
	}
	return numeric.Wrap(d)
}
