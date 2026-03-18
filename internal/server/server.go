package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bsv-blockchain/go-bsv-middleware/pkg/middleware"
	"github.com/bsv-blockchain/go-sdk/auth"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/config"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"

	_ "github.com/bsv-blockchain/go-uhrp-storage-server/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Server represents the UHRP storage HTTP server
type Server struct {
	HTTPServer *http.Server
	Logger     *slog.Logger
}

// New creates and configures a new Server instance
func New(cfg *config.Config, calc *pricing.Calculator, store *storage.FileStore, wp *wallet.Provider, publicDir string, logger *slog.Logger) *Server {
	logger = logger.With("component", "server")
	mimeMiddleware := &middlewares.MimeTypeMiddleware{CDNPath: store.CDNPath()}

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

	// Swagger Docs
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Pre-auth routes (no auth/payment required)
	registerPreAuthRoutes(cfg, r, store, wp, calc, logger)

	// Post-auth routes (require auth + payment middleware)
	registerPostAuthRoutes(wp, calc, r, cfg, logger)

	// 404 handler
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"status":"error","code":"ERR_ROUTE_NOT_FOUND","description":"Route not found."}`)
	})

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
	}

	return &Server{
		HTTPServer: srv,
		Logger:     logger,
	}
}

// Start begins listening and serving HTTP traffic
func (s *Server) Start() error {
	s.Logger.Info("UHRP Storage Server listening", "addr", s.HTTPServer.Addr)
	return s.HTTPServer.ListenAndServe()
}

func registerPostAuthRoutes(wp *wallet.Provider, calc *pricing.Calculator, r *chi.Mux, cfg *config.Config, logger *slog.Logger) {
	if wp.GetWallet() == nil {
		return
	}

	sessionManager := auth.NewSessionManager()
	authMiddleware := middleware.NewAuth(wp.GetWallet(), middleware.WithAuthSessionManager(sessionManager))
	paymentMiddleware := middleware.NewPayment(wp.GetWallet(), middleware.WithRequestPriceCalculator(
		handlers.RequestPriceCalculator(calc, wp),
	))

	r.Handle("/.well-known/auth", authMiddleware.HTTPHandler(http.NotFoundHandler()))

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.HTTPHandler)
		r.Use(middlewares.RequireIdentityKey)
		r.Use(paymentMiddleware.HTTPHandler)

		r.Post("/upload", (&handlers.UploadHandler{
			Calculator:        calc,
			WalletProvider:    wp,
			HostingDomain:     cfg.HostingDomain,
			MinHostingMinutes: cfg.MinHostingMinutes,
			Logger:            logger,
		}).ServeHTTP)

		r.Get("/list", (&handlers.ListHandler{
			WalletProvider: wp,
			Logger:         logger,
		}).ServeHTTP)

		r.Post("/renew", (&handlers.RenewHandler{
			Calculator:     calc,
			WalletProvider: wp,
			Logger:         logger,
		}).ServeHTTP)

		r.Get("/find", (&handlers.FindHandler{
			WalletProvider: wp,
			Logger:         logger,
		}).ServeHTTP)
	})
}

func registerPreAuthRoutes(cfg *config.Config, r *chi.Mux, store *storage.FileStore, wp *wallet.Provider, calc *pricing.Calculator, logger *slog.Logger) {
	// PUT /put - file upload via presigned URL
	r.Put("/put", (&handlers.PutHandler{
		Store:          store,
		WalletProvider: wp,
		HostingDomain:  cfg.HostingDomain,
		Logger:         logger,
	}).ServeHTTP)

	// POST /quote - get storage price quote
	r.Post("/quote", (&handlers.QuoteHandler{
		Calculator:        calc,
		MinHostingMinutes: cfg.MinHostingMinutes,
		Logger:            logger,
	}).ServeHTTP)
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
