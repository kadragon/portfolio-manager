// Package container wires the repositories (and, in later phases, services and
// external clients) over an open database, the Go counterpart of
// core/container.py's ServiceContainer. As the composition root it may depend on
// both the db and repository layers.
package container

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/kis"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// Container holds shared dependencies for the web layer.
type Container struct {
	DB                 *sql.DB
	Groups             *repositories.GroupRepository
	Stocks             *repositories.StockRepository
	Accounts           *repositories.AccountRepository
	Holdings           *repositories.HoldingRepository
	Deposits           *repositories.DepositRepository
	StockPrices        *repositories.StockPriceRepository
	OrderExecutions    *repositories.OrderExecutionRepository
	Portfolio          *services.PortfolioService
	Rebalance          *services.RebalanceService
	RebalanceExecution *services.RebalanceExecutionService
	OrderClient        services.OrderClient // nil if KIS not configured
}

// New opens the database at path (empty = default location) and builds the
// repositories. The caller is responsible for Close.
func New(path string) (*Container, error) {
	sqlDB, q, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	return newWithQueries(sqlDB, q, true), nil
}

// NewWithQueries builds a Container over an already-open database and queries
// handle (used by tests with an in-memory database; skips KIS wiring).
func NewWithQueries(sqlDB *sql.DB, q *sqlc.Queries) *Container {
	return newWithQueries(sqlDB, q, false)
}

func newWithQueries(sqlDB *sql.DB, q *sqlc.Queries, setupKIS bool) *Container {
	groups := repositories.NewGroupRepository(q)
	stocks := repositories.NewStockRepository(q)
	accounts := repositories.NewAccountRepository(q)
	holdings := repositories.NewHoldingRepository(q)
	deposits := repositories.NewDepositRepository(q)
	stockPrices := repositories.NewStockPriceRepository(q)
	orderExecutions := repositories.NewOrderExecutionRepository(q)

	var priceClient services.PriceClient
	var exchangeRate *services.ExchangeRateService
	var orderClient services.OrderClient

	if setupKIS {
		priceClient = buildKISClient()
		exchangeRate = buildExchangeRate()
		orderClient = buildOrderClient()
	}

	priceService := services.NewPriceService(stockPrices, priceClient)
	_ = services.NewStockService(stocks, priceService)
	portfolio := services.NewPortfolioService(groups, stocks, holdings, accounts, deposits, priceService, exchangeRate)
	rebalance := services.NewRebalanceService()

	execRepo := &execRepoAdapter{r: orderExecutions}
	rebalanceExecution := services.NewRebalanceExecutionService(orderClient, execRepo, nil)

	return &Container{
		DB:                 sqlDB,
		Groups:             groups,
		Stocks:             stocks,
		Accounts:           accounts,
		Holdings:           holdings,
		Deposits:           deposits,
		StockPrices:        stockPrices,
		OrderExecutions:    orderExecutions,
		Portfolio:          portfolio,
		Rebalance:          rebalance,
		RebalanceExecution: rebalanceExecution,
		OrderClient:        orderClient,
	}
}

// Close releases the database connection.
func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// execRepoAdapter wraps OrderExecutionRepository to satisfy services.ExecutionRepo.
type execRepoAdapter struct {
	r *repositories.OrderExecutionRepository
}

func (a *execRepoAdapter) Create(
	ctx context.Context,
	ticker, side string,
	quantity int,
	currency, status, message, exchange string,
	rawResponse map[string]any,
) (models.OrderExecutionRecord, error) {
	return a.r.Create(ctx, ticker, side, quantity, currency, status, message, exchange, rawResponse)
}

// buildKISClient reads KIS env vars and returns a UnifiedPriceClient, or nil if keys are absent.
func buildKISClient() services.PriceClient {
	appKey := strings.TrimSpace(os.Getenv("KIS_APP_KEY"))
	appSecret := strings.TrimSpace(os.Getenv("KIS_APP_SECRET"))
	if appKey == "" || appSecret == "" {
		return nil
	}

	env := strings.ToLower(strings.TrimSpace(os.Getenv("KIS_ENV")))
	if env == "" {
		env = "real"
	}
	if i := strings.IndexByte(env, '/'); i >= 0 {
		env = env[:i]
	}

	custType := strings.TrimSpace(os.Getenv("KIS_CUST_TYPE"))
	if custType == "" {
		custType = "P"
	}

	baseURL := "https://openapi.koreainvestment.com:9443"
	if env == "demo" || env == "vps" || env == "paper" {
		baseURL = "https://openapivts.koreainvestment.com:29443"
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	authClient := &kis.AuthClient{
		HTTPClient: httpClient,
		BaseURL:    baseURL,
		AppKey:     appKey,
		AppSecret:  appSecret,
	}
	store := kis.NewFileTokenStore(".data/kis_token_1.json")
	manager := kis.NewTokenManager(store, authClient, time.Minute)

	prdtTypeCd := strings.TrimSpace(os.Getenv("KIS_PRDT_TYPE_CD"))
	if prdtTypeCd == "" {
		prdtTypeCd = "300"
	}
	domesticInfoTrID := strings.TrimSpace(os.Getenv("KIS_DOMESTIC_INFO_TR_ID"))
	if domesticInfoTrID == "" {
		domesticInfoTrID = "CTPF1002R"
	}
	overseasInfoTrID := strings.TrimSpace(os.Getenv("KIS_OVERSEAS_INFO_TR_ID"))
	if overseasInfoTrID == "" {
		overseasInfoTrID = "CTPF1702R"
	}

	unified := &kis.UnifiedPriceClient{
		Domestic: &kis.DomesticPriceClient{
			HTTP:      httpClient,
			BaseURL:   baseURL,
			AppKey:    appKey,
			AppSecret: appSecret,
			CustType:  custType,
			Env:       env,
			Manager:   manager,
		},
		Overseas: &kis.OverseasPriceClient{
			HTTP:      httpClient,
			BaseURL:   baseURL,
			AppKey:    appKey,
			AppSecret: appSecret,
			CustType:  custType,
			Env:       env,
			Manager:   manager,
		},
		DomesticInfo: &kis.DomesticInfoClient{
			HTTP:      httpClient,
			BaseURL:   baseURL,
			AppKey:    appKey,
			AppSecret: appSecret,
			TrID:      domesticInfoTrID,
			CustType:  custType,
			Manager:   manager,
		},
		OverseasInfo: &kis.OverseasInfoClient{
			HTTP:      httpClient,
			BaseURL:   baseURL,
			AppKey:    appKey,
			AppSecret: appSecret,
			TrID:      overseasInfoTrID,
			CustType:  custType,
			Manager:   manager,
		},
		PrdtTypeCd: prdtTypeCd,
	}

	log.Printf("KIS price client initialized (env=%q)", env) //nolint:gosec // env is operator-controlled, not user input
	return unified
}

// buildOrderClient reads KIS env vars and returns a UnifiedOrderClient, or nil if keys are absent.
func buildOrderClient() services.OrderClient {
	appKey := strings.TrimSpace(os.Getenv("KIS_APP_KEY"))
	appSecret := strings.TrimSpace(os.Getenv("KIS_APP_SECRET"))
	cano := strings.TrimSpace(os.Getenv("KIS_CANO"))
	acntPrdtCd := strings.TrimSpace(os.Getenv("KIS_ACNT_PRDT_CD"))
	if appKey == "" || appSecret == "" || cano == "" {
		return nil
	}
	if acntPrdtCd == "" {
		acntPrdtCd = "01"
	}

	env := strings.ToLower(strings.TrimSpace(os.Getenv("KIS_ENV")))
	if env == "" {
		env = "real"
	}
	if i := strings.IndexByte(env, '/'); i >= 0 {
		env = env[:i]
	}

	custType := strings.TrimSpace(os.Getenv("KIS_CUST_TYPE"))
	if custType == "" {
		custType = "P"
	}

	baseURL := "https://openapi.koreainvestment.com:9443"
	if env == "demo" || env == "vps" || env == "paper" {
		baseURL = "https://openapivts.koreainvestment.com:29443"
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	authClient := &kis.AuthClient{
		HTTPClient: httpClient,
		BaseURL:    baseURL,
		AppKey:     appKey,
		AppSecret:  appSecret,
	}
	store := kis.NewFileTokenStore(".data/kis_token_1.json")
	manager := kis.NewTokenManager(store, authClient, time.Minute)

	log.Printf("KIS order client initialized (env=%q)", env) //nolint:gosec // env is operator-controlled, not user input
	return &kis.UnifiedOrderClient{
		Domestic: &kis.DomesticOrderClient{
			HTTP:       httpClient,
			BaseURL:    baseURL,
			AppKey:     appKey,
			AppSecret:  appSecret,
			CANO:       cano,
			AcntPrdtCd: acntPrdtCd,
			CustType:   custType,
			Env:        env,
			Manager:    manager,
		},
		Overseas: &kis.OverseasOrderClient{
			HTTP:      httpClient,
			BaseURL:   baseURL,
			AppKey:    appKey,
			AppSecret: appSecret,
			CustType:  custType,
			Env:       env,
			Manager:   manager,
		},
	}
}

// buildExchangeRate reads USD_KRW_RATE or EXIM_AUTH_KEY env vars.
// Priority: fixed rate > EXIM client > nil.
func buildExchangeRate() *services.ExchangeRateService {
	if raw := strings.TrimSpace(os.Getenv("USD_KRW_RATE")); raw != "" {
		rate, err := numeric.FromString(raw)
		if err != nil {
			log.Printf("invalid USD_KRW_RATE: %v", err)
			return nil
		}
		return services.NewFixedExchangeRateService(rate)
	}

	if authKey := strings.TrimSpace(os.Getenv("EXIM_AUTH_KEY")); authKey != "" {
		eximClient := &services.EximClient{
			HTTPClient: &http.Client{Timeout: 10 * time.Second},
			BaseURL:    "https://oapi.koreaexim.go.kr",
			AuthKey:    authKey,
		}
		return services.NewEximExchangeRateService(eximClient)
	}

	return nil
}
