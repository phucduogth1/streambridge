package streambridge

import (
	"errors"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/phucduogth1/streambridge/pkg/exchange/binance"
	"github.com/phucduogth1/streambridge/pkg/exchange/binance/futures"
	"github.com/phucduogth1/streambridge/pkg/exchange/binance/spot"
	"github.com/phucduogth1/streambridge/pkg/types"
)

// StreamBridgeFactory creates StreamBridge instances for different exchanges
type StreamBridgeFactory struct{}

// NewStreamBridgeFactory creates a new StreamBridge factory
func NewStreamBridgeFactory() *StreamBridgeFactory {
	return &StreamBridgeFactory{}
}

// CreateForBinance creates a StreamBridge for Binance
func (f *StreamBridgeFactory) CreateForBinance(exchange ccxt.Binance) (types.StreamBridge, error) {
	// Create a Binance StreamBridge
	streamBridge, err := binance.NewBinanceStreamBridge(exchange)
	if err != nil {
		return nil, err
	}

	// Create and register the Binance Futures provider
	binanceExchange := binance.NewBinanceExchange(exchange)

	// Register the futures provider
	futuresProvider := futures.NewBinanceFuturesProvider(binanceExchange)
	streamBridge.RegisterProvider(types.Futures, futuresProvider)

	// Register the spot provider
	spotProvider := spot.NewBinanceSpotProvider(binanceExchange)
	streamBridge.RegisterProvider(types.Spot, spotProvider)

	// Register the futures user data provider (replaces the old userdata provider)
	userDataProvider := futures.NewFuturesUserDataProvider(binanceExchange)
	streamBridge.RegisterUserDataProvider(userDataProvider)

	return streamBridge, nil
}

// NewStreamBridge is a convenience function to create a StreamBridge directly from a CCXT exchange
func NewStreamBridge(exchange interface{}) (types.StreamBridge, error) {
	factory := NewStreamBridgeFactory()

	// Check which exchange type it is
	switch e := exchange.(type) {
	case ccxt.Binance:
		return factory.CreateForBinance(e)
	default:
		return nil, errors.New("unsupported exchange type")
	}
}
