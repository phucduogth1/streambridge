package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/phucduogth1/streambridge/pkg/streambridge"
	"github.com/phucduogth1/streambridge/pkg/types"
)

func main() {
	// Initialize logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Binance USDM futures market data example...")

	// Create a CCXT exchange instance
	exchange := ccxt.NewBinance(map[string]interface{}{
		// Uncomment to use testnet
		// "test": true,
	})

	// Create a StreamBridge instance
	bridge, err := streambridge.NewStreamBridge(exchange)
	if err != nil {
		log.Fatalf("Failed to create StreamBridge: %v", err)
	}

	// Get the futures stream provider
	futuresProvider, err := bridge.GetStreamProvider(types.Futures)
	if err != nil {
		log.Fatalf("Failed to get futures provider: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up interrupt handler
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		log.Println("Received interrupt, shutting down...")
		cancel()
	}()

	// Define the symbol to watch
	symbol := "BTCUSDT"

	// 1. Watch ticker updates
	tickerChan, tickerCancel, err := futuresProvider.WatchTicker(ctx, symbol, nil)
	if err != nil {
		log.Fatalf("Failed to watch ticker: %v", err)
	}
	defer tickerCancel()

	// 2. Watch trade updates
	tradesChan, tradesCancel, err := futuresProvider.WatchTrades(ctx, symbol, nil)
	if err != nil {
		log.Fatalf("Failed to watch trades: %v", err)
	}
	defer tradesCancel()

	// 3. Watch order book updates
	orderBookChan, orderBookCancel, err := futuresProvider.WatchOrderBook(ctx, symbol, nil)
	if err != nil {
		log.Fatalf("Failed to watch order book: %v", err)
	}
	defer orderBookCancel()

	// Log start time
	startTime := time.Now()
	log.Printf("Subscribed to %s ticker, trades, and order book streams", symbol)
	log.Println("Press Ctrl+C to exit")

	// Process incoming messages from all streams
	for {
		select {
		case <-ctx.Done():
			log.Println("Context canceled, exiting...")
			return

		case ticker, ok := <-tickerChan:
			if !ok {
				log.Println("Ticker channel closed")
				return
			}
			log.Printf("Ticker: %s Last: %s High: %s Low: %s Volume: %s Change: %s%%",
				ticker.Symbol, ticker.Last, ticker.High, ticker.Low, ticker.Volume, ticker.ChangePercent)

		case trade, ok := <-tradesChan:
			if !ok {
				log.Println("Trades channel closed")
				return
			}
			log.Printf("Trade: %s ID: %d Price: %s Qty: %s Side: %s",
				trade.Symbol, trade.ID, trade.Price, trade.Amount, trade.Side)

		case bookUpdate, ok := <-orderBookChan:
			if !ok {
				log.Println("OrderBook channel closed")
				return
			}

			// Log only first bid and ask to avoid flooding the console
			bids := "no bids"
			asks := "no asks"

			if len(bookUpdate.Bids) > 0 {
				bids = bookUpdate.Bids[0][0] + "@" + bookUpdate.Bids[0][1]
			}
			if len(bookUpdate.Asks) > 0 {
				asks = bookUpdate.Asks[0][0] + "@" + bookUpdate.Asks[0][1]
			}

			log.Printf("OrderBook: %s Top Bid: %s Top Ask: %s",
				bookUpdate.Symbol, bids, asks)

		// Add a timeout to exit after a short demo period if desired
		case <-time.After(60 * time.Second):
			log.Printf("Demo ran for 60 seconds, received data from all streams")
			log.Printf("Uptime: %v", time.Since(startTime))
			cancel()
			return
		}
	}
}
