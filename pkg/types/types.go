package types

import (
	"context"
)

// Exchange represents any exchange provider
type Exchange interface {
	GetExchange() interface{}
	GetName() string
}

// OrderBookUpdate represents the structure for order book WebSocket updates
type OrderBookUpdate struct {
	Symbol    string      // Symbol like "BTCUSDT"
	Timestamp int64       // Timestamp in milliseconds
	Bids      [][]string  // Bids as [[price, quantity], ...]
	Asks      [][]string  // Asks as [[price, quantity], ...]
	Raw       interface{} // Raw exchange-specific data
}

// TradeUpdate represents the structure for trade WebSocket updates
type TradeUpdate struct {
	Symbol    string      // Symbol like "BTCUSDT"
	Timestamp int64       // Timestamp in milliseconds
	ID        int64       // Trade ID
	Price     string      // Trade price
	Amount    string      // Trade amount
	Side      string      // Buy or sell
	Raw       interface{} // Raw exchange-specific data
}

// TickerUpdate represents the structure for ticker WebSocket updates
type TickerUpdate struct {
	Symbol        string      // Symbol like "BTCUSDT"
	Timestamp     int64       // Timestamp in milliseconds
	Last          string      // Last price
	High          string      // 24h high
	Low           string      // 24h low
	Volume        string      // 24h base volume
	QuoteVolume   string      // 24h quote volume
	Change        string      // Price change
	ChangePercent string      // Price change percentage
	Raw           interface{} // Raw exchange-specific data
}

// OrderUpdate represents the structure for order updates via user data stream
type OrderUpdate struct {
	Symbol             string      // Symbol like "BTCUSDT"
	Timestamp          int64       // Timestamp in milliseconds
	ClientOrderID      string      // Client order ID
	OrderID            int64       // Exchange order ID
	Side               string      // Buy or sell
	Type               string      // Order type (LIMIT, MARKET, etc)
	TimeInForce        string      // Time in force
	OriginalQty        string      // Original quantity
	ExecutedQty        string      // Executed quantity
	CumulativeQuoteQty string      // Cumulative quote quantity
	Price              string      // Order price
	Status             string      // Order status
	Raw                interface{} // Raw exchange-specific data
}

// BalanceUpdate represents the structure for account balance updates
type BalanceUpdate struct {
	Timestamp int64                   // Timestamp in milliseconds
	Balances  map[string]AssetBalance // Map of asset to balance
	Raw       interface{}             // Raw exchange-specific data
}

// AssetBalance represents the balance of a single asset
type AssetBalance struct {
	Free   string // Free balance
	Locked string // Locked balance (in orders)
	Total  string // Total balance (free + locked)
}

// PositionUpdate represents the structure for futures position updates
type PositionUpdate struct {
	Symbol           string      // Symbol like "BTCUSDT"
	Timestamp        int64       // Timestamp in milliseconds
	Side             string      // Position side (LONG, SHORT, BOTH)
	EntryPrice       string      // Entry price
	MarkPrice        string      // Mark price
	PositionAmt      string      // Position amount
	UnrealizedProfit string      // Unrealized profit
	Leverage         int         // Leverage used
	MarginType       string      // Margin type (ISOLATED, CROSSED)
	IsolatedMargin   string      // Isolated margin if applicable
	LiquidationPrice string      // Liquidation price
	Raw              interface{} // Raw exchange-specific data
}

// MarketType represents different market types (spot, futures, etc)
type MarketType string

const (
	// Spot market type
	Spot MarketType = "spot"
	// Futures market type
	Futures MarketType = "futures"
	// Margin market type
	Margin MarketType = "margin"
	// Options market type
	Options MarketType = "options"
	// UserData for user data streams
	UserData MarketType = "userdata"
)

// StreamProvider defines the interface for WebSocket stream functionality
type StreamProvider interface {
	// WatchOrderBook returns a channel of order book updates and a cancel function
	WatchOrderBook(ctx context.Context, symbol string, options map[string]interface{}) (chan OrderBookUpdate, func(), error)

	// WatchTrades returns a channel of trade updates and a cancel function
	WatchTrades(ctx context.Context, symbol string, options map[string]interface{}) (chan TradeUpdate, func(), error)

	// WatchTicker returns a channel of ticker updates and a cancel function
	WatchTicker(ctx context.Context, symbol string, options map[string]interface{}) (chan TickerUpdate, func(), error)

	// GetMarketType returns the market type (spot, futures, etc.)
	GetMarketType() MarketType

	// GetExchange returns the underlying exchange implementation
	GetExchange() Exchange
}

// UserDataProvider defines the interface for user data stream functionality
type UserDataProvider interface {
	// WatchOrders returns a channel of order updates and a cancel function
	WatchOrders(ctx context.Context, options map[string]interface{}) (chan OrderUpdate, func(), error)

	// WatchBalances returns a channel of balance updates and a cancel function
	WatchBalances(ctx context.Context, options map[string]interface{}) (chan BalanceUpdate, func(), error)

	// GetExchange returns the underlying exchange implementation
	GetExchange() Exchange
}

// FuturesUserDataProvider extends UserDataProvider with futures-specific functionality
type FuturesUserDataProvider interface {
	UserDataProvider

	// WatchPositions returns a channel of position updates and a cancel function
	WatchPositions(ctx context.Context, options map[string]interface{}) (chan PositionUpdate, func(), error)
}

// StreamBridge is the main interface combining all functionality
type StreamBridge interface {
	// GetStreamProvider returns the appropriate stream provider for the given market type
	GetStreamProvider(marketType MarketType) (StreamProvider, error)

	// GetUserDataProvider returns the user data provider if available
	GetUserDataProvider() (UserDataProvider, error)

	// GetSupportedMarketTypes returns all market types supported by this exchange
	GetSupportedMarketTypes() []MarketType

	// GetDefaultStreamProvider returns the default stream provider
	GetDefaultStreamProvider() StreamProvider

	// GetName returns the name of the exchange
	GetName() string

	// GetCcxtExchange returns the underlying CCXT exchange object if available
	GetCcxtExchange() interface{}
}
