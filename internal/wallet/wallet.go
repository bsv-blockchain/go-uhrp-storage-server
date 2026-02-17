package wallet

import (
	"sync"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

// Provider manages the wallet singleton.
type Provider struct {
	serverPrivateKey string
	walletStorageURL string
	bsvNetwork       string
	wallet           sdkWallet.Interface
	mu               sync.Mutex
}

// NewProvider creates a wallet provider.
func NewProvider(serverPrivateKey, walletStorageURL, bsvNetwork string) *Provider {
	return &Provider{
		serverPrivateKey: serverPrivateKey,
		walletStorageURL: walletStorageURL,
		bsvNetwork:       bsvNetwork,
	}
}

// GetWallet returns the wallet instance.
// NOTE: In a full implementation this would initialize the wallet-toolbox-client
// similar to the TypeScript version. For now we return a placeholder that the
// auth/payment middleware can use. The actual wallet initialization depends on
// go-wallet-toolbox availability.
func (p *Provider) GetWallet() sdkWallet.Interface {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.wallet
}

// SetWallet allows injecting a wallet instance (for testing or external init).
func (p *Provider) SetWallet(w sdkWallet.Interface) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.wallet = w
}

// ServerPrivateKey returns the configured server private key.
func (p *Provider) ServerPrivateKey() string {
	return p.serverPrivateKey
}

// BSVNetwork returns the configured network.
func (p *Provider) BSVNetwork() string {
	return p.bsvNetwork
}
