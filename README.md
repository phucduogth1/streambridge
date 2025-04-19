# StreamBridge

StreamBridge enhances the CCXT Go library by adding production-ready WebSocket support for Binance markets, with a focus on both public market data streams and user data streams. It leverages CCXT's existing REST API functionality while providing real-time streaming capabilities.

## Features

- Real-time WebSocket streams for multiple Binance market types:
  - Futures (USDM)
  - Spot
- Market data streams:
  - Order book updates
  - Trade updates
  - Ticker updates
- User data streams for futures:
  - Order updates
  - Balance updates
  - Position updates (futures-specific)
- Connection management:
  - Automatic reconnection with exponential backoff
  - Proper ping/pong handling to maintain connections
  - Listen key management for user data streams
- Full access to CCXT's REST API functionality

## Installation

```bash
go get github.com/phucduogth1/streambridge
```

## Usage

### Basic Initialization

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    ccxt "github.com/ccxt/ccxt/go/v4"
    "github.com/phucduogth1/streambridge/pkg/streambridge"
    "github.com/phucduogth1/streambridge/pkg/types"
)

func main() {
    // Initialize CCXT Binance exchange
    exchange := ccxt.NewBinance(map[string]interface{}{
        "enableRateLimit": true,
        // For testnet:
        // "test": true,
        
        // For authentication (required for user data):
        // "apiKey": "YOUR_API_KEY",
        // "secret": "YOUR_SECRET_KEY",
    })

    // Create StreamBridge instance
    sb, err := streambridge.NewStreamBridge(exchange)
    if err != nil {
        panic(err)
    }
    
    // Use StreamBridge...
}
```

### Market Data Streams

#### Streaming Order Book

```go
// Get futures provider
futuresProvider, err := sb.GetStreamProvider(types.Futures)
if err != nil {
    panic(err)
}

// Create context that can be cancelled
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Stream order book for BTCUSDT
orderBookChan, closeFunc, err := futuresProvider.WatchOrderBook(ctx, "BTCUSDT", nil)
if err != nil {
    panic(err)
}
defer closeFunc()

// Process order book updates
for update := range orderBookChan {
    fmt.Printf("Order Book Update: Bids: %v, Asks: %v\n", update.Bids[0], update.Asks[0])
}
```

#### Streaming Trades

```go
// Stream trades for BTCUSDT
tradesChan, closeFunc, err := futuresProvider.WatchTrades(ctx, "BTCUSDT", nil)
if err != nil {
    panic(err)
}
defer closeFunc()

// Process trade updates
for trade := range tradesChan {
    fmt.Printf("Trade: Price: %s, Amount: %s, Side: %s\n", trade.Price, trade.Amount, trade.Side)
}
```

#### Streaming Ticker

```go
// Stream ticker for BTCUSDT
tickerChan, closeFunc, err := futuresProvider.WatchTicker(ctx, "BTCUSDT", nil)
if err != nil {
    panic(err)
}
defer closeFunc()

// Process ticker updates
for ticker := range tickerChan {
    fmt.Printf("Ticker: Last: %s, High: %s, Low: %s, Volume: %s\n", 
        ticker.Last, ticker.High, ticker.Low, ticker.Volume)
}
```

### User Data Streams

#### Streaming Order Updates (Futures)

```go
// Get user data provider
userDataProvider, err := sb.GetUserDataProvider()
if err != nil {
    panic(err)
}

// Create context that can be cancelled
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Stream order updates
orderChan, closeFunc, err := userDataProvider.WatchOrders(ctx, nil)
if err != nil {
    panic(err)
}
defer closeFunc()

// Process order updates
for orderUpdate := range orderChan {
    fmt.Printf("Order Update: Symbol: %s, OrderID: %d, Status: %s\n", 
        orderUpdate.Symbol, orderUpdate.OrderID, orderUpdate.Status)
}
```

#### Streaming Balance Updates (Futures)

```go
// Stream balance updates
balanceChan, closeFunc, err := userDataProvider.WatchBalances(ctx, nil)
if err != nil {
    panic(err)
}
defer closeFunc()

// Process balance updates
for balanceUpdate := range balanceChan {
    fmt.Printf("Balance Update: Timestamp: %d\n", balanceUpdate.Timestamp)
    for asset, balance := range balanceUpdate.Balances {
        fmt.Printf("  %s: Free: %s, Locked: %s, Total: %s\n", 
            asset, balance.Free, balance.Locked, balance.Total)
    }
}
```

#### Streaming Position Updates (Futures-specific)

```go
// Check if we have a FuturesUserDataProvider that supports position updates
futuresUserData, ok := userDataProvider.(types.FuturesUserDataProvider)
if !ok {
    panic("User data provider does not support futures position updates")
}

// Stream position updates
positionChan, closeFunc, err := futuresUserData.WatchPositions(ctx, nil)
if err != nil {
    panic(err)
}
defer closeFunc()

// Process position updates
for positionUpdate := range positionChan {
    fmt.Printf("Position Update: Symbol: %s, Side: %s, Amount: %s, Entry: %s\n", 
        positionUpdate.Symbol, positionUpdate.Side, 
        positionUpdate.PositionAmt, positionUpdate.EntryPrice)
}
```

### Accessing Multiple Market Types

```go
// Get spot provider
spotProvider, err := sb.GetStreamProvider(types.Spot)
if err != nil {
    panic(err)
}

// Stream spot market order book
spotOrderBookChan, closeSpotFunc, err := spotProvider.WatchOrderBook(ctx, "BTCUSDT", nil)
if err != nil {
    panic(err)
}
defer closeSpotFunc()

// Process spot order book updates
go func() {
    for update := range spotOrderBookChan {
        fmt.Printf("Spot Order Book: Bids: %v, Asks: %v\n", update.Bids[0], update.Asks[0])
    }
}()

// Get futures provider
futuresProvider, err := sb.GetStreamProvider(types.Futures)
if err != nil {
    panic(err)
}

// Stream futures market order book
futuresOrderBookChan, closeFuturesFunc, err := futuresProvider.WatchOrderBook(ctx, "BTCUSDT", nil)
if err != nil {
    panic(err)
}
defer closeFuturesFunc()

// Process futures order book updates
for update := range futuresOrderBookChan {
    fmt.Printf("Futures Order Book: Bids: %v, Asks: %v\n", update.Bids[0], update.Asks[0])
}
```

## Using CCXT REST API Functions

StreamBridge maintains full access to CCXT's REST API functionality:

```go
// Get the underlying CCXT exchange
ccxtExchange := sb.GetCcxtExchange()

// Use CCXT functions directly
ticker, err := ccxtExchange.FetchTicker("BTCUSDT")
if err != nil {
    panic(err)
}
fmt.Printf("Current price: %v\n", *ticker.Last)
```

## WebSocket Endpoints

### Futures Market Data
- Order Book: `wss://fstream.binance.com/ws/<symbol>@depth`
- Trades: `wss://fstream.binance.com/ws/<symbol>@trade`
- Ticker: `wss://fstream.binance.com/ws/<symbol>@ticker`

### Futures User Data
- User Data Stream: `wss://fstream.binance.com/ws/<listenKey>`

### Spot Market Data
- Order Book: `wss://stream.binance.com:9443/ws/<symbol>@depth`
- Trades: `wss://stream.binance.com:9443/ws/<symbol>@trade`
- Ticker: `wss://stream.binance.com:9443/ws/<symbol>@ticker`

For testing, use the testnet endpoints (when `test: true` is set):
- Futures: `wss://stream.binancefuture.com/ws/`
- Spot: `wss://testnet.binance.vision/ws/`

## Connection Management

StreamBridge handles WebSocket connections with robust error handling:

- Automatic reconnection on connection loss
- Exponential backoff for reconnection attempts (5s initial, max 5m)
- Ping/pong handling to maintain connection (3m ping interval)
- Listen key management for user data streams (auto-renewal)
- Proper cleanup of resources on stream closure

## Examples

Full working examples can be found in the examples directory:

- Market Data: `examples/market_data/market_data_example.go`
- Multi-Market Data: `examples/multi_market/multi_market_example.go`
- User Data: `examples/user_data/user_data_example.go`

## Testing

Run the tests:

```bash
go test ./...
```

For development/testing, use the testnet by setting the test option:

```go
exchange := ccxt.NewBinance(map[string]interface{}{
    "enableRateLimit": true,
    "test": true,
})
```

## License

MIT License
