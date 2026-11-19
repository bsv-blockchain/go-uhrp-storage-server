// Package wallet wraps the server's go-wallet-toolbox wallet: upload HMACs and
// the PushDrop UHRP advertisements that announce hosted files on the overlay network.
package wallet

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/bsv-blockchain/go-sdk/overlay"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/defs"
	wdkStorage "github.com/bsv-blockchain/go-wallet-toolbox/pkg/storage"
	toolbox "github.com/bsv-blockchain/go-wallet-toolbox/pkg/wallet"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/wdk"
)

// networkTestnet is the BSV_NETWORK value that selects testnet; any other value selects mainnet.
const networkTestnet = "testnet"

// Provider manages the wallet singleton.
type Provider struct {
	serverPrivateKey string
	walletStorageURL string
	bsvNetwork       string
	slapTrackers     []string
	wallet           sdkWallet.Interface
	mu               sync.Mutex
	Logger           *slog.Logger
}

// NewProvider creates a wallet provider.
func NewProvider(serverPrivateKey, walletStorageURL, bsvNetwork string, slapTrackers []string, logger *slog.Logger) *Provider {
	return &Provider{
		serverPrivateKey: serverPrivateKey,
		walletStorageURL: walletStorageURL,
		bsvNetwork:       bsvNetwork,
		slapTrackers:     slapTrackers,
		Logger:           logger.With("component", "wallet_provider"),
	}
}

// InitWallet initializes the wallet-toolbox wallet using the configured
// server private key and wallet storage URL. It must be called once at startup.
func (p *Provider) InitWallet(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.wallet != nil {
		return nil // already initialized
	}

	if p.serverPrivateKey == "" {
		return ErrMissingServerPrivateKey
	}

	chain := defs.NetworkMainnet
	if p.isTestnet() {
		chain = defs.NetworkTestnet
	}

	privKey, err := ec.PrivateKeyFromHex(p.serverPrivateKey)
	if err != nil {
		return fmt.Errorf("invalid server private key: %w", err)
	}

	// A toolbox.StorageProviderFactoryWithWalletReturningCleanupAndError: connects the wallet
	// to the remote wallet storage server.
	storageFactory := func(w sdkWallet.Interface) (wdk.WalletStorageProvider, func(), error) {
		return wdkStorage.NewClient(p.walletStorageURL, w)
	}

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

// OverlayNetwork returns the configured network for overlay SDK
func (p *Provider) OverlayNetwork() overlay.Network {
	if p.isTestnet() {
		return overlay.NetworkTestnet
	}

	return overlay.NetworkMainnet
}

// isTestnet reports whether the provider is configured for testnet.
func (p *Provider) isTestnet() bool {
	return p.bsvNetwork == networkTestnet
}
