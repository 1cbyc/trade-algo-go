package simulator

import (
	"math/rand"
	"sync"
	"time"

	"github.com/1cbyc/trade-algo-go/internal/models"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type MarketSimulator struct {
	symbols    map[string]*SymbolData
	logger     *zap.Logger
	mu         sync.RWMutex
	running    bool
	stopChan   chan struct{}
	updateChan chan *models.MarketData
}

type SymbolData struct {
	Symbol       string
	BasePrice    decimal.Decimal
	CurrentPrice decimal.Decimal
	Volatility   decimal.Decimal
	Trend        decimal.Decimal
	Volume       int64
	High         decimal.Decimal
	Low          decimal.Decimal
	Open         decimal.Decimal
	Close        decimal.Decimal
	LastUpdate   time.Time
}

func NewMarketSimulator(logger *zap.Logger) *MarketSimulator {
	return &MarketSimulator{
		symbols:    make(map[string]*SymbolData),
		logger:     logger,
		stopChan:   make(chan struct{}),
		updateChan: make(chan *models.MarketData, 1000),
	}
}

func (s *MarketSimulator) AddSymbol(symbol string, basePrice decimal.Decimal, volatility decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.symbols[symbol] = &SymbolData{
		Symbol:       symbol,
		BasePrice:    basePrice,
		CurrentPrice: basePrice,
		Volatility:   volatility,
		Trend:        decimal.Zero,
		Volume:       rand.Int63n(1000000) + 100000,
		High:         basePrice,
		Low:          basePrice,
		Open:         basePrice,
		Close:        basePrice,
		LastUpdate:   time.Now(),
	}

	s.logger.Info("Symbol added to simulator", zap.String("symbol", symbol), zap.String("base_price", basePrice.String()))
}

func (s *MarketSimulator) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Market simulator started")

	go s.priceGenerator()
	go s.volumeGenerator()
	go s.trendGenerator()
}

func (s *MarketSimulator) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}

	s.running = false
	close(s.stopChan)
	s.logger.Info("Market simulator stopped")
}

func (s *MarketSimulator) GetUpdateChannel() <-chan *models.MarketData {
	return s.updateChan
}

func (s *MarketSimulator) priceGenerator() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updatePrices()
		case <-s.stopChan:
			return
		}
	}
}

func (s *MarketSimulator) volumeGenerator() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateVolumes()
		case <-s.stopChan:
			return
		}
	}
}

func (s *MarketSimulator) trendGenerator() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateTrends()
		case <-s.stopChan:
			return
		}
	}
}

func (s *MarketSimulator) updatePrices() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for symbol, data := range s.symbols {
		priceChange := s.calculatePriceChange(data)
		newPrice := data.CurrentPrice.Add(priceChange)

		if newPrice.LessThanOrEqual(decimal.Zero) {
			newPrice = decimal.NewFromFloat(0.01)
		}

		data.Open = data.CurrentPrice
		data.CurrentPrice = newPrice
		data.Close = newPrice
		data.LastUpdate = time.Now()

		if newPrice.GreaterThan(data.High) {
			data.High = newPrice
		}
		if newPrice.LessThan(data.Low) {
			data.Low = newPrice
		}

		marketData := &models.MarketData{
			Symbol:    symbol,
			Price:     newPrice,
			Volume:    data.Volume,
			High:      data.High,
			Low:       data.Low,
			Open:      data.Open,
			Close:     data.Close,
			Timestamp: time.Now(),
		}

		select {
		case s.updateChan <- marketData:
		default:
			s.logger.Warn("Update channel full, dropping market data", zap.String("symbol", symbol))
		}
	}
}

func (s *MarketSimulator) calculatePriceChange(data *SymbolData) decimal.Decimal {
	randomFactor := decimal.NewFromFloat(rand.NormFloat64())
	volatilityImpact := data.Volatility.Mul(randomFactor)
	trendImpact := data.Trend.Mul(decimal.NewFromFloat(0.1))

	priceChange := volatilityImpact.Add(trendImpact)

	priceChangePercent := priceChange.Div(data.CurrentPrice)

	if priceChangePercent.Abs().GreaterThan(decimal.NewFromFloat(0.1)) {
		if priceChangePercent.IsPositive() {
			priceChangePercent = decimal.NewFromFloat(0.1)
		} else {
			priceChangePercent = decimal.NewFromFloat(-0.1)
		}
	}

	return data.CurrentPrice.Mul(priceChangePercent)
}

func (s *MarketSimulator) updateVolumes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, data := range s.symbols {
		volumeChange := rand.Int63n(100000) - 50000
		newVolume := data.Volume + volumeChange

		if newVolume < 10000 {
			newVolume = 10000
		}

		data.Volume = newVolume
	}
}

func (s *MarketSimulator) updateTrends() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, data := range s.symbols {
		trendChange := decimal.NewFromFloat(rand.NormFloat64() * 0.01)
		data.Trend = data.Trend.Add(trendChange)

		if data.Trend.Abs().GreaterThan(decimal.NewFromFloat(0.05)) {
			if data.Trend.IsPositive() {
				data.Trend = decimal.NewFromFloat(0.05)
			} else {
				data.Trend = decimal.NewFromFloat(-0.05)
			}
		}
	}
}

func (s *MarketSimulator) GetSymbolData(symbol string) *SymbolData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if data, exists := s.symbols[symbol]; exists {
		return data
	}
	return nil
}

func (s *MarketSimulator) GetAllSymbols() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	symbols := make([]string, 0, len(s.symbols))
	for symbol := range s.symbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

func (s *MarketSimulator) SetVolatility(symbol string, volatility decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if data, exists := s.symbols[symbol]; exists {
		data.Volatility = volatility
		s.logger.Info("Volatility updated", zap.String("symbol", symbol), zap.String("volatility", volatility.String()))
	}
}

func (s *MarketSimulator) SetTrend(symbol string, trend decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if data, exists := s.symbols[symbol]; exists {
		data.Trend = trend
		s.logger.Info("Trend updated", zap.String("symbol", symbol), zap.String("trend", trend.String()))
	}
}

func (s *MarketSimulator) AddMarketEvent(symbol string, eventType string, impact decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if data, exists := s.symbols[symbol]; exists {
		switch eventType {
		case "price_shock":
			data.CurrentPrice = data.CurrentPrice.Mul(decimal.NewFromFloat(1.0).Add(impact))
		case "volatility_spike":
			data.Volatility = data.Volatility.Mul(decimal.NewFromFloat(1.0).Add(impact))
		case "trend_change":
			data.Trend = data.Trend.Add(impact)
		}

		s.logger.Info("Market event applied",
			zap.String("symbol", symbol),
			zap.String("event_type", eventType),
			zap.String("impact", impact.String()))
	}
}
