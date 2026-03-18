package wallet

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/topic"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/storage"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
	"github.com/bsv-blockchain/go-sdk/util"
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
func CreateAdvertisement(ctx context.Context, wallet sdkWallet.Interface, network overlay.Network, p CreateAdParams) error {
	lockingScript, err := buildPushDropScript(ctx, wallet, p)
	if err != nil {
		return fmt.Errorf("failed to build advertisement script: %w", err)
	}

	uhrpURL, err := storage.GetURLForHash(p.Hash)
	if err != nil {
		return fmt.Errorf("failed to get URL for hash: %w", err)
	}

	result, err := wallet.CreateAction(ctx, sdkWallet.CreateActionArgs{
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
					"name_file",
					fmt.Sprintf("content_type_%s", p.ContentType),
					fmt.Sprintf("size_%d", p.ContentLength),
				},
			},
		},
		Labels: []string{BaseAdvertisementLabel},
		Options: &sdkWallet.CreateActionOptions{
			RandomizeOutputs: util.BoolPtr(false),
		},
	}, "")
	if err != nil {
		return fmt.Errorf("failed to broadcast advertisement: %w", err)
	}

	err = overlayBroadcast(result.Tx, network)
	if err != nil {
		return err
	}

	return nil
}

// RenewAdvertisement consumes an existing advertisement output and creates a new one with the updated script.
func RenewAdvertisement(ctx context.Context, wallet sdkWallet.Interface, network overlay.Network, matchedOutput *sdkWallet.Output, beef []byte, p CreateAdParams) error {
	if matchedOutput == nil {
		return fmt.Errorf("no matched output found")
	}

	lockingScript, err := decodeAndBuildPushDropLockingScript(ctx, wallet, matchedOutput, p)
	if err != nil {
		return fmt.Errorf("failed to build advertisement script: %w", err)
	}

	aResult, err := wallet.CreateAction(ctx, sdkWallet.CreateActionArgs{
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
				OutputDescription: "UHRP advertisement token (renewed)",
				Basket:            BasketName,
				Tags: []string{
					fmt.Sprintf("uhrp_url_%s", hex.EncodeToString([]byte(p.URL))),
					fmt.Sprintf("object_identifier_%s", hex.EncodeToString([]byte(p.ObjectID))),
					fmt.Sprintf("uploader_identity_key_%s", p.Uploader),
					fmt.Sprintf("expiry_time_%d", p.ExpirySecs),
					"name_file",
					fmt.Sprintf("content_type_%s", p.ContentType),
					fmt.Sprintf("size_%d", p.ContentLength),
				},
			},
		},
		Labels: []string{BaseAdvertisementLabel, RenewalLabel},
		Options: &sdkWallet.CreateActionOptions{
			RandomizeOutputs: util.BoolPtr(false),
		},
	}, "")
	if err != nil {
		return fmt.Errorf("error occurred while handling the renewal: %w", err)
	}

	unlockingScript, inputIndex, err := buildPushDropUnlockingScript(ctx, wallet, matchedOutput, aResult)
	if err != nil {
		return fmt.Errorf("failed to build unlocking script: %w", err)
	}

	sResult, err := wallet.SignAction(
		ctx,
		sdkWallet.SignActionArgs{
			Reference: aResult.SignableTransaction.Reference,
			Spends: map[uint32]sdkWallet.SignActionSpend{
				inputIndex: {
					UnlockingScript: unlockingScript.Bytes(),
				},
			},
		},
		WalletName,
	)
	if err != nil {
		return fmt.Errorf("error occurred while handling the renewal: %w", err)
	}

	err = overlayBroadcast(sResult.Tx, network)
	if err != nil {
		return err
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
		pubKey.PublicKey.Compressed(),
		p.Hash,
		[]byte(p.URL),
		util.VarInt(uint64(p.ExpirySecs)).Bytes(),
		util.VarInt(uint64(p.ContentLength)).Bytes(),
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

func buildPushDropUnlockingScript(ctx context.Context, wallet sdkWallet.Interface, matchedOutput *sdkWallet.Output, result *sdkWallet.CreateActionResult) (*script.Script, uint32, error) {
	pd := &pushdrop.PushDrop{
		Wallet: wallet,
	}

	lockingScript := script.NewFromBytes(matchedOutput.LockingScript)

	opts := pushdrop.UnlockOptions{
		SourceSatoshis: &matchedOutput.Satoshis,
		LockingScript:  lockingScript,
	}

	unlocker := pd.Unlock(
		ctx,
		Protocol,
		AnyonesKeyID,
		sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeAnyone},
		sdkWallet.SignOutputsAll,
		false,
		opts,
	)

	txBeef, txHash, err := transaction.NewBeefFromAtomicBytes(result.SignableTransaction.Tx)
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing signable transaction: %w", err)
	}

	tx := txBeef.FindTransactionForSigningByHash(txHash)
	if tx == nil {
		return nil, 0, fmt.Errorf("transaction not found in BEEF")
	}

	inputIndex := -1
	for i, input := range tx.Inputs {
		outpointStr := fmt.Sprintf("%s.%d", input.SourceTXID.String(), input.SourceTxOutIndex)
		if outpointStr == matchedOutput.Outpoint.String() {
			inputIndex = i
			break
		}
	}

	if inputIndex == -1 {
		return nil, 0, fmt.Errorf("could not find input matching outpoint %s in signable transaction", matchedOutput.Outpoint.String())
	}

	tx.Inputs[inputIndex].SetSourceTxOutput(&transaction.TransactionOutput{
		Satoshis:      matchedOutput.Satoshis,
		LockingScript: script.NewFromBytes(matchedOutput.LockingScript),
	})

	unlockingScript, err := unlocker.Sign(tx, inputIndex)
	if err != nil {
		return nil, 0, fmt.Errorf("error unlocking funding input: %w", err)
	}

	return unlockingScript, uint32(inputIndex), nil
}

func decodeAndBuildPushDropLockingScript(ctx context.Context, wallet sdkWallet.Interface, matchedOutput *sdkWallet.Output, p CreateAdParams) (*script.Script, error) {
	pd := &pushdrop.PushDrop{
		Wallet: wallet,
	}

	prevLockingScript := pushdrop.Decode((*script.Script)(&matchedOutput.LockingScript))
	if prevLockingScript == nil || len(prevLockingScript.Fields) < 5 {
		return nil, fmt.Errorf("invalid or missing pushdrop locking script")
	}

	fields := [][]byte{
		prevLockingScript.Fields[0],
		prevLockingScript.Fields[1],
		prevLockingScript.Fields[2],
		util.VarInt(uint64(p.ExpirySecs)).Bytes(),
		prevLockingScript.Fields[4],
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

func overlayBroadcast(tx []byte, network overlay.Network) error {
	broadcaster, err := topic.NewBroadcaster([]string{"tm_uhrp"}, &topic.BroadcasterConfig{
		NetworkPreset: network,
	})
	if err != nil {
		return fmt.Errorf("failed to create topic broadcaster: %w", err)
	}

	newBeef, newTxHash, err := transaction.NewBeefFromAtomicBytes(tx)
	if err != nil {
		return fmt.Errorf("error parsing signed transaction bytes into BEEF: %w", err)
	}

	newTx := newBeef.FindAtomicTransactionByHash(newTxHash)
	fmt.Println("Parsed signed transaction from BEEF", "version", newTx.Version, "inputs", len(newTx.Inputs), "outputs", len(newTx.Outputs))

	success, failure := broadcaster.Broadcast(newTx)
	if failure != nil {
		return fmt.Errorf("error occurred while handling the broadcasting: %w", failure)
	}

	fmt.Println("Success: ", success)

	return nil
}
