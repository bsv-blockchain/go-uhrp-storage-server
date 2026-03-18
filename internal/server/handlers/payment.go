package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// RequestPriceCalculator returns a function compatible with the go-bsv-middleware calculating the required price for a request.
func RequestPriceCalculator(calc *pricing.Calculator, wp *wallet.Provider) func(req *http.Request) (int, error) {
	return func(req *http.Request) (int, error) {
		if req.Body == nil {
			return 0, nil
		}

		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return 0, err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if strings.Contains(req.URL.Path, "/upload") {
			var payload UploadRequest
			if err := json.Unmarshal(bodyBytes, &payload); err == nil {
				price, err := calc.GetPrice(payload.FileSize, payload.RetentionPeriod)
				if err != nil {
					return 0, fmt.Errorf("calc.GetPrice: %w", err)
				}
				return int(price), nil
			}
		} else if strings.Contains(req.URL.Path, "/renew") {
			identityKey := middlewares.GetIdentityKey(req.Context())
			if identityKey == nil {
				return 0, fmt.Errorf("identityKey not found in context")
			}

			var payload RenewRequest
			if err := json.Unmarshal(bodyBytes, &payload); err == nil {
				fileSize, err := wp.GetFileSize(req.Context(), payload.UhrpURL, identityKey.ToDERHex())
				if err != nil {
					return 0, err
				}

				price, err := calc.GetPrice(fileSize, payload.AdditionalMinutes)
				if err != nil {
					return 0, fmt.Errorf("calc.GetPrice: %w", err)
				}
				return int(price), nil
			}
		}

		// Defaults to 0 if we can't calculate it or the route doesn't require payment (e.g., /list, /find)
		return 0, nil
	}
}
