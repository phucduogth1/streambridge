package futures

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/phucduogth1/streambridge/pkg/exchange/binance"
	"github.com/phucduogth1/streambridge/pkg/types"
)

// FuturesUserDataProvider implements the FuturesUserDataProvider interface
// specifically for Binance USDM Futures
type FuturesUserDataProvider struct {
	exchange *binance.BinanceExchange
	conn     *UserDataConnection
}

// NewFuturesUserDataProvider creates a new futures-specific user data provider
func NewFuturesUserDataProvider(exchange *binance.BinanceExchange) *FuturesUserDataProvider {
	return &FuturesUserDataProvider{
		exchange: exchange,
		conn:     NewUserDataConnection(exchange),
	}
}

// WatchOrders adds futures-specific user data order watching capability to the provider
func (p *FuturesUserDataProvider) WatchOrders(ctx context.Context, options map[string]interface{}) (chan types.OrderUpdate, func(), error) {
	// Ensure connection is established
	if err := p.conn.Connect(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to futures user data stream: %w", err)
	}

	// Create a channel for the order updates
	orderChan := make(chan types.OrderUpdate, 100)

	// Create a cancel function
	ctxWithCancel, cancel := context.WithCancel(ctx)

	// Start a goroutine to process updates
	go func() {
		defer close(orderChan)

		// Get the raw order updates channel
		rawUpdates := p.conn.GetOrderChannel()

		for {
			select {
			case <-ctxWithCancel.Done():
				return
			case rawMsg, ok := <-rawUpdates:
				if !ok {
					return
				}

				// Process only futures-specific order updates ("ORDER_TRADE_UPDATE")
				var baseEvent struct {
					EventType string `json:"e"`
				}

				if err := json.Unmarshal(rawMsg, &baseEvent); err != nil {
					log.Printf("Error parsing order update: %v", err)
					continue
				}

				// Skip non-futures order updates
				if baseEvent.EventType != "ORDER_TRADE_UPDATE" {
					continue
				}

				// Parse the futures order update
				orderUpdate, err := parseFuturesOrderUpdate(rawMsg)
				if err != nil {
					log.Printf("Error parsing futures order update: %v", err)
					continue
				}

				// Send the update to the client
				select {
				case <-ctxWithCancel.Done():
					return
				case orderChan <- orderUpdate:
					// Successfully sent
				}
			}
		}
	}()

	return orderChan, func() {
		cancel()
	}, nil
}

// WatchBalances adds futures-specific user data balance watching capability to the provider
func (p *FuturesUserDataProvider) WatchBalances(ctx context.Context, options map[string]interface{}) (chan types.BalanceUpdate, func(), error) {
	// Ensure connection is established
	if err := p.conn.Connect(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to futures user data stream: %w", err)
	}

	// Create a channel for the balance updates
	balanceChan := make(chan types.BalanceUpdate, 100)

	// Create a cancel function
	ctxWithCancel, cancel := context.WithCancel(ctx)

	// Start a goroutine to process updates
	go func() {
		defer close(balanceChan)

		// Get the raw balance updates channel
		rawUpdates := p.conn.GetBalanceChannel()

		for {
			select {
			case <-ctxWithCancel.Done():
				return
			case rawMsg, ok := <-rawUpdates:
				if !ok {
					return
				}

				// Process only futures-specific balance updates ("ACCOUNT_UPDATE")
				var baseEvent struct {
					EventType string `json:"e"`
				}

				if err := json.Unmarshal(rawMsg, &baseEvent); err != nil {
					log.Printf("Error parsing balance update: %v", err)
					continue
				}

				// Skip non-futures balance updates
				if baseEvent.EventType != "ACCOUNT_UPDATE" {
					continue
				}

				// Parse the futures balance update
				balanceUpdate, err := parseFuturesBalanceUpdate(rawMsg)
				if err != nil {
					log.Printf("Error parsing futures balance update: %v", err)
					continue
				}

				// Send the update to the client
				select {
				case <-ctxWithCancel.Done():
					return
				case balanceChan <- balanceUpdate:
					// Successfully sent
				}
			}
		}
	}()

	return balanceChan, func() {
		cancel()
	}, nil
}

// WatchPositions adds futures-specific position watching capability to the provider
func (p *FuturesUserDataProvider) WatchPositions(ctx context.Context, options map[string]interface{}) (chan types.PositionUpdate, func(), error) {
	// Ensure connection is established
	if err := p.conn.Connect(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to futures user data stream: %w", err)
	}

	// Create a channel for the position updates
	positionChan := make(chan types.PositionUpdate, 100)

	// Create a cancel function
	ctxWithCancel, cancel := context.WithCancel(ctx)

	// Start a goroutine to process updates
	go func() {
		defer close(positionChan)

		// Position updates come through the position channel
		rawUpdates := p.conn.GetPositionChannel()

		for {
			select {
			case <-ctxWithCancel.Done():
				return
			case rawMsg, ok := <-rawUpdates:
				if !ok {
					return
				}

				// Process each position update
				positionUpdate, err := parseFuturesPositionUpdate(rawMsg)
				if err != nil {
					log.Printf("Error parsing position update: %v", err)
					continue
				}

				select {
				case <-ctxWithCancel.Done():
					return
				case positionChan <- positionUpdate:
					// Successfully sent
				}
			}
		}
	}()

	return positionChan, func() {
		cancel()
	}, nil
}

// GetExchange returns the underlying exchange implementation
func (p *FuturesUserDataProvider) GetExchange() types.Exchange {
	return p.exchange
}

// parseFuturesOrderUpdate parses a raw message into an OrderUpdate
func parseFuturesOrderUpdate(data json.RawMessage) (types.OrderUpdate, error) {
	var futuresOrder struct {
		EventTime int64 `json:"E"`
		OrderData struct {
			Symbol        string `json:"s"`
			ClientOrderID string `json:"c"`
			OrderID       int64  `json:"i"`
			Side          string `json:"S"`
			Type          string `json:"o"`
			TimeInForce   string `json:"f"`
			OrigQty       string `json:"q"`
			ExecQty       string `json:"z"`
			QuoteQty      string `json:"Z"`
			Price         string `json:"p"`
			Status        string `json:"X"`
		} `json:"o"`
	}

	if err := json.Unmarshal(data, &futuresOrder); err != nil {
		return types.OrderUpdate{}, fmt.Errorf("error parsing futures order update: %v", err)
	}

	return types.OrderUpdate{
		Symbol:             futuresOrder.OrderData.Symbol,
		Timestamp:          futuresOrder.EventTime,
		ClientOrderID:      futuresOrder.OrderData.ClientOrderID,
		OrderID:            futuresOrder.OrderData.OrderID,
		Side:               futuresOrder.OrderData.Side,
		Type:               futuresOrder.OrderData.Type,
		TimeInForce:        futuresOrder.OrderData.TimeInForce,
		OriginalQty:        futuresOrder.OrderData.OrigQty,
		ExecutedQty:        futuresOrder.OrderData.ExecQty,
		CumulativeQuoteQty: futuresOrder.OrderData.QuoteQty,
		Price:              futuresOrder.OrderData.Price,
		Status:             futuresOrder.OrderData.Status,
		Raw:                data,
	}, nil
}

// parseFuturesBalanceUpdate parses a raw message into a BalanceUpdate
func parseFuturesBalanceUpdate(data json.RawMessage) (types.BalanceUpdate, error) {
	var futuresUpdate struct {
		EventTime int64 `json:"E"`
		Data      struct {
			Balances []struct {
				Asset  string `json:"a"`
				Free   string `json:"wb"`
				Locked string `json:"cw"`
			} `json:"B"`
		} `json:"a"`
	}

	if err := json.Unmarshal(data, &futuresUpdate); err != nil {
		return types.BalanceUpdate{}, fmt.Errorf("error parsing futures balance update: %v", err)
	}

	balanceUpdate := types.BalanceUpdate{
		Timestamp: futuresUpdate.EventTime,
		Balances:  make(map[string]types.AssetBalance),
		Raw:       data,
	}

	for _, bal := range futuresUpdate.Data.Balances {
		balanceUpdate.Balances[bal.Asset] = types.AssetBalance{
			Free:   bal.Free,
			Locked: bal.Locked,
			Total:  bal.Free, // In futures, wallet balance is considered total
		}
	}

	return balanceUpdate, nil
}

// parseFuturesPositionUpdate parses position data into a PositionUpdate
func parseFuturesPositionUpdate(data json.RawMessage) (types.PositionUpdate, error) {
	var position struct {
		Symbol           string `json:"s"`
		Side             string `json:"ps"`
		Amount           string `json:"pa"`
		EntryPrice       string `json:"ep"`
		MarkPrice        string `json:"mp"`
		UnrealizedProfit string `json:"up"`
		MarginType       string `json:"mt"`
		IsolatedMargin   string `json:"iw"`
		Leverage         int    `json:"l"`
	}

	if err := json.Unmarshal(data, &position); err != nil {
		return types.PositionUpdate{}, fmt.Errorf("error parsing position update: %v", err)
	}

	return types.PositionUpdate{
		Symbol:           position.Symbol,
		Side:             position.Side,
		PositionAmt:      position.Amount,
		EntryPrice:       position.EntryPrice,
		MarkPrice:        position.MarkPrice,
		UnrealizedProfit: position.UnrealizedProfit,
		MarginType:       position.MarginType,
		IsolatedMargin:   position.IsolatedMargin,
		Leverage:         position.Leverage,
		Raw:              data,
	}, nil
}

// Add the futures-specific user data watching methods to the BinanceFuturesProvider
func (b *BinanceFuturesProvider) WatchOrders(ctx context.Context, options map[string]interface{}) (chan types.OrderUpdate, func(), error) {
	provider := NewFuturesUserDataProvider(b.exchange)
	return provider.WatchOrders(ctx, options)
}

func (b *BinanceFuturesProvider) WatchBalances(ctx context.Context, options map[string]interface{}) (chan types.BalanceUpdate, func(), error) {
	provider := NewFuturesUserDataProvider(b.exchange)
	return provider.WatchBalances(ctx, options)
}

func (b *BinanceFuturesProvider) WatchPositions(ctx context.Context, options map[string]interface{}) (chan types.PositionUpdate, func(), error) {
	provider := NewFuturesUserDataProvider(b.exchange)
	return provider.WatchPositions(ctx, options)
}
