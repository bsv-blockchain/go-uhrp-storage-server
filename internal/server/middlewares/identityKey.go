package middlewares

import (
	"context"
	"net/http"

	"github.com/bsv-blockchain/go-bsv-middleware/pkg/middleware"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
)

type contextKey string

const IdentityContextKey contextKey = "identityKey"

// RequireIdentityKey is a middleware that extracts the identity key from the request context
// (placed there by the Auth middleware) and validates it. If missing or unknown, it aborts.
// It also makes the raw ec.PublicKey easily available under a strongly typed context key.
func RequireIdentityKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identityKey, err := middleware.ShouldGetIdentity(r.Context())
		if err != nil || isUnknown(identityKey) {
			responses.WriteError(w, http.StatusBadRequest, "ERR_MISSING_IDENTITY_KEY", "Missing authfetch identityKey.")
			return
		}

		ctx := context.WithValue(r.Context(), IdentityContextKey, identityKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isUnknown(key *ec.PublicKey) bool {
	return key == nil || middleware.IsUnknownIdentity(key)
}

// GetIdentityKey is a helper to easily pull the validated identity key from the context inside a handler.
func GetIdentityKey(ctx context.Context) *ec.PublicKey {
	if key, ok := ctx.Value(IdentityContextKey).(*ec.PublicKey); ok {
		return key
	}
	return nil
}
