package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/pricing"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
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
			var payload uploadRequest
			if err := json.Unmarshal(bodyBytes, &payload); err == nil {
				price, err := calc.GetPrice(payload.FileSize, payload.RetentionPeriod)
				if err != nil {
					return 0, fmt.Errorf("calc.GetPrice: %w", err)
				}
				return int(price), nil
			}
		} else if strings.Contains(req.URL.Path, "/renew") {
			var payload renewRequest
			if err := json.Unmarshal(bodyBytes, &payload); err == nil {
				wallet := wp.GetWallet()
				if wallet == nil {
					return 0, fmt.Errorf("wallet not available for renew price calculation")
				}

				fileSize, i, err1, shouldReturn := getFileSize(req.Context(), wallet, payload.UhrpURL)
				if shouldReturn {
					return i, err1
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

func getFileSize(ctx context.Context, wallet sdkWallet.Interface, uhrpURL string) (int64, int, error, bool) {
	includeCustom := true
	includeTags := true
	includeLocking := sdkWallet.OutputIncludeLockingScripts
	listResult, err := wallet.ListOutputs(ctx, sdkWallet.ListOutputsArgs{
		Basket:                    "uhrp advertisements",
		Include:                   includeLocking,
		IncludeCustomInstructions: &includeCustom,
		IncludeTags:               &includeTags,
		Tags:                      []string{fmt.Sprintf("uhrpUrl_%s", uhrpURL)},
	}, "")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query wallet outputs: %w", err), true
	}

	var fileSize int64
	matchFound := false
	for _, out := range listResult.Outputs {
		m := parseCustomInstructions(out.CustomInstructions)
		if m != nil && m["uhrpURL"] == uhrpURL {
			fmt.Sscanf(m["fileSize"], "%d", &fileSize)
			matchFound = true
			break
		}
	}

	if !matchFound {
		return 0, 0, fmt.Errorf("uhrpUrl not found in wallet outputs"), true
	}
	return fileSize, 0, nil, false
}
