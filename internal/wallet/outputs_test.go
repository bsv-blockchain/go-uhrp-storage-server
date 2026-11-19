package wallet_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

var errListFailed = errors.New("list failed")

func outputsMock(outputs ...sdkWallet.Output) *mocks.MockWallet {
	return &mocks.MockWallet{
		ListOutputsFunc: func(context.Context, sdkWallet.ListOutputsArgs, string) (*sdkWallet.ListOutputsResult, error) {
			return &sdkWallet.ListOutputsResult{Outputs: outputs}, nil
		},
	}
}

func urlTag(url string) string {
	return "uhrp_url_" + hex.EncodeToString([]byte(url))
}

func expiryTag(t time.Time) string {
	return fmt.Sprintf("expiry_time_%d", t.Unix())
}

func TestFindAdvertisementByUhrpURL(t *testing.T) {
	uhrpURL := "uhrp:test"
	uploader := uploaderIdentity(t)

	zeroHash, err := chainhash.NewHash(make([]byte, 32))
	require.NoError(t, err)

	tests := []struct {
		name        string
		outputs     []sdkWallet.Output
		expectedErr error
	}{
		{
			name: "found",
			outputs: []sdkWallet.Output{{
				Outpoint: transaction.Outpoint{Txid: *zeroHash, Index: 0},
				Tags: []string{
					urlTag(uhrpURL),
					"uploader_identity_key_" + uploader,
					"object_identifier_" + hex.EncodeToString([]byte("obj")),
					"content_type_text/plain",
					"name_file",
					"size_1024",
					"expiry_time_2000000000",
					"unknown_tag",
				},
			}},
		},
		{name: "not found", outputs: []sdkWallet.Output{}, expectedErr: wallet.ErrAdvertisementNotFound},
		{name: "multiple found", outputs: []sdkWallet.Output{{}, {}}, expectedErr: wallet.ErrMultipleAdvertisements},
		{
			name: "expired",
			outputs: []sdkWallet.Output{{
				Tags: []string{urlTag(uhrpURL), "uploader_identity_key_" + uploader, "expiry_time_100"},
			}},
			expectedErr: wallet.ErrAdvertisementExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, meta, _, err := newProvider(outputsMock(tt.outputs...)).FindAdvertisementByUhrpURL(t.Context(), uhrpURL, uploader, 0, 0)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, wallet.FileMetadata{
				UploaderIdentityKey: uploader,
				URL:                 uhrpURL,
				ObjectIdentifier:    "obj",
				Name:                "file",
				Size:                "1024",
				ContentType:         "text/plain",
				ExpiryTime:          2_000_000_000,
			}, *meta)
		})
	}
}

func TestFindAdvertisementByUhrpURLQuery(t *testing.T) {
	uploader := uploaderIdentity(t)

	var got sdkWallet.ListOutputsArgs

	mw := &mocks.MockWallet{
		ListOutputsFunc: func(_ context.Context, args sdkWallet.ListOutputsArgs, _ string) (*sdkWallet.ListOutputsResult, error) {
			got = args
			return &sdkWallet.ListOutputsResult{}, nil
		},
	}

	out, _, _, err := newProvider(mw).FindAdvertisementByUhrpURL(t.Context(), "uhrp:test", uploader, 0, 7)
	require.ErrorIs(t, err, wallet.ErrAdvertisementNotFound)
	assert.Nil(t, out)

	assert.Equal(t, wallet.BasketName, got.Basket)
	assert.Equal(t, sdkWallet.QueryModeAll, got.TagQueryMode)
	assert.Equal(t, []string{urlTag("uhrp:test"), "uploader_identity_key_" + uploader}, got.Tags)
	assert.Equal(t, uint32(200), *got.Limit, "a zero limit selects the default")
	assert.Equal(t, uint32(7), *got.Offset)
}

func TestFindAdvertisementByUhrpURLWalletErrors(t *testing.T) {
	out, _, _, err := wallet.NewProvider("", "", "mainnet", nil, slog.Default()).FindAdvertisementByUhrpURL(t.Context(), "u", "k", 0, 0)
	require.ErrorIs(t, err, wallet.ErrWalletNotInitialized)
	assert.Nil(t, out)

	mw := &mocks.MockWallet{
		ListOutputsFunc: func(context.Context, sdkWallet.ListOutputsArgs, string) (*sdkWallet.ListOutputsResult, error) {
			return nil, errListFailed
		},
	}

	out, _, _, err = newProvider(mw).FindAdvertisementByUhrpURL(t.Context(), "u", "k", 0, 0)
	require.ErrorIs(t, err, errListFailed)
	assert.Nil(t, out)
}

func TestGetFileSize(t *testing.T) {
	uploader := uploaderIdentity(t)

	size, err := newProvider(outputsMock(sdkWallet.Output{
		Tags: []string{urlTag("u"), "size_4096", "expiry_time_2000000000"},
	})).GetFileSize(t.Context(), "u", uploader)
	require.NoError(t, err)
	assert.Equal(t, int64(4096), size)

	_, err = newProvider(outputsMock(sdkWallet.Output{
		Tags: []string{urlTag("u"), "size_lots", "expiry_time_2000000000"},
	})).GetFileSize(t.Context(), "u", uploader)
	require.ErrorIs(t, err, wallet.ErrInvalidFileSize)
}

func TestFileMetadataSizeBytes(t *testing.T) {
	tests := []struct {
		size    string
		want    int64
		wantErr bool
	}{
		{size: "0", want: 0},
		{size: "1024", want: 1024},
		{size: "", wantErr: true},
		{size: "-1", wantErr: true},
		{size: "12abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			got, err := wallet.FileMetadata{Size: tt.size}.SizeBytes()
			if tt.wantErr {
				require.ErrorIs(t, err, wallet.ErrInvalidFileSize)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListAdvertisementsByUploader(t *testing.T) {
	uploader := uploaderIdentity(t)
	now := time.Now()

	mw := outputsMock(
		sdkWallet.Output{Tags: []string{urlTag("uhrp:1"), expiryTag(now.Add(10 * time.Minute))}},
		sdkWallet.Output{Tags: []string{urlTag("uhrp:2"), expiryTag(now.Add(-10 * time.Minute))}}, // expired
		sdkWallet.Output{Tags: []string{urlTag("uhrp:3"), expiryTag(now.Add(5 * time.Minute))}},
		sdkWallet.Output{Tags: []string{urlTag("uhrp:1"), expiryTag(now.Add(time.Hour))}},   // renewal
		sdkWallet.Output{Tags: []string{urlTag("uhrp:3"), expiryTag(now.Add(time.Minute))}}, // older copy
		sdkWallet.Output{Tags: []string{urlTag("uhrp:4"), "expiry_time_0"}},                 // no expiry
	)

	metas, err := newProvider(mw).ListAdvertisementsByUploader(t.Context(), uploader, 0, 0)
	require.NoError(t, err)

	urls := make([]string, 0, len(metas))
	for _, m := range metas {
		urls = append(urls, m.URL)
	}

	assert.Equal(t, []string{"uhrp:1", "uhrp:3", "uhrp:4"}, urls, "active URLs, in first-seen order")
	assert.Equal(t, now.Add(time.Hour).Unix(), metas[0].ExpiryTime, "the latest renewal wins")
	assert.Equal(t, now.Add(5*time.Minute).Unix(), metas[1].ExpiryTime, "an older copy does not replace a newer one")
}

func TestListAdvertisementsByUploaderErrors(t *testing.T) {
	_, err := wallet.NewProvider("", "", "mainnet", nil, slog.Default()).ListAdvertisementsByUploader(t.Context(), "k", 0, 0)
	require.ErrorIs(t, err, wallet.ErrWalletNotInitialized)

	mw := &mocks.MockWallet{
		ListOutputsFunc: func(context.Context, sdkWallet.ListOutputsArgs, string) (*sdkWallet.ListOutputsResult, error) {
			return nil, errListFailed
		},
	}

	_, err = newProvider(mw).ListAdvertisementsByUploader(t.Context(), "k", 0, 0)
	require.ErrorIs(t, err, errListFailed)
}
