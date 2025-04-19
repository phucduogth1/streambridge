package futures

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/phucduogth1/streambridge/pkg/exchange/binance"
)

const (
	// listenKeyRenewalInterval is the time between listen key renewals
	// Binance requires renewal every 30 minutes, we'll do it every 25 to be safe
	listenKeyRenewalInterval = 25 * time.Minute
)

// ListenKeyManager handles Binance futures listen key creation, renewal, and deletion
type ListenKeyManager struct {
	exchange  *binance.BinanceExchange
	listenKey string
	isTestnet bool
	stopRenew chan struct{}
	renewDone chan struct{}
}

// NewListenKeyManager creates a new listen key manager for futures
func NewListenKeyManager(exchange *binance.BinanceExchange) *ListenKeyManager {
	return &ListenKeyManager{
		exchange:  exchange,
		isTestnet: exchange.IsTestnet(),
		stopRenew: make(chan struct{}),
		renewDone: make(chan struct{}),
	}
}

// GetListenKey creates or returns the current futures listen key
func (lkm *ListenKeyManager) GetListenKey(ctx context.Context) (string, error) {
	// If we already have a listen key, return it
	if lkm.listenKey != "" {
		return lkm.listenKey, nil
	}

	// Get the CCXT exchange implementation
	ccxtExchange, ok := lkm.exchange.GetExchange().(*ccxt.Binance)
	if !ok || ccxtExchange == nil {
		return "", errors.New("invalid exchange implementation")
	}

	// Make the API call to get a futures listen key
	resultChan := ccxtExchange.FapiPrivatePostListenKey(nil)

	// Wait for and extract the result from the channel
	result, ok := (<-resultChan).(map[string]interface{})
	if !ok {
		return "", errors.New("unexpected response format from listen key endpoint")
	}

	listenKey, ok := result["listenKey"].(string)
	if !ok || listenKey == "" {
		return "", errors.New("listen key not found in response")
	}

	lkm.listenKey = listenKey
	return lkm.listenKey, nil
}

// StartRenewing begins the periodic renewal of the listen key
func (lkm *ListenKeyManager) StartRenewing(ctx context.Context) error {
	// Make sure we have a listen key
	if lkm.listenKey == "" {
		_, err := lkm.GetListenKey(ctx)
		if err != nil {
			return fmt.Errorf("failed to get listen key before starting renewal: %v", err)
		}
	}

	go lkm.renewalLoop(ctx)
	return nil
}

// renewalLoop periodically renews the listen key to keep it active
func (lkm *ListenKeyManager) renewalLoop(ctx context.Context) {
	defer close(lkm.renewDone)

	ticker := time.NewTicker(listenKeyRenewalInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-lkm.stopRenew:
			return
		case <-ticker.C:
			if err := lkm.RenewListenKey(ctx); err != nil {
				log.Printf("Failed to renew futures listen key: %v", err)
			}
		}
	}
}

// RenewListenKey makes an API call to renew the current futures listen key
func (lkm *ListenKeyManager) RenewListenKey(ctx context.Context) error {
	if lkm.listenKey == "" {
		return errors.New("no listen key to renew")
	}

	ccxtExchange, ok := lkm.exchange.GetExchange().(*ccxt.Binance)
	if !ok || ccxtExchange == nil {
		return errors.New("invalid exchange implementation")
	}

	// The PUT request to renew the futures listen key
	params := map[string]interface{}{
		"listenKey": lkm.listenKey,
	}

	// Use the futures-specific endpoint and wait for the result
	resultChan := ccxtExchange.FapiPrivatePutListenKey(params)

	// Wait for the result to ensure the request completes
	<-resultChan

	log.Printf("Successfully renewed futures listen key")
	return nil
}

// StopRenewing stops the renewal loop and invalidates the listen key
func (lkm *ListenKeyManager) StopRenewing() {
	if lkm.stopRenew == nil {
		return
	}

	// Signal the renewal loop to stop
	close(lkm.stopRenew)

	// Wait for the renewal loop to finish
	<-lkm.renewDone

	// Delete the listen key if we have one
	if lkm.listenKey != "" {
		lkm.DeleteListenKey()
		lkm.listenKey = ""
	}
}

// DeleteListenKey makes an API call to delete the current futures listen key
func (lkm *ListenKeyManager) DeleteListenKey() error {
	if lkm.listenKey == "" {
		return nil
	}

	ccxtExchange, ok := lkm.exchange.GetExchange().(*ccxt.Binance)
	if !ok || ccxtExchange == nil {
		return errors.New("invalid exchange implementation")
	}

	params := map[string]interface{}{
		"listenKey": lkm.listenKey,
	}

	// Use the futures-specific endpoint and wait for the result
	resultChan := ccxtExchange.FapiPrivateDeleteListenKey(params)

	// Wait for the result from the channel to ensure the request completes
	result := <-resultChan

	// Check if there was an error in the result
	if err, ok := result.(string); ok && len(err) > 6 && err[:6] == "panic:" {
		log.Printf("Warning: Failed to delete futures listen key: %v", err)
	}

	return nil
}
