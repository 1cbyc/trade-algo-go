package strategies

import (
	"context"
	"math"
	"time"

	"github.com/1cbyc/trade-algo-go/internal/models"
	"github.com/shopspring/decimal"
)

type Strategy interface {
	ID() string
	Name() string
	Execute(ctx context.Context, portfolio *models.Portfolio, marketData map[string]*models.MarketData) (*models.AlgorithmResult, error)
	ValidateOrder(order *models.Order, portfolio *models.Portfolio) error
	CalculateRisk(order *models.Order, portfolio *models.Portfolio) (*models.RiskMetrics, error)
	UpdateConfig(config *models.StrategyConfig) error
	GetConfig() *models.StrategyConfig
	IsEnabled() bool
}

type BaseStrategy struct {
	config *models.StrategyConfig
}

func NewBaseStrategy(config *models.StrategyConfig) *BaseStrategy {
	return &BaseStrategy{
		config: config,
	}
}

func (s *BaseStrategy) ID() string {
	return s.config.ID
}

func (s *BaseStrategy) Name() string {
	return s.config.Name
}

func (s *BaseStrategy) GetConfig() *models.StrategyConfig {
	return s.config
}

func (s *BaseStrategy) UpdateConfig(config *models.StrategyConfig) error {
	s.config = config
	s.config.UpdatedAt = time.Now()
	return nil
}

func (s *BaseStrategy) IsEnabled() bool {
	return s.config.Enabled
}

func (s *BaseStrategy) ValidateOrder(order *models.Order, portfolio *models.Portfolio) error {
	if order.Quantity <= 0 {
		return ErrInvalidQuantity
	}

	orderValue := order.Price.Mul(decimal.NewFromInt(order.Quantity))

	if orderValue.LessThan(s.config.MinOrderSize) {
		return ErrOrderTooSmall
	}

	if orderValue.GreaterThan(s.config.MaxOrderSize) {
		return ErrOrderTooLarge
	}

	if order.Side == models.OrderSideBuy {
		if portfolio.Cash.LessThan(orderValue) {
			return ErrInsufficientFunds
		}
	} else {
		position, exists := portfolio.Positions[order.Symbol]
		if !exists || position.Quantity < order.Quantity {
			return ErrInsufficientPosition
		}
	}

	return nil
}

func (s *BaseStrategy) CalculateRisk(order *models.Order, portfolio *models.Portfolio) (*models.RiskMetrics, error) {
	orderValue := order.Price.Mul(decimal.NewFromInt(order.Quantity))
	portfolioValue := portfolio.TotalValue

	if portfolioValue.IsZero() {
		return &models.RiskMetrics{}, nil
	}

	positionRisk := orderValue.Div(portfolioValue)

	if positionRisk.GreaterThan(s.config.MaxPositionSize) {
		return nil, ErrPositionTooLarge
	}

	totalRisk := portfolio.TotalRisk.Add(positionRisk)
	if totalRisk.GreaterThan(s.config.MaxPortfolioRisk) {
		return nil, ErrPortfolioRiskExceeded
	}

	volatility := s.calculateVolatility(order.Symbol, portfolio)
	beta := s.calculateBeta(order.Symbol, portfolio)
	var95 := s.calculateVaR(orderValue, volatility)
	expectedShortfall := s.calculateExpectedShortfall(var95, volatility)
	sharpeRatio := s.calculateSharpeRatio(orderValue, volatility)
	maxDrawdown := s.calculateMaxDrawdown(portfolio)

	return &models.RiskMetrics{
		VaR95:             var95,
		ExpectedShortfall: expectedShortfall,
		SharpeRatio:       sharpeRatio,
		MaxDrawdown:       maxDrawdown,
		Volatility:        volatility,
		Beta:              beta,
	}, nil
}

func (s *BaseStrategy) calculateVolatility(symbol string, portfolio *models.Portfolio) decimal.Decimal {
	if len(portfolio.TradeHistory) < 2 {
		return decimal.Zero
	}

	var returns []decimal.Decimal
	for i := 1; i < len(portfolio.TradeHistory); i++ {
		if portfolio.TradeHistory[i].Symbol == symbol {
			prevPrice := portfolio.TradeHistory[i-1].Price
			currPrice := portfolio.TradeHistory[i].Price
			if !prevPrice.IsZero() {
				returns = append(returns, currPrice.Sub(prevPrice).Div(prevPrice))
			}
		}
	}

	if len(returns) == 0 {
		return decimal.Zero
	}

	mean := decimal.Zero
	for _, ret := range returns {
		mean = mean.Add(ret)
	}
	mean = mean.Div(decimal.NewFromInt(int64(len(returns))))

	variance := decimal.Zero
	for _, ret := range returns {
		diff := ret.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.Div(decimal.NewFromInt(int64(len(returns))))

	volatility := decimal.NewFromFloat(0.0)
	if variance.GreaterThan(decimal.Zero) {
		volatility = decimal.NewFromFloat(math.Sqrt(variance.InexactFloat64()))
	}
	return volatility
}

func (s *BaseStrategy) calculateBeta(symbol string, portfolio *models.Portfolio) decimal.Decimal {
	return decimal.NewFromFloat(1.0)
}

func (s *BaseStrategy) calculateVaR(orderValue, volatility decimal.Decimal) decimal.Decimal {
	zScore := decimal.NewFromFloat(1.645)
	return orderValue.Mul(volatility).Mul(zScore)
}

func (s *BaseStrategy) calculateExpectedShortfall(var95, volatility decimal.Decimal) decimal.Decimal {
	return var95.Mul(decimal.NewFromFloat(1.25))
}

func (s *BaseStrategy) calculateSharpeRatio(orderValue, volatility decimal.Decimal) decimal.Decimal {
	if volatility.IsZero() {
		return decimal.Zero
	}
	excessReturn := orderValue.Sub(s.config.RiskFreeRate)
	return excessReturn.Div(volatility)
}

func (s *BaseStrategy) calculateMaxDrawdown(portfolio *models.Portfolio) decimal.Decimal {
	if len(portfolio.TradeHistory) == 0 {
		return decimal.Zero
	}

	peak := portfolio.TradeHistory[0].Price
	maxDrawdown := decimal.Zero

	for _, trade := range portfolio.TradeHistory {
		if trade.Price.GreaterThan(peak) {
			peak = trade.Price
		}
		drawdown := peak.Sub(trade.Price).Div(peak)
		if drawdown.GreaterThan(maxDrawdown) {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}
