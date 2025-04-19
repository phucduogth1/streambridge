package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/phucduogth1/streambridge/pkg/streambridge"
)

func main() {
	fmt.Println("StreamBridge User Data Stream Example")
	fmt.Println("====================================")

	// Check if API key is provided
	apiKey := os.Getenv("BINANCE_API_KEY")
	apiSecret := os.Getenv("BINANCE_API_SECRET")

	if apiKey == "" || apiSecret == "" {
		fmt.Println("❌ Error: BINANCE_API_KEY and BINANCE_API_SECRET environment variables must be set")
		fmt.Println("User data streams require authentication to access private data.")
		return
	}

	// Initialize CCXT Binance exchange
	exchange := ccxt.NewBinance(map[string]interface{}{
		"apiKey":          apiKey,
		"secret":          apiSecret,
		"enableRateLimit": true,
		// Uncomment to use testnet
		// "test": true,
	})

	fmt.Println("Creating StreamBridge instance...")
	// Create StreamBridge instance
	bridge, err := streambridge.NewStreamBridge(exchange)
	if err != nil {
		fmt.Printf("❌ Error creating StreamBridge: %v\n", err)
		return
	}

	// Get the user data provider
	userDataProvider, err := bridge.GetUserDataProvider()
	if err != nil {
		fmt.Printf("❌ Error getting user data provider: %v\n", err)
		return
	}

	fmt.Println("✅ Successfully connected to StreamBridge")

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup channels to receive order and balance updates
	fmt.Println("Setting up order and balance update streams...")

	// Subscribe to order updates
	orderUpdates, orderCancel, err := userDataProvider.WatchOrders(ctx, nil)
	if err != nil {
		fmt.Printf("❌ Error subscribing to order updates: %v\n", err)
		return
	}
	defer orderCancel()

	// Subscribe to balance updates
	balanceUpdates, balanceCancel, err := userDataProvider.WatchBalances(ctx, nil)
	if err != nil {
		fmt.Printf("❌ Error subscribing to balance updates: %v\n", err)
		return
	}
	defer balanceCancel()

	fmt.Println("✅ Successfully subscribed to user data streams")
	fmt.Println("\n📊 Listening for order and balance updates...")
	fmt.Println("(Place or cancel some orders to see updates)")
	fmt.Println("(Press Ctrl+C to exit)\n")

	// Set up a signal handler to gracefully exit
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// Main loop to process updates
	for {
		select {
		case <-signals:
			fmt.Println("\nShutting down...")
			return

		case orderUpdate, ok := <-orderUpdates:
			if !ok {
				fmt.Println("❌ Order updates channel closed")
				return
			}

			fmt.Printf("\n📝 Order Update:\n")
			fmt.Printf("  Symbol: %s\n", orderUpdate.Symbol)
			fmt.Printf("  Order ID: %d\n", orderUpdate.OrderID)
			fmt.Printf("  Client Order ID: %s\n", orderUpdate.ClientOrderID)
			fmt.Printf("  Side: %s\n", orderUpdate.Side)
			fmt.Printf("  Type: %s\n", orderUpdate.Type)
			fmt.Printf("  Price: %s\n", orderUpdate.Price)
			fmt.Printf("  Original Qty: %s\n", orderUpdate.OriginalQty)
			fmt.Printf("  Executed Qty: %s\n", orderUpdate.ExecutedQty)
			fmt.Printf("  Status: %s\n", orderUpdate.Status)
			fmt.Printf("  Time: %s\n", time.Unix(0, orderUpdate.Timestamp*int64(time.Millisecond)).Format(time.RFC3339))

		case balanceUpdate, ok := <-balanceUpdates:
			if !ok {
				fmt.Println("❌ Balance updates channel closed")
				return
			}

			fmt.Printf("\n💰 Balance Update:\n")
			fmt.Printf("  Time: %s\n", time.Unix(0, balanceUpdate.Timestamp*int64(time.Millisecond)).Format(time.RFC3339))
			fmt.Printf("  Assets:\n")

			for asset, balance := range balanceUpdate.Balances {
				fmt.Printf("    %s: Free=%s, Locked=%s, Total=%s\n",
					asset, balance.Free, balance.Locked, balance.Total)
			}
		}
	}
}
