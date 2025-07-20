package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/1cbyc/trade-algo-go/internal/models"
	"github.com/1cbyc/trade-algo-go/internal/strategies"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type TradingEngine struct {
	portfolio  *models.Portfolio
	strategies map[string]strategies.Strategy
	marketData map[string]*models.MarketData
	orderQueue chan *models.Order
	tradeQueue chan *models.Trade
	logger     *zap.Logger
	mu         sync.RWMutex
	running    bool
	stopChan   chan struct{}
}

func NewTradingEngine(initialCash decimal.Decimal, logger *zap.Logger) *TradingEngine {
	return &TradingEngine{
		portfolio: &models.Portfolio{
			ID:             generatePortfolioID(),
			Cash:           initialCash,
			Positions:      make(map[string]*models.Position),
			TotalValue:     initialCash,
			UnrealizedPnL:  decimal.Zero,
			RealizedPnL:    decimal.Zero,
			TotalRisk:      decimal.Zero,
			RiskMetrics:    models.PortfolioRiskMetrics{},
			TradeHistory:   []*models.Trade{},
			OrderHistory:   []*models.Order{},
			LastRebalanced: time.Now(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		strategies: make(map[string]strategies.Strategy),
		marketData: make(map[string]*models.MarketData),
		orderQueue: make(chan *models.Order, 1000),
		tradeQueue: make(chan *models.Trade, 1000),
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

func (e *TradingEngine) AddStrategy(strategy strategies.Strategy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strategies[strategy.ID()] = strategy
	e.logger.Info("Strategy added", zap.String("strategy_id", strategy.ID()), zap.String("name", strategy.Name()))
}

func (e *TradingEngine) RemoveStrategy(strategyID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.strategies, strategyID)
	e.logger.Info("Strategy removed", zap.String("strategy_id", strategyID))
}

func (e *TradingEngine) UpdateMarketData(symbol string, data *models.MarketData) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.marketData[symbol] = data
	e.logger.Debug("Market data updated", zap.String("symbol", symbol), zap.String("price", data.Price.String()))
}

func (e *TradingEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("trading engine already running")
	}
	e.running = true
	e.mu.Unlock()

	e.logger.Info("Starting trading engine")

	go e.orderProcessor(ctx)
	go e.tradeProcessor(ctx)
	go e.strategyExecutor(ctx)
	go e.riskManager(ctx)
	go e.portfolioUpdater(ctx)

	return nil
}

func (e *TradingEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return
	}

	e.running = false
	close(e.stopChan)
	e.logger.Info("Trading engine stopped")
}

func (e *TradingEngine) orderProcessor(ctx context.Context) {
	for {
		select {
		case order := <-e.orderQueue:
			e.processOrder(order)
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		}
	}
}

func (e *TradingEngine) tradeProcessor(ctx context.Context) {
	for {
		select {
		case trade := <-e.tradeQueue:
			e.processTrade(trade)
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		}
	}
}

func (e *TradingEngine) strategyExecutor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.executeStrategies(ctx)
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		}
	}
}

func (e *TradingEngine) riskManager(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.manageRisk()
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		}
	}
}

func (e *TradingEngine) portfolioUpdater(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.updatePortfolio()
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		}
	}
}

func (e *TradingEngine) executeStrategies(ctx context.Context) {
	e.mu.RLock()
	strategies := make([]strategies.Strategy, 0, len(e.strategies))
	for _, strategy := range e.strategies {
		strategies = append(strategies, strategy)
	}
	portfolio := e.portfolio
	marketData := e.marketData
	e.mu.RUnlock()

	for _, strategy := range strategies {
		if !strategy.IsEnabled() {
			continue
		}

		result, err := strategy.Execute(ctx, portfolio, marketData)
		if err != nil {
			e.logger.Error("Strategy execution failed", zap.String("strategy_id", strategy.ID()), zap.Error(err))
			continue
		}

		if result != nil {
			e.createOrderFromResult(result, strategy)
		}
	}
}

func (e *TradingEngine) createOrderFromResult(result *models.AlgorithmResult, strategy strategies.Strategy) {
	var side models.OrderSide
	if result.Action == "buy" {
		side = models.OrderSideBuy
	} else {
		side = models.OrderSideSell
	}

	order := &models.Order{
		ID:          generateOrderID(),
		Symbol:      result.Symbol,
		Side:        side,
		Type:        models.OrderTypeMarket,
		Quantity:    result.Quantity,
		Price:       result.Price,
		Status:      models.OrderStatusPending,
		Timestamp:   time.Now(),
		StrategyID:  result.StrategyID,
	}

	e.orderQueue <- order
}

func (e *TradingEngine) processOrder(order *models.Order) {
	e.mu.Lock()
	defer e.mu.Unlock()

	strategy, exists := e.strategies[order.StrategyID]
	if !exists {
		order.Status = models.OrderStatusRejected
		e.logger.Error("Strategy not found", zap.String("strategy_id", order.StrategyID))
		return
	}

	if err := strategy.ValidateOrder(order, e.portfolio); err != nil {
		order.Status = models.OrderStatusRejected
		e.logger.Error("Order validation failed", zap.String("order_id", order.ID), zap.Error(err))
		return
	}

	riskMetrics, err := strategy.CalculateRisk(order, e.portfolio)
	if err != nil {
		order.Status = models.OrderStatusRejected
		e.logger.Error("Risk calculation failed", zap.String("order_id", order.ID), zap.Error(err))
		return
	}

	order.RiskMetrics = *riskMetrics
	order.Status = models.OrderStatusFilled

	e.executeOrder(order)
	e.portfolio.OrderHistory = append(e.portfolio.OrderHistory, order)
}

func (e *TradingEngine) executeOrder(order *models.Order) {
	orderValue := order.Price.Mul(decimal.NewFromInt(order.Quantity))
	commission := orderValue.Mul(decimal.NewFromFloat(0.001))

	trade := &models.Trade{
		ID:          generateTradeID(),
		OrderID:     order.ID,
		Symbol:      order.Symbol,
		Side:        order.Side,
		Quantity:    order.Quantity,
		Price:       order.Price,
		Commission:  commission,
		Timestamp:   time.Now(),
		StrategyID:  order.StrategyID,
		RiskMetrics: order.RiskMetrics,
	}

	if order.Side == models.OrderSideBuy {
		e.portfolio.Cash = e.portfolio.Cash.Sub(orderValue).Sub(commission)
		e.updatePosition(order.Symbol, order.Quantity, order.Price)
	} else {
		e.portfolio.Cash = e.portfolio.Cash.Add(orderValue).Sub(commission)
		e.updatePosition(order.Symbol, -order.Quantity, order.Price)
	}

	e.tradeQueue <- trade
}

func (e *TradingEngine) processTrade(trade *models.Trade) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.portfolio.TradeHistory = append(e.portfolio.TradeHistory, trade)
	e.logger.Info("Trade executed",
		zap.String("trade_id", trade.ID),
		zap.String("symbol", trade.Symbol),
		zap.String("side", string(trade.Side)),
		zap.Int64("quantity", trade.Quantity),
		zap.String("price", trade.Price.String()),
	)
}

func (e *TradingEngine) updatePosition(symbol string, quantity int64, price decimal.Decimal) {
	position, exists := e.portfolio.Positions[symbol]
	if !exists {
		position = &models.Position{
			Symbol:        symbol,
			Quantity:      0,
			AveragePrice:  decimal.Zero,
			CurrentPrice:  price,
			UnrealizedPnL: decimal.Zero,
			RealizedPnL:   decimal.Zero,
			MarketValue:   decimal.Zero,
			RiskMetrics:   models.RiskMetrics{},
			LastUpdated:   time.Now(),
		}
		e.portfolio.Positions[symbol] = position
	}

	if quantity > 0 {
		totalCost := position.AveragePrice.Mul(decimal.NewFromInt(position.Quantity)).Add(price.Mul(decimal.NewFromInt(quantity)))
		totalQuantity := position.Quantity + quantity
		position.AveragePrice = totalCost.Div(decimal.NewFromInt(totalQuantity))
		position.Quantity = totalQuantity
	} else {
		position.Quantity += quantity
		if position.Quantity <= 0 {
			delete(e.portfolio.Positions, symbol)
		}
	}

	position.CurrentPrice = price
	position.LastUpdated = time.Now()
}

func (e *TradingEngine) updatePortfolio() {
	e.mu.Lock()
	defer e.mu.Unlock()

	totalValue := e.portfolio.Cash
	unrealizedPnL := decimal.Zero

	for symbol, position := range e.portfolio.Positions {
		marketData, exists := e.marketData[symbol]
		if exists {
			position.CurrentPrice = marketData.Price
			position.MarketValue = position.CurrentPrice.Mul(decimal.NewFromInt(position.Quantity))
			position.UnrealizedPnL = position.CurrentPrice.Sub(position.AveragePrice).Mul(decimal.NewFromInt(position.Quantity))
			totalValue = totalValue.Add(position.MarketValue)
			unrealizedPnL = unrealizedPnL.Add(position.UnrealizedPnL)
		}
	}

	e.portfolio.TotalValue = totalValue
	e.portfolio.UnrealizedPnL = unrealizedPnL
	e.portfolio.UpdatedAt = time.Now()
}

func (e *TradingEngine) manageRisk() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for symbol, position := range e.portfolio.Positions {
		if position.Quantity <= 0 {
			continue
		}

		drawdown := position.UnrealizedPnL.Div(position.MarketValue).Abs()
		if drawdown.GreaterThan(decimal.NewFromFloat(0.1)) {
			e.logger.Warn("Position drawdown exceeded", zap.String("symbol", symbol), zap.String("drawdown", drawdown.String()))
		}
	}
}

func (e *TradingEngine) GetPortfolio() *models.Portfolio {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.portfolio
}

func (e *TradingEngine) GetMarketData() map[string]*models.MarketData {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.marketData
}

func generatePortfolioID() string {
	return fmt.Sprintf("PORT-%d", time.Now().UnixNano())
}

func generateOrderID() string {
	return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}

func generateTradeID() string {
	return fmt.Sprintf("TRD-%d", time.Now().UnixNano())
}
