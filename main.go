package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/1cbyc/trade-algo-go/internal/engine"
	"github.com/1cbyc/trade-algo-go/internal/models"
	"github.com/1cbyc/trade-algo-go/internal/simulator"
	"github.com/1cbyc/trade-algo-go/internal/strategies"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func main() {
	var (
		initialCash = flag.Float64("cash", 100000.0, "Initial portfolio cash")
		duration    = flag.Duration("duration", 5*time.Minute, "Simulation duration")
		logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	logger := setupLogger(*logLevel)
	defer logger.Sync()

	logger.Info("Starting Trade Algorithm Go", zap.Float64("initial_cash", *initialCash))

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	tradingEngine := engine.NewTradingEngine(decimal.NewFromFloat(*initialCash), logger)
	marketSimulator := simulator.NewMarketSimulator(logger)

	setupSymbols(marketSimulator, logger)
	setupStrategies(tradingEngine, logger)

	if err := tradingEngine.Start(ctx); err != nil {
		logger.Fatal("Failed to start trading engine", zap.Error(err))
	}

	marketSimulator.Start()

	go handleMarketUpdates(tradingEngine, marketSimulator, logger)
	go printPortfolioStatus(tradingEngine, logger)

	handleShutdown(ctx, tradingEngine, marketSimulator, logger)
}

func setupLogger(level string) *zap.Logger {
	var config zap.Config
	switch level {
	case "debug":
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config = zap.NewProductionConfig()
	}

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}

	return logger
}

func setupSymbols(simulator *simulator.MarketSimulator, logger *zap.Logger) {
	symbols := map[string]struct {
		basePrice  float64
		volatility float64
	}{
		"AAPL":  {150.0, 0.02},
		"GOOGL": {2800.0, 0.025},
		"MSFT":  {300.0, 0.018},
		"TSLA":  {800.0, 0.04},
		"AMZN":  {3200.0, 0.022},
		"NFLX":  {500.0, 0.035},
		"NVDA":  {600.0, 0.03},
		"META":  {350.0, 0.028},
	}

	for symbol, data := range symbols {
		simulator.AddSymbol(symbol, decimal.NewFromFloat(data.basePrice), decimal.NewFromFloat(data.volatility))
		logger.Info("Symbol configured", zap.String("symbol", symbol), zap.Float64("base_price", data.basePrice))
	}
}

func setupStrategies(engine *engine.TradingEngine, logger *zap.Logger) {
	movingAvgConfig := &models.StrategyConfig{
		ID:                    "ma_crossover_001",
		Name:                  "Moving Average Crossover",
		MaxPositionSize:       decimal.NewFromFloat(0.2),
		MaxPortfolioRisk:      decimal.NewFromFloat(0.15),
		MaxDrawdown:           decimal.NewFromFloat(0.1),
		StopLossPercent:       decimal.NewFromFloat(0.05),
		TakeProfitPercent:     decimal.NewFromFloat(0.1),
		TrailingStopPercent:   decimal.NewFromFloat(0.03),
		RebalanceThreshold:    decimal.NewFromFloat(0.05),
		MaxOrdersPerDay:       50,
		MinOrderSize:          decimal.NewFromFloat(1000.0),
		MaxOrderSize:          decimal.NewFromFloat(10000.0),
		CommissionRate:        decimal.NewFromFloat(0.001),
		SlippageTolerance:     decimal.NewFromFloat(0.002),
		RiskFreeRate:          decimal.NewFromFloat(0.02),
		MarketDataWindow:      30,
		TechnicalIndicators:   []string{"SMA", "EMA", "RSI"},
		Enabled:               true,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	movingAvgStrategy := strategies.NewMovingAverageStrategy(movingAvgConfig)
	engine.AddStrategy(movingAvgStrategy)

	logger.Info("Strategy configured", zap.String("strategy_id", movingAvgStrategy.ID()), zap.String("name", movingAvgStrategy.Name()))
}

func handleMarketUpdates(engine *engine.TradingEngine, simulator *simulator.MarketSimulator, logger *zap.Logger) {
	updateChan := simulator.GetUpdateChannel()
	for marketData := range updateChan {
		engine.UpdateMarketData(marketData.Symbol, marketData)
	}
}

func printPortfolioStatus(engine *engine.TradingEngine, logger *zap.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		portfolio := engine.GetPortfolio()
		
		logger.Info("Portfolio Status",
			zap.String("portfolio_id", portfolio.ID),
			zap.String("total_value", portfolio.TotalValue.String()),
			zap.String("cash", portfolio.Cash.String()),
			zap.String("unrealized_pnl", portfolio.UnrealizedPnL.String()),
			zap.String("realized_pnl", portfolio.RealizedPnL.String()),
			zap.String("total_risk", portfolio.TotalRisk.String()),
			zap.Int("positions_count", len(portfolio.Positions)),
			zap.Int("trades_count", len(portfolio.TradeHistory)),
		)

		for symbol, position := range portfolio.Positions {
			logger.Info("Position",
				zap.String("symbol", symbol),
				zap.Int64("quantity", position.Quantity),
				zap.String("average_price", position.AveragePrice.String()),
				zap.String("current_price", position.CurrentPrice.String()),
				zap.String("market_value", position.MarketValue.String()),
				zap.String("unrealized_pnl", position.UnrealizedPnL.String()),
			)
		}
	}
}

func handleShutdown(ctx context.Context, engine *engine.TradingEngine, simulator *simulator.MarketSimulator, logger *zap.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		logger.Info("Simulation completed")
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	}

	logger.Info("Shutting down trading system")

	simulator.Stop()
	engine.Stop()

	finalPortfolio := engine.GetPortfolio()
	logger.Info("Final Portfolio Summary",
		zap.String("portfolio_id", finalPortfolio.ID),
		zap.String("initial_cash", decimal.NewFromFloat(100000.0).String()),
		zap.String("final_value", finalPortfolio.TotalValue.String()),
		zap.String("total_return", finalPortfolio.TotalValue.Sub(decimal.NewFromFloat(100000.0)).String()),
		zap.String("return_percentage", finalPortfolio.TotalValue.Sub(decimal.NewFromFloat(100000.0)).Div(decimal.NewFromFloat(100000.0)).Mul(decimal.NewFromFloat(100)).String()),
		zap.Int("total_trades", len(finalPortfolio.TradeHistory)),
		zap.Int("final_positions", len(finalPortfolio.Positions)),
	)

	logger.Info("Trading system shutdown complete")
}
