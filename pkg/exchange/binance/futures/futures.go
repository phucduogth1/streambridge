package futures

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/phucduogth1/streambridge/pkg/exchange/binance"
	"github.com/phucduogth1/streambridge/pkg/types"
)

// BinanceFuturesProvider implements the StreamProvider interface for Binance Futures
type BinanceFuturesProvider struct {
	exchange       *binance.BinanceExchange
	subManager     *SubscriptionManager
	subManagerLock sync.Once
}

// NewBinanceFuturesProvider creates a new Binance futures provider
func NewBinanceFuturesProvider(exchange *binance.BinanceExchange) *BinanceFuturesProvider {
	return &BinanceFuturesProvider{
		exchange: exchange,
	}
}

// GetMarketType returns the market type
func (b *BinanceFuturesProvider) GetMarketType() types.MarketType {
	return types.Futures
}

// GetExchange returns the underlying exchange implementation
func (b *BinanceFuturesProvider) GetExchange() types.Exchange {
	return b.exchange
}

// binanceOrderBookUpdate represents the structure for Binance Futures order book WebSocket updates
type binanceOrderBookUpdate struct {
	EventType     string     `json:"e"` // e.g., "depthUpdate"
	EventTime     int64      `json:"E"` // Event timestamp (ms)
	Symbol        string     `json:"s"` // e.g., "BTCUSDT"
	FirstUpdateID int64      `json:"U"` // First update ID in event
	FinalUpdateID int64      `json:"u"` // Final update ID in event
	Bids          [][]string `json:"b"` // Bid updates: [[price, qty], ...]
	Asks          [][]string `json:"a"` // Ask updates: [[price, qty], ...]
}

// binanceTradeUpdate represents the structure for Binance Futures trade WebSocket updates
type binanceTradeUpdate struct {
	EventType string `json:"e"` // e.g., "trade"
	EventTime int64  `json:"E"` // Event timestamp (ms)
	Symbol    string `json:"s"` // e.g., "BTCUSDT"
	TradeID   int64  `json:"t"` // Trade ID
	Price     string `json:"p"` // Trade price
	Quantity  string `json:"q"` // Trade quantity
	BuyerID   int64  `json:"b"` // Buyer order ID
	SellerID  int64  `json:"a"` // Seller order ID
	TradeTime int64  `json:"T"` // Trade time
	Maker     bool   `json:"m"` // Is maker
}

// binanceTickerUpdate represents the structure for Binance Futures ticker WebSocket updates
type binanceTickerUpdate struct {
	EventType    string `json:"e"` // e.g., "24hrTicker"
	EventTime    int64  `json:"E"` // Event timestamp (ms)
	Symbol       string `json:"s"` // e.g., "BTCUSDT"
	PriceChange  string `json:"p"` // Price change
	PriceChangeP string `json:"P"` // Price change percent
	WeightedAvgP string `json:"w"` // Weighted average price
	LastPrice    string `json:"c"` // Last price
	CloseQty     string `json:"Q"` // Last qty
	HighPrice    string `json:"h"` // High price
	LowPrice     string `json:"l"` // Low price
	BaseVolume   string `json:"v"` // Total traded base asset volume
	QuoteVolume  string `json:"q"` // Total traded quote asset volume
	OpenTime     int64  `json:"O"` // Statistics open time
	CloseTime    int64  `json:"C"` // Statistics close time
	FirstTradeId int64  `json:"F"` // First trade ID
	LastTradeId  int64  `json:"L"` // Last trade ID
	TradeCount   int64  `json:"n"` // Total number of trades
}

// binanceStreamMessage represents the outer structure for Binance WebSocket messages
type binanceStreamMessage struct {
	Stream string          `json:"stream"` // Stream name
	Data   json.RawMessage `json:"data"`   // Raw data
}

// getSubscriptionManager returns the subscription manager, initializing it if needed
func (b *BinanceFuturesProvider) getSubscriptionManager() *SubscriptionManager {
	b.subManagerLock.Do(func() {
		b.subManager = NewSubscriptionManager(b.exchange.IsTestnet())
	})
	return b.subManager
}

// WatchOrderBook streams real-time order book updates for the given symbol
func (b *BinanceFuturesProvider) WatchOrderBook(ctx context.Context, symbol string, options map[string]interface{}) (chan types.OrderBookUpdate, func(), error) {
	sm := b.getSubscriptionManager()

	// Subscribe to the order book stream
	ch, cancel, err := sm.Subscribe(strings.ToLower(symbol), OrderBookStream)
	if err != nil {
		return nil, nil, err
	}

	// Create a typed channel for order book updates
	typedCh := make(chan types.OrderBookUpdate, 100)

	// Start a goroutine to convert and forward messages
	go func() {
		defer close(typedCh)
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if update, ok := msg.(types.OrderBookUpdate); ok {
					select {
					case <-ctx.Done():
						return
					case typedCh <- update:
					}
				}
			}
		}
	}()

	return typedCh, cancel, nil
}

// WatchTrades streams real-time trade updates for the given symbol
func (b *BinanceFuturesProvider) WatchTrades(ctx context.Context, symbol string, options map[string]interface{}) (chan types.TradeUpdate, func(), error) {
	sm := b.getSubscriptionManager()

	// Subscribe to the trade stream
	ch, cancel, err := sm.Subscribe(strings.ToLower(symbol), TradeStream)
	if err != nil {
		return nil, nil, err
	}

	// Create a typed channel for trade updates
	typedCh := make(chan types.TradeUpdate, 100)

	// Start a goroutine to convert and forward messages
	go func() {
		defer close(typedCh)
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if update, ok := msg.(types.TradeUpdate); ok {
					select {
					case <-ctx.Done():
						return
					case typedCh <- update:
					}
				}
			}
		}
	}()

	return typedCh, cancel, nil
}

// WatchTicker streams real-time ticker updates for the given symbol
func (b *BinanceFuturesProvider) WatchTicker(ctx context.Context, symbol string, options map[string]interface{}) (chan types.TickerUpdate, func(), error) {
	sm := b.getSubscriptionManager()

	// Subscribe to the ticker stream
	ch, cancel, err := sm.Subscribe(strings.ToLower(symbol), TickerStream)
	if err != nil {
		return nil, nil, err
	}

	// Create a typed channel for ticker updates
	typedCh := make(chan types.TickerUpdate, 100)

	// Start a goroutine to convert and forward messages
	go func() {
		defer close(typedCh)
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if update, ok := msg.(types.TickerUpdate); ok {
					select {
					case <-ctx.Done():
						return
					case typedCh <- update:
					}
				}
			}
		}
	}()

	return typedCh, cancel, nil
}
