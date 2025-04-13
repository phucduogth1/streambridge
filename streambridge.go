package streambridge

import (
	"context"
	"log"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/gorilla/websocket"
)

// StreamBridge represents the main struct for the enhanced CCXT package
type StreamBridge struct {
	exchange ccxt.Binance
}

// OrderBookUpdate represents the structure for order book WebSocket updates
type OrderBookUpdate struct {
	EventType     string     `json:"e"` // e.g., "depthUpdate"
	EventTime     int64      `json:"E"` // Event timestamp (ms)
	Symbol        string     `json:"s"` // e.g., "BTCUSDT"
	FirstUpdateID int64      `json:"U"` // First update ID in event
	FinalUpdateID int64      `json:"u"` // Final update ID in event
	Bids          [][]string `json:"b"` // Bid updates: [[price, qty], ...]
	Asks          [][]string `json:"a"` // Ask updates: [[price, qty], ...]
}

// TradeUpdate represents the structure for trade WebSocket updates
type TradeUpdate struct {
	EventType string `json:"e"` // e.g., "trade"
	EventTime int64  `json:"E"` // Event timestamp (ms)
	Symbol    string `json:"s"` // e.g., "BTCUSDT"
	TradeID   int64  `json:"t"` // Trade ID
	Price     string `json:"p"` // Trade price
	Quantity  string `json:"q"` // Trade quantity
	Buyer     bool   `json:"m"` // True if buyer is the market maker
}

// NewStreamBridge creates a new instance of StreamBridge
func NewStreamBridge(exchange ccxt.Binance) *StreamBridge {
	return &StreamBridge{
		exchange: exchange,
	}
}

// GetExchange returns the underlying CCXT exchange object for REST calls
func (s *StreamBridge) GetExchange() *ccxt.Binance {
	return &s.exchange
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
func (s *StreamBridge) WatchOrderBook(symbol string) (chan OrderBookUpdate, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	updates := make(chan OrderBookUpdate)

	go func() {
		defer close(updates)

		backoff := 5 * time.Second
		maxBackoff := 5 * time.Minute
		url := "wss://ws-fapi.binance.com/ws/" + symbol + "@depth"

		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := connectWebSocket(url)
				if err != nil {
					log.Printf("Failed to connect to WebSocket: %v", err)
					time.Sleep(backoff)
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					continue
				}

				// Reset backoff on successful connection
				backoff = 5 * time.Second

				// Start ping goroutine
				pingTicker := time.NewTicker(3 * time.Minute)
				go func() {
					defer pingTicker.Stop()
					for {
						select {
						case <-ctx.Done():
							return
						case <-pingTicker.C:
							if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
								log.Printf("Failed to send ping: %v", err)
								return
							}
						}
					}
				}()

				// Read messages
				for {
					var update OrderBookUpdate
					err := conn.ReadJSON(&update)
					if err != nil {
						log.Printf("Error reading from WebSocket: %v", err)
						conn.Close()
						break
					}

					select {
					case <-ctx.Done():
						conn.Close()
						return
					case updates <- update:
					}
				}
			}
		}
	}()

	return updates, cancel
}

// WatchTrades streams real-time trade updates for the given symbol
func (s *StreamBridge) WatchTrades(symbol string) (chan TradeUpdate, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	updates := make(chan TradeUpdate)

	go func() {
		defer close(updates)

		backoff := 5 * time.Second
		maxBackoff := 5 * time.Minute
		url := "wss://ws-fapi.binance.com/ws/" + symbol + "@trade"

		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := connectWebSocket(url)
				if err != nil {
					log.Printf("Failed to connect to WebSocket: %v", err)
					time.Sleep(backoff)
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					continue
				}

				// Reset backoff on successful connection
				backoff = 5 * time.Second

				// Start ping goroutine
				pingTicker := time.NewTicker(3 * time.Minute)
				go func() {
					defer pingTicker.Stop()
					for {
						select {
						case <-ctx.Done():
							return
						case <-pingTicker.C:
							if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
								log.Printf("Failed to send ping: %v", err)
								return
							}
						}
					}
				}()

				// Read messages
				for {
					var update TradeUpdate
					err := conn.ReadJSON(&update)
					if err != nil {
						log.Printf("Error reading from WebSocket: %v", err)
						conn.Close()
						break
					}

					select {
					case <-ctx.Done():
						conn.Close()
						return
					case updates <- update:
					}
				}
			}
		}
	}()

	return updates, cancel
}
