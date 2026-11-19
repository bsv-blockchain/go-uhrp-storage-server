// Package main starts the UHRP storage server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/joho/godotenv"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/config"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/logger"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// @title UHRP Storage Server API
// @version 1.0
// @description The official UHRP Storage Server implementation in Go, allowing anyone to host their own public file CDN.
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey AuthfetchIdentity
// @in header
// @name Authorization
// @description Authentication via go-bsv-middleware using BRC-43 Authfetch.
func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

// run loads the configuration, initializes the wallet and serves HTTP until the server stops.
func run(ctx context.Context) error {
	_ = godotenv.Load() // a .env file is optional; the environment always wins

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	baseLogger := logger.Configure(cfg.LogLevel, cfg.LogFormat)

	publicDir, err := filepath.Abs(cfg.PublicDir)
	if err != nil {
		return fmt.Errorf("resolve PUBLIC_DIR %q: %w", cfg.PublicDir, err)
	}

	store, err := storage.NewFileStore(publicDir)
	if err != nil {
		return fmt.Errorf("open file store: %w", err)
	}

	defer func() { _ = store.Close() }()

	slog.Info("Serving uploaded files", "cdnDir", store.CDNPath())

	calc := pricing.NewCalculator(cfg.PricePerGBMonth, pricing.NewWhatsOnChainProvider())
	wp := walletpkg.NewProvider(cfg.ServerPrivateKey, cfg.WalletStorageURL, cfg.BSVNetwork, cfg.SLAPTrackers, baseLogger)

	initWallet(ctx, cfg, wp)

	return server.New(cfg, calc, store, wp, publicDir, baseLogger).Start()
}

// initWallet initializes the server wallet when it is configured. Without it the server
// still starts: /quote and the CDN work, and the wallet-backed routes are not mounted.
func initWallet(ctx context.Context, cfg *config.Config, wp *walletpkg.Provider) {
	if cfg.ServerPrivateKey == "" || cfg.WalletStorageURL == "" {
		slog.Info("SERVER_PRIVATE_KEY or WALLET_STORAGE_URL not set; wallet features disabled.")
	} else if err := wp.InitWallet(ctx); err != nil {
		slog.Warn("Failed to initialize wallet", "error", err)
		slog.Info("Wallet routes (/upload, /list, /renew, /find) are not mounted; restart once the wallet is reachable.")
	}

	if cfg.ServerPrivateKey == "" {
		return
	}

	if privKey, err := ec.PrivateKeyFromHex(cfg.ServerPrivateKey); err == nil {
		slog.Info("UHRP Host IdentityKey", "identityKey", privKey.PubKey().ToDERHex())
	}
}
