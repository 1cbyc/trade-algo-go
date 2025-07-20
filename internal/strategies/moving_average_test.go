package strategies

import (
	"context"
	"testing"
	"time"

	"github.com/1cbyc/trade-algo-go/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMovingAverageStrategy(t *testing.T) {
	config := &models.StrategyConfig{
		ID:   "test_ma",
		Name: "Test Moving Average",
	}

	strategy := NewMovingAverageStrategy(config)

	assert.NotNil(t, strategy)
	assert.Equal(t, "test_ma", strategy.ID())
	assert.Equal(t, "Test Moving Average", strategy.Name())
	assert.Equal(t, 10, strategy.shortPeriod)
	assert.Equal(t, 30, strategy.longPeriod)
	assert.Equal(t, 9, strategy.signalPeriod)
}

func TestMovingAverageStrategy_Execute_Disabled(t *testing.T) {
	config := &models.StrategyConfig{
		ID:      "test_ma",
		Name:    "Test Moving Average",
		Enabled: false,
	}

	strategy := NewMovingAverageStrategy(config)
	portfolio := createTestPortfolio()
	marketData := createTestMarketData()

	result, err := strategy.Execute(context.Background(), portfolio, marketData)

	assert.Nil(t, result)
	assert.Equal(t, ErrStrategyDisabled, err)
}

func TestMovingAverageStrategy_Execute_NoMarketData(t *testing.T) {
	config := &models.StrategyConfig{
		ID:      "test_ma",
		Name:    "Test Moving Average",
		Enabled: true,
	}

	strategy := NewMovingAverageStrategy(config)
	portfolio := createTestPortfolio()
	marketData := make(map[string]*models.MarketData)

	result, err := strategy.Execute(context.Background(), portfolio, marketData)

	assert.Nil(t, result)
	assert.NoError(t, err)
}

func TestMovingAverageStrategy_CalculateSMA(t *testing.T) {
	config := &models.StrategyConfig{
		ID:      "test_ma",
		Name:    "Test Moving Average",
		Enabled: true,
	}

	strategy := NewMovingAverageStrategy(config)
	portfolio := createTestPortfolioWithHistory()

	sma := strategy.calculateSMA("AAPL", 3, portfolio)

	assert.False(t, sma.IsZero())
	assert.True(t, sma.GreaterThan(decimal.Zero))
}

func TestMovingAverageStrategy_CalculateSMA_InsufficientData(t *testing.T) {
	config := &models.StrategyConfig{
		ID:      "test_ma",
		Name:    "Test Moving Average",
		Enabled: true,
	}

	strategy := NewMovingAverageStrategy(config)
	portfolio := createTestPortfolio()

	sma := strategy.calculateSMA("AAPL", 10, portfolio)

	assert.True(t, sma.IsZero())
}

func TestMovingAverageStrategy_CalculateOptimalQuantity(t *testing.T) {
	config := &models.StrategyConfig{
		ID:           "test_ma",
		Name:         "Test Moving Average",
		Enabled:      true,
		MaxOrderSize: decimal.NewFromFloat(10000.0),
	}

	strategy := NewMovingAverageStrategy(config)
	portfolio := createTestPortfolio()
	price := decimal.NewFromFloat(150.0)

	quantity := strategy.calculateOptimalQuantity(price, portfolio)

	assert.Greater(t, quantity, int64(0))

	maxQuantity := decimal.NewFromFloat(10000.0).Div(price).IntPart()
	assert.LessOrEqual(t, quantity, maxQuantity)
}

func TestMovingAverageStrategy_CalculateConfidence(t *testing.T) {
	config := &models.StrategyConfig{
		ID:      "test_ma",
		Name:    "Test Moving Average",
		Enabled: true,
	}

	strategy := NewMovingAverageStrategy(config)

	shortMA := decimal.NewFromFloat(155.0)
	longMA := decimal.NewFromFloat(150.0)
	currentPrice := decimal.NewFromFloat(157.0)
	signalMA := decimal.NewFromFloat(152.0)

	confidence := strategy.calculateConfidence(shortMA, longMA, currentPrice, signalMA)

	assert.True(t, confidence.GreaterThan(decimal.Zero))
	assert.True(t, confidence.LessThanOrEqual(decimal.NewFromFloat(1.0)))
}

func TestMovingAverageStrategy_GenerateSignal(t *testing.T) {
	config := &models.StrategyConfig{
		ID:      "test_ma",
		Name:    "Test Moving Average",
		Enabled: true,
	}

	strategy := NewMovingAverageStrategy(config)

	tests := []struct {
		name     string
		shortMA  decimal.Decimal
		longMA   decimal.Decimal
		signalMA decimal.Decimal
		price    decimal.Decimal
		expected string
	}{
		{
			name:     "Strong Buy",
			shortMA:  decimal.NewFromFloat(155.0),
			longMA:   decimal.NewFromFloat(150.0),
			signalMA: decimal.NewFromFloat(152.0),
			price:    decimal.NewFromFloat(157.0),
			expected: "strong_buy",
		},
		{
			name:     "Strong Sell",
			shortMA:  decimal.NewFromFloat(145.0),
			longMA:   decimal.NewFromFloat(150.0),
			signalMA: decimal.NewFromFloat(148.0),
			price:    decimal.NewFromFloat(143.0),
			expected: "strong_sell",
		},
		{
			name:     "Weak Buy",
			shortMA:  decimal.NewFromFloat(155.0),
			longMA:   decimal.NewFromFloat(150.0),
			signalMA: decimal.NewFromFloat(152.0),
			price:    decimal.NewFromFloat(149.0),
			expected: "weak_buy",
		},
		{
			name:     "Weak Sell",
			shortMA:  decimal.NewFromFloat(145.0),
			longMA:   decimal.NewFromFloat(150.0),
			signalMA: decimal.NewFromFloat(148.0),
			price:    decimal.NewFromFloat(151.0),
			expected: "weak_sell",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := strategy.generateSignal(tt.shortMA, tt.longMA, tt.signalMA, tt.price)
			assert.Equal(t, tt.expected, signal)
		})
	}
}

func TestMovingAverageStrategy_ValidateOrder(t *testing.T) {
	config := &models.StrategyConfig{
		ID:           "test_ma",
		Name:         "Test Moving Average",
		Enabled:      true,
		MinOrderSize: decimal.NewFromFloat(100.0),
		MaxOrderSize: decimal.NewFromFloat(10000.0),
	}

	strategy := NewMovingAverageStrategy(config)
	portfolio := createTestPortfolio()

	tests := []struct {
		name    string
		order   *models.Order
		wantErr error
	}{
		{
			name: "Valid Buy Order",
			order: &models.Order{
				Symbol:   "AAPL",
				Side:     models.OrderSideBuy,
				Quantity: 10,
				Price:    decimal.NewFromFloat(150.0),
			},
			wantErr: nil,
		},
		{
			name: "Invalid Quantity",
			order: &models.Order{
				Symbol:   "AAPL",
				Side:     models.OrderSideBuy,
				Quantity: 0,
				Price:    decimal.NewFromFloat(150.0),
			},
			wantErr: ErrInvalidQuantity,
		},
		{
			name: "Order Too Small",
			order: &models.Order{
				Symbol:   "AAPL",
				Side:     models.OrderSideBuy,
				Quantity: 1,
				Price:    decimal.NewFromFloat(50.0),
			},
			wantErr: ErrOrderTooSmall,
		},
		{
			name: "Order Too Large",
			order: &models.Order{
				Symbol:   "AAPL",
				Side:     models.OrderSideBuy,
				Quantity: 1000,
				Price:    decimal.NewFromFloat(150.0),
			},
			wantErr: ErrOrderTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := strategy.ValidateOrder(tt.order, portfolio)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func createTestPortfolio() *models.Portfolio {
	return &models.Portfolio{
		ID:             "test_portfolio",
		Cash:           decimal.NewFromFloat(100000.0),
		Positions:      make(map[string]*models.Position),
		TotalValue:     decimal.NewFromFloat(100000.0),
		UnrealizedPnL:  decimal.Zero,
		RealizedPnL:    decimal.Zero,
		TotalRisk:      decimal.Zero,
		RiskMetrics:    models.PortfolioRiskMetrics{},
		TradeHistory:   []*models.Trade{},
		OrderHistory:   []*models.Order{},
		LastRebalanced: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func createTestPortfolioWithHistory() *models.Portfolio {
	portfolio := createTestPortfolio()

	portfolio.TradeHistory = []*models.Trade{
		{
			ID:        "trade1",
			Symbol:    "AAPL",
			Price:     decimal.NewFromFloat(150.0),
			Timestamp: time.Now().Add(-time.Hour * 3),
		},
		{
			ID:        "trade2",
			Symbol:    "AAPL",
			Price:     decimal.NewFromFloat(152.0),
			Timestamp: time.Now().Add(-time.Hour * 2),
		},
		{
			ID:        "trade3",
			Symbol:    "AAPL",
			Price:     decimal.NewFromFloat(155.0),
			Timestamp: time.Now().Add(-time.Hour * 1),
		},
	}

	return portfolio
}

func createTestMarketData() map[string]*models.MarketData {
	return map[string]*models.MarketData{
		"AAPL": {
			Symbol:    "AAPL",
			Price:     decimal.NewFromFloat(155.0),
			Volume:    1000000,
			High:      decimal.NewFromFloat(157.0),
			Low:       decimal.NewFromFloat(153.0),
			Open:      decimal.NewFromFloat(154.0),
			Close:     decimal.NewFromFloat(155.0),
			Timestamp: time.Now(),
		},
		"GOOGL": {
			Symbol:    "GOOGL",
			Price:     decimal.NewFromFloat(2800.0),
			Volume:    500000,
			High:      decimal.NewFromFloat(2810.0),
			Low:       decimal.NewFromFloat(2790.0),
			Open:      decimal.NewFromFloat(2795.0),
			Close:     decimal.NewFromFloat(2800.0),
			Timestamp: time.Now(),
		},
	}
}

func TestMovingAverageStrategy_CalculateRisk(t *testing.T) {
	config := &models.StrategyConfig{
		ID:               "test_ma",
		Name:             "Test Moving Average",
		Enabled:          true,
		MaxPositionSize:  decimal.NewFromFloat(0.2),
		MaxPortfolioRisk: decimal.NewFromFloat(0.15),
	}

	strategy := NewMovingAverageStrategy(config)
	portfolio := createTestPortfolioWithHistory()

	order := &models.Order{
		Symbol:   "AAPL",
		Quantity: 100,
		Price:    decimal.NewFromFloat(155.0),
	}

	riskMetrics, err := strategy.CalculateRisk(order, portfolio)

	require.NoError(t, err)
	assert.NotNil(t, riskMetrics)
	assert.True(t, riskMetrics.VaR95.GreaterThanOrEqual(decimal.Zero))
	assert.True(t, riskMetrics.ExpectedShortfall.GreaterThanOrEqual(decimal.Zero))
	assert.True(t, riskMetrics.Volatility.GreaterThanOrEqual(decimal.Zero))
}
