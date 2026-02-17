package wallet

import (
	"context"
	"fmt"
	"log"
	"sync"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	toolbox "github.com/bsv-blockchain/go-wallet-toolbox/pkg/wallet"

	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/defs"
	wdkStorage "github.com/bsv-blockchain/go-wallet-toolbox/pkg/storage"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/wdk"
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

// InitWallet initializes the wallet-toolbox wallet using the configured
// server private key and wallet storage URL. It must be called once at startup.
func (p *Provider) InitWallet(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.wallet != nil {
		return nil // already initialized
	}

	if p.serverPrivateKey == "" {
		return fmt.Errorf("SERVER_PRIVATE_KEY is required for wallet initialization")
	}

	chain := defs.NetworkMainnet
	if p.bsvNetwork == "testnet" {
		chain = defs.NetworkTestnet
	}

	privKey, err := ec.PrivateKeyFromHex(p.serverPrivateKey)
	if err != nil {
		return fmt.Errorf("invalid server private key: %w", err)
	}

	// Create a storage factory that connects to the remote wallet storage server
	storageFactory := toolbox.StorageProviderFactoryWithWalletReturningCleanupAndError(
		func(w sdkWallet.Interface) (wdk.WalletStorageProvider, func(), error) {
			return wdkStorage.NewClient(p.walletStorageURL, w)
		},
	)

	w, err := toolbox.NewWithStorageFactory(chain, privKey, storageFactory)
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	p.wallet = w
	log.Printf("Wallet initialized, identity key: %s", privKey.PubKey().ToDERHex())
	return nil
}

// GetWallet returns the wallet instance. May be nil if not yet initialized.
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
