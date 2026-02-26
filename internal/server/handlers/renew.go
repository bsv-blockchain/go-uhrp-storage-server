package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-bsv-middleware/pkg/middleware"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
)

// RenewHandler handles POST /renew requests.
type RenewHandler struct {
	Calculator     *pricing.Calculator
	WalletProvider *walletpkg.Provider
}

type renewRequest struct {
	UhrpURL           string `json:"uhrpUrl"`
	AdditionalMinutes int64  `json:"additionalMinutes"`
	Limit             *int   `json:"limit,omitempty"`
	Offset            *int   `json:"offset,omitempty"`
}

type renewResponse struct {
	Status         string `json:"status"`
	PrevExpiryTime int64  `json:"prevExpiryTime,omitempty"`
	NewExpiryTime  int64  `json:"newExpiryTime,omitempty"`
	Amount         int64  `json:"amount,omitempty"`
	Code           string `json:"code,omitempty"`
	Description    string `json:"description,omitempty"`
}

func (h *RenewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey, err := middleware.ShouldGetIdentity(r.Context())
	if err != nil || isUnknown(identityKey) {
		writeError(w, http.StatusBadRequest, "ERR_MISSING_IDENTITY_KEY", "Missing authfetch identityKey.")
		return
	}

	var req renewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_MISSING_FIELDS", "Invalid request body.")
		return
	}

	if req.UhrpURL == "" || req.AdditionalMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "ERR_MISSING_FIELDS", "Missing objectIdentifier or additionalMinutes.")
		return
	}

	wallet := h.WalletProvider.GetWallet()
	if wallet == nil {
		writeError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized.")
		return
	}

	// 1. Find the existing advertisement via ListOutputs
	includeCustom := true
	includeLocking := sdkWallet.OutputIncludeLockingScripts
	listResult, err := wallet.ListOutputs(r.Context(), sdkWallet.ListOutputsArgs{
		Basket:                    "uhrp advertisements",
		Include:                   includeLocking,
		IncludeCustomInstructions: &includeCustom,
	}, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW", "Failed to query wallet outputs.")
		return
	}

	// Find the matching output
	var matchIdx int = -1
	var meta map[string]string
	for i, out := range listResult.Outputs {
		m := parseCustomInstructions(out.CustomInstructions)
		if m != nil && m["uhrpURL"] == req.UhrpURL {
			matchIdx = i
			meta = m
			break
		}
	}

	if matchIdx < 0 {
		writeError(w, http.StatusNotFound, "ERR_NOT_FOUND", "No advertisement found for the given uhrpUrl.")
		return
	}

	matchedOutput := listResult.Outputs[matchIdx]

	// 2. Calculate pricing
	prevExpiry := parseExpiryTime(meta["expiryTime"])
	var fileSize int64
	fmt.Sscanf(meta["fileSize"], "%d", &fileSize)

	amount, err := h.Calculator.GetPrice(fileSize, req.AdditionalMinutes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW", "Failed to calculate price.")
		return
	}

	// 3. Compute new expiry
	newExpiry := prevExpiry + (req.AdditionalMinutes * 60)
	newExpiryStr := time.Unix(newExpiry, 0).UTC().Format(time.RFC3339)

	// 4. Build updated custom instructions and locking script
	meta["expiryTime"] = newExpiryStr
	updatedInstructions, _ := json.Marshal(meta)

	newScript, err := buildPushDropScript(
		meta["uhrpURL"], meta["hostingDomain"], newExpiryStr,
		meta["contentType"], meta["fileSize"], meta["objectID"], meta["uploader"],
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW", "Failed to build renewal script.")
		return
	}

	// 5. Redeem old output and create new one via CreateAction
	_, err = wallet.CreateAction(r.Context(), sdkWallet.CreateActionArgs{
		Description: fmt.Sprintf("Renew UHRP advertisement for %s", req.UhrpURL),
		Inputs: []sdkWallet.CreateActionInput{
			{
				Outpoint:         matchedOutput.Outpoint,
				InputDescription: "Redeem previous UHRP advertisement",
			},
		},
		Outputs: []sdkWallet.CreateActionOutput{
			{
				LockingScript:      newScript,
				Satoshis:           1,
				OutputDescription:  "Renewed UHRP advertisement token",
				Basket:             "uhrp advertisements",
				CustomInstructions: string(updatedInstructions),
				Tags:               matchedOutput.Tags,
			},
		},
		Labels: []string{"uhrp-advertisement", "uhrp-renewal"},
	}, "")
	if err != nil {
		log.Printf("Failed to renew UHRP advertisement: %v", err)
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW",
			"An error occurred while handling the renewal.")
		return
	}

	writeJSON(w, http.StatusOK, renewResponse{
		Status:         "success",
		PrevExpiryTime: prevExpiry,
		NewExpiryTime:  newExpiry,
		Amount:         amount,
	})
}
