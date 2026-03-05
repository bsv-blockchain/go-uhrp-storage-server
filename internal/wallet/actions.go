package wallet

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/storage"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

const (
	WalletName             = "uhrp-server"
	BasketName             = "uhrp advertisements"
	BaseAdvertisementLabel = "uhrp-advertisement"
	RenewalLabel           = "uhrp-renewal"
	ProtocolID             = "uhrp advertisement"
)

var Protocol = sdkWallet.Protocol{
	SecurityLevel: sdkWallet.SecurityLevelEveryAppAndCounterparty,
	Protocol:      ProtocolID,
}

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
				Basket:            BasketName,
				Tags: []string{
					fmt.Sprintf("uhrp_url_%s", hex.EncodeToString([]byte(uhrpURL))),
					fmt.Sprintf("object_identifier_%s", hex.EncodeToString([]byte(p.ObjectID))),
					fmt.Sprintf("uploader_identity_key_%s", p.Uploader),
					fmt.Sprintf("expiry_time_%d", p.ExpirySecs),
					// TODO: maybe actual name should be added as a tag
					"name_file",
					fmt.Sprintf("content_type_%s", p.ContentType),
					fmt.Sprintf("size_%d", p.ContentLength),
				},
			},
		},
		Labels: []string{BaseAdvertisementLabel},
	}, "")
	if err != nil {
		log.Printf("Failed to create UHRP advertisement: %v", err)
		return fmt.Errorf("failed to broadcast advertisement: %w", err)
	}

	// TODO: create SHIPBroadcaster to broadcast the transaction to the network

	return nil
}

// RenewAdvertisement consumes an existing advertisement output and creates a new one with the updated script.
func RenewAdvertisement(ctx context.Context, wallet sdkWallet.Interface, matchedOutput *sdkWallet.Output, beef []byte, p CreateAdParams) error {
	if matchedOutput == nil {
		return fmt.Errorf("no matched output found")
	}

	lockingScript, err := buildPushDropScript(ctx, wallet, p)
	if err != nil {
		return fmt.Errorf("failed to build advertisement script: %w", err)
	}

	result, err := wallet.CreateAction(ctx, sdkWallet.CreateActionArgs{
		InputBEEF:   beef,
		Description: fmt.Sprintf("Renew UHRP advertisement for %s", p.URL),
		Inputs: []sdkWallet.CreateActionInput{
			{
				Outpoint:              matchedOutput.Outpoint,
				InputDescription:      "Redeem previous UHRP advertisement",
				UnlockingScriptLength: 74,
			},
		},
		Outputs: []sdkWallet.CreateActionOutput{
			{
				LockingScript:     lockingScript.Bytes(),
				Satoshis:          1,
				OutputDescription: "Renewed UHRP advertisement token",
				Basket:            BasketName,
				Tags:              matchedOutput.Tags,
			},
		},
		Labels: []string{BaseAdvertisementLabel, RenewalLabel},
	}, "")
	if err != nil {
		log.Printf("Failed to renew UHRP advertisement: %v", err)
		return fmt.Errorf("error occurred while handling the renewal: %w", err)
	}

	unlockingScript, err := buildPushDropUnlockingScript(ctx, wallet, result)
	if err != nil {
		return fmt.Errorf("failed to build unlocking script: %w", err)
	}

	_, err = wallet.SignAction(
		ctx,
		sdkWallet.SignActionArgs{
			Reference: result.SignableTransaction.Reference,
			Spends: map[uint32]sdkWallet.SignActionSpend{
				0: {
					UnlockingScript: unlockingScript.Bytes(),
				},
			},
		},
		WalletName,
	)
	if err != nil {
		return fmt.Errorf("error occurred while handling the renewal: %w", err)
	}

	return nil
}

// buildPushDropScript builds a PushDrop-compatible locking script using the go-sdk template.
func buildPushDropScript(ctx context.Context, wallet sdkWallet.Interface, p CreateAdParams) (*script.Script, error) {
	pd := &pushdrop.PushDrop{
		Wallet: wallet,
	}

	pubKey, err := wallet.GetPublicKey(ctx, sdkWallet.GetPublicKeyArgs{IdentityKey: true}, WalletName)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	fields := [][]byte{
		[]byte(pubKey.PublicKey.ToDERHex()),
		p.Hash,
		[]byte(p.URL),
		[]byte(fmt.Sprintf("%d", p.ExpirySecs)),
		[]byte(fmt.Sprintf("%d", p.ContentLength)),
	}

	lockScript, err := pd.Lock(
		ctx,
		fields,
		Protocol,
		AnyonesKeyID,
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

func buildPushDropUnlockingScript(ctx context.Context, wallet sdkWallet.Interface, result *sdkWallet.CreateActionResult) (*script.Script, error) {
	pd := &pushdrop.PushDrop{
		Wallet: wallet,
	}

	unlocker := pd.Unlock(
		ctx,
		Protocol,
		AnyonesKeyID,
		sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeAnyone},
		sdkWallet.SignOutputsAll,
		false,
	)

	txBeef, txHash, err := transaction.NewBeefFromAtomicBytes(result.SignableTransaction.Tx)
	if err != nil {
		return nil, fmt.Errorf("error parsing signable transaction: %w", err)
	}

	tx := txBeef.FindAtomicTransactionByHash(txHash)

	unlockingScript, err := unlocker.Sign(tx, 0)
	if err != nil {
		return nil, fmt.Errorf("error unlocking funding input: %w", err)
	}

	return unlockingScript, nil
}
