package strategies

import (
	"context"
	"time"

	"github.com/1cbyc/trade-algo-go/internal/models"
	"github.com/shopspring/decimal"
)

type MovingAverageStrategy struct {
	*BaseStrategy
	shortPeriod  int
	longPeriod   int
	signalPeriod int
}

func NewMovingAverageStrategy(config *models.StrategyConfig) *MovingAverageStrategy {
	return &MovingAverageStrategy{
		BaseStrategy: NewBaseStrategy(config),
		shortPeriod:  10,
		longPeriod:   30,
		signalPeriod: 9,
	}
}

func (s *MovingAverageStrategy) Execute(ctx context.Context, portfolio *models.Portfolio, marketData map[string]*models.MarketData) (*models.AlgorithmResult, error) {
	if !s.IsEnabled() {
		return nil, ErrStrategyDisabled
	}

	var bestSignal *models.AlgorithmResult
	maxConfidence := decimal.Zero

	for symbol, data := range marketData {
		signal, confidence, err := s.analyzeSymbol(symbol, data, portfolio)
		if err != nil {
			continue
		}

		if confidence.GreaterThan(maxConfidence) {
			maxConfidence = confidence
			bestSignal = signal
		}
	}

	return bestSignal, nil
}

func (s *MovingAverageStrategy) analyzeSymbol(symbol string, marketData *models.MarketData, portfolio *models.Portfolio) (*models.AlgorithmResult, decimal.Decimal, error) {
	shortMA := s.calculateSMA(symbol, s.shortPeriod, portfolio)
	longMA := s.calculateSMA(symbol, s.longPeriod, portfolio)
	signalMA := s.calculateSMA(symbol, s.signalPeriod, portfolio)

	if shortMA.IsZero() || longMA.IsZero() || signalMA.IsZero() {
		return nil, decimal.Zero, ErrInvalidMarketData
	}

	currentPrice := marketData.Price
	position, hasPosition := portfolio.Positions[symbol]

	var action string
	var quantity int64
	var confidence decimal.Decimal

	if shortMA.GreaterThan(longMA) && currentPrice.GreaterThan(signalMA) {
		if !hasPosition || position.Quantity <= 0 {
			action = "buy"
			quantity = s.calculateOptimalQuantity(currentPrice, portfolio)
			confidence = s.calculateConfidence(shortMA, longMA, currentPrice, signalMA)
		}
	} else if shortMA.LessThan(longMA) && currentPrice.LessThan(signalMA) {
		if hasPosition && position.Quantity > 0 {
			action = "sell"
			quantity = position.Quantity
			confidence = s.calculateConfidence(longMA, shortMA, signalMA, currentPrice)
		}
	}

	if action == "" || quantity <= 0 {
		return nil, decimal.Zero, nil
	}

	riskMetrics, err := s.calculatePositionRisk(symbol, quantity, currentPrice, portfolio)
	if err != nil {
		return nil, decimal.Zero, err
	}

	return &models.AlgorithmResult{
		StrategyID:     s.ID(),
		Symbol:         symbol,
		Action:         action,
		Quantity:       quantity,
		Price:          currentPrice,
		Confidence:     confidence,
		Signal:         s.generateSignal(shortMA, longMA, signalMA, currentPrice),
		Timestamp:      time.Now(),
		RiskScore:      s.calculateRiskScore(riskMetrics),
		ExpectedReturn: s.calculateExpectedReturn(shortMA, longMA, currentPrice),
	}, confidence, nil
}

func (s *MovingAverageStrategy) calculateSMA(symbol string, period int, portfolio *models.Portfolio) decimal.Decimal {
	if len(portfolio.TradeHistory) < period {
		return decimal.Zero
	}

	var prices []decimal.Decimal
	symbolTrades := s.getTradesForSymbol(symbol, portfolio.TradeHistory)

	if len(symbolTrades) < period {
		return decimal.Zero
	}

	for i := len(symbolTrades) - period; i < len(symbolTrades); i++ {
		prices = append(prices, symbolTrades[i].Price)
	}

	sum := decimal.Zero
	for _, price := range prices {
		sum = sum.Add(price)
	}

	return sum.Div(decimal.NewFromInt(int64(len(prices))))
}

func (s *MovingAverageStrategy) getTradesForSymbol(symbol string, trades []*models.Trade) []*models.Trade {
	var symbolTrades []*models.Trade
	for _, trade := range trades {
		if trade.Symbol == symbol {
			symbolTrades = append(symbolTrades, trade)
		}
	}
	return symbolTrades
}

func (s *MovingAverageStrategy) calculateOptimalQuantity(price decimal.Decimal, portfolio *models.Portfolio) int64 {
	availableCash := portfolio.Cash.Mul(decimal.NewFromFloat(0.95))
	maxQuantity := availableCash.Div(price).IntPart()

	if maxQuantity <= 0 {
		return 0
	}

	config := s.GetConfig()
	maxOrderValue := config.MaxOrderSize
	maxQuantityBySize := maxOrderValue.Div(price).IntPart()

	if maxQuantity > maxQuantityBySize {
		maxQuantity = maxQuantityBySize
	}

	return maxQuantity
}

func (s *MovingAverageStrategy) calculateConfidence(shortMA, longMA, currentPrice, signalMA decimal.Decimal) decimal.Decimal {
	maSpread := shortMA.Sub(longMA).Div(longMA).Abs()
	priceSpread := currentPrice.Sub(signalMA).Div(signalMA).Abs()

	confidence := maSpread.Add(priceSpread).Div(decimal.NewFromFloat(2))

	if confidence.GreaterThan(decimal.NewFromFloat(1.0)) {
		confidence = decimal.NewFromFloat(1.0)
	}

	return confidence
}

func (s *MovingAverageStrategy) calculatePositionRisk(symbol string, quantity int64, price decimal.Decimal, portfolio *models.Portfolio) (*models.RiskMetrics, error) {
	order := &models.Order{
		Symbol:   symbol,
		Quantity: quantity,
		Price:    price,
	}

	return s.CalculateRisk(order, portfolio)
}

func (s *MovingAverageStrategy) calculateRiskScore(riskMetrics *models.RiskMetrics) decimal.Decimal {
	if riskMetrics == nil {
		return decimal.Zero
	}

	volatilityScore := decimal.NewFromFloat(1.0).Sub(riskMetrics.Volatility)
	varScore := decimal.NewFromFloat(1.0).Sub(riskMetrics.VaR95.Div(decimal.NewFromFloat(100)))
	sharpeScore := riskMetrics.SharpeRatio.Div(decimal.NewFromFloat(2.0))

	if sharpeScore.GreaterThan(decimal.NewFromFloat(1.0)) {
		sharpeScore = decimal.NewFromFloat(1.0)
	}

	return volatilityScore.Add(varScore).Add(sharpeScore).Div(decimal.NewFromFloat(3.0))
}

func (s *MovingAverageStrategy) calculateExpectedReturn(shortMA, longMA, currentPrice decimal.Decimal) decimal.Decimal {
	maRatio := shortMA.Div(longMA)
	priceRatio := currentPrice.Div(longMA)

	return maRatio.Add(priceRatio).Div(decimal.NewFromFloat(2.0)).Sub(decimal.NewFromFloat(1.0))
}

func (s *MovingAverageStrategy) generateSignal(shortMA, longMA, signalMA, currentPrice decimal.Decimal) string {
	if shortMA.GreaterThan(longMA) && currentPrice.GreaterThan(signalMA) {
		return "strong_buy"
	} else if shortMA.LessThan(longMA) && currentPrice.LessThan(signalMA) {
		return "strong_sell"
	} else if shortMA.GreaterThan(longMA) {
		return "weak_buy"
	} else {
		return "weak_sell"
	}
}
