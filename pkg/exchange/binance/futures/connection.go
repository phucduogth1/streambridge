package futures

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/phucduogth1/streambridge/pkg/exchange/binance"
)

const (
	// maxBackoff is the maximum time to wait between reconnection attempts
	maxBackoff = 5 * time.Minute

	// initialBackoff is the initial time to wait between reconnection attempts
	initialBackoff = 5 * time.Second

	// pingInterval is the time between ping messages
	pingInterval = 3 * time.Minute

	// readDeadline is the time to wait for a response before timing out
	readDeadline = 10 * time.Minute

	// Futures-specific WebSocket URLs
	futuresUserDataBaseURL    = "wss://fstream.binance.com/ws/"
	futuresUserDataTestnetURL = "wss://stream.binancefuture.com/ws/"
)

// UserDataConnection manages the WebSocket connection for futures user data streams
type UserDataConnection struct {
	exchange     *binance.BinanceExchange
	listenKeyMgr *ListenKeyManager
	conn         *websocket.Conn
	isConnected  bool
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc

	// Channels for different event types
	orderChan    chan json.RawMessage
	balanceChan  chan json.RawMessage
	positionChan chan json.RawMessage
	accountChan  chan json.RawMessage
}

// NewUserDataConnection creates a new user data connection for futures
func NewUserDataConnection(exchange *binance.BinanceExchange) *UserDataConnection {
	ctx, cancel := context.WithCancel(context.Background())

	return &UserDataConnection{
		exchange:     exchange,
		listenKeyMgr: NewListenKeyManager(exchange),
		ctx:          ctx,
		cancel:       cancel,
		orderChan:    make(chan json.RawMessage, 100),
		balanceChan:  make(chan json.RawMessage, 100),
		positionChan: make(chan json.RawMessage, 100),
		accountChan:  make(chan json.RawMessage, 100),
	}
}

// Connect establishes a WebSocket connection to the futures user data stream
func (udc *UserDataConnection) Connect(ctx context.Context) error {
	udc.mu.Lock()
	defer udc.mu.Unlock()

	if udc.isConnected {
		return nil // Already connected
	}

	// Get a listen key for futures
	listenKey, err := udc.listenKeyMgr.GetListenKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to get futures listen key: %v", err)
	}

	// Start renewing the listen key
	if err := udc.listenKeyMgr.StartRenewing(ctx); err != nil {
		return fmt.Errorf("failed to start renewing futures listen key: %v", err)
	}

	// Determine the WebSocket URL based on testnet setting
	var wsURL string
	if udc.exchange.IsTestnet() {
		wsURL = futuresUserDataTestnetURL + listenKey
	} else {
		wsURL = futuresUserDataBaseURL + listenKey
	}

	log.Printf("Connecting to futures user data stream at %s", wsURL)

	// Establish the WebSocket connection
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to futures WebSocket: %v", err)
	}

	// Set read deadline for initial connection
	err = conn.SetReadDeadline(time.Now().Add(readDeadline))
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to set read deadline: %v", err)
	}

	// Setup pong handler to reset deadline
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(readDeadline))
	})

	udc.conn = conn
	udc.isConnected = true

	// Start reading messages
	go udc.readMessages(ctx)
	// Start sending pings
	go udc.keepAlive(ctx)

	return nil
}

// readMessages continuously reads messages from the WebSocket connection
func (udc *UserDataConnection) readMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !udc.isConnected {
				return
			}

			_, message, err := udc.conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading from futures user data WebSocket: %v", err)
				udc.reconnect(ctx)
				return
			}

			// Process the message
			go udc.processMessage(message)
		}
	}
}

// processMessage parses the raw WebSocket message and routes it to the appropriate channel
func (udc *UserDataConnection) processMessage(data []byte) {
	// Parse the event type from the message
	var baseEvent struct {
		EventType string `json:"e"`
	}

	if err := json.Unmarshal(data, &baseEvent); err != nil {
		log.Printf("Error parsing futures user data message: %v", err)
		return
	}

	// Route the message based on event type
	switch baseEvent.EventType {
	case "ORDER_TRADE_UPDATE":
		// Order update in futures
		udc.orderChan <- data
	case "ACCOUNT_UPDATE":
		// Account update in futures (includes balance and position updates)
		udc.balanceChan <- data

		// Extract position data if any
		var accountUpdate struct {
			EventType string `json:"e"`
			Data      struct {
				Positions []json.RawMessage `json:"P"`
			} `json:"a"`
		}

		if err := json.Unmarshal(data, &accountUpdate); err != nil {
			return
		}

		// If there are position updates, send them to the position channel
		if len(accountUpdate.Data.Positions) > 0 {
			for _, posData := range accountUpdate.Data.Positions {
				udc.positionChan <- posData
			}
		}
	case "MARGIN_CALL", "ACCOUNT_CONFIG_UPDATE":
		// Account configuration or margin call in futures
		udc.accountChan <- data
	default:
		log.Printf("Unknown futures event type: %s", baseEvent.EventType)
	}
}

// keepAlive sends periodic pings to keep the connection alive
func (udc *UserDataConnection) keepAlive(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			udc.mu.RLock()
			if udc.isConnected && udc.conn != nil {
				if err := udc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Failed to send ping to futures WebSocket: %v", err)
					udc.mu.RUnlock()
					udc.reconnect(ctx)
					return
				}
			}
			udc.mu.RUnlock()
		}
	}
}

// reconnect attempts to reconnect to the WebSocket with exponential backoff
func (udc *UserDataConnection) reconnect(ctx context.Context) {
	udc.mu.Lock()
	if udc.conn != nil {
		udc.conn.Close()
		udc.conn = nil
	}
	udc.isConnected = false
	udc.mu.Unlock()

	// Use exponential backoff for reconnection attempts
	backoff := initialBackoff

	for i := 0; i < 10; i++ { // Try up to 10 times
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			log.Printf("Attempting to reconnect to futures user data stream (try %d, backoff %v)...", i+1, backoff)

			err := udc.Connect(ctx)
			if err == nil {
				log.Printf("Successfully reconnected to futures user data stream")
				return
			}

			log.Printf("Reconnection attempt failed: %v", err)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	log.Printf("Failed to reconnect to futures user data stream after multiple attempts")
}

// Disconnect closes the WebSocket connection and stops listen key renewal
func (udc *UserDataConnection) Disconnect() {
	udc.mu.Lock()
	defer udc.mu.Unlock()

	if udc.conn != nil {
		udc.conn.Close()
		udc.conn = nil
	}

	udc.isConnected = false
	udc.listenKeyMgr.StopRenewing()
}

// Close completely shuts down the user data connection
func (udc *UserDataConnection) Close() {
	udc.Disconnect()
	udc.cancel() // Cancel the context

	// Close all message channels
	close(udc.orderChan)
	close(udc.balanceChan)
	close(udc.positionChan)
	close(udc.accountChan)
}

// GetOrderChannel returns the channel for order updates
func (udc *UserDataConnection) GetOrderChannel() <-chan json.RawMessage {
	return udc.orderChan
}

// GetBalanceChannel returns the channel for balance updates
func (udc *UserDataConnection) GetBalanceChannel() <-chan json.RawMessage {
	return udc.balanceChan
}

// GetPositionChannel returns the channel for position updates
func (udc *UserDataConnection) GetPositionChannel() <-chan json.RawMessage {
	return udc.positionChan
}

// GetAccountChannel returns the channel for account updates
func (udc *UserDataConnection) GetAccountChannel() <-chan json.RawMessage {
	return udc.accountChan
}
