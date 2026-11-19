package wallet

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/overlay/topic"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/storage"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
	"github.com/bsv-blockchain/go-sdk/util"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

const (
	// WalletName is the originator name the server uses for its own wallet calls.
	WalletName = "uhrp-server"

	// BasketName is the wallet basket holding the server's advertisement outputs.
	BasketName = "uhrp advertisements"

	// BaseAdvertisementLabel labels every advertisement action.
	BaseAdvertisementLabel = "uhrp-advertisement"

	// RenewalLabel additionally labels advertisement renewals.
	RenewalLabel = "uhrp-renewal"

	// ProtocolID is the wallet protocol name used for advertisement PushDrop keys.
	ProtocolID = "uhrp advertisement"

	// uhrpTopic is the overlay topic advertisements are broadcast to.
	uhrpTopic = "tm_uhrp"

	// pushDropUnlockingScriptLength is the estimated size of a PushDrop unlocking script (signature only).
	pushDropUnlockingScriptLength = 74

	// pushDropFieldCount is the number of data fields in an advertisement PushDrop script:
	// host identity key, content hash, URL, expiry and content length.
	pushDropFieldCount = 5

	// pushDropExpiryField is the index of the expiry field within the PushDrop fields.
	pushDropExpiryField = 3
)

// loggingLookupFacilitator wraps HTTPSOverlayLookupFacilitator and adds DEBUG logging.
type loggingLookupFacilitator struct {
	inner  *lookup.HTTPSOverlayLookupFacilitator
	logger *slog.Logger
}

func (f *loggingLookupFacilitator) Lookup(ctx context.Context, url string, question *lookup.LookupQuestion) (*lookup.LookupAnswer, error) {
	f.logger.Debug("Lookup query", "url", url, "service", question.Service)

	answer, err := f.inner.Lookup(ctx, url, question)
	if err != nil {
		f.logger.Debug("Lookup failed", "url", url, "service", question.Service, "error", err)
	} else {
		f.logger.Debug("Lookup success", "url", url, "service", question.Service, "type", answer.Type, "outputs", len(answer.Outputs))
	}

	return answer, err
}

// loggingBroadcastFacilitator wraps HTTPSOverlayBroadcastFacilitator and adds DEBUG logging.
type loggingBroadcastFacilitator struct {
	inner  *topic.HTTPSOverlayBroadcastFacilitator
	logger *slog.Logger
}

func (f *loggingBroadcastFacilitator) Send(url string, taggedBEEF *overlay.TaggedBEEF) (*overlay.Steak, error) {
	f.logger.Debug("Sending to overlay host", "url", url, "topics", taggedBEEF.Topics)

	steak, err := f.inner.Send(url, taggedBEEF)
	if err != nil {
		f.logger.Debug("Overlay send failed", "url", url, "error", err)
	} else {
		f.logger.Debug("Overlay send success", "url", url, "steak", steak)
	}

	return steak, err
}

// advertisementProtocol is the wallet protocol used for advertisement PushDrop keys.
func advertisementProtocol() sdkWallet.Protocol {
	return sdkWallet.Protocol{
		SecurityLevel: sdkWallet.SecurityLevelEveryAppAndCounterparty,
		Protocol:      ProtocolID,
	}
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
func (wp *Provider) CreateAdvertisement(ctx context.Context, network overlay.Network, p CreateAdParams) error {
	wallet := wp.GetWallet()
	if wallet == nil {
		return ErrWalletNotInitialized
	}

	lockingScript, err := wp.buildPushDropScript(ctx, p)
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
				Tags:              advertisementTags(uhrpURL, p),
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

	if err = wp.overlayBroadcast(result.Tx, network, wp.slapTrackers); err != nil {
		return err
	}

	wp.Logger.Debug("Advertisement broadcasted successfully",
		"uhrpUrl", p.URL,
		"expirySecs", p.ExpirySecs,
		"uploader", p.Uploader,
	)

	return nil
}

// RenewAdvertisement consumes an existing advertisement output and creates a new one with the updated script.
func (wp *Provider) RenewAdvertisement(
	ctx context.Context, network overlay.Network, matchedOutput *sdkWallet.Output, beef []byte, p CreateAdParams,
) error {
	wallet := wp.GetWallet()
	if wallet == nil {
		return ErrWalletNotInitialized
	}

	if matchedOutput == nil {
		return ErrNoMatchedOutput
	}

	lockingScript, err := wp.decodeAndBuildPushDropLockingScript(ctx, matchedOutput, p)
	if err != nil {
		return fmt.Errorf("failed to build advertisement script: %w", err)
	}

	aResult, err := wallet.CreateAction(ctx, sdkWallet.CreateActionArgs{
		InputBEEF:   beef,
		Description: "Renew UHRP advertisement for " + p.URL,
		Inputs: []sdkWallet.CreateActionInput{
			{
				Outpoint:              matchedOutput.Outpoint,
				InputDescription:      "Redeem previous UHRP advertisement",
				UnlockingScriptLength: pushDropUnlockingScriptLength,
			},
		},
		Outputs: []sdkWallet.CreateActionOutput{
			{
				LockingScript:     lockingScript.Bytes(),
				Satoshis:          1,
				OutputDescription: "UHRP advertisement token (renewed)",
				Basket:            BasketName,
				Tags:              advertisementTags(p.URL, p),
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

	unlockingScript, inputIndex, err := wp.buildPushDropUnlockingScript(ctx, matchedOutput, aResult)
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

	if err = wp.overlayBroadcast(sResult.Tx, network, wp.slapTrackers); err != nil {
		return err
	}

	wp.Logger.Debug("Advertisement renewed successfully",
		"uhrpUrl", p.URL,
		"expirySecs", p.ExpirySecs,
		"uploader", p.Uploader,
	)

	return nil
}

// advertisementTags returns the wallet tags that index an advertisement output; metadataFromTags reads them back.
func advertisementTags(uhrpURL string, p CreateAdParams) []string {
	return []string{
		tagUhrpURL + hex.EncodeToString([]byte(uhrpURL)),
		tagObjectIdentifier + hex.EncodeToString([]byte(p.ObjectID)),
		tagUploaderIdentityKey + p.Uploader,
		fmt.Sprintf("%s%d", tagExpiryTime, p.ExpirySecs),
		tagName + "file",
		tagContentType + p.ContentType,
		fmt.Sprintf("%s%d", tagSize, p.ContentLength),
	}
}

// buildPushDropScript builds a PushDrop-compatible locking script using the go-sdk template.
func (wp *Provider) buildPushDropScript(ctx context.Context, p CreateAdParams) (*script.Script, error) {
	expiry, err := varIntField(p.ExpirySecs)
	if err != nil {
		return nil, fmt.Errorf("expiry: %w", err)
	}

	contentLength, err := varIntField(p.ContentLength)
	if err != nil {
		return nil, fmt.Errorf("content length: %w", err)
	}

	wallet := wp.GetWallet()

	pubKey, err := wallet.GetPublicKey(ctx, sdkWallet.GetPublicKeyArgs{IdentityKey: true}, WalletName)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	fields := [][]byte{
		pubKey.PublicKey.Compressed(),
		p.Hash,
		[]byte(p.URL),
		expiry,
		contentLength,
	}

	return lockAdvertisement(ctx, wallet, fields)
}

func (wp *Provider) buildPushDropUnlockingScript(
	ctx context.Context, matchedOutput *sdkWallet.Output, result *sdkWallet.CreateActionResult,
) (*script.Script, uint32, error) {
	pd := &pushdrop.PushDrop{
		Wallet: wp.GetWallet(),
	}

	opts := pushdrop.UnlockOptions{
		SourceSatoshis: &matchedOutput.Satoshis,
		LockingScript:  script.NewFromBytes(matchedOutput.LockingScript),
	}

	unlocker := pd.Unlock(
		ctx,
		advertisementProtocol(),
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
		return nil, 0, ErrTransactionNotInBEEF
	}

	inputIndex, err := findInputIndex(tx, matchedOutput.Outpoint)
	if err != nil {
		return nil, 0, err
	}

	tx.Inputs[inputIndex].SetSourceTxOutput(&transaction.TransactionOutput{
		Satoshis:      matchedOutput.Satoshis,
		LockingScript: script.NewFromBytes(matchedOutput.LockingScript),
	})

	unlockingScript, err := unlocker.Sign(tx, int(inputIndex))
	if err != nil {
		return nil, 0, fmt.Errorf("error unlocking funding input: %w", err)
	}

	return unlockingScript, inputIndex, nil
}

// findInputIndex returns the index of the input in tx that spends outpoint.
func findInputIndex(tx *transaction.Transaction, outpoint transaction.Outpoint) (uint32, error) {
	want := outpoint.String()

	for i, input := range tx.Inputs {
		if fmt.Sprintf("%s.%d", input.SourceTXID.String(), input.SourceTxOutIndex) == want {
			return uint32(i), nil
		}
	}

	return 0, fmt.Errorf("%w: %s", ErrInputNotFound, want)
}

func (wp *Provider) decodeAndBuildPushDropLockingScript(
	ctx context.Context, matchedOutput *sdkWallet.Output, p CreateAdParams,
) (*script.Script, error) {
	prevLockingScript := pushdrop.Decode((*script.Script)(&matchedOutput.LockingScript))
	if prevLockingScript == nil || len(prevLockingScript.Fields) < pushDropFieldCount {
		return nil, ErrInvalidPushDropScript
	}

	expiry, err := varIntField(p.ExpirySecs)
	if err != nil {
		return nil, fmt.Errorf("expiry: %w", err)
	}

	// Keep the host key, hash, URL and content length; only the expiry changes on renewal.
	fields := make([][]byte, pushDropFieldCount)
	copy(fields, prevLockingScript.Fields[:pushDropFieldCount])
	fields[pushDropExpiryField] = expiry

	return lockAdvertisement(ctx, wp.GetWallet(), fields)
}

// lockAdvertisement builds the advertisement PushDrop locking script for fields, locked to "anyone".
func lockAdvertisement(ctx context.Context, wallet sdkWallet.Interface, fields [][]byte) (*script.Script, error) {
	pd := &pushdrop.PushDrop{
		Wallet: wallet,
	}

	return pd.Lock(
		ctx,
		fields,
		advertisementProtocol(),
		AnyonesKeyID,
		sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeAnyone},
		true,
		true,
		pushdrop.LockBefore,
	)
}

// varIntField encodes a non-negative value as a VarInt PushDrop field.
func varIntField(v int64) ([]byte, error) {
	if v < 0 {
		return nil, fmt.Errorf("%w: %d", ErrNegativeValue, v)
	}

	return util.VarInt(uint64(v)).Bytes(), nil
}

// overlayBroadcast submits a signed advertisement to the UHRP overlay topic.
//
// Admission is best effort: by the time it runs the file is stored and the wallet has
// already accepted the transaction, so a host that refuses the advertisement is logged
// rather than failing the request. Only a transaction that cannot be parsed or a
// broadcaster that cannot be built is returned as an error.
func (wp *Provider) overlayBroadcast(tx []byte, network overlay.Network, slapTrackers []string) error {
	resolver := lookup.NewLookupResolver(&lookup.LookupResolver{
		NetworkPreset: network,
		SLAPTrackers:  slapTrackers,
		Facilitator: &loggingLookupFacilitator{
			inner:  &lookup.HTTPSOverlayLookupFacilitator{Client: http.DefaultClient},
			logger: wp.Logger,
		},
	})

	broadcaster, err := topic.NewBroadcaster([]string{uhrpTopic}, &topic.BroadcasterConfig{
		NetworkPreset: network,
		Resolver:      resolver,
		Facilitator: &loggingBroadcastFacilitator{
			inner:  &topic.HTTPSOverlayBroadcastFacilitator{Client: http.DefaultClient},
			logger: wp.Logger,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create topic broadcaster: %w", err)
	}

	newBeef, newTxHash, err := transaction.NewBeefFromAtomicBytes(tx)
	if err != nil {
		return fmt.Errorf("error parsing signed transaction bytes into BEEF: %w", err)
	}

	newTx := newBeef.FindAtomicTransactionByHash(newTxHash)
	wp.Logger.Debug("Parsed signed transaction from BEEF", "version", newTx.Version, "inputs", len(newTx.Inputs), "outputs", len(newTx.Outputs))
	wp.Logger.Debug("Broadcasting to overlay network", "topic", uhrpTopic, "slapTrackers", slapTrackers)

	if success, failure := broadcaster.Broadcast(newTx); failure != nil {
		wp.Logger.Warn("Overlay broadcast failed — file stored but not yet advertised on UHRP network",
			"code", failure.Code,
			"description", failure.Description,
		)
	} else {
		wp.Logger.Debug("Overlay broadcast response", "txid", success.Txid, "message", success.Message)
	}

	return nil
}
