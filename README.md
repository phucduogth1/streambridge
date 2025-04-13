# StreamBridge

This package enhances the CCXT Go library by adding production-ready WebSocket support for Binance futures, focusing on public streams (order books and trades) while leveraging CCXT's existing REST API functionality.

## Features

- Real-time WebSocket streams for Binance Futures
  - Order book updates
  - Trade updates
- Automatic reconnection with exponential backoff
- Proper ping/pong handling to maintain connections
- Full access to CCXT's REST API functionality

## Installation

```bash
go get github.com/phucduogth1/streambridge
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "time"
    "github.com/ccxt/ccxt/go/v4"
    streambridge "github.com/phucduogth1/streambridge"
)

func main() {
    // Initialize CCXT Binance exchange
    exchange := ccxt.NewBinance(map[string]interface{}{
        "enableRateLimit": true,
    })

    // Create enhanced instance
    ws := streambridge.NewStreamBridge(exchange)
    
    // Stream order book
    orderBookChan, closer := ws.WatchOrderBook("BTCUSDT")
    go func() {
        for update := range orderBookChan {
            fmt.Printf("Order Book Update: Bids: %v, Asks: %v\n", update.Bids[0], update.Asks[0])
        }
    }()

    // Use REST API
    ticker, err := ws.GetExchange().FetchTicker("BTCUSDT")
    if err != nil {
        fmt.Printf("Error fetching ticker: %v\n", err)
        return
    }
    fmt.Printf("Current price: %v\n", *ticker.Last)

    // Stop after 10 seconds (example)
    time.Sleep(10 * time.Second)
    closer()
}
```

### Streaming Trades

```go
tradesChan, closer := ws.WatchTrades("BTCUSDT")
go func() {
    for trade := range tradesChan {
        fmt.Printf("Trade: Price: %s, Quantity: %s\n", trade.Price, trade.Quantity)
    }
}()

// Don't forget to close when done
defer closer()
```

## WebSocket Endpoints

- Order Book: `wss://ws-fapi.binance.com/ws/<symbol>@depth`
- Trades: `wss://ws-fapi.binance.com/ws/<symbol>@trade`

For testing, use the testnet endpoints:
- `wss://testnet.binancefuture.com/ws-fapi/v1`

## Features and Limitations

- Supports public data streams only (order books and trades)
- Automatic reconnection with exponential backoff (5s initial, max 5m)
- Ping/pong handling to maintain connection (3m ping interval)
- Connections valid for 24 hours (Binance limitation)
- Full access to CCXT's REST API functionality

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

## Error Handling

The package includes robust error handling:
- Automatic reconnection on connection loss
- Exponential backoff for reconnection attempts
- Proper cleanup of resources on stream closure
- Comprehensive logging of connection events

## License

MIT License# streambridge
