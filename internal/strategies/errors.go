package strategies

import "errors"

var (
	ErrInvalidQuantity        = errors.New("invalid quantity")
	ErrOrderTooSmall          = errors.New("order too small")
	ErrOrderTooLarge          = errors.New("order too large")
	ErrInsufficientFunds      = errors.New("insufficient funds")
	ErrInsufficientPosition   = errors.New("insufficient position")
	ErrPositionTooLarge       = errors.New("position too large")
	ErrPortfolioRiskExceeded  = errors.New("portfolio risk exceeded")
	ErrStrategyDisabled       = errors.New("strategy is disabled")
	ErrInvalidMarketData      = errors.New("invalid market data")
	ErrInvalidPortfolio       = errors.New("invalid portfolio")
	ErrInvalidConfig          = errors.New("invalid configuration")
	ErrMaxDrawdownExceeded    = errors.New("maximum drawdown exceeded")
	ErrMaxOrdersPerDayReached = errors.New("maximum orders per day reached")
)
