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

// Tag prefixes written on every advertisement output (see advertisementTags).
const (
	tagUhrpURL             = "uhrp_url_"
	tagObjectIdentifier    = "object_identifier_"
	tagUploaderIdentityKey = "uploader_identity_key_"
	tagExpiryTime          = "expiry_time_"
	tagName                = "name_"
	tagContentType         = "content_type_"
	tagSize                = "size_"
)

// defaultListLimit is the page size used when a caller does not ask for one.
const defaultListLimit uint32 = 200

// FileMetadata is the advertisement metadata recovered from an output's tags.
type FileMetadata struct {
	UploaderIdentityKey string `json:"uploaderIdentityKey"`
	URL                 string `json:"url"`
	ObjectIdentifier    string `json:"objectIdentifier"`
	Name                string `json:"name"`
	Size                string `json:"size"`
	ContentType         string `json:"contentType"`
	ExpiryTime          int64  `json:"expiryTime"` // Unix seconds
}

// SizeBytes parses the advertised file size. It returns ErrInvalidFileSize when the
// size tag is missing or is not a non-negative integer.
func (m FileMetadata) SizeBytes() (int64, error) {
	size, err := strconv.ParseInt(m.Size, 10, 64)
	if err != nil || size < 0 {
		return 0, fmt.Errorf("%w: %q", ErrInvalidFileSize, m.Size)
	}

	return size, nil
}

// expiredAt reports whether the advertisement has an expiry time that is before now (Unix seconds).
func (m FileMetadata) expiredAt(now int64) bool {
	return m.ExpiryTime > 0 && m.ExpiryTime < now
}

// FindAdvertisementByUhrpURL finds a single UHRP advertisement output by its UHRP URL.
// It returns ErrAdvertisementNotFound, ErrMultipleAdvertisements or ErrAdvertisementExpired
// when there is not exactly one active advertisement for the URL and uploader.
func (wp *Provider) FindAdvertisementByUhrpURL(
	ctx context.Context, uhrpURL, uploaderIdentityKeyHex string, limit, offset uint32,
) (*sdkWallet.Output, *FileMetadata, []byte, error) {
	wallet := wp.GetWallet()
	if wallet == nil {
		return nil, nil, nil, ErrWalletNotInitialized
	}

	includeCustom := true
	includeTags := true

	if limit == 0 {
		limit = defaultListLimit
	}

	listResult, err := wallet.ListOutputs(ctx, sdkWallet.ListOutputsArgs{
		Basket:                    BasketName,
		Include:                   sdkWallet.OutputIncludeLockingScripts,
		IncludeCustomInstructions: &includeCustom,
		IncludeTags:               &includeTags,
		Limit:                     &limit,
		Offset:                    &offset,
		TagQueryMode:              sdkWallet.QueryModeAll,
		Tags: []string{
			tagUhrpURL + hex.EncodeToString([]byte(uhrpURL)),
			tagUploaderIdentityKey + uploaderIdentityKeyHex,
		},
	}, "")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to query wallet outputs: %w", err)
	}

	switch len(listResult.Outputs) {
	case 0:
		return nil, nil, nil, ErrAdvertisementNotFound
	case 1:
	default:
		return nil, nil, nil, ErrMultipleAdvertisements
	}

	output := listResult.Outputs[0]
	metadata := metadataFromTags(output.Tags)

	if metadata.expiredAt(time.Now().Unix()) {
		return nil, nil, nil, ErrAdvertisementExpired
	}

	return &output, &metadata, listResult.BEEF, nil
}

// GetFileSize retrieves the file size for a given UHRP URL.
func (wp *Provider) GetFileSize(ctx context.Context, uhrpURL, uploaderIdentityKeyHex string) (int64, error) {
	_, meta, _, err := wp.FindAdvertisementByUhrpURL(ctx, uhrpURL, uploaderIdentityKeyHex, defaultListLimit, 0)
	if err != nil {
		return 0, err
	}

	return meta.SizeBytes()
}

// ListAdvertisementsByUploader lists the active advertisements for a specific uploader.
// When a URL has been advertised more than once (renewals), only the latest expiry is kept.
func (wp *Provider) ListAdvertisementsByUploader(
	ctx context.Context, uploaderIdentityKeyHex string, limit, offset uint32,
) ([]FileMetadata, error) {
	wallet := wp.GetWallet()
	if wallet == nil {
		return nil, ErrWalletNotInitialized
	}

	includeCustom := true
	includeTags := true

	if limit == 0 {
		limit = defaultListLimit
	}

	result, err := wallet.ListOutputs(ctx, sdkWallet.ListOutputsArgs{
		Basket:                    BasketName,
		Tags:                      []string{tagUploaderIdentityKey + uploaderIdentityKeyHex},
		IncludeCustomInstructions: &includeCustom,
		IncludeTags:               &includeTags,
		Limit:                     &limit,
		Offset:                    &offset,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list outputs: %w", err)
	}

	return latestActiveByURL(result.Outputs, time.Now().Unix()), nil
}

// latestActiveByURL drops expired advertisements and keeps the latest expiry for each URL,
// in the order each URL first appears in outputs.
func latestActiveByURL(outputs []sdkWallet.Output, now int64) []FileMetadata {
	metadatas := make([]FileMetadata, 0, len(outputs))
	indexByURL := make(map[string]int, len(outputs))

	for _, output := range outputs {
		meta := metadataFromTags(output.Tags)
		if meta.expiredAt(now) {
			continue
		}

		i, seen := indexByURL[meta.URL]
		if !seen {
			indexByURL[meta.URL] = len(metadatas)
			metadatas = append(metadatas, meta)

			continue
		}

		if meta.ExpiryTime > metadatas[i].ExpiryTime {
			metadatas[i] = meta
		}
	}

	return metadatas
}

// metadataFromTags recovers advertisement metadata from an output's tags.
func metadataFromTags(tags []string) FileMetadata {
	var meta FileMetadata
	for _, tag := range tags {
		applyTag(&meta, tag)
	}

	return meta
}

// applyTag sets the metadata field a single tag describes; unknown tags are ignored.
func applyTag(m *FileMetadata, tag string) {
	if v, ok := strings.CutPrefix(tag, tagUploaderIdentityKey); ok {
		m.UploaderIdentityKey = v
		return
	}

	if v, ok := strings.CutPrefix(tag, tagUhrpURL); ok {
		m.URL = decodeHexTag(v)
		return
	}

	if v, ok := strings.CutPrefix(tag, tagObjectIdentifier); ok {
		m.ObjectIdentifier = decodeHexTag(v)
		return
	}

	if v, ok := strings.CutPrefix(tag, tagExpiryTime); ok {
		m.ExpiryTime, _ = strconv.ParseInt(v, 10, 64) // a malformed expiry reads as "no expiry" (0)
		return
	}

	if v, ok := strings.CutPrefix(tag, tagName); ok {
		m.Name = v
		return
	}

	if v, ok := strings.CutPrefix(tag, tagSize); ok {
		m.Size = v
		return
	}

	if v, ok := strings.CutPrefix(tag, tagContentType); ok {
		m.ContentType = v
	}
}

// decodeHexTag decodes a hex-encoded tag value, returning the raw value if it is not valid hex.
func decodeHexTag(v string) string {
	decoded, err := hex.DecodeString(v)
	if err != nil {
		return v
	}

	return string(decoded)
}
