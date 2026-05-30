// Command portfolio-web is the HTTP entrypoint, the Go counterpart of
// portfolio_manager.web.app:run.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/web/handlers"
)

// dashboardPlaceholder stands in for "/" until the dashboard slice (Phase 6).
const dashboardPlaceholder = `<!doctype html><html lang="ko"><head><meta charset="UTF-8"><title>포트폴리오 매니저</title></head><body><p>대시보드는 이후 단계에서 구현됩니다. <a href="/groups">그룹</a></p></body></html>`

func main() {
	c, err := container.New("")
	if err != nil {
		log.Fatalf("init container: %v", err)
	}
	defer func() { _ = c.Close() }()

	e := newServer(c)

	addr := "127.0.0.1:8000"
	if v := os.Getenv("PORTFOLIO_ADDR"); v != "" {
		addr = v
	}

	go func() {
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// newServer builds the Echo instance with all routes registered.
func newServer(c *container.Container) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Static("/static", staticDir())

	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, dashboardPlaceholder)
	})

	handlers.NewGroupHandler(c).Register(e)
	handlers.NewStockHandler(c).Register(e)
	handlers.NewAccountHandler(c).Register(e)
	handlers.NewHoldingHandler(c).Register(e)
	handlers.NewDepositHandler(c).Register(e)

	return e
}

// staticDir returns the directory served at /static. During the Python→Go
// transition the existing assets are reused in place; cutover moves them under
// internal/web/static.
func staticDir() string {
	const transitional = "src/portfolio_manager/web/static"
	if _, err := os.Stat(transitional); err == nil {
		return transitional
	}
	return "internal/web/static"
}
