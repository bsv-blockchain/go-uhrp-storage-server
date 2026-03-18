package wallet

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/bsv-blockchain/go-sdk/overlay"
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
	Logger           *slog.Logger
}

// NewProvider creates a wallet provider.
func NewProvider(serverPrivateKey, walletStorageURL, bsvNetwork string, logger *slog.Logger) *Provider {
	return &Provider{
		serverPrivateKey: serverPrivateKey,
		walletStorageURL: walletStorageURL,
		bsvNetwork:       bsvNetwork,
		Logger:           logger.With("component", "wallet_provider"),
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
	p.Logger.Info("Wallet initialized", "identityKey", privKey.PubKey().ToDERHex())
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

// ToolboxNetwork returns the configured network for wallet-toolbox SDK
func (p *Provider) ToolboxNetwork() defs.BSVNetwork {
	if p.bsvNetwork == "testnet" {
		return defs.NetworkTestnet
	}
	return defs.NetworkMainnet
}

// OverlayNetwork returns the configured network for overlay SDK
func (p *Provider) OverlayNetwork() overlay.Network {
	if p.bsvNetwork == "testnet" {
		return overlay.NetworkTestnet
	}
	return overlay.NetworkMainnet
}
