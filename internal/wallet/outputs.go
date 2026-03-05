package wallet

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

type FileMetadata struct {
	URL              string `json:"url"`
	ObjectIdentifier string `json:"objectIdentifier"`
	Name             string `json:"name"`
	Size             string `json:"size"`
	ContentType      string `json:"contentType"`
	ExpiryTime       int64  `json:"expiryTime"` // minutes since the Unix epoch
}

// FindAdvertisementByUhrpURL finds a single UHRP advertisement output by its UHRP URL.
func FindAdvertisementByUhrpURL(ctx context.Context, wallet sdkWallet.Interface, uhrpURL string, uploaderIdentityKeyHex string) (*sdkWallet.Output, *FileMetadata, []byte, error) {
	includeCustom := true
	includeTags := true
	includeLocking := sdkWallet.OutputIncludeLockingScripts
	limit := uint32(1)
	listResult, err := wallet.ListOutputs(ctx, sdkWallet.ListOutputsArgs{
		Basket:                    "uhrp advertisements",
		Include:                   includeLocking,
		IncludeCustomInstructions: &includeCustom,
		IncludeTags:               &includeTags,
		Tags: []string{
			fmt.Sprintf("uhrp_url_%s", hex.EncodeToString([]byte(uhrpURL))),
			// fmt.Sprintf("uploader_identity_key_%s", uploaderIdentityKeyHex),
		},
		Limit: &limit,
	}, "")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to query wallet outputs: %w", err)
	}

	if len(listResult.Outputs) == 0 {
		return nil, nil, nil, fmt.Errorf("uhrpUrl not found in wallet outputs")
	}

	output := listResult.Outputs[0]

	metadata := mapOutputToMetadata(output)

	return &output, &metadata, listResult.BEEF, nil
}

// GetFileSize retrieves the file size for a given UHRP URL.
func GetFileSize(ctx context.Context, wallet sdkWallet.Interface, uhrpURL string) (int64, error) {
	_, meta, _, err := FindAdvertisementByUhrpURL(ctx, wallet, uhrpURL, "")
	if err != nil {
		return 0, err
	}

	var fileSize int64
	fmt.Sscanf(meta.Size, "%d", &fileSize)
	return fileSize, nil
}

// ListAdvertisementsByUploader lists all advertisements for a specific uploader.
func ListAdvertisementsByUploader(ctx context.Context, wallet sdkWallet.Interface, uploaderIdentityKeyHex string) ([]FileMetadata, error) {
	includeCustom := true
	includeTags := true
	result, err := wallet.ListOutputs(ctx, sdkWallet.ListOutputsArgs{
		Basket:                    "uhrp advertisements",
		Tags:                      []string{fmt.Sprintf("uploader_identity_key_%s", uploaderIdentityKeyHex)},
		IncludeCustomInstructions: &includeCustom,
		IncludeTags:               &includeTags,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list outputs: %w", err)
	}

	metadatas := make([]FileMetadata, 0)

	for _, output := range result.Outputs {
		metadatas = append(metadatas, mapOutputToMetadata(output))
	}

	return metadatas, nil
}

func mapOutputToMetadata(output sdkWallet.Output) FileMetadata {
	response := FileMetadata{}

	for _, tag := range output.Tags {
		if strings.HasPrefix(tag, "uhrp_url_") {
			hexStr := strings.TrimPrefix(tag, "uhrp_url_")
			if decoded, err := hex.DecodeString(hexStr); err == nil {
				response.URL = string(decoded)
			} else {
				response.URL = hexStr
			}
		}
		if strings.HasPrefix(tag, "object_identifier_") {
			hexStr := strings.TrimPrefix(tag, "object_identifier_")
			if decoded, err := hex.DecodeString(hexStr); err == nil {
				response.ObjectIdentifier = string(decoded)
			} else {
				response.ObjectIdentifier = hexStr
			}
		}
		if strings.HasPrefix(tag, "expiry_time_") {
			response.ExpiryTime, _ = strconv.ParseInt(strings.TrimPrefix(tag, "expiry_time_"), 10, 64)
		}
		if strings.HasPrefix(tag, "name_") {
			response.Name = strings.TrimPrefix(tag, "name_")
		}
		if strings.HasPrefix(tag, "size_") {
			response.Size = strings.TrimPrefix(tag, "size_")
		}
		if strings.HasPrefix(tag, "content_type_") {
			response.ContentType = strings.TrimPrefix(tag, "content_type_")
		}
	}

	return response
}
