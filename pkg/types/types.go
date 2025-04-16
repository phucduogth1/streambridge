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

// StreamBridge is the main interface combining all functionality
type StreamBridge interface {
	// GetStreamProvider returns the appropriate stream provider for the given market type
	GetStreamProvider(marketType MarketType) (StreamProvider, error)

	// GetSupportedMarketTypes returns all market types supported by this exchange
	GetSupportedMarketTypes() []MarketType

	// GetDefaultStreamProvider returns the default stream provider
	GetDefaultStreamProvider() StreamProvider

	// GetName returns the name of the exchange
	GetName() string

	// GetCcxtExchange returns the underlying CCXT exchange object if available
	GetCcxtExchange() interface{}
}
