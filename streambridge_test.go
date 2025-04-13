package streambridge

import (
	"testing"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

func TestNewStreamBridge(t *testing.T) {
	params := map[string]interface{}{
		"enableRateLimit": true,
	}
	exchange := ccxt.NewBinance(params)

	ce := NewStreamBridge(exchange)
	if ce == nil {
		t.Error("NewStreamBridge returned nil")
	}

	if ce.GetExchange() == nil {
		t.Error("GetExchange returned nil")
	}
}

func TestWatchOrderBook(t *testing.T) {
	params := map[string]interface{}{
		"enableRateLimit": true,
		"test":            true, // Use testnet
	}
	exchange := ccxt.NewBinance(params)

	ce := NewStreamBridge(exchange)
	updates, closer := ce.WatchOrderBook("BTCUSDT")

	// Set up a channel to signal if we received an update
	received := make(chan bool)

	go func() {
		update := <-updates
		if update.Symbol != "BTCUSDT" {
			t.Errorf("Expected symbol BTCUSDT, got %s", update.Symbol)
		}
		if len(update.Bids) == 0 && len(update.Asks) == 0 {
			t.Error("Expected non-empty bids or asks")
		}
		received <- true
	}()

	// Wait for update with timeout
	select {
	case <-received:
	// Success
	case <-time.After(5 * time.Second):
		t.Error("Timed out waiting for order book update")
	}

	closer()
}

func TestWatchTrades(t *testing.T) {
	params := map[string]interface{}{
		"enableRateLimit": true,
		"test":            true, // Use testnet
	}
	exchange := ccxt.NewBinance(params)

	ce := NewStreamBridge(exchange)
	updates, closer := ce.WatchTrades("BTCUSDT")

	// Set up a channel to signal if we received an update
	received := make(chan bool)

	go func() {
		update := <-updates
		if update.Symbol != "BTCUSDT" {
			t.Errorf("Expected symbol BTCUSDT, got %s", update.Symbol)
		}
		if update.Price == "" || update.Quantity == "" {
			t.Error("Expected non-empty price and quantity")
		}
		received <- true
	}()

	// Wait for update with timeout
	select {
	case <-received:
	// Success
	case <-time.After(5 * time.Second):
		t.Error("Timed out waiting for trade update")
	}

	closer()
}

func TestRESTIntegration(t *testing.T) {
	params := map[string]interface{}{
		"enableRateLimit": true,
		"test":            true, // Use testnet
	}
	exchange := ccxt.NewBinance(params)

	ce := NewStreamBridge(exchange)

	// Test fetching ticker
	ticker, err := ce.GetExchange().FetchTicker("BTCUSDT")
	if err != nil {
		t.Errorf("Failed to fetch ticker: %v", err)
		return
	}
	if ticker.Last == nil {
		t.Error("Expected non-nil last price in ticker")
		return
	}
	if *ticker.Last <= 0 {
		t.Error("Expected positive last price in ticker")
	}

	// Test fetching order book
	orderbook, err := ce.GetExchange().FetchOrderBook("BTCUSDT")
	if err != nil {
		t.Errorf("Failed to fetch order book: %v", err)
		return
	}
	if len(orderbook.Bids) == 0 && len(orderbook.Asks) == 0 {
		t.Error("Expected non-empty order book")
	}
}
