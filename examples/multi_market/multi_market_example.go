package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/phucduogth1/streambridge/pkg/streambridge"
	"github.com/phucduogth1/streambridge/pkg/types"
)

func main() {
	fmt.Println("StreamBridge Multi-Market Example")
	fmt.Println("=================================")

	// Initialize CCXT Binance exchange with testnet
	exchange := ccxt.NewBinance(map[string]interface{}{
		"enableRateLimit": true,
		"test":            true, // Use testnet for testing
	})

	// Create StreamBridge instance
	bridge, err := streambridge.NewStreamBridge(exchange)
	if err != nil {
		fmt.Printf("Error creating StreamBridge: %v\n", err)
		return
	}

	// Get the supported market types
	marketTypes := bridge.GetSupportedMarketTypes()
	fmt.Printf("Supported market types: %v\n", marketTypes)

	// Create a context that will be used for all streams
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup a channel to listen for interrupt signals to terminate the program gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start a stream on the futures market
	if futuresProvider, err := bridge.GetStreamProvider(types.Futures); err == nil {
		// Stream order book
		orderBookChan, closer, err := futuresProvider.WatchOrderBook(ctx, "BTCUSDT", nil)
		if err != nil {
			fmt.Printf("Error starting order book stream: %v\n", err)
		} else {
			defer closer()

			// Process order book updates in a goroutine
			go func() {
				fmt.Println("📊 [FUTURES] Listening for order book updates...")
				for update := range orderBookChan {
					// Print the first bid and ask if available
					if len(update.Bids) > 0 && len(update.Asks) > 0 {
						fmt.Printf("📊 [FUTURES] Order Book: Best Bid: %s | Best Ask: %s\n",
							update.Bids[0][0], update.Asks[0][0])
					}
				}
			}()
		}

		// Stream trades
		tradesChan, tradesCloser, err := futuresProvider.WatchTrades(ctx, "BTCUSDT", nil)
		if err != nil {
			fmt.Printf("Error starting trades stream: %v\n", err)
		} else {
			defer tradesCloser()

			// Process trade updates in a goroutine
			go func() {
				fmt.Println("🔄 [FUTURES] Listening for trade updates...")
				for trade := range tradesChan {
					fmt.Printf("🔄 [FUTURES] Trade: %s | Price: %s | Amount: %s\n",
						trade.Side, trade.Price, trade.Amount)
				}
			}()
		}
	} else {
		fmt.Printf("Futures market not supported: %v\n", err)
	}

	// Example of how you would use spot market (when fully implemented)
	if _, err := bridge.GetStreamProvider(types.Spot); err == nil {
		fmt.Println("Spot market is supported, but implementation is a skeleton for now.")
		/*
			// Stream order book for spot market
			orderBookChan, closer, err := spotProvider.WatchOrderBook(ctx, "BTCUSDT", nil)
			if err == nil {
				defer closer()
				// Process spot order book updates
			}
		*/
	} else {
		fmt.Printf("Spot market not fully supported yet: %v\n", err)
	}

	// Wait for interrupt signal or timeout (for demo purposes)
	fmt.Println("\nStreaming data (press Ctrl+C to exit or wait 30 seconds)...")
	select {
	case <-sigChan:
		fmt.Println("\nShutting down by user request...")
	case <-time.After(30 * time.Second):
		fmt.Println("\nShutting down after timeout...")
	}

	fmt.Println("Streams closed. Example complete.")
}
