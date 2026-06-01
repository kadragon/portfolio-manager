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

func main() {
	c, err := container.New("")
	if err != nil {
		log.Fatalf("init container: %v", err)
	}
	defer func() { _ = c.Close() }()

	e := newServer(c)

	addr := defaultAddr()

	syncCtx, syncCancel := context.WithCancel(context.Background())
	defer syncCancel()
	if c.PriceSync != nil {
		go c.PriceSync.Start(syncCtx)
	}

	go func() {
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	syncCancel()
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

	handlers.NewDashboardHandler(c).Register(e)
	handlers.NewGroupHandler(c).Register(e)
	handlers.NewStockHandler(c).Register(e)
	handlers.NewAccountHandler(c).Register(e)
	handlers.NewHoldingHandler(c).Register(e)
	handlers.NewDepositHandler(c).Register(e)
	handlers.NewRebalanceHandler(c).Register(e)

	return e
}

func staticDir() string {
	return "internal/web/static"
}

func defaultAddr() string {
	if v := os.Getenv("PORTFOLIO_ADDR"); v != "" {
		return v
	}
	return "0.0.0.0:8000"
}
