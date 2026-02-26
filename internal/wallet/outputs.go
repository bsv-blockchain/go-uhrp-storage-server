package wallet

import (
	"context"
	"fmt"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

// FindAdvertisementByUhrpURL finds a single UHRP advertisement output by its UHRP URL.
func FindAdvertisementByUhrpURL(ctx context.Context, wallet sdkWallet.Interface, uhrpURL string) (*sdkWallet.Output, map[string]string, error) {
	includeCustom := true
	includeTags := true
	includeLocking := sdkWallet.OutputIncludeLockingScripts
	listResult, err := wallet.ListOutputs(ctx, sdkWallet.ListOutputsArgs{
		Basket:                    "uhrp advertisements",
		Include:                   includeLocking,
		IncludeCustomInstructions: &includeCustom,
		IncludeTags:               &includeTags,
	}, "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query wallet outputs: %w", err)
	}

	for _, out := range listResult.Outputs {
		meta := ParseCustomInstructions(out.CustomInstructions)
		if meta != nil && meta["uhrpURL"] == uhrpURL {
			return &out, meta, nil
		}
	}

	return nil, nil, fmt.Errorf("uhrpUrl not found in wallet outputs")
}

// GetFileSize retrieves the file size for a given UHRP URL.
func GetFileSize(ctx context.Context, wallet sdkWallet.Interface, uhrpURL string) (int64, error) {
	_, meta, err := FindAdvertisementByUhrpURL(ctx, wallet, uhrpURL)
	if err != nil {
		return 0, err
	}

	var fileSize int64
	fmt.Sscanf(meta["fileSize"], "%d", &fileSize)
	return fileSize, nil
}

// ListAdvertisementsByUploader lists all advertisements for a specific uploader.
func ListAdvertisementsByUploader(ctx context.Context, wallet sdkWallet.Interface, uploaderIdentityKeyHex string) ([]sdkWallet.Output, error) {
	includeCustom := true
	includeTags := true
	result, err := wallet.ListOutputs(ctx, sdkWallet.ListOutputsArgs{
		Basket:                    "uhrp advertisements",
		Tags:                      []string{fmt.Sprintf("uploader-%s", uploaderIdentityKeyHex)},
		IncludeCustomInstructions: &includeCustom,
		IncludeTags:               &includeTags,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list outputs: %w", err)
	}
	return result.Outputs, nil
}
