# StreamBridge Project Instructions

## Overview

StreamBridge is a Go package that extends the CCXT Go library (`github.com/ccxt/ccxt/go`) to provide professional, production-ready WebSocket streaming capabilities for Binance's USDM futures. It is designed to facilitate real-time data handling for quantitative trading applications, supporting both market data and user data streams with high reliability, efficiency, and ease of use.

## Goals

- Establish and manage WebSocket connections to Binance's USDM futures API.
- Subscribe to market data streams (e.g., tickers, trades, order books).
- Handle user data streams for account and order updates.
- Parse incoming messages into CCXT's unified data structures.
- Provide a simple API for registering callbacks to process stream data.
- Ensure robust connection management with reconnection logic.
- Support concurrent handling of multiple streams.
- Include comprehensive logging and error handling.

## Project Structure

The project will be a separate Go module that depends on CCXT Go:

- `streambridge/`
  - `client.go`: Defines the main `StreamBridge` client and its methods.
  - `marketdata.go`: Handles market data streams.
  - `userdata.go`: Handles user data streams.
  - `types.go`: Defines custom types and structs.
  - `utils.go`: Utility functions (e.g., message parsing).

## Prerequisites

- Go 1.18 or higher.
- CCXT Go library (`go get github.com/ccxt/ccxt/go`).
- WebSocket library (`go get github.com/gorilla/websocket`).
- Logging library (`go get github.com/sirupsen/logrus`).

## Development Phases

### Phase 1: Setup and Basic Structure

#### Tasks

1. **Initialize the Project:**

   - Create a new directory `streambridge` and initialize a Go module:

     ```bash
     mkdir streambridge
     cd streambridge
     go mod init github.com/yourusername/streambridge
     ```
   - Add dependencies:

     ```bash
     go get github.com/ccxt/ccxt/go
     go get github.com/gorilla/websocket
     go get github.com/sirupsen/logrus
     ```

2. **Define the** `StreamBridge` **Struct:**

   - In `client.go`, create the `StreamBridge` struct to hold the CCXT exchange instance, WebSocket connections, and subscription data:

     ```go
     package streambridge
     
     import (
         "github.com/ccxt/ccxt/go"
         "github.com/gorilla/websocket"
         "github.com/sirupsen/logrus"
     )
     
     type StreamBridge struct {
         exchange    *ccxt.Binanceusdm
         logger      *logrus.Logger
         marketConn  *websocket.Conn
         userConn    *websocket.Conn
         marketSubs  map[string]func(interface{})
         userSubs    map[string]func(interface{})
         stopChan    chan struct{}
     }
     ```

3. **Implement the Constructor:**

   - Add `NewStreamBridge` to initialize the client:

     ```go
     func NewStreamBridge(exchange *ccxt.Binanceusdm) (*StreamBridge, error) {
         logger := logrus.New()
         logger.SetLevel(logrus.InfoLevel)
     
         return &StreamBridge{
             exchange:   exchange,
             logger:     logger,
             marketSubs: make(map[string]func(interface{})),
             userSubs:   make(map[string]func(interface{})),
             stopChan:   make(chan struct{}),
         }, nil
     }
     ```

4. **Create Placeholder Files:**

   - Create empty files: `marketdata.go`, `userdata.go`, `types.go`, `utils.go`.

#### Deliverables

- A functional Go module with a basic `StreamBridge` struct and constructor.

---

### Phase 2: Market Data Streams

#### Tasks

1. **WebSocket Client for Market Data:**

   - In `marketdata.go`, implement a method to connect to Binance's market data WebSocket:

     ```go
     package streambridge
     
     import (
         "github.com/gorilla/websocket"
     )
     
     func (sb *StreamBridge) connectMarketData() error {
         url := "wss://fstream.binance.com/ws"
         conn, _, err := websocket.DefaultDialer.Dial(url, nil)
         if err != nil {
             sb.logger.Errorf("Failed to connect to market data WebSocket: %v", err)
             return err
         }
         sb.marketConn = conn
         go sb.readMarketMessages()
         return nil
     }
     ```

2. **Subscription Method:**

   - Add `SubscribeMarketData` to subscribe to streams:

     ```go
     func (sb *StreamBridge) SubscribeMarketData(streamType, symbol string, callback func(interface{})) error {
         if sb.marketConn == nil {
             if err := sb.connectMarketData(); err != nil {
                 return err
             }
         }
     
         stream := symbol + "@" + streamType
         sb.marketSubs[stream] = callback
     
         subMsg := map[string]interface{}{
             "method": "SUBSCRIBE",
             "params": []string{stream},
             "id":     1,
         }
         return sb.marketConn.WriteJSON(subMsg)
     }
     ```

3. **Message Handling:**

   - Implement `readMarketMessages` to process incoming messages:

     ```go
     func (sb *StreamBridge) readMarketMessages() {
         defer sb.marketConn.Close()
         for {
             select {
             case <-sb.stopChan:
                 return
             default:
                 var msg map[string]interface{}
                 if err := sb.marketConn.ReadJSON(&msg); err != nil {
                     sb.logger.Errorf("Market data read error: %v", err)
                     return
                 }
                 sb.handleMarketMessage(msg)
             }
         }
     }
     
     func (sb *StreamBridge) handleMarketMessage(msg map[string]interface{}) {
         stream, ok := msg["stream"].(string)
         if !ok {
             return
         }
         if callback, exists := sb.marketSubs[stream]; exists {
             data := msg["data"]
             switch stream[strings.Index(stream, "@")+1:] {
             case "ticker":
                 ticker := sb.exchange.ParseTicker(data)
                 callback(ticker)
             case "aggTrade":
                 trade := sb.exchange.ParseTrade(data)
                 callback(trade)
             }
         }
     }
     ```

#### Deliverables

- Ability to connect to Binance market data WebSocket and subscribe to streams like `<symbol>@ticker`.

---

### Phase 3: User Data Streams

#### Tasks

1. **Listen Key Management:**

   - In `userdata.go`, implement listen key handling:

     ```go
     package streambridge
     
     func (sb *StreamBridge) getListenKey() (string, error) {
         result, err := sb.exchange.FapiPrivatePostListenKey(nil)
         if err != nil {
             return "", err
         }
         return result.(map[string]interface{})["listenKey"].(string), nil
     }
     
     func (sb *StreamBridge) keepAliveListenKey() {
         ticker := time.NewTicker(30 * time.Minute)
         defer ticker.Stop()
         for {
             select {
             case <-ticker.C:
                 sb.exchange.FapiPrivatePutListenKey(nil)
             case <-sb.stopChan:
                 return
             }
         }
     }
     ```

2. **WebSocket Client for User Data:**

   - Add connection logic:

     ```go
     func (sb *StreamBridge) connectUserData() error {
         listenKey, err := sb.getListenKey()
         if err != nil {
             return err
         }
         url := "wss://fstream.binance.com/stream?streams=" + listenKey
         conn, _, err := websocket.DefaultDialer.Dial(url, nil)
         if err != nil {
             sb.logger.Errorf("Failed to connect to user data WebSocket: %v", err)
             return err
         }
         sb.userConn = conn
         go sb.readUserMessages()
         go sb.keepAliveListenKey()
         return nil
     }
     ```

3. **Subscription Method:**

   - Add `SubscribeUserData`:

     ```go
     func (sb *StreamBridge) SubscribeUserData(eventType string, callback func(interface{})) error {
         if sb.userConn == nil {
             if err := sb.connectUserData(); err != nil {
                 return err
             }
         }
         sb.userSubs[eventType] = callback
         return nil
     }
     ```

4. **Message Handling:**

   - Implement `readUserMessages`:

     ```go
     func (sb *StreamBridge) readUserMessages() {
         defer sb.userConn.Close()
         for {
             select {
             case <-sb.stopChan:
                 return
             default:
                 var msg map[string]interface{}
                 if err := sb.userConn.ReadJSON(&msg); err != nil {
                     sb.logger.Errorf("User data read error: %v", err)
                     return
                 }
                 data := msg["data"].(map[string]interface{})
                 eventType := data["e"].(string)
                 if callback, exists := sb.userSubs[eventType]; exists {
                     switch eventType {
                     case "executionReport":
                         order := sb.exchange.ParseOrder(data)
                         callback(order)
                     case "outboundAccountPosition":
                         balance := sb.exchange.ParseBalance(data)
                         callback(balance)
                     }
                 }
             }
         }
     }
     ```

#### Deliverables

- Functional user data stream handling with listen key management.

---

### Phase 4: Connection Management and Error Handling

#### Tasks

1. **Reconnection Logic:**

   - Update `connectMarketData` and `connectUserData` with retries:

     ```go
     func (sb *StreamBridge) connectMarketData() error {
         for i := 0; i < 5; i++ {
             url := "wss://fstream.binance.com/ws"
             conn, _, err := websocket.DefaultDialer.Dial(url, nil)
             if err == nil {
                 sb.marketConn = conn
                 go sb.readMarketMessages()
                 return nil
             }
             sb.logger.Warnf("Retry %d for market data connection: %v", i+1, err)
             time.Sleep(time.Duration(i+1) * time.Second)
         }
         return errors.New("max retries reached for market data connection")
     }
     ```

2. **Subscription Restoration:**

   - Add a method to resubscribe after reconnection:

     ```go
     func (sb *StreamBridge) resubscribeMarketData() {
         for stream := range sb.marketSubs {
             subMsg := map[string]interface{}{
                 "method": "SUBSCRIBE",
                 "params": []string{stream},
                 "id":     1,
             }
             sb.marketConn.WriteJSON(subMsg)
         }
     }
     ```

3. **Stop Method:**

   - Add `Stop` to cleanly shut down:

     ```go
     func (sb *StreamBridge) Stop() {
         close(sb.stopChan)
         if sb.marketConn != nil {
             sb.marketConn.Close()
         }
         if sb.userConn != nil {
             sb.userConn.Close()
         }
     }
     ```

#### Deliverables

- Robust reconnection and shutdown capabilities.

---

### Phase 5: Concurrency and Performance

#### Tasks

1. **Channel-Based Message Handling:**

   - Modify `readMarketMessages` to use channels:

     ```go
     func (sb *StreamBridge) readMarketMessages() {
         msgChan := make(chan map[string]interface{}, 100)
         defer sb.marketConn.Close()
         go func() {
             for msg := range msgChan {
                 sb.handleMarketMessage(msg)
             }
         }()
         for {
             select {
             case <-sb.stopChan:
                 close(msgChan)
                 return
             default:
                 var msg map[string]interface{}
                 if err := sb.marketConn.ReadJSON(&msg); err != nil {
                     sb.logger.Errorf("Market data read error: %v", err)
                     return
                 }
                 msgChan <- msg
             }
         }
     }
     ```

2. **Test Concurrency:**

   - Ensure handling of multiple streams without blocking.

#### Deliverables

- Efficient, non-blocking message processing.

---

### Phase 6: Testing and Documentation

#### Tasks

1. **Unit Tests:**

   - In `client_test.go`, test subscription and callback:

     ```go
     func TestSubscribeMarketData(t *testing.T) {
         exchange := ccxt.NewBinanceusdm(nil)
         sb, _ := NewStreamBridge(exchange)
         called := false
         sb.SubscribeMarketData("ticker", "BTCUSDT", func(data interface{}) {
             called = true
         })
         // Simulate message handling here
     }
     ```

2. **Documentation:**

   - Add godoc comments to all public functions.

#### Deliverables

- Comprehensive tests and documentation.

---

## Binance USDM Futures WebSocket Details

### Base URLs

- Market Data: `wss://fstream.binance.com/ws`
- User Data: `wss://fstream.binance.com/stream?streams=<listenKey>`
- Testnet: `wss://stream.binancefuture.com`

### Market Data Streams

- **Subscription Message:**

  ```json
  {
    "method": "SUBSCRIBE",
    "params": ["btcusdt@ticker"],
    "id": 1
  }
  ```

- **Example Streams:**

  - `<symbol>@aggTrade`: Aggregate trades.
  - `<symbol>@ticker`: Ticker updates.
  - `<symbol>@depth@100ms`: Order book depth.

- **Sample Message (ticker):**

  ```json
  {
    "stream": "btcusdt@ticker",
    "data": {
      "e": "24hrTicker",
      "E": 123456789,
      "s": "BTCUSDT",
      "c": "0.0025",
      "h": "0.0025",
      "l": "0.0010",
      "v": "1000"
    }
  }
  ```

### User Data Streams

- **Listen Key Endpoint:** `POST /fapi/v1/listenKey`
- **Keep-Alive:** `PUT /fapi/v1/listenKey` every 30 minutes.
- **Sample Message (executionReport):**

  ```json
  {
    "stream": "<listenKey>",
    "data": {
      "e": "executionReport",
      "s": "BTCUSDT",
      "c": "order123",
      "S": "BUY",
      "o": "LIMIT",
      "q": "1.0",
      "p": "9000",
      "X": "FILLED",
      "T": 1591702613943
    }
  }
  ```

## Notes

- Use CCXT's `ParseTicker`, `ParseTrade`, `ParseOrder`, etc., for message parsing.
- Handle timestamps in milliseconds.
- Respect Binance rate limits to avoid `429` errors.