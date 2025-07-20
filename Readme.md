# Trade Algorithm Go

An algorithmic trading system built in Go for developing, testing, and executing trading strategies in a simulated market environment.

## Features

### Core Trading Engine
- **Real-time Order Processing**: High-performance order validation and execution
- **Portfolio Management**: Comprehensive portfolio tracking with P&L calculations
- **Risk Management**: Advanced risk metrics including VaR, Sharpe ratio, and drawdown analysis
- **Concurrent Processing**: Multi-threaded architecture for high-frequency operations

### Trading Strategies
- **Moving Average Crossover**: Sophisticated MA strategy with signal confirmation
- **Extensible Framework**: Easy to add new strategies with the Strategy interface
- **Risk-Adjusted Sizing**: Position sizing based on portfolio constraints and risk limits
- **Confidence Scoring**: Signal strength assessment for trade decisions

### Market Simulation
- **Realistic Price Generation**: Normal distribution with volatility modeling
- **Volume Simulation**: Dynamic volume changes with realistic patterns
- **Trend Modeling**: Gradual trend changes over time
- **Market Events**: Support for price shocks and volatility spikes

### Risk Management
- **Position-Level Risk**: VaR, Expected Shortfall, Volatility, Beta calculations
- **Portfolio-Level Risk**: Total risk exposure and correlation analysis
- **Risk Controls**: Stop loss, take profit, trailing stops, position limits
- **Real-time Monitoring**: Continuous risk assessment and alerting

## Quick Start

### Prerequisites
- Go 1.21 or higher
- Git

### Installation

```bash
git clone https://github.com/1cbyc/trade-algo-go.git
cd trade-algo-go
go mod tidy
```

### Basic Usage

```bash
# Run with default settings (5 minutes, $100k initial cash)
go run main.go

# Run with custom parameters
go run main.go -cash 500000 -duration 10m -log-level debug

# Build and run
go build -o trade-algo-go
./trade-algo-go -cash 1000000 -duration 1h
```

### Command Line Options

- `-cash`: Initial portfolio cash (default: $100,000)
- `-duration`: Simulation duration (default: 5 minutes)
- `-log-level`: Logging level - debug, info, warn, error (default: info)

## Architecture

<!-- ### Project Structure
```
trade-algo-go/
├── internal/
│   ├── models/          # Data structures and types
│   ├── engine/          # Trading engine core
│   ├── strategies/      # Trading strategy implementations
│   └── simulator/       # Market data simulation
├── docs/               # Documentation
├── main.go            # Application entry point
├── go.mod             # Go module definition
└── README.md          # This file
``` -->

### Key Components

#### Trading Engine (`internal/engine/`)
- **Order Processing**: Validates and executes trading orders
- **Trade Recording**: Maintains comprehensive trade history
- **Strategy Execution**: Runs trading algorithms periodically
- **Risk Management**: Monitors portfolio risk levels
- **Portfolio Updates**: Real-time portfolio value calculations

#### Strategies (`internal/strategies/`)
- **Base Strategy**: Common functionality for all strategies
- **Moving Average Strategy**: MA crossover implementation
- **Strategy Interface**: Contract for implementing new strategies
- **Risk Calculation**: Position and portfolio risk assessment

#### Market Simulator (`internal/simulator/`)
- **Price Generation**: Realistic price movements using normal distribution
- **Volume Simulation**: Dynamic volume changes
- **Trend Modeling**: Gradual market trend changes
- **Event System**: Market events and volatility spikes

## Trading Strategies

### Moving Average Crossover Strategy

This strategy implements a sophisticated moving average crossover system:

**Components:**
- **Short-term MA**: 10-period simple moving average
- **Long-term MA**: 30-period simple moving average
- **Signal MA**: 9-period signal line for confirmation

**Logic:**
1. **Buy Signal**: When short MA > long MA AND price > signal MA
2. **Sell Signal**: When short MA < long MA AND price < signal MA
3. **Position Sizing**: Based on available cash and risk limits
4. **Confidence Scoring**: Calculated from MA spread and price deviation

**Risk Features:**
- Volatility-adjusted position sizing
- Maximum position size limits (20% of portfolio)
- Portfolio risk concentration checks
- Drawdown monitoring and alerts

## Risk Management

### Position-Level Risk Metrics
- **VaR (Value at Risk)**: 95% confidence level risk calculation
- **Expected Shortfall**: Average loss beyond VaR threshold
- **Volatility**: Price movement standard deviation
- **Beta**: Market correlation measure
- **Max Drawdown**: Maximum peak-to-trough decline

### Portfolio-Level Risk Controls
- **Total Risk Limit**: Maximum 15% portfolio risk exposure
- **Position Limits**: Maximum 20% in any single position
- **Stop Loss**: 5% automatic position closure
- **Take Profit**: 10% automatic position closure
- **Trailing Stop**: 3% dynamic stop loss

## Configuration

### Strategy Configuration
```go
MaxPositionSize: 0.2        // 20% max position size
MaxPortfolioRisk: 0.15      // 15% max portfolio risk
StopLossPercent: 0.05       // 5% stop loss
TakeProfitPercent: 0.1      // 10% take profit
CommissionRate: 0.001       // 0.1% commission
```

### Market Configuration
- **Symbols**: AAPL, GOOGL, MSFT, TSLA, AMZN, NFLX, NVDA, META
- **Base Prices**: Realistic starting prices
- **Volatility**: Symbol-specific volatility levels
- **Update Frequency**: 1-second price updates

## Performance

### System Performance
- **Order Processing**: < 1ms latency
- **Strategy Execution**: Every 5 seconds
- **Risk Monitoring**: Every 10 seconds
- **Portfolio Updates**: Every 1 second
- **Concurrent Processing**: Multiple goroutines for high throughput

### Memory Management
- **Channel Buffering**: Prevents blocking on high-frequency updates
- **Mutex Protection**: Thread-safe data access
- **Efficient Data Structures**: Maps for O(1) symbol lookups
- **Decimal Arithmetic**: Precise financial calculations

## Monitoring and Logging

### Logging Levels
- **Debug**: Detailed execution traces and calculations
- **Info**: General system status and portfolio updates
- **Warn**: Risk threshold warnings and alerts
- **Error**: System errors and execution failures

### Metrics Tracked
- Portfolio value and P&L
- Position details and performance
- Trade execution statistics
- Risk metric calculations
- Strategy performance indicators

## Extending the System

### Adding New Strategies

1. Implement the Strategy interface:
```go
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
```

2. Register with the trading engine:
```go
strategy := NewYourStrategy(config)
engine.AddStrategy(strategy)
```

### Adding New Risk Models

1. Extend the RiskMetrics structure
2. Implement calculation methods
3. Add to strategy risk assessment
4. Update monitoring systems

### Market Data Integration

1. Implement market data interface
2. Connect to external data feeds
3. Transform data to internal format
4. Replace simulator with real data

## Production Considerations

### Scalability
- Horizontal scaling with multiple engines
- Database integration for persistence
- Message queues for order routing
- Load balancing for high-frequency trading

### Reliability
- Circuit breakers for risk protection
- Graceful degradation on failures
- Comprehensive error handling
- Health monitoring and alerting

### Security
- API authentication and authorization
- Encrypted data transmission
- Audit logging for compliance
- Input validation and sanitization

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

### Code Style

- Follow Go conventions and best practices
- Use meaningful variable and function names
- Add comments for complex logic
- Ensure all tests pass
- Update documentation as needed

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: Report bugs and request features on GitHub
- **Discussions**: Join community discussions
- **Documentation**: Check the [docs/](docs/) directory
- **Examples**: See the main.go file for usage examples

## Roadmap

See [docs/whats-next.md](docs/whats-next.md) for our development roadmap and future plans.

## Acknowledgments

- Built with Go and modern financial libraries
- Inspired by professional trading systems
- Designed for both educational and production use
- Community-driven development

---

**Disclaimer**: This software is for educational and research purposes. Use at your own risk. I am not responsible for any financial losses incurred through the use of this software.
