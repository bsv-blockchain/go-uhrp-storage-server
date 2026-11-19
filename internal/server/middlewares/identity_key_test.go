package middlewares_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
)

func TestGetIdentityKey(t *testing.T) {
	priv, err := ec.NewPrivateKey()
	require.NoError(t, err)

	pub := priv.PubKey()

	assert.Nil(t, middlewares.GetIdentityKey(t.Context()))
	assert.Nil(t, middlewares.GetIdentityKey(context.WithValue(t.Context(), middlewares.IdentityContextKey, "not a key")))
	assert.Same(t, pub, middlewares.GetIdentityKey(context.WithValue(t.Context(), middlewares.IdentityContextKey, pub)))
}

func TestRequireIdentityKeyRejectsUnauthenticatedRequests(t *testing.T) {
	called := false
	handler := middlewares.RequireIdentityKey(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/list", nil))

	assert.False(t, called, "the next handler must not run without an identity key")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp responses.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ERR_MISSING_IDENTITY_KEY", resp.Code)
}
