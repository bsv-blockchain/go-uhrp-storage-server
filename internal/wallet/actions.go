package wallet

import (
	"context"
	"fmt"
	"log"

	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/storage"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

// CreateAdParams holds the arguments needed to create a new UHRP advertisement Action.
type CreateAdParams struct {
	Hash          []byte
	URL           string
	ExpirySecs    int64
	ContentType   string
	ContentLength int64
	ObjectID      string
	Uploader      string
}

// CreateAdvertisement constructs the PushDrop script and executes a CreateAction wallet call to mint an advertisement.
func CreateAdvertisement(ctx context.Context, wallet sdkWallet.Interface, p CreateAdParams) error {
	lockingScript, err := buildPushDropScript(ctx, wallet, p)
	if err != nil {
		return fmt.Errorf("failed to build advertisement script: %w", err)
	}

	// TODO: check if this URL should be added or the actual one where the file is stored
	uhrpURL, err := storage.GetURLForHash(p.Hash)
	if err != nil {
		return fmt.Errorf("failed to get URL for hash: %w", err)
	}

	_, err = wallet.CreateAction(ctx, sdkWallet.CreateActionArgs{
		Description: "UHRP Content Availability Advertisement",
		Outputs: []sdkWallet.CreateActionOutput{
			{
				LockingScript:     lockingScript.Bytes(),
				Satoshis:          1,
				OutputDescription: "UHRP advertisement token",
				Basket:            "uhrp advertisements",
				Tags: []string{
					fmt.Sprintf("uhrp_url_%s", uhrpURL),
					fmt.Sprintf("object_identifier_%s", p.ObjectID),
					fmt.Sprintf("uploader_identity_key_%s", p.Uploader),
					fmt.Sprintf("expiry_time_%d", p.ExpirySecs),
					"name_file",
					fmt.Sprintf("content_type_%s", p.ContentType),
					fmt.Sprintf("size_%d", p.ContentLength),
				},
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
	lockingScript, err := buildPushDropScript(ctx, wallet, p)
	if err != nil {
		return fmt.Errorf("failed to build advertisement script: %w", err)
	}

	_, err = wallet.CreateAction(ctx, sdkWallet.CreateActionArgs{
		Description: fmt.Sprintf("Renew UHRP advertisement for %s", p.URL),
		Inputs: []sdkWallet.CreateActionInput{
			{
				Outpoint:         matchedOutput.Outpoint,
				InputDescription: "Redeem previous UHRP advertisement",
			},
		},
		Outputs: []sdkWallet.CreateActionOutput{
			{
				LockingScript:     lockingScript.Bytes(),
				Satoshis:          1,
				OutputDescription: "Renewed UHRP advertisement token",
				Basket:            "uhrp advertisements",
				Tags:              matchedOutput.Tags,
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

// buildPushDropScript builds a PushDrop-compatible locking script using the go-sdk template.
func buildPushDropScript(ctx context.Context, wallet sdkWallet.Interface, p CreateAdParams) (*script.Script, error) {
	pd := &pushdrop.PushDrop{
		Wallet: wallet,
	}

	pubKey, err := wallet.GetPublicKey(ctx, sdkWallet.GetPublicKeyArgs{IdentityKey: true}, "uhrp-server")
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Comply with the UHRP Protocol and TS version
	fields := [][]byte{
		// The identity Key of the storage host
		[]byte(pubKey.PublicKey.ToDERHex()),
		// The hash of the file
		p.Hash,
		// The URL of the file
		[]byte(p.URL),
		// The expiry time of the advertisement
		[]byte(fmt.Sprintf("%d", p.ExpirySecs)),
		// The size of the file
		[]byte(fmt.Sprintf("%d", p.ContentLength)),
	}

	protocolID := sdkWallet.Protocol{
		SecurityLevel: sdkWallet.SecurityLevelEveryAppAndCounterparty,
		Protocol:      "uhrp advertisement",
	}

	lockScript, err := pd.Lock(
		ctx,
		fields,
		protocolID,
		"1",
		sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeAnyone},
		true,
		true,
		pushdrop.LockBefore,
	)

	if err != nil {
		return nil, err
	}

	return lockScript, nil
}
