package wallet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/bsv-blockchain/go-sdk/script"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

// CreateAdParams holds the arguments needed to create a new UHRP advertisement Action.
type CreateAdParams struct {
	UhrpURL       string
	HostingDomain string
	ExpirySecs    int64
	ContentType   string
	FileSize      int64
	ObjectID      string
	Uploader      string
}

// VerifyUploaderHMAC verifies that the uploader provided a valid HMAC over the file metadata.
func VerifyUploaderHMAC(ctx context.Context, wallet sdkWallet.Interface, fileSizeStr, objectID, expiry, uploader, hmacHex string) error {
	str := fmt.Sprintf("fileSize=%s&objectID=%s&expiry=%s&uploader=%s", fileSizeStr, objectID, expiry, uploader)

	hmacBytes, err := hex.DecodeString(hmacHex)
	if err != nil || len(hmacBytes) != 32 {
		return fmt.Errorf("invalid HMAC format")
	}
	var hmacArr [32]byte
	copy(hmacArr[:], hmacBytes)

	verifyResult, err := wallet.VerifyHMAC(ctx, sdkWallet.VerifyHMACArgs{
		EncryptionArgs: sdkWallet.EncryptionArgs{
			ProtocolID: sdkWallet.Protocol{
				SecurityLevel: sdkWallet.SecurityLevelEveryAppAndCounterparty,
				Protocol:      "uhrp file hosting",
			},
			KeyID:        objectID,
			Counterparty: sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeSelf},
		},
		Data: []byte(str),
		HMAC: hmacArr,
	}, "")
	if err != nil {
		return fmt.Errorf("HMAC verification failed: %w", err)
	}
	if !verifyResult.Valid {
		return fmt.Errorf("invalid HMAC")
	}

	return nil
}

// CreateAdvertisement constructs the PushDrop script and executes a CreateAction wallet call to mint an advertisement.
func CreateAdvertisement(ctx context.Context, wallet sdkWallet.Interface, p CreateAdParams) error {
	expiryStr := fmt.Sprintf("%d", p.ExpirySecs)
	fileSizeStr := fmt.Sprintf("%d", p.FileSize)

	lockingScript, err := buildPushDropScript(p.UhrpURL, p.HostingDomain, expiryStr, p.ContentType, fileSizeStr, p.ObjectID, p.Uploader)
	if err != nil {
		return fmt.Errorf("failed to build advertisement script: %w", err)
	}

	customInstructions := map[string]string{
		"uhrpURL":       p.UhrpURL,
		"hostingDomain": p.HostingDomain,
		"expiryTime":    expiryStr,
		"contentType":   p.ContentType,
		"fileSize":      fileSizeStr,
		"objectID":      p.ObjectID,
		"uploader":      p.Uploader,
	}
	b, _ := json.Marshal(customInstructions)

	_, err = wallet.CreateAction(ctx, sdkWallet.CreateActionArgs{
		Description: fmt.Sprintf("UHRP advertisement for %s", p.UhrpURL),
		Outputs: []sdkWallet.CreateActionOutput{
			{
				LockingScript:      lockingScript,
				Satoshis:           1,
				OutputDescription:  "UHRP advertisement token",
				Basket:             "uhrp advertisements",
				CustomInstructions: string(b),
				Tags:               []string{"uhrp-ad", fmt.Sprintf("uploader-%s", p.Uploader)},
			},
		},
		Labels: []string{"uhrp-advertisement"},
	}, "")
	if err != nil {
		log.Printf("Failed to create UHRP advertisement: %v", err)
		return fmt.Errorf("failed to broadcast advertisement: %w", err)
	}
	return nil
}

// RenewAdvertisement consumes an existing advertisement output and creates a new one with the updated script.
func RenewAdvertisement(ctx context.Context, wallet sdkWallet.Interface, matchedOutput sdkWallet.Output, p CreateAdParams) error {
	expiryStr := fmt.Sprintf("%d", p.ExpirySecs)
	fileSizeStr := fmt.Sprintf("%d", p.FileSize)

	lockingScript, err := buildPushDropScript(p.UhrpURL, p.HostingDomain, expiryStr, p.ContentType, fileSizeStr, p.ObjectID, p.Uploader)
	if err != nil {
		return fmt.Errorf("failed to build advertisement script: %w", err)
	}

	customInstructions := map[string]string{
		"uhrpURL":       p.UhrpURL,
		"hostingDomain": p.HostingDomain,
		"expiryTime":    expiryStr,
		"contentType":   p.ContentType,
		"fileSize":      fileSizeStr,
		"objectID":      p.ObjectID,
		"uploader":      p.Uploader,
	}
	b, _ := json.Marshal(customInstructions)

	_, err = wallet.CreateAction(ctx, sdkWallet.CreateActionArgs{
		Description: fmt.Sprintf("Renew UHRP advertisement for %s", p.UhrpURL),
		Inputs: []sdkWallet.CreateActionInput{
			{
				Outpoint:         matchedOutput.Outpoint,
				InputDescription: "Redeem previous UHRP advertisement",
			},
		},
		Outputs: []sdkWallet.CreateActionOutput{
			{
				LockingScript:      lockingScript,
				Satoshis:           1,
				OutputDescription:  "Renewed UHRP advertisement token",
				Basket:             "uhrp advertisements",
				CustomInstructions: string(b),
				Tags:               matchedOutput.Tags,
			},
		},
		Labels: []string{"uhrp-advertisement", "uhrp-renewal"},
	}, "")
	if err != nil {
		log.Printf("Failed to renew UHRP advertisement: %v", err)
		return fmt.Errorf("error occurred while handling the renewal: %w", err)
	}
	return nil
}

// buildPushDropScript builds an OP_RETURN-based script with push data fields (original implementation).
func buildPushDropScript(uhrpURL, hostingDomain, expiry, contentType, fileSize, objectID, uploader string) ([]byte, error) {
	s := &script.Script{}
	if err := s.AppendOpcodes(script.OpFALSE, script.OpRETURN); err != nil {
		return nil, err
	}
	for _, field := range []string{uhrpURL, hostingDomain, expiry, contentType, fileSize, objectID, uploader} {
		if err := s.AppendPushDataString(field); err != nil {
			return nil, err
		}
	}
	return *s, nil
}
