// filepath: /Users/duongphuc/Projects/go-workspace/streambridge/pkg/exchange/binance/futures/subscription.go
package futures

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/phucduogth1/streambridge/pkg/types"
)

// SubscriptionType represents the type of WebSocket subscription
type SubscriptionType string

const (
	// OrderBookStream for order book updates
	OrderBookStream SubscriptionType = "depth"

	// TradeStream for trade updates
	TradeStream SubscriptionType = "trade"

	// TickerStream for ticker updates
	TickerStream SubscriptionType = "ticker"
)

// SubscriptionManager manages WebSocket subscriptions for Binance futures
type SubscriptionManager struct {
	conn          *websocket.Conn
	isConnected   bool
	subscriptions map[string][]chan interface{}
	symbolStreams map[string]bool
	lastID        int
	mu            sync.RWMutex
	reconnectMu   sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	isTestnet     bool
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(isTestnet bool) *SubscriptionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &SubscriptionManager{
		subscriptions: make(map[string][]chan interface{}),
		symbolStreams: make(map[string]bool),
		lastID:        0,
		ctx:           ctx,
		cancel:        cancel,
		isTestnet:     isTestnet,
	}
}

// Connect establishes a WebSocket connection to Binance futures
func (sm *SubscriptionManager) Connect() error {
	sm.reconnectMu.Lock()
	defer sm.reconnectMu.Unlock()

	if sm.isConnected {
		return nil
	}

	var wsURL string
	if sm.isTestnet {
		wsURL = "wss://stream.binancefuture.com/ws"
	} else {
		wsURL = "wss://fstream.binance.com/ws"
	}

	log.Printf("Connecting to Binance Futures WebSocket: %s", wsURL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %v", err)
	}

	// Set initial read deadline
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Minute)); err != nil {
		conn.Close()
		return fmt.Errorf("failed to set read deadline: %v", err)
	}

	// Setup pong handler
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	})

	sm.conn = conn
	sm.isConnected = true

	// Start the message reader
	go sm.readMessages()

	// Start the ping/pong handler
	go sm.keepAlive()

	// Resubscribe to all existing streams
	if len(sm.symbolStreams) > 0 {
		streams := make([]string, 0, len(sm.symbolStreams))
		for stream := range sm.symbolStreams {
			streams = append(streams, stream)
		}

		if err := sm.sendSubscription(streams); err != nil {
			log.Printf("Failed to resubscribe to streams: %v", err)
		}
	}

	return nil
}

// Disconnect closes the WebSocket connection
func (sm *SubscriptionManager) Disconnect() {
	sm.reconnectMu.Lock()
	defer sm.reconnectMu.Unlock()

	if !sm.isConnected {
		return
	}

	if sm.conn != nil {
		sm.conn.Close()
	}

	sm.isConnected = false
	log.Printf("Disconnected from Binance Futures WebSocket")
}

// Close shuts down the subscription manager
func (sm *SubscriptionManager) Close() {
	sm.cancel()
	sm.Disconnect()
}

// Subscribe adds a new subscription for a symbol and stream type
func (sm *SubscriptionManager) Subscribe(symbol string, streamType SubscriptionType) (chan interface{}, func(), error) {
	stream := fmt.Sprintf("%s@%s", symbol, streamType)

	// Create the channel for this subscription
	ch := make(chan interface{}, 100)

	sm.mu.Lock()

	// Check if this is a new stream
	isNewStream := false
	if _, exists := sm.symbolStreams[stream]; !exists {
		sm.symbolStreams[stream] = true
		isNewStream = true
	}

	// Add the channel to the subscriptions
	if _, exists := sm.subscriptions[stream]; !exists {
		sm.subscriptions[stream] = []chan interface{}{}
	}
	sm.subscriptions[stream] = append(sm.subscriptions[stream], ch)

	sm.mu.Unlock()

	// Ensure we're connected
	if err := sm.Connect(); err != nil {
		return nil, nil, err
	}

	// If this is a new stream, send the subscription message
	if isNewStream {
		if err := sm.sendSubscription([]string{stream}); err != nil {
			return nil, nil, err
		}
	}

	// Return the channel and a cancel function
	cancel := func() {
		sm.mu.Lock()
		defer sm.mu.Unlock()

		// Remove this channel from the subscriptions
		channels := sm.subscriptions[stream]
		for i, c := range channels {
			if c == ch {
				// Remove the channel
				channels = append(channels[:i], channels[i+1:]...)
				break
			}
		}

		// Update or remove the stream
		if len(channels) == 0 {
			delete(sm.subscriptions, stream)
			delete(sm.symbolStreams, stream)

			// If connected, send unsubscribe message
			if sm.isConnected && sm.conn != nil {
				sm.sendUnsubscription([]string{stream})
			}

			// If no more subscriptions, disconnect
			if len(sm.subscriptions) == 0 {
				sm.Disconnect()
			}
		} else {
			sm.subscriptions[stream] = channels
		}

		close(ch)
	}

	return ch, cancel, nil
}

// sendSubscription sends a subscription message for the given streams
func (sm *SubscriptionManager) sendSubscription(streams []string) error {
	sm.lastID++
	id := sm.lastID

	subMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": streams,
		"id":     id,
	}

	if sm.conn != nil {
		return sm.conn.WriteJSON(subMsg)
	}
	return fmt.Errorf("not connected")
}

// sendUnsubscription sends an unsubscribe message for the given streams
func (sm *SubscriptionManager) sendUnsubscription(streams []string) error {
	sm.lastID++
	id := sm.lastID

	unsubMsg := map[string]interface{}{
		"method": "UNSUBSCRIBE",
		"params": streams,
		"id":     id,
	}

	if sm.conn != nil {
		return sm.conn.WriteJSON(unsubMsg)
	}
	return fmt.Errorf("not connected")
}

// readMessages continuously reads messages from the WebSocket connection
func (sm *SubscriptionManager) readMessages() {
	defer func() {
		sm.isConnected = false
		log.Printf("WebSocket reader routine exiting")
	}()

	for {
		select {
		case <-sm.ctx.Done():
			return
		default:
			if !sm.isConnected || sm.conn == nil {
				return
			}

			// Read message
			_, message, err := sm.conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading from WebSocket: %v", err)
				sm.reconnect()
				return
			}

			// Process the message
			go sm.processMessage(message)
		}
	}
}

// processMessage processes a raw WebSocket message
func (sm *SubscriptionManager) processMessage(message []byte) {
	// Check if it's a response to subscription/unsubscription
	var response struct {
		ID     int     `json:"id"`
		Result *string `json:"result"`
		Error  *int    `json:"error"`
	}

	if err := json.Unmarshal(message, &response); err == nil {
		if response.ID > 0 {
			// This is a response to subscription/unsubscription
			if response.Result != nil && *response.Result == "" {
				log.Printf("Subscription/unsubscription successful (ID: %d)", response.ID)
			} else if response.Error != nil {
				log.Printf("Subscription/unsubscription error (ID: %d): %v", response.ID, response.Error)
			}
			return
		}
	}

	// Parse as a stream message
	var streamMsg struct {
		Stream string          `json:"stream"`
		Data   json.RawMessage `json:"data"`
	}

	// If stream field exists, this is a combined stream message
	if err := json.Unmarshal(message, &streamMsg); err == nil && streamMsg.Stream != "" {
		sm.mu.RLock()
		channels, exists := sm.subscriptions[streamMsg.Stream]
		sm.mu.RUnlock()

		if exists {
			// Determine message type based on stream name
			var update interface{}

			if streamType, symbol, found := parseStream(streamMsg.Stream); found {
				switch streamType {
				case OrderBookStream:
					var bookUpdate binanceOrderBookUpdate
					if err := json.Unmarshal(streamMsg.Data, &bookUpdate); err == nil {
						update = types.OrderBookUpdate{
							Symbol:    symbol,
							Timestamp: bookUpdate.EventTime,
							Bids:      bookUpdate.Bids,
							Asks:      bookUpdate.Asks,
							Raw:       bookUpdate,
						}
					}

				case TradeStream:
					var tradeUpdate binanceTradeUpdate
					if err := json.Unmarshal(streamMsg.Data, &tradeUpdate); err == nil {
						side := "sell"
						if !tradeUpdate.Maker {
							side = "buy"
						}

						update = types.TradeUpdate{
							Symbol:    symbol,
							Timestamp: tradeUpdate.EventTime,
							ID:        tradeUpdate.TradeID,
							Price:     tradeUpdate.Price,
							Amount:    tradeUpdate.Quantity,
							Side:      side,
							Raw:       tradeUpdate,
						}
					}

				case TickerStream:
					var tickerUpdate binanceTickerUpdate
					if err := json.Unmarshal(streamMsg.Data, &tickerUpdate); err == nil {
						update = types.TickerUpdate{
							Symbol:        symbol,
							Timestamp:     tickerUpdate.EventTime,
							Last:          tickerUpdate.LastPrice,
							High:          tickerUpdate.HighPrice,
							Low:           tickerUpdate.LowPrice,
							Volume:        tickerUpdate.BaseVolume,
							QuoteVolume:   tickerUpdate.QuoteVolume,
							Change:        tickerUpdate.PriceChange,
							ChangePercent: tickerUpdate.PriceChangeP,
							Raw:           tickerUpdate,
						}
					}
				}
			}

			if update != nil {
				// Send the update to all subscribed channels
				for _, ch := range channels {
					select {
					case ch <- update:
					default:
						// Channel buffer is full, log and continue
						log.Printf("Channel buffer full for stream %s", streamMsg.Stream)
					}
				}
			}
		}

		return
	}

	// For direct data messages (not combined streams)
	// Try to determine the message type and stream

	// First check if it's an order book update
	var bookUpdate binanceOrderBookUpdate
	if err := json.Unmarshal(message, &bookUpdate); err == nil && bookUpdate.EventType == "depthUpdate" {
		stream := fmt.Sprintf("%s@depth", bookUpdate.Symbol)

		sm.mu.RLock()
		channels, exists := sm.subscriptions[stream]
		sm.mu.RUnlock()

		if exists {
			update := types.OrderBookUpdate{
				Symbol:    bookUpdate.Symbol,
				Timestamp: bookUpdate.EventTime,
				Bids:      bookUpdate.Bids,
				Asks:      bookUpdate.Asks,
				Raw:       bookUpdate,
			}

			for _, ch := range channels {
				select {
				case ch <- update:
				default:
					log.Printf("Channel buffer full for stream %s", stream)
				}
			}
		}

		return
	}

	// Check if it's a trade update
	var tradeUpdate binanceTradeUpdate
	if err := json.Unmarshal(message, &tradeUpdate); err == nil && tradeUpdate.EventType == "trade" {
		stream := fmt.Sprintf("%s@trade", tradeUpdate.Symbol)

		sm.mu.RLock()
		channels, exists := sm.subscriptions[stream]
		sm.mu.RUnlock()

		if exists {
			side := "sell"
			if !tradeUpdate.Maker {
				side = "buy"
			}

			update := types.TradeUpdate{
				Symbol:    tradeUpdate.Symbol,
				Timestamp: tradeUpdate.EventTime,
				ID:        tradeUpdate.TradeID,
				Price:     tradeUpdate.Price,
				Amount:    tradeUpdate.Quantity,
				Side:      side,
				Raw:       tradeUpdate,
			}

			for _, ch := range channels {
				select {
				case ch <- update:
				default:
					log.Printf("Channel buffer full for stream %s", stream)
				}
			}
		}

		return
	}

	// Check if it's a ticker update
	var tickerUpdate binanceTickerUpdate
	if err := json.Unmarshal(message, &tickerUpdate); err == nil && tickerUpdate.EventType == "24hrTicker" {
		stream := fmt.Sprintf("%s@ticker", tickerUpdate.Symbol)

		sm.mu.RLock()
		channels, exists := sm.subscriptions[stream]
		sm.mu.RUnlock()

		if exists {
			update := types.TickerUpdate{
				Symbol:        tickerUpdate.Symbol,
				Timestamp:     tickerUpdate.EventTime,
				Last:          tickerUpdate.LastPrice,
				High:          tickerUpdate.HighPrice,
				Low:           tickerUpdate.LowPrice,
				Volume:        tickerUpdate.BaseVolume,
				QuoteVolume:   tickerUpdate.QuoteVolume,
				Change:        tickerUpdate.PriceChange,
				ChangePercent: tickerUpdate.PriceChangeP,
				Raw:           tickerUpdate,
			}

			for _, ch := range channels {
				select {
				case ch <- update:
				default:
					log.Printf("Channel buffer full for stream %s", stream)
				}
			}
		}

		return
	}

	// Log unhandled messages
	log.Printf("Unhandled message: %s", string(message))
}

// keepAlive sends periodic pings to keep the connection alive
func (sm *SubscriptionManager) keepAlive() {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			if sm.isConnected && sm.conn != nil {
				if err := sm.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Failed to send ping: %v", err)
					sm.reconnect()
					return
				}
			}
		}
	}
}

// reconnect attempts to reconnect the WebSocket
func (sm *SubscriptionManager) reconnect() {
	sm.Disconnect()

	// Use exponential backoff for reconnection attempts
	backoff := 1 * time.Second
	maxBackoff := 5 * time.Minute

	for i := 0; i < 10; i++ { // Try up to 10 times
		select {
		case <-sm.ctx.Done():
			return
		case <-time.After(backoff):
			// Try to reconnect
			log.Printf("Attempting to reconnect (try %d, backoff %v)...", i+1, backoff)

			if err := sm.Connect(); err != nil {
				log.Printf("Reconnection attempt failed: %v", err)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			} else {
				log.Printf("Successfully reconnected to WebSocket")
				return
			}
		}
	}

	log.Printf("Failed to reconnect after multiple attempts")
}

// parseStream parses a stream name into its type and symbol
func parseStream(stream string) (streamType SubscriptionType, symbol string, ok bool) {
	// Check common formats: <symbol>@<streamType>
	for i := len(stream) - 1; i >= 0; i-- {
		if stream[i] == '@' {
			symbol = stream[:i]
			streamType = SubscriptionType(stream[i+1:])
			return streamType, symbol, true
		}
	}
	return "", "", false
}
