package wallet

import (
	"context"
	"encoding/hex"
	"fmt"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

const (
	// AnyonesKeyID is the key ID used for upload HMACs and advertisement PushDrop keys.
	AnyonesKeyID = "1"

	// StorageUploadProtocol is the wallet protocol name used for upload HMACs.
	StorageUploadProtocol = "storage upload"

	// hmacSize is the length in bytes of a wallet HMAC (SHA-256).
	hmacSize = 32
)

// CreateUploaderHMAC creates an HMAC for the given string using the wallet.
func (wp *Provider) CreateUploaderHMAC(ctx context.Context, queryStr string) (string, error) {
	wallet := wp.GetWallet()
	if wallet == nil {
		return "", ErrWalletNotInitialized
	}

	hmacResult, err := wallet.CreateHMAC(ctx, sdkWallet.CreateHMACArgs{
		EncryptionArgs: uploadEncryptionArgs(),
		Data:           []byte(queryStr),
	}, "")
	if err != nil {
		return "", fmt.Errorf("failed to create HMAC: %w", err)
	}

	return hex.EncodeToString(hmacResult.HMAC[:]), nil
}

// VerifyUploaderHMAC verifies that the uploader provided a valid HMAC over the file metadata.
// It returns ErrInvalidHMACFormat or ErrInvalidHMAC when the HMAC is malformed or does not match.
func (wp *Provider) VerifyUploaderHMAC(ctx context.Context, fileSizeStr, objectID, expiry, uploader, hmacHex string) error {
	wallet := wp.GetWallet()
	if wallet == nil {
		return ErrWalletNotInitialized
	}

	hmacBytes, err := hex.DecodeString(hmacHex)
	if err != nil || len(hmacBytes) != hmacSize {
		return ErrInvalidHMACFormat
	}

	var hmacArr [hmacSize]byte

	copy(hmacArr[:], hmacBytes)

	str := fmt.Sprintf("fileSize=%s&objectID=%s&expiry=%s&uploader=%s", fileSizeStr, objectID, expiry, uploader)

	verifyResult, err := wallet.VerifyHMAC(ctx, sdkWallet.VerifyHMACArgs{
		EncryptionArgs: uploadEncryptionArgs(),
		Data:           []byte(str),
		HMAC:           hmacArr,
	}, "")
	if err != nil {
		return fmt.Errorf("HMAC verification failed: %w", err)
	}

	if !verifyResult.Valid {
		return ErrInvalidHMAC
	}

	return nil
}

// uploadEncryptionArgs returns the wallet key derivation used for upload HMACs:
// the server's own key under the storage upload protocol.
func uploadEncryptionArgs() sdkWallet.EncryptionArgs {
	return sdkWallet.EncryptionArgs{
		ProtocolID: sdkWallet.Protocol{
			SecurityLevel: sdkWallet.SecurityLevelEveryAppAndCounterparty,
			Protocol:      StorageUploadProtocol,
		},
		KeyID:        AnyonesKeyID,
		Counterparty: sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeSelf},
	}
}
