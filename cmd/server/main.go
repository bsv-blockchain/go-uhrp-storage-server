package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/config"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/pricing"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
)

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
	calc := pricing.NewCalculator(cfg.PricePerGBMonth)
	store := storage.NewFileStore(publicDir)
	wp := walletpkg.NewProvider(cfg.ServerPrivateKey, cfg.WalletStorageURL, cfg.BSVNetwork)

	// Initialize wallet (when wallet-toolbox is available)
	// For now the wallet is nil; endpoints requiring wallet will return appropriate errors.

	mimeMiddleware := &handlers.MimeTypeMiddleware{CDNPath: store.CDNPath()}

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware)

	// CDN MIME type middleware + static files
	r.Use(mimeMiddleware.Handle)
	fileServer := http.FileServer(http.Dir(publicDir))
	r.Handle("/favicon.ico", fileServer)
	r.Handle("/cdn/*", fileServer)

	// === Pre-auth routes (no auth/payment required) ===

	// PUT /put - file upload via presigned URL
	r.Put("/put", (&handlers.PutHandler{
		Store:          store,
		WalletProvider: wp,
		HostingDomain:  cfg.HostingDomain,
	}).ServeHTTP)

	// POST /quote - get storage price quote
	r.Post("/quote", (&handlers.QuoteHandler{
		Calculator:        calc,
		MinHostingMinutes: cfg.MinHostingMinutes,
	}).ServeHTTP)

	// === Post-auth routes (require auth + payment middleware) ===
	// In a full implementation, these would be wrapped with auth and payment middleware:
	//   authMiddleware := middleware.NewAuth(wallet, middleware.WithAuthAllowUnauthenticated())
	//   paymentMiddleware := middleware.NewPayment(wallet, middleware.WithRequestPriceCalculator(...))
	//   r.Group(func(r chi.Router) {
	//       r.Use(authMiddleware.HTTPHandler)
	//       r.Use(paymentMiddleware.HTTPHandler)
	//       ... routes ...
	//   })

	// For now, register them directly (auth middleware requires a non-nil wallet)
	r.Post("/upload", (&handlers.UploadHandler{
		Calculator:        calc,
		WalletProvider:    wp,
		HostingDomain:     cfg.HostingDomain,
		MinHostingMinutes: cfg.MinHostingMinutes,
	}).ServeHTTP)

	r.Get("/list", (&handlers.ListHandler{
		WalletProvider: wp,
	}).ServeHTTP)

	r.Post("/renew", (&handlers.RenewHandler{
		Calculator:     calc,
		WalletProvider: wp,
	}).ServeHTTP)

	r.Get("/find", (&handlers.FindHandler{
		WalletProvider: wp,
	}).ServeHTTP)

	// 404 handler
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"status":"error","code":"ERR_ROUTE_NOT_FOUND","description":"Route not found."}`)
	})

	// Log identity key
	if cfg.ServerPrivateKey != "" {
		privKey, err := ec.PrivateKeyFromHex(cfg.ServerPrivateKey)
		if err == nil {
			log.Printf("UHRP Host IdentityKey: %s", privKey.PubKey().ToDERHex())
		}
	}

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
	}

	log.Printf("UHRP Storage Server listening on port %s", cfg.HTTPPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
