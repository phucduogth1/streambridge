package binance

import (
	"errors"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/phucduogth1/streambridge/pkg/types"
)

// BinanceExchange implements the Exchange interface for Binance
type BinanceExchange struct {
	exchange  ccxt.Binance
	isTestnet bool
}

// NewBinanceExchange creates a new Binance exchange instance
func NewBinanceExchange(exchange ccxt.Binance) *BinanceExchange {
	// Check if testnet is enabled in the exchange options
	isTestnet := false
	if options := exchange.GetOptions(); options != nil {
		if testOption, ok := options["test"]; ok {
			if testBool, ok := testOption.(bool); ok && testBool {
				isTestnet = true
			}
		}
	}

	return &BinanceExchange{
		exchange:  exchange,
		isTestnet: isTestnet,
	}
}

// GetExchange returns the underlying exchange implementation
func (b *BinanceExchange) GetExchange() interface{} {
	return &b.exchange
}

// GetName returns the name of the exchange
func (b *BinanceExchange) GetName() string {
	return "binance"
}

// IsTestnet returns whether the exchange is connected to testnet
func (b *BinanceExchange) IsTestnet() bool {
	return b.isTestnet
}

// BinanceStreamBridge is the main implementation of StreamBridge for Binance
type BinanceStreamBridge struct {
	exchange     *BinanceExchange
	providers    map[types.MarketType]types.StreamProvider
	userProvider types.UserDataProvider
}

// NewBinanceStreamBridge creates a new Binance stream bridge
func NewBinanceStreamBridge(exchange ccxt.Binance) (*BinanceStreamBridge, error) {
	binanceExchange := NewBinanceExchange(exchange)

	sb := &BinanceStreamBridge{
		exchange:  binanceExchange,
		providers: make(map[types.MarketType]types.StreamProvider),
	}

	return sb, nil
}

// GetStreamProvider returns the appropriate stream provider for the given market type
func (b *BinanceStreamBridge) GetStreamProvider(marketType types.MarketType) (types.StreamProvider, error) {
	provider, ok := b.providers[marketType]
	if !ok {
		return nil, errors.New("unsupported market type: " + string(marketType))
	}
	return provider, nil
}

// GetUserDataProvider returns the user data provider if available
func (b *BinanceStreamBridge) GetUserDataProvider() (types.UserDataProvider, error) {
	if b.userProvider == nil {
		return nil, errors.New("user data provider not available")
	}
	return b.userProvider, nil
}

// GetSupportedMarketTypes returns all market types supported by this exchange
func (b *BinanceStreamBridge) GetSupportedMarketTypes() []types.MarketType {
	keys := make([]types.MarketType, 0, len(b.providers))
	for k := range b.providers {
		keys = append(keys, k)
	}

	// Add UserData if available
	if b.userProvider != nil {
		keys = append(keys, types.UserData)
	}

	return keys
}

// GetDefaultStreamProvider returns the default stream provider (futures)
func (b *BinanceStreamBridge) GetDefaultStreamProvider() types.StreamProvider {
	// Try futures first, then spot
	if provider, ok := b.providers[types.Futures]; ok {
		return provider
	}
	if provider, ok := b.providers[types.Spot]; ok {
		return provider
	}

	// Return the first available provider or nil
	for _, provider := range b.providers {
		return provider
	}

	return nil
}

// GetName returns the name of the exchange
func (b *BinanceStreamBridge) GetName() string {
	return b.exchange.GetName()
}

// GetCcxtExchange returns the underlying CCXT exchange object
func (b *BinanceStreamBridge) GetCcxtExchange() interface{} {
	return b.exchange.GetExchange()
}

// RegisterProvider registers a stream provider for a specific market type
func (b *BinanceStreamBridge) RegisterProvider(marketType types.MarketType, provider types.StreamProvider) {
	b.providers[marketType] = provider
}

// RegisterUserDataProvider registers a user data provider
func (b *BinanceStreamBridge) RegisterUserDataProvider(provider types.UserDataProvider) {
	b.userProvider = provider
}
