package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/phucduogth1/streambridge"
)

func main() {
	fmt.Println("StreamBridge Example")
	fmt.Println("===================")

	// Initialize CCXT Binance exchange with testnet
	exchange := ccxt.NewBinance(map[string]interface{}{
		"enableRateLimit": true,
		"test":            true, // Use testnet for testing
	})

	// Create StreamBridge instance
	bridge := streambridge.NewStreamBridge(exchange)

	// Test REST API functionality
	fmt.Println("\nTesting REST API (via CCXT):")
	ticker, err := bridge.GetExchange().FetchTicker("BTCUSDT")
	if err != nil {
		fmt.Printf("❌ Error fetching ticker: %v\n", err)
	} else {
		fmt.Printf("✅ Current BTC price: %v USDT\n", *ticker.Last)
	}

	// Setup a channel to listen for interrupt signals to terminate the program gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\nStarting WebSocket streams...")
	fmt.Println("(Press Ctrl+C to exit)\n")

	// Start order book stream
	orderBookChan, closeOrderBook := bridge.WatchOrderBook("BTCUSDT")
	go func() {
		fmt.Println("📊 Listening for order book updates...")
		for update := range orderBookChan {
			bidPrice, bidQty := "N/A", "N/A"
			askPrice, askQty := "N/A", "N/A"

			if len(update.Bids) > 0 && len(update.Bids[0]) >= 2 {
				bidPrice, bidQty = update.Bids[0][0], update.Bids[0][1]
			}

			if len(update.Asks) > 0 && len(update.Asks[0]) >= 2 {
				askPrice, askQty = update.Asks[0][0], update.Asks[0][1]
			}

			fmt.Printf("📊 Order Book: Best Bid: %s (%s) | Best Ask: %s (%s)\n",
				bidPrice, bidQty, askPrice, askQty)
		}
	}()

	// Start trade stream
	tradeChan, closeTrade := bridge.WatchTrades("BTCUSDT")
	go func() {
		fmt.Println("🔄 Listening for trade updates...")
		for trade := range tradeChan {
			side := "SELL"
			if trade.Buyer {
				side = "BUY"
			}
			fmt.Printf("🔄 Trade: %s | Price: %s | Quantity: %s\n",
				side, trade.Price, trade.Quantity)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\nShutting down...")

	// Close streams
	closeOrderBook()
	closeTrade()

	fmt.Println("Streams closed. Exiting.")
}
