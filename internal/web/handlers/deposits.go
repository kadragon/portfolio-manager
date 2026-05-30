package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/kadragon/portfolio-manager/internal/web/templates"
)

// DepositHandler handles /deposits routes.
type DepositHandler struct {
	c *container.Container
}

// NewDepositHandler creates a handler using the given container.
func NewDepositHandler(c *container.Container) *DepositHandler {
	return &DepositHandler{c: c}
}

// Register wires all deposit routes onto e.
func (h *DepositHandler) Register(e *echo.Echo) {
	e.GET("/deposits", h.list)
	e.POST("/deposits", h.create)
	e.GET("/deposits/:id", h.row)
	e.GET("/deposits/:id/edit", h.editForm)
	e.PUT("/deposits/:id", h.update)
	e.DELETE("/deposits/:id", h.deleteDeposit)
}

func (h *DepositHandler) list(c echo.Context) error {
	ctx := c.Request().Context()
	deposits, err := h.c.Deposits.ListAll(ctx)
	if err != nil {
		return err
	}
	total := sumDeposits(deposits)
	return templates.DepositsPage(deposits, total).Render(ctx, c.Response().Writer)
}

func (h *DepositHandler) row(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	deposit, err := h.c.Deposits.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if deposit == nil {
		return c.NoContent(http.StatusNotFound)
	}
	return templates.DepositRow(*deposit).Render(ctx, c.Response().Writer)
}

func (h *DepositHandler) create(c echo.Context) error {
	ctx := c.Request().Context()
	amount, err := parseDepositAmount(c.FormValue("amount"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	depositDate, err := datex.ParseDate(c.FormValue("deposit_date"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	noteStr := strings.TrimSpace(c.FormValue("note"))
	note := sql.NullString{}
	if noteStr != "" {
		note = sql.NullString{String: noteStr, Valid: true}
	}

	existing, err := h.c.Deposits.GetByDate(ctx, depositDate)
	if err != nil {
		return err
	}
	if existing != nil {
		// Upsert: date already exists — update instead of create.
		notePtr := &note
		updated, err := h.c.Deposits.Update(ctx, existing.ID, amount, depositDate, notePtr)
		if err != nil {
			return err
		}
		c.Response().Header().Set("HX-Refresh", "true")
		return templates.DepositRow(updated).Render(ctx, c.Response().Writer)
	}

	created, err := h.c.Deposits.Create(ctx, amount, depositDate, note)
	if err != nil {
		return err
	}
	return templates.DepositRow(created).Render(ctx, c.Response().Writer)
}

func (h *DepositHandler) editForm(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	deposit, err := h.c.Deposits.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if deposit == nil {
		return c.NoContent(http.StatusNotFound)
	}
	return templates.DepositForm(*deposit).Render(ctx, c.Response().Writer)
}

func (h *DepositHandler) update(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	amount, err := parseDepositAmount(c.FormValue("amount"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	depositDate, err := datex.ParseDate(c.FormValue("deposit_date"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}

	// Note sentinel: empty=keep unchanged, "/clear"=null, else update.
	trimmedNote := strings.TrimSpace(c.FormValue("note"))
	var notePtr *sql.NullString
	if trimmedNote != "" {
		if strings.ToLower(trimmedNote) == "/clear" {
			ns := sql.NullString{}
			notePtr = &ns
		} else {
			ns := sql.NullString{String: trimmedNote, Valid: true}
			notePtr = &ns
		}
	}

	updated, err := h.c.Deposits.Update(ctx, id, amount, depositDate, notePtr)
	if err != nil {
		return err
	}
	return templates.DepositRow(updated).Render(ctx, c.Response().Writer)
}

func (h *DepositHandler) deleteDeposit(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	if err := h.c.Deposits.Delete(ctx, id); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}

func parseDepositAmount(s string) (numeric.Decimal, error) {
	d, err := numeric.FromString(s)
	if err != nil {
		return numeric.Zero, err
	}
	return d, nil
}

func sumDeposits(deposits []models.Deposit) numeric.Decimal {
	total := numeric.Zero
	for _, d := range deposits {
		total = numeric.Wrap(total.Add(d.Amount.Decimal))
	}
	return total
}
