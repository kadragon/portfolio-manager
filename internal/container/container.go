// Package container wires the repositories (and, in later phases, services and
// external clients) over an open database, the Go counterpart of
// core/container.py's ServiceContainer. As the composition root it may depend on
// both the db and repository layers.
package container

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/kis"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// Container holds shared dependencies for the web layer.
type Container struct {
	DB          *sql.DB
	Groups      *repositories.GroupRepository
	Stocks      *repositories.StockRepository
	Accounts    *repositories.AccountRepository
	Holdings    *repositories.HoldingRepository
	Deposits    *repositories.DepositRepository
	StockPrices *repositories.StockPriceRepository
	Portfolio   *services.PortfolioService
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

	var priceClient services.PriceClient
	var exchangeRate *services.ExchangeRateService

	if setupKIS {
		priceClient = buildKISClient(stockPrices)
		exchangeRate = buildExchangeRate()
	}

	priceService := services.NewPriceService(stockPrices, priceClient)
	portfolio := services.NewPortfolioService(groups, stocks, holdings, accounts, deposits, priceService, exchangeRate)

	return &Container{
		DB:          sqlDB,
		Groups:      groups,
		Stocks:      stocks,
		Accounts:    accounts,
		Holdings:    holdings,
		Deposits:    deposits,
		StockPrices: stockPrices,
		Portfolio:   portfolio,
	}
}

// Close releases the database connection.
func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// buildKISClient reads KIS env vars and returns a UnifiedPriceClient, or nil if keys are absent.
func buildKISClient(stockPrices *repositories.StockPriceRepository) services.PriceClient {
	_ = stockPrices // reserved for future write-through wiring in the client itself

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

// buildExchangeRate reads USD_KRW_RATE env var. Returns nil if not set.
func buildExchangeRate() *services.ExchangeRateService {
	raw := strings.TrimSpace(os.Getenv("USD_KRW_RATE"))
	if raw == "" {
		return nil
	}
	rate, err := numeric.FromString(raw)
	if err != nil {
		log.Printf("invalid USD_KRW_RATE: %v", err)
		return nil
	}
	return services.NewFixedExchangeRateService(rate)
}
