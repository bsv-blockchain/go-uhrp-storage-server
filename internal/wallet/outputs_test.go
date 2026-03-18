package wallet_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

func TestFindAdvertisementByUhrpURL(t *testing.T) {
	uhrpURL := "uhrp:test"
	uploaderKey := "02cbc1404c96562479633e721b0e01476d05ebecfc6797a7e9df533f81daed48ed"
	zeroHash, _ := chainhash.NewHash(make([]byte, 32))

	tests := []struct {
		name          string
		mockOutputs   []sdkWallet.Output
		expectedError string
	}{
		{
			name: "Found",
			mockOutputs: []sdkWallet.Output{
				{
					Outpoint: transaction.Outpoint{Txid: *zeroHash, Index: 0},
					Tags: []string{
						"uhrp_url_" + hex.EncodeToString([]byte(uhrpURL)),
						"uploader_identity_key_" + uploaderKey,
						"size_1024",
						"expiry_time_2000000000",
					},
				},
			},
		},
		{
			name:          "Not Found",
			mockOutputs:   []sdkWallet.Output{},
			expectedError: "uhrpUrl not found",
		},
		{
			name:          "Multiple Found",
			mockOutputs:   []sdkWallet.Output{{}, {}},
			expectedError: "multiple advertisements found",
		},
		{
			name: "Expired",
			mockOutputs: []sdkWallet.Output{
				{
					Tags: []string{
						"uhrp_url_" + hex.EncodeToString([]byte(uhrpURL)),
						"uploader_identity_key_" + uploaderKey,
						"expiry_time_100",
					},
				},
			},
			expectedError: "advertisement for uhrpUrl is expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &mocks.MockWallet{
				ListOutputsFunc: func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error) {
					return &sdkWallet.ListOutputsResult{Outputs: tt.mockOutputs}, nil
				},
			}

			out, meta, _, err := wallet.FindAdvertisementByUhrpURL(context.Background(), mw, uhrpURL, uploaderKey, 0, 0)

			if tt.expectedError != "" {
				if err == nil || !contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if out == nil {
				t.Fatal("expected output, got nil")
			}

			if meta.URL != uhrpURL {
				t.Errorf("expected URL %s, got %s", uhrpURL, meta.URL)
			}
		})
	}
}

func TestListAdvertisementsByUploader(t *testing.T) {
	uploaderKey := "02cbc1404c96562479633e721b0e01476d05ebecfc6797a7e9df533f81daed48ed"

	mw := &mocks.MockWallet{
		ListOutputsFunc: func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error) {
			return &sdkWallet.ListOutputsResult{
				Outputs: []sdkWallet.Output{
					{
						Tags: []string{
							"uhrp_url_" + hex.EncodeToString([]byte("uhrp:1")),
							"uploader_identity_key_" + uploaderKey,
							fmt.Sprintf("expiry_time_%d", time.Now().Add(10*time.Minute).Unix()),
						},
					},
					{
						Tags: []string{
							"uhrp_url_" + hex.EncodeToString([]byte("uhrp:2")),
							"uploader_identity_key_" + uploaderKey,
							fmt.Sprintf("expiry_time_%d", time.Now().Truncate(10*time.Minute).Unix()), // expired
						},
					},
				},
			}, nil
		},
	}

	metas, err := wallet.ListAdvertisementsByUploader(context.Background(), mw, uploaderKey, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(metas) != 1 {
		t.Errorf("expected 1 active advertisement, got %d", len(metas))
	}

	if metas[0].URL != "uhrp:1" {
		t.Errorf("expected uhrp:1, got %s", metas[0].URL)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
