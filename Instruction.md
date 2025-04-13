### Instruction for LLM Coding Agent: Developing a Production-Ready Go Package for CCXT with Enhanced WebSocket Support for Binance Futures

As an instructor, I’m providing you, the LLM coding agent, with a detailed, step-by-step guide to develop a Go project that serves as a utility package. This package will enhance the open-source CCXT Go library by adding production-ready WebSocket support for Binance futures, while retaining its robust REST API capabilities. The goal is to create a reusable package that can be imported into a main project, offering a seamless interface for both REST and real-time WebSocket streaming. This instruction assumes you have no prior knowledge of CCXT or Binance’s latest documentation, so I’ll provide all necessary details, including domains, endpoints, data structures, and requirements, without writing the actual code (which you’ll handle).

#### Project Overview
- **Package Name**: `streambridge`
- **Purpose**: Enhance CCXT Go (`github.com/ccxt/ccxt/go`) with production-ready WebSocket support for Binance futures, focusing on public streams (order books and trades) while leveraging CCXT’s existing REST API for futures.
- **Scope**: For now, focus solely on Binance futures WebSocket streams (public data only), with REST API functionality inherited from CCXT Go without modification.
- **Outcome**: A Go package that can be imported (e.g., `github.com/yourusername/streambridge`) and used in a main project with simple function calls for both REST and WebSocket operations.

#### Step-by-Step Instructions

##### 1. Project Setup
- **Initialize the Go Module**:
  - Create a new directory named `streambridge`.
  - Inside the directory, run: `go mod init github.com/yourusername/streambridge` (replace `yourusername` with your actual GitHub username or preferred module path).
  - This sets up the project as a Go module.

- **Install Dependencies**:
  - Add CCXT Go: `go get github.com/ccxt/ccxt/go/v4@latest` (use the latest version available as of April 2025; this is a stable reference point).
  - Add Gorilla WebSocket for WebSocket handling: `go get github.com/gorilla/websocket@v1.5.0`.
  - These dependencies will be used for REST (CCXT) and WebSocket (Gorilla) functionality.

- **Directory Structure**:
  - Create a single file, `streambridge.go`, in the root directory for the main package logic.
  - Optionally, create a `README.md` file for documentation (instructions provided later).

##### 2. Define the Core Struct and Data Structures
- **Main Struct**: `streambridge`
  - Purpose: Acts as the central object for interacting with Binance futures via REST and WebSocket.
  - Fields:
    - `exchange *ccxt.Binance`: A pointer to a CCXT Binance exchange instance, initialized with user-provided API credentials (optional for public streams).
  - Constructor: Create a function `Newstreambridge(exchange *ccxt.Binance) *streambridge` to initialize this struct.

- **WebSocket Data Structures** (based on Binance Futures WebSocket API):
  - **OrderBookUpdate** (for order book streams):
    ```go
    type OrderBookUpdate struct {
        EventType     string     `json:"e"` // e.g., "depthUpdate"
        EventTime     int64      `json:"E"` // Event timestamp (ms)
        Symbol        string     `json:"s"` // e.g., "BTCUSDT"
        FirstUpdateID int64      `json:"U"` // First update ID in event
        FinalUpdateID int64      `json:"u"` // Final update ID in event
        Bids          [][]string `json:"b"` // Bid updates: [[price, qty], ...]
        Asks          [][]string `json:"a"` // Ask updates: [[price, qty], ...]
    }
    ```
    - Notes: Prices and quantities are strings per Binance’s API to preserve precision (convert to float64 as needed in your logic).
  - **TradeUpdate** (for trade streams):
    ```go
    type TradeUpdate struct {
        EventType string `json:"e"` // e.g., "trade"
        EventTime int64  `json:"E"` // Event timestamp (ms)
        Symbol    string `json:"s"` // e.g., "BTCUSDT"
        TradeID   int64  `json:"t"` // Trade ID
        Price     string `json:"p"` // Trade price
        Quantity  string `json:"q"` // Trade quantity
        Buyer     bool   `json:"m"` // True if buyer is the market maker
    }
    ```

- **Why These Structs?**: These match Binance’s WebSocket response format for futures streams (e.g., `@depth`, `@trade`), as detailed in the Binance Futures WebSocket API documentation.

##### 3. Implement WebSocket Connection Logic
- **Binance Futures WebSocket Endpoint**:
  - Base URL: `wss://ws-fapi.binance.com/ws-fapi/v1`
  - Testnet URL (for testing): `wss://testnet.binancefuture.com/ws-fapi/v1`
  - Stream Format: `<symbol>@<streamType>` (e.g., `btcusdt@depth` for order book, `btcusdt@trade` for trades).
  - Example Full URL: `wss://ws-fapi.binance.com/ws/btcusdt@depth`.

- **Connection Requirements** (from Binance Docs):
  - **Duration**: Connections are valid for 24 hours; expect disconnection after this period.
  - **Ping/Pong**: Server sends a ping every 3 minutes. Respond with a pong (copying the ping payload) within 10 minutes to avoid disconnection.
  - **Rate Limits**: Max 5 ping/pong frames per second; handshake costs 5 request weight (shared with REST API limits).

- **Helper Function**: `connectWebSocket(url string) (*websocket.Conn, error)`
  - Use `websocket.Dialer.Dial` from `gorilla/websocket` to establish a connection to the provided URL.
  - Return the connection object and any error.
  - Example URL: `wss://ws-fapi.binance.com/ws/btcusdt@depth`.

- **Ping/Pong Handling**:
  - Set a read deadline of 10 minutes on the connection (`conn.SetReadDeadline(time.Now().Add(10 * time.Minute))`).
  - Use `conn.SetPongHandler` to reset the deadline when a pong is received, ensuring the connection stays alive.
  - In a separate goroutine, send a ping every 3 minutes (`conn.WriteMessage(websocket.PingMessage, nil)`).

- **Reconnection Logic**:
  - If a read/write error occurs (e.g., connection closed), attempt to reconnect with exponential backoff:
    - Initial retry: 5 seconds.
    - Increase delay by a factor of 2 (e.g., 5s, 10s, 20s, up to a max of 5 minutes).
    - Reset delay to 5s on successful reconnection.

##### 4. Implement Watch Functions
- **Function**: `WatchOrderBook(symbol string) (chan OrderBookUpdate, func())`
  - Purpose: Streams real-time order book updates for the given symbol (e.g., `BTCUSDT`).
  - Endpoint: `wss://ws-fapi.binance.com/ws/<symbol>@depth` (e.g., `btcusdt@depth` for full depth updates).
  - Returns:
    - A channel (`chan OrderBookUpdate`) for receiving updates.
    - A closer function (`func()`) to stop the stream.
  - Logic:
    - Use `context.WithCancel` to manage goroutine lifecycle.
    - In a goroutine:
      - Connect to the WebSocket endpoint using the helper function.
      - Set up ping/pong handling as described.
      - Continuously read messages (`conn.ReadMessage()`), unmarshal into `OrderBookUpdate`, and send to the channel.
      - On error, close the connection, wait with backoff, and retry.
      - Exit the goroutine when the context is canceled (via the closer).
    - Return the channel and a function that calls `cancel()`.

- **Function**: `WatchTrades(symbol string) (chan TradeUpdate, func())`
  - Purpose: Streams real-time trade updates for the given symbol.
  - Endpoint: `wss://ws-fapi.binance.com/ws/<symbol>@trade` (e.g., `btcusdt@trade`).
  - Returns:
    - A channel (`chan TradeUpdate`) for receiving updates.
    - A closer function (`func()`) to stop the stream.
  - Logic: Identical to `WatchOrderBook`, but unmarshal into `TradeUpdate` instead.

- **Error Handling**:
  - Log errors (e.g., connection failures, JSON parsing issues) using `log.Printf` for debugging.
  - Ensure channels are closed (`defer close(updates)`) when the goroutine exits to prevent deadlocks.

##### 5. Leverage CCXT Go for REST API
- **No Modifications Needed**: CCXT Go’s REST API for Binance futures is production-ready and includes:
  - Market data: `FetchTicker`, `FetchOrderBook`, `FetchTrades`, `FetchOHLCV`.
  - Trading: `CreateMarketBuyOrder`, `CreateLimitSellOrder`, `CancelOrder`, `FetchOpenOrders`.
  - Account: `FetchBalance`, `FetchDepositAddress`, `Withdraw`.
- **Integration**: Expose the `exchange` field or methods like `GetExchange() *ccxt.Binance` to allow users to call these directly on the `streambridge` instance.

##### 6. Package Documentation
- **File**: `README.md`
- **Content**:
  - **Installation**:
    ```
    go get github.com/yourusername/streambridge
    ```
  - **Usage Example**:
    ```go
    package main

    import (
        "fmt"
        "github.com/ccxt/ccxt/go"
        "github.com/yourusername/streambridge"
    )

    func main() {
        exchange := ccxt.NewBinance(&ccxt.Exchange{
            EnableRateLimit: true,
        })
        ws := streambridge.Newstreambridge(exchange)
        
        // Stream order book
        orderBookChan, closer := ws.WatchOrderBook("BTCUSDT")
        go func() {
            for update := range orderBookChan {
                fmt.Printf("Order Book Update: Bids: %v, Asks: %v\n", update.Bids[0], update.Asks[0])
            }
        }()

        // Use REST API
        ticker, _ := ws.GetExchange().FetchTicker("BTCUSDT")
        fmt.Printf("Ticker: %v\n", ticker.Last)

        // Stop after 10 seconds (example)
        time.Sleep(10 * time.Second)
        closer()
    }
    ```
  - **API Reference**:
    - `Newstreambridge(exchange *ccxt.Binance) *streambridge`: Creates a new instance.
    - `WatchOrderBook(symbol string) (chan OrderBookUpdate, func())`: Streams order book updates.
    - `WatchTrades(symbol string) (chan TradeUpdate, func())`: Streams trade updates.
    - `GetExchange() *ccxt.Binance`: Returns the underlying CCXT exchange object for REST calls.
  - **Notes**:
    - WebSocket streams are public; private streams (e.g., user data) are not yet supported.
    - Use testnet (`wss://testnet.binancefuture.com/ws-fapi/v1`) for testing.

##### 7. Testing Instructions
- **Unit Tests**:
  - Create a `streambridge_test.go` file.
  - Test `Newstreambridge` to ensure it initializes correctly.
  - Test `WatchOrderBook` and `WatchTrades`:
    - Connect to the testnet (`wss://testnet.binancefuture.com/ws-fapi/v1`).
    - Verify that channels receive at least one update within 5 seconds.
    - Check that the closer function stops the stream cleanly.
  - Test REST API calls (e.g., `FetchTicker`) using the testnet (`exchange.Options["test"] = true`).

- **Manual Testing**:
  - Run the usage example against the live endpoint (`wss://ws-fapi.binance.com/ws-fapi/v1`).
  - Confirm order book and trade updates are received in real-time.
  - Disconnect after 5 minutes to verify reconnection logic.

##### 8. Additional Notes for Production Readiness
- **Logging**: Use the `log` package for errors and connection status (e.g., "Connected to WebSocket", "Reconnecting...").
- **Concurrency**: Ensure all WebSocket goroutines are managed with context to avoid leaks.
- **Versioning**: Tag the initial release as `v0.1.0` in your Git repo for stability.

#### Full Binance Futures WebSocket Details
- **Base Endpoint**: `wss://ws-fapi.binance.com/ws-fapi/v1`
- **Testnet Endpoint**: `wss://testnet.binancefuture.com/ws-fapi/v1`
- **Stream Types**:
  - Order Book: `<symbol>@depth` (full depth updates).
  - Trades: `<symbol>@trade` (individual trade events).
- **Message Format**: JSON text frames (e.g., `{"e":"depthUpdate","s":"BTCUSDT",...}`).
- **Connection Rules**:
  - 24-hour connection limit.
  - Ping every 3 minutes from server; respond with pong within 10 minutes.
  - Max 5 ping/pong frames per second.

#### Assumptions and Constraints
- CCXT Go’s REST API for Binance futures is assumed to be fully functional (confirmed via community usage).
- Private WebSocket streams (e.g., user data with authentication) are out of scope for this initial version.
- Focus is on public streams only, which don’t require API keys.

#### Deliverables
- A Go package (`streambridge`) with:
  - `streambridge.go`: Core implementation.
  - `streambridge_test.go`: Unit tests.
  - `README.md`: Documentation.
- Git repository hosted at `github.com/yourusername/streambridge`.

#### Final Guidance
- Build the package iteratively: start with the struct and REST integration, then add WebSocket functionality.
- Test thoroughly with the testnet before moving to live endpoints.
- Ensure all errors are logged and handled gracefully to meet production standards.

This instruction provides everything you need to develop the package from scratch, including endpoints, data structures, and design considerations. Proceed with coding, and let me know if you need clarification during implementation!