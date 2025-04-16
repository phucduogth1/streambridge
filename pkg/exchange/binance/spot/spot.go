package spot

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/phucduogth1/streambridge/pkg/exchange/binance"
	"github.com/phucduogth1/streambridge/pkg/types"
)

// BinanceSpotProvider implements the StreamProvider interface for Binance Spot
type BinanceSpotProvider struct {
	exchange *binance.BinanceExchange
}

// NewBinanceSpotProvider creates a new Binance spot provider
func NewBinanceSpotProvider(exchange *binance.BinanceExchange) *BinanceSpotProvider {
	return &BinanceSpotProvider{
		exchange: exchange,
	}
}

// GetMarketType returns the market type
func (b *BinanceSpotProvider) GetMarketType() types.MarketType {
	return types.Spot
}

// GetExchange returns the underlying exchange implementation
func (b *BinanceSpotProvider) GetExchange() types.Exchange {
	return b.exchange
}

// binanceSpotOrderBookUpdate represents the structure for Binance Spot order book WebSocket updates
type binanceSpotOrderBookUpdate struct {
	EventType     string     `json:"e"` // e.g., "depthUpdate"
	EventTime     int64      `json:"E"` // Event timestamp (ms)
	Symbol        string     `json:"s"` // e.g., "BTCUSDT"
	FirstUpdateID int64      `json:"U"` // First update ID in event
	FinalUpdateID int64      `json:"u"` // Final update ID in event
	Bids          [][]string `json:"b"` // Bid updates: [[price, qty], ...]
	Asks          [][]string `json:"a"` // Ask updates: [[price, qty], ...]
}

// binanceSpotTradeUpdate represents the structure for Binance Spot trade WebSocket updates
type binanceSpotTradeUpdate struct {
	EventType    string `json:"e"` // e.g., "trade"
	EventTime    int64  `json:"E"` // Event timestamp (ms)
	Symbol       string `json:"s"` // e.g., "BTCUSDT"
	TradeID      int64  `json:"t"` // Trade ID
	Price        string `json:"p"` // Trade price
	Quantity     string `json:"q"` // Trade quantity
	BuyerID      int64  `json:"b"` // Buyer order ID
	SellerID     int64  `json:"a"` // Seller order ID
	TradeTime    int64  `json:"T"` // Trade time
	IsBuyerMaker bool   `json:"m"` // Is buyer the maker
}

// connectWebSocket establishes a WebSocket connection with the given URL
func connectWebSocket(url string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	// Set read deadline for initial connection
	err = conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Setup pong handler to reset deadline
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	})

	return conn, nil
}

// WatchOrderBook streams real-time order book updates for the given symbol
func (b *BinanceSpotProvider) WatchOrderBook(ctx context.Context, symbol string, options map[string]interface{}) (chan types.OrderBookUpdate, func(), error) {
	updates := make(chan types.OrderBookUpdate)
	_, cancel := context.WithCancel(ctx)

	go func() {
		defer close(updates)

		if b.exchange.IsTestnet() {
			// Note: Binance spot doesn't have an official testnet WebSocket
			// This is a placeholder and would need to be updated with correct URL if available
			return
		} else {
			url := "wss://stream.binance.com:9443/ws/" + strings.ToLower(symbol) + "@depth"
			log.Printf("Connecting to OrderBook WebSocket URL: %s", url)
		}

		// WebSocket connection and message handling would be similar to futures implementation
		// For brevity, this implementation is omitted in this skeleton
	}()

	return updates, cancel, nil
}

// WatchTrades streams real-time trade updates for the given symbol
func (b *BinanceSpotProvider) WatchTrades(ctx context.Context, symbol string, options map[string]interface{}) (chan types.TradeUpdate, func(), error) {
	updates := make(chan types.TradeUpdate)
	_, cancel := context.WithCancel(ctx)

	go func() {
		defer close(updates)

		if b.exchange.IsTestnet() {
			// Note: Binance spot doesn't have an official testnet WebSocket
			// This is a placeholder and would need to be updated with correct URL if available
			return
		} else {
			url := "wss://stream.binance.com:9443/ws/" + strings.ToLower(symbol) + "@trade"
			log.Printf("Connecting to Trades WebSocket URL: %s", url)
		}

		// WebSocket connection and message handling would be similar to futures implementation
		// For brevity, this implementation is omitted in this skeleton
	}()

	return updates, cancel, nil
}

// WatchTicker streams real-time ticker updates for the given symbol
func (b *BinanceSpotProvider) WatchTicker(ctx context.Context, symbol string, options map[string]interface{}) (chan types.TickerUpdate, func(), error) {
	updates := make(chan types.TickerUpdate)
	_, cancel := context.WithCancel(ctx)

	go func() {
		defer close(updates)

		if b.exchange.IsTestnet() {
			// Note: Binance spot doesn't have an official testnet WebSocket
			return
		} else {
			url := "wss://stream.binance.com:9443/ws/" + strings.ToLower(symbol) + "@ticker"
			log.Printf("Connecting to Ticker WebSocket URL: %s", url)
		}

		// WebSocket connection and message handling would be similar to futures implementation
		// For brevity, this implementation is omitted in this skeleton
	}()

	return updates, cancel, nil
}
