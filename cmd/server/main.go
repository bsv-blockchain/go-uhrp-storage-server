package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/joho/godotenv"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/config"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// @title UHRP Storage Server API
// @version 1.0
// @description The official UHRP Storage Server implementation in Go, allowing anyone to host their own bitcom-based public file CDN.
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey AuthfetchIdentity
// @in header
// @name Authorization
// @description Authentication via go-bsv-middleware using BRC-43 Authfetch.

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Determine public dir path
	_, filename, _, _ := runtime.Caller(0)
	publicDir := filepath.Join(filepath.Dir(filename), "..", "..", "public")

	// Initialize components
	exchangeRateProvider := pricing.NewWhatsOnChainProvider()
	calc := pricing.NewCalculator(cfg.PricePerGBMonth, exchangeRateProvider)
	store := storage.NewFileStore(publicDir)
	wp := walletpkg.NewProvider(cfg.ServerPrivateKey, cfg.WalletStorageURL, cfg.BSVNetwork)

	// Initialize wallet using wallet-toolbox
	if cfg.ServerPrivateKey != "" && cfg.WalletStorageURL != "" {
		if err := wp.InitWallet(context.Background()); err != nil {
			log.Printf("WARNING: Failed to initialize wallet: %v", err)
			log.Println("Endpoints requiring wallet will return errors until wallet is available.")
		}
	} else {
		log.Println("WARNING: SERVER_PRIVATE_KEY or WALLET_STORAGE_URL not set; wallet features disabled.")
	}

	// Log identity key
	if cfg.ServerPrivateKey != "" {
		privKey, err := ec.PrivateKeyFromHex(cfg.ServerPrivateKey)
		if err == nil {
			log.Printf("UHRP Host IdentityKey: %s", privKey.PubKey().ToDERHex())
		}
	}

	srv := server.New(cfg, calc, store, wp, publicDir)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
