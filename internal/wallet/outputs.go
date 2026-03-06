package wallet

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

type FileMetadata struct {
	UploaderIdentityKey string `json:"uploaderIdentityKey"`
	URL                 string `json:"url"`
	ObjectIdentifier    string `json:"objectIdentifier"`
	Name                string `json:"name"`
	Size                string `json:"size"`
	ContentType         string `json:"contentType"`
	ExpiryTime          int64  `json:"expiryTime"` // minutes since the Unix epoch
}

// FindAdvertisementByUhrpURL finds a single UHRP advertisement output by its UHRP URL.
func FindAdvertisementByUhrpURL(ctx context.Context, wallet sdkWallet.Interface, uhrpURL, uploaderIdentityKeyHex string) (*sdkWallet.Output, *FileMetadata, []byte, error) {
	includeCustom := true
	includeTags := true
	includeLocking := sdkWallet.OutputIncludeLockingScripts

	listResult, err := wallet.ListOutputs(ctx, sdkWallet.ListOutputsArgs{
		Basket:                    "uhrp advertisements",
		Include:                   includeLocking,
		IncludeCustomInstructions: &includeCustom,
		IncludeTags:               &includeTags,
		Tags: []string{
			fmt.Sprintf("uhrp_url_%s", hex.EncodeToString([]byte(uhrpURL))),
		},
	}, "")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to query wallet outputs: %w", err)
	}

	if len(listResult.Outputs) == 0 {
		return nil, nil, nil, fmt.Errorf("uhrpUrl not found in wallet outputs")
	}

	var maxpiryOutput *sdkWallet.Output
	var maxpiryMetadata *FileMetadata
	var maxpiry int64 = 0

	for _, output := range listResult.Outputs {
		metadata := mapOutputToMetadata(output)

		// This check is more optimal than adding uploader_identity_key tag to the query
		if metadata.UploaderIdentityKey != uploaderIdentityKeyHex {
			continue
		}

		if metadata.ExpiryTime > maxpiry {
			maxpiry = metadata.ExpiryTime
			outCopy := output
			maxpiryOutput = &outCopy
			maxpiryMetadata = &metadata
		}
	}

	if maxpiryOutput == nil {
		return nil, nil, nil, fmt.Errorf("no valid advertisement found with an expiry time")
	}

	return maxpiryOutput, maxpiryMetadata, listResult.BEEF, nil
}

// GetFileSize retrieves the file size for a given UHRP URL.
func GetFileSize(ctx context.Context, wallet sdkWallet.Interface, uhrpURL, uploaderIdentityKeyHex string) (int64, error) {
	_, meta, _, err := FindAdvertisementByUhrpURL(ctx, wallet, uhrpURL, uploaderIdentityKeyHex)
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
	// TODO: add filter by spendable flag to reduce the number of outputs to process
	limit := uint32(1000)
	result, err := wallet.ListOutputs(ctx, sdkWallet.ListOutputsArgs{
		Basket:                    "uhrp advertisements",
		Tags:                      []string{fmt.Sprintf("uploader_identity_key_%s", uploaderIdentityKeyHex)},
		IncludeCustomInstructions: &includeCustom,
		IncludeTags:               &includeTags,
		Limit:                     &limit,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list outputs: %w", err)
	}

	metadatasMap := make(map[string]FileMetadata)

	now := time.Now().Unix()

	for _, output := range result.Outputs {
		meta := mapOutputToMetadata(output)

		if meta.ExpiryTime > 0 && meta.ExpiryTime < now {
			continue
		}

		if existing, ok := metadatasMap[meta.URL]; ok {
			if meta.ExpiryTime > existing.ExpiryTime {
				metadatasMap[meta.URL] = meta
			}
		} else {
			metadatasMap[meta.URL] = meta
		}
	}

	metadatas := make([]FileMetadata, 0, len(metadatasMap))
	for _, meta := range metadatasMap {
		metadatas = append(metadatas, meta)
	}

	return metadatas, nil
}

func mapOutputToMetadata(output sdkWallet.Output) FileMetadata {
	response := FileMetadata{}

	for _, tag := range output.Tags {
		if strings.HasPrefix(tag, "uploader_identity_key_") {
			response.UploaderIdentityKey = strings.TrimPrefix(tag, "uploader_identity_key_")
		}

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
