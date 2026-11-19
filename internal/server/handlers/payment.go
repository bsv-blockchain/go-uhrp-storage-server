package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// ErrMissingIdentityKey is returned when a priced request reaches the payment middleware unauthenticated.
var ErrMissingIdentityKey = errors.New("identityKey not found in context")

// RequestPriceCalculator returns a function compatible with the go-bsv-middleware calculating the required price for a request.
//
// /upload is priced by its file size and retention period, and /renew by the advertised file's
// size and the additional minutes. Every other route, and a body that is not valid JSON, costs
// nothing here; the route's handler then rejects an invalid body itself.
func RequestPriceCalculator(calc *pricing.Calculator, wp *wallet.Provider) func(req *http.Request) (int, error) {
	return func(req *http.Request) (int, error) {
		if req.Body == nil {
			return 0, nil
		}

		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return 0, fmt.Errorf("read request body: %w", err)
		}

		// Restore the body for the handler that runs after the payment middleware.
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		switch {
		case strings.Contains(req.URL.Path, "/upload"):
			return uploadPrice(req.Context(), calc, bodyBytes)
		case strings.Contains(req.URL.Path, "/renew"):
			return renewPrice(req.Context(), calc, wp, bodyBytes)
		default:
			return 0, nil
		}
	}
}

// uploadPrice prices an /upload request body.
func uploadPrice(ctx context.Context, calc *pricing.Calculator, body []byte) (int, error) {
	var payload UploadRequest
	if !decodeJSON(body, &payload) {
		return 0, nil // free here; the upload handler rejects the body with 400
	}

	price, err := calc.GetPrice(ctx, payload.FileSize, payload.RetentionPeriod)
	if err != nil {
		return 0, fmt.Errorf("calc.GetPrice: %w", err)
	}

	return int(price), nil
}

// renewPrice prices a /renew request body from the size of the caller's existing advertisement.
func renewPrice(ctx context.Context, calc *pricing.Calculator, wp *wallet.Provider, body []byte) (int, error) {
	identityKey := middlewares.GetIdentityKey(ctx)
	if identityKey == nil {
		return 0, ErrMissingIdentityKey
	}

	var payload RenewRequest
	if !decodeJSON(body, &payload) {
		return 0, nil // free here; the renew handler rejects the body with 400
	}

	fileSize, err := wp.GetFileSize(ctx, payload.UhrpURL, identityKey.ToDERHex())
	if err != nil {
		return 0, err
	}

	price, err := calc.GetPrice(ctx, fileSize, payload.AdditionalMinutes)
	if err != nil {
		return 0, fmt.Errorf("calc.GetPrice: %w", err)
	}

	return int(price), nil
}

// decodeJSON reports whether body is valid JSON for v, decoding it into v.
func decodeJSON(body []byte, v any) bool {
	return json.Unmarshal(body, v) == nil
}
