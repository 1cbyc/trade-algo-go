package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
	OrderTypeStop   OrderType = "stop"
)

type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRejected  OrderStatus = "rejected"
)

type Trade struct {
	ID          string          `json:"id"`
	OrderID     string          `json:"order_id"`
	Symbol      string          `json:"symbol"`
	Side        OrderSide       `json:"side"`
	Quantity    int64           `json:"quantity"`
	Price       decimal.Decimal `json:"price"`
	Commission  decimal.Decimal `json:"commission"`
	Timestamp   time.Time       `json:"timestamp"`
	StrategyID  string          `json:"strategy_id"`
	RiskMetrics RiskMetrics     `json:"risk_metrics"`
}

type Order struct {
	ID          string          `json:"id"`
	Symbol      string          `json:"symbol"`
	Side        OrderSide       `json:"side"`
	Type        OrderType       `json:"type"`
	Quantity    int64           `json:"quantity"`
	Price       decimal.Decimal `json:"price"`
	StopPrice   decimal.Decimal `json:"stop_price"`
	Status      OrderStatus     `json:"status"`
	Timestamp   time.Time       `json:"timestamp"`
	StrategyID  string          `json:"strategy_id"`
	RiskMetrics RiskMetrics     `json:"risk_metrics"`
}

type Position struct {
	Symbol        string          `json:"symbol"`
	Quantity      int64           `json:"quantity"`
	AveragePrice  decimal.Decimal `json:"average_price"`
	CurrentPrice  decimal.Decimal `json:"current_price"`
	UnrealizedPnL decimal.Decimal `json:"unrealized_pnl"`
	RealizedPnL   decimal.Decimal `json:"realized_pnl"`
	MarketValue   decimal.Decimal `json:"market_value"`
	RiskMetrics   RiskMetrics     `json:"risk_metrics"`
	LastUpdated   time.Time       `json:"last_updated"`
}

type Portfolio struct {
	ID             string               `json:"id"`
	Cash           decimal.Decimal      `json:"cash"`
	Positions      map[string]*Position `json:"positions"`
	TotalValue     decimal.Decimal      `json:"total_value"`
	UnrealizedPnL  decimal.Decimal      `json:"unrealized_pnl"`
	RealizedPnL    decimal.Decimal      `json:"realized_pnl"`
	TotalRisk      decimal.Decimal      `json:"total_risk"`
	RiskMetrics    PortfolioRiskMetrics `json:"risk_metrics"`
	TradeHistory   []*Trade             `json:"trade_history"`
	OrderHistory   []*Order             `json:"order_history"`
	LastRebalanced time.Time            `json:"last_rebalanced"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

type MarketData struct {
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Volume    int64           `json:"volume"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Open      decimal.Decimal `json:"open"`
	Close     decimal.Decimal `json:"close"`
	Timestamp time.Time       `json:"timestamp"`
}

type RiskMetrics struct {
	VaR95             decimal.Decimal `json:"var_95"`
	ExpectedShortfall decimal.Decimal `json:"expected_shortfall"`
	SharpeRatio       decimal.Decimal `json:"sharpe_ratio"`
	MaxDrawdown       decimal.Decimal `json:"max_drawdown"`
	Volatility        decimal.Decimal `json:"volatility"`
	Beta              decimal.Decimal `json:"beta"`
}

type PortfolioRiskMetrics struct {
	TotalVaR95      decimal.Decimal `json:"total_var_95"`
	TotalES         decimal.Decimal `json:"total_es"`
	PortfolioBeta   decimal.Decimal `json:"portfolio_beta"`
	Correlation     decimal.Decimal `json:"correlation"`
	Diversification decimal.Decimal `json:"diversification"`
}

type StrategyConfig struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	MaxPositionSize     decimal.Decimal `json:"max_position_size"`
	MaxPortfolioRisk    decimal.Decimal `json:"max_portfolio_risk"`
	MaxDrawdown         decimal.Decimal `json:"max_drawdown"`
	StopLossPercent     decimal.Decimal `json:"stop_loss_percent"`
	TakeProfitPercent   decimal.Decimal `json:"take_profit_percent"`
	TrailingStopPercent decimal.Decimal `json:"trailing_stop_percent"`
	RebalanceThreshold  decimal.Decimal `json:"rebalance_threshold"`
	MaxOrdersPerDay     int             `json:"max_orders_per_day"`
	MinOrderSize        decimal.Decimal `json:"min_order_size"`
	MaxOrderSize        decimal.Decimal `json:"max_order_size"`
	CommissionRate      decimal.Decimal `json:"commission_rate"`
	SlippageTolerance   decimal.Decimal `json:"slippage_tolerance"`
	RiskFreeRate        decimal.Decimal `json:"risk_free_rate"`
	MarketDataWindow    int             `json:"market_data_window"`
	TechnicalIndicators []string        `json:"technical_indicators"`
	Enabled             bool            `json:"enabled"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type AlgorithmResult struct {
	StrategyID     string          `json:"strategy_id"`
	Symbol         string          `json:"symbol"`
	Action         string          `json:"action"`
	Quantity       int64           `json:"quantity"`
	Price          decimal.Decimal `json:"price"`
	Confidence     decimal.Decimal `json:"confidence"`
	Signal         string          `json:"signal"`
	Timestamp      time.Time       `json:"timestamp"`
	RiskScore      decimal.Decimal `json:"risk_score"`
	ExpectedReturn decimal.Decimal `json:"expected_return"`
}
