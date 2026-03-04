package wallet

import (
	"context"
	"encoding/hex"
	"fmt"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

const (
	AnyonesKeyID          = "1"
	StorageUploadProtocol = "storage upload"
)

// CreateUploaderHMAC creates an HMAC for the given string using the wallet.
func CreateUploaderHMAC(ctx context.Context, wallet sdkWallet.Interface, queryStr string) (string, error) {
	hmacResult, err := wallet.CreateHMAC(ctx, sdkWallet.CreateHMACArgs{
		EncryptionArgs: sdkWallet.EncryptionArgs{
			ProtocolID: sdkWallet.Protocol{
				SecurityLevel: sdkWallet.SecurityLevelEveryAppAndCounterparty,
				Protocol:      StorageUploadProtocol,
			},
			KeyID:        AnyonesKeyID,
			Counterparty: sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeSelf},
		},
		Data: []byte(queryStr),
	}, "")
	if err != nil {
		return "", fmt.Errorf("failed to create HMAC: %w", err)
	}
	return hex.EncodeToString(hmacResult.HMAC[:]), nil
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
				Protocol:      StorageUploadProtocol,
			},
			KeyID:        AnyonesKeyID,
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
